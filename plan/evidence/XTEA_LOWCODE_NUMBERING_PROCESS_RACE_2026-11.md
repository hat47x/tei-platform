# 採番の2プロセス同時実行の確認（2026-11）

## 対象

- 実装: `internal/numberingsqlite/sqlite.go`（変更なし。永続格納そのもの）
- 試験: `internal/numberingsqlite/sqlite_process_test.go`
- 対象版: `main` の `8be6fc605` から分岐した`feat/numbering-process-race-20261110`
- 環境: Windows、Go `1.27.0`

## 実行

```powershell
go test ./internal/numberingsqlite -run TestTwoProcesses -count=3 -v
go test ./internal/numberingsqlite -count=1 -v
```

## 観測

- `TestTwoProcessesNeverIssueTheSameNumber` 成功（3回連続）。2プロセスが同時に発行しても、**同じ番号は1つも出ない**。カウンタは成功回数と一致し、版も成功回数と一致する。
- `TestTwoProcessesIssueDistinctNumbers` 成功。`000001`の後に別プロセスが`000002`を取る（再起動を挟んでも連続する）。
- 単体試験6件はそのまま成功。

## 途中で分かったこと（重要な訂正）

最初の試験は「2プロセスのうち**1つだけ**が成功する」と書いていました。実際には**両方が成功する**ことがあります。

- 理由: SQLiteは即時書き込み取引を直列化するので、2つ目の`Load`が1つ目の`Commit`後に走れば、**新しい版**を読んで正常に次を発行します。
- 版が重なった場合だけ`ErrNumberingConflict`で拒否されます。
- どちらも正しい結果です。壊れてはいけないのは「同じ番号が2回出る」ことと「カウンタが発行数を下回る」ことなので、試験をその**不変条件**へ直しました。

X-TEAは楽観的更新の結果を検査していませんでした（`Session.java` L1995–2002）。同じ番号が2回出ても気づけず、片方の発行が失われていました。参照実装では、この2つの不変条件を子プロセス2つで確認しています。

## 読み取れること

- 2プロセスが同じカウンタから発行しても、同一番号は出ない。
- カウンタと版は成功回数に一致する（更新の取りこぼしが無い）。
- 再起動を挟んでも番号は連続する。

## 読み取れないこと

- PostgreSQL実装（SQLiteのみ）。
- ネットワーク越しの2ホスト構成。
- 多数プロセス（3以上）の同時実行。
- 旧Runtimeとの同一性（#940）。

## 位置付け

参照実装の試験記録であり、製品の受入や本番適格性を示しません。SQLite上の2プロセス同時実行は確認済み、PostgreSQLと多プロセス構成は未実装です。
