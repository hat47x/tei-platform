# D11 複数入力の読取り整合性・Snapshot保証・競合時の失敗境界 Reference（#1708）

**状態: 合成Fixtureの読取り専用Reference。分散Transaction、全Provider共通snapshot API、汎用Cache、Event Sourcingを導入しない。**

このパッケージは、一回の導出評価が複数の入力（Database Contract Query／外部API／Data）を読むとき、**各入力が正しいこと**と**同じ時点の集合として整合していること**を区別する。入力鮮度・Cache・確定業務値の区別は D3（#1641）が所有し、ここでは読取り集合の整合保証レベルを扱う。

## 一致条件（Reference上の候補名）

- `same-record`: 同一recordから読んだ値。
- `same-transaction-snapshot`: Providerが保証する同一snapshot。snapshot tokenが一致することだけで判定し、2件のtimestampが近いことでは判定しない。
- `version-aligned`: 各入力の版を検証できる。版の集合同一性の供給は既存Driver/Providerの保証へ委譲する。
- `best-effort`: 同一時点保証なし。強い整合を偽装しない。

## 判定

`Evaluate` は要求と読取り集合を比較し、`guaranteed` / `unsupported` / `non-atomic` を返す。`Confirmed` は要求が集合全体で証明できたときだけ true。

- 証明できる集合は `guaranteed` / `Confirmed`。
- 証拠が無い（例: 共有snapshotを持たないProvider間読取り）場合は `unsupported`。`current` と偽らない。
- 同一Provider内で二つの非空な証拠が食い違う場合は `ErrMixedSnapshot` として拒否し、部分データを返さない。案件が明示的に非原子を許容する場合だけ `non-atomic` として返し、`Confirmed` にしない。
- `best-effort` は常に `non-atomic` 以上を主張しない。
- 読取りエラー（snapshot失効・行更新・部分取得）は `ErrInputUnavailable` とし、NULLや空集合へ変換しない。

`ReReadAfterConflict` は競合後の再読取りで**入力集合全体**を要求し、認可版が変わっていれば `ErrStaleAuthorization` とする。失敗したいくつかの入力だけを更新して前の集合へ混ぜることを表現できない。

## 合成Fixture（同一DBでも発生し得る競合）

在庫・引当済を一つのDBトランザクションで同時更新する。

| Revision | 在庫 | 引当済 | 引当可能 |
| --- | ---: | ---: | ---: |
| R0 | 120 | 100 | 20 |
| R1 | 110 | 90 | 20 |

R0の在庫120とR1の引当済90を混ぜると30となり、どちらの確定Revisionにもない値になる。

| 条件 | 期待 | 試験 |
| --- | --- | --- |
| R0で固定 | confirmed / 20 | `TestD11R0PinnedReadIsConfirmed` |
| R1で固定 | confirmed / 20 | `TestD11R1PinnedReadIsConfirmed` |
| R0/R1混在・同一Snapshot必須 | `ErrMixedSnapshot`。30を確定値として返さない | `TestD11MixedSnapshotIsNotAConfirmedValue` |
| Provider A/Bで強い同時点保証 | `unsupported`（確証なしに成立扱いしない） | `TestD11SameProviderMixRejectedButCrossProviderUnsupported` |
| best-effort許容／非原子受容 | `non-atomic`、`Confirmed` でない | `TestD11BestEffortAndAcceptedNonAtomicAreNotConfirmed` |
| 途中でsnapshot失効 | `ErrInputUnavailable`（NULLや0ではない） | `TestD11LostSnapshotIsUnavailableNotZero` |
| 版不一致・版未取得 | version-alignedは`unsupported` | `TestD11VersionAlignedNeedsVersions` |
| 再試行で認可版が変化／部分再読取り | 拒否 | `TestD11ReReadRequiresFullSetAndCurrentAuthorization` |

## 焦点試験

```sh
gofmt -l reference/derivation-snapshot/*.go
go test -count=1 ./reference/derivation-snapshot
go vet ./reference/derivation-snapshot
```

## 未実装・依存

- 入力鮮度・Cache・確定保存値は #1641、問合せ実行は #1640、固定時計は #1720、説明Projectionは #1696、行・テナント・項目の認可は #1693、Database Contract Queryは #179 が所有する。
- 単純な1行内の `数量×単価` に新しい宣言を強制しない。整合性保証が案件要求になった場面で判定する。
- DBのIsolation Levelの独自実装、分散トランザクション、全Provider共通snapshot API、複数DBの全体同期、汎用Cacheは行わない。
- 合成Fixtureのみ。実DBのsnapshot境界（M2）、外部Providerとの非原子例（M3）は未検証。実際に使用可能なProviderの保証、合成fixture、実DBの検証結果を区別して記録する。

実行の有無、exact SHA、成功／失敗／未検証を Issue #1708 と対象PRへ記録する。
