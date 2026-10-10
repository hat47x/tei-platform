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
```

ローカルに再構成したファイルとGitHubのGit Blob SHAを照合したうえで、Node.js 22のテスト16件（操作アダプター6件、JSON Dispatcher 6件、共通契約4件）がすべてPASSしました。GoのHTTP Handlerとそのテスト2ファイルについても、Git Blob SHAが一致するコピーを用意し、模擬Host／模擬Document Storeを組み合わせて`go test -race`と`go vet`をPASSしました。

ただし、これらは**正確なPR headをチェックアウトしたTEI全体のテストではありません**。Canonical Command解決、実セッション認可、SUI本番保存機構の検証結果も含まれません。統合時は正確なコミットを固定し、`tools/repository-health-check.sh`全項目PASSと実行前後のclean状態を確認してください。

## 検証範囲と未実装

参照テストは正常更新、認可拒否、Revision競合、同時更新、Resource IDの不一致、未知Card、非有限座標、不正Document構造、無関係な情報の保持を対象にします。`standard_protocol_test.go`ではSUI形式のCard移動と、独立したInventory形式の在庫調整を同じHandler・Hostに渡すテストを追加しました。双方とも模擬Storeであり、汎用プラグインの本番動作を証明しません。

未実装：SUIの確定コールバックからTEIランタイムへのAction dispatch、操作開始時のRevision捕捉と権限照合、SUIのDocumentV1検証器とTEI保存機構の実連携、実セッションからのSubject解決、実ランタイム権限、ブラウザー配備、SafeModeとExport保護の統合確認、性能測定。

作業窓口はGitHub Issue #993、設計の正本は`plan/issue/000052-sui-reference-adoption-and-plugin-surface-evaluation.md`です。GitHub Issue #52は別件です。
