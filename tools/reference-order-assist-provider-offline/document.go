package orderassistoffline

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
)

// The quote document is kept as raw JSON members so that members a reader
// does not know survive a read-modify-write (P-MIX-03). Field paths name the
// unit of synchronisation and conflict detection (P-MIX-02):
//
//	discountRateBp
//	lines/<sku>/<member>
//
// Each generation knows a fixed set of members. A member introduced by a
// later generation is optional and must be preserved by older readers.
var (
	topMembers = map[string]int{ // member -> generation that introduced it
		"quoteId": 1, "customerCode": 1, "currency": 1, "lines": 1,
		"discountRateBp": 2,
	}
	lineMembers = map[string]int{
		"sku": 1, "quantity": 1, "unitPriceMinor": 1, "priceListVersion": 1,
		"packaging": 2,
	}
)

// KnownToGeneration reports whether a field path's member exists in gen.
func KnownToGeneration(path string, gen int) bool {
	if strings.HasPrefix(path, "lines/") {
		parts := strings.Split(path, "/")
		if len(parts) != 3 {
			return false
		}
		g, ok := lineMembers[parts[2]]
		return ok && g <= gen
	}
	g, ok := topMembers[path]
	return ok && g <= gen
}

// Doc is a quote document with raw members.
type Doc struct {
	Top   map[string]json.RawMessage
	Lines []map[string]json.RawMessage
}

// ErrDocument is a malformed quote document.
var ErrDocument = errors.New("malformed quote document")

// ParseDoc decodes a document, keeping every member's raw bytes.
func ParseDoc(raw []byte) (Doc, error) {
	var d Doc
	if err := json.Unmarshal(raw, &d.Top); err != nil || d.Top == nil {
		return Doc{}, fmt.Errorf("%w: top level: %v", ErrDocument, err)
	}
	if rawLines, ok := d.Top["lines"]; ok {
		if err := json.Unmarshal(rawLines, &d.Lines); err != nil {
			return Doc{}, fmt.Errorf("%w: lines: %v", ErrDocument, err)
		}
	}
	seen := map[string]bool{}
	for _, l := range d.Lines {
		sku := lineSKU(l)
		if sku == "" || seen[sku] {
			return Doc{}, fmt.Errorf("%w: missing or duplicate line SKU %q", ErrDocument, sku)
		}
		seen[sku] = true
	}
	return d, nil
}

func lineSKU(l map[string]json.RawMessage) string {
	var s string
	_ = json.Unmarshal(l["sku"], &s)
	return s
}

// Marshal encodes the document. Member order is canonical (sorted).
func (d Doc) Marshal() []byte {
	top := make(map[string]json.RawMessage, len(d.Top))
	for k, v := range d.Top {
		top[k] = v
	}
	lines := []byte("[]")
	if len(d.Lines) > 0 {
		lines, _ = json.Marshal(d.Lines)
	}
	top["lines"] = lines
	raw, _ := json.Marshal(top)
	return raw
}

// Clone deep-copies the member maps.
func (d Doc) Clone() Doc {
	out, _ := ParseDoc(d.Marshal())
	return out
}

// Line returns the line with the SKU.
func (d Doc) Line(sku string) (map[string]json.RawMessage, bool) {
	for _, l := range d.Lines {
		if lineSKU(l) == sku {
			return l, true
		}
	}
	return nil, false
}

// Get returns the raw value at a field path.
func (d Doc) Get(path string) (json.RawMessage, bool) {
	if strings.HasPrefix(path, "lines/") {
		parts := strings.Split(path, "/")
		if len(parts) != 3 {
			return nil, false
		}
		l, ok := d.Line(parts[1])
		if !ok {
			return nil, false
		}
		v, ok := l[parts[2]]
		return v, ok
	}
	v, ok := d.Top[path]
	return v, ok
}

// Set writes the raw value at a field path. A nil value removes the member.
func (d *Doc) Set(path string, value json.RawMessage) error {
	if strings.HasPrefix(path, "lines/") {
		parts := strings.Split(path, "/")
		if len(parts) != 3 || parts[2] == "sku" {
			return fmt.Errorf("%w: path %q", ErrDocument, path)
		}
		l, ok := d.Line(parts[1])
		if !ok {
			return fmt.Errorf("%w: no line %q", ErrDocument, parts[1])
		}
		if value == nil {
			delete(l, parts[2])
		} else {
			l[parts[2]] = value
		}
		return nil
	}
	if path == "lines" || path == "quoteId" {
		return fmt.Errorf("%w: path %q is not settable", ErrDocument, path)
	}
	if value == nil {
		delete(d.Top, path)
	} else {
		d.Top[path] = value
	}
	return nil
}

// Paths lists every field path of the document except identity members.
func (d Doc) Paths() []string {
	var out []string
	for k := range d.Top {
		if k != "lines" && k != "quoteId" {
			out = append(out, k)
		}
	}
	for _, l := range d.Lines {
		sku := lineSKU(l)
		for k := range l {
			if k != "sku" {
				out = append(out, "lines/"+sku+"/"+k)
			}
		}
	}
	sort.Strings(out)
	return out
}

// EqualJSON compares two raw JSON values semantically.
func EqualJSON(a, b json.RawMessage) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	ca, errA := canonicalJSON(a)
	cb, errB := canonicalJSON(b)
	return errA == nil && errB == nil && bytes.Equal(ca, cb)
}

// canonicalJSON re-encodes a value with sorted object members and exact
// number tokens.
func canonicalJSON(raw []byte) ([]byte, error) {
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	var v any
	if err := dec.Decode(&v); err != nil {
		return nil, err
	}
	return json.Marshal(v)
}

func intValue(raw json.RawMessage) (int64, bool) {
	if raw == nil {
		return 0, false
	}
	var n json.Number
	if err := json.Unmarshal(raw, &n); err != nil {
		return 0, false
	}
	v, err := n.Int64()
	return v, err == nil
}

func rawInt(v int64) json.RawMessage { return json.RawMessage(fmt.Sprintf("%d", v)) }

// QuoteRequest builds the business request for a realization generation.
// Generation 1 cannot express a discount, so the member is omitted.
func (d Doc) QuoteRequest(gen int) ([]byte, error) {
	type line struct {
		SKU            string `json:"sku"`
		Quantity       int64  `json:"quantity"`
		UnitPriceMinor int64  `json:"unitPriceMinor"`
	}
	var currency string
	if err := json.Unmarshal(d.Top["currency"], &currency); err != nil {
		return nil, fmt.Errorf("%w: currency", ErrDocument)
	}
	lines := make([]line, 0, len(d.Lines))
	for _, l := range d.Lines {
		q, ok1 := intValue(l["quantity"])
		p, ok2 := intValue(l["unitPriceMinor"])
		if !ok1 || !ok2 {
			return nil, fmt.Errorf("%w: line %q quantity or price", ErrDocument, lineSKU(l))
		}
		lines = append(lines, line{lineSKU(l), q, p})
	}
	req := map[string]any{"currency": currency, "lines": lines}
	if gen >= 2 {
		if bp, ok := intValue(d.Top["discountRateBp"]); ok {
			req["discountRateBp"] = bp
		}
	}
	return json.Marshal(req)
}

// QuoteDigest is the identity carried from synchronisation to approval and
// credit: the canonical document (unknown members included), the decided
// result and the rule version, and the price list it was validated against.
func QuoteDigest(d Doc, j Judgment, priceListVersion int) string {
	doc, _ := canonicalJSON(d.Marshal())
	body, _ := json.Marshal(map[string]any{
		"schema":           "tei.reference.order-assist.quote-digest/v0",
		"document":         json.RawMessage(doc),
		"ruleVersion":      j.RuleVersion,
		"result":           j.Result,
		"priceListVersion": priceListVersion,
	})
	return digestBytes(body)
}
