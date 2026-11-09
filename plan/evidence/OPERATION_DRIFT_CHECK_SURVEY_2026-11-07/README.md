# 実行時の照合は何を止められるか（2026-11-07）

- 対象: [操作面の設計候補](../../design/OPERATION_SURFACE_DESIGN_2026-11.md)の10章（実行時の照合）と、15章 未決4（drift の照合を既定で有効にするか）
- 位置づけ: **実行して観測した記録**。既定をどうするかの判断・Owner受理・Gate合否ではない。
- 計測した版: `raw/head.txt`（`origin/main` `12b991ee` の作業ツリー）。

設計10章は、実行のたびに操作面を再導出せず、**照合値の一致だけ**を確かめ、一致しなければ `OPERATION_SURFACE_STALE` で止める、としています。この記録は、現行でそれがどこまで可能かを観測します。

## 1. 観測

### 1.1 意味を変えると、照合値は変わるが `basis` は変わらない

`reference/standard-crud/workspace`の複製に**属性を1つ足しただけ**で、操作面を2回取りました。

| | 変更前 | 変更後 |
| --- | --- | --- |
| `worldFingerprint` | `sha256:bd53dbac…` | `sha256:bf289866…`（**変化**） |
| `modelRevision` | `sha256:c60ed68c…` | `sha256:92102c0e…`（**変化**） |
| 操作の数 | 5 | 5 |
| 5件の`basis` | 記載の5値 | **すべて同じ値（変化なし）** |

`basis`は`persist`の写像から作られており、属性の追加では変わらないためです。

### 1.2 CLI には照合値を渡す口が無い

- `tei operations` のフラグに `-expected-fingerprint` はありません（`expected-fingerprint` を持つのは `trace` だけ）。
- `OPERATION_SURFACE_STALE` は**リポジトリ全体で0件**です（`internal`・`cmd`・`tools`）。

### 1.3 実行時に操作面を読む経路

非試験で操作面を読むのは4箇所です。

| 箇所 | 用途 |
| --- | --- |
| `cmd/tei/cli_operations_artifacts.go` | CLIの出力 |
| `internal/artifact/surface.go` | 成果物一式への埋め込み |
| `internal/execution/period_guard.go` | **期間の拒否判定**（実行の入口） |
| `tools/reference-operation-pattern-crosswalk` | 参照実装 |

`internal/execution/period_guard.go` は `operations.GuardCapabilityWith` に `Surface` を渡しますが、**`WorldFingerprint` や `ModelRevision` を比べる行はありません**。期間の判定だけを行います。

## 2. この観測が示すこと

- **照合の材料は既に出力にある**（`worldFingerprint`・`modelRevision`）。しかし**比べる相手を渡す口が無く**、比べるコードも無い。
- したがって未決4は「既定で有効にするか」の前に、**照合値をどう渡すか（CLIのフラグ、保存済み操作面の保持、実行時の再導出のどれか）**を決める必要があります。現行にはその経路が1つもありません。
- `basis` は属性追加では変わらないため、**`basis` だけを照合値にすると変更を見逃します**。設計10章の「`basis` が現在の意味の照合値と一致するか」を字義どおり実装すると、この変更は検出できません。`worldFingerprint`／`modelRevision` を使う必要があります。

## 3. この記録が示さないこと

- **既定で有効にすべきか**。15章の未決4はOwner判断です。
- **照合を実装する方法**。フラグか、保存済み操作面か、実行時の再導出かは決めていません。
- **`basis` の設計が誤りであること**。`basis`は「どの入力から導いたか」を示す値で、意味全体の版ではありません。設計10章の2番（`basis` と現在の意味の照合）は、`basis` を意味の版として扱う読み方になっており、その読み方が正しいかは本文で決めていません。
- **実DB・実Provider・権限**。合成のWorkspaceとCLIの読み取りのみです。

## 4. 生データ

| ファイル | 内容 |
| --- | --- |
| `run.sh` | 再現手順（複製に属性を1つ足して2回操作面を取る／フラグと照合コードを走査する） |
| `raw/head.txt` | 計測した版とToolchain |
| `raw/pre-status.txt` | 計測前の作業領域（0行） |
| `raw/before-operations.json`・`raw/after-operations.json` | 変更前後の操作面 |
| `raw/flag-scan.txt` | `operations` のフラグと `expected-fingerprint` の所在 |
| `raw/drift-scan.txt` | 操作面を読む非試験の箇所と、照合値の比較の有無、`OPERATION_SURFACE_STALE` の不在 |
| `raw/summary.txt` | 上の表を作る元の一覧 |

`*.err.txt` はすべて0バイトです。

## 5. 生データのSHA-256

| ファイル | SHA-256 | バイト数 |
| --- | --- | --- |
| `run.sh` | `b46d6d85b4cbe29225c1cf2f51e3daf70934c6fabdc6bb7e0bb2759903178148` | 5010 |
| `raw/head.txt` | `03e8680a35ddf276a59a14514097ea5568c14f8dedbf95e699294e8babbe234c` | 73 |
| `raw/pre-status.txt` | `e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855`（空） | 0 |
| `raw/before-operations.json` | `e6a8443c13d8715f2015de58d0abf25691065d8d1c8a42a0474723692622d171` | 3268 |
| `raw/after-operations.json` | `a1ecd7b8e5e27f097f53d03d24d263afbf26a9677893894ecf7cf019aaa19108` | 3268 |
| `raw/flag-scan.txt` | `3f2d7b70e36e74356937b862608e747cf8c9dc2e863cd1b3a8303b618e394850` | 1679 |
| `raw/drift-scan.txt` | `23e3235b1742dd82863a1d1c0b99a05814af8ce60d57efd4c3c102d3df4549bd` | 2643 |
| `raw/summary.txt` | `ac5c0fe225b5747cf158d8992f6f07e1899ac8f6fa2037658c3eda297257cbc8` | 1731 |

`raw/before-operations.json` のSHA-256は、[構成Kernel移行Gateの実行記録](../COMPOSITION_FIRST_CONSUMER_RUN_2026-11-07/README.md)の`raw/operations.json`と同じ値です。属性を足した`after`側は別の値になり、`summary.txt`の比較でWorld指紋と`modelRevision`が変わったことを示しています。

走査は grep による文字列探索で、Goの型解決をしていません。`OPERATION_SURFACE_STALE` が別名で実装されている可能性は、この記録だけでは否定できません。
