package api

import (
	"database/sql"
	"errors"

	"connectrpc.com/connect"
	"github.com/company/service-registry/internal/storage"
)

func codeForStorageError(err error) connect.Code {
	if errors.Is(err, sql.ErrNoRows) {
		return connect.CodeNotFound
	}
	switch storage.ClassifyError(err) {
	case storage.ErrorConflict:
		return connect.CodeAlreadyExists
	case storage.ErrorInvalid:
		return connect.CodeInvalidArgument
	case storage.ErrorPrecondition:
		return connect.CodeFailedPrecondition
	case storage.ErrorUnavailable:
		return connect.CodeUnavailable
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
