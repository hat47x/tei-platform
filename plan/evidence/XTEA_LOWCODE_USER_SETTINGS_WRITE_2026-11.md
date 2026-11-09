# 利用者設定の書込みの確認（2026-11）

## 対象

- 実装: `internal/usersetting/change.go`（`Change`・`Table.Change`・`Table.Apply`・`Store`・`MemoryStore`）
- 試験: `internal/usersetting/change_test.go`
- 対象版: `main` の `3c8bebefe` から分岐した`feat/user-setting-write-20261109`
- 環境: Windows、Go `1.27.0`

## 実行

```powershell
go test ./internal/usersetting -run 'TestChange|TestApply|TestMemoryStore' -count=1 -v
```

## 観測

- `TestChangeUpdatesADeclaredKey` 成功。宣言済みの鍵を同じ位置で置き換え、他の行を保つ。
- `TestChangeRefusesAnUndeclaredKey` 成功。`page_sise`のような綴り誤りは拒否され、2つ目の設定を作らない。
- `TestChangeAddsAUsersRow` 成功。`alice`の行を共有の既定の隣へ足し、`alice`は`200`、`bob`は共有の`50`を読む。
- `TestChangeRefusesAUserOnASharedData` 成功。利用者の役割を持たないDataへ利用者を指定すると拒否される。
- `TestChangeRequiresAKey` 成功。鍵の無い変更は拒否される。
- `TestApplyRecordsTheCheckedList` 成功。検査と書込みが1回で済み、**拒否された変更は格納へ届かない**。
- `TestApplyRefusesAMissingStore` 成功。格納の不在は拒否される。
- `TestMemoryStoreDistinguishesAbsentFromEmpty` 成功。未書込みと「空で書いた」を区別する。
- パッケージ全体は`ok`。

## 読み取れること

- 設定の更新が**宣言された鍵の範囲**で行われる。X-TEAは`setVariant`・`setUserValueOf`で任意のキーを書けた（`Session.java` L2112、`XFScriptable.java` L56–60）ため、綴り誤りが新しい設定を作り、誰も読まない値が増えました。
- 利用者の行の追加と共有の既定の維持が、同じ変更で両立する。
- 検査と書込みが同じ呼出しなので、検査した表と書く表がずれない。

## 読み取れないこと

- 変更の**監査**（誰がいつ何を変えたか）。今は最後の一覧を格納するだけで、履歴を持ちません。
- 権限（誰が設定を変えられるか）。
- 設定値の型の絞り込み（`page_size`が整数であること等）。
- Workspace例への設定Data追加、CLI/Studioからの設定更新。
- 実DB・旧Runtimeの実測。

## 位置付け

参照実装の試験記録であり、製品の受入や本番適格性を示しません。設定更新の検査と記録は参照実装で確認済み、監査・権限・画面は未実装です。
