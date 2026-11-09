package ordermigrationdualdb

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// Customer is one customer master row. On the legacy side Name is
// customer_name; on the primary side it is full_name.
type Customer struct {
	Code        string `json:"code"`
	Name        string `json:"name"`
	Address     string `json:"address"`
	CreditLimit string `json:"creditLimit"`
}

// Customer master fields whose owner is declared per field.
const (
	FieldName        = "name"
	FieldAddress     = "address"
	FieldCreditLimit = "credit_limit"
)

// CustomerFields lists the owned fields in a fixed order.
func CustomerFields() []string { return []string{FieldName, FieldAddress, FieldCreditLimit} }

func (c Customer) field(f string) string {
	switch f {
	case FieldName:
		return c.Name
	case FieldAddress:
		return c.Address
	default:
		return c.CreditLimit
	}
}

func (c *Customer) setField(f, v string) {
	switch f {
	case FieldName:
		c.Name = v
	case FieldAddress:
		c.Address = v
	default:
		c.CreditLimit = v
	}
}

// primaryColumn is the primary column of a field.
func primaryColumn(f string) string {
	if f == FieldName {
		return "full_name"
	}
	return f
}

// legacyColumn is the legacy column of a field.
func legacyColumn(f string) string {
	if f == FieldName {
		return "customer_name"
	}
	return f
}

func loadLegacyCustomers(ctx context.Context, q querier) ([]Customer, error) {
	return scanCustomers(ctx, q, `SELECT customer_code, customer_name, address, credit_limit::text FROM omdd.legacy_customer ORDER BY customer_code COLLATE "C"`)
}

func loadPrimaryCustomers(ctx context.Context, q querier) ([]Customer, error) {
	return scanCustomers(ctx, q, `SELECT customer_code, full_name, address, credit_limit::text FROM omdd.customer ORDER BY customer_code COLLATE "C"`)
}

func scanCustomers(ctx context.Context, q querier, sql string, args ...any) ([]Customer, error) {
	rows, err := q.Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Customer
	for rows.Next() {
		var c Customer
		if err := rows.Scan(&c.Code, &c.Name, &c.Address, &c.CreditLimit); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// loadCustomerMaster is the start of step 4: the customer master copied in
// step 2 becomes the primary customer master, every field owned by the
// legacy side, and orders stay under legacy authority. It is idempotent.
func loadCustomerMaster(ctx context.Context, tx pgx.Tx, plan Plan) error {
	if _, err := tx.Exec(ctx, `INSERT INTO omdd.customer(customer_code, full_name, address, credit_limit)
		SELECT customer_code, customer_name, address, credit_limit FROM omdd.stage_customer WHERE plan_digest=$1
		ON CONFLICT (customer_code) DO NOTHING`, plan.Digest); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `INSERT INTO omdd.customer_legacy_base(customer_code, customer_name, address, credit_limit)
		SELECT customer_code, customer_name, address, credit_limit FROM omdd.stage_customer WHERE plan_digest=$1
		ON CONFLICT (customer_code) DO NOTHING`, plan.Digest); err != nil {
		return err
	}
	for _, f := range CustomerFields() {
		tag, err := tx.Exec(ctx, `INSERT INTO omdd.master_field_owner(entity, field, owner) VALUES ('customer',$1,'legacy') ON CONFLICT DO NOTHING`, f)
		if err != nil {
			return err
		}
		if tag.RowsAffected() == 1 {
			if _, err := tx.Exec(ctx, `INSERT INTO omdd.master_field_owner_history(entity, field, owner, reason) VALUES ('customer',$1,'legacy','initial:copied-from-legacy')`, f); err != nil {
				return err
			}
		}
	}
	_, err := tx.Exec(ctx, `INSERT INTO omdd.mig_authority(entity, authority) VALUES ('order','legacy') ON CONFLICT DO NOTHING`)
	return err
}

func fieldOwners(ctx context.Context, q querier) (map[string]string, error) {
	rows, err := q.Query(ctx, `SELECT field, owner FROM omdd.master_field_owner WHERE entity='customer'`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]string{}
	for rows.Next() {
		var f, o string
		if err := rows.Scan(&f, &o); err != nil {
			return nil, err
		}
		out[f] = o
	}
	return out, rows.Err()
}

func orderAuthority(ctx context.Context, q pgx.Tx) (string, string, error) {
	var auth string
	var gate *string
	err := q.QueryRow(ctx, `SELECT authority, gate_digest FROM omdd.mig_authority WHERE entity='order'`).Scan(&auth, &gate)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", "", nil
	}
	g := ""
	if gate != nil {
		g = *gate
	}
	return auth, g, err
}

// SwitchFieldOwner moves the master of one customer field (P-INT-03). The
// switch is recorded with its reason; reverse sync carries later primary
// changes of the field to the legacy side.
func SwitchFieldOwner(ctx context.Context, primary *pgx.Conn, plan Plan, field, owner, reason string) error {
	if owner != "legacy" && owner != "primary" {
		return fmt.Errorf("owner %q: want legacy or primary", owner)
	}
	known := false
	for _, f := range CustomerFields() {
		known = known || f == field
	}
	if !known {
		return fmt.Errorf("unknown customer field %q", field)
	}
	tx, err := primary.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(context.Background())
	if _, err := requireBound(ctx, tx, plan); err != nil {
		return err
	}
	if err := loadCustomerMaster(ctx, tx, plan); err != nil {
		return err
	}
	if err := setOwner(ctx, tx, field, owner, "", reason); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func setOwner(ctx context.Context, tx pgx.Tx, field, owner, gate, reason string) error {
	if _, err := tx.Exec(ctx, `UPDATE omdd.master_field_owner SET owner=$2, gate_digest=NULLIF($3,''), changed_at=now() WHERE entity='customer' AND field=$1`, field, owner, gate); err != nil {
		return err
	}
	_, err := tx.Exec(ctx, `INSERT INTO omdd.master_field_owner_history(entity, field, owner, reason) VALUES ('customer',$1,$2,$3)`, field, owner, reason)
	return err
}

// ErrNotOwner is returned when the new side writes a field it does not own.
var ErrNotOwner = errors.New("field is not owned by the primary side")

// ErrFrozen is returned for writes on the primary while orders are being
// rolled back to the legacy side.
var ErrFrozen = errors.New("primary writes are frozen during rollback")

// CustomerChange is a new-application change of one customer. Nil fields
// are not changed. FamilyName and GivenName are the split name of P-MIG-03
// and need the expand phase.
type CustomerChange struct {
	Address     *string
	CreditLimit *string
	FamilyName  *string
	GivenName   *string
}

// UpdateCustomer is the new application's write of the customer master. It
// refuses fields the primary does not own.
func UpdateCustomer(ctx context.Context, primary *pgx.Conn, code string, ch CustomerChange) error {
	tx, err := primary.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(context.Background())
	if auth, _, err := orderAuthority(ctx, tx); err != nil {
		return err
	} else if auth == "rolling-back" {
		return ErrFrozen
	}
	owners, err := fieldOwners(ctx, tx)
	if err != nil {
		return err
	}
	need := func(f string) error {
		if owners[f] != "primary" {
			return fmt.Errorf("%w: %s (owner %q)", ErrNotOwner, f, owners[f])
		}
		return nil
	}
	sets, args := "", []any{code}
	add := func(col string, v *string, cast string) {
		args = append(args, *v)
		if sets != "" {
			sets += ", "
		}
		sets += fmt.Sprintf("%s = $%d%s", col, len(args), cast)
	}
	if ch.Address != nil {
		if err := need(FieldAddress); err != nil {
			return err
		}
		add("address", ch.Address, "")
	}
	if ch.CreditLimit != nil {
		if err := need(FieldCreditLimit); err != nil {
			return err
		}
		add("credit_limit", ch.CreditLimit, "::text::numeric")
	}
	if ch.FamilyName != nil || ch.GivenName != nil {
		if ch.FamilyName == nil || ch.GivenName == nil {
			return errors.New("family and given names are written together")
		}
		if err := need(FieldName); err != nil {
			return err
		}
		add("family_name", ch.FamilyName, "")
		add("given_name", ch.GivenName, "")
	}
	if sets == "" {
		return nil
	}
	if _, err := tx.Exec(ctx, `SET LOCAL omdd.origin = 'new-app'`); err != nil {
		return err
	}
	tag, err := tx.Exec(ctx, `UPDATE omdd.customer SET `+sets+` WHERE customer_code=$1`, args...)
	if err != nil {
		return err
	}
	if tag.RowsAffected() != 1 {
		return fmt.Errorf("customer %s not found", code)
	}
	return tx.Commit(ctx)
}

// applyLegacyCustomer applies one decoded legacy customer row under the
// field ownership rule. For each field that changed against the base:
//
//   - owner legacy: the primary takes the legacy value. If the primary had a
//     different value of its own, that is recorded as owner-wins:legacy.
//   - owner primary, and the primary also changed it since the base: the
//     primary value stays (owner-wins:primary); reverse sync overwrites the
//     legacy side.
//   - owner primary, and the primary did not change it: the legacy side
//     wrote a field it does not own and nothing decides who is right. The
//     change is queued for a person and the primary value stays.
//   - no declared owner: queued.
//
// The base becomes the legacy row in every case.
func applyLegacyCustomer(ctx context.Context, tx pgx.Tx, plan Plan, lsn LSN, legacyRow Customer, owners map[string]string, res *ApplyResult) error {
	var base Customer
	err := tx.QueryRow(ctx, `SELECT customer_code, customer_name, address, credit_limit::text FROM omdd.customer_legacy_base WHERE customer_code=$1 FOR UPDATE`, legacyRow.Code).
		Scan(&base.Code, &base.Name, &base.Address, &base.CreditLimit)
	if errors.Is(err, pgx.ErrNoRows) {
		// A customer created on the legacy side after the copy.
		if _, err := tx.Exec(ctx, `INSERT INTO omdd.customer(customer_code, full_name, address, credit_limit) VALUES ($1,$2,$3,$4::text::numeric)`,
			legacyRow.Code, legacyRow.Name, legacyRow.Address, legacyRow.CreditLimit); err != nil {
			return err
		}
		res.CustomersInserted++
		_, err = tx.Exec(ctx, `INSERT INTO omdd.customer_legacy_base(customer_code, customer_name, address, credit_limit) VALUES ($1,$2,$3,$4::text::numeric)`,
			legacyRow.Code, legacyRow.Name, legacyRow.Address, legacyRow.CreditLimit)
		return err
	}
	if err != nil {
		return err
	}
	cur, err := scanCustomers(ctx, tx, `SELECT customer_code, full_name, address, credit_limit::text FROM omdd.customer WHERE customer_code=$1 FOR UPDATE`, legacyRow.Code)
	if err != nil {
		return err
	}
	if len(cur) != 1 {
		return fmt.Errorf("customer %s has a base but no primary row", legacyRow.Code)
	}
	prim := cur[0]
	sets, args := "", []any{legacyRow.Code}
	conflict := func(f, owner, state, resolution string) error {
		res.Conflicts[resolution]++
		_, err := tx.Exec(ctx, `INSERT INTO omdd.master_conflict(plan_digest, customer_code, field, owner, legacy_value, primary_value, base_value, commit_lsn, state, resolution)
			VALUES ($1,$2,$3,NULLIF($4,''),$5,$6,$7,$8::pg_lsn,$9,$10)`,
			plan.Digest, legacyRow.Code, f, owner, legacyRow.field(f), prim.field(f), base.field(f), lsn.String(), state, resolution)
		return err
	}
	for _, f := range CustomerFields() {
		l, b, p := legacyRow.field(f), base.field(f), prim.field(f)
		if l == b {
			continue
		}
		owner := owners[f]
		switch owner {
		case "legacy":
			if p != b && p != l {
				if err := conflict(f, owner, "resolved", "owner-wins:legacy"); err != nil {
					return err
				}
			}
			if p != l {
				args = append(args, l)
				if sets != "" {
					sets += ", "
				}
				cast := ""
				if f == FieldCreditLimit {
					cast = "::text::numeric"
				}
				sets += fmt.Sprintf("%s = $%d%s", primaryColumn(f), len(args), cast)
			}
		case "primary":
			if p != b {
				if err := conflict(f, owner, "resolved", "owner-wins:primary"); err != nil {
					return err
				}
			} else if err := conflict(f, owner, "queued", "queued:non-owner-write-by-legacy"); err != nil {
				return err
			}
		default:
			if err := conflict(f, owner, "queued", "queued:no-declared-owner"); err != nil {
				return err
			}
		}
	}
	if sets != "" {
		if _, err := tx.Exec(ctx, `UPDATE omdd.customer SET `+sets+` WHERE customer_code=$1`, args...); err != nil {
			return err
		}
		res.CustomersUpdated++
	}
	_, err = tx.Exec(ctx, `UPDATE omdd.customer_legacy_base SET customer_name=$2, address=$3, credit_limit=$4::text::numeric WHERE customer_code=$1`,
		legacyRow.Code, legacyRow.Name, legacyRow.Address, legacyRow.CreditLimit)
	return err
}

// ResolveMasterConflict is a person's decision on a queued conflict. With
// acceptLegacy the primary takes the legacy value (and reverse sync confirms
// it on the legacy side); otherwise the primary value is kept and reverse
// sync overwrites the legacy value.
func ResolveMasterConflict(ctx context.Context, primary *pgx.Conn, plan Plan, id int64, acceptLegacy bool) error {
	tx, err := primary.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(context.Background())
	if _, err := requireBound(ctx, tx, plan); err != nil {
		return err
	}
	var code, field, state string
	var legacyValue *string
	if err := tx.QueryRow(ctx, `SELECT customer_code, field, state, legacy_value FROM omdd.master_conflict WHERE id=$1 FOR UPDATE`, id).Scan(&code, &field, &state, &legacyValue); err != nil {
		return err
	}
	if state != "queued" {
		return fmt.Errorf("conflict %d is %s", id, state)
	}
	resolution := "kept-primary"
	if _, err := tx.Exec(ctx, `SET LOCAL omdd.origin = 'conflict-resolution'`); err != nil {
		return err
	}
	if acceptLegacy && legacyValue != nil {
		resolution = "accepted-legacy"
		cast := ""
		if field == FieldCreditLimit {
			cast = "::text::numeric"
		}
		if _, err := tx.Exec(ctx, fmt.Sprintf(`UPDATE omdd.customer SET %s = $2%s WHERE customer_code=$1`, primaryColumn(field), cast), code, *legacyValue); err != nil {
			return err
		}
	}
	// Either way the legacy side must end with the primary value.
	if _, err := tx.Exec(ctx, `INSERT INTO omdd.reverse_outbox(plan_digest, entity, entity_key, fields) VALUES ($1,'customer',$2,ARRAY[$3])`, plan.Digest, code, field); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `UPDATE omdd.master_conflict SET state='resolved', resolution=$2, resolved_at=now() WHERE id=$1`, id, resolution); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
