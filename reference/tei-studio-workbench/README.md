# 読み取り専用Studioワークベンチの参照実装

[協調開発 #1824](https://github.com/hat47x/tei-platform/issues/1824)の画面接続を確認する小さな実装。既存のGo操作を持つWails Hostと、Editor・Graph・Commandの共通部品を利用する。新しいUI登録機構、意味モデル、受理・認可機構は追加しない。

## できることと表示の意味

Hostが起動時に許可した作業領域を選び、モデル一覧の検索、要素の参照、カタログに含まれる直接の親子関係の選択、検証結果の確認を行う。Editorへ渡すのは`InspectSubject`の`canonical`をJSONにした投影であり、ソース本文ではない。別枠で相対パスを指定した`.xead`の`ObserveXead` read-only projectionを表示するが、許可範囲内のメタデータ（Table/Field名・source ID・型ID・診断コード等）のみで、属性値やXML本文は表示しない。名前や識別子が業務スキーマを示す可能性は残る。要素種別により既存操作が未対応なら、その失敗を表示する。

Graphは選択要素を含む最大8件を表示する。省略件数を画面に示し、実在しない関係を補わない。画面の選択は権限にならない。`modelRevision`や指紋は参照結果の識別用であり、書込みの許可として利用しない。保存・受理・任意コマンド・イベント配信は提供しない。

`ok:true`でも検証診断にerrorがあれば操作結果を失敗として表示し、warningを消さない。通信失敗をモックの成功へ置き換えない。作業領域や選択の変更後に届いた古い参照応答は採用しない。

作業領域の切替・再読込ではCommandの`beginContext()`を呼び、旧コマンドの状態と診断を消去する。参照・一覧・検証・XEAD観察の応答を別々に管理し、同じ種類の要求は最新の応答だけを表示する。XEADのパス変更時も保留中の応答を無効化する。旧要求の失敗も新しい画面へ表示しない。これは表示上の処理であり、Hostの処理中止や副作用の取消しを保証しない。再読込コマンドは自身の表示状態も初期化するため、完了は更新後のモデルと検証結果で確認する。

## 依存する実装

この変更単独のmainには下記の並行開発部品が存在しない場合がある。Draft PRをmergeせず、各対象pathを固定版から取り出した検証用checkoutで確認する。元ブランチは変更しない。

- #1832: `cmd/tei-studio/`と`internal/studiohost/`。既存の`createReadOnlyClient`と5操作を利用。
- #1830: `reference/tei-studio-editor/`の`createReadOnlyTextEditor`、`codeMirrorRenderer`。
- #1828: `reference/tei-studio-graph/`の`mountReadonlyGraph`。
- #1829: `reference/tei-studio-command/`の`mountCommandPanel`、`beginContext`、`setDiagnosticsForContext`。世代管理APIを追加した版が必要。

部品の実装をこのディレクトリへ複製しない。依存の固定版、実際の試験結果、未確認事項は本PRと#1824のコメントへ記録する。

## ビルドと検査

Node.js 24以上で、このディレクトリから実行する。

```sh
npm ci
npm test
npm run typecheck
npm run build
```

CodeMirror・TypeScript・esbuildは専用のmanifest/lockへ固定する。兄弟ディレクトリのCodeMirror adapterからも同じ依存を参照できるよう、ビルドの`nodePaths`とTypeScriptの`paths`をこの専用依存先へ向ける。生成した`dist/dependencies.json`で取り込んだ原本ファイルを確認できる。バンドルした依存のライセンス本文は`dist/THIRD_PARTY_NOTICES.txt`へまとめる。これは依存解析結果であり、成果物署名や配布の真正性を証明するものではない。

検証用checkout内で、`dist/`の内容をPoC Aの`cmd/tei-studio/assets/`へコピーし、Hostを再ビルドして起動する。元PRのbootstrap資産を変更・commitする手順ではない。Wailsの`/wails/runtime.js`は実Hostが供給するため、この画面の静的配信だけでは実サービスへ接続できない。

```sh
cd ../../cmd/tei-studio
go mod tidy
CGO_ENABLED=0 go build -tags server -o tei-studio-server .
./tei-studio-server -workspace ../../reference/workspace -port 43124
```

WebはPoC Aと同じloopback・ローカル単一利用者・参照専用の範囲。実認証・複数利用者の隔離、Sourceの安全な読み書き、OS資源の認可をこのUIから補わない。

## 未確認の範囲

この文書の手順は、実行結果を示さない。Desktop実WebView、別アプリとの共通部品利用、PoC B生成構成との等価性、アクセシビリティ全項目、利用者による価値確認、公開Web適格性は別々に検証する。Studioの画面成功を生成業務Desktop Targetの合格へ転用しない。
