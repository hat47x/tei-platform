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
