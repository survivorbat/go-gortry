package gortry

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfig_Validate_ReturnsNilOnValid(t *testing.T) {
	t.Parallel()
	// Arrange
	cfg := &Config{
		MaxAttempts:        1,
		Interval:           100 * time.Millisecond,
		MaxInterval:        100 * time.Millisecond,
		IntervalMultiplier: 1,
		Timeout:            5 * time.Second,
	}

	// Act
	err := cfg.validate()

	// Assert
	require.NoError(t, err)
}

func TestConfig_Validate_ReturnsErrorOnInvalidValues(t *testing.T) {
	t.Parallel()
	// Arrange
	cfg := new(Config)

	// Act
	err := cfg.validate()

	// Assert
	require.ErrorIs(t, err, ErrInvalidConfig)

	expected := []string{
		"max attempts must at least be 1",
		"max interval must be higher than 100ms",
		"timeout must not be lower than 1s",
		"interval must be higher than 100ms",
		"interval multiplier must not be lower than 1",
	}

	for _, line := range expected {
		assert.ErrorContains(t, err, line)
	}
}

func TestConfig_IntervalFor_ReturnsExpectedValues(t *testing.T) {
	t.Parallel()
	// Arrange
	cfg := &Config{
		Interval:           time.Millisecond * 500,
		MaxInterval:        time.Second * 30,
		IntervalMultiplier: 2,
	}

	attempts := 10

	actual := make([]time.Duration, attempts)

	// Act
	for attempt := range attempts {
		actual[attempt] = cfg.intervalFor(attempt)
	}

	// Assert
	expected := []time.Duration{
		0,                      // Attempt 0
		time.Millisecond * 500, // Attempt 1
		time.Second * 1,        // Attempt 2
		time.Second * 2,        // Attempt 3
		time.Second * 4,        // Attempt 4
		time.Second * 8,        // Attempt 5
		time.Second * 16,       // Attempt 6
		time.Second * 30,       // Attempt 7
		time.Second * 30,       // Attempt 8
		time.Second * 30,       // Attempt 9
	}

	assert.Equal(t, expected, actual)
}
