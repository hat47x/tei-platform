# コマンド・診断表示部品（PoC C / #5）

Issue: [#1824](https://github.com/hat47x/tei-platform/issues/1824)。Studio専用の操作契約を増やさない、TEI非依存の**試験用UI部品**。Graph部品は#9、画面統合とfrontend依存管理は#3が担当します。本ディレクトリはそれらを置換しません。

## 内容と境界

- `command.ts`: `createCommandPort`（DOMなしの状態/実行委譲）と `mountCommandPanel`（標準DOM表示）。`setCommands` / `setDiagnostics` / `execute` / `subscribe` / `snapshot` / `dispose` を提供します。
- `command.test.mjs`: 正常・拒否・失敗、未知/無効ID、二重実行、例外内容非開示、破棄時abort、購読、DOMの安全な文字列表示とキーボード操作を対象にしたNodeテスト。DOMは**簡易Fake**であり、ブラウザー実機と同一であることを証明しません。
- `example.ts`: TEI意味モデルを知らない利用例。bundler/TypeScriptの組込みは#3のcomposition root側で行い、`command.ts`にWailsや特定フレームワークの依存を追加しません。これだけでは別アプリ実機での2-consumer合格を意味しません。

```sh
node --experimental-strip-types --test reference/tei-studio-command/command.test.mjs
```

Node v22系での実行を想定。依存パッケージ/ルートmanifest/lockの変更はありません。ブラウザーで表示するときは、呼び出し元のビルド工程で`command.ts`と使用コードをJavaScriptに変換し、画面側が用意した要素へ`mountCommandPanel(element, options)`を呼び出してください。生の`.ts`をそのままブラウザーimportすることを保証しません。

## Hostとの明示ポート

`CommandOptions.invoke(commandId, signal): Promise<CommandOutcome>` を、Hostに結び付けた呼び出し元から受け取ります。`CommandOutcome`は次の三種類です。

| kind | 画面状態 | 意味 |
| --- | --- | --- |
| `success` | 成功 | Hostが明示的に成功を返した |
| `denied` | 拒否 | Hostが操作を拒否した |
| `failure` | 失敗 | Hostが失敗を返した、または異常応答／例外 |

実行中は`running`、初期状態は`idle`。画面上の`enabled`は操作候補の表示上の可用性であり、**認証・認可ではありません**。`execute`が返す`true`も処理の受付を意味するだけで成功の判定ではありません。成功は`status.phase`を確認してください。

Host/共通Operation（#1158）は要求ごとにWorkspace・権限・Source revision・Resource境界を検査します。UIで任意shell、書込み、Accept、実行可否推論、任意URL/ファイル参照を行いません。UIへ渡してよい診断・結果だけをHost側で公開してください。例外は意図的に固定した一般メッセージへ変換しますが、明示`message`の秘匿化は提供元の責任です。

`dispose()`は表示を除去し、進行中の`AbortSignal`を中止通知状態にします。ただし、Host側で既に発生した副作用の取り消しを保証しません。遅延応答が発生しても破棄されたUIに再反映しません。既存#997のUI構成、#1529のDraft、#1124/#267のAcceptanceを再実装しません。

## 統合待ち・未確認事項

- #3: frontendへのimportとbundler設定/依存追加（必要時）、画面Slotへの接続、CSS（状態・フォーカス表示）と実ブラウザーでの確認。
- #8: TEIを持たない別アプリでの実利用と再利用判定。添付例だけをもって2-consumer合格としない。
- #7: 他部品と接続した黒箱検証。Host権限の拒否、診断の秘匿化、フォーカス復帰、画面再接続の実証。
- Wails Desktop WebView、Web/複数セッション、画面アクセシビリティ、負荷、パッケージ/ライセンスの総合評価は未実証。

この文書はPoCの利用方法と制限であり、TEIの正式Schema/Core Capability/標準UI契約ではありません。
