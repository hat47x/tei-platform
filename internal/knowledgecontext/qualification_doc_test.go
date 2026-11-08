package knowledgecontext

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

const qualificationGuide = "../../docs/business-system-design-reading-guide.md"

func qualificationSection(t *testing.T) string {
	t.Helper()
	body, err := os.ReadFile(qualificationGuide)
	if err != nil {
		t.Fatalf("read %s: %v", qualificationGuide, err)
	}
	text := string(body)
	start := strings.Index(text, "-qualification-requirements")
	if start < 0 {
		t.Fatalf("%s no longer mentions -qualification-requirements", qualificationGuide)
	}
	end := strings.Index(text[start:], "引き継ぎには知識の本文を含めません")
	if end < 0 {
		t.Fatalf("the qualification section lost its closing paragraph")
	}
	return text[start : start+end]
}

// The reading guide tells a caller which file to copy. That file must decode
// with the same strict decoder the CLI uses, or the example is a trap (#3186).
func TestDocumentedQualificationExampleFileDecodes(t *testing.T) {
	section := qualificationSection(t)
	// The guide links the example as `[label](../reference/...json)`, and the
	// link target is relative to docs/. The test lives two levels below the
	// repository root, so drop the leading "../" the guide uses.
	links := regexp.MustCompile("\\]\\(([^)]+\\.json)\\)").FindAllStringSubmatch(section, -1)
	if len(links) == 0 {
		t.Fatalf("the qualification section names no .json example file")
	}
	decoded := 0
	for _, match := range links {
		relative := strings.TrimPrefix(match[1], "../")
		path := filepath.Join("..", "..", relative)
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("the guide links %s but %s cannot be read: %v", match[1], path, err)
		}
		var got QualificationRequirements
		decoder := json.NewDecoder(strings.NewReader(string(raw)))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&got); err != nil {
			t.Fatalf("%s does not decode as QualificationRequirements: %v", match[1], err)
		}
		if _, err := NormalizeQualificationRequirements(got); err != nil {
			t.Fatalf("%s is rejected by NormalizeQualificationRequirements: %v", match[1], err)
		}
		decoded++
	}
	if decoded == 0 {
		t.Fatal("no example file was checked")
	}
}

// The rejection reasons the parser produces must be named in the guide, so a
// caller can match a failure to the rule instead of guessing (#3186).
func TestDocumentedQualificationNotesRejectedCombinations(t *testing.T) {
	section := qualificationSection(t)
	for _, want := range []string{
		"expectedKind=application-pattern",
		"json: unknown field",
		"KNOWLEDGE_BUILD_FAILED",
	} {
		if !strings.Contains(section, want) {
			t.Fatalf("the guide does not state %q", want)
		}
	}
}

// The documented minimum shape must actually qualify: `expectedKind` alone is
// the smallest accepted file, and the guide says so.
func TestDocumentedMinimumShapeIsAccepted(t *testing.T) {
	section := qualificationSection(t)
	if !strings.Contains(section, `{"expectedKind":"semantic-data"}`) {
		t.Fatalf("the guide no longer names the minimum shape")
	}
	var got QualificationRequirements
	if err := json.Unmarshal([]byte(`{"expectedKind":"semantic-data"}`), &got); err != nil {
		t.Fatal(err)
	}
	if _, err := NormalizeQualificationRequirements(got); err != nil {
		t.Fatalf("the documented minimum shape is rejected: %v", err)
	}
}
