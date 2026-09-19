# ISSUE-000007: 第一Web TargetへのProjection境界を設計する

- 状態: In Progress
- GitHub Issue: #70
- 本格着手条件: Gate B Ready + Handover Baseline固定
- 関連RQ: RQ-009 / RQ-010 / RQ-011 / RQ-012 / RQ-013 / RQ-014 / RQ-017 / RQ-018 / RQ-021 / RQ-022 / RQ-024
- Product Acceptance Scenario: PAS-04 primary

## 着手状況 — standard / Custom UI shared Command boundary

Gate B Ready / Handover Baseline `B / R`固定を受け、第一incrementを開始した。

Reference-specific implementation:

- `tools/reference-first-web-target/projection.go`
- `tools/reference-first-web-target/server.go`
- `tools/reference-first-web-target/main.go`
- `tools/reference-first-web-target/projection_test.go`

方針:

```text
Canonical / Driver meaning
  -> reference-specific Web projection
  -> HTML / JSON
```

このincrementでは汎用Presentation IRやWeb frameworkを追加しない。Go標準`net/http`で、Driver Resolverが解決したApplication Pattern / Contract Query / Capability参照を薄くProjectionする。

Gate C provenanceとして、

- Handover Baseline ID
- `baselineRepositoryRevision = R`
- `targetBuildRevision = T`

を別identityで必須入力にする。

projection shellに加え、`/scenario/read`へread-only Driver Scenarioを接続した。

```text
browse
  -> selected sales.order
  -> record resolution
  -> customer reference
     -> development / master_data.dev / oracle
  -> order-lines header-detail
     -> development / sales_core.dev / postgres
```

reference / header-detailは既存のEnvironment-aware Driver APIを使う。browseはcurrent Driver APIにEnvironment-aware executor selectionがないため、Web側でEnvironment instanceを推測せず、その非対称性をunresolvedとして明示する。

さらに`POST /scenario/submit`へ、Gate Bと同じ意味順序を保ったCommand Scenario sourceを接続した。

```text
Authentication
  -> read slice
  -> Canonical Validation
  -> Authorization
  -> Command Business Result
  -> separate explicit Observation
```

Authentication / Authorizationの`denied`、Validation `not-satisfied`は正常な業務結果として保持し、runnerのExecution Errorへ畳まない。Command acknowledgementだけでObserved state成立とはせず、Observationは`ExecuteObservationRead`の別readで取得する。credentialはruntime inputとしてのみ扱い、responseへ保持しない。

standalone `/scenario/read` はReference診断用で、productionの認証済みread endpointではない。Command flow内部ではAuthentication後に同じread runnerを利用する。

さらに`/custom-submit`へCustom HTML UIを追加し、JSON APIと同じ`commandScenarioRunner`を共有するsource pressureを置いた。

```text
standard JSON interaction
        ┐
        ├─> commandScenarioRunner
        │    -> Authentication
Custom UI form
        ┘    -> Validation
             -> Authorization
             -> Command
             -> Observation
```

Custom UIは独自のValidation / Authorization / Capability / Result modelを持たず、form / layout / HTML renderingだけをWeb固有差として持つ。

現在もSource Evidenceであり、actual server execution / browser replay / Functional Evidence / Visual Evidenceは未取得である。

次incrementでは同じprimary `ISSUE-000007` workstream上でGate C Evidence acquisitionの実行入口を整える。

## 背景

TEIでは、最初からWeb / Desktop / Mobile / API / MCP / VUIを一つの巨大なPresentation IRへ押し込みません。

まず`order_entry` ApplicationをApplication DriverでUIなしに成立させ、Gate Bを通過した意味を第一TargetであるWebへProjectionします。その過程から、Application上の意味、複数Targetで共有できそうなPresentation / Interaction候補、Web固有実装の境界を抽出します。

Gate CではWebとDriverが同じCanonical Application上の意味を共有していることをSemantic Conformanceとして確認します。

## Gate Bから受け取るbaseline

第一Web Targetは、曖昧な「最新Driver」から開始しません。

[`GATE_B_TO_C_HANDOVER_CONTRACT.md`](../GATE_B_TO_C_HANDOVER_CONTRACT.md) / `ISSUE-000030` / GitHub Issue #291に従い、Gate B通過時のHandover Baseline `B`を受け取ります。

少なくとも次へ追跡できる必要があります。

- Handover Baseline ID
- `baselineRepositoryRevision = R`
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

## Baseline RとWeb build Tを混同しない

Web Targetの実装commitを含むrevisionを`T`とします。

```text
Handover Baseline B at R
        ↓ baseline meaning
Web Target implementation at T
        ↓
Gate C execution / Evidence
```

通常`T`は`R`より後になります。Target buildが後続revisionであること自体は問題ではありません。

ただし`T`でbaseline subjectのApplication meaningを黙って変更しません。

Gate C進行中にApplication / Driver meaningを変更する必要が生じた場合:

1. Web固有fieldやruleで不足を埋めない
2. authoritative sourceへ戻って変更する
3. Driver / Gate B側の必要Evidenceを再確認する
4. 新しいHandover Baseline `B'` / revision `R'`を発行する
5. Gate C Evidenceを`B'`へ切り替える

旧baselineのEvidenceを新baselineへ黙ってcarry-forwardしません。

## 目的

Web画面を作ること自体ではなく、**Gate Bを通過したApplication meaningをWebへ落とすために本当に必要なPresentation / Interaction上の意味を、具体的なReferenceから抽出すること**を目的とします。

Web専用のApplication modelやresult modelを作りません。

## 対象Scenario

少なくとも次をHandover BaselineのDriver Scenarioと対応付けます。

- browse
- record open
- reference lookup
- header-detail read
- header-detail edit（baselineがwrite意味を持つ場合）
- Validation satisfied / not satisfied
- Authorization allowed / denied
- command invocation
- Business Result
- Execution Error
- Observation

Command acknowledgementだけでObserved state成立とはみなしません。

Handover Baselineに存在しない意味をWeb側だけで追加してScenario coverageを水増ししません。

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
5. execution artifact / ABI / Host差（観測可能な範囲）
6. Execution / Trust差
7. Web固有実装
8. Host control plane固有差
9. Canonicalへ持ち上げてはいけない差
10. unresolved

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

各Evidence runから少なくとも次へ戻れる必要があります。

- Handover Baseline ID
- `baselineRepositoryRevision = R`
- `targetBuildRevision = T`
- Target Adapter / Host
- run identity

browser automation、GUI agent、VLM、screenshot diff等は利用できますが、Visual Evidenceや外部judge scoreだけでSemantic / Functional Conformanceを合格にしません。

Functional Evidenceをreplayable behaviorとして残す場合は、少なくともHandover Scenario / semantic operation、behavior precondition、Target-specific interaction realization、Result / Observation、raw Target Evidenceへ戻れるようにします。selector、coordinate、route等をCanonical interaction identityへ昇格しません。

Replayable Behaviorは任意のEvidence形式です。Gate Cのためだけに専用automation frameworkやUniversal Event Modelを導入しません。

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

- [x] Gate B Ready後、`ISSUE-000030`の第一Handover Baseline `B` / `R`を受け取る
- [ ] Target build revision `T`をbaseline `R`と別に追跡する
- [ ] `order_entry`の主要Patternを同じCanonical meaningからWebへProjectionする
- [ ] Validation / Authorization / Capability / Result / ObservationをWeb側で再定義しない
- [ ] 各Gate C runからHandover Baselineへ追跡できる
- [ ] `T`でbaseline meaningが変わった場合、新Handover Baselineへ切り替える
- [ ] Semantic ConformanceをEvidenceから確認できる
- [ ] Reference ScenarioのInteraction / Functional Evidenceを再現できる
- [ ] replayable behaviorを利用する場合、semantic operation / raw Target Evidenceへtraceできる
- [ ] Visual Fidelityを別Evidenceとして残せる
- [ ] standard ProjectionとCustom UIを同じ上位意味へ接続できる
- [ ] Web Adapterを外してもCanonical Worldが変わらない
- [ ] Target Capability / Runtime Binding / execution artifact・ABI・Host / Execution・Trust差を`ISSUE-000026`へ渡す
- [ ] Gate CをIssue close状態ではなくEvidenceから判定できる

## 非目標

- Gate B前の本格Web実装
- Web frameworkの先行固定
- 完全なPresentation IRの先行設計
- 第一TargetだけでTarget Capability taxonomyを完成させること
- Gate Cのためだけに複数execution artifactやWasm実装を用意すること
- 第一Targetだけでexecution artifact / ABI / Host taxonomyを完成させること
- Webの不足からCanonicalを逆設計すること
- pixel-perfect design systemの完成
- Target build revisionをHandover Baseline revisionと同一に固定すること

## 関連

- `plan/GATE_B_TO_C_HANDOVER_CONTRACT.md`
- `plan/GATE_C_EVIDENCE_CONTRACT.md`
- `plan/issue/000028-first-web-target-conformance-evidence.md`
- `plan/issue/000030-gate-b-to-c-handover-baseline.md`
- GitHub Issue #70 / #259 / #270 / #291


## Gate C Evidence acquisition source

standard / Custom UI shared boundaryの次incrementとして、actual Target runからSemantic / Functional Evidenceを採取するsourceを追加した。

- `tools/run-first-web-target-gate-c-evidence.sh`
- `tools/reference-first-web-target-evidence-audit`
- `plan/GATE_C_FIRST_WEB_TARGET_EVIDENCE_RUNBOOK.md`

acquisition runnerはclean `targetBuildRevision = T`からserverを起動し、同一runで少なくとも次を保持する。

- manifest
- read-only Driver Scenario
- successful Command
- Authentication denied
- Authorization denied
- Canonical Validation not-satisfied
- Command Invocation Execution Error
- Custom UI GET / successful POST
- server / health
- source qualification

Validation not-satisfied用のorder `1002`、Execution Error用のorder `1003`はReference runtime fixtureでありCanonical dataではない。

Execution ErrorはBusiness Outcomeへ変換せず、HTTP Target層だけが`B / R / T`とpartial Scenario resultを伴う422 envelopeへProjectionする。

auditはVisual Fidelityを成功へ読み替えず、Semantic / Functionalが成立しても`gateCReady = false`を返す。

現時点ではrunner / auditの**Source Evidence**であり、actual clean checkout上のrun artifactはまだ取得していない。


## Visual Evidence intake source

Semantic / Functional acquisition sourceに続き、同一Target revision `T`のScreenshotと評価記録を受け入れるReference auditを追加した。

- `tools/reference-first-web-target-visual-evidence-audit`

capture toolは固定しない。外部で取得したScreenshotを、Semantic / Functional summaryと同じ`B / R / T`へ明示的に結び付ける。

第一Referenceでは少なくとも次をVisual capture stateとして要求する。

- standard Projection initial
- Custom UI initial
- Custom UI success
- Authorization denied
- Validation not-satisfied
- Execution Error

Visual評価はGate C contractのreadability / relation understanding / header-detail locality / feedback distinguishability / Business Result vs Observation / standard vs Custom UI差へ対応する。

auditはScreenshot exact bytesをfingerprintし、評価結果とcapture referenceの完全性を確認するが、evaluator判断をCanonical meaningへ昇格しない。

全Visual criteriaがpassでも`gateCReady = false`を維持し、最終Gate C判定はSemantic / Functional / Visualを別Evidenceとして揃えた後に行う。

現時点ではVisual auditのSource Evidenceであり、actual Screenshot / evaluator recordは未取得である。


## Architecture negative Evidence source

Gate C contractのnegative / architecture conformanceをactual Repository runで再現するため:

- `tools/reference-first-web-target-architecture-evidence`

を追加した。

第一Referenceでは`R → T`間で`reference/workspace/**`と`reference/world.json`が不変であることを要求し、Web Target配下にauthoritative `.tei` / `.sql` / native SQL本文を持ち込まないことをscanする。

さらにactual `T`でstandard / Custom UI shared Command boundary、Validation not-satisfied、Execution Error分離のfocused testを実行し、run前後同一clean revisionを要求する。

これは第一Web Target向けのcoarseなarchitecture Evidenceであり、一般化されたSemantic diff taxonomyではない。

現在はSource Evidenceのみで、actual `architecture-conformance.json`は未取得である。
