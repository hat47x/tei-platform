package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	generatorartifactstore "tei/tools/reference-generator-artifact-store"
)

// tei-generator-artifact-store stores a selected Go candidate by its verified
// source digest and retrieves it again. It does not sign or grant authority.
func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: tei-generator-artifact-store <put|get|list|prune|audit|checkpoint|verify-checkpoint> [options]")
		os.Exit(2)
	}
	command := os.Args[1]
	fs := flag.NewFlagSet(command, flag.ExitOnError)
	receiptPath := fs.String("receipt", "", "generator selection receipt JSON")
	candidateID := fs.String("candidate", "", "candidate id in the receipt")
	candidatesDir := fs.String("candidates", "", "directory containing candidate directories")
	storeDir := fs.String("store", "", "artifact store root")
	sourceDigest := fs.String("source-digest", "", "content-addressed source digest")
	keep := fs.String("keep", "", "comma-separated source digests to keep for prune")
	dryRun := fs.Bool("dry-run", false, "report prune candidates without deleting")
	outPath := fs.String("out", "", "write the checkpoint JSON to this file")
	checkpointFile := fs.String("checkpoint", "", "stored checkpoint JSON to verify")
	signaturePath := fs.String("signature", "", "detached signature over the checkpoint")
	allowedSignersPath := fs.String("allowed-signers", "", "ssh allowed signers file")
	identity := fs.String("identity", "", "principal in the allowed signers file")
	namespace := fs.String("namespace", "tei-audit", "ssh signature namespace")
	_ = fs.Parse(os.Args[2:])
	switch command {
	case "put":
		if *receiptPath == "" || *candidateID == "" || *candidatesDir == "" || *storeDir == "" {
			fail(fmt.Errorf("-receipt, -candidate, -candidates, and -store are required"))
		}
		receipt, err := generatorartifactstore.Put(context.Background(), generatorartifactstore.StoreOptions{
			ReceiptPath: *receiptPath, CandidateID: *candidateID,
			CandidatesDir: *candidatesDir, StoreDir: *storeDir,
		})
		if err != nil {
			fail(err)
		}
		emit(receipt)
		if !receipt.Passed {
			os.Exit(1)
		}
	case "get":
		if *storeDir == "" || *sourceDigest == "" {
			fail(fmt.Errorf("-store and -source-digest are required"))
		}
		receipt, err := generatorartifactstore.Get(*storeDir, *sourceDigest)
		if err != nil {
			fail(err)
		}
		emit(receipt)
		if !receipt.Passed {
			os.Exit(1)
		}
	case "list":
		if *storeDir == "" {
			fail(fmt.Errorf("-store is required"))
		}
		receipt, err := generatorartifactstore.List(*storeDir)
		if err != nil {
			fail(err)
		}
		emit(receipt)
		if !receipt.Passed {
			os.Exit(1)
		}
	case "prune":
		if *storeDir == "" || *keep == "" {
			fail(fmt.Errorf("-store and -keep are required"))
		}
		receipt, err := generatorartifactstore.Prune(generatorartifactstore.PruneOptions{
			StoreDir: *storeDir, Keep: splitCSV(*keep), DryRun: *dryRun,
		})
		if err != nil {
			fail(err)
		}
		emit(receipt)
		if !receipt.Passed {
			os.Exit(1)
		}
	case "audit":
		if *storeDir == "" {
			fail(fmt.Errorf("-store is required"))
		}
		receipt, err := generatorartifactstore.Audit(*storeDir)
		if err != nil {
			fail(err)
		}
		emit(receipt)
		if !receipt.Passed {
			os.Exit(1)
		}
	case "checkpoint":
		if *storeDir == "" {
			fail(fmt.Errorf("-store is required"))
		}
		receipt, err := generatorartifactstore.Checkpoint(*storeDir)
		if err != nil {
			fail(err)
		}
		if *outPath != "" {
			raw, err := json.MarshalIndent(receipt, "", "  ")
			if err != nil {
				fail(err)
			}
			if err := os.WriteFile(*outPath, append(raw, '\n'), 0o644); err != nil {
				fail(err)
			}
		}
		emit(receipt)
		if !receipt.Passed {
			os.Exit(1)
		}
	case "verify-checkpoint":
		if *storeDir == "" || *checkpointFile == "" {
			fail(fmt.Errorf("-store and -checkpoint are required"))
		}
		if *signaturePath != "" {
			if *allowedSignersPath == "" || *identity == "" {
				fail(fmt.Errorf("-signature requires -allowed-signers and -identity"))
			}
			receipt, err := generatorartifactstore.VerifySignedCheckpoint(context.Background(),
				*storeDir, *checkpointFile, *signaturePath, *allowedSignersPath, *identity, *namespace)
			if err != nil {
				fail(err)
			}
			emit(receipt)
			if !receipt.Passed {
				os.Exit(1)
			}
			return
		}
		receipt, err := generatorartifactstore.VerifyCheckpoint(*storeDir, *checkpointFile)
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

func splitCSV(value string) []string {
	var values []string
	for _, part := range strings.Split(value, ",") {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			values = append(values, trimmed)
		}
	}
	return values
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
