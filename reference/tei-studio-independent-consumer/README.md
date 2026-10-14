# TEI非依存の別consumer: Reading Journal（初期入口）

**状態: 独立アプリの最小入口のみ実装。共通UI再利用はBLOCKED。** [Studio協調 Issue #1824](https://github.com/hat47x/tei-platform/issues/1824) の「開発#8」担当。受入条件はIssueと既存の[横断検証手順](../../plan/evidence/TEI_STUDIO_POC_EVIDENCE_RUNBOOK_20261014.md)を参照する。新しいGate、UI契約、Registry、Resolverは作らない。

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

1. #4 Editor、#5 Command/Diagnostics、#9 Graphのうち、**実際にcommitされた公開API**とlicense/依存をソース・PRで確認する。作業開始時点の#9 `feat/tei-studio-poc-c-graph-reference` はmainと同一で `graph.ts` 未提供、#4/#5は提供パス未確認。
2. 実体のあるmoduleだけを **このconsumerの `main.mjs` に明示import** し、必要ならこのconsumer配下で局所的な接続コードを書く。実装のexport名や引数は未定なので、ここでは事前に仮定しない。共通UI本体・#3のStudio統合・#1の合成契約を改変しない。
3. 選択した読書ノートの `title/body/related` だけを渡し、共通部品で表示・操作を実際に観測する。Editorを初期read-onlyに保ち、Commandは表示可能と実行許可を区別し、Graphはこのconsumerの関係だけを示す。権限・副作用をUI部品へ与えない。
4. `node audit-imports.mjs main.mjs` を再実行する。import graphがTEI固有のWorkspace/Source/受理モジュールを含まないか再帰検査する。別途Bundlerの解決結果でも確認する（この簡易解析だけでは完全な証明にならない）。
5. `node check-components.mjs --graph=path/to/actual/graph.ts` のように実体ファイルの有無を事前確認できるが、**存在確認は組込み成功ではない**。実際のruntime呼出し・表示・契約拒否・unmount等はPRと横断手順C-05/C-06へ対象SHA付きで追記する。

## 検証の区別

- 本リポジトリで確認可能: 読書ノートの純粋な選択処理、別アプリ入口、自己完結するimport graph、未提供部品へのBLOCKED表示。
- まだ未確認: 共通Editor/Command/Graphのimport、Studioと同一の部品実体、イベント・権限拒否・lifecycle、実ブラウザーやWebViewでの描画、人による利用評価。
- [UI Composition Kit #997](https://github.com/hat47x/tei-platform/issues/997) と [Issue #1824](https://github.com/hat47x/tei-platform/issues/1824) が共通化の設計境界。TEIのstdデータ型や標準PluginをこのReferenceから新規定義しない。
