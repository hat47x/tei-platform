# Command境界の実行前検査と適用不明の記録（2026-11）

- 状態: **参照実装の変更案（A2）**。意味の受理・権限の正本・本番配備は扱わない。
- 関連: [#2996](https://github.com/hat47x/tei-platform/issues/2996)、[#2997](https://github.com/hat47x/tei-platform/issues/2997)、[#2999](https://github.com/hat47x/tei-platform/issues/2999)
- 対象コード: `internal/command/invoke.go`、`internal/command/safety_test.go`、`cmd/tei/cli_view.go`、`internal/studiohost/command.go`

## 1. なぜ必要か

Command境界は「宣言」と「実行」の間に立つ。これまでは、呼出し側が`Authorizer`や`PeriodState`を設定し忘れると、検査を省いて実行する既定だった。また、Executorが宣言外の効果を返した場合、境界はそれを「拒否」としてLifecycleへ渡していた。

どちらも、外部のHarnessやHostから見ると「検査済み」「適用されていない」と読み違えられる。この文書は、**未設定を成功にしない**、**適用有無が分からない失敗を拒否と混ぜない**という二つの判断を、コードと試験で固定する。

## 2. 判断したこと

| 対象 | 変更前 | 変更後 | 理由 |
| --- | --- | --- | --- |
| `Authorizer == nil` | 認可せず実行 | `COMMAND_AUTHORIZER_UNAVAILABLE`で拒否 | 未設定を「認可不要」と読む経路を残さない |
| 参照実行の認可 | 未指定 | `ReferenceAuthorizer{}`を明示 | 参照実行が**認可判断を省略した**ことを型で示す |
| 認可と期間の順序 | 期間→認可 | 認可→期間 | 認可されない呼出しに期間の開閉を返さない |
| `Periods == nil`で`period`宣言あり | 検査しない | `COMMAND_PERIOD_STATE_UNAVAILABLE`で拒否 | 宣言を検査できないことを「拒否なし」と扱わない |
| 宣言外の効果・結末 | rejectedとして記録 | `EffectUnknown`を立ててunknownとして記録 | Executor実行後に検出するため適用済みの可能性がある |
| 適用操作の不一致 | rejectedとして記録 | 同上 | `UnitOfWork`があってもExecutorの取引参加は契約外 |

`ReferenceAuthorizer`は「許可する」と同時に「本物の主体・権限・規則を評価していない」ことを名前で示す。本番用の認可契約ではない。

## 3. 変更後の検査順

```text
入力検証 (BuildCommandPlan)
  → Executor の存在
  → 認可 (Authorizer は必須)
  → 期間 (period が宣言されたら PeriodState は必須)
  → 取引の開始
  → Executor.Execute
  → 宣言との照合（効果・適用操作・結末）
  → Recorder
  → Commit
```

失敗した場合のLifecycleへの写像は次のとおり。

| 失敗 | Lifecycleの段階 | 再送 |
| --- | --- | --- |
| 入力不正・宣言なし・認可拒否・期間拒否 | observed / rejected | 同じowner/generationでは再送しない |
| `EffectUnknown`付きの失敗（commit失敗・rollback失敗・宣言外の効果） | unknown | 同じowner/generationでは再送しない。別owner・高いgenerationでtakeover |

## 4. 具体例

`tei view -command -execute`は参照実行であり、永続化しない。それでも`Authorizer`を省略しない。

```go
invoker := command.Invoker{
    Authorizer: command.ReferenceAuthorizer{},
    World:      loaded.World,
    Executor:   command.ReferenceExecutor{World: loaded.World},
    UnitOfWork: &command.MemoryUnitOfWork{},
}
```

これで、将来この初期化をコピーして別のHostを作った場合も、`Authorizer`を消すと`COMMAND_AUTHORIZER_UNAVAILABLE`で止まる。黙って無認可実行にはならない。

## 5. 根拠

- `go test ./internal/command -count=1`: PASS（`safety_test.go`を含む）
- `go test ./internal/command/... ./internal/execution/... ./internal/security/legacyauthorization/... ./internal/security/effectcommit/... ./internal/studiohost/... -count=1`: PASS
- `go test ./cmd/tei -count=1`: PASS
- `go build ./...`、`go vet ./internal/command ./cmd/tei ./internal/studiohost ./internal/security/legacyauthorization ./internal/security/effectcommit/...`: PASS
- 実行環境: Windows 11、go1.27.0。実PostgreSQLを使う試験はDSN未設定のためskip。

## 6. この文書が主張しないこと

- 本物の認可・主体・権限・メニュー制御が実装されたこと。
- `ReferenceAuthorizer`が本番で使えること。
- 宣言外の効果が実際にDBへ適用されたか、巻き戻されたかを区別できること。
- 実Provider・実DBで同じ結果を観測したこと。
- #2996・#2997が完了したこと。Issueの完了条件のうち、文書・出力契約・実DB確認は別途残る。
