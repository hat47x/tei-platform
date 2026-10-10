// Command reference-critical-lowering is a non-production contract pressure for #1121.
// It does not replace the TEI Driver, Capability resolver, or BuildArtifactLock.
package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
)

const (
	Schema = "tei.lowering-witness/v0"
	CandidateSchema = "tei.lowering-candidate/v0"
	AcceptanceSchema = "tei.lowering-acceptance-link/v0"
	NeutralSchema = "tei.lowering-neutral-plan/v0"
)

// Ref is an exact, externally defined identity. A digest does not establish Authority.
type Ref struct {
	ID          string `json:"id"`
	Version     string `json:"version"`
	Fingerprint string `json:"fingerprint"`
}

func (r Ref) valid() bool { return r.ID != "" && r.Version != "" && r.Fingerprint != "" }

type Subject struct {
	Kind string `json:"kind"` // business, data, function; owned by the semantic lane
	Ref  Ref    `json:"ref"`
}

// DataNeed describes a logical database Contract requirement, not a bound instance
// or a claim that a physical schema is currently compatible.
type DataNeed struct {
	Datasource string `json:"datasource"` // application-local alias, never a physical instance
	Database string `json:"database"`
	Contract string `json:"contract"`
	VersionConstraint string `json:"versionConstraint"`
}

func (n DataNeed) valid() bool {
	if n.Datasource == "" || n.Database == "" || n.Contract == "" || len(n.VersionConstraint) < 2 ||
		n.VersionConstraint[0] != '^' {
		return false
	}
	major, err := strconv.Atoi(n.VersionConstraint[1:])
	return err == nil && major > 0 && n.VersionConstraint == "^"+strconv.Itoa(major)
}

type Requirement struct {
	Contract Ref `json:"contract"`
	Scope string `json:"scope"`
	Effect string `json:"effect"`
	// Exact requirement belongs to Accepted meaning; current instance binding is Host-owned.
	DataContracts []DataNeed `json:"dataContracts,omitempty"`
	// This identity is a mandatory part of the accepted contract, not an optional sidecar.
	RequiredSlots []Ref `json:"requiredContextSlots"`
}

// Wiring is an application/target choice; it cannot modify Accepted meaning.
type Wiring struct {
	Scope    string `json:"scope"`
	Contract Ref    `json:"contract"`
	Provider Ref    `json:"provider"`
}

type Candidate struct {
	Schema       string        `json:"schema"`
	Source       Ref           `json:"source"`
	// Producer-declared source dependency witness; completeness belongs to Acceptance.
	SourceInputs []Ref         `json:"sourceInputs"`
	Language     Ref           `json:"language"`
	Subjects     []Subject     `json:"subjects"`
	Requirements []Requirement `json:"requirements"`
}

type Acceptance struct {
	Schema          string `json:"schema"`
	CandidateDigest string `json:"candidateDigest"`
	Snapshot        Ref    `json:"acceptedSnapshot"`
	Receipt         Ref    `json:"trustedAcceptanceReceipt"`
}

type Provider struct {
	Identity       Ref    `json:"identity"`
	Contract       Ref    `json:"contract"`
	Scope          string `json:"scope"`
	Target         string `json:"target"`
	Conformance    Ref    `json:"conformance"`
	ResolutionLock Ref    `json:"existingCapabilityResolutionLock"`
	Artifact       Ref    `json:"existingBuildArtifact"`
	Slots          []Ref  `json:"implementedContextSlots"`
	RuntimeDeps    []Ref  `json:"runtimeDependencies"`
	BuildOnlyDeps  []Ref  `json:"buildOnlyDependencies"`
}

type Target struct {
	ID      string   `json:"id"`
	Effects []string `json:"supportedEffects"`
}

type Step struct {
	Contract       Ref    `json:"contract"`
	Provider       Ref    `json:"provider"`
	DataContracts []DataNeed `json:"dataContracts,omitempty"`
	Scope          string `json:"scope"`
	Effect         string `json:"effect"`
	Conformance    Ref    `json:"conformance"`
	ResolutionLock Ref    `json:"capabilityResolutionLock"`
	Artifact       Ref    `json:"buildArtifact"`
	RequiredSlots  []Ref  `json:"requiredContextSlots"`
	RuntimeDeps    []Ref  `json:"runtimeDependencies"`
	Target         string `json:"target"`
}

// Origin is trace-only: it never substitutes for the accepted semantic snapshot.
type Origin struct {
	Source Ref `json:"source"`
	SourceInputs []Ref `json:"sourceInputs"`
	Language Ref `json:"language"`
	CandidateDigest string `json:"candidateDigest"`
	AcceptanceReceipt Ref `json:"acceptanceReceipt"`
}

// NeutralPlan records the minimum executable intent without Target, Provider or
// Environment Authority. It references accepted subjects; it is never authored.
type NeutralPlan struct {
	Schema   string        `json:"schema"`
	Snapshot Ref           `json:"snapshot"`
	Origin Origin          `json:"origin"`
	Intents  []Requirement `json:"intents"`
}

type Plan struct {
	Schema   string `json:"schema"`
	Snapshot Ref    `json:"snapshot"`
	Origin Origin   `json:"origin"`
	Steps    []Step `json:"steps"`
}

// Trust delegates to existing acceptance, publisher/conformance, and Host permission
// authorities. These callbacks must fail closed; never derive them from plugin claims.
type Trust interface {
	Accepted(candidateDigest string, receipt, snapshot Ref) bool
	Published(contract Ref) bool
	Conformant(provider Provider) bool
	// Trusted runtime checks the accepted snapshot, exact resolution lock and
	// artifact binding against this entire plan. A caller-supplied Plan is not Authority.
	ResolutionCurrent(plan Plan) bool
	// Host checks the current selected provider/artifact, not an earlier preview.
	RuntimeBindingCurrent(step Step) bool
	// Existing Data/Driver owner verifies current instance + contract version compatibility.
	// The callback must not infer equivalence from application and DB version numbers.
	DataCompatible(step Step) bool
	ContextBound(step Step) bool // trusted Host validates current, declared slot bindings
	Authorized(step Step) bool // call again at each invocation; not a cached grant
}

type Failure struct {
	Code string
}

func (f Failure) Error() string { return f.Code }
func reject(code string) error { return Failure{Code: code} }

// Digest is a deterministic serialization of struct-based evidence, not a signature.
// Its callers must not put secrets, current sessions, or access refs into these values.
func Digest(value any) string {
	b, err := json.Marshal(value)
	if err != nil {
		panic("reference digest cannot serialize: " + err.Error())
	}
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}

func equalSlots(a, b []Ref) bool {
	if len(a) != len(b) {
		return false
	}
	keys := func(refs []Ref) []string {
		out := make([]string, 0, len(refs))
		for _, r := range refs {
			out = append(out, r.ID+"\x00"+r.Version+"\x00"+r.Fingerprint)
		}
		sort.Strings(out)
		return out
	}
	x, y := keys(a), keys(b)
	for i := range x {
		if x[i] != y[i] || (i > 0 && x[i] == x[i-1]) {
			return false
		}
	}
	return true
}

func has(values []string, wanted string) bool {
	for _, v := range values {
		if v == wanted {
			return true
		}
	}
	return false
}

// Neutralize verifies that externally Accepted meaning is the source of intent.
// It cannot itself issue an acceptance receipt or validate a semantic graph.
func Neutralize(c Candidate, a Acceptance, trust Trust) (NeutralPlan, error) {
	if trust == nil || c.Schema != CandidateSchema || a.Schema != AcceptanceSchema ||
		!c.Source.valid() || !c.Language.valid() || !a.Snapshot.valid() || !a.Receipt.valid() {
		return NeutralPlan{}, reject("invalid-boundary")
	}
	// Source dependencies must not silently disappear between interpretation and
	// lowering. Completeness of this set must be checked by the real producer.
	if len(c.SourceInputs) == 0 {
		return NeutralPlan{}, reject("missing-source-dependencies")
	}
	seenSourceIDs := map[string]bool{c.Source.ID: true}
	previousSource := ""
	for _, src := range c.SourceInputs {
		if !src.valid() || seenSourceIDs[src.ID] || (previousSource != "" && previousSource >= src.ID) {
			return NeutralPlan{}, reject("invalid-source-dependencies")
		}
		seenSourceIDs[src.ID] = true
		previousSource = src.ID
	}
	d := Digest(c)
	if a.CandidateDigest != d || !trust.Accepted(d, a.Receipt, a.Snapshot) {
		return NeutralPlan{}, reject("unaccepted-meaning")
	}
	if len(c.Subjects) == 0 || len(c.Requirements) == 0 {
		return NeutralPlan{}, reject("invalid-boundary")
	}
	seen := make(map[string]bool)
	for _, s := range c.Subjects {
		if !s.Ref.valid() || s.Kind == "" || seen[s.Kind+":"+s.Ref.ID] {
			return NeutralPlan{}, reject("invalid-boundary")
		}
		seen[s.Kind+":"+s.Ref.ID] = true
	}
	intents := append([]Requirement(nil), c.Requirements...)
	seenRequirements := make(map[string]bool)
	for i := range intents {
		req := &intents[i]
		if !req.Contract.valid() || req.Scope == "" || req.Effect == "" {
			return NeutralPlan{}, reject("invalid-boundary")
		}
		key := req.Scope + ":" + req.Contract.ID
		if seenRequirements[key] {
			return NeutralPlan{}, reject("duplicate-capability")
		}
		seenRequirements[key] = true
		slots := make(map[string]bool)
		for _, slot := range req.RequiredSlots {
			if !slot.valid() || slots[slot.ID] {
				return NeutralPlan{}, reject("invalid-context-declaration")
			}
			slots[slot.ID] = true
		}
		req.RequiredSlots = append([]Ref(nil), req.RequiredSlots...)
		data := make(map[string]bool)
		for _, need := range req.DataContracts {
			if !need.valid() {
				return NeutralPlan{}, reject("invalid-data-requirement")
			}
			k := need.Datasource + ":" + need.Database + ":" + need.Contract
			if data[k] {
				return NeutralPlan{}, reject("duplicate-data-requirement")
			}
			data[k] = true
		}
		req.DataContracts = append([]DataNeed(nil), req.DataContracts...)
	}
	sort.Slice(intents, func(i, j int) bool {
		return intents[i].Scope+":"+intents[i].Contract.ID < intents[j].Scope+":"+intents[j].Contract.ID
	})
	return NeutralPlan{Schema: NeutralSchema, Snapshot: a.Snapshot, Origin: Origin{
		Source: c.Source, SourceInputs: append([]Ref(nil), c.SourceInputs...),
		Language: c.Language, CandidateDigest: d, AcceptanceReceipt: a.Receipt,
	}, Intents: intents}, nil
}

// Lower checks stage crossings only. Existing parsers, resolvers, and Driver still
// own meaning, capability selection, effect realization, and execution.
func Lower(c Candidate, a Acceptance, wiring []Wiring, providers []Provider, target Target, trust Trust) (Plan, error) {
	neutral, err := Neutralize(c, a, trust)
	if err != nil {
		return Plan{}, err
	}
	if target.ID == "" {
		return Plan{}, reject("unsupported-target")
	}
	steps := make([]Step, 0, len(neutral.Intents))
	seen := make(map[string]bool)
	for _, r := range neutral.Intents {
		if !r.Contract.valid() || r.Scope == "" || r.Effect == "" {
			return Plan{}, reject("invalid-boundary")
		}
		key := r.Scope + ":" + r.Contract.ID
		if seen[key] {
			return Plan{}, reject("duplicate-capability")
		}
		seen[key] = true
		if !has(target.Effects, r.Effect) {
			return Plan{}, reject("unsupported-target")
		}
		var pin *Wiring
		for i := range wiring {
			if wiring[i].Scope == r.Scope && wiring[i].Contract.ID == r.Contract.ID {
				if pin != nil {
					return Plan{}, reject("ambiguous-wiring")
				}
				pin = &wiring[i]
			}
		}
		if pin == nil || !pin.Provider.valid() {
			return Plan{}, reject("missing-wiring")
		}
		if pin.Contract != r.Contract {
			return Plan{}, reject("contract-drift")
		}
		var selected *Provider
		for i := range providers {
			if providers[i].Identity.ID == pin.Provider.ID {
				if providers[i].Identity.Version != pin.Provider.Version {
					continue
				}
				if selected != nil {
					return Plan{}, reject("ambiguous-provider")
				}
				selected = &providers[i]
			}
		}
		if selected == nil {
			for _, p := range providers {
				if p.Identity.ID == pin.Provider.ID {
					return Plan{}, reject("incompatible-version")
				}
			}
			return Plan{}, reject("missing-capability")
		}
		p := *selected
		if p.Identity != pin.Provider || p.Contract.ID != r.Contract.ID {
			return Plan{}, reject("contract-drift")
		}
		if p.Contract.Version != r.Contract.Version {
			return Plan{}, reject("incompatible-version")
		}
		if p.Contract.Fingerprint != r.Contract.Fingerprint {
			return Plan{}, reject("contract-drift")
		}
		if p.Scope != r.Scope || !p.Conformance.valid() || !p.ResolutionLock.valid() ||
			!p.Artifact.valid() || !equalSlots(r.RequiredSlots, p.Slots) {
			return Plan{}, reject("conformance-or-context-drift")
		}
		if p.Target != target.ID {
			return Plan{}, reject("unsupported-target")
		}
		if !trust.Published(p.Contract) || !trust.Conformant(p) {
			return Plan{}, reject("unverified-provider")
		}
		// Actual slot values, current Principal, BuildOnlyDeps, and credentials
		// must never appear in this target-neutral witness.
		steps = append(steps, Step{
			Contract: p.Contract, Provider: p.Identity, Scope: p.Scope, Effect: r.Effect,
			DataContracts: append([]DataNeed(nil), r.DataContracts...),
			Conformance: p.Conformance, ResolutionLock: p.ResolutionLock,
			Artifact: p.Artifact, RequiredSlots: append([]Ref(nil), r.RequiredSlots...),
			RuntimeDeps: append([]Ref(nil), p.RuntimeDeps...), Target: target.ID,
		})
	}
	sort.Slice(steps, func(i, j int) bool {
		return steps[i].Scope+":"+steps[i].Contract.ID < steps[j].Scope+":"+steps[j].Contract.ID
	})
	return Plan{Schema: Schema, Snapshot: neutral.Snapshot, Origin: neutral.Origin, Steps: steps}, nil
}

// AuthorizeInvocation is intentionally separate from deterministic lowering.
// The Host re-evaluates current grants/context bindings immediately before dispatch.
func AuthorizeInvocation(plan Plan, trust Trust) error {
	if trust == nil || plan.Schema != Schema || !plan.Snapshot.valid() || !plan.Origin.Source.valid() ||
		len(plan.Origin.SourceInputs) == 0 ||
		!plan.Origin.Language.valid() || !plan.Origin.AcceptanceReceipt.valid() ||
		plan.Origin.CandidateDigest == "" || len(plan.Steps) == 0 {
		return reject("invalid-boundary")
	}
	if !trust.ResolutionCurrent(plan) {
		return reject("stale-or-untrusted-resolution")
	}
	for _, s := range plan.Steps {
		if !trust.RuntimeBindingCurrent(s) {
			return reject("runtime-binding-unavailable")
		}
		if len(s.DataContracts) > 0 && !trust.DataCompatible(s) {
			return reject("data-contract-incompatible")
		}
		if len(s.RequiredSlots) > 0 && !trust.ContextBound(s) {
			return reject("missing-execution-context")
		}
		if !trust.Authorized(s) {
			return reject("authorization-denied")
		}
	}
	return nil
}

// CacheKeys model dependency edges, not a new execution IR or a replacement Lock.
// Unrelated providers and current session/grant values are deliberately excluded.
type CacheKeys struct {
	Interpret string
	Accept    string
	Resolve   string
	Target    string
	Build     string
}

// buildOnly is the selected builder's explicit dependency closure supplied by
// the existing Build Artifact owner; it stays out of runtime Plan/Step.
func Keys(c Candidate, a Acceptance, plan Plan, target Target, builder Ref, buildOnly []Ref) CacheKeys {
	interpret := Digest(struct {
		Source Ref
		SourceInputs []Ref
		Language Ref
	}{c.Source, c.SourceInputs, c.Language})
	accept := Digest(struct {
		Candidate string
		Acceptance Acceptance
	}{Digest(c), a})
	// Artifacts and Target are part of realization, not semantic composition.
	type resolvedStep struct {
		Contract, Provider, Conformance, ResolutionLock Ref
		Scope, Effect string
		DataContracts []DataNeed
		RequiredSlots, RuntimeDeps []Ref
	}
	resolved := make([]resolvedStep, 0, len(plan.Steps))
	for _, s := range plan.Steps {
		resolved = append(resolved, resolvedStep{
			Contract: s.Contract, Provider: s.Provider, Conformance: s.Conformance,
			ResolutionLock: s.ResolutionLock, Scope: s.Scope, Effect: s.Effect,
			DataContracts: s.DataContracts, RequiredSlots: s.RequiredSlots, RuntimeDeps: s.RuntimeDeps,
		})
	}
	resolve := Digest(struct {
		Accepted string
		Steps []resolvedStep
	}{accept, resolved})
	effects := append([]string(nil), target.Effects...)
	sort.Strings(effects) // Target effects are a capability set, not an ordered sequence.
	targetKey := Digest(struct {
		Resolved string
		Target Target
		Realization []Step
	}{resolve, Target{ID: target.ID, Effects: effects}, plan.Steps})
	buildInputs := append([]Ref(nil), buildOnly...)
	sort.Slice(buildInputs, func(i, j int) bool {
		if buildInputs[i].ID != buildInputs[j].ID {
			return buildInputs[i].ID < buildInputs[j].ID
		}
		if buildInputs[i].Version != buildInputs[j].Version {
			return buildInputs[i].Version < buildInputs[j].Version
		}
		return buildInputs[i].Fingerprint < buildInputs[j].Fingerprint
	})
	return CacheKeys{
		Interpret: interpret, Accept: accept, Resolve: resolve,
		Target: targetKey, Build: Digest(struct {
			Target string
			Builder Ref
			BuildOnlyDependencies []Ref
		}{targetKey, builder, buildInputs}),
	}
}

func main() {
	fmt.Println("reference only: go test ./tools/reference-critical-lowering")
}
