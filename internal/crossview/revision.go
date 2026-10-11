package crossview

import "strings"

// SourceRevision is a caller-supplied, content-addressed authority boundary.
// Digest must be computed by the source owner over the full authoritative
// source set, including membership and root configuration. This type does not
// calculate or independently authenticate the digest.
type SourceRevision struct {
 Scope string `json:"scope"`
 Digest string `json:"digest"`
}

// EvidenceBase identifies the model/source bases against which checks ran.
// World and Business are independent revision domains. Business may be
// absent only when the caller explicitly has no Business requirements.
type EvidenceBase struct {
 World SourceRevision `json:"world"`
 Business *SourceRevision `json:"business,omitempty"`
}

// VerifyBase rejects incomplete and changed evidence bases. Comparing these
// declarations is necessary but insufficient: the caller must independently
// recompute current digests from authoritative sources.
func VerifyBase(evidence, current EvidenceBase, requireBusiness bool) Status {
 if !validRevision(evidence.World) || !validRevision(current.World) {
  return Unresolved
 }
 if requireBusiness && (evidence.Business == nil || current.Business == nil) {
  return Unresolved
 }
 if evidence.Business != nil && !validRevision(*evidence.Business) {
  return Unresolved
 }
 if current.Business != nil && !validRevision(*current.Business) {
  return Unresolved
 }
 if evidence.World != current.World {
  return Stale
 }
 if (evidence.Business == nil) != (current.Business == nil) {
  return Stale
 }
 if evidence.Business != nil && *evidence.Business != *current.Business {
  return Stale
 }
 return Applicable
}

func validRevision(r SourceRevision) bool {
 return strings.TrimSpace(r.Scope)!="" && strings.TrimSpace(r.Digest)!=""
}

// EvaluateWithBase is an evidence-only projection. It cannot assert that the
// required inventory is complete or that source digests were recomputed.
func EvaluateWithBase(required []Requirement, checks []Check, evidence, current EvidenceBase, requireBusiness bool) Result {
 result:=EvaluateAgainstRequirements(required,checks)
 status:=VerifyBase(evidence,current,requireBusiness)
 switch status {
 case Stale:
  result.Outcome="blocked-stale"
 case Unresolved:
  if result.Outcome=="eligible-for-proposal" {
   result.Outcome="blocked-unresolved"
  }
 }
 return result
}
