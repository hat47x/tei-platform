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
