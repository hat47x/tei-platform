# 効果規則の宣言・検証と仮想項目の保存境界（設計候補）

- 状態: **提案中（未採用）**
- 対応課題: [#1129](https://github.com/hat47x/tei-platform/issues/1129)（Table lifecycle・仮想項目・副作用規則）
- 実行時の正本: [業務副作用の段階・確定後効果・再試行の設計](BUSINESS_EFFECT_LIFECYCLE_DESIGN_2026-10.md)（以下「副作用の設計」）。段階・確定境界・効果記録・補償の規則はこの文書が定める。本稿はそこへ**意味の層から宣言する手段**を足す。
- 前提: [操作面の導出](OPERATION_SURFACE_DESIGN_2026-11.md)
- 関連: #1130（一括）、#1637（D1、導出規則の意味契約）、#1640（D2）、#1692（D4）、#1694（D6）、#1707（D10）、`reference/table-lifecycle/`、`reference/effect-fence/`
- この文書の役割: 現在の `capability ... effect create <data>` だけでは表せない「いつ・どの段階で・どの条件で・誰の所有する記録に」効果を及ぼすかを、宣言と検証の契約にする。

## 1. 現在できていること・いないこと

できていること。

- `capability` は、一つの効果を持つ。`effect create sales_core.attempt` のように宣言し、[`internal/validation/capability_effect.go`](../../internal/validation/capability_effect.go) が、効果の種別と対象データの実在を検査する。
- 実行時の段階分け、確定境界、確定後効果、補償、一括処理の再開は、[副作用の設計](BUSINESS_EFFECT_LIFECYCLE_DESIGN_2026-10.md)が規則として定めている。
- 段階的な書込みの合成参照実装が [`reference/table-lifecycle/`](../../reference/table-lifecycle/README.md) にある（未採用・合成）。

できていないこと。

- 一つの能力が複数の効果を持つことを宣言できない（採番、履歴、在庫の引当て、通知の記録は、どれも同じ確定の中で起きる）。
- 効果の**段階**（書込み前・同一トランザクション内・確定後）を宣言できない。副作用の設計1章の段階は、実行時の規則として存在するが、宣言から導けない。
- 効果の**条件**を宣言できない。`status = confirmed` のときだけ在庫を引き当てる、といった条件が、実装（Goの関数）の中にしかない。
- **所有者の境界**を検査できない。副作用の設計1章は「他の所有者が持つ記録は所有者の業務操作を呼ぶ。所有者を経由しない直接更新は診断の対象とする」と定めるが、その診断は未実装である。
- **仮想項目**を保存するかどうかを宣言できない。導出値（D1〜D15）は純粋計算として扱われ、保存値との区別は #1707（D10）が表示の境界として扱うが、意味の層で「保存する」と宣言する手段がない。

## 2. 宣言の候補

次の二つを足す。どちらも**新しい中核要素ではなく**、`capability` と `data` の既存の宣言を補う任意の宣言として提案する。

### 2.1 効果規則（effect-rule）

```tei
effect-rule sales.order.submit.reserve_stock {
    on capability sales.order.submit
    stage with-write
    when sales.order.status equals "confirmed"
    effect update inventory.inventory_item
    via capability inventory.reserve
    on-failure abort
}

effect-rule sales.order.submit.notify_warehouse {
    on capability sales.order.submit
    stage after-commit
    effect create sales.notification
    on-failure record
}
```

| 要素 | 必須 | 意味 |
| --- | --- | --- |
| `on capability` | 必須 | どの能力に付く効果か。一つの能力に複数の規則を許す |
| `stage` | 必須 | `before-write`（書込み前の補正）、`with-write`（同一トランザクション内）、`after-commit`（確定後の外部効果） |
| `when` | 任意 | 効果の条件。入力または読取り済みの現在状態だけを参照する純粋な述語。外部への入出力を含めない |
| `effect` | 必須 | `create`／`update`／`delete` と対象データ。既存の `capability` の `effect` と同じ語彙を使う |
| `via capability` | 条件付き必須 | 対象データを保存するアプリケーションが、効果を起こす操作のアプリケーションと異なる場合に必須。省略した場合は診断で拒否する |
| `on-failure` | 任意 | `abort`（全体を取り消す。既定）／`record`（`after-commit` の段階だけで許す） |

宣言しないこと。

- 実行順序は書かない。順序は段階と、同一段階内では宣言の順で決まり、`order` のような数値で指定させない（数値の順序は並行変更で壊れる）。
- 接続先・資格情報・SQLは書かない。それらは実行先と提供元の責務である。
- 効果の再試行方針は書かない（副作用の設計11章が固定しないとしている）。

### 2.2 仮想項目の保存状態

導出規則の宣言形式は #1637 が定める（[データ導出規則の最小意味契約](DATA_DERIVATION_REFERENCE_CONTRACT.md)）。そこでは「規則定義」と「適用の結び付け」を分け、**規則が保存値を暗黙に更新しない**ことが定められている。本稿はその原則を変えず、適用先ごとに持つ**保存状態**だけを足す。

| 値 | 意味 |
| --- | --- |
| `never` | 毎回計算する。保存しない（既定） |
| `with-rule` | 効果規則でのみ保存できる。保存する規則が無ければ診断で拒否する |
| `always` | 常に保存する（導出結果を保存値として確定する）。同じ入力でも保存時点の値が残ることを明示する |

`always` は履歴を持つことと同じ意味なので、既定にはしない。`preview`（一時表示）は #1707（D10）の表示境界が持ち、意味の保存状態ではない。

宣言の構文（属性に付けるのか、適用の結び付けに付けるのか）は #1637 と #1707 の結論に従う。本稿が決めるのは、三つの値の意味と、`with-rule` に対して保存する効果規則が無い場合に受理を拒否することだけである。

## 3. 検証

新しい診断コードを、既存の命名（大文字のスネークケース、対象 Subject 付き）に合わせて追加する。

| コード | 条件 |
| --- | --- |
| `EFFECT_RULE_UNKNOWN_CAPABILITY` | `on capability` の参照先が無い |
| `EFFECT_RULE_UNKNOWN_DATA` | `effect` の対象データが無い |
| `EFFECT_RULE_STAGE_NOT_ALLOWED` | 段階で許されない効果（例: `after-commit` で業務データを update／delete する、`before-write` で他データを更新する） |
| `EFFECT_RULE_MISSING_VIA_CAPABILITY` | 保存するアプリケーションが異なるデータへの効果に `via capability` が無い |
| `EFFECT_RULE_CROSS_OWNER_DIRECT_UPDATE` | `via capability` の能力が、対象データを保存するアプリケーションの能力ではない |
| `EFFECT_RULE_CYCLE` | 効果規則の連鎖が循環する |
| `EFFECT_RULE_NON_PURE_CONDITION` | `when` が入出力や状態変更を含む（静的に判定できる範囲で） |
| `DERIVED_ATTRIBUTE_PERSISTED_WITHOUT_RULE` | 保存状態が `with-rule` の導出値を保存する効果規則が無い |

ここでいう「保存するアプリケーション」は、対象データに `persist` を宣言しているアプリケーションであり、**判断の所有者ではない**。誰が・いつ・何を根拠に決めたかの来歴はSEIが持ち、効果規則は判断の識別子を参照するだけにする（[TAI上流への立ち戻り](../workstreams/TAI_UPSTREAM_ALIGNMENT_2026-11.md)）。`persist` がこの作業領域に無いデータ（外部資産）は所有者が不明なので、`via capability` を必須にする。

`EFFECT_RULE_CROSS_OWNER_DIRECT_UPDATE` は、副作用の設計1章が「診断の対象とする」と書いてから未実装のまま残っている項目である。ここで実装の入口を固定する。

## 4. 取消・補償・冪等

- **取消**は、元の操作を更新するのではなく、`reverses` を持つ別の能力として宣言する。取消の効果規則は、元の効果の逆（在庫の返還、履歴の追加）を、同じ段階の規則として持つ。
- **補償**も同じく別の能力である。副作用の設計6章の規則（自動の補償をしない、補償元が未確定なら何もしない）を変えない。
- **冪等**は、業務キー（能力の入力のうち業務上の識別子）と、実行の範囲・識別子・要求の照合値の組で判定する（副作用の設計3章）。効果規則は、この組に参加する入力項目を `idempotency-key` として指定できる。

```tei
effect-rule ... {
    on capability sales.order.submit
    idempotency-key sales.order.order_id
    ...
}
```

`idempotency-key` を指定しない能力は、冪等の記録を持たない。持たない能力を `unattended` として自動実行の対象にしない（操作面の設計8章）。

## 5. 受入

### 5.1 正常系

| 番号 | 状況 | 合格の条件 |
| --- | --- | --- |
| P-1129-01 | 一つの能力に二つの効果規則（`with-write` と `after-commit`）を宣言する | 操作面と診断に両方が現れ、段階が区別される |
| P-1129-02 | 所有者の異なるデータへの効果を `via capability` 付きで宣言する | 診断が通り、実行時に呼ぶ能力の識別子が決まる |
| P-1129-03 | 仮想項目を `persisted with-rule` とし、保存する効果規則を宣言する | 保存の経路が一つに定まり、`always` との区別が読める |

### 5.2 負例（成功に丸めない）

| 番号 | 状況 | 期待する結果 |
| --- | --- | --- |
| N-1129-01 | `after-commit` で業務データを update する | `EFFECT_RULE_STAGE_NOT_ALLOWED`。確定後に業務データを変えない |
| N-1129-02 | 他者の所有するデータを `via capability` なしで更新する | `EFFECT_RULE_MISSING_VIA_CAPABILITY`。所有者を経由しない更新を受理しない |
| N-1129-03 | 効果規則の連鎖が循環する | `EFFECT_RULE_CYCLE`。受理前に拒否する |
| N-1129-04 | `when` に外部APIの呼出しを書く | `EFFECT_RULE_NON_PURE_CONDITION`。条件は入力と現在状態だけで評価する |
| N-1129-05 | `persisted with-rule` の属性に、保存する規則が無い | `DERIVED_ATTRIBUTE_PERSISTED_WITHOUT_RULE`。暗黙に保存しない |
| N-1129-06 | 未確定の元操作に対して取消を起動する | 何もしない（副作用の設計12章 EF-12）。取消の成功として記録しない |
| N-1129-07 | 同じ業務キーで内容の異なる要求を送る | 確定しなかったこととして拒否し、効果を起こさない（EF-03） |
| N-1129-08 | 確定後効果の配送中の権限失効 | 既定では配送し、`withheld` として記録する（EF-09） |

### 5.3 検証コマンド

```sh
go test ./internal/validation -run 'EffectRule' -count=1
go test ./internal/syntax -run 'EffectRule' -count=1
go run ./cmd/tei validate -workspace reference/effect-rules/workspace
go test -race -count=5 ./reference/table-lifecycle
```

## 6. 作業分解

| # | 作業 | 変更する場所 | 完了条件 | 規模 |
| --- | --- | --- | --- | --- |
| W1 | 構文（`effect-rule` と `derived by`／`persisted`） | `internal/syntax/effect_rule.go`（新規）、`parser.go`、`semantic/model.go` | 正負の構文試験。並び順・重複・未知トークンの拒否 | M |
| W2 | 意味モデルと正規化 | `internal/ir/`、`internal/semantic/` | 同一入力から同一の正規形。順序が宣言順で安定 | M |
| W3 | 所有者の判定 | `internal/validation/effect_rule.go`（新規） | `MISSING_VIA` と `CROSS_OWNER` の正負試験 | M |
| W4 | 段階と純粋性 | 同上 | `STAGE_NOT_ALLOWED` と `NON_PURE_CONDITION` の正負試験 | S |
| W5 | 仮想項目の保存境界 | `internal/validation/`、`internal/semantic/derivationref/`（参照） | `PERSISTED_WITHOUT_RULE` と `always`／`with-rule` の区別 | S |
| W6 | 操作面への反映 | `internal/operations/surface.go` | 操作の効果に段階と所有者が現れ、gap が増えない | S |
| W7 | 受入用の作業領域 | `reference/effect-rules/workspace/`（新規） | N-1129-01〜05 を一か所で再現でき、診断コードが固定される | S |

W1・W2 が前提。W3〜W5 は並行できる。W6 は W3 の後。

## 7. 停止条件

- 効果規則の実行を、副作用の設計の段階規則を通さずに実装した場合は停止する。
- `via capability` を、単なる名前の照合で通し、保存するアプリケーションの一致を確かめない場合は停止する。
- `always` を既定にした場合は停止する。
- 実行順序を数値や宣言順以外で決め始めた場合は停止する。

## 8. 効果と撤回

- 効果: 採番・履歴・在庫・通知が、Goの関数ではなく宣言から読める。所有者越境が受理前に分かる。仮想項目を保存するかどうかが意味の層で決まる。
- 観測: 一つの能力あたりの効果規則の数、`via capability` が必要になった件数、仮想項目のうち保存を選んだ割合、所有者越境の診断で止まった件数。
- 撤回: 参照作業領域で効果規則が二件以下にとどまり、`via capability` が一度も必要にならない場合は、規則を汎用化せず、`capability.effect` を複数化するだけの案へ戻す。

## 9. 未決（Owner判断）

1. `effect-rule` を `.tei` の新しい宣言にするか、`capability` の入れ子（`capability { effect-rule ... }`）にするか。前者は差分が読みやすく、後者は所有が明確になる。
2. `when` の述語に許す表現の範囲。既存の `validation` と同じ式を使うか、より狭い比較だけにするか。
3. `stage` の三区分で足りるか。副作用の設計1章は実行時の六段階を持つが、宣言としては三つに畳めるかは実証が要る。
4. `always` を残すか。保存値を確定する業務（締め後の確定金額など）は #2144 の期間の状態と重なるため、そちらへ寄せる案がある。
5. 「保存するアプリケーション」を、`persist` の持ち主とするか、データベースの `ownership`（`managed-here`／`external-observed`）とするか。本稿は前者を案にしている。後者なら外部資産の扱いが揃うが、同じデータを複数のアプリケーションが保存する場合に所有が曖昧になる。
## 実装状況（2026-11-03 追記）

P1（宣言・検証）を実装しました。期間の状態（#2144）で確立した型をそのまま適用しています。

**宣言**

```
effect-rule order.stock {
    on capability sales.order.submit
    stage after-commit
    effect update sales.stock
    via capability sales.stock.decrement
    on-failure compensate
}
```

- 項目: `on capability <id>`・`stage <段階>`・`effect <create|update> <data>`・`via capability <id>`・`on-failure <reject|compensate>`
- 段階: `derivation`・`validation`・`pre-commit`・`in-transaction`・`after-commit`・`compensation`
- 識別子は名前空間に属する（`order.stock` は `sales.order.stock` として解決）

**検証**（`internal/validation/effectrule.go`）

| 診断 | 意味 |
| --- | --- |
| `EFFECT_RULE_UNKNOWN_CAPABILITY` | 対象の能力が存在しない |
| `EFFECT_RULE_UNKNOWN_STAGE` | 段階が上記のいずれでもない |
| `EFFECT_RULE_UNKNOWN_DATA` | 効果の対象データが存在しない |
| `EFFECT_RULE_UNKNOWN_VIA` | 委譲先の能力が存在しない |
| `EFFECT_RULE_UNKNOWN_FAILURE` | 失敗時の扱いが `reject`・`compensate` 以外 |
| `EFFECT_RULE_REQUIRES_VIA` | 確定後・補償の効果が、実行する能力を名指していない |

**未着手**: 操作面・成果物への反映（P2）、実行時の扱い（P3）、`when` のような条件の宣言。

**設計上の注意**: 確定後の効果はトランザクションと一緒に巻き戻せないため、`via capability` を必須にしました。これは「失敗したときに戻せない効果を、戻せる効果と同じ顔で扱わない」という判断です。
### P2（操作面・成果物）の実装状況（2026-11-03 追記）

- 操作面: `binding` が効果規則の対象能力と一致する操作に `effectRules`（規則の識別子）を載せる（`internal/operations/surface.go`）。
- 成果物: 機能定義書に `effect-rule <id> on <capability> stage <stage> effect <kind> <data> [via <capability>] [on-failure <扱い>]` の行を出す（`internal/artifact/functiondef.go`）。
- 生成規則の行種別に `effect-rule` を追加。
### P3（実行時の扱い）の着手（2026-11-03 追記）

確定後・補償の効果は、能力の結末（成功コード）では**証人にならない**ため、独立に観測すべき対象として切り出しました（`internal/execution/effects.go` の `EffectsOutsideTransaction`）。

- 対象能力に束縛され、段階が `after-commit` または `compensation` の規則を返す。
- 戻り値は `EffectRequirement{RuleID, Stage}`。呼び出し側はこれを能力の結末と**混ぜずに**扱う。

**未着手**: 効果の実行そのもの（誰が実行し、成否をどこへ記録するか）、観測の記録形式、失敗時の補償の起動。
### 効果の記録（2026-11-03 追記）

確定後・補償の効果を「実行した事実」と「独立に観測した事実」に分けて残すため、記録の器を追加しました（`internal/effectlog`）。

```go
type Record struct {
    RuleID, Stage, Capability string
    Succeeded bool  // 能力の実行が成功した
    Observed  bool  // 独立した読取りが効果を確認した
    Detail    string
}
```

- `File` は JSON Lines（1行1件）で追記する。壊れた行はエラーとして報告し、**その行より前の記録は返す**（黙って捨てない）。
- `Succeeded` と `Observed` を別の欄にしているのは、成功コードを観測の代わりにしないため。
- 記録は受理済みの意味ではないので、意味モデルには置かない（期間の状態と同じ扱い）。

**未着手**: 効果の実行（`via capability` の呼出し）と記録の接続、失敗時の補償の起動、観測の取得。
### 効果の実行（2026-11-03 追記）

`execution.RunEffect` で、確定後・補償の効果を実行し、記録へ残す経路を追加しました。

1. 規則が `via capability` を名指していること、段階が `after-commit` または `compensation` であることを確かめる（違えばエラー）。
2. その能力を呼ぶ。失敗は**記録**（`Succeeded=false`）であり、エラーとしては返さない。
3. 成功した場合、独立した観測（`EffectObserver`）で効果を確かめ、`Observed` を立てる。**成功コードだけでは `Observed` にしない**。
4. 記録の追記に失敗した場合はエラー（記録できない効果を黙って実行しない）。

**未着手**: 観測の実装（DB の読取り、外部システムの照会）、補償の起動条件、再試行。
### 補償の選択（2026-11-03 追記）

失敗した効果に答える補償を選ぶ関数を追加しました（`execution.CompensationsFor`）。

- 同じ能力（`on capability`）で、**同じデータ**を対象にする `compensation` の規則だけを返す。
- 別のデータを対象にする補償は返さない。まだ望まれている作業を巻き戻してしまうため。
- `NeedsCompensation` は `on-failure compensate` のときだけ真。`reject` は補償を起動しない。

**未着手**: 選択した補償の実行（`RunEffect` と組み合わせる）、補償自体の失敗の扱い、再試行の上限。
### 補償の実行（2026-11-03 追記）

`execution.Compensate` で、選んだ補償を実行し記録します。

- `on-failure compensate` のときだけ起動する（`reject` は何もしない）。
- 補償は**宣言順**に実行し、すべての試行を記録する。**補償自体の失敗も記録**（`Succeeded=false`）として残し、消えない。
- 1件が失敗しても残りを試み、最初のエラーを最後に返す（記録の失敗など、実行を止めるべきものだけ）。

**未着手**: 補償の再試行回数、補償の完了を待つ境界、観測の実装。
### 観測の実装（2026-11-03 追記）

独立した観測の最初の実装として、**SQL の読取り**を追加しました（`internal/execution/observe.go` の `SQLEffectObserver`）。

- 問い合わせが返す件数が 1 以上なら `Observed=true`（`detail` に件数）。
- 0 件は「まだ見えていない」であり、成功ではない（`Observed=false`）。
- 読取りの失敗は**エラー**。観測できなかったことと、観測して見えなかったことを区別する。
- 問い合わせが未設定の場合は `Observed=false` と理由を返す（黙って成功にしない）。
- `Queryer` は 1 行の読取りだけを要求するので、**データベース無しでも試験できる**（偽の実装で件数・エラー・未設定を固定）。

**未着手**: 実データベースでの確認（DSN 未設定）、効果の対象を特定する引数の組み立て（`Args`）の実装、再試行。