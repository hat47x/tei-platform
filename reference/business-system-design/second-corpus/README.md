# Second realistic Reference — 受注生産の製造と在庫への受入

`order_entry`とは独立した、二つ目のBusiness Procedure Referenceです。`ISSUE-000050`（Contract ADR readinessのBlocker B）が求める「独立したBusiness needから生じた、TEI-originalの二つ目のHandoff」を、[Lane B Acceptance Contract](../../../plan/LANE_B_SECOND_REALISTIC_REFERENCE_ACCEPTANCE_CONTRACT_2026-09.md)のRRB-P1 / P2 / P3に沿って示します。

## 何を置いているか

| file | 役割 |
| --- | --- |
| `make-to-order-production-run.json` | Procedure 1: 受注生産の一回の製造。指示・物理的な製造・物理的な確認・完了記録を別のStepに分け、確認済みの品を入庫へ引き渡す |
| `finished-goods-stocking.json` | Procedure 2: 引き渡された完成品を倉庫責任の下で受け入れ、到着を確認し、入庫して在庫記録を残す |
| `production-and-stocking-reference-set.json` | 二つのProcedureと、責任の受け渡し（Handoff）一件。`order-entry-procedure-reference-set.json`と同じ形式 |
| `seed-ledger.json` | 出典、source-backed fact、TEI-originalな設計判断、未解決事項、cross-viewの問い、Handoffの根拠、reviewの状態。Reference用のledgerで、Canonical authorityではない |

fixtureは既存のloader（`readReferenceSet` / `resolveReferenceSet`）、正規化Business model、D4導出、standalone Handoff source形式へそのまま通ります。

## どのようなseedか

- **独立した業務ニーズ**（Handoffを取り除いても成り立つ）: 受注生産では、注文された品が完成数量ぶん物理的に存在するかを、それを求めた指示とも、それについて残した記録とも分けて言える必要がある。記録が物理的な現実より先へ進んだ状態で納期を約束すると、それは偽りの約束になる。別の必要として、入庫側は在庫記録を実際に届いたものと一致させ続ける必要がある。
- **範囲**: 製造一回と入庫までに限る。購買、仕掛原価、計画（MRP）、品質管理システムの詳細、ロット追跡、出荷、会計、複数拠点は範囲外（`seed-ledger.json`の`scope`）。
- **fake realizationを作らない**: どのStepもApplication Pattern / Data / Relation / Validation / Capabilityを参照せず、fixtureにApplication文脈もない。realizationは`manual-unresolved` / `manual-external-unresolved` / `observation-boundary`だけで、Observationのidentityは持たない。

## Handoff

```text
make-to-order.production-run / PR-06 （確認済みの品を入庫へ引き渡す）
        │ condition
        ▼
finished-goods.stocking / ST-00 （引き渡された品を受け入れ側の責任で引き受ける）
```

conditionは「引き渡された品が、生産側が確認した数量で受け入れ側の責任に引き受けられ、品と数量への責任が生産から倉庫へ移ること」である。次のいずれでもない。

- 前のStepが終わったこと（Stepの順序・隣接）
- target Procedureのtrigger
- Observationのidentity、または物理的な観測が正であること

Handoffが自然に生じるのは、二つのProcedureが別々のニーズと別の責任context（候補）を持ち、品と数量への責任が受け入れ側に引き取られるまで生産側にあるからである。この責任の移転はStepの並びとは別のBusiness meaningである。

## review状態

RRB-P1の14項目と、`ISSUE-000050`のseed review checklist（1〜8）に対する**作成者による自己review**を行った。**独立したreview（別の生成AI等）は未実施**で、`seed-ledger.json`の`review`に`self-reviewed` / `pending`として残している。

| RRB-P1 | 確認箇所 |
| --- | --- |
| 1 `order_entry`から独立したBusiness need | ledgerの`independentNeed`、`TestSecondRealisticReferenceIsIndependentOfOrderEntry` |
| 2 Handoffを除いても説明できる | `TestSecondRealisticReferenceProcedureBodiesStandAloneWithoutTheHandoff`、ledger test（需要文がHandoffに触れない） |
| 3, 5, 6 外部artifactのコピー・転載・license類推が無い | ledgerの`copiedArtifacts` / `licenseAnalogyMade`、`TestSecondRealisticReferenceDoesNotCopyExternalMaterial` |
| 4 出典の追跡 | ledgerの`sources`（`use: design-pressure-only`） |
| 7 fact / 判断 / 提案 / 未解決の区別 | ledgerの各section、`TestSecondRealisticReferenceLedgerSeparatesFactsDecisionsAndUnresolved` |
| 8, 9 未知を既知へ昇格しない、false completenessを作らない | ledgerの`unresolved`（9件）、trigger / actorが`candidate`であること |
| 10 範囲が限られている | ledgerの`scope` |
| 11 手作業・物理・外部のactualityをApplication結果へ還元すると壊れる圧力 | PR-03 / PR-04 / ST-01、`TestSecondRealisticReferenceKeepsInstructionEntryAndPhysicalCompletionApart` |
| 12 二視点以上のcross-viewの問い | ledgerの`crossViewQuestions` |
| 13 fake Application / Capability / Data identityが無い | `TestSecondRealisticReferenceCarriesNoFakeRealization`、baseline checker |
| 14 Handoffを捏造していない | ledgerの`handoff`（自然に生じる根拠と、導出元ではないものの列挙） |

RRB-P2（source / test level）は、同じloader・正規化model・D4導出・standalone Handoff source形式・baseline checkerへ通し、Handoffが隣接・actor差・正の物理Observationから導かれないこと、stale / 欠落 / 空のendpointがfail closedすること、conditionがBusiness meaningとして無損失に保持されることを確認している。O1は、protected-target roleが反復しない（Application Patternを参照しない）ため適用せず、target-role taxonomyも増やしていない。一方、Data-state / ownership境界のcross-check（RRB-P2項目4）は、Data identityが未解決（U-6）で照合する対象authorityが無いため未確認である。RRB-P3（actual execution）は`tools/run-business-procedure-gate-a.sh`のA7（critical test manifest）とA8（この二つのfixtureのbaseline checker）で確認する。R = `82e4301`での結果と、確認していない点は`plan/issue/000050-second-realistic-handoff-corpus.md`の実施記録に記録した。

## 主張しないこと

- Handoff一般、または製造・在庫業務一般の妥当性（二例あることは一般化の証明ではない）
- 実際のApplication、Data、Runtimeが存在すること（未解決事項の`data-identity` / `application-realization` / `actuality-evidence`）
- full Business Flow / DFDのConformance（H1/R3はhandoffだけの最小形）
- 独立したreviewを経たこと
