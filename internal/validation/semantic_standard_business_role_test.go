package validation

import (
	"testing"

	"tei/internal/ir"
	"tei/internal/semantic"
	"tei/internal/syntax"
)

// TestValidateStandardBusinessRolesAreAccepted checks that every role a
// standard business binder reads is a role the validation accepts. Without
// this, a declaration could be written and then refused by the checker, so the
// binder would never see a row.
func TestValidateStandardBusinessRolesAreAccepted(t *testing.T) {
	roles := []string{
		"tax-applies-from", "tax-applies-until", "tax-category", "tax-value", "tax-rounding",
		"currency-rate-currency", "currency-rate-fiscal-year", "currency-rate-month-sequence",
		"currency-rate-middle", "currency-rate-buying", "currency-rate-selling",
		"numbering-counter", "numbering-digits", "numbering-prefix", "numbering-check", "numbering-overflow",
		"holiday-date",
	}
	for _, role := range roles {
		src := "namespace std\n\ndata Rate {\n    attribute value type std.decimal role " + role + "\n}\n"
		namespace, err := syntax.NewParser(src).ParseSemanticFile()
		if err != nil {
			t.Fatalf("%s: parse: %v", role, err)
		}
		diagnostics := Validate(ir.World{Semantics: []semantic.Namespace{namespace}})
		for _, diagnostic := range diagnostics {
			if diagnostic.Code == "SEMANTIC_ATTRIBUTE_ROLE_UNSUPPORTED" {
				t.Fatalf("%s: %s", role, diagnostic.Message)
			}
		}
	}
}

// TestValidateStandardBusinessRoleTypoIsReported checks that a misspelt role
// is still refused, so the vocabulary extension does not accept anything.
func TestValidateStandardBusinessRoleTypoIsReported(t *testing.T) {
	src := "namespace std\n\ndata Rate {\n    attribute value type std.decimal role tax-valuee\n}\n"
	namespace, err := syntax.NewParser(src).ParseSemanticFile()
	if err != nil {
		t.Fatal(err)
	}
	if diagnosticByCode(Validate(ir.World{Semantics: []semantic.Namespace{namespace}}), "SEMANTIC_ATTRIBUTE_ROLE_UNSUPPORTED") == nil {
		t.Fatal("a misspelt standard business role was accepted")
	}
}
