# 行スコープのフックが行の出所を宣言する（B案の実装、2026-11-17）

## 対象

- `internal/application/model.go`（`Hook.Rows`・`HookRowSource`）
- `internal/syntax/application_pattern.go`（`rows from query <id>` の構文）
- `internal/validation/application_pattern.go`（出所の解決・警告の条件）
- `internal/hook/row_source.go`（`RowsForDeclaredSource`・`RowsForDeclaredSourceStrings`）
- `cmd/tei/cli_view_hooks_rowsource.go`・`internal/studiohost/hooks_rowsource.go`（ホストの入口）
- `internal/applicationview/view.go`・`html.go`・`print.go`、`cmd/tei/cli_view.go`（投影と表示）
- 環境: Windows、Go 1.27.0

## 決めた語彙

**`rows from query <query-id>`** を採用する。`rows from relation <relation-id>` は採らない。

理由:

1. **relation の結果をこの境界へ供給する経路が無い。** Relation Query の実行結果も行集合だが、それを `Hook` の宣言へ繋ぐコードは存在しない。宣言できるのに供給できない語を語彙へ入れると、「validate は通るが、どの実行でも skip する」宣言が増える。これは [設計判断](XTEA_LOWCODE_ROW_HOOK_DECLARATION_SITE_2026-11.md) が避けようとした形（D-9）そのものである。
2. **出所の種別を書かせる。** `rows from order_entry.order_list` のような裸の識別子ではなく `query` を挟む。種別を省略できると、後で第2の種別を足したときに、既に書かれた行の意味が黙って変わる。パーサは `query` 以外を拒否する（`rows source must be query; relation sources are not supplied yet`）。
3. **1フックにつき1出所。** 同じフックに `rows` を2回書いた場合は構文で拒否する。最後の宣言が黙って勝つ形を作らない。

## 構文と検証

構文（`internal/syntax/application_pattern.go`、hook のブロック内）:

```
hook before-update {
    scope row
    rows from query order_entry.order_list
    reads sales.order.quantity
}
```

検証（`internal/validation/application_pattern.go`）は、出所が**解決できなければ拒否**する。

| 状態 | 診断 | 種別 |
| --- | --- | --- |
| `scope row` かつ `rows` 無し | `APPLICATION_PATTERN_INTERACTION_HOOK_ROW_SCOPE_HAS_NO_SUPPLIER` | 警告（従来どおり） |
| `rows` の `query` が空 | `APPLICATION_PATTERN_INTERACTION_HOOK_ROW_SOURCE_INVALID` | エラー |
| Query が存在しない | `UNKNOWN_APPLICATION_PATTERN_INTERACTION_HOOK_ROW_SOURCE` | エラー |
| Query が別 Application のもの | `APPLICATION_PATTERN_INTERACTION_HOOK_ROW_SOURCE_APPLICATION_MISMATCH` | エラー |
| `scope` が row でないのに `rows` がある | `APPLICATION_PATTERN_INTERACTION_HOOK_ROW_SOURCE_SCOPE_MISMATCH` | エラー |
| 宣言した `reads` を result mapping が運ばない | `APPLICATION_PATTERN_INTERACTION_HOOK_ROW_SOURCE_READ_MISSING` | エラー |

**警告を出さない条件は「出所が宣言されていること」**（`hook.Scope == "row" && hook.Rows == nil` のときだけ警告）。出所が解決できない場合は警告ではなくエラーで止める。黙って skip される宣言を残さないためである。

`READ_MISSING` は、Query の `result` が明示した attribute の集合と `hooks.Reads` を突き合わせる。行の変換（`RowsFromColumns`）は attribute の無い列を載せないので、mapping に無い attribute を読む宣言は、どの行でも値が無い。出所を宣言したからには、宣言した読み取りにも供給者が要る。

## 供給経路

```
ホスト（CLI / Studio）が保持する Query の結果
  → hook.RowsForDeclaredSourceStrings(world, app, pattern, queryID, columns, values)
      ├ 宣言された出所と、渡された queryID の一致を確認（不一致は HOOK_ROW_SOURCE_MISMATCH）
      ├ 宣言が無い row スコープを拒否（HOOK_ROW_SOURCE_UNDECLARED）
      ├ 行スコープの宣言が無い pattern を拒否（HOOK_ROW_SOURCE_NOT_ROW_SCOPED）
      ├ 既存の RowsFromColumns で列→属性の変換（変換の二重実装をしない）
      └ 各行が宣言した reads を運ぶかを確認（HOOK_ROW_SOURCE_READ_MISSING）
  → hook.Runner.Rows
  → RunReachable / RunRows（行ごとに実行）
```

**層の選択: 入口は `internal/hook` に置き、ホスト側は薄いアダプタにした。**

理由:

1. 変換と宣言照合は**宣言を読む側の責務**である。CLI と Studio の両方に置くと、同じ照合が2実装になり、片方だけが更新される事故（過去に繰り返し起きた形）を招く。
2. `internal/hook` は `driver` を import しないという向きを保てる。ホストが自分の `MappedResult` を `hook.ColumnName` / 値の列へ写して渡す形にしたので、hook 側に driver 依存が入らない。
3. `cmd/tei` に置くと、Studio（別モジュールの `cmd/tei-studio` と `internal/studiohost`）が同じ入口を持てない。`internal/studiohost` に置くと CLI が使えない。共通の宣言照合は `internal/hook` が唯一の置き場所である。
4. ホスト側の関数（`runDeclaredHooksForRowSource`・`RunHooksForRowSource`）は、既存の `runDeclaredHooksWithRows` / `runHooksWithRows` をそのまま呼ぶ。**新しい実行経路を作っていない**ので、位相順序・到達範囲・`on-failure` の扱い・`uses` の呼出しは既存の実装がそのまま効く。

## まだ無いもの（重要）

**実際の Query 実行はまだ無い参照実装なので、行を渡す層までを作った。**

- CLI・Studio のどちらの既定実行も、`view -command -execute` の経路で Query を実行しない。したがって**既定の参照実行はいまも行集合を渡さず、`scope row` は理由つきで skip される**。変わったのは「出所の無い宣言には警告が出る」ことであり、「出所を宣言すれば警告が出ない」ことである。宣言が正しいことは示せるが、**既定経路が勝手に Query を実行するようになったわけではない**。
- `MappedResult` を `RowsForDeclaredSource*` へ繋ぐ呼出し元は、いまは試験だけである（`cmd/tei`・`internal/studiohost` の入口は実在するが、既定の view 実行からは呼ばれていない）。
- **空ページの区別が無い。** `hook.Runner.Rows` は行そのものを持ち、「供給されたか」を持たない。そのため出所を宣言して 0 行を渡すと、runner は「行集合が無い」と読んで位相を skip する。これは「出所を照会したが該当なし」と「呼出し元が行を渡していない」を区別できないという、[値の三状態](XTEA_LOWCODE_ROW_SCOPE_VALUE_STATE_2026-11.md) と同種の制約である。嘘をつかない側（skip と理由）に倒し、試験で固定した（`TestRunDeclaredHooksForRowSourceReportsAnEmptyPageAsSkipped`・Studio の同名試験）。塞ぐには行集合に「供給済み」を運ぶ型が要る。
- `Coverage`（全件か一部か）をフックへ渡す口は無い。一覧は page/filter で本質的に partial である。

## 難所: interaction が record 限定であることの非対称

browse の行を record の row フックへ渡す形になる。この非対称を、**意味として無理がないと判断した**。論拠:

1. **record Pattern は既に「対象の Data」を宣言している。** `data sales.order` があり、行フックの `reads` はその Data の attribute に限られていた（`UNKNOWN_APPLICATION_PATTERN_INTERACTION_HOOK_READ`）。つまり record の行フックは「`sales.order` の行集合のうち、この宣言した読み取りを持つ行」を判定するのであり、**何の行かは既に record が名指している**。足りなかったのは「その行がどこから来るか」だけで、これは出所の宣言で埋まる。
2. **「この query の行ごとに判定する」は record 上で読める。** `pattern order_entry.order_record record { data sales.order; interaction { hook before-update { scope row; rows from query order_entry.order_list; reads sales.order.quantity } } }` は、「`sales.order` の単票に対する更新の前に、`order_entry.order_list` が返す `sales.order` の各行について `quantity` を判定する」と読める。単票の Pattern が一覧の Query を参照するのは、**単票の対象 Data に対する行集合を調達するため**であり、単票の意味（1レコード）を壊していない。壊れるのは「record は query を参照しない」という**これまでの検証規則**であって、Pattern の意味ではない。その規則は `APPLICATION_PATTERN_QUERY_NOT_ALLOWED` として record の `pattern.Query` に残してあり、`Hook.Rows` は別の欄である。
3. **同じ Data かを検証で縛った。** 出所の Query が別 Application のものは拒否する。加えて、宣言した `reads` が Query の result mapping に無い場合は拒否する。したがって「別の Data の行を、この record の行として判定する」形は、少なくとも**宣言した読み取りについては**成立しない。ここを縛らないと、行の型が `map[CanonicalID]string` である以上、属性名が一致すれば別 Data の行でも通ってしまう。
4. **対称でないのは構文の入口だけである。** browse は `interaction` を持てないため、行フックを書く場所が record になる。しかし実行時の適用先は行集合であり、`RunRows` は browse pattern も受け付ける（`TestRunRowsReachesABrowsePattern`）。**宣言の置き場所と適用単位が違う**だけで、意味の二重化（`Items` が単票の項目と一覧の列の両方を指す）は起きていない。これが A 案（`interaction` を browse へ広げる）を避けた理由である。

限界（無理が残る点）:

- **行の同一性を Query の SQL で確認していない。** 「`order_entry.order_list` が返す行は `sales.order` の行である」ことは、result mapping の attribute が `sales.order` の attribute であることでしか示していない。SQL の FROM を解析する経路は無い。したがって、mapping が `sales.order.*` を名乗っていれば、実際に別 table を読む Query でも validate は通る。この穴は**宣言の解決**ではなく**実行の正当性**の問題であり、実 Query 実行を繋ぐ時に一緒に扱うべきものとして残す。
- **browse の page/filter と行集合の関係が未決定。** 出所の Query が page 付きで一部だけ返す場合、行フックは「その page の行」だけを判定する。全件を判定したと読める報告はしていない（`PhaseReport` は行数を主張しない）が、partial であることを型で示す口も無い。

## 試験

`internal/syntax`:

- `TestParseHookRowSource` 成功。`rows from query <id>` が Query 参照として読まれる。
- `TestParseHookRowSourceRefusesARelationSource` 成功。relation の出所は構文で拒否。
- `TestParseHookRowSourceRefusesABareIdentifier` 成功。種別の省略を拒否。
- `TestParseHookRowSourceRefusesADeclarationTwice` 成功。1フック2出所を拒否。

`internal/validation`:

- `TestHookRowSourceIsAcceptedAndFulfilsTheScope` 成功。出所が解決すれば**警告が出ない**。
- `TestHookRowSourceWithoutADeclarationStillWarns` 成功。出所の無い `scope row` は従来どおり警告（種別は Warning）。
- `TestHookRowSourceRefusesAnUnknownQuery`・`...RefusesAnotherApplicationsQuery`・`...RefusesAReadTheMappingDoesNotCarry`・`...RefusesWhenTheScopeIsNotRow`・`...RefusesAnEmptyQuery` 成功。解決できない出所は validate で拒否。

`internal/hook`:

- `TestRowsForDeclaredSourceConvertsTheDeclaredQueryResult` 成功。宣言された Query の結果が行集合になり、行ごとに実行される（10・20）。
- `TestRowsForDeclaredSourceRefusesAnotherQuerysResult`・`...RefusesAnUndeclaredSource`・`...RefusesAPatternWithNoRowScope`・`...RefusesARowWithoutTheDeclaredRead`・`...RefusesUnevenValues` 成功。
- `TestRowsForDeclaredSourceAcceptsAnEmptyPage` 成功。0 行は変換の失敗ではない。

ホスト:

- `TestRunDeclaredHooksForRowSourceAppliesTheHookPerRow`（CLI）・`TestRunHooksForRowSourceAppliesTheDeclaredSource`（Studio）成功。行ごとに実行され、report は `executed`。
- `TestRunDeclaredHooksForRowSourceRefusesAnotherQuerysResult`・`TestRunHooksForRowSourceRefusesAnotherQuerysResult` 成功。別 Query の結果を拒否。
- `TestRunDeclaredHooksForRowSourceReportsAnEmptyPageAsSkipped`・`TestRunHooksForRowSourceReportsAnEmptyPageAsSkipped` 成功。空ページは skip（未解決の制約を固定）。
- Studio の試験では、宣言が **validate を通る**こと（Error が無いこと、`..._HAS_NO_SUPPLIER` が出ないこと）も同時に確認している。

投影と表示:

- `TestBuildCarriesTheHookRowSource`・`TestBuildLeavesTheRowSourceAbsentWhenNoneIsDeclared`・`TestSurfacesCarryTheHookRowSource` 成功。HTML と print の両面に出所が出る。
- `TestWriteViewTextPrintsTheRowSource`・`TestWriteViewTextPrintsNoRowSourceWhenNoneIsDeclared` 成功。`tei view` の text に `rows from query <id>` が出る。

## 実行した検証

- `go build ./...` 成功。
- `go vet ./...` 成功。
- `gofmt -l ./cmd ./internal ./tools ./core` 出力なし。
- `go test ./internal/hook ./internal/validation ./internal/syntax -count=1` ok。
- `go test ./internal/applicationview ./internal/studiohost ./cmd/tei -count=1` ok。
- `go test ./tools/repository-plan-check -count=1`・`go test ./tools/repository-architecture-check -count=1` 成功（下記「ラチェット」）。

## 位置付け

参照実装の試験記録であり、製品の受入や本番適格性を示さない。**宣言と検証・変換・ホスト入口は実装済み、既定の参照実行の配線（Query 実行 → 行供給）は未実装**。設計判断は「保留」から「B案を実装した」へ更新した。仮承認。
