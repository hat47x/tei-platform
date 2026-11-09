-- Synthetic fixture only. Apply exclusively to an isolated disposable PostgreSQL database.
-- This file is a native SQL source fixture, not accepted TEI business meaning.
CREATE TABLE sales_order (
    id TEXT PRIMARY KEY,
    customer_id TEXT NOT NULL,
    amount NUMERIC(12,2) NOT NULL,
    status TEXT NOT NULL
);

INSERT INTO sales_order (id, customer_id, amount, status) VALUES
    ('O-001', 'C-101', 120.50, 'accepted'),
    ('O-002', 'C-101',   0.00, 'pending'),
    ('O-003', 'C-202',   5.25, 'accepted');
