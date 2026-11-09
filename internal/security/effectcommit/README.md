# 認可と副作用の同一確定境界

本パッケージは、実運用の保存先が満たす必要のある**最小の接続契約**です。保存先の実装や証明書ではありません。

既存の`security.authorize`で得た版付き許可を受け取り、保存先アダプターに次の処理を**一つの物理トランザクション**として要求します。

1. 主体・操作・対象の一致を確認し、認可正本の主体リンク・所属・規則版を確定時に検査する。権限変更とこの検査を直列化する。
2. **その後で**同一操作識別子の確定記録を確認する。別要求内容や別主体の記録は返さない。
3. 未確定なら対象データの版を比較更新する。同じトランザクションで、操作識別子・要求指紋・新しい対象版を確定記録として保存する。
4. 確定結果の応答が失われた場合は`CommitUncertainError`を返す。確実な不成立と誤認せず、**同じ操作識別子**で再認可・照会・再試行する。

`Runner.Execute`と`Runner.Lookup`はこの順序と入力の一致を保証する。ただし`Transaction`のメソッドが同じDB・同じロックに参加しているか、変更版の正本が実際の所属・権限データから発行されたかは**このGoインターフェースでは証明できない**。SQLiteの実接続試験は`reference/effect-fence`で継続し、実運用Providerには個別の適合性試験と証拠を要求する。

## 実装側の必須条件

- `Begin`から`Commit`まで全メソッドは同じ物理トランザクションに参加する。別のDBや遠隔認可APIへ版照合を委任した場合は永続的な原子的保証を認めない。
- `CheckAuthority`は主体リンク・所属・規則版を検証し、確定まで版の変更が割り込めないようにする。権限不足は`false,nil`、読取り障害や正本の欠落・破損は`error`で返す。既存の役割判定式を複製しない。
- `FindReceipt`と`SaveReceipt`は保存先全体での`OperationID`一意制約を持ち、更新データと同時に永続化する。独立サービスのキュー発行はこの契約の外に置く。
- `WriteResource`は`ExpectedVersion`の一致を保証する比較更新を行う。対象の版が違えば`ErrResourceConflict`を返す。
- DBコミットの応答喪失は「処理されなかった」と断定できない。確定記録を確認できなければ結果不明を維持する。
- Application/Capabilityは`security.authorize`の従来Grantには含まれないため、Provider側の`CheckAuthority`がScope全体の適用対象を検証すること。呼出側が版付きGrantを任意に作成してはならない。

## 変更範囲の試験

```bash
go test -race -count=3 ./internal/security/effectcommit
go vet ./internal/security/effectcommit
```

競合・版失効・再送・操作識別子衝突・対象版の衝突・確定記録保存障害の巻戻し・コミット応答喪失を、同一ロックを持つ模擬トランザクションで検証する。模擬試験の合格を実DBの永続保証へ格上げしない。

関連する実SQLite競合試験は`.github/workflows/tei-effect-fence-sqlite-ci.yml`で実行する。製品での実Provider証明、Web/CLIへの統合、外部SaaS・キューの結果不明と復旧はIssue #1174の未完了事項とする。
