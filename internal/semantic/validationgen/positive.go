package validationgen

import (
	"fmt"
	"go/format"
	"go/token"
	"strings"

	"tei/internal/core"
	"tei/internal/semantic"
)

// GeneratePositive produces executable Go behavior from a typed canonical
// Validation. It shares the existing validationeval Source/Result contract;
// it does not introduce a second reader, meaning model, or runtime API.
//
// The caller must first select and resolve the Rule and its Attribute type in
// the product Workspace. This small lowering accepts only std.decimal and
// positive. It is not a Plugin selection, signed artifact, or Owner acceptance.
func GeneratePositive(rule semantic.Validation, attributeType core.CanonicalID, packageName string) ([]byte, error) {
	if !token.IsIdentifier(packageName) {
		return nil, fmt.Errorf("invalid Go package name %q", packageName)
	}
	if rule.Kind != semantic.ValidationPositive {
		return nil, fmt.Errorf("semantic validation %s: unsupported generated kind %q", rule.ID, rule.Kind)
	}
	if rule.ID == "" || rule.Attribute == "" || rule.Relation != "" {
		return nil, fmt.Errorf("positive validation must have an ID, one Attribute, and no Relation: %+v", rule)
	}
	if attributeType != "std.decimal" {
		return nil, fmt.Errorf("semantic validation %s: unsupported Attribute type %q (requires resolved std.decimal)", rule.ID, attributeType)
	}

	var b strings.Builder
	b.WriteString("// Code generated from a selected TEI semantic.Validation; DO NOT EDIT.\n")
	b.WriteString("// Go lowering only: not an accepted Plugin artifact or execution grant.\n")
	fmt.Fprintf(&b, "package %s\n\n", packageName)
	b.WriteString("import (\n\t\"fmt\"\n\t\"tei/internal/semantic/validationeval\"\n)\n\n")
	fmt.Fprintf(&b, "// EvaluateSelected evaluates %q using the original Source contract.\n", rule.ID)
	b.WriteString("func EvaluateSelected(source validationeval.Source) (validationeval.Result, error) {\n")
	fmt.Fprintf(&b, "\tif source == nil { return \"\", fmt.Errorf(%q) }\n",
		fmt.Sprintf("no Source is specified to evaluate semantic validation %s", rule.ID))
	fmt.Fprintf(&b, "\tvalue, err := source.Number(%q)\n", string(rule.Attribute))
	fmt.Fprintf(&b, "\tif err != nil { return \"\", fmt.Errorf(%q, err) }\n",
		fmt.Sprintf("semantic validation %s cannot evaluate attribute %s: %%w", rule.ID, rule.Attribute))
	fmt.Fprintf(&b, "\tif value == nil { return \"\", fmt.Errorf(%q) }\n",
		fmt.Sprintf("semantic validation %s has no evaluable number for attribute %s", rule.ID, rule.Attribute))
	b.WriteString("\tif value.Sign() > 0 { return validationeval.ResultSatisfied, nil }\n")
	b.WriteString("\treturn validationeval.ResultNotSatisfied, nil\n")
	b.WriteString("}\n")
	result, err := format.Source([]byte(b.String()))
	if err != nil {
		return nil, fmt.Errorf("format generated validation Go: %w", err)
	}
	return result, nil
}
