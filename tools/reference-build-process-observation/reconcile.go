package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"
)

const (
	scopeSchema  = "tei.reference.build-observation-scope/v0"
	ledgerSchema = "tei.reference.build-observation-ledger/v0"

	absenceHolds     = "holds"
	absenceNotProven = "not-proven"

	roleUndeclared  = "undeclared"
	roleOSInterface = "os-interface"

	findingLimit = 40
)

// Scope is what one build attempt declared it would run and touch. It is
// written from the declaration side (a plan, an entry's report, a caller), never
// from the observation it is compared with.
type Scope struct {
	Schema string `json:"schema"`
	// Images are programs named by digest: the entry, the selected builder.
	Images []ScopeImage `json:"images"`
	// ImageRoots are closures named as a whole, such as a toolchain. Programs
	// under a root are covered as members of that closure; what they do inside
	// is not described by the declaration.
	ImageRoots []ScopeRoot `json:"imageRoots,omitempty"`
	Paths      []ScopePath `json:"paths"`
	// OSInterface lists paths every process touches because of the operating
	// system and the language runtime. They are covered as opaque, not as inputs
	// the build chose.
	OSInterface []string `json:"osInterface,omitempty"`
}

type ScopeImage struct {
	Role   string `json:"role"`
	Digest string `json:"digest"`
	// Source says who named this image: the entry's report, or the caller.
	Source string `json:"source,omitempty"`
}

type ScopeRoot struct {
	Role   string `json:"role"`
	Root   string `json:"root"`
	Source string `json:"source,omitempty"`
}

type ScopePath struct {
	Role string `json:"role"`
	Root string `json:"root"`
	// Write allows opens for writing under the root.
	Write bool `json:"write,omitempty"`
	// OwnerStep, when not zero, is the one direct step (1 = the first child the
	// command started) allowed to touch the root. The command itself may read it.
	OwnerStep int    `json:"ownerStep,omitempty"`
	Source    string `json:"source,omitempty"`
}

type Ledger struct {
	Schema      string        `json:"schema"`
	FilesTraced bool          `json:"filesTraced"`
	ExitCode    int           `json:"exitCode"`
	Steps       []LedgerStep  `json:"steps"`
	Execs       LedgerExecs   `json:"execs"`
	Files       LedgerFiles   `json:"files"`
	Absence     LedgerAbsence `json:"boundedAbsence"`
	NotObserved []string      `json:"notObserved"`
}

type LedgerStep struct {
	Ordinal     int    `json:"ordinal"`
	Seq         int    `json:"seq"`
	Image       string `json:"image"`
	ImageDigest string `json:"imageDigest"`
	Role        string `json:"role"`
	Execs       int    `json:"execs"`
}

type LedgerExecs struct {
	Total      int            `json:"total"`
	ByRole     map[string]int `json:"byRole"`
	Undeclared []ExecFinding  `json:"undeclared,omitempty"`
	// UndeclaredTotal counts every undeclared exec; Undeclared may be cut short.
	UndeclaredTotal int `json:"undeclaredTotal"`
}

type ExecFinding struct {
	Seq         int    `json:"seq"`
	Step        int    `json:"step"`
	Image       string `json:"image"`
	ImageDigest string `json:"imageDigest"`
	Reason      string `json:"reason"`
}

type RoleUse struct {
	Paths int `json:"paths"`
	Read  int `json:"read"`
	Write int `json:"write"`
}

type LedgerFiles struct {
	ByRole map[string]RoleUse `json:"byRole"`
	// Undeclared are successful opens the scope does not cover.
	Undeclared      []FileFinding `json:"undeclared,omitempty"`
	UndeclaredTotal int           `json:"undeclaredTotal"`
	// CrossStep are opens of a root owned by another step.
	CrossStep      []FileFinding `json:"crossStep,omitempty"`
	CrossStepTotal int           `json:"crossStepTotal"`
	// WriteToReadOnly are write opens under a root declared for reading.
	WriteToReadOnly      []FileFinding `json:"writeToReadOnly,omitempty"`
	WriteToReadOnlyTotal int           `json:"writeToReadOnlyTotal"`
	// ProbesOutsideScope counts opens that failed outside the scope: the process
	// asked and got nothing. They are not inputs, but a later file there would be.
	ProbesOutsideScope int `json:"probesOutsideScope"`
}

type FileFinding struct {
	Path  string `json:"path"`
	Step  int    `json:"step"`
	Image string `json:"image"`
	Read  int    `json:"read"`
	Write int    `json:"write"`
	Role  string `json:"role,omitempty"`
}

type LedgerAbsence struct {
	// State is "holds" only when, inside the traced process tree, nothing ran
	// and nothing was opened outside the declared scope.
	State   string   `json:"state"`
	Reasons []string `json:"reasons,omitempty"`
	Scope   string   `json:"scope"`
}

func reconcileCommand(args []string) int {
	flags := flag.NewFlagSet("reconcile", flag.ContinueOnError)
	observationPath := flags.String("observation", "", "observation record written by observe")
	scopePath := flags.String("scope", "", "declared scope")
	out := flags.String("out", "", "file to write the ledger to (default: stdout)")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if *observationPath == "" || *scopePath == "" {
		fmt.Fprintln(os.Stderr, "usage: reference-build-process-observation reconcile -observation record.json -scope scope.json [-out ledger.json]")
		return 2
	}
	var record Record
	if err := readJSON(*observationPath, &record); err != nil {
		fmt.Fprintln(os.Stderr, "read the observation:", err)
		return 2
	}
	var scope Scope
	if err := readJSON(*scopePath, &scope); err != nil {
		fmt.Fprintln(os.Stderr, "read the scope:", err)
		return 2
	}
	ledger, err := Reconcile(record, scope)
	if err != nil {
		fmt.Fprintln(os.Stderr, "reconcile:", err)
		return 2
	}
	encoded, err := json.MarshalIndent(ledger, "", "  ")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 2
	}
	encoded = append(encoded, '\n')
	if *out == "" {
		os.Stdout.Write(encoded)
	} else if err := os.WriteFile(*out, encoded, 0o644); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 2
	}
	if ledger.Absence.State != absenceHolds {
		return 1
	}
	return 0
}

func readJSON(path string, into any) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(raw, into)
}

// Reconcile sorts every observed exec and open into what the scope covers and
// what it does not. It never fills a gap in the observation with an assumption:
// when paths were not traced, or the record has no exec at all, bounded absence
// is not proven.
func Reconcile(record Record, scope Scope) (Ledger, error) {
	if record.Schema != observationSchema {
		return Ledger{}, fmt.Errorf("unsupported observation schema %q", record.Schema)
	}
	if scope.Schema != scopeSchema {
		return Ledger{}, fmt.Errorf("unsupported scope schema %q", scope.Schema)
	}
	for _, p := range scope.Paths {
		if p.Root == "" || p.Role == "" {
			return Ledger{}, fmt.Errorf("a scope path needs a role and a root: %+v", p)
		}
	}
	ledger := Ledger{
		Schema:      ledgerSchema,
		FilesTraced: record.FilesTraced,
		ExitCode:    record.ExitCode,
		NotObserved: append([]string(nil), record.NotObserved...),
	}
	ledger.Execs.ByRole = map[string]int{}
	ledger.Files.ByRole = map[string]RoleUse{}

	imageRole := map[string]string{}
	for _, image := range scope.Images {
		imageRole[image.Digest] = image.Role
	}
	// Direct steps, numbered in the order the command started them.
	ordinalOf := map[int]int{}
	for _, e := range record.Execs {
		if e.Seq != 0 && e.ParentSeq == 0 {
			ordinalOf[e.Seq] = len(ordinalOf) + 1
			ledger.Steps = append(ledger.Steps, LedgerStep{
				Ordinal: ordinalOf[e.Seq], Seq: e.Seq, Image: e.Image, ImageDigest: e.ImageDigest,
			})
		}
	}
	for _, e := range record.Execs {
		role, reason := classifyExec(e, imageRole, scope.ImageRoots)
		ledger.Execs.Total++
		ledger.Execs.ByRole[role]++
		if ordinal, ok := ordinalOf[e.Step]; ok {
			ledger.Steps[ordinal-1].Execs++
			if e.Seq == e.Step {
				ledger.Steps[ordinal-1].Role = role
			}
		}
		if role == roleUndeclared {
			ledger.Execs.UndeclaredTotal++
			if len(ledger.Execs.Undeclared) < findingLimit {
				ledger.Execs.Undeclared = append(ledger.Execs.Undeclared, ExecFinding{
					Seq: e.Seq, Step: ordinalOf[e.Step], Image: e.Image, ImageDigest: e.ImageDigest, Reason: reason,
				})
			}
		}
	}

	paths := append([]ScopePath(nil), scope.Paths...)
	// The longest root wins, so a state directory inside a run directory is
	// judged as state.
	sort.SliceStable(paths, func(i, j int) bool { return len(paths[i].Root) > len(paths[j].Root) })
	seenPath := map[string]map[string]bool{}
	for _, use := range record.Files {
		step := ordinalOf[use.Step]
		finding := FileFinding{Path: use.Path, Step: step, Image: use.Image, Read: use.Read, Write: use.Write}
		role, covering := classifyPath(use.Path, paths, scope)
		opened := use.Read+use.Write > 0
		if role == roleUndeclared {
			if !opened {
				ledger.Files.ProbesOutsideScope += use.Failed
				continue
			}
			ledger.Files.UndeclaredTotal++
			if len(ledger.Files.Undeclared) < findingLimit {
				ledger.Files.Undeclared = append(ledger.Files.Undeclared, finding)
			}
		}
		if !opened {
			continue
		}
		total := ledger.Files.ByRole[role]
		if seenPath[role] == nil {
			seenPath[role] = map[string]bool{}
		}
		if !seenPath[role][use.Path] {
			seenPath[role][use.Path] = true
			total.Paths++
		}
		total.Read += use.Read
		total.Write += use.Write
		ledger.Files.ByRole[role] = total
		if covering == nil {
			continue
		}
		finding.Role = role
		// Step 0 is the command itself: it may read what its steps own, because
		// it is the one that checks them, but it may not write there.
		foreign := covering.OwnerStep != 0 && step != covering.OwnerStep && (step != 0 || use.Write > 0)
		if foreign {
			ledger.Files.CrossStepTotal++
			if len(ledger.Files.CrossStep) < findingLimit {
				ledger.Files.CrossStep = append(ledger.Files.CrossStep, finding)
			}
		}
		if use.Write > 0 && !covering.Write {
			ledger.Files.WriteToReadOnlyTotal++
			if len(ledger.Files.WriteToReadOnly) < findingLimit {
				ledger.Files.WriteToReadOnly = append(ledger.Files.WriteToReadOnly, finding)
			}
		}
	}

	ledger.Absence = judgeAbsence(record, ledger)
	return ledger, nil
}

func classifyExec(e Exec, imageRole map[string]string, roots []ScopeRoot) (role, reason string) {
	if e.ImageDigest == "" {
		return roleUndeclared, "the image could not be read, so it cannot be matched to anything declared"
	}
	if role, ok := imageRole[e.ImageDigest]; ok {
		return role, ""
	}
	for _, root := range roots {
		if under(e.Image, root.Root) {
			return root.Role, ""
		}
	}
	return roleUndeclared, "no declared image has this digest and the file is under no declared closure"
}

func classifyPath(path string, paths []ScopePath, scope Scope) (string, *ScopePath) {
	for i := range paths {
		if under(path, paths[i].Root) {
			return paths[i].Role, &paths[i]
		}
	}
	for _, root := range scope.ImageRoots {
		if under(path, root.Root) {
			return root.Role, nil
		}
	}
	for _, prefix := range scope.OSInterface {
		if path == prefix || (strings.HasSuffix(prefix, "/") && strings.HasPrefix(path, prefix)) {
			return roleOSInterface, nil
		}
	}
	return roleUndeclared, nil
}

// under reports whether path is root itself or lies below it.
func under(path, root string) bool {
	root = strings.TrimRight(root, "/")
	return path == root || strings.HasPrefix(path, root+"/")
}

func judgeAbsence(record Record, ledger Ledger) LedgerAbsence {
	absence := LedgerAbsence{
		Scope: "the process tree the observed command started, on this host, for this one run; programs under a declared closure are covered as members of it and what they do inside is not described",
	}
	if len(record.Execs) == 0 {
		absence.Reasons = append(absence.Reasons, "the observation holds no exec at all")
	}
	if !record.FilesTraced {
		absence.Reasons = append(absence.Reasons, "opened paths were not traced")
	}
	if n := ledger.Execs.UndeclaredTotal; n > 0 {
		absence.Reasons = append(absence.Reasons, fmt.Sprintf("%d executed program(s) are not covered by the declaration", n))
	}
	if n := ledger.Files.UndeclaredTotal; n > 0 {
		absence.Reasons = append(absence.Reasons, fmt.Sprintf("%d opened path(s) are not covered by the declaration", n))
	}
	if n := ledger.Files.CrossStepTotal; n > 0 {
		absence.Reasons = append(absence.Reasons, fmt.Sprintf("%d open(s) touched a root owned by another step", n))
	}
	if n := ledger.Files.WriteToReadOnlyTotal; n > 0 {
		absence.Reasons = append(absence.Reasons, fmt.Sprintf("%d write open(s) fell under a root declared for reading", n))
	}
	if len(absence.Reasons) == 0 {
		absence.State = absenceHolds
	} else {
		absence.State = absenceNotProven
	}
	return absence
}
