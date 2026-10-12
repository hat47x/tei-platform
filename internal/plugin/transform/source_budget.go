package transform

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

// sourceSerializedSize checks the JSON size before encoding allocates a buffer.
// It mirrors the fields and omitempty rules of encoding/json for TEI's closed
// Source structs. It is only a resource gate: encoding/json remains the format
// authority, and the final bytes are checked and parsed separately.
func sourceSerializedSize(source Source, limit int) error {
	budget := sourceSizeBudget{
		remaining: limit,
		active:    make(map[uintptr]bool),
	}
	if err := budget.measure(reflect.ValueOf(source), 0); err != nil {
		return fmt.Errorf("transform Source exceeds serialized size limit or structural budget: %w", err)
	}
	return nil
}

type sourceSizeBudget struct {
	remaining int
	nodes     int
	active    map[uintptr]bool
}

func (b *sourceSizeBudget) add(n int) error {
	if n < 0 || n > b.remaining {
		return fmt.Errorf("serialized size limit exceeded")
	}
	b.remaining -= n
	return nil
}

func (b *sourceSizeBudget) quoted(s string) error {
	n, err := jsonStringEncodedSize(s, b.remaining)
	if err != nil {
		return err
	}
	return b.add(n)
}

func (b *sourceSizeBudget) measure(v reflect.Value, depth int) error {
	if depth > 512 {
		return fmt.Errorf("source nesting is too deep")
	}
	b.nodes++
	if b.nodes > 1_000_000 {
		return fmt.Errorf("source node count exceeds limit")
	}
	if !v.IsValid() {
		return b.add(4)
	}

	// Literal and allowed values are already JSON, not base64 byte slices.
	if v.Type() == reflect.TypeOf(json.RawMessage(nil)) {
		if v.IsNil() {
			return b.add(4)
		}
		return b.add(v.Len())
	}

	switch v.Kind() {
	case reflect.Pointer:
		if v.IsNil() {
			return b.add(4)
		}
		p := v.Pointer()
		if b.active[p] {
			return fmt.Errorf("cyclic transform Source")
		}
		b.active[p] = true
		defer delete(b.active, p)
		return b.measure(v.Elem(), depth+1)
	case reflect.String:
		return b.quoted(v.String())
	case reflect.Bool:
		if v.Bool() {
			return b.add(4)
		}
		return b.add(5)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return b.add(len(strconv.FormatInt(v.Int(), 10)))
	case reflect.Slice, reflect.Array:
		if v.Kind() == reflect.Slice && v.IsNil() {
			return b.add(4)
		}
		if err := b.add(2); err != nil {
			return err
		}
		for i := 0; i < v.Len(); i++ {
			if i > 0 {
				if err := b.add(1); err != nil {
					return err
				}
			}
			if err := b.measure(v.Index(i), depth+1); err != nil {
				return err
			}
		}
		return nil
	case reflect.Struct:
		if err := b.add(2); err != nil {
			return err
		}
		first := true
		t := v.Type()
		for i := 0; i < v.NumField(); i++ {
			f := t.Field(i)
			if f.PkgPath != "" {
				continue
			}
			tag := strings.Split(f.Tag.Get("json"), ",")
			if tag[0] == "-" {
				continue
			}
			field := v.Field(i)
			if len(tag) > 1 && containsJSONOmitEmpty(tag[1:]) && sourceFieldEmpty(field) {
				continue
			}
			if !first {
				if err := b.add(1); err != nil {
					return err
				}
			}
			first = false
			name := tag[0]
			if name == "" {
				name = f.Name
			}
			if err := b.quoted(name); err != nil {
				return err
			}
			if err := b.add(1); err != nil {
				return err
			}
			if err := b.measure(field, depth+1); err != nil {
				return err
			}
		}
		return nil
	default:
		return fmt.Errorf("unsupported Source field type %s", v.Type())
	}
}

func containsJSONOmitEmpty(options []string) bool {
	for _, option := range options {
		if option == "omitempty" {
			return true
		}
	}
	return false
}

func sourceFieldEmpty(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.Array, reflect.Slice, reflect.Map, reflect.String:
		return v.Len() == 0
	case reflect.Bool:
		return !v.Bool()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int() == 0
	case reflect.Interface, reflect.Pointer:
		return v.IsNil()
	default:
		return false
	}
}
