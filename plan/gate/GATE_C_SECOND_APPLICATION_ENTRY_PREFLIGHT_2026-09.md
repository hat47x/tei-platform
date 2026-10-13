# Gate C → Second Application Entry Preflight

- 状態：準備済み／試験専用
- 日付：2026年9月19日
- 親文書：`plan/gate/GATE_C_SECOND_APPLICATION_START_PACKET_2026-09.md`
- 関連文書：
  - `plan/gate/GATE_C_EVIDENCE_CONTRACT.md`
  - `plan/gate/GATE_B_TO_C_HANDOVER_BASELINE_B.md`
  - `plan/V1_REFERENCE_APPLICATION_PLAN.md`
  - `plan/MASTER_PLAN.md`
- 正本のアプリケーションへの影響：なし
- 製品の判定段階への影響：なし

## 1. 目的

判定段階CのReady判定後に、第二アプリケーションの作業へ入る際、

- 判定段階Cの状態を示す文字列だけがある
- 表示に関する検証根拠しかない
- 基準版のリビジョンが不明
- ウェブ実行先の構築版が不明
- 実行先の違いが整理されていない

といった状態でアプリケーションの記述を誤って始めないよう、開始手順書の入口条件を試験専用の検証で確かめます。

この事前確認は、判定段階Cの合否を決める正本ではありません。

```text
Gate C authority
  = Gate C Evidence / readiness decision

Entry Preflight
  = already accepted Gate C Evidenceが
    second Application開始に必要な形で揃っているか確認
```

である。

## 2. Authority boundary

事前確認では、次の事項を導出しません。

- 判定段階CのReady判定そのもの
- 意味の準拠確認の合否
- 機能に関する検証根拠の合否
- 表示に関する検証根拠の合否
- 最初のウェブ実行先で生じる違いの分類
- 第二アプリケーションの受理
- 正本上のアプリケーションID

判定段階Cの状態は、外部の権威ある入力として受け取ります。

```text
gateCStatus = ready
  != preflight automatically complete
```

である。

## 3. 必須entry inputs

少なくとも次の入力を要求します。

### Gate C decision

```text
gateCStatus = ready
```

### Handover baseline

- 引継ぎ基準版のID
- 基準となるリポジトリのリビジョン`R`

現在の最初の基準版：

```text
B
R = e5231839a5dcc62cce1181a2731fcb7c70d16736
```

将来、後継の基準版を使用するときも、IDとリビジョンを明示します。

### First Target build

- 実行先の構築版`T1`

基準版`R`と実行先の構築版`T1`を混同しません。

### Evidence classes

- 意味の準拠確認に関する検証根拠への参照
- 対話操作・機能に関する検証根拠への参照
- 表示の忠実度に関する検証根拠への参照

表示の検証根拠だけで、ほかの二種類を代用しません。

### Target differences

最初のウェブ実行先で観測した違いを明示的に記録します。

少なくとも、次の項目について、

- 実行先の能力の候補
- 実行時の対応付けの違い
- 実行成果物・ABI・ホストの違い（観測できる範囲）
- 実行と信頼に関する違い
- ウェブ固有の実装
- 正本へ取り込まない違い
- 未解決事項

何が該当するかを後続の作業から参照できるようにします。

違いがなかった場合も、その確認結果を明示的に記録します。

## 4. Preflight result

試験専用の確認結果は次の二状態に限ります。

```text
entry-evidence-complete
blocked
```

これ自体は製品に関する判断権限を持ちません。

### entry-evidence-complete

開始に必要な検証根拠が揃っている状態です。

意味:

> S0 Gate C entry captureを完了できる入力形がある。

この結果からは、次の事項は認められません。

- 第二アプリケーションのソース作成を自動的に受理したこと
- アプリケーションIDが決定したこと
- 候補の適格性確認が完了したこと
- 環境の対応付けを作成してよいこと
- 共有の受理手続きが完了したこと

### blocked

開始に必要な入力が不足している状態です。

不足している項目を明示します。

## 5. Current repository state

2026-09-19時点:

- Gate B = Ready
- Handover Baseline = `B`
- `R = e5231839a5dcc62cce1181a2731fcb7c70d16736`
- First Web Target build `T = 0ba87a4e0a96f70a198b03ff7652fb00e3826abe`
- Gate C = Ready
- actual Semantic / Functional / Visual Evidence = acquired
- Target Difference Handoff = recorded

actual entry record:


actual pressure:

- `tools/reference-business-procedure-check/second_application_gate_c_actual_entry_pressure_test.go`

current entryは`entry-evidence-complete`として扱える。

これはsecond ApplicationのAcceptanceではない。S3 destination Decisionsは別途必要である。

## 6. Negative pressure

### A. Visual only

```text
gateCStatus = ready
visualEvidence = exists
semanticEvidence = missing
functionalEvidence = missing
```

結果:

```text
blocked
missing:
  semantic-evidence
  functional-evidence
```

表示の評価者や画面画像の評点だけで開始条件を満たしたとは扱いません。

### B. Ready label only

```text
gateCStatus = ready
B / R = missing
T1 = missing
```

結果:

```text
blocked
```

判定段階Cの状態を示す文字列だけを、第二アプリケーションの開始を認める根拠にはしません。

### C. Target difference recordなし

三種類の検証根拠が揃っていても、実行先の違いが記録されていなければ開始を認めません。

理由:

第二アプリケーションや知識の再利用に関する作業で、最初の実行先固有の違いを正本上の意味と誤認するおそれがあるためです。

### D. Current pre-target state

```text
B / R = known
Gate C = not-ready
T1 = missing
Gate C Evidence = missing
```

この状態は明示的に開始不可とします。

## 7. Test-only source

- `tools/reference-business-procedure-check/gate_c_second_application_entry_preflight_pressure_test.go`

確認する事項は次のとおりです。

1. 必要事項が揃った合成の開始条件は`entry-evidence-complete`
2. 実行先の準備前の状態は開始不可
3. 表示の検証根拠だけでは開始不可
4. `ready`という表示だけでは開始不可
5. 実行先の違いの記録がなければ開始不可
6. 適格性確認の型に記述作業への近道を追加しない

## 8. Prohibited shortcuts

試験専用の適格性確認には、次の項目を含めません。

- `AutoStart`
- `CreateApplication`
- `Accepted`
- `Approved`
- `CanonicalApplicationCreated`
- `SourceAuthoringAttempted`
- `EnvironmentBindingCreated`
- `TargetImplementationStarted`

事前確認では、開始に必要な検証根拠が揃っているかどうかだけを扱います。

## 9. Executable Evidence runner

環境を移しても利用できる実行器：

- `tools/run-knowledge-portability-reference-evidence.sh`

対象となる名前の接頭辞に、

- `TestGateCSecondApplication`

これを追加します。

GitHubが提供する実行環境を再び利用できるようになった場合は、

- `.github/workflows/knowledge-portability-reference-evidence.yml`

同じスクリプトを実行します。

実行が成功した記録を得るまでは、実行可能な検証根拠について合格とは扱いません。

## 10. Master Plan同期

`MASTER_PLAN.md`では、当時の検証根拠に合わせて、

- Gate B Ready
- Handover Baseline `B` / `R`固定済み
- Gate C Not Ready（起票時点。2026-09-19にT = `0ba87a4e`でReadyとなった）
- First Web Target `T1`未成立

これらを一貫して示します。

上位計画間の状態の不整合を、第二アプリケーションの開始条件に持ち込みません。

## 11. Gate C Ready後

判定段階CのReadyが正式に成立した場合は、次の順序で進めます。

1. 判定段階Cの検証根拠の正本を確認
2. 事前確認の入力に`B/R/T1`、三種類の検証根拠、実行先の違いを記録
3. `entry-evidence-complete`を確認
4. `GATE_C_SECOND_APPLICATION_START_PACKET_2026-09.md`のS0を完了
5. 同じ主担当のISSUE-000040作業ブランチでS1へ進む

事前確認に合格しただけで、S4以降へ進んではいけません。

## 12. 完了条件

- [x] Gate C authorityとentry completenessを分離した
- [x] B / R / T1を別identityとして要求した
- [x] Semantic / Functional / Visualを別Evidenceとして要求した
- [x] Visual-onlyを拒否した
- [x] Target差分recordを必須にした
- [x] pre-target stateをblockedとして説明できる
- [x] actual Gate C Ready entryを`entry-evidence-complete`として固定した
- [x] authoring shortcutをqualificationへ持ち込んでいない
- [x] portable runnerへ接続した
- [x] Master PlanのGate B stale記述を修正した
