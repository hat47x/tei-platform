package monthendclose

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"unicode/utf8"

	"golang.org/x/text/encoding"
	"golang.org/x/text/encoding/japanese"
)

// FileSpecSchema identifies the declarative fixed-length file specification.
const FileSpecSchema = "tei.reference.mecc-file-spec/v0"

// Field is one byte range of a record.
type Field struct {
	Name   string `json:"name"`
	Offset int    `json:"offset"`
	Length int    `json:"length"`
	// Type is "const" (Value, compared byte for byte), "alnum" (ASCII,
	// left-aligned, space padded), "text" (encoded with the file encoding,
	// left-aligned, padded with ASCII spaces), "number" (unsigned decimal,
	// right-aligned, zero padded), "date" (YYYYMMDD), "filler" (spaces).
	Type  string `json:"type"`
	Value string `json:"value,omitempty"`
}

// RecordSpec lists the fields of one record kind, identified by the first
// byte of the record.
type RecordSpec struct {
	Kind   string  `json:"kind"`
	Fields []Field `json:"fields"`
}

// FileSpec is the declarative specification of a partner's fixed-length
// file. Lengths and offsets are bytes, never characters.
type FileSpec struct {
	Schema         string       `json:"schema"`
	ID             string       `json:"id"`
	Version        int          `json:"version"`
	Encoding       string       `json:"encoding"`
	RecordLength   int          `json:"recordLength"`
	LineTerminator string       `json:"lineTerminator"`
	Header         RecordSpec   `json:"header"`
	Data           RecordSpec   `json:"data"`
	Trailer        RecordSpec   `json:"trailer"`
	TrailerCount   string       `json:"trailerCountField"`
	TrailerSum     string       `json:"trailerSumField"`
	DataSumField   string       `json:"dataSumField"`
	FileSeqField   string       `json:"fileSeqField"`
	SenderField    string       `json:"senderField"`
	MonthField     string       `json:"monthField"`
	CompareEquivs  [][2]string  `json:"compareEquivalents"`
	EventMapping   EventMapping `json:"eventMapping"`
	encodingCached encoding.Encoding
}

// EventMapping names the data fields that form a shipment event.
type EventMapping struct {
	ShipmentID string `json:"shipmentId"`
	OrderNo    string `json:"orderNo"`
	ShipDate   string `json:"shipDate"`
	Qty        string `json:"qty"`
	Amount     string `json:"amount"`
	ItemName   string `json:"itemName"`
}

// ValidateSpec checks that each record's fields tile the record exactly.
func ValidateSpec(s *FileSpec) error {
	if s.Schema != FileSpecSchema {
		return fmt.Errorf("file spec schema %q, want %q", s.Schema, FileSpecSchema)
	}
	switch s.Encoding {
	case "windows-31j":
		// golang.org/x/text's ShiftJIS follows the WHATWG Shift_JIS index,
		// which is the Microsoft CP932 mapping (0x8160 is U+FF5E, the NEC and
		// IBM extensions are decoded). Duplicate codes are not round-trip safe;
		// the parser detects them per field.
		s.encodingCached = japanese.ShiftJIS
	default:
		return fmt.Errorf("encoding %q is not implemented", s.Encoding)
	}
	if s.RecordLength < 2 {
		return fmt.Errorf("record length %d", s.RecordLength)
	}
	for _, r := range []RecordSpec{s.Header, s.Data, s.Trailer} {
		if len(r.Kind) != 1 {
			return fmt.Errorf("record kind %q must be one byte", r.Kind)
		}
		pos := 0
		for _, f := range r.Fields {
			if f.Offset != pos || f.Length < 1 {
				return fmt.Errorf("record %s field %s at %d+%d: fields must tile the record without gaps", r.Kind, f.Name, f.Offset, f.Length)
			}
			switch f.Type {
			case "const", "alnum", "text", "number", "date", "filler":
			default:
				return fmt.Errorf("record %s field %s: unknown type %q", r.Kind, f.Name, f.Type)
			}
			pos += f.Length
		}
		if pos != s.RecordLength {
			return fmt.Errorf("record %s covers %d bytes, want %d", r.Kind, pos, s.RecordLength)
		}
	}
	return nil
}

// Record is one parsed record. Values holds the decoded field values; Raw is
// the record's exact bytes (without the line terminator).
type Record struct {
	Seq        int               `json:"seq"`
	Kind       string            `json:"kind"`
	Values     map[string]string `json:"values"`
	Raw        []byte            `json:"-"`
	HeldReason string            `json:"heldReason,omitempty"`
	HeldField  string            `json:"heldField,omitempty"`
}

// ParsedFile is the result of parsing a whole file.
type ParsedFile struct {
	SHA256  string
	Size    int
	Header  Record
	Data    []Record
	Trailer Record
}

// RejectError rejects a whole file. Nothing of a rejected file is applied.
type RejectError struct {
	Reason string
	Detail string
}

func (e *RejectError) Error() string { return "file rejected: " + e.Reason + ": " + e.Detail }

// Rejection reasons.
const (
	RejectTruncated     = "truncated"
	RejectMalformed     = "malformed-record"
	RejectTrailerCount  = "trailer-count-mismatch"
	RejectTrailerSum    = "trailer-sum-mismatch"
	RejectDuplicateFile = "duplicate-file"
	RejectSeqReused     = "file-sequence-reused"
	RejectMonth         = "month-mismatch"
	RejectAfterCutoff   = "received-after-cutoff"
	RejectDecision      = "decision-not-active"
)

// Hold reasons of a data record whose text cannot be stored as Unicode and
// written back byte for byte.
const (
	HoldUndecodable  = "undecodable-bytes"
	HoldNonRoundtrip = "non-roundtrip-code"
)

func reject(reason, format string, a ...any) error {
	return &RejectError{Reason: reason, Detail: fmt.Sprintf(format, a...)}
}

// IsReject reports whether err rejects the whole file, and why.
func IsReject(err error) (string, bool) {
	var r *RejectError
	if errors.As(err, &r) {
		return r.Reason, true
	}
	return "", false
}

// FileSHA256 is the identity hash of a file.
func FileSHA256(b []byte) string {
	sum := sha256.Sum256(b)
	return "sha256:" + hex.EncodeToString(sum[:])
}

// ParseFile parses a whole file by bytes. Any structural error rejects the
// whole file; undecodable or non-round-trip text holds only its record.
func (s *FileSpec) ParseFile(b []byte) (ParsedFile, error) {
	if s.encodingCached == nil {
		if err := ValidateSpec(s); err != nil {
			return ParsedFile{}, err
		}
	}
	pf := ParsedFile{SHA256: FileSHA256(b), Size: len(b)}
	term := []byte(s.LineTerminator)
	step := s.RecordLength + len(term)
	if len(b) == 0 || len(b)%step != 0 {
		return pf, reject(RejectTruncated, "%d bytes is not a whole number of %d-byte records", len(b), step)
	}
	n := len(b) / step
	var recs []Record
	for i := 0; i < n; i++ {
		line := b[i*step : i*step+s.RecordLength]
		if !bytes.Equal(b[i*step+s.RecordLength:(i+1)*step], term) {
			return pf, reject(RejectMalformed, "record %d: line terminator missing (record length is bytes, not characters)", i+1)
		}
		var spec RecordSpec
		switch string(line[:1]) {
		case s.Header.Kind:
			spec = s.Header
		case s.Data.Kind:
			spec = s.Data
		case s.Trailer.Kind:
			spec = s.Trailer
		default:
			return pf, reject(RejectMalformed, "record %d: unknown kind %q", i+1, line[:1])
		}
		rec := Record{Seq: i + 1, Kind: spec.Kind, Values: map[string]string{}, Raw: append([]byte(nil), line...)}
		for _, f := range spec.Fields {
			raw := line[f.Offset : f.Offset+f.Length]
			v, hold, err := s.decodeField(f, raw)
			if err != nil {
				return pf, reject(RejectMalformed, "record %d field %s: %v", i+1, f.Name, err)
			}
			if hold != "" && rec.HeldReason == "" {
				rec.HeldReason, rec.HeldField = hold, f.Name
			}
			if f.Type != "filler" && f.Type != "const" {
				rec.Values[f.Name] = v
			}
		}
		recs = append(recs, rec)
	}
	if recs[0].Kind != s.Header.Kind {
		return pf, reject(RejectMalformed, "first record is not a header")
	}
	last := recs[len(recs)-1]
	if last.Kind != s.Trailer.Kind {
		return pf, reject(RejectTruncated, "last record is %q, not the trailer", last.Kind)
	}
	pf.Header, pf.Trailer = recs[0], last
	var sum int64
	for _, r := range recs[1 : len(recs)-1] {
		if r.Kind != s.Data.Kind {
			return pf, reject(RejectMalformed, "record %d: %q between header and trailer", r.Seq, r.Kind)
		}
		v, _ := strconv.ParseInt(r.Values[s.DataSumField], 10, 64)
		sum += v
		pf.Data = append(pf.Data, r)
	}
	for _, r := range []Record{pf.Header, pf.Trailer} {
		if r.HeldReason != "" {
			return pf, reject(RejectMalformed, "record %d: %s in %s", r.Seq, r.HeldReason, r.HeldField)
		}
	}
	count, _ := strconv.Atoi(pf.Trailer.Values[s.TrailerCount])
	if count != len(pf.Data) {
		return pf, reject(RejectTrailerCount, "trailer says %d data records, file has %d", count, len(pf.Data))
	}
	tsum, _ := strconv.ParseInt(pf.Trailer.Values[s.TrailerSum], 10, 64)
	if tsum != sum {
		return pf, reject(RejectTrailerSum, "trailer sum %d, data sum %d", tsum, sum)
	}
	return pf, nil
}

func (s *FileSpec) decodeField(f Field, raw []byte) (value, hold string, err error) {
	switch f.Type {
	case "const":
		if string(raw) != f.Value {
			return "", "", fmt.Errorf("want %q, got %q", f.Value, raw)
		}
		return f.Value, "", nil
	case "filler":
		if len(bytes.Trim(raw, " ")) != 0 {
			return "", "", fmt.Errorf("filler is not spaces")
		}
		return "", "", nil
	case "number", "date":
		for _, c := range raw {
			if c < '0' || c > '9' {
				return "", "", fmt.Errorf("non-digit byte 0x%02X", c)
			}
		}
		if f.Type == "date" {
			if _, perr := parseCivil(string(raw[:4]) + "-" + string(raw[4:6]) + "-" + string(raw[6:8])); f.Length != 8 || perr != nil {
				return "", "", fmt.Errorf("invalid date %q", raw)
			}
			return string(raw), "", nil
		}
		v, _ := strconv.ParseInt(string(raw), 10, 64)
		return strconv.FormatInt(v, 10), "", nil
	case "alnum":
		for _, c := range raw {
			if c < 0x20 || c > 0x7E {
				return "", "", fmt.Errorf("non-ASCII byte 0x%02X in alnum field", c)
			}
		}
		return strings.TrimRight(string(raw), " "), "", nil
	}
	// text: decode the exact byte range. A double-byte character cut by the
	// field boundary leaves a lone lead byte, which the decoder reports as
	// U+FFFD; the record is held, never shifted.
	dec, derr := s.encodingCached.NewDecoder().Bytes(raw)
	if derr != nil || bytes.ContainsRune(dec, utf8.RuneError) || !utf8.Valid(dec) {
		return "", HoldUndecodable, nil
	}
	v := strings.TrimRight(string(dec), " ")
	back, eerr := s.encodeText(v, f.Length)
	if eerr != nil || !bytes.Equal(back, raw) {
		return v, HoldNonRoundtrip, nil
	}
	return v, "", nil
}

func (s *FileSpec) encodeText(v string, length int) ([]byte, error) {
	b, err := s.encodingCached.NewEncoder().Bytes([]byte(v))
	if err != nil {
		return nil, err
	}
	if len(b) > length {
		return nil, fmt.Errorf("%d bytes exceed field length %d", len(b), length)
	}
	return append(b, bytes.Repeat([]byte(" "), length-len(b))...), nil
}

// EncodeRecord writes a record from its decoded values. Held records are
// written from their raw bytes, so a file that holds records still re-emits
// byte for byte.
func (s *FileSpec) EncodeRecord(r Record) ([]byte, error) {
	if r.HeldReason != "" {
		return append([]byte(nil), r.Raw...), nil
	}
	var spec RecordSpec
	switch r.Kind {
	case s.Header.Kind:
		spec = s.Header
	case s.Data.Kind:
		spec = s.Data
	case s.Trailer.Kind:
		spec = s.Trailer
	default:
		return nil, fmt.Errorf("unknown kind %q", r.Kind)
	}
	var out []byte
	for _, f := range spec.Fields {
		v := r.Values[f.Name]
		switch f.Type {
		case "const":
			out = append(out, f.Value...)
		case "filler":
			out = append(out, bytes.Repeat([]byte(" "), f.Length)...)
		case "number":
			p := v
			if len(p) > f.Length {
				return nil, fmt.Errorf("%s: %q exceeds %d digits", f.Name, v, f.Length)
			}
			out = append(out, strings.Repeat("0", f.Length-len(p))+p...)
		case "date":
			out = append(out, v...)
		case "alnum":
			if len(v) > f.Length {
				return nil, fmt.Errorf("%s: too long", f.Name)
			}
			out = append(out, v+strings.Repeat(" ", f.Length-len(v))...)
		case "text":
			b, err := s.encodeText(v, f.Length)
			if err != nil {
				return nil, fmt.Errorf("%s: %w", f.Name, err)
			}
			out = append(out, b...)
		}
	}
	if len(out) != s.RecordLength {
		return nil, fmt.Errorf("encoded %d bytes, want %d", len(out), s.RecordLength)
	}
	return out, nil
}

// EmitFile re-emits a whole file from records.
func (s *FileSpec) EmitFile(recs []Record) ([]byte, error) {
	var out []byte
	for _, r := range recs {
		b, err := s.EncodeRecord(r)
		if err != nil {
			return nil, fmt.Errorf("record %d: %w", r.Seq, err)
		}
		out = append(append(out, b...), s.LineTerminator...)
	}
	return out, nil
}

// NormalizeForCompare maps each declared equivalent to its canonical form
// (the second element, the Windows-31J side), so the same text written by a
// JIS-mapping sender (U+301C) and a CP932 file (U+FF5E) compares equal.
func (s *FileSpec) NormalizeForCompare(v string) string {
	for _, p := range s.CompareEquivs {
		v = strings.ReplaceAll(v, p[0], p[1])
	}
	return v
}

// ReadFileSpec reads and validates a file specification.
func ReadFileSpec(path string) (*FileSpec, error) {
	var s FileSpec
	if err := readJSON(path, &s); err != nil {
		return nil, err
	}
	if err := ValidateSpec(&s); err != nil {
		return nil, err
	}
	return &s, nil
}

// SpecDigest identifies the specification a file was imported with.
func SpecDigest(s *FileSpec) string {
	c := *s
	c.encodingCached = nil
	return digestOf(c)
}
