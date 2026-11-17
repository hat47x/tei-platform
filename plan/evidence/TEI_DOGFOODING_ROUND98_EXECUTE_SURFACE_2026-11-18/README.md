# 宣言した Application は本当に実行でき、実行結果は自分の限界を自分で言う（2026-11-18、第98ラウンド）

- 起票: **なし**（欠陥を1件も作らなかった回）
- 前段: [第97ラウンド](../TEI_DOGFOODING_ROUND97_BUSINESS_PROCEDURE_2026-11-18/README.md)
- 状態: **`-execute` の経路が成立し、その出力は「何を主張しないか」を自分で述べていた。**

## 1. 目的

目的文の「再びドッグフーディングに戻る」を実行した。**宣言した Application が本当に実行できるか**を、実利用者が最後に問う形で確かめる。

## 2. 固定した入力

| 要素 | 値 |
| --- | --- |
| Source | `c0bb9f61f`（`origin/main`） |
| CLI実行物 | `sha256:f77876cd9716d0b51e19dbd9421f9297285264398d15ff9c3d3fc281a954dad7` |
| 環境 | WSL2 Ubuntu 22.04.5、Go `1.26.11 linux/amd64` |

## 3. 体系的走査を先に行った（新規欠陥ゼロ）

前ラウンドまでの「閉じた語彙なのに有効値を言わない」類型を、**網羅的に走査**した。

```text
$ grep -rn 'valid values are' internal/ cmd/ | grep -v _test   -> 21 件（良い形）
$ grep -rnE 'unknown (%q|"%q"|%s)' ... | grep -v 'valid values' -> 3 件
```

3件のうち `UNKNOWN_APPLICATION_PACKAGE_REQUIREMENT` を追跡したところ、**パーサ側が既に値を名指ししていた**:

```go
// internal/syntax/application_package.go
if kindTok.Text != "semantic" && kindTok.Text != "relation" {
    return out, p.errAt(kindTok, "package requires semantic or relation")
}
```

**残る2件は threeelement の graph 参照で、語彙が閉じていない**（任意の Subject を指せる）ため、値の列挙は成立しない。**この類型は打ち止め**だった。

## 4. 実行の経路は成立した

| Attempt | 応答 | 学んだこと |
| --- | --- | --- |
| 1 | 診断なし（command pattern 受理） | `pattern X command { capability ... }` |
| 2 | `UNBOUND_*` | 環境束縛が要る |
| 3 | `VIEW_FAILED: capability <pattern id>がありません` | **`-command` は capability id**（flag help が "command capability to plan" と書いている） |
| 4 | `INVALID_COMMAND_INPUT: missing inputs: [loan]` | `-input loan=...` |
| 5 | **成功** | `mode: reference` / `committed: true` / `persisted: false` |

## 5. 出力は自分の限界を自分で述べていた（重要な確認）

```text
mode: reference (no durable effects)
authorization: reference-allow-all
outcome: created -> equip.loan
effect: create equip.loan
committed: true (reference unit of work)
persisted: false (persistence: not-applicable, durability: none)
difference-check: not observed (this executor does not enumerate its operations)
```

**`committed: true` だけを読むと「永続化された」と誤読しうる**が、同じ出力が `persisted: false`・`durability: none`・`difference-check: not observed` を**並べて**いる。JSON 側も `appliedUnobserved: true` を持つ。**私はこれを欠陥と呼ばない。**

## 6. 直さなかったもの（記録）

| 観察 | 判断 |
| --- | --- |
| `-command` に pattern id を渡すと `capability <id>がありません` | **その id は実際に capability でない**ので記述は正確。flag help が "command capability to plan" と種別を書いている |
| 一部のエラー文が日本語（JSON API 内） | 非テスト Go に **108 件**あり、リポジトリの文書言語と一致する。**方針の判断なしに一括変換しない** |

**どちらも起票しない。** 前者は誤りではなく、後者は言語方針の判断が要る。

## 7. 読み取れないこと（NOT RUN）

- 実DB への永続化、実 Provider 起動（reference unit of work の外）。
- 日本語エラー文の言語方針（# 未起票）。
- 実Host、HTTP・Desktop。

## 8. 位置付け

**「実行できるか」を最後に問うた回**であり、答えは**成立する**だった。同時に、**出力が自分の限界を自分で述べている**ことを確認した。**欠陥を作らなかったことと、直さないと決めた理由を記録するのも成果**である。
