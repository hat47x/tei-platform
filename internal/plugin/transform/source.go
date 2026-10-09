package transform

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
)

// ParseSourceJSON decodes the provisional file representation of TEI's
// deterministic transform IR. JSON is only a serialization of the TEI-owned
// IR; it is not a general-purpose Plugin language or an authority boundary.
func ParseSourceJSON(raw []byte) (Source, error) {
	if _, err := ParseJSON(raw); err != nil {
		return Source{}, fmt.Errorf("transform source JSON: %w", err)
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	var source Source
	if err := decoder.Decode(&source); err != nil {
		return Source{}, fmt.Errorf("decode transform source: %w", err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		if err == nil {
			return Source{}, fmt.Errorf("transform source contains multiple values")
		}
		return Source{}, fmt.Errorf("decode trailing transform source content: %w", err)
	}
	if err := source.Validate(); err != nil {
		return Source{}, err
	}
	return source, nil
}
