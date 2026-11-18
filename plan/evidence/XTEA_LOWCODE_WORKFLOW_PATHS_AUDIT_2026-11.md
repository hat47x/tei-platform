# `paths` 絞り込みで隠れている失敗の全面監査（2026-11-18）

## 目的

前ラウンドの記録 [`XTEA_LOWCODE_POSTGRES_SUITES_PATHS_2026-11.md`](XTEA_LOWCODE_POSTGRES_SUITES_PATHS_2026-11.md)
は、`paths` の絞り込みで隠れていた `-read-pg-dsn` の失敗が、別パッケージを触った
PR で初めて露見したことを記録し、**「他のジョブにも同じ形の潜伏があるか全面調査は
していない」**と残した。この文書はその全面調査の結果と、同じ穴が再び開かないための
機械検査（ラチェット）を記録する。

診断コード・スキーマ版・公開文字列定数の追加は無い（CI 設定と検査コードのみ）。

## 監査の方法

`.github/workflows/*.yml`（97本）を機械的に読み、次を突き合わせた。

1. workflow の `on: <event>: paths:` にあるパス一覧（`&anchor` / `*alias` と
   `[a, b]` 形式を含む）。
2. その workflow が実行する `go test` の**パッケージ引数**（`./<pkg>` と
   `./<pkg>/...`）。行末がバックスラッシュで継続する行は1コマンドとして連結した。
3. 判定: `paths` のいずれかの項目がパッケージ**ディレクトリ配下**を丸ごと覆うか
   （`<pkg>` または `<pkg>/**`）。`<pkg>/<file>.go` は「そのファイル」を覆うだけで、
   パッケージは覆わない。

除外したもの（除外理由も検査コードに書いた）:

- `go test ./...` / `go test .` — モジュール全体の実行であり、パッケージ単位の主張ではない。
- `working-directory:` が別ディレクトリの step — `cmd/tei-studio` と
  `tools/reference-plugin-registry` は独立した go.mod を持つ別ビルド単位で、
  外側の `paths` では判定できない。
- `paths` を持たない workflow（`tei-repository-health-ci.yml` など）— 全 PR で走るため潜伏が無い。

検査は `tools/repository-architecture-check/workflow_paths_test.go` に置いた
（`TestWorkflowsListEveryPackageTheyTest`）。YAML 依存は足さず、必要な部分集合だけを
テキストで読む。既存の `workflow_go_version_test.go` と同じ「glob で workflow を列挙 →
違反を `t.Errorf` で列挙 → allow 一覧は理由必須」の形を踏襲した。

## 結果

### 修正した組（ディレクトリ glob を `paths` へ追加）

| workflow | 追加した glob | 追加理由 | 起動が増えるコスト |
| --- | --- | --- | --- |
| `tei-ra1-browse-preexecutor-tests.yml` | `internal/syntax/**`, `tools/reference-browse-read/**` | このジョブは `./internal/driver ./internal/query ./internal/syntax ./internal/validation` と `./tools/reference-browse-read ./cmd/tei` を実行するのに、`paths` に両パッケージが無かった。**`internal/syntax` の変更でこのジョブは起動しなかった**（高頻度に変更されるパッケージ） | syntax 変更の PR でジョブ1本（Linux、テストは軽い）。早期発見の価値が上回る |
| `tei-multilingual-overlays-ci.yml` | `internal/sourceacceptance/**` | ジョブは `./internal/sourceacceptance` を `go vet` と `go test -race` で実行するが、`paths` は reference 側だけだった | sourceacceptance 変更時に1本。多言語オーバーレイの受入元を触る変更はまさにこのジョブの対象 |
| `rea-external-evidence-intake.yml` | `internal/jsonl/**` | ジョブは `./internal/externalanalysis/ ./internal/jsonl/` を実行するが `paths` に jsonl が無かった | jsonl 変更時に1本（外部依存なしの短いジョブ） |
| `tei-md-parser-unbound-ci.yml` | `internal/workspace/**`, `internal/syntax/**`（5つの個別ファイルから置換） | ジョブは両パッケージを**丸ごと**実行するのに、`paths` は5ファイルのみだった。**同じパッケージ内の他のファイルを壊しても起動しなかった** | 置換なので、ファイル5件→ディレクトリ2件。起動は増えるが、対象は「パーサ/ローダの境界」そのもの |
| `tei-binding-execution-reference-ci.yml` | `internal/deployment/routing/**`（`routing/executor.go` から置換） | ジョブは `./internal/deployment/routing` を**丸ごと**実行するが `paths` は `executor.go` 1ファイルのみ | routing 配下の他ファイル変更で1本。置換なので executor.go の起動は変わらない |
| `tei-project-advice-authority-ci.yml` | `internal/plugin/runtime/**`（`process_host_start_admission*.go` から置換） | ジョブは `./internal/plugin/runtime` を丸ごと実行するが `paths` は1グロブのみ | runtime 配下の他ファイル変更で1本。権限境界のジョブなので対象を広げる方が正しい |

これら6本は**「ジョブが丸ごと実行するパッケージなのに、そのパッケージ自身が paths に無い」**
組である。個別ファイル列挙は、そのファイルを触った時だけ起動するという意味で
「壊れたことは別のパッケージを触るまで見えない」形そのものだった。

### 現状のまま登録した組（allow 一覧、理由付き）

`workflowPathsAllowed` に17組を登録した。共通の理由は**「`-run` で数件に絞った試験で、
パッケージ全体の glob を足すと起動コストが対象に見合わない」**である。代表例:

- `tei-postgres-unrun-suites-ci.yml` の `cmd/tei` / `internal/operations` /
  `tools/reference-first-web-target`。ジョブは service container 付きで30分。
  対象は `-run` で名指しした数件であり、`paths` はその試験ファイルを列挙している。
  パッケージ glob にすると**ほぼ毎 PR で30分のジョブが走る**。
  ただし前ラウンドの教訓（`read_scenario_page_postgres_test.go` が paths に無く、
  `-read-pg-dsn` の失敗が隠れた）を踏まえ、**この3パッケージの paths は
  「その試験が実際に読むファイル」を列挙する形を維持**し、列挙漏れは
  ラチェットが検出できるようにした（下の「残る限界」）。
- `tei-runtime-engine-evidence-replay.yml` の `internal/plugin/runtime`。
  `-run` 1件、35分、`paths` はその試験ファイル1本。
- `tei-transform-historical-head-qualification.yml` の `internal/plugin/transform`。
  このジョブは**固定の過去コミット**（`BROKEN_SHA`/`FIX_SHA`）に対して実行するので、
  チェックアウトのパッケージは対象ではない。
- `tei-workspace-source-root-platform.yml` の `internal/workspace`。
  Windows ランナーで固定のセキュリティ試験群のみ。`paths` は source-open 系の
  ファイルを名指ししている。
- `rea-external-*` / `tei-org-*` の `internal/operations`。`-run` 名指しで、
  `paths` は該当テストファイルを名指ししている。

### 変数パッケージ一覧（読めないもの）

`tei-windows-go-test-measurement.yml` と `tei-windows-go-test-survey.yml` は
PowerShell の `$packages` / `@packages` でパッケージ一覧を渡すため、静的に読めない。
**黙って飛ばさず** `workflowUnresolvedAllowed` に登録し、理由を書いた。新しい
変数一覧が現れたら検査は落ちる（literal 引数にするか、理由付きで登録するかを強制）。

## 検査の自己検証（負例）

`TestWorkflowPathsCheckFailsClosed` が、一時ディレクトリに次を作って検査自体を試す。

- `paths` が `cmd/tei/cli_view.go` だけの workflow が `./cmd/tei` と
  `./internal/driver` を実行 → **2組とも報告**（見落としが起きないこと）。
- `paths` を `cmd/tei/**` と `internal/driver/**` にすると → **0組**（覆えば通ること）。
- `working-directory: cmd/tei-studio` の step → **対象外**（別モジュールを誤判定しないこと）。
- `go test @packages` → **unresolved として報告**（読めないものを緑にしないこと）。

`TestPathCovers` は glob 規則の単体対照で、`internal/x/y.go` が `internal/x` を
覆わないこと（＝前ラウンドの穴の形）を固定する。

allow 一覧は**縮小専用**にした。覆われた組は「もう覆われているので削除せよ」と
報告する。加えて `TestWorkflowPathsAllowlistDoesNotGrow` で件数の上限（17組、
変数一覧は2本）を固定した。新しい穴は「一組直して消す」か「同じ変更で上限と理由を
書き換える」のどちらかを強制する。これは `tools/repository-plan-check` の
`TestTheUncoveredCodeAllowlistDoesNotGrow` と同じ形で、既存のラチェットの
書き方を踏襲した結果である。各組の理由も必須にした。

## 検証

実行したもの（このworktree、Go 1.27.0）:

```
go test ./tools/repository-architecture-check -count=1     # ok
go test ./tools/repository-plan-check -count=1             # ok
```

上記2つは変更後の最終状態で実行した。`go build ./...` / `go vet ./...` /
`gofmt -l` も同じ状態で実行した。結果:

```
$ gofmt -l tools plan cmd internal
（出力なし）

$ go build ./...
（出力なし、exit 0）

$ go vet ./...
（出力なし、exit 0）

$ go test ./tools/repository-architecture-check -count=1
ok  	tei/tools/repository-architecture-check	（約8秒）

$ go test ./tools/repository-plan-check -count=1
ok  	tei/tools/repository-plan-check	（約16秒）
```

`repository-plan-check` は最初の実行で**自分の文書**が
`TestMarkdownHasNoEscapedBackticks` に落ちた（行末の継続を1文字のバックスラッシュで
書いた箇所）。文書側を「行末がバックスラッシュで継続する行」と書き換えて再実行し、
上記の PASS を得た。検査を回避する除外はしていない。

## 残る限界

- 検査は「`paths` がパッケージを覆うか」だけを見る。**覆っていても `-run` が
  実行する試験を名指ししていない**場合（例: `-run '^TestPostgres'` が新規の
  `TestPostgresFoo` を拾うか）は対象外。前ラウンドで `LoadRequiresARuleID` が
  フィルタ外だった形は、`required` 配列の検査が受けている。
- 許可した組（17組）は**穴が残っている**。列挙したファイルだけを触る変更では
  ジョブが起動するが、同じパッケージの他のファイルを壊した場合は今も見えない。
  これはコスト判断であり、`repository-health`（paths 無し・全 PR で `go test ./...`）
  が同じパッケージを実行するため、**コンパイル破壊はそちらで見える**。見えないのは
  「特定の `-run` 対象だけが壊れる」ケースである。
- `paths` の `**` は「ディレクトリ配下すべて」として扱う。GitHub の実際の
  パス照合とは `*`（スラッシュを跨がない）の細部で差があり得るが、判定に使うのは
  ディレクトリ全体の被覆だけなので影響しない。

## 位置付け

CI 設定とテスト整合の記録。製品の受入・本番適格性を示さない。仮承認。
