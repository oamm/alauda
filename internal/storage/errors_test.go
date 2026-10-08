package storage

import (
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
)

func TestClassifyPostgresErrors(t *testing.T) {
	for _, tc := range []struct {
		state string
		want  ErrorKind
	}{
		{"23505", ErrorConflict}, {"23503", ErrorPrecondition}, {"23514", ErrorInvalid},
		{"40001", ErrorUnavailable}, {"40P01", ErrorUnavailable}, {"XX000", ErrorOther},
	} {
		if got := ClassifyError(&pgconn.PgError{Code: tc.state}); got != tc.want {
			t.Errorf("SQLSTATE %s classified as %v, want %v", tc.state, got, tc.want)
		}
	}
}
