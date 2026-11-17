package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Sources of a scope entry. They are kept apart because they are different
// kinds of statement: a plan declares before the build, a builder reports
// after it, and a caller names things the entry was never told about.
const (
	sourcePlan           = "plan"
	sourceEntryReport    = "entry-report"
	sourceBuilderReport  = "builder-report"
	sourceDerivedFromRun = "derived-from-report"
	sourceCaller         = "caller"

	roleToolchainNamed    = "toolchain"
	roleToolchainReported = "toolchain-reported-not-declared"
)

// defaultOSInterface is what a Go program and the dynamic loader touch on
// linux regardless of what the build asked for. /etc/localtime and /tmp are
// deliberately absent: the time zone and the shared temporary directory are
// ambient inputs, and a build that reads them should show it.
//
// Paths in an observation are the ones the kernel resolved, so a request such
// as /proc/self/root/etc/passwd arrives here as /etc/passwd and is not covered.
var defaultOSInterface = []string{
	"/dev/null",
	"/etc/ld.so.cache",
	"/lib/x86_64-linux-gnu/",
	"/usr/lib/x86_64-linux-gnu/",
	"/proc/self/cgroup",
	"/proc/self/maps",
	"/proc/self/mountinfo",
	"/sys/devices/system/cpu/online",
	"/sys/fs/cgroup/",
	"/sys/kernel/mm/transparent_hugepage/",
}

// selfBuildReport is the part of a `tei self-build` report this tool reads.
type selfBuildReport struct {
	Workspace  string `json:"workspace"`
	SupplyRoot string `json:"supplyRoot"`
	BuildState string `json:"buildState"`
	Output     string `json:"output"`
	ChildRun   *struct {
		InvokerDigest string `json:"invokerDigest"`
		InvokerPath   string `json:"invokerPath"`
		Invoked       *struct {
			Executable string `json:"executable"`
		} `json:"invoked"`
	} `json:"childRun"`
	ChildIsolation *struct {
		Root string `json:"root"`
	} `json:"childIsolation"`
	BuilderBinding *struct {
		DeclaredBuilderDigest string `json:"declaredBuilderDigest"`
	} `json:"builderBinding"`
	ToolchainObservation *struct {
		Reported *struct {
			GoRoot string `json:"goRoot"`
		} `json:"reported"`
		NamedByPlan bool `json:"namedByPlan"`
		MatchesPlan bool `json:"matchesPlan"`
	} `json:"toolchainObservation"`
	PlanHandover *struct {
		State    string `json:"state"`
		PlanPath string `json:"planPath"`
	} `json:"planHandover"`
	ArtifactReproducibility *struct {
		SecondStateDir string `json:"secondStateDir"`
	} `json:"artifactReproducibility"`
}

type repeated []string

func (r *repeated) String() string     { return strings.Join(*r, ",") }
func (r *repeated) Set(v string) error { *r = append(*r, v); return nil }

func scopeCommand(args []string) int {
	flags := flag.NewFlagSet("scope", flag.ContinueOnError)
	reportPath := flags.String("self-build-report", "", "report printed by `tei self-build`")
	out := flags.String("out", "", "file to write the scope to (default: stdout)")
	var reads, writes repeated
	flags.Var(&reads, "read", "role=path the caller names as readable (repeatable)")
	flags.Var(&writes, "write", "role=path the caller names as writable (repeatable)")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if *reportPath == "" {
		fmt.Fprintln(os.Stderr, "usage: reference-build-process-observation scope -self-build-report report.json [-read role=path] [-write role=path] [-out scope.json]")
		return 2
	}
	var report selfBuildReport
	if err := readJSON(*reportPath, &report); err != nil {
		fmt.Fprintln(os.Stderr, "read the report:", err)
		return 2
	}
	scope, err := ScopeFromSelfBuild(report)
	if err != nil {
		fmt.Fprintln(os.Stderr, "scope:", err)
		return 2
	}
	for _, list := range []struct {
		values []string
		write  bool
	}{{reads, false}, {writes, true}} {
		for _, value := range list.values {
			role, path, ok := strings.Cut(value, "=")
			if !ok || role == "" || path == "" {
				fmt.Fprintf(os.Stderr, "expected role=path, got %q\n", value)
				return 2
			}
			scope.Paths = append(scope.Paths, ScopePath{Role: role, Root: path, Write: list.write, Source: sourceCaller})
		}
	}
	encoded, err := json.MarshalIndent(scope, "", "  ")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 2
	}
	encoded = append(encoded, '\n')
	if *out == "" {
		os.Stdout.Write(encoded)
		return 0
	}
	if err := os.WriteFile(*out, encoded, 0o644); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 2
	}
	return 0
}

// ScopeFromSelfBuild reads the declared side out of a self-build report. It
// refuses a report that does not name the entry or the builder: a scope with
// no declared image would cover nothing and say so only by failing later.
func ScopeFromSelfBuild(report selfBuildReport) (Scope, error) {
	scope := Scope{Schema: scopeSchema, OSInterface: append([]string(nil), defaultOSInterface...)}
	if report.ChildRun == nil || report.ChildRun.InvokerDigest == "" {
		return Scope{}, fmt.Errorf("the report does not name the entry image (childRun.invokerDigest)")
	}
	scope.Images = append(scope.Images, ScopeImage{Role: "entry", Digest: report.ChildRun.InvokerDigest, Source: sourceEntryReport})
	if report.BuilderBinding == nil || report.BuilderBinding.DeclaredBuilderDigest == "" {
		return Scope{}, fmt.Errorf("the report does not carry the builder the plan declares (builderBinding.declaredBuilderDigest)")
	}
	scope.Images = append(scope.Images, ScopeImage{Role: "builder", Digest: report.BuilderBinding.DeclaredBuilderDigest, Source: sourcePlan})

	if t := report.ToolchainObservation; t != nil && t.Reported != nil && t.Reported.GoRoot != "" {
		root := ScopeRoot{Role: roleToolchainReported, Root: t.Reported.GoRoot, Source: sourceBuilderReport}
		// The plan names a toolchain by digest, not by path. The path is the
		// builder's; it counts as declared only when the entry found the
		// reported toolchain to be the one the plan names.
		if t.NamedByPlan && t.MatchesPlan {
			root.Role, root.Source = roleToolchainNamed, sourcePlan
		}
		scope.ImageRoots = append(scope.ImageRoots, root)
	}

	add := func(role, root string, write bool, owner int, source string) {
		if root != "" {
			scope.Paths = append(scope.Paths, ScopePath{Role: role, Root: root, Write: write, OwnerStep: owner, Source: source})
		}
	}
	add("entry-file", report.ChildRun.InvokerPath, false, 0, sourceEntryReport)
	if report.ChildRun.Invoked != nil {
		add("builder-file", report.ChildRun.Invoked.Executable, false, 0, sourceEntryReport)
	}
	add("workspace", report.Workspace, false, 0, sourceEntryReport)
	add("supply", report.SupplyRoot, false, 0, sourceEntryReport)
	add("state", report.BuildState, true, 1, sourceEntryReport)
	add("output", report.Output, true, 1, sourceEntryReport)
	if report.ChildIsolation != nil {
		add("isolated-tree", report.ChildIsolation.Root, true, 1, sourceBuilderReport)
	}
	if h := report.PlanHandover; h != nil && h.State == "handed-over" {
		// The entry wrote this file itself and pointed the child at it.
		// It sits alone in a directory the entry made for it and removes.
		add("plan", filepath.ToSlash(filepath.Dir(h.PlanPath)), true, 0, sourceDerivedFromRun)
	}
	if r := report.ArtifactReproducibility; r != nil && r.SecondStateDir != "" {
		add("state", r.SecondStateDir, true, 2, sourceEntryReport)
		// The report names the second state directory but not the second output;
		// the entry puts both in one scratch directory.
		add("rebuild-scratch", filepath.ToSlash(filepath.Dir(r.SecondStateDir)), true, 2, sourceDerivedFromRun)
	}
	return scope, nil
}
