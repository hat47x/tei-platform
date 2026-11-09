package dataimportcheck

import (
	"encoding/csv"
	"fmt"
	"sort"
	"strings"

	"tei/internal/ir"
	"tei/internal/semantic"
)

const BatchSchema = "tei.reference.data-import-batch-check/v0"

// TableInput is one TSV table in a batch import. Role is "initial" for
// reference data accepted with the meaning, "operational" for case-owned data,
// or empty when the caller has not decided.
type TableInput struct {
	Data string `json:"data"`
	Role string `json:"role"`
	TSV  string `json:"tsv"`
}

// RowOutcome explains what happened to one data row.
type RowOutcome struct {
	Row    int      `json:"row"`
	Status string   `json:"status"`
	Codes  []string `json:"codes,omitempty"`
}

// ReferenceFinding locates a broken reference without carrying the value.
type ReferenceFinding struct {
	Row       int    `json:"row,omitempty"`
	Data      string `json:"data"`
	Attribute string `json:"attribute"`
	Relation  string `json:"relation"`
	Code      string `json:"code"`
}

// TableReport combines the per-table shape report with row outcomes.
type TableReport struct {
	Data  string       `json:"data"`
	Role  string       `json:"role"`
	Shape Report       `json:"shape"`
	Rows  []RowOutcome `json:"rows"`
}

// Boundary separates initial data from operational data.
type Boundary struct {
	Initial     []string `json:"initial"`
	Operational []string `json:"operational"`
	Undeclared  []string `json:"undeclared,omitempty"`
}

// BatchReport is the result. ReferencesOK is false when a reference is broken
// or cannot be checked because the target table or key column is missing.
type BatchReport struct {
	Schema       string             `json:"schema"`
	Tables       []TableReport      `json:"tables"`
	References   []ReferenceFinding `json:"references"`
	Boundary     Boundary           `json:"boundary"`
	Unchecked    []string           `json:"unchecked"`
	ReferencesOK bool               `json:"referencesOk"`
	OK           bool               `json:"ok"`
}

// CheckBatch checks the shape of every table, then the references between
// them. It never writes and never puts a cell value into a finding.
func CheckBatch(world ir.World, tables []TableInput) (BatchReport, error) {
	out := BatchReport{
		Schema:     BatchSchema,
		Tables:     []TableReport{},
		References: []ReferenceFinding{},
		Boundary:   Boundary{Initial: []string{}, Operational: []string{}, Undeclared: []string{}},
		Unchecked:  []string{"business-validation", "authorization", "nullability-not-declared"},
	}
	byData := map[string]TableInput{}
	parsed := map[string]parsedTable{}
	for _, table := range tables {
		shape, err := Check(world, table.Data, strings.NewReader(table.TSV))
		if err != nil {
			return BatchReport{}, err
		}
		pt, err := parseTable(table.TSV)
		if err != nil {
			return BatchReport{}, fmt.Errorf("%s: %w", table.Data, err)
		}
		byData[table.Data] = table
		parsed[table.Data] = pt
		rows := rowOutcomes(shape)
		out.Tables = append(out.Tables, TableReport{Data: table.Data, Role: table.Role, Shape: shape, Rows: rows})
		switch table.Role {
		case "initial":
			out.Boundary.Initial = append(out.Boundary.Initial, table.Data)
		case "operational":
			out.Boundary.Operational = append(out.Boundary.Operational, table.Data)
		default:
			out.Boundary.Undeclared = append(out.Boundary.Undeclared, table.Data)
		}
	}
	if len(out.Boundary.Undeclared) > 0 {
		out.Unchecked = append(out.Unchecked, "role-not-declared")
	}

	datas := dataMap(world)
	seen := map[string]bool{}
	for _, table := range tables {
		source, ok := datas[table.Data]
		if !ok {
			continue
		}
		src := parsed[table.Data]
		for _, relation := range source.Relations {
			targetID := string(relation.Target)
			targetData, targetKnown := datas[targetID]
			_, tableKnown := byData[targetID]
			if !targetKnown {
				out.Unchecked = append(out.Unchecked, "relation-target-not-defined:"+string(relation.ID))
				continue
			}
			if !tableKnown {
				out.Unchecked = append(out.Unchecked, "reference-table-not-provided:"+targetID)
				continue
			}
			tgt := parsed[targetID]
			for _, key := range relation.Keys {
				srcCol := attributeColumn(source, traversalHeader(src.header), string(key.SourceAttribute))
				tgtCol := attributeColumn(targetData, traversalHeader(tgt.header), string(key.TargetAttribute))
				if srcCol < 0 || tgtCol < 0 {
					out.Unchecked = append(out.Unchecked, "reference-key-column-missing:"+string(relation.ID))
					continue
				}
				values := map[string]bool{}
				for _, row := range tgt.rows {
					if tgtCol < len(row) && row[tgtCol] != "" {
						values[row[tgtCol]] = true
					}
				}
				for index, row := range src.rows {
					if srcCol >= len(row) || row[srcCol] == "" {
						continue
					}
					if values[row[srcCol]] {
						continue
					}
					finding := ReferenceFinding{
						Row: index + 2, Data: table.Data,
						Attribute: string(key.SourceAttribute), Relation: string(relation.ID),
						Code: "reference-target-missing",
					}
					fingerprint := fmt.Sprintf("%d|%s|%s|%s", finding.Row, finding.Data, finding.Attribute, finding.Relation)
					if !seen[fingerprint] {
						seen[fingerprint] = true
						out.References = append(out.References, finding)
					}
				}
			}
		}
	}

	// Reference findings mark the row as rejected in the per-row outcome.
	for _, finding := range out.References {
		for i := range out.Tables {
			if out.Tables[i].Data != finding.Data {
				continue
			}
			markRow(out.Tables[i].Rows, finding.Row, finding.Code)
		}
	}
	sort.Slice(out.References, func(i, j int) bool {
		if out.References[i].Data != out.References[j].Data {
			return out.References[i].Data < out.References[j].Data
		}
		if out.References[i].Row != out.References[j].Row {
			return out.References[i].Row < out.References[j].Row
		}
		return out.References[i].Relation < out.References[j].Relation
	})
	sort.Strings(out.Unchecked)

	out.ReferencesOK = len(out.References) == 0 && !hasReferenceUnchecked(out.Unchecked)
	ok := out.ReferencesOK
	for _, table := range out.Tables {
		if !table.Shape.ShapeOK {
			ok = false
		}
	}
	out.OK = ok
	return out, nil
}

type parsedTable struct {
	header []string
	rows   [][]string
}

func parseTable(tsv string) (parsedTable, error) {
	reader := csv.NewReader(strings.NewReader(tsv))
	reader.Comma = '\t'
	reader.FieldsPerRecord = -1
	records, err := reader.ReadAll()
	if err != nil {
		return parsedTable{}, fmt.Errorf("read rows: %w", err)
	}
	if len(records) == 0 {
		return parsedTable{}, fmt.Errorf("input is empty: a header row is required")
	}
	header := records[0]
	if len(header) > 0 {
		header[0] = strings.TrimPrefix(header[0], "\ufeff")
	}
	return parsedTable{header: header, rows: records[1:]}, nil
}

// traversalHeader trims surrounding spaces once, so the column lookup does not
// depend on the caller.
func traversalHeader(header []string) []string {
	out := make([]string, len(header))
	for i, h := range header {
		out[i] = strings.TrimSpace(h)
	}
	return out
}

func attributeColumn(data semantic.Data, header []string, attributeID string) int {
	for i, h := range header {
		for _, attribute := range data.Attributes {
			if string(attribute.ID) != attributeID {
				continue
			}
			if h == attribute.Symbol || h == string(attribute.ID) {
				return i
			}
		}
	}
	return -1
}

func dataMap(world ir.World) map[string]semantic.Data {
	out := map[string]semantic.Data{}
	for _, ns := range world.Semantics {
		for _, d := range ns.Data {
			out[string(d.ID)] = d
		}
	}
	return out
}

func rowOutcomes(shape Report) []RowOutcome {
	byRow := map[int][]string{}
	for _, finding := range shape.Findings {
		if finding.Row == 0 {
			continue
		}
		byRow[finding.Row] = append(byRow[finding.Row], finding.Code)
	}
	rows := make([]RowOutcome, 0, shape.Rows)
	for row := 2; row < 2+shape.Rows; row++ {
		codes := uniqueStrings(byRow[row])
		status := "ok"
		if len(codes) > 0 {
			status = "rejected"
		}
		rows = append(rows, RowOutcome{Row: row, Status: status, Codes: codes})
	}
	return rows
}

func markRow(rows []RowOutcome, row int, code string) {
	for i := range rows {
		if rows[i].Row != row {
			continue
		}
		rows[i].Status = "rejected"
		rows[i].Codes = uniqueStrings(append(rows[i].Codes, code))
		return
	}
}

func uniqueStrings(values []string) []string {
	set := map[string]bool{}
	var out []string
	for _, value := range values {
		if set[value] {
			continue
		}
		set[value] = true
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}

func hasReferenceUnchecked(unchecked []string) bool {
	for _, value := range unchecked {
		if strings.HasPrefix(value, "reference-") || strings.HasPrefix(value, "relation-target-not-defined:") {
			return true
		}
	}
	return false
}
