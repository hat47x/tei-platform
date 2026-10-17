// Package dataimportcheck is stage D0 of business data import: it checks the
// shape of tab-separated rows against one Data definition of an accepted
// World. It never writes, never evaluates business Validation or authorization,
// and never puts a cell value into its findings.
package dataimportcheck

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"

	"tei/internal/ir"
	"tei/internal/semantic"
)

const Schema = "tei.reference.data-import-shape-check/v0"

// Finding locates a problem without carrying the value that caused it.
type Finding struct {
	Row    int    `json:"row,omitempty"`
	Column string `json:"column,omitempty"`
	Code   string `json:"code"`
}

type Report struct {
	Schema string `json:"schema"`
	Data   string `json:"data"`
	Rows   int    `json:"rows"`
	// Key lists the attributes used for key checks, or is empty when the Data
	// declares no candidate key.
	Key      []string  `json:"key"`
	Findings []Finding `json:"findings"`
	// Unchecked names what this stage cannot decide, so a clean report is not
	// read as "the data is valid".
	Unchecked []string `json:"unchecked"`
	// ShapeOK is true only when there is no finding. It says nothing about
	// business Validation, authorization, or whether a row would be accepted.
	ShapeOK bool `json:"shapeOk"`
}

// Check reads TSV (first line is the header) and checks it against dataID.
// Columns are matched by attribute symbol or by attribute ID.
func Check(world ir.World, dataID string, r io.Reader) (Report, error) {
	var data *semantic.Data
	types := map[string]semantic.Type{}
	for _, ns := range world.Semantics {
		for _, t := range ns.Types {
			types[string(t.ID)] = t
		}
		for i := range ns.Data {
			if string(ns.Data[i].ID) == dataID {
				d := ns.Data[i]
				data = &d
			}
		}
	}
	if data == nil {
		return Report{}, fmt.Errorf("Data %q is not defined", dataID)
	}
	rep := Report{Schema: Schema, Data: dataID, Key: []string{}, Findings: []Finding{}, Unchecked: []string{
		"business-validation", "authorization", "reference-targets", "nullability-not-declared",
	}}

	byName := map[string]semantic.Attribute{}
	for _, a := range data.Attributes {
		byName[a.Symbol] = a
		byName[string(a.ID)] = a
	}
	if len(data.CandidateKeys) > 0 {
		for _, a := range data.CandidateKeys[0].Attributes {
			rep.Key = append(rep.Key, string(a))
		}
	} else {
		rep.Unchecked = append(rep.Unchecked, "no-candidate-key: duplicate rows cannot be detected")
	}

	cr := csv.NewReader(r)
	cr.Comma = '\t'
	cr.FieldsPerRecord = -1
	cr.LazyQuotes = false
	records, err := cr.ReadAll()
	if err != nil {
		return Report{}, fmt.Errorf("read rows: %w", err)
	}
	if len(records) == 0 {
		return Report{}, fmt.Errorf("input is empty: a header row is required")
	}
	header := records[0]
	if len(header) > 0 {
		header[0] = strings.TrimPrefix(header[0], "\ufeff")
	}

	cols := make([]*semantic.Attribute, len(header))
	seen := map[string]bool{}
	byID := map[string]int{}
	for i, h := range header {
		h = strings.TrimSpace(h)
		a, ok := byName[h]
		if !ok {
			rep.add(0, "", "unknown-column")
			continue
		}
		if seen[string(a.ID)] {
			rep.add(0, a.Symbol, "duplicate-column")
			continue
		}
		seen[string(a.ID)] = true
		aa := a
		cols[i] = &aa
		byID[string(a.ID)] = i
	}
	for _, k := range rep.Key {
		if _, ok := byID[k]; !ok {
			rep.add(0, k, "missing-key-column")
		}
	}

	keys := map[string]int{}
	rows := records[1:]
	rep.Rows = len(rows)
	for n, rec := range rows {
		row := n + 2 // header is line 1
		if len(rec) != len(header) {
			rep.add(row, "", "field-count-mismatch")
			continue
		}
		for i, cell := range rec {
			a := cols[i]
			if a == nil || cell == "" {
				continue
			}
			if code := checkValue(root(types, string(a.TypeRef)), cell); code != "" {
				rep.add(row, a.Symbol, code)
			}
		}
		if len(rep.Key) == 0 {
			continue
		}
		var parts []string
		complete := true
		for _, k := range rep.Key {
			i, ok := byID[k]
			if !ok || rec[i] == "" {
				if ok {
					rep.add(row, byName[k].Symbol, "missing-key-value")
				}
				complete = false
				continue
			}
			parts = append(parts, rec[i])
		}
		if !complete {
			continue
		}
		fingerprint, _ := json.Marshal(parts)
		if first, dup := keys[string(fingerprint)]; dup {
			rep.add(row, "", fmt.Sprintf("duplicate-key-of-row-%d", first))
		} else {
			keys[string(fingerprint)] = row
		}
	}

	sort.SliceStable(rep.Findings, func(i, j int) bool {
		if rep.Findings[i].Row != rep.Findings[j].Row {
			return rep.Findings[i].Row < rep.Findings[j].Row
		}
		if rep.Findings[i].Column != rep.Findings[j].Column {
			return rep.Findings[i].Column < rep.Findings[j].Column
		}
		return rep.Findings[i].Code < rep.Findings[j].Code
	})
	rep.ShapeOK = len(rep.Findings) == 0
	return rep, nil
}

func (r *Report) add(row int, column, code string) {
	r.Findings = append(r.Findings, Finding{Row: row, Column: column, Code: code})
}

// root follows an alias chain to its primitive type. A type that cannot be
// resolved returns its own ID, which checkValue treats as unchecked.
func root(types map[string]semantic.Type, id string) string {
	seen := map[string]bool{}
	for !seen[id] {
		seen[id] = true
		t, ok := types[id]
		if !ok || t.Kind == semantic.TypePrimitive || string(t.Underlying) == "" {
			return id
		}
		id = string(t.Underlying)
	}
	return id
}

// checkValue returns a finding code, or "" when the cell fits the type. Types
// whose text form is open (text, code, identifier) and unknown types are not
// checked; they are not reported as findings.
func checkValue(rootType, cell string) string {
	switch rootType {
	case "std.integer":
		if _, err := strconv.ParseInt(cell, 10, 64); err != nil {
			return "type-mismatch"
		}
	case "std.decimal", "std.money":
		f, err := strconv.ParseFloat(cell, 64)
		if err != nil || math.IsNaN(f) || math.IsInf(f, 0) || strings.ContainsAny(cell, "eE") {
			return "type-mismatch"
		}
	case "std.boolean":
		if cell != "true" && cell != "false" {
			return "type-mismatch"
		}
	case "std.date":
		if _, err := time.Parse("2006-01-02", cell); err != nil {
			return "type-mismatch"
		}
	case "std.datetime":
		if _, err := time.Parse(time.RFC3339, cell); err != nil {
			return "type-mismatch"
		}
	case "std.time":
		if _, err := time.Parse("15:04:05", cell); err != nil {
			return "type-mismatch"
		}
	case "std.json":
		if !json.Valid([]byte(cell)) {
			return "type-mismatch"
		}
	}
	return ""
}
