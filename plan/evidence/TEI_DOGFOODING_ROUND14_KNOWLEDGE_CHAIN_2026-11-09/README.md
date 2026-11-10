# AIが使う knowledge-* 連鎖の端から端までの実測（2026-11-09、第14ラウンド）

- 対象: [#3149](https://github.com/hat47x/tei-platform/issues/3149)（ドッグフーディング）、[#3186](https://github.com/hat47x/tei-platform/issues/3186)（Closed）、[#2621](https://github.com/hat47x/tei-platform/issues/2621)。
- 実行日: 2026-11-09（Asia/Tokyo）。
- 状態: **read-only の実測**。意味の受理・権限付与・配備は行っていない。

## 1. 目的

第13ラウンドで「モデルに対する設計作業はCLIで完結できる」ことを確かめた。このラウンドは、その設計作業でAIが実際に使う `knowledge-*` の連鎖を、**端から端まで1回通す**。

## 2. 固定した入力

| 要素 | 値 |
| --- | --- |
| Source | `2dcb75fafa44121907b9f9f93847cfc5c9d7650b`（`origin/main`） |
| CLI実行物 | `sha256:49c0ef003b8799df66adfeb90a56f0505a57edf0af9c9f81e8d37c2f5bf9243d` |
| 環境 | WSL2 Ubuntu 22.04.5、Go `1.27.3 linux/amd64` |
| 対象 | `reference/workspace` の `semantic-data:sales.order` |

## 3. 生データ

| ファイル | 内容 |
| --- | --- |
| `00-head.txt`・`01-binary-sha256.txt` | 対象 commit と実行物 |
| `02-context.json` | `knowledge-context`（`mode=context`） |
| `03-plan.json` | `knowledge-review-plan`（`mode=plan`） |
| `04-handoff.json` | `knowledge-review-handoff`（`mode=handoff`） |
| `06-disclose.json` | `knowledge-context -disclose source`（`mode=disclosure`） |
| `07-qualify.json` | `knowledge-qualify`（`mode=qualification`） |
| `08-review-disclose.json` | `knowledge-review-disclose`（handoff を入力） |
| `09-plan-ok.json`・`10-plan-stale.json` | 正しい指紋／古い指紋での `knowledge-review-plan` |
| `11-missing-handoff.json`・`12-cross-subject.json` | 存在しない handoff／別subjectの handoff |
| `13-requirements-min.json` | 最小の要件ファイル |

## 4. 観測

### 4.1 連鎖は通る

| 手順 | 結果 |
| --- | --- |
| `knowledge-context` | `ok:true`・`mode=context`・`dataDesign` あり・`authorityLimits` 4件 |
| `knowledge-review-plan` | `ok:true`・`mode=plan`・`reviewSequence`・`stopRules`・`recommendedDisclosureSections` |
| `knowledge-review-handoff` | `ok:true`・`mode=handoff`・`qualification` を含む |
| `knowledge-context -disclose source` | `ok:true`・`mode=disclosure`・`availableSections` と `disclosedSections` |
| `knowledge-qualify` | `ok:true`・`status=applicable`・`evidence` 1件 |
| `knowledge-review-disclose` | `ok:true`・`status=complete`・4セクション |

**AIが「この要素を設計するために何を知るべきか」を、CLIだけで一巡できる。**

### 4.2 戻れる（拒否される）

| 操作 | 結果 |
| --- | --- |
| 正しい指紋を渡す `knowledge-review-plan` | `ok:true` |
| 古い指紋を渡す | `KNOWLEDGE_BUILD_FAILED`・exit 1・`fingerprint mismatch: expected=… current=…` |
| 存在しない handoff を `knowledge-review-disclose` に渡す | `KNOWLEDGE_BUILD_FAILED`・exit 2 |
| 別subjectの handoff を使い回す | `KNOWLEDGE_BUILD_FAILED`・exit 2・`subject mismatch: handoff=semantic-data:sales.order context=semantic-data:sales.order_line` |

指紋を取り違えると失敗し、理由に期待値と現在値の両方が出る。handoff の使い回しも件名の不一致で拒否される。**「戻せる」の意味での安全側は効いている。**

### 4.3 入力ファイルの形はCLIから分からない

最小の要件ファイルは `{"expectedKind":"semantic-data"}` である。私は最初 `{"subject":…,"requiredDependencies":[],"authoritativeSourceRequired":true}` と書き、`json: unknown field "subject"` で拒否された。

- `commands` の entry は `-qualification-requirements` の存在だけを広告し、**中身の形は伝えない**。
- `-h` も "knowledge qualification requirements JSON" としか出さない。
- 形を知るには `docs/business-system-design-reading-guide.md` を読む必要がある（#3186 で例は追加済み）。

つまり **AIがCLIだけで連鎖を通そうとすると、入力ファイルの形のところで一度止まる**。文書を読めば解決するが、CLIの応答からは辿れない。

## 5. 起票しない理由

形の例は #3186（Closed）で文書に追加済みであり、`commands` にファイル中身のスキーマを載せるかは #3419（共通APIの型付き契約）の範囲である。重複Issueを作らない方針に従い、**この実測を記録として残す**。

## 6. 読み取れないこと（NOT RUN）

- 実AI（外部Harness）がこの連鎖を使って設計した結果の質。
- `knowledge-data-design-candidate-review`（候補が無いと空になるため、別途フィクスチャが要る）。
- HTTP・Desktop、実DB・実Provider、権限。

## 7. 位置付け

`knowledge-*` の連鎖が端から端まで通り、古い指紋と別subjectの handoff が拒否されることを確認した。残る摩擦は「入力ファイルの形がCLIの応答から辿れない」1点で、これは文書（#3186）と共通契約（#3419）の担当である。
