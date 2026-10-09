package transform

import (
	"bytes"
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
	rootValues := make(map[string]Value, len(inputs))
	for name, value := range inputs {
		// Isolate caller-owned slices once. Evaluation treats Value as immutable
		// and may then share internal subtrees without repeated deep copies.
		rootValues[name] = value.Clone()
	}
	env := &evalEnv{root: rootValues}
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

func (e *evaluator) eval(expr Expr, env *evalEnv, depth int) (Value, error) {
	if err := e.tick(depth); err != nil {
		return Value{}, err
	}
	switch expr.Op {
	case OpInput:
		value, ok := env.lookup(expr.Name)
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
		index, ok, err := e.findObjectField(source, expr.Field)
		if err != nil {
			return Value{}, err
		}
		if !ok {
			if expr.Optional {
				return Null(), nil
			}
			return Value{}, fmt.Errorf("required field %q is missing", expr.Field)
		}
		return source.Object[index].Value, nil
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
			local := evalEnv{parent: env, bindingName: expr.Item, bindingValue: item, hasBinding: true}
			value, err := e.eval(*expr.Body, &local, depth+1)
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
		items := make([]sortItem, len(source.Array))
		for i, item := range source.Array {
			local := evalEnv{parent: env, bindingName: expr.Item, bindingValue: item, hasBinding: true}
			keys := make([]string, len(expr.Keys))
			for k := range expr.Keys {
				key, err := e.eval(expr.Keys[k], &local, depth+1)
				if err != nil {
					return Value{}, fmt.Errorf("sort-array item %d key %d: %w", i, k, err)
				}
				if key.Kind != ValueString {
					return Value{}, fmt.Errorf("sort-array keys must be strings, got %s", key.Kind)
				}
				keys[k] = key.String
			}
			items[i] = sortItem{value: item, keys: keys}
		}
		if err := stableSortItems(e, items); err != nil {
			return Value{}, err
		}
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
		index, exists, err := e.findObjectField(source, expr.Field)
		if err != nil {
			return Value{}, err
		}
		if !exists {
			return source, nil
		}
		replacement, err := e.eval(*expr.Value, env, depth+1)
		if err != nil {
			return Value{}, err
		}
		out := source
		out.Object = append([]Field(nil), source.Object...)
		out.Object[index].Value = replacement
		return out, nil
	case OpOmitField:
		source, err := e.eval(*expr.From, env, depth+1)
		if err != nil {
			return Value{}, err
		}
		index, exists, err := e.findObjectField(source, expr.Field)
		if err != nil {
			return Value{}, err
		}
		if !exists {
			return source, nil
		}
		out := Value{Kind: ValueObject, Object: make([]Field, 0, len(source.Object)-1)}
		out.Object = append(out.Object, source.Object[:index]...)
		out.Object = append(out.Object, source.Object[index+1:]...)
		return out, nil
	case OpAssertEqual:
		actual, err := e.eval(*expr.Actual, env, depth+1)
		if err != nil {
			return Value{}, err
		}
		expected, err := e.eval(*expr.Expected, env, depth+1)
		if err != nil {
			return Value{}, err
		}
		equal, err := e.equalValues(actual, expected)
		if err != nil {
			return Value{}, err
		}
		if !equal {
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
			equal, err := e.equalValues(actual, allowed)
			if err != nil {
				return Value{}, err
			}
			if equal {
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

type sortItem struct {
	value Value
	keys  []string
}

func stableSortItems(e *evaluator, items []sortItem) error {
	if len(items) < 2 {
		return nil
	}
	buffer := make([]sortItem, len(items))
	src := items
	dst := buffer
	srcIsItems := true
	for width := 1; width < len(items); width *= 2 {
		for left := 0; left < len(items); left += 2 * width {
			mid := left + width
			if mid > len(items) {
				mid = len(items)
			}
			right := left + 2*width
			if right > len(items) {
				right = len(items)
			}
			i, j, k := left, mid, left
			for i < mid && j < right {
				cmp, err := compareSortKeys(e, src[i].keys, src[j].keys)
				if err != nil {
					return err
				}
				if err := e.consume(1); err != nil {
					return err
				}
				if cmp <= 0 {
					dst[k] = src[i]
					i++
				} else {
					dst[k] = src[j]
					j++
				}
				k++
			}
			for i < mid {
				if err := e.consume(1); err != nil {
					return err
				}
				dst[k] = src[i]
				i++
				k++
			}
			for j < right {
				if err := e.consume(1); err != nil {
					return err
				}
				dst[k] = src[j]
				j++
				k++
			}
		}
		src, dst = dst, src
		srcIsItems = !srcIsItems
	}
	if !srcIsItems {
		for i := range items {
			if err := e.consume(1); err != nil {
				return err
			}
			items[i] = src[i]
		}
	}
	return nil
}

func compareSortKeys(e *evaluator, a, b []string) (int, error) {
	if len(a) != len(b) {
		return 0, fmt.Errorf("sort-array key tuple length mismatch")
	}
	for i := range a {
		if err := e.consume(1); err != nil {
			return 0, err
		}
		if a[i] < b[i] {
			return -1, nil
		}
		if a[i] > b[i] {
			return 1, nil
		}
	}
	return 0, nil
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

func (e *evaluator) findObjectField(value Value, name string) (int, bool, error) {
	for i := range value.Object {
		if err := e.consume(1); err != nil {
			return 0, false, err
		}
		if value.Object[i].Name == name {
			return i, true, nil
		}
	}
	return 0, false, nil
}

func (e *evaluator) equalValues(a, b Value) (bool, error) {
	if err := e.consume(1); err != nil {
		return false, err
	}
	if a.Kind != b.Kind {
		return false, nil
	}
	switch a.Kind {
	case ValueNull:
		return true, nil
	case ValueBool:
		return a.Bool == b.Bool, nil
	case ValueNumber:
		return a.Number == b.Number, nil
	case ValueString:
		return a.String == b.String, nil
	case ValueBytes:
		return bytes.Equal(a.Bytes, b.Bytes), nil
	case ValueArray:
		if len(a.Array) != len(b.Array) {
			return false, nil
		}
		for i := range a.Array {
			equal, err := e.equalValues(a.Array[i], b.Array[i])
			if err != nil || !equal {
				return equal, err
			}
		}
		return true, nil
	case ValueObject:
		if len(a.Object) != len(b.Object) {
			return false, nil
		}
		for i := range a.Object {
			if err := e.consume(1); err != nil {
				return false, err
			}
			if a.Object[i].Name != b.Object[i].Name {
				return false, nil
			}
			equal, err := e.equalValues(a.Object[i].Value, b.Object[i].Value)
			if err != nil || !equal {
				return equal, err
			}
		}
		return true, nil
	default:
		return false, fmt.Errorf("unsupported value kind %q", a.Kind)
	}
}

type evalEnv struct {
	root         map[string]Value
	parent       *evalEnv
	bindingName  string
	bindingValue Value
	hasBinding   bool
}

func (e *evalEnv) lookup(name string) (Value, bool) {
	for current := e; current != nil; current = current.parent {
		if current.hasBinding && current.bindingName == name {
			return current.bindingValue, true
		}
		if current.root != nil {
			value, ok := current.root[name]
			return value, ok
		}
	}
	return Value{}, false
}
