package verification

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"
)

// DiagnosticLevel is the severity of a diagnostic message.
type DiagnosticLevel string

const (
	// DiagnosticError indicates a definite mismatch.
	DiagnosticError DiagnosticLevel = "error"
	// DiagnosticWarning indicates a suspicious condition that may not be a mismatch.
	DiagnosticWarning DiagnosticLevel = "warning"
	// DiagnosticInfo indicates an informational observation.
	DiagnosticInfo DiagnosticLevel = "info"
)

// DiagnosticRecord is one diagnostic entry produced during verification.
type DiagnosticRecord struct {
	Level   DiagnosticLevel `json:"level"`
	Code    string          `json:"code"`
	Message string          `json:"message"`
	// Context carries structured data about the difference.
	Context map[string]string `json:"context,omitempty"`
}

// DiagnosticReport is a collection of diagnostic records.
type DiagnosticReport struct {
	SchemaVersion string             `json:"schemaVersion"`
	ScenarioID    string             `json:"scenarioId"`
	OK            bool               `json:"ok"`
	Records       []DiagnosticRecord `json:"records"`
	Summary       string             `json:"summary"`
}

// NewDiagnosticReport creates a new diagnostic report.
func NewDiagnosticReport(scenarioID string) DiagnosticReport {
	return DiagnosticReport{
		SchemaVersion: "tei.verification.diagnostics/v0",
		ScenarioID:    scenarioID,
		OK:            true,
		Records:       []DiagnosticRecord{},
	}
}

// AddError appends an error-level diagnostic.
func (r *DiagnosticReport) AddError(code, message string, context map[string]string) {
	r.Records = append(r.Records, DiagnosticRecord{
		Level:   DiagnosticError,
		Code:    code,
		Message: message,
		Context: context,
	})
	r.OK = false
}

// AddWarning appends a warning-level diagnostic.
func (r *DiagnosticReport) AddWarning(code, message string, context map[string]string) {
	r.Records = append(r.Records, DiagnosticRecord{
		Level:   DiagnosticWarning,
		Code:    code,
		Message: message,
		Context: context,
	})
}

// AddInfo appends an info-level diagnostic.
func (r *DiagnosticReport) AddInfo(code, message string, context map[string]string) {
	r.Records = append(r.Records, DiagnosticRecord{
		Level:   DiagnosticInfo,
		Code:    code,
		Message: message,
		Context: context,
	})
}

// BuildSummary computes a one-line summary from the records.
func (r *DiagnosticReport) BuildSummary() {
	errors := 0
	warnings := 0
	for _, rec := range r.Records {
		switch rec.Level {
		case DiagnosticError:
			errors++
		case DiagnosticWarning:
			warnings++
		}
	}
	parts := []string{fmt.Sprintf("%d diagnostic records", len(r.Records))}
	if errors > 0 {
		parts = append(parts, fmt.Sprintf("%d errors", errors))
	}
	if warnings > 0 {
		parts = append(parts, fmt.Sprintf("%d warnings", warnings))
	}
	if errors == 0 && warnings == 0 {
		parts = append(parts, "no issues")
	}
	r.Summary = strings.Join(parts, ", ")
}

// WriteJSON writes the diagnostic report as JSON to w.
func (r *DiagnosticReport) WriteJSON(w io.Writer) error {
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	return enc.Encode(r)
}

// WriteHuman writes a human-readable diagnostic report to w.
func (r *DiagnosticReport) WriteHuman(w io.Writer) error {
	if _, err := fmt.Fprintf(w, "Diagnostic Report: %s\n", r.ScenarioID); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "Status: %s\n", statusString(r.OK)); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "Summary: %s\n", r.Summary); err != nil {
		return err
	}

	sort.Slice(r.Records, func(i, j int) bool {
		if r.Records[i].Level != r.Records[j].Level {
			return levelOrder(r.Records[i].Level) < levelOrder(r.Records[j].Level)
		}
		return r.Records[i].Code < r.Records[j].Code
	})

	for _, rec := range r.Records {
		prefix := strings.ToUpper(string(rec.Level))
		if _, err := fmt.Fprintf(w, "[%s] %s: %s\n", prefix, rec.Code, rec.Message); err != nil {
			return err
		}
		if len(rec.Context) > 0 {
			keys := make([]string, 0, len(rec.Context))
			for k := range rec.Context {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			for _, k := range keys {
				if _, err := fmt.Fprintf(w, "    %s = %s\n", k, rec.Context[k]); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

// BuildDiagnosticsFromReport converts a VerificationReport into a DiagnosticReport.
func BuildDiagnosticsFromReport(report VerificationReport) DiagnosticReport {
	diag := NewDiagnosticReport(report.ScenarioID)
	diag.AddInfo("VERIFICATION_START", fmt.Sprintf("scenario=%s mode=%s", report.ScenarioID, report.ComparisonMode), nil)
	diag.AddInfo("FULL_RUN", fmt.Sprintf("label=%s ok=%v durationMs=%d", report.FullRun.Label, report.FullRun.OK, report.FullRun.DurationMs), map[string]string{
		"worldFingerprint":       report.FullRun.WorldFingerprint,
		"sourceIndexFingerprint": report.FullRun.SourceIndexFingerprint,
	})
	diag.AddInfo("INCREMENTAL_RUN", fmt.Sprintf("label=%s ok=%v durationMs=%d skippedSteps=%d",
		report.IncrementalRun.Label,
		report.IncrementalRun.OK,
		report.IncrementalRun.DurationMs,
		len(report.IncrementalRun.SkippedSteps),
	), map[string]string{
		"worldFingerprint":       report.IncrementalRun.WorldFingerprint,
		"sourceIndexFingerprint": report.IncrementalRun.SourceIndexFingerprint,
	})

	for _, skip := range report.IncrementalRun.SkippedSteps {
		diag.AddInfo("CACHE_SKIP", fmt.Sprintf("skipped step: %s", skip), map[string]string{
			"run": report.IncrementalRun.Label,
		})
	}

	for _, d := range report.Diffs {
		level := DiagnosticError
		if d.Severity == "warning" {
			level = DiagnosticWarning
		}
		context := map[string]string{
			"expected": d.Expected,
			"actual":   d.Actual,
		}
		if d.ArtifactPath != "" {
			context["artifactPath"] = d.ArtifactPath
		}
		rec := DiagnosticRecord{
			Level:   level,
			Code:    string(d.Kind),
			Message: d.Message,
			Context: context,
		}
		switch level {
		case DiagnosticError:
			diag.Records = append(diag.Records, rec)
			diag.OK = false
		case DiagnosticWarning:
			diag.Records = append(diag.Records, rec)
		}
	}

	if len(report.CounterexampleIDs) > 0 {
		diag.AddInfo("COUNTEREXAMPLES", fmt.Sprintf("exercised: %s", strings.Join(report.CounterexampleIDs, ", ")), nil)
	}
	for _, note := range report.Notes {
		diag.AddInfo("NOTE", note, nil)
	}

	diag.BuildSummary()
	return diag
}

// DiffDetail returns a structured description of a single diff for use in
// diagnostic output.
func DiffDetail(d DiffEntry) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("[%s] %s\n", d.Kind, d.Severity))
	sb.WriteString(fmt.Sprintf("  message : %s\n", d.Message))
	if d.ArtifactPath != "" {
		sb.WriteString(fmt.Sprintf("  artifact : %s\n", d.ArtifactPath))
	}
	sb.WriteString(fmt.Sprintf("  expected : %s\n", d.Expected))
	sb.WriteString(fmt.Sprintf("  actual   : %s\n", d.Actual))
	return sb.String()
}

// WriteDiffReport writes a human-readable diff report to w.
func WriteDiffReport(w io.Writer, report VerificationReport) error {
	if report.OK {
		_, err := fmt.Fprintf(w, "VERIFICATION PASS: %s\n", report.ScenarioID)
		if err != nil {
			return err
		}
		_, err = fmt.Fprintf(w, "  full=%s incremental=%s mode=%s\n",
			report.FullRun.Label, report.IncrementalRun.Label, report.ComparisonMode)
		return err
	}

	_, err := fmt.Fprintf(w, "VERIFICATION FAIL: %s (%s)\n", report.ScenarioID, report.ScenarioName)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(w, "  full=%s ok=%v\n", report.FullRun.Label, report.FullRun.OK)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(w, "  incremental=%s ok=%v\n", report.IncrementalRun.Label, report.IncrementalRun.OK)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(w, "  diffs=%d mode=%s\n", len(report.Diffs), report.ComparisonMode)
	if err != nil {
		return err
	}

	for _, d := range report.Diffs {
		if _, err := fmt.Fprintf(w, "\n%s\n", DiffDetail(d)); err != nil {
			return err
		}
	}

	if len(report.CounterexampleIDs) > 0 {
		if _, err := fmt.Fprintf(w, "\nCounterexamples exercised: %s\n", strings.Join(report.CounterexampleIDs, ", ")); err != nil {
			return err
		}
	}
	return nil
}

// DiagnosticSummary returns a short summary string.
func (r *DiagnosticReport) DiagnosticSummary() string {
	parts := []string{fmt.Sprintf("scenario=%s", r.ScenarioID)}
	if r.OK {
		parts = append(parts, "status=PASS")
	} else {
		parts = append(parts, fmt.Sprintf("status=FAIL errors=%d", countLevel(r.Records, DiagnosticError)))
	}
	parts = append(parts, fmt.Sprintf("records=%d", len(r.Records)))
	parts = append(parts, fmt.Sprintf("summary=%s", r.Summary))
	return strings.Join(parts, " ")
}

// WriteCIBadge writes a GitHub Actions compatible badge line to w.
func (r *DiagnosticReport) WriteCIBadge(w io.Writer) error {
	status := "pass"
	if !r.OK {
		status = "fail"
	}
	_, err := fmt.Fprintf(w, "::%s::Verification %s - %s (records=%d)\n",
		status, r.ScenarioID, r.Summary, len(r.Records))
	return err
}

func statusString(ok bool) string {
	if ok {
		return "PASS"
	}
	return "FAIL"
}

func levelOrder(level DiagnosticLevel) int {
	switch level {
	case DiagnosticError:
		return 0
	case DiagnosticWarning:
		return 1
	case DiagnosticInfo:
		return 2
	default:
		return 3
	}
}

func countLevel(records []DiagnosticRecord, level DiagnosticLevel) int {
	count := 0
	for _, r := range records {
		if r.Level == level {
			count++
		}
	}
	return count
}

// AssertEmpty is a test helper that fails t when the report contains errors or warnings.
func (r *DiagnosticReport) AssertEmpty(t interface {
	Helper()
	Fatalf(string, ...any)
}) {
	t.Helper()
	if !r.OK {
		t.Fatalf("diagnostic report failed: %s\nrecords:\n%s",
			r.DiagnosticSummary(), diagnosticRecordsText(r.Records))
	}
}

func diagnosticRecordsText(records []DiagnosticRecord) string {
	var sb strings.Builder
	for _, rec := range records {
		sb.WriteString(fmt.Sprintf("  [%s] %s: %s\n", rec.Level, rec.Code, rec.Message))
		for k, v := range rec.Context {
			sb.WriteString(fmt.Sprintf("    %s = %s\n", k, v))
		}
	}
	return sb.String()
}
