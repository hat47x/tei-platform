# reference-target-profile（#2019）

同じ受理済み契約・fixture・生成器候補を、複数のTarget Profileで実行し、profileの宣言と実行結果を一つのreceiptへ記録する参照です。組込み/OS寄りのone-shot profileと、サーバ向けのlong-running profileを並べ、意味の検証が同じであることを確かめます。

```text
go run ./tools/reference-target-profile/cmd/tei-target-profile \
  validate -profile tools/reference-target-profile/testdata/server.json

go run ./tools/reference-target-profile/cmd/tei-target-profile \
  run -profile tools/reference-target-profile/testdata/server.json \
  -workspace reference/workspace \
  -contract tools/reference-generator-selection/testdata/line-amount/contract.json \
  -fixtures tools/reference-generator-selection/testdata/line-amount/fixtures.json \
  -candidates tools/reference-generator-selection/testdata/candidates
```

## Profileの項目

| 項目 | 内容 |
| --- | --- |
| `processModel` | `oneshot`、`long-running`、`batch` |
| `storage` | `none`、`filesystem`、`sqlite`、`remote` |
| `network` | `none`、`loopback`、`allowed` |
| `update` | `replace`、`in-place`、`rolling` |
| `memoryLimitBytes` | 正の整数 |
| `observability` | 観測手段の一覧 |
| `unsupported` | このprofileで扱わない機能 |
| `generatorId` | このprofileで選ぶ生成器のid（任意） |

`testdata/server.json`はlong-running・filesystem・network allowed・rolling updateで`template`を選び、`testdata/embedded.json`はone-shot・sqlite・network none・replace updateで`alternative`を選びます。

## 実行とreceipt

`run`は次を一つのreceiptへまとめます。

- profileのid・versionと全項目
- 選択した契約・fixture・候補の選抜receipt
- 実行した制約と、実際に強制した方法（`enforcement`）
- profileが扱わない機能（`unsupported`）
- 同じfixtureで合格したか

memory limitは`GOMEMLIMIT`としてselectionのサブプロセスへ渡します。これはGo runtimeのsoft limitであり、OSや実機のメモリ制約を証明しません。

## 守っている境界

- profileは意味を変えない。契約・fixture・候補は同じものを使う。
- profileは`generatorId`で生成器を一つ選ぶ。選ばれなかった候補は`not selected by target profile`として拒否し、別の生成器へ暗黙にフォールバックしない。
- `processModel`・`storage`・`network`・`update`・`observability`は**宣言**であり、このReferenceでは強制していない。receiptの`enforcement`に`declared-only`と記録する。
- profileの合格は、実機・実OS・実Targetでの動作を証明しない。
- 組込み固有のリアルタイム性・安全認証・電源断耐性は扱わない。

## 未実装

- OS・CPU・RTOS別の実行環境と、実機での資源測定。
- `network`・`storage`・`update`の強制。
- 実Targetへの配備、ロールバック、長期運用。
