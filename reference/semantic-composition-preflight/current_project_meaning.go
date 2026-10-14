package semanticcompositionpreflight

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"tei/internal/modelref"
	"tei/internal/sourceacceptance"
	"tei/internal/sourceindex"
	"tei/internal/validation"
	"tei/internal/workspace"
)

// ProjectMeaningSelection identifies one explicitly accepted imported
// namespace by typed logical identity and exact normalized semantic content.
// The scope is the modelref logical scope, not the Composition service scope.
// There is no imported namespace version in the current Workspace contract,
// therefore Ref.Version cannot be independently validated here.
type ProjectMeaningSelection struct {
	Alias          string       `json:"alias"`
	Resource       modelref.Ref `json:"resource"`
	SemanticDigest string       `json:"semanticDigest"`
}

// CurrentProjectMeaning is ONE Host-owned decision snapshot. Source and
// Selected must be read together from the same current acceptance decision;
// two independent reads would not prove their common revision or provenance.
// The receipt must not be constructed from a CLI parameter, imported manifest,
// self-attested JSON, or the preflight's signature verification result.
type CurrentProjectMeaning struct {
	Source   sourceacceptance.Record   `json:"source"`
	Selected []ProjectMeaningSelection `json:"selected"`
}

// CurrentProjectMeaningReader is a reference-only, read-only Host seam.
// Implementations are trusted Host responsibilities: authenticate decision
// origin, check revocation and return one coherent current decision snapshot.
// This interface does not implement a trust store, production acceptance,
// project authoring, or runtime authorization.
type CurrentProjectMeaningReader interface {
	ReadCurrentAcceptedProjectMeaning(context.Context, string) (CurrentProjectMeaning, error)
}

// VerifiedProjectMeaning contains the exact Workspace Load whose own managed
// Source bytes and selected external semantic digests were inspected.
// Although its receipt matches this inspection, neither the Reader's trust
// provenance nor current invocation authority is established by this type.
type VerifiedProjectMeaning struct {
	Loaded      workspace.Result
	SourceIndex sourceindex.Index
	Acceptance  CurrentProjectMeaning
}

type sourceReceiptReader struct{ record sourceacceptance.Record }

func (r sourceReceiptReader) ReadCurrentAcceptedSource(_ context.Context, _ string) (sourceacceptance.Record, error) {
	return r.record, nil
}

// InspectCurrentProjectMeaning demonstrates the necessary acceptance binding
// for current Path Workspace imports. It does NOT admit data-only/CAS assets,
// authorize a runtime, or generate an acceptance decision.
//
// The actual source acceptance verification is delegated to the existing
// sourceacceptance.VerifyCurrent, not reproduced. Every imported namespace
// must have exactly one typed and digest-fixed entry in the SAME Host decision.
// Unknown, duplicated, stale, and cross-identity entries fail closed.
func InspectCurrentProjectMeaning(
	ctx context.Context,
	reader CurrentProjectMeaningReader,
	applicationID, root string,
) (VerifiedProjectMeaning, error) {
	if reader == nil {
		return VerifiedProjectMeaning{}, errors.New("PROJECT_MEANING_READER_REQUIRED")
	}
	if err := ctx.Err(); err != nil {
		return VerifiedProjectMeaning{}, fmt.Errorf("PROJECT_MEANING_CURRENT_UNAVAILABLE: %w", err)
	}
	loaded, err := workspace.Load(root)
	if err != nil {
		return VerifiedProjectMeaning{}, fmt.Errorf("PROJECT_MEANING_WORKSPACE_LOAD_FAILED: %w", err)
	}
	index, err := workspace.BuildSourceIndex(root, loaded)
	if err != nil {
		return VerifiedProjectMeaning{}, fmt.Errorf("PROJECT_MEANING_SOURCE_INDEX_FAILED: %w", err)
	}
	if validation.HasErrors(validation.Validate(loaded.World)) {
		return VerifiedProjectMeaning{}, errors.New("PROJECT_MEANING_SEMANTIC_VALIDATION_FAILED")
	}
	receipt, err := reader.ReadCurrentAcceptedProjectMeaning(ctx, applicationID)
	if err != nil {
		return VerifiedProjectMeaning{}, fmt.Errorf("PROJECT_MEANING_CURRENT_UNAVAILABLE: %w", err)
	}
	if err := ctx.Err(); err != nil {
		return VerifiedProjectMeaning{}, fmt.Errorf("PROJECT_MEANING_CURRENT_UNAVAILABLE: %w", err)
	}
	currentSource, err := sourceacceptance.VerifyCurrent(
		ctx, sourceReceiptReader{record: receipt.Source}, applicationID, index,
	)
	if err != nil {
		return VerifiedProjectMeaning{}, fmt.Errorf("PROJECT_MEANING_SOURCE_NOT_ACCEPTED: %w", err)
	}
	if len(loaded.Imports) != len(receipt.Selected) {
		return VerifiedProjectMeaning{}, errors.New("PROJECT_MEANING_SELECTION_SET_MISMATCH")
	}

	// Workspace Load has already rejected duplicate import aliases and logical
	// namespaces. The Host receipt must match that explicit set exactly;
	// neither a missing entry nor an unrelated additional one is optional.
	byAlias := make(map[string]workspace.ResolvedImport, len(loaded.Imports))
	for _, imported := range loaded.Imports {
		byAlias[imported.Alias] = imported
	}
	seen := make(map[string]bool, len(receipt.Selected))
	for _, selected := range receipt.Selected {
		if selected.Alias == "" || seen[selected.Alias] {
			return VerifiedProjectMeaning{}, errors.New("PROJECT_MEANING_SELECTION_AMBIGUOUS")
		}
		seen[selected.Alias] = true
		imported, ok := byAlias[selected.Alias]
		if !ok {
			return VerifiedProjectMeaning{}, fmt.Errorf("PROJECT_MEANING_SELECTION_UNDECLARED: %q", selected.Alias)
		}
		ref := selected.Resource
		if strings.TrimSpace(ref.Scope) == "" || ref.Scope != strings.TrimSpace(ref.Scope) ||
			strings.ContainsAny(ref.Scope, "\n\r\t") || ref.Subject.Kind != "semantic-namespace" ||
			ref.Subject.ID != imported.Namespace || ref.Version != "" {
			return VerifiedProjectMeaning{}, fmt.Errorf("PROJECT_MEANING_RESOURCE_MISMATCH: %q", selected.Alias)
		}
		if selected.SemanticDigest != imported.SemanticDigest {
			return VerifiedProjectMeaning{}, fmt.Errorf("PROJECT_MEANING_SEMANTIC_DIGEST_MISMATCH: %q", selected.Alias)
		}
	}
	receipt.Source = currentSource
	receipt.Selected = append([]ProjectMeaningSelection(nil), receipt.Selected...)
	return VerifiedProjectMeaning{Loaded: loaded, SourceIndex: index, Acceptance: receipt}, nil
}
