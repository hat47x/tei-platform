package transform

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
	"unicode/utf8"
)

type EvaluationResult struct {
	Output Value
	Steps  int
}

type evaluator struct {
	limits Limits
	steps  int
}

func Evaluate(source Source, inputs map[string]Value) (EvaluationResult, error) {
	if err := source.Validate(); err != nil {
		return EvaluationResult{}, err
	}
	if err := validateInputs(source.Inputs, inputs); err != nil {
		return EvaluationResult{}, err
	}
	if err := validateInputBudgets(source.Limits, inputs); err != nil {
		return EvaluationResult{}, err
	}
	e := evaluator{limits: source.Limits}
	env := make(map[string]Value, len(inputs))
	for name, value := range inputs {
		// Isolate caller-owned slices once. Evaluation treats Value as immutable
		// and may then share internal subtrees without repeated deep copies.
		env[name] = value.Clone()
	}
	for i := range source.Program.Assertions {
		if _, err := e.eval(source.Program.Assertions[i], env, 0); err != nil {
			return EvaluationResult{}, fmt.Errorf("assertion %d: %w", i, err)
		}
	}
	output, err := e.eval(source.Program.Output, env, 0)
	if err != nil {
		return EvaluationResult{}, err
	}
	if _, err := encodedOutputSize(output, source.Limits.MaxOutputBytes); err != nil {
		return EvaluationResult{}, err
	}
	return EvaluationResult{Output: output, Steps: e.steps}, nil
}

func validateInputs(specs []InputSpec, inputs map[string]Value) error {
	expected := map[string]InputKind{}
	expectedNames := make([]string, 0, len(specs))
	for _, spec := range specs {
		expected[spec.Name] = spec.Kind
		expectedNames = append(expectedNames, spec.Name)
	}
	sort.Strings(expectedNames)

	actualNames := make([]string, 0, len(inputs))
	for name := range inputs {
		actualNames = append(actualNames, name)
	}
	sort.Strings(actualNames)
	for _, name := range actualNames {
		if _, ok := expected[name]; !ok {
			return fmt.Errorf("undeclared transform input %q", name)
		}
	}
	for _, name := range expectedNames {
		kind := expected[name]
		value, ok := inputs[name]
		if !ok {
			return fmt.Errorf("required transform input %q is missing", name)
		}
		if !inputKindMatches(kind, value.Kind) {
			return fmt.Errorf("transform input %q requires %s, got %s", name, kind, value.Kind)
		}
	}
	return nil
}

func inputKindMatches(expected InputKind, actual ValueKind) bool {
	switch expected {
	case InputObject:
		return actual == ValueObject
	case InputArray:
		return actual == ValueArray
	case InputString:
		return actual == ValueString
	case InputBytes:
		return actual == ValueBytes
	default:
		return false
	}
}

func (e *evaluator) consume(count int) error {
	if count < 0 || count > e.limits.MaxSteps-e.steps {
		return fmt.Errorf("transform exceeded maxSteps")
	}
	e.steps += count
	return nil
}

func (e *evaluator) tick(depth int) error {
	if depth > e.limits.MaxNestingDepth {
		return fmt.Errorf("transform exceeded maxNestingDepth")
	}
	return e.consume(1)
}

func (e *evaluator) eval(expr Expr, env map[string]Value, depth int) (Value, error) {
	if err := e.tick(depth); err != nil {
		return Value{}, err
	}
	switch expr.Op {
	case OpInput:
		value, ok := env[expr.Name]
		if !ok {
			return Value{}, fmt.Errorf("transform input or binding %q is not available", expr.Name)
		}
		return value, nil
	case OpLiteral:
		return ParseJSON(expr.Literal)
	case OpSelectField:
		source, err := e.eval(*expr.From, env, depth+1)
		if err != nil {
			return Value{}, err
		}
		if source.Kind != ValueObject {
			return Value{}, fmt.Errorf("select-field requires object, got %s", source.Kind)
		}
		value, ok := fieldShared(source, expr.Field)
		if !ok {
			if expr.Optional {
				return Null(), nil
			}
			return Value{}, fmt.Errorf("required field %q is missing", expr.Field)
		}
		return value, nil
	case OpConstructObject:
		fields := make([]Field, 0, len(expr.Fields))
		for _, field := range expr.Fields {
			value, err := e.eval(field.Value, env, depth+1)
			if err != nil {
				return Value{}, fmt.Errorf("construct field %q: %w", field.Name, err)
			}
			fields = append(fields, Field{Name: field.Name, Value: value})
		}
		return objectShared(fields...)
	case OpMapArray:
		source, err := e.eval(*expr.From, env, depth+1)
		if err != nil {
			return Value{}, err
		}
		if source.Kind != ValueArray {
			return Value{}, fmt.Errorf("map-array requires array, got %s", source.Kind)
		}
		if len(source.Array) > e.limits.MaxCollection {
			return Value{}, fmt.Errorf("map-array collection exceeds maxCollection")
		}
		out := make([]Value, len(source.Array))
		for i, item := range source.Array {
			local := cloneEnv(env)
			local[expr.Item] = item
			value, err := e.eval(*expr.Body, local, depth+1)
			if err != nil {
				return Value{}, fmt.Errorf("map-array item %d: %w", i, err)
			}
			out[i] = value
		}
		return arrayShared(out), nil
	case OpSortArrayBy:
		source, err := e.eval(*expr.From, env, depth+1)
		if err != nil {
			return Value{}, err
		}
		if source.Kind != ValueArray {
			return Value{}, fmt.Errorf("sort-array requires array, got %s", source.Kind)
		}
		if len(source.Array) > e.limits.MaxCollection {
			return Value{}, fmt.Errorf("sort-array collection exceeds maxCollection")
		}
		type keyed struct {
			value Value
			keys  []string
		}
		if err := e.consume(sortWorkBudget(len(source.Array))); err != nil {
			return Value{}, err
		}
		items := make([]keyed, len(source.Array))
		for i, item := range source.Array {
			local := cloneEnv(env)
			local[expr.Item] = item
			keys := make([]string, len(expr.Keys))
			for k := range expr.Keys {
				key, err := e.eval(expr.Keys[k], local, depth+1)
				if err != nil {
					return Value{}, fmt.Errorf("sort-array item %d key %d: %w", i, k, err)
				}
				if key.Kind != ValueString {
					return Value{}, fmt.Errorf("sort-array keys must be strings, got %s", key.Kind)
				}
				keys[k] = key.String
			}
			items[i] = keyed{value: item, keys: keys}
		}
		sort.Slice(items, func(i, j int) bool {
			for k := range items[i].keys {
				if items[i].keys[k] == items[j].keys[k] {
					continue
				}
				return items[i].keys[k] < items[j].keys[k]
			}
			return false
		})
		out := make([]Value, len(items))
		for i := range items {
			out[i] = items[i].value
		}
		return arrayShared(out), nil
	case OpReplaceField:
		source, err := e.eval(*expr.From, env, depth+1)
		if err != nil {
			return Value{}, err
		}
		if source.Kind != ValueObject {
			return Value{}, fmt.Errorf("replace-field requires object, got %s", source.Kind)
		}
		if _, exists := fieldShared(source, expr.Field); !exists {
			return source, nil
		}
		replacement, err := e.eval(*expr.Value, env, depth+1)
		if err != nil {
			return Value{}, err
		}
		out, _, err := replaceFieldShared(source, expr.Field, replacement)
		return out, err
	case OpOmitField:
		source, err := e.eval(*expr.From, env, depth+1)
		if err != nil {
			return Value{}, err
		}
		return omitFieldShared(source, expr.Field)
	case OpAssertEqual:
		actual, err := e.eval(*expr.Actual, env, depth+1)
		if err != nil {
			return Value{}, err
		}
		expected, err := e.eval(*expr.Expected, env, depth+1)
		if err != nil {
			return Value{}, err
		}
		if !Equal(actual, expected) {
			return Value{}, fmt.Errorf("assert-equal failed")
		}
		return actual, nil
	case OpAssertEnum:
		actual, err := e.eval(*expr.Actual, env, depth+1)
		if err != nil {
			return Value{}, err
		}
		for _, raw := range expr.Allowed {
			allowed, err := ParseJSON(raw)
			if err != nil {
				return Value{}, err
			}
			if Equal(actual, allowed) {
				return actual, nil
			}
		}
		return Value{}, fmt.Errorf("assert-enum failed")
	case OpCanonicalJSON:
		source, err := e.eval(*expr.From, env, depth+1)
		if err != nil {
			return Value{}, err
		}
		raw, err := source.JSONWithinLimit(e.limits.MaxOutputBytes)
		if err != nil {
			return Value{}, fmt.Errorf("canonical-json: %w", err)
		}
		return bytesShared(raw), nil
	case OpSHA256:
		source, err := e.eval(*expr.From, env, depth+1)
		if err != nil {
			return Value{}, err
		}
		if source.Kind != ValueBytes {
			return Value{}, fmt.Errorf("sha256 requires bytes, got %s", source.Kind)
		}
		sum := sha256.Sum256(source.Bytes)
		return String(hex.EncodeToString(sum[:])), nil
	case OpAppendBytes:
		source, err := e.eval(*expr.From, env, depth+1)
		if err != nil {
			return Value{}, err
		}
		if source.Kind != ValueBytes {
			return Value{}, fmt.Errorf("append-bytes requires bytes, got %s", source.Kind)
		}
		if len(source.Bytes) > e.limits.MaxOutputBytes || len(expr.Suffix) > e.limits.MaxOutputBytes-len(source.Bytes) {
			return Value{}, fmt.Errorf("append-bytes exceeds maxOutputBytes")
		}
		out := make([]byte, 0, len(source.Bytes)+len(expr.Suffix))
		out = append(out, source.Bytes...)
		out = append(out, expr.Suffix...)
		return bytesShared(out), nil
	case OpJoinStrings:
		parts := make([]string, len(expr.Values))
		total := 0
		for i := range expr.Values {
			value, err := e.eval(expr.Values[i], env, depth+1)
			if err != nil {
				return Value{}, err
			}
			if value.Kind != ValueString {
				return Value{}, fmt.Errorf("join-strings requires strings, got %s", value.Kind)
			}
			if len(value.String) > e.limits.MaxOutputBytes-total {
				return Value{}, fmt.Errorf("join-strings exceeds maxOutputBytes")
			}
			total += len(value.String)
			if i > 0 {
				if len(expr.Separator) > e.limits.MaxOutputBytes-total {
					return Value{}, fmt.Errorf("join-strings exceeds maxOutputBytes")
				}
				total += len(expr.Separator)
			}
			parts[i] = value.String
		}
		return String(strings.Join(parts, expr.Separator)), nil
	default:
		return Value{}, fmt.Errorf("unsupported transform operation %q", expr.Op)
	}
}



func encodedOutputSize(value Value, maxBytes int) (int, error) {
	if value.Kind == ValueBytes {
		if len(value.Bytes) > maxBytes {
			return 0, fmt.Errorf("transform output exceeds maxOutputBytes")
		}
		return len(value.Bytes), nil
	}
	raw, err := value.JSONWithinLimit(maxBytes)
	if err != nil {
		return 0, fmt.Errorf("encode transform output: %w", err)
	}
	return len(raw), nil
}

func sortWorkBudget(n int) int {
	if n <= 1 {
		return 0
	}
	levels := 0
	for size := 1; size < n; size <<= 1 {
		levels++
	}
	return n * (levels + 1)
}


func validateValueShape(value Value) error {
	hasArray := len(value.Array) != 0
	hasObject := len(value.Object) != 0
	hasBytes := len(value.Bytes) != 0
	switch value.Kind {
	case ValueNull:
		if value.Bool || value.Number != "" || value.String != "" || hasArray || hasObject || hasBytes {
			return fmt.Errorf("null value contains fields for another value kind")
		}
	case ValueBool:
		if value.Number != "" || value.String != "" || hasArray || hasObject || hasBytes {
			return fmt.Errorf("bool value contains fields for another value kind")
		}
	case ValueNumber:
		if value.Bool || value.String != "" || hasArray || hasObject || hasBytes {
			return fmt.Errorf("number value contains fields for another value kind")
		}
		if !validJSONNumber(value.Number) {
			return fmt.Errorf("invalid JSON number %q", value.Number)
		}
	case ValueString:
		if value.Bool || value.Number != "" || hasArray || hasObject || hasBytes {
			return fmt.Errorf("string value contains fields for another value kind")
		}
		if !utf8.ValidString(value.String) {
			return fmt.Errorf("string value is not valid UTF-8")
		}
	case ValueArray:
		if value.Bool || value.Number != "" || value.String != "" || hasObject || hasBytes {
			return fmt.Errorf("array value contains fields for another value kind")
		}
	case ValueObject:
		if value.Bool || value.Number != "" || value.String != "" || hasArray || hasBytes {
			return fmt.Errorf("object value contains fields for another value kind")
		}
		seen := map[string]struct{}{}
		for _, field := range value.Object {
			if field.Name == "" {
				return fmt.Errorf("object field name is required")
			}
			if !utf8.ValidString(field.Name) {
				return fmt.Errorf("object field name is not valid UTF-8")
			}
			if _, exists := seen[field.Name]; exists {
				return fmt.Errorf("object field %q is declared more than once", field.Name)
			}
			seen[field.Name] = struct{}{}
		}
	case ValueBytes:
		if value.Bool || value.Number != "" || value.String != "" || hasArray || hasObject {
			return fmt.Errorf("bytes value contains fields for another value kind")
		}
	default:
		return fmt.Errorf("unsupported input value kind %q", value.Kind)
	}
	return nil
}

type inputBudget struct {
	nodes int
	bytes int
}

func validateInputBudgets(limits Limits, inputs map[string]Value) error {
	budget := inputBudget{}
	names := make([]string, 0, len(inputs))
	for name := range inputs {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		value := inputs[name]
		budget.bytes += len(name)
		if budget.bytes > limits.MaxInputBytes {
			return fmt.Errorf("transform inputs exceed maxInputBytes")
		}
		if err := walkInputBudget(value, 0, &budget, limits); err != nil {
			return fmt.Errorf("transform input %q: %w", name, err)
		}
	}
	return nil
}

func walkInputBudget(value Value, depth int, budget *inputBudget, limits Limits) error {
	if err := validateValueShape(value); err != nil {
		return err
	}
	if depth > limits.MaxNestingDepth {
		return fmt.Errorf("input exceeds maxNestingDepth")
	}
	budget.nodes++
	if budget.nodes > limits.MaxInputNodes {
		return fmt.Errorf("inputs exceed maxInputNodes")
	}
	switch value.Kind {
	case ValueNull, ValueBool:
	case ValueNumber:
		budget.bytes += len(value.Number)
	case ValueString:
		budget.bytes += len(value.String)
	case ValueBytes:
		budget.bytes += len(value.Bytes)
	case ValueArray:
		if len(value.Array) > limits.MaxCollection {
			return fmt.Errorf("array exceeds maxCollection")
		}
		for _, item := range value.Array {
			if err := walkInputBudget(item, depth+1, budget, limits); err != nil {
				return err
			}
		}
	case ValueObject:
		if len(value.Object) > limits.MaxCollection {
			return fmt.Errorf("object exceeds maxCollection")
		}
		for _, field := range value.Object {
			budget.bytes += len(field.Name)
			if budget.bytes > limits.MaxInputBytes {
				return fmt.Errorf("inputs exceed maxInputBytes")
			}
			if err := walkInputBudget(field.Value, depth+1, budget, limits); err != nil {
				return err
			}
		}
	default:
		return fmt.Errorf("unsupported input value kind %q", value.Kind)
	}
	if budget.bytes > limits.MaxInputBytes {
		return fmt.Errorf("inputs exceed maxInputBytes")
	}
	return nil
}



func bytesShared(raw []byte) Value {
	return Value{Kind: ValueBytes, Bytes: raw}
}

func objectShared(fields []Field) (Value, error) {
	seen := map[string]struct{}{}
	out := Value{Kind: ValueObject, Object: append([]Field(nil), fields...)}
	for _, field := range out.Object {
		if field.Name == "" {
			return Value{}, fmt.Errorf("object field name is required")
		}
		if _, exists := seen[field.Name]; exists {
			return Value{}, fmt.Errorf("object field %q is declared more than once", field.Name)
		}
		seen[field.Name] = struct{}{}
	}
	return out, nil
}

func fieldShared(value Value, name string) (Value, bool) {
	if value.Kind != ValueObject {
		return Value{}, false
	}
	for _, field := range value.Object {
		if field.Name == name {
			return field.Value, true
		}
	}
	return Value{}, false
}

func replaceFieldShared(value Value, name string, replacement Value) (Value, bool, error) {
	if value.Kind != ValueObject {
		return Value{}, false, fmt.Errorf("replace-field requires object, got %s", value.Kind)
	}
	out := value
	out.Object = append([]Field(nil), value.Object...)
	for i := range out.Object {
		if out.Object[i].Name == name {
			out.Object[i].Value = replacement
			return out, true, nil
		}
	}
	return out, false, nil
}

func omitFieldShared(value Value, name string) (Value, error) {
	if value.Kind != ValueObject {
		return Value{}, fmt.Errorf("omit-field requires object, got %s", value.Kind)
	}
	out := Value{Kind: ValueObject, Object: make([]Field, 0, len(value.Object))}
	for _, field := range value.Object {
		if field.Name == name {
			continue
		}
		out.Object = append(out.Object, field)
	}
	return out, nil
}

func cloneEnv(in map[string]Value) map[string]Value {
	out := make(map[string]Value, len(in)+1)
	for name, value := range in {
		out[name] = value
	}
	return out
}
