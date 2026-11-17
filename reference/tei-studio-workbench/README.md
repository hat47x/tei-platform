# 読み取り専用Studioワークベンチの参照実装

[協調開発 #1824](https://github.com/hat47x/tei-platform/issues/1824)の画面接続を確認する小さな実装。既存のGo操作を持つWails Hostと、Editor・Graph・Commandの共通部品を利用する。新しいUI登録機構、意味モデル、受理・認可機構は追加しない。

## できることと表示の意味

Hostが起動時に許可した作業領域を選び、モデル一覧の検索、要素の参照、カタログに含まれる直接の親子関係の選択、検証結果の確認を行う。Editorへ渡すのは`InspectSubject`が返す正本のJSON投影であり、ソース本文ではない。別枠では相対パスを指定した`.xead`の`ObserveXead`が返す読み取り専用の観察結果を表示する。許可リストに定めた属性から得たテーブル・項目の名前や元ID、型参照ID、および診断分類と該当数を表示する。表示するIDはTEIの正本IDではなく、文書内の一意性も保証しない。項目IDはテーブル間で重複する場合がある。型はDataType定義の解決結果ではなく参照IDである。診断には未対応要素名・属性名と診断コード、該当数が含まれる。許可リスト外の属性値やXML本文は表示しない。名前や識別子から業務スキーマが分かる可能性は残る。表示はテーブル200件・項目2000件・診断分類500件までとし、省略数を示す。要素種別により既存操作が未対応なら、その失敗を表示する。

**参照結果が保証しないこと**：現在のWorkbenchは`InspectSubject`の`authorityLimits`にある四つの制限（読み取り専用の派生情報、Source位置と意味受理・書込み権限の違い、ローカル読取と外部開示権限の違い）を右側のInspectorへ表示する。必要な制限が欠けるHost応答は`INVALID_INSPECTION_RESPONSE`として採用せず、空欄や旧対象の制限を現在の権限と取り違えない。文字列は`textContent`で描画する。この説明は実際のGrant、Meaning受理、Build/Pluginの実使用や外部開示を保証しない。対象・Workspace変更時の消去と後着結果の拒否は既存の`responses.ts`の表示レーンに従う。

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

## Inspect応答の必須項目を契約だけで足す

Inspector Readerは`inspection-reader.contract.json`と`builder-sources/inspection-reader-v1.transform.json`から生成する（手順は[生成tool](../../tools/reference-studio-part-generator/README.md)）。応答に必須とする項目は契約の`requiredResponseFields`に`{"name", "type"}`で並べる。`type`は`string`・`number`・`boolean`・`present`（値を問わず存在だけを要求）のいずれか。

```json
"requiredResponseFields": [
  {"name": "schemaVersion", "type": "string"}
]
```

項目を一つ足す変更は**契約だけ**で済み、Generator Source・生成tool・transform実行機構は変えない。契約を変えたら、部品を再生成し、`npm test`・`npm run build`の結果を`cmd/tei-studio/assets/workbench/`へ写す。生成Workbenchの固定版（`tools/reference-studio-static-composition/connect-workbench.mjs`）は生成部品のblobを含むため、新しい固定版を追加する。既存の固定版は書き換えない。

足す項目は実際の`InspectSubject`が返すものに限る。`TestContractNamesWhatTheRealOperationReturns`が、項目の有無とJSONの型を実Operationの応答と照合する。手書きの応答fixtureに項目が無ければ、そのfixtureは新しいReaderで拒否される。

これ以外の規則（`effect`、Subjectの一致、fingerprint、`authorityLimits`の4条件）は、まだGenerator Sourceの名前付きスロットである。生成toolは、実装していない契約の節や項目を拒否する（#5365）。

## 未確認の範囲

この文書の手順は、実行結果を示さない。Desktop実WebView、別アプリとの共通部品利用、PoC B生成構成との等価性、アクセシビリティ全項目、利用者による価値確認、公開Web適格性は別々に検証する。Studioの画面成功を生成業務Desktop Targetの合格へ転用しない。

## S-LOC-0：Source参照位置を扱う独立したread-only投影

`source-locations.ts`は、現行`InspectSubject`の`sources.artifacts[]`と`sources.references[]`を、ひとつのCatalog/Inspect応答に対する検証済みの表示用データへ変換する。まず`readInspection`で正確なSubject、read-only応答、Hostが付した`authorityLimits`、World／Source Index指紋の一致を確認してから、参照元の相対Path、Role、行・列・**byte offset**、Artifact指紋を取り出す。同一Subjectへの複数参照や異Roleを統合しない。最大20件だけを返し、観測件数と表示上の省略件数を別に保持する。

Source Indexの`references`が0件、またはGoのnil slice由来でnullだった場合も、**観測0件**であって「Sourceに参照が存在しない」「Workspaceを完全に調べ終わった」ではない（`completenessEstablished:false`）。相対パスの不正、別Subject、Artifact指紋欠落、byte範囲の逆転、旧Catalogとの指紋不一致は拒否する。日本語やCRLFのbyte位置をCodeMirrorのUTF-16列位置と混同しない。

**このモジュールはまだWorkbench画面やWailsの埋込Bundleから利用されない。** パスをFile URL・絶対パスに変換せず、原文を読み取らず、Source Write／Grant／Disclosure／Meaning Acceptanceを推定しない。実画面のS-LOC表示と、実Root・原文byteをHostが再確認するS-OPENは、Desktop製品契約§0.18.6の独立した後続段階である。

`npm test`に`source-locations.test.mjs`を追加しており、対象不一致、複数Role、20件超、0件、旧版、相対Path・範囲の不正、HTML風Pathを含むSourceメタデータを検証する。通常の`npm run typecheck`と同じソースを使用する。純粋投影のCI成功と、Node/esbuildの生成Asset／Wails実Windowの受領は別の資格として扱う。
