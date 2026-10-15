# D7 導出値の説明可能性・評価根拠・診断の開示境界 Reference（#1696）

**状態: 合成Fixtureの読取り専用Reference。Canonical仕様、受理済みTrace、認可判断、本番値の保証ではない。**

このパッケージは、一つの導出値について「どの受理済み規則・適用先・実装・入力・評価文脈から得られたか」を機械可読な最小の説明へ投影し、利用者ごとに開示を制限する。規則の意味契約は D1 `internal/semantic/derivationref`、確定値は D3 の `ConfirmedFact`、認可判断は #1693 に委譲し、Canonicalの新しい型・Trace・別の正本を作らない。

## 三つを混同しない

- **定義の由来（Design Provenance）**: `RuleOrigin`。受理済みの `RuleID` と `Revision` が意味の同一性であり、`SourcePath` は Source Index 上の位置を示す情報にすぎない。ファイル移動や改名は意味の同一性を変えない。
- **計算の入力根拠（Evaluation Evidence）**: `Evidence`。適用先 `TargetID`、実現方式 `Realization`、実際に使った入力の版と状態、評価文脈クラス、結果状態を関連付ける。
- **結果の業務上の確定（Business Fact）**: `derivationref.ConfirmedFact`。現在評価とは別の事実であり、新しい評価が上書きしない。`Project` は両者を別の欄として投影する。

## 結果状態と診断

`Classify` は D1 の結果か失敗を、粗い `Outcome`（`value` / `null` / `no-value`）と細かい `Reason`（`input-missing` / `absent-relation` / `multiple-relations` / `authorization-denied` / `not-evaluable` / `unknown`）へ分ける。原因不明を `NULL` や `0` に潰さない。

D1 は入力失敗をすべて `ErrInput` で包むため、関係の多重度の理由をエラー文字列から復元しない。原因を既知の入力状態として持つ呼出し側は `ReasonForState` を使う。これにより「参照先なし」と「多重度違反」と「未取得」を文字列解析なしで区別する。

## 説明の同一性

`Evidence.Identity` は、規則ID・規則版・適用先ID・実現方式とその版・入力IDとその版・評価文脈クラスから決定的な digest を作る。Source file のパス、結果の値、評価時刻は含めない。したがって、規則ファイルを移動しても、同じ規則で別の値を再計算しても、説明の同一性は変わらない。規則版・入力版・実現方式版が変われば同一性は変わる。

## 開示の境界

`Project` は `Field` ごとに `DisclosureAuthority` へ問い合わせる。このインタフェースは Host / #1693 が実装する判断点であり、このReferenceはロール・Grant・資格情報を一切扱わない。非 nil のエラーは該当項目だけを fail-closed で伏せ、そのエラー文字列を外部へ出さない。

| Field | 対象 element | 伏せたときの投影 |
| --- | --- | --- |
| `result` | 適用先ID | 結果を見られない利用者には何も返さない（`ErrNotDisclosable`） |
| `rule` / `rule-source` | 規則ID | 規則の同一性 / Source file 位置を別々に隠す |
| `target` | 適用先ID | 適用先を省略 |
| `realization` | 実現方式ID | 実装同一性と版を省略 |
| `input-identity` | 入力ID | その入力の項目ごと省略し、存在も主張しない |
| `input-value` | 入力ID | 入力IDは残し、状態と値だけ省略 |
| `context` | 適用先ID | 評価文脈を省略 |
| `reason` | 適用先ID | 細かい理由だけ省略し、粗い `Outcome` は残す |
| `confirmed` | 適用先ID | 確定値を省略 |

入力は項目単位で開示する。読み取り権限のない入力は値だけでなく項目ごと省略するため、秘匿入力の存在・行数・原価が一般利用者の投影へ漏れない。`VersionKnown` が false の版は空のまま `Unknown` として扱い、`0` などの版を捏造しない。

## 焦点試験

```sh
gofmt -l reference/derivation-explainability/*.go
go test -count=1 ./reference/derivation-explainability
go vet ./reference/derivation-explainability
```

正常例: 数量2・単価300→600 の説明、管理者が規則・適用先・実装・両入力・Source位置・確定値を見る。負例: 一般利用者に原価の値と存在が見えない、確定600と現行700が別事実として残る、未知版を捏造しない、パスを意味IDとして拒否する、入力順序の非正規を拒否する、`reason` を伏せても `Outcome` は残る、結果を開示できないときは部分データを返さず秘密を漏らさない。

## 未実装・依存

- 規則定義と純粋評価は D1 #1637、鮮度・確定値の読み取りは D3 #1641 が所有する。このReferenceは実Operator登録、実Source Index、Canonical trace、CLI/標準UIの操作契約を実装しない（#1165/#1178、#1180へ委譲）。
- 開示判断の実体は #1693 が所有する。ここでの `DisclosureAuthority` は呼出し口であり、テストの許可表は合成Fixtureであって製品Authorityの権限を証明しない。
- 入力の現在版を取得できない Provider では完全再現を主張せず `VersionKnown=false` として `Unknown` を表示する。実Providerからの版取得、原子的Snapshot、実行記録の真正性は未成立。
- SQL Provider での実行、実DB、Wasm、外部業務Runtimeとの同等性、X-TEA #940 の実動作との一致は未検証。合成試験の成功を本番値の保証や実Providerの同等性とは扱わない。

実行の有無、exact SHA、成功／失敗／未検証を Issue #1696 と対象PRへ記録する。
