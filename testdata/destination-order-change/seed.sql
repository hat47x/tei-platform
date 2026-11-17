-- 到達点「受注の一変更」の明細。値は期待と一緒に手で決めた。
INSERT INTO public.order_lines (order_line_id, order_id, line_number, item_code, quantity, unit_price) VALUES
  (1, 1001, 1, 'A-100', 2, 100),
  (2, 1001, 2, 'B-200', 0, 250),
  (3, 1001, 2, 'C-300', 5, 0),
  (4, 1002, 1, 'A-100', 3, 100),
  (5, 1002, 2, 'B-200', 12, 250);
-- 受注1003には明細を置かない。
