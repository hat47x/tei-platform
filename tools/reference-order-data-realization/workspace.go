package orderdatarealization

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"tei/internal/ir"
	"tei/internal/sourceindex"
	"tei/internal/validation"
	"tei/internal/workspace"
)

// ErrRevisionMismatch stops a step that receives a workspace whose model
// revision differs from the one the previous step handed over.
var ErrRevisionMismatch = errors.New("model revision mismatch")

// RevisionMismatch carries the step and both revisions.
type RevisionMismatch struct {
	Step      int
	Workspace string
	Want      string
	Got       string
}

func (e *RevisionMismatch) Error() string {
	return fmt.Sprintf("step %d: %v: want %s, workspace %s has %s", e.Step, ErrRevisionMismatch, e.Want, e.Workspace, e.Got)
}

func (e *RevisionMismatch) Unwrap() error { return ErrRevisionMismatch }

// CopyWorkspace copies a workspace tree byte for byte. Only directories and
// regular files are copied; anything else stops the copy.
func CopyWorkspace(src, dst string) error {
	if _, err := os.Stat(dst); err == nil {
		return fmt.Errorf("copy destination %s already exists", dst)
	}
	return filepath.WalkDir(src, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		switch {
		case entry.IsDir():
			return os.MkdirAll(target, 0o755)
		case entry.Type().IsRegular():
			raw, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			return os.WriteFile(target, raw, 0o644)
		default:
			return fmt.Errorf("workspace copy: %s is neither a directory nor a regular file", path)
		}
	})
}

// Revision is the model revision the shared acceptance tool uses
// (sourceindex.ModelRevision over the loaded workspace's source index).
func Revision(root string) (string, error) {
	loaded, err := workspace.Load(root)
	if err != nil {
		return "", fmt.Errorf("load %s: %w", root, err)
	}
	index, err := workspace.BuildSourceIndex(root, loaded)
	if err != nil {
		return "", fmt.Errorf("source index %s: %w", root, err)
	}
	return sourceindex.ModelRevision(index), nil
}

// LoadValidated loads a workspace and returns its World with the validation
// errors (not warnings) as text.
func LoadValidated(root string) (ir.World, []string, error) {
	loaded, err := workspace.Load(root)
	if err != nil {
		return ir.World{}, nil, fmt.Errorf("load %s: %w", root, err)
	}
	var errs []string
	for _, d := range validation.Validate(loaded.World) {
		if d.Severity == validation.Error {
			errs = append(errs, string(d.Code)+": "+d.Message)
		}
	}
	return loaded.World, errs, nil
}

// LineEdit is a direct edit of one workspace file. The anchor must match
// exactly one line (ignoring a trailing carriage return). Lines are inserted
// after or before the anchor, or replace it.
type LineEdit struct {
	File    string   `json:"file"`
	Purpose string   `json:"purpose"`
	Anchor  string   `json:"anchor"`
	Mode    string   `json:"mode"` // after | before | replace
	Lines   []string `json:"lines"`
}

// EditRecord is what the chain records about a direct edit.
type EditRecord struct {
	File          string `json:"file"`
	Purpose       string `json:"purpose"`
	Mode          string `json:"mode"`
	LinesInserted int    `json:"linesInserted"`
	LinesRemoved  int    `json:"linesRemoved"`
	Direct        bool   `json:"directEditOfCopy"`
}

func applyLineEdit(root string, e LineEdit) (EditRecord, error) {
	path := filepath.Join(root, filepath.FromSlash(e.File))
	raw, err := os.ReadFile(path)
	if err != nil {
		return EditRecord{}, err
	}
	lines := strings.Split(string(raw), "\n")
	at := -1
	for i, l := range lines {
		if strings.TrimSuffix(l, "\r") == e.Anchor {
			if at >= 0 {
				return EditRecord{}, fmt.Errorf("%s: anchor %q matches more than one line", e.File, e.Anchor)
			}
			at = i
		}
	}
	if at < 0 {
		return EditRecord{}, fmt.Errorf("%s: anchor %q not found", e.File, e.Anchor)
	}
	eol := ""
	if strings.HasSuffix(lines[at], "\r") {
		eol = "\r"
	}
	ins := make([]string, len(e.Lines))
	for i, l := range e.Lines {
		ins[i] = l + eol
	}
	rec := EditRecord{File: e.File, Purpose: e.Purpose, Mode: e.Mode, LinesInserted: len(ins), Direct: true}
	var out []string
	switch e.Mode {
	case "after":
		out = append(append(append(out, lines[:at+1]...), ins...), lines[at+1:]...)
	case "before":
		out = append(append(append(out, lines[:at]...), ins...), lines[at:]...)
	case "replace":
		out = append(append(append(out, lines[:at]...), ins...), lines[at+1:]...)
		rec.LinesRemoved = 1
	default:
		return EditRecord{}, fmt.Errorf("unknown edit mode %q", e.Mode)
	}
	if err := os.WriteFile(path, []byte(strings.Join(out, "\n")), 0o644); err != nil {
		return EditRecord{}, err
	}
	return rec, nil
}

// attributeEdits are the step-1 direct edits: the attribute, its column in the
// storage mapping, and the list query that shows it with the external number.
func attributeEdits(d Declarations) []LineEdit {
	a := d.Attribute
	return []LineEdit{
		{File: "semantic/sales/order.tei", Purpose: "受注へ属性を追加する", Mode: "after",
			Anchor: "    attribute externalId type std.text",
			Lines:  []string{fmt.Sprintf("    attribute %s type %s", a.Symbol, a.Type)}},
		{File: "databases/sales-core/persistence/order.tei", Purpose: "追加した属性を保存写像の列へ対応づける", Mode: "after",
			Anchor: "    column sales.order.external_id -> external_id storage varchar optional",
			Lines:  []string{fmt.Sprintf("    column %s -> %s storage %s %s", a.Attribute, a.Column, a.Storage, a.Nullability)}},
		{File: "applications/order-entry/queries/recent_orders.sql", Purpose: "一覧に外部番号と追加した属性を出す", Mode: "replace",
			Anchor: "            o.total_amount",
			Lines:  []string{"            o.total_amount,", "            o.external_id,", "            o.delivery_note"}},
		{File: "applications/order-entry/queries/recent_orders.tei", Purpose: "一覧の列を意味の属性へ対応づける", Mode: "after",
			Anchor: "        column total_amount -> sales.order.total",
			Lines:  []string{"        column external_id -> sales.order.external_id", fmt.Sprintf("        column %s -> %s", a.Column, a.Attribute)}},
	}
}

// uniqueEdit is the step-3 direct edit that declares the unique constraint.
func uniqueEdit(d Declarations) LineEdit {
	lines := []string{fmt.Sprintf("    unique %s {", d.UniqueDecision.Name)}
	for _, a := range d.UniqueDecision.Attributes {
		lines = append(lines, "        attribute "+a)
	}
	lines = append(lines, "    }", "")
	return LineEdit{File: "databases/sales-core/persistence/order.tei", Purpose: "業務キーに一意制約を宣言する", Mode: "before",
		Anchor: "    primary-key sales.order.order_id", Lines: lines}
}

// ToolBinaries are the existing tools that are package main and therefore run
// as separate processes.
type ToolBinaries struct {
	SharedAcceptance   string `json:"sharedAcceptance"`
	StandardFunctions  string `json:"standardFunctionCandidates"`
	GoVersion          string `json:"goVersion"`
	BuiltFromRepoRoot  string `json:"-"`
	BuildCommandPrefix string `json:"buildCommand"`
}

// GoEnv overrides the go command used to build the existing tools.
const GoEnv = "TEI_ODR_GO"

func goCommand() (string, error) {
	if g := os.Getenv(GoEnv); g != "" {
		return g, nil
	}
	if g, err := exec.LookPath("go"); err == nil {
		return g, nil
	}
	g := filepath.Join(runtime.GOROOT(), "bin", "go")
	if _, err := os.Stat(g); err != nil {
		return "", fmt.Errorf("go command not found (set %s)", GoEnv)
	}
	return g, nil
}

// FindRepoRoot walks up from dir to the directory whose go.mod declares
// module tei.
func FindRepoRoot(dir string) (string, error) {
	dir, err := filepath.Abs(dir)
	if err != nil {
		return "", err
	}
	for {
		raw, err := os.ReadFile(filepath.Join(dir, "go.mod"))
		if err == nil && strings.HasPrefix(strings.ReplaceAll(string(raw), "\r", ""), "module tei\n") {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("repository root (go.mod with module tei) not found")
		}
		dir = parent
	}
}

// BuildTools builds the package-main tools the chain calls without changing
// them.
func BuildTools(ctx context.Context, repoRoot, outDir string) (ToolBinaries, error) {
	g, err := goCommand()
	if err != nil {
		return ToolBinaries{}, err
	}
	ver, err := exec.CommandContext(ctx, g, "version").Output()
	if err != nil {
		return ToolBinaries{}, fmt.Errorf("go version: %w", err)
	}
	t := ToolBinaries{
		SharedAcceptance:   filepath.Join(outDir, "tei-shared-acceptance"),
		StandardFunctions:  filepath.Join(outDir, "tei-standard-function-candidates"),
		GoVersion:          strings.TrimSpace(string(ver)),
		BuiltFromRepoRoot:  repoRoot,
		BuildCommandPrefix: "go build -o <tmp> ./tools/reference-shared-acceptance ./tools/reference-standard-function-candidates",
	}
	for bin, pkg := range map[string]string{t.SharedAcceptance: "./tools/reference-shared-acceptance", t.StandardFunctions: "./tools/reference-standard-function-candidates"} {
		cmd := exec.CommandContext(ctx, g, "build", "-o", bin, pkg)
		cmd.Dir = repoRoot
		if out, err := cmd.CombinedOutput(); err != nil {
			return ToolBinaries{}, fmt.Errorf("go build %s: %v: %s", pkg, err, out)
		}
	}
	return t, nil
}

func runTool(ctx context.Context, bin string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, bin, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if err := cmd.Run(); err != nil {
		return stdout.Bytes(), fmt.Errorf("%s %s: %v: %s", filepath.Base(bin), strings.Join(args, " "), err, strings.TrimSpace(stderr.String()))
	}
	return stdout.Bytes(), nil
}

func hasAttribute(world ir.World, id string) bool {
	for _, ns := range world.Semantics {
		for _, data := range ns.Data {
			for _, a := range data.Attributes {
				if string(a.ID) == id {
					return true
				}
			}
		}
	}
	return false
}
