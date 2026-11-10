// A reference-only matrix projection for TEI plugin composition.
// It does not resolve providers, build code, execute tests, or certify evidence.
package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"
)

type Axis struct {
	Name    string   `json:"name"`
	Options []string `json:"options"`
}
type Selection struct {
	Axis  string `json:"axis"`
	Value string `json:"value"`
}
type Lock struct {
	Selections        []Selection `json:"selections"`
	Fingerprint       string      `json:"fingerprint"`
	SourceFingerprint string      `json:"sourceFingerprint"`
	ArtifactDigest    string      `json:"artifactDigest"`
	Target            string      `json:"target"`
}
type Check struct {
	LockFingerprint        string `json:"lockFingerprint"`
	CheckID                string `json:"checkId"`
	Status                 string `json:"status"`
	EvidenceRef            string `json:"evidenceRef"`
	IndependentExpectedRef string `json:"independentExpectedRef"`
}
type Input struct {
	Axes           []Axis   `json:"axes"`
	MaxCases       int      `json:"maxCases"`
	RequiredChecks []string `json:"requiredChecks"`
	Locks          []Lock   `json:"locks"`
	Checks         []Check  `json:"checks"`
}
type Row struct {
	CaseID          string      `json:"caseId"`
	Selections      []Selection `json:"selections"`
	LockFingerprint string      `json:"lockFingerprint,omitempty"`
	Status          string      `json:"status"`
	MissingChecks   []string    `json:"missingChecks,omitempty"`
}
type Output struct {
	ClaimCeiling   string `json:"claimCeiling"`
	Qualified      bool   `json:"qualified"`
	MatrixComplete bool   `json:"matrixComplete"`
	Rows           []Row  `json:"rows"`
}

func validToken(s string) bool { return s != "" && !strings.ContainsAny(s, "=|\n\r") }

func key(selections []Selection) string {
	p := make([]string, len(selections))
	for i, x := range selections {
		p[i] = x.Axis + "=" + x.Value
	}
	sort.Strings(p)
	return strings.Join(p, "|")
}

// Project only maps exact, already resolved locks to supplied test observations.
// In particular, 'observed-pass' is not a certified or currently authorized BOM.
func Project(input Input) (Output, error) {
	out := Output{ClaimCeiling: "reference-fixture-only", Qualified: false, Rows: []Row{}}
	if input.MaxCases < 1 || len(input.Axes) == 0 || len(input.RequiredChecks) == 0 {
		return out, errors.New("axes, requiredChecks and positive maxCases are required")
	}
	axes := make([]Axis, len(input.Axes))
	for i, axis := range input.Axes {
		axes[i] = Axis{Name: axis.Name, Options: append([]string(nil), axis.Options...)}
	}
	sort.Slice(axes, func(i, j int) bool { return axes[i].Name < axes[j].Name })
	knownAxes := map[string]map[string]bool{}
	total := 1
	for i := range axes {
		a := &axes[i]
		if !validToken(a.Name) || len(a.Options) == 0 || knownAxes[a.Name] != nil {
			return out, fmt.Errorf("invalid or duplicate axis %q", a.Name)
		}
		knownAxes[a.Name] = map[string]bool{}
		for _, v := range a.Options {
			if !validToken(v) || knownAxes[a.Name][v] {
				return out, fmt.Errorf("invalid or duplicate option %q", v)
			}
			knownAxes[a.Name][v] = true
		}
		sort.Strings(a.Options)
		if total > input.MaxCases/len(a.Options) {
			return out, errors.New("matrix exceeds explicit maxCases; narrow axes or use an external selection strategy")
		}
		total *= len(a.Options)
	}
	required := map[string]bool{}
	for _, x := range input.RequiredChecks {
		if !validToken(x) || required[x] {
			return out, fmt.Errorf("invalid or duplicate required check %q", x)
		}
		required[x] = true
	}
	validateSelection := func(selections []Selection) error {
		if len(selections) != len(axes) {
			return errors.New("lock selections must cover every matrix axis")
		}
		seen := map[string]bool{}
		for _, s := range selections {
			if seen[s.Axis] || !knownAxes[s.Axis][s.Value] {
				return errors.New("lock contains invalid or duplicated axis/value")
			}
			seen[s.Axis] = true
		}
		return nil
	}
	locks := map[string]Lock{}
	fpToKey := map[string]string{}
	for _, lock := range input.Locks {
		if err := validateSelection(lock.Selections); err != nil {
			return out, err
		}
		if !validToken(lock.Fingerprint) || lock.SourceFingerprint == "" || lock.ArtifactDigest == "" || lock.Target == "" {
			return out, errors.New("lock missing exact source/target/artifact identity")
		}
		k := key(lock.Selections)
		if _, exists := locks[k]; exists {
			return out, errors.New("duplicate exact lock for one combination")
		}
		if _, exists := fpToKey[lock.Fingerprint]; exists {
			return out, errors.New("same lock fingerprint bound to two combinations")
		}
		locks[k], fpToKey[lock.Fingerprint] = lock, k
	}
	checks := map[string]map[string]Check{}
	for _, c := range input.Checks {
		if _, exists := fpToKey[c.LockFingerprint]; !exists {
			return out, errors.New("check refers to an unknown exact lock")
		}
		if !required[c.CheckID] {
			return out, errors.New("check not in independently required set")
		}
		if c.Status != "pass" && c.Status != "fail" && c.Status != "skip" && c.Status != "unknown" {
			return out, errors.New("invalid check status")
		}
		if checks[c.LockFingerprint] == nil {
			checks[c.LockFingerprint] = map[string]Check{}
		}
		if _, exists := checks[c.LockFingerprint][c.CheckID]; exists {
			return out, errors.New("duplicate check observation")
		}
		checks[c.LockFingerprint][c.CheckID] = c
	}
	var walk func(int, []Selection)
	walk = func(i int, selected []Selection) {
		if i < len(axes) {
			for _, v := range axes[i].Options {
				walk(i+1, append(append([]Selection{}, selected...), Selection{Axis: axes[i].Name, Value: v}))
			}
			return
		}
		row := Row{CaseID: key(selected), Selections: append([]Selection{}, selected...), Status: "unresolved"}
		lock, exists := locks[row.CaseID]
		if !exists {
			out.Rows = append(out.Rows, row)
			return
		}
		row.LockFingerprint = lock.Fingerprint
		row.Status = "observed-pass"
		for _, requirement := range input.RequiredChecks {
			c, ok := checks[lock.Fingerprint][requirement]
			if !ok {
				row.MissingChecks = append(row.MissingChecks, requirement)
				if row.Status == "observed-pass" {
					row.Status = "not-run"
				}
				continue
			}
			if c.Status == "fail" {
				row.Status = "failed"
				continue
			}
			if c.Status != "pass" || c.EvidenceRef == "" || c.IndependentExpectedRef == "" {
				if row.Status != "failed" {
					row.Status = "unverified"
				}
			}
		}
		out.Rows = append(out.Rows, row)
	}
	walk(0, nil)
	out.MatrixComplete = true
	for _, row := range out.Rows {
		if row.Status != "observed-pass" {
			out.MatrixComplete = false
		}
	}
	return out, nil
}

func main() {
	input := flag.String("input", "", "read an exact-lock matrix fixture as JSON")
	strategy := flag.String("plan", "", "emit pairwise or supplied-locks-pairwise planning-only cases without qualifying evidence")
	flag.Parse()
	if *input == "" {
		fmt.Fprintln(os.Stderr, "-input is required")
		os.Exit(2)
	}
	data, err := os.ReadFile(*input)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	var in Input
	if err = json.Unmarshal(data, &in); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if *strategy == "pairwise" {
		result, err := PlanPairwise(in)
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
		return
	}
	if *strategy == "supplied-locks-pairwise" {
		result, err := PlanResolvedPairwise(in)
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
		return
	}
	if *strategy != "" {
		fmt.Fprintln(os.Stderr, "unsupported planning strategy; choose pairwise, supplied-locks-pairwise, or leave -plan unset")
		os.Exit(2)
	}
	result, err := Project(in)
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
}
