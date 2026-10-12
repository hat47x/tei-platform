package modelref

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

// DecodeGraph reads a single complete v0 graph without losing unfamiliar
// fields or silently overwriting repeated JSON members. It only validates
// the wire shape and schema. Call Validate with the applicable plugin rules
// before interpreting references; this function grants no Acceptance.
func DecodeGraph(data []byte) (Graph, error) {
	duplicates := json.NewDecoder(bytes.NewReader(data))
	if err := rejectRepeatedKeys(duplicates); err != nil {
		return Graph{}, fmt.Errorf("modelref graph JSON: %w", err)
	}
	if _, err := duplicates.Token(); err != io.EOF {
		if err == nil {
			return Graph{}, fmt.Errorf("modelref graph JSON: unexpected trailing value")
		}
		return Graph{}, fmt.Errorf("modelref graph JSON: %w", err)
	}

	reader := json.NewDecoder(bytes.NewReader(data))
	reader.DisallowUnknownFields()
	var graph Graph
	if err := reader.Decode(&graph); err != nil {
		return Graph{}, fmt.Errorf("modelref graph JSON: %w", err)
	}
	if graph.Schema != "tei.modelref.v0" {
		return Graph{}, fmt.Errorf("modelref graph JSON: unsupported schema %q", graph.Schema)
	}
	return graph, nil
}

// rejectRepeatedKeys walks a JSON value without interpreting its semantics.
// json.Unmarshal silently keeps the last occurrence of a repeated object
// member. That behavior could discard a prior subject, scope or provenance.
func rejectRepeatedKeys(decoder *json.Decoder) error {
	token, err := decoder.Token()
	if err != nil {
		return err
	}
	delim, ok := token.(json.Delim)
	if !ok {
		return nil
	}
	switch delim {
	case '{':
		seen := make(map[string]bool)
		for decoder.More() {
			nameToken, err := decoder.Token()
			if err != nil {
				return err
			}
			name, ok := nameToken.(string)
			if !ok {
				return fmt.Errorf("object key is not a string")
			}
			normalized := strings.ToLower(name)
			if seen[normalized] {
				return fmt.Errorf("repeated JSON field %q", name)
			}
			seen[normalized] = true
			if err := rejectRepeatedKeys(decoder); err != nil {
				return err
			}
		}
		end, err := decoder.Token()
		if err != nil {
			return err
		}
		if end != json.Delim('}') {
			return fmt.Errorf("object was not closed")
		}
	case '[':
		for decoder.More() {
			if err := rejectRepeatedKeys(decoder); err != nil {
				return err
			}
		}
		end, err := decoder.Token()
		if err != nil {
			return err
		}
		if end != json.Delim(']') {
			return fmt.Errorf("array was not closed")
		}
	default:
		return fmt.Errorf("unexpected JSON delimiter %q", delim)
	}
	return nil
}
