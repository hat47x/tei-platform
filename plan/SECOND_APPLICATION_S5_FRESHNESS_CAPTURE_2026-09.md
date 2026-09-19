# Second Application Reference — S5 Freshness / Base Capture

- Status: Freshness captured / explicit Acceptance witness pending
- Date: 2026-09-19
- Primary workstream: ISSUE-000040
- Parent: `plan/SECOND_APPLICATION_S5_SHARED_ACCEPTANCE_REVIEW_PACKET_2026-09.md`
- Proposal SHA-256: `b347b41b75c9864e66b4b191eefc3d668c0d8bf3dfdf186d0733f794159467c2`
- Review base Repository revision: `e2856e6e568b6be966c6b8689ebea479ec184a4a`
- Shared Acceptance result: Pending
- Acceptance witness: Absent

## 1. 目的

S5 Shared Acceptanceの前提として、Proposalが参照するshared authorityとorigin provenanceをcurrent mainで再解決し、review baseを明示的に固定する。

このcaptureはAcceptanceそのものではない。

```text
freshness captured
  != accepted

repository revision captured
  != runtime authority
```

である。

## 2. Review base

2026-09-19、S5 Proposal merge後のcurrent mainを次で固定した。

```text
reviewBaseRepositoryRevision =
  e2856e6e568b6be966c6b8689ebea479ec184a4a
```

S5 Proposalをmainへ統合したrevision:

```text
4d4f8acc39a569d0f99c7fc617de30bf11e455f8
```

前回capture後にはR3B authority boundary、solo-development flow、repository health check、Issue番号整理、Gate A runner等の変更が入った。またS4/S5 test-only sourceのcompile / gofmt修正も入った。しかしSecond Applicationが参照する以下のCanonical source blobは今回のreview base `e2856e6e568b6be966c6b8689ebea479ec184a4a` でも不変だった。

- `reference/workspace/semantic/customer/customer.tei`
- `reference/workspace/databases/master-data/contracts/customer.tei`
- `reference/workspace/applications/order-entry/patterns/customer-reference.tei`

これは「main上の全変更が無関係なら永久にfresh」という規則ではない。Acceptance時点でbaseが進んでいる場合は再captureする。

## 3. Exact authority source fingerprints

Source Indexのfingerprint contractはraw source bytesのSHA-256である。

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

origin Patternはprovenanceであり、destination linked authorityではない。

## 4. Requalification result

current RepositoryからS1/S2 qualificationを再導出した場合、Acceptance前に必要な期待状態は次。

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

source fingerprintが上記captureから変化した場合、silent applyしない。

## 5. Proposal freshness

S4 Application + Pattern sourceのexact Proposal fingerprintは引き続き:

```text
b347b41b75c9864e66b4b191eefc3d668c0d8bf3dfdf186d0733f794159467c2
```

である。

freshness captureはProposal本文を変更しない。

```text
same Proposal fingerprint
+ same shared authority fingerprints
+ explicit review base
  -> freshness evidence complete

freshness evidence complete
  != Shared Acceptance
```

## 6. Acceptance entry state

このcaptureにより、S5の二つの前提のうちbase/freshness側は満たせる。

現在:

```text
proposalFingerprint = captured
reviewBase          = captured
shared freshness    = captured
acceptanceWitness   = absent
decision            = pending
```

したがって:

```text
S5 freshness = complete
Shared Acceptance = blocked-missing-witness
S6 source authoring = blocked
```

である。

## 7. Drift policy

explicit Acceptance witnessを取得する前にmainが進んだ場合:

1. current main revisionを確認する
2. S5 Proposal自体が変化していないことを確認する
3. shared authority / provenance source fingerprintを再導出する
4. relevant fingerprintが同じなら、後続の無関係なcommitだけを理由にProposalをstaleへしない
5. relevant fingerprintが変わったらS1/S2 requalificationとS4/S5 reviewをやり直す
6. Acceptance witness取得時には、その時点のcurrent mainと最後に検査したbase revisionの関係を記録する

`reviewBaseRepositoryRevision`は「該当Sourceを実際に検査したrevision」であり、この文書自身のcommitなど無関係な後続commitによって自己無効化しない。commit数やpath名だけでfreshnessを判断しない。

## 8. Test-only Source pressure

- `tools/reference-business-procedure-check/second_application_s5_freshness_capture_pressure_test.go`

確認すること:

1. current S1/S2 packetが引き続きapplicable
2. shared subjectsがexact source pathへ解決する
3. exact source SHA-256がcaptureと一致する
4. origin Patternはprovenanceのままである
5. S4 Proposal fingerprintが変化していない
6. freshness completeでもAcceptance witness absentならS6へ進めない

## 9. Evidence height

この文書とtest-only sourceはSource / Design Evidenceである。

focused testのactual PASSが得られた場合のみExecutable Evidenceへ昇格する。

GitHub Actions runner availabilityとは別に、portable runner:

- `tools/run-knowledge-portability-reference-evidence.sh`

で同じtest prefixを実行可能にする。

## 10. Conclusion

S5 Proposalに対するcurrent base / freshnessは、revisionとexact source fingerprintまで固定できた。

残るShared Acceptanceの前提はexplicit Acceptance witnessである。

それを取得するまではS6 authoritative source authoringを開始しない。
