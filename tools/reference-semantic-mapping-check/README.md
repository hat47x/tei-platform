# reference-semantic-mapping-check（#2095）

業務上の意味の定義と、物理項目(列・属性)の対応について、次の三つを機械で検査する読取り専用の参照です。

1. 各要素から受理済みの定義へ遡る道が、高々一つであること。二つ以上に一致する要素は`ambiguous`として警告し、報告全体を`rejected`にします。
2. 受理済みの定義がない要素を、`unfilled`として一覧に出すこと。推測では補いません。
3. 定義を足したとき、変わる要素が、その定義に結び付く要素に限られること。

```text
go run ./tools/reference-semantic-mapping-check/cmd/tei-semantic-mapping-check \
  -input tools/reference-semantic-mapping-check/testdata/input.json
go run ./tools/reference-semantic-mapping-check/cmd/tei-semantic-mapping-check \
  -input <追加前>.json -after <追加後>.json
```

## 入力

`definitions`に定義の`id`と`name`、`elements`に要素の`id`・`name`・実装上の観測位置`observedIn`・対応`mappings`を並べます。対応には`status`(`accepted`または`candidate`)、`origin`(`human`または`ai`)を持たせ、`accepted`には受理した主体`acceptedBy`が必須です。`candidate`は提案であり、結果に影響せず、`candidates`として別に表示します(#267)。

## 報告

| 状態 | 意味 |
| --- | --- |
| `complete` | 全要素が受理済みの定義を一つだけ持つ |
| `incomplete` | 未記入の要素がある。曖昧な要素はない |
| `rejected` | 受理済みの定義が複数ある要素がある |

要素ごとに、実装から分かる事実`what`(`observed`または`not-observed`)と、人が受理した定義`why`を別に記録します。`observedIn`があっても`why`は埋まりません。終了コードは`complete`(比較では`within-scope`)のときだけ0です。

`-after`を指定すると、追加前後の解決結果を比べます。要素の集合が同じで、定義が追加だけの入力を対象とし、変わった要素のうち追加した定義に結び付かないものを`outsideScope`に出して`rejected`にします。定義の削除や要素集合の変更を含む入力は、比較せずに拒否します。

## 範囲と非主張

- `testdata/`は合成した例です。実在のデータ辞書、カタログ、Reference workspaceのDataとは接続していません。受入条件のうち「既存ReferenceのData 10〜20項目での確認」は未実施です。
- 定義の作成・受理・提案、要素の自動対応づけはしません。
- 定義の置き場(#1808)、意味参照(#1120)の決定には依存せず、決定もしません。
