// Package studiohost is the small, host-neutral read-only entry point for
// TEI Studio PoC A. It delegates meaning to internal/operations.
package studiohost

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"tei/internal/command"
	"tei/internal/operations"
	"tei/internal/sourceindex"
	"tei/internal/validation"
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
	OK    bool                            `json:"ok"`
	Value *operations.WorkspaceValidation `json:"value,omitempty"`
	// Proposals are candidate corrections, one per diagnostic. They carry the
	// world fingerprint the diagnostic was read from and are never accepted
	// automatically: accepting one is a separate decision.
	Proposals []validation.Proposal `json:"proposals,omitempty"`
	Failure   *Failure              `json:"failure,omitempty"`
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
// The services expose the generated operations through concrete, typed
// methods. They do not accept renderer-controlled filesystem paths.
type CatalogService interface {
	Catalog(namespace string) (operations.ModelCatalog, error)
}
type ValidationService interface {
	Validate() (operations.WorkspaceValidation, error)
}

// InspectionService is a host-selected, typed read-only service. Its presence
// does not prove Plugin resolution, invocation provenance, or current authority.
type InspectionService interface {
	Inspect(subject sourceindex.Subject) (operations.SubjectInspection, error)
}
type RootServices struct {
	Catalog    CatalogService
	Validation ValidationService
	Inspection InspectionService
}

// ServiceFactory is called only after the host has resolved an allowlisted ID.
// It is never registered as a Wails service or exposed to the renderer.
type ServiceFactory func(trustedRoot string) (RootServices, error)

type ReadOnly struct {
	roots        map[string]string
	rootInfos    map[string]os.FileInfo
	refs         []WorkspaceRef
	services     ServiceFactory
	xeadObserver XeadObservationObserver

	// CommandLifecycle is composed by the host. When set, the
	// ExecuteCommandWithLifecycle path records dispatch/observe/unknown stages
	// in this store; the other ExecuteCommand methods keep their in-memory
	// behaviour.
	CommandLifecycle command.StageLifecycle
}

func NewReadOnly(paths []string) (*ReadOnly, error) {
	return NewReadOnlyWithFactoryAndObserver(paths, nil, nil)
}

func NewReadOnlyWithXeadObserver(paths []string, observer XeadObservationObserver) (*ReadOnly, error) {
	return NewReadOnlyWithFactoryAndObserver(paths, nil, observer)
}

// NewReadOnlyWithFactory preserves the normal Operation path when factory is
// nil. A non-nil factory receives only roots selected from the startup allowlist.
func NewReadOnlyWithFactory(paths []string, factory ServiceFactory) (*ReadOnly, error) {
	return NewReadOnlyWithFactoryAndObserver(paths, factory, nil)
}

// NewReadOnlyWithFactoryAndObserver retains the same allowlist boundary for
// normal operations, generated services, and the Studio-only source observer.
func NewReadOnlyWithFactoryAndObserver(paths []string, factory ServiceFactory, observer XeadObservationObserver) (*ReadOnly, error) {
	if len(paths) == 0 {
		return nil, errors.New("at least one -workspace is required")
	}
	host := &ReadOnly{
		roots:        make(map[string]string, len(paths)),
		rootInfos:    make(map[string]os.FileInfo, len(paths)),
		services:     factory,
		xeadObserver: observer,
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
	// Pair the diagnostics with version-bound proposals so the author surface
	// shows the same unanswered corrections the CLI reports.
	var proposals []validation.Proposal
	if value.WorldFingerprint != "" {
		diagnostics := make([]validation.Diagnostic, 0, len(value.Diagnostics))
		for _, projected := range value.Diagnostics {
			diagnostics = append(diagnostics, projected.Diagnostic)
		}
		if value.Direct != nil {
			diagnostics = append(diagnostics, validation.Diagnostic{
				Code: value.Direct.Code, Severity: value.Direct.Severity, Message: value.Direct.Message,
			})
		}
		proposals = validation.Propose(diagnostics, value.WorldFingerprint)
	}
	return ValidationResponse{OK: true, Value: &value, Proposals: proposals}
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
	subject := sourceindex.Subject{Kind: kind, ID: subjectID}
	var value operations.SubjectInspection
	var err error
	if h.services == nil {
		// Preserve the existing PoC path when no Host factory is configured.
		// This path is not evidence of a selected Plugin Provider.
		value, err = operations.InspectSubject(root, subject)
	} else {
		var services RootServices
		services, err = h.services(root)
		if err == nil && services.Inspection == nil {
			// Do not silently fall back to the built-in Inspector when a
			// Host-selected provider is absent or its binding is incomplete.
			err = errors.New("inspection service is missing")
		}
		if err == nil {
			value, err = services.Inspection.Inspect(subject)
		}
		if err == nil {
			err = operations.ValidateInspectionProjection(subject, value)
		}
	}
	if err != nil {
		if h.services != nil {
			// Provider-supplied typed errors may embed private roots, tokens,
			// or a forged code. Treat both fields as untrusted at this boundary.
			return InspectionResponse{Failure: selectedInspectionFailure(err)}
		}
		return InspectionResponse{Failure: operationFailure(err)}
	}
	return InspectionResponse{OK: true, Value: &value}
}

// selectedInspectionFailure preserves only the closed set of public Inspector
// error codes. A Host-selected Provider must not set arbitrary client-visible
// codes or include its private diagnostics in a typed InspectionError message.
// Direct built-in Inspector behavior is intentionally kept separate.
func selectedInspectionFailure(err error) *Failure {
	var inspection *operations.InspectionError
	if errors.As(err, &inspection) {
		switch inspection.Code {
		case operations.InspectionInvalidSubject,
			operations.InspectionUnsupportedKind,
			operations.InspectionNotFound,
			operations.InspectionAmbiguous:
			return &Failure{Code: inspection.Code, Message: "inspector could not resolve the requested subject"}
		default:
			return &Failure{Code: "OPERATION_FAILED", Message: "inspection service failed"}
		}
	}
	return operationFailure(err)
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
