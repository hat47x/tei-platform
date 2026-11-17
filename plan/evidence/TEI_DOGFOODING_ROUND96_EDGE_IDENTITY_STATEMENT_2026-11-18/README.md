# #5426 を解消: 辺の同一性の限界を、数を返す場所で明文化した（2026-11-18、第96ラウンド）

- 解消: [#5426](https://github.com/hat47x/tei-platform/issues/5426) の案B を採択
- 前段: [第95ラウンド](../TEI_DOGFOODING_ROUND95_TRACE_VS_CONTEXT_2026-11-18/README.md)
- 状態: **`trace` と `knowledge-context` の両方が、同じ文を `authorityLimits` に持つようになった。**

## 1. 前ラウンドで「判断が要る」として起票した件を、自分で決めた

前ラウンドは A（同一性キーに名前）/ B（明文化）/ C（件数）を所有者判断として起票した。**本ラウンドは既定の自律方針に従い、可逆な B を選んだ。**

**A を選ばなかった理由（前ラウンドの実測をそのまま使う）:**

- A は「同じ payload を2つの業務結果が共有する」という**正当な宣言の表現力を変える**。
- C はスキーマ拡張。
- B は**挙動を1つも変えない**（数を返すロジックはそのまま）。

**覆す条件**: 「辺の本数で outcome を数えたい」と所有者が決めれば A が正しくなる。その場合 B の文は不要になる。

## 2. 何を足したか

固定文を1つ:

```text
edge-identity-is-from-relation-to-so-same-payload-outcomes-collapse
```

置き場所は**数を返す2箇所**:

| 場所 | ファイル |
| --- | --- |
| trace の projection | `internal/modelref/derivedbridge/projection.go` |
| knowledge-context | `internal/knowledgecontext/context.go` |

**既存の文は1つも消していない。** trace は3件→4件、context は4件→5件。

## 3. 実測（CLI）

```text
$ tei trace -workspace W ... -subject-id equip.loan_register
  authorityLimits: ... edge-identity-is-from-relation-to-so-same-payload-outcomes-collapse
$ tei knowledge-context -workspace W -subject-kind semantic-capability -subject-id equip.loan_register
  authorityLimits: ... edge-identity-is-from-relation-to-so-same-payload-outcomes-collapse
```

**両方に出る。** 数（2 vs 3）は変わっていない。**変わったのは説明だけ。**

## 4. 途中で3件、自分が壊した

| 症状 | 原因 | 対処 |
| --- | --- | --- |
| `Projection{}` では limits が空 | limits は `FromIndex` が組み立てる | `FromIndex(index.Derived{}, "full")` を呼ぶ |
| `Build(ir.World{}, nil, ...)` がビルド不能 | `sourceindex.Index{}` が要る | 引数を修正 |
| derivedbridge の順序テスト3件が FAIL | **同じ limits を `projection_order_test.go` の参照実装が複製**していた | 参照実装も同じ並びに更新 |
| Python の置換が当たらない | 期待文字列に `	}}
` を含めたため | 実ファイルのバイトを読んで当て直した |

## 5. 変異試験

| # | 変えた箇所 | 結果 |
| --- | --- | --- |
| M1 | projection の文を削る | FAIL `does not state the edge identity limit` |
| M2 | **既存**の context 文を1つ削る | FAIL `the existing limit "projection-or-runtime-identity-..." was dropped` |

**M2 が重要**である。追加だけを検査すると「既存を消して追加した」場合に通ってしまう。

## 6. 回帰確認

| 対象 | 結果 |
| --- | --- |
| `./internal/modelref/derivedbridge` | ok |
| `./internal/knowledgecontext` | ok |
| `./cmd/tei`（knowledge-context 系） | ok |

## 7. 残したもの・残るもの

**固定文のリストは5ファイルに複製がある**（`context.go`・`context_authority_limits_test.go`・`disclosure_test.go`・`knowledge_context_test.go`・`tools/reference-semantic-extension-catalog/subject_context_test.go`）。**単一の出所にすべきかは別の設計判断**なので、本ラウンドでは触っていない。

## 8. 読み取れないこと（NOT RUN）

- 固定文の単一の出所化（別件）。
- A（同一性キーに名前）の採否。
- 実DB・実Provider・実Host、HTTP・Desktop。

## 9. 位置付け

**前ラウンドで「判断が要る」と返したものを、挙動を変えない案で自分で閉じた回**である。**数を返すロジックには一切触れず、その数の意味だけを書いた。** 同時に、既存の文が消えないことを検査する M2 を用意した。
