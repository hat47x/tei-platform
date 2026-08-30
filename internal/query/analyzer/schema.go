package analyzer

import "tei/internal/database/physical"

func SchemaFromSnapshot(s physical.SchemaSnapshot) SchemaContext {
	ctx := SchemaContext{}
	for _, t := range s.Normalize().Tables {
		entry := SchemaTable{Name: t.QualifiedName()}
		for _, c := range t.Columns {
			entry.Columns = append(entry.Columns, c.Name)
		}
		ctx.Tables = append(ctx.Tables, entry)
	}
	return ctx
}
