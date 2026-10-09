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

`Host.Commit`はApplication ID、Document ID、Action ID、期待Revision、アプリケーション固有のPayloadを受け取ります。更新前にActionの対応と認可を確認し、保存側へ委譲します。**Hostだけでは同時更新を防げません。** Revisionの原子的な照合と更新はStoreの責務です。

## SUIの実コードとの対応

SUI `03_Implement/frontend/src/domain/types.ts` の`DocumentV1`では、Cardの座標は`card.x`、`card.y`です。`card.position`という構造ではありません。参照用`JSONDocumentStore`はこの座標形式に合わせ、`version: 1`と文書本体の`id`を確認します。Sourceは`card.meta.source`、保留は`card.holdState`、レビューは`reviewAttribution`などを保持します。未知のEdge種別も移動によって変更しません。

ただし、参照用StoreはSUIのTypeScript検証器を呼び出さず、島への自動所属やUndo/Redoも再現しません。構造が似ていることと、SUIのドメイン処理に接続したことを混同しないでください。

**Canvas接続前に解消すべき差分**（SUI mainを2026-10-10に確認）：

- `src/canvas/CardView.tsx`の`handlePointerMove`は移動中の各イベントで`onMove`を呼びます。Drop時だけ呼び出すAPIではありません。
- `src/App.tsx`の`handleCardMove`は移動中に`setHistory`と`setIsDirty`を実行します。
- 同じ`App.tsx`では`pointerup`と`pointercancel`の両方で`commitCardDragSnapshot`を呼びます。現状の取消しを「永続状態に影響しない」と扱うことはできません。
- Drop時の処理はCardの移動だけでなく、島への所属変更を同じUndo操作にまとめます。単純な`x/y`更新だけでは動作互換になりません。

したがって、TEI接続時は既存のCanvasを置き換えるのではなく、**局所プレビュー／取消し／確定**を分けるSUI側の接続口を用意し、確定時にだけ`Host.Commit`へ最終座標とRevisionを渡す必要があります。認可拒否やRevision競合では、プレビューを破棄し、Documentを再取得します。SUIの既存島所属・Undo/Redo規則はSUI側のAction実装に残します。

## ブラウザー側の操作アダプター（参照実装）

`native-action-bridge.mjs` は、FrameworkやNode.js専用APIに依存しないES Moduleです。既存のReact Canvasを置き換えず、画面側が`begin`、`preview`、`cancel`、`drop`を呼び分ける接続口だけを提供します。

- `begin`: 操作対象と期待Revisionを固定します。画面入力のApplication IDや権限を信用するという意味ではありません。Host側で再検証が必要です。
- `preview`: 最終候補のPayloadのコピーをローカルに保持します。永続化・TEI Actionは呼びません。
- `cancel`: プレビューを破棄します。書き込みは行いません。
- `drop`: 最後のPayloadで1回だけ`dispatchCommit`を呼び、処理中の二重操作を拒否します。拒否や競合を勝手に再実行せず、例外を呼出元へ戻します。

SUI側では [PR #3222](https://github.com/hat47x/sui-sensemaking/pull/3222) で、`CardView`の局所プレビューと`pointerup`のみの確定コールバック、および`CanvasShell`の任意の`onCardMoveCommit`を実装しました。これは**任意選択のUI接続口**であり、現行`App.tsx`は利用していません。TEIの`native-action-bridge.mjs`や実行時の`Host.Commit`と結線した状態でもありません。SUI側のVitestと実ブラウザー検証は未実施です。

**既存SUIのReact CanvasとTEI Actionを接続した状態ではありません。** ドラッグ中の描画と取消しをSUIの局所状態へ切り分け、SUI自身の島所属変更・履歴確定と統合する仕事が残ります。ブラウザーではこのES Moduleを取り込めますが、`structuredClone`を利用できる実行環境が必要です。永続化用通信と認証セッションは別途接続します。

## 実行

```sh
go test -race -count=1 ./tools/reference-sui-action-boundary/actionboundary
node --test tools/reference-sui-action-boundary/native-action-bridge.test.mjs
```

操作アダプターのNode.js 22単体テスト6件は2026-10-10に実行してPASSを確認しました。GitHub上の正確なPR headをcheckoutして実行した結果ではありません。Goテストとリポジトリ全体の健全性は未検証です。統合時は対象コミットを固定し、`tools/repository-health-check.sh`全項目のPASSと実行前後のclean状態を別途確認してください。

## 検証範囲と未実装

参照テストは正常更新、認可拒否、Revision競合、同時更新、Document IDの不一致、未知Card、非有限座標、不正Document構造、無関係な情報の保持を対象にします。第二Applicationのテストは模擬Storeを用いており、汎用プラグインの本番動作を証明しません。

未実装：実際のReact Canvasとの接続、SUIのDocumentV1検証器と保存機構の呼出し、実セッションからのSubject解決、実ランタイム権限、ブラウザー配備、SafeModeとExport保護、性能測定、変更箇所数の比較。

作業窓口はGitHub Issue #993、設計の正本は`plan/issue/000052-sui-reference-adoption-and-plugin-surface-evaluation.md`です。GitHub Issue #52は別件です。
