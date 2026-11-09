package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"

	queryinputnullability "tei/tools/reference-query-input-nullability"
)

// tei-query-nullability-postgres runs the reference contract's missing/null
// cases against an ephemeral PostgreSQL database. It does not prove production
// behavior or Provider equivalence.
func main() {
	contractPath := flag.String("contract", "", "query input nullability contract JSON")
	dsn := flag.String("dsn", os.Getenv("TEI_TEST_POSTGRES_DSN"), "test-only PostgreSQL DSN")
	flag.Parse()
	if *contractPath == "" || *dsn == "" {
		fail(fmt.Errorf("-contract and -dsn (or TEI_TEST_POSTGRES_DSN) are required"))
	}
	receipt, err := queryinputnullability.RunPostgresEvidence(context.Background(), *dsn, *contractPath)
	if err != nil {
		fail(err)
	}
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(receipt); err != nil {
		fail(err)
	}
	if !receipt.Passed {
		os.Exit(1)
	}
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
