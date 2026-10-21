package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"

	generatorartifactstore "tei/tools/reference-generator-artifact-store"
)

// tei-generator-artifact-store stores a selected Go candidate by its verified
// source digest and retrieves it again. It does not sign or grant authority.
func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: tei-generator-artifact-store <put|get> [options]")
		os.Exit(2)
	}
	command := os.Args[1]
	fs := flag.NewFlagSet(command, flag.ExitOnError)
	receiptPath := fs.String("receipt", "", "generator selection receipt JSON")
	candidateID := fs.String("candidate", "", "candidate id in the receipt")
	candidatesDir := fs.String("candidates", "", "directory containing candidate directories")
	storeDir := fs.String("store", "", "artifact store root")
	sourceDigest := fs.String("source-digest", "", "content-addressed source digest")
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
