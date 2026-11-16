package main

import (
	"encoding/json"
	"strings"
	"testing"
)

// The expectations in this file are written by hand from the rules in the
// README. None of them is produced by running Reconcile.

const (
	entryDigest   = "sha256:e0"
	builderDigest = "sha256:b0"
	goDigest      = "sha256:90"
)

func testScope() Scope {
	return Scope{
		Schema: scopeSchema,
		Images: []ScopeImage{
			{Role: "entry", Digest: entryDigest},
			{Role: "builder", Digest: builderDigest},
		},
		ImageRoots: []ScopeRoot{{Role: roleToolchainNamed, Root: "/toolchain"}},
		Paths: []ScopePath{
			{Role: "source", Root: "/repo"},
			{Role: "run", Root: "/run", Write: true},
			{Role: "state", Root: "/run/state1", Write: true, OwnerStep: 1},
			{Role: "state", Root: "/run/state2", Write: true, OwnerStep: 2},
		},
		OSInterface: []string{"/dev/null", "/proc/self/"},
	}
}

// A clean run: the entry starts the builder twice, each builder starts the
// toolchain, and every open stays in its own territory.
func cleanRecord() Record {
	return Record{
		Schema:      observationSchema,
		FilesTraced: true,
		Execs: []Exec{
			{Seq: 0, ParentSeq: -1, Step: 0, Image: "/bin/tei", ImageDigest: entryDigest},
			{Seq: 1, ParentSeq: 0, Step: 1, Image: "/run/builder", ImageDigest: builderDigest},
			{Seq: 2, ParentSeq: 1, Step: 1, Image: "/toolchain/bin/go", ImageDigest: goDigest},
			{Seq: 3, ParentSeq: 0, Step: 3, Image: "/run/builder", ImageDigest: builderDigest},
			{Seq: 4, ParentSeq: 3, Step: 3, Image: "/toolchain/bin/go", ImageDigest: goDigest},
		},
		Files: []FileUse{
			{Path: "/repo/a.go", Image: "/bin/tei", Step: 0, Read: 1},
			{Path: "/repo/a.go", Image: "/toolchain/bin/go", Step: 1, Read: 2},
			{Path: "/repo/a.go", Image: "/toolchain/bin/go", Step: 3, Read: 2},
			{Path: "/run/state1/x", Image: "/toolchain/bin/go", Step: 1, Read: 1, Write: 1},
			{Path: "/run/state2/x", Image: "/toolchain/bin/go", Step: 3, Read: 1, Write: 1},
			{Path: "/run/state1/x", Image: "/bin/tei", Step: 0, Read: 1},
			{Path: "/toolchain/src/fmt/print.go", Image: "/toolchain/bin/go", Step: 1, Read: 1},
			{Path: "/proc/self/maps", Image: "/bin/tei", Step: 0, Read: 1},
			{Path: "/etc/passwd", Image: "/toolchain/bin/go", Step: 1, Failed: 3},
		},
		NotObserved: []string{"bytes read"},
	}
}

func mustReconcile(t *testing.T, record Record, scope Scope) Ledger {
	t.Helper()
	ledger, err := Reconcile(record, scope)
	if err != nil {
		t.Fatal(err)
	}
	return ledger
}

func TestCleanRunHoldsAndCountsByRole(t *testing.T) {
	ledger := mustReconcile(t, cleanRecord(), testScope())
	if ledger.Absence.State != absenceHolds {
		t.Fatalf("absence = %q, reasons %v", ledger.Absence.State, ledger.Absence.Reasons)
	}
	wantExecs := map[string]int{"entry": 1, "builder": 2, roleToolchainNamed: 2}
	for role, want := range wantExecs {
		if got := ledger.Execs.ByRole[role]; got != want {
			t.Errorf("execs[%s] = %d, want %d", role, got, want)
		}
	}
	if ledger.Execs.Total != 5 || ledger.Execs.UndeclaredTotal != 0 {
		t.Errorf("execs total %d undeclared %d", ledger.Execs.Total, ledger.Execs.UndeclaredTotal)
	}
	if len(ledger.Steps) != 2 || ledger.Steps[0].Seq != 1 || ledger.Steps[1].Seq != 3 {
		t.Fatalf("steps = %+v", ledger.Steps)
	}
	for _, step := range ledger.Steps {
		if step.Role != "builder" || step.Execs != 2 {
			t.Errorf("step %+v, want role builder with 2 execs", step)
		}
	}
	// /repo/a.go is one path opened five times for reading by three users.
	if got := ledger.Files.ByRole["source"]; got != (RoleUse{Paths: 1, Read: 5}) {
		t.Errorf("source use = %+v", got)
	}
	if got := ledger.Files.ByRole["state"]; got != (RoleUse{Paths: 2, Read: 3, Write: 2}) {
		t.Errorf("state use = %+v", got)
	}
	// The failed open of /etc/passwd is a probe, not an input.
	if ledger.Files.ProbesOutsideScope != 3 || ledger.Files.UndeclaredTotal != 0 {
		t.Errorf("probes %d undeclared %d", ledger.Files.ProbesOutsideScope, ledger.Files.UndeclaredTotal)
	}
	if len(ledger.NotObserved) != 1 {
		t.Errorf("notObserved was not carried over: %v", ledger.NotObserved)
	}
}

func assertNotProven(t *testing.T, ledger Ledger, fragment string) {
	t.Helper()
	if ledger.Absence.State != absenceNotProven {
		t.Fatalf("absence = %q, want %q", ledger.Absence.State, absenceNotProven)
	}
	for _, reason := range ledger.Absence.Reasons {
		if strings.Contains(reason, fragment) {
			return
		}
	}
	t.Fatalf("no reason mentions %q: %v", fragment, ledger.Absence.Reasons)
}

// A step the declaration never mentioned ran inside the tree.
func TestUndeclaredProgramIsReported(t *testing.T) {
	record := cleanRecord()
	record.Execs = append(record.Execs, Exec{Seq: 5, ParentSeq: 4, Step: 3, Image: "/usr/bin/sign", ImageDigest: "sha256:51"})
	ledger := mustReconcile(t, record, testScope())
	assertNotProven(t, ledger, "executed program")
	if len(ledger.Execs.Undeclared) != 1 || ledger.Execs.Undeclared[0].Image != "/usr/bin/sign" || ledger.Execs.Undeclared[0].Step != 2 {
		t.Fatalf("undeclared = %+v", ledger.Execs.Undeclared)
	}
}

// The declared builder path was started, but other bytes ran: the digest
// decides, not the path.
func TestSamePathOtherBytesIsUndeclared(t *testing.T) {
	record := cleanRecord()
	record.Execs[3].ImageDigest = "sha256:b1"
	ledger := mustReconcile(t, record, testScope())
	assertNotProven(t, ledger, "executed program")
	if ledger.Steps[1].Role != roleUndeclared {
		t.Fatalf("second step role = %q", ledger.Steps[1].Role)
	}
}

// A launcher that hands over to a compiler outside the named toolchain.
func TestProgramOutsideTheNamedToolchainIsUndeclared(t *testing.T) {
	record := cleanRecord()
	record.Execs[4].Image = "/other-toolchain/bin/go"
	ledger := mustReconcile(t, record, testScope())
	assertNotProven(t, ledger, "executed program")
}

// The second build read the first build's state: equal artifacts would then
// not be two independent builds.
func TestReadingAnotherStepsStateIsCrossStep(t *testing.T) {
	record := cleanRecord()
	record.Files = append(record.Files, FileUse{Path: "/run/state1/x", Image: "/toolchain/bin/go", Step: 3, Read: 1})
	ledger := mustReconcile(t, record, testScope())
	assertNotProven(t, ledger, "owned by another step")
	if len(ledger.Files.CrossStep) != 1 || ledger.Files.CrossStep[0].Step != 2 || ledger.Files.CrossStep[0].Role != "state" {
		t.Fatalf("crossStep = %+v", ledger.Files.CrossStep)
	}
}

// The command may read what its steps own, but writing there is interference.
func TestCommandWritingIntoAStepsStateIsCrossStep(t *testing.T) {
	record := cleanRecord()
	record.Files = append(record.Files, FileUse{Path: "/run/state2/y", Image: "/bin/tei", Step: 0, Write: 1})
	ledger := mustReconcile(t, record, testScope())
	assertNotProven(t, ledger, "owned by another step")
}

func TestOpenOutsideTheScopeIsUndeclared(t *testing.T) {
	record := cleanRecord()
	record.Files = append(record.Files, FileUse{Path: "/etc/localtime", Image: "/toolchain/bin/go", Step: 1, Read: 2})
	ledger := mustReconcile(t, record, testScope())
	assertNotProven(t, ledger, "opened path")
	if ledger.Files.Undeclared[0].Path != "/etc/localtime" {
		t.Fatalf("undeclared = %+v", ledger.Files.Undeclared)
	}
}

func TestWriteUnderAReadRootIsReported(t *testing.T) {
	record := cleanRecord()
	record.Files = append(record.Files, FileUse{Path: "/repo/generated.go", Image: "/run/builder", Step: 1, Write: 1})
	ledger := mustReconcile(t, record, testScope())
	assertNotProven(t, ledger, "declared for reading")
}

// Without path tracing nothing can be said about what was not opened.
func TestExecOnlyObservationNeverHolds(t *testing.T) {
	record := cleanRecord()
	record.FilesTraced = false
	record.Files = nil
	ledger := mustReconcile(t, record, testScope())
	assertNotProven(t, ledger, "not traced")
}

func TestEmptyObservationNeverHolds(t *testing.T) {
	ledger := mustReconcile(t, Record{Schema: observationSchema, FilesTraced: true}, testScope())
	assertNotProven(t, ledger, "no exec")
}

// A path that only shares a name prefix with a root is not under it.
func TestRootMatchIsByPathComponent(t *testing.T) {
	record := cleanRecord()
	record.Files = append(record.Files, FileUse{Path: "/repository/secret", Image: "/bin/tei", Step: 0, Read: 1})
	ledger := mustReconcile(t, record, testScope())
	assertNotProven(t, ledger, "opened path")
}

func TestScopeFromSelfBuildKeepsDeclaredAndReportedApart(t *testing.T) {
	var report selfBuildReport
	raw := `{"workspace":"/w","supplyRoot":"/supply","buildState":"/r/state","output":"/r/artifact",
	 "childRun":{"invokerDigest":"sha256:e0","invokerPath":"/bin/tei","invoked":{"executable":"/r/builder"}},
	 "childIsolation":{"root":"/tmp/tree1"},
	 "builderBinding":{"declaredBuilderDigest":"sha256:b0"},
	 "toolchainObservation":{"reported":{"goRoot":"/toolchain"},"namedByPlan":false},
	 "artifactReproducibility":{"secondStateDir":"/tmp/rb/state"}}`
	if err := json.Unmarshal([]byte(raw), &report); err != nil {
		t.Fatal(err)
	}
	scope, err := ScopeFromSelfBuild(report)
	if err != nil {
		t.Fatal(err)
	}
	if scope.ImageRoots[0].Role != roleToolchainReported || scope.ImageRoots[0].Source != sourceBuilderReport {
		t.Fatalf("an unnamed toolchain was presented as declared: %+v", scope.ImageRoots[0])
	}
	owners := map[string]int{}
	for _, p := range scope.Paths {
		owners[p.Root] = p.OwnerStep
	}
	if owners["/r/state"] != 1 || owners["/tmp/rb/state"] != 2 || owners["/tmp/tree1"] != 1 {
		t.Fatalf("owners = %v", owners)
	}
	report.ToolchainObservation.NamedByPlan = true
	scope, _ = ScopeFromSelfBuild(report)
	if scope.ImageRoots[0].Role != roleToolchainNamed || scope.ImageRoots[0].Source != sourcePlan {
		t.Fatalf("a named toolchain was not presented as declared: %+v", scope.ImageRoots[0])
	}
	report.BuilderBinding = nil
	if _, err := ScopeFromSelfBuild(report); err == nil {
		t.Fatal("a report without a declared builder produced a scope")
	}
}
