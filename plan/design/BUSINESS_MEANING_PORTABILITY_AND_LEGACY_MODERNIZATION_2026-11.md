# 業務意味の可搬性・連携Adapter・既存システム再実現の横断設計（2026-11-07）

- **状態**: 複数Issueを横断する**設計候補／未採用／新規E2E NOT RUN**。製品要求・正式API/DSL・Core型・Target実装の採用を宣言しない。
- **目的**: REST APIを必須中核にせず、同一の受理済み業務意味から、連携方式と実装・配備先を独立に選び、既存システムの意味抽出→再生成→段階的移行を反証可能にする。
- **既存正本**: [業務システム設計](../../docs/business-system-design.md)、[全体構成](../../docs/architecture.md)、[ADR-0043（Host非必須）](../adr/0043-target-native-artifacts-without-tei-host.md)、[Web API Adapter](WEB_READ_WRITE_API_ADAPTER_DESIGN.md)、[Target/Build構成](RUNTIME_TARGET_BINARY_COMPOSITION_DESIGN.md)、[ETL/ELT・MFT](ETL_ELT_MFT_CROSSBOUNDARY_CONTRACT_PRESSURE_2026-11.md)。本書はそれらの第二正本ではない。
- **Issueへの受渡し**: #1656（HTTP/OData）、#951/#1145（Capability/Invocation/Transport）、#2542（File/Message）、#2622（Data Provider）、#2555（旧システム並走とAuthority cutover）、#259（第二Target）、#1180（Scenario）、#2497/#2498（意味保存と外部形式）、#2831（Self-Build）。実装・受理担当は各IssueのOwnerを維持。

## 1. 判断の中心：API-firstではなくMeaning/Contract-first

Wagby 11等で有効な「REST APIを介して複数UIを接続する」構成は、TEIにおいても標準Web実現の有力候補。ただし**HTTPエンドポイントやDBテーブルをCanonicalな業務意味の正本にしない**。UI・API・File・MCPは、既存の型付きQuery/Action、Capability、Data Operations、Business Resultに対する別の入出力／公開／実行方式である。

```text
既存Source/実行観測 ──→ Source facts・推定・未解決 ──→ 意味の受理境界
                                                     ↓
                          Accepted Data / Function / Business meaning
                                                     ↓
                   Typed Query / Action / Capability / Data Operations
                           ├── HTTP/JSON REST（OpenAPIは公開仕様）
                           ├── OData（採用・対応範囲を明示）
                           ├── File / Message / ETL/ELT / MFT
                           ├── MCP（公開Toolと実行許可を分離）
                           └── Go binding / RPC / in-process
                                                     ↓
                   Target/Provider/Trust/Deploymentの明示Binding
                           ├── Go + Web / Wails Desktop
                           ├── Java / JVM既存環境
                           ├── .NET既存環境
                           └── その他の資格があるTarget
```

矢印は**論理的対応付け**であり、一つの常駐Server、HTTP通信、TEI共通Host、全方式の同梱、DB直接接続を要求しない。StudioのWails実証は生成された業務Desktopの第二Target資格に転用しない。

## 2. 混同しない独立軸

| 軸 | 選択または変化する要素 | 維持／検証する条件 |
| --- | --- | --- |
| Business meaning | Data、Function、Business、Intent、制約、必要なOperation | Owner受理を欠く推定をMeaningへ自動昇格しない |
| Published interface | REST/OpenAPI、OData、FileSpec、Message Schema、MCP Tool | 外部公開が許されたOperationとSchema・Version・Disclosure |
| Transport / delivery | HTTP、RPC、SFTP、MFT、MQ、ファイル配置、in-process | 送出・到着・可視化・取込・効果を別々に観測 |
| Runtime realization | Go、Java、.NET、既存ライブラリ、SQL、Wasm等 | Required Capability・Loss/Unsupported、実Binding |
| Deployment / trust | Web、Wails内蔵、別APIサーバー、既存APサーバー、Batch | Caller/Subject、現在Authority、Secret、Transaction、失効 |
| Source and evidence | 既存Source版、実観測、受理済み版、生成物版、Scenario | 同じ版と同じ試行の証拠で比較、混合世代を拒否 |

「RESTかODataか」は公開インターフェースの選択。「CSVか固定長か」はPayload形式。「SFTPかHULFT等か」は配送。「GoかJavaか」は実現言語。「二層か三層か」は配置・信頼境界であり、これらを一つの排他的な`appType` enumや巨大な`IntegrationMode`に閉じない。

ODataを全Targetの内部Query言語としない。OpenAPIをTEI全体の意味記述言語とせず、HTTP APIの公開仕様として扱う。File形式と転送方式も分離する。仮にWails一体型でGo serviceをin-process呼び出ししても、実行側が必要な権限を持たなければ副作用を許さない。

## 3. 「同じ動作」をどの範囲で要求するか

移行元コードの逐語・byte同一性は通常不要。ただし、業務契約の同等性は**対象Operation、入力集合、状態前提、実行主体、可視範囲、期待Outcome、必要な外部Effectと独立Observation**に相対化して比較する。

- **形式互換**: Schema、項目Presence/null/empty、桁・精度・単位、時刻、順序、重複、文字コード・外字等の保存性。型が同じだけでは通過しない。
- **動作互換**: 検証、分岐、業務拒否、競合、Transaction、再試行、Error/Unknown、確定効果、実読込まで含む。HTTP 200やFile ACKだけでは通過しない。
- **業務受理**: 旧仕様の不具合まで再現した「legacy-compatibility」と、Ownerが意図的に変更した「corrected-behavior」を別の受理対象とする。修正差分を「同等」と偽らない。
- **運用互換**: 配備・監視・復旧・認証・秘密情報・性能などはTarget/Environmentごとの別資格。同じ意味の生成に成功しても本番運用適合を主張しない。

互換性の判定を単一boolにせず、対象Consumerと観測条件を限定したClaimと未確認事項として扱う。受理済み意味の変更が必要なら、生成側の補正ではなくSource/Shared Acceptanceへ戻す。

## 4. 既存ソース→意味モデル→再生成：一方向の推論を受理へすり替えない

| 段階 | 入力・結果 | 停止条件 |
| --- | --- | --- |
| S0 Source inventory | Java/C#等のソース、設定、DDL/SQL、UI、FileSpec、OpenAPI/OData、依存Closureを版・出典付きで列挙 | 未取得Sourceを存在しないと断定しない |
| S1 Static facts | 型、Enum、Key/FD候補、呼出グラフ、SQL・画面項目、外部IO等を出典へ結ぶ | コード構造や命名から業務意図を確定しない |
| S2 Behavior evidence | 実Targetの入出力、更新、実DB変化、失敗・再試行・副作用を独立観測 | テスト通過と仕様受理を混同しない |
| S3 Candidate meaning | S1/S2を照合し、意図/既知不具合/疑義/情報欠落を分けた変更候補 | 推定をCanonicalへ無断投入しない |
| S4 Accepted meaning | Subject/Scope、変更理由、Owner、Source切替とrollback判断を受理 | 受理済みであることを現在の実行Authorityに流用しない |
| S5 Realization | 元言語・別言語/Target向けにBinding、Provider、生成Source/Artifactを選定 | 未対応Operationを削って成功扱いしない |
| S6 Differential qualification | 旧実装と生成実装を同一Scenario/独立Oracleで比較 | 相違を業務変更と互換故障に分け、未知の外部効果を自動replayしない |

S0〜S6は**研究用の比較手順**で、新しい恒久的Runtime Pipeline/Source SSOT/新Gateではない。Source Index、既存Parser、Scenario、Driver、Meaning/Acceptance、Source Authority Migration、Execution/Observationを再利用する。既存ソースをTEIへ取り込めたことと、正本の管理権限を移せたことは別。移行中の旧・新双方への書込みは #2555 のOwner/切替と既存CDC・rollback検証が必要。

最初は元のJava→TEI→Java、またはC#→TEI→C#という**同言語再生成**を評価し、言語差と意味抽出差を分離する。その後にJava/.NET/Goの異Target比較へ進む。ただし最初の実Consumerは既存`order_entry`を用いてよく、元言語再生成を全作業の前提にはしない。

## 5. 既存Fixtureに返す最小反例（新たな試験の第二正本を作らない）

| 反例 | 誤って成功と見なす状態 | 既存担当・観測 |
| --- | --- | --- |
| X1 | 同じOpenAPIでCommandは200だが異なる行へ作用 | #1656/#1174/#1180：Action target、Effect、独立Read |
| X2 | OData`$top`/Sortが非公開列に依存し、公開結果だけから秘密を推定可能 | #1656：行・項目認可、実Providerでの適用順 |
| X3 | 2-file Datasetの一方だけ更新・転送ACKあり、Consumerは混合版を読む | #2542/#2604：ArtifactSet、世代、Transfer、Importを分離 |
| X4 | 同じFileを再送し、Transfer Attempt IDをBusiness Operation IDとして再利用 | #2542/#1174：冪等の対象を分け、未確定Effectを観測 |
| X5 | Java/.NET生成物がNullable/Decimal/Date/Enumの意味を黙って変更 | #2497/#2498/#259：実Consumerの正負入力とTarget版 |
| X6 | 旧ソースにある例外動作をAIが「当然の仕様」としてCanonicalへ採用 | #1125/#2555：Source-static、Observed、Intended、Accepted |
| X7 | WailsがローカルDB接続できるのでユーザー認可を省略 | #259/#951/#1174：各副作用直前の現在Authority |
| X8 | 旧Sourceから新Sourceへ切替直後、旧画面/旧API/旧Fileで更新が成功 | #2555/#1124/#1180：版・主体・実Bindingの失効とrollback |

## 6. 優先順・Fit/Gap（2026-11-07時点の提案）

| 順 | 最小成果 | 再利用する既存Issue / 不足証拠 |
| --- | --- | --- |
| **P0-A** | `order_entry`の単一型付きQuery/ActionからWeb JSON APIと非HTTP呼出しの同じBusiness Result/拒否/独立Readを比較 | #1656/#1158/#1180/#1174。実同一Attemptの資格は未確認 |
| **P0-B** | 旧Sourceの小範囲についてSource-static・Observed・推定・Acceptedを分離し、移行対象のSource/Closure/Ownerを明示 | #1125/#2555/#2387。旧Target実測・人の意味受理は未確認 |
| **P1-A** | 同じ論理OperationとFileSpec/Manifestを結び、Transfer/ACK/Import/Business完了を別観測 | #2542/#2827/#2604。実MFT/取引先試験は未実施 |
| **P1-B** | 同一Meaning B/RからWeb Target T1と生成Wails Desktop T2を比較 | #259/#1180。Studio Wailsの証拠を転用しない |
| **P2-A** | Javaまたは.NETの同言語再生成→既存TargetとのOracle比較、次に異言語へ展開 | #259/#2498/#2555。実Targetと意味受理は未実施 |
| **P2-B** | 必要な範囲のOData規格適合、Application MCP、顧客環境向け配送/運用 | #1656/#2093/#2542。必要Consumerの証拠で選定 |

Self-Build #2831と重なるのは「同じ定義から必要なBuild/Artifact閉包を追跡する」点のみ。Build Artifactが存在することと、Web/Java/.NETの実動作、File到達、Applicationの業務受理は別資格。ETL専用DSL、万能Parser、共通Host、第二Resolver、専用Migration Authorityを先行追加しない。

## 7. 採否判断と成果物の取扱い

設計段階では、最初の小さなConsumer/Provider縦切りで以下を収集する。

1. 固定したSource/Accepted revision、Scenarioと独立Expected、Source Indexと用いた実Provider。
2. 必要なQuery/Action/Capabilityの一覧、公開したAdapter能力と非対応を含むCoverage。
3. 各Targetの実Artifact digest、依存閉包、実行条件・Caller/Authority、Observation。
4. 同等性が成立した対象範囲と負例、未解決、Adapter固有の追加コード量・実行時間・メモリの実測。
5. Candidate→Acceptedへ移す際のOwner判断、既存Sourceからの管理権限切替が必要なら移行/rollback条件。

採用候補の一般化は少なくともHTTPと非HTTPの異質Consumer、および異なるTargetまたは旧・新実装の差分観測が成立してから行う。必要なOperator/Mapping/ConformanceがTEI標準構造で表現できないと実証された箇所だけを既存Ownerへ返す。

**現状の主張上限**: 本書は各既存文書とIssueの静的照合に基づく横断設計であり、新たなGo/Java/.NET/Wails/HTTP/OData/MFT/実DB/実旧システム試験やOwnerによる意味受理を実施していない。

## 8. 旧実装・受理意味・新生成物の三角検収（2026-11-07追加、設計／NOT RUN）

旧実装と新生成物の結果一致を、業務上の正しさと同一視しない。「同じ動作」には**旧仕様との互換**と**受理済みMeaningへの適合**の二つの独立した問いがある。前者は移行元への差分実行、後者はOwnerが採用した規則・既存Scenarioの独立Expectedで判定する。実際の副作用と独立観測も別判定。これは既存[ScenarioのCross-route契約](CROSS_ROUTE_FUNCTIONAL_PARITY_AND_TEST_DERIVATION.md)の利用法であり、第二Scenario/Oracle SSOTではない。

| 限定した入力・権限・状態前提での判定 | 受理済みMeaningに適合 | 受理済みMeaningに不適合／不明 |
| --- | --- | --- |
| 旧実装と新生成物が同じ結果・Effect | **互換かつ適合の候補**。観測した範囲に限定して申告 | **旧実装の誤り・非受理規則の温存候補**。一致を理由に合格としない |
| 旧実装と新生成物が異なる結果・Effect | **意図的な修正の候補**。旧互換を破るため変更理由・Owner受理・移行方針が必要 | **変換欠落／不適合／観測不足の候補**。既定値で埋めず差戻し |
| 同じ条件で旧動作が観測できない | 比較不能。新側の適合だけを限定申告 | 比較不能・未検証。旧互換も適合も名乗らない |

「旧実装の正しい動作」を事後に生成AIが創作して独立Expectedを上書きしない。受理前の候補・Source-static・合成Fixture・実Provider観測の証拠種を分ける。旧/新双方が同じ観測結果でも、**観測していない業務規則・権限・副作用の同等性は導出できない**。

### 8.1 最小Witness（新しい永続Schemaではなく既存Evidenceの照合列）

- **Source**: 旧ソース実byte/digest・対象ファイルと依存閉包、DB DDL/Trigger/Procedure、設定・Version、必要に応じてfile encoding。Source Indexと[Source改版対応](SOURCE_REVISION_CORRESPONDENCE_AND_CONSUMER_IMPACT_2026-11.md)へリンク。パス・画面IDをCanonical Subject IDへ昇格させない。
- **Meaning**: 対象Subject/Scope、Accepted revision、どの旧Source factからどのIntent/Ruleを**推定したか**、受理/未受理/競合・Unknown。Source所有者の切替と意味の受理は別。
- **Operation**: 同じScenario revision、独立Expected、入力集合・NULL/欠損/空、時間・文字集合・Decimal精度・Enum値域、実Caller/Subject、適用した現行Policy/Resource Grant。権限同一の根拠が無ければ異なる条件の結果として扱う。
- **Realization**: 実使用したBuilder/Generator/Provider/Library/ABI/Runtime/DB/Target Bindingと依存閉包。Baseline B/Rと生成物Tlegacy/Tnew・digestを区別。単なる同じファイル名、同じLockの解決予定、CI build成功から実ロードを推定しない。
- **Observation**: Raw output、公開可能な結果、更新対象・Receipt、独立Read、外部受領と時点、Unknown、ログと例外（秘密情報を除く）。HTTP 200、MCP Tool成功、Wails戻値、MFT ACKを業務完了へ自動昇格しない。

同じ試行の照合列が埋まらなければ**not comparable / unqualified**とし、必要なSource・Target・Observerの欠落理由を記録する。SourceがA→Bに移動・編集されてもパス一致で継承せず、版間対応候補と受理・再検収を分ける。異なるHost/OSの改行・文字集合変換は形式/byte同一性と業務意味同一性の両方に圧力をかける（#2497/#2498）。

## 9. 旧ソース吸上げで見逃しやすい閉包と非可逆性（設計／NOT RUN）

最初に「解析できたファイル」ではなく「対象Operationの動作へ寄与し得るSource・外部依存の有限集合」を列挙する。入口（画面・API・Batch・File）からの実到達関係、SQL/ストアド/DB Trigger、非同期Consumer、設定・フラグ、Code Table、入力形式、時点・権限、エラー経路を区別する。解析器が認識できない動的呼出し、DB外効果、外部サービス側規則は**不在ではなく未解決**。全面解析・完全再現を先に要求せず、対象Operation/Consumerを限定する。

| Source由来の事象 | 吸上げの候補 | 無断でしてはならないこと |
| --- | --- | --- |
| SQL JOIN・CASE・集約・型変換 | Data/Functionの構造・Query lineage・導出規則の候補 | SQLの構造だけから業務Relation/Function Intentを確定する（#2387/#1126） |
| Java/C# Enum・定数・if分岐 | 閉じた値域・必須条件・状態遷移の候補 | 実装enumとMECEな業務分類が同義と推定する（#1126/強属性検討） |
| Null/default、丸め、タイムゾーン、Collation、外字 | 観測対象の表現・計算意味 | 同じJSON/DTO形状・Parser成功で往復意味保存とする（#2498） |
| DB Trigger、外部ファイル、Job、メッセージ | Hidden Effect、Delivery/Import、Business Eventの候補 | ストアド/External Consumer未観測のまま無副作用と報告する（#2542） |
| UI表示・APIレスポンス・Projection | 利用者操作の入口／開示された結果 | 表示から隠れたSource identity、対象集合、権限を復元したと主張する（#2497/#2516） |

非単射なProjectionから原データを一意に復元できない場合、AIによる「補完」をSource factに変えてはならない。必要なSource Reader/明示Bindingが無ければ不明のまま残す。旧画面や一つのAPIの正常応答だけでSource全体・Data/Function/Business三視点のCoverageが揃ったとしない。

## 10. 最初の有限な実証と移行時の停止線（既存Ownerへの受渡し）

**段階A：同一Accepted Meaning、異なる入口。** 既存`order_entry`から一つのQueryと一つのCommandを選ぶ。既存Driver Scenarioと独立Expectedをそのまま使い、GoのHTTP/JSON経路と非HTTP/in-processまたは既存CLI経路のResult/拒否/対象/独立Readを比較。未対応Operationをstubで通さず、必要な入口が未接続なら明示して停止（#1656/#1158/#1180）。File ExportはFileSpec/Manifestの独立資格、ODataは採用判断待ち、Generated Wails Desktopは第二Targetで個別資格（#259）。

**段階B：旧コードの一点抽出。** 旧JavaまたはC#の小さな関数/Queryについて、Sourceの静的事実と実観測を別収集する。特に(1)欠損/NULL/default、(2)Decimal丸め、(3)Enum範囲外、(4)外部Effect/例外の**正負対**を独立Oracleへ結ぶ。元言語→TEI→同言語の再生成を先に比較し、意味抽出誤差と言語変更誤差を区別する。旧実Targetや動作Oracleが得られないときはsource-static／proposed止まりとする（#2387/#2498/#2555）。

**段階C：安全な段階移行。** 旧と新の両方へ同じ変更系Operationを本番Mirroringしない。Read-only shadow/合成Snapshotの比較は可能だが、Writeは隔離された使い捨てProvider・独立Readに限定した検証から始める。並走・CDC・書込Ownerの切替、旧Clientの失効、Rollback/再同期は#2555の既存受理範囲に委譲する。実DBの接続・`dbcheck`成功だけでMigration Cutover、無副作用、全Consumer適合を名乗らない。

**最小の拒否ケース（新Test SSOTを作らない）**:

- **TRI-01**：旧と新の出力が一致するが独立Expectedには不適合（旧不具合を温存）→旧互換のみ候補、業務適合拒否。
- **TRI-02**：旧と新の出力が異なるが新は受理済みMeaningに適合→互換破壊と意図的修正を明示し、旧受入へ無断切替しない。
- **TRI-03**：表面上の結果は同じで、実更新対象または独立Readが異なる→互換不成立（#1174/#1180）。
- **TRI-04**：未公開列を使うSort/Count、欠落Dataの0件化、非可逆Projection後の復元推測→Consumerの開示/完全性と原Source Claimを分離（#1656/#2497）。
- **TRI-05**：Wails一体型でCaller/Grantを省略、または旧ClientがCutover後も更新できる→Build形態や「ローカル」をAuthorityの代用品にしない（#259/#2555）。
- **TRI-06**：旧Source依存の一つ（SQL Trigger/Job/設定）を読み落とし、新Targetへ未同梱→Source閉包/Artifact閉包/Target実消費の欠損として未資格（#2831/#2600）。

P0は段階Aの一つのQuery/Commandと段階BのSource-static/Observedの分離票まで。最初の反証で不足が証明されない限り、共通Runtime・巨大Parser・独自二重Generator/Scenario/Authority・新規Core kindの導入を保留する。既存の#2514独立Conformance、#1180 Scenario、#2604 Publication、#2555 Cutoverを集めて**一つの見通しのよい比較票**にするが、比較票自体をAcceptanceやE2Eの証明にはしない。

**現時点の結果上限**：TRI-01〜06、旧言語の同言語再生成、生成Wails Desktop、外部File/MFT/OData/実DBでの同一試行E2E、業務Owner受理はいずれも**NOT RUN／未受理**。既存Issue・資料にある限定Referenceの過去PASSと、本追補の新規受入は区別する。

## 11. 論理Subsystemを移行境界として使う場合の追加条件（2026-11-07、設計／未実証）

[サブシステム分割提案 #3250 / Draft #3255](https://github.com/hat47x/tei-platform/pull/3255)の「論理的なOwner・直接CUD・有効Effect・物理配置は別軸」を移行に利用する。**Subsystem候補の最適化、移行対象の選択、実行中の書込Authority切替は異なる意思決定**であり、分割候補が採用された時点で旧システムのDB書込を止めてはならない。#1132のread-only診断も再実装しない。

例：旧販売Functionが受注Orderを登録し、在庫Stockへ直接SQL UPDATEする構造を考える。新実現では販売Functionから在庫Ownerが管理する明示Commandへ委譲する設計候補を評価できる。しかしこれは単にtransportをHTTPに替える話ではない。Go/Wailsのin-process、Java/.NET内のmethod、HTTP、MQなど**どの実現形でも**、在庫を更新する主体と承認、順序、部分成功、再試行、回復、独立観測を示す必要がある。

```text
【旧実装】販売Function ── Direct U(Stock) ──→ 旧DB
       │                  （Source-static候補／実Effectは別）
       └── C(Order) ────────────────────────→ 旧DB

【新実現候補】販売Function ── C(Order) ──────→ 対象DB/Provider
       │
       └── Typed Intent / Capability Invocation
                     ↓  明示Binding・現在Grant
                 在庫Owner Command ── U(Stock) → 対象DB/Provider
                     ↓
                 Result / Receipt / independent Read
```

直接越境CUDがゼロになっても、委譲経路は依存として残す。もとの単一DB transactionを、別サービスへの呼出しに変更しただけではatomicityは保持されない。新Targetが同一プロセスでも実Providerの権限と失敗・観測境界を飛ばしてはならない。現行Targetが必要なtransaction/compensation/reconciliation能力を満たさなければ **incompatible / unresolved** とする。分割のために新しい業務手続・状態を暗黙に採用するのではなくShared Acceptanceへ戻す。

### 11.1 移行Sliceの有限な検収票（説明用・新Core型ではない）

| 固定するもの | 必要な観測 | 誤成功として拒否すること |
| --- | --- | --- |
| **意味と所属**: Subsystem候補、Data/Function/Business Subject、Accepted revision、Source closure | Owner・Read/CUD・Effective依存、未解析Edgeの母数 | CRUD表の直接U=0から実越境Effect=0を結論する |
| **操作と境界**: Order登録→Stock引当の型付きIntentと、現在の実Target/Grant | 旧direct Uと新Owner Commandの対象、時点、拒否、結果・必要な副作用 | 呼出成功、HTTP 2xx、Wails戻値から在庫確定を結論する |
| **確定と観測**: 同じ独立Scenario Expected、Commit/Receipt、別Read、通信断/重複 | 在庫と受注の必要な一貫性、不明/部分成功、独立Read | 原子的保証がないのに旧単一Txと同等と断言する |
| **Cutover**: 旧新のWrite Owner、Client cohort、CDC/逆同期、コード変換版・有効時点 | #2555が所有する切替/旧Writer失効・再開/rollbackの具体的Evidence | 分割提案やBuild成功をAuthority切替済みと扱う |

最初の比較は**read-onlyまたは使い捨てDB上で**旧・新両経路を独立に実行する。旧と新の本番副作用を二重送出しない。中間状態が異なる場合、受理済み業務契約が同等と見なす時間・観測範囲を明示し、無条件に結果一致へ縮約しない。#2555の既存P0 DB移行/Owner受理より前に新しいCutover Gateを作らない。

## 12. 抽出したTraitと各連携先の意味保存を分ける（設計／未実証）

[意味的特性の設計](SEMANTIC_TRAIT_CLAIMS_AND_CONFORMANCE_2026-11.md)の**Trait Claim / Source fact / 実装側Enforcement / Assessment**を再利用する。Java/C#のEnumや定数表、SQLのCHECK/UNIQUE、ファイルのコード体系、OData EDM、OpenAPI Schemaは、特性を調べるための別個のSource/Target証拠となる。**これらの記述が一致しても、それだけでMECE・強属性・業務キー・全Writerにわたる制約強制が受理されたとは言えない**。

例：Java `OrderStatus {NEW, ACTIVE, CLOSED}` に対して、DB列が `VARCHAR NOT NULL` で、旧ファイル経路から `HOLD` が書き込める可能性があるとする。Java Enumの3値だけを取り込んでTEIの業務値域を自動確定すると、ファイルを利用していたConsumerや旧レコードの意味が失われ得る。逆に未知値を無条件に`NEW`へ置換すると、欠落した検証・履歴・業務判断を隠す。Source-staticの3値、DBで保証されたNOT NULL、別Writerの実到達性、Ownerが意図した分類・有効時点は**別Claim**とする。

| 対象の変形 | 保存を検証する性質 | 必要な停止線・Owner |
| --- | --- | --- |
| 旧Java/C# → TEI → 同言語 | 値域、例外、Decimal/Date/Null、業務Identifierとその出典 | 意図未受理ならCandidate止まり。#2497/#2498、Trait #2495 |
| TEI → Go/Wails / Java / .NET | 同じTyped Operationと許可された結果・拒否、実行中のGrant/Provider | Target別実行・現在Authorityなしに同等性を宣言しない。#259/#1180 |
| TEI → REST/OData | 公開Read scope、Filter/Sort/Count、ErrorとClient Reaction、schema/metadata | 非公開列・未確定Effectを漏らさない。#1656／HTTP #3274 |
| TEI → File/ETL/MFT | Code Table有効時点、文字集合、Null/欠損、レコード順序、ACK/Import/Business Event | Delivery成功を業務Effectへ昇格しない。#2542/#2604 |
| 旧→新DB・並走 | 主キーと業務同一性、変換表の多重度と有効期間、CDC、更新のOwner | 未対応の多対多・曖昧値を推測変換しない。#2555 |

### 12.1 一つの変更で両方を見られる最小負例

- **MS-01（値域に対する閉包不足）**：Java Enumは3値、DBに値域CHECKはなく、別ファイルWriter未観測。`closed-domain`はSource-staticに留め、全業務閉値域・MECE PASSを主張しない。業務上`HOLD`を新値として採るか拒否するかはOwner判断。
- **MS-02（時点と変換表）**：同じ旧コード`A`が有効期間によって別の新コードへ対応する。ファイル取込時点と業務上の適用時点を明示せずに最新表へ変換し、過去再実行を一致と主張しない（#2555）。
- **MS-03（契約は同じ、配備が変化）**：在庫引当を同期Callから非同期Messageへ置換。公開Action IDと入力が同じでも、応答時点の確定性と再試行可能性は異なる。未確定の`accepted-to-queue`を`stock-reserved`としない（#1174/#2542）。
- **MS-04（非可逆Projection）**：OData/画面へ公開した列には元の区分・Source identityが欠落。これを元データの同等性や変更権限の証明に使わない（#2497/#2516）。

これらは#1180/既存独立Expected/各Ownerへの**既存Caseの差分条件**であり、新しいHarness、第二のTrait Engine、Migration Mapping Registry、全方式共通の意味論や正式仕様を増やす根拠ではない。必要な実Target・実Consumer・Owner/Policy評価が無ければ**NOT RUN / not qualified**と記録する。

**本追補の主張上限**：Subsystem分割、Source抽出、同言語再生成、Java/.NET/Go/Wailsの同一Scenario、REST/OData/File/MFT実連携、現行Grant・実DB・Cutoverの追加実証は行っていない。設計の責務接合と最小反証条件のみであり、[Draft #3255](https://github.com/hat47x/tei-platform/pull/3255)・#2555・Trait各Ownerの採用判断を代行しない。

## 13. 一つの業務Functionに対する再生成準備の有限判定（2026-11-08、設計／NOT RUN）

この節の独自の問いは、**現行TEIが何を実際に読み・解釈・受理・利用できたなら、特定Operationの旧→新再生成を資格ありと呼んでよいか**である。単なるReader成功・生成成功・2 Targetの値一致では足りない。他のOwnerがすでに持つRole→Type/Bindingの[ST-02/03/04/09](PLUGIN_DEPENDENCY_IO_MAPPING_DESIGN_2026-10.md)、[Design Context #3262](TEI_DESIGN_CONTEXT_TEST_HARNESS_INTEGRATION_2026-11.md)、[Trait履行 #3278](TRAIT_ATOMIC_COMPOSITION_AND_FULFILLMENT_2026-11.md)を、**対象Operationだけ**に絞って組み合わせる。追加の永続Schema/Readiness Engine/Scenario正本を作らない。

### 13.1 現行コードと未成立を分ける（mainとDraftの実装版を再照合）

**V0（統合済み）**：[PR #3248](https://github.com/hat47x/tei-platform/pull/3248) は `internal/validation/validation.go` の `attributeRoles` に税・為替・採番・休日の17 Role語彙を追加し、綴り誤りを拒否する。元の[語彙試験記録](../evidence/XTEA_LOWCODE_ROLE_VOCABULARY_2026-11.md)は**当該時点**の「Role→型検査未実装」の記録であり、後続mainの現在状況ではない。

**V1（統合済み・限定的）**：[PR #3308](https://github.com/hat47x/tei-platform/pull/3308) の `internal/validation/attribute_role_type.go` は、Roleが要求する `date/decimal/integer` と宣言TypeRefのAlias終端Primitiveを比較し、`SEMANTIC_ATTRIBUTE_ROLE_TYPE_MISMATCH` と `UNKNOWN_SEMANTIC_ATTRIBUTE_TYPE` を分離する。これは**静的Role→primitive照合**であり、値の精度・税率有効日、行データ、外部Writer/実DB、Plugin Publisherの真正性を保証しない。さらに2026-11-08確認mainの `primitiveFamily` はCanonical IDの**末尾文字列のみ**を比較するため、別Namespaceの `foreign.decimal` を同じ `decimal` と誤認する可能性がある。この安全側の修正と実Workspaceからの負例は担当の[Draft #3320](https://github.com/hat47x/tei-platform/pull/3320)で進行中。正本への統合済み修正として扱わない。

**V2（統合済み・限定的）**：[PR #3318](https://github.com/hat47x/tei-platform/pull/3318) は、`examples/xtea-lowcode/semantic/master/standard.tei` に税・為替・採番・休日の実Role宣言を追加し、**テスト内で投入した行データ**を既存Binderへ結合する正常系を確認した。ただし行データの実Provider取得・旧Runtime・異なるTarget/Consumerでの意味保存を実証していない。V2を「すべての型/Binding/Targetが検証された」とは解釈しない。

[Issue #3262](https://github.com/hat47x/tei-platform/issues/3262)と[Design Context試験計画](TEI_DESIGN_CONTEXT_TEST_HARNESS_INTEGRATION_2026-11.md)では、独立Required Ruleと実CLI Source Index/Inspect/Relationとの突合が進み、正常JSONと同じFingerprintでも必要なRuleが欠け得ることを判別対象にした。**限定Rule/Source Slice**の検出を、注文Functionの全Required閉包・現在のGrant・旧システムの全Writer・AI未知Rule発見へ昇格しない。旧版の「9 CLI Reply」だけを最新の総実証範囲と見なさず、各候補PRのexact HEADと統合mainで独立に資格化する。

### 13.2 対象限定Witnessの判別列（実Schema提案ではない）

| 判別列（Operation一つ・固定Scopeに限る） | 必要な根拠 | 欠落時の扱い／Owner |
| --- | --- | --- |
| **Required closure** | 対象OperationのData/Function/Business/Rule/Relation/Scenario/Sourceと外部Writer候補を、Sourceから独立に必要集合として列挙 | 必要集合が定義できない場合は `unknown-required`。#3262/#2823。空集合へ勝手に縮約しない |
| **Actually loaded & interpreted** | 実Source byte・Parser/Role version、実Loaded occurrence、意味として解釈を許すSource Binding | read-only結果に項目が見えたことは、全Sourceを読んだ証拠でない。#2620/#3030 |
| **Meaning qualification** | 属性Role・値域/時点/Unit・Semantic TypeRef、業務OwnerのAccepted revisionと未解決 | Role名を受理しただけなら `vocabulary-only`。#1223/#2495/#267。Owner受理に自動昇格しない |
| **Selected & actually consumed** | Target生成器の選択Provider/Plugin/Binding、実ロード/呼出、Source/Artifact digestと実Toolchain | 候補解決と実消費を混同しない。#951/#1121/#2831 |
| **Target obligation** | 必要なValidation/ConditionalWrite/Failure/Transaction/Recoveryを実Targetが満たす証拠 | Traitラベル・実装署名・Hintだけなら `unverified`。#3278/#259/#1174 |
| **Independent Scenario & observation** | 元コードと生成結果を別Oracleで検査したInput/Outcome/Effect、独立Read、現行Authority | 自己生成Expected・一つのHTTP 200・CSV ACKは不足。#1180/#2514/#2542/#2555 |

**判定規則の提案**：一つの必須列に `unknown/partial/unsupported` があれば、対象Operationについては **qualified migration / qualified regenerationを宣言しない**。別の用途では `read-only inventory`、`candidate generation`、`source-static analysis` の限定成果を記録してよい。すべての列が揃っても、Target別実権限・実外部Effect・Owner Cutoverの追加資格を自動的に付与しない。判別列を一つのBooleanや固定Core enumへ縮退させない。

### 13.3 もっとも小さい実験候補：注文の税額と旧ファイルWriter

**架空の業務変更例**：注文金額に税を適用する旧Java Functionと、旧固定長/CSV入力経路があると仮定する。意味上必要なものは、金額・通貨・税区分・税率の有効日・丸め規則・不明値の扱い・注文の書込対象。現実の旧ソース・実環境にこれらが存在すると主張するものではない。

| 比較段階 | 期待する限定成果 | 誤ったPASSの反例 |
| --- | --- | --- |
| **C0 Source inventory** | 旧Java/DB/ファイルScript、未観測Writer、必要な税Rate・日付・丸め規則の有無をSource位置付きで記す | Java側だけ検査し、他のWriterが未取得なのに「全入口で同じ税計算」と主張 |
| **C1 Role/Type meaning** | 統合済みV0 Role・V1 Role→primitive/unknown・V2合成row→Binderを**別々の証拠**に分解。Decimal Scale/Tax effective-time/業務Rule/Publisherは未資格 | Role名・Primitive末尾名・合成row Binder PASSから旧Writer/実Targetの税計算適合を宣言 |
| **C2 Generated target** | 同じAccepted Rule/Scenarioを固定してGo/Java/.NET候補へ明示Binding。使った実Source/Toolchain/Generated bytesを識別 | TypeRefの名前が同じだけでTarget実行・丸め同等性・Grant同等性を認定 |
| **C3 Cross-route consumer** | 非HTTP内部QueryとHTTP Read、File/ETL Consumerで同じ対象・時点・可視範囲の**必要な観測**を比較 | JSON/CSVのNULL、Decimal scale、順序/未認識コード、公開範囲の損失を無視 |
| **C4 Effect / cutover** | 隔離実DBで旧/新を独立実行し、Action対象・Receipt・独立Readを照合。旧Writer停止/権限切替は別Owner | 非同期送信受理を税額確定、または旧DB Write Owner切替済みと報告 |

**最初の実装可能な差分**は、新Parser・Providerや全Targetの生成ではない。#3262が進めるP-1b1の対象FunctionのRequired/Loaded/Parsed/Role/Meaningの欠損を、既存#1180のScenario候補・#1223のST-02/03および本書の三角検収へ**同じSource revisionと独立Expected**で受け渡すこと。少なくとも `vocabulary-only`、`primitive-static-only`、`same-suffix-foreign-primitive`、`required-source-unobserved` を混同せず判別できれば、次の実Consumerを選べる。旧Codeの本番書込みと生成Targetの本番二重実行はしない。

### 13.4 既存計画に対する優先順・失効

- **P0 (この横断レーン)**：#3262の **一Functionの必要集合／実Source閉包**と、統合済み#3248 **V0**・#3308 **V1（静的primitive）**・#3318 **V2（試験内row Binder）**の上限を一つの比較票に束ねる。別NamespaceのPrimitive末尾名の誤適合は#3320のDraftで個別修正し、未統合を安全と見なさない。業務Rule/精度/時点/全Writerは **unverified** に保つ。
- **P1**：#1223 ST-02/03、#2498の既存外部表現反例へ、Decimal/NULL/時点の一因子変更を渡し、二Consumerで保存/損失を別判定する。ファイル輸送・受領は#2542/#2604が担当。
- **P2**：同言語再生成→異Target比較、Wails ApplicationとStudio Hostを別資格、現行Grantと旧Writer失効を別検証。#259/#2555を主担当とし、このPRに新Driver/Runtime/Targetを実装しない。
- **変更時の失効**：Required Source/Schema/Role定義、業務Rule/Accepted revision、Plugin/Binder、Target/Artifact、Disclosure/Grant、実旧Writerやコード変換表の有効期間が変われば、影響するClaimを再評価。意味不変でもArtifactと実Target資格の再確認は必要。A→B→Aで旧RunのOwner/Grant/Effectを自動復活させない。

**証拠の上限**：#3248 V0、#3308 V1静的型照合、#3318 V2試験内row Binderはそれぞれ統合済み実装/既存試験証拠に帰属する。#3320は同名異Namespace負例と修正を提案するDraftで、採用/統合は未了。#3262の限定CLI/Scenario実測も同Issueの試験に帰属する。**本PRでは**C0〜C4のLegacy税額比較、実Providerの行取得、Java/.NET/Wails/HTTP/File異Consumer、全Source closure・Owner受理を実施していない。P0 Self-Build #2831・Studio #1124へのHard Dependencyは追加しない。

## 14. 変更から再資格化までの影響範囲：旧・新TargetとConsumerの差分引継ぎ（2026-11-08、設計／NOT RUN）

この節は第13節の**初回資格**に対する**変更後の再資格化**の補足である。変化したSourceを全件再パースするか、ビルドを再実行するかだけではなく、**どのClaimが現在も利用できるか**と**どのConsumerで再取得/再検証が必要か**を判別する。既存[Source改版追跡](SOURCE_REVISION_CORRESPONDENCE_AND_CONSUMER_IMPACT_2026-11.md)の版間対応候補と、#2582の依存伝播、#2600の増分Build、#2604の一式公開、#2516の開示判断を引き継ぐ。新しいImpact Engine・Evidence Store・正本となる変更IDは作らない。

### 14.1 再生成・再資格化・再公開・現在実行の四つを独立にする

```text
固定した受理Meaning / Source + 独立Required Operation Closure
             │                  （不足・Unknownを明記）
      ┌──────┴────────┐
      │               │
  変更前Witness   変更後Source / Role / Policy / Provider / Toolchain
      │               │
      └──────┬────────┘
             ↓ read-only 影響候補（確実な無影響・要再検査・未知）
  ┌──────────┼────────────┬─────────────┐
  ↓          ↓            ↓             ↓
Meaning     Build/      Publication/   Current Execution/
re-review   Artifact    Consumer Read  Effect/Grant/Owner
             ↓            ↓             ↓
          実消費入力      実Reader      実Invocation/独立Read
```

**保持してよいのは、該当Claimに必要な独立Required閉包が既知で、変更がその依存・意味・使用条件を変えないことを証拠で説明できる場合のみ**。Fingerprint一致や差分0行だけで「必要な依存が存在しない」とはしない。Unknownな依存が関連し得るなら局所再確認候補を残す。各Claimの以前のPASSは歴史として保持できるが、**現在の利用資格を自動復活させない**。

| 一因子だけ変える例（固定した一つのOrder Function） | Meaningの再評価 | Build/Artifactの再確認 | 公開・Consumerの再確認 | Current Grant/Effectの再確認 |
| --- | --- | --- | --- | --- |
| **Δ1 文書レイアウトのみ**。実Source/Meaning/依存は同一と独立に判明 | 原則不要候補。Source→Meaning対応が曖昧なら保留 | 表示Artifactのみ変更時に対象を絞る | 表示ConsumerのByte/Disclosureを再照合 | 書込を伴わなければ過去Effectを再実行しない |
| **Δ2 元Sourceの内容/位置変更**。旧ID・Roleの対応が多対多 | 版間候補をOwner判断へ。値/Schema/Meaning不変を推測しない | 実Builderが消費した対象のみ特定。必要閉包未知は保留 | SourceRef、旧リンク、公開Projectionを再確認 | 同一画面ID/Action名でも現在Owner/Grantを再照合 |
| **Δ3 未選択Plugin変更／Default Provider追加** | 意味変更は自動発生しない | 未選択が実Binaryへ混入していないか、選択が変わったかを別検査 | 実公開入口・Capabilityの差のみ確認 | Provider選択や現在Grantが変われば再資格化 |
| **Δ4 TargetのBuilder/Templateだけ更新** | Accepted Meaning自体の再受理は通常不要候補 | 生成byte・依存/実消費・Target適合を再評価 | Manifest/Readerの版・OS正規化を再検証 | 過去のReceiptを新Buildの実行実績に使わない |
| **Δ5 Disclosure/Grant・Recipient変更** | 業務意味は保持できる場合がある | 同byteなら再Build不要候補 | 閲覧範囲・受領宛先集合・開示資格を再評価 | **現在**の認可・主体/旧Client失効・効果を再照合 |
| **Δ6 A→B→A**。Source byte/Content ID/Artifact digestが以前と同じ | 意味の現在版・Owner Decisionの有効性を確認 | 古いBuild成果物の再利用可否は依存閉包で別判断 | 新しいAttemptのManifest/Reader/開示を再確認 | 以前のGrant/Receipt/Owner Cutover資格を再利用しない |

この表は無条件の「差分種類→再検査フラグ」の固定ルールではない。**実際のDependent Set、Target、Consumer、Authorityを既存Ownerから収集できる場合に限る条件付き候補**である。データは同じでも、業務時点・コード変換表・行集合・集計/順位や未公開列への依存が変われば別のClaimとして再評価する。内部変更の種類・件数・欠落理由を権限のないConsumerへ明かさない。

### 14.2 Subsystem境界と差分の独立性

[サブシステム分割 Draft #3255](https://github.com/hat47x/tei-platform/pull/3255)が区別する**所属だけの再割当**と**Owner境界修復（委譲Command/Function split/Business redesign）**を混ぜない。固定されたOwner/Direct CUDのもとで越境Updateの下界が残る場合、Subsystemの箱を配置し直すだけで越境Effectが消えたと主張しない。Command委譲を加えるなら、Meaning revision、Source Dependency、Provider/Transaction、部分成功・Recovery、旧WriterのCutoverはすべて影響候補。図のレイアウトのみの編集でBusiness Meaningを失効させるのも誤り。

### 14.3 既存Issueへ返す最小の反証と計測

**P0 — 1 Function / 1 Required Slice / 2 Consumer**：まず[Design Context #3262](TEI_DESIGN_CONTEXT_TEST_HARNESS_INTEGRATION_2026-11.md)の独立Required 3 Rule＋Relation/Source Anchorの限定資料を参照し、**必須Ruleが一つ消えても旧Fingerprintが同じ**場合と**必須Ruleは維持して表示順序だけ変える**場合を分ける。Source変更実行・実Rule修正は#3262のOwnerへ留め、このPRは結果とClaimの依存を照合する。

**P1 — 変更の有無ではなく影響の誤分類を測る**：既存#2505/#2582/#2600/#2604の各FixtureにΔ1〜Δ6のうち該当する一因子を対応させ、(a)必要なのに再検証しない **false retained**、(b)本来保持できる独立Claimまで再検証する **unnecessary invalidation**、(c)Dependency未知を無影響扱いする **unknown erased**、(d)Consumerごとの旧版混在/非開示漏洩、を**別々の母数**で報告する。分母が未知なら率は算出せず件数/未評価として残す。

**P2 — 本番を副作用で検証しない**：旧→新Target、HTTP/非HTTP、File/ETLの一つの実Operationで、実Builder入力、選択Provider/Grant、独立Expected、Publisher/Recipientの実Byte、Effect/Readback/Ownerを**同一試行に相対化して比較**する。ただし実副作用の重複検証は隔離DB/安全なread-only routeに限定。実MFT配送・外部H3 Tool Receiver・異Target Runtimeは既存Ownerの独立EvidenceなしにQualifiedとしない。

**閉包が分からない場合の正しい出口**は、「全変更不要」でも「全モデル確定」でもなく、対象Operation・Consumerの利用資格が未確認であることと、次に取得すべきSource/Binding/Authorityを明示すること。これを、生成AIの支援で計画・図表・テスト案を作る際にも保つ。セルフビルド#2831のStage1→Stage2実プロセス実証やStudio #1124の実CLI→Workbench接合には新たなHard Dependencyを増やさない。

**本節の証拠上限**：既存main/関連Issue・Draftの静的照合と、再資格化の比較票／計画追加。Δ1〜Δ6の実TEI試験、Legacy実行・Java/.NET/Go/Wails再生成、外部Consumer実受領、DB/Cutover/Owner受理は**NOT RUN**。他担当のCI 14/14や限定実CLI11 Replyを本PRの新実証と混同しない。

## 15. Role→型の実装進展を移行Witnessに接続する（2026-11-08、main実装照合／新試験NOT RUN）

前節の一般論を、**実際にmainにある一つのSource宣言**へ狭める。固定した公開例 `examples/xtea-lowcode/semantic/master/standard.tei` には、`master.tax_rate.rate` に相当する `TaxRate.rate type std.decimal role tax-value` が存在する（mainで読んだFile blob `292188047e7be360e31fec2abbda53cf3d9f0e2f`）。これは**Source宣言の実在**であって、旧Javaの税計算Source、税率行の実DB取得、当時の業務意味受理の証拠ではない。

### 15.1 同一の宣言に対する反証を区別する

| 一因子変更（必要な版とSubjectを固定） | 現行main／担当Draftが判定できること | 移行としてまだ保証できないこと・引継ぎ |
| --- | --- | --- |
| **R0 元宣言**：`std.decimal`＋`tax-value` | V0 Role受理、V1 Primitive一致、#3318の試験内行Binding正常という**各々別Evidence** | Decimal精度/Scale、適用期間、税額計算のTarget間一致、実Provider rowは未実証 |
| **R1**：TypeRefを `std.date` に変更 | 既存#3308は `SEMANTIC_ATTRIBUTE_ROLE_TYPE_MISMATCH` を要求。#3320は**実WorkspaceのWorld in-memory変化**を検査予定 | Product Source Editorでの実変更/再Load、実DB書込み拒否の証明ではない |
| **R2**：未解決TypeRefへ変更 | 既存#3308は `UNKNOWN_SEMANTIC_ATTRIBUTE_TYPE` への委譲を要求し、Role mismatchと二重報告しない | 「存在しないData」「外部から値が来ない」とは判定しない |
| **R3**：別Namespaceに `foreign.decimal` Primitiveを追加し、参照 | mainの `primitiveFamily` は末尾 `decimal` だけを返すため、合成IR上は**Role不一致を検出できない可能性**。#3320の完全修飾 `std.decimal` 比較が修正候補 | 実Source Loaderが外部Namespace宣言をどう読むか、Publisherが信頼可能か、正式互換性は別未実証 |
| **R4**：Domain Alias `master.rate_type` → `std.decimal` | 既存#3308はAlias終端を辿る。#3320案も標準Primitiveへ到達するAliasを許す | Alias名一致・Type構造一致で、丸め/Tax Unit/当時のOwner Decisionまで受理されない |
| **R5**：Role/Type/Sourceは同一、行だけNULL・精度・有効日・未認識コードを変更 | V0/V1では**値が対象外**、V2の正常系は試験内の限定rowのみ | #1223 ST-03／#2498／#2542で、Value/Provider/Consumer別に独立Expectedが必要 |

ここでの `foreign.decimal` は**名目一致を調べる合成IRの反例**であり、任意の外部型が現行Product Source/Hostに登録できることや、権限を迂回してDB更新できることを意味しない。さらに `std.decimal` の**文字列としての完全一致**も、Trusted Publisher、Semantic Source revision、現在のGrant、正式な外部互換契約を認定した証拠ではない。#3320での局所修正をMerge後の完全保証に読み替えない。

### 15.2 再生成計画に置く最小Witness（新しい永続形式ではない）

`Required(Accepted Operation, Scope, time)` を業務上必要なRule/Source/Role/型/外部Writer/出力/Effectとして**独立に固定**し、次の少数の観測を既存Evidenceへ結び付ける。

1. **Source/Role**：mainに存在する `standard.tei` の実byte、Source Index/実Loaded OccurrenceとRole/TypeRefの現在解決。R0〜R4の不一致を、#3308/#3320と#3262（Rule/Relation/Source）で別検査。
2. **Value/Meaning**：Tax Valueだけでなく適用開始/終了・税区分・丸め規則、入力のNULL/Presence/Scale、対象期間と受理済みMeaning revisionを固定。#3318のテスト内rowの正常系から**実Provider値まで資格を広げない**。
3. **実消費・独立Expected**：同じSubject/Source/Scopeの実Binder／Target選択・Builder/Artifact・Consumerを、独立に書いた期待値と照合。未利用Property/外部Writer/未解析RuleをUnknownとして残す。Source digest一致だけで実消費閉包を証明しない。
4. **異ConsumerとCurrent Effect**：まずread-only／隔離Fixtureで非HTTP、Web、File/ETLの必要なValue/Disclosure差を比較。物理Delivery/ACK、実DB Readback、旧Writerから新OwnerへのAuthority移管は#2542/#2604/#2555の既存Ownerで別確認。

**進捗に応じたExit**：V0＋V1＋V2の範囲は静的・試験内Referenceの段階まで進んだ。#3320のR3を独立負例で判別し、R5の**本物のSource/Provider行・二種類のConsumer**を扱うまでは「旧→新で業務税額を完全再実現」と言わない。#3320の現HEAD CI／最終統合資格は本書では再判定しない。Source/Operationが同一でもDisclosure/Grant/Toolchain/Targetが変わった時は§14のClaim別再資格化を利用する。

**責務と試験の上限**：R0はmain静的Sourceと既存PRのReported Evidence、R1/R2/R4は統合済みValidatorの試験意図、R3はmainコードの静的判定と他Owner Draftの修正候補、R5は未実証の追加ケース。本PRの実Go試験・旧Java/DB/OData/MFT・Target Generator・Owner正式受理は**NOT RUN**。新しいTrait Engine、Role Resolver、Migration Authority、全Target共通Host、第二Scenario/Oracleを設けない。
