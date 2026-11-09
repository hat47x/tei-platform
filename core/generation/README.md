# 世代管理（Core Generation）

ここは、旧 `internal/composition/router.go` にあった **実行世代の保持と切替**の実装正本です。型 `Binding`・`Router`・`Lease`・`Retired` と、世代の `Acquire` / `Publish` / `Withdraw` / `Activate` / `Drain` を提供します。

- `core/effect.Scope` が一時的な資源の解放を担当し、世代のLeaseが残っている間は旧世代のDrainを完了しません。
- `Binding.Valid()` は識別子・Digest・世代などの**構造検証のみ**です。権限、契約適合、現在の受理、成果物の検証を意味しません。
- `Router.Acquire` と切替時に渡す **Host所有のGuard** は必須です。現在のPermission、Lease、Fencingなどの認可はHost側で検査します。
- Registryの契約選択、`Coordinator` のPrepare・Ready・ReplaceSelectedなどは `internal/composition` に残します。新たなContract Resolver、Host Authority、永続的なビジネス効果の巻戻しは実装しません。
- 旧 `internal/composition` の公開Go型はCore側への型エイリアスで、重複する正本を設けません。

## 検証入口

リポジトリ直下から実行します。

```sh
go test -count=1 ./core/generation ./internal/composition
go vet ./core/generation ./internal/composition
bash tools/check-core-isolation.sh
bash tools/check-core-identity-isolation.sh
go test -count=1 ./tools/repository-architecture-check
```

隔離検査では、実ソースを一時的なGoモジュールに取り込みます。サブプロジェクトごとの永続的な `go.mod` は必要ありません。依存境界検査は `core/` 内の本番GoファイルからTEI内の実装層への直接importを禁止します。外部ライブラリや各プラグインに対する一律の性能・依存制約ではありません。

これはIssue #1251の段階的Core移管の一部であり、Service Contract全体や製品用Composition Kernelの完成を宣言するものではありません。
