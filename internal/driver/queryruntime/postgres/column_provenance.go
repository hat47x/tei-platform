package postgresquery

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/jackc/pgx/v5/stdlib"

	postgresprovider "tei/internal/database/providers/postgres"
	"tei/internal/functioncontext"
)

// ObserveColumnProvenance asks the Provider to describe the statement the read will run and
// reports, for each result column, which relation column it is (or that it is not one). The
// statement is prepared, not executed, so no row is read.
//
// database/sql does not expose the wire-level row description, so this borrows a second
// connection from the pool and talks to the driver directly. That is safe only because the
// read's transaction already holds ACCESS SHARE on the declared relations (see
// ObserveAcquisition): a rename or drop cannot change what a name resolves to meanwhile.
//
// A failure is returned as a note, never an error, so the read still runs and reports its own.
func ObserveColumnProvenance(
	ctx context.Context,
	db *sql.DB,
	source string,
	parameters map[string]any,
) ([]functioncontext.ColumnProvenance, string) {
	bound, err := postgresprovider.BindNamedParameters(source, parameters)
	if err != nil {
		return nil, fmt.Sprintf("cannot bind the statement: %v", err)
	}
	conn, err := db.Conn(ctx)
	if err != nil {
		return nil, "cannot borrow a connection to describe the statement"
	}
	defer conn.Close()

	var out []functioncontext.ColumnProvenance
	var note string
	rawErr := conn.Raw(func(driverConn any) error {
		pgConn, ok := driverConn.(*stdlib.Conn)
		if !ok {
			note = "the connection is not a pgx connection"
			return nil
		}
		pc := pgConn.Conn()
		description, err := pc.Prepare(ctx, "", bound.SQL)
		if err != nil {
			note = fmt.Sprintf("cannot describe the statement: %v", err)
			return nil
		}
		for _, field := range description.Fields {
			column := functioncontext.ColumnProvenance{Column: field.Name}
			if field.TableOID != 0 && field.TableAttributeNumber > 0 {
				var relation, kind, attname string
				err := pc.QueryRow(ctx, `SELECT n.nspname || '.' || c.relname, c.relkind::text, a.attname
					FROM pg_class c
					JOIN pg_namespace n ON n.oid = c.relnamespace
					JOIN pg_attribute a ON a.attrelid = c.oid AND a.attnum = $2
					WHERE c.oid = $1`, field.TableOID, int32(field.TableAttributeNumber)).Scan(&relation, &kind, &attname)
				if err != nil {
					// A column whose relation cannot be resolved is reported as computed:
					// nothing is claimed about it.
					out = append(out, column)
					continue
				}
				column.Relation, column.RelKind, column.Attname = relation, kind, attname
			}
			out = append(out, column)
		}
		return nil
	})
	if rawErr != nil {
		return nil, fmt.Sprintf("cannot describe the statement: %v", rawErr)
	}
	if note != "" {
		return nil, note
	}
	return out, ""
}
