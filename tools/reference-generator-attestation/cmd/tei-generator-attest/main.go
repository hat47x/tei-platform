package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"

	generatorattestation "tei/tools/reference-generator-attestation"
)

// tei-generator-attest builds, signs, or verifies a canonical statement about
// one generator candidate. It does not decide whether the signer is trusted.
func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: tei-generator-attest <statement|sign|verify> [options]")
		os.Exit(2)
	}
	command := os.Args[1]
	fs := flag.NewFlagSet(command, flag.ExitOnError)
	receiptPath := fs.String("receipt", "", "generator selection receipt JSON")
	candidateID := fs.String("candidate", "", "candidate id in the receipt")
	statementPath := fs.String("statement", "", "statement JSON")
	keyPath := fs.String("key", "", "ssh private key for sign")
	signaturePath := fs.String("signature", "", "detached signature")
	allowedSignersPath := fs.String("allowed-signers", "", "ssh allowed signers file")
	identity := fs.String("identity", "", "principal in the allowed signers file")
	namespace := fs.String("namespace", generatorattestation.Namespace, "ssh signature namespace")
	outPath := fs.String("out", "", "output path")
	_ = fs.Parse(os.Args[2:])
	switch command {
	case "statement":
		if *receiptPath == "" || *candidateID == "" || *outPath == "" {
			fail(fmt.Errorf("-receipt, -candidate, and -out are required"))
		}
		statement, encoded, err := generatorattestation.BuildStatementFromSelection(*receiptPath, *candidateID)
		if err != nil {
			fail(err)
		}
		if err := os.WriteFile(*outPath, encoded, 0o644); err != nil {
			fail(err)
		}
		emit(statement)
	case "sign":
		if *statementPath == "" || *keyPath == "" {
			fail(fmt.Errorf("-statement and -key are required"))
		}
		receipt, err := generatorattestation.Sign(context.Background(), *statementPath, *keyPath, *outPath, *namespace)
		if err != nil {
			fail(err)
		}
		emit(receipt)
	case "verify":
		if *statementPath == "" || *signaturePath == "" || *allowedSignersPath == "" || *identity == "" {
			fail(fmt.Errorf("-statement, -signature, -allowed-signers, and -identity are required"))
		}
		receipt, err := generatorattestation.Verify(context.Background(), *statementPath, *signaturePath, *allowedSignersPath, *identity, *namespace)
		if err != nil {
			fail(err)
		}
		emit(receipt)
		if !receipt.Verified {
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
