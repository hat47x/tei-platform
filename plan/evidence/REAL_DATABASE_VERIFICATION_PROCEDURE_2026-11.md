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