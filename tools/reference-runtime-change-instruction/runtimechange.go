// Package runtimechange maps a developer instruction about a running screen element to
// the design subjects it reaches, separates appearance-only changes from business-rule
// and execution-authorization changes, and refuses to apply a change that was not
// accepted. It is read-only: it never writes a Source, never treats a screen-side
// disable as a server-side authorization, and never guesses an unknown mapping.
//
// The reference does not inspect a live screen. It consumes the element-to-subject
// bindings a caller (a developer tool) resolved, and enforces the boundaries around a
// change instruction.
package runtimechange

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

const Schema = "tei.reference.runtime-change-instruction/v0"

// Binding is the caller-supplied link from a rendered element to design subjects.
type Binding struct {
	Element string `json:"element"`
	Part    string `json:"part"` // generated | native | custom
	// Resolved is false when the mapping from the element to a design subject is
	// unknown. An unresolved binding must not be guessed.
	Resolved bool     `json:"resolved"`
	Subjects []string `json:"subjects"`
	Kinds    []string `json:"kinds"` // presentation | data | action | authorization
}

// Instruction is a developer request about one element.
type Instruction struct {
	ID      string `json:"id"`
	Element string `json:"element"`
	// Target is the proposed category: presentation | business-rule | authorization.
	Target string `json:"target"`
	// ScreenOnly is true when the request only hides or disables the element in the
	// screen. That never establishes a server-side authorization.
	ScreenOnly bool `json:"screenOnly"`
}

// Input is one read-only analysis request.
type Input struct {
	Bindings     []Binding     `json:"bindings"`
	Instructions []Instruction `json:"instructions"`
	// Accepted lists the instruction IDs a human accepted. An unaccepted change leaves
	// the Source unchanged.
	Accepted []string `json:"accepted"`
}

// Finding is a diagnostic. It never applies a change.
type Finding struct {
	Code    string `json:"code"`
	Subject string `json:"subject"`
}

// Proposal is one classified instruction.
type Proposal struct {
	Instruction        string   `json:"instruction"`
	Element            string   `json:"element"`
	Category           string   `json:"category"` // presentation | business-rule | authorization | unresolved
	Resolved           bool     `json:"resolved"`
	Subjects           []string `json:"subjects"`
	Support            string   `json:"support"` // supported | review-required | unsupported
	RequiresAcceptance bool     `json:"requiresAcceptance"`
	ScreenOnly         bool     `json:"screenOnly"`
}

// Applied is one accepted change, per subject.
type Applied struct {
	Instruction string `json:"instruction"`
	Element     string `json:"element"`
	Category    string `json:"category"`
	Subject     string `json:"subject"`
}

// Report is the read-only result.
type Report struct {
	Schema             string     `json:"schema"`
	Proposals          []Proposal `json:"proposals"`
	AppearanceOnly     []string   `json:"appearanceOnly"`
	BusinessRule       []string   `json:"businessRule"`
	Authorization      []string   `json:"authorization"`
	Unresolved         []string   `json:"unresolved"`
	Applied            []Applied  `json:"applied"`
	SourceChanged      bool       `json:"sourceChanged"`
	MeaningFingerprint string     `json:"meaningFingerprint"`
	Findings           []Finding  `json:"findings"`
}

func supportFor(part string) string {
	switch part {
	case "generated":
		return "supported"
	case "native":
		return "review-required"
	case "custom":
		return "unsupported"
	default:
		return "unsupported"
	}
}

// Plan classifies each instruction without applying anything. A change needs acceptance
// unless it is appearance-only.
func Plan(in Input) Report {
	bindings := map[string]Binding{}
	for _, b := range in.Bindings {
		bindings[b.Element] = b
	}

	proposals := make([]Proposal, 0, len(in.Instructions))
	appearance := []string{}
	business := []string{}
	authz := []string{}
	unresolved := []string{}
	findings := []Finding{}

	for _, ins := range in.Instructions {
		p := Proposal{
			Instruction:        ins.ID,
			Element:            ins.Element,
			Category:           ins.Target,
			RequiresAcceptance: ins.Target != "presentation",
			ScreenOnly:         ins.ScreenOnly,
		}
		b, ok := bindings[ins.Element]
		switch {
		case !ok:
			p.Category = "unresolved"
			p.Resolved = false
			p.Support = "unsupported"
			findings = append(findings, Finding{Code: "unknown-element", Subject: ins.Element})
		case !b.Resolved:
			p.Category = "unresolved"
			p.Resolved = false
			p.Support = supportFor(b.Part)
			findings = append(findings, Finding{Code: "unresolved-binding", Subject: ins.Element})
		case len(b.Subjects) == 0:
			p.Category = "unresolved"
			p.Resolved = false
			p.Support = supportFor(b.Part)
			findings = append(findings, Finding{Code: "binding-without-subject", Subject: ins.Element})
		default:
			p.Resolved = true
			p.Subjects = append([]string(nil), b.Subjects...)
			sort.Strings(p.Subjects)
			p.Support = supportFor(b.Part)
			if p.Support == "unsupported" {
				findings = append(findings, Finding{Code: "unsupported-part", Subject: ins.Element})
			}
			// Hiding or disabling a control in the screen does not establish an
			// authorization; only an authorization change bound to an authorization
			// subject can.
			if ins.ScreenOnly && p.Category != "presentation" {
				findings = append(findings, Finding{Code: "screen-disable-does-not-grant-authorization", Subject: ins.ID})
			}
		}
		proposals = append(proposals, p)
		switch p.Category {
		case "presentation":
			appearance = append(appearance, ins.ID)
		case "business-rule":
			business = append(business, ins.ID)
		case "authorization":
			authz = append(authz, ins.ID)
		default:
			unresolved = append(unresolved, ins.ID)
		}
	}

	sort.Slice(proposals, func(i, j int) bool { return proposals[i].Instruction < proposals[j].Instruction })
	sort.Strings(appearance)
	sort.Strings(business)
	sort.Strings(authz)
	sort.Strings(unresolved)
	sortFindings(findings)

	return Report{
		Schema:             Schema,
		Proposals:          proposals,
		AppearanceOnly:     appearance,
		BusinessRule:       business,
		Authorization:      authz,
		Unresolved:         unresolved,
		Applied:            []Applied{},
		SourceChanged:      false,
		MeaningFingerprint: meaningFingerprint(nil),
		Findings:           findings,
	}
}

// Accept applies only the accepted, resolved, supported instructions. An unaccepted
// change leaves the Source unchanged: SourceChanged stays false and the meaning
// fingerprint does not move. An appearance-only change never moves the fingerprint.
func Accept(in Input) Report {
	r := Plan(in)
	accepted := map[string]bool{}
	for _, id := range in.Accepted {
		accepted[id] = true
	}

	applied := []Applied{}
	findings := append([]Finding(nil), r.Findings...)
	for _, p := range r.Proposals {
		if !accepted[p.Instruction] {
			continue
		}
		switch {
		case p.Category == "unresolved":
			findings = append(findings, Finding{Code: "refused-accept:unresolved-mapping", Subject: p.Instruction})
			continue
		case p.Support == "unsupported":
			findings = append(findings, Finding{Code: "refused-accept:unsupported-part", Subject: p.Instruction})
			continue
		case p.ScreenOnly && p.Category != "presentation":
			findings = append(findings, Finding{Code: "refused-accept:screen-only-not-authorization", Subject: p.Instruction})
			continue
		}
		for _, s := range p.Subjects {
			applied = append(applied, Applied{Instruction: p.Instruction, Element: p.Element, Category: p.Category, Subject: s})
		}
		if p.Category != "presentation" {
			r.SourceChanged = true
		}
	}

	sort.Slice(applied, func(i, j int) bool {
		if applied[i].Instruction != applied[j].Instruction {
			return applied[i].Instruction < applied[j].Instruction
		}
		return applied[i].Subject < applied[j].Subject
	})
	sortFindings(findings)

	r.Applied = applied
	r.MeaningFingerprint = meaningFingerprint(applied)
	r.Findings = findings
	return r
}

// meaningFingerprint hashes only the meaning-affecting changes (business rule and
// authorization). An appearance-only change therefore leaves it unchanged.
func meaningFingerprint(applied []Applied) string {
	pairs := []string{}
	for _, a := range applied {
		if a.Category == "business-rule" || a.Category == "authorization" {
			pairs = append(pairs, a.Category+":"+a.Subject)
		}
	}
	sort.Strings(pairs)
	sum := sha256.Sum256([]byte(strings.Join(pairs, "|")))
	return fmt.Sprintf("sha256:%x", sum)
}

// Fingerprint is exposed for callers that want to compare a plan and an acceptance.
func Fingerprint(r Report) string {
	raw, err := json.Marshal(struct {
		Schema        string    `json:"schema"`
		Applied       []Applied `json:"applied"`
		SourceChanged bool      `json:"sourceChanged"`
	}{r.Schema, r.Applied, r.SourceChanged})
	if err != nil {
		return ""
	}
	sum := sha256.Sum256(raw)
	return fmt.Sprintf("sha256:%x", sum)
}

func sortFindings(findings []Finding) {
	sort.Slice(findings, func(i, j int) bool {
		if findings[i].Code != findings[j].Code {
			return findings[i].Code < findings[j].Code
		}
		return findings[i].Subject < findings[j].Subject
	})
}
