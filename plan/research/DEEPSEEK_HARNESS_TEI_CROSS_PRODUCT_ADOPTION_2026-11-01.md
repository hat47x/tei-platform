# DeepSeek Harness（DSH）実装研究：TEIのComposition・AI-first開発・関連製品への適用

> **2026-11-01のソース確認に基づく設計研究。Research / Proposed / Owner未受理。TEI連携・DSH実行はNOT RUN。**
>
> 外部参照：DeepSeek公式リポジトリ [`deepseek-ai/deepseek-harness`](https://github.com/deepseek-ai/deepseek-harness)、調査時点の`master` commit **`ce7073d51254c26cd63b22a60835b8f948deb447`**。TEI`main` `43517af5450aacb2fdd6c48585dc143f52d57da4`、接続先の[AI-first / Headless-first Draft #2492](https://github.com/hat47x/tei-platform/pull/2492)。ここでいうDSHは**独立OSSのDeepSeek Harness**であり、DeepSeekのモデルAPIそのものや、CSWのfresh-context専用評価ハーネスとは分ける。公式は開発中のPreviewで、interface互換を永久保証しない。
>
> 主な既存責務：[ADR-0032](../adr/0032-composition-kernel-replacement.md)、[Plugin Composition](../design/AI_NATIVE_PLUGIN_COMPOSITION_DESIGN_2026-10.md)、[TEI #1223](https://github.com/hat47x/tei-platform/issues/1223)、[#951](https://github.com/hat47x/tei-platform/issues/951)、[#1121](https://github.com/hat47x/tei-platform/issues/1121)、[#2390](https://github.com/hat47x/tei-platform/issues/2390)、[#2492](https://github.com/hat47x/tei-platform/pull/2492)、[TAI #168](https://github.com/hat47x/tai-pleroma/issues/168)、[TAI #232](https://github.com/hat47x/tai-pleroma/issues/232)。

## 1. 研究の問い：TEIがハーネスになるべきか

**結論候補は「外側のDSHを使い、内側のTEI/SEI等は目的に応じた能力を提供する」**である。DSHのAgent Loop・Prompt・MCP Client・Subagent実行・セッション管理は既にある。TEIがそれを作り直すより、DSH等の複数Harnessから同じサービス契約・型付き操作・検証結果に到達できることに価値を置く。

ただしDSHから学べる要点は、モデルがToolを呼べること自体よりも、次の構造である。

- **全体を小さな交換可能な能力で組み立てること**。特に`Service Definition → Provider → Consumer`を配布物・実行実体・利用者の要求から分ける。
- **AIに見える機能の構成と、実際に許される作用の境界を分けること**。ToolはProviderの能力を広告するが、業務意味の受理、現在Authority、実効果を自動で保証しない。
- **モデルが見た入力・Toolの履歴を復元可能にすること**。ただし秘密情報の記録方式と、TEI/SEIの正本の歴史的再構成は別問題。
- **Pluginの導入・破棄がサービスのLifecycleを管理すること**。ただし一時Effectの解放と、DB/外部の永続副作用の取消、可用性保証は分ける。
- **同じ機能の複数の外側（Web/headless/SDK/ACP）を、Compositionで組み立てられること**。TEIの標準CUI/型付きOperationはGUIより先に固めるという既存方針を補強する。

**重複注意**：TEIは既に2026-10-12に[ADR-0032](../adr/0032-composition-kernel-replacement.md)を目標アーキテクチャとして採用し、`internal/composition/`でScope/Generation/ProviderのReferenceを持つ。既存[AI-native Plugin Composition設計](../design/AI_NATIVE_PLUGIN_COMPOSITION_DESIGN_2026-10.md) §8・§13にもDSHの調査がある。したがって今回追加するのは**最新の外側の連携・セッション/Tool/Evidence/機密性の差分圧力**であり、Cordis型Kernelの再提案ではない。

## 2. 一次資料で確認したDSHの仕組み

| DSHで確認した構造・振る舞い | 正確な位置づけ | TEIにとっての圧力・留保 |
| --- | --- | --- |
| **Everything is a plugin** | Cordis Context、typed event、revertible effect。モデルAdapter、Tool Registry、Session Log、Agent Loop自身がPlugin | [ADR-0032](../adr/0032-composition-kernel-replacement.md)の契約中心Kernelに整合。**TSの実行系やCore APIをコピーする決定ではない** |
| **Definition / Provider / Consumer seam** | Service契約の宣言、実現Provider、利用Consumerをそれぞれ識別。File SystemとSubprocessを同じExecution WorldへBinding可能 | [#951](https://github.com/hat47x/tei-platform/issues/951)のContract/Provider/Scope解決、[#2390](https://github.com/hat47x/tei-platform/issues/2390)の外部利用と同一問題。Host import・Target制約は独立 |
| **Profile / Bundle / Config Patch** | `web`, `headless`, `sdk`, `sdk-minimal`, `acp`等。Bundle→Profile→Home→CLI patchの合成。**Patchは同じIDの行全体の置換で、deep mergeではない** | TEIは合成前後のProvider/Effect/Tool reachabilityの差分を導出する。誤Patchによる権限・設定消失を負例にする |
| **Agent単位のScope** | 一つのAgentにだけTool/Prompt/Event/制限を提供し、破棄時に登録を解放 | TEIでApplication/Tenant/Invocation/Plugin Scopeの対応を検討。AgentのScopeを法人・業務権限と同一視しない |
| **Agent event / Tool pipeline** | `agent/pre-step` → `agent/request` → `tool/call` → `tools/pre-execute`/guards → `tools/execute`/post → `tool/result`。拒否・介入と結果通知が独立 | 既存Typed Operations、検証前/実行前/実効果観測を異なる境界に保持。Tool名/annotationによる自己申告は実権限でない |
| **Durable Session Event** | Session Logはモデル履歴の復元元。「Model-visible means logged」。中断/再試行/失敗した試行を含む状態遷移を分離。派生ProjectionやJSONL世代を管理 | TEIのSource/Decision/Build Artifact、SEIの認識・判断履歴とは別の**Harness側実行証拠**。コンテキストの一致を業務意味の一致へ昇格させない |
| **MCP Client** | opt-inで一つのServerを一つの構成行から呼び、MCP tools/resourcesを通常のDSH Toolsとして登録。MCP Serverは初期状態では接続されない | TEI側は最初CLI/typed read-only Operation、必要な場面でMCP Server Adapter。**TEIがMCP Client実装を標準で所有する必要はない** |
| **Subagent Provider** | in-process spawn/fork、Codex、Claude Code、ACP、DSH SDK等を同じProvider面から使える。親の文脈継承、実行ハンドル、取消/終了の扱いを区別 | [TAI #168](https://github.com/hat47x/tai-pleroma/issues/168)の外部Harness委譲を優先。SEIは専門Method/根拠条件、EKIは必要時の限定委任と独立Result |
| **FS / subprocess / Sandbox seam** | local/remoteのProvider交換でToolの実行世界を切替える。sandbox/approvalは別の能力 | [TAI #232](https://github.com/hat47x/tai-pleroma/issues/232)と同様、**実OS/HostでRaw accessとegressを拒否できるか**が証拠。Patch/Promptで秘密は守れない |
| **Headless JSON stream** | `dsh --profile headless --json`はイベントを行単位で返し、`--session-id`で条件付き再開できる。0=一つのturnが完了、1=abort/error等。JSON出力は**サイズ制限のあるProjectionで、完全なSession Logではない** | [#2492](https://github.com/hat47x/tei-platform/pull/2492)のAI-first headlessに適合。Exit0とToolの成功とTEIの業務結果は別。JSON Streamと原ログの区別を固定 |
| **Plugin Manager / Creator mode** | プラグインの導入・作成を製品Surfaceから利用できる | AIがコードを書けることとProviderの契約適合・Supply Chain・公開許可は別。TEIは構成提案→試験→正規の受理を保持 |

出典（調査時点の同一commit）：

- [Architecture](https://github.com/deepseek-ai/deepseek-harness/blob/ce7073d51254c26cd63b22a60835b8f948deb447/docs/architecture.md)
- [Capability seams](https://github.com/deepseek-ai/deepseek-harness/blob/ce7073d51254c26cd63b22a60835b8f948deb447/docs/capability-seams.md)
- [Tool execution pipeline](https://github.com/deepseek-ai/deepseek-harness/blob/ce7073d51254c26cd63b22a60835b8f948deb447/docs/tool-execution-pipeline.md)
- [Subagent](https://github.com/deepseek-ai/deepseek-harness/blob/ce7073d51254c26cd63b22a60835b8f948deb447/docs/subsystems/subagent.md)
- [MCP](https://github.com/deepseek-ai/deepseek-harness/blob/ce7073d51254c26cd63b22a60835b8f948deb447/docs/subsystems/mcp.md)・[MCP Client](https://github.com/deepseek-ai/deepseek-harness/blob/ce7073d51254c26cd63b22a60835b8f948deb447/packages/mcp/mcp-client/README.md)
- [Headless](https://github.com/deepseek-ai/deepseek-harness/blob/ce7073d51254c26cd63b22a60835b8f948deb447/packages/bundle/headless/README.md)・[CLI](https://github.com/deepseek-ai/deepseek-harness/blob/ce7073d51254c26cd63b22a60835b8f948deb447/apps/cli/reference/README.md)

### 2.1 モデルの選択とハーネスの選択を分ける

追加の一次資料：[DSH `llm-pi-ai` multi-provider adapter](https://github.com/deepseek-ai/deepseek-harness/blob/ce7073d51254c26cd63b22a60835b8f948deb447/packages/llm/llm-pi-ai/README.md)。このPluginは複数のProvider、OpenAI-compatible Gateway、自己運用Endpointを一つのAdapter面から扱い、Provider名・Model名・Wire API・CredentialsとReplay metadataを区別する。資格情報は`apiKeyEnv`等の**参照**として供給でき、Model SelectionとPlugin構成は別の変更単位となる。

TEIの設計研究では、**(a) DSHというHarnessを使うか、(b) DSH経由でどのModel/Providerを使うか、(c) TEIのどのService Contractを使うか**を独立した選択として測る。同じHarness上で違うModelでも、Tool宣言・Sandbox・Result検査の差を観測できる形にする。

OpenCode Goなど**特定の月額利用枠をDSHから共用できるかは、APIの到達性だけで決めない**。当該サービスの規約・認証方式・有効なEndpoint・モデル名・課金/利用枠を別途照合する。今回、実アカウント接続や利用枠の相互利用は未検証。Claudeのサブエージェント機能が存在することも、Claude Proの契約を第三者のAPIクライアントで使えるという意味ではない。

**根拠の強さ**：公式実装ドキュメントと指定版の文書内容を確認した。DSHのローカルInstall、起動実物、Tool/Plugin APIの試用、TEIとのE2Eは未実施。コード断片への依存は基準コミットの変更を再点検してから用いる。

### 2.2 追加の調査（2026-11-01）：Goal自律継続・Schedule・SDK-minimalの実効構成と外部送信

[Goal/Schedule/大量並列開発・機密送出の詳細調査](DSH_SCALE_GOAL_SCHEDULE_EGRESS_RESEARCH_2026-11-01.md)を分離し、実試験Ownerを[TEI #2509](https://github.com/hat47x/tei-platform/issues/2509)、多Workstreamの対照を[TAI #234](https://github.com/hat47x/tai-pleroma/issues/234)へ接続した。以下は指定版のDSH公式構成・説明資料で確認した**設計上重要な追加差分**であり、実起動観測ではない。

- **`sdk-minimal`と`headless`は機能・権限が異なる**。standalone `sdk-minimal`はGoal/Goal Round Driver/Tool GoalやSubagent、MCP Client、Scheduleを含まず、`sandbox-policy.mode: danger-full-access`を明示する。一方`base/headless`ではGoal関連Pluginを構成し、Headless側の既定Sandbox policyは`workspace-write`。いずれも実OSのアクセス可否は別実証。
- **Goal保存と自律再開は別**：一Session一現行Goal、round既定上限256（Token/金額予算ではない）。再起動・Fork後は`disarmed`で、明示re-armが必要。Goal完了は独立評価器が検証した完了ではない。
- **Scheduleは別責務**：Web側のScheduleはHostが稼働中にSessionへ予定の指示を届ける。停止中の複数回を完全実行・復旧する保証ではない。Task配達と実業務の結果を分ける。
- **外部へのRequest拡張送出**：`dsh-session-log-deepseek`は既定で`enabled:true`。公式DeepSeek APIへ、通常のモデル入力`messages`とは別の`dsh_session_log`にSession Eventを送信し得る（既定最大8MiB/Request）。無効化中のEventは再有効化時にBacklog送信対象になり得る。In-flight payloadは切替前に準備済みのまま送信される可能性がある。**`sdk-minimal`にもこのPluginが組み込まれる**。セッション/機密データに関わる実利用時は[TAI #232](https://github.com/hat47x/tai-pleroma/issues/232)でRaw/Network強制・送出範囲を独立検査する。

この知見は、安価なTokenによる探索範囲・Workstreamの拡大方針を弱めず、**並列探索を広げるための有効な実行単位（別Session/Issue/branch）と実通信・独立受理の条件**を具体化する。既存のCordis型Kernel採用判断と本番Runtimeは変更しない。

### 2.3 継続調査（2026-11-02）：Experimental Agent TeamsとSEI/TEI横断の用途別Context

DSHには[opt-inのExperimental Agent Teams](https://github.com/deepseek-ai/deepseek-harness/blob/ce7073d51254c26cd63b22a60835b8f948deb447/packages/experimental/agent-team/README.md)があり、Lead/Teammate、永続Task DAG・CAS更新、Agent間Messagesを構成できる。**同一Workspace・writeScopesはadvisory・自動Task Owner解放なし**という制約がある。標準Goal（一Session一件）のみをDSHの全並列能力と読み替えない。Team Task完了と独立したTEIのMeaning/Build/Host/Effect受理を分離する。

他チャットで進展した[TEI #2515（外部Harness実効隔離・全送出）](https://github.com/hat47x/tei-platform/issues/2515)、[#2516（Construction Contextの用途相対Projection）](https://github.com/hat47x/tei-platform/issues/2516)を、[詳細研究のT-01〜T-12](DSH_AGENT_TEAMS_CONTEXT_EXPERT_HANDOFF_2026-11-02.md)へ接続した。SEIが「必要な専門性／Source／Unknown」を評価し、DSHが利用可能なAgent/MCP/ACPへ照会し、戻る見解をSEIが候補として区別する場合、**fork/fresh/ACPへの情報投影、Teamの全体Broadcast、MCP instructionsの非信頼性、実OS権限**を別々に観測する。

この追加は新しい実Agent Host/汎用Task Coordinator/Context Contract正本を作るものではない。既存TEI/SEI/TAIのOwnerを活用し、Harness間で同じ意味・不確実性・開示範囲を保てるか比較する。**現状は文書読解・設計候補のみで実結合NOT RUN。**

## 3. TEI側：既に採用していること、DSHから新たに得られること

### 3.1 Composition Kernelは「参考」から「十分な検証」へ

[ADR-0032](../adr/0032-composition-kernel-replacement.md)は既に採用された目標アーキテクチャであり、以下を設計済みである。

- Service ContractはPlugin IDと物理成果物から独立。Provider適合、現在Host Authority、Scopeはそれぞれ別。
- Revertible Effectは終了時に資源を解放する。**コミット済みの永続DB/外部作用のRollbackではない**。
- GenerationをInvocation開始時に固定。Providerの計画更新はReadiness→CAS切替→旧in-flight drain→cleanupで可用性を検査する。
- 静的Target/Go/Wasm/IoTでは、不要なdynamic loader/HMR/Agent LoopをBuildから除外できる。

DSHの動作を加味した新しい比較課題は、(a) **構成パッチの差分をAIがどう理解するか**、(b) **一つのAgent/Sessionに許すCapabilityと実Hostの現在Authorityをどこで再照合するか**、(c) **Provider切替中のAgentのTool広告と実際のInvocationが一致するか**、(d) **監査時にTool/Shell/Model入力のどこまで復元できるか**、である。

一方、DSHのHMRやPlugin破棄の完了は、TEIの**無停止更新・生成成果物の構成変更・外部副作用の安全性**を自動で証明しない。[#1223](https://github.com/hat47x/tei-platform/issues/1223)のGeneration/Availability試験を正とする。

### 3.2 AIに発見させるのは「Plugin名」でなく利用できる能力の集合

TEIで実行前に公開すべき情報を、同一の正本から用途別に投影する。

| AIが必要とする情報 | 現行の参照先／強化先 | 危険な短絡 |
| --- | --- | --- |
| 実装したい仕事とデータの意味 | TEIのCanonical Model、Data/Function/Procedure/Action、Source・Trace | 同じフィールド名なら意味が一致する |
| 必要なService Contract | #951、既存の合成Registry | 名前が見つかれば実装済み |
| Provider候補とConsumer接続点 | `internal/composition`の候補解決と`SlotID`、依存世代 | Providerを自動選抜できる |
| 適合根拠 | #951のConformance、#943 Build input/artifact | schema一致や自己申告が受理済み |
| 対象Targetでの実行可能性 | #1121 Lowering、#2390 外部Consumer | ローカルPluginの存在をTarget実装へ一般化 |
| 有効な権限・作用 | #1174、#1118、実Host/Provider | Permission Prompt/Tool schemaだけで本番を許可 |
| 独立した結果 | #1129、#1180、#1119、実Provider | Tool完了を業務完了とする |

**AI向けIndex/Discoveryは、Sourceと実Registryから導出**する。別のAI専用Capability定義・Resolverを生やさない。LLMの説明負担、候補の見落とし、間違った共通化を同一Fixtureで測る。

### 3.3 Composition Profile：ソフトウェアの構成と意味の正本を分ける

DSH Profile/Bundle/Config Patchの特徴を受け、TEIでも四つを区別する。

1. **意味の定義**：TEI受理された業務/技術の意味、型、制約、操作。Composition自体は意味を更新しない。
2. **利用者・案件の選択**：必要な機能、対象Target、UI有無、データ/組織範囲、必要なHost機能。
3. **Build時の成果物閉包**：選ばれたContract/Provider/生成器/Source/Resource/Artifact bytes/Build optionsを固定。Bundle名だけでは十分でない。
4. **実行時の束縛**：正確なProvider instance/Generation、Hostの現在Authority、Resource/Lease、独立Result。

**DSHのwhole-row Patch**が示す負例：小さな設定変更の意図でPatchを適用し、既存の安全設定または依存行が落ちる。TEIのPatch表現をDSHと同一にする必要はないが、**Patch適用前後のeffective configuration、差分と失われたCapability/制限を機械で説明**できることを望ましい要件候補とする。

### 3.4 Effect/Lifecycleと持続的業務の結果

DSHの`tools/pre-execute`、実呼出し、`tool/result`、`agent/turn-stopping`、Plugin teardownを、TEIのどの責務へ写像すべきかを区別する。

```text
TEI semantic requirement & accepted operation
    -> proposed Plugin composition / tool affordance
    -> verified actual contract / artifact / target
    -> Host current Authority / scope / version at call and commit
    -> Provider execution and durable external side effect
    -> independently observed result + business outcome
    -> SEI/TAI review of unknowns and effects
```

Toolが拒否されたことと、モデルへの表示だけが拒否に見えたことは別。Toolが成功と報告しても、外部DB commit、Rollback失敗、外部ACK喪失、Lease失効、重複再試行を検査したことにはならない。`ToolGuard`はHost側の再検査を補強するが、**業務意味受理、現在Authorization/Transaction、Provider idempotency**を代替しない。

## 4. Headless-firstの具体的なDSH×TEI PoC（候補、NOT RUN）

最初の比較は**既存TEI CLIと既存の合成モデル/fixture**に限定する。DSH公式のLauncher経路を使い、独自Pluginが不要な段階は作らない。初期からDSHのCustom Agent Loopや第三者Pluginを必須にしない。

| レーン | 作業内容・固定する条件 | 正例 | 必須の負例・未保証 |
| --- | --- | --- | --- |
| **DSH-0 構成の棚卸し** | DSH指定Commit/配布版、実Node・OS、`--dump-config`の実Profile/Tool/FS/Sandboxを確認。TEIのCLI版・Source/Model revision・使用Toolを列挙 | 同じ入力でHeadlessとWebのTool資格の差を説明できる | Patchで旧設定が失われた、MCP Serverが実際は未起動、CLIが内部構造を無版で出力 |
| **DSH-1 CLI最小往復** | `dsh --profile headless --json`の一タスクから、利用許可されたShell経由でTEI read-only `commands`→`model catalog`/検証/参照を呼ぶ（公開済みCLI実コマンドを確認）。TaskとSource versionを固定 | Tool提出、実Exit、TEIのtyped output、根拠参照を分離できる | DSH Exit0だけでTEI検証PASSにする、`--json`をlossless Session証拠にする |
| **DSH-2 Plugin合成の差** | 既存RegistryでContract充足候補を求め、(A)静的Provider (B)Runtime Plugin交換 (C)別TargetへのBindingを比較。Provider実行は許可済みReferenceを利用 | 同じ意味契約と異なる物理実現、実行中世代Pinを説明 | Bundle名だけ固定、Provider drift、適合Evidenceなし、旧Generation Cleanupを本番完了とする |
| **DSH-3 MCP Adapter** | TEIのread-only操作一件をMCP Server Adapterから公開し、DSH側の既存`dsh-mcp-client`で接続。CLIと同一Contract/negative fixtureを使う。MCP Serverが未実装ならまず契約・試験計画だけにする | MCP列挙→呼出し→Tool結果→TEIの同一機能が、機械的に追跡できる | MCP Tool名重複、schema/contract drift、サーバー指示によるプロンプト汚染、Permission/認証欠落。**Tool列挙だけで機能成立としない** |
| **DSH-4 ハーネス交代** | DSH→Claude Code/Codex/OpenCode等へ、必要最低限の`source ref + revision + scope + unknown + proposal + evidence`を通常ファイルで渡す。SEIの履歴と区別 | 同じ対象版・未決・改訂条件を復元 | DSH session transcriptをDecisionへ昇格、未確認仮説/秘密Token/個人の内部思考を受渡し |
| **DSH-5 強いHost境界** | 選択的にProduct内部Raw storeを別UID/FS/Networkへ分離。DSH側にはread-only最小Projectionだけ。TEI/TAI #232の実OS評価を再利用 | 許された要約・契約検査だけ到達可能 | Tool一覧から隠しただけで直接FS/Port/Log/Clipboard/DB credentialに到達可能。外部送信を抑止できない |
| **DSH-6 UI操作の比較** | [#2391](https://github.com/hat47x/tei-platform/issues/2391)のA=Production外部操作/B=Test/C=Test+計測/W0=Wails内蔵MCPとの比較。TEI追加InstrumentationだけをProductionからBuild除外 | 同一Scenarioの意味期待を再利用し、Build closureの差を確認 | Test API/Debugger/MCP機能が本番生成物に残る、表示のTool resultと実UI状態が食い違う |

**実施の軽量な順序**：DSH-0/1→有用なら3・4→Runtime/Host検証の必要がある場合だけ2・5・6へ。複数レーンの研究・負例生成は並行可能。ただし正式採用の順序や既存Issue #1119のGateを再定義するものではない。DSHを使わず他Harnessで同じ仕事ができる対照も常に残す。

### 4.1 実行要求の最小の説明形式（概念例・現行CLIの入力形式ではない）

```json
{
  "task": "合成受注モデルの読取り・差分診断",
  "source": {"repoRevision": "<固定したSHA>", "modelRef": "<論理参照>"},
  "required": {"operations": ["discover", "inspect", "validate"], "effects": "read-only"},
  "constraints": {"sensitiveRawData": "unavailable-to-harness", "liveExternalAction": false},
  "expectedEvidence": ["actual-tool-invocation", "typed-output", "unresolved-items"],
  "handoff": {"doNotPromote": ["AI proposal", "tool success", "stale authority"]}
}
```

このPackageは、既存のtask packet/TEI CLI契約にある情報をまとめて**読むための評価例**である。新しい製品共通Manifest/Schemaや独自権限判定器を作る提案ではない。

### 4.2 リスクの高いことほど独立させる負例

- **不十分な権限**：DSHのAgent Scopeを変えてToolにアクセスできても、TEI/実Hostの現在Authority・作用・Tenant・Leaseでは拒否される。
- **入力と実Buildのずれ**：Plugin Lock/Build plan/Source SHAは一致しても、実起動した二次クラス・スクリプト・Wasmを含む閉包が一致しなければ同一成果物とは報告しない。
- **停止と再試行**：DSHの取消通知・Child stop reasonは外部DB rollback保証ではない。ACK-lossで再度呼ぶ前に独立Result確認を行う。
- **Session LogとPrivacy**：DSHはモデルに見せた入力をログから復元するが、これを理由にSEI/SUIの非公開原文をモデルへ送ってはならない。必要なら**局所Raw＋用途限定Projection＋開示範囲付きEvidence ref**を選び、モデルに見えた内容と閲覧権限を固定する。JSONL、stderr、Thinking、tool return、telemetryにも秘密が残り得る。
- **構成/Plugin供給経路**：Prompt経由で作ったPlugin、外部MCP Serverの説明、AGENTS.md、Skill本文は信頼元・検証済み契約と区別。モデルへの指示注入、秘密送信、Root越境の反例を設ける。
- **二重の正本**：Session Store（DSH）、Semantic State（SEI）、業務Model/Decision（TEI）、Activity（TAI）、Result Authority（Provider/EKI）を「全てイベントで記録する」だけで統一正本としない。

## 5. 関連製品へ活かす差分（役割と提案）

| 製品 | DSHから学べる技術・判断 | 優先する実用的な改善 | 導入時の条件 |
| --- | --- | --- | --- |
| **TEI Platform** | Service/Provider/Consumer・Scope/Generation・複数Profile・Tool pipeline | AIが契約・実現候補・根拠・不足を発見できるheadless CUI、Plugin版とSource/Artifact/Authorityの対応。Host非依存で同じContractを配布 | #951/#1223は既存。動的機構を全Targetへ強制しない |
| **SEI Cognition** | Model-visible contextの再構成、限定したWorking Set、Method/Providerの切替 | **専門知識不足や外部専門家の必要性を問う**→Harnessへ技能・MCP/Agent/Human照会を依頼→返った資料/意見をHypothesis/Source/Unknownとして受取。#660/#677。Session LogをDecision正本へしない | AI専門家≠有資格人間専門家、Toolの成功≠組織の正式判断 |
| **SUI Sensemaking** | Scope付きTool/イベント、Web plugin差込、差分/再開 | Canvasでの違和感/少数案/複数の見方を、元の人の記録とAI候補を分離してHarnessから利用できる読取/投影へ接続 | 未レビューCardや非公開関係の露出を防ぐ。Human reviewed状態の自動昇格はしない |
| **SHO/MMI** | SessionのProjectionと根拠の再現、ツール/成果物の区別 | 資料・図・数式の引用/出典と、AI草稿/納品固定版を別に扱う。長時間の文書生成はHarness Jobを活用、正式文書の受理は別 | Source Referenceの正しい原本再解決・表示依存・編集履歴の保持 |
| **EKI Fabric** | Subagent Provider・取消・背景Job・Tool結果 | 外部HarnessのWorkerを必要な資源/期限/権限で限定委任。独立Result Acceptanceと停止・部分成功の観測 | 既存Harness Jobで足りる用途に新Scheduler/Agentを重複実装しない |
| **TAI Pleroma** | 複数Harnessを同じTask Packet/能力に結び、外側を交換可能にする | DSH/OpenCode/Claude Code/Codex等の比較、案件再開、増分価値・コスト・人の介入負担を同一fixtureで測定 | TAIを新Agent Loopや共有Hostの必須正本にしない |

### 5.1 前の議論「専門家連携」に対する具体的な帰結

専門領域の判断を扱うときは、**必要な専門性を特定するSEIの認知機能**と、**適切なTool/Human/Agentを呼ぶHarnessの実行機能**を分けられる。

```text
SEI: 問い・法域・時点・専門性不足・必要な証拠/守秘条件を候補化
        ↓ request for specialized assistance（情報要求、権限付与ではない）
DSH等: available capabilityを探索（内蔵知識・MCP Tool・Subagent・人間照会）
        ↓ scopeと開示条件を満たした実調査/相談
外部: 出典・適用条件・助言・反対説・残余の不明を返す
        ↓
SEI: 新Observation/Interpretation/Unknown、Decisionへの候補
TEI/Provider: 正式業務意味・Authority・実外部作用は既存Ownerで別受理
```

初回はSEI→DSHの**説明的なTask/Context Handoffと結果回収**で十分かを確かめ、専門家専用のMCP/A2A ClientをSEIへ直接内蔵する判断を先送りする。専門Agentが外部MCP Toolとして動作する、既存CLIで十分、外部AIが別のHarnessでしか使えない等の差を**Unsupported / Unknown**として残す。

## 6. 計画とOwner判断への差分

- **既存[#1223](https://github.com/hat47x/tei-platform/issues/1223)/[#951](https://github.com/hat47x/tei-platform/issues/951)**：追加のKernel/Contract設計を起こすのでなく、Agent scoped capabilityと有効構成、世代切替時の広告/実Invocationのずれという負例を次のReference圧力として渡す。
- **既存[#2492](https://github.com/hat47x/tei-platform/pull/2492)/[#1155](https://github.com/hat47x/tei-platform/issues/1155)/[#2354](https://github.com/hat47x/tei-platform/issues/2354)**：DSH headlessの`--json`とTEI CLIのschema/失敗区分を比較。Toolの結果とSemantic/Ownerの受理を混同せず、生成AIに必要な発見機能のGapを調べる。
- **既存[#2390](https://github.com/hat47x/tei-platform/issues/2390)**：TEI提供能力をDSH以外のHostへも公開できるか、CLI/Library/Wasm/MCPの代替・選抜を比較。MCPは必須ABIとしない。
- **既存[#2391](https://github.com/hat47x/tei-platform/issues/2391)**：UI操作のInstrumented Buildを必要に応じて選択し、本番の配布閉包では物理除去する方式を継続。
- **既存[TAI #168](https://github.com/hat47x/tai-pleroma/issues/168)/[#232](https://github.com/hat47x/tai-pleroma/issues/232)**：Harness A/B comparisonと実OS境界にDSHを一候補として追加。MCP/CLI/Agentの接続証拠と生データ非開示を別に評価。
- **既存[SEI #381](https://github.com/hat47x/sei-cognition/issues/381)/[#660](https://github.com/hat47x/sei-cognition/issues/660)/[#677](https://github.com/hat47x/sei-cognition/issues/677)**：専門家へ相談すべき問い・文脈・証拠不足の認知と、DSH側のAgent/Tool実行を分ける。

### 優先順位案

**高優先・机上研究として実施可能**：DSH公式APIとTEIの意味/Tool/Plugin対応を固定Sourceで比較（本書）。AI向けDiscovery/CLI不足の既存#2354/#2492へ差分を戻す。

**次の実証候補**：DSHをHostとして実際に利用し、合成資料1件の`TEI CLI read-only`をHeadless JSON Streamで呼ぶ。確認するのは**DSH実行物、TEI CLI実行物、読んだSource、Tool Call、結果、未観測**の対応。OpenCode/Claude Code等からの同一操作を対照として選ぶ。

**差が示された場合にのみ追加**：MCP Adapter、専用Skill、Provider Hot-swap、長時間実行など。先にCLI/既存Capabilityを再利用し、必要性が確かめられてから実装先のOwnerを決める。

**Owner未決**：DSHを最初の代表Harnessにするか、複数Harnessの順序、Plugin配布方法、公式Contract採否、実秘密情報の使用条件、実本番作用の権限。旧新Kernel間の受理と本番移行は既存ADRのプロセスを尊重する。

## 7. 検証状況の表明

- **確認済**：2026-11-01時点のDeepSeek公式Docs上の構造、TEI mainおよび開発中Draft/Issueで扱う責務、同一公式commitにある参照文書。
- **提案**：TEI/SUI/SEI/SHO/EKI/TAIへの適用方法、DSH-0〜6の比較設計。
- **NOT RUN**：DSH実起動・Installer/Bundle、DSH PluginからのTEI/SEI等の実呼出し、MCP/ACP相互運用、実業務の結果、OS隔離、正式な意味受理、Security qualification、Ownerの製品採用。
- **保証しないこと**：DSHのEventログやTool結果の存在だけから、TEIの業務意味の受理、外部Providerの副作用・元データの機密保持、無停止更新、可搬性を推測しない。

研究から得た価値は、**「高度なAgent Loopを自作する」ことではなく、利用者・AI・Hostを入れ替えても、意味/実装/証拠/権限の区別を保持しながら能力を構成・利用できるTEIの基盤を育てること**にある。
