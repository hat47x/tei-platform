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

## 6. `committed` と `persisted` の分離（#2998 追記）

### 見つかった誤読

`tei view -command -execute` とStudioの参照実行は、どちらも `committed: true` を固定で返していた。参照実行の `UnitOfWork` は `MemoryUnitOfWork` で、Commitの呼出し回数を数えるだけで何も保存しない。しかし応答の欄名と値だけを見ると「保存された」と読める。

### 決めたこと

| 欄 | 意味 | 参照実行の値 |
| --- | --- | --- |
| `committed` | 構成された `UnitOfWork` の `Commit` が成功したこと。取引境界の呼出し結果であり、永続化の証拠ではない | `true` |
| `persisted` | 実行先へデータが保存されたこと | `false` |
| `durability` | 保存境界の種類。`none` は保存しない参照実行 | `"none"` |

`committed` は名称も意味も変えない。既存の読み手（Studio画面、`product_slice_test.go`、外部Harness）が使っているため、欄を削除・改名する案は採らない。`persisted` と `durability` は**欄の追加**であり、`tei.cli.view/v0` の互換な変更の範囲に収まる（[CLI JSON出力契約](CLI_JSON_OUTPUT_CONTRACT_2026-10.md) §5）。

CLIのテキスト出力は `committed: true (reference unit of work)` と `persisted: false (durability: none)` を別行で出す。Studioの画面も同じ3欄を別行で表示する。

### この分離が防ぐこと

- 参照実行の成功を、実DBへの保存成功と混同しない。
- 実Providerを接続した後も、`committed`（取引のCommit）と `persisted`（独立した読取りで確認した保存）を別の欄として扱える。将来 `persisted` を「独立読取りで確認済み」の意味に強める場合は、`durability` の値と合わせて別の判断として記録する。

### まだ決めていないこと

- 実Provider接続時に `committed` を「取引のCommit成功」と「独立した読取りで確認できたこと」のどちらへ対応させるか。本追記では参照実行の意味だけを固定し、実Providerの意味は未決のまま。
- `schemaVersion` を `v1` へ上げる必要があるか。欄の追加だけなので `v0` のままと判断したが、外部読み手の互換性確認は別途必要。

## 7. 効果を宣言しないCommandの `appliedUnobserved`（#3068 追補）

### 見つかった誤読

`OperationRecord.AppliedUnobserved` は「Executorが適用操作を列挙しなかった」ことを示す。しかし `Applied` が空になる理由は二つある。

1. Executorが列挙できない（**未観測**）。
2. 能力が効果を宣言していないので、比較する対象が無い（**比較対象なし**）。

どちらも `len(result.Applied) == 0` になるため、単純な実装では2を「未観測」と表示していた。効果を宣言しないCommandでは `difference-check: not observed` が常に出る。

### 決めたこと

`AppliedUnobserved` は **能力が効果を宣言している場合だけ** true にする。

- `plan.Effect != nil && len(result.Applied) == 0` → `appliedUnobserved:true`
- `plan.Effect == nil` → `appliedUnobserved:false`

CLIとStudioは、記録が無い場合の既定値を「効果が宣言されていればtrue、されていなければfalse」にした。効果なしのCommandでは `difference-check: not observed` を出さない。

### この分離が防ぐこと

- 「比較する対象が無い」を「Executorが答えられなかった」と読み違えない。
- 効果なしのCommandに、存在しない差の確認を要求しない。

### まだ決めていないこと

- 効果を宣言したが `Applied` が空のままのExecutorを、構成時に拒否するか（#2996の未決）。
- `AppliedUnobserved` を `Differences` の有無と独立に扱うか、`differenceCheck` のような三状態（`checked`・`not-observed`・`not-applicable`）へ改めるか。

## 8. この文書が主張しないこと

- 本物の認可・主体・権限・メニュー制御が実装されたこと。
- `ReferenceAuthorizer`が本番で使えること。
- 宣言外の効果が実際にDBへ適用されたか、巻き戻されたかを区別できること。
- 実Provider・実DBで同じ結果を観測したこと。
- #2996・#2997・#2998が完了したこと。Issueの完了条件のうち、実DB確認と外部読み手の互換性確認は別途残る。

## 8. 実Provider経路での `committed` の候補（未採用）

参照実行の意味は §6 で固定した。実Providerを接続したときの `committed` をどう扱うかは、まだ決めていない。候補を整理する。

| 案 | `committed` の意味 | `persisted` の意味 | 利点 | 欠点 |
| --- | --- | --- | --- | --- |
| P1: 取引のCommit成功 | ProviderのCommitが成功した | 独立した読取りで確認できた | 現在の欄名と一致 | 「Commitした」と「読めた」を混同しやすい。P1だけでは保存の証拠にならない |
| P2: 独立読取りでの確認 | 独立した読取りで対象状態を確認できた | `committed` と同じ | 観測に基づく | 既存の読み手の意味が変わる。`v1` が必要 |
| P3: 取引境界の結末 | ProviderのCommitが成功した | 独立した読取りで確認できた。未実施は`null` | `null`で「まだ確認していない」を表せる | JSONの`bool`から`*bool`への型変更は互換でない |

この文書はどれも採用しない。実Providerの試験（`TEI_EFFECT_POSTGRES_DSN`）でCommit成功と独立読取りを同じ実行で観測できるようになった時点で、候補ごとの結果を記録してから決める。

`committed` をP2へ変える場合は `tei.cli.view` の版を上げる。P1のまま `persisted` を独立読取りの意味に強めるなら、`persisted` が「確認済み」であることを示す別欄（例: `persistenceObserved`）が要る。
