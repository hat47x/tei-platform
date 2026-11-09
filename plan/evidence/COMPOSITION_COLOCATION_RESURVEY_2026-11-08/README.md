# 構成の二つの層は、いまも同じ場所に居ないか（2026-11-08 再調査）

- 対象: [構成Kernel移行Gate](../../gate/COMPOSITION_KERNEL_MIGRATION_GATE_2026-10.md)と、先行の[構成の次の縁の静的調査](../COMPOSITION_NEXT_EDGE_STATIC_SURVEY_2026-11-07/README.md)
- 位置づけ: **先行調査の主な数を、現在の main で取り直した記録**。先行記録は当時の観測として残し、訂正しません。
- 計測した版: `raw/head.txt`（`origin/main` `61e62de7` の作業ツリー）。

先行調査（2026-11-07）は「`internal/operations` と `internal/composition` を同じファイルで import するものは0件」「`internal/servicehost` をimportする非試験ファイルは0件」と報告しました。この記録は、その2つが**いまも成り立つか**を確かめます。あわせて、先行調査が**ファイル単位だけ**で数えていた点を補い、**パッケージの依存閉包**でも数えます。

## 1. 観測

```text
== file level ==
go files scanned: 2976
files importing operations: 57
files importing both operations and composition: 0
non-test files importing servicehost: 0

== package level ==
main packages: 177
packages whose closure includes composition: 12
packages whose closure includes operations: 20
packages whose closure includes servicehost: 1
packages whose closure includes BOTH composition and operations: 1
main packages among those: 0
main packages whose closure includes composition: 4
main packages whose closure includes operations: 6
```

| 先行（2026-11-07） | 今回（2026-11-08） |
| --- | --- |
| Goファイル 2,876 | 2,976 |
| 両方をimportするファイル **0** | **0** |
| main パッケージ 167 | **177** |
| 両方を含む main パッケージ **0** | **0** |
| `servicehost` の非試験 importer **0** | **0** |

先行の2つの数は**どちらも成り立っています**。

## 2. ファイル単位だけでは足りない理由（今回追加した観測）

ファイル単位の「0」は、**同じパッケージ内の別ファイルがそれぞれ片方をimportしている場合を見落とします**。パッケージの依存閉包で数えると:

- **`tei/reference/interface-adapter` が、両方の閉包に入る唯一のパッケージ**です。ただし**直接のimportはどちらも無く**、`reference/web-composition/go-host`（`http.go` が `webcomposition.NewHost` を呼ぶ）経由で間接的に届いています。
- つまり「同じ場所に居ない」は**ファイル単位では正しいが、パッケージ単位では1件だけ反例がある**。ただしその1件は、参照実装が別の参照実装を呼ぶ**参照同士の接続**であり、中核の2層が混ざったわけではありません。
- `composition` の閉包に入る main パッケージは4件、`operations` は6件で、**両方に入る main パッケージは0件**です。先行調査の「0」は、パッケージ単位でも維持されています。

## 3. この観測が示すこと

- 先行調査の結論「A（操作面）と B（構成）は同じ場所に居ない」は、**ファイル単位でもパッケージ単位でも維持**されています。
- ただし**パッケージ単位で1件の反例**（`reference/interface-adapter`）があり、これは**間接的な参照同士の接続**です。ファイル単位の数だけを見ていると、この1件は見えません。
- `servicehost` に**非試験の呼出し元が無い**ことも変わっていません。構成Kernelの受け側は、まだ本番の入口に接続されていません。

## 4. この記録が示さないこと

- **`reference/interface-adapter` の接続が問題であること**。参照実装同士の接続であり、製品の経路ではありません。
- **`servicehost` を接続すべきか**。Gateの判定は[構成Kernel移行Gate](../../gate/COMPOSITION_KERNEL_MIGRATION_GATE_2026-10.md)の担当です。
- **先行調査の誤り**。先行調査はファイル単位と明記しており、その範囲では正しいまです。
- **実行時の挙動**。静的なimportと依存閉包の走査のみです。

## 5. 生データ

| ファイル | 内容 |
| --- | --- |
| `run.sh` | 再現手順（ファイル単位とパッケージ単位の2つの数え方） |
| `raw/head.txt` | 計測した版とToolchain |
| `raw/summary.txt` | 上の表を作る元の一覧 |
| `raw/co-located-files.txt`・`raw/servicehost-importers.txt` | 空（該当0件であることの証拠） |
| `raw/main-pkgs-both.txt` | 空（mainパッケージの両方所属が0件） |
| `raw/pkgs-both.txt` | 唯一の該当（`tei/reference/interface-adapter`） |
| `raw/pkgs-composition.txt`・`raw/pkgs-operations.txt`・`raw/pkgs-servicehost.txt` | 閉包ごとのパッケージ一覧 |
| `raw/main-pkgs-composition.txt`・`raw/main-pkgs-operations.txt` | mainパッケージの内訳 |
| `raw/operations-importers.txt`・`raw/main-packages.txt` | 分母 |

## 6. 生データのSHA-256

| ファイル | SHA-256 | バイト数 |
| --- | --- | --- |
| `run.sh` | `b4c70c71d9de38e7cc4c02fb8f5a3067fba2ae05ccd120c265a01dbfdf2443b7` | 4535 |
| `raw/head.txt` | `001f0b579ba9c119064b12bb9f0a6ba796b56441694bf38b6be6bc43a33dbf18` | 73 |
| `raw/pre-status.txt` | `44390fc6913a79e2efbded2c2311693da0658bfeae80458c72de43bdb5f56d1e` | 23 |
| `raw/summary.txt` | `9a1105a12c4d994ee6d8ee2bbff50de0add2ea6d384982430503717d173ebf66` | 543 |
| `raw/pkgs-both.txt` | `85763c157f48a2381cb4de87b79459977c713a0dccec0eaaf85b4482b7c235ac` | 32 |
| `raw/pkgs-servicehost.txt` | `44a901a38a869dacf698e5fede34fff3783e9688eec9f67cc9672b210dec8936` | 25 |
| `raw/pkgs-composition.txt` | `e8e5c73c3a8298f6a207fb7dbe56372dd1c1361267a7343e3f55e2f162fc1018` | 462 |
| `raw/pkgs-operations.txt` | `d13ae15c467986cfdc6889ce425740fdc60ebca7637035cc9d5e59c0b6bd3a2b` | 748 |
| `raw/main-pkgs-composition.txt` | `c4be2302d07b37046caec84fb66bce7f89039121b337b535b03faa7d26d4efb0` | 169 |
| `raw/main-pkgs-operations.txt` | `2264a2f86ddd58f626d3905a166b55e7284b78d161d41354a53cfd31ab69c9db` | 292 |
| `raw/main-packages.txt` | `88aa71c3f1ef695ed42572ba9274b8e002e7711cacd80ff1f37e84cfbe051114` | 9769 |
| `raw/operations-importers.txt` | `23060316ff650c9b680285f0c0501c977aa34db36e1967ebe92558b4c0ce4fa5` | 2519 |

`raw/pkg-deps.txt`（約1MB）は**証拠ディレクトリに置かず**、`mktemp`の作業領域に置いて要約後すぐ削除します。`raw/pkg-deps.err.txt` は0バイトです。`raw/main-packages.txt` と `raw/main-packages-sorted.txt` は同じSHA-256です（並べ替えのみ）。
