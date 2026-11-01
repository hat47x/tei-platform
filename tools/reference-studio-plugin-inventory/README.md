# reference-studio-plugin-inventory（#2045）

Studio のプラグインについて、導入状態・有効設定・現在の利用可否・権限・保持データの形式/版への対応を**別の値として**整理し、読取り一覧と AI 診断を作る参照です。無効なプラグインを起動せず、未完了の走査や過去の件数を「0件」「現在確認済み」と表示しません。

Studio の実画面、プラグインのローダー、Resolver、意味の正本ではありません。管理操作（無効化・削除）とソース書込みは含みません。グリッドの比較・絞込み・キーボード操作・詳細表示の**実画面確認**は [#2547](https://github.com/hat47x/tei-platform/issues/2547) へ移しています。

## 実行

```bash
# 一覧と診断
go run ./tools/reference-studio-plugin-inventory/cmd/tei-studio-plugin-inventory \
  -input tools/reference-studio-plugin-inventory/testdata/inventory.json

# どのプラグインを観測してよいか（無効・未導入・権限なし・対象外を呼ばない）
go run ./tools/reference-studio-plugin-inventory/cmd/tei-studio-plugin-inventory \
  -mode observe -input tools/reference-studio-plugin-inventory/testdata/inventory.json
```

## 入力

```json
{
  "scan": {
    "state": "complete",
    "observedAt": "2026-11-01T10:00:00Z",
    "settingsVersion": "settings-7",
    "allowedTargets": ["enabled-plugin"]
  },
  "plugins": [
    {
      "id": "enabled-plugin",
      "version": "1",
      "installed": true,
      "enabled": true,
      "permission": "allowed",
      "supportedVersions": ["1"],
      "namespaces": ["alpha"],
      "readState": "readable",
      "files": 5
    }
  ],
  "areas": [
    {
      "key": "area-a",
      "formatId": "fmt",
      "namespace": "alpha",
      "version": "1",
      "providers": ["enabled-plugin"],
      "readState": "readable"
    }
  ]
}
```

- `scan.state`は`complete`・`partial`・`not-run`。`partial`と`not-run`は合計を`null`にし、`confirmed=false`にします
- `scan.allowedTargets`は明示の許可リストです。空は「すべて許可」ではなく**何も許可しない**
- `installed`（導入済みか）と`enabled`（有効設定か）は別の値です。グリッドの行選択はどちらでもありません
- `permission`は`allowed`・`denied`・`unknown`
- `supportedVersions`はプラグインが読めるデータ形式の版です。領域の`version`と照合します
- `areas[].providers`は同じ領域を提供するプラグインの一覧です。複数あっても領域は1回だけ数えます
- `pastFiles`は過去の件数です。現在の件数へ加算せず、`pastFiles`として別に示します

## 行の状態

| 状態 | 意味 |
| --- | --- |
| `enabled` | 導入済み・有効・権限あり・版対応・提供元が一つで読める |
| `disabled` | 導入済みだが無効。処理を呼ばない |
| `not-installed` | 未導入。`kind=reference`の参照行として、導入済み行と区別する |
| `version-unsupported` | 保持データの版に未対応。`version-unsupported:<領域>:<版>` |
| `provider-unknown` | 提供元がない、または観測した提供元が未知。導入済み行と`kind=aggregate`の集約行を区別する |
| `multiple-providers` | 同じ領域に複数の提供元。領域は重複計上しない |
| `unreadable` | 読取り不能。`counts-unknown`を付け、ファイル数を0にしない |
| `permission-denied` | 読取り権限がない（`unknown`は`permission-unknown`） |

## 件数の扱い

- 領域数とファイル数は別の欄です
- `areasKnown`・`filesKnown`は観測できた数、`areasTotal`・`filesTotal`は全体の数です
- 走査が未完了、読取り不能、版非対応、提供元不明のとき、全体の数は`null`です。**0 ではありません**
- `confirmed`は、走査が完了し、無効・読取り不能・版非対応・提供元不明・複数提供元・権限なしが無い場合だけ`true`です
- 過去の件数は`pastFiles`にだけ残し、現在の合計へ足しません

## 観測の許可

`-mode observe`は、`installed=true`かつ`enabled=true`かつ`permission=allowed`かつ`allowedTargets`に含まれるプラグインだけを呼びます。それ以外は`skipped`に理由を返し、**無効プラグインの処理を呼びません**。

## AI 診断

各状態は、表示文言から独立した**理由コード**を持ちます。`messages`はコードごとの表示文で、文言を変えてもコード・状態・件数は変わりません。

- 対象: `pluginId`または`areaKey`、`target`（名前空間）
- 根拠: `evidence[]`（領域のキーと指紋）。生データや値を含めません
- 対処候補: `remediation[]`（`install-plugin`・`enable-plugin`・`grant-permission`・`update-plugin`・`fix-read`・`assign-provider`・`choose-provider`等）
- 追加対象: `additionalTargets[]`（複数提供元のもう一方、未知の提供元）
- 影響: `impact`（`readable`・`no-data-access`・`ambiguous-provider`・`no-data`）
- 指紋と設定版: `sourceFingerprint`・`settingsVersion`
- 再検査条件: `recheck[]`
- 残条件: `remaining[]`（`resolve-blocker`・`allow-and-rescan`等）

診断は既存の開示制御の範囲を広げません。ファイル内容・顧客データ・権限を新たに開示せず、ID・領域キー・指紋だけを返します。

## 実画面の確認（未実施）

受入のうち「グリッドの比較・絞込み・キーボード操作・詳細表示を実画面で確認し、利用者価値は別に記録する」は、Studio の実画面を必要とするため、この参照では行っていません。[#2547](https://github.com/hat47x/tei-platform/issues/2547) で扱います。行の種類（導入済み・参照・集約）、行選択と有効設定の分離、領域の重複計上をしないことは、この参照の出力で確認できます。

## 試験

```bash
go test ./tools/reference-studio-plugin-inventory/... -count=1
```

- 7つの状態（有効・無効・未導入・版非対応・提供元不明・複数提供元・読取不能）と権限なしを区別する
- 走査が部分・未実行のとき、合計を`null`にし、`confirmed=false`にする（0件にしない）
- 過去の件数を現在の件数と分ける
- 読取不能のファイル数を0にしない
- 同じ入力から同じ診断を得て、表示文言を変えても理由コードが変わらない
- 無効・未導入・権限なし・許可外のプラグインを呼ばない
- 複数提供元の領域を1回だけ数え、集約行を重複させない
- 導入済み行・参照行・集約行を区別する

## 範囲と非主張

- 合成入力に対する静的な判定です。実 Studio・実プラグイン・実データの観測ではありません
- プラグインを起動・停止・導入・削除せず、ソースを書き換えません
- 意味の正本・第二 Resolver・第二の権限判定を作りません
- 実画面の操作・利用者価値・キーボード操作は未検証です
- 形式の版の自動変換・移行は行いません
