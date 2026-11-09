package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"

	generatorselection "tei/tools/reference-generator-selection"
)

// tei-generator-meta runs one externally supplied generator and records the
// candidate it produced. Only one generator level (depth 1) is allowed.
func main() {
	id := flag.String("id", "", "candidate id")
	provider := flag.String("provider", "", "model provider")
	model := flag.String("model", "", "model name or version")
	contractPath := flag.String("contract", "", "contract JSON")
	generatorSource := flag.String("generator", "", "generator source file (must write Go source to stdout)")
	prompt := flag.String("prompt", "", "prompt file")
	contextPath := flag.String("context", "", "context JSON")
	out := flag.String("out", "", "new candidate directory")
	note := flag.String("note", "", "optional note")
	timeout := flag.Duration("timeout", 5*time.Minute, "generator build/run timeout")
	flag.Parse()
	if *id == "" || *provider == "" || *model == "" || *contractPath == "" || *generatorSource == "" || *prompt == "" || *contextPath == "" || *out == "" {
		fail(fmt.Errorf("-id, -provider, -model, -contract, -generator, -prompt, -context, and -out are required"))
	}
	manifest, err := generatorselection.GenerateWithMetaGenerator(context.Background(), generatorselection.MetaOptions{
		ID: *id, Provider: *provider, Model: *model,
		ContractPath: *contractPath, GeneratorSourcePath: *generatorSource,
		PromptPath: *prompt, ContextPath: *contextPath,
		OutDir: *out, Note: *note, Timeout: *timeout,
	})
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
