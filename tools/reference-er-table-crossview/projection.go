// Package ercrossview projects accepted-structure World data into a
// read-only joint logical ER and physical table comparison.
package ercrossview

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"sort"

	"tei/internal/core"
	"tei/internal/ir"
)

// Views represent explicit declarations only. They never create DDL,
// physical constraints, an accepted change, or verified business facts.
type Projection struct {
	Schema            string          `json:"schema"`
	StructureRevision string          `json:"structureRevision"`
	Data              []DataView      `json:"data"`
	Relations         []RelationView  `json:"relations"`
	Tables            []TableView     `json:"tables"`
	KeyComparisons    []KeyComparison `json:"keyComparisons"`
	Unresolved        []Unresolved    `json:"unresolved"`
}

type DataView struct {
	ID                 string          `json:"id"`
	Symbol             string          `json:"symbol"`
	Attributes         []AttributeView `json:"attributes"`
	CandidateKeys      []LogicalKey    `json:"candidateKeys"`
	Dependencies       []Dependency    `json:"functionalDependencies"`
	PhysicalMappingIDs []string        `json:"physicalMappingIds"`
}

type AttributeView struct {
	ID     string `json:"id"`
	Symbol string `json:"symbol"`
	Type   string `json:"type"`
}

type LogicalKey struct {
	ID         string   `json:"id"`
	Symbol     string   `json:"symbol"`
	Attributes []string `json:"attributes"`
}

type Dependency struct {
	ID           string   `json:"id"`
	Symbol       string   `json:"symbol"`
	Determinants []string `json:"determinants"`
	Dependents   []string `json:"dependents"`
}

type RelationView struct {
	ID          string        `json:"id"`
	Symbol      string        `json:"symbol"`
	SourceData  string        `json:"sourceData"`
	TargetData  string        `json:"targetData"`
	Cardinality string        `json:"cardinality"`
	Keys        []RelationKey `json:"keys"`
}

type RelationKey struct {
	SourceAttribute string `json:"sourceAttribute"`
	TargetAttribute string `json:"targetAttribute"`
}

type TableView struct {
	MappingID      string       `json:"mappingId"`
	DatabaseID     string       `json:"databaseId"`
	SemanticDataID string       `json:"semanticDataId"`
	Name           string       `json:"name"`
	Columns        []ColumnView `json:"columns"`
	PrimaryKey     []string     `json:"primaryKey"`
	UniqueKeys     []UniqueKey  `json:"uniqueKeys"`
	ForeignKeys    []ForeignKey `json:"foreignKeys"`
}

type ColumnView struct {
	Attribute   string `json:"attribute"`
	Name        string `json:"name"`
	Storage     string `json:"storage"`
	Nullability string `json:"nullability"`
}

type UniqueKey struct {
	Name       string   `json:"name"`
	Attributes []string `json:"attributes"`
}

type ForeignKey struct {
	Name                 string   `json:"name"`
	Attributes           []string `json:"attributes"`
	ReferencedMapping    string   `json:"referencedMapping"`
	ReferencedAttributes []string `json:"referencedAttributes"`
}

// Same attribute set is a textual comparison, not authorization to turn the
// logical rule into a PK or proof of provider-specific uniqueness.
type KeyComparison struct {
	LogicalKeyID   string `json:"logicalKeyId"`
	MappingID      string `json:"mappingId"`
	ConstraintKind string `json:"constraintKind"`
	ConstraintName string `json:"constraintName,omitempty"`
	Status         string `json:"status"`
}

type Unresolved struct {
	Code    string `json:"code"`
	Subject string `json:"subject"`
}

func Project(world ir.World) (Projection, error) {
	out := Projection{
		Schema: "tei.reference.er-physical-crossview/v0",
		Data:   []DataView{}, Relations: []RelationView{}, Tables: []TableView{},
		KeyComparisons: []KeyComparison{}, Unresolved: []Unresolved{},
	}
	b, err := json.Marshal(struct {
		Semantics   interface{} `json:"semantics"`
		Databases   interface{} `json:"databases"`
		Persistence interface{} `json:"persistenceMappings"`
	}{world.Semantics, world.Databases, world.Persistence})
	if err != nil {
		return Projection{}, err
	}
	out.StructureRevision = fmt.Sprintf("sha256:%x", sha256.Sum256(b))
	ids := map[string]bool{}
	attrByData := map[string]map[string]bool{}
	for _, ns := range world.Semantics {
		for _, data := range ns.Data {
			id := string(data.ID)
			if id == "" || ids[id] {
				return Projection{}, fmt.Errorf("duplicate or empty semantic Data %q", id)
			}
			ids[id] = true
			d := DataView{
				ID: id, Symbol: data.Symbol,
				Attributes: []AttributeView{}, CandidateKeys: []LogicalKey{},
				Dependencies: []Dependency{}, PhysicalMappingIDs: []string{},
			}
			attrs := map[string]bool{}
			for _, a := range data.Attributes {
				key := string(a.ID)
				if key == "" || attrs[key] {
					return Projection{}, fmt.Errorf("duplicate or empty attribute %s in %s", key, id)
				}
				attrs[key] = true
				d.Attributes = append(d.Attributes, AttributeView{ID: key, Symbol: a.Symbol, Type: string(a.TypeRef)})
			}
			attrByData[id] = attrs
			for _, k := range data.CandidateKeys {
				d.CandidateKeys = append(d.CandidateKeys, LogicalKey{
					ID: string(k.ID), Symbol: k.Symbol, Attributes: stringsOf(k.Attributes),
				})
			}
			for _, f := range data.FunctionalDependencies {
				d.Dependencies = append(d.Dependencies, Dependency{
					ID: string(f.ID), Symbol: f.Symbol, Determinants: stringsOf(f.Determinants),
					Dependents: stringsOf(f.Dependents),
				})
			}
			for _, r := range data.Relations {
				rel := RelationView{ID: string(r.ID), Symbol: r.Symbol, SourceData: id,
					TargetData: string(r.Target), Cardinality: string(r.Cardinality), Keys: []RelationKey{}}
				for _, k := range r.Keys {
					rel.Keys = append(rel.Keys, RelationKey{
						SourceAttribute: string(k.SourceAttribute), TargetAttribute: string(k.TargetAttribute),
					})
				}
				out.Relations = append(out.Relations, rel)
			}
			sort.Slice(d.CandidateKeys, func(i, j int) bool { return d.CandidateKeys[i].ID < d.CandidateKeys[j].ID })
			sort.Slice(d.Dependencies, func(i, j int) bool { return d.Dependencies[i].ID < d.Dependencies[j].ID })
			out.Data = append(out.Data, d)
		}
	}
	sort.Slice(out.Data, func(i, j int) bool { return out.Data[i].ID < out.Data[j].ID })
	sort.Slice(out.Relations, func(i, j int) bool { return out.Relations[i].ID < out.Relations[j].ID })
	dataIndex := map[string]*DataView{}
	for i := range out.Data {
		dataIndex[out.Data[i].ID] = &out.Data[i]
	}
	for _, rel := range out.Relations {
		if !ids[rel.TargetData] {
			out.Unresolved = append(out.Unresolved, Unresolved{Code: "RELATION_TARGET_NOT_PRESENT", Subject: rel.ID})
		}
	}
	mappingIDs := map[string]bool{}
	for _, m := range world.Persistence {
		id, owner := string(m.ID), string(m.SemanticData)
		if id == "" || mappingIDs[id] {
			return Projection{}, fmt.Errorf("duplicate or empty mapping ID %q", id)
		}
		mappingIDs[id] = true
		table := TableView{
			MappingID: id, DatabaseID: string(m.Database), SemanticDataID: owner,
			Name: m.Table, Columns: []ColumnView{}, PrimaryKey: stringsOf(m.PrimaryKey),
			UniqueKeys: []UniqueKey{}, ForeignKeys: []ForeignKey{},
		}
		mapped := map[string]bool{}
		for _, col := range m.Columns {
			attr := string(col.Attribute)
			if mapped[attr] {
				return Projection{}, fmt.Errorf("mapping %s repeats attribute %s", id, attr)
			}
			mapped[attr] = true
			table.Columns = append(table.Columns, ColumnView{
				Attribute: attr, Name: col.Column, Storage: string(col.Storage),
				Nullability: string(col.Nullability),
			})
			if !attrByData[owner][attr] {
				out.Unresolved = append(out.Unresolved, Unresolved{
					Code: "MAPPED_ATTRIBUTE_NOT_IN_DATA", Subject: id + ":" + attr,
				})
			}
		}
		for _, a := range m.PrimaryKey {
			if !mapped[string(a)] {
				out.Unresolved = append(out.Unresolved, Unresolved{
					Code: "PK_ATTRIBUTE_HAS_NO_COLUMN", Subject: id + ":" + string(a),
				})
			}
		}
		for _, u := range m.UniqueConstraints {
			table.UniqueKeys = append(table.UniqueKeys, UniqueKey{Name: u.Name, Attributes: stringsOf(u.Attributes)})
		}
		for _, fk := range m.ForeignKeys {
			table.ForeignKeys = append(table.ForeignKeys, ForeignKey{
				Name: fk.Name, Attributes: stringsOf(fk.Attributes),
				ReferencedMapping:    string(fk.ReferencedMapping),
				ReferencedAttributes: stringsOf(fk.ReferencedAttributes),
			})
		}
		out.Tables = append(out.Tables, table)
		if dataIndex[owner] == nil {
			out.Unresolved = append(out.Unresolved, Unresolved{Code: "MAPPING_DATA_NOT_PRESENT", Subject: id})
		} else {
			dataIndex[owner].PhysicalMappingIDs = append(dataIndex[owner].PhysicalMappingIDs, id)
		}
	}
	sort.Slice(out.Tables, func(i, j int) bool { return out.Tables[i].MappingID < out.Tables[j].MappingID })
	out.Unresolved = append(out.Unresolved, physicalConstraintFindings(out.Tables)...)
	for i := range out.Data {
		sort.Strings(out.Data[i].PhysicalMappingIDs)
		if len(out.Data[i].PhysicalMappingIDs) == 0 {
			out.Unresolved = append(out.Unresolved, Unresolved{
				Code: "NO_EXPLICIT_PHYSICAL_MAPPING", Subject: out.Data[i].ID,
			})
		}
	}
	for _, table := range out.Tables {
		owner := dataIndex[table.SemanticDataID]
		if owner == nil {
			continue
		}
		for _, k := range owner.CandidateKeys {
			add := func(kind, name string, attrs []string) {
				status := "distinct-attribute-set"
				if !constraintHasMappedColumns(table, attrs) {
					// The physical declaration refers to absent or duplicate
					// columns, so even a textual set match is inconclusive.
					status = "unresolved-physical-constraint"
				} else if sameSet(k.Attributes, attrs) {
					status = "same-attribute-set"
				}
				out.KeyComparisons = append(out.KeyComparisons, KeyComparison{
					LogicalKeyID: k.ID, MappingID: table.MappingID,
					ConstraintKind: kind, ConstraintName: name, Status: status,
				})
			}
			if len(table.PrimaryKey) > 0 {
				add("primary-key", "", table.PrimaryKey)
			}
			for _, u := range table.UniqueKeys {
				add("unique", u.Name, u.Attributes)
			}
		}
	}
	sort.Slice(out.KeyComparisons, func(i, j int) bool {
		a, b := out.KeyComparisons[i], out.KeyComparisons[j]
		return a.LogicalKeyID+"\x00"+a.MappingID+"\x00"+a.ConstraintKind+"\x00"+a.ConstraintName <
			b.LogicalKeyID+"\x00"+b.MappingID+"\x00"+b.ConstraintKind+"\x00"+b.ConstraintName
	})
	sort.Slice(out.Unresolved, func(i, j int) bool {
		a, b := out.Unresolved[i], out.Unresolved[j]
		return a.Code+"\x00"+a.Subject < b.Code+"\x00"+b.Subject
	})
	return out, nil
}

func stringsOf(ids []core.CanonicalID) []string {
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		out = append(out, string(id))
	}
	return out
}

func sameSet(a, b []string) bool {
	if len(a) == 0 || len(a) != len(b) {
		return false
	}
	x, y := append([]string(nil), a...), append([]string(nil), b...)
	sort.Strings(x)
	sort.Strings(y)
	for i := range x {
		if x[i] != y[i] || (i > 0 && (x[i] == x[i-1] || y[i] == y[i-1])) {
			return false
		}
	}
	return true
}
