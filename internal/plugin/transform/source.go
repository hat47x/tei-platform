package transform

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
)

const MaxSerializedSourceBytes = 8 << 20

// ParseSourceJSON decodes the provisional file representation of TEI's
// deterministic transform IR. JSON is only a serialization of the TEI-owned
// IR; it is not a general-purpose Plugin language or an authority boundary.
func ParseSourceJSON(raw []byte) (Source, error) {
	if len(raw) > MaxSerializedSourceBytes {
		return Source{}, fmt.Errorf("transform Source exceeds serialized size limit")
	}
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


// SourceValue is the Runtime Engine's typed input projection for a validated,
// normalized transform Source. It contains no Host state or source locator.
func SourceValue(source Source) (Value, error) {
	normalized, err := NormalizeSource(source)
	if err != nil {
		return Value{}, err
	}
	raw, err := json.Marshal(normalized)
	if err != nil {
		return Value{}, err
	}
	return ParseJSON(raw)
}
