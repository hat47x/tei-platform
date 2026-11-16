# `StageUnknown`からの復帰をCLIから実行する経路（2026-11-17）

## 対象

- 実装: `cmd/tei/cli_view.go`（`-lifecycle-takeover`）、`cmd/tei/cli_flag_contract.go`、`cmd/tei/cli_contract.go`
- 試験: `cmd/tei/cli_lifecycle_takeover_test.go`、`cmd/tei/testdata/takeover-prepare/`
- 対象版: `f1d09ac8f`（`main`）
- 環境: Windows、Go `1.26.10`

## これまで何が無かったか

`internal/command` には `Lifecycle.Takeover` と `DispatchOptions.Takeover` があり、段階排他の試験（`TestTwoProcessesCannotBothTakeOver`）も通っていました。しかし **CLI からその操作を実行する入口がありません**でした。

つまり、適用有無が確定しない実行（`StageUnknown`）が起きたとき、運用者がコマンドラインから復帰させる手段が無く、**「実運用手順」が存在しない**状態でした。

## 実装

`view -execute` に任意指定の `-lifecycle-takeover` を追加しました。

- `-lifecycle-state` と `-lifecycle-command` が必須（どちらか欠ければ `INVALID_ARGUMENT`）
- `DispatchOptions.Takeover = true` を渡すだけで、**新しい機構は足していません**。`LifecycleInvoker` が `Dispatch` の代わりに `Takeover` を呼びます
- takeover は成功ではありません。段階は `dispatched` のままで、`Observe` して初めて結末が記録されます

### 途中で見つけた誤り（重要）

最初は CLI 側で `store.Takeover(...)` を呼んでから `LifecycleInvoker.Invoke` を呼ぶ形にしました。ところが `Takeover` は記録を既に `dispatched` にするため、直後の `Dispatch` が「already in flight」で**拒否**されました。実測で確認したうえで、その実装を捨て、既存の `DispatchOptions.Takeover` を使う形に直しました。

**二重 dispatch は、takeover が成功したのに失敗に見える**という最悪の読み方になります。実装を足す前に、既存の入口が無いかを確かめるべきでした。

## 実測

実プロセスで確認しました（`lifecycle-takeover` の有無で3通り）。

| 操作 | 結果 |
| --- | --- |
| 通常実行（`owner-a`/1）→ 同じコマンドを takeover（`owner-b`/2） | `COMMAND_LIFECYCLE_REFUSED`「takeover requires an unknown outcome, command is observed」 |
| `unknown` を用意 → takeover（`owner-b`/2） | **成功**（`ok:true`、コマンドが実行され結末が記録される） |
| 同じ takeover の2回目 | `COMMAND_LIFECYCLE_REFUSED`「takeover requires an unknown outcome, command is dispatched」 |

`unknown` は CLI から作れない（参照実行は結末を観測するか失敗する）ため、試験では段階ストアへ直接書いています。`cmd/tei/testdata/takeover-prepare/` がそれを行い、`internal` パッケージを import するため module の内側に置いています。

## 試験（2件）

- `TestLifecycleTakeoverRecoversAnUnknownOutcome` — `unknown` から takeover で復帰し、2回目は拒否される
- `TestLifecycleTakeoverRefusesAClaimedCommand` — 観測済みのコマンドに takeover を使うと拒否される（**このフラグが observed-once 規則の抜け道にならない**ことの確認）

## 実行

```powershell
go test ./cmd/tei -run 'TestLifecycleTakeover' -count=1
go test ./cmd/tei -run 'TestCommandCatalogFlagsMatchAcceptedFlags|TestContractDocumentCoversEveryAdvertisedCommand' -count=1
go vet ./cmd/tei
gofmt -l cmd/tei
```

## 読み取れること

- 運用者がコマンドラインから `StageUnknown` のコマンドを引き継げる。
- takeover は観測済み・実行中のコマンドには使えない（拒否される）。

## 読み取れないこと

- **実際の障害時の運用手順**（誰が、どの条件で takeover を実行するか）。仕組みはありますが、運用の決まりは未決定です。
- 複数運用者が同時に takeover を試みた場合の調停（段階排他の試験はありますが、実運用は未検証）。
- `unknown` を生む実際の失敗（commit 失敗など）からの復帰を、実Providerで通した記録はありません。

## 位置付け

参照実装の確認記録であり、製品の受入や本番適格性を示しません。`AC-5`の「`StageUnknown`からの復帰を跨ぐ結末の実運用手順」のうち、**経路を用意した**部分の記録です。運用の決まりは未決定のまま残します。
