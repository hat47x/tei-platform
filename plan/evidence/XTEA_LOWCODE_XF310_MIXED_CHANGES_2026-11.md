# XF310（見出し明細保守）の一確定で3種の変更を載せる（2026-11-17）

## 対象

- 実装: `internal/saveunit/memory.go` の `applyMemoryEntity`（子と関連先へ再帰するよう修正）
- 試験: `internal/saveunit/mixed_change_kinds_test.go`
- 知識目録 **F-21**（見出し明細保守の行の状態）

## 見つけたこと

知識目録 F-21 は、X-TEA の見出し明細保守が明細行を `CURRENT`・`NEW`・削除予定の状態で持ち、**確定時に更新・追加・削除を振り分ける**と記録している。つまり1回の確定が3種の変更を同時に運ぶ。

TEI の `saveunit` は `EntityChange.Kind` で create/update/delete を持ち、`Children` と `Relations` で親子を1つの単位に入れられる。**設計としては受け止められている。**

しかし**実際には子が適用されていなかった。**

```go
// 修正前の applyMemoryEntity
switch change.Kind {
case ChangeKindCreate: ...
case ChangeKindUpdate: ...
case ChangeKindDelete: ...
}
*applied = append(*applied, AppliedChange{...})
return nil      // Children も Relations も見ていない
```

`Saver.Save` は `Planner` で順序を決め（`plan.Order`）、スナップショットの取得にはその順序を使う。しかし `Store.ApplyChanges(ctx, unit, tx)` は `unit` をそのまま受け取り、`MemoryStoreAdapter` は `unit.Changes`（**トップレベルだけ**）をループしていた。**子は計画にも数えられるのに、適用されない。**

これは試験で見落とされていた。既存の `Children` を使う試験は `Planner.Plan` の**順序だけ**を確かめており、アダプタを通して**適用**していなかった。

## なぜ危ないか

**成功として報告される。** `applied` に載るのはトップレベルだけで、`Save` はそれを成功として返す。明細行が書かれていないのに「確定した」と読める結果になる。X-TEA の「一確定で振り分ける」を写したつもりで、**行が落ちる**。

これは D-9 の変種である（宣言はある、計画もある、しかし適用されない）。

## 変更

`applyMemoryEntity` が `Children` と `Relations[].Children` へ**宣言順に再帰**するようにした。

- 順序は宣言順（X-TEA の「行を並べ替えてから」ではなく、呼出し側が並べた順）。依存順の解決は `Planner` の仕事であり、アダプタは渡された単位をそのまま適用する。
- `applied` には子も載る。呼出し側が「何が適用されたか」を読める。

## 観測

`TestOneSaveUnitCarriesHeaderUpdateAndEveryDetailChange`（新規）:

- 見出しの `update`（status → confirmed）と、明細の `update`（qty 2→5）・`create`（line 4）・`delete`（line 3）を**1つの保存単位**に入れる。
- 4つすべてが適用される。
- **触っていない行（line 2）は変わらない**（1確定が「ページ全体の書き換え」にならない）。

この試験は修正前に落ちた（`the detail update did not land: 2`）。修正後に通る。

- `internal/saveunit`・`internal/command`・`internal/operations`・`internal/applicationview` は ok。
- `go build ./...`・`go vet`・`gofmt -l`・`tools/repository-plan-check` 成功。

## 読み取れること

- **1回の確定が、見出しの更新と明細の追加・更新・削除を同時に運べる**（F-21 の形）。
- 子と関連先が**実際に適用される**（修正前は計画にだけ載っていた）。
- 触っていない行は影響を受けない。

## 読み取れないこと

- **他の Store 実装**（SQLite・PostgreSQL の保存）が同じように子を適用するか。今回はメモリのアダプタだけを直した。**同じ穴が他にもある可能性がある。**
- 行番号の再採番（F-21 が「画面の行番号は表示位置、業務上の行番号は別」と記録している点）。
- 明細行の状態（`CURRENT`/`NEW`/削除予定）を**宣言として**持つ方法。今は呼出し側が `Kind` に翻訳する。
- 実DB での観測。

## 位置付け

参照実装の試験記録であり、製品の受入や本番適格性を示しません。メモリアダプタの子適用は修正済み、他の Store 実装の確認と行状態の宣言は未実装。仮承認。
