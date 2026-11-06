# TEI：設計可読性・意味Authoring・標準Consumerの横断接続計画（2026-11）

- **状態**：構成統合・実証計画のDraft。新しいCanonical schema、Parser、実行Runtime、Acceptance、Gateを採用する文書ではない。
- **基準main**：`9c9ef4d160640908e06ad13afb6c2b45c664c748`（2026-11-02の照合時点）。最新PR/Issue・実行証拠は着手時に再確認。
- **親の調整**：[Critical Path #1119](https://github.com/hat47x/tei-platform/issues/1119)と[Everything is Data #2619](https://github.com/hat47x/tei-platform/issues/2619)。意味定義・各技術実証は既存Issueを正本とする。
- **履歴の扱い**：クリティカルパス文書§1〜32は維持。別担当の[Draft #2599](https://github.com/hat47x/tei-platform/pull/2599)が§33を編集するため、本計画は競合しない独立文書として管理する。従来[Draft #2610](https://github.com/hat47x/tei-platform/pull/2610)の§33/§34で検討した内容を本書§1/§2へ移した。
- **先行順位**：既存S0/J5、P0-a実PG結果/Lookup、P0-b二Function Owner-write、S3 Owner意味受理、G0〜G4を変更せず、以下をP1/P2の並列Consumer Pressureとして扱う。

## 1. 2026-11-02：一業務の設計可読性・Headless評価・実効果・文書を接合する

**位置づけ・固定根拠**：本節は [#1119](https://github.com/hat47x/tei-platform/issues/1119) の既存C0〜C4、G0〜G4、§31/§32の履歴を改変しない**横断の実施順序とConsumer受渡し案**。照合main `9c9ef4d160640908e06ad13afb6c2b45c664c748`（2026-11-02）。既存[業務設計方針](../../docs/business-system-design.md)、[RQ-010〜013/RQ-025〜030](../../requirements/TEI_PRODUCT_REQUIREMENTS.md)、[M3/M4/M9](../MILESTONES.md)を参照する。[#2484](https://github.com/hat47x/tei-platform/issues/2484)のD-1〜D-3の採用・D-4保留と、[統合済PR #2588](https://github.com/hat47x/tei-platform/pull/2588)を前提とするが、採用済み要件から実Target/Owner受入済みとは推論しない。[AI-first/Headless-first追補PR #2492](https://github.com/hat47x/tei-platform/pull/2492)はDraft/未統合の検討資料として参照する。

### 1.1 最初の成果は「意味が読める一業務」であり、生成コードではない

既存`order_entry`を題材に、**受注確定の設計を一枚で読める**ことを最初のConsumer価値に置く。ただし公開Referenceの合成業務例であって、原典X-TEAのOwner受理済み業務仕様ではない。設計票は、新たなCanonical正本やWorkflow DSLではなく、既存Data/Function/BusinessモデルとSource/Evidenceを指す**read-only Projection**として構成する。

| 人が確認する事項 | 必須の説明・未決の保持 | 既存担当と受渡し |
| --- | --- | --- |
| 目的・対象・責任 | Business Operation、業務上の一件、提出対象集合、担当/手作業/外部Handoffと適用時点。Function/Capabilityの宣言を業務Actionの意味受理に変えない | [#1125](https://github.com/hat47x/tei-platform/issues/1125)、[#267](https://github.com/hat47x/tei-platform/issues/267)、[#2505](https://github.com/hat47x/tei-platform/issues/2505) |
| Data・値の根拠 | Source→Query→Function Data Contextの値lineage、1:Nの取得集合・粒度・重複・欠損/明示NULL・通貨/単位・時点・部分取得。数値一致だけで再集計可としない | [#1126](https://github.com/hat47x/tei-platform/issues/1126)、[#2506](https://github.com/hat47x/tei-platform/issues/2506)、[#2586](https://github.com/hat47x/tei-platform/issues/2586)、[#2593](https://github.com/hat47x/tei-platform/issues/2593) |
| 条件・評価範囲 | 規則本文だけでなく先頭/編集対象/全明細の量化範囲、空集合の扱い、Snapshot/履歴の必要条件。評価済み不成立と未評価を分ける | [#1063](https://github.com/hat47x/tei-platform/issues/1063)、[#1125](https://github.com/hat47x/tei-platform/issues/1125)、[#1708](https://github.com/hat47x/tei-platform/issues/1708) |
| 操作と実Request | 表示・選択→候補Intent→採用Action→Operation SurfaceのBinding→実Payload対象を別々に結ぶ。generic updateをsubmit/cancelの意味一致とみなさない | [#2485](https://github.com/hat47x/tei-platform/issues/2485)、[#2500](https://github.com/hat47x/tei-platform/issues/2500)、[#2598](https://github.com/hat47x/tei-platform/issues/2598) |
| 確定と業務完了 | Invocation、Commit/ACK、Receipt/Lookup、独立Observation、歴史的BusinessResult、現在状態、確認の現在利用資格、Ownerの完了条件を別に示す | [#1174](https://github.com/hat47x/tei-platform/issues/1174)、[#1129](https://github.com/hat47x/tei-platform/issues/1129)、[#1163](https://github.com/hat47x/tei-platform/issues/1163)、[Draft #2518](https://github.com/hat47x/tei-platform/pull/2518) |
| 説明・納品 | モデル意味版、生成Artifact、SHO文書Occurrence/局所編集、表示依存（図等）、説明レビュー、納品固定版を分け、意味の編集逆写像を自動承認しない | [#1803](https://github.com/hat47x/tei-platform/issues/1803)、[#2497](https://github.com/hat47x/tei-platform/issues/2497)、[#2516](https://github.com/hat47x/tei-platform/issues/2516)、SHO/MMI既存計画 |

一枚にまとめるとは、すべての証拠を一つの永続schemaへ複製することではない。根拠のOwner・版・scope・時点・取得可能範囲へ**戻れる**ことをいう。設計書の可読性、業務意味のOwner受理、Target実装の資格、文書レビュー/公開可否は独立。

### 1.2 Headlessでの判別を先行し、実UIは固有の保証だけ追加する

| 段階（追加Gateではない） | 最初の判別ケース・期待する限定証拠 | 待たないもの |
| --- | --- | --- |
| 設計を読む | 受注確定の対象・条件・手作業・結果をread-onlyの表/関係一覧へ展開し、Source/未決へ戻る。Graphの未解決端点・診断対象と描画省略を混同しない | Wails完成、旧原モデル全件移植、Owner未受理箇所の勝手な補完 |
| Headless意味評価 | **二明細を入替え**、先頭行・編集対象行・全行の検査範囲を比較。通貨/単位未受理、部分Result Slice、JOIN増幅、履歴上の集合出入りをそれぞれ未評価または反例として明示 | 汎用Simulation Engine/Leanの全面採用、新Evaluator/第二の検証基準 |
| 操作対象と実効果 | Intentのsubmitがgeneric update/cancelへ誤結合されないこと、Intent対象X≠Payload/Effect対象Y、Commit不明と確定後Observation不明、同じ現在値でもBusiness完了は異なることを別のClaimとして比較 | 新Host、汎用Retry/Approval/Recovery Engine |
| 表示・文書へ投影 | Headlessで固定した同版・同scopeの根拠をCUI/Modeler/SHOへ表示。未保存Patch、旧応答、改版後の図/引用/文書上書きを理由なく元Sourceへ書き戻さず、説明適用不能を表示 | 全GUI完成、新共通文書正本、旧レビューの自動復活 |

**判別の先行証拠**：[#1125の三規則の合成正常系](https://github.com/hat47x/tei-platform/issues/1125#issuecomment-6470719087)は存在するが、全明細被覆・第3規則負例停止・正式Providerの証拠ではない。[#2485の合成bound-write](https://github.com/hat47x/tei-platform/issues/2485)と[#2482の成果物内部CheckSet](https://github.com/hat47x/tei-platform/issues/2482)はそれぞれ成立範囲を持つが、導出Intent→実Requestの意味同一性や公開Artifact一式の同版性へ自動昇格しない。原Driverの共有Transactionに生じるambient effectは旧挙動の観測であり、TEIの望ましいBusiness規則として無条件採用しない（[#940](https://github.com/hat47x/tei-platform/issues/940)、[#1129](https://github.com/hat47x/tei-platform/issues/1129)）。

### 1.3 優先順位と終了条件（既存Ownerを変えない）

- **既存P0を維持**：S0の原Source/J5検収、[#1174](https://github.com/hat47x/tei-platform/issues/1174)の実PostgreSQL確定後結果不正→Receipt/独立読取/Lookup（P0-a）、[#1129](https://github.com/hat47x/tei-platform/issues/1129)・[#1127](https://github.com/hat47x/tei-platform/issues/1127)の同一Data Rule二Function/Owner-write（P0-b）、[#1125](https://github.com/hat47x/tei-platform/issues/1125)→[#267](https://github.com/hat47x/tei-platform/issues/267)の業務意味受理（S3）。これらを設計票のread-only生成待ちで直列停止しない。
- **P1として並行**：[#2586](https://github.com/hat47x/tei-platform/issues/2586)のQuery→Data Context→Claim→Pattern→Intent、[#2598](https://github.com/hat47x/tei-platform/issues/2598)の実Request束縛、[#2597](https://github.com/hat47x/tei-platform/issues/2597)のdirty/Undo/ABA、[#2505](https://github.com/hat47x/tei-platform/issues/2505)の版間追跡、[#2516](https://github.com/hat47x/tei-platform/issues/2516)のConsumer別開示・SHOを既存試験資産で接続する。AI-first/Headless-first/GUI-laterは実証順序であり、GUI固有のAccessibility、操作、表示、配布された実装をHeadless PASSで代替しない。
- **Target/AIを並行拡張**：Java/.NET/WASM等の外部Host向けLibrary適合は[#2390](https://github.com/hat47x/tei-platform/issues/2390)、[#2513](https://github.com/hat47x/tei-platform/issues/2513)等で扱う。モデルの意味、Adapter字句/型、全実行Artifact、実Consumer、Host権限を別証拠にする。外部Harnessは[#2016](https://github.com/hat47x/tei-platform/issues/2016)/[#2515](https://github.com/hat47x/tei-platform/issues/2515)でPrepared/Delivered/Produced/Recorded/Verifiedと実OS・情報開示を確認し、CLI exit0やTool完了を業務完了にしない。
- **部分結果の終点**：そのConsumerが要求する対象集合、粒度、時点、履歴、現在Authority、実送出許可を確認できない場合、正当なread-only部分説明は維持できるが、**全件・不存在・完全な業務完了・実行可能・公開可能**とは表示しない。確認対象がA→B→Aへ戻った場合も、撤回・集合membership変更・遅着訂正・過去Receiptの適用資格は別に評価する。
- **既存計画との整合**：[#2484 D-4](https://github.com/hat47x/tei-platform/issues/2484)の#982「締めと訂正」追加は保留。periodは採用済みのC4の位置付けを使い、新C5や期間実装を本節で採用しない。[#2226](https://github.com/hat47x/tei-platform/pull/2226)はDraft/Openであり、ADR-0041の番号衝突は未決。TEI独自共通Host/Registry、第二の意味正本、第二のAcceptance/検証基準、追加Gateを作らない。

**本節の完了の意味**：既存Ownerが次の作業・判別入力・必要証拠・Stopを辿れる計画追補としては完成。設計票UI、Headless縦断、実PostgreSQL/Windows、Java/.NET/WASM Consumer、SHO保存、Owner意味受理、G1〜G4製品Gateの新しいPASSを意味しない。過去§31/§32の証拠時点は書き換えない。

## 2. 2026-11-02：Script-first標準Studioと意味Pluginの拡張を、一業務のConsumerへ接続する

**新たに明確になったOwner方針**：[ #988 Script/DSL firstと共通Inspect](https://github.com/hat47x/tei-platform/issues/988#issuecomment-6573241174)／[ #1124 Studioの標準Projection](https://github.com/hat47x/tei-platform/issues/1124#issuecomment-6573251117)（2026-11-02）。本節は§1の「一業務の可読な設計書」を**将来の各DSL/Pluginを導入しても専用画面の追加を前提としない利用体験**へ一般化するための、既存IssueのConsumer接合計画。いずれも方針・設計案であり、新Syntax・公開Schema・製品実装が受理済みと宣言するものではない。照合main `9c9ef4d160640908e06ad13afb6c2b45c664c748`。§31/§32歴史、C0〜C4、G0〜G4、#2484のD-1〜D-4、既存P0-a/P0-b/S0/S3の順序を変更しない。

### 2.1 標準データ構造で完結する最初のStudio価値

| 操作・表示 | 正本・意味の所有 | 共通UI/Consumerが最低限できること | 採用済みとみなせない条件 |
| --- | --- | --- | --- |
| Script/DSLの著述 | 該当DSLの正式Source（テキストの場合はtext＋採用済みExtension Lock/Revision、構造Sourceの場合はそのauthoritative schema） | テキストと、parse/resolveで得られるsymbol・source span・型・参照/依存・診断・説明をgeneric table/tree/propertyへ投影 | AST/ノード座標・生成Graphを第二の編集正本としない。全DSLのtext-only化も要求しない |
| Generic UIでの編集 | [#1124](https://github.com/hat47x/tei-platform/issues/1124)のSource-aware編集、[#2497](https://github.com/hat47x/tei-platform/issues/2497)の可逆性/損失 | 書戻し先Source spanとMeaning Revisionが一意で可逆ならtyped change→Preview→既存Acceptance→reparse/reproject。曖昧/派生/非可逆ならread-only＋原Sourceへnavigation | Preview成功・画面上の「保存済み」をOwnerの意味受理へ変えない。ScriptとGUIで別正本を立てない |
| 項目のvisible/enabled/required | [#988](https://github.com/hat47x/tei-platform/issues/988)のOperator/DSL評価、[#2506](https://github.com/hat47x/tei-platform/issues/2506)のContext、[#2582](https://github.com/hat47x/tei-platform/issues/2582)の変化再評価 | 小さなtyped predicateとして依存・対象Instance/epoch・入力取得/NULL・未評価/循環を説明し、同じSourceからCLI・Studioへ投影 | UIでenabledでも実AuthorityがなければEffect不可、hiddenでも直接Request/データ取得はHost側で拒否。非開示データの存在や件数を漏らさない（[#2516](https://github.com/hat47x/tei-platform/issues/2516)/[#1178](https://github.com/hat47x/tei-platform/issues/1178)） |
| Pluginを追加 | [#951](https://github.com/hat47x/tei-platform/issues/951)/[#988](https://github.com/hat47x/tei-platform/issues/988)の意味拡張、[#1824](https://github.com/hat47x/tei-platform/issues/1824)のStudio共通UI、[#997](https://github.com/hat47x/tei-platform/issues/997)の生成アプリUI | 未知のDSL/Operatorでも標準text/inspect/diagnostic/dependency navigationが、専用Studio画面の追加なしに成立するか実証 | Native Canvas等の領域固有UIは正規の追加選択肢。「無限の拡張性」は専用UI種類で拡張上限を固定しない設計目標であり、任意Target・任意Langでの無条件実行保証ではない |

**最初の反証**：[#988 SF-01〜05](https://github.com/hat47x/tei-platform/issues/988#issuecomment-6573241174) と [#1124 SU-01〜05](https://github.com/hat47x/tei-platform/issues/1124#issuecomment-6573251117)を**同じ小さな架空の表示/活性Script**で接合する。最初はtext→typed dependency→CLI generic inspectを固定し、Studio generic property一往復を次のConsumerとして追加。改版後の古いProjection、複数source span、Macro展開、NULL/未取得、UI enabledでもGrant失効の負例で誤書戻し・実効果を拒否する。専用ノードエディターや新Graph Rendererの完成を待たず、真正なUI制御・実Host拒否は別試験にする。

### 2.2 意味Pluginの「利用可能性」を、型だけで判断しない

| 境界 | 統合して得られた区別 | 次の既存Consumerへの確認 |
| --- | --- | --- |
| Function Patternの合成 | [#2507 AG-01〜20](https://github.com/hat47x/tei-platform/issues/2507#issuecomment-6572909015)／[Draft #2613](https://github.com/hat47x/tei-platform/pull/2613)で、partial→full、group≠line、AVGの中間状態不足、submit≠cancel、Clock mismatch、nested unknown、Instance状態共有を**Python合成20/20 PASS**で識別。判定語はcontradicted/not_established/fixture_consistent（有限fixtureの結果） | #2507のAssumptions/Guaranteesを[#2586](https://github.com/hat47x/tei-platform/issues/2586)の同一Function/Source/Target縦断へ持ち込む。Go/実Provider/実Host/形式証明・Owner受理は未成立 |
| DefinitionとPlacement | 同じDefinitionでもInstanceごとにState/Clock/seedが異なる。改版したDefinition・Provider Generation・Domain Ruleの発効時点を一つの版に潰さない | [#2507](https://github.com/hat47x/tei-platform/issues/2507)、[Draft #2612](https://github.com/hat47x/tei-platform/pull/2612)、[#988](https://github.com/hat47x/tei-platform/issues/988)で、状態移行が部分写像のとき「新VersionでもState保存済み」を自動認定しない |
| Pattern→Action→実Request | 同じData/Effectのsubmit/cancel、Capability追加による暗黙のBinding差替え、同じSemantic Dataの異なるDB Alias/Persistence経路を**別の不一致**として扱う | [#2485 W7-B1〜08](https://github.com/hat47x/tei-platform/issues/2485)、[#2611 PM-01〜07](https://github.com/hat47x/tei-platform/issues/2611)、[#2598](https://github.com/hat47x/tei-platform/issues/2598)で候補・入力写像・実Provider対象を個別に照合。Draft [#2587](https://github.com/hat47x/tei-platform/pull/2587)の選択規則と合成bound-writeを正式なAction意味受理に拡張しない |
| Data/Measureの利用資格 | 同じDecimal、同じResult列でも、Relation occurrence/Bag/Grain、時点、集計軸、sum/count、丸めの段階で合法な再集計が異なる | [#2593](https://github.com/hat47x/tei-platform/issues/2593)、[#2506](https://github.com/hat47x/tei-platform/issues/2506)、[#2387](https://github.com/hat47x/tei-platform/issues/2387)を共通inspectの診断・Context説明へ投影。Query/SQL Providerの製品PASSは別 |
| Skillと外部Consumer | [#2614](https://github.com/hat47x/tei-platform/issues/2614)/[Draft #2607](https://github.com/hat47x/tei-platform/pull/2607)は実TEI CLIを案内するProduct Skillの提案。Skillが説明する操作と現在のCLI機能・実装・権限を区別 | #2614はSkillの対象/根拠、[#2515](https://github.com/hat47x/tei-platform/issues/2515)は実OS/開示、[#2016](https://github.com/hat47x/tei-platform/issues/2016)は入出力Witness。TAIの横断SkillとTEI Domain Pluginの第二実装を作らない |

ここでの**構成原理**：同じ型・UI部品・Capability名であっても、適用条件（集合/粒度/時点/Instance/Target/Authority）と、既存のGuaranteeが不足する場合は「その用途で利用可能」と結論しない。意図的な新しいCoreの適合状態enumや一律A/G Engineは先決せず、各OwnerのReferenceと既存Contractへ必要な情報を返す。

### 2.3 文書まで届く際の最後の境界

共通Studioが意味を読めること、ActionがTargetへ束縛されたこと、Artifact/Manifestが内部整合であること、**外部SHO/CLI/人手Consumerが同じ一回の成果物集合を受け取れたこと**は独立の保証。複数Artifactの公開中に旧版残存・途中書込み・異なるManifestが見える反例は[#2604](https://github.com/hat47x/tei-platform/issues/2604)の既存P1へ引き渡す（[#2482](https://github.com/hat47x/tei-platform/issues/2482)の内部整合とは別）。設計書中の引用・図・文書上のReviewと業務Owner Acceptanceは[#2516](https://github.com/hat47x/tei-platform/issues/2516)/SHO担当のまま。Graph、表、文書とNative UIはいずれも同じ意味を使う**用途別Projection**であるが、非可逆投影の編集、未列挙依存、外部開示は自動合格にしない。

**作業配列**：§1のS0/J5、P0-a実DB確定/Lookup、P0-b同Rule二Function、S3 Owner意味受理は引き続き優先。§2のSF/SU/AG/PM/公開Artifact検証は、既存[#2493](https://github.com/hat47x/tei-platform/issues/2493)の並列要件拡大型開発でP1/P2として拡大する。初回各PoCの小ささを製品スコープの上限とせず、実証で見つかった不足を標準Plugin・共通Studioへ還流する。正式の新型/DSL Syntax/Host/Registry/第二受理基盤/新Gateは導入しない。

**証拠状態**：本節は2026-11-02指定mainと既存GitHub Issue/PR、既報の合成試験結果を照合した**計画文書の編集**。#2507のPython20件は別担当の局所報告であって、この追補で再実行していない。TEI Go/実Provider・DB/Studio GUI/生成Target/SHO納品/外部Skill・Owner意味受理は本節では**NOT RUN/NOT ACCEPTED**。


## 3. Everything is Data：複数構文から単一の型付き意味へ接続する

2026-11-02の新たな横断判断材料：[親 #2619](https://github.com/hat47x/tei-platform/issues/2619)、[意味Role・自己記述 #2620](https://github.com/hat47x/tei-platform/issues/2620)、[Markdown／表テストと配置 #2621](https://github.com/hat47x/tei-platform/issues/2621)、[Data Operations／Provider一般化 #2622](https://github.com/hat47x/tei-platform/issues/2622)。各Issueは**設計・受入候補**であり、正式な内部データ型・表面構文・「Data Operations」という名称は未採用。

### 3.1 構文、標準データ、意味Role、実現を四層に分ける

```text
Markdown表 / 式・関数 / method chain / Graphviz DOT / Goa風 / MukuroL風
       ↓ syntax plugin：parse、元位置／版の保持、明示的desugar
標準の意味データ形：Record・Sequence・Ref・Relation・Expression等
       ↓ Role/Operator plugin：identity・型・範囲・依存・評価時点・Effect要求をresolve
既存の受理済み意味・適用条件・Function Data Context
       ↓ 既存Lowering / Provider / Host
Target実行・実Effect・独立Observation
       ↘ Studio共通UI／CLI／SHO文書／同版テストの各Projection
```

- **同じデータ形は同じ業務意味の無条件保証ではない。** Role/Operatorの正確なID・version、参照scope、値/宣言/式の区別、順序・Bag/Grain・NULL/未取得・単位/丸め、pure/effect、必要Contextまで解決した範囲で比較する。意味Roleの適用成功はOwner Acceptance/実行認可と別。
- **Syntax Sugarは書きやすさの選択肢**。関数定義、表、チェーン、DOTは内部意味モデルを増やさずプレーンデータへdesugarする候補。DOTの配置だけなら表示Projection、Operator/入力を表す束縛が欠ければ編集用の可逆Syntaxとは判定しない。メソッドチェーンの評価順・短絡・副作用の順序はデータ側へ残す。
- **一つのAuthoritative Sourceを指定する**。Markdownを編集正本とするProjectならnormalize結果・AST・Studio viewは派生。保存済みデータ構造を編集正本とするProjectならテキストはAuthoring Projection。元構文のコメント・レイアウト・改版由来を意味変更へ取り違えず、逆書戻し条件を[#2497](https://github.com/hat47x/tei-platform/issues/2497)/[#1124](https://github.com/hat47x/tei-platform/issues/1124)で判別する。
- **自己記述は段階的に成立させる**。基本値・参照・Plugin解決のbootstrapを先に固定し、RoleのRequired Shape/適用条件/Operator/診断をデータとして記述できる範囲を測る。循環・重複Role・解釈順の衝突は未解決として示し、Coreの第二Parserや自己実行Engineを先行導入しない。
- **生成AI向け構文は比較する**。Goaの契約の宣言性とMukuroLの簡潔な階層構造を参考に、Script/DSLの短いGrammar＋Role schema＋例/負例を既存CLI/Skillで取得できるか試す。候補の見た目で正式Syntaxを選ばず、変更の局所性・Source定位・修復可能性・曖昧な語彙・Token消費で比較する。

### 3.2 既存Issueに渡す一つのHeadless縦断（試験は計画・NOT RUN）

| Case | 反証可能な入力・操作 | 判定して引き渡すもの | 実施Owner |
| --- | --- | --- | --- |
| EID-A：表層正規化 | 同じ合成Function/Rule/TestをMarkdown表、式/チェーン、Goa風・インデント構文、構造化データで入力 | 解決したRole/Operator/typed ref/Scopeが一致する範囲でMeaning比較。元表層のSource spansはそれぞれ保持。異なる式順・副作用を同一としない | [#2620 SY/AI-DSL](https://github.com/hat47x/tei-platform/issues/2620)／[#988](https://github.com/hat47x/tei-platform/issues/988) |
| EID-B：情報損失のある表示 | 同じFunctionをGraphvizの表示ラベルのみで表現し、根拠付きDOTと比較 | ラベルだけからOperator・predicate・出力grainを復元しない。逆書戻しは限定領域だけ許可、layout変更はSource表示差とする | #2620／[#2497](https://github.com/hat47x/tei-platform/issues/2497)／[#1124](https://github.com/hat47x/tei-platform/issues/1124) |
| EID-C：Markdownテスト | 同じ文書内で説明用表、定義表、Fixture、独立Expectedを近接配置、文書内で移動/並替え | Profileと明示Roleで実行対象を選別。説明表の誤実行なし、意味IDとSource locationを分離、Expectedを生成実装から独立 | [#2621](https://github.com/hat47x/tei-platform/issues/2621)／[#1131](https://github.com/hat47x/tei-platform/issues/1131)／[#2514](https://github.com/hat47x/tei-platform/issues/2514) |
| EID-D：三層Contextの利用 | 親→子→孫の明示Context Mapping、集合の一部のみ取得、孫を親別の二層Summaryへ投影 | Relation path、粒度・完全性・Lineage・Unit/NULL・集計法則を提示。Relationだけで全件取得を推定しない | [#2622 DO](https://github.com/hat47x/tei-platform/issues/2622)／[#2506](https://github.com/hat47x/tei-platform/issues/2506)／[#2593](https://github.com/hat47x/tei-platform/issues/2593) |
| EID-E：編集・実Effect | 二層表示を無変更保存、対象ID一意な変更、集約・重複・隠れた行・古いRevisionの変更を対照 | 無変更でUpdate 0、可逆PATCHだけ既存Intentに接続、全置換・部分失敗・Commit unknownは実Provider保証と別 | [#2500](https://github.com/hat47x/tei-platform/issues/2500)／[#2597](https://github.com/hat47x/tei-platform/issues/2597)／[#2598](https://github.com/hat47x/tei-platform/issues/2598)／[#1174](https://github.com/hat47x/tei-platform/issues/1174) |
| EID-F：Providerと異Domain | RDB系の実ProviderとAPI系（例OData）の異なるCapability/原子性を照合。同じ構造Roleで業務外（ゲーム/書籍/ロボティクス）の一件を表現 | 実Providerに不足する機能は不適合/未解決。合成Type一致を同一Effect・同一Host権限へ昇格しない。業務専用語彙の流入を測定 | #2622／[#2507](https://github.com/hat47x/tei-platform/issues/2507)／[#2390](https://github.com/hat47x/tei-platform/issues/2390) |
| EID-G：Studio/SHO/AIの消費 | 同じ固定Source+Role/Operator版の結果をCLI inspect、Studio標準表/Tree、SHO文書、AI Skillへ投影 | 各Consumerで解決した対象・範囲・開示を比べる。表示一致・構文PASS・成果物byte一致を意味受理/現在公開資格としない | [#1124](https://github.com/hat47x/tei-platform/issues/1124)／[#2516](https://github.com/hat47x/tei-platform/issues/2516)／[#2614](https://github.com/hat47x/tei-platform/issues/2614) |

上記は[ #2620 SY-01〜06 / AI-DSL-01〜05](https://github.com/hat47x/tei-platform/issues/2620)、[#2622 DO-01〜08](https://github.com/hat47x/tei-platform/issues/2622)を**再実装・再起票しない横断Consumer接続表**。新しいScenario SSOT、Role Engine、Evaluator、テストRunner、Plugin Runtimeを追加する意図ではない。

### 3.3 同一性の「比較対象」を固定し、勝手な合格を防ぐ

| 比較 | 同一性/適合の最小根拠 | 意味していないこと |
| --- | --- | --- |
| Source | 固定した原Source bytesと構文Plugin/版／入力契約 | 可読表層間のMeaning一致、受理済み業務意味 |
| Normalized / typed Meaning | Role/Operator ID/版、参照scope、評価順、NULL/Unit/Grain/時点/Effectと、受理が必要な判断の適用範囲 | 旧Reviewの現在有効性、実Target Authority |
| Target / actual Effect | Provider/Target/Hostの実Binding、Request対象とPayload対象、K/Receipt・独立Observation・現在の資格 | 旧OperationのResultを再生できること、業務過程の完了 |
| Consumer / Artifact | 適用した同版Artifact集合、表示依存、文書Occurrenceと公開先・現行開示Policy | 他Consumerの取得成功、意味同一、文書レビュー/正式Owner Acceptance |

反例：正規化結果が同じでも、別Source版でRole解決の根拠を失った場合、無条件に現在利用可能としない。逆にlayout、段落、表の見た目が違うだけなら意味差と断定しない。Source比較・意味比較・コード/効果比較・利用資格は異なるClaimであり、[#2505](https://github.com/hat47x/tei-platform/issues/2505)／[#2582](https://github.com/hat47x/tei-platform/issues/2582)／[#2604](https://github.com/hat47x/tei-platform/issues/2604)が既存Owner。

### 3.4 計画上の採否・実証・未決

- **採用済み前提**：三視点を単一の意味から投影する方針、TEI独自Hostを標準実行前提にしない方針、M3/M4操作面の導出・非対応理由の説明（[採用D-1〜3 / PR #2588](https://github.com/hat47x/tei-platform/pull/2588)）。AI-first・Headless-first・GUI-laterは別[Draft #2492](https://github.com/hat47x/tei-platform/pull/2492)での計画化が進行中。
- **この文書で採用を決めないもの**：標準データ型・純データ正規形のschema、Role名、DSL Grammar、Goa/MukuroL風記法、Markdown Profileの正式仕様、SmartDB一般化名、Providerの固定ABI、Self-describing bootstrapの完成規則。
- **最小の次実証**：最初に#2620のEID-A/EID-Bを**同じ意味Fixture**で試し、#2621のMarkdown表＋Expectedを一件接合する。その後#2622 EID-D/EID-Eと、既存#2586 Headless Consumerへ接続する。#2619は横断Coverage/不足と測定、#1119は製品S0/P0/S3/Gateの優先・証拠の統括。個別実装は各Ownerの独立PRへ置く。
- **記録する差**：解釈成功・型/意味一致・実評価・独立Expected比較・Target実Effect・外部Consumer取得・Owner受理を別々に証拠化する。計画表の存在・自作Fixture PASSから正式採用/本番動作を宣言しない。

**当追補の検証状態**：2026-11-02時点の公開GitHub Issue/PRと、既存の局所合成試験記録の照合・文書編集のみ。EID-A〜G新試験、実TEI Parser/Go、RDB/API Provider E2E、GUI、SHO保存・外部Skill、正式Owner意味受理は**NOT RUN / NOT ACCEPTED**。原典の非公開本文・固有ID・名称・locatorは本文へ掲載しない。
