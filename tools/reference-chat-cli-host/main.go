// Command reference-chat-cli-host is an E0 read-only probe for a trusted
// development host. It is not a qualification gate or TEI runtime authority.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
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
	EligibleForLocalGate bool   `json:"eligibleForLocalGate"`
	LiveRemoteMain       string `json:"liveRemoteMain"`
	Credentials          string `json:"credentials"`
	RepositoryHealth     string `json:"repositoryHealth"`
}

func commandOutput(root string, args ...string) (string, error) {
	cmd := exec.Command("git", append([]string{"-C", root}, args...)...)
	// Never include command stderr in a machine-readable status: it may
	// contain a configured remote URL or other local environment details.
	data, err := cmd.Output()
	return strings.TrimSpace(string(data)), err
}

func probe(root, expectedHead, expectedMain string) ProbeReport {
	result := ProbeReport{
		Schema: "tei.dev-host-probe/v0", Git: "unavailable", Go: "unavailable",
		Checkout: "unavailable", ExpectedHeadMatch: "not-checked",
		TrackingMainMatch: "not-checked", MainAncestor: "not-checked",
		LiveRemoteMain: "not-checked", Credentials: "not-checked",
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
	result.EligibleForLocalGate = result.Go == "available"
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
