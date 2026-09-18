# ISSUE-000054: SEI Resume / Reviewを第二ApplicationのReference Adoptionとして圧力試験する

- 状態: In Progress — Change 2 Product behavior probeはGate B / C境界でdeferred
- 提案日: 2026-09-18
- 性格: Reference adoption / cross-application pressure / implementation strategy validation
- 主担当要求: RQ-003 / RQ-004 / RQ-010 / RQ-011 / RQ-021 / RQ-022 / RQ-023
- SEI counterpart: `hat47x/sei-cognition` ISSUE-000079

## このIssueで減らす不確実性

SEI Cognitionは、Work / Interaction、Projection / Review、Evidence / Governance等のApplication PlaneについてTEIを第一候補とする方針を置いています。

本Issueでは、TEIを使うこと自体を成功条件にせず、次を確認します。

> **SEIが所有するDecision Continuityの意味をTEI Coreへ移さず、Resume / Review ProjectionをTEI側でread-onlyに導出すると、Application実装の重複とchange amplificationを減らせるか。**

SUI Reference Adoptionで得た設計知見は利用しますが、SUIのEvidenceをSEIへ流用しません。SEIは、temporal integrity、Recommendation / Decision分離、unresolved保持、Evidence / Governanceという別種の圧力を与える第二Applicationとして扱います。

## Product Requirementとの関係

### RQ-003 — 暗黙の複数正本を作らない

Stage 0ではSEI fixtureをauthorityとし、TEI側copyは固定Reference inputに限定します。TEI側で第二のauthoritative Resume sourceを作りません。

### RQ-004 — Native assetをnativeなまま利用する

SEI独自のDecision Continuity fixtureをTEI利用のためだけに新DSLへ全面変換しません。既存JSONをread-only入力として扱います。

### RQ-010 / RQ-011 — Review Surface / Projection

Decision、Recommendation、historical basis、unresolved、post-decision changeを、source meaningを変えずにProjectionできるかをPressureします。

### RQ-021 — Reference / Conformance

文書だけでなく、小さいfixtureとdeterministic checker/testで意味境界を再現します。

### RQ-022 — 研究実装と製品成立を分ける

Stage 0 checkerのPASSを、Gate C、production Application Plane、SEI human usabilityへ昇格させません。

### RQ-023 — 一人でも維持可能な構造

新しいcross-repo schema、専用runtime、Pattern Kindを増やさず、Reference-local toolで最初の不確実性を減らします。

## Stage 0 source authority

固定source:

- repository: `hat47x/sei-cognition`
- revision: `e943683fbd71ea689bf40fb4c7009531fe98ece5`
- source path: `evaluation/personal-inquiry-decision/decision_continuity_counterfactual_001.json`
- source Git blob: `6354fce0a01093cb5545ed6bb9730aceb3b20023`
- TEI snapshot: `reference/sei-adoption/decision-continuity-counterfactual-001.json`

```text
SEI source
  = authority

TEI reference snapshot
  = frozen experiment input
  != second authority
```

## 入力とoracle

既存fixtureの役割を次に分けます。

```text
scenario
  = Reference input

conditions.C_sei_resume.surface_text
  = expected Projection oracle

reference_state
  = semantic audit oracle
```

既にSEIが生成したResume ProjectionをTEIへの入力にせず、`scenario`からProjectionを導くことで自己循環を避けます。

## Stage 0 checker

`tools/sei-resume-reference`はTEI Coreへ新しい意味を追加しない隔離toolです。

次を決定論的に導出・監査します。

- current question
- Recommendation
- Decision
- Decisionの`basis_refs`からhistorical basis
- Decisionの`unresolved_refs`からunresolved
- `post_decision_evidence`からchanges since decision

fail closed境界:

- Recommendation refがRecommendation以外を指す
- missing ref
- unknown role
- phase_1 / post-decision間のID collision
- strict JSON違反
- Recommendation / Decision混同
- historical basis / unresolved / post-decision idsがoracleと不一致
- post-decision Evidenceがhistorical basisへ漏れる

## Current Evidence

2026-09-18に、Go 1.23.2の隔離環境で次を実行しました。

```text
gofmt -d tools/sei-resume-reference/main.go tools/sei-resume-reference/main_test.go
  -> diff 0 bytes

go test ./tools/sei-resume-reference
  -> PASS

go vet ./tools/sei-resume-reference
  -> PASS
```

これは**Stage 0 Reference mappingのScoped Executable Evidence**です。

### Exact-blob再実行

2026-09-18、current GitHub blobを隔離環境へ再構成し、次を再確認した。

- `tools/sei-resume-reference/main.go`: `a62c0a10ac32d3c84156b74cec57b83f12866dfb`
- `tools/sei-resume-reference/main_test.go`: `7e06b9e289fa00bb19345240146844abad8a5989`
- fixture: `6354fce0a01093cb5545ed6bb9730aceb3b20023`
- `gofmt -d`: 0 bytes
- `go test -v ./tools/sei-resume-reference`: 9/9 PASS
- `go vet ./tools/sei-resume-reference`: PASS
- `go run ./tools/sei-resume-reference`: semantic audit全項目true

machine-readable receiptは `reference/sei-adoption/change1-execution-20260918.json` に保存する。

ただし実行は、上記exact blobとfixtureをminimal temporary `go.mod`へ置いたisolated packageで行った。したがってfull repository verificationやnative SEI側の再実行を主張しない。


```text
Reference checker PASS
  != Gate B / Gate C
  != first Web Target
  != SEI Product Value
  != human usability
  != TEI Application Pattern adoption
  != TEI Core gap established
```

## 最初の観測

current checkerは、新しい`resume` / `review` Pattern Kindを必要としていません。

一方、この段階はTEI Application Pattern / Targetへ実際にProjectionしたEvidenceではなく、Reference-local mappingが成立しただけです。

したがって現時点の分類は次です。

```text
SEI Resume semantic decomposition
  = Reference-local mappingで表現可能

TEI Application / Projection leverage
  = Not Yet Verified

new Pattern Kind
  = Not Justified

TEI Core gap
  = Not Observed
```

## 次の高情報量step

次は機能を増やす前に、同じ意味変更をnative SEI projectionとTEI Referenceで比較します。

第一候補:

> Decision後Observationを一件追加する。

期待する不変条件:

- Decisionは変わらない。
- historical basisは変わらない。
- unresolvedは自動解消されない。
- changes since decisionだけに新Observationが現れる。

比較するもの:

- source owner数
- hand-written mapping / Projection変更
- test変更
- manual synchronization
- semantic duplication
- derived artifact再生成
- authority ambiguity

この比較で負担が減らない、あるいは第二の正本が増えるなら、TEI-first適用範囲を狭めるEvidenceとして扱います。

## Change 1 comparison — 2026-09-18

Decision後Observationを一件追加するchallengeをTEI Reference checkerへ追加した。

TEI側では、fixtureをin-memoryで複製し、

- `scenario.post_decision_evidence`へ新Observationを1件追加
- semantic audit oracleの`reference_state.post_decision_ids`へ同じIDを追加

しただけで、Projection implementationは変更していない。

結果:

```text
gofmt -d
  -> 0 bytes

go test ./tools/sei-resume-reference
  -> PASS

go vet ./tools/sei-resume-reference
  -> PASS
```

testでは次を確認した。

- initial Resumeは完全不変
- historical basisは不変
- unresolvedは不変
- changes since decisionだけが1件増える
- temporal separationを含むsemantic auditはPASS

### native SEIとの比較

SEI current `runtime/go/interactionprobe/review.go`は、Decisionと同時に保存したreview checkpointとcurrent canonical recordsの差から`CanonicalChanges`を導く。

existing `review_test.go`には、Decision後にObservation / Conflict / Recommendationを追加してもhistorical basis / unresolvedを変更せず、後続recordだけを`CanonicalChanges`へ出す同型test sourceが既にある。

このためChange 1に対する実装変更増幅は次のように整理できる。

| 観測 | native SEI | TEI Stage 0 Reference |
|---|---|---|
| authoritative semantic source owner | SEI canonical state | SEI source |
| Projection implementation変更 | 0 | 0 |
| later Observationを扱うための新semantic kind | 0 | 0 |
| historical basis rule変更 | 0 | 0 |
| unresolved rule変更 | 0 | 0 |
| test challenge追加 | existing同型testあり | 1 test追加 |
| cross-repo frozen snapshot同期 | なし | Stage 0では必要 |
| current executable verification | Go 1.25環境で未再実行 | Go 1.23.2 targeted PASS |

したがって、

```text
Change 1
  -> TEIのchange-amplification reductionは未観測
  -> TEI Core gapも未観測
  -> semantic compatibilityは確認
  -> Stage 0 snapshot同期は追加負担
```

である。

これはTEI-first仮説の失敗を意味しない。現在のReference snapshotはauthority migration前の安全な実験境界であり、production pathではない。ただし、この追加同期を無視してTEIの優位を主張してはいけない。

Change 1は両方式にとって容易すぎて差別化力が低いことも分かった。次は、Presentation / Projection concernだけを変更するChallenge 2、またはEvidence / Governance Surfaceへ進む前に、TEI current Gateで実際にPressureできる範囲を再監査する。

## Change 2 gate-aware boundary — 2026-09-18

次の候補だった「Resumeの表示順・progressive disclosureだけを変える」Change 2について、current TEI Gateを再監査した。

current stateは次である。

```text
Gate B = Not Ready
Gate C = Blocked
First Web Target = Not Implemented
complete Target-neutral Presentation IR = Not Established
```

この状態でSEI Reference専用の`display_order`、`resume-layout`、独自Presentation model等を作ると、既存TEI能力のleverageではなく、このReferenceのために新設したsurrogate infrastructureを評価することになる。

そのためChange 2の**Product behavior probeはdefer**する。これはTEIが失敗したという判定ではなく、比較の妥当性を守る停止線である。

再開条件は次のいずれかとする。

1. Gate B Ready後、Handover Baseline `B` / revision `R`が固定され、First Web Target workへ正式に進む。
2. またはSEIとは独立した根拠でtarget-neutral Presentation機構が成立し、SEI固有概念を追加せずPressureできる。

machine-readable boundaryは`reference/sei-adoption/change2-presentation-gate-boundary-20260918.json`へ保存する。

```text
Change 2 deferred
  != TEI failure

custom Reference presentation helper PASS
  != TEI Presentation leverage

current action
  = do not invent surrogate layer
```

この間に進められるのは、Target behaviorを要求しないEvidence / Governance Surfaceのmapping等に限る。

## Challenge B mapping — Evidence / Governance

Target behaviorを要求しない範囲で、SEI Evidence bundleとTEI current Evidence / Provenance境界を対応付けた。

結果は、すべてをTEIへ寄せる構造ではない。

### Existing TEIとして再利用できる境界

- Evidence artifactをCanonical authorityにしない
- subject / revision / build / Scenario / contextへのtrace
- exact artifact fingerprint / retained-file integrity
- source provenance / processing
- secret-safe Evidence採取
- deterministic / synthetic / external / interactive / production等のEvidence natureを相互代替しない

### Native SEI Governanceとして残す境界

- human reviewer observationとsemantic auditの意味分離
- review開始時の条件固定、reference reveal順序
- reviewer authorship / prior exposure declaration
- SEI固有のEvidence高さの解釈
- Product Value / Stable Requirementへの昇格可否

TEIは将来これらを表示・traceできても、意味authorityを取得しない。

Review / Diff VisualizationはSUIとSEIの双方で需要が見えるためreusable extension候補ではあるが、Gate C Target Evidenceがない現時点ではcandidateに留める。

machine-readable mappingは`reference/sei-adoption/challenge-b-evidence-governance-mapping-20260918.json`へ保存する。

```text
generic Evidence integrity / provenance
  -> Existing TEI

SEI review semantics / evidence-height authority
  -> Native SEI Governance

interactive Review / Diff Surface
  -> Deferred / candidate only
```

## 非目標

- SEIのdomain vocabularyをTEI Coreへ取り込む
- `resume` / `review` Pattern Kindを先に追加する
- SEI semantic authorityをTEIへ移す
- Gate C前にproduction UIを作る
- SUI ReferenceのPASSをSEI Evidenceへ流用する
- 一件のReferenceからTEIを全Applicationの必須基盤とする
- human usabilityをchecker testから推定する

## 完了条件

- [x] SEI source authorityと固定snapshotを明示する
- [x] scenario / Projection oracle / semantic audit oracleを分離する
- [x] Reference-local checkerを実装する
- [x] Recommendation / Decision、historical basis、unresolved、post-decision temporal boundaryをtestする
- [x] Go 1.23.2でfixture-local `go test`を実行する
- [x] `go vet`を実行する
- [x] representative Change 1についてnative / TEI Referenceのchange amplificationを比較する
- [x] Change 1とChange 2 Gate監査から、Existing / reusable extension / Native / Product gap candidateを再分類する
- [x] SEI側ISSUE-000079へChange 1のEvidence高さを返す
- [x] broader adoption判断に必要な次EvidenceとChange 2再開条件を明示する

## Natural Japanese review

Stage 0では、SEIのResume BriefをTEI固有形式へ翻訳すること自体を目的にしていません。既存SEI fixtureをそのまま保持し、source意味からResume concernを導く小さいReferenceを作ることで、TEIのApplication Planeとしての適性を段階的に検証しています。

最初のcheckerが通ったことから分かるのは、Recommendation / Decision、historical basis、unresolved、later changeというSEIの意味差を、TEI Coreへ新しいkindを追加せずReference-localに保持できることまでです。Application PatternやTargetを使った実装負荷低減はまだ未検証です。

したがって、次に新しいTEI primitiveを追加するのではなく、同一意味変更のchange amplificationを比較し、TEIを使うことで本当にowner・同期・重複が減るかを先に確かめます。
