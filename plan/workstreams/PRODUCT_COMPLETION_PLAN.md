# 理想の利用像を製品の受入へつなぐ計画

- この計画は設計書の読書きから実行結果の確認、分野別の標準構成、導入・変更・引継ぎまでを対象とする。
- 作業と評価の入口は [#3796](https://github.com/hat47x/tei-platform/issues/3796)。G01〜G71はこの計画の引渡しを追うための項目IDである。G01〜G64を引き継ぎ、案件の一巡に必要なG65〜G71を加える。正式な製品要求や独立した試験番号は既存の正本を使う。
- 実装方法は[製品設計](../design/PRODUCT_COMPLETION_DESIGN.md)、着手順の基準は[開発全体計画](../MASTER_PLAN.md)、現行の実証は[実装状況](../../docs/status.md)で確認する。

## 完成と判断する単位

一つの案件で、必要な設計を人が読み、許された範囲を変更し、選んだ実装で実行し、利用側の結果を独立に確認できることを判定する。案件に必要な導入、失敗時対応、更新、引継ぎも受入に含める。

受注では確定した値、公開では利用側から読める版、外部連携では相手側で観測した状態を確認する。受付、コード生成、単体試験、参照実装の成功は、それぞれ確認した範囲として保持する。案件の必須条件を満たした結果と結び付けてから成立に数える。

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
- **R0/R1の負の依存とSource存在（B0-S1追加負例）**：入力に`BaseRevision/CandidateRevision`があっても、B0-Rの`Edges`は一式の申告Graphであり**両世界のEdge差分や実Consumed Closureの観測ではない**。R0にあった`Source → Function/Consumer`の依存辺をR1で削除・別Subjectへ再束縛した場合、R1の変更元からConsumerへ到達0でもConsumer不変とは断定しない。旧辺の依存先を変更候補・未知のまま保ち、R0/R1のSource・対象・方向・Scopeを揃えた比較と独立Expectedを[#4518](https://github.com/hat47x/tei-platform/issues/4518) N16／[#2582](https://github.com/hat47x/tei-platform/issues/2582)へ渡す。`SELECT *`／否定述語／Dialect別SQLの新規候補追加も**変わらない元SQL byteだけでは選択・結果・作用の不変を証明しない**ため、Queryの実選択は#2622/#2586 HJ-08で扱う。また、単一Subject参照0件とSubject不存在を分け、[実測Issue #4577](https://github.com/hat47x/tei-platform/issues/4577)とmain統合済み[PR #4584](https://github.com/hat47x/tei-platform/pull/4584)の`SUBJECT_NOT_FOUND`拒否を区別する。CLI Subject存在検査の修正は参照0件の全域不使用判定、現在Source・Grant、Native依存閉包を証明しない。第二Query Parser/Graph Solver/Registryを作らない。
- **問い相対の旧依存辺の検収（N16/N17）**：明示されたR0-only依存辺から旧Consumerを影響候補に残す限定Referenceは[main #4591](https://github.com/hat47x/tei-platform/pull/4591)へ統合済み（`8709aae7`）。ただし、旧辺すべてを変更元とみなすと**当該Required Consumerとは無関係な削除辺**や、別のRelease ConsumerへのR1下流分岐まで誤って失効させ得る。[PR #4592](https://github.com/hat47x/tei-platform/pull/4592)でR0-only辺とR1候補辺の**問い相対Required上流集合**だけを旧辺変更の対象とする修正・二段旧依存削除・無関係下流Siblingの独立Expectedを提案中。明示`changed`の通常到達は別扱いにする。N16/N17の申告Graph上の成功はR0/R1実Source読取・Imported Source／Native/Queryの完全な依存探索の証明ではない。#3262がRequired Universeの取得根拠、#2582が局所再取得を所有する。#4592のCI・main統合は別確認。
- **Studio同一HEADの実Windows B0とBuild Intentの断線**：[統合Draft #4587](https://github.com/hat47x/tei-platform/pull/4587)の実Windows CIは、一つのexact HEADで生成Inspector readerの使用・配信Workbench JS byte一致・Native Wails WebView2を確認（[#4583 B0](https://github.com/hat47x/tei-platform/issues/4583)）。ただしDraftであり、**N0 Inspector Contract→N1 PoC B Go/TS→N2 esbuild/Embed→N3 nested Studio Wails exeの全工程がTEI BuildIntent/実Builderへ宣言・実消費対応したことは未実証**。B1には変更前のBuild依存辺が削除された場合の実Consumed残留、無関係な旧Sourceの過剰再Build、別HEADの成功の継ぎ合わせをそれぞれ独立負例として追加する（[#4583のBD-07〜09](https://github.com/hat47x/tei-platform/issues/4583#issuecomment-6846881454)）。B2 Stage1実TEI Process→Stage2 Desktop、B3 Stage2 WebView2は別のNOT RUN資格。TEI Studio全体のセルフビルドはB2Cの新Hard Gateを増やさず#2831/#1824で優先する。
- **B0-Rの宣言入力検証とStudio優先度（2026-11-13）**：既存のJSON入力が重複項目・Case Shadow・容量超過により曖昧だった問題は[main #4565](https://github.com/hat47x/tei-platform/pull/4565)で修正済み（`8a119731`）。UnicodeのSimple Fold迂回は[#4580](https://github.com/hat47x/tei-platform/pull/4580)の未マージ修正候補で、確認時のexact-head健全性CIは**gofmt不一致で失敗**。製品全体の最優先は[#2831](https://github.com/hat47x/tei-platform/issues/2831)／[#1824](https://github.com/hat47x/tei-platform/issues/1824)の**実TEI Studio Desktopセルフビルド**であり、[生成Inspector部品の実Windows試行Draft #4572](https://github.com/hat47x/tei-platform/pull/4572)と[生成Desktop WebView2検証Draft #4579](https://github.com/hat47x/tei-platform/pull/4579)は区別して扱う。全体計画main[#4581](https://github.com/hat47x/tei-platform/pull/4581)に不足したBuild graph／実Builderの接続工程を反映済み。B2Cは同じFunction Contextの異質Consumerとして並行し、Studio P0への新規Hard Gateを設けない。
- **順序と証拠の上限**：#2586の実Function→実Query/Provider→Rowset→Function Data Context→ConsumerをP0に維持し、B0-Sの完成をHard Gateとしない。#4350 DF-X06は標準表現/Nativeの異質Consumerの同一意味を検査。#4496/#4515は固定Subject/Scope/Read epoch/操作資格を所有し、#4507/#4514のSource-only driftでは同byteの復帰とSource/Authorityの現行性を区別する。B1a実Native Consumer、B1b実Commerce Effect/別Read、B2実配信/Exposure/Consent、B3のCVR・利益因果評価は**別の未実証資格**。
- **性能・プラグイン・Evidence**：#2018でH0 Native単独/H1 TEIのBuild時契約/H2任意Experiment Adapterを、同じ変更・独立Expectedと導入/改訂/運用費用で比較する。#951/#4198/#3278のProvider/Graph/Traitは実選択と実呼出しを照合。#4307/#2516に従い非公開Evidence・その存在情報をGitHubや外部Harnessへ必須送出しない。公開の権限、Consent/Grant、現在価格、実注文Effect、売上改善をB0で宣言しない。

<!-- b2c-b1a-principal-scope-20261113 -->
## G49 B1a：独立Native Consumerに実データを渡す際の主体と対象範囲（2026-11-13）

**最初の対象を実Functionの意味に合わせる。** main [#4539](https://github.com/hat47x/tei-platform/pull/4539)は`order_entry.recent_orders`の実PG Read→Function Data Contextに限定し、main [#4540](https://github.com/hat47x/tei-platform/pull/4540)は申告されたNative PDPの`order-id`項目→Semantic Source一項目の束縛に限定する。両者を同じ実Frontend Functionとみなさない。B1aの最小候補は非公開の合成データによる**read-only注文履歴Consumer**とし、Function/Query Input/Result列・出典・用途・実Consumer受領の対応を[#2586](https://github.com/hat47x/tei-platform/issues/2586)の同一Attemptで検証する。商品詳細PDPを構築するときは、その目的に適した受理済み商品Data/Queryを別途必要とする。

**B2C利用者と組織所属を分ける。** 公開閲覧者・ログイン済み消費者・ゲスト購入者・業務運用者が、必ずしもOrg Membershipを持つとは限らない。組織モデルをB2C利用の必須条件にせず、案件が必要とする場合に限り[#2386](https://github.com/hat47x/tei-platform/issues/2386)へ結び付ける。認証方法/本人の出所は[#3412](https://github.com/hat47x/tei-platform/issues/3412)、Browser Session/要求保護は[#1043](https://github.com/hat47x/tei-platform/issues/1043)、読み取り用途・行/列Scopeは[#1656](https://github.com/hat47x/tei-platform/issues/1656)、現在の実行Authority/Effectは[#1174](https://github.com/hat47x/tei-platform/issues/1174)、用途別開示は[#2516](https://github.com/hat47x/tei-platform/issues/2516)を既存Ownerとする。TEI Coreに第二のUser/Identity/Tenant台帳を追加しない。**`:from`日時条件やFrontendが指定したcustomerIdは、利用者本人の注文を選別する認可根拠ではない。** Host側の実Principal/業務Subject対応とRead Grant/行範囲が確認できない注文データを一般向けAPIへ公開しない。

**順序・検証資格**：製品`tei`/既存Application経路から実Provider/Raw Resultを読み、独立Native Consumerが同じ値・Presence・Scope/Completeness・版を受領したことを確かめる。Public/Privateの実Hostアクセス、Guest/Checkout、Consent/Experiment、旧Clientの契約版と価格の再確認は後続の独立資格で検査する。既存B0-S1 [#4518](https://github.com/hat47x/tei-platform/issues/4518)を新しい公開APIの前提Gateにしない。独立Expected候補BC-43〜48と担当・停止線は[親Issueの引渡し](https://github.com/hat47x/tei-platform/issues/3509#issuecomment-6847438801)を参照し、別のテスト正本を作らない。

**状態**：B1a Product CLI→実Native Consumer、顧客単位の実Grant/Read、公開Browser、Commerce/売上効果の検証は**NOT RUN / NOT ACCEPTED**。Studio Desktopセルフビルド[#2831](https://github.com/hat47x/tei-platform/issues/2831)の最優先を変えず、Runtime-zeroを選べるHeadless/外部Frontend方式を維持する。

<!-- b2c-b1a-qualified-consumer-20261113 -->
### G49 B1aの受入を二段化する（2026-11-13）

**B1a-I（非公開・隔離した機能結線）**：最初は受理範囲が明示された合成注文履歴のread-only Consumerでよい。既存製品`tei` CLI／型付きApplication Operationから、選択した実Provider、実Queryと宣言済みの型付き入力、実Rowset、Function Data Contextを同一Attemptでつなぎ、**別プロセスのNative Consumer自身が受領した内容**を独立に確認する。DB Reader間の値一致、Reference CLIの成功、宣言Native FieldのSource束縛は、それぞれ異なる証拠であり合算しない。`order_entry.recent_orders`の`:from`は現状Query側の入力宣言を満たしたと認定されておらず、日時条件は本人の注文Scopeも与えない。Query入力・物理列・Function/Result Fieldの対応が未受理なら`unmapped/unsupported`として止める。

**B1a-P（一般消費者への公開資格）**：B1a-Iの成功とは別に、Hostが検証した認証Principal（Issuer/Subject/Session）と業務Customerの対応、対象注文との所有関係、現在Read Grant、行・列・用途別の許可済みProjection、Cache/旧Client/失効の整合を既存Ownerで確認する。公開閲覧者、ログイン済みConsumer、Guest、運用者は同じ主体・権限ではない。消費者に組織Membershipを一律要求せず、Cookie、Frontend指定customerId、メール一致、匿名ID、日時フィルターから権限を導かない。Guest→登録済みConsumerの履歴結合も、選択したCommerce/Hostの明示的なLink・許可なしに行わない。認証だけでは開示資格にならず、診断用ContextのConnection/Provider等をNativeレスポンスやログへそのまま流さない。

**表示・Cacheの受入**：`NULL`／未取得／非開示／RLS・Mask／0行／partial／completeをConsumerまで区別し、Claimに必要な完全性が確立していないとき「全注文を表示」「購入履歴なし」を断定しない。公開資格を確認できない場合は注文内容を返さず、画面側にも無根拠な「0件」成功を表示させない。本人専用レスポンスを匿名CDN/共有Cacheへ混在させず、Client・SSR・Edgeで版、Scope、失効を別々に検証する。実験割当・分析識別子は認証Subject/業務Customerと自動統合せず、Consent/目的・保持条件を別途扱う。

**受領票と停止線**：`Attempt / Source・Function・Query revision / typed parameter declaration / selected→actual provider / Rowset・Context presence / Scope・Completeness / Native receipt / Current Grant / allowed disclosure / cache・release / independent expected / status`を、観測済み・未観測・非開示ごとに分けて記録する。BC-43〜54の独立Expected候補は[親#3509](https://github.com/hat47x/tei-platform/issues/3509)から[#1180](https://github.com/hat47x/tei-platform/issues/1180)/[#2514](https://github.com/hat47x/tei-platform/issues/2514)へ帰属し、新たなテスト正本は作らない。B1a-Iは[#2586](https://github.com/hat47x/tei-platform/issues/2586)の製品実Read・独立Consumer、B1a-Pは[#3412](https://github.com/hat47x/tei-platform/issues/3412)/[#1043](https://github.com/hat47x/tei-platform/issues/1043)/[#1656](https://github.com/hat47x/tei-platform/issues/1656)/[#2516](https://github.com/hat47x/tei-platform/issues/2516)のHost・認可・開示を利用する。H0 Native単独／H1任意の型付き契約／H2任意の実験Adapterの費用・実測比較は後続B3へ置き、B1a-PやB2C専用Pluginを[#2831](https://github.com/hat47x/tei-platform/issues/2831) Desktop Self-Buildの新Hard Gateにしない。**製品CLI→実Native受領、公開認可、Browser/Cache、Consent、Commerce/売上効果はNOT RUN / NOT ACCEPTED**。

<!-- b2c-crosslane-acceptance-20261114 -->
### G49 横断適合：変更・配布・利用者・実作用の境界（2026-11-14）

B1a-I/Pの二段受入を維持しつつ、独自Frontendを「TEIが制約しない」と認定するための比較対象を明示する。業務上の正本は受理済みSemantic/Function/Action、実行権限はHost/Providerに残し、Frontend・CMS・Edge・実験・検索等の実装先は案件が選ぶ。Journey/Experiment/Releaseは既存の型付きGraph・Operation・Artifact/Scenarioの**投影とAdapter**で扱い、B2C専用のIdentity台帳、Analytics正本、実行Host、Lifecycle/Graph Engineを増やさない。

| 境界・既存担当 | 同じ実Consumerで確認する最小の正負対照 | 判定上限 |
| --- | --- | --- |
| **B1a-Iの値・型・Consumer受領** [#2586](https://github.com/hat47x/tei-platform/issues/2586)/[#2497](https://github.com/hat47x/tei-platform/issues/2497) | 選択した実Providerと実Invoke、Queryの宣言されたTyped Param、Raw Rowset→一個のContext→別プロセスNative/標準表示の受領を対応付ける。NULL・欠損・0行、投影後の同値重複と業務Identity、金額の単位・精度を区別し、同型Fieldからの暗黙Mappingを拒否 | 現在の#4539実PG参照と#4540宣言Native Fieldは**別Attempt**。通常`tei` CLIと実Native受領は未実証 |
| **個人データの開示非干渉** [#1656](https://github.com/hat47x/tei-platform/issues/1656)/[#2516](https://github.com/hat47x/tei-platform/issues/2516)/[#1043](https://github.com/hat47x/tei-platform/issues/1043) | 同一公開資格・公開Sliceで、W0/W1の**非公開の注文行だけ**を変更。HTTP Body/metadata、件数・aggregate・sort/page、diagnostic・digest・ログ・Cache key・Tool応答に秘密由来の差が出ないことをConsumerまで比較。private注文はHost検証済みPrincipal↔Customerと現在Grant/行列/用途Projectionが揃ったときだけ返す | Sourceの参照可能性や`Disclosure.Rows/Columns`欄の存在、Reader成功は実Policy適用の証明ではない。比較していない媒体・Timing等は保証しない |
| **変更影響とAI編集の対象** [#4518](https://github.com/hat47x/tei-platform/issues/4518)/[#2582](https://github.com/hat47x/tei-platform/issues/2582)/[#4515](https://github.com/hat47x/tei-platform/issues/4515) | R0でのみ存在した旧依存辺を消したR1、Required Consumerと無関係な旧辺、C0文言内の実効価格/Consent変更を対照。必要上流は失効対象へ残し、無関係なRelease/Consumerは過剰失効させない。AIの旧Source/Pin、未保存Dirty Buffer、現在選択・実際のBuild入力を区別し、無音Applyしない | #4591は申告Graph局所、#4592は修正提案。実Source全Universe・Source読取り時点・Current Grantの証明に拡大しない |
| **配布・Plugin実消費と撤去可能性** [#1016](https://github.com/hat47x/tei-platform/issues/1016)/[#951](https://github.com/hat47x/tei-platform/issues/951)/[#2604](https://github.com/hat47x/tei-platform/issues/2604) | H0=Native単独、H1=任意の型付き契約/生成Adapter、H2=任意のExperiment Adapterを、同じFrontend・機能・変更で比較。Selected→実Artifact同梱→実Invoke→Consumer受領、未選択Plugin除去、旧Client/SSR/Edge/Service Worker/Cacheの残存と実公開版を別観測 | 「Runtime-zero」はTEI標準UI/常駐JS/Router/Gatewayを必須としない選択であり、外部API/認証/運用コストがゼロという意味ではない。完全静的Compositionは#1016後続P1 |
| **注文Effect・実験計測** [#1174](https://github.com/hat47x/tei-platform/issues/1174)/[#1129](https://github.com/hat47x/tei-platform/issues/1129)/[#1180](https://github.com/hat47x/tei-platform/issues/1180) | Click/Intent→Cart/Checkoutの外部Provider実Call→確定Outcome/Receipt→独立Readを分ける。Timeout/ACK喪失/Hookの`effectUnknown`・`lookup_first`を成功や安全な再送へ降格しない。Assignment≠Exposure≠Outcome、Consent/欠測/SRM/返品等のguardrailを別に観測 | B1b実Commerce、B2実験、B3 CVR/粗利益の因果主張は後段。現在のHook局所Draftを二重購入防止の実証にしない |

**実行順・費用比較**：①#2586の製品`tei`入口からB1a-Iの**同一Attempt実Consumer受領**を一件成立させる、②B1a-PのPublic/Private・W0/W1負例を既存Host/Projectionで実行する、③B2の旧配布版・Effect Unknown・Experimentを段階的に追加し、④#2018でH0/H1/H2の初期実装時間、変更時間、追加JS/Request/TTFB/CWV、誤検出・欠測、Provider/運用費用を同じ独立Expectedで比較する。計測していないCVR改善やRuntime overheadの不存在を主張しない。

**証拠管理・優先順位**：Source/Context/Consumer/Release/Grant/Effect/測定の観測時点と受領媒体を分け、同じID・digestでも現在性や許可を推測しない。非公開注文、Session、内部接続先、非公開Sourceの存在・件数・HashをGitHub/外部AIへ必須送出しない。独立Expectedは[#1180](https://github.com/hat47x/tei-platform/issues/1180)/[#2514](https://github.com/hat47x/tei-platform/issues/2514)の既存Scenarioへ返し、新しいCase台帳は作らない。#2831/#4583のStudio Desktop Self-Build（B1 BuildIntent→B2 Stage1再入→B3 Stage2実Windows受領）を全体P0に据えたまま、B2Cは並行する異質Consumer試験とする。**本節は設計・検証計画であり、各実Target・公開認可・Commerce/Experimentの新しい実行結果を含まない**。

<!-- b2c-b1ai-executable-handoff-20261114 -->
### G49 B1a-I 最初の実行可能な一往復：実装受渡しと停止条件（2026-11-14）

**固定する対象**：`order_entry.recent_orders`の意味に合う、**非公開・使い捨て合成注文履歴のread-only表示**だけを選ぶ。商品詳細PDP用の`native-pdp.order-id`申告Binding（main #4540）はこのQueryへのFunction/Result Mappingではない。注文を一般ユーザーへ公開しない。現在の`:from`は実行用のtyped値を渡せても、Query Source側の`input`宣言・意味束縛が未受理なので、宣言不足を暗黙defaultで埋めない。

| 一Attempt内の検証箇所（既存Owner） | 最小の実Witness | 不合格・不明の場合の扱い |
| --- | --- | --- |
| **A. 対象と入力**（#2586/#4518） | 同一Workspace Load、実Canonical Function/Query Subject、Source occurrenceと版、`:from`のQuery宣言・型・Runtime値・物理SQL選択を照合する | Subject不在を参照0件と扱わず、Queryのinput未宣言ならTyped Contractは`unmapped`。製品`source-index`の不存在拒否はmain [#4584](https://github.com/hat47x/tei-platform/pull/4584)で修正済みだが、Function→Queryの意味受理は別 |
| **B. 製品入口と実Provider**（#2586/#1251/#951） | 通常の実`tei`プロセスまたは既存Product Application Entry、選択Provider/Instanceと**実Call**、実SQL/param shape、物理DB/Session/RLS・Reader時点の観測を区別する | `tools/reference-browse-read`のみ、Provider B選択→A実Call、別物理DBで偶然同値なら製品接続PASSにしない |
| **C. 一つのResult/Context**（#2506/#2497） | 同じ実Rowset→同じContext identity/epoch/Result Sliceから、列Attribute、Presence（missing/NULL/value）、Decimal scale、Grain、Scope、Coverage、完受（EOF/End marker）を保持する | DB cursor完走でも業務母集団/RLSの`complete`は導かない。masked NULL/SQL `NULL AS`/未取得を合算しない |
| **D. 独立Consumerの実受領**（#2586/#2516） | 同じContextを標準表示と独立Native read-only表示の**二Receiver**へ渡し、受信側でAttribute ID・行/Cell・Result revision/ScopeをReadback。Nativeへは事前に許可した投影だけを渡す | 二度のDB Read一致、DB Readerだけの照合、Reference JSON全項目のNative返却はConsumer資格としない。Connection/User/内部Instanceや非公開存在情報を出さない |
| **E. 独立Expectedと結果票**（#1180/#2514/#1163） | 事前固定の正常3行/期間境界/0行、列欠落・NULL/RLS、Provider不一致、Response途中切断を既存Scenarioで比較し、`declared / selected / invoked / consumed / verified`を分けて記録する | 一部局所PASSをB1a-I全体PASSにしない。未実行はNOT RUN、契約不支持はUNSUPPORTED、未確立はUNKNOWNのまま保持 |

**対象外を明示**：ここではPublic APIの認証、本人の顧客ID連携、現在Read Grant、共有CDN/Browser Cache、安全なGuest→会員結合、Cart/Checkout、公開Release/Experiment、CVRを合格とみなさない。B1a-Pでは、許可済みAudience/ScopeのPublic Sliceを先に確定してから、W0/W1で非公開注文だけを変え、Body/DOM/ARIA/metadata/count/sort/aggregate/diagnostic/log/cacheの許可外差分を実Host・Consumerで確認する。情報を伏せたことによる存在・件数の漏えいにも注意するが、Timingや外部経路を未測定のまま非干渉完了とは主張しない。

**変更・配布と次段の接続**：B0-RのR0-only辺／R1 Required上流は[#4592](https://github.com/hat47x/tei-platform/pull/4592)（未統合候補）と#2582へ、Source Root実在性が構造Fingerprint不変でも変わる負例は[#4616](https://github.com/hat47x/tei-platform/pull/4616)の限定提案へ接続する。Desktop側の[#4587](https://github.com/hat47x/tei-platform/pull/4587)ではTEI実CLI `self-build` primitive子起動→Inspector TS→実Windows WebView2まで限定成功したが、正式BuildIntentとStage1→Stage2は未実証。この成功をB1a-IのProduct Read/Native Consumer実証に合算しない。H0/H1/H2の実際の時間・通信/JS・運用費用比較は#2018で別々に採点する。**新CLI名・Host・Identity/Analytics/Commerce正本・Scenario台帳・Hard Gateは作らない**。

<!-- b2c-bridge-entry-temporal-20261114 -->
### G49の実装境界：Query Read・公開Entry・履歴と現在資格を分離する（2026-11-14）

**B1a-I最初の不足辺は公開Functionの新設ではない。** 現行の[Script Launch §12〜13 Draft #4618](https://github.com/hat47x/tei-platform/pull/4618)のSource照合では、製品`cmd/tei/cli_commands.go`に公開Function起動Verbはまだない。既存`semantic/capabilityinvoke.BusinessResult`の単一Output/Outcome区分はあるが、Outputの型/NULL/単位を自動的に全検証するわけではなく、`command.Result`は業務Command/Effect側の契約である。したがって、Read-onlyなB1a-Iではまず#2586の**既存Product Query/Function Data Contextの型付きOperation**を実Providerへ結び、その実Resultを検査する。次に、合意したCLI契約の**薄いAdapter**から同じOperationを呼ぶ。公開Business Entryの発見・発行、pure Functionの正式起動、Command/Effectは[#4495](https://github.com/hat47x/tei-platform/issues/4495)/[#3422](https://github.com/hat47x/tei-platform/issues/3422)の別資格であり、B1a-Iへ公開Entry/Action実行エンジンを偽装・新設しない。

**A〜Eの前段差分と独立Expected**：①現行`order_entry.recent_orders`の`:from`をQuery Sourceで型付き宣言できないなら`unmapped`を維持し、実行時paramだけから公開Operationと認定しない。②同じResult bodyを返しても、選択Provider B→実Invoke A、別物理DB・別RLS Session、欠落したCompletion/Consumer Readbackは不合格。③`source-index`のCanonical存在成功とStudioのPinが成立していても、公開Entry・Provider・現在Grant・実Consumerは未検証。④実ReadのContextは、`Connection.User/Database/Instance`等の内部診断を含むため、Native用の許可済みProjectionだけを配信する。これらは#2586/#1251/#1180/#2514に既存の判別票として引き渡し、新CLI Verb/Schema/Runner/正本をここで確定しない。

**履歴と現在時点の分離**：[組織C1二時点補遺 Draft #4622](https://github.com/hat47x/tei-platform/pull/4622)の`validAt/knownAt/observedAt/effectAt`区分を、B2Cでは必要な場合にだけ採用する。注文履歴の`ordered_at >= :from`は検索期間条件であり、**購入時点の商品価格・注文確定Fact、後日の訂正/返品・現時点の商品価格、現在の本人Read Grant/顧客Link**を保証しない。業務上の履歴値は当該Source/Providerの履歴・訂正契約に従い、Grant/Disclosure/Sessionは**閲覧時**に再評価する。W0（同じ注文内容、現在Grantあり）とW1（同じ注文内容、閲覧前にGrant失効）、または履歴価格同一・現在価格のみ変更を独立対照とし、0行・同byte・以前のReceiptを「購入なし」「現在許可」「現在価格」に昇格させない。全B2Cに二時点テーブルや組織Membershipを強制しない。実Host/DB・販売履歴の二時点試験はNOT RUN。

**Source現在性と配布の独立実証**：[main統合 #4616](https://github.com/hat47x/tei-platform/pull/4616)はWorld/Model revisionを変えずに未モデル化Source Rootの実在が変わり、Catalog/Map/Locate/Validateの判定が変化する局所Go試験を通した。これをNativeの現在Source/Validation再確認へ引き継ぐが、実Source全依存閉包、原子的Snapshot、現在Grantは未証明。Studio側は[#4623](https://github.com/hat47x/tei-platform/pull/4623)でCaller Pinに加え**Opt-inのResolved Builder→起動前Digest照合**を提案し、[#4624](https://github.com/hat47x/tei-platform/pull/4624)で宣言とResolver応答の不一致拒否を局所Goテストで検証した。ただし両PRはDraft・未mainで、B2C Frontendの正式BuildIntent/ArtifactSet、OS Loader TOCTOU、Stage1→Stage2、実B2C Provider受領の保証ではない。全体P0は[#2831](https://github.com/hat47x/tei-platform/issues/2831)/[#4583](https://github.com/hat47x/tei-platform/issues/4583)のB1→B2→B3を維持する。

<!-- b2c-h1-distribution-consumer-witness-20261114 -->
### G49 H0/H1/H2：実成果物・Consumer受領・変更後資格の判定票（2026-11-14）

**B1a-Iの後にH1へ追加するのはビルド時の任意契約であり、TEI標準RendererやBrowser Runtimeではない。** H0 Native単独はこの追加受入の対象外とし、H1/ H2の適用先だけで、案件が選んだContract/Provider/Builder/Memberと実際の生成・配布・受領を同一対象・版・用途で比較する。Plugin管理InventoryやBuilder Resolverの「解決済み」、署名済みReleaseの「検証済み」だけでは、公開画面・注文Effect・現在Grantの資格にならない。

| 検証対象 | 必須の正負対照／不足の場合の停止線 | 現在の実証と担当 |
| --- | --- | --- |
| **H1-A 選択と実Build** | Application/Target Profileが選んだContract・Provider/Builder Aに対して、Resolver応答B・実子image C・必要Artifact Memberを分離。同digest/同値でもID/契約/Target不一致、必須依存欠落は拒否。未選択Pluginの追加・非選択は既存H0成果物と契約選択を勝手に変更しない | [#4624](https://github.com/hat47x/tei-platform/pull/4624)はBuildIntent宣言対Resolverの局所修正（Goテスト確認・Draft）、[#4623](https://github.com/hat47x/tei-platform/pull/4623)は実子Builderの起動前pinと選択Bindingの候補（Draft）。B2C実TargetはNOT RUN。#951/#1016/#1080/#1097を再利用 |
| **H1-B 外部Releaseの独立受領** | H1の生成Member/契約版R1を選んだら、Consumerが実取得したRelease Envelopeと各Member Body/Hash/Targetを独立照合。署名R1なのにBody R0、欠落Member、旧Client/SSR/Edge/Cacheが版混在の場合は**そのReleaseの受領資格なし**。署名確認だけで現在配信・撤去・信頼失効を証明しない | [#4617](https://github.com/hat47x/tei-platform/pull/4617)は別Node ProcessのHTTP署名Release＋実Body取得の限定ReferenceでE1専用CI成功。Draft・固定鍵/localhostなので本番Release/Origin/Trust/CDNはNOT RUN。#2589/#2604/#2516へ |
| **H1-C Nativeの閲覧と編集** | Native FieldのQualified Subject/Sourceと、Data Contextの実Row Identity/Version/Grainは別。表示できても、Projectionがnon-injective・集計・多重Edge・Slot不明ならNative/AIからの自動Source/DB逆編集は禁止。明示Slotだけを変更する場合も現在Source span・OwnerのWriter資格・業務Operationの独立検査を要求 | [#4619](https://github.com/hat47x/tei-platform/pull/4619)は同一製品計画ファイルに別DraftとしてSlot相対のread/edit適格性を提案。#2497/#1124/#2500/#2514がOwner。B2C実Edit/WriterはNOT RUN |
| **B1a-P/B2 現在認可・切替** | W0とW1で注文値同一・Grantだけ失効、または旧Client/R1・新Release/R2・変更後の現在価格/Consentを比較。表示済みの値、Cache、Variant Assignment、Checkout Intentは現在Grantや確定注文/Exposure/Outcomeを代行しない。Plugin/Provider旧世代in-flightと新世代選択、Effect Unknownでは再送前Receipt/現在権限を別照会 | #1043/#1656/#2516（Read/Disclosure）、#1174（Effect/Receipt）、#2044/#1223（Plugin更新）、#2604（Release）、#3509（B2C）。公開Host/切替/Commerce/Experimentの実E2EはNOT RUN |

**評価と作業順**：B1a-I（#2586）を先に実Provider/同一Context/独立Native受領で成立させ、H1-A〜Cは選択して採用したTarget/Pluginに限り局所追加する。B1a-Pの現在Read Grant/W0W1は独立した公開資格、B2の実験はAssignment/Exposure/Consent/Outcomeを別に試す。#2018では同じ変更シナリオをH0 Nativeのみ・H1選択Contract・H2選択Experiment Adapterに適用し、正しい結果・Build/更新時間・JS/通信/Cache・レビュー工数と実費を比較する。H1/H2の自動採用・売上向上を事前に主張しない。新配布Host・Builder/Policy/Graph Engine・Analytics/Identity台帳・Test SSOTを設けない。**この節は既存DraftとReferenceを横断した受入設計であり、新規B2C実行・製品採用はNOT RUN**。

<!-- b2c-b1ai-first-missing-edge-20261114 -->
### G49実装引渡し：B1a-Iの最初の欠落辺を一つずつ埋める（2026-11-14）

**先に未成立の一辺を特定する。** main [#4539](https://github.com/hat47x/tei-platform/pull/4539)のEnvironment→実PG Read→Function Data Contextと、main [#4540](https://github.com/hat47x/tei-platform/pull/4540)の申告Native Field→Source occurrenceは**別の契約・別Attempt**。製品CLI未結線、Query `:from`の`declaredByQuery=false`、Product Function/Query/Resultの正式対応、二Receiverの実受領という欠落を、変更前の固定Workspace/Source/Contract版で区別する。すべてを一PRで修正することを要件にしない。

| 実装Slice（既存Owner） | 最初に固定する最小の実成果と独立負例 | 合格上限・次への引渡し |
| --- | --- | --- |
| **R-A 型付きProduct Read Operation**（#2586/#2506/#2622） | 既存Sourceの`order_entry.recent_orders`と実SQL/`:from`の**宣言・型・Runtime value**を照合し、実Provider選択→実Invoke→一つのFunction Contextの行/列/Scope/PresenceをProduct経路で観測。必要なQuery `input`宣言がなければOwnerへ不足箇所を戻し、暗黙の文字列paramで適合扱いしない。別SQL/別DB・RLS role・Provider A/B取り違えを負例にする | **このSliceだけの局所Product Read**。実ContextであってもCurrent Grant、全母集団、Native受領、正式Business Entry公開は未合格 |
| **R-B 製品CLIへの薄い接続**（#2586/#3422/#1251） | R-Aと**同じProduct Operation**を、通常`tei`実バイナリの子プロセスから呼び、Source/contract/実Provider/Context/CLI exit・stdoutを区別してReadback。正式VerbやWireはCLI Owner判断後に決める。未知Verb/未公開対象/不正入力は副作用・秘密stdoutなしで失敗とする | [Draft #4625](https://github.com/hat47x/tei-platform/pull/4625)は既存`operations.ExternalCandidates`をRead-only CLIへ薄く投影した**別領域の参考実装**。ローカルREA Evidence PathやTool Present、実Process成功をB2C PG Read/現在認可の証拠へ流用しない |
| **R-C 同一Contextから独立Receiverへ**（#2586/#2497/#2516/#3509） | **一度だけ**実Query→Contextを生成し、同じResult/Context identity・epoch・Source/Query/Provider版の許可Projectionを標準表示Aと独立Native read-only表示Bへ配送。受領側プロセスが双方でAttribute ID、セルのvalue/NULL/missing/withheld、行数・grain・Scope/partial/unsupportedを確認する。別々にPGを2回読んで同じ数値を得る方式やPublisher receiptだけでは不可 | Aだけ成功・Bへの配送途中切断・旧Context混入・内部`Connection.User/Database/Instance`のWire漏えい・`COUNT`や`SUM`の未宣言精度/NULLを独立Expectedで拒否。Context ID一致だけでは同一原子的DB SnapshotもGrant受理も保証しない |
| **R-D B1a-Pを別票で資格化**（#1043/#1656/#2516/#1174） | 公開Hostが検証した現在Principal↔Customer/Order、Read Grant、Audience/用途別Projection、Private Cache partitionと失効を確認。注文値同一でGrantだけ失効したW0/W1、別Customerの注文増減、旧Client/ReleaseのCacheを負例にする | R-A〜CがPASSでも非公開合成Readerの資格のみ。自動的なBrowser公開、Guest Link、Checkout Effect、Consent/Exposure/CVRは未合格 |

**選択・配布の横断Witness**：今後H1契約を選択した案件では、[Draft #4624](https://github.com/hat47x/tei-platform/pull/4624)のBuildIntent宣言⇄Resolver整合と、[Draft #4623](https://github.com/hat47x/tei-platform/pull/4623)の**実ResolveしたBuilder digest⇄起動前Child image**を別々に照合する。2026-11-14確認時点で、#4623 exact HEAD `f1e219a`はGitHub Actions 8件すべてsuccess、#4624 exact HEAD `2d403f4`は取得できた5件すべてsuccess。ただし**どちらもDraft・未main、Plugin主体の限定BuildPlan／非Plugin Studio全工程は未受入**。Builder一致は成果物のメンバー実同梱・実配信・Runtime受領でもない。[Draft #4617](https://github.com/hat47x/tei-platform/pull/4617)のtest-only HTTP署名Release/Node Consumer検証とも、同じB2C実Releaseだと合算しない。H0 Native単独にはこれらを必須化しない。

**用途と写像の交差検収**：並行[Draft #4619](https://github.com/hat47x/tei-platform/pull/4619)のD0〜D3、P/J/Q（局所観測・異種Evidence Join・問い相対閉包）を、このR-Cでは**読み取る側の必要条件**として参照する。表・集計・Graphで同じ表示が成立しても、JOIN fanout/重複投影、同Label別Identity、non-injective Map、非表示列や否定条件のUnknownを消さない。表示・閲覧適格性、Slot局所編集候補、現在Writer/Effect Grantは別。#4619のG08等の原文は**同一ファイルを編集する別Draft**に残し、このG49節はB2C実受領への引渡しだけを所有する。採用時は両方の最新main差分を照合し、内容を欠落・二重化しない。

**最小の証拠票と停止条件**：既存#1180/#2514/#1163で`Exact HEAD / Workspace・Source・Function・Query Revision / typed-input declared? / selected→invoked Provider・DB Session / actual Rowset+Context ID・epoch・Scope・Coverage / A,B各Consumer delivered・received Body / allowed Projection・Grant qualification / independent Expected / PASS|FAIL|NOT RUN|UNSUPPORTED|UNKNOWN`を別列管理する。Consumerへ送らない内部接続・個人情報や非公開Source ID/件数/ダイジェストをGitHub/外部Harnessへ必須送信しない。最初に不足した辺を修正し、局所成功を次Sliceの成功へ昇格させない。**この節自体はdocs-only、R-A〜Dの新実行とOwner意味受理はNOT RUN**。

<!-- b2c-ra-source-to-executor-failclosed-20261114 -->
### G49 R-AのCode適合ギャップ：既存Query.Inputsを実Provider呼出し前に束縛する（2026-11-14）

**今回のmain Source静的照合（成果の上限：実装着手点の確定）**：
- `internal/query/model.go`に既存`query.Input {Name core.LocalName, Type core.CanonicalID}`と`query.EmbeddedSQL.Inputs []Input`がある。したがってQueryの型付き引数を表す**新しい公開Model/Core型は不要**。
- `reference/workspace/applications/order-entry/queries/recent_orders.tei`には`query order_entry.recent_orders`、`datasource sales`、SQL source・Result mappingがあるが、**`input from`の宣言がない**。同ディレクトリの`recent_orders.sql`は`WHERE o.ordered_at >= :from`を使用する。Reference WorldのDriver testが通ることと、Query入力意味受理は異なる。
- `internal/driver/browse_query_environment.go`の`ExecuteBrowseQueryInEnvironment`は、Environment/Instance/ProviderとExecutorの照合後に、`map[string]any`を`r.ExecuteBrowseQuery`へ渡す。**Query.Inputsの必須性/型/未知パラメータを、この入口で実Executor呼出し前に検査していない**。Registry側の物理接続が同一Instanceかはhost-assertedで、実Read Scope/現在Grantも別資格。
- `tools/reference-browse-read/main.go`の`parseParams`はCLI引数`from=time:RFC3339`を`time.Time`にする一方、`DeclaredByQuery`を**SQL/Executor実行後**のContext構築前に照合する。したがって参照CLIのtyped tag成功、実PG3行取得、`declaredByQuery=false`をProduct契約適合へ昇格しない。

**Owner #2586/#2622/#3422への実装PR分割案（新Command/APIの採用ではない）**：
1. **RA-0 Source/Parser/型選択**：現行Query DSL Loaderが受け付ける`input`構文とCanonical time型を、既存Reader・Role/型Contractから確認する。採用済み構文・意味へ沿って`recent_orders.tei`へ`:from`を宣言し、再Loadした`Query.Inputs`に**名前・型・Source出典**が実際に反映されることをGo独立Expectedで固定。見た目だけの新DSLや、SQL placeholderからの推測による暗黙宣言はしない。型・構文が未受理なら`UNMAPPED`で止める。
2. **RA-1 Provider前の強制検査**：同じSource/Queryと実Invocationの引数セットを比較し、必須`:from`欠落、型違い、未宣言名、explicit NULL/未取得の不許可、時刻のoffset/precision変換、別Query同名を独立負例にする。**拒否時は既存Counting Executorの実Call 0、Result/成功Receiptなし**。Mapの重複キー表現不可はCLI raw flags段の既存重複拒否と区別する。互換な既存Query/DSLへ一律のBreaking変更を加える前に対象Scope/ContractをOwnerが決める。
3. **RA-2 一Attemptの実Value**：宣言成功かつType適合後だけ、既存#4539の隔離PostgreSQL read-only/REPEATABLE READ Fixtureに実引数を渡し、O-003/O-002/O-001の全3行・10:05境界2行・後方0行、Decimal scale2、Result Scope/0行の`unestablished`を独立Expectedへ照合。`selected provider`・実Executor・SQL byte/params・物理DB/Session/RLSとFunction Contextを同じ実Attemptで記録。選択と実Call一致なし/違うDBで偶然同じ行はFAIL。
4. **RA-3 R-B/R-Cへ引渡し**：受理済み型付きOperationのResultを薄い製品`tei`入口へ渡した後、**同一Context**を標準/Native二Receiverへ配信。Draft [#4625](https://github.com/hat47x/tei-platform/pull/4625)の`external-candidates`は、既存Operation→実CLI childの別領域の**構造参照例のみ**。その実CLIが起動できてもB2C実PGやNative受領ではない。

**Readと更新の境界**：同じ注文が3行返った事実、`:from`の時刻、ProviderのSessionを、全顧客の母集団・本人所有・現在Read Grant・注文確定Effectには読み替えない。非公開合成Readの単一Contextを、必要なAudience/用途の投影を通してからNative Consumerへ渡す。動的依存/Dirty保持/Source A→B→A、二Consumer・二Provider snapshotのCoherenceは移植したG08〜G13データ様式節の#2582/#2597/#1708へ委譲し、B2C用Reactive Engine/中央Snapshot Coordinatorを作らない。

**現行の実コードと受入資格（2026-11-14）**：RA-0 [Draft #4626](https://github.com/hat47x/tei-platform/pull/4626) HEAD `9eecf2b9`は、既存構文`input from std.datetime`を採用し、現在のWorld/Golden/Fingerprintを更新した。歴史RunnerではSHA-256固定の旧Queryだけを隔離Frozen Worktree内で再現する。**同HEADのHealth・歴史Self-buildを含め18 CI中17成功**、API実成果物の`sqlgen`依存検査1件は[#1182](https://github.com/hat47x/tei-platform/issues/1182)の別不合格であり、PR全体は未受入。RA-1 [Draft #4631](https://github.com/hat47x/tei-platform/pull/4631)は既存`Resolver.ExecuteBrowseQuery`で宣言済み入力と実Go値をExecutor前に照合する。`std.datetime`→`time.Time`のみを適格として、欠落・NULL・文字列・未知キー等は実Call 0で拒否し、歴史的に入力宣言のない合成Replayを型付きProduct資格へ昇格させない。**修正RA-0をbaseとした統合[Draft #4644](https://github.com/hat47x/tei-platform/pull/4644) HEAD `b5c23044`では、RA-1のGo/旧Web回帰、RA-2Rの空2 DBを使った実PG16参照Reader親＋7ケース非SKIP、設計検査の3 Workflowがすべて成功**。ただし通常製品`tei` CLIから選択Provider/実Connection/SQL/Params/Rowsetを同一Attemptで受けるRA-2P、標準・独立Native Consumer二者のRA-3、現在Principal/Host GrantのB1a-Pは未実証である。これらのDraftはmain未統合。Stage1→Stage2 Studio Desktopセルフビルド#2831/#4583が全体P0、Product Read#2586が並行P0。

<!-- b2c-ra0-historical-revision-ci-20261114 -->
**Source改訂が他Targetへ波及したときの資格（実CI由来）**：[RA-0 Draft #4626](https://github.com/hat47x/tei-platform/pull/4626)の初回HEAD `1b12c833`では、固定Web Targetの歴史Source受入Fixtureが、新しいQuery InputによりWorld Revision不一致を検出して停止した。これは単なるノイズではなく、**今のSourceが変更された証拠**である。元の受入Fixtureは現在Sourceで自動再受理せず、別のScratch内だけで旧Query byte SHAを照合して復元する変更をHEAD `4f238139`へ追加した。この隔離テストが成功しても、新しいQuery Inputを持つ現行SourceのOwner意味受理・Web/G1資格は得られない。また別Workflowで`tei/internal/database/sqlgen`の予期しないAPIバイナリリンクが見つかった。Query宣言の変更が原因と決めつけず、実API成果物の依存検査Ownerが独立調査し、Allowlist緩和で隠さない。**#4626最新HEAD `4f238139`では歴史Fixture差分の2つのWeb/固定World Workflowは再実行でsuccess**。一方、実APIバイナリの`go tool nm`で`tei/internal/database/sqlgen`が2 symbol出る[E1依存除外failure](https://github.com/hat47x/tei-platform/issues/1182#issuecomment-6869938385)は継続し、セルフビルド系もそのHEADで失敗。Browser Asset側の走査PASSと、E1全体の成功を混同しない。Go/Windows/PGの他WorkflowやRA-1の最新CIを別に確認するまで**PR全体CI PASSは保留**。


<!-- g49-ra1-driver-studio-server-witness-20261114 -->
**Studio Server→B2C公開Hostの連携上限**：[Draft #4627](https://github.com/hat47x/tei-platform/pull/4627)は現行Wails Server Bridgeのexported 15 methodを実Go型・ASTで棚卸しし、非ReadのCommand/外部Child Processも登録されていることを局所検査した。現時点のServerはloopback PoCであり、B1a-Pの公開Host/複数Actor/Workspace隔離/現在Read Grantの実資格ではない。将来のSaaSで必要なread-only façadeと許可済みProjection、WebSocket受信・配信の認可は[#1824](https://github.com/hat47x/tei-platform/issues/1824)/[#3421](https://github.com/hat47x/tei-platform/issues/3421)/[#1043](https://github.com/hat47x/tei-platform/issues/1043)の責務に置く。RA-0/RA-1やStudioセルフビルドのHard Gateへ多利用者Webを追加しない。

<!-- b2c-ra1-gateb-studio-n2n3-capability-preflight-20261114 -->
### G49 横断の実行前拒否とDesktop使用実証を区別する（2026-11-14、実コード・実CI照合）

**共通する設計上の規則は「実実行者へ渡す前に、選択済み契約に照らして拒否できること」であり、すべての操作を単一Runnerへ集約することではない。** 対象・Authority・受入票を次のように区別する。

| 実行境界／Owner | 今回照合した観測 | Product採用へ残る条件 |
| --- | --- | --- |
| **Browse Query入力 #2586** | Draft [#4626](https://github.com/hat47x/tei-platform/pull/4626)の`input from std.datetime`に、積上げDraft [#4631](https://github.com/hat47x/tei-platform/pull/4631)で`Query.Inputs`対`time.Time`をExecutor前照合。最初の実Go CIはGate Bの`:from`欠落を検出してFAILし、Scenarioを修正。旧R0の合成Replayは型付き資格を持たない | 最新HEADのGo/旧Web PASS、Environment選択→実Provider→Function Contextの同Attempt、未宣言/型違い/NULLの実Call 0、通常`tei` CLI・独立Native Receiver |
| **Capability Result契約 #4495/#3422** | [PR #4629](https://github.com/hat47x/tei-platform/pull/4629)は**main統合済み**。不正なResult contractを`capabilityinvoke`の実Adapter前で拒否し、不正6例でCall 0。事前Context cancelと正しいBusiness Outcomeを維持。Exact PR HEADでは9 Workflow success | 公開Entry/選択・実呼出し一致、Owner受理、Current Grant、実Host Effect・Product CLIは**別資格**。Queryの`:from`判定もここへ移管しない |
| **Studio生成WebView2 #4583** | [Draft #4587](https://github.com/hat47x/tei-platform/pull/4587) HEAD `0fcb7b62`は同一HEADで12 Workflow success。WindowsのN2 esbuild出力（`workbench.bundle`）→N3 Go`EmbedFiles`のSHA・role・Attemptをpre/post照合し、変更byteを拒否。実WebView2配信も対応 | 正式`BuildIntent.Resolve`/Builder選択→実`ArtifactSet`、N0→N2接続、**Stage1 TEI生成→Stage2 Wails再構築・Windows実使用**。N2/N3のbyte一致をセルフビルド全体成功へ昇格しない |
| **Studio Server/Web公開 #1824/#3421/#1043** | [Draft #4627](https://github.com/hat47x/tei-platform/pull/4627)はWails ServerのService/公開メソッドを棚卸しし、CommandやSHO Child起動も存在することを確認 | Local loopback PoCと公共Webを分け、Read-only Facade、Actor/Workspaceごとの現在Grant、WS受信者別開示を別検証。Server従属プロセス＋Named Pipe/UDS案は性能・障害回復・配布の**実測前は方式未確定** |

**切替条件**：現在のSource/Query契約と実引数に合致すること、実Providerへ正しい引数が渡されたこと、Function ContextをConsumerが実受領したこと、公開Hostに現在Grantがあることは**4枚の別票**。Result contractの事前拒否やStudioのN2/N3 byte固定は各票を強めるが相互代替しない。#4629のmain成功・#4587のWindows限定成功を#4631の不合格Runへ加算しない。適格な集計Law/粒度は前節のD1.2、Studioは依然P0 #2831/#4583。新Host・Registry・Measure Engine・全工程を待つ巨大Gateは追加しない。

<!-- g49-ra2-reference-pg-and-shared-actual-consumption-20261114 -->
### G49 RA-2の実DB証拠：参照Reader 7ケース全PASSとProduct受入の欠落辺（2026-11-14）

**新たな実行結果**：[積上げDraft PR #4638](https://github.com/hat47x/tei-platform/pull/4638) HEAD `0ce3a52f070314741f72462982ff940a1f791a28`、base [RA-1 #4631](https://github.com/hat47x/tei-platform/pull/4631) HEAD `02af8339`。専用[実PostgreSQL Workflow #43553849861](https://github.com/hat47x/tei-platform/actions/runs/43553849861) は**completed/success（1/1 Job）**。実PR merge SHA `05ab66fae60aa349900c93f306c605d7a819e697`と実HEAD親を照合し、PostgreSQL 16の空DB 2つとReader Roleで、既存`TestBrowseReadAgainstDisposablePostgreSQL`の親・A/B/C/D/E/G/Fの全7 subtestを**非SKIP・全PASS**で実行。これは`tools/reference-browse-read`の実Connection/SQL/Rowsetを使った証拠で、手書き期待値と独立DB readerを維持。Reference用の第二Fixture正本は追加していない。

| 受入対象と資格 | 今回の実観測 | なお未成立の条件（既存Owner） |
| --- | --- | --- |
| **RA-0** Source宣言 | [#4626](https://github.com/hat47x/tei-platform/pull/4626)新HEAD `9eecf2b9`：`input from std.datetime`、Current World/Golden/Fingerprintと旧Frozen Query rawを分離し、旧HEAD `94876821`の未使用`reflect` importによるGo test/vet失敗も修正。旧HEAD `94876821`で16/18成功・HealthとAPI E1不合格は履歴票 | 同じ`9eecf2b9`の全体CI 18系統を[実Run](https://github.com/hat47x/tei-platform/pull/4626/checks)で再検査。E1は依然`sqlgen`のAPIリンク異常として[#1182](https://github.com/hat47x/tei-platform/issues/1182)に独立不合格。Current GoldenとFrozen Historyの意味受理・main採用・他レーンのCIはこの修正だけでは確定しない |
| **RA-1** Go typed preinvoke | [#4631](https://github.com/hat47x/tei-platform/pull/4631) HEAD `02af8339`で[Go回帰CI #43548961899](https://github.com/hat47x/tei-platform/actions/runs/43548961899) completed/success。Gate Bの欠落`:from`をScenarioの`time.Time`入力に直し、防護を緩めなかった | 実Environment-bound Product入口の毎回適用、全型/NULL契約、旧R0のuntyped ReferenceをProduct Readerへ通さない。最新main/RA-0更新後は**積上げ差分を再検証** |
| **RA-2 実PG参照** | [#4638](https://github.com/hat47x/tei-platform/pull/4638)は2 DB／RLS／Source変更・JOIN重複／0行／NULL／列権限／時刻境界を一つの隔離PG Attemptで観測。全7 Case success | **正式Product `tei` CLIから実Operation→実Provider→Context**のSame Attemptと、選択Provider A／実Call B負例、実Consumed SourceとSession Scope/epochの記録は未成立。参照CLIのpassをProduct E2Eへ昇格しない |
| **RA-3/本件R-C** 実受領 | 参照Contextは実読取の結果だが、Receiver間の同一Context deliveryには未接続 | **同一Context ID・epochを持つ標準Display A＋独立Native Bの各受領Body**、Source/Scope/NULL/partial/projection、Expected独立照合。AggregateはD1.2の別受入 |
| **B1a-P** Current Grant | 参照DB Read・RLS Roleの結果は確認できたが、Public Principal/Customer所有のAuthorityではない | Host current Grant、Audience Projection、失効W0/W1、Cache隔離とRelease/旧Clientの別票 |

<!-- b2c-ra0-current-golden-historical-byte-pin-20261114 -->
**RA-0 Source改訂をCurrent/Historyに分離した修正候補（2026-11-14）**：[Draft #4626](https://github.com/hat47x/tei-platform/pull/4626) HEAD `94876821f784d5fa9f71b15a05ac4156221b2bc8`で、旧HEADの[Repository Health #43544373270](https://github.com/hat47x/tei-platform/actions/runs/43544373270)および[セルフビルド #43544373386](https://github.com/hat47x/tei-platform/actions/runs/43544373386)で実際に失敗した契約を4ファイルで修正した。

| 修正箇所 | 観測済みの旧失敗→修正の意味 | 実装後の受入停止線 |
| --- | --- | --- |
| `examples/data-source-workbench/query_route_separation_test.go` | SQL `:from`は宣言なしという旧Expectedが、実Loader宣言後にも`want [from]`として残っていた。新Sourceには`input from std.datetime`を要求し、未宣言入力集合は空へ更新 | SQL字面の静的検知は実PG Parameter Binderの証明ではない。XTEA別Workspaceの`:status`をこのQueryへ混入しない |
| `reference/world.json` | Canonical current Worldの`order_entry.recent_orders`に`inputs:[{name:from,type:std.datetime}]`が欠け、`workspace.Load`とのWorld同一性でFAIL | 旧受理Worldを新Sourceと同一扱いしない。新Goldenは変更後Current Worldに限定 |
| `internal/workspace/load_listing_fingerprint_test.go` | 旧Fingerpint `61ebb5ca...`に対しGo実CIで`435b687e...`を観測。値の来歴とCurrent World意味改訂を注記し、測定済み新値を固定 | この文字列の更新単体はOwner意味受理や過去のSource履歴保証にならない。最新HEAD Loadを実Testで再検証 |
| `tools/reference-plugin-self-build/historical_runner_parity_e2e_test.go` | 旧RunnerのLoaderは新しい`input`行を解析できず実セルフビルドでFAIL。Frozen Worktreeだけに**実git commit `e888bb8f...`の旧Query raw bytes**を戻す。SHA-256 `a7c1c4f6...`と新Sourceとの差を独立検査し、Frozen Diffは旧Manifest 2件＋旧Query 1件の3パス限定 | **現WorkspaceのQueryから宣言を消さない**。旧RunnerがFrozen版を通過しても、現Worldの新Queryを受理したことにはしない |

**再検査**：[#4626最新HEADのCI](https://github.com/hat47x/tei-platform/pull/4626/checks)はexact HEAD `9eecf2b9`で**18/18 Workflow完了（17 success、1 failure）**。[全体Health #43558268996](https://github.com/hat47x/tei-platform/actions/runs/43558268996)と[歴史Self-build #43558269076](https://github.com/hat47x/tei-platform/actions/runs/43558269076)はsuccess。残るAPI実依存E1 failureは別Owner #1182であり、PR全体PASSではない。別[Browser→JSON API実経路](https://github.com/hat47x/tei-platform/actions/runs/43544373457)の旧HEAD失敗はAPIの`tei/internal/database/sqlgen`2シンボル検出（Browser byte自体の走査成功）で、RA-0による因果は不明。#1182に別の実依存調査を残し、許可リストを黙って拡張しない。新RA-0 HEADの合格を確認したあとRA-1 #4631・RA-2 #4638の**積上げbaseが更新された事実**を明記して新Checkoutで再検証する。旧HEADのRA-1 Go成功、RA-2実PG 7/7成功を新baseのPASSとして再利用しない。

<!-- g49-current-windows-e1c-buildintent-heads-20261114 -->
**他レーンの最新実態を受入順に同期（2026-11-14）**：
- **Desktopの実使用**：[Draft #4587](https://github.com/hat47x/tei-platform/pull/4587) はHEAD `9b58533d`で[Windows WebView2 Workflow #43572235468](https://github.com/hat47x/tei-platform/actions/runs/43572235468)の**通常Desktop・生成Desktopの両Jobが成功**し、同HEADの12系統のCIもすべて成功した。旧HEAD `357345d7`のOS版取得に伴うタイムアウトは、Pythonの`sys.getwindowsversion()`による測定へ変更して解消した。これは両Desktopの限定的な実WebView2検証であり、**Stage1のTEI StudioがStage2のStudio実行ファイルを生成し、その実行ファイルを利用した**というセルフビルド一往復の証明ではない。全体P0 #2831/#4583は維持する。
- **Builder選択／実Child pin**：[Draft #4634](https://github.com/hat47x/tei-platform/pull/4634) 最新HEAD `7d6a1238` は**8/8 Workflow completed/success**。BuildIntent宣言→Resolver応答／選択Builder→起動前Image SHAを同じCheckoutで検証したが、Plugin主体の限定BuildPlanであり、非Plugin Desktop `N0〜N3`・OS Loader read-time byte/外部依存閉包、Stage2生成Desktop完成ではない。
- **署名Release独立HTTP受領**：[PR #4617](https://github.com/hat47x/tei-platform/pull/4617)は**main統合済み**。外部Node Processがテスト用loopback HTTPから署名Releaseと2 Member bytesを取得し、正常1／拒否9を検査。実GitHub Releases/外部Assetの配布・現在Trust・Audience Grant/Availabilityとは別であり、[Draft #4640](https://github.com/hat47x/tei-platform/pull/4640)の外部Release readback（GET 200/302、混入、Issuer差、実Body digest）は設計上の次工程。G49 H0 NativeにPlugin/JS/外部Releaseを必須条件として追加しない。
- **Script Launch**：[PR #4639](https://github.com/hat47x/tei-platform/pull/4639)はmain統合済みの計画改訂で、`#4636`の宣言preflightを現行設計に反映。Query RA-0/RA-1/RA-2のProduct Read、実Go Provider Entry/Effect、Studio Stage2のQualified Passには流用しない。

<!-- g49-currentworld-integrated-ra012-and-provider-identity-20261114 -->
### G49 RA-0〜RA-2Rの一体再検収とRA-2Pの実Provider境界（2026-11-14、最新他レーン整合）

**修正後Current Worldで再検証するための独立Draft**：[PR #4644](https://github.com/hat47x/tei-platform/pull/4644) HEAD `b5c23044ebaf8671d73a140da92fc9c4d74edff4` は、[#4626 RA-0](https://github.com/hat47x/tei-platform/pull/4626) HEAD `9eecf2b9`を**base**とし、[#4631 RA-1](https://github.com/hat47x/tei-platform/pull/4631) のGo本体・検査と[#4638 RA-2R](https://github.com/hat47x/tei-platform/pull/4638) の実PG16 Workflowを同一checkoutへ組み合わせた。旧RA-1の変更前Go blobと新RA-0上のblobが5ファイルすべて一致していることを事前検査し、RA-1のGo実装とテストおよびGo CIは**byte同一**。RA-2R CIは本PRでも起動するため`pull_request.branches`にRA-0 branchを追加したのみで、試験値やExecutor/DB契約は変更していない。元DraftはOwner正本として保持し、本統合PRは競合解消・merge順決定の代行ではない。

| 同一HEADの検収票 | 現時点に存在する証拠 | 失効・未成立の条件 |
| --- | --- | --- |
| **RA-0 Current/History** | `9eecf2b9`の18 Workflowは実起動、exact HEADの18 Workflowは**17 completed/success、1 completed/failure、未完了0**。実[Health #43558268996](https://github.com/hat47x/tei-platform/actions/runs/43558268996)と[歴史セルフビルド #43558269076](https://github.com/hat47x/tei-platform/actions/runs/43558269076)が両方success。旧 `94876821` は未使用`reflect`でHealth失敗、新HEADで除去済み | Current World/Golden/Fingerprintと歴史Frozen Runnerの**限定Go/実セルフビルド回帰は成功**。ただしBrowser→JSON APIの`sqlgen`2-symbol E1 failureは[#1182](https://github.com/hat47x/tei-platform/issues/1182)の別Ownerで、**PR全体は不合格・Draft未main**。Health合格を業務Owner Meaning/Current Grantへ昇格しない |
| **RA-0/1同一checkout** | [統合Draft #4644](https://github.com/hat47x/tei-platform/pull/4644)のRA-1 Go／旧Web回帰 Workflow [#43558633790](https://github.com/hat47x/tei-platform/actions/runs/43558633790)が**completed/success**。統合PRの3 Workflowは**同一HEADで3/3 completed/success**（RA-1 Go回帰、RA-2実PG、Issue-to-Design）。Go Driver/Query/Syntax/Validation・旧Web Target・参照Readerと`cmd/tei`の既存Goテスト・`go vet`をPR merge-to-head/clean照合つきで完走 | 旧#4631の[成功Run #43548961899](https://github.com/hat47x/tei-platform/actions/runs/43548961899)は**旧RA-0 base**の限定Go票。新仮統合HEADで`Query.Inputs`／Actual `time.Time`の正常1 Call・不正0 Callを再検証し、仮統合PASSをmain merge合格にしない |
| **RA-2R実PG16同一checkout** | 新Draftの[PostgreSQL Workflow #43558633773](https://github.com/hat47x/tei-platform/actions/runs/43558633773)は**completed/success、空2 DB/RLS/親＋7 subtestが非SKIP・全PASS**。仮統合SHA `fbf8fe55b92f2537cd4197e3c81df077d6f9ba1c`とPR HEAD `b5c23044...`を実Logで照合した。旧[#4638の実Run #43553849861](https://github.com/hat47x/tei-platform/actions/runs/43553849861)では隔離2 DB/RLS/7 subtest全PASS・非SKIP | **修正RA-0を含む同一Checkoutで実PG全7ケース非SKIP成功を確認済み**。ただし本当の通常製品`tei` CLIからのQuery Provider実選択・実呼出しではなく、**参照Readerのみ**の資格 |
| **RA-2P Product Operation** | 既存`ExecuteBrowseQueryInEnvironment`／`BrowseQueryExecutorMap`を利用できる。Provider・Instance選択などの**宣言値**と、実DBが返したByte/Rowsetは区別可能な受入票を作れる | **通常`tei` CLI→実選択Provider→実DB/Role/SQL/Params/Rowset→Function Data Context**の同一Attemptは未実装・未検証。Context実二ConsumerはRA-3、現在GrantはB1a-Pの別資格 |

**RA-2Pの実装を進める前の所有境界（データ様式Draft #4619の最新検討を採用）**：`DatabaseProviderBoundBrowseQueryExecutor.DatabaseProvider()`はExecutorの**自己申告**、`BrowseQueryEnvironmentExecution.Instance`は宣言側の束縛、`QuerySourceSHA256`は渡したQuery Source byteのDigestであり、**実Connection先PG16のDatabase/Role・実消費SQL/Params・Session/Scopeを保証する情報ではない**。製品PRでは既存DriverとEnvironmentの選択をそのまま使い、正例を`selected A→actual A`、負例を`selected A→actual B`／同名異Root・SQL・別DBへ分ける。事前拒否できる失格ParamはDB/Executor Call 0、**接続後に判別した別DBや権限Scopeの違いは不適格Resultとして成功Contextを返さない**。同一Attemptで確認できない項目は`unknown`と記録する。新しいProvider Registry/Host/Source正本/Reactive Engineは作らず、#2586/#2514/#1180へ実結果を渡す。

**並行する他チャットの最新適格性**：[#4642](https://github.com/hat47x/tei-platform/pull/4642)は**main統合済みのScript Launch計画更新（docs-only）**。[#4641](https://github.com/hat47x/tei-platform/pull/4641)はBuilderの**論理Graph→物理Build→実Artifact Consumer**対応とGCX-01〜08を整理した設計Draftで、非PluginのStage1→Stage2 Windows実成功ではない。[#4640](https://github.com/hat47x/tei-platform/pull/4640)は署名Release test-only loopbackで得た実GET証拠を、**実GitHub Releases Asset GET 200/302・署名Issuer/リダイレクト・実Body・現在Trust**へ継ぐ条件を置いた設計Draft。両者はG49のProvider/Native Readをブロックする追加Hard Gateにしない。Studio Desktopセルフビルド#2831/#4583が全体P0、Product Read #2586が並行P0であり、旧版の合格を同じ受入票へ加算しない。

<!-- g49-cross-lane-acquisition-migration-toolchain-20261114 -->
### G49 実際の受領結果に結び付ける追加条件（2026-11-14）

**一つの受注履歴の読取り**は、型付きQueryが実Providerを呼ぶことだけでは成立しない。Sourceと実行対象の対応、現在の利用者に対する読取り範囲、取得結果の被覆、生成したアプリがDBへ及ぼす作用、利用側で受け取った本文までを区別する。この整理は[main統合済みの横断Consumer設計 #4653](https://github.com/hat47x/tei-platform/pull/4653)のDefinition→Binding→Authority→Execution→Acquisition→Delivery→Acceptanceに従う**既存受入票の補足**であり、新しい共通Runner、権限管理機構、永続Evidence正本を追加しない。

| 境界 | 成熟した成果と、今回計画へ追加する条件 | Ownerが引き続き検収すること |
| --- | --- | --- |
| **取得できた範囲**（D1／RA-2P） | [取得条件観測Draft #4647](https://github.com/hat47x/tei-platform/pull/4647) HEAD `a962a3f6`は7系統のCI成功。実PG16でRLS、LIMIT、宣言外JOIN、ViewやPartitionによる制限を観測できる範囲では`partial`として理由を運び、通常のNative SQLでは`native-sql-predicate-not-closed`を`open`に残す。**制限が見えない状態は`complete`でも不存在証明でもない**。0行と未取得・非開示を区別し、会員自身の履歴に別会員の存在を推測させる表示を作らない | #2586／#2514。W4の実列OID・NULLの保存元、実DBと宣言Instanceの一致、Source現在性、Grantは未実証。#4647の参照Readを通常`tei` CLIのRA-2P成功へ加算しない |
| **業務操作によるDB変更**（G13／G25） | [生成アプリ修正Draft #4662](https://github.com/hat47x/tei-platform/pull/4662) HEAD `095c5a11`は、生成された`query`／`command`／`validate`でMigrationを暗黙実行しないコード候補と、生成バイナリの別Process負例を追加。Repository Health／Cache／既知脆弱性の**3系統CI成功**。通常Queryは既存DBを読取専用で開き、Commandは既存DBに限り更新し、ValidateはDBなしで実行する。**DB未存在・Schema不適合で業務操作が勝手にDDLやDB作成を行わないこと**を製品側の安全条件とする | #4657／#4234／#4125。これはSQLite単一Provider向けのRB0安全停止候補。実Schema/History/Release検証、明示Init/Upgrade、複数DB・PostgreSQL移行、旧新ClientのCutoverは未完成。DB接続成功を移行適格・現在Grantと同一視しない |
| **再現可能なビルド**（Studio P0） | [Toolchain観測Draft #4648](https://github.com/hat47x/tei-platform/pull/4648) HEAD `64fb0f9a`は8系統CI成功。Go 1.26.11／1.27.0の実測で同じBuild計画でも生成バイトが異なる例を確認し、`toolchainObservation.matched`と`namedByPlan=false`を区別する。前者は申告されたファイルの読戻し一致で、実際にそのToolchainを起動した証明ではない | #2831／#4583。ToolchainをBuild Intentの正式入力に含めるか、Build子Processの実起動とStage1→Stage2成果物受領をどう束縛するかはOwner判断。現時点のSeed Trust／Stage2完成を宣言しない |
| **Desktopでの利用** | [Windows再検証Draft #4587](https://github.com/hat47x/tei-platform/pull/4587) HEAD `9b58533d`は12/12 CI成功。通常版と生成版のWebView2実Jobが両方成功した | B0の実使用に限定。正式な非Plugin Build Subject→実ArtifactSet→Stage1再生成→Stage2起動→WebView2利用を同じ実行で結ぶB2/B3は未達。計画優先順位は変更しない |
| **専門Toolkit・Plugin** | [数理・統計・科学Toolkit #4663](https://github.com/hat47x/tei-platform/pull/4663)の**設計文書はmain統合済み**。これはWindow集計やPERT等の計算機能が実装済みという意味ではない。[役割別Plugin構成 #4664](https://github.com/hat47x/tei-platform/pull/4664)は未統合で、HEAD `dcb694cd`のRepository Healthに失敗が残る。二つの計画の実装資格は別に検証する。統計Window／WBS・PERT／科学計算等の独立Expected、Studioと他Targetの受領条件を整理した | 新Core型・専用DSL・科学Runtime・第二Plugin Hostは先行採用しない。P0のQuery実Read・Studioセルフビルドを待たせず、外部Runtime不要の小さなWindow集計から既存Operator/Consumerで適合性を測る |

**実装順**：最初に[RA-0〜RA-2R一体検証Draft #4644](https://github.com/hat47x/tei-platform/pull/4644)の実Go／参照PG成功を維持したまま、RA-2Pで**通常製品`tei` CLI→実Provider/実DB・Role・SQL・typed Params→Function Data Context**を一つのAttemptとして検査する。取得範囲を`complete`へ推測昇格せず、通常操作が暗黙DDLを実行しない負例も別Ownerへ渡す。RA-3で標準Displayと独立Native Consumerの受領本文を独立に照合し、B1a-Pで会員・匿名・運用者ごとの現在Grantと開示範囲を別途確認する。Studioは非Plugin DesktopのStage1→Stage2一往復を優先し、Toolkit・Migration本格適用・公開Web共同編集をその完成の前提にしない。

**次に実装する一本の最短経路**：#2586 Ownerが`RA-0 main適格化→RA-1 exact base再検証`を先に行い、RA-2参照PGで既に観測した同じQuery/Input/Resultを**正式Product Operation→通常`tei` CLI（同一Subprocess）→実Context→Receiver A/B**へ接続する。Providerは既存Environment/Registryを選択し、Actual Executor/SQL bytes/Session/Rowsetと独立Expectedが同一Attemptで照合できる場合に限りD1 Productの限定受理を検討する。新Host・第二Resolver・暗黙SQL Placeholder推論・新Measure Engine・関数型DSLの表面露出はしない。RA-2のPG Fixtureから再利用するのは**試験手順と観測上限**であって、Reference入口自体を本番Runnerへ名前変更しない。

**横断的停止線**：main統合された[#4636](https://github.com/hat47x/tei-platform/pull/4636)のCapability Input/Effect宣言preflight、[#4634](https://github.com/hat47x/tei-platform/pull/4634)のResolver→Child Pin、[#4637](https://github.com/hat47x/tei-platform/pull/4637)のREA Host固定Selector byte readは、各別OwnerのDraft・限定CI成功であり、Query/Persist/Native HostのGrantを代替しない。共有するのは`Source/Declaration→Selection→Actual Call/Consumed Bytes→Actual Receiver→Independent Expected→Owner Acceptance`の**受入票の構造**のみ。Web/Wails公開は#1824/#3421へ、Studio Desktopセルフビルド#2831/#4583は全体P0のまま。

**計画統合状況**：同一計画ファイルを変更する並行[PR #4600](https://github.com/hat47x/tei-platform/pull/4600)の「問い相対旧依存辺N16/N17」「Studio Windows B0とBuildIntentギャップ」の二行を、本Draftにも**内容を欠落させず**取り込んだ。双方はまだ別Branch/未mainであり、#4600の採否・閉鎖、mainへの実マージ、競合解消を代行したものではない。新実行試験・Owner Meaning受理はNOT RUN。

<!-- g49-b1ap-trust-scope-matrix-20261114 -->
## G49 B1a-P：B2Cの本人・公開・計測を別資格として検収する（2026-11-14）

**位置づけ**：B1a-IのRA-0〜RA-2Rは、[統合Draft #4644](https://github.com/hat47x/tei-platform/pull/4644)の限定同一checkoutでGo引数拒否・参照PostgreSQL 7ケースが成功したにすぎない。現在の製品入口RA-2P、同じContextの独立二Consumer受領RA-3、実公開HostでのB1a-Pは**未成立**。公開資格を得るために新しいTEI Host・Identity台帳・専用Frontend SDK/Runtimeを設けない。実Hostが既存の認証・認可責務を持ち、TEIの型付きOperationとProviderは業務意味・データ取得・結果構造を共有する。

| 利用場面 | 必要な本人・対象資格 | 開示と操作の境界 | 最小の独立反例 |
| --- | --- | --- | --- |
| **匿名の商品閲覧** | 認証主体を要求しないPublic Scope。公開済み商品・価格の「版」と有効期間は区別する | Publicに分類されたField/Artifactだけ。個人価格、会員在庫予約、運用情報は混入させない | Public CacheへPrivate Projectionを保存、旧公開版だけが残る |
| **ゲストのカート** | 一時Session/所有証拠。認証済み顧客や組織Membershipへ自動昇格しない | 仮CartはOrder確定でもPayment受領でもない。Session回復・紐付け時は別の実Host判定 | 別Guest Sessionから同じCart IDを読める／ログイン時に他人のCartへ誤結合 |
| **本人の注文履歴** | 実Hostで検証した現在Principalと、対象Customer/Orderとの権威ある対応、閲覧時点の現在Grant。期間入力`:from`は絞り込みであって認可ではない | Row Scope＋Field/Column Projection＋用途に沿う説明を独立判定。実DBのRLS/Maskは有効な防御層だが、Customer本人の資格そのものではない | 同じ注文値・同じ`:from`でW0許可→W1失効、他人の注文ID、RLSが0件を返すが全件0と誤認 |
| **運用者・サポート** | 別の運用者Principal、用途・対象・必要な現在権限。一般利用者としてのGrantやOrg Membershipを流用しない | 特権閲覧・訂正・代理操作を混在させず、利用目的と監査を実Hostの契約で判定 | 会員向けPublic/Private Cacheへ運用者の補助列・件数・診断を返す |
| **分析・実験計測** | Consent/目的/有効版を実計測時点で検査。Experiment Allocation IDを本人Customer IDと同一視しない | Assignment・実Exposure・Action Intent・業務Effect・Outcomeを別証拠とする。Consentなし・取得不能を非購入に変換しない | 未同意Event送出、露出のない割当をCVへ計上、母集団/欠測を落としたCVR改善主張 |

**B1a-Pの最小実行順**（R-Dの内部受入手順であり、RA-2P/RA-3の前提Gateにはしない）：

1. **P-A Host identity / audience**：既存Hostで受領したPrincipal・Session/Guest・Audienceの出所を確認。Frontendの自己申告Header、Query、DOM選択、旧Cookieだけでは現在の業務本人としない。匿名Publicはその旨を明示する。
2. **P-B Subject link / Grant**：Customer↔Orderの実関係と操作用途・時点・Scopeを突合。照合不能、permission denied、旧Grant、部分Sourceはunknown/deniedを区別して拒否または限定返却とする。公開向けResponseの許可判定は実返却直前に再確認する。
3. **P-C Actual projection / delivery**：選択Query/Providerだけでなく実SQL/DB Role/実Rowset、Row/Column/Relation/COUNT/SUM/診断Metaの実開示、Serializer/HTTP Response、Cache・旧Clientまで**同一Attempt**で検査。内部DSN、Source Root、秘匿列名、件数などがError/Timing/Cacheから推知できる経路を否定対照に含める。
4. **P-D Revocation / independent readback**：W0閲覧成功→W1 Grant失効、別Principal、旧CDN/Browser Cache、別Tab、Client再送で再試行。実Hostの現在認可と独立受領を確認し、保存済み旧証拠やResult byte一致で権限を復活させない。強い全層原子公開は暗黙の必須要件にせず、残存旧配信を検出・遮断・説明する。

**公開資格票**は `Principal source / customer link / purpose / current Grant / row scope / column projection / actual receiver / response version / cache audience / revocation check / independent Expected / Owner acceptance` を列として持つ。原データが0件、Field NULL、RLSによる部分集合、列Mask、データSource欠落では`empty`と`complete`を同一視せず、取得範囲と不明理由を記録する。既存[#2506](https://github.com/hat47x/tei-platform/issues/2506) Context、[#1656](https://github.com/hat47x/tei-platform/issues/1656)/[#2516](https://github.com/hat47x/tei-platform/issues/2516)開示、[#1043](https://github.com/hat47x/tei-platform/issues/1043)/[#3412](https://github.com/hat47x/tei-platform/issues/3412)本人/Session、[#3271](https://github.com/hat47x/tei-platform/issues/3271)/[#2604](https://github.com/hat47x/tei-platform/issues/2604)失効、[#1180](https://github.com/hat47x/tei-platform/issues/1180)/[#2514](https://github.com/hat47x/tei-platform/issues/2514)独立Expectedへ委譲する。

**先行順位を維持**：全体P0はStudio Desktop Stage1→Stage2セルフビルド[#2831](https://github.com/hat47x/tei-platform/issues/2831)/[#4583](https://github.com/hat47x/tei-platform/issues/4583)。並行P0は[#2586](https://github.com/hat47x/tei-platform/issues/2586) RA-2P通常`tei` Product Read→RA-3二Consumer。B1a-Pはその**外部公開への追加資格**。B1b Cart/Order/Payment副作用、B2 Variant/Experiment、B3 H0/H1/H2の費用・性能比較、B4予約/Subscriptionへの拡張を初回ReadのHard Gateにしない。成功資格はSource宣言、局所Go、参照PG、製品CLI、Native、公開Host、Business Effect、実験効果で別々に記録。本文は設計・受入計画であり、実公開認証・Commerce・Browser E2E・CVR実証を行った意味ではない。

<!-- g49-b1b-effect-measurement-separation-20261114 -->
## G49 B1b〜B3：購入の確定・公開切替・計測の資格を連結せず照合する（2026-11-14）

**採る判断**：B2Cの差別化は新しいCheckout/Analytics Engineを作ることではなく、既存の型付きOperation/Effect/Receiptと、独自Frontend/Commerce/Flag/Analyticsの**異なる正本の間で、何が実際に確定したかを取り違えず変更できること**にある。B1a-Iの正式Product Read（RA-2P→RA-3）とB1a-PのCurrent Grantはそれぞれ未成立のまま。以下はその後段の**実施手順・独立Expected候補**であり、実Commerce・Browser・Field実験の完了を表さない。新StateMachine/Host/決済処理/統計Engine/必須Client SDKは設けず、[#1174](https://github.com/hat47x/tei-platform/issues/1174)、[#1129](https://github.com/hat47x/tei-platform/issues/1129)、[#3503](https://github.com/hat47x/tei-platform/issues/3503)、[#951](https://github.com/hat47x/tei-platform/issues/951)、[#1180](https://github.com/hat47x/tei-platform/issues/1180)を再利用する。

| 一つの購入導線の観測点 | 正本と許される主張 | 禁止する昇格・必須反例 |
| --- | --- | --- |
| **① Product/Offer表示** | Product/Price/Promotionの公開版・通貨・有効時刻・Audience、SSR/Edge/Client/Cacheで実表示したVariantを別票に記録。表示は見積りまたは案内である | 古いCDN/Clientの割引表示を現在の販売価格とみなさない。C0の文言変更に実効Coupon/Disclosure/Consent変更を混入させない |
| **② Cart/Checkout Intent** | 同一操作対象・選択Customer/Guest・数量・Quote版、二重送信を判別できるOperation Identity。Optimistic UIは仮状態に限る | 画面上の「完了」、FlagのAssignment、Click/IntentをOrder・Payment確定とみなさない。Guest→会員結合は現在Hostで別判定 |
| **③ 実Provider確定境界** | 権威あるServer/Commerce Providerが現在の価格・税・Coupon・在庫・対象・実行権限を再評価し、選択Providerと実呼出しProvider・送信要求・応答・Commitを突合 | 旧VariantのOffer、旧Grant、A選択→B実Call、Hookのbefore/after成功だけでCommit成功としない。実Provider能力を超えるexactly-onceを宣言しない |
| **④ Effect/Receiptの確定** | 同じOperation IdentityでReceipt・再照会可能なOrder/Payment Outcome・独立Readを分ける。重複要求はProviderの冪等契約に従い再照合する | Timeout/応答喪失/unknown commitでは**未確定**のまま。安全な自動再送と仮定せずlookup first、照会不能はunknownを維持。Client UIの成功表示やAnalytics Eventで補完しない |
| **⑤ Experiment/Analytics** | Assignment、実際のExposure、許可されたEvent、Action、業務Effect、Outcomeを異なる観測とし、母集団・欠測・Consent版・割当版・観測時刻を記録 | Exposure未観測を非露出確定、計測不能を非購入、Order確定を因果upliftに変換しない。SRM、返品・取消・粗利益等のguardrailは既存分析/Providerを利用 |
| **⑥ Release/撤回** | UI/Creative/Flagのロールバックと、既存Order/Paymentの訂正・返金は**別操作**。旧SSR/Edge/Client/Service Worker・Private Cacheの生存と現在Backend受理条件を照合 | Flag巻戻しで既存の注文Effectを消したと扱わない。既配信・オフライン保存済みbyteを遡及的に消去できる保証はしない。失効後の**新規配信**を現在Grantで抑止し、残存配信は別途検出・運用対処する |

**B1bの最小縦切り**：公開の本番決済ではなく、隔離したCommerce Sandboxを一つ選ぶ。正常Cart→Checkoutだけでなく、同じOperation IDによる重複送信、Commit前拒否、Commit後ACK喪失、旧Quote/旧Grant、Provider選択と実Callの不一致を各一組の正負対照にする。外部Effectについては、Client表示・実Call・Provider Receipt・独立Order再読込みを別列に残す。Writeの効果や権限を証明できないときは`unknown`とし、既存[#1174](https://github.com/hat47x/tei-platform/issues/1174)/[#1129](https://github.com/hat47x/tei-platform/issues/1129)へ未成立を返す。B1a-Pの公開Host合格をSandbox内のEffect実装の着手条件にせず、外部公開の認定には別途現在Grantを要求する。

**B2の最小縦切り**：既存Experiment/Flag Providerを任意選択し、**一つの既存Native画面**でAssignment→SSR実表示→Hydration→Exposure→任意Analytics送出→Checkout Outcomeを時点と版付きで追跡。Variant混在、旧Client復帰、未同意、計測欠落、遅延Event、既存注文取消後のOutcome誤計上を反例にする。Payload/集計で個人識別情報・顧客関係・DSN・内部Evidence位置を無断露出しない。結果の`consent-denied`、`not-observed`、`unavailable`をゼロEvent／ゼロ購入と同一視しない。

**変更影響／失効への引渡し**：Native Frontend・CMS/Creative・Flag/Allocation・Target Build/Release・業務Rule・Provider/Contract・Current Grant/Consent・Source Relationの実効差分から、既存[#4518](https://github.com/hat47x/tei-platform/issues/4518)/[#3271](https://github.com/hat47x/tei-platform/issues/3271)/[#2582](https://github.com/hat47x/tei-platform/issues/2582)/[#2600](https://github.com/hat47x/tei-platform/issues/2600)の必要上流・実消費・失効候補を照合する。ファイル種別が文言/画像でも価格/同意へ影響するなら軽い変更レーンで逃がさない。旧Graphから削除されたRequired辺、A→B→A、Provider/Generator版変更、Audience/時点切替を「byte一致だから安全」と判断しない。逆に無関係なNative Routeや未選択H2 Adapterを巻き込む過剰失効も検出する。Graph/Source Universeがpartial/unavailableなら`no impact`と断定しない。

**B3の比較条件**：[#2018](https://github.com/hat47x/tei-platform/issues/2018)の既存比較に、H0=Nativeのみ、H1=選択した型付きContract/Build-time Adapter、H2=さらに任意Experiment Adapterを、**同じNative Frontend・同じ実Provider/Scenario・同じ機能・同じ変更・同じ測定条件**で投入する。正確性を独立Expectedで先に判定し、初回実装工数・変更時間・Review/復旧工数・追加JS/HTTP Request/TTFB/INP/LCP/CLS・Provider/運用費・発見した不正/欠測を別票に採る。初期値は未測定。H0にTEI JS/DOM/Router/Runtimeを強制しない。H1/H2の費用が便益を超えるときは**Plugin非採用・撤去を正式な合格選択肢**とする。CVR/粗利益の因果効果は標本、割付、Consent、追跡欠測、期間、統計的不確かさを確認した実Experimentがなければ`not measured`とする。

**順序と責務**：全体P0は[#2831](https://github.com/hat47x/tei-platform/issues/2831)/[#4583](https://github.com/hat47x/tei-platform/issues/4583) Stage1→Stage2 Desktop Self-build。並行P0の[#2586](https://github.com/hat47x/tei-platform/issues/2586) RA-2P/RA-3からB1a-Iを実Consumerへ接続。B1a-Pは[#1043](https://github.com/hat47x/tei-platform/issues/1043)/[#1656](https://github.com/hat47x/tei-platform/issues/1656)/[#2516](https://github.com/hat47x/tei-platform/issues/2516)の実Host開示で別検収。B1bは#1174/#1129/#951、B2は#3503/#2604/#3271、B3は#2018へ受入を返す。すべて[#1180](https://github.com/hat47x/tei-platform/issues/1180)/[#2514](https://github.com/hat47x/tei-platform/issues/2514)の既存Independent Expectedに追記し、**B2C専用の第二Scenario/Effect/Analytics正本を作らない**。本文は設計と検収順の更新であり、新規Go/PG/Browser/Commerce/実験試験とOwner意味受理は`NOT RUN / NOT ACCEPTED`。

## プロセス・イベント・応答を共通の意味から説明する（設計候補）

Webの要求、CLIの起動、タイマー、Message、画面操作には、それぞれ異なるHost・起動方式がある。ここで標準化を検討するのは一つのイベントループではなく、**Trigger→型付きInvocation→Function/Rule→State/Effect→Outcome→Response／Consumer受領**という意味の対応である。長寿命Processの起動・停止と、一件の処理の成功・失敗も混同しない。

初期の受入は、受注受付の説明用Fixtureを使い、HTTP/CLIの二入口が同じ意味Operationへ至るかをHeadlessで照合する。期限の境界、現在Grant、DB Commit後の通知結果不明、重複・逆順Event、未実装のTimer Providerを負例にする。外部部品はContractとTODOを先に記述し、既存Driver ScenarioのMockと独立Expectedで基本設計からシミュレーションする。ただし、Mockの成功は実DBやBrokerの実行証拠にならない。

詳細は[Lifecycle・Trigger・Operationの共通意味設計](../design/PROCESS_LIFECYCLE_TRIGGER_OPERATION_SEMANTICS_2026-11.md)を参照する。Ownerは[共通Application API #1158](https://github.com/hat47x/tei-platform/issues/1158)、[Script Launch #4495](https://github.com/hat47x/tei-platform/issues/4495)等の既存Issueに分け、新Runtime・Dispatcher・Schedulerを前提にしない。**現行P0のStudio Stage1→Stage2と通常CLIの実Readには、新しい先行依存を追加しない。**


<!-- g49-lifecycle-process-invocation-acceptance-20261114 -->
### G49の読取りと業務確定を、同じ意味モデルから別々に検証する

[共通実行モデルの設計 #4665](https://github.com/hat47x/tei-platform/pull/4665)と[Lifecycle・Triggerの設計 #4666](https://github.com/hat47x/tei-platform/pull/4666)はmain統合済みである。ただし、両方とも**設計上の対応関係**を定めたもので、共通Event Loop、Scheduler、Graph Compiler、Host、正式DSL構文の新規採用ではない。G49では、Processの生存状態と、利用者による一件のInvocationが成功したかを別に扱う。

**最初は二種類のOperationを混ぜずに試す。** `order_entry.recent_orders`は指定日時以降の注文履歴を読み取るOperationであり、RA-0〜RA-2Rでは型付き入力と使い捨てPGの参照読取りまでを検証している。受注確定`submit`はDB Commit・Receipt・通知などの外部作用が関係し、読取りが成功してもその資格を引き継がない。どちらも、元のSourceにあるSubject・Guard・入力・Output/Outcome・必要なEffectを意味の対応元とし、説明文・処理図・Scenarioを派生表現として生成する。生成した説明文や図を第二の業務意味正本にはしない。

| 段階 | 最小の確認 | 必ず区別する反例 | Owner |
| --- | --- | --- | --- |
| **LifecycleとTrigger** | CLI起動とHTTP要求が同じ意味Operationを指せるか。単発CLIに常駐Loopを要求しない | Triggerの宣言を、実Schedulerの発火や現在の実行許可と混同しない（PL-01/02/06） | [#1158](https://github.com/hat47x/tei-platform/issues/1158)・[#4495](https://github.com/hat47x/tei-platform/issues/4495) |
| **Typed InvocationとRead** | `from`の宣言・実引数・選択Provider・実Connection・取得したRowsetとCoverageが同一Attemptに対応するか | 選択A／実B、型違い、RLSの`partial`、0行を全件不存在と誤認する場合（PL-01/10、PX-10） | [#2586](https://github.com/hat47x/tei-platform/issues/2586)・[#2514](https://github.com/hat47x/tei-platform/issues/2514) |
| **意味評価と実作用** | 受注確定のGuard、期限と現在Grant、提案されたState/EffectをContractとMock／独立Expectedで先に対照する | Mockの受注確定を実DB Commitと呼ぶ、期限の発生時刻と判定時刻を取り違える（PL-03/07） | [#1129](https://github.com/hat47x/tei-platform/issues/1129)・[#1174](https://github.com/hat47x/tei-platform/issues/1174) |
| **EffectとOutcome** | Commit成功・通知送信・通知受領・結果不明を別Receiptで検証する | Commit後のACK喪失を未実行と誤認し、無条件再送で注文を重複確定する（PL-04/05/09、PX-03/04） | [#1174](https://github.com/hat47x/tei-platform/issues/1174)・[#1129](https://github.com/hat47x/tei-platform/issues/1129) |
| **Responseと実Consumer** | HTTP/CLI/Nativeへ同じ対象・版・結果の適切なProjectionが届いたか。必要な現在Grantを受領時にも照合する | HTTP 200、CLI終了コード0、表示成功をConsumerの意味受理／業務確定と同一視する（PL-10、PX-01/11） | [#1251](https://github.com/hat47x/tei-platform/issues/1251)・[#3509](https://github.com/hat47x/tei-platform/issues/3509) |
| **図解・Target移植** | 同じ意味からLifecycle／Control／Effect／Traceの図を読取専用で投影し、Source位置と安定IDを保持する | 図の配置変更で意味Revisionが変わる、同期→非同期移植で順序・期限・Effect保証を落とす（PX-05/06/12） | [#4198](https://github.com/hat47x/tei-platform/issues/4198)・[#1124](https://github.com/hat47x/tei-platform/issues/1124) |

**実現順**は、既存のProduct Read RA-2PとStudio Stage1→Stage2を先に完結させ、共通Lifecycleの最初の読取り専用Projectionをその結果へ結び付ける。TODO・Mockで外部部品を置き換える設計段階では、入力・結果・欠損・副作用・実装未接続を明示し、Mockと本番Providerを同一の成功票にしない。実Targetへの移植は、実際のInvocation、作用、Response、Consumerの受領を比較してから判断する。**この節は設計とScenarioの引渡しであり、新たなGo／DB／Windows／Scheduler実験の完了ではない。**

## 最新の作業単位へ接続する

2026年11月11日のmain `9d31f82486b2eac42791f4f3bc5cd588d69cb4fc`と関連Issueを照合した。以下は着手先の更新であり、Issueの起票・限定Reference・候補Parserを製品の完成へ数えない。以後の進捗は各Issueと[実装状況](../../docs/status.md)から再取得する。

| 既存の項目 | 現在の足場と着手先 | 次に満たす条件 |
| --- | --- | --- |
| G08〜G11/G37〜G42 | [#4350](https://github.com/hat47x/tei-platform/issues/4350)のS1〜S3（[#4351](https://github.com/hat47x/tei-platform/issues/4351) / [#4352](https://github.com/hat47x/tei-platform/issues/4352) / [#4353](https://github.com/hat47x/tei-platform/issues/4353)）は合成Headless Referenceとしてmain統合済み。横断Matrix [PR #4464](https://github.com/hat47x/tei-platform/pull/4464)と限定Source編集 [PR #4538](https://github.com/hat47x/tei-platform/pull/4538)はOpen（2026-11-14照合） | 読取り用構造写像の判定・用途別Observer・操作Slot単位の逆写像・現在Grant/実Consumerを分離する。11/32セルの合成判定やSource編集の隔離試験を実Provider→通常`tei`/Studio→独立Consumerの製品合格へ転用しない。単位・軸・完全性の未宣言はunknown/unsupportedで残す |
| G61 | [#4359](https://github.com/hat47x/tei-platform/issues/4359)が構文比較と実装順を所有。比較票は[#4361](https://github.com/hat47x/tei-platform/issues/4361)、候補Parserは[#4362](https://github.com/hat47x/tei-platform/issues/4362) | Parse、既存意味への変換、Source位置、互換性、正式採用を順に区別する。DSL案を製品Loaderの既定へ先に入れない |
| G19/G25/G61 | [#1130](https://github.com/hat47x/tei-platform/issues/1130)のETL/Batch設計は[PR #4409](https://github.com/hat47x/tei-platform/pull/4409)へ、取得・変換の最初の縦断は[#4410](https://github.com/hat47x/tei-platform/issues/4410)、再開・外部実行は[#4411](https://github.com/hat47x/tei-platform/issues/4411)へ分割済み | 同じ固定入力と独立ExpectedをGo/CLIで照合する。新しいETL専用DSLや必須Schedulerを加えず、再実行と外部作用の結果を分ける |
| G21/G23/G28 | プラグイン管理の一覧・診断は[#4376](https://github.com/hat47x/tei-platform/issues/4376)、計画・差分は[#4377](https://github.com/hat47x/tei-platform/issues/4377)、設定源と原子性は[#4378](https://github.com/hat47x/tei-platform/issues/4378) | 実Workspaceからの解決と選択、実呼出し、更新・失効の結果を別に観測する。管理画面の一覧を実Provider適合へ昇格しない |
| G54 | [#4134](https://github.com/hat47x/tei-platform/issues/4134)の検証義務候補P0-Aと、固定有限データP0-Bを[#1131](https://github.com/hat47x/tei-platform/issues/1131)/[#1180](https://github.com/hat47x/tei-platform/issues/1180)へ接続 | 生成した試験の数と判別力を分ける。独立Expectedを固定し、満足例、単一違反、相互作用、観測不足と失敗の最小化を対照にする |
| G13 | [#2586](https://github.com/hat47x/tei-platform/issues/2586)がP0。旧PR #4076はclosed/unmerged。[PR #4539](https://github.com/hat47x/tei-platform/pull/4539)はEnvironment経由の`public.orders/:from`実PG読取、一時的Function Context、別プロセスReference CLIを実装し、exact-head CI成功後にmain統合済み（`ee1af44c`）。使い捨てPG16での局所実測も記録。ただし**製品`tei` CLI・独立Native Consumer・現在Grantは未接続** | Referenceの`public.orders/:from`とXTEAの`sales."order"/:status`は別Query。#4539 mainの実Read/Scope/接続先身元・Typed Param・Rowset・Context/独立Readerを基線に、RLS/0件/欠落列/マスクで`unestablished`を保持する。次に製品`tei` CLI、同一Attemptを受領する独立Native Consumerを順に検証。PG局所PASSやCI成功を現在Grant/全件取得/Checkoutへ転用しない |
| G49（B2C Native変更のCoverage） | [#3509](https://github.com/hat47x/tei-platform/issues/3509)の[最小B0判定票](https://github.com/hat47x/tei-platform/issues/3509#issuecomment-6813494503)をG49の初回受入候補に接続。案件固有Frontend/Edge/CMSは標準Rendererの採用を強制せず、実Source/Artifact R0→R1と既知のOperation/Context/Consumerだけをread-onlyで関連付ける。Native未登録部分の依存関係を自動で完全とみなさない | 既存BC-16/18/19/27/29/32/35/36/37/39/41を使い、通常文言C0と割引・同意条件を含む変更C3、旧Variant/旧Cartの正負を同一Fixtureで比較する。Change identity、Coverage、複数Lane候補、Required verification、Observed evidenceを別票にし、Coverage不足からNo Impact/公開安全/CVR向上を宣言しない。B0は**設計・試験候補**。#2586の実Readを先行し、H0/H1/H2の生産性比較は#2018、Commerce実Effect/実Browser/収益効果は後段で個別に検証 |

### G08〜G13/G37〜G42：データ様式の用途・操作相対適格性を実Product経路へ接続（2026-11-14追補）

[#4350](https://github.com/hat47x/tei-platform/issues/4350)は段階・依存の索引、[#2495](https://github.com/hat47x/tei-platform/issues/2495)/[#2544](https://github.com/hat47x/tei-platform/issues/2544)はFacet/演算意味、[#2497](https://github.com/hat47x/tei-platform/issues/2497)は写像/損失、[#4133](https://github.com/hat47x/tei-platform/issues/4133)は表現型、[#2506](https://github.com/hat47x/tei-platform/issues/2506)/[#2586](https://github.com/hat47x/tei-platform/issues/2586)はFunction Data Context/実Read、[#1124](https://github.com/hat47x/tei-platform/issues/1124)は設計Source編集、[#2500](https://github.com/hat47x/tei-platform/issues/2500)は業務データのViewChange→Operation Intentを所有する。本追補はその**製品受入順序**だけを明確にし、第二の意味正本・型Registry・Projection Engine・Action/Grant/Host・Scenario SSOTを作らない。

**判断を混同しない**：A) 構造写像の`verdict`（既存5語彙）、B) **目的・問い・Observer・Scope・許容損失に相対的なread適合**、C) **対象Subject/Relation occurrence・Slot・具体Operationに相対的な逆編集候補**、D) **現在Source/Context epoch・権限・実Target/Effectの資格**は別列とする。一般の`lossy` Graph→Treeでも、明示ID・一意値span・恒等的に表示されたNode所有属性の局所更新は候補になりうる。一方、Edge IDなしのreparent/unlink、集計値の明細への自動配分、同ラベルからのIdentity復元は拒否する。これは既存Checkerの全写像`inverse=ineligible`を直ちに`eligible`へ変更する指示ではない。Slot/Operationスコープと独立Expectedを先に比較する（[#4533](https://github.com/hat47x/tei-platform/issues/4533)、[PR #4538](https://github.com/hat47x/tei-platform/pull/4538)）。

**同一性と現在性**：`Subject ID`／`Edge occurrence`／`Source byte span`／`View occurrence`／`Selection epoch`／`Workspace`／`Provider snapshot`／`Grant`を混ぜない。Raw byte fingerprint一致はA→B→Aの履歴同一性、現在の意味受理、Writer権限を証明しない。編集Apply直前に現在Source/Revision・span値・Writer権限・独立Requiredを再照合し、旧AI Replyや別Contextの選択から書き戻さない。設計SourceへのPatch/Acceptanceと実DBへのAction/Commit/Receipt/独立再Readは別の縦断で検証する。表現の`null / missing / unfetched / withheld`と集合の`empty / partial / complete / unknown`、Queryの非表示・否定依存（`NOT EXISTS`等）は用途相対の必要閉包を保持する。

| 順序 | 既存Owner・既存ケースを使う最小縦断 | 合格として要求する証拠 / 主張しないこと |
| --- | --- | --- |
| **D0：Reference契約の整理** | #4351〜53を基線に、#4464の11/32セル、`best`用途条件、`unknown/unqualified/uncovered`、#4533/#4538のSlot編集差を照合（XMAP-02/03/07、RSP-02、DF-X03、VE-01/03）。 | 同じ構造損失でもreadとSlot Editの判定が区別でき、既知Mutantと独立Expectedで偽許可・偽拒否を発見する。未検査21セルを安全/不可能と決めない。 |
| **D1：実Read→二Consumer** | #4539でmainに統合された限定PG16 Reference Readを出発点に、#2586/#2506の同一Function・同一Attempt・同一Context/epoch/Scopeを表と集計Consumerへ渡す（DF-X01、HJ-03〜05）。 | `observed slice`と母集団、JOIN fanout、`COUNT/SUM`、0行・RLS・NULL・未取得・非開示・否定依存を独立Expectedで区別。通常`tei` CLIと実Consumer受領は未実証なので接続するまではNOT RUN。 |
| **D2：設計Sourceの局所編集** | #1124/#2497とPR #4538で、表/Propertyの一意な型付き属性Patch→既存Preview/Review/Accept→実Source再Read/再投影を接続。Graph→TreeはNode属性の正例と多重Edgeの負例を追加（DF-X03）。 | 異Workspace同名、旧Revision、A→B→A、Edge ID欠落、集計編集、非可逆投影を独立に判別し、無関係byte不変・失敗時書込0を確認。限定合成Passを現在のOwner意味受理へ昇格しない。 |
| **D3：業務Function操作・異種Target** | #2500/#1174で同じ明示Entity/Relationに対するIntent候補→現在Grant→実Effect Receipt→独立Read。#2498/#4133/#3827、SHO/SUI、外部Java/.NET Hostは用途に必要な異種Consumerを順に追加（DF-X04〜08）。 | Selection・UI配置からAction/Authorityを創出しない。実Host/DB、Owner Effect、外部Consumer、媒体同期の各Claimにexact版と実観測を要求。 |

**停止線**：`Source Revision`・`Context epoch`・`Identity/Slot`・`Quantity/Time/CRS`・`Required Scope`・`current Grant`のうち当該操作に必要なものが未確認なら、read説明は可能でもEdit/Executeを許可しない。実行証拠はWorld/Model/Artifact revision、実選択Provider/Generator、実消費byte、独立Expected、Owner受理を分離し、既存[#2514](https://github.com/hat47x/tei-platform/issues/2514)/[#1180](https://github.com/hat47x/tei-platform/issues/1180)/[#1119](https://github.com/hat47x/tei-platform/issues/1119)のGateに載せる。新たな公開DSL・Core TypeKind・固定15表現Enumは異種Consumerの反例を得てからADRで判断する。


<!-- g49-crosslane-current-head-correction-20261114 -->
**2026-11-14最新の横断差分（下段の節は#4619の作業票として移植した時点の記録）**：Script Launch [#4636](https://github.com/hat47x/tei-platform/pull/4636)はHEAD `da807abc`で**main統合済み**（2026-11-14 03:50 UTC）となった。これはCapability Input/Effect**宣言の構造的preflight**であって、実入力値／業務Owner意味／現在Grantではない。現HEADでは後発のCI実行が混在しており、**一部Workflow failure・未完了**も観測したため、移植元にある旧「Draft・9/9成功」を**最新状態としては採用しない**。BuildIntent統合[Draft #4634](https://github.com/hat47x/tei-platform/pull/4634)は現在HEAD `7d6a1238`へ更新され、旧HEAD `653d0d9`での**8/8成功は履歴票**。新HEADのCIは別Attemptで判定する。REA Host Reader [Draft #4637](https://github.com/hat47x/tei-platform/pull/4637) HEAD `170f9be5`の**12/12 CI成功**は限定ローカルReaderの実装資格であり、現在Principal/GrantやWails/公開Webの資格ではない。これらはいずれも#4638参照PGの成功とは別受入。

### 異なる入力経路に共通する「選択→実受領」の証拠連鎖（2026-11-14更新）

本節は**既存の各Ownerの証拠を比較するための作業票**であり、全Target共通の新Core Schema、統一Invoker、第二の許可/Store/Runnerを設ける提案ではない。Source/Workspaceの同一性、契約・利用者の選択、実際に使われたbyte・実行者、受信者・開示範囲、Ownerによる受理は異なる判断である。**同じdigest、同じ表示値、同じ時刻、同じPRのgreen CIだけでは、その間の接合を実証したことにならない**。Source A→B→A、同名異Root、Actor/Grant失効、旧Responseの後着を個別の負例として残す。

| 経路／現HEADの局所証拠 | 最初に未成立な接合・実際に拒否すべき事例 | Ownerと受入順（既存Issueを使う） |
| --- | --- | --- |
| **L-R：Query→実DB**。[#4631](https://github.com/hat47x/tei-platform/pull/4631) `02af8339`専用CI [#43548961899](https://github.com/hat47x/tei-platform/actions/runs/43548961899)は限定Go合格。#4626型付きSourceはDraft。 | RA-0現在Golden/旧Frozen履歴の分離→RA-1現HEAD検証→**RA-2の実選択Provider A／実呼出Bの不一致拒否**と実Consumed SQL/Params/Rowsetの照合。型検査成功・合成Contextのみを実Provider実行としない。 | #2586/#1180/#2514。最初のD1、次にRA-3独立二Consumer、D1.2再集計は後続。 |
| **L-P：Function→Go Adapter**。[#4629](https://github.com/hat47x/tei-platform/pull/4629)はmain統合済みのResult契約不正→Adapter call0。[#4636](https://github.com/hat47x/tei-platform/pull/4636) `da807abc`はInput/Effect宣言不正・nil Contextのcall0、**exact HEAD 9/9 CI成功**だがDraft。 | 実入力値が型/意味に合うか、公開Entryが本当に選択されたか、Host現在Grant・選択Providerが実際に呼ばれたかを別に確認。宣言`Effect`≠Commit Receipt、契約拒否成功≠業務完了。 | #4495/#1174/#3422、既存Script Launch §17とその独立Expected。RA-2実Readの成功票へ加算しない。 |
| **R3：REA Evidence→Host→受信者**。[#4625](https://github.com/hat47x/tei-platform/pull/4625) ローカルCLI 9/9 CI、[#4630](https://github.com/hat47x/tei-platform/pull/4630) byte-only R2 12/12 CI、[#4637](https://github.com/hat47x/tei-platform/pull/4637) Host固定Selector/Root＋4MiB制限Reader `170f9be5` **12/12 CI成功**。すべてDraft、#4637は#4630の差分を含む。 | 正しいEvidence ID/Digestでも、異Workspace・Root差替え・別Principal・古いGrant・別Audienceへの公開は許されない。**ローカルCLIの任意PathをWeb/MCP入力へ転用しない**。ReaderのGo内部受領≠Wails/外部Consumerの配送受領。HostでSource/Grant/Disclosureを再評価し、R3-C異種二Consumerを別実証。 | #3394/#4503/#2516/#3419、#4627 Web Bridgeと別検収。#4630の統合前に#4637を独立mergeしない。 |
| **BuildIntent→実Builder→Desktop**。[#4634](https://github.com/hat47x/tei-platform/pull/4634) `653d0d9`はResolver宣言整合と選択Builder→子Image SHAの結合で**8/8 CI成功**、Draft。 | Plugin主体BuildPlanでの選択B／実起動A拒否は、非Plugin `cmd/tei-studio` N0～N3と生成Stage1→Stage2 Windows WebView2の受領ではない。Builder digestだけでLoader TOCTOU・外部依存・Owner Authorityを満たしたとしない。 | #4583/#2831。Desktopセルフビルドが独立P0、[#1824](https://github.com/hat47x/tei-platform/issues/1824)のlocal IPC A/B/C比較は後続。 |
| **Wails Server Bridge→公開Web**。[#4627](https://github.com/hat47x/tei-platform/pull/4627) `b9cf5447`はService登録・実Go exported 15 method棚卸し、**5/5 CI成功**のDraft。 | 一覧に`ExecuteCommand*`とSHO子Process入口がある。**静的InventoryのPASSは公開AuthN/AuthZでも安全なWeb配送でもない**。不正Actor/Workspaceからの実HTTP/WS拒否、二受信者間の秘密非開示、read-only façadeを別に確かめる。Browser提供の`allowed []string`をHost Grantと認めない。 | #1824/#3421/#2516。local loopback Server PoCをmulti-user公開へ自動昇格しない。 |

**RA-0に対する具体的な修正の受入票（既存Ownerへの指示）**：Sourceに`input from std.datetime`を追加した結果を「全部Golden更新」で解決しない。
1. `internal/workspace/load_listing_fingerprint_test.go`および`reference/world.json`との比較は、**現Queryを含むCurrent Worldの意味改訂と新Revision**を個別に検査する。受入済み旧値を文字列置換するだけでなく、旧SourceのExpectedがどこに属するか記録する。
2. `tools/reference-shared-acceptance/boundary_test.go` のScratch baselineは、歴史Gate B/Cと現在Worldのどちらを検証するか対象を固定し、`sourceRevision/World`の混在を拒否する。旧受理を現Sourceの受理へ昇格しない。
3. `tools/reference-plugin-self-build/historical_runner_parity_e2e_test.go`のFrozen Worktreeは、現在HEADのQuery新入力を旧Readerが読めない実ログを踏まえ、**歴史比較用treeに限定してQueryの旧raw byteを固定・digest検査**する変更を検討する。その場合、現時点の「差分がmanifest 2ファイルだけ」という断定・検査も変更内容と一致させる。現在Queryの`input`を削除したり、旧Parserの失敗をskipして解決したことにしない。
4. Browser/API `sqlgen`二symbolの検査失敗は[#1182](https://github.com/hat47x/tei-platform/issues/1182)で実依存を調べる。RA-0 Source変更が直接原因と決めつけず、Allowlistを広げて正常扱いにしない。

**受入順序**：#4626の各修正HEADで現在World/歴史Fixture/Healthを再検証→積上げ#4631の新baseでGo/旧Web回帰を再実証→#2586のRA-2非SKIP実PG→RA-3独立受信者。ここから別P0のDesktop Stage1→Stage2を遅延させない。上表のCI成功は**各Draftのexact HEADの限定実証**であり、本追補によるGo/PG/Windows/HTTP/WS実行、Business Owner受理は**NOT RUN**。

**次に通す最短の実Product経路（2026-11-14更新）**：型付きQuery SourceのCurrent World/歴史World分離はRA-0の限定Health・Self-buildで、Executor前の型検証はRA-1のGo CIで、隔離PostgreSQL参照ReadはRA-2Rの非SKIP実CIで、**修正RA-0を含む同一仮統合HEADでも再検証済み**である。次の未成立辺は、既存Environment／Driverを用いて**通常の`tei`製品入口→実選択Query/Inputs→実引数→実選択Provider→実接続DB/Role・SQL bytes/Rowset→Function Data Context**までを一つのAttemptとして記録するRA-2Pである。正例の選択A→実A、負例の選択A→実B、同名異Root、Source A→B→A、未宣言入力・NULL、0行とRLS可視範囲を独立Expectedで判定し、申告だけのProvider種別と実DB受領を同一視しない。続くRA-3で標準Displayと独立Native Consumerが**同じContextをそれぞれ実受領**したことを確認する。B1a-P公開Hostの現在Grant、D1.2 Measureの演算法則・集計粒度、D2編集、D3業務Effectは別Owner・別資格であり、初回ReadのHard Dependencyを増やさない。RA-0に残るAPI依存除外[#1182](https://github.com/hat47x/tei-platform/issues/1182)の失敗を検査緩和で隠さず、Source/Context/Consumerの独立検査と区別する。全工程の製品受入・Owner意味受理、通常Product CLI・二Consumerの新E2Eは**NOT RUN**。

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

<!-- g49-d12-ra1-ci-evidence-20261114 -->
**横断引渡し**：同一計画文書を変更する[Draft #4619](https://github.com/hat47x/tei-platform/pull/4619)のMeasure再集計D1.2節を、見出しと表を含めてこの下へ移植した。#4619は維持し、受理・main採用・Merge Conflict解消を代行しない。R-A/RA-1が認めるのは実Query Input/Provider前型検証であって、Measure/COUNT/SUM/AVG・同一母集団・同一Snapshotや未観測Consumerの意味保証ではない。対象Scope・中間状態・演算Law・NULL/丸めを満たす再集計は#2593/#2544/#1705の別Owner資格とする。D1.2はR-Cの後続でありRA-1/Studio P0のHard Gateとしない。

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

## プロセス・イベント・応答を共通の意味から説明する（設計候補）

Webの要求、CLIの起動、タイマー、Message、画面操作には、それぞれ異なるHost・起動方式がある。ここで標準化を検討するのは一つのイベントループではなく、**Trigger→型付きInvocation→Function/Rule→State/Effect→Outcome→Response／Consumer受領**という意味の対応である。長寿命Processの起動・停止と、一件の処理の成功・失敗も混同しない。

初期の受入は、受注受付の説明用Fixtureを使い、HTTP/CLIの二入口が同じ意味Operationへ至るかをHeadlessで照合する。期限の境界、現在Grant、DB Commit後の通知結果不明、重複・逆順Event、未実装のTimer Providerを負例にする。外部部品はContractとTODOを先に記述し、既存Driver ScenarioのMockと独立Expectedで基本設計からシミュレーションする。ただし、Mockの成功は実DBやBrokerの実行証拠にならない。

詳細は[Lifecycle・Trigger・Operationの共通意味設計](../design/PROCESS_LIFECYCLE_TRIGGER_OPERATION_SEMANTICS_2026-11.md)を参照する。Ownerは[共通Application API #1158](https://github.com/hat47x/tei-platform/issues/1158)、[Script Launch #4495](https://github.com/hat47x/tei-platform/issues/4495)等の既存Issueに分け、新Runtime・Dispatcher・Schedulerを前提にしない。**現行P0のStudio Stage1→Stage2と通常CLIの実Readには、新しい先行依存を追加しない。**

## 進行の記録と完了条件

各G項目には、対象と版、既存の足場、次の作業、独立した正例・反例、依存、担当Issue、得た結果または未実施の理由を残す。状態は定義済み、実装候補、局所試験、通常入口での実行、利用側確認、案件受入を用途に応じて記す。Issueをclosedにしただけで、別の組合せや実行先が成立したと扱わない。

中小規模案件の対応率は事前に定めた案件と必須条件の集合に対して測る。標準のみ、既存資産接続、新規拡張、未成立を分け、保留・実施不能の理由を示す。大規模案件の省力化は生成時間に加え、追加質問、手修正、レビュー、統合、更新と引継ぎを比較する。性能は正しい結果を保った入力・構成で測り、量、待ち時間、メモリ、費用と取消後の処理を残す。

計画への反映が完了した状態は全G項目の担当、設計への接続、次の受入、依存を確認できることとする。製品としての達成は各Issueと既存の受入経路で判定する。理想像が更新された場合は、既存の項目との関係を先に確認し、不足する作業と条件を同じ担当へ戻す。 [生産性 #2018](https://github.com/hat47x/tei-platform/issues/2018) · [案件対応範囲 #3796](https://github.com/hat47x/tei-platform/issues/3796)
