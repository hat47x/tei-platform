package operations

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"tei/internal/core"
	"tei/internal/semantic"
	"tei/internal/semantic/validationeval"
	"tei/internal/semantic/validationgen"
	"tei/internal/sourceindex"
	"tei/internal/validation"
	"tei/internal/workspace"
)

const (
	tableDateSource = "semantic/organization/transfer-request.tei.md"
	tableDateRule   = "organization.transfer_request.period_in_order"
	tableDateStart  = "organization.transfer_request.effective_from"
	tableDateEnd    = "organization.transfer_request.effective_to"
)

func loadTableWithDateRule(t *testing.T) (string, []byte) {
	t.Helper()
	base := filepath.Join("..", "workspace", "testdata", "tei-md-data-table")
	root := t.TempDir()
	manifest, err := os.ReadFile(filepath.Join(base, "tei.workspace.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "tei.workspace.yaml"), manifest, 0o644); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filepath.Join(base, filepath.FromSlash(tableDateSource)))
	if err != nil {
		t.Fatal(err)
	}
	for _, previous := range []string{"| effectiveFrom | std.datetime |", "| effectiveTo | std.datetime |"} {
		if bytes.Count(raw, []byte(previous)) != 1 {
			t.Fatalf("fixture changed: %q", previous)
		}
		raw = bytes.Replace(raw, []byte(previous), []byte(strings.Replace(previous, "std.datetime", "std.date", 1)), 1)
	}
	closing := "  organization.transfer_request.allocation is positive\n" + "```\n"
	if bytes.Count(raw, []byte(closing)) != 1 {
		t.Fatal("marked Rule code fence has changed")
	}
	added := "  organization.transfer_request.allocation is positive\n\n" +
		"rule organization.transfer_request.period_in_order:\n" +
		"  organization.transfer_request.effective_from is not after organization.transfer_request.effective_to if present\n" + "```\n"
	raw = bytes.Replace(raw, []byte(closing), []byte(added), 1)
	dest := filepath.Join(root, filepath.FromSlash(tableDateSource))
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dest, raw, 0o644); err != nil {
		t.Fatal(err)
	}
	return root, raw
}

type dateTableSourceReader struct {
	values map[core.CanonicalID]validationeval.DateReading
	reads  []core.CanonicalID
}

func (s *dateTableSourceReader) ReadDate(id core.CanonicalID) (validationeval.DateReading, error) {
	s.reads = append(s.reads, id)
	v, ok := s.values[id]
	if !ok {
		return validationeval.DateReading{}, errors.New("undeclared date read")
	}
	return v, nil
}
func (s *dateTableSourceReader) ReadNumber(core.CanonicalID) (validationeval.NumberReading, error) {
	return validationeval.NumberReading{}, errors.New("wrong numeric read")
}
func (s *dateTableSourceReader) ReadRelationKeys(core.CanonicalID, core.CanonicalID) (validationeval.RelationReading, error) {
	return validationeval.RelationReading{}, errors.New("wrong relation read")
}

func datedCell(t *testing.T, v string) validationeval.DateReading {
	t.Helper()
	switch v {
	case "NULL":
		return validationeval.DateReading{State: validationeval.ValueNull}
	case "UNREAD":
		return validationeval.DateReading{State: validationeval.ValueNotFetched}
	}
	day, err := semantic.ParseCalendarDate(v)
	if err != nil {
		t.Fatal(err)
	}
	return validationeval.DateReading{State: validationeval.ValuePresent, Value: day}
}

func TestMarkedDataTableNotAfterUsesPButExplicitlyRejectsUnsupportedG(t *testing.T) {
	root, raw := loadTableWithDateRule(t)
	loaded, err := workspace.Load(root)
	if err != nil {
		t.Fatalf("selected Reader cannot load Rule: %v", err)
	}
	if len(loaded.AuthoringSources) != 1 ||
		loaded.AuthoringSources[0].Language.LanguageID != "tei.authoring" ||
		loaded.AuthoringSources[0].Language.ContractVersion != "v1" {
		t.Fatalf("incorrect Reader: %+v", loaded.AuthoringSources)
	}
	if findings := validation.Validate(loaded.World); len(findings) != 0 {
		t.Fatalf("date Rule and two date Attributes did not validate: %+v", findings)
	}
	index, err := workspace.BuildSourceIndexFromLoad(root, loaded)
	if err != nil {
		t.Fatal(err)
	}
	subject := sourceindex.Subject{Kind: "semantic-validation", ID: tableDateRule}
	view, err := InspectionFromLoad(loaded, index, subject)
	if err != nil {
		t.Fatal(err)
	}
	if err := ValidateInspectionProjection(subject, view); err != nil {
		t.Fatal(err)
	}
	if view.Effect != "read-only" || len(view.Sources.References) != 1 ||
		len(view.Sources.Artifacts) != 1 ||
		view.Sources.References[0].Role != sourceindex.RoleDefinition ||
		view.Sources.References[0].ArtifactPath != tableDateSource ||
		view.Sources.Artifacts[0].Fingerprint != sourceindex.Fingerprint(raw) {
		t.Fatalf("date Rule lost its Source: %+v", view)
	}
	if err := view.Sources.References[0].Range.Validate(raw); err != nil {
		t.Fatal(err)
	}
	if view.Sources.References[0].Range.Start.Offset != bytes.Index(raw, []byte(tableDateRule)) {
		t.Fatal("date Rule Source anchor moved to a different definition")
	}
	var rule semantic.Validation
	if err := json.Unmarshal(view.Canonical, &rule); err != nil {
		t.Fatal(err)
	}
	if rule.Kind != semantic.ValidationNotAfter || rule.Attribute != tableDateStart ||
		rule.Compared != tableDateEnd || !rule.IfPresent {
		t.Fatalf("not-after Rule lost its meaning: %+v", rule)
	}
	if generated, err := validationgen.GeneratePositive(rule, "std.date", "main"); err == nil ||
		!strings.Contains(err.Error(), "unsupported generated kind") || len(generated) != 0 {
		t.Fatalf("unsupported G silently lowered date Rule: %q %v", generated, err)
	}
	positiveView, err := InspectionFromLoad(loaded, index, sourceindex.Subject{
		Kind: "semantic-validation", ID: "organization.transfer_request.allocation_positive",
	})
	if err != nil {
		t.Fatal(err)
	}
	var positive semantic.Validation
	if err := json.Unmarshal(positiveView.Canonical, &positive); err != nil {
		t.Fatal(err)
	}
	if goSource, err := validationgen.GeneratePositive(positive, "std.decimal", "main"); err != nil ||
		!bytes.Contains(goSource, []byte("value.Sign() > 0")) {
		t.Fatalf("a supported sibling Rule unexpectedly lost G: %v %s", err, goSource)
	}
	for _, tc := range []struct {
		name, start, end string
		want             validationeval.Result
		reason           validationeval.UndecidedReason
		about            core.CanonicalID
	}{
		{"before", "2026-01-01", "2026-03-01", validationeval.ResultSatisfied, "", ""},
		{"equal", "2026-03-01", "2026-03-01", validationeval.ResultSatisfied, "", ""},
		{"after", "2026-04-01", "2026-03-01", validationeval.ResultNotSatisfied, "", ""},
		{"open", "2026-01-01", "NULL", validationeval.ResultSatisfied, "", ""},
		{"unread-start-open-end", "UNREAD", "NULL", validationeval.ResultSatisfied, "", ""},
		{"unread-end", "2026-01-01", "UNREAD", "", validationeval.UndecidedValueNotFetched, tableDateEnd},
		{"null-start", "NULL", "2026-03-01", "", validationeval.UndecidedValueNull, tableDateStart},
	} {
		t.Run(tc.name, func(t *testing.T) {
			source := &dateTableSourceReader{values: map[core.CanonicalID]validationeval.DateReading{
				tableDateStart: datedCell(t, tc.start), tableDateEnd: datedCell(t, tc.end),
			}}
			got, err := validationeval.Decide(rule, source, validationeval.Operators{})
			if err != nil || got.Result != tc.want || len(source.reads) != 2 ||
				source.reads[0] != tableDateStart || source.reads[1] != tableDateEnd {
				t.Fatalf("P decision or Read changed: %+v error=%v reads=%v", got, err, source.reads)
			}
			if tc.reason == "" && got.Undecided != nil {
				t.Fatalf("expected a decided answer: %+v", got)
			}
			if tc.reason != "" && (got.Undecided == nil ||
				got.Undecided.Reason != tc.reason || got.Undecided.Attribute != tc.about) {
				t.Fatalf("wrong undecided reason: %+v", got)
			}
		})
	}
}

func TestMarkedDataTableNotAfterRejectsDatetimeAttribute(t *testing.T) {
	root, raw := loadTableWithDateRule(t)
	before, after := []byte("| effectiveFrom | std.date |"), []byte("| effectiveFrom | std.datetime |")
	if bytes.Count(raw, before) != 1 {
		t.Fatal("fixture date column changed")
	}
	dest := filepath.Join(root, filepath.FromSlash(tableDateSource))
	if err := os.WriteFile(dest, bytes.Replace(raw, before, after, 1), 0o644); err != nil {
		t.Fatal(err)
	}
	loaded, err := workspace.Load(root)
	if err != nil {
		t.Fatalf("datetime must parse before Validation rejects it: %v", err)
	}
	findings := validation.Validate(loaded.World)
	if len(findings) != 1 || findings[0].Code != "SEMANTIC_VALIDATION_NOT_AFTER_REQUIRES_DATE_ATTRIBUTES" {
		t.Fatalf("not-after accepted std.datetime or changed rejection: %+v", findings)
	}
}
