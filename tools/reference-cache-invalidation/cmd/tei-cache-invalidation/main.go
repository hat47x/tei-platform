package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"tei/tools/reference-cache-invalidation"
)

func main() {
	inPath := ""
	for i := 1; i < len(os.Args); i++ {
		if os.Args[i] == "-input" && i+1 < len(os.Args) {
			inPath = os.Args[i+1]
		}
	}
	var r io.Reader = os.Stdin
	if inPath != "" && inPath != "-" {
		f, err := os.Open(inPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "open input: %v\n", err)
			os.Exit(2)
		}
		defer f.Close()
		r = f
	}
	var in struct {
		Previous            cacheinvalidation.CacheKey
		Current             cacheinvalidation.CacheKey
		Dependencies        []cacheinvalidation.DependencyEdge
		CurrentRevisions    []cacheinvalidation.Revision
		PreviousRevisions   []cacheinvalidation.Revision
		Evidence            cacheinvalidation.EvidenceRecord
		PreviousEvidence    cacheinvalidation.EvidenceRecord
		DependenciesChanged bool
		Status              cacheinvalidation.Status
	}
	if err := json.NewDecoder(r).Decode(&in); err != nil {
		fmt.Fprintf(os.Stderr, "decode input: %v\n", err)
		os.Exit(2)
	}
	engine := cacheinvalidation.NewEngine(in.Dependencies)
	prevClosure, err := engine.ComputeClosure(in.PreviousRevisions)
	if err != nil {
		fmt.Fprintf(os.Stderr, "previous closure: %v\n", err)
		os.Exit(2)
	}
	currClosure, err := engine.ComputeClosure(in.CurrentRevisions)
	if err != nil {
		fmt.Fprintf(os.Stderr, "current closure: %v\n", err)
		os.Exit(2)
	}
	if prevClosure.Digest != currClosure.Digest {
		fmt.Fprintf(os.Stderr, "closure digest mismatch: %s != %s\n", prevClosure.Digest, currClosure.Digest)
		os.Exit(2)
	}
	artifact := engine.ReuseArtifact(in.Previous, in.Current, currClosure)
	build := engine.ReuseBuild(in.Previous, in.Current, currClosure, in.DependenciesChanged, in.Status)
	evidence := engine.ReuseEvidence(in.PreviousEvidence, in.Evidence, currClosure, false)
	report := struct {
		Schema    string                          `json:"schema"`
		Authority string                          `json:"authority"`
		Artifact  cacheinvalidation.ReuseDecision `json:"artifact"`
		Build     cacheinvalidation.ReuseDecision `json:"build"`
		Evidence  cacheinvalidation.ReuseDecision `json:"evidence"`
		Closure   cacheinvalidation.Closure       `json:"closure"`
		Notes     []string                        `json:"notes"`
	}{
		Schema:    "tei.reference.cache-invalidation/v0",
		Authority: "derived-reference",
		Artifact:  artifact,
		Build:     build,
		Evidence:  evidence,
		Closure:   currClosure,
		Notes:     []string{"read-only reference; does not write sources or issue receipts"},
	}
	out, _ := json.MarshalIndent(report, "", "  ")
	fmt.Println(string(out))
	if !artifact.Allowed || !build.Allowed || !evidence.Allowed {
		os.Exit(1)
	}
}
