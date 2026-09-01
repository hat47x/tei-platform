package persistence

import (
	"fmt"
	"strings"

	"tei/internal/core"
	"tei/internal/database/physical"
)

// RealizeDesired projects provider-neutral Persistence Mapping intent into a
// provider-specific Desired Physical Schema using an offline provider policy.
// The mapping remains canonical and the resulting snapshot is deterministic.
func RealizeDesired(mappings []Mapping, databaseID string, policy PhysicalRealizationPolicy) (physical.SchemaSnapshot, error) {
	if policy == nil {
		return physical.SchemaSnapshot{}, fmt.Errorf("physical realization policy is nil")
	}
	out := physical.SchemaSnapshot{Database: mustDatabaseID(databaseID)}
	mappingByID := make(map[core.CanonicalID]Mapping, len(mappings))
	for _, m := range mappings {
		mappingByID[m.ID] = m
	}

	for _, m := range mappings {
		if string(m.Database) != databaseID {
			continue
		}
		schema, table := splitQualified(m.Table)
		t := physical.Table{Schema: schema, Name: table, Kind: physical.TableKindTable}
		columnByAttribute := columnNamesByAttribute(m)
		for i, c := range m.Columns {
			pt, err := policy.PhysicalType(c.Storage)
			if err != nil {
				return physical.SchemaSnapshot{}, fmt.Errorf("mapping %s attribute %s: %w", m.ID, c.Attribute, err)
			}
			t.Columns = append(t.Columns, physical.Column{Name: c.Column, PhysicalType: pt, Nullable: c.Nullability == Optional, Ordinal: i + 1})
		}
		for _, attr := range m.PrimaryKey {
			name, ok := columnByAttribute[attr]
			if !ok {
				return physical.SchemaSnapshot{}, fmt.Errorf("mapping %s primary key attribute %s has no column mapping", m.ID, attr)
			}
			t.PrimaryKey = append(t.PrimaryKey, name)
		}
		for _, constraint := range m.UniqueConstraints {
			uc, err := realizeUniqueConstraint(m, constraint, columnByAttribute)
			if err != nil {
				return physical.SchemaSnapshot{}, err
			}
			t.UniqueConstraints = append(t.UniqueConstraints, uc)
		}
		seenIndexNames := make(map[string]struct{}, len(m.Indexes))
		for _, idx := range m.Indexes {
			name := strings.TrimSpace(idx.Name)
			if _, exists := seenIndexNames[name]; exists {
				return physical.SchemaSnapshot{}, fmt.Errorf("mapping %s declares duplicate index name %q", m.ID, idx.Name)
			}
			seenIndexNames[name] = struct{}{}
			pi, err := realizeIndex(m, idx, columnByAttribute, policy)
			if err != nil {
				return physical.SchemaSnapshot{}, err
			}
			t.Indexes = append(t.Indexes, pi)
		}
		for _, fk := range m.ForeignKeys {
			pfk, err := realizeForeignKey(m, fk, mappingByID, columnByAttribute)
			if err != nil {
				return physical.SchemaSnapshot{}, err
			}
			t.ForeignKeys = append(t.ForeignKeys, pfk)
		}
		out.Tables = append(out.Tables, t)
	}
	return out.Normalize(), nil
}

func realizeUniqueConstraint(source Mapping, constraint UniqueConstraintMapping, columns map[core.CanonicalID]string) (physical.UniqueConstraint, error) {
	if len(constraint.Attributes) == 0 {
		return physical.UniqueConstraint{}, fmt.Errorf("mapping %s unique constraint %s has no attributes", source.ID, constraint.Name)
	}
	out := physical.UniqueConstraint{Name: constraint.Name}
	for _, attr := range constraint.Attributes {
		name, ok := columns[attr]
		if !ok {
			return physical.UniqueConstraint{}, fmt.Errorf("mapping %s unique constraint %s attribute %s has no column mapping", source.ID, constraint.Name, attr)
		}
		out.Columns = append(out.Columns, name)
	}
	return out, nil
}

func realizeIndex(source Mapping, idx IndexMapping, columns map[core.CanonicalID]string, policy PhysicalRealizationPolicy) (physical.Index, error) {
	if strings.TrimSpace(idx.Name) == "" {
		return physical.Index{}, fmt.Errorf("mapping %s index name is empty", source.ID)
	}
	if len(idx.Attributes) > 0 && len(idx.Keys) > 0 {
		return physical.Index{}, fmt.Errorf("mapping %s index %s mixes simple attributes and advanced keys", source.ID, idx.Name)
	}
	if len(idx.Attributes) == 0 && len(idx.Keys) == 0 {
		return physical.Index{}, fmt.Errorf("mapping %s index %s has no keys", source.ID, idx.Name)
	}
	if !idx.UniqueNullTreatment.Valid() {
		return physical.Index{}, fmt.Errorf("mapping %s index %s has unknown unique NULL treatment %q", source.ID, idx.Name, idx.UniqueNullTreatment)
	}
	if !idx.Unique && idx.UniqueNullTreatment != "" {
		return physical.Index{}, fmt.Errorf("mapping %s non-unique index %s cannot declare NULL treatment", source.ID, idx.Name)
	}

	defaults := policy.IndexDefaults()
	method := strings.TrimSpace(idx.Method)
	if method == "" {
		method = defaults.Method
	}
	out := physical.Index{Name: idx.Name, Unique: idx.Unique, Method: method}
	if idx.Unique {
		out.UniqueNullTreatment = defaults.UniqueNullTreatment
		if idx.UniqueNullTreatment != "" {
			out.UniqueNullTreatment = physical.UniqueNullTreatment(idx.UniqueNullTreatment)
		}
	}

	if len(idx.Attributes) > 0 {
		for _, attr := range idx.Attributes {
			name, ok := columns[attr]
			if !ok {
				return physical.Index{}, fmt.Errorf("mapping %s index %s attribute %s has no column mapping", source.ID, idx.Name, attr)
			}
			out.Keys = append(out.Keys, physical.ColumnIndexKey(name))
		}
	} else {
		for position, key := range idx.Keys {
			physicalKey, err := realizeIndexKey(source, idx.Name, position, key, columns, policy, method)
			if err != nil {
				return physical.Index{}, err
			}
			out.Keys = append(out.Keys, physicalKey)
		}
	}

	for _, attr := range idx.IncludedAttributes {
		name, ok := columns[attr]
		if !ok {
			return physical.Index{}, fmt.Errorf("mapping %s index %s INCLUDE attribute %s has no column mapping", source.ID, idx.Name, attr)
		}
		out.IncludedColumns = append(out.IncludedColumns, name)
	}
	if idx.Predicate != nil {
		expression, err := realizeProviderExpression(source, idx.Name, "predicate", *idx.Predicate, policy)
		if err != nil {
			return physical.Index{}, err
		}
		out.Predicate = expression
	}
	return out, nil
}

func realizeIndexKey(source Mapping, indexName string, position int, key IndexKeyMapping, columns map[core.CanonicalID]string, policy PhysicalRealizationPolicy, method string) (physical.IndexKey, error) {
	hasAttribute := key.Attribute != ""
	hasExpression := key.Expression != nil
	if hasAttribute == hasExpression {
		return physical.IndexKey{}, fmt.Errorf("mapping %s index %s key %d must declare exactly one attribute or expression", source.ID, indexName, position+1)
	}
	if !key.Order.Valid() {
		return physical.IndexKey{}, fmt.Errorf("mapping %s index %s key %d has unknown order %q", source.ID, indexName, position+1, key.Order)
	}
	if !key.NullsOrder.Valid() {
		return physical.IndexKey{}, fmt.Errorf("mapping %s index %s key %d has unknown NULL ordering %q", source.ID, indexName, position+1, key.NullsOrder)
	}

	order := physical.IndexOrderDefault
	if key.Order != "" {
		order = physical.IndexOrder(key.Order)
	}
	nullsOrder := physical.IndexNullsOrderDefault
	if key.NullsOrder != "" {
		nullsOrder = physical.IndexNullsOrder(key.NullsOrder)
	}
	out := physical.IndexKey{
		Collation:     strings.TrimSpace(key.Collation),
		OperatorClass: strings.TrimSpace(key.OperatorClass),
		Order:         order,
		NullsOrder:    nullsOrder,
	}
	if hasAttribute {
		column, ok := columns[key.Attribute]
		if !ok {
			return physical.IndexKey{}, fmt.Errorf("mapping %s index %s key %d attribute %s has no column mapping", source.ID, indexName, position+1, key.Attribute)
		}
		out.Column = column
		return policy.NormalizeIndexKey(out, method), nil
	}

	expression, err := realizeProviderExpression(source, indexName, fmt.Sprintf("key %d expression", position+1), *key.Expression, policy)
	if err != nil {
		return physical.IndexKey{}, err
	}
	out.Expression = expression
	return policy.NormalizeIndexKey(out, method), nil
}

func realizeProviderExpression(source Mapping, indexName, field string, expression ProviderExpressionMapping, policy PhysicalRealizationPolicy) (*physical.ProviderExpression, error) {
	if !expression.Provider.Known() {
		return nil, fmt.Errorf("mapping %s index %s %s uses unknown provider %q", source.ID, indexName, field, expression.Provider)
	}
	provider := policy.Provider()
	if expression.Provider != provider {
		return nil, fmt.Errorf("mapping %s index %s %s provider %q does not match desired provider %q", source.ID, indexName, field, expression.Provider, provider)
	}
	dialect := strings.TrimSpace(expression.Dialect)
	if dialect == "" {
		return nil, fmt.Errorf("mapping %s index %s %s dialect is empty", source.ID, indexName, field)
	}
	text := strings.TrimSpace(expression.Source)
	if text == "" {
		return nil, fmt.Errorf("mapping %s index %s %s source is empty", source.ID, indexName, field)
	}
	return &physical.ProviderExpression{
		Provider:       string(expression.Provider),
		Dialect:        dialect,
		Representation: physical.ProviderExpressionAuthored,
		Source:         text,
	}, nil
}

func realizeForeignKey(source Mapping, fk ForeignKeyMapping, mappings map[core.CanonicalID]Mapping, sourceColumns map[core.CanonicalID]string) (physical.ForeignKey, error) {
	if len(fk.Attributes) == 0 {
		return physical.ForeignKey{}, fmt.Errorf("mapping %s foreign key %s has no source attributes", source.ID, fk.Name)
	}
	if len(fk.Attributes) != len(fk.ReferencedAttributes) {
		return physical.ForeignKey{}, fmt.Errorf("mapping %s foreign key %s source/target attribute count mismatch: %d != %d", source.ID, fk.Name, len(fk.Attributes), len(fk.ReferencedAttributes))
	}
	target, ok := mappings[fk.ReferencedMapping]
	if !ok {
		return physical.ForeignKey{}, fmt.Errorf("mapping %s foreign key %s references unknown mapping %s", source.ID, fk.Name, fk.ReferencedMapping)
	}
	if target.Database != source.Database {
		return physical.ForeignKey{}, fmt.Errorf("mapping %s foreign key %s crosses database boundary from %s to %s", source.ID, fk.Name, source.Database, target.Database)
	}
	targetColumns := columnNamesByAttribute(target)
	refSchema, refTable := splitQualified(target.Table)
	out := physical.ForeignKey{Name: fk.Name, ReferencedSchema: refSchema, ReferencedTable: refTable}
	for i, attr := range fk.Attributes {
		name, ok := sourceColumns[attr]
		if !ok {
			return physical.ForeignKey{}, fmt.Errorf("mapping %s foreign key %s source attribute %s has no column mapping", source.ID, fk.Name, attr)
		}
		refAttr := fk.ReferencedAttributes[i]
		refName, ok := targetColumns[refAttr]
		if !ok {
			return physical.ForeignKey{}, fmt.Errorf("mapping %s foreign key %s target attribute %s has no column mapping in %s", source.ID, fk.Name, refAttr, target.ID)
		}
		out.Columns = append(out.Columns, name)
		out.ReferencedColumns = append(out.ReferencedColumns, refName)
	}
	return out, nil
}

func columnNamesByAttribute(m Mapping) map[core.CanonicalID]string {
	out := make(map[core.CanonicalID]string, len(m.Columns))
	for _, c := range m.Columns {
		out[c.Attribute] = c.Column
	}
	return out
}

func mustDatabaseID(s string) core.CanonicalID { return core.MustCanonicalID(s) }

func splitQualified(name string) (string, string) {
	parts := strings.Split(name, ".")
	if len(parts) == 1 {
		return "", parts[0]
	}
	return strings.Join(parts[:len(parts)-1], "."), parts[len(parts)-1]
}
