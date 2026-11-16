package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"tei/internal/operations"
	"tei/internal/sourceindex"
)

func runModelSearch(t *testing.T, args ...string) (modelSearchCLIResponse, int) {
	t.Helper()
	var out bytes.Buffer
	code, err := runModelCommand(append([]string{"search"}, args...), &out)
	if err != nil {
		t.Fatalf("search %v: err=%v body=%s", args, err, out.String())
	}
	var response modelSearchCLIResponse
	if err := json.Unmarshal(out.Bytes(), &response); err != nil {
		t.Fatalf("search %v: not JSON: %v (%s)", args, err, out.String())
	}
	return response, code
}

// `model search` reads the same Workspace load as `model catalog`, so a hit and
// the catalog describe one model version: a person reading a Source and an AI
// reading the hits can confirm they look at the same state.
func TestModelSearchAgreesWithCatalogOnTheReferenceWorkspace(t *testing.T) {
	root := filepath.Join("..", "..", "reference", "workspace")
	var catalogOut bytes.Buffer
	if code, err := runModelCommand([]string{"catalog", "-workspace", root}, &catalogOut); err != nil || code != 0 {
		t.Fatalf("catalog: code=%d err=%v body=%s", code, err, catalogOut.String())
	}
	var catalog modelCLIResponse
	if err := json.Unmarshal(catalogOut.Bytes(), &catalog); err != nil || catalog.Result == nil {
		t.Fatalf("catalog decode: %v %s", err, catalogOut.String())
	}

	response, code := runModelSearch(t, "-workspace", root, "-term", "namespace", "-limit", "500")
	if code != 0 || !response.OK || response.Result == nil {
		t.Fatalf("search: code=%d body=%+v", code, response)
	}
	got := response.Result
	if response.SchemaVersion != "tei.cli.model-search/v0" || got.Effect != "read-only" {
		t.Fatalf("envelope: %+v", response)
	}
	if got.WorldFingerprint != catalog.Result.WorldFingerprint ||
		got.SourceIndexFingerprint != catalog.Result.SourceIndexFingerprint ||
		got.ModelRevision != catalog.Result.ModelRevision {
		t.Fatal("search and catalog disagree on the model version")
	}
	if got.Total == 0 || len(got.Hits) == 0 {
		t.Fatalf("the reference workspace declares no namespace line: %+v", got)
	}

	// Every hit is re-checked against the file on disk: the path is
	// workspace-relative, the line exists, the text is that line, and the line
	// carries the term. The projection is not trusted for its own evidence.
	for _, hit := range got.Hits {
		if !operations.SupportedInspectionKind(hit.Subject.Kind) {
			t.Fatalf("hit kind %q is outside the closed set", hit.Subject.Kind)
		}
		if hit.ArtifactPath != filepath.ToSlash(filepath.Clean(hit.ArtifactPath)) ||
			strings.HasPrefix(hit.ArtifactPath, "/") || strings.Contains(hit.ArtifactPath, "..") {
			t.Fatalf("hit path %q is not workspace-relative", hit.ArtifactPath)
		}
		content, readErr := os.ReadFile(filepath.Join(root, filepath.FromSlash(hit.ArtifactPath)))
		if readErr != nil {
			t.Fatalf("hit points at %s, which cannot be read: %v", hit.ArtifactPath, readErr)
		}
		lines := strings.Split(strings.ReplaceAll(string(content), "\r\n", "\n"), "\n")
		if hit.Line < 1 || hit.Line > len(lines) {
			t.Fatalf("hit line %d is outside %s", hit.Line, hit.ArtifactPath)
		}
		if lines[hit.Line-1] != hit.Text {
			t.Fatalf("hit text %q is not line %d of %s (%q)",
				hit.Text, hit.Line, hit.ArtifactPath, lines[hit.Line-1])
		}
		if !strings.Contains(strings.ToLower(hit.Text), "namespace") {
			t.Fatalf("hit %+v does not carry the term", hit)
		}
	}
	// kindCounts counts hits, so its sum is total; the subject list names each
	// distinct subject once and its matchCounts add back up to the hit total.
	sum := 0
	for _, kc := range got.KindCounts {
		sum += kc.Count
	}
	if sum != got.Total {
		t.Fatalf("kindCounts sum %d != total %d", sum, got.Total)
	}
	if got.SubjectCount != len(got.Subjects) || got.SubjectCount > got.Total {
		t.Fatalf("subjectCount=%d subjects=%d total=%d",
			got.SubjectCount, len(got.Subjects), got.Total)
	}
	if got.DistinctSubjectsOmitted != 0 {
		t.Fatalf("distinctSubjectsOmitted=%d, want 0 when the page holds every match",
			got.DistinctSubjectsOmitted)
	}
	counted := 0
	for _, subject := range got.Subjects {
		counted += subject.MatchCount
	}
	if counted != got.Total {
		t.Fatalf("subject matchCounts sum %d != total %d", counted, got.Total)
	}
}

// distinctSearchSubjects counts the subjects a hit list names, the way the
// projection's kindCounts must: one subject is one count however many lines of
// its declaration matched.
func distinctSearchSubjects(hits []operations.ModelSearchHit) int {
	seen := map[string]bool{}
	for _, hit := range hits {
		seen[hit.Subject.Kind+"\x00"+hit.Subject.ID] = true
	}
	return len(seen)
}

// The limit and offset page one deterministic answer, and a kind filter
// narrows it to the kinds the caller named.
func TestModelSearchPagingAndKindFilter(t *testing.T) {
	root := filepath.Join("..", "..", "reference", "workspace")
	full, code := runModelSearch(t, "-workspace", root, "-term", "namespace", "-limit", "500")
	if code != 0 || full.Result == nil || full.Result.Total < 2 {
		t.Fatalf("full search: code=%d body=%+v", code, full.Result)
	}
	page, code := runModelSearch(t, "-workspace", root, "-term", "namespace", "-limit", "1")
	if code != 0 || page.Result == nil {
		t.Fatalf("first page: code=%d", code)
	}
	if len(page.Result.Hits) != 1 || page.Result.Omitted != page.Result.Total-1 ||
		page.Result.NextOffset == nil || *page.Result.NextOffset != 1 {
		t.Fatalf("first page: %+v", page.Result)
	}
	if page.Result.Hits[0] != full.Result.Hits[0] {
		t.Fatal("the first page is not a prefix of the full answer")
	}
	last, code := runModelSearch(t, "-workspace", root, "-term", "namespace", "-limit", "1",
		"-offset", "500")
	if code != 0 || last.Result == nil {
		t.Fatalf("clamped page: code=%d", code)
	}
	if len(last.Result.Hits) != 0 || last.Result.Offset != last.Result.Total ||
		last.Result.RequestedOffset != 500 || last.Result.NextOffset != nil {
		t.Fatalf("clamped page: total=%d offset=%d requested=%d hits=%d",
			last.Result.Total, last.Result.Offset, last.Result.RequestedOffset, len(last.Result.Hits))
	}

	filtered, code := runModelSearch(t, "-workspace", root, "-term", "namespace",
		"-kind", "semantic", "-limit", "500")
	if code != 0 || filtered.Result == nil {
		t.Fatalf("kind filter: code=%d body=%+v", code, filtered.Result)
	}
	if filtered.Result.Total == 0 || filtered.Result.Total > full.Result.Total {
		t.Fatalf("filtered total=%d, unfiltered=%d", filtered.Result.Total, full.Result.Total)
	}
	for _, hit := range filtered.Result.Hits {
		if hit.Subject.Kind != "semantic" {
			t.Fatalf("filtered hit kind=%q", hit.Subject.Kind)
		}
	}
	// Without a term the answer is the kind's own Sources, page by page.
	listed, code := runModelSearch(t, "-workspace", root, "-kind", "semantic", "-limit", "500")
	if code != 0 || listed.Result == nil {
		t.Fatalf("kind listing: code=%d body=%+v", code, listed.Result)
	}
	if listed.Result.Total < filtered.Result.Total {
		t.Fatalf("kind listing total=%d is smaller than the term search %d",
			listed.Result.Total, filtered.Result.Total)
	}
	for _, hit := range listed.Result.Hits {
		if hit.Subject.Kind != "semantic" {
			t.Fatalf("kind listing leaked %q", hit.Subject.Kind)
		}
	}
}

// The two count units answer different questions: a term one application writes
// on three lines of its declaration is three hits and one subject, so kindCounts
// -- a breakdown of the hits by kind -- reports the three hits while
// subjectCount reports the one subject.
func TestModelSearchSalesCoreIsThreeHitsAndOneSubject(t *testing.T) {
	root := filepath.Join("..", "..", "reference", "workspace")
	response, code := runModelSearch(t, "-workspace", root, "-term", "sales_core", "-limit", "500")
	if code != 0 || !response.OK || response.Result == nil {
		t.Fatalf("search: code=%d body=%+v", code, response)
	}
	got := response.Result
	if got.Total != 3 || len(got.Hits) != 3 {
		t.Fatalf("total=%d hits=%d, want the three matching lines", got.Total, len(got.Hits))
	}
	for _, hit := range got.Hits {
		if hit.Subject.Kind != "application" || hit.Subject.ID != "order_entry" {
			t.Fatalf("hit=%+v, want the one order_entry application", hit)
		}
	}
	// kindCounts counts hits, so the application count is the three lines.
	if len(got.KindCounts) != 1 || got.KindCounts[0].Kind != "application" ||
		got.KindCounts[0].Count != 3 {
		t.Fatalf("kindCounts=%+v, want three application hits", got.KindCounts)
	}
	// The subject list names that one application once, at its first matching
	// line, with all three matching lines counted, so an agent asking which
	// subjects declare the term reads one subject here.
	if len(got.Subjects) != 1 {
		t.Fatalf("subjects=%+v, want one entry", got.Subjects)
	}
	// subjectCount answers "how many subjects" as one number, so the three
	// matching lines are never read as three subjects and total is never read
	// as a subject count.
	if got.SubjectCount != 1 {
		t.Fatalf("subjectCount=%d, want the one order_entry application", got.SubjectCount)
	}
	wantSubject := operations.ModelSearchSubject{
		Subject:      sourceindex.Subject{Kind: "application", ID: "order_entry"},
		ArtifactPath: "applications/order-entry/application.tei",
		Line:         10,
		MatchCount:   3,
	}
	if got.Subjects[0] != wantSubject {
		t.Fatalf("subject=%+v want %+v", got.Subjects[0], wantSubject)
	}
}

// A page of one hit does not shrink the subject answer: subjectCount still
// reports the full count and distinctSubjectsOmitted reports how many subjects
// the page does not return. Every namespace line belongs to its own subject, so
// a page of the first namespace leaves the other subjects to the count.
func TestModelSearchSubjectCountSurvivesAPageOfOneHit(t *testing.T) {
	root := filepath.Join("..", "..", "reference", "workspace")
	full, code := runModelSearch(t, "-workspace", root, "-term", "namespace", "-limit", "500")
	if code != 0 || full.Result == nil {
		t.Fatalf("full search: code=%d body=%+v", code, full.Result)
	}
	if full.Result.SubjectCount < 2 {
		t.Fatalf("subjectCount=%d, want several namespace subjects", full.Result.SubjectCount)
	}
	page, code := runModelSearch(t, "-workspace", root, "-term", "namespace", "-limit", "1")
	if code != 0 || page.Result == nil {
		t.Fatalf("page search: code=%d body=%+v", code, page.Result)
	}
	got := page.Result
	if len(got.Hits) != 1 || got.Omitted != got.Total-1 {
		t.Fatalf("page: hits=%d omitted=%d", len(got.Hits), got.Omitted)
	}
	if got.SubjectCount != full.Result.SubjectCount || len(got.Subjects) != full.Result.SubjectCount {
		t.Fatalf("page subjectCount=%d subjects=%d, want the full count %d",
			got.SubjectCount, len(got.Subjects), full.Result.SubjectCount)
	}
	// The returned hit is the only match of its subject, so the omitted matches
	// name every other subject and the count is subjectCount-1.
	if got.DistinctSubjectsOmitted != got.SubjectCount-1 {
		t.Fatalf("distinctSubjectsOmitted=%d, want subjectCount-1 %d",
			got.DistinctSubjectsOmitted, got.SubjectCount-1)
	}
}

// The JSON itself carries both subject counts, so a caller that reads the
// envelope without this package's types still sees them.
func TestModelSearchJSONCarriesBothSubjectCounts(t *testing.T) {
	root := filepath.Join("..", "..", "reference", "workspace")
	var out bytes.Buffer
	if code, err := runModelCommand([]string{
		"search", "-workspace", root, "-term", "sales_core", "-limit", "1",
	}, &out); err != nil || code != 0 {
		t.Fatalf("search: code=%d err=%v body=%s", code, err, out.String())
	}
	var envelope struct {
		Result map[string]json.RawMessage `json:"result"`
	}
	if err := json.Unmarshal(out.Bytes(), &envelope); err != nil {
		t.Fatalf("not JSON: %v (%s)", err, out.String())
	}
	for _, field := range []string{"subjectCount", "distinctSubjectsOmitted"} {
		raw, ok := envelope.Result[field]
		if !ok {
			t.Fatalf("result carries no %s: %s", field, out.String())
		}
		var value int
		if err := json.Unmarshal(raw, &value); err != nil {
			t.Fatalf("%s is not a number: %s (%s)", field, raw, out.String())
		}
	}
}

// Zero hits is a success, not a failure and not "the text is nowhere": the
// coverage statement names what was not searched.
func TestModelSearchZeroHitsIsASuccessWithCoverage(t *testing.T) {
	root := filepath.Join("..", "..", "reference", "workspace")
	response, code := runModelSearch(t, "-workspace", root, "-term", "zzz-no-such-text")
	if code != 0 || !response.OK || response.Result == nil {
		t.Fatalf("zero hits: code=%d body=%+v", code, response)
	}
	if response.Result.Total != 0 || len(response.Result.Hits) != 0 ||
		len(response.Result.KindCounts) != 0 || response.Result.Omitted != 0 {
		t.Fatalf("zero-hit answer is not clean: %+v", response.Result)
	}
	if response.Result.Subjects == nil {
		t.Fatal("subjects is null on a zero-hit answer")
	}
	if response.Result.SubjectCount != 0 {
		t.Fatalf("subjectCount=%d on a zero-hit answer", response.Result.SubjectCount)
	}
	if response.Result.DistinctSubjectsOmitted != 0 {
		t.Fatalf("distinctSubjectsOmitted=%d on a zero-hit answer", response.Result.DistinctSubjectsOmitted)
	}
	if len(response.Result.Coverage.Derived) == 0 || len(response.Result.Coverage.NotDerived) == 0 ||
		len(response.Result.AuthorityLimits) == 0 {
		t.Fatalf("coverage/authorityLimits: %+v", response.Result)
	}
}

func TestModelSearchFailuresAreMachineReadable(t *testing.T) {
	root := filepath.Join("..", "..", "reference", "workspace")
	cases := []struct {
		name string
		args []string
		exit int
		code string
	}{
		{"missing workspace", []string{"-term", "order"}, 2, "INVALID_ARGUMENT"},
		{"empty workspace", []string{"-workspace=", "-term", "order"}, 2, "INVALID_ARGUMENT"},
		{"padded workspace", []string{"-workspace", " " + root, "-term", "order"}, 2, "INVALID_ARGUMENT"},
		{"neither term nor kind", []string{"-workspace", root}, 2, "INVALID_ARGUMENT"},
		{"empty term", []string{"-workspace", root, "-term="}, 2, "INVALID_ARGUMENT"},
		{"padded term", []string{"-workspace", root, "-term", " order "}, 2, "INVALID_ARGUMENT"},
		{"unknown kind", []string{"-workspace", root, "-term", "order", "-kind", "semantic-nope"}, 2, "INVALID_ARGUMENT"},
		{"empty kind", []string{"-workspace", root, "-kind="}, 2, "INVALID_ARGUMENT"},
		{"limit zero", []string{"-workspace", root, "-term", "order", "-limit", "0"}, 2, "INVALID_ARGUMENT"},
		{"limit over max", []string{"-workspace", root, "-term", "order", "-limit", "501"}, 2, "INVALID_ARGUMENT"},
		{"negative offset", []string{"-workspace", root, "-term", "order", "-offset", "-1"}, 2, "INVALID_ARGUMENT"},
		{"positional", []string{"-workspace", root, "-term", "order", "extra"}, 2, "INVALID_ARGUMENT"},
		{"repeated term", []string{"-workspace", root, "-term", "order", "-term", "customer"}, 2, "INVALID_ARGUMENT"},
		{"repeated workspace", []string{"-workspace", root, "-workspace", root, "-term", "order"}, 2, "INVALID_ARGUMENT"},
		{"missing workspace directory", []string{"-workspace", filepath.Join(root, "missing"), "-term", "order"}, 2, "WORKSPACE_LOAD_FAILED"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			response, code := runModelSearch(t, tc.args...)
			if code != tc.exit {
				t.Fatalf("exit=%d, want %d (%+v)", code, tc.exit, response)
			}
			if response.OK || response.Error == nil || response.Error.Code != tc.code ||
				response.SchemaVersion != "tei.cli.model-search/v0" {
				t.Fatalf("response=%+v", response)
			}
		})
	}
}

// `-h` explains this command's own grammar and succeeds on standard output,
// like every other read-only command (#3185).
func TestModelSearchHelpListsItsOwnGrammar(t *testing.T) {
	var out bytes.Buffer
	code, err := runModelCommand([]string{"search", "-h"}, &out)
	if err != nil || code != 0 {
		t.Fatalf("exit=%d err=%v out=%q", code, err, out.String())
	}
	text := out.String()
	if !strings.HasPrefix(text, "usage: tei model search") {
		t.Fatalf("help=%q", text)
	}
	for _, want := range []string{"-workspace", "-term", "-kind", "-limit", "-offset"} {
		if !strings.Contains(text, want) {
			t.Fatalf("help omits %q: %q", want, text)
		}
	}
	// Another command's flag must not appear in this grammar.
	for _, other := range []string{"-namespace", "-subject-id", "-expected-fingerprint"} {
		if strings.Contains(text, other) {
			t.Fatalf("help leaked another command's flag -%s: %q", other, text)
		}
	}
}

// What `tei commands` advertises for this command is exactly the grammar its
// own FlagSet accepts: a caller planning an invocation from the discovery
// document must not be told about a flag the parser refuses, or miss one it
// reads (#2332, #4166).
func TestModelSearchAdvertisesTheFlagsItAccepts(t *testing.T) {
	record, ok := discoveryRecord("model search")
	if !ok {
		t.Fatal("model search is not advertised by tei commands")
	}
	fs := flag.NewFlagSet("model search", flag.ContinueOnError)
	modelSearchFlags(fs)
	accepted := map[string]bool{}
	fs.VisitAll(func(f *flag.Flag) { accepted[f.Name] = true })
	advertised := append([]string{}, record.RequiredFlags...)
	advertised = append(advertised, record.OptionalFlags...)
	if len(advertised) != len(accepted) {
		t.Fatalf("advertised %v, accepts %v", advertised, accepted)
	}
	for _, name := range advertised {
		if !accepted[name] {
			t.Fatalf("advertises -%s, which the parser does not define", name)
		}
	}
	for name := range accepted {
		found := false
		for _, advertisedName := range advertised {
			if advertisedName == name {
				found = true
			}
		}
		if !found {
			t.Fatalf("accepts -%s, which tei commands does not advertise", name)
		}
	}
	// The workspace is the only required flag; the rest is optional, and -kind
	// carries the closed value set the projection validates against.
	if len(record.RequiredFlags) != 1 || record.RequiredFlags[0] != "workspace" {
		t.Fatalf("requiredFlags=%v", record.RequiredFlags)
	}
	if len(record.ArgumentValues["kind"]) != len(operations.InspectionSubjectKinds) {
		t.Fatalf("advertised kind values=%v", record.ArgumentValues["kind"])
	}
}

// An unknown flag is an argument failure with a named code, and a kind outside
// the closed set is refused the same way rather than answering nothing.
func TestModelSearchRefusesUnknownFlagAndKind(t *testing.T) {
	root := filepath.Join("..", "..", "reference", "workspace")
	for name, args := range map[string][]string{
		"unknown flag": {"-workspace", root, "-term", "order", "-not-a-flag"},
		"unknown kind": {"-workspace", root, "-term", "order", "-kind", "not-a-kind"},
	} {
		t.Run(name, func(t *testing.T) {
			response, code := runModelSearch(t, args...)
			if code != 2 || response.OK || response.Error == nil ||
				response.Error.Code != "INVALID_ARGUMENT" {
				t.Fatalf("code=%d body=%+v", code, response)
			}
		})
	}
}
