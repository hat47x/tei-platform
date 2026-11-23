package validationgen

import (
	"tei/internal/codegen/goemit"
	"tei/internal/core"
	"tei/internal/semantic"
)

// GeneratePositive lowers one typed canonical positive Validation to Go and
// returns the rule file: the decision function (for example
// QuantityIsPositive) and the check function (CheckQuantityIsPositive).
// The generated code imports only the standard library. The Result type and
// ParseDecimal it uses come in result.go and decimal.go, which
// goemit.Emit returns together with this file.
//
// The caller must first select and resolve the Rule and its Attribute type in
// the product Workspace. This lowering accepts only std.decimal and positive.
// It is not a Plugin selection, signed artifact, or Owner acceptance.
//
// The work is done by goemit. This function remains until its callers move to
// goemit.Emit (#5770).
func GeneratePositive(rule semantic.Validation, attributeType core.CanonicalID, packageName string) ([]byte, error) {
	out := goemit.Emit(goemit.Input{
		PackageName: packageName,
		Rules:       []goemit.Rule{{Validation: rule, AttributeType: attributeType}},
	})
	if err := out.Err(); err != nil {
		return nil, err
	}
	return out.File(out.Trace[0].File), nil
}
