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

## 21. 外部逆解析・Provider Source変化・同一Consumer受入（11/08）

### 21.1 REA／Workbenchからの構造観測はModeler正本の別入口

[REA連携 #3394](https://github.com/hat47x/tei-platform/issues/3394)の現行「二経路」設計を本三視点Consumerへ接続する。**REA JSON・SQLite Snapshot・旧X-TEA XML・物理DB FKをTEI標準`*.tei.md`のAccepted Sourceとして直接Loaderへ投入しない**。具体的には次の二経路の責任を分ける。

| Lane | 入力・出力 | 図面とCRUDが宣言できる範囲 |
| --- | --- | --- |
| External Observation（read-only） | 元Source bytes／Snapshot、実行したREA/DB Provider、Raw Evidence／FK・UNIQUE・実SQL発見、Source locator、観測時刻、unknown／coverage | **観測された構造・Potential CRUD・未解決Relationの候補図**。元のバグやSQL UPDATE構文を「あるべき業務仕様」またはActual Commitにしない |
| Explicit TEI Authoring/Acceptance | 選択した候補を専門家可読`*.tei.md`等のSource Draftへ変換、実Loader/Parser/Role、Source span、既存Ownerの意味受理 | **適格な意味モデルの三視点・CRUD図**。Reader未対応なら`parser-unbound`で保留し、SHO上のドラフト保存だけでは受理済みとしない |

**個々のIDを合同化しない**：REA Evidence/Module ID、外部DBの物理Table/Key、TEI Candidate ID、受理済みData/Function/Business ID、Graph Edge ID、Canvas View Occurrence、SHO Document Occurrenceは別名前空間。多対多の出典付きMappingとして扱い、見かけの同名・同型・同digest・同Source pathだけで一意同義にしない。SHOの再編やREA再解析は、TEI Sourceへ暗黙のPatchを送らない。外部REAの実解析・Bundle readback、`*.tei.md` Authoring Reader、SHO Writer/再開、旧新Runtime比較はいずれも本PRでNOT RUN。#3394のT0/T1-A/T1-B/T2が所有し、本PRに第二のImporter/Parserを設けない。

[Workbench Draft #3372](https://github.com/hat47x/tei-platform/pull/3372)で保持する物理FK/UNIQUE・Index・CapturedAt/Fingerprintも同じ**Physical Observation→Structural Candidate→Owner Accepted Relation**の順序を適用。複合キー・非ユニーク・式/部分Index・外部表が存在するとき、UIの「Relation 1本」に単純変換して多重性や制約のlossを隠さない。

### 21.2 ConsumerのWorld revision一致だけでは外部Sourceの鮮度は保証できない

[main統合#3477](https://github.com/hat47x/tei-platform/pull/3477)の同一WorkspaceのSource-only改訂拒否を、外部Provider Source全体への保証と誤拡張しない。[Draft #3479](https://github.com/hat47x/tei-platform/pull/3479)は**importしたProvider raw**がConsumerの`Source Index`に入らない場合、ConsumerのWorld/Model/Source Index digestが不変でも元Providerのraw bytes・locatorの現在版が変わり得る反例を検討中。本PRではimported Provider側の試験コードを複製しない。

Diagram/CRUD/SHO引用の資格に必要なのは、消費先Workspaceだけの`sourceIndexFingerprint`ではなく、**対象を導出するのに現実に使ったSource/Import/REA Providerの依存閉包・取得時点・exact byte/locator・Plugin/Rule・開示・Reader mode**である。閉包が未観測なら「現在も有効な図」を示さず、Read-only Candidate/partialへ降格する。物理Evidence fingerprintをStable Semantic Relation IDへ転用しない。

Go側は[modelref Fit/Gapテスト](../../reference/diagram-crossref/modelref_fitgap_test.go)を拡張し、**合成GraphのEdgeでSource Observationのfingerprintのみを変えると`reference-evidence-changed`になり、SourceTokenがあってもartifact fingerprintがなければ`unbound-source-evidence`で拒否する**ことを検査する。これは`modelref`が**明示的に渡された**Evidence変化を検出できるという局所的能力であり、実Loaderが外部Provider rawを読んだ、入力closureが完全、現在Grantを満たす、という証明ではない。未解決のEdge IDや編集対象一意性も引き続き不足（§20）。

### 21.3 Headless／CLI／Studio／SHO間の引渡し条件

- [CLI実ドッグフーディング #3455](https://github.com/hat47x/tei-platform/issues/3455)で、`view`・`source-index`・`knowledge-context`等はWorkspace identityが応答から欠落する実例が確認された。修正候補[PR #3467](https://github.com/hat47x/tei-platform/pull/3467)は未統合として扱う。**現行Responseで取得されていない欄を補って等値比較してはならない**。一時的にはCallerがrealpath/manifest byte・actual flags・result modeと対象Revisionを記録し、正しい結合が証明できなければ`incomplete`。API正式対応は#3419/#3455のOwner。
- Studioは選択したInspector Plugin/Capabilityが実際に呼ばれたWitnessを検査する。Graph/CRUDのNodeを表示しただけ、CLIと見た目が同じだけでは二Consumerの同一Model/同一Provider適格としない（#3279/#3371）。
- SHOではSource/Document occurrence、Renderer/BOM、納品Asset、Reader readbackと開示を区別。Source/Provider変更後の旧引用は歴史図として表示可能でも、Current/Edit資格は再検査が必要。
- [HTTP Graph main統合 #3478](https://github.com/hat47x/tei-platform/pull/3478)は既存Go Hostの実Handlerの10経路に限定したAudit/Branch検査を進めた。**HTTP 202やAudit completedを業務Effect確定としない**。同じ標準Graph/Diagram Rendererの別ドッグフード対象に使うが、Host PathとTEI一般Graphが接続済みとは主張しない。
- [Ontology/Twin長期Issue #3475](https://github.com/hat47x/tei-platform/issues/3475)は、型付き対象・Relation・時系列・Decision Inquiryを本基礎の先へ延長する構想。SEIへの判断移譲・SUI Canvasの実装・KON Dataspace Provider化は**セルフビルド/SUI基礎より後の別Owner**に置く。三視点Graph/CRUDのP0に第二Ontology CanonicalやKON必須依存を持ち込まない。

### 21.4 受入ケースと優先実装の再確認

| 試験（新しいRunner/Gate番号ではない） | 入力対照 | 期待する確かな結論 |
| --- | --- | --- |
| Source evidence drift | Node/Relation意味不変・観測Source fingerprintだけ変更 | 現`modelref.Diff`は局所`reference-evidence-changed`。**実Providerの変更検知**は#3479が別に確認する |
| Unbound imported observation | SourceTokenあり・fingerprintなし | `modelref.Validate`は`unbound-source-evidence`、正当なGraph→Source逆編集不可 |
| REA vs TEI Source | REA Raw JSONを`*.tei.md`と改名 | 未対応Reader/Roleを正式Authoringに昇格しない。T1-Aだけ局所PASSでもT1-Bはpending |
| Physical FK→Relation | 複合物理FK・同名別Table・業務意味未受理 | Physical/Structural Candidateを図示し、Accepted Business Relationと別表示。Owner reviewを要求 |
| Multi-Consumer | CLI ContextのWorkspace未証明／SHO旧Source引用 | `partial/unqualified`で継続・表示は可能だが、元SourceへのPatchとCurrent Claimを拒否 |
| HTTP graph | HandlerのGuard/Audit順と実Business Outcome | 実Host Pathの局所観測を保持。静的GraphとHTTP応答だけでEffect確定しない |

**当面の最短実装**は変更しない。M1で実Loader/Roleへ入場できた一つのSourceを選び、M2でStable Subject・2つの別Relation occurrenceとCRUDセルをSourceへ逆引きできることを最初に示す。次にM3で**一意な1本だけ**のSource Patchを実行し、非単射・source-only drift・権限なしを拒否する。M1が未成立なら今回のSource Evidence TestのPASSをM2/M3に昇格しない。独立Expected・実Runの適格性は#2514/#1180/#1163がOwner。

本節の情報は既存Ownerの実装事実と未検証構想を分けた計画更新であり、Core API/ABI、Protocol、REA Adapter、Twinエンジン、Registryや新受入Gateを正式採用するものではない。

## 22. 実Workspaceの限定M1入場とADR-0048の共通API契約（2026-11-08）

### 22.1 検証対象をSynthetic Graphから実Sourceの二つのSubjectへ進める

従来の`modelref_fitgap_test.go`は合成9 Node/7 Relationを実`modelref`へ渡す**構造・Diffの局所試験**だった。これと独立に、今回[real_workspace_admission_test.go](../../reference/diagram-crossref/real_workspace_admission_test.go)を追加。現行`internal/operations.InspectSubject`を実際の`reference/workspace`に適用し、次の**二つの明示Subject**を読む。

| Subject | 操作 | 確認する最小証拠 | できない推論 |
| --- | --- | --- | --- |
| `query:order_entry.recent_orders` | `operations.InspectSubject(root, query)` | `tei.operations.inspect/v0`、Source Index上の参照、Canonical JSON、World/Source Index fingerprint、Workspace identity | 実SQL/Provider Read実行、FunctionのCRUD、Queryと下記Relationの依存確定 |
| `semantic-relation:sales.order.lines` | 同じInspector、別指定Subject | 同じ操作型で明示Relation IDがSourceへ遡れるか、観測Fingerprint・Workspace identity一致 | すべてのData/Relation/FD閉包、Graph Edge Instance ID、Business Action・可逆Patch |

この試験は**実TEI Source→Workspace Loader→World＋Source Index→型付きApplication Operation**に限る。候補を勝手に生成せず、`SourceStatus=indexed`とSource refの非空・スキーマ・Workspace identityをテストする。読み取りは2回の独立`InspectSubject`呼出しなので、結果の`worldFingerprint/sourceIndexFingerprint`が同じでも原子的な同一Read SnapshotやHost Grantの保証にはならない。Source Indexに含まれない外部Provider rawはここでは追跡しない（#3479）。「同一Workspaceの二つの適格なSubjectをReaderから引けた」という上限でM1を評価し、**M2三図・CRUD自動生成ができたことにしない**。

### 22.2 採用済み共通API方針とMCP参照実装の現在位置

[ADR-0048](../adr/0048-shared-application-api-and-transport-adapters.md)（[main統合PR #3439](https://github.com/hat47x/tei-platform/pull/3439)）は、**CLI・MCP・HTTP/Web・Desktopで共通の型付きアプリケーション操作を使用する方針を採用済み**。APIはHTTP Serverを意味せず、`internal/operations`を出発点とする。よって本Diagram/CRUD ProducerをCLI専用コマンド本体として増設するのでなく、既存の意味・ReaderのOwnerと接続する共通Operationの可否を先に比較する。標準データGraphに特殊なGraph Hostや第二Registry、巨大万能dispatcherを設けない。

[main統合PR #3403](https://github.com/hat47x/tei-platform/pull/3403)の`tools/reference-mcp-readonly-adapter/main.go`は、**実装済みのread-only参照MCP入口**である。起動時にWorkspaceを固定し、4 tool（`tei_model_catalog`、`tei_inspect`、`tei_trace`、`tei_validate`）を既存`internal/operations`へ委譲する。以前の「MCP入口なし」は誤り。逆に、`tei_source_index`、`tei_diagram`、`tei_crud_matrix`、`tei_edit`等のMCP toolは**この参考Adapterの公開4 toolに存在しない**。本PRで公開MCP/CLI Schemaを推測・追加しない。

[ADR-0048 §2.3〜2.5](../adr/0048-shared-application-api-and-transport-adapters.md)の意味を図面Consumerに適用する。

- **共通Operationで同じ意味の結果**と、CLI exit/stdout・MCP JSON-RPC/tool content・HTTP status/Problem・Desktop IPCのTransport Shapeを分離。JSON-RPCの数値Error Codeを業務/操作のFailure Codeの代わりにしない。
- **MCP参考Adapterの内部Resultは全CLI応答と同型・同じ成否ではない**。採用ADRでも`validate`診断・`isError`・`result`の違いが未解決と明記されている（#3420）。指紋や数欄一致を全操作・全失敗の同値性として扱わない。
- MCP側の固定Workspaceはモデルからのパス指定による拡大防止であり、OS symlink・Root置換や許可されたActor/Disclosureの独立認証証明ではない。Graph/CRUDにより非公開Entity/Relationの件数・存在を漏らさない。
- CLIやStudioの`view`等の「read-only」は**意味正本を変更しない**ことと、ファイル/外部効果の不在を同時に保証しない。実Invocation VariantとWriteFlags、Hostの適用権限を別に観測する。
- 初期のDiagram/CRUDがまだOperation未提供なら、各Transportは`unsupported`の根拠を示す。MCP専用の仮Graphを作ってCLIとは別正本にしない。

### 22.3 最初の実ReadからGraph＋CRUDへの切断面

| 段階 | 独立に検収するもの | 既存Owner・止める条件 |
| --- | --- | --- |
| **M1a: 2 Subject実Read** | それぞれの`Canonical`と`Sources.References`、Workspace/World/Source Indexの観測FP、Reader実呼出しの結果 | 今回のGo Test。2 Subjectより広い閉包や原子的Snapshotを主張しない |
| **M1b: Role/Dependency closure** | RelationのFrom/To/Key/FDとFunction→Data Context/CRUD、Business Action→Functionの必要なSource帰属を同じRevision/Scopeで解決 | #3262/#3346/#2621、全Requiredに未読込があれば`partial` |
| **M2: Projection** | 適格WorldのData/Function/BusinessからGraph/CRUDを一度導出、同型2 Relationの安定ID・Source/Cell Locator、Loss保持 | #1124/#1125/#1126/#2497。Graph EdgeID不明ならread-only |
| **M3: Safe Edit** | View Occurrence→一意Semantic Relation→適格Source Span→Preview/Acceptance→Patch→再読込 | #1124/#2500。Scope/Revision/Grant/選択epoch不適格なら拒否 |
| **D1: 共通Operationと受領** | 同じOperation入力をCLI/MCP/Studio/SHOの対応可能なConsumerへ投影し、版・Disclosure・実受領を比較 | ADR-0048、#3419〜3422、SHO #5/#19。Transport/Host差と業務意味差を区別 |

**設計順序としてM1aを単独実証できても、M1b/M2へ自動進級させない。** Source参照の件数や`worldFingerprint`だけから業務対象の完全性を推測しない。Go codeの実CI結果・exact HEAD、実Provider・CLI/MCP/Studio/SHO Consumer・Owner意味受理は後続の別票に留める。

## 23. 標準Model MapとGraphライフサイクルの図解統合（2026-11-08 横断成果）

### 23.1 RQ-036の現在の標準要求を三視点Consumerへ接続

[main統合 PR #3505](https://github.com/hat47x/tei-platform/pull/3505)により、全体Map／Data/Org topology／Event・Business Flow／Lifecycle・State／Function・Pattern Composition／Cross-reference Matrix／Execution・Source Evidenceの**標準図表・ナビゲーション要求**が[三要素Methodの図面正本 §19](THREE_ELEMENT_METHOD_DIAGRAM_CONTRACT_2026-10.md)とMaster Planへ追加済み。この文書は新しい標準図面体系を再定義するものではなく、**三視点Data/Function/BusinessとCRUDの最初の受入Consumer**を担う。

図面を複数に分ける場合でも、全体Mapから「何を表すNode/Edgeか」「この視点のScopeに何が含まれ／含まれないか」「現在の選択／Sourceをどこで確認できるか」を辿れることが基本要件。利用者の経験を前提に、未確認のフローに完成の凡例を付けない。

| RQ-036の表示 | 三視点側で受け持つ最初の事実 | まだ保証しない部分 |
| --- | --- | --- |
| 全体Map／Navigation | Source Qualified Subject/Relationの型付き参照、Data・Function・Business別Viewへのリンク、表示OccurrenceとSource IDの区別 | Event/State/全Plugin/Targetの完全集合、GUI体験全般 |
| Data topology／CRUD | Key/Relation/FD候補のSource位置、Function→Dataの宣言済みCRUD、UnknownとNo-Operationの分離 | 外部DB物理FKがAccepted業務Relation、CRUD `R`が実Provider Read |
| Business/Event・State | Action／ProcedureのSourceとFunction/Stateへの参照が実際にある範囲で投影 | Task/Action/Stateが未取得の場合の全業務完結 |
| Pattern／Operation／Plugin | `required/selected/loaded/invoked`のどの段の証拠が存在するかを図上の別列で示す | UIに出したPluginが実Invokeされたという推論 |
| Graph Source／Evidence | World/Source Index/外部ProviderのCurrent/Unknown、Edge/Cellから実Sourceへ逆引きできるか | 消費先FP一致だけで外部Import原文の現行性、Source Grant |

[Draft #3518](https://github.com/hat47x/tei-platform/pull/3518)は実`workspace.Load/Source Index/CatalogWorld`から`operations.MapWorld/MapWorkspace`を導く**read-only型付きModel Map M0**の別Owner実装候補。Data→Relation Target、Pattern→Data/Query/Capability、Query→Result Attributeの型付きLinkと、unresolved/ambiguousを区別する。Coverageは明示的に`partial-projection`であり、Event/State/実Effect/Plugin Binding等を含める完成した全体図ではない。**このPRではM0のGraph Generatorを複製しない**。#3518がmainへ統合・Source/API/expectedを資格化してから、M1aの実Query/Relation入場をM0 Subject/Linkと同じWorkspace・Source versionで照合する試験を追加する。

### 23.2 型付きライフサイクルGraphと可変HookはPresentationより深い

[main統合 PR #3510](https://github.com/hat47x/tei-platform/pull/3510)は原子的Trait/全Pathの必要保証を設計として固定。標準図面正本§19.7のMAP-11〜16は、標準Graph・Plugin Patch・実効Graph・実Target Traceを別物として示す受入候補であり、実Target生成の達成済みとはしない。[Draft #3507](https://github.com/hat47x/tei-platform/pull/3507)がAPI/Web/Desktop/CLI/MCP/VUI、AI Web Chat、B2CのHook・Interaction Flow設計とGeneratorへの引継ぎを所有。本PRは**同じ型付きGraphのNode/Edge/Flowを閲覧し、Sourceと変更資格を示すView Consumer**だけを担当する。

**固定するのは全Pathで必要な義務、固定手続き列ではない**。画面上の色/位置/折畳みはLayout変更。Plugin Hookの追加・Guard修正・Effect/Resultの経路変更はMeaningとTarget生成資格が必要なGraph Patch。実行中Attemptを新RevisionのGraphへ暗黙移行しない。Graph Rendererの完全なPathが表示されても、実実行証跡・Host Grant・副作用確定は別。UI上の流れをWorker/Hostの実効果へ自動昇格させない。

**受入対照（新Runnerではなく既存#2514/#1180の期待項目）**：

| 対照 | 合格の上限 | 拒否・未資格の例 |
| --- | --- | --- |
| NAV-A: M0→三視点 | 同じWorkspace・Subject・Source refsへナビでき、Unknown/Unresolvedを表示 | M0 `partial-projection`を全部取得と表示、同名別Subjectを混同 |
| NAV-B: Reader/Graph入場 | 実QueryとRelationのCanonical ID・Source refs・fingerprintを再確認 | Kindを入れ替えた同名照会をSourceとして受理、Raw Markdown/REAをLoader入力と推測 |
| NAV-C: Matrix／Pattern | Function宣言の既知操作だけをセル根拠へ連結 | Business potential path／SQL構文上のUPDATEを実Effectや許可済CUDへ昇格 |
| NAV-D: Graph HookとPage | Standard/Proposed/Effective/Observedの版とPathを別面で比較 | Patch受理前の変更をEffective/Executedとして表示、途中Pageを全Graphとして扱う |
| NAV-E: Consumer／Source drift | Source/外部Provider/raw/Workspace/Model/Rule版の取得範囲と更新必要性を示す | 現Source Indexが一致しただけで外部Import変更なしと断定 |

各対照は型・Scope/Role・Display Occurrence・業務効果までの**証拠不足を独立列**で維持する。MAP-01〜16・TLG/LGXの既存検証Ownerへ帰属し、独自MAP Score/独立Expected/Canonical Ledgerを新設しない。

### 23.3 今回の実M1a Go Testの確認精度を一段上げた

[real_workspace_admission_test.go](../../reference/diagram-crossref/real_workspace_admission_test.go)では現`reference/workspace`からのQuery・Relation取得に加え、**戻されたCanonical JSON内IDが要求したIDと一致すること**を検査する。また、Kind/IDの組合せを逆にした`query:sales.order.lines`と`semantic-relation:order_entry.recent_orders`は`SUBJECT_NOT_FOUND`で拒否する。Subject文字列の一部一致やSource Indexのみから意味解決を推測しない。

前HEAD `9d2830b61a7656324ba6028a9925f376d30e85f4`の[実CI Repository Health #42463319599](https://github.com/hat47x/tei-platform/actions/runs/42463319599)は**completed/failure**だが、Log上は`go vet ./...`、`go test -count=1 -timeout 60m ./...`（本M1a Testを含む）、`tei validate -workspace reference/workspace`が**PASS**。失敗の唯一の確認済み理由は`reference/diagram-crossref/modelref_fitgap_test.go`の`gofmt -l .` driftであり、手順を無視してGreenとは記さない。差分をローカル`gofmt -d`で確認し、`Scope:   scope`→`Scope: scope`の1行を修正した。**この修正以降のHEADに対するGo／Health実CIは、結果を確認するまで未資格**。

[main統合 PR #3525](https://github.com/hat47x/tei-platform/pull/3525)のPlugin/Hook/Performance Evidence、[main統合 #3381](https://github.com/hat47x/tei-platform/pull/3381)の選択済みInspector Service Port、[main統合 #3458](https://github.com/hat47x/tei-platform/pull/3458)のCLI/MCP validate同値性は、それぞれ限定した実証。M1aでInspectorのBuiltin Readが成功したことは、選択した外部Plugin/Hostが呼ばれたこと、MCP/Studio/SHOへ受領されたことを証明しない。Function実Provider・Business CRUD/Target Effectも引き続き未検証。

**次に測る一本の実証経路**は、M0（#3518の合格後）で実Data Relation・Pattern Linkを読み、M1bでSource closure/Role/Revisionを確定し、M2のGraph・Matrixの各Edge/Cellが何を宣言し・どのSourceへ戻れるかを示すHeadless Consumerである。Read-onlyを先行し、Stable Relation Instance IDが未回収なら逆編集には進まない。

## 24. Reader三入口同一性とRQ-036/037・実Providerの接合（2026-11-08 後半の横断反映）

### 24.1 実M1a合格を三入口の限定同一性へ拡張

直前HEAD `ed9b9a87f4fc1e113a51ca83a612ed01891ba292`に対する[Repository Health #42486601201](https://github.com/hat47x/tei-platform/actions/runs/42486601201)と[Dependency #42486601010](https://github.com/hat47x/tei-platform/actions/runs/42486601010)は**両方 completed/success**。このHEADで実Workspaceにある明示2 Subject（`query:order_entry.recent_orders`／`semantic-relation:sales.order.lines`）の`operations.InspectSubject`、Canon IDとSource Index参照、Kind違いによる`SUBJECT_NOT_FOUND`拒否、同World/Source Index FPの観測はGo実試験として合格した。ただし上位Consumerである三視点Graph/CRUDへの導出、受理済みモデル意味の網羅、CLI/MCP/SHO Readerを保証するものではない。

次の小さな実装差分として、同じ[real_workspace_admission_test.go](../../reference/diagram-crossref/real_workspace_admission_test.go)へ**既存`operations.CatalogWorkspace`・`InspectSubject`・`TraceSubjectSources`の三つのGo読取りOperation**を使った比較を追加する。新Readerを導入せず、同じ二Subjectについて、Catalogの一意・`prelude=false`・`indexed`、Inspectのexact Subject/Workspace identity/WorldとSource Index FP、Source Traceの実Reference・Artifact配列の一致を確かめる。

この試験の意味は**「同じ対象・同じSourceを異なる既存Readerが矛盾なく指す」**に限定する。各Operationが別々にWorkspaceを再Loadするため、同じFPが見えても**単一の原子的Read・実際に満たしたRequired closure・時点を跨ぐ外部Providerの鮮度・同一Agent Authority**を証明しない。Source-only改訂、imported Provider raw変更、ファイルアクセス権や実Source未取得を全件認定するものでもない。Source/Artifactの実Locator・Fingerprintを図面で使う場合も、Inspectの`Sources`とProvider側Evidenceを区別してCurrentを再資格化する。

### 24.2 最新のModel MapとBOMの要求番号を分離する

- [RQ-036 main採用 #3505](https://github.com/hat47x/tei-platform/pull/3505)はModel Map/全体図表・相互Navigationであり、三視点Diagramはその一部をProducer/Consumerとして検収する。[Draft #3518](https://github.com/hat47x/tei-platform/pull/3518)の実`MapWorkspace/MapWorld`型付きread-only M0は別Owner・まだ`partial-projection`。今回はCatalog/Inspect/Source Traceの既存現行APIのみを使用し、`#3518`の未統合API依存を持ち込まない。
- **Verified Composition BOMはRQ-036ではなくRQ-037へ分離する方向**。要件・PAS/Gap Auditの新番号整合は[Draft #3538](https://github.com/hat47x/tei-platform/pull/3538)、Reference BOMの2×2/Pairwise・Lock検証は[Draft #3541](https://github.com/hat47x/tei-platform/pull/3541)が所有。両方未統合の間は図面の凡例として`Plugin selected/invoked`または`BOM verified`を採用済みProduct資格として表示しない。`#3297`旧RQ-036番号との履歴的衝突を再度このPRで書き換えない。
- [索引整理Draft #3542](https://github.com/hat47x/tei-platform/pull/3542)が`plan/design/README.md`の重複を独立解消する。本PRは既存Sourceの索引1件だけを扱い、新しい全体図表／BOM文書のIndexを二重登録しない。#3542のmain統合時には**唯一の図面正本と個別設計Consumerの相互リンク**の保持を再確認する。

### 24.3 実Model/Graphと実Provider Readの横方向の結合

[Draft #3500](https://github.com/hat47x/tei-platform/pull/3500)はNative Caseの既存Source/Pattern・Persistence・Postgres Executor→テスト用Instrumented Connector→明示列Mapを**局所実Go経路**で検証する。隔離された実PostgreSQLを使う選択試験は接続情報がない環境ではSKIPであり、実DB/Current Provider Grant/Result Sliceの完全性が成立したことではない。この**Query/Driverの性能と実呼出し**を三視点M2のGraph/CRUD宣言`R`へ自動合算しない。

[Draft #3540](https://github.com/hat47x/tei-platform/pull/3540)は、main統合済みのPlugin選択Inspector入口（[#3381](https://github.com/hat47x/tei-platform/pull/3381)）を通常CLI・Studio Hostへ結ぶ`S2-reference`候補。両Consumerの選択実Callと、Build Artifact/Selected B==Actual B/Scope・Source/Host Grant・Disclosure・MCP result deliveryは別の資格。共通Operationの意味処理と、Plugin/Target/Transportの適用可否を図面へ同じ「実行可能」印でまとめない。

**図面Consumerに必要な最短結合証拠**を次の独立列で管理する。

| 領域 | 必要な比較の対 | 進捗の上限 |
| --- | --- | --- |
| M1 Source／Catalog | Reader 3入口の`Kind+ID`、Workspace、SourceRef、World/Source Index FP | **実2 Subject**の読取り一致のみ。原子的SnapshotとRequired closureは別 |
| M0 Map／M2 三視点 | `Subject/Relation stable identity`、型付きEndPoint、Edge/Cell SourceRef、Scope、Gap/Loss | #3518が統合・実結果確認後に接合。Graphが欠落をUnknownとして表示する能力が必要 |
| P0-R 実Read | Function→Query/SQL byte+args→選択Provider実Invoke→Raw Rows→Semantic Result Slice | #3500は局所合成Connectorまで。実DB/Scoped Provider・実Consumer別 |
| D1 Provider／Composition | `selected`、`loaded`、`invoked`、`received`の同一Attempt/Entry/BOM Scope | #3540はS2-reference候補で正式Plugin E2Eではない |
| SHO／Studio Consumer | Subject/Display Occurrence、Diagram Asset、Revision、実Receipt、Disclosure | Rendererの図が生成されたことと、Readerで受領・再参照されたことを分離 |

**M1aの既存API一致試験と#3518のMap Graph試験を統合して「三視点完成」と呼ばない。** 次は#3518の型付きGraph候補と、同じRoot/Subjectでのこの実Reader合格を照合し、実Relation1本とFunction/Queryの明示Bindingを現在Versionで辿る。そこで宣言・受理されたCRUDの`unknown`と`none`を検収し、それ以後に実Provider Result Slice・二Consumerの同一性を接続する。図面編集はStable Relation Occurrence/Source Locatorが得られるまでread-onlyを維持する。

### 24.4 引継ぎと資格表示

`#3492`は三視点Diagram/CRUDの**Source/Graph/図面編集・ナビの境界と局所Reference検査**に限定し、Product Map M0=#3518、実Function Read=#2586/#3500、実Plugin Call=#1251/#3540、Verified BOM=#3267/#3538/#3541、Graphライフサイクル=#3507/#3510、受理/適用=#2514/#1180、SHO文書#5/#19を既存Ownerに残す。第二Canonical、Graph Resolver、独立MCP Tool、Runner、Host、Source Evidence Ledgerを作らない。

**新しく追加した三入口Goテストは本節執筆時点でexact-HEAD CI未確認**。直前HEADのHealth成功を、この新コードがPASSした証拠へ自動移送しない。Studio/Wails実操作、実Postgres通信、SHO Asset/Reader/Writer、独立ExpectedのOwner承認と業務Effectは今回NOT RUN。

## 25. 実RelationのParentとTargetを同じCanonical Worldへ結ぶ（11/08後半の成熟差分）

### 25.1 このPRで実装する最小の有向Edge

[real_workspace_admission_test.go](../../reference/diagram-crossref/real_workspace_admission_test.go)に限定的な`TestRealWorkspaceRelationParentAndTargetMatchCatalog`を追加。対象は**実`reference/workspace`の`semantic-relation:sales.order.lines`一件だけ**。従来のM1a「Catalog・Inspect・Traceが同じSubjectとSourceを指す」に加え、以下の**型付き意味構造**を確認する。

1. `CatalogWorkspace`にRelationの`Kind+ID`が一意に現れ、`Parent.Kind=semantic-data`。
2. `InspectSubject(relation)`のCanonical `ID`と`Target`がCatalogの同一Relation `ID`／`Detail`と一致する。空のTargetを成功としない。
3. `InspectSubject(parentData)`のCanonical Data IDがCatalog Parent IDと一致し、`Relations[]`内に当該Relation IDが**ちょうど1件**現れる。
4. ReaderのWorld/Source Index fingerprintが矛盾しない。**個別Reader呼出し間の原子的Snapshotを証明するものではない**。

これは`Data→Relation→Target Data`という最初のナビゲーション可能なChainのうち、**Data→RelationとRelationの宣言されたTarget ID**を既存Readerから照合する試験である。Target Dataの実存在／Source Ref／必要な外部Provider依存の全取得、MultiplicityやKey/FDの意味受理まではここでは確認しない。Targetが文字列として宣言されただけでGraphの`resolved`や可逆編集を資格化してはいけない。

### 25.2 標準Model Mapと三視点図面の結合を最短にする

[Draft #3518](https://github.com/hat47x/tei-platform/pull/3518)の`MapWorld`候補は、`contains(Data→Relation)`と`refers-to(Relation→Data Target)`という型付きLinkを、**同じWorld＋Catalog**から導出する。このLinkを#3492側で第二の`ModelMap`実装としてコピーしない。**#3518がmainへ入り正式Go APIのSourceとCIが確認されてから**、上記M1aの実Relationとの`From/To/Resolution/Source`の同値性を1つのHeadless Contractとして評価する。

この順で得るものは「同じSourceに属するRelationのParent／Target宣言を型付きMapと図面の双方が辿れる」という小さな**意味保存の実証経路**であり、三視点全体・CRUD 4×9・Business Graph・Event/State/Function CRUD・実Query/Providerについての合格ではない。

Graph上のEdge表示・2 Canvasでの共有・SHO図面引用は、Relationの安定Identity／Source位置とViewer Disclosureを保持する必要がある。旧`modelref.Edge`に同端点・同Kindの二辺がある場合、ID復元不能なので編集はread-only。Sourceの意味と物理FKの観測候補、Plugin selected/actual Invoked、生成Targetや実Effectを別評価する。

### 25.3 他チャットの縦断証拠とRepository Healthの正しい境界

- **旧HEAD `aa96be49`**の[Health #42496378478](https://github.com/hat47x/tei-platform/actions/runs/42496378478)はcompleted/**failure**。Log上は`go vet`／`go test ./...`（追加した3 Reader整合試験含む）／`tei validate`／ADR indexがPASSで、`real_workspace_admission_test.go`の**空行1行の`gofmt`不一致**が唯一確認された失敗。後続コミットで当該空行を削除。旧HEADの全体Healthを成功扱いしない。
- [SHO Composer G2-W1/G2-P](https://github.com/hat47x/sho-composer)のSource Reader／Byte Patch／Working Copyと、TEI図面のSource逆編集は共通の**Source同一性・現行Revision・プレビュー／受理／再読込**の条件を持つ。ただしSHO文書Patchの成功はTEI Relation Owner受理・図面エディタ成功と同義ではない。双方向保存は#1124とSHO #5/#19で別実証。
- [main統合 #3479](https://github.com/hat47x/tei-platform/pull/3479)の外部Provider raw-only改訂では、Consumer World/Source Index指紋が不変でも外部原文が変わる。図表の現在参照資格には`external source/manifest`の捕捉・版を追加して審査する。実same-Load Witnessは[Draft #3524](https://github.com/hat47x/tei-platform/pull/3524)の別Owner。
- [Draft #3500](https://github.com/hat47x/tei-platform/pull/3500)の実Native Query→既存Executor→instrumented Connectorと、TEI Function Data Context→Provider→実Rowset/Result Slice→Consumerの同一Function/Attemptの確認はまだ別レーン。**CRUDの宣言されたR ≠ Providerが実際に読んだ**。
- [Draft #3540](https://github.com/hat47x/tei-platform/pull/3540)が通常CLIとStudioから選択Inspectorを実Callする`S2-reference`を担当。ここで利用した`operations.InspectSubject`のBuiltin読取を、選択Pluginの実Invoke資格へ誤昇格させない。

**この節の差分はGoテスト追加・既存Plan更新に限定**。新しいGraph Engine、Source Registry、正式Schema、第二API、別Evidence Storeは作らない。最新HEADの`gofmt/go test`とProduct Projection/Studio/SHOは最新Actionsの結果に従い、未確認なら未確認と記す。

## 26. F→M→S→P→C→E の同一Attempt検収とSHO文書編集境界（2026-11-09横断差分）

### 26.1 三視点の「図示できる」と実Applicationの「動く」を一列に束ねない

本書§22〜25のM1a Goテストは、限定された二つの実SubjectからSourceを読めること、`Catalog/Inspect/Trace`が一意の対象・Referenceを指すこと、Data→Relationの親子・宣言Targetを辿れることを確認する。これはP0-M（Source/Meaningとその表示）の**一部**であって、P0-R（Provider実Read）／P0-C（Result SliceのFunction・Modelへの同一性接合）を完了していない。

横断F→M→S→P→C→Eを次の**1つの検収パケット**として整理し、各欄は別の資格とする。新台帳／正式Schemaの追加を意味せず、既存Read-only Context Query、Scenario/Independent Expected、Receiptを結合するための観測項目である。

| 順 | 証拠欄 | 三視点/CRUDの最小検収 | 成功と取り違えないもの |
| --- | --- | --- | --- |
| F | Function/Business Action・PatternのQualified ID、宣言操作・引数とActor/Intent | CRUD Cellの`C/R/U/D`が**どのFunctionの宣言**に由来するか追える。未列挙は`unknown` | Cellの`R`記載だけで実データ取得 |
| M | Data/Relation/Key、Pattern→Query/Function依存、Role/Scope/単位/NULL | 実Data・RelationのSource ID、起点・終点、同型Edgeの別Identityを保持し、M0の型付きLinkと整合 | Graph形状・件数だけで完全閉包や業務Meaning受理 |
| S | Workspace/SourceRef/Manifest/外部Provider原文Revision、取得資格・必要集合 | F/Mの各Sourceと依存が同じ要求時点で取得され、fresh/partial/unavailableを区別 | Consumer World/Source IndexのFPだけでProvider raw/current Grantを証明 |
| P | Query/SQL byte、選択/解決/InvokeしたProviderのEntry/Scope/Rowset、物理→意味Mapping | Functionが指定した実Providerと実行時Inputを確認。外部Row列順から意味属性を推測しない | 合成ConnectorへのInvokeで実PostgreSQL完了や当時権限を宣言 |
| C | CLI/MCP/Studio/SHOなどの実Consumer受領、Disclosure、Page/Continuation/End | Readerが渡した部分集合・Unknownと、表示/文書として受け取った内容の同一性を確認 | Response 0 exitや描画成功で外部受領・全世界完結 |
| E | 独立Expected、Source/Build/実Attempt、Effect/Receipt、Owner受理の判定 | Negative（同名別ID、同Endpoint別Relation、Partial、Source drift、未受領）に判別力がある | Fixture自己整合・見た目の一致から業務Effect/Owner承認 |

**join key**は少なくとも`Workspace identity + Qualified Subject/Relation + Request Scope + Observation/Revision + Attempt`。各証拠の取得時刻・由来・受領対象が違えば「同じFunctionの完了」として統合しない。`worldFingerprint`一致と`Source occurrence`一致の両方が必要な部分もあり、外部ProviderのSource-only変更には別のWitnessが必要（#3479/#3524）。F/M段階の**論理Data SchemaのRelation**とP段階の**物理DB JOIN/FK**も別の表現型であり、自動Meaning昇格させない。

### 26.2 この縦断での次の実装一歩は「同一実RelationのM0 Link」

[Draft #3518](https://github.com/hat47x/tei-platform/pull/3518)の`operations.MapWorld/MapWorkspace`は、同じ`World`／`Source Index`から`contains(parent Data→Relation)`と`refers-to(Relation→target Data)`を導出する**型付きread-only M0**の担当。現在の#3492では`sales.order.lines`の`Catalog Parent`、Relationの`Canonical Target`、Parent Dataの`relations[]`を直接検査することで**M0と比較する対象を明示**した。#3518はDraft・main未統合であり、このPRが同名の`MapWorld`や専用Graph Engineを実装するのは非効率。統合後に同じWorkspace/RevisionでFrom/To/Resolutionの一致と、不在Targetの`unresolved`を比較するのが最短。

[PR #3518の旧HEAD Health #42475774669](https://github.com/hat47x/tei-platform/actions/runs/42475774669)はGo Test全体FAILだが、ログ上の原因は`repository-plan-check`の既存設計索引欠落。Go Map Test本体がFAILしたという証拠ではない。しかし**全体Health PASSを示すものでもない**。mainの設計READMEは別Ownerで重複排除済み（#3542）。#3518の実Source・現main追随とactual APIの形を再確認してから結ぶ。

### 26.3 SHOのReader・Working Copy・Save As・Editor座標を四つの資格へ分離

SHO Composerの並行Draftは、TEIのSource-safe逆編集に直接役立つ負例を提供した。いずれも**別アプリケーション・別文書Sourceの限定検査**でありTEI図面や正式Editorの完成ではない。

| SHOの限定成果 | 三視点View/図面Assetで再利用する不変条件 | 未実証の領域 |
| --- | --- | --- |
| [#27 K1R](https://github.com/hat47x/sho-composer/pull/27)：公開TEIの固定4 Markdownをprivate SHO実G1b Readerで読む | Source identity、parser version、Source Byte/Link Coverage/Consumer受領は読み取り側で別々に記録 | 正式OKF K2・意味受理・SHO文書生成 |
| [#25 G2-W1](https://github.com/hat47x/sho-composer/pull/25)：Working Copy作成／別Open／A→B→A負例 | Fileの`dev/ino`とbyte同一性は異なる。A→B→Aを同SHAから復元できない場合、Source Revisionは`unknown` | 本番保存、Source CAS/Fence、クラッシュ復旧 |
| [#28 Save As](https://github.com/hat47x/sho-composer/pull/28)：相対参照先変化のpreflight | 図面の保存場所/文書内埋込み先を変える場合、相対Asset参照はraw文面同一でも再資格化対象 | 実File/Resource解決、Writer、公開 |
| [#29 E0-MAP](https://github.com/hat47x/sho-composer/pull/29)：UTF-16 Editor Selection→UTF-8 byte span | View Occurrence選択とSource上のbyte/character positionを分け、BOM/EOL/dirty buffer/Normalizationにより不一致ならEdit拒否 | 実Editor同期・実Source保存・Owner受理 |

三視点の**Semantic Relation ID→元Source byte span**とSHOの**Document Occurrence→Document byte span**は似た安全要件を持つが、対応するOwner/Scopeは別。TEIのModelerで図面中のRelationを選択し、SHO内で同じ図を引用し、文書をSave Asしても、TEI Canonicalを文書rawから再生成しない。Source再取得／Presentation Layout変更／意味Patchの受理／Document保存後再Openは別の業務として扱う。文書の版更新で元TEI Sourceが変更されたと推論しない。

### 26.4 実装・CIの現在地

[PR #3492 HEAD `9369842cdaaa0eb4cf08d4da7f25012e71bd3680`](https://github.com/hat47x/tei-platform/pull/3492)の[Repository Health #42504682718](https://github.com/hat47x/tei-platform/actions/runs/42504682718)は**completed/failure**。ログでは`go test ./...`／`go vet`／`tei validate`／ADR IndexがPASS、`real_workspace_admission_test.go`の`gofmt`のみFAIL。これはGo実Source/Parent/Target局所試験がPASSしている根拠ではあるが、全Health Greenではない。該当する余分な空行を削除した修正コミットは旧Branch上で作成した。**この変更と本設計更新を最新mainへ移植したHEADのCIは別に確認する。**

同じ時期に[Draft #3540](https://github.com/hat47x/tei-platform/pull/3540)は通常CLI/Studioの`S2-reference`配線を前進させ、HEAD `a5cf28acfdb31f2e583416f1f3bc2eddcf97a388`の[Health #42505204119](https://github.com/hat47x/tei-platform/actions/runs/42505204119)がcompleted/success。一方#3524のSource/Manifest same-Load WitnessはDraftで、Health successとWails Host smoke failureの別判定が存在する。**局所Provider選択・read-only Inspector・raw Witnessの成功を、一つのProduct Plugin Read/Business Effect/SHO正式受領の成功へ合算しない。**

本Planは既存Issue・RQ-036 Model MapとRQ-037 BOM提案・ADR-0048・各Ownerへの実証要件の接続を担当し、第二Canonical/Graph Schema/Host/Source Ledger/Runnerを導入しない。

## 27. Source-indexed Relationの三点閉包をM1bに昇格させる条件（2026-11-09）

### 27.1 直前の実CIを正しく受理する

直前のPR HEAD `862009d129a89795fe563e95f5fc9538b3721a90` の[Repository Health #42555519613](https://github.com/hat47x/tei-platform/actions/runs/42555519613)と[脆弱性確認 #42555519690](https://github.com/hat47x/tei-platform/actions/runs/42555519690)は、GitHub Actions上で**completed/success**と確認した。対象は**Plan＋実Workspace Readerテストの2変更ファイル**であり、合成Graph／Fixture／`modelref`の既存ファイルはmainへ統合済みで、このPRへの重複追加を避けている。

ただし前HEADの成功は、今回の新しいRelation Target検査を含まない。新HEADの実`go test`／Healthは別に確認し、PR main追随も別判定とする。mainは移行後も進んでおり、**旧ベースのCI合格・mergeableと現mainとの統合証拠は異なる**。分岐履歴を無理に書き換えてCI履歴を消さず、現在のmainで同一Blobの既存Graph/Source Index/Coreを優先する。

### 27.2 親Data→Relation→参照先Dataの閉包は文字列一致だけでは不足

以前の`TestRealWorkspaceRelationParentAndTargetMatchCatalog`は、実`operations.CatalogWorkspace`で`semantic-relation:sales.order.lines`が一意な`semantic-data`のParentを持つこと、Relationの`Canonical.target`が`Catalog.Detail`と一致し、親Dataの`relations[]`に同IDが一度だけ含まれることまで検査した。しかし**参照先のData自身が実World/Source Indexへ入場したかは検証していなかった**。

今回は同じTestに、Parent→Relation→Targetの**三点**だけを対象とする実Source照合を追加した。正式な新スキーマ・Read-only Graph Generatorを設けず、既存`operations.CatalogWorkspace`／`InspectSubject`のみで次を要求する。

| 照合対象 | 新しい最小検査 | 未証明のもの |
| --- | --- | --- |
| RelationのTarget | `Canonical.target`が非空、`Catalog.Detail`と一致 | Target文字列が全Worldで解決済みとはまだいえない |
| Targetの宣言 | `{kind: semantic-data, id: Target}`がCatalogに**ちょうど1件**、`Ambiguous=false`、`Prelude=false`、`SourceStatus=indexed` | Sourceにない外部DB Table、同名別Scopeの暗黙の同一視 |
| TargetのCanonical | 実`InspectSubject`で指定Kind/IDを厳密に再解決、`Canonical.id`一致、Source refが非空 | Readerが同じOS snapshot／OwnerがMeaning受理済み |
| Version/Workspace | Parent/Relation/Targetの`Workspace identity`、`WorldFingerprint`、`SourceIndexFingerprint`が一致 | 外部Provider rawの鮮度、実Provider Rowset、実Binary/Grant |
| Graphへの引継ぎ | 実在した一つのData→Relation→Dataトポロジーの入力として使える | 同端点の別Relation Instance ID、Function CRUD/Business/全Graph閉包 |

**M1b-local**の対象はこの三点に限定し、`Required`の未列挙Relation・Function・State・Pluginまで適格になったとは主張しない。仮に参照先が未宣言・未Index・重複なら、UIは既知Relation IDと`unresolved/ambiguous`を維持し、Targetの存在やSource逆編集を補完しない。将来的に対象Workspaceが外部参照を正式サポートする場合でも、現在のFixtureに合わせた黙示的インポートや別Source正本を設けない。

### 27.3 最新横断成果の取込み順と過剰拡張の抑止

- **M0 Model Map [Draft #3518](https://github.com/hat47x/tei-platform/pull/3518)**：`contains(Data→Relation)`／`refers-to(Relation→Target)`を型付きMapへ投影する別Owner。今回の三点テストはその**実TEI Source側の比較元**であって、M0を重複実装しない。`partial-projection`を全Mapの完全性へ昇格させない。
- **Query/Provider [#2586](https://github.com/hat47x/tei-platform/issues/2586)、[Draft #3500](https://github.com/hat47x/tei-platform/pull/3500)**：最初の実Readは`order_entry.recent_orders`。同一Function/SQL byte+args/Selected Provider/Result Slice/Row→Semantic Mapping/Consumer/Attemptの観測が必要。今回のRelation Target検査はProvider Readを一切呼ばない。三視点の`CRUD R`は実Read成功に読み替えない。
- **Plugin D1 [main #3381](https://github.com/hat47x/tei-platform/pull/3381)、[#3540](https://github.com/hat47x/tei-platform/pull/3540)**：型付きInspection Port・選択Provider Go実Callの限定経路は重要だが、Source三点をBuiltin Inspectorで読めることと、製品CLI・Studioが選択Pluginを実Invokeすることを別票にする。
- **SHO/Document**：図面の関係Sourceを3点解決できても、SHO Document OccurrenceのByte Span/UTF-16 Editor Selection/Save As参照、受信Asset、文書再Openは別Consumerの資格。三視点Modelerの選択をSHO Writerの現在編集資格として使わない。
- **Self-build/Runtime**：生成用Plan・Build Receipt・実Artifact・選択Plugin・実Target Invocationと、現在のRead-only Model Projectionを独立に管理。新たなGraph Engine/Host/Registry/Acceptance Ledgerを追加しない。

**次の最短作業**：上記三点の実Go試験をexact HEADで通す → #3518の型付きMap `contains/refers-to`と同じ3 Subjectで一致試験（main採用後）→ **1本のRelation ID＋Source Refを欠落せずM2 Headless Graphへ出す**。その際、Sourceで同型Relationが2本あることと、`modelref.Edge`のID不足による逆編集不可は独立の制約。Function×Data CRUD MatrixのActual Source/Unknownセルは同Graphの完成と同時に保証されたことにしない。

## 28. Source Locatorの実Byte資格と意味・編集資格の分離（2026-11-09）

### 28.1 既存CLI/ModelerのSource Indexを実FileのByteへつなぐ

§25–27で`operations.InspectSubject`／`CatalogWorkspace`／`TraceSubjectSources`を使用し、実Query `order_entry.recent_orders` と実Relation `sales.order.lines`、さらに親Dataと参照先DataのCanonical ID・Source参照を取得できることを検証した。直前のPR HEAD `3081fb3189209e59a95e7efa50bbe1e380d6d4e3`は、[Repository Health #42604971069](https://github.com/hat47x/tei-platform/actions/runs/42604971069)・[脆弱性確認 #42604971086](https://github.com/hat47x/tei-platform/actions/runs/42604971086)とも**completed/success**。これは実Loader/Readerの限定的なSource入場を裏付けるが、Source Locatorの指す**現在の実File Byteを独立に検査した実証**とは別である。

今回、[同一Goテストファイル](../../reference/diagram-crossref/real_workspace_admission_test.go)へ`TestRealWorkspaceIndexedSourceRangeAndByteFingerprint`を追加。既存の`sourceindex.Index.Validate`、`sourceindex.NormalizeWorkspacePath`、`sourceindex.Range.Validate`と**チェックアウト済みの実Source File Byte**を利用し、QueryとRelationの二つのSubjectについて次を照合する。

| 反証 / 検査 | 保証できる最小条件 | これだけでは保証できないこと |
| --- | --- | --- |
| SL-01 Current bytes | Subjectから見つかったArtifactの相対Pathが正規化済みで、対象Fileの現在byte fingerprintが既存Source Indexと一致する | 別ファイル・外部Provider rawの閉包／Read AttemptのOS上原子的同時点 |
| SL-02 Locator span | 全ReferenceのSubject ID・Artifact Pathが要求と結び、Offset・Line・Columnが現Byteで妥当 | Spanが本当に同じ業務意味を持つ、Ownerの受理 |
| SL-03 Source-only変化 | Reference Rangeが収まる程度の末尾1 byteの追加でも、古いArtifact Fingerprintでは`Index.Validate`を拒否 | 過去のA→B→A履歴、Provider Import済み原文の現在性 |
| SL-04 Span破損 | End Offsetを現在File長外へ変更すると、Fingerprintが同じでも`Range.Validate`を拒否 | Editor上のUTF-16/Unicode位置、dirty buffer、現在のEdit Grant |
| SL-05 Consumer制約 | Source側の`Role/Scope/Artifact`と図上のSemantic Subject/Relation IDを結合し、UI Navigationの可否を別に判定 | SourceへPatch可能、Inspector Plugin実Call、SQL/DB Read、SHO Writer保存 |

このテストは**登録済みの信頼された`reference/workspace`の実Fileのみ**を読み、`NormalizeWorkspacePath`をOS symlink隔離/Host許可の代用としない。Runtimeの権限検査・External Provider rawのsame-load witnessは別Owner（#3524/#3479）を使用する。範囲が現在のByteに一致しても、同型Relationの`Edge ID`を`modelref.Edge`に自動補完しない。またArtifact全体Fingerprintは現在のSource残存性を示すが、意味を受け入れたモデルのRevision/Source移管/Operation Grantとは別である。

### 28.2 他チャットの最新進展との接合順

- **RQ-036／Model Map [Draft #3518](https://github.com/hat47x/tei-platform/pull/3518)**：`MapWorld/MapWorkspace`の型付き`contains/refers-to`はM0 Producer候補。実Source側の`Query/Relation/Parent/Target + SourceRef + Byte Witness`を比較元とし、同じArtifactの別View OccurrenceでもSourceを二重登録しない。M0がmainへ統合される前に第二Model Map APIを作らない。
- **Function Data Context [#2586](https://github.com/hat47x/tei-platform/issues/2586)、[Draft #3500](https://github.com/hat47x/tei-platform/pull/3500)**：FunctionのRequired Data/Relation/粒度/時点/取得条件と、実Query/SQL/Provider/Rowset/Result Sliceを別に束縛する。今回のSL-01〜04は`CRUD R`、現在DBの内容、Provider actual invocationを検収しない。
- **Plugin-first Inspector [main #3540](https://github.com/hat47x/tei-platform/pull/3540)**：通常CLI・Studio Hostで選択した静的Reference Inspectorを呼ぶ経路は統合済み。ただし今回の`operations.InspectSubject`は組込み関数の呼出しであり、選択Plugin/Verified BOM/Principal Grant/外部Provider一致はこのTestから認定しない。
- **SHO Composer**：UTF-16 Editor Selection→UTF-8 Byte Span、Working Copy/Save As/Relative Link、読取りAssetの現在Sourceの検収を、TEI図上選択とは独立のDocument Scopeで扱う。Source変更がなければ文書作業のRevisionを暗黙更新しない。TEIが利用するReader/Byte Spanの不変条件を共有してもWriter/Ownerを統合しない。
- **最新mainの実証管理**：以前の成功したRepository Healthと、この新Testを含むコミットのHealth/PR mergeabilityを混同しない。mainの共通Plan/Source Reader/CLI/MCPの変更をこの小PRへ再実装しない。新規のGo testが実行されたHEADの結果を確認するまではSL-01〜04は**テストコードを追加した段階**にとどめる。

### 28.3 次のM2で必ず保持する情報

`P0-M`の最初のHeadless Diagramは、同じSourceで解決された`semantic-data(Parent)`→`semantic-relation(ID)`→`semantic-data(Target)`を、**Qualified Identity、Role、Source artifact fingerprint、Reference Range、World/Index Revision、Required Scope/Partial性**とともに追跡できることを優先する。別Canvasへ複数配置してもSubject/Relation IDの正本は一つ、描画Occurrenceは複数とする。`modelref.Edge`で固有Relation IDの復元ができない場合はread-only Navigationを許容し、図上Selectionからの逆Patchを不許可とする。`CRUD`はFunction宣言の既知セルを出典へ結び、未取得セルをUnknownのまま維持する。

新しいAccepted Ledger／Graph Schema／Source Index／Host／MCPだけのDiagram Interpreter／Runtime Engineを追加しない。既存#1124/#2497/#3262/#2514のOwnerへ最小Source-correct Consumerの検収を引き渡す。

## 29. Source IndexのByte妥当性とSemantic Bindingの認証を区別（2026-11-09）

### 29.1 Sourceを指せることは、そのSourceが意味要素を定義する証明ではない

§28は管理対象の実Byte/Range/Fingerprintを検査した。次の独立した負例で、**正しいByte範囲を指すReferenceの`Subject`欄だけを別の未宣言Relation IDへ書き換えても、`sourceindex.Index.Validate`はSourceが存在して範囲・Fingerprintが正しければ成功し得る**ことを固定する。現在のIndex APIはSource Index構造のValidationであり、Canonical Worldへの意味上の束縛の受入APIではない。

新設した`TestSourceIndexStructuralValidityDoesNotAuthenticateSemanticBinding`では、実`reference/workspace`の`semantic-relation:sales.order.lines`の`Artifacts`と`References`を取得し、**テスト内のIndexだけ**の1 Reference Subjectを存在しない`sales.order.nonexistent_target_for_negative_control`へ差し替える。実Source File Byteには書き込まない。

| 検査 | 期待結果 | 誤った上位推論 |
| --- | --- | --- |
| 実`Index.Validate(read)` | **PASS**：Byte/Fingerprint/Positionは整合するため | Reference Subjectの意味まで保証済み |
| `TraceSubjectSourcesFromIndex(forgedIndex, forgedID)` | **Traceは生成できる**：入力IndexのSubject情報に従う | Traced IDがCanonical宣言として存在する |
| 実`operations.InspectSubject(root, forgedID)` | **`SUBJECT_NOT_FOUND`**：真のWorkspace Canonical Worldには存在しない | Source Indexの偽装Referenceから自動的にEntity/Relationを採用 |
| 三視点/CRUD/Editor | **Relation→Canonical/Source Locatorの突合とMeaning/Grant確認前はWrite不可** | 図面上のNodeやCRUD Cellが元Sourceを安全にPatch可能 |

ここでの**Validation PASSは意図された負例の一部**であり、TEIが偽造Subjectを正式受理する設計判断ではない。検査はSyntheticなIndexの改変、実Workspaceでの`InspectSubject`拒否という対照であり、外部Sourceの攻撃耐性、実UI編集拒否、Owner承認、独立Oracleを実証しない。既存Index全体を「無効」扱いする必要もなく、正しい用途（Byteの根拠・参照の構造）に限定する。

### 29.2 三視点Graph/CRUDの受入に必要な連鎖を小さく固定

- **Source観測**：同一WorkspaceでSource IndexのArtifact Byte/Range/Roleを取得、どのSourceを実Loaderが読んだかとIndexのRoleを区別。
- **Canonical突合**：`(Kind, ID)`の完全一致で`Catalog`と`InspectSubject`を解決。Relationについては親Data・Relation ID・Target Dataの3点をSource参照付きで検査する。同型2 Edgeの場合、関係Typeだけで個別Relation IDを代用しない。
- **受理済み意味・閉包**：上記2つが成立しても`Required`集合、Source観測Revision、業務Role・粒度・NULL、OwnerのMeaning Acceptanceは独立。未入場は`partial/unresolved/unknown`として残す。
- **機能と実体**：Functionの宣言CRUD、Query/Providerの選択・実Read・Result Slice/Consumer、Pluginの選択と実Invoke、Host現在Grant/Effect/Receiptはそれぞれ独立の証拠。CLI/MCP/Studio/SHOの反映・受領も別。
- **編集**：図面Occurrenceの選択・Sourceの有効なbyte span・Canonical ID/Relation Instance ID・現在のRevision/GrantとSource編集中の変更を照合し、1件に確定できないPatchを拒否する。Source Indexの`Validate`単独をEditor書戻し資格に使用しない。

この区別は、SHOでいう「Document Byte Spanが正しい」と「元段落を適切に編集できる」の分離、KON/SUIでいう「Graph/Cardが表示されている」と「対象Entityが業務上受理されている」の分離と対応する。ただしTEI/SHO/KONの正式データ構造やOwnerを統合しない。

**実装優先順**は変えない。#3518 M0がmainへ統合され型付きGraphの同一Source・同一ID照合が可能になった後、三視点の**read-only Edge→Canonical Subject→現在のSource Reference**の小経路をProduct Operationの候補として比較する。今回の差分は[実Workspace限定Goテスト](../../reference/diagram-crossref/real_workspace_admission_test.go)と本Planのみ。新Core Graph/Validator、第二Resolver/Registry/Host/Runner/Owner Ledgerは作らない。

### 29.3 CI・前提への限定

前HEAD `536a8364a873610997ad8749f74aa463ddcdfdc1` の[Health #42609326764](https://github.com/hat47x/tei-platform/actions/runs/42609326764)はこの更新時点**in_progress**、[脆弱性確認 #42609326691](https://github.com/hat47x/tei-platform/actions/runs/42609326691)は**success**である。新しいSubject偽装負例と本節を含むHEADのGo試験・Healthを、旧HEADの状態から推定しない。現在main進行中の[RQ-036 Model Map Draft #3518](https://github.com/hat47x/tei-platform/pull/3518)、[組織責任切替 Draft #3741](https://github.com/hat47x/tei-platform/pull/3741)、[Org 実DB検証 Draft #3500](https://github.com/hat47x/tei-platform/pull/3500)の局所試験・Meaningと合算しない。

## 30. 型付きModel Map M0の実Workspace対照と三視点・実行証拠の切断面（2026-11-09）

### 30.1 RQ-036の実装主体を現在mainへ統合する

[Draft #3518](https://github.com/hat47x/tei-platform/pull/3518)の`internal/operations/model_map.go`／`model_map_test.go`（前HEAD `40420ea70ec1ed518ddbd294bee7356dcfb5c8a5`、2件の旧Source Blob）を、現在のmain `7a37d3558cf28d8890af6fc8289ef18b7c10ac50`を第一親とする**非強制二親合成**で同じPRへ前方移植した。旧コードに第二のGraph Store/Role Registry/Provider/Hostはなく、`MapWorkspace(root)`が一回の`workspace.Load`と`BuildSourceIndex`から既存`CatalogWorld`へつなぐ、**read-only／`partial-projection`のM0**を提供する。旧Draftの本PR以外への重複統合は行わない。

M0の`ModelMapLink`は`From(Kind,ID)`／`To(Kind,ID)`／`Relation`／`Resolution`を保持する。`Resolution=resolved`は**Catalog中に両Endpointが一意に宣言された**意味だけであり、Sourceの現在Byte資格、Owner Meaning Acceptance、Role/Scope Disclosure、Host選択・実呼出し・業務Effectの証明ではない。Source Indexに存在するReferenceもParserで採用されたSubjectかを別に検証する（§29）。Data/Function/Business三視点とそのCross-referenceは、**このM0にSource-backedな表示／操作の接続を追加するConsumer**として扱い、M0を第二のDomain Canonicalへ昇格させない。

### 30.2 本物の同じRelationをM0とSource Readerの双方から照合

[実Workspace Go Test](../../reference/diagram-crossref/real_workspace_admission_test.go)に`TestRealWorkspaceModelMapMatchesSourceIndexedRelationTriplet`を追加し、実`reference/workspace`にある一つの`semantic-relation:sales.order.lines`のParent DataとTarget Dataを、実`InspectSubject`から取得する。これを同一Workspace上の`MapWorkspace`から生成された次の**2本の型付きLink**と比較する。

| 実Source側の宣言 | M0から要求するLink | 断定できること／できないこと |
| --- | --- | --- |
| CatalogのRelation Parentが`semantic-data`で一意 | `Parent Data --contains--> Relation`、`resolved`、1本 | Parent/Relationの宣言一意性。実Owner関係・更新順序・トランザクション境界ではない |
| Relationの`Canonical.target`で指定したDataが実SourceでIndexed | `Relation --refers-to--> Target Data`、`resolved`、1本 | 1件の型付き参照。全Relation閉包・FK・繰返し関係のGraph Instance IDではない |
| 3 Subjectの各`InspectSubject` | 個別Source Referenceが非空、同Workspace/World/Source Index fingerprintとKind/IDが一致 | 一定範囲の観測整合性。原子的multi-read Snapshot・外部Provider raw/current Grantではない |
| `MapWorkspace`のCoverage | `partial-projection`で未資格Rolesを明示 | Event/State/Business/Plugin/Function CRUD/Result Sliceが完成したという推論を拒否 |

このテストの一意なRelationは、実Source三点（§27）、Byte/Range（§28）、Canonical偽装の負例（§29）に続く**新しいProduct M0/Reader照合の足場**。しかし、Source Browserが安全に位置を表示できること、M0が一意なLinkを描けること、図面で同型Edge Instanceを逆編集できることは別。引き続き`modelref.Edge`にRelation固有IDがない経路のWriteは`ambiguous/read-only`とする。

### 30.3 他チャット最新の境界：観測・認知・実呼出しを分けて表示

- **SUI/KONの複数Island/Canvas・Ontology**：M0で表示される型付き`Data→Relation→Data`の意味参照を一つに保ち、View Occurrence/局所接続／権限で見えない対象は別の表示状態。現Viewの対象集合から**存在しないこと**を推論しない。プライベートRelationやFieldの情報を現在Disclosure資格なしに返さない。Ontology DSL用の第二Graph正本を作らない。
- **組織・REA**：[Draft #3741](https://github.com/hat47x/tei-platform/pull/3741)のRecord Scope/責務Cutover、[REA Draft #3496](https://github.com/hat47x/tei-platform/pull/3496)の外部Observation→Structural Candidate→Accepted Meaningは、M0の`resolved`とは独立。人／組織の関係を図示できても、異動後の現在Grant・遡及訂正・Handoff・外部Observationの受理までは示さない。
- **Plugin Provider**：[main #3757](https://github.com/hat47x/tei-platform/pull/3757)はEnvironmentの複数宣言について**解決到達と実行到達を別々に観測**。`ObservationProviders/EvidenceProviders`がBindingで解決できても、非テスト経路のRouter Observe/Collectは未接続と記録。Graph上のPlugin Linkは`declared/selected/resolved/loaded/invoked/observed/consumer-received`の区別を残し、選択Providerの宣言を実Invokeの証明に転用しない。
- **実データ読取**：標準M0のPattern→QueryやRelation解決を、`order_entry.recent_orders`の実Provider Read・Result Slice／Row/Field Mappingに昇格させない。P0-Rは#2586、組織Pressureは[Draft #3500](https://github.com/hat47x/tei-platform/pull/3500)、P0-Cの同一Attempt接続は既存Ownerへ残す。
- **Lifecycle／SHO**：Graph/Hookの意味上の変更（[main #3761](https://github.com/hat47x/tei-platform/pull/3761)の宣言外Outcome拒否等）とLayout/Markdown上の引用は異なる。SHO Document Writerで図を保存することはTEI Semantic Patch・実Runtime Hook・Host Grant・Effect Receiptの承認を意味しない。Editor位置とSource Rangeが現在Byteに照合できることもMeaning受理ではない。

### 30.4 次に製品として進める最短の受入経路

**M0 read-only Mapの実Source一致 → Source-backed Relation ID/Role/Scope／Coverageを付けた最小Headless Diagram Consumer → Function Data Context由来CRUDの既知セルだけをSource逆引き → 実Query/Provider/Result Slice → Studio/SHOの実受領**を順序として保持。全工程を一つの巨大なGraph Editor／業務Runtimeにまとめず、各々既存Operation・Generator/Consumer・Authority/ScenarioのOwnerへ渡す。

今回の`MapWorld/MapWorkspace`はまだ**Draft上のGo実装**であり、前方移植後の全Go/Health、OS横断、正式M0 API、Event/State/Hookの完全集合、GUI、SHO、業務Effect、Owner意味受理は**CIと別検収で判断**。前のPR HEADや別Productの成功を現HEADの成功に転記しない。

### 30.5 M0に対する2つの誤認防止反例（実Go候補）

`internal/operations/model_map_test.go`に次の独立した合成負例を追加する。どちらもM0 Producerの**読み取り投影**を対象とし、Source Indexの`Index.Validate`やCurrent Grantを再実装しない。

1. **M0-N01：宣言上のResolved ≠ Source Indexed** — Data、Relation、Target Dataの宣言が一意であれば`ModelMapLink.Resolution=resolved`になるが、入力Source Indexが空なら各`Catalog.Entry.SourceStatus=not-indexed`のまま。`resolved`をSource-current／Owner-accepted／編集可能に読み替えない。
2. **M0-N02：偽Source Index ≠ Canonical Relation** — 実IRに存在しないRelation IDをSource IndexのReferenceだけに足しても、`Catalog`へ新Subjectは生まれず、`MapWorld`にそのSubjectを端点とするLinkも生まれない。旧§29のByte-valid Reference偽装をM0側でも第二正本へ昇格させない。

二つの局所Goテスト（`TestMapWorldDeclaredLinkIsNotSourceQualified`、`TestMapWorldDoesNotInventRelationFromForgedSourceReference`）は、§30.2の**実Native Source→M0 Link比較**と別の検収である。実Source・OS敵対的Path／漏洩、外部Provider、Source/Meaning Owner受理、Event/State/Function CRUD、実Consumer/Editor/SHO/Effectは未証明。最新HEADのCI終了前は「実装済み負例候補」として扱う。

### 30.6 多重Relation Identityと同ID衝突の局所Graph負例（#3866）

既存`model_map_test.go`へ次の2つの**合成IR負例**を追加した。旧#3518が未マージでCloseされたため、コード検収Ownerは**新Draft #3866**に集約する。`modelref.Edge`の型/Endpointだけに縮約した場合の逆編集損失を、M0の`Relation Subject Kind/ID`を保持したread-onlyリンクで回避できるか確認する。

| Case | 要求する型付きM0結果 | 依然として未実証 |
| --- | --- | --- |
| M0-N03：同じFrom/To、同じラベル`lines`、異なる2つのRelation ID | `Parent Data --contains--> Relation --refers-to--> Target Data`を各IDについて保持し、**計4 Link**を失わない。IDをラベル・端点・配列位置で代用しない。 | Synthetic Source Indexは空で、`SourceStatus=not-indexed`。どちらのEdgeもSource逆Patch資格なし。 |
| M0-N04：二つの親Dataが**同じRelation ID**を重複宣言 | 2つのParent→Relation Linkと1つのRelation→Target Linkを保持し、重複Identityの`Catalog.Ambiguous`／`Link.Resolution=ambiguous`を表明。First-winsしない。 | 重複IDを正当化・自動改名・Owner受理したわけではない。 |

これは**宣言した関係のナビゲーション構造**の反証であり、Data/Function/Business図面の見た目・CRUD自動生成、実Source位置の最新性、現在Grantや実行を試していない。#2497のLoss/Reverse編集と#2514の独立Expectedへ分離して引き継ぐ。二つのGoテストの実行・CI成功は、**新HEADのログで確認するまで未資格**。

### 30.7 Model locate・Studio選択・AI Harnessは既存Ownerへ接続

[PR #3865](https://github.com/hat47x/tei-platform/pull/3865)の`model locate`候補はCatalogに基づく親経路・近傍（現在地G1/G3）をCLI/MCPへ返す**別のread-only Consumer**。新Draft #3866のM0型付きLinkと同じ`Subject Kind/ID`・Workspace/Source版で比較するが、LocateのGraphをM0へ複製せず、`model locate`の現在地表示成功を全Data/Function/Business Closureや編集資格に読み替えない。#3865は現時点ではPRであり、main統合と全E2Eは未資格。

StudioでUI要素を選択しAI/DSH系Harnessへ細かく指示する設計は、**main統合済み[PR #3863](https://github.com/hat47x/tei-platform/pull/3863)** のAI性能計画§21がOwner。ここでは、M0型付きSubjectとView Occurrenceを分けた選択Contextから許容されたread-only Sourceへ辿り、AIのProposalと現在Grant付きApplyを分離する、という**既存検収条件への入力**のみを定義。第二Harness Loop/Gateway/Adapterを本PRに作らず、特定の定額AI契約を製品の基本動作の前提としない。

## 31. M0実装Ownerの現行PRと合格条件（2026-11-09）

**現行のコード検収Ownerは[Draft PR #3866](https://github.com/hat47x/tei-platform/pull/3866)。** 本書§23〜30で歴史的に参照している旧[#3518](https://github.com/hat47x/tei-platform/pull/3518)は**close/unmerged**。理由は現在mainへ二親統合してもGitHub PR Files上では975ファイルの旧差分が表示され、レビュー可能な変更集合と一致しなかったため。#3866では現main単一親に4つのGit Blobのみを加え、GitHubのPRファイル一覧が**厳密に4件**（`model_map.go`、`model_map_test.go`、本Plan、`real_workspace_admission_test.go`）であることを確認した。旧M0正本は削除せず、コード/Scenario/Planの内容を引き継ぐ。**この置換はmainへmergeした意味ではない**。

初期M0の採用判定は、新PRの**exact-head**に対して`gofmt`、`go test`、実`reference/workspace`のReader→型付きM0 Linkの2本一致、M0-N01/N02の拒否対照、Repository Healthの結果を照合し、Ownerの意味受理と製品全体受入を別にする。旧Branchのテスト成功やSHA一致は再検証の代替ではない。#3492のRead-only Source入場はmain統合済みだが、標準M0/三視点Graph・CRUDの**本番表示・編集・実Provider read・Studio/SHO実受領**の受入はまだ行っていない。

実装順：M0 Producerを最小read-onlyに維持 → 同じSourceと型付きLinkを使うData三点のHeadless Diagram Consumer → Function×Data CRUD Source Cell → 実Provider Result Slice → Studio/SHOの別Consumer。**見えている集合、取得済み集合、公開許可された集合、Ownerが受理した集合、実行した集合は互いに同一視しない**。SUIのCanvas/島やKON DataspaceのOntology表示も、この情報区分を踏まえて設計し、第二Source/Graph/Authorityを作らない。

## 32. Source-only RevisionとH0/H1/H2の次段検収（2026-11-09横断更新）

### 32.1 M0のTopologyは変更されなくてもSourceの現在資格は変わる

新Go局所テスト `TestMapWorldSourceOnlyRevisionMustNotBeConfusedWithSemanticTopology` は、**同じ宣言World**に対しSource Indexの参照元Pathの移動、Artifact Byteの変更、A→B→Aの復帰を与える。既存`sourceindex.NewArtifact`／`workspaceProjectionFingerprints`／`CatalogWorld`／`MapWorld`を使用し、新Cache/Revision Engineを追加しない。

| Case | M0の観測 | 引き継ぐべき判定 |
| --- | --- | --- |
| M0-N05a Path Move | Link topology、World Fingerprintは同じ。Source Index Fingerprint／Model Revisionは変化 | Display Nodeに変化がなくてもSource Locator・Source-origin Evidence/編集資格の再検証が必要 |
| M0-N05b Raw Byte Change | 同じPath・同じWorldでもArtifact fingerprintが変わり、Source Index Fingerprint／Model Revisionが変化 | World同一性や描画Cacheの一致を現在のSource Span・Host Grantの証明に転用しない |
| M0-N05c A→B→A | Aと同じ現在Byte/Pathへ戻れば、現在のGraph/Index/Model RevisionはAに一致し得る | **中間Bがなかったことは証明できない**。履歴・実Effect・Owner受理は別Evidenceが必要 |

これらは`Source Index`の**合成Artifactと宣言World**を使う限定試験であり、OS実Fileの移動・中間BのAudit保存、同一Load原子性、Source Write/CAS、実Provider Importの`same-load`証拠を試験しない。以前の§28・§29の**実Workspace Byte/偽Subject負例**と併せても、全部のRequired Source closure/Meaning Acceptanceへ自動昇格させない。

### 32.2 横断計画の優先順位を製品の利用価値へ戻す

[横断受入Draft #3535](https://github.com/hat47x/tei-platform/pull/3535)と、[Studio AI性能計画main #3863](https://github.com/hat47x/tei-platform/pull/3863)の成熟した順序を、このGraph/Modelerレーンで参照する。全体受入をM0/Diagramの完成待ちにしない。

| 既存OwnerのP0 | M0/三視点側の役割 | 未実証のまま残すもの |
| --- | --- | --- |
| **D1/S2-product** #1251: 正式Profileで選択されたProviderが通常CLI/Studio経路で実際に呼ばれる | Map上は`declared/selected/resolved`をナビできるようにする。実Plugin CallはExecution/Host側から観測 | 同一Go関数・バイナリの一致だけでProvider実差替え／現在Principal/Grant／BOMを認定しない |
| **R0実Read** #2586: Function→Query→実DB Provider→SQL/Rowset→明示Mapping→Data Context→Consumer Receiptを同一Attemptで取得 | Function→Data/RelationとQueryのSource定位、`CRUD R`宣言とObserved Readの別列 | [隔離SQL Fixture Draft #3862](https://github.com/hat47x/tei-platform/pull/3862)の3/2/0 Expectedはまだ実DBクエリ実行ではない |
| **G1/G3現在地** [Draft #3865](https://github.com/hat47x/tei-platform/pull/3865) | M0 `Subject Kind/ID` とCatalog Parent/Link・World/Index Revisionを共有。Graphが不完全でも親経路と近傍の欠落理由を提示 | CLI/MCP locateが存在してもG2/G4〜G7全体図・往復ナビ・Studio/E2Eは未資格 |
| **AI選択** main #3863 | 選択UI Occurrence→Qualified Subject/Source/Current Disclosureの**必要なフィールド**を提示。H0決定論的検証／H1外部Harness／H2モデルProvider切替を同一Task/Expectedで比較する | 外部DSHへの送出・実OS隔離・費用/利用規約・Owner受理・実変更はNOT RUN |

**停止線**：read-only Map/Locateは日常利用者に役立つ最初の全体案内であって、正式Provider・実Readを妨げるP0新Gateではない。ユーザーがUIで細かく選んだ要素にAIが応答する際、選択文字列だけからSource/Grantを推測しない。既存Operationが返す範囲だけを明示的に渡し、AI Proposal→Source Revision再検査→共有Acceptance→Authorized Apply→Receipt/Consumer受領を**別観測**する。定額生成AI契約による自動API利用許可・企業内の情報持出し可能性を推定しない。

### 32.3 今回の変更の到達点

**実コード変更**はM0のSource-only Revisionに対する局所Go試験1件と本設計追補。新CLI/MCPコマンド・正式Studio Renderer・SHO Writer・Graph Patch Adapter・Cache Engine・専用Harness/Hostは追加しない。最小のread-only全体案内から、実Function→Provider→Data Contextの証拠と照合できる状態へ段階化する。旧#3518はclosed/unmerged、採用検討対象は現在の[Draft #3866](https://github.com/hat47x/tei-platform/pull/3866)のみである。

**CI判断**はこの新HEADの実Runで行う。`ModelRevision`・`World Fingerprint`・`Source Index Fingerprint`の用途を記述したことは、Productの全キャッシュ失効やHistory Auditの実装達成を意味しない。

## 33. 選択した実RelationのModel Map／model locate同一性とHarnessへの限定Context（2026-11-10）

### 33.1 RQ-036の二つのread-only Consumerを同一対象へ束縛する

[main統合 #3865](https://github.com/hat47x/tei-platform/pull/3865)で`tei model locate`およびMCP `tei_model_locate`が**実装済み**。モデル内の位置／親経路／兄弟／近傍参照を型付きで返し、`limit`で省略された件数や未導出の関係種別を明示する。一方、[本Draft #3866](https://github.com/hat47x/tei-platform/pull/3866)の`MapWorkspace/MapWorld`はData・Relation・Pattern等の**型付き関係のread-only M0 Projection**を提供する。双方とも既存`World/Catalog/SourceIndex`に由来するが、**別々の役割**を担う。Locatorを第二Graph Storeにせず、MapのGraphを第二Source Authorityにも使わない。

新設の`TestRealWorkspaceTypedModelMapAndModelLocateAgreeOnRelation`は、現行`reference/workspace`の**実`semantic-relation:sales.order.lines`一件**について、両既存Operationを呼ぶ。独立する観測として：

| 観測 | 期待する同一性 | 未資格 |
| --- | --- | --- |
| Source/Version | `Workspace identity`、`WorldFingerprint`、`SourceIndexFingerprint`、`ModelRevision`の値がMapとLocateで一致 | **別Load**なので原子的な同時観測、外部Provider Raw/Manifest/A→B→A履歴 |
| 選択要素 | `Kind+ID=semantic-relation:sales.order.lines`、`SourceStatus=indexed`、`read-only` | 図上に表示されたLabelだけでCanonical選択を確定 |
| 親とTarget | Locate親経路の末尾`semantic-data(parent)`と`relation-target`のFrom/Toを、Mapの`contains(parent→relation)`／`refers-to(relation→target)`の**各一件**へ照合 | Functionや業務Action全件の存在、Same-EndPoint別Edgeの安全なWrite資格 |
| 部分集合 | `limit=1`でも`coverage.notDerived`に`query-data-dependency`が残ること | 表示されない依存が「不存在」、Map`resolved`でMeaning/GrantがAccept |

**検収上限**は実Sourceの一Relationのread-only二Reader同一性。以前の3 Reader (Catalog/Inspect/Trace)と型付きM0への接続を確認するが、Graphの全体像・Function CRUD・データ取得・Studio画面・MCP実受領のE2Eには昇格しない。CLI/MCP間の輸送形式やDisclosureも別検査。ユーザーがDiagramの一部を選択してAIへ問う場面で、`selected string`だけでGraph・Source・権限を推論させない。

### 33.2 選択要素からHarnessへ渡すContextを、小さく・拒否可能にする

[main統合 #3863](https://github.com/hat47x/tei-platform/pull/3863)において、Studio上のFunction/Query/Rule/Trace等の**細粒度Selection→限定Design Context→AI提案→検証・承認・適用**の設計方針は既に計画へ採用済み。ここでは三視点Modeler固有のSelection Requestを既存Ownerへの受入票として定義するだけで、Harness/AI製品をModelerに内蔵せず、DSH専用の接続層を新設しない。

| 選択Contextの最低欄 | 実行時の取扱い |
| --- | --- |
| `Product/Workspace identity`、`Qualified Subject Kind+ID`、`View/Occurrence ID`、選択時点の`World/Index/Model Revision` | 異なるCanvasに同じSubjectが何度現れても同じ意味正本。別Workspaceや別表示Occurrenceと混同しない |
| `Purpose/Requested Operation`、`Bounded Scope`、`Limit/Total/Omitted`、`coverage.notDerived`、`unresolved/ambiguous` | 要求された周辺情報だけを投影。見えていない部分をゼロ件・完全と扱わない。大量の会話ログ・Source全文を既定送信しない |
| `SourceRef/Observation Status`、外部Rawの鮮度不明、`Current Disclosure` | Loader/Index上の参照可能性と、Actorへの開示・外部AI送信許可を別評価。欠落・非開示・古いSourceをAIが補完しない |
| `Candidate Diff`／再Load後の`Revision`／`Owner Acceptance`／`Host Grant`／`Effect Receipt` | AI提案を自動的にSource/業務効果へ適用せず、現在版を再検査して実Ownerへ受理を戻す |

同一Harness SessionをTEI/SUI/SEI/SHOの間で共有する場合も、Sessionは**選択対象への参照と復帰位置**を引き継ぐにとどまる。元製品の権限・意味受理・DisclosureをSessionへ集約して移植しない。DSH先行の理由はTEIのEverything is Pluginとの実装適合性であり、他のHarnessを排除しない。H0決定論的TEI操作、H1外部Harness、H2交換可能なHarness/モデルの同一Task・同一Independent Expected比較は#3863側が所有する。

### 33.3 最初の実業務縦断を妨げない実装順

- **P0-R：#2586の`order_entry.recent_orders`**。単一Function→Query/SQL byteとargs→正式に選択/InvokeしたProvider→実Rowset→明示Mapping→Data Context→Consumerまでの同一Attemptを最優先。Go実Referenceの限定結果、テストConnectorと実DB、Current Grant/Owner Meaningを別列で判定する。AIモデルの強い推論は、この最小横断のRoot Cause/不足接続/負例に集中する。
- **P0-D1：#1251の正式Plugin Provider選択→実呼出し**。main統合[#3843](https://github.com/hat47x/tei-platform/pull/3843)の宣言制限付き能力呼出しや[#3540](https://github.com/hat47x/tei-platform/pull/3540)のReference Inspectorを、外部Provider/S2-product/Verified BOMの完成としない。
- **P1-M：今回のM0↔Locate Go Test → 三視点Graph/CRUDとUI Selection**。先に実RelationのSource連鎖を確認し、M2で一意なRelation Identity、CRUD Cell Source、Unknown、Consumer Disclosureを条件にread-only投影。編集はSource/RevisionとOwnerで一意性が成立する場合に限る。
- **P2：Studio/SHO**。SHOのUTF-16→Source byte range、Working Copy/Save As、文書参照の変更は同様の安全対照として利用するが、Document WriterとTEI意味モデルの適用・共有Harness Sessionを別受入に残す。

新しいGate／Registry／Domain Parser／Graph Authority／Host／Runner／専用Harness Session Storeを導入しない。必要なテストは既存Go/CLI、[#2514](https://github.com/hat47x/tei-platform/issues/2514)/[#1180](https://github.com/hat47x/tei-platform/issues/1180)のIndependent Expectedに接続する。

**現在の状態**：本§33・二Reader照合Go試験は当該Draftのソースとして保存する計画変更。新Headの`gofmt/go test/vet`とRepository Healthが実際に成功するまで、Product MapやAI-Harness操作を合格扱いしない。図表の完全性・実Studio/SHO/DSH送出、業務効果・Meaning Acceptanceは未実証。

## 34. TEA三要素×三層を「問い相対の証拠」として投影する（2026-11-12、採用前）

**状態**：Method／Cross-viewの設計・受入候補。新Core Kind・9セルCanonical・Evidence Store・Policy Engine・Runner・Hostを採用しない。新たな実CLI/Go/Node/Studio/Provider試験や業務Owner受理は主張しない。[三要素Method Draft #4478](https://github.com/hat47x/tei-platform/pull/4478)はLifecycle/AI Harness、[実CLI→Node Draft #4396](https://github.com/hat47x/tei-platform/pull/4396)はCatalog/Validation表示を担当。本節は図表間Cross-referenceに**何を根拠として表示・診断できるか**だけを扱い、両PRの変更pathを触らない。

### 34.1 三要素と三層は直交するが、9種類の正本を作らない

[三要素分析法の原典説明](https://atmarkit.itmedia.co.jp/fbiz/cbuild/complete/doa/01/01.html)ではData／Function／Businessの各観点を論理要件・物理制約・実装で見る。TEIではこれを**9個のモデルや状態フィールドではなく、既存Source・Semantic Role・Binding/Target・Evidenceに対する問い相対のProjection**として扱う。元の方法論は業務システムに特化するので、汎用TEIの他Domainへ全語彙を強制しない。

| 観点 | 論理要件 | 物理制約 | 実装・観測 |
| --- | --- | --- | --- |
| Data | 業務Identity、Key/FD、値域、Relation | DBMS、Storage、所在地・分離制約、Schema/Provider選択条件 | 実Schema・制約、実Rows/Read、移行と品質の独立観測 |
| Function | Interaction/UI、Command、I/O、Function-level CRUD | 言語/Framework、Capability、実行・Transport制約 | 実Artifact/Provider選択、Invoke/Read、Effect/Receipt/独立Readback |
| Business | Procedure、Manual/External Step、Actor、Trigger、業務規則 | 人員・組織・担当配置、業務時間、社内制度・社会的制約（IT設定とは限らない） | 実際の作業遂行と業務結果の観測。ログだけで人の実施やOwner受理を代替しない |

物理制約のうち人員・社会的条件は自動抽出を前提にせず、出典・時点・開示資格を必要とする。Business Actorを実Authorization Principalへ昇格させない。物理FKから業務Keyを、実Rowから全母集団を、Function宣言から実Action Effectを逆算しない。

### 34.2 Cross-view Queryの最小構成（説明用で正式Schemaではない）

同じSource/Accepted Meaningから導出した図・CRUD表・CLI/MCP・SHO表示は、用途に応じ公開資格済みSliceの**問Qに対する証拠**を示す。既存Typed Ref・Evidence・Coverage・Projectionの組合せで次を保持する。別のStore、DSL、Authorityを作らない。

1. **Question/Use-site**：例えば「Business ActionがOrderを更新したか」と「FunctionがUを宣言するか」を別の問にする。
2. **Subject/Relationの同一性**：Workspace観測、Qualified Subject（Scope/Kind/ID）、多重Relation occurrence、Source/Meaning Revision、effective time、各Reader Attemptの対応関係を保持。同名・表示ID・Source locator・Fingerprintだけで同一Workspace/Readと見なさない。
3. **Premises/Derivation**：Source-backed、declared Function-level C/R/U/D、derived-potential、inferred-indirect、observed-effectの起源を分離。導出Rule/version、複数Source、競合宣言、未解決Endpointを保存。
4. **Coverage/Disclosure**：問に必要なSource、Role、Relation、Writer、Observationの母集団と被覆を宣言。非表示・非開示・not-examinedは0件にしない。Raw CountsやPrivate Edgeは公開Sliceより前に資格審査する。
5. **Realization/Observation**：Provider/Artifact/Target/Build Unit、Operation/Attempt、I/O、Tx/Recovery、Receipt・独立Readbackを別Evidenceへ参照。静的Sourceの一致は実呼出し成功ではない。
6. **Assessmentの限界**：根拠付き矛盾、実現保証の欠落、証拠不足/失効、明示的対象外を区別し既存Validation/Trait Assessmentから説明する。新しい全域Core Enumにはしない。

**不存在判定**：あるQとScopeについて必要なSource/Relation/Roleの閉包、現行のReader/Rule版、公開資格が成立する範囲でのみ「そのScopeでは該当なし」と言える。未知Writer・未読込Source・部分Model Map・非開示Relationがある場合はunknownであり、全域の越境CUD 0や「図にないので不存在」と表示しない。非公開対象の有無や件数も漏らさない。

### 34.3 三視点を跨いでもDirectnessと実行の資格を失わない

- Business Action→Function→Dataは導出経路であり、Function-level UをAction固有の実Uへ昇格させない。Functionを持たないManual/External Actionも残す。[#1031](https://github.com/hat47x/tei-platform/issues/1031)のX-TEA Source観測は別Evidence。
- 同じFunction→Dataの複数R/Uや矛盾宣言を行・Occurrence単位で残す。Draft [#4403](https://github.com/hat47x/tei-platform/pull/4403)は、合成CharacterizationでCRUD表の複数行とCrossReferenceの先頭優先、矛盾報告があってもOK=trueになり得る境界を記録した。これは製品Validator変更・全Writer把握の証拠ではない。#1127/#1132/#3250が判断Owner。
- ETL/ELT Transform/Join/AggregateはFunction/Operation側のData lineage。スケジュール・承認・手動引継ぎ・File Relayは該当Business/External Operation Roleを別に示す。Batchの結果値、全Member完了、Publish、Recipient受領、Resume、Delta、Effect/Recoveryを同一視しない。[ETL/Batch設計](ETL_BATCH_FUNCTION_COMPOSITION_DESIGN_2026-11.md)と[#1130](https://github.com/hat47x/tei-platform/issues/1130)へ委譲する。
- Subsystemの論理更新責任とData所有候補、direct CUD、委譲、effective/observed Effect、物理Target、業務Ownerを別軸にする。[#3250](https://github.com/hat47x/tei-platform/issues/3250)の候補・ScoreはAcceptedや実Tx証拠ではない。
- モデル由来TestはObligation候補を[#4134](https://github.com/hat47x/tei-platform/issues/4134)へ渡すが、モデルから生成したExpectedで同じモデルの正しさを自己証明しない。受理済みRule／手書き固定Expected／独立実観測が必要（#2514/#1180）。

### 34.4 追加の反証候補（すべて新試験NOT RUN）

| ID | 対照と最小期待 | Owner |
| --- | --- | --- |
| TEA-Q01 | 実sales.order.lines RelationのModel Map/Locateは同一Qualified意味を指すが、Source移動後は旧Locatorを失効。別Read Attemptを原子的同一と呼ばない | #1120/#1124 |
| TEA-Q02 | 同じFunction→DataにRとU、または相反するCRUD宣言。先頭行への縮約による喪失・競合を明示し、唯一Writerや無矛盾を断言しない | #1127/#3250/#4403 |
| TEA-Q03 | Functionを持たないManual ActionとFunction-level Uの経路が同居。欠落扱いせずAction固有実Effectは不明 | #1125/#1031 |
| TEA-Q04 | ETL Batch exit成功でも一部Member未公開・Recipient独立Read欠落。全件完了や自動再実行適格を推定しない | #1130/#2604/#1174 |
| TEA-Q05 | Subsystem間direct Updateを委譲候補へ置換。score改善をEffective Effect/Tx/Recovery証明にしない | #3250/#267 |
| TEA-Q06 | Closed Domain IntentとJava/SQL制約観測はあるがWriter集合または人員条件が未知。保証欠落と実違反を区別する | #3278/#1126/#2386 |
| TEA-Q07 | Private Relationのみ変えた二世界、同名異Scope、別Workspace同Snapshot。Public Graph/ARIA/ToolにPrivate件数を漏らさず旧Bindを拒否 | #2516/#1124 |
| TEA-Q08 | 意味RuleとExpectedを同一生成器で作った後にMutantを実行。自己生成Expectedでは合格不可、独立固定Expectedで照合 | #4134/#2514/#1180 |

### 34.5 最小組込み順と停止線

**P0**：#1124の同一Consumer実Source/選択/currentnessと#2586の実Provider Readを最優先に保ち、実Relation一件からData/Function/Businessの根拠付きread-only Queryを試す。#4396の実CLI→別Node限定証拠をSame Attempt/Owner受理と呼ばない。**P1**：#4403の多重CRUD・矛盾を既存Model/Consumerの受入候補としてOwnerへ返す。Validator Error化、Directness、Owner採否はここで決めない。**P2**：ETL・Subsystem・Traitの必要なEvidenceだけを問Qへ結合する。**P3**：一意で可逆なEdit IntentだけProposal→Impact/Validation→既存Shared Acceptance→再投影。Plugin Build/複数Instance/Self-buildは#951/#2831の別資格。

**撤回条件**：既存modelref・Evidence・Cross-view表で問いと根拠を説明でき、追加抽象に効果が無い／第二正本を作る／Private件数を漏らす／derivedをActualへ昇格する／Method更新だけでAccepted Meaningが変わる場合は専用の実装を追加せず既存Projectionへ戻す。**TEA-Q01〜08の新試験、実Policy/Studio GUI/Provider/業務Owner受理はNOT RUN／NOT ACCEPTED**。

### 34.6 量化を伴う問いと、成立・反証に必要な証拠を分離する（2026-11-12 横断深化）

**設計上の差分**：現行§34.2はQuestion/Scope/Evidenceを扱うが、「一件見つければよい問い」と「全件を確認する問い」を同じClosure条件へ押し込むと、正当な局所発見を不必要に拒否したり、部分観測から誤った不存在を断定したりする。ここでは新Proof Engine/正式Schemaを作らず、既存Typed Ref・Evidence・[#3262](https://github.com/hat47x/tei-platform/issues/3262) Design Context Required Closure・[#2514](https://github.com/hat47x/tei-platform/issues/2514)独立Expectedへの**解釈規則**を記録する。

| 問いの論理形と証拠対象 | 成立を支え得る証拠 | 反証を支え得る証拠 | 誤用してはいけない根拠 |
| --- | --- | --- | --- |
| **存在**：固定Scope内に宣言済みDirect Updateが一件以上ある | Qualified Relation/Occurrence・出典・現在Meaningが結びついた**一件の陽性Witness**。全件列挙はこの存在主張の必要条件でない | 対象Function・Data・必要Source/Roleの**閉包が確認された範囲での0件**。未読込はunknown | 部分CRUD表の空セル、derived-potentialからのActual Update |
| **全称**：対象Functionすべてが特定の制約を満たす | 閉じた対象集合を網羅して検査した証拠、または適用範囲を明示した有資格の意味規則・実現保証とその適用証拠 | 同じ母集団・条件に属する**一件の有資格な反例** | 少数の正常サンプル、Validation OKだけで全Sourceの閉包を代用 |
| **不存在／禁止**：越境Direct CUDがない | 全Writer/Operation/Scopeの定義と対象閉包、およびdirect性の解決 | 対象閉包内の越境**Direct** C/U/Dの有資格一件で禁止主張を反証し得る | 未解決Writerを0件扱い、委譲や潜在EffectをDirectと同一視 |
| **実行事実**：特定のActionでOrderが実際に更新された | 実Operation/Attempt・対象・時点に束縛したEffect/Receiptと必要な独立Readback | 十分に観測した同一実行範囲での反例。ただし見えなかっただけならunknown | Function-level U宣言、UI操作成功、HTTP ACK、Batch exit 0 |
| **業務受理**：更新は業務的に正当だった | Accepted Rule/Owner判断と実業務結果を、主体・適用時点を揃えて比較 | 適用条件に反した有資格の業務反例 | 実DB値の変化、生成器のPass、MethodやProviderのロード成功 |

**証拠の非対称性**：一つの陽性Witnessは存在命題を限定的に支持し得るが、全称の成立には十分でない。一件の反例は対応する全称を否定し得るが、反例が未公開・Scope不一致・旧版・別Attemptなら現在の命題の反証にしない。逆に存在の否定は完全閉包を要する。有限データ全件の走査以外にも、有資格な不変条件の演繹的保証は成立方法の候補だが、宣言した規則の正しさと実Providerが実際に履行したことは別の証拠で確認する。特に業務MECEの母集団・排他・網羅をJava Enum単体の存在から導かない。

**限定判定の手順（新Core State enumを増やさない）**：
1. 現在のUse-siteに適した**公開済みSlice**をOwnerから受ける。Secret/Privateの有無・件数・候補Scoreを低権限側で推測しない。
2. 問Qの母集団、対象Scope、量化（存在／全称／不存在／一試行の事実）、意味対象、Revision・時点を固定する。SubjectとRelation Occurrenceを混同せず、表示上の同じラベルから同一性を作らない。
3. 必要なSource・Role・Relation・Writer・Provider／実Observerと、返却・ページ・Consumer完全受領のCoverageを別々に照合する。局所の陽性Witnessを全件Completeと呼ばず、exit 0やtraceのunresolved 0をQueryの完全性へ昇格しない。
4. 主張の種類に適合したWitness／Counterexample／Rule依存を評価し、**どの主張を支持・否定できるか**と残余Unknown／古い根拠を提示する。Validation/Assessment/Source Acceptance/Current Grant/Observed Effectは既存Ownerからの別資格であり、一つのverified Booleanへ縮約しない。
5. Claim由来のDataset・Scenario候補は[#4134](https://github.com/hat47x/tei-platform/issues/4134)へ引き渡す。**同じ生成器が候補と期待値を同時生成した成功は独立Oracleではない**。人手固定Golden／Owner受理Rule／独立Expectedを維持する。Method/Source/Rule改版時は先行のWitness・Expected・Receiptを現在の正当性へ無条件に流用しない。

### 34.7 問い相対の反証対照（TEA-Q09〜Q14、全件NOT RUN）

| ID | 同じ入力をどう問うか | 必要な独立Expected・拒否条件 |
| --- | --- | --- |
| **TEA-Q09** | 部分ロードしたFunction→Data CRUDに一件だけSource-backed Uがある | 「この固定Scopeに宣言Uが存在」は限定支持可能。ただし「すべてのWriterを取得」「越境Uは0」への昇格を拒否 |
| **TEA-Q10** | 十Functionのうち一件に同じ適用条件を満たさないUpdate宣言がある | 全Function制約の反例は一件で示せるが、Business Action固有Effectや別Scopeの実違反へ誤伝播しない |
| **TEA-Q11** | LIMIT/ページ欠落・Loader未取得によりCRUD表・Business図で対象が見えない | 既知の0件、未読込、未受領、権限不明を別に保持し、存在否定・全件完了を拒否。実Consumer完了条件は#3262 |
| **TEA-Q12** | Function-level UとBusiness Action→Function経路があり、実行ACKのみ存在 | Declared／derived potentialは説明可能だが、当該Actionの実Updateと業務完了を支持しない。Receipt/独立Readbackは#1174/#1180へ |
| **TEA-Q13** | ETLで各Chunkの部分集計は正しいが、総Member閉包・重複排除・丸めの適合が未観測 | 「観測済みChunk内で成立」と「対象期間の全件で成立」を別判定。Data型/単位/NULL/精度を保存、#1130/#2514 |
| **TEA-Q14** | 同じ公開Source/Relation Sliceを維持したまま非公開Writerだけ変更 | 公開Consumerの図・Counts・診断・順位は変わらない。全体の不存在証明を公開側へ漏らさずunknown。実Disclosure/side channelは#2516 |

**位置づけと優先順**：本節は#4473のPinned Target/媒体横断Contract、#4478のLifecycle Graph、#4403のCRUD検証コード、#4396の実CLI→Node、#4134のTest Generator、#3262のDesign Context Queryの**代替実装ではない**。まず#1124同一Consumerと#2586 Product実Readを進め、TEA-Q09〜14は既存Read/Rule/Expectedを再利用した小さな局所対照から検討する。追加試験のGo/Node/DB/Workbench/SHO/AI実行、現在Policy、正式受理はすべて**NOT RUN**。独立の新Proof Store・汎用Rule Engine・Method専用Hostを作らない。
