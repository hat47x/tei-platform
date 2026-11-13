// Package studiohost is the small, host-neutral read-only entry point for
// TEI Studio PoC A. It delegates meaning to internal/operations.
package studiohost

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"tei/internal/command"
	"tei/internal/failure"
	"tei/internal/operations"
	"tei/internal/sourceindex"
	"tei/internal/validation"
)

type WorkspaceRef struct {
	ID    string `json:"id"`
	Label string `json:"label"`
}

// Failure is the response shape a Studio host returns instead of a value. The
// Kind, Retryable and EffectUnknown fields carry the unified failure contract
// (internal/failure) so a client can tell a business rejection from a
// technical failure without reading the code list.
//
// The classification is filled by MarshalJSON when the producer left it empty,
// so a site that constructs a Failure directly - and there are many, one per
// entrance - cannot emit an unclassified failure by forgetting to ask. A value
// the producer set explicitly wins, because a boundary that knows better than
// the code table (commandFailure reads the error's own contract) must be able
// to say so.
type Failure struct {
	Code    string `json:"code"`
	Stage   string `json:"stage,omitempty"`
	Message string `json:"message"`
	// Kind is business_rejection, technical_failure or partial_success.
	Kind string `json:"kind,omitempty"`
	// Retryable is retryable, not_retryable or lookup_first.
	Retryable string `json:"retryable,omitempty"`
	// EffectUnknown reports that a partial effect may have been applied, so a
	// caller must not resend with a new operation identity.
	EffectUnknown bool `json:"effectUnknown,omitempty"`
}

// studioFailureClassification is the decision one Studio code maps to. It is
// separate from failure.Kind and failure.RetryPolicy only so this file does not
// have to name the package in every entry; the wire values are the same.
type studioFailureClassification struct {
	kind      string
	retryable string
	effect    bool
}

// studioFailureCodes is the closed table this host's codes are classified from.
//
// The split is the same one internal/failure makes: a code that refuses the
// request itself carries no effect and must not be resent unchanged, while a
// code that reports the host or a provider failing leaves the effect state
// undecidable from here. An unlisted code is technical, because reporting an
// unrecognised failure as a permanent business rejection would make a caller
// give up on something that may succeed later.
var studioFailureCodes = map[string]studioFailureClassification{
	// The request itself is refused. Nothing ran, so there is no effect to look
	// up and resending it unchanged answers the same way.
	"WORKSPACE_NOT_ALLOWED":     {"business_rejection", "not_retryable", false},
	"WORKSPACE_ROOT_CHANGED":    {"business_rejection", "not_retryable", false},
	"VIEW_UNSUPPORTED":          {"business_rejection", "not_retryable", false},
	"INVALID_LIFECYCLE_ATTEMPT": {"business_rejection", "not_retryable", false},
	"INVALID_PERIOD_STATE":      {"business_rejection", "not_retryable", false},
	"COMMAND_LIFECYCLE_REFUSED": {"business_rejection", "not_retryable", false},
	"INVALID_SUBJECT":           {"business_rejection", "not_retryable", false},
	"UNSUPPORTED_SUBJECT_KIND":  {"business_rejection", "not_retryable", false},
	"SUBJECT_NOT_FOUND":         {"business_rejection", "not_retryable", false},
	"SUBJECT_AMBIGUOUS":         {"business_rejection", "not_retryable", false},
	"XEAD_UNSUPPORTED_FORMAT":   {"business_rejection", "not_retryable", false},
	"XEAD_SOURCE_TOO_LARGE":     {"business_rejection", "not_retryable", false},
	// The three-view request is refused as asked: the host has no Provider for
	// it, or the named source is not one the selected Provider reads.
	"THREE_VIEW_UNAVAILABLE":        {"business_rejection", "not_retryable", false},
	"THREE_VIEW_UNSUPPORTED_SOURCE": {"business_rejection", "not_retryable", false},
	"THREE_VIEW_SOURCE_TOO_LARGE":   {"business_rejection", "not_retryable", false},
	"THREE_VIEW_SOURCE_REJECTED":    {"business_rejection", "not_retryable", false},

	// The host or a provider failed. Whether anything was applied is not
	// decidable from the code alone, so the caller must look the result up
	// before resending rather than send again blindly.
	"WORKSPACE_LOAD_FAILED":         {"technical_failure", "lookup_first", true},
	"SOURCE_INDEX_FAILED":           {"technical_failure", "lookup_first", true},
	"VIEW_FAILED":                   {"technical_failure", "lookup_first", true},
	"OPERATION_FAILED":              {"technical_failure", "lookup_first", true},
	"COMMAND_LIFECYCLE_UNAVAILABLE": {"technical_failure", "lookup_first", true},
	"COMMAND_FAILED":                {"technical_failure", "lookup_first", true},
	"XEAD_SOURCE_READ_FAILED":       {"technical_failure", "lookup_first", true},
	"XEAD_PARSE_FAILED":             {"technical_failure", "lookup_first", true},
	"XEAD_PROJECTION_FAILED":        {"technical_failure", "lookup_first", true},
	"THREE_VIEW_SOURCE_READ_FAILED": {"technical_failure", "lookup_first", true},
	"THREE_VIEW_PROJECTION_FAILED":  {"technical_failure", "lookup_first", true},
}

// classify fills the three axes from the table when the producer left them
// empty. It returns the receiver so it reads as one step at the call site.
func (f *Failure) classify() *Failure {
	if f == nil || f.Kind != "" {
		return f
	}
	c, ok := studioFailureCodes[f.Code]
	if !ok {
		c = studioFailureClassification{"technical_failure", "lookup_first", true}
	}
	f.Kind = c.kind
	f.Retryable = c.retryable
	f.EffectUnknown = c.effect
	return f
}

// classifiedFailure builds a Failure whose axes are already filled, so an
// in-process reader of the response sees the same classification the wire
// carries. MarshalJSON covers a Failure built directly anywhere; this covers
// the ones this package builds itself.
func classifiedFailure(code, stage, message string) *Failure {
	return (&Failure{Code: code, Stage: stage, Message: message}).classify()
}

// MarshalJSON emits the failure with its classification filled in.
//
// Doing it here rather than at each construction site is what makes the
// property hold: a Failure built directly anywhere in this package, or by a
// host composing it, still answers with all three axes. Without it the code and
// message would keep arriving and only the classification would go missing,
// which is exactly the silent drop this replaces.
func (f *Failure) MarshalJSON() ([]byte, error) {
	type failureAlias Failure
	return json.Marshal((*failureAlias)(f.classify()))
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

	// ThreeView is composed by the host. When nil, NavigateThreeView answers
	// THREE_VIEW_UNAVAILABLE; there is no built-in three-view reading.
	ThreeView ThreeViewFactory

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
		return "", classifiedFailure("WORKSPACE_NOT_ALLOWED", "", "workspace ID is not in this host's allowlist")
	}
	current, err := os.Stat(root)
	if err != nil || !os.SameFile(h.rootInfos[id], current) {
		return "", classifiedFailure("WORKSPACE_ROOT_CHANGED", "", "workspace root identity changed after host startup")
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
			return businessRejection(inspection.Code, "inspector could not resolve the requested subject")
		default:
			return technicalFailure("OPERATION_FAILED", "inspection service failed")
		}
	}
	return operationFailure(err)
}

func operationFailure(err error) *Failure {
	var validation *operations.ValidationFailure
	if errors.As(err, &validation) {
		switch validation.Stage {
		case operations.StageLoad:
			return technicalFailureAt("WORKSPACE_LOAD_FAILED", string(validation.Stage), "workspace loading failed")
		case operations.StageSourceIndex:
			return technicalFailureAt("SOURCE_INDEX_FAILED", string(validation.Stage), "source-index construction failed")
		}
		return technicalFailure("OPERATION_FAILED", "workspace operation failed")
	}
	var inspection *operations.InspectionError
	if errors.As(err, &inspection) {
		return businessRejection(inspection.Code, inspection.Message)
	}
	// Unclassified errors may contain absolute paths or host-private details.
	return technicalFailure("OPERATION_FAILED", "workspace operation failed")
}

// businessRejection marks a Failure the caller refused for a reason the
// request itself carries. Retrying the same request cannot change the answer.
func businessRejection(code, message string) *Failure {
	return &Failure{
		Code:      code,
		Message:   message,
		Kind:      string(failure.BusinessRejection),
		Retryable: string(failure.NotRetryable),
	}
}

// technicalFailure marks a Failure caused by the host or a provider. The same
// request may succeed later, but the effect state is unknown.
func technicalFailure(code, message string) *Failure {
	return &Failure{
		Code:          code,
		Message:       message,
		Kind:          string(failure.TechnicalFailure),
		Retryable:     string(failure.LookupFirst),
		EffectUnknown: true,
	}
}

func technicalFailureAt(code, stage, message string) *Failure {
	f := technicalFailure(code, message)
	f.Stage = stage
	return f
}

// commandFailure builds a Failure for an error that came back from a Command
// or Hook boundary. The code and message keep the values the boundary already
// reported; Kind, Retryable and EffectUnknown are read from the unified failure
// contract (internal/failure) when the error carries one. An ExecutionError or
// hook.Error therefore keeps its classification across the Studio bridge, so a
// client sees "business rejection, do not retry" instead of a bare code.
func commandFailure(code, stage, message string, err error) *Failure {
	f := &Failure{Code: code, Stage: stage, Message: message, EffectUnknown: command.EffectMayBeUnknown(err)}
	if unified, ok := failure.AsFailure(err); ok {
		f.Kind = string(unified.Kind())
		f.Retryable = string(unified.Retryable())
		f.EffectUnknown = f.EffectUnknown || unified.HasEffectUnknown()
	}
	// An error carrying no contract still gets the table's classification rather
	// than none, so this entrance cannot emit an unclassified failure either.
	return f.classify()
}
