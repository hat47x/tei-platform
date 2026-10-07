package xeadsource

import "sort"

const RelationalDiagnosticNature = "unresolved"

type RelationalDiagnostic struct {
	Code                  string `json:"code"`
	Locator               string `json:"locator"`
	Nature                string `json:"nature"`
	Side                  string `json:"side,omitempty"`
	RelationshipID        string `json:"relationshipId,omitempty"`
	TableLegacyID         string `json:"tableLegacyId,omitempty"`
	KeyLegacyID           string `json:"keyLegacyId,omitempty"`
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
	tableIndex := indexTableFieldIdentity(doc.Root)
	tableFields := tableIndex.Fields
	keys := map[string]map[string][]LegacyKeyFact{}
	for _, key := range facts.LegacyKeys {
		if _, ok := keys[key.TableLegacyID]; !ok {
			keys[key.TableLegacyID] = map[string][]LegacyKeyFact{}
		}
		keys[key.TableLegacyID][key.KeyLegacyID] = append(keys[key.TableLegacyID][key.KeyLegacyID], key)
	}

	out := make([]RelationalDiagnostic, 0)
	for _, locator := range tableIndex.TableLocators[""] {
		out = append(out, RelationalDiagnostic{
			Code:    "table-id-missing",
			Locator: locator,
			Nature:  RelationalDiagnosticNature,
		})
	}
	for tableID, count := range tableIndex.TableCounts {
		if tableID == "" || count <= 1 {
			continue
		}
		out = append(out, RelationalDiagnostic{
			Code:          "table-id-duplicate",
			Locator:       tableIndex.TableLocators[tableID][0],
			Nature:        RelationalDiagnosticNature,
			TableLegacyID: tableID,
			Count:         count,
		})
	}
	for tableID, fieldCounts := range tableIndex.FieldCounts {
		if tableID == "" || tableIndex.TableCounts[tableID] != 1 {
			continue
		}
		if count := fieldCounts[""]; count > 0 {
			for _, locator := range tableIndex.FieldLocators[tableID][""] {
				out = append(out, RelationalDiagnostic{
					Code:          "field-id-missing",
					Locator:       locator,
					Nature:        RelationalDiagnosticNature,
					TableLegacyID: tableID,
				})
			}
		}
		for fieldID, count := range fieldCounts {
			if fieldID == "" || count <= 1 {
				continue
			}
			out = append(out, RelationalDiagnostic{
				Code:          "field-id-duplicate",
				Locator:       tableIndex.FieldLocators[tableID][fieldID][0],
				Nature:        RelationalDiagnosticNature,
				TableLegacyID: tableID,
				FieldLegacyID: fieldID,
				Count:         count,
			})
		}
	}
	for tableID, tableKeys := range keys {
		for keyID, matches := range tableKeys {
			if tableID == "" || tableIndex.TableCounts[tableID] != 1 || keyID == "" || len(matches) <= 1 {
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
		tableCount := tableIndex.TableCounts[key.TableLegacyID]
		fields := tableFields[key.TableLegacyID]
		if key.TableLegacyID == "" {
			out = append(out, RelationalDiagnostic{
				Code:        "key-table-id-missing",
				Locator:     key.Locator,
				Nature:      RelationalDiagnosticNature,
				KeyLegacyID: key.KeyLegacyID,
			})
		} else if tableCount == 0 {
			out = append(out, RelationalDiagnostic{
				Code:          "key-table-not-found",
				Locator:       key.Locator,
				Nature:        RelationalDiagnosticNature,
				TableLegacyID: key.TableLegacyID,
				KeyLegacyID:   key.KeyLegacyID,
			})
		} else if tableCount > 1 {
			out = append(out, RelationalDiagnostic{
				Code:          "key-table-ambiguous",
				Locator:       key.Locator,
				Nature:        RelationalDiagnosticNature,
				TableLegacyID: key.TableLegacyID,
				KeyLegacyID:   key.KeyLegacyID,
				Count:         tableCount,
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
			if tableCount != 1 {
				continue
			}
			fieldCount := tableIndex.FieldCounts[key.TableLegacyID][member.FieldLegacyID]
			if fieldCount == 0 {
				out = append(out, RelationalDiagnostic{
					Code:          "key-field-not-found",
					Locator:       member.Locator,
					Nature:        RelationalDiagnosticNature,
					TableLegacyID: key.TableLegacyID,
					KeyLegacyID:   key.KeyLegacyID,
					FieldLegacyID: member.FieldLegacyID,
				})
			} else if fieldCount > 1 {
				out = append(out, RelationalDiagnostic{
					Code:          "key-field-ambiguous",
					Locator:       member.Locator,
					Nature:        RelationalDiagnosticNature,
					TableLegacyID: key.TableLegacyID,
					KeyLegacyID:   key.KeyLegacyID,
					FieldLegacyID: member.FieldLegacyID,
					Count:         fieldCount,
				})
			} else if _, ok := fields[member.FieldLegacyID]; !ok {
				out = append(out, RelationalDiagnostic{
					Code:          "key-field-metadata-unresolved",
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
		left, leftOK := diagnoseRelationshipSide(&out, relation, "table1", relation.Table1LegacyID, relation.Key1LegacyID, tableIndex, keys)
		right, rightOK := diagnoseRelationshipSide(&out, relation, "table2", relation.Table2LegacyID, relation.Key2LegacyID, tableIndex, keys)
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
		tableCount := tableIndex.TableCounts[key.TableLegacyID]
		if key.TableLegacyID == "" {
			out = append(out, RelationalDiagnostic{
				Code:    "runtime-key-table-id-missing",
				Locator: key.Locator,
				Nature:  RelationalDiagnosticNature,
			})
			continue
		}
		if tableCount == 0 {
			out = append(out, RelationalDiagnostic{
				Code:          "runtime-key-table-not-found",
				Locator:       key.Locator,
				Nature:        RelationalDiagnosticNature,
				TableLegacyID: key.TableLegacyID,
				KeyLegacyID:   key.KeyLegacyID,
			})
			continue
		}
		if tableCount > 1 {
			out = append(out, RelationalDiagnostic{
				Code:          "runtime-key-table-ambiguous",
				Locator:       key.Locator,
				Nature:        RelationalDiagnosticNature,
				TableLegacyID: key.TableLegacyID,
				KeyLegacyID:   key.KeyLegacyID,
				Count:         tableCount,
			})
			continue
		}
		for _, fieldID := range key.Fields {
			fieldCount := tableIndex.FieldCounts[key.TableLegacyID][fieldID]
			if fieldCount == 0 {
				out = append(out, RelationalDiagnostic{
					Code:          "runtime-key-field-not-found",
					Locator:       key.Locator,
					Nature:        RelationalDiagnosticNature,
					TableLegacyID: key.TableLegacyID,
					KeyLegacyID:   key.KeyLegacyID,
					FieldLegacyID: fieldID,
				})
			} else if fieldCount > 1 {
				out = append(out, RelationalDiagnostic{
					Code:          "runtime-key-field-ambiguous",
					Locator:       key.Locator,
					Nature:        RelationalDiagnosticNature,
					TableLegacyID: key.TableLegacyID,
					KeyLegacyID:   key.KeyLegacyID,
					FieldLegacyID: fieldID,
					Count:         fieldCount,
				})
			}
		}
	}

	for _, refer := range facts.RuntimeRefers {
		sourceCount := tableIndex.TableCounts[refer.SourceTableLegacyID]
		targetCount := tableIndex.TableCounts[refer.ToTableLegacyID]
		sourceOK := sourceCount == 1
		targetOK := targetCount == 1
		if refer.SourceTableLegacyID == "" {
			out = append(out, RelationalDiagnostic{
				Code:    "runtime-refer-source-table-id-missing",
				Locator: refer.Locator,
				Nature:  RelationalDiagnosticNature,
			})
		} else if sourceCount == 0 {
			out = append(out, RelationalDiagnostic{
				Code:          "runtime-refer-source-table-not-found",
				Locator:       refer.Locator,
				Nature:        RelationalDiagnosticNature,
				TableLegacyID: refer.SourceTableLegacyID,
			})
		} else if sourceCount > 1 {
			out = append(out, RelationalDiagnostic{
				Code:          "runtime-refer-source-table-ambiguous",
				Locator:       refer.Locator,
				Nature:        RelationalDiagnosticNature,
				TableLegacyID: refer.SourceTableLegacyID,
				Count:         sourceCount,
			})
		}
		if refer.ToTableLegacyID == "" {
			out = append(out, RelationalDiagnostic{
				Code:    "runtime-refer-target-table-id-missing",
				Locator: refer.Locator,
				Nature:  RelationalDiagnosticNature,
			})
		} else if targetCount == 0 {
			out = append(out, RelationalDiagnostic{
				Code:          "runtime-refer-target-table-not-found",
				Locator:       refer.Locator,
				Nature:        RelationalDiagnosticNature,
				TableLegacyID: refer.ToTableLegacyID,
			})
		} else if targetCount > 1 {
			out = append(out, RelationalDiagnostic{
				Code:          "runtime-refer-target-table-ambiguous",
				Locator:       refer.Locator,
				Nature:        RelationalDiagnosticNature,
				TableLegacyID: refer.ToTableLegacyID,
				Count:         targetCount,
			})
		}
		if sourceOK {
			for _, fieldID := range refer.WithKeyFields {
				fieldCount := tableIndex.FieldCounts[refer.SourceTableLegacyID][fieldID]
				if fieldCount == 0 {
					out = append(out, RelationalDiagnostic{
						Code:          "runtime-refer-source-field-not-found",
						Locator:       refer.Locator,
						Nature:        RelationalDiagnosticNature,
						TableLegacyID: refer.SourceTableLegacyID,
						FieldLegacyID: fieldID,
					})
				} else if fieldCount > 1 {
					out = append(out, RelationalDiagnostic{
						Code:          "runtime-refer-source-field-ambiguous",
						Locator:       refer.Locator,
						Nature:        RelationalDiagnosticNature,
						TableLegacyID: refer.SourceTableLegacyID,
						FieldLegacyID: fieldID,
						Count:         fieldCount,
					})
				}
			}
		}
		if targetOK {
			for _, fieldID := range refer.ToKeyFields {
				fieldCount := tableIndex.FieldCounts[refer.ToTableLegacyID][fieldID]
				if fieldCount == 0 {
					out = append(out, RelationalDiagnostic{
						Code:          "runtime-refer-target-field-not-found",
						Locator:       refer.Locator,
						Nature:        RelationalDiagnosticNature,
						TableLegacyID: refer.ToTableLegacyID,
						FieldLegacyID: fieldID,
					})
				} else if fieldCount > 1 {
					out = append(out, RelationalDiagnostic{
						Code:          "runtime-refer-target-field-ambiguous",
						Locator:       refer.Locator,
						Nature:        RelationalDiagnosticNature,
						TableLegacyID: refer.ToTableLegacyID,
						FieldLegacyID: fieldID,
						Count:         fieldCount,
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
	tableIndex tableFieldIdentityIndex,
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
	tableCount := tableIndex.TableCounts[tableID]
	if tableCount == 0 {
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
	if tableCount > 1 {
		*out = append(*out, RelationalDiagnostic{
			Code:           "relationship-table-ambiguous",
			Locator:        relation.Locator,
			Nature:         RelationalDiagnosticNature,
			Side:           side,
			RelationshipID: relation.RelationshipID,
			TableLegacyID:  tableID,
			KeyLegacyID:    keyID,
			Count:          tableCount,
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
	key := matches[0]
	for _, member := range key.Members {
		if member.FieldLegacyID == "" {
			*out = append(*out, RelationalDiagnostic{
				Code:           "relationship-key-field-id-missing",
				Locator:        relation.Locator,
				Nature:         RelationalDiagnosticNature,
				Side:           side,
				RelationshipID: relation.RelationshipID,
				TableLegacyID:  tableID,
				KeyLegacyID:    keyID,
			})
			return LegacyKeyFact{}, false
		}
		fieldCount := tableIndex.FieldCounts[tableID][member.FieldLegacyID]
		if fieldCount == 0 {
			*out = append(*out, RelationalDiagnostic{
				Code:           "relationship-key-field-not-found",
				Locator:        relation.Locator,
				Nature:         RelationalDiagnosticNature,
				Side:           side,
				RelationshipID: relation.RelationshipID,
				TableLegacyID:  tableID,
				KeyLegacyID:    keyID,
				FieldLegacyID:  member.FieldLegacyID,
			})
			return LegacyKeyFact{}, false
		}
		if fieldCount > 1 {
			*out = append(*out, RelationalDiagnostic{
				Code:           "relationship-key-field-ambiguous",
				Locator:        relation.Locator,
				Nature:         RelationalDiagnosticNature,
				Side:           side,
				RelationshipID: relation.RelationshipID,
				TableLegacyID:  tableID,
				KeyLegacyID:    keyID,
				FieldLegacyID:  member.FieldLegacyID,
				Count:          fieldCount,
			})
			return LegacyKeyFact{}, false
		}
	}
	return key, true
}
