package transform

import "fmt"

// rejectUnpairedJSONSurrogates checks the raw spelling of JSON string escapes.
// encoding/json replaces unpaired UTF-16 surrogates with U+FFFD, which would
// otherwise make distinct, malformed Source documents share one fingerprint.
// The JSON decoder remains responsible for all other syntax and value checks.
func rejectUnpairedJSONSurrogates(raw []byte) error {
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
				i++ // A non-Unicode escape also consumes the following byte.
				continue
			}
			if i+6 > len(raw) {
				return fmt.Errorf("JSON contains an incomplete Unicode escape")
			}
			first, ok := jsonHex16(raw[i+2 : i+6])
			if !ok {
				return fmt.Errorf("JSON contains an invalid Unicode escape")
			}
			switch {
			case first >= 0xd800 && first <= 0xdbff:
				if i+12 > len(raw) || raw[i+6] != '\\' || raw[i+7] != 'u' {
					return fmt.Errorf("JSON contains an unpaired Unicode surrogate escape")
				}
				second, ok := jsonHex16(raw[i+8 : i+12])
				if !ok || second < 0xdc00 || second > 0xdfff {
					return fmt.Errorf("JSON contains an unpaired Unicode surrogate escape")
				}
				i += 11 // Both \\uXXXX escapes form one Unicode scalar.
			case first >= 0xdc00 && first <= 0xdfff:
				return fmt.Errorf("JSON contains an unpaired Unicode surrogate escape")
			default:
				i += 5
			}
		}
	}
	return nil
}

func jsonHex16(raw []byte) (uint16, bool) {
	if len(raw) != 4 {
		return 0, false
	}
	var value uint16
	for _, b := range raw {
		value <<= 4
		switch {
		case b >= '0' && b <= '9':
			value |= uint16(b - '0')
		case b >= 'a' && b <= 'f':
			value |= uint16(b-'a') + 10
		case b >= 'A' && b <= 'F':
			value |= uint16(b-'A') + 10
		default:
			return 0, false
		}
	}
	return value, true
}
