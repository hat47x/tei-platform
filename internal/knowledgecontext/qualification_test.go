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

func TestQualifyWithSourcesCapturesRequiredDependencyEvidence(t *testing.T) {
	ctx := Context{
		Subject: subject.Subject{Kind: "application-pattern", ID: "order_entry.customer_reference"},
		References: []sourceindex.Reference{{
			Subject:      subject.Subject{Kind: "application-pattern", ID: "order_entry.customer_reference"},
			Role:         sourceindex.RoleDefinition,
			ArtifactPath: "applications/order_entry/patterns/customer_reference.tei",
		}},
		Artifacts: []sourceindex.Artifact{{
			Path:        "applications/order_entry/patterns/customer_reference.tei",
			Fingerprint: "sha256:candidate",
		}},
		ApplicationPattern: &index.ApplicationPatternIndexEntry{Application: "order_entry"},
		Outgoing: []Edge{{
			From:     subject.Subject{Kind: "application-pattern", ID: "order_entry.customer_reference"},
			Relation: "contract-query",
			To:       subject.Subject{Kind: "database-contract-query", ID: "master_data.customer/find_by_id"},
		}},
	}
	sources := sourceindex.Index{
		Artifacts: []sourceindex.Artifact{
			{Path: "applications/order_entry/patterns/customer_reference.tei", Fingerprint: "sha256:candidate"},
			{Path: "database-contracts/master_data/customer.tei", Fingerprint: "sha256:dependency"},
		},
		References: []sourceindex.Reference{
			{
				Subject:      subject.Subject{Kind: "application-pattern", ID: "order_entry.customer_reference"},
				ArtifactPath: "applications/order_entry/patterns/customer_reference.tei",
				Role:         sourceindex.RoleDefinition,
			},
			{
				Subject:      subject.Subject{Kind: "database-contract-query", ID: "master_data.customer/find_by_id"},
				ArtifactPath: "database-contracts/master_data/customer.tei",
				Role:         sourceindex.RoleDefinition,
			},
		},
	}
	result, err := QualifyWithSources(ctx, sources, QualificationRequirements{
		ExpectedKind: "application-pattern",
		Application:  "order_entry",
		RequiredDependencies: []DependencyRequirement{{
			Direction: "outgoing",
			Relation:  "contract-query",
			Kind:      "database-contract-query",
			ID:        "master_data.customer/find_by_id",
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != QualificationApplicable {
		t.Fatalf("status=%s reasons=%+v", result.Status, result.Reasons)
	}
	if len(result.DependencyEvidence) != 1 {
		t.Fatalf("dependencyEvidence=%+v", result.DependencyEvidence)
	}
	evidence := result.DependencyEvidence[0]
	if evidence.SubjectKind != "database-contract-query" ||
		evidence.Subject != "master_data.customer/find_by_id" ||
		evidence.Source != "database-contracts/master_data/customer.tei" ||
		evidence.SourceFingerprint != "sha256:dependency" {
		t.Fatalf("dependency Evidence=%+v", evidence)
	}
}

func TestQualifyWithSourcesKeepsDependencyWithoutAuthoritativeEvidenceUnknown(t *testing.T) {
	ctx := Context{
		Subject: subject.Subject{Kind: "application-pattern", ID: "order_entry.customer_reference"},
		References: []sourceindex.Reference{{
			Subject:      subject.Subject{Kind: "application-pattern", ID: "order_entry.customer_reference"},
			Role:         sourceindex.RoleDefinition,
			ArtifactPath: "applications/order_entry/patterns/customer_reference.tei",
		}},
		Artifacts: []sourceindex.Artifact{{
			Path:        "applications/order_entry/patterns/customer_reference.tei",
			Fingerprint: "sha256:candidate",
		}},
		ApplicationPattern: &index.ApplicationPatternIndexEntry{Application: "order_entry"},
		Outgoing: []Edge{{
			From:     subject.Subject{Kind: "application-pattern", ID: "order_entry.customer_reference"},
			Relation: "contract-query",
			To:       subject.Subject{Kind: "database-contract-query", ID: "master_data.customer/find_by_id"},
		}},
	}
	sources := sourceindex.Index{
		Artifacts: []sourceindex.Artifact{
			{Path: "applications/order_entry/patterns/customer_reference.tei", Fingerprint: "sha256:candidate"},
			{Path: "runtime/customer.sql", Fingerprint: "sha256:implementation"},
		},
		References: []sourceindex.Reference{
			{
				Subject:      subject.Subject{Kind: "application-pattern", ID: "order_entry.customer_reference"},
				ArtifactPath: "applications/order_entry/patterns/customer_reference.tei",
				Role:         sourceindex.RoleDefinition,
			},
			{
				Subject:      subject.Subject{Kind: "database-contract-query", ID: "master_data.customer/find_by_id"},
				ArtifactPath: "runtime/customer.sql",
				Role:         sourceindex.RoleImplementation,
			},
		},
	}
	result, err := QualifyWithSources(ctx, sources, QualificationRequirements{
		ExpectedKind: "application-pattern",
		Application:  "order_entry",
		RequiredDependencies: []DependencyRequirement{{
			Direction: "outgoing",
			Relation:  "contract-query",
			Kind:      "database-contract-query",
			ID:        "master_data.customer/find_by_id",
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != QualificationUnknown {
		t.Fatalf("status=%s reasons=%+v", result.Status, result.Reasons)
	}
	if len(result.Evidence) != 0 || len(result.DependencyEvidence) != 0 {
		t.Fatalf("unknown qualification leaked reusable Evidence: evidence=%+v dependencies=%+v", result.Evidence, result.DependencyEvidence)
	}
	var found bool
	for _, reason := range result.Reasons {
		if reason.Code == "required-dependency-authoritative-evidence-unavailable" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("missing dependency Evidence was not explained: %+v", result.Reasons)
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
