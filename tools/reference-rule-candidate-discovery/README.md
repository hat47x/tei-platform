# reference-rule-candidate-discovery（#1446 B2）

入出力の実例から複数の規則仮説を提示し、境界例で仮説が割れる点を示す読取り専用の参照です。実例が少ないうちは単一の規則を確定せず、人が仮説を選ぶまで受理しません。SourceやWorkspaceを書き換えず、規則を受理もしません。

~~~text
go run ./tools/reference-rule-candidate-discovery/cmd/tei-rule-candidate-discovery -input examples.json
~~~

`-input`は実例を書いたJSONです。`-`で標準入力から読みます（既定）。

## 入力

~~~json
{
  "examples": [
    {"id": "e1", "input": "ABC-001", "output": "ABC"},
    {"id": "e2", "input": "XYZ-025", "output": "XYZ"},
    {"id": "e3", "input": "DEF-100", "output": "DEF"}
  ]
}
~~~

## 仮説の形

区切り文字は`-`に固定します。実例から区切り文字の約束を読み取らないためです。

| 仮説 | 内容 |
| --- | --- |
| `prefix-before-first-delimiter` | 最初の区切りより前を残す |
| `prefix-before-last-delimiter` | 最後の区切りより前を残す |
| `leading-alpha` | 先頭の英字の連なりを残す |
| `strip-trailing-digits` | 末尾の数字を除く |
| `identity` | 入力をそのまま返す |
| `literal-mapping` | 観測した組をそのまま引く。未知の入力へ一般化しない |

## 出力

- 各仮説が全実例に一致するか（`fitsAll`）、形式だけの規則か（`formatOnly`）、未知の入力へ一般化できるか（`generalizable`）
- 実例から作った境界入力（`probes`）と、一致する仮説ごとの出力。仮説で出力が割れる場合は`disagree`
- `ambiguous`と`reviewRequired`
- 診断
- 指紋（`fingerprint`）

## 守っている境界

- `reviewRequired`は常に`true`です。実例が3件以上あり、一般化できる仮説が2つ以上一致する場合は`ambiguous`とし、単一の規則を勝手に確定しません（`multiple-hypotheses-fit`）。
- 実例が3件未満の場合は、追加の実例を求めます（`insufficient-examples`）。
- 境界入力は、区切りを除いた形と、最後の区切りへ`-X`を挿し込んだ形です。前者は`prefix-*`と`leading-alpha`を、後者は最初の区切りと最後の区切りを分けます。自動生成した境界入力であり、実際の業務データではありません。
- どの仮説も`formatOnly`です。文字列の形式は、業務上の識別規則や認可を決めません。`Confirm`は`format`以外の意味の主張（`business-identity`、`authorization`）を拒否します（`format-cannot-establish-*`）。
- `Confirm`は人が選んだ仮説と、その根拠になった実例のIDを結び付けます。存在しない実例を指す選択や、実例のない選択は拒否します。
- `Execute`は選んだ仮説を、AIを使わず決定的に新しい入力へ適用します。区切りが無い入力など、形が答えを持たない場合は`ok=false`を返します。規則を実行権限や受理済みの意味へ直結しません。

## 選んだ仮説を適用する

~~~text
go run ./tools/reference-rule-candidate-discovery/cmd/tei-rule-candidate-discovery \
  -input examples.json -select prefix-before-first-delimiter -claim format -probe AB-C-001
~~~

`-select`を指定すると`Confirm`の結果を、`-probe`も指定すると`Execute`の結果を出力します。`-claim`の既定は`format`です。

## 未実装

- 受理（#267）と、受理済みの規則をDSL／変換（#988、#1063）へ接続する経路
- 型付きDataset、Snapshot、Scenario（#1131）との結線
- 受理済みScenarioからの回帰試験生成（#1180）
- 実例の出所確認と、AIが提案した候補と人が確定した規則の区別の記録
- 区切り文字や仮説の形の拡張。現在は`-`と上表の形に限ります
