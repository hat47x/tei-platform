package orderruledesign

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"tei/tools/internal/rmtevidence"
)

// TargetTable is the legacy table that corresponds to the order line.
const TargetTable = "ORDER_DETAIL"

// Synthetic legacy material. The sample writes it itself; it is not an
// external model and stands for the designer's own material. Each variant is
// one XML text. The source adapter and the cross-view reader read it as a
// local file.
const (
	// A table that a function and a task path reach, plus one description
	// the contract cannot handle (LegacyMacro).
	sourceBound = `<System FormatVersion="1.2"><Subsystem ID="SALES">` +
		`<Table ID="ORDER_DETAIL"><TableField ID="UNIT_PRICE"/></Table>` +
		`<Function ID="F_ENTRY"><IOTable ID="IO_ENTRY" TableID="ORDER_DETAIL" OpC="+"/></Function>` +
		`<Task ID="T_ENTRY"><TaskAction ID="A_ENTRY"><TaskFunctionIO FunctionID="F_ENTRY" IOID="0"/></TaskAction></Task>` +
		`</Subsystem><LegacyMacro Detail="price-rounding"/></System>`
	// The same table, no function or task reaches it.
	sourceTableOnly = `<System FormatVersion="1.2"><Subsystem ID="SALES">` +
		`<Table ID="ORDER_DETAIL"><TableField ID="UNIT_PRICE"/></Table>` +
		`</Subsystem><LegacyMacro Detail="price-rounding"/></System>`
	// No table that corresponds to the order line.
	sourceNoTarget = `<System FormatVersion="1.2"><Subsystem ID="SALES">` +
		`<Table ID="CUSTOMER"><TableField ID="NAME"/></Table>` +
		`</Subsystem><LegacyMacro Detail="price-rounding"/></System>`
	// The material was replaced after stage 1: the same table, two more
	// descriptions the contract cannot handle.
	sourceReplaced = `<System FormatVersion="1.2"><Subsystem ID="SALES">` +
		`<Table ID="ORDER_DETAIL"><TableField ID="UNIT_PRICE"/></Table>` +
		`<Function ID="F_ENTRY"><IOTable ID="IO_ENTRY" TableID="ORDER_DETAIL" OpC="+"/></Function>` +
		`<Task ID="T_ENTRY"><TaskAction ID="A_ENTRY"><TaskFunctionIO FunctionID="F_ENTRY" IOID="0"/></TaskAction></Task>` +
		`</Subsystem><LegacyMacro Detail="price-rounding"/><LegacyMacro Detail="discount-table"/>` +
		`<LegacyExit Detail="batch-exit"/></System>`
)

// stage1Out is what stage 1 hands on.
type stage1Out struct {
	Dir            string
	SourcePath     string
	ContractPath   string
	AccountingPath string
	Carried        Carried
	Detail         map[string]any
	TableOrdinal   int
	EvidenceBytes  []byte
	Evidence       *rmtevidence.Evidence
}

type adapterOutput struct {
	Receipt struct {
		ArtifactSHA256     string         `json:"artifactSha256"`
		ElementCounts      map[string]int `json:"elementCounts"`
		ReceiptFingerprint string         `json:"receiptFingerprint"`
	} `json:"receipt"`
	ClassificationCoverage struct {
		ContractSHA256 string `json:"contractSha256"`
		Complete       bool   `json:"complete"`
	} `json:"classificationCoverage"`
	UnresolvedProjection struct {
		Items []struct {
			Code   string `json:"code"`
			Kind   string `json:"kind"`
			Nature string `json:"nature"`
			Count  int    `json:"count"`
		} `json:"items"`
	} `json:"unresolvedProjection"`
}

type crossViewOutput struct {
	Accounting struct {
		SourceArtifactDigest                  string `json:"sourceArtifactDigest"`
		MappingContractFingerprint            string `json:"mappingContractFingerprint"`
		ProjectionFingerprint                 string `json:"projectionFingerprint"`
		PotentialActionFunctionTablePathCount int    `json:"potentialActionFunctionTablePathCount"`
		IOTableCount                          int    `json:"ioTableCount"`
	} `json:"accounting"`
}

// tableOrdinals returns the 1-based document-order ordinal of every element
// named Table whose ID is id, as the source adapter numbers nodes. The
// sample reads the designer's own material here to choose which node to ask
// the adapter about; the adapter, not the sample, decides what the node is.
func tableOrdinals(src []byte, id string) ([]int, error) {
	dec := xml.NewDecoder(strings.NewReader(string(src)))
	var out []int
	n := 0
	for {
		tok, err := dec.Token()
		if errors.Is(err, io.EOF) {
			return out, nil
		}
		if err != nil {
			return nil, err
		}
		if se, ok := tok.(xml.StartElement); ok {
			n++
			if se.Name.Local == "Table" {
				for _, a := range se.Attr {
					if a.Name.Local == "ID" && a.Value == id {
						out = append(out, n)
					}
				}
			}
		}
	}
}

// Stage 1 reads one legacy material and one contract with the source adapter
// and the cross-view reader, unchanged, and decides the provenance state.
func (s *Sample) runStage1(ctx context.Context, label, source string, dir string) (stage1Out, error) {
	out := stage1Out{Dir: dir}
	out.SourcePath = filepath.Join(dir, "legacy.xead")
	out.ContractPath = filepath.Join(dir, "contract.json")
	if err := writeFile(out.SourcePath, []byte(source)); err != nil {
		return out, err
	}
	// The shared synthetic contract is copied, never rewritten.
	if err := copyFile(filepath.Join(s.Repo, "reference", "xead-source-transcription-contract.synthetic.json"), out.ContractPath); err != nil {
		return out, err
	}
	raw, err := s.Tools.mustRun(ctx, s.Tools.SourceAdapter, dir, "-format", "xead", "-input", out.SourcePath, "-contract", out.ContractPath)
	if err != nil {
		return out, err
	}
	var ad adapterOutput
	if err := json.Unmarshal(raw, &ad); err != nil {
		return out, fmt.Errorf("source adapter output: %w", err)
	}
	cvRaw, err := s.Tools.mustRun(ctx, s.Tools.CrossView, dir, "-input", out.SourcePath, "-contract", out.ContractPath)
	if err != nil {
		return out, err
	}
	var cv crossViewOutput
	if err := json.Unmarshal(cvRaw, &cv); err != nil {
		return out, fmt.Errorf("cross-view output: %w", err)
	}
	out.AccountingPath = filepath.Join(dir, "accounting.json")
	if err := writeFile(out.AccountingPath, cvRaw); err != nil {
		return out, err
	}
	var un Unresolved
	for _, it := range ad.UnresolvedProjection.Items {
		un.Total += it.Count
		switch it.Nature {
		case "source-observation-unresolved":
			un.SourceObservation += it.Count
		case "mapping-unresolved":
			un.Mapping += it.Count
		case "semantic-candidate-unresolved":
			un.SemanticCandidate += it.Count
		}
	}
	contractDigest, err := fileDigest(out.ContractPath)
	if err != nil {
		return out, err
	}
	state := StateNoTarget
	evidence := "none"
	ords, err := tableOrdinals([]byte(source), TargetTable)
	if err != nil {
		return out, err
	}
	var evErr string
	if len(ords) == 1 {
		out.TableOrdinal = ords[0]
		evRaw, err := s.Tools.mustRun(ctx, s.Tools.SourceAdapter, dir, "-format", "xead", "-input", out.SourcePath,
			"-contract", out.ContractPath, "-candidate-evidence-node", fmt.Sprint(ords[0]))
		if err != nil {
			return out, err
		}
		ev, err := rmtevidence.Decode(evRaw)
		if err != nil {
			return out, fmt.Errorf("transcription evidence: %w", err)
		}
		if ev.SourceConstruct != "Table" || ev.MappingStatus != rmtevidence.MappingMapped ||
			ev.SourceArtifactDigest != "sha256:"+ad.Receipt.ArtifactSHA256 {
			return out, fmt.Errorf("evidence is not a mapped Table of this material: %+v", ev)
		}
		out.EvidenceBytes, out.Evidence = evRaw, &ev
		evidence = ev.EvidenceFingerprint
		state = StateTableOnly
		// A path reaches the table when the cross-view reader found a
		// task action, function and table path; the sample checks that the
		// io declaration names this table.
		if cv.Accounting.PotentialActionFunctionTablePathCount > 0 && ioReaches([]byte(source), TargetTable) {
			state = StateBound
		}
	} else if len(ords) > 1 {
		evErr = "several table nodes carry the target identity; no single provenance"
	}
	out.TableOrdinal = firstOr(ords, 0)
	out.Carried = Carried{
		SourceDigest:          "sha256:" + ad.Receipt.ArtifactSHA256,
		ContractDigest:        contractDigest,
		ReceiptFingerprint:    ad.Receipt.ReceiptFingerprint,
		AccountingFingerprint: cv.Accounting.ProjectionFingerprint,
		ProvenanceState:       state,
		Unresolved:            &un,
		EvidenceFingerprint:   evidence,
	}
	out.Detail = map[string]any{
		"label":                   label,
		"adapterContractComplete": ad.ClassificationCoverage.Complete,
		"elementCounts":           ad.Receipt.ElementCounts,
		"tableOrdinal":            out.TableOrdinal,
		"potentialPathCount":      cv.Accounting.PotentialActionFunctionTablePathCount,
		"ioTableCount":            cv.Accounting.IOTableCount,
		"stateRule":               "bound: table node mapped and a cross-view path reaches it; table-only: mapped, no path; no-target: no table node",
		"unresolvedBy":            "source-observation = descriptions the contract cannot handle",
		"ambiguity":               evErr,
		"sourceValuesInChain":     false,
	}
	return out, nil
}

func ioReaches(src []byte, table string) bool {
	dec := xml.NewDecoder(strings.NewReader(string(src)))
	for {
		tok, err := dec.Token()
		if err != nil {
			return false
		}
		if se, ok := tok.(xml.StartElement); ok && se.Name.Local == "IOTable" {
			for _, a := range se.Attr {
				if a.Name.Local == "TableID" && a.Value == table {
					return true
				}
			}
		}
	}
}

func firstOr(v []int, d int) int {
	if len(v) == 0 {
		return d
	}
	return v[0]
}

func (s *Sample) stage1(ctx context.Context) error {
	out, err := s.runStage1(ctx, "main", sourceBound, filepath.Join(s.Work, "stage1"))
	if err != nil {
		return err
	}
	s.s1 = out
	// The material is read once; a later replacement is a fault, not a rerun.
	if out.Carried.ProvenanceState != StateBound || out.Carried.Unresolved.SourceObservation < 1 {
		return fmt.Errorf("the main material must give state %s with at least one description the contract cannot handle: %s %+v",
			StateBound, out.Carried.ProvenanceState, out.Carried.Unresolved)
	}
	return s.Chain.Append(1, out.Carried, out.Detail)
}

func fileExists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}
