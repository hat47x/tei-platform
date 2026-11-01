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

	data := semanticDataIndex(world)
	applications := append([]application.Application(nil), world.Applications...)
	sort.Slice(applications, func(i, j int) bool { return applications[i].ID < applications[j].ID })

	var gaps []SurfaceGap
	for _, app := range applications {
		if applicationFilter != "" && string(app.ID) != applicationFilter {
			continue
		}
		ops, appGaps := deriveApplicationOperations(world, app, data)
		out.Operations = append(out.Operations, ops...)
		gaps = append(gaps, appGaps...)
	}

	sortOperations(out.Operations)
	sortGaps(gaps)
	out.Gaps = gaps
	out.Summary = summarize(out)
	return out
}

func deriveApplicationOperations(
	world ir.World,
	app application.Application,
	data map[core.CanonicalID]semantic.Data,
) ([]Operation, []SurfaceGap) {
	var ops []Operation
	var gaps []SurfaceGap
	seen := map[string]struct{}{}

	aliases := append([]application.DatabaseRequirement(nil), app.Databases...)
	sort.Slice(aliases, func(i, j int) bool { return aliases[i].Alias < aliases[j].Alias })

	for _, alias := range aliases {
		for _, mapping := range persistenceForDatabase(world.Persistence, alias.TargetDatabase) {
			element, ok := data[mapping.SemanticData]
			if !ok {
				// The missing semantic data is reported by the existing
				// workspace validation; the surface does not duplicate it.
				continue
			}
			if alias.Access == database.AccessWrite {
				gaps = append(gaps, SurfaceGap{
					Target: string(mapping.SemanticData),
					Kind:   "list",
					Reason: "conflict",
					Detail: "datasource alias " + string(alias.Alias) + " is write-only",
				})
				continue
			}

			listID := operationID(app.ID, "list", mapping.SemanticData)
			if _, exists := seen[listID]; !exists {
				seen[listID] = struct{}{}
				ops = append(ops, Operation{
					ID:       listID,
					Kind:     "list",
					Target:   string(mapping.SemanticData),
					Origin:   "derived",
					Autonomy: "unattended",
					Basis:    mappingBasis(mapping),
				})
			}

			key := surfaceKey(element, mapping)
			readID := operationID(app.ID, "read", mapping.SemanticData)
			if len(key) == 0 {
				gaps = append(gaps, SurfaceGap{
					Target: string(mapping.SemanticData),
					Kind:   "read",
					Reason: "missing-semantics",
					Detail: "no candidate key or persistence primary key is declared",
				})
				continue
			}
			if _, exists := seen[readID]; !exists {
				seen[readID] = struct{}{}
				ops = append(ops, Operation{
					ID:       readID,
					Kind:     "read",
					Target:   string(mapping.SemanticData),
					Origin:   "derived",
					Inputs:   key,
					Autonomy: "unattended",
					Basis:    mappingBasis(mapping),
				})
			}
		}
	}
	return ops, gaps
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
