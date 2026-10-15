# reference-procedure-formalization（#1440 B6）

業務手順書の文章から作った意味候補を受け取り、形式化を妨げる不足と矛盾を、根拠の原文範囲を伴う質問として示す読取り専用の参照です。確定した回答は原文と別に保持し、原文の版が変わった回答を再確認へ戻します。Sourceや正本を書き換えず、候補を受理しません。

この参照は文章を読み取りません。人が、または生成AIが提案した候補を入力として受け取り、その提案を受理済みの意味から分ける境界だけを扱います。

~~~text
go run ./tools/reference-procedure-formalization/cmd/tei-procedure-formalization -input request.json
~~~

`-input`は解析の依頼を書いたJSONです。`-`で標準入力から読みます（既定）。

## 入力

- `text`: 手順書の一つの版。`id`、`revision`、`segments`（`id`と`text`）
- `candidates`: 意味の候補。`id`、`kind`（`actor` / `action` / `trigger` / `condition` / `data` / `function` / `exception` / `result`）、`segment`（原文範囲）、`subject`、`value`、`stated`
- `answers`: 質問への回答。`question`、`segment`、`revision`、`value`、`by`
- `knownFunctions`: 既知の機能の識別子

`stated`は、原文にその記載があるかを示します。`false`はAIなどによる補完候補であり、回答がないかぎり受理へ進めません。

## 出力

- `questions`: 不足と矛盾。`code`、`subject`、`segment`、根拠の候補ID（`evidence`）、止められる候補ID（`blocks`）、回答の有無
- `accepted`: 受理できる候補（常に`stated`）
- `blocked`: 受理できない候補と理由（`inferred-not-accepted` / `candidate-anchors-unknown-segment` / `blocked-by-question`）
- `answerStates`: 回答が現在の版で使えるか（`applicable` / `stale-revision` / `source-missing`）
- 診断と、入力の指紋（`fingerprint`）

## 示す質問

| `code` | 条件 |
| --- | --- |
| `missing-<kind>` | 原文にある候補の`value`が空。閾値や期限の未確定をここで示す |
| `missing-actor` | 原文にある`action`に、同じ範囲・同じ対象の`actor`が無い |
| `contradictory-candidates` | 同じ対象の`condition`に、異なる値が二つ以上ある |
| `unknown-function-reference` | 原文にある`function`参照が`knownFunctions`に無い |

## 守っている境界

- 空の値は質問にします。閾値や期限を勝手に補いません。
- 矛盾はどちらかを選ばず、両方を止めます。
- 補完候補（`stated=false`）は、回答があっても`inferred-not-accepted`のままです。回答は原文の不足を埋めるものであり、AIの補完を受理に変えるものではありません。
- 未確定の条件が止めるのは、同じ範囲・同じ対象の操作だけです。手順の他の部分は受理を進めます。これを部分的受理として示します。
- 回答は原文と別に保持し、`segment`と`revision`へ結び付けます。版が変わった回答は`stale-revision`、範囲が消えた回答は`source-missing`として再確認へ戻し、古い回答で候補を解放しません。
- 同じ入力と確定回答から、AIを使わずに同じ報告を再現できます（`fingerprint`）。

## 未実装

- 文章からの候補抽出そのもの。候補は依頼側が用意します
- 受理（#267）と、受理済みの業務手順（#1125）・三視点表示（#1124）への接続
- 原文の範囲と候補の対応の永続化、再読込後の逆引き
- 質問の優先順位付け（業務上の重要性、下流の依存数、回答負担）
- 文章改訂時の差分と影響の提示
