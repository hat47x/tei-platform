# 三視点Diagram／Cross-referenceの製品受入計画（2026-11-07／設計提案）

> 状態: Proposed / Owner未受理 / 新規Go・Web・Desktop・SHO E2E NOT RUN。正式Canonical/API/Renderer採用やG0〜G4資格を宣言しない。
> 所有: TEI #1124（統合モデル編集）, #1126（Data）, #1125（Business）, #1031（Legacy cross-view）, #2497（Projection/逆写像）。製品横断の順序は #1119。SHO側は sho-composer #5/#6/#19。

## 1. 狙いと採否判断

X-TEA Modelerが担う Data / Function / Business の各図面表示・編集と、CRUDなどのクロスリファレンスはTEI Modelerの根幹機能であり、Sourceをinspectできるだけで代替しない。SHOでは、通常Markdownから記述量の少ない図解を作成し、またTEIの同一図面を文書に参照・注釈できることを目指す。

Script/Markdown/表（移行対象の *.tei.md は #2621 のParser資格に依存）が意味Authoringの正本。Diagram/Matrix/業務フロー/機能階層は同一Source/Accepted Meaningの派生Projection。図面を第二Canonical、独立Authority、専用Runtimeへ昇格させない。一方で、可逆な種類について図上編集の成功を製品受入から外さない。

## 2. 標準構造、Diagram Profile、図面インスタンス

| 層 | 担当する情報 | 不変条件 |
| --- | --- | --- |
| Standard Data Structure | Record/Relation/Graph/Tree/Sequence、Stable Subject、辺Identity/方向/多重度、Grain、Unit/Time、Presence/Closure、Source/Revision | 表示によって失われた関係を「不存在」にしない |
| Semantic Role/Meaning | Entity、Key/FD、親子/参照、Function/Declared CRUD、Business Procedure/Actorなど | Declared、Derived、Accepted、Observedを混同しない |
| Diagram/Matrix Projection | view kind、対象Scope、対象ノード/エッジ、根拠となるpath、coverage、rule/profile version | 同一ViewからSourceと構成Factへ遡れる |
| Presentation Profile | 水平属性配置、実例行、N側インデント、読み順、edge routing、凡例、折畳み、表示名、Density | layout/説明変更はMeaningを変更しない |
| Presentation Instance | Function内の対象/目的に束縛した表示、選択・ハイライト・viewport/文書Occurrence | 表現型の定義と、配置済みInstanceを混同しない |
| Renderer/Target | Mermaid、D2、Graphviz、ELK/SVG/Canvas等、HTML/PDF/DOCX向け生成 | 結果の見た目が同じでも編集性/構造/Accessibilityの同値を推定しない |

最小Projection契約は新Core schema採用を先取せず、既存#2497等で表現可能な字段の合成として評価する: model source identity/revision、view kind、viewport scope、projection rule+version、ordered nodes/typed edges、edge provenance、closure/unknown、loss、presentation profile version、source navigation、edit eligibilityと拒否理由。

## 3. データ図の表現力を汎用Graphへ落とし込む

単純な1:Nはトポロジーの分類ではない。少なくとも、(A)通常参照、(B)親の中で反復する子/複合業務キー、(C)独立した二つの定義域の組合せに制約をかける関係を、それぞれ独立したKey/FD/Relation identityで扱う。surrogate PK/DB FKから業務Keyや帰属を推定しない。

X-TEA式の読みやすい表示Profileを、属性を横に並べて具体値を下に置く／多重度が大きい対象を右方向へインデントする／日本語の業務名称と短い識別子を切り替える、という具体的なPressureで試す。具体値は #1131 の型付きDataset/Snapshotであり、DDL・業務Meaning・実Provider Dataの存在証明ではない。

Organizationの兼務・異動履歴では、Graphを一つのTreeへ投影すると複数親・有効期間・時点が隠れ得る。表示されない第二の所属や有効期間を暗黙に消去する編集は拒否し、Lossの説明を残す。

参考一次情報: https://dbc.in.coocan.jp/ ; https://dbconcept.hatenablog.com/entry/2025/09/14/074541 ; https://dbconcept.hatenablog.com/entry/2026/01/21/102610

## 4. クロスリファレンスは根拠付きRelation集計

- Function × Data: Function-level declared C/R/U/Dと使用対象のKey/Scopeをセルの一次表示候補にする。
- Business × Function: TaskActionによるFunction参照・manual/external action・順序やRoleを別関係として保持する。
- Business × Data: TaskAction→Function→IOTable→Tableはderived potential path。TaskAction固有EffectやHost許可を表すセルに自動昇格させない。
- Runtime Effect: Command disposition、Provider effect、Receipt、独立したTarget再Readは観測範囲ごとに分け、設計時CRUDとは別列/overlayで扱う。
- Matrixの行列/セルにはSource scope、母集団の完全性、未解決、非開示、対象時点/Revision、rule versionを保持し、空セルを「絶対に関係なし」と読み替えない。
- Cell→constituting typed edge→Source location / unresolved reason を遡り、関係の追加・削除・Scope変更時には影響範囲を再評価する。
- Subsystem境界の越境CUD診断は#1132、分割提案/比較は#3250のOwnerを再利用し、Matrix Rendererへ新しい業務Authorityを入れない。

## 5. 編集を四分類する

| 図上操作 | 認める効果 |
| --- | --- |
| ノード位置・幅・折畳み・ハイライト・表示名の選択 | Presentation metadataだけを変更する。Source意味を変えない |
| 同一Semantic対象の一意・可逆なRelation編集 | Stable Subject+Source span+Revisionへのtyped Proposal→Preview/Impact→既存Acceptance→局所Patch→reparse/reproject |
| Derived pathや集計Matrixのセルを直接変更 | 更新先・根拠が一意でなければread-only。元のFunction/Relation定義へNavigate |
| 部分取得、権限失効、非単射、旧Rev/古いUI selection、unknown Role | 無断書戻し拒否。現在の対象/Reason/Sourceへ戻す |

一般のGraph操作すべてに往復Lens法則を強制しない。安全性は一意な対象・保持した情報・現在Revision/Authority・実適用経路の条件で評価する。Graphの編集で外部Provider effectは発生しない。

## 6. SHOテキスト図とTEI参照図

- Authored Diagram: 著者Markdownのfenced textが正本。Mermaid相当の短い記法から、標準Graph/RelationのPresentation Profileを利用する構成を比較。SHO専用Semantic Parserの増殖は避ける。
- TEI-linked Diagram: 文書Occurrenceはsubject/view/profile/意図したrevisionと引用条件を持ち、実取得Projectionはread-only。SHO側レイアウト・注釈をTEI意味Sourceへ自動Patchしない。
- Static Figure: SVG/PNG等は実asset byteとcaption/alt/anchor/reviewを固定し、画像だけから元Graph意味や編集性を推測しない。
- Renderer engine/layout/font/theme/viewport、raw source/revision、入力Data/Rule、実生成asset bytes、Target別lossは文書Preview/納品資格の依存。Previewと固定納品の必要member実Readは別。
- Mermaid/D2等は候補Adapter。実版/対応図種/結線・表組み・可視密度で比較する。MermaidのELK同梱条件は採用したversion/buildごとに検証する。

参考: https://mermaid.ai/open-source/config/layouts.html ; https://www.d2lang.com/examples/overview/

## 7. 実証用Reference（元モデルとは独立したExpected）

一組の合成世界: 受注見出し、受注明細（受注番号+行番号）、顧客、品目、倉庫別品目残高（品目+倉庫）、受注登録/変更/出荷確定Function、手作業確認・差戻しを含むBusiness Procedure、在庫は別Subsystem owner。Datasetには1:N、同ラベル異ID、多重有向辺、可変参照、同時点兼務、部分取得をそれぞれ独立に含める。

| 試験 | 独立Expected（現時点はすべてNOT RUN） |
| --- | --- |
| M-01 | CLIが同じSource版からData、Function、Business図、Function×Data CRUDを生成し同一Subjectへnavigate |
| M-02 | 1:Nである三種のRelationを同一視せず、実例行/業務Key/FDを保持 |
| M-03 | Function宣言CRUDとBusiness potential path、実Effect、手作業を異なる根拠資格で表示 |
| M-04 | 部分取得・非開示・未解決セルを「空」とせず、coverageが変われば再判定 |
| M-05 | 1件の可逆な図上Relation編集を既存Source→Preview/Acceptance→reprojectへ往復。非可逆変更は拒否 |
| M-06 | Graph→Treeによる隠れた第二親/方向/多重辺、旧Revision、曖昧名で誤Patchしない |
| M-07 | 日本語長名・具体値・大規模図で交差/欠落/文字重なり、関係追跡、印刷/縮小を実測。見た目の描画成功のみで読解合格としない |
| M-08 | SHO単独Markdownの図を保存再Read。TEI参照図にSource版・出典・Occurrenceを束縛し、文書注釈変更がTEI Meaningを変更しない |
| M-09 | HTML/SVG、PDF、DOCXで図のalt/参照/編集性/asset版とlossをTarget別に検査 |
| M-10 | 原Source不変・renderer/profile版変更またはTEI参照版変更で、必要な図・説明/納品だけ再確認。旧Render後着を現在資格としない |

## 8. 開発順・責務・停止線

1. P0: Headless CLI/MCPに既存inspect系の読み取りProjection/Matrix/coverage/source navigationを合成。新規公開コマンド名の採用は保留。
2. P0: 最小Data図＋Function図＋Business図＋CRUDを同一Fixture/同一Revisionで照合。#1125/#1126/#1031既存の意味Readerを再利用。
3. P0: 1件の可逆な図上編集を既存Proposal/Shared Acceptanceまで通し、部分取得・異名同ID・旧Revの負例を拒否。
4. P0→P1: Web/Studio consumerの選択・表示・編集資格を同じ独立Expectedで検証。Wails依存を初回前提にしない。
5. P1: SHOのReader/Writer縦断が成立した後、独立テキスト図→TEI参照図→Target別出力/固定納品を段階実証。SHO S2完成をTEIのG1/G2実Provider Gateの前提にしない。

今回変更はdocs-only/設計計画のみ。Source/Diagram/SHOの実製品E2E、Renderer採用、独立Read、Owner Meaning Acceptance、Host権限、Public model転載資格は未確認。元の参照モデルや資料図面の配布可否は#941等の既存Publication Boundaryに従う。

### 関連Issue（担当の二重化を避ける）

TEI: #1119 #1124 #1125 #1126 #1031 #1131 #1132 #1803 #2497 #997 #2621 #2506 #2507 #3250 / SHO: hat47x/sho-composer#5 #6 #19。

## 9. 2026-11-07 横断成熟化：三つのトポロジーを跨ぐ選択・編集資格

他レーンの[標準表現型トポロジー Draft #3192](https://github.com/hat47x/tei-platform/pull/3192)、[表現間Coordination Draft #3222](https://github.com/hat47x/tei-platform/pull/3222)、[Data/Function/Businessの引継ぎ Draft #3225](https://github.com/hat47x/tei-platform/pull/3225)と本書の**担当範囲を重複させない**。これらのDraftはmain統合・正式契約採用を前提としない。

- **Data topology**: Subject/Relationの意味、Edge ID、Multiplicity、Path occurrence、Business Key/FD、単位・時点・Grain。DB JOIN resultやUI rowをDomain Entityと同定しない。
- **Presentation topology**: Graph/Tree/Matrix/Flow/Pageの表示配置、group/fold/viewport/legend、表示候補の適合条件。図形座標やページ番号をRelation Identityにしない。
- **Binding topology**: Function（設計Inspect等のread-only機能も含む）のContext/目的/選択集合とRepresentation Instance/Slotを接続。型としてのGraphは独立に定義できるが、利用中の図面Instanceの意味・操作資格は明示された用途文脈なしに成立したと主張しない。SHOの図面Occurrenceは文書機能に属し、TEI業務Functionと同一の正本にはならない。

**選択同期は更新ではない**。Matrixセル→Function宣言／Typed Edge集合→Data図の辺選択→Source locator、という同期をread-onlyで先に成立させる。次を常に分ける。

| ID/状態 | 例 | 禁止する暗黙推定 |
| --- | --- | --- |
| Semantic Subject / Relation / Path occurrence | `data.line`、`rel.order.lines`、明細出現L1 | 同じ見た目・同じLabelなら同一Subject |
| Projection occurrence | 複数図に現れる1 Subject、同一セルに重なる複数Edge | 1表示行＝1業務対象、セル1つ＝元定義1つ |
| Function Context / Result Slice | 対象選択母集団、取得Filter、Scope、Closure、Version | 可視2行＝全件2件 |
| Interaction Selection / Epoch | 現図の選択、古いAsync応答、Zoom/fold | 選択済みなら現在の操作Grantを取得済み |
| Change Proposal / Action Eligibility | 一意なRelationのoptional変更、明示的なDiff | 図のドラッグ・選択・直接リンク操作が業務Update |
| Artifact / Effect / Delivery | 図面生成、SHO納品、実Provider receipt | SVG成功＝DB永続化、文書出力成功＝受信者実Read |

このマッピングを#1124のCUI/Studio Consumerで再利用し、選択同期Engine、第二Source Index、独立Action Resolverを新設しない。CUIで提案するJSON shapeは正式ABIではなく既存`model inspect/trace/diff`、Source Locator、#2506 Context、#2497 loss/逆写像へ適合する最小フィールドから導出する。

## 10. Pluginとしての図解・CRUD基盤と互換性BOM

「Core以外はPlugin」の方針を満たし、Studio専用Diagram EngineやSHO専用の第二の意味解釈器を作らない。同じ読み取りProjectionをCLI/Studio/SHOに使わせること自体をdogfoodingとする。

| 役割（候補、正式Plugin名ではない） | 担当 | 境界 |
| --- | --- | --- |
| Cross-reference projection | Function/Data/Businessの型付きEdge→CRUD/対応表 | 業務Operation宣言と実Effectを区別。表計算ファイルそのものを正本にしない |
| Diagram projection | Data/Business/FunctionのSource/Meaning→loss-aware Graph | 構造保存と逆写像は#2497、意味Roleは#2620 |
| Presentation profile | X-TEA式属性・具体例・N側配置、用途・対象読者別Density | 意味Graph・Accepted Sourceは書換えない |
| Layout/Renderer adapter | 実際のGraph→SVG/Canvas/PDF向け図 | Target/renderer/asset/font versionを固定し、適合を検査 |
| Interaction adapter | 選択同期、Source navigation、型付き編集案のPreview | 現在のAuthority/Grantを付与しない。GUI固有意味を増やさない |
| Document consumer | SHOのMarkdown fenced block/TEI引用/Occurrence/注釈 | 原文編集、固定納品、開示はSHO #4/#5/#6/#19のOwner |

**Verified BOM**は、Plugin Contract/Provider/Renderer/Profile/CLI/Studio/SHO consumer、実行Target、font/locale、Schema・Operator/Generatorの実消費版、およびその組合せで実際に実行されたCase集合とReceiptを区別して表す。単に解決できる依存グラフを「検証済み」としない。複数バージョンの試験では、全組合せ・pairwise・境界組合せ・変更影響に基づく再試験を別にし、未実行組合せを合格扱いしない。Plugin入替時は**意味Projection digest / 表示geometry・asset digest / document review**の失効を独立判断する。Ownerは既存Plugin/Composition/BOM/Scenario側に維持し、本書で第二BOM Resolverや専用CIを設けない。

## 11. 再生成・鮮度・描画規模・読みやすさ

世界の意味が同じでも表示資産が異なる場合と、見た目が同じでもMeaning/Authorityが失効した場合を分離する。

| 変更 | 再確認対象の代表例 | 再受理しないもの |
| --- | --- | --- |
| Relation/FD/Function/CRUDのSourceまたはScope変更 | 構造Projection、CRUD、影響解析、図面/SHO引用Review | Layoutの座標だけを業務意味へ昇格しない |
| 前回は存在しなかったEdge/Functionが追加 | 「該当なし」Matrixセル、閉包/欠落依存、related graph | 空結果を「影響なし」と再利用しない |
| source A→B→A、Owner受理撤回、Access/Grant変更 | 最新受理・Disclosure/編集資格、旧Selection/Document引用 | byteやdigest復帰で過去Grantを復活させない |
| Dataset/Snapshot/例値変更 | 例値の表示、図解説明、文書Review | Schema/Key/FDを例値から更新しない |
| Renderer/layout/font/theme/viewport/locale変更 | geometry、読みやすさ、SVG/PDF/DOCX/alt、出力Review | Meaning revisionへの不必要な受理要求 |
| Diagram Profile/Projection Rule/Plugin版変更 | 定義済みLoss/Orientation/Cardinality、構造DigestとBOM適合 | Source同一で旧Ruleの資格を引き継がない |

大規模モデルではSubsystem/Role/関心領域での**根拠付き折畳みと段階的展開**、ノード・Edge数/文字数/線交差数/操作応答時間/増分再計算の測定を優先する。Graphの全件読込みや全体再描画を常に要件とせず、取込の部分性と今見えている図の範囲を明示する。増分導出・Cache・MaterializationはDerived Data/世界・モデル・Artifact Revisionの既存資格に沿わせ、表示ArtifactのCache hitから現在の開示・編集許可を推定しない。

人間向けの可読性目標は、読み順・日本語長名・1画面あたりの構造密度・Relationを辿れること・凡例・代替表・印刷を**別の観測項目**として扱う。自動判定（文字の重なり/切れ・関係線の一致）とAIによる説明可能性評価、Ownerの最終意味判断を独立させる。現在実際の読者試験は未実施であり、「高い視認性を実証済み」とは記載しない。

## 12. 新しい実行可能Referenceへの橋渡し（保存済みだが実行なし）

本PRに以下を付ける。入力はTEI正式DSLではなく合成Pressure data、期待は**実装出力から生成していない別JSON**として明示した。両方とも今回AIで草案作成したため、独立Expectedとしての品質検査・Owner受理は**未取得**。文書からモデル解釈や合格を自動認定してはいけない。

- [synthetic pressure input](../../reference/diagram-crossref/pressure-input-v0.json): Entity 9、型付きRelation 7、Function 4、Business Action 3、二つの1:N子集合の2×3 JOIN、同ラベル別ID、同端点の多重有向辺、手作業、外部Functionの宣言不完全、partial／Actual未観測のPressure。
- [separately specified expected](../../reference/diagram-crossref/expected-v0.json): 宣言CRUD Matrix、Business Action経由のpotential path、manual/跨Subsystem、非単射の書戻し、SHO投影・選択同期・Renderer変更と12個の既知Mutantに期待するFAILを分離。

**受入の段階**:

1. **P-1（即着手）**: CLI/Plugin/Studio現行コードが合成Inputをどの型付きReaderまで読めるか静的Fit/Gapを報告する。JSONの存在はLoader実装・Go Test PASSではない。既存CLIの実出力と必要最小限の差分を固定。
2. **P0-H**: 当該Inputを既存TEI Source/Accepted Worldで表現できる最小部分に変換し、World/Scope/Source locatorを保持したread-only GraphとMatrixを*実際のCLI*から取得。Oracleは別担当の検証/既知Mutant棄却後に使用。
3. **P0-E**: 1件の可逆なRelation `rel.line.sku.alt.optional=true→false`（説明用Synthetic）とnon-injective Graph→Tree、曖昧選択、古いRevision、部分取得の拒否を既存Edit/Acceptance経路で実行。必要なOwner判断なしに業務上の変更を採用しない。
4. **P0-C**: CLI/MCPとStudioを同一Source/Graph/Matrixの異なるConsumerとして、Semantic IDs・coverage・失敗分類・Selection Epochを比較。Provider実Effectが要らないread-only設計検証を先行させ、G1/G2実DB Read/Commit資格と合算しない。
5. **P1-SHO**: SHOのRaw Markdown/Reader/Writerが成立した後、短い独立テキスト図とTEI参照図を実Renderer/Target別に検証。図面出力のみで文書固定納品の一式実Readを主張しない。

**異質なDogfooding**: 同じGraph/Node/Edge/表の汎用契約を、[HTTP Response Decision Graph #3274](https://github.com/hat47x/tei-platform/issues/3274)のServer/Client decision graphとPlugin依存BOMに適用する。HTTP側のHDG-08・10等が要求する並列辺/Guard/条件/未取得/部分/実Trace区別を本ProjectionへPressureとして戻す。ただしHTTP業務意味/Client Reaction/実Guard合否は#3274とその既存Owner、Plugin BOM資格はComposition/Test Ownerとし、図面PoCから実HTTPを受理しない。

### Referenceの判定票（本PR時点）

| 項目 | 状況 | 留意点 |
| --- | --- | --- |
| GitHubへの合成Input・期待JSONの保存 | 保存済み（GitHub内容再読取・JSON構文検査は別） | 新TEI実行機能ではない |
| 既知Mutantの設計 | 12種類を期待結果として記載 | Mutation runner・独立oracle reviewer未実施 |
| Headless Graph/CRUD/CUI | NOT RUN | 開発する場合は#1124/CLI既存入口を拡張 |
| Source Patch/Studio/SHO実Consumer | NOT RUN | Writer/Renderer実装とArtifact資格は別 |
| Actual Provider/Owner受理/Production | NOT RUN / NOT ACCEPTED | 既存Gate・Authorityを迂回しない |

## 13. Source AdmissionとCLI実装Fit/Gap（11/07実Sourceに基づくP-1停止線）

別レーンの[#3262](https://github.com/hat47x/tei-platform/issues/3262)が「用途・Subjectに相対的なRequired closureを満たすDesign Context Query」を所有している。ここでは**Diagram/Matrixに対する具体的なConsumer requirements**だけを返し、第二Design Context QueryやGraph-only Registryは導入しない。

今回の[現行 `cmd/tei/cli_contract.go`](../../cmd/tei/cli_contract.go)の静的照合では、`tei.cli.discovery/v0` は `selected-read-only-commands` を範囲とし、`model catalog`、`workspace-files`、`inspect`、`trace`、`source-index`、`knowledge-context`、`validate` などを列挙する。一方、**独立したDiagram/CRUD投影を返す公開CLIの契約は、このcatalog内では確認できない**。別の内部Readerや古いLegacy Reference Toolまで存在しないと断定するものではない。`cmd/tei/main.go`の未知コマンド拒否と、旧`tools/reference-xead-cross-view/main.go`のlocal/private Projectionを参照し、公開CLIとLegacy専用観測・製品Canonical意味を混同しない。実CLI起動、`*.tei.md`のParser/admission、Diagram consumerは今回NOT RUN。

**最小Request（概念比較票、採用済みABIではない）**: `workspace/scope + subject IDs + intended view(Data/Function/Business/CRUD) + source revision + required relations/operation granularity + admissible disclosure scope + expected closure + page budget`。

**最小Response（既存Readerから本当に観測できるものだけ）**:

| 列 | 役割 | 完全性・安全性 |
| --- | --- | --- |
| `requested/declared/discovered/loaded/parsed/bound/returned` | 対象集合の各段階の観測 | Source未読込と業務上の対象ゼロを混同しない |
| Subject/Edge ID、Relation/Path occurrence、source locator | Source→三視点→Matrixセルの根拠 | 同じ表示ラベルと同じ業務対象を同一視しない |
| `view scope/revision/rule` と `coverage/unknown/unresolved` | ユーザが現在理解できる範囲 | 未取得を「存在しない」へ、partialを全件へ昇格しない |
| `selected cell constituting edges` | MatrixがどのFunction宣言、Business path、Provider Observationに由来するか | scalarな`C`/空文字では根拠が失われる |
| `page index/continuation/end marker` | 大規模図・Harness向けの完全受領 | exit0でも途中切断・後着・混合Revならcomplete不可 |
| `loss + allowed operations + reason` | 逆編集可能か、read-onlyか、別Action要か | UI資格をHost Grant/Meaning Acceptanceへ昇格させない |

**戻り値の重要な意味論**: 行列の空セルは、declared operationの列挙閉包が当該Function/Data/時点/Scopeについて**確認できたときだけ** `none` とする。それ以外は `unknown-not-enumerated` など明示の未確定値を使う。現在の例では `fn.external.dispatch` の全9セルをunknownにし、独立Oracle側も同じ区別を求める。Businessの`TaskAction→Function→Table`はFunction-level potential pathで、Business-specific actual CUDではない。

**P-1入口を満たすために取得する実証**: 一つの合成Subjectで `workspace-files`とLoader、`model catalog`、`inspect`、`trace`、`source-index`、`validate`の実結果を同じcheckout/Scopeへ帰属させ、source byte とLoader parsed/read、Candidate Meaning、Acceptedの区別を一覧化する。Graph/Matrixを要求したのに公式Readerが未提供なら`unsupported`とし、Fixture JSONを`tee`しただけの自作描画を製品Projection合格としない。参照Xead XMLからの観測結果を新Canonicalとして扱わない。

Sourceを`*.tei.md`へ統一する方針は#2621に従い、現状のLoader/Parser Gapが残る場合は実行資格を未実証にする。CLI/MCP/Skillで同一用途の応答を比較する際には、対象閉包・出力の実byte数・Tool Reply件数・end marker・Disclosureを記録する。#3262の既存DCQ-01〜10をこの製品Diagram/Matrix Consumerへ適用し、新番号の同等試験は重複起票しない。

## 14. 最小のFixture監査を実Go Testへ追加

[合成Fixture](../../reference/diagram-crossref/pressure-input-v0.json)と[AI草案Expected](../../reference/diagram-crossref/expected-v0.json)について、[fixture契約Test](../../reference/diagram-crossref/fixture_contract_test.go)を追加。**これはモデルParser/Projection/CLIのテストではなく、入出力のテスト前段で必要なFixture自身の整合検査**である。

チェック内容: Entity/Edge/Function/Business Actionの重複ID、Relation端点、同端点の多重辺、同ラベル別ID、Function-level declared CRUD表の全列、宣言不完全時のunknown、既知Mutation ID対応。さらに、edge欠落、identityの衝突、宣言不完全の誤完全扱い、参照切れという4種類の改変がFixture監査に検出されることを確認する。今回の12種の`knownMutantRejections`は**12種類の失敗を実ProductのProjectionで検出した結果ではない**。実際のMutant Scoreは検証対象の実現と各CaseのRun Receiptに基づき#2514/#2811で集計する。

初回ローカルのGo試験はGo 1.23.2/隔離された簡略化Stub入力で2つのTest FunctionがPASS（4 subtestを含む）。これはGitHubに保存した**実JSONのGoテスト実行と同一ではない**。リポジトリ内での実Fixture TestとRepository Healthは、新しいcommit headに対するCIとログで別途検収する。上位CLI/Studio/SHO/実Provider、Owner意味受理はNOT RUN。

## 15. 次の最小実装への判定（図面Tooling側で独立に所有しないもの）

- **L0/Source**: #3262/#2621のDesign Context/Loaderを通ったSubject/Relationのみを図面の入力候補にする。旧Xead local projectionのsource-tokenやクロスビューの計数をAccepted Domainへ暗黙昇格させない。
- **L1/Graph・CrossRef**: #1124/#1031/#1125/#1126のReader/ProjectionにSource ID、Edge ID、direct vs derived、unknown、selection scopeを保持させる。不可逆ならread-only。
- **L2/CLI Consumer**: `model catalog`/inspect/trace等の既存コマンドを使い切ってから、Diagram/Matrix独自不足を最小契約として補う。新コマンド・新MCPスキーマを先に標準化しない。
- **L3/Studio Consumer**: 同じ構造をGraph/Table二つの標準表現で照合し、最初の可逆な図上編集1件を既存Proposal/Acceptanceへ往復させる。Sourceの一意性・現行Revision・Selection epochを厳守。
- **L4/SHO Consumer**: 同じGraphを引用するだけの読み取り経路と、SHOのMermaid風Authoring本文／図解Assetを別正本で扱い、旧図・未取得・秘匿・納品欠落を説明する。

このL0〜L4は説明用の作業分解であり、既存G0〜G4やP0のGateを新設・置換しない。図面Core/Runtimeに特別扱いを固定する前に**業務三視点とHTTP Response Graphという二つの異質Consumer**へ同一契約が成立するか測る。

## 16. 複数キャンバスと意味要素の多重配置（SUIのCard/Islandとの横断成熟差分）

巨大な単一図面を分割しても、同じEntity/Relation/Functionの意味とSource位置を一意の正本として保持する。**一つの意味要素が複数キャンバスへ現れることを許容**し、別々のPresentation Occurrenceと、同一のSemantic Subject/Relationを分ける。これはSUIの複数Canvas/Island間でカードを共有する要件とも同型だが、TEI Data/Function/Businessの意味正本をSUI Cardテーブルへ移す提案ではない。

| Identity/State | 何を表すか | 同じ／異なる |
| --- | --- | --- |
| Semantic Subject/Relation | `data.order`, `rel.order.lines` のSource・意味・Relation identity | どのCanvasに出しても同一 |
| Canvas / View Scope | 業務全体図、受注明細図、サブシステム別在庫図 | 互いに異なる表示・探索Scope |
| View Occurrence | `view.sales.order`, `view.fulfillment.order` | 同じSubjectの別表示位置であり別の操作対象 |
| Presentation State | ノード座標、折畳み、強調、別名、viewport、レイアウトProfile | semantic commitではない |
| Interaction Selection | Canvas内で選ばれたOccurrenceと、そのSubject ID、Current Epoch | 選択同期は実行許可・Source書戻しではない |
| Document Occurrence | SHO文章内での引用図・説明・注記、参照版 | 書籍中での再掲はTEIモデルの複製ではない |

**同一Graphから複数Viewへ投影**する場合、各Viewに属するノード・関係の「見えている集合」は、元Sourceの全Related集合ではない。省略される外部Node/Relationはloss ledgerとSource navへ保持し、図が見やすくなっただけで削除したと判定しない。Canvas間リンクや同一Subjectハイライトはread-onlyで開始する。Canvas内オブジェクトをdrag/duplicate/deleteしても、明示的に選択したDomain Edit IntentとSource編集資格を経ない限り、業務Entityを生成・削除しない。

### 新Fixture・反証候補

既存[合成Input](../../reference/diagram-crossref/pressure-input-v0.json)に、`canvas.sales-overview`と`canvas.fulfillment`を追加。同じ`data.order`・`data.line`が異なるOccurrence IDで両Canvasに現れ、`rel.order.lines`も異なる描画Occurrenceを持つ。合計2 View・9 node occurrence。これらは**正式TEIグラフ表現ではない合成Pressure JSON**である。

- **CV-01**: 同じSubjectを別Canvas上で選択し、Semantic IDとSource locatorは同じだが、選択Occurrence/Epochは別と認識。9個のView occurrenceを9個の新Data Entityとして数えない。
- **CV-02**: Canvas分割・統合、位置・折畳み・Focus変更後も9 Entity / 7 RelationのSemantic集合は変わらない。外部参照とCanvasに描かれなかった多重辺をSourceから消さない。
- **CV-03**: 同じRelationを2 Canvasに描く際にRelation IDは同一、描画Edge occurrenceは別。片方のレイアウト変更で他方のMeanings/CRUDを更新しない。
- **CV-04**: 意図しないView Occurrence IDの重複、異なるSubjectへの参照、削除済みSource上のノードを含む古いCanvasを検出。独立なSelection ContextなしのPatchを拒否。
- **CV-05**: 概念的に共有されたEntityをSHOの複数箇所へ挿入した場合、Document occurrence・キャプション・局所Reviewは別。Source改訂後は該当引用図の更新必要性を説明し、全部を黙って改稿しない。
- **CV-06**: 同一Sourceに対し2つのCanvasから互いに異なるProposalを送る場合、Source Revision/Authority/対象RelationをApply時に照合し、見かけ上の並行編集成功を意味受理・業務Commitとしない。

[AI草案Expected](../../reference/diagram-crossref/expected-v0.json)にはView ID、同一Semantic Subjectの複数Occurrence、Relationの複数Edge Occurrence、Presentationのみの操作を記録し、[Fixture Go audit](../../reference/diagram-crossref/fixture_contract_test.go)へ重複View ID/Occurrence・未解決参照の**局所整合検査**とMutation 1件（view occurrenceの衝突）を追加した。ここでの合格はDiagram Renderer/Studio/SUI/SHOの実体験や、版違いのSourceに対するedit grantの正しさを証明しない。SUI固有CardのSchema/OKF意味・Shō WriterのOwnerは各製品に残す。

## 17. 最新実装・CIを反映した実証優先順位（11/07）

[統合候補Draft #3303](https://github.com/hat47x/tei-platform/pull/3303)は、先の#3236の三視点・標準Representation設計を**現mainへ移し直す別4ファイルの計画Owner**。本#3263は図面/CRUDの合成Reference・Mutation契約と最小可逆編集検収を所有し、#3303のRP/MV/CX/WX/QS/DM・`MASTER_PLAN.md`追記を複製しない。P0-M（モデリングGraph/CRUD）、P0-R（実Product Function→Query→Provider Read）、P0-C（同一ID/Scope/Revisionで接合）を独立の証拠として管理する。**CRUD宣言`R`は実Provider Readとは異なる**。X-TEAのBusiness Action→Function参照による派生可能性も業務操作の現在実行資格ではない。

[main統合済み#3296](https://github.com/hat47x/tei-platform/pull/3296)で、既存Context Gateが実CLI9応答と16の合成拒否対照を実測した。一方、`boundedReference=partial`、`qualifiedTestGeneration=false`であり、Function/Business/RelationのRequired Closureおよび図面・CRUDの製品生成を証明しない。このため本#3263の最初の未完成辺は今なお**実Source/Loader→Graph・Function/Business Relation・CRUD Projection→CLI対象Source/Cell逆参照→実Consumer受領**である。Test Case生成T1は#3273、BOM版Matrixは#3267/[Draft #3297](https://github.com/hat47x/tei-platform/pull/3297)、Plugin Inspector D1は#3279/Studio別Owner、Product Readは#2586が管理する。

現`tei.cli.discovery/v0`の`view`はread-onlyの広告に`-execute`系を含めており、[Issue #3182](https://github.com/hat47x/tei-platform/issues/3182)/[Draft #3302](https://github.com/hat47x/tei-platform/pull/3302)が実際のファイル生成を確認して修正を提案中。したがって**read-onlyの`model catalog/inspect/trace/source-index`等を優先し、`view -execute`系を図面閲覧の無副作用前提としない**。Plugin宣言のEffect/ReadOnlyから外部効果・ファイル出力の不在を無検査に決めない。#3302の最終merge前でもコマンド呼出しを明示的に区別する。

### PR #3263 exact-HEAD CI障害の根拠と所有範囲

旧HEAD `aeea395ca79c2fe84e068cb94d6f43c8982e1bdf` に対する[Repository Health #42373811364](https://github.com/hat47x/tei-platform/actions/runs/42373811364) は**completed/failure**。実ログでは `gofmt -l . is empty` がPASS、追加した `reference/diagram-crossref` を含む全Go package testはPASSとして列挙されている一方、`tools/repository-plan-check`が以下の既存計画整合性でFAIL: (1) RQ-035の受入・Gap Audit未登録、(2) 追加設計6件の `plan/design/README.md`索引欠落（本PRの`THREE_VIEW_...`を含む）、(3) `plan/evidence/README.md`のPlugin-first D1索引欠落。**全Health FAILをGo Fixture実装のFAILと読み替えず、同時に「全体CIは緑」とも表現しない**。

RQ-035/RQ-036・横断Indexは[Draft #3297](https://github.com/hat47x/tei-platform/pull/3297)や別Ownerが並列修正中。本PRは自身の設計文書の索引確認/追加だけを担当し、他領域のRequirement本体/受入対応/Plugin D1文書索引を競合して書き換えない。#3297等がmainへ統合後、#3263の最新mainで対象ファイルを照合し、exact-head Health/合成Fixture Testを再資格化する。Goテスト内のMutation数やSource/Expected更新により旧`aeea395c`の成功範囲を現HEADへ自動引き継がない。

**本節時点の新CI/Owner受理/実Studio・SHOは未確認**。全図面の編集・CRUD Matrixの実Product投影・実Provider読取完了を、Fixtureが増えたことだけから主張しない。

## 18. 不完全なFunction CRUD宣言で「既知」を消さない（11/07 実コード修正）

他チャットのDesign Context／未取得・部分集合／C0/C1テストの議論を、既存`reference/diagram-crossref/fixture_contract_test.go`の**実コード上の誤判定**へ還元した。以前の監査関数は`DeclaredCrudComplete=false`の場合、明示`uses`が存在しても全Dataセルを`unknown-not-enumerated`と判定していた。基底Fixtureの`fn.external.dispatch`は`uses=[]`であるため、この誤りは既存正常系／5破損Caseからは検出されなかった。

### 18.1 修正した意味契約と最小反証

| Function×Dataセル | 列挙された`uses` | Function宣言の網羅性 | 正しい局所結果 |
| --- | --- | --- | --- |
| 既知`R`セル | 明示`R` | `false` | **`R`**（既知の肯定事実を維持） |
| 未列挙セル | なし | `false` | `unknown-not-enumerated`（不存在とは言わない） |
| 未列挙セル | なし | `true` | `none`（**宣言の限定Scope内のみ**） |
| 既知`C/U/D/R`セル | 明示 | `true` | 宣言した操作を維持 |

新`TestPartialFunctionCRUDKeepsKnownCellsAndUnknownRemainder`は、既存公開Fixtureをコピーして`fn.external.dispatch`へ既存の`data.customer=R`を一つ追加し、期待票にも`R`を明示する。この**正例が通ること**に加えて、(a) 既知`R`を`unknown`に潰した偽陰性、(b) 未列挙`data.order`を`none`へ昇格した偽陰性を、それぞれ拒否する。元Fixture/Expectedの保存内容やDomain Oracleは書き換えず、監査関数と同じGo packageの有限テストとして加える。

### 18.2 所有と受入の上限

- #3263：合成Fixture・Function CRUD表示の局所意味。**不完全であっても実際に列挙されたセルを消さない**という不変条件を保持する。
- #3262：実Source/LoaderからFunctionと`uses`をどの範囲で発見・受領・解析できたかのDesign Context閉包を担当。未発見Functionの操作を`none`と推測しない。
- #2514/#1180：この最小正例と**2種の拒否対照**を既存Independent Expected／Scenarioの判別力へ引き継ぐ。監査関数自身の計算値を独立Oracleとして扱わない。
- #2586/#2506：同じ区別を実Product Query／Read Slice／Function Data Contextへ接続するが、CRUD宣言`R`は実ProviderのRead実行や行集合完全性ではない。
- #2507/#2497/SHO：表・図・文書で空欄に変換する際、局所的な`unknown`と確認済み`none`と明示`R`が視覚上合流した場合、逆編集や不存在推論を許可しない。

**証拠状態**：この追補時点でGoテストSourceはブランチへ追加済みだが、当該commitの正式Repository Health/Go Test結果は別途照合する。仮に局所テストがPASSしても、実TEI Parser・Graph/CRUD projector・CLI/Studio/SHO Consumer・Function/Business全Required Closure・独立Owner Oracle・実Provider/Host Effectは**未検証**。新しいCore Enum/Parser/Runner/Authorityは追加しない。

## 19. 現mainへ追随したSource入場の最小縦断（2026-11-08）

### 19.1 最初の欠落は描画器ではなくReader→Meaning→Cross-reference

同日の実ドッグフーディング[#3346](https://github.com/hat47x/tei-platform/issues/3346)により、`docs/`・`plan/`のMarkdownをWorkspace manifestの`sources`へ指定しても、`managedFiles`上の発見と`readByLoader`は一致せず、現Loaderでは`*.tei`以外の文書が設計意味として読まれない場合があると確認された。**ファイル存在・Blob読取・Roleとしての意味Parse・Accepted Meaning・Current Scopeでの投影**を独立に評価する。`*.tei.md`を含むMarkdown Authoring契約は#2621、Everything is Dataは#2619、Workspace読取りの製品挙動は#3346/#3262のOwner。SHOのMarkdown原文ReaderもTEI Parserと同一であるとは限らない。

**P0-M最初の実Product試験では、当合成JSONを受理済みTEI Sourceと偽装して使わない。** 現時点で実Readerが認識する`reference/workspace`等のSourceに存在する一つの実Data/Relation/Function/Business対象を先に選び、`workspace-files` `model catalog` `source-index` `inspect` `trace`の実Resultから、必要Sourceの所在とRoleを一つのSnapshot/Scopeに対応付ける。図面表示とCRUD導出に必要なRelation/Function CRUD/Business Actionのいずれかが得られない場合、その欄を **`unavailable/unsupported`** とし「Graph完成」ではなく**First Missing Edge**として報告する。既存`modelref`・ER/物理表Reference・Business DFD DOT・Studio Graphは局所のProducer能力比較に利用できるが、別Sourceや別Loaderからの合成結果を仮想の同一Accepted Worldとして扱わない。

| 用途 | 閉包で必要な根拠 | 出力資格の停止線 |
| --- | --- | --- |
| Data Graph | Entity/RelationのRole、ID、方向、多重辺・Key/FD、Source ref、Scope | Edgeが未解決、旧Xead固有Tokenだけ、非公開Edgeの識別不能 |
| Function Diagram/CRUD | Functionの宣言、Data参照と明示CRUD、列挙範囲・否定依存 | 不完全なFunctionの明示Rを消す、未列挙をnoneとする、actual Provider Readと混同 |
| Business/Function対応 | Action/Task、manual、Functionへの参照、関係の出典と粒度 | manualをMissing Functionエラー化、潜在pathをactual business Effect化 |
| 複数Canvas・SHO引用 | Semantic IDとOccurrence ID、Source rev、Projection/Renderer rule、Disclosure | Viewが非表示にした辺を削除扱い、古い引用図からSourceを書戻し |
| Product Function実Read | 実Query SQL raw+args/physical Provider/結果Slice/取得時点/Attempt | GraphやCRUD表のRセルで実DB読取があったと主張 |

ここから汎用CLI/MCP/schemaを先に増やさず、既存Read-only Context Packetから用途相対の欠落を示す最小Consumerを先行する。P0-MをP0-Rの完成待ちにせず、P0-Cでのみ同World・Scope・Role・Revision・Grain・Consumer Receiptを突き合わせる。

### 19.2 同一性・鮮度の比較範囲を明示する

実CLIの[#3345](https://github.com/hat47x/tei-platform/issues/3345)が示す`view -format json`の`semanticFingerprint`はApplicationが使う名前空間に限定され、`worldFingerprint`や`sourceIndexFingerprint`と同じ値にはならない。比較範囲が異なるFingerprintを等値検査して偽の不整合・偽の再資格化を起こさない。Source/World/Projection/Context/Build/Asset・Consumerの各版は**対象と何をHashしているか**を明記し、一方が存在しない場合はcurrent-use資格を未確認とする。

[#3347](https://github.com/hat47x/tei-platform/issues/3347)では、CLIの`diagnostics`配列の二重ラップ、`requirements.result`の配列形、builtin標準定義が`model catalog.entries`に混在、Plugin Lockの二重版といった**実応答形の相違**が観測されている。Diagram/MatrixのConsumerは、既存Schemaのversion/mode/成功・失敗形を確認し、知らない形を勝手な標準形へFlattenしない。非互換な応答変更はCLI契約Ownerへ返し、図面専用の第二Envelopeを作らない。

同じ正本を後続のStudio/SHOへ届ける際には、**実Producer出力→Asset生成→対象Consumerでの受領→表示/参照資格**を別観測とする。Hash一致は内容証拠の一部であり、実際に誰が閲覧・操作できるかのGrant/Disclosure資格ではない。表示のViewport/Canvas partialと、Source全体の完全性を同じ列で表さない。

### 19.3 P0-M/P0-R/P0-C・Testの最小切り分け

| 次の実証 | 作業と判断可能な出力 | 検証/Owner |
| --- | --- | --- |
| **M1 Source Admission** | 現実の1 Source/Roleの`declared→discovered→loaded→parsed→bound`とScope/Reader mode、Source locatorを固定し、業務Relation・CRUDのRequired Gapを記録。ドキュメントのRaw保存とMeaning入場は別 | #3262/#3346/#2621。Sourceは実際に現行Readerが扱う形式に限定 |
| **M2 Graph/CRUD** | 同じ実Acceptedまたは資格が明示されたModel RevisionからData/Function/Businessの各1 projectionとFunction×Data matrixをHeadless出力し、Edge/Cell→source refを確認 | #1124/#1125/#1126/#2497。ReferenceとProduct Pathの区別 |
| **M3 Consumer/Edit** | 同じSubjectのCanvas間選択同期、一つの**可逆**Relation属性変更のPreview→Accepted Source Patch→reparse→再投影、非単射拒否 | #1124/#2500/#997。意味受理は既存Owner |
| **R1 Physical Read** | `order_entry.recent_orders`の実Function選択SQL byte+args→PostgreSQL Provider→Header rows→Function Data Contextの同一Attempt観測 | #2586/#2506/#2507。既存Reference PG局所PASSとは別 |
| **C1 Join** | M2のData identityとR1のResult occurrence/Scope/Read Epochを照合。異なるWorkspace・旧Result/部分取得・単に`CRUD R`の例を拒否 | #3262/#2514/#1180。互いの不存在や業務完了を推論しない |
| **D1 Delivery** | CLI→Studioと、別にSHO図引用へ出力し、受信bytes・Asset/Source rev・Disclosure/文書Occurrence・Lossを検収 | #997/SHO #5/#6/#19。文書出版・外部配信は別 |

いずれも**新Gate名ではなく既存Ownerへの手順・停止条件の分解**。P0-Mに未取得があってもP0-Rの独立実証を進め、P0-Cだけは双方の現在資格が満たされるまで保留する。Test側は既存#2514独立Expected・#1180 Scenarioを使用し、今回のAI草案ExpectedやGo Fixture Test成功を実製品の合格として採点しない。

### 19.4 Go Fixture CIから得た正しい結論とブランチ更新

旧PR HEAD `2bc0d0f3aa3960751d57745209d94a8253538d72`の[Repository Health #42385303844](https://github.com/hat47x/tei-platform/actions/runs/42385303844)は**failure**。実Logでは追加`reference/diagram-crossref` packageなどのGo tests、`gofmt -l . is empty`はPASS、唯一の失敗は`tools/repository-plan-check`による**本設計文書が`plan/design/README.md`に未掲載**というもの。ほかのRQ/索引不足をこのHEADの失敗原因に混同しない。

そのため旧ブランチを現mainの固定commit `6cf510d6301a1e133ebbe69ad7bf004f03361b54`と**履歴を保つ二親merge**で統合し、最新main内容を保持した`plan/design/README.md`へこの設計文書の索引1件を追加。差分は本Design、合成Input/Expected、Fixture Go Test、Indexの5ファイルのみ。二親commit `6d3eb7f6cf7fc68c4401947faf9cad157ae565af`の作成後比較で**main比ahead22/behind0、5ファイルのみ**を確認した。これは既存PRのreview候補更新であってmainへのmergeではない。

**新HEADのexact CI（Health/脆弱性）や新実Consumer・Owner正式採用は別に確認する**。古いGo PASSや旧失敗理由を新HEADへ無条件転記しない。

### 19.5 現行Typed GraphとCLI Discoveryに固有のGap（実Source静的照合）

`internal/modelref/model.go`の`tei.modelref.v0`を静的確認した。現`Node`は`Key{Scope,Subject}`を持ち、同一Keyの重複を検査する。一方、`Edge`は`From Key / To Ref / Relation string / Evidence[]`であり、**Edge自身に独立した安定ID・View Occurrence ID・表示位置は含まれていない**。現`Validate`はSource/Target参照、Extension Rule、循環、Evidenceを検査するが、これをそのまま図面編集の一意なRelationインスタンス・Source spanの証明にはできない。Relation欄が異なれば並列の型区別は可能でも、**同一端点・同一関係種別の2件を表示・選択・Sourceへ逆参照する保証は別に必要**。この事実をCoreへ即時EdgeID追加する根拠とはせず、#1124/#2497/#2620の既存標準Graph/Source Refの表現力を先に調べ、欠ける時だけ局所Adapterのtyped occurrence+source locatorを検証する。

`tools/reference-business-dfd/main.go`は`procedure/review/set`という明示のReference FixtureとWorkspaceを受けるread-only投影Adapterであり、DOT/JSON・select/previewが存在しても**本番のAccepted Business手順を全件Sourceから導出できたことは示さない**。ここでもGraph用途/出典とAccepted Meaningを別に保持する。

またmain統合済み[#3333](https://github.com/hat47x/tei-platform/pull/3333)ではCLIの`view`が記録系`-operation-log`/`-lifecycle-state`により実ファイルを書き得ることを`writeFlags`と実実行Testで明示した。以前の[#3302](https://github.com/hat47x/tei-platform/pull/3302)は別の修正Draftであり、最新mainで`view`の`effect:read-only`という**名前上の分類だけ**を外部副作用なしの根拠にしてはならない。図面検査は`argv`の実Variantと書込Sink、ConsumerのDisclosure/Tool Deliveryを分ける。

最初の実Probeは「同一Scopeの`modelref`候補Graphから**異なるStable EdgeIDを持つ2 Relationship occurrence**を元Sourceへ回収できるか」「`view -format json`のApplication局所FingerprintをWorld/Source Indexと混同しないか」を判別し、満たさない場合は**read-only/unsupportedと不足Contractの位置を返す**。ModelRef/Studio/SHOの静的型が合致しただけで製品機能の対応済み扱いにしない。

## 20. 実modelrefのGraph受入とEdge Identity欠落の独立実証（11/08）

本PRの[modelref Fit/Gap Go Test](../../reference/diagram-crossref/modelref_fitgap_test.go)を追加した。これは単に合成JSON同士を比較する従来のFixture auditとは違い、**実際のTEI `internal/modelref` の `Graph`/`Validate`/`Diff`を動かすテストコード**である。ただし入力は引き続き本PRの合成Data/Relation JSONであり、実Source Loader・Function/Business Parser・Accepted Meaning・Product CLI/Studio/SHOを経由しない。テスト内の写像は一時的な実証Adapterであり、Core/新Projection Schemaを定義しない。

### 20.1 なぜRelation TypeとRelation InstanceのIdentityを分離するか

現在の`modelref.Edge`は`From Key`/`To Ref`/`Relation string`/`Evidence[]`を保持するが、`EdgeID`を含まない。今回の合成入力は、`data.line→data.sku`という同じ端点に対して`rel.line.sku.primary`と`rel.line.sku.alt`の二つの**別の業務関係Occurrence**を持ち、両方とも型は`reference`である。

- **MRX-01（正例）**: 合成9 Data Node・7 Edgeを型付き`tei.modelref.v0`へ一時写像し、`Validate`がすべての端点とEdge Typeを許すことを確認する（Source provenanceは`proposal`として留める）。
- **MRX-02（負例・限定検出）**: 同じ`From/To/Relation/Evidence`である2 Edgeを、`modelref`自身では区別できないことを確認。**両方に別の元Source Relation IDがある事実は、現在のtyped Graphから復元できない**。したがってGraphからの逆Patchは`ambiguous`・`read-only`にとどめる。
- **MRX-03（正例だが不十分）**: 片方のEdgeを失うと`modelref.Diff`が`reference-changed`を報告できることを確認。ただし「どちらの元Relationを削除したか」は報告できない。**変更の検出**と**変更対象の一意な特定**の適格性を区別する。
- **MRX-04（次段）**: ID/Source locatorを持つ既存標準Graph/Source RefのCandidateを調査し、型とOccurrenceを保存できる場合だけCLI Headless出力のEdge→元Sourceを要求する。`EdgeID`を直ちに`internal/modelref`へ追加してABIを固定することはしない。

**反例設計の補正**: 旧N01は「2本あるEdgeのうち1本を落としたら件数差を検出」という局所弱判別に過ぎない。**ModelRefが2本の同型Edgeを表示・保持できても、片方を選択して元Sourceの`rel.line.sku.alt.optional`へ可逆編集できるとは限らない**。独立Expected（#2514）では `Graph shape/count`、`Edge identity/source locator`、`edit target uniqueness`、`current revision/authority`を別のAssertとして持つ。新テストでModelRefのGraph全体・Data/Function/Business全意味対応が完成したとは主張しない。

### 20.2 他チャット最新の実Product能力とモデル解釈の分離

| 既存の実証・所有境界 | 今回の図面・CRUD Consumerへの反映 |
| --- | --- |
| [main統合 #3354](https://github.com/hat47x/tei-platform/pull/3354)：実CLI **11応答**と限定**32 Consumer Controls**でRule/Relation/Sourceの不整合を拒否 | `Required`は実Function/Business/Schemaを含まない`partial`。Graph 9/Edge 7・CRUD 4×9の完全性証拠とはしない。Reader/Parser Role入場とData Meaning Scopeを別票で返す。 |
| [main統合 #3353](https://github.com/hat47x/tei-platform/pull/3353)：空集合のHandoffについて`HandoffConfirmed`未確認なら完成扱いしない | Source/Relation/CRUDの既知0件、Provider空集合、外部受領確認を**三つの独立条件**で扱う。取得0件とモデル中に要素が無いこと、結果Asset未受領とBusiness完了を混同しない。 |
| [Draft #3371](https://github.com/hat47x/tei-platform/pull/3371)：StudioHostの`InspectionService`型付きHost経路、未提供・不一致拒否 | Diagramも暫定的に既存Read-only Inspectorを入口にできるが、**CLI直接ReaderとHost選択Pluginの実Provider一致は未確認**。別ホスト・別Coreではなく#1824/#3279のPlugin/Composition Ownerへ渡す。 |
| [Draft #3372](https://github.com/hat47x/tei-platform/pull/3372)：Workbenchの物理FK/UNIQUE・観測Revisionを失わないReverse情報の保持 | 物理FK/UNIQUEは業務Relation/Keyの**Structural Candidate**であり、Ownerが受理するまでSemantic Edgeへ昇格させない。逆向きWorkBench→Modeler→SHOで`observed`・`candidate`・`accepted`を保つ。 |
| #3303：標準Representation Topology、Function-owned Instance、実Read/交差表示 | Canvas Subject/Occurrence/Result Sliceの結線を共有、表現型の型/意味正本をこのPRで再定義しない。 |

**M1の次に進める条件**: `reference/workspace`から認識された実Source/Subjectの情報で、`readByLoader/parsed/bound`の各段階と、標準Graph内RelationのIdentity/Sourceを明示的に対応させること。対象の関係が解決不能なら、同型Edgeへの**Read-only表示**だけを受入し、Source逆編集を拒否する。Function CRUDの列挙が不完全なら、明示された`R/C/U/D`は残し、未列挙セルはUnknownとする（§18）。この停止線が成立するまではStudio/SHOのRenderer/E2Eの成功を宣言しない。

### 20.3 テスト資格の明確化

- **Fixture Audit**: JSONの内部整合、既知の局所破損の拒否。独立Owner Oracleではない。
- **`modelref` Fit/Gap**: TEI実Go packageの型・参照・Diffに対する合成入力の限定実行。Source Loaderや製品図面生成の実証ではない。
- **Source Admission**: 実Workspaceに管理されるSourceがLoader/Parser/Roleに入場し、同一SnapshotでSubject/Relationが解決するか（#3262/#3346）。
- **Product Projection**: 上記のReal SourceからData/Function/Business図面/CRUDを作成し、Edge/Cell→元Source・Loss/Scope/開示の現在資格まで辿れるか（#1124/#1125/#1126/#2497）。
- **Reverse Edit/Consumer**: Canvas/Studio/SHO/実Provider/Hostで、一意性・適用可能性・実配送/実効果を既存Owner/Scenario/独立Expectedで検証する。

上の5資格を一つの`PASS`としない。新`modelref_fitgap_test.go`のgo test、gofmt、GitHub Health、最後の変更HEADは**その実Runで確認してから**判定する。
