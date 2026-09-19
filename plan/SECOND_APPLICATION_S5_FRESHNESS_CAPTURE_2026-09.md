# Second Application Reference — S5 Freshness / Base Capture

- Status: Revised Proposal freshness captured / executable pressure and Acceptance witness pending
- Date: 2026-09-20
- Primary workstream: ISSUE-000040
- Parent: `plan/SECOND_APPLICATION_S5_SHARED_ACCEPTANCE_REVIEW_PACKET_2026-09.md`
- Proposal SHA-256: `c18ae0fdef8e01aa965e2ad84c86ecfc21ca0608c07b1808e73368a5f434458d`
- Superseded Proposal SHA-256: `b347b41b75c9864e66b4b191eefc3d668c0d8bf3dfdf186d0733f794159467c2`
- Review base Repository revision: `0dd20cf9100823b22499473d26c7caad7881a9dc`
- Shared Acceptance result: Not established
- Acceptance witness: Absent

## 1. 目的

revised three-source Proposalが参照するshared authorityとorigin provenanceをcurrent mainで再解決し、S5 review用のfreshness baseを固定する。

このcaptureはAcceptanceでもExecutable Validationでもない。

```text
freshness captured
  != validation passed
  != accepted
```

## 2. Review base

今回実際にshared authority / provenanceとValidation implementationをinspectしたrevision:

```text
0dd20cf9100823b22499473d26c7caad7881a9dc
```

このrevisionで次のGit blob identityを確認した。

```text
customer semantic source:
  c348122b33581e450974d23393d98fb8aa838388

customer contract source:
  c1eea63b053fb936ae7aebbac82180af015b42e1

origin customer-reference pattern:
  252f6396cbc2394d1f26e748b01d54f49a2e1ecf

application-pattern validation:
  e137189a5945efaec0c14d82f92c62ba98fa16fd

reference contract-query cross validation:
  8c2a4b5bd89baf3c4562d62bfed009ce3094bf40
```

後続の無関係なcommitだけでこのcaptureを自動staleにしない。Acceptance時にはrelevant source identityを再確認する。

## 3. Exact shared-authority Source Index fingerprints

### Customer Semantic / Customer ID Type

```text
source:
  semantic/customer/customer.tei

subjects:
  semantic-data:customer.customer
  semantic-type:customer.customer_id

fingerprint:
  sha256:d0897ed9ac8ce7e01200b05285e445fa57fa402abd0725542c7cf7ebfa0ad23b
```

### Customer Contract / Contract Query

```text
source:
  databases/master-data/contracts/customer.tei

subjects:
  database-contract:master_data.customer
  database-contract-query:master_data.customer/find_by_id

fingerprint:
  sha256:bfbc0ed04d258fa05e254dc6b0c6a9779069e98d4f93511d0f062631ddb5fc15
```

### Origin Pattern provenance

```text
source:
  applications/order-entry/patterns/customer-reference.tei

subject:
  application-pattern:order_entry.customer_reference

fingerprint:
  sha256:ae17bb1078c0a27ceeb20a5ece20431d98bee403c28fe7bc468643691263ef21
```

origin Patternはprovenanceでありdestination authorityではない。

## 4. Requalification result

```text
semantic-data:customer.customer
  qualification = applicable
  reuse = linked-reuse

semantic-type:customer.customer_id
  qualification = applicable
  reuse = linked-reuse

database-contract:master_data.customer
  qualification = applicable
  reuse = conditional-linked-reuse

database-contract-query:master_data.customer/find_by_id
  qualification = applicable
  reuse = conditional-linked-reuse

application-pattern:order_entry.customer_reference
  qualification = applicable-as-provenance
  reuse = adopted-local-variation
```

shared source fingerprintが変わった場合はsilent applyせず、S1/S2 requalificationへ戻る。

## 5. Revised Proposal freshness

current exact Proposal:

```text
semantic source
  + application source
  + pattern source
  = c18ae0fdef8e01aa965e2ad84c86ecfc21ca0608c07b1808e73368a5f434458d
```

旧two-source fingerprintはsuperseded済みであり、Acceptance対象にしない。

## 6. Validation source-inspection result

current implementation inspectionで確認した境界:

- reference Patternは実在Semantic Relationを要求する
- Relation keyはsource / target attributeを明示する
- key両端は同じSemantic type identityを要求する
- Contract Query resultはRelation targetと一致する
- Relation `one` とQuery `zero-or-one` は互換

revised Proposalはこれらを満たすよう構成した。

ただしこれはimplementation source inspectionであり、actual loader / validator executionではない。

## 7. Acceptance entry state

```text
proposalFingerprint       = captured
reviewBase                = captured
shared freshness          = captured
source-inspection preflight = complete
revised executable pressure = absent
acceptanceWitness         = absent
decision                  = pending
```

したがって:

```text
S5 freshness = current for inspected inputs
Shared Acceptance eligibility = blocked
S6 source authoring = blocked
```

## 8. Drift policy

Acceptance前にmainが進んだ場合:

1. current main revisionを確認する
2. revised Proposal component bytesが変化していないか確認する
3. shared authority / provenance fingerprintを再確認する
4. Validation contractに関係するsource変更があればpreflightを再評価する
5. relevant input不変なら無関係なcommitだけでstale-allしない
6. relevant input変更ならrequalification / review regenerationへ戻る

## 9. Evidence height

この文書はSource / Design Evidenceである。

revised Proposal向けGo pressureとfull repository health checkのactual PASSはまだ取得していない。

## 10. Conclusion

revised Proposalのshared authority / provenance / validation preflight inputsはcurrent mainに対してcaptureできた。

残るpre-Acceptance条件は:

```text
health-checked revised executable pressure
explicit Acceptance witness
```

である。