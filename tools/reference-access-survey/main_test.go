package main

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// source returns a source file whose direct accesses are known by construction,
// so the survey is checked against a fixture rather than against the repository
// (which changes under it).
func source() []byte {
	return []byte(`package example

import (
	"fmt"
	"os"
	"net"
	"net/http"
)

// os.Remove in a comment must not count.
const documented = "os.WriteFile is described here, not called"

func touch() {
	fmt.Println("hello")
	_, _ = os.ReadFile("in.txt")
	_ = os.WriteFile("out.txt", nil, 0o600)
	_ = http.Get("https://example.invalid")
	_, _ = net.Dial("tcp", "db:5432")
	_ = os.Getenv("TEI_MODE")
}
`)
}

func TestInspectCountsTheDirectAccesses(t *testing.T) {
	file, err := inspect("example.go", source())
	if err != nil {
		t.Fatalf("inspect: %v", err)
	}
	if len(file.Calls) != 5 {
		t.Fatalf("a comment or a string must not be counted: %+v", file.Calls)
	}
	// The result is ordered by position, so a reader can find the line.
	expected := []string{"os.ReadFile", "os.WriteFile", "http.Get", "net.Dial", "os.Getenv"}
	for index, call := range expected {
		if file.Calls[index].Call != call {
			t.Fatalf("position %d: expected %s, got %+v", index, call, file.Calls)
		}
	}
	if file.Calls[0].Line != 15 {
		t.Fatalf("the line must be the call site: %+v", file.Calls[0])
	}
}

func TestInspectGroupsTheCallsByResource(t *testing.T) {
	file, err := inspect("example.go", source())
	if err != nil {
		t.Fatalf("inspect: %v", err)
	}
	categories := map[string]int{}
	for _, call := range file.Calls {
		categories[call.Category]++
	}
	if categories["filesystem"] != 2 || categories["network"] != 2 || categories["environment"] != 1 {
		t.Fatalf("a network call must not be counted as a file, and an environment read must not be dropped: %+v", categories)
	}
}

func TestInspectRefusesAnUnparseableFile(t *testing.T) {
	// A file the parser cannot read must not be reported as "no access": that
	// would hide exactly the call a reader is looking for.
	if _, err := inspect("broken.go", []byte("package example\nfunc {")); err == nil {
		t.Fatal("an unparseable file must be reported, not silently counted as clean")
	}
}

func TestInspectReadsTheGuardedFlagAndTheBuildTag(t *testing.T) {
	guarded := []byte(`//go:build linux && amd64

package example

import "tei/internal/access"

var _ = access.Guard{}
`)
	file, err := inspect("guarded.go", guarded)
	if err != nil {
		t.Fatalf("inspect: %v", err)
	}
	if !file.Guarded {
		t.Fatal("a file that imports the guarded layer must be marked as guarded")
	}
	if len(file.BuildTags) != 1 || file.BuildTags[0] != "linux && amd64" {
		t.Fatalf("the build constraint must be recorded: %+v", file.BuildTags)
	}
}

func TestSurveyCountsGuardedFilesSeparately(t *testing.T) {
	directory := t.TempDir()
	if err := writeFile(directory, "guarded.go", `package example

import "tei/internal/access"

func touch() {
	_ = access.Guard{}
}
`); err != nil {
		t.Fatal(err)
	}
	if err := writeFile(directory, "direct.go", string(source())); err != nil {
		t.Fatal(err)
	}
	report, err := survey([]string{directory}, false)
	if err != nil {
		t.Fatalf("survey: %v", err)
	}
	if report.Totals.FilesScanned != 2 || report.Totals.FilesWithAccess != 1 {
		t.Fatalf("a guarded file with no direct call must not count as access: %+v", report.Totals)
	}
	if report.Totals.DirectCalls != 5 || report.Totals.FilesGuarded != 0 {
		t.Fatalf("unexpected totals: %+v", report.Totals)
	}
	if len(report.ByDirectory) != 1 || report.ByDirectory[0].Calls != 5 {
		t.Fatalf("the directory count must reflect the calls: %+v", report.ByDirectory)
	}
}

func TestSurveySkipsTestFilesUnlessAsked(t *testing.T) {
	directory := t.TempDir()
	if err := writeFile(directory, "code.go", string(source())); err != nil {
		t.Fatal(err)
	}
	if err := writeFile(directory, "code_test.go", string(source())); err != nil {
		t.Fatal(err)
	}
	without, err := survey([]string{directory}, false)
	if err != nil {
		t.Fatalf("survey: %v", err)
	}
	with, err := survey([]string{directory}, true)
	if err != nil {
		t.Fatalf("survey: %v", err)
	}
	if without.Totals.DirectCalls == with.Totals.DirectCalls {
		t.Fatalf("asking for test files must change the count: %d vs %d", without.Totals.DirectCalls, with.Totals.DirectCalls)
	}
	if without.Totals.FilesScanned != 1 || with.Totals.FilesScanned != 2 {
		t.Fatalf("unexpected scanned counts: %d vs %d", without.Totals.FilesScanned, with.Totals.FilesScanned)
	}
}

// TestSurveyDoesNotCountAnOutboundRequestThroughAClientValue pins a known gap
// rather than hiding it: an outbound request made as `client.Do(req)` goes
// through a value, so the call is not `http.Get` or `net.Dial` and the survey
// does not see it. Counting every method named Do would be worse (it is a common
// method name), so the gap stays and the number must not be read as complete.
func TestSurveyDoesNotCountAnOutboundRequestThroughAClientValue(t *testing.T) {
	file, err := inspect("client.go", []byte(`package example

import "net/http"

func call(client *http.Client, request *http.Request) {
	_, _ = client.Do(request)
}
`))
	if err != nil {
		t.Fatalf("inspect: %v", err)
	}
	if len(file.Calls) != 0 {
		t.Fatalf("the survey must not claim to see this call, or the number reads as complete: %+v", file.Calls)
	}
}

// TestSummaryShowsTheNumbersADocumentQuotes checks that the summary carries the
// totals and the busiest directories, so a document can quote it without asking
// a reader to open the full JSON.
func TestSummaryShowsTheNumbersADocumentQuotes(t *testing.T) {
	report := Report{
		Totals: Totals{
			FilesScanned: 3, FilesWithAccess: 2, FilesGuarded: 0, DirectCalls: 5,
			ByCategory: map[string]int{"filesystem": 4, "network": 1},
		},
		ByDirectory: []DirectoryUse{
			{Directory: "tools/busiest", Files: 2, Calls: 5},
			{Directory: "tools/quiet", Files: 1, Calls: 0},
		},
	}
	text := summary(report)
	for _, want := range []string{"scanned 3", "calls 5", "filesystem 4", "network 1", "tools/busiest"} {
		if !strings.Contains(text, want) {
			t.Fatalf("the summary must contain %q:\n%s", want, text)
		}
	}
}

// TestSummaryIsBounded checks that the summary stays short even when the survey
// finds many directories. An unbounded summary is the full JSON again.
func TestSummaryIsBounded(t *testing.T) {
	report := Report{Totals: Totals{ByCategory: map[string]int{}}}
	for index := 0; index < 40; index++ {
		report.ByDirectory = append(report.ByDirectory, DirectoryUse{Directory: "tools/d" + strconv.Itoa(index), Calls: 40 - index})
	}
	lines := strings.Count(strings.TrimRight(summary(report), "\n"), "\n") + 1
	if lines > 12 {
		t.Fatalf("the summary must stay readable, or it is not a summary: %d lines", lines)
	}
}

func TestRenderIsStable(t *testing.T) {
	report, err := survey([]string{t.TempDir()}, false)
	if err != nil {
		t.Fatalf("survey: %v", err)
	}
	first, err := render(report)
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	second, err := render(report)
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if string(first) != string(second) {
		t.Fatal("the same survey must render the same bytes, or two runs cannot be compared")
	}
	if !strings.HasSuffix(string(first), "\n") {
		t.Fatal("the artifact must end with a newline")
	}
}

// writeFile writes one file into the directory.
func writeFile(directory, name, content string) error {
	return os.WriteFile(filepath.Join(directory, name), []byte(content), 0o600)
}
