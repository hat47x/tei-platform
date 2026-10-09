package transform

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"unicode/utf8"
)

type ValueKind string

const (
	ValueNull   ValueKind = "null"
	ValueBool   ValueKind = "bool"
	ValueNumber ValueKind = "number"
	ValueString ValueKind = "string"
	ValueArray  ValueKind = "array"
	ValueObject ValueKind = "object"
	ValueBytes  ValueKind = "bytes"
)

type Field struct {
	Name  string
	Value Value
}

// Value is the deterministic value domain used by the transform evaluator.
// Objects preserve explicit field order and reject duplicate fields.
type Value struct {
	Kind   ValueKind
	Bool   bool
	Number string
	String string
	Array  []Value
	Object []Field
	Bytes  []byte
}

func Null() Value                 { return Value{Kind: ValueNull} }
func Bool(v bool) Value           { return Value{Kind: ValueBool, Bool: v} }
func Number(v string) Value       { return Value{Kind: ValueNumber, Number: v} }
func String(v string) Value       { return Value{Kind: ValueString, String: v} }
func Bytes(v []byte) Value        { return Value{Kind: ValueBytes, Bytes: append([]byte(nil), v...)} }
func Array(values ...Value) Value { return Value{Kind: ValueArray, Array: append([]Value(nil), values...)} }

func Object(fields ...Field) (Value, error) {
	seen := map[string]struct{}{}
	out := Value{Kind: ValueObject, Object: make([]Field, len(fields))}
	for i, field := range fields {
		if field.Name == "" {
			return Value{}, fmt.Errorf("object field name is required")
		}
		if _, exists := seen[field.Name]; exists {
			return Value{}, fmt.Errorf("object field %q is declared more than once", field.Name)
		}
		seen[field.Name] = struct{}{}
		out.Object[i] = Field{Name: field.Name, Value: field.Value.Clone()}
	}
	return out, nil
}

func (v Value) Clone() Value {
	out := v
	out.Bytes = append([]byte(nil), v.Bytes...)
	out.Array = make([]Value, len(v.Array))
	for i := range v.Array {
		out.Array[i] = v.Array[i].Clone()
	}
	out.Object = make([]Field, len(v.Object))
	for i := range v.Object {
		out.Object[i] = Field{Name: v.Object[i].Name, Value: v.Object[i].Value.Clone()}
	}
	return out
}

func (v Value) Field(name string) (Value, bool) {
	if v.Kind != ValueObject {
		return Value{}, false
	}
	for _, field := range v.Object {
		if field.Name == name {
			return field.Value.Clone(), true
		}
	}
	return Value{}, false
}

func (v Value) ReplaceField(name string, replacement Value) (Value, bool, error) {
	if v.Kind != ValueObject {
		return Value{}, false, fmt.Errorf("replace-field requires object, got %s", v.Kind)
	}
	out := v.Clone()
	for i := range out.Object {
		if out.Object[i].Name == name {
			out.Object[i].Value = replacement.Clone()
			return out, true, nil
		}
	}
	return out, false, nil
}

func (v Value) OmitField(name string) (Value, error) {
	if v.Kind != ValueObject {
		return Value{}, fmt.Errorf("omit-field requires object, got %s", v.Kind)
	}
	out := Value{Kind: ValueObject}
	for _, field := range v.Object {
		if field.Name == name {
			continue
		}
		out.Object = append(out.Object, Field{Name: field.Name, Value: field.Value.Clone()})
	}
	return out, nil
}

func ParseJSON(raw []byte) (Value, error) {
	if !utf8.Valid(raw) {
		return Value{}, fmt.Errorf("JSON is not valid UTF-8")
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	value, err := decodeJSONValue(decoder, 0)
	if err != nil {
		return Value{}, err
	}
	if token, err := decoder.Token(); err != io.EOF {
		if err != nil {
			return Value{}, err
		}
		return Value{}, fmt.Errorf("JSON contains multiple values after %v", token)
	}
	return value, nil
}

func decodeJSONValue(decoder *json.Decoder, depth int) (Value, error) {
	if depth > 128 {
		return Value{}, fmt.Errorf("JSON nesting exceeds 128 levels")
	}
	token, err := decoder.Token()
	if err != nil {
		return Value{}, err
	}
	switch value := token.(type) {
	case nil:
		return Null(), nil
	case bool:
		return Bool(value), nil
	case string:
		return String(value), nil
	case json.Number:
		return Number(value.String()), nil
	case json.Delim:
		switch value {
		case '{':
			var fields []Field
			seen := map[string]struct{}{}
			for decoder.More() {
				keyToken, err := decoder.Token()
				if err != nil {
					return Value{}, err
				}
				key, ok := keyToken.(string)
				if !ok {
					return Value{}, fmt.Errorf("JSON object key is not a string")
				}
				if _, exists := seen[key]; exists {
					return Value{}, fmt.Errorf("JSON object key %q is declared more than once", key)
				}
				seen[key] = struct{}{}
				fieldValue, err := decodeJSONValue(decoder, depth+1)
				if err != nil {
					return Value{}, err
				}
				fields = append(fields, Field{Name: key, Value: fieldValue})
			}
			closeToken, err := decoder.Token()
			if err != nil {
				return Value{}, err
			}
			if closeToken != json.Delim('}') {
				return Value{}, fmt.Errorf("JSON object is not terminated")
			}
			return Object(fields...)
		case '[':
			var values []Value
			for decoder.More() {
				item, err := decodeJSONValue(decoder, depth+1)
				if err != nil {
					return Value{}, err
				}
				values = append(values, item)
			}
			closeToken, err := decoder.Token()
			if err != nil {
				return Value{}, err
			}
			if closeToken != json.Delim(']') {
				return Value{}, fmt.Errorf("JSON array is not terminated")
			}
			return Array(values...), nil
		default:
			return Value{}, fmt.Errorf("unexpected JSON delimiter %q", value)
		}
	default:
		return Value{}, fmt.Errorf("unsupported JSON token %T", token)
	}
}

func (v Value) JSON() ([]byte, error) {
	var out bytes.Buffer
	if err := writeJSON(&out, v, 0); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}

func (v Value) JSONWithinLimit(maxBytes int) ([]byte, error) {
	if maxBytes < 0 {
		return nil, fmt.Errorf("JSON byte limit must be non-negative")
	}
	if _, err := jsonEncodedSizeWithin(v, maxBytes, 0); err != nil {
		return nil, err
	}
	return v.JSON()
}

func jsonEncodedSizeWithin(v Value, maxBytes, depth int) (int, error) {
	if depth > 128 {
		return 0, fmt.Errorf("JSON value nesting exceeds 128 levels")
	}
	add := func(total, amount int) (int, error) {
		if amount < 0 || total < 0 || total > maxBytes || amount > maxBytes-total {
			return 0, fmt.Errorf("JSON encoding exceeds byte limit")
		}
		return total + amount, nil
	}
	switch v.Kind {
	case ValueNull:
		return add(0, len("null"))
	case ValueBool:
		if v.Bool {
			return add(0, len("true"))
		}
		return add(0, len("false"))
	case ValueNumber:
		if !validJSONNumber(v.Number) {
			return 0, fmt.Errorf("invalid JSON number %q", v.Number)
		}
		return add(0, len(v.Number))
	case ValueString:
		if !utf8.ValidString(v.String) {
			return 0, fmt.Errorf("string value is not valid UTF-8")
		}
		raw, _ := json.Marshal(v.String)
		return add(0, len(raw))
	case ValueBytes:
		return 0, fmt.Errorf("byte value cannot be encoded as JSON directly")
	case ValueArray:
		total, err := add(0, 2)
		if err != nil {
			return 0, err
		}
		for i, item := range v.Array {
			if i > 0 {
				total, err = add(total, 1)
				if err != nil {
					return 0, err
				}
			}
			remaining := maxBytes - total
			size, err := jsonEncodedSizeWithin(item, remaining, depth+1)
			if err != nil {
				return 0, err
			}
			total, err = add(total, size)
			if err != nil {
				return 0, err
			}
		}
		return total, nil
	case ValueObject:
		total, err := add(0, 2)
		if err != nil {
			return 0, err
		}
		seen := map[string]struct{}{}
		for i, field := range v.Object {
			if field.Name == "" {
				return 0, fmt.Errorf("object field name is required")
			}
			if !utf8.ValidString(field.Name) {
				return 0, fmt.Errorf("object field name is not valid UTF-8")
			}
			if _, exists := seen[field.Name]; exists {
				return 0, fmt.Errorf("object field %q is declared more than once", field.Name)
			}
			seen[field.Name] = struct{}{}
			if i > 0 {
				total, err = add(total, 1)
				if err != nil {
					return 0, err
				}
			}
			name, _ := json.Marshal(field.Name)
			total, err = add(total, len(name)+1)
			if err != nil {
				return 0, err
			}
			remaining := maxBytes - total
			size, err := jsonEncodedSizeWithin(field.Value, remaining, depth+1)
			if err != nil {
				return 0, err
			}
			total, err = add(total, size)
			if err != nil {
				return 0, err
			}
		}
		return total, nil
	default:
		return 0, fmt.Errorf("unknown value kind %q", v.Kind)
	}
}

func writeJSON(out *bytes.Buffer, v Value, depth int) error {
	if depth > 128 {
		return fmt.Errorf("JSON value nesting exceeds 128 levels")
	}
	switch v.Kind {
	case ValueNull:
		out.WriteString("null")
	case ValueBool:
		if v.Bool {
			out.WriteString("true")
		} else {
			out.WriteString("false")
		}
	case ValueNumber:
		if !validJSONNumber(v.Number) {
			return fmt.Errorf("invalid JSON number %q", v.Number)
		}
		out.WriteString(v.Number)
	case ValueString:
		if !utf8.ValidString(v.String) {
			return fmt.Errorf("string value is not valid UTF-8")
		}
		raw, _ := json.Marshal(v.String)
		out.Write(raw)
	case ValueArray:
		out.WriteByte('[')
		for i, item := range v.Array {
			if i > 0 {
				out.WriteByte(',')
			}
			if err := writeJSON(out, item, depth+1); err != nil {
				return err
			}
		}
		out.WriteByte(']')
	case ValueObject:
		seen := map[string]struct{}{}
		out.WriteByte('{')
		for i, field := range v.Object {
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
			if i > 0 {
				out.WriteByte(',')
			}
			name, _ := json.Marshal(field.Name)
			out.Write(name)
			out.WriteByte(':')
			if err := writeJSON(out, field.Value, depth+1); err != nil {
				return err
			}
		}
		out.WriteByte('}')
	case ValueBytes:
		return fmt.Errorf("byte value cannot be encoded as JSON directly")
	default:
		return fmt.Errorf("unknown value kind %q", v.Kind)
	}
	return nil
}



func validJSONNumber(raw string) bool {
	if strings.TrimSpace(raw) != raw || raw == "" {
		return false
	}
	decoder := json.NewDecoder(strings.NewReader(raw))
	decoder.UseNumber()
	token, err := decoder.Token()
	if err != nil {
		return false
	}
	if _, ok := token.(json.Number); !ok {
		return false
	}
	_, err = decoder.Token()
	return err == io.EOF
}

func Equal(a, b Value) bool {
	if a.Kind != b.Kind {
		return false
	}
	switch a.Kind {
	case ValueNull:
		return true
	case ValueBool:
		return a.Bool == b.Bool
	case ValueNumber:
		return a.Number == b.Number
	case ValueString:
		return a.String == b.String
	case ValueBytes:
		return bytes.Equal(a.Bytes, b.Bytes)
	case ValueArray:
		if len(a.Array) != len(b.Array) {
			return false
		}
		for i := range a.Array {
			if !Equal(a.Array[i], b.Array[i]) {
				return false
			}
		}
		return true
	case ValueObject:
		if len(a.Object) != len(b.Object) {
			return false
		}
		for i := range a.Object {
			if a.Object[i].Name != b.Object[i].Name || !Equal(a.Object[i].Value, b.Object[i].Value) {
				return false
			}
		}
		return true
	default:
		return false
	}
}
