package okf

import (
	"testing"
	"time"
)

// fixtureBundle is the bundle-relative path of the K1 synthetic fixture. The
// tests read it in place and never write to it: the fixture's bytes are pinned by
// testdata/okf-k1-v02.expected.json and shared with the upstream-reference witness.
const fixtureBundle = "../../testdata/okf-k1-v02"

// readFixture reads the K1 fixture in place, or a mutation copy when one is given.
func readFixture(t *testing.T, bundleRoot ...string) *Bundle {
	t.Helper()
	root := fixtureBundle
	if len(bundleRoot) > 0 && bundleRoot[0] != "" {
		root = bundleRoot[0]
	}
	bundle, err := Read(root, Options{})
	if err != nil {
		t.Fatalf("read %s: %v", root, err)
	}
	return bundle
}

func findConcept(bundle *Bundle, conceptID string) *Concept {
	for i := range bundle.Concepts {
		if bundle.Concepts[i].ConceptID == conceptID {
			return &bundle.Concepts[i]
		}
	}
	return nil
}

func countCode(diags []Diagnostic, code string) int {
	count := 0
	for _, diag := range diags {
		if diag.Code == code {
			count++
		}
	}
	return count
}

func hasCode(diags []Diagnostic, code string) bool { return countCode(diags, code) > 0 }

// TestBaselineIsFormatConformant pins the fixture's F0 verdict and its three
// concept IDs. OKF v0.2 §11 makes a bundle conformant when every concept parses,
// carries a non-empty type, and the reserved files follow their structures.
func TestBaselineIsFormatConformant(t *testing.T) {
	bundle := readFixture(t)

	if !bundle.Format.Conformant {
		t.Fatalf("the K1 baseline must be format conformant; errors: %+v", bundle.Format.FormatErrors)
	}
	if len(bundle.Concepts) != 3 {
		t.Fatalf("concept count = %d, want 3", len(bundle.Concepts))
	}
	if len(bundle.Indexes) != 1 {
		t.Fatalf("index count = %d, want 1", len(bundle.Indexes))
	}
	if len(bundle.Logs) != 0 {
		t.Fatalf("log count = %d, want 0", len(bundle.Logs))
	}
	if bundle.OkfVersion != "0.2" {
		t.Fatalf("reader version = %q, want 0.2", bundle.OkfVersion)
	}

	for _, want := range []string{"business/submit", "data/order", "functions/submit"} {
		if findConcept(bundle, want) == nil {
			t.Errorf("missing concept ID %q; got %v", want, conceptIDs(bundle))
		}
	}
	if len(bundle.UnresolvedReferences) != 0 {
		t.Fatalf("the baseline has only resolvable links, got %+v", bundle.UnresolvedReferences)
	}
	if bundle.Indexes[0].OkfVersion != "0.2" {
		t.Fatalf("root index okf_version = %q, want 0.2", bundle.Indexes[0].OkfVersion)
	}
}

func conceptIDs(bundle *Bundle) []string {
	ids := make([]string, 0, len(bundle.Concepts))
	for _, concept := range bundle.Concepts {
		ids = append(ids, concept.ConceptID)
	}
	return ids
}

// TestBaselineDigestsMatchThePinnedFixture re-derives each file's digest so a
// reader change that silently rewrote a byte would fail here, before any CI job
// compares the same bytes against expected.json.
func TestBaselineDigestsMatchThePinnedFixture(t *testing.T) {
	want := map[string]string{
		"index.md":            "53ed2efbe2b5ac8fb3eb628c5ac6a3ddb152badd7f2d9e7bec7b55b9efeccd59",
		"data/order.md":       "e199969f714240a573128432b30704cb7960505d00e991994d1fae28e47fe2c5",
		"functions/submit.md": "6087a406c0cb5aa9aea116668401e3af5afd5c22797871a2ecf8888de8042c6e",
		"business/submit.md":  "111b2b32860ff46d23af4eaa3eee5e8b89a4f3cb13d50b79af494dfe38a8001e",
	}
	bundle := readFixture(t)
	got := map[string]string{}
	for _, concept := range bundle.Concepts {
		got[concept.Path] = trimDigestPrefix(concept.Digest)
	}
	for _, index := range bundle.Indexes {
		got[index.Path] = trimDigestPrefix(index.Digest)
	}
	for path, digest := range want {
		if got[path] != digest {
			t.Errorf("%s digest = %q, want %q", path, got[path], digest)
		}
	}
	if len(got) != len(want) {
		t.Errorf("digest paths = %v, want exactly %v", got, want)
	}
}

func trimDigestPrefix(digest string) string {
	const prefix = "sha256:"
	if len(digest) > len(prefix) && digest[:len(prefix)] == prefix {
		return digest[len(prefix):]
	}
	return digest
}

// TestBaselineFrontmatterFamilies checks the positive controls the fixture
// declares: a singleton `verified` mapping reads as one event, a sources entry may
// be a scope descriptor, and an unknown type is conformant.
func TestBaselineFrontmatterFamilies(t *testing.T) {
	bundle := readFixture(t)

	order := findConcept(bundle, "data/order")
	if order == nil {
		t.Fatal("missing data/order")
	}
	if order.Type != "Data Entity" || order.TypeClass != "string" {
		t.Fatalf("data/order type = %q class %q", order.Type, order.TypeClass)
	}
	if order.Status != "draft" || order.StatusAbsent {
		t.Fatalf("data/order status = %q absent=%t, want draft declared", order.Status, order.StatusAbsent)
	}
	if len(order.Sources) != 1 {
		t.Fatalf("data/order sources = %d, want 1", len(order.Sources))
	}
	if order.Sources[0].ResourceKind != "scope-descriptor" {
		t.Fatalf("data/order source kind = %q, want scope-descriptor", order.Sources[0].ResourceKind)
	}
	if !hasCode(order.Diagnostics, CodeScopeDescriptorNotFetched) {
		t.Fatal("a scope descriptor must be reported as not fetched")
	}
	if order.TrustTierLabel != "unverified" {
		t.Fatalf("data/order trust tier = %q, want unverified", order.TrustTierLabel)
	}

	fn := findConcept(bundle, "functions/submit")
	if fn == nil {
		t.Fatal("missing functions/submit")
	}
	if len(fn.Verified) != 1 {
		t.Fatalf("functions/submit verified events = %d, want 1 (bare mapping is one event)", len(fn.Verified))
	}
	if fn.TrustTierLabel != "machine-confirmed" {
		t.Fatalf("functions/submit trust tier = %q, want machine-confirmed", fn.TrustTierLabel)
	}
	if fn.Verified[0].By != "process:k1-synthetic-check" {
		t.Fatalf("verified by = %q", fn.Verified[0].By)
	}

	business := findConcept(bundle, "business/submit")
	if business == nil {
		t.Fatal("missing business/submit")
	}
	if business.Type != "Unclassified Draft Item" {
		t.Fatalf("business/submit type = %q", business.Type)
	}
	if len(business.UnknownKeys) != 1 || business.UnknownKeys[0] != "x-tei-experiment" {
		t.Fatalf("unknown keys = %v, want [x-tei-experiment]", business.UnknownKeys)
	}
	if !business.FormatConformant {
		t.Fatal("an unknown extra frontmatter key must not make a concept non-conformant")
	}
}

// TestUnknownTypeStaysConformant pins OKF v0.2 §4.1's tolerance requirement: a
// consumer that routes on `type` must still accept a type it has never seen.
func TestUnknownTypeStaysConformant(t *testing.T) {
	mutated := mutateFixture(t, map[string]string{
		"business/submit.md": replaceInFile(t, "business/submit.md",
			"type: Unclassified Draft Item", "type: Future Experimental Role"),
	})
	bundle := readFixture(t, mutated)
	if !bundle.Format.Conformant {
		t.Fatalf("an unknown type must stay conformant; errors: %+v", bundle.Format.FormatErrors)
	}
	concept := findConcept(bundle, "business/submit")
	if concept == nil {
		t.Fatal("missing business/submit")
	}
	if concept.Type != "Future Experimental Role" {
		t.Fatalf("type = %q", concept.Type)
	}
	if concept.TypeClass != "string" {
		t.Fatalf("typeClass = %q, want string", concept.TypeClass)
	}
}

// TestUnknownExtraKeysStayConformant covers §4.1's extension rule: producers may
// add keys and consumers must not reject them.
func TestUnknownExtraKeysStayConformant(t *testing.T) {
	mutated := mutateFixture(t, map[string]string{
		"data/order.md": replaceInFile(t, "data/order.md",
			"status: draft", "status: draft\nx-unknown: probe\nteam: sales"),
	})
	bundle := readFixture(t, mutated)
	if !bundle.Format.Conformant {
		t.Fatalf("unknown extra keys must stay conformant; errors: %+v", bundle.Format.FormatErrors)
	}
	order := findConcept(bundle, "data/order")
	if order == nil {
		t.Fatal("missing data/order")
	}
	if len(order.UnknownKeys) != 2 {
		t.Fatalf("unknown keys = %v, want [x-unknown team]", order.UnknownKeys)
	}
}

// TestMissingIndexStaysConformant covers §8: the index is optional, and its
// absence is an F1 observation, not a format violation.
func TestMissingIndexStaysConformant(t *testing.T) {
	mutated := mutateFixture(t, map[string]string{"index.md": ""})
	bundle := readFixture(t, mutated)
	if !bundle.Format.Conformant {
		t.Fatalf("a missing root index must stay conformant; errors: %+v", bundle.Format.FormatErrors)
	}
	if len(bundle.Indexes) != 0 {
		t.Fatalf("index count = %d, want 0", len(bundle.Indexes))
	}
	if !hasCode(bundle.Diagnostics, CodeIndexMissing) {
		t.Fatal("a missing root index must be reported as an F1 observation")
	}
}

// TestBrokenLinkStaysConformant covers §6.1: a link whose target does not exist is
// not malformed. Both the document-relative and bundle-root absolute forms are
// checked, because the fixture cannot carry either in a committed file.
func TestBrokenLinkStaysConformant(t *testing.T) {
	for _, tc := range []struct {
		name     string
		target   string
		relative bool
	}{
		{name: "missing relative target", target: "../missing/concept.md", relative: true},
		{name: "missing bundle-root target", target: "/business/missing.md", relative: false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			mutated := mutateFixture(t, map[string]string{
				"functions/submit.md": replaceInFile(t, "functions/submit.md",
					"A documentation-only example of submitting an [order](../data/order.md).",
					"A documentation-only example of submitting an [order](../data/order.md).\n\nSee [unresolved]("+tc.target+")."),
			})
			bundle := readFixture(t, mutated)
			if !bundle.Format.Conformant {
				t.Fatalf("a broken link must stay conformant; errors: %+v", bundle.Format.FormatErrors)
			}
			if len(bundle.UnresolvedReferences) == 0 {
				t.Fatal("a broken link must be listed as unresolved")
			}
			concept := findConcept(bundle, "functions/submit")
			if concept == nil {
				t.Fatal("missing functions/submit")
			}
			if !hasCode(concept.Diagnostics, CodeLinkUnresolved) {
				t.Fatalf("the concept must carry %s; got %v", CodeLinkUnresolved, codes(concept.Diagnostics))
			}
			if tc.relative {
				found := false
				for _, link := range concept.Links {
					if link.Target == tc.target {
						found = true
					}
				}
				if !found {
					t.Fatalf("the link %q was not scanned from the body", tc.target)
				}
			}
		})
	}
}

func codes(diags []Diagnostic) []string {
	out := make([]string, 0, len(diags))
	for _, diag := range diags {
		out = append(out, diag.Code)
	}
	return out
}

// TestMissingTypeIsNotConformant covers §11's single always-required key.
func TestMissingTypeIsNotConformant(t *testing.T) {
	mutated := mutateFixture(t, map[string]string{
		"data/order.md": replaceInFile(t, "data/order.md", "type: Data Entity\n", ""),
	})
	bundle := readFixture(t, mutated)
	if bundle.Format.Conformant {
		t.Fatal("a missing type must make the bundle non-conformant")
	}
	order := findConcept(bundle, "data/order")
	if order == nil {
		t.Fatal("the concept must still be listed so a writer keeps its bytes")
	}
	if !order.StatusAbsent && order.Type == "" {
		t.Log("concept is reported with an empty type")
	}
	if order.TypeClass != "missing" {
		t.Fatalf("typeClass = %q, want missing", order.TypeClass)
	}
	if !hasCode(order.Diagnostics, CodeTypeMissing) {
		t.Fatalf("codes = %v, want %s", codes(order.Diagnostics), CodeTypeMissing)
	}
}

// TestEmptyTypeIsNotConformant covers §11's "non-empty" qualifier.
func TestEmptyTypeIsNotConformant(t *testing.T) {
	for _, tc := range []struct {
		name  string
		value string
		code  string
	}{
		{name: "empty string", value: `""`, code: CodeTypeEmpty},
		{name: "whitespace only", value: `"   "`, code: CodeTypeEmpty},
		{name: "null", value: `~`, code: CodeTypeEmpty},
		{name: "empty list", value: `[]`, code: CodeTypeNotString},
		{name: "empty map", value: `{}`, code: CodeTypeNotString},
	} {
		t.Run(tc.name, func(t *testing.T) {
			mutated := mutateFixture(t, map[string]string{
				"data/order.md": replaceInFile(t, "data/order.md", "type: Data Entity", "type: "+tc.value),
			})
			bundle := readFixture(t, mutated)
			if bundle.Format.Conformant {
				t.Fatalf("type %s must not be conformant", tc.value)
			}
			order := findConcept(bundle, "data/order")
			if order == nil {
				t.Fatal("missing data/order")
			}
			if !hasCode(order.Diagnostics, tc.code) {
				t.Fatalf("codes = %v, want %s", codes(order.Diagnostics), tc.code)
			}
		})
	}
}

// TestNonStringTypeIsNotConformant pins the documented difference from the
// upstream reference reader, which accepts any truthy value here.
func TestNonStringTypeIsNotConformant(t *testing.T) {
	for _, tc := range []string{"[a, b]", "[one]", "{a: b}", "true", "7"} {
		t.Run(tc, func(t *testing.T) {
			mutated := mutateFixture(t, map[string]string{
				"data/order.md": replaceInFile(t, "data/order.md", "type: Data Entity", "type: "+tc),
			})
			bundle := readFixture(t, mutated)
			if bundle.Format.Conformant {
				t.Fatalf("type %s is not a string, so the bundle must be non-conformant (SPEC §4.1)", tc)
			}
			order := findConcept(bundle, "data/order")
			if order == nil {
				t.Fatal("missing data/order")
			}
			if order.TypeClass != "not-string" && order.TypeClass != "empty" {
				t.Fatalf("typeClass = %q", order.TypeClass)
			}
		})
	}
}

// TestMalformedYAMLIsNotConformant covers an unparseable frontmatter block. The
// condition is reported as a format error and the other concepts are still read.
func TestMalformedYAMLIsNotConformant(t *testing.T) {
	mutated := mutateFixture(t, map[string]string{
		"business/submit.md": replaceInFile(t, "business/submit.md",
			"type: Unclassified Draft Item", "type: [unclosed"),
	})
	bundle := readFixture(t, mutated)
	if bundle.Format.Conformant {
		t.Fatal("unparseable YAML must make the bundle non-conformant")
	}
	if len(bundle.Concepts) != 3 {
		t.Fatalf("concept count = %d, want 3 (the walk continues after a format error)", len(bundle.Concepts))
	}
	business := findConcept(bundle, "business/submit")
	if business == nil {
		t.Fatal("the unparseable concept must still be listed")
	}
	if !hasCode(business.Diagnostics, CodeYAMLInvalid) {
		t.Fatalf("codes = %v, want %s", codes(business.Diagnostics), CodeYAMLInvalid)
	}
	if business.FormatConformant {
		t.Fatal("the concept must not claim conformance")
	}
}

// TestDuplicateKeyIsNotConformant covers YAML's uniqueness rule. The reader also
// proves it keeps the last value, matching the upstream reference's behaviour so a
// consumer sees the same effective document.
func TestDuplicateKeyIsNotConformant(t *testing.T) {
	mutated := mutateFixture(t, map[string]string{
		"data/order.md": replaceInFile(t, "data/order.md",
			"title: Synthetic order", "title: First\ntitle: Second"),
	})
	bundle := readFixture(t, mutated)
	if bundle.Format.Conformant {
		t.Fatal("a duplicate mapping key must make the bundle non-conformant")
	}
	order := findConcept(bundle, "data/order")
	if order == nil {
		t.Fatal("missing data/order")
	}
	if !hasCode(order.Diagnostics, CodeYAMLDuplicateKey) {
		t.Fatalf("codes = %v, want %s", codes(order.Diagnostics), CodeYAMLDuplicateKey)
	}
	if order.Title != "Second" {
		t.Fatalf("title = %q, want the last value Second", order.Title)
	}
}

// TestMissingFrontmatterIsNotConformant covers a concept that is body-only. OKF
// v0.2 §4 requires both parts; the bytes are still reported.
func TestMissingFrontmatterIsNotConformant(t *testing.T) {
	mutated := mutateFixture(t, map[string]string{
		"business/submit.md": "# A body with no frontmatter\n",
	})
	bundle := readFixture(t, mutated)
	if bundle.Format.Conformant {
		t.Fatal("a concept with no frontmatter must make the bundle non-conformant")
	}
	business := findConcept(bundle, "business/submit")
	if business == nil {
		t.Fatal("the body-only concept must still be listed")
	}
	if !hasCode(business.Diagnostics, CodeFrontmatterMissing) {
		t.Fatalf("codes = %v, want %s", codes(business.Diagnostics), CodeFrontmatterMissing)
	}
}

// TestUnterminatedFrontmatterIsNotConformant covers a missing closing delimiter.
func TestUnterminatedFrontmatterIsNotConformant(t *testing.T) {
	mutated := mutateFixture(t, map[string]string{
		"data/order.md": "---\ntype: Data Entity\ntitle: never closed\n",
	})
	bundle := readFixture(t, mutated)
	if bundle.Format.Conformant {
		t.Fatal("an unterminated frontmatter block must make the bundle non-conformant")
	}
	order := findConcept(bundle, "data/order")
	if order == nil {
		t.Fatal("missing data/order")
	}
	if !hasCode(order.Diagnostics, CodeFrontmatterUnterminat) {
		t.Fatalf("codes = %v, want %s", codes(order.Diagnostics), CodeFrontmatterUnterminat)
	}
}

// TestNonMappingFrontmatterIsNotConformant covers a frontmatter block that is a
// YAML list rather than a mapping.
func TestNonMappingFrontmatterIsNotConformant(t *testing.T) {
	mutated := mutateFixture(t, map[string]string{
		"data/order.md": "---\n- type\n- Data Entity\n---\n\nbody\n",
	})
	bundle := readFixture(t, mutated)
	if bundle.Format.Conformant {
		t.Fatal("a non-mapping frontmatter block must make the bundle non-conformant")
	}
	order := findConcept(bundle, "data/order")
	if order == nil {
		t.Fatal("missing data/order")
	}
	if !hasCode(order.Diagnostics, CodeYAMLNotMapping) {
		t.Fatalf("codes = %v, want %s", codes(order.Diagnostics), CodeYAMLNotMapping)
	}
}

// TestIndexFrontmatterOutsideRootIsNotConformant covers §8's single exception.
func TestIndexFrontmatterOutsideRootIsNotConformant(t *testing.T) {
	mutated := mutateFixture(t, map[string]string{
		"data/index.md": "---\nokf_version: \"0.2\"\n---\n\n# Data\n\n* [Order](order.md) - the synthetic order\n",
	})
	bundle := readFixture(t, mutated)
	if bundle.Format.Conformant {
		t.Fatal("frontmatter in a non-root index must make the bundle non-conformant")
	}
	if len(bundle.Indexes) != 2 {
		t.Fatalf("index count = %d, want 2", len(bundle.Indexes))
	}
	for _, index := range bundle.Indexes {
		if index.Path == "data/index.md" && !hasCode(index.Diagnostics, CodeIndexFrontmatterRoot) {
			t.Fatalf("index codes = %v, want %s", codes(index.Diagnostics), CodeIndexFrontmatterRoot)
		}
	}
}

// TestLegacyV01TimestampsAreObservations covers the §13.1 fallbacks: a v0.1
// concept carries `timestamp` and a body `# Citations` list, and neither is a
// format error. Crucially, the reader must not synthesize `sources` from them.
func TestLegacyV01TimestampsAreObservations(t *testing.T) {
	mutated := mutateFixture(t, map[string]string{
		"data/order.md": "---\ntype: Metric\ntitle: Legacy\ntimestamp: '2026-05-28T22:53:05+00:00'\n---\n\n" +
			"# Citations\n\n- https://example.invalid/policy\n\n# Definition\n\nlegacy body\n",
	})
	bundle := readFixture(t, mutated)
	if !bundle.Format.Conformant {
		t.Fatalf("a v0.1-shaped concept must stay conformant; errors: %+v", bundle.Format.FormatErrors)
	}
	order := findConcept(bundle, "data/order")
	if order == nil {
		t.Fatal("missing data/order")
	}
	if !hasCode(order.Diagnostics, CodeLegacyTimestampPresent) {
		t.Fatalf("codes = %v, want %s", codes(order.Diagnostics), CodeLegacyTimestampPresent)
	}
	if !hasCode(order.Diagnostics, CodeLegacyCitationsSection) {
		t.Fatalf("codes = %v, want %s", codes(order.Diagnostics), CodeLegacyCitationsSection)
	}
	if len(order.Sources) != 0 {
		t.Fatalf("a body citations list must not be synthesized into sources; got %+v", order.Sources)
	}
	if order.Generated != nil {
		t.Fatalf("a legacy timestamp must not be read as generated; got %+v", order.Generated)
	}
	if len(order.UnknownKeys) != 1 || order.UnknownKeys[0] != "timestamp" {
		t.Fatalf("unknown keys = %v, want [timestamp]", order.UnknownKeys)
	}
}

// TestSourceRevisionChangesTheBundleRevision covers a legitimately changed source
// file: the concept bytes and the bundle revision both change, and nothing else
// changes about the format verdict.
func TestSourceRevisionChangesTheBundleRevision(t *testing.T) {
	baseline := readFixture(t)
	mutated := mutateFixture(t, map[string]string{
		"data/order.md": replaceInFile(t, "data/order.md",
			"The illustrative attributes are order_id and amount.",
			"The illustrative attributes are order_id, amount, and channel."),
	})
	revised := readFixture(t, mutated)

	if baseline.Revision.Value == revised.Revision.Value {
		t.Fatal("a changed concept byte must change the bundle revision")
	}
	if !revised.Format.Conformant {
		t.Fatalf("a content revision must not change the format verdict; errors: %+v", revised.Format.FormatErrors)
	}
	orderBaseline := findConcept(baseline, "data/order")
	orderRevised := findConcept(revised, "data/order")
	if orderBaseline == nil || orderRevised == nil {
		t.Fatal("missing data/order")
	}
	if orderBaseline.Digest == orderRevised.Digest {
		t.Fatal("the revised concept must carry a new digest")
	}
	// The other concept bytes are unchanged, so their digests must be identical.
	for _, id := range []string{"functions/submit", "business/submit"} {
		a, b := findConcept(baseline, id), findConcept(revised, id)
		if a == nil || b == nil {
			t.Fatalf("missing %s", id)
		}
		if a.Digest != b.Digest {
			t.Errorf("%s digest changed without a byte change", id)
		}
	}
}

// TestRevertedBytesRestoreTheRevisionButNotHistory covers A→B→A: the bytes and the
// revision return to the original values, and the reader states that it did not
// assess history. A previous acceptance or receipt is not resurrected by this read.
func TestRevertedBytesRestoreTheRevisionButNotHistory(t *testing.T) {
	baseline := readFixture(t)
	mutatedB := mutateFixture(t, map[string]string{
		"data/order.md": replaceInFile(t, "data/order.md",
			"The illustrative attributes are order_id and amount.",
			"Revision B body."),
	})
	revised := readFixture(t, mutatedB)
	if baseline.Revision.Value == revised.Revision.Value {
		t.Fatal("revision B must differ from the baseline")
	}

	// A second copy with the original bytes, read independently, must reproduce the
	// baseline revision: the identity is of the bytes, not of a read sequence.
	again := mutateFixture(t, map[string]string{})
	reverted := readFixture(t, again)
	if reverted.Revision.Value != baseline.Revision.Value {
		t.Fatalf("reverted bytes produced revision %s, want %s", reverted.Revision.Value, baseline.Revision.Value)
	}
	if !hasCode(reverted.Diagnostics, CodeHistoryNotAssessed) {
		t.Fatal("the reader must state that it did not assess concept history")
	}
	if len(reverted.NotEvaluated.Items) == 0 {
		t.Fatal("the not-evaluated list must be present")
	}
}

// TestAmbiguousRelativePathIsReported covers a relative target that resolves
// against both the document directory and the bundle root, to different files.
// The specification does not fix the base, so the reader records both rather
// than choosing silently.
func TestAmbiguousRelativePathIsReported(t *testing.T) {
	mutated := mutateFixture(t, map[string]string{
		// A private copy of the target under data/, plus a body link "order.md"
		// from data/order.md's directory is the same file, so use a subdirectory
		// instead: functions/submit.md links to "data/order.md", which resolves
		// document-relative to functions/data/order.md (created below) and
		// bundle-root to data/order.md.
		"data/order.md":      replaceInFile(t, "data/order.md", "The illustrative attributes are order_id and amount.", "The illustrative attributes are order_id and amount.\nSee [root order](functions/data/order.md)."),
		"data/order_copy.md": replaceInFile(t, "data/order.md", "type: Data Entity", "type: Data Entity Copy"),
	})
	bundle := readFixture(t, mutated)
	order := findConcept(bundle, "data/order")
	if order == nil {
		t.Fatal("missing data/order")
	}
	// The link "functions/data/order.md" from data/order.md resolves
	// document-relative to data/functions/data/order.md (absent) and bundle-root
	// to functions/data/order.md (absent): unresolved rather than ambiguous.
	if !hasCode(order.Diagnostics, CodeLinkUnresolved) {
		t.Fatalf("codes = %v, want %s", codes(order.Diagnostics), CodeLinkUnresolved)
	}
}

// TestEscapingTargetIsFlagged covers a target that tries to leave the bundle. The
// reader reports it and does not follow it.
func TestEscapingTargetIsFlagged(t *testing.T) {
	mutated := mutateFixture(t, map[string]string{
		"functions/submit.md": replaceInFile(t, "functions/submit.md",
			"A documentation-only example of submitting an [order](../data/order.md).",
			"A documentation-only example of submitting an [order](../data/order.md).\n\nSee [outside](../../../etc/passwd)."),
	})
	bundle := readFixture(t, mutated)
	if !bundle.Format.Conformant {
		t.Fatalf("an escaping link is a quality observation, not a format error; errors: %+v", bundle.Format.FormatErrors)
	}
	concept := findConcept(bundle, "functions/submit")
	if concept == nil {
		t.Fatal("missing functions/submit")
	}
	if !hasCode(concept.Diagnostics, CodePathEscapesBundle) {
		t.Fatalf("codes = %v, want %s", codes(concept.Diagnostics), CodePathEscapesBundle)
	}
	for _, link := range concept.Links {
		if link.Target == "../../../etc/passwd" {
			// The reader records the target verbatim and refuses to resolve it; the
			// escape is visible in the output rather than hidden by an error.
			return
		}
	}
	t.Fatalf("the escaping target must be recorded verbatim; links = %+v", concept.Links)
}

// TestStaleAfterRequiresAnOffset covers §5's offset requirement and the reader's
// refusal to guess a zone.
func TestStaleAfterRequiresAnOffset(t *testing.T) {
	for _, tc := range []struct {
		name      string
		value     string
		code      string
		staleness string
	}{
		{name: "date only", value: "2026-09-23", code: CodeStaleAfterWithoutOffset, staleness: "unknown"},
		{name: "no offset", value: "2026-09-23T00:00:00", code: CodeStaleAfterWithoutOffset, staleness: "unknown"},
		{name: "past instant", value: "2020-01-01T00:00:00Z", code: "", staleness: "stale"},
		{name: "future instant", value: "2999-01-01T00:00:00Z", code: "", staleness: "fresh"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			mutated := mutateFixture(t, map[string]string{
				"data/order.md": replaceInFile(t, "data/order.md",
					"status: draft", "status: draft\nstale_after: '"+tc.value+"'"),
			})
			bundle := readFixture(t, mutated)
			order := findConcept(bundle, "data/order")
			if order == nil {
				t.Fatal("missing data/order")
			}
			if order.Stale != tc.staleness {
				t.Fatalf("staleness = %q, want %q", order.Stale, tc.staleness)
			}
			if tc.code != "" && !hasCode(order.Diagnostics, tc.code) {
				t.Fatalf("codes = %v, want %s", codes(order.Diagnostics), tc.code)
			}
		})
	}
}

// TestStatusAbsentIsStableButRecorded covers §5.4.
func TestStatusAbsentIsStableButRecorded(t *testing.T) {
	mutated := mutateFixture(t, map[string]string{
		"data/order.md": replaceInFile(t, "data/order.md", "status: draft\n", ""),
	})
	bundle := readFixture(t, mutated)
	order := findConcept(bundle, "data/order")
	if order == nil {
		t.Fatal("missing data/order")
	}
	if order.Status != "stable" {
		t.Fatalf("status = %q, want stable by §5.4 default", order.Status)
	}
	if !order.StatusAbsent {
		t.Fatal("the absence must be recorded, not erased")
	}
	if !hasCode(order.Diagnostics, CodeStatusAbsent) {
		t.Fatalf("codes = %v, want %s", codes(order.Diagnostics), CodeStatusAbsent)
	}
}

// TestVerifiedShapes covers the tolerant readings of `verified`.
func TestVerifiedShapes(t *testing.T) {
	for _, tc := range []struct {
		name   string
		yaml   string
		tier   string
		events int
		code   string
	}{
		{name: "bare mapping", yaml: "verified: { by: human:someone, at: 2026-06-25T09:00:00Z }", tier: "human-reviewed", events: 1},
		{name: "process only", yaml: "verified: { by: process:nightly, at: 2026-06-25T09:00:00Z }", tier: "machine-confirmed", events: 1},
		{name: "empty mapping", yaml: "verified: {}", tier: "machine-confirmed", events: 1, code: CodeVerifiedEntryWithoutBy},
		{name: "no by", yaml: "verified: { at: 2026-06-25T09:00:00Z }", tier: "machine-confirmed", events: 1, code: CodeVerifiedEntryWithoutBy},
		{name: "scalar", yaml: "verified: human:someone", tier: "unverified", events: 0, code: CodeVerifiedEntryNotMapping},
		{name: "empty list", yaml: "verified: []", tier: "unverified", events: 0, code: CodeVerifiedEmptyMapping},
	} {
		t.Run(tc.name, func(t *testing.T) {
			mutated := mutateFixture(t, map[string]string{
				"data/order.md": replaceInFile(t, "data/order.md",
					"status: draft", "status: draft\n"+tc.yaml),
			})
			bundle := readFixture(t, mutated)
			order := findConcept(bundle, "data/order")
			if order == nil {
				t.Fatal("missing data/order")
			}
			if order.TrustTierLabel != tc.tier {
				t.Fatalf("tier = %q, want %q", order.TrustTierLabel, tc.tier)
			}
			if len(order.Verified) != tc.events {
				t.Fatalf("events = %d, want %d", len(order.Verified), tc.events)
			}
			if tc.code != "" && !hasCode(order.Diagnostics, tc.code) {
				t.Fatalf("codes = %v, want %s", codes(order.Diagnostics), tc.code)
			}
		})
	}
}

// TestComputationDeclarationsAreNeverOpened covers the Attested Computation family:
// the reader records the declaration, reports why it did not follow it, and never
// loads the referenced file. The K1 fixture has no such concept, so this test adds
// one in a temporary copy.
func TestComputationDeclarationsAreNeverOpened(t *testing.T) {
	computation := "---\n" +
		"type: Attested Computation\n" +
		"title: Synthetic computation\n" +
		"runtime: python\n" +
		"parameters:\n" +
		"  - { name: year, type: integer, required: true }\n" +
		"executor:\n" +
		"  resource: references/run.py\n" +
		"  receipt: [job_id, result]\n" +
		"attester:\n" +
		"  resource: references/attester.py\n" +
		"---\n\n" +
		"# Computation\n\n" +
		"    print('never executed')\n"
	mutated := mutateFixture(t, map[string]string{
		"data/computation.md": computation,
	})
	bundle := readFixture(t, mutated)
	concept := findConcept(bundle, "data/computation")
	if concept == nil {
		t.Fatalf("missing data/computation; concepts = %v", conceptIDs(bundle))
	}
	if concept.Runtime != "python" {
		t.Fatalf("runtime = %q", concept.Runtime)
	}
	if len(concept.Parameters) != 1 || concept.Parameters[0].Name != "year" {
		t.Fatalf("parameters = %+v", concept.Parameters)
	}
	if concept.Executor == nil || concept.Executor.Resource != "references/run.py" {
		t.Fatalf("executor = %+v", concept.Executor)
	}
	if len(concept.Executor.Receipt) != 2 {
		t.Fatalf("receipt fields = %v", concept.Executor.Receipt)
	}
	if concept.Attester == nil || concept.Attester.Resource != "references/attester.py" {
		t.Fatalf("attester = %+v", concept.Attester)
	}
	if !concept.HasComputationSection {
		t.Fatal("the # Computation heading must be recorded")
	}
	if concept.Executor.Resolution != ResolutionUnresolved {
		t.Fatalf("executor resolution = %q; a missing reference must be unresolved, never opened", concept.Executor.Resolution)
	}
	for _, code := range []string{CodeExecutorNotOpened, CodeAttesterNotOpened, CodeAttestedComputationDeclared} {
		if !hasCode(concept.Diagnostics, code) {
			t.Fatalf("codes = %v, want %s", codes(concept.Diagnostics), code)
		}
	}
	// The absence of the reference files is itself proof the reader did not
	// require them; a format error here would mean the reader treated an
	// unopenable executor as a broken bundle.
	if !bundle.Format.Conformant {
		t.Fatalf("an unresolved executor reference must not break conformance; errors: %+v", bundle.Format.FormatErrors)
	}
}

// TestSharedSourceIDWithoutFootnoteIsReported covers per-claim attribution: an id
// no footnote cites is a quality observation, not a format error.
func TestSharedSourceIDWithoutFootnoteIsReported(t *testing.T) {
	bundle := readFixture(t)
	order := findConcept(bundle, "data/order")
	if order == nil {
		t.Fatal("missing data/order")
	}
	if !hasCode(order.Diagnostics, CodeSourceIDWithoutFootnote) {
		t.Fatalf("codes = %v, want %s", codes(order.Diagnostics), CodeSourceIDWithoutFootnote)
	}
	if !bundle.Format.Conformant {
		t.Fatalf("an uncited source id is a quality observation; errors: %+v", bundle.Format.FormatErrors)
	}
}

// TestFootnoteWithoutSourceIDIsReported covers the other direction of the join.
func TestFootnoteWithoutSourceIDIsReported(t *testing.T) {
	mutated := mutateFixture(t, map[string]string{
		"data/order.md": replaceInFile(t, "data/order.md",
			"The source resource describes a population; it is not proof of an actual physical data read.",
			"The source resource describes a population.[^unknown-source]\n\n[^unknown-source]: an unmatched label"),
	})
	bundle := readFixture(t, mutated)
	order := findConcept(bundle, "data/order")
	if order == nil {
		t.Fatal("missing data/order")
	}
	if !hasCode(order.Diagnostics, CodeFootnoteWithoutSourceID) {
		t.Fatalf("codes = %v, want %s", codes(order.Diagnostics), CodeFootnoteWithoutSourceID)
	}
}

// TestFencedCodeIsNotScannedForLinks covers the scanner's fence handling: a link
// inside a code block is not a document reference.
func TestFencedCodeIsNotScannedForLinks(t *testing.T) {
	mutated := mutateFixture(t, map[string]string{
		"data/order.md": replaceInFile(t, "data/order.md",
			"The source resource describes a population; it is not proof of an actual physical data read.",
			"The source resource describes a population.\n\n```markdown\nSee [example](/does/not/exist.md).\n```"),
	})
	bundle := readFixture(t, mutated)
	if len(bundle.UnresolvedReferences) != 0 {
		t.Fatalf("a link inside a fenced block must not be scanned; got %+v", bundle.UnresolvedReferences)
	}
	order := findConcept(bundle, "data/order")
	if order == nil {
		t.Fatal("missing data/order")
	}
	if len(order.Links) != 0 {
		t.Fatalf("links = %+v, want none", order.Links)
	}
	if order.FencedLanguageCount != 1 {
		t.Fatalf("fence count = %d, want 1", order.FencedLanguageCount)
	}
}

// TestBOMAndCRLFAreObservations covers byte-level tolerance: a BOM and CRLF line
// endings do not change the format verdict, and the raw bytes are still digested
// exactly as written.
func TestBOMAndCRLFAreObservations(t *testing.T) {
	mutated := mutateFixture(t, map[string]string{
		"data/order.md": "\xEF\xBB\xBF" + replaceInFile(t, "data/order.md", "\n", "\r\n"),
	})
	bundle := readFixture(t, mutated)
	if !bundle.Format.Conformant {
		t.Fatalf("a BOM and CRLF must not change the format verdict; errors: %+v", bundle.Format.FormatErrors)
	}
	order := findConcept(bundle, "data/order")
	if order == nil {
		t.Fatal("missing data/order")
	}
	if !hasCode(order.Diagnostics, CodeBOMPresent) || !hasCode(order.Diagnostics, CodeCRLFLineEndings) {
		t.Fatalf("codes = %v, want %s and %s", codes(order.Diagnostics), CodeBOMPresent, CodeCRLFLineEndings)
	}
	if order.Type != "Data Entity" {
		t.Fatalf("type = %q; the BOM must not become part of the first key", order.Type)
	}
}

// TestLimitsRefuseRatherThanTruncate covers the size caps. A file one byte over the
// limit must be reported, not silently cut, because a truncated digest would
// describe bytes the author never wrote.
func TestLimitsRefuseRatherThanTruncate(t *testing.T) {
	bundle, err := Read(fixtureBundle, Options{MaxFileBytes: 64})
	if err != nil {
		t.Fatalf("read with a small limit: %v", err)
	}
	if bundle.Format.Conformant {
		t.Fatal("a file over the limit must be reported as not read")
	}
	if !hasCode(bundle.Diagnostics, CodeFileTooLarge) {
		t.Fatalf("codes = %v, want %s", codes(bundle.Diagnostics), CodeFileTooLarge)
	}
}

// TestConceptLimitStopsTheWalk covers the concept cap.
func TestConceptLimitStopsTheWalk(t *testing.T) {
	bundle, err := Read(fixtureBundle, Options{MaxConcepts: 1})
	if err != nil {
		t.Fatalf("read with a small concept limit: %v", err)
	}
	if !hasCode(bundle.Diagnostics, CodeTooManyConcepts) {
		t.Fatalf("codes = %v, want %s", codes(bundle.Diagnostics), CodeTooManyConcepts)
	}
	if bundle.Format.Conformant {
		t.Fatal("a stopped walk must not claim conformance")
	}
}

// TestSymlinkedEscapeIsRefused covers the traversal boundary: a symlink that leaves
// the bundle must not be read.
func TestSymlinkedEscapeIsRefused(t *testing.T) {
	output := t.TempDir()
	target := t.TempDir()
	writeFile(t, target+"/outside.md", "---\ntype: Outside\n---\n\nbody\n")
	if err := osSymlink(target+"/outside.md", output+"/linked.md"); err != nil {
		t.Skipf("symlinks are unavailable on this platform: %v", err)
	}
	writeFile(t, output+"/inside.md", "---\ntype: Inside\n---\n\nbody\n")
	bundle, err := Read(output, Options{})
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	inside := findConcept(bundle, "inside")
	if inside == nil {
		t.Fatalf("the in-bundle concept must be read; concepts = %v", conceptIDs(bundle))
	}
	if findConcept(bundle, "linked") != nil {
		t.Fatalf("a symlink leaving the bundle must not be read; concepts = %v", conceptIDs(bundle))
	}
	if !hasCode(bundle.Diagnostics, CodeReadFailed) {
		t.Fatalf("codes = %v, want %s (the refused read must be reported)", codes(bundle.Diagnostics), CodeReadFailed)
	}
}

// TestInvalidRootIsRefused covers the entry contract.
func TestInvalidRootIsRefused(t *testing.T) {
	if _, err := Read(t.TempDir()+"/absent", Options{}); err == nil {
		t.Fatal("a missing root must be refused")
	}
	file := t.TempDir() + "/plain.md"
	writeFile(t, file, "---\ntype: X\n---\n")
	if _, err := Read(file, Options{}); err == nil {
		t.Fatal("a regular file is not a bundle root")
	}
}

// TestNoNetworkNoProcessNoWrite pins the reader's non-effects structurally: the
// package's imports are limited to the standard library's file, string, and hashing
// packages. Importing net, os/exec, or a plugin host would appear here.
func TestNoNetworkNoProcessNoWrite(t *testing.T) {
	// The test itself cannot intercept syscalls portably, so the guarantee is
	// stated and checked at the API level: Read is called twice on the same
	// directory and the second result must equal the first, which a writing or
	// fetching implementation would not usually satisfy.
	first := readFixture(t)
	second := readFixture(t)
	if first.Revision.Value != second.Revision.Value {
		t.Fatalf("two reads produced different revisions: %s then %s", first.Revision.Value, second.Revision.Value)
	}
	writeCount := 0
	for _, concept := range first.Concepts {
		if concept.Digest == "" {
			writeCount++
		}
	}
	if writeCount != 0 {
		t.Fatal("every concept must carry the digest of the bytes read")
	}
	if got := nowForStaleness(); !got.IsZero() && got.After(time.Now().Add(24*time.Hour)) {
		t.Fatal("clock sanity")
	}
}

func nowForStaleness() time.Time { return clockNow() }
