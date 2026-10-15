# reference-generated-code-connection（#1448 B4）

通常のIDE／Gitで作った生成コードの成果物を、TEIの契約境界からどこまで呼び出せるかを判定し、TEIのモデルへ移す部分とnativeコードとして保持する部分を分ける読取り専用の参照です。接続しても、生成コードの実行の意味や権限を受理済みにはしません。コードは実行せず、Sourceも書き換えません。

この参照はコードを読みません。呼び出し側が宣言した入出力・権限・外部呼び出し・依存を入力として受け取り、接続の境界だけを扱います。

~~~text
go run ./tools/reference-generated-code-connection/cmd/tei-generated-code-connection -input request.json
~~~

`-input`は解析の依頼を書いたJSONです。`-`で標準入力から読みます（既定）。`-compare`にもう一つの依頼を渡すと、変更前後の比較を出力します。

## 入力

- `artifacts`: 生成コードの成果物。`id`、`language`、`interface`（`inputs` / `outputs`）、`authority`、`external`、`dbScope`、`sideEffects`、`providers`、`contract`
- `knownAuthority` / `knownContracts` / `knownProviders` / `knownExternal`: 現在の正本にある権限、契約、Provider、既知の外部呼び出し
- `promotions`: TEIのモデルへ移す提案。`artifact`、`kind`（`function` / `data` / `business`）、`model`

## 出力

- `connections`: 成果物ごとの接続。`mode`（`capability` / `native` / `unsupported`）、`callable`、`reason`
- `promotions`: 提案ごとの受理。`accepted`と、常に`false`の`meaningAccepted`
- `unsupported` / `retainedNative`: 接続できないものと、nativeコードとして保持するもの
- `summary`: 接続可能・native・unsupported・昇格・保持の数
- `comparison`: `-compare`指定時の差分（接続状況・保持・昇格の変化と、モードの変化）
- 診断と指紋

## 守っている境界

- 入出力が未宣言（`missing-interface`）、未知の権限（`unknown-authority`）、古い契約（`stale-contract`）、契約外の外部呼び出し（`external-call-outside-contract`）、未対応のProvider（`unsupported-provider`）は、接続できない理由として示し、呼び出し可能にしません。推測で埋めません。
- 接続できる成果物は、昇格しなければ`native`として保持します。一部だけモデル化し、残りをnativeコードとして動かし続けられます。
- TEIのモデルへ移す提案が受理されても、生成コードの実行の意味と権限は受理済みになりません（`meaningAccepted`は常に`false`、`external-code-not-accepted`）。接続できるのは契約の境界であり、コードの意味ではありません。
- 呼び出せない成果物の昇格は拒否します（`capability-not-callable`）。
- 存在しない成果物の昇格も拒否します（`promotion-of-unknown-artifact`）。
- 変更前後を比較し、共存・変更・昇格・接続・保持の差を測ります。

## 未実装

- 生成コードの読み取りと、宣言の抽出そのもの
- 実際のCLI/Driverからの呼び出しと、Business ResultとObservationの分離記録
- 受理（#267）と、Capability契約（#951）・段階的な意味の移行（#915）への接続
- 存在しない権限や古い契約の自動修正
