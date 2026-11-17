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

- ~~他の Store 実装（SQLite・PostgreSQL の保存）が同じように子を適用するか~~ **確認済み**: `saveunit.Store` の実装は `MemoryStoreAdapter` と `FailingStoreAdapter` の2つだけで、後者は前者へ委譲する。**他の実装は存在しない**（「未確認」ではなく「無い」が正しい。独立レビューの指摘）。
- 行番号の再採番（F-21 が「画面の行番号は表示位置、業務上の行番号は別」と記録している点）。
- 明細行の状態（`CURRENT`/`NEW`/削除予定）を**宣言として**持つ方法。今は呼出し側が `Kind` に翻訳する。
- 実DB での観測。

## 位置付け

参照実装の試験記録であり、製品の受入や本番適格性を示しません。メモリアダプタの子適用は修正済み、他の Store 実装の確認と行状態の宣言は未実装。仮承認。
## 追記（2026-11-18）: 独立レビューが見つけた2つの穴を塞いだ

独立レビュー（読取専用）が、このPRの範囲に**2つの実害ある穴**を指摘した。どちらも直した。

### 1. 関係先の子が計画に載っていなかった

`Planner` の `collectRelations` は `rel.Children[0]` **だけ**を計画に載せていた。

```go
child := &r.Children[0]
```

一方、私が広げた `applyMemoryEntity` は `Relations[].Children` を**全件**適用する。したがって2件目以降の子は、**循環検査にも順序（`plan.Order`）にも載らずに適用されていた**。ヘッダ明細の確定は「明細行を全部1単位で運ぶ」形なので、これは実際に起きる。

`collectRelations` を全子へ広げ、計画が適用範囲を覆うようにした。

### 2. 同じキーを2度書く単位を拒否していない

`collect` は重複キーを**想定**しており（`if _, exists := flat[c.Key]; exists { continue }`）、`applyMemoryEntity` は宣言順に適用する。したがって delete → update の順で同じキーを置くと、update が pending の `nil`（削除）を「現在値なし」として読み、**行が部分的に復活する**。

計画は「2度書かれたもの」を順序できません。だから単位ごと拒否する（`SAVE_DUPLICATE_KEY`）。fail-closed を選んだ。最後の書き込みだけを黙って適用すると、矛盾が隠れる。

## 読み取れること

- 関係先の子が**全件計画に載る**（循環検査と順序が適用範囲を覆う）。
- 同じキーを2度書く単位は**拒否される**（部分的に復活しない）。

## 読み取れないこと

- `RelationChange` 自身（`Kind`/`Key`/`Fields`）は適用しておらず、`relation.Children` だけを適用する。**子を持たない関係変更（関係の削除など）は黙って落ちる。** 未修正。
- `MemoryStoreAdapter.Commit` は `commitableStore` が無いと**何も書かずに成功を返す**（既存の穴、今回の試験ストアは実装しているため露見しない）。
- `saveunit` はリポジトリ内に importer が無い。今回の修正効果は**パッケージ内試験に閉じている**。
