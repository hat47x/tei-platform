# #4178 は既に解消済みで、本文の再現手順は現在の main で成立しない（2026-11-18、第101ラウンド）

- 解消: [#4178](https://github.com/hat47x/tei-platform/issues/4178) を**実測にもとづいて close**
- 前段: [第100ラウンド](../TEI_DOGFOODING_ROUND100_REQUIREMENTS_DERIVATION_2026-11-18/README.md)
- 状態: **起票済みの issue が既に直っていた。私は本文を信じて「未解決」と扱っていた。**

## 1. 目的

目的文の「起票～解消まで」を全うするため、**自分の生きている起票を棚卸し**した。`#4178`（「validate が最初の構文エラーで停止し、他のファイルを検査しない」）が OPEN のままだったので、**本文の再現手順をそのまま実行**して現状を測った。

## 2. 固定した入力

| 要素 | 値 |
| --- | --- |
| Source | `44cbd8c9e`（`origin/main`） |
| CLI実行物 | `sha256:1b141fe0ed390f09c3ea935fc10276ccf3331a39833589adf80449bfd2c365bf` |
| 環境 | WSL2 Ubuntu 22.04.5、Go `1.26.11 linux/amd64` |

## 3. 実測: 本文の3ファイルで3件とも報告される

```text
semantic/a.tei   data Broken { attribute x type }
semantic/b.tei   data Broken2 { attribute y frobnicate }
semantic/c.tei   type Bad from
```

```text
$ tei validate -workspace <ws>
semantic/a.tei:2:32  error PARSE_ERROR unexpected token "}"
semantic/b.tei:2:28  error PARSE_ERROR expected "type"
semantic/c.tei:3:1   error PARSE_ERROR unexpected token ""
note VALIDATION_INCOMPLETE 3件の構文の指摘を同じ読み込みで集めました。…
```

**本文の「報告されるのは1件」は、現在の挙動ではない。**

## 4. 解消した版

| commit | 内容 |
| --- | --- |
| `9ba441278`（PR #4400、main の ancestor） | `feat(validate): collect independent source syntax errors` — **収集経路** |
| #4248 / #4385 | 「止まったことを出力に明示する」段階 |

`9ba441278` は `cmd/tei/validate_multiple_syntax_test.go` と `internal/operations/validate_test.go` を追加しており、**複数ファイルの収集はテストで固定されている**（第99ラウンドで実際に PASS を確認済み）。

## 5. 副次に確認したこと

- 構文エラーがあると**意味検証は実際に走らない**（構文を直すと別ファイルの意味エラーが現れる）。note の「意味検証は実行していません」は**文字通り真**。
- JSON 側も `incomplete.reason` で同じ事実を持つ。健康なワークスペースでは `incomplete` は現れない。

## 6. 残る未検証範囲（本 issue の主張ではない）

本文が併記していた**「import をまたぐ依存の収集」「Application 宣言失敗時の子 Source 解決」**は、今回の再現範囲（独立3ファイル）の外にあり、**未検証**である。**本 issue の主張は成立しないので close するが、この範囲は別途確認が必要** — と comment に明記し、**起票はしなかった**（まず測るべき対象であり、まだ欠陥と確定していない）。

## 7. 位置付け

**「起票した issue を、本文ではなく実測で判定した」回**である。私は `#4178` を「未解決」として持ち続けていたが、**本文の再現手順をそのまま流せば 5分で判定できた**。**起票は状態ではなく、測り直すべき仮説である。**
