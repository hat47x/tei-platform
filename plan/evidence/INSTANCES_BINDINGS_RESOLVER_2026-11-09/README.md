# Instances と Bindings には、既に解決器がある（2026-11-09）

- 対象: [判断材料の第12節](../../workstreams/COMPOSITION_CONNECT_POINT_BRIEF_2026-11.md)、[6つの宣言の調査](../BINDING_FORMS_CONSUMER_SURVEY_2026-11-09/README.md)
- 位置づけ: **読み取り専用の調査**。何も変更しません。
- 計測した版: `raw/head.txt`（`origin/main` `e25fb0fe`）。

判断材料の第12節は、`Instances` と `Bindings` について「解析・索引・検証の実読者がいる」とし、**「専用Resolver／実実行を要するとまだ判断しない」**と書いています。この記録は、**解決器が既にあり、製品コードがそれを呼んで実行まで届いている**ことを示します。

## 1. 既にある解決器

```text
== does a resolver over Instances+Bindings already exist? ==
  internal/realization/contract_query_binding.go:42  func ResolveContractQueryExecutor(

== the resolved value it returns ==
  type ResolvedContractQueryExecutor struct {
      Binding    ContractQueryExecutorBinding `json:"binding"`
      Instance   environment.DatabaseInstance `json:"instance"`
      Capability plugin.ResolvedCapability    `json:"capability"`
  }
```

**Binding と Instance と Capability を1つの値に束ねて返します。**

## 2. 製品コードが呼んでいる

```text
== non-test product call sites ==
  internal/driver/header_detail_contract_query_environment.go:67
  internal/driver/reference_contract_query_environment.go:104
```

## 3. 解決の後に実行まで届く

`reference_contract_query_environment.go` は、解決した値で **3段**を続けます。

1. `registry.ExecutorForContractQuery(selected)` で実行器を引く（無ければエラー）
2. database provider が既知で汎用でないことを確かめる
3. **`selected.Instance.Provider` と実行器の provider の一致**を確かめる

**つまり `Instances` と `Bindings` は、宣言 → 解決 → 実行器の選択 → 実行、まで繋がっています。**

## 4. この観測が示すこと

- **第12節の「Resolver を要するとまだ判断しない」は、控えめすぎます。** 解決器は既にあり、製品の2経路がそれを呼んで実行まで届いています。
- **形は C-3 と同じです。** C-3 は `RouteCommand` の解決結果を `BuildPlan` が認めるか確かめましたが、こちらは `ResolveContractQueryExecutor` が Instance と Capability を束ね、実行器の provider 一致まで確かめます。
- **ただし対象は Contract Query に限られます。** `Bindings` は Application と Database instance の対応一般ですが、この解決器は **Contract Query の実行器**を選ぶ用途です。**「Instances/Bindings 一般の解決器がある」とは言えません。**

## 5. この記録が示さないこと

- **すべての Binding 用途に解決器があること**。確認したのは Contract Query の1用途です。
- **実 Database への接続**。`DatabaseInstance` は宣言上の実体で、実接続は driver の外です。
- **第12節の表を書き換えるべきか**。この記録は事実を示すだけで、表の表現の変更は判断ではありません（ただし次に触る人が参照できるよう、この記録を残します）。
- 製品 CLI へ繋ぐべきか（C-1〜C-4 は未決）。

## 6. 生データ

| ファイル | 内容 |
| --- | --- |
| `run.sh` | 再現手順（解決器の存在・呼出し元・検査内容の列挙） |
| `raw/head.txt` | 計測した版とToolchain |
| `raw/pre-status.txt` | 計測前の作業領域（0バイト＝クリーン） |
| `raw/anchors.txt` | 上の3節の出力 |

## 7. 生データのSHA-256

| ファイル | SHA-256 | バイト数 |
| --- | --- | --- |
| `run.sh` | `021d3bb8ff7c0b33dc82386fd9d1186ce630bab96db07f00b1c8c9578aaacf14` | 2138 |
| `raw/head.txt` | `3e7826618a0c32afa7a829c982bfea5bf811d23ec89ded8405246149e8aa9b4a` | 73 |
| `raw/pre-status.txt` | `e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855` | 0 |
| `raw/anchors.txt` | `431257db69cc7f2949e6da3b40901a1ade3d8fe2f5c37ab0d52f9d9cb47707c5` | 1230 |
