# 採番の永続CounterStoreの確認（2026-11）

## 対象

- 実装: `internal/numberingsqlite/sqlite.go`
- 試験: `internal/numberingsqlite/sqlite_test.go`
- 対象版: `main` の `d934e7cdd` から分岐した`feat/numbering-sqlite-store-20261110`
- 環境: Windows、Go `1.27.0`

## 実行

```powershell
go test ./internal/numberingsqlite -count=1 -v
```

## 観測

- `TestStoreStartsAtZero` 成功。行の無い規則は現在値0・版0として読める（規則の初期状態）。
- `TestReserveAcrossRestarts` 成功。`ORD-00001`を発行してCommitし、**格納を閉じて開き直しても**現在値1・版1のまま、次の発行は`ORD-00002`。再起動で同じ番号を再発行しない。
- `TestSaveRefusesAStaleVersion` 成功。古い版での書き込みは`ErrNumberingConflict`になり、表は変わらない。
- `TestSaveRefusesAForeignFirstWrite` 成功。既にある規則への「初回書き込み」も競合として拒否される。
- `TestSaveRefusesANegativeCounter` 成功。負の現在値は保存されない。
- `TestLoadRequiresARuleID` 成功。空の規則IDは拒否される。
- パッケージ全体は`ok`。`internal/numbering`も`ok`（`CounterStore`として差し替え可能）。

## 読み取れること

- 採番のカウンタが**再起動を越えて残る**。参照用の記憶域では失われていた性質です。
- 版の一致を**同じSQL文の条件**にしているので、2つの呼出しが同時に同じ版を読んでも両方は成功しません。
- 初回書き込みも`INSERT`の失敗で競合を検出するので、「行が無い」を根拠に2つが同時に作りません。

## 読み取れないこと

- 2プロセス同時実行の実測（`_txlock=immediate`と`busy_timeout`は設定していますが、プロセス間の競合試験は未実施です）。
- 実PostgreSQLの実装（SQLiteのみ）。
- 桁あふれ・巻き戻しの運用上の扱い（`Contract`の契約は別試験）。
- 旧Runtimeとの同一性（#940）。

## 位置付け

参照実装の試験記録であり、製品の受入や本番適格性を示しません。単一プロセスでの永続化と版の競合検出は参照実装で確認済み、2プロセス同時実行とPostgreSQLは未実装です。
