package plancheck

import "testing"

// A named string constant with an exported name is a declared vocabulary word:
// a JSON value, a state, a kind. If no test mentions the name, nothing shows that
// the word is produced, which is the same gap the diagnostic codes had (#4032).
//
// The list below is the current state, sorted. One entry is `package/file.go/Name`.
// It may only shrink: a constant that gains a test has to be removed here, and a
// new exported string constant fails the check until it has a test or an entry.
// An entry is a to-do with a reason, not a licence to leave a word untested.
var unmentionedStringConstants = []string{
	// internal/query の語彙（relation.go の4つと standard/model.go の1つ）は、
	// 並行変更で追加された語がまだどの試験にも現れていない。語を足した変更の
	// 担当が試験を付けるまでの to-do としてここへ置く。
	"internal/query/relation.go/RelationFetchModeEager",
	"internal/query/relation.go/RelationQueryModeGraph",
	"internal/query/relation.go/RelationQueryModePath",
	"internal/query/relation.go/RelationQueryModeTree",
	"internal/query/standard/model.go/SeverityUnsupported",
}

func TestExportedStringConstantsHaveATestOrAnEntry(t *testing.T) {
	root := repositoryRoot(t)
	unmentioned := unmentionedExportedStringConstants(t, root)
	allowed := map[string]bool{}
	for _, entry := range unmentionedStringConstants {
		allowed[entry] = true
	}

	var unexpected, stale []string
	for _, entry := range unmentioned {
		if !allowed[entry] {
			unexpected = append(unexpected, entry)
		}
	}
	sorted := append([]string(nil), unmentioned...)
	for _, entry := range unmentionedStringConstants {
		found := false
		for _, actual := range sorted {
			if actual == entry {
				found = true
				break
			}
		}
		if !found {
			stale = append(stale, entry)
		}
	}
	for _, entry := range stale {
		t.Errorf("%s now has a test; remove it from unmentionedStringConstants", entry)
	}
	for _, entry := range unexpected {
		t.Errorf("%s is a declared word that no test mentions; add a test or an entry", entry)
	}
	if len(stale) != 0 || len(unexpected) != 0 {
		t.Fatalf("%d new constants and %d stale entries", len(unexpected), len(stale))
	}
}
