package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"tei/internal/validation"
	"tei/internal/workspace"
)

// secondApplicationS4ProbeErrors writes the revised three-source Proposal into a temporary copy of the
// actual Reference Workspace (never the Repository's authoritative reference/workspace), loads it with the
// shared loader and returns the blocking Validation diagnostics. mutate may break the candidate to prove
// that the Validation path is not silently skipped.
func secondApplicationS4ProbeErrors(t *testing.T, mutate func(candidate *secondApplicationS4CompositionCandidate)) []validation.Diagnostic {
	t.Helper()

	root := repoPath()
	probeRoot := filepath.Join(t.TempDir(), "workspace")
	copyDirectory(t, filepath.Join(root, "reference", "workspace"), probeRoot)

	candidate := secondApplicationS4Composition(secondApplicationS3Decision())
	if mutate != nil {
		mutate(&candidate)
	}
	sources := map[string]string{
		filepath.Join("semantic", "repair-intake", "intake.tei"):                             candidate.SemanticSource,
		filepath.Join("applications", "repair-intake", "application.tei"):                    candidate.ApplicationSource,
		filepath.Join("applications", "repair-intake", "patterns", "customer-reference.tei"): candidate.PatternSource,
	}
	for relative, source := range sources {
		path := filepath.Join(probeRoot, relative)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(source), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	loaded, err := workspace.Load(probeRoot)
	if err != nil {
		t.Fatal(err)
	}
	var blocking []validation.Diagnostic
	for _, diagnostic := range validation.Validate(loaded.World) {
		if diagnostic.Severity == validation.Error {
			blocking = append(blocking, diagnostic)
		}
	}
	return blocking
}

// replaceExactlyOnce fails the test when the negative control would silently be a no-op.
func replaceExactlyOnce(t *testing.T, source, old, replacement string) string {
	t.Helper()
	if count := strings.Count(source, old); count != 1 {
		t.Fatalf("negative control setup: %q occurs %d times, want exactly 1", old, count)
	}
	return strings.Replace(source, old, replacement, 1)
}

func hasSecondApplicationDiagnostic(diagnostics []validation.Diagnostic, code string) bool {
	for _, diagnostic := range diagnostics {
		if diagnostic.Code == code {
			return true
		}
	}
	return false
}

func TestSecondApplicationS4ThreeSourceProposalPassesFullWorkspaceValidation(t *testing.T) {
	if blocking := secondApplicationS4ProbeErrors(t, nil); len(blocking) != 0 {
		t.Fatalf("revised Second Application Proposal failed full Validation: %#v", blocking)
	}
}

func TestSecondApplicationS4FullValidationRejectsUnknownRelation(t *testing.T) {
	blocking := secondApplicationS4ProbeErrors(t, func(candidate *secondApplicationS4CompositionCandidate) {
		candidate.PatternSource = replaceExactlyOnce(t, candidate.PatternSource,
			"relation repair_intake.intake.customer\n", "relation repair_intake.intake.missing\n")
	})
	if !hasSecondApplicationDiagnostic(blocking, "UNKNOWN_APPLICATION_PATTERN_RELATION") {
		t.Fatalf("unknown Relation did not reach ordinary Validation: %#v", blocking)
	}
}

func TestSecondApplicationS4FullValidationRejectsQueryCardinalityMismatch(t *testing.T) {
	blocking := secondApplicationS4ProbeErrors(t, func(candidate *secondApplicationS4CompositionCandidate) {
		candidate.SemanticSource = replaceExactlyOnce(t, candidate.SemanticSource,
			"relation customer to customer.customer one\n", "relation customer to customer.customer many\n")
	})
	if !hasSecondApplicationDiagnostic(blocking, "REFERENCE_CONTRACT_QUERY_RESULT_CARDINALITY_MISMATCH") {
		t.Fatalf("Relation/Query cardinality mismatch did not reach ordinary Validation: %#v", blocking)
	}
}
