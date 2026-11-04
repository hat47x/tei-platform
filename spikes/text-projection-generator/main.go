package main

// Spike: a text/record projection contract, evaluated by two fixed strategies.
// It exists to test whether the existing generator-selection idea can carry a
// non-numeric projection. It is not a product contract and does not change the
// reference-generator-selection tool.

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"runtime"
	"strings"
	"time"
)

type typeSpec struct {
	Primitive string `json:"primitive"`
}

type input struct {
	Attribute string `json:"attribute"`
	Primitive string `json:"primitive"`
}

type contract struct {
	Schema     string   `json:"schema"`
	ID         string   `json:"id"`
	Version    string   `json:"version"`
	Status     string   `json:"status"`
	Inputs     []input  `json:"inputs"`
	Output     typeSpec `json:"output"`
	Projection struct {
		Kind string `json:"kind"`
	} `json:"projection"`
}

type testCase struct {
	ID    string         `json:"id"`
	Input map[string]any `json:"input"`
	Want  map[string]any `json:"want"`
}

type fixture struct {
	Schema          string     `json:"schema"`
	ContractID      string     `json:"contractId"`
	ContractVersion string     `json:"contractVersion"`
	Cases           []testCase `json:"cases"`
}

type result struct {
	CaseID string         `json:"caseId"`
	Want   map[string]any `json:"want"`
	Got    map[string]any `json:"got"`
	Passed bool           `json:"passed"`
}

type receipt struct {
	Schema          string    `json:"schema"`
	ContractID      string    `json:"contractId"`
	ContractVersion string    `json:"contractVersion"`
	ContractDigest  string    `json:"contractDigest"`
	FixturesDigest  string    `json:"fixturesDigest"`
	ProjectionKind  string    `json:"projectionKind"`
	Results         []result  `json:"results"`
	Passed          bool      `json:"passed"`
	GeneratedAt     time.Time `json:"generatedAt"`
	GoVersion       string    `json:"goVersion"`
	GOOS            string    `json:"goos"`
	GOARCH          string    `json:"goarch"`
}

func digest(raw []byte) string {
	sum := sha256.Sum256(raw)
	return "sha256:" + hex.EncodeToString(sum[:])
}

func readJSON(path string, target any) ([]byte, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(raw, target); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	return raw, nil
}

// project applies the fixed strategy. Missing, empty and unresolved are kept
// distinct; an empty string is present and empty, not missing.
func project(kind string, inputMap map[string]any, key string) (map[string]any, error) {
	value, ok := inputMap[key]
	if !ok {
		return map[string]any{"state": "missing"}, nil
	}
	if value == nil {
		return map[string]any{"state": "unresolved"}, nil
	}
	text, ok := value.(string)
	if !ok {
		return nil, fmt.Errorf("input %s is not a string", key)
	}
	if kind == "normalize" {
		text = strings.TrimSpace(text)
	}
	return map[string]any{"state": "present", "text": text}, nil
}

func equal(a, b map[string]any) bool {
	if len(a) != len(b) {
		return false
	}
	for key, value := range a {
		other, ok := b[key]
		if !ok || fmt.Sprint(other) != fmt.Sprint(value) {
			return false
		}
	}
	return true
}

func main() {
	contractPath := flag.String("contract", "", "projection contract JSON")
	fixturePath := flag.String("fixtures", "", "fixture JSON")
	outPath := flag.String("out", "", "write the receipt to this file")
	flag.Parse()
	if *contractPath == "" || *fixturePath == "" {
		fmt.Fprintln(os.Stderr, "-contract and -fixtures are required")
		os.Exit(2)
	}

	var c contract
	contractRaw, err := readJSON(*contractPath, &c)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	if c.Projection.Kind != "identity" && c.Projection.Kind != "normalize" {
		fmt.Fprintf(os.Stderr, "unsupported projection kind %q\n", c.Projection.Kind)
		os.Exit(2)
	}
	if len(c.Inputs) != 1 {
		fmt.Fprintln(os.Stderr, "the spike expects exactly one input attribute")
		os.Exit(2)
	}
	key := strings.TrimPrefix(c.Inputs[0].Attribute, "sui.card.")
	if key != "text" {
		fmt.Fprintln(os.Stderr, "the spike expects the attribute sui.card.text")
		os.Exit(2)
	}

	var f fixture
	fixtureRaw, err := readJSON(*fixturePath, &f)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	if f.ContractID != c.ID || f.ContractVersion != c.Version {
		fmt.Fprintln(os.Stderr, "fixture contract identity does not match the contract")
		os.Exit(2)
	}

	r := receipt{
		Schema: "tei.spike.text-projection-selection/v0", ContractID: c.ID,
		ContractVersion: c.Version, ContractDigest: digest(contractRaw),
		FixturesDigest: digest(fixtureRaw), ProjectionKind: c.Projection.Kind,
		Passed: true, GeneratedAt: time.Now().UTC(), GoVersion: runtime.Version(),
		GOOS: runtime.GOOS, GOARCH: runtime.GOARCH,
	}
	for _, tc := range f.Cases {
		got, err := project(c.Projection.Kind, tc.Input, key)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(2)
		}
		passed := equal(got, tc.Want)
		if !passed {
			r.Passed = false
		}
		r.Results = append(r.Results, result{CaseID: tc.ID, Want: tc.Want, Got: got, Passed: passed})
	}

	encoded, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	encoded = append(encoded, '\n')
	if *outPath != "" {
		if err := os.WriteFile(*outPath, encoded, 0o644); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(2)
		}
	}
	os.Stdout.Write(encoded)
	if !r.Passed {
		os.Exit(1)
	}
}
