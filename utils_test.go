package gortry

import (
	"net"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestIsRetryable(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		input    error
		expected bool
	}{
		"no error": {
			input:    nil,
			expected: false,
		},
		"random error": {
			input:    assert.AnError,
			expected: false,
		},
		"net error": {
			input:    &net.DNSError{},
			expected: true,
		},
		"safe to retry error": {
			input:    RetryError{},
			expected: true,
		},
		"pg error 08000": {
			input:    &pgconn.PgError{Code: "08000"},
			expected: true,
		},
		"pg error 08001": {
			input:    &pgconn.PgError{Code: "08001"},
			expected: true,
		},
		"pg error 08003": {
			input:    &pgconn.PgError{Code: "08003"},
			expected: true,
		},
		"pg error 08006": {
			input:    &pgconn.PgError{Code: "08006"},
			expected: true,
		},
		"pg error 57P01": {
			input:    &pgconn.PgError{Code: "57P01"},
			expected: true,
		},
		"pg error 57P02": {
			input:    &pgconn.PgError{Code: "57P02"},
			expected: true,
		},
		"pg error 57P03": {
			input:    &pgconn.PgError{Code: "57P03"},
			expected: true,
		},
	}

	for name, testData := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			// Act
			actual := isRetryable(testData.input)

			// Assert
			assert.Equal(t, testData.expected, actual)
		})
	}
}

func TestIsTransaction_ReturnsFalseIfNot(t *testing.T) {
	t.Parallel()
	// Arrange
	db, err := gorm.Open(sqlite.Open(":memory:"))
	require.NoError(t, err)

	// Act
	actual := isTransaction(db)

	// Assert
	assert.False(t, actual)
}

func TestIsTransaction_ReturnsTrueIfSo(t *testing.T) {
	t.Parallel()
	// Arrange
	db, err := gorm.Open(sqlite.Open(":memory:"))
	require.NoError(t, err)

	var actual bool

	// Act
	_ = db.Transaction(func(tx *gorm.DB) error {
		actual = isTransaction(tx)
		return nil
	})

	// Assert
	assert.True(t, actual)
}

type RetryError struct{}

func (r RetryError) SafeToRetry() bool {
	return true
}

func (r RetryError) Error() string {
	return "retryable"
}

func (r RetryError) Is(err error) bool {
	_, ok := err.(RetryError)
	return ok
}
