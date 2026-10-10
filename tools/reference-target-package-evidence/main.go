package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
)

func main() {
	entry := flag.String("entry", "./tools/reference-first-web-target", "literal Go entry package")
	tags := flag.String("tags", "", "optional Go build tags")
	flag.Parse()
	report, err := produceEvidence(*entry, *tags)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(report); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
