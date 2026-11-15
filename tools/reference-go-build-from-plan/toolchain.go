package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"tei/tools/internal/gobuilder"
)

// The plan this tool builds does not name the Go toolchain: -go is an argument,
// and the same plan built with two toolchains yields two different artifacts
// (plan/evidence/TEI_GO_SELF_BUILD_SEED_REACH_2026-11-13.md). Until the plan
// carries it, the least this process can do is say which toolchain it ran, so a
// reader of the report can tell two such builds apart.

const toolchainBoundary = "the Go toolchain this process resolved from -go before building, hashed by this process; when the plan names a toolchain (go.toolchain.closureDigest) this process refuses to build with any other, and when it does not, this is a report of what was used, not a check against anything declared. It confers no trust in the toolchain"

// toolchainReport is what this process resolved and hashed. Every value is this
// builder's own statement; the entry re-reads the executable to compare.
type toolchainReport struct {
	// GoExecutable is the exact file -go resolved to (PATH is consulted only when
	// -go is a bare name, and the result is recorded here).
	GoExecutable       string `json:"goExecutable,omitempty"`
	GoExecutableDigest string `json:"goExecutableDigest,omitempty"`
	// GoRoot is the directory the executable sits in as bin/go. ClosureDigest
	// covers VERSION, bin/go, pkg/tool/<os>_<arch> and src under it, with the
	// encoding the other reference builders use (gobuilder.HashToolchain).
	GoRoot        string `json:"goRoot,omitempty"`
	ClosureDigest string `json:"closureDigest,omitempty"`
	// PlannedClosureDigest is the toolchain the plan names, when it names one
	// (plugin.BuildParameterGoToolchainClosureDigest). MatchesPlan is true only
	// when the plan names one and ClosureDigest equals it.
	PlannedClosureDigest string `json:"plannedClosureDigest,omitempty"`
	MatchesPlan          bool   `json:"matchesPlan"`
	// NotObserved says what could not be read, so an absent digest is never taken
	// for an unchanged toolchain.
	NotObserved string `json:"notObserved,omitempty"`
	Boundary    string `json:"boundary"`
}

// observeToolchain resolves -go to one file and hashes it and the toolchain
// around it. It never fails the build: a toolchain it cannot read is reported as
// not observed.
func observeToolchain(goExe string) *toolchainReport {
	record := &toolchainReport{Boundary: toolchainBoundary}
	path := strings.TrimSpace(goExe)
	if path == "" {
		record.NotObserved = "no Go executable was named"
		return record
	}
	if !strings.ContainsAny(path, `/\`) {
		found, err := exec.LookPath(path)
		if err != nil {
			record.NotObserved = "the Go executable was not found on PATH: " + err.Error()
			return record
		}
		path = found
	}
	resolved, err := gobuilder.SelectExecutable(path, "", runtime.GOOS)
	if err != nil {
		record.NotObserved = "the Go executable could not be resolved: " + err.Error()
		return record
	}
	record.GoExecutable = resolved
	digest, err := digestOfFile(resolved)
	if err != nil {
		record.NotObserved = "the Go executable could not be read: " + err.Error()
		return record
	}
	record.GoExecutableDigest = digest

	// A Go distribution keeps the command at <root>/bin/go. Anything else (a
	// wrapper script, a copied binary) has no toolchain around it to hash, and
	// saying so is better than hashing whatever happens to be nearby.
	root := filepath.Dir(filepath.Dir(resolved))
	if _, statErr := os.Stat(filepath.Join(root, "VERSION")); statErr != nil {
		record.NotObserved = "the executable is not at <root>/bin/go of a Go distribution (no VERSION beside bin), so only the executable was hashed"
		return record
	}
	closure, err := gobuilder.HashToolchain(root, runtime.GOOS, runtime.GOARCH)
	if err != nil {
		record.NotObserved = "the toolchain around the executable could not be hashed: " + err.Error()
		return record
	}
	record.GoRoot = root
	record.ClosureDigest = closure
	return record
}

// checkPlannedToolchain records the toolchain the plan names and returns why the
// build must not start, or "" when it may. A plan that names no toolchain never
// stops a build here; one that names a toolchain this process could not hash, or
// a different one, does.
func checkPlannedToolchain(planned string, planNames bool, record *toolchainReport) string {
	if !planNames {
		return ""
	}
	record.PlannedClosureDigest = planned
	switch {
	case record.ClosureDigest == "":
		return "the plan names Go toolchain " + planned + ", but this process could not hash the toolchain -go resolves to" + reasonSuffix(record.NotObserved)
	case record.ClosureDigest != planned:
		return "the plan names Go toolchain " + planned + ", but -go resolves to " + record.ClosureDigest + "; a build with another toolchain would produce an artifact the plan does not describe"
	}
	record.MatchesPlan = true
	return ""
}

func reasonSuffix(reason string) string {
	if strings.TrimSpace(reason) == "" {
		return ""
	}
	return ": " + reason
}
