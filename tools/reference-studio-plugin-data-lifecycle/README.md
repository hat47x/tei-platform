# reference-studio-plugin-data-lifecycle（#2047）

Studio の**標準編集と明示的なデータ削除**について、生ソースの保存、意味の受理・実行許可との分離、外部編集との競合検査、プラグイン削除とデータ削除の区別、共有・不明領域の扱い、変換規則の適用を判定する読取り専用の参照です。

Studio のエディタ、Host のファイル書込み、受理エンジン、Resolver ではありません。ファイルを書かず、プラグインを操作せず、データを削除しません。実 Host 保存・実画面・利用者による操作確認は [#2553](https://github.com/hat47x/tei-platform/issues/2553) へ移しています。

## 実行

```bash
# 生ソースの保存（プラグイン不在・構文不正でも保存できる）
go run ./tools/reference-studio-plugin-data-lifecycle/cmd/tei-studio-data-lifecycle \
  -mode save -input tools/reference-studio-plugin-data-lifecycle/testdata/save-missing-plugin.json

# 明示的なデータ削除の計画
go run ./tools/reference-studio-plugin-data-lifecycle/cmd/tei-studio-data-lifecycle \
  -mode delete -input tools/reference-studio-plugin-data-lifecycle/testdata/delete-confirmed.json

# 形式版の更新（変換なしは拒否）
go run ./tools/reference-studio-plugin-data-lifecycle/cmd/tei-studio-data-lifecycle \
  -mode convert -input tools/reference-studio-plugin-data-lifecycle/testdata/convert-version-only.json
```

## 標準編集（`save`）

入力は、編集開始時の `baseRevision`・`baseFingerprint`、現在の `currentRevision`・`currentFingerprint`、原文 `original`、編集後 `edited` です。

- プラグインが未導入・無効・版非対応でも、**保存は成功**します（`plugin-unavailable:<状態>`を診断に残す）
- 構文不正・参照切れでも保存は成功し、`syntax-invalid`・`unknown-reference:<ref>`を診断に残します
- `protected` に列挙した断片（未知属性・未知子要素・既知子内の拡張）が編集後に残っていることを確認します。消えていれば`protected-content-lost`で**保存を拒否**し、元のバイト列を返しません
- `currentRevision`が`baseRevision`と違えば`stale-revision`、`currentFingerprint`が`baseFingerprint`と違えば`source-bytes-mismatch`で拒否します。外部編集・未保存内容を上書きしません
- **保存は受理でも実行許可でもありません。** `accepted`は入力の事実のまま、`acceptanceChanged=false`、`executable`は既存の許可と構文・参照・プラグイン可用性が揃う場合だけ`true`です

## プラグイン削除とデータ削除

`delete`の`action`は2つです。

| `action` | 意味 |
| --- | --- |
| `remove-plugin` | プラグインを外す。**データは削除しない**。全領域を`plugin-removal-keeps-data`として残す |
| `delete-data` | 明示選択した領域のデータだけを削除する。プラグインの削除とは別操作 |

削除計画は次を分けて返します。

- `confirmed`：対応関係が確定した領域
- `requiresSelection`：共有領域。`selectedAreas`と`confirmShared`の両方が必要
- `retained`：削除しない領域と理由（`not-selected`・`shared-not-selected`・`shared-requires-explicit-selection`・`unknown-provider`・`referenced-by-others:<ids>`）
- `referenceImpact`：選択した領域を参照する他のプラグイン
- `diff`：ファイル数・領域数・削除バイト数・保持バイト数
- `manualEditAllowed`：常に`true`

規則:

- 対応元不明（`unknown`）の領域は削除しません。`deleteAllUnconfirmed=true`は`unknown-area-not-guessable`で全体を拒否します
- 共有領域は、明示選択と`confirmShared`が揃った場合だけ削除します
- 他のプラグインから参照されている領域は、`confirmReferenced`が無ければ残します
- 選択した領域の中に、削除できない種別（`extension`・`unknown-attribute`・`unknown-element`）のバイト列があれば、**部分保存せず**`preservation-conflict:<segment>`で全体を拒否し、元ソースを残します（`sourceUnchanged=true`、`filesAfter`なし）
- 削除後も、選択していない領域・未知属性・未知子要素・拡張のバイト列はそのまま残します

## 変換（`convert`）

- `convert=false`（形式版だけの更新）は`conversion-required`で拒否します。版だけを上げません
- 変換規則が空なら`no-conversion-rules`で拒否します
- `preservable=false`の変更が1つでもあり、`allowPartial=false`なら`unpreservable-change`で全体を拒否し、元ソースを残します
- `allowPartial=true`なら、保持できる変更だけを適用し、`partial=true`・`unpreservable`に保持できない箇所を残します
- 変換後も、所有外のセグメント（未知属性・未知子要素・拡張）のバイト列は変えません（`preservedUnknown`）

## 試験

```bash
go test ./tools/reference-studio-plugin-data-lifecycle/... -count=1
```

- プラグイン不在・無効・版非対応・構文不正・参照切れでも保存できる
- 保存が受理・実行許可を変えない
- 改訂不一致・ソース指紋不一致で保存を拒否する
- 保護した断片が消えた保存を拒否し、部分保存しない
- プラグイン削除がデータを削除しない
- 確定領域だけを削除し、未知属性・未知子要素・拡張をバイト同一で残す
- 共有領域は明示選択と確認が揃うまで削除しない
- 参照影響のある領域は確認が揃うまで残す
- 不明領域を推測で削除しない
- 保持不能なバイト列がある削除は全体を拒否し、元ソースを残す
- 形式版だけの更新を拒否する
- 保持不能な変換は部分保存せず、`allowPartial`では保持できる変更だけを適用する

## 範囲と非主張

- 合成入力に対する静的な判定です。実 Studio・実 Host・実ファイルの保存・削除ではありません
- 実 Host のファイル権限・作業領域境界、実画面、利用者による操作確認は未実施です（[#2553](https://github.com/hat47x/tei-platform/issues/2553)）
- 外部DBの削除、確定済み効果の取消、生成物の自動再配備は扱いません
- #1529の競合検知、#1124/#267の受理、#1920の版・変換規則を置き換えません
- 第二の保存・変換・受理の正本を作りません
- ファイルを書かず、データを削除しません
