package transform

import "fmt"

// rejectUnpairedSurrogateEscapes is a narrow strictness check before the
// standard JSON decoder. encoding/json replaces unmatched UTF-16 surrogate
// escapes with U+FFFD, which would silently change Source identity and
// literal fingerprints. The JSON decoder still validates the full grammar.
func rejectUnpairedSurrogateEscapes(raw []byte) error {
	inString := false
	for i := 0; i < len(raw); i++ {
		switch raw[i] {
		case '"':
			inString = !inString
		case '\\':
			if !inString || i+1 >= len(raw) {
				continue
			}
			if raw[i+1] != 'u' {
				i++
				continue
			}
			if i+6 > len(raw) {
				continue // The JSON decoder reports the malformed escape.
			}
			r, ok := fourJSONHexDigits(raw[i+2 : i+6])
			if !ok {
				continue // The JSON decoder reports the malformed escape.
			}
			if r >= 0xdc00 && r <= 0xdfff {
				return fmt.Errorf("JSON contains an unpaired low UTF-16 surrogate escape")
			}
			if r >= 0xd800 && r <= 0xdbff {
				if i+12 > len(raw) || raw[i+6] != '\\' || raw[i+7] != 'u' {
					return fmt.Errorf("JSON contains an unpaired high UTF-16 surrogate escape")
				}
				low, ok := fourJSONHexDigits(raw[i+8 : i+12])
				if !ok || low < 0xdc00 || low > 0xdfff {
					return fmt.Errorf("JSON contains an unpaired high UTF-16 surrogate escape")
				}
				i += 11
			} else {
				i += 5
			}
		}
	}
	return nil
}

func fourJSONHexDigits(raw []byte) (uint16, bool) {
	if len(raw) != 4 {
		return 0, false
	}
	var value uint16
	for _, ch := range raw {
		value <<= 4
		switch {
		case ch >= '0' && ch <= '9':
			value |= uint16(ch - '0')
		case ch >= 'a' && ch <= 'f':
			value |= uint16(ch-'a') + 10
		case ch >= 'A' && ch <= 'F':
			value |= uint16(ch-'A') + 10
		default:
			return 0, false
		}
	}
	return value, true
}
