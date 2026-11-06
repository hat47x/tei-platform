# 実DBで実行時の判断を確かめる計画（2026-11）

- 状態: **試験は実装済み。実行はまだしていません（`NOT RUN`）。** DSN が無い環境ではスキップします。
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