# 期間の要件を `tei view` へ投影する確認（2026-11）

## 対象

- 実装: `internal/applicationview/application_view.go`の`ApplicationPeriodRef`、`cmd/tei/cli_view.go`の`writeApplicationText`、`cmd/tei-studio/assets/view-format.mjs`の「期間」表
- 試験: `internal/applicationview/period_requirement_view_test.go`
- 対象版: `main` の `efe7763b4` から分岐した`feat/period-requirement-view-20261111`
- 環境: Windows、Go `1.27.0`、Node.js `24.19.0`

## 見つけた不足

期間は`close-requires`と`correction via`を宣言でき、**Commandの計画**（`internal/applicationview.CommandPeriodRef`）には載っていました。ところが**Applicationの投影**（`ApplicationPeriodRef`）には載っていませんでした。

つまり `tei view` で期間を読んだ人は「この期間の閉めに何が必要か」を見られませんでした。Commandの入口だけが知っている状態です。X-TEAでは閉めの必要な処理が画面の実装に埋まっていたので、TEIでは宣言として読めるべきところです。

## 変更

- `ApplicationPeriodRef`へ`CloseRequires`・`Correction`を追加（`closeRequires`・`correction`として投影）。
- テキスト出力へ`period <id> close-requires <capability>`・`period <id> correction via <capability>`を追加（宣言があるときだけ）。
- Studioの「期間」表へ「閉めに必要」「訂正」の列を追加。

## 観測

- `TestApplicationViewProjectsThePeriodRequirements` 成功。`closeRequires`・`correction`が投影される。
- `TestApplicationViewLeavesAnUndeclaredRequirementEmpty` 成功。宣言の無い期間は両方空（既定の能力を捏造しない）。
- `go run ./cmd/tei view -workspace examples/xtea-lowcode -application order_entry -format text` は`period sales.month ...`の3行（状態・暦・締め日）を出し、**例は宣言していないので`close-requires`・`correction`の行は出ない**。
- Studioの`view-format`試験は10件すべて成功。
- `internal/applicationview`・`cmd/tei`の試験は`ok`（`cmd/tei`全体は372秒）。

## 読み取れること

- 期間の要件が**Command計画とApplication投影の両方**から読める。
- 宣言の無い期間は空で、既定の能力を捏造しない。
- テキスト・Studio表の両方に列が増え、`view`だけが取り残されない。

## 読み取れないこと

- **例のWorkspaceで`close-requires`・`correction`を宣言した場合の表示**。例は宣言していないため、実際の値が入った出力は確認していません。
- Studioのブラウザ表示（`view-format.mjs`の表組み試験のみで、実際の画面は未確認）。
- 実Provider・実DB・旧Runtimeでの観測。

## 位置付け

参照実装の試験記録であり、製品の受入や本番適格性を示しません。期間の要件の投影は参照実装で確認済み、例での宣言と画面での確認は未実装です。
