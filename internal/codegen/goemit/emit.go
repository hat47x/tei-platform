package goemit

import (
	"bytes"
	"embed"
	"errors"
	"fmt"
	"go/format"
	"go/parser"
	"go/printer"
	"go/token"
	"sort"
	"strings"

	"tei/internal/core"
	"tei/internal/semantic"
)

// Diagnostic codes. Each one names a reason Emit produced no files.
const (
	CodeInvalidPackageName = "INVALID_PACKAGE_NAME"
	CodeUnsupportedKind    = "UNSUPPORTED_KIND"
	CodeInvalidRuleShape   = "INVALID_RULE_SHAPE"
	CodeUnsupportedType    = "UNSUPPORTED_TYPE"
	CodeNameInvalid        = "NAME_INVALID"
	CodeNameReserved       = "NAME_RESERVED"
	CodeNameCollision      = "NAME_COLLISION"
)

// SeverityError marks a diagnostic that stops emission.
const SeverityError = "error"

// Diagnostic explains why a rule could not be emitted. It is local to goemit
// until a shared diagnostic record exists (#5771).
type Diagnostic struct {
	Code     string
	Severity string
	// Element is the design element the diagnostic is about, when there is one.
	Element core.CanonicalID
	Message string
}

// Rule is one selected rule and the resolved type of its Attribute.
type Rule struct {
	Validation    semantic.Validation
	AttributeType core.CanonicalID
	// DisplayName is the business name of the rule, when the caller has one.
	// It appears only in doc comments.
	DisplayName string
}

// Input is everything Emit reads.
type Input struct {
	PackageName string
	Rules       []Rule
}

// File is one emitted Go file.
type File struct {
	Path    string
	Content []byte
}

// Trace ties one rule to the Go it produced.
type Trace struct {
	Element   core.CanonicalID
	Attribute core.CanonicalID
	File      string
	Decision  string
	Check     string
}

// String renders the trace as one line.
func (t Trace) String() string {
	return fmt.Sprintf("%s -> %s: %s, %s (value of %s)", t.Element, t.File, t.Decision, t.Check, t.Attribute)
}

// Output is the result of Emit. Files and Trace are empty whenever
// Diagnostics is not.
type Output struct {
	Files       []File
	Trace       []Trace
	Diagnostics []Diagnostic
}

// File returns the content emitted at path, or nil.
func (o Output) File(path string) []byte {
	for _, file := range o.Files {
		if file.Path == path {
			return file.Content
		}
	}
	return nil
}

// Err joins the diagnostic messages into one error, or returns nil.
func (o Output) Err() error {
	if len(o.Diagnostics) == 0 {
		return nil
	}
	messages := make([]string, len(o.Diagnostics))
	for i, diagnostic := range o.Diagnostics {
		messages[i] = diagnostic.Message
	}
	return errors.New(strings.Join(messages, "; "))
}

const decimalType core.CanonicalID = "std.decimal"

//go:embed internal/stdpart/result.go internal/stdpart/decimal.go
var standardParts embed.FS

// Emit produces the Go package for the given rules. The same input gives the
// same bytes, whatever the order of Rules.
func Emit(input Input) Output {
	rules := append([]Rule(nil), input.Rules...)
	sort.SliceStable(rules, func(i, j int) bool {
		a, b := rules[i].Validation, rules[j].Validation
		if a.ID != b.ID {
			return a.ID < b.ID
		}
		return a.Attribute < b.Attribute
	})

	var diagnostics []Diagnostic
	report := func(code string, element core.CanonicalID, format string, args ...any) {
		diagnostics = append(diagnostics, Diagnostic{
			Code: code, Severity: SeverityError, Element: element,
			Message: fmt.Sprintf(format, args...),
		})
	}
	if !token.IsIdentifier(input.PackageName) {
		report(CodeInvalidPackageName, "", "invalid Go package name %q", input.PackageName)
	}

	type named struct {
		rule  Rule
		names positiveNames
	}
	var accepted []named
	owner := map[string]core.CanonicalID{}
	for _, rule := range rules {
		v := rule.Validation
		switch {
		case v.Kind != semantic.ValidationPositive:
			report(CodeUnsupportedKind, v.ID, "semantic validation %s: unsupported generated kind %q", v.ID, v.Kind)
			continue
		case v.ID == "" || v.Attribute == "" || v.Relation != "":
			report(CodeInvalidRuleShape, v.ID, "positive validation must have an ID, one Attribute, and no Relation: %+v", v)
			continue
		case rule.AttributeType != decimalType:
			report(CodeUnsupportedType, v.ID, "semantic validation %s: unsupported Attribute type %q (requires resolved std.decimal)", v.ID, rule.AttributeType)
			continue
		}
		names, ok := namePositive(string(v.Attribute))
		if !ok {
			report(CodeNameInvalid, v.ID, "semantic validation %s: attribute %s does not give a Go name; rename it with ASCII letters and digits that start with a letter", v.ID, v.Attribute)
			continue
		}
		if why := reservedName(names.Parameter); why != "" {
			report(CodeNameReserved, v.ID, "semantic validation %s: the parameter name %q from attribute %s is %s; rename the attribute", v.ID, names.Parameter, v.Attribute, why)
			continue
		}
		// Decision and check functions share the package scope, so one
		// rule's decision name ("CheckQuantityIsPositive" from attribute
		// check_quantity) can equal another rule's check name.
		collided := false
		for _, name := range []string{names.Decision, names.Check} {
			if first, taken := owner[name]; taken {
				report(CodeNameCollision, v.ID, "semantic validations %s and %s both give the Go name %s; rename one of the attributes", first, v.ID, name)
				collided = true
				break
			}
		}
		if collided {
			continue
		}
		owner[names.Decision] = v.ID
		owner[names.Check] = v.ID
		accepted = append(accepted, named{rule: rule, names: names})
	}
	if len(diagnostics) > 0 {
		return Output{Diagnostics: diagnostics}
	}

	var out Output
	for _, item := range accepted {
		v := item.rule.Validation
		out.Files = append(out.Files, File{
			Path:    item.names.File,
			Content: positiveFile(input.PackageName, item.rule, item.names),
		})
		out.Trace = append(out.Trace, Trace{
			Element: v.ID, Attribute: v.Attribute, File: item.names.File,
			Decision: item.names.Decision, Check: item.names.Check,
		})
	}
	out.Files = append(out.Files,
		File{Path: "result.go", Content: standardPart("result.go", input.PackageName,
			"// Code generated by TEI; DO NOT EDIT.")},
		File{Path: "decimal.go", Content: standardPart("decimal.go", input.PackageName,
			"// Code generated by TEI from std.decimal; DO NOT EDIT.")},
	)
	sort.Slice(out.Files, func(i, j int) bool { return out.Files[i].Path < out.Files[j].Path })
	return out
}

// standardPart copies one file of internal/stdpart into the target package.
// The part is compiled and tested in place, so a failure here is a defect of
// goemit itself, not of the input.
func standardPart(name, packageName, header string) []byte {
	source, err := standardParts.ReadFile("internal/stdpart/" + name)
	if err != nil {
		panic(fmt.Sprintf("goemit: standard part %s is not embedded: %v", name, err))
	}
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, name, source, parser.ParseComments)
	if err != nil {
		panic(fmt.Sprintf("goemit: standard part %s does not parse: %v", name, err))
	}
	file.Name.Name = packageName
	var body bytes.Buffer
	if err := printer.Fprint(&body, fset, file); err != nil {
		panic(fmt.Sprintf("goemit: print standard part %s: %v", name, err))
	}
	return withHeader(header, body.Bytes())
}

// withHeader puts the generated-code header above the package clause and
// formats the whole file.
func withHeader(header string, body []byte) []byte {
	var b bytes.Buffer
	b.WriteString(header)
	b.WriteString("\n\n")
	b.Write(body)
	formatted, err := format.Source(b.Bytes())
	if err != nil {
		panic(fmt.Sprintf("goemit: emitted Go does not format: %v\n%s", err, b.Bytes()))
	}
	return formatted
}
