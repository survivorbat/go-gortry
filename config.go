package gortry

import (
	"errors"
	"fmt"
	"math"
	"time"
)

// ErrInvalidConfig is returned from the config validation
var ErrInvalidConfig = errors.New("invalid config")

// baseInterval is an arbitrary number that serves as the default base time.
const baseInterval = 100 * time.Millisecond

type Config struct {
	// MaxAttempts describes the maxmimum of atempts that should be made before giving up
	MaxAttempts int

	// Interval is the base interval between attempts, multiplied by IntervalMultiplier
	Interval time.Duration

	// MaxInterval is the maximum amount of time to wait between query attempts. A minimum of
	// 100 milliseconds is required.
	MaxInterval time.Duration

	// Timeout is the total amount of time a retry may take, this overrides MaxAttempts
	Timeout time.Duration

	// IntervalMultiplier is the multiplier applied to each attempt's interval, until it hits
	// MaxInterval. Must be at least 1 as a negative backoff does not make sense.
	IntervalMultiplier float64
}

const (
	defaultMaxAttempts        = 10
	defaultInterval           = time.Second
	defaultMaxInterval        = time.Second * 30
	defaultIntervalMultiplier = 1.5
	defaultTimeout            = time.Second * 60
)

func DefaultConfig() *Config {
	return &Config{
		MaxAttempts:        defaultMaxAttempts,
		Interval:           defaultInterval,
		MaxInterval:        defaultMaxInterval,
		IntervalMultiplier: defaultIntervalMultiplier,
		Timeout:            defaultTimeout,
	}
}

func (c *Config) validate() error {
	var errs []error

	if c.Interval < baseInterval {
		errs = append(errs, fmt.Errorf("interval must be higher than %v: %w", baseInterval, ErrInvalidConfig))
	}

	if c.MaxInterval < baseInterval {
		errs = append(errs, fmt.Errorf("max interval must be higher than %v: %w", baseInterval, ErrInvalidConfig))
	}

	if c.Timeout < time.Second {
		errs = append(errs, fmt.Errorf("timeout can not be lower than a second: %w", ErrInvalidConfig))
	}

	if c.IntervalMultiplier < 1 {
		errs = append(errs, fmt.Errorf("interval multiplier must not be lower than 1: %w", ErrInvalidConfig))
	}

	if c.MaxAttempts < 1 {
		errs = append(errs, fmt.Errorf("max attempts must at least be 1: %w", ErrInvalidConfig))
	}

	return errors.Join(errs...)
}

// intervalFor calculates the interval time for the attempt, capping at the max interval
func (c *Config) intervalFor(attempt int) time.Duration {
	multiplier := math.Pow(c.IntervalMultiplier, float64(attempt-1))

	attemptInterval := c.Interval * time.Duration(multiplier)

	if attemptInterval > c.MaxInterval {
		return c.MaxInterval
	}

	return attemptInterval
}
