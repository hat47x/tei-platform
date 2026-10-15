// Package generatedcode decides how a generated-code artifact connects to TEI: whether
// it can be called through the TEI contract boundary, which artifacts stay as native
// code, and which parts are promoted into TEI Data/Function/Business models. It is
// read-only. Connecting an artifact never makes the artifact's execution meaning or
// authority accepted: only the contract around it can be accepted.
//
// The reference does not read or run the generated code. It consumes the interface and
// dependencies a caller declared and enforces the boundary.
package generatedcode

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"sort"
)

const Schema = "tei.reference.generated-code-connection/v0"

// Interface is the declared input and output of an artifact.
type Interface struct {
	Inputs  []string `json:"inputs"`
	Outputs []string `json:"outputs"`
}

// Artifact is one generated code unit with its declared dependencies.
type Artifact struct {
	ID          string     `json:"id"`
	Language    string     `json:"language"`
	Interface   *Interface `json:"interface,omitempty"`
	Authority   []string   `json:"authority"`
	External    []string   `json:"external"`
	DBScope     []string   `json:"dbScope"`
	SideEffects []string   `json:"sideEffects"`
	Providers   []string   `json:"providers"`
	// Contract is the contract revision the artifact was written against.
	Contract string `json:"contract"`
}

// Promotion is a caller's proposal to promote an artifact into a TEI model.
type Promotion struct {
	Artifact string `json:"artifact"`
	Kind     string `json:"kind"`  // function | data | business
	Model    string `json:"model"` // TEI model identifier
}

// Input is one read-only analysis request.
type Input struct {
	Artifacts      []Artifact  `json:"artifacts"`
	KnownAuthority []string    `json:"knownAuthority"`
	KnownContracts []string    `json:"knownContracts"`
	KnownProviders []string    `json:"knownProviders"`
	KnownExternal  []string    `json:"knownExternal"`
	Promotions     []Promotion `json:"promotions"`
}

// Finding is a diagnostic. It never grants a meaning or an authority.
type Finding struct {
	Code    string `json:"code"`
	Subject string `json:"subject"`
}

// Connection is the connection status of one artifact.
type Connection struct {
	Artifact string `json:"artifact"`
	Mode     string `json:"mode"` // capability | native | unsupported
	Callable bool   `json:"callable"`
	Reason   string `json:"reason,omitempty"`
}

// PromotionView records a proposed promotion. MeaningAccepted is always false: the
// execution meaning of the generated code is not accepted by connecting it.
type PromotionView struct {
	Artifact        string `json:"artifact"`
	Kind            string `json:"kind"`
	Model           string `json:"model"`
	Accepted        bool   `json:"accepted"`
	MeaningAccepted bool   `json:"meaningAccepted"`
}

// Summary counts the connection outcomes.
type Summary struct {
	Total       int `json:"total"`
	Connectable int `json:"connectable"`
	Native      int `json:"native"`
	Unsupported int `json:"unsupported"`
	Promoted    int `json:"promoted"`
	Retained    int `json:"retained"`
}

// Report is the read-only result.
type Report struct {
	Schema         string          `json:"schema"`
	Connections    []Connection    `json:"connections"`
	Promotions     []PromotionView `json:"promotions"`
	Unsupported    []string        `json:"unsupported"`
	RetainedNative []string        `json:"retainedNative"`
	Summary        Summary         `json:"summary"`
	Findings       []Finding       `json:"findings"`
	Fingerprint    string          `json:"fingerprint"`
}

func set(list []string) map[string]bool {
	m := make(map[string]bool, len(list))
	for _, s := range list {
		m[s] = true
	}
	return m
}

// callableReason returns an empty string when the artifact can be called through the
// contract boundary, or the first blocking reason.
func callableReason(a Artifact, known Input) string {
	if a.Interface == nil || len(a.Interface.Inputs) == 0 || len(a.Interface.Outputs) == 0 {
		return "missing-interface"
	}
	authority := set(known.KnownAuthority)
	for _, need := range a.Authority {
		if !authority[need] {
			return "unknown-authority"
		}
	}
	contracts := set(known.KnownContracts)
	if !contracts[a.Contract] {
		return "stale-contract"
	}
	external := set(known.KnownExternal)
	for _, call := range a.External {
		if !external[call] {
			return "external-call-outside-contract"
		}
	}
	providers := set(known.KnownProviders)
	for _, p := range a.Providers {
		if !providers[p] {
			return "unsupported-provider"
		}
	}
	return ""
}

// Plan decides the connection mode of each artifact and records the promotions. A
// promotion of a non-callable artifact is refused.
func Plan(in Input) Report {
	catalog := map[string]Artifact{}
	artifacts := append([]Artifact(nil), in.Artifacts...)
	sort.Slice(artifacts, func(i, j int) bool { return artifacts[i].ID < artifacts[j].ID })
	for _, a := range artifacts {
		catalog[a.ID] = a
	}

	findings := []Finding{}
	connections := make([]Connection, 0, len(artifacts))
	reasonOf := map[string]string{}
	for _, a := range artifacts {
		reason := callableReason(a, in)
		reasonOf[a.ID] = reason
		c := Connection{Artifact: a.ID, Callable: reason == "", Reason: reason}
		if reason != "" {
			c.Mode = "unsupported"
			findings = append(findings, Finding{Code: reason, Subject: a.ID})
		} else {
			c.Mode = "native"
		}
		connections = append(connections, c)
	}

	promotions := append([]Promotion(nil), in.Promotions...)
	sort.Slice(promotions, func(i, j int) bool {
		if promotions[i].Artifact != promotions[j].Artifact {
			return promotions[i].Artifact < promotions[j].Artifact
		}
		return promotions[i].Model < promotions[j].Model
	})
	views := make([]PromotionView, 0, len(promotions))
	promoted := map[string]bool{}
	for _, p := range promotions {
		view := PromotionView{Artifact: p.Artifact, Kind: p.Kind, Model: p.Model, MeaningAccepted: false}
		switch {
		case reasonOf[p.Artifact] != "":
			view.Accepted = false
			findings = append(findings, Finding{Code: "capability-not-callable", Subject: p.Artifact})
		case reasonOf[p.Artifact] == "" && known(catalog, p.Artifact):
			view.Accepted = true
			promoted[p.Artifact] = true
		default:
			view.Accepted = false
			findings = append(findings, Finding{Code: "promotion-of-unknown-artifact", Subject: p.Artifact})
		}
		views = append(views, view)
	}
	if len(views) > 0 {
		findings = append(findings, Finding{Code: "external-code-not-accepted", Subject: "all"})
	}

	// Apply the accepted promotions to the connection modes.
	for i := range connections {
		if promoted[connections[i].Artifact] {
			connections[i].Mode = "capability"
		}
	}

	unsupported := []string{}
	retained := []string{}
	summary := Summary{Total: len(artifacts)}
	for _, c := range connections {
		switch c.Mode {
		case "capability":
			summary.Connectable++
			summary.Promoted++
		case "native":
			summary.Connectable++
			summary.Native++
			retained = append(retained, c.Artifact)
		default:
			summary.Unsupported++
			unsupported = append(unsupported, c.Artifact)
		}
	}
	summary.Retained = len(retained)
	if len(retained) > 0 {
		findings = append(findings, Finding{Code: "native-code-retained", Subject: fmt.Sprint(len(retained))})
	}
	sortFindings(findings)

	return Report{
		Schema:         Schema,
		Connections:    connections,
		Promotions:     views,
		Unsupported:    unsupported,
		RetainedNative: retained,
		Summary:        summary,
		Findings:       findings,
		Fingerprint:    fingerprint(connections, views),
	}
}

func known(catalog map[string]Artifact, id string) bool {
	_, ok := catalog[id]
	return ok
}

func fingerprint(connections []Connection, promotions []PromotionView) string {
	payload := struct {
		Schema      string          `json:"schema"`
		Connections []Connection    `json:"connections"`
		Promotions  []PromotionView `json:"promotions"`
	}{Schema, connections, promotions}
	raw, err := json.Marshal(payload)
	if err != nil {
		raw = []byte(fmt.Sprintf("%v", payload))
	}
	return fmt.Sprintf("sha256:%x", sha256.Sum256(raw))
}

// ModeChange records a connection mode change of one artifact.
type ModeChange struct {
	Artifact string `json:"artifact"`
	From     string `json:"from"`
	To       string `json:"to"`
}

// Comparison measures the differences between two connection plans.
type Comparison struct {
	TotalDelta       int          `json:"totalDelta"`
	ConnectableDelta int          `json:"connectableDelta"`
	NativeDelta      int          `json:"nativeDelta"`
	UnsupportedDelta int          `json:"unsupportedDelta"`
	PromotedDelta    int          `json:"promotedDelta"`
	RetainedDelta    int          `json:"retainedDelta"`
	ModeChanges      []ModeChange `json:"modeChanges"`
	Added            []string     `json:"added"`
	Removed          []string     `json:"removed"`
}

// Compare plans both inputs and measures the change, so a caller can see coexistence,
// change, acceptance, connection and retention deltas.
func Compare(before, after Input) Comparison {
	b := Plan(before)
	a := Plan(after)

	modes := func(r Report) map[string]string {
		m := map[string]string{}
		for _, c := range r.Connections {
			m[c.Artifact] = c.Mode
		}
		return m
	}
	bm, am := modes(b), modes(a)

	changes := []ModeChange{}
	added := []string{}
	removed := []string{}
	for id, to := range am {
		from, ok := bm[id]
		if !ok {
			added = append(added, id)
			continue
		}
		if from != to {
			changes = append(changes, ModeChange{Artifact: id, From: from, To: to})
		}
	}
	for id := range bm {
		if _, ok := am[id]; !ok {
			removed = append(removed, id)
		}
	}
	sort.Strings(added)
	sort.Strings(removed)
	sort.Slice(changes, func(i, j int) bool { return changes[i].Artifact < changes[j].Artifact })

	return Comparison{
		TotalDelta:       a.Summary.Total - b.Summary.Total,
		ConnectableDelta: a.Summary.Connectable - b.Summary.Connectable,
		NativeDelta:      a.Summary.Native - b.Summary.Native,
		UnsupportedDelta: a.Summary.Unsupported - b.Summary.Unsupported,
		PromotedDelta:    a.Summary.Promoted - b.Summary.Promoted,
		RetainedDelta:    a.Summary.Retained - b.Summary.Retained,
		ModeChanges:      changes,
		Added:            added,
		Removed:          removed,
	}
}

func sortFindings(findings []Finding) {
	sort.Slice(findings, func(i, j int) bool {
		if findings[i].Code != findings[j].Code {
			return findings[i].Code < findings[j].Code
		}
		return findings[i].Subject < findings[j].Subject
	})
}
