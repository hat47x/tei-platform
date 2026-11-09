# 例の締め日・設定Dataと構文の修正の確認（2026-11）

## 対象

- 実装: `internal/syntax/parser.go`の`readPeriodNames`（`closes`で停止）、`internal/applicationview/standard_business.go`の`setting`種別
- 例: `examples/xtea-lowcode/semantic/sales/order.tei`（`closes on day 20`）、`semantic/master/standard.tei`（`master.setting`）
- 試験: `internal/syntax/period_test.go`、`internal/applicationview/standard_business_test.go`、`cmd/tei/xtea_example_test.go`、`cmd/tei/xtea_standard_business_test.go`
- 対象版: `main` の `687bd491e` から分岐した`feat/example-settings-closing-20261109`
- 環境: Windows、Go `1.27.0`、Node.js `24.19.0`

## 実行

```powershell
go test ./internal/syntax -run TestSemanticPeriodParsesClosingDay -count=1 -v
go test ./internal/applicationview -count=1
go test ./cmd/tei -run 'TestXTEAExample' -count=1
go run ./cmd/tei view -workspace examples/xtea-lowcode -application order_entry -format text
```

## 観測

### 見つけた不具合と修正

`states open closing closed`の後ろに`closes on day 20`を置くと、**状態名の読み取りが`closes`と`on`と`day`と`20`を状態名として飲み込みました**。締め日は0のまま、状態一覧は7語になっていました。

- 原因: `readPeriodNames`の停止語に`closes`が無く、締め日の宣言が状態一覧へ吸い込まれていた。
- 修正: 停止語へ`closes`を追加。
- 回帰試験 `TestSemanticPeriodParsesClosingDay` を追加（締め日20、状態3語、`closed-rejects`1件）。

修正後の出力:

```text
period sales.month data sales.order states open,closing,closed
period sales.month calendar sales.fiscal
period sales.month closes on day 20
```

JSON投影は`{"id":"sales.month",...,"closingDay":20}`を返します。

### 例への追加

- `sales.month`へ`closes on day 20`を追加（例の期間が月の半分で閉じる）。
- `master.setting`（`setting-key`・`setting-value`・`setting-user`）を追加。
- `view`の標準業務表に`setting`種別が出る（`master.setting`の3属性）。
- 例の試験で、宣言→行の結合→`Table.Change`（`alice`の`200`）→共有の既定（`bob`の`50`）を確認。

## 読み取れること

- 構文の停止語が不足すると、後続の宣言が**静かに別の欄へ吸い込まれる**。締め日を例へ入れたことで初めて表面化しました。
- 例のWorkspaceで、締め日と利用者設定が宣言・投影・結合の三段を通る。

## 読み取れないこと

- 締め日の区分に応じた実際の閉め（`periods -advance`で状態は進められるが、例の操作面に期間付きの操作が無いため拒否一覧は空のまま）。
- 設定の監査・権限。
- 実DB・旧Runtimeの実測。

## 位置付け

参照実装の試験記録であり、製品の受入や本番適格性を示しません。構文の修正と例への追加は確認済み、閉めの実行内容と設定の監査は未実装です。
