package orderdatarealization

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"

	"tei/internal/ir"
)

// RecordSchema identifies the chain record.
const RecordSchema = "tei.reference.order-data-realization-chain/v0"

// AssetUse records whether an existing asset was called, and how, or why not.
type AssetUse struct {
	Asset  string `json:"asset"`
	Called bool   `json:"called"`
	How    string `json:"how,omitempty"`
	Reason string `json:"reason,omitempty"`
}

// StepRecord is one step of the chain. ModelRevision is the revision the step
// was bound to when it ran; for steps 1 to 3, which change the copy,
// RevisionAfter is the revision they hand over.
type StepRecord struct {
	Step          int        `json:"step"`
	Name          string     `json:"name"`
	ModelRevision string     `json:"modelRevision"`
	RevisionAfter string     `json:"revisionAfter,omitempty"`
	Assets        []AssetUse `json:"assets"`
	Output        any        `json:"output"`
}

// Check is one observed pass/fail judgement: a fault injection or a
// completion condition.
type Check struct {
	ID       string `json:"id"`
	Step     string `json:"step"`
	Injected string `json:"injected,omitempty"`
	Expect   string `json:"expect"`
	Observed any    `json:"observed"`
	Passed   bool   `json:"passed"`
}

// Gap is a platform shortfall the chain had to bridge on the sample side.
type Gap struct {
	Area   string `json:"area"`
	Gap    string `json:"gap"`
	Bridge string `json:"bridgedBy"`
}

// Revisions are the model revisions the chain goes through.
type Revisions struct {
	R0                 string `json:"r0"`
	R1                 string `json:"r1AttributeAdded"`
	R2                 string `json:"r2BusinessKeyAccepted"`
	R3                 string `json:"r3UniqueDeclaredFinal"`
	VariantNoDecisions string `json:"variantWithoutDecisions"`
}

// Record is the single chain record.
type Record struct {
	Schema             string            `json:"schema"`
	RunID              string            `json:"runId"`
	DeclarationsDigest string            `json:"declarationsDigest"`
	Declarations       Declarations      `json:"declarations"`
	Revisions          Revisions         `json:"revisions"`
	Tools              ToolBinaries      `json:"tools"`
	Steps              []StepRecord      `json:"steps"`
	Faults             []Check           `json:"faultInjections"`
	Completion         []Check           `json:"completionConditions"`
	Gaps               []Gap             `json:"platformGaps"`
	Limits             []string          `json:"limits"`
	Environment        map[string]string `json:"environment"`
	Passed             bool              `json:"passed"`
}

// Chain holds the state handed from step to step.
type Chain struct {
	Decl       Declarations
	RepoRoot   string
	Source     string // workspace to copy (reference/workspace); never written
	Dir        string // temporary root owned by the chain
	R0Dir      string // untouched copy at R0
	WorkDir    string // the copy the chain changes
	VariantDir string // copy with the attribute but without the two decisions
	Tools      ToolBinaries
	Rec        Record

	final        string
	worldR0      ir.World
	worldFinal   ir.World
	worldVariant ir.World
	missing      map[string][]string // step-4 missing items per intent for sales.order

	db *dbState
}

// NewChain prepares a chain over a temporary directory. The source workspace
// is only read.
func NewChain(repoRoot, source, dir string) (*Chain, error) {
	d, digest, err := LoadDeclarations()
	if err != nil {
		return nil, err
	}
	var b [6]byte
	if _, err := rand.Read(b[:]); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	c := &Chain{
		Decl: d, RepoRoot: repoRoot, Source: source, Dir: dir,
		R0Dir:      filepath.Join(dir, "workspace-r0"),
		WorkDir:    filepath.Join(dir, "workspace"),
		VariantDir: filepath.Join(dir, "workspace-variant"),
	}
	c.Rec = Record{
		Schema: RecordSchema, RunID: hex.EncodeToString(b[:]),
		DeclarationsDigest: digest, Declarations: d,
		Environment: map[string]string{},
	}
	return c, nil
}

// require binds a step to the final revision: the workspace it reads must
// still have exactly the revision step 3 handed over.
func (c *Chain) require(step int, dir string) error {
	if c.final == "" {
		return fmt.Errorf("step %d: the final revision is not fixed yet (run steps 1-3 first)", step)
	}
	got, err := Revision(dir)
	if err != nil {
		return fmt.Errorf("step %d: %w", step, err)
	}
	if got != c.final {
		return &RevisionMismatch{Step: step, Workspace: dir, Want: c.final, Got: got}
	}
	return nil
}

// FinalRevision is the revision fixed by step 3.
func (c *Chain) FinalRevision() string { return c.final }

func (c *Chain) addStep(s StepRecord) { c.Rec.Steps = append(c.Rec.Steps, s) }

func (c *Chain) fault(ch Check) { c.Rec.Faults = append(c.Rec.Faults, ch) }

func (c *Chain) gap(area, gap, bridge string) {
	c.Rec.Gaps = append(c.Rec.Gaps, Gap{Area: area, Gap: gap, Bridge: bridge})
}

// RunDesign runs steps 1 to 4, which need no database.
func (c *Chain) RunDesign(ctx context.Context) error {
	tools, err := BuildTools(ctx, c.RepoRoot, filepath.Join(c.Dir, "bin"))
	if err != nil {
		return err
	}
	c.Tools = tools
	c.Rec.Tools = tools
	for _, f := range []func(context.Context) error{c.Step1AddAttribute, c.Step2AcceptBusinessKey, c.Step3CrossviewAndUnique, c.Step4StandardFunctions} {
		if err := f(ctx); err != nil {
			return err
		}
	}
	return nil
}
