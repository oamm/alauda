package api

import (
	"strings"

	"connectrpc.com/connect"
)

func codeForStorageError(err error) connect.Code {
	message := strings.ToLower(err.Error())
	if strings.Contains(message, "unique constraint failed") {
		return connect.CodeAlreadyExists
	}
	if strings.Contains(message, "foreign key constraint failed") {
		return connect.CodeFailedPrecondition
	}
	return connect.CodeInternal
}
