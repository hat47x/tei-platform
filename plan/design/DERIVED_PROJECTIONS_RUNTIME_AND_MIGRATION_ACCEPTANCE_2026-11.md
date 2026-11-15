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
## 7. 一つの受注機能から、異なる成果物まで意味を保って追跡する（2026-11-15 横断統合）

### 7.1 三つの業務対象を、一つの画面・テーブル・フローへ潰さない

最初の縦断例は「受注の明細編集→確定→一覧・帳票→別Entryでの照会」とする。既存の`order_entry`関連のSourceとDriver Scenarioを候補として使うが、実Sourceの未確認項目を実装済みと仮定しない。モデルを増やす前に、次の役割をそれぞれ固定する。

| 役割 | 例と受理対象 | 各Projectionで保持すること |
| --- | --- | --- |
| Data | 受注・明細・商品・取引時点単価、業務キー／複合キー | 業務Identityと物理PK／外部キー、NULL／未取得、Relation経路・取得時点を分離 |
| Function | 受注編集、受注検索、納品一覧の出力 | 同じData／Ruleを共有しても、Query・入力・計算範囲・表示・書込み可否はFunctionごと |
| Business | 受注確定、承認、更新の責任、取消・再実行条件 | Actionと処理手順、状態の変化、現在のActor/Grant、Effect Receiptを分離 |

この区分は三要素分析法のデータ・機能・業務モデルとの相互照合に使う。ただし原典のモデルとTEIの標準Data／Function／Processを同一Schemaと決め付けず、Source-backedの対応表として扱う。表示図やCRUD表は派生表現であり、物理FKだけから業務操作を逆算しない（#1124／#2497／#2506）。

### 7.2 同じ項目・式でも、評価と保存の時点で意味が変わる

- **参照**：商品名の現在参照、受注時単価の保存Snapshot、参照先キーの保存を分ける。保存時刻の単価と再表示時の現在価格は、一致しない場合が正しい。Lookupを一つの「参照」という型だけで済ませない。既存Relation Path／Temporal／Data Ruleを使用し、新しいLookup Engineを先行作成しない。
- **計算**：初期値、Draft表示中の親子合計、保存前の再評価、読取時計算は同じ式でもInput Scope・完全性・時点・許される副作用が異なる。式の一次定義（型付きOperator／Rule）と、その適用Phase・Function Contextを分ける。式をUI・保存処理・帳票に複製して直す方式は採用しない。
- **親子変更**：画面に見えている明細だけでなく、変更されなかった保存済み行、追加・修正・削除予定行を揃えたDraft Snapshotを計算対象とする。ただしDraftの表示結果は実DB Commitではない。保存直前にCurrent DB、複合キー、Version、Owner認可、Ruleを再検証し、確定後に独立Readで照合する。
- **検索**：絞り込み条件、現在のDisclosure／RLS、索引の非同期鮮度、ページングによるpartialを別に保持。単一のQueryが0件でも、対象全体に存在しないことやほかのRoleで見えないことは証明しない。
- **業務状態**：申請時の経路と必要人数のSnapshotは、現在の候補者の実行資格や委任の有効性とは違う。Business Instanceは元データ側の状態を原則とし、単一の全業務共通State表／新Workflow Engineを前提にしない。

第一段階では既存Data Rule／Function Data Context／Process／Authorityの適用条件を対応付ける。実際の評価／保存／承認／全文索引／通知を確認していない辺は`unverified`のままにする（#1637／#1640／#1129／#2386／#2622／#2586）。

### 7.3 同じ意味に対する異なる表現を、用途別の派生物として扱う

一つの受理済みSource／Ruleから、型付きInspector、Data・Function・Business図、Graph/ノード、設計書、Go実装、Web/Desktop画面、帳票、API/CLI/MCPの入口を派生させる。表現が異なっても同一であるべきものと、異なってよいものを分ける。

- **意味が保存されるべきもの**：Qualified ID、Relation identity・向き・多重性、型とNULL、Rule依存、OperationとResult、Source/Plugin/Generatorの版、対象データの粒度・時点。
- **用途ごとに変えてよいもの**：図の座標と表示順、画面レイアウト、設計書の章選択、レポートTemplate、CLIの表示形式、MCP Toolの説明。ただし変更の影響がないかは依存閉包で確かめる。
- **生成だけでは同一性が保証されないもの**：実ロード済みWeb資産、実Builderが読んだGo byte、実DBの列とデータ、外部Hostの現在認可、帳票Bodyと取得可否、APIの実Call、Consumerが受け取った値。

これを`Working Draft → Selected Source → Accepted Meaning → Projection → Artifact → Loaded/Invoked → Observed Result`の別欄で追跡する。A→B→AのSource移動、同名別Root、失効したPlugin Lock、検索Indexの遅延などを単純なbyte一致で相殺しない。既存のWorld Fingerprint／Model Revision／Artifact Revision／Evidence／Source Indexを利用し、第二のCanonicalや汎用ロード台帳を作らない。

特にGraphとNode Flowでは、**構造を描く線、値の依存線、制御の遷移線、Plugin提供・依存線**を意味Roleごとに区別する。見た目が同じPortでも適用Phase・入出力型・時間／認可条件が合わなければ接続不適格とする。座標編集は意味変更でなくView改版、Sourceへの逆書戻しは元spanと一意の編集対象が確認できた場合だけ候補化する（#2497／#4198／#951／#1124）。

## 8. 横断Plugin・生成・環境移送の受入境界

### 8.1 まず一つのRuleを実Goで消費する

現在の内蔵Markdown Rule Readerは、`*.tei.md`をManifestで選択し、限定Ruleを型付き意味へ読める経路を持つ。したがって次に作るのは別Parser／新DSLではない。

1. 選択された`*.tei.md`から、Rule ID・元byte位置・型・依存Field・適用Phaseを一件読み、選択ReaderとPlugin/Source版を固定する。未選択・誤配置・部分読込みは拒否する。
2. 同じRuleから生成された**実Go byte**を、既存Builderが実際に受け取る入力と突合し、実ArtifactのMember・digestを記録する。参照Generatorの単独実行結果を製品Builder実績へ昇格させない。
3. 通常CLIで受理済みFunction一件の実Provider Readを行い、入力Query/Environment・実SQL/Parameter・返却RowsetとFunction Context、独立Expected・一つのConsumerを対応付ける。Providerの選択とactual Call、0件とpartial、NULLと未取得を別観測とする。
4. Rule生成と実Providerの各経路が独立に合格した後に、一つの業務Functionで結合する。Write／Commit／ACK lost／Outboxと現在Grantは#1174／#1129の別資格で検収し、Readの成功をWriteに一般化しない。

同じPlugin PackageがLanguage／Operator／Data Provider／Target Generator／Rendererを提供しても、**提供したRole、選択された版、実際に生成・呼出された機能**を個別に検証する。静的合成をGo単一バイナリの第一経路とし、必要な場合だけ外部Process Plugin／外部Host Adapterの隔離・配布・更新を評価する（#951／#988／#1121／#1251／#2595／#4495）。

### 8.2 移行計画はDB構造・データ・現在認可の三つを同時に扱うが、同一の確定処理にはしない

Migrationの最初の受入は**隔離DBのread-only観測とDry-run**で、旧新のStable ID、業務キー／複合キー、物理列、型、制約、DDL所有者、Data Conversion案、既存データ、移行履歴を並べる。その後、許可されたProviderで実DDL/変換を適用し、独立Readerで再取得する。Historyだけの修復をDB実型変更と扱わない。

業務データの移送で旧環境のRole・Grant・Owner ID・申請中のRoute Snapshotを無条件にコピーしない。照合できない主体・権限・添付Bodyの状態は`unknown`／`partial`として停止する。Stagingからの参照解決や行単位の部分確定を必要とする場合は、既存Batch/Operation／Receiptの設計を使用する。検証のみ、全件原子確定、正常行のみ確定の違いを自動的に統一しない（#3270／#1129／#1130／#1174／#2622）。

### 8.3 非同期成果物は、受付・処理・生成・実取得を別に観測する

帳票、検索索引、通知、外部タスクは共通のProcess／Execution／Resultの表現で照合できるが、同じQueue／Scheduler／常駐TEI Hostを要求するものではない。ResultのBody digestだけで利用者への開示、独立検証、配達完了、取得期限を保証しない。Template改版や現在Grant変更があれば、生成・取得のどの資格を再確認するかを限定する。MCP／REST／CLIのTool Discoveryも、対象Functionの実行能力と現在のScopeを代替しない。

## 9. 次の実装パケット：既存Scenarioへ少数の差分を返す

本書§4のXP/XR/XM/XA/XEを**新しいテスト正本にしない**。既存#1180 Driver Scenarioと#2514独立Expectedへ、下記の一因子変化を関連付ける。既存DW／DC／CRP／WB／PLV等と入力・Expectedが同じものは再作成しない。

| 既存Case | 最初の一因子変更 | 合格に必要な別々の観測 | 担当 |
| --- | --- | --- | --- |
| XP-01 | 未保存Draft→保存済みSnapshot、設計書のモデル選択だけ変更 | 選択Root・保存rev・章Scope・出力byte・出典。Draftを出力したと誤記しない | #2621／#1124、SHO |
| XP-02 | Graph座標だけ→Relation role／Edge ID変更 | 表示差と意味差の失効区分、同名二Node・多重有向Edge・元Source span | #2497／#1124／#951 |
| XR-01 | Artifact byte固定、実ロードUI／Plugin／Config版だけ旧 | Builder実入力／Runtime実読取／Consumerを区別し、同fingerprintでも総合PASSを拒否 | #2600／#3267／#2831／#4583 |
| XM-01/02 | 物理列型だけ旧、移行先OwnerのCurrent Grantだけ不適格 | Dry-run、Apply Attempt、実DB Schema・値、Grant/OwnerとReceiptの独立観測 | #3270／#2622／#1174 |
| XA-01 | Report受付後Template変更／取得時Grant失効 | Template版、Query scope／Data snapshot、Jobと実Artifact body、取得拒否 | #1130／#1158／#2514 |
| XE-01 | CLIの既知Functionに対しMCPは検索しか提供しない | Tool Discovery、Binding適合、actual Provider Call、現在GrantとDisclosureを別扱い | #1180／#1251／#4495 |

**最初の実装差分**は、選択済みRule一件の`Source span → Typed Rule → generated Go → actual Builder input`を実Productで結び、既存Scenarioの正常・NULL・欠落・条件変更・旧Sourceの負例を走らせること。次の別差分で通常CLI→実Provider→Raw Rowset→Context→独立Consumerを確認する。双方の結果を一つの受注Functionに結び、SHO設計書やStudio図面はread-only Consumerとして追加する。実DB MigrationやMCP Writeは最初の二差分のHard Gateにしない。

**並行P0**：StudioのStage1→Stage2 Windows実セルフビルドは#2831／#4583に従い継続する。生成Goの実Builderと正規Wails Assetの埋込はそれぞれ実byteで検査し、どちらかのCI合格を他方へ転用しない。

## 10. 変更時の再検証と成果の測定

- **変更の波及**：説明文、Graph配置、式・Query・依存Relation、Generator／Plugin、Loaded Runtime、Grant、Target Schema/DB値の各変更について、元Source・意味・生成物・実Call・実Effect・Consumerのどこまで失効するかを既存依存Graph／Evidenceで示す。Source A→B→Aや同byte別Rootも再選択を要求する。
- **成功の定義**：ドキュメントやテストの数でなく、(a) 人が保守する定義の重複減少、(b) 不適格な接続・旧版を誤受入しない率、(c) 変更後に必要な再検査範囲、(d) 実Builder/Provider/Windows/Consumerの同一Attemptでの証拠、(e) 診断からSourceまで戻れる手数を比較する。計測はPlugin・設定・生成コードを含めた総作業量とする。
- **停止条件**：選択Readerが実Sourceを読めない、型／Phaseが不明、Builder入力が観測できない、DB Providerの実Callが無い、Graph対応が多義、現在Grant／Result Disclosureが不明、Runtimeだけ旧版などの場合は、その辺を`unsupported`／`unverified`／`stale`として止める。説明やMockが成功しても埋めない。
- **保留**：新Core型・新Grammar・Graph Compiler・Workflow Host・Migration Engine・Outbox Store・独立テスト実行基盤・万能Registry・AIによる自動業務意味受理は、この段階で追加しない。

**検証上限**：ここまでの§7〜10は既存成果の設計統合と次の実装パケットである。新たなGo/Node/Windows/DB/MCP実行試験、業務Ownerの意味受理、製品の完全適合は本追補では未実施。
