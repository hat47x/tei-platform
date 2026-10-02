package knowledgecontext

import (
	"testing"

	"tei/internal/index"
	"tei/internal/sourceindex"
	"tei/internal/subject"
)

func TestQualifyApplicableWithAuthoritativeSourceAndExactDependency(t *testing.T) {
	ctx := Context{
		Subject:            subject.Subject{Kind: "application-pattern", ID: "order_entry.customer_reference"},
		References: []sourceindex.Reference{{
			Role:         sourceindex.RoleDefinition,
			ArtifactPath: "applications/order_entry/patterns/customer_reference.tei",
		}},
		Artifacts: []sourceindex.Artifact{{
			Path:        "applications/order_entry/patterns/customer_reference.tei",
			Fingerprint: "sha256:pattern",
		}},
		ApplicationPattern: &index.ApplicationPatternIndexEntry{Application: "order_entry"},
		Outgoing: []Edge{{
			From:     subject.Subject{Kind: "application-pattern", ID: "order_entry.customer_reference"},
			Relation: "contract-query",
			To:       subject.Subject{Kind: "database-contract-query", ID: "master_data.customer/find_by_id"},
		}},
	}
	result, err := Qualify(ctx, QualificationRequirements{
		ExpectedKind: "application-pattern",
		Application:  "order_entry",
		RequiredDependencies: []DependencyRequirement{{
			Direction: "outgoing", Relation: "contract-query", Kind: "database-contract-query", ID: "master_data.customer/find_by_id",
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != QualificationApplicable {
		t.Fatalf("status=%s reasons=%+v", result.Status, result.Reasons)
	}
	if len(result.Evidence) != 1 {
		t.Fatalf("evidence=%+v", result.Evidence)
	}
	evidence := result.Evidence[0]
	if evidence.SubjectKind != "application-pattern" ||
		evidence.Subject != "order_entry.customer_reference" ||
		evidence.Source != "applications/order_entry/patterns/customer_reference.tei" ||
		evidence.SourceFingerprint != "sha256:pattern" {
		t.Fatalf("exact qualification evidence=%+v", evidence)
	}
}

func TestQualifyApplicationScopeMismatchIsIncompatible(t *testing.T) {
	ctx := Context{
		Subject:            subject.Subject{Kind: "application-pattern", ID: "order_entry.customer_reference"},
		References:         []sourceindex.Reference{{Role: sourceindex.RoleDefinition}},
		ApplicationPattern: &index.ApplicationPatternIndexEntry{Application: "order_entry"},
	}
	result, err := Qualify(ctx, QualificationRequirements{ExpectedKind: "application-pattern", Application: "repair_intake"})
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != QualificationIncompatible {
		t.Fatalf("status=%s reasons=%+v", result.Status, result.Reasons)
	}
	if len(result.Evidence) != 0 {
		t.Fatalf("incompatible qualification leaked reusable evidence: %+v", result.Evidence)
	}
}

func TestQualifyUnknownWithoutAuthoritativeSource(t *testing.T) {
	ctx := Context{
		Subject:    subject.Subject{Kind: "semantic-data", ID: "customer.customer"},
		References: []sourceindex.Reference{{Role: sourceindex.RoleDerived}},
	}
	result, err := Qualify(ctx, QualificationRequirements{ExpectedKind: "semantic-data"})
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != QualificationUnknown {
		t.Fatalf("status=%s reasons=%+v", result.Status, result.Reasons)
	}
	if len(result.Evidence) != 0 {
		t.Fatalf("unknown qualification leaked reusable evidence: %+v", result.Evidence)
	}
}

func TestQualifyStaysUnknownWhenExactAuthoritativeEvidenceCannotBeBuilt(t *testing.T) {
	ctx := Context{
		Subject: subject.Subject{Kind: "semantic-data", ID: "customer.customer"},
		References: []sourceindex.Reference{{
			Role:         sourceindex.RoleDefinition,
			ArtifactPath: "semantic/customer/customer.tei",
		}},
	}
	result, err := Qualify(ctx, QualificationRequirements{ExpectedKind: "semantic-data"})
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != QualificationUnknown {
		t.Fatalf("status=%s reasons=%+v", result.Status, result.Reasons)
	}
	if len(result.Evidence) != 0 {
		t.Fatalf("unknown qualification leaked Evidence: %+v", result.Evidence)
	}
	var found bool
	for _, reason := range result.Reasons {
		if reason.Code == "exact-authoritative-evidence-unavailable" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("exact Evidence failure was not explained: %+v", result.Reasons)
	}
}

func TestQualifyMissingDependencyFailsClosed(t *testing.T) {
	ctx := Context{
		Subject:    subject.Subject{Kind: "semantic-data", ID: "customer.customer"},
		References: []sourceindex.Reference{{Role: sourceindex.RoleDefinition}},
	}
	result, err := Qualify(ctx, QualificationRequirements{
		ExpectedKind:         "semantic-data",
		RequiredDependencies: []DependencyRequirement{{Direction: "outgoing", Relation: "attribute", Kind: "semantic-attribute", ID: "customer.customer.missing"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != QualificationIncompatible {
		t.Fatalf("status=%s reasons=%+v", result.Status, result.Reasons)
	}
}

func TestQualifyMissingDependencyStaysUnknownWhenContextIsUnresolved(t *testing.T) {
	ctx := Context{
		Subject:    subject.Subject{Kind: "semantic-data", ID: "customer.customer"},
		References: []sourceindex.Reference{{Role: sourceindex.RoleDefinition}},
		Unresolved: []string{"semantic-element-not-found-in-derived-index"},
	}
	result, err := Qualify(ctx, QualificationRequirements{
		ExpectedKind:         "semantic-data",
		RequiredDependencies: []DependencyRequirement{{Direction: "outgoing", Relation: "attribute", Kind: "semantic-attribute", ID: "customer.customer.customer_id"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != QualificationUnknown {
		t.Fatalf("status=%s reasons=%+v", result.Status, result.Reasons)
	}
}

func TestQualifyMissingDependencyWithDerivedOnlySourceStaysUnknown(t *testing.T) {
	ctx := Context{
		Subject:    subject.Subject{Kind: "semantic-data", ID: "customer.customer"},
		References: []sourceindex.Reference{{Role: sourceindex.RoleDerived}},
	}
	result, err := Qualify(ctx, QualificationRequirements{
		ExpectedKind:         "semantic-data",
		RequiredDependencies: []DependencyRequirement{{Direction: "outgoing", Relation: "attribute", Kind: "semantic-attribute", ID: "customer.customer.customer_id"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != QualificationUnknown {
		t.Fatalf("status=%s reasons=%+v", result.Status, result.Reasons)
	}
}

func TestQualifyRejectsAmbiguousRequirements(t *testing.T) {
	_, err := Qualify(Context{}, QualificationRequirements{ExpectedKind: "semantic-data", Application: "order_entry"})
	if err == nil {
		t.Fatal("application scope on non-pattern qualification was accepted")
	}
}
