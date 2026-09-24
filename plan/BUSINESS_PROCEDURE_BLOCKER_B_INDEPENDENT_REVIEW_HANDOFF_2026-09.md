# Business Procedure Blocker B — 独立生成AIレビュー依頼

- 作成日: 2026-09-24
- 主担当Issue: [ISSUE-000048](issue/000048-business-procedure-contract-adr-readiness.md)（Contract ADR readiness） / [ISSUE-000050](issue/000050-second-realistic-handoff-corpus.md)（second realistic Handoff corpus）
- 判定の正本: [Pre-ADR Audit](BUSINESS_PROCEDURE_CANONICAL_CONTRACT_PRE_ADR_AUDIT_2026-09.md) §11 / §13 / §17
- レビュー種別: **作成者とは別の生成AI**による、Pre-ADR Audit再判定前の独立レビュー
- レビュー状態: **未実施**
- Contract ADR: **Not Ready**（このパケットは状態を変えない）

## この文書の役割

Blocker B（independent second realistic TEI-original responsibility transfer）のEvidenceは2026-09-21に取得済みですが、確認は作成者のself-reviewだけです。利用者の方針（2026-09-21、commit `a3fc260b`）により、Pre-ADR AuditによるBlocker Bの再判定は**別の生成AIによる独立レビューの後**に行います。

この文書は、そのレビューを別のAIがRepositoryだけを手がかりに実施できるよう、対象・判定語彙・必須の問い・記録先を固定する依頼パケットです。Second Application S5の[独立レビュー依頼](SECOND_APPLICATION_S5_INDEPENDENT_REVIEW_HANDOFF_2026-09.md)と同じ形に揃えています。

- 何の正本か: Blocker B独立レビューの**入力と出力形式**
- 何の正本ではないか: レビュー結果、Blocker Bの充足判定、Contract ADR readiness（いずれもPre-ADR Auditが正本）
- 非主張範囲: 本文中の「準備者の観察」はレビュー結果ではありません。作成者もこのパケットの準備者も、独立レビュアーの代わりになりません

## 1. 背景と、防ぎたい失敗

Contract ADRは、Business Procedureの最小Canonical contract（Procedure-scoped Step address、handoff-only relation `H1` / `R3`、`D4`、`S4-A`等）を採用する判断です。二つ目のrealisticなHandoffは、「一つ目のReference（`order_entry`）の形に合わせただけのcontractではないか」を反証するために要求されています。

防ぎたい失敗は次の三つです。

1. **同じ作成者・同じprotocolによる確証の自己循環。** 二例目も一例目と同じAIが同じ手順でauthorしました。self-reviewだけで充足扱いにすると、contractの一般性を誰も反証していないことになります
2. **Handoffのための業務の逆算。** Handoff endpointを置くためだけにStepを作る、conditionを受け手Stepの言い換えで満たす、といった形でも、構造testは通り得ます
3. **未確認点の読み替え。** 圧力を掛けていない点（同一actor、condition根拠のevent / document identity等）を「不足が出なかった」へ読み替えると、reopen triggerが働かなくなります

## 2. レビュアーの条件と作業

### レビュアーの条件

- Blocker B Evidenceの作成者・self-reviewerは **Claude Sonnet 5（Anthropic）** と記録されています（commit `175b1918` / `a3fc260b` のCo-Authored-By）
- したがって独立レビュアーは **Anthropic Claude系以外の生成AI** とします。S5の前例（作成者Claude Sonnet 5 → レビュアーGPT-5.6 Sol）に揃えます
- **このパケットの準備者はClaude Opus 5.5であり、同じ理由でレビュアーの資格を持ちません**

### 作業

Blocker B Evidenceを§3の固定identityで読み直し、§6の問いへ答えてください。Pre-ADR Auditの再判定、Contract ADRの起票、fixtureやtestの修正は行いません。

判定は次の三つから一つを返します。

```text
review-clear
review-clear-with-conditions
review-blocked
```

あわせて次を記録します。

- レビュアー / modelの識別とレビュー日
- 指摘ごとのseverity（clear / minor / major / blocking）と根拠（file:line）
- 指摘ごとに「Audit再判定の前に直すべきか」「unresolvedとして保持してよいか」とその理由
- RRB-P1 / P2 / P3それぞれの達成可否と、B-F19b status（[Acceptance Contract §6](LANE_B_SECOND_REALISTIC_REFERENCE_ACCEPTANCE_CONTRACT_2026-09.md)）の推奨値
- §6.4の準備者の観察それぞれへの明示的な判断
- 既存のGo test / runnerを自分の環境で再実行したかどうか（しなかった場合は、記録済みEvidenceとして扱ったと明記）

## 3. レビュー対象の固定identity

### Repository revision

```text
R = 82e430109f8f02944519738045f3d6c0bef4f295
```

Evidenceはこの`R`で取得されています。レビューも`R`の内容を対象にします。

### Business source（`reference/business-system-design/second-corpus/`、`R`時点）

| file | SHA-256 | bytes |
| --- | --- | --- |
| `make-to-order-production-run.json` | `27e9834e88f6f53aefc676e5ddb6eb04737a35231b5c232017ab27a78d302266` | 1973 |
| `finished-goods-stocking.json` | `c10ed6b01440c3ccac6a49e64817466eb15a071654e9d05765e700702649f69f` | 1329 |
| `production-and-stocking-reference-set.json` | `568d353555bae09de6fdae56595cc0438fdfe04aa43fc87d05feaf391aa32c6b` | 812 |
| `seed-ledger.json` | `7ed3a3d24494e282d974e98372c3fc922db747eadcc50ffd5a5a8251ff078f59` | 9740 |
| `README.md` | `10e0f182b7f36fd3e88fd956abc193e911ca0487aea0598911f0ac3bb8744659` | 6578 |

`R`以降の変更は`README.md`の1行だけ（commit `175b1918`）で、RRB-P2項目4が未確認である旨の追記です。fixtureとledgerは`R`から変わっていません。

### 検査コードとrunner（`R`時点）

```text
tools/reference-business-procedure-check/second_realistic_reference_pressure_test.go
  SHA-256 362af57253e8983f5b5e26d9141fb808cd20667bd1d1751e49c56c6de0d79f2a（28648 bytes）
  TestSecondRealisticReference* 13件

tools/run-business-procedure-gate-a.sh
  Step A1〜A8（A7 = critical test manifest、A8 = 二つのfixtureのbaseline checker）
```

### 実行Evidence

```text
GitHub Release: business-procedure-gate-a-evidence-R-82e4301（tagはRを指す）
asset:          tei-lane-c-gate-a-evidence-R-82e4301.tar.gz（6,009 bytes）
SHA-256:        3a433835241b82b564e47cb5eb48e63c24f7a5afd3797ae1090afa9add058d7a
内容:           MANIFEST.sha256で各fileを照合できる
```

## 4. 対象の意味（何がauthorされたか）

二つのProcedureと一件のHandoffです。

```text
make-to-order.production-run（受注生産の製造一回）
  PR-00 要求される品目と数量を決める
  PR-01 必要な材料が物理的に揃うようにする
  PR-02 作業指示を出す
  PR-03 製造そのものを行う
  PR-04 物理的な確認で、完成し受け入れ可能な数量を確定する
  PR-05 確認に基づいて製造完了を記録する
  PR-06 確認済みの品を入庫へ引き渡す
            │
            │ Handoff（explicit source: production-and-stocking-reference-set.json）
            ▼
finished-goods.stocking（完成品の入庫）
  ST-00 引き渡された品を引き受ける
  ST-01 物理的に届いたものを、生産が確認した数量と照合する
  ST-02 受け入れた品を保管場所へ置く
  ST-03 在庫移動と、その結果の在庫状態を記録する
```

Handoffのcondition（原文）:

```text
the released goods, in the quantity production confirmed, have been taken in
charge by the receiving responsibility, so that accountability for the goods and
for their count moves from production to the warehouse
```

trigger / actorは両Procedureとも`candidate`のままです。seed ledgerは未解決事項をU-1〜U-9として持ちます（actor identity、exact trigger、step順序のauthority、責任境界の詳細、condition根拠のdocument / record、Data identity、Application realization、actualityのEvidence、受け入れ基準）。

## 5. 作成者が主張していること / 主張していないこと

作成者の主張と限定は[ISSUE-000050 実施記録](issue/000050-second-realistic-handoff-corpus.md)と[Pre-ADR Audit §17](BUSINESS_PROCEDURE_CANONICAL_CONTRACT_PRE_ADR_AUDIT_2026-09.md)が正本です。要点は次のとおりです。

- RRB-P1: 14項目すべてに確認箇所を対応づけた（self-review）
- RRB-P2: 項目4（Data-state / ownership境界のcross-check）は、Data identityが未解決（U-6）のため**未確認**
- RRB-P3: `R`でA1〜A8がexit 0、A7はcritical 39件すべてPASS（うち13件が`TestSecondRealisticReference*`）、A8でblocking diagnosticなし
- 主張しないこと: 独立レビューを経たこと、Handoff一般の妥当性、実Application / Data / Runtimeの存在、full Business Flow / DFDのConformance、Contract ADR ready

## 6. 必須の問い

### 6.1 RRB-P1の独立評価

[Acceptance Contract §2のRRB-P1](LANE_B_SECOND_REALISTIC_REFERENCE_ACCEPTANCE_CONTRACT_2026-09.md) 14項目を、作成者の対応表（`second-corpus/README.md`）に依存せず評価してください。特に次を重視します。

- 項目2 / 14: Handoffを取り除いてもReferenceが独立した価値を持つか。`TestSecondRealisticReferenceProcedureBodiesStandAloneWithoutTheHandoff`は**構造上**の独立を確かめるtestです。**業務上**の独立は構造testでは判定できないため、レビュアーの判断を求めます
- 項目8 / 9: 出典で確認できないactor / trigger / ordering / responsibilityが、accepted factとして紛れ込んでいないか
- 項目13: Referenceを動かすためだけのfake realizationが無いか

### 6.2 seed review checklist

[ISSUE-000050](issue/000050-second-realistic-handoff-corpus.md)の「Gate A green後のRRB-P1 seed review checklist」8観点（Independent value / Provenance / Bounded scope / Meaning-unknown ledger / Actuality boundary / Cross-view value / No fake realization / No forced Handoff）へ、それぞれ回答してください。

### 6.3 Handoff受け入れ条件

ISSUE-000050「Responsibility transfer / Handoff受け入れ条件」の規則に照らして評価してください。

- `from` / `to`を隣接だけで選んでいないか
- actor差だけをtransfer根拠にしていないか
- conditionをtarget triggerへ暗黙に同期していないか
- 「前Stepが終わったら」型のtautologyで満たしていないか。満たしている場合、その完了に独立したBusiness meaningがあるか
- system result / physical fact / state transitionからconditionをbackfillしていないか

### 6.4 準備者の観察（レビュー結果ではない）

パケット準備者がfixtureとtestを読んで気づいた点です。**判定ではありません。** 妥当かどうかをレビュアーが判断してください。

**O-1 conditionと受け手Stepの循環。** conditionの中心は「the released goods … have been taken in charge by the receiving responsibility」で、受け手Step `ST-00`のBusiness Actionは「take the released goods in charge」です。conditionが**Handoff先Stepの完了そのもの**を述べている可能性があります。testは完全一致だけを弾きます（`second_realistic_reference_pressure_test.go` 557〜563行：`strings.EqualFold(got.Condition, s.BusinessAction)`）。内包や言い換えは機械検査の外です。conditionが「責任継続のpredicate」なのか、「`to`の言い換え」なのかを判断してください。

**O-2 数量の責任移転と照合順序。** conditionは「in the quantity production confirmed」「accountability … for their count moves」と、引き受けの時点で**数量への責任**も移ると述べています。一方、届いた数量を生産の確認数量と照合するのは後続の`ST-01`です。conditionが後続Stepの照合結果を先取りしていないか（physical factからのbackfillに当たらないか）を判断してください。U-4（責任境界の詳細）とU-9（受け入れ基準）との関係も確認してください。

**O-3 triggerとfrom Stepの一致。** `finished-goods.stocking`のtrigger「finished goods released for stocking are waiting to be taken in charge」は、`PR-06`「release the confirmed goods for stocking」の完了状態をほぼそのまま述べています。trigger・`from`・conditionが一つの出来事の言い換えに縮退していないか、それが受け入れ条件上許されるかを判断してください。

**O-4 endpointのためのStep。** `PR-06`（引き渡し）がHandoffのendpointを置くために存在していないかを判断してください。受注生産の一回としての業務上の必要から独立に説明できるなら問題ありません。

**O-5 同一作成者・同一protocol。** 作成者自身が「二例目も同じ作成者が同じprotocolでauthorした」と明記しています。二例目が一例目の形を**写した**のか、**独立に反証の機会を与えた**のかを、U-3（step順序は作成者のauthoring判断）を踏まえて評価してください。

### 6.5 RRB-P2項目4とB-F19b status

RRB-P2の項目4は未確認です。次のどちらが妥当か判断してください。

1. 項目4が未確認のままでもP2を達成と扱え、B-F19bを`Execution-qualified breadth Evidence`（P1 + P2 + P3）としてよい
2. 項目4が未確認である以上、B-F19bは`Source-level corpus exists`等の下位statusに留め、Contract ADRの前にData-state / ownership境界を持つ別の圧力が要る

### 6.6 圧力を掛けていない点

作成者は次を「不足が出なかった」ではなく「圧力を掛けていない」と記録しています。それぞれについて、Contract ADRの**前に**圧力が必要か、ADR後のreopen triggerとして保持してよいかを判断してください。

- actorが同じ場合にtransferを否定しないこと（両Procedureのactorは異なる）
- conditionを支えるevent / documentのidentityの要否（U-5）
- actor / responsibility detailをstructuredにする要否（actorは`candidate`ラベルのまま）
- `S4-A`で、別ProcedureのStep IDが衝突するcase（接頭辞`PR-` / `ST-`が異なるため未確認）
- `C1`で、複数Applicationを跨ぐProcedure（Applicationを持たないため未確認）

### 6.7 Evidenceの完全性

可能であれば§3のRelease assetを取得し、SHA-256と`MANIFEST.sha256`を照合してください。できない場合は「未照合」と明記してください。照合できなかったことを、Evidenceの否定へも肯定へも読み替えません。

## 7. 読むべきfile

判定基準:

- [`LANE_B_SECOND_REALISTIC_REFERENCE_ACCEPTANCE_CONTRACT_2026-09.md`](LANE_B_SECOND_REALISTIC_REFERENCE_ACCEPTANCE_CONTRACT_2026-09.md) §2（RRB-P1 / P2 / P3）、§6（B-F19b closure rule）
- [`BUSINESS_PROCEDURE_CANONICAL_CONTRACT_PRE_ADR_AUDIT_2026-09.md`](BUSINESS_PROCEDURE_CANONICAL_CONTRACT_PRE_ADR_AUDIT_2026-09.md) §11、§13、§17
- [`BUSINESS_PROCEDURE_DESIGN_QUALITY_MODEL_2026-09.md`](BUSINESS_PROCEDURE_DESIGN_QUALITY_MODEL_2026-09.md)

作業記録:

- [`issue/000050-second-realistic-handoff-corpus.md`](issue/000050-second-realistic-handoff-corpus.md)（Handoff受け入れ条件、seed review checklist、実施記録）
- [`issue/000048-business-procedure-contract-adr-readiness.md`](issue/000048-business-procedure-contract-adr-readiness.md)

対象（`R`時点で読む）:

- `reference/business-system-design/second-corpus/`（5 file）
- `tools/reference-business-procedure-check/second_realistic_reference_pressure_test.go`
- `tools/run-business-procedure-gate-a.sh`

比較用（一例目）:

- `reference/business-system-design/order-entry-*.json`（`order_entry`のBusiness Procedure Reference）

## 8. 出力の記録先

レビュー結果は新しいfile `plan/BUSINESS_PROCEDURE_BLOCKER_B_INDEPENDENT_REVIEW_RESULT_2026-09.md` に置き、要約をISSUE-000050の完了条件「独立review」の項へ記録します。

```text
reviewer / model:
independence basis:            （作成者Claude Sonnet 5との関係）
review date:
revision reviewed:             82e430109f8f02944519738045f3d6c0bef4f295
review result:                 review-clear | review-clear-with-conditions | review-blocked
RRB-P1:                        達成 / 未達（項目ごとの根拠）
RRB-P2:                        達成 / 未達（項目4の扱い）
RRB-P3:                        達成 / 未達
recommended B-F19b status:
O-1〜O-5 judgment:
findings:
required fixes before Audit re-judgment:
retained unresolved with rationale:
not-pressured points (6.6) judgment:
evidence integrity check:      照合済み / 未照合
test / runner re-executed:     yes / no
```

レビュー結果は**Pre-ADR Auditの再判定そのものではありません**。再判定はレビュー結果と必要な修正を受けて、Audit §17を更新する別の作業として行います。`review-clear`であっても、Contract ADR readyはAuditが判定するまで主張しません。

## 9. このパケットが主張しないこと

- Blocker Bが充足していること
- §6.4の観察が妥当であること（判断はレビュアーが持つ）
- 二例あることがHandoff一般の妥当性を示すこと
- Contract ADRがreadyであること
