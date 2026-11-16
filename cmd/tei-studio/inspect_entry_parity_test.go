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
// and diagnostics are matched. The inner canonical attribute order is also not
// compared, because the projection emits sorted IDs.
//
// Expectations are restated here from
// testdata/inspect-entry-parity/independent-expected.candidate.json, which was
// derived from the Workspace Source bytes and the .tei declarations rather than
// read back from any Inspector answer.

const (
	parityReferenceRoot    = "../../reference/workspace"
	parityStandardCRUDRoot = "../../reference/standard-crud/workspace"

	parityReferenceName     = "tei_reference"
	parityReferenceManifest = "sha256:8e367b05de26a715ced16e384d273c983ac612dd2ff980d4310886d4f471af74"

	parityReferenceOrderPath  = "semantic/sales/order.tei"
	parityReferenceOrderFP    = "sha256:f2063418b893a60a9f948edbb2b23ec821f7a15ed8e5863c36e1b72abade1e33"
	parityReferenceOrderStart = 124

	parityStandardName     = "standard_crud"
	parityStandardManifest = "sha256:f4e74f4da36dfa41eeeefd078a3a13fecfd82bb419eb8b331a81a18ac5535bb9"

	parityStandardOrderPath  = "semantic/sales/order.tei"
	parityStandardOrderFP    = "sha256:f6a7b246e42118a0cf6fdc2392fa1defb9f7546f8cc766806da31dc82972a358"
	parityStandardOrderStart = 56

	parityAttrCustomer = "sales.order.customer_id"
	parityAttrForeign  = "customer.customer.customer_id"

	// parityCanonicalEntry is the linked symbol both production entries select.
	parityCanonicalEntry = "tei/internal/operations.InspectSubject"
)

// parityReferenceOrderAttributes is the attribute SET declared in
// reference/workspace/semantic/sales/order.tei.
var parityReferenceOrderAttributes = []string{
	"sales.order.customer_id",
	"sales.order.external_id",
	"sales.order.order_id",
	"sales.order.ordered_at",
	"sales.order.status",
	"sales.order.total",
}

// parityStandardOrderAttributes is the attribute SET declared in
// reference/standard-crud/workspace/semantic/sales/order.tei.
var parityStandardOrderAttributes = []string{
	"sales.order.order_id",
	"sales.order.status",
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

func parityCanonicalFields(canonical json.RawMessage) (string, []string, []string, error) {
	var body struct {
		ID         string `json:"id"`
		Attributes []struct {
			ID string `json:"id"`
		} `json:"attributes"`
		Relations []struct {
			ID string `json:"id"`
		} `json:"relations"`
	}
	if err := json.Unmarshal(canonical, &body); err != nil {
		return "", nil, nil, err
	}
	var attributes, relations []string
	for _, attribute := range body.Attributes {
		attributes = append(attributes, attribute.ID)
	}
	for _, relation := range body.Relations {
		relations = append(relations, relation.ID)
	}
	return body.ID, attributes, relations, nil
}

// parityFromOperation normalizes a SubjectInspection that either entry produced.
func parityFromOperation(t *testing.T, inspection operations.SubjectInspection) parityObserved {
	t.Helper()
	id, attributes, relations, err := parityCanonicalFields(inspection.Canonical)
	if err != nil {
		t.Fatalf("canonical is not a declaration object: %v", err)
	}
	observed := parityObserved{
		Kind:        inspection.Subject.Kind,
		ID:          inspection.Subject.ID,
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
		t.Skipf("the tei CLI cannot be built in this environment: %v\n%s", err, output)
	}
	return binary
}

// TestInspectParityExpectationsMatchTheSourceBytes guards the hand-written
// expectations against transcription drift. A hash typed by hand from a tool's
// output once disagreed with the file it described in one character; this test
// recomputes each recorded value from the pinned bytes so the constants above
// can only be wrong if the fixture itself changed.
func TestInspectParityExpectationsMatchTheSourceBytes(t *testing.T) {
	for _, expected := range []struct {
		name     string
		path     string
		want     string
		recorded string
	}{
		{"reference order artifact", filepath.Join(parityReferenceRoot, "semantic", "sales", "order.tei"), parityReferenceOrderFP, "parityReferenceOrderFP"},
		{"reference manifest", filepath.Join(parityReferenceRoot, "tei.workspace.yaml"), parityReferenceManifest, "parityReferenceManifest"},
		{"standard-crud order artifact", filepath.Join(parityStandardCRUDRoot, "semantic", "sales", "order.tei"), parityStandardOrderFP, "parityStandardOrderFP"},
		{"standard-crud manifest", filepath.Join(parityStandardCRUDRoot, "tei.workspace.yaml"), parityStandardManifest, "parityStandardManifest"},
	} {
		t.Run(expected.name, func(t *testing.T) {
			content, err := os.ReadFile(expected.path)
			if err != nil {
				t.Fatalf("read the pinned Source: %v", err)
			}
			if got := sourceindex.Fingerprint(content); got != expected.want {
				t.Fatalf("%s (%s) records %q but the file is %q", expected.recorded, expected.path, expected.want, got)
			}
		})
	}
	// The offsets name the declaration start, so they must land on the declared
	// identifier inside the file rather than on any byte that happens to match.
	for _, offset := range []struct {
		path   string
		start  int
		prefix string
	}{
		{filepath.Join(parityReferenceRoot, "semantic", "sales", "order.tei"), parityReferenceOrderStart, "Order {"},
		{filepath.Join(parityStandardCRUDRoot, "semantic", "sales", "order.tei"), parityStandardOrderStart, "Order {"},
	} {
		content, err := os.ReadFile(offset.path)
		if err != nil {
			t.Fatal(err)
		}
		if offset.start >= len(content) {
			t.Fatalf("%s: offset %d is beyond the file", offset.path, offset.start)
		}
		if !bytes.HasPrefix(content[offset.start:], []byte(offset.prefix)) {
			t.Fatalf("%s: offset %d does not start %q", offset.path, offset.start, offset.prefix)
		}
	}
}

// IEP-1: for one target ID and one Source revision, both entries report the same
// semantic result.
func TestInspectEntriesAgreeOnOneSubjectAtOneSourceRevision(t *testing.T) {
	binary := parityBuildCLI(t)
	bridge := parityNewBridge(t, parityReferenceRoot)
	subject := sourceindex.Subject{Kind: "semantic-data", ID: "sales.order"}

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
		if !paritySameSet(got.Attributes, parityReferenceOrderAttributes) {
			t.Fatalf("%s attribute meaning drifted: got %v want %v",
				entry.name, got.Attributes, parityReferenceOrderAttributes)
		}
		if want := []string{"sales.order.customer", "sales.order.lines"}; !paritySameSet(got.Relations, want) {
			t.Fatalf("%s relation meaning drifted: got %v want %v", entry.name, got.Relations, want)
		}
		if got.Workspace != parityReferenceName || got.ManifestSHA != parityReferenceManifest {
			t.Fatalf("%s read another Workspace: %+v", entry.name, got)
		}
		if got.SourcePath != parityReferenceOrderPath {
			t.Fatalf("%s named another Source artifact: %+v", entry.name, got)
		}
		if got.SourceFP != parityReferenceOrderFP {
			t.Fatalf("%s reported Source fingerprint %q, expected %q",
				entry.name, got.SourceFP, parityReferenceOrderFP)
		}
		if got.SourceRole != string(sourceindex.RoleDefinition) {
			t.Fatalf("%s reported Source role %q: %+v", entry.name, got.SourceRole, got)
		}
		if got.Offset != parityReferenceOrderStart {
			t.Fatalf("%s reported Source offset %d, expected %d", entry.name, got.Offset, parityReferenceOrderStart)
		}
		if !reflect.DeepEqual(got.Limits, operations.InspectionAuthorityLimits()) {
			t.Fatalf("%s changed the authority limits: %+v", entry.name, got)
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
	binary := parityBuildCLI(t)
	bridge := parityNewBridge(t, parityReferenceRoot)

	observations := map[string]parityObserved{}
	for _, id := range []string{parityAttrCustomer, parityAttrForeign} {
		subject := sourceindex.Subject{Kind: "semantic-attribute", ID: id}
		fromCLI := parityCLIObserve(t, binary, parityReferenceRoot, subject)
		fromStudio := parityStudioObserve(t, bridge, "workspace-1", subject)
		if fromCLI.FailureCode != "" || fromStudio.FailureCode != "" {
			t.Fatalf("%s was not inspectable through both entries: cli=%+v studio=%+v", id, fromCLI, fromStudio)
		}
		if !reflect.DeepEqual(fromCLI, fromStudio) {
			t.Fatalf("%s disagreed between entries:\nCLI   =%+v\nStudio=%+v", id, fromCLI, fromStudio)
		}
		if fromCLI.ID != id || fromCLI.CanonicalID != id {
			t.Fatalf("%s was answered as another subject: %+v", id, fromCLI)
		}
		observations[id] = fromCLI
	}

	customer, foreign := observations[parityAttrCustomer], observations[parityAttrForeign]
	if customer.SourcePath == foreign.SourcePath {
		t.Fatalf("two Attributes sharing a symbol reported one Source: %+v / %+v", customer, foreign)
	}
	if customer.ID == foreign.ID || customer.SourceFP == foreign.SourceFP || customer.Offset == foreign.Offset {
		t.Fatalf("two Attributes sharing a symbol were merged: %+v / %+v", customer, foreign)
	}
}

// IEP-3: one canonical ID declared by two Workspaces is kept apart.
func TestInspectEntriesKeepSameIDWorkspacesApart(t *testing.T) {
	binary := parityBuildCLI(t)
	bridge := parityNewBridge(t, parityReferenceRoot, parityStandardCRUDRoot)
	subject := sourceindex.Subject{Kind: "semantic-data", ID: "sales.order"}

	targets := []struct {
		name          string
		root          string
		workspace     string
		workspaceName string
		manifest      string
		attributes    []string
		sourcePath    string
		sourceFP      string
		offset        int
	}{
		{
			name: "reference", root: parityReferenceRoot, workspace: "workspace-1",
			workspaceName: parityReferenceName, manifest: parityReferenceManifest,
			attributes: parityReferenceOrderAttributes,
			sourcePath: parityReferenceOrderPath, sourceFP: parityReferenceOrderFP,
			offset: parityReferenceOrderStart,
		},
		{
			name: "standard-crud", root: parityStandardCRUDRoot, workspace: "workspace-2",
			workspaceName: parityStandardName, manifest: parityStandardManifest,
			attributes: parityStandardOrderAttributes,
			sourcePath: parityStandardOrderPath, sourceFP: parityStandardOrderFP,
			offset: parityStandardOrderStart,
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
		if fromCLI.Workspace != target.workspaceName || fromCLI.ManifestSHA != target.manifest {
			t.Fatalf("%s reported another Workspace identity: %+v", target.name, fromCLI)
		}
		if !paritySameSet(fromCLI.Attributes, target.attributes) {
			t.Fatalf("%s meaning drifted: got %v want %v", target.name, fromCLI.Attributes, target.attributes)
		}
		if fromCLI.SourcePath != target.sourcePath || fromCLI.SourceFP != target.sourceFP ||
			fromCLI.Offset != target.offset {
			t.Fatalf("%s Source evidence drifted: %+v", target.name, fromCLI)
		}
		observed[target.name] = fromCLI
	}

	a, b := observed["reference"], observed["standard-crud"]
	if a.Workspace == b.Workspace || a.ManifestSHA == b.ManifestSHA ||
		a.SourceFP == b.SourceFP || paritySameSet(a.Attributes, b.Attributes) {
		t.Fatalf("two Workspaces declaring sales.order were treated as one subject:\nA=%+v\nB=%+v", a, b)
	}
}

// IEP-4: after a Source update, an earlier result is not treated as current.
func TestInspectEntriesDoNotServeAStaleResultAfterSourceUpdate(t *testing.T) {
	binary := parityBuildCLI(t)
	editedRoot := t.TempDir()
	// Copy the public Workspace; the committed fixture is never written to.
	if err := copyWorkspaceTree(parityReferenceRoot, editedRoot); err != nil {
		t.Fatalf("copy the reference Workspace: %v", err)
	}
	bridge := parityNewBridge(t, editedRoot)
	subject := sourceindex.Subject{Kind: "semantic-data", ID: "sales.order"}

	beforeCLI := parityCLIObserve(t, binary, editedRoot, subject)
	beforeStudio := parityStudioObserve(t, bridge, "workspace-1", subject)
	if beforeCLI.FailureCode != "" || beforeStudio.FailureCode != "" {
		t.Fatalf("pre-edit inspection failed: cli=%+v studio=%+v", beforeCLI, beforeStudio)
	}
	if !reflect.DeepEqual(beforeCLI, beforeStudio) {
		t.Fatalf("the entries disagreed before the edit:\nCLI   =%+v\nStudio=%+v", beforeCLI, beforeStudio)
	}

	sourcePath := filepath.Join(editedRoot, filepath.FromSlash(parityReferenceOrderPath))
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

	wantAttributes := append(append([]string(nil), parityReferenceOrderAttributes...), "sales.order.note")
	for _, entry := range []struct {
		name  string
		value parityObserved
	}{{"cli", afterCLI}, {"studio", afterStudio}} {
		got := entry.value
		if !paritySameSet(got.Attributes, wantAttributes) {
			t.Fatalf("%s did not read the edited Source: got %v want %v", entry.name, got.Attributes, wantAttributes)
		}
		if got.SourceFP == parityReferenceOrderFP || got.SourceFP == beforeCLI.SourceFP {
			t.Fatalf("%s served a pre-edit Source as current: %+v", entry.name, got)
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
func TestInspectEntriesDoNotClaimPathIdentityFromEqualResults(t *testing.T) {
	root, err := filepath.Abs(parityReferenceRoot)
	if err != nil {
		t.Fatal(err)
	}
	subject := sourceindex.Subject{Kind: "semantic-data", ID: "sales.order"}

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
	// Path identity is NOT established, because the entered symbols differ.
	if firstSeen.ObservedEntry != parityParallelEntryA || secondSeen.ObservedEntry != parityParallelEntryB {
		t.Fatalf("the entered symbols were not the declared ones: %+v / %+v", firstSeen, secondSeen)
	}
	if firstSeen.ObservedEntry == secondSeen.ObservedEntry {
		t.Fatal("the two Providers share one symbol; the fixture no longer distinguishes them")
	}
	if !firstSeen.OriginVerified || !secondSeen.OriginVerified {
		t.Fatalf("a Provider answer was not bound to the Host root: %+v / %+v", firstSeen, secondSeen)
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
	if canonicalSeen.ObservedEntry == firstSeen.ObservedEntry ||
		canonicalSeen.ObservedEntry == secondSeen.ObservedEntry {
		t.Fatalf("the canonical Operation shares a symbol with a distinct Provider: %+v", canonicalSeen)
	}
}

// IEP-6: a failure on only one entry is reported as an asymmetry, never as parity.
func TestInspectEntriesReportAOneSidedFailure(t *testing.T) {
	binary := parityBuildCLI(t)
	// The Studio Host allowlists exactly one root. Naming a second Workspace ID is
	// a Studio-only refusal, because the CLI takes its root from argv.
	bridge := parityNewBridge(t, parityReferenceRoot)
	subject := sourceindex.Subject{Kind: "semantic-data", ID: "sales.order"}

	fromCLI := parityCLIObserve(t, binary, parityReferenceRoot, subject)
	fromStudio := parityStudioObserve(t, bridge, "workspace-2", subject)

	if fromCLI.FailureCode != "" {
		t.Fatalf("the CLI entry unexpectedly failed: %+v", fromCLI)
	}
	if fromStudio.FailureCode != "WORKSPACE_NOT_ALLOWED" {
		t.Fatalf("the Studio refusal was missing or reclassified: %+v", fromStudio)
	}
	if reflect.DeepEqual(fromCLI, fromStudio) {
		t.Fatal("a one-sided failure was reported as parity")
	}
	// This is a Scope difference, not a meaning difference: the requested target
	// must survive the refusal.
	if fromStudio.Kind != subject.Kind || fromStudio.ID != subject.ID {
		t.Fatalf("the refusal lost the requested target: %+v", fromStudio)
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
