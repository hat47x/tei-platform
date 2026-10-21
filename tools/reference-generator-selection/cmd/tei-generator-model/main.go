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

// tei-generator-model runs one trusted external generation command and records
// the source it writes. It does not embed a vendor API or a model credential.
func main() {
	id := flag.String("id", "", "candidate id")
	provider := flag.String("provider", "", "model provider")
	model := flag.String("model", "", "model name or version")
	contractPath := flag.String("contract", "", "generator contract JSON")
	language := flag.String("language", "go", "candidate language: go, python, javascript, or java")
	prompt := flag.String("prompt", "", "prompt file")
	contextPath := flag.String("context", "", "context JSON file")
	out := flag.String("out", "", "new candidate directory")
	command := flag.String("command", "", "trusted shell command; write source to $TEI_SOURCE")
	timeout := flag.Duration("timeout", 10*time.Minute, "command timeout")
	flag.Parse()
	if *id == "" || *provider == "" || *model == "" || *contractPath == "" || *prompt == "" || *contextPath == "" || *out == "" || *command == "" {
		fail(fmt.Errorf("-id, -provider, -model, -contract, -prompt, -context, -out, and -command are required"))
	}
	manifest, err := generatorselection.GenerateWithCommand(context.Background(), generatorselection.ModelOptions{
		ID: *id, Provider: *provider, Model: *model, Language: *language, ContractPath: *contractPath,
		PromptPath: *prompt, ContextPath: *contextPath, OutDir: *out,
		Command: *command, Timeout: *timeout,
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
