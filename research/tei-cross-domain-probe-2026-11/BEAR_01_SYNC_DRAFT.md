# BEAR-01 の同期手順草案（保留中）

**親Issue**：BEAR-01（VSクマ [#95](https://github.com/hat47x/vs-bear-simulator/issues/95) / [#17](https://github.com/hat47x/vs-bear-simulator/issues/17)）。**状態**：草案（2026-11-11）。

## 背景

VSクマの `BearTypedDecision` の列挙・境界値・wire形式・fixtureは、TEIのread-only投影（`research/tei-cross-domain-probe-2026-11/bear/`）で保持しています。ただし、TEIが保持できるのは**形状のみ**で、上流が変わったときにprobeは自動追随しません。

## 草案：3点の同期手順

1. **列挙の一致試験**
   - `bear/contract_test.go` の `upstreamAnchors` を、VSクマの `cognition/bear/*.gd` と `tests/fixtures/bot_scenario_smoke.json` に対して実行する。
   - 追加・変更があれば、anchorを更新する。

2. **境界値・wire形式・fixtureの同期**
   - `docs/BEAR_TYPED_COGNITION_BOUNDARY.md` の検証節に、次の1行を追加する：
     > 列挙の一致試験に加え、境界値・wire形式・fixtureの同期は、`research/tei-cross-domain-probe-2026-11/bear/` のprobeで確認する。専用runnerは作らない。
   - 同期の手順：
     1. VSクマの `typed_decision.gd` / `typed_wire_codec.gd` の値を読む
     2. `bear/contract.go` の定数と比較する
     3. 差分があれば、probeを更新し、anchor試験を追加する

3. **上流変更の追跡**
   - 方法：VSクマのreleaseノートまたはCHANGELOGに、TEI投影の更新を記載する。
   - 頻度：リリースごとに1回。

## 受け入れ条件

- 上記1〜3のうち最低1点が実施され、証拠（文書または計測）が残っている
- その時点で再検討し、受け入れるか・保留を続けるかを決定する

## 非目標

- GDScript生成、ゲームループへの依存、既定providerの変更。
