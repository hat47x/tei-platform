select order_id, customer_id
from sales.order
where case :status_operator when 'eq' then status = :status when 'ne' then status <> :status end
order by status asc, order_id asc
limit :page_size offset :page_offset
