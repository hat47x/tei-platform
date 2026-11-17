-- Independent database model. Same CLI, but a different physical table,
-- Query input, column names and composite identity.
CREATE TABLE public.inventory_entries (
    entry_id bigint NOT NULL,
    batch_key bigint NOT NULL,
    slot bigint NOT NULL,
    sku varchar NOT NULL,
    units numeric(12,3) NOT NULL,
    cost numeric(12,2) NOT NULL,
    PRIMARY KEY (entry_id, slot)
);
INSERT INTO public.inventory_entries (entry_id, batch_key, slot, sku, units, cost) VALUES
    (11, 1001, 1, 'X-1',  3, 100),
    (12, 1001, 2, 'X-2',  0, 200),
    (13, 1001, 2, 'X-3',  6, 300),
    (14, 1002, 1, 'Y-1',  0, 400),
    (15, 1002, 2, 'Y-2',  9, 500);
