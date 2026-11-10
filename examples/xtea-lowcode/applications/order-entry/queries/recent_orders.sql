select order_id, customer_id
from sales.order
where case when :status_operator = 'ne' then status <> :status else status = :status end
order by status asc, order_id asc
limit 20 offset :page_offset
