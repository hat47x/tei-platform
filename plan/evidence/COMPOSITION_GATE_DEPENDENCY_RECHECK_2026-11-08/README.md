# 構成Kernel移行Gateの依存の形の主張を、いまの main で確かめる（2026-11-08）

- 対象: [構成Kernel移行Gate](../../gate/COMPOSITION_KERNEL_MIGRATION_GATE_2026-10.md) の 4.1 にある3つの主張
- 位置づけ: **Gateが記録した依存の形を、現在の main で取り直した記録**。Gateの合否判定ではありません。
- 計測した版: `raw/head.txt`（`origin/main` `91befb7b` の作業ツリー）。

Gateの 4.1 は、実行記録の補足として次の3つを述べています。

1. A（`cmd/tei`）は `internal/composition` を含まない
2. B（Host所有の実行単位）は `internal/operations` を含まない
3. `internal/servicehost` は非試験のファイルからどこも import されていない

この記録は、**その3つが現在も成り立つか**を、Goの依存閉包で確かめます。

## 1. 観測

```text
== tei/cmd/tei ==
includes internal/composition: 0
includes internal/operations: 1
includes internal/servicehost: 0

== reference/web-composition/go-host ==
includes internal/operations: 0
includes internal/composition: 1
includes internal/servicehost: 0

== packages whose closure includes internal/servicehost ==
count (excluding servicehost itself): 0
file importers (all): 3
file importers (non-test): 0
```

| Gateの主張 | 現在の観測 | 成否 |
| --- | --- | --- |
| A `cmd/tei` は `internal/composition` を含まない | 含まない（0） | **成立** |
| B `reference/web-composition/go-host` は `internal/operations` を含まない | 含まない（0） | **成立** |
| `internal/servicehost` の非試験 importer は 0 | ファイル3件・**非試験0件**、閉包でも自分以外0件 | **成立** |

`internal/servicehost` を import するファイル3件は、すべて `_test.go` です。

```text
internal/driver/reference_current_host_service_route_test.go
reference/web-composition/go-host/canonical_generation_service_handoff_test.go
reference/web-composition/go-host/two_application_host_service_carrier_test.go
```

## 2. この観測が示すこと

- Gateが記録した**3つの依存の形は、いまもすべて成り立っています**。
- **A と B は、互いに相手側の層を含みません。** A（CLI）は `operations` を含み `composition` を含まず、B（Host）は `composition` を含み `operations` を含みません。**この非対称が、Gateが「A/B/Cの個別PASSを混ぜない」と言う根拠**です。
- `servicehost` に**非試験の呼出し元が無い**ことも維持されています。つまり B の試験が成功しても、それは**実HostからのService Carrier呼出し**ではありません。

## 3. この観測が示さないこと

- **Gateの合否**。この記録は依存の形だけを見ており、`G1`〜`G4` の判定は[Gate本文](../../gate/COMPOSITION_KERNEL_MIGRATION_GATE_2026-10.md)の担当です。
- **A・B・Cが同じ意味を通ったこと**。依存の形が違うことは、経路が同じでないことの証明ではありません。
- **実行時の挙動**。`go list` による静的な依存閉包のみです。
- **`internal/servicehost` を使うべきか**。接続の是非はGateの判断です。

## 4. 生データ

| ファイル | 内容 |
| --- | --- |
| `run.sh` | 再現手順（3つの主張を閉包とファイルの両方で確認） |
| `raw/head.txt` | 計測した版とToolchain |
| `raw/pre-status.txt` | 計測前の作業領域（0行） |
| `raw/claim-a.txt`・`raw/claim-b.txt`・`raw/claim-c.txt` | 主張ごとの結果 |
| `raw/servicehost-closure-callers.txt` | 空（自分以外の閉包に含むパッケージが0件） |
| `raw/servicehost-file-importers.txt` | import するファイル3件（すべて試験） |
| `raw/servicehost-file-importers-nontest.txt` | 空（非試験0件） |

約1MBの生依存ダンプは証拠ディレクトリに置かず、`mktemp` の作業領域で要約後すぐ削除しています。

## 5. 生データのSHA-256

| ファイル | SHA-256 | バイト数 |
| --- | --- | --- |
| `run.sh` | `c5405e4f39c6467ec81c224803e7f963516aa5ba809810bd0fb87ed631ba50d7` | 3898 |
| `raw/head.txt` | `f7789e390ebd40e9ee5a7e349c6a6a6b83d693abc5822cfee4fda753a816e0ad` | 73 |
| `raw/pre-status.txt` | `e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855` | 0 |
| `raw/claim-a.txt` | `8afea8def4010592895b6b8c7e886ab5dcc44f6cc32452509a65dfc40728726a` | 116 |
| `raw/claim-b.txt` | `d9b32adeac3e876273593293bc6cdaa2bf309a413c9ff0ed66e9da64eb5adf50` | 138 |
| `raw/claim-c.txt` | `f8979a1393e626db15ccd84fb39fdf3c96da7a122b87126d4f0c7192fb6dfcf7` | 152 |
| `raw/servicehost-closure-callers.txt` | `e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855` | 0 |
| `raw/servicehost-file-importers.txt` | `86b0c658f44738fb359c82f3490b691be63c6f2ade9187bc5f5a92c484377fea` | 219 |
| `raw/servicehost-file-importers-nontest.txt` | `e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855` | 0 |
| `raw/go-list.err.txt` | `e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855` | 0 |

`raw/go-list.err.txt` が0バイトであることは、`go list` がエラーを出さずに全パッケージを列挙できたことを示します。
