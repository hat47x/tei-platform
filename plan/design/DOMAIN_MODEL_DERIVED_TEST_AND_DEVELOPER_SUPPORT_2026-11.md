# ドメインモデル由来のテスト生成・検査・開発者支援（設計／段階的受入）

- 更新: 2026-11-07
- 状態: 設計・受入候補。新しい製品Generator、Constraint Solver、実業務E2Eの完了を意味しない。
- 主担当の再利用: #1131（型付きDataset/Snapshot/Scenario・テストデータ生成）、#1126（Key/FD/Relation）、#2495（値域・単位・数学的Facet）、#1806（Semantic Type）、#2593（Measure）、#1125（Business Procedure/手作業）、#1127（Operation）、#2506（Result Slice）、#2586（実Read接続）、#2514（独立Expected/Conformance）、#1180（Scenario正本）、#1163（Evidence）、#2811（AI-firstテスト生成）、#3223（テスト製品調査）。
- 設計原則: 生成AIを候補探索に使い、決定論的な制約検査と独立Expectedの照合で検収する。業務意味はOwner、現時点の許可はHost/Providerに残す。第二のScenario/Expected/Meaning/Policy正本や新Runnerを設けない。

## 1. 目指す開発者体験

テスト支援を「ランダムデータの大量作成」から、**三要素（Data/Function/Business）の意味から検証義務を発見する開発者支援**へ拡張する。

1. モデルとSourceをread-onlyに解析し、キー、値域、関係、Function入出力・効果、業務手順、責任・契機、業務上の不変条件を出典／改訂付きで一覧化する。
2. その情報から、正常例・一因子違反の反例・境界値・同値類・状態遷移・並行操作・集計/参照の非単調例を**候補**として列挙し、どのRequirementやモデル要素を試すための候補かを明示する。
3. 制約の整合性を検査し、生成可能／矛盾／必要入力不足／権限不明／対応できない式を分けて報告する。解なしを「0件の正常データ」と扱わない。
4. 受理済み意味に照らし、Domain担当が固定した独立Expectedと候補生成物を突き合わせて既存#1180 Scenarioへ引き渡す。ActualからExpectedをコピーしない。
5. 隔離環境で実行し、必要Caseの発見・非SKIP・Source/Attempt/Providerの同一性、実Rowset・Receiptと別Read、Consumer受領範囲まで検査する。
6. 開発者へ「なぜこの試験が必要か」「何が未検証か」「どのモデル要素・Rule・Function・業務手順が関係するか」「最小反例」「変更時の再検証集合」を説明する。修復案はSource候補へ返し、Owner受理やProduction Effectを自動実施しない。

GUIに先行してCLI/CUIの構造化結果で一往復を成立させ、Studio/SHO/外部Targetは後から既存のRead-only Projectionで利用する。

## 2. モデルから取り出す制約の6階層

| 層 | 主な制約と検査要求 | 固定を避ける推論 |
| --- | --- | --- |
| Data / Semantic | 基本型、値域、桁/精度、単位/次元、NULL/欠損/非開示、Closed/Exclusive/Exhaustive、業務キー/物理PK/候補キー/FD、1:1・1:N、valid-time | 単なるEnum/CHECK/UNIQUEから業務MECEや正式な意味を証明しない |
| Data / topology | Entity、Relation occurrence、Path、Set/Bag/Sequence、時間・Graph、JOIN行増幅、Aggregation grain、外部参照、跨DB schema | 表示行数・物理行・集計結果を同一業務対象集合としない |
| Function | Input/Output、Queryと非出力依存、派生値、Guard、Rule hit policy、read/write set、更新前提条件、ID/Version/CAS、Effect拒否 | Form入力の妥当性だけで実Read/Commit/Receipt/Business完了を主張しない |
| Business | Actor/責任、Procedure、手作業、外部イベント、承認/差戻し、例外/取消、引渡し、完了条件 | DFD矢印を自動Workflow起動、業務担当をHost Grantに変換しない |
| 広域整合性 | 複数集合同士、参照先、在庫・金額・割当、組織兼務、取引全体、履歴の有効期間、同時更新、跨Tenant/Provider | 一テーブルのCHECKや一回のReadの成功で全域・全時点を保証しない |
| 証拠/開示 | Source/Plugin Revision、actual Query、Scope、Read epoch、Target L/Predicate P、結果のField Presence、秘密に依存するQuery/結果集合、Consumer | 非投影のNULLプレースホルダーを業務NULLと扱わず、非公開列を勝手に取得・露出しない |

### 「強属性」の扱い

#2495等の現在の記録では、**強属性の原典定義と正式なMeaning受理は未確認**。正式定義が確定するまで「必須属性」「業務キー」「強い関数従属性」「分類に用いる強い区別条件」を勝手に同義としない。

入力Modelにこの語がある場合は、Source位置・意味候補・Owner・適用範囲・対照例・受理状態を提示し、具体的な制約（例：識別の単射性、変更不可性、NULL拒否、複合キーの有効時点）に**受理済みの明示対応**がある場合だけ生成Ruleへ利用する。未受理なら観測・確認課題として保持し、制約を作り出さない。

## 3. 生成と検査を分離する契約

入力：利用目的/対象Claim、出典・Scope・Model Revision、受理済みConstraintとReference候補、独立ExpectedとNegative Control、現在利用可能なTest Provider/Authority。

各候補が最低限持つ説明：Source locator/Meaning decision、対象Constraint ID、Applicabilityと版、Positive/Negative区分、生成seed/Algorithm version、Entity/Relation/Occurrence/時点/Unit、最小入力Dataset、期待違反の因子、残る未検証範囲、実行に必要なCapability、データ取扱い分類。

検査フェーズ：
- **構造整合**：型、値域、単位、Key、FD、Relation、重複、NULL、履歴、Multi-facetを決定論的に検証。
- **独立Expected**：生成器/実Evaluatorとは独立に固定した正負Case。正常系も必要で、常時拒否・常時unknownを合格にしない。
- **実行整合**：ScenarioのRequired→Loaded→Bound→Executed→Observed→Qualifiedの各列。SKIP・timeout・未発見・非出力Predicateの無根拠自己申告・別AttemptのReceiptを区別。
- **Coverage**：モデル要素数や生成件数ではなく、検証義務と判別力（反例で検出できた既知Mutant）を数える。全Domain/全状態への外挿をしない。

必要ならSAT/SMT/制約ソルバーを**候補を生成する補助手段**として比較するが、解が存在することと業務意味が受理済みであることは別。Solver非対応式、無解、timeout、非決定性、矛盾したConstraintを区別し、Unknownから正常Fixtureを捏造しない。実DB分布・個人情報・秘密Sourceは既定で外部モデルへ送出しない。

## 4. 最小の独立対照（テスト導出案）

各行は既存#1131のDataset/Snapshot、#1180 Scenario、#2514の独立Expectedに接続する**Coverage Obligation案**。新しいScenario正本ではない。記載の採否・本実行はすべて未完了。

| ID | 成立する例 | 一因子違反・独立Expected | 優先 |
| --- | --- | --- | --- |
| DGT-01 値域 | 受理済みClosed値域の全値を合法に生成 | 未定義値と「まだExhaustiveではない」値域を区別、Enum一致だけで業務完備としない | P0 |
| DGT-02 境界/量 | 0、上限、精度、通貨/時間単位を定義どおり処理 | overflow、境界±1、丸め、異単位。物理Decimal一致だけで再集計可としない | P0 |
| DGT-03 強属性 | Ownerが明示受理した制約に対し正負対照 | 意味未受理の「強属性」語だけからKey・不可変更・NOT NULLを生成しない | P0 |
| DGT-04 Key/Relation | 異なる物理PKでも業務ID/Occurrenceが一致 | Key衝突、孤立子、重複Parent、異なる有効期間、関係多重度違反 | P0 |
| DGT-05 NULL/Presence | フィールドの実値NULL・未取得・非開示・非選択を区別 | SQL NULL placeholderを実値NULLと誤認、欠測を0/falseで置換 | P0 |
| DGT-06 Join/Grain | D1=2、D2=3から別Relationとして読み6行のJOINを説明 | JOIN6行を6 Entity、SUMの三重計上、COUNTとCOUNT DISTINCT混同 | P0 |
| DGT-07 Coverage | 独立L={A/3,A/9}と実R={A/3,A/9}が同Scopeで一致 | R=[A/3,A/3]同件数異集合、TOP1、stale/異Tenantの負例。#3238参照 | P0 |
| DGT-08 非出力Predicate | 明示許可されたWHERE/Operatorで必要値を評価し、Public ResultはIDのみ | 値未評価なのにHolds=true、または非許可Predicateで件数から秘密漏えい | P0 |
| DGT-09 業務Rule | 入力とContext固定で受理済みGuard/拒否・hit policyが再現 | 短絡の有無/Rule順序を入れ替え、未許可の外部項目を先読み | P1 |
| DGT-10 Business Procedure | 受注→確認→承認→出荷準備、手作業の確認を別Witnessで扱う | Function成功だけで承認・外部受領済みを生成、差戻し後に旧Grantで遷移 | P1 |
| DGT-11 時間・組織 | 兼務・所属・代理権のvalid-timeが一致 | 代理権失効・遡及訂正・A→B→Aで旧承認を現在有効にする | P1 |
| DGT-12 横断不変条件 | 同じ業務Key/Units/Revisionで注文・明細・在庫・請求の関係が成立 | 取消済受注の出荷、部分出荷と二重請求、別Providerの非同期遅延を同期とみなす | P1 |
| DGT-13 並行操作 | 版照合/CASとOperation ID、Commit/Readが追跡できる | lost update、再送二重作用、ACK-loss、Replica遅延、old Receipt流用 | P1 |
| DGT-14 多Consumer | 同じData ContextをForm/Table/CLIへ目的相対に投影する | 表示順のID誤認、hidden Relationの候補件数漏えい、Projectionから自動Patch | P1 |
| DGT-15 変更影響 | Rule/Relation/Type変更の必要依存Caseだけを再走 | 同byteでもOperator/Owner撤回、または無関係な説明文変更で誤った証拠再利用/過剰失効 | P2 |
| DGT-16 分布と縮約 | Seed固定の小Fixtureで既知違反を再現し最小化 | 生成をランダムに拡張して再現不能、実DB PIIを平文Snapshotへ記録 | P2 |

## 5. 開発者へ提供する支援の優先順位

### P0 — Headlessな要求導出と安全なデータ候補

既存のTEI inspect/validate/Scenarioに「モデル要素・受理済みConstraint→Coverage Obligation→Positive/Negative fixture Candidate→独立Expected照合」の読取専用ルートを**Pluginまたは既存コマンド拡張として比較**する。新CLIコマンド名/ABIは固定しない。初回は1つの受注Domainで、値域、Key/FD、1:N、Quantity/Amount、NULL/部分取得を検証する。生成の成否・候補の根拠不足を同じ診断形式で示す。

### P1 — 業務フローと横断整合性

Business Procedure/Functionの関連に対してstateful Scenario候補、承認・拒否・取消・手作業待ち、部分Effect/外部ACK、組織権限失効、Cross-DB Read/Commit、並行実行の反例を導出する。**Business Procedureの矢印だけからWorkflow Executorを自動起動しない**。安全なSynthetic/Disposable Providerの実行に限定し、書込み時は既存Host Grant/Commit Fenceを経由する。

### P2 — AI-first高度支援

開発者に「未到達のモデル義務」「矛盾する制約の最小集合候補」「独立Expectedが弱いケース」「Known Mutantに生き残られたCase」「影響を受けた機能・業務手順・文書・テスト」を、出典と不足情報付きで提示する。AIによる追加Case探索、境界のshrinking、モデル更新案、修正PR候補の生成を進める。ただしAIにOwner Decisionや秘密データの開示・本番Effectを委任しない。

## 6. 最初の実証の判定表とOwner引渡し

| Gate | 独立して確認すること | Owner |
| --- | --- | --- |
| G0 Source | Required ConstraintをSource/Revision/受理区分で確認。存在/未読込を区別 | #2495/#1126/#1131 |
| G1 Candidate | 同一Seedで再生成可能、Value Domain/Key/Ruleの未対応式を拒否・説明 | #1131/#2811 |
| G2 Expected | 正負例のExpectedがActualやCandidate生成器から独立。Known Mutantの判別力あり | #2514 |
| G3 Execution | 既存Scenario全必須Caseが実発見・実Run・非SKIPで別Attemptを識別 | #1180/#1163 |
| G4 Realization | Query/Provider/State Transition/Consumerの実Read/Effect Witnessが同一対象へ束縛 | #2586/#2506/#1125 |
| G5 Meaning/Authority | Owner受理と現在Host Grantが独立に適用され、秘匿情報や本番作用を越境しない | #267/#2516/#1174 |

**初回受入の最低条件**：受理済みValue Domain、Key、Relation、Ruleを1件ずつ持つ小Domainから、正常1件・独立して固定された異なる違反2件をseed付きで生成。生成器で直接見つかる同じ誤りだけを採点せず、既存Reference Validatorと実Product/Providerを別に照合し、1つ既知の誤実装を既存Scenarioが検出する。失敗時の修復提案は元Model SourceとAffected Function/Businessまでたどれること。各Gateの未達を個別に示す。

**未達**：本書のDGT-01〜16、P0〜P2、実AI生成・ソルバー・ビジネス全域E2E・正式Owner受理はいずれも**NOT RUN／未採用**。PR #3238の実PG限定Read成功はDGT-07/08の参照根拠であり、この新Generatorの成功実績ではない。