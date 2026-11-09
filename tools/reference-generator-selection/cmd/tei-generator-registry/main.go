package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	generatorselection "tei/tools/reference-generator-selection"
)

// tei-generator-registry validates a generator registry or resolves the single
// active version for an id. It does not grant authority.
func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: tei-generator-registry <validate|resolve> -registry FILE -contract FILE [-id ID]")
		os.Exit(2)
	}
	command := os.Args[1]
	fs := flag.NewFlagSet(command, flag.ExitOnError)
	registryPath := fs.String("registry", "", "registry JSON")
	contractPath := fs.String("contract", "", "generator contract JSON")
	id := fs.String("id", "", "generator id for resolve")
	_ = fs.Parse(os.Args[2:])
	if *registryPath == "" || *contractPath == "" {
		fail(fmt.Errorf("-registry and -contract are required"))
	}
	registry, err := generatorselection.LoadRegistry(*registryPath, *contractPath)
	if err != nil {
		fail(err)
	}
	switch command {
	case "validate":
		emit(registry)
	case "resolve":
		if *id == "" {
			fail(fmt.Errorf("-id is required for resolve"))
		}
		entry, err := generatorselection.ActiveEntry(registry, *id)
		if err != nil {
			fail(err)
		}
		emit(entry)
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
