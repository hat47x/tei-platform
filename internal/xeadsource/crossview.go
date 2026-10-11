package xeadsource

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"
)

const CrossViewDerivationPolicyRevision = "tei.xeadsource.cross-view/v2"

const (
	CrossViewNatureSource  = "source-backed"
	CrossViewNatureDerived = "derived"
)

type CrossViewEdge struct {
	Kind                      string   `json:"kind"`
	Nature                    string   `json:"nature"`
	FromToken                 string   `json:"fromToken"`
	ToToken                   string   `json:"toToken"`
	TargetKind                string   `json:"targetKind,omitempty"`
	Operations                string   `json:"operations,omitempty"`
	SourceTokens              []string `json:"sourceTokens"`
	DerivationRuleFingerprint string   `json:"derivationRuleFingerprint,omitempty"`
	AuthorityLimits           []string `json:"authorityLimits"`
}

type CrossViewActionObservation struct {
	ActionToken               string   `json:"actionToken"`
	TaskToken                 string   `json:"taskToken,omitempty"`
	FunctionIOCount           int      `json:"functionIoCount"`
	MissingLegacyID           bool     `json:"missingLegacyId"`
	ManualOrExternalCandidate bool     `json:"manualOrExternalCandidate"`
	AuthorityLimits           []string `json:"authorityLimits"`
}

type CrossViewUnresolved struct {
	Code         string   `json:"code"`
	RelationKind string   `json:"relationKind"`
	SourceTokens []string `json:"sourceTokens"`
}

type CrossViewProjection struct {
	SchemaVersion int `json:"schemaVersion"`
	SourceFormat string `json:"sourceFormat"`
	SourceFormatVersion string `json:"sourceFormatVersion"`
	SourceArtifactDigest string `json:"sourceArtifactDigest"`
	MappingContractFingerprint string `json:"mappingContractFingerprint"`
	DerivationPolicyRevision string `json:"derivationPolicyRevision"`
	DerivationPolicyFingerprint string `json:"derivationPolicyFingerprint"`
	TaskCount int `json:"taskCount"`
	TaskActionCount int `json:"taskActionCount"`
	TaskActionWithoutFunctionIOCount int `json:"taskActionWithoutFunctionIoCount"`
	TaskActionMissingIDCount int `json:"taskActionMissingIdCount"`
	TaskFunctionIOCount int `json:"taskFunctionIoCount"`
	TaskFunctionIONoTargetCount int `json:"taskFunctionIoNoTargetCount"`
	DistinctActionFunctionPairCount int `json:"distinctActionFunctionPairCount"`
	IOTableCount int `json:"ioTableCount"`
	PotentialActionFunctionTablePathCount int `json:"potentialActionFunctionTablePathCount"`
	TaskFunctionIOTargetKindCounts map[string]int `json:"taskFunctionIoTargetKindCounts,omitempty"`
	PotentialOperationCombinationCounts map[string]int `json:"potentialOperationCombinationCounts,omitempty"`
	SourceEdgeCounts map[string]int `json:"sourceEdgeCounts"`
	DerivedEdgeCounts map[string]int `json:"derivedEdgeCounts"`
	UnresolvedCounts map[string]int `json:"unresolvedCounts,omitempty"`
	AuthorityLimits []string `json:"authorityLimits"`

}

type crossViewNode struct {
	node          Node
	token         string
	taskToken     string
	actionToken   string
	functionToken string
}

type actionFunctionPair struct {
	actionToken   string
	functionToken string
	sourceTokens  []string
}

var crossViewFingerprintPattern = regexp.MustCompile(`^sha256:[0-9a-f]{64}$`)

func BuildCrossViewProjection(
	format Format,
	source []byte,
	contractBytes []byte,
) (CrossViewProjection, error) {
	inspection, err := InspectWithContract(format, source, contractBytes)
	if err != nil {
		return CrossViewProjection{}, err
	}
	return buildCrossViewProjection(inspection)
}

func buildCrossViewProjection(inspection Inspection) (CrossViewProjection, error) {
	doc := inspection.Document
	mappingContractFingerprint, err := normalizeCrossViewContractFingerprint(inspection.TranscriptionContractFingerprint)
	if err != nil {
		return CrossViewProjection{}, err
	}
	if doc.Format != FormatXead {
		return CrossViewProjection{}, fmt.Errorf("cross-view projection currently supports xead only: %s", doc.Format)
	}
	if doc.SHA256 == "" {
		return CrossViewProjection{}, fmt.Errorf("source document digest is empty")
	}
	sourceFormatVersion, ok := nonEmptyExactAttribute(doc.Root, "FormatVersion")
	if !ok {
		return CrossViewProjection{}, fmt.Errorf("validated source document has no usable FormatVersion")
	}

	nodes := flattenCrossViewNodes(doc)
	functionsByID := map[string][]crossViewNode{}
	tablesByID := map[string][]crossViewNode{}
	targetsByFunction := map[string]map[string][]crossViewNode{}
	actions := map[string]*CrossViewActionObservation{}
	taskCount := 0
	taskFunctionIOCount := 0
	taskFunctionIONoTargetCount := 0
	ioTableCount := 0

	for _, ref := range nodes {
		if !ref.node.Known {
			continue
		}
		switch ref.node.Name {
		case "Task":
			taskCount++
		case "TaskAction":
			_, hasID := nonEmptyExactAttribute(ref.node, "ID")
			actions[ref.token] = &CrossViewActionObservation{
				ActionToken:               ref.token,
				TaskToken:                 ref.taskToken,
				MissingLegacyID:           !hasID,
				ManualOrExternalCandidate: true,
				AuthorityLimits: []string{
					"source-token-is-not-canonical-business-identity",
					"missing-legacy-id-must-not-be-repaired-from-sortkey-or-order",
					"absence-of-function-io-is-only-manual-or-external-candidate-evidence",
				},
			}
		case "TaskFunctionIO":
			taskFunctionIOCount++
			if ioID, ok := exactAttribute(ref.node, "IOID"); ok && strings.TrimSpace(ioID) == "0" {
				taskFunctionIONoTargetCount++
			}
			if action := actions[ref.actionToken]; action != nil {
				action.FunctionIOCount++
				action.ManualOrExternalCandidate = false
			}
		case "Function":
			if id, ok := nonEmptyExactAttribute(ref.node, "ID"); ok {
				functionsByID[id] = append(functionsByID[id], ref)
			}
		case "Table":
			if id, ok := nonEmptyExactAttribute(ref.node, "ID"); ok {
				tablesByID[id] = append(tablesByID[id], ref)
			}
		case "IOPanel", "IOSpool", "IOTable":
			if ref.node.Name == "IOTable" {
				ioTableCount++
			}
			if ref.functionToken == "" {
				continue
			}
			if id, ok := nonEmptyExactAttribute(ref.node, "ID"); ok {
				if targetsByFunction[ref.functionToken] == nil {
					targetsByFunction[ref.functionToken] = map[string][]crossViewNode{}
				}
				targetsByFunction[ref.functionToken][id] = append(targetsByFunction[ref.functionToken][id], ref)
			}
		}
	}

	projection := CrossViewProjection{
		SchemaVersion:               1,
		SourceFormat:                string(doc.Format),
		SourceFormatVersion:         sourceFormatVersion,
		SourceArtifactDigest:        "sha256:" + doc.SHA256,
		MappingContractFingerprint:  mappingContractFingerprint,
		DerivationPolicyRevision:    CrossViewDerivationPolicyRevision,
		DerivationPolicyFingerprint: crossViewDerivationPolicyFingerprint(),
		TaskCount:                   taskCount,
		TaskFunctionIOCount:         taskFunctionIOCount,
		TaskFunctionIONoTargetCount: taskFunctionIONoTargetCount,
		IOTableCount:                ioTableCount,
		AuthorityLimits: []string{
			"cross-view-projection-is-not-canonical-authority",
			"source-reachability-is-not-runtime-effect",
			"source-reachability-is-not-transaction-authority",
			"source-reachability-is-not-authorization-or-business-ownership",
			"derived-path-is-not-business-action-specific-crud",
			"source-profile-conformance-does-not-prove-semantic-correctness",
		},
	}

	for _, ref := range nodes {
		if !ref.node.Known {
			continue
		}
		switch ref.node.Name {
		case "TaskAction":
			if ref.taskToken == "" {
				projection.Unresolved = append(projection.Unresolved, unresolvedCrossView(
					"task-action-without-task",
					"task-action",
					[]string{ref.token},
				))
				continue
			}
			projection.SourceEdges = append(projection.SourceEdges, CrossViewEdge{
				Kind:         "task-action",
				Nature:       CrossViewNatureSource,
				FromToken:    ref.taskToken,
				ToToken:      ref.token,
				SourceTokens: normalizedTokens(ref.taskToken, ref.token),
				AuthorityLimits: []string{
					"source-structure-is-not-canonical-business-identity",
					"task-order-is-not-runtime-transaction-order",
				},
			})

		case "TaskFunctionIO":
			if ref.actionToken == "" {
				projection.Unresolved = append(projection.Unresolved, unresolvedCrossView(
					"task-function-io-without-task-action",
					"action-function-use",
					[]string{ref.token},
				))
				continue
			}
			functionID, ok := nonEmptyExactAttribute(ref.node, "FunctionID")
			if !ok {
				projection.Unresolved = append(projection.Unresolved, unresolvedCrossView(
					"task-function-io-function-id-missing",
					"action-function-use",
					[]string{ref.actionToken, ref.token},
				))
				continue
			}
			functions := functionsByID[functionID]
			if len(functions) != 1 {
				code := "task-function-io-function-not-found"
				if len(functions) > 1 {
					code = "task-function-io-function-ambiguous"
				}
				projection.Unresolved = append(projection.Unresolved, unresolvedCrossView(
					code,
					"action-function-use",
					[]string{ref.actionToken, ref.token},
				))
				continue
			}
			function := functions[0]
			projection.SourceEdges = append(projection.SourceEdges, CrossViewEdge{
				Kind:         "action-function-use",
				Nature:       CrossViewNatureSource,
				FromToken:    ref.actionToken,
				ToToken:      function.token,
				SourceTokens: normalizedTokens(ref.actionToken, ref.token, function.token),
				AuthorityLimits: []string{
					"function-use-is-not-business-action-effect",
					"taskfunctionio-operations-text-is-not-business-action-semantics",
				},
			})

			ioID, hasIOID := exactAttribute(ref.node, "IOID")
			ioID = strings.TrimSpace(ioID)
			if !hasIOID || ioID == "" {
				projection.Unresolved = append(projection.Unresolved, unresolvedCrossView(
					"task-function-io-target-id-missing",
					"task-function-io-target",
					[]string{ref.token, function.token},
				))
				continue
			}
			if ioID == "0" {
				continue
			}
			targets := targetsByFunction[function.token][ioID]
			if len(targets) != 1 {
				code := "task-function-io-target-not-found"
				if len(targets) > 1 {
					code = "task-function-io-target-ambiguous"
				}
				projection.Unresolved = append(projection.Unresolved, unresolvedCrossView(
					code,
					"task-function-io-target",
					[]string{ref.token, function.token},
				))
				continue
			}
			target := targets[0]
			projection.SourceEdges = append(projection.SourceEdges, CrossViewEdge{
				Kind:         "task-function-io-target",
				Nature:       CrossViewNatureSource,
				FromToken:    ref.token,
				ToToken:      target.token,
				TargetKind:   target.node.Name,
				SourceTokens: normalizedTokens(ref.token, function.token, target.token),
				AuthorityLimits: []string{
					"target-interaction-is-not-business-action-semantics",
					"io-target-is-not-business-data-effect",
				},
			})

		case "IOTable":
			if ref.functionToken == "" {
				projection.Unresolved = append(projection.Unresolved, unresolvedCrossView(
					"iotable-without-function",
					"function-table-operation",
					[]string{ref.token},
				))
				continue
			}
			tableID, ok := nonEmptyExactAttribute(ref.node, "TableID")
			if !ok {
				projection.Unresolved = append(projection.Unresolved, unresolvedCrossView(
					"iotable-table-id-missing",
					"function-table-operation",
					[]string{ref.functionToken, ref.token},
				))
				continue
			}
			tables := tablesByID[tableID]
			if len(tables) != 1 {
				code := "iotable-table-not-found"
				if len(tables) > 1 {
					code = "iotable-table-ambiguous"
				}
				projection.Unresolved = append(projection.Unresolved, unresolvedCrossView(
					code,
					"function-table-operation",
					[]string{ref.functionToken, ref.token},
				))
				continue
			}
			operations, invalidOps := iotableOperations(ref.node)
			if len(invalidOps) != 0 {
				projection.Unresolved = append(projection.Unresolved, unresolvedCrossView(
					"iotable-operation-marker-unsupported",
					"function-table-operation",
					[]string{ref.functionToken, ref.token, tables[0].token},
				))
				continue
			}
			if operations == "" {
				projection.Unresolved = append(projection.Unresolved, unresolvedCrossView(
					"iotable-operation-missing",
					"function-table-operation",
					[]string{ref.functionToken, ref.token, tables[0].token},
				))
				continue
			}
			projection.SourceEdges = append(projection.SourceEdges, CrossViewEdge{
				Kind:         "function-table-operation",
				Nature:       CrossViewNatureSource,
				FromToken:    ref.functionToken,
				ToToken:      tables[0].token,
				Operations:   operations,
				SourceTokens: normalizedTokens(ref.functionToken, ref.token, tables[0].token),
				AuthorityLimits: []string{
					"function-wide-iotable-is-potential-access-not-business-action-specific-effect",
					"declared-crud-is-not-actual-runtime-effect",
				},
			})
		}
	}

	pairs := actionFunctionPairs(projection.SourceEdges)
	tableEdgesByFunction := map[string][]CrossViewEdge{}
	for _, edge := range projection.SourceEdges {
		if edge.Kind == "function-table-operation" {
			tableEdgesByFunction[edge.FromToken] = append(tableEdgesByFunction[edge.FromToken], edge)
		}
	}
	for _, pair := range pairs {
		for _, tableEdge := range tableEdgesByFunction[pair.functionToken] {
			projection.DerivedEdges = append(projection.DerivedEdges, CrossViewEdge{
				Kind:                      "action-function-table-potential-operation",
				Nature:                    CrossViewNatureDerived,
				FromToken:                 pair.actionToken,
				ToToken:                   tableEdge.ToToken,
				Operations:                tableEdge.Operations,
				SourceTokens:              normalizedTokens(append(pair.sourceTokens, tableEdge.SourceTokens...)...),
				DerivationRuleFingerprint: projection.DerivationPolicyFingerprint,
				AuthorityLimits: []string{
					"derived-path-is-not-business-action-specific-effect",
					"function-wide-iotable-is-potential-access-not-business-action-specific-effect",
					"derived-path-is-not-runtime-observation",
					"derived-path-is-not-transaction-or-authorization-evidence",
				},
			})
		}
	}

	for _, action := range actions {
		projection.Actions = append(projection.Actions, *action)
	}
	sortCrossViewProjection(&projection)
	copyProjection := projection
	copyProjection.ProjectionFingerprint = ""
	projection.ProjectionFingerprint = crossViewFingerprint(copyProjection)
	return projection, nil
}

func CrossViewPublicAccounting(projection CrossViewProjection) (CrossViewAccounting, error) {
	if err := validateCrossViewProjectionForPublication(projection); err != nil {
		return CrossViewAccounting{}, err
	}
	out := CrossViewAccounting{
		SchemaVersion:                       1,
		SourceFormat:                        projection.SourceFormat,
		SourceFormatVersion:                 projection.SourceFormatVersion,
		SourceArtifactDigest:                projection.SourceArtifactDigest,
		MappingContractFingerprint:          projection.MappingContractFingerprint,
		DerivationPolicyRevision:            projection.DerivationPolicyRevision,
		DerivationPolicyFingerprint:         projection.DerivationPolicyFingerprint,
		ProjectionFingerprint:               projection.ProjectionFingerprint,
		TaskFunctionIOTargetKindCounts:      map[string]int{},
		PotentialOperationCombinationCounts: map[string]int{},
		SourceEdgeCounts:                    map[string]int{},
		DerivedEdgeCounts:                   map[string]int{},
		UnresolvedCounts:                    map[string]int{},
		AuthorityLimits: []string{
			"public-accounting-does-not-publish-individual-cross-view-relations",
			"aggregate-count-is-not-semantic-equivalence",
			"potential-operation-count-is-not-action-specific-runtime-effect",
			"benchmark-observation-is-not-universal-xtea-rule",
		},
	}
	actionFunctionPairs := map[string]struct{}{}
	for _, action := range projection.Actions {
		out.TaskActionCount++
		if action.FunctionIOCount == 0 {
			out.TaskActionWithoutFunctionIOCount++
		}
		if action.MissingLegacyID {
			out.TaskActionMissingIDCount++
		}
	}
	out.TaskCount = projection.TaskCount
	out.TaskFunctionIOCount = projection.TaskFunctionIOCount
	out.TaskFunctionIONoTargetCount = projection.TaskFunctionIONoTargetCount
	out.IOTableCount = projection.IOTableCount
	for _, edge := range projection.SourceEdges {
		out.SourceEdgeCounts[edge.Kind]++
		switch edge.Kind {
		case "action-function-use":
			actionFunctionPairs[edge.FromToken+"\x00"+edge.ToToken] = struct{}{}
		case "task-function-io-target":
			out.TaskFunctionIOTargetKindCounts[edge.TargetKind]++
		}
	}
	out.DistinctActionFunctionPairCount = len(actionFunctionPairs)
	for _, edge := range projection.DerivedEdges {
		out.DerivedEdgeCounts[edge.Kind]++
		if edge.Kind == "action-function-table-potential-operation" {
			out.PotentialActionFunctionTablePathCount++
			out.PotentialOperationCombinationCounts[edge.Operations]++
		}
	}
	for _, unresolved := range projection.Unresolved {
		out.UnresolvedCounts[unresolved.Code]++
	}
	if len(out.TaskFunctionIOTargetKindCounts) == 0 {
		out.TaskFunctionIOTargetKindCounts = nil
	}
	if len(out.PotentialOperationCombinationCounts) == 0 {
		out.PotentialOperationCombinationCounts = nil
	}
	if len(out.UnresolvedCounts) == 0 {
		out.UnresolvedCounts = nil
	}
	return out, nil
}

func validateCrossViewProjectionForPublication(projection CrossViewProjection) error {
	if projection.SourceFormat != string(FormatXead) {
		return fmt.Errorf("cross-view public accounting sourceFormat %q is unsupported", projection.SourceFormat)
	}
	if !regexp.MustCompile(`^[0-9]+(?:\\.[0-9]+){1,3}package xeadsource

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"
)

const CrossViewDerivationPolicyRevision = "tei.xeadsource.cross-view/v2"

const (
	CrossViewNatureSource  = "source-backed"
	CrossViewNatureDerived = "derived"
)

type CrossViewEdge struct {
	Kind                      string   `json:"kind"`
	Nature                    string   `json:"nature"`
	FromToken                 string   `json:"fromToken"`
	ToToken                   string   `json:"toToken"`
	TargetKind                string   `json:"targetKind,omitempty"`
	Operations                string   `json:"operations,omitempty"`
	SourceTokens              []string `json:"sourceTokens"`
	DerivationRuleFingerprint string   `json:"derivationRuleFingerprint,omitempty"`
	AuthorityLimits           []string `json:"authorityLimits"`
}

type CrossViewActionObservation struct {
	ActionToken               string   `json:"actionToken"`
	TaskToken                 string   `json:"taskToken,omitempty"`
	FunctionIOCount           int      `json:"functionIoCount"`
	MissingLegacyID           bool     `json:"missingLegacyId"`
	ManualOrExternalCandidate bool     `json:"manualOrExternalCandidate"`
	AuthorityLimits           []string `json:"authorityLimits"`
}

type CrossViewUnresolved struct {
	Code         string   `json:"code"`
	RelationKind string   `json:"relationKind"`
	SourceTokens []string `json:"sourceTokens"`
}

type CrossViewProjection struct {
	SchemaVersion                 int                          `json:"schemaVersion"`
	SourceFormat                  string                       `json:"sourceFormat"`
	SourceFormatVersion           string                       `json:"sourceFormatVersion"`
	SourceArtifactDigest          string                       `json:"sourceArtifactDigest"`
	MappingContractFingerprint    string                       `json:"mappingContractFingerprint"`
	DerivationPolicyRevision      string                       `json:"derivationPolicyRevision"`
	DerivationPolicyFingerprint   string                       `json:"derivationPolicyFingerprint"`
	TaskCount                     int                          `json:"taskCount"`
	TaskFunctionIOCount           int                          `json:"taskFunctionIoCount"`
	TaskFunctionIONoTargetCount   int                          `json:"taskFunctionIoNoTargetCount"`
	IOTableCount                  int                          `json:"ioTableCount"`
	SourceEdges                   []CrossViewEdge              `json:"sourceEdges"`
	DerivedEdges                  []CrossViewEdge              `json:"derivedEdges"`
	Actions                       []CrossViewActionObservation `json:"actions"`
	Unresolved                    []CrossViewUnresolved        `json:"unresolved,omitempty"`
	ProjectionFingerprint         string                       `json:"projectionFingerprint"`
	AuthorityLimits               []string                     `json:"authorityLimits"`
}

type CrossViewAccounting struct {
	SchemaVersion                              int            `json:"schemaVersion"`
	SourceFormat                               string         `json:"sourceFormat"`
	SourceFormatVersion                        string         `json:"sourceFormatVersion"`
	SourceArtifactDigest                       string         `json:"sourceArtifactDigest"`
	MappingContractFingerprint                 string         `json:"mappingContractFingerprint"`
	DerivationPolicyRevision                   string         `json:"derivationPolicyRevision"`
	DerivationPolicyFingerprint                string         `json:"derivationPolicyFingerprint"`
	ProjectionFingerprint                      string         `json:"projectionFingerprint"`
	TaskCount                                  int            `json:"taskCount"`
	TaskActionCount                            int            `json:"taskActionCount"`
	TaskActionWithoutFunctionIOCount           int            `json:"taskActionWithoutFunctionIoCount"`
	TaskActionMissingIDCount                   int            `json:"taskActionMissingIdCount"`
	TaskFunctionIOCount                        int            `json:"taskFunctionIoCount"`
	TaskFunctionIONoTargetCount                int            `json:"taskFunctionIoNoTargetCount"`
	DistinctActionFunctionPairCount            int            `json:"distinctActionFunctionPairCount"`
	IOTableCount                               int            `json:"ioTableCount"`
	PotentialActionFunctionTablePathCount      int            `json:"potentialActionFunctionTablePathCount"`
	TaskFunctionIOTargetKindCounts             map[string]int `json:"taskFunctionIoTargetKindCounts,omitempty"`
	PotentialOperationCombinationCounts        map[string]int `json:"potentialOperationCombinationCounts,omitempty"`
	SourceEdgeCounts                           map[string]int `json:"sourceEdgeCounts"`
	DerivedEdgeCounts                          map[string]int `json:"derivedEdgeCounts"`
	UnresolvedCounts                           map[string]int `json:"unresolvedCounts,omitempty"`
	AuthorityLimits                            []string        `json:"authorityLimits"`
}

type crossViewNode struct {
	node          Node
	token         string
	taskToken     string
	actionToken   string
	functionToken string
}

type actionFunctionPair struct {
	actionToken  string
	functionToken string
	sourceTokens []string
}

var crossViewFingerprintPattern = regexp.MustCompile(`^sha256:[0-9a-f]{64}$`)

func BuildCrossViewProjection(
	format Format,
	source []byte,
	contractBytes []byte,
) (CrossViewProjection, error) {
	inspection, err := InspectWithContract(format, source, contractBytes)
	if err != nil {
		return CrossViewProjection{}, err
	}
	return buildCrossViewProjection(inspection)
}

func buildCrossViewProjection(inspection Inspection) (CrossViewProjection, error) {
	doc := inspection.Document
	mappingContractFingerprint, err := normalizeCrossViewContractFingerprint(inspection.TranscriptionContractFingerprint)
	if err != nil {
		return CrossViewProjection{}, err
	}
	if doc.Format != FormatXead {
		return CrossViewProjection{}, fmt.Errorf("cross-view projection currently supports xead only: %s", doc.Format)
	}
	if doc.SHA256 == "" {
		return CrossViewProjection{}, fmt.Errorf("source document digest is empty")
	}
	sourceFormatVersion, ok := nonEmptyExactAttribute(doc.Root, "FormatVersion")
	if !ok {
		return CrossViewProjection{}, fmt.Errorf("validated source document has no usable FormatVersion")
	}

	nodes := flattenCrossViewNodes(doc)
	functionsByID := map[string][]crossViewNode{}
	tablesByID := map[string][]crossViewNode{}
	targetsByFunction := map[string]map[string][]crossViewNode{}
	actions := map[string]*CrossViewActionObservation{}
	taskCount := 0
	taskFunctionIOCount := 0
	taskFunctionIONoTargetCount := 0
	ioTableCount := 0

	for _, ref := range nodes {
		if !ref.node.Known {
			continue
		}
		switch ref.node.Name {
		case "Task":
			taskCount++
		case "TaskAction":
			_, hasID := nonEmptyExactAttribute(ref.node, "ID")
			actions[ref.token] = &CrossViewActionObservation{
				ActionToken:               ref.token,
				TaskToken:                 ref.taskToken,
				MissingLegacyID:           !hasID,
				ManualOrExternalCandidate: true,
				AuthorityLimits: []string{
					"source-token-is-not-canonical-business-identity",
					"missing-legacy-id-must-not-be-repaired-from-sortkey-or-order",
					"absence-of-function-io-is-only-manual-or-external-candidate-evidence",
				},
			}
		case "TaskFunctionIO":
			taskFunctionIOCount++
			if ioID, ok := exactAttribute(ref.node, "IOID"); ok && strings.TrimSpace(ioID) == "0" {
				taskFunctionIONoTargetCount++
			}
			if action := actions[ref.actionToken]; action != nil {
				action.FunctionIOCount++
				action.ManualOrExternalCandidate = false
			}
		case "Function":
			if id, ok := nonEmptyExactAttribute(ref.node, "ID"); ok {
				functionsByID[id] = append(functionsByID[id], ref)
			}
		case "Table":
			if id, ok := nonEmptyExactAttribute(ref.node, "ID"); ok {
				tablesByID[id] = append(tablesByID[id], ref)
			}
		case "IOPanel", "IOSpool", "IOTable":
			if ref.node.Name == "IOTable" {
				ioTableCount++
			}
			if ref.functionToken == "" {
				continue
			}
			if id, ok := nonEmptyExactAttribute(ref.node, "ID"); ok {
				if targetsByFunction[ref.functionToken] == nil {
					targetsByFunction[ref.functionToken] = map[string][]crossViewNode{}
				}
				targetsByFunction[ref.functionToken][id] = append(targetsByFunction[ref.functionToken][id], ref)
			}
		}
	}

	projection := CrossViewProjection{
		SchemaVersion:               1,
		SourceFormat:                string(doc.Format),
		SourceFormatVersion:         sourceFormatVersion,
		SourceArtifactDigest:        "sha256:" + doc.SHA256,
		MappingContractFingerprint:  mappingContractFingerprint,
		DerivationPolicyRevision:    CrossViewDerivationPolicyRevision,
		DerivationPolicyFingerprint: crossViewDerivationPolicyFingerprint(),
		TaskCount:                   taskCount,
		TaskFunctionIOCount:         taskFunctionIOCount,
		TaskFunctionIONoTargetCount: taskFunctionIONoTargetCount,
		IOTableCount:                ioTableCount,
		AuthorityLimits: []string{
			"cross-view-projection-is-not-canonical-authority",
			"source-reachability-is-not-runtime-effect",
			"source-reachability-is-not-transaction-authority",
			"source-reachability-is-not-authorization-or-business-ownership",
			"derived-path-is-not-business-action-specific-crud",
			"source-profile-conformance-does-not-prove-semantic-correctness",
		},
	}

	for _, ref := range nodes {
		if !ref.node.Known {
			continue
		}
		switch ref.node.Name {
		case "TaskAction":
			if ref.taskToken == "" {
				projection.Unresolved = append(projection.Unresolved, unresolvedCrossView(
					"task-action-without-task",
					"task-action",
					[]string{ref.token},
				))
				continue
			}
			projection.SourceEdges = append(projection.SourceEdges, CrossViewEdge{
				Kind:         "task-action",
				Nature:       CrossViewNatureSource,
				FromToken:    ref.taskToken,
				ToToken:      ref.token,
				SourceTokens: normalizedTokens(ref.taskToken, ref.token),
				AuthorityLimits: []string{
					"source-structure-is-not-canonical-business-identity",
					"task-order-is-not-runtime-transaction-order",
				},
			})

		case "TaskFunctionIO":
			if ref.actionToken == "" {
				projection.Unresolved = append(projection.Unresolved, unresolvedCrossView(
					"task-function-io-without-task-action",
					"action-function-use",
					[]string{ref.token},
				))
				continue
			}
			functionID, ok := nonEmptyExactAttribute(ref.node, "FunctionID")
			if !ok {
				projection.Unresolved = append(projection.Unresolved, unresolvedCrossView(
					"task-function-io-function-id-missing",
					"action-function-use",
					[]string{ref.actionToken, ref.token},
				))
				continue
			}
			functions := functionsByID[functionID]
			if len(functions) != 1 {
				code := "task-function-io-function-not-found"
				if len(functions) > 1 {
					code = "task-function-io-function-ambiguous"
				}
				projection.Unresolved = append(projection.Unresolved, unresolvedCrossView(
					code,
					"action-function-use",
					[]string{ref.actionToken, ref.token},
				))
				continue
			}
			function := functions[0]
			projection.SourceEdges = append(projection.SourceEdges, CrossViewEdge{
				Kind:         "action-function-use",
				Nature:       CrossViewNatureSource,
				FromToken:    ref.actionToken,
				ToToken:      function.token,
				SourceTokens: normalizedTokens(ref.actionToken, ref.token, function.token),
				AuthorityLimits: []string{
					"function-use-is-not-business-action-effect",
					"taskfunctionio-operations-text-is-not-business-action-semantics",
				},
			})

			ioID, hasIOID := exactAttribute(ref.node, "IOID")
			ioID = strings.TrimSpace(ioID)
			if !hasIOID || ioID == "" {
				projection.Unresolved = append(projection.Unresolved, unresolvedCrossView(
					"task-function-io-target-id-missing",
					"task-function-io-target",
					[]string{ref.token, function.token},
				))
				continue
			}
			if ioID == "0" {
				continue
			}
			targets := targetsByFunction[function.token][ioID]
			if len(targets) != 1 {
				code := "task-function-io-target-not-found"
				if len(targets) > 1 {
					code = "task-function-io-target-ambiguous"
				}
				projection.Unresolved = append(projection.Unresolved, unresolvedCrossView(
					code,
					"task-function-io-target",
					[]string{ref.token, function.token},
				))
				continue
			}
			target := targets[0]
			projection.SourceEdges = append(projection.SourceEdges, CrossViewEdge{
				Kind:         "task-function-io-target",
				Nature:       CrossViewNatureSource,
				FromToken:    ref.token,
				ToToken:      target.token,
				TargetKind:   target.node.Name,
				SourceTokens: normalizedTokens(ref.token, function.token, target.token),
				AuthorityLimits: []string{
					"target-interaction-is-not-business-action-semantics",
					"io-target-is-not-business-data-effect",
				},
			})

		case "IOTable":
			if ref.functionToken == "" {
				projection.Unresolved = append(projection.Unresolved, unresolvedCrossView(
					"iotable-without-function",
					"function-table-operation",
					[]string{ref.token},
				))
				continue
			}
			tableID, ok := nonEmptyExactAttribute(ref.node, "TableID")
			if !ok {
				projection.Unresolved = append(projection.Unresolved, unresolvedCrossView(
					"iotable-table-id-missing",
					"function-table-operation",
					[]string{ref.functionToken, ref.token},
				))
				continue
			}
			tables := tablesByID[tableID]
			if len(tables) != 1 {
				code := "iotable-table-not-found"
				if len(tables) > 1 {
					code = "iotable-table-ambiguous"
				}
				projection.Unresolved = append(projection.Unresolved, unresolvedCrossView(
					code,
					"function-table-operation",
					[]string{ref.functionToken, ref.token},
				))
				continue
			}
			operations, invalidOps := iotableOperations(ref.node)
			if len(invalidOps) != 0 {
				projection.Unresolved = append(projection.Unresolved, unresolvedCrossView(
					"iotable-operation-marker-unsupported",
					"function-table-operation",
					[]string{ref.functionToken, ref.token, tables[0].token},
				))
				continue
			}
			if operations == "" {
				projection.Unresolved = append(projection.Unresolved, unresolvedCrossView(
					"iotable-operation-missing",
					"function-table-operation",
					[]string{ref.functionToken, ref.token, tables[0].token},
				))
				continue
			}
			projection.SourceEdges = append(projection.SourceEdges, CrossViewEdge{
				Kind:         "function-table-operation",
				Nature:       CrossViewNatureSource,
				FromToken:    ref.functionToken,
				ToToken:      tables[0].token,
				Operations:   operations,
				SourceTokens: normalizedTokens(ref.functionToken, ref.token, tables[0].token),
				AuthorityLimits: []string{
					"function-wide-iotable-is-potential-access-not-business-action-specific-effect",
					"declared-crud-is-not-actual-runtime-effect",
				},
			})
		}
	}

	pairs := actionFunctionPairs(projection.SourceEdges)
	tableEdgesByFunction := map[string][]CrossViewEdge{}
	for _, edge := range projection.SourceEdges {
		if edge.Kind == "function-table-operation" {
			tableEdgesByFunction[edge.FromToken] = append(tableEdgesByFunction[edge.FromToken], edge)
		}
	}
	for _, pair := range pairs {
		for _, tableEdge := range tableEdgesByFunction[pair.functionToken] {
			projection.DerivedEdges = append(projection.DerivedEdges, CrossViewEdge{
				Kind:                      "action-function-table-potential-operation",
				Nature:                    CrossViewNatureDerived,
				FromToken:                 pair.actionToken,
				ToToken:                   tableEdge.ToToken,
				Operations:                tableEdge.Operations,
				SourceTokens:              normalizedTokens(append(pair.sourceTokens, tableEdge.SourceTokens...)...),
				DerivationRuleFingerprint: projection.DerivationPolicyFingerprint,
				AuthorityLimits: []string{
					"derived-path-is-not-business-action-specific-effect",
					"function-wide-iotable-is-potential-access-not-business-action-specific-effect",
					"derived-path-is-not-runtime-observation",
					"derived-path-is-not-transaction-or-authorization-evidence",
				},
			})
		}
	}

	for _, action := range actions {
		projection.Actions = append(projection.Actions, *action)
	}
	sortCrossViewProjection(&projection)
	copyProjection := projection
	copyProjection.ProjectionFingerprint = ""
	projection.ProjectionFingerprint = crossViewFingerprint(copyProjection)
	return projection, nil
}

func CrossViewPublicAccounting(projection CrossViewProjection) (CrossViewAccounting, error) {
	if err := validateCrossViewProjectionForPublication(projection); err != nil {
		return CrossViewAccounting{}, err
	}
	out := CrossViewAccounting{
		SchemaVersion:                       1,
		SourceFormat:                        projection.SourceFormat,
		SourceFormatVersion:                 projection.SourceFormatVersion,
		SourceArtifactDigest:                projection.SourceArtifactDigest,
		MappingContractFingerprint:          projection.MappingContractFingerprint,
		DerivationPolicyRevision:            projection.DerivationPolicyRevision,
		DerivationPolicyFingerprint:         projection.DerivationPolicyFingerprint,
		ProjectionFingerprint:               projection.ProjectionFingerprint,
		TaskFunctionIOTargetKindCounts:      map[string]int{},
		PotentialOperationCombinationCounts: map[string]int{},
		SourceEdgeCounts:                    map[string]int{},
		DerivedEdgeCounts:                   map[string]int{},
		UnresolvedCounts:                    map[string]int{},
		AuthorityLimits: []string{
			"public-accounting-does-not-publish-individual-cross-view-relations",
			"aggregate-count-is-not-semantic-equivalence",
			"potential-operation-count-is-not-action-specific-runtime-effect",
			"benchmark-observation-is-not-universal-xtea-rule",
		},
	}
	actionFunctionPairs := map[string]struct{}{}
	for _, action := range projection.Actions {
		out.TaskActionCount++
		if action.FunctionIOCount == 0 {
			out.TaskActionWithoutFunctionIOCount++
		}
		if action.MissingLegacyID {
			out.TaskActionMissingIDCount++
		}
	}
	out.TaskCount = projection.TaskCount
	out.TaskFunctionIOCount = projection.TaskFunctionIOCount
	out.TaskFunctionIONoTargetCount = projection.TaskFunctionIONoTargetCount
	out.IOTableCount = projection.IOTableCount
	for _, edge := range projection.SourceEdges {
		out.SourceEdgeCounts[edge.Kind]++
		switch edge.Kind {
		case "action-function-use":
			actionFunctionPairs[edge.FromToken+"\x00"+edge.ToToken] = struct{}{}
		case "task-function-io-target":
			out.TaskFunctionIOTargetKindCounts[edge.TargetKind]++
		}
	}
	out.DistinctActionFunctionPairCount = len(actionFunctionPairs)
	for _, edge := range projection.DerivedEdges {
		out.DerivedEdgeCounts[edge.Kind]++
		if edge.Kind == "action-function-table-potential-operation" {
			out.PotentialActionFunctionTablePathCount++
			out.PotentialOperationCombinationCounts[edge.Operations]++
		}
	}
	for _, unresolved := range projection.Unresolved {
		out.UnresolvedCounts[unresolved.Code]++
	}
	if len(out.TaskFunctionIOTargetKindCounts) == 0 {
		out.TaskFunctionIOTargetKindCounts = nil
	}
	if len(out.PotentialOperationCombinationCounts) == 0 {
		out.PotentialOperationCombinationCounts = nil
	}
	if len(out.UnresolvedCounts) == 0 {
		out.UnresolvedCounts = nil
	}
).MatchString(projection.SourceFormatVersion) {
		return fmt.Errorf("cross-view public accounting sourceFormatVersion %q is not a bounded numeric version", projection.SourceFormatVersion)
	}
	for name, value := range map[string]string{
		"sourceArtifactDigest":       projection.SourceArtifactDigest,
		"mappingContractFingerprint": projection.MappingContractFingerprint,
		"derivationPolicyFingerprint": projection.DerivationPolicyFingerprint,
		"projectionFingerprint":       projection.ProjectionFingerprint,
	} {
		if !crossViewFingerprintPattern.MatchString(value) {
			return fmt.Errorf("cross-view public accounting %s %q is not exact sha256", name, value)
		}
	}
	if projection.DerivationPolicyRevision != CrossViewDerivationPolicyRevision {
		return fmt.Errorf(
			"cross-view derivation policy revision %q is not current %q",
			projection.DerivationPolicyRevision,
			CrossViewDerivationPolicyRevision,
		)
	}
	if want := crossViewDerivationPolicyFingerprint(); projection.DerivationPolicyFingerprint != want {
		return fmt.Errorf(
			"cross-view derivation policy fingerprint mismatch: recorded=%s current=%s",
			projection.DerivationPolicyFingerprint,
			want,
		)
	}
	copyProjection := projection
	copyProjection.ProjectionFingerprint = ""
	if want := crossViewFingerprint(copyProjection); projection.ProjectionFingerprint != want {
		return fmt.Errorf(
			"cross-view projection fingerprint mismatch: recorded=%s calculated=%s",
			projection.ProjectionFingerprint,
			want,
		)
	}
	validSourceKinds := map[string]struct{}{
		"task-action": {},
		"action-function-use": {},
		"task-function-io-target": {},
		"function-table-operation": {},
	}
	validTargetKinds := map[string]struct{}{
		"IOPanel": {},
		"IOSpool": {},
		"IOTable": {},
	}
	validUnresolvedCodes := map[string]struct{}{
		"task-action-without-task": {},
		"task-function-io-without-task-action": {},
		"task-function-io-function-id-missing": {},
		"task-function-io-function-not-found": {},
		"task-function-io-function-ambiguous": {},
		"task-function-io-target-id-missing": {},
		"task-function-io-target-not-found": {},
		"task-function-io-target-ambiguous": {},
		"iotable-without-function": {},
		"iotable-table-id-missing": {},
		"iotable-table-not-found": {},
		"iotable-table-ambiguous": {},
		"iotable-operation-marker-unsupported": {},
		"iotable-operation-missing": {},
	}
	validUnresolvedRelations := map[string]struct{}{
		"task-action": {},
		"action-function-use": {},
		"task-function-io-target": {},
		"function-table-operation": {},
	}
	for _, edge := range projection.SourceEdges {
		if edge.Nature != CrossViewNatureSource {
			return fmt.Errorf("cross-view source edge %q has invalid nature %q", edge.Kind, edge.Nature)
		}
		if _, ok := validSourceKinds[edge.Kind]; !ok {
			return fmt.Errorf("cross-view source edge kind %q is not public vocabulary", edge.Kind)
		}
		if err := validateCrossViewEdgeTokens(edge); err != nil {
			return err
		}
		if edge.DerivationRuleFingerprint != "" {
			return fmt.Errorf("source-backed edge %q must not carry derivation rule fingerprint", edge.Kind)
		}
		switch edge.Kind {
		case "task-function-io-target":
			if _, ok := validTargetKinds[edge.TargetKind]; !ok {
				return fmt.Errorf("cross-view target kind %q is not public vocabulary", edge.TargetKind)
			}
			if edge.Operations != "" {
				return fmt.Errorf("target interaction edge must not carry CRUD operations")
			}
		case "function-table-operation":
			if edge.TargetKind != "" {
				return fmt.Errorf("function-table-operation must not carry target kind")
			}
			if !validCrossViewOperations(edge.Operations) {
				return fmt.Errorf("function-table-operation CRUD %q is invalid", edge.Operations)
			}
		default:
			if edge.TargetKind != "" || edge.Operations != "" {
				return fmt.Errorf("cross-view source edge %q carries unexpected target/operation data", edge.Kind)
			}
		}
	}
	for _, edge := range projection.DerivedEdges {
		if edge.Nature != CrossViewNatureDerived ||
			edge.Kind != "action-function-table-potential-operation" {
			return fmt.Errorf("cross-view derived edge kind/nature is not public vocabulary: %q/%q", edge.Kind, edge.Nature)
		}
		if err := validateCrossViewEdgeTokens(edge); err != nil {
			return err
		}
		if edge.DerivationRuleFingerprint != projection.DerivationPolicyFingerprint {
			return fmt.Errorf("cross-view derived edge is not bound to current derivation policy")
		}
		if edge.TargetKind != "" || !validCrossViewOperations(edge.Operations) {
			return fmt.Errorf("cross-view derived edge target/CRUD is invalid")
		}
	}
	for _, action := range projection.Actions {
		if !crossViewFingerprintPattern.MatchString(action.ActionToken) {
			return fmt.Errorf("cross-view action token %q is not exact sha256", action.ActionToken)
		}
		if action.TaskToken != "" && !crossViewFingerprintPattern.MatchString(action.TaskToken) {
			return fmt.Errorf("cross-view task token %q is not exact sha256", action.TaskToken)
		}
	}
	for _, unresolved := range projection.Unresolved {
		if _, ok := validUnresolvedCodes[unresolved.Code]; !ok {
			return fmt.Errorf("cross-view unresolved code %q is not public vocabulary", unresolved.Code)
		}
		if _, ok := validUnresolvedRelations[unresolved.RelationKind]; !ok {
			return fmt.Errorf("cross-view unresolved relation kind %q is not public vocabulary", unresolved.RelationKind)
		}
		if len(unresolved.SourceTokens) == 0 {
			return fmt.Errorf("cross-view unresolved %q has no source token", unresolved.Code)
		}
		for _, token := range unresolved.SourceTokens {
			if !crossViewFingerprintPattern.MatchString(token) {
				return fmt.Errorf("cross-view unresolved token %q is not exact sha256", token)
			}
		}
	}
	return nil
}

func validateCrossViewEdgeTokens(edge CrossViewEdge) error {
	if !crossViewFingerprintPattern.MatchString(edge.FromToken) ||
		!crossViewFingerprintPattern.MatchString(edge.ToToken) {
		return fmt.Errorf("cross-view edge %q has invalid endpoint token", edge.Kind)
	}
	if len(edge.SourceTokens) == 0 {
		return fmt.Errorf("cross-view edge %q has no source tokens", edge.Kind)
	}
	for _, token := range edge.SourceTokens {
		if !crossViewFingerprintPattern.MatchString(token) {
			return fmt.Errorf("cross-view edge %q has invalid source token %q", edge.Kind, token)
		}
	}
	return nil
}

func validCrossViewOperations(value string) bool {
	if value == "" {
		return false
	}
	last := -1
	seen := map[rune]struct{}{}
	order := map[rune]int{'C': 0, 'R': 1, 'U': 2, 'D': 3}
	for _, marker := range value {
		position, ok := order[marker]
		if !ok || position <= last {
			return false
		}
		if _, duplicate := seen[marker]; duplicate {
			return false
		}
		seen[marker] = struct{}{}
		last = position
	}
	return true
}

func normalizeCrossViewContractFingerprint(value string) (string, error) {
	value = strings.TrimSpace(value)
	if crossViewFingerprintPattern.MatchString(value) {
		return value, nil
	}
	if regexp.MustCompile(`^[0-9a-f]{64}$`).MatchString(value) {
		return "sha256:" + value, nil
	}
	return "", fmt.Errorf("inspection transcription contract fingerprint %q is not exact sha256", value)
}

func flattenCrossViewNodes(doc Document) []crossViewNode {
	out := make([]crossViewNode, 0)
	ordinal := 0
	var walk func(Node, string, string, string)
	walk = func(node Node, taskToken, actionToken, functionToken string) {
		ordinal++
		token := crossViewSourceToken(doc.SHA256, ordinal)
		if node.Known {
			switch node.Name {
			case "Task":
				taskToken = token
			case "TaskAction":
				actionToken = token
			case "Function":
				functionToken = token
			}
		}
		out = append(out, crossViewNode{
			node:          node,
			token:         token,
			taskToken:     taskToken,
			actionToken:   actionToken,
			functionToken: functionToken,
		})
		for _, child := range node.Children {
			walk(child, taskToken, actionToken, functionToken)
		}
	}
	walk(doc.Root, "", "", "")
	return out
}

func actionFunctionPairs(edges []CrossViewEdge) []actionFunctionPair {
	byKey := map[string]*actionFunctionPair{}
	for _, edge := range edges {
		if edge.Kind != "action-function-use" {
			continue
		}
		key := edge.FromToken + "\x00" + edge.ToToken
		pair := byKey[key]
		if pair == nil {
			pair = &actionFunctionPair{actionToken: edge.FromToken, functionToken: edge.ToToken}
			byKey[key] = pair
		}
		pair.sourceTokens = normalizedTokens(append(pair.sourceTokens, edge.SourceTokens...)...)
	}
	out := make([]actionFunctionPair, 0, len(byKey))
	for _, pair := range byKey {
		out = append(out, *pair)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].actionToken != out[j].actionToken {
			return out[i].actionToken < out[j].actionToken
		}
		return out[i].functionToken < out[j].functionToken
	})
	return out
}

func iotableOperations(node Node) (string, []string) {
	var operations strings.Builder
	var invalid []string
	for _, item := range []struct {
		name string
		mark string
	}{
		{"OpC", "C"},
		{"OpR", "R"},
		{"OpU", "U"},
		{"OpD", "D"},
	} {
		value, ok := exactAttribute(node, item.name)
		if !ok || value == "" {
			continue
		}
		if value == "+" {
			operations.WriteString(item.mark)
			continue
		}
		invalid = append(invalid, item.name)
	}
	return operations.String(), invalid
}

func exactAttribute(node Node, name string) (string, bool) {
	for _, attr := range node.Attributes {
		if attr.Name == name {
			return attr.Value, true
		}
	}
	return "", false
}

func nonEmptyExactAttribute(node Node, name string) (string, bool) {
	value, ok := exactAttribute(node, name)
	value = strings.TrimSpace(value)
	return value, ok && value != ""
}

func unresolvedCrossView(code, relationKind string, tokens []string) CrossViewUnresolved {
	return CrossViewUnresolved{
		Code:         code,
		RelationKind: relationKind,
		SourceTokens: normalizedTokens(tokens...),
	}
}

func normalizedTokens(tokens ...string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(tokens))
	for _, token := range tokens {
		token = strings.TrimSpace(token)
		if token == "" {
			continue
		}
		if _, duplicate := seen[token]; duplicate {
			continue
		}
		seen[token] = struct{}{}
		out = append(out, token)
	}
	sort.Strings(out)
	return out
}

func sortCrossViewProjection(projection *CrossViewProjection) {
	sort.Slice(projection.SourceEdges, func(i, j int) bool {
		return crossViewEdgeKey(projection.SourceEdges[i]) < crossViewEdgeKey(projection.SourceEdges[j])
	})
	sort.Slice(projection.DerivedEdges, func(i, j int) bool {
		return crossViewEdgeKey(projection.DerivedEdges[i]) < crossViewEdgeKey(projection.DerivedEdges[j])
	})
	sort.Slice(projection.Actions, func(i, j int) bool {
		return projection.Actions[i].ActionToken < projection.Actions[j].ActionToken
	})
	sort.Slice(projection.Unresolved, func(i, j int) bool {
		if projection.Unresolved[i].Code != projection.Unresolved[j].Code {
			return projection.Unresolved[i].Code < projection.Unresolved[j].Code
		}
		return strings.Join(projection.Unresolved[i].SourceTokens, "\x00") <
			strings.Join(projection.Unresolved[j].SourceTokens, "\x00")
	})
}

func crossViewEdgeKey(edge CrossViewEdge) string {
	return strings.Join([]string{
		edge.Kind,
		edge.Nature,
		edge.FromToken,
		edge.ToToken,
		edge.TargetKind,
		edge.Operations,
		strings.Join(edge.SourceTokens, ","),
	}, "\x00")
}

func crossViewDerivationPolicyFingerprint() string {
	return crossViewFingerprint(struct {
		Revision string   `json:"revision"`
		Rules    []string `json:"rules"`
	}{
		Revision: CrossViewDerivationPolicyRevision,
		Rules: []string{
			"source bytes and exact transcription contract pass InspectWithContract source-profile validation before cross-view derivation",
			"source tokens are bound to the exact artifact digest and preorder traversal ordinal",
			"only known unqualified source constructs participate in standard cross-view relations",
			"only exact unqualified source attributes participate in legacy reference resolution",
			"FunctionID TableID and IOID references resolve only when exactly one matching source node exists",
			"IOID values are trimmed and zero means no target interaction edge while remaining a source-level count",
			"IOTable requires at least one recognized C R U or D marker and unsupported or empty operation declarations remain unresolved",
			"Task to TaskAction action to Function TaskFunctionIO to target and Function to Table CRUD remain source-backed edge kinds",
			"group TaskFunctionIO by exact action source token and uniquely resolved Function source token",
			"join each distinct action/function pair to every source-backed Function/IOTable/Table operation edge",
			"carry Function-level declared CRUD unchanged and never relabel it as action-specific effect",
			"bind every derived path to the union of supporting source tokens",
		},
	})
}

func crossViewSourceToken(sourceSHA256 string, ordinal int) string {
	return crossViewDigest(fmt.Sprintf("%s|node-ordinal:%d", sourceSHA256, ordinal))
}

func crossViewFingerprint(value any) string {
	encoded, _ := json.Marshal(value)
	return crossViewDigest(string(encoded))
}

func crossViewDigest(value string) string {
	sum := sha256.Sum256([]byte(value))
	return "sha256:" + hex.EncodeToString(sum[:])
}
