package crossview

import (
 "tei/internal/validation"
)

// ValidationFindings converts authoritative validation errors into negative
// cross-view evidence. Absence of diagnostics is NOT positive proof: this
// adapter deliberately never emits Applicable. A caller must separately
// establish source freshness, check coverage, and positive evidence.
func ValidationFindings(diagnostics []validation.Diagnostic, evidence string) []Check {
 out:=make([]Check,0,len(diagnostics))
 for _, d:=range diagnostics {
  if d.Severity!=validation.Error { continue }
  target:="world"
  if d.Subject!=nil {
   target=d.Subject.Kind+":"+d.Subject.ID
  }
  out=append(out,Check{
   Subject:target,
   Relation:"validation-diagnostic:"+d.Code,
   Owner:"validation",
   Evidence:evidence,
   Status:Incompatible,
   Required:true,
  })
 }
 return Evaluate(out).Checks
}
