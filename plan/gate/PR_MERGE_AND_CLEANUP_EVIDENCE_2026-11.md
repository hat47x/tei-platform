# PR #4306 / #4310 のマージと後始末の記録

- **実行日**: 2026-11-11
- **対象PR**:
  - [#4306](https://github.com/hat47x/tei-platform/pull/4306) `*.tei.md` ドメイン図表→Build 縦断設計と実装導線 — **MERGED**（merge commit `4c9bb28b1`）
  - [#4310](https://github.com/hat47x/tei-platform/pull/4310) マイグレーション生成・起動適用と P0-A 縦断検証 — **MERGED**（merge commit `9d31f8248`）
- **役割**: 本記録は、オーナー判断を代行してレビュー・修正・マージ・後始末を行った際の判断と、その確認結果を残すものである。統合の判断自体は [ADR-0035](../adr/0035-autonomy-levels-and-verification-packets.md) の A3（機械確認できる範囲）に従っている。

## 1. 代行したオーナー判断

| 判断 | 内容 | 根拠 |
| --- | --- | --- |
| 旧PR #4293/#4295 を close し、`origin/main` から作り直した #4310/#4306 を採用 | 旧ブランチは古いbaseに由来し、未コミットの `internal/failure` 等を含んでCIが壊れていた | `package tei/internal/failure is not in std` をCIログで確認。自分の担当分（`tools/reference-application-binary` と P0-A 証拠）のみを切り出せば自己完結する |
| `plan/design/README.md` の BOM を維持して書き戻す | BOM を落とすと1行目が336行目の重複見出しと一致し `TestMarkdownTitleAppearsOnce` が本ブランチだけでFAILする | main の当該ファイルは BOM 付き。BOM があれば PASS することを確認 |
| 生成moduleの `modernc.org/sqlite` 版をこのツール自身の版に合わせる | 版が不一致だと `go mod tidy` が第二版を引き、module cache 検証で失敗する | v1.34.5（生成側）と v1.62.0（ツール側）の不一致を確認 |
| 生成moduleに専用の `GOMODCACHE` を切る | 継承した共有 cache が書き込み不可の場合、tidy が verifying module で失敗する | `open C:\Users\...\sumdb\sum.golang.org\latest: Access is denied.` を確認 |
| 空コミットでCIを再トリガ | force-push 後、health workflow は `pull_request` の `synchronize` でしか走らず、新SHAのrunが無かった | `pull_request` trigger を workflow 定義で確認 |

## 2. マージ前の検証（両PR）

### #4310（実装）

```
go build ./...                                   exit 0
go vet ./...                                     exit 0
gofmt -l .                                       空

go test -count=1 -timeout 30m -p 1 ./...
  ok  tools/reference-application-binary          1544.807s
  ok  .../internal/teiparse                           1.289s
```

- `TestP0AStageParse` / `Model` / `Generate` / `Build` すべて PASS
- クロスコンパイル（linux/amd64, darwin/amd64, darwin/arm64）PASS
- マージ後も `main` の checkout で P1-B/P0-A の主要テストを再実行し PASS を確認

### #4306（設計・ツール）

```
go run ./tools/reference-tei-md-p1b-check    5/5 PASS
go test -count=1 ./tools/reference-tei-md-p1b-check   ok
go test ./tools/repository-plan-check の link/index 検査   ok
```

## 3. main の CI に対する影響（重要）

マージ前後の `TEI PR・main 健全性確認` を比較した。

| | 失敗テスト数 |
| --- | --- |
| マージ前の main（`40f34f98`） | 27 |
| マージ後の main（`9d31f824`） | 26 |

**新たに壊れたテストは0件**。`TestToolsReadmeMentionsEveryEntry` はマージ前は失敗していた（本PRの未登録ツールを指していた）が、マージ後は PASS した。

残る26件の失敗は、`origin/main` の pristine checkout で同一に再現される main 自体の既存故障である。主なもの:

- `TestDiscoveryAdvertisesEveryModeAResponseCanCarry` / `TestDiscoveryWriteFlagsMatchObservedFileEffects`（`unknown command "rea-evidence"`）
- `TestThreeElementReaIntake*` 9件（`unknown three-element subcommand "rea-intake"`）
- `TestWorkflowsFollowGoModVersion`、`python3 tools/ci-workflows-test.py` 5件
- `TestMarkdownRelativeLinksResolve`、`TestSchemaVersionLiteralsHaveATest`、`TestExportedStringConstantsHaveATestOrAnEntry`、Java artifact 系

したがって本変更は main の健全性を**新たに劣化させていない**。

## 4. 後始末（実施済み）

| 項目 | 結果 |
| --- | --- |
| マージ済みブランチ `feat/reference-application-binary-migrations-20261111` | remote で自動削除済み、local の追跡refも prune 済み |
| 旧 `docs/tei-md-diagram-to-build-20261111` | マージにより remote から削除済み |
| 旧スーパシードブランチ `test/reference-application-binary-20261111` | local から削除（内容は main の564行版に吸収。394行の旧版で、未マージcommitは1件のみ） |
| 自分のworktree（`tei-wt-p1b-20261111`, `tei-wt-4310`, `tei-wt-maincheck2`） | 削除済み |
| local `main` | `origin/main` に ff 済み、作業ツリー clean |
| 旧PR #4293 / #4295 | CLOSED（superseded コメント付き） |

**残したもの（意図的）**: `feat/issue-4218-step-code-adapter` と `fix/issue-4218-step-code-adapter-20261111`。この2つは #4218 の dispatcher 作業を含み、**未マージ**であり、open PR も無い。別Issueの作業なので削除していない。担当者に確認してほしい。

## 5. 成立しなかったもの（変更なし）

- **Execute 実Provider適合は unverified**: workspace migration SQL は PostgreSQL 方言で SQLite 非互換（`near "."` 構文エラー）
- 新 `*.tei.md` Reader 接続は NOT RUN（#3030）
- P2-C（Reader接続・旧新比較）、P3-D（複数Consumer一致）は NOT RUN
- Owner意味受理・実Host Grant は NOT ACCEPTED / NOT RUN
