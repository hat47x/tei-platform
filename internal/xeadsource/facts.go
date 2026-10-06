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
	Locator       string `json:"locator"`
	FieldLegacyID string `json:"fieldLegacyId"`
	AttributeType string `json:"attributeType,omitempty"`
	SortKey       string `json:"sortKey,omitempty"`
	SourceOrdinal int    `json:"sourceOrdinal"`
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
	Locator         string   `json:"locator"`
	RelationshipID  string   `json:"relationshipId,omitempty"`
	Table1LegacyID  string   `json:"table1LegacyId,omitempty"`
	Key1LegacyID    string   `json:"key1LegacyId,omitempty"`
	Table2LegacyID  string   `json:"table2LegacyId,omitempty"`
	Key2LegacyID    string   `json:"key2LegacyId,omitempty"`
	Nature          string   `json:"nature"`
	AuthorityLimits []string `json:"authorityLimits"`
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
	Nature              string   `json:"nature"`
	AuthorityLimits     []string `json:"authorityLimits"`
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
	tableFieldTypes := indexTableFieldTypes(doc.Root)

	var walk func(Node, string)
	walk = func(n Node, tableLegacyID string) {
		if localName(n.Name) == "Table" {
			tableLegacyID = attributeValue(n, "ID")
		}

		switch localName(n.Name) {
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
				if localName(child.Name) != "TableKeyField" {
					continue
				}
				fieldLegacyID := attributeValue(child, "FieldID")
				key.Members = append(key.Members, LegacyKeyMemberFact{
					Locator:       child.Locator,
					FieldLegacyID: fieldLegacyID,
					AttributeType: tableFieldTypes[tableLegacyID][fieldLegacyID],
					SortKey:       attributeValue(child, "SortKey"),
					SourceOrdinal: i,
				})
			}
			facts.LegacyKeys = append(facts.LegacyKeys, key)

		case "Relationship":
			facts.RelationshipKeys = append(facts.RelationshipKeys, RelationshipKeyFact{
				Locator:         n.Locator,
				RelationshipID:  attributeValue(n, "ID"),
				Table1LegacyID:  attributeValue(n, "Table1ID"),
				Key1LegacyID:    attributeValue(n, "TableKey1ID"),
				Table2LegacyID:  attributeValue(n, "Table2ID"),
				Key2LegacyID:    attributeValue(n, "TableKey2ID"),
				Nature:          SourceFactNature,
				AuthorityLimits: cloneStrings(relationshipKeyAuthorityLimits),
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
			if doc.Format == FormatXeaf {
				facts.RuntimeRefers = append(facts.RuntimeRefers, RuntimeReferFact{
					Locator:             n.Locator,
					ReferLegacyID:       attributeValue(n, "ID"),
					SourceTableLegacyID: tableLegacyID,
					ToTableLegacyID:     attributeValue(n, "ToTable"),
					TableAlias:          attributeValue(n, "TableAlias"),
					ToKeyFields:         splitLegacyFieldList(attributeValue(n, "ToKeyFields")),
					WithKeyFields:       splitLegacyFieldList(attributeValue(n, "WithKeyFields")),
					Nature:              SourceFactNature,
					AuthorityLimits:     cloneStrings(runtimeReferAuthorityLimits),
				})
			}
		}

		if raw := attributeValue(n, "KeyFields"); raw != "" {
			fields := splitLegacyFieldList(raw)
			facts.DeclaredKeyFields = append(facts.DeclaredKeyFields, DeclaredKeyFieldsFact{
				Locator:         n.Locator,
				SourceName:      localName(n.Name),
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

func indexTableFieldTypes(root Node) map[string]map[string]string {
	out := map[string]map[string]string{}
	var walk func(Node)
	walk = func(n Node) {
		if localName(n.Name) == "Table" {
			tableID := attributeValue(n, "ID")
			if _, ok := out[tableID]; !ok {
				out[tableID] = map[string]string{}
			}
			for _, child := range n.Children {
				childName := localName(child.Name)
				if childName != "TableField" && childName != "Field" {
					continue
				}
				out[tableID][attributeValue(child, "ID")] = attributeValue(child, "AttributeType")
			}
		}
		for _, child := range n.Children {
			walk(child)
		}
	}
	walk(root)
	return out
}

func attributeValue(n Node, name string) string {
	for _, attribute := range n.Attributes {
		if localName(attribute.Name) == name {
			return attribute.Value
		}
	}
	return ""
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
