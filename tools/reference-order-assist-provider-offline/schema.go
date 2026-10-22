package orderassistoffline

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// Two schemas in one PostgreSQL database, each owned by its own login role:
// oapo (the order side: quotes, synchronisation, approvals, the credit step)
// and oapo_credit (the simulated external credit provider). The roles cannot
// read each other's schema, so the order side learns the provider's state
// only through the provider's HTTP API.
const (
	RoleServer = "oapo_server"
	RoleCredit = "oapo_credit"
)

const serverDDL = `
CREATE SCHEMA oapo AUTHORIZATION oapo_server;
SET ROLE oapo_server;
CREATE TABLE oapo.clock (
  id int PRIMARY KEY CHECK (id = 1),
  seq bigint NOT NULL,
  business_date date NOT NULL
);
CREATE TABLE oapo.price_list (
  version int PRIMARY KEY,
  revised_seq bigint NOT NULL
);
CREATE TABLE oapo.price (
  version int NOT NULL REFERENCES oapo.price_list(version),
  sku text NOT NULL,
  unit_price_minor bigint NOT NULL CHECK (unit_price_minor >= 0),
  PRIMARY KEY (version, sku)
);
CREATE TABLE oapo.accepted_generation (
  id int PRIMARY KEY CHECK (id = 1),
  min_generation int NOT NULL,
  max_generation int NOT NULL
);
CREATE TABLE oapo.terminal_realization (
  generation int NOT NULL,
  artifact_digest text NOT NULL,
  PRIMARY KEY (generation, artifact_digest)
);
CREATE TABLE oapo.quote (
  quote_id text PRIMARY KEY,
  doc jsonb NOT NULL,
  field_versions jsonb NOT NULL,
  version bigint NOT NULL,
  status text NOT NULL CHECK (status IN ('valid','stale-price','conflict','divergent')),
  digest text NOT NULL,
  revalidation jsonb NOT NULL,
  created_by_op text NOT NULL,
  updated_seq bigint NOT NULL
);
CREATE TABLE oapo.sync_op (
  op_id text PRIMARY KEY,
  terminal_id text NOT NULL,
  quote_id text NOT NULL,
  kind text NOT NULL,
  body jsonb NOT NULL,
  generation int NOT NULL,
  realization_digest text NOT NULL,
  price_list_version int NOT NULL,
  outcome text NOT NULL,
  detail jsonb NOT NULL,
  applied_seq bigint NOT NULL
);
CREATE TABLE oapo.conflict (
  conflict_id bigserial PRIMARY KEY,
  quote_id text NOT NULL,
  path text NOT NULL,
  op_id text NOT NULL REFERENCES oapo.sync_op(op_id),
  base_version int NOT NULL,
  op_value jsonb,
  server_value jsonb,
  server_version int NOT NULL,
  status text NOT NULL CHECK (status IN ('open','resolved')),
  resolution jsonb
);
CREATE TABLE oapo.event (
  seq bigint PRIMARY KEY,
  kind text NOT NULL,
  quote_id text,
  case_id text,
  detail jsonb NOT NULL
);
CREATE TABLE oapo.principal (
  principal_id text PRIMARY KEY,
  role text NOT NULL
);
CREATE TABLE oapo.delegation (
  delegation_id bigserial PRIMARY KEY,
  delegator text NOT NULL REFERENCES oapo.principal(principal_id),
  delegate text NOT NULL REFERENCES oapo.principal(principal_id),
  role text NOT NULL,
  valid_from date NOT NULL,
  valid_to date NOT NULL CHECK (valid_to >= valid_from)
);
CREATE TABLE oapo.policy (
  version text PRIMARY KEY,
  digest text NOT NULL,
  definition jsonb NOT NULL,
  active_for_new_cases boolean NOT NULL DEFAULT false
);
CREATE UNIQUE INDEX policy_one_active ON oapo.policy ((true)) WHERE active_for_new_cases;
CREATE TABLE oapo.approval_case (
  case_id text PRIMARY KEY,
  quote_id text NOT NULL REFERENCES oapo.quote(quote_id),
  quote_digest text NOT NULL,
  policy_version text NOT NULL REFERENCES oapo.policy(version),
  policy_digest text NOT NULL,
  stage text,
  status text NOT NULL CHECK (status IN ('pending','approved','rejected')),
  opened_seq bigint NOT NULL,
  history jsonb NOT NULL DEFAULT '[]'
);
CREATE TABLE oapo.approval_action (
  action_id bigserial PRIMARY KEY,
  case_id text NOT NULL REFERENCES oapo.approval_case(case_id),
  policy_version text NOT NULL,
  stage text NOT NULL,
  decision text NOT NULL CHECK (decision IN ('approve','sendback','reject')),
  actor text NOT NULL,
  on_behalf_of text,
  delegation_id bigint,
  quote_digest text NOT NULL,
  business_date date NOT NULL,
  seq bigint NOT NULL,
  valid boolean NOT NULL,
  invalidated_reason text,
  migrated_from text
);
CREATE TABLE oapo.credit_step (
  case_id text PRIMARY KEY REFERENCES oapo.approval_case(case_id),
  quote_digest text NOT NULL,
  idempotency_key text NOT NULL UNIQUE,
  customer_code text NOT NULL,
  amount_minor bigint NOT NULL,
  state text NOT NULL CHECK (state IN ('reserve-pending','unknown','consumed','refused','release-pending','released')),
  reservation_id bigint,
  attempts int NOT NULL DEFAULT 0,
  release_attempts int NOT NULL DEFAULT 0,
  history jsonb NOT NULL DEFAULT '[]'
);
CREATE TABLE oapo.handoff (
  case_id text PRIMARY KEY REFERENCES oapo.approval_case(case_id),
  quote_id text NOT NULL,
  quote_digest text NOT NULL,
  policy_version text NOT NULL,
  outcome text NOT NULL CHECK (outcome IN ('ready-for-confirmation','withdrawn')),
  credit_reservation_id bigint,
  credit_state text,
  detail jsonb NOT NULL,
  seq bigint NOT NULL
);
RESET ROLE;
`

const creditDDL = `
CREATE SCHEMA oapo_credit AUTHORIZATION oapo_credit;
SET ROLE oapo_credit;
CREATE TABLE oapo_credit.account (
  customer_code text PRIMARY KEY,
  limit_minor bigint NOT NULL,
  used_minor bigint NOT NULL CHECK (used_minor >= 0)
);
CREATE TABLE oapo_credit.reservation (
  reservation_id bigserial PRIMARY KEY,
  idempotency_key text UNIQUE,
  customer_code text NOT NULL REFERENCES oapo_credit.account(customer_code),
  amount_minor bigint NOT NULL CHECK (amount_minor > 0),
  quote_ref text NOT NULL,
  status text NOT NULL CHECK (status IN ('consumed','released')),
  release_count int NOT NULL DEFAULT 0
);
CREATE TABLE oapo_credit.request_log (
  id bigserial PRIMARY KEY,
  op text NOT NULL,
  idempotency_key text,
  outcome text NOT NULL
);
CREATE TABLE oapo_credit.fault (
  op text PRIMARY KEY CHECK (op IN ('reserve','release')),
  mode text NOT NULL CHECK (mode IN ('drop-after-commit')),
  remaining int NOT NULL
);
RESET ROLE;
`

// ResetSchemas drops and recreates both schemas and their roles. The caller
// must have confirmed the database is the disposable one.
func ResetSchemas(ctx context.Context, admin *pgx.Conn) error {
	stmts := []string{
		`DROP SCHEMA IF EXISTS oapo CASCADE`,
		`DROP SCHEMA IF EXISTS oapo_credit CASCADE`,
	}
	for _, role := range []string{RoleServer, RoleCredit} {
		stmts = append(stmts,
			fmt.Sprintf(`DO $$ BEGIN IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = '%[1]s') THEN CREATE ROLE %[1]s LOGIN; END IF; END $$`, role),
			fmt.Sprintf(`GRANT CREATE, CONNECT ON DATABASE %s TO %s`, pgx.Identifier{admin.Config().Database}.Sanitize(), role),
		)
	}
	stmts = append(stmts, serverDDL, creditDDL,
		`REVOKE ALL ON SCHEMA oapo FROM PUBLIC`,
		`REVOKE ALL ON SCHEMA oapo_credit FROM PUBLIC`,
	)
	for _, s := range stmts {
		if _, err := admin.Exec(ctx, s); err != nil {
			return fmt.Errorf("schema: %w\n%s", err, s)
		}
	}
	return nil
}
