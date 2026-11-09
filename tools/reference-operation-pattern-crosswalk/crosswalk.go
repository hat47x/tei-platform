// Package operationpatterncrosswalk is a read-only reference for the W3
// crosswalk described in plan/design/OPERATION_SURFACE_DESIGN_2026-11.md.
//
// It reads an accepted World and reports which declared Pattern can be linked
// to which derived standard operation, using only three declared subjects:
// a Data id, a Capability id, and a Relation's target Data. It never writes an
// identifier, an operation, or an acceptance back into the World, and a link is
// a reading, not an execution authority.
package operationpatterncrosswalk

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"sort"

	"tei/internal/application"
	"tei/internal/core"
	"tei/internal/ir"
	"tei/internal/operations"
	"tei/internal/semantic"
)

const (
	// Schema identifies the report. It is a reference output, not a product
	// contract.
	Schema = "tei.reference.operation-pattern-crosswalk/v0"
	// Authority states what the report is allowed to claim.
	Authority = "derived-read-only"

	RuleID      = "tei.reference.operation-pattern-crosswalk"
	RuleVersion = "v0"
)

// rulePolicy is covered by the fingerprint. Changing how a link is decided
// without changing this text or RuleVersion must fail review: the fingerprint
// is what lets a consumer tell that two reports used the same rules.
const rulePolicy = `operation-pattern-crosswalk/v0
browse        -> list   matched on Pattern.Data = Operation.Target
record        -> read   matched on Pattern.Data = Operation.Target
command       -> create or update, only when the Capability declares an effect
                 on the same Data; matched on Capability = Operation.Binding
reference     -> read   on the Relation target Data
header-detail -> list and read on the Relation target Data; a Kind that matches
                 more than one operation is reported as ambiguous, not chosen
unmatched declarations are reported with the operation-surface gap vocabulary
(missing-input / missing-semantics / conflict / out-of-scope) and are never
filled with a default
`

// Via names the one subject that decided a link. A link is never inferred from
// a similar name, a prefix, or a load order.
const (
	ViaData       = "data"
	ViaCapability = "capability"
	ViaRelation   = "relation"
)

// Operation kinds this rule reads. They are the vocabulary of the derivation
// rule, not a core enumeration.
const (
	OperationList   = "list"
	OperationRead   = "read"
	OperationCreate = "create"
	OperationUpdate = "update"
)

// Link is one declared Pattern read as one derived operation. It does not
// change the operation, the Pattern, or the authority of either.
type Link struct {
	Operation     string `json:"operation"`
	OperationKind string `json:"operationKind"`
	Pattern       string `json:"pattern"`
	PatternKind   string `json:"patternKind"`
	Via           string `json:"via"`
	// Subject is the Data id that was compared. For a Relation link it is the
	// Relation's target Data, after the declared read-through.
	Subject string `json:"subject"`
}

// Unlinked is a declared Pattern that this rule could not link. Code uses the
// operation-surface gap vocabulary so a reader does not have to learn a
// second set of names.
type Unlinked struct {
	Pattern string `json:"pattern"`
	Kind    string `json:"kind"`
	Code    string `json:"code"`
	Detail  string `json:"detail"`
}

// Conflict is two declarations competing for the same operation. The rule
// reports it and does not choose one.
type Conflict struct {
	Operation string   `json:"operation"`
	Patterns  []string `json:"patterns"`
	Detail    string   `json:"detail"`
}

// Application groups the report for one Application. Operations and Patterns
// are listed so a reader can see the whole input, not only the links.
type Application struct {
	ID         string     `json:"id"`
	Operations []string   `json:"operations"`
	Patterns   []string   `json:"patterns"`
	Links      []Link     `json:"links"`
	Unlinked   []Unlinked `json:"unlinked"`
	Conflicts  []Conflict `json:"conflicts"`
	// Ambiguous names Patterns whose Kind maps to more than one operation kind
	// (currently header-detail). All matching links are reported.
	Ambiguous []string `json:"ambiguous,omitempty"`
}

// Report is the whole crosswalk for one accepted World.
type Report struct {
	Schema          string        `json:"schema"`
	Authority       string        `json:"authority"`
	RuleID          string        `json:"ruleId"`
	RuleVersion     string        `json:"ruleVersion"`
	RuleFingerprint string        `json:"ruleFingerprint"`
	Applications    []Application `json:"applications"`
	Notes           []string      `json:"notes"`
}

type relationTargets map[core.CanonicalID]core.CanonicalID

func relationTargetIndex(world ir.World) relationTargets {
	out := relationTargets{}
	for _, namespace := range world.Semantics {
		// Relations belong to a Data in the accepted model; the Pattern names
		// the Relation, so the index is keyed by the Relation id, not by Data.
		for _, data := range namespace.Data {
			for _, relation := range data.Relations {
				if relation.Target == "" {
					continue
				}
				if _, exists := out[relation.ID]; !exists {
					out[relation.ID] = relation.Target
				}
			}
		}
	}
	return out
}

func capabilityIndex(world ir.World) map[core.CanonicalID]semantic.Capability {
	out := map[core.CanonicalID]semantic.Capability{}
	for _, namespace := range world.Semantics {
		for _, capability := range namespace.Capabilities {
			if _, exists := out[capability.ID]; !exists {
				out[capability.ID] = capability
			}
		}
	}
	return out
}

// Derive reads the accepted World and reports the crosswalk. The result is
// deterministic: inputs are sorted and no map iteration order reaches the
// output.
func Derive(world ir.World) Report {
	report := Report{
		Schema:          Schema,
		Authority:       Authority,
		RuleID:          RuleID,
		RuleVersion:     RuleVersion,
		RuleFingerprint: fmt.Sprintf("sha256:%x", sha256.Sum256([]byte(rulePolicy))),
		Notes: []string{
			"link is a reading of the accepted meaning; it is not execution authority",
			"a missing link is reported, never filled with a default",
			"header-detail can match more than one operation kind; both are listed and the Pattern is marked ambiguous",
			"the operation surface this reads is the same rule the CLI reports (tei.operation-surface/standard-crud)",
		},
	}
	relations := relationTargetIndex(world)
	capabilities := capabilityIndex(world)

	apps := append([]application.Application(nil), world.Applications...)
	sort.Slice(apps, func(i, j int) bool { return apps[i].ID < apps[j].ID })
	for _, app := range apps {
		report.Applications = append(report.Applications, deriveApplication(world, app, relations, capabilities))
	}
	return report
}

// candidate is one Pattern read as one operation kind on one Data.
type candidate struct {
	operationKind string
	subject       string
	via           string
	// code and detail explain a Pattern that cannot be read at all; when set,
	// the candidate is reported as unlinked instead of linked.
	code   string
	detail string
}

func deriveApplication(
	world ir.World,
	app application.Application,
	relations relationTargets,
	capabilities map[core.CanonicalID]semantic.Capability,
) Application {
	out := Application{
		ID: string(app.ID),
		// Empty slices are encoded as [] rather than null so a reader never has
		// to distinguish "no operations" from "the field was absent".
		Operations: []string{},
		Patterns:   []string{},
		Links:      []Link{},
		Unlinked:   []Unlinked{},
		Conflicts:  []Conflict{},
	}

	// The derived operations come from the existing operation surface, so this
	// reference does not invent a second derivation of the same meaning.
	surface := operations.DeriveOperationSurface(world, string(app.ID))
	operationsByKey := map[string]operations.Operation{}
	for _, operation := range surface.Operations {
		out.Operations = append(out.Operations, operation.ID)
		operationsByKey[operation.Kind+"\x00"+operation.Target] = operation
	}
	sort.Strings(out.Operations)

	type claim struct {
		patternID string
		kind      string
		link      Link
	}
	claims := map[string][]claim{}
	patterns := append([]application.Pattern(nil), app.Patterns...)
	sort.Slice(patterns, func(i, j int) bool { return patterns[i].ID < patterns[j].ID })
	for _, pattern := range patterns {
		out.Patterns = append(out.Patterns, string(pattern.ID))
		var candidates []candidate
		var unlinked *Unlinked
		switch pattern.Kind {
		case application.PatternBrowse:
			candidates, unlinked = onData(pattern, OperationList, ViaData)
		case application.PatternRecord:
			candidates, unlinked = onData(pattern, OperationRead, ViaData)
		case application.PatternCommand:
			candidates, unlinked = onCommand(pattern, capabilities)
		case application.PatternReference:
			candidates, unlinked = onRelation(pattern, relations, ViaRelation, OperationRead)
		case application.PatternHeaderDetail:
			candidates, unlinked = onRelation(pattern, relations, ViaRelation, OperationList, OperationRead)
		default:
			unlinked = &Unlinked{
				Pattern: string(pattern.ID), Kind: string(pattern.Kind),
				Code:   "missing-semantics",
				Detail: "this rule reads only browse, record, reference, command and header-detail",
			}
		}
		if unlinked != nil {
			out.Unlinked = append(out.Unlinked, *unlinked)
			continue
		}
		if len(candidates) > 1 {
			out.Ambiguous = append(out.Ambiguous, string(pattern.ID))
		}
		for _, candidate := range candidates {
			operation, ok := operationsByKey[candidate.operationKind+"\x00"+candidate.subject]
			if !ok {
				out.Unlinked = append(out.Unlinked, Unlinked{
					Pattern: string(pattern.ID), Kind: string(pattern.Kind),
					Code:   "missing-input",
					Detail: "no derived " + candidate.operationKind + " operation on " + candidate.subject,
				})
				continue
			}
			claims[operation.ID] = append(claims[operation.ID], claim{
				patternID: string(pattern.ID), kind: candidate.operationKind,
				link: Link{
					Operation: operation.ID, OperationKind: candidate.operationKind,
					Pattern: string(pattern.ID), PatternKind: string(pattern.Kind),
					Via: candidate.via, Subject: candidate.subject,
				},
			})
		}
	}

	operationIDs := make([]string, 0, len(claims))
	for id := range claims {
		operationIDs = append(operationIDs, id)
	}
	sort.Strings(operationIDs)
	for _, id := range operationIDs {
		group := claims[id]
		if len(group) > 1 {
			// Two declarations competing for one operation are a conflict.
			// The rule does not pick a winner.
			patterns := make([]string, 0, len(group))
			for _, c := range group {
				patterns = append(patterns, c.patternID)
			}
			sort.Strings(patterns)
			out.Conflicts = append(out.Conflicts, Conflict{
				Operation: id, Patterns: patterns,
				Detail: "more than one declared operation reads as the same derived operation",
			})
			continue
		}
		out.Links = append(out.Links, group[0].link)
	}
	sort.Slice(out.Links, func(i, j int) bool { return out.Links[i].Operation < out.Links[j].Operation })
	sort.Slice(out.Unlinked, func(i, j int) bool {
		if out.Unlinked[i].Code != out.Unlinked[j].Code {
			return out.Unlinked[i].Code < out.Unlinked[j].Code
		}
		return out.Unlinked[i].Pattern < out.Unlinked[j].Pattern
	})
	sort.Slice(out.Conflicts, func(i, j int) bool { return out.Conflicts[i].Operation < out.Conflicts[j].Operation })
	return out
}

func onData(pattern application.Pattern, operationKind, via string) ([]candidate, *Unlinked) {
	if pattern.Data == "" {
		return nil, &Unlinked{
			Pattern: string(pattern.ID), Kind: string(pattern.Kind),
			Code:   "missing-semantics",
			Detail: "no data is declared",
		}
	}
	return []candidate{{operationKind: operationKind, subject: string(pattern.Data), via: via}}, nil
}

func onRelation(
	pattern application.Pattern,
	relations relationTargets,
	via string,
	operationKinds ...string,
) ([]candidate, *Unlinked) {
	if pattern.Relation == "" {
		return nil, &Unlinked{
			Pattern: string(pattern.ID), Kind: string(pattern.Kind),
			Code:   "missing-semantics",
			Detail: "no relation is declared",
		}
	}
	target, ok := relations[pattern.Relation]
	if !ok {
		return nil, &Unlinked{
			Pattern: string(pattern.ID), Kind: string(pattern.Kind),
			Code:   "missing-input",
			Detail: "relation " + string(pattern.Relation) + " has no declared target",
		}
	}
	out := make([]candidate, 0, len(operationKinds))
	for _, kind := range operationKinds {
		out = append(out, candidate{operationKind: kind, subject: string(target), via: via})
	}
	return out, nil
}

func onCommand(
	pattern application.Pattern,
	capabilities map[core.CanonicalID]semantic.Capability,
) ([]candidate, *Unlinked) {
	if pattern.Capability == "" {
		return nil, &Unlinked{
			Pattern: string(pattern.ID), Kind: string(pattern.Kind),
			Code:   "missing-semantics",
			Detail: "no capability is declared",
		}
	}
	capability, ok := capabilities[pattern.Capability]
	if !ok {
		return nil, &Unlinked{
			Pattern: string(pattern.ID), Kind: string(pattern.Kind),
			Code:   "missing-input",
			Detail: "capability " + string(pattern.Capability) + " is not declared",
		}
	}
	if capability.Effect == nil {
		// A read-only command has no derived write operation to bind. This rule
		// does not add an effect to make the link appear.
		return nil, &Unlinked{
			Pattern: string(pattern.ID), Kind: string(pattern.Kind),
			Code:   "out-of-scope",
			Detail: "capability " + string(pattern.Capability) + " declares no effect, so no derived write operation exists to bind",
		}
	}
	kind := ""
	switch capability.Effect.Kind {
	case semantic.DataEffectCreate:
		kind = OperationCreate
	case semantic.DataEffectUpdate:
		kind = OperationUpdate
	default:
		return nil, &Unlinked{
			Pattern: string(pattern.ID), Kind: string(pattern.Kind),
			Code:   "out-of-scope",
			Detail: "effect " + string(capability.Effect.Kind) + " has no standard operation in this rule",
		}
	}
	return []candidate{{
		operationKind: kind, subject: string(capability.Effect.Data), via: ViaCapability,
	}}, nil
}

// Marshal renders the report as stable, indented JSON.
func (r Report) Marshal() ([]byte, error) {
	return json.MarshalIndent(r, "", "  ")
}
