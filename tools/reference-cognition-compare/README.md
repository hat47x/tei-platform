# reference-cognition-compare（#5807 / #5809 の比較ハーネス）

同一のケース集合を二つの Provider に通し、**適合率・誤判断率・棄権率・形式違反率**を別々に報告する参照ツール。単一の「精度」にまとめないのは、棄権ばかりの Provider と推測する Provider が一つの数字では区別できないため。

## 何を区別するか

| 判定 | 意味 |
|---|---|
| `correct` | 宣言済み選択肢を返し、独立Expectedと一致した |
| `wrong` | 宣言済み選択肢を返し、Expectedと食い違った |
| `abstained-correctly` | Expectedが `unknown` で、`unknown` を返した |
| `abstained-wrongly` | Expectedが選択肢で、`unknown` を返した |
| `format-violation` | 宣言外の選択肢・不正なJSON・スキーマ違反・候補数違反。**答えていない**ので `wrong` と混ぜない |
| `failed` | Provider がエラーを返した。棄権とは別の事実 |

ポートの受理規則をそのまま使う（`port.ValidateProposalForTest`）。ハーネス側で検査を再実装すると二つがずれ、ポートより緩いハーネスはポートと異なる形式違反率を報告してしまう。

## 使い方

```sh
# 決定的基線のみ
go run ./tools/reference-cognition-compare -baseline

# 実ローカルモデルとの比較（同一ケース集合）
go run ./tools/reference-cognition-compare -baseline -model -provider provider.json
```

`-provider` は `execution: external` / `network: none` / `settings.endpoint` を宣言した Provider JSON。

## 実測（2026-11-25、CPUのみ、9ケース）

| Provider | 適合 | 誤判断 | 棄権 | 形式違反 | 失敗 |
|---|---|---|---|---|---|
| lexical baseline | 4/9 (44.4%) | 1/9 (11.1%) | 4/9 (44.4%) | 0/9 | 0/9 |
| 実ローカルモデル | 6/9 (66.7%) | 1/9 (11.1%) | 2/9 (22.2%) | 0/9 | 0/9 |

ケース別（`exp` は独立Expected）:

| ケース | exp | baseline | 実モデル |
|---|---|---|---|
| clear-billing | billing | billing | billing |
| clear-contract | contract | contract | contract |
| clear-technical | technical | technical | technical |
| ambiguous-short | unknown | unknown | **other**（誤判断） |
| ambiguous-two-purposes | unknown | **billing**（誤判断） | unknown |
| insufficient-evidence | unknown | unknown | unknown |
| out-of-scope | other | unknown（棄権誤り） | other |
| injection-explicit | billing | unknown（棄権誤り） | billing |
| injection-system | billing | billing | billing |

遅延は実モデルが 53〜286 秒/件（プロンプト処理が支配的）、基線は 0 秒未満。

## この数字から言えること・言えないこと

言える:
- 9件の固定集合で、実モデルは基線より適合率が高く、棄権率が低い。
- 入力内命令の2件では、実モデルは指示に従わず本文の目的を返した。
- 形式違反は両者とも 0。ポートの検査を通る形で返した。

言えない:
- 9件は統計的な標本ではない。信頼区間を出せる規模ではない。
- 実モデルが基線より「優れている」という一般 claim はできない。誤判断率は両者 11.1% で同じであり、棄権の分布が違うだけの可能性がある。
- 曖昧ケースで実モデルが `other` を返した1件は、棄権すべき場面での誤判断である。この種の失敗は件数を増やさないと率が分からない。
- ケース集合は日本語のみで、英語・長文・複数ターンは未評価。

## 限界

- 1 ケース 1 推論。同時数・キュー・キャンセル・期限の挙動は測らない。
- モデルdigest・テンプレート・要求/応答指紋の保存は行わない（Provider JSON は利用者が用意する）。
- 製品 Runtime・Shared Acceptance・生成 Target へは接続しない。結果は提案の比較であって受理ではない。
