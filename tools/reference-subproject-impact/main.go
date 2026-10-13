// Command reference-subproject-impact reports the source-level change surface
// between exact Git commits. It is evidence, not a pass/fail policy.
package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"sort"
	"strings"
)

var exactSHA = regexp.MustCompile("^[a-fA-F0-9]{40}$")

type impact struct {
	BaseSHA              string   `json:"baseSha"`
	HeadSHA              string   `json:"headSha"`
	ChangedPaths         int      `json:"changedPaths"`
	PluginSubprojects    []string `json:"pluginSubprojects"`
	PluginSourcePaths    []string `json:"pluginSourcePaths"`
	CoreCandidateSources []string `json:"coreCandidateSources"`
	CoreCandidateTests   []string `json:"coreCandidateTests"`
	OtherPaths           int      `json:"otherPaths"`
}

// The directories here are an intentionally conservative measurement scope,
// not a declaration that the final Core must contain all these packages.
var coreCandidates = []string{
	"core/",
	"internal/core/",
	"internal/composition/",
	"internal/subject/",
	"internal/permission/",
	"internal/semantic/",
	"internal/stdlib/",
}

func git(repo string, args ...string) ([]byte, error) {
	cmd := exec.Command("git", append([]string{"-C", repo}, args...)...)
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git %s: %w", strings.Join(args, " "), err)
	}
	return out, nil
}

func changedPaths(repo, base, head string) ([]string, error) {
	// No refs, branch names or user-supplied options: both endpoints must be
	// exact commits. Renames are deliberately shown as deletion + addition,
	// preventing a move out of a Core directory from being overlooked.
	for _, sha := range []string{base, head} {
		if !exactSHA.MatchString(sha) {
			return nil, fmt.Errorf("expected exact 40-character commit SHA, got %q", sha)
		}
		if _, err := git(repo, "cat-file", "-e", sha+"^{commit}"); err != nil {
			return nil, fmt.Errorf("commit %s is unavailable: %w", sha, err)
		}
	}
	out, err := git(repo, "diff", "--no-renames", "--name-only", "-z", base, head, "--")
	if err != nil {
		return nil, err
	}
	if len(out) == 0 {
		return nil, nil
	}
	if out[len(out)-1] != 0 {
		return nil, errors.New("git diff did not return NUL-terminated paths")
	}
	parts := bytes.Split(out[:len(out)-1], []byte{0})
	paths := make([]string, 0, len(parts))
	for _, raw := range parts {
		if len(raw) == 0 {
			return nil, errors.New("git diff returned an empty path")
		}
		paths = append(paths, string(raw))
	}
	return paths, nil
}

func measure(base, head string, paths []string) impact {
	report := impact{
		BaseSHA: base, HeadSHA: head, ChangedPaths: len(paths),
		PluginSubprojects: []string{}, PluginSourcePaths: []string{},
		CoreCandidateSources: []string{}, CoreCandidateTests: []string{},
	}
	plugins := map[string]bool{}
	for _, file := range paths {
		if strings.HasPrefix(file, "plugins/") {
			parts := strings.Split(file, "/")
			if len(parts) >= 3 && parts[1] != "" {
				plugins[parts[1]] = true
				if strings.HasSuffix(file, ".go") && !strings.HasSuffix(file, "_test.go") {
					report.PluginSourcePaths = append(report.PluginSourcePaths, file)
				}
				continue
			}
		}
		foundCore := false
		for _, prefix := range coreCandidates {
			if !strings.HasPrefix(file, prefix) {
				continue
			}
			if strings.HasSuffix(file, ".go") {
				if strings.HasSuffix(file, "_test.go") {
					report.CoreCandidateTests = append(report.CoreCandidateTests, file)
				} else {
					report.CoreCandidateSources = append(report.CoreCandidateSources, file)
				}
			} else {
				report.OtherPaths++
			}
			foundCore = true
			break
		}
		if !foundCore {
			report.OtherPaths++
		}
	}
	for plugin := range plugins {
		report.PluginSubprojects = append(report.PluginSubprojects, plugin)
	}
	sort.Strings(report.PluginSubprojects)
	sort.Strings(report.PluginSourcePaths)
	sort.Strings(report.CoreCandidateSources)
	sort.Strings(report.CoreCandidateTests)
	return report
}

func report(repo, base, head string) (impact, error) {
	paths, err := changedPaths(repo, base, head)
	if err != nil {
		return impact{}, err
	}
	return measure(base, head, paths), nil
}

func main() {
	var repo, base, head string
	flag.StringVar(&repo, "repo", ".", "repository working directory")
	flag.StringVar(&base, "base", "", "exact 40-character base commit SHA")
	flag.StringVar(&head, "head", "", "exact 40-character candidate commit SHA")
	flag.Parse()
	if flag.NArg() != 0 {
		fmt.Fprintln(os.Stderr, "unexpected positional arguments")
		os.Exit(2)
	}
	result, err := report(repo, base, head)
	if err != nil {
		fmt.Fprintln(os.Stderr, "subproject impact:", err)
		os.Exit(1)
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(result); err != nil {
		fmt.Fprintln(os.Stderr, "encode subproject impact:", err)
		os.Exit(1)
	}
	// Even if existing Core sources changed, the command does not block a PR.
	// Change counts are observations; reasons and authority require review.
}
