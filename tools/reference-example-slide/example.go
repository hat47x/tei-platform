package exampleslide

import (
	"encoding/json"
	"fmt"
	"html"
	"math"
	"sort"
	"strings"
)

// Catalog is a read-only adapter input. It is not a TEI canonical source.
type Catalog struct {
	Revision string           `json:"revision"`
	Tables   map[string]Table `json:"tables"`
	Relations []SemanticRelation `json:"relations,omitempty"`
}

type Table struct {
	Fields []Field `json:"fields"`
}

type Field struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	Required bool   `json:"required,omitempty"`
	Visible  bool   `json:"visible,omitempty"`
	RefTable string `json:"refTable,omitempty"`
}

// SemanticRelation preserves directed semantic relationships separately from
// physical foreign keys. The pair of attribute IDs is not a record-ID link.
type SemanticRelation struct {
	ID          string `json:"id"`
	SourceTable string `json:"sourceTable"`
	TargetTable string `json:"targetTable"`
	Cardinality string `json:"cardinality"`
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
	// CompleteTables lists the tables whose records are known to be fully
	// enumerated at each illustration stage. Omitted tables remain partial.
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
	Parent   string `json:"parent"`
	Child    string `json:"child"`
	ViaField string `json:"viaField,omitempty"`
	RelationID string `json:"relationId,omitempty"`
}

type Slide struct {
	ID          string       `json:"id"`
	Title       string       `json:"title"`
	Stage       string       `json:"stage"`
	Tables      []ShownTable `json:"tables"`
	Connections []Connection `json:"connections"`
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
	complete, err := completeTableSet(in.Catalog, in.Example.CompleteTables)
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
				if source.Table != relation.SourceTable { continue }
				for _, target := range in.Diagram.Placements {
					if target.Table != relation.TargetTable { continue }
					// Cardinality indicates how the relation's target is viewed:
					// "one" -> referenced parent target; "many" -> child target.
					parent, child := target.ID, source.ID
					if relation.Cardinality == "many" {
						parent, child = source.ID, target.ID
					}
					slide.Connections = append(slide.Connections, Connection{
						Parent:parent, Child:child, RelationID:relation.ID,
					})
				}
			}
		}
		slide.RelationFindings = assessRelations(in.Catalog, snapshot, complete)
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
			return fmt.Errorf("invalid semantic relation %q",relation.ID)
		}
		if relation.Cardinality != "one" && relation.Cardinality != "many" {
			return fmt.Errorf("%s: unknown cardinality %q",relation.ID,relation.Cardinality)
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
				var number float64
				if json.Unmarshal(value, &number) != nil || math.IsInf(number, 0) || math.IsNaN(number) {
					return fmt.Errorf("%s.%s: expected number", id, field.ID)
				}
				if field.Type == "integer" && math.Trunc(number) != number {
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

// SVG is a deterministic proof renderer; it is not a complete ER notation.
// It keeps authored coordinates even when a parent appears right of a child.
func SVG(slide Slide) string {
	const width = 225
	height := func(table ShownTable) int { return 42 + len(table.Fields)*20 }
	byID := map[string]ShownTable{}
	minX, minY, maxX, maxY := 0, 0, 360, 140
	for _, table := range slide.Tables {
		byID[table.PlacementID] = table
		if table.X < minX {
			minX = table.X
		}
		if table.Y < minY {
			minY = table.Y
		}
		if table.X+width+12 > maxX {
			maxX = table.X + width + 12
		}
		if table.Y+height(table)+12 > maxY {
			maxY = table.Y + height(table) + 12
		}
	}
	var out strings.Builder
	fmt.Fprintf(&out, `<svg xmlns="http://www.w3.org/2000/svg" viewBox="%d %d %d %d" role="img">`,
		minX-10, minY-10, maxX-minX+20, maxY-minY+20)
	for _, edge := range slide.Connections {
		parent, pok := byID[edge.Parent]
		child, cok := byID[edge.Child]
		if pok && cok {
			fmt.Fprintf(&out, `<path d="M%d %d L%d %d" fill="none" stroke="#667788"/>`,
				parent.X+width, parent.Y+21, child.X, child.Y+21)
		}
	}
	for _, table := range slide.Tables {
		fmt.Fprintf(&out, `<rect x="%d" y="%d" width="%d" height="%d" fill="white" stroke="#334455"/>`,
			table.X, table.Y, width, height(table))
		fmt.Fprintf(&out, `<text x="%d" y="%d" font-family="sans-serif" font-size="13">%s</text>`,
			table.X+7, table.Y+19, html.EscapeString(table.Table))
		for i, field := range table.Fields {
			label := field
			if len(table.Records) > 0 {
				if value, ok := table.Records[0].Values[field]; ok {
					label += ": " + string(value)
				}
			}
			runes := []rune(label)
			if len(runes) > 30 {
				label = string(runes[:30]) + "…"
			}
			fmt.Fprintf(&out, `<text x="%d" y="%d" font-family="sans-serif" font-size="11">%s</text>`,
				table.X+7, table.Y+38+i*20, html.EscapeString(label))
		}
	}
	out.WriteString("</svg>")
	return out.String()
}
