# 語彙診断の第4弾: migration と release plan の6件（2026-11-10、第56ラウンド）

- 対象: [#4094](https://github.com/hat47x/tei-platform/issues/4094)（第4弾）。
- 前段: [第55ラウンド](../TEI_DOGFOODING_ROUND55_VOCABULARY_PLUGIN_2026-11-10/README.md)。
- 実行日: 2026-11-10（Asia/Tokyo）。
- 状態: **6件を修正。#4094 の語彙系はこれで完了。**

## 1. 目的

#4094 の残り8件のうち、`migration` と `releaseplan`/`release` の6件を処理する。

## 2. 固定した入力

| 要素 | 値 |
| --- | --- |
| Source | `9b3434f90`（`origin/main`） |
| CLI実行物（修正前） | `sha256:3bfef1018e399dd5c20a0e8317e5bbf38c8f0ee6c6b5f6e68b20973a3402409e` |
| CLI実行物（修正後） | `sha256:917967ff6193d4c78b637884dce8fbede81a986f4fabdaad1d881fa82685162d` |
| 環境 | WSL2 Ubuntu 22.04.5、Go `1.27.3 linux/amd64`（gofmt は 1.26.10） |

## 3. 修正した6件

| コード | 語彙 | 値 |
| --- | --- | --- |
| `INVALID_MIGRATION_RECOVERY` | recovery strategy | forward-fix, explicit-rollback, not-reversible |
| `INVALID_MIGRATION_STEP_KIND` | migration step kind | schema, data, verify |
| `INVALID_DATABASE_RELEASE_DATA_READINESS` | data readiness | ready, preparing, not-ready, unknown |
| `INVALID_SOLUTION_RELEASE_STEP_KIND` | solution step kind | database-release, application-release, application-retire, database-gate |
| `INVALID_SOLUTION_RELEASE_PHASE` | phase | expand, populate, transition, rollout, contract, verify, other |
| `INVALID_ROLLOUT_STRATEGY` | rollout strategy | direct, canary, blue-green, gradual（**または省略**） |

単一情報源として `migration.RecoveryStrategies`・`migration.StepKinds`・`release.DataReadinesses`・`releaseplan.Phases`・`releaseplan.RolloutStrategies`・`releaseplan.StepKinds` を追加した。

`RolloutStrategy.Valid()` は**空文字も受理する**（「宣言していない」の意味）。そこで一覧には空を含めず、診断文に `(or omit the strategy)` と書いた（第55ラウンドの sandbox と同じ扱い）。

## 4. 到達性の実測

| 診断 | CLI から到達 |
| --- | --- |
| `INVALID_MIGRATION_RECOVERY` | する |
| **`INVALID_MIGRATION_STEP_KIND`** | **しない** |
| `INVALID_DATABASE_RELEASE_DATA_READINESS` | する |
| `INVALID_SOLUTION_RELEASE_STEP_KIND` | する |
| `INVALID_SOLUTION_RELEASE_PHASE` | する |
| `INVALID_ROLLOUT_STRATEGY` | する |

`INVALID_MIGRATION_STEP_KIND` が到達しない理由を実測した。migration の step は `schema`／`data`／`verify` のブロックとして書くため、未知の語を書くと**構文エラーが先に出る**。

```text
migration probe_step3 { ... gibberish sql postgres { ... } }
  -> PARSE_ERROR expected database, parent, recovery, schema, data, or verify
```

つまり著者が不正な step kind を書く経路が無い。診断文は正しいが CLI では観測できない（第54ラウンドの `INVALID_SEMANTIC_TYPE_KIND` と同じ性質）。

## 5. 修正後の実測（CLI で確認）

```text
migration recovery:
  migration probe_recovery has invalid recovery strategy "gibberish";
  valid values are forward-fix, explicit-rollback, not-reversible

data readiness:
  database sales_core release 9.9.9 has invalid data readiness "gibberish";
  valid values are ready, preparing, not-ready, unknown

solution release step kind:
  solution release probe_sol@1.0.0 step one has invalid kind "gibberish";
  valid values are database-release, application-release, application-retire, database-gate

solution release phase:
  solution release probe_sol2@1.0.0 step one has invalid phase "gibberish";
  valid values are expand, populate, transition, rollout, contract, verify, other

rollout strategy:
  solution release probe_sol4@1.0.0 step one has invalid rollout strategy "gibberish";
  valid values are direct, canary, blue-green, gradual (or omit the strategy)
```

正常なワークスペースは `ok` のまま。

## 6. 試験

| ファイル | 試験 |
| --- | --- |
| `internal/database/migration/vocabulary_test.go` | `RecoveryStrategies`・`StepKinds` と `Valid()` の一致 |
| `internal/database/release/vocabulary_test.go` | `DataReadinesses` と `Valid()` の一致 |
| `internal/releaseplan/vocabulary_test.go` | `Phases`・`RolloutStrategies`・`StepKinds` と `Valid()` の一致。空の strategy が受理されるが一覧に載らないこと |
| `internal/validation/migration_release_diagnostic_test.go` | 6つの診断が有効値をすべて名指し、「valid values are」で導入する。rollout strategy は省略可能であることを示す |

### 6.1 変異試験

| # | 変えた箇所 | 結果 |
| --- | --- | --- |
| M1 | phase 診断から一覧を削る | FAIL `solution release phase: message does not name "expand"` |
| M2 | `RolloutStrategies` に空を加える | FAIL（空を選択肢にしないことを固定している試験） |
| M3 | `releaseplan.Phases` から `verify` を落とす | FAIL `Phases has 6 entries but 7 are declared` |

## 7. 回帰確認

| 対象 | 結果 |
| --- | --- |
| `./internal/database/...`（10パッケージ） | ok |
| `./internal/releaseplan ./internal/releasegraph` | ok |
| `./internal/validation ./internal/workspace ./internal/operations` | ok |
| `./cmd/tei` | ok（409.5s） |
| `go vet` / `gofmt -l .` | 指摘なし / 空 |

## 8. #4094 の残り

起票時の17件は 17→13→8→2 となった。残り2件は語彙ではなく構造の検証（`INVALID_PLUGIN_ARTIFACT` 等）で、本 Issue の対象外。

**語彙系は本ラウンドで完了した。** ただし到達不能な2件（`INVALID_SEMANTIC_TYPE_KIND`・`INVALID_MIGRATION_STEP_KIND`）は、診断文は正しいが CLI からは観測できない。

## 9. 読み取れないこと（NOT RUN）

- 到達不能な2件を「作者が踏めない経路のための防御」として残すか、削るか（設計判断）。
- 実DB・実Provider・実Host、HTTP・Desktop、意味の受理・権限・配備。

## 10. 位置付け

**4ラウンド続けた語彙診断の整備を、対象範囲について完了させた回**である。手順は第53ラウンドで確立した3段（単一情報源→診断で列挙→2種の試験）をそのまま適用した。

途中で「到達しない診断」が2件見つかったのは収穫だった。診断文を良くしても、著者がその誤りを書けないなら CLI では観測できない。**改善の効果が及ぶ範囲を実測で確かめた**ことになる。
