// Command reference-harness-wire-summary turns captured mock-provider request
// bodies into structural summaries.
//
// It reads only files the caller points at, writes only to the named output
// directory, and never sends anything itself. A summary shows what a request
// carried; it does not decide whether the disclosure was permitted.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"tei/tools/internal/harnesswire"
)

type document struct {
	Note    string   `json:"note"`
	Markers []string `json:"markers"`
	// ToolNames is every tool name advertised by at least one request. It is
	// not a list of tools that ran.
	ToolNames []string            `json:"toolNames"`
	Requests  []requestFile       `json:"requests"`
	Totals    map[string]int      `json:"totals"`
	Limits    map[string][]string `json:"limits"`
}

type requestFile struct {
	File    string              `json:"file"`
	Summary harnesswire.Summary `json:"summary"`
	Problem string              `json:"problem,omitempty"`
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	fs := flag.NewFlagSet("reference-harness-wire-summary", flag.ContinueOnError)
	requestsDir := fs.String("requests", "", "directory containing captured *-request.json files")
	markerFlag := fs.String("markers", "", "comma-separated synthetic markers to count")
	outPath := fs.String("out", "", "output JSON path (must not exist)")
	args := os.Args[1:]
	if len(args) > 0 && args[0] == "probe" {
		args = args[1:]
	}
	if err := fs.Parse(args); err != nil {
		return fmt.Errorf("parse flags: %w", err)
	}

	if *requestsDir == "" || *outPath == "" {
		return fmt.Errorf("both -requests and -out are required")
	}
	if _, err := os.Stat(*requestsDir); err != nil {
		return fmt.Errorf("read requests: %w", err)
	}
	if _, err := os.Stat(*outPath); err == nil {
		return fmt.Errorf("output %s already exists; removable output only", *outPath)
	}
	if err := os.MkdirAll(filepath.Dir(*outPath), 0o755); err != nil {
		return fmt.Errorf("prepare output directory: %w", err)
	}

	markers := splitMarkers(*markerFlag)
	if len(markers) == 0 {
		return fmt.Errorf("-markers needs at least one value")
	}

	entries, err := os.ReadDir(*requestsDir)
	if err != nil {
		return fmt.Errorf("list requests: %w", err)
	}

	doc := document{
		Note:    "Structural summary of captured request bodies. It shows what was carried, not whether it was permitted.",
		Markers: markers,
		Totals: map[string]int{
			"requests": 0, "withSessionLog": 0, "withMarkerInModelInput": 0, "withMarkerInSessionLog": 0,
			"withAdvertisedTools": 0,
		},
		Limits: map[string][]string{
			"notProven": {
				"that the harness sent nothing else on other paths (telemetry, MCP, subagents)",
				"that a captured field was permitted to leave the host",
				"that the provider stored, retained or reused the body",
				"that an advertised tool name was ever invoked",
			},
		},
	}

	seen := map[string]bool{}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".json" {
			continue
		}
		names = append(names, e.Name())
	}
	sort.Strings(names)

	for _, name := range names {
		path := filepath.Join(*requestsDir, name)
		body, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read %s: %w", name, err)
		}
		sum, sumErr := harnesswire.Summarize(body, markers)
		entry := requestFile{File: name, Summary: sum}
		if sumErr != nil {
			entry.Problem = sumErr.Error()
		}
		doc.Requests = append(doc.Requests, entry)
		doc.Totals["requests"]++
		if sum.SessionLog != nil {
			doc.Totals["withSessionLog"]++
		}
		if len(sum.ModelInputHits) > 0 {
			doc.Totals["withMarkerInModelInput"]++
		}
		if sum.SessionLog != nil && len(sum.SessionLog.MarkerHits) > 0 {
			doc.Totals["withMarkerInSessionLog"]++
		}
		if len(sum.ToolNames) > 0 {
			doc.Totals["withAdvertisedTools"]++
			for _, name := range sum.ToolNames {
				seen[name] = true
			}
		}
	}

	doc.ToolNames = sortedSeen(seen)
	encoded, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal summary: %w", err)
	}
	encoded = append(encoded, '\n')
	if err := os.WriteFile(*outPath, encoded, 0o644); err != nil {
		return fmt.Errorf("write summary: %w", err)
	}
	fmt.Fprintf(os.Stderr, "wrote %s (%d requests)\n", *outPath, doc.Totals["requests"])
	return nil
}

func splitMarkers(raw string) []string {
	if raw == "" {
		return nil
	}
	var out []string
	start := 0
	for i := 0; i <= len(raw); i++ {
		if i == len(raw) || raw[i] == ',' {
			if value := trim(raw[start:i]); value != "" {
				out = append(out, value)
			}
			start = i + 1
		}
	}
	return out
}

func sortedSeen(seen map[string]bool) []string {
	out := make([]string, 0, len(seen))
	for name := range seen {
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}

func trim(s string) string {
	start, end := 0, len(s)
	for start < end && (s[start] == ' ' || s[start] == '\t' || s[start] == '\n' || s[start] == '\r') {
		start++
	}
	for end > start && (s[end-1] == ' ' || s[end-1] == '\t' || s[end-1] == '\n' || s[end-1] == '\r') {
		end--
	}
	return s[start:end]
}
