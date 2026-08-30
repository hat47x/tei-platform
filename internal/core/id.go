package core

import (
	"fmt"
	"regexp"
	"strings"
)

type CanonicalID string
type LocalName string

type Kind string

const (
	KindSemantic    Kind = "semantic"
	KindApplication Kind = "application"
	KindDatabase    Kind = "database"
	KindContract    Kind = "database-contract"
	KindEnvironment Kind = "environment"
	KindCapability  Kind = "capability"
)

var canonicalIDPattern = regexp.MustCompile(`^[a-z][a-z0-9_]*(\.[a-z][a-z0-9_]*)*$`)
var localNamePattern = regexp.MustCompile(`^[a-z][a-z0-9_]*$`)

func ParseCanonicalID(s string) (CanonicalID, error) {
	s = strings.TrimSpace(s)
	if !canonicalIDPattern.MatchString(s) {
		return "", fmt.Errorf("invalid canonical id %q", s)
	}
	return CanonicalID(s), nil
}

func MustCanonicalID(s string) CanonicalID {
	id, err := ParseCanonicalID(s)
	if err != nil {
		panic(err)
	}
	return id
}

func ParseLocalName(s string) (LocalName, error) {
	s = strings.TrimSpace(s)
	if !localNamePattern.MatchString(s) {
		return "", fmt.Errorf("invalid local name %q", s)
	}
	return LocalName(s), nil
}

func MustLocalName(s string) LocalName {
	n, err := ParseLocalName(s)
	if err != nil {
		panic(err)
	}
	return n
}

type Ref struct {
	Kind Kind
	ID   CanonicalID
}

func (r Ref) String() string {
	return fmt.Sprintf("%s:%s", r.Kind, r.ID)
}

// SymbolToSegment converts a human-facing TEI symbol such as OrderStatus or
// orderId to a canonical snake_case ID segment. Canonical identity remains a
// separate concern; this is only the default derivation used by the M0 parser.
func SymbolToSegment(symbol string) (string, error) {
	symbol = strings.TrimSpace(symbol)
	if symbol == "" {
		return "", fmt.Errorf("empty symbol")
	}
	var b strings.Builder
	var prevLowerOrDigit bool
	var prevUpper bool
	runes := []rune(symbol)
	for i, r := range runes {
		if r == '-' || r == ' ' {
			if b.Len() > 0 && !strings.HasSuffix(b.String(), "_") {
				b.WriteByte('_')
			}
			prevLowerOrDigit = false
			prevUpper = false
			continue
		}
		if r == '_' {
			if b.Len() > 0 && !strings.HasSuffix(b.String(), "_") {
				b.WriteByte('_')
			}
			prevLowerOrDigit = false
			prevUpper = false
			continue
		}
		if r >= 'A' && r <= 'Z' {
			nextLower := i+1 < len(runes) && runes[i+1] >= 'a' && runes[i+1] <= 'z'
			if b.Len() > 0 && (prevLowerOrDigit || (prevUpper && nextLower)) && !strings.HasSuffix(b.String(), "_") {
				b.WriteByte('_')
			}
			b.WriteRune(r + ('a' - 'A'))
			prevLowerOrDigit = false
			prevUpper = true
			continue
		}
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			prevLowerOrDigit = true
			prevUpper = false
			continue
		}
		return "", fmt.Errorf("invalid symbol %q", symbol)
	}
	seg := strings.Trim(b.String(), "_")
	if !localNamePattern.MatchString(seg) {
		return "", fmt.Errorf("symbol %q cannot derive canonical segment", symbol)
	}
	return seg, nil
}

func MustSymbolSegment(symbol string) string {
	s, err := SymbolToSegment(symbol)
	if err != nil {
		panic(err)
	}
	return s
}
