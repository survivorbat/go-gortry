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
	err = db.Callback().Query().Replace("gorm:row", p.wrap(originalRow, callbacks.RowQuery))
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
		timeout := time.After(p.cfg.Timeout)

		for attempt := 1; attempt <= p.cfg.MaxAttempts; attempt++ {
			callback(tx)

			if !shouldRetry(tx) {
				return
			}

			delay := time.After(p.cfg.intervalFor(attempt))

			select {
			case <-tx.Statement.Context.Done():
				// Give up, context is cancelled
				return

			case <-timeout:
				// Give up, timed out
				return

			case <-delay:
				// Next attempt
			}

			tx.Error = nil
		}
	}
}

// shouldRetry determines if the given query should be retried
func shouldRetry(tx *gorm.DB) bool {
	return tx.Error != nil && !isTransaction(tx) && isRetryable(tx.Error)
}
