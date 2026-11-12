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
- **B0-S0（同一Loadの実Source観測）**：[PR #4517](https://github.com/hat47x/tei-platform/pull/4517)で既存`workspace.Load`＋`BuildSourceIndexFromLoad`＋`InspectionFromLoad`を直接用いる局所Go試験を提案。Source A→B→A、Load後のファイル変更、別Workspace root拒否と実Queryの出典を対照する。**このPRの検証と統合は別資格**。Load時ByteからのIndexは原子的な多ファイルSnapshot、Imported Provider/Native依存の閉包、Current Grantの証明ではない。
- **B0-S1（次の実装担当）**：[Issue #4518](https://github.com/hat47x/tei-platform/issues/4518)で`Workspace + Qualified Subject/occurrence + Source revision + 既存Operation/Context`から、案件が明示したNative Touchpoint/Action/Consumer Bindingへ限定投影する。Required/Observed/Resolved/Unresolved/Excludedを**用途と質問に相対的に**診断し、未探索のJS/CMS/Edge/外部Tagはunknownとする。Native UIへの項目追加はDB列の作成ではなく、Data/Function Mappingがない場合は未対応として診断する。明示的なData項目削除でMappingが失効してもUI要素は残り得る一方、未取得・非開示・partialを削除として扱わない。`SELECT *`／`alias.*`／`NATURAL JOIN`などの暗黙的な列集合・結合条件への依存は、明示列の参照が0件でも影響候補とし、実Schema/Queryの取得範囲が不足する場合はunknownとする。既存#2582のClaim単位Required Setに従い、影響外と証明できるContextは保持し、影響範囲だけを再検証する。未MappingのNative UI項目はDraftでは保持できても、必要なBindingを欠いた公開・業務Actionを検証済みとしない。削除・改名・DDLの実Applyは既存Data/Schema Ownerが所有し、Source/Scope/Selectionの再確認なしに自動付替えしない。完全なNative Source解析、第二Source Reader/Graph Resolver/Hostは実装しない。
- **順序と証拠の上限**：#2586の実Function→実Query/Provider→Rowset→Function Data Context→ConsumerをP0に維持し、B0-Sの完成をHard Gateとしない。#4350 DF-X06は標準表現/Nativeの異質Consumerの同一意味を検査。#4496/#4515は固定Subject/Scope/Read epoch/操作資格を所有し、#4507/#4514のSource-only driftでは同byteの復帰とSource/Authorityの現行性を区別する。B1a実Native Consumer、B1b実Commerce Effect/別Read、B2実配信/Exposure/Consent、B3のCVR・利益因果評価は**別の未実証資格**。
- **性能・プラグイン・Evidence**：#2018でH0 Native単独/H1 TEIのBuild時契約/H2任意Experiment Adapterを、同じ変更・独立Expectedと導入/改訂/運用費用で比較する。#951/#4198/#3278のProvider/Graph/Traitは実選択と実呼出しを照合。#4307/#2516に従い非公開Evidence・その存在情報をGitHubや外部Harnessへ必須送出しない。公開の権限、Consent/Grant、現在価格、実注文Effect、売上改善をB0で宣言しない。

## 最新の作業単位へ接続する

2026年11月11日のmain `9d31f82486b2eac42791f4f3bc5cd588d69cb4fc`と関連Issueを照合した。以下は着手先の更新であり、Issueの起票・限定Reference・候補Parserを製品の完成へ数えない。以後の進捗は各Issueと[実装状況](../../docs/status.md)から再取得する。

| 既存の項目 | 現在の足場と着手先 | 次に満たす条件 |
| --- | --- | --- |
| G08〜G11/G37〜G42 | [#4350](https://github.com/hat47x/tei-platform/issues/4350)が型・構造Facet・表現・Context・写像を段階化。最初の検査は[#4351](https://github.com/hat47x/tei-platform/issues/4351)、構造は[#4352](https://github.com/hat47x/tei-platform/issues/4352)、Context束縛は[#4353](https://github.com/hat47x/tei-platform/issues/4353) | 既存の意味Ownerと独立Expectedへ接続する。単位・軸の記述欄が不足する現行Sourceを、型名の一致だけで表現できたと扱わない |
| G61 | [#4359](https://github.com/hat47x/tei-platform/issues/4359)が構文比較と実装順を所有。比較票は[#4361](https://github.com/hat47x/tei-platform/issues/4361)、候補Parserは[#4362](https://github.com/hat47x/tei-platform/issues/4362) | Parse、既存意味への変換、Source位置、互換性、正式採用を順に区別する。DSL案を製品Loaderの既定へ先に入れない |
| G19/G25/G61 | [#1130](https://github.com/hat47x/tei-platform/issues/1130)のETL/Batch設計は[PR #4409](https://github.com/hat47x/tei-platform/pull/4409)へ、取得・変換の最初の縦断は[#4410](https://github.com/hat47x/tei-platform/issues/4410)、再開・外部実行は[#4411](https://github.com/hat47x/tei-platform/issues/4411)へ分割済み | 同じ固定入力と独立ExpectedをGo/CLIで照合する。新しいETL専用DSLや必須Schedulerを加えず、再実行と外部作用の結果を分ける |
| G21/G23/G28 | プラグイン管理の一覧・診断は[#4376](https://github.com/hat47x/tei-platform/issues/4376)、計画・差分は[#4377](https://github.com/hat47x/tei-platform/issues/4377)、設定源と原子性は[#4378](https://github.com/hat47x/tei-platform/issues/4378) | 実Workspaceからの解決と選択、実呼出し、更新・失効の結果を別に観測する。管理画面の一覧を実Provider適合へ昇格しない |
| G54 | [#4134](https://github.com/hat47x/tei-platform/issues/4134)の検証義務候補P0-Aと、固定有限データP0-Bを[#1131](https://github.com/hat47x/tei-platform/issues/1131)/[#1180](https://github.com/hat47x/tei-platform/issues/1180)へ接続 | 生成した試験の数と判別力を分ける。独立Expectedを固定し、満足例、単一違反、相互作用、観測不足と失敗の最小化を対照にする |
| G13 | [#2586](https://github.com/hat47x/tei-platform/issues/2586)の実ReadがP0の主線。旧PR #4076はclosed/unmergedで、このmainには対応するFunction Context実装がない | Referenceの`public.orders/:from`とXTEAの`sales."order"/:status`を別入力として扱い、一つを選ぶ。実Bind、実Rowset、Mapping、Context、CLI受領、独立Expectedを同一試行でつなぐ。追加の受入候補である物理DB同一性とSession/RLSも同じ試行で観測し、宣言名・正常0件だけで実対象を確定しない |
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
