package plancheck

import (
    "os"
    "path/filepath"
    "strconv"
    "strings"
    "testing"
)

// A valid UTF-8 file can still lose its Japanese title or leave duplicate
// numbered headings after a concurrent merge. Keep the research plan
// readable and its cumulative section history unambiguous.
func TestResearchIntegrationPlanReadableAndSequential(t *testing.T) {
    path := filepath.Join("..", "..", "plan", "research", "TECH_RESEARCH_INTEGRATION_2026-10.md")
    raw, err := os.ReadFile(path)
    if err != nil {
        t.Fatal(err)
    }
    if err := readableMarkdownBytes(raw); err != nil {
        t.Fatalf("research integration plan is not readable UTF-8: %v", err)
    }
    body := string(raw)
    if !strings.HasPrefix(body, "# TEIの技術調査を設計へ反映する方針 — 2026年10月\n") {
        t.Fatal("the Japanese research-plan title was altered or damaged")
    }

    nextSection := 1
    for _, line := range strings.Split(body, "\n") {
        if !strings.HasPrefix(line, "### 16.") {
            continue
        }
        number, _, hasTitle := strings.Cut(strings.TrimPrefix(line, "### 16."), " ")
        section, err := strconv.Atoi(number)
        if !hasTitle || err != nil || section != nextSection {
            t.Fatalf("section sequence is missing, duplicated or out of order at %q: want 16.%d", line, nextSection)
        }
        nextSection++
    }
    if nextSection <= 40 {
        t.Fatalf("recent research integration sections are missing: next is 16.%d", nextSection)
    }
}
