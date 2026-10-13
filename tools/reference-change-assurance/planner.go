// Package main is a synthetic, read-only change-scoped evidence reuse reference.
// It does not establish the authority of the input evidence or permit execution.
package main

import (
	"fmt"
	"sort"
	"strings"
)

type Fixture struct {
	Cases []Case `json:"cases"`
}

type Case struct {
	ID      string   `json:"id"`
	Purpose string   `json:"purpose"`
	Changes []Change `json:"changes"`
	Claims  []Claim  `json:"claims"`
}

type Change struct {
	Subject string   `json:"subject"`
	Aspects []string `json:"aspects"`
}

type Claim struct {
	ID             string            `json:"id"`
	Coverage       string            `json:"coverage"`
	Dependencies   []string          `json:"dependencies"`
	CoveredAspects []string          `json:"coveredAspects"`
	ImpactAspects  []string          `json:"impactAspects"`
	CurrentContext map[string]string `json:"currentContext"`
	Receipts       []Receipt         `json:"receipts"`
	RuntimeChecks  []string          `json:"runtimeChecks,omitempty"`
}

type Receipt struct {
	ID           string            `json:"id"`
	Outcome      string            `json:"outcome"`
	OracleOrigin string            `json:"oracleOrigin"`
	Context      map[string]string `json:"context"`
}

type Decision struct {
	CaseID           string   `json:"caseId"`
	Purpose          string   `json:"purpose"`
	ClaimID          string   `json:"claimId"`
	Action           string   `json:"action"`
	ReasonCodes      []string `json:"reasonCodes"`
	ReceiptRef       string   `json:"receiptRef,omitempty"`
	RuntimeChecks    []string `json:"runtimeChecks"`
	ExecutionAllowed bool     `json:"executionAllowed"`
}

type Report struct {
	SchemaVersion string     `json:"schemaVersion"`
	Authority     string     `json:"authority"`
	Decisions     []Decision `json:"decisions"`
}

func validText(s string) bool { return s != "" && strings.TrimSpace(s) == s }

func uniqueStrings(v []string) bool {
	seen := map[string]bool{}
	for _, s := range v {
		if !validText(s) || seen[s] {
			return false
		}
		seen[s] = true
	}
	return true
}

func subset(xs, ys []string) bool {
	m := make(map[string]bool, len(ys))
	for _, y := range ys {
		m[y] = true
	}
	for _, x := range xs {
		if !m[x] {
			return false
		}
	}
	return true
}

func validContext(m map[string]string) bool {
	if len(m) == 0 {
		return false
	}
	for k, v := range m {
		if !validText(k) || !validText(v) {
			return false
		}
	}
	return true
}

func validate(f Fixture) error {
	if len(f.Cases) == 0 {
		return fmt.Errorf("at least one case required")
	}
	ids := map[string]bool{}
	for _, c := range f.Cases {
		if !validText(c.ID) || ids[c.ID] || !validText(c.Purpose) {
			return fmt.Errorf("case id or purpose invalid/duplicated: %q", c.ID)
		}
		ids[c.ID] = true
		claimIDs := map[string]bool{}
		for _, change := range c.Changes {
			if !validText(change.Subject) || len(change.Aspects) == 0 || !uniqueStrings(change.Aspects) {
				return fmt.Errorf("case %s has invalid change", c.ID)
			}
		}
		if len(c.Claims) == 0 {
			return fmt.Errorf("case %s has no claims", c.ID)
		}
		for _, claim := range c.Claims {
			if !validText(claim.ID) || claimIDs[claim.ID] {
				return fmt.Errorf("case %s has invalid/duplicate claim", c.ID)
			}
			claimIDs[claim.ID] = true
			if claim.Coverage != "complete" && claim.Coverage != "opaque" {
				return fmt.Errorf("case %s claim %s coverage invalid", c.ID, claim.ID)
			}
			if len(claim.Dependencies) == 0 || !uniqueStrings(claim.Dependencies) || len(claim.CoveredAspects) == 0 || !uniqueStrings(claim.CoveredAspects) || !uniqueStrings(claim.ImpactAspects) || !subset(claim.ImpactAspects, claim.CoveredAspects) || !uniqueStrings(claim.RuntimeChecks) || !validContext(claim.CurrentContext) {
				return fmt.Errorf("case %s claim %s contract invalid", c.ID, claim.ID)
			}
			receiptIDs := map[string]bool{}
			for _, receipt := range claim.Receipts {
				if !validText(receipt.ID) || receiptIDs[receipt.ID] || (receipt.Outcome != "pass" && receipt.Outcome != "fail") || (receipt.OracleOrigin != "independent" && receipt.OracleOrigin != "implementation-derived" && receipt.OracleOrigin != "unknown") || !validContext(receipt.Context) {
					return fmt.Errorf("case %s claim %s receipt invalid", c.ID, claim.ID)
				}
				receiptIDs[receipt.ID] = true
			}
		}
	}
	return nil
}

func contextEqual(a, b map[string]string) bool {
	if len(a) != len(b) {
		return false
	}
	for k, av := range a {
		if bv, ok := b[k]; !ok || bv != av {
			return false
		}
	}
	return true
}

func contains(list []string, x string) bool {
	for _, v := range list {
		if v == x {
			return true
		}
	}
	return false
}

// assess recommends only whether a historical fixture receipt is a candidate
// for reuse. A candidate is never an authorization, PASS, or release decision.
func assess(caseID, purpose string, claim Claim, changes []Change) Decision {
	d := Decision{CaseID: caseID, Purpose: purpose, ClaimID: claim.ID, RuntimeChecks: append([]string{}, claim.RuntimeChecks...), ExecutionAllowed: false}
	sort.Strings(d.RuntimeChecks)
	if claim.Coverage != "complete" {
		d.Action, d.ReasonCodes = "unresolved", []string{"dependency-closure-opaque"}
		return d
	}
	var affected, unknown bool
	for _, change := range changes {
		if !contains(claim.Dependencies, change.Subject) {
			continue
		}
		for _, aspect := range change.Aspects {
			if !contains(claim.CoveredAspects, aspect) {
				unknown = true
			}
			if contains(claim.ImpactAspects, aspect) {
				affected = true
			}
		}
	}
	if unknown {
		d.Action, d.ReasonCodes = "unresolved", []string{"changed-aspect-not-covered"}
		return d
	}
	if affected {
		d.Action, d.ReasonCodes = "recheck", []string{"relevant-change"}
		return d
	}
	for _, receipt := range claim.Receipts {
		if receipt.Outcome == "pass" && receipt.OracleOrigin == "independent" && contextEqual(receipt.Context, claim.CurrentContext) {
			d.Action, d.ReasonCodes, d.ReceiptRef = "reuse-candidate", []string{"closure-complete", "no-relevant-change", "receipt-context-matched"}, receipt.ID
			return d
		}
	}
	d.Action, d.ReasonCodes = "recheck", []string{"no-matching-independent-passing-receipt"}
	return d
}

func evaluate(f Fixture) (Report, error) {
	if err := validate(f); err != nil {
		return Report{}, err
	}
	out := Report{SchemaVersion: "tei.reference.change-assurance/v0", Authority: "synthetic-fixture-only-no-execution", Decisions: []Decision{}}
	for _, c := range f.Cases {
		for _, claim := range c.Claims {
			out.Decisions = append(out.Decisions, assess(c.ID, c.Purpose, claim, c.Changes))
		}
	}
	return out, nil
}
