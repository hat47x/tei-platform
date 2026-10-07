// reference-xead-source-inspect reads a Repository-external Xead/Xeaf source
// and emits only aggregate, non-canonical inspection Evidence. It deliberately
// omits raw source text, business names, legacy IDs, and full source locators.
package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	"tei/internal/xeadsource"
)

const reportSchemaVersion = "reference-xead-source-inspection-v2"

type sourceIdentity struct {
	Format xeadsource.Format `json:"format"`
	SHA256 string            `json:"sha256"`
}

type inventorySummaryEntry struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}

type factSummary struct {
	LegacyKeyCount                    int            `json:"legacyKeyCount"`
	LegacyKeyTypes                    map[string]int `json:"legacyKeyTypes,omitempty"`
	CompositeLegacyKeyCount           int            `json:"compositeLegacyKeyCount"`
	DerivedLegacyKeyMemberCount       int            `json:"derivedLegacyKeyMemberCount"`
	RelationshipKeyCount              int            `json:"relationshipKeyCount"`
	RelationshipsUsingDerivedKeyCount int            `json:"relationshipsUsingDerivedKeyCount"`
	DeclaredKeyFieldsCount            int            `json:"declaredKeyFieldsCount"`
	RuntimeKeyCount                   int            `json:"runtimeKeyCount"`
	RuntimeReferCount                 int            `json:"runtimeReferCount"`
	RuntimeReferExplicitKeyCount      int            `json:"runtimeReferExplicitKeyCount"`
	RuntimeReferPKFallbackCount       int            `json:"runtimeReferPkFallbackCount"`
}

type candidateSummary struct {
	Total            int            `json:"total"`
	Mapped           int            `json:"mapped"`
	Unresolved       int            `json:"unresolved"`
	ByClassification map[string]int `json:"byClassification,omitempty"`
}

type diagnosticEvidence struct {
	Code          string `json:"code"`
	LocatorDigest string `json:"locatorDigest"`
}

type diagnosticSummary struct {
	Total    int                  `json:"total"`
	ByCode   map[string]int       `json:"byCode,omitempty"`
	Evidence []diagnosticEvidence `json:"evidence,omitempty"`
}

type factResolutionCount struct {
	Total       int `json:"total"`
	Resolved    int `json:"resolved"`
	Unresolved  int `json:"unresolved"`
	Unvalidated int `json:"unvalidated"`
}

type factResolutionSummary struct {
	Total       int                            `json:"total"`
	Resolved    int                            `json:"resolved"`
	Unresolved  int                            `json:"unresolved"`
	Unvalidated int                            `json:"unvalidated"`
	ByKind      map[string]factResolutionCount `json:"byKind,omitempty"`
}

type report struct {
	SchemaVersion                    string                  `json:"schemaVersion"`
	Source                           sourceIdentity          `json:"source"`
	Inventory                        []inventorySummaryEntry `json:"inventory"`
	UnknownConstructCount            int                     `json:"unknownConstructCount"`
	Candidates                       candidateSummary        `json:"candidates"`
	RelationalFacts                  factSummary             `json:"relationalFacts"`
	RelationalDiagnostics            diagnosticSummary       `json:"relationalDiagnostics"`
	RelationalFactResolutions         factResolutionSummary   `json:"relationalFactResolutions"`
	RelationalResolutionStatus       string                  `json:"relationalResolutionStatus"`
	ClassificationPolicyFingerprint  string                  `json:"classificationPolicyFingerprint"`
	TranscriptionContractFingerprint string                  `json:"transcriptionContractFingerprint"`
	AuthorityLimits                  []string                `json:"authorityLimits"`
}

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("reference-xead-source-inspect", flag.ContinueOnError)
	flags.SetOutput(stderr)
	formatValue := flags.String("format", "", "source format: xead or xeaf")
	sourcePath := flags.String("source", "", "Repository-external Xead/Xeaf source file")
	contractPath := flags.String("contract", "reference/reference-model-transfer/transcription-contract-v0.json", "transcription contract JSON")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if strings.TrimSpace(*formatValue) == "" || strings.TrimSpace(*sourcePath) == "" {
		fmt.Fprintln(stderr, "usage: reference-xead-source-inspect -format xead|xeaf -source FILE [-contract FILE]")
		return 2
	}
	format := xeadsource.Format(*formatValue)
	if format != xeadsource.FormatXead && format != xeadsource.FormatXeaf {
		fmt.Fprintf(stderr, "unsupported format %q; expected xead or xeaf\n", *formatValue)
		return 2
	}
	source, err := os.ReadFile(*sourcePath)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	contract, err := os.ReadFile(*contractPath)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	inspection, err := xeadsource.InspectWithContract(format, source, contract)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	result := buildReport(inspection)
	encoder := json.NewEncoder(stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(result); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	return 0
}

func buildReport(inspection xeadsource.Inspection) report {
	result := report{
		SchemaVersion: reportSchemaVersion,
		Source: sourceIdentity{
			Format: inspection.Document.Format,
			SHA256: inspection.Document.SHA256,
		},
		Inventory:                        summarizeInventory(inspection.Document.Inventory),
		UnknownConstructCount:            countUnknownInventory(inspection.Document.Inventory),
		Candidates:                       summarizeCandidates(inspection.Candidates),
		RelationalFacts:                  summarizeFacts(inspection.RelationalFacts),
		RelationalDiagnostics:            summarizeDiagnostics(inspection.RelationalDiagnostics),
		RelationalFactResolutions:         summarizeFactResolutions(inspection.RelationalFactResolutions),
		RelationalResolutionStatus:       inspection.RelationalResolutionStatus,
		ClassificationPolicyFingerprint:  inspection.ClassificationPolicyFingerprint,
		TranscriptionContractFingerprint: inspection.TranscriptionContractFingerprint,
		AuthorityLimits:                  append([]string(nil), inspection.AuthorityLimits...),
	}
	sort.Strings(result.AuthorityLimits)
	return result
}

func summarizeInventory(inventory []xeadsource.InventoryEntry) []inventorySummaryEntry {
	counts := map[string]int{}
	for _, entry := range inventory {
		if !entry.Known {
			continue
		}
		name := entry.Name
		if i := strings.LastIndex(name, "}"); i >= 0 {
			name = name[i+1:]
		}
		counts[name] += entry.Count
	}
	names := make([]string, 0, len(counts))
	for name := range counts {
		names = append(names, name)
	}
	sort.Strings(names)
	out := make([]inventorySummaryEntry, 0, len(names))
	for _, name := range names {
		out = append(out, inventorySummaryEntry{Name: name, Count: counts[name]})
	}
	return out
}

func countUnknownInventory(inventory []xeadsource.InventoryEntry) int {
	total := 0
	for _, entry := range inventory {
		if !entry.Known {
			total += entry.Count
		}
	}
	return total
}

func summarizeCandidates(candidates []xeadsource.Candidate) candidateSummary {
	out := candidateSummary{
		Total:            len(candidates),
		ByClassification: map[string]int{},
	}
	for _, candidate := range candidates {
		if candidate.Unresolved {
			out.Unresolved++
		} else {
			out.Mapped++
		}
		for _, classification := range candidate.Classifications {
			out.ByClassification[classification]++
		}
	}
	if len(out.ByClassification) == 0 {
		out.ByClassification = nil
	}
	return out
}

func summarizeFacts(facts xeadsource.RelationalSourceFacts) factSummary {
	out := factSummary{
		LegacyKeyCount:         len(facts.LegacyKeys),
		LegacyKeyTypes:         map[string]int{},
		RelationshipKeyCount:   len(facts.RelationshipKeys),
		DeclaredKeyFieldsCount: len(facts.DeclaredKeyFields),
		RuntimeKeyCount:        len(facts.RuntimeKeys),
		RuntimeReferCount:      len(facts.RuntimeRefers),
	}
	type keyIdentity struct {
		table string
		key   string
	}
	derivedKeys := map[keyIdentity]bool{}
	for _, key := range facts.LegacyKeys {
		out.LegacyKeyTypes[key.KeyType]++
		if len(key.Members) > 1 {
			out.CompositeLegacyKeyCount++
		}
		for _, member := range key.Members {
			if member.AttributeType == "DERIVABLE" {
				out.DerivedLegacyKeyMemberCount++
				derivedKeys[keyIdentity{table: key.TableLegacyID, key: key.KeyLegacyID}] = true
			}
		}
	}
	if len(out.LegacyKeyTypes) == 0 {
		out.LegacyKeyTypes = nil
	}

	for _, relation := range facts.RelationshipKeys {
		if derivedKeys[keyIdentity{table: relation.Table1LegacyID, key: relation.Key1LegacyID}] ||
			derivedKeys[keyIdentity{table: relation.Table2LegacyID, key: relation.Key2LegacyID}] {
			out.RelationshipsUsingDerivedKeyCount++
		}
	}
	for _, refer := range facts.RuntimeRefers {
		if len(refer.ToKeyFields) == 0 {
			out.RuntimeReferPKFallbackCount++
		} else {
			out.RuntimeReferExplicitKeyCount++
		}
	}
	return out
}

func summarizeDiagnostics(diagnostics []xeadsource.RelationalDiagnostic) diagnosticSummary {
	out := diagnosticSummary{
		Total:  len(diagnostics),
		ByCode: map[string]int{},
	}
	for _, diagnostic := range diagnostics {
		out.ByCode[diagnostic.Code]++
		sum := sha256.Sum256([]byte(diagnostic.Locator))
		out.Evidence = append(out.Evidence, diagnosticEvidence{
			Code:          diagnostic.Code,
			LocatorDigest: hex.EncodeToString(sum[:]),
		})
	}
	if len(out.ByCode) == 0 {
		out.ByCode = nil
	}
	sort.Slice(out.Evidence, func(i, j int) bool {
		if out.Evidence[i].Code != out.Evidence[j].Code {
			return out.Evidence[i].Code < out.Evidence[j].Code
		}
		return out.Evidence[i].LocatorDigest < out.Evidence[j].LocatorDigest
	})
	return out
}


func summarizeFactResolutions(values []xeadsource.RelationalFactResolution) factResolutionSummary {
	out := factResolutionSummary{
		Total:  len(values),
		ByKind: map[string]factResolutionCount{},
	}
	for _, value := range values {
		count := out.ByKind[value.Kind]
		count.Total++
		switch value.Status {
		case xeadsource.RelationalFactResolutionResolved:
			out.Resolved++
			count.Resolved++
		case xeadsource.RelationalFactResolutionUnresolved:
			out.Unresolved++
			count.Unresolved++
		case xeadsource.RelationalFactResolutionUnvalidated:
			out.Unvalidated++
			count.Unvalidated++
		}
		out.ByKind[value.Kind] = count
	}
	if len(out.ByKind) == 0 {
		out.ByKind = nil
	}
	return out
}
