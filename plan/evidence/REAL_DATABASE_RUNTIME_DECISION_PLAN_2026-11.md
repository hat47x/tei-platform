# 実DBで実行時の判断を確かめる計画（2026-11）

- 状態: **四系統（期間・効果・実行計画・権限）の六つの試験を、一時 PostgreSQL 16.15 で実行し、すべて PASS しました。** 実行の記録は第11章。DSN の無い環境ではスキップします。
- 目的: 期間・効果・権限・実行計画の**実行時の判断**が、実際の PostgreSQL を前にしても期待どおりに働くことを確かめる。
- 関連: [実データベースでの確認手順](REAL_DATABASE_VERIFICATION_PROCEDURE_2026-11.md)、[検証記録](SURFACE_AND_ARTIFACT_VERIFICATION_2026-11.md)

## 1. なぜ別に要るのか

既存の実DB確認は「期待状態の比較による更新」と DDL・NULL・Web E2E を見ています。**期間が閉じているから書かない／効果を観測できないから止まる／付与が無いから実行しない**という判断は、いずれもライブラリの試験でしか確かめていません。

## 2. 確かめること（案）

| 試験 | 手順 | 期待 |
| --- | --- | --- |
| 期間が閉じている | 一時表に行を置き、`PeriodClosed` の状態で `execution.Invoke` を呼ぶ | 呼出しが拒否され、**行が変わらない** |
| 期間が開いている | 同じ行を `PeriodOpen` で呼ぶ | 1行が変わる |
| 権限が無い | 付与を空にして書込み能力を呼ぶ | `blocked` になり、**行が変わらない** |
| 効果を観測できない | 確定後効果を持つ能力を無人で呼ぶ | `blocked` になり、**呼出しに到達しない** |

## 3. 準備

1. `TEI_TEST_POSTGRES_DSN` を用意し、`tei dbcheck` が終了コード 0 を返すことを確かめる。
2. 一時スキーマ（または一時データベース）を作り、試験の最後に削除する。
3. 各試験は**独立した行**を使い、前の試験の影響を受けないようにする。

## 4. 何を確かめないか

- 実運用の権限（Grant/Authority）や外部Provider。
- 性能・同時実行の耐久性（それは `tools/reference-expected-state-concurrency` の担当）。
- 期間の締め・訂正の**業務手順**（意味の層の話であり、DB の話ではない）。

## 5. 実行の記録

実行したら、本ファイルに次を追記します。

```
- 実行日時: YYYY-MM-DDTHH:MM:SS+09:00
- サーバー: PostgreSQL x.y（tei dbcheck の表示）
- 結果: PASS / FAIL（試験ごと、変更行数つき）
- 気づいたこと:
```

## 6. この計画が主張しないこと

- 実行時の判断が実DBで正しいこと（まだ実行していない）。
- 実装が完成していること。
## 7. 実装状況（2026-11-06 追記）

`tools/reference-runtime-decisions/decisions_test.go` に試験を置きました。

- `TEI_TEST_POSTGRES_DSN` が無ければ**スキップ**。あれば一時表（`tei_decisions_probe`）を作って実行します。
- `TestAClosedPeriodLeavesTheRowUntouched`: 期間が閉 → step が `blocked`、**行は `new` のまま**。
- `TestAnOpenPeriodUpdatesTheRow`: 期間が開 → step が `resolved`、**行は `closed` に変わる**。

**確かめていないこと**: 実際の実行（この環境には DSN がありません）。したがって「実DBで期待どおりだった」とはまだ言えません。
## 8. 効果と実行計画の試験（2026-11-06 追記）

同じ雛形（一時表・SKIP・実DB）で、効果と実行計画の判断も足しました。

| 試験 | 期待 |
| --- | --- |
| `TestAnUnobservableEffectBlocksBeforeTheWrite` | 確定後効果を観測できない → step が呼ばれず（calls=0）、**行は `new` のまま** |
| `TestACompletedRunIsRefusedWithoutASecondWrite` | 1回目は実行して行が `closed` に変わる。2回目は `refuse` で **step が呼ばれず**、行も変わらない |

**確かめていないこと**: 実行（DSN 未設定のため SKIP）。三つ（期間・効果・実行計画）とも、DSN のある環境で走らせれば同じ手順で確かめられます。
## 9. 権限の試験（2026-11-06 追記）

四系統目として、権限の判断も足しました。

| 試験 | 期待 |
| --- | --- |
| `TestAMissingGrantBlocksBeforeTheWrite` | 付与なし → step が呼ばれず（calls=0）、**行は `new` のまま** |
| `TestACoveringGrantLetsTheWriteThrough` | `read-write` の付与が `write` の要求を覆う → 実行され、**行が `closed` に変わる** |

これで**期間・効果・実行計画・権限の四系統**が、同じ雛形（一時表・SKIP・実DB）で揃いました。実行は DSN のある環境で行います。
## 10. 実行の経路をCIに置く（2026-11-06 追記）

DSN のある環境が無いまま「実行済み」と書かないために、実行の経路そのものを CI に置きました（`.github/workflows/tei-runtime-decisions-postgres-ci.yml`）。

- `postgres:16-alpine` の一時 service に対して、`tei dbcheck` で接続と一時表作成の権限を先に確かめる。DSN が無ければ `dbcheck` は終了コード 2 で止まる（**実行していない**を成功に数えない）。
- `go test -v ./tools/reference-runtime-decisions` のログを取り、**六つの試験名すべてに `--- PASS:` があること**を機械で確かめる。
- 同じログに `--- SKIP:` があれば失敗にする。DSN の配線が外れた状態を PASS に数えないため。
- 試験ログは 7 日間の診断 Artifact として残す。

**この経路が証明しないこと**: 本番 Provider、実運用の権限、性能、期間の締め・訂正の業務手順。CI の service は使い捨ての PostgreSQL であり、本番の同等性は示しません。

## 11. 実行の記録（2026-11-06 追記）

上の経路を実際に走らせました。受入は PR #3032 の CI job `runtime-decisions-postgres` です。

```
- 実行日時: 2026-11-06T14:32:08Z（job 開始 14:31:11Z）
- サーバー: PostgreSQL 16.15 on x86_64-pc-linux-musl（`tei dbcheck` の表示、一時表の作成: 成功）
- 実行: go test -count=1 -v ./tools/reference-runtime-decisions
- 結果: 4系統6試験すべて PASS、SKIP 0件（機械確認）
- Artifact: tei-runtime-decisions-42234631521（7日）
```

| 試験 | 結果 |
| --- | --- |
| `TestAClosedPeriodLeavesTheRowUntouched` | PASS (0.12s) |
| `TestAnOpenPeriodUpdatesTheRow` | PASS (0.06s) |
| `TestAnUnobservableEffectBlocksBeforeTheWrite` | PASS (0.09s) |
| `TestACompletedRunIsRefusedWithoutASecondWrite` | PASS (0.01s) |
| `TestAMissingGrantBlocksBeforeTheWrite` | PASS (0.01s) |
| `TestACoveringGrantLetsTheWriteThrough` | PASS (0.13s) |

**1回目の実行は失敗しました**（job 127933291451）。`TestACompletedRunIsRefusedWithoutASecondWrite` が `the first run must succeed: results=[{Operation:sales.close Status:blocked Detail:no precondition check is configured}]` で落ちています。**試験の欠陥**であり、実装の欠陥ではありません。無人実行（`Autonomy == "unattended"`）の計画が前提条件の検査関数を渡していなかったため、実行計画の判断ではなく「検査が無い」という別の理由で止まっていました。この試験は「完了した実行を拒否する」ことを見るものなので、`noConditions()`（何も止めない）を明示して渡すよう直しました。DSN の無い環境ではスキップしていたので、**それまでの「試験が通っている」は実行の証拠ではありませんでした**。

**この記録が示すこと**: 四系統の判断が、実際の PostgreSQL 16 の行の状態を変えない／変えることとして観測できた。
**示さないこと**: 本番の Provider・実運用の権限・性能・業務手順。CI の service は使い捨てで、本番と同じ構成ではありません。実装が完成したことでもありません。

