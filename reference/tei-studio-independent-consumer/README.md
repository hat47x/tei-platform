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
| #1830 / Editor | `a7084fe1711cc48eff56fd20e3dc81c508f520a3` | `reference/tei-studio-editor/` |

`git restore --source=<exact-sha> -- <専用path>` のように対象ファイルを取り出す。対象コミットのオブジェクトがローカルにない場合は事前にfetchしてから行う。これは実装のmerge・mainの統合・正式な採用証拠ではない。

## 実ブラウザー用ビルド

consumerディレクトリだけに依存を置く。#3のfrontend root package/lockは触らない。

```sh
cd reference/tei-studio-independent-consumer
node --version                # Node 22.16+（Node 24でも可）
npm ci --no-audit --no-fund
npm test
npm run audit:imports
npm run build                 # esbuild: 実#4/#5/#9 TypeScript + CodeMirrorをbundle
python3 browser-e2e.py        # Playwright Python + 実Chromiumがある環境のみ
```

`index.html`の参照先は `main.mjs`、`graph-entry.mjs`、`command-entry.mjs`、`editor-entry.mjs`。esbuild `bundle.mjs` が各入口と実TypeScriptを変換し、CodeMirrorの `@codemirror/*` bare importをconsumer側 `node_modules` から解決して `dist/*.mjs` を作る。`dist/index.html`を生成し、`dist/build-provenance.json`へ各提供元ソースのSHA-256、npm lock SHA-256と入力modulesを記録。**raw `index.html` をPython HTTPで開いても `.ts` とbare importは動かない。必ず `dist/` を配信する。**

配信だけなら `python3 -m http.server 4173 --bind 127.0.0.1 --directory dist`。インターネットへ公開しない。`browser-e2e.py`はloopbackの一時サーバーを自動起動し、実Chromiumで検査する。必要なら `TEI_STUDIO_CHROMIUM`にChromium実行ファイルを指定する。

直接依存は `@codemirror/commands@6.11.1`、`@codemirror/lint@6.9.7`、`@codemirror/state@6.7.6`、`@codemirror/view@6.43.14`、`esbuild@0.25.12`。Editorの`standardKeymap`登録に伴い、consumerから共有Editorをbundleするためのcommands依存もconsumer専用manifestとlock（lockfileVersion 3）へ固定する。更新後の実行環境・lock hash・bundle結果はIssue #1824に記録する。

## 部品の扱い

- **Editor**: `editor-entry.mjs`が実#4の `createReadOnlyTextEditor` + `codeMirrorRenderer` をimportする。`journalEditorDocument`は `journal:<id>` / `fixture-v1` とテキストを作るだけ。ノート変更時に `setDocument`、`Unmount Editor`押下または `pagehide`で `destroy`（冪等）。さらに別の `Remount Editor` で新しいインスタンスを生成し、現在選択中のノート文書を表示する。`EditorState.readOnly` / `EditorView.editable(false)`は#4側で設定。保存権限は付与しない。
- **Graph**: `graph-entry.mjs`が実#9 `mountReadonlyGraph`をimport、選択ノードIDをノートpickerへ反映。#9新headでは480幅Graphを約240pxパネルで縮めず、内部横スクロールで14pxのラベルを保つ。実ブラウザーの幅/スクロール測定が必要。旧 `build-graph.mjs` は**同一snapshotのGraph原本SHAの比較**用に残すが、ブラウザ配布物の生成はesbuildを使う。#Workが旧可読性修正head `2d7148c3…` で計測した結果は `clientWidth=225 / scrollWidth=480 / svgWidth=480 / fontSize=14px`。同headにはノード円中心のポインタークリックが選択を更新しない既知FAILがあり、#9は後続head `36799dba…` で修正を提出した。#Workの固定consumer旧head+Graph `36799dba…` のWindows実ブラウザーでWindow sketchクリックとNeighborhood walk Enter→picker・見出し・Editor更新を確認済み（Issue #1824 コメント #6181303406）。#8がこの新しいremount実装で再実行した結果ではない。
- **Command**: `command-entry.mjs`が実#5 `mountCommandPanel`をimport。consumer側 `invoke`から関連ノート数の成功、削除の拒否を明示する。権限判定やshellを部品へ委譲しない。
- **Source依存検査**: `npm run audit:imports` は限定された相対import検査、esbuildのmetafileは実際にbundleした全入力を検査し、Editor/Graph/Command原本とCodeMirror依存を確認してTEI意味/Host専用module混入を拒否する。ただし形式名だけから完全な意味分離まで証明するものではない。

## 実行証拠の範囲

`npm test`中のEditor Port単体試験は、#1830の `editor.ts` が固定snapshotに存在するときだけ実行し、`setDocument`、同一versionで別内容の拒否、`destroy`冪等を確認する。#1830が無い単独#1825 checkoutでは明示SKIPとなる。`browser-e2e.py`は**ビルドした実共通部品**でCodeMirror read-only、ノート切替、Graph横スクロール/クリック・Enter選択、Command success/denied、Editor/Graph/Commandの破棄・個別再表示・pagehide最終破棄を検査する。ブラウザー試験が未実行ならPASSと記さない。Desktop WebView、認証、利用者価値、PoC C全体資格はこのconsumerのテストから推定しない。

以前の#Work固定snapshot（main + #1825旧head + #1828旧head + #1829 + #1830）ではWindows Node24.21.0、25中24 PASS・audit URL→パス変換で1 FAIL。この問題は#1825で `fileURLToPath` に修正済みだが、**修正後Windowsで再実行した結果**と区別する。経緯は既存横断手順§9を参照。

#4 Editorは旧head `4970921e…` に対する#Work実CodeMirror検証でCRLF座標ずれが判明し、#4が `a7084fe1…` へ修正済み。snapshotは修正後の `codemirror-coordinates.ts` を含む **`reference/tei-studio-editor/` 全体**を取り出すこと。新headでの実ブラウザー再確認は未実施と区別する。

## Studioとは独立した別用途の検証記録（2026-10-15）

#Workの[Reading Journal独立観測](https://github.com/hat47x/tei-platform/issues/1824#issuecomment-6181303406)は、consumer **#1825 `8cb42899ca2799c4ac4082db51412a8d45a3d6a1`**、Editor **`a7084fe1711cc48eff56fd20e3dc81c508f520a3`**、Graph **`36799dba33fe0e19db3f7b56e7af1e18da134a71`**、Command **`26f47772ed49f5c4a9e1c50026c335783bc5e153`** の専用pathを抽出した**非merge検証snapshot**。Windows Node24.21.0で `npm test` 12/12 PASS、`npm run build` PASS、実ブラウザーでEditorノート変更、Graphクリック/Enter、Command成功/拒否、Editor unmount→reload後再表示を確認済み。観測主体は**#Work**であり、#8の独立実行とは数えない。

[Studio側Draft #1849](https://github.com/hat47x/tei-platform/pull/1849)の実Host接続・実ブラウザーは別snapshot（head `9a290dc61f585dfbf292e9032e5c5bbbad04351a`）。StudioのEditor依存入力 `cc57052f23b69582fc0d3800ab92b2210a28bb4f` とReading Journalの `a7084fe1…` はcommit SHAが異なるが、双方の `editor.ts` / `codemirror.ts` / `codemirror-coordinates.ts` のGit blob SHAはそれぞれ**一致**する。GraphとCommandの入力headも一致する。したがって**同一の共通実装コード**を別の意味モデル（TEIのモデルJSONと読書ノート）に適用した範囲の根拠がある。ただしStudioとconsumerではHost/authorityを共有せず、実OS・UI操作条件も個別の結果である。

新たに#8が追加した **Unmount/Remount Editor・Graph・Command** と、それらを連続する `browser-e2e.py` の自動試験は、上記#Workの旧consumer固定入力には存在しない。新headの実ブラウザー実行が必要で、PASSと呼ばない。#WorkのブラウザーはPython E2Eスクリプト実行ではなく実ブラウザーUIの直接観測だったことも区別する。

## 最小入力・選択通知・破棄再表示の試験入口

- `node --test journal-command.test.mjs` はconsumerだけで実行できる。#8がNode v22.16.0で **2/2 PASS**（GitHub原本とGit blob SHA一致のローカル複製）。ノート選択の結果値、削除拒否の判断が部品から独立していることを検査する。
- `node --test shared-ports.test.mjs` は #1828 `graph.ts` と #1829 `command.ts` の固定Sourceを専用pathに取り出した**同一snapshotで**実行する。Graphの最小入力、クリック・Enter選択ID、旧インスタンス破棄後の通知抑止と再マウント、Commandの成功・拒否・新しいPort生成を検査。Providerのない単独checkoutでは明示的にSKIPするため、SKIPをPASSと誤読しない。
- `python3 browser-e2e.py` は実esbuild/CodeMirrorを配信したChromiumで、UIの各Unmount/Remountと選択連動を検査する。**この追加時点では未実行だった**。後続の固定CIで実行してPASS（下記）。#Workが旧consumer `8cb42899…` で確認したEditor unmount→reload表示とは別テストである。

再利用のため必要なのは独立したEditor文書、Graphのノード/辺と選択通知、Commandの候補と`invoke`だけ。共有部品にTEI固有のSubject/Acceptance等を渡さず、読書ノートのIDと処理結果はconsumer側に保持する。

## 固定4原本の独立CI完走（2026-10-15）

[GitHub Actions #38686075764](https://github.com/hat47x/tei-platform/actions/runs/38686075764) のjob `fixed-provider-journal`は**completed/success**。`test/tei-studio-poc-evidence`のconsumer exact `c780696bfa5bf8a5c8ed6f9b5497ab6be9beac41`と以下の3部品を、各Draftの所有pathだけ`git archive`で一時worktreeに抽出した（**mergeなし**）。

- Graph #1828 `de9ef8f0ac40b164d60ad5d43637ba3afc585c5e` (`graph.ts` blob `436c7d07c56802dbbc6d1dfeb1251470fca75cf6`)
- Command #1829 `63f7ce3af1e6e5a27124aa4aa8ae1331d405f83c` (`command.ts` blob `bb56f3dba6387f7d9bf9604de9b519fb68feb50d`)
- Editor #1830 `472c5d90b0b196f44b203e710c88ff05821d74fa` (`codemirror.ts` blob `eddc3c3e9320b9f61a6b99d970224072528500a0`)

Ubuntu24.04 / Node24.21.0で`npm ci`、`npm test` **18/18 PASS・skip 0**、`npm run audit:imports` PASS、`npm run build` PASS。Python Playwright1.57.0 / Chromium143.0.7499.4の`python browser-e2e.py`も**PASS**。ノート切替、Graph狭幅（480pxの図/245pxのviewport、14pxラベル）、クリック/Enter、Command成功/拒否、Editor/Graph/Commandの個別Unmount/Remountとpagehide破棄を実DOMで確認。新規「選択→Graph同期再描画→Enter/Spaceフォーカス継続」のNode/FakeDOM回帰も18件に含む。**その同期再描画＋連続Spaceの組合せ自体をブラウザーE2Eで実行したわけではない**。

詳細は[既存runbook §12.1](../../plan/evidence/TEI_STUDIO_POC_EVIDENCE_RUNBOOK_20261014.md)を参照。固定4ソースのブラウザーPoCがPASSした範囲であり、実Wails Host、Desktop WebView、全a11y、公開Web認可、メモリリーク完全不在や製品資格は主張しない。

## mainのWorkbenchと同一ソースを使う独立確認

追加CI `.github/workflows/tei-studio-independent-consumer-main-source-ci.yml` は、PRのbase SHAから専用checkoutを作り、その版のWorkbenchとReading Journal consumerがEditor・CodeMirror・Graph・Commandの同じソースファイルをimportすることを確認する。Git blobとSHA-256、consumerの依存グラフ、実際のbundle入力を照合し、base版consumerの依存導入・試験・build・実Chromium E2Eを実行する。続いてPR head側でも、同じ依存導入・試験・build・bundle監査・実Chromium E2Eを実行する。EditorのCodeMirror依存監査は、対象版の`codemirror.ts`がimportするpackageを確認する。`workflow_dispatch`では実行対象SHAを両checkoutに使う。mainへのpushだけではこのPR workflowは自動起動しない。

この確認はIssue #1824のPoC Cにおける「同じUI部品ソースをTEI Workspaceなしのconsumerで利用できる」という受入項目を支える。consumerのブラウザー動作とソース再利用を検査する範囲であり、Workbench自体のブラウザー起動、Wails Host/Desktop WebView、CLIとの同等性、Resolver統合、利用者価値、PoC C全体の受入完了は示さない。既存の固定Provider snapshot CIとWails PoC A CIは独立しており、このworkflowは両方を置き換えない。
