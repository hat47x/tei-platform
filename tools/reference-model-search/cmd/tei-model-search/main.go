package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	"tei/internal/workspace"
	modelsearch "tei/tools/reference-model-search"
)

// The CLI only reads a Workspace and prints a search result or a replacement
// proposal. It cannot write sources or accept a proposal.
func main() {
	root := flag.String("workspace", "", "source Workspace root to read")
	text := flag.String("text", "", "text to search for")
	cs := flag.Bool("case-sensitive", false, "match case")
	exact := flag.Bool("exact", false, "match the whole Symbol")
	kinds := flag.String("kinds", "", "comma-separated element kinds")
	with := flag.String("replace", "", "replacement; prints a proposal instead of hits")
	flag.Parse()
	if err := run(*root, *text, *cs, *exact, *kinds, *with); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(root, text string, cs, exact bool, kinds, with string) error {
	if root == "" || text == "" {
		return fmt.Errorf("-workspace and -text are required")
	}
	loaded, err := workspace.Load(root)
	if err != nil {
		return fmt.Errorf("load Workspace: %w", err)
	}
	q := modelsearch.Query{Text: text, CaseSensitive: cs, Exact: exact}
	if kinds != "" {
		q.Kinds = strings.Split(kinds, ",")
	}
	res, err := modelsearch.Search(loaded.World, q)
	if err != nil {
		return err
	}
	var out interface{} = res
	if with != "" {
		if out, err = modelsearch.Propose(loaded.World, res, with, nil); err != nil {
			return err
		}
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(out)
}
