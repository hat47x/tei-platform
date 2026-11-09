package xeadsource

import (
	"sort"
	"strings"
)

const SourceFactNature = "source-fact"

var legacyKeyAuthorityLimits = []string{
	"legacy-key-is-not-business-identity",
	"legacy-key-is-not-candidate-key",
	"legacy-key-is-not-functional-dependency",
	"legacy-key-is-not-physical-key-authority",
}

var relationshipKeyAuthorityLimits = []string{
	"relationship-key-reference-is-not-business-identity",
	"relationship-key-reference-is-not-functional-dependency",
	"ambiguous-key-field-mapping-must-remain-unresolved",
}

var declaredKeyFieldsAuthorityLimits = []string{
	"declared-key-fields-are-not-data-identity",
	"declared-key-fields-are-not-business-identity",
	"declared-key-fields-are-not-functional-dependency",
}

var runtimeKeyAuthorityLimits = []string{
	"runtime-key-is-realization-not-business-identity",
	"runtime-key-is-not-functional-dependency",
	"runtime-key-does-not-replace-design-key-evidence",
}

var runtimeReferAuthorityLimits = []string{
	"runtime-refer-is-realization-not-semantic-relation-authority",
	"runtime-refer-key-fields-are-not-business-identity",
	"runtime-pk-fallback-is-not-business-identity",
}

type LegacyKeyMemberFact struct {
	Locator          string `json:"locator"`
	FieldLegacyID    string `json:"fieldLegacyId"`
	AttributeType    string `json:"attributeType,omitempty"`
	DataTypeLegacyID string `json:"dataTypeLegacyId,omitempty"`
	SortKey          string `json:"sortKey,omitempty"`
	SourceOrdinal    int    `json:"sourceOrdinal"`
}

type LegacyKeyFact struct {
	Locator         string                `json:"locator"`
	TableLegacyID   string                `json:"tableLegacyId,omitempty"`
	KeyLegacyID     string                `json:"keyLegacyId,omitempty"`
	KeyType         string                `json:"keyType,omitempty"`
	Members         []LegacyKeyMemberFact `json:"members,omitempty"`
	Nature          string                `json:"nature"`
	AuthorityLimits []string              `json:"authorityLimits"`
}

type RelationshipKeyFact struct {
	Locator          string   `json:"locator"`
	RelationshipID   string   `json:"relationshipId,omitempty"`
	RelationshipType string   `json:"relationshipType,omitempty"`
	ExistWhen1       *string  `json:"existWhen1,omitempty"`
	ExistWhen2       *string  `json:"existWhen2,omitempty"`
	Table1LegacyID   string   `json:"table1LegacyId,omitempty"`
	Key1LegacyID     string   `json:"key1LegacyId,omitempty"`
	Table2LegacyID   string   `json:"table2LegacyId,omitempty"`
	Key2LegacyID     string   `json:"key2LegacyId,omitempty"`
	Nature           string   `json:"nature"`
	AuthorityLimits  []string `json:"authorityLimits"`
}

type DeclaredKeyFieldsFact struct {
	Locator         string   `json:"locator"`
	SourceName      string   `json:"sourceName"`
	OwnerLegacyID   string   `json:"ownerLegacyId,omitempty"`
	Fields          []string `json:"fields"`
	Nature          string   `json:"nature"`
	AuthorityLimits []string `json:"authorityLimits"`
}

type RuntimeKeyFact struct {
	Locator         string   `json:"locator"`
	TableLegacyID   string   `json:"tableLegacyId,omitempty"`
	KeyLegacyID     string   `json:"keyLegacyId,omitempty"`
	KeyType         string   `json:"keyType,omitempty"`
	Fields          []string `json:"fields,omitempty"`
	Nature          string   `json:"nature"`
	AuthorityLimits []string `json:"authorityLimits"`
}

type RuntimeReferFact struct {
	Locator             string   `json:"locator"`
	ReferLegacyID       string   `json:"referLegacyId,omitempty"`
	SourceTableLegacyID string   `json:"sourceTableLegacyId,omitempty"`
	ToTableLegacyID     string   `json:"toTableLegacyId,omitempty"`
	TableAlias          string   `json:"tableAlias,omitempty"`
	ToKeyFields         []string `json:"toKeyFields,omitempty"`
	WithKeyFields       []string `json:"withKeyFields,omitempty"`
	// Preserve the literal declarations independently of normalized field lists.
	// A nil pointer means the attribute was absent; a non-nil empty value
	// means it was explicitly declared as empty in the source.
	ToKeyFieldsRaw   *string  `json:"toKeyFieldsRaw,omitempty"`
	WithKeyFieldsRaw *string  `json:"withKeyFieldsRaw,omitempty"`
	Nature           string   `json:"nature"`
	AuthorityLimits  []string `json:"authorityLimits"`
}

type RelationalSourceFacts struct {
	LegacyKeys        []LegacyKeyFact         `json:"legacyKeys,omitempty"`
	RelationshipKeys  []RelationshipKeyFact   `json:"relationshipKeys,omitempty"`
	DeclaredKeyFields []DeclaredKeyFieldsFact `json:"declaredKeyFields,omitempty"`
	RuntimeKeys       []RuntimeKeyFact        `json:"runtimeKeys,omitempty"`
	RuntimeRefers     []RuntimeReferFact      `json:"runtimeRefers,omitempty"`
}

// ExtractRelationalSourceFacts projects only facts explicitly present in the
// source snapshot. It deliberately does not infer business identifiers,
// candidate keys, physical key authority, or functional dependencies.
func ExtractRelationalSourceFacts(doc Document) RelationalSourceFacts {
	var facts RelationalSourceFacts
	tableFields := indexTableFieldIdentity(doc.Root).Fields

	var walk func(Node, string)
	walk = func(n Node, tableLegacyID string) {
		if !n.Known {
			// Preserve unknown nodes in the raw Reader. A foreign Table
			// lookalike is an ownership boundary: its known children must
			// not inherit the enclosing real Table's identity.
			if localName(n.Name) == "Table" {
				tableLegacyID = ""
			}
			for _, child := range n.Children {
				walk(child, tableLegacyID)
			}
			return
		}
		if n.Name == "Table" {
			tableLegacyID = attributeValue(n, "ID")
		}

		switch n.Name {
		case "TableKey":
			key := LegacyKeyFact{
				Locator:         n.Locator,
				TableLegacyID:   tableLegacyID,
				KeyLegacyID:     attributeValue(n, "ID"),
				KeyType:         attributeValue(n, "Type"),
				Nature:          SourceFactNature,
				AuthorityLimits: cloneStrings(legacyKeyAuthorityLimits),
			}
			for i, child := range n.Children {
				if !child.Known || child.Name != "TableKeyField" {
					continue
				}
				fieldLegacyID := attributeValue(child, "FieldID")
				key.Members = append(key.Members, LegacyKeyMemberFact{
					Locator:          child.Locator,
					FieldLegacyID:    fieldLegacyID,
					AttributeType:    tableFields[tableLegacyID][fieldLegacyID].AttributeType,
					DataTypeLegacyID: tableFields[tableLegacyID][fieldLegacyID].DataTypeLegacyID,
					SortKey:          attributeValue(child, "SortKey"),
					SourceOrdinal:    i,
				})
			}
			facts.LegacyKeys = append(facts.LegacyKeys, key)

		case "Relationship":
			relationshipType, _ := exactAttribute(n, "Type")
			facts.RelationshipKeys = append(facts.RelationshipKeys, RelationshipKeyFact{
				Locator:          n.Locator,
				RelationshipID:   attributeValue(n, "ID"),
				RelationshipType: relationshipType,
				ExistWhen1:       exactOptionalAttribute(n, "ExistWhen1"),
				ExistWhen2:       exactOptionalAttribute(n, "ExistWhen2"),
				Table1LegacyID:   attributeValue(n, "Table1ID"),
				Key1LegacyID:     attributeValue(n, "TableKey1ID"),
				Table2LegacyID:   attributeValue(n, "Table2ID"),
				Key2LegacyID:     attributeValue(n, "TableKey2ID"),
				Nature:           SourceFactNature,
				AuthorityLimits:  cloneStrings(relationshipKeyAuthorityLimits),
			})
		case "Key":
			if doc.Format == FormatXeaf {
				facts.RuntimeKeys = append(facts.RuntimeKeys, RuntimeKeyFact{
					Locator:         n.Locator,
					TableLegacyID:   tableLegacyID,
					KeyLegacyID:     attributeValue(n, "ID"),
					KeyType:         attributeValue(n, "Type"),
					Fields:          splitLegacyFieldList(attributeValue(n, "Fields")),
					Nature:          SourceFactNature,
					AuthorityLimits: cloneStrings(runtimeKeyAuthorityLimits),
				})
			}
		case "Refer":
			// A foreign-namespaced lookalike is an unknown source node,
			// not a typed XEAF runtime Refer declaration.
			if doc.Format == FormatXeaf {
				toKeyFieldsRaw := exactOptionalAttribute(n, "ToKeyFields")
				withKeyFieldsRaw := exactOptionalAttribute(n, "WithKeyFields")
				toKeyFields := ""
				withKeyFields := ""
				if toKeyFieldsRaw != nil {
					toKeyFields = *toKeyFieldsRaw
				}
				if withKeyFieldsRaw != nil {
					withKeyFields = *withKeyFieldsRaw
				}
				facts.RuntimeRefers = append(facts.RuntimeRefers, RuntimeReferFact{
					Locator:             n.Locator,
					ReferLegacyID:       exactAttributeValue(n, "ID"),
					SourceTableLegacyID: tableLegacyID,
					ToTableLegacyID:     exactAttributeValue(n, "ToTable"),
					TableAlias:          exactAttributeValue(n, "TableAlias"),
					ToKeyFields:         splitLegacyFieldList(toKeyFields),
					WithKeyFields:       splitLegacyFieldList(withKeyFields),
					ToKeyFieldsRaw:      toKeyFieldsRaw,
					WithKeyFieldsRaw:    withKeyFieldsRaw,
					Nature:              SourceFactNature,
					AuthorityLimits:     cloneStrings(runtimeReferAuthorityLimits),
				})
			}
		}

		if raw := attributeValue(n, "KeyFields"); raw != "" {
			fields := splitLegacyFieldList(raw)
			facts.DeclaredKeyFields = append(facts.DeclaredKeyFields, DeclaredKeyFieldsFact{
				Locator:         n.Locator,
				SourceName:      n.Name,
				OwnerLegacyID:   attributeValue(n, "ID"),
				Fields:          fields,
				Nature:          SourceFactNature,
				AuthorityLimits: cloneStrings(declaredKeyFieldsAuthorityLimits),
			})
		}

		for _, child := range n.Children {
			walk(child, tableLegacyID)
		}
	}
	walk(doc.Root, "")

	sort.Slice(facts.LegacyKeys, func(i, j int) bool {
		return facts.LegacyKeys[i].Locator < facts.LegacyKeys[j].Locator
	})
	sort.Slice(facts.RelationshipKeys, func(i, j int) bool {
		return facts.RelationshipKeys[i].Locator < facts.RelationshipKeys[j].Locator
	})
	sort.Slice(facts.DeclaredKeyFields, func(i, j int) bool {
		return facts.DeclaredKeyFields[i].Locator < facts.DeclaredKeyFields[j].Locator
	})
	sort.Slice(facts.RuntimeKeys, func(i, j int) bool {
		return facts.RuntimeKeys[i].Locator < facts.RuntimeKeys[j].Locator
	})
	sort.Slice(facts.RuntimeRefers, func(i, j int) bool {
		return facts.RuntimeRefers[i].Locator < facts.RuntimeRefers[j].Locator
	})
	return facts
}

type tableFieldMetadata struct {
	AttributeType    string
	DataTypeLegacyID string
}

type tableFieldIdentityIndex struct {
	Fields        map[string]map[string]tableFieldMetadata
	TableCounts   map[string]int
	TableLocators map[string][]string
	FieldCounts   map[string]map[string]int
	FieldLocators map[string]map[string][]string
}

func indexTableFieldIdentity(root Node) tableFieldIdentityIndex {
	out := tableFieldIdentityIndex{
		Fields:        map[string]map[string]tableFieldMetadata{},
		TableCounts:   map[string]int{},
		TableLocators: map[string][]string{},
		FieldCounts:   map[string]map[string]int{},
		FieldLocators: map[string]map[string][]string{},
	}
	var walk func(Node)
	walk = func(n Node) {
		if n.Known && n.Name == "Table" {
			tableID := attributeValue(n, "ID")
			out.TableCounts[tableID]++
			out.TableLocators[tableID] = append(out.TableLocators[tableID], n.Locator)
			if out.TableCounts[tableID] == 1 {
				out.Fields[tableID] = map[string]tableFieldMetadata{}
				out.FieldCounts[tableID] = map[string]int{}
				out.FieldLocators[tableID] = map[string][]string{}
			} else {
				delete(out.Fields, tableID)
			}

			var collectFields func(Node)
			collectFields = func(current Node) {
				for _, child := range current.Children {
					// A nested foreign Table must not donate its Field
					// declarations to the enclosing real Table.
					if localName(child.Name) == "Table" {
						continue
					}
					if child.Known && (child.Name == "TableField" || child.Name == "Field") {
						fieldID := attributeValue(child, "ID")
						if _, ok := out.FieldCounts[tableID]; !ok {
							out.FieldCounts[tableID] = map[string]int{}
							out.FieldLocators[tableID] = map[string][]string{}
						}
						out.FieldCounts[tableID][fieldID]++
						out.FieldLocators[tableID][fieldID] = append(out.FieldLocators[tableID][fieldID], child.Locator)
						if out.TableCounts[tableID] == 1 && out.FieldCounts[tableID][fieldID] == 1 {
							out.Fields[tableID][fieldID] = tableFieldMetadata{
								AttributeType:    attributeValue(child, "AttributeType"),
								DataTypeLegacyID: attributeValue(child, "DataTypeID"),
							}
						} else if fields, ok := out.Fields[tableID]; ok {
							delete(fields, fieldID)
						}
					}
					collectFields(child)
				}
			}
			collectFields(n)
		}
		for _, child := range n.Children {
			walk(child)
		}
	}
	walk(root)
	return out
}

func attributeValue(n Node, name string) string {
	// Source Fact lookup must not promote a foreign-namespaced lookalike.
	return exactAttributeValue(n, name)
}

// exactAttributeValue reads only the unqualified declaration used by a
// typed Source Fact. Foreign-namespaced lookalikes remain on the raw Node.
func exactAttributeValue(n Node, name string) string {
	value, _ := exactAttribute(n, name)
	return value
}

// exactOptionalAttribute keeps a declared empty source value distinct from absence.
func exactOptionalAttribute(n Node, name string) *string {
	value, ok := exactAttribute(n, name)
	if !ok {
		return nil
	}
	return &value
}

func splitLegacyFieldList(raw string) []string {
	parts := strings.Split(raw, ";")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		if value := strings.TrimSpace(part); value != "" {
			out = append(out, value)
		}
	}
	return out
}

func cloneStrings(values []string) []string {
	return append([]string(nil), values...)
}
