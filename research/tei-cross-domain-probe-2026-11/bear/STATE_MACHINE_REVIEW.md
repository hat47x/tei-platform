# SM-01 review記録：回復状態機械のread-only投影

要求ロール：`reviewer`。対象：[状態機械のread-only投影](../../plan/research/TEI_CROSS_DOMAIN_PROBE_CSW_VS_BEAR_2026-11.md) §5.6 と `bear/statemachine.go`。
親Issue：[#4193](https://github.com/hat47x/tei-platform/issues/4193)。review日：2026-11-11。

本記録は、実装者とは別のセッションで、**上流ファイルを直接読んで**投影を再確認した結果です。blocking／non-blocking／question に分類します。**#4193 の結論は変更しません。**

## 1. reviewした版

| 対象 | 版 |
| --- | --- |
| VSクマ（上流） | `2a7e2990bb22da57deafb2aedfecb9a654094048`（`controllers/bear/traversal_recovery.gd`、523行） |
| probe（被review） | `research/cross-domain-probe-20261111` の `f579245d5` 時点 |

## 2. 受入条件の確認

| 受入条件 | 結果 | 根拠 |
| --- | --- | --- |
| 1. 遷移表11件が上流`traversal_recovery.gd`と一致する | **一部不一致（blocking 1件）** | 下記 F-1 |
| 2. 「優先度はソース順」「global遷移が平坦な表で隠れる」「列挙は状態集合ではない」の3点が上流から独立に確認できる | **確認できた** | 下記 §3 |
| 3. Domainが1つしかないことの限界が明記されている | **明記されている** | §5.6 末尾、§9 SM-02 |

## 3. 3点の独立確認

### 3.1 優先度はソース順

`step()` の if/elif 連鎖を読み、各状態の分岐を列挙すると、投影の優先度1..nと一致しました。

- WALL_FOLLOW：`provider_speed <= 0.0` → `not recovery_affordance.is_empty()` → `_wall_follow_clear_progress(...)`（112〜131行）
- ENTER_TRAVERSABLE：`provider_speed <= 0.0` → `not replacement_affordance.is_empty()` → `_healthy_clear_progress(...) and not selected_still_traversable`（133〜165行）
- CLEARING：`provider_speed > 0.0 and not clearing_affordance.is_empty()` → `provider_speed > 0.0 and (_blocking_forward or _followed_boundary_blocking)` → `_healthy_clear_progress(...)`（167〜193行）
- IDLE：`provider_speed >= meaningful_drive_mps and not direct_affordance.is_empty()` → `_stall_age_s >= stall_activation_s`（86〜110行）

**確認**：優先度はコード上の分岐順であり、どこにも宣言された policy はありません。3点のうち1点目は成立します。

### 3.2 global遷移が平坦な表で隠れる

`observation_invalid` の扱いは `step()` の冒頭、状態分岐の**前**にあります（74〜82行）。`_state != IDLE` なら `_finish_episode()`（=IDLEへ）して返します。

**確認**：状態別の行だけを並べた表では、この退出は現れません。global として別行にする投影の判断は妥当です。2点目は成立します。

### 3.3 列挙は状態集合ではない

`motion_feedback.gd` は、`recovery.get("state")` が4状態のいずれでもなければ `"unknown"` を書きます（56・61行）。`typeof(state)` 検査でも `"unknown"` を許容します（108行）。

**確認**：`unknown` は記録上のフォールバックであり、`RecoveryState` enum には存在しません。3点目は成立します。

## 4. 指摘

### F-1（blocking）IDLE→WALL_FOLLOW の guard が上流の stall 検出条件を落としている

投影の `t-idle-2` の guard は `stall_age_s >= stall_activation_s` ですが、上流で `_stall_age_s` を進める条件は `_update_stall_age()`（281〜300行）にあり、次の4条件の**連言**です。

```
provider_speed >= meaningful_drive_mps
and lagged_progress_valid
and lagged_progress_mps < stall_progress_mps
and _blocking_forward(observation)
```

投影は「時間が経った」ことだけを guard として書いており、**何が「詰まり」なのか（前進が閾値未満で、前方が塞がれている）を落としています。** この guard だけを読むと、「5秒止まっていれば詰まり」と読めます。上流はそうではありません。

これは #4193 §5.2（遷移決定性）と §5.4（不変条件）の review で人が読むべき情報が欠けている状態です。修正候補：

- guard を4条件の連言へ直す、または
- guard に `_update_stall_age()` の条件を要約して書き、関数名を併記する。

**anchor 試験では検出できません**（`stall_age_s >= stall_activation_s` は上流の106行に実在するため、存在確認は通ります）。§2.2 の限界がそのまま現れた例です。

### F-2（non-blocking）`_latched_side` が状態間で保持されることを表が示していない

`_latched_side` は `reset()`（43行）と `_finish_episode()`（267行）でしか 0 に戻りません。したがって：

- `blocking_returned`（CLEARING→WALL_FOLLOW）では、**以前に選んだ側が保持されます**（179行の `if _latched_side == 0:` が分岐の根拠）。
- `traversable_returned`（CLEARING→ENTER_TRAVERSABLE）でも保持されます。

投影の遷移表は状態名だけを運ぶため、この「保持される側」は表に出ません。TEIの列挙で状態を書いた場合も同じく落ちます。

**これは投影の誤りではなく、投影が写していないものの不足**です。§5.6 の「具体的な不足」に4点目として追加することを提案します：「状態だけでは、遷移をまたいで保持される付随変数（ここでは `_latched_side`）を表せない」。

### F-3（non-blocking）`_update_stall_age` と `_update_retained_asymmetry` は遷移表の外にある

IDLE では `_wall_follow_clear_progress` に相当する「詰まりの蓄積」が `_update_stall_age()` で行われ、`_update_retained_asymmetry()` が側の候補を更新します。これらは遷移ではなく**状態変数の更新**であり、遷移表では表現されません。

投影が遷移だけを写すのは妥当ですが、表を読む人が「IDLE は何もしない状態」と誤解しないよう、§5.6 に一行あるとよいです。

### F-4（question）`t-trav-2` は自己遷移だが、実際には `_selected_affordance` が変わる

投影の `t-trav-2`（ENTER_TRAVERSABLE→ENTER_TRAVERSABLE、`traversable_retargeted`）は、上流では `_select_affordance(replacement_affordance)` を呼び、`_clear_age_s = 0.0` にします（149〜154行）。つまり**状態名は同じでも内部の選択と経過時間は変わります**。

「自己遷移＝何も起きない」と読むのは誤りです。投影のコメントは「retargets in place」と書いており妥当ですが、TEIの列挙で状態を書く場合、同一状態への遷移が「無変化」と読まれる危険があります。§5.6 の不足として扱うかは判断が必要です。

### F-5（non-blocking）`_command()` の `active` と `side` は写していない

`_command()` は `active`（`_state != IDLE`）と `side`（`_latched_side`）を返します（222〜258行）。投影はこれらを写していません。制御出力の契約としては妥当な取捨ですが、§5.6 に「何を写していないか」の一覧があると、後続が探す手間を減らせます。

## 5. reviewの結論

- **受入条件1は未達**です（F-1）。guard の修正後に再 review が必要です。
- 受入条件2・3は満たしています。
- #4193 へ返せる「具体的な不足」は3点から**4点**へ増えます。追加分は「状態だけでは、遷移をまたいで保持される付随変数を表せない」（F-2）。
- F-1 は、**anchor 試験（文字列の存在確認）では検出できない**種類の誤りです。§2.2・TRACE-01 の限界が実例で裏づけられました。

## 6. 実施した検証

- 上流 `traversal_recovery.gd` の全523行のうち、`step()`・`_update_stall_age()`・`_healthy_clear_progress()`・`_wall_follow_clear_progress()`・`_command()`・`_finish_episode()`・`_enter_clearing()`・`_latched_side` の全出現を直接読んだ。
- `motion_feedback.gd` の `unknown` の3出現（56・61・108行）を読んだ。
- `go test ./research/tei-cross-domain-probe-2026-11/... -count=1` を実行し 36件 PASS を確認（review対象の版）。

## 7. 未実施（NOT RUN）

- 修正の実装（F-1）と、その再 review。
- Godot の起動と実行時遷移の観測（投影の正しさは静的比較のみ）。
- `_ray()`・`_best_recovery_affordance()`・`_blocking_forward()` など観測系ヘルパの意味の review（今回は遷移の骨格に限定）。
- #4193 の結論変更、ADR 起案。
