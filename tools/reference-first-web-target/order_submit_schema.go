//go:build !tei_repair_only

package main

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"

	"tei/internal/core"
	"tei/internal/database/physical"
	postgresprovider "tei/internal/database/providers/postgres"
	dbrealization "tei/internal/database/realization"
	"tei/internal/database/schemadiff"
	"tei/internal/environment"
	"tei/internal/ir"
)

// errPhysicalSchemaNotReady refuses the submit Command before it writes when
// the observed physical table does not match what the accepted World declares
// (S02, physical side). It is distinct from a Source revision mismatch, which
// the Source Gate refuses earlier. The public error carries no table name or
// connection detail.
var errPhysicalSchemaNotReady = errors.New("PHYSICAL_SCHEMA_NOT_READY")

// The submit store writes and reads exactly one table. The readiness check
// covers that table only; other tables of the Database are not this Command's
// write set and are not checked here.
const (
	orderSubmitEnvironment = "development"
	orderSubmitAlias       = "sales"
	orderSubmitSchemaName  = "public"
	orderSubmitTableName   = "orders"
)

// orderSubmitSchemaReadiness is the evidence of one comparison. Changes and
// Unresolved list schemadiff results by kind and object; they never contain
// data or connection detail.
type orderSubmitSchemaReadiness struct {
	Database   string   `json:"database"`
	Table      string   `json:"table"`
	Compatible bool     `json:"compatible"`
	Changes    []string `json:"changes,omitempty"`
}

// desiredOrderSubmitTable derives the declared physical table from the
// accepted World's Persistence Mapping through the environment binding of
// order_entry, without a caller-supplied target.
func desiredOrderSubmitTable(world ir.World) (physical.SchemaSnapshot, environment.DatabaseInstance, error) {
	var env *environment.Environment
	for i := range world.Environments {
		if world.Environments[i].ID == core.MustCanonicalID(orderSubmitEnvironment) {
			env = &world.Environments[i]
		}
	}
	if env == nil {
		return physical.SchemaSnapshot{}, environment.DatabaseInstance{}, errors.New("environment binding not found")
	}
	var matches []environment.ApplicationDatabaseBinding
	for _, b := range env.Bindings {
		if b.Application == core.MustCanonicalID("order_entry") && b.Alias == core.LocalName(orderSubmitAlias) {
			matches = append(matches, b)
		}
	}
	if len(matches) != 1 {
		return physical.SchemaSnapshot{}, environment.DatabaseInstance{}, fmt.Errorf("%d order_entry database bindings", len(matches))
	}
	instance, ok := env.InstanceByID(matches[0].Instance)
	if !ok || instance.Provider != environment.ProviderPostgres {
		return physical.SchemaSnapshot{}, environment.DatabaseInstance{}, errors.New("order_entry database instance is not PostgreSQL")
	}
	desired, err := dbrealization.RealizeDesired(world.Persistence, string(instance.Definition), instance.Provider)
	if err != nil {
		return physical.SchemaSnapshot{}, environment.DatabaseInstance{}, err
	}
	desired = onlyOrderSubmitTable(desired)
	if len(desired.Tables) != 1 {
		return physical.SchemaSnapshot{}, environment.DatabaseInstance{}, errors.New("declared orders table not found")
	}
	return desired, instance, nil
}

func onlyOrderSubmitTable(snapshot physical.SchemaSnapshot) physical.SchemaSnapshot {
	out := physical.SchemaSnapshot{Database: snapshot.Database}
	for _, table := range snapshot.Tables {
		schema := table.Schema
		if schema == "" {
			schema = orderSubmitSchemaName
		}
		if schema == orderSubmitSchemaName && table.Name == orderSubmitTableName {
			out.Tables = append(out.Tables, table)
		}
	}
	return out
}

// checkOrderSubmitSchema observes the catalog over the store's write pool and
// compares it with the declaration. It only reads the catalog.
func (s *postgresOrderSubmitStore) checkOrderSubmitSchema(ctx context.Context, world ir.World) (orderSubmitSchemaReadiness, error) {
	desired, instance, err := desiredOrderSubmitTable(world)
	if err != nil {
		return orderSubmitSchemaReadiness{}, errPhysicalSchemaNotReady
	}
	readiness := orderSubmitSchemaReadiness{Database: string(desired.Database), Table: orderSubmitSchemaName + "." + orderSubmitTableName}
	actual, err := postgresprovider.Provider{Catalog: postgresprovider.SQLCatalogReader{DB: s.write}}.Introspect(ctx, instance)
	if err != nil {
		return readiness, errPhysicalSchemaNotReady
	}
	diff, err := schemadiff.Compare(desired, onlyOrderSubmitTable(actual))
	if err != nil {
		return readiness, errPhysicalSchemaNotReady
	}
	for _, c := range diff.Changes {
		part := string(c.Kind) + " " + c.Table
		switch {
		case c.Column != nil:
			part += "." + c.Column.Name
		case c.BeforeColumn != nil:
			part += "." + c.BeforeColumn.Name
		case c.Index != nil:
			part += "." + c.Index.Name
		case c.UniqueConstraint != nil:
			part += "." + c.UniqueConstraint.Name
		case c.ForeignKey != nil:
			part += "." + c.ForeignKey.Name
		}
		readiness.Changes = append(readiness.Changes, part)
	}
	for _, u := range diff.Unresolved {
		readiness.Changes = append(readiness.Changes, "unresolved "+u.Table+"."+u.ObjectName)
	}
	sort.Strings(readiness.Changes)
	readiness.Compatible = len(readiness.Changes) == 0
	if !readiness.Compatible {
		return readiness, errPhysicalSchemaNotReady
	}
	return readiness, nil
}

// schemaCheckedOrderSubmitStore re-checks the physical table immediately
// before each write, so a table changed after startup is refused before the
// UPDATE runs. Reads are not gated: an Observation of a changed table is
// already reported separately.
type schemaCheckedOrderSubmitStore struct {
	store *postgresOrderSubmitStore
	world ir.World
}

func (s schemaCheckedOrderSubmitStore) Submit(ctx context.Context, orderID int64) error {
	if _, err := s.store.checkOrderSubmitSchema(ctx, s.world); err != nil {
		return errPhysicalSchemaNotReady
	}
	return s.store.Submit(ctx, orderID)
}

func (s schemaCheckedOrderSubmitStore) ReadStatus(ctx context.Context, orderID int64) (string, error) {
	return s.store.ReadStatus(ctx, orderID)
}

// requireOrderSubmitSchema is the startup form: a Web process whose real store
// does not match the accepted declaration does not publish HTTP. Stores other
// than the PostgreSQL one are returned unchanged.
func requireOrderSubmitSchema(ctx context.Context, world ir.World, store orderSubmitStore) (orderSubmitStore, error) {
	pg, ok := store.(*postgresOrderSubmitStore)
	if !ok {
		return store, nil
	}
	if _, err := pg.checkOrderSubmitSchema(ctx, world); err != nil {
		return nil, err
	}
	return schemaCheckedOrderSubmitStore{store: pg, world: world}, nil
}

// renderOrderSubmitTableDDL renders the declared orders table as PostgreSQL DDL
// with the existing Provider renderer (used by tests to build a fixture that
// matches the declaration instead of hand-written DDL).
func renderOrderSubmitTableDDL(world ir.World) ([]string, error) {
	desired, _, err := desiredOrderSubmitTable(world)
	if err != nil {
		return nil, err
	}
	diff, err := schemadiff.Compare(desired, physical.SchemaSnapshot{Database: desired.Database})
	if err != nil {
		return nil, err
	}
	statements, err := postgresprovider.RenderMigrationSQL(diff)
	if err != nil {
		return nil, err
	}
	for i := range statements {
		statements[i] = strings.TrimSpace(statements[i])
	}
	return statements, nil
}
