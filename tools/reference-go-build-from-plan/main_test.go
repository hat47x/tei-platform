package main

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"tei/internal/buildintent"
	"tei/internal/plugin"
)

// These tests pin §4.39 stage 1: the builder reads the plan, checks that the
// sources it names are the sources on disk, and reports what it read, hashed and
// ran. The plan's digests are computed with the same resolver the entry uses, so
// a disagreement means the tool and the entry disagree — which is the failure this
// tool exists to surface.

func repoRoot(t *testing.T) string {
	t.Helper()
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	return root
}

// planFor builds a plan that names one source, with the digest the resolver
// produces for it.
func planFor(t *testing.T, root, locator, entrypoint, outputKind string) plugin.BuildPlan {
	t.Helper()
	resolved, err := (buildintent.DirSourceResolver{Root: root}).Resolve(buildintent.IntentSource{
		Kind: "go-package", ID: "tei", Ref: locator,
	})
	if err != nil {
		t.Fatalf("resolve %s: %v", locator, err)
	}
	plan := plugin.BuildPlan{
		FormatVersion:               plugin.BuildPlanFormatVersion,
		Plugin:                      plugin.Identity{ID: "tei.command.probe", Version: "0.0.0"},
		PluginDefinitionFingerprint: strings.Repeat("a", 64),
		Sources: []plugin.BuildSource{{
			Kind:    "go-package",
			ID:      "tei",
			Locator: locator,
			Digest:  resolved.Source.Digest,
		}},
		Builder: plugin.BuilderIdentity{
			ID:              "tei.builder.go",
			Version:         "1.2.0",
			ContractVersion: "tei.plugin-build/v1",
			ArtifactDigest:  "sha256:" + strings.Repeat("b", 64),
		},
		Recipe:     plugin.BuildRecipe{Kind: "go-command", Entrypoint: entrypoint},
		OutputKind: plugin.BuildOutputKind(outputKind),
		Target:     "host/linux/amd64",
	}
	if err := plan.Validate(); err != nil {
		t.Fatalf("the fixture plan does not validate: %v", err)
	}
	return plan
}

func writePlan(t *testing.T, dir string, plan plugin.BuildPlan) string {
	t.Helper()
	body, err := json.Marshal(plan)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "plan.json")
	if err := os.WriteFile(path, body, 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func runTool(t *testing.T, args ...string) (report, error) {
	t.Helper()
	var out bytes.Buffer
	err := run(args, &out)
	var rep report
	if out.Len() > 0 {
		if decodeErr := json.Unmarshal(out.Bytes(), &rep); decodeErr != nil {
			t.Fatalf("decode report: %v: %s", decodeErr, out.String())
		}
	}
	return rep, err
}

// The digest the plan names must be the digest on disk. This is the check that
// makes "the artifact came from the planned sources" meaningful.
func TestRunRefusesASourceThePlanDoesNotDescribe(t *testing.T) {
	root := repoRoot(t)
	dir := t.TempDir()
	plan := planFor(t, root, "tools/reference-build-authoring-intent", "./tools/reference-build-authoring-intent", "runtime-artifact")
	plan.Sources[0].Digest = "sha256:" + strings.Repeat("c", 64)
	planPath := writePlan(t, dir, plan)

	_, err := runTool(t,
		"-plan", planPath,
		"-repo", root,
		"-artifact-out", filepath.Join(dir, "artifact"),
		"-state", filepath.Join(dir, "state"),
	)
	if err == nil {
		t.Fatal("a plan naming a different source was accepted")
	}
	if !strings.Contains(err.Error(), "the plan does not describe") && !strings.Contains(err.Error(), "the artifact the plan does not describe") {
		t.Fatalf("the refusal does not say what disagreed: %v", err)
	}
}

// An unknown key is a refusal: a plan this process did not fully understand must
// never be built from.
func TestRunRefusesAnUnknownPlanKey(t *testing.T) {
	root := repoRoot(t)
	dir := t.TempDir()
	plan := planFor(t, root, "tools/reference-build-authoring-intent", "./tools/reference-build-authoring-intent", "runtime-artifact")
	body, err := json.Marshal(plan)
	if err != nil {
		t.Fatal(err)
	}
	body = bytes.Replace(body, []byte(`"formatVersion":1`), []byte(`"formatVersion":1,"surprise":true`), 1)
	planPath := filepath.Join(dir, "plan.json")
	if err := os.WriteFile(planPath, body, 0o600); err != nil {
		t.Fatal(err)
	}

	_, runErr := runTool(t,
		"-plan", planPath,
		"-repo", root,
		"-artifact-out", filepath.Join(dir, "artifact"),
		"-state", filepath.Join(dir, "state"),
	)
	if runErr == nil {
		t.Fatal("a plan with an unknown key was accepted")
	}
	if !strings.Contains(runErr.Error(), "unknown field") {
		t.Fatalf("the refusal does not name the unknown key: %v", runErr)
	}
}

func TestRunRequiresItsInputs(t *testing.T) {
	cases := []struct {
		name string
		args []string
		want string
	}{
		{name: "no plan", args: []string{"-artifact-out", "a", "-state", "s"}, want: "-plan is required"},
		{name: "no artifact", args: []string{"-plan", "p", "-state", "s"}, want: "-artifact-out is required"},
		{name: "no state", args: []string{"-plan", "p", "-artifact-out", "a"}, want: "-state is required"},
		{name: "positional", args: []string{"-plan", "p", "-artifact-out", "a", "-state", "s", "extra"}, want: "positional"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := runTool(t, tc.args...)
			if err == nil {
				t.Fatalf("the tool accepted %v", tc.args)
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error = %q, want it to mention %q", err, tc.want)
			}
		})
	}
}

// The whole point of the tool: a real build from a plan, reporting the plan
// fingerprint, the artifact digest and a lock it formed and validated itself.
func TestRunBuildsFromThePlanAndReportsWhatItDid(t *testing.T) {
	if _, err := exec.LookPath("go"); err != nil {
		t.Skipf("Go toolchain unavailable: %v", err)
	}
	root := repoRoot(t)
	dir := t.TempDir()
	plan := planFor(t, root, "tools/reference-build-authoring-intent", "./tools/reference-build-authoring-intent", "runtime-artifact")
	planPath := writePlan(t, dir, plan)
	artifact := filepath.Join(dir, "artifact")
	state := filepath.Join(dir, "state")

	rep, err := runTool(t,
		"-plan", planPath,
		"-repo", root,
		"-artifact-out", artifact,
		"-state", state,
	)
	if err != nil {
		t.Fatalf("build failed: %v", err)
	}
	if !rep.BuildOK {
		t.Fatalf("buildSucceeded = false: %s", rep.BuildError)
	}
	wantFingerprint, err := plugin.BuildPlanFingerprint(plan)
	if err != nil {
		t.Fatal(err)
	}
	if rep.PlanFingerprint != wantFingerprint {
		t.Fatalf("planFingerprint = %q, want %q", rep.PlanFingerprint, wantFingerprint)
	}
	if rep.ArtifactDigest == "" || !strings.HasPrefix(rep.ArtifactDigest, "sha256:") {
		t.Fatalf("artifactDigest = %q", rep.ArtifactDigest)
	}
	if rep.StateDir != state {
		t.Fatalf("stateDir = %q, want %q", rep.StateDir, state)
	}
	if rep.Lock == nil {
		t.Fatal("no lock was reported")
	}
	if !rep.Lock.Validated || rep.Lock.LockFingerprint == "" {
		t.Fatalf("the lock was not formed and validated: %+v", rep.Lock)
	}
	if rep.Lock.PlanFingerprint != wantFingerprint || rep.Lock.ArtifactDigest != rep.ArtifactDigest {
		t.Fatalf("the lock does not carry the values this run holds: %+v", rep.Lock)
	}
	if len(rep.Sources) != 1 || !rep.Sources[0].Matched {
		t.Fatalf("the checked sources are not reported as matched: %+v", rep.Sources)
	}
	if _, err := os.Stat(artifact); err != nil {
		t.Fatalf("the artifact was not written: %v", err)
	}
}
