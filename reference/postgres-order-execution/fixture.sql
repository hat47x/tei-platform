-- Dedicated disposable PostgreSQL database only. Existing objects cause failure.
CREATE TABLE public.orders (
    order_id bigint PRIMARY KEY,
    customer_id bigint NOT NULL,
    ordered_at timestamptz NOT NULL,
    status text NOT NULL,
    total_amount numeric NOT NULL
);
CREATE TABLE public.order_lines (
    order_line_id bigint PRIMARY KEY,
    order_id bigint NOT NULL REFERENCES public.orders(order_id),
    line_number integer NOT NULL,
    item_code text NOT NULL,
    quantity numeric NOT NULL,
    unit_price numeric NOT NULL
);
CREATE TABLE public.tei_reference_execution_grants (
    scope text NOT NULL,
    provider_id text NOT NULL,
    contract_id text NOT NULL,
    contract_version text NOT NULL,
    artifact_digest text NOT NULL,
    generation bigint NOT NULL,
    fencing_token bigint NOT NULL,
    expires_at timestamptz NOT NULL,
    revoked boolean NOT NULL DEFAULT false,
    model_revision text NOT NULL DEFAULT '',
    PRIMARY KEY(scope, provider_id)
);
COMMENT ON TABLE public.orders IS 'tei.reference.postgres-order-execution.v1';
COMMENT ON TABLE public.order_lines IS 'tei.reference.postgres-order-execution.v1';
COMMENT ON TABLE public.tei_reference_execution_grants IS 'tei.reference.postgres-order-execution.v1';

INSERT INTO public.orders VALUES
    (1001, 11, '2026-09-18T09:30:00Z', 'draft', 12500.00),
    (1002, 12, '2026-09-18T08:15:00Z', 'draft', 8300.00),
    (1003, 13, '2026-09-18T07:30:00Z', 'draft', 6200.00);
INSERT INTO public.order_lines VALUES
    (5001, 1001, 1, 'ITEM-A', 2, 4000.00),
    (5002, 1001, 2, 'ITEM-B', 1, 4500.00),
    (5003, 1002, 1, 'ITEM-C', 0, 8300.00),
    (5004, 1003, 1, 'ITEM-D', 1, 6200.00);

-- The host supplies the digest of the actual executable through psql -v.
-- These rows are reference host permission, never canonical business data.
INSERT INTO public.tei_reference_execution_grants
    (scope, provider_id, contract_id, contract_version, artifact_digest, generation, fencing_token, expires_at, revoked) VALUES
    ('reference/order_entry/development', 'order-read', 'tei.reference.order-entry.read', '1', :'artifact_digest', 1, 1, clock_timestamp() + interval '10 minutes', false),
    ('reference/order_entry/development', 'order-submit', 'tei.reference.order-entry.postgres-submit', '1', :'artifact_digest', 1, 1, clock_timestamp() + interval '10 minutes', false),
    ('reference/order_entry/development', 'order-report', 'tei.reference.order-entry.report', '1', :'artifact_digest', 1, 1, clock_timestamp() + interval '10 minutes', false),
    ('reference/order_entry/development', 'order-batch', 'tei.reference.order-entry.batch', '1', :'artifact_digest', 1, 1, clock_timestamp() + interval '10 minutes', false);
