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
## 11. 同じ意味モデルでも実装選択と現在性が違う：実Go回帰からの再整理（2026-11-15）

### 11.1 既に実証した範囲を起点にする

前回§8〜9の「生成Go→実Builderはこれから」という説明は、製品コードの進展に応じて精密化が必要になった。現在mainには、次の**それぞれ独立した**実装と回帰がある。

| 実装済みの限定Witness | 実際に確認すること | これだけでは保証しないこと |
| --- | --- | --- |
| [#5020](https://github.com/hat47x/tei-platform/pull/5020)・`internal/semantic/validationgen/positive_surface_identity_test.go` | 同一型付きRuleをMarkdown／従来構文から読んだ場合、元Source位置・指紋・Model Revisionは区別しながら、生成GoとASTのRead・条件が一致。説明文だけの変更で旧Source Indexが失効 | 異なるSourceが同じ受理経緯／編集資格を持つこと、実Provider／Owner受理 |
| [#5009](https://github.com/hat47x/tei-platform/pull/5009)・`cmd/tei/cli_self_build_generated_rule_go_test.go` | 選択済み`quantity_positive`の生成Goをテスト用Go Entrypointへ配置し、通常`tei self-build`、Plan／Artifact照合、別Process Binaryの正・0・負・NULL・欠落を検査 | Authoring Source／Reader／Generator版を**同一Build Intentへ束縛したこと**、正式公開Function・実Provider、他Target |
| [#5007](https://github.com/hat47x/tei-platform/pull/5007)・`cmd/tei/scenario_operator_implementation_test.go` | 宣言Worldを変えずに`implementations:`の組込み実装を`non-negative`と`non-zero`で切替えると、入力0の結果と実Called IDが変わる。未選択・曖昧・未搭載は拒否 | 選択実装が業務上の意味に適合すること、外部Plugin Lock／実Provider一致、生成Scenarioの旧版失効 |
| [#5025](https://github.com/hat47x/tei-platform/pull/5025)・`cmd/tei-studio/test/desktop_webview_smoke.py` | 実WebView2用テストにWorkspace再読込→旧Pin観測失効→利用者の明示再確認を追加 | 同テストの全HEAD CI合格、Stage1からStage2へのWindows自己再構築、現在の業務Grant |

**注意**：この四つは同じRule／Workspace／実Attemptの連続E2Eではない。特に`quantity_positive`のGo生成試験と`discount_non_negative`のOperator選択試験は**別Fixture・別Subject**である。片方のPASSから他方の機能実装を推論しない。選択済み実装の`non-zero`は負値も真となり得るため、実装選択が成功しても「負値を禁止する」という業務要求への意味適合は別に検証する。

### 11.2 保存するのは万能の一致フラグではなく、根拠付きの対応関係

次の対象は異なるIdentityと改版軸を持つ。**同じ文字列や同じGo byteで一つに統一しない**。

1. **Authoring Source**：Workspace Root、Source path／occurrence／原byte／範囲、選択Readerと版、保存済みか未保存か。
2. **Accepted Meaning**：Qualified Rule／Operator／Relation IDと型、適用Phase、必要なRead、Ownerが受理した業務条件。未受理の候補は候補のまま。
3. **Resolved Realization**：Workspaceが選んだOperator実装、Plugin/Provider/Target Generator、選択版・Lock、Toolchain。宣言上の候補と実搭載・実Callを分ける。
4. **Build / Loaded Artifact**：実Builder入力Go byte、依存閉包、Artifact digest、実ロードBinary／埋込UI／設定とその観測時点。実生成結果と実ロード版は別。
5. **Execution / Consumer**：実Context、入力取得範囲／Snapshot、現在Grant、実Called ID、結果・Effect Receipt、独立Consumerの受領時点とAudience。

この5対象はSource→意味→実現選択→実Build→実利用という**検証可能な辺**で結び、全対象に共通の万能Revisionを新設しない。World Fingerprint／Model Revision／Artifact Revisionと、Runtime／Provider／Consumerの観測票を引き続き分ける。既存Evidenceの再利用可能性は各辺の依存集合とAuthorityを検査して判断する（#2250／#2255／#2600／#3267）。

### 11.3 変更の種類ごとに、残せる資格と失効する資格を区別する

| 一因子変更 | 保持できる候補 | 必ず再確認する境界 |
| --- | --- | --- |
| Markdownの説明文だけ変更 | 型付きRule、生成Goの等価性 | 元Source位置・指紋・引用／説明の扱い、旧Indexの現在性 |
| Rule条件・参照Field・評価Phase変更 | 無関係なData定義・Renderer | Typed Meaning、Read集合、生成Go、Builder／実行結果、必要なConsumer |
| 同じWorldで`implementations:`だけ変更 | Source Meaningの構造、無関係な成果物 | 実装選択・解決可能性・選択実Call、業務要求への適合、旧Scenario／生成物の実利用資格 |
| Source Rootの切替・A→B→A・同名別ID | 過去の発行記録 | 現行Root／Occurrence、Source Index、編集対象・依存先。byte一致だけでは復活させない |
| Graphの座標・設計書の表示順だけ変更 | 元Graph／Ruleの意味 | 該当Renderer／表示Consumer。実行Goを一律再生成しない |
| 商品価格・Relation・Read Scope変更 | 取引時点の確定Snapshotと無関係Rule | 現在参照値、Query取得完全性、照会時点、業務Ruleの依存。過去取引値の黙示的再計算は禁止 |
| 申請先組織の変更・委任／Grant期限切れ | 申請時Route Snapshot・承認実績の履歴 | 現在の行為主体と許可、残りの承認人数、後続Actionの適格性 |
| Generator／Plugin／Embed／Config変更 | 直接無関係のSource Meaning | 実Artifact、実Loaded／Invoked版、現在Consumer。必要な配布・再起動を実測 |
| DB列型・Migration履歴だけ変更 | 未変更の業務モデル・説明書式 | 実Schema／実データとMigration適用履歴を**別々に**照合。履歴修復を型変換済みとしない |

「残せる候補」はPASSを自動継承する意味ではない。必要な依存閉包・取得時点・Observer／Audienceがそろわない場合は`unknown`とする。これにより、意味が不変の文書編集で全Buildを無駄に失効させず、実装だけ変更されたのに古い判断・実行資格が残ることも防ぐ。

## 12. 時点・権限・部分取得を一つの受注例で比較する

受注の明細・価格Snapshot・検索・承認・帳票・環境移行を同じ案件で扱う際、少なくとも**取引上の有効時点、組織が知っていた時点、Provider記録時刻、外部観測時刻、Rule評価時点**を区別する。全Functionへ常時五つの日時列を物理保存する要求ではない。問いに必要な時点と取得根拠を、Data Context／Process Instance／Evidenceへ選択的に対応付ける（組織C1／#2386、業務State #1129、独立期待値 #1180／#2514）。

例えば、注文時の単価は保存Snapshot、商品マスターの価格は現在参照、承認経路は申請時Snapshot、実行資格は操作直前のGrantである。二つの異なる時点の値が一致しないことをエラーとしない一方、どの時点を使うか未定義の計算を自動確定しない。全文検索索引の準備完了と、元データ改版・添付解析の完了も別の観測となる。

- **Draft→Commit**：全子明細の追加・変更・削除予定を集めたDraft結果と、実DB保存前の再検証結果を分ける。親子のAtomic Commitと外部通知／添付BlobのAtomicityは同一視しない。
- **Batch→再実行**：検証のみ、全件原子確定、行単位部分確定を別Policyとする。同じファイル・同じ値でも、実行要求ID・行Operation ID・ExpectedVersion・Receiptを確認しなければ再送可能とはしない。
- **移行→公開**：旧DBからデータを移せても、移行先のPerson／Role／Grant、実Loader／API、帳票取得権限は改めて検査する。履歴のある業務主体と現在のログイン主体を一致と推測しない。

## 13. 次の実装は「二つの小さな接続試験」に限定する

| 作業Packet | 実装対象と反例 | 合格上限と既存Owner |
| --- | --- | --- |
| **PQ-A：出典付き実Build** | 現存する`TestSelectedRuleGoThroughRealCLISelfBuild`を基線に、同じBuild Attemptで**原Rule byte／Reader選択／生成器版／生成Go digest／実Build Intent source／実Builder入力／Artifact**を照合する。説明だけの改版、参照Field変更、Generator改版、Source Root A→B→Aを別に操作 | 出典と実Builder消費の正しい束縛まで。外部Host／実DB／公開Functionを含めない（#2621／#4544／#2595／#2600） |
| **PQ-B：実装選択の鮮度** | 現存する`TestScenarioDecidesAnOperatorRuleWithTheSelectedImplementation`を基線に、同じWorldに対する実装A→Bと、**旧選択で生成したScenarioコマンド／結果を再使用する負例**を追加する。選択ID・実Called ID・結果・版を区別し、意味に不適合な実装を期待値に合わせて正当化しない | 実装選択／Scenario再資格化まで。外部Plugin Lock、通常Function→実ProviderのR3やGrantまで主張しない（#988／#1018／#1180／#2514／#2600） |

両Packetは**別Subjectの独立回帰**として着手でき、最初から一つの巨大な横断テストへ統合しない。PQ-A／Bが成立してから、受注Function一件について`typed input → Rule → actual Provider Read → Result → independent Consumer`を同一試行で連結する。次段で選択済みRule/Pluginの実Target適合、必要なData Migration、帳票／APIなどの派生Consumerを検収する。

**並行する最優先課題**は変えない：通常`tei` CLIからの実Provider／独立Consumer（#2586／#1251／#4495）と、Stage1実TEI→Stage2実Windows WebView2（#2831／#4583）。Pin再確認試験と生成Go CLI Buildの結果を、Stage2完了の代用にしない。新Core Schema、第二Rule Engine、Manager／Registry、常駐Host、別Scenario正本、Migration Engineを加えない。

## 14. Ownerへの引渡しと今回の検証上限

PQ-A／Bの期待値は既存Driver Scenario SSOT #1180と独立Expected #2514へ返す。従来のXP/XR/XM/XA/XEを廃止・改番せず、必要な反例だけを各Ownerへ追加する。Source／Graph／文書は#2621／#2497／#1124、生成／Plugin／Builderは#951／#1121／#4544、DBと移行は#3270／#1174、業務状態は#1129／#2386、実行可否と閲覧は対象Host／Providerが判断する。

計画の成熟度は、モデルや文書の数でなく、**人が維持する定義の削減、誤った実装選択や古い成果物の拒否、局所再検証の精度、実行結果まで到達した割合**で評価する。特に同じMeaningでも実装選択を変えた時に旧版の再利用を止められるかを優先して測る。

**この§11〜14は現行Source／統合済みPRとの静的照合と設計上の実装引渡しであり、新しいGo・Windows・DB・MCP試験、Ownerの業務意味受理は本追補で実行していない。** 既存PRのマージや一部CI成功を、この追加計画のPASSとしない。
