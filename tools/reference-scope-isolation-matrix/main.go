// Command reference-scope-isolation-matrix は、SUI の scope isolation 要件
// （ADR-0059 implementation gate 5 の14経路）を言語非依存の機械可読な行列として
// 検査する。
//
// この tool は Product runtime でも認可の正本でもない。検査するのは行列の
// 形（経路の網羅・coverage の状態・根拠の有無）だけで、実装の越境遮断その
// ものを主張しない。`-sui-repo` を渡した場合だけ、行列が指す試験名とファイル
// が実在することを確かめる（CI に SUI の checkout が無いときはこの照合を飛ばす）。
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const (
	// requirementIDs は ADR-0059 gate 5 が列挙する経路の識別子。行列はこの集合を
	// 一つも欠かしてはならない（欠落は「該当機能が無い」とも書けない）。
	// 識別子は小文字で持ち、要件文の表記ゆれ（GET/MCP/agent）と対応させる。
	requirementIDs = "get,put,list,search,count,export,share,import,mcp,webhook,job,audit,cache,agent"
	// requirementText は要件の原文（ADR-0059 gate 5）。行列表記との対応を
	// 記録しておき、両者がずれたら人が気づけるようにする。
	requirementText = "GET,PUT,list,search,count,export,share,import,MCP,webhook,job,audit,cache,agent"
	schemaVersion   = "tei.conformance.scope-isolation-matrix/v1"
	// TEI authority は definition/checker のみ。実装の越境遮断を主張しない。
	teiAuthority = "checker-only"
)

type testRef struct {
	Repo string `json:"repo"`
	File string `json:"file"`
	Name string `json:"name"`
}

type pathEntry struct {
	ID     string    `json:"id"`
	Route  string    `json:"route"`
	Status string    `json:"status"`
	Reason string    `json:"reason"`
	Owner  string    `json:"owner"`
	Note   string    `json:"note"`
	Tests  []testRef `json:"tests"`
}

type matrix struct {
	Schema string `json:"schema"`
	Source struct {
		Requirement         string `json:"requirement"`
		RequirementPath     string `json:"requirementPath"`
		RequirementText     string `json:"requirementText"`
		Authority           string `json:"authority"`
		LanguageIndependent bool   `json:"languageIndependent"`
	} `json:"source"`
	Paths      []pathEntry `json:"paths"`
	Extensions []pathEntry `json:"extensions"`
}

type finding struct {
	Path   string `json:"path"`
	Detail string `json:"detail"`
}

type report struct {
	SchemaVersion string    `json:"schemaVersion"`
	MatrixSchema  string    `json:"matrixSchema"`
	TEIAuthority  string    `json:"teiAuthority"`
	RequirementOK bool      `json:"requirementComplete"`
	PathsCount    int       `json:"pathCount"`
	Covered       int       `json:"covered"`
	NotApplicable int       `json:"notApplicable"`
	Uncovered     int       `json:"uncovered"`
	TestRefs      int       `json:"testRefs"`
	VerifiedRefs  int       `json:"verifiedTestRefs"`
	Extensions    int       `json:"extensionCount"`
	Findings      []finding `json:"findings"`
	VerifiedTests bool      `json:"verifiedAgainstSuiRepo"`
}

func main() {
	matrixPath := flag.String("matrix", "reference/sui-adoption/scope-isolation-matrix.v1.json", "conformance matrix")
	suiRepo := flag.String("sui-repo", "", "optional SUI checkout; when set, test names are verified against it")
	flag.Parse()

	result, err := run(*matrixPath, *suiRepo)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(result); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if len(result.Findings) != 0 {
		os.Exit(1)
	}
}

func run(matrixPath, suiRepo string) (report, error) {
	raw, err := os.ReadFile(matrixPath)
	if err != nil {
		return report{}, fmt.Errorf("matrix を読めません: %w", err)
	}
	var m matrix
	if err := json.Unmarshal(raw, &m); err != nil {
		return report{}, fmt.Errorf("matrix が JSON として不正です: %w", err)
	}
	return check(m, suiRepo)
}

func check(m matrix, suiRepo string) (report, error) {
	out := report{SchemaVersion: schemaVersion, MatrixSchema: m.Schema, TEIAuthority: teiAuthority}
	if m.Schema != schemaVersion {
		out.Findings = append(out.Findings, finding{Path: "schema", Detail: "schema must be " + schemaVersion})
	}
	if !m.Source.LanguageIndependent {
		out.Findings = append(out.Findings, finding{Path: "source.languageIndependent", Detail: "must be true: the matrix is language independent"})
	}
	if strings.TrimSpace(m.Source.RequirementText) == "" || strings.TrimSpace(m.Source.RequirementPath) == "" {
		out.Findings = append(out.Findings, finding{Path: "source", Detail: "requirement and its path must be recorded"})
	}

	byID := map[string]pathEntry{}
	for _, p := range m.Paths {
		if p.ID == "" {
			out.Findings = append(out.Findings, finding{Path: "paths", Detail: "an entry has an empty id"})
			continue
		}
		if _, dup := byID[p.ID]; dup {
			out.Findings = append(out.Findings, finding{Path: p.ID, Detail: "duplicate path id"})
		}
		byID[p.ID] = p
	}
	need := strings.Split(requirementIDs, ",")
	for _, id := range need {
		if _, ok := byID[id]; !ok {
			out.Findings = append(out.Findings, finding{Path: id, Detail: "requirement path is missing from the matrix"})
		}
	}
	out.RequirementOK = len(out.Findings) == 0

	for _, p := range m.Paths {
		checkEntry(p, &out)
	}
	for _, p := range m.Extensions {
		checkEntry(p, &out)
	}
	out.PathsCount = len(m.Paths)
	out.Extensions = len(m.Extensions)

	// 試験名の実在照合。SUI の checkout が渡されたときだけ行う。
	if suiRepo != "" {
		out.VerifiedTests = true
		for _, p := range append(append([]pathEntry{}, m.Paths...), m.Extensions...) {
			for _, t := range p.Tests {
				if err := verifyTestRef(suiRepo, t); err != nil {
					out.Findings = append(out.Findings, finding{Path: p.ID, Detail: err.Error()})
					continue
				}
				out.VerifiedRefs++
			}
		}
	}
	sort.Slice(out.Findings, func(i, j int) bool {
		if out.Findings[i].Path == out.Findings[j].Path {
			return out.Findings[i].Detail < out.Findings[j].Detail
		}
		return out.Findings[i].Path < out.Findings[j].Path
	})
	return out, nil
}

func checkEntry(p pathEntry, out *report) {
	out.TestRefs += len(p.Tests)
	switch p.Status {
	case "covered":
		out.Covered++
		if len(p.Tests) == 0 {
			out.Findings = append(out.Findings, finding{Path: p.ID, Detail: "covered needs at least one test reference"})
		}
		for _, t := range p.Tests {
			if t.Repo == "" || t.File == "" || t.Name == "" {
				out.Findings = append(out.Findings, finding{Path: p.ID, Detail: "test reference needs repo, file and name"})
			}
		}
	case "not-applicable":
		out.NotApplicable++
		if strings.TrimSpace(p.Reason) == "" {
			out.Findings = append(out.Findings, finding{Path: p.ID, Detail: "not-applicable needs a reason"})
		}
		if len(p.Tests) != 0 {
			out.Findings = append(out.Findings, finding{Path: p.ID, Detail: "not-applicable must not carry test references"})
		}
	case "uncovered":
		out.Uncovered++
		if strings.TrimSpace(p.Reason) == "" || strings.TrimSpace(p.Owner) == "" {
			out.Findings = append(out.Findings, finding{Path: p.ID, Detail: "uncovered needs a reason and an owner"})
		}
		if len(p.Tests) != 0 {
			out.Findings = append(out.Findings, finding{Path: p.ID, Detail: "uncovered must not claim test references"})
		}
	default:
		out.Findings = append(out.Findings, finding{Path: p.ID, Detail: "status must be covered, not-applicable or uncovered"})
	}
}

// verifyTestRef checks that the named file exists in the SUI checkout and that
// the file mentions the test function. A rename on the SUI side must fail here
// rather than silently leaving the matrix stale.
func verifyTestRef(suiRepo string, t testRef) error {
	if t.Repo != "sui-sensemaking" {
		return fmt.Errorf("test repo %q is not checked out here", t.Repo)
	}
	path := filepath.Join(suiRepo, filepath.FromSlash(t.File))
	content, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("test file %s is not in the SUI checkout: %w", t.File, err)
	}
	if !strings.Contains(string(content), t.Name) {
		return fmt.Errorf("test %s is not found in %s (renamed?)", t.Name, t.File)
	}
	return nil
}
