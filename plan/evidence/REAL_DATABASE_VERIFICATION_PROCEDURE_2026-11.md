# 実データベースでの確認手順（2026-11）

- 状態: **手順の固定のみ。実データベースでの実行はまだ行っていません。**
- 目的: 合成の保存領域でしか確かめていない「期待状態の比較による更新」を、実際の PostgreSQL で確かめる。
- 関連: [#2483](https://github.com/hat47x/tei-platform/issues/2483)、[検証記録](SURFACE_AND_ARTIFACT_VERIFICATION_2026-11.md)

## 1. 前提

| 前提 | 確かめ方 |
| --- | --- |
| PostgreSQL に接続できる | `TEI_TEST_POSTGRES_DSN='…' tei dbcheck` が終了コード 0 |
| 一時表を作れる権限がある | 同上（`dbcheck` が作成と削除を行う） |
| 作業ツリーが汚れていない | `git status --porcelain` が空 |

`tei dbcheck` が終了コード 2 を返した場合、**DSN が無い**という意味です。確認は何も実行していません。

## 2. 手順

```sh
# 1. 接続と権限
TEI_TEST_POSTGRES_DSN='postgres://…' tei dbcheck

# 2. 期待状態の比較による更新（二つのトランザクション）
TEI_TEST_POSTGRES_DSN='postgres://…' go test ./tools/reference-expected-state-concurrency -count=1 -v

# 3. 結果を下の記録に残す
```

## 3. 何を確かめるか

| 試験 | 確かめること |
| --- | --- |
| `TestExpectedStateStopsTheSecondWriter` | 二つのトランザクションが同じ状態を読み、**同時に**更新を試みる。片方だけが1行を更新し、もう片方は**0行**。争った側の値が残る |
| `TestExpectedStateAllowsTheSecondWriterAfterAReRead` | 現在の状態を読み直せば、第二の書き手は正しく1行を更新できる |

## 4. 何を確かめないか

- **分離水準の変更**（`SERIALIZABLE` の書込み競合）。扱うのは一つの行と `READ COMMITTED` の分離水準です。
- **実際のアプリケーションの実行経路**（副作用・呼出し・Driver）。SQL の意味だけを確かめる参照試験です。
- **期間の状態・実行計画との組み合わせ**。これらはライブラリの試験で確かめており、実DBでは未確認です。
- **性能・耐久性・高可用性**。本番運用の確認ではありません。

## 5. 失敗したときの切り分け

| 症状 | 見るところ |
| --- | --- |
| `connect:` で失敗 | DSN、ネットワーク、認証。`tei dbcheck` で切り分け |
| 一時表の作成で失敗 | 権限（`CREATE` 相当）。読取り専用の接続先では実行できない |
| 試験がスキップされる | `TEI_TEST_POSTGRES_DSN` が空。環境変数の綴りを確認 |
| 期待と違う行数 | 争った側の値が残っているかを確認。`READ COMMITTED` の前提を思い出す |

## 6. 記録の形式

実行したら、次の形で本ファイルに追記します。

```
- 実行日時: YYYY-MM-DDTHH:MM:SS+09:00
- サーバー: PostgreSQL x.y（`tei dbcheck` の表示）
- コマンド: go test ./tools/reference-expected-state-concurrency -count=1 -v
- 結果: PASS / FAIL（試験ごと）
- 気づいたこと: （競合の起き方、時間、ログなど）
```

## 7. 実行記録（2026-11-06）

- 実行環境: WSL2 Ubuntu、Docker 28.3.3、`postgres:16` イメージ。使い捨てコンテナと空のデータベースを使い、終了後に削除した。DSNとパスワードは記録していない。
- サーバー: PostgreSQL 16.14（Debian 16.14-1.pgdg13+1、x86_64-pc-linux-gnu）。
- `tei dbcheck`: 接続成功、一時表の作成成功。
- `go test ./tools/reference-expected-state-concurrency -count=1 -v`:
  - 初回: `TestExpectedStateStopsTheSecondWriter` が FAIL。結果は `1 and 1` で、二人の書き手が両方1行を更新した。
  - 原因: 試験が二人の読み手の完了を待たずに `start` を閉じていた。速い書き手が先に commit した後に、遅い書き手が新しい状態を読んで更新できた。製品のCASではなく、試験の同期不足である。
  - 修正: 各読み手が読み終えたことを `readDone` で通知し、二人とも読み終えてから `start` を閉じる。再実行は2件とも PASS（0.344s）。
- `go test ./tools/reference-ddl-entry -count=1`: PASS（17.801s）。新規DDLの収束、既存表への制約追加、主キーの置換・削除、制約名の変更、パーティション表、生成名、`reference/workspace` の `sales_core` 収束を含む。
- `go test ./tools/reference-query-input-nullability -count=1`: PASS（0.799s）。欠落・明示NULL・空文字・値の区別、SQL前の拒否を含む。
- `go test ./tools/reference-first-web-target -run '^TestOneRevisionOrderEntryE2EPacket$' -count=1 -v`:
  - 健康診断receiptなしの初回（`a17c28e5`）: PASS（25.09s）。実PostgreSQLへのWebプロセスからの書込みと、別OSプロセスからの独立Readを通した。auditは `INCOMPLETE` / `UNQUALIFIED` で、`E2_PROVIDER_GUARANTEE_NOT_ATOMIC`、`PROVISIONAL_ACCEPTANCE_NOT_FORMAL_EVIDENCE`、`TRUSTED_FULL_HEALTH_NOT_RECORDED` を診断した。
  - 健康診断receiptありの再実行（`005bbee7`、cleanなcloneで `HEAD=origin/main`）: PASS（34.12s）。Actions run [42152908986](https://github.com/hat47x/tei-platform/actions/runs/42152908986)（push、completed、success）のreceiptを使い、`TRUSTED_FULL_HEALTH_NOT_RECORDED` は解消した。残る診断は `E2_PROVIDER_GUARANTEE_NOT_ATOMIC` と `PROVISIONAL_ACCEPTANCE_NOT_FORMAL_EVIDENCE` の2件。
  - receipt: [`QUALIFICATION_FULL_HEALTH_005BBEE7_42152908986.json`](QUALIFICATION_FULL_HEALTH_005BBEE7_42152908986.json)、audit: [`REAL_PROVIDER_PROVISIONAL_AUDIT_005BBEE7.json`](REAL_PROVIDER_PROVISIONAL_AUDIT_005BBEE7.json)。
  - 一度目のreceipt付き実行は、cloneの改行差分で作業ツリーがcleanでなかったため `TRUSTED_FULL_HEALTH_NOT_RECORDED` が残った。worktreeをcleanにした再実行で解消した。receipt自体の不一致ではない。
- **暫定オーナー判断（2026-11-06）**: 残る2診断は、実外部Providerの原子性保証と正式なAuthorityが未提供であることに対応する。実物での再検証までの暫定として受け入れ、`productGate=UNQUALIFIED` は維持する。実外部Provider、Host Grant/Authority、本番相当データが揃った時点で、この判断を見直す。
- 未実行: 実外部Provider、Host Grant/Authorityの実運用、本番データ量、実Release状態でのgate、独立Consumer。これらは[#2619の作業票](https://github.com/hat47x/tei-platform/issues/2619#issuecomment-6673122238)の環境条件が揃うまで NOT RUN とする。