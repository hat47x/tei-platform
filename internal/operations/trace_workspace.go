package operations

import (
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

// Keep path normalization, symlink containment and regular-file checks at
// this read boundary. The index validator detects drift using fingerprints,
// but it does not provide a filesystem transaction or close the documented
// symlink-swap race (#1017).
func readWorkspaceSource(root string) (func(string) ([]byte, error), error) {
	rootReal, err := filepath.EvalSymlinks(root)
	if err != nil {
		return nil, fmt.Errorf("resolve Workspace root: %w", err)
	}
	return func(raw string) ([]byte, error) {
		relative, err := sourceindex.NormalizeWorkspacePath(raw)
		if err != nil {
			return nil, err
		}
		physical, err := filepath.EvalSymlinks(filepath.Join(rootReal, filepath.FromSlash(relative)))
		if err != nil {
			return nil, err
		}
		within, err := filepath.Rel(rootReal, physical)
		if err != nil || within == ".." || strings.HasPrefix(within, ".."+string(os.PathSeparator)) {
			return nil, fmt.Errorf("source %q resolves outside Workspace", raw)
		}
		file, err := os.Open(physical)
		if err != nil {
			return nil, err
		}
		defer file.Close()
		info, err := file.Stat()
		if err != nil {
			return nil, err
		}
		if !info.Mode().IsRegular() {
			return nil, fmt.Errorf("source %q is not a regular file", raw)
		}
		return io.ReadAll(file)
	}, nil
}
