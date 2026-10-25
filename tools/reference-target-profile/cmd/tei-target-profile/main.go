package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	targetprofile "tei/tools/reference-target-profile"
)

// tei-target-profile validates a Target Profile, applies it to one generator
// selection run, or compares the per-case output of multiple profiles. It does
// not deploy to a real target, enforce every declared constraint, or prove that
// two targets are equivalent.
func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: tei-target-profile <validate|run|compare|deploy|rollback> [-profile FILE] [-profiles FILE,...] [options]")
		os.Exit(2)
	}
	command := os.Args[1]
	fs := flag.NewFlagSet(command, flag.ExitOnError)
	profilePath := fs.String("profile", "", "target profile JSON")
	profilePaths := fs.String("profiles", "", "comma-separated target profile JSON files for compare")
	workspaceDir := fs.String("workspace", "reference/workspace", "accepted Workspace root")
	contractPath := fs.String("contract", "", "generator contract JSON")
	fixturesPath := fs.String("fixtures", "", "fixed fixture JSON")
	candidatesDir := fs.String("candidates", "", "directory containing candidate directories")
	enforceNetwork := fs.Bool("enforce-network", false, "require Linux network namespace isolation for network:none (run/compare only)")
	enforceCPU := fs.Bool("enforce-cpu", false, "require Linux per-process CPU time limits (run/compare only)")
	rootDir := fs.String("root", "", "local deployment root for deploy")
	rollbackTo := fs.String("to", "", "history source digest for rollback")
	_ = fs.Parse(os.Args[2:])
	if command != "compare" && command != "rollback" && *profilePath == "" {
		fail(fmt.Errorf("-profile is required"))
	}
	if *enforceNetwork && command != "run" && command != "compare" {
		fail(fmt.Errorf("-enforce-network is supported only for run/compare"))
	}
	if *enforceCPU && command != "run" && command != "compare" {
		fail(fmt.Errorf("-enforce-cpu is supported only for run/compare"))
	}
	switch command {
	case "validate":
		profile, err := targetprofile.Load(*profilePath)
		if err != nil {
			fail(err)
		}
		emit(profile)
	case "run":
		if *contractPath == "" || *fixturesPath == "" || *candidatesDir == "" {
			fail(fmt.Errorf("-contract, -fixtures, and -candidates are required for run"))
		}
		receipt, err := targetprofile.Run(context.Background(), targetprofile.RunOptions{
			WorkspaceDir: *workspaceDir, ContractPath: *contractPath,
			FixturePath: *fixturesPath, CandidatesDir: *candidatesDir, ProfilePath: *profilePath, EnforceNetwork: *enforceNetwork,
			EnforceCPU: *enforceCPU,
		})
		if err != nil {
			fail(err)
		}
		emit(receipt)
		if !receipt.Passed {
			os.Exit(1)
		}
	case "compare":
		if *contractPath == "" || *fixturesPath == "" || *candidatesDir == "" {
			fail(fmt.Errorf("-contract, -fixtures, and -candidates are required for compare"))
		}
		var paths []string
		for _, path := range strings.Split(*profilePaths, ",") {
			if trimmed := strings.TrimSpace(path); trimmed != "" {
				paths = append(paths, trimmed)
			}
		}
		if len(paths) < 2 {
			fail(fmt.Errorf("-profiles requires at least two profile files"))
		}
		receipt, err := targetprofile.Compare(context.Background(), targetprofile.CompareOptions{
			WorkspaceDir: *workspaceDir, ContractPath: *contractPath,
			FixturePath: *fixturesPath, CandidatesDir: *candidatesDir, ProfilePaths: paths, EnforceNetwork: *enforceNetwork,
			EnforceCPU: *enforceCPU,
		})
		if err != nil {
			fail(err)
		}
		emit(receipt)
		if !receipt.Passed {
			os.Exit(1)
		}
	case "deploy":
		if *candidatesDir == "" {
			fail(fmt.Errorf("-candidates is required for deploy"))
		}
		receipt, err := targetprofile.Deploy(context.Background(), targetprofile.DeployOptions{
			ProfilePath: *profilePath, CandidatesDir: *candidatesDir, RootDir: *rootDir,
		})
		if err != nil {
			fail(err)
		}
		emit(receipt)
		if !receipt.Passed {
			os.Exit(1)
		}
	case "rollback":
		if *rootDir == "" {
			fail(fmt.Errorf("-root is required for rollback"))
		}
		var receipt targetprofile.RollbackReceipt
		var err error
		if *rollbackTo != "" {
			receipt, err = targetprofile.RollbackTo(*rootDir, *rollbackTo)
		} else {
			receipt, err = targetprofile.Rollback(*rootDir)
		}
		if err != nil {
			fail(err)
		}
		emit(receipt)
		if !receipt.Passed {
			os.Exit(1)
		}
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q\n", command)
		os.Exit(2)
	}
}

func emit(value any) {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(value); err != nil {
		fail(err)
	}
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
