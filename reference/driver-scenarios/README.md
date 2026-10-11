# Driver Scenario CLI generation — narrow Reference (#1180)

This is a small, explicit **semantic Pattern resolution and synthetic Browse Query** reference, not a full Driver execution Scenario. The test-case source is `order-entry-pattern-smoke.json`; neither the CLI command plan nor its run evidence becomes a second business or test SSOT.

With a trusted full checkout and Go:

```bash
go run ./cmd/tei scenario verify -scenario reference/driver-scenarios/order-entry-pattern-smoke.json -workspace reference/workspace
go run ./cmd/tei scenario generate -scenario reference/driver-scenarios/order-entry-pattern-smoke.json -workspace reference/workspace > /tmp/tei-scenario-plan.json
go run ./cmd/tei scenario check -scenario reference/driver-scenarios/order-entry-pattern-smoke.json -workspace reference/workspace -plan /tmp/tei-scenario-plan.json
go run ./cmd/tei scenario run -scenario reference/driver-scenarios/order-entry-pattern-smoke.json -workspace reference/workspace -step browse-pattern
```

Generated structured argv uses installed `tei` as the executable. The `go run` wrapper is only for running the current source checkout, not part of the generated argv. Running the generated command requires a `tei` binary from the matching checkout.

The runner calls the **existing** `driver.Resolver.ResolvePattern` and `driver.Resolver.ExecuteBrowseQuery`; the latter receives an explicitly named **synthetic** fixture result from `fixtures/recent_orders.json`. The Driver applies the Canonical Query's column-to-semantic-attribute mapping, and the Scenario independently asserts **all mapped attributes of all returned rows**. Unlike a real database Query provider, the synthetic executor returns fixture rows and never runs SQL against a database. It does not execute external SQL, Command, Authorization, independent Observation, or Web E2E. No Credential or external effect is produced. Unknown operations or omitted expectations fail before generation. The generated command plan pins the Scenario source SHA-256, synthetic Fixture source SHA-256, a hash of the resolved Canonical World and Workspace plugin-lock file hash. `check` rejects modified commands or any changed inputs. This is *not* an Accepted Canonical revision / resolved deployment attestation, so full cross-route qualification remains unresolved.

Usage for the added Query step:
```bash
go run ./cmd/tei scenario run -scenario reference/driver-scenarios/order-entry-pattern-smoke.json -workspace reference/workspace -step browse-synthetic
```

`verify` and `generate` validate the fixture binding but do not execute the Query; the result evidence explicitly says `synthetic-fixture-bound; query-not-executed`. `run` executes the synthetic Driver Query and reports `synthetic-browse-query-executed`; neither result is real-provider evidence. No secret values or authenticated authority are captured. The fixture path must remain inside the Scenario directory even through symlinks.

This small Reference should be extended only after the typed operation adapter and safely isolated fixture described in #1158/#1180 are available. Do not infer product-wide operation coverage from a single Pattern resolution fixture.
