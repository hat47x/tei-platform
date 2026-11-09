# database-provider の選択を、技術名ではなく能力で解決する（2026-11-08）

- 対象: [`docs/plugins.md` の「利用する機能の解決」](../../../docs/plugins.md)と、先行の[宣言されたプラグイン能力は環境から選ばれているか](../PLUGIN_CAPABILITY_COVERAGE_REFERENCE_2026-11-08/README.md)
- 位置づけ: **参照実装の修正と実行記録**。選択の意味を変えるものではない。
- 計測した版: `raw/head.txt`（`origin/main` `bffb7e3f` の作業ツリー）。

先行の記録は、`database-provider` の2件を「数え方の限界」として**数えない**選択をしました。環境の `database-instance` が指すのは `postgres` という**提供技術名**で、能力ID `database.postgres` ではないためです。

しかし `internal/validation/validation.go` は、同じ問いを**すでに解決しています**。`Capability.SupportsDatabaseProvider(instance.Provider)` で、能力が宣言した対応技術の集合と instance の provider を突き合わせ、0件なら `DATABASE_PROVIDER_PLUGIN_MISSING`、2件以上なら `DATABASE_PROVIDER_PLUGIN_AMBIGUOUS` を返します。**技術名と能力IDを文字列で比べるのではなく、能力の宣言した対応表を引いています。**

この記録は、参照実装が同じ解決を使うように直し、結果がどう変わるかを観測します。

## 1. 何を変えたか

`tools/reference-plugin-capability-coverage` の `database-instance` の扱いを、「数えない」から「**既存の解決と同じ問いを聞く**」に変えました。

```go
for _, instance := range env.Instances {
    for _, def := range world.Plugins {
        for _, capability := range def.Provides {
            if capability.Kind != plugin.CapabilityDatabaseProvider { continue }
            if !capability.SupportsDatabaseProvider(instance.Provider) { continue }
            record(string(plugin.CapabilityDatabaseProvider), string(capability.ID), name)
        }
    }
}
```

規則の指紋（`rulePolicy`）も同じ内容に更新しました。**技術名と能力IDを文字列で比べない**ことを明記しています。

## 2. 観測

| | 先行（数えない） | 今回（技術名で解決） |
| --- | --- | --- |
| 宣言 | 12 | 12 |
| **選ばれていない** | **6** | **4** |
| `database.postgres` | unselected | **selected**（development, production） |
| `database.oracle` | unselected | **selected**（development, production） |

残る4件は、`contract-query-executor` 2件・`editor-intelligence` 1件・`migration-provider` 1件です。

## 3. この観測が示すこと

- **`database-provider` は「選ばれていない」のではなく、解決の仕方が違っていただけでした。** 技術名で解決すると、Postgres/Oracle の2件は両環境で選ばれています。
- 先行の記録の「数え方の限界」は**正しい観測**でしたが、**既存の解決を使えば限界は消えます**。参照実装が独自の比較（文字列一致）をしていたのが原因です。
- 残る4件は、先行の記録で整理した3つの意味（能力の Binding が無い、能力の一部だけ選ばれている、契約実行器が選ばれていない）がそのまま当てはまります。

## 4. この記録が示さないこと

- **先行の記録を訂正すること**。先行の記録は当時の実装の観測として正しく、この記録はその後の修正です。先行の記録は履歴として残します。
- **`database-provider` の実装が存在すること**。`selected` は宣言の解決であり、実装の存在・適合・権限の証明ではありません。
- **`contract-query-executor` などが不要であること**。選ばれていないことは欠陥の証明ではありません。
- **実DB・実Provider**。合成の Workspace と読み取りのみです。

## 5. 生データ

| ファイル | 内容 |
| --- | --- |
| `run.sh` | 再現手順 |
| `raw/head.txt` | 計測した版とToolchain |
| `raw/pre-status.txt` | 計測前の作業領域（修正中のツールのみ） |
| `raw/reference-coverage.json` | 宣言と選択の一覧（修正後） |
| `raw/summary.txt` | 上の表を作る元の一覧 |

## 6. 生データのSHA-256

| ファイル | SHA-256 | バイト数 |
| --- | --- | --- |
| `run.sh` | `20d8f9c0038f3106ea69b2c96f1554b55c1570751939063d1b6a268ed8bd135b` | 1378 |
| `raw/head.txt` | `018e92d0194a93ae8323218335874bb9174ee9df03aa2791424cae6ab7c4df31` | 73 |
| `raw/pre-status.txt` | `fc320f94b2d0b5c9763ab35ba54243abe3ca6a400fa51390856cc7f6f32445a3` | 155 |
| `raw/reference-coverage.json` | `e14c7ddeadf48d46d10e7e6875b6445b0f1ca37a85b26a3f0b56c396203983fb` | 3609 |
| `raw/summary.txt` | `f1a8a43535e748a14c454601866afe92d64f0dbcf1077308136ca5cae274e83f` | 1385 |

規則の指紋は `sha256:319077584a43514838adce63403af48fbdd6a68e5f30426cf0d50c0cd557167b` で、先行の記録の指紋とは異なります（数え方が変わったため）。
