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
