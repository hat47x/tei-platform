# ISSUE-000058: Gate C Visual Fidelityの独立再評価と、第一Web Targetの視覚課題を整理する

- 状態: Proposed
- 提案日: 2026-09-19
- 性格: Evidence confidence / Reference UI quality
- 主担当要求: RQ-010 / RQ-011 / RQ-013 / RQ-021
- 関連: ISSUE-000028 / ISSUE-000007 / ISSUE-000057

## 何に困っているのか

Gate Cは2026-09-19にReadyと判定されました。ただし、Visual Fidelityの評価者は**単一のAI評価者**（`claude-sonnet-5`、方式`vlm-screenshot-review`）で、人による確認は行っていません。契約は評価方式を限定していませんが、Visual Fidelityは主観を含む判定です。同じ評価者が実行者でもあったため、別の視点からの確認が残っていません。

あわせて、評価の過程で見つかった見た目の課題が、非blockingとして記録されただけで、追跡先がありません。

## 減らしたい不確実性

- 別の評価者（人、または独立に指示された別のVLM）が同じScreenshotを見て、同じ判断になるか。
- 見つかった視覚課題のうち、Referenceとして直すべきものはどれか。

## 現在分かっていること

`GATE_C_READINESS_AUDIT.md`に記録した非blockingの見た目の課題です。

- 狭い値列で長いidentifierが途中で折り返される（例: `sales.order.custome` / `r`）
- 標準Projectionの先頭cardの見出しが末尾1文字折り返される
- Custom UIで2列panelと`Order lines` panelの境界線が接する
- Custom UIのScenario state表示が、状態が変わっても段階の強調を変えない。Execution Error時のbadgeが`command`（段階名）になる
- validationの`satisfied` / `not-satisfied`が文字だけで区別され、視覚的な強調差がない（monochrome Referenceの設計）

Runbookの手順にも課題があります。Visual manifestのscaffoldは既定で`viewport-height 1000`を使いますが、それでは標準ProjectionもCustom UIも下部が切れます。Runbookに注記は追加しましたが、既定値は変えていません。Screenshotの撮影自体は手作業です（固定したcapture toolはありません）。

## 決めること

1. 誰が再評価するか（利用者自身、独立に指示した別のVLM、その両方）。
2. 再評価の記録の残し方。保持したScreenshotに対して、別の評価者identityで新しいVisual manifest（別のVisual run）を作り、Visual auditと最終readiness auditを同じ`T`で再実行します。元の評価は書き換えず、追記として残します。
3. 再評価で`fail`または`unresolved`が出た場合の扱い。最終監査は`fail`でfail closedするため、Gate C Readyの記録を更新する必要があります。
4. 視覚課題を直す場合、Web Targetの変更は`T`を変えます。直した`T'`でSemantic / Functional / Visual / Architectureを取り直すか、presentation-onlyの変更として既存の`T`のReadyを維持するかを、明示的に決めます。Evidenceは`T`ごとの主張であり、旧`T`のEvidenceを新`T`へ引き継ぎません。
5. scaffoldの既定viewport高さを変えるか、明示指定を必須にするか。

## 完了条件

- [ ] 再評価者と方式を決める
- [ ] 独立再評価をVisual runとして記録し、audit結果を得る
- [ ] 結果を`GATE_C_READINESS_AUDIT.md`へ追記する（既存の記述は書き換えない）
- [ ] 視覚課題ごとに、修正する / しないを判断し理由を残す
- [ ] 修正した場合の`T`の扱い（取り直し / 維持）を決め、Runbookへ反映する
- [ ] scaffoldの既定viewport高さを見直す
- [ ] 保持Screenshotの保管先が決まっていること（`ISSUE-000057`）

## 非目標

- 特定のWeb framework・design systemの選定
- 完全なPresentation IRの設計
- pixel-perfectなdesignの達成
- Visual Evidenceだけによる、Semantic / Functional Conformanceの合否の上書き

## 関連

- `plan/GATE_C_READINESS_AUDIT.md`
- `plan/GATE_C_FIRST_WEB_TARGET_EVIDENCE_RUNBOOK.md`
- `plan/GATE_C_EVIDENCE_CONTRACT.md`
