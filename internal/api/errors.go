package api

import (
	"database/sql"
	"errors"

	"connectrpc.com/connect"
	"github.com/mattn/go-sqlite3"
)

func codeForStorageError(err error) connect.Code {
	if errors.Is(err, sql.ErrNoRows) {
		return connect.CodeNotFound
	}
	var sqlite sqlite3.Error
	if errors.As(err, &sqlite) {
		if sqlite.Code == sqlite3.ErrConstraint {
			if sqlite.ExtendedCode == sqlite3.ErrConstraintForeignKey {
				return connect.CodeFailedPrecondition
			}
			if sqlite.ExtendedCode == sqlite3.ErrConstraintCheck || sqlite.ExtendedCode == sqlite3.ErrConstraintNotNull {
				return connect.CodeInvalidArgument
			}
			return connect.CodeAlreadyExists
		}
		if sqlite.Code == sqlite3.ErrBusy || sqlite.Code == sqlite3.ErrLocked {
			return connect.CodeUnavailable
		}
	}
	return connect.CodeInternal
}

func safeConnectError(code connect.Code, err error) *connect.Error {
	if code == connect.CodeInternal {
		code = codeForStorageError(err)
	}
	message := "Operation failed."
	switch code {
	case connect.CodeNotFound:
		message = "Resource does not exist."
	case connect.CodeAlreadyExists:
		message = "Resource conflicts with an existing resource."
	case connect.CodeInvalidArgument:
		message = "Invalid resource fields."
	case connect.CodeUnavailable:
		message = "Operation is temporarily unavailable."
	case connect.CodeFailedPrecondition:
		message = "Resource cannot be used in its current state."
	}
	return connect.NewError(code, errors.New(message))
}
