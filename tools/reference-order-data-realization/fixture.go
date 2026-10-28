package orderdatarealization

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"strings"
)

// The chain uses exactly one disposable database across all steps. It checks
// that the database is empty before it starts, marks every object it creates
// with an ownership comment, and at the end drops only marked objects.

const ownerPrefix = "tei:reference-order-data-realization:"

// metaSchema holds the sample's own bookkeeping (writer registry, numbering
// ledger). It is not part of the storage mapping and the schema comparison
// is projected to the mapped schema (public) only.
const metaSchema = "odr_meta"

const userRelationsQuery = `
SELECT n.nspname || '.' || c.relname
FROM pg_catalog.pg_class c
JOIN pg_catalog.pg_namespace n ON n.oid = c.relnamespace
WHERE c.relkind IN ('r','p','v','m','S','f')
  AND n.nspname NOT IN ('pg_catalog','information_schema')
  AND n.nspname !~ '^pg_toast' AND n.nspname !~ '^pg_temp'
ORDER BY 1`

const userSchemasQuery = `
SELECT nspname FROM pg_catalog.pg_namespace
WHERE nspname NOT IN ('public','pg_catalog','information_schema')
  AND nspname !~ '^pg_'
ORDER BY 1`

// DatabaseSafety is what the chain records about the database it used.
type DatabaseSafety struct {
	Database            string   `json:"database"`
	UserRelationsBefore int      `json:"userRelationsBefore"`
	ExtraSchemasBefore  int      `json:"extraSchemasBefore"`
	EventTriggersBefore int      `json:"eventTriggersBefore"`
	OwnershipMarker     string   `json:"ownershipMarker"`
	Created             []string `json:"created"`
	Dropped             []string `json:"dropped,omitempty"`
	LeftUnowned         []string `json:"leftUnowned,omitempty"`
	UserRelationsAfter  int      `json:"userRelationsAfter"`
	ExtraSchemasAfter   int      `json:"extraSchemasAfter"`
}

func listStrings(ctx context.Context, db *sql.DB, query string) ([]string, error) {
	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var s string
		if err := rows.Scan(&s); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

func checkDedicatedEmpty(ctx context.Context, db *sql.DB, marker string) (DatabaseSafety, error) {
	s := DatabaseSafety{OwnershipMarker: marker}
	if err := db.QueryRowContext(ctx, `SELECT current_database()`).Scan(&s.Database); err != nil {
		return s, err
	}
	switch s.Database {
	case "", "postgres", "template0", "template1":
		return s, fmt.Errorf("refusing database %q: a dedicated empty database is required", s.Database)
	}
	rels, err := listStrings(ctx, db, userRelationsQuery)
	if err != nil {
		return s, err
	}
	schemas, err := listStrings(ctx, db, userSchemasQuery)
	if err != nil {
		return s, err
	}
	if err := db.QueryRowContext(ctx, `SELECT count(*) FROM pg_catalog.pg_event_trigger`).Scan(&s.EventTriggersBefore); err != nil {
		return s, err
	}
	s.UserRelationsBefore, s.ExtraSchemasBefore = len(rels), len(schemas)
	if len(rels) != 0 || len(schemas) != 0 || s.EventTriggersBefore != 0 {
		return s, fmt.Errorf("refusing database %s: not empty (relations %v, schemas %v, event triggers %d)", s.Database, rels, schemas, s.EventTriggersBefore)
	}
	return s, nil
}

func quoteIdent(s string) string { return `"` + strings.ReplaceAll(s, `"`, `""`) + `"` }

func quoteLiteral(s string) string { return `'` + strings.ReplaceAll(s, `'`, `''`) + `'` }

// markStatement returns the COMMENT statement that marks a created object.
func markStatement(kind, qualified, marker string) string {
	return fmt.Sprintf("COMMENT ON %s %s IS %s", kind, qualified, quoteLiteral(marker))
}

// cleanupOwned drops what the chain created. A relation in public is dropped
// only when its comment equals the marker; the meta schema is dropped only
// when the schema and every relation in it carry the marker.
func cleanupOwned(ctx context.Context, db *sql.DB, s *DatabaseSafety) error {
	var problems []string
	rels, err := listStrings(ctx, db, userRelationsQuery)
	if err != nil {
		return err
	}
	for _, rel := range rels {
		schema, name, _ := strings.Cut(rel, ".")
		if schema != "public" {
			continue
		}
		qualified := quoteIdent(schema) + "." + quoteIdent(name)
		tx, err := db.BeginTx(ctx, nil)
		if err != nil {
			return err
		}
		var comment sql.NullString
		if _, err := tx.ExecContext(ctx, "SET LOCAL lock_timeout = '5s'"); err == nil {
			_, err = tx.ExecContext(ctx, "LOCK TABLE "+qualified+" IN ACCESS EXCLUSIVE MODE")
			if err == nil {
				err = tx.QueryRowContext(ctx, `SELECT pg_catalog.obj_description($1::regclass, 'pg_class')`, qualified).Scan(&comment)
			}
			if err == nil && comment.Valid && comment.String == s.OwnershipMarker {
				_, err = tx.ExecContext(ctx, "DROP TABLE "+qualified)
				if err == nil {
					err = tx.Commit()
					if err == nil {
						s.Dropped = append(s.Dropped, rel)
						continue
					}
				}
			}
			if err != nil {
				problems = append(problems, fmt.Sprintf("%s: %v", rel, err))
			}
		}
		tx.Rollback()
		s.LeftUnowned = append(s.LeftUnowned, rel)
	}
	var schemaComment sql.NullString
	err = db.QueryRowContext(ctx, `SELECT pg_catalog.obj_description(oid, 'pg_namespace') FROM pg_catalog.pg_namespace WHERE nspname = $1`, metaSchema).Scan(&schemaComment)
	switch {
	case err == sql.ErrNoRows:
	case err != nil:
		problems = append(problems, err.Error())
	case schemaComment.String != s.OwnershipMarker:
		s.LeftUnowned = append(s.LeftUnowned, "schema "+metaSchema)
	default:
		unmarked, err := listStrings(ctx, db, `
SELECT c.relname FROM pg_catalog.pg_class c JOIN pg_catalog.pg_namespace n ON n.oid = c.relnamespace
WHERE n.nspname = '`+metaSchema+`' AND c.relkind IN ('r','p','v','m','S','f')
  AND coalesce(pg_catalog.obj_description(c.oid, 'pg_class'), '') <> `+quoteLiteral(s.OwnershipMarker))
		if err != nil {
			problems = append(problems, err.Error())
		} else if len(unmarked) > 0 {
			s.LeftUnowned = append(s.LeftUnowned, "schema "+metaSchema+" (unmarked: "+strings.Join(unmarked, ",")+")")
		} else if _, err := db.ExecContext(ctx, "DROP SCHEMA "+metaSchema+" CASCADE"); err != nil {
			problems = append(problems, err.Error())
		} else {
			s.Dropped = append(s.Dropped, "schema "+metaSchema)
		}
	}
	after, err := listStrings(ctx, db, userRelationsQuery)
	if err != nil {
		return err
	}
	schemas, err := listStrings(ctx, db, userSchemasQuery)
	if err != nil {
		return err
	}
	s.UserRelationsAfter, s.ExtraSchemasAfter = len(after), len(schemas)
	sort.Strings(s.Dropped)
	if len(problems) > 0 || len(s.LeftUnowned) > 0 || len(after) > 0 || len(schemas) > 0 {
		return fmt.Errorf("cleanup incomplete: problems %v, left %v, relations after %v, schemas after %v", problems, s.LeftUnowned, after, schemas)
	}
	return nil
}
