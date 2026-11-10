# 「設計変更の影響範囲をCLIだけで答えられるか」の実測（2026-11-09、第13ラウンド）

- 対象: [#3149](https://github.com/hat47x/tei-platform/issues/3149)（ドッグフーディング）、D-9、[#2619](https://github.com/hat47x/tei-platform/issues/2619)・[#2621](https://github.com/hat47x/tei-platform/issues/2621)。
- 実行日: 2026-11-09（Asia/Tokyo）。
- 状態: **read-only の実測**。意味の受理・権限付与・配備は行っていない。

## 1. 目的

ここまでのラウンドは「CLIの応答が正しいか」を測ってきた。このラウンドは、**実際の設計作業の問いにCLIだけで答えられるか**を試す。

題材は、実際に起きた改名（`snapshotFingerprint` → `graphFingerprint`、PR #3633）。設計者が最初に知りたいのは「この名前はどこで宣言され、どこで参照され、どこを直すと壊れるか」である。

## 2. 固定した入力

| 要素 | 値 |
| --- | --- |
| Source | `1fd77acf3a1d95fd9159000e1ac1f918c0e08d28`（`origin/main`） |
| CLI実行物 | `sha256:fc83a50671af2eed9645d3f551f484a99275de975e2930b6fddb9ec67f2ace36`（`main` `5d5d5de6b` でビルド） |
| 環境 | WSL2 Ubuntu 22.04.5、Go `1.27.3 linux/amd64` |

## 3. 試した問いと結果

### Q1 モデル要素の定義箇所は分かるか → **分かる**

```bash
tei inspect -workspace reference/workspace -subject-kind semantic-data -subject-id sales.order
```

`result.source.references` に `semantic/sales/order.tei` が返る。**どのファイルのどこで定義されているかを、CLIだけで特定できる。**

### Q2 リポジトリ全体をWorkspaceにできるか → **できない**

```bash
tei workspace-files -workspace .
# WORKSPACE_LOAD_FAILED（tei.workspace.yaml が無い）
```

リポジトリのルートに `tei.workspace.yaml` は無く、`-workspace .` は失敗する。Workspaceは11個あるが、どれも部分木（`reference/workspace`・`examples/*` など）である。

### Q3 設計文書だけをWorkspaceにできるか → **索引には載るが、名前は subject にならない**

`plan/` と `docs/` を `sources` にした一時Workspaceを作ると、`validate` と `source-index` は成功する。

```bash
tei source-index -workspace /tmp/design_docs
# ok:true, artifacts: 3, references: 0
```

しかし問いを立てると失敗する。

```bash
tei inspect -workspace /tmp/design_docs -subject-kind semantic-data -subject-id snapshotFingerprint
# ok:false, SUBJECT_NOT_FOUND
```

**ファイルは索引に載るが、その中の名前（欄名・型名・関数名）は subject にならない。** `references` が0件なのは、Markdownを意味として解釈していないためである。

### Q4 設計変更の影響範囲は答えられるか → **答えられない**

「`snapshotFingerprint` を `graphFingerprint` に改名すると何が壊れるか」に対して、CLIが出せるのは次の2つだけだった。

- `reference/workspace` に対する `semantic-data:sales.order` の trace（モデル内の関係。CLIの欄名とは無関係）
- `design_docs` の `source-index`（ファイル一覧と `format: md`。中身の名前は出ない）

**どの文書のどの行がその名前を含むか**は、`grep` を使うしかない。つまり設計作業の中心的な問い（名前の参照箇所）に、CLIは答えられない。

## 4. 読み取れること

- 意味モデル（`.tei` の宣言）については、CLIは「定義箇所」も「参照関係」も答えられる。**モデルに対する設計作業はCLIで完結できる。**
- 設計文書（Markdown）については、CLIは「ファイルとして管理する」ところまでで、**中の名前を対象にした問いには答えられない**。D-9 の具体的な帰結である。
- リポジトリ全体を1つのWorkspaceとして扱う経路が無いため、「この改名で壊れる箇所を全部挙げる」という横断の問いは、そもそも入力を作れない。

## 5. 起票しない理由

この不足は既に [#2619](https://github.com/hat47x/tei-platform/issues/2619)（Everything is Data）・[#2621](https://github.com/hat47x/tei-platform/issues/2621)（Markdown設計のAuthoring契約）の設計範囲であり、D-9 として第1ラウンドから記録している。重複Issueを作らない方針に従い、**この実測を D-9 の具体的な証拠として残す**。

## 6. 読み取れないこと（NOT RUN）

- Markdownを意味として読む設計（#2621 の判断待ち）。
- リポジトリ全体を1つのWorkspaceにする経路。
- HTTP・Desktop、実DB・実Provider、権限。

## 7. 位置付け

「モデルについては設計作業をCLIで完結できるが、設計文書についてはできない」ことを、実際の改名を題材に示した記録である。D-9 の解消には #2621 の設計判断が必要で、CLIの欄を増やしても埋まらない。
