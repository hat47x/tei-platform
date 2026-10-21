# reference-generator-context（#2016 / #2021）

生成器モデルへ渡す文脈を、受理済み契約・固定fixture・Workspace revisionに限定して作るReferenceです。モデルの呼出しや提案の受理は行いません。

```text
go run ./tools/reference-generator-context/cmd/tei-generator-context \
  -workspace reference/workspace \
  -contract tools/reference-generator-selection/testdata/line-amount/contract.json \
  -fixtures tools/reference-generator-selection/testdata/line-amount/fixtures.json \
  -max-bytes 32768 -out /tmp/generator-context.json
```

出力は、そのまま`tei-generator-model -context`や外部コマンドの入力に渡せます。

## 内容

- 契約のid・version・status・digest
- 受理済み属性とprimitiveの組
- 出力primitiveと丸め設定
- fixtureのcontract id・version・digestと全case
- WorkspaceのSource Index revision
- 生成時の制約（属性を追加しない、fixtureに一致する、秘密情報を含めない）
- 最大byte数と実際のbyte数

`max-bytes`を超える場合は、切り詰めずにエラーにします。fixtureや契約を黙って落としません。

## 守っている境界

- 入力は受理済み契約と固定fixture、Workspaceのrevisionだけです。未受理の候補や秘密情報を混ぜません。
- 生成モデルの出力を受理・適用しません。提案の評価は別の境界で行います。
- 文脈のdigestとrevisionを記録し、同じ入力から同じ文脈が得られるようにします。
- byte数の上限は、モデルの入力上限やコストの保証ではありません。

## 未実装

- モデル固有のprompt形式、token数の精密な見積り。
- 契約・fixture以外のSource断片の選択。
- 秘密情報の自動検出・除去。
- 生成結果の受理、Shared Acceptanceへのhandoff。
