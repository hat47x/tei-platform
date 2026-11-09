package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	queryinputnullability "tei/tools/reference-query-input-nullability"
)

// tei-query-nullability binds a reference query request and prints whether it
// may proceed to SQL execution. It does not execute SQL or contact PostgreSQL.
func main() {
	contractPath := flag.String("contract", "", "query input contract JSON")
	requestPath := flag.String("request", "", "query request JSON")
	flag.Parse()
	if *contractPath == "" || *requestPath == "" {
		fail(fmt.Errorf("-contract and -request are required"))
	}
	receipt, err := queryinputnullability.BindFiles(*contractPath, *requestPath)
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
