// Package offlinestorage is a Reference local key-value store for the
// individual delivery profile. It confines keys to one root and does not use a
// network service.
package offlinestorage

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
)

type Store struct {
	rootPath string
	root     *os.Root
}

func Open(rootPath string) (*Store, error) {
	if strings.TrimSpace(rootPath) == "" {
		return nil, fmt.Errorf("storage root is required")
	}
	absolute, err := filepath.Abs(rootPath)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(absolute, 0o755); err != nil {
		return nil, err
	}
	root, err := os.OpenRoot(absolute)
	if err != nil {
		return nil, err
	}
	info, err := root.Stat(".")
	if err != nil {
		root.Close()
		return nil, err
	}
	if !info.IsDir() {
		root.Close()
		return nil, fmt.Errorf("storage root is not a directory: %s", absolute)
	}
	return &Store{rootPath: absolute, root: root}, nil
}

func (s *Store) Close() error { return s.root.Close() }

func keyPath(key string) (string, error) {
	key = strings.TrimSpace(strings.ReplaceAll(key, "\\", "/"))
	if key == "" || strings.HasPrefix(key, "/") || filepath.IsAbs(key) {
		return "", fmt.Errorf("key %q must be relative", key)
	}
	for _, part := range strings.Split(key, "/") {
		if part == ".." {
			return "", fmt.Errorf("key %q must not contain ..", key)
		}
	}
	clean := path.Clean(key)
	if clean == "." || clean == ".." || strings.HasPrefix(clean, "../") {
		return "", fmt.Errorf("key %q escapes the storage root", key)
	}
	return filepath.FromSlash(clean), nil
}

func (s *Store) checkPath(relative string, mayCreate bool) error {
	parts := strings.Split(relative, string(filepath.Separator))
	for i := range parts {
		part := filepath.Join(parts[:i+1]...)
		info, err := s.root.Lstat(part)
		if os.IsNotExist(err) && mayCreate {
			continue
		}
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("storage path %s is a symbolic link", part)
		}
		if i < len(parts)-1 && !info.IsDir() {
			return fmt.Errorf("storage parent %s is not a directory", part)
		}
		if i == len(parts)-1 && !info.Mode().IsRegular() {
			return fmt.Errorf("storage path %s is not a regular file", part)
		}
	}
	return nil
}

func (s *Store) makeDirs(relative string) error {
	parts := strings.Split(relative, string(filepath.Separator))
	for i := 0; i < len(parts)-1; i++ {
		dir := filepath.Join(parts[:i+1]...)
		info, err := s.root.Lstat(dir)
		if err == nil {
			if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
				return fmt.Errorf("storage parent %s is not a real directory", dir)
			}
			continue
		}
		if !os.IsNotExist(err) {
			return err
		}
		if err := s.root.Mkdir(dir, 0o755); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) Put(key string, value []byte) error {
	relative, err := keyPath(key)
	if err != nil {
		return err
	}
	if err := s.checkPath(relative, true); err != nil {
		return err
	}
	if err := s.makeDirs(relative); err != nil {
		return err
	}
	file, err := s.root.OpenFile(relative, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return err
	}
	if _, err := file.Write(value); err != nil {
		file.Close()
		return err
	}
	if err := file.Sync(); err != nil {
		file.Close()
		return err
	}
	return file.Close()
}

func (s *Store) Get(key string) ([]byte, error) {
	relative, err := keyPath(key)
	if err != nil {
		return nil, err
	}
	if err := s.checkPath(relative, false); err != nil {
		return nil, err
	}
	return s.root.ReadFile(relative)
}

func (s *Store) Delete(key string) error {
	relative, err := keyPath(key)
	if err != nil {
		return err
	}
	if err := s.checkPath(relative, false); err != nil {
		return err
	}
	return s.root.Remove(relative)
}

func (s *Store) List() ([]string, error) {
	var keys []string
	err := filepath.WalkDir(s.rootPath, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return nil
		}
		relative, err := filepath.Rel(s.rootPath, path)
		if err != nil {
			return err
		}
		keys = append(keys, filepath.ToSlash(relative))
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Strings(keys)
	return keys, nil
}
