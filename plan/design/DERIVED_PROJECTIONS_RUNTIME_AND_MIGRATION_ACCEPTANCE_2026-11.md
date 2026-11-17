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

## 15. 画面・API・一括取込で「更新の意思」を失わない（2026-11-15追補）

前節までは、Source／生成物／実利用の対応を中心に検討した。ここでは一つの業務項目を複数の入口から扱う際に、**表示された値、送信された値、書き換えたい値、書込みを許される値**を分ける。Data／Function／Businessの一次定義を増やさず、既存Data Operation・Function Context・Patch／Record Projection・Provider／Grantの契約へ対応付ける。

### 15.1 一つの項目が持つ四つの異なる状態

| 位置 | 保持する情報 | 同一視してはいけないもの |
| --- | --- | --- |
| **元DataとRead結果** | Field Identity、元Value／NULL／不存在、取得範囲と時点 | 欠損、非開示、表示上の空欄 |
| **Consumerへの開示** | Audience・Role・Scope、表示値／マスク／非表示 | マスク文字列と保存値、非表示と元データの不存在 |
| **Draft・入力UI** | 一時編集値、操作可能・入力不可・読取専用、未保存かどうか | Draft値と送信済みPatch、UI制御とサーバ認可 |
| **更新Intentと実Effect** | Field Presence、明示NULL／値指定／変更なし、ExpectedVersion、Current Grant、実Write Receiptと独立Readback | 送信省略とNULL、許可された候補と実Commit |

更新Intentの最小判別は**項目省略・明示的な変更なし・明示NULL・値指定**の四者とする。省略と変更なしは更新後の値が一致することがあるが、更新要求の経緯、登録時の扱い、既定値適用、入力検証にまで意味同一性を広げない。通常の空文字・空欄はInput Codecの規則に従ってから解釈し、NULLや変更なしと決め付けない。

UIの入力不可は、画面から当該項目を送らない挙動を実現できる。しかし任意のAPI／CLI／Batch Consumerに対する更新権限の拒否ではない。サーバ側は保存済みレコード状態、対象Key／Version、操作Actor、現在Grant、対象FieldとRuleを**実書込み前に再評価**する。画面を開いた時の編集可否や、検証のみの結果をそのまま権限として流用しない。

### 15.2 入口の表現を統一しても、同じ操作が行われたとは限らない

- **Web／Desktop**：表示中に値を変更した後、当該Fieldを入力不可にした場合、Draftは残り得るがPatchへ値を含めない。送信された値がないことを「NULLを送った」に変換しない。
- **REST／CLI／MCP**：同じData Fieldを操作する入口であっても、公開されたOperation、入力Schema、実Providerの更新方式と現在Grantを確認する。Tool Discoveryやread-only応答から書込みを生成しない。
- **CSV／TSV／XLSX・外部File**：raw byte、文字コード／BOM、列の有無、空欄、NULL指定、値保持指定、日付／数値の書式、複合キー解釈を順に確定してから型付きPatch候補を作る。Input Codecを経ない文字列値をそのまま業務Meaningにしない。
- **部分取得されたDocument**：秘匿Fieldや未取得Columnがあるとき、限定PATCHを提供するProviderと全件置換のみのProviderは同等ではない。後者へ現在見えている項目だけを送り全レコードを置換する操作を自動生成しない（既存Workbench WB-16、#2585）。

すべてのEntryが同じ要求を必ず実現できる必要はない。必要な更新能力・元Dataの完全性・対象Identity／Version・Current Grantがそろわなければ、実行可能でなく`unsupported`／`unverified`と診断する。

## 16. 検証・確定・採番同期とデータ移行の再開条件を分ける

一括取込では、**検証のみ・全件原子確定・正常行だけの確定**を別のCommit Policyとして維持する。検証結果のError CSVには元入力値や列位置が含まれ得るため、公開可能なエラー件数と、入力値を含む出力Artifactの現在Disclosureを別に判定する。部分確定後の再実行は行Operation ID／Receipt／Keyと現在DB状態を確認し、無条件のファイル再投入を禁止する。

移行用に元のIDを保存する操作と、移行先の**次の採番位置を安全な値へ進める操作**は、別の実行・回復単位である。DBへの行確定後に採番同期だけが不明／失敗なら、書込み済みの行を「なかったこと」にせず、独立Readと採番状態を照合して同期の再実行要否を判断する。通常の新規登録再開も、別の受入条件とする（既存データ取込DI-X05／#1130／#3270／#1174）。

入力FileのValidationが成功しても、その後に参照先・行Version・Grant・DB列型が変わる可能性がある。**反映は新しいAttempt**として再検証する。検証のみの外部副作用が実際にゼロであることは実Provider／Hookを確認して判定し、設計上のモード名から保証を推定しない。新しいImport Engine、全Provider共通のSequence ManagerやReceipt Storeは追加しない。

## 17. 小さな反証と実装への引渡し

| ID | 一因子差分と独立Expected | 不合格にする短絡／Owner |
| --- | --- | --- |
| **PW-01** | 保存値15、Draftで25へ変更後に入力不可。実送信は項目省略、Provider側の更新意図なし、独立Readは15。対照は明示NULLと値25 | 画面のDraft値を保存した／省略をNULLへ変換。#1129／#2585／#2500 |
| **PW-02** | 同じRecordを異なるScopeで開示。hidden／masked／NULL／未取得を区別し、マスク文字列の書戻しを拒否 | マスク表示＝実値、非表示＝不存在、表示できる＝書込める。#2516／#3262／#1174 |
| **PW-03** | 同じ入力列の「列なし・空欄・明示NULL・変更なし・値」を、選択Codecで型付き更新Intentへ変換 | Fileの空欄を一律NULL／変更なしにする、部分取得を全置換に使う。#1130／#2622／#2585 |
| **PW-04** | 検証のみが成功→参照先またはVersion／Grant変更→反映Attempt。更新前のDB状態と実Provider結果を再観測 | 古いValidationをCommit許可へ昇格。#1129／#1174／#1180 |
| **既存DI-X05** | 移行IDの行確定後、採番位置同期だけ失敗。行Receiptと次採番状態を別に判定 | ファイル全件再送／通常登録を無検査で再開。#1130／#3270／#2514 |

この表は**新しいScenario正本ではなく既存#1180／#2514の差分候補**。PW-01〜03の最初の一歩は、固定したRecord／Role／一Field／Codecを既存ReaderとOperationへ通し、送信前IntentとRead Disclosureの**read-only照合票**を作ることに限定する。実Write、全入口の同値、実DB Commitはその次の独立Provider試験とする。DI-X05は既存ケースを再利用し、試験番号を重複させない。

**P0への影響なし**：正式`*.tei.md` Reader→生成Go→実Builderの出典束縛（PQ-A）、Operator選択の鮮度（PQ-B）、通常CLI→実Providerの実Read、Studio Stage1→Stage2 Windows実セルフビルドを優先する。PWの広範な実Write適合、環境Migration、全文検索・帳票・MCP書込みをP0の新しいHard Gateにしない。

**検証上限**：§15〜17は既存計画・公開製品マニュアルから得た意味上の反例を一般化した設計追補である。ここで書いたケースの新Go／DB／Wails／API実証や、業務Ownerの意味受理は実施していない。

## 18. 確定・反映・利用を別々に検収する（2026-11-15横断追補）

§15〜17は入口ごとの更新Intentと実DBへの反映を分けた。さらに、**同じ保存済みDataから派生する検索結果・帳票・承認操作・通知が、同じ時点の状態を見ているとは限らない**。この差を新しい一律の「成功」「最新」状態に押し込まず、既存のData Context、Rule Phase、Process Instance、Effect Receipt、Query／Artifact Projection、現在Disclosureへ対応付ける。

### 18.1 一つの操作を、異なる資格を持つ八つの観測に分解する

| 観測位置 | 必要なWitness／入力 | そこからは言えないこと |
| --- | --- | --- |
| ① Source／実装選択 | Original Source、Meaning／Rule、Phase、Reader／Operator／Plugin選択・版 | 実Builderが当該実装を消費した、実Hostでロードされた |
| ② Read／開示 | Qualified Data／Relation、Scope、取得範囲・Snapshot・欠損、Audience・現在Grant | 全件取得、書込み可、別Audienceへも開示可 |
| ③ Draft／送信準備 | 全子集合の追加・削除予定を含む変更差分、Codec、Patch Presence | Preview計算がDBに保存済み、画面表示値が送信済み |
| ④ 書込み前判定 | 現DBの対象Key／Version、実行主体、Rule条件、許可されたEffect | 入力時の判定が現在も有効、実Commitに成功した |
| ⑤ Transaction内処理 | 依存Ruleの計算順、保存前後Hookの実Call、同一Tx内の変更・Outbox要求 | 「保存後」Hookの実行だけで耐久的なCommit済み、外部配送済み |
| ⑥ 確定と回復 | Commit Receipt、行／業務Operation ID、独立Readback、unknown／partial | Batch全件確定、外部API／Outbox配送、採番同期まで成功 |
| ⑦ 非同期Projection | 元Data Revisionと対象集合、索引水位、添付解析の完備性、Job／Template／Artifact版 | 検索可能＝最新の全件を索引化済み、帳票生成＝現在取得可能 |
| ⑧ Consumer利用 | 実ロードUI／API／Plugin／設定、取得時Audience・Grant、結果byte／受領観測 | 実行要求受付＝業務意味の受理、過去のGrantで現在も閲覧可 |

必要な時点とWitnessは**問いに応じて選ぶ**。すべてのData Rowへ一律に八種類のタイムスタンプや証明テーブルを作る提案ではない。存在しないProvider能力や未観測の段階は、ほかの段階の成功で補完せず、`unsupported`／`unknown`／`partial`を区別する。

### 18.2 RuleとProcess／ノード表現を分け過ぎず、混ぜ過ぎない

Ruleの意味定義は一つでよいが、**適用Phase・評価可能な入力・依存Read・副作用・有効な権限**はAttachmentごとに確かめる。入力中の親子再計算、DB保存前の制約、Transaction内の保存後処理、Commit後のOutbox配送、検索索引更新は別Phaseである。Transaction内Hookの「after」を「commit後」と解釈しない。Outboxが同一DB Transactionに参加できないProviderでは、原子的な通知・配送を保証するかのように表示しない。

業務Processの申請時Route Snapshot・必要承認人数・実績履歴と、組織の現在状態／委任／参照Grantも分ける。申請時に候補だった者が現在も承認できるとは限らず、代理操作では**委任元と実Actor**を別のIdentityとして記録する。Graph／Node View／プロセスフロー／標準UIはこの関係のProjectionであり、独立した承認状態Storeや第二Rule Engineを作る契機にはしない（#1129／#2386／#2497）。

索引や帳票の出力を扱うSUI／SHO／KON等のConsumerには、元Data Revision、取得Coverage、派生Artifact Revision、現在開示を投影する。ただし索引の準備完了、Job完了、Model Fingerprint一致を、元Dataの現行性、実ロード版の一致、取得時Grantの代用にしない。配備済みUI、REST、MCP、外部Hostの現在性も別々に観測する（#2250／#2600／#3267）。

## 19. 一つの受注例を固定し、必要な辺だけ反証する

**固定Fixture案**：複合キーの受注一件と子明細二件（明細A=200、B=300）、合計500、二つの画面タブ、保存時Rule一件。子Aを削除予定にするとDraft計算は300になる。別の操作ではその受注を申請し、承認候補3人・必要2人とする。帳票・検索索引は独立Consumerとして接続する。これは**合成ケースの提案**であり、現行TEI製品の一連の実走行ではない。

| ケース | 一因子差分と独立Expected | 誤った昇格／既存の担当 |
| --- | --- | --- |
| **EV-01 Draft範囲** | Aを削除予定にし、表示はBタブだけ。Draft合計300、DBには未確定。Commit時に全子行とKey／Versionを再検証 | 表示中の子だけを全件と見なす、Draft値を保存済みにする。#1129／#2585／#1180 |
| **EV-02 Tx後処理** | Transaction内の保存後Hookを意図的に失敗させる。同一TxのDB変更／Outbox要求が巻戻ったかを**独立Read**で確認する | Hook実Call＝Commit成功、DB Rollback＝外部配送も取消済み。#1129／#1174／#2514 |
| **EV-03 索引水位** | DBは受注改版済み、全文索引は旧版、添付解析は待機中。検索応答に版／Coverageの不足を残す | 検索0件⇒元DBに存在しない、索引が検索可能⇒添付本文も完備。#2622／#2516 |
| **EV-04 経路と現在資格** | 3候補中1人承認済み・必要2人。残る候補のGrantを失効させる。経路・過去実績を保存しつつ、現資格で進行可否を判定 | 申請時候補＝現在操作可、委任＝参照許可。#1129／#2386／#1174 |
| **EV-05 帳票利用** | 帳票Job完了後に元Data／Template／取得権限のうち一つを変更する。Artifact版・無効化・現在の取得結果を別々に照合 | Job成功＝取得成功・内容一致、旧Grantを無期限に流用。#1130／#1158／#2514 |
| **EV-06 実ロード版** | 同じMeaning Fingerprintのまま、生成済みPlugin／埋込UI／Configの実ロード版だけを変える | Model一致だけで実ConsumerのRule／画面／権限が同一と認定。#2250／#3267／#2831 |

EV-02は、Outboxを**同一DB Transactionへ投入できる実装を選んだ場合**の期待値であり、どのProviderにも同じ保証を仮定しない。EV-04は有効時点の資格を実際に再観測する条件付きのExpectedであり、時刻経過やDB直変更を全経路で自動検知できるとはみなさない。EV-05はTemplate変更・Data変更・Grant変更を同時に混ぜず、各々を独立した試行で検証する。

## 20. 実装計画への最小引渡しと停止線

**着手順は追加の大規模基盤開発ではなく、既存の二つの細い経路へ載せる。** まず§17のPW-01〜03（Read／Disclosure／Patch Intent）の固定一Fieldから、§18の①〜④を**read-onlyの時点別照合票**として既存Reader／Operationに接続する。Source位置・選択済みOperator／Provider・Scope・必要Readの閉包・現在Grantが不明なら実Write候補を発行しない。独立Expectedは#1180／#2514の既存Scenarioに置き、本書を別Scenario SSOTにしない。

次に、#2586の通常CLI一Function→**実Provider Read→Data Context→独立Consumer**の同じ対象で、EV-01（Draftと実Readの差）を一つだけ具体化する。実Write／DB Transactionが接続できた段階でEV-02に進み、非同期索引・帳票・申請／委任は各Ownerの実能力を確認してから順に増やす。#1251のPlugin、#951／#2831のセルフビルド、#3270のWorkbench、#2600のEvidence失効には**その実証で消費した辺だけ**を報告する。公開Design Context、Graph／Node View、SUI・SHO・KON、MCPは後続のConsumer候補であり、P0の追加必須前提としない。

**優先順位は不変更**：PQ-A（正式Source→選択Reader／Generator→生成Go→実Builderの出典束縛）、PQ-B（Operator選択A→B→旧Scenario失効）、#2586通常CLI実Read、#2831 Stage1→Stage2 Windows実セルフビルドを先行する。EV-01〜06は新しい判定段階や横断Hard Gateではない。

**今回の証拠上限**：§18〜20は既存設計・公開仕様記述の照合から得た受入計画であり、EVケースのGo／実DB／実OS／検索索引／帳票／承認／配送を新たに実行していない。外部仕様の読解をTEIの実Provider適合・Ownerの業務意味受理へ昇格させない。


## 21. 同じ定義をコード生成と汎用関数のどちらで実現するか（2026-11-15）

従来の比較では、モデルからGo／Java等のソースを生成する経路を主に扱ってきた。しかし、定義された表・図・Ruleのパラメータを既存の型付き汎用関数で解釈すれば、**生成コードを増やさずに業務動作を実現できる場合**もある。これを生成方式に対する例外や暫定実装と決め付けず、**同じ受理済み意味を実現する二つの方法**として比較する。

ここで共通化するのはData／Function／Business、型・Rule・Relation、適用Phaseと入出力・失敗・副作用の契約である。実現方法は、選択されたOperator／Generator／Providerと実行環境の組合せとして持ち、**モデル上の二重正本は作らない**。

| 判断する点 | 汎用関数による解釈 | Targetコード生成 |
| --- | --- | --- |
| 向く範囲 | 既存Operatorで意味を不足なく表せる純粋計算・検証、宣言的な表示や単純なData操作 | 固定Targetの最適化、複数言語への配布、複雑な業務制約、実行先に合わせた静的合成 |
| 更新と観測 | 定義・実行器の版、選択された実装、実Callと入力Scopeを毎回区別する | Reader・Generator版・生成byte・Builder入力・実ロードArtifactを区別する |
| 主な落とし穴 | 汎用関数にない意味を黙って近似する、未選択実装へのfallback、解釈器の現行版と定義版の不一致 | 生成結果が元Ruleを参照せず別実装へ委譲する、古い成果物の再利用、TargetごとのNULL・精度差 |
| 共通の検収 | 対象Identity・値／NULL／未取得・Read集合・結果・失敗・現在Scope | 同左。さらに原Source→生成物→実Builder／Consumerの資格を確認する |

**実現方式は業務意味でも、配布Packageの固定属性でもない。** 同じRuleをGoの生成コードで動かす場合とGo単一バイナリ内の型付き関数で解釈する場合がある。外部.NET／Java Hostを選ぶ場合も、Hostが受け取るBindingとその実行資格を別に検証する。コード生成・解釈のいずれも、業務上の副作用を実行するときは既存のOperation／Provider／Effect／Receiptと現在Grantを省略できない。

### 21.1 まず「型付きValidation一件」で可否を見極める

最初の比較対象は、製品Workspaceが選択した `tei.rule-authoring/v0` 由来の `sales.order_line.quantity_positive` とする。このRuleを同じ`semantic.Validation`から二通りに実行する。

1. **解釈経路**：既存`validationeval.Evaluate`が定義済みの比較を、合成Sourceの`Number`から読み取って実施する。
2. **生成経路**：既存`validationgen.GeneratePositive`が、判断条件を埋め込んだGoを生成し、実際にGoツールチェーンでビルド・実行する。生成処理は`validationeval.Evaluate`へ判断を丸投げしない。

実行結果だけでは同じ実装を二度呼んだことを見落とすため、生成Goの比較処理と、二経路で実際に呼ばれた`Source.Number`の対象・回数を観測する。正値、ゼロ、負値、明示NULL、値の未取得を同じ入力表で比較し、エラーを業務上の`not-satisfied`へ変換しない。基準となる期待値は実装出力から自動生成しない。

この比較は**形式が同じなら実現可能性まで同じだと証明するものではない**。現状、Go生成が対応するのは限定したRule Kind・型であり、未対応の指定を汎用関数へ黙って切り替えない。型・Phase・Read集合・依存Provider・Targetを含む適合は、対象ごとに確かめる。

## 22. 現行mainの実装を基線に、重複しない検収を進める

§21で選んだ一Ruleの二方式比較は、並行する実装で**すでに局所的なGo試験へ到達した**。同じ比較を別のGoテストとして増やすより、何が実際につながったかを確認し、残る別の辺を検収する。

- `cmd/tei/cli_self_build_generated_rule_go_test.go` の `TestSelectedRuleGoThroughRealCLISelfBuild`：製品のRule Readerから正値Ruleを選び、既存`validationgen.GeneratePositive`でGoを生成する。通常の`tei self-build`によるGo Binaryの出力を独立Expectedで確認した後、同じ型付きRuleを既存`validationeval.Evaluate`で評価し、正値・ゼロ・負値・NULL・未取得の五値と、対象FieldのRead回数を照合する。**生成Binary内の判断と、テスト側のEvaluatorが一つの実Provider／Contextを共有したという証拠ではない。**
- `tools/reference-generator-selection/selected_markdown_rule_rebind_test.go` の `TestSelectedRuleInterpretedAndGeneratedRealizationsAgree`：別のGenerator選択経路で同じ製品Ruleを読み、型・原文参照・生成されたGoを隔離Buildしたうえで、正値・小数・ゼロ・負値・負のゼロ・NULL・未取得・型不適合を比較する。JSON Adapterでの診断の粒度と、Evaluatorが評価不能として返すGo errorは別の観測である。
- `internal/semantic/validationgen/positive_product_test.go`：限定した正値Ruleについて、生成Goが比較・Field Readを実装し、単に既存Evaluatorへ委譲するだけではないことを検査する。生成成功はOwner意味受理を含まない。

これらは**互いに異なるReader／Generator／Build／Consumerの試行**である。片方の実Binaryで得た正値判定と、もう片方のGo Evaluatorの結果を合わせて、正式なP/G実装選択、現在Grant、実DB Provider、外部Hostの相互代替可能性まで認めてはいけない。並列した局所試験に合わせて独立のExpected Store、第二の実行選択器、類似のGoテストを追加しない。

**次の不足は実装選択の履行にある**。同じFunction Context／型・Snapshot／Phaseを固定した一Attemptで、選択されたP/G実現と実Called ID・Reader／Builder／Providerの資格を別々に照合する。選択GなのにPへ無断fallbackした例、選択Pなのに古い生成Binaryを呼んだ例は、出力値が一致しても失格とする。ここでの一Attemptは将来の実証条件であり、現行mainにそのRuntime選択機能が実装済みだとは主張しない。現時点の不足は`unknown`のまま既存#1180／#2514と#951／#1251／#2586へ渡す。

## 23. 次段階では「実現方式の差」だけを増やす

現行mainで確認した五値比較を基線として、同じCanonical Ruleのまま**一因子だけを変える**。新しいテスト正本は設けず、既存#1180／#2514の独立Expectedへ追加する。

| 比較候補 | 固定する条件／変更する一点 | 不合格にする挙動と既存Owner |
| --- | --- | --- |
| RV-01 | 入力Source／型／値を固定し、汎用関数と生成Goを選び替える | 結果が偶然同じでも実Called ID不明、Read先や例外の不一致（#2621／#1121／#2514） |
| RV-02 | 意味を変えず説明文だけ改版する | 原Sourceの旧位置・改版証拠を残したまま「再確認不要」とする。意味・Artifactの無関係な一律再ビルドも避ける（#2600／#2250） |
| RV-03 | Ruleの参照Attributeだけを変更し、旧生成物または旧解釈用Indexを再利用する | 旧SourceからのRead対象や旧判断を新定義の結果へ昇格（#2621／#2600） |
| RV-04 | Ruleの適用PhaseをDraftから保存前へ変更する | 同じ計算結果だけで、取得完全性、現在Grant、原子性と実DB保存を同じ資格にする（#1129／#1174／#2586） |
| RV-05 | 実装の対応型・Provider Capabilityが欠ける | 無断fallback、NULL／partialの丸め、対象外Read、Write可能なふりをする（#1251／#2622／#2516） |

**実装の順序**：今回のGo二経路比較を局所回帰として先行し、正式Rule Source→実BuilderのPQ-A、Operator実装選択と旧版失効のPQ-B、通常CLI→実Provider Read（#2586）へ結果を**別々の資格として**引き渡す。その後、同じFunctionへの接合が実際に成立した場合に限って、親子Draft、Transaction内Hook、Outbox、索引水位、帳票等のEV-01〜06を選択する。いずれもP0のStudio Desktop Stage1→Stage2セルフビルドを妨げる新しい必須条件にはしない。

判断基準は方式の好みではなく、**人が書く定義をどれだけ簡潔に保てるか、意味を失わずに実行できるか、生成／解釈の実装と版を正確に追跡できるか**である。計画の合意と実Provider／配備の検証を同じ達成報告へまとめない。

## 24. 同じ式を使っても評価場面が違えば同じ動作にはならない（2026-11-15追補）

§21〜23では一つの型付きRuleに対する解釈と生成を比較した。しかし、「同じRuleを実行できる」という事実から「どの入口・どの時点でも同じ結果を保存できる」とは言えない。ここでは他の設計で進めたRule Attachment、Data Operation、標準UI／API入口、SmartDB／Queryの議論を、実現方式の比較に接続する。

### 24.1 式・評価時点・保存先を別の選択として扱う

| 使用場面 | 評価に使う情報 | 処理の位置付け | 混同しないこと |
| --- | --- | --- | --- |
| **画面の初期候補** | その画面を初期化した時点の入力・利用者Context | UI Draftの初期値 | 入力後の自動再計算、API／Batchにも必ず設定される保存値 |
| **Draftの再計算** | 編集中の明細と削除予定を含む必要な全Scope | 未確定の候補 | 保存済みDB値、表示中タブだけでの全件性 |
| **登録・更新時の計算** | 現DB Version、送信Intent、Ruleが必要とするRead | 永続化前の検査・導出 | フォームを開いたときの古い値や判定、Commit済みReceipt |
| **表示時の導出** | 現在の参照値・Audience・取得時点 | 非保存のResult Projection | 元DBへ保存済み、DB側検索・ソートにも当然利用可能 |
| **検索・一括取込** | Query Predicate、Codec、Source Coverage、実Provider能力 | 独立したEntry／Data Operation | 画面初期候補と同じ自動補完、検索0件から元Data不存在の断定 |

これらは**異なる式言語を追加する理由にはならない**。共通の式やRuleが利用できるかを、Evaluation AttachmentごとのRequired Read／Context／Phase／Type／Persistence／Provider Capabilityで確かめる。例えば初期候補と保存時計算に同じ算術式を指定しても、適用時点が違えば値が異なることが正常な場合もある。対象時点の情報が揃わなければ「正しい値」と仮定せず、未検証のまま止める。

**実現方式の比較条件を補正する**：§21の汎用関数Pと生成コードGの同値性は、同じ型付きRule、同じAttachment、同じ入力Snapshot／Read集合、同じTarget精度・失敗契約を満たした**問いの範囲**でだけ評価する。同じGo述語を返すという局所試験から、フォーム初期化・保存・表示の全経路で動作が等しいと推論しない。AIが新しい実装を生成する場合も、入出力と既存Operatorの組合せを限定し、独立Expectedで反証できる定型処理から拡張する。

### 24.2 初期値・保存時Rule・APIは同じ「書込み機能」ではない

UI初期値は入力支援であり、保存時に必須な値の保証や、API／CLI／Batchの更新規則を置き換えない。複数入口に対して必須の導出がある場合は、既存Function／Data Ruleが保存時の責任を持つかを確認する。入口によって再計算を行わない場合は、その差を仕様として表示し、保存値を勝手に補完しない。

画面で入力したDraft、送信したPatch Presence、保存前の現在Version／Grant、実Provider Commit、独立Readbackを§15〜18と同じ順序で分離する。保存時Ruleを表示用Projectionの生成器へ流用するときは、逆方向のPutbackが一意に決まるかを別に検証する（#2500／#2585）。表示に成功した値を無条件に保存対象にする経路は認めない。

## 25. 参照値の保存方式と、参照経路の現在性を分ける

同じ商品・顧客などの関係から得た名称であっても、**取得時点の値を保存する**、**最終参照先のIdentityを保存する**、**表示時に現在の関係をたどる**の三方式は同じ意味ではない。新しい固定Core型を増やす前に、既存のData Field、Relation path、Persistence Mapping、Query／Derived Attribute、時点付きResult ProjectionとMigrationを用いて表せるかを検証する。

| 保持方式 | 永続化するもの | 参照先が変わったときの要点 |
| --- | --- | --- |
| 取得時点の値 | 確定時に取得した名称・金額等のSnapshot | 元マスターが更新されても過去の取引値は自動同期しない |
| 最終参照先のIdentity | Relationをたどって到達した実体のKey | 中間の経路が変わっても保存したKeyが自動的に付け替わるとは限らない |
| 現在の関係による導出 | 当該項目には値・Keyを保存しない | 取得時のRelationと現在Grantを使うため結果が変わり得る |

この区別は参照元と参照先の**Stable Identity／Qualified Relation／Occurrence／有効時点・観測時点**に結び付ける。特に「同じラベル」「同じ表示値」「同じ参照元Key」だけでは、保存済みの最終先Keyと現在の参照経路が一致するとは断定できない。Graph／ER／Node Viewは、その差を表示しても、保存値を暗黙に書き換えない。

保存方式を値からKeyへ変更する場合は、モデル属性の修正だけで完結させず、**既存格納値と正しいKeyの対応づけ、物理列型、移行前後の実Readback**を別票として検証する。表示値が偶然Keyに似ていても、意味上のKeyと認定しない。復元できない値・曖昧な対応・存在しない参照先は未解決として残し、変換後の参照に応じて現在の開示も判断する（#1121／#1174／#3270）。

## 26. 最初の受入例と実装順への接合

既存のScenario／独立Expected（#1180／#2514）への差分候補を、これまでのPW／EV／RVと重複しない**評価場面・保存方式の反例**に絞る。次の全件は現時点で**未実行**である。

| 反例 | 同じものを維持して変える一因子 | 見落としてはいけない結果 |
| --- | --- | --- |
| **EP-01** | 初期候補表示後に入力値だけ変更 | 初期候補は当時の値。自動再計算されるとの誤解を拒否 |
| **EP-02** | 同じDataへ画面以外のEntryから登録 | UI初期候補が存在してもAPIの保存値を保証しない。保存時Ruleの責務を別に評価 |
| **EP-03** | 非保存の表示用導出を検索・ソートへ要求 | Query／Providerの対応がなければ対応済みとしない |
| **EP-04** | 商品マスターの名称のみ更新 | 保存Snapshotは維持し、現在参照値だけが変わり得る |
| **EP-05** | 多段Relationの中間辺だけ変更 | 保存済み終端Keyと現在経路の終端を別Identityとして追跡 |
| **EP-06** | 参照値保存から終端Key保存へ方式変更 | 現Schema／履歴／実データのMigration受理がなければ適用不可 |

**実装順**：まず§21〜23のRule P/G比較を既存製品コード上で確認し、同一Ruleの適用Phase・Snapshot・Read集合が異なる場合の不一致を示すread-onlyの対照（EP-01）を追加する。次いで通常CLI一Function→実Provider Read（#2586）の受入経路で、EP-04／05の取得範囲・Identity・時点を説明する。EP-02／03／06の書込み・Query実対応・Migrationは、実ProviderのCapability／Owner受理が確認できたときだけ進める。

生成AIによるコード生成は、当面、**入力Shape・参照先・Phase・NULL／欠損・結果・副作用が厳密に指定できる狭い定型処理**から適用する。生成コードがボイラープレートとして確実に再現可能かを、生成器を入れ替える負例、実Buildと独立Expectedで測り、自然言語で似ているという理由だけで新しい業務判断を自動実装へ昇格させない。コード生成を省略した場合も、選択した汎用関数の版・実Call・適用Phaseを追跡する。

**現在の証拠上限**：本追補は既存の標準Data／Process／Rule／Query／移行計画と外部公開文書の比較を整理した設計案である。新しいGo／実DB／MCP／Windows／移行適用は実行していない。正式Source→BuilderのPQ-A、実装選択鮮度PQ-B、通常CLI実Provider、Studio Stage1→Stage2 WindowsセルフビルドというP0の順序を変更せず、新たなParser／Runtime／Registry／Host／Rule Engineを追加しない。

## 27. 生成メソッド名に依存しない「意味経路と状態」の検査（2026-11-16横断統合）

§21〜26で扱った同一Ruleの直接評価／Go生成、評価Phase、参照値の保存方式を、他チャットの「抽象レイヤーにおける経路検査」へ接続する。TargetやGeneratorの版によってGoのメソッド名・分割・呼出順序は変わり得る。テストを生成メソッド名へ直接結び付けると、業務意味が変わらない再生成で偽の失敗が生じる。逆に結果値だけを比較すると、選択されていない実装を通った誤動作まで合格する。

**検査の中心は、Entryから結果・Effectまでに必要な意味上の通過点、禁止する通過点、その点で守るべき状態不変条件である。** 内部の全Call Graphではなく、既存Semantic ID、Rule Attachment／Phase、Source Index、Read Scope、Provider／Builder／ConsumerのReceiptで観測できる境界に限る。新しいTrace Engine、独立Expected Store、固定的な経路DSLを先行導入しない。

### 27.1 ひとつの試行で照合する情報

| 検査面 | 必要な確認 | 誤った合格 |
| --- | --- | --- |
| Entry／選択 | Operation・RuleのCanonical ID、AttachmentとPhase、選択したP/G/委譲実現、版とCaller | 同名の画面操作・Go関数を通っただけで同じ経路と認める |
| 開始状態 | 入力値とPresence、Snapshot、必要Read集合・Coverage、Record Version、Audience／Grant | 画面表示値が同じなのでDB取得範囲や権限も同じと仮定 |
| 必須通過点 | 型付きRuleの実評価、対象Field／Relationの実Read、必要な次段への引渡し | Source宣言・ログ・生成コードの存在を実Callとみなす |
| 禁止通過点 | 選択外Field、旧実装、無許可Effect、非保存表示値の暗黙Write | 出力が一致すれば余計なRead/Writeも許す |
| 終了状態 | 判定・評価不能・Intent、該当時のみ実Commit／独立Readback／実Consumer | 戻り値をDB確定・外部配送・Owner受理に昇格 |

必須・禁止・未観測を区別する。外部Providerの内部通過点が見えない場合、実API呼出しと受領結果は証明できても、内部で特定Ruleを評価したとは言えない。その点はunknown／unsupportedのまま残す。禁止したEffectが「見えなかった」だけでは、Effect不存在の証明にもならない。

### 27.2 既存の実Go試験を重複させず、対象を広げる

最初の題材は既存の型付き正値Rule sales.order_line.quantity_positiveに固定する。現行mainのcmd/tei/cli_self_build_generated_rule_go_test.goとtools/reference-generator-selection/selected_markdown_rule_rebind_test.goで得たReader・生成Go・汎用評価の局所比較を再利用する。これらの実証は別々のBuild・Callerの範囲であり、同一Runtimeの実現選択が完全に接続したことを意味しない。

| ケース | 守る状態・抽象経路 | 確認状況 |
| --- | --- | --- |
| **AR-01 正値** | quantity=2.5、指定Rule→同一Quantityの一回Read→satisfied、選択外Readなし | pure RuleのRead／判定だけ既存Go試験で局所確認 |
| **AR-02 非正値** | 0・負値でも同一Field一回Read、not-satisfied | pure RuleのRead／判定だけ局所確認 |
| **AR-03 NULL／未取得** | 不明な値をfalseへ丸めず評価不能。保存成功へ短絡しない | pure Ruleのエラーまで局所確認。Commitは未実証 |
| **AR-04 実装選択変更** | 同じ結果でもG→P切替後の旧生成Binary呼出しを拒否 | 選択と実Callerの同一Attempt照合は未検収 |
| **AR-05 評価Phase変更** | 同じ式でもForm Init／Before Save／Displayの通過先と保存責任を区別 | 入り口とAttachmentの実証は未検収 |
| **AR-06 Relation変更** | 保存済み最終Keyと現在のRelation経路の終端を別Identityとして追跡 | 実DB Query／独立Readbackは未検収 |

§26のEP-01〜06は評価時点や保存方式の意味差を検証する題材であり、AR-01〜06はそれらを**どの通過点・状態で確かめるかという検査方法**である。既存#1180／#2514のScenarioと独立Expectedへ対応付け、第二のScenario正本は作らない。

### 27.3 実装へ落とす際の順序と停止線

**第一段階**：正値Ruleに限り、Rule／Field ID、Phase、Required Read、一回の評価と結果を既存Go試験からread-onlyの検収票へ投影する。Goメソッド名は追跡の補助情報であり、業務テストの安定IDではない。

**第二段階**：実Source→Generator→実Builderが消費したGo Byteと、Manifest／Operator選択A→Bに伴う旧Build資格失効を同一Attemptへ結ぶ（PQ-A/B、#1121／#4544／#2600）。結果が合っていても古い実装を呼べば不合格とする。

**第三段階**：通常CLI→実Provider Read→Function Context→独立Consumer（#2586）でScope／Snapshot／現在Grantを検査。Before Saveの判定、Commit Receipt、非同期配送を別の観測とし、結果判定だけでEffectの成立を主張しない（#1129／#1174）。

**第四段階**：Node／Diagram Occurrence、Port／Role、複合DataとAdapter、Wails／Desktopの実ロード版へ同じ観測を投影するが、同名ラベルを同一Meaning IDとみなさない（#951／#1251／#4198／#2831）。Studio Stage1→Stage2実Windowsセルフビルドは独立P0を維持する。

**停止条件**：対象EntryとRuleの対応不明、必須ReadのCoverage不足、実Caller／宣言Selection不一致、実Builder／Provider未観測、旧Snapshot／Grant、表示投影と実Writeの混同。未観測はnot-provenと明示し、「不実行を観測した」という負の証拠とも区別する。

本節は既存の実Go基線と研究をつなぐ検査設計である。新Goテスト・実Provider・Windows・正式Targetの追加実行を完了したものではなく、再生成可能な狭い処理から反証可能性を高める。

## 28. 抽象経路の検査を、既存実装へ渡す最小単位にする（2026-11-16）

§27ではメソッド名に依存しない通過点と状態不変条件を定義した。次は検査項目を増やすより、**既存の検証結果をどこまで受け継げるか**、**別の実装を選んだとき何を新たに証明する必要があるか**を、一つの試行の中で明確にする。ここでは同一Ruleの評価と生成、Operator実装の切替、型付きPort／Adapter、抽象Relationの検査を共通の検収方法に接続する。

### 28.1 検収票は三層に分ける

| 層 | 問い | 必要な証拠と停止線 |
| --- | --- | --- |
| **意味の同一性** | 何を実行することにしたか | Canonical ID、選択Reader／Role、Meaning Revision、Rule／Relation／Functionの型と適用Phase。図上の同名ラベルや生成Goシンボルだけでは対象を特定しない |
| **実現の履行** | 選んだ処理を本当に通ったか | Operator／Generator／Provider Selection、Sourceと実Parameter、Build Intent・実配置byte、実Call ID。IとGの値が同じでも、旧実装への到達は失格 |
| **状態の保持と受領** | 必要な値と資格を保って通ったか | Input Presence、Identity、粒度・多重度・Read Coverage、Snapshot、型変換、現在Grant、Result、必要時のCommit／Effect／Consumer Receipt。観測できない下流は未受理 |

「宣言が存在する」「実装候補がある」「Buildが通った」「実Callerが選んだ」「Consumerが受け取った」は別の段階である。各段階を省略して一枚の成功票にまとめない。ただし、これらを収容する新しい常設Evidence Storeも作らず、既存#1180／#2514のScenario、Source Index、Builder／Providerの証拠を**対象の問いに応じて参照**する。

### 28.2 既存成果から着手する二つの短い検証経路

**経路A：一Ruleの実現選択と旧成果物の失効。** 現行mainにある正値Ruleの製品Reader、`validationeval.Evaluate`、`validationgen.GeneratePositive`、通常`tei self-build`の限定試験を基線とする。正値・0・負値・NULL・未取得の結果およびRead回数を独立Expectedで確認したうえで、実装選択の鮮度だけを一因子で変える。

並行PR [#5114](https://github.com/hat47x/tei-platform/pull/5114)は、Operator実装A→Bの切替後に古い生成Scenario Command／Planを拒否する修正候補である。**PRがmainへ統合され、当該HEADの試験が確認できるまでは、現行mainの成立済み機能には数えない。** それが成立しても、「実装選択の旧Planを拒否できること」と「選択されたP/G方式が正式Targetで実際に呼ばれること」は別の資格である。

最初の実装Packetは、同じMeaningと独立Expectedを維持し、A選択→B選択→元のA再選択を異なる試行として扱う。古いCommand／Planの再利用を拒否し、再生成したものだけを新しい実装選択へ接続する。Source byte一致や出力一致で、古い実行許可を復活させない。実Builderが消費した入力、実Invoker、Runtimeでの選択結果を観測できない場合は、その境界を`not-proven`と記載する（#2600／#951／#1251）。

**経路B：一Relationの意味を保つ型付き接続。** 並行PR [#5115](https://github.com/hat47x/tei-platform/pull/5115)は組織C1の実Sourceから、Membership→OrgUnitのRelation ID・Key・向きとQueryの対象を検査する局所回帰の候補である。これもmain統合とHEADの確認前には実績へ算入しない。実際のRelation traversalや実Provider Readまで同じ経路を通過したとは主張しない。

ノード・Plugin間のPortをつなぐ場合も、型が一致するだけでは経路Bの意味が保たれない。**複合型全体、Fieldごとの接続、明示Adapter**のいずれであっても、Identity、粒度、多重度、値の欠損・NULL、必要集合の完全性、時点、公開範囲を確認する。必要な変換が未定義なら自動的に「似た型」を選ばず停止する。この検査は既存[Service Flowの型付き配線設計](SERVICE_FLOW_STANDARD_COMPOSITION_2026-11.md)と[#5121](https://github.com/hat47x/tei-platform/pull/5121)の検討を参照するもので、新しいCore Port型やGraph Runtimeを要求しない。

### 28.3 最初に拒否することを決める

| Case | 一因子の変更 | 期待する判定 | 既存の担当 |
| --- | --- | --- | --- |
| **AP-01** | 選択A→B、Meaning・出力結果は同じ | A時代の生成Command／Planを実行前に拒否。新Bの実Call証拠は別に確認 | #951／#2600／#1180 |
| **AP-02** | A→B→Aで選択が元に戻る | 過去Aの資格をByte一致だけで復活させない。新Attemptの実装選択を再確認 | #2595／#2600／#2514 |
| **AP-03** | Relation表示名だけ変更 | Canonical ID・Key・向き・多重度が変わらなければ業務意味は維持。元Sourceの変更資格は別に検査 | #1124／#2497／#2514 |
| **AP-04** | PortのField Mappingだけ変更 | 型だけでなくIdentity／grain／Coverageを照合し、必要なRead集合の欠落を拒否 | #951／#1251／#2622 |
| **AP-05** | 指定した実Callerだけ別実装へ変更 | 値が一致しても選択と実Callの不一致を拒否。実Call未観測なら未証明 | #4495／#1180／#2514 |
| **AP-06** | Save PhaseをDisplay Phaseへ変更 | 計算値が一致しても表示投影を永続化・Commit Receiptとして扱わない | #1129／#1174／#2586 |

APは新しい独立Expectedの体系ではなく、§27のAR、§26のEPを**製品Ownerへ渡すための試行差分**である。AR-01〜03で既に確認したpure Ruleの判定とRead回数を重複実装せず、AP-01〜06の未実証部分だけに試験を追加する。実装のMethod名や図表の見た目を固定するのではなく、同じ意味ID、必要な通過点、不変条件、実消費・実受領の証拠を固定する。

**優先順位**：最初にAP-01を既存Scenario修正で実証し、次にAP-05を通常CLIの実Provider／独立Consumerと接続する。AP-03／04はSource／型付き配線の局所検査で進め、AP-06の保存・表示の実Provider差はその後に扱う。正式`*.tei.md` Reader→実Builderの出典束縛とStudio Stage1→Stage2 Windowsセルフビルドを後回しにせず、追加Hard Gateも設けない。

**検証状態**：ここに追加したのは、現行mainの局所実Go基線、並行PRの提案、既存設計を区別して構成した実装Packetである。本節による新Go／PostgreSQL／Windows実行、正式Providerの採用、Ownerの意味受理は行っていない。


## 29. 業務アプリの定型生成を、保存・確定・利用の区別から検収する（2026-11-17）

### 29.1 設計対象は一つの画面ではなく、一つの業務操作である

設計情報から定型コードを安定して生成し、個別処理のみを拡張点へ閉じ込める方式は、TEIでも有効な到達目標である。ただし「コードの多くを生成できた」ことは、実際の業務操作の品質を保証しない。標準Data／Relation／Rule／Function／Process／UI Projectionと既存Plugin Contractを再利用し、まず**受注一件と複数明細の登録・更新**という単一の縦断例で、意味・入力・保存・効果・利用を区別して検証する。

- **要求の正本**：業務項目・複合キー／枝番・保存対象の親子集合・条件・認可・更新競合・結果の意味は、Ownerが受理するモデル側に置く。生成されたREST Route、Go関数名、Webフォームを業務意味の正本にしない。
- **標準で実現するもの**：入力検証、参照値の扱い、限定されたpure計算、操作対象の同定、一般的なCRUDと標準UIは、既存の型付きOperator／Data Operations／Target Generatorで実現を検討する。既存Generatorがない対象を「標準生成済み」とは呼ばない。
- **個別実装の境界**：定型範囲を超える処理は、意味ID、必要Data、実行Phase、現在Grant、許されるEffectを持つContractとして既存のProvider／Generator拡張へ渡す。実装候補や自然文Ruleだけで実Call成立・業務受理を主張しない。
- **実行形態**：Goの静的合成、既存汎用Evaluator、Target-native生成、外部Process Providerは同じ意味の異なる実現候補である。Web／Wails／CLI／MCPは入口の違い。特定WebフレームワークやREST ServerをCoreの必須Hostにはしない。

### 29.2 親子編集を保存処理へ渡すときの不変条件

| 観測対象 | 受入候補 | 混同してはならないもの |
| --- | --- | --- |
| 編集中の親子 | 変更前の各行Identity、追加／更新／削除の意思、未入力／NULL／未取得、表示中でない子集合の扱いを保持する | フォームで非表示＝子の削除、送信にない項目＝NULL更新 |
| 計算 | Draft／保存前／保存時／表示時の評価時点、集計対象集合、端数・欠損、依存順と保存先を明記する | 画面内の金額表示＝DBへ保存済み |
| 保存前判定 | 親と必要な全子の権限、入力Rule、複合Key、行Version、完全な更新対象を照合する | 型が一致＝対象が同じ、画面で操作可能＝現在Grantあり |
| DB更新 | 親・全子の変更、履歴／同一Transaction内の追加処理が一つのCommit結果へ結び付く。子失敗時に親だけ残らない | 子ごとのAPI成功＝親子全体の確定 |
| 保存後処理 | DB書込み後・Commit前のHookと、Commit後のメール／外部APIのEffectを区別する。外部Effectは必要時にOutbox・冪等Key・再試行Receiptで扱う | 「保存後」Hook実行＝Commit確定、DB Rollback＝送信取消 |
| 起動・配布 | 同一AttemptのSource、選択Generator／Provider、実Build Input、生成Artifact、実際のLoaded版を照合する | 改版履歴の追記＝実Binary入替え、Build成功＝稼働版更新 |

この表は新しい固定Workflow Runtimeや共通Hook総覧を要請するものではない。処理順と必要Effectは既存Process／Operation／Provider契約に表現し、Targetごとの実現可能性を検査する。保存要求の再送、Commit成功後の応答喪失、Outbox送信途中の失敗は、それぞれ異なる結果として扱う。

### 29.3 設計改版・実装整合・稼働版を三枚の証拠に分ける

設計の説明や改版履歴は、いつ何を意図して変えたかを人へ伝えるために有用である。一方、次の三つは独立して検査する。

1. **設計と実装の対応**：Model Revision、Rule／FunctionのCanonical ID、選択したOperator・Provider／Generator・実装Fingerprint、実装の欠落／重複／古さ。対応表を検査しても処理内容の正しさは保証しない（既存R2-P／R3に接続）。
2. **生成・配布されたもの**：Sourceの原byte、BuildPlan、Builderが実際に読んだ入力、Artifact Revision／Digest、Loaded Binary／UI／Config。設計履歴が新しくても、旧バイナリのままなら新仕様を提供したとは言えない。
3. **業務で観測されたもの**：実Operation、対象RecordとVersion、実Provider、現在Grant、DB Commitと独立Read、外部EffectとReceipt、Consumerが取得できた範囲。結果が偶然同じでも実装選択が違えば不合格にできる。

単一の「生成済み」「実装済み」「確認済み」印でこの三枚を代用しない。既存#2250／#2255、#1180／#2514、#2586、#2831へ引き渡す。

### 29.4 最初に追加する反例と実装順

| Case | 一因子の変更・失敗 | 期待する検収 | 接続先 |
| --- | --- | --- | --- |
| BAP-01 | 子一行だけ入力不正、親は正常 | 保存を拒否し、実DB独立Readで親・他の子が半端に変わらない | #2622／#1174／#2514 |
| BAP-02 | 既存子の削除意思と未送信子を入れ替える | 未送信を削除と誤認せず、対象Identityを維持 | #2497／#2506 |
| BAP-03 | 親が複合Key、子が全親Key＋枝番。片方のKeyだけ一致 | 別親の子へ付け替えず、誤った更新／削除を拒否 | #1124／#2622 |
| BAP-04 | Draft再計算は成功、保存前で値／権限／対象集合が変化 | 表示値を確定値と扱わず、保存Phaseで再判定 | #1129／#2586 |
| BAP-05 | DB保存後Hookが成功したが、Commitを失敗させる | 操作・履歴の部分確定を拒否。「Hook成功」をCommit Receiptとしない | #1174／#2514 |
| BAP-06 | Commit成功、応答喪失後に同じ要求を再送 | 業務キーと冪等Keyの役割を区別し、二重の履歴／外部要求を作らない | #1180／#1174 |
| BAP-07 | 設計版だけ更新、実Runtimeは旧Artifact | 改版履歴をもって配備済みとせず、実Loaded版不一致を通知 | #2250／#2831 |
| BAP-08 | 同じ結果を返す別Providerへ無断切替 | Result一致でも選択・実Call不一致を拒否 | #951／#2586 |

**着手順**：①現行mainの正値Rule／生成Go／Evaluator検証を基線としてBAP-02／04のread-onlyな対象・Phase差を既存Fixtureで確認する。②P0の正式Reader→実Builderと通常CLI→実Provider／独立Consumerを進め、隔離DB上でBAP-01／03を接続する。③同じ保存OperationにCommit・Receiptが乗った後でBAP-05／06を扱う。④BAP-07をStudio Stage1→Stage2の実Windows試行に接続する。⑤承認／代理承認・組織ロール・現在Grantは、既存組織C1・BRMS／Case・AuthZのOwner受理を待ち、汎用状態表や別ワークフローエンジンを先行導入しない。

BAPは**独立Expectedの新しい正本ではなく既存Scenarioへ渡す候補**である。本追補は設計整理だけであり、新たなGo／実DB／Windows試験、Ownerによる業務意味受理、Wails Stage2セルフビルドの完了を表すものではない。P0の順序と既存Ownerを変えず、新しいHard Gateも追加しない。
