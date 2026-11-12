# B2C change coverage — bounded read-only Reference (#3509)

**NOT a product feature or a complete native-code analyzer.** The tool accepts only an explicitly declared dependency inventory and computes reachable impacts. Its `declaredScopeClosed` field is an **unverified input assertion**: even the `declared-closed-only` outcome is never a whole-site non-impact, release, authority, payment, or conversion claim.

Run locally (standalone standard-library Go package):

```sh
go test ./tools/reference-b2c-change-inspector
cat tools/reference-b2c-change-inspector/example-input.json | go run ./tools/reference-b2c-change-inspector
```

The edges point from a changed subject **toward its dependents**. Layers provide hints only (`content → C0`, `presentation → C1`, `interaction → C2`, `business/consent → C3`, `release/measurement → C4`). There are no inferred dynamic imports, provider capabilities, authoritative business checks, consent permissions, or actual delivery readbacks. Unknown nodes and the caller-declared `unobserved` list prevent a scoped noncritical result. Duplicate node identity and unknown layer are errors.

Provenance of real source bytes, exact provider invocation, host authorization, complete requested set, actual browser delivery, production statistics, and actual effect remain owned by #2586, #1158, #951, #1174, #2514, #2600, #3271, #3509 and existing target consumers. This reference must not be wired as an automatic deploy approval or production security gate.

The manually fixed `example-expected.json` is checked against the CLI output by `TestFixedFixtureMatchesIndependentExpected`. The expected output is not regenerated from the inspector.

Cases demonstrate the existing BC-16/27/41 (C0 vs C3), BC-18/19/32 (partial Graph), BC-35 (identity), and BC-36 (bounded H1) concerns, but they do not claim those BCs have passed end-to-end. The JSON representation is **fixture-local**, not an accepted TEI contract, plugin ABI, schema, or new scenario SSOT.
