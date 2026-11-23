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
	"unicode"
	"unicode/utf8"

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
	// PackageName is the package clause of every emitted file. "_" is
	// refused because a package of that name cannot be built. "main" is
	// accepted, but Emit writes no main function: the caller supplies it in
	// a file of its own.
	PackageName string
	Rules       []Rule
}

// File is one emitted Go file.
type File struct {
	Path    string
	Content []byte
}

// Trace ties one rule to the Go it produced. It is the machine-readable
// record of what each check function takes: Check receives the value of
// Attribute through its one parameter, named Parameter. Doc comments say the
// same for people and are not meant to be parsed.
type Trace struct {
	Element   core.CanonicalID
	Attribute core.CanonicalID
	File      string
	Decision  string
	Check     string
	Parameter string
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
	switch {
	case !token.IsIdentifier(input.PackageName):
		report(CodeInvalidPackageName, "", "invalid Go package name %q", input.PackageName)
	case input.PackageName == "_":
		report(CodeInvalidPackageName, "", "invalid Go package name %q: a package named _ cannot be built", input.PackageName)
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
		case !commentText(string(v.ID)) || !commentText(string(v.Attribute)) || !commentText(rule.DisplayName):
			report(CodeInvalidRuleShape, v.ID, "semantic validation %q: the ID, Attribute or display name holds a line break, another control character or invalid UTF-8, so it cannot be written into a Go comment", v.ID)
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
		content, err := positiveFile(input.PackageName, item.rule, item.names)
		if err != nil {
			report(CodeInvalidRuleShape, v.ID, "semantic validation %q: the emitted Go does not format: %v", v.ID, err)
			continue
		}
		out.Files = append(out.Files, File{Path: item.names.File, Content: content})
		out.Trace = append(out.Trace, Trace{
			Element: v.ID, Attribute: v.Attribute, File: item.names.File,
			Decision: item.names.Decision, Check: item.names.Check,
			Parameter: item.names.Parameter,
		})
	}
	for _, part := range []struct{ name, header string }{
		{"result.go", "// Code generated by TEI; DO NOT EDIT."},
		{"decimal.go", "// Code generated by TEI from std.decimal; DO NOT EDIT."},
	} {
		content, err := standardPart(part.name, input.PackageName, part.header)
		if err != nil {
			report(CodeInvalidPackageName, "", "standard part %s does not format in package %q: %v", part.name, input.PackageName, err)
			continue
		}
		out.Files = append(out.Files, File{Path: part.name, Content: content})
	}
	if len(diagnostics) > 0 {
		return Output{Diagnostics: diagnostics}
	}
	sort.Slice(out.Files, func(i, j int) bool { return out.Files[i].Path < out.Files[j].Path })
	return out
}

// commentText reports whether text can be written into a "//" comment
// without ending it or breaking the file: valid UTF-8 with no control
// character (CR and LF among them), byte order mark or line separator.
func commentText(text string) bool {
	if !utf8.ValidString(text) {
		return false
	}
	for _, r := range text {
		if unicode.IsControl(r) || r == '\uFEFF' || r == '\u2028' || r == '\u2029' {
			return false
		}
	}
	return true
}

// standardPart copies one file of internal/stdpart into the target package.
// The part is compiled and tested in place, so a missing or unparsable part
// is a defect of goemit itself and panics. Printing and formatting depend on
// the package name and return an error instead.
func standardPart(name, packageName, header string) ([]byte, error) {
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
		return nil, err
	}
	return withHeader(header, body.Bytes())
}

// withHeader puts the generated-code header above the package clause and
// formats the whole file.
func withHeader(header string, body []byte) ([]byte, error) {
	var b bytes.Buffer
	b.WriteString(header)
	b.WriteString("\n\n")
	b.Write(body)
	return format.Source(b.Bytes())
}
