package management

import (
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
	"testing"
)

// declaredReasonCodes reads the reason constants from this package's own
// source. The reachability test in vocabulary_coverage_test.go names the codes
// it expects in a hand-written list, and that list had already drifted: 18
// codes are declared, 16 were listed. Reading the declarations makes the list
// unable to lag the vocabulary, which is the property the reachability test
// claims to have and did not.
func declaredReasonCodes(t *testing.T) map[string]string {
	t.Helper()
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "vocabulary.go", nil, 0)
	if err != nil {
		t.Fatalf("parse vocabulary.go: %v", err)
	}
	codes := map[string]string{}
	for _, decl := range file.Decls {
		gen, ok := decl.(*ast.GenDecl)
		if !ok || gen.Tok != token.CONST {
			continue
		}
		for _, spec := range gen.Specs {
			value, ok := spec.(*ast.ValueSpec)
			if !ok {
				continue
			}
			for i, name := range value.Names {
				if !strings.HasPrefix(name.Name, "Reason") || i >= len(value.Values) {
					continue
				}
				lit, ok := value.Values[i].(*ast.BasicLit)
				if !ok || lit.Kind != token.STRING {
					continue
				}
				codes[name.Name] = strings.Trim(lit.Value, `"`)
			}
		}
	}
	if len(codes) == 0 {
		t.Fatal("no reason codes were found; the declarations moved")
	}
	return codes
}

// The reason vocabulary is closed: every declared code has a message, and the
// message table names no code the constants do not declare. A code with no
// message would surface as an empty string to a reader, and a message for an
// undeclared code is dead.
func TestEveryReasonCodeHasAMessageAndNothingExtra(t *testing.T) {
	declared := declaredReasonCodes(t)
	table := messages()
	for name, code := range declared {
		text, ok := table[code]
		if !ok {
			t.Errorf("%s (%q) has no message; a row naming it would show an empty string", name, code)
			continue
		}
		if strings.TrimSpace(text) == "" {
			t.Errorf("%s (%q) has an empty message", name, code)
		}
	}
	if len(table) != len(declared) {
		for code := range table {
			if !declaresCode(declared, code) {
				t.Errorf("the message table names %q, which no constant declares", code)
			}
		}
	}
}

// declaresCode reports whether any constant in the vocabulary holds code. The
// vocabulary is a name-to-code map, so this is the reverse lookup the message
// table needs.
func declaresCode(declared map[string]string, code string) bool {
	for _, declaredCode := range declared {
		if declaredCode == code {
			return true
		}
	}
	return false
}

// The codes are kebab-case words, which is what a reader branches on. Two codes
// that differ only in punctuation would be a trap.
func TestReasonCodesAreDistinctKebabCaseWords(t *testing.T) {
	declared := declaredReasonCodes(t)
	seen := map[string]string{}
	for name, code := range declared {
		if code != strings.ToLower(code) {
			t.Errorf("%s = %q is not lower case", name, code)
		}
		if strings.HasPrefix(code, "-") || strings.HasSuffix(code, "-") {
			t.Errorf("%s = %q starts or ends with a separator", name, code)
		}
		if strings.ContainsAny(code, " _") {
			t.Errorf("%s = %q is not kebab-case", name, code)
		}
		if other, duplicate := seen[code]; duplicate {
			t.Errorf("%s and %s both declare %q", name, other, code)
		}
		seen[code] = name
	}
}

// The count is pinned so a code added without a message or a producing input
// is a deliberate edit to this number, not an accident.
func TestTheReasonVocabularyHasTheExpectedSize(t *testing.T) {
	if got := len(declaredReasonCodes(t)); got != 18 {
		t.Fatalf("the reason vocabulary has %d codes; add the producing input and a message for a new one", got)
	}
}
