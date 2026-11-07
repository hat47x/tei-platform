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

**現行Goの必須Fit/Gap（2026-11-07）**：CLI/Studioの`inspect`はまだ`operations.InspectSubject`の直接呼出であり、`internal/plugin/runtime/service_carrier.go`の移行Carrierは`application-runtime` scope専用で`authoring-tool`を拒否する。D1実装でその拒否を文字列置換で緩めず、既存`composition`のexact選択とHost所有Inspector Adapterに結線する。選択B／実呼出A・別Scope・Root差替え・未取得・Tool途中切断などの独立Expected、具体的な変更seamと根拠は[Authoring Inspector D1 Fit/Gap／AS-01〜12](../evidence/PLUGIN_FIRST_D1_AUTHORING_ROUTE_FIT_GAP_2026-11.md)に記載する。これは実装経路の追加監査であって、正本Contractや第二Runnerを新設しない。

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

## 8. 他チャット横断で確定する「同一プラグイン基盤」の正確な意味（2026-11-07深化）

**結論**：CLI／Studio／生成Applicationで共有するのは、Service Contract・Provider適格性・構成選択・Build/Artifact/Invocation/Evidenceの**仕組み**であり、同じ権限、同じProvider、同じDeployment Lock、同じ表示機能、同じSource受理を強制することではない。これは #951 が既に分けているSemantic／Authoring／Build／Runtimeのscope、および #1016 の単一Composition入口候補を維持する判断である。

| 軸 | Authoring（CLI/Studio） | Build/Test | Application Runtime | 分ける理由 |
| --- | --- | --- | --- | --- |
| Subject/Meaning | Source Observation・Candidate・Accepted Meaningを参照 | Accepted SourceとScenario/Expectedを参照 | 導出されたFunctionと対象Dataを利用 | 同じSourceは未受理状態を含み得る |
| Provider選択 | Authoring Source/Parser/Inspector、read-onlyを含む | Generator/Builder/Test Adapter | Query/Rule/Data/API Provider | 同Contract IDがあっても役割とscopeは別 |
| Lock/Artifact | Tooling Profileのexact選択、CLI/Studioの実成果物 | Build-only入力閉包、Runner/Toolchain | Target/Hostに適合するexact選択 | 意味版・Builder版・実Binary・現在Grantは別 |
| Authority | Workspace操作権限、Source秘匿、編集Intent | テスト用隔離・Artifact検査とEvidence | 現Host/Principal/Resource Grant、Effect | デバッグ可 ≠ 本番書込み可 |
| Presentation | Command/Inspector/Editor/Help | Matrix/Report/Trace | Form/Web/API/Native | Surface Contribution自体は意味・作用を所有しない |
| 実証上限 | 操作表示＋実Tool消費 | 実Build/実Runnerと独立Expected | 実Load/Invoke/Effect/独立再観測 | 局所PASSを横断PASSに足さない |

**四つのComposition対象**（Authoring / Build / Test / Runtime）は、**四つのResolverを意味しない**。既存 #951/#1016 の共通解決・固定原理を各Scopeへ適用する。Accepted Semantic SetはEnvironmentのRuntime Provider選択によって書き換えられない。Build-only依存、Test-only依存、Authoring Plugin、Runtime Providerは別の閉包として収集する。必要な間は一つのBundleが複数Scopeに対応してよいが、未選択側を実Artifactへ強制同梱しない。

**Core境界のブートストラップ問題**：全機能をPlugin化しても、そのPluginの記述を読む機構までPluginだけで自己発見することはできない。最小Coreには、型・Refと値/Record/Relationの基礎、限られた既知形式の構造読取、Source/Revisionと曖昧・未知・エラーの表現、契約照合・構成選択の安全な入口を残し得る。ただし特定業務DSL、全面的なMarkdown/SQL Parser、Graph Renderer、認証方式、DB接続、標準業務検証の実装はCoreに固定しない。「構造的に読める」ことと「Roleに従って意味解釈できる」ことを分離（#2620）。CoreがProvider不在でも診断・最低限のSource保全を行えることをD0で確認する。

**Pluginがなくても持続するデータ**：対応するParser/Editor/Validator/Rendererがdisable/uninstallされてもSource byte・Git履歴・明示的なユーザー修正は消さない。対応Role不明、型未解釈、Source未読込、開示不可を別表示。Source保存はMeaning受理ではなく、UI表示ができても実行許可ではない（#2044/#2620/#3262）。

## 9. Trait/RoleとPluginの合成は似ていて異なる：強制力の設計

他チャットのTrait原子性・Implementationの強制、およびEverything is Data #2619/#2620 を結ぶ。**Roleはデータに意味を与える解釈上の主張、Traitは対象が満たす小さな契約上の性質、Providerは具体的な実現**とし、互いを自動同一化しない。

| 宣言の種類 | できること | 検証と禁止事項 |
| --- | --- | --- |
| Required Trait | そのTraitを満たす実装の存在を要求する | 適合Providerが不足なら未実装診断。表示上のHintで代用しない |
| Default-backed Trait | 明示された標準実装を選択可能にする | Default選択も既存Resolverに従う。複数候補から無断先勝ちしない |
| Manual-implementation Trait | 利用者が指定Interfaceの実装・試験を提供するまで不成立 | AI生成のstubやメソッド名一致を完了扱いしない |
| Advisory Trait | Hint/補完/最適化候補として利用する | 「実行できる」「安全である」「対応済み」とDiscoveryで宣言しない |

**原子的粒度**：一つのTraitは観測可能な一責務と、必要なShape・Precondition・Effect/Failure/Scope・独立Conformanceを持つ。例としてSource `Read`、Schema `Inspect`、Data `Mutate`、Result `Render`、Execution `Observe`を**説明用**に分けられるが、固定Core enumではない。Trait合成は可能でも、`Read`を持つから`Mutate`やDDLを推論しない。互いに排他的なConstraintや同一Operationへ異なるEffect義務があれば診断。結合して初めて成立する振る舞いはBridge Role/Contractを明示し、個別Trait PASSを合成PASSにしない。

**三つの適合確認**：Structural（型・多重度・NULL/欠損の形式）、Semantic（意味・単位・精度・時点・拒否・Effect）、Operational（実Provider/Artifact/Target/現在Grant）を別観測。Traitを付けただけでDefault実装の物理収録、Semantic Acceptance、Host Authorityを得ることはない。

## 10. CLI・Studioで「同じProviderを実利用」の判定を厳密化

従来D1の`model catalog`／`inspect`を使う場合、**選択を通さずに共通`operations`のGo関数を直接呼んだだけ**では、UI Adapter共通化の実証に過ぎずPlugin dogfoodingではない。一方、手書きコマンドから毎回別Processを起動する実装も必須ではない。

**D1で実装者が提出する一つのContext表**：
1. Request主体：Subject kind+ID、Workspace/Source Revision、Required Closure、disclosure/Scope（#3262）。
2. Contract：version・fingerprint・Shape・Failure/Unknown/Effect・Trait適合・要求Context。契約の公開資格は#951。
3. Composition：どのTooling Profile/Targetが要求し、既存#951 ResolverでどのProviderを選んだか。未選択・複数候補・Scope不一致は停止。
4. Implementation/Build：Provider実装とartifact exact identity、実選択→コンパイル→実起動の閉包。宣言だけのロードを実消費と呼ばない（#943/#1251）。
5. Invocation：CLIとStudio各々の**実call-site→同じ選択済みProvider経路**、入力・結果の一致と実Invocationの来歴を保持。CLIがJSON化してStudioがそれを読んだだけでは独立Consumer二本とは呼ばない。
6. Result：stable data、diagnostic、not-loaded/partial/unknown、read/returned/effective scope、Tool受領complete。画面に表示しないフィールドも消滅扱いしない。
7. Claim：Source候補の意味と現在実行Authority、UI/Runner/外部API効果を分ける。

**操作の公開を三段階にする**：`declared contribution`（メタデータ発見）→ `selected & present in exact build/load`（利用可能な実体）→ `invoked with current authority`（実施）。VS CodeのContribution PointsはCommandやMenuの宣言と実ハンドラを分離しており、その構成上の示唆を取り込むが、VS CodeのExtension Host/ActivationをTEI全Targetへ義務付けない。発見時に未選択Pluginの任意コードを実行しない。CLIのHelp/Commands、StudioのMenu/Panel、MCP Tool schemaを**選択済みOfferのProjection**とする。コマンド名衝突、Stable Aliasの版不一致、Studio非表示条件、未許可の存在・件数を表示で漏らすケースは拒否する。

**静的配布での拡張性の実態**：Go単一バイナリの新しいProvider実装は、通常、選択されたソースをBuildへ含めて再ビルドする必要がある。CLI起動後にPluginファイルを置くだけで任意のGo実装を「自動ロード」できるとは約束しない。開発時に外部Processで利用するなら別の明示Realization/Protocol/実行権限の適格性が必要。静的実行ではCompositionで固定したPluginだけを生成されたimport/registration入口で結合し、`init`/blank import/埋込assetsによる無関係依存混入を測る。

**移行を二段に分割**：D1aは現行CLI/Studioの動作・Source/Result/Unknownを固定し、既存コード経路と新Plugin経路をshadow比較する（shadowは安全なread-onlyに限定）。D1bで双方の本番使用入口をProvider経由へ切り替え、**切替対象だけは旧直接経路へ黙ってfallbackしない**。旧互換入口を残す場合は期限・契約変換・実行不可条件を明記する。D1a合格はD1b完了ではない。

## 11. 追加の横断Pressure：単一仕組みを異種用途で壊してみる

| 圧力例 | 再利用するOwner | dogfoodingで確認したいもの |
| --- | --- | --- |
| Design Context QueryとAI Skill | #3262 #2605 #2509 #2516 | 全Required Closure、ページ/opaque cursor、8,192byte途中切断、非公開メタ情報、Skill/CLI/MCPでの同一契約と各Consumer受領 |
| Data Source Workbench | #3270 #2622 #1174 | SQL/NoSQL/Logのcapability差、read/DDL/Create/Alter/Dropの実Provider、実環境Grant・破壊効果と独立再読取。DBA機能を標準へ混入させない |
| HTTP Response Decision Graph | #3274 #1129 #2497 | API失敗/不確定結果/Client Reactionの有向多重Graphを標準Record/Relation/Graphで持ち、同じSourceからCLI表・Studio図・実Traceへ投影。HTTP statusだけでCommit結果を推定しない |
| テストと検証済BOM | #1180 #2514 #3267 #3223 | Scenario/Independent Expectedが同じでも、Provider×Consumer×Target exact tupleで実同居Link/Load/Invokeが必要。pairwise未試験3-wayを保証しない |
| Snapshot/Cache | #3271 #2600 #1641 | Cache hit／Build Artifact再利用／Evidence再利用／現在Grant／業務確定値の違い。Provider同版でもInput/Scope drift時には資格を使い回さない |
| Studio Pluginの失効・Source保持 | #2044 #2046 #2047 #2620 | disable/uninstall/対応不明でもRaw Sourceを保全。UIのUnknown保持・再接続・開示拒否がCLIでも一致 |
| SHO・三要素モデル・Graph Projection | #1124 #2497 #2620、SHO Composer | 三視点Data/Function/BusinessとCRUD/Cross-reference、Graph/表/Markdownを**同じMeaning Ref**から投影。SHOの文章化もモデルの第二正本にしない |
| 外部Target／AI Harness | #2390 #1121 #2509 #2831 | 同ContractのGo静的/外部Host Library等で意味とFailureを保持。HarnessのModel推論・Tool Egress・実Provider effectは別Evidence |

**共通Producerと独立Consumerを取り違えない**：CLI/Studio/Runtimeのどれも同じServiceを読めることは重要だが、3つが同じモック結果を参照していたら3つの実E2E合格ではない。Runtimeで同一Providerを共有できる要件と、Targetが異なるため契約準拠した別Providerを選ぶ要件は両立する。役割がDeveloper Tool専用なら生成業務アプリへ含めないこと自体が正しい。

## 12. D0〜D2の完成判定を小さく、しかし検証可能にする

| Gate候補 | 実験手順 | 合格／停止条件 |
| --- | --- | --- |
| **PD-0 実依存** | 現行`cmd/tei`・`cmd/tei-studio`・`internal/studiohost`・`internal/plugin`・`internal/composition`の実Go importと操作traceを記録。Go Toolchainの実入力・module/asset漏れも記録 | 依存Graphが宣言と一致している範囲とUnknownを明示。将来設計の命名だけで「移行済み」としない |
| **PD-1 Contract** | 同一`inspect` RequestについてContract/Provider exact Lock、Trait/Shape/Failure、実Provider選択の2候補と欠落を試す | 同一名前だけのProvider選択、Version/Scope不一致、undefined/Null/partialの欠落を拒否 |
| **PD-2 Two Consumers** | 実CLIの`inspect`とWails Studio Bridge経由で、同Source Revision、同Required Context、同選択Providerを呼ぶ。旧直接経路との同時比較はread-only | 診断・データ・Source/Scope・Unknownの意味が等しく、実2 call-site/実Provider呼出を観測。StudioをCLI subprocess経由の代理表示だけにしない |
| **PD-3 No Core Edit** | 互換Contractを実装する第2のInspector Providerを追加・選択してD1bを再試験。Pluginを外す対照 | CoreのKind固定列挙、CLI dispatch、Studio業務Bridgeの手書き変更0。旧経路fallback、曖昧先勝ちを拒否。ビルド定義・選択Profileの変更は可 |
| **PD-4 Physical Closure** | same toolchain/flags/seedで、未選択Providerの追加前後に`go list -deps`・実Build入力・`go tool nm`（利用可能なTarget）・Artifact byteを対照 | 条件付き実Build closureの非干渉。Manifest/Command表示や`-plugin`flagのみでは不十分 |
| **PD-5 Authority/Lifecycle** | Source Grant失効・Plugin disable・StudioのWorkspace境界・CLIの旧経路・再入／timeoutを再現 | unauthorized/unknownを隠さず、PluginからHost Authorityを昇格しない。Source保存と利用資格を別票にする |
| **PD-6 Cross Target** | #1180/#2514の同Scenario/Independent Expectedを固定し、Tooling Provider（または同Contract）と生成Go Runtimeで実消費 | semantic/input/output一致だけでなくTarget別Artifact/Load/Invoke/Current Grant/必要なら独立Effect Observation。外部Hostは別資格 |
| **PD-7 Build Dogfood** | 実Source/Provider/BuilderのBuild graphからCLIと選択済み標準Pluginを構築し、stage0→stage1→stage2を比較 | #2831のL1/L2/L3を区別。単なるGoで2回ビルドした結果をTEI self-buildと呼ばない |

**優先順位**：PD-0〜PD-3をP0、PD-4をP0並行、PD-5は同一縦切りの拒否試験、PD-6以降はP1へ。PD-0を完璧な全Repo棚卸しに膨張させない。必要なD1操作のClosureと明確に示された不足から開始し、実際の不足だけ補う。

**PD-1/2最小Fixture**：既存Reference Workspaceから`inspect`可能なSourceを一つ選び、同名異Scope Provider A/B、AのみEnabled、A Missing、Source revision drift、Required relation missing、NULL/未取得/secret、途中切断を各独立Expectedへ記録。表示Modeだけでなく実InvocationのTarget/Source/Provider/Artifactを固定。未選択Providerの追加でCLI/StudioのOfferや実Binaryに予期しない差が出る例を調査する。Fixtureの存在だけを正式Accepted Meaningとみなさない。

**合否観測値**：Coreおよび手書き分岐の変更行数、追加PluginのContract/Provider/Artifact参照欠落数、shadow差分、実Tool受領完了率、無関係Plugin Build byte混入数、誤PASS/誤失効、実Bundle size・startup p50/p95・memory・AI手順数、正確な否定試験の通過率を記録。意味Conformance、実Target/Host資格、Performanceは別Gate。性能閾値は#2600/Targetの事前Budgetへ委ねる。

## 13. Owner配分と実行記録の扱い（第二正本を作らない）

- **本計画・#1251**：CLI/Studio dogfoodingの移行順、実依存の証拠、PD-0〜PD-7の接合票。プラグイン全機能実装を#1251に集中させない。
- **#951/#1016**：Contract/Provider/Selectionの権限・曖昧性・exact Lock・Tooling/Runtimeのscope相違。新Resolver不要。
- **#1158/#1824/#1124**：型付きOperation、CLI/Studio Surface Contribution、Wails実画面。Providerに業務意味を再解釈させない。
- **#2620/#988**：Role/DSL/Expressionの意味、Trait検証、最低限のBootstrap読取。Role評価を単純なPlugin実行権限へ昇格しない。
- **#1180/#2514/#943/#1097/#3267/#2831**：Scenario/Oracle、actual Build/Evidence、BOM、全体Self-build。第二Runner/Artifact Storeを設けない。
- **#2044/#3270/#3271/#3274/#2390**：Source保持・Data/DDL・Cache・HTTP Graph・外部Hostは別々のPressure/受入Owner。各自のScopeをPlugin基盤へ要求する側であり独立第二Kernelを開発しない。

**外部比較からの採否**：VS Codeの`contributes.commands/menus`は「表示宣言と実ハンドラを分離する」を採用対象とする。Gradle TestKitは「Plugin単体テストでなく、隔離した実Consumerの実Build/実呼出を検証する」を採用対象とする。WIT/Wasm Component Modelは「Target native interface型を定義し得る」を選択肢として残すが、TEIの共通ABIや全PluginのWasm化を先に確約しない。これらは参照方式の比較でありTEIでの実測PASSではない。

参照：
- https://code.visualstudio.com/api/references/contribution-points
- https://docs.gradle.org/current/userguide/test_kit.html
- https://github.com/WebAssembly/component-model/blob/main/design/mvp/WIT.md

**実施状況**：2026-11-07の他チャット・現行main・既存Issueの静的突合と移行条件の設計追補。PD-0〜PD-7、Trait合成、actual two consumer、実Go Build/Studio/UI/Host、CLI Skill/MCP、実DB/HTTP副作用、Self-buildの新規試験は**NOT RUN**。正式Schema/ABI、数値Budget、Owner Meaning Acceptanceは**NOT ACCEPTED**。本追補は#1119のG0〜G4を置換せず、実証時の必要証拠を既存Gateへ渡す。


## 14. 2026-11-08 現物証拠から組み直すPlugin-firstとBOMの最短受入

この追補は、直近の別レーン成果を**新しい抽象基盤の提案でなく、実際のSource・選択・Build・呼出しの照合条件**へ還元する。現行mainで受理された限定Referenceと、未統合Draftの資格を区別する。#3278のAtomic Trait原則、#951のContract/Resolver、#1016のApplication Composition、#3267のVersion Matrix/BOM、#3262のDesign Context、#1180/#2514のScenario/Independent Expected、#2831のSelf-buildが引き続き各正本・Ownerである。

### 14.1 実際に確認された新しい反例：宣言、選択、能力、実行は別

- **実main #3337**（merge `4f1541658e617a376051995605cfca1963c646a3`）に読み取り専用`tools/reference-plugin-capability-coverage`が追加された。合成`reference/workspace`では2 Environmentに対して12 Capabilityを宣言、**6 CapabilityはEnvironmentの対象Bindingから未選択**と観測。ただしこれは**欠陥の件数ではない**。`database-provider`の2件は技術名(`postgres`)と能力ID(`database.postgres`)を同一視できない算定上の限界がある。さらに、Plugin自体は選択されてもその中の`editor-intelligence` CapabilityのBindingがない例、同じ`migration.atlas` IDでも`deployment-executor`だけ選ばれ`migration-provider`は選ばれない例がある。**Plugin selection ≠ Capability selection**を具体的に実証した範囲に留める。actual Go link/Runtime invoke/第三者BOM適合は確認していない。
- **実main #3336**（merge `04f75301f408eb98b95ca90a103d6d8ad21ffdba`）に`tools/reference-go-build-stage-driver`が追加された。Buildを別プロセスとして実駆動し、**invoker実行ファイルのdigest／invoked実行ファイルのdigest・argv・PID・exit／子の申告artifact digestとディスクで読み直したdigest**を別々に観測できた。しかしStage1生成物によるStage2の真の再帰的セルフビルド(L3)は未実施であり、`l3Claimed=false`。BOMの`Built`を「Builderが呼ばれた」「Reportにdigestがある」だけで保証しないための実証部品とする。
- **実CLIドッグフーディング #3149**（横断実測はclosed）から、#3345（`view`でWorld/Source Indexの版照合値が欠落・異なる範囲のfingerprint）、#3346（`docs/plan`内MarkdownはmanagedでもLoaderで解釈されない）、#3347（応答JSON shapeの違い）が個別課題として残った。**読み取れるデータ、管理されるファイル、意味を解釈したSource、AI Toolが完全受領した出力を同一視しない**。#3183/3182等の旧観測がclosedであっても、全CLIが同じLock形式をそのまま再入力できる、すべての`read-only`宣言は無副作用、といった上位保証への自動昇格はしない。
- **#3262/#3273系のT0.5/T1**：限定的なContext不足の検出と固定Scenario実行の進展はあるが、Required Rule/RelationをSource/Loader/Meaningから閉じてAIが未知の独立Negativeを発見し、同じ実SourceでMutantを殺した**T1-B**はまだ別資格。#3324はclosed/unmergedのDraftであり、単に最新Issue一覧へ現れたことをmain実装に数えない。
- **BOM [Draft #3297](https://github.com/hat47x/tei-platform/pull/3297)**：RQ-035はData Source Workbench、BOMは**RQ-036**へ分離。純Go ReferenceはFull/Pairwise/提供済exact Lock相対の計画と同一Attempt入力の整合を検査する。直近の**PR head `a6cddd6778613fa9d6b2ecedfcf0aaade5e51cbc`でGitHubのHealth/依存脆弱性Workflow双方がSUCCESS**。ただし同HEADとmainの比較は`diverged`、作業時点でmain側が**74 commit ahead**、Draft側が18 commit ahead。後続のmain更新を取り込まずに**現行mainの受入済実装**とは言わない。実Provider/Runtime/Host/正式BOM発行はNOT RUN。

### 14.2 利用と資格の五段階を一つの「Plugin使用済み」に潰さない

| 観測対象 | 必要な独立証拠（既存Reader/Owner） | 検証できないこと |
| --- | --- | --- |
| **Declared** | Source上の`provides`、Contract/trait identity・version、Source occurrence | Pluginが選択・実装された |
| **Selected** | Environment/Function/Targetに相対的なPlugin Resolution/Lock。Kind+Capability IDが一致したBinding（#3337）。不一致・未Binding・未知を別分類 | Providerの具体Memberが選択された、実Load済み |
| **Realized/Built** | 既存#943/#1097/#2831の実Build Closure、実Link/import/init、invoker/invoked、toolchainとartifact byte照合。Go static と外部Hostは別Target | 実際にそのCapabilityが呼ばれた |
| **Invoked/Observed** | 実Consumerによる型付き`Service`のProvider identity・operation・attempt・target・入力/出力・失敗・読取範囲。単なる関数結果の値一致より**どのProviderが呼ばれたか**を優先 | Contract意味適合、テストの独立性 |
| **Conformant/Currently usable** | #1180/#2514の独立Expected/負例/Scope付きCoverage＋#3267 exact BOM、現在Host Grant/Disclosure/Owner Meaning・Effectはさらに別照合 | 別Version、別Target、別Tenant、将来時点までの無条件保証 |

この表の行はProduct新Core状態列挙でなく、同じ既存Subjectの異なるread-only投影である。公開面の設計案では`Open/unpromised`を基本、明示`Stable`を区別するが、**同Bundleにある未公開Internal/Secretを発見可能として外部に渡さない**。Open依存でのAdapter修復責任は下流が持ち、発行者によるStable保証や実現在Grantと混同しない。

### 14.3 最小縦断：Tooling Consumerの実選択を先に証明する

試験対象はまず**read-only `inspect`相当の一つの実Service/Contract**とする。新しいPlugin Kernel/Resolver/Script Runtimeを作らず、標準Provider Aと代替Provider B、CLI Consumer、次にStudio Consumerを同一契約へ接続する。既存CLIを使えることと、既存CLI自身がPluginとして実装済みであることは別である。

| Case | 改変する条件 | 独立Expectedと実観測 | Owner |
| --- | --- | --- | --- |
| **PDC-01** | 同じContractにProvider A/Bを宣言、Aだけ選択 | Bの存在を未選択＝欠陥としない。実CLI invocationはA。Source/Capability/Target/Lock identityを追跡 | #951/#1251/#3279 |
| **PDC-02** | Plugin Bを選ぶがCapability BindingはAのまま／未指定 | Plugin選択成功をCapability使用済みとしない。必要なBindingが未決ならunknown/失敗。Provider Aが呼ばれてもB合格としない | #3337/#951/#1016 |
| **PDC-03** | BへBindingしても実配線はAを呼ぶ（A/Bは同じ値を返す） | Provider identityと実Consumer invocation traceで**必ずFAIL**。Result一致やexit0だけで合格にしない | #1158/#1824/#2514 |
| **PDC-04** | 同Bundleへ未使用Plugin Cを同梱 | Go import/init/actual BuildにCが影響しなければ既存選択は不変。実混入するなら非干渉PASSを拒否 | #1251/#943/#1097 |
| **PDC-05** | invokerとinvokedの実行物を差替え、Child reportだけ同じDigestとする | 実Binary・Child argv/PID/exit・Disk Artifact digestの不一致を診断。L3への自動昇格なし | #2831/#3336 |
| **PDC-06** | Plugin/Ruleの必須Sourceを`docs/plan`に置くだけ／CLI出力を途中切断 | managed/parsed/loaded/returned/end-markerを分け、Required Closure不明ならBOM qualification不許可 | #3262/#3346/#2509 |
| **PDC-07** | A1/A2×Consumer C1/C2の4 exact tupleを実行 | #3297のMatrixは**計画のみ**。各tupleに同じTarget/Artifact/Independent Expected/Attemptと実際に呼び出したProviderを束縛。実呼出が欠ければnot-run | #3267/#1180/#2514 |
| **PDC-08** | A/B個別PASS・同一Attemptに見えるが別Source/Artifact/Expected/Scope | 旧資格の寄せ集めでQualified Unionを発行しない。失効Scopeだけ再検証 | #3267/#2600 |
| **PDC-09** | 本体ではCLI read-only正常だがStudioから異Provider経由／Local-only権限喪失 | Serviceの意味とProvider使用を独立観測。UI表示と実Runtime資格を別判定 | #1824/#1124/#1174 |
| **PDC-10** | Manual Required Trait未実装、代替Open Providerが存在する | #3278のAtomic Requirementが未充足なら勝手に代替実装へfallbackしない。Publisher Stableを強制しない | #3278/#3279 |

負例は既存#1180 Scenario/Independent Expectedに委譲し、ここへ第二Scenario/Oracle/Runnerを作らない。D0/PDC-01〜03（Go CLI実選択→実呼出）を最小P0、PDC-04〜08をParallel/次段P0、PDC-09〜10をStudio/異Targetで拡張。各Caseの対象Source/Revision/Tool/Process/Artifact/Conformance Evidenceは別欄のまま、同一のRequest／Run／Versionに束縛する。

### 14.4 開発の順序と受理可能な限界

1. **read-only P-1**：#3337 Referenceの`Declared/Selected/Unselected`を、#3262のRequired Closure（#3345のfingerprint範囲、#3346のMarkdown Loader非対応を含む）へ投影する。Kind/IDだけで部分的に把握できるものをcompleteとせず、`database-provider`など未照合はunknownとして分ける。
2. **P0-A**：`inspect`に相当する実ServiceをProvider A/Bで選択・実呼出しし、PDC-01〜03を同じBinary/独立Expectedで実測。**最短の合格は一つの実Consumer/実Providerの正しい束縛**であって、Studio全面移植ではない。
3. **P0-B**：#3336 Stage Driver等が持つ実invoker/invoked・Disk artifact照合を、選択したProviderと同じ実Build/Targetの証拠へ接続する。ただし本当のSelf-build L3は別#2831の資格。
4. **P0-C**：#3297のReferenceを最新mainへ適合させ、実Resolver→実Scenario Runner→実Artifact→限定BOM rowへのjoinを検証。Matrix計画、Result相互一致、実Provider Conformance、BOM発行を**4段階の別結果**にする。CIが過去のHEADでPASSでも新HEADの意味検証を省略しない。
5. **P1**：Studio/CLIから同じServiceへ呼出して意味・actual provider identityの一致を検証。Markdown Authoring/三視点/HTTP Graph/Cache Provider/SHOは既存#2621/#1124/#3274/#3271等へ渡し、P0のHard Gateへ積み増さない。

**成功判定の母数**も分ける。能力選択率、要求/Scenario coverage、コードC0/C1、Version pairwise、実Provider Invocation、HTTP Guard/Reaction、独立Expected/EffectのCoverageは**異なる分母**であり、最も高い値や一つの総合`verified`で代理しない。AI生成のテスト候補はOwnerが採用するまで提案に留め、同じLLMが作った期待値の自己採点を独立検証へ数えない。

**今後の停止条件**：未知のRequired Source、Loader未読込、別World fingerprintのCLI出力混在、Provider identityの実証欠落、旧Artifact/別TargetへのEvidence流用、同名Capabilityの異Kind無断統合、Tool結果の部分配送、Cache hit＝Current Qualification、HTTP success＝Business complete、現在Host Grant不明。これらを`empty/no-plugin/pass`へ読み替えない。

**位置づけ**：#3337/#3336の限定Referenceがmainに存在すること、および#3297の指定HEADでCI PASSしたことは**確認済み**。PDC-01〜10の新規実Consumer E2E、#3297のmain再基線化、正式Plugin API/ABI、Studio実呼出、Trust/Owner受理、本番BOM/Effectは**NOT RUN / NOT ACCEPTED**。この節は単一の新Gateを増設せず、#1119 G0〜G4および既存Ownerへ証拠を返すための計画である。
