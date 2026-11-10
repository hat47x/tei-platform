# 三要素・ドメイン知識に基づくテストデータ生成と開発者支援

2026-11-07。**状態：設計・実証計画／新しい生成器と一体化した製品経路はNOT RUN**。本書は既存Ownerを接合する設計であり、新しいDataset/Scenario/独立期待値の正本ではない。

## 1. 目的

TEIの開発支援は、ランダムな入力値を大量に作るだけでなく、**ドメインモデルからテスト義務と反例を導き、有効なDataset・意図的な制約違反・独立Expected・実行結果・影響分析を一貫して扱う**。生成AIは不足の発見、候補説明、反例縮小、修正案・レビュー順序を支援する。意味の受理やHost認可は代行しない。

三要素分析法は **Data（Entity/Key/Relation/Domain）、Function（入力/Query/Rule/操作/出力）、Business（目的/契機/役割/責任/手作業/DFDデータフロー）** の相互照合に使う。業務モデルのDFD矢印はWorkflowの実行順・自動起動・人間の引受完了を直接意味しない。Procedure/Workflow Instanceは明示的な参照で接続する。

### 三つの視点だけでなく、論理要件・物理制約・実装観測を分ける

三要素分析法の解説では、Data／Function／Businessの三視点を、**論理要件・物理制約・実装**にも分けて捉える。テスト支援ではこの違いが「設計上のルール」と「Targetで強制されるルール」を区別する根拠になる。

| 視点 | 論理要件の例 | 物理制約の例 | 実装・Actualの検証 |
| --- | --- | --- | --- |
| Data | 受理されたKey/FD/値域/Relation/時間 | DB型、DDL、CHECK/FK、Provider実現能力 | 実Rowset、格納値、Query・更新結果、Snapshot |
| Function | 入力、Rule、Query、操作、Output契約 | 言語/Runtime、API、Target差、Disclosure | 実呼出し、SQL+bind、Operator、Consumerの実受領 |
| Business | 業務目的、Actor/責任、Event、DFD、Procedure/Handoff | 実組織・人員・運用手段、現実の作業制約 | 手作業/承認/外部引受の独立観測と未観測 |

1つのCaseでも、「論理上は拒否すべき」「DBの現CHECKでは拒否できない」「実際のFunctionは局所PASS」「人間の承認は未実施」は同時に成立し得る。**モデル一致・Target実行・業務完了の評価結果を合算しない**。物理制約を変更しても業務意味の受理が自動更新されるわけではない。

参照：[三要素分析法（1）](https://atmarkit.itmedia.co.jp/fbiz/cbuild/complete/doa/01/02.html)（三視点×論理要件・物理制約・実装）。これは出典に基づく構造の補足であり、原著の「強属性」の定義を確認したものではない。

## 2. 既存契約の再利用・責務分担

| 正本／担当 | 本計画で再利用するもの | 追加する仕事 |
| --- | --- | --- |
| #1131 / #1180 / #2621 | Dataset・Snapshot・Scenario・ExpectationとMarkdown記述 | 同じ正本へモデル駆動のCase候補を提出 |
| #1126 / #2495 / #1120 | 業務Identity、Key、FD、Relation、Trait Claim、型付き参照 | 制約と出典を持つ検証義務の導出 |
| #1125 / #1440 / #1444 | 業務DFD・Procedure・Actor・Handoff | 手作業・外部結果・責任・例外の負例提案 |
| #2387 / #2506 / #2586 / #2622 | Query、Result Grain、Data Context、Provider実Read | データの完全性と生成Caseが検査可能な範囲を説明 |
| #2514 / #2811 / #1163 | 独立Expected、Known Mutant、AIテスト追加、実行証拠 | Test生成の検出力と適用資格の評価 |
| #1445 / #2018 / #2516 | AI支援工程、UX/費用比較、用途相対Context | 開発者向け不足検出、診断と再検証案 |

**新しいSQL Runner、Scenario SSOT、Host、Domain Authority、普遍的Constraint Solverは前提にしない。** 生成元のAccepted Meaningと提案中の推論を区別する。

## 3. モデルから読み取る検証条件

- **値域**：基本型、区間境界、閉集合、Enum、NULL/欠損/空文字、単位・通貨・精度・丸め時点、時間範囲。Closed Domain、Exclusive、Exhaustive、MECEは別のClaimであり、Java EnumやDB CHECKだけから業務MECEを推定しない。
- **強属性**：「強属性」の一次資料に基づく厳密な定義は現時点で未確認。これを新Core特性として固定せず、候補キーへの寄与、識別不変性、時間的変更許容、FD、Authority、Roleの別々の検証軸を提示する。原義への一致はOwner/原典確認まで未解決。
- **Data構造**：複合業務Keyと物理PK、関数従属性、親子・参照・サブタイプ、Role別多重度と有効期間、Occurrence identity、JOIN増幅と集計Grain。
- **Function**：入力必須/任意・未入力、Data Rule/Context Rule、Query Predicateと非出力依存、更新前提・Version・Scope、Provider能力、Permission/Disclosure、独立readとConsumer受領。
- **Business**：イベント／手動／時刻契機、Actor/責任、業務間のデータ授受、手作業の未実施、承認/差戻し/部分処理/取消、外部応答なし、Handoffの条件。
- **広域整合性**：伝票と明細・出荷・請求・在庫・組織・会計の関係を、受理済みRuleに沿って横断照合。同一DBトランザクション内の不変条件と、別ProviderやReplica間の途中/不明状態は分離する。全システムの原子性は推測しない。

## 4. 生成から診断までの最小処理

1. **Inventory**：実際に読み込んだSource、型付きID、Model/Rule Revision、受理状態、Relation path、明示的なScopeを収集。Loader非対象・参照未解決・Source欠落は不足として表示。
2. **Obligation Derivation**：各Rule/Constraintに必要な対象集合、入力項目、時間、Actor、Function、Provider Capability、期待する観測先を列挙。未受理のConstraintは提案であり必須受入条件にしない。
3. **Candidate Generation**：固定Seedの正常例、境界値、単一Constraint違反の負例、関係グラフを維持した変形、意図的な部分集合・重複を生成。AIは未知の業務規則を候補として提案するが自動確定しない。
4. **Preflight**：候補の分類を「検査済み範囲で充足」「意図的な負例」「矛盾が証明された不能」「未対応/Timeout」「未検証」に分ける。Generatorが値を作れなかっただけでUnsatisfiableと断定しない。無許可の実DB投入を禁止。
5. **Independent Expected**：生成器と同一の誤仕様に依存した期待値の自己採点を禁止。Owner受理済みRule、事前固定Fixture、人手算定、別実現、既知Mutantなどの独立根拠を明記し、未受理Expectedと正式Expectedを区別する。
6. **Execution/Observation**：既存Go Test/Scenario/Provider/Hostへ渡し、実Step・Actual・Receipt・Read・Consumer・SKIPを別記録。局所PASSと業務全体の受理を同一視しない。
7. **Diagnosis and Change Impact**：どのモデル要素/制約/Source/Query/Scope/時点が差分に関係するかを逆追跡。最小反例へ縮小し、修正候補、再検証範囲、未実行の必須検査を提示。Sourceや業務意味は明示承認なく書き換えない。

## 5. 初期Case Matrix：公開合成の受注例

固定例：受注H、明細L1=数量2×単価100、L2=数量1×単価50、顧客C、出荷S、承認Procedure。**合計250はこの例で仮定したRule**であり全案件に自動適用しない。Tenant A/B、Snapshot T0/T1、Actorを明示する。

| ID | 誘導する正負対照 | 誤成功として検出する内容 |
| --- | --- | --- |
| DG-01 | 値域境界・NULL・欠損・閉値域外コード | 単なる型適合を業務値域の完全性と扱う |
| DG-02 | Tenant込みの複合業務キー、同OrderNoの別Tenant | 異Tenant同番号を重複と扱う／Tenantフィルタ漏れ |
| DG-03 | 同じ決定項に異なる従属値、履歴の別時点 | FDの不整合と合法な時間変化を混同 |
| DG-04 | 親子の孤児、Role別1:N、JOIN 2×3増幅 | 6行を6業務明細と誤認、金額の重複集計 |
| DG-05 | 数量・金額・通貨・丸めの境界 | 受理されていない計算法・単位換算の推測 |
| DG-06 | 兼務、代理権失効、遡及訂正、Membership期間 | Person・Position・Occupancy・Grantの混同 |
| DG-07 | Target L={3,9}に対するR=[3,3]／TOP1=[3] | 行数一致・ページ取得から全件完了を推測 |
| DG-08 | DB格納status=READYだがSELECT非選択列にNULL alias | Raw nilを業務値NULLと誤読（既存Q08） |
| DG-09 | 非出力ordered_atでSQL WHERE評価、statusは未評価 | 実評価されたPredicateと未評価の述語を混同（既存Q07/Q09） |
| DG-10 | DFDの業務イベント、電話確認未実施、承認待ち | データが流れただけで手作業/承認が完了したと推論 |
| DG-11 | 一部出荷・取消・差戻し・異順序の行動 | 受理されたRuleがない状態でWorkflow順序を創作 |
| DG-12 | コミット応答喪失、再送、独立Read不可 | Command/Receipt/Resource/Businessの成功を一本化 |
| DG-13 | D1完全・D2部分、NOT EXISTS・SUM・JOIN | 部分観測を不存在や全件集計の証拠へ昇格 |
| DG-14 | A→B選択後の旧A遅延応答、A→B→A版変化 | 旧Epochや旧Policyの証拠を現在のResultへ混入 |
| DG-15 | private属性によるFilterで行の有無/順序が変化 | 非投影であっても推測開示が起こり得る点を見逃す |
| DG-16 | Go/Java/DBやCLI/Webで同じ受理済みRuleを実現 | 一Targetの局所PASSを他Targetの保証へ転記 |

**Coverage分母**はモデルObligation×制約種別×正例/負例×Function/Provider×Scope/時点/観測先。テスト件数ではなく、独立Expectedと既知Mutantを使った判別力を測る。直積総当たりを初期要件にせず、依存閉包と重要な不変条件から優先する。

## 6. 開発者支援の段階

**P0：Headless/CUI＋生成AI（read-onlyの提案）**。指定したData/Function/Business Subjectの受理済み制約、未解決Rule、未検証Obligationを提示。候補Fixture/反例/Seed、根拠と適用範囲、独立Expectedの必要箇所、曖昧な質問、既存Scenarioとの重複を説明。実在するCLI・Test APIだけを使用し、未実装の新CLIコマンドを採用済みと書かない。

**P1：検証・修復支援**。Candidate→Review→既存Dataset/Scenarioへの採用差分を示す。失敗時の最小反例、根拠Sourceへの移動、Data→Function→Businessの影響グラフ、変更後に再実行すべきテストと保持可能なEvidenceを表示。外部Data/秘密/手作業/現Grantを自動取得・変更しない。

**P2：制約探索と多Target適合**。Property-based生成、状態モデル、境界組合せ、Metamorphic Test、Mutation、差分Target、必要ならSolverのPluginを比較。各手法には前提、Scope、計算予算、最小化能力、判定不能条件を明示。第二Rule Engineや常駐全域Solverにしない。

## 7. 説明用の最小Evidence項目（正式Schema未採用）

Case/Subject typed ID、Model+Meaning+Rule Revision、Source provenance、Scope/Tenant/validAt/knownAt、Constraint、Rule Owner/Status、必要TargetSet/Grain、Dataset/Snapshot/Seed/意図的Mutant、独立Expectedと出所、実Query+Args/Provider/Read Epoch、Raw Cellの選択状態・DB NULL・非開示、Actual、Consumer Delivery、Effect/Independent Read、判定可能範囲、Unknown/NotRun、反例・変更影響を**個別列**として扱う。

特に **Declared → Loaded → Accepted → Generated → Executed → Observed → Qualified** を混同しない。Goの全テスト成功、実PG Referenceの成功、形式的な制約充足は、独立のBusiness Meaning・Product Function→Provider→Consumer全経路の資格ではない。

## 8. 実装順と受入条件

| 段階 | 既存Owner | 実証する最小条件 | 現在状態 |
| --- | --- | --- | --- |
| D0 三モデルのObligation Index | #1126/#1125/#2495/#1131 | 同じ公開受注例を三視点から辿り、受理/候補/欠落を区別 | 計画 |
| D1 決定論的Fixture候補 | #1131/#2811 | DG-01〜04・07の正負を固定Seedで生成し、意図的違反を区別 | 未実施 |
| D2 独立期待値の判別 | #2514 | 値域・FD・JOIN・NULL・ScopeをKnown Mutantで区別、AI自身のExpectedを盲信しない | 未実施 |
| D3 Business Scenario候補 | #1125/#1131 | DFD≠Workflow、人手未観測、取消/承認/差戻しの有限状態差分 | 未実施 |
| D4 実Read/Consumer | #2586/#2506/#1163 | 実Product Function→選択Query→Provider→Context/Rule→Consumerの同一Attempt | 未実施 |
| D5 工程・効率 | #1445/#2018 | 手動厳密・AI支援厳密・コード先行を同じ独立Expected/検出力で比較 | 未実施 |

最初の縦断は**DG-01〜04・07**とする。制約の矛盾、受理不足、Provider不完全性などが判定不能なら診断を残す。データが生成できた事実だけを「意味的に正しいテスト」「Owner受理済み」へ昇格しない。

## 9. 実施状況と停止線

この変更は既存SourceとIssueに基づく**計画の統合**であり、ここで新しいGenerator製品コードやAIテスト生成Attemptは実装していない。先行#3238のGo/隔離PG Readと#2514のQ05〜Q09は**独立した限定的な参照実証**。本計画のD0〜D5を実行済みと主張しない。正式「強属性」の定義は未受理、Business Rule／現在Host Grant／全Provider／Owner業務意味の受理は別境界。

関連：#1131 #1180 #1445 #1120 #1125 #1126 #2495 #2506 #2586 #2514 #2811 #1163 #2622。
