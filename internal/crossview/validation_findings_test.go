package crossview

import (
 "testing"
 "tei/internal/subject"
 "tei/internal/validation"
)

func TestValidationFindingsNegativeOnly(t *testing.T) {
 s:=subject.Subject{Kind:"application-pattern",ID:"customer_reference"}
 findings:=ValidationFindings([]validation.Diagnostic{
  {Code:"UNKNOWN_APPLICATION_PATTERN_RELATION",Severity:validation.Error,Subject:&s},
  {Code:"WARNING_ONLY",Severity:validation.Warning,Subject:&s},
 }, "world@rev1")
 if len(findings)!=1 {t.Fatalf("got %d findings, want 1: %#v",len(findings),findings)}
 got:=findings[0]
 if got.Status!=Incompatible || got.Subject!="application-pattern:customer_reference" || got.Relation!="validation-diagnostic:UNKNOWN_APPLICATION_PATTERN_RELATION" || !got.Required {
  t.Fatalf("unexpected finding: %#v",got)
 }
}

func TestValidationFindingsEmptyDoesNotAuthorize(t *testing.T) {
 findings:=ValidationFindings(nil,"world@rev1")
 if len(findings)!=0 {t.Fatalf("invented positive evidence: %#v",findings)}
 if got:=Evaluate(findings).Outcome; got!="blocked-unresolved" {t.Fatalf("outcome %q",got)}
}

func TestValidationFindingsMissingProvenanceBlocks(t *testing.T) {
 findings:=ValidationFindings([]validation.Diagnostic{{Code:"INVALID",Severity:validation.Error}},"")
 if got:=Evaluate(findings).Outcome; got!="blocked-unresolved" {t.Fatalf("outcome %q",got)}
}
