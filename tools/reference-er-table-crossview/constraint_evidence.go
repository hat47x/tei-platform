package ercrossview

// Physical declaration checks are read-only evidence qualification. They
// inspect this same World snapshot, not provider catalogs or live constraints.
// An identifier-only resemblance is never upgraded to database enforcement.

// constraintHasMappedColumns verifies all referenced semantic Attributes
// resolve to distinct, explicit ColumnMappings in the declared table.
func constraintHasMappedColumns(table TableView, attributes []string) bool {
	if len(attributes) == 0 {
		return false
	}
	mapped := make(map[string]bool, len(table.Columns))
	for _, col := range table.Columns {
		mapped[col.Attribute] = true
	}
	seen := make(map[string]bool, len(attributes))
	for _, attr := range attributes {
		if attr == "" || seen[attr] || !mapped[attr] {
			return false
		}
		seen[attr] = true
	}
	return true
}

// physicalConstraintFindings makes unresolved mapping references explicit.
// It does NOT claim the provider has applied any of the declared constraints.
func physicalConstraintFindings(tables []TableView) []Unresolved {
	byMapping := make(map[string]TableView, len(tables))
	for _, t := range tables {
		byMapping[t.MappingID] = t
	}
	out := []Unresolved{}
	for _, t := range tables {
		mapped := make(map[string]bool, len(t.Columns))
		for _, col := range t.Columns {
			mapped[col.Attribute] = true
		}
		for _, unique := range t.UniqueKeys {
			prefix := t.MappingID + ":" + unique.Name
			if len(unique.Attributes) == 0 {
				out = append(out, Unresolved{Code: "UNIQUE_ATTRIBUTES_EMPTY", Subject: prefix})
			}
			seen := make(map[string]bool, len(unique.Attributes))
			for _, attr := range unique.Attributes {
				if seen[attr] {
					out = append(out, Unresolved{Code: "UNIQUE_ATTRIBUTE_REPEATED", Subject: prefix + ":" + attr})
				}
				seen[attr] = true
				if !mapped[attr] {
					out = append(out, Unresolved{Code: "UNIQUE_ATTRIBUTE_HAS_NO_COLUMN", Subject: prefix + ":" + attr})
				}
			}
		}
		for _, fk := range t.ForeignKeys {
			prefix := t.MappingID + ":" + fk.Name
			if len(fk.Attributes) == 0 || len(fk.Attributes) != len(fk.ReferencedAttributes) {
				out = append(out, Unresolved{Code: "FK_ATTRIBUTE_COUNT_MISMATCH", Subject: prefix})
			}
			localSeen := make(map[string]bool, len(fk.Attributes))
			for _, attr := range fk.Attributes {
				if localSeen[attr] {
					out = append(out, Unresolved{Code: "FK_ATTRIBUTE_REPEATED", Subject: prefix + ":" + attr})
				}
				localSeen[attr] = true
				if !mapped[attr] {
					out = append(out, Unresolved{Code: "FK_ATTRIBUTE_HAS_NO_COLUMN", Subject: prefix + ":" + attr})
				}
			}
			target, exists := byMapping[fk.ReferencedMapping]
			if !exists {
				out = append(out, Unresolved{Code: "FK_REFERENCED_MAPPING_NOT_PRESENT", Subject: prefix + "->" + fk.ReferencedMapping})
				continue
			}
			targetMapped := make(map[string]bool, len(target.Columns))
			for _, col := range target.Columns {
				targetMapped[col.Attribute] = true
			}
			targetSeen := make(map[string]bool, len(fk.ReferencedAttributes))
			for _, attr := range fk.ReferencedAttributes {
				if targetSeen[attr] {
					out = append(out, Unresolved{Code: "FK_REFERENCED_ATTRIBUTE_REPEATED", Subject: prefix + ":" + attr})
				}
				targetSeen[attr] = true
				if !targetMapped[attr] {
					out = append(out, Unresolved{Code: "FK_REFERENCED_ATTRIBUTE_HAS_NO_COLUMN", Subject: prefix + ":" + attr})
				}
			}
		}
	}
	return out
}
