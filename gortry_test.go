package gortry

import (
	"context"
	"slices"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type dummyObject struct {
	Name string
}

func TestPlugin_RetriesOnErrors(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		errs        []error
		maxAttempts int
		timeout     time.Duration

		expectedCalls int
		expectedErr   error
	}{
		"no errors": {
			errs:        []error{},
			maxAttempts: 1,
			timeout:     time.Minute,

			expectedCalls: 1,
		},
		"non-retryable error": {
			errs:        []error{assert.AnError},
			maxAttempts: 1,
			timeout:     time.Minute,

			expectedCalls: 1,
			expectedErr:   assert.AnError,
		},
		"one retryable error": {
			errs:        []error{RetryError{}},
			maxAttempts: 2,
			timeout:     time.Minute,

			expectedCalls: 2,
		},
		"three retryable errors": {
			errs:        []error{RetryError{}, RetryError{}, RetryError{}},
			maxAttempts: 4,
			timeout:     time.Minute,

			expectedCalls: 4,
		},
		"retryable errors that and up in a generic error": {
			errs:        []error{RetryError{}, RetryError{}, assert.AnError},
			maxAttempts: 3,
			timeout:     time.Minute,

			expectedCalls: 3,
			expectedErr:   assert.AnError,
		},
		"max attempts reached": {
			errs:        []error{RetryError{}, RetryError{}},
			maxAttempts: 2,
			timeout:     time.Minute,

			expectedCalls: 2,
			expectedErr:   RetryError{},
		},
		"timeout reached": {
			errs:        slices.Repeat([]error{RetryError{}}, 100), // Enough for the time
			maxAttempts: 100,
			timeout:     time.Second,

			expectedCalls: 10,
			expectedErr:   RetryError{},
		},
	}

	for name, testData := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			// Arrange
			db := newDB(t)

			cfg := DefaultConfig()
			cfg.MaxAttempts = testData.maxAttempts
			cfg.Timeout = testData.timeout
			cfg.MaxInterval = time.Millisecond * 100 // As fast as possible

			plugin, err := New(cfg)
			require.NoError(t, err)

			var calls int
			data := []*dummyObject{{Name: "foo"}, {Name: "bar"}}

			// Override the default query behaviour and return errors
			_ = db.Callback().Query().Replace("gorm:query", newVerifyCallback(&calls, testData.errs, data))

			err = db.Use(plugin)
			require.NoError(t, err)

			var actual []*dummyObject

			// Act
			tx := db.Find(&actual)

			// Assert
			if testData.expectedErr == nil {
				require.NoError(t, tx.Error)
				assert.Equal(t, data, actual)
			} else {
				require.ErrorIs(t, tx.Error, testData.expectedErr)
				assert.Empty(t, actual)
			}

			assert.Equalf(t, testData.expectedCalls, calls, "expected %d calls", calls)
		})
	}
}

func TestPlugin_ExitsOnCancelledContext(t *testing.T) {
	t.Parallel()
	// Arrange
	db := newDB(t)

	plugin, err := New(DefaultConfig())
	require.NoError(t, err)

	var calls int
	data := []*dummyObject{{Name: "foo"}, {Name: "bar"}}
	errs := []error{RetryError{}}

	// Override the default query behaviour and return errors
	_ = db.Callback().Query().Replace("gorm:query", newVerifyCallback(&calls, errs, data))

	err = db.Use(plugin)
	require.NoError(t, err)

	var actual []*dummyObject

	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	// Act
	tx := db.WithContext(ctx).Find(&actual)

	// Assert
	require.ErrorIs(t, tx.Error, RetryError{})
}

func TestPlugin_DoesNotRetryInTransaction(t *testing.T) {
	t.Parallel()
	// Arrange
	db := newDB(t)

	plugin, err := New(DefaultConfig())
	require.NoError(t, err)

	var calls int
	data := []*dummyObject{{Name: "foo"}, {Name: "bar"}}
	errs := []error{RetryError{}}

	// Override the default query behaviour and return errors
	_ = db.Callback().Query().Replace("gorm:query", newVerifyCallback(&calls, errs, data))

	err = db.Use(plugin)
	require.NoError(t, err)

	var actual []*dummyObject

	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	// Act
	err = db.Transaction(func(tx *gorm.DB) error {
		// Returns a retryuble error
		return tx.WithContext(ctx).Find(&actual).Error
	})

	// Assert
	require.ErrorIs(t, err, RetryError{})
	assert.Equal(t, 1, calls)
}

// newDB makes it easy to instantiate a database
func newDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"))
	require.NoError(t, err)

	return db
}

// newVerifyCallback updates the counter for every call and returns errors in the slice.
// Once we run out of errors we set the result in the destination.
func newVerifyCallback(counter *int, errs []error, result []*dummyObject) func(d *gorm.DB) {
	return func(d *gorm.DB) {
		*counter++

		if *counter <= len(errs) {
			_ = d.AddError(errs[*counter-1])
			return
		}

		dest := d.Statement.Dest.(*[]*dummyObject)
		*dest = result
	}
}
