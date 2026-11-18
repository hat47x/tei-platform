-- 第一Web Targetの一覧Queryが読む親受注。明細はseed.sqlで別に投入する。
-- 受注1003は明細0件なので、一覧の空明細ケースも実DBで選べる。
INSERT INTO public.orders (order_id, customer_id, ordered_at, status, total_amount) VALUES
  (1001, 11, '2026-09-18T09:30:00Z', 'draft', 12500.00),
  (1002, 12, '2026-09-18T08:15:00Z', 'draft', 8300.00),
  (1003, 13, '2026-09-18T07:30:00Z', 'draft', 6200.00);
