package postgresquery

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"sort"

	"github.com/jackc/pgx/v5"

	postgresprovider "tei/internal/database/providers/postgres"
	"tei/internal/functioncontext"
)

// ExecQueryer is a transaction: it can both query and run a statement. A pool is not enough,
// because the observations only mean something on the connection that does the read.
type ExecQueryer interface {
	Queryer
	ExecContext(context.Context, string, ...any) (sql.Result, error)
}

// ObserveAcquisition observes, on the transaction the read will use, what the read is about
// to touch. It must be the FIRST work on a fresh read-only transaction:
//
//  1. declared relations are locked ACCESS SHARE, so no DDL (for example enabling row level
//     security) can change them under the read;
//  2. the Provider is asked for its system identifier;
//  3. the statement the read will run is explained with the real arguments;
//  4. every relation the plan reads, and every declared relation, is asked for its kind and
//     for row_security_active in this session.
//
// It reports facts. It never decides coverage, and a failure of any single observation is
// recorded in the result rather than returned, because the read must still run and report its
// own error. The catalog flag pg_class.relrowsecurity is deliberately not read: inside a
// snapshot it can be stale, while row_security_active answers for the current session.
//
// Each statement runs under a savepoint, so a failing EXPLAIN (a wrong argument type, a
// missing column) does not abort the transaction and hide the read's own error.
func ObserveAcquisition(
	ctx context.Context,
	tx ExecQueryer,
	source string,
	parameters map[string]any,
	declaredTables []string,
) (functioncontext.AcquisitionObservation, error) {
	var obs functioncontext.AcquisitionObservation
	if tx == nil {
		return obs, fmt.Errorf("PostgreSQL acquisition observation needs a transaction")
	}

	for _, table := range declaredTables {
		name, err := qualified(table)
		if err != nil {
			obs.LockNote = err.Error()
			break
		}
		if err := inSavepoint(ctx, tx, func() error {
			_, err := tx.ExecContext(ctx, "LOCK TABLE "+name+" IN ACCESS SHARE MODE")
			return err
		}); err != nil {
			obs.LockNote = fmt.Sprintf("cannot lock %s: %v", table, err)
			break
		}
		obs.LockedTables = append(obs.LockedTables, table)
	}
	if obs.LockNote != "" {
		obs.LockedTables = nil
	}

	if err := inSavepoint(ctx, tx, func() error {
		rows, err := tx.QueryContext(ctx, `SELECT system_identifier::text FROM pg_control_system()`)
		if err != nil {
			return err
		}
		defer rows.Close()
		if rows.Next() {
			if err := rows.Scan(&obs.SystemIdentifier); err != nil {
				return err
			}
		}
		return rows.Err()
	}); err != nil {
		obs.SystemIdentifier = ""
	}

	bound, err := postgresprovider.BindNamedParameters(source, parameters)
	if err != nil {
		obs.PlanNote = fmt.Sprintf("cannot bind the statement: %v", err)
		return obs, nil
	}
	var raw []byte
	if err := inSavepoint(ctx, tx, func() error {
		rows, err := tx.QueryContext(ctx, "EXPLAIN (VERBOSE, FORMAT JSON) "+bound.SQL, bound.Args...)
		if err != nil {
			return err
		}
		defer rows.Close()
		if !rows.Next() {
			return fmt.Errorf("EXPLAIN returned no row")
		}
		if err := rows.Scan(&raw); err != nil {
			return err
		}
		return rows.Err()
	}); err != nil {
		obs.PlanNote = fmt.Sprintf("cannot explain the statement: %v", err)
		return obs, nil
	}
	relations, nodes, err := planFacts(raw)
	if err != nil {
		obs.PlanNote = err.Error()
		return obs, nil
	}
	obs.PlanRelations, obs.PlanNodes = relations, nodes

	seen := map[string]bool{}
	var names []string
	for _, n := range append(append([]string(nil), obs.PlanRelations...), declaredTables...) {
		if !seen[n] {
			seen[n] = true
			names = append(names, n)
		}
	}
	sort.Strings(names)
	for _, n := range names {
		rel := functioncontext.RelationObservation{Name: n}
		ident, err := qualified(n)
		if err == nil {
			err = inSavepoint(ctx, tx, func() error {
				rows, err := tx.QueryContext(ctx,
					`SELECT c.relkind::text, row_security_active(c.oid) FROM pg_class c WHERE c.oid = to_regclass($1)`, ident)
				if err != nil {
					return err
				}
				defer rows.Close()
				if rows.Next() {
					rel.Found = true
					if err := rows.Scan(&rel.RelKind, &rel.RowSecurityActive); err != nil {
						return err
					}
				}
				return rows.Err()
			})
		}
		if err != nil {
			// A relation that cannot be examined is reported as not found, never as plain.
			rel.Found = false
		}
		obs.Relations = append(obs.Relations, rel)
	}
	return obs, nil
}

// planFacts reads the relations (schema-qualified) and node types out of EXPLAIN JSON.
func planFacts(raw []byte) (relations, nodes []string, err error) {
	var docs []map[string]any
	if err := json.Unmarshal(raw, &docs); err != nil || len(docs) != 1 {
		return nil, nil, fmt.Errorf("cannot read the EXPLAIN JSON")
	}
	root, ok := docs[0]["Plan"].(map[string]any)
	if !ok {
		return nil, nil, fmt.Errorf("the EXPLAIN JSON has no Plan")
	}
	rel, kind := map[string]bool{}, map[string]bool{}
	var walk func(map[string]any)
	walk = func(n map[string]any) {
		if t, ok := n["Node Type"].(string); ok {
			kind[t] = true
		}
		if r, ok := n["Relation Name"].(string); ok {
			schema, _ := n["Schema"].(string)
			if schema == "" {
				// VERBOSE always names the schema; without it nothing can be matched safely.
				rel["?."+r] = true
			} else {
				rel[schema+"."+r] = true
			}
		}
		if kids, ok := n["Plans"].([]any); ok {
			for _, k := range kids {
				if m, ok := k.(map[string]any); ok {
					walk(m)
				}
			}
		}
	}
	walk(root)
	for r := range rel {
		relations = append(relations, r)
	}
	for k := range kind {
		nodes = append(nodes, k)
	}
	sort.Strings(relations)
	sort.Strings(nodes)
	return relations, nodes, nil
}

// qualified turns "schema.name" into a safely quoted identifier. An unqualified name is
// refused: the schema is not guessed from search_path.
func qualified(table string) (string, error) {
	for i := 0; i < len(table); i++ {
		if table[i] == '.' {
			schema, name := table[:i], table[i+1:]
			if schema == "" || name == "" {
				break
			}
			return pgx.Identifier{schema, name}.Sanitize(), nil
		}
	}
	return "", fmt.Errorf("relation %q is not schema-qualified", table)
}

func inSavepoint(ctx context.Context, tx ExecQueryer, run func() error) error {
	if _, err := tx.ExecContext(ctx, "SAVEPOINT tei_observation"); err != nil {
		return err
	}
	if err := run(); err != nil {
		_, _ = tx.ExecContext(ctx, "ROLLBACK TO SAVEPOINT tei_observation")
		return err
	}
	_, err := tx.ExecContext(ctx, "RELEASE SAVEPOINT tei_observation")
	return err
}
