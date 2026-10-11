# Driver Scenario CLI generation — narrow Reference (#1180)

This is a small, explicit **semantic Pattern resolution** reference, not a full Driver execution Scenario. The test-case source is `order-entry-pattern-smoke.json`; neither the CLI command plan nor its run evidence becomes a second business or test SSOT.

With a trusted full checkout and Go:

```bash
go run ./cmd/tei scenario verify -scenario reference/driver-scenarios/order-entry-pattern-smoke.json -workspace reference/workspace
go run ./cmd/tei scenario generate -scenario reference/driver-scenarios/order-entry-pattern-smoke.json -workspace reference/workspace > /tmp/tei-scenario-plan.json
go run ./cmd/tei scenario check -scenario reference/driver-scenarios/order-entry-pattern-smoke.json -workspace reference/workspace -plan /tmp/tei-scenario-plan.json
go run ./cmd/tei scenario run -scenario reference/driver-scenarios/order-entry-pattern-smoke.json -workspace reference/workspace -step browse-pattern
```

Generated structured argv uses installed `tei` as the executable. The `go run` wrapper is only for running the current source checkout, not part of the generated argv. Running the generated command requires a `tei` binary from the matching checkout.

The current runner calls the **existing** `driver.Resolver.ResolvePattern` and asserts expected Pattern kind/Data/Query. It does not execute SQL, Command, Authorization, independent Observation, or Web E2E. No Credential or external effect is produced. Unknown operations or omitted expectations fail before generation. The source SHA-256 binds generated commands to the exact Scenario bytes; `check` rejects a stale or modified plan. It does **not** yet bind to a Canonical revision or Plugin Lock, so full cross-route qualification is not possible.

This small Reference should be extended only after the typed operation adapter and safely isolated fixture described in #1158/#1180 are available. Do not infer product-wide operation coverage from a single Pattern resolution fixture.
