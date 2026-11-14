# 理想の利用像を製品の受入へつなぐ計画

- この計画は設計書の読書きから実行結果の確認、分野別の標準構成、導入・変更・引継ぎまでを対象とする。
- 作業と評価の入口は [#3796](https://github.com/hat47x/tei-platform/issues/3796)。G01〜G71はこの計画の引渡しを追うための項目IDである。G01〜G64を引き継ぎ、案件の一巡に必要なG65〜G71を加える。正式な製品要求や独立した試験番号は既存の正本を使う。
- 実装方法は[製品設計](../design/PRODUCT_COMPLETION_DESIGN.md)、着手順の基準は[開発全体計画](../MASTER_PLAN.md)、現行の実証は[実装状況](../../docs/status.md)で確認する。

## 完成と判断する単位

一つの案件で、必要な設計を人が読み、許された範囲を変更し、選んだ実装で実行し、利用側の結果を独立に確認できることを判定する。案件に必要な導入、失敗時対応、更新、引継ぎも受入に含める。

受注では確定した値、公開では利用側から読める版、外部連携では相手側で観測した状態を確認する。受付、コード生成、単体試験、参照実装の成功は、それぞれ確認した範囲として保持する。案件の必須条件を満たした結果と結び付けてから成立に数える。

### 統合後、次の実装で確かめること（2026年11月14日）

[横断統合 #4683](https://github.com/hat47x/tei-platform/pull/4683)と[実対象の回帰試験統合 #4684](https://github.com/hat47x/tei-platform/pull/4684)によって、以前の計画・一部の生成コード・安全性試験はmainに入りました。ただし、**統合できたコード、通常の製品入口から動いた機能、実際に利用先が受け取った結果**は別々の成果です。以前のDraftの状態をそのまま現在の実装状況として読まないため、直近の着手先を以下に固定します。

| 優先順位と着手先 | 最初に通す実装・検証 | 合格と呼ぶ前に残すもの |
| --- | --- | --- |
| **全体P0：Studioの実セルフビルド**（#2831／#4583） | 非PluginのStudio Build Subjectを既存BuildExecution／ArtifactSetへ結び、**Stage1の実ProcessがStage2 Desktopを生成**し、そのStage2をWindows WebView2で起動する | [Windows部品試験 #4587](https://github.com/hat47x/tei-platform/pull/4587)の成功だけではStage2再構築とみなさない。実Child、Builder、Toolchain、生成物のByteと受領先を同一の試行で照合 |
| **並行P0：通常製品入口の実Read**（#2586／#1251） | RA-0／1／2RのReference結果を入口の基線とし、RA-2Pで通常`tei`→選択Provider／実DB／SQL・型付き引数→Function Data Contextを実行。RA-3でTableとAggregate等の異なる二Consumerへ渡す | [RA-0 Draft #4626](https://github.com/hat47x/tei-platform/pull/4626)のE1 `sqlgen`依存拒否（#1182）を残し、参照PGの成功やContextの`partial`を実Consumer・全件取得の成功に読み替えない |
| **CLI先行の小実証：TEI自身による生成**（#2621／#951／#2831） | 既存の[CLI先行計画 #4676](https://github.com/hat47x/tei-platform/pull/4676)を使い、同名Node二つ・別IDの多重有向EdgeとRuleを、Sourceから**実際の判定ロジック**へ生成。向き変更／ID重複／レイアウトのみの変更を対照にする | `*.tei.md`の正式Readerが未対応なら`unsupported`。Goへの委譲だけを自己実装と数えない。選択Pluginの実Call、生成コードの由来、固定した義務の分母、再生成・保守負荷を別記録 |
| **P1：Plugin説明と公開モデルの探索**（[#4679](https://github.com/hat47x/tei-platform/issues/4679)／#951） | 上位A・利用側Bの二Pluginを固定し、Aが公開したContract／Graph／Manual索引へ、Bのread-only Inspectから正しい版で辿る。短いProduct Skillは既存の実コマンドの発見方法だけを案内 | 非公開文書・図・上位依存の存在情報を漏らさない。閲覧可を再公開可・実行可へ昇格させない。新Docs Registry／未実装CLIコマンドを前提にしない |
| **P1：横断Concernと専門計算の適合**（#4198／#3509／#4660） | main統合済みの[Application Graph設計 #4681](https://github.com/hat47x/tei-platform/pull/4681)では、まず一つのRoute／Entryに対するread-only適用表を作る。計算側は[SQLite Window実DB試験 #4670](https://github.com/hat47x/tei-platform/pull/4670)を基線に、必要な入力範囲とProvider能力を調べる | 計測の挿入・同意・二重発火、外部Solver、NoSQLでの同等性、通常CLIの実計算・Consumer受領は未検証。新Aspect Weaverや第二Query Engineを導入しない |

最初の各PRには、**既存のSourceと対象版、具体的な正例・反例、実行された入口・Builder／Provider、独立Expected、確認できたConsumer、未検証の範囲**を短い一枚の引渡しとして残す。Ownerの意味受理、現在のHost Grant、外部Effect、実DB Migrationは別資格とし、推測による補完はしない。図面・Markdown表・Graphvizは標準データへの可読な表現として使い、表示上の変更を意味や実行権限へ昇格させない。

### Script LaunchとBackend配置の製品受入：目標構成と現在の基線を分ける（2026年11月14日）

11月14日の横断検討では、Studio Desktopの**将来の設計目標**を「Wails ClientがローカルBackend子プロセスを管理し、IPCで共通Operationを利用する」構成として整理した。一方、現在の**製品P0の検証基線**は、Wails Nativeの同一プロセス構成である。[Studio配置比較 #4635](https://github.com/hat47x/tei-platform/issues/4635)と、統合中の[Script Launch設計 #4688](https://github.com/hat47x/tei-platform/pull/4688)／[Backend設計 #4690](https://github.com/hat47x/tei-platform/pull/4690)は、この二つを混同せず、子Backend採用を新しいStage1→Stage2の必須Gateにしない。両設計は[統合PR #4697](https://github.com/hat47x/tei-platform/pull/4697)でレビュー中であり、統合を製品実行の新たな合格とは扱わない。

**CLIではADR-0048のin-process既定を維持**する。Wailsには依存させず、軽量・短命の操作は共通Go Application APIを直接呼ぶ。Plugin隔離や長時間処理が必要な場合の同一実行ファイル子Backend、Harness向けの明示Session再利用、接続先を指定したRemote CLIは比較対象であり、既定の変更ではない。Remoteで使うPluginと現在Grantは接続先Serverが所有する。CLI側へ同じPlugin版を要求したり、通信断でLocalへ黙って切り替えたりしない。

| 実証単位 | Ownerと着手対象 | 最小の合格条件／拒否条件 |
| --- | --- | --- |
| **SL-0：現行基線** | #1158／#1824／ADR-0048。通常tei CLI・Native Wails・共通Go Operation | ListWorkspaces／InspectSubject／ValidateWorkspaceのSource、Subject、Revision、Failureを同一Fixtureで照合。CLIのWails非依存と直接実行の維持 |
| **SL-1：Product Function実呼出し** | #4495／#4502。Owner受理済みpure Entry一件 | Typed API→Host-selected actual Go Providerと、通常tei実Process→同じ意味Contractを独立Attemptで検証。Reference実行やMock／宣言だけのProviderでは不合格 |
| **SL-2：配置の比較** | #4635／#3419〜#3421。Native一体と明示local child | 実Child image／argv／親子所有、専有Endpoint／Peer、Protocol／Operation Contract／Server世代、同一Sourceの診断を比較。同一exe Worker、loopback、Pipe／UDSの優劣は実測で判定 |
| **SL-3：Pluginの実利用** | #951／#1251／#2044。Server側Composition／Load | Discovery→Lock→選択→実ロードProvider→actual Callを同一Attemptで照合。Clientが送るPlugin名／Role／Grantは実行資格ではない。RemoteはServer側選択版を正本とする |
| **SL-4：Session・復旧** | #4635／#1174／#1129。短命CLI Child／明示Session／Studio子Backend | Process Instance、Server generation、Session、Invocation、Attempt、Operation、Effect Receipt、Consumer応答を混同しない。Child Crash後の未保存BufferとPin保持、旧世代返信の拒否 |
| **SL-5：TEIの自己生成** | #2831／#4583／#2621。CLI実Codegen→Stage1→Stage2 | SourceのGraph／Ruleが生成コードと動作を決定。Stage0による二度Build、Go委譲のみ、Builderの自己申告を除外。Windows WebView2受領は独立Gate |
| **SL-6：異質Consumer** | #1158／#3421／#2586／#4679。MCP／Remote／Studio／Browser | 意味が同一でもAudience、Host Grant、Read Scope、Plugin版、Service Generationを個別に再確認。Manualの閲覧可をEntryの実行権限にしない |

**子プロセスの生存と、業務Operationの完了は別**である。親はChildの起動・監視・終了と孤児化防止を担当し、子は選択済みPlugin／Providerの実利用とHostによる現在資格の確認を担う。Childの再起動は、前のInvocationを再送する許可ではない。更新・切断・取消・Commit後ACK喪失で外部作用が不明なら、既存FailureのeffectUnknown／lookup_firstに従い、Operation IDとReceiptの独立照合まで自動再送しない。Read-onlyの再取得でもSource RevisionとAudienceを再固定し、旧Cacheを現在の情報として表示しない。

最低限の反例は、(a)別Child／再利用Endpoint、(b)Handshake後のPlugin Lock変更、(c)選択Provider B・実呼出A、(d)親終了による孤児Child、(e)作用後Crash・ACK消失、(f)再起動後の旧Resultが別Subjectへ届く、(g)Remote接続断後の暗黙Local fallbackである。既存#1180/#2514のExpectedへ引き渡し、第二Scenario正本・万能Session Registryを追加しない。

**優先順位と判定上限**：全体P0のStudio Stage1→Stage2、並行P0の実DB Function Read、CLI先行のTEI意味Codegenは変えない。SL-1の製品pure Entryは#4495の並行課題とし、SL-2〜4の子Backend比較・Session復旧を先行Hard Gateにはしない。今回の追補は計画・実装引渡しのみであり、Child IPC／Session再接続／Product Function実Call／Plugin更新／CLI2再構築の新しい試験は**NOT RUN／NOT ACCEPTED**。

### SL-1の最初の実コード差分と、受理可能な証拠（2026年11月15日）

Script Launch §25、ADR-0048、#4495に基づき、最初は**公開済みのpure Function一件を直接Go APIから実Go Providerへ呼ぶ**差分を閉じる。次に、同じ意味契約を通常の製品tei実行ファイルの別OSプロセスから呼ぶ。これは二つの独立Attemptであり、プロセス・Request IDが同じであると仮定しない。Stage1→Stage2セルフビルド、実PostgreSQL Function Read、Studio子Backend/IPC、MCP、外部Effectはそれぞれの別Gateに残す。

**実装の着手前条件**：Ownerが受理したFunction ID、公開Entry/Contract Version、入力の意味型（不在／明示NULL／0を区別）、純粋性を保証する依存範囲、対象Host/選択Go Provider、Independent Expectedを一つの票に固定する。数量2×単価300＝600などの例は候補であり、公開・純粋性・現在Grantが未受理なら製品実装のPASSに使用しない。適格候補が無ければ「公開Entry未受理」と明示してOwnerへ返す。

| 最小差分 | 使う既存Goの接合位置 | 新たに確認すること |
| --- | --- | --- |
| **SL-1a：直接API→実Provider** | 既存のComposition/Host選択、typed Application API、internal/semantic/capabilityinvoke.Invoke | Source/Meaning/Entryと、実ロードArtifact・selected Provider・actual Invoke・BusinessResultを同じAttemptで照合。Generic AdapterのPayload anyを業務値の型検証やHost Grant発行に使わない |
| **SL-1b：通常tei CLI別Process** | cmd/teiの既存起動分岐、#1158のTransport-neutral Operation | 製品Entryから同じ意味契約へ到達し、入力／出力Body・Failureを独立Expectedと比較。cmd/tei/cli_view.goのReferenceExecutor/MemoryUnitOfWorkの成功、実行ファイルのexit0、JSONの値一致だけでは不合格 |
| **SL-1c：負例と実Call位置** | 既存#4495 ELケース、capabilityinvokeのGo focused test、#1180/#2514 | Entry非公開・Grant失効・不正な実InputはAdapter call=0。selected B/actual Aを結果同値でも拒否。実Adapter call=1後の未定義Outcome等はResult契約違反とし、外部Effect不在や安全Retryと判定しない |

**Goの検査で成立した範囲**：[PR #4712](https://github.com/hat47x/tei-platform/pull/4712)は8ワークフローの成功を確認したうえでmainへ統合済み（squash `d21210d6`）。`capabilityinvoke.Invoke`の実装を変えず、Adapter後の未宣言Outcome／異Capability／Mode不一致と取消について、呼出し`call=1`、不正Resultの拒否、取消errorの同一性を検査する。既存の実行前取消`call=0`と対になる**局所Go回帰試験**であり、公開Entry・Host現在Grant・selected/actual Provider・通常tei CLI・実EffectのProduct E2Eには拡大しない。

最初のPRには、**実行した正確なHEAD、通常製品バイナリと実Process、固定したEntry/Contract/Source Revision、Host/選択Providerと実Provider/Artifact、Call Count/Attempt、具体的なInputとResult Body、Expected、実Consumer、未確認事項**を短い一枚に残す。Runtimeに万能Execution Envelopeや第二Registryを追加せず、必要なEvidenceを既存OwnerのReceipt/Source/Build/Provider記録から参照する。

この一件が合格するまで、子Backendへの実行移設やPlugin Hot ReloadをP0の前提にしない。Graph/Ruleの意味変更→実Go生成→通常tei実動作の差分検証（#2831/#2621/#951）には同じ受入の考え方を適用する。ただしmainの[限定Graph読取り参照実装 #4722](https://github.com/hat47x/tei-platform/pull/4722)で成立したのは、Markdown表のNode/Edge候補・Source byte span・同名別ID/多重辺・独立Expected照合までである。製品`*.tei.md`のLanguage Plugin/Workspace Reader、生成された実ロジック、実Plugin／CLI Consumer、Stage1→Stage2は未成立。CLI実Provider成功をTEI由来Codegen成功、Go delegateのみを自己構築、pure Function成功を実DB Commitへ昇格しない。**本追補は実装着手票であり、新たなProduct Entry/実Provider/製品CLI E2E/Owner業務意味受理はNOT RUN／NOT ACCEPTED。**

## 不足を戻す担当

各担当Issueには以下の範囲に属するG項目の作業と正負の受入条件を引き渡す。実装・実行結果はそのIssueと既存のEvidenceに記録する。

| 範囲 | 製品として揃えること | 担当 |
| --- | --- | --- |
| G01〜G07 | 全体案内、通常の編集、診断、改名、復旧、実行の追跡、応答性能 | [#1124](https://github.com/hat47x/tei-platform/issues/1124)、[#3788](https://github.com/hat47x/tei-platform/issues/3788)、[#3789](https://github.com/hat47x/tei-platform/issues/3789)、[#3790](https://github.com/hat47x/tei-platform/issues/3790) |
| G08〜G12 | 同一性、関係、数量、欠測、項目Profile、時点、導出と失効 | [#2495](https://github.com/hat47x/tei-platform/issues/2495)、[#3816](https://github.com/hat47x/tei-platform/issues/3816)、[#3791](https://github.com/hat47x/tei-platform/issues/3791)、[#2600](https://github.com/hat47x/tei-platform/issues/2600) |
| G13〜G20 | 実Read、状態・判断、合成保証、編集保存、現在権限、外部連携、公開確認 | [#2586](https://github.com/hat47x/tei-platform/issues/2586)、[#1125](https://github.com/hat47x/tei-platform/issues/1125)、[#3278](https://github.com/hat47x/tei-platform/issues/3278)、[#1174](https://github.com/hat47x/tei-platform/issues/1174)、[#3796](https://github.com/hat47x/tei-platform/issues/3796) |
| G21〜G29 | 実Provider選択、入出力、構成と配布、独自差分、外部接続、移行、Target、初回利用、セルフビルド | [#1251](https://github.com/hat47x/tei-platform/issues/1251)、[#951](https://github.com/hat47x/tei-platform/issues/951)、[#3267](https://github.com/hat47x/tei-platform/issues/3267)、[#3270](https://github.com/hat47x/tei-platform/issues/3270)、[#1168](https://github.com/hat47x/tei-platform/issues/1168) |
| G30〜G36 | 確定データ、固定・可変・混合版面、PDF/Excel、既存帳票、画像移行、発行運用、符号の実読取り | [#3815](https://github.com/hat47x/tei-platform/issues/3815)、[#3820](https://github.com/hat47x/tei-platform/issues/3820)、[#3798](https://github.com/hat47x/tei-platform/issues/3798) |
| G37〜G44 | 標準表現型としての複合表現、混在座標、多次元、時間、入力・逆編集、Browser/Wasm、科学処理 | [#3827](https://github.com/hat47x/tei-platform/issues/3827)、[#3821](https://github.com/hat47x/tei-platform/issues/3821)、[#4133](https://github.com/hat47x/tei-platform/issues/4133)、[#4122](https://github.com/hat47x/tei-platform/issues/4122) |
| G45〜G48 | 色、精度、資源の上限と代替、VR、実物との状態照合 | [#3827](https://github.com/hat47x/tei-platform/issues/3827)、[#3790](https://github.com/hat47x/tei-platform/issues/3790)、[#1174](https://github.com/hat47x/tei-platform/issues/1174) |
| G49〜G53 | 高度B2C、多言語・アクセシビリティ、AI拡張、用途限定文脈、CLI/MCP/VUIの共通操作 | [#3509](https://github.com/hat47x/tei-platform/issues/3509)、[#3796](https://github.com/hat47x/tei-platform/issues/3796)、[#2515](https://github.com/hat47x/tei-platform/issues/2515)、[#2516](https://github.com/hat47x/tei-platform/issues/2516)、[#3503](https://github.com/hat47x/tei-platform/issues/3503) |
| G54〜G60 | 独立検証、証拠の帰属、多分野の要求化、案件対応率、総費用、資産の再利用、計画の更新 | [#2514](https://github.com/hat47x/tei-platform/issues/2514)、[#1163](https://github.com/hat47x/tei-platform/issues/1163)、[#3791](https://github.com/hat47x/tei-platform/issues/3791)、[#3796](https://github.com/hat47x/tei-platform/issues/3796)、[#2018](https://github.com/hat47x/tei-platform/issues/2018) |
| G61〜G64 | DSL文法と互換、外部形式の取込・交換、オフライン再同期、継続運用と復旧 | [#1063](https://github.com/hat47x/tei-platform/issues/1063)、[#2621](https://github.com/hat47x/tei-platform/issues/2621)、[#3270](https://github.com/hat47x/tei-platform/issues/3270)、[#2597](https://github.com/hat47x/tei-platform/issues/2597)、[#3796](https://github.com/hat47x/tei-platform/issues/3796) |
| G65〜G71 | 業務資産、検索、通知、外部契約、継続イベント、デバッグ・模擬実行、利用者のライフサイクル | [#4426](https://github.com/hat47x/tei-platform/issues/4426)、[#4427](https://github.com/hat47x/tei-platform/issues/4427)、[#4428](https://github.com/hat47x/tei-platform/issues/4428)、[#2542](https://github.com/hat47x/tei-platform/issues/2542)、[#3274](https://github.com/hat47x/tei-platform/issues/3274)、[#3789](https://github.com/hat47x/tei-platform/issues/3789)、[#3412](https://github.com/hat47x/tei-platform/issues/3412) |

## G49 B0の実装Referenceから実Source・Native Consumerへの引渡し（2026-11-12）

- **実装済みの限定足場**：[PR #4506](https://github.com/hat47x/tei-platform/pull/4506)は修正後HEADの健全性・Cache・既知脆弱性CI成功を確認してmain統合（`8c3d1a55`）。`tools/reference-b2c-change-inspector`の明示Graph/Scopeの到達検査と固定Expectedは**合成read-only Reference**であり、実Nativeコード解析・公開許可・Provider/Commerce/Experiment検証ではない。G49受入計画の[PR #4499](https://github.com/hat47x/tei-platform/pull/4499)と[PR #4512](https://github.com/hat47x/tei-platform/pull/4512)もmain統合済み。
- **B0-S0（同一Loadの実Source観測）**：[PR #4517](https://github.com/hat47x/tei-platform/pull/4517)はexact-headの全体健全性・Cache・既知脆弱性検証成功後にmain統合（`2374f338`）。既存`workspace.Load`＋`BuildSourceIndexFromLoad`＋`InspectionFromLoad`を用いる**局所Go試験**で、Source A→B→A、Load後のファイル変更、Workspace manifestの同一性差分、別Root拒否と実Queryの出典を対照する。**製品B0-S1の実Native依存解析の合格ではない**。Load時ByteからのIndexは原子的な多ファイルSnapshot、Imported Provider/Native依存の閉包、Current Grantの証明ではない。
- **B0-S1（次の実装担当）**：[Issue #4518](https://github.com/hat47x/tei-platform/issues/4518)で`Workspace + Qualified Subject/occurrence + Source revision + 既存Operation/Context`から、案件が明示したNative Touchpoint/Action/Consumer Bindingへ限定投影する。Required/Observed/Resolved/Unresolved/Excludedを**用途と質問に相対的に**診断し、未探索のJS/CMS/Edge/外部Tagはunknownとする。Native UIへの項目追加はDB列の作成ではなく、Data/Function Mappingがない場合は未対応として診断する。明示的なData項目削除でMappingが失効してもUI要素は残り得る一方、未取得・非開示・partialを削除として扱わない。`SELECT *`／`alias.*`／`NATURAL JOIN`などの暗黙的な列集合・結合条件への依存は、明示列の参照が0件でも影響候補とし、実Schema/Queryの取得範囲が不足する場合はunknownとする。既存#2582のClaim単位Required Setに従い、影響外と証明できるContextは保持し、影響範囲だけを再検証する。未MappingのNative UI項目はDraftでは保持できても、必要なBindingを欠いた公開・業務Actionを検証済みとしない。削除・改名・DDLの実Applyは既存Data/Schema Ownerが所有し、Source/Scope/Selectionの再確認なしに自動付替えしない。完全なNative Source解析、第二Source Reader/Graph Resolver/Hostは実装しない。
- **問い相対の欠落と再利用**：既存[#2582](https://github.com/hat47x/tei-platform/issues/2582)のRequired Set／Consumer Scopeを利用し、当該変更の到達先・明示的Requiredに接続する欠落は未解決として保持する。一方、無関係な未知Node・未解決Edgeが存在するだけで全Consumerを失効させない。[PR #4530](https://github.com/hat47x/tei-platform/pull/4530)は合成read-only検査の過剰失効と必要依存の見落としを正負テストで抑止し、exact-headの健全性・Cache・既知脆弱性CI成功後にmain統合済み（`8d9a0d97`）。`WHERE`にのみ使う非表示列、`NOT EXISTS`等の否定条件、集計・結合とSource/Schemaの版差を「明示出力列0件＝影響なし」に含めない。実Queryの演算依存は#2387のOwnerに残し、B0に第二SQL解析器を加えない。
- **次の最小受入（B0-S1a）**：[#4518](https://github.com/hat47x/tei-platform/issues/4518)で公開合成Native PDPの**一つの明示Field Binding**を、既存Workspaceの**一つの実Canonical Subject/Source occurrence**と同一Reader結果で照合する。#3262が所有するRequired Universeの列挙根拠、#2586のProvider選択と実Read、実Consumerに到達したResult Scope、#4503/#2516の用途別開示・実送出、現在のGrantを、別の証拠欄として記録する。Toolのページ・End markerが欠落した応答や未取得Native/CMS依存を「閉包済み」としない。対象が一意に解決してもSourceの値Span・書込権限は別条件であり、[#4533](https://github.com/hat47x/tei-platform/issues/4533)の操作単位逆編集の実測をNativeからの自動Source書戻し許可へ拡大しない。**新CLI/API名、第二Reader/Registry、Nativeの必須TEI Runtime、公開Gateは導入しない**。
- **B0-S1aの局所到達とB1aの受渡し（2026-11-13）**：[PR #4540](https://github.com/hat47x/tei-platform/pull/4540)をexact-head CI3種成功後にmain統合（`305e024c`）。一つの宣言Native PDP Fieldを実Workspaceの`semantic-attribute`／Source occurrence／Source bytes／Workspace Rootへ同一Loadで束縛し、誤Target・Source driftを拒否する**Go test-onlyの証拠**を得た。Native/Edgeは入力者が申告した仮説で、ブラウザー・Provider・Consumerの実動作を観測していない。対して[#2586](https://github.com/hat47x/tei-platform/issues/2586)の[PR #4539](https://github.com/hat47x/tei-platform/pull/4539)は、実PostgreSQLへのEnvironment束縛・Query実行・Raw Rows→Function Data Context・独立Reader比較を使い捨てDB上で試行し、exact-head CI成功後にmain統合済み（`ee1af44c`）。**製品の`tei` CLIではなく、別プロセスの限定Reference CLIによる受領**である。Product `tei` CLI接続、独立Native Consumerの実受領、接続先の物理同一性（現状host-asserted）、現行Grant・Owner受理は未実証として残す。B0-S1aの`native-pdp.order-id → semantic-attribute:sales.order.order_id`は、#4539の`order_entry.recent_orders`/`:from`や物理`public.orders.order_id`との正式なMappingではない。近似名・同じSQL Source digestだけでFunction入力・物理列・Provider対象を自動結合しない。#4539では`:from`がQueryの`input`として未宣言（`declaredByQuery=false`）であるため、Native側のTyped Input契約の確認が残る。次のB1aは**同一Attemptの実Product Query/Providerと、Nativeまたは独立CLI Consumerが実際に受け取ったContextを照合**する。RLS/列権限/マスク/0行での`unestablished`、ページ・終了マーカーの欠落、実ReadとBrowser公開・業務Effectの違いを保持し、別Readerによる一致を実Consumer受領へ昇格しない。
- **R0/R1の負の依存とSource存在（B0-S1追加負例）**：入力に`BaseRevision/CandidateRevision`があっても、B0-Rの`Edges`は一式の申告Graphであり**両世界のEdge差分や実Consumed Closureの観測ではない**。R0にあった`Source → Function/Consumer`の依存辺をR1で削除・別Subjectへ再束縛した場合、R1の変更元からConsumerへ到達0でもConsumer不変とは断定しない。旧辺の依存先を変更候補・未知のまま保ち、R0/R1のSource・対象・方向・Scopeを揃えた比較と独立Expectedを[#4518](https://github.com/hat47x/tei-platform/issues/4518) N16／[#2582](https://github.com/hat47x/tei-platform/issues/2582)へ渡す。`SELECT *`／否定述語／Dialect別SQLの新規候補追加も**変わらない元SQL byteだけでは選択・結果・作用の不変を証明しない**ため、Queryの実選択は#2622/#2586 HJ-08で扱う。また、単一Subject参照0件とSubject不存在を分け、[実測Issue #4577](https://github.com/hat47x/tei-platform/issues/4577)と修正PR #4584のCI・採用状況を独立確認する。第二Query Parser/Graph Solver/Registryを作らない。
- **問い相対の旧依存辺の検収（N16/N17）**：明示されたR0-only依存辺から旧Consumerを影響候補に残す限定Referenceは[main #4591](https://github.com/hat47x/tei-platform/pull/4591)へ統合済み（`8709aae7`）。ただし、旧辺すべてを変更元とみなすと**当該Required Consumerとは無関係な削除辺**や、別のRelease ConsumerへのR1下流分岐まで誤って失効させ得る。[PR #4592](https://github.com/hat47x/tei-platform/pull/4592)でR0-only辺とR1候補辺の**問い相対Required上流集合**だけを旧辺変更の対象とする修正・二段旧依存削除・無関係下流Siblingの独立Expectedを提案中。明示`changed`の通常到達は別扱いにする。N16/N17の申告Graph上の成功はR0/R1実Source読取・Imported Source／Native/Queryの完全な依存探索の証明ではない。#3262がRequired Universeの取得根拠、#2582が局所再取得を所有する。#4592のCI・main統合は別確認。
- **Studio同一HEADの実Windows B0とBuild Intentの断線**：[統合Draft #4587](https://github.com/hat47x/tei-platform/pull/4587)の実Windows CIは、一つのexact HEADで生成Inspector readerの使用・配信Workbench JS byte一致・Native Wails WebView2を確認（[#4583 B0](https://github.com/hat47x/tei-platform/issues/4583)）。ただしDraftであり、**N0 Inspector Contract→N1 PoC B Go/TS→N2 esbuild/Embed→N3 nested Studio Wails exeの全工程がTEI BuildIntent/実Builderへ宣言・実消費対応したことは未実証**。B1には変更前のBuild依存辺が削除された場合の実Consumed残留、無関係な旧Sourceの過剰再Build、別HEADの成功の継ぎ合わせをそれぞれ独立負例として追加する（[#4583のBD-07〜09](https://github.com/hat47x/tei-platform/issues/4583#issuecomment-6846881454)）。B2 Stage1実TEI Process→Stage2 Desktop、B3 Stage2 WebView2は別のNOT RUN資格。TEI Studio全体のセルフビルドはB2Cの新Hard Gateを増やさず#2831/#1824で優先する。
- **B0-Rの宣言入力検証とStudio優先度（2026-11-13）**：既存のJSON入力が重複項目・Case Shadow・容量超過により曖昧だった問題は[main #4565](https://github.com/hat47x/tei-platform/pull/4565)で修正済み（`8a119731`）。UnicodeのSimple Fold迂回は[#4580](https://github.com/hat47x/tei-platform/pull/4580)の未マージ修正候補で、確認時のexact-head健全性CIは**gofmt不一致で失敗**。製品全体の最優先は[#2831](https://github.com/hat47x/tei-platform/issues/2831)／[#1824](https://github.com/hat47x/tei-platform/issues/1824)の**実TEI Studio Desktopセルフビルド**であり、[生成Inspector部品の実Windows試行Draft #4572](https://github.com/hat47x/tei-platform/pull/4572)と[生成Desktop WebView2検証Draft #4579](https://github.com/hat47x/tei-platform/pull/4579)は区別して扱う。全体計画main[#4581](https://github.com/hat47x/tei-platform/pull/4581)に不足したBuild graph／実Builderの接続工程を反映済み。B2Cは同じFunction Contextの異質Consumerとして並行し、Studio P0への新規Hard Gateを設けない。
- **順序と証拠の上限**：#2586の実Function→実Query/Provider→Rowset→Function Data Context→ConsumerをP0に維持し、B0-Sの完成をHard Gateとしない。#4350 DF-X06は標準表現/Nativeの異質Consumerの同一意味を検査。#4496/#4515は固定Subject/Scope/Read epoch/操作資格を所有し、#4507/#4514のSource-only driftでは同byteの復帰とSource/Authorityの現行性を区別する。B1a実Native Consumer、B1b実Commerce Effect/別Read、B2実配信/Exposure/Consent、B3のCVR・利益因果評価は**別の未実証資格**。
- **性能・プラグイン・Evidence**：#2018でH0 Native単独/H1 TEIのBuild時契約/H2任意Experiment Adapterを、同じ変更・独立Expectedと導入/改訂/運用費用で比較する。#951/#4198/#3278のProvider/Graph/Traitは実選択と実呼出しを照合。#4307/#2516に従い非公開Evidence・その存在情報をGitHubや外部Harnessへ必須送出しない。公開の権限、Consent/Grant、現在価格、実注文Effect、売上改善をB0で宣言しない。

## 最新の作業単位へ接続する

2026年11月11日のmain `9d31f82486b2eac42791f4f3bc5cd588d69cb4fc`と関連Issueを照合した。以下は着手先の更新であり、Issueの起票・限定Reference・候補Parserを製品の完成へ数えない。以後の進捗は各Issueと[実装状況](../../docs/status.md)から再取得する。

| 既存の項目 | 現在の足場と着手先 | 次に満たす条件 |
| --- | --- | --- |
| G08〜G11/G37〜G42 | [#4350](https://github.com/hat47x/tei-platform/issues/4350)のS1〜S3（[#4351](https://github.com/hat47x/tei-platform/issues/4351) / [#4352](https://github.com/hat47x/tei-platform/issues/4352) / [#4353](https://github.com/hat47x/tei-platform/issues/4353)）は合成Headless Referenceとしてmain統合済み。横断Matrix [PR #4464](https://github.com/hat47x/tei-platform/pull/4464)と限定Source編集 [PR #4538](https://github.com/hat47x/tei-platform/pull/4538)はOpen（2026-11-14照合） | 読取り用構造写像の判定・用途別Observer・操作Slot単位の逆写像・現在Grant/実Consumerを分離する。11/32セルの合成判定やSource編集の隔離試験を実Provider→通常`tei`/Studio→独立Consumerの製品合格へ転用しない。単位・軸・完全性の未宣言はunknown/unsupportedで残す。Graph→TreeのNode属性Slotが写像全体の損失で一律拒否されるGapは保持し、既存Checkerで操作相対の条件を表せるまでは自動許可しない |
| G61 | [#4359](https://github.com/hat47x/tei-platform/issues/4359)が構文比較と実装順を所有。比較票は[#4361](https://github.com/hat47x/tei-platform/issues/4361)、候補Parserは[#4362](https://github.com/hat47x/tei-platform/issues/4362) | Parse、既存意味への変換、Source位置、互換性、正式採用を順に区別する。DSL案を製品Loaderの既定へ先に入れない |
| G19/G25/G61 | [#1130](https://github.com/hat47x/tei-platform/issues/1130)のETL/Batch設計は[PR #4409](https://github.com/hat47x/tei-platform/pull/4409)へ、取得・変換の最初の縦断は[#4410](https://github.com/hat47x/tei-platform/issues/4410)、再開・外部実行は[#4411](https://github.com/hat47x/tei-platform/issues/4411)へ分割済み | 同じ固定入力と独立ExpectedをGo/CLIで照合する。新しいETL専用DSLや必須Schedulerを加えず、再実行と外部作用の結果を分ける |
| G21/G23/G28 | プラグイン管理の一覧・診断は[#4376](https://github.com/hat47x/tei-platform/issues/4376)、計画・差分は[#4377](https://github.com/hat47x/tei-platform/issues/4377)、設定源と原子性は[#4378](https://github.com/hat47x/tei-platform/issues/4378) | 実Workspaceからの解決と選択、実呼出し、更新・失効の結果を別に観測する。管理画面の一覧を実Provider適合へ昇格しない |
| G54 | [#4134](https://github.com/hat47x/tei-platform/issues/4134)の検証義務候補P0-Aと、固定有限データP0-Bを[#1131](https://github.com/hat47x/tei-platform/issues/1131)/[#1180](https://github.com/hat47x/tei-platform/issues/1180)へ接続 | 生成した試験の数と判別力を分ける。独立Expectedを固定し、満足例、単一違反、相互作用、観測不足と失敗の最小化を対照にする |
| G13 | [#2586](https://github.com/hat47x/tei-platform/issues/2586)がP0。旧PR #4076はclosed/unmerged。[PR #4539](https://github.com/hat47x/tei-platform/pull/4539)はEnvironment経由の`public.orders/:from`実PG読取、一時的Function Context、別プロセスReference CLIを実装し、exact-head CI成功後にmain統合済み（`ee1af44c`）。使い捨てPG16での局所実測も記録。ただし**製品`tei` CLI・独立Native Consumer・現在Grantは未接続** | Referenceの`public.orders/:from`とXTEAの`sales."order"/:status`は別Query。#4539 mainの実Read/Scope/接続先身元・Typed Param・Rowset・Context/独立Readerを基線に、RLS/0件/欠落列/マスクで`unestablished`を保持する。次に製品`tei` CLI、同一Attemptを受領する独立Native Consumerを順に検証。PG局所PASSやCI成功を現在Grant/全件取得/Checkoutへ転用しない |
| G49（B2C Native変更のCoverage） | [#3509](https://github.com/hat47x/tei-platform/issues/3509)の[最小B0判定票](https://github.com/hat47x/tei-platform/issues/3509#issuecomment-6813494503)をG49の初回受入候補に接続。案件固有Frontend/Edge/CMSは標準Rendererの採用を強制せず、実Source/Artifact R0→R1と既知のOperation/Context/Consumerだけをread-onlyで関連付ける。Native未登録部分の依存関係を自動で完全とみなさない | 既存BC-16/18/19/27/29/32/35/36/37/39/41を使い、通常文言C0と割引・同意条件を含む変更C3、旧Variant/旧Cartの正負を同一Fixtureで比較する。Change identity、Coverage、複数Lane候補、Required verification、Observed evidenceを別票にし、Coverage不足からNo Impact/公開安全/CVR向上を宣言しない。B0は**設計・試験候補**。#2586の実Readを先行し、H0/H1/H2の生産性比較は#2018、Commerce実Effect/実Browser/収益効果は後段で個別に検証 |

### 2026-11-13 データ様式の製品受入：問いに必要な根拠と操作範囲を限定する

既存[#4350](https://github.com/hat47x/tei-platform/issues/4350)の段階計画に沿い、次の差分だけを優先する。これは**新しい共通Schema・Dependency Engine・Runnerの採用ではない**。

- **問いごとの依存閉包**：[#4530](https://github.com/hat47x/tei-platform/pull/4530)のB2C限定Referenceでは無関係な未知依存の過剰失効を抑え、main統合済み[#4550](https://github.com/hat47x/tei-platform/pull/4550)でRequired Consumerの上流にある**多段の**未知入力も検出対象となった。ただしCaller申告Graph内での局所的な検査であり、Source Universeの完全性ではない。TEI全体でも、用途・対象・Scope・Required Source/Relation/SQL/Plugin・取得時点を先に固定し、必要な否定依存を確認する。Required集合が空または根拠不明なら「完全」としない。全世界の完全性をこの局所修正から推論しない（#3262/#2582/#1143）。
- **二Consumerの別々の適格性**：#4539の`Cell=value/null`や`rowset.fullyRead`は取得した行の観測であり、欠損/非選択/非開示の区別、集合の完全性、NULL由来、金額の単位/精度/丸め、外部Disclosureを保証しない。表の行表示と集計の個体・順序・多重度・数量法則について、既存[#2497](https://github.com/hat47x/tei-platform/issues/2497) OQ-01/02/03/05と#2514の独立Expectedを**同一Context・同一Attempt**で比較する。集計値を業務確定値としたり、Chartの一点を元行への一意な更新先としない。
- **読み取りと編集の条件を混同しない**：#4538の`set-attribute-type`限定実験では、Source定義・Span・Revisionを一意に決めた機械的編集と、Graph→Tree全体の非可逆性を分離する必要性が明らかになった。一律に「損失あり=全属性編集不可能」とする過剰拒否は解消候補とするが、Source一意性・対象Slotの情報保持・無関係部分の不変性・現在資格を実証するまでは、操作を自動許可しない。ABAと実Grantは未実証のまま残す（#2497/#2500/#1124）。
- **検証の順番と停止線**：R0a（#4539のmain統合・実PG/CI確認済み）→R0b（通常CLIで実ProviderとContext）→R1（異質二Consumerの受領と独立Expected）。Editorの参照・修正候補（#3788）、`*.tei.md`近傍直接テスト（#2621）、Studio Pin（#4515）、Native B2C（#4518）は別の並行作業として扱い、完了待ちをR0の新しいHard Gateにしない。

実PostgreSQLを用いた限定結果を含む#4539はmain統合済みだが、**この追補による通常製品CLI・二Consumer・実UI・現在Disclosure/Grant・意味受理の新試験はNOT RUN**。PRの状態と試験の達成範囲は、それぞれのOwnerと実行記録で判断する。

### 問いと操作に応じた判定の具体例（既存Expectedへ接続）

写像の分類（preserved/lossy等）は残し、判定対象の**問い・必要Observer・取得Scope・単位と時間・要求操作**を先に固定する。以下は新しい試験正本やCore判定語彙ではなく、#2497 OQ、#2514、#2586 R0P/R1C、#1143 RN、#2582 SC-IMへ渡す具体的な対照である。

| 既存対照 | 入力・問い | 独立に確認する期待結果 |
| --- | --- | --- |
| OQ-02／R1C-01 | #4539のPQ-02で3行を読了し、表が「取得した行は何行か」を問う。次に同じ結果で「全注文は3件か」を問う | 前者は観測した3行と答えられる。後者は`coverage=unestablished`のため確定できない。`rowset.fullyRead=true`を母集団の完全性へ転用しない |
| OQ-02/03／R1C-03 | 同じQueryでもRLSで可視行が3→2に変わる。また0行やマスクされたNULLを観測する | 可視行数と全体件数、DB保存NULLと秘匿由来NULLを区別。可視2行から全体のCOUNTや存在しないことを確定せず、第二ConsumerのChart・順位・診断にも開示条件を適用 |
| OQ-01/05／R1C-02 | JOINで同じ注文行が二度現れる、あるいは同じ行集合を順序なしBagへ写す | Rawの行重複・個体識別と結果粒度を保持。`SUM`・`COUNT DISTINCT`・`first`はそれぞれ独立の要件。観測行の数値和が計算できても、金額・通貨・丸め・全件範囲が未確定なら業務の確定集計としない |
| OQ-01／#2544 | 明示Fixtureで人物P1の所属0.6/0.4 FTE、P2の所属1.0 FTEを集計する | **単位FTE・対象期間・所属の重複処理を明示した場合**、人物2・所属3・FTE合計2.0を区別。現在のTEI SourceでFTE Unitや期間を宣言していなければ、この数値を製品上の受理済み意味として使わない |
| OQ-05/08／#4538 | Chartの集計点とGraph→TreeのNode属性Slotを編集候補として比較 | Chartの点・Graphの表示Edgeを元行/Relationへ直接書き戻さない。一方、個別Node属性は一意なSource Subject・Attribute・Span・版・操作の根拠が揃えば候補として検証できる。写像全体の`lossy`だけで全属性を永久に不可とせず、根拠の無い自動許可もしない |
| RN-07／OQ-06／SC-IM | Source A→B→A、別Workspace同ID・同byte、旧Pin/AI提案の後着 | 表示値やContent hashが戻っても、現在のSource/Workspace/対象/開示・編集資格を再確認するまでApplyしない。旧Replyで自動再束縛しない |
| OQ-04／RN-03 | `NOT EXISTS`/`SELECT *`などの暗黙依存が変化、または無関係なPlugin/列だけ変更 | 問いに必要な否定依存・Schema/Parser/Pluginが未知なら当該Claimを未確認とする。無関係なUnknownまで全Claim失効にしない。#4530の局所結果を全TEIの依存完全性証明へ昇格させない |

**最初に実測する組合せ**はPQ-02の同一Readを表Consumerと集計系の独立Headless Consumerへ渡すOQ-02/05でよい。まず両者が**同一Context/Attemptを実際に受け取った証拠**を残し、未確定のSUM/不存在/書戻しは正しく保留できるか調べる。FTEやGraph→Treeの実Source編集は後続の個別Ownerで検証する。実Consumer・数量/編集Owner受理・現在Grantの新しいPASSはこの表では主張しない。

### Native項目と実Query・Result Consumerをつなぐ際の対応根拠

2026年11月12日夜に[PR #4540](https://github.com/hat47x/tei-platform/pull/4540)がmainへ統合され、公開合成Native PDPの一つのFieldを、実Workspaceの`semantic-attribute:sales.order.order_id`・Source occurrence・Workspace Root・Load時Source bytesへ限定的に束縛する**Go test-only**の検証が成立した。これはNativeコード/DOMの自動解析でも、FunctionやSQLの実呼出しでもない。対象の[main統合済みPR #4539](https://github.com/hat47x/tei-platform/pull/4539)では`order_entry.recent_orders`の入力`:from`が実Query定義では未宣言であり、同じWorkspaceに`order_id`属性があるというだけでNativeの型付き入力、`public.orders.order_id`列、実Providerの接続先を推定できない。

次の小さな受入では、**Native Source Subject→明示Function/Operation Input→Queryの宣言・Parameter Shape→選択Environment/Instanceと実呼出Provider→Rowset列/Attribute Mapping→Function Data Context→Consumer実受領**の各辺を別々のEvidenceとして照合する。後段まで確認できない辺は`unbound`/`unobserved`/`unknown`のままとし、名前一致・同一Digest・別Readerの値一致で補わない。R0実Readの同一AttemptをB1aへ再利用できる場合のみ実Consumer接合を後段で試す（#4518/#2586/#2497/#3262/#2516）。

既存Expectedに足す**一因子の負対照**：①Native `order_id`と未宣言`:from`を同じ列/入力と自動接続しない、②別Workspace同名属性・旧Source occurrence/版を拒否、③同じQuery Source digestでも実Connection/Instance・Grantを確認できなければ受領資格は保留、④別独立Readerの検査成功をNative Consumerの実受信と誤報しない。Source修正前後の型付きMapping/Source位置の照合が不可能なら、公開/書戻しは保留する。新API/Registry/Parserを設けず、Provider Read #2586を優先しB2C Nativeを新たなHard Gateとしない。

**全体優先目標との関係**：TEI Studio Desktopのセルフビルド（#2831等）は引き続き独立最優先課題。このデータ様式計画は実Read・Trace・Consumerの検収で基盤品質を高めるが、Native B1aや二Consumerすべての完成をセルフビルド開始条件にしない。
### 三要素分析法・CRUDの派生結果を表現する際の適格性

main統合済み[#4534](https://github.com/hat47x/tei-platform/pull/4534)では、明示CRUD定義の矛盾を全件報告し、`okScope=structural-diagnostics-only` と `crudConflictStatus=conflicts-require-review` を分けた。構造診断`OK`、CRUD表が生成できたこと、業務上の整合・意味受理は同じではない。X-TEA由来のData/Function/Businessの図・CRUD cross-referenceも、Sourceへの参照・逆参照とレビュー要否を保持してからStudio/SHOへ投影する（#1124/#4472）。

さらに現行`main`の`internal/threeelement/crud.go`には、[#4551](https://github.com/hat47x/tei-platform/pull/4551)の明示CRUD-onlyのFunction×Data行の補完と、矛盾行の`conflictingDeclarations`、`UnusedData()`の明示CRUD参照を考慮する修正が反映されている。ただし、これは**既存三要素モデルのread-only派生とその局所試験**の範囲であり、Studio/SHOの全Consumerが矛盾を不適格停止として扱うことの証明ではない。最低限、(a)明示-only、(b)Operations由来、(c)同一ペアで双方宣言、(d)矛盾定義、(e)同Kind同IDの重複、(f)宣言が指す未知Functionを対照として、表示行数と参照数/unused候補を独立Expectedに照合する。`read-only派生成功≠実CRUD Provider Effect≠現在編集Grant`を守る。新しい三要素モデル正本や第二CRUD生成器は作らない。

### 異なる取得元・表現形式・実装Targetを横断する際の検収

この段は、#2506のCX-C01〜10、#2497のBPR-01〜05、#2498のXT-01/02および既存#2514/#1180の独立Expectedへ引き渡す**検収条件**であり、新しい標準Codec・全形式Adapter・共通Proof Engineの採用ではない。

| 比較対象 | 具体的な反例・検収条件 | 到達範囲と担当 |
| --- | --- | --- |
| **取得の完了** | `rowset.fullyRead=true`（一Cursor読了）、REAのTool/Evidence `complete`（その取得・Pageの読了）、Functionの必要母集団に対するCoverage、Consumerの開示可能性を別々に示す。Tool absent／欠けたPage終端／RLS／0行／部分対象で全件不存在に昇格させない | #2506 CX-C01〜10、#2586、#3394、#2516。main #4539の完了はCursorのみ。外部REA main #4547のObservation/Relationは意味受理でない |
| **生値と直列化値** | Raw `[]byte{0xff}`と`[]byte{0xfe}`はJSON化で同じ置換文字になり得る。現`functioncontext.cell()`の`[]byte→string`とJSON化したRowの重複検出では、Raw byteの異同・Binary Identityまで証明できない。異なるbyteを同一行や同一キーとして確定しない | #2497 BPR-01〜05、#2506、#2498。Product bytea/invalid UTF-8での実テストとLossless Codecの採用は未実施。元Rawとの一致が未確認ならConsumer適格性は保留 |
| **Presence・軸・Quantity** | Protobuf implicit scalarと`optional`のPresence差、xarrayの次元名/座標alignment、Arrow Mapの業務Key一意性、地理CRSの軸順を値/shape/表示順だけで推定しない。組織合成Fixture固有の`raw 100=1 FTE`は金融の100bpとは違い、Stock/Flow・JOIN多重・加重平均・時点・丸め順序を別に検査 | #2498 XT-01/02、#2495/#1705/#2544、#2593。外部ライブラリでの限定比較とTEI Product Adapterの受入は別。新しいCore Unit規約は本追補で採用しない |
| **生成物と実Consumer** | 同じPQ値を返すDriver/PG ReadとSource生成Go CLIでも、選択Provider・Target/Snapshot・Generator/Artifact Revisionが違えば同じAttemptとは言えない。Studio生成Inspector readerの実Windows利用（Draft #4572）と、三要素分析法のモデル用Bridge Pluginのserver画面到達（Draft #4570）を、TEIの一貫したDesktopセルフビルド/実Wails三視点受入と同一視しない | #4559/#1121、#4572/#2831/#1824、#4570/#1124。既存Trace・Receipt・独立Expectedを使い、Stage2 bundlingと実Desktop UIの未達を残す |
| **宣言の解釈の一意性** | 同じJSONキー・大文字小文字違いのキー・容量上限後のtrailing入力で、`declaredScopeClosed`等の意味を後勝ちで変えない。入力が一意に読めても、必要SourceのUniverseが本当に完全とは限らない | #4518の限定Referenceで#4565の負例を再利用。正式DSL/JSON全読込の実適用範囲は別Owner。現在Grant/Effectの証明には使わない |

**最初の着手順**：主線はmain #4539から通常`tei` CLI R0b→同一Contextの二Consumer R1。次にBPRの生値保持とCX-Cの完了判定をConsumerで反証し、Protobuf等の異種Codecは二系統程度から段階実測する。Studioの生成部品・三視点Bridge、組織FTE、外部REAは並走とし、どれもDesktopセルフビルドやR0bの新Hard Gateにしない。外部ライブラリ観測やDraft PRの局所PASSを現mainの製品機能へ格上げしない。

### 「空・不存在・未利用」を誤認しないSource束縛とDesktop検証

**参照の0件は成功ではない場合がある。** `tei source-index`に明示したSubjectが実Sourceに存在しないのに`ok:true`＋空配列を返す現象が[#4577](https://github.com/hat47x/tei-platform/issues/4577)で確認され、修正は[Draft #4584](https://github.com/hat47x/tei-platform/pull/4584)にある。**Subject未発見/重複、Scope内で参照0件、全Indexが実際に空、権限・未取得・partial**をそれぞれ区別する。特にStudioの固定選択（#4515）、項目レベル参照/未使用表示（#1143）、SHO図面/CRUDのApply判定（#1124）では、指定Subjectの取得失敗を「参照0＝未利用」や「対象なしにつき変更安全」に変換しない。正常なSubject特定→当該Source Indexの観測範囲→参照数→各Consumer利用資格の順に、独立Expectedを置く。

**入力解釈の一意性は依存閉包の前提であり、閉包の証明ではない。** mainへ入った[#4565](https://github.com/hat47x/tei-platform/pull/4565)はJSON重複・大小文字影・容量外トレーリングを拒否するが、`strings.ToLower`では`scope`/`ſcope`（Unicode simple-fold同値）によるGo `encoding/json`の上書きを再現できず、追加修正は[Draft #4580](https://github.com/hat47x/tei-platform/pull/4580)で検証中。`DeclaredScopeClosed`やRequiredの上書き、未知Sourceの偽閉包をそれぞれ判別し、Reference入力に限った修正を正式DSL/API全体の同等保証と扱わない（#2514/#4518/#3262）。

**実Desktopで部品が動いたこととStudio全体のセルフビルドを分ける。** [Draft #4579](https://github.com/hat47x/tei-platform/pull/4579)のWindows Wails/WebView2実CIは生成Workbench/Go ServiceをWebView2が受け取ったことまで観測し、[Draft #4572](https://github.com/hat47x/tei-platform/pull/4572)は生成Inspector readerの実Desktop使用を報告した。一方、[Draft #4570](https://github.com/hat47x/tei-platform/pull/4570)の三要素分析法Pluginはserver版の合成Source到達であり、Desktop実Window/実X-TEA原典は別資格。さらに[#4582](https://github.com/hat47x/tei-platform/pull/4582)のSelfBuild S2境界は**書込み共有のみ**制限し、読取り共有・実Builder来歴・Stage1生成Childを証明しない。次のDesktop検収は**実生成Source→Bundleのbyte/版→Wails埋込みArtifact→実Consumer受領→Stage1/Stage2 Buildの帰属**を分け、部分的な実Windows PASSを製品全体の自己再構築完了へ昇格しない（#2831/#1824）。

**旧依存辺を問いに必要な範囲で追跡する**：現mainの[#4591](https://github.com/hat47x/tei-platform/pull/4591)は、Caller申告の`baseOnlyEdges`でR0にあった辺の`to`を変更候補に加え、R1下流の影響を保存する。一方、このままでは**無関係なR0-only辺まで変更扱い**となり得るため、[#4592](https://github.com/hat47x/tei-platform/pull/4592)がRequired ConsumerからR0/R1の宣言辺の和集合を上流へたどり、影響が必要な旧辺に限る修正を提案中。既存#2514/#2582の独立Expectedで「二段の旧依存鎖の削除→当該Consumerは影響/上流不明を保持」と「別Consumerの旧辺削除→当該問いを過剰失効しない」を同時に区別する。**#4592は未統合**のため、現在の実装が両者を満たすとは扱わない。いずれも申告Graphに限る局所検査であって、実R0 Source/Worldの差分観測・完全なDependency Universe/現在Grantの証明ではない。G49のN16とSource不在区別を既存計画に保持し、第二Query/Graph Resolverは設けない（#4518/#2582）。

**同一HEADの生成Desktop受領を到達点として数える**：[#4587](https://github.com/hat47x/tei-platform/pull/4587)のexact HEAD `b26e87781e00b19bf142b22a35cfc62a857e88a3`では、生成Inspectorの契約digest/使用回数と生成Workbench JSの実WebView2配信byteを、同一Windows Wails Desktop Jobで検査して**completed/success**を得た。これは従来の別HEAD成功を足し合わせた証拠ではない。ただしPRは未統合で、Stage1自身がB1 Build Catalog/Intentを通じてStage2 DesktopをBuilderで実BuildするB2、そのStage2生成物の独立WebView2検査B3、実X-TEA Source、Current Grantは未受入。#4583/#2831の既存Gateへ証拠を渡し、この計画に別Build/Scenario正本を設けない。

この3点は既存#2514/#1180・RN/SC-IMとStudio/自動テストのOwnerが所管する反例であり、新たな共通Index/Parser/Build Planner/承認Engineを追加しない。#2586のR0b実Provider CLIと#2831 Desktopセルフビルドは引き続き最優先、#4584/#4580/全Plugin統合を両者の新Hard Gateとはしない。

### G08〜G13/G37〜G42：データ様式の用途・操作相対適格性を実Product経路へ接続（2026-11-14追補）

[#4350](https://github.com/hat47x/tei-platform/issues/4350)は段階・依存の索引、[#2495](https://github.com/hat47x/tei-platform/issues/2495)/[#2544](https://github.com/hat47x/tei-platform/issues/2544)はFacet/演算意味、[#2497](https://github.com/hat47x/tei-platform/issues/2497)は写像/損失、[#4133](https://github.com/hat47x/tei-platform/issues/4133)は表現型、[#2506](https://github.com/hat47x/tei-platform/issues/2506)/[#2586](https://github.com/hat47x/tei-platform/issues/2586)はFunction Data Context/実Read、[#1124](https://github.com/hat47x/tei-platform/issues/1124)は設計Source編集、[#2500](https://github.com/hat47x/tei-platform/issues/2500)は業務データのViewChange→Operation Intentを所有する。本追補はその**製品受入順序**だけを明確にし、第二の意味正本・型Registry・Projection Engine・Action/Grant/Host・Scenario SSOTを作らない。

**判断を混同しない**：A) 構造写像の`verdict`（既存5語彙）、B) **目的・問い・Observer・Scope・許容損失に相対的なread適合**、C) **対象Subject/Relation occurrence・Slot・具体Operationに相対的な逆編集候補**、D) **現在Source/Context epoch・権限・実Target/Effectの資格**は別列とする。一般の`lossy` Graph→Treeでも、明示ID・一意値span・恒等的に表示されたNode所有属性の局所更新は候補になりうる。一方、Edge IDなしのreparent/unlink、集計値の明細への自動配分、同ラベルからのIdentity復元は拒否する。これは既存Checkerの全写像`inverse=ineligible`を直ちに`eligible`へ変更する指示ではない。Slot/Operationスコープと独立Expectedを先に比較する（[#4533](https://github.com/hat47x/tei-platform/issues/4533)、[PR #4538](https://github.com/hat47x/tei-platform/pull/4538)）。

**同一性と現在性**：`Subject ID`／`Edge occurrence`／`Source byte span`／`View occurrence`／`Selection epoch`／`Workspace`／`Provider snapshot`／`Grant`を混ぜない。Raw byte fingerprint一致はA→B→Aの履歴同一性、現在の意味受理、Writer権限を証明しない。編集Apply直前に現在Source/Revision・span値・Writer権限・独立Requiredを再照合し、旧AI Replyや別Contextの選択から書き戻さない。設計SourceへのPatch/Acceptanceと実DBへのAction/Commit/Receipt/独立再Readは別の縦断で検証する。表現の`null / missing / unfetched / withheld`と集合の`empty / partial / complete / unknown`、Queryの非表示・否定依存（`NOT EXISTS`等）は用途相対の必要閉包を保持する。

| 順序 | 既存Owner・既存ケースを使う最小縦断 | 合格として要求する証拠 / 主張しないこと |
| --- | --- | --- |
| **D0：Reference契約の整理** | #4351〜53を基線に、#4464の11/32セル、`best`用途条件、`unknown/unqualified/uncovered`、#4533/#4538のSlot編集差を照合（XMAP-02/03/07、RSP-02、DF-X03、VE-01/03）。 | 同じ構造損失でもreadとSlot Editの判定が区別でき、既知Mutantと独立Expectedで偽許可・偽拒否を発見する。未検査21セルを安全/不可能と決めない。 |
| **D1：実際に読取り、別の利用先で確かめる** | #2586/#1251が担当。型付きSourceのRA-0と実行前検査のRA-1は修復後の仮統合試験[#4644](https://github.com/hat47x/tei-platform/pull/4644)で再検証済み。RA-2Rでは使い捨てPG16の7ケースを非SKIPで実行。次はRA-2Pで通常`tei`入口から実Provider・SQL・行・Contextを結び、RA-3で異なる二つのConsumerがそれぞれ受け取った値を検証する（DF-X01、HJ-03〜05）。 | RA-0のE1依存境界の失敗は別に残る。RA-2Rは参照入口の成功であり、製品CLIや受領者の成功ではない。入力不備は実行前に拒否し、実接続先や取得範囲は観測して判断する。空結果、RLS、NULL、部分取得と非開示を、独立した期待値で区別する。 |
| **D2：設計Sourceの局所編集** | #1124/#2497とPR #4538で、表/Propertyの一意な型付き属性Patch→既存Preview/Review/Accept→実Source再Read/再投影を接続。Graph→TreeはNode属性の正例と多重Edgeの負例を追加（DF-X03）。 | 異Workspace同名、旧Revision、A→B→A、Edge ID欠落、集計編集、非可逆投影を独立に判別し、無関係byte不変・失敗時書込0を確認。限定合成Passを現在のOwner意味受理へ昇格しない。 |
| **D3：業務Function操作・異種Target** | #2500/#1174で同じ明示Entity/Relationに対するIntent候補→現在Grant→実Effect Receipt→独立Read。#2498/#4133/#3827、SHO/SUI、外部Java/.NET Hostは用途に必要な異種Consumerを順に追加（DF-X04〜08）。 | Selection・UI配置からAction/Authorityを創出しない。実Host/DB、Owner Effect、外部Consumer、媒体同期の各Claimにexact版と実観測を要求。 |

**直近のP0実行カード（2026-11-14、RA-0/1/2R同一仮統合HEAD再検証後）**：新規の型・Parser・Storeを増やさず、**取得済みの限定Go＋実PG検証と、未達の製品Operation→実Provider→Context／独立Consumerを切り分ける**ことを優先する。これは既存#2586/#1251の進行表であり、第二のScenario SSOT・Gate・Product Operationを追加しない。

| 段階と現在状態 | 実装担当・最小変更 | 独立した合格／停止条件 |
| --- | --- | --- |
| **RA-0：型付きQuery Source（Draft #4626、現/歴史Source回帰成立、E1未解決）** | `order_entry.recent_orders`の`input from std.datetime`と`Query.Inputs`。現行Golden/World Fingerprintを更新、歴史旧QueryはSHA-256固定しFrozen Worktreeでのみ再現。修正HEAD `9eecf2b9`の[Health #43558268996](https://github.com/hat47x/tei-platform/actions/runs/43558268996)・[Self-build #43558269076](https://github.com/hat47x/tei-platform/actions/runs/43558269076)は成功。 | **18 CI中17成功／1失敗**。残るE1 [#43558269132](https://github.com/hat47x/tei-platform/actions/runs/43558269132)のAPI exe`sqlgen`2 symbol混入は#1182別Owner。SourceのMeaning受理・main統合は未成立。Allowlist緩和を合格扱いにしない。 |
| **RA-1：Executor前型検査（Draft #4631、修復RA-0上でも限定Go成功）** | 実選択`Query.Inputs`と`time.Time`実引数を照合。不正・未宣言等のpre-Executor Call0と正常Call1候補。元[#4631](https://github.com/hat47x/tei-platform/pull/4631)の成功に加え、修復RA-0を基底にした[統合Draft #4644](https://github.com/hat47x/tei-platform/pull/4644) HEAD `b5c23044`の[RA-1 Go/旧Web CI #43558633790](https://github.com/hat47x/tei-platform/actions/runs/43558633790)が成功。 | 同一仮統合HEADのGo成功は、E1の依存境界、正式製品CLI、実Provider/Current Grant、Owner意味受理の成功ではない。元PRの統合順・採否はOwner判断。 |
| **RA-2R：参照PostgreSQL16（Draft #4638、修復RA-0上で7ケース非SKIP再成功）** | [#4644](https://github.com/hat47x/tei-platform/pull/4644) HEAD `b5c23044`の[実PG16 CI #43558633773](https://github.com/hat47x/tei-platform/actions/runs/43558633773)で空の2 DB・RLS Role・独立Readerを使い、**親＋7サブテスト全PASS、SKIP 0**。時刻境界、3/2/1/0行、NULL/列欠落、別DB・JOIN・RLS/権限を再実証。RA-1/Issue-to-Designを含む当該仮統合HEADの**3/3 Workflow成功**。 | 参照入口`tools/reference-browse-read`の取得証拠だけ。通常`tei` CLI→実Provider、Function Context、独立二Consumer、現在Grant、Owner受理は未証明。元RA-0→RA-1→RA-2 Draftを無条件mergeしない。 |
| **RA-2P：製品Operation→実Provider→Function Context（次の実装）** | #2586/#1251。既存`ExecuteBrowseQueryInEnvironment`・`BrowseQueryExecutorMap`で選択Query/Source/Env/Instance/Executorを接続し、**実Connection DB/Role・Consumed SQL/typed Params・Raw Rowset/Context**を同一Attemptで独立観測。並走Draft [#4647](https://github.com/hat47x/tei-platform/pull/4647)ではRLS/Limit/関係差を観測しContextへ`partial`を運ぶW1〜W3実装候補があり、同HEAD **7/7 CI成功**。 | Provider自己申告A→実接続B、選択A→実B、別SQL/Source、旧Contextの負例を分ける。実行前はCall0、接続後の不一致は成功Context0。W1〜W3の成功や参照実PGを通常Product`tei`の実行・二Consumer受領へ昇格しない。 |
| **RA-3：通常CLIと独立Consumer** | #1251/#2586/#1824。既存Typed Application Operationを通常`tei` CLIと独立Native/Studio受領へ結線し、同じSource/Query/Context pinでTableとAggregate等の異質Consumerがそれぞれ受け取った値を確認。 | 各Consumerの`selected/invoked/received`を別々に実観測。JSON/stdoutの複写や固定UI Screenshotだけでは二Consumer合格にしない。Web共有Serverは[#4627](https://github.com/hat47x/tei-platform/pull/4627)の実Bridge exposure（command・外部Process含む）をそのまま公開せず、read-only facadeと現在Actor/Workspace Grantの実証を別に要求。 |

**直近の検証で分かったこと**：RA-0の現行World・Goldenと歴史用Sourceを分ける修正は[#4626](https://github.com/hat47x/tei-platform/pull/4626) HEAD `9eecf2b9`でHealth・歴史セルフビルドとも成功した。旧HEADで失敗していた未使用importやFrozen Readerの互換問題を、現在の未修正課題として数えない。ただし同HEADのE1検証[#43558269132](https://github.com/hat47x/tei-platform/actions/runs/43558269132)は、実APIバイナリに`tei/internal/database/sqlgen`のシンボルが2件含まれるため失敗している。これは#1182で依存の入り口を調べる。許可リストを広げて通過扱いにしない。

修復済みRA-0を土台にした[#4644](https://github.com/hat47x/tei-platform/pull/4644) HEAD `b5c23044`では、RA-1のGo・過去Web回帰とRA-2Rの実PG16・7ケース、Issue-to-Designの**3系統すべてが成功**した。基底PRのマージ、製品入口、現在の権限、業務上の意味の受理まで済んだわけではない。失敗・成功の履歴は試行とコミットごとに残し、異なるHEADの証拠を一回の製品実行にまとめない。

**段階別Contributionと共通意味を同一視しない（Plugin接合）**：[#4646](https://github.com/hat47x/tei-platform/pull/4646) HEAD `77463edb`は、共通のMeaning ID/Version/FingerprintとStudio・Generator・Runtime・Testの各段階Contractを区別し、既存契約Resolverで対応付ける**Reference試験**として3/3 CI成功。段階ごとのPackage版が異なる正例と、Meaning/Target/Scope/Provider/Artifactの不整合・曖昧選択・順序の負例を検査する。これは**実Pluginロード/配布、実Go Builder利用、DB Provider/Current Grant、Semantic Meaning受理ではない**。Pluginを本体とStudioに一体導入する判断は#951/#2639/#4632のOwnerへ戻し、将来のProvider/Target/Artifactの資格を「同じ意味Version」だけで承認しない。第2の意味正本・Resolver・Hostは作らず、Go単一バイナリの配布要件とStudioセルフビルド優先を維持する。

**Function起動とDB読取りの二経路を一つに潰さない（他チャット接合）**：L-Pは[Script Launch #4495](https://github.com/hat47x/tei-platform/issues/4495)が公開Entry→Typed Invocation→選択Capability/Provider→実呼出/Resultを所有し、L-Rは[#2586](https://github.com/hat47x/tei-platform/issues/2586)がBrowse Pattern→Query Inputs/Provider→実Rowset→Function Data Contextを所有する。現在のPG Read #4539 はL-Rの限定実行であり、公開Business FunctionのEntryを実行した証拠ではない。製品計画の**一機能**への受入では、各経路の`declared/selected/actual invoked/actual result/Consumer received`を同一の受理済みFunction/Workspace/Sourceに必要な範囲で結び、別Attemptの成功を新しい合成Runへ昇格しない。L-RのRead成功に業務Effectを足さず、L-Pの契約拒否成功にProvider選択・GrantやRaw Rowsetの証拠を足さない。ProviderとHostの認可・Result/Effectは既存Ownerに残し、共通Invoker/Runner・第二Authorityを追加しない。

### 異なる入力経路に共通する「選択→実受領」の証拠連鎖（2026-11-14更新）

本節は**既存の各Ownerの証拠を比較するための作業票**であり、全Target共通の新Core Schema、統一Invoker、第二の許可/Store/Runnerを設ける提案ではない。Source/Workspaceの同一性、契約・利用者の選択、実際に使われたbyte・実行者、受信者・開示範囲、Ownerによる受理は異なる判断である。**同じdigest、同じ表示値、同じ時刻、同じPRのgreen CIだけでは、その間の接合を実証したことにならない**。Source A→B→A、同名異Root、Actor/Grant失効、旧Responseの後着を個別の負例として残す。

| 経路／現HEADの局所証拠 | 最初に未成立な接合・実際に拒否すべき事例 | Ownerと受入順（既存Issueを使う） |
| --- | --- | --- |
| **L-R：Query→実DB**。[#4644](https://github.com/hat47x/tei-platform/pull/4644) `b5c23044`は修復RA-0＋RA-1 Go＋RA-2R実PG16（非SKIP7ケース）を**同一仮統合HEADで3/3 CI成功**。独立Draft #4626の17/18失敗1件はE1の`sqlgen`依存。 | **RA-2P通常製品Operationから実Provider/Consumed SQL/Contextを同一Attempt受領**、選択A／実Bを拒否。W1〜W3 [#4647](https://github.com/hat47x/tei-platform/pull/4647)は観測制限`partial`であり、native SQL全件性の証明ではない。 | #2586/#1180/#2514。RA-2Rは参照取得済み、RA-2PとRA-3二Consumerを次に行い、D1.2再集計は観測Scopeの上限を継承。 |
| **L-P：Function→Go Adapter**。[#4629](https://github.com/hat47x/tei-platform/pull/4629)はmain統合済みのResult契約不正→Adapter call0。[#4636](https://github.com/hat47x/tei-platform/pull/4636) `da807abc`はInput/Effect宣言不正・nil Contextのcall0を**exact HEAD 9/9 CI成功で確認しmainに統合済み**。 | 実入力値が型/意味に合うか、公開Entryが本当に選択されたか、Host現在Grant・選択Providerが実際に呼ばれたかを別に確認。宣言`Effect`≠Commit Receipt、契約拒否成功≠業務完了。 | #4495/#1174/#3422、既存Script Launch §17とその独立Expected。RA-2実Readの成功票へ加算しない。 |
| **R3：REA Evidence→Host→受信者**。[#4637](https://github.com/hat47x/tei-platform/pull/4637)旧HEAD `f38117dc`は**12/12 CI成功**。同一Raw EvidenceをRoot A/Bから独立に読み、Bの改変はProducer Receiptの不整合で拒否する回帰を修正した新HEAD `9b8f5ca9`は**12/12 Workflow完了・成功**（旧HEADの12/12とは別Attemptで検収）。Host固定Selector/Root、4MiB制限と子Process試験に限定。 | Raw Digestが一致してもProducer Deliveryの資格にはならず、A→B→A・Root差替えを現在Source/Grantの承認へ昇格させない。製品Studio/HTTP/MCPでの配送とAudience別Disclosureは未検収。 | #3394/#4503/#2516/#3419。旧HEADの12/12を新HEADへ移植せず、R3-C独立Consumer、#4627 Web公開は別Owner。 |
| **BuildIntent→実Builder→Desktop**。[#4634](https://github.com/hat47x/tei-platform/pull/4634)限定Builder pin 8/8 CI成功。[#4587](https://github.com/hat47x/tei-platform/pull/4587) HEAD `9b58533d`のWindows生成版・通常版WebView2を含む**12/12 CI成功**。[#4648](https://github.com/hat47x/tei-platform/pull/4648) `64fb0f9a`では同一PlanでGo 1.26.11／1.27.0の実コンパイラが**異なるArtifact byte**を生成したことを実測、Toolchainの申告とEntry側の再読取を追加し**8/8 CI成功**。 | `toolchainObservation=matched`は申告されたpath/digestの一致にすぎず、**実Compilerの選択証明でもPlanに含む入力でもない**（`namedByPlan=false`、`qualified=false`）。Toolchain pinの採用は#2831 Owner未決。Stage1→Stage2 WebView2/BOM/Seed Trustは別P0。 | #2831/#4583/#1119。別Compiler・同じPlan→別Artifact byteを再利用・Owner受理へ昇格しない。 |
| **Wails Server Bridge→公開Web**。[#4627](https://github.com/hat47x/tei-platform/pull/4627) `b9cf5447`はService登録・実Go exported 15 method棚卸し、**5/5 CI成功**のDraft。 | 一覧に`ExecuteCommand*`とSHO子Process入口がある。**静的InventoryのPASSは公開AuthN/AuthZでも安全なWeb配送でもない**。不正Actor/Workspaceからの実HTTP/WS拒否、二受信者間の秘密非開示、read-only façadeを別に確かめる。Browser提供の`allowed []string`をHost Grantと認めない。 | #1824/#3421/#2516。local loopback Server PoCをmulti-user公開へ自動昇格しない。 |

**RA-0の未解決作業票（旧Healthの修復待ちを完了に修正）**：現`input from std.datetime`と現行Golden/Fingerprint、旧QueryのFrozen Worktree内SHA固定復元は[#4626](https://github.com/hat47x/tei-platform/pull/4626) `9eecf2b9`で実装済み、**同HEADのHealthと歴史Self-buildもPASS**。未使用`reflect` importも削除済み。これらを再度「未実装」としない。残る実CIは**E1 Browser→JSON API binaryの`sqlgen`2 symbol混入**のみであり、#1182が実バイナリの依存閉包を調べる。Allowlist緩和・SKIPは不可。元RA-0/1/2の統合順・Meaning受理はOwnerが判断する。

**次の実装順**：① RA-0修復済みSource＋RA-1／RA-2Rの**同一仮統合HEAD #4644 3/3成功**は局所証拠として保持、② #1182でE1の実API依存境界を独立修復、③ #2586 RA-2Pで通常`tei`入口→実選択Provider→実Connection・SQL・Raw→Contextの同一Attemptを接続、④ RA-3の独立二Consumer受領。**E1の別Owner障害解消を理由にRA-2Pの設計・実装準備を停止する必要はなく**、main統合時には各Ownerが未解消CIを別資格で明示する。Desktop Stage1→Stage2の独立P0を遅延させない。Script宣言preflight #4636と計画#4642/#4645に加え、**型付きQuery・取得制約・実Build入力を切り分けるScript §19の[#4649](https://github.com/hat47x/tei-platform/pull/4649)もmain統合済み**。ただし同§はdocs-onlyであり、公開Function実行の資格を与えない。

**Function Data Contextの被覆・欠測・否定判断（TASK 9 W1〜W4、2026-11-14）**：[#4647](https://github.com/hat47x/tei-platform/pull/4647) HEAD `a962a3f6`は**7/7 GitHub CI成功**。別途、WSL2/Linuxの使い捨てPG16で手書きExpected・独立ReaderによるW1〜W3の非SKIP実行がPRに記録されている。ただし当HEADのGitHub「実PostgreSQL未実行スイート」成功は、新規のDSN付き統合試験の非SKIP実行を意味しない。
- **W1（実観測）**：Readと同じDB Transactionで宣言関係を`ACCESS SHARE` Lockし、実引数付き`EXPLAIN (VERBOSE)`、`relkind`、実Roleの`row_security_active`を取得。カタログの`relrowsecurity`やPlanだけで現在のRLS状態を決めない。Observation失敗はSavepointで隔離し、Read本体のエラーを隠さない。Lock以前のSnapshotと並行DDLは過剰な`partial`を生みうる。
- **W2（宣言関係）**：Persistence MappingからDataの宣言済み関係／主キーだけを解く。SQL本文・同名列から意味を推定しない。0件／複数候補は解決不能として保持する。
- **W3（被覆）**：RLS、LIMIT、宣言外JOIN、View/Partition/Foreign/Function scanなど**観測した制限のみ**`coverage=partial`と理由へ写す。制限が見えない場合でも`coverage=unestablished`、`open`には常に`native-sql-predicate-not-closed`。`complete`と`EmptyJustified=true`は出さない。STABLE SQL関数で基表3行を結果2行へ絞るのにPlan上は制限が見えない反例を保持する。
- **W4（列の由来、未実装）**：結果列の表OID／列番号が未観測のため、NULLを`stored`、粒度を`entity`と断定しない。`nullOrigin=unestablished`と`declaredKey`の一意性を分離し、実ProviderのFieldDescription/pgx等への小さな接続が必要なら既存Ownerで検証する。
- **D1.2／問いの閉包**：`partial`は「欠けた全行数の確定」ではなく、制限が少なくとも一つ観測されたという事実。0件から「存在しない」、RLSで縮んだRowsetから全件総計、重複JOINから元Dataの粒度を導出しない。Table/Aggregate等のConsumerへ`Observed Slice`と`unestablished/partial`を改変せず渡す。否定Claim・全母集団・必要集合の完了は#3262/Ownerが別に受理する。**W1〜W3をRA-2P/RA-3の新Hard Gateにしない**。

**RA-2RとRA-2Pの継承禁止（2026-11-14、実PG受入票）**：[#4638の実CI](https://github.com/hat47x/tei-platform/actions/runs/43553849861) は、空DB 2個の作成、`CONFIRM_ISOLATED`、RLS Reader DSN、親＋A/B/C/D/E/G/FのGo JSON eventの**fail/skipゼロ**をJobが検査したもの。これは実Reference Executor/DBの証拠であり、**正式Product `tei` CLIからのselected→invoked→receivedの証拠ではない**。既存`TestBrowseReadAgainstDisposablePostgreSQL`の7ケースを再作成する代わりに、次のRA-2Pへ既存の独立Expectedを参照して接続する。新たに必要な反例は選択A→実B、選択したQuery/SQL byteと実消費byteの相違、Context ID/epoch/Scopeの取り違え、Root・Source A→B→A、他ActorでのRead claimの不当な再利用。Null/Rowset/Maskingの値が参照Runと同じでも現在Grantを創出しない。Source/Provider/Target別の受入票は各Ownerに戻す。

**RA-2Pの最小接続点（既存コード確認に基づく）**：[RA-1の`internal/driver/browse_query_environment.go`](https://github.com/hat47x/tei-platform/pull/4631)には既に`ExecuteBrowseQueryInEnvironment`、`BrowseQueryExecutorMap`（`Environment+Database instance`をKeyとする）、任意実装の`DatabaseProviderBoundBrowseQueryExecutor`、`BrowseQueryEnvironmentExecution`がある。**最初の製品PRではこれらを再利用し、別の汎用Executor Registryを作らない**。ただし現`DatabaseProvider()`はExecutorの**自己申告**であり、`BrowseQueryEnvironmentExecution.Instance`も宣言上の束縛、`QuerySourceSHA256`はExecutorに渡したQuery Source byteのDigestである。いずれも**接続先PG16の実体確認ではない**。実際に開かれたConnectionのDatabase/Role・ReadOnly/権限、実SQLと引数、Raw Rowsetを、Product CLIの当該Attemptから独立観測して初めてRA-2Pを判定する。
- **P-正例**：型付き`from:time.Time`→実選択Env/Instance→該当KeyのExecutor→実PG DB/Role→Source SQL byte→Raw 3/2/0/境界→Mapped/Contextへ受け渡す。参照PGの手書きExpectedを読み直すが、**参照CLIの成功JSONを流用しない**。
- **P-負例**：宣言されたInstance Aに対してRegistryがBのExecutorを返す／Runtime Providerが異なる／Aと名乗るExecutorが実際には別DBへ接続する／Source SHAだけ一致してWorkspace/Revisionが古い、を区別。拒否すべき入力は**実DB Call 0**、接続後に初めて判別できる誤接続は**不適格Result（成功Contextを出さない）**として扱い、無条件に「呼出し0」とは表現しない。
- **後続へ渡す最低限の観測**：`selectedQuery/QuerySourceSHA256/SourceRevision`、`selectedEnv/Instance/Executor`、`actual DB/Role/SQL/typed Params`、`Raw/Mapping/Context epoch+scope/coverage`、`attempt/receiver`のうち、現インターフェースが保持しない値は**未観測をunknown**にする。Carrierの申告値、実OS/DB観測値、Consumer実受領票を一項目へ混ぜない。これは既存#2586/#2514の受入手順であり新たな公開APIや永続Schemaではない。

**RA-2Pの入口契約を閉じる最小実装票（生成Functionとの非同値）**：[Reference生成CLI #4559](https://github.com/hat47x/tei-platform/pull/4559)は、旧SourceのSQL `:from`を見つけても`inputDeclaration=undeclared`として残し、生成CLIは`--from`を**文字列**で受ける。ローカルPGで3/2/0行等の受入結果はあるが、型付き`Query.Inputs`や通常`tei`製品Operationの実証ではない。main統合済み[Script Launch §19 #4649](https://github.com/hat47x/tei-platform/pull/4649)もこの差を明記している。
1. **入口の宣言を一致させる**：まずRA-0現Sourceの`from:std.datetime`とSQLのPlaceholder集合を**既存Parser/Resolverの能力範囲で**照合し、未宣言・余分・重複を成功させない。Placeholder名から型を推定しない。SQL方言/静的解析で完全閉包を証明できない箇所は`unknown`を保持し、公開資格を得たと主張しない。
2. **CLIの文字列→Typed Operation境界**：CLI引数の受信を`time.Time`型適合とせず、型変換を行う入口・認める日時表現・時差と精度の扱いを既存#3422/#2586 Ownerが定める。変換後の`time.Time`だけをRA-1へ渡し、missing／空／明示NULL／不正形式／未知名／重複指定は**実Provider Call0**で棄却する。暗黙の時刻補正や`NULL`の代入をしない。
3. **Runtime・Context・Receiverを別照合**：選んだEnvironment/Instance/Providerと実接続DB/Role・Consumed SQL/typed Parametersを一致させた後、Raw Rowset→Function Contextの`scope/coverage/open/nullOrigin`を不必要に格上げせず渡す。**現在Grant/Audienceを評価した上で**二つの独立Consumerが実際に受領するかをRA-3で確かめる。RLS/関係名/非公開行の存在を示す内部Observationや`partial`の詳細理由は、B2C受信者へ無条件に開示しない。
4. **既存Scenarioでの判別**：旧Source`undeclared`なのに生成CLIが動く、受信文字列をそのままExecutorへ渡す、選択A→実接続B、RLSで0行なのに「注文なし」と表示、会員AのContextを会員Bへ再送、同byte A→B→AのRevision/Grant流用を異なる負例として検査する。ReferenceのJSON転記、UI Screenshot、静的生成Manifestは正常Product Call/独立Delivery Receiptの代わりにならない。

**横断受入設計との役割分担**：[main統合済み#4653](https://github.com/hat47x/tei-platform/pull/4653)は、定義から利用者の受入に至る全工程とCL-01〜10の負例をまとめた正本である。本節はそのうち、型付きQueryを通常`tei`の実Providerへ接続するための作業手順を補う。新たな受入台帳や意味定義を作らない。

### DSL診断から製品実行までの引渡し（#4652と並行する実装）

[設計#4652](https://github.com/hat47x/tei-platform/pull/4652)はmain統合済みで、診断と拒否を四つの段階に分けている。

| 段階 | 先に確かめること | 現状と受入の条件 |
| --- | --- | --- |
| A：Source解析 | 文法、識別子、元のSource位置 | 未知のEffectなど、構文上判定できる誤りは正しい位置で知らせる。Runtimeのエラーへ後から架空のSource位置を付けない。 |
| B：Workspaceの意味検査 | 型・Input名の重複、OutputとOutcomeの矛盾、参照先 | 既存Validatorで診断する。実装Draft[#4655](https://github.com/hat47x/tei-platform/pull/4655)はGoテストを追加したが、HEAD `3261d172`のHealthは整形不一致で失敗。mainの索引欠落は[#4659](https://github.com/hat47x/tei-platform/pull/4659)で修復済み。両者を混同しない。 |
| C：実行直前 | 不正宣言・不適合引数のとき、選択したAdapter／Executorを呼ばない | Capabilityの限定Input/Effect拒否は#4629/#4636でmainへ統合済み。Source→Adapterを同じケースで試す[#4654](https://github.com/hat47x/tei-platform/pull/4654)はGo試験の整形不一致でHealth未合格。RA-1の日時入力検査とは別経路。 |
| D：製品入口と結果の受領 | 公開Entry、実Provider、Context、現在Grant、Consumer | **未受入**。A〜Cの試験が通っても、Functionが公開され、正しいProviderから結果が届いたことにはならない。 |

A〜Dは新たな検査Engineの階層ではなく、既存Parser・Validator・Invoker・Hostの担当分担である。Studioでは診断一覧から該当Sourceへ戻れることを最初に確かめ、Source位置を正確に保持できない場合は誤ったジャンプ先を表示しない。新しい文法・Plugin採用や生成AIの常時接続を、Desktopセルフビルドの前提条件にしない。

### 読取りとSchema変更を切り離す

[main統合済みの生成アプリ修正 #4662](https://github.com/hat47x/tei-platform/pull/4662)では、生成された`query`／`command`／`validate`が、起動時にMigrationを暗黙適用する既存経路が見つかった。**照会を実行しただけでDBを初期化・更新しない**ことを、製品入口の安全条件とする。

`query`は既存DBを読取り専用で開き、`validate`はDBなしで構造を検査できるようにする。変更が必要な`command`もMigrationの暗黙実行とは分離し、DB作成・Schema変更は別の明示Operationと権限・Receiptに委ねる。存在しないSQLite DB、Memory DSN、別Schema、Migration履歴不整合を「暗黙に作り直して成功」としてはいけない。特に**通常`tei`のPostgreSQL実Read（RA-2P）と生成SQLiteアプリの修正（RB0）は別Target**であり、一方の試験成功をもう一方へ適用しない。

#4662は[統合PR #4683](https://github.com/hat47x/tei-platform/pull/4683)を通じてmainへ反映済み。修正元HEAD `095c5a11`は [Repository Health #43576908564](https://github.com/hat47x/tei-platform/actions/runs/43576908564)・Cache・既知脆弱性確認の**3/3 Workflow成功**を確認済みで、正規のGo vet／Go testを含む。ただし本番Schema・History・外部Provider・明示Migrationの適格性は証明されていない。Source Revisionをファイル名や仮のDigestで代用する問題も、[#2600](https://github.com/hat47x/tei-platform/issues/2600)の所有範囲として残す。#4234/#4125のDB移行計画と#2586のProduct Read計画を、同じ手順で安全性を評価できるよう接続する。

**判別の最短ループ（次の実装単位を固定）**：RA-1の**限定Go/歴史Web回帰の同一HEAD成功は取得済み**。修復済みRA-0のHealth/Historyと、#4644仮統合HEADのRA-1 Go・RA-2R実PG7ケース**3/3 CI成功は取得済み**。現在の別Owner障害はE1の`sqlgen`依存#1182のみであり、D1製品として次に不足するのはRA-2Pの正式CLI→実接続→Contextへの実呼出。続く**RA-2P**は、`実選択Query/Inputs→型付きactual Parameters→Environment内の実選択Provider→実Consumed SQL byte/実Rowset→Context Result Slice`を**通常製品Operationの同一Attempt**で観測する。選択A→実Call Aを正例、**選択A→実Call B**・Source/SQL別版を負例とし、SQL Placeholder全閉包・Snapshot/Current Grant・Coverageは実証範囲を超えて昇格させない。次にRA-3通常`tei`と独立Consumerへ進める。**RA-0/RA-1局所PASSをD1製品合格と表示しない**。D1.2 Measure、D2編集、D3 Effect、REA/SHO/SUI、Web共同編集は既存Ownerが並行するが、この最初の実ReadにHard Dependencyを増やさない。RA-1〜3・E1修正・実Web Grantの新試験は本計画変更では**NOT RUN**。

**停止線**：`Source Revision`・`Context epoch`・`Identity/Slot`・`Quantity/Time/CRS`・`Required Scope`・`current Grant`のうち当該操作に必要なものが未確認なら、read説明は可能でもEdit/Executeを許可しない。実行証拠はWorld/Model/Artifact revision、実選択Provider/Generator、実消費byte、独立Expected、Owner受理を分離し、既存[#2514](https://github.com/hat47x/tei-platform/issues/2514)/[#1180](https://github.com/hat47x/tei-platform/issues/1180)/[#1119](https://github.com/hat47x/tei-platform/issues/1119)のGateに載せる。新たな公開DSL・Core TypeKind・固定15表現Enumは異種Consumerの反例を得てからADRで判断する。


### 外部候補・三要素・実Readの受領を結ぶ判別条件（2026-11-14追加）

この表は新しいScenarioや型Schemaの正本ではない。D0〜D3で必要な既存Fixture（#4350 DF-X01〜08、#2497 XMAP、#2500 VE、#4472 TE-CR、#3394 REA、#2514/#1180 独立Expected）を**同じデータ様式契約へ接続する順序と、各Ownerの停止条件**だけを示す。REA/SHO/SUIや三要素の実装完了を、実Provider→通常`tei`/StudioのP0 Hard Dependencyにしない。

| 接合点 | 共通データ様式の検査 | 必ず判別する正負・Owner |
| --- | --- | --- |
| **存在と探索結果** | `Subject existence`、`Reference count`、`Read coverage`、`Disclosure`を**別事実**とする。存在しないSubject・存在するが参照0・未列挙のSource・非開示・部分読取を空配列/成功一語に畳まない。 | #4584は「不存在Subjectの成功扱い」をmainで修正済みだが、外部REA→独立Consumerの同一性は未検収（#3394/#4603）。同Label別Subjectと、取得済み0件/未取得を既存Expectedで分離。 |
| **GraphとREA観測** | 外部EvidenceのNode/Edge ID・有向・並行辺・根拠Source・coverageを保った候補Projectionを行う。REA観測→TEI正式Entity/Relation・三要素のData/Function/Businessへの昇格はOwnerの選択と意味検査が必要。 | Node label一致だけで同一化、並行/逆向きEdgeの消滅、未取得Sourceから関係不存在を断定、外部CandidateだけでDSL/CRUDを書き換えるものを拒否（#3394/#1124/#2497、DF-X03/04）。 |
| **三要素CRUDの再投影** | `Data identity`と`Business/Function identity`、明示CRUD定義と推導Operation、矛盾・未掲載を別に扱う。図/表/相互参照の表示成功を業務意味の一致に昇格しない。 | 異なる索引の同名、CRUDDefinitionsのみのpairの脱落、同Pair矛盾の後勝ち、`OK:true`と未解決の矛盾共存を検査。実装Owner #4472、Source編集Owner #1124、DF-X04の再投影へ渡す。 |
| **実DBの行とモデルSubject** | Source Pin（論理Subject/Source位置）と、Provider Result Sliceの行Identity/Version・grain・scopeを分ける。RLS/NULL/empty/partial、JOIN fanout、集計をContract相対のRead claimとして照合。 | 限定PG実ReadのRowsetを同一Function/AttemptでTableとAggregate Consumerへ渡す。**別の既存合成Fixture**の`2行×3行=6 JOIN行`についてはEntity6件への誤昇格を独立に拒否する。独立Readerと二Consumerの実受領を証明するまでは未合格（#2586/#2506、DF-X01/02）。 |
| **編集と実行** | Studioの`Source span Patch→Preview/Review/Accept→再Projection`と、実DBの`Pending Change→Current Grant/CAS→Commit/Receipt→独立Read`を二経路に分ける。Selection PinだけではWriter/DB権限が生じない。 | 旧epoch/Source A→B→A、Row version競合、ACK-loss、非可逆Chart、未許可AI提案を正負比較。失敗または不確定時に自動再書込しない（#1124/#2500/#1174、DF-X05/VE-05）。 |
| **文書・多Consumer・開示** | SUI Card/Island/Canvas→SHO Markdownの非単射性と、Raw byte/改行/BOM/Parser版・Source occurrenceを保持。表示から消えた情報や非公開Evidenceの存在をConsumerへ漏らさない。 | 同じMarkdown byteでも異なる元Graphを識別できない。Source byte一致とRendered一致を別採点。内部Path/Locator、hidden count、診断/DOM/Tool応答への二次開示を既存Disclosure Ownerが判定（#2497/#2516/#3394）。 |

**D0→D1の最初の同一Attempt Witness**（既存Evidenceを結合する比較票であり新公開Schemaではない）：`Workspace+Source/Model Revision`、`実選択Query/Provider/実読byte`、`Context ID+epoch+grain+scope+coverage`、`Consumer A/Bの実受領結果と版`、`各ConsumerのObserver/必要閉包/未知`、`独立ExpectedとCurrent Disclosure/Grantの観測有無`を、同じ試行IDに照合する。各列が揃っていても**同一原子的Snapshot**やOwnerのMeaning Acceptanceを自動主張しない。Consumer Aだけ取得できた場合はAの限定Claimまでとし、Bの成功を作らない。

**順序と非目標**：まずD1の実PG Rowset→Table/集計の二Consumer受領を製品/実Consumer経路で証明する。並行してD0のGraph→Tree Node Slot編集と三要素CRUDの矛盾負例を既存Ownerが検証する。外部REAのR3（CLI＋第二Consumer）は#3394/[Draft #4603](https://github.com/hat47x/tei-platform/pull/4603)が所有し、証拠を再利用するがD1の先行条件にはしない。生成Wailsのread-only WebView2成功 [Draft #4579](https://github.com/hat47x/tei-platform/pull/4579)は固定Fixture/WindowsでのConsumer能力の証拠であり、上記の実DB Context受領・現在Grant・セルフビルドL3を証明しない。第二Resolver/Graph Engine/CRUD正本/Consumer Adapter/公開Gate・Core型・実行Hostは増やさない。


### 問い相対の証拠到達・異種Join・否定判断をD0〜D3へ適用（2026-11-14追加）

横断レーンのREA R3設計 [Draft #4603](https://github.com/hat47x/tei-platform/pull/4603) の **P/J/Q（局所観測／異種Evidence Join／問い相対の必要閉包）** は、標準データ構造やCore型の増設ではなく、既存#4350 DF-X・#2586 HJ・#2497 MP・#4487 TEA-Q41/Q43・#2514/#1180の独立Expectedで**主張の到達範囲を点検する方法**として取り込む。#4603のMASTER_PLAN追補を複製せず、本計画では通常TEI Functionと二Consumerで検証すべき差分だけを所有する。

1. **P：局所観測を壊さない。** 原Sourceの`Required / managed / actually loaded / parsed / selected Function+Query / provider actually invoked / rowset actually observed / consumer actually received`を、既存Reader/Operation/Evidenceから**別々の事実**として照合する。この列は一律の直列APIでも新しい状態Enumでもない。構成上Selected Aでも実Invoke B、SourceをLoadしても実Callなし、Raw digest一致でもParser/Role/Plugin pin更新、Consumerに配送せずCLIだけ成功を、合格へ縮約しない。各段で`UNBOUND_DATASOURCE`、未取得、empty、partial、Provider Error、非開示、Current Source/Grant未確認を該当する観測根拠と結ぶ（#2586/#1251/#2506）。
2. **J：異種Evidenceの結合を無断で行わない。** REA External Node/Edge/Observation、三要素Data/Function/Business、TEI受理済みMeaning、Provider Rowset、旧システムのActual、TEI TargetのActualはProducer・Subject/occurrence・granularity・有効/観測時刻・Source版が異なる。同名、同IDの別Namespace、同じJSON/Source digest、同時刻、同じUI表示では結ばない。Qualified MappingとOwnerの受理範囲を明示し、未受理なら並列候補とunknownを残す（#3394/#1124/#4472/#2497）。
3. **Q：否定・全称の主張は必要閉包を検査する。** 「参照0件」や「三要素図に表示されない」から、未使用・存在しない・安全に削除できる、と結論しない。各Questionに必要なSource/Relation family/動的呼出し/未ロードPlugin/手作業/未取得page/必要権限を既存Required Setで点検。十分でなければ`observed=0`と`absence unknown`を両立して表示する。非公開Source有無で公開件数・diagnostic・DOM/ARIA・Tool JSONが変わる二世界の反例は既存#2516へ渡す。R2のGraphCoverageをData/Function/Business全分野のCompletenessへ一般化しない。

**既存受入に追加する独立の判別（新Fixture正本は作らない）**

| 既存ケース・Owner | 受理側が必ず区別する二つのWorld |
| --- | --- |
| TEA-Q41／#4487・DF-X03/04／#4350 | REAがData候補+Edgeを届けGraph配送completeでも、未観測Function/Businessは`unsupported/unknown`。図の0件と実不存在を別主張にする。 |
| TEA-Q43／#4487・DF-X01/02／#2586 | 旧SourceのFunction宣言と現PG `order_entry.recent_orders`のRead成功だけでは同一Functionの旧実行・移行一致を主張しない。Legacy Actual (L)、Owner Accepted Desired (D)、Target Observed Actual (T)を別観測で比較する。 |
| TE-CR-01〜04／#4472・DF-X04／#4350 | Function×Dataに現れた宣言occurrence数、CRUD表のセル数、実業務更新回数を分ける。Index取り違え・明示のみpairの消失・矛盾後勝ち・`OK:true`への過剰昇格を検出。 |
| HJ/DF-X01／#2586・#2506 | 実Readに成功しても`selected≠invoked`、Consumer Aのみ受領、B未受領、unknown/partialの隠蔽を非合格にする。2×3 JOINの合成反例は現PGの実Read事実と別証拠で扱う。 |
| DF-X05／#1124・#1174・#2516 | Same byte/Source A→B→A/Grant変更・旧AI Replyを組み合わせても編集/再表示資格を自動復帰させない。低権限Consumerにprivate Evidenceの存在・パス/件数を推測させない。 |

**D1への最小引渡し**：まず`order_entry.recent_orders`について同じ固定Workspace・Function・Provider選択・SQL引数・Source Revisionで、`selected → actual invoke → actual rowset → Context instance → Consumer A/B selected/invoked/received`の**最初の欠落辺**を一つだけ確定する。既存#4539の実PG16 Readと、別Headで成立したWebView2 Browser/InspectorやREA Consumerの成功を合算しない。同一Attemptと同一Contextでも全Source原子的Snapshotは推定せず、各Consumerの独立Expectedと現在の必要Scope/Disclosureを照合する。現時点でこの縦断の実行・Owner受理は **NOT RUN**。

**D2/D3への停止線**：P/J/QはRead主張の十分性判定であり、書込みGrantやOperation Intentの発行元ではない。設計Sourceは#1124のAccept→再Projection、業務Actionは#2500/#1174のGrant→Effect→Receipt→独立Read、外部Hostは独立Consumerとして検証する。静的ObservationやAIが示したSource spanから、DB Schema更新・業務Effect・公開権限へ自動昇格しない。Plugin/Build時の`declared/selected/actually invoked/observed/enforced`も既存#2018/#3278/#2831のEvidenceに接続し、CLI/Browserの限定PASSをStage1→Stage2自己再構築成功と呼ばない。


### Desktop独立編集／Web共同編集／AI候補に共通なデータ様式の保存・再資格化（2026-11-14追補）

[Studio #1824](https://github.com/hat47x/tei-platform/issues/1824)の Desktop＝ローカルWorkspace/ローカルGitによる独立編集と、将来のWeb＝共有サーバWorkspaceでのチーム同時編集は、**保存先・同期・Session・認可の違い**である。両者とCLI/AI/SHOは既存Application Operation／Qualified Subject・Source/Model Context／Preview・Acceptance／Function Data Context／Build Evidenceを共通使用する（#1124/#1251/#2621）。これはWeb共同編集機構の実装済みClaimではない。Wails Desktopの実WebView2 read-only成功 #4579をWebでの共同編集・Server HA・同時保存合格へ転用しない。

**6つの現在性・同一性を分ける**（形式/API名の採用ではなく既存Evidenceの比較観点）：(1) editorのunsaved **Draft/Buffer Revision**、(2) 管理Root中の**Source byte/occurrence/Revision**、(3) Owner受理済み**Model/Meaning Revision**、(4) UIの**Workspace/Branch/Session・Selection epoch/Context ID**、(5) 共有用**Git Commit/Tree/merge base**、(6) 生成器と入力閉包に基づく**World Fingerprint/Artifact Revision**。Sourceのbyteが一致してもA→B→A、別Workspace、別Parser/Role/Provider、Grant変更なら旧Meaning/Review/実行資格は戻らない。逆に同じ意味Subjectへの別表示位置・Markdown装飾変更だけでは意味変更を推定しない。

**Web共同編集の三つの競合レベル**を分離する（#1824が体験/Host、#2621がRaw Authoring、#1124が意味編集、#1174/#2598が実Effectを所有）。
- **L1 Buffer/Text**：同時文字編集・同期は草稿の問題。CRDT/Yjs等は候補であり、採用済みの共通Source Writerではない。未解決ConflictでもDraftは失わず別保存・Review可能とする。MarkdownのCRLF/BOM/未知Roleや未対応`*.tei.md`を自動正規化・登録しない。
- **L2 Typed/Semantic Change**：Stable Subject+Source occurrence/Slot+base Source/Model Revision+必要依存の現在性から局所Patch候補を導出。異なるSlotなら自動許可ではなく依存と受理範囲を確認、同一Slotの競合・Graph edge identity欠落・CRUD矛盾・旧AI提案は明示Review/保留。CRDTの収束やGitの自動MergeはMeaning Acceptanceでない。
- **L3 Git/Workspace/Build**：Desktopのcommit/rebaseとWebの共有Workspace確定版は別の操作。Merge後は実Sourceから再Index/再Validate/再Buildを必要範囲で再資格化し、Read-only Preview・Builder選択・実Builder起動・生成Artifact・Desktop実Consumerを別Evidenceへ記録。#4623の限定Plugin Builder選択→Child起動前digest一致はDraft・CI限定の成果で、非Plugin Wails DesktopのBuildIntent→Stage1→Stage2を証明しない（#2831）。

**原本と表示を分ける**：通常Markdownは原文を保持して保存できる一方、TEI意味定義・宣言付きTable/Graph・生成物はAuthoring/Meaning/Read-onlyを区別し、`*.tei.md`へのParser/Role/Loader正式採用前に自動意味昇格しない（#2621）。同じCanonical SubjectでもTable、Graph、SHO Document、SUI Canvas上のView occurrence/座標は別。表示・Selection・AI pinからSourceのWriter・DBのAction・Owner Authorityを発行しない。外部Host/TEI Studioの実Consumerへ開示するContextはAudience/用途相対の許可済みSliceから作り、秘密の存在を公開件数/unknown/診断/DOM/Tool結果から漏らさない（#2516）。Source/Policyの変化をConsumerが**実際に観測したとき**、後着Replyの新規表示拒否と、既表示情報の資格失効を独立に評価する。検知されていない変更の即時撤回保証は主張しない。

**既存ScenarioとExpectedへ渡す5つの最小受入（新Fixture SSOTは作らない）**

| ケース | Desktop/Web/AIで共通に要求する正負対照 | Owner・開始条件 |
| --- | --- | --- |
| **CW-01** | 同じSourceの異Slotを2編集者が更新：Raw Diffで衝突しなくてもMeaning依存が交差する場合はReviewへ。片方の更新によってもう片方の未保存Draftや承認が黙って消えない。 | #1824/#2621/#1124。最初はHeadless 2 Draft/1 Sourceの合成試験。Webサーバ実運用は後段。 |
| **CW-02** | 同じSubject/Slotへの別値・同名別Subject・多重Edgeの差分では、last-writer-winsやGraph表示IDへの転写をしない。既存Source span/Identity/Revisionが一致しないApplyは0件。 | #2497/#4533/#4538/#1124。D2のSlot編集負例に追加。 |
| **CW-03** | Draft A→Source B→A、別Branch/Workspace・Parser/Role版変更・Git Merge後の同byteから、旧Model Acceptance・Review・Build Evidenceを自動復活させない。 | #2621/#1124/#2831/#2600。World/Model/Artifactを別資格として照合。 |
| **CW-04** | 旧Workspace/旧Grantで生成したAI提案や非同期Inspectが新Sessionへ後着：再束縛のない自動Patch/Result表示を拒否。既表示情報の失効と許可済み無関係領域の継続表示を別確認。 | #1124 DS-X11、#2598/#2516。実WebView2/Policy検証は別Claim。 |
| **CW-05** | 同じFunction Contextと許可済みSource SliceをDesktopとWebの異なるConsumerへ投影。各受領/Disclosureを実検証し、Reader/Consumerの受入不足を「描画成功」で代替しない。 | #2586 D1、#1251/#1824。Desktop/Web同時編集をD1のHard Gateへ追加しない。 |

**実装順**：D1の実PG→同一Context→二ConsumerはP0のまま先行。並行してD2でCW-02（既存#4538の局所Patch）とCW-04（既存#1124 DS-X11）のHeadless負例を先に採用候補として検証し、CW-01/03のWeb共同編集とGit Mergeの実装は後段の#1824/#2621へ。何らかのCRDT/集中DB/共有Lock/新Server/新Authorityが必要かは実測してから判断する。最新mainへ本Draft計画が取り込まれたこと、Web共同編集実装・Studio GUI/Policy/DB実験・Owner意味受理の成功は**NOT RUN / NOT ACCEPTED**。


### データ様式の動的意味：依存閉包・差分失効・複数Context保存（2026-11-14追補）

静的な **Structure/Facet→Projection→Operation適格性** だけでは、複数Data Contextが変化したときにどのClaimを維持・撤回・再評価できるか説明できない。[#2582](https://github.com/hat47x/tei-platform/issues/2582)（Query/Context依存とRefresh）と[#2597](https://github.com/hat47x/tei-platform/issues/2597)（複数編集Session/Save）、[#1706](https://github.com/hat47x/tei-platform/issues/1706)（Relation集合の粒度）、[#1708](https://github.com/hat47x/tei-platform/issues/1708)（複数Read Snapshot整合）、[#2564](https://github.com/hat47x/tei-platform/issues/2564)（期待状態/CAS）を**データ様式の時間・変化に対する適格性**として既存D1〜D3へ接続する。新しいReactive Engine、万能Change Log、Snapshot Coordinator、Composite Transaction Runtime、第二Session/Cache/Operation SSOTを作らない。

**変化の意味を4つの独立軸で追跡する**（新Core型名ではない）。
1. **Observed Result Slice**：Subject/Relation/path/aggregateの粒度、値とNULL/未取得/非開示、collectionの必要Scopeとcomplete/partial/unknown、実Consumed Query/Provider Revision。0行はread-covered sliceの結果であり世界の不存在ではない。
2. **Dependency/Required closure**：当該Claimが実際に必要とするField/Relation/Predicate/Query演算/時点。JOINやAggregateの出力列だけでは不足する。`WHERE`の非表示Field、`NOT EXISTS`・LEFT JOIN未一致・Top-N・LIMIT/ORDER・集合差等の**否定/非単調依存**と未知Dependencyを区別し、必要な入力閉包を説明する（#2387/#2582）。
3. **Freshness/Coherence**：Source/Rule/Model版、Function Instanceと選択epoch、Providerの各Read snapshot/評価時点、Consumer実受領を分ける。同じ画面時刻や同じ値でも複数Providerが原子的Snapshotを共有したとは扱わない。Providerが保証する整合性以上は`unqualified`とする（#1708）。
4. **Pending Change/Effect**：View/Sourceの表示差分、Instance+Anchorに属するDirty Change Set、型付きOperation Intent候補、HostがCommitしたResult/Receiptを分離。Read/Refreshは未保存Patchを消さず、Receipt/別Readの値一致だけで当該Attemptの因果Effectを断定しない（#2597/#2598/#1174）。

**局所維持の判断**：必要な依存範囲に変更がなく、現在性・必要Scopeが満たされるClaimだけ維持候補とする。直接変更を観測したClaimは失効/純粋再評価/再Read候補へ。影響依存が未解決なら`no impact`ではなく`unknown/recheck required`へ。Providerが証明できない差分適用の代わりには、既存Queryによるread-only再取得を基準とする。単なるUI再描画やAI推測を整合したデータ変更として扱わない。全面失効も誤成功と同様に計測し、対象外のConsumer/Contextを不必要に停止させない。

| 既存ケース・正本 | 順当な判断／誤判定を防ぐ負例 | 結合先 |
| --- | --- | --- |
| **RC-01/02 (#2582)** | 親H→D1明細2件・D2履歴3件。D2だけ変更してもD1のみの合計は局所保持候補、D1×D2 JOIN/依存Claimは再評価。Opaque依存が残る場合は「無影響」ではない。 | D1の二Consumer実Read後、D1.1 read-only Claim差分判定 |
| **RC-04/05 (#2582)** | LEFT JOINの旧`(H,NULL)`はD2一致行の後着で撤回され得る。NOT EXISTS/Top-N/集合差/集計も追加行で旧結果が消える。部分ページから不存在/完全集計を断定しない。 | #2387/#1706/#1708、D1.1 |
| **RC-03/07/08 (#2582)** | Parent A→Bで遅れて到着したA結果をBへ混入しない。他のInstance Aは保持。Rule改版のみとRenderer改版のみで再計算/再描画範囲を区別し、Provider別Snapshot不一致を明示。 | #1824/CW-04とD1 Consumer、必要ならGo/WASM同じ判断を比較 |
| **ES-01〜04/09 (#2597)** | 同じFunction Definitionの2配置I1/I2でもDirty/Selection/Resultは別。A→B→Aで未保存Patchを誤移転せず、独立Context D1/D2・page1/3に表示された行だけで全件更新を作らない。 | D2 Source編集・Function Edit、#2506/#2500 |
| **ES-05/06/11＋RC-06 (#2597/#2582)** | 2 Patchが同じUI Saveから送られてもall-or-noneではない。実Provider同一Tx確認まで個別結果を保持し、D1成功/D2失敗、ACK-loss/Commit unknown、期待版競合、再Readの因果同一性を別判断。 | D3 #1174/#2564 Effect/Receipt、業務Owner |
| **RC-09/ES-13 (#2582/#2597)** | 遅着・訂正EventのWindowと現在最終値、値は同じでも業務確認/Grantが失効した状態を区別する。可変Stream/Time軸の同値を値一致だけで決めない。 | #4133/#1708、必要なDomainのP2 |

**D1/D2の最初の縦断**：既存#4539の実PG Read→同一Function Context→TableとAggregateの二Consumer受領が成立した範囲でのみ、RC-01とRC-04の**限定Headless** read-only対照を続ける。RC合成Fixture（H/D1/D2）を`public.orders`の実PG Rowsetと混同せず、同一入力を都合よく別世界の証拠へ昇格しない。並行D2ではES-01/02/03/09の2配置・Dirty保持・部分取得を既存#2597/#1180へ委譲する。D3の実Tx・CAS・Receiptは独立Evidenceがある場合だけ検収し、E3実DBの別Operation成功を単一業務Saveの原子性に合算しない。

**実行配置の判断**：Desktop/Webの共通State/Context判定をGo Native／Browser WASM等に置く可能性は#1824/#1145のTarget/Performance比較対象だが、ここでWASM必須ABIや第二のBusiness Rule Engineを決めない。依存失効・Anchor/epoch・Operation候補の**同じ入力→同じ判断**がHeadlessと異種Consumerで再現することを先に確認し、I/O・Grant・CommitはHost/Providerに残す。表示反応速度と再Query回数・過剰失効/誤維持の率を分離して測る。

**今回の資格**：設計・受入順序の統合のみ。#2582/#2597のRC/ESケースは未実装・未実行の計画であり、実PG二Consumer、Headless selective refresh、実Web/WASM、実多Context Tx/Owner意味受理の新試験は**NOT RUN**。新DSL/Core/正式Reactive APIを本追補で採用しない。


### D1.2 測定値の意味・集計軸・再集計の条件付き適格性（2026-11-14追補）

静的なDecimal/NUMERIC型・`SUM/AVG/COUNT`名・Chart値一致だけでは、**Measure×集計軸×粒度×時点×対象集合×演算法則**の合成適格性を決められない。既存[#2593](https://github.com/hat47x/tei-platform/issues/2593)がMeasure/再集計の意味的Pressure、[#2544](https://github.com/hat47x/tei-platform/issues/2544)が演算Law束縛、[#1705](https://github.com/hat47x/tei-platform/issues/1705)がQuantity/Decimal/Unit/丸め、[#1706](https://github.com/hat47x/tei-platform/issues/1706)がRelation集合・fanout、[#2498](https://github.com/hat47x/tei-platform/issues/2498)が異Targetの実Consumer転写、[#2582](https://github.com/hat47x/tei-platform/issues/2582)が変化後の再評価を所有する。新Metric DSL/BI Engine/Query Optimizer/第二Measure Registry/独立Expected正本を導入しない。

**実装基線と資格上限**：[Draft PR #4423](https://github.com/hat47x/tei-platform/pull/4423) は、既存`reference/derivation-relationsets`の集約済み入力を従来`Sum`で再加算する誤成功を拒否し、read-only `Reaggregate(Claim)` で`eligible/ineligible/unknown`を説明する候補実装。PR記載の局所Go/SQLite検査・期待値・独立レビューは**限定Referenceの成果**であり、PR未統合、実SQL生成・実Provider集計・本番Consumer/Owner意味受理は別資格。特に宣言された`Partition`、`Scope`、`AsOf`の真偽や、構成要素の単位整合・丸めの由来を**実元Factから独立に検証しない**ので、`eligible`は「呼び手の宣言が成立する範囲での局所結果」であり、製品の集計保証ではない。

| 既存MCケース／独立Expected | 正当な合成・元Factの条件 | 偽PASSを許さない条件 |
| --- | --- | --- |
| **MC-02 平均** | A:2件×平均10＋B:8件×平均20 → (20+160)/10=**18**。sum/countの対応と被覆が必要 | 平均10と20から単純平均15。局所平均しか残らない場合は中間状態欠落 |
| **MC-03 比率** | A:1/2、B:8/10 → 9/12=**0.75**。分子・分母・対象集合の対応を確認 | 50%と80%の単純平均65%、分母ゼロ、分子分母の対象範囲不一致 |
| **MC-04 重複除去** | A={U1,U2}、B={U2,U3} → 全体**3人**。元Fact識別または適格な distinct 状態を要求 | 部門別2+2=4人、`SUM(DISTINCT amount)`で同額の別業務Factを消す |
| **MC-05 時点残高** | 同日A110+B190=**300**。時点と対象範囲を保持 | 日付を跨ぐ300+300=600を「現在残高」扱い。呼び手のAsOf宣言だけで元行の同時点保証を偽装 |
| **MC-06 取得・NULL・丸め** | 完全な0行／NULL／未取得／partialを区別。0.004×1000は厳密和4と、先に小数2桁丸め済なら0を区別 | 未取得をゼロ、部分AVGを全体AVG、ScaleなしJSON float64再解釈、先丸めと最終丸めを黙って混合 |
| **MC-10 異種Consumer** | 同じMeasure Contract/Observerの結果をGo/SQL/Java・Table/Chart等へ渡し、decimal lexeme/scale、Result Scope/Versionと実受領を対照 | JSON Parse/可視数値一致、集計器単体PASSを外部Provider/Business Meaningの同値とする |

**D1の合格判定を不必要に拡大しない**：まず既存#2586 `order_entry.recent_orders` の限定実PG Read→同一Context→TableとAggregate Consumer受領を成立させる（D1）。次の**D1.2（独立後続の試験単位であり新公開Gateではない）**で、#2593 MC-02/03/04/05の元Fact・明示Scope・演算法則と局所Resultを対照し、Provider側計算とConsumer側計算を実測する。#4423のReferenceは判別用の足場とし、`eligible`を最終製品適合へ自動昇格させない。Quantityの精度/単位・丸めフェーズは#1705、Query演算・JOIN重複は#2387/#1706、異Target転写は#2498/#2077、Provider別Snapshotは#1708と各Ownerで資格判定する。Measureの結果行を元Entityとして編集しない（#2500/#2497）。

**再評価の接続**：#2582の局所失効で`D1-only sum`が維持可能でも、平均の`count`、比率の`denominator`、distinctの共通Person ID、残高の`AsOf`、丸めRule等が変われば合法な再集計条件が変わる。再評価の必要集合には可視値だけでなく**Measure Lawと中間状態の出典**を含め、異なるSource/Rule/Provider版や非単調な入力変化を「値一致なので影響なし」としない。未知または開示できない入力から、許容されない精度・母集団の数値を捏造しない。

**今回の資格**：#4423の既存限定試験とは別に、この追補に伴うGo/SQLite/PG/Java/Wasm/実二Consumer試験・Owner受理は**NOT RUN**。P0実ReadとTEI StudioセルフビルドをMetric BIの全機能実装でブロックしない。正式Measure型・SQL書換え・Core拡張は反例と異種Consumerを検証してから既存Owner/ADRで判断する。

### 標準データの表現と実行を、元の意味へ結び付ける（2026年11月14日）

標準データをGraph、Tree、Table、フォーム、処理図で見せることと、それらを相互に書き戻せることは別である。各画面は同じSubjectと意味定義を参照するが、Sourceの原文、構造間の対応、実行時に選ばれた実装、利用者が受け取った結果を一つの同一性へまとめない。**新しい共通Runtimeや第二の意味正本を作るのではなく、既存の契約に不足する条件だけを実際の反例で確かめる。**

| 対象と設計上の位置付け | 最初に既存試験へ渡す反例・受入条件 | 担当と実装順 |
| --- | --- | --- |
| **構造と宣言的構文**：[main統合済みの設計 #4667](https://github.com/hat47x/tei-platform/pull/4667)はGraph／Tree／Tableの意味対応、JSON Schema、Language Plugin、Studio表示の**実装前の検討**。 | **SM-01／02**：同名Node・多重有向Edge・複数親をTreeへ表示できても、失ったEdge identityを推定して書き戻さない。**SM-06／07／08**：`oneOf`、再帰`$ref`、共有参照の制約と取得範囲を保持できなければread-onlyに退避する。外部Schemaの読み取り、JSONインスタンスの入力、DB Migrationは別操作。 | #2495/#2497/#988/#2620/#1124。まずHeadless表示とSource位置・独立Expectedを比較し、必要になった編集だけPatch→Preview→再解析へ進める。Grammar IR／新Parser基盤の採用は保留。 |
| **処理と図解**：[実行モデル #4665](https://github.com/hat47x/tei-platform/pull/4665)、[Lifecycle #4666](https://github.com/hat47x/tei-platform/pull/4666)、[実行規則の整理 #4668](https://github.com/hat47x/tei-platform/pull/4668)はmainに設計を統合済み。 | 同じFunctionでも**受理したDefinition／選択したEffective Plan／実際のObserved Trace／図面Projection**は別に比較する。**PX-13／14／15**：多重Edgeの誤結合、Provider AとBの取違え、常駐Webと単発CLIの実行規則の違いを判別する。処理切出しでは型、依存、評価順、時刻、Effect、Grant、実受領を欠かさない。 | #4198/#4193/#4502/#1121/#2514。Headlessの意味・分岐確認を先行し、Mock成功を実DB Commitや通知配達に格上げしない。中央Dispatcher／Graph Compiler／独自Hostは増やさない。 |
| **公開FunctionとPlugin**：[Script Launchの設計Draft #4669](https://github.com/hat47x/tei-platform/pull/4669)は未統合。[Plugin構成設計 #4671](https://github.com/hat47x/tei-platform/pull/4671)はmain統合済みで、両者の実行・配布の製品検証はこれから。 | 公開Entry→選択Capability／Provider→実呼出し→Resultを、Source解析成功と区別する。Studio／Generator／Runtime／TestのContributionが同じMeaningを指していても、実Packageロード・Builderの実使用・現在Grantは別証拠。RoleだけでEffect／Actionを実行許可しない。 | #4495/#951/#1016/#1180。既存Resolver／Language Pluginを使い、実Provider・受領者の確認を先に行う。P0にPlugin配布全方式や新ABIを要求しない。 |
| **専門ツールキット**：[数理・統計・計画・科学の設計 #4663](https://github.com/hat47x/tei-platform/pull/4663)はmainに統合済み。[SQLiteの限定実DB回帰 #4670](https://github.com/hat47x/tei-platform/pull/4670)はmain統合済み。ただしTEI製品Query・NoSQL・実Consumerへの横断は未実証。 | WBSの包含関係とPERTの先行関係を区別する。Window集計は粒度・順序・NULL・欠測を、Solverや科学計算は単位・精度・収束条件・独立Expectedを対象ごとに照合する。設計文書中の計算例をR／Mathematica／外部Solverの実行証拠とはしない。 | #4660/#2495/#2544/#2586。Window→WBS／CPM→Solver／DOEの小例を並行評価し、正式なOperator／Provider採用は実測後に判断する。 |

**意味保存を三つに分けて検証する**：[#4667](https://github.com/hat47x/tei-platform/pull/4667)の新しい設計判断を、別のCore enumにはせず、既存のProjection・Query・編集契約の検査条件として使う。

- **構造上の同値**：Node／EdgeのID、向き、多重性、Source occurrence、NULL／missingなどが、必要な範囲で保たれること。GraphをTreeに描けても、二つのEdgeを一つに畳んだ表示から元の関係を復元できるとは限らない。
- **評価上の同値**：同じ入力とProviderの下で、SQLのJOIN粒度・行順・RLS・LIMIT・否定条件・Snapshotなどを含めて、指定された観測が一致すること。列名と型が同じだけでは不十分である。対象全体の取得を確認できないときは、0件を「存在しない」、部分集計を「全件の合計」と説明しない。
- **変更上の同値**：表示からの合法な編集が、意図したSource occurrenceだけを変更し、再解析後にも同じ変更として観測できること。変更していないViewを保存しただけで原文byteを書き換えたり、集計値から更新先の明細を推測したりしない。双方向変換のGet–Put／Put–Get等は既存#2497の**条件付き試験の参考**とし、新しい一律の編集Engineを要求しない。

**次の小さな検証を三件に絞る**。それぞれ既存のIndependent Expectedと担当Issueを使い、別のScenario正本を作らない。

| 順序 | 試すこと | 成功・拒否の判断 |
| --- | --- | --- |
| 1．SourceとViewの安全な往復 | 同名Node・多重有向EdgeのGraph表示に、原文の書式を含む一意属性の局所編集を加える（SM-01／17／18／24、#2497／#1124）。 | 変更しなければSource byteは不変。合法な属性PatchだけをPreview→再Parse→再表示で照合。集計値の明細更新、古いRevision、共有参照先の曖昧な変更は拒否する。 |
| 2．読取専用Schema表示 | 固定したJSON Schema Draft 2020-12の単純・複合制約、共有／再帰参照、未対応VocabularyをHeadlessで読み、Source位置へ戻る（SM-06／07／23、#988／#2620）。 | `oneOf`と`anyOf`、`default`と値の確定を混同しない。未解決参照は不完全と示す。非開示対象の有無を補完件数や診断で漏らさず、編集はまだ有効にしない。 |
| 3．Queryから異なる二つの表示へ | #2586の通常製品Readが成立した**同一Attemptの実Rowset**をTableとAggregateへ渡す（SM-20、#2506／#2586）。 | 実Source・Provider・Row/NULL・粒度とCoverageを保持。RLS・LIMIT・Native SQLの閉包不明を全量へ昇格しない。各Consumerの受領を独立確認し、Reference PGの成功JSONを製品受領の代わりにしない。 |

**処理の切出し・Target移行は上記と異なる検証**である。[main統合済み#4673](https://github.com/hat47x/tei-platform/pull/4673)のPX-23〜29を#4198／#2514へ引き渡す。表示上の辺の置換を制御依存の変更と混同しない。純粋計算の抽出でも、暗黙Read、NULL・丸め・短絡評価、時刻、現在Grant、Generator／Toolchain、実行先の能力とConsumerの受領まで必要な閉包を確かめる。注文確定と通知を分割する場合は、Commit前後、ACK不明、重複配送の差が残るため、同じResult値だけで互換とは判断しない。**World Fingerprint／Model Revision／Artifact Revision、Build省略／Artifact再利用／Evidence再利用も別の判断**として保つ。

**RDBとNoSQLにまたがる統計のFit／Gap**は、[SQLiteの限定実DB回帰をmainへ統合した #4670](https://github.com/hat47x/tei-platform/pull/4670)のCTK-01／04／06を#2387／#2586／#3270へ渡す。四観測のWindow平均（10、15、20、30）、JOIN後の明細金額の過剰集計、全件取得できないDocumentへの黙ったローカル計算を、それぞれ異なる反例にする。Providerが保証する順序・Window・入力範囲・NULL／missing・費用・開示を確認し、能力が足りないときは理由を示して不適格にする。**このFit／Gapや専門ツールキットの実装を、StudioセルフビルドP0の新しい先行条件にはしない**。

**Studioへの引渡し**：どのViewからでも、同じ対象のSource位置・Revision・意味Role・未解決の制約へ戻れることを重視する。図の座標や描画用Edge IDを正式な意味IDにしない。人間とAIの編集候補は共通の型付き対象選択と差分確認を経るが、AIの提案を意味受理や現在の実行権限に昇格させない。未対応のDialect、部分取得、曖昧な逆変換では、読める情報だけを表示し、安全な編集は保留する。

**優先順位は変えない**：全体P0は①非Plugin StudioのStage1→Stage2 Desktop実再構築と当該Stage2のWindows WebView2利用、並行P0は②通常`tei`の実Provider Read→Function Data Context→独立二Consumerの受領である。上記SM/PX/専門ツールキットの試験は既存#1180/#2514の期待値へ分配し、新しい一括受入Gateを作らない。各設計PRのmain統合は実装済み・Ownerによる意味受理済みを意味しない。

## 案件の一巡で追加した七つの作業

申請、現場保守、公開、外部連携、設備監視、予約・制御の模擬実行を[製品設計](../design/PRODUCT_COMPLETION_DESIGN.md#案件の一巡から標準構成の不足を補う)で具体化した。以下の作業は既存G項目の読取り・外部作用・出典追跡へ接続し、案件に必要な入出力と寿命を補う。

| 項目・既存項目との差分 | 最初の作業と独立した正負の受入 | 段階と担当 |
| --- | --- | --- |
| G65 業務添付・大容量資産。G25の接続に、転送・関連付け・派生・利用終了を加える | 小さい合成資産を中断・再開し、確認した版へ関連付けて別Readで取得する。DBだけ成功、物体だけ成功、位置・内容の不一致、期限・権限失効、派生の途中失敗を試す | P1。転送・保管・変換の選択を既存構成へ結ぶ。[#4426](https://github.com/hat47x/tei-platform/issues/4426) |
| G66 アプリケーション内の全文・複合検索。G25の取得に、索引の追随と探索操作を加える | 同じ申請集合へ構造・全文条件とページ継続を適用する。更新・削除・索引遅れ・同順位・非開示の件数/候補/抜粋・失効した継続を対照にする | P1。既存Query/Contextの最小能力、言語のFixture、必要な索引の順で検証。[#4427](https://github.com/hat47x/tei-platform/issues/4427)、[#2622](https://github.com/hat47x/tei-platform/issues/2622) |
| G67 業務通知。G18/G19の送信に、宛先・本文版・期限と結果を加える | 同じ確定申請から合成送信先へ通知し、宛先ごとの結果を読む。確定前、重複、宛先変更、版混在、部分失敗、応答断、期限切れを試す。受付・配達を人の承認にしない | P1。既存Scenarioと隔離Providerを先に使い、実配信は別試験。[#4428](https://github.com/hat47x/tei-platform/issues/4428)、[#1130](https://github.com/hat47x/tei-platform/issues/1130) |
| G68 同期API・イベントの外部契約。G19の方式選択に、Schema・向き・版互換を加える | 一要求・一応答または一イベントをNative原本から既存Operationへ対応付ける。未知拡張、NULL/欠落、送受の逆転、旧Client、新Server、業務不成立と技術エラーを独立Expectedで照合する | P0でread-onlyの契約比較、P1/P2で必要な実接続。型・Functionは#4350、表層は#4359の責務。[#2542](https://github.com/hat47x/tei-platform/issues/2542)、[#3274](https://github.com/hat47x/tei-platform/issues/3274) |
| G69 継続イベントの時間・流量・再接続。G19の連携能力に、続けて受ける間の条件を加える | 固定イベント列で窓、遅着、上限、再開位置を比較する。Snapshot/差分間の欠落、重複、保持期限超過、確定後の訂正、購読失効を試す。過負荷の間引きを取引や実機指令へ流用しない | 必要な案件のP2。有限列の検査は先行可。Broker/Targetの保証範囲を明示する。[#2542](https://github.com/hat47x/tei-platform/issues/2542)、[#1130](https://github.com/hat47x/tei-platform/issues/1130)、[#3790](https://github.com/hat47x/tei-platform/issues/3790) |
| G70 設計に戻るデバッグと模擬実行。G06の出典移動に、観察・停止・再現を加える | 最初は同じ試行の入力・規則・分岐・結果をread-onlyでたどる。段階実行は対応Targetだけに追加する。時計・乱数・外部応答を固定した期限の対照で、再生時の外部作用0件を確認し、旧版・開示不足・対応外を明示する | P0で診断、P2で必要な段階実行・模擬時計。[#3789](https://github.com/hat47x/tei-platform/issues/3789)、[#1180](https://github.com/hat47x/tei-platform/issues/1180)、[#1131](https://github.com/hat47x/tei-platform/issues/1131)、[#988](https://github.com/hat47x/tei-platform/issues/988) |
| G71 利用者とSessionの一巡。G17の現在権限に、参加・回復・終了を加える | 初期案件で選んだ認証方式の登録/招待、再認証、組織切替、終了を通す。別IdPの同名主体、回復の誤結合、失効後の別タブ/端末、IdP接続断を試す。RP・IdP・進行中の操作の終了を個別に確認する | P1。方式別適合は[#3412](https://github.com/hat47x/tei-platform/issues/3412)、Sessionは[#1043](https://github.com/hat47x/tei-platform/issues/1043)、実作用の権限は[#1174](https://github.com/hat47x/tei-platform/issues/1174) |

P1では最初の申請に必要な基本添付・検索・通知・認証を選ぶ。動画の派生処理、専用検索エンジン、複数IdPの連携、無制限の継続配信までを一律の開始条件にしない。各担当は利用に必要な最小能力を先に宣言し、未対応と未実施を残す。

## 最初の申請を、部品の成功から案件の成立へ進める

公開合成データで組織Aの申請者と審査者、組織Bの対照利用者、添付一件、検索語、通知先を固定する。構成は一つの認証方式、DB内検索、小さいファイル、隔離した通知先から始める。独立Expectedには対象ID、確定した値と版、許可される操作、隠す対象、確認する外部結果を先に記す。

| 一巡の区間 | 同じ対象と版で確かめること | 必須の反例 |
| --- | --- | --- |
| 参加・下書き・添付 | 本人と組織の選択、未確定の下書き、内容を検査した資産の関連付け | 組織Bの同名主体、中断、DBだけ成功、転送だけ成功 |
| 検索・提出 | 本文と条件による同じ申請の特定、提出時の現在権限、確定した版 | 非開示の候補・件数、索引遅れ、基準版の競合 |
| 通知・審査 | 確定後の正しい宛先、同じ申請の独立再読取り、審査操作 | 失効したSession、旧宛先、二重通知、一部配信不明、旧版の承認 |
| 帳票・取得・訂正 | 同じ確定値からの帳票、許可付き取得、旧発行物と訂正版の別ID | マスターやテンプレート改版、取得時の失効、旧発行物の上書き |
| 更新・引継ぎ | 構成と版を固定した再実行、案件差分の保持、復旧手順の実行 | Providerの非互換、再生成による独自修正の欠落、復旧後の二重作用 |

この一巡では最初にHeadlessで業務操作と結果を確認し、同じScenarioをWeb等のTargetへ接続する。各担当の単体成功を合算して一巡の成功としない。同じ条件・同じ試行で必要な区間が通った範囲を記録し、画面操作、視覚品質、実配信・印刷、利用者の受入は該当する実環境で確認する。構造契約や合成Providerの検査はP0の実Readと並行できるが、製品の案件成立には実Readと利用側確認を省かない。

## 依存に沿って進める

| 段階 | 優先する利用と成果 | 着手に必要なもの |
| --- | --- | --- |
| P0 | 一つのSourceと改訂から、読める設計、通常の編集・診断、選択した実装、実Read、利用側での確認へつなぐ。独立Expectedと用途限定の文脈を用意する | 既存の管理Sourceと意味の再構築、実行・検証の契約を使う。提供実装の実呼出しと、Functionの実Readはそれぞれの担当で並行して進める |
| P1 | 標準の初回構成で一件を完了し、帳票、基本の複合表現、別Target、導入・更新・復旧まで検収する。性能と費用の基準を得る | 各操作に必要なP0の契約と結果。ヘッドレスで配置・編集差分を検証し、その操作をGUIへつなぐ |
| P2 | 公開サイト、非同期連携、画像からの帳票化、科学・時間媒体、複数Plugin生成を対照に共通化を見直す | 対象分野に必要な標準構造、変換、実行先を固定する。全部のP1が終わることを一律の条件にしない |
| P3 | VR、デジタルツイン、必要な構成の継続運用・段階更新・災害復旧を実環境で確認する | 対象の識別、時間と座標、入力と編集、権限、精度、資源上限に必要な検証。姿勢・観測の鮮度、継続中の仕事と復元後の状態を各実行先で確認する |

段階名は実装順の整理である。個々の作業は、その入力と契約がそろえば先に試せる。例えば帳票の確定データと既存エンジン比較は、複合表現やWasmの完成を待たずに進められる。複合表現の軸・選択・編集の評価もXR機器なしで合成データから始められる。GUIは対応する操作のヘッドレス検証の後に接続し、全分野を一つの依存鎖にしない。

## 最初に通す五つの利用

| 利用 | 同じ試行で確認するもの | 最小の反例と次の判断 |
| --- | --- | --- |
| 一つの受注Functionを読む | Source、Query、Alias、実Provider、Rowset、Context、利用側、独立Expected | 同名の別SQLや別集合、空・partialを検出する。読取りの成功を注文確定や発行の成功へ広げない |
| 一つの設計変更を引き受ける | 未保存の内容、診断、参照、差分、実際に使った定義と再試験 | 外部改訂、旧応答、同名異ID、途中保存を試す。利用者がSourceへ戻り、同じ条件で再確認できることを測る |
| 確定値を帳票へ出す | 同じ確定値からPDFとExcelを出し、保存済み発行物と訂正版を取り出す | 価格・コード表の改訂、多数明細、長文、部分取得を試す。値、版面、実印刷と読取り、配布結果をそれぞれ確認する |
| 表現から同じ対象を編集する | ヘッドレスの配置・選択・差分を、帳票と科学の小入力で照合する | 25%とmmの混在、投影で複数になる候補、旧版・権限不足を試す。その後にWebの保存・再読込み・PDFと別入力手段へつなぐ |
| 案件を標準構成から引き継ぐ | 受注・申請・公開等の必須条件、構成、実接続、更新と復旧を記録する | 未成立や対応範囲外も含めて集計する。既存部品で足りない箇所は短い依頼から拡張し、単体と組合せを検査する |

最初のFixtureは物流の由来とGISの座標変換という既存の分野横断計画も併用する。台帳、医療情報連携、設備、科学、映像、編集、教育・組織、申請、イベント、学習データは #3791・#2501 の事例として継続する。学習データでは同じ対象の学習・評価への混入、同名ファイルの改版、非公開行の混入を反例にする。分野の意味と受入判断を専門の担当へ戻し、共通形式に合わせるために意味を落とさない。

## 最新の横断検証を実Consumerの受入へ接続する（2026年11月14日）

局所的な試験が増えても、同じ要求を通常の製品入口から実行し、必要な情報を権限に応じた利用先へ届けたことは、別に確かめる必要がある。Queryの隔離PostgreSQL参照試験、取得範囲の限定観測、BuildのToolchain観測、外部Evidenceの読取り、Pluginの段階別Contribution、B2Cと組織の二時点照会を一つの成功として扱わない。

新しい検査Engineや独立した期待値の正本を追加するのではなく、各担当の試験へ共通の照合観点を渡す。対象Sourceと意味の版、実Provider・実Child、Contextの取得範囲、現在権限、最終Consumerの受領、独立Expectedを段階ごとに確認する。製品のP0は、**StudioのStage1→Stage2実Windowsセルフビルド**と、**通常CLI→実Read→独立Consumer**の二本を維持する。B2Cの本人確認、組織履歴、外部REA、Pluginの実使用は各Ownerの後続資格として進め、未確認の部分を完成扱いしない。

詳しい境界、10件の負例、Owner別の次の検収は[Sourceから実Consumerへの横断受入設計](../design/CONSUMER_QUALIFIED_CROSS_LANE_ACCEPTANCE_2026-11-14.md)に整理した。以下の「進行の記録」では、その観点を既存IssueのAttemptと結果へ結び付ける。

## CLIを起点とするTEIセルフビルドとPluginの段階的な自己構築（2026年11月14日）

TEIのセルフビルドは、最終的にはTEI Studio Desktopを含むTEI開発基盤自身をTEIで構築することを目指す。ただし**最初の実証にStudioの画面は必要ない**。既存のTEI CLIを起点として、TEIの定義から次の世代のCLIと小さなPluginを生成し、生成されたCLI自身が同じ仕事を続けられるかを先に確かめる。MCPは同じ型付き操作へ接続する別の入口として利用できるが、MCP Serverの完成をセルフビルドの必須条件にはしない。DesktopのStage1→Stage2と実WebView2は、このHeadlessな実証で得た証拠を引き継いで後から接続する。

**Goで書いた既存関数を呼び出すだけの`*.tei.md`は、セルフビルドの成功例とは数えない。** TEI定義に書いた条件や関係が、図表を含む意味モデルとして読まれ、実Goコードや設定へ生成され、成果物の振る舞いを変えることが必要である。手書きGoは当面のSeed、Target依存部品、複雑な計算の実装として許容し、TEIが担う部分と委譲する部分を明示する。Plugin自身についても同じ基準で実装範囲を広げ、定義を簡素化できたかを継続的に確かめる。

### 最初に通す五つの工程

| 工程 | 行うこと | 検収すること |
| --- | --- | --- |
| **SB-0 固定Seed** | 現行Go製TEI CLIを起点として、Source、Target、Builder、依存の版と入力を固定 | 既存Plugin Buildの正常・拒否テストを維持。非PluginのCLI／Studioを架空Plugin IDへ押し込まない |
| **SB-1 意味のある定義** | 一つのread-only Inspectorまたは純粋変換Pluginを選び、`*.tei.md`に処理・関係の**図**と、入力・出力・分岐・制約の**表または構造化宣言**を置く | 図のNode／Edge、表の項目、Source occurrence、意味候補、受理済み意味を対応付ける。未実装の正式DSL文法を捏造しない |
| **SB-2 実コード生成** | 同じ定義から必要なGoコードと契約情報を生成し、実CLIで読み・検査・実行する。TEI側の条件を変更したとき実際の生成内容・結果が変わる一例を作る | 実コンパイル入力と生成SourceのRole／Digest、選択したProviderの実Call、独立Expected。委譲用ラッパーだけなら未達 |
| **SB-3 CLI再入とPlugin自己構築** | SeedがStage1 CLIを生成し、その**Stage1実Process**がStage2 CLIを生成する。同じ仕組みで少なくとも一つのPluginをTEIから生成してCLIへ組み込み、実呼出しする | 親子Build Attempt、実Child・Toolchain・入力Byte・成果物・Entry／実Consumerの関係。Stage0が別々に二回Buildしたものを自己再構築と数えない |
| **SB-4 Studioへ展開** | 検証済みSource／Operation／PluginをStudioと生成Windows Desktopから使用する | 同じ意味と各Targetの実Artifact／実Consumerを照合。WebView2のB0成功をStage1→Stage2の自己再構築へ自動昇格しない |

最初から大きなDSL処理系を作らず、既存Markdown・Markdown表・Mermaid（必要に応じてGraphviz）・標準データ構造・Language／Operator／Target Generatorの機能をできるだけ使う。図の座標や説明文を実行意味の第二正本にはしない。外部で実装する部品はTEI側に型付きInput／Output／Effect／Failure／Provider契約とTODO／Mockを持たせるが、Mock成功と実Provider成功は区別する。Coreに新しい共通Runner、第二Resolver、専用Hostを増やさない。

### 最初の実装PRを、一つのGraph／Ruleから始める

S0〜S4を一度に実装しない。**最初のPRはS0のReaderとS1の実装ロジック生成を最小単位**とし、選択したPluginの実呼出し（S2）、そのPluginを用いる次Stage再構築（S3）、定義変更からの再生成（S4）は、同じFixtureを引き継いで別の証拠で判定する。

最初に固定する例は、**同じ表示名を持つNode `n-1` と `n-2`、`n-1 → n-2`のEdge `e-1` と `e-2`、Edge IDの重複を拒否する検査規則**である。初期のAuthoringでは**Markdown表を関係の定義元とし、MermaidをGitHubでも読める導出図**とする。Graphviz等の別表層は後続の対応範囲とし、Role／Identity／向き／多重性／規則のうち実装上意味を持つ部分を、実Readerがどこから取得したか追跡する。架空のTEI構文はここでは定めない。

| 固定する対照 | 独立Expectedと合格の上限 |
| --- | --- |
| 元の定義：Nodeは2件、同じ向きの別ID Edgeは2件 | 生成処理でNode IDを区別し、両Edgeを保持した検査／参照結果を返す。**Node名や端点だけで重複排除しない**。書き出しただけの静的JSONではなく、定義から生成した検証／選択ロジックが動いたことを確認する |
| `e-2`の向きだけを逆にする | 実生成コードの該当判断・実Plugin出力が変わり、正方向のEdgeは1件、逆方向は1件になる。外部の固定Go処理へ全部委譲していれば不合格 |
| `e-2`のIDを`e-1`と重複させる | Reader／Validator／生成された規則の担当境界を明示したうえで、実行前または実検査時に不適格を報告する。誤ったGraphを「Edge 1件」として黙って正規化しない |
| Mermaidの配置や文書説明だけを変える | Source byte／表示の変更は残す。意味上のGraphと生成すべきロジックが同じなら、その範囲を不要に再Buildしない。ただしCache／Evidenceの再利用資格は別に判定する |
| Stage1が生成したPluginを次Stageの実Buildへ渡す | Stage1実Process ID／Input closure／選択Builder／生成Artifact／Stage2での実Plugin Callを照合する。**Seedが二回Buildしただけでは達成としない** |

**読取り専用の最初のGo実証**：[reference/tei-md-graph-reader](../../reference/tei-md-graph-reader/README.md) では、前記5入力の二つのMarkdown表からNode/Edge候補を取得し、元byte位置、同名別ID、並行Edge、重複ID拒否、導出Mermaidの不一致を独立Expectedへ照合する。製品の`*.tei.md` ReaderやSource/Roleの正式受理は**未成立**であり、現行`TEI_MD_PARSER_UNBOUND`を維持する。次はOwnerが既存Language Plugin／Workspace Loadへ選択されたSource byteを結び付け、さらに実Go Generatorと通常CLIの実使用を検収する。

**固定した入力と期待結果**：[testdata/tei-md-graph](../../testdata/tei-md-graph/README.md)に、基準、`e-2`反転、Mermaidのレイアウト変更、Edge ID重複、表と導出図の不一致の5例を通常の`*.tei.md`で置き、結果を別の`independent-expected.candidate.json`に記録した。これらは**既存#1180/#2514に採用を引き渡す候補**であり、`*.tei.md`を実Loaderが読めるという意味ではない。管理Source Rootの未対応Guard `TEI_MD_PARSER_UNBOUND`はReaderが正式に接続されるまで維持する。最初の技術課題は、選択した表・Fenceの元byte位置を保存し、Role／意味IDに結び付けること。MermaidのN1/N2やEdgeラベルを実Relation IDへ黙って昇格させない。

**実装担当へ渡す受入票**：最初に「Edge同一性の保持」「重複IDの拒否」「方向別の抽出」など対象の機能上の義務を固定する。義務ごとに、(a) TEIのどのSource/Ruleが実装を規定したか、(b) 生成したGoのどこが挙動を担うか、(c) 外部Goが担当する境界、(d) 実Builderが読み込んだByteとArtifact、(e) 選択Pluginの実Callと独立Expected、(f) 定義変更前後の手修正・重複記述・再検証作業を記録する。**未観測の項目は未達とし、比率の分母を試験後に縮めない**。

最初のプルリクエストで`*.tei.md`が正式にLoadできなければ、結果をunsupportedと明示し、#2621/#4359の**既存Reader/Source索引を通る最小対応**だけを追加する。旧`*.tei`や手動変換を使うReference経路は分けて記録する。文法全体、Studio GUI、汎用Graph Compiler、DB Provider、独立Hostを初回の完成条件に加えない。

### 「TEIで実装できた範囲」と「定義を簡素化できたか」を測る

| 観測 | 記録方法 | 成功とみなさない例 |
| --- | --- | --- |
| **意味実装の内訳** | 最初に機能・分岐・検証規則の対象集合を固定し、①TEI定義からの生成、②TEIから外部Goへ委譲、③手書きGoのみ、④未対応に分類する。PluginについてもRole・Targetを分けて分母と未対応を残す | 一行の委譲コードを生成しただけでFunction全体をTEI実装済みと数える |
| **生成物の実消費** | Source／Model Revision、Generator／Toolchain版、出力Role／Digest、実Child／Builder、実Entry／受信結果を連結する | ファイルの存在や同じJSON結果だけで、選択・コンパイル・実使用が証明できたとする |
| **定義の簡素化** | 同じ独立Expectedで、保守するTEI定義、補助Go、重複設定、例外指定、手修正、意味理解のための行き来を改訂前後で比較する | 行数が減っても隠れたGo条件や暗黙の既定値へ意味を移しただけの場合 |
| **変更への耐性** | 一つの表行／有向関係／Plugin契約を変え、生成差分→再Build→独立Expected→実Consumerを検証する | 旧Artifactの再利用、同名別Subject、A→B→Aの旧Evidenceを現在適格とする |
| **Pluginの自己構築割合** | 先に列挙したPlugin機能集合を対象に、TEIが生成して実呼出しまで確認できた機能数と、外部実装へ委譲した機能数を併記し、改訂ごとに比較する | Packageを一つ生成したので、内部の全OperatorとProviderが自己構築されたと数える |

初期段階に恣意的な百分率目標は置かない。**まず一機能で、TEI定義の変更が実行結果へ反映され、手書きGoの増加なしに正負の独立Expectedが通ること**を証明する。次に性質の異なるPluginへ拡大し、同じ基準で比率と保守量の変化を測る。数値だけを良くするために機能分割や委譲箇所を変更しない。

### 既存Expectedへ引き渡す負例

SBX-01：図のレイアウト変更では意味が変わらず、関係の向きや多重度を変更すれば生成内容・結果が正しく変わる。SBX-02：TEIの表の条件を変えてもGoの固定処理しか動かない場合は「委譲のみ」として失格。SBX-03：生成GoのDigestが一致しても、Builderが別Sourceや未選択Pluginを実消費すれば失格。SBX-04：Stage0が別々に二つのBinaryをBuildしただけではStage1→Stage2再入不適格。SBX-05：Plugin A→B→Aの後に旧Binding／Grant／Receiptが自動復活すれば失格。SBX-06：定義を短くした結果、NULL／欠損／評価順／Effect・Failureが変われば不適格。これらは[#1180](https://github.com/hat47x/tei-platform/issues/1180)・[#2514](https://github.com/hat47x/tei-platform/issues/2514)の既存Independent Expectedへ渡し、新しいScenario正本は作らない。

### 既存計画との関係

**CLI先行はDesktopセルフビルドの最終目標を撤回するものではない。** 最初の検証をCLIで軽量化しつつ、[#2831](https://github.com/hat47x/tei-platform/issues/2831)・[#4583](https://github.com/hat47x/tei-platform/issues/4583)のBuildExecution／ArtifactSet／親子Attemptを共通化する。現行BuildIntentの`Subject.Kind=plugin`制約は[#1080](https://github.com/hat47x/tei-platform/issues/1080)・[#1097](https://github.com/hat47x/tei-platform/issues/1097)で非Plugin Build Subjectの不足として扱う。Plugin選択・段階別Contributionは[#951](https://github.com/hat47x/tei-platform/issues/951)・[#1251](https://github.com/hat47x/tei-platform/issues/1251)、自己記述Languageと糖衣形式は[#988](https://github.com/hat47x/tei-platform/issues/988)・[#1018](https://github.com/hat47x/tei-platform/issues/1018)の責務とする。

CLI／MCP／Studioは同じ型付きApplication Operationを異なる入口から使うものであり、入口の共通化はHost認可や実効果の証拠を省く理由にならない。Runtimeで使わないGenerator／Editor専用Pluginは、必要のない生成成果物に無条件同梱しない。本節は実装順と測定方法の合意案であり、TEIが生成したGoによるCLI／Plugin再構築、定義の簡素化、Stage1→Stage2製品成功を**実証済みとはしない**。

### 一つの変更を、生成・実行・再検証まで追跡する

SB-1/SB-2で導入するGraph/Ruleの固定例について、**初回から変更前後を同じ独立Expectedで比較**する。最初の実装PRは新しい汎用Graph Compilerや増分Cacheを要求しないが、将来#2600で正しい再利用を行うため、何が変わったかと、どの工程を実行したかを取り違えず記録する。詳細な契約は[実行モデル§37〜39](../design/PROGRAM_EXECUTION_MODEL_PROJECTIONS_2026-11.md)に集約する。

| 比較する入力 | 最初に見る意味・生成の差 | 実行・再利用上の停止条件 |
| --- | --- | --- |
| 基準：同名Node二件・別IDの同向Edge二本 | 出辺ID・件数の計算とID重複の拒否が、TEIのRuleから生成される | 固定JSONや外部Goへの丸投げは自己生成実装として扱わない |
| `e-2`の方向だけ反転 | `n-1`からの出辺が二本→一本、`n-2`から零本→一本 | Graph表示だけ更新して実Go処理・実Provider結果が旧状態なら不合格 |
| `e-2`を重複IDへ変更 | 一意性違反をReader/Validator/生成規則の適切な位置で拒否 | 同名や同端点を理由にEdgeを黙って統合しない |
| 図の配置・注釈だけ変更 | 意味上の計算は不変、Source/表示の変更は追跡 | Raw Source再確認を省略したと偽らず、無関係な下流再Buildだけを省略可能か検査 |
| Provider/Builderの選択やSourceのA→B→A | 結果が同じでも選択先・実消費byte・現在資格が変化し得る | 旧ArtifactやReceiptを現在の実Provider使用・Owner受理へ自動昇格しない |

記録の最小単位は**一つのSubject、固定したSourceとRole、選択Provider/Generator、Build Attempt、実Artifact、実Consumer**である。未接続ならその境界を明示する。各工程でParser/意味解決/生成/Go Build/Plugin実行に費やした時間、実行を省略した理由、未観測の入力も残す。Source/Meaning/Artifactの三種のRevisionを一つのFingerprintへまとめない。

初回の実Go生成は、意味処理の実装部分と手書きの補助Goを分けて検証し、生成定義から出辺IDの判定が実際に変わることを確かめる。通常`tei` CLIと独立Consumerの実Call、Stage1 CLIによるStage2 CLI再入、正式Studio DesktopのStage1→Stage2は、**異なる到達点として順番に実証**する。部分的な成功を足して全体成功と呼ばない。

この比較はSBX-01〜06と#1180/#2514、#2595の実消費Source、#2600の再利用、#4495/#1251の製品Provider、#2831/#4583の実セルフビルドの担当へ引き渡す。新たなScenario正本やRuntimeは設けない。**状態：実証手順の追補。新Go生成・増分skip・実CLI再入・正式Desktop再入の合格は未確認。**

## プロセス・イベント・応答を共通の意味から説明する（設計候補）

Webの要求、CLIの起動、タイマー、Message、画面操作には、それぞれ異なるHost・起動方式がある。ここで標準化を検討するのは一つのイベントループではなく、**Trigger→型付きInvocation→Function/Rule→State/Effect→Outcome→Response／Consumer受領**という意味の対応である。長寿命Processの起動・停止と、一件の処理の成功・失敗も混同しない。

初期の受入は、受注受付の説明用Fixtureを使い、HTTP/CLIの二入口が同じ意味Operationへ至るかをHeadlessで照合する。期限の境界、現在Grant、DB Commit後の通知結果不明、重複・逆順Event、未実装のTimer Providerを負例にする。外部部品はContractとTODOを先に記述し、既存Driver ScenarioのMockと独立Expectedで基本設計からシミュレーションする。ただし、Mockの成功は実DBやBrokerの実行証拠にならない。

詳細は[Lifecycle・Trigger・Operationの共通意味設計](../design/PROCESS_LIFECYCLE_TRIGGER_OPERATION_SEMANTICS_2026-11.md)を参照する。Ownerは[共通Application API #1158](https://github.com/hat47x/tei-platform/issues/1158)、[Script Launch #4495](https://github.com/hat47x/tei-platform/issues/4495)等の既存Issueに分け、新Runtime・Dispatcher・Schedulerを前提にしない。**現行P0のStudio Stage1→Stage2と通常CLIの実Readには、新しい先行依存を追加しない。**

## 進行の記録と完了条件

各G項目には、対象と版、既存の足場、次の作業、独立した正例・反例、依存、担当Issue、得た結果または未実施の理由を残す。状態は定義済み、実装候補、局所試験、通常入口での実行、利用側確認、案件受入を用途に応じて記す。Issueをclosedにしただけで、別の組合せや実行先が成立したと扱わない。

中小規模案件の対応率は事前に定めた案件と必須条件の集合に対して測る。標準のみ、既存資産接続、新規拡張、未成立を分け、保留・実施不能の理由を示す。大規模案件の省力化は生成時間に加え、追加質問、手修正、レビュー、統合、更新と引継ぎを比較する。性能は正しい結果を保った入力・構成で測り、量、待ち時間、メモリ、費用と取消後の処理を残す。

計画への反映が完了した状態は全G項目の担当、設計への接続、次の受入、依存を確認できることとする。製品としての達成は各Issueと既存の受入経路で判定する。理想像が更新された場合は、既存の項目との関係を先に確認し、不足する作業と条件を同じ担当へ戻す。 [生産性 #2018](https://github.com/hat47x/tei-platform/issues/2018) · [案件対応範囲 #3796](https://github.com/hat47x/tei-platform/issues/3796)
