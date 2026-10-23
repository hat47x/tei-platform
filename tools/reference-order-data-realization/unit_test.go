package orderdatarealization

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"

	"tei/internal/database/persistence"
	"tei/internal/database/physical"
	"tei/internal/database/schemadiff"
)

func TestClassifyRegistration(t *testing.T) {
	cases := []struct {
		err  error
		want string
		code string
	}{
		{nil, OutcomeCommitted, ""},
		{fmt.Errorf("wrap: %w", errCancelled), OutcomeCancelled, ""},
		{fmt.Errorf("wrap: %w", &pgconn.PgError{Code: "23505"}), OutcomeConfirmedFailure, "23505"},
		{fmt.Errorf("wrap: %w", &pgconn.PgError{Code: "57P01"}), OutcomeUnknown, "57P01"},
		{fmt.Errorf("wrap: %w", &pgconn.PgError{Code: "08006"}), OutcomeUnknown, "08006"},
		{errors.New("unexpected EOF"), OutcomeUnknown, ""},
	}
	for _, c := range cases {
		got, code := classifyRegistration(c.err)
		if got != c.want || code != c.code {
			t.Errorf("%v: got %s %s, want %s %s", c.err, got, code, c.want, c.code)
		}
	}
}

func TestClassifyObservationNeverObservedOnFailure(t *testing.T) {
	if classifyObservation(true, errors.New("closed"), true) != ObsReadFailed {
		t.Fatal("a failed read became observed")
	}
	if classifyObservation(false, nil, true) != ObsNotCommitted {
		t.Fatal("an uncommitted command became observed")
	}
	if classifyObservation(true, nil, false) != ObsMismatch || classifyObservation(true, nil, true) != ObsObserved {
		t.Fatal("classification")
	}
}

func snapshotWith(cols []physical.Column, unique []physical.UniqueConstraint) physical.SchemaSnapshot {
	return physical.SchemaSnapshot{Database: databaseID, Tables: []physical.Table{{
		Schema: "public", Name: "orders", Columns: cols, PrimaryKey: []string{"order_id"}, UniqueConstraints: unique,
	}}}
}

func TestPlanMigrationStagesAndDecisions(t *testing.T) {
	d, _, err := LoadDeclarations()
	if err != nil {
		t.Fatal(err)
	}
	base := []physical.Column{{Name: "order_id", PhysicalType: "pg_catalog.int8", Ordinal: 1}, {Name: "external_id", PhysicalType: "pg_catalog.varchar", Nullable: true, Ordinal: 2}}
	desired := snapshotWith(append(append([]physical.Column(nil), base...), physical.Column{Name: "delivery_note", PhysicalType: "pg_catalog.varchar", Nullable: true, Ordinal: 3}),
		[]physical.UniqueConstraint{{Name: "orders_external_id_key", Columns: []string{"external_id"}}})
	actual := snapshotWith(base, nil)
	p, err := planMigration("t", "sha256:r3", d.Migration, desired, actual)
	if err != nil {
		t.Fatal(err)
	}
	expand, contract := p.stage("expand"), p.stage("contract")
	if expand == nil || contract == nil || expand.RenderedBy != "provider:postgres" || contract.RenderedBy != "sample" || contract.ProviderRenderError == "" {
		t.Fatalf("stages %+v", p.Stages)
	}
	if !strings.Contains(contract.Statements[0], `ADD CONSTRAINT "orders_external_id_key" UNIQUE ("external_id")`) || !contract.RefuseWhileOldWriters {
		t.Fatalf("contract %+v", contract)
	}
	if p.Definition.BasedOnActualFingerprint != observationFingerprint(actual) || p.DefinitionFingerprint == "" {
		t.Fatal("expand definition not bound to the observation")
	}
	// A physical-only column would be dropped: not applied automatically.
	drift := snapshotWith(append(append([]physical.Column(nil), base...), physical.Column{Name: "drift_note", PhysicalType: "pg_catalog.text", Nullable: true, Ordinal: 3}), nil)
	p2, err := planMigration("t", "sha256:r3", d.Migration, desired, drift)
	if !errors.Is(err, ErrNeedsDecision) || len(p2.NeedsDecision) != 1 || p2.NeedsDecision[0].Kind != schemadiff.DropColumn {
		t.Fatalf("drop-column not held for a decision: %v %+v", err, p2)
	}
	if observationFingerprint(drift) == observationFingerprint(actual) {
		t.Fatal("fingerprint does not see the drift")
	}
	a2 := actual
	a2.CapturedAt = time.Now().Format(time.RFC3339)
	if observationFingerprint(a2) != observationFingerprint(actual) {
		t.Fatal("fingerprint depends on the capture time")
	}
}

func TestRenderSampleChangeOnlyUnique(t *testing.T) {
	if _, err := renderSampleChange(schemadiff.Change{Kind: schemadiff.AddForeignKey, Table: "public.orders"}); err == nil {
		t.Fatal("foreign key rendered by the sample")
	}
}

func TestExternalNumberPolicies(t *testing.T) {
	iss := newIssuer(70001)
	n, how, err := externalNumber(context.Background(), iss, PolicyProvisional, "PROV-", 1)
	if err != nil || how != "issued" || n != "0070001" {
		t.Fatalf("issued %q %q %v", n, how, err)
	}
	iss.stop()
	n, how, err = externalNumber(context.Background(), iss, PolicyProvisional, "PROV-", 42)
	if err != nil || how != "provisional" || n != "PROV-42" {
		t.Fatalf("provisional %q %q %v", n, how, err)
	}
	if _, _, err := externalNumber(context.Background(), iss, PolicyStop, "PROV-", 43); !errors.Is(err, ErrIssuerUnavailable) {
		t.Fatalf("stop policy: %v", err)
	}
}

func TestCompareValueSemantics(t *testing.T) {
	cases := []struct {
		pg, sq     any
		st         persistence.StorageType
		equal, scl bool
	}{
		{"800.00", "800.00", persistence.StorageDecimal, true, false},
		{"800.00", json.Number("800"), persistence.StorageDecimal, true, true},
		{"123456789012345678901234.5678", json.Number("1.2345678901234569e+23"), persistence.StorageDecimal, false, false},
		{"0070001", json.Number("70001"), persistence.StorageVarchar, false, false},
		{"", nil, persistence.StorageVarchar, false, false},
		{nil, nil, persistence.StorageVarchar, true, false},
		{time.Date(2026, 10, 22, 23, 30, 0, 0, time.FixedZone("x", -5*3600)), "2026-10-23T04:30:00.000000Z", persistence.StorageDateTimeTZ, true, false},
		{int64(5), json.Number("5"), persistence.StorageInt64, true, false},
	}
	for i, c := range cases {
		_, _, eq, scl := compareValue(c.pg, c.sq, c.st)
		if eq != c.equal || scl != c.scl {
			t.Errorf("case %d: equal %v scale %v", i, eq, scl)
		}
	}
}

func TestSQLiteSchemaFromStorageMapping(t *testing.T) {
	c := newTestChain(t)
	if err := CopyWorkspace(c.Source, c.WorkDir); err != nil {
		t.Fatal(err)
	}
	if _, err := applyLineEdit(c.WorkDir, uniqueEdit(c.Decl)); err != nil {
		t.Fatal(err)
	}
	w, errs, err := LoadValidated(c.WorkDir)
	if err != nil || len(errs) > 0 {
		t.Fatal(err, errs)
	}
	s, err := sqliteSchema(w, c.Decl.SQLite)
	if err != nil {
		t.Fatal(err)
	}
	ddl := strings.Join(s["orders"].DDL, "\n")
	for _, want := range []string{`"total_amount" TEXT NOT NULL`, `"external_id" TEXT`, `CONSTRAINT "orders_external_id_key" UNIQUE ("external_id")`, `PRIMARY KEY ("order_id")`, `CREATE INDEX "orders_customer_idx"`} {
		if !strings.Contains(ddl, want) {
			t.Errorf("orders DDL lacks %s:\n%s", want, ddl)
		}
	}
	if !strings.Contains(strings.Join(s["order_lines"].Columns, ","), "order_id") {
		t.Error("relation storage column missing in order_lines")
	}
	naive, err := sqliteSchema(w, c.Decl.SQLiteNaive)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(strings.Join(naive["orders"].DDL, "\n"), `"total_amount" NUMERIC`) {
		t.Error("naive dialect not applied")
	}
}

// TestSQLiteRunnerNeedsPython checks the interpreter lookup fails instead of
// skipping when the configured interpreter does not exist.
func TestSQLiteRunnerNeedsPython(t *testing.T) {
	t.Setenv(PythonEnv, "/nonexistent/python3")
	if _, err := runSQLite(context.Background(), sqliteRequest{Path: t.TempDir() + "/x.sqlite"}); err == nil {
		t.Fatal("a missing interpreter did not fail")
	}
}

func TestInsertForRefusesUnmappedAttribute(t *testing.T) {
	c := newTestChain(t)
	if err := CopyWorkspace(c.Source, c.R0Dir); err != nil {
		t.Fatal(err)
	}
	w0, _, err := LoadValidated(c.R0Dir)
	if err != nil {
		t.Fatal(err)
	}
	rw, err := rowWriterOf(w0)
	if err != nil {
		t.Fatal(err)
	}
	vals := map[string]any{"sales.order.order_id": 1, "sales.order.customer_id": 1, "sales.order.ordered_at": baseTime, "sales.order.status": "x", "sales.order.total": "1", "sales.order.external_id": nil}
	if _, _, err := insertFor(rw.order, vals, nil); err != nil {
		t.Fatal(err)
	}
	vals[c.Decl.Attribute.Attribute] = "note"
	if _, _, err := insertFor(rw.order, vals, nil); err == nil {
		t.Fatal("R0 mapping accepted the new attribute")
	}
}
