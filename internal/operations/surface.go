package operations

import (
	"sort"
	"strings"

	"tei/internal/application"
	"tei/internal/core"
	"tei/internal/database"
	"tei/internal/database/persistence"
	"tei/internal/ir"
	"tei/internal/semantic"
	"tei/internal/sourceindex"
	"tei/internal/workspace"
)

const (
	// OperationSurfaceSchemaVersion identifies the read-only operation surface.
	OperationSurfaceSchemaVersion = "tei.operations.operation-surface/v0"

	operationSurfaceRuleID      = "tei.operation-surface/standard-crud"
	operationSurfaceRuleVersion = "v1"
)

// operationSurfaceRulePolicy is the derivation policy text covered by the rule
// fingerprint. Changing a derivation rule without changing this text, or the
// version, must fail review: the fingerprint is what lets a consumer tell that
// two surfaces were produced by the same rules.
const operationSurfaceRulePolicy = `standard-crud/v1
list: requires a readable datasource alias and a persistence mapping
read: requires a candidate key on the semantic data, else a persistence primary key
autonomy: list and read are unattended because they have no effect`

// OperationSurface is a read-only, target-neutral derivation over the accepted
// meaning. It lists what an application can do, what is missing and which
// operations may run unattended. It is a completeness and navigation aid: it is
// not an execution authority, it does not create canonical elements and it does
// not grant runtime authority.
type OperationSurface struct {
	SchemaVersion    string            `json:"schemaVersion"`
	Effect           string            `json:"effect"`
	Application      string            `json:"application,omitempty"`
	WorldFingerprint string            `json:"worldFingerprint"`
	ModelRevision    string            `json:"modelRevision"`
	Rule             SurfaceRule       `json:"rule"`
	Operations       []Operation       `json:"operations"`
	Gaps             []SurfaceGap      `json:"gaps"`
	Conflicts        []SurfaceConflict `json:"conflicts"`
	Summary          SurfaceSummary    `json:"summary"`
	AuthorityLimits  []string          `json:"authorityLimits"`
}

// SurfaceRule identifies the derivation rules that produced a surface.
type SurfaceRule struct {
	ID          string `json:"id"`
	Version     string `json:"version"`
	Fingerprint string `json:"fingerprint"`
}

// Operation is one derived or declared business operation.
type Operation struct {
	ID            string   `json:"id"`
	Kind          string   `json:"kind"`
	Target        string   `json:"target"`
	Origin        string   `json:"origin"`
	OverriddenBy  string   `json:"overriddenBy,omitempty"`
	Inputs        []string `json:"inputs,omitempty"`
	Preconditions []string `json:"preconditions,omitempty"`
	Effects       []string `json:"effects,omitempty"`
	// Concurrency names how an update detects a concurrent change. The PoC
	// hypothesis is expected-state comparison; a version attribute would be a
	// second, declared method (see plan/workstreams/POC_HYPOTHESES_2026-11.md).
	Concurrency string `json:"concurrency,omitempty"`
	// Binding names the capability that executes a write operation. The PoC
	// does not add a generic write executor: a write runs through an existing
	// capability that declares the same effect on the same data.
	Binding string `json:"binding,omitempty"`
	// Period names the period that governs the data of this operation. When
	// the period is closed, PeriodRejects lists the operations it rejects.
	Period string `json:"period,omitempty"`
	// EffectRules lists the effect rules that run around the bound capability.
	EffectRules   []string `json:"effectRules,omitempty"`
	PeriodRejects []string `json:"periodRejects,omitempty"`
	Autonomy      string   `json:"autonomy"`
	Basis         string   `json:"basis"`
}

// SurfaceGap records why a standard operation could not be derived.
type SurfaceGap struct {
	Target string `json:"target"`
	Kind   string `json:"kind"`
	Reason string `json:"reason"`
	Detail string `json:"detail"`
}

// SurfaceConflict records a declaration that contradicts the derivation.
type SurfaceConflict struct {
	Target string `json:"target"`
	Kind   string `json:"kind"`
	Detail string `json:"detail"`
}

// SurfaceSummary counts the surface for readers and tests.
type SurfaceSummary struct {
	Operations int `json:"operations"`
	Derived    int `json:"derived"`
	Declared   int `json:"declared"`
	Unattended int `json:"unattended"`
	Gaps       int `json:"gaps"`
}

// SurfaceWorkspace reads a workspace root and derives its operation surface.
// It uses the existing loader, Source Index and fingerprint helpers, so the
// surface is bound to the same normalized World as the other read-only
// commands. It performs no write and records no acceptance.
func SurfaceWorkspace(root, applicationFilter string) (OperationSurface, error) {
	loaded, err := workspace.Load(root)
	if err != nil {
		return OperationSurface{}, &ValidationFailure{Stage: StageLoad, Err: err}
	}
	idx, err := workspace.BuildSourceIndex(root, loaded)
	if err != nil {
		return OperationSurface{}, &ValidationFailure{Stage: StageSourceIndex, Err: err}
	}
	worldFP, _, err := workspaceProjectionFingerprints(loaded.World, idx)
	if err != nil {
		return OperationSurface{}, err
	}
	out := DeriveOperationSurface(loaded.World, applicationFilter)
	out.WorldFingerprint = worldFP
	out.ModelRevision = sourceindex.ModelRevision(idx)
	return out, nil
}

// DeriveOperationSurface derives the standard operations of every application
// from the accepted meaning. The result is deterministic: inputs are sorted and
// no map iteration order or file enumeration order reaches the output.
func DeriveOperationSurface(world ir.World, applicationFilter string) OperationSurface {
	out := OperationSurface{
		SchemaVersion: OperationSurfaceSchemaVersion,
		Effect:        "read-only",
		Application:   applicationFilter,
		Rule: SurfaceRule{
			ID:          operationSurfaceRuleID,
			Version:     operationSurfaceRuleVersion,
			Fingerprint: sourceindex.Fingerprint([]byte(operationSurfaceRulePolicy)),
		},
		Operations: []Operation{},
		Gaps:       []SurfaceGap{},
		Conflicts:  []SurfaceConflict{},
		AuthorityLimits: []string{
			"operation-surface-is-a-derived-read-only-projection",
			"operation-surface-is-not-execution-authority",
			"derived-operations-do-not-create-canonical-elements",
			"autonomy-class-does-not-grant-runtime-authority",
		},
	}

	index := newSurfaceIndex(world)
	applications := append([]application.Application(nil), world.Applications...)
	sort.Slice(applications, func(i, j int) bool { return applications[i].ID < applications[j].ID })

	var gaps []SurfaceGap
	for _, app := range applications {
		if applicationFilter != "" && string(app.ID) != applicationFilter {
			continue
		}
		ops, appGaps := deriveApplicationOperations(world, app, index)
		out.Operations = append(out.Operations, ops...)
		gaps = append(gaps, appGaps...)
	}

	sortOperations(out.Operations)
	out.Gaps = dedupeGaps(gaps)
	sortGaps(out.Gaps)
	out.Summary = summarize(out)
	return out
}

// surfaceIndex holds the semantic lookups the derivation needs. It is built
// once per surface so the derivation stays a pure function of the World.
type surfaceIndex struct {
	data         map[core.CanonicalID]semantic.Data
	validations  map[core.CanonicalID]semantic.Validation
	updateIntent map[core.CanonicalID]bool
	// binding maps "<effect kind>\x00<data>" to the capability that declares it.
	binding map[string]string
	// periods maps a data to the period that governs it.
	periods map[core.CanonicalID]semantic.Period
	// effectRules maps a capability to the effect rules that run around it.
	effectRules map[core.CanonicalID][]string
}

func newSurfaceIndex(world ir.World) surfaceIndex {
	index := surfaceIndex{
		data:         semanticDataIndex(world),
		validations:  map[core.CanonicalID]semantic.Validation{},
		updateIntent: map[core.CanonicalID]bool{},
		binding:      map[string]string{},
		periods:      map[core.CanonicalID]semantic.Period{},
		effectRules:  map[core.CanonicalID][]string{},
	}
	for _, namespace := range world.Semantics {
		for _, validation := range namespace.Validations {
			if _, exists := index.validations[validation.ID]; !exists {
				index.validations[validation.ID] = validation
			}
		}
		for _, period := range namespace.Periods {
			if _, exists := index.periods[period.Data]; !exists {
				index.periods[period.Data] = period
			}
		}
		for _, rule := range namespace.EffectRules {
			index.effectRules[rule.OnCapability] = append(index.effectRules[rule.OnCapability], string(rule.ID))
		}
		for _, capability := range namespace.Capabilities {
			if capability.Effect == nil {
				continue
			}
			if !capabilityAcceptsData(capability, capability.Effect.Data) {
				continue
			}
			if capability.Effect.Kind == semantic.DataEffectUpdate {
				index.updateIntent[capability.Effect.Data] = true
			}
			key := string(capability.Effect.Kind) + "\x00" + string(capability.Effect.Data)
			if existing, ok := index.binding[key]; !ok || string(capability.ID) < existing {
				index.binding[key] = string(capability.ID)
			}
		}
	}
	return index
}

// capabilityAcceptsData reports whether the capability takes the data it writes
// as an input or returns it. A capability that does not accept the data can not
// be the execution binding of a standard write operation.
func capabilityAcceptsData(capability semantic.Capability, data core.CanonicalID) bool {
	if capability.Output == data {
		return true
	}
	for _, input := range capability.Inputs {
		if input.Type == data {
			return true
		}
	}
	return false
}

// bindingFor returns the capability that executes an operation of this kind on
// this data, when the workspace declares one.
func (i surfaceIndex) bindingFor(kind string, data core.CanonicalID) string {
	return i.binding[kind+"\x00"+string(data)]
}

func deriveApplicationOperations(
	world ir.World,
	app application.Application,
	index surfaceIndex,
) ([]Operation, []SurfaceGap) {
	var ops []Operation
	var gaps []SurfaceGap
	seen := map[string]struct{}{}

	aliases := append([]application.DatabaseRequirement(nil), app.Databases...)
	sort.Slice(aliases, func(i, j int) bool { return aliases[i].Alias < aliases[j].Alias })

	for _, alias := range aliases {
		for _, mapping := range persistenceForDatabase(world.Persistence, alias.TargetDatabase) {
			periodID, rejects := periodFields(index, mapping.SemanticData)
			element, ok := index.data[mapping.SemanticData]
			if !ok {
				// The missing semantic data is reported by the existing
				// workspace validation; the surface does not duplicate it.
				continue
			}
			target := string(mapping.SemanticData)
			basis := mappingBasis(mapping)
			key := surfaceKey(element, mapping)
			readable := alias.Access != database.AccessWrite
			writable := alias.Access != database.AccessRead

			if !readable {
				gaps = append(gaps, SurfaceGap{
					Target: target,
					Kind:   "list",
					Reason: "conflict",
					Detail: "datasource alias " + string(alias.Alias) + " is write-only",
				})
			} else {
				emitOperation(&ops, seen, Operation{
					ID:       operationID(app.ID, "list", mapping.SemanticData),
					Kind:     "list",
					Target:   target,
					Origin:   "derived",
					Autonomy: "unattended",
					Basis:    basis,
				})
				if len(key) == 0 {
					gaps = append(gaps, SurfaceGap{
						Target: target,
						Kind:   "read",
						Reason: "missing-semantics",
						Detail: "no candidate key or persistence primary key is declared",
					})
				} else {
					emitOperation(&ops, seen, Operation{
						ID:       operationID(app.ID, "read", mapping.SemanticData),
						Kind:     "read",
						Target:   target,
						Origin:   "derived",
						Inputs:   key,
						Autonomy: "unattended",
						Basis:    basis,
					})
				}
			}

			if !writable {
				continue
			}
			if len(key) == 0 {
				gaps = append(gaps, SurfaceGap{
					Target: target,
					Kind:   "create",
					Reason: "missing-semantics",
					Detail: "create requires an identifying key",
				})
				continue
			}

			emitOperation(&ops, seen, Operation{
				ID:            operationID(app.ID, "create", mapping.SemanticData),
				Kind:          "create",
				Target:        target,
				Origin:        "derived",
				Inputs:        createInputs(mapping),
				Preconditions: preconditionsFor(app, element, index),
				Effects:       []string{"create:" + target},
				Binding:       index.bindingFor("create", mapping.SemanticData),
				Period:        periodID,
				PeriodRejects: rejects,
				Autonomy:      "attended",
				Basis:         basis,
			})

			// PoC hypothesis H1: an update is derived with expected-state
			// comparison. The caller sends the key and the values it read; the
			// write applies only when they still match. A declared version
			// attribute is a later, explicit alternative, not a requirement.
			emitOperation(&ops, seen, Operation{
				ID:            operationID(app.ID, "update", mapping.SemanticData),
				Kind:          "update",
				Target:        target,
				Origin:        "derived",
				Inputs:        key,
				Effects:       []string{"update:" + target},
				Concurrency:   "expected-state",
				Binding:       index.bindingFor("update", mapping.SemanticData),
				Period:        periodID,
				PeriodRejects: rejects,
				Autonomy:      "attended",
				Basis:         basis,
			})
			if index.updateIntent[mapping.SemanticData] {
				gaps = append(gaps, SurfaceGap{
					Target: target,
					Kind:   "retire",
					Reason: "missing-input",
					Detail: "no lifecycle or state declaration exists for this data",
				})
			}
		}
	}
	for i := range ops {
		if rules := index.effectRules[core.CanonicalID(ops[i].Binding)]; len(rules) > 0 {
			ops[i].EffectRules = append([]string(nil), rules...)
		}
		if ops[i].Period == "" {
			periodID, rejects := periodFields(index, core.CanonicalID(ops[i].Target))
			if periodID == "" {
				continue
			}
			ops[i].Period = periodID
			ops[i].PeriodRejects = rejects
		}
		// A write the closed period rejects needs proof that the period is
		// open before it runs.
		if periodRejectsOperation(ops[i].PeriodRejects, ops[i].Kind) && !hasOperationPrecondition(ops[i], "period:"+ops[i].Period+":open") {
			ops[i].Preconditions = append(ops[i].Preconditions, "period:"+ops[i].Period+":open")
		}
	}
	return ops, gaps
}

// emitOperation appends an operation once. Two aliases may reach the same
// persistence mapping; the surface keeps the first deterministic result.
func emitOperation(ops *[]Operation, seen map[string]struct{}, operation Operation) {
	if _, exists := seen[operation.ID]; exists {
		return
	}
	seen[operation.ID] = struct{}{}
	*ops = append(*ops, operation)
}

// createInputs lists the persisted attributes a caller supplies. Column order
// is the authored order, so the result does not depend on map iteration.
func createInputs(mapping persistence.Mapping) []string {
	seen := map[core.CanonicalID]struct{}{}
	var out []string
	for _, column := range mapping.Columns {
		if _, exists := seen[column.Attribute]; exists {
			continue
		}
		seen[column.Attribute] = struct{}{}
		out = append(out, string(column.Attribute))
	}
	return out
}

// preconditionsFor returns the validations the application declares and that
// belong to the data element or one of its attributes.
func preconditionsFor(app application.Application, element semantic.Data, index surfaceIndex) []string {
	if len(app.Validations) == 0 {
		return nil
	}
	attributes := map[core.CanonicalID]struct{}{}
	for _, attribute := range element.Attributes {
		attributes[attribute.ID] = struct{}{}
	}
	var out []string
	for _, id := range app.Validations {
		validation, ok := index.validations[id]
		if !ok {
			continue
		}
		if _, ok := attributes[validation.Attribute]; !ok {
			continue
		}
		out = append(out, string(validation.ID))
	}
	sort.Strings(out)
	return out
}

// semanticDataIndex returns every semantic data element by canonical ID.
func semanticDataIndex(world ir.World) map[core.CanonicalID]semantic.Data {
	out := map[core.CanonicalID]semantic.Data{}
	for _, namespace := range world.Semantics {
		for _, element := range namespace.Data {
			if _, exists := out[element.ID]; exists {
				continue
			}
			out[element.ID] = element
		}
	}
	return out
}

// persistenceForDatabase returns the mappings owned by one database, sorted by
// semantic data so the derivation does not depend on declaration order.
func persistenceForDatabase(mappings []persistence.Mapping, databaseID core.CanonicalID) []persistence.Mapping {
	var out []persistence.Mapping
	for _, mapping := range mappings {
		if mapping.Database == databaseID {
			out = append(out, mapping)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].SemanticData < out[j].SemanticData })
	return out
}

// surfaceKey returns the attributes that identify one record. A candidate key,
// which is an authored business identity, is preferred over the persistence
// primary key, which is a physical decision. The caller records which one was
// used through the operation basis.
func surfaceKey(element semantic.Data, mapping persistence.Mapping) []string {
	keys := append([]semantic.CandidateKey(nil), element.CandidateKeys...)
	sort.Slice(keys, func(i, j int) bool { return keys[i].ID < keys[j].ID })
	if len(keys) > 0 && len(keys[0].Attributes) > 0 {
		out := make([]string, 0, len(keys[0].Attributes))
		for _, attribute := range keys[0].Attributes {
			out = append(out, string(attribute))
		}
		return out
	}
	if len(mapping.PrimaryKey) > 0 {
		out := make([]string, 0, len(mapping.PrimaryKey))
		for _, attribute := range mapping.PrimaryKey {
			out = append(out, string(attribute))
		}
		return out
	}
	return nil
}

func operationID(applicationID core.CanonicalID, kind string, target core.CanonicalID) string {
	return string(applicationID) + "." + kind + "." + string(target)
}

// mappingBasis fingerprints the persistence decision an operation was derived
// from. It is not a Shared Acceptance revision.
func mappingBasis(mapping persistence.Mapping) string {
	parts := []string{string(mapping.ID), mapping.Table}
	for _, attribute := range mapping.PrimaryKey {
		parts = append(parts, string(attribute))
	}
	return sourceindex.Fingerprint([]byte(strings.Join(parts, "\n")))
}

func sortOperations(operations []Operation) {
	sort.Slice(operations, func(i, j int) bool {
		if operations[i].Target != operations[j].Target {
			return operations[i].Target < operations[j].Target
		}
		if operationKindRank(operations[i].Kind) != operationKindRank(operations[j].Kind) {
			return operationKindRank(operations[i].Kind) < operationKindRank(operations[j].Kind)
		}
		return operations[i].ID < operations[j].ID
	})
}

func operationKindRank(kind string) int {
	switch kind {
	case "list":
		return 0
	case "read":
		return 1
	case "create":
		return 2
	case "update":
		return 3
	case "retire":
		return 4
	default:
		return 5
	}
}

func sortGaps(gaps []SurfaceGap) {
	sort.Slice(gaps, func(i, j int) bool {
		if gaps[i].Target != gaps[j].Target {
			return gaps[i].Target < gaps[j].Target
		}
		if gaps[i].Kind != gaps[j].Kind {
			return gaps[i].Kind < gaps[j].Kind
		}
		return gaps[i].Reason < gaps[j].Reason
	})
}

// dedupeGaps keeps one gap per target, kind, reason and detail. Two aliases may
// reach the same mapping, and the same missing declaration must not be reported
// twice.
func dedupeGaps(gaps []SurfaceGap) []SurfaceGap {
	seen := map[string]struct{}{}
	out := make([]SurfaceGap, 0, len(gaps))
	for _, gap := range gaps {
		key := gap.Target + "\x00" + gap.Kind + "\x00" + gap.Reason + "\x00" + gap.Detail
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, gap)
	}
	return out
}

func summarize(surface OperationSurface) SurfaceSummary {
	summary := SurfaceSummary{Operations: len(surface.Operations), Gaps: len(surface.Gaps)}
	for _, operation := range surface.Operations {
		switch operation.Origin {
		case "derived":
			summary.Derived++
		case "declared":
			summary.Declared++
		}
		if operation.Autonomy == "unattended" {
			summary.Unattended++
		}
	}
	return summary
}

// periodFields returns the period that governs a data and the operations it
// rejects once closed. A data without a period yields empty values.
func periodFields(index surfaceIndex, data core.CanonicalID) (string, []string) {
	period, ok := index.periods[data]
	if !ok {
		return "", nil
	}
	return string(period.ID), append([]string(nil), period.ClosedRejects...)
}

// periodRejectsOperation reports whether a closed period rejects this operation.
func periodRejectsOperation(rejects []string, kind string) bool {
	for _, rejected := range rejects {
		if rejected == kind {
			return true
		}
	}
	return false
}

// hasOperationPrecondition reports whether the operation already declares it.
func hasOperationPrecondition(operation Operation, want string) bool {
	for _, precondition := range operation.Preconditions {
		if precondition == want {
			return true
		}
	}
	return false
}
