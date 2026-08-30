SELECT
            o.order_id,
            o.customer_id,
            o.ordered_at,
            o.status,
            o.total_amount
        FROM public.orders AS o
        WHERE o.ordered_at >= :from
        ORDER BY o.ordered_at DESC
