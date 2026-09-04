# ISSUE-000007: 第一Web TargetへのProjection境界を設計する

- 状態: 提案中
- GitHub Issue: #70
- 本格着手条件: Gate B Ready + Handover Baseline固定
- 関連RQ: RQ-009 / RQ-010 / RQ-011 / RQ-012 / RQ-013 / RQ-014 / RQ-017 / RQ-018 / RQ-021 / RQ-022 / RQ-024

## 背景

TEIでは、最初からWeb / Desktop / Mobile / API / MCP / VUIを一つの巨大なPresentation IRへ押し込みません。

まず`order_entry` ApplicationをApplication DriverでUIなしに成立させ、Gate Bを通過した同じ意味を第一TargetであるWebへProjectionします。その過程から、Application上の意味、複数Targetで共有できそうなPresentation / Interaction候補、Web固有実装の境界を抽出します。

Gate CではWebとDriverが同じCanonical Application上の意味を共有していることをSemantic Conformanceとして確認します。

## Gate Bから受け取るbaseline

第一Web Targetは、曖昧な「その時点の最新Driver」から開始しません。

[`GATE_B_TO_C_HANDOVER_CONTRACT.md`](../GATE_B_TO_C_HANDOVER_CONTRACT.md) / `ISSUE-000030` / GitHub Issue #291に従い、Gate B通過時のHandover Baselineを受け取ります。

少なくとも次へ追跡できる必要があります。

- repository revision
- Application: `order_entry`
- browse / record / reference / header-detail / command
- Data / Relation / Query / Datasource
- Validation
- Authentication / Authorization
- Capability / Binding
- Business Result / named Outcome
- Execution Error
- Observationで利用する明示read境界
- Driver Scenario identity
- Gate B Evidence
- unresolved

Handover recordは第二の正本ではありません。authoritative source / Canonical ID / Evidenceへの参照を束ねます。

Gate C進行中にApplication / Driverの意味を変更する必要が生じた場合、旧baselineへ黙って上書きしません。Driver側へ戻って必要なEvidenceを再確認し、新しいHandover Baselineを発行します。

## 目的

Web画面を作ること自体ではなく、**Gate Bを通過したApplication / Driver baselineをWebへ落とすために本当に必要なPresentation / Interaction上の意味を、具体的なReferenceから抽出すること**を目的とします。

Web専用のApplication modelやresult modelを作りません。

## 対象Scenario

少なくとも次をHandover BaselineのDriver Scenarioと対応付けます。

- browse
- record open
- reference lookup
- header-detail read / edit
- Validation satisfied / not satisfied
- Authorization allowed / denied
- command invocation
- Business Result
- Execution Error
- Observation

Command acknowledgementだけでObserved state成立とはみなしません。

## 最初に観測するPresentation / Interaction候補

具体的なUI部品名をCoreへ固定する前に、少なくとも次がどの層に属するかをReferenceから観測します。

- collection / item
- read / edit
- field grouping
- relation / reference selection
- detail collection
- validation feedback
- command invocation
- authorizationによる利用可否
- Business Result / named Outcomeのfeedback
- Execution Errorのfeedback
- observed completion / observed state
- read-only / unavailable
- navigation / focus target

これらをすべて共通IRへ入れるとは限りません。

各差を次へ分類します。

1. Application上の意味
2. Target-neutral Presentation候補
3. Target Capability候補
4. Runtime Binding差
5. Execution / Trust差
6. Web固有実装
7. Host control plane固有差
8. Canonicalへ持ち上げてはいけない差
9. unresolved

分類できない差を無理に共通概念へ押し込みません。

## Web Adapterの原則

Web固有実装はCoreへ入れず、独立したAdapter / Packageとして組み立てます。

- Web frameworkをCanonical meaningへ持ち込まない
- native SQLをWeb側へ複製しない
- Capability implementationをWeb側へ複製しない
- Validation / AuthorizationをWeb固有ruleとして再実装しない
- read-only DatasourceをTarget都合でwrite可能にしない
- HTTP statusやUI messageからSemantic Outcomeを逆定義しない
- `output` valueをWeb都合の暗黙`success` Outcomeへ変換しない
- Web Adapterを外してもCanonical Worldが変わらない

Web側に必要な意味がHandover Baselineに存在しない場合、Web固有fieldやruleで埋めてGate Cを通しません。

## Business Result / Execution Error / Observation

Webではtoast、inline message、dialog、画面遷移、HTTP response等の表現を選べます。しかし表現と意味を同一視しません。

```text
Business Result
  ├─ Output value
  └─ Named Outcome(name, payload)

Execution Error
  └─ Capability呼び出しそのものを完了できなかった状態

Observation
  └─ 明示read境界から観測した外部状態
```

Observationが必要な処理では、Command Business Resultだけを最終的な外部状態成立へ畳みません。

また次を一種類のWeb errorへまとめません。

- Validation feedback
- Authentication / Authorization等のBusiness Outcome
- Execution Error
- Observation上の未成立 / 不明
- developer Diagnostic

## Custom UI

標準Patternから生成できる画面だけをTEIの世界とはしません。

第一Web Targetから少なくとも一箇所でCustom UIを接続し、標準ProjectionとCustom implementationが同じApplication / Validation / Authorization / Capability / Result境界へ接続できることを確認します。

Custom UIを新しい業務正本にはしません。

## Gate C Evidence

[`GATE_C_EVIDENCE_CONTRACT.md`](../GATE_C_EVIDENCE_CONTRACT.md) / `ISSUE-000028` / GitHub Issue #270に従い、少なくとも次を別Evidenceとして取得します。

1. **Semantic Conformance**
2. **Interaction / Functional Evidence**
3. **Visual Fidelity Evidence**

各Evidence runからHandover Baseline identityへ戻れる必要があります。

browser automation、GUI agent、VLM、screenshot diff等は利用できますが、Visual Evidenceや外部judge scoreだけでSemantic / Functional Conformanceを合格にしません。

## Performance

RQ-024の最終性能contractは第二Target以降で比較して決めますが、第一Web Targetから次の測定点を用意します。

- initial projection / load
- browse result render
- record open
- reference lookup
- validation
- authorization
- command response
- Business Result projection
- Execution Error projection
- observed completion

第一Targetだけから根拠のない一律閾値を固定しません。

## 第二Targetへの引き継ぎ

Gate Cで得たTarget差を`ISSUE-000026` / GitHub Issue #259へ渡し、第二Target / Hostから圧力試験します。

第一TargetだけでTarget-neutral設計が完成したとはみなしません。第二Targetで境界修正が必要になった場合も、それ自体を失敗とはせず、第一Targetの都合を共通意味へ持ち上げていなかったかを検証するEvidenceとして扱います。

## 完了条件

- [ ] Gate B Ready後、`ISSUE-000030`の第一Handover Baselineを受け取る
- [ ] `order_entry`の主要Patternを同じCanonical meaningからWebへProjectionする
- [ ] Validation / Authorization / Capability / Result / ObservationをWeb側で再定義しない
- [ ] 各Gate C runからHandover Baselineへ追跡できる
- [ ] Semantic ConformanceをEvidenceから確認できる
- [ ] Reference ScenarioのInteraction / Functional Evidenceを再現できる
- [ ] Visual Fidelityを別Evidenceとして残せる
- [ ] standard ProjectionとCustom UIを同じ上位意味へ接続できる
- [ ] Web Adapterを外してもCanonical Worldが変わらない
- [ ] Target Capability / Execution / Trust差を`ISSUE-000026`へ渡す
- [ ] Gate CをIssue close状態ではなくEvidenceから判定できる

## 非目標

- Gate B前の本格Web実装
- Web frameworkの先行固定
- 完全なPresentation IRの先行設計
- 第一TargetだけでTarget Capability taxonomyを完成させること
- Webの不足からCanonicalを逆設計すること
- pixel-perfect design systemの完成

## 関連

- `plan/GATE_B_TO_C_HANDOVER_CONTRACT.md`
- `plan/GATE_C_EVIDENCE_CONTRACT.md`
- `plan/issue/000028-first-web-target-conformance-evidence.md`
- `plan/issue/000030-gate-b-to-c-handover-baseline.md`
- GitHub Issue #70 / #259 / #270 / #291
