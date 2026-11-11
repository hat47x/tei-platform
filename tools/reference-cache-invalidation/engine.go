// Package cacheinvalidation is a read-only reference for a dependency-based
// cache invalidation engine. It tracks World/Model/Artifact/Evidence revisions,
// computes a deterministic dependency closure, and decides whether cached
// artifacts, builds, and evidence can each be reused.
//
// It is not a product cache, a distributed store, an authorization engine,
// or an accepted API. It never writes a Source. It defaults to the safe side
// when a dependency is unknown.
package cacheinvalidation

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

// RevisionKind distinguishes the kinds of revision tracked by the engine.
type RevisionKind string

const (
	RevisionKindWorld     RevisionKind = "world"
	RevisionKindModel     RevisionKind = "model"
	RevisionKindArtifact  RevisionKind = "artifact"
	RevisionKindEvidence  RevisionKind = "evidence"
	RevisionKindOperator  RevisionKind = "operator"
	RevisionKindPlugin    RevisionKind = "plugin"
	RevisionKindGenerator RevisionKind = "generator"
	RevisionKindSource    RevisionKind = "source"
)

// Revision is a content-addressed revision of a specific scope.
type Revision struct {
	Kind   RevisionKind `json:"kind"`
	Scope  string       `json:"scope"`
	Digest string       `json:"digest"`
	// Label is an optional human-readable label for auditability.
	Label string `json:"label,omitempty"`
}

// DependencyEdge describes that one revision set depends on another.
type DependencyEdge struct {
	// From is the consumer side.
	From RevisionKind `json:"from"`
	// To is the producer side.
	To RevisionKind `json:"to"`
	// Scope restricts the dependency to a named scope.
	Scope string `json:"scope"`
	// Required declares whether the dependency is mandatory for reuse.
	Required bool `json:"required"`
}

// CacheKey is the identity of a cached entry.
type CacheKey struct {
	// SourceDigest is the digest of the source input.
	SourceDigest string `json:"sourceDigest"`
	// ModelRevision is the model revision used.
	ModelRevision string `json:"modelRevision"`
	// WorldFingerprint is the world fingerprint at the time of computation.
	WorldFingerprint string `json:"worldFingerprint"`
	// ArtifactRevision is the artifact revision produced.
	ArtifactRevision string `json:"artifactRevision"`
	// EvidenceRevision is the evidence revision produced.
	EvidenceRevision string `json:"evidenceRevision"`
	// OperatorPin is the operator pin.
	OperatorPin string `json:"operatorPin,omitempty"`
	// PluginPin is the plugin pin.
	PluginPin string `json:"pluginPin,omitempty"`
	// GeneratorVersion is the generator version.
	GeneratorVersion string `json:"generatorVersion,omitempty"`
}

// EvidenceRecord describes a cached evidence entry.
type EvidenceRecord struct {
	// Scope is the evidence scope.
	Scope string `json:"scope"`
	// Revision is the evidence revision.
	Revision string `json:"revision"`
	// Subject is the evidence subject.
	Subject string `json:"subject"`
	// WorldFingerprint is the world fingerprint at evidence collection time.
	WorldFingerprint string `json:"worldFingerprint"`
	// ArtifactRevision is the artifact revision the evidence was collected for.
	ArtifactRevision string `json:"artifactRevision"`
}

// Status is the build status of a cached entry.
type Status string

const (
	// StatusComplete means the build finished normally.
	StatusComplete Status = "complete"
	// StatusFailed means the build failed.
	StatusFailed Status = "failed"
	// StatusPartial means the build is incomplete (mid-flight failure).
	StatusPartial Status = "partial"
)

// CachedEntry is a stored cache entry.
type CachedEntry struct {
	Key     CacheKey
	Status  Status
	Closure Closure
}

// Closure is a deterministic fingerprint of the transitive dependency set.
type Closure struct {
	// Digest is sha256:hex of the canonical closure encoding.
	Digest string `json:"digest"`
	// Members lists the revisions included in the closure.
	Members []Revision `json:"members"`
	// Unknown is true when some required dependency could not be resolved.
	Unknown bool `json:"unknown"`
	// Missing lists the names of unresolved dependencies when Unknown is true.
	Missing []string `json:"missing,omitempty"`
}

// ReuseDecision is the outcome for one reuse query.
type ReuseDecision struct {
	// Scope is the reuse scope queried.
	Scope string `json:"scope"`
	// Allowed is true when reuse is safe under the current rules.
	Allowed bool `json:"allowed"`
	// Reason explains why reuse is allowed or denied.
	Reason string `json:"reason"`
	// MustRecomputeArtifact is true when the artifact must be rebuilt.
	MustRecomputeArtifact bool `json:"mustRecomputeArtifact"`
	// MustRecomputeBuild is true when the build step must be rerun.
	MustRecomputeBuild bool `json:"mustRecomputeBuild"`
	// MustRecomputeEvidence is true when evidence must be recomputed.
	MustRecomputeEvidence bool `json:"mustRecomputeEvidence"`
	// StaleReasons lists the specific stale findings.
	StaleReasons []string `json:"staleReasons,omitempty"`
}

// Engine performs dependency-based cache invalidation.
type Engine struct {
	dependencies []DependencyEdge
}

// NewEngine creates an engine with the given dependency graph.
func NewEngine(dependencies []DependencyEdge) *Engine {
	return &Engine{dependencies: dependencies}
}

// ComputeClosure returns the closure fingerprint for the given revisions.
// The closure includes the transitive required dependencies from the graph.
// Unknown required dependencies are surfaced rather than silently ignored.
func (e *Engine) ComputeClosure(current []Revision) (Closure, error) {
	index := indexRevisions(current)
	members := make([]Revision, 0, len(current))
	seen := map[string]bool{}

	// Resolve transitive required dependencies.
	var queue []RevisionKind
	for _, dep := range e.dependencies {
		if !dep.Required {
			continue
		}
		if _, ok := index[dep.To]; !ok {
			return Closure{}, fmt.Errorf("missing required dependency: %s -> %s (%s)", dep.From, dep.To, dep.Scope)
		}
		if !seen[revisionKey(dep.To, index[dep.To])] {
			seen[revisionKey(dep.To, index[dep.To])] = true
			members = append(members, index[dep.To])
		}
		queue = append(queue, dep.To)
	}

	// Bounded transitive closure (one level is enough for the reference;
	// deeper graphs use the same rule recursively).
	for len(queue) > 0 {
		next := queue[0]
		queue = queue[1:]
		for _, dep := range e.dependencies {
			if dep.From == next && dep.Required {
				if _, ok := index[dep.To]; !ok {
					return Closure{}, fmt.Errorf("missing required transitive dependency: %s -> %s (%s)", dep.From, dep.To, dep.Scope)
				}
				if !seen[revisionKey(dep.To, index[dep.To])] {
					seen[revisionKey(dep.To, index[dep.To])] = true
					members = append(members, index[dep.To])
					queue = append(queue, dep.To)
				}
			}
		}
	}

	if len(members) == 0 {
		members = make([]Revision, 0)
	}

	sort.Slice(members, func(i, j int) bool {
		if members[i].Kind != members[j].Kind {
			return strings.Compare(string(members[i].Kind), string(members[j].Kind)) < 0
		}
		if members[i].Scope != members[j].Scope {
			return members[i].Scope < members[j].Scope
		}
		return members[i].Digest < members[j].Digest
	})

	b, err := json.Marshal(members)
	if err != nil {
		return Closure{}, fmt.Errorf("marshal closure members: %w", err)
	}
	digest := fmt.Sprintf("sha256:%x", sha256.Sum256(b))
	return Closure{Digest: digest, Members: members}, nil
}

// ReuseArtifact decides whether a previously produced artifact can be reused.
// Reuse requires exact source digest, model revision, world fingerprint,
// artifact revision, and transitive closure of required dependencies to be
// unchanged. Source byte equality alone is not sufficient.
func (e *Engine) ReuseArtifact(prev, current CacheKey, closure Closure) ReuseDecision {
	var reasons []string

	if prev.SourceDigest != current.SourceDigest {
		reasons = append(reasons, fmt.Sprintf("source digest changed %s -> %s", prev.SourceDigest, current.SourceDigest))
	}
	if prev.ModelRevision != current.ModelRevision {
		reasons = append(reasons, fmt.Sprintf("model revision changed %s -> %s", prev.ModelRevision, current.ModelRevision))
	}
	if prev.WorldFingerprint != current.WorldFingerprint {
		reasons = append(reasons, fmt.Sprintf("world fingerprint changed %s -> %s", prev.WorldFingerprint, current.WorldFingerprint))
	}
	if prev.ArtifactRevision != current.ArtifactRevision {
		reasons = append(reasons, fmt.Sprintf("artifact revision changed %s -> %s", prev.ArtifactRevision, current.ArtifactRevision))
	}
	if closure.Unknown {
		reasons = append(reasons, fmt.Sprintf("dependency closure is unknown (missing: %s)", strings.Join(closure.Missing, ", ")))
	}
	if prev.ArtifactRevision != current.ArtifactRevision || closure.Unknown {
		// Already captured above, keep for clarity.
	}

	allowed := len(reasons) == 0
	return ReuseDecision{
		Scope:                 "artifact",
		Allowed:               allowed,
		Reason:                decisionReason(allowed, "artifact identity and required dependency closure are unchanged", reasons),
		MustRecomputeArtifact: !allowed,
		StaleReasons:          reasons,
	}
}

// ReuseBuild decides whether the build step itself can be skipped.
// Build skip requires all artifact reuse conditions plus unchanged
// operator/plugin/generator pins and no dependency topology change.
func (e *Engine) ReuseBuild(prev, current CacheKey, closure Closure, dependenciesChanged bool, buildStatus Status) ReuseDecision {
	artifact := e.ReuseArtifact(prev, current, closure)
	var reasons []string
	reasons = append(reasons, artifact.StaleReasons...)

	if prev.OperatorPin != "" && current.OperatorPin != "" && prev.OperatorPin != current.OperatorPin {
		reasons = append(reasons, fmt.Sprintf("operator pin changed %s -> %s", prev.OperatorPin, current.OperatorPin))
	}
	if prev.PluginPin != "" && current.PluginPin != "" && prev.PluginPin != current.PluginPin {
		reasons = append(reasons, fmt.Sprintf("plugin pin changed %s -> %s", prev.PluginPin, current.PluginPin))
	}
	if prev.GeneratorVersion != "" && current.GeneratorVersion != "" && prev.GeneratorVersion != current.GeneratorVersion {
		reasons = append(reasons, fmt.Sprintf("generator version changed %s -> %s", prev.GeneratorVersion, current.GeneratorVersion))
	}
	if dependenciesChanged {
		reasons = append(reasons, "dependency topology changed (add/remove)")
	}
	if buildStatus == StatusPartial || buildStatus == StatusFailed {
		reasons = append(reasons, fmt.Sprintf("previous build status is %s", buildStatus))
	}

	allowed := len(reasons) == 0
	return ReuseDecision{
		Scope:                 "build",
		Allowed:               allowed,
		Reason:                decisionReason(allowed, "build inputs, toolchain pins, closure, and previous build status are unchanged", reasons),
		MustRecomputeArtifact: !allowed,
		MustRecomputeBuild:    !allowed,
		MustRecomputeEvidence: !allowed,
		StaleReasons:          reasons,
	}
}

// ReuseEvidence decides whether cached evidence can be reused.
// Evidence reuse is the most conservative: it requires matching scope,
// revision, subject, world fingerprint, artifact revision, and closure.
// A→B→A cycles without continuity proof are rejected.
func (e *Engine) ReuseEvidence(prev, current EvidenceRecord, closure Closure, continuityProof bool) ReuseDecision {
	var reasons []string

	if prev.Scope != current.Scope {
		reasons = append(reasons, fmt.Sprintf("evidence scope changed %s -> %s", prev.Scope, current.Scope))
	}
	if prev.Revision != current.Revision {
		reasons = append(reasons, fmt.Sprintf("evidence revision changed %s -> %s", prev.Revision, current.Revision))
	}
	if prev.Subject != current.Subject {
		reasons = append(reasons, fmt.Sprintf("evidence subject changed %s -> %s", prev.Subject, current.Subject))
	}
	if prev.WorldFingerprint != current.WorldFingerprint {
		reasons = append(reasons, fmt.Sprintf("world fingerprint changed %s -> %s", prev.WorldFingerprint, current.WorldFingerprint))
	}
	if prev.ArtifactRevision != current.ArtifactRevision {
		reasons = append(reasons, fmt.Sprintf("artifact revision changed %s -> %s", prev.ArtifactRevision, current.ArtifactRevision))
	}
	if closure.Unknown {
		reasons = append(reasons, fmt.Sprintf("dependency closure is unknown (missing: %s)", strings.Join(closure.Missing, ", ")))
	}

	// A->B->A cycle detection: if world fingerprint matches but artifact
	// revision differs, intermediate changes occurred. Without explicit
	// continuity proof, the evidence cannot be reused.
	if prev.WorldFingerprint == current.WorldFingerprint && prev.ArtifactRevision != current.ArtifactRevision {
		if !continuityProof {
			reasons = append(reasons, "A->B->A cycle: world fingerprint matches but artifact revision differs; evidence continuity is unproven")
		}
	}

	allowed := len(reasons) == 0
	return ReuseDecision{
		Scope:                 "evidence",
		Allowed:               allowed,
		Reason:                decisionReason(allowed, "evidence scope, revision, subject, artifact revision, world fingerprint, and closure are unchanged", reasons),
		MustRecomputeEvidence: !allowed,
		StaleReasons:          reasons,
	}
}

// DependencyChanged reports whether the dependency topology changed between
// previous and current edge sets. It compares by canonical form.
func DependencyChanged(prev, current []DependencyEdge) bool {
	prevSet := canonicalDeps(prev)
	currSet := canonicalDeps(current)
	if len(prevSet) != len(currSet) {
		return true
	}
	for k := range prevSet {
		if !currSet[k] {
			return true
		}
	}
	return false
}

func indexRevisions(revs []Revision) map[RevisionKind]Revision {
	index := make(map[RevisionKind]Revision, len(revs))
	for _, r := range revs {
		index[r.Kind] = r
	}
	return index
}

func revisionKey(kind RevisionKind, rev Revision) string {
	return fmt.Sprintf("%s|%s|%s", kind, rev.Scope, rev.Digest)
}

func canonicalDeps(deps []DependencyEdge) map[string]bool {
	out := make(map[string]bool, len(deps))
	for _, d := range deps {
		key := fmt.Sprintf("%s->%s|%s|%t", d.From, d.To, d.Scope, d.Required)
		out[key] = true
	}
	return out
}

func decisionReason(allowed bool, ok string, reasons []string) string {
	if allowed {
		return ok
	}
	return fmt.Sprintf("reuse denied: %s", strings.Join(reasons, "; "))
}
