package realization

import (
	"strings"
	"unicode"

	"tei/internal/core"
)

type CustomGoBinding struct {
	Application core.CanonicalID `json:"application,omitempty"`
	Capability  core.CanonicalID `json:"capability"`
}

func (b CustomGoBinding) ExpectedAdapterFilename() string {
	return strings.ReplaceAll(string(b.Capability), ".", "__") + ".bind.go"
}
func (b CustomGoBinding) ExpectedHookName() string {
	parts := strings.FieldsFunc(string(b.Capability), func(r rune) bool { return r == '.' || r == '_' || r == '-' })
	var out strings.Builder
	out.WriteString("TeiBind")
	for _, p := range parts {
		if p == "" {
			continue
		}
		rr := []rune(p)
		rr[0] = unicode.ToUpper(rr[0])
		out.WriteString(string(rr))
	}
	return out.String()
}
