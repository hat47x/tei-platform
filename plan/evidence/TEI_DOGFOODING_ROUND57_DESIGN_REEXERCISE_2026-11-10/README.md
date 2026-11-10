# 語彙改善後に設計演習を再試行し、release 系の可視性欠陥を検出（2026-11-10、第57ラウンド）

- 対象: [#4128](https://github.com/hat47x/tei-platform/issues/4128)（起票）、[#4094](https://github.com/hat47x/tei-platform/issues/4094)（語彙診断の改善）。
- 前段: 第52〜56ラウンドで語彙診断を整備した。本ラウンドは**その改善が実際の設計作業に効くかを測った**。
- 実行日: 2026-11-10（Asia/Tokyo）。
- 状態: **改善の効果を実測で確認し、別の欠陥を1件検出して起票。**

## 1. 目的

第56ラウンドで「次のラウンドは実際の設計作業を再試行し、改善が効いているかを測る」と書いた。そのとおりに実行する。

**今回はソースを読まず、CLI の診断だけを頼りに設計した。** 前回（第51ラウンド）は `grep` で語彙を割り出しており、それでは「CLI 利用主体」の検証になっていなかった。

## 2. 固定した入力

| 要素 | 値 |
| --- | --- |
| Source | `8b57cfaf7`（`origin/main`） |
| CLI実行物 | `sha256:7e9e3be5e1f8c3696c01fc70b9444898384eccd150fd8fe9415bccf2d38384f0` |
| 作業場所 | `reference/workspace` の複製（`/tmp/design58`）。リポジトリは変更しない |
| 環境 | WSL2 Ubuntu 22.04.5、Go `1.27.3 linux/amd64`（gofmt は 1.26.10） |

## 3. 設計課題（着手前に宣言）

> `sales_core` に索引を追加する migration、それを含む database release 5.0.0、それを配備する solution release を追加する。

着手時点で、recovery strategy・migration step kind・data readiness・solution step kind/phase/strategy の**有効値を知らなかった**。

## 4. 設計の経過（CLI の応答のみ）

### Attempt 1（推測で書く）

```text
releases/databases/sales-core/5.0.0.tei:3:5  error PARSE_ERROR
  expected schema-fingerprint, includes, provides, or data-readiness
```

`migrations` ではなく **`includes`** だと分かった。

### Attempt 2

```text
releases/databases/sales-core/5.0.0.tei:3:14  error PARSE_ERROR  expected "migration"
```

`includes migration <id>` の形だと分かった。

### Attempt 3（語彙エラーが一括で返る）

```text
INVALID_MIGRATION_RECOVERY
  migration 20261111_001 has invalid recovery strategy "retry";
  valid values are forward-fix, explicit-rollback, not-reversible

INVALID_DATABASE_RELEASE_DATA_READINESS
  database sales_core release 5.0.0 has invalid data readiness "done";
  valid values are ready, preparing, not-ready, unknown

INVALID_SOLUTION_RELEASE_STEP_KIND
  ... has invalid kind "database-migration";
  valid values are database-release, application-release, application-retire, database-gate

INVALID_SOLUTION_RELEASE_PHASE
  ... has invalid phase "upgrade";
  valid values are expand, populate, transition, rollout, contract, verify, other

INVALID_ROLLOUT_STRATEGY
  ... has invalid rollout strategy "php";
  valid values are direct, canary, blue-green, gradual (or omit the strategy)

ROLLOUT_POLICY_ON_NON_APPLICATION_STEP
  ... has step apply has rollout policy but is database-migration
```

**5つの語彙エラーが1回の実行で同時に返り、それぞれ有効値を名指しした。** 加えて、strategy は application step にのみ意味があるという意味論的な助言も出た。

### Attempt 4（教わった値を適用）

```text
ok
```

**修正3回で設計が通った。ソースは一度も読んでいない。**

## 5. 改善の効果（測定）

| 項目 | 第51ラウンド（改善前） | 本ラウンド（改善後） |
| --- | --- | --- |
| 語彙の入手方法 | `grep` でソースから | **CLI の診断から** |
| 語彙あたりの往復 | 語彙ごとに推測→失敗 | **5語彙を1回で同時に取得** |
| ソース参照 | 必要だった | **不要** |

**語彙診断の改善は、設計作業の往復を実際に減らした。**

## 6. 検出した欠陥: release 系が「モデルを見る」系に出ない

設計は通ったが、**追加した3つの成果物を `model` 系のコマンドで確認できなかった**。

| コマンド | migration | database-release | solution-release |
| --- | --- | --- | --- |
| `validate` | 扱う | 扱う | 扱う |
| `world` | **返す** | **返す** | **返す** |
| `model catalog` | 出ない | 出ない | 出ない |
| `model overview` | 出ない | 出ない | 出ない |
| `inspect` | `UNSUPPORTED_SUBJECT_KIND` | 同左 | 同左 |
| `trace` | `UNSUPPORTED_SUBJECT_KIND` | 同左 | 同左 |

`model overview` は `sectionTotal: 8` を返す（application 2・query 1・semantic 5）。**database・environment・migration・release は1セクションも無い。**

### 6.1 正確に書き分ける

`tei world` は3つの問いすべてに答えた。

```text
release 5.0.0 はどの migration を含むか -> ['20261111_001']
solution release は何をするか           -> apply database-release rollout db=sales_core ver=5.0.0
migration 20261111_001 とは何か          -> recovery: forward-fix, steps: ['schema']
```

**「答えられない」のではない。「`world` を知っている必要がある」。** 私は今回 `world` の JSON を手で解析して確認した。CLI 利用主体の設計作業としては迂回である。

[#4128](https://github.com/hat47x/tei-platform/issues/4128) として起票した。5案を示し、**C（`inspect` の受理種を広げる）+ E（`overview` の対象範囲を明示する）が最小**と書いた。

## 7. 読み取れないこと（NOT RUN）

- #4128 の案の採否（`model catalog` の設計思想に関わるため Owner 判断）。
- `trace` に release 系を載せるか（グラフに subject が無いので、まず subject 化が要る）。
- 実DB・実Provider・実Host、HTTP・Desktop、意味の受理・権限・配備。

## 8. 位置付け

**改善の効果を測る、という前ラウンドの宿題を果たした回**である。結果は明確で、5つの語彙を1回の実行で教わり、ソース参照なしで設計が通った。第51ラウンドでは同じ作業に `grep` が必要だった。

同時に、**語彙とは別の層の欠陥**（release 系が `model` 系に出ない）を検出した。診断を良くしても、成果物を確認する経路が無ければ設計は完結しない。
