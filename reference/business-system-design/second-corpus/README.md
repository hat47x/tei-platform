# Second realistic Reference — 受注生産の製造と在庫への受入

`order_entry`とは独立した、二つ目のBusiness Procedure Referenceです。`ISSUE-000050`（Contract ADR readinessのBlocker B）が求める「独立したBusiness needから生じた、TEI-originalの二つ目のHandoff」を、[Lane B Acceptance Contract](../../../plan/LANE_B_SECOND_REALISTIC_REFERENCE_ACCEPTANCE_CONTRACT_2026-09.md)のRRB-P1 / P2 / P3に沿って示します。

## 何を置いているか

| file | 役割 |
| --- | --- |
| `make-to-order-production-run.json` | Procedure 1: 受注生産の一回の製造。指示・物理的な製造・物理的な確認・完了記録を別のStepに分け、物理確認済みの品をproduction controlからterminal releaseして下流のcustodyへ渡せる状態にする |
| `finished-goods-stocking.json` | Procedure 2: 引き渡された完成品を倉庫責任の下で受け入れ、到着を確認し、入庫して在庫記録を残す |
| `production-and-stocking-reference-set.json` | 二つのProcedureと、責任の受け渡し（Handoff）一件。`order-entry-procedure-reference-set.json`と同じ形式 |
| `seed-ledger.json` | 出典、source-backed fact、TEI-originalな設計判断、未解決事項、cross-viewの問い、Handoffの根拠、reviewの状態。Reference用のledgerで、Canonical authorityではない |

fixtureは既存のloader（`readReferenceSet` / `resolveReferenceSet`）、正規化Business model、D4導出、standalone Handoff source形式へそのまま通ります。

## どのようなseedか

- **独立した業務ニーズ**（Handoffを取り除いても成り立つ）: 受注生産では、注文された品が完成数量ぶん物理的に存在するかを、それを求めた指示とも、それについて残した記録とも分けて言える必要がある。記録が物理的な現実より先へ進んだ状態で納期を約束すると、それは偽りの約束になる。物理的完成を確立した後、そのrunを閉じるには確認済みの品をproduction controlから明示的にreleaseして下流custodyへ渡せる状態にする必要があり、これは特定の受入Procedureが無くてもproduction responsibilityの終端として成立する。別の必要として、入庫側は在庫記録を実際に届いたものと一致させ続ける必要がある。
- **範囲**: 製造一回と入庫までに限る。購買、仕掛原価、計画（MRP）、品質管理システムの詳細、ロット追跡、出荷、会計、複数拠点は範囲外（`seed-ledger.json`の`scope`）。
- **fake realizationを作らない**: どのStepもApplication Pattern / Data / Relation / Validation / Capabilityを参照せず、fixtureにApplication文脈もない。realizationは`manual-unresolved` / `manual-external-unresolved` / `observation-boundary`だけで、Observationのidentityは持たない。

## Handoff

```text
make-to-order.production-run / PR-06 （確認済みの品をproduction controlからterminal release）
        │ condition: release済み・warehouse intake可能、received-count reconciliationはpending
        ▼
finished-goods.stocking / ST-00 （warehouse intakeを開始し、released goodsのphysical custodyを取る）
```

conditionは「productionが物理確認済みの品をproduction controlからdownstream custody向けに明示releaseし、そのreleaseによってwarehouse intakeを開始できること。received countのreconciliationはまだpendingであること」である。stocking Procedureのtriggerは「finished goods expected for stocking are awaiting warehouse receipt and physical intake」であり、conditionとは別authorityとして保持する。次のいずれでもない。

- target Step `ST-00`がすでに完了したこと
- received countの照合が成功したこと
- count accountabilityがすでにwarehouseへ移転したこと
- target Procedureのtriggerそのもの
- Observationのidentity、または物理的な観測が正であること

production側のterminal releaseは、特定のHandoff fixtureのためではなくproduction responsibilityを閉じる独立Business Actionとして置く。stocking側はそのreleased goodsに対してwarehouse intakeを開始する。Handoffはこの二つの境界を接続するが、ST-00完了やST-01のcount reconciliation結果を先取りしない。

## review状態

2026-09-24の独立review（OpenAI GPT-5.6 Sol）は旧revision `R = 82e4301`に対して`review-blocked`だった。current sourceはそのF-1/F-2/F-4を受けたremediationであり、reviewer自身が修正をauthorしたため**独立再reviewが必要**である。`seed-ledger.json`の`review`も`remediation-authored-after-independent-review` / `re-review-required`としている。旧review結果をcurrent sourceのacceptanceへ読み替えない。

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
| 14 Handoffを捏造していない | remediationではproduction purpose / TD-1へHandoff非依存のterminal releaseを置き、conditionからST-00完了・count transferを除いた。**独立再review待ちで、達成とはまだ扱わない** |

RRB-P2 item 4（Data-state / ownership境界をaccepted authorityからcross-check）は引き続き未確認であり、current remediationでfake Data identityを追加していない。したがってP2達成は主張しない。旧R = `82e4301`のRRB-P3 actual execution Evidenceはhistorical Evidenceとして保持するが、current remediation source / pressureは別revisionなのでnew Gate A runが必要である。

## 主張しないこと

- Handoff一般、または製造・在庫業務一般の妥当性（二例あることは一般化の証明ではない）
- 実際のApplication、Data、Runtimeが存在すること（未解決事項の`data-identity` / `application-realization` / `actuality-evidence`）
- full Business Flow / DFDのConformance（H1/R3はhandoffだけの最小形）
- 独立したreviewを経たこと
