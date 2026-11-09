package registryreference

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"unicode/utf8"
)

// This deliberately small wire shape is for the P0-A fixture, not a TEI schema.
var wireFields = map[string]map[string]string{
	"fixture":     {"releaseText": "string", "publisherSignatureBase64": "string", "profileText": "string", "artifacts": "artifacts", "attestation": "attestation"},
	"artifact":    {"role": "string", "bytesBase64": "string"},
	"attestation": {"subject": "string", "profileDigest": "string", "target": "string", "result": "string", "signatureBase64": "string"},
}

func parseWireValue(dec *json.Decoder, shape string, depth int) error {
	if depth > 32 {
		return errors.New("wire nesting too deep")
	}
	tok, err := dec.Token()
	if err != nil {
		return err
	}
	switch shape {
	case "fixture", "artifact", "attestation":
		if tok != json.Delim('{') {
			return fmt.Errorf("%s: expected object", shape)
		}
		fields := wireFields[shape]
		seen := make(map[string]bool, len(fields))
		for dec.More() {
			keyToken, err := dec.Token()
			if err != nil {
				return err
			}
			key, ok := keyToken.(string)
			if !ok {
				return errors.New("non-string wire key")
			}
			child, allowed := fields[key]
			if !allowed || seen[key] {
				return fmt.Errorf("%s: unknown or duplicate wire field %q", shape, key)
			}
			seen[key] = true
			if err := parseWireValue(dec, child, depth+1); err != nil {
				return err
			}
		}
		closeToken, err := dec.Token()
		if err != nil {
			return err
		}
		if closeToken != json.Delim('}') || len(seen) != len(fields) {
			return fmt.Errorf("%s: missing field or invalid object framing", shape)
		}
	case "artifacts":
		if tok != json.Delim('[') {
			return errors.New("artifacts: expected array")
		}
		for dec.More() {
			if err := parseWireValue(dec, "artifact", depth+1); err != nil {
				return err
			}
		}
		closeToken, err := dec.Token()
		if err != nil {
			return err
		}
		if closeToken != json.Delim(']') {
			return errors.New("artifacts: invalid array framing")
		}
	case "string":
		if _, ok := tok.(string); !ok {
			return errors.New("expected wire string")
		}
	default:
		return fmt.Errorf("unsupported wire shape %q", shape)
	}
	return nil
}

func ParseFixture(raw []byte) (Fixture, error) {
	if len(raw) > 1<<20 || !utf8.Valid(raw) {
		return Fixture{}, errors.New("invalid UTF-8 or oversized wire")
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	if err := parseWireValue(dec, "fixture", 0); err != nil {
		return Fixture{}, err
	}
	if _, err := dec.Token(); err != io.EOF {
		return Fixture{}, errors.New("trailing wire data")
	}
	var f Fixture
	if err := json.Unmarshal(raw, &f); err != nil {
		return Fixture{}, err
	}
	return f, nil
}

// StdEncoding.Strict still ignores CR/LF. Re-encoding fixes that discrepancy.
func decodeCanonicalBase64(s string) ([]byte, error) {
	b, err := base64.StdEncoding.Strict().DecodeString(s)
	if err != nil || base64.StdEncoding.EncodeToString(b) != s {
		return nil, errors.New("noncanonical base64")
	}
	return b, nil
}
