# ISSUE-000020: Target CapabilityとExecution Profileの境界を検証する

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
2. **Execution / Trust境界の差**
   - Desktop / Mobileでは主にin-processのWebView bridgeとして実行される
   - Server BuildではHTTP / WebSocketを介して複数browser clientから利用される
   - ServerではTLS、origin制御、input validation等、network越しの信頼境界を扱う必要がある

この二つを一つの「Target差」へ畳むと、UI機能の不足と、認証・認可・通信・並行性・Resource境界の違いが混同される可能性があります。

本Issueでは、特定frameworkのAPIをTEIへ写すのではなく、複数Targetの実証から、**何がTarget Capabilityで、何がExecution / Trust Profileとして独立して扱うべきか**を確認します。

## 関連するProduct Requirement

- RQ-001 業務上の意味を実装技術から独立して保持できること
- RQ-007 Provider固有知識をCoreへ無制限に持ち込まないこと
- RQ-008 Capability単位で標準実装と固有実装を置き換えられること
- RQ-011 同じCanonical meaningを複数のProjectionで扱えること
- RQ-012 Driverを設計検証用の実行・シミュレーション環境として利用できること
- RQ-013 Target-neutralなPresentation / Interaction境界を持つこと
- RQ-019 実行時の権限とResourceを明示的に制約できること
- RQ-021 ReferenceとConformanceで再現可能性を示すこと
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

2026年9月時点の公式資料では、次が確認できます。

- iOS / Android / Desktopで同じ`main.go`、Go backend、frontend、service binding、event modelを共有できる
- Mobileではmulti-window、window geometry、menu、tray等が利用できず、haptics、geolocation、biometrics等のMobile Capabilityが追加される
- Server Buildでも同じservice bindingとevent modelを利用できる
- Serverではnative window、tray、dialog等が存在せず、browser clientをHTTP / WebSocket越しに扱う
- Serverのproduction利用ではTLS、reverse proxy、WebSocket origin制御、input validation等が必要になる

この差を利用し、同じCanonical Applicationを異なるHostへProjectionしたとき、Capability差とExecution / Trust差を別々に説明できるか確認します。

参考資料:

- Wails v3 Mobile Overview: https://v3.wails.io/guides/mobile/
- Wails v3 Server Build: https://v3.wails.io/guides/server-build/
- Wails v3 Changelog: https://v3.wails.io/changelog/

## 実証の進め方

### Gate Cまで

`ISSUE-000007`の第一Web Targetで、次のEvidenceを残します。

- Web固有Capability
- 上位Presentation候補
- Runtime Bindingで吸収している差
- HTTP / browser由来の差
- Canonicalへ持ち上げるべきでない実装都合

この段階ではTarget Capability schemaやExecution Profile schemaを固定しません。

### Gate C後

第一Targetとの差を説明できるようになったら、第二Target候補を選びます。

第二Targetは、第一Targetと十分に異なる圧力を与えられるものを優先します。たとえば次の候補があります。

- Wails Desktop / WebView host
- Wails Mobile
- Wails Server Build
- API / MCP

単に実装が容易という理由だけで選ばず、RQ-013の不確実性を最も減らせるTargetを選びます。

### 比較軸

同じReference Applicationについて、少なくとも次を比較します。

1. Canonical Application上の意味
2. 必須 / 任意のTarget Capability
3. Targetが提供するCapability
4. Projection差
5. Runtime Binding差
6. Execution / Trust境界
7. Authorization / Resource境界
8. Observation方法
9. 性能測定点
10. Target固有コードを削除したときCanonicalが変化しないこと

## Conformance候補

実証結果が十分になった場合、次のようなConformanceを検討します。

- Target Adapterが提供Capabilityを機械的に列挙できる
- Application側のrequired Capability不足を実行前に検出できる
- optional Capability不足をApplication意味の破損として扱わない
- Target Adapterを交換してもCanonical IDやApplication Patternが変わらない
- network越しのExecution Profileをlocal profileと同じ暗黙のtrustで扱わない
- Target固有transportからBusiness Result / Execution Error / Observationを逆定義しない
- secret、endpoint、filesystem path等のHost materialをCanonicalへ不要に保存しない
- 同じScenarioの意味上の結果を複数Targetで比較できる

## 非目標

- WailsをTEI標準Runtimeとして確定すること
- Wailsのplatform APIをそのままTEI Capability名へ写すこと
- Gate C前に完全なCapability taxonomyを設計すること
- Desktop / Mobile / Server / API / MCPを最初から同じ深さで実装すること
- Security policyをPresentation modelへ埋め込むこと
- 一つのTargetだけからTarget-neutral境界を確定すること

## 完了条件

少なくとも二つの性質が異なるTarget / Hostを同じReference Applicationへ接続し、次をEvidenceから説明できることを完了条件とします。

- Targetが変わっても残るCanonical Application上の意味
- Target Capabilityとして扱うべき差
- Execution / Trust Profileとして扱うべき差
- Runtime Bindingで扱う差
- Target固有実装へ残す差
- Canonicalへ持ち上げてはいけない差

その結果から、Target CapabilityとExecution / Trust Profileを正式な別概念として採用するか、既存のRuntime Binding / Capabilityモデルへ統合するかをADRで判断します。

## 関連文書

- `requirements/TEI_PRODUCT_REQUIREMENTS.md`
- `plan/MASTER_PLAN.md`
- `plan/V1_REFERENCE_APPLICATION_PLAN.md`
- `plan/issue/000006-application-driver-vertical-slice.md`
- `plan/issue/000007-first-web-target-projection.md`
- `plan/issue/000008-editor-and-ai-followup-note.md`
