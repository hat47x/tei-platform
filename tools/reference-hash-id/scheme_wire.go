package hashid

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"unicode/utf8"
)

const SchemeWireFormat = "tei.hashid.scheme/v1"
const maxSchemeJSONBytes = 4096

// schemeWire is the canonical, mandatory, versioned representation of a
// derivation Scheme. Field order is significant only for canonical wire bytes.
// Presentation is in the scheme, but not included in encodedDigest's hash frame.
type schemeWire struct {
	SchemeID         string `json:"schemeId"`
	Purpose          string `json:"purpose"`
	Scope            string `json:"scope"`
	InputKind        string `json:"inputKind"`
	Canonicalization string `json:"canonicalization"`
	Algorithm        string `json:"algorithm"`
	Mode             string `json:"mode"`
	OutputBits       int    `json:"outputBits"`
	Encoding         string `json:"encoding"`
	Alphabet         string `json:"alphabet"`
	Length           int    `json:"length"`
}

type schemeEnvelope struct {
	Format string     `json:"format"`
	Scheme schemeWire `json:"scheme"`
}

func wireScheme(s Scheme) schemeWire {
	return schemeWire{
		SchemeID:         s.SchemeID,
		Purpose:          s.Purpose,
		Scope:            s.Scope,
		InputKind:        s.InputKind,
		Canonicalization: s.Canonicalization,
		Algorithm:        s.Algorithm,
		Mode:             s.Mode,
		OutputBits:       s.OutputBits,
		Encoding:         s.Encoding,
		Alphabet:         s.Alphabet,
		Length:           s.Length,
	}
}

func (w schemeWire) scheme() Scheme {
	return Scheme{
		SchemeID:         w.SchemeID,
		Purpose:          w.Purpose,
		Scope:            w.Scope,
		InputKind:        w.InputKind,
		Canonicalization: w.Canonicalization,
		Algorithm:        w.Algorithm,
		Mode:             w.Mode,
		OutputBits:       w.OutputBits,
		Encoding:         w.Encoding,
		Alphabet:         w.Alphabet,
		Length:           w.Length,
	}
}

// MarshalSchemeJSON emits a single canonical JSON object, without trailing
// newline or formatting whitespace. Its v1 field order is frozen and matches
// the standalone TypeScript serializer. No implicit defaults are omitted.
//
// The wire envelope is not a signature, stable semantic identity, or
// replacement for existing TEI artifact fingerprints.
func MarshalSchemeJSON(s Scheme) ([]byte, error) {
	if err := s.Validate(); err != nil {
		return nil, err
	}
	var out bytes.Buffer
	encoder := json.NewEncoder(&out)
	encoder.SetEscapeHTML(false) // v1 ASCII alphabets may include <, > and &.
	if err := encoder.Encode(schemeEnvelope{
		Format: SchemeWireFormat,
		Scheme: wireScheme(s),
	}); err != nil {
		return nil, err
	}
	return bytes.TrimSuffix(out.Bytes(), []byte("\n")), nil
}

// ParseSchemeJSON accepts *only* the canonical v1 wire representation.
// This prevents different languages from silently accepting duplicate JSON
// keys, unknown fields, nulls, absent zero-valued fields, alternate orders,
// noncanonical escapes and whitespace as equivalent scheme declarations.
// Authors may normalize input explicitly, but cannot silently do so here.
func ParseSchemeJSON(data []byte) (Scheme, error) {
	if len(data) == 0 || len(data) > maxSchemeJSONBytes || !utf8.Valid(data) {
		return Scheme{}, errors.New("invalid scheme JSON byte length or UTF-8")
	}
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	var doc schemeEnvelope
	if err := dec.Decode(&doc); err != nil {
		return Scheme{}, fmt.Errorf("invalid scheme JSON: %w", err)
	}
	var tail any
	if err := dec.Decode(&tail); err != io.EOF {
		return Scheme{}, errors.New("scheme JSON contains trailing data")
	}
	if doc.Format != SchemeWireFormat {
		return Scheme{}, fmt.Errorf("unsupported scheme wire format %q", doc.Format)
	}
	s := doc.Scheme.scheme()
	expected, err := MarshalSchemeJSON(s)
	if err != nil {
		return Scheme{}, fmt.Errorf("invalid scheme: %w", err)
	}
	if !bytes.Equal(data, expected) {
		return Scheme{}, errors.New("noncanonical scheme JSON: exact v1 wire bytes required")
	}
	return s, nil
}
