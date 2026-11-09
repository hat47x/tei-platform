package main

import (
	"fmt"
	"strings"
)

const (
	receiptSchema  = "tei.reference.order-entry-persistence/v0"
	evidenceNature = "real-postgres-ephemeral-fixture"
)

type scopeStatement struct {
	ProductE2                  bool   `json:"productE2"`
	SyntheticSource            bool   `json:"syntheticSource"`
	SyntheticHostEffectGrant   bool   `json:"syntheticHostEffectGrant"`
	HandWrittenOverlay         bool   `json:"handWrittenOverlay"`
	DDLFromPersistenceMapping  bool   `json:"ddlFromPersistenceMapping"`
	GeneratedCRUD              bool   `json:"generatedCrud"`
	StandardWebPath            bool   `json:"standardWebPath"`
	SeparateProcessesRequested bool   `json:"separateProcessesRequested"`
	Note                       string `json:"note"`
}

func fixedScope() scopeStatement {
	return scopeStatement{
		ProductE2:                  false,
		SyntheticSource:            true,
		SyntheticHostEffectGrant:   true,
		HandWrittenOverlay:         false,
		DDLFromPersistenceMapping:  true,
		GeneratedCRUD:              false,
		StandardWebPath:            false,
		SeparateProcessesRequested: true,
		Note:                       "order_entryのPersistence MappingからDDLを導出した使い捨てfixtureへの書込みと別processの読取。Source受理もHost許可も合成で、人的Acceptance、通常のWeb経路、生成CRUD、本番E2を示さない",
	}
}

type processInfo struct {
	ProcessID  int   `json:"processId"`
	BackendPID int64 `json:"backendPid,omitempty"`
}

type repositoryState struct {
	Revision string `json:"repositoryRevision"`
	Clean    bool   `json:"workingTreeClean"`
	PostRun  any    `json:"postRunRepository,omitempty"`
}

type safetyEvidence struct {
	GuardPassed                      bool `json:"guardPassed"`
	NonDefaultDatabaseVerified       bool `json:"nonDefaultDatabaseVerified"`
	UserRelationsBeforeFixture       int  `json:"userRelationsBeforeFixture"`
	EventTriggersBeforeFixture       int  `json:"eventTriggersBeforeFixture"`
	FixtureObjectsAbsentBeforeCreate bool `json:"fixtureObjectsAbsentBeforeCreate"`
}

type cleanupEvidence struct {
	Verified                  bool `json:"verified"`
	UserRelationsAfterCleanup int  `json:"userRelationsAfterCleanup"`
	EventTriggersAfterCleanup int  `json:"eventTriggersAfterCleanup"`
}

type prepareEvidence struct {
	DDLStatements        int    `json:"ddlStatements"`
	OwnershipMarker      string `json:"ownershipMarker"`
	SeedOrderID          int64  `json:"seedOrderId"`
	SchemaMatchesDesired bool   `json:"schemaMatchesDesiredAfterCreate"`
}

type writeEvidence struct {
	CommandID         string `json:"commandId"`
	OrderID           int64  `json:"orderId"`
	BindingIdentity   string `json:"bindingIdentity"`
	GateRefused       bool   `json:"gateRefused"`
	CommandDispatched bool   `json:"commandDispatched"`
	WriteAttempts     int    `json:"writeAttempts"`
	ResultMode        string `json:"resultMode,omitempty"`
	Failure           string `json:"failure,omitempty"`
}

type observationEvidence struct {
	ReadIdentity          string `json:"readIdentity"`
	Status                string `json:"observationStatus"`
	CommandID             string `json:"commandId"`
	OrderID               int64  `json:"orderId"`
	RowStatus             string `json:"rowStatus,omitempty"`
	TotalAmount           string `json:"totalAmount,omitempty"`
	LineCount             int    `json:"lineCount,omitempty"`
	WriteProcessID        int    `json:"writeProcessId"`
	WriteBackendPID       int64  `json:"writeBackendPid"`
	ReadBackendPID        int64  `json:"readBackendPid"`
	SourceRevisionMatches bool   `json:"sourceRevisionMatchesWrite"`
	ReadFailure           string `json:"readFailure,omitempty"`
}

type receipt struct {
	Schema              string               `json:"schema"`
	Mode                string               `json:"mode"`
	EvidenceNature      string               `json:"evidenceNature"`
	Application         string               `json:"application"`
	Repository          repositoryState      `json:"repository"`
	SourceModelRevision string               `json:"sourceModelRevision"`
	Scope               scopeStatement       `json:"scope"`
	Process             processInfo          `json:"process"`
	Safety              *safetyEvidence      `json:"fixtureDatabaseSafety,omitempty"`
	Preflight           []stageResult        `json:"preflight,omitempty"`
	Prepare             *prepareEvidence     `json:"prepare,omitempty"`
	Write               *writeEvidence       `json:"write,omitempty"`
	Observation         *observationEvidence `json:"observation,omitempty"`
	Cleanup             *cleanupEvidence     `json:"cleanup,omitempty"`
	NotExecuted         []string             `json:"notExecuted"`
}

// notExecutedは、この証拠が主張しないことを必ず添える。
func notExecuted() []string {
	return []string{
		"human-acceptance-of-source",
		"authentic-host-current-accepted-source-reader",
		"authentic-host-effect-grant",
		"standard-web-command-path-to-database",
		"generated-crud",
		"schema-migration-on-accepted-source-change",
	}
}

type observationStatus string

const (
	observationObserved               observationStatus = "observed"
	observationReadFailed             observationStatus = "read-failed"
	observationMismatch               observationStatus = "mismatch"
	observationCommandNotCommitted    observationStatus = "command-not-committed"
	observationSourceRevisionMismatch observationStatus = "source-revision-mismatch"
)

// classifyObservationは、Command結果とそれとは別に実行した読取りの結果から観測状態を決める。
// 読取りが失敗した場合、別のSource revisionで読んだ場合、行が期待と一致しない場合は、
// 決してobservedにしない。Commandの成功応答だけでは外部状態を確認済みと扱わない(N11)。
func classifyObservation(commandResultPresent bool, readErr error, sameSource, rowMatches bool) observationStatus {
	switch {
	case !commandResultPresent:
		return observationCommandNotCommitted
	case readErr != nil:
		return observationReadFailed
	case !sameSource:
		return observationSourceRevisionMismatch
	case !rowMatches:
		return observationMismatch
	default:
		return observationObserved
	}
}

// rowMatchesSubmissionは、読んだ行がsubmit後の期待と一致するかを返す。
func rowMatchesSubmission(o orderObservation) bool {
	return o.Status == "submitted" && o.TotalAmount == fixtureTotalAmount && o.LineCount == fixtureLineCount
}

// checkExpectedSourceは、実行時に読んだSourceのrevisionが、prepareで記録したものと
// 一致することを要求する。これはPhysical schemaの不一致とは別の拒否理由である(S02)。
func checkExpectedSource(expected, actual string) stageResult {
	const stage = "source-revision"
	switch {
	case expected == "":
		return stageResult{Stage: stage, Status: stageUnresolved, Detail: "SOURCE_REVISION_EXPECTATION_MISSING"}
	case expected != actual:
		return stageResult{Stage: stage, Status: stageUnresolved, Detail: "SOURCE_REVISION_MISMATCH"}
	default:
		return stageResult{Stage: stage, Status: stageResolved}
	}
}

// verifyJoinは、writeとreadの二つのreceiptが、同じ一回の試行の証拠として結合できるかを調べる。
// 結合できない理由を全て返す。空なら結合できる。手書きoverlayのreceipt、別Application、
// 別Source、同一process、観測未成立のものは結合しない。
func verifyJoin(write, read receipt) []string {
	var problems []string
	add := func(format string, args ...any) { problems = append(problems, fmt.Sprintf(format, args...)) }
	for name, r := range map[string]receipt{"write": write, "read": read} {
		if r.Schema != receiptSchema {
			add("%s receiptのschemaが%qです(期待: %s)", name, r.Schema, receiptSchema)
		}
		if r.Application != "order_entry" {
			add("%s receiptのapplicationが%qです", name, r.Application)
		}
		if r.EvidenceNature != evidenceNature {
			add("%s receiptのevidenceNatureが不正です", name)
		}
		if r.Scope.HandWrittenOverlay {
			add("%s receiptは手書きoverlayの証拠です", name)
		}
		if !r.Repository.Clean || r.Repository.Revision == "" {
			add("%s receiptのrepositoryがexact revision・cleanではありません", name)
		}
	}
	if write.Mode != "write" || write.Write == nil {
		add("write receiptにwrite証拠がありません")
	}
	if read.Mode != "read" || read.Observation == nil {
		add("read receiptにobservation証拠がありません")
	}
	if write.Write == nil || read.Observation == nil {
		return problems
	}
	if write.Write.GateRefused || !write.Write.CommandDispatched || write.Write.Failure != "" {
		add("Commandが完了していません")
	}
	if write.Write.CommandID == "" || write.Write.CommandID != read.Observation.CommandID {
		add("Command IDが一致しません")
	}
	if write.Write.OrderID != read.Observation.OrderID {
		add("注文のkeyが一致しません")
	}
	if write.SourceModelRevision == "" || write.SourceModelRevision != read.SourceModelRevision {
		add("Source revisionが一致しません")
	}
	if write.Repository.Revision != read.Repository.Revision {
		add("repository revisionが一致しません")
	}
	if write.Process.ProcessID == 0 || write.Process.ProcessID == read.Process.ProcessID {
		add("writeとreadが別processではありません")
	}
	if write.Process.BackendPID != 0 && write.Process.BackendPID == read.Process.BackendPID {
		add("writeとreadが同じDB接続です")
	}
	if read.Observation.Status != string(observationObserved) {
		add("観測状態が%qです(observedではありません)", read.Observation.Status)
	}
	return problems
}

func joinSummary(problems []string) string {
	if len(problems) == 0 {
		return "JOINABLE"
	}
	return "UNJOINABLE: " + strings.Join(problems, "; ")
}
