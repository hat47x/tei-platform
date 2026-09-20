# ISSUE-000026: Target CapabilityとExecution / Trust Profileの境界を検証する

- 状態: Ready — Gate C Ready確認済み（T = `0ba87a4e0a96f70a198b03ff7652fb00e3826abe`、2026-09-19）/ 第二Target候補の選定から着手可能
- GitHub Issue: #259
- 本格着手条件: Gate C Ready（成立済み）
- Product Acceptance Scenario: PAS-04 primary

## 背景

TEIはRQ-013で、Web、Desktop、Mobile、API、MCP、VUI等を同じ上位設計から扱い、Targetごとの差をCapability negotiationやRuntime Bindingで明示する方針を置いています。

`ISSUE-000007`では、第一Web Targetを通してApplication上の意味、Target-neutralなPresentation候補、Web固有差分を分離し、Gate Cでその差を説明できる状態を目指します。

一方、Target差には少なくとも二種類あります。

1. **Target Capabilityの差**
   - native window、menu、tray、dialog、touch、haptics、geolocation等の有無
   - Desktop / Mobile / Server等で利用可能な操作・入出力が異なる
2. **Execution / Trust Profileの差**
   - local / in-processかnetwork越しか
   - single-user / single-clientか複数clientか
   - TLS、origin、identity / session、Resource、並行性等の境界が異なる

この二つを一つの「Target差」へ畳むと、UI機能差とsecurity / runtime差が混同されます。

本Issueでは特定frameworkのAPIをTEIへ写すのではなく、複数Targetの実証から、**何がTarget Capabilityで、何がExecution / Trust Profileとして独立して扱うべきか**を確認します。

## 関連するProduct Requirement

- RQ-001 業務上の意味を実装技術から独立して保持できること
- RQ-007 Provider固有知識をCoreへ無制限に持ち込まないこと
- RQ-008 Capability単位で標準実装と固有実装を置き換えられること
- RQ-011 同じCanonical meaningを複数のProjectionで扱えること
- RQ-012 Driverを設計検証用の実行・シミュレーション環境として利用できること
- RQ-013 Target-neutralなPresentation / Interaction境界を持つこと
- RQ-019 実行時の権限とResourceを明示的に制約できること
- RQ-021 ReferenceとConformanceで再現可能性を示すこと
- RQ-022 完成製品と設計検証段階を明確に区別すること
- RQ-024 性能要求をTargetごとに検証可能にすること

## Portable execution artifact / ABIを第三の観測軸としてPressureする

2026-09の技術リサーチで、Hikeのような極小freestanding WebAssembly、WebAssembly Component Model / WASI 0.3、agent-friendly UI、replayable interaction Evidence等を現行境界へ当てた結果、Target CapabilityとExecution / Trust Profileだけでは説明を省略しやすい差が一つ見えている。

```text
Target
  != execution artifact
  != execution host
```

WasmはWeb / Desktop / Mobile / API / MCP / VUIと同列のTargetではない。同じWeb Targetでもremote/native runtime、freestanding Wasm、Worker-hosted Wasm等を取り得て、同じWasm artifactもbrowser / desktop host / standalone runtime等で再利用できる可能性がある。

現時点では新しいproduction conceptを追加せず、Runtime Bindingの内部差として少なくとも次を観測する。

- execution host family
- execution artifact form（native Go / freestanding Wasm / Wasm Component / external process等）
- ABI / interface contract（native API / WIT / C ABI / JS bridge / RPC等）
- host imports / exports
- scheduling / concurrency mode
- execution / trust / resource condition

この差が第二Target実証後も独立したidentity / lifecycle / Conformanceを必要とする場合にだけ、Target Capability / Execution Profileとは別のformal conceptが必要かをADRで判断する。

Hikeは任意のGo sourceをWasmへ変換するcompilerではなく、Go風の別言語からLLVM IR等へ下ろす実装であるため、TEI Go Coreの置換候補とは扱わない。検討対象は「accepted meaningから小さいtyped execution subsetを導出し、portable artifact providerへ下ろせるか」である。

詳細なread-only Pressureは[`PORTABLE_EXECUTION_ARTIFACT_ARCHITECTURE_PRESSURE_2026-09.md`](../PORTABLE_EXECUTION_ARTIFACT_ARCHITECTURE_PRESSURE_2026-09.md)に置く。

## 第一Targetから受け取る比較baseline

第二Targetは曖昧な「最新Web」や「最新Canonical」と比較しません。

Gate Cで使ったHandover Baseline `B`と、そのbaseline revision `R`を比較の意味基準として保持します。

各Targetの実装revisionは別に追跡します。

```text
Application meaning baseline B at R
        │
        ├─ Web Target build T1
        │    └─ Gate C Evidence
        │
        └─ Second Target build T2
             └─ comparison Evidence
```

第二Target着手時にApplication meaningが既に更新されている場合、旧`B`へ無理に合わせず、明示的なsuccessor Handover Baseline `B'` / `R'`を選びます。

Web `T1`、第二Target `T2`、baseline `R`を同じrevisionとして扱いません。

## 現時点の設計仮説

### Target Capability

Target Capabilityは、あるTarget / Hostで利用可能なPresentation、Interaction、Device機能を表す候補とします。

例:

- interactive UI
- multi-window
- native dialog
- system tray
- touch input
- keyboard / pointer
- file open / save
- notification
- haptics
- geolocation
- camera等のdevice access

Canonical Applicationがframework APIを直接要求するのではなく、必要な上位Capabilityを要求し、Target Adapter / Host側が提供可能な集合を示す形を比較します。

Capabilityが存在しない場合も、直ちにApplication意味の欠落とはみなしません。代替Projection、利用不可表示、別Targetへの委譲等を選べる余地を残します。

### Execution / Trust Profile

Execution / Trust Profileは「何を表示できるか」ではなく、実行環境と信頼境界の性質を表す候補とします。

- local / in-processかnetwork越しか
- single-user / single-clientか複数clientか
- transportが内部bridgeかHTTP / WebSocket等か
- origin / connection sourceを検証する必要があるか
- TLS等のtransport protectionが必要か
- clientごとのidentity / session / authorizationが必要か
- Resource / secretをどこで解決するか
- 並行実行やbroadcast eventをどう扱うか
- process / device / browser tab等のlifecycleをどう観測するか

これらをPresentation Capabilityへ押し込まず、Runtime BindingやSecurity / Resource境界と接続できる別軸として扱う必要があるかを実証します。

## Wailsを使う場合の境界

WailsはDesktop / WebView host等の圧力試験材料になり得ますが、Wails自体をTEI仕様の根拠にはしません。

### Studio Wails HostとGenerated Wails Targetを分ける

TEI Studio / Editorがdeveloper-facing desktop authoring clientとしてWailsを使う場合でも、それをApplication Target Evidenceへ数えません。

```text
Studio / Editor
  frontend
    ↓
Wails developer host
    ↓
adapters/studiowails
    ↓
app/studio
    ↓
shared Acceptance Boundary
    ↓
Canonical Source
```

これは**authoring toolのHost**です。

一方、将来のGenerated Wails Desktop Targetは、Canonical Applicationをend-user向けDesktop ApplicationとしてProjectionする別のTargetです。

```text
Canonical Application
    ↓
Desktop Target Adapter
    ↓
Wails end-user Host
    ↓
Business Application interaction
```

StudioのWails採用・動作・GUI Evidenceを、第二TargetのSemantic / Functional / Visual Evidenceへ読み替えません。

Wails Host外層にbusiness validation、Authorization rule、Persistence semantics、Canonical mutation authorityを持たせない方針も維持します。

### MCP / control面をApplication Targetと混同しない

Host inspectionやdevelopment controlのMCP interfaceを持っても、それだけでTEI ApplicationがMCP Targetを持つことにはなりません。

```text
Host / Tool Control Plane
  └─ inspection / automation / development control

Application Target
  └─ 業務上のPresentation / InteractionをMCPとして公開
```

同じprotocolでもauthority、subject、lifecycle、Conformance対象が異なります。

## 実証の進め方

### Gate Cまで

`ISSUE-000007`の第一Web Targetで次をEvidenceとして残します。

- Handover Baseline ID / revision `R`
- Web Target build revision `T1`
- Web固有Capability
- Target-neutral Presentation候補
- Runtime Binding差
- execution artifact / ABI / Host差（観測できる範囲）
- HTTP / browser由来の差
- Execution / Trust差
- Canonicalへ持ち上げるべきでない実装都合
- Semantic / Functional / Visual Evidenceの分類
- performance measurement
- unresolved

この段階ではTarget Capability schema、Execution / Trust Profile schema、portable execution artifact / ABI用の独立schemaを固定しません。

### Gate C後

2026-09-21: Gate CはReadyとなった（[Gate C Readiness Audit](../GATE_C_READINESS_AUDIT.md)）。第一Targetの観測（`targetDifferenceHandoff`のTarget Capability候補、Runtime Binding、execution artifact・ABI・Host差、Execution・Trust差）は同auditの「第二Targetへのhandoff」から受け取る。同auditがcarry-forwardとして本Issueへ渡したunresolvedは`browse-environment-instance-binding-not-exposed-by-current-driver-api` / `general-target-neutral-presentation-taxonomy-not-yet-defined` / `target-capability-taxonomy-remains-candidate-until-second-target`である。

第一Targetとの差を説明できるようになったので、第二Target候補を選びます。

第二Targetは、第一Targetと十分に異なる圧力を与えられるものを優先します。

候補例:

- Generated Wails Desktop Target
- Mobile
- Server
- API
- MCP Application Target

単に実装が容易という理由だけで選ばず、RQ-013 / RQ-024の不確実性を最も減らせるTarget / Hostを選びます。

Studio developer hostを「すでにDesktop Targetがある」という理由には使いません。

### 比較軸

同じReference Applicationについて少なくとも次を比較します。

1. Handover Baseline / Canonical Application上の意味
2. Target build revision `T1` / `T2`
3. 必須 / 任意のTarget Capability
4. Targetが提供するCapability
5. Projection差
6. Runtime Binding差
7. execution artifact form / ABI / Host差
8. Execution / Trust Profile
9. Authorization / Resource境界
10. Observation方法
11. 性能測定点
12. Semantic / Functional / Visual Evidence
13. Target固有コードやartifact実現を削除・交換したときCanonicalが変化しないこと

## Conformance候補

実証結果が十分になった場合、次のようなConformanceを検討します。

- Target Adapterが提供Capabilityを機械的に列挙できる
- Application側のrequired Capability不足を実行前に検出できる
- optional Capability不足をApplication意味の破損として扱わない
- Target Adapterを交換してもCanonical IDやApplication Patternが変わらない
- execution artifact / ABI / Hostを交換しても、それだけでCanonical Application meaningが変わらない
- Wasm等のartifact形式をApplication Target kindへ短絡しない
- artifact / ABI側の制約をBusiness / Canonical meaningの否定へ短絡しない
- network越しのExecution / Trust Profileをlocal profileと同じ暗黙trustで扱わない
- Target固有transportからBusiness Result / Execution Error / Observationを逆定義しない
- secret、endpoint、filesystem path等のHost materialをCanonicalへ不要に保存しない
- Host control planeとApplication Targetを区別できる
- Studio Wails HostとGenerated Wails Desktop Targetを区別できる
- 同じScenarioの意味上の結果を複数Targetで比較できる
- RQ-024の測定値をTarget / Host / runtime条件と一緒に比較できる

## 非目標

- WailsをTEI標準Runtimeとして確定すること
- Wailsのplatform APIをそのままTEI Capability名へ写すこと
- Gate C前に完全なCapability taxonomyを設計すること
- `WasmTarget`等のartifact形式をTarget kindとして固定すること
- `ExecutionArtifactProfile`等のproduction schemaを先行固定すること
- Hike / LLVM / WIT / WASIをTEI Coreの必須実装へすること
- Desktop / Mobile / Server / API / MCPを最初から同じ深さで実装すること
- Security policyをPresentation modelへ埋め込むこと
- 一つのTargetだけからTarget-neutral境界を確定すること
- HostのMCP control interfaceを無条件に業務MCP Targetとみなすこと
- Studio / Editorのdeveloper-facing hostを第二Application Target Evidenceへ数えること

## 完了条件

少なくとも二つの性質が異なるApplication Target / Hostを同じ明示baselineへ接続し、次をEvidenceから説明できることを完了条件とします。

- 比較に使ったHandover Baseline ID / revision
- 各Target build revision
- Targetが変わっても残るCanonical Application上の意味
- Target Capabilityとして扱うべき差
- execution artifact / ABI / Hostとして観測すべき差
- Execution / Trust Profileとして扱うべき差
- Runtime Bindingで扱う差
- Target固有実装へ残す差
- Host control planeに残す差
- Canonicalへ持ち上げてはいけない差
- performance差と測定条件
- Studio developer hostがApplication Target比較へ混入していないこと

その結果から、Target Capability、Execution / Trust Profile、portable execution artifact / ABI差のどこまでを正式な別概念として採用するか、既存Runtime Binding / Capabilityモデルへ統合するかをADRで判断します。

## 関連文書

- `plan/GATE_C_EVIDENCE_CONTRACT.md`
- `plan/GATE_B_TO_C_HANDOVER_CONTRACT.md`
- `plan/TECH_RESEARCH_INTEGRATION_2026-09.md`
- `requirements/TEI_PRODUCT_REQUIREMENTS.md`
- `plan/MASTER_PLAN.md`
- `plan/V1_REFERENCE_APPLICATION_PLAN.md`
- `plan/issue/000007-first-web-target-projection.md`
- `plan/issue/000008-editor-and-ai-followup-note.md`
- `plan/issue/000028-first-web-target-conformance-evidence.md`
