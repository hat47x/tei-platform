package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
)

// rejectAmbiguousJSON rules out duplicate keys at any object depth before
// decoding imported evidence as a typed E0 Report. The standard Go JSON
// decoder otherwise accepts repeated keys and silently overwrites values.
func rejectAmbiguousJSON(content []byte) error {
	decoder := json.NewDecoder(bytes.NewReader(content))
	var walk func(depth int) error
	walk = func(depth int) error {
		if depth > 64 {
			return fmt.Errorf("JSON nesting exceeds E0 safety limit")
		}
		token, err := decoder.Token()
		if err != nil {
			return fmt.Errorf("read JSON token: %w", err)
		}
		open, isDelimiter := token.(json.Delim)
		if !isDelimiter {
			return nil
		}
		switch open {
		case '{':
			keys := make(map[string]struct{})
			for decoder.More() {
				property, err := decoder.Token()
				if err != nil {
					return fmt.Errorf("read JSON object property: %w", err)
				}
				key, ok := property.(string)
				if !ok {
					return fmt.Errorf("JSON object has non-string property")
				}
				if _, exists := keys[key]; exists {
					return fmt.Errorf("duplicate JSON property %q", key)
				}
				keys[key] = struct{}{}
				if err := walk(depth + 1); err != nil {
					return err
				}
			}
			end, err := decoder.Token()
			if err != nil || end != json.Delim('}') {
				return fmt.Errorf("JSON object has invalid ending")
			}
		case '[':
			for decoder.More() {
				if err := walk(depth + 1); err != nil {
					return err
				}
			}
			end, err := decoder.Token()
			if err != nil || end != json.Delim(']') {
				return fmt.Errorf("JSON array has invalid ending")
			}
		default:
			return fmt.Errorf("unexpected JSON delimiter %q", open)
		}
		return nil
	}
	if err := walk(0); err != nil {
		return err
	}
	if _, err := decoder.Token(); err != io.EOF {
		return fmt.Errorf("JSON contains trailing content")
	}
	return nil
}
