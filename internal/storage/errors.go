package storage

import (
	"errors"

	"github.com/jackc/pgx/v5/pgconn"
	sqlite3 "github.com/mattn/go-sqlite3"
)

type ErrorKind int

const (
	ErrorOther ErrorKind = iota
	ErrorConflict
	ErrorInvalid
	ErrorPrecondition
	ErrorUnavailable
)

// ClassifyError normalizes common constraint and contention errors across providers.
func ClassifyError(err error) ErrorKind {
	var sqlite sqlite3.Error
	if errors.As(err, &sqlite) {
		if sqlite.Code == sqlite3.ErrBusy || sqlite.Code == sqlite3.ErrLocked {
			return ErrorUnavailable
		}
		if sqlite.Code == sqlite3.ErrConstraint {
			switch sqlite.ExtendedCode {
			case sqlite3.ErrConstraintForeignKey:
				return ErrorPrecondition
			case sqlite3.ErrConstraintCheck, sqlite3.ErrConstraintNotNull:
				return ErrorInvalid
			default:
				return ErrorConflict
			}
		}
	}
	var pg *pgconn.PgError
	if errors.As(err, &pg) {
		switch pg.Code {
		case "23505":
			return ErrorConflict
		case "23503":
			return ErrorPrecondition
		case "23514", "23502", "22P02":
			return ErrorInvalid
		case "40001", "40P01", "55P03":
			return ErrorUnavailable
		}
	}
	return ErrorOther
}
