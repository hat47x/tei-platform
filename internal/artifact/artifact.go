// Package artifact defines the contract for derived, read-only documents.
//
// A derived artifact is produced from accepted meaning by a versioned rule. It
// is not a canonical source, it never grants acceptance, and it must be
// reproducible: the same input revision and the same rule must produce the same
// bytes. The unresolved list is part of the contract, so a missing declaration
// is reported instead of being filled with a default.
package artifact

import (
	"fmt"
	"sort"
	"strings"

	"tei/internal/sourceindex"
)

// Kind names one kind of derived artifact. The set is a rule vocabulary, not a
// fixed core enum: a producer may add its own kind together with its rule.
type Kind string

const (
	KindTableDefinition    Kind = "table-definition"
	KindFunctionDefinition Kind = "function-definition"
	KindCrossReference     Kind = "cross-reference"
	KindDDL                Kind = "ddl"
	KindOperationSurface   Kind = "operation-surface"
)

// Rule identifies the derivation rules that produced an artifact.
type Rule struct {
	ID          string `json:"id"`
	Version     string `json:"version"`
	Fingerprint string `json:"fingerprint"`
}

// Unresolved records a part of the artifact that could not be decided from the
// accepted meaning. An artifact with unresolved entries is still an artifact;
// it is not a successful one.
type Unresolved struct {
	Subject string `json:"subject"`
	Reason  string `json:"reason"`
	Detail  string `json:"detail"`
}

// Artifact is a derived document bound to its input revision and rules.
type Artifact struct {
	Kind          string       `json:"kind"`
	InputRevision string       `json:"inputRevision"`
	Rule          Rule         `json:"rule"`
	Generator     string       `json:"generator"`
	Content       string       `json:"content"`
	Digest        string       `json:"digest"`
	Unresolved    []Unresolved `json:"unresolved"`
}

// New builds an artifact and computes its digest. The unresolved list is sorted
// so that two producers that report the same gaps in a different order produce
// the same bytes and the same digest.
func New(kind Kind, rule Rule, generator, inputRevision, content string, unresolved []Unresolved) (Artifact, error) {
	if kind == "" {
		return Artifact{}, fmt.Errorf("artifact kind is required")
	}
	if rule.ID == "" || rule.Version == "" || rule.Fingerprint == "" {
		return Artifact{}, fmt.Errorf("artifact rule requires id, version and fingerprint")
	}
	if generator == "" {
		return Artifact{}, fmt.Errorf("artifact generator is required")
	}
	if inputRevision == "" {
		return Artifact{}, fmt.Errorf("artifact input revision is required")
	}
	sorted := append([]Unresolved(nil), unresolved...)
	sort.Slice(sorted, func(i, j int) bool {
		a, b := sorted[i], sorted[j]
		if a.Subject != b.Subject {
			return a.Subject < b.Subject
		}
		if a.Reason != b.Reason {
			return a.Reason < b.Reason
		}
		return a.Detail < b.Detail
	})
	artifact := Artifact{
		Kind:          string(kind),
		InputRevision: inputRevision,
		Rule:          rule,
		Generator:     generator,
		Content:       content,
		Unresolved:    sorted,
	}
	artifact.Digest = fingerprint(artifact)
	return artifact, nil
}

// Verify recomputes the digest. A caller that reads a stored artifact uses this
// to detect a changed content, rule or unresolved list.
func (a Artifact) Verify() error {
	if a.Digest != fingerprint(a) {
		return fmt.Errorf("artifact digest does not match its content")
	}
	return nil
}

// fingerprint covers the identity fields, the content and the unresolved list.
// It is not a Shared Acceptance revision and it does not authorize a write.
func fingerprint(a Artifact) string {
	var b strings.Builder
	b.WriteString("tei.artifact/v0\n")
	b.WriteString(a.Kind + "\n")
	b.WriteString(a.InputRevision + "\n")
	b.WriteString(a.Rule.ID + "\n")
	b.WriteString(a.Rule.Version + "\n")
	b.WriteString(a.Rule.Fingerprint + "\n")
	b.WriteString(a.Generator + "\n")
	b.WriteString(a.Content)
	b.WriteString("\n--unresolved--\n")
	for _, unresolved := range a.Unresolved {
		b.WriteString(unresolved.Subject + "\t" + unresolved.Reason + "\t" + unresolved.Detail + "\n")
	}
	return sourceindex.Fingerprint([]byte(b.String()))
}
