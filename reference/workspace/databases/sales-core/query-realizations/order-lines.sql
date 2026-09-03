SELECT
    order_line_id,
    line_number,
    item_code,
    quantity,
    unit_price
FROM public.order_lines
WHERE order_id = :order_id
ORDER BY line_number;
