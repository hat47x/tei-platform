# 測定値の集計軸・粒度・時間・再集計適合：Function Data Contextの横断圧力試験

- **状態**：2026-11-02／設計と合成Reference。**意味契約採用・TEI製品実装・実Provider/Target受入は未**。
- **担当**：[#2593](https://github.com/hat47x/tei-platform/issues/2593)。
- **文脈**：[#2506](https://github.com/hat47x/tei-platform/issues/2506)のData Context／[#2507](https://github.com/hat47x/tei-platform/issues/2507)の機能適合／[#2586](https://github.com/hat47x/tei-platform/issues/2586)の同一Fixture統合／[#2582](https://github.com/hat47x/tei-platform/issues/2582)の再評価。横断設計Draft [#2503](https://github.com/hat47x/tei-platform/pull/2503)の追加圧力として参照する。
- **狙い**：Data Contextが「何行取得したか」の正しさを超えて、**測定値を別の軸・時点・集合へ集約し直してよい条件を説明できる**ことを、実証によって確認する。包括的BI/OLAP実行器の開発と同義ではない。

## 1. 背景――同じ数値・集計行でも、何を集約したかが違う

受注Hに金額100のLineがL1/L2の2件、独立した配送実績が3件あれば、Lineの業務合計は200。Line×配送の結合結果6行を単純加算すると600となる。金額の値だけを`DISTINCT`しても100であり、正しいLine同一性は保たれない。

この基本的な集合・JOINの境界には、既存[#1706](https://github.com/hat47x/tei-platform/issues/1706)の**完了済み合成Reference**がある。ただし、次のような問題は別の設計圧力になる。

- 平均の平均：件数の異なる二つのグループ平均を単純平均すると全体平均と異なる。
- 割合の平均：部門別の割合を平均しても全体の割合と一致しない。
- 重複ありの人数：部門ごとに一意な人数でも部門横断の人数とは違う。
- 在庫残高：同じ日時の倉庫・品目別数量は集約できる場合があるが、異なる日の残高を加算して「現在在庫」とはできない。
- 中央値：各グループの中央値だけでは全体中央値を一般に復元できない。

したがって「関係集合の粒度」と「数値の型」だけでは、Patternに必要なMeasureの再集計可能性を表し切れない。

## 2. 設計の分担

| 意味上の問い | 既存Owner | 本検討で増える質問 |
| --- | --- | --- |
| この実体／関係／結果行は何か | #1126／#1706／#2506 | 同じValueの複数occurrenceが同じFactか |
| この演算の結果は正しいか | #988／#1705／#1640／#2544 | 指定された集計軸に対してこの演算を適用できるか |
| このQueryは何に依存するか | #2387 | JOIN/Group後の出力がどのBase grainのMeasureに属するか |
| これを表示・選択・更新できるか | #2507／#2497／#2500 | 集計行に元Entity更新の一意な逆対応があるか |
| 後続の変化で結果が維持されるか | #2582／#1708 | 更新・時点移動により過去の集計結果を撤回するか |
| 正しい期待値と実行結果を区別できるか | #2514／#2586／#1180 | 再集計演算の判別力とTarget依存差を検証できるか |

**Measureは別の正本にはしない**。Accepted Semantic・既存Rule/Operator・Query Sourceと結び付く参照および適合条件をまず比較する。既存の意味型・Field/Relation/Operatorで表現できないことが立証された場合にのみ、Core／共有Contractへの追加を議論する。

## 3. 必要な情報の見取り図（正式API/DSLではない）

`Measure Applicability = F(baseGrain, factIdentity, aggregationOperator, axes, time, scope, completeness, unit, nullPolicy, sourceBinding, targetEvidence)`

| 項目 | 質問・具体例 | 欠けた場合 |
| --- | --- | --- |
| Base Grain | Line 1件／Account×Date 1件／Event 1件 | 同じ金額のJOIN複製を別実体扱い |
| Fact Identity | 明細ID、Relation occurrence、Group ID、Path occurrence | `DISTINCT value`による誤った重複除去 |
| Aggregation axes | 商品、組織、時点、期間、顧客、Graph path | 加算可能な次元の範囲を過大に推定 |
| Operator Binding | sum、weighted mean、ratio、countDistinct、closing balance | 演算名だけで可換性・分配性を誤推定 |
| Intermediate State | sum+count、numerator+denominator、ID集合、Sketch | 局所集計からの復元に必要な情報が消失 |
| Temporal Binding | 観測時点、業務有効時点、期間flow、時点stock | 在庫や所属人数を日跨ぎで二重計上 |
| Coverage / Scope | 対象集合、Filter、権限内範囲、ページ、Snapshot | 部分平均を全体平均と宣言 |
| Value Constraints | 単位、通貨、Decimal、NULL、欠落、桁・丸め | 無単位のSUM、NULL=0、精度差 |
| Target Evidence | SQL方言、Provider、演算実装、実Artifact版 | 論理上適合するが実Target非対応を見落とす |

### 3.1 集計可能性は「Measure×軸×演算」の条件

**完全加法的**：宣言した次元で合計できる。元Factが重複なく分割され、対象時点／単位／換算／Scope等が揃うことを別途要する。

**半加法的**：軸によって可能性が異なる。例えば指定日時の口座残高は複数口座間で加算できるが、日を跨いで単純加算した結果は当日の残高ではない。

**非加法的**：割合・平均・分位点などは、その値だけで総合結果を得る一般的な加算演算を持たない。元に戻して算出するか、意味に適合する中間状態を保持する。

これらは設計上の**質問の分類**であり、Measureを型階層に固定する提案ではない。集計可能性は対象となる具体的な演算、次元、Conditionに依存する。

### 3.2 再集計の条件を明示する

- **SUMの再集計**：元Fact集合の互いに重ならない分割、同一Unit/時点/範囲・適切なNULL/丸め規則。JOINがFactを増殖させる場合はFact identity・適合したQuery方法を再確認する。
- **Averageの再集計**：`sum`と分母となる`count`が同一のNULLポリシーと範囲を持つとき、`(sum1+sum2)/(count1+count2)`。
- **Ratioの再集計**：定義された分子・分母それぞれが同じ対象範囲と時点で再集計可能な場合、`(n1+n2)/(d1+d2)`。分母0・欠落・Unitの定義は別契約。
- **Exact distinctの再集計**：ID集合のunion、またはグループ間の非重複性の証拠が必要。部分集計`countDistinct`の数値だけから一般に再計算できない。
- **中央値など**：原Factまたは要求精度を満たす中間表現・専門Providerが必要。近似Sketchなら誤差・信頼区間と関係するTarget条件を含める。
- **Stock量**：業務上意味のある`as-of`で同一時点へ整合する。期間Flowやイベント積分との変換はRule/専門Operatorが明示された場合に行う。

### 3.3 組織と時間・状態モデルへの一般化

所属者数は時点・兼務・所属Roleと集計対象の意味により異なる。ある部門に所属する人数の和と、組織全体で重複を除いた人物数は異なる。同一の人物が2つの経路で現れる場合、経路出現を計測するか人物実体を計測するかは用途による。

時系列の計算モデルについては、イベント/周期/継続状態、Clock/Step/整合時点を別の契約として扱う。物理量の時間積分、最後に観測された値、時間区間内の平均を同じ`SUM`へ固定しない。実際のSolver、Scheduler、物理シミュレーションは専門Plugin／外部Targetの責務に分配する。

## 4. 正負一対の自作合成Fixture

| Case | 独立した正しい期待 | 同じ値から生じる誤推定 | 必要な根拠 |
| --- | --- | --- | --- |
| MC-01 JOIN | 100+100=200 | JOIN後600、`SUM(DISTINCT value)=100` | Line IDとJOIN occurrence、対象集合 |
| MC-02 AVG | A=2人×10、B=8人×20 → 18 | (10+20)/2=15 | sum+count、NULL分母方針 |
| MC-03 Ratio | A=1/2、B=8/10 → 9/12=0.75 | (0.5+0.8)/2=0.65 | 対応する分子と分母 |
| MC-04 Distinct | A={U1,U2},B={U2,U3} → 3人 | 2+2=4人 | Person ID集合と母集団 |
| MC-05 Stock | 日1=A100+B200、日2=A110+B190 → 日2=300 | 日1と日2の単純加算600 | as-ofと対象口座集合 |
| MC-06 Empty/NULL | 空は0行、NULLのみは1行 | `SUM(x)`はいずれもSQL NULL | `COUNT(*)`、値状態の扱い |

本PRの`plan/fixtures/metric-aggregation/verify_sqlite.py`は**Python標準ライブラリのSQLite in-memory**を使い、これらの合成例について実行値と独立したPythonの手計算相当値を突き合わせる。SQLの出力をExpectedとして再利用しない。TEIのSQL解析器／生成物／実DBの実行確認は含まない。

## 5. Patternへの接続：ReaderとAggregatorを分けて適合診断

初期構成のRead-onlyな処理の流れ：

```text
Accepted Semantic / Measure-related Rule
    │    grain, identity, dimension, unit, timing
    ▼
Query / Context Result Slice ───> source scope, completeness, lineage
    │
    ▼
Pattern requirement + requested group axes + operator
    │    preserved? / need original facts? / unsupported? / unknown?
    ▼
Chart / Table / Drilldown / Metric projection
    │
    └─ ViewChange → Intent candidate (#2500)
         → Host-owned Execution/Receipt/Observation (#1174/#2485)
```

分析Patternは、測定値の**読取り**を提示できても、集計された結果行が業務上の**書込み対象**となる条件を満たすとは限らない。更新を要する場合は元Entityを指定する操作、または受理された専用業務Actionを利用する。

**AI用診断**：単なる`pass/fail`より、`applicable`／`requires disaggregation`／`requires denominator`／`requires time alignment`／`lossy`／`unknown`のような利用者に説明できる分類を比較する。名称・JSON構造は正式採用前。

## 6. 再評価：計算可能なClaimと現在のClaimは別

初回の正しい集計でも、SourceやRuleが改版、Factが訂正、Scopeが変更、Provider通知が遅延すれば現在値としての資格が変わる。#2582のClaim依存・失効・差分更新に対し、**集計済み中間状態が増分で正しく維持できる条件**を渡す。

例えばAVGは`(sum,count)`を保持すれば挿入・削除を算術上反映できるが、対象が正しく同定され、差分に対応する旧値・新値と時点が得られることが前提。Distinct人数は追加と削除を扱う場合、単一Counterだけでは一般に足りない。これらは**Providerが適切な変更証拠を供給する場合の設計候補**であり、TEIにCDC／Incremental View Maintenanceが既にあるという主張ではない。

## 7. 外部製品・設計との照合

| 出典事実・文献 | TEIが検証する設計上の対応（採用未決） |
| --- | --- |
| [Kimball Group：加法・半加法・非加法Fact](https://www.kimballgroup.com/data-warehouse-business-intelligence-resources/kimball-techniques/dimensional-modeling-techniques/additive-semi-additive-non-additive-fact/) | 集計可能性が軸・用途に相対的であること |
| [dbt MetricFlow：EntityとJoin](https://docs.getdbt.com/docs/build/join-logic) | Entityと関係経路を用い、fanout/chasm結合を検査する |
| [dbt：Semantic Models](https://docs.getdbt.com/docs/build/semantic-models) | Entity/Dimension/Measureの意味参照とQueryでの利用 |
| [Google Looker：JOINと対称集計](https://docs.cloud.google.com/looker/docs/working-with-joins) | 値の重複ではなく正しいEntityキー・方言の能力が必要 |
| [dbt：累積Metric](https://docs.getdbt.com/docs/build/cumulative) | 集計対象の時間幅・期間軸・必要な時系列構造 |
| [Cube：Pre-aggregation](https://docs.cube.dev/reference/data-modeling/cube) | 物理的な事前集計と意味上の集計要求を分離 |

外部資料の商用仕様をTEI実装済み機能として扱わない。各方式の適合性はProvider・SQL方言・TargetとModel版ごとに検証する。既存参照モデル由来の文面・固有構造を転載せず、自作合成Fixtureを使う。

## 8. 実証計画・測定

1. **P0**：MC-01〜06の合成SQLite独立Oracleと反例を固定。#1706の既存Referenceと比較し、合成集合正しさと再集計適合の追加差分を明示。
2. **P1**：#2506のData Context／#2507のPattern適合票からMeasure要求を読み取るread-only説明。元Query/Contextから必要情報が欠落すれば`unknown`を保つ。
3. **P2**：#2586の一貫したSource/Context/Grain/Claimの縦断Scenarioに接続。#2387/SQL Workerの実解析入力と明示ID/Timeを照合。
4. **P3**：#2582のSource変更／削除／遅延入力で局所集計の撤回・再計算を比較。先行して単純再取得を参照方法にし、Provider差分演算は独立評価。
5. **P4**：2種以上の異質Consumer／外部Hostで意味保持・適合診断・同一Contract再利用・新Core種類数・Adapter量を計測し、#2018へ結果を返す。

**反証条件**：グループ平均の平均を通す、異日残高を加算可能と誤判定、部門別Distinct人数を和にする、集計結果に自動更新権限を付ける、隠れた集計母集団を全件確定と扱う、未確認Provider実行を正式適合とする、のいずれも誤成功として検出すべき。

**設計判断待ち**：既存Semantic Field/Rule/OperatorのどこへMeasureの軸依存適合を表現するか、必要な中間状態をData Context側で表すかPlugin結果で維持するか、式への自動書換えと正式に提示する診断の範囲。Core新型・新Interpreter・再集計エンジンの採用は、最初の異質Consumer比較後に判断する。
