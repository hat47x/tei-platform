package validationgen_test

import (
	"bytes"
	"fmt"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"tei/internal/codegen/goemit"
	"tei/internal/codegen/goemit/goemittest"
	"tei/internal/core"
	"tei/internal/semantic"
	"tei/internal/semantic/validationeval"
	"tei/internal/workspace"
)

// Both realizations use one selected typed positive Rule and an independent
// expected result table. This is local evaluation conformance, NOT evidence
// of a selected production Provider, runtime authority or a product Artifact.
func TestSelectedPositiveRuleDirectAndGeneratedEvaluation(t *testing.T) {
	if testing.Short() {
		t.Skip("generated Go must be compiled and run")
	}
	goemittest.GoExecutable(t)
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("test source path unavailable")
	}
	root := filepath.Clean(filepath.Join(filepath.Dir(filename), "..", "..", ".."))
	fixture := filepath.Join(root, "internal", "workspace", "testdata", "tei-md-rule")
	loaded, err := workspace.Load(fixture)
	if err != nil {
		t.Fatal(err)
	}
	if len(loaded.AuthoringSources) != 1 ||
		loaded.AuthoringSources[0].Language.LanguageID != "tei.rule-authoring" {
		t.Fatalf("unexpected selected Language Reader: %+v", loaded.AuthoringSources)
	}

	var rule semantic.Validation
	var attributeType core.CanonicalID
	for _, ns := range loaded.World.Semantics {
		for _, candidate := range ns.Validations {
			if candidate.ID == "sales.order_line.quantity_positive" {
				if rule.ID != "" {
					t.Fatal("duplicate selected Rule")
				}
				rule = candidate
			}
		}
	}
	for _, ns := range loaded.World.Semantics {
		for _, data := range ns.Data {
			for _, attribute := range data.Attributes {
				if attribute.ID == rule.Attribute {
					attributeType = attribute.TypeRef
				}
			}
		}
	}
	if rule.ID != "sales.order_line.quantity_positive" ||
		rule.Kind != semantic.ValidationPositive ||
		rule.Attribute != "sales.order_line.quantity" ||
		rule.Relation != "" || attributeType != "std.decimal" {
		t.Fatalf("unsupported or unresolved selected Rule: %+v, type=%s", rule, attributeType)
	}

	out := goemit.Emit(goemit.Input{PackageName: "main", Rules: []goemit.Rule{{Validation: rule, AttributeType: attributeType}}})
	if err := out.Err(); err != nil {
		t.Fatal(err)
	}
	if generated := out.File("quantity_is_positive.go"); !bytes.Contains(generated, []byte("return quantity.Sign() > 0")) ||
		bytes.Contains(generated, []byte("validationeval")) {
		t.Fatal("generated Go does not own the positive decision")
	}

	// Each realization is checked against fixed expectations, not against
	// the other realization's output. Runtime evaluation is limited to the
	// Shape that goemit accepts. A null value and a value that was not
	// fetched are both undecided: Decide says so with its reason, and the
	// generated check function, which receives nil for both, says Undecided.
	cases := []struct {
		name, value, want string
		state             validationeval.ValueState
	}{
		{"positive", "2.5", "satisfied", validationeval.ValuePresent},
		{"zero", "0", "not-satisfied", validationeval.ValuePresent},
		{"negative", "-1", "not-satisfied", validationeval.ValuePresent},
		{"null", "", "undecided", validationeval.ValueNull},
		{"unavailable", "", "undecided", validationeval.ValueNotFetched},
	}

	// Direct: the existing evaluator, in this process.
	for _, c := range cases {
		source := &directNumber{reading: validationeval.NumberReading{State: c.state}}
		if c.state == validationeval.ValuePresent {
			value, err := semantic.ParseOperatorCaseValue(c.value)
			if err != nil {
				t.Fatal(err)
			}
			source.reading.Value = value
		}
		decision, err := validationeval.Decide(rule, source, validationeval.Operators{})
		if err != nil {
			t.Fatalf("direct/%s: %v", c.name, err)
		}
		got := string(decision.Result)
		if decision.Undecided != nil {
			got = "undecided"
			if decision.Undecided.Attribute != rule.Attribute {
				t.Fatalf("direct/%s: undecided on %s", c.name, decision.Undecided.Attribute)
			}
		}
		if got != c.want || source.reads != 1 || source.requested != rule.Attribute {
			t.Fatalf("direct/%s: got %s, reads=%d %s; want %s", c.name, got, source.reads, source.requested, c.want)
		}
	}
	if _, err := validationeval.Decide(rule, nil, validationeval.Operators{}); err == nil {
		t.Fatal("direct: nil source accepted")
	}

	// Generated: the emitted package in a module of its own. The harness
	// reads the attribute the Trace binds to the check function.
	var harness []goemittest.Case
	var want strings.Builder
	for _, c := range cases {
		row := map[string]goemittest.Cell{}
		switch c.state {
		case validationeval.ValuePresent:
			row[string(rule.Attribute)] = goemittest.Cell{Text: c.value}
		case validationeval.ValueNull:
			row[string(rule.Attribute)] = goemittest.Cell{Null: true}
		}
		harness = append(harness, goemittest.Case{Name: c.name, Row: row})
		fmt.Fprintf(&want, "%s:%s:1:%s\n", c.name, c.want, rule.Attribute)
	}
	if got := goemittest.Run(t, out, harness); got != want.String() {
		t.Fatalf("generated evaluation:\n%s\nwant:\n%s", got, want.String())
	}
}

// directNumber reports one fixed reading and counts the reads.
type directNumber struct {
	reading   validationeval.NumberReading
	reads     int
	requested core.CanonicalID
}

func (s *directNumber) ReadNumber(attribute core.CanonicalID) (validationeval.NumberReading, error) {
	s.reads++
	s.requested = attribute
	return s.reading, nil
}

func (s *directNumber) ReadRelationKeys(core.CanonicalID, core.CanonicalID) (validationeval.RelationReading, error) {
	return validationeval.RelationReading{}, fmt.Errorf("unexpected relation read")
}
