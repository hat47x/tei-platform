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

## Current qualification boundary

The `scenario generate` output embeds `-scenario-sha256`, `-world-sha256`, `-plugin-lock-sha256` and, for synthetic Browse steps, `-fixture-sha256`. The `scenario run` entry compares supplied pins **before** calling the Driver Query. A direct hand-written `scenario run` without these options is permitted for exploratory testing, but it is not evidence of executing the previously generated command revision.

The test-only typed values include string, int64, float64 and RFC3339 timestamp. NaN and infinity are refused. The whole raw row is mapped using the actual Canonical Query declaration; Scenario assertions must cover every semantic mapped column on every row. A missing mapping or wrong fixture parameter is an execution failure, not a passing zero-result Query.

This remains a **draft implementation candidate** until actual exact-head full-checkout Go tests, repository health check and CLI integration with parallel PRs are verified. World and Plugin Lock *hashes* are not substitutes for Accepted Revision/Policy/Authority qualification or external provider evidence.

## Canonical Validation scenarios

Two additional persisted steps (`validate-positive`, `validate-denied`) execute the existing `driver.ResolveApplicationValidations` and `validationeval.Evaluate` over explicit synthetic input fixtures. Every Application-declared Validation must appear in the Scenario's expected outcomes; rules are **never selected from the Scenario**.

The expected negative business outcome `not-satisfied` is a valid, successful *negative test*, not an Execution Error. A missing input, unexpected outcome, duplicated or undeclared rule, or extra unused fixture input is an Execution Error. These steps do not authenticate or authorize the caller, invoke Command, or perform independent Observation, and are **not** evidence for those paths.

```bash
go run ./cmd/tei scenario run -scenario reference/driver-scenarios/order-entry-pattern-smoke.json -workspace reference/workspace -step validate-positive
go run ./cmd/tei scenario run -scenario reference/driver-scenarios/order-entry-pattern-smoke.json -workspace reference/workspace -step validate-denied
```
