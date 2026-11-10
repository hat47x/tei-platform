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

## 5. main での解消（2026-11-09 追記）

この記録を書いたあと、`main` で同じ問題が独立に解消された。`e75d599ba`（PR #3633
「trace-graph-fingerprint-rename」）が次の形を採っている。

- 内側の `result.trace.snapshotFingerprint` を **`graphFingerprint`** に改名し、旧名を
  非推奨の別名として同じ値で残す。
- `result.trace.continuation` に `fingerprint` と `field` を置く。`nextOffset` の隣で、
  継続に使う値とその取得元（`result.snapshotFingerprint`）が分かる。
- `continuation` は `nextOffset` があるページにだけ付く。

修正後の実測（`main` `5d5d5de6b`、実行物 `sha256:fc83a506…`）:

| 位置 | 値（先頭24文字） |
| --- | --- |
| `result.snapshotFingerprint` | `sha256:420e7a2002dcc093f…` |
| `result.trace.graphFingerprint` | `sha256:706c056c5b13b34ed…` |
| `result.trace.snapshotFingerprint`（別名） | 同上 |
| `result.trace.continuation.fingerprint` | `sha256:420e7a2002dcc093f…` |
| `result.trace.continuation.field` | `result.snapshotFingerprint` |

- `continuation.fingerprint` を `-expected-fingerprint` に渡すと `ok:true`・rows=2。
- `graphFingerprint` を渡すと `STALE_SNAPSHOT`・exit 1（継続トークンではないので正しい）。
- `graphFingerprint` と別名 `snapshotFingerprint` は同じ値。

生データは `plan/evidence/TEI_DOGFOODING_ROUND12_CONTINUATION_MAIN_2026-11-09/` にある。

## 6. 重複の扱い

この記録のために私が書いた実装（`continuationFingerprint` という平坦な欄を足す案）は
採用しなかった。`main` の形が上位互換である（改名で同名衝突そのものを解消し、取得元まで
示す）ため、重複する実装を持ち込まない。

## 7. 残した範囲

- `-scope` を変えた場合の継続拒否は未確認。
- `-direction` を変えた場合の拒否は未確認。
- MCP の `tei_trace` が `continuation` を返すかは、この記録では実測していない。
- 欄の完全な一本化（別名 `snapshotFingerprint` の削除）は未実施。既存の読み手の移行が前提。

## 8. 読み取れないこと（NOT RUN）

- HTTP・Desktop、実DB・実Provider・実Host、意味の受理・権限・配備。
- MCP 後続 protocol 版、実Harness 接続。

## 9. 位置付け

「同名で値の違う2つの欄があり、`nextOffset` の隣の値を選ぶと失敗する」ことを実測し、
`main` の解消を確認した記録である。意味の受理や権限を示さない。
