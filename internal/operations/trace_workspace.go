package operations

import (
	"fmt"
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
	AuthorityLimits        []string                  `json:"authorityLimits"`
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
	loaded, err := workspace.Load(root)
	if err != nil {
		return WorkspaceTrace{}, &ValidationFailure{Stage: StageLoad, Err: err}
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
	reader, err := readWorkspaceSource(root)
	if err != nil {
		return WorkspaceTrace{}, &ValidationFailure{Stage: StageSourceIndex, Err: err}
	}
	resolver, err := sourcebridge.NewResolver(projection.Graph, sources, reader)
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
	out := WorkspaceTrace{
		SchemaVersion:          "tei.operations.trace/v0",
		SnapshotFingerprint:    snapshotFP,
		WorldFingerprint:       worldFP,
		SourceIndexFingerprint: sourceFP,
		Trace:                  result,
		Source:                 resolver.Locate(key),
		ProjectionUnresolved:   append([]string(nil), projection.Unresolved...),
		AuthorityLimits:        append([]string(nil), projection.AuthorityLimits...),
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

// readWorkspaceSource returns the byte reader behind the Source Index
// Resolver. Each read goes through workspace.ReadSource: one traversal-resistant
// os.Root open, then a regular-file Stat and a bounded read on that same
// handle. There is no EvalSymlinks followed by os.Open, so a path component
// replaced by a symlink between a check and the open cannot redirect the read
// outside the Workspace (#1017).
//
// The root is canonicalized once, here, so an alias of the Workspace root
// (a symlink or junction) retargeted after the Source Index was built cannot
// redirect later reads either. This is still not a filesystem transaction:
// the index validator detects an in-place content change by fingerprint, and
// each read opens its own root, so the reader is not a pinned multi-file
// snapshot.
func readWorkspaceSource(root string) (func(string) ([]byte, error), error) {
	rootReal, err := filepath.EvalSymlinks(root)
	if err != nil {
		return nil, fmt.Errorf("resolve Workspace root: %w", err)
	}
	// EvalSymlinks does not resolve a Windows directory junction, so also
	// remember which directory the root was. A root alias retargeted later
	// fails the read instead of redirecting it. This narrows the window only;
	// the pinned open below is what contains each read.
	rootInfo, err := os.Stat(rootReal)
	if err != nil {
		return nil, fmt.Errorf("stat Workspace root: %w", err)
	}
	return func(raw string) ([]byte, error) {
		relative, err := sourceindex.NormalizeWorkspacePath(raw)
		if err != nil {
			return nil, err
		}
		if now, err := os.Stat(rootReal); err != nil || !os.SameFile(rootInfo, now) {
			return nil, fmt.Errorf("Workspace root changed after the Source Index was built")
		}
		return workspace.ReadSource(rootReal, relative, maxWorkspaceSourceBytes)
	}, nil
}
