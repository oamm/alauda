package api

import (
	"testing"

	"github.com/company/service-registry/internal/testutil"
)

func fixtureDatabasePath(t *testing.T) string {
	t.Helper()
	return testutil.DatabasePath(t)
}
