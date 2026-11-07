# Plugin-first D1：Authoring Inspectorの実Provider選択・呼出しへ到達する最短Fit/Gap

- **状態**：現行Go実ソースの静的監査と実装着手票（2026-11-07）。設計候補であり新API/ABI採用ではない。**新しいGo build/test、実CLI/Wails、Current Grant/実Provider/実Target E2Eは NOT RUN**。
- **基線**：`hat47x/tei-platform` main `d6a452f79023744e7926669c3556d2699f546697`。GitHubで以下の実ファイルを個別参照した。行番号はその基線での目安であり実装時に再取得する。
- **既存正本**：[Plugin-first全体設計](../design/PLUGIN_FIRST_TOOLING_RUNTIME_DOGFOOD_2026-11.md) §4、§8〜13、[ADR-0032](../adr/0032-composition-kernel-replacement.md)、[ADR-0043](../adr/0043-target-native-artifacts-without-tei-host.md)。実装Owner：[Plugin Core移管 #1251](https://github.com/hat47x/tei-platform/issues/1251)、[Contract #951](https://github.com/hat47x/tei-platform/issues/951)、[Profile #1016](https://github.com/hat47x/tei-platform/issues/1016)、[Typed Adapter #1158](https://github.com/hat47x/tei-platform/issues/1158)、[Studio #1824](https://github.com/hat47x/tei-platform/issues/1824)、[Context #3262](https://github.com/hat47x/tei-platform/issues/3262)。
- **本票の責務**：既に計画された「CLIとStudioの同一Inspector Pluginを呼ぶ」を、現コードでどの入口を変更し、どの入口を流用してはならないかまで絞る。新Resolver、Registry、Lock、Test Runner、Authoring Host、第二意味正本を作らない。

## 1. 実ソースで新たに特定した決定的なGap

| 位置 | 観測した事実 | 実装上の拘束 |
| --- | --- | --- |
| `cmd/tei/main.go:79-87,138-143,342-350` | CLIは`isAcceptedCommand`を先に検査し、`inspect`だけを特別扱いで`inspectWorkspaceJSON`へ分岐する | 最初の移行は`inspect`一経路に限定。Plugin追加のため手書き分岐を増やすのではない |
| `cmd/tei/cli_inspect.go` | `operations`を直接使うCLI用結果投影がある | 互換Output/終了コードの保持とProvider dispatchは別に検証する |
| `cmd/tei-studio/main.go:27-45` | Wails `StudioBridge.InspectSubject`は`studiohost.ReadOnly`へ委譲する | ブラウザからProvider選択情報・Trusted Rootを受け取らない |
| `internal/studiohost/readonly.go:58-75,87-105` | `RootServices`とHost提供`ServiceFactory`があるが、現行は`Catalog`と`Validation`の注入に限定 | Host allowlist解決後にサービスを渡す**既存seam**を再利用。任意PathをServiceへ渡す新APIを作らない |
| `internal/studiohost/readonly.go:231-241` | `InspectSubject`はrootを検査後に`operations.InspectSubject`へ直行し、登録済みProviderを選択しない | `inspect`をHost所有の選択済みOperationへ委譲する必要。共通Go関数の利用だけではPlugin-firstにならない |
| `internal/operations/inspect.go:30-94` | `SubjectInspection`はWorld/SourceIndex fingerprint、sourceStatus、Canonicalを返す。World fingerprintはrevision token/acceptance grantではないと明記 | `inspect`成功＝Required Domain Closure完全・現在の開示資格・CAS済みではない。#3262と分ける |
| `internal/composition/registry.go:12-54` | ContractはID/Version/Shape、ProviderはID/Scope/Offers/Requiresを持つ。選択はRuntime権限付与ではない | 候補Providerの一意性・exact Scope・Dependency Epochを既存Compositionで解決する |
| `internal/composition/bootstrap.go:1-32`、`coordinator.go:39-64` | `NewSelectedCoordinator`は選択されたRegistryとHost guardを必要とし、`Acquire`で両者を検査 | 実Call前にHostのTrusted Rootと現在の操作可否を照合するための足場。ただしBuild Artifact/正式Conformance/現在GrantをCoordinatorがすべて証明するわけではない |
| **`internal/plugin/runtime/service_carrier.go:76-92`** | **`validateServiceSelection`が`Scope != "application-runtime"`を拒否する** | **既存`InvokeSelectedServiceCarrier`をそのまま`authoring-tool`へ使う実装は失敗する。文字列置換だけでScopeを緩めない** |
| `internal/plugin/runtime/model.go:516-588` | 旧Runtime RegistryにBuiltinRegistrationとArtifact/Engine固定の照合がある | これをそのままAuthoring用Universal ABIに昇格しない。必要な証拠のAdapterに限定する |

**結論**：論理CompositionのScopeは一般的である一方、**実行へ接続するLegacy Service Carrierはapplication-runtimeに限定**される。旧Carrierへ`authoring-tool`を通すためにScope検査を緩める変更は、Trusted Host/Contract Acceptance/Provider Conformanceの抜け道になり得る。D1はこの境界を尊重し、**Host-owned Authoring Operation Adapterを既存Compositionの選択へ結ぶ**小さな縦切りを先行させる。これも独立Runner/Authorityではない。

## 2. 最小の実装差分（既存コードを生かす）

1. **Contract・Required Contextを外部正本から引用**：当面は既存`tei.operations.inspect/v0`の正確なRequest/Resultを固定し、`Subject kind/id`・Exact Root・World/Source Index fingerprint・失敗の集合を記録。Required Closure、`expectedRevision`、Disclosure、`complete`が現行Operationに存在しない部分は**補助診断または未適格**とし、根拠なく既存v0が保証すると言わない。公開Contract版・ConformanceのOwnerは#951/#3262。
2. **共通Inspector Operation Portを最小化**：`Inspect(trustedRoot, typedSubject) -> typedInspection/error`相当のHost内部Portは既存`operations.InspectSubject`を実装Providerとして包む。正式API名・Go interfaceの配置はD0の依存を確認して決める。Portの存在自体はResolver／契約Publisherではない。
3. **Hostが選んだBinding**：`composition.Registry.Select`と`NewSelectedCoordinator`の既存Referenceを使う場合、`Contract Shape`・Provider ID・Scope・Epochを検証した後に、Trusted Hostだけが実装関数を注入する。Providerから`Root`やGrantを自己発行させない。`Registry.Select`は正式`Provider Conformance`でもBuild実体検証でもないため、D1のAdmission票に未充足を別記する。静的配布ではBuild対象の実Providerを選択済みに固定。
4. **CLI**：`inspectWorkspaceJSON`の内側で、引数とJSON Envelope/exit codeを保ちながら、選択済みOperationを呼ぶ。最初は旧直接経路とread-only shadow比較、切替後は旧経路への自動fallbackを拒否。
5. **Studio**：既存`ReadOnly.workspace(id)`のallowlist／root identity検査後、Host設定の選択済みInspectorを呼ぶ。`RootServices`の拡張・Host-only Adapterのどちらが変更最小かをD0で比較。Renderer/Wails bridgeはOperationを要求するだけでContract/Provider選択・Resource権限を持たない。
6. **実呼出証拠**：`Selected`と`Invoked`を別記録にする。`Provider ID+Contract Shape+Scope+Build member/Artifact ID+Source Revision+Invocation Attempt`を一致検証し、戻り値が偶然同じでも「Bを選択・Aを呼出」は不合格。現在のProvider/Artifact読取りが実装から取得できないときはunknownとする。
7. **移行の完了**：実二ConsumerのProvider実Call、否定試験、実選択Build、Host deny、同じSource/Meaningと同じ情報欠損診断がそろって初めてD1b完了。Studioの画面描画やCLI出力だけではない。D1b合格後にHelp/Studio Contributionへ一般化（D2）する。

**非採用**：`service_carrier.go`の`application-runtime`判定を`authoring-tool`も許すように単純変更すること、`StudioBridge`へProvider選択/生のPathを追加すること、`operations.InspectSubject`直呼出のままDescriptorだけPlugin登録すること、全Target共通の常駐Hostを作ること、テストHarnessだけで未接続のFunctionを「実Invoke済み」と言い換えること。

## 3. 独立Expectedを伴う最小Acceptance Packet

| ID | 固定入力/操作 | Expectedと観測される主体 |
| --- | --- | --- |
| **AS-01** | same source、同Scope、A selected/A actual calledをCLIとStudioで個別実行 | 両call-siteの実Provider invocationが観測され、Result/Failure/Source資格が一致。単一Mock結果再表示は不可 |
| **AS-02** | B selected、実AdapterがAを呼ぶ。A/Bの返却JSONはbyte同一 | **FAIL**。SelectionとActual Provider/Artifactが一致しない。結果の同一性で上書きしない |
| **AS-03** | 別Scope A/B、同Contract/ID、Host tool only許可 | application-runtime Providerはauthoring-toolへ暗黙流用不可。旧Service Carrierの拒否維持 |
| **AS-04** | Studio browserからroot path/Provider指定、または起動後にWorkspace実pathすり替え | allowlist/OS root identityによる拒否。戻り値のSource/パス/未公開存在の漏洩も検査 |
| **AS-05** | Providerなし、Provider同一適格候補二件、更新後のstale Epoch | missing/ambiguous/staleの理由つき拒否、CLIの旧直呼出へfallback不可 |
| **AS-06** | `operations.InspectSubject`が`sourceStatus:not-indexed`、または必要Source未読込 | 「未索引」「不完全」と「Subject不存在」「完全」を区別。#3262 Required集合が完結するまでは`complete`を主張しない |
| **AS-07** | stdout/UI/Harnessで内容を途中切断、Receiverがpage/end未受領 | 実Provider成功とConsumer受領完了を分離。8,192byteの成功応答でも完全としない |
| **AS-08** | Default Traitの実装なし/Manual Trait未実装/Advisoryのみ | 必須義務の未履行を拒否。Hintで実行資格を与えない（#3278） |
| **AS-09** | 同一Bundleに未選択Providerを追加 | 実Build/Link inputと実Binaryを照合して混入を検出。静的Goでは再Buildが必要 |
| **AS-10** | Provider出力同じ、Source/Grant/ScopeまたはArtifactのみ変更 | 必要な再資格化と不要失効を分離。Source fingerprint一致だけで現在Disclosure/Authorityを復活させない |
| **AS-11** | 同Contract Provider A1/A2×Consumer C1/C2 | 各exact tupleの実Target/Build/Call/ExpectedだけQualified。Pairwise全被覆でも未実行高次組合せは保証なし（#3267） |
| **AS-12** | CLI`view`のread-only広告と、実行Optionに依存した書込みの対照 | Command広告だけで実Effectを保証しない。Invocation variant/実副作用を観測（#3182） |

**実験順**：AS-03（既存scope隔離を壊さない）→AS-01/02（D1の中心）→AS-04/05（Authority/Scope）→AS-06/07（情報完全性）→AS-08/09（Traitと物理Build）→AS-10/11/12（横断）。既存#1180 Scenarioと#2514独立Expectedを参照し、本票の行を新しいScenario SSOTとして使わない。特にAS-02は**出力一致では見逃す偽陽性**を直接反証する。

## 4. 横断計画との依存関係・証拠の受け渡し

```text
#1251 D0: actual source/import & scope audit
         |
         v
#951/#1016: exact selection and typed Provider binding
         |
         v
#1158/#1824: CLI and Studio separate actual call-sites
         |        (AS-01/02 + host deny)
         v
#3262: required context and qualified read completeness
         |
         +--> #3278: Trait required fulfillment
         +--> #3267: Verified BOM exact tuple
         +--> #3270: Data Workbench read & schema-apply
         +--> #3274: HTTP error graph projection
         +--> #2831: actual CLI/studio self-build
```

**必要条件の方向に注意**：D1の基本的な実呼出を成立させるために、完全な#3262 Design Context Query、正式Trait汎用Schema、検証済BOM製品化、実DB DDL、Studio全GUIを先に完了させない。後段で同じPort/Contract/Evidenceを圧力試験に使う。逆に、D1だけでSource/Contextの完全性、BOMの組合せ資格、Current Grant、実DB/Target効果が証明されたとも言わない。

**次の実装変更に対して要求するEvidence**：exact GitHub commit・実Go toolchain・実ビルド対象・Source closure・Contract/Provider Selection/Scope/Epoch・CLI/Studio call-site・actual Provider entry・Result/Failure・Consumer receipt・Authority/Disclosure decision・独立Expectedを1 Attemptへ結ぶ。実装者が提供できない列は`unknown/not-run`で保全し、空欄をPASSにしない。#2831の再帰Self-build L1/L2/L3とは別資格とする。

## 5. 状態と採否

この時点では**静的コード検査のみ**。Scopeの既存拒否、Studio/CLIの直呼出、Operationのfingerprint項目、Compositionの既存APIは確認したが、AS-01〜12の新規実行はしていない。並行するDraft [#3280](https://github.com/hat47x/tei-platform/pull/3280)（Desktop固有）、[#3282](https://github.com/hat47x/tei-platform/pull/3282)（Trait）、[#3292](https://github.com/hat47x/tei-platform/pull/3292)（Master PlanのD1）と、提案内容を分担し、先行Draftの完成を推論しない。旧`service_carrier`は今のままapplication-runtime専用のまま保持し、authoring専用Contract/Providerの実接続は未実装として扱う。既存Source/Meaning/Runtime AuthorityのOwnerと現在資格を保存し、**二重Contract/Registry/Host/Runner/Lockを追加しない**。
