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
	"internal/businesscalendar/calendar.go/WeekStartsSunday",
	"internal/composition/proposal.go/ReasonStructurallyEligible",
	"internal/driver/observation.go/ObservationEvidenceManagedHere",
	"internal/numbering/numbering.go/OverflowStop",
	"internal/plugin/build.go/BuildOutputBuilderArtifact",
	"internal/plugin/transform/model.go/OpConstructObject",
	"internal/plugin/transform/value.go/ValueBool",
	"internal/plugin/transform/value.go/ValueNumber",
	"internal/query/relational.go/RelDistinct",
	"internal/query/relational.go/RelOpaque",
	"internal/releasegraph/model.go/NodeApplicationRelease",
	"internal/releasegraph/model.go/NodeDatabaseReadiness",
	"internal/releasegraph/model.go/NodeDatabaseState",
	"internal/security/effectcommit/evidence.go/ErrorClassContext",
	"internal/security/effectcommit/evidence.go/ErrorClassMissingProvider",
	"internal/security/provisionaleffectgrant/grant.go/MarkProvisional",
	"internal/sourceacceptance/hoststore/store.go/StateCurrent",
	"internal/sourceacceptance/hoststore/store.go/StateRevoked",
	"internal/sourceacceptance/hoststore/store.go/StateSuperseded",
	"internal/xeadsource/crossview.go/CrossViewNatureSource",
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
