package knowledgecontext

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// The reading guide is where a caller learns the -qualification-requirements
// shape. The struct is the authority, so a documented example that the parser
// would reject is a documentation defect (#3186). This test extracts the JSON
// blocks from the guide and round-trips each through the real decoder.
func documentedQualificationExamples(t *testing.T) []string {
	t.Helper()
	guide := filepath.Join("..", "..", "docs", "business-system-design-reading-guide.md")
	body, err := os.ReadFile(guide)
	if err != nil {
		t.Fatalf("read %s: %v", guide, err)
	}
	text := string(body)
	// Only the section that documents the requirements file.
	start := strings.Index(text, "-qualification-requirements")
	if start < 0 {
		t.Fatalf("%s no longer mentions -qualification-requirements", guide)
	}
	end := strings.Index(text[start:], "引き継ぎには知識の本文を含めません")
	if end < 0 {
		t.Fatalf("the qualification section lost its closing paragraph")
	}
	section := text[start : start+end]
	blocks := regexp.MustCompile("(?s)```json\\n(.*?)```").FindAllStringSubmatch(section, -1)
	if len(blocks) == 0 {
		t.Fatalf("the qualification section has no json example")
	}
	examples := make([]string, 0, len(blocks))
	for _, match := range blocks {
		examples = append(examples, match[1])
	}
	return examples
}

func TestDocumentedQualificationExamplesDecode(t *testing.T) {
	for i, example := range documentedQualificationExamples(t) {
		var got QualificationRequirements
		decoder := json.NewDecoder(strings.NewReader(example))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&got); err != nil {
			t.Fatalf("example %d does not decode: %v\n%s", i, err, example)
		}
		if _, err := NormalizeQualificationRequirements(got); err != nil {
			t.Fatalf("example %d is rejected by NormalizeQualificationRequirements: %v\n%s", i, err, example)
		}
	}
}

// The document must also state the rule the parser enforces for `application`,
// so the rejected combination is not something a reader has to discover.
func TestDocumentedQualificationNotesRejectedCombination(t *testing.T) {
	guide := filepath.Join("..", "..", "docs", "business-system-design-reading-guide.md")
	body, err := os.ReadFile(guide)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"qualification expectedKind is required",
		"requires expectedKind=application-pattern",
		"json: unknown field",
	} {
		if !strings.Contains(string(body), want) {
			t.Fatalf("the guide does not state %q", want)
		}
	}
}
