// Command reference-critical-lowering is a non-production contract pressure for #1121.
// It does not replace the TEI Driver, Capability resolver, or BuildArtifactLock.
package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
)

const Schema = "tei.lowering-witness/v0"

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

type Requirement struct {
	Contract Ref    `json:"contract"`
	Scope    string `json:"scope"`
	Effect   string `json:"effect"`
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
	Source       Ref           `json:"source"`
	Language     Ref           `json:"language"`
	Subjects     []Subject     `json:"subjects"`
	Requirements []Requirement `json:"requirements"`
}

type Acceptance struct {
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
	Scope          string `json:"scope"`
	Effect         string `json:"effect"`
	Conformance    Ref    `json:"conformance"`
	ResolutionLock Ref    `json:"capabilityResolutionLock"`
	Artifact       Ref    `json:"buildArtifact"`
	RequiredSlots  []Ref  `json:"requiredContextSlots"`
	RuntimeDeps    []Ref  `json:"runtimeDependencies"`
	Target         string `json:"target"`
}

type Plan struct {
	Schema   string `json:"schema"`
	Snapshot Ref    `json:"snapshot"`
	Steps    []Step `json:"steps"`
}

// Trust delegates to existing acceptance, publisher/conformance, and Host permission
// authorities. These callbacks must fail closed; never derive them from plugin claims.
type Trust interface {
	Accepted(candidateDigest string, receipt, snapshot Ref) bool
	Published(contract Ref) bool
	Conformant(provider Provider) bool
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
	b, _ := json.Marshal(value)
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

// Lower checks stage crossings only. Existing parsers, resolvers, and Driver still
// own meaning, capability selection, effect realization, and execution.
func Lower(c Candidate, a Acceptance, wiring []Wiring, providers []Provider, target Target, trust Trust) (Plan, error) {
	if trust == nil || !c.Source.valid() || !c.Language.valid() || !a.Snapshot.valid() || !a.Receipt.valid() {
		return Plan{}, reject("invalid-boundary")
	}
	d := Digest(c)
	if a.CandidateDigest != d || !trust.Accepted(d, a.Receipt, a.Snapshot) {
		return Plan{}, reject("unaccepted-meaning")
	}
	if target.ID == "" {
		return Plan{}, reject("unsupported-target")
	}
	if len(c.Subjects) == 0 || len(c.Requirements) == 0 {
		return Plan{}, reject("invalid-boundary")
	}
	seenSubjects := make(map[string]bool)
	for _, s := range c.Subjects {
		if !s.Ref.valid() || s.Kind == "" || seenSubjects[s.Kind+":"+s.Ref.ID] {
			return Plan{}, reject("invalid-boundary")
		}
		seenSubjects[s.Kind+":"+s.Ref.ID] = true
	}
	steps := make([]Step, 0, len(c.Requirements))
	seen := make(map[string]bool)
	for _, r := range c.Requirements {
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
			Conformance: p.Conformance, ResolutionLock: p.ResolutionLock,
			Artifact: p.Artifact, RequiredSlots: append([]Ref(nil), r.RequiredSlots...),
			RuntimeDeps: append([]Ref(nil), p.RuntimeDeps...), Target: target.ID,
		})
	}
	sort.Slice(steps, func(i, j int) bool {
		return steps[i].Scope+":"+steps[i].Contract.ID < steps[j].Scope+":"+steps[j].Contract.ID
	})
	return Plan{Schema: Schema, Snapshot: a.Snapshot, Steps: steps}, nil
}

// AuthorizeInvocation is intentionally separate from deterministic lowering.
// The Host re-evaluates current grants/context bindings immediately before dispatch.
func AuthorizeInvocation(plan Plan, trust Trust) error {
	if trust == nil || plan.Schema != Schema || !plan.Snapshot.valid() || len(plan.Steps) == 0 {
		return reject("invalid-boundary")
	}
	for _, s := range plan.Steps {
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

func Keys(c Candidate, a Acceptance, plan Plan, target Target, builder Ref) CacheKeys {
	interpret := Digest(struct{ Source, Language Ref }{c.Source, c.Language})
	accept := Digest(struct {
		Candidate string
		Acceptance Acceptance
	}{Digest(c), a})
	resolve := Digest(struct {
		Accepted string
		Steps    []Step
	}{accept, plan.Steps})
	targetKey := Digest(struct {
		Resolved string
		Target   Target
	}{resolve, target})
	return CacheKeys{
		Interpret: interpret, Accept: accept, Resolve: resolve,
		Target: targetKey, Build: Digest(struct {
			Target string
			Builder Ref
		}{targetKey, builder}),
	}
}

func main() {
	fmt.Println("reference only: go test ./tools/reference-critical-lowering")
}
