# TEI非依存の別consumer: Reading Journal（初期入口）

**状態: 独立アプリの最小入口とGraph/Command接続コードを実装。固定snapshotでの実ブラウザー再利用は未判定。** [Studio協調 Issue #1824](https://github.com/hat47x/tei-platform/issues/1824) の「開発#8」担当。受入条件はIssueと既存の[横断検証手順](../../plan/evidence/TEI_STUDIO_POC_EVIDENCE_RUNBOOK_20261014.md)を参照する。新しいGate、UI契約、Registry、Resolverは作らない。

このアプリは読書ノート3件を選択・参照する小さな独立consumerであり、TEIの Workspace / Source / 三視点 / Preview / Acceptance を持たない。Editor、Command/Diagnostics、Graphの領域を配置したが、それらの**代用品を実装していない**。未接続の領域にはBLOCKEDと表示する。Web/Desktopのホストが2つあることではなく、**TEI Studioとこの別アプリの2種類のconsumer**で実部品を使って初めて再利用の証拠になる。

## 起動と検査

リポジトリのルートで:

```bash
# スタンドアロン表示用。インターネットへの公開は禁止。
python3 -m http.server 4173 --bind 127.0.0.1
# http://127.0.0.1:4173/reference/tei-studio-independent-consumer/

cd reference/tei-studio-independent-consumer
node --version                       # Node.js >=20
node --test *.test.mjs               # データ・依存解析の単体テスト
node audit-imports.mjs main.mjs notes.mjs
node check-components.mjs           # 既定では BLOCKED / exit 2
```

ブラウザー起動とページ操作は、実ブラウザーの観測を別に記録すること。Node単体試験はブラウザー実画面の合格ではない。依存はNode標準ライブラリのみでnpm installは不要。

## 共通部品の接続（実体提供後）

1. #4 Editor、#5 Command/Diagnostics、#9 Graphのうち、**実際にcommitされた公開API**とlicense/依存をソース・PRで確認する。#9は作業中に `feat/tei-studio-poc-c-graph-reference` へ `reference/tei-studio-graph/graph.ts` を提供した（確認時head `e652a1b655c6dd5f3e52afde9390f1f6cd0771dd`）。#8単体branchには依存原本を置かない。#5は `mountCommandPanel`、#4は `createReadOnlyTextEditor` と `codeMirrorRenderer` の提供を確認済み（CodeMirror依存は別途解決が必要）。
2. 実体のあるmoduleだけをこのconsumerへ接続する。**Graphは#9の実export `mountReadonlyGraph` を `graph-entry.mjs` で使用する接続を実装済み**。Commandも#5の実export `mountCommandPanel` を `command-entry.mjs` で使用。Editorは#4の実APIを確認済みだが未接続。共通UI本体・#3のStudio統合・#1の合成契約を改変しない。
3. 選択した読書ノートの `title/body/related` だけを渡し、共通部品で表示・操作を実際に観測する。Editorを初期read-onlyに保ち、Commandは表示可能と実行許可を区別し、Graphはこのconsumerの関係だけを示す。権限・副作用をUI部品へ与えない。
4. `node audit-imports.mjs main.mjs` を再実行する。import graphがTEI固有のWorkspace/Source/受理モジュールを含まないか再帰検査する。別途Bundlerの解決結果でも確認する（この簡易解析だけでは完全な証明にならない）。
5. `node check-components.mjs --graph=path/to/actual/graph.ts` のように実体ファイルの有無を事前確認できるが、**存在確認は組込み成功ではない**。実際のruntime呼出し・表示・契約拒否・unmount等はPRと横断手順C-05/C-06へ対象SHA付きで追記する。

## 検証の区別

- 本リポジトリで確認可能: 読書ノートの純粋な選択処理、別アプリ入口、自己完結するimport graph、未提供部品へのBLOCKED表示。
- まだ未確認: 共通Editorの接続、Graph/Command原本を固定snapshotへ抽出した状態での実DOM/実ブラウザー描画、Studioと同一の部品実体を使った2 consumer実証、イベント・権限拒否・lifecycle、実ブラウザーやWebViewでの描画、人による利用評価。
- [UI Composition Kit #997](https://github.com/hat47x/tei-platform/issues/997) と [Issue #1824](https://github.com/hat47x/tei-platform/issues/1824) が共通化の設計境界。TEIのstdデータ型や標準PluginをこのReferenceから新規定義しない。

## #9 Graphへの具体的接続（作業中）

`graph-entry.mjs` は #9 の実ソースで公開されている `mountReadonlyGraph(container, input, {onSelect})` を呼ぶ。ノード選択をこのconsumerの読書ノート選択に反映し、ページ破棄時に `dispose()` を呼ぶ。`journal-graph.mjs` は読書ノート3件の表示位置と関係を明示する**consumer固有の入力変換**だけで、Graph描画実装ではない。

Graphのソースが**同じcheckoutに揃っている場合のみ**、次を実施する。

```bash
cd reference/tei-studio-independent-consumer
node build-graph.mjs  # Node.js >=22.16。#9の../tei-studio-graph/graph.tsを型除去
node audit-imports.mjs main.mjs journal-graph.mjs graph-entry.mjs
# 以降、リポジトリルートからloopback http.serverでページを開く
```

生成した `.generated/graph.mjs` はgit管理せず、#9のSourceのSHA-256を生成物冒頭へ記録する。Graphの元コードを別consumerへ複製しない。#9のbranch上にGraphが存在していても、**#8 branchにはまだ含まれない**ので、上記ビルドは現在BLOCKED（exit 2）。#WorkのWindows検証snapshotでは #1825/#1828/#1829/#1830 のexact headから対象専用pathを**mergeなしで取得**し、Graph/Commandのbuildは成功と報告されている。#8単体branchに存在しないことは、検証snapshotを組み立てられないことを意味しない。修正済みconsumer headを使う次回のsnapshotでも対象SHA・取得pathを固定し、実ブラウザー描画は独立に確認する。

## #5 Commandへの具体的接続（作業中）

#5の `reference/tei-studio-command/command.ts` にある実export `mountCommandPanel(target, options)` を `command-entry.mjs` から呼ぶ。読書ノートの関連項目を数えるコマンドはconsumer側の `invoke` が明示成功を返し、削除要求はconsumerが拒否する。部品に権限判定を委ねない。ページ破棄時は `dispose()` を呼ぶ。

同一checkoutに#5の原本が揃った場合のみ:

```bash
cd reference/tei-studio-independent-consumer
node build-command.mjs # Node.js >=22.16; ../tei-studio-command/command.tsを型除去
node audit-imports.mjs main.mjs graph-entry.mjs command-entry.mjs
```

`.generated/command.mjs` は管理対象から除外し、原本のSHA-256を先頭に記録する。#5の原本は別Draft PRに保持する。#Workの固定snapshotではGraph/Commandのbuild成功を確認したが、実呼出し・実DOM描画の成功は別証拠が必要。**Draft PRをmergeせず**各exact headの対象pathを検証用checkoutに取り出し、固定入力で実行する。

## Windows検証snapshotの結果・回帰確認

#Work報告: Windows、Node v24.21.0、main `128429482e02d96afe0123ee7da023256094dfca` とDraft #1825 `7714c79e7e28b335e6706af642efc9e7e285bfff`、#1828 `e652a1b655c6dd5f3e52afde9390f1f6cd0771dd`、#1829 `26f47772ed49f5c4a9e1c50026c335783bc5e153`、#1830 `4970921e613c68e6da29fb603536f2e71fc2d462` の専用path抽出snapshotで **25試験中24 PASS、1 FAIL**。Graph/Command buildは成功。唯一のFAILは `audit-imports.test.mjs` のWindows URL pathname誤用による `ENOENT lstat 'C:\\C:'`。

#8は `audit-imports.test.mjs` と `check-components.mjs` を `fileURLToPath` に修正し、3件の回帰検査を追加した。修正後のローカル **Linux Node v22.16.0では10/10 PASS**。Windows上での修正後実行は別途必要。詳細と環境区分は上記の既存[横断検証手順](../../plan/evidence/TEI_STUDIO_POC_EVIDENCE_RUNBOOK_20261014.md)へ集約する。
