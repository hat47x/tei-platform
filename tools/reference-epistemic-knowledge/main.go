package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
)

const usage = `usage:
  reference-epistemic-knowledge -corpus FILE -context FILE [-out FILE]
  reference-epistemic-knowledge -corpus FILE -context FILE -review-command FILE [-out FILE]
  reference-epistemic-knowledge -corpus FILE -context FILE -detail-query FILE [-detail-access FILE] [-out FILE]
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
	reviewCommandPath := flags.String("review-command", "", "optional human review mutation command JSON")
	detailQueryPath := flags.String("detail-query", "", "optional epistemic detail query JSON")
	detailAccessPath := flags.String("detail-access", "", "optional detail access context JSON; requires detail-query")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if *corpusPath == "" || *contextPath == "" {
		fmt.Fprint(stderr, usage)
		return 2
	}
	if *reviewCommandPath != "" && *detailQueryPath != "" {
		fmt.Fprintln(stderr, "review-command and detail-query are mutually exclusive")
		fmt.Fprint(stderr, usage)
		return 2
	}
	if *detailAccessPath != "" && *detailQueryPath == "" {
		fmt.Fprintln(stderr, "detail-access requires detail-query")
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
	var output any
	switch {
	case *reviewCommandPath != "":
		commandData, err := os.ReadFile(*reviewCommandPath)
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		command, err := decodeStrict[ReviewCommand](commandData)
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		result, err := applyReviewCommand(corpus, context, command)
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		output = result
	case *detailQueryPath != "":
		queryData, err := os.ReadFile(*detailQueryPath)
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		query, err := decodeStrict[DetailQuery](queryData)
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		access := TransportAccess{}
		if *detailAccessPath != "" {
			accessData, err := os.ReadFile(*detailAccessPath)
			if err != nil {
				fmt.Fprintln(stderr, err)
				return 1
			}
			access, err = decodeStrict[TransportAccess](accessData)
			if err != nil {
				fmt.Fprintln(stderr, err)
				return 1
			}
		}
		result, err := resolveEpistemicDetail(corpus, context, query, access)
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		output = result
	default:
		projection, err := evaluateCorpus(corpus, context)
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		output = projection
	}
	encoded, err := jsonBytes(output)
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

func jsonBytes(value any) ([]byte, error) {
	return json.MarshalIndent(value, "", "  ")
}
