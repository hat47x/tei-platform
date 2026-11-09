// Package digestchangereuse is a read-only reference for CON-12c on the layer
// that actually carries identity fields: generation.Binding.
//
// CON-12c asks: when the artifact digest changes, does anything keep reusing the
// old binding? This reference pins a lease under the old digest, changes the
// digest, and asks the same three questions after the change:
//
//  1. does a NEW acquisition get the old digest, or the new one?
//  2. does the OLD lease still report the digest it was pinned to?
//  3. does a lease acquired after the change carry the new digest?
//
// It uses only the existing Router. It adds no resolver and no second authority.
package digestchangereuse

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"

	"tei/core/generation"
	"tei/internal/composition"
)

const (
	Schema    = "tei.reference.digest-change-reuse/v0"
	Authority = "derived-reference"

	RuleID      = "tei.reference.digest-change-reuse"
	RuleVersion = "v0"
)

const rulePolicy = `digest-change-reuse/v0
path: generation.Router (the type composition.Router aliases)
pin: Router.Acquire, which locks the current generation for one invocation
change: Router.Publish with a new generation number and a new artifact digest
observed: the digest the lease reports, before and after the change
not verified: that the new artifact exists, is trusted, or was verified
not issued: authentication, authority, permission, or a durable receipt
`

// Digest values used by the runs. They are structurally valid sha256 strings,
// not real artifact digests.
const (
	OldDigest = "sha256:0000000000000000000000000000000000000000000000000000000000000001"
	NewDigest = "sha256:0000000000000000000000000000000000000000000000000000000000000002"
)

// Outcome is one question's answer.
type Outcome struct {
	Name string `json:"name"`
	// Digest is the artifact digest the path reported for this question.
	Digest string `json:"digest,omitempty"`
	// Matches describes which digest that is, in words, so a reader does not
	// have to compare hex strings.
	Matches string `json:"matches"`
	Detail  string `json:"detail,omitempty"`
}

// Report is the whole answer.
type Report struct {
	Schema          string    `json:"schema"`
	Authority       string    `json:"authority"`
	RuleID          string    `json:"ruleId"`
	RuleVersion     string    `json:"ruleVersion"`
	RuleFingerprint string    `json:"ruleFingerprint"`
	OldDigest       string    `json:"oldDigest"`
	NewDigest       string    `json:"newDigest"`
	Outcomes        []Outcome `json:"outcomes"`
	// OldLeaseReused reports whether a lease acquired before the change still
	// reports the old digest afterwards. Pinning means it should.
	OldLeaseReused bool `json:"oldLeaseStillPinned"`
	// NewLeaseUsesNewDigest reports whether a lease acquired after the change
	// carries the new digest.
	NewLeaseUsesNewDigest bool     `json:"newLeaseUsesNewDigest"`
	Notes                 []string `json:"notes"`
}

// Run performs the three questions. It never invents an artifact digest and it
// never performs a real cutover: the change happens inside one Router.
func Run() (Report, error) {
	report := Report{
		Schema: Schema, Authority: Authority,
		RuleID: RuleID, RuleVersion: RuleVersion,
		RuleFingerprint: fingerprint(),
		OldDigest:       OldDigest,
		NewDigest:       NewDigest,
		Outcomes:        []Outcome{},
		Notes:           notes(),
	}

	old := Binding(1, OldDigest)
	router, err := generation.NewRouter(old, composition.NewScope())
	if err != nil {
		return Report{}, fmt.Errorf("create router: %w", err)
	}

	// Pin the old generation, as an in-flight invocation would.
	oldLease, err := router.Acquire(func(generation.Binding) error { return nil })
	if err != nil {
		return Report{}, fmt.Errorf("pin the old generation: %w", err)
	}
	defer oldLease.Release()

	report.Outcomes = append(report.Outcomes, Outcome{
		Name:    "before the change",
		Digest:  oldLease.Binding.ArtifactDigest,
		Matches: matches(oldLease.Binding.ArtifactDigest),
	})

	// The change: a new generation with a new digest.
	next := Binding(2, NewDigest)
	if _, err := router.Publish(1, next, composition.NewScope(),
		func(generation.Binding) error { return nil }); err != nil {
		return Report{}, fmt.Errorf("publish the new generation: %w", err)
	}

	// Question 1: a new acquisition after the change.
	freshLease, err := router.Acquire(func(generation.Binding) error { return nil })
	if err != nil {
		return Report{}, fmt.Errorf("acquire after the change: %w", err)
	}
	defer freshLease.Release()

	report.NewLeaseUsesNewDigest = freshLease.Binding.ArtifactDigest == NewDigest
	report.Outcomes = append(report.Outcomes, Outcome{
		Name:    "a new acquisition after the change",
		Digest:  freshLease.Binding.ArtifactDigest,
		Matches: matches(freshLease.Binding.ArtifactDigest),
	})

	// Question 2: the lease pinned before the change.
	report.OldLeaseReused = oldLease.Binding.ArtifactDigest == OldDigest
	report.Outcomes = append(report.Outcomes, Outcome{
		Name:    "the lease pinned before the change",
		Digest:  oldLease.Binding.ArtifactDigest,
		Matches: matches(oldLease.Binding.ArtifactDigest),
	})

	return report, nil
}

// Binding builds a structurally valid binding. Generation 0 is invalid, so the
// caller must supply a non-zero one.
func Binding(gen uint64, digest string) generation.Binding {
	return generation.Binding{
		ContractID:      "sales.quote-lines",
		ContractVersion: "1.0.0",
		ProviderID:      "logical-quote",
		ArtifactDigest:  digest,
		Generation:      gen,
	}
}

func matches(digest string) string {
	switch digest {
	case OldDigest:
		return "the OLD digest"
	case NewDigest:
		return "the NEW digest"
	case "":
		return "no digest at all"
	default:
		return "some other digest"
	}
}

func notes() []string {
	return []string{
		"the change happens inside one Router; it performs no real cutover",
		"a new acquisition after the change is expected to carry the new digest",
		"a lease pinned before the change is expected to keep the old digest",
		"neither digest was produced by a real artifact build or verified by a Host",
		"no authentication, authority, permission or durable receipt is issued here",
	}
}

func fingerprint() string { return fmt.Sprintf("sha256:%x", sha256.Sum256([]byte(rulePolicy))) }

// Marshal renders the report as stable, indented JSON.
func (r Report) Marshal() ([]byte, error) { return json.MarshalIndent(r, "", "  ") }
