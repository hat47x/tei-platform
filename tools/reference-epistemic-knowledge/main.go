package main

import (
	"flag"
	"fmt"
	"io"
	"os"
)

const usage = `usage:
  reference-epistemic-knowledge -corpus FILE -context FILE [-out FILE]
`

func main() {
	os.Exit(runCLI(os.Args[1:], os.Stdout, os.Stderr))
}

func runCLI(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("reference-epistemic-knowledge", flag.ContinueOnError)
	flags.SetOutput(stderr)
	corpusPath := flags.String("corpus", "", "epistemic corpus JSON")
	contextPath := flags.String("context", "", "current context JSON")
	outPath := flags.String("out", "", "output JSON file; stdout when omitted")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if *corpusPath == "" || *contextPath == "" {
		fmt.Fprint(stderr, usage)
		return 2
	}

	corpusData, err := os.ReadFile(*corpusPath)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	contextData, err := os.ReadFile(*contextPath)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	corpus, err := decodeStrict[Corpus](corpusData)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	context, err := decodeStrict[ContextQuery](contextData)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	projection, err := evaluateCorpus(corpus, context)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	encoded, err := jsonBytes(projection)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	if *outPath == "" {
		_, _ = stdout.Write(encoded)
		return 0
	}
	if err := os.WriteFile(*outPath, encoded, 0o644); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	return 0
}
