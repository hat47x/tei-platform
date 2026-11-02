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

## 範囲と非主張

- 合成入力に対する静的な判定であり、実DB・実トランザクション・分散ノードの証拠ではない
- 版属性（`version`）による比較、自動再試行、複数ノード間の調停は扱わない
- 所有と世代は#2146、実行時の段階と不明の扱いは#1129が所有する
- 秘密項目は`digest`比較で扱えるが、ハッシュの強度・鍵付きハッシュ・監査は未検証
- 実装の製品接続・実DBでの競合試験は未実施
