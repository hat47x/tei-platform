package buildintent

import (
	"fmt"
	"reflect"
	"strings"
	"unicode"

	"tei/internal/plugin/transform"
)

// ValidateInputJSON shares the existing strict value parser with plugin
// transforms before decoding an authoring schema. It also rejects field aliases
// that encoding/json would fold into the same struct member.
func ValidateInputJSON(raw []byte) error {
	value, err := transform.ParseJSON(raw)
	if err != nil {
		return err
	}
	return rejectFoldedInputFields(value)
}

// foldedInputField matches encoding/json's Unicode simple folding (including
// Kelvin sign and long s), not only ASCII lowercasing. Using a set keeps field
// checks linear.
func foldedInputField(name string) string {
	return strings.Map(func(r rune) rune {
		smallest := r
		for next := unicode.SimpleFold(r); next != r; next = unicode.SimpleFold(next) {
			if next < smallest {
				smallest = next
			}
		}
		return smallest
	}, name)
}

func rejectFoldedInputFields(value transform.Value) error {
	switch value.Kind {
	case transform.ValueObject:
		seen := map[string]bool{}
		for _, field := range value.Object {
			key := foldedInputField(field.Name)
			if seen[key] {
				return fmt.Errorf("JSON field %q has a duplicate case-folded alias", field.Name)
			}
			seen[key] = true
			if err := rejectFoldedInputFields(field.Value); err != nil {
				return err
			}
		}
	case transform.ValueArray:
		for _, item := range value.Array {
			if err := rejectFoldedInputFields(item); err != nil {
				return err
			}
		}
	}
	return nil
}

// RejectKeyCaseVariants refuses a key that does not match a struct's json name
// exactly. encoding/json accepts case variants, which would let a differently
// spelled key pass.
func RejectKeyCaseVariants(v any, t reflect.Type, path string) error {
	switch t.Kind() {
	case reflect.Pointer:
		return RejectKeyCaseVariants(v, t.Elem(), path)
	case reflect.Slice:
		list, ok := v.([]any)
		if !ok {
			return nil
		}
		for i, item := range list {
			if err := RejectKeyCaseVariants(item, t.Elem(), fmt.Sprintf("%s[%d]", path, i)); err != nil {
				return err
			}
		}
		return nil
	}
	node, ok := v.(map[string]any)
	if !ok || t.Kind() != reflect.Struct {
		return nil
	}
	fields := map[string]reflect.Type{}
	for i := 0; i < t.NumField(); i++ {
		name := strings.Split(t.Field(i).Tag.Get("json"), ",")[0]
		if name != "" && name != "-" {
			fields[name] = t.Field(i).Type
		}
	}
	for key, child := range node {
		ft, exact := fields[key]
		if !exact {
			return fmt.Errorf("input JSON key %q at %s does not match a field name exactly (case variants and unknown keys are rejected)", key, path)
		}
		if err := RejectKeyCaseVariants(child, ft, path+"."+key); err != nil {
			return err
		}
	}
	return nil
}
