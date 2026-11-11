---
application: order_entry
tei-test-scenario: "true"
---

# Order entry scenario from Markdown authoring

This block is explicit test authoring, not a normal explanatory table.

```tei-scenario-md
| step id | operation | pattern | expect | fixture | expected result |
|---|---|---|---|---|---|
| browse-pattern | resolve-pattern | order_entry.recent_orders | browse:sales.order,query:order_entry.recent_orders | - | - |
| validate-positive | evaluate-validations | - | - | fixtures/order_valid.json | sales.order_line.quantity_positive=satisfied,sales.order.lines.line_number_unique=satisfied,sales.order_line.line_number_positive=satisfied |
| observe-synthetic | observe-synthetic-state | - | - | fixtures/order_state_pending.json | observation:pending |
```
