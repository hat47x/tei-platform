# Core以外をPluginにする実装方式：CLI／StudioからApplication Runtimeまで同一基盤で検証する

- **状態**：方式設計候補・実装移行計画（2026-11-07）。この文書で正式な新Contract/ABI、全Plugin移行、実Target適合、Ownerの業務意味受理が完了したとは扱わない。
- **静的確認基準**：`hat47x/tei-platform` main `358b8cbf55d825de118b15dd590b892b94f5f24e`。GitHub上の現在のソースと既存設計を照合。ローカルcheckoutのGo/Node実行・実バイナリ構築・CLI/StudioのE2E実行は**NOT RUN**。
- **既存方針**：[ADR-0032](../adr/0032-composition-kernel-replacement.md)（Contract中心）、[ADR-0033](../adr/0033-core-and-plugin-subprojects.md)（サブプロジェクト）、[ADR-0043](../adr/0043-target-native-artifacts-without-tei-host.md)（共通Host非必須）、[ADR-0044](../adr/0044-authentication-not-a-core-concern.md)（認証方式はCore外）。
- **責務の既存Owner**：[#951](https://github.com/hat47x/tei-platform/issues/951) Contract/Resolver、[#1016](https://github.com/hat47x/tei-platform/issues/1016) Composition、[#1251](https://github.com/hat47x/tei-platform/issues/1251) 移管、[#1158](https://github.com/hat47x/tei-platform/issues/1158) UI Adapter、[#1824](https://github.com/hat47x/tei-platform/issues/1824) Studio、[#3262](https://github.com/hat47x/tei-platform/issues/3262) Design Context、[#1180](https://github.com/hat47x/tei-platform/issues/1180)/[#2514](https://github.com/hat47x/tei-platform/issues/2514) Scenario/Expected、[#943](https://github.com/hat47x/tei-platform/issues/943)/[#1097](https://github.com/hat47x/tei-platform/issues/1097)/[#2600](https://github.com/hat47x/tei-platform/issues/2600) Build、[#3267](https://github.com/hat47x/tei-platform/issues/3267) BOM、[#2831](https://github.com/hat47x/tei-platform/issues/2831) 全体Self-build。
- **本書の限定責務**：現行実装のFit/Gap、開発環境Pluginを実アプリPluginの試金石とする構成方法、先行縦切りの検査条件を接続する。**第二のResolver、Lock、Runner、Runtime Authority、Source/Meaning正本、Plugin Registryを作らない**。

## 1. 現行実装の静的監査：目標とはまだ異なる

| 観点 | 現行Mainで確認できる事実 | 目標へのGap／変更の入口 |
| --- | --- | --- |
| Coreの物理境界 | `core/subject`、`core/identity`、`core/effect`、`core/generation`への実ソース移管と旧Aliasが存在する。`core/README.md`はCore単独製品ビルド未成立と明記。 | 基本型や世代Routerの抽出は先行しているが、全Contract/Resolution/CanonicalがCoreへ収束したとはいえない。後続の実依存検査が必要。 |
| Legacy Plugin定義 | `internal/plugin/model.go:16-45`に固定`CapabilityKind`と`RuntimeKind = builtin / process`。`Definition.Requires`はPlugin ID中心。 | 新契約追加時のCore/旧Pluginモデル変更を招き得る。新しいService Contract/Provider主体に移し、Legacyは移行用read-only対応に限定。 |
| Legacy Resolver | `internal/plugin/resolve.go:16-36,67-123`はPackage ID＋compatible majorで候補を解決し、環境のPlugin/Bundleを走査する。 | Target/Scope/Shape/Effect/ConformanceのContract選択と同一ではない。旧Resolverと新Resolverを二重正本で残さない。 |
| 新Composite試作 | `internal/composition/registry.go`にContract/Provider/Scope、`selection.go`にSelection/Epochがある。 | 実製品のすべてのCLI/Studio操作へ接続されたとは証明されていない。Registry試作の存在から製品用汎用Runtime完了を推論しない。 |
| 移行Carrier | `internal/plugin/runtime/service_carrier.go`は契約選択を既存Capabilityに束縛し、現在のHost Acceptanceを呼出時に検査するが、将来のContract Resolver自体ではない。 | 移行時の互換経路として有用。旧Carrierを標準的なUniversal ABIに昇格しない。Shape・Publisher Authority・Conformanceを独立に確認。 |
| CLI | `cmd/tei/cli_commands.go:10-93`の静的`acceptedCommands`と`init`追記、`cmd/tei/main.go`の分岐ディスパッチを利用。`cli_contract.go:42-67`は選択されたread-onlyの手書きCatalog。 | CLI機能は現状、共通Contract/Providerから導出されたPlugin Contributionではない。最初は既存コマンドを保持しつつ、一つのread-only操作を同一Provider経由へ切替。 |
| Studio | `cmd/tei-studio/main.go:27-45`のWails `StudioBridge`はList/Catalog/Validate/Inspect/ObserveXeadを手書き公開。`internal/studiohost/readonly.go`にTyped Service/ReadOnly構成あり。`cmd/tei-studio/go.mod`は別のWails依存モジュール。 | 型付きAdapter分離は足場になるが、Window/Editor/Commandの標準Plugin Contributionと実Consumer同値は未実証。Wails Bridgeを業務契約正本にしない。 |
| Plugins配置 | `plugins/README.md`は参照例の存在と「製品用Pluginへの一括移行未実施」を明記。参照Go/JS/Process実装がある。 | 通常の標準CLI/Studio機能をPluginとして構築・選択・試験・利用する一連の成功例が必要。 |
| Plugin runtime | `internal/plugin/runtime/adapters.go`はDeploymentExecutor/Observation/Evidence等の固定能力を既存Registry.Invokeに接続。 | Provider実行・拒否・来歴の資産として活用。ただし種別追加ごとにAdapter固定追加する構成を標準にしない。 |
| 製品状態の線引き | `docs/status.md`のStudio PoCとComposition Referenceは、実測版・局所的範囲を限定している。 | PoC成功や`validate.ok`は「CLIとStudioがPlugin基盤で動く」の証明ではない。 |

**監査結論**：目標思想と基礎部品はあるが、**現在のCLIとStudioの通常機能は、原則としてCore以外をPluginにした製品形態ではない**。一括ソース移動ではなく、代表操作の実経路を新しいContract/Provider/Compositionへ一本ずつ切り替える。

## 2. 物理構成より先に固定する論理境界

1. **最小Core**：論理Subject/Ref、標準Record/Relation/Graphの最小表現とSource Location/Revision、Contractを参照・比較する汎用契約、Scope/Effectの不変条件、同じ入力から一意に選択するResolverへの最小入口、結果の失敗/未知状態と証拠参照。不変条件がない実務機能を「便宜上Core」に追加しない。現行の`core/`がこれら全てを実装済みという主張ではない。
2. **Core外の標準Plugin**：Role/DSL/Parser/Validator/Operator/Query/Data Provider/Generator/Builder、Formatter/Test/Driver Adapter、CLI command surface/Studio inspector・editor・view、MCP/Web/REST/API・Desktop surface、認証、Logger、業務Rule/Workflow、Target実行支援。どれも同一`Service Contract → Provider → Composition`で接続し、固定`CapabilityKind`列挙を新規追加しない。
3. **実行環境・信頼境界**：CLI process、Wails Studio、Web server、.NET/Javaの既存Host、生成Goバイナリは**Consumer/Target**であり、共通のTEI常駐Hostを全てに要求しない。OS/Hostの権限実装は適切な境界が所有し、Pluginによる自己許可を拒否する。
4. **Bootstrap例外の明示**：`cmd/tei`とStudioの起動点は選択されたPlugin集合をロードまたは静的結合して最初の操作を公開する小さな実行入口として残してよい。ただしCLIコマンド一覧/業務操作/画面メニューをもう一つの意味的正本として埋め込まない。起動失敗・構成診断の最小表示をBootstrapに認める。
5. **Plugin ≠ Class ≠ Package ≠ Implementation ≠ Instance ≠ Build Artifact**：Service Contract/Version、Provider/Implementation ID、Plugin Package/Bundle、Component型/Instance、構成上の`selected`、実Buildの`linked`、実呼出の`invoked`、現在の`authorized`を別のRef/Relationに置く。二つの論理Pluginを単一Goバイナリにまとめても、権限と効果まで共通化しない。

### Everything is DataとTraitの合成

- Plugin定義、Contract、Parameter、Invocation Offer、UI Contribution、Test Pattern、Scenario、Build Definitionを、標準データ構造上のRoleを持つデータとして読めるようにする。Markdown/表/Graph表示は同一意味構造へのProjection。
- **原子的Trait**は用途別の契約要求として合成する（例：`Discover` / `Read` / `Mutate` / `Validate` / `Render` / `Observe`）。これは列挙の提案ではなく粒度の例。各Traitは単一責務・前提条件・Failure/Effect・Scopeを持ち、明示された実装を必須とするか検証済みの標準Providerで充足する。単なるHintであればHintと明示し、未実装の実行能力として発見させない。
- 無関係なTraitは採用を強制しない。例：Read-only logはDDL Traitを持たず、認証が必要なSaaSだけAuthentication契約をCompositionに要求する。Traitの宣言だけではContract Publisher資格、Provider適合、Host Grantを得られない。
- 「Plugin自身でContractを公開できる」と「Contractが公的に認められた」は異なる。Open/unpromised APIへの下流Adapterを許し、変更責任は明示した依存側へ帰属させる。

## 3. 同一Pluginが開発環境とアプリケーションの両方で使われる形

```text
             Canonical / Source / Design Context (#3262)
                            |
       Composition Profile → Contract Resolver (#1016/#951)
                            |
              exact Provider/Artifact Lock
                            |
               typed Application Operation
                   /        |         \
      CLI Projection   Studio Projection    Runtime Projection
       (command)       (menu/panel/editor)   (Web/Go/.NET/Java)
                   \        |         /
                  same Operation/Diagnostic/Result
                           |
              Scenario / Test Evidence (#1180/#2514)
                           |
           real Build/Artifact/Invocation evidence
```

**共通なのは意味操作と契約**であり、UIそのものでも、各TargetのOS認可でもない。各Surfaceは同じ型付きRequest/Result、Source/Subject、Failure taxonomy、Expected Revision、必要Current Grantを渡し、Target固有の制約を追加する。CLI経由の成功でStudioのUI操作や外部Hostの権限・業務副作用を合格にしない。

- **一つの`Operation Offer`**（概念名、正式Schema未採用）はContract/Providerと公開可能な入力/出力Schema、read/mutate/execute等のEffect、Scope、参照要求、欠落・未許可理由を記述する。#951/#1016の選択と現在のHost権限が使えるOfferだけ提示する。
- **CLI Surface Plugin**はOperation Offerをコマンド・引数・stdout JSON/stderr・exit code・Help/Discoveryへ投影する。固定`acceptedCommands`の隣に新Registryを増やすのではなく、切替対象の一操作を従来分岐から共通Dispatchへ移し、互換シェルを縮退する。ユーザーの低負荷を優先し、全操作に冗長な`plugin invoke`を要求しない。
- **Studio Surface Plugin**は同じOfferと標準表現型（table/tree/graph/formなど）をSurface Contribution（メニュー、パネル、Inspector、Editor）へ投影する。最初はWails `StudioBridge`がTyped Operationを委譲する形でよく、Pluginが任意JavaScript/ネイティブ権限を暗黙取得しない。Surfaceそのものと操作対象インスタンス、ユーザー入力、保存対象を明確にする。
- **MCP Surface Plugin**は同じOperationをTools/Resourcesに必要時だけ公開する。CLI呼出しをハーネスが既に利用できるので、TEI専用MCP Serverを必須前提としない。開発者用MCP Controlと業務AppのMCP Targetは異なる。
- **生成Application**は開発用Surfaceを通常含まず、必要なOperator/Role/ProviderとTarget Adapterのみを選択・生成する。単一Goバイナリが既定、Java/.NET等では対象Hostの認証/権限/ライフサイクルへ接続する。開発用外部制御・テスト専用資産はproduction Buildから実際に除外する（#2391）。

## 4. 具体的な移行縦切り：手書きCLIとStudioから始める

| Phase | 置き換える実物／検証作業 | 成功条件 | Owner |
| --- | --- | --- | --- |
| **D0 監査・非侵襲計測** | `cmd/tei`のcommand分岐、Studio `Bridge`→`studiohost`→`operations`、`workspace.Load`、旧`plugin.ResolveEnvironment`と新`composition`の呼出し依存Graphを実Goから計測。 | Go importと実Request/Reader/Providerの対応表、Target別実Build closure、暗黙`init`/embed/旧KindとUnknownを列挙。 | #1251 #951 #1158 |
| **D1 Read-onlyの同一Service** | 既存`model catalog`または`inspect`の一方を、既存のTyped Serviceから**1 Contract + 1 Provider**として提示。CLI既存互換経路とStudioから同じ操作を呼ぶ。 | 同じSubject/Source/Required Closure/欠落/部分/Revisionが両Surfaceで一致。Pluginなし・不一致・Provider二重競合は失敗。 | #3262 #951 #1158 #1824 |
| **D2 Pluginとしての開発者機能** | Inspector/Validation/Graph Viewなどの代表Contributionを`plugins/<name>/`のサブプロジェクトへ。CLI Help/DiscoveryとStudioメニューを選択されたOfferから導出。 | 追加Pluginに`core/`、手書き`acceptedCommands`、`StudioBridge`の意味分岐変更が不要。未選択Pluginが機能一覧・実Buildに混入しない。 | #1251 #1158 #1824 #1124 |
| **D3 Test Pluginのドッグフーディング** | 同じContractを使うScenario Runner Adapter、Fixture/Data Generator、Coverage/Result ProjectionをPlugin化し、CLIから検証してStudioでもレポートを読む。 | #1180 Scenarioと#2514 Independent Expectedが唯一の正本。CLI/Studioと本体実行のTest Runner/受入基準が同じ、Actual Runner/Target固有E2Eは区別。 | #1180 #2514 #3262 |
| **D4 実データの圧力** | Data Source Workbench（SQLite等のread/structure inspect→DDL preview→sandbox apply）を同じOperationに載せる。 | #3270のread-only/DDL/Schema Scope、実Provider Grant、Effect Receiptと独立再読取を区別。CLI経由の結果とStudio経由の表示が整合。 | #3270 #2622 #1174 |
| **D5 Runtime向け再利用** | 開発ツールで使用したValidator/Query/Providerまたは同一Contractを実装する別Providerを**生成業務アプリのGo静的Target**へ組込み。後で外部HostへPressure。 | CLIと実Applicationの意味契約は一致。Link/Load/Execution/Host Grantが独立にPASS。開発Surfaceは除去。 | #1121 #1016 #943 #2390 |
| **D6 Full self-build** | 構成されたPlugin集合とBuild GraphでCLI、Studio、業務Runtimeの実Artifactを記述・再構築。 | #2831 stage0 seed→stage1→stage2/入力閉包/byte比較/false cache hit/独立Expected。1PluginのSelf-buildを全製品Self-buildと呼ばない。 | #2831 #943 #2600 |

D1は**新たな汎用Plugin Manager/Runnerを先に全面実装しない**ための最小実験。D2以降の採否はD1の「新コア変更件数、実Consumerの同値性、変更量」に基づいて決める。

### D1の候補Contract（図解用データ。正式Schemaでも既存CLIコマンドでもない）

```yaml
operation:
  ref: tei.std.design.inspect   # 名前は候補
  request: {subject: SubjectRef, expectedRevision: RevisionRef, requiredScope: ScopeRef}
  result: {data: Record, relations: RelationSet, diagnostics: DiagnosticSet, completeness: Completeness}
  effect: read-only
  requiredTraits: [Discover, Read]
  failures: [not-found, not-loaded, permission-denied, ambiguous, stale, partial, unavailable]
provides:
  - provider: existing-operations-inspect-adapter
    realization: in-process-static
surfaces:
  - cli: inspect
  - studio: subject-inspector
tests:
  scenarioRef: existing-driver-scenario
  expectedRef: independently-maintained-expected
```

実装時には`SubjectRef/RevisionRef`の正式型とRequired Closureを既存の意味正本へ対応させる。ここに仮の正本を新設しない。Operationがread-onlyでもSource metadataの開示権限をHostに問い合わせる。

## 5. 方式選定：Static/Process/Target-nativeは同じ契約を満たす代替実現

| 方式 | 主用途・優先 | 制限と受入 |
| --- | --- | --- |
| **Static Go composite** | CLI・Studioバックエンド・単独Go実行物の既定。選択済みPluginのみBuild/Link。 | Go package/import/`init`/embed経由の未選択実装混入を実Buildで検査。選択済みProvider変更は再ビルド。 |
| **Target-native Library/Generated code** | 外部.NET/Java Host、Web/Node/Wasmなど。 | 契約同一性と形状・失敗・Effectの変換を検証し、Host固有のSecurity/Transactionを二重化しない。 |
| **Independent process** | 信頼境界、独立更新、異言語Providerが必要なTarget。 | handshake/protocol版、resource/permission、cancellation、timeout、出力欠落、部分作用、クラッシュ・復帰をテスト。Isolation≠自動的な権限隔離。 |
| **Dynamic/Generation switch** | 長寿命サーバー等、更新価値のあるTargetだけ。 | in-flight pin/drainとEffect release、Provider切替失敗時の旧世代継続。静的CLIへWatcherや常駐Hostを強制しない。 |

同じPluginの意味構造を使って、ビルド時だけ必要なBuilderと実Runtimeに必要なProviderを**別の依存閉包**に置く。Package/BOMは実装の束、選択済みexact Lockは契約解決結果、Build Artifactは物理成果物、Verified BOMは実組合せの限定的試験証拠。#3267の2×2実Target Matrixを共有し、単独Plugin PASSから複合PASSを合成しない。

## 6. 受入負例：開発ツール経路を実Runtimeの検査器にする

| ID | 反例／観測 | 独立Expected |
| --- | --- | --- |
| **EDP-01** | 新Inspector Pluginを追加 | `core/`と`cmd/tei`のコマンド分岐・Studioの業務分岐変更数0でCLIとStudioにOfferが現れる。未対応Targetに現れない。 |
| **EDP-02** | Plugin選択を外す／同Contract二Providerを宣言 | 消えた機能を旧手書き経路から実行できない。競合をロード順/latestで自動選択しない。 |
| **EDP-03** | 同一SubjectでCLI JSONとStudio Operationを比較 | Source/Meaningの採否/Null/未取得/partial/Revision/Required Closureが一致し、UI表示の省略で完全性が変化しない。 |
| **EDP-04** | Pluginデータを変更せずCLI/Studio UIだけ変更 | Canonical/Accepted Meaning/Runtime Providerは不必要に失効しない。ただしSurface Build Artifact/実入力の変更は検査する。 |
| **EDP-05** | 同Bundleに未選択Providerを追加・更新 | Candidate Catalogは変わり得るが選択済みBuild closureと実Link byteは、toolchain固定条件では不変。単なる`-plugin` flagでは除去証拠にしない。 |
| **EDP-06** | Contract ID/出力型は同じでNULL/Effect/Scopeだけ変更 | 旧ClientのPASSやBOMを使い回さず、該当ConsumerのConformance/Meaningを再検査。 |
| **EDP-07** | 実行前Grant失効・Cross-Tenant・同Bundleの別ProviderにGrantなし | Studio/CLI/生成AppのいずれもHostの実行直前Checkを迂回しない。 |
| **EDP-08** | stdoutの8,192 byte切断/ページ間Source変更/Response欠落 | Command exit0やUI描画を`complete`と誤認しない。snapshotに束縛したcontinuationとend markerが必要。 |
| **EDP-09** | write Provider実行後にCommit返信断 | success/failedに決め打ちせずunknownとし、同一Operation identityで確認・独立再観測。無条件再送不可。 |
| **EDP-10** | 同一ContractのProvider二版×Consumer二版 | 各exact Lock+Target+実Artifact+Independent Expectedに対する同一Attemptの結果のみVerifiedとし、未実施組合せを候補に留める。 |
| **EDP-11** | 開発用外部操作PluginをTest Buildだけ選択 | 本番Artifact/route/module/listener/debug assetsに専用外部制御面が残らない。通常UI/Accessibilityは維持する。 |
| **EDP-12** | Coreを独立Buildし、Pluginなしの最小構成を生成 | Registryや一般Runtimeを失っても基本型/Source/診断が扱える。特定Database/IdP/Wails/LoggerにCoreが逆依存しない。 |

**計測**：新Pluginを実装してCLI/Studioで発見・試験するまでの操作数と差分行数、Core修正件数、手書きAdapter追加件数、Provider競合の検知率、現Build closureとモデル上の宣言との差、Plugin未選択時のBinary size/startup/memory、CLIとStudioの同一契約テスト結果、未観測/過大PASS件数を記録する。生成AIハーネスが`inspect → scaffold → test → build → consume → inspect evidence`を**CLIのみ**で達成できることを先に検証する（コマンド名は実装が出現するまで候補扱い）。

## 7. 移行時の契約と停止条件

- 旧`CapabilityKind`で動く機能を一括削除しない。**Consumer単位**で旧経路・新Providerの対照テスト→実Consumer切替→旧経路削除/限定read-only変換。新旧を並列の正本にしない。
- TEI固有の常駐Hostや一つの万能IPC/ABIを、全Targetの前提にしない。StudioのWails Service Adapterと生成Web/Desktop ApplicationのTarget同値を混同しない。
- 実装を`plugins/`へ移したこと、Plugin Descriptorを作ったこと、Command Helpに表示したこと、Mock Providerが応答したことだけで**実Plugin消費・Conformance・Host Effect**を成功扱いにしない。
- D1を最小Go Targetで実証した後、D2で追加した抽象が開発負荷と安全上の効果を改善しなければ、Interface/Roleを縮小・撤回する。意味・認可・Buildの新しい第二基盤は採用しない。
- 本書に含むCLI/Studio共通Plugin実利用、Generated App再利用、実Build Closureテスト、Studio E2E、Self-build Stage試験は**NOT RUN**。独立Expectedとexact-head CI、実Target負例を実施してから段階的にQualificationを判断する。

## 参考実装からの限定的な示唆

- [VS Code Extension API](https://code.visualstudio.com/api/)は標準機能のExtension化、[Extension Host](https://code.visualstudio.com/api/advanced-topics/extension-host)はUI/Workspace/Remoteという実行場所の分離・Activationを参考にできる。ただしTEIは同じExtension Hostを義務付けない。
- [HashiCorp go-plugin](https://github.com/hashicorp/go-plugin)はProcess/RPC・Protocol version・Handshakeの実装例。これを全Pluginの義務ABIにしない。
- [Gradle TestKit](https://docs.gradle.org/current/userguide/test_kit.html)はPluginを実Consumer（実Build）で試す機能試験の参考。TEIは既存Driver ScenarioとIndependent Expected、actual Build/Runnerに接続し、新Runnerを作らない。


---

## 8. Traitの原子性と実装義務：Extension TypeからPlugin/BOM/CLI/Studioへ（2026-11-07追補）

**位置づけ**：他レーンで具体化した「拡張型を明示すると、デフォルト実装が選ばれるか、手動実装が必須と診断される」「Traitは合成できるよう原子的にする」という要求を、上記 D1〜D6 の実証条件へ接続する。正式Trait DSL、Core enum、特定実装方式の採用ではない。既存[#2619](https://github.com/hat47x/tei-platform/issues/2619)/[#2620](https://github.com/hat47x/tei-platform/issues/2620) のEverything is Data・Semantic Role、[#951](https://github.com/hat47x/tei-platform/issues/951) のContract/Resolver、[#2507](https://github.com/hat47x/tei-platform/issues/2507) の型付きFunction Pattern適合、[#3267](https://github.com/hat47x/tei-platform/issues/3267) のVerified BOMを第二の正本なしに再利用する。

### 8.1 ContractとしてのTraitとHintを区別する

1. **一Trait＝一つの独立して判定できる義務**。実行要求・前提・入力/出力・失敗/未知・Effect・適用Scope/Target・Conformance Obligation・Contract identity/versionを追跡できる粒度で定義する。「DB全部」「完全なCache」「編集可能UI全体」という巨大なTraitを作らず、単一の意味要求から複数Traitを合成する。Trait名をCore固定enumとして追加しない。
2. **明示的なExtension Type/Roleの指定は、原則として行動を変える**。宣言したTrait義務が (a)適格な標準Provider/Generatorによる既定の実装に束縛される、または (b)対象Consumer/Targetでの手動実装をRequiredとして診断・検証される。実装が存在しないのに型ラベルだけ有効化することを標準の成功状態にしない。
3. **Hint/Advisoryは別の弱い関係**。提案・最適化ヒントだけの場合は実装義務・実行可能性を付与しないことを明記し、discover/inspect/Design Queryで実効Traitと区別する。利用者が「Traitを明示すれば機能が実装される」と期待したのに、単なる色付けで終わる状態を避ける。
4. **Default implementationはContract定義と別**。Defaultがあっても、選択Profile/Target/Scopeに適格で、実装Version/Artifact/入力Closureと独立Conformanceを確認できる場合だけ解決可能。未実装、Provider不在、競合、循環、Target非対応を任意の「空成功」「無断fallback」「最新Provider後勝ち」にしない。
5. **手動実装必須時はどこで止まるかを明示**。候補編集・探索時は未解決を診断として保持できるが、対象TargetのBuild/Published Offer/実Invocationに要求Traitが必要なら、実装不足を不適格として停止する。未必要のOptional Traitまで全世界に一律実装を求めない。
6. **合成衝突は解決順序で隠さない**。二Traitが同じOperation名・異なるEffect/Null/Precision/Unit/Scopeを要求した場合、単に最後に現れた宣言を選ばず、Conflictまたは明示的なBridge/Adapter/Bindingを要求する。多重参照した同一Traitは同じ意味IDに正規化できるが、異なる契約を同一視しない。
7. **Traitの静的成立と実資格は別**。Trait要求 → 具体Provider解決 → 実Builder/Link/Load → 実Dispatcher/Invocation → 現Host/Provider Grant → Effect/Receipt → 独立Observation/Consumer受領を分離。Trait実装と適合試験はAuthority、現在Disclosure、Business Acceptance、業務確定値ではない。

### 8.2 Everything is Dataの候補表現（正式ABIは作らない）

新しい専用ASTや独立Registryを設けず、既存の標準Data/Ref/Relation上に以下の読み取り関係を投影できるかを試す。

| Subject | 最小のRelationと証拠 | 誤解してはいけないこと |
| --- | --- | --- |
| Trait Contract | stable ref/version、単一義務、requires/obliges、Scope、Effect/Failure、Conformance ref | 宣言が実Providerや実権限を生むわけではない |
| Data/Function/Representation Extension Type | どのTraitをどの文脈で要求し、どのBindingを使うか | Target/Function contextなしの万能「編集可」「読取可」は不可 |
| Default/Manual Implementation | provides/implements、Contract fingerprint、Target/Profile、実選択根拠、actual artifact | ContractにDefaultを同梱しただけで実Byteが選ばれたことにしない |
| Composition | requires closure、excludes、Conflict、resolved Lock、選択理由とunknown | 依存未探索／ロード順後勝ちを成功扱いしない |
| Test/Proof projection | Requirement/Scenario ID、Independent Expected、Mutant、实际実行Actor/Attempt、Coverage/Claim ceiling | 個別Traitのテスト合格から合成BOMの合格を作らない |

この構造は**編集用言語構文の決定ではない**。日常DSLにGoのstruct、func、package、型引数の露出を強制せず、ドメイン専門家が読む順序（対象→付加する意味/義務→既定実装または実装要求→テスト）を最優先する。

### 8.3 具体的なドッグフーディング：実際の効果を宣言へ照合する

最初は、デフォルト実装を持つread-onlyな**設計Inspect/Discover**と、標準実装が存在しない**変更Intent/Schema Alter等**の対比を使う。前者も「外部開示許可」を伴わず、後者は承認されたProvider/Hostの実作用がない限り実行不能にする。Cacheでは Atomic CAS、Snapshot Consistency、Read、Purgeを用途上必要な原子Trait候補に分け、単なる保存成功と整合性・確定値の意味を混ぜない（[#3271](https://github.com/hat47x/tei-platform/issues/3271)）。

**現行実装から得た反例**：[CLI #3182](https://github.com/hat47x/tei-platform/issues/3182)では、command discoveryがviewをread-onlyと広告する一方、明示execute+operation-log/lifecycle-stateが指定されるとローカルファイルへ実際に書き込む。同一Command名の静的なEffectラベルだけでは、特定引数・モードにおける「業務データ操作のEffect」と「ToolのFilesystem出力」を正確に表せない。今回Traitを導入するなら、**Operation Variant/Invocation Context相対のEffectの正確性を必須テストとする**。勝手にread-onlyというラベルを消す/書込権限を変える仕様変更は本追補では行わず、#3182のOwnerが決める。

### 8.4 最小の独立反証ケース（TRF-01〜12、今回NOT RUN）

| ID | 対照入力 | 要求される独立Expected |
| --- | --- | --- |
| TRF-01 Default binding | Extension Typeが1原子Traitを要求、対応Defaultがexact Targetで選択できる | 明示Version/実装Ref/Target、選択理由が出る。契約宣言だけのProvider存在・実InvocationをPASSにしない |
| TRF-02 Manual required | 同Traitを要求、Defaultなし／手動実装欠落 | 設計候補段階ではunresolved、必要TargetのBuild/Invoke前に実装不足を機械診断。無断no-opなし |
| TRF-03 Advisory only | Hintのみ付けたSubject | Hintが機能Offer/必須Plugin/Runtime Grantを増やさない |
| TRF-04 Atomic composition | ReadとValidateを別々に提供し、両方を要求 | 両方の義務が独立に充足、片方欠落時は部分不適格。無関係Traitの実装は強制しない |
| TRF-05 Collision | 同名Operationに異なるEffect/NULL/Unit要求、並びだけ反転 | 明示Bridge/Conflictを要求。列挙順・Import順・last-winsで結果が変化しない |
| TRF-06 Real effect | #3182のview previewとexecute+named output、CLIとStudioの同契約 | 宣言されたread-onlyと実Filesystem writeの違いを診断。静的Command EffectだけでNo Writeを保証しない |
| TRF-07 Provider change | 同Version別Artifact、古いBOMと新Default実装 | 現Lock/実Artifact/Input Closureと独立適合試験が一致するまでqualifiedにしない |
| TRF-08 Target mismatch | Go静的でDefaultあり、外部Host/Browserでは実装・Grant不足 | 代替候補またはunsupported。黙ったGo実装の移植・Host許可の偽装なし |
| TRF-09 Read/Write split | Readable Log Data SourceにSchema Dropを要求 | Unsupportedまたは明示外部Providerの実装要求。readだけでDDL Offerを生成しない |
| TRF-10 Reversible edit | 表への編集写像は一意、集約Graph/SHO文は多対一 | 一意でないTargetへの自動Writeを拒否。元Source navigate/差分Previewを維持（#2497） |
| TRF-11 Cache/Snapshot | Local CASのDefault・Remote Providerによる手動実装、TTL/Grant変更 | Byte hit、Snapshot整合性、Effect、現在開示の別資格。実CAS保証未観測なら失敗/unknown（#3271） |
| TRF-12 Verified composition | Trait実装Provider A1/A2 × Consumer B1/B2 | 2×2 exact tupleごとに実Link/Invoke+独立Expected。未実行行をVerified BOMへ含めない（#3267） |

### 8.5 既存D0〜D6へ載せる実装順と採否

- **D0/現状棚卸し**：実Core/Plugin/Composition/CLI/Studioで既に存在するTrait相当の要求/実装/Constraintを対応づける。新Core enumを入れる前に**標準DataとしてのRelation+Constraintだけで表現可能か**比較。未確認Source/Typed Request/Builder inputはunknownのまま。
- **D1の小さな実験**：TRF-01/02/04/06を先行。既存Inspect read-only Operationを一つだけ共有Contract/Providerに束縛し、無Defaultの場合は手動実装不在を検出する。CLI実argv/実stdout・Studio実Service呼出を独立Expectedと照合。現実に操作未実装なら未実装を明示し、DocumentだけでPASSにしない。
- **D2/D3**：表示可能なTrait義務・Missing implementation・Conflict・Target非対応を#3262 Design Query→#1180 Scenario/独立Expectedの候補へ、Source span/Revision付きで投影。AIは「必要な実装を作るか・組合せるか」を最少の操作で選べること。CandidateのExpectedを独立の正解としない。
- **D4/D5**：Data Source WorkbenchのReadとDDL、Cache Snapshot Provider、Rendererの逆編集を異質なConsumerとして検証。選択済みProviderが実TargetでBuild/Invokeされた証拠と、Hostが与えた現在Grantを別票にする。
- **D6/BOM/セルフビルド**：原子Trait単独の緑とTrait合成/Plugin Version Matrixの緑、実TEI自身のStage2再入を分けて評価。#3267はComposition資格、#2831は真のSelf-build、#2600はCache再利用の資格を持ち、相互に自動認定しない。

**採否判断の計測軸**：新Trait 1件追加に要するCore変更件数（理想0）、Extension記述の構文ノイズ、必須未実装の検出率、合成衝突の偽PASS/過剰拒否、Tool/Runtimeの実Effect広告の正確性、CLI/Studio/Targetの同一契約適合、Provider別手動コード量、Target/Lock差分による再試験範囲。少なくとも2種類の異質なConsumerと正負1件ずつを先行する。

**現在の資格**：本節はIssue・現行Code・他チャット方針を照合した設計計画であり、Trait公開Schema/DSL、実Plugin移行、TRF-01〜12、新しいGo/CLI/Studio/Provider試験、Owner正式採用は **NOT RUN／未採用**。本DocのD1試験を通したときだけ、実装方式の判断を更新する。
