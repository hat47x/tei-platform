# TEI×EKI 回復の実行資格・期限・予算を同時に判定する最小受入計画（2026-11-03）

> **段階：横断計画の実装前検証／合成Reference。** TEI横断計画§31（受注対象・Commit不明）、§32（Source/意味版）、§35（Resource Group）、§36（期限・回復余力）、§37（組織Policy）およびEKI既存ADR/Issueを結ぶ。新しい製品Scheduler、Receipt Engine、Recovery Permit、Central Cost Registry、TEI Hostを提案しない。

## 1. 「再実行してよい」と「再実行する価値がある」を同じ成功状態に潰さない

**A — 実行資格（再試行の可否）**：実Job/ExecutorのSource・意味・契約Pin、Required ReadのScope/Completeness、Command選択対象と明示Payload/実Write対象の対応、現在Host Grant/Policy版、Business Operation IDと先行Receipt、Commit uncertain、EKI Attempt/Lease/Fence/Recovery Permit、下位Backendの内部自動再実行を照合する。

**B — 期限内回復の実行可能性（配置を選ぶ理由）**：実Resource Groupの適格なsubset・現在Reserve/Allocation、残Attempt・物理Budget、Queue/起動/Input Transfer/Compute/Output Transfer/Verification/回復Allowance、外部Ownerの承認済み支出枠、障害相関・Fairnessを照合する。実行時間の予測とSLA、Capacity forecastと実予約、EKI物理BudgetとCore外の通貨建て予算は別物である。

A・Bが両方真でも**現実の配置権限は発行されない**。実EKI Admission/Permit/Lease、実Host/ProviderのAuthorization/Tx/Receiptによる現在判断を待つ。「安全に再試行できる」ことは「選ぶべき最安経路」ではなく、順位付けの前提。

| 状態の組合せ | 残す判断 | 不適切な自動動作 |
| --- | --- | --- |
| Commit unknown、期限は残る | 同Operation IDのReceipt/実Effectを再照合 | 違うIDで再Submitする、失敗と断定する |
| Effect committed、Output body不明 | Result取得・独立Read・Consumer到達を別々に検証 | 結果欠落を理由にBusiness Writeを繰返す |
| EKI completed、accepted body未受領 | Result取得・検証の経路へ進む | 完了ACKだけでTEI Business complete |
| 旧Attempt/Lease active・回収ACKのみ | 現行Fence・実下位Job状態を確認 | 新Groupへ二重配置する |
| Source/Role/Schema/Org Policy/Host Grantだけ変更 | 対象Claimの現在資格を再評価、過去Artifactは履歴として保持 | Build skipから実Action/Disclosureの資格を推測 |
| Pure Compute、実証済み入力・Permit・資源・期限・予算を満たす | **新Attempt候補**をEKI/Callerへ返す | 合成評価の結果だけでLease/権限を発行 |
| Budgetは十分だが残り時間・結果検証に不足 | not feasible、期限・要件変更の必要性を通知 | 最安の初回計算だけを選ぶ |

例：Hard期限90分、Queue25＋Input転送15＋Compute35＋Verify10＝85分。初回成功だけなら間に合うが、失敗後の最短回復25分を要求するなら計110分となり「期限内回復可能」は成立しない。**初回の最安費用を選ぶ前に、再試行適格性と回復余力を確認する。** Providerの見込み空き枠、障害の独立性、価格表・Grant/Permitを未観測なら0分/0円/安全と仮定しない。

## 2. 実Go・SQL・組織Policy・EKIの境界を同一Targetで検査

| 入力証拠 | 既存Owner・具体的な反例 | 止める位置 |
| --- | --- | --- |
| 実Target・Receipt・UoW | [TEI #2780](https://github.com/hat47x/tei-platform/pull/2780) MC-B01：Command ORD-1にPayload ORD-2がFake WriteResourceへ、MC-C01：内側Fake Commit後に外側Commit失敗、MC-D01/D02：新規/ReplayのCommand結果同値・Commit unknown一般エラー | [#2598](https://github.com/hat47x/tei-platform/issues/2598)/[#1174](https://github.com/hat47x/tei-platform/issues/1174)で正規Target変換・実Host current Grant・同物理Tx/Receipt/独立Readを受理するまで再投入停止 |
| SQL対象集合 | [TEI #2778](https://github.com/hat47x/tei-platform/pull/2778) SM-15実SQLiteのNATURAL JOIN：SQL byte同一でも新同名Fieldにより結果2→1、SELECT * Shapeも変更 | [#2622](https://github.com/hat47x/tei-platform/issues/2622)/[#2600](https://github.com/hat47x/tei-platform/issues/2600)で実Schema列集合・暗黙Join・Function Required Relation/Fieldの消費閉包を再照合 |
| Source/Business Meaning | TEI外部ImportのsemanticDigest一致≠原Source/Owner受理、組織階層/広域Record Policy/個別除外が変わる | §32/37、[#2386](https://github.com/hat47x/tei-platform/issues/2386)/[#2516](https://github.com/hat47x/tei-platform/issues/2516)：現在のScope/Read/Effect/Disclosureを用途別に再資格化 |
| Group・期限/Permit・内部Retry | [EKI #115](https://github.com/hat47x/eki-fabric/issues/115) hidden backend retry、[#154](https://github.com/hat47x/eki-fabric/issues/154)既存Recovery Permit、[#155](https://github.com/hat47x/eki-fabric/issues/155)公平性、[#58](https://github.com/hat47x/eki-fabric/issues/58)Suitability | Current Fence/Lease・実Budget・適格subset・現在Reserve・期限内Output検証のどれかが欠ければ配置候補に昇格しない |
| Consumer結果 | [EKI #112](https://github.com/hat47x/eki-fabric/issues/112) Result Artifact Body、[#126](https://github.com/hat47x/eki-fabric/issues/126) Task Group、TEI #1130 Job | SHA/ACK/Task completedがあるだけでは実Body/業務Result/全件検査完了を認定しない |

**証拠の階段**：#2780の固定HEAD c9927f247839106c28811e14abba16a53bd38e98に対する[Focused Go Actions #41632111078](https://github.com/hat47x/tei-platform/actions/runs/41632111078)では、MC-B/C/Dの**4/4 Characterization PASS**。これは**不安全な現行構成や情報消失を検出した**合格であり、製品が安全に対象漂移を拒否・実DB Commitを回復したという成功ではない。#2778のSQL/Goテスト、#2681 Native組織モデル、#2781組織Policy合成、EKIの局所テストを**別々の実証**として保持し、同一Product E2Eへ合算しない。

## 3. 合成Fixture RA-01〜26（Referenceの形式を公式ABIへ昇格しない）

[check_recovery_admissibility.cjs](../fixtures/recovery-admissibility/check_recovery_admissibility.cjs)は、**26の合成一軸対照**について独立Expectedラベルと小さい決定関数を照合する。Nodeによる手元実行入口を持つが、本追補作成時の証拠はGitHubから保存済みJSを取得してV8で実行した**26/26 PASS**に限られる。CodeとExpectedは同じReferenceにあるため、実製品の別実装に対する独立Oracle合格を意味しない。

| 反証束 | Case | 期待する分離 |
| --- | --- | --- |
| History/Commit/Body | RA-01〜05, 25 | Verified既存Resultで新Attemptを起こさず、Commit unknownはOutputがあってもreconcile-only。Committed/Task completedとBody実取得を区別 |
| Target/Source/Grant/Read | RA-06〜08, 21, 23, 26 | O1/O2・Grant失効・partial Input/Source・不適格資源・Owner未受理のRetryを拒否/保留 |
| 下位Retry/Lease/Permit | RA-09〜13 | Effect付きの不透明Retry、Pureでも予算計測不能なRetry、旧Lease存続、現Permit失効を分ける |
| 物理・外部金額・期限 | RA-14〜19, 22, 24 | Forecast≠Reserve、物理Budget≠外部価格承認、Verification/回復込み期限・未知時間 |
| 相関リスク | RA-20 | 同じ障害Domainを独立複製として扱わない |

ラベルはfixtureに閉じた説明用文字列。candidate-onlyは実Policy/Permitを与えず、reconcile-only/retrieve-only/observe-onlyも実Reader権限や結果取得を保証しない。特に**Commit uncertainを最優先で分離**し、既Verified Bodyだけで業務Commitを確定しない。

## 4. 実証の順序・採用しないもの

**Q0**：TEI #1130の実read-only Job Source/Executor/ExpectedとEKI #112のExternal Artifact/Resultを、同一Source版・Input Byte・Task/Attempt・実Consumer用途で棚卸し。現APIが不足すればNOT RUNで止める。

**Q1**：まずEffect-freeな一件でGo単独／EKI Native／EKI→実外部Groupを比較し、Artifact Bodyを独立Readerが読むこと、Dataset/Read Cohort、Queue/Transfer/Compute/Verify/Recovery時間、Provider cost、Attempt数を固定。EKI #19のMLはDeterministic Baselineに対するshadow評価のみ。

**Q2**：OwnerによりBusiness Target変換、OperationID、現Grant、同物理Tx/Receipt、Commit unknown/Lookup/独立Readの実Provider受入が揃った後に限り、Effect付き再試行を検証。Opaque Backend retryをdisable/観測できず、実冪等性根拠もなければ委譲を棄却。

**Q3**：EKI #154のPermitと#155 Fairness、#67のGroup候補/実Reserve、#115内部Retryを既存Control Planeで実観測。Cost/外部通貨Policyの受理はEKI Core外。価格・Capacity予測で現Grantを上書きしない。

**Q4**：Source移動/Role変更/NATURAL JOIN Schema変更/組織Policy版・Grantだけの失効を注入。Build再利用と、Source再確認、実Provider Read、現在Action資格、Consumer開示の再資格を分離。危険な再利用と不必要な全再Buildの両方を採点する。

**停止条件**：実Target・Receipt・同一AttemptのEffect安全性が未証明なら、期限内に間に合う予測でもBusiness Effect再投入を行わない。期限/Verification/物理Budget/外部支出許可に違反するなら再投入が許可されていても新配置を行わない。現在のEligible Candidateは、実EKI Permit/Host Grantを発行しない。正式Owner判断、実Provider、Node/Go CI、外部Job、金額費用・性能の実測、実Job ABIは**NOT RUN／未採用**。既存TEI #1119 P0-a/P0-b・EKI ADR-0013/0015を変更せず、TEIのGo単一バイナリ直接実行を維持する。
