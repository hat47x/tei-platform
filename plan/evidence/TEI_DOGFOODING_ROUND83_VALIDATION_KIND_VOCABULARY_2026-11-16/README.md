# 未知の validation kind が有効値を挙げない（2026-11-16、第83ラウンド）

- 対象: `semantic.ValidationKind` の閉じた語彙。既存の #4169 系と同じ型。
- 前段: [第82ラウンド](../TEI_DOGFOODING_ROUND82_OPERATIONS_BUILD_2026-11-16/README.md)
- 状態: **診断が5つの有効値を名指しするようになり、語彙の単一の出所を追加した。**

## 1. 目的

目的文の「再びドッグフーディングに戻る」を実行した。**CLI のフィードバックだけで** semantic validation を設計する。

## 2. 固定した入力

| 要素 | 値 |
| --- | --- |
| Source | `0bffd4f83`（`origin/main`） |
| CLI実行物（設計時） | `sha256:61c0068980e3c728ecef2eee7efb23afe224c395eb3a3dcf5e1f26b58650a78f` |
| CLI実行物（修正後） | `sha256:44c10ff1ffffc3f0c589aedef1d2c2d087357ded6e40bc15a38d38f8baeb35ab` |
| 環境 | WSL2 Ubuntu 22.04.5、Go `1.26.11 linux/amd64` |

## 3. 設計の経過（CLI のみ）

| Attempt | 応答 | 学んだこと |
| --- | --- | --- |
| 1 | `expected attribute, relation, candidate-key, or functional-dependency` | validation は `data` の中ではない |
| 2 | `unsupported semantic validation kind "frobnicate"` | トップレベル宣言。**しかし有効値を言わない** |
| 3 | `invalid canonical id "probe.Thing.id"`（`[a-z][a-z0-9_]*` 付き） | 大文字が駄目 |
| 4 | `SEMANTIC_VALIDATION_POSITIVE_REQUIRES_NUMERIC_ATTRIBUTE` | positive は数値属性を要する |
| 5 | `ok` ＋ 読み返しで `kind: positive / attribute: probe.thing.amount` | 完成 |

**Attempt 2 が穴**である。`unsupported` だけでは**5つのうちどれが正しいか分からない**。

## 4. 修正

1. `semantic.ValidationKinds` を足し、`Valid()` がそこを読む（`TypeKinds` と同じ形）。
2. 診断が語彙を列挙する。

```text
unsupported semantic validation kind "frobnicate"; valid values are
positive, relation-unique, operator, not-after, no-overlap
```

**単一の出所**にしたので、kind を足すときに更新する場所は1か所で済む。

## 5. 途中で4回、自分の仮定が外れた

| 誤った仮定 | 実際 |
| --- | --- |
| `validation` は `data` の中 | **トップレベル宣言** |
| パーサの入口は `ParseValidation` | **`ParseSemanticFile`** |
| body は `operator`/`compared` を取る | **`attribute` と `relation` だけ** |
| 全ての kind が同じ body で通る | `relation-unique` は **`relation` を要する** |

**すべて推測で書いて落ちて、コードを読んで直した。**

## 6. 変異試験

| # | 変えた箇所 | 結果 |
| --- | --- | --- |
| M1 | 有効値の列挙を削る（修正前の状態） | FAIL `the diagnostic does not name the accepted kinds` |
| M2 | `ValidationKinds` から `not-after` を落とす | FAIL `"not-after" is declared but missing from ValidationKinds` |

## 7. 回帰確認

| 対象 | 結果 |
| --- | --- |
| `./internal/semantic` | ok |
| `./internal/syntax` | ok |
| `./internal/workspace` | ok |
| `./internal/validation` | **1件のみ FAIL**（`TestDiagnosticCodeCoverageOnlyImproves`） |
| `go build ./...` / `go vet` / `gofmt` | rc=0 / 指摘なし / 空 |

### 7.1 `TestDiagnosticCodeCoverageOnlyImproves` は**私の変更が原因ではない**

- **plain main で同じく FAIL**（3回連続で 1件）。
- 私のブランチでも 3回連続で 1件。
- `TestTEIValidationDeclarationSelfDescriptionIsOnlyData` は**単独では両方 PASS**。当初これを回帰と疑ったが、**私の `-run` 正規表現が2テストを同時に走らせたため**の見かけの失敗だった。**計測し直して誤りを訂正した。**

## 8. 読み取れないこと（NOT RUN）

- `TestDiagnosticCodeCoverageOnlyImproves` の原因（本ラウンドの範囲外、plain main でも FAIL）。
- 5つの kind の**意味の妥当性**（設計判断）。
- 実DB・実Provider・実Host、HTTP・Desktop。

## 9. 位置付け

**「CLI が語彙を教える」ようにした回**である。設計者は `frobnicate` と書いて `unsupported` と言われ、**5つの候補を推測するしかなかった**。今は候補が並ぶ。

**同時に、単一の出所を作った。** 診断と `Valid()` が別々のリストを持つと、片方だけ更新されて「受理されるが名指しされない kind」が生まれる。M2 がその drift を捕まえる。
