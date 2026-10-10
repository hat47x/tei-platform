// Package crossview provides a non-authoritative, deterministic projection of
// explicitly supplied cross-view checks. It does not infer business meaning,
// resolve source authority, or grant permission to accept a design change.
package crossview

import (
 "sort"
 "strings"
)

type Status string

const (
 Applicable Status = "applicable"
 Incompatible Status = "incompatible"
 Unresolved Status = "unresolved"
 Stale Status = "stale"
)

type Check struct {
 Subject string `json:"subject"`
 Relation string `json:"relation"`
 Owner string `json:"owner"`
 Evidence string `json:"evidence,omitempty"`
 Status Status `json:"status"`
 Required bool `json:"required"`
}

type Result struct {
 Outcome string `json:"outcome"`
 Checks []Check `json:"checks"`
}

// Evaluate sorts explicit checks without interpreting their semantic direction.
// A missing owner, evidence or known status fails closed on required checks.
// The caller is responsible for verifying evidence freshness against source
// revisions before marking a check Applicable.
func Evaluate(input []Check) Result {
 checks := append([]Check(nil), input...)
 sort.SliceStable(checks, func(i, j int) bool {
  a, b := checks[i], checks[j]
  ak := a.Subject+"\x00"+a.Relation+"\x00"+a.Owner+"\x00"+a.Evidence+"\x00"+string(a.Status)
  bk := b.Subject+"\x00"+b.Relation+"\x00"+b.Owner+"\x00"+b.Evidence+"\x00"+string(b.Status)
  return ak < bk
 })
 outcome := "eligible-for-proposal"
 rank := 0
 for _, check := range checks {
  if !check.Required { continue }
  status := check.Status
  if strings.TrimSpace(check.Subject)=="" || strings.TrimSpace(check.Relation)=="" || strings.TrimSpace(check.Owner)=="" || strings.TrimSpace(check.Evidence)=="" {
   status = Unresolved
  }
  next := 0
  switch status {
  case Applicable:
  case Incompatible: next = 2
  case Stale: next = 3
  default: next = 1
  }
  if next > rank { rank = next }
 }
 switch rank {
 case 1: outcome = "blocked-unresolved"
 case 2: outcome = "blocked-incompatible"
 case 3: outcome = "blocked-stale"
 }
 return Result{Outcome:outcome, Checks:checks}
}
