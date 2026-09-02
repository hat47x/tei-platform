package validation

import (
	"testing"

	"tei/internal/application"
	"tei/internal/database"
	"tei/internal/ir"
)

func TestValidateReferenceContractQueryInvocationsAcceptsExplicitInvocation(t *testing.T) {
	world := referenceContractQueryWorld(PatternReferenceInvocationFixture())

	if got := ValidateReferenceContractQueryInvocations(world); len(got) != 0 {
		t.Fatalf("明示的で整合したContract Query呼び出しを拒否しています: %#v", got)
	}
}

func TestValidateReferenceContractQueryInvocationsRejectsNonReferencePattern(t *testing.T) {
	pattern := PatternReferenceInvocationFixture()
	pattern.Kind = application.PatternBrowse

	diagnostics := ValidateReferenceContractQueryInvocations(referenceContractQueryWorld(pattern))
	if !hasReferenceContractQueryDiagnostic(diagnostics, "CONTRACT_QUERY_INVOCATION_REQUIRES_REFERENCE_PATTERN") {
		t.Fatalf("reference以外のPatternでContract Query呼び出しを許可しています: %#v", diagnostics)
	}
}

func TestValidateReferenceContractQueryInvocationsRequiresApplicationContract(t *testing.T) {
	pattern := PatternReferenceInvocationFixture()
	pattern.ContractQuery.Contract = "master_data.unrelated"

	diagnostics := ValidateReferenceContractQueryInvocations(referenceContractQueryWorld(pattern))
	if !hasReferenceContractQueryDiagnostic(diagnostics, "REFERENCE_CONTRACT_QUERY_CONTRACT_NOT_REQUIRED") {
		t.Fatalf("Applicationが要求していないContractの呼び出しを拒否していません: %#v", diagnostics)
	}
}

func TestValidateReferenceContractQueryInvocationsRejectsAmbiguousBindings(t *testing.T) {
	pattern := PatternReferenceInvocationFixture()
	pattern.ContractQuery.Inputs = append(pattern.ContractQuery.Inputs,
		application.ContractQueryInputBinding{
			TargetAttribute: "customer.customer.customer_id",
			Input:           "another_id",
		},
		application.ContractQueryInputBinding{
			TargetAttribute: "customer.customer.tenant_id",
			Input:           "id",
		},
	)

	diagnostics := ValidateReferenceContractQueryInvocations(referenceContractQueryWorld(pattern))
	if !hasReferenceContractQueryDiagnostic(diagnostics, "DUPLICATE_REFERENCE_CONTRACT_QUERY_TARGET_ATTRIBUTE") {
		t.Fatalf("同じtarget attributeを複数inputへ割り当てる曖昧さを拒否していません: %#v", diagnostics)
	}
	if !hasReferenceContractQueryDiagnostic(diagnostics, "DUPLICATE_REFERENCE_CONTRACT_QUERY_INPUT") {
		t.Fatalf("同じQuery inputへ複数target attributeを割り当てる曖昧さを拒否していません: %#v", diagnostics)
	}
}

func TestValidateReferenceContractQueryInvocationsReportsPatternSubject(t *testing.T) {
	pattern := PatternReferenceInvocationFixture()
	pattern.ContractQuery.Query = ""

	diagnostics := ValidateReferenceContractQueryInvocations(referenceContractQueryWorld(pattern))
	for _, diagnostic := range diagnostics {
		if diagnostic.Code != "REFERENCE_CONTRACT_QUERY_NAME_REQUIRED" {
			continue
		}
		if diagnostic.Subject == nil || diagnostic.Subject.Kind != "application-pattern" || diagnostic.Subject.ID != "order_entry.customer_reference" {
			t.Fatalf("Diagnosticがauthoring元のPattern identityへ結び付いていません: %#v", diagnostic)
		}
		return
	}
	t.Fatalf("Query名欠如のDiagnosticがありません: %#v", diagnostics)
}

func PatternReferenceInvocationFixture() application.Pattern {
	return application.Pattern{
		ID:         "order_entry.customer_reference",
		Kind:       application.PatternReference,
		Relation:   "sales.order.customer",
		Datasource: "master",
		ContractQuery: &application.ContractQueryInvocation{
			Contract: "master_data.customer",
			Query:    "find_by_id",
			Inputs: []application.ContractQueryInputBinding{
				{
					TargetAttribute: "customer.customer.customer_id",
					Input:           "id",
				},
			},
		},
	}
}

func referenceContractQueryWorld(pattern application.Pattern) ir.World {
	return ir.World{
		Applications: []application.Application{
			{
				ID: "order_entry",
				Databases: []application.DatabaseRequirement{
					{
						Alias:          "master",
						TargetDatabase: "master_data",
						Access:         database.AccessRead,
						RequiredContracts: []database.ContractRequirement{
							{ContractID: "master_data.customer", CompatibleMajor: 2},
						},
					},
				},
				Patterns: []application.Pattern{pattern},
			},
		},
	}
}

func hasReferenceContractQueryDiagnostic(diagnostics []Diagnostic, code string) bool {
	for _, diagnostic := range diagnostics {
		if diagnostic.Code == code {
			return true
		}
	}
	return false
}
