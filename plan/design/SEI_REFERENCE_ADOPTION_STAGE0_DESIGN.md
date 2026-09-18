# SEI Reference Adoption Stage 0 — Resume / Review Projection設計

- 状態: Reference Design + fixture-local Executable Evidence
- 更新日: 2026-09-18
- Parent Issue: `plan/issue/000054-sei-reference-adoption-resume-review.md`
- SEI counterpart: `hat47x/sei-cognition` ISSUE-000079
- Gate impact: None
- Product Requirement impact: None in this document

## 1. 目的

SEI Cognitionは、Application PlaneをTEI-firstで実装できるかを検討しています。

このStage 0では、SEIのProduct / Semantic authorityをTEIへ移さず、既存Decision Continuity fixtureからResume / Review Projectionをread-onlyに導出し、TEI Application基盤へ接続する前段の意味分解が成立するかを確認します。

成功条件は、SEIの画面をTEIで再現することではありません。

```text
SEI source meaning
  -> deterministic Reference projection
  -> later TEI Application / Projection pressure

SEI source meaning
  != TEI-owned meaning
```

## 2. なぜStage 0をReference-localにするか

TEI current Application Pattern v1には`browse`、`record`、`reference`、`command`、`header-detail`があります。

SEI Resumeは、一つのProjectionに、

- current question
- Recommendation
- Decision
- historical basis
- unresolved
- post-decision change

を異なる意味役割として束ねます。

現時点でこれを`record`や`browse`へ押し込むと、既存Patternの意味をSEI都合で拡張する可能性があります。

逆に、SEIが必要とするという理由だけで`resume`や`review` Pattern Kindを追加するのも早すぎます。

そこで最初に、

1. SEI sourceを変えずに意味差を導出できるか
2. Reference-localなmappingで不足なく扱えるか
3. どの差だけがApplication / Projectionへ持ち上がるか

を確認します。

## 3. 固定source

TEI repositoryには、SEI fixtureの固定snapshotをReference inputとして置きます。

- source repository: `hat47x/sei-cognition`
- source revision: `e943683fbd71ea689bf40fb4c7009531fe98ece5`
- source path: `evaluation/personal-inquiry-decision/decision_continuity_counterfactual_001.json`
- source Git blob: `6354fce0a01093cb5545ed6bb9730aceb3b20023`
- TEI snapshot: `reference/sei-adoption/decision-continuity-counterfactual-001.json`

TEI snapshotは固定実験入力であり、第二正本ではありません。TEI側snapshot自身のGit blobも`6354fce0a01093cb5545ed6bb9730aceb3b20023`で、固定SEI sourceと同一bytesである。

### 3.1 pinned baselineとcurrent sourceを分ける

このReferenceのauthorityはbranch名ではなく、上記の**pinned revision / source blob**へ束縛する。

```text
pinned revision/blob conformance
  != current SEI main conformance

branch reachability
  != semantic authority
```

2026-09-18のcross-repository再確認では、

- SEI current `main`: `a485a658ee622433990dcb7323859a70f4be6f49`
- pinned `e943683f...`はcurrent `main`の254 commit後の子孫
- current `main`には固定fixture pathが存在しない
- `develop/v0.2.0`、ISSUE-000073 runtime branch、一部evaluation branchはpinned revisionを祖先として保持
- `develop/v0.2.0`上のfixture blobは固定blobと同一

を確認した。

この状態を理由にcurrent `main`を「古い」「誤り」と評価しない。Stage 0 reportは`authority_scope = pinned-revision-only`、`current_source_conformance = not-claimed`として、固定実験baselineとcurrent Product sourceを混同しない。

将来current sourceへの適合を主張する場合は、その時点のactual dependency blobを別途固定・比較する。repository-wide revision変化だけでStage 0をstaleにするglobal freshnessへは戻さない。

## 4. input / oracle分離

fixture内の情報をすべて入力にはしません。

```text
scenario
  -> projection input

conditions.C_sei_resume.surface_text
  -> expected Projection oracle

reference_state
  -> semantic audit oracle
```

この分離により、既にSEI側で作成されたResume ProjectionをTEIが読み直すだけの自己循環を避けます。

### 4.1 scenarioから導くもの

Decisionの参照をsourceとして、次を導きます。

```text
initial_question
  -> current question

decision.recommendation_refs
  -> Recommendation

decision
  -> Decision

decision.basis_refs
  -> historical basis

decision.unresolved_refs
  -> unresolved

post_decision_evidence
  -> changes since decision
```

### 4.2 oracleの用途

`C_sei_resume`からは、RecommendationとDecisionの期待ID、およびinitial Resumeではchanges since decisionが空であることを監査します。

`reference_state`からは、historical basis、unresolved、post-decision IDsを監査します。

oracleに含まれる自然言語summaryを、source意味の代わりに入力へ戻しません。

## 5. temporal integrity

最優先の不変条件です。

```text
Decision-time basis
  != later Observation

post-decision Observation
  -> changes since decision
  != historical basis
```

fixtureではProvider Bの429とProvider Aのmaintenance公表がDecision後に追加されます。

Referenceが短く読みやすくても、これらをDecision時点のbasisへ混ぜれば失敗です。

## 6. tool責務

`tools/sei-resume-reference`は、Reference-local checkerです。

責務:

- strict JSON decode
- fixture schema / evaluation-only status確認
- item roleとIDの検査
- phase_1内のref解決
- Recommendation / Decision分離
- historical basis / unresolved導出
- post-decision分離
- oracleとのsemantic audit
- fail closed

非責務:

- TEI Canonical WorldへSEI domain kindを追加する
- Application Patternを追加する
- Target UIを生成する
- SEI sourceを書き換える
- human usabilityを評価する
- Gate B / Gate Cを判定する

## 7. fail-closed boundary

少なくとも次をrejectします。

- unknown top-level JSON field
- unsupported role
- missing referenced item
- Recommendation refがRecommendation以外を指す
- duplicate / temporal collision ID
- multiple / missing Decision
- oracleとDecision / Recommendationが不一致
- basis / unresolved / post-decision IDsがreference_stateと不一致
- later Evidenceがhistorical basisへ漏れる

未知や解決不能なrefを黙って省略し、完全なResumeに見せません。

## 8. current Executable Evidence

2026-09-18、Go 1.23.2の隔離環境で次を実行しました。

```text
gofmt -d tools/sei-resume-reference/main.go tools/sei-resume-reference/main_test.go
  -> 0 bytes diff

go test ./tools/sei-resume-reference
  -> PASS

go vet ./tools/sei-resume-reference
  -> PASS
```

この結果はReference-local mappingのScoped Executable Evidenceです。

```text
checker/test PASS
  != TEI Application Pattern adoption
  != Target implementation
  != Gate C
  != SEI Product Value
  != human outcome
```

## 9. Stage 0から得た初期観測

Reference-local mappingでは、SEI Resumeの意味差を保持するためにTEI Coreや新Pattern Kindを追加する必要はありませんでした。

これは「TEI Application Planeへそのまま実装できる」と証明したわけではありません。

current conclusion:

| 問い | Current state |
|---|---|
| SEI fixtureを第三schemaへ翻訳せず読めるか | Yes — Reference-local |
| Recommendation / Decisionを分離できるか | Yes — Scoped Executable Evidence |
| historical basis / later changeを分離できるか | Yes — Scoped Executable Evidence |
| unresolvedを保持できるか | Yes — Scoped Executable Evidence |
| new Pattern Kindが必要か | Not justified |
| TEI Core gapがあるか | Not observed |
| TEIでApplication開発負荷が下がるか | Not Yet Verified |
| SUIと共通Pluginが必要か | Candidate only |

## 10. 次の比較 — Change 1

最初のchange amplification比較は、Decision後Observationの追加です。

### source change

SEI fixtureの`post_decision_evidence`へ一件追加する。

### invariants

- current Decisionは変わらない
- historical basisは変わらない
- unresolvedは自動解決されない
- changes since decisionだけが増える

### 比較

native SEI ProjectionとTEI Referenceについて、次を記録します。

- source owner数
- hand-written mapping変更
- Projection / config変更
- test変更
- manual synchronization
- duplicated semantics
- derived artifact再生成
- authority ambiguity

TEI側で追加code/config変更なしにderived Projectionへ追随できるなら、structure-to-application leverageの最初のEvidenceになります。

反対に、同じ一件のObservation追加のために複数TEI assetを手動同期する必要があるなら、TEI-firstを狭めるEvidenceです。

## 11. SUIとのcross-application pressure

SUI Reference Adoptionはspatial canvas、hierarchy、proposal / merge reviewをPressureしています。

SEIはtemporal integrity、Decision memory、Evidence / GovernanceをPressureします。

両方からReview / Diff Visualizationの需要が再現した場合、reusable extension候補として強くなります。

ただし二Applicationで使えただけでCore primitiveにはしません。Target component、Plugin、Adapterで意味追加なしに扱えるかを先に比較します。

## 12. 次段への条件

TEI Application / Projectionへの本格接続へ進む前に、

1. Change 1のchange amplification比較を取得する
2. native SEI側とのowner / sync差を記録する
3. 既存Application / Projection機構で何が使えるかを具体化する
4. 新Pattern Kindなしで進められるか再確認する
5. SUIとの共通extension候補を必要なら比較する

ことを優先します。

Stage 0のcheckerが通ったことだけを理由に、Gate C前のWeb TargetやShared Acceptanceをproduction-readyとして利用しません。

## 13. Natural Japanese review

このReferenceはSEIをTEIへ「移した」ものではありません。

SEIの意味を固定fixtureとして借り、Recommendation / Decision、historical basis、unresolved、later changeという差を壊さずにApplication Projectionへ持ち上げる前段が、現在のTEI Coreを増やさなくても成立するかを確かめています。

最初の結果は肯定的ですが、そのEvidenceはReference-localです。TEIのProduct ValueであるApplication開発負荷の低減を判断するには、同じ意味変更をnative実装とTEI Referenceへ与えたときのchange amplificationを比較する必要があります。

したがって、次の作業は新しいPatternやPluginの実装ではなく、Decision後Observationを一件追加する小さいchange challengeです。
