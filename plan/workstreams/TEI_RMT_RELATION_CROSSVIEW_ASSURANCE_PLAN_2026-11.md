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

### Case別に失敗原因と検証資格を固定する（本表の10統合ケースは未実行）

| ケース | 入力・判別条件 | Headlessで照合する期待／禁止事項 | 担当 |
| --- | --- | --- | --- |
| **RV-F1** | 合成FAMILY：親Key1成分、子PK2成分。親と対応し得る子成分が一意 | 原文Type・Key全member・余剰member・SortKeyを保持。単純な次数差だけでSource形状不正へ昇格しない。対応の意味受理は別 | #950／#1126 |
| **RV-F2** | 同FAMILYで対応候補が0件、または複数件 | 長さ差と対応不能・曖昧を別理由で提示。型・名前・Aliasの一致を無断の確定対応へ用いない | #950／#1126 |
| **RV-R1** | REFFER 1→2、対照として同数2→2／SUBTYPE | Type固有の構造規則を比較。合致する次数から業務Relation・物理FK生成を自動受理しない | #950／#1126 |
| **RV-K1** | FAMILY子側のSortKey空・重複・辞書順と数値順の差 | Relationshipのarity先行short-circuitと独立Key順序原因を分離。診断不在≠順序の実証。既存提案patch採否は担当が判断 | #950／#1126 |
| **RV-N1** | `x:Refer`などの外来namespace要素、unqualified `Refer`上の `x:ToTable`／`x:ToKeyFields` などの属性類似名 | Readerが持つ `Known=false`、正確な属性修飾名、typed Source Fact抽出結果を対照。外来類似名を正規Refer・Table/Key参照へ黙って昇格しない。`attributeValue` の `localName` 利用が残る経路はコード上の未検証リスクとして個別検査。 `x:ToKeyFields` の実CI拒否例は[Draft #2603](https://github.com/hat47x/tei-platform/pull/2603)のSource-only補修範囲 | #950／#1126 |
| **RV-C1** | 設計Relation1宣言／runtime Refer2候補、各々別Role・Alias | Source1／Runtime2／対応候補2／Accepted0（synthetic開始時）を別々に数え、Diagramと辺一覧でも区別。自動分割・合成を行わない | #1031／#1126／#982 |
| **RV-C2** | 複合キー・Referの1 XML属性に複数Field成分。属性なし／明示空／区切り成分付き／外来namespaceも対照 | node／atom／区切り成分の所属・順序を別々に保持。normalized listとraw declarationの存在・字面を分離（[Draft #2603](https://github.com/hat47x/tei-platform/pull/2603)はSource Factの限定修正候補）。全atom計上から全意味成分保持を推定しない | #950／#1031／private #4 |
| **RV-V1** | 版A→Bのsource-ref変動、移動、同ID異Role、fan-in、view外ZIP member変更 | exact版差と意味差、旧Reviewの履歴と新版Claim適用を区別。旧ViewChangeを新版Actionへ自動rebaseしない | #2505／#2500／#1803 |
| **RV-D1** | Role付きRelation pathを同じFunctionの複数Context occurrenceで使用。ある集合はcomplete、別集合はpartial | Entity Identity／Path occurrence／結果grain／取得scope／選択epochを保持。閲覧・選択・編集Binding／実Effectの資格を別々に示す | #2506／#2507／#2586 |
| **RV-H2** | 実Workの同一生成版ZIP・Probe・Producer・Builder・Candidate・Ledger・receipt | 既存2checkerの実exit／原byteに加え、**一つのSource node→実TargetのJ5参照実解決**を確認。個別未解決は保留し、独立PASSを結合PASSと呼ばない | #1121／#Work／#3／private #4 |

`RV-*` は本節の比較票内の仮ラベルであり、#2505の実装済みRV試験番号／新たな共通Schema・Gate・APIではない。期待は実行済みPASSではなく反証可能な受入候補である。原実体の利用が必要なケースはprivate固定版と使い捨て環境で検査し、原本変更・内容公開を行わない。

### 着手順と成果物の受渡し

1. **S0 P0-a（#950）**：既存Reader・Diagnosticsをexact mainで合成対照し、診断の原因／局所Resolution／全体Status／欠落・明示空の保持を固定する。改善を行う場合はSource-only診断の限定PRとし、Owner Semantic Relationの採用とは別。
2. **S0 P0-b（#Work/#3/#8）**：同じH2生成機会の原byte・正式candidate receipt・正式処遇Ledger・target-ref実解決を、privateの既存ツールと手動J5一行で確かめる。元の別checkpointでの成功を流用しない。
3. **並列P1（#1126/#1031/#2505）**：CrossView候補グラフと版間候補グラフをread-only CLIで比較し、両者の曖昧性・fan-in/out・Coverage母数・開示範囲を別々に表示。#1805はClosedの既存部分取込み責務として参照し、二重実装・再起票をしない。
4. **設計書／Functionの後段（#1803／SHO／#2506/#2507）**：原典宣言・実装候補・受理済み意味を、図＋一覧・説明の根拠版で区別。意味構造（Entity/Relation/Role/Path occurrence）と簡潔なTEI Script/DSLを標準データ構造として保持し、Modeler/Studioの標準UIで可視化する。操作・表示の条件も用途に応じた標準Projectionとして表現し、専用ノード画面や各機能タイプ専用UIを初期要件にしない。SHOの文書正本とTEIの意味正本は独立、MMIのSource Reference再解決・完成版納品の実証は別途必要。構造的に表示可能でも、編集可能／現在資格あり／業務完了とは扱わない。

**CIで判別した追加のSource Fact境界**：2026-11-02の[Draft #2603](https://github.com/hat47x/tei-platform/pull/2603)では、合成`x:ToKeyFields`／`x:WithKeyFields`が旧`attributeValue(localName)`で正規化済みのtyped Refer Factへ取り込まれる反例をGitHub CIで確認した（旧headのRepository HealthはFAIL）。新しい候補パッチはraw Presenceと通常属性の厳密一致を分離し、外来`x:Refer`ノードもtyped Fact対象から外す。**当該PRの最新head `f68dd486f85e01b9e060535c0443d4792ffa3730`ではRepository Health、self-build、Wails smoke、PoC、依存監査の5ワークフローが全てsuccessと確認済み**。これはsynthetic code及びCI実行版の資格であり、原典意味・実Work H2/J5・Owner Acceptanceを証明しない。raw Source Fact保持が進んでも、CrossViewの業務意味、宣言を利用するFunction Context、外部AIへの開示権限は自動的に成立しない。#950/#2516/#941の既存境界を維持する。

**Owner未決**：旧FAMILYの各Role・対応FieldとTEI Semantic Relationの個数、条件付きの任意性・cardinality、物理FK/DDL・SQLによる実現、業務Actionの対象・失敗/Commit意味、旧版Reviewの新版適用と公開権利。これらはSource構造診断の改善や合成PoC成功で自動受理しない。**本書初版時にGo実原典試験・既存正式H2 receipt/gate・Work Candidate実体再読・旧GUI/DB・Owner正式受理は実施していない**。
