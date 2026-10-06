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
	FieldLegacyID         string `json:"fieldLegacyId,omitempty"`
	LeftFieldLegacyID     string `json:"leftFieldLegacyId,omitempty"`
	RightFieldLegacyID    string `json:"rightFieldLegacyId,omitempty"`
	LeftDataTypeLegacyID  string `json:"leftDataTypeLegacyId,omitempty"`
	RightDataTypeLegacyID string `json:"rightDataTypeLegacyId,omitempty"`
	Count                 int    `json:"count,omitempty"`
	LeftCount             int    `json:"leftCount,omitempty"`
	RightCount            int    `json:"rightCount,omitempty"`
}

func DiagnoseRelationalSourceFacts(doc Document, facts RelationalSourceFacts) []RelationalDiagnostic {
	tableFields := indexTableFieldMetadata(doc.Root)
	keys := map[string]map[string][]LegacyKeyFact{}
	for _, key := range facts.LegacyKeys {
		if _, ok := keys[key.TableLegacyID]; !ok {
			keys[key.TableLegacyID] = map[string][]LegacyKeyFact{}
		}
		keys[key.TableLegacyID][key.KeyLegacyID] = append(keys[key.TableLegacyID][key.KeyLegacyID], key)
	}

	out := make([]RelationalDiagnostic, 0)
	for tableID, tableKeys := range keys {
		for keyID, matches := range tableKeys {
			if tableID == "" || keyID == "" || len(matches) <= 1 {
				continue
			}
			out = append(out, RelationalDiagnostic{
				Code:          "key-id-duplicate",
				Locator:       matches[0].Locator,
				Nature:        RelationalDiagnosticNature,
				TableLegacyID: tableID,
				KeyLegacyID:   keyID,
				Count:         len(matches),
			})
		}
	}
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
		if !leftOK || !rightOK {
			continue
		}
		if len(left.Members) != len(right.Members) {
			out = append(out, RelationalDiagnostic{
				Code:           "relationship-key-arity-mismatch",
				Locator:        relation.Locator,
				Nature:         RelationalDiagnosticNature,
				RelationshipID: relation.RelationshipID,
				LeftCount:      len(left.Members),
				RightCount:     len(right.Members),
			})
			continue
		}

		leftMembers, leftOrderOK := orderedLegacyKeyMembers(left)
		rightMembers, rightOrderOK := orderedLegacyKeyMembers(right)
		if !leftOrderOK {
			out = append(out, RelationalDiagnostic{
				Code:           "relationship-key-order-unresolved",
				Locator:        relation.Locator,
				Nature:         RelationalDiagnosticNature,
				Side:           "table1",
				RelationshipID: relation.RelationshipID,
				TableLegacyID:  relation.Table1LegacyID,
				KeyLegacyID:    relation.Key1LegacyID,
			})
		}
		if !rightOrderOK {
			out = append(out, RelationalDiagnostic{
				Code:           "relationship-key-order-unresolved",
				Locator:        relation.Locator,
				Nature:         RelationalDiagnosticNature,
				Side:           "table2",
				RelationshipID: relation.RelationshipID,
				TableLegacyID:  relation.Table2LegacyID,
				KeyLegacyID:    relation.Key2LegacyID,
			})
		}
		if !leftOrderOK || !rightOrderOK {
			continue
		}

		for i := range leftMembers {
			leftType := leftMembers[i].DataTypeLegacyID
			rightType := rightMembers[i].DataTypeLegacyID
			if leftType == "" || rightType == "" || leftType == rightType {
				continue
			}
			out = append(out, RelationalDiagnostic{
				Code:                  "relationship-key-data-type-mismatch",
				Locator:               relation.Locator,
				Nature:                RelationalDiagnosticNature,
				RelationshipID:        relation.RelationshipID,
				LeftFieldLegacyID:     leftMembers[i].FieldLegacyID,
				RightFieldLegacyID:    rightMembers[i].FieldLegacyID,
				LeftDataTypeLegacyID:  leftType,
				RightDataTypeLegacyID: rightType,
			})
		}
	}

	for _, key := range facts.RuntimeKeys {
		fields, tableExists := tableFields[key.TableLegacyID]
		if key.TableLegacyID == "" {
			out = append(out, RelationalDiagnostic{
				Code:    "runtime-key-table-id-missing",
				Locator: key.Locator,
				Nature:  RelationalDiagnosticNature,
			})
			continue
		}
		if !tableExists {
			out = append(out, RelationalDiagnostic{
				Code:          "runtime-key-table-not-found",
				Locator:       key.Locator,
				Nature:        RelationalDiagnosticNature,
				TableLegacyID: key.TableLegacyID,
				KeyLegacyID:   key.KeyLegacyID,
			})
			continue
		}
		for _, fieldID := range key.Fields {
			if _, ok := fields[fieldID]; !ok {
				out = append(out, RelationalDiagnostic{
					Code:          "runtime-key-field-not-found",
					Locator:       key.Locator,
					Nature:        RelationalDiagnosticNature,
					TableLegacyID: key.TableLegacyID,
					KeyLegacyID:   key.KeyLegacyID,
					FieldLegacyID: fieldID,
				})
			}
		}
	}

	for _, refer := range facts.RuntimeRefers {
		sourceFields, sourceOK := tableFields[refer.SourceTableLegacyID]
		targetFields, targetOK := tableFields[refer.ToTableLegacyID]
		if refer.SourceTableLegacyID == "" {
			out = append(out, RelationalDiagnostic{
				Code:    "runtime-refer-source-table-id-missing",
				Locator: refer.Locator,
				Nature:  RelationalDiagnosticNature,
			})
		} else if !sourceOK {
			out = append(out, RelationalDiagnostic{
				Code:          "runtime-refer-source-table-not-found",
				Locator:       refer.Locator,
				Nature:        RelationalDiagnosticNature,
				TableLegacyID: refer.SourceTableLegacyID,
			})
		}
		if refer.ToTableLegacyID == "" {
			out = append(out, RelationalDiagnostic{
				Code:    "runtime-refer-target-table-id-missing",
				Locator: refer.Locator,
				Nature:  RelationalDiagnosticNature,
			})
		} else if !targetOK {
			out = append(out, RelationalDiagnostic{
				Code:          "runtime-refer-target-table-not-found",
				Locator:       refer.Locator,
				Nature:        RelationalDiagnosticNature,
				TableLegacyID: refer.ToTableLegacyID,
			})
		}
		if sourceOK {
			for _, fieldID := range refer.WithKeyFields {
				if _, ok := sourceFields[fieldID]; !ok {
					out = append(out, RelationalDiagnostic{
						Code:          "runtime-refer-source-field-not-found",
						Locator:       refer.Locator,
						Nature:        RelationalDiagnosticNature,
						TableLegacyID: refer.SourceTableLegacyID,
						FieldLegacyID: fieldID,
					})
				}
			}
		}
		if targetOK {
			for _, fieldID := range refer.ToKeyFields {
				if _, ok := targetFields[fieldID]; !ok {
					out = append(out, RelationalDiagnostic{
						Code:          "runtime-refer-target-field-not-found",
						Locator:       refer.Locator,
						Nature:        RelationalDiagnosticNature,
						TableLegacyID: refer.ToTableLegacyID,
						FieldLegacyID: fieldID,
					})
				}
			}
		}
		if len(refer.ToKeyFields) > 0 &&
			len(refer.WithKeyFields) > 0 &&
			len(refer.ToKeyFields) != len(refer.WithKeyFields) {
			out = append(out, RelationalDiagnostic{
				Code:       "runtime-refer-key-arity-mismatch",
				Locator:    refer.Locator,
				Nature:     RelationalDiagnosticNature,
				LeftCount:  len(refer.WithKeyFields),
				RightCount: len(refer.ToKeyFields),
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

func orderedLegacyKeyMembers(key LegacyKeyFact) ([]LegacyKeyMemberFact, bool) {
	members := append([]LegacyKeyMemberFact(nil), key.Members...)
	if len(members) <= 1 {
		return members, true
	}
	seen := map[string]struct{}{}
	for _, member := range members {
		if member.SortKey == "" {
			return nil, false
		}
		if _, duplicate := seen[member.SortKey]; duplicate {
			return nil, false
		}
		seen[member.SortKey] = struct{}{}
	}
	sort.Slice(members, func(i, j int) bool {
		return members[i].SortKey < members[j].SortKey
	})
	return members, true
}

func diagnoseRelationshipSide(
	out *[]RelationalDiagnostic,
	relation RelationshipKeyFact,
	side string,
	tableID string,
	keyID string,
	tableFields map[string]map[string]tableFieldMetadata,
	keys map[string]map[string][]LegacyKeyFact,
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
	matches := keys[tableID][keyID]
	if len(matches) == 0 {
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
	if len(matches) > 1 {
		*out = append(*out, RelationalDiagnostic{
			Code:           "relationship-key-ambiguous",
			Locator:        relation.Locator,
			Nature:         RelationalDiagnosticNature,
			Side:           side,
			RelationshipID: relation.RelationshipID,
			TableLegacyID:  tableID,
			KeyLegacyID:    keyID,
			Count:          len(matches),
		})
		return LegacyKeyFact{}, false
	}
	return matches[0], true
}
