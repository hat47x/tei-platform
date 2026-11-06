# 権限の強制の適用計画（2026-11）

- 状態: **計画**。既存アダプタの移行は未着手です。
- 目的: 「誰が付与を決め、誰が突き合わせ、どこから移すか」を決める。
- 関連: [権限の現状調査](PERMISSION_ENFORCEMENT_SURVEY_2026-11.md)、[アクセス経路の棚卸し](ACCESS_PATH_INVENTORY_2026-11.md)

## 1. 役割の分担（既存の宣言に合わせる）

| 役割 | 既存の持ち主 | 内容 |
| --- | --- | --- |
| **要求** | プラグイン能力（`Capability.Permissions`） | その能力が何を必要とするか |
| **付与** | 環境（`PermissionGrants`） | 能力ID・種別・`Permission`・`ResourceBinding` |
| **突き合わせ（著者時）** | 検証（`checkCapabilityPermissions`） | 要求が付与で覆われているか、資源の対応があるか |
| **突き合わせ（実行時）** | **まだ無い** | 実行の直前に同じ対応を再確認し、記録する |

新しい政策は足しません。**著者時に確かめている対応を、実行時にもう一度確かめる**だけです。実行時にもう一度確かめる理由は、著者時の受理と実行時の環境が同じとは限らないためです（環境の取り違え、古い世代の実行）。

## 2. 標準操作は対象外（明示）

アプリケーションの標準操作（`list`・`read`・`create`・`update`）は意味の能力であり、`Permissions` を持ちません。したがって**権限の突き合わせの対象外**です。対象になるのは、外部に触る能力（プラグイン能力・参照アダプタ）です。

## 3. 移行の順序

| 順 | 対象 | 理由 |
| --- | --- | --- |
| 1 | プラグインの実行（`internal/plugin/runtime`） | 要求と付与の宣言が既にあり、著者時の検証もある。実行時の再確認を足すだけで済む |
| 2 | 参照アダプタ（`tools` のうち外部に触るもの） | 要求の宣言が無いので、まず宣言を足す必要がある |
| 3 | CLI（`cmd/tei`） | 書き込み先は引数で決まる。付与の出所を決めてから |

## 4. 「通っていない経路」の扱い

移行は一斉にはできません。**通っていない経路を一覧で維持**し、強制が効く範囲を偽らないようにします。

- `access.FileSystem`・`access.Network`・`access.Secrets` を通るアダプタ: 強制あり
- `os`・`net` を直接呼ぶアダプタ: **強制なし**（一覧に載せる）

一覧は [`ACCESS_PATH_INVENTORY_2026-11.md`](ACCESS_PATH_INVENTORY_2026-11.md) の棚卸しを基準に更新します。

## 5. 最初の一歩（提案）

1. `internal/execution` に、環境の `PermissionGrants` から `access.Guard` を組み立てる関数を置く。
2. プラグイン実行の直前に `Guard.Check` を呼び、拒否は `blocked` として返す（期間・効果・権限の停止条件と同じ形）。
3. 拒否・許可を `access.File` に記録する。

## 6. この計画が決めていないこと

- 参照アダプタの要求を誰が書くか（著者は開発者か、受理の一部か）。
- OS のサンドボックスをどこまで担うか（実行環境側の担当として切り分ける想定）。
- 付与の変更を誰が承認するか。
## 7. 最初の一歩の実装状況（2026-11-06 追記）

`execution.GuardFromGrants(grants, capabilityKind, capabilityID, journal)` を追加しました。

- 環境の `PermissionGrants` から、対象能力の付与だけを選んで `access.Guard` を組み立てる。
- 選ぶ条件は**種別と識別子の両方**。識別子が同じでも種別が違えば別の能力として扱う（`plugin` と `service` を取り違えない）。
- 組んだ `Guard` は判定と記録をそのまま行う。

これで「環境の付与 → 実行時の突き合わせ」の入口ができました。**プラグイン実行の直前に呼ぶ配線はまだです**。
## 8. 追記（2026-11-06）: 宣言の引き出し口

`execution.RequirementsFromCatalog(catalog)` を追加しました。プラグインのカタログから、能力ごとに宣言された権限を引き出します。

```go
requirements := execution.RequirementsFromCatalog(catalog)
preconditions := execution.PermissionPreconditions(requirements, granted)
decision, results, err := execution.RunPlan(ctx, plan, key, journal, step, execution.CombinePreconditions(preconditions, period, effects))
```

これで**実行計画の停止条件**に権限を載せるのに必要な部品が揃いました（カタログ → 宣言 → 停止条件）。
## 9. 追記（2026-11-06）: 成果物への反映

機能定義書に権限の行を出すようにしました（`internal/artifact/functiondef.go`）。

```
permission sales.order.register filesystem write /var/reports
permission sales.order.register network connect db:5432
```

- 能力の識別子順、その中では `Chars`（種別・アクセス・スコープ）順。
- 宣言の無い能力は行を出さない。
- 生成規則 `function-definition/v1` の行種別に `permission` を追加。

**未着手**: 作業領域の読み込みからこの入力を渡す配線（いまは試験でのみ入力している）。標準操作の能力は権限を宣言しないため、行が出るのは外部に触る能力だけです。