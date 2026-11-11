package cswprobe

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The probe transcribes four attribution labels and one sentence each from
// skills/cultural-substrate-weaving/SKILL.md. Like the VS Bear Simulator
// anchors, these literals must be re-checkable against the file they came
// from: a self-consistent probe can still disagree with the upstream text.
type upstreamAnchor struct {
	file    string
	literal string
	why     string
}

// cswAttributionAnchors ties each declared value to the SKILL.md sentence that
// states the provenance distinction it carries. The sentence is quoted
// verbatim; a reworded upstream sentence fails the test on purpose, because a
// reworded sentence is a changed distinction and a human must re-decide
// whether the value still maps to it.
var cswAttributionAnchors = []upstreamAnchor{
	{"skills/cultural-substrate-weaving/SKILL.md",
		"原稿や資料にすでにあること。",
		"source_material"},
	{"skills/cultural-substrate-weaving/SKILL.md",
		"体系から生まれた問い・仮説・構成案。",
		"generated_proposal"},
	{"skills/cultural-substrate-weaving/SKILL.md",
		"原稿と体系の接触から新しく生まれた読み。",
		"contact_reading"},
	{"skills/cultural-substrate-weaving/SKILL.md",
		"著者が明示した方針や判断と、生成AIが新しく提示した案。",
		"author_stated"},
}

// cswSectionAnchors tie the enclosing section heading to the probe, so a
// upstream reorganisation that moves these sentences elsewhere is visible.
var cswSectionAnchors = []upstreamAnchor{
	{"skills/cultural-substrate-weaving/SKILL.md",
		"生成過程の由来を保持する",
		"section heading that groups the four distinctions"},
	{"skills/cultural-substrate-weaving/SKILL.md",
		"全候補へのラベルや一覧表は要らない",
		"the sentence that keeps labelling optional; a mandatory label would contradict it"},
}

// TestCSWAttributionAnchorsResolveInSKILLMD re-checks the transcription
// against a CSW checkout when one is reachable. When it is not, the test
// skips: an unreachable checkout is not evidence that the literals still
// match.
func TestCSWAttributionAnchorsResolveInSKILLMD(t *testing.T) {
	root := os.Getenv("CSW_REPO")
	if root == "" {
		root = filepath.Join("..", "..", "..", "..", "cultural-substrate-weaving")
	}
	info, err := os.Stat(root)
	if err != nil || !info.IsDir() {
		t.Skipf("Cultural Substrate Weaving checkout not reachable at %s; set CSW_REPO to re-check the transcription", root)
	}

	cache := map[string]string{}
	read := func(rel string) string {
		if text, ok := cache[rel]; ok {
			return text
		}
		data, readErr := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
		if readErr != nil {
			t.Fatalf("read upstream %s: %v", rel, readErr)
		}
		cache[rel] = string(data)
		return cache[rel]
	}

	for _, anchor := range append(append([]upstreamAnchor(nil), cswAttributionAnchors...), cswSectionAnchors...) {
		if !strings.Contains(read(anchor.file), anchor.literal) {
			t.Fatalf("%s no longer contains %q (%s); the probe transcription is stale",
				anchor.file, anchor.literal, anchor.why)
		}
	}
}

// TestCSWAttributionAnchorsMatchDeclaredValues keeps the anchor table and the
// accepted value set in step. An anchor whose value the probe no longer accepts
// would check a label that cannot be recorded.
func TestCSWAttributionAnchorsMatchDeclaredValues(t *testing.T) {
	if len(cswAttributionAnchors) != 4 {
		t.Fatalf("anchors = %d want 4, one per SKILL.md distinction", len(cswAttributionAnchors))
	}
	for _, anchor := range cswAttributionAnchors {
		if !knownAttribution(anchor.why) {
			t.Fatalf("anchor names %q (%s), which the probe does not accept", anchor.why, anchor.literal)
		}
	}
}
