# Reading Journal — TEI非依存の第二consumer（PoC C）

対象: [Issue #1824](https://github.com/hat47x/tei-platform/issues/1824)、[Draft PR #1825](https://github.com/hat47x/tei-platform/pull/1825)、[既存の横断検証手順](../../plan/evidence/TEI_STUDIO_POC_EVIDENCE_RUNBOOK_20261014.md)。

読書ノート3件を選択する小アプリ。TEIのWorkspace / Source / 三視点 / Acceptanceを持たず、#4の共通Editor、#5のCommand、#9のGraphを**各担当の同一ソース**から組み込む。UI部品の代用品、第二Resolver、実行権限、データ書込みは作らない。Studio側の同じ実部品利用と本consumerの実画面は**別々に検証**する。

## 固定入力とDraft PRの保全

最新の利用検証は、以下のコミットから**必要なpathだけを検証専用checkoutに取り出す**。Draft PRを通常merge/merge APIで統合しない。#8や他担当のbranchへ検証用のコピーを書き込まない。

| ソース | 固定SHA | path |
| --- | --- | --- |
| main | `128429482e02d96afe0123ee7da023256094dfca` | ベース |
| #1825 / consumer | 最新headをPRから再取得 | `reference/tei-studio-independent-consumer/`, `plan/evidence/TEI_STUDIO_POC_EVIDENCE_RUNBOOK_20261014.md` |
| #1828 / Graph | `36799dba33fe0e19db3f7b56e7af1e18da134a71` | `reference/tei-studio-graph/` |
| #1829 / Command | `26f47772ed49f5c4a9e1c50026c335783bc5e153` | `reference/tei-studio-command/` |
| #1830 / Editor | `4970921e613c68e6da29fb603536f2e71fc2d462` | `reference/tei-studio-editor/` |

`git restore --source=<exact-sha> -- <専用path>` のように対象ファイルを取り出す。対象コミットのオブジェクトがローカルにない場合は事前にfetchしてから行う。これは実装のmerge・mainの統合・正式な採用証拠ではない。

## 実ブラウザー用ビルド

consumerディレクトリだけに依存を置く。#3のfrontend root package/lockは触らない。

```sh
cd reference/tei-studio-independent-consumer
node --version                # Node 22.16+（Node 24でも可）
npm install --no-audit --no-fund
npm test
npm run audit:imports
npm run build                 # esbuild: 実#4/#5/#9 TypeScript + CodeMirrorをbundle
python3 browser-e2e.py        # Playwright Python + 実Chromiumがある環境のみ
```

`index.html`の参照先は `main.mjs`、`graph-entry.mjs`、`command-entry.mjs`、`editor-entry.mjs`。esbuild `bundle.mjs` が各入口と実TypeScriptを変換し、CodeMirrorの `@codemirror/*` bare importをconsumer側 `node_modules` から解決して `dist/*.mjs` を作る。`dist/index.html`を生成し、`dist/build-provenance.json`へ各提供元ソースのSHA-256と入力modulesを記録。**raw `index.html` をPython HTTPで開いても `.ts` とbare importは動かない。必ず `dist/` を配信する。**

配信だけなら `python3 -m http.server 4173 --bind 127.0.0.1 --directory dist`。インターネットへ公開しない。`browser-e2e.py`はloopbackの一時サーバーを自動起動し、実Chromiumで検査する。必要なら `TEI_STUDIO_CHROMIUM`にChromium実行ファイルを指定する。

直接依存の版は `@codemirror/state@6.7.6`, `@codemirror/view@6.43.14`, `@codemirror/lint@6.9.7`, `esbuild@0.25.12`。現状、本担当の実行環境はnpmレジストリ到達不可のため**consumerのlockfile生成と実npm installは未実施**。固定入力snapshotのnpm利用可能な実行環境で `npm install --package-lock-only` によりlock生成・一致検査をし、**lock未固定を残課題として扱う**。直接依存のexact pinだけでは推移依存の完全固定を主張しない。

## 部品の扱い

- **Editor**: `editor-entry.mjs`が実#4の `createReadOnlyTextEditor` + `codeMirrorRenderer` をimportする。`journalEditorDocument`は `journal:<id>` / `fixture-v1` とテキストを作るだけ。ノート変更時に `setDocument`、`Unmount Editor`押下または `pagehide`で `destroy`（冪等）。`EditorState.readOnly` / `EditorView.editable(false)`は#4側で設定。保存権限は付与しない。
- **Graph**: `graph-entry.mjs`が実#9 `mountReadonlyGraph`をimport、選択ノードIDをノートpickerへ反映。#9新headでは480幅Graphを約240pxパネルで縮めず、内部横スクロールで14pxのラベルを保つ。実ブラウザーの幅/スクロール測定が必要。旧 `build-graph.mjs` は**同一snapshotのGraph原本SHAの比較**用に残すが、ブラウザ配布物の生成はesbuildを使う。#Workが旧可読性修正head `2d7148c3…` で計測した結果は `clientWidth=225 / scrollWidth=480 / svgWidth=480 / fontSize=14px`。同headにはノード円中心のポインタークリックが選択を更新しない既知FAILがあり、#9は後続head `36799dba…` で修正を提出した。**新headでの実ポインター操作は再検証待ち**。
- **Command**: `command-entry.mjs`が実#5 `mountCommandPanel`をimport。consumer側 `invoke`から関連ノート数の成功、削除の拒否を明示する。権限判定やshellを部品へ委譲しない。
- **Source依存検査**: `npm run audit:imports` は限定された相対import検査、esbuildのmetafileは実際にbundleした全入力を検査し、Editor/Graph/Command原本とCodeMirror依存を確認してTEI意味/Host専用module混入を拒否する。ただし形式名だけから完全な意味分離まで証明するものではない。

## 実行証拠の範囲

`npm test`中のEditor Port単体試験は、#1830の `editor.ts` が固定snapshotに存在するときだけ実行し、`setDocument`、同一versionで別内容の拒否、`destroy`冪等を確認する。#1830が無い単独#1825 checkoutでは明示SKIPとなる。`browser-e2e.py`は**ビルドした実共通部品**でCodeMirror read-only、ノート切替、Graph横スクロール/ノードクリック、Command success/denied、EditorとGraph/Commandの破棄を検査する。ブラウザー試験が未実行ならPASSと記さない。Desktop WebView、認証、利用者価値、PoC C全体資格はこのconsumerのテストから推定しない。

以前の#Work固定snapshot（main + #1825旧head + #1828旧head + #1829 + #1830）ではWindows Node24.21.0、25中24 PASS・audit URL→パス変換で1 FAIL。この問題は#1825で `fileURLToPath` に修正済みだが、**修正後Windowsで再実行した結果**と区別する。経緯は既存横断手順§9を参照。
