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

# 比較と同時に、何を測ったかの記録を残す
go run ./tools/reference-cognition-compare -baseline -model -provider provider.json \
  -provenance provenance.json \
  -model-artifact /path/to/model.gguf \
  -template-source http://127.0.0.1:8080/props
```

`-provider` は `execution: external` / `network: none` / `settings.endpoint` を宣言した Provider JSON。

## Provenance（何を測ったかの記録）

#5807 / #5809 は「実際の構成・モデルdigest・テンプレート・要求/応答の指紋を保存」を求める。率だけの報告は、未知のシステムについての数字であり、後日の再実行で差分がモデル由来かハーネス由来かを判定できない。

`-provenance` は以下を1ファイルに書き出す（`0600`）。

- ケース集合のダイジェスト
- モデル成果物の SHA-256 とバイト数（`-model-artifact`）
- チャットテンプレートの SHA-256 と取得元（`-template-source`）
- ケースごとの要求ダイジェスト・応答ダイジェスト・実時間

**読めなかった事実は空欄にせず `unrecorded` に列挙する。** 空欄は「該当なし」と読めるが、真相は「観測できなかった」のことが多いため。成果物が読めないときはゼロダイジェストを書かない。異なる2モデルが同一に見え、存在しない provenance を主張することになる。

### 実測の provenance（2026-11-25、CPUのみ）

```
caseSetDigest  sha256:0f37b2e9729cd618b4ae9c88563be23f75bc898c974b83ecf105c622f7852beb
modelDigest    sha256:cbdf3fa9dc50a4124977980f37da4f1fa764b0cfe800d275cd43d26ce2a81af4 (17,627,940,768 bytes)
templateDigest sha256:a04653367fbabaf38e91c426b8572ec18b1342dee6675caa8ce9564b6877da94
unrecorded     []
```

基線側はモデル成果物・テンプレートを持たないため、この2項目は空欄になる。基線が「決定的」であることは、同一入力で同一応答ダイジェストを返すことで別途確認している。

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
