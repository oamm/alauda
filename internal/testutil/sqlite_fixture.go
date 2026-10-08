package testutil

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// DatabasePath owns a file-backed SQLite fixture directory; close databases before test cleanup.
func DatabasePath(t *testing.T) string {
	t.Helper()
	dir, err := os.MkdirTemp("", "alauda-sqlite-")
	if err != nil {
		t.Fatal(err)
	}
	absolute, err := filepath.Abs(dir)
	if err != nil {
		t.Fatal(err)
	}
	tempRoot, err := filepath.Abs(os.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if !strings.EqualFold(filepath.Dir(absolute), tempRoot) || !strings.HasPrefix(filepath.Base(absolute), "alauda-sqlite-") {
		t.Fatal("unsafe fixture cleanup path")
	}
	t.Cleanup(func() {
		deadline := time.Now().Add(2 * time.Second)
		retries := 0
		for {
			err := os.RemoveAll(absolute)
			if err == nil {
				if retries > 0 {
					t.Logf("SQLite fixture cleanup needed %d retries after Close", retries)
				}
				return
			}
			if time.Now().After(deadline) {
				t.Errorf("persistent fixture cleanup failure: %v", err)
				return
			}
			// Windows may retain delete-pending file handles briefly after SQLite closes.
			retries++
			time.Sleep(20 * time.Millisecond)
		}
	})
	return filepath.Join(absolute, "registry.db")
}
