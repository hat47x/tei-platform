package operations

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"tei/internal/index"
	"tei/internal/modelref"
	"tei/internal/modelref/derivedbridge"
	"tei/internal/modelref/sourcebridge"
	"tei/internal/sourceindex"
	"tei/internal/workspace"
)

// UnresolvedScopes names what each unresolved count measures, so the two are
// never compared as if they answered the same question (#3906).
type UnresolvedScopes struct {
	// TraceUnresolvedIsPerSubject states that trace.unresolved counts only the
	// edges of the subject this response traces.
	TraceUnresolvedIsPerSubject bool `json:"traceUnresolvedIsPerSubject"`
	// ProjectionUnresolvedIsWholeGraph states that projectionUnresolved counts
	// the edges dropped while the whole graph was built, so it is identical for
	// every subject traced from one Workspace.
	ProjectionUnresolvedIsWholeGraph bool `json:"projectionUnresolvedIsWholeGraph"`
	// WhichToTrustForTheSubject names the count that answers "is this subject
	// reachable", and why the other one does not.
	WhichToTrustForTheSubject string `json:"whichToTrustForTheSubject"`
}

// WorkspaceTrace is a read-only projection, not an accepted Canonical graph,
// a proposal, or authority to edit any located source.
type WorkspaceTrace struct {
	SchemaVersion          string                    `json:"schemaVersion"`
	SnapshotFingerprint    string                    `json:"snapshotFingerprint"`
	WorldFingerprint       string                    `json:"worldFingerprint"`
	SourceIndexFingerprint string                    `json:"sourceIndexFingerprint"`
	Trace                  modelref.TraceResult      `json:"trace"`
	Source                 sourcebridge.Navigation   `json:"source"`
	RelatedSources         []sourcebridge.Navigation `json:"relatedSources,omitempty"`
	ProjectionUnresolved   []string                  `json:"projectionUnresolved,omitempty"`
	// UnresolvedScopes states what each of the two counts above measures. They
	// answer different questions and a caller who compares them directly reads
	// a zero as "clean" when the graph is not (#3906): trace.unresolved counts
	// the edges of the traced subject, projectionUnresolved counts the edges
	// dropped while the whole graph was built, so the second is the same for
	// every subject in one Workspace and the first is per subject.
	UnresolvedScopes UnresolvedScopes `json:"unresolvedScopes"`
	AuthorityLimits  []string         `json:"authorityLimits"`
	// Workspace names the Workspace this trace was read from (#3181). The
	// same Canonical ID may exist in several Workspaces with different meaning,
	// and worldFingerprint identifies content, not the Workspace.
	Workspace *workspace.WorkspaceIdentity `json:"workspace,omitempty"`
}

// TraceSubjectKinds is the closed set of subject kinds the typed graph can
// carry: the projection's own list, not a second copy of it. The CLI advertises
// this same set for -subject-kind on trace (#3242, #3180).
var TraceSubjectKinds = derivedbridge.SupportedSubjectKinds

// supportedTraceKind reports whether derivedbridge can place this kind in the
// typed graph at all. It is narrower than supportedInspectionKind: inspect can
// read a candidate key or a functional dependency straight from the World, but
// the reference projection carries neither, so trace has nothing to walk. The
// set lives with the projection itself, so a kind added there is not silently
// accepted here (or refused here while the graph would carry it).
func supportedTraceKind(kind string) bool {
	return derivedbridge.SupportsSubjectKind(kind)
}

// TraceWorkspace joins the established Derived Index modelref projection and
// Source Index navigation. The only edge semantics are those provided by
// derivedbridge, which itself consumes existing declared dependencies.
func TraceWorkspace(root string, key modelref.Key, options modelref.TraceOptions) (WorkspaceTrace, error) {
	if strings.TrimSpace(key.Scope) == "" || key.Scope != strings.TrimSpace(key.Scope) ||
		strings.TrimSpace(key.Subject.Kind) == "" || key.Subject.Kind != strings.TrimSpace(key.Subject.Kind) ||
		strings.TrimSpace(key.Subject.ID) == "" || key.Subject.ID != strings.TrimSpace(key.Subject.ID) {
		return WorkspaceTrace{}, &modelref.TraceError{
			Code: "INVALID_SUBJECT", Message: "exact nonempty scope, subject kind and subject id are required",
		}
	}
	// A kind the typed graph cannot carry is not "a subject with no references".
	// Report it before loading anything, so a caller never reads SUBJECT_NOT_FOUND
	// as "this subject is unrelated to everything" (#3180). inspect supports a
	// wider kind set than the projection does; both fall outside the graph here.
	if !supportedTraceKind(key.Subject.Kind) {
		return WorkspaceTrace{}, &modelref.TraceError{
			Code: "UNSUPPORTED_SUBJECT_KIND",
			Message: fmt.Sprintf("trace does not project subject kind %q; the typed graph carries namespace, type, data, attribute, relation, validation, capability, application, application-pattern, query and the data layer (database, database-contract, database-contract-query)",
				key.Subject.Kind),
		}
	}

	loaded, err := workspace.Load(root)
	if err != nil {
		return WorkspaceTrace{}, &ValidationFailure{Stage: StageLoad, Err: err}
	}
	// Pin the Workspace root once for this trace, before the Source Index is
	// built, and read every Source through this handle until the trace ends.
	// Its error is reported below, where the reader used to be created.
	reader, readerErr := readWorkspaceSource(root)
	if readerErr == nil {
		defer reader.Close()
	}
	// The derived bridge intentionally coalesces repeated uses of a Subject.
	// For kinds that can be selected from Canonical World, first require one
	// and only one declaration using the same resolver as "inspect". This
	// prevents duplicate authoritative declarations from masquerading as a
	// uniquely resolved node in a read-only candidate projection.
	if supportedInspectionKind(key.Subject.Kind) {
		if _, err := inspectCanonical(loaded.World, key.Subject); err != nil {
			if finding, ok := err.(*InspectionError); ok {
				return WorkspaceTrace{}, &modelref.TraceError{
					Code: finding.Code, Message: finding.Message,
				}
			}
			return WorkspaceTrace{}, err
		}
	}
	// A request that TraceDirect is certain to reject (direction, mode, offset
	// or limit) fails here, before anything is built, as inspectCanonical's
	// findings do. Otherwise it would wait for the Source Index side below.
	if err := modelref.ValidateTraceRequest(key, options); err != nil {
		return WorkspaceTrace{}, err
	}
	// index.Build is not read-only on the World: resolving an Environment calls
	// plugin.NormalizeDefinition, which sorts the nested slices of the loaded
	// plugin definitions in place and reassigns their fields. Load has already
	// normalized them, so the content does not change, but the writes still
	// race with any concurrent reader, and the World fingerprint has always
	// been taken after them. Nothing else may read the World until Build has
	// returned.
	derived := index.Build(loaded.World)
	// From here the World is only read. The Source Index side and the World
	// serialization depend on neither the modelref projection nor the trace,
	// and for a large Workspace each is as expensive as the trace itself, so
	// they run beside it. Errors are still reported in the order the sequential
	// steps used to fail: trace, Source Index, reader, resolver, World
	// serialization, Source Index serialization. The goroutines do not recover:
	// a panic in either ends the process, as it did on the caller's goroutine,
	// but a caller that recovers cannot catch it.
	var (
		wg      sync.WaitGroup
		side    sourceIndexSide
		worldFP string
		worldFE error
	)
	wg.Add(2)
	go func() {
		defer wg.Done()
		worldFP, worldFE = worldFingerprint(loaded.World)
	}()
	go func() {
		defer wg.Done()
		side = buildSourceIndexSide(root, loaded)
	}()
	projection := derivedbridge.FromIndex(derived, key.Scope)
	// The operation-level expected fingerprint includes the Source Index and
	// normalized World, unlike modelref's graph-only candidate token.
	expected := options.ExpectedFingerprint
	options.ExpectedFingerprint = ""
	result, err := modelref.TraceDirect(projection.Graph, projection.Rules, key, options)
	// Join before any return below, so that no goroutine outlives the call.
	wg.Wait()
	if err != nil {
		return WorkspaceTrace{}, err
	}
	if side.buildErr != nil {
		return WorkspaceTrace{}, &ValidationFailure{Stage: StageSourceIndex, Err: side.buildErr}
	}
	sources := side.sources
	if readerErr != nil {
		return WorkspaceTrace{}, &ValidationFailure{Stage: StageSourceIndex, Err: readerErr}
	}
	resolver, err := sourcebridge.NewResolver(projection.Graph, sources, reader.Read)
	if err != nil {
		return WorkspaceTrace{}, &ValidationFailure{Stage: StageSourceIndex, Err: err}
	}
	if worldFE != nil {
		return WorkspaceTrace{}, worldFE
	}
	if side.fingerprintErr != nil {
		return WorkspaceTrace{}, side.fingerprintErr
	}
	sourceFP := side.fingerprint
	snapshotFP := workspaceTraceFingerprint(worldFP, sourceFP, result.SnapshotFingerprint)
	if expected != "" && expected != snapshotFP {
		return WorkspaceTrace{}, &modelref.TraceError{
			Code:    "STALE_SNAPSHOT",
			Message: "trace graph, selector, World, or Source Index differs from the prior page",
		}
	}
	identity := loaded.Identity()
	out := WorkspaceTrace{
		SchemaVersion:          "tei.operations.trace/v0",
		SnapshotFingerprint:    snapshotFP,
		WorldFingerprint:       worldFP,
		SourceIndexFingerprint: sourceFP,
		Trace:                  result,
		Source:                 resolver.Locate(key),
		ProjectionUnresolved:   append([]string(nil), projection.Unresolved...),
		UnresolvedScopes: UnresolvedScopes{
			TraceUnresolvedIsPerSubject:      true,
			ProjectionUnresolvedIsWholeGraph: true,
			WhichToTrustForTheSubject:        "result.trace.unresolved",
		},
		AuthorityLimits: append([]string(nil), projection.AuthorityLimits...),
		Workspace:       &identity,
	}
	// A page that can be continued must say, next to nextOffset, which value
	// continues it. The value is this Workspace-level token; modelref's own
	// graphFingerprint is not a continuation token (#3625).
	if out.Trace.NextOffset != nil {
		out.Trace.Continuation = &modelref.TraceContinuation{
			Fingerprint: snapshotFP,
			Field:       "result.snapshotFingerprint",
		}
	}
	if result.Mode == "summary" {
		return out, nil
	}

	// Navigate only subjects occurring on this explicit page. Do not run a
	// second Workspace/index scan for each neighbor.
	seen := map[modelref.Key]bool{key: true}
	var keys []modelref.Key
	add := func(k modelref.Key) {
		if !seen[k] {
			seen[k] = true
			keys = append(keys, k)
		}
	}
	for _, row := range result.Rows {
		add(row.Edge.From)
		if row.ResolvedTo != nil {
			add(*row.ResolvedTo)
		}
	}
	sort.Slice(keys, func(i, j int) bool {
		a, b := keys[i], keys[j]
		if a.Scope != b.Scope {
			return a.Scope < b.Scope
		}
		if a.Subject.Kind != b.Subject.Kind {
			return a.Subject.Kind < b.Subject.Kind
		}
		return a.Subject.ID < b.Subject.ID
	})
	for _, related := range keys {
		out.RelatedSources = append(out.RelatedSources, resolver.Locate(related))
	}
	return out, nil
}

// sourceIndexSide is the Source Index and its serialized fingerprint, built
// from the loaded World and the Workspace files only.
type sourceIndexSide struct {
	sources        sourceindex.Index
	fingerprint    string
	buildErr       error
	fingerprintErr error
}

func buildSourceIndexSide(root string, loaded workspace.Result) sourceIndexSide {
	var side sourceIndexSide
	side.sources, side.buildErr = workspace.BuildSourceIndex(root, loaded)
	if side.buildErr != nil {
		return side
	}
	side.fingerprint, side.fingerprintErr = sourceIndexFingerprint(side.sources)
	return side
}

// maxWorkspaceSourceBytes bounds the bytes one Source read may retain. A Source
// file above it is refused, never truncated. It is a variable only so a test
// can lower it; production code never assigns it.
var maxWorkspaceSourceBytes int64 = 64 << 20

// workspaceSourceReader is the byte reader behind the Source Index Resolver.
// It holds ONE os.Root, opened once per trace, and every read goes through that
// same handle: a regular-file check and a bounded read on the opened file. A
// Workspace root alias (symlink or junction) retargeted after the handle was
// opened therefore cannot redirect a later read, and a path component replaced
// below the root cannot escape it (#1017).
//
// What this does not do: it is not a multi-file snapshot (each read sees the
// file as it is then; the Source Index fingerprint detects in-place content
// changes), the root is pinned when the reader is created and not before
// (Load and the Source Index build resolve the root path on their own), and
// mount-level changes are outside os.Root's protection.
type workspaceSourceReader struct {
	root *os.Root
}

func readWorkspaceSource(root string) (*workspaceSourceReader, error) {
	handle, err := openWorkspaceSourceRoot(root)
	if err != nil {
		return nil, fmt.Errorf("open Workspace root: %w", err)
	}
	return &workspaceSourceReader{root: handle}, nil
}

// Close releases the pinned root. Reads after Close fail.
func (r *workspaceSourceReader) Close() error { return r.root.Close() }

// Read returns the bytes of one Workspace-relative regular file.
func (r *workspaceSourceReader) Read(raw string) ([]byte, error) {
	relative, err := sourceindex.NormalizeWorkspacePath(raw)
	if err != nil {
		return nil, err
	}
	limit := maxWorkspaceSourceBytes
	if limit <= 0 {
		return nil, workspace.ErrInvalidSourceReadLimit
	}
	file, err := r.root.OpenFile(filepath.FromSlash(relative), workspaceSourceOpenFlags, 0)
	if err != nil {
		return nil, fmt.Errorf("open Workspace source: %w", err)
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("stat Workspace source: %w", err)
	}
	if !info.Mode().IsRegular() {
		return nil, errors.New("Workspace source is not a regular file")
	}
	if info.Size() > limit {
		return nil, workspace.ErrSourceTooLarge
	}
	content, err := io.ReadAll(io.LimitReader(file, limit+1))
	if err != nil {
		return nil, fmt.Errorf("read Workspace source: %w", err)
	}
	if int64(len(content)) > limit {
		return nil, workspace.ErrSourceTooLarge
	}
	return content, nil
}
