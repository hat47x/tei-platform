# trace の継続トークンが2つの同名欄に分かれている（2026-11-09、第11ラウンド）

- 対象: [#3625](https://github.com/hat47x/tei-platform/issues/3625)、[#3420](https://github.com/hat47x/tei-platform/issues/3420) T06、[ADR-0048](../../adr/0048-shared-application-api-and-transport-adapters.md) §5 T06。
- 前提: [第10ラウンド](../TEI_DOGFOODING_ROUND10_CAPI2A_PARITY_2026-11-09/README.md)で、`trace` のページ継続は未検査と記録した。
- 実行日: 2026-11-09（Asia/Tokyo）。
- 状態: **read-only の比較実測と、1つの欄の追加**。

## 1. 目的

T06（継続トークンと版）を、実CLIで確かめる。`-offset`・`-limit` と `nextOffset` の往復、古い指紋の拒否、そして「どの値が継続に使えるか」。

## 2. 固定した入力

| 要素 | 値 |
| --- | --- |
| Source（比較の基準） | `d65490d622a641c0badceb34ae6215ae5cbb9c79`（`origin/main`） |
| CLI実行物 | `sha256:0a8fc33a3fb906987be4c1a93328296c7317137be019f425dbb223424af66103` |
| MCP実行物 | `sha256:88b558efe69a185dfa5a4838c6f247af35bce36e00500fc9781f81b4363cba22` |
| 環境 | WSL2 Ubuntu 22.04.5、Go `1.27.3 linux/amd64` |
| 対象Workspace | `reference/workspace`（`semantic-data:sales.order`） |

## 3. 生データ

| ファイル | 内容 |
| --- | --- |
| `00-head.txt`・`01-binaries.txt` | 対象 commit と実行物の識別 |
| `02-full-all.json`・`03-full-limit2.json`・`04-full-offset2.json` | ページングの基本 |
| `05-page1.json`・`06-page2-ok.json` | 継続（内側の値を使った場合） |
| `07-other-subject.json` | 別subjectで同じoffsetを使った場合 |
| `08-offset-beyond.json` | total を超えるoffset |
| `09-mcp-limit2.json` | MCP の `tei_trace` に `limit` を渡した場合 |
| `10-A.json`〜`13-D.json` | 同一入力の繰り返しとページ違いでの指紋の安定 |
| `14-page1.json`・`15-outer.txt`・`16-inner.txt`・`17-page2-outer.json`・`18-page2-inner.json` | **どちらの値が継続に使えるか** |
| `09-new-tests.txt`・`10-operations-package.txt` | 追加した試験と operations パッケージ |
| `07-gofmt.txt`・`08-vet.txt` | 0 byte |
| `11-page1-after.json`・`12-alias.txt`・`13-page2-after.json` | 修正後の応答と継続 |

## 4. 観測（修正前）

### 4.1 ページングは動く

`-mode full` は全4行を返し、`-limit 2` は2行と `nextOffset: 2`、`-offset 2 -limit 2` は残り2行で `nextOffset` を省く。`-offset 99` は0行で成功（エラーにしない）。MCP の `tei_trace` も `limit` を受け、同じく2行と `nextOffset: 2` を返した。

### 4.2 同じ入力なら指紋は安定

`-limit 2` を2回、`-offset 2 -limit 2` を2回実行し、`result.snapshotFingerprint` はどちらも一致した。ページを変えても `result.snapshotFingerprint` は同じ値で、`modelref` の内側の値も同じだった。

### 4.3 同名の欄が2つあり、継続に使えるのは外側だけ

| 位置 | 値（先頭24文字） |
| --- | --- |
| `result.snapshotFingerprint` | `sha256:9a09be1be966b4aaf…` |
| `result.trace.snapshotFingerprint` | `sha256:954fc77cbaaae2a6a…` |
| `result.trace.nextOffset` | `2` |

継続の結果:

| `-expected-fingerprint` に渡した値 | 結果 |
| --- | --- |
| 外側（`result.snapshotFingerprint`） | `ok:true`・rows=2 |
| 内側（`result.trace.snapshotFingerprint`） | `STALE_SNAPSHOT`・exit 1 |

`nextOffset` は内側（`result.trace`）にあるため、**`nextOffset` の隣にある同名の欄**を選ぶのが自然な読み方であり、その選択が毎回失敗する。しかも `STALE_SNAPSHOT` は「版が変わった」と読めるため、実際には変わっていない版を変わったと誤解させる。

内側は `modelref` のグラフ・規則・対象・方向だけを含み、外側はそれに World fingerprint と Source Index fingerprint を加えたもの。`docs/getting-started.md` は外側を使うと明記しているが、応答だけからは区別できない。

## 5. 修正

外側の値を、`nextOffset` と同じ階層に `continuationFingerprint` としても載せた。既存の2つの欄は改名せず残す（互換）。

- 変更: `internal/modelref/trace.go` に `ContinuationFingerprint` を追加、`internal/operations/trace_workspace.go` が外側の値と同じ値を入れる。
- 修正後の応答: `result.trace.continuationFingerprint` が `sha256:9a09be1be966…` で、`result.snapshotFingerprint` と一致し、`result.trace.snapshotFingerprint` とは異なる。
- その値を `-expected-fingerprint` に渡すと `ok:true`・rows=2。`summary` でも同じ値を持つ。
- 回帰試験: `internal/operations/trace_workspace_continuation_test.go` の2件。外側との一致、内側との不一致、継続の成功、内側を渡した場合の拒否、summary での同値を検査する。
- 変異（alias を入れない／内側の値を入れる）はどちらも FAIL した。

## 6. 残した範囲

- 欄の**改名**や、`nextOffset` の位置の変更は行っていない。既存の読み手への影響が大きく、移行の宣言が必要になるため。
- `-scope` を変えた場合の継続拒否は未確認（`modelref` のダイジェストは `Subject` と `Direction` を含むが `Scope` を含まない。`operations` 側の外側の値は Scope に依存しない）。
- `-direction` を変えた場合の拒否は未確認。
- MCP の `tei_trace` は `expectedFingerprint` を受けけるが、`continuationFingerprint` を返すかは本ラウンドで確認していない（`content[0].text` は同じ operation の投影なので同じ欄を持つはずだが、実測していない）。

## 7. 位置付け

T06 のうち「どの値が継続トークンか」を応答から判別できるようにした。改名・移動という抜本的な整理は #3625 に残す。この変更は意味の受理や権限を与えない。
