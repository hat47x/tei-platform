package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	generatorselection "tei/tools/reference-generator-selection"
)

// tei-generator-record records an externally generated candidate together with
// the prompt and context it came from. It never calls a model and never selects
// the candidate; selection remains a separate step.
func main() {
	id := flag.String("id", "", "candidate id")
	origin := flag.String("origin", "generated-ai", "candidate origin")
	provider := flag.String("provider", "", "model provider")
	model := flag.String("model", "", "model name or version")
	contractPath := flag.String("contract", "", "generator contract JSON")
	language := flag.String("language", "go", "candidate language: go, python, or javascript")
	source := flag.String("source", "", "generated source file (.go, .py, or .js)")
	prompt := flag.String("prompt", "", "prompt file")
	context := flag.String("context", "", "context JSON file")
	out := flag.String("out", "", "new candidate directory")
	note := flag.String("note", "", "optional note")
	flag.Parse()
	if *id == "" || *provider == "" || *model == "" || *contractPath == "" || *source == "" || *prompt == "" || *context == "" || *out == "" {
		fail(fmt.Errorf("-id, -provider, -model, -contract, -source, -prompt, -context, and -out are required"))
	}
	manifest, err := generatorselection.Record(generatorselection.RecordOptions{
		ID: *id, Origin: *origin, Provider: *provider, Model: *model, Language: *language,
		ContractPath: *contractPath,
		SourcePath:   *source, PromptPath: *prompt, ContextPath: *context, Note: *note,
	}, *out)
	if err != nil {
		fail(err)
	}
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(manifest); err != nil {
		fail(err)
	}
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
