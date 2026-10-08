package integration_test

import (
	"testing"

	"github.com/company/service-registry/internal/testutil"
)

func integrationDatabasePath(t *testing.T) string {
	t.Helper()
	return testutil.DatabasePath(t)
}
