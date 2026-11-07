# 参照モデルRelationship移植：Type固有構造・CrossView・証拠接合の実証計画（2026-11-02）

> **状態：RMT Relationship実証計画はmain統合済み（PR #2599、2026-11-03）。以下のケースは実証計画／新追補は提案。** 根拠の版・実行資格を分離する。原典固有情報はprivateに保持（#941）。
>
> **関係する正本**：[Critical Path §32](TEI_PRODUCT_CRITICAL_PATH_2026-10.md)／[横断設計Script-first main統合済み #2610](https://github.com/hat47x/tei-platform/pull/2610)／[#950 Source Adapter](https://github.com/hat47x/tei-platform/issues/950)／[#1126 ERモデル](https://github.com/hat47x/tei-platform/issues/1126)／[Source改版 Open #2615](https://github.com/hat47x/tei-platform/pull/2615)。本書の内容はOwner判断・正式Gate・製品資格を追加しない。

本書はCritical Pathの[§32](TEI_PRODUCT_CRITICAL_PATH_2026-10.md)における **S0 Source／包含範囲** と **S1 synthetic read-only** の実証項目を具体化する独立作業計画である。[main統合済み #2610](https://github.com/hat47x/tei-platform/pull/2610) が同じCritical Path本文の§33/34で横断設計・Script-first Studioの接合を担当しているため、そこでの節番号・編集経路を維持し、本書がRelationship固有の技術詳細を所有する。既存G0〜G4／S0〜S5、#1163の一改訂Evidence、#1174/#1129の実Effect P0、#267のOwner Acceptanceを維持する。実装順は外部要件の机上評価→意味・Plugin→CUI/標準Plugin並行→Headless Driver/Provider→GUIとし、[Draft #2492](https://github.com/hat47x/tei-platform/pull/2492)の提案と整合させる（Draftの採用・統合済みという意味ではない）。
> **状態：計画文書をPR #2599でmain統合済み（2026-11-02）。** 以下のRVケースの**実証・Owner意味受理は別途未完了**。本更新は2026-11-03の追加受入候補であり、原典固有情報はprivateに保持（#941）。
>
> **関係する正本**：[Critical Path §32](TEI_PRODUCT_CRITICAL_PATH_2026-10.md)／[横断設計Script-first PR #2610（main統合済み）](https://github.com/hat47x/tei-platform/pull/2610)／[#950 Source Adapter](https://github.com/hat47x/tei-platform/issues/950)／[#1126 ERモデル](https://github.com/hat47x/tei-platform/issues/1126)／[Source改版 Draft #2615](https://github.com/hat47x/tei-platform/pull/2615)。本書の内容はOwner判断・正式Gate・製品資格を追加しない。

本書はCritical Pathの[§32](TEI_PRODUCT_CRITICAL_PATH_2026-10.md)における **S0 Source／包含範囲** と **S1 synthetic read-only** の実証項目を具体化する独立作業計画である。[PR #2610](https://github.com/hat47x/tei-platform/pull/2610) も2026-11-02に独立の横断計画としてmain統合され、Critical Path本文の§33/34の検討内容はその計画の§1/§2へ移された。本書はRelationship固有の技術詳細のみを所有する。既存G0〜G4／S0〜S5、#1163の一改訂Evidence、#1174/#1129の実Effect P0、#267のOwner Acceptanceを維持する。実装順は外部要件の机上評価→意味・Plugin→CUI/標準Plugin並行→Headless Driver/Provider→GUIとし、[Draft #2492](https://github.com/hat47x/tei-platform/pull/2492)の提案と整合させる（Draftの採用・統合済みという意味ではない）。

### 観測済みの差分と判断の上限

- 固定main `9c9ef4d160640908e06ad13afb6c2b45c664c748` の `internal/xeadsource/facts.go` は `RelationshipType` と `ExistWhen1/2` の原文・明示空／属性欠落を保持する。一方 `diagnostics.go` blob `270de63791c9e76b04f230556542daca7e81ef12` は、Relationship Typeによらず左右のKey構成member数が違えば `relationship-key-arity-mismatch` とし、順序・型の比較前に `continue` する。
- 公開固定旧Modeler `DialogAddRelationshipOnDatamodel.java` の [L473–560](https://github.com/xead/XeadModeler/blob/68f1535b1c5dd80a327dfced37be401ecd3093c9/src/xeadModeler/DialogAddRelationshipOnDatamodel.java#L473-L560) には、対応Fieldの候補を探し、対象側のPK/SKがより長い場合に `FAMILY`、同数のFKなら `REFFER`、同数のPK/SKなら `SUBTYPE` を選ぶ経路がある。したがって **FAMILYの合法な非対称キー構成** と **真の対応不能・曖昧性** を区別する必要がある。旧GUIの全挙動・原典業務意味までは、この静的コードから受理しない。
- 原典に関する個別ID、locator、ZIP・XML内容、SourceRefは[#941](https://github.com/hat47x/tei-platform/issues/941)に従ってprivateに保持する。[#950 Type-aware対照](https://github.com/hat47x/tei-platform/issues/950#issuecomment-6557438352)、[#1126 設計Relationと複数Refer](https://github.com/hat47x/tei-platform/issues/1126#issuecomment-6557446867)、非公開Assurance #4の同版検収を技術正本とする。

### 同じ「対応」の二つのグラフを混同しない

| 別のグラフ | 頂点と候補辺 | 主張可能な範囲 | 別に必要な受入 |
| --- | --- | --- | --- |
| **G-V：同版CrossView** | 設計Relationship／Key構成・実装Refer／Alias・Field成分・TEI候補を別の頂点集合とし、候補辺にSource locator・役割・比較理由・不一致を付与 | 原典宣言1件→実装Refer 0/1/複数件の**候補**と情報保全状態。設計1件＝Refer1件＝Semantic Relation1件の前提を置かない | 各役割・成分対応のOwner意味受理、実CandidateのTarget参照実在 |
| **G-R：版A/BのSource対応** | 前版node集合A・新版node集合B・理由付き候補辺E。fan-out、fan-in、未到達、重複を区別 | 版内SourceRefの変動と、版間の対応候補を区別。各A頂点の候補1件でも、同じB頂点へのfan-inがあれば全体1:1ではない | Meaning revision、最新Context/Rule/Grant、旧ViewChangeやReceiptの再利用資格 |

G-VとG-Rは **異なる母集団・異なる用途** に対する証拠である。一方の辺を他方の確定したIdentityへ自動写像しない。実装Referの複数候補を単一の `target_ref` へ押し込まず、既存Source node／attribute atom・成分位置、候補の根拠とTargetの実参照を個別に追跡する。候補集合の数、Source node数、XML atom数、区切り成分数、実Candidate要素数、Owner Accepted Claim数は別のCoverage母数とする。

#2505版間差分との非重複を明確にする。G-Rの先行実測は[#2505 11/2独立Headless PoC](https://github.com/hat47x/tei-platform/issues/2505#issuecomment-6573011380)：合成Go 1.23.2 standaloneは三版A→B→Cの候補経路まで拡張され、**15 top-level test・race・vetが局所PASS**。対応候補の連鎖は意味上の同一性や旧操作の再利用許可にはならない。版間比較・Claim別適用・生成/公開の後段Consumerへの計画は[既存Open #2615](https://github.com/hat47x/tei-platform/pull/2615)が担当し、本節は**同版CrossViewとの異なる母集団・接合点**に限定する。**GitHub製品コード・Go 1.26系・正式H2/J5・実原典・Owner受理は未実証**。合成PoCをS0の正式検収器へ代入しない。

### 現行CrossView実装と、設計対実装Relation候補の実証範囲

固定mainの `internal/xeadsource/crossview.go` `BuildCrossViewProjection` は、`InspectWithContract` の後で **`FormatXead` のみを受け付ける**構造になっている。現在のCrossView実装はXEAD Task/Action/Function等の候補を投影するもので、**XEAD RelationshipとXEAF Runtime Referを一つの実行から照合した製品機能の証明ではない**。本書のG-Vは、両形式を別々にSource Factとして保持した後に検査する **同版対応グラフの設計・実証候補**とする。

P1の最初のHeadless確認では、固定した同一配布版に属するXEA[D]/XEAFの**原文SourceRef・構造Key・Field成分とRefer Alias/Role**をread-onlyで観測し、対応候補の理由・不一致・未解決を別々に記録する。設計Relation1件→Runtime Refer複数件の対応は候補辺の集合として保持し、実XEA[D]/XEAFのRelation対応が完全に自動解決できたとの主張を避ける。Raw bytes/probe/receiptの同一版条件とTargetの参照実在性はS0 P0-bの既存H2/J5に委譲する。

`attributeValue(localName)` による他の型付きSource Factへの外来namespace混入も、Refer固有の成功だけで安全とみなせない。[main統合済み #2603](https://github.com/hat47x/tei-platform/pull/2603)にTable/Key/Relationship/KeyFields全般の **Reader Known + exact attribute** を確認する合成テストを追加した。**追加head／統合headのCI資格は各々のexact commitに帰属し、この計画だけから全件PASSと推定しない**。最新headによる同種別・異種別の回帰PASSを確認してから、#950のsource-only品質判断へ送る。
`attributeValue(localName)` による他の型付きSource Factへの外来namespace混入も、Refer固有の成功だけで安全とみなせない。[PR #2603](https://github.com/hat47x/tei-platform/pull/2603)でTable/Key/Relationship/KeyFields全般の **Reader Known + exact attribute** を確認する合成テストを追加し、2026-11-02にmainへ統合済み（merge commit `1fd1ac214a83aa90e66ce0b5ee45264b3c05ca8b`）。最終head `7352ee76ad93529c1f02f61ba0b3d707ab7f4a21` のRepository Health、Go self-build、Wails smoke/PoC、依存監査が**同一headの5 workflowでsuccess**。この限定品質を原典の意味対応・Rule/Host・Owner Acceptanceへ昇格しない。

### 追加の横断結論：1対多Relationの「見える線」と「書き戻せる対象」は別

[#2610 §34](https://github.com/hat47x/tei-platform/pull/2610)のScript-first/generic Studio、[#2497](https://github.com/hat47x/tei-platform/issues/2497)の構造保存Projection、[#2500](https://github.com/hat47x/tei-platform/issues/2500)のView→Typed Intentを、このRelation移植の一事例に接続する。

設計Relationship1件に実装Refer候補2件があるとき、グラフを一つの線で示すことは**説明用の縮約Projection**として有用である。一方、異なるSource状態`s1`／`s2`が同じ表示`P(s1)=P(s2)`を持つなら、表示上の線だけから元の編集対象は決定できない。見た目が1本でも「2本の実装参照を一括修正してよい」という許可にはならない。

**可読性と編集提案の具体的な接続条件**：
- generic Modeler/Studioは、線と同時に**原典宣言、実装Refer候補0..N、Role、Source span、版、未解決の対応理由**を標準データ構造から参照可能にする。専用Relation画面の先行実装を要求せず、generic inspector／候補一覧／navigationを利用する。
- 表示の単一辺から**編集対象を逆引きできるか**は別の条件。元Source spanと型付きRole/Path occurrence、Base Meaning Revision、更新対象、編集意図が一意に対応するときだけ[#2500](https://github.com/hat47x/tei-platform/issues/2500)のTyped Edit Proposalへ渡す。1本の表示線が2候補を畳んでいるときはread-onlyで複数候補を表示し、利用者・Ownerが次の対象を判断する。
- Typed Edit Proposalの生成、Shared Acceptanceでの意味採用、Host現在認可、実Provider Commit／Receipt、実Consumerでの表示更新はそれぞれ別の資格。[#2505](https://github.com/hat47x/tei-platform/issues/2505)の版A/B候補が一意でも旧Intentを新版に黙ってrebaseしない。
- 旧Sourceの欠落・明示空などraw差分は内部Factと局所説明で保全する。公開一覧・外部AI/SHOには[#2516](https://github.com/hat47x/tei-platform/issues/2516)/[#1178](https://github.com/hat47x/tei-platform/issues/1178)の許可されたProjectionだけを渡し、候補の**件数・診断の有無**自体も非公開情報になり得る点を確認する。

この判断は**設計・受入条件であり、実UI/Putback実装成功ではない**。今後のCode/GUI/Ownerはそれぞれの既存Issueを使用する。

### Key順序のdiagnosticと操作別readinessの証拠接合（#1805 Closedの既存記録を再利用）

**成熟した判断**：[旧#1805の原因・影響と診断順序の照合](https://github.com/hat47x/tei-platform/issues/1805#issuecomment-6471603888)を、現役#950/#1126のSource-only／ER・Relation検証へ接続する。**Key自身の重複SortKeyをRelationship非依存に検出する候補パッチはmain未適用**（`diagnostics.go` blob `270de63791c9e76b04f230556542daca7e81ef12`）。既存 `orderedLegacyKeyMembers` を利用する改修の採否はSource担当が決め、ここでは事前受入条件だけを固定する。

**同じSnapshotでもConsumer操作で判定を変える**：
- **Jの原形read-only閲覧**：権限のあるprivate Consumerが同じexact Source版からJを一意に選び、Raw/Source Fact・Jの局所`RelationalFactResolutions`を読める。global `unresolved` は併記し、独立Jの表示を一律停止しない。原形を表示できること自体にlocal `resolved`を要求しない。J局所`resolved`は現行validatorでの構造準備度であってOwner意味Acceptedではない。
- **Jに基づく派生Relation/Graph・Function表示**：選択Jだけでなく、その表示で**実際に消費する参照閉包・Role/Path occurrence・Context／版**を検証する。単なる同名ID・隣接Graph・`diagnostics=0`で未検査のField対応や意味受理を推定しない。編集Targetが非一意なら[#1124/#2500](https://github.com/hat47x/tei-platform/issues/2500)の既存read-only候補選択に留める。
- **K/J全体の決定的lowering／成果物発行**：`RelationalResolutionStatus` のglobal `unresolved`／`partial`を対象全体の未完として扱う。局所Jの`resolved`で全体完成を代替しない。Source Inspection ReportのCLI `exit 0`は**集計レポート生成成功**であり、全体変換成功／業務意味受理／公開権利・実Effectではない。未受理の設計説明やSource観測資料を作ることと、正式な実行可能成果物を発行することは別。

**既存公開Consumerの制約**：`tools/reference-xead-source-inspect/main.go` は個別`RelationalFactResolutions`を`ByKind`集計し、source ID/locatorを公開しない。公開報告の「legacy-key resolved=1」から特定Jを逆同定しない。既存`diagnosticToken`はsource digest＋`RelationalResolutionPolicyRevision`＋診断ordinal由来の**レポート内相関**であり、原因診断の追加・arity短絡の見直しで同じ旧診断の順序・tokenも変わり得る。変更前後のBoolean（診断0/非0）・件数・code・順序・`byKind`・token・global/local状態を比較し、**判定算法revision更新要否**を審査する。公開schema v3の形状維持を意味互換の保証としない。Modelerの表示済みR10と更新後R11も別snapshotの評価結果として扱い、旧応答で現在表示を上書きしない（[#1124の既存Consumer境界](https://github.com/hat47x/tei-platform/issues/1124#issuecomment-6470746209)）。

**検証未実行に関する注意**：`diagnostics.go`はRelationshipのarity mismatchで`continue`しSortKey/型比較を**未実施**とする。左右の型情報が片方でも空なら`relationship-key-data-type-mismatch`検査を飛ばす。したがって個別診断が0でも「全ての比較条件にPASSした」とみなさない。FAMILYの許容され得る非対称形状を無条件の構造失敗にも、無条件の解決済みにも昇格しない。Source記録・型別の比較可能性・構造診断・Meaning Acceptanceは別々のEvidence欄に記す。

### 検証器自体の判別力：独立Expected・到達範囲・Consumer観測の三点照合

[#2514の独立検証基準と観測上の十分性](https://github.com/hat47x/tei-platform/issues/2514#issuecomment-6572161189) を、この移植作業に限定して適用する。診断の有無をExpectedとしてコピーする「実装と同じ実装での自己証明」は避ける。期待値は**合成fixtureの明示宣言、検証すべきClaim、公開旧Modelerの固定ソースにより説明可能な構造条件**から先に手作業で作り、現行`DiagnoseRelationalSourceFacts`の出力をExpectedに転記しない。旧ModelerのType分岐はSource構造の参照であり、業務列pairやOwner meaning Acceptedの検証基準ではない。

- **判定到達範囲の検査票（テスト用であり新製品enum/APIではない）**：各fixtureについて `input observed / precondition met / compared / rejected / skipped by prior check / no authority to decide` を明示し、該当する入力と理由を別列に書く。既存`RelationalDiagnostic`や`RelationalFactResolution`がこれら全状態を公開表現できるとは主張しない。特に`arity-mismatch`で`continue`したSortKey・型比較、および型情報が空でskipした比較を「一致」と数えない。
- **誤判定を見破れるか**：FAMILY非対称形状を常にrejectする誤実装、FAMILYをTypeだけで常にpassする誤実装、独立Keyの重複SortKeyをRelationship依存時にだけ検知する誤実装、Source属性の`missing`と`present-empty`を混同する誤実装を、それぞれ別の意図的Mutantとして仮定する。正例と負例の両方で区別できるExpectedを要求する。Mutantの実装／実行は**NOT RUN**であり、殺せると確認済みではない。
- **観測上の識別不能対**：同じSourceのK=局所unresolvedとJ=局所resolvedについて、公開`ByKind`件数は共通であり、Consumerの選択対象がKかJかを区別できない。公開集計だけを入力とする「Jの操作可否検証基準」は成立しない。同様に`ToKeyFields`の正規化配列`[]`だけでは、原属性の欠落と明示空を区別できない。元XMLのprivate raw declaration／`Node.Known`・exact属性、当該Factの局所Resolutionを必要範囲で参照し、公開Reportのsource SHAや件数を個別Factの識別子へ昇格させない。
- **範囲限定**：独立検証基準、Source一致、Owner意味確認、公開適格性、実Work H2/J5 Target参照は別の検証。実Go/GUIでのMutant試験や正式Acceptanceを新しく行ったものではない。#2514の検証基準／Conformance Profileと#950のSource Fact保持のOwnerを変えず、本書はRelationship固有のfixture入力・Expected・識別不能対だけを管理する。

### 2026-11-02 追補：Metamorphic検証とFunction実消費の境界

[#2514の独立検証基準とMutation感度](https://github.com/hat47x/tei-platform/issues/2514#issuecomment-6576646220)を、実行済み製品検収と区別しながらRelationship Source検証へ適用する。また[#2506の表示MappingとRule依存](https://github.com/hat47x/tei-platform/issues/2506#issuecomment-6576904883)、[#2586のQuery→Context→Claim](https://github.com/hat47x/tei-platform/issues/2586#issuecomment-6576783393)で、**同じ表示値でも成立するClaimと実際に消費した入力が異なる**ことが明確になった。本workstreamはそのRelation固有の圧力例だけを所有する。

1. **変換前後に保存すべき主張を限定する**：同じ二Fieldの`SortKey=010/020`を固定してXML上の出現順だけを交換すると、`SourceOrdinal`・source digest・raw locator等は変化し得る一方、現行のSortKey比較で確定する**順序付きField組とKey局所readiness**は、同じ独立Key・同じ検証規則のもとでは不変であるはず。さらに無関係Jの追加で公開`byKind`総数・Source SHA・`diagnosticToken`は変化しても、依存しないKの局所結果は変わらないはず。**何を不変とするかをClaimごとに指定**し、raw差分・公開tokenの不変までは要求しない。SortKeyが`"2"/"10"`なら文字列順と数値順の違いを別の独立Expectedで検証する。
2. **表示範囲とRule/Query依存を区別する**：同じRelation図・同じJの表示Fieldから、Relation pathのRole、非表示WHERE/EXISTS入力、Validation Rule入力、Action対象を復元できるとは限らない。FunctionごとのRoot/Relation Path/Field/Scope/Grainは明示Mappingで定義・追跡し、表示の簡略化から暗黙のRead Grantを作らない。部分取得で`exists` Witnessは確認できても`NOT EXISTS`・全件SUM・全件Validationは成立しない場合がある。入力不足は業務拒否（false）でなく未確定として記録する。
3. **宣言・物理取得・Materialize・開示の分離**：同じFunction要求/Result Slice/表示であっても、Providerが別Role/秘密Fieldを余計に読んだ世界とは**実Read Footprint**が異なる。Source/Design RMTで言えるのは要求候補の由来までであり、実Provider SQL/HTTP/read trace、同期snapshot、現在Principalの権限、Consumerへ実際に送ったbyte/fieldsはそれぞれ#2622/#2611/#2506/#2516/実Host担当が観測する。必要な補助Readが表示Field以外に存在しても、**Function Contextに明示されたRule/Query/Action依存とData Mappingの範囲内でのみ取得**できる。`DisplayFields == PhysicalReadFields`を強制しないことは、暗黙の取得を許可することではない。Sourceが適切でも過剰取得/開示が安全と自動確定しない。

本節は**計画上の判別条件**であり、定義・Data Context、Provider、Rule、公開・実Effectの新しいOwner、第二の検証基準、Runtime Host、製品enum/APIを作らない。固定Source/Policy/Function Context、Claim、独立Expected、検証到達範囲、実入力を試験票に記録する。Mutant/MetamorphicのTEI製品実行、実Provider/実GUI/Owner意味受理は**NOT RUN**。

### G-V候補0件の証明可能範囲と候補集合の変化（Claim相対／非採用）

[#2514のConsumer観測上の十分性](https://github.com/hat47x/tei-platform/issues/2514#issuecomment-6572161189)と[#2505の版間候補・Claim相対再資格化](https://github.com/hat47x/tei-platform/issues/2505#issuecomment-6572493377)を、**同一版G-V（設計Relationship→実装Refer候補）固有の列挙・選択**に限定して取り込む。[#2516の用途別Consumer確認](https://github.com/hat47x/tei-platform/issues/2516#issuecomment-6579615015)も参照するが、第二のEvidence/Authority体系を作らない。

**否定Claimの範囲を固定する**：設計Relation Rについて「候補辺0件」と「対応する実装Referが存在しない」は同値でない。候補探索の根拠として、(a) exactな設計XEAD・実装XEAFそれぞれのsource version/byte、(b) Reader/profile適合・外来namespaceとunknown/unsupportedの隔離、(c) 対象Table/Key/Role/Alias/属性成分の探索集合と**列挙完了範囲**、(d) 候補生成Rule/Policy revision・省略／打切り／非公開の有無、(e) 当該Consumerが見られる証拠を記録する。**対象集合の完全性が検証できる場合だけ「そのSource版・規則・scopeでは一致候補0件」**と言える。これでも原典に「業務的Relationが存在しない」ことは証明せず、実施した候補照合規則で発見できなかったことの限定的な主張である。未探索・アクセス不可・抽出未対応なら `not observed / incomplete / undisclosed` と説明上区別する（製品state型を新設しない）。private candidate数が秘匿されるConsumerにpublic `0`を返す設計は採らず、開示資格に応じて非表示・判断保留にする。

**Open-world候補増加を再検証の契機にする**：同一Source byteでもRole照合規則・候補列挙範囲の拡大によって、RのRefer候補集合が `{a}`→`{a,b}` に変わり得る。「候補が一意だった」という旧**機械的仮説**を新版の確定対応として引き継がず、Source/Policy/探索範囲を固定した当時の説明と現在の候補集合を並置する。逆に、以前Ownerが明示的に受理した「特定aを対象とする意味」は、新候補bの発見だけで歴史的に取り消さない。**現在の編集・生成・Actionへ受理済み証拠を適用できるか**は、選択aのTarget/Role/MeaningRevision、最新Sourceと現在Authorityの再照合で判断する（[#2500](https://github.com/hat47x/tei-platform/issues/2500)・[#2505](https://github.com/hat47x/tei-platform/issues/2505)へ委譲）。一つの描画線から複数参照を自動編集しないRV-E1を維持する。

これはRMT専用の**合成検証計画**である。現行`internal/xeadsource/crossview.go`の`BuildCrossViewProjection`がXEAD-onlyである事実は変わらず、XEAD↔XEAFのG-V列挙・候補集合増加・Current Owner判定は**製品機能として未実証**。実Work H2/J5、旧GUI、正式Owner受理・公開/配布・外部Harness実消費もNOT RUN。

### 2026-11-02 追加：Relationを利用するClaimの有効期間と、生成成果物の実消費を分離する

**動的依存（#2582/#2586）**：参照Relationを確認しただけでは、Functionがその後も同じRule入力を読んでいるとは限らない。表示を変えない `Order O1.amount=600` のまま `customerRef=C1→C2` が起きると、Ruleが消費する `Account.creditLimit` の対象はA1→A2へ切り替わり得る。これを単なるField値の差分やGraphの表示差として扱わず、**実際に評価したRelation occurrence→Binding→Predicate/Collection→Result Slice→Rule/Operator→Claim**の同一Run証拠を照合する。0件の`NOT EXISTS`は、既存RowIDがないことと依存がないことを混同しない。Predicate/Scope・Provider Read Cohort・列挙完全性・新規加入の変更観測を別に置き、`0件complete`と`0件partial`を別判定する。合成Python 12/12の報告は[#2582](https://github.com/hat47x/tei-platform/issues/2582#issuecomment-6579762087)の局所実績であり、本書やTEIで再実行したものではない。

**依存閉包と変更窓は独立**：採用されたRuleで必要なRelation/Value/Predicate/Role全てを把握していても、前回Readから現時点までのChange Witnessに欠落・Lag・Provider毎の未被覆区間があれば、旧Claimの現在資格は保持できない。[#1125のCWT](https://github.com/hat47x/tei-platform/issues/1125#issuecomment-6579870075)は既存Ownerの証拠であり、RMTのSource Fact・`Source SHA`・`Meaning Revision`や`Materialized Slice`から、実変更通知の完全性を自動推定しない。現行Valueが元へ戻ったABAでも「現在値一致」と「その間不変」は異なる。用途別にread-only表示、Rule Claim、Operation/EffectとSHO説明の失効範囲を区別する。

**生成・納品（#2604/#2630）**：一体モデルをAcceptedとした後にも、生成したLibrary・SHO文書・Manifestが同版で揃うこと、公開が途中で壊れないこと、各Consumerが必要集合を**実際に読んだこと**は別の事実。main統合済み[#2587](https://github.com/hat47x/tei-platform/pull/2587)の`tei artifacts -workspace reference/standard-crud/workspace -application ledger -out <fresh-dir>`は現行CLIの観測入口として利用できるが、そのファイル書き出し経路は個別ファイル→Manifest順のため、原子的な一式公開・Readerへの実同一世代到達まで示すものではない。書式だけの変更、Type/Ruleの意味変更、Producerの変更、公開ファイル欠損、Consumer側の旧世代読取を、用途別の再生成・再評価・再配布で区別する。検証責務は[#2604](https://github.com/hat47x/tei-platform/issues/2604)/[#2630](https://github.com/hat47x/tei-platform/pull/2630)/[#2482](https://github.com/hat47x/tei-platform/issues/2482)へ渡し、本書で第二のPublisher/検証基準を作らない。

この接続は既存Issueの**試験入力と引渡し条件**であり、正式Reactive Engine、共通Changefeed/Host、生成物の新しい権限Schemaを導入しない。TEIの実Query・Provider変更追跡・Artifact Reader・SHOレビュー・Owner受理は**未実行**。

### 2026-11-03 追補：Data Script／StateのSource-faithful移植（RMTの受渡し境界）

他レーンの[#1129 Data Scriptの凝集・DRX-01〜05](https://github.com/hat47x/tei-platform/issues/1129#issuecomment-6587145160)、[Script IFを基本にしたAuthoring選択](https://github.com/hat47x/tei-platform/issues/1129#issuecomment-6588906800)、[本体属性を既定とするState Storage方針](https://github.com/hat47x/tei-platform/issues/1129#issuecomment-6589126361)、[#2622 Provider別実現](https://github.com/hat47x/tei-platform/issues/2622#issuecomment-6589134587)を、**既存RMTが追加で失ってはならないSourceの区別**として接続する。これらは設計・Owner方針の記録であって、X-TEA原典の特定Data Script動作を実測した証拠ではない。**X-TEA Table Script/Validation/Derivationの完全な構文・実行タイミング・遷移意味は本書では未照合**。private原典の引用・転載を本公開計画へ加えない。

1. **Source保存と意味抽出を分ける**：原文のTable/Entity/Field・Key・状態属性の宣言、Scriptのraw body、Attachment先、呼出し元、source locator、版、明示された更新契機と処理順を、取得・解釈可能な範囲で別々に記録。Data Script候補を Pure Derive／Validation／状態Guard・Transition／Before-write・in-Tx／After-commit の**分類候補**へ分け、根拠となるSource spanと不明範囲を残す。静的に理解できないDynamic Call、外部Effect、更新契機は`opaque/unknown`であり、全Rule抽出・自動Validation成功・暗黙のListener登録を主張しない。状態Fieldが変更されたという事実だけで任意のBatch/外部Effectが**Commit後に一回実行された**とは推論しない。
2. **二重正本を禁止し、投影を限定**：移植先のData Scriptを唯一の業務Rule Sourceとする案では、Function/UI/API/Batchは同じRule ID/版へ接続し、IFから得た状態遷移図は**read-onlyかつCoverage付きの部分Projection**。明示Transition定義を選択する場合だけその定義を単一正本へ移し、Guard/Effectを別IFへコピーしない。どちらのSourceを正式に採用するか、Validation対象集合と空集合の扱い、Ownerが受理した状態・条件は別判断。元Sourceのコメント・手順的記述を自動でAccepted Meaningに変換しない。
3. **RelationshipとWorkflow Identityの境界**：同じ設計Relation／旧Keyから、Business Subject＝`DataType/Role/tenant + 型付き複合Key`、Workflow Instance＝`Process definition/version + attempt`、State Occurrence＝`active token/revision`の一対一を推定しない。同じ`orderNo`でもtenant差、同じSubjectの再申請、並行Active state、キーの0埋め／NULL／桁・型差が反例となる。元のKey/SortKey/Field構造はSource Factとして保持し、Subject Binding／Storage Bindingは[ERの#1126](https://github.com/hat47x/tei-platform/issues/1126)・[#1129](https://github.com/hat47x/tei-platform/issues/1129)の別のMeaning/Provider検証へ渡す。
4. **物理状態StoreをSourceから決め打ちしない**：本体status・version（必要なら複数Field/JSONB/ARRAY）が第一の検討候補だが、State Instance 1:N・History・並行token・権限/寿命の別管理が必要ならsidecar/外部Store等を比較する。旧Buri型中央Schemaや単一数値/文字列キー結合をTEIの必須形状へ移植せず、横断検索の要求を中央Writer必須とも扱わない。RMTは**どの旧Source属性・Scriptがどの意味候補を支えるか**までを担当し、生成SQL、条件付きCAS、同一Tx、Outbox、ACK-loss後のReceipt、実DB Read/Writeは#2622/#1174/#1129の既存実装・試験Ownerへ渡す。

**ケースごとのExpectedの独立性**：公開合成の原Source例と、そのFixture Authorが別に手書きした`Source Fact expected`／`意味候補に必要な未解決項目`を用意する。旧Scriptの業務的意味やイベント順序を実測せずにExpectedとして書き込まない。実Sourceが手に入らない場合は`NOT OBSERVED`、Source解釈不可は`OPAQUE/UNKNOWN`、必要なデータを取得できないRule Claimは`UNRESOLVED`であり、拒否(false)・処理成功と混同しない。UI/図とGo生成・実Effectを同じPASSで表示しない。新しい共通StateMachine Core、Central Workflow DB、第二Rule Engine/Host/Registryは本計画に追加しない。

### Case別に失敗原因と検証資格を固定する（本表の28統合ケースは未実行）
### 2026-11-03更新：Source資格・明示Mapping・現在Grant・成果物実読取の同一試行

**変更前提の固定（main `a4840b1a54d94f81f2f107c38968605232cd4c66`）**：Source Factの名前空間/欠落・明示空保持（#2603）、横断設計（#2610）、明示Function Mappingの合成検証計画（#2624）、Load時byteに束縛したHost署名入口（#2642）、Derived SelectのProvider呼出し前と返却前のField Read再検査（#2637）は、それぞれ**2026-11-02にmain統合済み**。これらを一体の移植E2E完了とみなさない。旧XMLの実Source対応とTEI Workspaceの署名済みSource Indexは**異なる対象**であり、同じdigest/IDという記法だけで同一原本を推定しない。

**取得境界を確定**：Functionが必要とするDataは、表示、検証Rule、Query Filter/Sort/Existence、Action前提、例外的な補助取得を含め、**Function Context内の明示的Data Mapping**から収集しなければならない。Modeler/Studioの提案はMapping補完候補であって、Ownerに無断で暗黙のJoinや未宣言Field/RelationをProviderへ読み出してよいことにはならない。Role/Path/Field/Scope/Grainと選択Physical Mapping（列・Store・版）の両方を確認し、利用目的とGrantを交差させる。SQLはモデルでは表せない例外的処理に限定し、SQL記述がData Model/Meaning Acceptanceの代替にならない（#1126/#2506/#2624/#2611）。実Providerからの物理Read footprintの計測は#2622等の担当へ委ねる。

**ロード・承認・読取を混ぜない**：`workspace.BuildSourceIndex(root, result)` はRead-only解析用source projectionで、**正式`Load`結果に捕捉済みのParsed/Managed fileなら再読せず保持byteを使う**。Standalone／部分Result等で未捕捉Pathがある場合はfilesystemへfallbackし得る。`BuildSourceIndexFromLoad` はLoad時の対象byteしか使わず、未捕捉パス・Load snapshot欠落・imports宣言を拒否するHost署名経路（#2642）。ただし**原子的な複数ファイルsnapshot／workspace manifestの束縛／外部Xead ZIPとTEIモデルの正しい対応**を証明しない。別途#950/#1121のH2/J5現物、#267 Owner Source Acceptance、#2482/#2604の生成/公開と#2630のConsumedMembersを確認する。

**Read認可の時間的境界**：mainの`internal/driver/derived_query_execution.go`は、`query.ValidateDerivedSelect`をProvider実行前と返却直前に呼び、許可失効時にBrowse+Derived値の返却を拒否する構成（#2637）。これは**当該同期コードパスの返却前fence**であり、Provider自身による未宣言Field取得、実ネットワークへの先行送出、同時Grant切替（ABA）、実Tx snapshot、他Consumerへの発行、Row-level RLSを保証しない。Function Data Mapping適合、Actual Read、Current Grant、Consumer Disclosureを独立証拠として照合する。

**最低一件の縦断受入票（新Gate／Schemaではない）**：既存S0/S1/S3・P0-a/P0-bを守り、同一Scenarioの記録で次を順に固定する。
1. **原典入口**：固定XEAD/XEAFの原byte・Profile・`Known`とraw/normalized Fact・独立Expected（#950）。設計Relation1とRefer候補0..N、対応不確定を明示（G-V）。formal Source→TargetのJ5解決が未実施ならそこだけunknownとする。
2. **TEI意味入口**：提案されたEntity/Relation/Role/Path occurrenceとOwner Accepted MeaningRevision、TEI WorkspaceのLoad時Source Index／必要なCurrent Applicabilityを独立に固定（#267/#2642/#2505）。元Xeadの原文照合からTEI Source Indexの同一性を勝手に導かない。
3. **Function実消費**：明示MappingのRole/Field/Scope/Rule/Query/Action依存、選択Physical Mapping、Provider実Read・完全性・Cohort、関係再束縛／否定依存／Gap/ABAの変更観測を独立に記録（#2506/#2622/#2582）。未宣言Readは合成例で禁止し、実際のRead証拠がなければ「検証済み」としない。
4. **権限・実効果**：Provider呼出し前／結果返却前のHost Read Grant、Operation対象/ExpectedVersion、書込前の現Grant、実Commit/Receipt/独立Observation（#2637/#2598/#1174）。一地点のAllowやRequest hashだけでは未来の再利用資格としない。
5. **生成・利用**：同一意味・Source/Generator版のLibrary/SHO用Artifact、Manifestの構造内整合、Published set、Consumer別Required set、実際のLoaded file bytes・公開資格（#2482/#2604/#2630/#2516）。Consumer用途別に未充足ならdelivery/completionを保留する。

この縦断票は**設計上の同一Scenario記録形式案**。既存Product CLI/検証器を再利用し、独立Oracle、原典の機密/権利境界、Owner受理、実H2/J5・実Provider/DB/SHO/AIへの送出の実行結果を捏造しない。**これら5つの段を同一RunでE2E検収した実績はまだない。**

### 2026-11-03 追補：FAMILY Source形状の限定実装と実SQL Mapping Witness

**旧Modelerの作成条件と型別診断**：[公開旧Modeler`DialogAddRelationshipOnDatamodel.java`](https://github.com/xead/XeadModeler/blob/68f1535b1c5dd80a327dfced37be401ecd3093c9/src/xeadModeler/DialogAddRelationshipOnDatamodel.java)は、source側Key成分がtarget側のより長いPK/SKの部分集合であり得る場合に`FAMILY`のRelationshipを作る。このコードだけで**任意の既存FAMILYのField対応**は特定できない。Source Factを保持した[#2603（main統合）](https://github.com/hat47x/tei-platform/pull/2603)の次段として、[Draft #2670](https://github.com/hat47x/tei-platform/pull/2670)を**Source-onlyの限定修正候補**として起票した。

**診断の局所変更案**：固定main `2e16b2af9531225b73430a32a6dbb2031e07f4c1`の`diagnostics.go`が発した一律`relationship-key-arity-mismatch`を、`FAMILY`かつ**左成分数>0、左<右、右Key Type=PK/SK**のときだけ`relationship-family-correspondence-unresolved`へ差し替える。この診断は、次数差の存在を一律欠損扱いしないが、**候補対応が未特定であるためlocal/global unresolvedは維持**する。既存の`REFFER/SUBTYPE`や右側がFK／逆向きはarity mismatchのまま。FAMILYでも`orderedLegacyKeyMembers`を両側評価し、右Keyの重複/空`SortKey`原因を別診断に出すが、**対応するFieldの位置比較・型比較は実行しない**。原XML子順だけで一致させない。

**版と診断公開**：候補パッチは`RelationalResolutionPolicyRevision`を`v1→v2`とする。公開Inspection Reportの`ByKind`・診断Code/件数・ordinal/`diagnosticToken`・Resolution/Statusは、旧v1と新v2で**互換不変と仮定しない**。CLI report生成exit 0は診断0やOwner意味受理でない。レビュー時は同一Raw fixtureを両revisionで比較し、既存[#2514](https://github.com/hat47x/tei-platform/issues/2514)の独立Expected、[#2516](https://github.com/hat47x/tei-platform/issues/2516)の公開範囲を使う。単独Keyの順序診断と元原典のField pairing/GUI/Target実解決はなお別タスク。**#2670のmain統合・CI/Owner確認は、本計画更新時には未確定**。

**SQLによる明示Mappingの異質な反例**：[Draft #2665](https://github.com/hat47x/tei-platform/pull/2665)はmain統合済み[#2624](https://github.com/hat47x/tei-platform/pull/2624)の合成Rule/Mapping対照を、Python/SQLiteの実SQLと`SQLITE_READ` callbackの比較へ拡げ、局所14件PASSを報告している。これは**SQL compile時の参照列**であって、Storage Page/Rowの実読取、RLS、Host認可やネットワーク送出の証人ではない。本書のRV-M1では、Function Contextの**Resolved Data Mapping**を設計上の許容Read集合とし、`SQLITE_READ`の要求列集合を独立Witnessとして照合するが、両集合一致だけでProvider実Read Footprintの同値を確定しない。表示同一の`SELECT *`＋Client側秘匿と最小列SELECTを正負対照とする。二ProviderのCohortや変更通知、SHO/AI向けDisclosureはさらに別資格。#2665の14件を本計画の29件実行済みへ算入しない。

### 2026-11-03 追加検証：参照されないKey、公開Envelope、Commit後の因果可視性

**独立Keyの潜在的な過少診断（#950/#1805）**：固定main `a5cb7c4d58b5cf42d1c9c73810f86398987ae32c` の `DiagnoseRelationalSourceFacts` は、Keyの存在/Field参照を検証する一方、**複合Keyの`SortKey`重複・欠落による順序一意性検査をRelationship経由の`orderedLegacyKeyMembers`呼出しに依存**する。`ProjectRelationalFactResolutions`はKey局所診断を基に`resolved`を作るため、**どのRelationshipにも参照されない複合KeyでSortKeyが重複していても、現行構造診断0・局所resolvedになり得る**。これは観測されたGoの静的制御経路からの反例であり、実TEI fixtureでの再現は未実施。別正常Key Jを同じSnapshotへ置き、Kの構造不備がJの原文閲覧を不当に妨げず、全件決定的loweringはKを含むため保留するかを試験する。[Draft #2670](https://github.com/hat47x/tei-platform/pull/2670)は**FAMILYに参照された両側Keyの順序**を検査するだけで、このK単独の問題は修正していない。同一担当#950内で別の限定修正が必要となる場合、Public診断Code・PolicyRevision・局所/全体ステータスの移行をセットで判断する。文字列`SortKey`の辞書順／数値順のどちらが旧仕様かも原典根拠を採ってから決め、無断の並べ替えをしない。

**Artifactの二層Digestと検査の到達範囲（#2482/#2604/#2630）**：[main統合#2633](https://github.com/hat47x/tei-platform/pull/2633)はAC-01〜04を不備の再現として特性化し、[Draft #2659](https://github.com/hat47x/tei-platform/pull/2659)が実`Artifact.Verify()`・Surface schema・Gap理由を`CheckSet`で検査する修正候補。**#2659のhead `10ad24748308eb47152050411b52e6edce614311` はRepository Health・self-build・dependencyの3 workflowがsuccess（2026-11-03照合）。ただし未mergeで実公開Reader/Owner受理は別**。[Draft #2662](https://github.com/hat47x/tei-platform/pull/2662)／[#2604 AP-10/11](https://github.com/hat47x/tei-platform/issues/2604)は、`Artifact.Digest`が`Content`だけではなくGenerator/Rule/Unresolved等のEnvelopeに束縛される一方、既存`tei artifacts -out`で出力したBodyとManifestには**外部ReaderがEnvelopeを再構成するための全情報がない**ことを指摘した。よって`CheckSet`内部PASS、Public manifest、DIR内のBody一致、実Reader consumed byte、Consumer要求Member網羅、SHO/Libraryの同版受領は別判定。[PR #2673](https://github.com/hat47x/tei-platform/pull/2673)のCLI `-check`や`-out`は検査・出力の入口候補だが、**検査終了コード0を公開一式の原子性/Consumer検収へ昇格しない**。必要なSource/Generator/Envelope証拠を、既存Ownerが物理Readerで比較可能な形で固定するまでは、第二Artifact Storeや新共通Publisherを採用しない。

**Commit receiptからOwner完了への非含意（#1125/#2671）**：[Draft #2671](https://github.com/hat47x/tei-platform/pull/2671)の合成15ケースは、受理済み業務意味→実Request（U）、Commit/Receipt（K）、後続Readが**当該Commitを含む**保証（V）、Postimage実観測（O）、Owner業務完了（B）、Consumer開示（D）を分離する。例えばRead値が期待値と一致しても、Replica lag・事前同値・他Writer補正・別Provider route/Read時点によって**Kへの因果証明にはならない**。このため、Source→Target同版確認済みのRelationとFunction Mappingを用いても、業務Actionが実現・完了したとするには、既存#1174/#1129のEffect/Txと#1125の実Owner判断が別途必要。合成15 PASSは同一Run実DB/HostのK/V/O/B/D PASSではない。

**検証結合**：この3件は既存S0/P0-a/P0-b/Consumerの**未取得Evidenceを正確に示す追加の比較票**であり、G0〜G4やOwnerのAccept/Release境界を増設しない。独立Expectedは生成器／検証器が書き出した`want`から独立に用意し、Fixtureで真として与えたGrant／Receipt／Read Cohortを「実測」と呼ばない。下面RV-K5/P3/O4は**計画、NOT RUN**。

### P0-a着手用：Draft #2670のv1→v2診断差分をexact-headで検収する

[既存#950のFAMILY Source構造責務](https://github.com/hat47x/tei-platform/issues/950#issuecomment-6585872832)と[Draft #2670](https://github.com/hat47x/tei-platform/pull/2670)を、**実装候補の採否を判断できる作業票**へ具体化する。本節は既存RV-F1/F2/R1/K1/K4/O1とRV-K5の**実施順序**であり、比較ケースの増設ではない。

**同一入力・異版の比較条件**：まず#2670のbase commit `2e16b2af9531225b73430a32a6dbb2031e07f4c1` のdiagnostic policy v1と、検証するPRの**exact head**（2026-11-03初回観測 `5d3cd1bf6735c2f9068e1e4018091e0f47747a3a`、v2候補）を別々に固定する。同一の公開合成XEAD byte、Reader/Classification/Transcription contract、実行Go版、公開Report入力を比較し、head更新時にはテスト・CI結果を再取得する。`git diff`で変更が既存Source診断とそのテスト・policy revision・公開レポート検証に限られることを先に確認する。

| 固定fixture | v1側の期待 | v2候補側の期待・別途維持する不変条件 |
| --- | --- | --- |
| FAMILY／左1・右PKまたはSKの2成分、右SortKey一意 | `relationship-key-arity-mismatch` | `relationship-family-correspondence-unresolved`。両版ともlocal/global `unresolved`。右候補Fieldが型上一意でも**Field pair・業務意味はAcceptedにしない** |
| 同FAMILYで右SortKey重複または空 | arityで`continue`、当該Relationshipの順序検査は未実施 | family対応未解決と右`relationship-key-order-unresolved`を別々に記録。構成の型・位置比較は依然未実施 |
| REFFER/SUBTYPE/unknown、FAMILY右FK、FAMILY逆方向 | `relationship-key-arity-mismatch` | 同コードを維持し、FAMILY限定例外の拡大を拒否する |
| 独立複合Key KのSortKey重複と正常J | Key単独順序診断の不足を限定観測 | **#2670では解決しない**。改善試験は既存RV-K5へ委譲し、今回のv2成果へ誤算入しない |

**一つの検証Attemptで残す項目**：`raw fixture digest / git base・head / 読込contract・policy / private Source FactとRelationalFactResolutions（対象Key・Relationship） / 診断code・side・数・先行short-circuit・型比較実施範囲 / global status / public schema v3・ByCode・ByKind・diagnosticToken / 独立Expectedの出典 / Go test・vet・GitHub exact-head workflow run`。公開データには元ID・Locatorを出さない。**Source digestが同じでも判定rule revisionが異なる**ため、公開tokenや診断件数が不変であることを合格条件にせず、公開出力形状の一致を診断意味の同一性と扱わない。

**P0-a合否と次段の停止**：Source-only patchの評価は「構造上の異形と対応未確定を区別できる」「無関係Jの局所結果を汚染しない」「公開Reportでprivate原典が漏れない」「独立Expectedとexact-head実行が照合できる」まで。これがPASSでも、G-VのXEAD↔XEAF対応、正式H2/J5、実Provider Read、Target関係／FK、OwnerのField対応・Meaning Acceptance、成果物同一世代の実Consumer読取は**別資格**。実Go/CI/正式実体の再実行なしに計画へPASSを記入しない。新Host、別Oracle、独立Key診断の#2670への混入を要求しない。

### Case別に失敗原因と検証資格を固定する（本表の32統合ケースは一体実行未完了）

| ケース | 入力・判別条件 | Headlessで照合する期待／禁止事項 | 担当 |
| --- | --- | --- | --- |
| **RV-F1** | 合成FAMILY：親Key1成分、子PK2成分。親と対応し得る子成分が一意 | 原文Type・Key全member・余剰member・SortKeyを保持。Draft #2670の条件に一致する場合はarity-mismatchでなくfamily-correspondence-unresolved、local/globalは未解決のまま。対応Field/意味受理は別 | #950／#1126 |
| **RV-F2** | 同FAMILYで対応候補が0件、または複数件 | 長さ差と対応不能・曖昧を別理由で提示。型・名前・Aliasの一致を無断の確定対応へ用いない | #950／#1126 |
| **RV-R1** | REFFER 1→2、対照として同数2→2／SUBTYPE | Type固有の構造規則を比較。合致する次数から業務Relation・物理FK生成を自動受理しない | #950／#1126 |
| **RV-K1** | FAMILY子側のSortKey空・重複・辞書順と数値順の差 | Draft #2670はFAMILY拡張時も左右SortKeyを独立診断する候補。独立Key（Relation非参照）の順序原因は未対応で別。arityによる未検査を成功と扱わない | #950／#1126 |
| **RV-K2** | 同一合成snapshotで独立Kは複合SortKey重複、Jは参照・依存を持たない正常Key。Jの原形閲覧、Jの派生説明、K/J全体loweringを同順に要求 | 独立Key診断候補の適用時はK局所unresolved・J局所resolved・global unresolvedを期待。Jのread-only原形は保持し、派生説明は消費閉包、全体loweringはglobalで保留。公開`byKind`件数からJのIDを逆特定しない | #950／#1126／#1124 |
| **RV-K3** | REFFERの2→2構成で両側SortKey一意、型情報も両側同じ、XML出現順とSortKey順は異なる。対照として片側DataType欠落 | 構造診断0・局所resolvedがあり得ても列pair/Business identityはOwner未受理のまま。欠落型を比較済みとしない。source宣言順・SortKey字面・Validatorが判定した順・意味採用を混同しない | #950／#1126／#267 |
| **RV-K4** | 同一Source byteを旧/候補の判定規則で比較。独立Kのみの0→非0と、既にR1/R2影響がある非0→非0を別fixtureで実施。FAMILY arity分岐も対照 | 診断Boolean/total/byCode/ordinal/token/global/localとpolicy revisionを差分表に固定。影響件数はarity短絡で変わるため「原因1＋順序影響2」を一般化しない。既存Report CLI exit 0をrelease/meaning PASSとみなさず、旧版判定結果の無言再利用を禁止 | #950／#1126／#2514 |
| **RV-N1** | 外来`x:Refer`／`x:Relationship`／`x:TableKey`／`x:Key`／`x:Table`、通常`Refer`上の`x:ToTable`／`x:ToKeyFields`等の属性類似名、外来Tableの子にある通常Referを対照 | Readerの`Known=false`、修飾名、局所Fact、親Table文脈の伝播を独立に調べる。外来宣言・外来親IDを標準Fact/Relationへ昇格させない。既存[#950 RMP-048/RML-023](https://github.com/hat47x/tei-platform/issues/950)が一般契約。**#2603の先行Refer限定headはCI成功の記録がある**。追加headではXEA[D]の`Relationship/TableKey`、XEAFの`Key/Refer`と外来Tableの入れ子Scopeを合成テストへ拡張したが、**統合headのCIと対象Source/Caseの同一性は別途判定**。外来宣言を元のField・Tableの意味受理へ転換しない | #950／#1126 |
| **RV-K4** | 同一Source byteを旧/候補の判定規則で比較。独立Kのみの0→非0と、既にR1/R2影響がある非0→非0を別fixtureで実施。FAMILY arity分岐も対照 | Draft #2670の診断Code変更・SortKey複合診断・policy v1→v2を実験対照に追加。Boolean/total/byCode/ordinal/token/global/localを比較し、原因件数を一般化しない。CLI exit 0で意味PASSとせず、旧版判定結果を再利用しない | #950／#1126／#2514 |
| **RV-K5** | 同じXEAD Snapshotに、独立複合Key K（SortKey重複・欠落）と正常Key Jを配置。どのRelationshipもKを参照しない。KのField ID自体は有効 | 現mainはKの順序診断がRelationship経由のみなので、K局所resolved・global resolvedを誤表示し得る経路を静的に特定。独立OracleでK局所unresolved・J局所resolved・global unresolvedへ変わること、Jのread-only原形が利用できることを検証する。Draft #2670をK単独修正済みと呼ばない | #950／#1805／#2514 |
| **RV-N1** | 外来`x:Refer`／`x:Relationship`／`x:TableKey`／`x:Key`／`x:Table`、通常`Refer`上の`x:ToTable`／`x:ToKeyFields`等の属性類似名、外来Tableの子にある通常Referを対照 | Readerの`Known=false`、修飾名、局所Fact、親Table文脈の伝播を独立に調べる。外来宣言・外来親IDを標準Fact/Relationへ昇格させない。既存[#950 RMP-048/RML-023](https://github.com/hat47x/tei-platform/issues/950)が一般契約。**PR #2603でXEADの`Relationship/TableKey`、XEAFの`Key/Refer`と外来Tableの入れ子Scopeを含む合成テストが最終headでCI 5 workflow success、main統合済み**。Reader層から外部Hostまでの総合Security適合は別証拠。外来宣言を元のField・Tableの意味受理へ転換しない | #950／#1126 |
| **RV-X1** | 固定同一配布版のXEAD Relationship・XEAF Runtime ReferをSource Factとして独立読取。設計1件→実装候補0／1／複数、同型でもAlias・Role・Field成分が異なる対照 | 現行CrossViewはXEADのみ対応という実装境界を維持。候補グラフG-Vをread-onlyで表示し、原因・多重対応・未解決・元byte/候補Target参照を別出典で照合。次の意味受理・物理FKへ自動昇格しない | #950／#1126／#1031／private #4 |
| **RV-X2** | 同じ設計Relation R。実装Refer候補0件の対照：(A)固定XEAF全対象nodeを列挙・profile適合・Role/Key条件で照合済み、(B)Reader未対応/探索打切り/一部scope非許可で未観測あり、(C)存在を知るが外部Consumerへ候補情報を開示できない | Aのみ「固定版・探索規則・許可scopeにおける一致候補0」を限定主張。Bは未探索を実在しないと扱わず、Cは非開示をpublic `0`へ変換しない。候補数・source node/atom/属性成分・unknown・列挙完了/非公開の各母数を区別し、業務Relationの非存在・意味Acceptedをいずれも推論しない | #950／#1031／#1126／#2514／#2516 |
| **RV-X3** | 同じexact Source byte・設計Relation Rで、探索Rule/対象Role範囲を拡張し候補集合`{a}`→`{a,b}`。旧UIはaを選択済み。別枝ではaに明示Owner判断履歴あり | 旧「候補一意」という導出仮説を新しい列挙規則の現行一意性に昇格させない。aをTargetとしていた歴史的Accepted判断はbの発見だけで自動取消しないが、**現在のProposal/Putback/発行**はpinしたSource・Policy・候補集合・Role/対象・MeaningRevision・現在権限の適用を再照合。見た目の一線/SourceRef一致だけでbを無断採用しない | #1126／#2500／#2505／#2514／#1124 |
| **RV-C1** | 設計Relation1宣言／runtime Refer2候補、各々別Role・Alias | Source1／Runtime2／対応候補2／Accepted0（synthetic開始時）を別々に数え、Diagramと辺一覧でも区別。自動分割・合成を行わない | #1031／#1126／#982 |
| **RV-C2** | 複合キー・Referの1 XML属性に複数Field成分。属性なし／明示空／区切り成分付き／外来namespaceも対照 | node／atom／区切り成分の所属・順序を別々に保持。normalized listとraw declarationの存在・字面を分離（[PR #2603（main統合済み）](https://github.com/hat47x/tei-platform/pull/2603)のSource Fact修正は実装済み。ただし元のXML byte/atom/segmentの一体Coverageは未検証）。全atom計上から全意味成分保持を推定しない | #950／#1031／private #4 |
| **RV-E1** | 一つの表示線が設計Relation1／実装Refer2候補を縮約。片側だけ変更するUI操作と、一意なSource span/Roleを指定する操作を対照 | generic inspectorからSource/Role/候補0..Nを読めること。表示線だけの曖昧な変更はread-only・navigationへ留め、一意なType/Role/Source/MeaningRevisionのときだけTyped Edit Proposalへ。Proposal→受理→権限→Effectを別々に確認 | #1124／#2497／#2500／#2391 |
| **RV-V1** | 版A→Bのsource-ref変動、移動、同ID異Role、fan-in、view外ZIP member変更 | exact版差と意味差、旧Reviewの履歴と新版Claim適用を区別。旧ViewChangeを新版Actionへ自動rebaseしない | #2505／#2500／#1803 |
| **RV-D1** | Role付きRelation pathを同じFunctionの複数Context occurrenceで使用。ある集合はcomplete、別集合はpartial | Entity Identity／Path occurrence／結果grain／取得scope／選択epochを保持。閲覧・選択・編集Binding／実Effectの資格を別々に示す | #2506／#2507／#2586 |
| **RV-O1** | 同じ親1／子2の合成構成で`FAMILY`を正例に、`REFFER`へTypeだけ変えた場合、子のSortKeyを重複にした場合、同じTypeで対応候補0件／複数件の場合を別対照とする | 独立Expectedが「非対称は一律reject」「FAMILYなら無条件pass」「重複SortKeyでも順序成功」「曖昧なのに先頭候補Accept」のMutantを区別できること。診断0≠比較実施済み、Source形状の許容≠業務列pair受理。Type分岐に関する正負Expectedは旧固定Sourceと自作fixtureから記述し、現行TEI診断を検証基準にしない | #950／#1126／#2514 |
| **RV-O2** | 同一snapshotでKとJのlocal状態が異なるがpublic`ByKind`が集計だけのケース、およびXEAF Referの原属性`missing`／`present-empty`／外来namespaceを正規化配列だけで見たケース | 各Consumerの実入力Projectionを固定し、区別すべきClaimに必要なSource presence／owner scope／対象Factのlocal Resolutionが**その入力に存在するか**を比較。存在しなければ「検証器失敗」以前の入力観測不足として保留。raw保持の修正候補[Draft #2603](https://github.com/hat47x/tei-platform/pull/2603)は最新head試験の資格を別計上し、公開Reportの件数からprivate対象を逆同定しない | #950／#1124／#2514／#2516 |
| **RV-O1** | 同じ親1／子2の合成構成で`FAMILY`を正例に、`REFFER`へTypeだけ変えた場合、子のSortKeyを重複にした場合、同じTypeで対応候補0件／複数件の場合を別対照とする | 独立Expectedが「非対称は一律reject」「FAMILYなら無条件pass」「重複SortKeyでも順序成功」「曖昧なのに先頭候補Accept」のMutantを区別できること。診断0≠比較実施済み、Source形状の許容≠業務列pair受理。Type分岐に関する正負Expectedは旧固定Sourceと自作fixtureから記述し、現行TEI診断をOracleにしない | #950／#1126／#2514 |
| **RV-O2** | 同一snapshotでKとJのlocal状態が異なるがpublic`ByKind`が集計だけのケース、およびXEAF Referの原属性`missing`／`present-empty`／外来namespaceを正規化配列だけで見たケース | 各Consumerの実入力Projectionを固定し、区別すべきClaimに必要なSource presence／owner scope／対象Factのlocal Resolutionが**その入力に存在するか**を比較。存在しなければ「検証器失敗」以前の入力観測不足として保留。raw保持は[PR #2603（main統合済み）](https://github.com/hat47x/tei-platform/pull/2603)のテスト済み限定性にとどめ、公開Reportの件数からprivate対象を逆同定しない | #950／#1124／#2514／#2516 |
| **RV-O3** | 二memberの独立Kで`SortKey=010/020`を固定しXML子出現順だけ反転。別対照で正常・無依存Jを追加。文字列`"2"/"10"`の対照は別fixtureに固定 | 独立Expectedは明示SortKeyで確定するKのordered member/局所readinessのみ不変とする。SourceOrdinal/raw XML/digest、公開総数・Tokenは変化可能と明示。無関係J追加でKの局所結果が変わるMutant、XML順をSortKey順に取り違えるMutantを判別し、旧Source版の判定を新Sourceへ自動再利用しない | #950／#1126／#2514 |
| **RV-D2** | 一つのRelation pathを持つFunctionで表示Field/Graphが同じ。非表示Rule input又はEXISTS条件の値・Path Role・評価Scopeだけ変え、全件と部分Pageを対照 | Display MappingだけでRule/Query/Action Claimを判定しない。Root/Relation Path/Field/Scope/Grain、表示D・選択S・検証V・Action Aの依存由来を区別。部分Pageから不存在/全件SUM/全称Validationを確定しない。非表示入力が変わるとRule結果が変わる独立Expectedを用意し、Read Grantは自動拡大しない | #1126／#2506／#2507／#2586 |
| **RV-D3** | 同一Mapping・同一Result Slice・同一画面で、Provider P1は宣言されたRole/Fieldだけ、P2は追加の秘密Field/未宣言Relationも実読取してから同じ結果を返す（合成） | 同じOutput/Rule結果・Provider応答成功を取得権限へ昇格しない。**Function Contextで明示MappingしていないRelation/Fieldの取得は拒否対象**。必要な補助ReadもFunction側へ明示Mappingし、用途・Scope・現在Grantを検証したもののみ計画へ含める。Declared Read／Actual Provider Read／Materialized Slice／Consumer Disclosedを独立に照合し、現Source診断から実Provider read/OS隔離成功を主張しない | #2506／#2622／#2611／#2516／#2514 |
| **RV-D4** | O1表示amount=600不変でcustomerRef C1→C2、Rule入力A1.limit700→A2.limit500、別対照で有効Hold0件→新規1件、0件complete/partialを変える | Relation pathの元IDではなく再束縛先・実評価Scope・Predicate・Read Cohort・Rule Claimを再観測。既存行IDのない否定依存を保持し、表示の継続とRule/Actionの失効を別判定 | #2582／#2586／#2506／#2514 |
| **RV-D5** | Relation/Rule依存閉包は固定。ただし変更通知WindowにGap/Lag、又はC1→C2→C1のABAを注入する | 実Change Witnessの被覆がない場合、Value再一致や閉じた依存集合だけでcurrent/reuse許可を出さない。現時点のRead再評価と「期間を通じて不変」を別Claimにする | #1125／#2582／#2622／#1708 |
| **RV-P1** | 同じAcceptedモデルからLibrary／SHO document／Manifestを別世代で生成し、(A)file byteは揃うがConsumer不足member、(B)旧file残存、(C)途中manifest切替、(D)正常一式pinを対照 | Generator internally consistent／publish generation atomicity／ExpectedMembers(Consumer)充足／実Reader consumed bytesを別々に照合。意味・版が同じだけで納品済み、読取済み、実行可能を宣言しない | #2482／#2604／#2630／SHO／#1803 |
| **RV-S1** | 公開合成の旧Data Scriptで、同じstatus更新を扱う (a)Pure Derive、(b)全対象Validation、(c)状態Guard、(d)in-Tx consequence、(e)after-commit effect を別の添付/Phase/Triggerで表す。未記述イベントとDynamic Callも対照 | 原文Script/Attachment/Source span/Phaseを保存し、観測できたSource Factと提案されたTyped Ruleを分離。未解釈Callや明示されない契機を「確実に発火」へ昇格させない。純粋派生≠保存・通知、Validation unknown≠false、in-Tx≠after-commit、status値変更≠外部effect Receiptを別Expectedとする。旧原典の同一動作や実発火は実測前NOT OBSERVED | #950／#1129／#988／#1174／#2514 |
| **RV-S2** | 合成OrderのIFで`pending→submitted`、失敗(状態不正/欠損/Rule不満足)を定義。IFからread-only遷移図を作る対照と、別Sourceに同じGuardをコピーした二重Authoring対照 | IF原文を正本とした場合、Diagramは判別できたEdgeだけを出しOpaque branchはpartial表示。独立IF＋Transition表の両方をAuthoritativeにはしない。移植後Function/UI/API/Batchの共有Rule Binding候補と、Owner意味受理・Target実効果を区別。明示TransitionへMode変更するのは別Acceptance | #1129／#1124／#1127／#2506／#2514 |
| **RV-S3** | `tenantId, orderNo`の型付き複合Business Keyで二Tenantに同番号、同Subjectに再申請2 Workflow Instance／同時Active2 Occurrenceを持つ対照。status単一列／複数列／sidecarのBinding候補 | SourceのKey全member・SortKey・型/名前空間と原Workflow参照を保持。Business Subject、Process Instance、State tokenを一つのlegacy PK/string/hashへ畳まない。埋込単一Fieldが複数Instance/Activeを保持できないときはlossy/unsupportedと明示し、Storage選択はOwner/Providerへ。共有Store義務化、型変換でのtenant衝突、実DB CAS/Receipt成功の推定は禁止 | #950／#1126／#1129／#2622／#1174 |
| **RV-M1** | 表示に必要なDと隠れたRule入力Vを明示MappingしたP1と、V又は補助Readを未宣言のままProviderが暗黙取得したP2を対照。物理MappingのField→Columnだけを変える対照も実施 | Provider実Readは明示Function Context Data Mapping/Scope/用途/Grant内のみ許可。単なる同一表示・同一Result/Field IDでは物理ReadやMeaning版一致を主張しない。未宣言Readは拒否対象、必要ならMappingを提案→受理して再評価 | #2506／#2624／#2611／#2622／#2516 |
| **RV-S1** | (A)正式`Load`結果でManaged fileをLoad後に書換：通常Indexもstrict Indexも保持byteから導く。(B)Standalone/部分Resultで未捕捉Pathが存在：通常Indexはfilesystem fallback候補、strictは拒否。(C)imports宣言／別root／manifest変更を個別対照 | Aの一致を通常API一般のSource pinと呼ばず、Bのfallbackとstrict fail-closedを独立期待にする。Cのstrict拒否・manifest非束縛と、原子的Multi-file snapshot・外部Xead ZIP・Target意味・同版公開を同一視しない | #267／#2642／#1017／#1121 |
| **RV-A1** | Derived SelectでP1=preflight Granted／Provider後Revoke、P2=preflight Denied、P3=両地点Granted、P4=途中Revoke→Regrant、P5=未宣言秘密FieldのProvider取得 | mainの二地点fenceの返却拒否と限界を区別。P1/P2でRaw/Derived値を返さず、P4は二地点成功でも期間を通じたGrantの継続を証明しない。P5はFunction Mapping違反として返却結果以外のActual Read/外部送出も追跡（未実証） | #2637／#2506／#2622／#2516／#1174 |
| **RV-P2** | 同じMeaningRevisionでもLoad時TEI SourceとGenerator入力の版が異なる／Manifest一式は正しいが別Consumerが旧Memberを読む／SHOの説明とLibraryの実ロード対象が違う | Source acceptance、Generated、Published、Required/ConsumedMemberを同じ根拠束とみなさず、各実byte・版・Reader時点とOwner承認の適用性を別判定。旧Receipt／成功CIを現在Consumerの受領証明にしない | #2642／#2482／#2604／#2630／SHO／#2516 |
| **RV-P3** | Published Bodyは同じなのにRule/Generator/UnresolvedのEnvelopeが異なる2世代、Manifest Digestだけ新しい版、Missing Metadata、別Consumerへの旧Content混入を対照。対照として同一Envelope・全RequiredMember・実Read一致 | `CheckSet`内部／CLI `-check`／公開Body bytes／Envelope完全再検証／同時公開世代／Reader ConsumedMemberは独立証拠。ManifestとBodyだけでは再構成不能なEnvelopeは「検証済み」と表示しない。正例もOwner納品承認や外部Harness受領を別扱い | #2482／#2604／#2630／#2659／#2662／#2673 |
| **RV-O4** | 同一Operation/Receipt Kと期待値一致の後続Readを用意し、(A)同一Primary位置でCommitを含むRead、(B)Replica lag、(C)別Writerによる同値、(D)Owner対象集合L不足、(E)Consumer非開示を対照 | U→K→V（Commit含有のRead保証）→O→B→Dを別々に判定。Read内容同値・K存在・CLI成功だけでV/B/Dへ昇格しない。実Provider位置/Tx fence/Owner対象Lが欠ける場合unknownを保持し、#1174/#1125既存Ownerへ引渡す | #1125／#1174／#1129／#2671／#2516 |
| **RV-H2** | 実Workの同一生成版ZIP・Probe・Producer・Builder・Candidate・Ledger・receipt | 既存2checkerの実exit／原byteに加え、**一つのSource node→実TargetのJ5参照実解決**を確認。個別未解決は保留し、独立PASSを結合PASSと呼ばない | #1121／#Work／#3／private #4 |

`RV-*` は本節の比較票内の仮ラベルであり、#2505の実装済みRV試験番号／新たな共通Schema・Gate・APIではない。期待は実行済みPASSではなく反証可能な受入候補である。原実体の利用が必要なケースはprivate固定版と使い捨て環境で検査し、原本変更・内容公開を行わない。

### 着手順と成果物の受渡し

1. **S0 P0-a（#950）**：既存Reader・Diagnosticsをexact mainで合成対照し、診断の原因／局所Resolution／全体Status／欠落・明示空の保持を固定する。改善を行う場合はSource-only診断の限定PRとし、Owner Semantic Relationの採用とは別。
2. **S0 P0-b（#Work/#3/#8）**：同じH2生成機会の原byte・正式candidate receipt・正式処遇Ledger・target-ref実解決を、privateの既存ツールと手動J5一行で確かめる。元の別checkpointでの成功を流用しない。
3. **並列P1（#1126/#1031/#2505）**：CrossView候補グラフと版間候補グラフをread-only CLIで比較し、両者の曖昧性・fan-in/out・Coverage母数・開示範囲を別々に表示。#1805はClosedの既存部分取込み責務として参照し、二重実装・再起票をしない。
4. **設計書／Functionの後段（#1803／SHO／#2506/#2507）**：原典宣言・実装候補・受理済み意味を、図＋一覧・説明の根拠版で区別。意味構造（Entity/Relation/Role/Path occurrence）と簡潔なTEI Script/DSLを標準データ構造として保持し、Modeler/Studioの標準UIで可視化する。操作・表示の条件も用途に応じた標準Projectionとして表現し、専用ノード画面や各機能タイプ専用UIを初期要件にしない。SHOの文書正本とTEIの意味正本は独立、MMIのSource Reference再解決・完成版納品の実証は別途必要。構造的に表示可能でも、編集可能／現在資格あり／業務完了とは扱わない。

**CIで判別した追加のSource Fact境界**：2026-11-02の[Draft #2603](https://github.com/hat47x/tei-platform/pull/2603)では、合成`x:ToKeyFields`／`x:WithKeyFields`が旧`attributeValue(localName)`で正規化済みのtyped Refer Factへ取り込まれる反例をGitHub CIで確認した（旧headのRepository HealthはFAIL）。新しい候補パッチはraw Presenceと通常属性の厳密一致を分離し、外来`x:Refer`ノードもtyped Fact対象から外す。**先行したRefer限定head `f68dd486f85e01b9e060535c0443d4792ffa3730`では5ワークフローすべてsuccess**。追加のnamespace・Table scope負例を加えたhead `1ad75599b418520f9a3f3d492c04fe4352494fda` のRepository Healthは **gofmtのみFAIL**、同runのGo vet・全Go test・workspace validateはPASS。gofmt fixture整列を修正した**後続head `7352ee76ad93529c1f02f61ba0b3d707ab7f4a21`はPR #2603として2026-11-03 main統合済み**。当時のCI未確認記述を現在のCI状態へ流用せず、必要なexact-head WorkflowとSource-only Caseを別途照合する。これはsynthetic code及びCI実行版の資格であり、原典意味・実Work H2/J5・Owner Acceptanceを証明しない。raw Source Fact保持が進んでも、CrossViewの業務意味、宣言を利用するFunction Context、外部AIへの開示権限は自動的に成立しない。#950/#2516/#941の既存境界を維持する。
**CIで判別した追加のSource Fact境界（変更履歴）**：2026-11-02の[PR #2603（main統合済み）](https://github.com/hat47x/tei-platform/pull/2603)では、合成`x:ToKeyFields`／`x:WithKeyFields`が旧`attributeValue(localName)`で正規化済みのtyped Refer Factへ取り込まれる反例をGitHub CIで確認した（旧headのRepository HealthはFAIL）。新しい候補パッチはraw Presenceと通常属性の厳密一致を分離し、外来`x:Refer`ノードもtyped Fact対象から外す。**先行したRefer限定head `f68dd486f85e01b9e060535c0443d4792ffa3730`では5ワークフローすべてsuccess**。追加のnamespace・Table scope負例を加えたhead `1ad75599b418520f9a3f3d492c04fe4352494fda` のRepository Healthは **gofmtのみFAIL**、同runのGo vet・全Go test・workspace validateはPASS。gofmt fixture整列を修正した**最終head `7352ee76ad93529c1f02f61ba0b3d707ab7f4a21` は2026-11-02に5 workflow全てsuccess確認済み、その後main統合済み**。これはsynthetic code及びCI実行版の資格であり、原典意味・実Work H2/J5・Owner Acceptanceを証明しない。raw Source Fact保持が進んでも、CrossViewの業務意味、宣言を利用するFunction Context、外部AIへの開示権限は自動的に成立しない。#950/#2516/#941の既存境界を維持する。

**2026-11-03追加Owner確認**：独立KeyのSortKey order判断／`RelationalResolutionPolicyRevision`の継続版、Artifact Public EnvelopeのReader再検証可能な出力条件、実ProviderのK→V→O因果保証、業務完了Bと開示Dの権限境界は、各既存Issue Ownerが個別に決定する。Source診断改善やCLI `-check`成功で一括採用しない。

**Owner未決**：旧FAMILYの各Role・対応FieldとTEI Semantic Relationの個数、条件付きの任意性・cardinality、物理FK/DDL・SQLによる実現、業務Actionの対象・失敗/Commit意味、旧版Reviewの新版適用と公開権利。これらはSource構造診断の改善や合成PoC成功で自動受理しない。**本書初版時にGo実原典試験・既存正式H2 receipt/gate・Work Candidate実体再読・旧GUI/DB・Owner正式受理は実施していない**。
