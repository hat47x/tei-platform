package languagereference

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"tei/internal/core"
	"tei/internal/semantic"
	"tei/internal/syntax"
)

// This file characterizes a limited Everything-is-Data authoring handoff.
// The JSON "language" is an intentionally test-only adapter. The plain-data
// representation below is NOT a new TEI Canonical, DSL schema or accepted role.
// Both paths exercise the existing Selection, source spans, operator pin and
// typed-lowering reference, rather than a second semantic resolver.

type fixtureJSONPipeline struct {
	Input     string   `json:"input"`
	Operators []string `json:"operators"`
}

func parseFixtureJSONPipeline(source string) (PipelineSyntax, error) {
	var input fixtureJSONPipeline
	dec := json.NewDecoder(strings.NewReader(source))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&input); err != nil {
		return PipelineSyntax{}, err
	}
	if input.Input == "" || len(input.Operators) == 0 {
		return PipelineSyntax{}, fmt.Errorf("missing input or operators")
	}
	// Deliberately small: fixtures use only unescaped ASCII token IDs and
	// unambiguous occurrences. Real languages own general span handling.
	locate := func(token string, from int) (PipelineToken, int, error) {
		quoted := string('"') + token + string('"')
		index := strings.Index(source[from:], quoted)
		if index < 0 {
			return PipelineToken{}, from, fmt.Errorf("token %q not found", token)
		}
		start := from + index + 1
		end := start + len(token)
		return PipelineToken{ID: token, Span: PipelineSpan{Start: start, End: end}}, end + 1, nil
	}
	first, cursor, err := locate(input.Input, 0)
	if err != nil {
		return PipelineSyntax{}, err
	}
	result := PipelineSyntax{Input: first}
	for _, id := range input.Operators {
		token, next, err := locate(id, cursor)
		if err != nil {
			return PipelineSyntax{}, err
		}
		result.Operators = append(result.Operators, token)
		cursor = next
	}
	return result, nil
}

type fixturePlainStep struct {
	OperatorID string `json:"operatorId"`
	Version    string `json:"version"`
	Digest     string `json:"digest"`
	InputType  string `json:"inputType"`
	OutputType string `json:"outputType"`
	Effect     string `json:"effect"`
}

type fixturePlainCandidate struct {
	Role       string             `json:"role"`
	InputRef   string             `json:"inputRef"`
	Steps      []fixturePlainStep `json:"steps"`
	ResultType string             `json:"resultType"`
}

// A read-only, fully qualified candidate for this fixture. No Source bytes,
// Source span, Authorization/Host proof, Acceptance, execution or Result is
// stored in this projection. Those belong to their existing TEI owners.
func projectFixturePlainCandidate(syntax PipelineSyntax, pins, catalog []OperatorContract) (fixturePlainCandidate, error) {
	lowered, err := LowerPinnedPipeline(syntax, pins, catalog)
	if err != nil {
		return fixturePlainCandidate{}, err
	}
	input := syntax.Input.ID
	result := fixturePlainCandidate{Role: "function", InputRef: input}
	current := input
	for _, step := range lowered {
		c := step.Contract
		result.Steps = append(result.Steps, fixturePlainStep{
			OperatorID: c.ID, Version: c.Version, Digest: c.Fingerprint,
			InputType: step.InputType, OutputType: c.OutputType, Effect: c.Effect,
		})
		current = c.OutputType
	}
	result.ResultType = current
	return result, nil
}

func fixtureLanguagePins() (Selection, Selection) {
	return Selection{
		LanguageID: "fixture.pipe", ContractVersion: "1",
		ContractFingerprint: "pipe-1", ImplementationID: "fixture.pipe.impl",
		ImplementationFingerprint: "pipe-source-1",
	}, Selection{
		LanguageID: "fixture.json", ContractVersion: "1",
		ContractFingerprint: "json-1", ImplementationID: "fixture.json.impl",
		ImplementationFingerprint: "json-source-1",
	}
}

func fixtureOperators() (OperatorContract, OperatorContract) {
	return OperatorContract{
		ID: "std.where", Version: "1", Fingerprint: "where-fixture-1",
		Effect: "pure", InputType: "rows/order", OutputType: "rows/order",
	}, OperatorContract{
		ID: "std.sum", Version: "1", Fingerprint: "sum-fixture-1",
		Effect: "pure", InputType: "rows/order", OutputType: "decimal",
	}
}

func TestEverythingDataRealPinnedPipelineCrossSurface(t *testing.T) {
	pipePin, jsonPin := fixtureLanguagePins()
	where, sum := fixtureOperators()
	sourceA := "　rows/order |> std.where |> std.sum"
	sourceB := `{"input":"rows/order","operators":["std.where","std.sum"]}`
	providers := []Provider[PipelineSyntax]{
		{Selection: pipePin, Parse: ParsePipelineSyntax},
		{Selection: jsonPin, Parse: parseFixtureJSONPipeline},
	}
	a, err := ParseSelected(pipePin, providers, sourceA)
	if err != nil {
		t.Fatal(err)
	}
	b, err := ParseSelected(jsonPin, providers, sourceB)
	if err != nil {
		t.Fatal(err)
	}
	for sourceIndex, data := range []struct {
		source string
		value  PipelineSyntax
	}{{sourceA, a}, {sourceB, b}} {
		for _, token := range append([]PipelineToken{data.value.Input}, data.value.Operators...) {
			if token.Span.Start < 0 || token.Span.End > len(data.source) ||
				data.source[token.Span.Start:token.Span.End] != token.ID {
				t.Fatalf("source %d lost byte span: %#v", sourceIndex, token)
			}
		}
	}
	if a.Input.Span.Start != len("　") {
		t.Fatalf("UTF-8 byte span was mistaken for character offset: %v", a.Input.Span)
	}
	pins := []OperatorContract{where, sum}
	x, err := projectFixturePlainCandidate(a, pins, []OperatorContract{sum, where})
	if err != nil {
		t.Fatal(err)
	}
	y, err := projectFixturePlainCandidate(b, pins, []OperatorContract{where, sum})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(x, y) {
		t.Fatalf("two source spellings did not yield equal pinned candidates: %#v vs %#v", x, y)
	}
	if x.ResultType != "decimal" || x.InputRef != "rows/order" {
		t.Fatalf("unexpected typed result: %#v", x)
	}
	payloadA, err := json.Marshal(x)
	if err != nil {
		t.Fatal(err)
	}
	payloadB, err := json.Marshal(y)
	if err != nil {
		t.Fatal(err)
	}
	if string(payloadA) != string(payloadB) {
		t.Fatalf("non-deterministic candidate projection: %s vs %s", payloadA, payloadB)
	}
	if sha256.Sum256([]byte(sourceA)) == sha256.Sum256([]byte(sourceB)) {
		t.Fatal("different Source representations were incorrectly treated as identical")
	}
}

func TestEverythingDataRealPinnedPipelineRejectsVersionDrift(t *testing.T) {
	where, sum := fixtureOperators()
	syntax, err := ParsePipelineSyntax("rows/order |> std.where |> std.sum")
	if err != nil {
		t.Fatal(err)
	}
	v2 := where
	v2.Version = "2"
	v2.Fingerprint = "where-fixture-2"
	_, err = projectFixturePlainCandidate(syntax,
		[]OperatorContract{where, sum}, []OperatorContract{v2, sum})
	if !errors.Is(err, ErrOperatorMissing) {
		t.Fatalf("unavailable accepted v1 became v2: %v", err)
	}
}

func TestEverythingDataRealPinnedPipelineRejectsImpureRoleEscalation(t *testing.T) {
	where, sum := fixtureOperators()
	sum.Effect = "write"
	syntax, err := ParsePipelineSyntax("rows/order |> std.where |> std.sum")
	if err != nil {
		t.Fatal(err)
	}
	_, err = projectFixturePlainCandidate(syntax,
		[]OperatorContract{where, sum}, []OperatorContract{where, sum})
	if !errors.Is(err, ErrEffectEscalation) {
		t.Fatalf("impure operator became executable pure data: %v", err)
	}
}

func TestEverythingDataRealPinnedPipelineRejectsTypeMismatch(t *testing.T) {
	where, sum := fixtureOperators()
	sum.InputType = "rows/customer"
	syntax, err := ParsePipelineSyntax("rows/order |> std.where |> std.sum")
	if err != nil {
		t.Fatal(err)
	}
	_, err = projectFixturePlainCandidate(syntax,
		[]OperatorContract{where, sum}, []OperatorContract{where, sum})
	if !errors.Is(err, ErrOperatorInvalid) {
		t.Fatalf("unconnected data type was accepted: %v", err)
	}
}

func TestEverythingDataRealPinnedPipelineRejectsUnselectedJSONLanguage(t *testing.T) {
	pipePin, jsonPin := fixtureLanguagePins()
	providers := []Provider[PipelineSyntax]{
		{Selection: pipePin, Parse: ParsePipelineSyntax},
	}
	_, err := ParseSelected(jsonPin, providers,
		`{"input":"rows/order","operators":["std.where","std.sum"]}`)
	if !errors.Is(err, ErrProviderMissing) {
		t.Fatalf("missing locked JSON parser silently fell back: %v", err)
	}
}

func TestEverythingDataRealPinnedPipelineRejectsUnsupportedJSONShape(t *testing.T) {
	_, err := parseFixtureJSONPipeline(`{"input":"rows/order","operators":["std.where"],"execute":true}`)
	if err == nil {
		t.Fatal("unexpected execution attribute was accepted by fixture parser")
	}
}

func TestEverythingDataRealPinnedPipelineRejectsInvalidSource(t *testing.T) {
	if _, err := ParsePipelineSyntax("rows/order |>"); err == nil {
		t.Fatal("incomplete authoring source accepted")
	}
}

// EG-08: join two syntax paths to an actual semantic.Namespace and an actual
// pinned operator reference. This checks identity/type handoff only, NOT the
// operator's predicate, cardinality, SQL or any accepted source revision.
func TestEverythingDataActualSemanticParserCrossLanguageBind(t *testing.T) {
	teiPin := testPin()
	_, jsonPin := fixtureLanguagePins()
	syntaxSource := "namespace sales\ndata Order { attribute orderId type std.identifier }"
	jsonSource := `{"id":"sales","data":[{"id":"sales.order","symbol":"Order","attributes":[{"id":"sales.order.order_id","symbol":"orderId","type":"std.identifier"}]}]}`
	providers := []Provider[semantic.Namespace]{
		{Selection: teiPin, Parse: func(source string) (semantic.Namespace, error) {
			return syntax.NewParser(source).ParseSemanticFile()
		}},
		{Selection: jsonPin, Parse: func(source string) (semantic.Namespace, error) {
			var value semantic.Namespace
			err := json.Unmarshal([]byte(source), &value)
			return value, err
		}},
	}
	a, err := ParseSelected(teiPin, providers, syntaxSource)
	if err != nil {
		t.Fatalf("TEI parser: %v", err)
	}
	b, err := ParseSelected(jsonPin, providers, jsonSource)
	if err != nil {
		t.Fatalf("test-only JSON adapter: %v", err)
	}
	for _, ns := range []semantic.Namespace{a, b} {
		if ns.ID != core.CanonicalID("sales") || len(ns.Data) != 1 ||
			ns.Data[0].ID != core.CanonicalID("sales.order") ||
			len(ns.Data[0].Attributes) != 1 ||
			ns.Data[0].Attributes[0].ID != core.CanonicalID("sales.order.order_id") ||
			ns.Data[0].Attributes[0].TypeRef != core.CanonicalID("std.identifier") {
			t.Fatalf("data/attribute identity or type not resolved: %#v", ns)
		}
	}
	where := OperatorContract{
		ID: "std.where", Version: "1", Fingerprint: "where-semantic-fixture",
		Effect: "pure", InputType: "rows/sales.order", OutputType: "rows/sales.order",
	}
	calls := []PinnedCall{{Contract: where, InputType: where.InputType}}
	outA, err := BindPinnedSemanticInput(a, "sales.order", calls, []OperatorContract{where})
	if err != nil {
		t.Fatalf("TEI semantic bind: %v", err)
	}
	outB, err := BindPinnedSemanticInput(b, "sales.order", calls, []OperatorContract{where})
	if err != nil {
		t.Fatalf("test-only JSON semantic bind: %v", err)
	}
	if outA != "rows/sales.order" || outA != outB {
		t.Fatalf("mismatching bounded data/typed pipeline: %q, %q", outA, outB)
	}
}

// EG-09 is deliberately a LIMITATION witness. The existing binder checks
// uniqueness of Data ID and exact operator pins, not full Attribute Type
// semantics or accepted revision. Identical outputs must not be promoted to
// evidence that source/model meaning stayed unchanged.
func TestEverythingDataBoundDataIDCannotProveModelSemanticEquality(t *testing.T) {
	source := semantic.Namespace{
		ID: core.CanonicalID("sales"),
		Data: []semantic.Data{{
			ID: core.CanonicalID("sales.order"),
			Attributes: []semantic.Attribute{{
				ID:      core.CanonicalID("sales.order.amount"),
				TypeRef: core.CanonicalID("std.decimal"),
			}},
		}},
	}
	changed := semantic.Namespace{
		ID: core.CanonicalID("sales"),
		Data: []semantic.Data{{
			ID: core.CanonicalID("sales.order"),
			Attributes: []semantic.Attribute{{
				ID:      core.CanonicalID("sales.order.amount"),
				TypeRef: core.CanonicalID("std.string"),
			}},
		}},
	}
	if reflect.DeepEqual(source, changed) {
		t.Fatal("the pressure worlds must have different semantic attributes")
	}
	where := OperatorContract{
		ID: "std.where", Version: "1", Fingerprint: "where-semantic-fixture",
		Effect: "pure", InputType: "rows/sales.order", OutputType: "rows/sales.order",
	}
	calls := []PinnedCall{{Contract: where, InputType: where.InputType}}
	x, err := BindPinnedSemanticInput(source, "sales.order", calls, []OperatorContract{where})
	if err != nil {
		t.Fatal(err)
	}
	y, err := BindPinnedSemanticInput(changed, "sales.order", calls, []OperatorContract{where})
	if err != nil {
		t.Fatal(err)
	}
	if x != y {
		t.Fatalf("expected binder's limited data-ID result to be unchanged; %q != %q", x, y)
	}
	// Expected observation: x==y despite changed Attribute Type. The missing
	// model/Role/Context/Operator-law qualification is for existing owners.
}
