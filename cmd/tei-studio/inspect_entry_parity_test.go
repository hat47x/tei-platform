//go:build server

package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"sort"
	"testing"

	"tei/internal/inspectoradapter"
	"tei/internal/operations"
	"tei/internal/sourceindex"
)

// This file compares the two PRODUCT entries a person actually uses: the real
// tei CLI subprocess and the Studio Host's InspectSubject method, reached through
// the Bridge a renderer calls. Both are expected to select the same local
// reference Inspector Provider and to enter internal/operations.InspectSubject.
//
// That shared Operation is why their answers should agree, and it is equally why
// agreement by itself does not prove the Operation is correct.
//
// Deliberately NOT compared: the transport envelopes have different shapes
// (tei.cli.inspect/v0 report vs studiohost InspectionResponse), so whole-response
// byte equality is never asserted. Only target identity, meaning, Source, Scope
// and diagnostics are matched. The inner canonical attribute ORDER is also not
// required to match the declaration order: the projection emits sorted IDs, so
// attribute and relation IDs are compared as sets.
//
// Expectations are loaded from
// testdata/inspect-entry-parity/independent-expected.candidate.json, which was
// derived from the Workspace Source bytes and the .tei declarations rather than
// read back from any Inspector answer. That file is parsed by this test, so it
// cannot drift into an unenforced second copy.

const (
	parityReferenceRoot    = "../../reference/workspace"
	parityStandardCRUDRoot = "../../reference/standard-crud/workspace"

	// The two Workspaces are addressed by directory in this process; every other
	// value (identity, meaning, Source) is read from the Expected document, so
	// there is exactly one copy of each expectation.

	parityAttrCustomer = "sales.order.customer_id"
	parityAttrForeign  = "customer.customer.customer_id"

	// parityCanonicalEntry is the linked symbol both production entries select.
	parityCanonicalEntry = "tei/internal/operations.InspectSubject"

	// parityExpectedFile is the adopted independent Expected. It is parsed rather
	// than transcribed, so it cannot become an unenforced second copy.
	parityExpectedFile = "../../testdata/inspect-entry-parity/independent-expected.candidate.json"
)

// parityExpected is the subset of the Expected document this test enforces. Every
// value used below comes from the file, not from a constant, so a fixture edit
// that contradicts the Workspace is caught here instead of silently ignored.
type parityExpected struct {
	Schema    string `json:"schema"`
	Workspace struct {
		Path     string `json:"path"`
		Identity struct {
			Name           string `json:"name"`
			ManifestSha256 string `json:"manifestSha256"`
		} `json:"identity"`
		Subjects map[string]parityExpectedSubject `json:"subjects"`
	} `json:"workspaceA"`
	WorkspaceB struct {
		Path     string `json:"path"`
		Identity struct {
			Name           string `json:"name"`
			ManifestSha256 string `json:"manifestSha256"`
		} `json:"identity"`
		Subjects map[string]parityExpectedSubject `json:"subjects"`
	} `json:"workspaceB"`
	Expected []struct {
		ID string `json:"id"`
	} `json:"expected"`
}

type parityExpectedSubject struct {
	Kind       string   `json:"kind"`
	ID         string   `json:"id"`
	Attributes []string `json:"attributes"`
	Relations  []string `json:"relations"`
	Symbol     string   `json:"symbol"`
	Definition struct {
		ArtifactPath        string `json:"artifactPath"`
		ArtifactFingerprint string `json:"artifactFingerprint"`
		Role                string `json:"role"`
		StartOffset         int    `json:"startOffset"`
	} `json:"definition"`
}

func parityLoadExpected(t *testing.T) parityExpected {
	t.Helper()
	raw, err := os.ReadFile(parityExpectedFile)
	if err != nil {
		t.Fatalf("read the independent Expected: %v", err)
	}
	var expected parityExpected
	decoder := json.NewDecoder(bytes.NewReader(raw))
	if err := decoder.Decode(&expected); err != nil {
		t.Fatalf("the independent Expected is not valid JSON for this test: %v", err)
	}
	// The document also carries prose (adoption, authority, per-case check/expect),
	// so unknown-key rejection is not usable here. A misspelled key inside a
	// subject would instead decode to a zero value, which the required-field
	// checks below turn into a failure rather than a silently weaker assertion.
	if expected.Schema != "tei-inspect-entry-parity-independent-expected/v1" {
		t.Fatalf("the independent Expected has an unexpected schema %q", expected.Schema)
	}
	// Every field these comparisons rely on must be present, so a fixture edit
	// cannot drop one and turn the corresponding check into a no-op.
	for _, workspace := range []struct {
		label    string
		path     string
		name     string
		sha      string
		subjects map[string]parityExpectedSubject
	}{
		{"workspaceA", expected.Workspace.Path, expected.Workspace.Identity.Name,
			expected.Workspace.Identity.ManifestSha256, expected.Workspace.Subjects},
		{"workspaceB", expected.WorkspaceB.Path, expected.WorkspaceB.Identity.Name,
			expected.WorkspaceB.Identity.ManifestSha256, expected.WorkspaceB.Subjects},
	} {
		if workspace.path == "" || workspace.name == "" || workspace.sha == "" {
			t.Fatalf("%s is missing its path, identity name or manifest digest", workspace.label)
		}
		if len(workspace.subjects) == 0 {
			t.Fatalf("%s declares no subject", workspace.label)
		}
		for id, subject := range workspace.subjects {
			if subject.Kind == "" || subject.ID == "" || subject.Symbol == "" {
				t.Fatalf("%s/%s is missing kind, id or symbol", workspace.label, id)
			}
			if subject.Definition.ArtifactPath == "" ||
				subject.Definition.ArtifactFingerprint == "" ||
				subject.Definition.Role == "" {
				t.Fatalf("%s/%s is missing a complete definition", workspace.label, id)
			}
			if subject.Definition.StartOffset <= 0 {
				t.Fatalf("%s/%s is missing a positive startOffset", workspace.label, id)
			}
		}
	}
	return expected
}

// paritySameSet compares two ID lists as sets. Duplicates stay significant, so a
// repeated declaration cannot hide behind set equality.
func paritySameSet(got, want []string) bool {
	if len(got) != len(want) {
		return false
	}
	g := append([]string(nil), got...)
	w := append([]string(nil), want...)
	sort.Strings(g)
	sort.Strings(w)
	return reflect.DeepEqual(g, w)
}

// parityObserved is what one entry reported for one request, reduced to the
// fields that are supposed to agree. Adapting both transports into this shape
// first keeps a presentation difference from masquerading as a meaning one.
type parityObserved struct {
	Kind        string
	ID          string
	Symbol      string
	Effect      string
	CanonicalID string
	Attributes  []string
	Relations   []string
	Workspace   string
	ManifestSHA string
	SourcePath  string
	SourceFP    string
	SourceRole  string
	Offset      int
	WorldFP     string
	SourceIndex string
	Limits      []string
	FailureCode string
}

func parityCanonicalFields(canonical json.RawMessage) (string, string, []string, []string, error) {
	var body struct {
		ID         string `json:"id"`
		Symbol     string `json:"symbol"`
		Attributes []struct {
			ID string `json:"id"`
		} `json:"attributes"`
		Relations []struct {
			ID string `json:"id"`
		} `json:"relations"`
	}
	if err := json.Unmarshal(canonical, &body); err != nil {
		return "", "", nil, nil, err
	}
	var attributes, relations []string
	for _, attribute := range body.Attributes {
		attributes = append(attributes, attribute.ID)
	}
	for _, relation := range body.Relations {
		relations = append(relations, relation.ID)
	}
	return body.ID, body.Symbol, attributes, relations, nil
}

// parityFromOperation normalizes a SubjectInspection that either entry produced.
func parityFromOperation(t *testing.T, inspection operations.SubjectInspection) parityObserved {
	t.Helper()
	id, symbol, attributes, relations, err := parityCanonicalFields(inspection.Canonical)
	if err != nil {
		t.Fatalf("canonical is not a declaration object: %v", err)
	}
	observed := parityObserved{
		Kind:        inspection.Subject.Kind,
		ID:          inspection.Subject.ID,
		Symbol:      symbol,
		Effect:      inspection.Effect,
		CanonicalID: id,
		Attributes:  attributes,
		Relations:   relations,
		WorldFP:     inspection.WorldFingerprint,
		SourceIndex: inspection.SourceIndexFingerprint,
		Limits:      append([]string(nil), inspection.AuthorityLimits...),
	}
	if inspection.Workspace != nil {
		observed.Workspace = inspection.Workspace.Name
		observed.ManifestSHA = inspection.Workspace.Manifest.SHA256
	}
	if len(inspection.Sources.Artifacts) == 1 {
		observed.SourcePath = inspection.Sources.Artifacts[0].Path
		observed.SourceFP = inspection.Sources.Artifacts[0].Fingerprint
	}
	if len(inspection.Sources.References) == 1 {
		reference := inspection.Sources.References[0]
		observed.SourceRole = string(reference.Role)
		observed.Offset = reference.Range.Start.Offset
	}
	return observed
}

// parityNewBridge composes the production Studio host shape over the given
// startup-allowlisted roots.
func parityNewBridge(t *testing.T, roots ...string) *StudioBridge {
	t.Helper()
	inspector, err := inspectoradapter.NewLocalInspectorHost()
	if err != nil {
		t.Fatalf("compose the reference Inspector Host: %v", err)
	}
	host, err := newStudioHostWithInspector(roots, inspector)
	if err != nil {
		t.Fatalf("compose the Studio Host: %v", err)
	}
	return &StudioBridge{host: host}
}

func parityStudioObserve(t *testing.T, bridge *StudioBridge, workspaceID string, subject sourceindex.Subject) parityObserved {
	t.Helper()
	response := bridge.InspectSubject(workspaceID, subject.Kind, subject.ID)
	if response.OK && response.Value != nil && response.Failure == nil {
		return parityFromOperation(t, *response.Value)
	}
	observed := parityObserved{Kind: subject.Kind, ID: subject.ID}
	if response.Failure != nil {
		observed.FailureCode = response.Failure.Code
	}
	return observed
}

// parityCLIReport is the minimal CLI envelope shape. The CLI lives in a different
// Go module, so this file reads its JSON rather than importing its package.
type parityCLIReport struct {
	OK     bool                          `json:"ok"`
	Result *operations.SubjectInspection `json:"result"`
	Error  *struct {
		Code string `json:"code"`
	} `json:"error"`
}

// parityCLIDiagnostic maps the CLI's classified error code onto the Studio
// Failure vocabulary. Only existing codes are translated; anything else is kept
// so an unexpected classification fails a comparison instead of hiding in it.
func parityCLIDiagnostic(code string) string {
	switch code {
	case operations.InspectionNotFound:
		return "SUBJECT_NOT_FOUND"
	case operations.InspectionAmbiguous:
		return "SUBJECT_AMBIGUOUS"
	case operations.InspectionInvalidSubject:
		return "INVALID_SUBJECT"
	case operations.InspectionUnsupportedKind:
		return "UNSUPPORTED_SUBJECT_KIND"
	}
	return code
}

func parityCLIObserve(t *testing.T, binary, root string, subject sourceindex.Subject) parityObserved {
	t.Helper()
	command := exec.Command(binary, "inspect", "-workspace", root,
		"-subject-kind", subject.Kind, "-subject-id", subject.ID)
	var stdout, stderr bytes.Buffer
	command.Stdout, command.Stderr = &stdout, &stderr
	runErr := command.Run()
	exit := 0
	if runErr != nil {
		var status *exec.ExitError
		if !errors.As(runErr, &status) {
			t.Fatalf("start the tei CLI for %v: %v", subject, runErr)
		}
		exit = status.ExitCode()
	}
	var report parityCLIReport
	if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
		t.Fatalf("CLI did not emit one JSON document for %v: exit=%d stderr=%q stdout=%q",
			subject, exit, stderr.String(), stdout.String())
	}
	if report.OK {
		if exit != 0 || report.Result == nil || report.Error != nil {
			t.Fatalf("CLI reported success inconsistently for %v: exit=%d %+v", subject, exit, report)
		}
		return parityFromOperation(t, *report.Result)
	}
	if exit == 0 || report.Result != nil || report.Error == nil {
		t.Fatalf("CLI reported failure inconsistently for %v: exit=%d %+v", subject, exit, report)
	}
	return parityObserved{Kind: subject.Kind, ID: subject.ID, FailureCode: parityCLIDiagnostic(report.Error.Code)}
}

// parityBuildCLI builds the real tei executable for this test.
func parityBuildCLI(t *testing.T) string {
	t.Helper()
	binary := filepath.Join(t.TempDir(), "tei")
	if runtime.GOOS == "windows" {
		binary += ".exe"
	}
	build := exec.Command("go", "build", "-o", binary, "tei/cmd/tei")
	build.Dir = filepath.Join("..", "..")
	if output, err := build.CombinedOutput(); err != nil {
		// A missing CLI is not evidence about either entry's behaviour, and the
		// Studio CI step that runs this package does not use -v, so a skip would
		// be invisible. Fail instead of reporting an unrun comparison as a pass.
		t.Fatalf("the tei CLI must be buildable for this comparison: %v\n%s", err, output)
	}
	return binary
}

// TestInspectParityExpectationsMatchTheSourceBytes ties the independent Expected
// to the pinned bytes. A hash typed by hand from a tool's output once disagreed
// with the file it described by one character, so every recorded fingerprint and
// offset is recomputed here from the files the document names.
func TestInspectParityExpectationsMatchTheSourceBytes(t *testing.T) {
	expected := parityLoadExpected(t)

	// Every fingerprint in the document must equal the bytes at the path the
	// document pairs it with.
	for _, workspace := range []struct {
		name     string
		path     string
		manifest string
		subjects map[string]parityExpectedSubject
	}{
		{"workspaceA", expected.Workspace.Path, expected.Workspace.Identity.ManifestSha256, expected.Workspace.Subjects},
		{"workspaceB", expected.WorkspaceB.Path, expected.WorkspaceB.Identity.ManifestSha256, expected.WorkspaceB.Subjects},
	} {
		t.Run(workspace.name+" manifest", func(t *testing.T) {
			content, err := os.ReadFile(filepath.Join("..", "..", filepath.FromSlash(workspace.path), "tei.workspace.yaml"))
			if err != nil {
				t.Fatalf("read the manifest named by the Expected: %v", err)
			}
			if got := sourceindex.Fingerprint(content); got != workspace.manifest {
				t.Fatalf("%s records manifest %q but the file is %q", workspace.name, workspace.manifest, got)
			}
		})
		for id, subject := range workspace.subjects {
			subject := subject
			t.Run(workspace.name+" "+id, func(t *testing.T) {
				if subject.Definition.ArtifactPath == "" {
					t.Fatalf("%s records no Source artifact path", id)
				}
				sourcePath := filepath.Join("..", "..", filepath.FromSlash(workspace.path), filepath.FromSlash(subject.Definition.ArtifactPath))
				content, err := os.ReadFile(sourcePath)
				if err != nil {
					t.Fatalf("read the Source named by the Expected: %v", err)
				}
				if got := sourceindex.Fingerprint(content); got != subject.Definition.ArtifactFingerprint {
					t.Fatalf("%s records artifact %q but the file is %q",
						id, subject.Definition.ArtifactFingerprint, got)
				}
				// A recorded offset must land on the identifier it names, not on
				// some earlier byte that happens to match.
				start := subject.Definition.StartOffset
				if start <= 0 || start >= len(content) {
					t.Fatalf("%s records offset %d, outside the %d-byte Source", id, start, len(content))
				}
				if subject.Symbol == "" {
					t.Fatalf("%s records no symbol to anchor its offset", id)
				}
				if !bytes.HasPrefix(content[start:], []byte(subject.Symbol)) {
					t.Fatalf("%s records offset %d, which does not start the symbol %q",
						id, start, subject.Symbol)
				}
			})
		}
	}

	// The document must still describe the six cases this test enforces.
	want := []string{
		"IEP-1-same-subject-same-meaning",
		"IEP-2-same-symbol-different-id",
		"IEP-3-same-id-different-workspace",
		"IEP-4-stale-result-not-current",
		"IEP-5-result-equality-is-not-path-identity",
		"IEP-6-one-sided-failure-is-reported",
	}
	var got []string
	for _, item := range expected.Expected {
		got = append(got, item.ID)
	}
	sort.Strings(got)
	sort.Strings(want)
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("the Expected document names %v, want %v", got, want)
	}
}

// parityAuthorityLimits is the fixed statement set pinned from the Expected, not
// read from operations.InspectionAuthorityLimits(). Comparing the operation to the
// function that produced its limits would pass even if a statement disappeared.
var parityAuthorityLimits = []string{
	"inspection-is-a-derived-read-only-projection",
	"source-location-is-not-semantic-acceptance",
	"source-location-is-not-write-authority",
	"local-read-is-not-disclosure-permission",
}

// IEP-1: for one target ID and one Source revision, both entries report the same
// semantic result.
func TestInspectEntriesAgreeOnOneSubjectAtOneSourceRevision(t *testing.T) {
	expected := parityLoadExpected(t)
	if expected.Workspace.Path != "reference/workspace" {
		t.Fatalf("the Expected no longer names the compared Workspace: %q", expected.Workspace.Path)
	}
	order, ok := expected.Workspace.Subjects["sales.order"]
	if !ok {
		t.Fatal("the Expected no longer describes sales.order in workspaceA")
	}

	binary := parityBuildCLI(t)
	bridge := parityNewBridge(t, parityReferenceRoot)
	subject := sourceindex.Subject{Kind: order.Kind, ID: order.ID}

	fromCLI := parityCLIObserve(t, binary, parityReferenceRoot, subject)
	fromStudio := parityStudioObserve(t, bridge, "workspace-1", subject)

	for _, entry := range []struct {
		name  string
		value parityObserved
	}{{"cli", fromCLI}, {"studio", fromStudio}} {
		got := entry.value
		if got.FailureCode != "" {
			t.Fatalf("%s failed to inspect the declared subject: %+v", entry.name, got)
		}
		if got.Kind != subject.Kind || got.ID != subject.ID || got.CanonicalID != subject.ID {
			t.Fatalf("%s reported another target: %+v", entry.name, got)
		}
		if got.Effect != "read-only" {
			t.Fatalf("%s did not report a read-only effect: %+v", entry.name, got)
		}
		if !paritySameSet(got.Attributes, order.Attributes) {
			t.Fatalf("%s attribute meaning drifted: got %v want %v",
				entry.name, got.Attributes, order.Attributes)
		}
		if !paritySameSet(got.Relations, order.Relations) {
			t.Fatalf("%s relation meaning drifted: got %v want %v",
				entry.name, got.Relations, order.Relations)
		}
		if got.Workspace != expected.Workspace.Identity.Name ||
			got.ManifestSHA != expected.Workspace.Identity.ManifestSha256 {
			t.Fatalf("%s read another Workspace: %+v", entry.name, got)
		}
		if got.SourcePath != order.Definition.ArtifactPath {
			t.Fatalf("%s named another Source artifact: %+v", entry.name, got)
		}
		if got.SourceFP != order.Definition.ArtifactFingerprint {
			t.Fatalf("%s reported Source fingerprint %q, expected %q",
				entry.name, got.SourceFP, order.Definition.ArtifactFingerprint)
		}
		if got.SourceRole != order.Definition.Role {
			t.Fatalf("%s reported Source role %q: %+v", entry.name, got.SourceRole, got)
		}
		if got.Offset != order.Definition.StartOffset {
			t.Fatalf("%s reported Source offset %d, expected %d",
				entry.name, got.Offset, order.Definition.StartOffset)
		}
		if !reflect.DeepEqual(got.Limits, parityAuthorityLimits) {
			t.Fatalf("%s changed the authority limits: got %v want %v",
				entry.name, got.Limits, parityAuthorityLimits)
		}
		if got.WorldFP == "" || got.SourceIndex == "" {
			t.Fatalf("%s omitted an observation fingerprint: %+v", entry.name, got)
		}
	}

	// One Subject at one unchanged revision: the normalized views must be equal,
	// fingerprints included.
	if !reflect.DeepEqual(fromCLI, fromStudio) {
		t.Fatalf("the entries disagreed on one Subject at one revision:\nCLI   =%+v\nStudio=%+v", fromCLI, fromStudio)
	}
}

// IEP-2: two attributes that share a symbol but have different canonical IDs stay
// separate; neither is answered with the other's Source.
func TestInspectEntriesDoNotMergeSameSymbolAcrossIDs(t *testing.T) {
	expected := parityLoadExpected(t)
	customerExpected, okCustomer := expected.Workspace.Subjects[parityAttrCustomer]
	foreignExpected, okForeign := expected.Workspace.Subjects[parityAttrForeign]
	if !okCustomer || !okForeign {
		t.Fatalf("the Expected no longer describes both attributes %q and %q",
			parityAttrCustomer, parityAttrForeign)
	}
	// The premise of this case is a SHARED symbol on distinct IDs. If the fixture
	// stopped sharing it, the case would pass without testing anything.
	if customerExpected.Symbol == "" || customerExpected.Symbol != foreignExpected.Symbol {
		t.Fatalf("the Expected no longer gives both attributes one symbol: %q vs %q",
			customerExpected.Symbol, foreignExpected.Symbol)
	}
	if customerExpected.ID == foreignExpected.ID {
		t.Fatalf("the Expected no longer gives the two attributes distinct IDs: %q", customerExpected.ID)
	}

	binary := parityBuildCLI(t)
	bridge := parityNewBridge(t, parityReferenceRoot)

	observations := map[string]parityObserved{}
	for _, wanted := range []parityExpectedSubject{customerExpected, foreignExpected} {
		subject := sourceindex.Subject{Kind: wanted.Kind, ID: wanted.ID}
		fromCLI := parityCLIObserve(t, binary, parityReferenceRoot, subject)
		fromStudio := parityStudioObserve(t, bridge, "workspace-1", subject)
		if fromCLI.FailureCode != "" || fromStudio.FailureCode != "" {
			t.Fatalf("%s was not inspectable through both entries: cli=%+v studio=%+v",
				wanted.ID, fromCLI, fromStudio)
		}
		if !reflect.DeepEqual(fromCLI, fromStudio) {
			t.Fatalf("%s disagreed between entries:\nCLI   =%+v\nStudio=%+v", wanted.ID, fromCLI, fromStudio)
		}
		if fromCLI.ID != wanted.ID || fromCLI.CanonicalID != wanted.ID {
			t.Fatalf("%s was answered as another subject: %+v", wanted.ID, fromCLI)
		}
		// The shared symbol must be present on each answer, which is what makes
		// the non-merging claim meaningful.
		if fromCLI.Symbol != wanted.Symbol {
			t.Fatalf("%s reported symbol %q, expected %q", wanted.ID, fromCLI.Symbol, wanted.Symbol)
		}
		if fromCLI.Offset != wanted.Definition.StartOffset {
			t.Fatalf("%s reported Source offset %d, expected %d",
				wanted.ID, fromCLI.Offset, wanted.Definition.StartOffset)
		}
		observations[wanted.ID] = fromCLI
	}

	customer, foreign := observations[parityAttrCustomer], observations[parityAttrForeign]
	if customer.SourcePath == foreign.SourcePath {
		t.Fatalf("two Attributes sharing a symbol reported one Source: %+v / %+v", customer, foreign)
	}
	// The two answers must carry the same symbol, and that is exactly what makes
	// the distinct Sources and spans meaningful: the fixture premise above forces
	// the expected symbols equal, so this compares two OBSERVED values.
	//
	// A single Source artifact covering both declarations cannot exist here,
	// because the two offsets fall in different files; asserting the spans differ
	// is what would break if the operation resolved both IDs to one declaration.
	if customer.SourceFP == foreign.SourceFP {
		t.Fatalf("two Attributes sharing a symbol resolved to one artifact: %+v / %+v", customer, foreign)
	}
	if customer.Offset == foreign.Offset {
		t.Fatalf("two Attributes sharing a symbol resolved to one Source position: %+v / %+v", customer, foreign)
	}
	if customer.Symbol != foreign.Symbol {
		t.Fatalf("the shared symbol was not preserved across the two answers: %+v / %+v", customer, foreign)
	}
}

// IEP-3: one canonical ID declared by two Workspaces is kept apart.
func TestInspectEntriesKeepSameIDWorkspacesApart(t *testing.T) {
	expected := parityLoadExpected(t)
	referenceOrder, okReference := expected.Workspace.Subjects["sales.order"]
	standardOrder, okStandard := expected.WorkspaceB.Subjects["sales.order"]
	if !okReference || !okStandard {
		t.Fatal("the Expected no longer describes sales.order in both Workspaces")
	}
	// The premise is one ID in two Workspaces. Without it the case proves nothing.
	if referenceOrder.ID != standardOrder.ID {
		t.Fatalf("the Expected no longer uses one ID for both Workspaces: %q vs %q",
			referenceOrder.ID, standardOrder.ID)
	}
	if reflect.DeepEqual(referenceOrder.Attributes, standardOrder.Attributes) {
		t.Fatal("the Expected no longer gives the two Workspaces different meaning")
	}

	binary := parityBuildCLI(t)
	bridge := parityNewBridge(t, parityReferenceRoot, parityStandardCRUDRoot)
	subject := sourceindex.Subject{Kind: referenceOrder.Kind, ID: referenceOrder.ID}

	targets := []struct {
		name      string
		root      string
		workspace string
		identity  struct {
			name     string
			manifest string
		}
		subject parityExpectedSubject
	}{
		{
			name: "reference", root: parityReferenceRoot, workspace: "workspace-1",
			identity: struct {
				name     string
				manifest string
			}{expected.Workspace.Identity.Name, expected.Workspace.Identity.ManifestSha256},
			subject: referenceOrder,
		},
		{
			name: "standard-crud", root: parityStandardCRUDRoot, workspace: "workspace-2",
			identity: struct {
				name     string
				manifest string
			}{expected.WorkspaceB.Identity.Name, expected.WorkspaceB.Identity.ManifestSha256},
			subject: standardOrder,
		},
	}

	observed := map[string]parityObserved{}
	for _, target := range targets {
		fromCLI := parityCLIObserve(t, binary, target.root, subject)
		fromStudio := parityStudioObserve(t, bridge, target.workspace, subject)
		if fromCLI.FailureCode != "" || fromStudio.FailureCode != "" {
			t.Fatalf("%s was not inspectable through both entries: cli=%+v studio=%+v",
				target.name, fromCLI, fromStudio)
		}
		if !reflect.DeepEqual(fromCLI, fromStudio) {
			t.Fatalf("%s disagreed between entries:\nCLI   =%+v\nStudio=%+v", target.name, fromCLI, fromStudio)
		}
		if fromCLI.Workspace != target.identity.name || fromCLI.ManifestSHA != target.identity.manifest {
			t.Fatalf("%s reported another Workspace identity: %+v", target.name, fromCLI)
		}
		if !paritySameSet(fromCLI.Attributes, target.subject.Attributes) {
			t.Fatalf("%s meaning drifted: got %v want %v",
				target.name, fromCLI.Attributes, target.subject.Attributes)
		}
		if !paritySameSet(fromCLI.Relations, target.subject.Relations) {
			t.Fatalf("%s relations drifted: got %v want %v",
				target.name, fromCLI.Relations, target.subject.Relations)
		}
		if fromCLI.SourcePath != target.subject.Definition.ArtifactPath ||
			fromCLI.SourceFP != target.subject.Definition.ArtifactFingerprint ||
			fromCLI.Offset != target.subject.Definition.StartOffset {
			t.Fatalf("%s Source evidence drifted: %+v", target.name, fromCLI)
		}
		observed[target.name] = fromCLI
	}

	// Both per-target loops above pinned the observed values to the fixture, so
	// this is a FIXTURE premise check, not a second observation check: it fails if
	// the Expected stopped making the two Workspaces distinguishable, which is the
	// condition that gives the whole case its meaning.
	a, b := observed["reference"], observed["standard-crud"]
	if a.Workspace == b.Workspace || a.ManifestSHA == b.ManifestSHA ||
		a.SourceFP == b.SourceFP || paritySameSet(a.Attributes, b.Attributes) {
		t.Fatalf("the Expected no longer separates two Workspaces declaring sales.order:\nA=%+v\nB=%+v", a, b)
	}
}

// IEP-4: after a Source update, an earlier result is not treated as current.
func TestInspectEntriesDoNotServeAStaleResultAfterSourceUpdate(t *testing.T) {
	expected := parityLoadExpected(t)
	order, ok := expected.Workspace.Subjects["sales.order"]
	if !ok {
		t.Fatal("the Expected no longer describes sales.order in workspaceA")
	}

	binary := parityBuildCLI(t)
	editedRoot := t.TempDir()
	// Copy the public Workspace; the committed fixture is never written to.
	if err := copyWorkspaceTree(parityReferenceRoot, editedRoot); err != nil {
		t.Fatalf("copy the reference Workspace: %v", err)
	}
	bridge := parityNewBridge(t, editedRoot)
	subject := sourceindex.Subject{Kind: order.Kind, ID: order.ID}

	beforeCLI := parityCLIObserve(t, binary, editedRoot, subject)
	beforeStudio := parityStudioObserve(t, bridge, "workspace-1", subject)
	if beforeCLI.FailureCode != "" || beforeStudio.FailureCode != "" {
		t.Fatalf("pre-edit inspection failed: cli=%+v studio=%+v", beforeCLI, beforeStudio)
	}
	if !reflect.DeepEqual(beforeCLI, beforeStudio) {
		t.Fatalf("the entries disagreed before the edit:\nCLI   =%+v\nStudio=%+v", beforeCLI, beforeStudio)
	}
	if beforeCLI.SourceFP != order.Definition.ArtifactFingerprint {
		t.Fatalf("the pre-edit Source was not the pinned one: %+v", beforeCLI)
	}

	sourcePath := filepath.Join(editedRoot, filepath.FromSlash(order.Definition.ArtifactPath))
	original, err := os.ReadFile(sourcePath)
	if err != nil {
		t.Fatal(err)
	}
	const anchor = "attribute externalId type std.text"
	if bytes.Count(original, []byte(anchor)) != 1 {
		t.Fatalf("the pinned Source no longer contains exactly one %q declaration", anchor)
	}
	edited := bytes.Replace(original, []byte(anchor), []byte(anchor+"\n    attribute note type std.text"), 1)
	if err := os.WriteFile(sourcePath, edited, 0o644); err != nil {
		t.Fatal(err)
	}

	afterCLI := parityCLIObserve(t, binary, editedRoot, subject)
	afterStudio := parityStudioObserve(t, bridge, "workspace-1", subject)
	if afterCLI.FailureCode != "" || afterStudio.FailureCode != "" {
		t.Fatalf("post-edit inspection failed: cli=%+v studio=%+v", afterCLI, afterStudio)
	}
	if !reflect.DeepEqual(afterCLI, afterStudio) {
		t.Fatalf("the entries disagreed after the Source update:\nCLI   =%+v\nStudio=%+v", afterCLI, afterStudio)
	}

	wantAttributes := append(append([]string(nil), order.Attributes...), "sales.order.note")
	for _, entry := range []struct {
		name  string
		value parityObserved
	}{{"cli", afterCLI}, {"studio", afterStudio}} {
		got := entry.value
		if !paritySameSet(got.Attributes, wantAttributes) {
			t.Fatalf("%s did not read the edited Source: got %v want %v", entry.name, got.Attributes, wantAttributes)
		}
		if got.SourceFP == order.Definition.ArtifactFingerprint {
			t.Fatalf("%s served the pre-edit Source as current: %+v", entry.name, got)
		}
		if got.SourceIndex == beforeCLI.SourceIndex || got.WorldFP == beforeCLI.WorldFP {
			t.Fatalf("%s served a pre-edit revision as current: %+v", entry.name, got)
		}
	}
	if reflect.DeepEqual(beforeCLI, afterCLI) {
		t.Fatal("the CLI reported the pre-edit result as the current one")
	}
}

// parallelEntryA and parallelEntryB are two distinct Inspector Providers compiled
// into this test binary. Each answers through the canonical Operation, so an
// identical result cannot be used to claim the two entries followed one path.
func parallelEntryA(root string, subject sourceindex.Subject) (operations.SubjectInspection, error) {
	return operations.InspectSubject(root, subject)
}

func parallelEntryB(root string, subject sourceindex.Subject) (operations.SubjectInspection, error) {
	return operations.InspectSubject(root, subject)
}

const (
	parityParallelEntryA = "tei/cmd/tei-studio.parallelEntryA"
	parityParallelEntryB = "tei/cmd/tei-studio.parallelEntryB"
)

// IEP-5: equal results from different Providers do not establish path identity.
//
// Scope note: the CLI runs as a subprocess, so its entered Provider symbol cannot
// be observed from here. This case therefore demonstrates the property on the
// Studio entry alone, using two distinct Provider symbols, and records that the
// production Studio selection enters the canonical Operation. It does NOT prove
// the CLI's linked symbol; that is a separate observation (#1251 I2/I3).
func TestInspectEntriesDoNotClaimPathIdentityFromEqualResults(t *testing.T) {
	expected := parityLoadExpected(t)
	order, ok := expected.Workspace.Subjects["sales.order"]
	if !ok {
		t.Fatal("the Expected no longer describes sales.order in workspaceA")
	}
	root, err := filepath.Abs(parityReferenceRoot)
	if err != nil {
		t.Fatal(err)
	}
	subject := sourceindex.Subject{Kind: order.Kind, ID: order.ID}

	run := func(providerID, entrySymbol string, entry func(string, sourceindex.Subject) (operations.SubjectInspection, error)) (parityObserved, inspectoradapter.Invocation) {
		t.Helper()
		inspector, err := inspectoradapter.NewLocalInspectorHostSelecting(providerID, []inspectoradapter.LocalProvider{
			{ID: providerID, Entry: entrySymbol, Inspect: entry, CanonicalMessages: true},
		})
		if err != nil {
			t.Fatalf("compose %s: %v", providerID, err)
		}
		var seen inspectoradapter.Invocation
		inspector.Observe(func(invocation inspectoradapter.Invocation) { seen = invocation })
		host, err := newStudioHostWithInspector([]string{root}, inspector)
		if err != nil {
			t.Fatalf("compose the Studio Host for %s: %v", providerID, err)
		}
		return parityStudioObserve(t, &StudioBridge{host: host}, "workspace-1", subject), seen
	}

	first, firstSeen := run("inspector-distinct-a", parityParallelEntryA, parallelEntryA)
	second, secondSeen := run("inspector-distinct-b", parityParallelEntryB, parallelEntryB)
	if first.FailureCode != "" || second.FailureCode != "" {
		t.Fatalf("a distinct Provider entry failed: first=%+v second=%+v", first, second)
	}
	// The results coincide; that is exactly the trap this case exists for.
	if !reflect.DeepEqual(first, second) {
		t.Fatalf("the two Providers were expected to answer identically:\nfirst =%+v\nsecond=%+v", first, second)
	}
	// Path identity is NOT established, because the entered symbols differ. These
	// two observations are the evidence: without them, the identical results above
	// would be indistinguishable from one implementation being called twice.
	if firstSeen.ObservedEntry != parityParallelEntryA || secondSeen.ObservedEntry != parityParallelEntryB {
		t.Fatalf("the entered symbols were not the declared ones: %+v / %+v", firstSeen, secondSeen)
	}
	if !firstSeen.OriginVerified || !secondSeen.OriginVerified {
		t.Fatalf("a Provider answer was not bound to the Host root: %+v / %+v", firstSeen, secondSeen)
	}
	if firstSeen.RegisteredProviderID != "inspector-distinct-a" ||
		secondSeen.RegisteredProviderID != "inspector-distinct-b" {
		t.Fatalf("the Host bound the wrong entry per selection: %+v / %+v", firstSeen, secondSeen)
	}

	// Path identity may be claimed only when both entries enter the canonical
	// Operation, which is the production selection.
	inspector, err := inspectoradapter.NewLocalInspectorHost()
	if err != nil {
		t.Fatal(err)
	}
	var canonicalSeen inspectoradapter.Invocation
	inspector.Observe(func(invocation inspectoradapter.Invocation) { canonicalSeen = invocation })
	host, err := newStudioHostWithInspector([]string{root}, inspector)
	if err != nil {
		t.Fatal(err)
	}
	parityStudioObserve(t, &StudioBridge{host: host}, "workspace-1", subject)
	if canonicalSeen.ObservedEntry != parityCanonicalEntry {
		t.Fatalf("the production Studio entry did not enter the canonical Operation: %+v", canonicalSeen)
	}
	// The canonical entry must also be a Provider the Host actually registered,
	// not merely a symbol that happens to differ from the two above.
	if canonicalSeen.RegisteredProviderID != canonicalSeen.SelectedProviderID {
		t.Fatalf("the canonical entry was registered under another ID: %+v", canonicalSeen)
	}
}

// IEP-6: a failure on only one entry is reported as an asymmetry, never as parity.
func TestInspectEntriesReportAOneSidedFailure(t *testing.T) {
	expected := parityLoadExpected(t)
	order, ok := expected.Workspace.Subjects["sales.order"]
	if !ok {
		t.Fatal("the Expected no longer describes sales.order in workspaceA")
	}

	binary := parityBuildCLI(t)
	// The Studio Host allowlists exactly one root. Naming a second Workspace ID is
	// a Studio-only refusal, because the CLI takes its root from argv.
	bridge := parityNewBridge(t, parityReferenceRoot)
	subject := sourceindex.Subject{Kind: order.Kind, ID: order.ID}

	fromCLI := parityCLIObserve(t, binary, parityReferenceRoot, subject)
	fromStudio := parityStudioObserve(t, bridge, "workspace-2", subject)

	// The asymmetry is the point: the same request succeeds on one entry and is
	// refused on the other.
	if fromCLI.FailureCode != "" {
		t.Fatalf("the CLI entry unexpectedly failed: %+v", fromCLI)
	}
	if fromCLI.Workspace != expected.Workspace.Identity.Name {
		t.Fatalf("the successful entry read another Workspace: %+v", fromCLI)
	}
	if len(fromCLI.Attributes) == 0 {
		t.Fatalf("the successful entry reported no meaning to contrast with: %+v", fromCLI)
	}
	if fromStudio.FailureCode != "WORKSPACE_NOT_ALLOWED" {
		t.Fatalf("the Studio refusal was missing or reclassified: %+v", fromStudio)
	}
}

// Same-unknown-ID diagnostics: both entries classify a missing Subject the same
// way, which is the diagnostics axis of the contract.
func TestInspectEntriesAgreeOnAMissingSubjectCode(t *testing.T) {
	binary := parityBuildCLI(t)
	bridge := parityNewBridge(t, parityReferenceRoot)
	subject := sourceindex.Subject{Kind: "semantic-data", ID: "sales.absent_entry_parity_subject"}

	fromCLI := parityCLIObserve(t, binary, parityReferenceRoot, subject)
	fromStudio := parityStudioObserve(t, bridge, "workspace-1", subject)
	if fromCLI.FailureCode != "SUBJECT_NOT_FOUND" || fromStudio.FailureCode != "SUBJECT_NOT_FOUND" {
		t.Fatalf("a missing Subject was classified differently: cli=%+v studio=%+v", fromCLI, fromStudio)
	}
	if !reflect.DeepEqual(fromCLI, fromStudio) {
		t.Fatalf("a missing Subject disagreed between entries:\nCLI   =%+v\nStudio=%+v", fromCLI, fromStudio)
	}
}

// copyWorkspaceTree copies a Workspace so a test can edit managed Source without
// touching the committed fixture.
func copyWorkspaceTree(source, destination string) error {
	entries, err := os.ReadDir(source)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		from := filepath.Join(source, entry.Name())
		to := filepath.Join(destination, entry.Name())
		if entry.IsDir() {
			if err := os.MkdirAll(to, 0o755); err != nil {
				return err
			}
			if err := copyWorkspaceTree(from, to); err != nil {
				return err
			}
			continue
		}
		content, err := os.ReadFile(from)
		if err != nil {
			return err
		}
		if err := os.WriteFile(to, content, 0o644); err != nil {
			return err
		}
	}
	return nil
}
