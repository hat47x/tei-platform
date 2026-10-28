// A read-only reference for selecting developer context around an existing TEI Go source.
package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const maxFileBytes int64 = 1 << 20
const maxCandidates = 12

type Candidate struct {
	Path               string             `json:"path"`
	Kind               string             `json:"kind"`
	SHA256             string             `json:"sha256"`
	Basis              string             `json:"basis"`
	BaselineRank       int                `json:"baselineRank"`
	ModelChoice        string             `json:"modelChoice,omitempty"`
	ModelProbabilities map[string]float64 `json:"modelProbabilities,omitempty"`
}

type Evaluation struct {
	State      string `json:"state"`
	Model      string `json:"model,omitempty"`
	Diagnostic string `json:"diagnostic,omitempty"`
}

type Report struct {
	Schema          string      `json:"schema"`
	Target          string      `json:"target"`
	TargetSHA256    string      `json:"targetSha256"`
	ReadOnly        bool        `json:"readOnly"`
	Accepted        bool        `json:"accepted"`
	GrantsAuthority bool        `json:"grantsAuthority"`
	Evaluation      Evaluation  `json:"evaluation"`
	Candidates      []Candidate `json:"candidates"`
}

func digest(b []byte) string {
	sum := sha256.Sum256(b)
	return "sha256:" + hex.EncodeToString(sum[:])
}

func safeRelative(p string) (string, error) {
	if p == "" || filepath.IsAbs(p) || filepath.VolumeName(p) != "" || strings.Contains(p, "\\") {
		return "", errors.New("target must be a nonempty workspace-relative slash path")
	}
	cleaned := filepath.Clean(p)
	if cleaned == "." || cleaned == ".." || strings.HasPrefix(cleaned, ".."+string(filepath.Separator)) || cleaned != p {
		return "", errors.New("target path is not normalized or escapes workspace")
	}
	return cleaned, nil
}

func readInside(root, rel string) ([]byte, error) {
	full := filepath.Join(root, filepath.FromSlash(rel))
	resolved, err := filepath.EvalSymlinks(full)
	if err != nil {
		return nil, err
	}
	inside, err := filepath.Rel(root, resolved)
	if err != nil || inside == ".." || strings.HasPrefix(inside, ".."+string(filepath.Separator)) {
		return nil, errors.New("path escapes workspace through symlink")
	}
	info, err := os.Lstat(full)
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() || info.Size() > maxFileBytes {
		return nil, errors.New("not a regular file or file too large")
	}
	return os.ReadFile(full)
}

func Discover(root, target string) (Report, error) {
	rel, err := safeRelative(target)
	if err != nil {
		return Report{}, err
	}
	if filepath.Ext(rel) != ".go" || strings.HasSuffix(rel, "_test.go") {
		return Report{}, errors.New("target must be a Go source file (not a test)")
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		return Report{}, err
	}
	abs, err = filepath.EvalSymlinks(abs)
	if err != nil {
		return Report{}, err
	}
	source, err := readInside(abs, rel)
	if err != nil {
		return Report{}, err
	}
	report := Report{Schema: "tei.reference.developer-context/v0", Target: filepath.ToSlash(rel),
		TargetSHA256: digest(source), ReadOnly: true, Evaluation: Evaluation{State: "baseline"},
		Candidates: []Candidate{}}
	dir := filepath.Dir(rel)
	stem := strings.TrimSuffix(filepath.Base(rel), ".go")
	entries, err := os.ReadDir(filepath.Join(abs, dir))
	if err != nil {
		return Report{}, err
	}
	for _, entry := range entries {
		if len(report.Candidates) >= maxCandidates {
			break
		}
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), "_test.go") {
			continue
		}
		path := filepath.ToSlash(filepath.Join(dir, entry.Name()))
		body, readErr := readInside(abs, path)
		if readErr != nil {
			continue
		}
		basis := "same-package-test"
		if strings.HasPrefix(entry.Name(), stem+"_") || strings.HasPrefix(entry.Name(), stem+"test") {
			basis = "filename-matched-test"
		} else if strings.Contains(string(body), stem) {
			basis = "content-matched-test"
		} else {
			continue
		}
		report.Candidates = append(report.Candidates, Candidate{Path: path, Kind: "test", SHA256: digest(body), Basis: basis})
	}
	sort.Slice(report.Candidates, func(i, j int) bool {
		if report.Candidates[i].Basis != report.Candidates[j].Basis {
			order := map[string]int{"filename-matched-test": 0, "content-matched-test": 1, "same-package-test": 2}
			return order[report.Candidates[i].Basis] < order[report.Candidates[j].Basis]
		}
		return report.Candidates[i].Path < report.Candidates[j].Path
	})
	// Only explicit source path/directory references count as document candidates.
	// WalkDir does not follow directory symlinks; keep both the scope and size bounded.
	for _, top := range []string{"docs", filepath.Join("plan", "adr")} {
		if len(report.Candidates) >= maxCandidates {
			break
		}
		base := filepath.Join(abs, top)
		if info, statErr := os.Lstat(base); statErr != nil || !info.IsDir() {
			continue
		}
		scanned := 0
		_ = filepath.WalkDir(base, func(path string, entry os.DirEntry, walkErr error) error {
			if walkErr != nil || len(report.Candidates) >= maxCandidates || scanned >= 1000 {
				return filepath.SkipAll
			}
			if entry.IsDir() {
				return nil
			}
			if !entry.Type().IsRegular() || filepath.Ext(path) != ".md" {
				return nil
			}
			scanned++
			item, relErr := filepath.Rel(abs, path)
			if relErr != nil {
				return nil
			}
			relPath := filepath.ToSlash(item)
			body, readErr := readInside(abs, relPath)
			if readErr != nil {
				return nil
			}
			text := string(body)
			if !strings.Contains(text, filepath.ToSlash(rel)) && !strings.Contains(text, filepath.ToSlash(dir)+"/") {
				return nil
			}
			report.Candidates = append(report.Candidates, Candidate{Path: relPath, Kind: "document", SHA256: digest(body), Basis: "explicit-path-reference"})
			return nil
		})
	}
	for i := range report.Candidates {
		report.Candidates[i].BaselineRank = i + 1
	}
	return report, nil
}

type decisionQuestion struct {
	Type         string            `json:"type"`
	Instructions string            `json:"instructions"`
	Criteria     map[string]string `json:"criteria"`
}
type decisionAnswer struct {
	Type          string             `json:"type"`
	Choice        string             `json:"choice"`
	Probabilities map[string]float64 `json:"probabilities"`
}
type decisionResponse struct {
	Answers map[string]decisionAnswer `json:"answers"`
}

func validateEndpoint(raw string) error {
	u, err := url.Parse(raw)
	if err != nil || u.Scheme != "http" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || u.Path != "/v1/systemone" {
		return errors.New("local endpoint must be http://127.0.0.1:<port>/v1/systemone (or [::1])")
	}
	host := u.Hostname()
	ip := net.ParseIP(host)
	if ip == nil || !ip.IsLoopback() || u.Port() == "" {
		return errors.New("endpoint must use a literal loopback IP and port")
	}
	return nil
}

// Rerank only changes display order. Invalid or unavailable model responses
// leave every deterministic candidate in its original order.
func Rerank(ctx context.Context, report *Report, endpoint, model string) {
	report.Evaluation = Evaluation{State: "model-failed", Model: model}
	if err := validateEndpoint(endpoint); err != nil {
		report.Evaluation.Diagnostic = err.Error()
		return
	}
	if strings.TrimSpace(model) == "" {
		report.Evaluation.Diagnostic = "model identifier required"
		return
	}
	if len(report.Candidates) == 0 {
		report.Evaluation.State = "no-candidates"
		return
	}
	state := map[string]any{"target": report.Target, "targetSha256": report.TargetSHA256,
		"candidates": report.Candidates, "scope": "Read-only candidate relevance, not test success, meaning, or authority"}
	questions := make(map[string]decisionQuestion, len(report.Candidates))
	for i, c := range report.Candidates {
		questions[fmt.Sprintf("c%d", i)] = decisionQuestion{Type: "choice",
			Instructions: "Is inspecting " + c.Path + " useful when changing " + report.Target + "? Treat uncertain links as uncertain.",
			Criteria:     map[string]string{"relevant": "Directly useful to inspect", "uncertain": "Insufficient evidence of relevance", "not_relevant": "Not useful for this change"}}
	}
	request, err := json.Marshal(map[string]any{"model": model, "state": state, "questions": questions})
	if err != nil {
		report.Evaluation.Diagnostic = err.Error()
		return
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(request))
	if err != nil {
		report.Evaluation.Diagnostic = err.Error()
		return
	}
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: 5 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	resp, err := client.Do(req)
	if err != nil {
		report.Evaluation.Diagnostic = err.Error()
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		report.Evaluation.Diagnostic = fmt.Sprintf("local model HTTP %d", resp.StatusCode)
		return
	}
	limited, err := io.ReadAll(io.LimitReader(resp.Body, 256*1024+1))
	if err != nil || len(limited) > 256*1024 {
		report.Evaluation.Diagnostic = "invalid or oversized model response"
		return
	}
	var result decisionResponse
	if err := json.Unmarshal(limited, &result); err != nil {
		report.Evaluation.Diagnostic = "invalid response JSON"
		return
	}
	for i := range report.Candidates {
		ans, ok := result.Answers[fmt.Sprintf("c%d", i)]
		if !ok || ans.Type != "choice" || (ans.Choice != "relevant" && ans.Choice != "uncertain" && ans.Choice != "not_relevant") {
			report.Evaluation.Diagnostic = "missing or invalid typed decision"
			return
		}
		sum := 0.0
		for _, key := range []string{"relevant", "uncertain", "not_relevant"} {
			prob, exists := ans.Probabilities[key]
			if !exists || prob < 0 || prob > 1 {
				report.Evaluation.Diagnostic = "invalid option probabilities"
				return
			}
			sum += prob
		}
		if len(ans.Probabilities) != 3 || sum < 0.98 || sum > 1.02 {
			report.Evaluation.Diagnostic = "option probabilities do not sum to one"
			return
		}
	}
	for i := range report.Candidates {
		ans := result.Answers[fmt.Sprintf("c%d", i)]
		report.Candidates[i].ModelChoice = ans.Choice
		report.Candidates[i].ModelProbabilities = ans.Probabilities
	}
	rank := map[string]int{"relevant": 0, "uncertain": 1, "not_relevant": 2}
	sort.SliceStable(report.Candidates, func(i, j int) bool {
		a, b := report.Candidates[i], report.Candidates[j]
		if rank[a.ModelChoice] != rank[b.ModelChoice] {
			return rank[a.ModelChoice] < rank[b.ModelChoice]
		}
		if a.ModelProbabilities["relevant"] != b.ModelProbabilities["relevant"] {
			return a.ModelProbabilities["relevant"] > b.ModelProbabilities["relevant"]
		}
		return a.BaselineRank < b.BaselineRank
	})
	report.Evaluation = Evaluation{State: "model-ranked", Model: model}
}

func main() {
	root := flag.String("root", ".", "repository root (read-only)")
	target := flag.String("target", "", "existing Go source path relative to root")
	mode := flag.String("mode", "baseline", "baseline or systemone")
	endpoint := flag.String("endpoint", "", "local model endpoint; loopback IP only")
	model := flag.String("model", "", "exact local model ID")
	flag.Parse()
	if *mode != "baseline" && *mode != "systemone" {
		fmt.Fprintln(os.Stderr, "mode must be baseline or systemone")
		os.Exit(2)
	}
	report, err := Discover(*root, *target)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	if *mode == "systemone" {
		Rerank(context.Background(), &report, *endpoint, *model)
	}
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(report); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
