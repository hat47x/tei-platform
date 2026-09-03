# ISSUE-000026: Target CapabilityとExecution / Trust Profileの境界を検証する

- 状態: Proposed
- GitHub Issue: #259
- 本格着手条件: Gate C

## 背景

TEIはRQ-013で、Web、Desktop、Mobile、API、MCP、VUI等を同じ上位設計から扱い、Targetごとの差をCapability negotiationやRuntime Bindingで明示する方針を置いています。

`ISSUE-000007`では、第一Web Targetを通してApplication上の意味、Target-neutralなPresentation候補、Web固有差分を分離し、Gate Cでその差を説明できる状態を目指しています。

一方、2026年9月時点のWails v3を調査すると、Target差には少なくとも二種類あることが分かります。

1. **Target Capabilityの差**
   - native window、menu、tray、dialog、touch、haptics、geolocation等の有無
   - Desktop / Mobile / Serverで利用可能な操作や入出力が異なる
2. **Execution / Trust Profileの差**
   - Desktop / Mobileでは主にlocal / in-processのWebView bridgeとして実行される
   - ServerではHTTP / WebSocket等を介して複数clientから利用され得る
   - network越しではTLS、origin、identity / session、input validation、Resource境界等を明示する必要がある

この二つを一つの「Target差」へ畳むと、UI機能の不足と、認証・認可・通信・並行性・Resource境界の違いが混同される可能性があります。

本Issueでは、特定frameworkのAPIをTEIへ写すのではなく、複数Targetの実証から、**何がTarget Capabilityで、何がExecution / Trust Profileとして独立して扱うべきか**を確認します。

このRepository内Issueは、main側で先行起票されていた`ISSUE-000020`と、収束branchで既に利用されていた別の`ISSUE-000020`の番号衝突を解消するため、`000026`へ統一します。GitHub Issueは引き続き#259を利用します。

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

Canonical Applicationが個々のframework APIを直接要求するのではなく、必要な意味上のCapabilityを要求し、Target Adapter / Host側が提供可能な集合を提示する形を比較します。

Capabilityが存在しない場合も、直ちにApplication意味の欠落とはみなしません。代替Projection、利用不可表示、別Targetへの委譲等を選べる余地を残します。

### Execution / Trust Profile

Execution / Trust Profileは、「何を表示できるか」ではなく、実行環境と信頼境界の性質を表す候補とします。

たとえば次の差を検討します。

- local / in-processか、network越しか
- single-user / single-clientか、複数clientを扱うか
- transportが内部bridgeか、HTTP / WebSocket等か
- originや接続元を検証する必要があるか
- TLS等のtransport protectionが必要か
- clientごとのidentity / session / authorizationが必要か
- Resourceやsecretをどの境界で解決するか
- 並行実行やbroadcast eventをどう扱うか
- process / device / browser tab等のlifecycleをどう観測するか

これらをPresentation Capabilityへ押し込まず、Runtime BindingやSecurity / Resource境界と接続できる別軸として扱う必要があるかを実証します。

## Wails v3を使う場合の検証材料

Wails v3はこの境界を考えるための有力な圧力試験材料ですが、Wails自体をTEI仕様の根拠にはしません。

2026年9月時点の公式資料から、Desktopを中心とするGo backend / frontendの構成に加えて、Mobile、Server、MCP等へ機能が広がっていることを確認できます。

この差を利用し、同じCanonical Applicationを異なるHostへProjectionしたとき、Capability差とExecution / Trust差を別々に説明できるか確認します。

Mobile等には実験的な位置づけが残るため、現時点ではWailsをTarget-neutral設計の前提にはせず、Adapter / Host候補かつarchitecture probeとして扱います。

### MCP / control面をApplication Targetと混同しない

Wails等がHost inspectionやcontrolのためにMCP interfaceを提供しても、それだけでTEI ApplicationがMCP Targetを持つことにはなりません。

次を区別します。

```text
Host / Tool Control Plane
  └─ inspection / automation / development control

Application Target
  └─ 業務上のPresentation / InteractionをMCPとして公開
```

同じprotocolを利用していても、authority、subject、lifecycle、Conformanceの対象が異なります。

第二Target候補としてMCPを選ぶ場合は、どちらの意味でMCPを利用するのかを明示します。

## 実証の進め方

### Gate Cまで

`ISSUE-000007`の第一Web Targetで、次のEvidenceを残します。

- Web固有Capability
- 上位Presentation候補
- Runtime Bindingで吸収している差
- HTTP / browser由来の差
- Canonicalへ持ち上げるべきでない実装都合
- Semantic / Functional / Visual Evidenceのどこに属する差か

この段階ではTarget Capability schemaやExecution / Trust Profile schemaを固定しません。

### Gate C後

第一Targetとの差を説明できるようになったら、第二Target候補を選びます。

第二Targetは、第一Targetと十分に異なる圧力を与えられるものを優先します。たとえば次の候補があります。

- Wails Desktop / WebView host
- Wails Mobile
- Wails Server
- API
- MCP Application Target

単に実装が容易という理由だけで選ばず、RQ-013の不確実性を最も減らせるTarget / Hostを選びます。

### 比較軸

同じReference Applicationについて、少なくとも次を比較します。

1. Canonical Application上の意味
2. 必須 / 任意のTarget Capability
3. Targetが提供するCapability
4. Projection差
5. Runtime Binding差
6. Execution / Trust Profile
7. Authorization / Resource境界
8. Observation方法
9. 性能測定点
10. Semantic / Functional / Visual Evidence
11. Target固有コードを削除したときCanonicalが変化しないこと

## Conformance候補

実証結果が十分になった場合、次のようなConformanceを検討します。

- Target Adapterが提供Capabilityを機械的に列挙できる
- Application側のrequired Capability不足を実行前に検出できる
- optional Capability不足をApplication意味の破損として扱わない
- Target Adapterを交換してもCanonical IDやApplication Patternが変わらない
- network越しのExecution / Trust Profileをlocal profileと同じ暗黙trustで扱わない
- Target固有transportからBusiness Result / Execution Error / Observationを逆定義しない
- secret、endpoint、filesystem path等のHost materialをCanonicalへ不要に保存しない
- Host control planeとApplication Targetを区別できる
- 同じScenarioの意味上の結果を複数Targetで比較できる

## 非目標

- WailsをTEI標準Runtimeとして確定すること
- Wailsのplatform APIをそのままTEI Capability名へ写すこと
- Gate C前に完全なCapability taxonomyを設計すること
- Desktop / Mobile / Server / API / MCPを最初から同じ深さで実装すること
- Security policyをPresentation modelへ埋め込むこと
- 一つのTargetだけからTarget-neutral境界を確定すること
- HostのMCP control interfaceを無条件に業務MCP Targetとみなすこと

## 完了条件

少なくとも二つの性質が異なるTarget / Hostを同じReference Applicationへ接続し、次をEvidenceから説明できることを完了条件とします。

- Targetが変わっても残るCanonical Application上の意味
- Target Capabilityとして扱うべき差
- Execution / Trust Profileとして扱うべき差
- Runtime Bindingで扱う差
- Target固有実装へ残す差
- Host control planeに残す差
- Canonicalへ持ち上げてはいけない差

その結果から、Target CapabilityとExecution / Trust Profileを正式な別概念として採用するか、既存のRuntime Binding / Capabilityモデルへ統合するかをADRで判断します。

## 関連文書

- `plan/TECH_RESEARCH_INTEGRATION_2026-09.md`
- `requirements/TEI_PRODUCT_REQUIREMENTS.md`
- `plan/MASTER_PLAN.md`
- `plan/V1_REFERENCE_APPLICATION_PLAN.md`
- `plan/issue/000006-application-driver-vertical-slice.md`
- `plan/issue/000007-first-web-target-projection.md`
- `plan/issue/000008-editor-and-ai-followup-note.md`
