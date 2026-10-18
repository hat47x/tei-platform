package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"

	targetprofile "tei/tools/reference-target-profile"
)

// tei-target-profile validates a Target Profile or applies it to one generator
// selection run. It does not deploy to a real target or enforce every declared
// constraint.
func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: tei-target-profile <validate|run> -profile FILE [options]")
		os.Exit(2)
	}
	command := os.Args[1]
	fs := flag.NewFlagSet(command, flag.ExitOnError)
	profilePath := fs.String("profile", "", "target profile JSON")
	workspaceDir := fs.String("workspace", "reference/workspace", "accepted Workspace root")
	contractPath := fs.String("contract", "", "generator contract JSON")
	fixturesPath := fs.String("fixtures", "", "fixed fixture JSON")
	candidatesDir := fs.String("candidates", "", "directory containing candidate directories")
	_ = fs.Parse(os.Args[2:])
	if *profilePath == "" {
		fail(fmt.Errorf("-profile is required"))
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
			FixturePath: *fixturesPath, CandidatesDir: *candidatesDir, ProfilePath: *profilePath,
		})
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
