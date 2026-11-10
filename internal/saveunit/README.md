# 汎用保存機構（saveunit）

`internal/saveunit` は、TEIの標準データモデル（セマンティックモデル）を前提に、親子関係・複合キー・参照関係を持つ階層データを安全に永続化するための汎用契約と参照実装を提供する。

## 設計方針

- **Data Provider非依存**: 保存処理の共通契約（`Store` インターフェイス）を定義し、DB固有処理は各Providerが吸収する。
- **モデル主導**: セマンティックモデルのリレーション定義から保存順序を topological sort で導出する。
- **明示的な部分更新**: 未変更・NULL・削除・値の欠落を明確に区別し、意図しないデータ消失を防止する。
- **トランザクション境界の明確化**: 保存処理は `UnitOfWorkTx` / `Transaction` の共通契約で包み、DBトランザクションと外部副作用の境界を明示する。

## 主要な型

| 型 | 役割 |
|---|------|
| `SaveUnit` | 保存要求のルート。Application/Pattern/World/Changes/Hooks を持つ |
| `EntityChange` | 1エンティティの変更意図（Create/Update/Delete） |
| `RelationChange` | リレーション経由の子エンティティ変更 |
| `FieldChange` | フィールド単位の変更意味（Value/SetNull/Delete/Unchanged） |
| `PatchSet` | 1エンティティのフィールド変更マップ |
| `Planner` | セマンティックモデルから保存順序を plan する |
| `Saver` | 検証→フック→適用→コミット/ロールバック を実行する |
| `Store` | Data Provider が実装する永続化の共通契約 |
| `Transaction` | 保存処理のトランザクション境界 |
| `Hook` | 保存前後の検証・変換・編集フック |

## 保存順序（Planner）

`Planner` は `SemanticWorld` からリレーション定義を取得し、依存関係をグラフ化して topological sort を実行する。

- 親エンティティが子エンティティより先に処理される
- 循環依存（Cycle）は `PlanResult.Cycles` で検出され、保存は拒否される
- 複合キーは `EntityChange.Key` 文字列で表現され、親子の紐付けに使用される

## 部分更新のセマンティクス

`FieldChange` は4つの状態を持つ:

| 状態 | 意味 | 保存時の挙動 |
|------|------|-------------|
| `Value` | 明示的な値の設定 | 指定値を保存 |
| `SetNull` | 明示的なNULL代入 | 空文字またはNULLとして保存 |
| `Delete` | フィールドの削除 | 列自体を削除または除外 |
| `Unchanged` | 変更しない | 既存値を保持 |

これにより、`PatchSet` に含まれないフィールドは「指定がなかった」として扱われ、既存値を保持する。

## フック機構

`HookPlan` により、保存の前後に検証・変換・編集フックを挿入できる。

- `Validation`: 保存前に実行（制約チェックなど）
- `BeforeSave`: 適用前に実行
- `AfterSave`: コミット後に実行
- `OnFailure`: 失敗時に実行

フックが `Allow: false` を返した場合、保存はロールバックされ `HookFailure` を返す。

## トランザクションとロールバック

```go
type UnitOfWorkTx struct {
    Start func(ctx context.Context) (Transaction, error)
}

type Transaction interface {
    Commit(ctx context.Context) error
    Rollback(ctx context.Context) error
}
```

- `Saver.Save` は `UnitOfWorkTx.Begin` でトランザクションを開始する
- 適用中にエラーが発生した場合、`Rollback` を実行し、変更を破棄する
- すべての変更が正常に適用された後、`Commit` で確定する

## 楽観的排他制御

フック機構を利用して楽観的排他を実現できる。

```go
hook := HookFunc(func(ctx context.Context, input HookInput) HookResult {
    snap, _ := store.ReadSnapshot([]EntityKey{{Data: "sales.order", Key: "order=ORD-1"}})
    if row, ok := snap[EntityKey{...}]; ok && row["sales.order.status"] != nil && *row["sales.order.status"] == "closed" {
        return HookResult{Allow: false, Message: "concurrent modification detected"}
    }
    return HookResult{Allow: true}
})
```

## Data Provider との統合

`Store` インターフェイスを実装することで、任意のData Providerを統合できる。

```go
type Store interface {
    ReadSnapshot(keys []EntityKey) (StoreSnapshot, error)
    ApplyChanges(ctx context.Context, unit *SaveUnit, tx Transaction) (StoreSnapshot, []AppliedChange, error)
    Release()
}
```

### 参照実装: MemoryStoreAdapter

`internal/saveunit/memory.go` には、テスト用の `MemoryStoreAdapter` を提供する。

- `MemoryStore` インターフェイスを実装したインメモリストア
- トランザクション境界をシミュレート
- ロールバック、再試行、フックの動作を確認可能

### DBアダプタの実装指針

実際のDBアダプタ（PostgreSQL, SQLite等）は以下の責務を持つ:

1. `ApplyChanges` 内でトランザクションを管理
2. `PlanResult.Order` に従って親→子の順で INSERT/UPDATE/DELETE を実行
3. 複合キーを WHERE 句として使用
4. 外部キー制約に応じた `ON DELETE CASCADE/RESTRICT` の処理
5. 制約違反・一意性違反を `SaveError` として報告
6. 楽観的排他用の `generation` カラムまたはタイムスタンプの検査

## テスト

```bash
go test ./internal/saveunit -count=1 -v
```

### テストケース一覧

| テスト | 内容 |
|--------|------|
| `TestPlannerOrdersParentBeforeChildren` | 親が子より先に保存される |
| `TestPlannerRejectsDependencyCycles` | 循環依存を検出して拒否 |
| `TestSaveUnitPreservesMissingVersusNullVersusDelete` | Value/SetNull/Delete/Unchanged の区別 |
| `TestSaveUnitDoesNotLoseUnspecifiedFields` | 未指定フィールドの保持 |
| `TestSaveUnitRejectsEmptySetReplacement` | 空集合置換の拒否 |
| `TestSaveUnitRollsBackOnMidSaveFailure` | 途中失敗時のロールバック |
| `TestSaveUnitRetriesTransientFailure` | 過渡的失敗の再試行 |
| `TestSaveUnitOptimisticConflictViaHook` | 楽観的競合の検出 |
| `TestSaveUnitBeforeAfterHooksRunAroundCommit` | Before/Afterフックの実行 |
| `TestSaveUnitRollsBackOnHookFailure` | フック拒否時のロールバック |
| `TestPlannerOrdersCompositeKeyChildren` | 複合キーの子の順序 |
| `TestSaveUnitCompositeKeyUpdatePreservesUnspecifiedFields` | 複合キー部分更新の保持 |

## Owner受理が必要な事項

[plan/issues/SAVEUNIT_OWNER_ACCEPTANCE_ITEMS_2026-11.md](../plan/issues/SAVEUNIT_OWNER_ACCEPTANCE_ITEMS_2026-11.md) を参照。
