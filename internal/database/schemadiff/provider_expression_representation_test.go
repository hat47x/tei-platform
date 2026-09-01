package schemadiff_test

import (
	"testing"

	"tei/internal/database/physical"
	"tei/internal/database/schemadiff"
)

func TestAuthoredExpressionsWithDifferentTextAreDifferentIntent(t *testing.T) {
	desired := expressionIndexWithRepresentation("lower(name)", physical.ProviderExpressionAuthored, nil)
	actual := expressionIndexWithRepresentation("upper(name)", physical.ProviderExpressionAuthored, nil)
	actual.State = &physical.IndexState{Valid: true, Ready: true, Live: true}

	diff, err := schemadiff.Compare(indexSnapshot(desired), indexSnapshot(actual))
	if err != nil {
		t.Fatal(err)
	}
	assertIndexReplacement(t, diff)
}

func TestAuthoredAndProviderNormalizedDifferentTextAreUnresolved(t *testing.T) {
	desired := expressionIndexWithRepresentation("lower(name)", physical.ProviderExpressionAuthored, nil)
	actual := expressionIndexWithRepresentation(
		"lower((name)::text)",
		physical.ProviderExpressionProviderNormalized,
		&physical.ExpressionNormalization{Authority: "postgresql-catalog-deparse"},
	)
	actual.State = &physical.IndexState{Valid: true, Ready: true, Live: true}

	diff, err := schemadiff.Compare(indexSnapshot(desired), indexSnapshot(actual))
	if err != nil {
		t.Fatal(err)
	}
	if len(diff.Changes) != 0 || len(diff.Unresolved) != 1 {
		t.Fatalf("diff=%#v", diff)
	}
}

func TestProviderNormalizedExpressionsWithSameKnownContractCanDifferStructurally(t *testing.T) {
	normalization := &physical.ExpressionNormalization{Authority: "postgresql-catalog-deparse", ProviderVersion: "17"}
	desired := expressionIndexWithRepresentation("lower(name)", physical.ProviderExpressionProviderNormalized, normalization)
	actual := expressionIndexWithRepresentation(
		"upper(name)",
		physical.ProviderExpressionProviderNormalized,
		&physical.ExpressionNormalization{Authority: "postgresql-catalog-deparse", ProviderVersion: "17"},
	)
	actual.State = &physical.IndexState{Valid: true, Ready: true, Live: true}

	diff, err := schemadiff.Compare(indexSnapshot(desired), indexSnapshot(actual))
	if err != nil {
		t.Fatal(err)
	}
	assertIndexReplacement(t, diff)
}

func TestProviderNormalizedExpressionsWithoutKnownVersionRemainUnresolved(t *testing.T) {
	desired := expressionIndexWithRepresentation(
		"lower(name)",
		physical.ProviderExpressionProviderNormalized,
		&physical.ExpressionNormalization{Authority: "postgresql-catalog-deparse"},
	)
	actual := expressionIndexWithRepresentation(
		"lower((name)::text)",
		physical.ProviderExpressionProviderNormalized,
		&physical.ExpressionNormalization{Authority: "postgresql-catalog-deparse"},
	)
	actual.State = &physical.IndexState{Valid: true, Ready: true, Live: true}

	diff, err := schemadiff.Compare(indexSnapshot(desired), indexSnapshot(actual))
	if err != nil {
		t.Fatal(err)
	}
	if len(diff.Changes) != 0 || len(diff.Unresolved) != 1 {
		t.Fatalf("diff=%#v", diff)
	}
}

func TestProviderNormalizedExpressionsWithDifferentAuthorityRemainUnresolved(t *testing.T) {
	desired := expressionIndexWithRepresentation(
		"lower(name)",
		physical.ProviderExpressionProviderNormalized,
		&physical.ExpressionNormalization{Authority: "postgresql-catalog-deparse", ProviderVersion: "17"},
	)
	actual := expressionIndexWithRepresentation(
		"lower((name)::text)",
		physical.ProviderExpressionProviderNormalized,
		&physical.ExpressionNormalization{Authority: "other-normalizer", ProviderVersion: "17"},
	)
	actual.State = &physical.IndexState{Valid: true, Ready: true, Live: true}

	diff, err := schemadiff.Compare(indexSnapshot(desired), indexSnapshot(actual))
	if err != nil {
		t.Fatal(err)
	}
	if len(diff.Changes) != 0 || len(diff.Unresolved) != 1 {
		t.Fatalf("diff=%#v", diff)
	}
}

func TestEqualExpressionTextConvergesAcrossRepresentations(t *testing.T) {
	desired := expressionIndexWithRepresentation("lower(name)", physical.ProviderExpressionAuthored, nil)
	actual := expressionIndexWithRepresentation(
		"lower(name)",
		physical.ProviderExpressionProviderNormalized,
		&physical.ExpressionNormalization{Authority: "postgresql-catalog-deparse"},
	)
	actual.State = &physical.IndexState{Valid: true, Ready: true, Live: true}

	diff, err := schemadiff.Compare(indexSnapshot(desired), indexSnapshot(actual))
	if err != nil {
		t.Fatal(err)
	}
	if len(diff.Changes) != 0 || len(diff.Unresolved) != 0 {
		t.Fatalf("equal text did not converge: %#v", diff)
	}
}

func expressionIndexWithRepresentation(source string, representation physical.ProviderExpressionRepresentation, normalization *physical.ExpressionNormalization) physical.Index {
	index := expressionIndex(source)
	index.Keys[0].Expression.Representation = representation
	index.Keys[0].Expression.Normalization = normalization
	return index
}

func assertIndexReplacement(t *testing.T, diff schemadiff.Diff) {
	t.Helper()
	if len(diff.Unresolved) != 0 || len(diff.Changes) != 2 {
		t.Fatalf("diff=%#v", diff)
	}
	if diff.Changes[0].Kind != schemadiff.DropIndex || diff.Changes[1].Kind != schemadiff.AddIndex {
		t.Fatalf("changes=%#v", diff.Changes)
	}
}
