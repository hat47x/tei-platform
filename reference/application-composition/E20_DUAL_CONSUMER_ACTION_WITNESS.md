# E20：受注明細とSUI候補を同じAction接続口で比較する

**対象：** [Issue #1016](https://github.com/hat47x/tei-platform/issues/1016)。これは構成機構の**読み取りと局所回帰試験**であり、SUIアプリケーションがTEIで実行できた証拠ではない。

## 既存の正本と今回の検証

| 関心事 | 受注明細（Web側） | SUI Native Canvas側 |
| --- | --- | --- |
| ApplicationとCommand | 実`reference/workspace`に定義された`order_entry.submit_order`。既存`driver.NewResolver`が実Capability `sales.order.submit`を解決する | 現状の`reference/sui-adoption`は意味定義の一部のみ。TEI Applicationはない。試験内だけの`sui.move`候補として解決する |
| 操作発火 | 受注明細の確定済みActionをHostへ | Native Canvasの局所プレビュー／取消しはSUIが所有。**確定済みMove Intentのみ**をHostへ送ると仮定。実ポインター操作は未試験 |
| Payloadの意味・保存 | `order_entry`の所有する記録用Store（試験専用） | 既存`actionboundary.NewScopedJSONDocumentStore`（SUI形状のメモリ模擬）。正式なSUI DocumentV1処理・SQLite/PostgreSQLへ接続しない |
| 共有Host境界 | 既存の`actionboundary.Host`・`CanonicalCommandGate`・`JSONCommitHandler` | 同じHost・Handlerを使用。Application／Actionの組、Resourceと期待Revision、実行時Authorizationを別々に確認 |
| 本番認証・送信先 | 試験用Authorizerのみ。実セッションのGrantなし | 同左。実SUIへの委譲は[#993](https://github.com/hat47x/tei-platform/issues/993)と[PR #1329](https://github.com/hat47x/tei-platform/pull/1329)が所有し、ここでは変更しない |

[Go回帰試験](e20_dual_consumer_action_test.go)は、**一つのHostに二つのApplication／Action Binding**を登録し、標準のHTTP JSON Action Envelopeからそれぞれ別のStoreへ一度ずつ書き込む。受注明細は実WorkspaceのCommandを解決するが、SUIは未定義のため**試験限定の候補Command**であることを厳密に区別する。

### 必ず保持する拒否条件

- 一方のApplication IDで他方のActionを呼んでもBindingやPayloadの文法をすり替えない。
- 他Document／他OrderのResourceに書き込まない。
- 期待Revisionが古い場合はStoreに書かない。
- 実行時のAuthorization拒否・Authorizer欠落を、成功または無変化という曖昧な正常結果に変換しない。
- 不正なPayloadを拒否し、成功したSUI Card座標更新でもSource、Hold、Review、未知Edgeなどを保存する。

## 構成を二重化せずに得られた比較結果

共通化するのは**接続に必要なAction EnvelopeとHostへのBinding**だけである。Domain Payload、Canonical Commandの正本、Resource/Revisionの同一性、Storage、実行時のAuthorityは案件側／既存機構へ残す。UI Kitの構成やTarget自動選択、Providerの候補優先順位をこの試験で決めない。

試験に必要な構成選択として、人間または生成AIが確認するべき点は次のとおり。

| 選択 | Web `order_entry` | SUI Native |
| --- | --- | --- |
| Command定義・Capability | 既存Workspaceから解決済み | SUI Stage 0には存在せず、実TEI Applicationを定義するか未決定 |
| Store Adapter | 試験の記録用Store（実Provider未接続） | SUI既存保存経路の採用と、Tenant・Revision・CASの責務境界が未確定 |
| 操作境界 | 既存Commandへの確定送信 | Pointer preview/cancelの間は送信せず、Dropで1回だけ送信。**SUIの実React Canvas接続は未確認** |
| Runtime Grant | 模擬の許可／拒否のみ | 実セッション認可と委譲Token／Audience／Resource検証が必要 |
| Framework/TargetとUI Kit | 現在のWeb/React Reference | SUI CanvasをNative Assetとして保持し、標準Formへ移植しない |

**追加した製品側の依存解決機構：0件**。追加したのは参照用Goテストと、この検証資料だけである。実BrowserとSUI Backendを接続するための編集箇所・手動接続工程・AIへの追加質問数は**未測定**。この参照試験の配線量を実導入コストとして報告しない。

受入判定は「既存の共通Action Hostが二つの異なるPayload・Storeに対応できた」に限定する。「同一Composition Profileで二つの実Applicationを稼働させた」「現時点のユーザーがSUIを操作できる」「本番安全性を満たした」とは判断しない。
