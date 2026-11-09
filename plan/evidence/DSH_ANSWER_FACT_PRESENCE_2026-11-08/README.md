# 外部Harness四回答のFact出現チェック（2026-11-08、#2509 / TAI #168）

- 対象Issue: [#2509](https://github.com/hat47x/tei-platform/issues/2509)、上位計画は[TAI #168](https://github.com/hat47x/tai-pleroma/issues/168)の初回Runbook。
- 実行日: 2026-11-08（Asia/Tokyo）
- 状態: **機械的な出現チェックのみ**。正答の採点ではない。Reviewer-onlyの独立Expectedによる採点は**未実施（NOT RUN）**。

## 1. 何をしたか

[四Session実行の記録](../DSH_H2_H3_FOUR_SESSION_2026-11-07/README.md)が保存した4つの回答（H2-A・H3-A・H3-B・H2-B）について、**同じ実行物が返したCLI出力に現れた事実**を回答が言及しているかだけを機械的に確かめた。

事実の集合は**手書きの期待表ではなく、保存済みのCLI応答から作る**。使ったのは次の2つ。

| 入力 | 用途 |
| --- | --- |
| `cli-inspect-sales-order.txt` | Subject・属性（ID・型・記号）・関係・Source artifactとdigest・World指紋 |
| `cli-validate-json.txt` | `ok`・診断件数・重大度別件数・診断コード |

道具は `tools/reference-harness-answer-presence`（解析は `tools/internal/harnessanswer`）。

## 2. 観測した結果

13の事実すべてが、4回答の**どれにも出現した**。

| 回答 | 一致 | 不一致 |
| --- | --- | --- |
| H2-A（Skill非公開） | 13 | 0 |
| H3-A（Skill実Load） | 13 | 0 |
| H3-B（Skill実Load） | 13 | 0 |
| H2-B（Skill非公開） | 13 | 0 |

一致した事実の内訳（4回答で共通）:

- `canonical-id`（`sales.order`）
- 属性6件（`customer_id`・`external_id`・`order_id`・`ordered_at`・`status`・`total`）
- 関係2件（`sales.order.customer`・`sales.order.lines`）
- Source artifact `semantic/sales/order.tei`
- `worldFingerprint`
- 診断コード2種（`APPLICATION_RELEASE_CONTRACT_NOT_CURRENT`・`DATABASE_RELEASE_CONTRACT_NOT_CURRENT`）

## 3. この記録が示すこと

- 4回答とも、**CLIが返した識別子・型・関係・Source path・指紋・診断コードを、少なくとも文字として含んでいる**。
- したがって「対象を取り違えた」「存在しない属性を書いた」といった**初歩的な不一致は、この機械チェックの範囲では見つからない**。
- **属性は`customerId`のような記号（symbol）で書かれることが多く**、canonical ID（`sales.order.customer_id`）だけを照合すると5件が不一致に見える。回答の書き方に合わせて**ID・型・記号の3形を受け入れる**必要があった（実際、初版はこの理由で5件を誤って不一致と判定した）。

## 4. この記録が示さないこと

- **回答が正しいこと**。語が出現しても、意味・型・関係の向き・時点・未確認の扱いが正しいとは限らない。
- **未知を残したか**。本チェックは「書かれた事実」だけを見ており、**書かれていない未確認事項**を評価しない。
- **Skillの効果**。4回答が同じ13事実に一致したことは、**腕の違いを示さない**（どれも同じCLI出力を読んでいる）。
- 独立Expectedによる採点、Runbookの合否、Owner受理、#1119のGate。
- 事実の集合は**この2つのCLI応答に限る**。`trace`・`model catalog`・`source-index`の全内容は含まない。

## 5. 生データ

| ファイル | 内容 |
| --- | --- |
| `presence.json` | 4回答それぞれの一致/不一致、一致した事実と一致した表記、`limits`（この検査が示さないこと） |

回答本文は[四Session実行の記録](../DSH_H2_H3_FOUR_SESSION_2026-11-07/README.md)の`answer-*.md`をそのまま参照し、複製していない。

再現する場合の要点:

```sh
go run ./tools/reference-harness-answer-presence \
  -answer plan/evidence/DSH_H2_H3_FOUR_SESSION_2026-11-07/answer-h2-a.md \
  -inspect plan/evidence/DSH_H2_TOOL_ONLY_READ_ONLY_2026-11-07/cli-inspect-sales-order.txt \
  -validate plan/evidence/DSH_H2_TOOL_ONLY_READ_ONLY_2026-11-07/cli-validate-json.txt \
  -out <new file>
```

保存済みCLI出力は**JSONの後ろにPowerShellホストの診断行**（`[stderr] ...`）が続く。道具は**最初のJSON値だけを読む**。ファイル全体を単一のJSONとして解析すると失敗する。

## 6. 採点への含意（判断はしない）

本記録は「Reviewerが原文を読み直す手間を減らす」ための**下読み**であり、採点の代替ではない。13の事実すべてに到達しているため、**残る争点は語の一致ではなく、意味・未確認の扱い・Source版の区別**にある。どこまでを機械で確認し、どこからを人が読むかは、ReviewerとOwnerの判断である。
