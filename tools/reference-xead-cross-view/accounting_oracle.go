package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"reflect"
	"strings"

	"tei/internal/xeadsource"
)

// verifyPublicAccountingBaseline compares a newly sourced public aggregate
// with a previously recorded *public* command output. Exact source digest,
// mapping-contract fingerprint, derivation policy, all counts and authority
// limits must agree. This is a reproducibility gate, not proof that either
// document came from the original system or had accepted business meaning.
func verifyPublicAccountingBaseline(
	actual xeadsource.CrossViewAccounting, raw []byte,
) error {
	check := json.NewDecoder(bytes.NewReader(raw))
	if err := rejectDuplicateJSONMembers(check); err != nil {
		return fmt.Errorf("invalid public accounting baseline: %w", err)
	}
	if _, err := check.Token(); err != io.EOF {
		return fmt.Errorf("public accounting baseline must contain exactly one JSON value")
	}

	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	var saved struct {
		Accounting *xeadsource.CrossViewAccounting `json:"accounting"`
	}
	if err := dec.Decode(&saved); err != nil {
		return fmt.Errorf("invalid public accounting baseline: %w", err)
	}
	if saved.Accounting == nil {
		return fmt.Errorf("public accounting baseline is missing accounting")
	}
	if _, err := dec.Token(); err != io.EOF {
		return fmt.Errorf("public accounting baseline has trailing JSON content")
	}
	if !reflect.DeepEqual(*saved.Accounting, actual) {
		// Intentionally omit counts, source paths, token values and digests
		// from errors. Verification output is safe to publish by default.
		return fmt.Errorf("source-derived public accounting differs from saved baseline")
	}
	return nil
}

// rejectDuplicateJSONMembers rejects duplicate keys at every depth, including
// case-only aliases that encoding/json would otherwise accept for struct
// fields. A refingerprinted or overwritten expected count must not be
// interpreted as an unambiguous conformance record.
func rejectDuplicateJSONMembers(dec *json.Decoder) error {
	token, err := dec.Token()
	if err != nil {
		return err
	}
	delim, ok := token.(json.Delim)
	if !ok {
		return nil
	}
	switch delim {
	case '{':
		var names []string
		for dec.More() {
			keyToken, err := dec.Token()
			if err != nil {
				return err
			}
			key, ok := keyToken.(string)
			if !ok {
				return fmt.Errorf("JSON object member is not a string")
			}
			for _, name := range names {
				if strings.EqualFold(name, key) {
					return fmt.Errorf("public accounting baseline has duplicate JSON members")
				}
			}
			names = append(names, key)
			if err := rejectDuplicateJSONMembers(dec); err != nil {
				return err
			}
		}
		end, err := dec.Token()
		if err != nil {
			return err
		}
		if end != json.Delim('}') {
			return fmt.Errorf("invalid JSON object terminator")
		}
	case '[':
		for dec.More() {
			if err := rejectDuplicateJSONMembers(dec); err != nil {
				return err
			}
		}
		end, err := dec.Token()
		if err != nil {
			return err
		}
		if end != json.Delim(']') {
			return fmt.Errorf("invalid JSON array terminator")
		}
	default:
		return fmt.Errorf("unexpected JSON delimiter")
	}
	return nil
}
