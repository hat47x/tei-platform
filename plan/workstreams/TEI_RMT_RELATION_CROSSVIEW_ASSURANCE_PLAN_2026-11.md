# 参照モデルRelationship移植：Type固有構造・CrossView・証拠接合の実証計画（2026-11-02）

> **状態：実証計画／Draft #2599候補。** 根拠の版・実行資格を分離する。原典固有情報はprivateに保持（#941）。
>
> **関係する正本**：[Critical Path §32](TEI_PRODUCT_CRITICAL_PATH_2026-10.md)／[横断設計Script-first Draft #2610](https://github.com/hat47x/tei-platform/pull/2610)／[#950 Source Adapter](https://github.com/hat47x/tei-platform/issues/950)／[#1126 ERモデル](https://github.com/hat47x/tei-platform/issues/1126)／[Source改版 Draft #2615](https://github.com/hat47x/tei-platform/pull/2615)。本書の内容はOwner判断・正式Gate・製品資格を追加しない。

本書はCritical Pathの[§32](TEI_PRODUCT_CRITICAL_PATH_2026-10.md)における **S0 Source／包含範囲** と **S1 synthetic read-only** の実証項目を具体化する独立作業計画である。[Draft #2610](https://github.com/hat47x/tei-platform/pull/2610) が同じCritical Path本文の§33/34で横断設計・Script-first Studioの接合を担当しているため、そこでの節番号・編集経路を維持し、本書がRelationship固有の技術詳細を所有する。既存G0〜G4／S0〜S5、#1163の一改訂Evidence、#1174/#1129の実Effect P0、#267のOwner Acceptanceを維持する。実装順は外部要件の机上評価→意味・Plugin→CUI/標準Plugin並行→Headless Driver/Provider→GUIとし、[Draft #2492](https://github.com/hat47x/tei-platform/pull/2492)の提案と整合させる（Draftの採用・統合済みという意味ではない）。

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

#2505版間差分との非重複を明確にする。G-Rの先行実測は[#2505 11/2独立Headless PoC](https://github.com/hat47x/tei-platform/issues/2505#issuecomment-6573011380)：合成Go 1.23.2 standaloneは三版A→B→Cの候補経路まで拡張され、**15 top-level test・race・vetが局所PASS**。対応候補の連鎖は意味上の同一性や旧操作の再利用許可にはならない。版間比較・Claim別適用・生成/公開の後段Consumerへの計画は[既存Draft #2615](https://github.com/hat47x/tei-platform/pull/2615)が担当し、本節は**同版CrossViewとの異なる母集団・接合点**に限定する。**GitHub製品コード・Go 1.26系・正式H2/J5・実原典・Owner受理は未実証**。合成PoCをS0の正式検収器へ代入しない。

### 現行CrossView実装と、設計対実装Relation候補の実証範囲

固定mainの `internal/xeadsource/crossview.go` `BuildCrossViewProjection` は、`InspectWithContract` の後で **`FormatXead` のみを受け付ける**構造になっている。現在のCrossView実装はXEAD Task/Action/Function等の候補を投影するもので、**XEAD RelationshipとXEAF Runtime Referを一つの実行から照合した製品機能の証明ではない**。本書のG-Vは、両形式を別々にSource Factとして保持した後に検査する **同版対応グラフの設計・実証候補**とする。

P1の最初のHeadless確認では、固定した同一配布版に属するXEA[D]/XEAFの**原文SourceRef・構造Key・Field成分とRefer Alias/Role**をread-onlyで観測し、対応候補の理由・不一致・未解決を別々に記録する。設計Relation1件→Runtime Refer複数件の対応は候補辺の集合として保持し、実XEA[D]/XEAFのRelation対応が完全に自動解決できたとの主張を避ける。Raw bytes/probe/receiptの同一版条件とTargetの参照実在性はS0 P0-bの既存H2/J5に委譲する。

`attributeValue(localName)` による他の型付きSource Factへの外来namespace混入も、Refer固有の成功だけで安全とみなせない。[Draft #2603](https://github.com/hat47x/tei-platform/pull/2603)にTable/Key/Relationship/KeyFields全般の **Reader Known + exact attribute** を確認する合成テストを追加した。**当該追加headのCI結果はまだ未確定**。最新headによる同種別・異種別の回帰PASSを確認してから、#950のsource-only品質判断へ送る。

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

[#2514の独立Oracleと観測上の十分性](https://github.com/hat47x/tei-platform/issues/2514#issuecomment-6572161189) を、この移植作業に限定して適用する。診断の有無をExpectedとしてコピーする「実装と同じ実装での自己証明」は避ける。期待値は**合成fixtureの明示宣言、検証すべきClaim、公開旧Modelerの固定ソースにより説明可能な構造条件**から先に手作業で作り、現行`DiagnoseRelationalSourceFacts`の出力をExpectedに転記しない。旧ModelerのType分岐はSource構造の参照であり、業務列pairやOwner meaning AcceptedのOracleではない。

- **判定到達範囲の検査票（テスト用であり新製品enum/APIではない）**：各fixtureについて `input observed / precondition met / compared / rejected / skipped by prior check / no authority to decide` を明示し、該当する入力と理由を別列に書く。既存`RelationalDiagnostic`や`RelationalFactResolution`がこれら全状態を公開表現できるとは主張しない。特に`arity-mismatch`で`continue`したSortKey・型比較、および型情報が空でskipした比較を「一致」と数えない。
- **誤判定を見破れるか**：FAMILY非対称形状を常にrejectする誤実装、FAMILYをTypeだけで常にpassする誤実装、独立Keyの重複SortKeyをRelationship依存時にだけ検知する誤実装、Source属性の`missing`と`present-empty`を混同する誤実装を、それぞれ別の意図的Mutantとして仮定する。正例と負例の両方で区別できるExpectedを要求する。Mutantの実装／実行は**NOT RUN**であり、殺せると確認済みではない。
- **観測上の識別不能対**：同じSourceのK=局所unresolvedとJ=局所resolvedについて、公開`ByKind`件数は共通であり、Consumerの選択対象がKかJかを区別できない。公開集計だけを入力とする「Jの操作可否Oracle」は成立しない。同様に`ToKeyFields`の正規化配列`[]`だけでは、原属性の欠落と明示空を区別できない。元XMLのprivate raw declaration／`Node.Known`・exact属性、当該Factの局所Resolutionを必要範囲で参照し、公開Reportのsource SHAや件数を個別Factの識別子へ昇格させない。
- **範囲限定**：独立Oracle、Source一致、Owner意味確認、公開適格性、実Work H2/J5 Target参照は別の検証。実Go/GUIでのMutant試験や正式Acceptanceを新しく行ったものではない。#2514のOracle/Conformance Profileと#950のSource Fact保持のOwnerを変えず、本書はRelationship固有のfixture入力・Expected・識別不能対だけを管理する。

### 2026-11-02 追補：Metamorphic検証とFunction実消費の境界

[#2514の独立OracleとMutation感度](https://github.com/hat47x/tei-platform/issues/2514#issuecomment-6576646220)を、実行済み製品検収と区別しながらRelationship Source検証へ適用する。また[#2506の表示MappingとRule依存](https://github.com/hat47x/tei-platform/issues/2506#issuecomment-6576904883)、[#2586のQuery→Context→Claim](https://github.com/hat47x/tei-platform/issues/2586#issuecomment-6576783393)で、**同じ表示値でも成立するClaimと実際に消費した入力が異なる**ことが明確になった。本workstreamはそのRelation固有の圧力例だけを所有する。

1. **変換前後に保存すべき主張を限定する**：同じ二Fieldの`SortKey=010/020`を固定してXML上の出現順だけを交換すると、`SourceOrdinal`・source digest・raw locator等は変化し得る一方、現行のSortKey比較で確定する**順序付きField組とKey局所readiness**は、同じ独立Key・同じ検証規則のもとでは不変であるはず。さらに無関係Jの追加で公開`byKind`総数・Source SHA・`diagnosticToken`は変化しても、依存しないKの局所結果は変わらないはず。**何を不変とするかをClaimごとに指定**し、raw差分・公開tokenの不変までは要求しない。SortKeyが`"2"/"10"`なら文字列順と数値順の違いを別の独立Expectedで検証する。
2. **表示範囲とRule/Query依存を区別する**：同じRelation図・同じJの表示Fieldから、Relation pathのRole、非表示WHERE/EXISTS入力、Validation Rule入力、Action対象を復元できるとは限らない。FunctionごとのRoot/Relation Path/Field/Scope/Grainは明示Mappingで定義・追跡し、表示の簡略化から暗黙のRead Grantを作らない。部分取得で`exists` Witnessは確認できても`NOT EXISTS`・全件SUM・全件Validationは成立しない場合がある。入力不足は業務拒否（false）でなく未確定として記録する。
3. **宣言・物理取得・Materialize・開示の分離**：同じFunction要求/Result Slice/表示であっても、Providerが別Role/秘密Fieldを余計に読んだ世界とは**実Read Footprint**が異なる。Source/Design RMTで言えるのは要求候補の由来までであり、実Provider SQL/HTTP/read trace、同期snapshot、現在Principalの権限、Consumerへ実際に送ったbyte/fieldsはそれぞれ#2622/#2611/#2506/#2516/実Host担当が観測する。必要な補助Readもあり得るので、`DisplayFields == PhysicalReadFields`を一律に強制しない。Sourceが適切でも過剰取得/開示が安全と自動確定しない。

本節は**計画上の判別条件**であり、定義・Data Context、Provider、Rule、公開・実Effectの新しいOwner、第二Oracle、Runtime Host、製品enum/APIを作らない。固定Source/Policy/Function Context、Claim、独立Expected、検証到達範囲、実入力を試験票に記録する。Mutant/MetamorphicのTEI製品実行、実Provider/実GUI/Owner意味受理は**NOT RUN**。

### G-V候補0件の証明可能範囲と候補集合の変化（Claim相対／非採用）

[#2514のConsumer観測上の十分性](https://github.com/hat47x/tei-platform/issues/2514#issuecomment-6572161189)と[#2505の版間候補・Claim相対再資格化](https://github.com/hat47x/tei-platform/issues/2505#issuecomment-6572493377)を、**同一版G-V（設計Relationship→実装Refer候補）固有の列挙・選択**に限定して取り込む。[#2516の用途別Consumer確認](https://github.com/hat47x/tei-platform/issues/2516#issuecomment-6579615015)も参照するが、第二のEvidence/Authority体系を作らない。

**否定Claimの範囲を固定する**：設計Relation Rについて「候補辺0件」と「対応する実装Referが存在しない」は同値でない。候補探索の根拠として、(a) exactな設計XEAD・実装XEAFそれぞれのsource version/byte、(b) Reader/profile適合・外来namespaceとunknown/unsupportedの隔離、(c) 対象Table/Key/Role/Alias/属性成分の探索集合と**列挙完了範囲**、(d) 候補生成Rule/Policy revision・省略／打切り／非公開の有無、(e) 当該Consumerが見られる証拠を記録する。**対象集合の完全性が検証できる場合だけ「そのSource版・規則・scopeでは一致候補0件」**と言える。これでも原典に「業務的Relationが存在しない」ことは証明せず、実施した候補照合規則で発見できなかったことの限定的な主張である。未探索・アクセス不可・抽出未対応なら `not observed / incomplete / undisclosed` と説明上区別する（製品state型を新設しない）。private candidate数が秘匿されるConsumerにpublic `0`を返す設計は採らず、開示資格に応じて非表示・判断保留にする。

**Open-world候補増加を再検証の契機にする**：同一Source byteでもRole照合規則・候補列挙範囲の拡大によって、RのRefer候補集合が `{a}`→`{a,b}` に変わり得る。「候補が一意だった」という旧**機械的仮説**を新版の確定対応として引き継がず、Source/Policy/探索範囲を固定した当時の説明と現在の候補集合を並置する。逆に、以前Ownerが明示的に受理した「特定aを対象とする意味」は、新候補bの発見だけで歴史的に取り消さない。**現在の編集・生成・Actionへ受理済み証拠を適用できるか**は、選択aのTarget/Role/MeaningRevision、最新Sourceと現在Authorityの再照合で判断する（[#2500](https://github.com/hat47x/tei-platform/issues/2500)・[#2505](https://github.com/hat47x/tei-platform/issues/2505)へ委譲）。一つの描画線から複数参照を自動編集しないRV-E1を維持する。

これはRMT専用の**合成検証計画**である。現行`internal/xeadsource/crossview.go`の`BuildCrossViewProjection`がXEAD-onlyである事実は変わらず、XEAD↔XEAFのG-V列挙・候補集合増加・Current Owner判定は**製品機能として未実証**。実Work H2/J5、旧GUI、正式Owner受理・公開/配布・外部Harness実消費もNOT RUN。

### 2026-11-02 追加：Relationを利用するClaimの有効期間と、生成成果物の実消費を分離する

**動的依存（#2582/#2586）**：参照Relationを確認しただけでは、Functionがその後も同じRule入力を読んでいるとは限らない。表示を変えない `Order O1.amount=600` のまま `customerRef=C1→C2` が起きると、Ruleが消費する `Account.creditLimit` の対象はA1→A2へ切り替わり得る。これを単なるField値の差分やGraphの表示差として扱わず、**実際に評価したRelation occurrence→Binding→Predicate/Collection→Result Slice→Rule/Operator→Claim**の同一Run証拠を照合する。0件の`NOT EXISTS`は、既存RowIDがないことと依存がないことを混同しない。Predicate/Scope・Provider Read Cohort・列挙完全性・新規加入の変更観測を別に置き、`0件complete`と`0件partial`を別判定する。合成Python 12/12の報告は[#2582](https://github.com/hat47x/tei-platform/issues/2582#issuecomment-6579762087)の局所実績であり、本書やTEIで再実行したものではない。

**依存閉包と変更窓は独立**：採用されたRuleで必要なRelation/Value/Predicate/Role全てを把握していても、前回Readから現時点までのChange Witnessに欠落・Lag・Provider毎の未被覆区間があれば、旧Claimの現在資格は保持できない。[#1125のCWT](https://github.com/hat47x/tei-platform/issues/1125#issuecomment-6579870075)は既存Ownerの証拠であり、RMTのSource Fact・`Source SHA`・`Meaning Revision`や`Materialized Slice`から、実変更通知の完全性を自動推定しない。現行Valueが元へ戻ったABAでも「現在値一致」と「その間不変」は異なる。用途別にread-only表示、Rule Claim、Operation/EffectとSHO説明の失効範囲を区別する。

**生成・納品（#2604/#2630）**：一体モデルをAcceptedとした後にも、生成したLibrary・SHO文書・Manifestが同版で揃うこと、公開が途中で壊れないこと、各Consumerが必要集合を**実際に読んだこと**は別の事実。main統合済み[#2587](https://github.com/hat47x/tei-platform/pull/2587)の`tei artifacts -workspace reference/standard-crud/workspace -application ledger -out <fresh-dir>`は現行CLIの観測入口として利用できるが、そのファイル書き出し経路は個別ファイル→Manifest順のため、原子的な一式公開・Readerへの実同一世代到達まで示すものではない。書式だけの変更、Type/Ruleの意味変更、Producerの変更、公開ファイル欠損、Consumer側の旧世代読取を、用途別の再生成・再評価・再配布で区別する。検証責務は[#2604](https://github.com/hat47x/tei-platform/issues/2604)/[#2630](https://github.com/hat47x/tei-platform/pull/2630)/[#2482](https://github.com/hat47x/tei-platform/issues/2482)へ渡し、本書で第二のPublisher/Oracleを作らない。

この接続は既存Issueの**試験入力と引渡し条件**であり、正式Reactive Engine、共通Changefeed/Host、生成物の新しい権限Schemaを導入しない。TEIの実Query・Provider変更追跡・Artifact Reader・SHOレビュー・Owner受理は**未実行**。

### Case別に失敗原因と検証資格を固定する（本表の25統合ケースは未実行）

| ケース | 入力・判別条件 | Headlessで照合する期待／禁止事項 | 担当 |
| --- | --- | --- | --- |
| **RV-F1** | 合成FAMILY：親Key1成分、子PK2成分。親と対応し得る子成分が一意 | 原文Type・Key全member・余剰member・SortKeyを保持。単純な次数差だけでSource形状不正へ昇格しない。対応の意味受理は別 | #950／#1126 |
| **RV-F2** | 同FAMILYで対応候補が0件、または複数件 | 長さ差と対応不能・曖昧を別理由で提示。型・名前・Aliasの一致を無断の確定対応へ用いない | #950／#1126 |
| **RV-R1** | REFFER 1→2、対照として同数2→2／SUBTYPE | Type固有の構造規則を比較。合致する次数から業務Relation・物理FK生成を自動受理しない | #950／#1126 |
| **RV-K1** | FAMILY子側のSortKey空・重複・辞書順と数値順の差 | Relationshipのarity先行short-circuitと独立Key順序原因を分離。診断不在≠順序の実証。既存提案patch採否は担当が判断 | #950／#1126 |
| **RV-K2** | 同一合成snapshotで独立Kは複合SortKey重複、Jは参照・依存を持たない正常Key。Jの原形閲覧、Jの派生説明、K/J全体loweringを同順に要求 | 独立Key診断候補の適用時はK局所unresolved・J局所resolved・global unresolvedを期待。Jのread-only原形は保持し、派生説明は消費閉包、全体loweringはglobalで保留。公開`byKind`件数からJのIDを逆特定しない | #950／#1126／#1124 |
| **RV-K3** | REFFERの2→2構成で両側SortKey一意、型情報も両側同じ、XML出現順とSortKey順は異なる。対照として片側DataType欠落 | 構造診断0・局所resolvedがあり得ても列pair/Business identityはOwner未受理のまま。欠落型を比較済みとしない。source宣言順・SortKey字面・Validatorが判定した順・意味採用を混同しない | #950／#1126／#267 |
| **RV-K4** | 同一Source byteを旧/候補の判定規則で比較。独立Kのみの0→非0と、既にR1/R2影響がある非0→非0を別fixtureで実施。FAMILY arity分岐も対照 | 診断Boolean/total/byCode/ordinal/token/global/localとpolicy revisionを差分表に固定。影響件数はarity短絡で変わるため「原因1＋順序影響2」を一般化しない。既存Report CLI exit 0をrelease/meaning PASSとみなさず、旧版判定結果の無言再利用を禁止 | #950／#1126／#2514 |
| **RV-N1** | 外来`x:Refer`／`x:Relationship`／`x:TableKey`／`x:Key`／`x:Table`、通常`Refer`上の`x:ToTable`／`x:ToKeyFields`等の属性類似名、外来Tableの子にある通常Referを対照 | Readerの`Known=false`、修飾名、局所Fact、親Table文脈の伝播を独立に調べる。外来宣言・外来親IDを標準Fact/Relationへ昇格させない。既存[#950 RMP-048/RML-023](https://github.com/hat47x/tei-platform/issues/950)が一般契約。**Draft #2603の先行headはRefer限定のCI成功**。追加headではXEA[D]の`Relationship/TableKey`、XEAFの`Key/Refer`と外来Tableの入れ子Scopeを合成テストへ拡張したが、**最新headのCI結果は別途判定**。外来宣言を元のField・Tableの意味受理へ転換しない | #950／#1126 |
| **RV-X1** | 固定同一配布版のXEAD Relationship・XEAF Runtime ReferをSource Factとして独立読取。設計1件→実装候補0／1／複数、同型でもAlias・Role・Field成分が異なる対照 | 現行CrossViewはXEADのみ対応という実装境界を維持。候補グラフG-Vをread-onlyで表示し、原因・多重対応・未解決・元byte/候補Target参照を別出典で照合。次の意味受理・物理FKへ自動昇格しない | #950／#1126／#1031／private #4 |
| **RV-X2** | 同じ設計Relation R。実装Refer候補0件の対照：(A)固定XEAF全対象nodeを列挙・profile適合・Role/Key条件で照合済み、(B)Reader未対応/探索打切り/一部scope非許可で未観測あり、(C)存在を知るが外部Consumerへ候補情報を開示できない | Aのみ「固定版・探索規則・許可scopeにおける一致候補0」を限定主張。Bは未探索を実在しないと扱わず、Cは非開示をpublic `0`へ変換しない。候補数・source node/atom/属性成分・unknown・列挙完了/非公開の各母数を区別し、業務Relationの非存在・意味Acceptedをいずれも推論しない | #950／#1031／#1126／#2514／#2516 |
| **RV-X3** | 同じexact Source byte・設計Relation Rで、探索Rule/対象Role範囲を拡張し候補集合`{a}`→`{a,b}`。旧UIはaを選択済み。別枝ではaに明示Owner判断履歴あり | 旧「候補一意」という導出仮説を新しい列挙規則の現行一意性に昇格させない。aをTargetとしていた歴史的Accepted判断はbの発見だけで自動取消しないが、**現在のProposal/Putback/発行**はpinしたSource・Policy・候補集合・Role/対象・MeaningRevision・現在権限の適用を再照合。見た目の一線/SourceRef一致だけでbを無断採用しない | #1126／#2500／#2505／#2514／#1124 |
| **RV-C1** | 設計Relation1宣言／runtime Refer2候補、各々別Role・Alias | Source1／Runtime2／対応候補2／Accepted0（synthetic開始時）を別々に数え、Diagramと辺一覧でも区別。自動分割・合成を行わない | #1031／#1126／#982 |
| **RV-C2** | 複合キー・Referの1 XML属性に複数Field成分。属性なし／明示空／区切り成分付き／外来namespaceも対照 | node／atom／区切り成分の所属・順序を別々に保持。normalized listとraw declarationの存在・字面を分離（[Draft #2603](https://github.com/hat47x/tei-platform/pull/2603)はSource Factの限定修正候補）。全atom計上から全意味成分保持を推定しない | #950／#1031／private #4 |
| **RV-E1** | 一つの表示線が設計Relation1／実装Refer2候補を縮約。片側だけ変更するUI操作と、一意なSource span/Roleを指定する操作を対照 | generic inspectorからSource/Role/候補0..Nを読めること。表示線だけの曖昧な変更はread-only・navigationへ留め、一意なType/Role/Source/MeaningRevisionのときだけTyped Edit Proposalへ。Proposal→受理→権限→Effectを別々に確認 | #1124／#2497／#2500／#2391 |
| **RV-V1** | 版A→Bのsource-ref変動、移動、同ID異Role、fan-in、view外ZIP member変更 | exact版差と意味差、旧Reviewの履歴と新版Claim適用を区別。旧ViewChangeを新版Actionへ自動rebaseしない | #2505／#2500／#1803 |
| **RV-D1** | Role付きRelation pathを同じFunctionの複数Context occurrenceで使用。ある集合はcomplete、別集合はpartial | Entity Identity／Path occurrence／結果grain／取得scope／選択epochを保持。閲覧・選択・編集Binding／実Effectの資格を別々に示す | #2506／#2507／#2586 |
| **RV-O1** | 同じ親1／子2の合成構成で`FAMILY`を正例に、`REFFER`へTypeだけ変えた場合、子のSortKeyを重複にした場合、同じTypeで対応候補0件／複数件の場合を別対照とする | 独立Expectedが「非対称は一律reject」「FAMILYなら無条件pass」「重複SortKeyでも順序成功」「曖昧なのに先頭候補Accept」のMutantを区別できること。診断0≠比較実施済み、Source形状の許容≠業務列pair受理。Type分岐に関する正負Expectedは旧固定Sourceと自作fixtureから記述し、現行TEI診断をOracleにしない | #950／#1126／#2514 |
| **RV-O2** | 同一snapshotでKとJのlocal状態が異なるがpublic`ByKind`が集計だけのケース、およびXEAF Referの原属性`missing`／`present-empty`／外来namespaceを正規化配列だけで見たケース | 各Consumerの実入力Projectionを固定し、区別すべきClaimに必要なSource presence／owner scope／対象Factのlocal Resolutionが**その入力に存在するか**を比較。存在しなければ「検証器失敗」以前の入力観測不足として保留。raw保持の修正候補[Draft #2603](https://github.com/hat47x/tei-platform/pull/2603)は最新head試験の資格を別計上し、公開Reportの件数からprivate対象を逆同定しない | #950／#1124／#2514／#2516 |
| **RV-O3** | 二memberの独立Kで`SortKey=010/020`を固定しXML子出現順だけ反転。別対照で正常・無依存Jを追加。文字列`"2"/"10"`の対照は別fixtureに固定 | 独立Expectedは明示SortKeyで確定するKのordered member/局所readinessのみ不変とする。SourceOrdinal/raw XML/digest、公開総数・Tokenは変化可能と明示。無関係J追加でKの局所結果が変わるMutant、XML順をSortKey順に取り違えるMutantを判別し、旧Source版の判定を新Sourceへ自動再利用しない | #950／#1126／#2514 |
| **RV-D2** | 一つのRelation pathを持つFunctionで表示Field/Graphが同じ。非表示Rule input又はEXISTS条件の値・Path Role・評価Scopeだけ変え、全件と部分Pageを対照 | Display MappingだけでRule/Query/Action Claimを判定しない。Root/Relation Path/Field/Scope/Grain、表示D・選択S・検証V・Action Aの依存由来を区別。部分Pageから不存在/全件SUM/全称Validationを確定しない。非表示入力が変わるとRule結果が変わる独立Expectedを用意し、Read Grantは自動拡大しない | #1126／#2506／#2507／#2586 |
| **RV-D3** | 同一Mapping・同一Result Slice・同一画面で、Provider P1は宣言されたRole/Fieldだけ、P2は追加の秘密Field/未宣言Relationも実読取してから同じ結果を返す（合成） | 同じOutput/Rule結果・Provider応答成功を物理Readの同一性、取得権限、開示許可に昇格しない。Declared Read／Actual Provider Read／Context Materialized Slice／Consumer Disclosedを別Evidenceで照合し、必要な補助Readは許可・目的条件で個別判断。現Source診断やCrossViewから実Provider read/OS隔離実証を主張しない | #2506／#2622／#2611／#2516／#2514 |
| **RV-D4** | O1表示amount=600不変でcustomerRef C1→C2、Rule入力A1.limit700→A2.limit500、別対照で有効Hold0件→新規1件、0件complete/partialを変える | Relation pathの元IDではなく再束縛先・実評価Scope・Predicate・Read Cohort・Rule Claimを再観測。既存行IDのない否定依存を保持し、表示の継続とRule/Actionの失効を別判定 | #2582／#2586／#2506／#2514 |
| **RV-D5** | Relation/Rule依存閉包は固定。ただし変更通知WindowにGap/Lag、又はC1→C2→C1のABAを注入する | 実Change Witnessの被覆がない場合、Value再一致や閉じた依存集合だけでcurrent/reuse許可を出さない。現時点のRead再評価と「期間を通じて不変」を別Claimにする | #1125／#2582／#2622／#1708 |
| **RV-P1** | 同じAcceptedモデルからLibrary／SHO document／Manifestを別世代で生成し、(A)file byteは揃うがConsumer不足member、(B)旧file残存、(C)途中manifest切替、(D)正常一式pinを対照 | Generator internally consistent／publish generation atomicity／ExpectedMembers(Consumer)充足／実Reader consumed bytesを別々に照合。意味・版が同じだけで納品済み、読取済み、実行可能を宣言しない | #2482／#2604／#2630／SHO／#1803 |
| **RV-H2** | 実Workの同一生成版ZIP・Probe・Producer・Builder・Candidate・Ledger・receipt | 既存2checkerの実exit／原byteに加え、**一つのSource node→実TargetのJ5参照実解決**を確認。個別未解決は保留し、独立PASSを結合PASSと呼ばない | #1121／#Work／#3／private #4 |

`RV-*` は本節の比較票内の仮ラベルであり、#2505の実装済みRV試験番号／新たな共通Schema・Gate・APIではない。期待は実行済みPASSではなく反証可能な受入候補である。原実体の利用が必要なケースはprivate固定版と使い捨て環境で検査し、原本変更・内容公開を行わない。

### 着手順と成果物の受渡し

1. **S0 P0-a（#950）**：既存Reader・Diagnosticsをexact mainで合成対照し、診断の原因／局所Resolution／全体Status／欠落・明示空の保持を固定する。改善を行う場合はSource-only診断の限定PRとし、Owner Semantic Relationの採用とは別。
2. **S0 P0-b（#Work/#3/#8）**：同じH2生成機会の原byte・正式candidate receipt・正式処遇Ledger・target-ref実解決を、privateの既存ツールと手動J5一行で確かめる。元の別checkpointでの成功を流用しない。
3. **並列P1（#1126/#1031/#2505）**：CrossView候補グラフと版間候補グラフをread-only CLIで比較し、両者の曖昧性・fan-in/out・Coverage母数・開示範囲を別々に表示。#1805はClosedの既存部分取込み責務として参照し、二重実装・再起票をしない。
4. **設計書／Functionの後段（#1803／SHO／#2506/#2507）**：原典宣言・実装候補・受理済み意味を、図＋一覧・説明の根拠版で区別。意味構造（Entity/Relation/Role/Path occurrence）と簡潔なTEI Script/DSLを標準データ構造として保持し、Modeler/Studioの標準UIで可視化する。操作・表示の条件も用途に応じた標準Projectionとして表現し、専用ノード画面や各機能タイプ専用UIを初期要件にしない。SHOの文書正本とTEIの意味正本は独立、MMIのSource Reference再解決・完成版納品の実証は別途必要。構造的に表示可能でも、編集可能／現在資格あり／業務完了とは扱わない。

**CIで判別した追加のSource Fact境界**：2026-11-02の[Draft #2603](https://github.com/hat47x/tei-platform/pull/2603)では、合成`x:ToKeyFields`／`x:WithKeyFields`が旧`attributeValue(localName)`で正規化済みのtyped Refer Factへ取り込まれる反例をGitHub CIで確認した（旧headのRepository HealthはFAIL）。新しい候補パッチはraw Presenceと通常属性の厳密一致を分離し、外来`x:Refer`ノードもtyped Fact対象から外す。**先行したRefer限定head `f68dd486f85e01b9e060535c0443d4792ffa3730`では5ワークフローすべてsuccess**。追加のnamespace・Table scope負例を加えたhead `1ad75599b418520f9a3f3d492c04fe4352494fda` のRepository Healthは **gofmtのみFAIL**、同runのGo vet・全Go test・workspace validateはPASS。gofmt fixture整列を修正した**現行候補head `7352ee76ad93529c1f02f61ba0b3d707ab7f4a21`は改めてCI結果を確認する**（確認時点で未完了）。これはsynthetic code及びCI実行版の資格であり、原典意味・実Work H2/J5・Owner Acceptanceを証明しない。raw Source Fact保持が進んでも、CrossViewの業務意味、宣言を利用するFunction Context、外部AIへの開示権限は自動的に成立しない。#950/#2516/#941の既存境界を維持する。

**Owner未決**：旧FAMILYの各Role・対応FieldとTEI Semantic Relationの個数、条件付きの任意性・cardinality、物理FK/DDL・SQLによる実現、業務Actionの対象・失敗/Commit意味、旧版Reviewの新版適用と公開権利。これらはSource構造診断の改善や合成PoC成功で自動受理しない。**本書初版時にGo実原典試験・既存正式H2 receipt/gate・Work Candidate実体再読・旧GUI/DB・Owner正式受理は実施していない**。
