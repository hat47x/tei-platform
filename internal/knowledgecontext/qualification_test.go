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
		References:         []sourceindex.Reference{{Role: sourceindex.RoleDefinition}},
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
