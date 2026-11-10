package dispatch

import (
	"os"
	"path/filepath"
)

func ensureDir(path string) error {
	dir := filepath.Dir(path)
	return os.MkdirAll(dir, 0o755)
}
