-- TEST/REFERENCE ONLY. Not part of a TEI production migration.
-- Run manually in an isolated PostgreSQL test database with suitable rights.
-- The existing order_entry.recent_orders canonical SQL still uses public.orders;
-- this separate relation supplies a tenant discriminator for scope tests.
CREATE TABLE public.tei_web_scoped_orders_reference (
    tenant_id text NOT NULL,
    order_id bigint NOT NULL,
    customer_id bigint NOT NULL,
    ordered_at timestamptz NOT NULL,
    status text NOT NULL,
    total_amount numeric(18,2) NOT NULL,
    PRIMARY KEY (tenant_id, order_id)
);

INSERT INTO public.tei_web_scoped_orders_reference
    (tenant_id, order_id, customer_id, ordered_at, status, total_amount)
VALUES
    ('tenant-a', 3, 11, '2026-10-01T08:00:00Z', 'OPEN', 3000.00),
    ('tenant-a', 9, 12, '2026-10-02T08:00:00Z', 'READY', 9000.00),
    ('tenant-b', 1, 21, '2026-10-01T08:00:00Z', 'OPEN', 100.00),
    ('tenant-b', 100, 22, '2026-10-02T08:00:00Z', 'READY', 99000.00);
