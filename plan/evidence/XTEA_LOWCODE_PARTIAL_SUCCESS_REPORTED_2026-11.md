# 部分成功を Command 境界から報告する（2026-11-17）

## 対象

- 実装: `internal/command/invoke.go` の `ReportedFailure`・`isPartialSuccess`
- 試験: `internal/command/partial_success_boundary_test.go`
- 契約文書: `plan/design/CLI_JSON_OUTPUT_CONTRACT_2026-10.md` の `view` 行
- 判断の記録: [XF110の部分成功とpartial_successの接続](XTEA_LOWCODE_PARTIAL_SUCCESS_CONNECTION_2026-11.md)

## 何を変えたか

`ReportedFailure` が `failure.PartialError` を `COMMAND_PARTIAL_SUCCESS` として報告するようにした。順序は**部分成功を先、効果不明を後**にした。

X-TEA の一括更新（知識目録 F-14）は「途中で全体をロールバックするが、先に実行した外部機能の効果は戻せない」「部分成功を表す結果がない」という弱点を持っていた。TEI には `PartialError` という意味が実在したが、Command 境界からは**何も届いていなかった**（裸の `PartialError` は `("", "", false)` を返す）。

## 実装中に分かった2つのこと

### 1. `failure.IsPartialSuccess` では届かない

最初は `failure.IsPartialSuccess(err)` で判定したが、`ExecutionError` が `PartialError` を `Cause` に持つ場合に **false** になった。

理由: `failure.AsFailure` は `errors.As` で最初に見つかった `Failure` を返し、`ExecutionError` 自身が `Failure` である。その `Kind()` は自分の符号から導かれ、`default: return failure.TechnicalFailure` なので、**包まれた `PartialError` は見えない**。

そこで `isPartialSuccess` を足し、`errors.As(err, &partial)` で**型を名指して鎖を辿る**ようにした。「一部適用」は、原因を隠すと盲再送を招くまさにその場合だからである。

### 2. 既存の契約試験が「効果不明が優先」を要求していた

`TestContractDocumentNamesTheEffectUnknownOverride` は文書に「優先」という語があることを要求する。私が最初に文書を「部分成功を先に報告」へ書き換えたとき、この語が消えて落ちた。

**規則が1つから2つになったので、文書も2段で書くのが正しい。** 「部分成功を先に報告する（情報が多いため）。そうでなく効果が適用された可能性がある場合は `COMMAND_EFFECT_UNKNOWN` が優先する（下位の符号を置き換える）。どちらの場合も再送前の照会は省かない」とし、試験にも「`COMMAND_PARTIAL_SUCCESS` を名指すこと」「先に報告と書くこと」を足した。**古い規則を消さずに、新しい規則を足す**形にした。

## 観測

- `TestReportedFailureNamesAPartialSuccess` 成功。`COMMAND_PARTIAL_SUCCESS` が返り、元の理由（`2 of 3 compensations were applied`）が message に残る。
- `TestPartialSuccessIsReportedBeforeEffectUnknown` 成功。`EffectUnknown: true` かつ `PartialError` を包む `ExecutionError` で、**部分成功の名が勝つ**。同時に `Retryable()` が `LookupFirst` のままであることも確かめる。
- `TestEffectUnknownStillWinsWhenNothingIsPartial` 成功。部分成功でない効果不明は従来どおり `COMMAND_EFFECT_UNKNOWN`。
- `TestAPlainExecutionErrorKeepsItsCode` 成功。通常の符号は変わらない。
- `TestAPartialErrorIsStillLookedUpBeforeResend` 成功。`PartialError.Retryable()` が `LookupFirst`、`HasEffectUnknown()` が真。
- `internal/command`・`cmd/tei`（契約試験・CLI カタログ・フラグ試験）・`tools/repository-plan-check` は ok。`go build ./...`・`go vet`・`gofmt -l` 成功。

## 読み取れること

- 「一部だけ適用された」が **Command 境界の符号として呼出し元へ届く**。
- 効果不明より部分成功が**先に**報告される（情報量が多い方）。
- **再送の危険は増えていない**: 両方とも `lookup_first` で、照会せずに再送しない。

## 読み取れないこと

- 実 PostgreSQL での `COMPENSATION_PARTIAL` の到達（`effect_real_pg_test.go` は DSN 未設定で skip）。
- 一括更新の「範囲」の宣言（`BatchKeyFields` 相当）は未設計で、**何が残っているか**は伝えていない（伝えるのは「全部ではない」ことだけ）。
- CLI の実際の応答での `COMMAND_PARTIAL_SUCCESS` の表示（`view -execute` の経路で `ReportedFailure` を通るが、部分成功を実際に発生させる参照実行は未整備）。
- 実Provider・旧Runtime での観測。

## 位置付け

参照実装の試験記録であり、製品の受入や本番適格性を示しません。「全部ではない」を伝える経路は実装済み、「何が残っているか」の宣言と実DB観測は未実装。仮承認。
