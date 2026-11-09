// Package formatversion defines the single conversion boundary for definition
// formats (#1920).
//
// The Owner decision is that versions convert through one central
// representation: every version decodes to Central, and Central encodes to the
// target version. Plugins provide the per-version codec; the core owns the
// entry point, the result classification and the invariants:
//
//   - converting to the same version is an identity (idempotent)
//   - no attribute may disappear without an element-scoped diagnostic
//   - a conversion that changes only the version value fails
//   - an unknown version fails as a unit of work, naming the supported versions
//   - a declared version that disagrees with the requested source version fails
//
// Diagnostics produced by a codec must name the element they concern.
package formatversion

import (
	"fmt"
	"reflect"
	"sort"
	"strings"
)

// Version is a definition format generation, not a content contract version.
type Version int

// AttributeState is the three-valued presence of an attribute value.
type AttributeState int

const (
	// Absent means the attribute has no value clause.
	Absent AttributeState = iota
	// Empty means the attribute has an explicitly empty value.
	Empty
	// Value means the attribute has a non-empty value.
	Value
)

// String renders the state for diagnostics.
func (s AttributeState) String() string {
	switch s {
	case Absent:
		return "absent"
	case Empty:
		return "empty"
	case Value:
		return "value"
	default:
		return fmt.Sprintf("unknown(%d)", int(s))
	}
}

// Attribute is one attribute in the central representation.
type Attribute struct {
	Name  string
	State AttributeState
	Value string
}

// Element is one addressable definition element in the central representation.
type Element struct {
	ID         string
	Kind       string
	Attributes []Attribute
}

// Document is the central representation. Declared is the version found in the
// source; zero means the source did not declare one and inherits its container.
type Document struct {
	Declared Version
	Elements []Element
}

// Severity separates warnings from failures. Both are element-scoped when a
// codec produces them.
type Severity int

const (
	SeverityWarning Severity = iota
	SeverityFailure
)

// Diagnostic is an element-scoped conversion message.
type Diagnostic struct {
	Severity  Severity
	Code      string
	ElementID string
	Attribute string
	Message   string
}

// Status classifies one conversion.
type Status int

const (
	Success Status = iota
	SuccessWithWarnings
	Failure
)

// Result is the outcome of one conversion.
type Result struct {
	Status      Status
	From        Version
	To          Version
	Source      []byte
	Output      []byte
	Document    Document
	Diagnostics []Diagnostic
	// Reason is set for system-level failures that are not tied to an element.
	Reason string
}

// OK reports whether the conversion succeeded, with or without warnings.
func (r Result) OK() bool { return r.Status != Failure }

// Codec is the per-version plugin boundary. Decode must fill Document.Declared
// when the source declares a version. Encode must write the target version.
type Codec interface {
	Version() Version
	Decode(source []byte) (Document, []Diagnostic, error)
	Encode(document Document) ([]byte, []Diagnostic, error)
}

// Registry holds one codec per version and is the single conversion entry.
type Registry struct {
	codecs map[Version]Codec
}

// NewRegistry returns an empty registry.
func NewRegistry() *Registry {
	return &Registry{codecs: map[Version]Codec{}}
}

// Register adds a codec. A duplicate version is rejected so two plugins cannot
// disagree about the same generation.
func (r *Registry) Register(codec Codec) error {
	if codec == nil {
		return fmt.Errorf("formatversion: nil codec")
	}
	version := codec.Version()
	if version < 1 {
		return fmt.Errorf("formatversion: codec version must be positive, got %d", version)
	}
	if _, exists := r.codecs[version]; exists {
		return fmt.Errorf("formatversion: codec for version %d is already registered", version)
	}
	r.codecs[version] = codec
	return nil
}

// Supported lists the registered versions in ascending order.
func (r *Registry) Supported() []Version {
	out := make([]Version, 0, len(r.codecs))
	for version := range r.codecs {
		out = append(out, version)
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

func (r *Registry) supportedText() string {
	versions := r.Supported()
	parts := make([]string, 0, len(versions))
	for _, version := range versions {
		parts = append(parts, fmt.Sprint(int(version)))
	}
	if len(parts) == 0 {
		return "none"
	}
	return strings.Join(parts, ", ")
}

// Convert is the single conversion entry point.
func (r *Registry) Convert(source []byte, from, to Version) (Result, error) {
	result := Result{From: from, To: to, Source: append([]byte(nil), source...)}
	fromCodec, ok := r.codecs[from]
	if !ok {
		return systemFailure(result, "UNKNOWN_SOURCE_VERSION",
			fmt.Sprintf("no codec for version %d; supported versions: %s", from, r.supportedText()))
	}
	document, decodeDiagnostics, err := fromCodec.Decode(source)
	if err != nil {
		return systemFailure(result, "DECODE_FAILED", err.Error())
	}
	if err := validateCodecDiagnostics(decodeDiagnostics); err != nil {
		return Result{}, err
	}
	if hasFailure(decodeDiagnostics) {
		result.Status = Failure
		result.Document = document
		result.Output = append([]byte(nil), source...)
		result.Diagnostics = decodeDiagnostics
		return result, nil
	}
	if document.Declared != 0 && document.Declared != from {
		return systemFailure(result, "VERSION_MISMATCH",
			fmt.Sprintf("source declares format version %d but was converted from %d", document.Declared, from))
	}
	if from == to {
		// Identity: the same version must not be rewritten.
		result.Status = statusOf(decodeDiagnostics)
		result.Document = document
		result.Output = append([]byte(nil), source...)
		result.Diagnostics = decodeDiagnostics
		return result, nil
	}
	toCodec, ok := r.codecs[to]
	if !ok {
		return systemFailure(result, "UNKNOWN_TARGET_VERSION",
			fmt.Sprintf("no codec for version %d; supported versions: %s", to, r.supportedText()))
	}
	encoded, encodeDiagnostics, err := toCodec.Encode(document)
	if err != nil {
		return systemFailure(result, "ENCODE_FAILED", err.Error())
	}
	if err := validateCodecDiagnostics(encodeDiagnostics); err != nil {
		return Result{}, err
	}
	if hasFailure(encodeDiagnostics) {
		result.Status = Failure
		result.Document = document
		result.Output = encoded
		result.Diagnostics = append(append([]Diagnostic{}, decodeDiagnostics...), encodeDiagnostics...)
		return result, nil
	}
	diagnostics := append(append([]Diagnostic{}, decodeDiagnostics...), encodeDiagnostics...)
	redecoded, _, err := toCodec.Decode(encoded)
	if err != nil {
		return systemFailure(result, "ENCODE_UNREADABLE",
			fmt.Sprintf("encoded version %d cannot be decoded again: %v", to, err))
	}
	if lost := missingAttributes(document, redecoded, diagnostics); len(lost) > 0 {
		result.Status = Failure
		result.Document = redecoded
		result.Output = encoded
		result.Diagnostics = append(diagnostics, lost...)
		return result, nil
	}
	if redecoded.Declared != 0 && redecoded.Declared != to {
		return systemFailure(result, "VERSION_MISMATCH",
			fmt.Sprintf("encoded output declares format version %d but was written as %d", redecoded.Declared, to))
	}
	if !hasWarnings(diagnostics) && sameElements(document.Elements, redecoded.Elements) {
		return systemFailure(result, "VERSION_ONLY_BUMP",
			fmt.Sprintf("conversion from %d to %d changed no content; version values may only advance with a real conversion", from, to))
	}
	result.Status = statusOf(diagnostics)
	result.Document = redecoded
	result.Output = encoded
	result.Diagnostics = diagnostics
	return result, nil
}

func systemFailure(result Result, code, message string) (Result, error) {
	result.Status = Failure
	result.Reason = message
	result.Diagnostics = []Diagnostic{{Severity: SeverityFailure, Code: code, Message: message}}
	return result, nil
}

func validateCodecDiagnostics(diagnostics []Diagnostic) error {
	for _, diagnostic := range diagnostics {
		if diagnostic.ElementID == "" {
			return fmt.Errorf("formatversion: codec diagnostic %q has no element id", diagnostic.Code)
		}
	}
	return nil
}

func statusOf(diagnostics []Diagnostic) Status {
	status := Success
	for _, diagnostic := range diagnostics {
		switch diagnostic.Severity {
		case SeverityFailure:
			return Failure
		case SeverityWarning:
			status = SuccessWithWarnings
		}
	}
	return status
}

func hasWarnings(diagnostics []Diagnostic) bool {
	for _, diagnostic := range diagnostics {
		if diagnostic.Severity == SeverityWarning {
			return true
		}
	}
	return false
}

func hasFailure(diagnostics []Diagnostic) bool {
	for _, diagnostic := range diagnostics {
		if diagnostic.Severity == SeverityFailure {
			return true
		}
	}
	return false
}

// missingAttributes returns a failure for every attribute that is not present in
// the converted document and is not covered by a diagnostic for the same
// element and attribute.
func missingAttributes(before, after Document, diagnostics []Diagnostic) []Diagnostic {
	reported := map[string]bool{}
	for _, diagnostic := range diagnostics {
		if diagnostic.ElementID == "" || diagnostic.Attribute == "" {
			continue
		}
		reported[attributeKey(diagnostic.ElementID, diagnostic.Attribute)] = true
	}
	present := map[string]int{}
	for _, element := range after.Elements {
		for _, attribute := range element.Attributes {
			present[attributeKey(element.ID, attribute.Name)]++
		}
	}
	var lost []Diagnostic
	for _, element := range before.Elements {
		for _, attribute := range element.Attributes {
			key := attributeKey(element.ID, attribute.Name)
			if present[key] > 0 {
				present[key]--
				continue
			}
			if reported[key] {
				continue
			}
			lost = append(lost, Diagnostic{
				Severity:  SeverityFailure,
				Code:      "ATTRIBUTE_LOST",
				ElementID: element.ID,
				Attribute: attribute.Name,
				Message:   fmt.Sprintf("attribute %q of %q (%s %s) disappeared without a diagnostic", attribute.Name, element.ID, attribute.State, attribute.Value),
			})
		}
	}
	return lost
}

func attributeKey(elementID, attribute string) string {
	return elementID + "\x00" + attribute
}

func sameElements(left, right []Element) bool {
	return reflect.DeepEqual(normalizeElements(left), normalizeElements(right))
}

func normalizeElements(elements []Element) []Element {
	out := make([]Element, len(elements))
	copy(out, elements)
	for i := range out {
		sort.Slice(out[i].Attributes, func(a, b int) bool { return out[i].Attributes[a].Name < out[i].Attributes[b].Name })
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}
