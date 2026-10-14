package main

import (
	"fmt"
	"strings"
	"unicode"

	"tei/internal/plugin/transform"
)

// validateInputJSON shares the existing strict value parser with plugin
// transforms before decoding the reference authoring schema. It also rejects
// field aliases that encoding/json would fold into the same struct member.
func validateInputJSON(raw []byte) error {
	value, err := transform.ParseJSON(raw)
	if err != nil {
		return err
	}
	return rejectFoldedInputFields(value)
}

// Match encoding/json's Unicode simple folding (including Kelvin sign and
// long s), not only ASCII lowercasing. Using a set keeps field checks linear.
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
