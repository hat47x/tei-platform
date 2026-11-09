// Package tenantisolation is a Reference foundation for SaaS delivery
// profiles. It confines an application's reads and writes to one tenant root
// and records cross-tenant attempts instead of treating them as empty data.
package tenantisolation

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
)

const (
	CodeCrossTenant       = "CROSS_TENANT_DENIED"
	CodeTenantRequired    = "TENANT_REQUIRED"
	CodeOutsideTenantRoot = "OUTSIDE_TENANT_ROOT"
)

var tenantPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]{0,63}$`)

// Denial is a tenant-boundary refusal. It is not a business "not found" result
// and must not be collapsed into an empty success by callers.
type Denial struct {
	Code            string `json:"code"`
	TenantID        string `json:"tenantId"`
	RequestedTenant string `json:"requestedTenant,omitempty"`
	Path            string `json:"path,omitempty"`
	Detail          string `json:"detail,omitempty"`
}

func (d *Denial) Error() string {
	if d.Path != "" {
		return fmt.Sprintf("%s: tenant=%s path=%s", d.Code, d.TenantID, d.Path)
	}
	return fmt.Sprintf("%s: tenant=%s requested=%s", d.Code, d.TenantID, d.RequestedTenant)
}

// Scope is one tenant's pinned root. Close it when the request or session ends.
type Scope struct {
	tenantID string
	rootPath string
	root     *os.Root
	mu       sync.Mutex
	denials  []Denial
}

func Open(tenantID, rootPath string) (*Scope, error) {
	if !tenantPattern.MatchString(tenantID) {
		return nil, fmt.Errorf("tenant id %q is invalid", tenantID)
	}
	if strings.TrimSpace(rootPath) == "" {
		return nil, fmt.Errorf("tenant root is required")
	}
	absolute, err := filepath.Abs(rootPath)
	if err != nil {
		return nil, err
	}
	root, err := os.OpenRoot(absolute)
	if err != nil {
		return nil, fmt.Errorf("open tenant root: %w", err)
	}
	info, err := root.Stat(".")
	if err != nil {
		root.Close()
		return nil, err
	}
	if !info.IsDir() {
		root.Close()
		return nil, fmt.Errorf("tenant root is not a directory: %s", absolute)
	}
	return &Scope{tenantID: tenantID, rootPath: absolute, root: root}, nil
}

func (s *Scope) Close() error { return s.root.Close() }

func (s *Scope) TenantID() string { return s.tenantID }

func (s *Scope) Denials() []Denial {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]Denial(nil), s.denials...)
}

func (s *Scope) record(denial Denial) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.denials = append(s.denials, denial)
}

// RequireTenant fails closed when the request does not name the current tenant.
func (s *Scope) RequireTenant(requested string) error {
	requested = strings.TrimSpace(requested)
	if requested == "" {
		denial := Denial{Code: CodeTenantRequired, TenantID: s.tenantID, Detail: "request did not name a tenant"}
		s.record(denial)
		return &denial
	}
	if requested != s.tenantID {
		denial := Denial{
			Code: CodeCrossTenant, TenantID: s.tenantID, RequestedTenant: requested,
			Detail: "request names another tenant",
		}
		s.record(denial)
		return &denial
	}
	return nil
}

func safeRelative(value string) (string, error) {
	value = strings.TrimSpace(strings.ReplaceAll(value, "\\", "/"))
	if value == "" || strings.HasPrefix(value, "/") || filepath.IsAbs(value) {
		return "", fmt.Errorf("path %q must be relative", value)
	}
	for _, part := range strings.Split(value, "/") {
		if part == ".." {
			return "", fmt.Errorf("path %q must not contain ..", value)
		}
	}
	clean := path.Clean(value)
	if clean == "." || clean == ".." || strings.HasPrefix(clean, "../") {
		return "", fmt.Errorf("path %q escapes the tenant root", value)
	}
	return clean, nil
}

// checkPath rejects a symbolic link at every existing component. os.Root is the
// actual containment boundary; this check makes the refusal explicit.
func (s *Scope) checkPath(relative string, mayCreate bool) error {
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
			return fmt.Errorf("tenant path %s is a symbolic link", part)
		}
		if i < len(parts)-1 && !info.IsDir() {
			return fmt.Errorf("tenant parent %s is not a directory", part)
		}
		if i == len(parts)-1 {
			if info.IsDir() {
				return fmt.Errorf("tenant path %s is a directory", part)
			}
			if !info.Mode().IsRegular() {
				return fmt.Errorf("tenant path %s is not a regular file", part)
			}
		}
	}
	return nil
}

func (s *Scope) makeDirs(relative string) error {
	parts := strings.Split(relative, string(filepath.Separator))
	for i := 0; i < len(parts)-1; i++ {
		dir := filepath.Join(parts[:i+1]...)
		info, err := s.root.Lstat(dir)
		if err == nil {
			if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
				return fmt.Errorf("tenant parent %s is not a real directory", dir)
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

func (s *Scope) denyPath(relative, detail string) error {
	denial := Denial{Code: CodeOutsideTenantRoot, TenantID: s.tenantID, Path: relative, Detail: detail}
	s.record(denial)
	return &denial
}

func (s *Scope) Read(relative string) ([]byte, error) {
	rel, err := safeRelative(relative)
	if err != nil {
		return nil, s.denyPath(relative, err.Error())
	}
	rel = filepath.FromSlash(rel)
	if err := s.checkPath(rel, false); err != nil {
		if os.IsNotExist(err) {
			return nil, err
		}
		return nil, s.denyPath(relative, err.Error())
	}
	content, err := s.root.ReadFile(rel)
	if err != nil {
		return nil, err
	}
	return content, nil
}

func (s *Scope) Write(relative string, content []byte) error {
	rel, err := safeRelative(relative)
	if err != nil {
		return s.denyPath(relative, err.Error())
	}
	rel = filepath.FromSlash(rel)
	if err := s.checkPath(rel, true); err != nil {
		return s.denyPath(relative, err.Error())
	}
	if err := s.makeDirs(rel); err != nil {
		return s.denyPath(relative, err.Error())
	}
	file, err := s.root.OpenFile(rel, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return s.denyPath(relative, err.Error())
	}
	if _, err := file.Write(content); err != nil {
		file.Close()
		return err
	}
	if err := file.Sync(); err != nil {
		file.Close()
		return err
	}
	return file.Close()
}
