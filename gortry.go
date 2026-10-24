package gortry

import (
	"fmt"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/callbacks"
)

// name is the name of the plugin as registered in gorm
const name = "gortry"

// Compile-time interface checks
var _ gorm.Plugin = new(Plugin)

// Plugin implements gorm.Plugin and can be inserted in the db.Use method.
type Plugin struct {
	cfg *Config
}

// New instantiates the plugin with the given config options
func New(cfg *Config) (*Plugin, error) {
	err := cfg.validate()
	if err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	return &Plugin{cfg: cfg}, nil
}

// Name implements Plugin.Name
func (p *Plugin) Name() string {
	return name
}

// Initialize implements Plugin.Initialize. It overrides the specified queries
func (p *Plugin) Initialize(db *gorm.DB) error {
	originalQuery := db.Callback().Query().Get("gorm:query")

	err := db.Callback().Query().Replace("gorm:query", p.wrap(originalQuery, callbacks.Query))
	if err != nil {
		return fmt.Errorf("failed to register query callback: %w", err)
	}

	originalRow := db.Callback().Query().Get("gorm:row")
	err = db.Callback().Row().Replace("gorm:row", p.wrap(originalRow, callbacks.RowQuery))
	if err != nil {
		return fmt.Errorf("failed to register row callback: %w", err)
	}

	return nil
}

// wrap wraps the original callback in a for-loop that
func (p *Plugin) wrap(original func(*gorm.DB), fallback func(*gorm.DB)) func(*gorm.DB) {
	callback := original
	if callback == nil {
		callback = fallback
	}

	return func(tx *gorm.DB) {
		timeout := make(<-chan time.Time, 1) // Dummy timeout

		// If a timeout is set, use a timer, if not, continue to use the dummy version
		if p.cfg.Timeout != 0 {
			timeout = time.After(p.cfg.Timeout)
		}

		for attempt := 1; attempt <= p.cfg.MaxAttempts; attempt++ {
			callback(tx)

			if !shouldRetry(tx) {
				return
			}

			wait := time.After(p.cfg.intervalFor(attempt))

			select {
			case <-tx.Statement.Context.Done():
				return

			case <-timeout:
				return

			case <-wait:
				// Next attempt
			}

			// Don't reset the error when max attempts reached
			if p.cfg.MaxAttempts != attempt {
				tx.Error = nil
			}
		}
	}
}

// shouldRetry determines if the given query should be retried
func shouldRetry(tx *gorm.DB) bool {
	return tx.Error != nil && !isTransaction(tx) && isRetryable(tx.Error)
}
