package main

import (
	"bytes"
	"encoding/json"

	"tei/internal/sourceindex"
)

// procedureIDValueSpan identifies the encoded value of the *top-level*
// procedure.id, rather than any occurrence of the same text elsewhere.
// Duplicate procedure objects or ID keys are ambiguous and fail closed.
// The returned half-open byte span excludes the JSON string delimiters.
func procedureIDValueSpan(raw []byte) (id string, start, end int, ok bool) {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	token, err := decoder.Token()
	if err != nil || token != json.Delim('{') {
		return "", 0, 0, false
	}
	foundProcedure := false
	foundID := false
	for decoder.More() {
		keyToken, err := decoder.Token()
		key, isString := keyToken.(string)
		if err != nil || !isString {
			return "", 0, 0, false
		}
		if key != "procedure" {
			var skipped json.RawMessage
			if err := decoder.Decode(&skipped); err != nil {
				return "", 0, 0, false
			}
			continue
		}
		if foundProcedure {
			return "", 0, 0, false
		}
		foundProcedure = true
		nested, err := decoder.Token()
		if err != nil || nested != json.Delim('{') {
			return "", 0, 0, false
		}
		for decoder.More() {
			nameToken, err := decoder.Token()
			name, isString := nameToken.(string)
			if err != nil || !isString {
				return "", 0, 0, false
			}
			if name != "id" {
				var skipped json.RawMessage
				if err := decoder.Decode(&skipped); err != nil {
					return "", 0, 0, false
				}
				continue
			}
			if foundID {
				return "", 0, 0, false
			}
			before := int(decoder.InputOffset())
			valueToken, err := decoder.Token()
			value, isString := valueToken.(string)
			after := int(decoder.InputOffset())
			if err != nil || !isString || value == "" || before < 0 || after > len(raw) || before >= after {
				return "", 0, 0, false
			}
			opening := bytes.IndexByte(raw[before:after], '"')
			if opening < 0 {
				return "", 0, 0, false
			}
			quotedStart := before + opening
			var encodedValue string
			if err := json.Unmarshal(raw[quotedStart:after], &encodedValue); err != nil || encodedValue != value ||
				after-1 <= quotedStart {
				return "", 0, 0, false
			}
			id, start, end = value, quotedStart+1, after-1
			foundID = true
		}
		closeToken, err := decoder.Token()
		if err != nil || closeToken != json.Delim('}') {
			return "", 0, 0, false
		}
	}
	closeToken, err := decoder.Token()
	if err != nil || closeToken != json.Delim('}') || !foundProcedure || !foundID {
		return "", 0, 0, false
	}
	return id, start, end, true
}

// The Workspace SourceContribution pathway currently records full-file
// ranges, while an exact authoring reference may select just the ID text or
// a zero-width point within it. All three are valid source locations.
func procedureIDCoveredByReference(r sourceindex.Range, start, end int) bool {
	if r.Start.Offset == r.End.Offset {
		return r.Start.Offset >= start && r.Start.Offset <= end
	}
	return r.Start.Offset <= start && r.End.Offset >= end
}
