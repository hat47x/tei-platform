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

## 14. 横断収束：実行可能Offer・表示逆写像・受信側SinkをD1の契約監査へ接続（2026-11-08案）

**位置づけ**：#1251のD1/PD-1〜5に渡す最小の検証差分。#951/#1016のContract/Provider/Resolution、#3278のAtomic Trait、#1158のOperation/Surface、#1180/#2514のScenario/Independent Expected、#3267のBOM、#2516のDisclosure、#2497のProjection/逆写像が各正本。新しいCore enum、第二Resolver、権限・Evidence・Scenario SSOT、共通TEI Hostを作らない。以下のOperation Offerは**説明上の集合名**であり、正式Schema/ABIの採用ではない。

### 14.1 「広告できる」と「実行できる」の間に残る空白

一つの操作の受入で、次の独立観測を揃える。異なる層を一つの `available=true` に潰さない。

| 層 | 既存Ownerと確認対象 | 不成立時にしてはいけない推論 |
| --- | --- | --- |
| 意味・型 | #2619/#2620/#3278：Role、Trait義務、型、NULL、Effect、Failure | 名前/構文が合うだけで意味が満たされたとしない |
| 広告・入力 | #1155/#1158/#3242：CLI/MCP/Studioに投影する引数許容値、既定値、条件付きEffect | Help/Discoveryに載っただけで全入力Variantが実呼出し可としない |
| 到達可能性 | #1251/#951/#1016、Hookの既存Validation：必要なPattern/Handler/ProviderとScopeが実接続される | HookやSurface Contributionの宣言だけで実行済み・実装済みとしない |
| 実構成 | #943/#1089/#1251：選択Provider、実Build input、linked Artifact、Host入口で観測した実Provider | 選択B・実呼出A、同じResult bytesを合格にしない |
| 現在の効果 | #1174/#2516：Operation Variant、実Write/Receipt、対象・Grant、Unknown | `read-only`名義のNamed File Outputや実DB unknownを無作用扱いしない |
| 受信・公開 | #2509/#2515/#2516/#3262：CLI/stdout、MCP Tool、AI Harness Session/Log、Studio/SHOへの配送と保存/外部送出の観測 | ProviderがReadであるだけで結果の非永続・非外部送出・完全受領を保証しない |

**既存実例への接続**：
- Draft [#3306](https://github.com/hat47x/tei-platform/pull/3306) はHook `uses` がApplicationの `command` Patternへ到達しない場合を検証時に拒否する実装候補。PluginのOfferでも同種の「宣言はあるが実Handlerが無い」をD1負例にする。HookのCommand経路をそのままPlugin専用Core固定仕様へ転用しない。
- [#3307](https://github.com/hat47x/tei-platform/pull/3307) は既存CLI Discoveryの引数許容値・既定値と実受理を一致させる限定的変更。これを、全Input/Operation Variant/Provider適合が既に自動導出された証拠とはしない。
- 実CLI [#3182](https://github.com/hat47x/tei-platform/issues/3182) とDraft [#3302](https://github.com/hat47x/tei-platform/pull/3302)：`view`の`-execute` VariantはNamed Local Fileへ書ける。実出力と契約上のEffect広告を別々に検証し、実DB Commitへは昇格しない。
- [#3309](https://github.com/hat47x/tei-platform/pull/3309) が報告するDSH headlessの**合成canary限定**観測は、read-onlyで取得したTool結果が永続Session Logにも現れ得ることを示す。他のSink、TEI製品、第三者環境へ無条件に一般化しない。受信側保存先の実測は別Ownerで継続。

### 14.2 表現型Pluginと標準データ構造の境界

**データのIdentity/Relation/Topologyは、Canvas・Table・Graph・Markdown・SHOの表示上のCardやNodeとは別**。SUI/KJ/マインドマップで同じItemが複数Canvasへ現れても、Itemの同一性・Source/Accepted Meaning・依存を重複して複製しない。未分類のRaw ItemはPlugin未導入・無効化後も保持し、未解釈を「存在しない」に縮退させない（#2044/#2497）。表示Pluginは必要なShape/Axis/Unit/Source/Multiplicity/Orderを契約として要求し、非単射・損失あるProjectionは `renderable` と `editable/writeback` を区別する。逆写像の対象Identity/Revision/権限が一意に定まらない限り自動Writeを拒否する。新しい万能ItemKindやCanvas専用Core型を要求しない。

Data Workbench #3270、REST Decision Graph #3274、三要素モデルとCRUD/SHO、SUI Canvasを**異種Consumer**として使う。どれも同じContract/Providerの意味とEvidenceを参照できるが、画面で操作できることから元データへのWrite資格を推論しない。Source保存は開示や業務意味受理とは別Claim。

### 14.3 最小追加負例（既存#1180/#2514のIndependent Expectedへ委譲）

| ID | 反例 | Expected / 既存受入Owner |
| --- | --- | --- |
| PO-01 | Plugin Surface/HookがOperationを宣言、実Handler/Pattern到達経路なし | `declared`のみ、`executable`不成立と診断。#1251/#1158／Hook類似#3306 |
| PO-02 | CLI広告の許容値/既定値/条件付きEffectと実Provider受理が違う | Offerの対象Variantを拒否・診断。#1155/#1158/#3182 |
| PO-03 | Bを選択したがCLIまたはStudioがAへ実呼出、返却bytes一致 | Host由来の実Entry不一致でFAIL。#1251 D1-W01〜03 |
| PO-04 | Read-only SourceをToolで取得し、受信Harnessが結果を永続Sessionへ記録 | Provider EffectのReadと受信SinkのWriteを別観測。Grant/開示とSinkの実測範囲外はunknown。#2515/#2516 |
| PO-05 | 同一Itemを二Canvasへ表示し片方で非可逆編集を試す | 同一Identity・重複計数なし。逆写像/Revision未確定ならWrite禁止。#2497/#1124 |
| PO-06 | Renderer Plugin無効化、未知Raw Itemが存在 | Raw Source/Refは保持、対応するView/Write/Meaningだけunknown/unsupported。#2044 |
| PO-07 | NULL・単位・時点が不適合なProviderを型名だけで選択 | Structural/semantic Conformanceを別検証し不適合を拒否。#951/#3278 |
| PO-08 | 4バージョンtuple中Synthetic Witnessが1件だけ実Call未観測 | 実組合せQualifiedへ加えずmatrix-incomplete。#3267 |
| PO-09 | 同じLockでも受信先のDisclosure/GrantまたはSource Revisionが変化 | 過去のResult/Cache/BOM Evidenceを現在の開示・実行資格へ流用しない。#3262/#3271/#2516 |

### 14.4 最短の実装順と停止線

1. **D1を肥大化させない**：既存`inspect`一操作を、A/B Provider・CLI実Call-site・Studio実Call-site・Host観測で実証。PO-01/02/03/07を優先し、同一結果bytesの誤配線を検出する。
2. **独立した横断pressure**：D1で得たTyped Offerのread/unknown/closureを一つのData Workbench Readerへ接続する。別レーンでPO-04の受信Sink実験を#2515へ、PO-05/06の非可逆表示・Raw保持を#2497へ返す。これらをD1完了のHard Dependencyにしない。
3. **資格の引渡し**：実選択・実Build/Invoke・独立Expectedだけを#3267に渡す。合成witnessや独立Tool PASSのUnionからVerified BOMを作らない。外部Host、実DB Effect、SHO/SUI双方向編集、G4 Self-buildはそれぞれ実Target証拠で後続評価する。
4. **費用対効果**：新しいContract/Providerを一つ追加するためのCore差分、Consumer別特別処理、Open APIへのAdapter変更量、D1誤PASS/誤失効、Sinkの未観測割合、Build closure混入、入力/出力配送の完全性を記録する。第二正本や新Core enumが必要になれば抽象境界を見直す。

**状態**：横断議論とGitHub上の既存Issue/PRからの**設計・受入案**。PO-01〜09をTEI製品経路で実行したわけではない。Plugin実2 Consumer、実BOM資格、SUI/SHO Writeback、実MCP Disclosure/Sink、正式Owner Meaning Acceptanceは未受理・未実証。旧/進行中PRの局所PASSをこの統合全体のPASSへ拡張しない。

## 15. Function配置・Topology Binding・変更影響をPlugin-first適合に接続（2026-11-08横断案）

**今回の差分**：§14は「Plugin Operationを宣言・表示・実呼出できるか」を扱った。本節はその次の独立条件、すなわち**同じ能力をどのFunction Data Contextの、どの粒度・関係・表示インスタンスへ適用できるか**を限定する。他チャットの標準データ構造／標準表現型分離（#2495/#2497/#2507）、Functionに置く表現インスタンスとData Context（#2506/#1127）、ViewChange→Intent（#2500）、依存範囲の再資格化（#2582/#2600）、Plugin-first（#1251/#951/#1016）を接続する。正式新Schema/ABI/Core列挙を本節で採用しない。

### 15.1 機能の提供とFunction内の適用・配置は異なる

- **Representation Definition / Pluginの能力**：Table/Tree/Graph/Timeline/Audio/Video等の表現型、必要Shape・Axis・Unit・Cardinality・編集Intent・Target制約を宣言する。Pluginがinstall/discoverされたことは実装選択・Function配置・現在Grantを意味しない。
- **Data Context Definition/Instance**：Functionが参照するSemantic Subject/Relation path、Entity/Edge/Path occurrence/Aggregate等の粒度、取得Sliceの完全性/partial/unknown、Source/Query/Rule Revision・selection epochを持つ（#2506）。複数Contextを一つのJOIN出力へ無断同一化しない。
- **Representation Instance in Function**：表現型の実利用は必ずFunction内のインスタンスとして配置し、どのContextとRelation/Topologyに接続するかを明示する。表現型Pluginの登録時に全Functionへ自動配置しない。複数Canvasが一つのItemを別Occurrenceとして表示してもData Identityを複製しない。
- **Structure/Topology Binding**：Data側の同一性・向き・多重辺・順序・時間軸/座標系・単位・NULL・部分集合を、表現型側でどの範囲まで保持/変換/喪失するかを宣言する。形状適合と用途適切性（例えば集計値を個別Entityとして編集できるか）は独立に診断。自動推測で元の関係を増減しない。
- **Operation Intent／Writeback**：表示で可能なSelect/Drag/Editは、元のEntity/Edge/Path occurrenceとRevisionへ一意に戻せるときだけ型付きIntent候補を作る。Provider適合・Host現在Grant・実Commit/独立Readはその後の別資格（#2500/#1174）。非単射な表・Graph・SHO文章の逆書戻しは原則拒否し、Candidate/Reviewへ戻す。
- **Openな拡張との両立**：未知の表現Pluginや独自Traitでも、契約を満たす範囲を表示・適用候補にできる。ただしOpen/Stable/Internalの公開保証、利用可否、Owner Meaning Acceptance、現在実行権限は別。Trait Fulfillmentの正本は#3278、Consumer実証は#3279。

**解釈上の重要点**：`Plugin selected`、`Function-bound`、`renderable`、`editable`、`operation-offered`、`executable-now`、`effect-observed`は、同じBooleanで管理しない。必要な既存契約の結果をread-onlyに合成した用途相対の判断とし、第二Availability Registryを作らない。

### 15.2 変更時には「何を再Buildするか」と「どの表示・操作Claimを再確認するか」を別々に判断

| 変更 | 表示／Functionへの影響 | Build/BOM／実行資格への影響 |
| --- | --- | --- |
| 未選択の表現Plugin/Provider追加 | Catalog候補のみ変化。Functionの既存Bindingは原則維持。ただしDefault選択へ作用する場合はSelection再確定 | 実Build input不変ならArtifact維持候補。実import/init/Embed混入時は再評価（#1251/#2600） |
| 選択Pluginの実装改版、Contract意味は同じ | 対象Representation/AdapterのConformanceを再確認。受理済みData Meaningを自動変更しない | 実Provider/Artifact/Targetのexact資格を再確認。旧BOMのVersion文字列一致だけではPASSにしない |
| Function内の表示位置・スタイルだけ変更 | 表示Artifact/Occurrenceは変更。Entity/Relationと独立の意味・データClaimは維持候補 | Buildへ実消費された範囲だけ変更。UIだけの編集をRuntime/DB全体失効にしない |
| DataのRelation/多重度/軸・単位/粒度の変更 | 該当Binding、集計、操作Intentの意味・逆写像を再確認。別Contextの無影響は依存根拠がある場合のみ維持 | Data/Query/Provider/Generatorの実消費閉包を照合し局所再Build |
| D2部分取得・親Selection epoch更新・遅延旧応答 | D2依存Claimはpartial/stale。独立D1 Claimまで無根拠に失効させず、旧結果を新Contextへ混ぜない | Plugin実装が同じならBinaryを再Buildする理由とは限らない。Current disclosure/Action資格は再評価 |
| Provider/Source A→B→A、旧Lock・Grant再登場 | 見かけの表示一致だけで旧現在資格は復活させない。必要時に再取得・再解決 | #2600/#3267の旧Evidence再利用条件・Artifact来歴とCurrent Grantを別判定 |
| 表現Plugin無効化・不明Source | 未解釈Raw Item/Sourceを保持、対応Viewはunsupported/unknown。存在しない・削除済みへ変換しない | 選択済みProvider欠落時は黙ってDefault/旧直呼出へfallbackしない |

**増分更新は追加だけではない**：#2582のLEFT JOINの未一致行、NOT EXISTS、Top-N、集計のように、入力追加で既存表示が消える場合は`retract/replace`の必要性を明示する。Producerが部分取得しか保証できないとき、0件と「存在しない」の同一化を拒否。Cache hitやBOM適合履歴でCurrent Result/Disclosureを保証しない。

### 15.3 既存Scenario/Independent Expectedへ渡す追加圧力（PFB-01〜11）

| ID | 自作最小ケース | 独立Expected／担当 |
| --- | --- | --- |
| PFB-01 | Table Plugin install済み、FunctionにRepresentation Instance未配置 | CatalogでdiscoverableでもFunction-bound/renderedとしない。#2507/#1251 |
| PFB-02 | 同一ItemをCanvas A/Bに二Occurrenceとして表示 | Item Identityは1件、View Occurrenceは2件。片側の位置変更でDataを複製/意味変更しない。#2497/#1124 |
| PFB-03 | 同labelのNode 2件＋同source/targetの異なるEdge 2件をGraphへ投影 | Node/Edge ID・方向・多重性を保持。表示label一致を同一Entity/Edgeへ畳み込まない。#2495/#2497 |
| PFB-04 | 子D1=2件、D2=3件、JOIN投影6行とD1集計1行 | 6表示行≠子Entity 6件。集計行から個別書戻し不可。#2506/#2507/#2500 |
| PFB-05 | D1全件、D2先頭ページのみ。D1だけの合計とD1×D2のJOIN集計 | D1依存の限定Claimは維持候補。D2依存の全件Claimはpartial/unknown。#2582/#2586 |
| PFB-06 | Window1で親A→B、AのProvider応答が遅延到着。Window2はAを表示中 | W1のB Contextへ旧A結果を混ぜず、W2のAまで誤失効しない。#2506/#2582 |
| PFB-07 | Tree/Graphで多親のEdgeをdrag/edit、表示上はEntityカードを移動 | Edge/Path occurrenceを型付き対象にし、Entity deleteや全親削除へ変換しない。対象曖昧ならIntent拒否。#2500/#2497 |
| PFB-08 | RendererのAxis/Unit/時点/座標系不適合、形状だけは成立 | Render候補と適切性/変換資格を分離し、無音の単位変換・意味受理を禁止。#2495/#2507 |
| PFB-09 | 未選択Plugin追加／Default変更／実Buildへの`init`混入の3例 | Catalog変更・Selection変化・実Build混入を判別し、不要失効と誤Reuseの双方を拒否。#1016/#1251/#2600 |
| PFB-10 | 同Resultを返すProvider A/B、Bを選択・Aを実Invoke | 値一致でも実Callの不一致をFAIL。Function Bindingが正しくても実Implementation資格は別。#1251/#951 |
| PFB-11 | Plugin取り外し・Raw Itemのみ残存、CLI/Studio/SHOは異なるRenderer | Source/Identity保持と表現不能を区別。開示不能の件数/Pathを無断表示しない。#2044/#2516 |

### 15.4 段階・Owner・非目標

- **P0／D1既存優先**：#1251のInspector一操作、A/B Provider、CLI/Studio実call-site、Hostでの実Entry観測と独立Expectedを先に通す。PFB-10はD1-Wと同じScenarioを再利用し、別Runnerを作らない。
- **P0並行／Headless Binding**：#2506/#2507/#2497/#2500の既存FixtureからPFB-01〜05/07/08を対象に、一つのFunction、二Context、Table+Graphの二表現インスタンスを**read-only**でinspectする。合成入力だけならReferenceとして明記し、Product Parser/Studioでの実利用と区別する。
- **P1／変更影響**：PFB-06/09と#2582/#2600の依存Closureを結合し、Build Artifactの失効とFunction表示/Action Readinessの失効を別票で実測。未知依存はunknown。BOMのexact tuple資格は#3267が所有する。
- **P1以降／Writeback/Target**：#2500/#1174で非単射View→Intent候補→Host実Grant→実Effect→独立Readを順に検証。Java/.NET/Go、SUI/SHO/Studioは必要な実Consumerだけ選択し、新TEI Host前提を設けない。
- **測定**：Core変更数、Plugin独自Adapter数、誤Entity/Edge対象数、NULL/partial過大Claim、無関係Context/Artifactの誤失効、依存変更見落とし、CLI/Studioの意味差、独立Expectedの負例検出率、構成手順・生成AI指示数。新特別機構のコストが優位なら本抽象案を縮小/撤回する。

**責務**：#2495構造分類、#2506 Data Context、#2507 Function Pattern適合、#2497投影、#2500逆写像、#2582反応/局所失効、#951/#1016 Resolver、#1251 CLI/Studio Plugin移行、#3278/#3279 Trait、#2600 Build/Cache、#3267 BOM、#1180/#2514試験、#2516開示。既存正本へ参照・引継ぎし、新Function Binding Store、第二Resolver/Renderer Engine/Cache/Runnerや新Gateは増やさない。

**状態**：設計・負例・受入順の追補。新Function Bindingの正式Schema/ABI、PFBの実Go/Studio/GUI/Host/DB、製品Graph/Canvasの二Consumer、BOM資格、Ownerの意味受理は**NOT RUN/NOT ACCEPTED**。過去の合成Reference PASSをこの統合PASSとしない。

### 15.5 三つのTopologyと媒体軸をPlugin Consumerが再解釈しない（PR #3192/#3222からの限定受入）

二つの表現型を単純に並べるのでなく、**(a) Data topology**（同一性、Relation/Edge、集合/順序/量/時間・空間の意味）、**(b) Representation topology**（Panel/Tab/Tree/Graph/Chart/Media trackのSlot/軸/Occurrence）、**(c) Function-owned Binding topology**（どのContext/Relation occurrenceをどの表現Slot・軸へ接続し、何を変換・喪失するか）を分ける。Functionに属するのは表現型の**InstanceとBinding**であって、標準表現型のDefinitionそのものではない。各層は標準Record/Relation/Graphでread-onlyに説明可能な候補とし、独自Canvas Store/Binding Resolverを新設しない。

**軸は名前だけで同一にしない**。元DataのDomain/Event time、SourceのObservation/Acquisition time、MediaのPlayback time、文書のStructural position（文字/段落/ページ）を区別する。文章→TTS朗読、数値→Sonification、ページ→動画のように新しい時間軸を生成する変換は、原データの時間軸を「保持」したと宣言できない。Source↔Playback/Span対応の変換版、部分対応、丸め・量子化、Source Revisionを根拠として記録し、同期Selectionを元Sourceへの編集や許可に自動昇格させない。Decimal sliderの有限stepでは高精度値を無損失更新できるとしない。Tree上で多親DAGの第二親を不可視にした場合も元Relationを消さない。

**差分のOwner**：標準構造・軸/媒体の定義と既存RP-01〜08は [Draft PR #3192](https://github.com/hat47x/tei-platform/pull/3192)、複数View間のSelection/State/更新境界と既存MV-01〜08は [Draft PR #3222](https://github.com/hat47x/tei-platform/pull/3222)、Projection損失は#2497、Function Bindingは#2506/#2507、逆写像は#2500、Target Rendererは#997、Source書戻しは#1124を参照する。本Plugin-first計画は**そのRequirementがProvider選択・実Load/Invoke・Display/Write Offerへ正しく伝播するか**だけを検証し、RP/MV試験を重複定義しない。

**Plugin側の追加拒否条件（PFB-12〜14）**：
- **PFB-12**：再生時刻が一致しても原Event time・文書Source spanが異なる二Representation。選択同期は許容し得るがSource更新対象の一意性/原時刻一致を推定しない（#3192/#3222）。
- **PFB-13**：Decimal値を低分解能Sliderへ投影。読み取り/近似表示は可能でも同じ正確値へWritebackできる保証がなければ編集不可（#3192/#2497）。
- **PFB-14**：Function A/Bで同じ表現型Definitionを使うがContext/Disclosure/Selection epochが異なる。共有Plugin/Renderer実装は可、Functionの操作対象・現在Grant・保存/受信Sinkを共用しない（#2506/#2516）。

**受入停止線**：#3192/#3222は現時点ではDraftであり、実Media/Renderer/複数Consumerの合格を示さない。Plugin-first D1 Inspectorに媒体/Graph GUIの完成を要求せず、PFB-12〜14は後続Consumer pressureに留める。既存Test/Independent Expected #1180/#2514へ接続し、第二テストRunnerを置かない。
## Desktop固有のD1受入：共通Inspectorを実際に呼んだProviderへ束縛する（2026-11-07）
<!-- tei-desktop-inspect-d1-actual-consumer-20261107 -->

**統合位置と既存証拠**：§10〜13のD1a shadow／D1b切替・PD-0〜7を受け、[#1824](https://github.com/hat47x/tei-platform/issues/1824)の**既定`/workbench/` Consumer**だけの実装・試験停止線を定める。旧[Draft #3280](https://github.com/hat47x/tei-platform/pull/3280)を最新mainへそのまま適用せず、重複するContract/Plugin/Trait一般論を削除した置換差分。[Draft #3292](https://github.com/hat47x/tei-platform/pull/3292)はMaster Plan Gの縦断優先順であり本節の実Consumer受入正本ではない。**本節の追加自体は実コード・実UI・実ProviderのPASSではない**。

### D1が今不足している実Callと、先に使える既存実装

- 現main `internal/operations/inspect.go::InspectSubject` は、対象Subjectのread-only Canonical投影・World/Source Index Fingerprintを返す。ただしSourceコメントのとおりFingerprintは**記述的な同一性情報であってCAS用Revision TokenでもMeaning Acceptanceでもない**。用途別Required Domain Closure、実Parser消費、Hostの現在Disclosure、Provider Scopeの完全性を自動保証しない。
- 現CLI `cmd/tei/main.go` は `inspectWorkspaceJSON` へ静的dispatch、Studio `cmd/tei-studio/main.go::StudioBridge.InspectSubject` は `studiohost.ReadOnly.InspectSubject` へ委譲。Studio側は `operations.InspectSubject` へ接続済み。**低位の同じInspector関数の再利用は存在するが、#951/#1016でexact選択したPlugin Providerの二Consumer実Invokeの証拠ではない**。大規模移管を先行せず既存Read Serviceを同じContract Adapterで再利用する。
- `reference/tei-studio-workbench/main.ts` はCatalog SubjectとInspect、read-only `validate/refresh/inspect`に限定。`presentation.ts::relatedGraph` は単親Catalogの最大8 Entry表示で、多重Relationや実Function Result Sliceではない。Document Raw Editor、Function Data Context、Data Source実Row／DDL、HTTP Decision Graphの実Graph表示はD1のInspector成功をもって資格を付与しない。
- [merged #3285](https://github.com/hat47x/tei-platform/pull/3285)の原子的Trait・契約充足をD1でも判定し、宣言だけのRead Traitや`read-only`広告を実挙動と混同しない。[#3182](https://github.com/hat47x/tei-platform/issues/3182)ではCLIの広告と記録系副作用の不一致が既知。D1の`inspect`は外部Writeなしに限定し、Tooling側の記録やReadの外部送出は別のEffect/Disclosureとして観測する。

### 一改訂・一対象で固定する証拠と受入

| ケース（EDP/PDに従属、独立Scenario正本は増やさない） | 一因子差と期待 | 状態 |
| --- | --- | --- |
| **DC-01 exact selected→actual** | Provider A/Bが同Contractを公開、LockがBを選択しAを実呼出する故障を必ずFAIL。Caller/Host・Contract/Scope/Target・選択Pin・**実Invoker ID/Version/Artifact**・Result/Evidenceを同一Attemptで照合。単にCLI/StudioのJSONが一致しても合格にしない | 新D1検証NOT RUN |
| **DC-02 CLI↔Studio二Consumer** | 同一Workspace実Manifest bytes/Source Root・Subject・公開範囲の`InspectSubject`をCLI JSONと実`/workbench/`で観測。Source/Revision/Failure/Null/Unknownは対応し、UI列省略は`complete`を増やさない。UI後着/別Workspace/同Subject異epochは既存`createResponseScope`等で拒否 | 新D1検証NOT RUN |
| **DC-03 不在・曖昧・Trait欠落** | Plugin未選択、同優先順位A/B二Provider、Read Trait必須未実装、same ID別digest/Scopeで暗黙first-winner・旧手書きCLI/Wails Bridgeへの無断fallbackなし。適格Default Implementationのみ採用し、助言Traitを実行能力にしない | 新D1検証NOT RUN |
| **DC-04 Source/Tool閉包** | World/Source FP一致でもRequired Family欠損、Source Root片側、`*.tei.md` Parser未接続、8192-byte途中切断、JSON重複member、end marker欠落では`complete`/AIテスト生成資格にしない | [merged #3284](https://github.com/hat47x/tei-platform/pull/3284)/[#3296](https://github.com/hat47x/tei-platform/pull/3296)は別の限定Gate/CI証拠。**CLI/Studio実Provider D1のPASSではない** |
| **DC-05 異質データ・Graph** | [merged #3287](https://github.com/hat47x/tei-platform/pull/3287)の合成Postgres 3行・JSONL 5件はFixture契約。実Row/Scope/History/DDL Applyは#2586/#3270の別試験。HTTP多重辺/自己ループは現Catalog Graphへ押し込まずTable/Source参照＋unsupportedを返す | Fixture限定試験とProduct Readを分離 |
| **DC-06 実Build/TargetとCurrent Authority** | 未選択Pluginのimport/`init`/assets混入、選択版と実Linked版不一致、対象/Principal/Grant更新、receipt-old/unknown writeを別Claimとする。UIで表示できたProviderやBOM pairwise/独立Reader成功から実Target資格を推測しない | 実Build/Windows/Effect NOT RUN |

### Desktop D1のUI後着応答検収：Workspace同名ABAを独立に固定
<!-- tei-desktop-inspect-aba-regression-20261107 -->
- [Draft #3327](https://github.com/hat47x/tei-platform/pull/3327)は既存`reference/tei-studio-workbench/responses.workspace-switch.test.mjs`の`createResponseScope`に**A1→B→A2（同じWorkspace名・同じSubject ID）**を追加。旧A1の成功・失敗がA2のInspector表示／エラーを上書きしない2対照。mainと等価な`responses.ts`と追加ロジックのローカルNode 22で**2/2 PASS**、PRはDraftで新exact-head CI／Node24`npm test`／実WebViewは未確認。
- **この局所合格の用途**：DC-02の「UI response-scope epochをvalue ABAに依存させない」を限定的に補強する。ただし既存Host Bridge／CLIとStudioの選択Plugin Providerはまだ接続されておらず、**二Consumerの実Provider同一Invoke／Read cohort／Function Context・Grant／実EffectはPASSしていない**。`createResponseScope`は旧Host操作を停止・取り消すものではない。
- **次の不合格対照**：DC-01選択Provider B・実Invoke A、DC-04 Source/Tool一部欠落、DC-02 CLIと`/workbench/`の実二Consumer、MV-05 Function Context同ID ABAは、既存#951/#1016/#2586/#2514のOwnerへ残す。UIだけの2 PASSをそのままD1b移行許可にはしない。

**確認済みテストの上限**：[merged #3284](https://github.com/hat47x/tei-platform/pull/3284)は9件の既存read-only CLI応答を基にした限定Design Context `partial`／資格保留と7つの合成Negativeを、[#3296](https://github.com/hat47x/tei-platform/pull/3296)はManifest追加2・Tool応答byte改変7を足した計16対照をCIで実証した。ただし独立の外部Harness応答切断ではなく受領済JSONコピーへの改変であり、Rule/Function/Business Required Closure、AIによる新Case生成、C1、外部Host／Studio実適合は未達。`qualifiedTestGeneration=false`を維持し、D1のSource閉包の下限診断として**参照**するだけとする。

**着手順**：P0 現`InspectSubject`→CLI／Wailsの実Call Graphと`Read`の外部効果/Disclosure棚卸し → P1 既存Contract Resolverから**exact一Provider**をshadow実呼出し、同一Identity/Scope/Result＋実Invoker記録をReadback → P2 誤Provider選択、無断fallback、不完全Source／Tool応答の負例を先行 → P3 CLIとStudio`/workbench/`を同一Operationへ切替えて実Node/DOM・Windows WebView2で別Consumerを検査 → P4 #3270 Data Workbench/#3274 HTTP Graph/#3267 Verified BOM/`*.tei.md` SHOへ用途別展開。#3292全体計画Gとの二重Hard Gate化はしない。

### D1の実Studio二画面と、新しく実装された宣言投影の受入境界
<!-- tei-desktop-d1-dual-studio-surface-and-unknown-effect-20261107 -->
- [main統合 #3334](https://github.com/hat47x/tei-platform/pull/3334) は [main統合 #3318](https://github.com/hat47x/tei-platform/pull/3318) の標準業務Data（`tax/exchange/numbering/holiday`）のRole付き例を受け、`ApplicationView.standardBusiness`→`tei view`テキスト→`cmd/tei-studio/assets/view-format.mjs`の「標準業務」表へ**宣言したData/属性の投影**を追加した。Go局所試験・Node formatter試験の成功はPR記録にあり、実Wails RPC/HTTP・実Rowの計算/実DBは未実証。このformatterはStudioの**bootstrap `/`画面**であり、既定`/workbench/`の`reference/tei-studio-workbench/main.ts`が同じApplication投影を受領した証拠ではない。
- **二Consumerを二画面と混同しない**：DC-02の対象はまず**CLI `inspect`と実`/workbench/`**という別Call-site。bootstrap `/`の`view`成功・Workbench `inspect`成功・CLI `view`成功はそれぞれのAction/Schema/Route/Source/公開Scopeを記録する。内容が似た表示だからPlugin-selected Provider actual Invoke、同じFunction Context、Read Slice、Host/DB Readerを通過したと認定しない。次に追加するなら明示した`view`の二Consumerを**別Case**とし、`inspect`のD1に混ぜない。
- [Draft #3327](https://github.com/hat47x/tei-platform/pull/3327)のexact HEAD `dd2c782d4127405e86c04ebca71e88b7db5873be`では、[Node24 Workbench CI #42385644277](https://github.com/hat47x/tei-platform/actions/runs/42385644277)で既存+ABAの**20/20 test、typecheck、build PASS**。しかし`dist/main.js`対Wails埋込み`assets/workbench/main.js`の**byte一致はFAIL**（649340byte地点）。他のStudio #8・Wails PoC A・Repository Health成功も埋込み整合の代替ではない。A→B→A応答抑止は限定合格だが、**Sourceと実配布資産が不一致のため該当PR全体は未合格**。Bundle Ownerがmain基線で再現・更新方式/原典一致を確認後、exact head CIを再検査する。静的/合成テストの成果を捨てる必要はない。
- [Draft #3328](https://github.com/hat47x/tei-platform/pull/3328)はExecutor後のRecorder／Lifecycle記録失敗と`MayReplay`の矛盾を`EffectUnknown`として保全する**Command層の修正候補**。D1 `inspect`はread-onlyでありこのPRに書込機能や新UI結果Enumを追加しない。将来、`/workbench/`からeffectful Commandを公開する場合は、`CommandRecordFailed`等の**Executor後unknown**、事前拒否、Receipt観測、resource再読取を別に表示し、旧UIの一律`failure`を安全な再試行承認としない。未知の作用は#1174/#2996/#2564でReceipt/Fenceを照合するまで自動再送禁止。
- [Open #3306](https://github.com/hat47x/tei-platform/pull/3306)の「Hookの`uses`に対応するCommand Patternが無い」Validator拒否は**実行可能性の宣言チェックの改善候補**であり、複数Commandの曖昧さ・位相対応・実Plugin/Provider/Studio invokeは未証明。#3334のBusiness Role表示やHook宣言のScreen/Print一致を「業務操作可能」と表示しない。

**この追加の適格性**：#3334/#3318のmain実装と#3327 CIの限定結果をreadbackし、Open #3328/#3306の計画/実装候補を分離。新しい実DB／Wails GUI／Function Context／Plugin actual Invoke/Effect試験・Owner受理はNOT RUN。D1a→D1bのexit Gateや既存Scenario SSOTを変更せず、**既存受入DC-02/DC-06への証拠注記**のみとする。

**停止線**：Provider実Invokeを一つも観測しなければ`not-connected`。Required Source/Parser/Role/Disclosureが欠落すれば`partial/unknown/unqualified`。Commandの実副作用が不明なら`failure`へ潰して再送しない。Core/Resolver/Scenario/Authority/Host/Registryの第二正本と正式新Schema/CLIコマンドを本節で新設せず、実Provider／正本意味の受理は各Ownerへ委ねる。
