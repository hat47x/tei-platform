// Command reference-chat-cli-host is an E0 read-only probe for a trusted
// development host. It is not a qualification gate or TEI runtime authority.
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

var gitSHA = regexp.MustCompile(`^[0-9a-fA-F]{40}$`)

// ProbeReport deliberately contains only status values and public commit IDs.
// In particular it never serializes Git remotes, URLs or credentials.
type ProbeReport struct {
	Schema               string `json:"schema"`
	Git                  string `json:"git"`
	Go                   string `json:"go"`
	Checkout             string `json:"checkout"`
	CandidateHead        string `json:"candidateHead,omitempty"`
	ExpectedHeadMatch    string `json:"expectedHeadMatch"`
	TrackingMainMatch    string `json:"trackingMainMatch"`
	MainAncestor         string `json:"mainAncestor"`
	CheckoutClean        string `json:"checkoutClean"`
	IgnoredArtifacts     string `json:"ignoredArtifacts"`
	HiddenIndexFlags     string `json:"hiddenIndexFlags"`
	HistoryOverrides     string `json:"historyOverrides"`
	EligibleForLocalGate bool   `json:"eligibleForLocalGate"`
	LiveRemoteMain       string `json:"liveRemoteMain"`
	Credentials          string `json:"credentials"`
	RepositoryHealth     string `json:"repositoryHealth"`
}

// E0 must remain bounded even if a checkout has huge paths or Git hangs.
// A limit breach is an unavailable observation, not a truncated success.
var errCommandOutputLimit = errors.New("Git observation exceeds output limit")

type limitedOutput struct {
	buffer   bytes.Buffer
	maxBytes int
}

func (b *limitedOutput) Write(p []byte) (int, error) {
	if len(p) > b.maxBytes-b.buffer.Len() {
		return 0, errCommandOutputLimit
	}
	return b.buffer.Write(p)
}

func commandOutputWithContext(ctx context.Context, root string, maxBytes int, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", append([]string{
		"--no-replace-objects", "--no-optional-locks", "-C", root, "-c", "core.fsmonitor=false",
	}, args...)...)
	// Never run configured FS monitor hooks or take optional index locks.
	// Git stderr can contain credentials and remote URLs; discard it entirely.
	output := &limitedOutput{maxBytes: maxBytes}
	cmd.Stdout = output
	cmd.Stderr = io.Discard
	if err := cmd.Run(); err != nil {
		return "", err
	}
	return strings.TrimSpace(output.buffer.String()), nil
}

func commandOutput(root string, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	return commandOutputWithContext(ctx, root, 8<<20, args...)
}

func probe(root, expectedHead, expectedMain string) ProbeReport {
	result := ProbeReport{
		Schema: "tei.dev-host-probe/v0", Git: "unavailable", Go: "unavailable",
		Checkout: "unavailable", ExpectedHeadMatch: "not-checked",
		TrackingMainMatch: "not-checked", MainAncestor: "not-checked",
		CheckoutClean: "not-checked", IgnoredArtifacts: "not-checked", HiddenIndexFlags: "not-checked",
		HistoryOverrides: "not-checked",
		LiveRemoteMain:   "not-checked", Credentials: "not-checked",
		RepositoryHealth: "not-run",
	}
	if _, err := exec.LookPath("git"); err != nil {
		return result
	}
	result.Git = "available"
	if _, err := exec.LookPath("go"); err == nil {
		result.Go = "available"
	}
	physicalRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		return result
	}
	physicalRoot, err = filepath.Abs(physicalRoot)
	if err != nil {
		return result
	}
	worktree, err := commandOutput(root, "rev-parse", "--show-toplevel")
	if err != nil {
		return result
	}
	canonicalWorktree, err := filepath.EvalSymlinks(worktree)
	if err != nil || canonicalWorktree != physicalRoot {
		// Probing a subdirectory as if it were the repo root is misleading.
		return result
	}
	result.Checkout = "present"
	// A replacement or graft can change local ancestry without changing the
	// visible commit SHA. Do not qualify such a repository via this probe.
	// --no-replace-objects also protects the Git reads themselves.
	replacements, err := commandOutput(root, "for-each-ref", "--format=%(refname)", "refs/replace")
	if err != nil {
		result.HistoryOverrides = "unavailable"
		return result
	}
	graftsPath, err := commandOutput(root, "rev-parse", "--git-path", "info/grafts")
	if err != nil || graftsPath == "" {
		result.HistoryOverrides = "unavailable"
		return result
	}
	if !filepath.IsAbs(graftsPath) {
		graftsPath = filepath.Join(root, graftsPath)
	}
	_, graftsError := os.Lstat(graftsPath)
	if graftsError != nil && !os.IsNotExist(graftsError) {
		result.HistoryOverrides = "unavailable"
		return result
	}
	result.HistoryOverrides = "absent"
	if replacements != "" || graftsError == nil {
		result.HistoryOverrides = "present"
		return result
	}
	head, err := commandOutput(root, "rev-parse", "--verify", "HEAD^{commit}")
	if err != nil {
		return result
	}
	result.CandidateHead = head
	if head != expectedHead {
		result.ExpectedHeadMatch = "mismatch"
		return result
	}
	result.ExpectedHeadMatch = "match"
	main, err := commandOutput(root, "rev-parse", "--verify", "refs/remotes/origin/main^{commit}")
	if err != nil {
		result.TrackingMainMatch = "unavailable"
		return result
	}
	if main != expectedMain {
		result.TrackingMainMatch = "mismatch"
		return result
	}
	result.TrackingMainMatch = "match"
	_, err = commandOutput(root, "merge-base", "--is-ancestor", expectedMain, expectedHead)
	if err != nil {
		result.MainAncestor = "mismatch"
		return result
	}
	result.MainAncestor = "match"
	// These checks are deliberately a fail-early convenience, not another
	// implementation of RQ-034 qualification. Its runner must check them
	// again before and after executing full Repository Health.
	status, err := commandOutput(root, "status", "--porcelain=v1", "--untracked-files=all")
	if err != nil {
		result.CheckoutClean = "unavailable"
		return result
	}
	result.CheckoutClean = "clean"
	if status != "" {
		result.CheckoutClean = "dirty"
	}
	ignored, err := commandOutput(root, "ls-files", "--others", "--ignored", "--exclude-standard")
	if err != nil {
		result.IgnoredArtifacts = "unavailable"
		return result
	}
	result.IgnoredArtifacts = "absent"
	if ignored != "" {
		result.IgnoredArtifacts = "present"
	}
	indexed, err := commandOutput(root, "ls-files", "-v")
	if err != nil {
		result.HiddenIndexFlags = "unavailable"
		return result
	}
	result.HiddenIndexFlags = "absent"
	for _, entry := range strings.Split(indexed, "\n") {
		// Git prints H for ordinary tracked entries, lowercase flags for
		// assume-unchanged, and S for skip-worktree. Never return paths.
		if len(entry) > 0 && ((entry[0] >= 'a' && entry[0] <= 'z') || entry[0] == 'S') {
			result.HiddenIndexFlags = "present"
			break
		}
	}
	result.EligibleForLocalGate = result.Go == "available" && result.CheckoutClean == "clean" &&
		result.IgnoredArtifacts == "absent" && result.HiddenIndexFlags == "absent" &&
		result.HistoryOverrides == "absent"
	return result
}

func run(args []string) int {
	fs := flag.NewFlagSet("reference-chat-cli-host", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	root := fs.String("root", ".", "trusted checkout root (no shell commands executed from arguments)")
	head := fs.String("expected-head", "", "exact expected candidate commit SHA")
	main := fs.String("expected-main", "", "expected local origin/main commit SHA")
	if err := fs.Parse(args); err != nil || fs.NArg() != 0 || !gitSHA.MatchString(*head) || !gitSHA.MatchString(*main) {
		fmt.Fprintln(os.Stderr, "usage: reference-chat-cli-host --root DIR --expected-head SHA --expected-main SHA")
		return 2
	}
	report := probe(*root, strings.ToLower(*head), strings.ToLower(*main))
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(report); err != nil {
		return 1
	}
	if !report.EligibleForLocalGate {
		return 1
	}
	return 0
}

func main() { os.Exit(run(os.Args[1:])) }
