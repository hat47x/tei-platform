# 一時Effect Scope

`core/effect/` は、サービス契約、Providerの選択、Pluginの種類、認証、データベース、実行先を知らずに、一時的な資源の解放を登録・実行する小さなCore値・動作単位です。`internal/composition/scope.go`にあった製品実装の**唯一の正本**をこちらへ移しました。

`Scope.Track` は、一時資源を確保した後に、その解放関数を登録します。`Scope.Close` は登録済み関数を逆順に実行し、失敗・panicがあっても残りの解放を試みます。複数の`Close`は同じ解放結果を受け取り、解放の再試行は行いません。閉鎖後の`Track`は`ErrScopeClosed`で拒否します。

旧`composition.Scope`はGoの**型エイリアス**であり別の状態管理器を持ちません。`composition.NewScope`と`composition.ErrScopeClosed`も互換入口です。したがって古い利用者と新しい利用者は同一の型・エラー識別子を使います。

## 試験

```sh
go test -count=1 ./core/effect ./internal/composition
bash tools/check-core-isolation.sh
```

隔離試験では`core/effect`と旧`internal/composition`の実ソースを**それぞれ別の一時Goモジュール**へ変更せず複製し、ネットワークのモジュール取得を禁止して試験・Vetを実行します。別の利用者モジュールからも、旧`composition.Scope`と新しい`effect.Scope`が同一型であることをコンパイル・実行で確認します。依存を理由にコア全体へ`go.mod`を追加することはしません。

## 保証の限界

このScopeは、プラグインを構築・起動する権限、契約の発行・適合性、実行世代のCAS切替、別プロセス・クラスタの後始末、永続的な外部副作用の取り消しを保証しません。登録した解放関数の実行は行いますが、物理資源が実際に解放された証拠や安全なロールバックを作るものではありません。ホストがAuthorityとGenerationを保持する境界は[#951](https://github.com/hat47x/tei-platform/issues/951)・[#1223](https://github.com/hat47x/tei-platform/issues/1223)・ADR-0032の担当です。

当面は旧Compositionの型とAPIを保ちつつ、将来の正式な公開契約と独立した、一時Effect管理の正本として利用します。
