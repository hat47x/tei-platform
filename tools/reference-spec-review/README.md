# Reference Spec Review

このReference toolは、既存のレビュー結果を一つのJSONへ束ねます。仕様の正しさを新しく判定する仕組みではありません。

現在は次の三つを入力にできます。

- `tei knowledge-context`の出力
- `tools/reference-business-procedure-check`の出力
- Shared AcceptanceのPreview / Decision出力

それぞれの判定責務は元のtoolに残します。このtoolが行うのは、未解決事項、cross-viewの差や衝突、古い基準revisionを同じreview packetへ並べることだけです。

## 使い方

customer editのReferenceでは、まずAccepted側の局所contextとBusiness Procedure checkerの結果を取得します。

```bash
go run ./cmd/tei knowledge-context \
  -workspace reference/workspace \
  -subject-kind application-pattern \
  -subject-id order_entry.customer_reference \
  > /tmp/knowledge.json

go run ./tools/reference-business-procedure-check \
  -workspace reference/workspace \
  -fixture reference/business-system-design/order-entry-business-procedure.json \
  -probe reference/business-system-design/order-entry-customer-edit-conflict.json \
  -expect-diagnostic BUSINESS_FUNCTION_DATASOURCE_ACCESS_CONFLICT \
  > /tmp/cross-view.json
```

staleなProposalも同時に確認する場合は、Shared Acceptanceが返したDecision等をJSONで用意します。

```bash
go run ./tools/reference-spec-review \
  -case-id ref-spec-01-customer-edit \
  -knowledge-context /tmp/knowledge.json \
  -cross-view /tmp/cross-view.json \
  -shared-acceptance /tmp/decision.json
```

Shared Acceptanceの入力が無い場合も実行できます。その場合は`missingInputs`へ残り、問題が無いという結論にはなりません。

## 出力

`status`は次の三値です。

- `review-needed`: 入力済みEvidenceの中に確認対象がある
- `partial-no-finding`: 確認対象は見つかっていないが、入力が不足している
- `no-finding-in-provided-evidence`: 三つの入力を確認した範囲ではfindingが無い

最後の状態も仕様の完全性を証明しません。新しいSourceや別の観点を調べれば、追加の問題が見つかる可能性があります。

findingの`class`は、現在次を使います。

- `ambiguity`: Knowledge Contextやcross-view側に未解決事項がある
- `inconsistency`: Accepted側と変更要求の明示的な衝突
- `cross-view-difference`: 二つのviewの参照集合などに差があり、人の確認が必要
- `stale-premise`: Proposalの基準revisionが現在と一致しない
- `source-diagnostic`: 元toolの診断を、このtoolが別の意味へ読み替えずに保持したもの

`validationScopeReview`を含むcross-view JSONも読めます。このfieldの意味はBusiness Procedure checker側が所有します。このtoolは`review-needed`や`unresolved-*`をreview packetへ写すだけです。

## 境界

このtoolはauthoritative sourceを書き換えません。また、次を行いません。

- Knowledge Contextの`unresolved`を自動解決する
- Business Procedure checkerのdiagnosticを別の業務規則へ読み替える
- findingが無いことから仕様の完全性を推論する
- Review結果をAcceptanceやwrite authorityへ昇格する
- 不足した入力を「問題なし」のEvidenceとして扱う

目的は、利用者が実装前に「何を確認すべきか」を一つのpacketで読めるようにし、元のSourceや判定器へ戻れる状態を保つことです。
