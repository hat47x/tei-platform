# reference-expected-state-update（#2483）

PoC仮説H1「同時更新の検出を期待状態の比較で行う」を、実行時の判定として確かめる読取り専用の参照です。更新の要求に、読んだときの値（または照合値）と冪等キーを含め、現在の値と一致した場合だけ適用します。一致しない場合は**業務結果として拒否**し、実行エラーへ丸めません。

データベース・トランザクション管理・版属性（`version`）による比較・分散調停・自動再試行ではありません。ファイルを書かず、要求と状態は合成入力です。

## 実行

```bash
go run ./tools/reference-expected-state-update/cmd/tei-expected-state-update \
  -input tools/reference-expected-state-update/testdata/conflict.json
```

## 入力

```json
{
  "state": {
    "records": {"order:1": "old"},
    "applied": {}
  },
  "requests": [
    {
      "key": "order:1",
      "expectedPresent": true,
      "expected": "old",
      "newValue": "A",
      "idempotencyKey": "update-a",
      "authorized": true,
      "valid": true
    },
    {
      "key": "order:1",
      "expectedPresent": true,
      "expected": "old",
      "newValue": "B",
      "idempotencyKey": "update-b",
      "authorized": true,
      "valid": true
    }
  ]
}
```

- `expectedPresent`は、期待値が空文字であることと、期待値を渡していないことを区別します
- `compare`は`value`（既定）または`digest`です。`digest`では生の値を要求へ載せず、`expectedDigest`だけを比較します
- `idempotencyKey`は同じ要求の再送を識別します
- `authorized`・`valid`は、権限と検証を別の事実として渡します

## 結果

| `outcome` | 意味 | `executionError` |
| --- | --- | --- |
| `APPLIED` | 現在値が期待値と一致し、更新を適用した | `false` |
| `CONFLICT` | 現在値が期待値と一致しない。後から来た更新を拒否し、先の更新を残す | `false`（業務結果） |
| `IDEMPOTENT_REPLAY` | 同じ冪等キーと同じ要求の再送。二重に適用しない | `false` |
| `DENIED` | 権限がない（`reason=permission`） | `false` |
| `INVALID` | 検証に失敗した（`reason=validation`）、または要求が不正 | `false` |
| `MISSING` | 対象のレコードがない（`reason=missing-record`） | `false` |
| `REJECTED` | 同じ冪等キーを別の更新に再利用した（`reason=idempotency-key-reused`） | `false` |

拒否はすべて業務結果です。`executionError`は常に`false`で、実行基盤の障害とは区別します。

## 観測

`observation`は実行の記録から再現できる件数です。

- `comparisons`：比較した回数
- `applied`：適用した回数
- `conflictRejections`／`lostUpdatesPrevented`：競合で拒否した回数（後勝ちで消えたはずの更新）
- `idempotentReplays`：再送として扱った回数
- `permissionRejections`／`validationRejections`／`missingRecords`：拒否理由の内訳
- `comparisonInputBytes`：比較に入力したバイト数。`digest`では照合値の長さだけ
- `idempotencyKeyRejections`：冪等キーの再利用を拒否した回数

`digest`は最終状態と観測の正規化JSONのSHA-256です。同じ入力から同じ`state`・`results`・`digest`になります。時刻・乱数・環境依存の値を混ぜません。

## 実DBでのcompare-and-swap（#2564）

`PGStore`は、同じ判定を**実PostgreSQLのcompare-and-swap**として実行する検証用の経路です。合成入力の`Apply`と同じ結果コードを使いますが、状態はDBの行にあります。

```sql
UPDATE public.tei_expected_state
   SET value = $3
 WHERE key = $1 AND value = $2
```

影響行数が1なら`APPLIED`、0なら現在値を読み直して`CONFLICT`または`MISSING`を返します。競合は業務結果であり、実行エラーではありません。

実行（使い捨てDB専用。試験は2つの固定テーブルをTRUNCATEします）:

```bash
CREATE DATABASE tei_expected_state_e2e;

TEI_EXPECTED_STATE_PG_DSN='postgres://.../tei_expected_state_e2e?sslmode=disable' \
TEI_EXPECTED_STATE_PG_CONFIRM_ISOLATED=tei_expected_state_e2e \
  go test ./tools/reference-expected-state-update/ -run TestPG -v
```

実PostgreSQL 14.24（WSL2、使い捨てクラスタ）で確認したこと:

- 同じ期待値に対する2接続の同時更新で、**ちょうど1件だけ**`APPLIED`、他は`CONFLICT`。先の更新が残る
- `READ COMMITTED`の単一文CASは**アプリ側の再試行を必要としない**。8接続同時でも実行エラーは出ず、適用は1件
- 対照として、`REPEATABLE READ`で**読んでから書く**とSQLSTATE `40001`（serialization failure）になり、呼び出し側の再試行が必要になる。だから実装は単一文CASにしている
- 冪等キーは主キーinsertで先に確保する。同じ要求の再送は`IDEMPOTENT_REPLAY`、別の更新への再利用は`REJECTED`
- 権限・検証の拒否は書き込み前に返し、冪等キーを消費しない
- 不在（`MISSING`）と競合（`CONFLICT`）は別の結果で、不在は冪等キーを残さない

### 実DB経路で実装していないこと

- `compare=digest`は**拒否**する。正式採用は「サーバ側で生値を比較」であり、実DB経路は`value`比較だけを提供する。比較はSQLのCAS文の中で行い、**現在値を結果・観測・ログへ返さない**（#2564項目3の判断）。
- 製品Runtime（#1127の標準CRUD・#1129の実行時段階）への接続
- 分散ノード間の調停（#2146）、自動再試行、観測の永続保存と保持期間

## 実行記録（観測の保存と再現）

1回の実行を、生の値を持たない**再現可能な記録**として保存できる（#2564項目4）。

```bash
go run ./tools/reference-expected-state-update/cmd/tei-expected-state-update \
  -input tools/reference-expected-state-update/testdata/conflict.json \
  -record /tmp/run-record.json

# 同じ入力の再実行が記録を再現するか検証する
go run ./tools/reference-expected-state-update/cmd/tei-expected-state-update \
  -input tools/reference-expected-state-update/testdata/conflict.json \
  -verify /tmp/run-record.json
```

記録に含めるもの:

- 各要求の`key`、冪等キーのdigest、`outcome`、`reason`、`executionError`
- `before`/`currentValue`/`newValue`の**digestのみ**（生の値は保存しない）
- `observation`の件数と`comparisonInputBytes`
- 記録全体の`digest`（entriesのcanonical JSONのSHA-256）

保存先は呼び出し側が指定する（この参照は1実行1ファイル）。保持は`RetainRuns`の既定で**最新N件**を残す。`DecodeRecord`はschemaとdigestの不一致を拒否し、`VerifyRecord`は再実行の結果・件数・比較バイト数のずれをentry単位で報告する。

生の値を保存しない理由は、項目3で採用した「サーバ側で生値を比較」と、記録が要求より長く残るためである。

実DB経路（`PGStore.Apply`）の`Result`も同じ`RecordRun`で記録できる。`TestPGRunIsRecordedAndReproducedWithoutRawValues`が、実PostgreSQLで適用→競合→再送を実行し、記録の再現と生値非保存を確認する（`TEI_EXPECTED_STATE_PG_DSN`設定時のみ）。

### 署名（任意）

`SignRecord` / `VerifySignedRecord` が、記録をEd25519署名のenvelopeで包む。署名は`keyId`とrecordのcanonical JSONを対象にし、keyIdの付け替え・entry改変（digest再計算を含む）・別鍵での検証を拒否する。参照実装は鍵を持たず、鍵は呼出側（Host）が供給する。署名なしの既存recordが既定のままである。

### 本番方針（仮承認・実物ベースで見直し）

2026-11-06のOwner仮承認として、次を暫定の運用方針とする。**本番保証ではなく、実物の監査要件・法令・テナント分離を確認して見直す**（#2845）。

| 項目 | 仮方針 | 実装状況 |
| --- | --- | --- |
| 保存先 | 共有オブジェクトストレージ（S3互換）。参照はローカル1実行1ファイルのまま | 未実装 |
| 保持期間 | 最新1000件かつ90日。削除は運用承認を要する | `RetainRuns`は件数のみ |
| 監査アクセス | 読取を監査ログへ記録し、テナント・案件単位で分離 | 未実装 |
| 署名 | Ed25519、`keyId`付き、Host供給鍵。ローテーションはkeyIdで識別 | `SignRecord`/`VerifySignedRecord` |
| 再現性 | record・署名・元Evidenceが揃う対象に限り条件付きで再計算 | `VerifyRecord`（合成・実DB） |

`digest validated` ≠ `record authentic` ≠ `input replayable` ≠ `effect committed` ≠ `disclosable` の区別は維持する。署名は「record+signatureの同時改変」を鍵の管理下で検出する手段であり、鍵が漏えいした場合や、元Evidenceが失われた場合の再現性を保証しない。

## 実行時段階（unknown / takeover）

`RuntimeStage`は、CASコマンドの実行時段階を検証する参照です（#2564 → #1129）。

- `Dispatch` → 送信済み（`dispatched`）
- `Observe` → 業務結果を観測（`observed`）。owner と generation が一致する必要がある
- `MarkUnknown` → 応答が不明（`unknown`）

不変条件（fail-closed）:

- `unknown` のコマンドは、同じ owner/generation では**再送しない**（`Dispatch`が拒否）
- `Takeover` は `unknown` のときだけ、**より大きい generation** と新しい owner で許可する
- takeover 自体は成功ではない。`dispatched` のままで、`Observe` して初めて結果を記録する
- `observed` のコマンドは再送しない
- 観測の owner/generation が送信時と違えば拒否する

`Observe`の結果には、CAS参照と同じ`APPLIED`/`CONFLICT`/`MISSING`/`DENIED`等をそのまま記録する。

## 試験

```bash
go test ./tools/reference-expected-state-update/... -count=1
```

- 二つの更新が競合すると、後から来た方が`CONFLICT`で拒否され、先の更新が残る
- 拒否が業務結果（`executionError=false`）で返る
- 同じ要求の再送が二重に適用されない
- 同じ冪等キーを別の更新に使うと拒否される
- 権限・検証・不在が別の結果になる
- 期待値が空文字であることと、期待値を渡していないことを区別する
- `digest`比較が生の値を要求へ載せず、比較入力を照合値の長さに抑える
- 同じ入力から同じ状態・結果・観測・digestを得て、入力を変更しない
- 署名した記録は検証を通り、entry改変（digest再計算を含む）・別鍵・keyId付け替え・欠落署名を拒否する

## 範囲と非主張

- 合成入力の`Apply`は静的な判定であり、実DBの証拠ではない。実DBの証拠は`PGStore`の試験で別に取得する
- 版属性（`version`）による比較、自動再試行、複数ノード間の調停は扱わない
- 所有と世代は#2146、実行時の段階と不明の扱いは#1129が所有する
- 秘密項目は「サーバ側で生値を比較」する方針を採用した。期待値は保護された要求経路で送り、DBのCAS文で比較する。現在値は結果・観測へ返さない。鍵付きハッシュ・監査ログの保持期間・マスキングは未実装
- 実DB経路は `internal/command` の `Lifecycle` と `effectcommit` Provider へ接続し、`commandadapter` 経由のend-to-endを実PostgreSQLで確認した（#2852・#2853）。ただし`RuntimeStage`/`Lifecycle`はin-memoryであり、再起動後の排他・durable owner fencingは保証しない
- **ABA（RCAS-01）**: 値一致CASだけでは同じ論理Commandを一つの効果に束縛できない。別idempotency keyでの再適用を反例として`pg_aba_test.go`に固定した。対象の版属性・operation identity・Owner受理済み再試行Policyの選択は未決
- **再起動（RCAS-02）**: `RuntimeStage`はin-memoryであり、別プロセス・別workerはunknownの記録を持たない。`pg_restart_test.go`は、再起動後の再Dispatchが拒否されないことと、同じidempotency keyの再送だけがProviderの記録で二重適用を防ぐことを固定する。durable owner/generationの実装は未着手
- **応答喪失（RCAS-03）**: TCP proxyでcommit後のCommandCompleteを呼出側へ渡さず、サーバは確定済み・呼出側は不明になる実接続断を`pg_ack_loss_test.go`で固定する。process死亡・再起動を伴う障害は未実施
- 観測の保存先・保持期間・監査アクセスは仮方針（上記）で、共有ストレージ・監査ログ・保持運用は未実装。署名は参照実装に追加したが、鍵の保管・ローテーション・失効の運用は未実装
