package access

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"tei/internal/core"
	"tei/internal/permission"
)

// FileSystem performs file access on behalf of one capability. Every access is
// checked against the granted permissions and recorded before it happens, so a
// refused access leaves no file behind. Root confines the paths the capability
// may name; a path outside it is refused even when the permission would allow it.
type FileSystem struct {
	Root       string
	Capability core.CanonicalID
	Guard      Guard
}

// ReadFile reads a file inside the root.
func (fs FileSystem) ReadFile(name string) ([]byte, error) {
	resolved, err := fs.resolve(name, permission.AccessRead)
	if err != nil {
		return nil, err
	}
	root, err := os.OpenRoot(fs.Root)
	if err != nil {
		return nil, fmt.Errorf("open root %s: %w", fs.Root, err)
	}
	defer root.Close()
	data, err := root.ReadFile(resolved)
	if err != nil {
		fs.recordRefused(permission.AccessRead, name, err)
		return nil, fmt.Errorf("read %s: %w", resolved, err)
	}
	return data, nil
}

// WriteFile writes a file inside the root.
func (fs FileSystem) WriteFile(name string, data []byte, mode os.FileMode) error {
	resolved, err := fs.resolve(name, permission.AccessWrite)
	if err != nil {
		return err
	}
	root, err := os.OpenRoot(fs.Root)
	if err != nil {
		return fmt.Errorf("open root %s: %w", fs.Root, err)
	}
	defer root.Close()
	if dir := filepath.Dir(resolved); dir != "." {
		// The directory may already exist, as a directory or as a link that
		// resolves to one; in that case there is nothing to prepare and
		// MkdirAll would report that it already exists. A link that leaves the
		// root is not an existing directory inside it, so it falls through to
		// MkdirAll and is refused there.
		if err := root.MkdirAll(dir, 0o755); err != nil {
			if _, statErr := os.Stat(filepath.Join(fs.Root, dir)); statErr != nil {
				fs.recordRefused(permission.AccessWrite, name, err)
				return fmt.Errorf("prepare %s: %w", resolved, err)
			}
		}
	}
	file, err := root.OpenFile(resolved, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, mode)
	if err != nil {
		fs.recordRefused(permission.AccessWrite, name, err)
		return fmt.Errorf("write %s: %w", resolved, err)
	}
	if _, err := file.Write(data); err != nil {
		file.Close()
		fs.recordRefused(permission.AccessWrite, name, err)
		return fmt.Errorf("write %s: %w", resolved, err)
	}
	if err := file.Close(); err != nil {
		fs.recordRefused(permission.AccessWrite, name, err)
		return fmt.Errorf("write %s: %w", resolved, err)
	}
	return nil
}

// existsInside reports whether dir is already a directory inside the root. A link
// that points inside the root is one, and preparing it is not an error.
func existsInside(root *os.Root, dir string) bool {
	info, err := root.Stat(dir)
	return err == nil && info.IsDir()
}

// resolve checks the requirement and records the decision. The returned name is
// relative to the root, which is what os.Root expects.
//
// A name that cannot be a path inside the root at all is refused before the
// permission is consulted, and recorded as such. A name that this check accepts
// but the operating system refuses (a link that leaves the root) is recorded when
// the refusal happens, so the record describes what happened rather than what
// this check guessed.
func (fs FileSystem) resolve(name string, access permission.Access) (string, error) {
	relative, err := fs.inside(name)
	if err != nil {
		fs.recordOutsideRoot(access, name, err)
		return "", err
	}
	requirement := permission.Spec{Kind: permission.KindFilesystem, Access: access, Scope: filepath.Join(fs.Root, relative)}
	decision, err := fs.Guard.Check(fs.Capability, []permission.Spec{requirement})
	if err != nil {
		return "", err
	}
	if !decision.Allowed {
		return "", fmt.Errorf("capability %s may not %s %s: the granted permissions do not cover it", fs.Capability, access, requirement.Scope)
	}
	return relative, nil
}

// recordOutsideRoot records a name that cannot be inside the root.
func (fs FileSystem) recordOutsideRoot(access permission.Access, name string, reason error) {
	fs.record(access, name, "outside the root: "+reason.Error())
}

// recordRefused records an access the operating system refused, such as a link
// that leaves the root. The permission was granted, so the record says the path
// was refused, not that the permission was missing.
func (fs FileSystem) recordRefused(access permission.Access, name string, reason error) {
	fs.record(access, name, "refused by the root: "+reason.Error())
}

// record writes one refusal. A journal failure is not returned: the access was
// refused anyway, and reporting a journal error instead would hide the refusal.
func (fs FileSystem) record(access permission.Access, name, why string) {
	if fs.Guard.Journal == nil {
		return
	}
	_ = fs.Guard.Journal.Append(Decision{
		Capability: string(fs.Capability),
		Allowed:    false,
		Missing:    []string{string(permission.KindFilesystem) + ":" + string(access) + " " + name + " (" + why + ")"},
	})
}

// inside returns the path relative to the root, refusing the names that are
// outside it by construction.
//
// This is a first, cheap pass: `..` that climbs above the root, and an absolute
// name, are refused without touching the operating system. The binding
// confinement is done by os.Root at the moment of the access, because a check
// that resolves the path and then names it again has a gap for a link created
// between the two.
func (fs FileSystem) inside(name string) (string, error) {
	if fs.Root == "" {
		return "", fmt.Errorf("no root is configured, so no path is allowed")
	}
	if filepath.IsAbs(name) {
		return "", fmt.Errorf("%s is an absolute path, not a name inside the root", name)
	}
	cleaned := filepath.Clean(name)
	if cleaned == ".." || strings.HasPrefix(cleaned, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("%s is outside the root %s", name, fs.Root)
	}
	if cleaned == "." {
		return "", fmt.Errorf("%s names no file inside the root", name)
	}
	return cleaned, nil
}
