# SUIの操作をTEI Actionへ接続する参照実装

この実装は、非業務アプリケーションの独自UIをTEI ApplicationのActionへ接続する際の境界を検証するものです。**本番用のCanvas接続、SUIの移行、永続化、実セッション認可は実装していません。**

## 所有する責務

| 関心事 | 所有者・現状 |
| --- | --- |
| ポインター操作、選択、ドラッグ中の表示、取消し | SUIの既存React Canvas。TEIには未接続 |
| ApplicationとActionの対応 | 参照用`Host` |
| 宣言済みCommandの解決 | `CanonicalCommandGate`。`driver.Resolver.ResolvePattern`へ接続可能 |
| 実行時の認可 | `CapabilityAuthorizer`。既存`security.authorize`を使用するが、実セッションは未接続 |
| Documentの識別、Revision照合、更新 | アプリケーション所有の`Store`。今回の実装はメモリ内の模擬保存 |
| SUIのDocumentV1、島への所属、履歴、SafeMode | SUIの既存実装。TEIには移植しない |

`Host.Commit`はApplication ID、**Resource ID**、Action ID、期待Revision、アプリケーション固有のPayloadを受け取ります。更新前にActionの対応と認可を確認し、保存側へ委譲します。**Hostだけでは同時更新を防げません。** Revisionの原子的な照合と更新はStoreの責務です。

## SUIの実コードとの対応

SUI `03_Implement/frontend/src/domain/types.ts` の`DocumentV1`では、Cardの座標は`card.x`、`card.y`です。`card.position`という構造ではありません。参照用`JSONDocumentStore`はこの座標形式に合わせ、`version: 1`と文書本体の`id`を確認します。Sourceは`card.meta.source`、保留は`card.holdState`、レビューは`reviewAttribution`などを保持します。未知のEdge種別も移動によって変更しません。

ただし、参照用StoreはSUIのTypeScript検証器を呼び出さず、島への自動所属やUndo/Redoも再現しません。構造が似ていることと、SUIのドメイン処理に接続したことを混同しないでください。

**旧SUI実装で確認した差分**（2026-10-10に確認した旧経路。PR #3222の新経路とは区別）：

- `src/canvas/CardView.tsx`の`handlePointerMove`は移動中の各イベントで`onMove`を呼びます。Drop時だけ呼び出すAPIではありません。
- `src/App.tsx`の`handleCardMove`は移動中に`setHistory`と`setIsDirty`を実行します。
- 同じ`App.tsx`では`pointerup`と`pointercancel`の両方で`commitCardDragSnapshot`を呼びます。現状の取消しを「永続状態に影響しない」と扱うことはできません。
- Drop時の処理はCardの移動だけでなく、島への所属変更を同じUndo操作にまとめます。単純な`x/y`更新だけでは動作互換になりません。

したがって、TEI接続時は既存のCanvasを置き換えるのではなく、**局所プレビュー／取消し／確定**を分けるSUI側の接続口を用意し、確定時にだけ`Host.Commit`へ最終座標とRevisionを渡す必要があります。認可拒否やRevision競合では、プレビューを破棄し、Documentを再取得します。SUIの既存島所属・Undo/Redo規則はSUI側のAction実装に残します。

## ブラウザー側の操作アダプター（参照実装）

`native-action-bridge.mjs` は、FrameworkやNode.js専用APIに依存しないES Moduleです。既存のReact Canvasを置き換えず、画面側が`begin`、`preview`、`cancel`、`drop`を呼び分ける接続口だけを提供します。

- `begin`: Application、Resource、Actionと期待Revisionを固定します。共通の`protocolVersion: "1"`を生成します。画面入力のApplication IDや権限を信用するという意味ではありません。Host側で再検証が必要です。
- `preview`: 最終候補のPayloadのコピーをローカルに保持します。永続化・TEI Actionは呼びません。
- `cancel`: プレビューを破棄します。書き込みは行いません。
- `drop`: 最後のPayloadで1回だけ`dispatchCommit`を呼び、処理中の二重操作を拒否します。拒否や競合を勝手に再実行せず、例外を呼出元へ戻します。

SUI側では [PR #3222](https://github.com/hat47x/sui-sensemaking/pull/3222) が2026-10-10にmainへ統合され、`CardView`の局所プレビュー、`pointerup`のみの確定コールバック、`CanvasShell`の`onCardMoveCommit`、そして`App.tsx`の確定処理への接続を追加しました。`domain/card_drag_commit.ts`は、従来の位置変更とドロップ先の島への所属変更を一つのDocument更新へまとめます。Appは`applyDocumentChange`を一回呼ぶため、Undo履歴も一単位になります。Cardのポインター操作開始時にはSUI Documentの参照とETagを保持し、確定時にいずれかが変化していれば操作を拒否します。これはSUI内部のローカル整合性ガードであり、TEIサーバーでのRevision照合や認可の代用にはなりません。既存の`onMove`経路は別の利用者との互換用に保持しています。

ただし、**SUI内の局所的な確定と、TEIランタイムのAction実行は別段階です。** このPRでは、SUIからTEIの`native-action-bridge.mjs`、`Host.Commit`、実セッション認可を呼び出していません。DocumentのRevisionを操作開始時に捕捉して送信する経路も未実装です。現行のSUI保存は引き続きSUI自身が所有しています。

現段階の変更量はSUIの既存3ファイル（App、CanvasShell、CardView）への編集と、新規ドメイン実装1ファイル・テスト2ファイルです。SUI側のVitest・typecheck・build・実ブラウザー確認は未実行のため、UIの動作互換が実証された状態ではありません。ブラウザーではTEI側のES Moduleを取り込めますが、`structuredClone`を利用できる実行環境が必要です。

## SUI実接続の条件（2026-10-10時点）

SUI main の`App.tsx`は、`handleCardMoveBegin`で`DocumentV1`の参照と`docEtag`を捕捉し、`handleCardMoveCommit`から`commitCardDrag`を呼んで`applyDocumentChange`へ**ローカル変更**を渡しています。現行の`CanvasShell.onCardMoveCommit`は同期的な`void`コールバックです。このまま同じ確定処理にHTTP dispatchを足すと、サーバー側の拒否より先にSUIのUndo履歴・dirty状態を書き換える二重の更新経路ができてしまいます。

実接続では、次の境界を一つずつ確定します。

1. **保存先の一本化**：通常のSUIローカル編集と、試験用のTEIサーバー確定モードを明示的に分離する。後者では承認済みのサーバー応答を受ける前に`applyDocumentChange`を呼ばない。失敗時にローカル保存へ黙ってフォールバックしない。
2. **Revisionの同一性**：TEI要求の`expectedRevision`は実際にCASする保存先のRevisionを使用する。SUIの`docEtag`を採用する場合は、サーバー側のSUI保存Revisionと同一条件であることを実証する。値がない場合や、ドラッグ開始後に未保存編集が入った場合は送信しない。
3. **SUIの意味論の保持**：最終位置だけを書き換える模擬`JSONDocumentStore`は、島への所属変更とUndo/Redoを再現できない。本番アダプターではSUI所有の`commitCardDrag`相当の処理をSUIの権威ある保存境界で原子的に実行し、TEI HostにはCardやIslandの業務規則を実装しない。
4. **確定後の再取得**：成功時は保存側のRevisionとDocumentを再取得し、失敗・競合時はプレビューを破棄して差分を表示する。通信断の場合は成功／失敗を推測せず、保存先を照会する。
5. **保護の継承**：SUIのReadOnly、SafeMode、未レビュー情報、Export制限、テナント境界、監査を既存より弱めない。認証セッション・CSRF・認可・冪等性はTEI実配備側で検証する。

まず`SUI CardView → SUI commit callback → Action dispatcher → TEI Host → SUI所有の保存処理 → 再読込`を一本の接続テストで検証します。旧Revision、認可拒否、通信断、島所属、Undo、来歴保持を合否条件とします。**現時点では未接続・未検証**であり、この文書は実装完了の証拠ではありません。

## 確定結果の再照合（ブラウザー参照実装）

`confirmed-action-commit.mjs`は、UIがサーバーの書込み応答だけを根拠にDocumentを更新することを防ぐ、小さな接続部品です。`createNativeActionBridge`の送信先として組み込みます。

- `dispatchCommit(intent)`：`createJSONActionDispatcher`などで一度だけ確定要求を送ります。権限拒否・Revision競合でローカル状態は更新しません。
- `readResource(intent)`：成功後に、**同じ権威ある保存先**からキャッシュを使わずに取得します。アプリケーション側の読取アダプターは`{applicationID, resourceID, revision, value}`形式へ正規化します。`value`はアプリケーション所有の状態です。
- `isCurrent(intent)`：送信前と読取後に、操作開始時のローカル状態が現在も有効か検査します。**同期的に真偽値の`true`を返した場合だけ通過**させ、Promise、例外、その他の真と評価される値は拒否します。SUIであればDocument参照・ETag・未保存編集状態などを確認します。判定できない場合は成功扱いにしません。
- `applySnapshot(resource, intent)`：Resource IDとRevisionが確定応答と一致し、ローカル状態も変化していない場合にだけ、アプリケーション自身が確定済みSnapshotを反映します。この処理自身も**最新状態を原子的に再確認**する必要があります。反映が原子的に完了した場合に限り、同期・非同期を問わず**真偽値の`true`を明示的に返す**必要があります。例外、`false`、`undefined`などは反映未確認として扱います。

同じResourceに対する処理中の二重確定は拒否します。別Resourceは同時に処理できます。エラー`ActionConfirmationError`の`committedRevision`が非nullの場合、**サーバーは成功を応答済みですが、UIへの反映は未完了**です。読取失敗、Revisionの不一致、UIの状態変化、反映処理の失敗をサーバー側のロールバックとみなさず、旧Revisionで自動再送しません。通信断など、成功応答そのものを受け取れなかった場合も、保存先に変更済みの可能性があるため再照会が必要です。

SUIの`applyDocumentChange`はローカルUndo履歴とdirty状態を更新するため、そのままサーバー確定Snapshotの反映に使うことはできません。実接続では**サーバー確定Snapshotを履歴・dirty状態と整合させて取り込む別のSUI所有経路**が必要です。TEI側がSUIのUndoやIsland規則を解釈することはありません。

検証用の`confirmed-action-commit.test.mjs`（15ケース）と`action-browser-roundtrip.test.mjs`（4ケース）は、取消し・正常確定・失敗時の非反映・通信応答喪失・第二Applicationへの再利用を対象にします。後者のHTTPサーバーと保存処理は**テスト内の合成実装**であり、実Go Host・本番SUI保存経路・実Canvasの動作証拠ではありません。

## Web向けのJSON Action接続口（参照実装）

CLIや同一プロセス内の実行では、`Host.Commit`を直接呼べます。Webではオプションとして、`JSONCommitHandler`（Go標準`net/http`）と`json-action-dispatch.mjs`（ブラウザーの`fetch`）を利用できます。外部Pluginプロセスを必須にしません。

`JSONCommitHandler`は利用側がApplicationとActionの組合せに対応する`DecodePayload`を渡さない限り、503で拒否します。共通層の詳細契約は[`PROTOCOL.md`](PROTOCOL.md)にまとめ、要求は[`action-protocol-v1.schema.json`](action-protocol-v1.schema.json)、成功・失敗の応答は[`action-result-v1.schema.json`](action-result-v1.schema.json)に定義しています。認可は`Host.Authorizer`の責務であり、認証済みセッションから主体を取り出す実装が必要です。HTTP層は任意の主体IDを受信しません。JSONの未知フィールド、複数連結されたリクエスト、64KiB超の入力を拒否します。

要求例（既存の`native-action-bridge.mjs`の`dispatchCommit`に、`createJSONActionDispatcher("/actions/commit")`を渡す）。この外側の形式はSUI以外のアプリケーションでも共通です：

```json
{
  "protocolVersion": "1",
  "applicationID": "sui",
  "resourceID": "doc",
  "actionID": "sui.move",
  "expectedRevision": "r1",
  "payload": {"cardId": "a", "x": 5, "y": 6}
}
```

`POST`には`Content-Type: application/json`と`X-TEI-Action: commit`が必要です。許可された同一オリジンの要求だけを受け付け、クロスサイトの利用を禁止します。成功時は`{"revision":"..."}`、拒否時は`{"error":"..."}`を返します。`403 action_denied`と`409 revision_conflict`はブラウザーでも区別して通知できます。この同一オリジン検査とカスタムヘッダーは、**本番向けのCSRF対策・セッション検証・永続化を代替しません**。

HTTP接続口とブラウザー側Dispatcherの動作は模擬Store・模擬fetchを使ったテストの範囲です。HTTP→Host→SUI形式JSONDocumentStoreで正常更新と再送時の競合を確認する統合テストも追加しましたが、このテストはまだ実リポジトリで実行していません。SUIの本番サーバーにこのHandlerを登録した状態ではありません。

## 実行

```sh
go test -race -count=1 ./tools/reference-sui-action-boundary/actionboundary
node --test tools/reference-sui-action-boundary/native-action-bridge.test.mjs
node --test tools/reference-sui-action-boundary/json-action-dispatch.test.mjs
node --test tools/reference-sui-action-boundary/confirmed-action-commit.test.mjs
node --test tools/reference-sui-action-boundary/action-browser-roundtrip.test.mjs
```

ローカルに再構成したファイルとGitHubのGit Blob SHAを照合したうえで、Node.js 22のテスト16件（操作アダプター6件、JSON Dispatcher 6件、共通契約4件）がすべてPASSしました。GoのHTTP Handlerとそのテスト2ファイルについても、Git Blob SHAが一致するコピーを用意し、模擬Host／模擬Document Storeを組み合わせて`go test -race`と`go vet`をPASSしました。

ただし、これらは**正確なPR headをチェックアウトしたTEI全体のテストではありません**。Canonical Command解決、実セッション認可、SUI本番保存機構の検証結果も含まれません。統合時は正確なコミットを固定し、`tools/repository-health-check.sh`全項目PASSと実行前後のclean状態を確認してください。

## 検証範囲と未実装

参照テストは正常更新、認可拒否、Revision競合、同時更新、Resource IDの不一致、未知Card、非有限座標、不正Document構造、無関係な情報の保持を対象にします。`standard_protocol_test.go`ではSUI形式のCard移動と、独立したInventory形式の在庫調整を同じHandler・Hostに渡すテストを追加しました。双方とも模擬Storeであり、汎用プラグインの本番動作を証明しません。

未実装：SUIの確定コールバックからTEIランタイムへのAction dispatch、操作開始時のRevision捕捉と権限照合、SUIのDocumentV1検証器とTEI保存機構の実連携、実セッションからのSubject解決、実ランタイム権限、ブラウザー配備、SafeModeとExport保護の統合確認、性能測定。

作業窓口はGitHub Issue #993、設計の正本は`plan/issue/000052-sui-reference-adoption-and-plugin-surface-evaluation.md`です。GitHub Issue #52は別件です。
