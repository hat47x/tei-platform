# `unattended` の主張は、拘束されていなかった（2026-11-10）

- 対象: [`internal/operations/surface.go`](../../../internal/operations/surface.go)、[操作面の判断材料](../../workstreams/OPERATION_SURFACE_DECISION_BRIEF_2026-11.md)の D-3
- 位置づけ: **試験の追加と実行記録**。製品コードの挙動は変えていません。
- 計測した版: `raw/head.txt`（`origin/main` `2387b92b` の作業ツリー＋このPRの追加）。

## 1. 規則の文言

`surface.go` の規則テキストはこう書いています。

```text
autonomy: list and read are unattended because they have no effect
```

**「効果が無いから unattended」**という主張です。しかし、この主張を**検査する仕組みはありませんでした**。`Autonomy` は各 `Operation` に直接代入され、`Summary` は数えるだけです。

## 2. 現状の割り当て（コードを読んだ結果）

| 種別 | Autonomy | 行 |
| --- | --- | --- |
| `list` | `unattended` | 297 |
| `read` | `unattended` | 314 |
| `create` | `attended` | 344 |
| `update` | `attended` | 363 |

**いまは正しく割り当てられています。** 問題は、**書き込み系の種別に `unattended` を誤って与えても、何も止めない**ことです。

## 3. 追加した試験

```text
--- PASS: TestOnlyEffectFreeKindsAreUnattended
--- PASS: TestTheUnattendedCountMatchesTheOperations
```

1. **`TestOnlyEffectFreeKindsAreUnattended`**: 派生した操作面を走査し、`unattended` が付いている種別が `list`/`read` だけであることを確かめます。逆向き（`list`/`read` は必ず `unattended`）も確かめます。**空回りしないよう、少なくとも1件の `unattended` が出ることも要求**しています。
2. **`TestTheUnattendedCountMatchesTheOperations`**: `Summary.Unattended` が操作一覧の実数と一致することを確かめます（要約が一覧から乖離しないように）。

## 4. 違反を検出できることの実測

`create` を `unattended` に変えて実行しました。

```text
marked an effectful kind (create) unattended
--- FAIL: TestOnlyEffectFreeKindsAreUnattended (0.00s)
    surface_autonomy_test.go:27: effectful kind "create" was marked unattended (operation order_entry.create.sales.order)
FAIL
restored surface.go
```

**試験が違反を捕まえることを確認しました。**

## 5. この記録が示すこと

- **規則の文言が主張する「効果が無いから unattended」が、機械的に守られるようになりました。** 将来 `create` や `retire` に `unattended` を付けると、この試験が落ちます。
- **D-3（`unattended` にサービス主体の宣言を必須にするか）の判断材料になります。** いまの `unattended` は**「効果のある種別ではない」ことだけ**を意味し、**誰が判断するか**は何も言っていません。**`human-gated` は一度も出力されません**（`internal/artifact/consistency.go` が値として受理するだけです）。

## 6. この記録が示さないこと

- **D-3 をどう決めるべきか。** 未決のままです。
- **`unattended` が実行時に安全であること。** この試験は派生した**分類**を検査するだけで、実行権限や効果の有無を確かめません。
- **`attended` の側の妥当性。** 「`attended` なら人が判断する」ことを検査していません。
- 実配備、実 Provider。

## 7. 生データ

| ファイル | 内容 |
| --- | --- |
| `run.sh` | 再現手順（試験と違反の再現） |
| `raw/head.txt` | 計測した版とToolchain |
| `raw/pre-status.txt` | 計測前の作業領域 |
| `raw/tests.txt`・`raw/tests-summary.txt` | 試験の出力と要約 |
| `raw/violation.txt` | `create` を `unattended` にしたときの失敗 |

## 8. 生データのSHA-256

| ファイル | SHA-256 | バイト数 |
| --- | --- | --- |
| `run.sh` | `eaee0bfc6c8328990c917781992943367b2e19cb08f39b0d10a00da747a87b09` | 2160 |
| `raw/head.txt` | `c033bd9ea474110e0a08807b429e3a8c888ec2841a3fc54052e9c481c1b092e9` | 73 |
| `raw/pre-status.txt` | `4a28042ca7258f0a603611cd8a9e1b6ad99b6342aa2502c27b2f74c2af412461` | 74 |
| `raw/tests.txt` | `c6ca1ed4da3765365054c503c9f1778123f320789a7fae6423323d48c46dc01e` | 257 |
| `raw/tests-summary.txt` | `744e53b407abf409c7357819f401d786beefa21545b5847e4b6ca16a0f9e39a7` | 252 |
| `raw/violation.txt` | `f79a28caaf94dd0e2bafecc1c8c6dc04989761bfb71ba4256abc7699bf83ded0` | 288 |
