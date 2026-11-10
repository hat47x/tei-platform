-- Synthetic fixture for the REAL order_entry.recent_orders query shape only.
-- Apply only to a fresh, isolated disposable PostgreSQL database.
-- Distinct from postgres_seed.sql (sales_order), which intentionally tests generic Workbench data.
-- Source-static input; does not prove TEI Provider binding, runtime SQL or Rowset retrieval.
CREATE TABLE public.orders (
    order_id TEXT PRIMARY KEY,
    customer_id TEXT NOT NULL,
    ordered_at TIMESTAMPTZ NOT NULL,
    status TEXT NOT NULL,
    total_amount NUMERIC(12,2) NOT NULL
);

INSERT INTO public.orders (order_id, customer_id, ordered_at, status, total_amount) VALUES
    ('O-001', 'C-101', '2026-11-07T10:00:00Z', 'accepted', 120.50),
    ('O-002', 'C-101', '2026-11-07T10:05:00Z', 'pending', 0.00),
    ('O-003', 'C-202', '2026-11-07T10:10:00Z', 'accepted', 5.25);
