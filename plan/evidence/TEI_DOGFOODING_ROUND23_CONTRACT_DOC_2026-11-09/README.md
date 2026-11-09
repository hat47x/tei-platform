# 契約文書の符号一覧が実装より狭くなっていた（2026-11-09、第23ラウンド）

- 対象: [#3419](https://github.com/hat47x/tei-platform/issues/3419)、[#3807](https://github.com/hat47x/tei-platform/issues/3807)（前ラウンドで符号を増やした）、[#3420](https://github.com/hat47x/tei-platform/issues/3420)、[ADR-0048](../../adr/0048-shared-application-api-and-transport-adapters.md) §2.3「失敗」。
- 実行日: 2026-11-09（Asia/Tokyo）。
- 状態: **文書の補完と、散文を実装に縛る試験の追加**。

## 1. 目的

前ラウンドで `view -execute` が `internal/command` の21符号をそのまま返すようにした。その結果、契約文書 `plan/design/CLI_JSON_OUTPUT_CONTRACT_2026-10.md` の `view` 行が実装より狭くなっていないかを確かめる。あわせて、並行して出ていた重複 PR を整理する。

## 2. 固定した入力

| 要素 | 値 |
| --- | --- |
| Source | `c3d37f804`（`origin/main`） |
| 環境 | WSL2 Ubuntu 22.04.5、Go `1.27.3 linux/amd64`（gofmt は 1.26.10） |

## 3. 観測

### 3.1 文書は「嘘」ではなく「不足」

契約表の各行が名指す符号27個は、すべて Go のソースに文字列として存在した（存在しない符号を書いている箇所は0）。つまり文書は誤りではなく、**実装が広がった分を書いていない**。

`view` 行の修正前の符号は11個（うち `COMMAND_*` は4つ）。実装は `-execute` の失敗で `internal/command` の21符号をそのまま返すため、たとえば `COMMAND_UNAUTHORIZED`・`COMMAND_DATASOURCE_READ_ONLY`・`COMMAND_COMMIT_FAILED` が文書に無い。

### 3.2 discovery は符号を広告しない

`tei commands` の各 entry が持つ欄は次の15個で、**失敗符号を広告する欄は無い**。

```text
argumentDefaults argumentType argumentValues effect jsonContract modes
name optional output repeatableFlags required resultShape schemaVersion
successShape writeFlags
```

したがって、呼出し元が「このコマンドはどんな符号を返しうるか」を知る方法は契約文書しかない。文書が狭いことは、そのまま発見可能性の欠落になる。

### 3.3 並行していた重複 PR

[PR #3812](https://github.com/hat47x/tei-platform/pull/3812) が #3797 を別実装で解消していた。私の #3799 と #3810 が先に統合されて `DIRTY` になっていたため、閉じた。

## 4. 修正

`view` 行の失敗欄に、`-execute` で現れうる21符号を列挙し、`COMMAND_EFFECT_UNKNOWN` が優先することを明記した。`COMMAND_FAILED` は「符号を持たない失敗」の受け皿として残ることを併記した。

そのうえで、**散文を実装に縛る試験**を追加した。`cmd/tei/cli_contract_codes_test.go` が次を行う。

1. `internal/command/*.go`（テストを除く）を AST で走査し、`Code:` に現れる全リテラルを集める。
2. 契約表の `view` 行から符号を抜き出す。
3. 1 が 2 に含まれることを確かめる。
4. `COMMAND_EFFECT_UNKNOWN` と「優先」の記述が行にあることを確かめる。

ソースを真とするのは、符号を出す経路がインフラを必要とし単体試験から呼べないため。試験側に一覧を書けば、文書と同じようにずれる。

## 5. 試験が効くことの確認

| # | 変えた箇所 | 結果 |
| --- | --- | --- |
| M1 | 文書の `view` 行を前ラウンド前の11符号へ戻す | FAIL: `the view row does not name 20 code(s)` |
| M2 | `internal/command` に新しい符号 `COMMAND_BRAND_NEW_PROBE` を追加（文書はそのまま） | FAIL: `the view row does not name 1 code(s)` |

M1 は「文書が実装から遅れる」方向、M2 は「実装が文書から先行する」方向。両方が落ちる。戻すと PASS。

`go test ./cmd/tei -count=1` は ok（117.6s）。gofmt・vet・`repository-plan-check` の指摘なし。

## 6. 契約文書の更新が「散文の修正」で終わらない理由

今回の変更は文書だけだが、**文書と実装の一致を実行可能な形で固定した**ため、次に `internal/command` が符号を増やしたときは文書を直さないと CI が落ちる。前ラウンドで `ReportedFailure` の AST 走査を入れたのと同じ考え方で、対象を「入口が報告するか」から「文書が名指すか」へ広げた。

## 7. 読み取れないこと（NOT RUN）

- `view` 以外の行（`inspect`・`trace`・`plugin-*`・`knowledge-*`）について、文書の符号が実装と一致しているかの機械検査。今回は `view` だけを対象にした（前ラウンドで符号が広がったのが `view` だけだったため）。
- discovery に失敗符号を載せること（`commands` の欄を増やす変更）。これは契約の形を変えるため、#3419 の判断事項として残る。
- 実DB・実Provider・実Host、HTTP・Desktop、意味の受理・権限・配備。

## 8. 位置付け

前ラウンドで実装が広がった分を文書へ反映し、両者の一致を試験で固定した。discovery が符号を広告しない以上、文書が唯一の発見経路であり、その文書が実装より狭いことは実際の障害につながる。今回は `view` について閉じたが、他の行の機械検査と discovery への符号広告は #3419 に残る。
