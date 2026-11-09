# digestが変わったとき、旧いピン留めはどうなるか（CON-12c）（2026-11-09）

- 対象: [判断材料の第12節 CON-12c](../../workstreams/COMPOSITION_CONNECT_POINT_BRIEF_2026-11.md)、[識別子を持つ層の調査](../BINDING_IDENTITY_LAYERS_2026-11-09/README.md)
- 位置づけ: **参照実装と実行記録**。何も変更しません。
- 計測した版: `raw/head.txt`（`origin/main` `dc0856af` の作業ツリー＋このPRの追加）。

前の記録は、**観測と証拠の宣言には digest も epoch も Scope も無く、CON-12c はその2層では問えない**ことを示しました。識別子を持つのは `generation.Binding` と `ServiceProviderSelection` の2層だけです。この記録は、**そのうち `generation.Binding` で CON-12c を実際に問います。**

## 1. 何を作ったか

`tools/reference-digest-change-reuse`（新規・読み取り専用）。既存の `generation.Router` だけを使い、3つの問いに答えます。

1. 変更**前**にピン留めした lease は、どの digest を持つか
2. 変更**後**の新しい取得は、どの digest を持つか
3. 変更**後**も、変更前にピン留めした lease は、どの digest を持つか

変更は `Router.Publish`（世代1 → 世代2、digest を差し替え）で行います。**実配備の切替ではありません。**

## 2. 観測

```text
oldDigest: sha256:0000...0001
newDigest: sha256:0000...0002

before the change                      -> the OLD digest
a new acquisition after the change     -> the NEW digest
the lease pinned before the change     -> the OLD digest

oldLeaseStillPinned:   True
newLeaseUsesNewDigest: True
```

## 3. この観測が示すこと

- **CON-12c は `generation.Binding` の層では満たされています。** 変更後の新しい取得は**新しい digest** を得て、変更前にピン留めした lease は**古い digest のまま**です。旧い binding が新しい作業へ再利用されていません。
- **これは「実行中の呼出しが新しい Artifact へ黙って切り替わらない」ことを意味します。** 逆に、新しい作業が古い digest で走ることもありません。
- **`Publish` は世代番号と contract の同一性を要求します。** テストで、同じ世代番号での差し替えと、contract をまたぐ差し替えが**どちらも拒否される**ことを固定しました。digest だけを変える操作は、この前提の上でしか成立しません。
- **旧世代の `Drain` は、lease が残っている間はタイムアウトします。** 「まだ使われている」ことが観測可能で、前提ではありません。

## 4. この記録が示さないこと

- **観測・証拠の層で CON-12c が満たされること。** その2層には比較項が無く（[前の記録](../BINDING_IDENTITY_LAYERS_2026-11-09/README.md)）、`ServiceProviderSelection` の層も試していません。
- **実配備の切替**。`Publish` はメモリ内の Router を進めただけで、外部 Host・実 Artifact・現在の Grant には触れていません。
- **digest の真正性**。使った digest は構造的に妥当な `sha256:` 文字列で、実ビルドの産物ではありません。
- **Host/Grant epoch**。`generation.Binding` は `Generation` を持ちますが、これは Lease の世代番号で、Grant の epoch ではありません。
- CON-12d（部分成功）。

## 5. 生データ

| ファイル | 内容 |
| --- | --- |
| `run.sh` | 再現手順 |
| `raw/head.txt` | 計測した版とToolchain |
| `raw/pre-status.txt` | 計測前の作業領域 |
| `raw/report.json` | 3つの問いの出力 |
| `raw/summary.txt` | 上の表を作る元の一覧 |

`raw/report.err.txt` は0バイトです。

## 6. 生データのSHA-256

| ファイル | SHA-256 | バイト数 |
| --- | --- | --- |
| `run.sh` | `09a32b20ebc2613d282347fac8e9da39515597e922ab347da16ec8215ab2f033` | 1898 |
| `raw/head.txt` | `ca5451f10649b6f3364b49667480d1aa605a4e536ddfb5bc32e81251279700a6` | 73 |
| `raw/pre-status.txt` | `9d73a0935af7e0a6761e0008e9056f27123f73db8ba7ae199bd885494a8d42a6` | 223 |
| `raw/report.json` | `de92396c9fa36a040900e2198cc6b6600e9c826c02d2a67594b6304a2e709cc4` | 1483 |
| `raw/summary.txt` | `a01d236d997643d508802ee91477f4fb3531006683f99c41a9453966fe81a530` | 490 |

規則の指紋は `sha256:2492daf32562efc27dec61517b7a0e945606546ba3ae53c7cb8bc63720469bf2` です。
