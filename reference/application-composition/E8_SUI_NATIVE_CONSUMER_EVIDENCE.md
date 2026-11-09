# E8：SUI Native Canvasを第二Consumerとして確認した結果

- **状態**：両リポジトリの**既存ソースを読んだ評価**。SUIの実行、TEI Runtimeとの結線、Production採用の証拠ではない。
- **管理**：[Issue #1016](https://github.com/hat47x/tei-platform/issues/1016)、SUI機構 [#993](https://github.com/hat47x/tei-platform/issues/993)。既存の[Native Gesture→Application Action Reference](../../tools/reference-sui-interaction/README.md)を再利用する。
- **SUI確認基点**：`hat47x/sui-sensemaking` の`main`、`5cb767e64e3a1bca1e406f6b772f06db1b122108`（2026-10-10）。以下のGit blobは対象ファイルごとの内容識別であり、以降の変更へ自動追従する契約ではない。

## 実Consumerの現状

| SUI Source / blob SHA | 実ソースで確認した責務 | TEI案件構成が保持すべき境界 |
| --- | --- | --- |
| `03_Implement/frontend/src/canvas/CardView.tsx` / `ae0051131264d9b01f6ff26fb8341edce493b211` | `onCommitMove`が提供された場合、pointermove中はローカルプレビューだけ。pointerupで最終差分を一度渡し、pointercancelではローカル状態を解除。従来`onMove`経路も残る | Cardのレンダリングとドラッグ途中の処理はNative UIが所有。pointermove単位のTEI RPCを要求しない |
| `03_Implement/frontend/src/canvas/CanvasShell.tsx` / `e15bed9c2451c28be09bf4cb237e78c0bf44927c` | `onCardMoveBegin`／`onCardMoveCommit`をCardViewへ明示接続し、Screen座標差分をZoomでWorld座標へ変換 | Canvas座標系はUI固有。汎用Coreへ強制移植しない |
| `03_Implement/frontend/src/App.tsx` / `1ae4a3d74a12bab33ab2e8a50cfee0d284166324` | begin時のDocument/ETagを保持し、commit時に`isCardDragOriginCurrent`を検査。更新を`applyDocumentChange`へ渡し、`CanvasShell`へ両callbackを設定 | **現在の移動CommitはSUI Application内**。TEI Actionへ接続済みとは言えない |
| `03_Implement/frontend/src/domain/card_drag_commit.ts` / `0a7761973b7ca2bf066e5f91bda20da37cb71f72` | 座標差分・Grid Snap・Islandへの所属変更をSUIのDocumentV1上で扱う。古いDocument/ETagなら適用しない | Domainの整合とDocument形状はSUIに保持。TEIが第二のCardモデルを作らない |
| `03_Implement/frontend/src/api/client.ts` / `e614aa83363df69333e6b2a23cf1c1386ecce012` | `PUT /docs/{docId}`でETagがあれば`If-Match`を設定し、Tenant Session Contextによっては`If-None-Match: *`を設定。引数によってはどちらも付けず送信できる | **保存時の競合・認証・Tenant境界は別途実運用で検証**。ローカル移動Commitと永続化CASは別 |
| `03_Implement/frontend/src/domain/card_drag_commit.test.ts` / `69ee68b3439630b480a5ab9ccb95c9f744c13955` | 古いoriginの拒否、位置・Island操作、Grid Snapの単体試験が存在 | テストファイルの存在だけではこの時点のTest PASSを主張しない |

TEI側の`reference/sui-adoption/workspace/tei.workspace.yaml`は`semantic`だけをSource rootとし、`semantic/sui/island.tei`に一部分の階層関係を置く。**Application定義、Canvas Host、Native Action・保存Providerまでを備える第二Consumerではない**。したがって、E5の`workspace.Load`でSUI Stage 0から正式なApplicationを一つ得られると期待するのは誤りである。

## E1〜E7の結論に加える修正

以前のReference評価ではSUIのpointermoveがDocument更新を起こすこと、pointercancelが移動を確定してしまうことを残課題としていた。しかし2026-10-10時点のSUIの`CardView`・`CanvasShell`・`App`には、**opt-inのNative preview／一回Commit経路がソースとして存在する**。これを未実装の機能として改めて生成すると、正本の競合と二重実装になる。

ただし、Native Gesture分離があるというだけでは、`App`の更新経路がTEI Runtime Actionへ渡されたわけでも、SUIの保存処理がTEI側の認可・Revision CASを満たすわけでもない。

## 案件構成の追加宣言として必要なもの

この実Consumerに対し、新しい共通Pluginを先に作るよりも、次の**接続選択が未確定**であることを残す。

| 選択・境界 | 既に存在する正本 | 案件側でまだ要る判断 |
| --- | --- | --- |
| Native UI / Domainの所有者 | SUIのReact Canvas／DocumentV1／domain functions | SUIをNative Assetとして採用するか。変更時に何をSUI側へ戻すか |
| 最終操作のAction接続 | SUIの`onCardMoveCommit`→`applyDocumentChange`、TEIの`tools/reference-sui-interaction` | 実TEI Runtimeへ接続する場合の**一度きりのcommit境界**とAdapter |
| 保存・競合・権限 | SUIのDoc API、ETag、Tenant Session、TEIのCapability/Authority機構 | Hostの現時点の権限・Document revision・CASを誰が担保するか。構成だけでGrantしない |
| Source変更の再検証 | 各Git blob、TEIのSource Index、[E7aの限定証跡](E7A_MINIMAL_DISCLOSURE.md) | 別リポジトリのNative資産はTEI Workspace Indexだけで網羅できない。出典・採用対象を明示する |
| UI Kit / Framework | SUIの実Native Canvas、TEI UI Kit #997 | 全Canvasを標準Formへ置換せず、公開接続点だけ共通化するか |

## 次の受入試験

1. SUIの実Frontendで、pointerdown→複数pointermove→pointercancel時はDocumentが不変、pointerup時だけ一つのDocument変更とUndoになることを確認する。Grid SnapとIsland所属も保持する。
2. SUIの現実の保存先で、期待Revisionの付与と競合時の拒否、権限失効時の拒否を検証する。局所SUI Commitは永続化の完了とは区別する。
3. TEI接続用薄いAdapterでNative Canvasの最終Intentのみを引き渡し、既存のAction・Permission／Resource・Providerの正本を再利用する。ドメイン意味を新たなTEI Core enumとして追加しない。
4. 変更したファイル、共有契約、重複した定義、必要なAIへの追加質問数、実行までの手動接続工程を`order_entry`と比較する。

以上の実証ができるまで、**同じApplication Composition Profileで二つの実Consumerが正式に動いたとは主張しない**。新しいDSL／Resolverや独立Lockの採用判断を前倒ししない。
