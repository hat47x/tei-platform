package transform

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

// sourceSerializedSize checks the JSON size before encoding allocates a buffer.
// It mirrors the fields and omitempty rules of encoding/json for TEI's closed
// Source structs. It is only a resource gate: encoding/json remains the format
// authority, and the final bytes are checked and parsed separately.
func sourceSerializedSize(source Source, limit int) error {
	budget := sourceSizeBudget{
		remaining:    limit,
		rawRemaining: MaxSerializedSourceBytes,
		active:       make(map[uintptr]bool),
	}
	if err := budget.measure(reflect.ValueOf(source), 0); err != nil {
		return fmt.Errorf("transform Source exceeds serialized size limit or structural budget: %w", err)
	}
	return nil
}

type sourceSizeBudget struct {
	remaining    int
	rawRemaining int // cumulative unnormalized literal bytes
	nodes        int
	active       map[uintptr]bool
}

func (b *sourceSizeBudget) add(n int) error {
	if n < 0 || n > b.remaining {
		return fmt.Errorf("serialized size limit exceeded")
	}
	b.remaining -= n
	return nil
}

func (b *sourceSizeBudget) quoted(s string) error {
	n, err := jsonStringEncodedSize(s, b.remaining)
	if err != nil {
		return err
	}
	return b.add(n)
}

// rawJSON accounts for the largest possible canonical JSON representation
// before a literal or an allowed value is decoded. RawMessage may contain
// unescaped HTML-sensitive characters or U+2028/U+2029, which grow during
// canonical encoding. Count those bytes now rather than allocating the
// expanded canonical literal first.
func (b *sourceSizeBudget) rawJSON(raw []byte) error {
	// Compact JSON can be tiny even when the original RawMessage contains
	// megabytes of whitespace. Bound the input bytes separately from the
	// encoded output to prevent normalization from bypassing the budget.
	if len(raw) > b.rawRemaining {
		return fmt.Errorf("transform Source raw literal bytes exceed limit")
	}
	b.rawRemaining -= len(raw)
	size := 0
	inString := false
	escaped := false
	for i := 0; i < len(raw); i++ {
		ch := raw[i]
		// Whitespace outside strings disappears when RawMessage is
		// compacted. Counting it would reject ordinary valid input near
		// the byte boundary even though the canonical bytes fit.
		if !inString && (ch == ' ' || ch == '\n' || ch == '\r' || ch == '\t') {
			continue
		}
		// Normalize escapes that encode a Unicode scalar before sizing it.
		// In particular, a surrogate pair becomes one UTF-8 scalar in the
		// canonical JSON value, so counting the twelve source bytes would
		// reject a valid Source whose canonical artifact exactly fits.
		if inString && !escaped && ch == '\\' && i+1 < len(raw) {
			if raw[i+1] == '/' {
				// The JSON encoder emits an unescaped slash.
				size++
				i++
				if size > b.remaining {
					return fmt.Errorf("serialized size limit exceeded")
				}
				continue
			}
			if raw[i+1] == 'u' {
				if encodedBytes, consumed, ok := canonicalUnicodeEscapeSize(raw[i:]); ok {
					size += encodedBytes
					i += consumed - 1
					if size > b.remaining {
						return fmt.Errorf("serialized size limit exceeded")
					}
					continue
				}
			}
		}
		// The expansion estimate remains conservative for malformed JSON;
		// ParseJSON is responsible for rejecting invalid syntax.
		encodedBytes := 1
		switch ch {
		case '<', '>', '&':
			encodedBytes = 6
		case 0xe2:
			if i+2 < len(raw) && raw[i+1] == 0x80 &&
				(raw[i+2] == 0xa8 || raw[i+2] == 0xa9) {
				encodedBytes = 6
				i += 2
			}
		}
		size += encodedBytes
		if size > b.remaining {
			return fmt.Errorf("serialized size limit exceeded")
		}
		if inString {
			if escaped {
				escaped = false
			} else if ch == '\\' {
				escaped = true
			} else if ch == '"' {
				inString = false
			}
		} else {
			if ch == '"' {
				inString = true
			} else if ch == '{' || ch == '[' || ch == ',' {
				b.nodes++
				if b.nodes > 1_000_000 {
					return fmt.Errorf("source literal node count exceeds limit")
				}
			}
		}
	}
	return b.add(size)
}

// canonicalUnicodeEscapeSize returns the encoded size of one decoded JSON
// Unicode escape and how many source bytes it consumes. The JSON parser still
// validates the literal; invalid or unpaired escapes fall back to conservative
// byte counting and are rejected during normalization.
func canonicalUnicodeEscapeSize(raw []byte) (size, consumed int, ok bool) {
	if len(raw) < 6 || raw[0] != '\\' || raw[1] != 'u' {
		return 0, 0, false
	}
	first, valid := jsonHex16(raw[2:6])
	if !valid {
		return 0, 0, false
	}
	r := rune(first)
	consumed = 6
	switch {
	case first >= 0xd800 && first <= 0xdbff:
		if len(raw) < 12 || raw[6] != '\\' || raw[7] != 'u' {
			return 0, 0, false
		}
		second, valid := jsonHex16(raw[8:12])
		if !valid || second < 0xdc00 || second > 0xdfff {
			return 0, 0, false
		}
		r = 0x10000 + (rune(first)-0xd800)<<10 + (rune(second) - 0xdc00)
		consumed = 12
	case first >= 0xdc00 && first <= 0xdfff:
		return 0, 0, false
	}
	encoded, err := jsonStringEncodedSize(string(r), 8)
	if err != nil {
		return 0, 0, false
	}
	return encoded - 2, consumed, true
}

func (b *sourceSizeBudget) measure(v reflect.Value, depth int) error {
	if depth > 512 {
		return fmt.Errorf("source nesting is too deep")
	}
	b.nodes++
	if b.nodes > 1_000_000 {
		return fmt.Errorf("source node count exceeds limit")
	}
	if !v.IsValid() {
		return b.add(4)
	}

	// Literal and allowed values are already JSON, not base64 byte slices.
	if v.Type() == reflect.TypeOf(json.RawMessage(nil)) {
		if v.IsNil() {
			return b.add(4)
		}
		return b.rawJSON(v.Bytes())
	}

	switch v.Kind() {
	case reflect.Pointer:
		if v.IsNil() {
			return b.add(4)
		}
		p := v.Pointer()
		if b.active[p] {
			return fmt.Errorf("cyclic transform Source")
		}
		b.active[p] = true
		defer delete(b.active, p)
		return b.measure(v.Elem(), depth+1)
	case reflect.String:
		return b.quoted(v.String())
	case reflect.Bool:
		if v.Bool() {
			return b.add(4)
		}
		return b.add(5)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return b.add(len(strconv.FormatInt(v.Int(), 10)))
	case reflect.Slice, reflect.Array:
		if v.Kind() == reflect.Slice && v.IsNil() {
			return b.add(4)
		}
		if err := b.add(2); err != nil {
			return err
		}
		for i := 0; i < v.Len(); i++ {
			if i > 0 {
				if err := b.add(1); err != nil {
					return err
				}
			}
			if err := b.measure(v.Index(i), depth+1); err != nil {
				return err
			}
		}
		return nil
	case reflect.Struct:
		if err := b.add(2); err != nil {
			return err
		}
		first := true
		t := v.Type()
		for i := 0; i < v.NumField(); i++ {
			f := t.Field(i)
			if f.PkgPath != "" {
				continue
			}
			tag := strings.Split(f.Tag.Get("json"), ",")
			if tag[0] == "-" {
				continue
			}
			field := v.Field(i)
			if len(tag) > 1 && containsJSONOmitEmpty(tag[1:]) && sourceFieldEmpty(field) {
				continue
			}
			if !first {
				if err := b.add(1); err != nil {
					return err
				}
			}
			first = false
			name := tag[0]
			if name == "" {
				name = f.Name
			}
			if err := b.quoted(name); err != nil {
				return err
			}
			if err := b.add(1); err != nil {
				return err
			}
			if err := b.measure(field, depth+1); err != nil {
				return err
			}
		}
		return nil
	default:
		return fmt.Errorf("unsupported Source field type %s", v.Type())
	}
}

func containsJSONOmitEmpty(options []string) bool {
	for _, option := range options {
		if option == "omitempty" {
			return true
		}
	}
	return false
}

func sourceFieldEmpty(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.Array, reflect.Slice, reflect.Map, reflect.String:
		return v.Len() == 0
	case reflect.Bool:
		return !v.Bool()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int() == 0
	case reflect.Interface, reflect.Pointer:
		return v.IsNil()
	default:
		return false
	}
}
