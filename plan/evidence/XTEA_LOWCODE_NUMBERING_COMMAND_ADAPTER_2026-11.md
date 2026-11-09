# 採番とCommand取引境界のアダプタの確認（2026-11）

## 対象

- 実装: `internal/numberingcommand/adapter.go`
- 試験: `internal/numberingcommand/adapter_test.go`
- 対象版: `main` の `0c73b7f5e` から分岐した`feat/numbering-command-uow-adapter-20261108`
- 環境: Windows、Go `1.27.0`

## 実行

```powershell
go test ./internal/numberingcommand -count=1 -v
```

## 観測

- `TestAdapterCommitsTheCommandTransaction` 成功。Command側の取引を1回だけCommitし、カウンタが1になる。
- `TestAdapterRollsBackAndReturnsTheCounter` 成功。RollbackでCommand側の取引が戻り、カウンタも0に戻る。
- `TestAdapterReportsABeginFailure` 成功。Command側のBegin失敗がそのまま返る（番号側の障害と混同しない）。
- パッケージ全体は`ok`。`internal/numbering`・`internal/command`も`ok`。

## 読み取れること

- 番号の発行とCommandの変更を、**1つの取引**で確定・取消できる。
- 2つの境界（`command.UnitOfWork`と`numbering.UnitOfWork`）は形が同じでも別パッケージのままにし、アダプタだけが両者を結ぶ。意味の中核が番号プロバイダへ依存せず、番号パッケージもCommandへ依存しない。

## 読み取れないこと

- 実DBの取引（両側とも参照用の記憶域・スタブ）。
- `CounterStore`の永続実装。
- 実Provider・実DBでの確認。

## 位置付け

参照実装の試験記録であり、製品の受入や本番適格性を示しません。アダプタの配線は参照実装で確認済み、実DBの取引と永続CounterStoreは未実装です。
