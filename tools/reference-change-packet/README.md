# reference-change-packet（#2017）

変更の範囲、危険度、必須の検証、停止条件を一つのJSONへ宣言し、変更pathの範囲確認と検証の実行、証拠の記録を行う参照です。ADR-0035の作業票（task packet）の最小実装で、意味の受理・権限の付与・PRの統合は行いません。

```text
go run ./tools/reference-change-packet/cmd/tei-change-packet validate -packet plan/...
go run ./tools/reference-change-packet/cmd/tei-change-packet scope    -packet plan/... -repo .
go run ./tools/reference-change-packet/cmd/tei-change-packet verify   -packet plan/... -repo .
```

`verify`は、変更pathが許可範囲内であることを確認してから、宣言された検証コマンドを実行します。1つでも範囲外・禁止path・検証失敗があれば`passed=false`を返し、終了コード1で終わります。

## 作業票の項目

| 項目 | 内容 |
| --- | --- |
| `schema` | `tei.reference.change-packet/v0` |
| `id` | 作業票の識別子（英小文字で始まる64字まで） |
| `issue` | 対象Issue番号 |
| `title` | 作業の要約 |
| `level` | `A0`〜`A4`。ADR-0035の段階 |
| `base` | 変更範囲を比較するbase。A2/A3では必須 |
| `allowedPaths` | 変更してよいpath。`dir/**`と`path.Match`のglobを使える |
| `forbiddenPaths` | 明示的に禁止するpath |
| `verification` | 実行する検証。`name`、`run`、任意の`timeoutSeconds` |
| `stopConditions` | 人へ戻す条件。文章で宣言し、receiptへ残す |
| `evidence` | 残す証拠の種類（任意） |

## 危険度

- `A0`: 読み取り・分析のみ。
- `A1`: 受理済み意味からの決定的な派生物。正本を変更しない。
- `A2`: 隔離branchでのSource・コード・試験の変更。
- `A3`: 事前承認された低リスク変更の統合。範囲・検証・証拠がすべて機械確認できた場合だけ。
- `A4`: 意味の受理、権限、本番配備、秘密情報、ライセンス判断、破壊的移行。人間の判断を必須とする。

このツールは`A3`の判断を代行しません。`passed=true`は、宣言した範囲と検証が一致したことだけを示します。

## 守っている境界

- 変更pathは、`git diff <base>`とuntracked fileから求める。ステージ済み・未ステージ・未追跡を区別せず、baseとの差分として扱う。
- 許可path外・禁止path・不正なpath（絶対path、`..`）を拒否する。
- 検証コマンドは、作業票に明示されたものだけを実行する。任意のShellを既定で受け付けない。検証コマンドは秘密情報を出力しないよう定義する。
- 失敗・範囲外・git状態の不読を、成功として扱わない。
- 作業票、Workspace、Gitの状態を書き換えない。PRの統合、意味の受理、権限の付与を行わない。
- 実行環境は`bash`を前提とする。Windows nativeでの同等性は未検証。

## 証拠

`verify`の出力（receipt）には、次が入る。

- 作業票のidと危険度
- 検証したexact headとbase
- baseとの差分にある変更file
- 範囲違反（あれば）
- 各検証の名前、コマンド、終了コード、出力（16 KiBまで）
- 停止条件
- 総合の`passed`

時刻や実行者のidentityは含めない。実行者の認証は別の境界で扱う。

## 例

`examples/2017-change-packet.json`は、このツール自身の変更に使う例です。

```text
go run ./tools/reference-change-packet/cmd/tei-change-packet verify \
  -packet tools/reference-change-packet/examples/2017-change-packet.json -repo .
```

## 未実装

- 変更範囲から検証を自動選択する規則。現状は作業票に明示する。
- 危険度`A3`の承認と統合の自動化。意図的に対象外とする。
- 証拠の署名、保存期間、PRへの自動添付。
- Windows native、DSHプラグイン、CIとの統合。
- 停止条件の機械評価。現状はreceiptへ残すだけで、内容の判断は人または呼出し側が行う。
