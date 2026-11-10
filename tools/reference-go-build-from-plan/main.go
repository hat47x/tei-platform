// reference-go-build-from-plan は、planを読んでGoのcommandをビルドする参照実装です
// （GitHub #2831 SB-C0 §4.39 段1）。
//
// このtoolは、計画が名指すsource closureを自分で解釈し直して突き合わせ、一致した
// ときだけビルドします。計画のfingerprint、読んだ宣言のdigest、成果物のdigest、
// 使ったstateを申告します。受理・署名・配備・権限は行いません。
package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"tei/internal/buildintent"
	"tei/internal/plugin"
	"tei/tools/internal/gobuilder"
)

const schemaVersion = "tei.reference.go-build-from-plan/v0"

const claimBoundary = "one build of a plan that this process read and checked against the working tree; it reports what it read, hashed and ran, and confers no acceptance, no L5 or runtime authority, and no supply trust"

// report is one JSON object on stdout.
type report struct {
	SchemaVersion string `json:"schemaVersion"`
	Nature        string `json:"nature"`

	PlanPath        string `json:"planPath"`
	PlanBytesDigest string `json:"planBytesDigest"`
	PlanFingerprint string `json:"planFingerprint"`

	IntentPath        string `json:"intentPath,omitempty"`
	IntentBytesDigest string `json:"intentBytesDigest,omitempty"`

	SourceRoot string          `json:"sourceRoot"`
	Sources    []checkedSource `json:"sources"`
	// DefinitionBytesDigest is the digest of the plugin-definition source this
	// process checked. The entry compares it with its own plan (§4.36), so a
	// builder that read something else cannot pass unnoticed.
	DefinitionBytesDigest string `json:"definitionBytesDigest,omitempty"`

	ArtifactDigest string `json:"artifactDigest,omitempty"`
	ArtifactPath   string `json:"artifactPath,omitempty"`
	StateDir       string `json:"stateDir,omitempty"`

	// IsolatedTree is set when the build ran in a tree made only of the planned
	// sources; it records the staged members and the module context taken.
	IsolatedTree       *isolatedTree `json:"isolatedTree,omitempty"`
	IsolatedTreeDigest string        `json:"isolatedTreeDigest,omitempty"`

	// DependencyCoverage answers "does the plan cover what the build compiles?".
	DependencyCoverage *dependencyCoverage `json:"dependencyCoverage,omitempty"`

	BuildCommand []string `json:"buildCommand"`
	BuildOK      bool     `json:"buildSucceeded"`
	BuildError   string   `json:"buildError,omitempty"`

	Lock *lockReport `json:"buildArtifactLock,omitempty"`

	Boundary string `json:"claimBoundary"`
}

type checkedSource struct {
	Kind    string `json:"kind"`
	ID      string `json:"id"`
	Locator string `json:"locator"`
	Planned string `json:"plannedDigest"`
	Actual  string `json:"actualDigest"`
	Matched bool   `json:"matched"`
}

// dependencyCoverage records which compiled packages the plan covers. External
// modules come from the supply rather than from the plan's sources, so they are
// listed separately instead of being counted as uncovered.
type dependencyCoverage struct {
	Checked           bool     `json:"checked"`
	Covered           []string `json:"covered,omitempty"`
	External          []string `json:"external,omitempty"`
	Uncovered         []string `json:"uncovered,omitempty"`
	UnperformedReason string   `json:"notPerformedReason,omitempty"`
	Boundary          string   `json:"boundary"`
}

type lockReport struct {
	FormatVersion   int    `json:"formatVersion"`
	PlanFingerprint string `json:"planFingerprint"`
	ArtifactDigest  string `json:"artifactDigest"`
	LockFingerprint string `json:"lockFingerprint"`
	Validated       bool   `json:"validated"`
}

func main() {
	if err := run(os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string, out io.Writer) error {
	fs := flag.NewFlagSet("reference-go-build-from-plan", flag.ContinueOnError)
	planPath := fs.String("plan", "", "ビルドする計画（必須）")
	intentPath := fs.String("intent", "", "計画の元になった宣言（省略可）")
	repo := fs.String("repo", ".", "sourceを解決するroot（必須に近い。既定は現在地）")
	goExecutable := fs.String("go", "", "exact Go実行ファイル（省略時はPATHのgo）")
	artifactOut := fs.String("artifact-out", "", "成果物を置くパス（必須）")
	stateDir := fs.String("state", "", "分離したBuild state（必須）")
	isolate := fs.Bool("isolate", false, "計画のsourceだけを新しい木へ写してビルドする（照合とcompileの間を空けない）")
	supplyDir := fs.String("supply", "", "モジュール供給ディレクトリ（依存がある対象では必須）")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() > 0 {
		return fmt.Errorf("positional arguments are not accepted")
	}
	if strings.TrimSpace(*planPath) == "" {
		return errors.New("-plan is required")
	}
	if strings.TrimSpace(*artifactOut) == "" {
		return errors.New("-artifact-out is required")
	}
	if strings.TrimSpace(*stateDir) == "" {
		return errors.New("-state is required")
	}

	root, err := filepath.Abs(*repo)
	if err != nil {
		return err
	}

	planBytes, err := os.ReadFile(*planPath)
	if err != nil {
		return fmt.Errorf("read plan: %w", err)
	}
	plan, err := decodePlan(planBytes)
	if err != nil {
		return err
	}
	fingerprint, err := plugin.BuildPlanFingerprint(plan)
	if err != nil {
		return fmt.Errorf("plan fingerprint: %w", err)
	}

	rep := report{
		SchemaVersion:   schemaVersion,
		Nature:          "one process that read an exact BuildPlan, checked its sources against the working tree, and ran go build",
		PlanPath:        *planPath,
		PlanBytesDigest: digestOfBytes(planBytes),
		PlanFingerprint: fingerprint,
		SourceRoot:      root,
		Boundary:        claimBoundary,
	}

	if strings.TrimSpace(*intentPath) != "" {
		intentBytes, readErr := os.ReadFile(*intentPath)
		if readErr != nil {
			return fmt.Errorf("read intent: %w", readErr)
		}
		rep.IntentPath = *intentPath
		rep.IntentBytesDigest = digestOfBytes(intentBytes)
	}

	// The sources the plan names must be the sources on disk. A build from
	// different bytes would produce an artifact the plan does not describe, and
	// nothing downstream would notice.
	checked, err := checkSources(root, plan)
	if err != nil {
		return err
	}
	rep.Sources = checked
	for _, source := range checked {
		if source.Kind == "plugin-definition" {
			rep.DefinitionBytesDigest = source.Actual
			break
		}
	}

	goExe := *goExecutable
	if strings.TrimSpace(goExe) == "" {
		goExe = "go"
	}
	// With -isolate the build runs in a tree made only of the planned sources, so
	// nothing can change between the check and the compile. Without it, the working
	// tree is checked and then compiled, which leaves that window open.
	buildRoot := root
	if *isolate {
		stageRoot, stageErr := os.MkdirTemp("", "tei-plan-tree-")
		if stageErr != nil {
			return fmt.Errorf("create the isolated tree: %w", stageErr)
		}
		defer os.RemoveAll(stageRoot)
		tree, stageErr := stageSources(root, stageRoot, plan, checked)
		if stageErr != nil {
			return stageErr
		}
		treeDigest, digestErr := digestTree(stageRoot)
		if digestErr != nil {
			return fmt.Errorf("digest the isolated tree: %w", digestErr)
		}
		rep.IsolatedTree = &tree
		rep.IsolatedTreeDigest = treeDigest
		buildRoot = stageRoot
	}

	// Ask what the build would compile before it compiles: a plan that does not
	// cover the module-local packages the build pulls in does not describe it.
	coverage := checkDependencyCoverage(buildRoot, goExe, plan, *stateDir, *supplyDir, checked)
	rep.DependencyCoverage = &coverage
	if len(coverage.Uncovered) > 0 {
		rep.BuildError = "the plan does not cover every package this build compiles: " + strings.Join(coverage.Uncovered, ", ")
		return emit(out, rep)
	}
	if coverage.UnperformedReason != "" {
		rep.BuildError = "the dependency coverage check did not run: " + coverage.UnperformedReason
		return emit(out, rep)
	}

	rep.BuildCommand = []string{goExe, "build", "-trimpath", "-buildvcs=false", "-mod=readonly", "-o", *artifactOut, plan.Recipe.Entrypoint}

	var buildErr error
	if strings.TrimSpace(*supplyDir) != "" {
		buildErr = gobuilder.BuildPackageWithSupply(buildRoot, goExe, plan.Recipe.Entrypoint, *artifactOut, *stateDir, *supplyDir)
	} else {
		buildErr = gobuilder.BuildPackage(buildRoot, goExe, plan.Recipe.Entrypoint, *artifactOut, *stateDir)
	}
	rep.StateDir = *stateDir
	if buildErr != nil {
		rep.BuildError = buildErr.Error()
		return emit(out, rep)
	}
	rep.BuildOK = true
	rep.ArtifactPath = *artifactOut

	artifactDigest, err := digestOfFile(*artifactOut)
	if err != nil {
		return fmt.Errorf("digest the artifact: %w", err)
	}
	rep.ArtifactDigest = artifactDigest

	// Form and validate the lock in this process, from the values it holds.
	lock, lockErr := plugin.LockBuildOutput(plan, artifactDigest)
	if lockErr != nil {
		rep.BuildError = "the plan-to-artifact lock does not validate: " + lockErr.Error()
		return emit(out, rep)
	}
	lockFingerprint, lockErr := plugin.BuildArtifactLockFingerprint(plugin.BuildArtifactLock{
		FormatVersion: plugin.BuildArtifactLockFormatVersion,
		Builds:        []plugin.LockedBuild{lock},
	})
	if lockErr != nil {
		rep.BuildError = "the lock fingerprint could not be derived: " + lockErr.Error()
		return emit(out, rep)
	}
	rep.Lock = &lockReport{
		FormatVersion:   plugin.BuildArtifactLockFormatVersion,
		PlanFingerprint: lock.PlanFingerprint,
		ArtifactDigest:  lock.ArtifactDigest,
		LockFingerprint: lockFingerprint,
		Validated:       true,
	}
	return emit(out, rep)
}

// decodePlan reads the plan strictly: unknown keys, case variants and trailing
// values are refusals, so a plan this process did not fully understand is never
// built from.
func decodePlan(raw []byte) (plugin.BuildPlan, error) {
	var plan plugin.BuildPlan
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&plan); err != nil {
		return plugin.BuildPlan{}, fmt.Errorf("decode plan: %w", err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		if err == nil {
			return plugin.BuildPlan{}, errors.New("plan contains multiple JSON values")
		}
		return plugin.BuildPlan{}, fmt.Errorf("decode trailing plan content: %w", err)
	}
	if err := plan.Validate(); err != nil {
		return plugin.BuildPlan{}, fmt.Errorf("plan does not validate: %w", err)
	}
	return plan, nil
}

// checkSources resolves every source the plan names from the same root, using the
// same resolver the entry uses, and refuses when any of them differs. Reusing the
// resolver is the point: a second implementation of the framing would drift.
func checkSources(root string, plan plugin.BuildPlan) ([]checkedSource, error) {
	if len(plan.Sources) == 0 {
		return nil, errors.New("the plan names no sources, so there is nothing to build from")
	}
	resolver := buildintent.DirSourceResolver{Root: root}
	checked := make([]checkedSource, 0, len(plan.Sources))
	for _, source := range plan.Sources {
		resolved, err := resolver.Resolve(buildintent.IntentSource{
			Kind: source.Kind,
			ID:   source.ID,
			Ref:  source.Locator,
		})
		if err != nil {
			return nil, fmt.Errorf("resolve source %q: %w", source.Locator, err)
		}
		entry := checkedSource{
			Kind:    source.Kind,
			ID:      source.ID,
			Locator: source.Locator,
			Planned: source.Digest,
			Actual:  resolved.Source.Digest,
		}
		if entry.Planned != entry.Actual {
			return nil, fmt.Errorf("source %q is %s on disk but the plan names %s; this build would produce an artifact the plan does not describe", source.Locator, entry.Actual, entry.Planned)
		}
		entry.Matched = true
		checked = append(checked, entry)
	}
	sort.Slice(checked, func(i, j int) bool { return checked[i].Locator < checked[j].Locator })
	return checked, nil
}

// checkDependencyCoverage compares the packages the build compiles with the
// sources the plan names. A module-local package that no planned source contains
// means the plan and the build describe different things.
//
// External modules are listed separately: they come from the supply, and this
// check is about the tree the plan claims to describe.
func checkDependencyCoverage(root, goExe string, plan plugin.BuildPlan, stateDir, supply string, checked []checkedSource) dependencyCoverage {
	coverage := dependencyCoverage{
		Boundary: "every module-local package the build compiles is inside a source the plan names; external modules come from the supply and are listed, not judged",
	}
	lines, err := gobuilder.ListDependencies(root, goExe, plan.Recipe.Entrypoint, stateDir, supply)
	if err != nil {
		coverage.UnperformedReason = err.Error()
		return coverage
	}
	coverage.Checked = true

	planned := make([]string, 0, len(checked))
	for _, source := range checked {
		if source.Kind == "go-package" {
			planned = append(planned, filepath.Clean(filepath.Join(root, filepath.FromSlash(source.Locator))))
		}
	}
	root = filepath.Clean(root)
	for _, line := range lines {
		parts := strings.SplitN(line, "\t", 2)
		if len(parts) != 2 {
			continue
		}
		importPath, dir := parts[0], filepath.Clean(parts[1])
		if dir == "" || strings.HasPrefix(importPath, "vendor/") {
			continue
		}
		if !strings.HasPrefix(dir, root) {
			coverage.External = append(coverage.External, importPath)
			continue
		}
		if containsPath(planned, dir) {
			coverage.Covered = append(coverage.Covered, importPath)
			continue
		}
		coverage.Uncovered = append(coverage.Uncovered, importPath+" ("+dir+")")
	}
	sort.Strings(coverage.Covered)
	sort.Strings(coverage.External)
	sort.Strings(coverage.Uncovered)
	return coverage
}

// containsPath reports whether dir is one of the planned directories or inside
// one of them.
func containsPath(planned []string, dir string) bool {
	for _, base := range planned {
		if dir == base || strings.HasPrefix(dir, base+string(filepath.Separator)) {
			return true
		}
	}
	return false
}

func emit(out io.Writer, rep report) error {
	encoded, err := json.Marshal(rep)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(out, string(encoded))
	return err
}

func digestOfBytes(data []byte) string {
	sum := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(sum[:])
}

func digestOfFile(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	h := sha256.New()
	if _, err := io.Copy(h, file); err != nil {
		return "", err
	}
	return "sha256:" + hex.EncodeToString(h.Sum(nil)), nil
}
