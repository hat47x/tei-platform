//go:build !windows

package main

import (
	"os"
	"path/filepath"
)

// finalPath returns the real location of an open file (symbolic links resolved).
func finalPath(f *os.File) (string, error) {
	return filepath.EvalSymlinks(f.Name())
}
