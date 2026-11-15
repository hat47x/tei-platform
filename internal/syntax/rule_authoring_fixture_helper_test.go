package syntax_test

import (
	"fmt"
	"strings"

	"tei/internal/semantic"
	"tei/internal/syntax"
	"testing"
)

// ruleSurfaceValues resolves either surface of the fixture to the same typed
// value. The dispatch is by the first declaration keyword, which is what
// distinguishes the accepted grammar from the candidate one; it is a test
// helper, not a product selector.
func ruleSurfaceValues(t *testing.T, source string) ([]semantic.Validation, error) {
	t.Helper()
	trimmed := strings.TrimSpace(source)
	switch {
	case strings.HasPrefix(trimmed, "namespace"):
		body := strings.TrimPrefix(trimmed, "namespace")
		body = strings.TrimLeft(body, " \t\r\n")
		if strings.HasPrefix(body, "rule") || strings.Contains(body, "\nrule ") {
			file, err := syntax.ParseRuleAuthoringFile(source)
			if err != nil {
				return nil, err
			}
			ns, issues := syntax.RuleNamespace(file)
			if len(issues) != 0 {
				return nil, fmt.Errorf("conversion issues: %v", issues)
			}
			return ns.Validations, nil
		}
		ns, err := syntax.NewParser(source).ParseSemanticFile()
		if err != nil {
			return nil, err
		}
		return ns.Validations, nil
	case strings.HasPrefix(trimmed, "|"), strings.HasPrefix(trimmed, "check"):
		return nil, fmt.Errorf("surface is recorded in the fixture but not implemented")
	default:
		return nil, fmt.Errorf("unknown surface: %q", firstLine(trimmed))
	}
}

func firstLine(s string) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return s[:i]
	}
	return s
}
