# 標準業務Dataの投影の確認（2026-11）

## 対象

- 実装: `internal/applicationview/standard_business.go`、`ApplicationView.standardBusiness`、`cmd/tei/cli_view.go`のテキスト出力、`cmd/tei-studio/assets/view-format.mjs`
- 試験: `internal/applicationview/standard_business_test.go`、`cmd/tei-studio/test/view-format.test.mjs`
- 対象版: `main` の `6a956e92d` から分岐した`feat/studio-standard-business-projection-20261108`
- 環境: Windows、Go `1.27.0`、Node.js `24.19.0`

## 実行

```powershell
go test ./internal/applicationview ./internal/studiohost ./cmd/tei -count=1
node --test cmd/tei-studio/test/client.test.mjs cmd/tei-studio/test/preview-suite.test.mjs cmd/tei-studio/test/view-format.test.mjs
go run ./cmd/tei view -workspace examples/xtea-lowcode -application order_entry -format text
```

## 観測

- `TestBuildApplicationProjectsStandardBusiness` 成功。税と休日のDataが種別・Data・属性で投影され、標準業務の役割を持たないDataは出ない。属性は結合器が読む順に並ぶ。
- `TestStandardBusinessSkipsADataWithNoBinderRole` 成功。表示用の役割しか持たないDataは投影されない。
- `TestStandardBusinessListsOnlyRolesTheBinderReads` 成功。`note`のような表示用の役割は結合器の属性一覧へ混ざらない。
- `view-format`の「application view」試験成功。StudioのApplication投影に「標準業務」の表（種別・Data・属性）が出る。
- `node --test`は12件すべて成功。
- 例のテキスト出力は次を出す（抜粋）。

```text
standard-business exchange master.exchange_rate (master.exchange_rate.currency ... )
standard-business holiday master.holiday (master.holiday.holiday_on)
standard-business numbering master.numbering_rule (master.numbering_rule.counter ... )
standard-business tax master.tax_rate (master.tax_rate.applies_from ... )
```

## 読み取れること

- 例のWorkspaceで、標準業務のDataがどの種別の結合器に読まれるか、どの属性が読まれるかを投影から読める。
- 属性の並びは結合器の要求順（必須が先）なので、欠けた宣言は短い一覧として見える。
- 標準業務の役割を持たないDataは混ざらない。

## 読み取れないこと

- 行データと計算結果。投影は宣言の読み取りであり、税額・換算・採番の発行は行わない。
- StudioのWails RPC経由の実HTTP確認。
- 実DB・旧Runtimeの実測。

## 位置付け

参照実装の試験記録であり、製品の受入や本番適格性を示しません。標準業務Dataの投影は参照実装で確認済み、Wails RPC経由の確認と実DBは未実施です。
