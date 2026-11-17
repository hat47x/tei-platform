# 2ファイルの構文エラーは両方reportされ、text と json は5件で一致する（2026-11-18、第99ラウンド）

- 起票: **なし**（欠陥を1件も作らなかった回）。加えて**自分の計器の誤りを1件訂正した。**
- 前段: [第98ラウンド](../TEI_DOGFOODING_ROUND98_EXECUTE_SURFACE_2026-11-18/README.md)
- 状態: **多ファイルの構文収集は成立。text/json の件数も一致。**

## 1. 目的

目的文の「再びドッグフーディングに戻る」を実行した。**2つの独立ファイルがそれぞれ構文エラーを持つとき、両方が報告されるか**を確かめる。古い記録に「そのファイル以外は調べていない」という文があり、**それが今も真か**を検証したかった。

## 2. 固定した入力

| 要素 | 値 |
| --- | --- |
| Source | `02c91aeac`（`origin/main`） |
| CLI実行物 | `sha256:ba6e9a3eb2efd5e01749693216cfb29c51fc367290ae14507fa4ed1d6aa11df9` |
| 環境 | WSL2 Ubuntu 22.04.5、Go `1.26.11 linux/amd64` |

## 3. 実測: 両方reportされた

```text
semantic/probe/a.tei:8:22  error PARSE_ERROR unexpected token "}"
semantic/probe/b.tei:4:31  error PARSE_ERROR expected attribute, relation, candidate-key, ...
note VALIDATION_INCOMPLETE 2件の構文の指摘を同じ読み込みで集めました。…
```

**古い文（"files other than that one were not examined"）はもう出ない。** 代わりに「2件を集めた」と**件数まで言う**。

## 4. 構文エラーは意味検証を止める — そしてそれは正直に書かれている

| 手順 | 結果 |
| --- | --- |
| a.tei に構文エラー、b.tei に意味エラー（未知の型） | **PARSE_ERROR 1件のみ**、`意味検証は実行していません` |
| a.tei の構文エラーを直す | **`UNKNOWN_SEMANTIC_ATTRIBUTE_TYPE` が現れる** |

**対照実験で、note の文が文字通り真であることを確認した**（意味検証は本当に走っていない）。JSON 側も同じ事実を構造化して持つ:

```json
"incomplete": {"reason": "syntax-coverage-unverified",
               "detail": "… coverage is unverified and semantic validation was not performed"}
```

健康なワークスペースでは `incomplete` は**現れない**ことも確認した。

## 5. 自分が誤った計器を使った（訂正）

「text は2件、json は5件」と読み、**モード間で件数が食い違う**と疑った。実際は私の `grep -E "error|warning"` が**行頭一致**で、Subject を持たない診断（`APPLICATION_RELEASE_CONTRACT_NOT_CURRENT` 等）は**行が severity で始まる**ため取りこぼしていた。

生の出力をタブ区切りで読み直すと:

```text
[$1] [warning] HEADER_DETAIL_BINDING_COVERAGE_UNDECLARED
[$1] [error]   UNKNOWN_SEMANTIC_ATTRIBUTE_TYPE
[$1] [APPLICATION_RELEASE_CONTRACT_NOT_CURRENT] …
[$1] [DATABASE_RELEASE_CONTRACT_NOT_CURRENT] …（2件）
```

**text 5行 / json 5件で一致。** 実装は同じ `operations.ValidateWorkspace` を共有しており（`#3420` の「両transportが同じ文書を返す」）、**私の数え方が誤っていた。**

## 6. 既存のテストが既に守っていた

| テスト | 内容 |
| --- | --- |
| `TestValidateWorkspaceCollectsSyntaxWithoutPartialWorld` | **3ファイル**の独立した構文エラーを集め、`AdditionalDirect` 2件・`InspectionIncomplete`・`incomplete.reason` を固定 |
| `TestTextOutputSaysTheRunIsIncomplete` | 収集時に `were not examined` を**出さないこと**を明示的に検査 |
| `TestTextOutputDoesNotClaimIncompletenessWhenClean` / `…ForSemanticFindings` | 誤って不完全と言わないこと |

**すべて PASS。** 私が確かめたかったことは**既にテストで固定されていた**。

## 7. 読み取れないこと（NOT RUN）

- a.tei と b.tei が**同じ名前空間**で相互依存する場合の収集範囲（今回は独立2ファイルのみ）。
- import を跨ぐ依存の収集。
- 実DB・実Provider・実Host、HTTP・Desktop。

## 8. 位置付け

**「古い記録の文が今も真か」を測った回**であり、答えは**偽（改善済み）**だった。同時に**私は計器を誤り、「モード間で食い違う」と一瞬疑った**。生データを読み直して訂正した。**既存テストが既に守っていたことも確認した**ので、起票しない。
