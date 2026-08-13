package webstatic

import (
	"embed"
	"io/fs"
	"os"
)

// webDist contains the built frontend bundle when copied to internal/webstatic/dist.
//go:embed dist/*
var webDist embed.FS

// FileSystem returns embedded assets when available, otherwise falls back to local web/dist.
func FileSystem() fs.FS {
	sub, err := fs.Sub(webDist, "dist")
	if err == nil {
		if entries, readErr := fs.ReadDir(sub, "."); readErr == nil && len(entries) > 0 {
			return sub
		}
	}
	return os.DirFS("web/dist")
}
