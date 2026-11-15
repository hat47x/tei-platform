# 設計Projection・実行版・データ移行を横断する受入計画（2026-11）

## 位置付け

本書は、既存の標準Data／Relation／Rule／Function／Process、生成契約、Evidenceを、設計書・関係図・帳票・配備・データ移行・複数Entryへ適用する**追加の検証計画**である。新しいCanonical／独立Host／Registry／Migration Engine／業務権限発行器を新設する提案ではない。新規に追加するIDは独立Expectedの候補であり、既存#1180/#2514をテスト正本として使う。製品実装・Owner受理・実Provider実証を完了したという意味ではない。

## 1. 共通の4層を分ける

1. **Source/Meaning**：編集元byte、Stable IDとRef、Accepted Meaning Revision、Owner受理。未保存Draftも区別する。
2. **Projection/Artifact**：文書・表・図・帳票・コード・APIの生成選択、Generator/Plugin/Template版、Dependency closure、成果物実byteとdigest。表示順序やレイアウトは業務上の意味とは独立。
3. **Loaded Runtime**：外部HostやToolが実際に読み込んだBinary、Plugin、設定、埋込UI、Schemaとその版・時点。ビルド済み≠ロード済み。
4. **Business Effect/Consumer**：実Provider操作・DB移行・Job出力・現在Grant・独立Read・結果検証。実行成功やJob完了だけをBusiness Verifiedとしない。

World Fingerprint、Model Revision、Artifact Revisionは既存指標を利用する。Runtimeの読込状態を同じ指標へ押し込まず、Observerの別票として記録する。

## 2. 標準Projectionの小さな横断契約

- **Design Report**：Source Revision・対象モデル集合・選択した出力章・Audience・書式Template版を受け取り、Markdown/Excel/PDF等へ展開。モデル別選択が全体章へ及ぶと推測しない。編集途中Draftが保存済みと違う場合は出力の出典を明記する。
- **Graph/ER/Node View**：Node/EdgeのStable ID・向き・多重辺・Relation roleと、座標・倍率・フィルターを分離。同名Node、複数Edge、部分表示、未解決Refを組み合わせても、見た目だけで保存対象を確定しない。表示形式の相互変換で落ちる意味は明示診断。
- **Report Artifact**：Rule／Queryで定めたデータ集合、対象Snapshot、Template Revision、Generator/Renderer版、結果Artifact byte、取得可能期間・現在Disclosureを区別する。
- **Function Entry**：Web、Desktop、CLI、REST、MCP等はFunction Meaningの異なるEntry Adapter。公開ツール一覧、実行能力、現在Grant、Result disclosureは別資格。MCP等を無条件に全機能の代替としない。

これらは既存Standard Data／Renderer Plugin／Source Index／Function Operationを利用し、形式ごとに第二意味モデルを作らない。

## 3. 定義変更から安全なデータ移行へ

`Model Delta`、`Generated DDL`、`Migration Plan`、`Current Database Schema`、`Applied Migration History`、`Observed Data`は別の対象である。

- スキーマ名やフィールド表示名の改訂と、Stable ID／物理列／保存型の変更を区別する。
- 生成SQLがあることと、対象DBがそのSQLを適用済みであることを区別する。
- 旧データの取込みと、現行環境のロール・権限・主体・所有関係の入替えを区別する。
- 互換性が不明な変換、欠測、Version mismatch、現在Grant不明の場合は自動適用しない。破棄可能なFixtureと実データ環境も同一扱いしない。
- 実行とReceiptはTarget Provider/Hostの責務。TEIは受理された変更意味、生成された計画、実観測の一致・不一致を説明する。

## 4. 次の負例（合成、実行未了）

| ID | 固定する差分 | 不合格とすべき誤主張 |
| --- | --- | --- |
| XP-01 | 保存済みS1、未保存Draft D2、一部モデルだけの設計書出力 | D2を出力した／選択外の全体章も絞られた |
| XP-02 | 同ラベル2Node・多重有向Edge・表示フィルター差 | レイアウト移動がRelation変更／非表示Edgeは削除済み |
| XR-01 | Model rev同一、埋込UI/Plugin/Configの実ロード版が旧 | fingerprint一致だけで配備完了 |
| XM-01 | 新モデル・生成DDLは正しいが、実DB列型は旧型 | 『生成成功』をMigration成功・実Read成功へ昇格 |
| XM-02 | 移送元データに旧権限・所有者ID、移行先AuthZは別世代 | データ移送成功＝移行先Grantまで置換可 |
| XA-01 | 帳票生成前後でTemplate版変更、結果取得時Grant失効 | Job完了から新書式出力・取得許可を推論 |
| XE-01 | CLI/RESTはFunction対応、MCPは検索のみ、Token権限は異なる | MCP上のツール発見を全Functionの実行許可にする |

Expectedは既存Scenario Test SSOT #1180に置き、各Targetの実Runは#2514の独立Observer検証と結ぶ。上記だけで新しいテスト実行基盤を増設しない。

## 5. 実装順と受入条件

- **P0（変更なし）**：正式`*.tei.md` Reader選択→型付き意味→実Go生成→実Builder→通常CLI・実Provider→独立Consumer。並行してStudio Windows Stage1→Stage2（#2621/#1018/#4495/#2831/#4583）。
- **P1-A**：XP-01/XP-02を既存Source Index、Graph、SHO Composer／Studio read-only投影で試験。出典ID・出力選択・未保存版・図座標の差が説明できること。
- **P1-B**：XR-01を実Build/Runtimeの同一試行のEvidenceへ接続。World/Model/Artifact/Runtime readを分離し、部分不一致で総合PASSを返さないこと（#2250/#2255/#1163）。
- **P1-C**：XM-01/02をDry-run→隔離された実DB Provider→独立Readへ接続。移行先の権限を勝手に置換しないこと（#1121/#1129/#1174）。
- **P2**：XA-01/XE-01を実Template／Artifact body／現在Grantと、Web/CLI/REST/MCPの共通Function＋入口別適合へ展開（#1130/#1180/#2514）。Adapter上の存在だけで実運用対応を主張しない。

## 6. 非目標とOwner境界

- 対象アプリの表示・設計図・出力書式をTEI Coreの業務Meaningへ固定しない。
- 外部Hostの実認証／配備をTEIが肩代わりしない。実GrantはHost/Provider、業務意味の受理はOwnerに置く。
- 履歴・fingerprint・生成成功を、DBが変更済み／権限が有効／業務結果が検証済みとの証拠に使わない。
- Template互換・MCP書込み・全ProviderのMigrationを初期要件として約束しない。

**現況**：上記は計画の追加のみ。新Go／Node／Windows／実DB／実API／帳票Jobの製品E2Eは本差分で実行していない。新しいEvidenceで上限を超えて受入を主張しない。
