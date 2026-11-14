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

**計画統合状況**：同一計画ファイルを変更する並行[PR #4600](https://github.com/hat47x/tei-platform/pull/4600)の「問い相対旧依存辺N16/N17」「Studio Windows B0とBuildIntentギャップ」の二行を、本Draftにも**内容を欠落させず**取り込んだ。双方はまだ別Branch/未mainであり、#4600の採否・閉鎖、mainへの実マージ、競合解消を代行したものではない。新実行試験・Owner Meaning受理はNOT RUN。

## 最新の作業単位へ接続する

2026年11月11日のmain `9d31f82486b2eac42791f4f3bc5cd588d69cb4fc`と関連Issueを照合した。以下は着手先の更新であり、Issueの起票・限定Reference・候補Parserを製品の完成へ数えない。以後の進捗は各Issueと[実装状況](../../docs/status.md)から再取得する。

| 既存の項目 | 現在の足場と着手先 | 次に満たす条件 |
| --- | --- | --- |
| G08〜G11/G37〜G42 | [#4350](https://github.com/hat47x/tei-platform/issues/4350)が型・構造Facet・表現・Context・写像を段階化。最初の検査は[#4351](https://github.com/hat47x/tei-platform/issues/4351)、構造は[#4352](https://github.com/hat47x/tei-platform/issues/4352)、Context束縛は[#4353](https://github.com/hat47x/tei-platform/issues/4353) | 既存の意味Ownerと独立Expectedへ接続する。単位・軸の記述欄が不足する現行Sourceを、型名の一致だけで表現できたと扱わない |
| G61 | [#4359](https://github.com/hat47x/tei-platform/issues/4359)が構文比較と実装順を所有。比較票は[#4361](https://github.com/hat47x/tei-platform/issues/4361)、候補Parserは[#4362](https://github.com/hat47x/tei-platform/issues/4362) | Parse、既存意味への変換、Source位置、互換性、正式採用を順に区別する。DSL案を製品Loaderの既定へ先に入れない |
| G19/G25/G61 | [#1130](https://github.com/hat47x/tei-platform/issues/1130)のETL/Batch設計は[PR #4409](https://github.com/hat47x/tei-platform/pull/4409)へ、取得・変換の最初の縦断は[#4410](https://github.com/hat47x/tei-platform/issues/4410)、再開・外部実行は[#4411](https://github.com/hat47x/tei-platform/issues/4411)へ分割済み | 同じ固定入力と独立ExpectedをGo/CLIで照合する。新しいETL専用DSLや必須Schedulerを加えず、再実行と外部作用の結果を分ける |
| G21/G23/G28 | プラグイン管理の一覧・診断は[#4376](https://github.com/hat47x/tei-platform/issues/4376)、計画・差分は[#4377](https://github.com/hat47x/tei-platform/issues/4377)、設定源と原子性は[#4378](https://github.com/hat47x/tei-platform/issues/4378) | 実Workspaceからの解決と選択、実呼出し、更新・失効の結果を別に観測する。管理画面の一覧を実Provider適合へ昇格しない |
| G54 | [#4134](https://github.com/hat47x/tei-platform/issues/4134)の検証義務候補P0-Aと、固定有限データP0-Bを[#1131](https://github.com/hat47x/tei-platform/issues/1131)/[#1180](https://github.com/hat47x/tei-platform/issues/1180)へ接続 | 生成した試験の数と判別力を分ける。独立Expectedを固定し、満足例、単一違反、相互作用、観測不足と失敗の最小化を対照にする |
| G13 | [#2586](https://github.com/hat47x/tei-platform/issues/2586)がP0。旧PR #4076はclosed/unmerged。[PR #4539](https://github.com/hat47x/tei-platform/pull/4539)はEnvironment経由の`public.orders/:from`実PG読取、一時的Function Context、別プロセスReference CLIを実装し、exact-head CI成功後にmain統合済み（`ee1af44c`）。使い捨てPG16での局所実測も記録。ただし**製品`tei` CLI・独立Native Consumer・現在Grantは未接続** | Referenceの`public.orders/:from`とXTEAの`sales."order"/:status`は別Query。#4539 mainの実Read/Scope/接続先身元・Typed Param・Rowset・Context/独立Readerを基線に、RLS/0件/欠落列/マスクで`unestablished`を保持する。次に製品`tei` CLI、同一Attemptを受領する独立Native Consumerを順に検証。PG局所PASSやCI成功を現在Grant/全件取得/Checkoutへ転用しない |
| G49（B2C Native変更のCoverage） | [#3509](https://github.com/hat47x/tei-platform/issues/3509)の[最小B0判定票](https://github.com/hat47x/tei-platform/issues/3509#issuecomment-6813494503)をG49の初回受入候補に接続。案件固有Frontend/Edge/CMSは標準Rendererの採用を強制せず、実Source/Artifact R0→R1と既知のOperation/Context/Consumerだけをread-onlyで関連付ける。Native未登録部分の依存関係を自動で完全とみなさない | 既存BC-16/18/19/27/29/32/35/36/37/39/41を使い、通常文言C0と割引・同意条件を含む変更C3、旧Variant/旧Cartの正負を同一Fixtureで比較する。Change identity、Coverage、複数Lane候補、Required verification、Observed evidenceを別票にし、Coverage不足からNo Impact/公開安全/CVR向上を宣言しない。B0は**設計・試験候補**。#2586の実Readを先行し、H0/H1/H2の生産性比較は#2018、Commerce実Effect/実Browser/収益効果は後段で個別に検証 |

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

## 進行の記録と完了条件

各G項目には、対象と版、既存の足場、次の作業、独立した正例・反例、依存、担当Issue、得た結果または未実施の理由を残す。状態は定義済み、実装候補、局所試験、通常入口での実行、利用側確認、案件受入を用途に応じて記す。Issueをclosedにしただけで、別の組合せや実行先が成立したと扱わない。

中小規模案件の対応率は事前に定めた案件と必須条件の集合に対して測る。標準のみ、既存資産接続、新規拡張、未成立を分け、保留・実施不能の理由を示す。大規模案件の省力化は生成時間に加え、追加質問、手修正、レビュー、統合、更新と引継ぎを比較する。性能は正しい結果を保った入力・構成で測り、量、待ち時間、メモリ、費用と取消後の処理を残す。

計画への反映が完了した状態は全G項目の担当、設計への接続、次の受入、依存を確認できることとする。製品としての達成は各Issueと既存の受入経路で判定する。理想像が更新された場合は、既存の項目との関係を先に確認し、不足する作業と条件を同じ担当へ戻す。 [生産性 #2018](https://github.com/hat47x/tei-platform/issues/2018) · [案件対応範囲 #3796](https://github.com/hat47x/tei-platform/issues/3796)
