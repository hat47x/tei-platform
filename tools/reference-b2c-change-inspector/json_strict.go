package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"unicode"
)

// inspectJSONKeys rejects repeated object keys at every depth before the
// regular typed decoder sees the input. JSON field names that differ only
// by case are also rejected: encoding/json accepts case-insensitive matches
// for struct fields, which otherwise permits shadowed control flags.
func inspectJSONKeys(data []byte) error {
	dec := json.NewDecoder(bytes.NewReader(data))
	if err := walkJSONObjectKeys(dec); err != nil {
		return err
	}
	if _, err := dec.Token(); err != io.EOF {
		if err == nil {
			return errors.New("trailing JSON value")
		}
		return fmt.Errorf("trailing JSON value: %w", err)
	}
	return nil
}

func walkJSONObjectKeys(dec *json.Decoder) error {
	tok, err := dec.Token()
	if err != nil {
		return err
	}
	delim, ok := tok.(json.Delim)
	if !ok {
		return nil
	}
	switch delim {
	case '{':
		keys := map[string]bool{}
		for dec.More() {
			keyToken, err := dec.Token()
			if err != nil {
				return err
			}
			key, ok := keyToken.(string)
			if !ok {
				return errors.New("JSON object key is not a string")
			}
			canonical := simpleFoldJSONKey(key)
			if keys[canonical] {
				return fmt.Errorf("duplicate or case-shadowed JSON key %q", key)
			}
			keys[canonical] = true
			if err := walkJSONObjectKeys(dec); err != nil {
				return err
			}
		}
		_, err := dec.Token() // closing brace, checked by Decoder
		return err
	case '[':
		for dec.More() {
			if err := walkJSONObjectKeys(dec); err != nil {
				return err
			}
		}
		_, err := dec.Token() // closing bracket, checked by Decoder
		return err
	default:
		return fmt.Errorf("unexpected JSON delimiter %q", delim)
	}
}


// simpleFoldJSONKey canonicalizes each Unicode simple-fold equivalence class.
// encoding/json matches struct fields with Unicode simple folding, which is
// broader than strings.ToLower: "scope" and "ſcope" address the same field.
// A map keyed by strings.ToLower therefore permits a later declaration to
// overwrite an earlier one without a duplicate-key error. This normalization
// is deterministic and linear in key length; it does not infer any domain
// meaning from the JSON document.
func simpleFoldJSONKey(key string) string {
	var folded strings.Builder
	folded.Grow(len(key))
	for _, r := range key {
		min := r
		for next := unicode.SimpleFold(r); next != r; next = unicode.SimpleFold(next) {
			if next < min {
				min = next
			}
		}
		folded.WriteRune(min)
	}
	return folded.String()
}
