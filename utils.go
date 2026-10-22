package gortry

import (
	"errors"
	"net"

	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
)

// isTransaction determines whether a query was executed within a transaction,
// which'd make it ineligible for retrying.
func isTransaction(db *gorm.DB) bool {
	_, ok := db.Statement.ConnPool.(gorm.TxCommitter)
	return ok
}

// retryablePostgresCodes are standard postgres error codes, for more info:

// https://www.postgresql.org/docs/current/errcodes-appendix.html
var retryablePostgresCodes = map[string]struct{}{
	"08000": {}, // Connection exception
	"08001": {}, // Unable to establish connection
	"08003": {}, // Connection does not exist
	"08006": {}, // Connection failure
	"57P01": {}, // Admin shutdown
	"57P02": {}, // Crash shutdown
	"57P03": {}, // Temporarily unavailable
}

// isRetryable determines whether the error can be retried
func isRetryable(err error) bool {
	if err == nil {
		return false
	}

	// Generic network errors
	_, isNet := errors.AsType[net.Error](err)
	if isNet {
		return true
	}

	// Postgres driver's own safe retry method
	if pgconn.SafeToRetry(err) {
		return true
	}

	// PG error codes
	pgErr, isPgErr := errors.AsType[*pgconn.PgError](err)
	if isPgErr {
		_, ok := retryablePostgresCodes[pgErr.Code]
		return ok
	}

	return false
}
