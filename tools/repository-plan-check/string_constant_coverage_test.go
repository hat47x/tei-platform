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
	"internal/database/migration/provider.go/ToolFlyway",
	"internal/database/migration/provider.go/ToolGolangMigrate",
	"internal/database/physical/model.go/TableKindMaterializedView",
	"internal/database/physical/model.go/TableKindView",
	"internal/database/plan/model.go/NodeAggregate",
	"internal/database/plan/model.go/NodeGather",
	"internal/database/plan/model.go/NodeIndexScan",
	"internal/database/plan/model.go/NodeJoin",
	"internal/database/plan/model.go/NodeLimit",
	"internal/database/plan/model.go/NodeMaterialize",
	"internal/database/plan/model.go/NodeModify",
	"internal/database/plan/model.go/NodeOther",
	"internal/database/plan/model.go/NodeSetOperation",
	"internal/database/provider/provider.go/CapabilityCancel",
	"internal/database/provider/provider.go/CapabilityDataMutation",
	"internal/database/provider/provider.go/CapabilityExplainAnalyze",
	"internal/database/provider/provider.go/CapabilityTransaction",
	"internal/database/reconcile/model.go/FindingDivergedMigration",
	"internal/database/reconcile/model.go/FindingFailedMigration",
	"internal/database/reconcile/model.go/FindingOrphanApplied",
	"internal/database/reconcile/model.go/FindingPendingMigration",
	"internal/deployment/durable/model.go/EventActionCompleted",
	"internal/deployment/durable/model.go/EventCommandAttempted",
	"internal/deployment/durable/model.go/EventEvidenceAccepted",
	"internal/deployment/durable/model.go/EventEvidenceRejected",
	"internal/deployment/durable/model.go/EventRecoveryPlanned",
	"internal/deployment/durable/model.go/EventRecoveryRunLinked",
	"internal/deployment/durable/model.go/EventRetryAuthorized",
	"internal/deployment/durable/model.go/EventRunAborted",
	"internal/deployment/durable/model.go/EventRunPaused",
	"internal/deployment/durable/model.go/EventRunResumed",
	"internal/deployment/durable/model.go/ModeObservation",
	"internal/deployment/durable/model.go/RecoveryManual",
	"internal/deployment/durable/model.go/RecoveryRetry",
	"internal/deployment/durable/model.go/RecoveryRollback",
	"internal/deployment/execution/model.go/EventApplicationStarted",
	"internal/deployment/execution/model.go/EventDatabaseGateSatisfied",
	"internal/deployment/execution/model.go/EventDatabaseReleaseApplied",
	"internal/deployment/execution/model.go/EventDrainStarted",
	"internal/deployment/execution/model.go/EventRolloutCheckpoint",
	"internal/deployment/kinds.go/ActionDrainCompleteGate",
	"internal/deployment/kinds.go/ActionStopAcceptingWork",
	"internal/deployment/model.go/LifecycleStarting",
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
