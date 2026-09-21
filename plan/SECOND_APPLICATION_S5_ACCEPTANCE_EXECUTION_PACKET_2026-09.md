# Second Application Reference — S5 Acceptance Execution Packet

- Date: 2026-09-21
- Primary workstream: ISSUE-000040
- Proposal fingerprint: `c18ae0fdef8e01aa965e2ad84c86ecfc21ca0608c07b1808e73368a5f434458d`
- Independent review: completed — `review-clear-with-conditions`
- Required pre-Acceptance fixes: none
- Human Acceptance decision: **still required**
- This document is an execution contract, **not** an Acceptance witness.

## 1. Stop condition

Do not run actual `accept` until an explicit human witness unambiguously accepts or rejects the exact Proposal fingerprint above.

A general instruction such as「進めてください」is not that witness.

The witness record must include:

- exact Proposal fingerprint
- accepted / rejected decision
- decision context
- retained unresolved
- witness identity / locus
- decision time or repository revision

If the decision is rejection, do not run the remainder of this packet.

## 2. Preconditions

Use a clean, Go-enabled full checkout on current `main`.

Repository policy:

```text
direct-to-main
no new remote work branch
pull --rebase before push
never force-push main
code change => tools/repository-health-check.sh before push
```

Before any write:

```bash
git status --short
git pull --rebase
PRE_ACCEPT_HEAD="$(git rev-parse HEAD)"
```

`git status --short` must be empty.

Relevant current Workspace revision recorded by the successful Preview:

```text
sha256:f70da9b64a5b48c7682b24c64f455b0b0c2a7278be4faa297b6ff99c2d3707bf
```

Check it again:

```bash
go run ./tools/reference-shared-acceptance revision \
  -workspace reference/workspace
```

If the revision differs, **stop**. Do not silently edit the candidate's base revision. Re-run freshness / Preview qualification and determine whether the independent review remains applicable.

## 3. Exact typed candidate

Create a temporary candidate outside authoritative source, for example:

```bash
cat > /tmp/tei-second-application-candidate.json <<'JSON'
{
  "schema": "tei.reference.change-candidate/v0",
  "id": "cand-second-application-repair-intake",
  "baseModelRevision": "sha256:f70da9b64a5b48c7682b24c64f455b0b0c2a7278be4faa297b6ff99c2d3707bf",
  "origin": {
    "kind": "cli",
    "actor": "human-acceptance"
  },
  "intent": "Second Application repair_intake（改訂版S4 composition）をSemantic / Application / Patternの3 sourceとして追加する",
  "evidence": [
    {
      "subject": "order_entry.customer_reference",
      "note": "origin。provenanceだけで、意味はcopyしない"
    },
    {
      "subject": "customer.customer"
    }
  ],
  "delta": [
    {
      "op": "add-data",
      "placement": "semantic/repair-intake/intake.tei",
      "data": {
        "namespace": "repair_intake",
        "id": "repair_intake.intake",
        "symbol": "Intake",
        "attributes": [
          {
            "id": "repair_intake.intake.customer_id",
            "symbol": "customerId",
            "type": "customer.customer_id"
          }
        ],
        "relations": [
          {
            "id": "repair_intake.intake.customer",
            "symbol": "customer",
            "target": "customer.customer",
            "cardinality": "one",
            "keys": [
              {
                "sourceAttribute": "repair_intake.intake.customer_id",
                "targetAttribute": "customer.customer.customer_id"
              }
            ]
          }
        ]
      }
    },
    {
      "op": "add-application",
      "placement": "applications/repair-intake/application.tei",
      "application": {
        "id": "repair_intake",
        "usesSemantic": [
          "repair_intake",
          "customer"
        ],
        "databases": [
          {
            "alias": "customer_master",
            "target": "master_data",
            "access": "read",
            "requires": [
              {
                "contract": "master_data.customer",
                "major": 2
              }
            ]
          }
        ]
      }
    },
    {
      "op": "add-reference-pattern",
      "placement": "applications/repair-intake/patterns/customer-reference.tei",
      "pattern": {
        "id": "repair_intake.customer_reference",
        "application": "repair_intake",
        "relation": "repair_intake.intake.customer",
        "datasource": "customer_master",
        "contractQuery": {
          "contract": "master_data.customer",
          "query": "find_by_id",
          "inputs": [
            {
              "attribute": "customer.customer.customer_id",
              "input": "id"
            }
          ]
        }
      }
    }
  ]
}
JSON
```

With the recorded base revision, normalization must bind this meaning to the already-reviewed Acceptance candidate fingerprint:

```text
sha256:8a936cf96ddeb4faa220285dcef5055d418030373f9425ed8d0d1c2b3e7c9b11
```

The S4 source regenerated from these typed operations must still bind to:

```text
741 bytes
c18ae0fdef8e01aa965e2ad84c86ecfc21ca0608c07b1808e73368a5f434458d
```

## 4. Mandatory re-Preview immediately before accept

```bash
go run ./tools/reference-shared-acceptance preview \
  -workspace reference/workspace \
  -candidate /tmp/tei-second-application-candidate.json \
  -out /tmp/tei-second-application-preview.json
```

Require:

```text
acceptable                    = true
stages                        = 11 / 11 pass
candidateFingerprint          = sha256:8a936cf96ddeb4faa220285dcef5055d418030373f9425ed8d0d1c2b3e7c9b11
baseModelRevision             = sha256:f70da9b64a5b48c7682b24c64f455b0b0c2a7278be4faa297b6ff99c2d3707bf
modelRevisionAfter            = sha256:732763a2b645a6783dd8fbe61b861f2a2c7842f464c1759de846f71eaf18f762
changed subjects              = 6
changed collections           = [applications, semantics]
unresolved                    = ENVIRONMENT_BINDING_NOT_AUTHORED
                                DRIVER_SCENARIO_NOT_RUN
                                TARGET_PROJECTION_NOT_EVALUATED
```

Impact edge count is currently 15. Treat identity / revision / changed-subject / unresolved drift as blocking. A pure future change in presentation of impact ordering/count must be inspected rather than silently accepted.

Confirm again:

```bash
git status --short
```

must remain empty after Preview.

## 5. Actual accept — only after explicit human witness

Do **not** pass `-derived-out reference/world.json`.

```bash
go run ./tools/reference-shared-acceptance accept \
  -workspace reference/workspace \
  -candidate /tmp/tei-second-application-candidate.json \
  -allow-reference-baseline \
  -out /tmp/tei-second-application-accept-decision.json
```

Expected authoritative source additions:

```text
reference/workspace/semantic/repair-intake/intake.tei
reference/workspace/applications/repair-intake/application.tei
reference/workspace/applications/repair-intake/patterns/customer-reference.tei
```

No Environment / Provider / Permission / Persistence Mapping source is authorized by this Acceptance.

## 6. Golden integration condition

Immediately after a successful source accept:

```bash
go run ./tools/reference-golden -write
git diff -- reference/workspace reference/world.json
```

Expected semantic integration:

```text
+ reference/workspace/semantic/repair-intake/intake.tei
+ reference/workspace/applications/repair-intake/application.tei
+ reference/workspace/applications/repair-intake/patterns/customer-reference.tei
~ reference/world.json
```

`reference/workspace/.tei/plugin-lock.json` may be regenerated by the golden tool but should have no semantic/content change for this Proposal. Any unexpected plugin-lock, Environment, database, persistence, migration, release, query, or origin-Application change is blocking.

Then:

```bash
go run ./tools/reference-golden
tools/run-knowledge-portability-reference-evidence.sh
tools/repository-health-check.sh
```

All must PASS before commit / push.

## 7. Failure rollback after source accept

The Acceptance tool itself rolls back source writes when its own post-write Conformance fails.

If a later step in §6 fails, return the working tree to `$PRE_ACCEPT_HEAD` before retrying. The only intended new source paths are the three paths in §5.

A safe explicit rollback is:

```bash
rm -f \
  reference/workspace/semantic/repair-intake/intake.tei \
  reference/workspace/applications/repair-intake/application.tei \
  reference/workspace/applications/repair-intake/patterns/customer-reference.tei

rmdir reference/workspace/applications/repair-intake/patterns 2>/dev/null || true
rmdir reference/workspace/applications/repair-intake 2>/dev/null || true
rmdir reference/workspace/semantic/repair-intake 2>/dev/null || true

git restore --source="$PRE_ACCEPT_HEAD" -- \
  reference/world.json \
  reference/workspace/.tei/plugin-lock.json

git status --short
```

If any additional path is modified, inspect it before cleanup; do not use a broad destructive clean command.

## 8. Commit and branch discipline

After all checks PASS:

1. re-fetch / `git pull --rebase` only if doing so does not mix an uncommitted Acceptance working tree; if main moved, first compare/reconcile deliberately;
2. commit the accepted three source files + matching golden as one semantic integration unit;
3. run the binding repository health check again immediately before push if code/source changed after the previous run;
4. push direct-to-main without force;
5. do not create a new remote work branch;
6. fast-forward any legacy Second Application branch to the resulting main if it still exists.

## 9. Evidence record after successful integration

Append to ISSUE-000040:

```text
proposal fingerprint:
human Acceptance decision:
witness identity / locus:
decision time:
repository revision before accept:
baseModelRevision:
Acceptance candidate fingerprint:
modelRevisionAfter:
accepted source paths:
golden update result:
reference-golden verification:
reference evidence runner:
repository health check:
retained unresolved:
historical Gate B/C scope:
resulting main commit:
```

Historical Gate B / Gate C evidence remains attached to its original exact revisions:

```text
R = e5231839a5dcc62cce1181a2731fcb7c70d16736
T = 0ba87a4e0a96f70a198b03ff7652fb00e3826abe
```

Do not relabel the post-Acceptance current Workspace as the old B/R/T baseline. If a future Target / Gate needs the new Second Application meaning as a baseline, issue a successor baseline / Evidence at that future revision.

## 10. S6 handoff

S5 integration commitが成功した後のnormal Workspace reload / Source Index / golden / shared-authority確認は:

- `plan/SECOND_APPLICATION_S6_SOURCE_RELOAD_VALIDATION_PACKET_2026-09.md`

を正本とする。

S6は別のauthoring経路ではなく、accepted sourceがAcceptance toolやexploration index無しで通常TEI sourceとして自立することを確認するEvidence段階である。
