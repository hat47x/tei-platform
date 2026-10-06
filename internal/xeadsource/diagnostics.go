package xeadsource

import "sort"

const RelationalDiagnosticNature = "unresolved"

type RelationalDiagnostic struct {
	Code           string `json:"code"`
	Locator        string `json:"locator"`
	Nature         string `json:"nature"`
	Side           string `json:"side,omitempty"`
	RelationshipID string `json:"relationshipId,omitempty"`
	TableLegacyID  string `json:"tableLegacyId,omitempty"`
	KeyLegacyID    string `json:"keyLegacyId,omitempty"`
	FieldLegacyID  string `json:"fieldLegacyId,omitempty"`
	LeftCount      int    `json:"leftCount,omitempty"`
	RightCount     int    `json:"rightCount,omitempty"`
}

func DiagnoseRelationalSourceFacts(doc Document, facts RelationalSourceFacts) []RelationalDiagnostic {
	tableFields := indexTableFieldTypes(doc.Root)
	keys := map[string]map[string]LegacyKeyFact{}
	for _, key := range facts.LegacyKeys {
		if _, ok := keys[key.TableLegacyID]; !ok {
			keys[key.TableLegacyID] = map[string]LegacyKeyFact{}
		}
		keys[key.TableLegacyID][key.KeyLegacyID] = key
	}

	out := make([]RelationalDiagnostic, 0)
	for _, key := range facts.LegacyKeys {
		fields, tableExists := tableFields[key.TableLegacyID]
		if key.TableLegacyID == "" {
			out = append(out, RelationalDiagnostic{
				Code:        "key-table-id-missing",
				Locator:     key.Locator,
				Nature:      RelationalDiagnosticNature,
				KeyLegacyID: key.KeyLegacyID,
			})
		} else if !tableExists {
			out = append(out, RelationalDiagnostic{
				Code:          "key-table-not-found",
				Locator:       key.Locator,
				Nature:        RelationalDiagnosticNature,
				TableLegacyID: key.TableLegacyID,
				KeyLegacyID:   key.KeyLegacyID,
			})
		}
		for _, member := range key.Members {
			if member.FieldLegacyID == "" {
				out = append(out, RelationalDiagnostic{
					Code:          "key-field-id-missing",
					Locator:       member.Locator,
					Nature:        RelationalDiagnosticNature,
					TableLegacyID: key.TableLegacyID,
					KeyLegacyID:   key.KeyLegacyID,
				})
				continue
			}
			if _, ok := fields[member.FieldLegacyID]; !ok {
				out = append(out, RelationalDiagnostic{
					Code:          "key-field-not-found",
					Locator:       member.Locator,
					Nature:        RelationalDiagnosticNature,
					TableLegacyID: key.TableLegacyID,
					KeyLegacyID:   key.KeyLegacyID,
					FieldLegacyID: member.FieldLegacyID,
				})
			}
		}
	}

	for _, relation := range facts.RelationshipKeys {
		left, leftOK := diagnoseRelationshipSide(&out, relation, "table1", relation.Table1LegacyID, relation.Key1LegacyID, tableFields, keys)
		right, rightOK := diagnoseRelationshipSide(&out, relation, "table2", relation.Table2LegacyID, relation.Key2LegacyID, tableFields, keys)
		if leftOK && rightOK && len(left.Members) != len(right.Members) {
			out = append(out, RelationalDiagnostic{
				Code:           "relationship-key-arity-mismatch",
				Locator:        relation.Locator,
				Nature:         RelationalDiagnosticNature,
				RelationshipID: relation.RelationshipID,
				LeftCount:      len(left.Members),
				RightCount:     len(right.Members),
			})
		}
	}

	sort.Slice(out, func(i, j int) bool {
		if out[i].Locator != out[j].Locator {
			return out[i].Locator < out[j].Locator
		}
		if out[i].Code != out[j].Code {
			return out[i].Code < out[j].Code
		}
		return out[i].Side < out[j].Side
	})
	return out
}

func diagnoseRelationshipSide(
	out *[]RelationalDiagnostic,
	relation RelationshipKeyFact,
	side string,
	tableID string,
	keyID string,
	tableFields map[string]map[string]string,
	keys map[string]map[string]LegacyKeyFact,
) (LegacyKeyFact, bool) {
	if tableID == "" {
		*out = append(*out, RelationalDiagnostic{
			Code:           "relationship-table-id-missing",
			Locator:        relation.Locator,
			Nature:         RelationalDiagnosticNature,
			Side:           side,
			RelationshipID: relation.RelationshipID,
			KeyLegacyID:    keyID,
		})
		return LegacyKeyFact{}, false
	}
	if _, ok := tableFields[tableID]; !ok {
		*out = append(*out, RelationalDiagnostic{
			Code:           "relationship-table-not-found",
			Locator:        relation.Locator,
			Nature:         RelationalDiagnosticNature,
			Side:           side,
			RelationshipID: relation.RelationshipID,
			TableLegacyID:  tableID,
			KeyLegacyID:    keyID,
		})
		return LegacyKeyFact{}, false
	}
	if keyID == "" {
		*out = append(*out, RelationalDiagnostic{
			Code:           "relationship-key-id-missing",
			Locator:        relation.Locator,
			Nature:         RelationalDiagnosticNature,
			Side:           side,
			RelationshipID: relation.RelationshipID,
			TableLegacyID:  tableID,
		})
		return LegacyKeyFact{}, false
	}
	key, ok := keys[tableID][keyID]
	if !ok {
		*out = append(*out, RelationalDiagnostic{
			Code:           "relationship-key-not-found",
			Locator:        relation.Locator,
			Nature:         RelationalDiagnosticNature,
			Side:           side,
			RelationshipID: relation.RelationshipID,
			TableLegacyID:  tableID,
			KeyLegacyID:    keyID,
		})
		return LegacyKeyFact{}, false
	}
	return key, true
}
