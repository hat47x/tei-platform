# `postgres-suites` が赤かった理由と、paths を足したことで露見したこと（2026-11-18）

## 対象

- `.github/workflows/tei-postgres-unrun-suites-ci.yml` の `paths`
- `tools/reference-first-web-target/read_scenario_page_postgres_test.go`
- 先行コミット `5d49bfb58`（読取DBの接続文字列を argv から環境変数へ）

## 見つけたこと

行フックの出所（#5460）を統合した直後、`postgres-suites` が **failure** になった。

```
flag provided but not defined: -read-pg-dsn
--- FAIL: TestShippedWebPageShowsRealPostgresRowAnswers
```

**原因は私の変更ではない。** 先行コミット `5d49bfb58` が接続文字列の受け渡しを argv から環境変数へ変え、フラグを `-serve-read-from-db`（真偽値）にした。DSN は `TEI_WEB_READ_PG_DSN` から読む。**しかし試験は旧フラグのままだった。**

## なぜ今まで見えなかったか

`tei-postgres-unrun-suites-ci.yml` は `paths` で起動を絞っている。そのパス一覧に `tools/reference-first-web-target/**` は**無く**、`read_scenario_page_postgres_test.go` を変えたPRでも起動しない。**このジョブは、他のパス（`internal/driver/**` など）を触ったPRでしか走らない。**

私は採番スイートを走らせるために `internal/numbering/**` と `internal/numberingpostgres/**` を paths へ足した（#5449）。**その結果このジョブが起動し、潜んでいた失敗が表に出た。**

つまり:
- `main` は**赤かった**（少なくとも、このジョブが起動した時点では）。
- しかし paths の絞り込みのため、**誰も気づいていなかった**。
- 私が paths を足したことで、**無関係な失敗が自分のPRの失敗として現れた**。

## 変更

- 旧フラグ `-read-pg-dsn` を `-serve-read-from-db` へ。
- `t.Setenv(readDSNEnv, dsn)` でテスト中だけ環境変数を設定（サーバは環境から読む）。
- 旧フラグ前提のコメントを実装に合わせる。

## 観測

- CI（`53dcc34ba`）で `--- PASS: TestShippedWebPageShowsRealPostgresRowAnswers (0.12s)`。`postgres-suites` = **success**。
- `required` の一覧がこの試験名を名指しているため、「走らなかったのに緑」ではない。
- ローカルでは `go test ./tools/reference-first-web-target` 成功（DSN 未設定分は SKIP）。

## 読み取れること

- 出荷バイナリの読取DB契約は**環境変数**であり、argv ではない（#1119 の意図: 接続文字列を引数やログへ出さない）。
- `postgres-suites` の `paths` 絞り込みは、**そのパスを触らない限りジョブを走らせない**。したがって `paths` に無いパッケージの破損は、別のパッケージを触るまで発見されない。

## 読み取れないこと

- **他のジョブにも同じ形の潜伏があるか**。`paths` の絞り込みは各workflowにあり、同様に「触らないと走らない」試験が残っている可能性がある。**全面調査はしていない。**
- `read_scenario_page_postgres_test.go` の `paths` 追加はしていない（追加すれば今後この失敗は早期に出るが、起動コストが増える）。**判断を保留した。**

## 位置付け

CI設定とテストの整合の記録。製品の受入や本番適格性を示さない。仮承認。
