# Business Procedure Second-Corpus Handoff Inference Pressure — 2026-09

Status: **Lane C / test-only negative pressure / no production schema adoption**

## 1. 目的

manufacturing / inventory second-corpus outlineでは、physical completionやstock availabilityがresponsibility continuationの前提になり得る。

しかし次を区別する必要がある。

```text
Business fact observed
  != responsibility transferred

Procedure Step adjacency
  != accepted Handoff
```

本Pressureは、H1/R3をscreen / Step順 / Observationから推測せず、explicit Business Handoff authorityだけからnormalized relationを作ることを固定する。

## 2. Current H1/R3 derivation

Current test-only normalized Business modelはHandoffを次からだけ構成する。

```text
referenceSet.Handoffs[]
  -> From { Procedure, Step }
  -> To   { Procedure, Step }
  -> Condition
```

Procedure body、Step adjacency、Application Pattern、Observation resultはHandoff generation inputではない。

これはcurrent implementation detailとして偶然残すのではなく、authority directionとしてPressureする。

## 3. H-INF-01 — adjacency does not author Handoff

Primary ProcedureではBP-08がBP-09より前にある。

Explicit Handoff sourceを除去してもProcedure bodyとStep orderは残る。

期待:

```text
ordered steps exist
BP-08 before BP-09
explicit Handoff source absent
  -> normalized Handoffs = 0
```

Step隣接からresponsibility transferを推測しない。

## 4. H-INF-02 — positive physical Observation does not author Handoff

Second-corpus test-local external Observationがpositiveでも:

```text
physical fact observed = true
explicit Handoff source absent
  -> normalized Handoffs = 0
```

とする。

ObservationはBusiness continuationの前提事実を確認できても、誰から誰へ、どのStepからどのStepへ、どのconditionでresponsibilityが移るかを所有しない。

## 5. H-INF-03 — explicit relation is required

同じProcedure fixturesで:

```text
referenceSet.Handoffs = []
  -> normalized Handoffs = 0

referenceSet.Handoffs = [explicit relation]
  -> normalized Handoffs = 1
  -> exact endpoint + condition validation
```

をPressureする。

したがってH1/R3 relationのauthority directionは:

```text
explicit accepted Handoff source
  -> normalized H1/R3 relation

Step order / adjacency
  -X-> Handoff

physical Observation
  -X-> Handoff

Application Function sequence
  -X-> Handoff
```

である。

## 6. Business Flowとの境界

この結果はH1/R3がBusiness Flow全体を表すことを意味しない。

H1/R3は依然として:

```text
Procedure/Step -> Procedure/Step + condition
```

だけを扱う。

material / document / information movementやexternal participantが必要になれば、T-GAP-06を別pressureとしてreopenする。

## 7. X-TEA fallbackとの整合

X-TEAではBusiness Model / Business Flow / Function利用が一つのruntime sequenceへ潰されていない。

TEIでも、Application操作順や観測結果からBusiness責任移管を自動推定せず、Business側でacceptされたrelationを独立して保持する方が三視点の相互批判性を保つ。

ただしこのEvidenceはproduction `BusinessHandoff` Canonical kindの採用を証明しない。

## 8. 現時点の判断

```text
H1/R3 explicit authority direction
  test-onlyで明確化

adjacency inference
  reject

Observation inference
  reject

production Handoff Canonical kind
  still unproven

Gate 15c
  not activated
```

H1/R3は引き続きnarrow relation pressureとして維持する。

## 9. Verification status

Repository上へ`second_corpus_handoff_inference_pressure_test.go`を追加したがactual runner greenは未取得である。

`gofmt / go vet / go test / checker` PASSはまだ主張しない。
