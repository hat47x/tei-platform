# 行範囲フックへ行集合を渡す呼出し側の入口（2026-11-13）

## 対象

- 実装: `cmd/tei/cli_view_hooks.go`（`runDeclaredHooksWithRows`・`runDeclaredHooksHalfWithRows`）、`internal/studiohost/hooks.go`（`runHooksWithRows`・`runHooksHalfWithRows`）
- 試験: `cmd/tei/cli_view_hooks_rows_test.go`、`internal/studiohost/hooks_rows_test.go`
- 状態文書: `docs/status.md`（`Runner.Rows`を設定するホストが無いという記述の更新）
- ブランチ: `fix/hook-row-scope-wiring-r104`

この記録は、`internal/hook`の行単位実行（`RunRows`・`applyReachableRows`）が既にあるのに、CLIとStudioの呼出し側が`hook.Runner.Rows`を埋めないため、`scope row`の宣言が参照実行で理由つきのskipになる、という穴に対する作業である。

## 1. 事実確認（依頼の前提をコードで確かめた）

### 1.1 `hook.Runner.Rows`の全参照（変更前）

`git grep -n "r\.Rows\|Runner.Rows\|Rows \[\]map" -- internal/hook cmd/tei internal/studiohost`の結果、Goコードでの参照は次のとおり。

| 場所 | 内容 |
| --- | --- |
| `internal/hook/run.go`（宣言） | `Rows []map[core.CanonicalID]string`。単票の実行では空で、「行集合が必要」を報告する前提がコメントに書かれている |
| `internal/hook/run.go`の`Run` | `len(r.Rows) > 0`なら`RunRows`へ委譲。空なら`HOOK_SCOPE_NEEDS_ROWS` |
| `internal/hook/reach.go`の`runReachable` | `scope == "row" && len(r.Rows) > 0`で`applyReachableRows`へ。空なら`skipped`と理由 |
| `internal/hook/reach_rows.go` | `applyReachableRows`が`r.Rows`を1行ずつ回し、`uses`は行ごとに`RunRowCalls`へ |
| `internal/hook/scope_through_test.go` | 試験が`runner.Rows`を明示的に設定（宣言→検証→実行の貫通試験） |

**`cmd/tei`と`internal/studiohost`には`Rows:`の設定が1件も無かった。** つまり行経路は実装済みで、入口が無かった。

### 1.2 `runDeclaredHooks`系の呼出し元

| 呼出し元 | 呼ぶ関数 |
| --- | --- |
| `cmd/tei/cli_view.go`（`view -command -execute`、`-reference-store`なし） | `runDeclaredHooks` |
| `cmd/tei/cli_view.go`（`-reference-store`あり、変更前＝before-*の半巡） | `runDeclaredHooksHalf` |
| `cmd/tei/cli_view_hooks.go`（`runDeclaredHooksWithRule`） | `runDeclaredHooksHalf` |
| `cmd/tei/cli_view_hooks.go`（`runAfterHalfRevertingOnRefusal`、変更後＝after-*の半巡） | `runDeclaredHooksHalf` |
| 試験（`cmd/tei/cli_view_hooks_test.go`・`cli_view_hooks_half_test.go`） | `runDeclaredHooks`・`runDeclaredHooksWithRule`・`runDeclaredHooksHalf` |

いずれも`hook.Runner`を組み立てるときに`Rows`を渡していない。したがって`row`の宣言は`runReachable`の分岐で`skipped`になり、理由は「row set が必要」になる。

### 1.3 `command.Result`は行集合を返さない

`internal/command/invoke.go`の`Result`が持つのは`Outcome`・`Effect`・`OutcomeUndetermined`・`AuthorizationMode`・`Applied`・`Persistence`だけである。**行集合（`[]map[core.CanonicalID]string`）を返す欄は無い。** `Request`も`Inputs map[string]string`だけで、行を持たない。

### 1.4 一覧の行を持っている経路は無い

- `applicationview.BuildBrowse`（`internal/applicationview/browse.go`）は、browse Patternの**列・絞り込み・並び順**を宣言から解決する投影で、データ行を返さない（`BrowseView`に行の欄が無い）。
- `cmd/tei/scenario_browse.go`はScenarioのBrowse Fixtureを`[]any`の行として実行・照合するが、これはシナリオ試験の経路であり、`view -command -execute`やStudioの参照実行の経路ではない。Canonicalな行（属性ID→値）への変換もそこで止まる。
- Studioの`view.go`・`command.go`は`inputs map[string]string`しか受け取らない。

**結論: 参照実行の経路には、行集合を供給できるデータ源が無い。** 無い行を作らないという制約の下では、CLI/Studioが自分の実行で行を渡すことはできない。

### 1.5 依頼の前提のうち、コードと食い違った点（重要）

依頼文には「row範囲 + usesの組合せは現状`HOOK_SCOPE_ROW_USES_UNSUPPORTED`で拒否される」とあったが、**このコードは現在のリポジトリに存在しない**。`git grep -n "ROW_USES" -- '*.go'`は0件で、文字列は`plan/evidence/`と`plan/workstreams/`の履歴記録にだけ残っている。

現在のコードでは、行単位の`uses`は1行ごとに果たされる。

- `internal/hook/rows.go`の`RunRows`: `len(declared.Uses) > 0`なら`RunRowCalls(ctx, rowInvocation, declared, rows[index:index+1])`を各行で呼ぶ。
- `internal/hook/reach_rows.go`の`applyReachableRows`: 同じく行ごとに`RunRowCalls`を呼ぶ。

過去の記録（`XTEA_LOWCODE_HOOK_ROW_USES_PARITY_2026-11.md`等）にも「`refuseRowScopeWithUses`は削除し、`HOOK_SCOPE_ROW_USES_UNSUPPORTED`は現在のコードに存在しません」と追記されている。**依頼の「触らない（fail-closedのまま）」は、現状では「触らない（既に1行ごとに実行される）」と読み替えた。** 本作業では`internal/hook`を一切変更していないので、この点の挙動は変わっていない。

## 2. 決めたことと理由

**呼出し側が明示的に行を渡せる入口を足し、渡された行を`hook.Runner.Rows`へ入れる。** 行データをCLIフラグやJSONで受け取る形は採らなかった。

理由:

1. 行集合を供給できるデータ源が無い（1.4）ので、CLIのフラグで行データを受け取る形にすると、**運用者が手で行を書き起こす新しい入力形式**を発明することになる。行はデータ源から来るべきもので、境界の問題（呼出し側が渡していない）を入力形式の問題に置き換えてしまう。
2. 穴は「呼出し側が行集合を供給しない」ことなので、最小の正しい一手は**供給できる入口を用意し、渡されたら実行経路へ流す**ことである。入口が無いままフラグだけ足しても、実行経路の配線は確かめられない。
3. CLIのフラグを足すと、`cli_flag_contract.go`の許可表・`commandCatalog`・契約文書・引数検査のすべてに公開面が増える。行データの形式（複数行をどう表すか）は意味の設計判断であり、この作業の範囲を超える。
4. 既定の経路（`view -command -execute`・Studioの参照実行）は**行を渡さないまま**にしてある。したがって「行が無いのに1行として走る」ことはなく、従来どおり理由つきのskipである。`docs/status.md`もその意味に更新した。

入口の形:

- CLI: `runDeclaredHooksWithRows(ctx, world, applicationID, patternID, inputs, rows, rule)`。既定の`runDeclaredHooks`・`runDeclaredHooksWithRule`・`runDeclaredHooksHalf`は`rows=nil`で同じ本体へ委譲する（既存の呼出し元と試験は無変更）。
- Studio: `runHooksWithRows(world, applicationID, patternID, inputs, rows, rule)`。`runHooks`・`runHooksHalf`は`rows=nil`で同じ本体へ委譲する。
- どちらも本体は`hook.Runner{World: ..., Implementation: rule, Rows: rows}`を組み立てる1行が差分の中心で、位相の選択・読み取りの絞り込み・拒否の扱いは`internal/hook`の既存規則をそのまま使う（CLIとStudioが独自の規則を持たない）。

## 3. 追加した試験

### 3.1 CLIの入口（`cmd/tei/cli_view_hooks_rows_test.go`）

| 試験 | 確かめること |
| --- | --- |
| `TestRunDeclaredHooksWithRowsAppliesTheRowScopeToEveryRow` | 行集合を渡すと`before-update`（`scope row`）が**3行それぞれで1回ずつ**走る。各行の値は行の値（記録側の入力`999`ではない）。宣言していない属性（`sales.order.note`）は行から実装へ渡らない。位相は`executed`に入り、`skipped`・`notEvaluated`に入らない |
| `TestRunDeclaredHooksWithRowsNamesTheRefusedRow` | 2行目が不成立（`0`）のとき、**2行目で止まり**（実装の呼出しは2回）、返るエラーが`*hook.RowRefusal`で`Row == 1`、報告の`refused`が`before-update`、`outcome`が`value-rejected`、理由に`row 1`が残る。拒否した位相は`executed`に入らない |
| `TestRunDeclaredHooksWithoutRowsSkipsTheRowScope` | 行集合を渡さない既定の入口では、フックは**1回も走らず**、`skipped`に理由（`row set`）が付く。「行が無い」を「1行」に読み替えない |

### 3.2 Studioの入口（`internal/studiohost/hooks_rows_test.go`）

同じ3点をStudio側で確かめる。ワークスペースは一時ディレクトリへ`.tei`を書き出して`workspace.Load`で読み込むため、**宣言の解析を通ったWorld**で実行している。`TestRunHooksWithRowsAppliesTheRowScopeToEveryRow`は`validation.Validate`に`Error`が無いことも確かめ、`record` Pattern + `scope row` + `reads`が検証を通る宣言であることを実行の前提にしている。拒否試験は`CommandOutcome`が`value-rejected`（Command側の拒否名）になることまで見る。

### 3.3 試験が配線に依存していることの確認（陰性対照）

`Rows: rows`を両方の本体から外して実行すると、次のとおり失敗した。

```
--- FAIL: TestRunDeclaredHooksWithRowsAppliesTheRowScopeToEveryRow (0.00s)
    cli_view_hooks_rows_test.go:83: the hook was not applied once per row: []
--- FAIL: TestRunDeclaredHooksWithRowsNamesTheRefusedRow (0.00s)
    cli_view_hooks_rows_test.go:111: a non-positive row did not stop the run
FAIL	tei/cmd/tei	0.163s
```

```
--- FAIL: TestRunHooksWithRowsAppliesTheRowScopeToEveryRow (0.02s)
    hooks_rows_test.go:114: the hook was not applied once per row: []
--- FAIL: TestRunHooksWithRowsNamesTheRefusedRow (0.02s)
    hooks_rows_test.go:148: a non-positive row did not stop the run
FAIL	tei/internal/studiohost	1.111s
```

外した行を戻すと両方とも成功した。試験は`Rows`の1行に依存している。

## 4. 実行した検証と結果

| コマンド | 結果 |
| --- | --- |
| `go build ./...`（Go 1.27.0 windows/amd64） | 終了0 |
| `go test ./internal/hook -count=1` | `ok tei/internal/hook 1.159s` |
| `go test ./internal/studiohost -count=1` | `ok tei/internal/studiohost 3.053s` |
| `go test ./cmd/tei -run 'TestRunDeclaredHooks' -count=1` | `ok tei/cmd/tei 0.157s`（11件、すべてPASS） |
| `go test ./cmd/tei -count=1`（パッケージ全体） | **この記録を書いた時点では `ok`（299秒）。後の独立レビューで、この版では再現しないと指摘された**（`TestREAIntakeCompleteGraphDoesNotProveFunctionOrBusiness` が panic。別ブランチの未統合変更が原因で、本変更とは無関係だが、全件PASSとしては裏取りできない）。**全件PASSを根拠にしないこと** |
| `go test ./tools/repository-plan-check -count=1` | `ok tei/tools/repository-plan-check` |
| `go test ./tools/repository-architecture-check -count=1` | `ok tei/tools/repository-architecture-check` |
| `gofmt -l`（**変更したファイルのみ**） | 出力なし（空） |

`internal/hook`の試験を全件流しているのは、`Runner.Rows`の意味（空が「行なし」であること）を変えていないことの確認である。`internal/hook`は無変更なので、既存の行試験（`reach_rows_test.go`・`rows_reachability_test.go`・`rows_uses_parity_test.go`）がそのまま通る。

## 5. 読み取れること

- **呼出し側が行集合を渡せば、CLIとStudioのどちらの入口からも、`scope row`のフックが行ごとに実行される。** 実行は`hook.Runner.Rows` → `applyReachableRows`の既存経路を通る。
- 各行に渡る値は**その行の値で、宣言した`reads`だけ**に絞られる（宣言外の属性は行にあっても実装へ渡らない）。記録側（command inputs）の値は行の判定に混ざらない。
- 拒否した行で止まり、**行番号が呼出し側まで残る**（`*hook.RowRefusal.Row`、報告の`reasons`の`row N`、`outcome`）。拒否した位相は`executed`に入らない。
- 行集合を渡さない既定の経路は、従来どおり「行集合が必要」で`skipped`になり、**フックを1回も走らせない**。入口を足したことで暗黙に1行実行へ変わることはない。
- Studio側の試験は、`workspace.Load`と`validation.Validate`を通した宣言（`record` + `scope row` + `reads`）で実行している。手組みのWorldだけで成立する話ではない。
- 依頼文の前提のうち`HOOK_SCOPE_ROW_USES_UNSUPPORTED`は現在のコードに無く、行単位の`uses`は1行ごとに実行される（1.5）。

## 6. 読み取れないこと（この記録で言えないこと）

- **参照実行（`view -command -execute`・Studioの`ExecuteCommand`）が行フックを実行するようになった、とは言えない。** それらの経路は行集合を渡さないままで、`row`の宣言は理由つきでskipされる。入口はできたが、供給元が無い。
- **例のWorkspace（`reference/workspace`）で`scope row`を宣言したときの実際の参照実行の応答は確かめていない。** `.tei`は編集していない（Gate B/Cで固定された版のため）。行集合が無いときの応答がskipになることは、CLI・Studioの入口の試験（3.1・3.2の3件目）で確かめた範囲であり、実Workspaceの宣言での確認ではない。
- 実データ源・DB・一覧APIから行を取る経路は無いままで、**行の型・列の対応・ページングをどうフックへ渡すかは未決定**である。今回の入口は`[]map[core.CanonicalID]string`という既存の行の型をそのまま受ける。
- `scope row`と`after-*`の組合せは、行集合を渡す入口でも**位相の到達判定より先に**行の適用可否が判定される（`internal/hook/reach.go`の順序）。参照実行（書込みなし）で`after-*`の行フックに行を渡すと、変更が無い時点で走りうる。今回の入口は`ReferenceReach`固定なので、適用後の`after-*`に行を渡す入口（`AppliedChangeReach`と組む形）は用意していない。この順序は既存の`internal/hook`の挙動で、本作業では変更していない。
- CLIフラグ・SchemaVersion・診断コードは追加していない。したがって`tei commands`・契約文書・JSON契約に新しい公開面は無い。
- 実運用のホスト（一覧を返すCLI/Studioの画面）からの呼出し、認可、監査記録は未確認。

## 7. 残る穴と次の一手（候補）

1. **行の供給元を作る**: browse結果を持つ経路（ScenarioのBrowse実行や、将来の一覧API）から、Canonicalな行（`[]map[core.CanonicalID]string`）を取り出して`runDeclaredHooksWithRows`／`runHooksWithRows`へ渡す。どの列を行の属性へ対応づけるかはQuery結果列の対応（`query.EmbeddedSQL.ResultColumns`）が既に宣言しているので、そこから導くのが筋。
2. 参照コマンドが一覧の行を返す設計にするなら、`command.Result`へ行を足すのではなく、**一覧の投影とCommandの実行を別の境界として呼出し側で結合する**方が、Commandの意味（結末と効果）を汚さない。
3. `scope row` + `after-*`の到達判定の順序を揃えるかは、意味の設計判断（行のフックを「変更後」に限るか）として別に扱う。

## 8. 触っていないもの

- `internal/hook`の実装（行単位実行・`uses`の行ごとの扱い・`RowRefusal`・到達範囲）。挙動は変更していない。
- `reference/workspace`配下の`.tei`（Gate B/Cで固定された版）。
- CLIのフラグ・`command.Result`・SchemaVersion・診断コード・公開文字列定数（新規追加なし）。
- 他のworktree（`tei-r102`・`tei-xa`・`tei-xc`・`tei-platform`）。
## 9. 独立レビューによる訂正（2026-11-17）

統合後、独立レビュー担当が本記録を検証し、**記載が事実と違う点**を指摘した。訂正する。

- **`gofmt -l` の対象**: 本記録は「`cmd/tei`・`internal/studiohost`・`internal/hook` 全体が空」と書いたが、統合時点では同スコープに未整形のファイルが4件あった（`cli_contract.go`・`cli_flag_contract.go`・`uncertain_effect_test.go`・`cli_self_build_generated_rule_go_test.go`）。**私が変更したファイルは整形済み**で、ゲートは通る。全体が空という主張は誤りだった。
- **`go test ./cmd/tei` の全件PASS**: 統合後に再現しない。`TestREAIntakeCompleteGraphDoesNotProveFunctionOrBusiness` が panic する（別ブランチの未統合変更が原因）。**この版の全件PASSを根拠にしないこと。**

レビューは他に次を指摘した。いずれも**本変更の欠陥ではなく、既知の範囲として記録済み**のもの、または入口を配線する時点で扱うべきもの。

- 入口は `hook.ReferenceReach` 固定で、行と `AppliedChangeReach` を組む口が無い。**行を渡すと `after-*` の行フックが書込みなしでも実行される**（レビュー担当が実測）。本番呼出し元はゼロなので現時点で実害は無いが、**入口を配線した時点で blocking へ昇格する**。
- 行が非空で、ある行が宣言 `reads` の一部を持たない場合、`HOOK_READ_NOT_OBSERVED` で拒否される（skip/notEvaluated ではない）。この非対称は意図の確認が要る。
- 入口は**本番呼出し元がゼロ**で、参照するのは試験だけ。**供給源が無いまま入口だけがある状態は D-9 と同型**である。

## 10. 供給源の設計候補（独立担当の結論）

独立担当は、穴は「行を返す経路が無い」のではなく**「フック境界の呼出し元に繋がっていない」**と結論した。driver 層には行を返す経路が既にある（`query_execution.go`・`contract_query_execution.go`）が、executor は fixture 読みの合成実装である。構造上の制約は2つ。

1. `interaction` を持てるのは **record Pattern だけ**なので、browse + row は宣言できず到達しない。
2. record Pattern は `relation` も `query` も拒否され、`Hook` の宣言欄に**行の出所を書く場所が無い**。

既定案は **「フック境界専用の行供給インタフェースを定義し、Host が実装を差し込む」**。ただし port の追加だけでは不足で、(i) 行スコープのフックが行の出所を名指す宣言欄（または browse への interaction 許可の判断）と、(ii) NULL と未取得を潰さない行の型（現行は `[]map[CanonicalID]string` で、`narrowTo` が欠落属性を落とすだけなので両者を区別できない）を前提条件とする。

**skip を続ける現状の評価**: fail-closed として正しく、1行として走らず理由も印字されるため D-9 の完全な無信号ではない。ただし `record` + `scope row` + `reads` は validate を通り、**供給経路が無いことを宣言時にも CI でも言わない**ため、受理された宣言が全 production 経路で一度も走らない状態を作れる。よって skip は維持してよいが、**「供給経路の無い行スコープ宣言」を宣言時または CI で見えるようにするまでを穴の完了条件に含める**。
