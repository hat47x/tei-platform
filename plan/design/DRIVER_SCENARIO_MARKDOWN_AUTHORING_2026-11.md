# Driver Scenario Markdown Authoring

- Status: design candidate
- Owner issue: #1180
- Reuse: #1131, #2514, #2621, #988, #2620
- Contract: existing JSON Scenario is the SSOT for Scenario tests. Markdown authoring is a human-friendly authoring surface for the same JSON.

## 1. Goal

Domain experts write concise test tables in `*.tei.md`. A reader converts them into the existing `tei.driver-scenario/v0` JSON so the current `cmd/tei scenario verify/generate/run/check` pipeline works unchanged.

## 2. Non-goals

- Do not add a second Scenario store.
- Do not change Scenario semantics.
- Do not replace `reference/driver-scenarios/*.json` as the executable Scenario source.
- Do not create a new generic test DSL language.

## 3. Authoring contract

A Markdown test table is one explicit block with the frontmatter tag `tei-test-scenario: "true"`. Normal documentation tables in the same document are ignored by the Scenario converter.

Each table must have exactly these columns:
`step id` | `operation` | `pattern` | `expect` | `fixture` | `expected result`

### 3.1 Operation vocabulary (initial)

- `resolve-pattern`
- `evaluate-validations`
- `observe-synthetic-state`
- `execute-browse-query` is recognized as a JSON operation but fails closed in this Markdown surface until a typed row-expectation shorthand exists. It never emits a Scenario with empty `expectRows`.

### 3.2 Mapping to JSON

| Markdown column | JSON field |
|---|---|
| step id | `step.id` |
| operation | `step.operation` |
| pattern | `step.pattern` |
| expect | `step.expectKind` / `step.expectData` / `step.expectQuery` |
| fixture | `step.fixture` |
| expected result | validation/observation/row expectation |

### 3.3 Expect cell shorthand (first implementation)

- Pattern step: `browse:query-name` or `record:data-name`
- Validation step: comma list of `rule-id=satisfied|not-satisfied`
- Observation step: `observation:<value>` with independent fixture

For `observe-synthetic-state`, the converter synthesizes `readIdentity` as `synthetic.<scenario-id>.<step-id>`. The independent Observation Fixture must declare the same `readIdentity`; the converter does not read fixture files.

### 3.4 Example

```markdown
---
application: order_entry
tei-test-scenario: "true"
---

# Order entry smoke scenario

| step id | operation | pattern | expect | fixture | expected result |
|---|---|---|---|---|---|
| browse-pattern | resolve-pattern | order_entry.recent_orders | browse:sales.order,query:order_entry.recent_orders | - | - |
| validate-positive | evaluate-validations | - | - | fixtures/order_valid.json | sales.order_line.quantity_positive=satisfied,sales.order.lines.line_number_unique=satisfied |
| observe-synthetic | observe-synthetic-state | - | - | fixtures/order_state_pending.json | observation:pending |
```

## 4. Reader behavior

1. Find every fenced code block with language `tei-scenario-md`.
2. Parse the frontmatter-like heading lines `application:` and `tei-test-scenario: "true"`.
3. Parse the first Markdown table under that heading.
4. Convert each row to a Step. Unsupported operation or malformed cell fails closed.
5. Emit deterministic JSON equal to `scenariotest.Generate` input.
6. If the same Scenario ID appears twice in one block, reject it.

## 5. Source preservation

- Store generated JSON in `reference/driver-scenarios/from-md/`.
- Keep the original Markdown as the review surface.
- Keep Markdown-referenced fixtures under `reference/driver-scenarios/from-md/fixtures/` so the generated JSON resolves them relative to its own directory.
- Do not overwrite existing hand-authored JSON files.

## 6. First verification

Run the existing Scenario CLI against the generated JSON and compare `generate/check` output with the smoke Scenario.

## 7. Remaining tasks

- Define a human-friendly typed row-expectation shorthand for `execute-browse-query` and remove its fail-closed rejection.
- Decide whether multiple `tei-scenario-md` blocks in one document are one Scenario or several, and reject duplicate Scenario IDs explicitly.
- Tighten the table contract to reject duplicate or missing required columns instead of relying on per-operation cell checks.

