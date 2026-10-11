package exampleslide

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html"
	"math/big"
	"sort"
	"strings"
)

// Catalog is a read-only adapter input. It is not a TEI canonical source.
type Catalog struct {
	Revision  string             `json:"revision"`
	Tables    map[string]Table   `json:"tables"`
	Relations []SemanticRelation `json:"relations,omitempty"`
}

type Table struct {
	Fields []Field `json:"fields"`
}

type Field struct {
	ID   string `json:"id"`
	Type string `json:"type"`
	// SemanticType retains the original TEI attribute type identity where known.
	// Equality of JSON values does not imply equality of semantic domains.
	SemanticType string `json:"semanticType,omitempty"`
	Required     bool   `json:"required,omitempty"`
	Visible      bool   `json:"visible,omitempty"`
	RefTable     string `json:"refTable,omitempty"`
}

// SemanticRelation preserves directed semantic relationships separately from
// physical foreign keys. The pair of attribute IDs is not a record-ID link.
type SemanticRelation struct {
	ID          string                `json:"id"`
	SourceTable string                `json:"sourceTable"`
	TargetTable string                `json:"targetTable"`
	Cardinality string                `json:"cardinality"`
	Keys        []SemanticRelationKey `json:"keys,omitempty"`
}
type SemanticRelationKey struct {
	SourceField string `json:"sourceField"`
	TargetField string `json:"targetField"`
}

type Record struct {
	ID     string                     `json:"id"`
	Table  string                     `json:"table"`
	Values map[string]json.RawMessage `json:"values"`
}

type Change struct {
	Op     string `json:"op"`
	Record Record `json:"record"`
}

type Stage struct {
	ID      string   `json:"id"`
	Changes []Change `json:"changes"`
	// A later stage must independently assert its complete tables. The
	// base snapshot's completeness is never implicitly carried forward.
	CompleteTables []string `json:"completeTables,omitempty"`
}

type Question struct {
	ID   string `json:"id"`
	Text string `json:"text"`
}

type Example struct {
	ID            string     `json:"id"`
	ModelRevision string     `json:"modelRevision"`
	Base          []Record   `json:"base"`
	Stages        []Stage    `json:"stages,omitempty"`
	Questions     []Question `json:"questions,omitempty"`
	// CompleteTables declares the tables fully represented in the base
	// snapshot only. Each later stage must independently declare completeness.
	CompleteTables []string `json:"completeTables,omitempty"`
}

type Placement struct {
	ID    string   `json:"id"`
	Table string   `json:"table"`
	X     int      `json:"x"`
	Y     int      `json:"y"`
	Show  []string `json:"show,omitempty"`
	Hide  []string `json:"hide,omitempty"`
}

type Diagram struct {
	ID         string      `json:"id"`
	Placements []Placement `json:"placements"`
}

type Frame struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	Stage string `json:"stage"`
}

type Presentation struct {
	Frames []Frame `json:"frames"`
}

type Input struct {
	Catalog      Catalog      `json:"catalog"`
	Example      Example      `json:"example"`
	Diagram      Diagram      `json:"diagram"`
	Presentation Presentation `json:"presentation"`
}

type ShownRecord struct {
	ID     string                     `json:"id"`
	Values map[string]json.RawMessage `json:"values"`
}

type ShownTable struct {
	PlacementID string        `json:"placementId"`
	Table       string        `json:"table"`
	X           int           `json:"x"`
	Y           int           `json:"y"`
	Fields      []string      `json:"fields"`
	Records     []ShownRecord `json:"records"`
}

type Connection struct {
	Parent     string `json:"parent"`
	Child      string `json:"child"`
	ViaField   string `json:"viaField,omitempty"`
	RelationID string `json:"relationId,omitempty"`
}

type Slide struct {
	ID               string            `json:"id"`
	Title            string            `json:"title"`
	Stage            string            `json:"stage"`
	Tables           []ShownTable      `json:"tables"`
	Connections      []Connection      `json:"connections"`
	RelationFindings []RelationFinding `json:"relationFindings,omitempty"`
}

type Projection struct {
	ExampleID string     `json:"exampleId"`
	DiagramID string     `json:"diagramId"`
	Slides    []Slide    `json:"slides"`
	Questions []Question `json:"questions,omitempty"`
}

// Build validates example records, replays illustration-only changes, and
// projects snapshots onto manually positioned tables. It never writes input.
// Stage changes are not evidence of TEI Action or real-world execution.
func Build(in Input) (Projection, error) {
	var empty Projection
	if in.Catalog.Revision == "" || in.Example.ModelRevision != in.Catalog.Revision {
		return empty, fmt.Errorf("stale or missing model revision")
	}
	if in.Example.ID == "" || in.Diagram.ID == "" {
		return empty, fmt.Errorf("example and diagram IDs required")
	}
	if err := validateCatalog(in.Catalog); err != nil {
		return empty, err
	}
	if err := validateDiagram(in.Catalog, in.Diagram); err != nil {
		return empty, err
	}
	baseComplete, err := completeTableSet(in.Catalog, in.Example.CompleteTables)
	if err != nil {
		return empty, err
	}
	state := make(map[string]Record)
	for _, record := range in.Example.Base {
		if record.ID == "" || record.Table == "" {
			return empty, fmt.Errorf("record ID and table required")
		}
		if _, exists := state[record.ID]; exists {
			return empty, fmt.Errorf("duplicate record %q", record.ID)
		}
		state[record.ID] = copyRecord(record)
	}
	if err := validateState(in.Catalog, state); err != nil {
		return empty, fmt.Errorf("base: %w", err)
	}
	snapshots := map[string]map[string]Record{"base": copyState(state)}
	stageComplete := map[string]map[string]bool{"base": baseComplete}
	for _, stage := range in.Example.Stages {
		if stage.ID == "" || stage.ID == "base" {
			return empty, fmt.Errorf("invalid stage ID %q", stage.ID)
		}
		if _, exists := snapshots[stage.ID]; exists {
			return empty, fmt.Errorf("duplicate stage ID %q", stage.ID)
		}
		for _, change := range stage.Changes {
			record := change.Record
			previous, exists := state[record.ID]
			switch change.Op {
			case "create":
				if exists || record.ID == "" || record.Table == "" {
					return empty, fmt.Errorf("invalid create %q", record.ID)
				}
				state[record.ID] = copyRecord(record)
			case "update":
				if !exists || previous.Table != record.Table {
					return empty, fmt.Errorf("invalid update %q", record.ID)
				}
				next := copyRecord(previous)
				for key, value := range record.Values {
					next.Values[key] = append(json.RawMessage(nil), value...)
				}
				state[record.ID] = next
			case "delete":
				if !exists || previous.Table != record.Table {
					return empty, fmt.Errorf("invalid delete %q", record.ID)
				}
				delete(state, record.ID)
			default:
				return empty, fmt.Errorf("unknown change operation %q", change.Op)
			}
		}
		if err := validateState(in.Catalog, state); err != nil {
			return empty, fmt.Errorf("stage %s: %w", stage.ID, err)
		}
		snapshots[stage.ID] = copyState(state)
		complete, err := completeTableSet(in.Catalog, stage.CompleteTables)
		if err != nil {
			return empty, fmt.Errorf("stage %s: %w", stage.ID, err)
		}
		stageComplete[stage.ID] = complete
	}
	out := Projection{
		ExampleID: in.Example.ID, DiagramID: in.Diagram.ID,
		Questions: append([]Question(nil), in.Example.Questions...),
	}
	frameIDs := map[string]bool{}
	for _, frame := range in.Presentation.Frames {
		if frame.ID == "" || frameIDs[frame.ID] {
			return empty, fmt.Errorf("duplicate or empty frame ID %q", frame.ID)
		}
		frameIDs[frame.ID] = true
		snapshot, found := snapshots[frame.Stage]
		if !found {
			return empty, fmt.Errorf("frame %s: unknown stage %q", frame.ID, frame.Stage)
		}
		slide := Slide{ID: frame.ID, Title: frame.Title, Stage: frame.Stage}
		for _, placement := range in.Diagram.Placements {
			fields := shownFields(in.Catalog.Tables[placement.Table], placement)
			box := ShownTable{
				PlacementID: placement.ID, Table: placement.Table,
				X: placement.X, Y: placement.Y, Fields: fields,
			}
			var ids []string
			for id, record := range snapshot {
				if record.Table == placement.Table {
					ids = append(ids, id)
				}
			}
			sort.Strings(ids)
			for _, id := range ids {
				values := map[string]json.RawMessage{}
				for _, field := range fields {
					if value, found := snapshot[id].Values[field]; found {
						values[field] = append(json.RawMessage(nil), value...)
					}
				}
				box.Records = append(box.Records, ShownRecord{ID: id, Values: values})
			}
			slide.Tables = append(slide.Tables, box)
		}
		for _, child := range in.Diagram.Placements {
			for _, field := range in.Catalog.Tables[child.Table].Fields {
				if field.RefTable == "" {
					continue
				}
				for _, parent := range in.Diagram.Placements {
					if parent.Table == field.RefTable {
						slide.Connections = append(slide.Connections, Connection{
							Parent: parent.ID, Child: child.ID, ViaField: field.ID,
						})
					}
				}
			}
		}
		for _, relation := range in.Catalog.Relations {
			for _, source := range in.Diagram.Placements {
				if source.Table != relation.SourceTable {
					continue
				}
				for _, target := range in.Diagram.Placements {
					if target.Table != relation.TargetTable {
						continue
					}
					// Cardinality indicates how the relation's target is viewed:
					// "one" -> referenced parent target; "many" -> child target.
					parent, child := target.ID, source.ID
					if relation.Cardinality == "many" {
						parent, child = source.ID, target.ID
					}
					slide.Connections = append(slide.Connections, Connection{
						Parent: parent, Child: child, RelationID: relation.ID,
					})
				}
			}
		}
		slide.RelationFindings = assessRelations(in.Catalog, snapshot, stageComplete[frame.Stage])
		out.Slides = append(out.Slides, slide)
	}
	return out, nil
}

func validateCatalog(c Catalog) error {
	if len(c.Tables) == 0 {
		return fmt.Errorf("catalog has no tables")
	}
	for id, table := range c.Tables {
		if id == "" {
			return fmt.Errorf("empty table ID")
		}
		fields := map[string]bool{}
		for _, field := range table.Fields {
			if field.ID == "" || fields[field.ID] {
				return fmt.Errorf("%s: invalid field ID %q", id, field.ID)
			}
			fields[field.ID] = true
			switch field.Type {
			case "string", "integer", "number", "boolean", "ref", "opaque":
			default:
				return fmt.Errorf("%s.%s: unsupported type %q", id, field.ID, field.Type)
			}
			if field.Type == "ref" {
				if _, ok := c.Tables[field.RefTable]; field.RefTable == "" || !ok {
					return fmt.Errorf("%s.%s: unknown reference target", id, field.ID)
				}
			} else if field.RefTable != "" {
				return fmt.Errorf("%s.%s: refTable requires reference type", id, field.ID)
			}
		}
	}
	relationIDs := map[string]bool{}
	for _, relation := range c.Relations {
		if relationIDs[relation.ID] {
			return fmt.Errorf("duplicate semantic relation %q", relation.ID)
		}
		relationIDs[relation.ID] = true
		from, srcOK := c.Tables[relation.SourceTable]
		to, dstOK := c.Tables[relation.TargetTable]
		if relation.ID == "" || !srcOK || !dstOK {
			return fmt.Errorf("invalid semantic relation %q", relation.ID)
		}
		if relation.Cardinality != "one" && relation.Cardinality != "many" {
			return fmt.Errorf("%s: unknown cardinality %q", relation.ID, relation.Cardinality)
		}
		keyPairs := map[SemanticRelationKey]bool{}
		for _, key := range relation.Keys {
			if !hasField(from, key.SourceField) || !hasField(to, key.TargetField) {
				return fmt.Errorf("%s: unknown relation key %s -> %s", relation.ID, key.SourceField, key.TargetField)
			}
			if keyPairs[key] {
				return fmt.Errorf("%s: duplicate relation key pair", relation.ID)
			}
			keyPairs[key] = true
		}
	}
	return nil
}

func hasField(table Table, id string) bool {
	for _, field := range table.Fields {
		if field.ID == id {
			return true
		}
	}
	return false
}

func validateDiagram(c Catalog, d Diagram) error {
	placements := map[string]bool{}
	for _, placement := range d.Placements {
		if placement.ID == "" || placements[placement.ID] {
			return fmt.Errorf("duplicate or empty placement ID %q", placement.ID)
		}
		placements[placement.ID] = true
		table, ok := c.Tables[placement.Table]
		if !ok {
			return fmt.Errorf("unknown diagram table %q", placement.Table)
		}
		overrides := map[string]bool{}
		ids := append(append([]string(nil), placement.Show...), placement.Hide...)
		for _, id := range ids {
			if !hasField(table, id) || overrides[id] {
				return fmt.Errorf("%s: invalid display override %q", placement.ID, id)
			}
			overrides[id] = true
		}
	}
	return nil
}

func shownFields(table Table, placement Placement) []string {
	show, hide := map[string]bool{}, map[string]bool{}
	for _, id := range placement.Show {
		show[id] = true
	}
	for _, id := range placement.Hide {
		hide[id] = true
	}
	var fields []string
	for _, field := range table.Fields {
		if (field.Visible || show[field.ID]) && !hide[field.ID] {
			fields = append(fields, field.ID)
		}
	}
	return fields
}

func validateState(c Catalog, state map[string]Record) error {
	var ids []string
	for id := range state {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		record := state[id]
		table, found := c.Tables[record.Table]
		if !found {
			return fmt.Errorf("%s: unknown table %q", id, record.Table)
		}
		for key := range record.Values {
			if !hasField(table, key) {
				return fmt.Errorf("%s: unknown field %q", id, key)
			}
		}
		for _, field := range table.Fields {
			value, present := record.Values[field.ID]
			if !present || strings.TrimSpace(string(value)) == "null" {
				if field.Required {
					return fmt.Errorf("%s: required field %q absent", id, field.ID)
				}
				continue
			}
			if !json.Valid(value) {
				return fmt.Errorf("%s.%s: invalid JSON", id, field.ID)
			}
			switch field.Type {
			case "opaque":
				// Preserve original typed JSON without inventing a primitive check.
				// This is not evidence that the field's domain constraint holds.
			case "string", "ref":
				var text string
				if json.Unmarshal(value, &text) != nil {
					return fmt.Errorf("%s.%s: expected string", id, field.ID)
				}
				if field.Type == "ref" {
					target, exists := state[text]
					if !exists || target.Table != field.RefTable {
						return fmt.Errorf("%s.%s: broken reference %q", id, field.ID, text)
					}
				}
			case "integer", "number":
				// Do not round a JSON integer through float64. Values above
				// 2^53 can otherwise turn a fractional key into an integer.
				decoder := json.NewDecoder(bytes.NewReader(value))
				decoder.UseNumber()
				var decoded any
				if err := decoder.Decode(&decoded); err != nil {
					return fmt.Errorf("%s.%s: expected number", id, field.ID)
				}
				number, ok := decoded.(json.Number)
				if !ok || !boundedNumber(number.String()) {
					return fmt.Errorf("%s.%s: expected number (within bounded size)", id, field.ID)
				}
				rational, valid := new(big.Rat).SetString(number.String())
				if !valid {
					return fmt.Errorf("%s.%s: invalid numeric value", id, field.ID)
				}
				if field.Type == "integer" && !rational.IsInt() {
					return fmt.Errorf("%s.%s: expected integer", id, field.ID)
				}
			case "boolean":
				var b bool
				if json.Unmarshal(value, &b) != nil {
					return fmt.Errorf("%s.%s: expected boolean", id, field.ID)
				}
			}
		}
	}
	return nil
}

func copyRecord(record Record) Record {
	out := Record{ID: record.ID, Table: record.Table, Values: map[string]json.RawMessage{}}
	for key, value := range record.Values {
		out.Values[key] = append(json.RawMessage(nil), value...)
	}
	return out
}

func copyState(state map[string]Record) map[string]Record {
	out := make(map[string]Record, len(state))
	for id, record := range state {
		out[id] = copyRecord(record)
	}
	return out
}

// SVG renders the entire illustrative snapshot, not merely the first record.
// Authored positions are preserved; overlapping placements are not silently
// rearranged. It is an accessible proof view, not complete ER notation.
func SVG(slide Slide) string {
	const width = 264
	const rowHeight = 17
	height := func(box ShownTable) int {
		if len(box.Records) == 0 {
			return 46 + len(box.Fields)*rowHeight
		}
		return 34 + len(box.Records)*(rowHeight*(len(box.Fields)+1)+7)
	}
	minX, minY, maxX, maxY := 0, 0, 360, 140
	byID := map[string]ShownTable{}
	for _, box := range slide.Tables {
		byID[box.PlacementID] = box
		if box.X < minX {
			minX = box.X
		}
		if box.Y < minY {
			minY = box.Y
		}
		if box.X+width+12 > maxX {
			maxX = box.X + width + 12
		}
		if box.Y+height(box)+12 > maxY {
			maxY = box.Y + height(box) + 12
		}
	}
	// Reserve a header band for the slide context and the review result counts.
	left, top := minX-10, minY-58
	viewWidth, viewHeight := maxX-minX+24, maxY-minY+76
	var out strings.Builder
	fmt.Fprintf(&out, `<svg xmlns="http://www.w3.org/2000/svg" viewBox="%d %d %d %d" role="img">`, left, top, viewWidth, viewHeight)
	description := slide.Title
	if description == "" {
		description = slide.ID
	}
	fmt.Fprintf(&out, "<title>%s</title>", html.EscapeString(description))
	fmt.Fprintf(&out, `<text x="%d" y="%d" font-family="sans-serif" font-size="16">%s</text>`, left+5, top+21, html.EscapeString(description))
	counts := map[string]int{}
	for _, finding := range slide.RelationFindings {
		counts[finding.Result]++
	}
	badges := fmt.Sprintf("Stage: %s | Supported: %d | Contradictions: %d | Unresolved: %d", slide.Stage, counts["supported"], counts["contradiction"], counts["unresolved"])
	fmt.Fprintf(&out, `<text x="%d" y="%d" font-family="sans-serif" font-size="11">%s</text>`, left+5, top+41, html.EscapeString(badges))
	for _, edge := range slide.Connections {
		p, okP := byID[edge.Parent]
		c, okC := byID[edge.Child]
		if !okP || !okC {
			continue
		}
		label := edge.RelationID
		if label == "" {
			label = edge.ViaField
		}
		fmt.Fprintf(&out, `<path d="M%d %d L%d %d" fill="none" stroke="#667788" aria-label="%s"/>`, p.X+width, p.Y+18, c.X, c.Y+18, html.EscapeString(label))
	}
	for _, box := range slide.Tables {
		fmt.Fprintf(&out, `<rect x="%d" y="%d" width="%d" height="%d" fill="white" stroke="#334455"/>`, box.X, box.Y, width, height(box))
		fmt.Fprintf(&out, `<text x="%d" y="%d" font-family="sans-serif" font-size="13">%s</text>`, box.X+7, box.Y+20, html.EscapeString(box.Table))
		if len(box.Records) == 0 {
			fmt.Fprintf(&out, `<text x="%d" y="%d" font-family="sans-serif" font-size="10">(no example records)</text>`, box.X+7, box.Y+34)
			for i, field := range box.Fields {
				fmt.Fprintf(&out, `<text x="%d" y="%d" font-family="sans-serif" font-size="11">%s</text>`, box.X+7, box.Y+50+i*rowHeight, html.EscapeString(shortLabel(field, 35)))
			}
			continue
		}
		cursor := box.Y + 36
		for _, record := range box.Records {
			fmt.Fprintf(&out, `<text x="%d" y="%d" font-family="sans-serif" font-size="11" font-weight="bold">%s</text>`, box.X+7, cursor, html.EscapeString(shortLabel(record.ID, 38)))
			cursor += rowHeight
			for _, field := range box.Fields {
				value := "∅"
				if v, ok := record.Values[field]; ok {
					value = formatSlideValue(v)
				}
				label := shortLabel(field+": "+value, 39)
				fmt.Fprintf(&out, `<text x="%d" y="%d" font-family="sans-serif" font-size="11">%s</text>`, box.X+12, cursor, html.EscapeString(label))
				cursor += rowHeight
			}
			cursor += 7
		}
	}
	out.WriteString("</svg>")
	return out.String()
}

func formatSlideValue(raw json.RawMessage) string {
	if len(raw) > 0 && raw[0] == '"' {
		var value string
		if json.Unmarshal(raw, &value) == nil {
			return value
		}
	}
	return string(raw)
}

func shortLabel(value string, maxRunes int) string {
	runes := []rune(value)
	if len(runes) > maxRunes {
		return string(runes[:maxRunes-1]) + "…"
	}
	return value
}
