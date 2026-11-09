# 操作面の drift 照合を、参照実装で試す（2026-11-07）

- 対象: [操作面の設計候補](../../design/OPERATION_SURFACE_DESIGN_2026-11.md)の10章（実行時の照合）、W6、15章 未決4
- 位置づけ: **参照実装と実行記録**。既定で有効にするかどうかの判断・Owner受理・Gate合否ではない。
- 計測した版: `raw/head.txt`（`origin/main` `f38a63cc` の作業ツリー）。

前の記録（[実行時の照合は何を止められるか](../OPERATION_DRIFT_CHECK_SURVEY_2026-11-07/README.md)）は、照合の材料は出力にあるが**渡す口も比べるコードも無い**ことを示しました。この記録は、その**最初の問い（照合値をどこから渡すか）の一案**を、動く形で試します。

## 1. 何を作ったか

`tools/reference-operation-drift-check`（新規・読み取り専用の参照）。

- 比べるのは**2つだけ**: `WorldFingerprint` と `ModelRevision`。どちらも `tei operations` が既に出力している値です。
- **`Basis` は比べません**（属性追加では変わらないため。前の記録で観測済み）。
- 照合値を**呼出し側が渡す**形（案(a) CLIフラグ）を採りました。`-expected-world-fingerprint` / `-expected-model-revision`。
- 一致しなければ `STALE_OPERATION_SURFACE` を返し、終了コード1。**未指定は `unchecked`** で、PASS ではありません。
- 操作面の**再導出はしません**（渡された値と、いま読んだ操作面の値だけを比べます）。

## 2. 何を観測したか

`reference/standard-crud/workspace` を複製し、片方に属性を1つ足して3つの場合を実行しました。

| 場合 | 期待 | 結果 | 終了コード |
| --- | --- | --- | --- |
| 同じWorkspace + 保存した照合値 | current | `status=current`、2件とも `match=true` | 0 |
| 変更したWorkspace + 同じ照合値 | stale | `status=stale`、`diagnostic=STALE_OPERATION_SURFACE`、2件とも `match=false` | **1** |
| 照合値を渡さない | unchecked | `status=unchecked`、findings 0件 | 0 |

## 3. この観測が示すこと

- **案(a)（CLIフラグ）は、既存の出力だけで成立する。** 操作面を取り直して保存し、後で渡すだけで、意味の変化を検出できます。`trace -expected-fingerprint` と同じ形です。
- **未指定を `unchecked` として区別する必要がある。** 「照合しなかった」を「current」と同じ値にすると、呼出し側は照合済みと誤読します。
- **`Basis` を照合値にしない**理由が、実装でもそのまま現れます（テスト `TestBasisIsNotACounterAndTheReportSaysSo` が、`Basis` だけが違う2つの操作面で判定が変わらないことを固定）。
- **再導出しない照合は、呼出し側が保持した操作面と現在の出力が一致するかしか見ません。** 「実行の直前に意味が変わった」ことは、この形では検出できません（読んだ操作面は現在の意味から導いたものなので、当然一致します）。

## 4. この記録が示さないこと

- **既定で有効にすべきか**。15章の未決4は Owner 判断です。この参照は案(a)が動くことだけを示します。
- **案(b)（保存済み操作面）、案(c)（実行時の再導出）との比較**。実装していません。
- **本物の実行の入口への接続**。`execution.Invoke` や `Command` の経路には接続していません。
- **意味の受理・権限・実DB**。合成の Workspace と読み取りのみです。

## 5. 生データ

| ファイル | 内容 |
| --- | --- |
| `run.sh` | 再現手順（複製・属性追加・3つの場合の実行） |
| `raw/head.txt` | 計測した版とToolchain |
| `raw/pre-status.txt` | 計測前の作業領域（新規ツール3ファイルと`aw/`のみ） |
| `raw/before-operations.json` | 変更前の操作面（照合値の出所） |
| `raw/stored-expectation.txt` | 呼出し側が保存した照合値 |
| `raw/same-workspace.json`・`raw/changed-workspace.json`・`raw/no-expectation.json` | 3つの場合の出力 |
| `raw/changed-workspace.exit.txt` | stale の終了コード |
| `raw/summary.txt` | 上の表を作る元の一覧 |

## 6. 生データのSHA-256

| ファイル | SHA-256 | バイト数 |
| --- | --- | --- |
| `run.sh` | `6a478679473057a91e82ca20d0f79ef93d5a35feaafa0d6da2fc8ea58279ca60` | 3495 |
| `raw/head.txt` | `017969a8a1bc7db159a0d879c3a7dba5fea57f7a0fb87c6657976b0e15b16b99` | 73 |
| `raw/pre-status.txt` | `f252425ebd5f35e1f71e2c623ab574cf1cf3f89e0e1ca35bb28c1286ad26c82e` | 218 |
| `raw/before-operations.json` | `e6a8443c13d8715f2015de58d0abf25691065d8d1c8a42a0474723692622d171` | 3268 |
| `raw/stored-expectation.txt` | `affa95d8b9c9abbee2be5efa716584d5d26c381cfce5a27bd20b3060bf18f636` | 175 |
| `raw/same-workspace.json` | `50c82a80c7be567d11432b7487d322a8752360cde7f5d45d153a9c5757efe2f4` | 1067 |
| `raw/changed-workspace.json` | `2b50cb059cb9f5b1a8de3168e8b8ba127f1e5c0cb52eb5031e62fa0e26f796d5` | 1259 |
| `raw/no-expectation.json` | `03a6f6a39ef2d7f33f902ab32b7646370891d7ef73920447cadaa202063440ec` | 566 |
| `raw/changed-workspace.exit.txt` | `cf205dbb8cea84897b488abcc281bf96698d5e94b1096b16657b4caba9082a22` | 7 |
| `raw/summary.txt` | `106d7b51c7ff546bc47f07b4f95afe019cf05a5d2987e7cb668d9ce768c71782` | 339 |

`run.sh` は空のコピーではなく、この記録に同梱した再現スクリプトです。`raw/*.err.txt` のうち `same-workspace.err.txt`（7バイト）と `changed-workspace.err.txt`（14バイト）は、`go run` の `exit status N` 行だけを含みます。

`raw/pre-status.txt` が0行でないのは、計測時に**新規ツール本体が未追跡**だったためです（`tools/reference-operation-drift-check/`の3ファイルと`aw/`）。ツールはこのPRで別途コミットします。
