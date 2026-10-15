// Package studiohost is the small, host-neutral read-only entry point for
// TEI Studio PoC A. It delegates meaning to internal/operations.
package studiohost

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"tei/internal/operations"
	"tei/internal/sourceindex"
)

type WorkspaceRef struct {
	ID    string `json:"id"`
	Label string `json:"label"`
}

type Failure struct {
	Code    string `json:"code"`
	Stage   string `json:"stage,omitempty"`
	Message string `json:"message"`
}

// Each response deliberately separates transport/operation completion from
// validation diagnostics. ok=true does NOT mean that a workspace is valid.
type ListResponse struct {
	OK    bool           `json:"ok"`
	Value []WorkspaceRef `json:"value"`
}
type ValidationResponse struct {
	OK      bool                            `json:"ok"`
	Value   *operations.WorkspaceValidation `json:"value,omitempty"`
	Failure *Failure                        `json:"failure,omitempty"`
}
type CatalogResponse struct {
	OK      bool                     `json:"ok"`
	Value   *operations.ModelCatalog `json:"value,omitempty"`
	Failure *Failure                 `json:"failure,omitempty"`
}
type InspectionResponse struct {
	OK      bool                          `json:"ok"`
	Value   *operations.SubjectInspection `json:"value,omitempty"`
	Failure *Failure                      `json:"failure,omitempty"`
}

// ReadOnly has no per-browser or selected-workspace state. The host supplies
// and fixes the workspace allowlist on startup; browser calls supply IDs only.
// CatalogService and ValidationService expose the generated services' concrete,
// typed operations. They do not accept renderer-controlled filesystem paths.
type CatalogService interface {
	Catalog(namespace string) (operations.ModelCatalog, error)
}
type ValidationService interface {
	Validate() (operations.WorkspaceValidation, error)
}
type RootServices struct {
	Catalog    CatalogService
	Validation ValidationService
}

// ServiceFactory is called only after the host has resolved an allowlisted ID.
// It is never registered as a Wails service or exposed to the renderer.
type ServiceFactory func(trustedRoot string) (RootServices, error)

type ReadOnly struct {
	roots     map[string]string
	rootInfos map[string]os.FileInfo
	refs      []WorkspaceRef
	services  ServiceFactory
}

func NewReadOnly(paths []string) (*ReadOnly, error) {
	return NewReadOnlyWithFactory(paths, nil)
}

// NewReadOnlyWithFactory preserves the normal Operation path when factory is
// nil. A non-nil factory receives only roots selected from the startup allowlist.
func NewReadOnlyWithFactory(paths []string, factory ServiceFactory) (*ReadOnly, error) {
	if len(paths) == 0 {
		return nil, errors.New("at least one -workspace is required")
	}
	host := &ReadOnly{
		roots:     make(map[string]string, len(paths)),
		rootInfos: make(map[string]os.FileInfo, len(paths)),
		services:  factory,
	}
	seen := map[string]bool{}
	for i, path := range paths {
		if path == "" {
			return nil, errors.New("empty workspace path")
		}
		absolute, err := filepath.Abs(path)
		if err != nil {
			return nil, fmt.Errorf("resolve workspace %d: %w", i+1, err)
		}
		resolved, err := filepath.EvalSymlinks(absolute)
		if err != nil {
			return nil, fmt.Errorf("resolve workspace %d: %w", i+1, err)
		}
		stat, err := os.Stat(resolved)
		if err != nil || !stat.IsDir() {
			return nil, fmt.Errorf("workspace %d must be an existing directory", i+1)
		}
		// Windows FileInfo resolves file IDs lazily from its pathname when
		// os.SameFile is first called. Resolve that identity now, before the
		// pathname could later be replaced.
		if !os.SameFile(stat, stat) {
			return nil, fmt.Errorf("workspace %d identity could not be established", i+1)
		}
		if seen[resolved] {
			return nil, fmt.Errorf("workspace %d duplicates an earlier root", i+1)
		}
		seen[resolved] = true
		id := fmt.Sprintf("workspace-%d", i+1)
		host.roots[id] = resolved
		host.rootInfos[id] = stat
		host.refs = append(host.refs, WorkspaceRef{ID: id, Label: filepath.Base(resolved)})
	}
	return host, nil
}

func (h *ReadOnly) ListWorkspaces() ListResponse {
	items := make([]WorkspaceRef, len(h.refs))
	copy(items, h.refs)
	return ListResponse{OK: true, Value: items}
}

func (h *ReadOnly) workspace(id string) (string, *Failure) {
	root, ok := h.roots[id]
	if !ok {
		return "", &Failure{Code: "WORKSPACE_NOT_ALLOWED", Message: "workspace ID is not in this host's allowlist"}
	}
	current, err := os.Stat(root)
	if err != nil || !os.SameFile(h.rootInfos[id], current) {
		return "", &Failure{Code: "WORKSPACE_ROOT_CHANGED", Message: "workspace root identity changed after host startup"}
	}
	return root, nil
}

func (h *ReadOnly) ValidateWorkspace(id string) ValidationResponse {
	root, denied := h.workspace(id)
	if denied != nil {
		return ValidationResponse{Failure: denied}
	}
	var value operations.WorkspaceValidation
	var err error
	if h.services == nil {
		value, err = operations.ValidateWorkspace(root)
	} else {
		var services RootServices
		services, err = h.services(root)
		if err == nil && services.Validation == nil {
			err = errors.New("validation service is missing")
		}
		if err == nil {
			value, err = services.Validation.Validate()
		}
	}
	if err != nil {
		return ValidationResponse{Failure: operationFailure(err)}
	}
	return ValidationResponse{OK: true, Value: &value}
}

func (h *ReadOnly) CatalogWorkspace(id, namespace string) CatalogResponse {
	root, denied := h.workspace(id)
	if denied != nil {
		return CatalogResponse{Failure: denied}
	}
	var value operations.ModelCatalog
	var err error
	if h.services == nil {
		value, err = operations.CatalogWorkspace(root, namespace)
	} else {
		var services RootServices
		services, err = h.services(root)
		if err == nil && services.Catalog == nil {
			err = errors.New("catalog service is missing")
		}
		if err == nil {
			value, err = services.Catalog.Catalog(namespace)
		}
	}
	if err != nil {
		return CatalogResponse{Failure: operationFailure(err)}
	}
	return CatalogResponse{OK: true, Value: &value}
}

func (h *ReadOnly) InspectSubject(id, kind, subjectID string) InspectionResponse {
	root, denied := h.workspace(id)
	if denied != nil {
		return InspectionResponse{Failure: denied}
	}
	value, err := operations.InspectSubject(root, sourceindex.Subject{Kind: kind, ID: subjectID})
	if err != nil {
		return InspectionResponse{Failure: operationFailure(err)}
	}
	return InspectionResponse{OK: true, Value: &value}
}

func operationFailure(err error) *Failure {
	var validation *operations.ValidationFailure
	if errors.As(err, &validation) {
		switch validation.Stage {
		case operations.StageLoad:
			return &Failure{Code: "WORKSPACE_LOAD_FAILED", Stage: string(validation.Stage), Message: "workspace loading failed"}
		case operations.StageSourceIndex:
			return &Failure{Code: "SOURCE_INDEX_FAILED", Stage: string(validation.Stage), Message: "source-index construction failed"}
		}
		return &Failure{Code: "OPERATION_FAILED", Message: "workspace operation failed"}
	}
	var inspection *operations.InspectionError
	if errors.As(err, &inspection) {
		return &Failure{Code: inspection.Code, Message: inspection.Message}
	}
	// Unclassified errors may contain absolute paths or host-private details.
	return &Failure{Code: "OPERATION_FAILED", Message: "workspace operation failed"}
}
