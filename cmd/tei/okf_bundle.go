package main

import (
	"fmt"
	"io"
	"strings"

	"tei/internal/okf"
	"tei/internal/workspace"
)

// okfBundleValue is the CLI projection of one read-only OKF bundle pass. It carries
// the bundle facts plus the invocation shape, so a consumer can tell "which bytes"
// apart from "which workspace".
//
// In mode=concept the concept list is omitted and the single selected concept is
// projected, because returning the whole bundle would make the requested concept
// indistinguishable from the rest.
type okfBundleValue struct {
	Root         string            `json:"root"`
	OkfVersion   string            `json:"okfVersion"`
	Revision     okf.Revision      `json:"bundleRevision"`
	Concepts     []okf.Concept     `json:"concepts,omitempty"`
	Concept      *okf.Concept      `json:"concept,omitempty"`
	Indexes      []okf.Index       `json:"indexes"`
	Logs         []okf.Log         `json:"logs"`
	Unresolved   []okf.ResolvedRef `json:"unresolvedReferences"`
	Format       okf.Format        `json:"format"`
	Diagnostics  []okf.Diagnostic  `json:"diagnostics"`
	NotEvaluated okf.NotEvaluated  `json:"notEvaluated"`
	Options      okf.Options       `json:"options"`
}

// newOKFBundleValue projects the reader's result. Every slice is non-nil so an
// empty bundle answers the same JSON type as a populated one.
func newOKFBundleValue(bundle *okf.Bundle) okfBundleValue {
	value := okfBundleValue{
		Root:         bundle.Root,
		OkfVersion:   bundle.OkfVersion,
		Revision:     bundle.Revision,
		Concepts:     append([]okf.Concept{}, bundle.Concepts...),
		Indexes:      append([]okf.Index{}, bundle.Indexes...),
		Logs:         append([]okf.Log{}, bundle.Logs...),
		Unresolved:   append([]okf.ResolvedRef{}, bundle.UnresolvedReferences...),
		Format:       bundle.Format,
		Diagnostics:  append([]okf.Diagnostic{}, bundle.Diagnostics...),
		NotEvaluated: bundle.NotEvaluated,
		Options:      bundle.Options,
	}
	if value.Format.FormatErrors == nil {
		value.Format.FormatErrors = []okf.Diagnostic{}
	}
	if value.LogosEmpty() {
		// no-op; kept for readability of the nil handling below
	}
	return value
}

// LogosEmpty is a placeholder that always reports false. It exists only so the
// projection's nil normalization stays in one place; delete it when a second caller
// needs the same check.
func (v okfBundleValue) LogosEmpty() bool { return false }

// okfBundleJSON is `tei okf-bundle`. It reads an OKF bundle and reports the format
// conformance and the documented entities. It performs no network access, starts no
// process, loads no plugin, and writes no file: the only output is this process's
// own stdout.
func okfBundleJSON(
	out io.Writer,
	rootPath, conceptID, expectedRevision, workspacePath, worldPath string,
	maxFileBytes, maxBundleBytes, maxConcepts int64,
	extraArgs int,
) (int, error) {
	const command = "okf-bundle"
	if strings.TrimSpace(rootPath) == "" {
		return resultInvalidArgument(out, command, "okf-bundle requires -root")
	}
	if worldPath != "" {
		return resultInvalidArgument(out, command, "okf-bundle reads a bundle directory; it does not accept -world")
	}
	if extraArgs != 0 {
		return resultInvalidArgument(out, command, "okf-bundle does not take positional arguments")
	}

	bundle, err := okf.Read(rootPath, okf.Options{
		MaxFileBytes:   maxFileBytes,
		MaxBundleBytes: maxBundleBytes,
		MaxConcepts:    int(maxConcepts),
	})
	if err != nil {
		// A root that cannot be opened is the caller's argument problem, not a
		// document format problem.
		return resultInvalidArgument(out, command, "okf-bundle: "+err.Error())
	}

	if expectedRevision != "" && bundle.Revision.Value != expectedRevision {
		return resultInvalidArgument(out, command,
			fmt.Sprintf("okf-bundle: bundle revision is %s, expected %s", bundle.Revision.Value, expectedRevision))
	}

	mode := resultModeBundle
	value := newOKFBundleValue(bundle)

	if concept := strings.TrimSpace(conceptID); concept != "" {
		mode = resultModeConcept
		selected := findOKFConcept(bundle, concept)
		if selected == nil {
			// A concept ID that names nothing is an argument problem, not an empty
			// result: the caller asked for one concept and there is none.
			return resultInvalidArgument(out, command,
				fmt.Sprintf("okf-bundle: no concept %q in this bundle; got %s", concept, strings.Join(okfConceptIDs(bundle), ", ")))
		}
		value.Concept = selected
		value.Concepts = nil
	}

	// The workspace identity is attached only when the caller named a workspace and
	// the bundle root is inside it. The reader does not infer an identity it did not
	// observe.
	if workspacePath != "" {
		loaded, err := workspace.Load(workspacePath)
		if err != nil {
			return resultInvalidArgument(out, command, "okf-bundle: -workspace could not be loaded: "+err.Error())
		}
		if !pathWithin(workspacePath, rootPath) {
			return resultInvalidArgument(out, command,
				"okf-bundle: -workspace names a workspace that does not contain -root; omit -workspace or point it at the containing workspace")
		}
		return emitResult(out, resultEnvelopeWithWorkspace(command, mode, value, workspaceIdentityPointer(loaded)), 0)
	}
	return emitResult(out, resultEnvelope(command, mode, value), 0)
}

// findOKFConcept returns a copy of the named concept, or nil.
func findOKFConcept(bundle *okf.Bundle, conceptID string) *okf.Concept {
	for i := range bundle.Concepts {
		if bundle.Concepts[i].ConceptID == conceptID {
			selected := bundle.Concepts[i]
			return &selected
		}
	}
	return nil
}

// okfConceptIDs lists the concept IDs of a bundle for an error message.
func okfConceptIDs(bundle *okf.Bundle) []string {
	ids := make([]string, 0, len(bundle.Concepts))
	for _, concept := range bundle.Concepts {
		ids = append(ids, concept.ConceptID)
	}
	return ids
}

// pathWithin reports whether candidate is inside root. Both values are compared
// after normalizing separators only; this is an advisory containment check for the
// CLI's arguments, not the reader's traversal boundary (the reader uses os.Root).
func pathWithin(root, candidate string) bool {
	cleanRoot := strings.TrimSuffix(strings.ReplaceAll(strings.TrimSpace(root), "\\", "/"), "/")
	cleanCandidate := strings.TrimSuffix(strings.ReplaceAll(strings.TrimSpace(candidate), "\\", "/"), "/")
	if cleanRoot == "" || cleanCandidate == "" {
		return false
	}
	return cleanCandidate == cleanRoot || strings.HasPrefix(cleanCandidate, cleanRoot+"/")
}
