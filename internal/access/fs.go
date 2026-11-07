package access

import (
	"fmt"
	"os"
	"path/filepath"

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
	return os.ReadFile(resolved)
}

// WriteFile writes a file inside the root.
func (fs FileSystem) WriteFile(name string, data []byte, mode os.FileMode) error {
	resolved, err := fs.resolve(name, permission.AccessWrite)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(resolved), 0o755); err != nil {
		return fmt.Errorf("prepare %s: %w", resolved, err)
	}
	return os.WriteFile(resolved, data, mode)
}

// resolve checks the requirement, records the decision, and returns the path.
//
// A path outside the root is refused before the permission is consulted, and that
// refusal is recorded too. A refusal nobody can read back is indistinguishable
// from an access that never happened: an escape attempt that leaves no trace
// cannot be investigated afterwards.
func (fs FileSystem) resolve(name string, access permission.Access) (string, error) {
	resolved, err := fs.inside(name)
	if err != nil {
		fs.recordOutsideRoot(access, name, err)
		return "", err
	}
	requirement := permission.Spec{Kind: permission.KindFilesystem, Access: access, Scope: resolved}
	decision, err := fs.Guard.Check(fs.Capability, []permission.Spec{requirement})
	if err != nil {
		return "", err
	}
	if !decision.Allowed {
		return "", fmt.Errorf("capability %s may not %s %s: the granted permissions do not cover it", fs.Capability, access, resolved)
	}
	return resolved, nil
}

// recordOutsideRoot records a refusal that the permission check never saw. The
// journal is what the guard would have written, so the two refusals read the
// same way in the log. A journal failure is not returned: the path was refused
// anyway, and turning that refusal into a journal error would report a failure to
// record a decision instead of the decision itself.
func (fs FileSystem) recordOutsideRoot(access permission.Access, name string, reason error) {
	if fs.Guard.Journal == nil {
		return
	}
	_ = fs.Guard.Journal.Append(Decision{
		Capability: string(fs.Capability),
		Allowed:    false,
		Missing:    []string{string(permission.KindFilesystem) + ":" + string(access) + " " + name + " (outside the root: " + reason.Error() + ")"},
	})
}

// inside reports the absolute path, refusing anything outside the root.
func (fs FileSystem) inside(name string) (string, error) {
	if fs.Root == "" {
		return "", fmt.Errorf("no root is configured, so no path is allowed")
	}
	root, err := filepath.Abs(fs.Root)
	if err != nil {
		return "", fmt.Errorf("resolve root: %w", err)
	}
	resolved, err := filepath.Abs(filepath.Join(root, name))
	if err != nil {
		return "", fmt.Errorf("resolve %s: %w", name, err)
	}
	relative, err := filepath.Rel(root, resolved)
	if err != nil {
		return "", fmt.Errorf("compare %s with the root: %w", name, err)
	}
	if relative == ".." || len(relative) > 3 && relative[:3] == ".."+string(filepath.Separator) {
		return "", fmt.Errorf("%s is outside the root %s", name, root)
	}
	return resolved, nil
}
