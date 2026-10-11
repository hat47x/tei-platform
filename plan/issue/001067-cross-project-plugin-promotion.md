# ISSUE-001067: 案件固有の開発機構を共有プラグインへ昇格する条件

- 状態: Trigger Waiting
- 起票日: 2026-10-09
- GitHub Issue: #1067
- 検討PR: #1069
- 関連: #951 / #979 / #988 / #997 / #998 / #1016 / #1063

## 背景

TEIでは、案件ごとに小さなDSL拡張、生成器、構成プラグインを作り、別案件で同じ関心事が現れたときに共通部分を取り出す開発方式を重視する。

生成AIを使えば、複数リポジトリの仕様、実装、試験を比較し、共通部分の候補を以前より低いコストで見つけられる。一方、候補が見つかったことと、共有プラグインへ昇格する価値があることは同じではない。

早い段階で共通化すると、案件固有の意味を共有側へ押し込んだり、使われない抽象層を保守したりする危険がある。そこで本Issueでは、共通化を促進する仕組みと同時に、**共通化しない判断を安全に行える条件**を明らかにする。

## 現時点の結論

共有プラグインへの昇格は、コードの類似や案件数だけでは開始しない。

次の三条件がそろった時点で初めて、実装を伴う昇格作業へ進む。

1. 同じ派生機構を実際に必要とする、独立したConsumerが少なくとも二つある。
2. 入出力、失敗、Effect、Authority、順序、欠損値、適用範囲まで比較して、同じ意味契約であると確認できる。
3. 共有側の固定費を含めても、変更量、試験量、レビュー負担、保守箇所のいずれかが実測で減る。

この条件がそろわない間は、案件内の小さな重複を許容する。

生成AIによって後から共通部分を再発見しやすくなるため、早すぎる抽象化を避けるコストは従来より小さい。

## 2026-10-09〜10 の検討結果

### 合成二案件での共通化候補

受注と請求を模した二つの小さな生成処理から、次の二規則を共有する候補を作った。

- 必須文字列
- 非負の安全な整数

承認条件は案件ごとに異なるため共有しなかった。

隔離したNode.js環境では、抽出前後の回帰試験15件、案件ごとの採用・切戻し試験7件が通った。これは合成Referenceの成立を示すだけであり、実案件への導入やRepository全体の適合を示すEvidenceではない。

### 実リポジトリ二件の比較

SUIの文書取込検証とMMIの候補選択検証を、固定revisionの実ソースで比較した。

両者は「入力を検査する」という表面上の類似を持つが、次が異なった。

- 対象とする意味
- 正規化の許容範囲
- 時刻依存
- 診断の用途
- 採用を決めるAuthority

このため、共通Validationプラグインへの昇格を見送った。

この負例は、類似した名前や処理形状だけでは共有化しないための重要なReferenceとなった。

### 採用・版固定・切戻し

案件ごとの採用状態を新しい専用機構で管理する必要はない。

TEIには既に次の機構がある。

- EnvironmentごとのPlugin要求
- Runtime Plugin Lock
- BuildPlan
- Build Artifact Lock
- BuildPlan fingerprintとartifact digestの照合

共有候補が成立した場合も、これらを使って案件ごとに明示的に採用し、版を固定し、旧版へ戻せる。

#1067専用の採用LockやResolverは作らない。

### Capabilityとの境界

外部read/writeやRuntime Authorityを必要としない純粋変換を、汎用Capability Invocationへ押し込まない。

Capabilityは、外部schema取得、Database introspection、remote service参照など、実行Authorityを伴う処理を明示的に分離するために使う。

純粋な派生変換だけなら、既存の決定論的Transformまたは正本の直接利用を優先する。

### Transform拡張の検討と取り下げ

一時は、二つの項目参照から固定長の規則配列を作るために、`construct-array(values: Expr[])`相当のTransform演算を候補とした。

その後、実際の受注Consumerを確認すると、Applicationが選んだ業務検証規則を既存の`driver.ResolveApplicationValidations`が解決し、`validationeval.Evaluate`が正本の`semantic.Validation`を直接評価していた。

First Web Targetも同じ規則を利用しており、別のvalidation planを必要とする具体的Consumerは無かった。

したがって、#1067を理由とする`construct-array`追加要求は取り下げた。

新しいprimitiveが不適切という判断ではない。**具体的Consumerが無い段階では追加しない**という判断である。

### 共通化の費用対効果

合成二案件の生成関数だけを非空行で数えると、次の結果になった。

```text
抽出前
  9 + 9 = 18行

共有化後
  6 + 6 + 共有生成器19 = 31行
```

二案件だけでは約72%増えた。

重複したrule宣言は減ったが、共有側の契約検査と抽象化の固定費が上回った。

現在の小さなfixtureをそのまま一般化するものではないが、「二案件で重複した」という事実だけでは共有化の採算が成立しないことを示す。

## 昇格を開始する手順

具体的なConsumerが現れたら、次の順で進める。

### 1. 比較対象を固定する

各Consumerについて、少なくとも次を固定する。

- exact repository revision
- 仕様の正本
- 現在の実装
- 受入試験
- 使用するPluginと版
- Sourceの所有者
- 越境してはいけない情報

比較中に対象が動いた場合は、同一比較として扱わない。

### 2. 意味を比較する

コード差分より先に、次を比較する。

- 入力と出力
- 型と多重度
- NULL、欠損、空値
- 順序
- 失敗と診断
- Effect
- 認証・認可
- Authority
- Target
- 適用範囲
- 時刻依存
- 再現性

同じ意味契約だと説明できなければ、共通化を止める。

### 3. 最小の共通契約を切り出す

最大の共通実装ではなく、安定した最小契約を共有側へ置く。

案件固有のAdapter、Overlay、承認条件、例外処理、UI差分は案件側へ残す。

### 4. 既存機構で足りるか確認する

新しい基盤機構を作る前に、次を再利用できるか確認する。

- 既存Semantic / Driver
- 決定論的Transform
- Capability Contract
- Environment
- Plugin Lock
- BuildPlan
- Build Artifact Lock

具体的Consumerが要求していないprimitive、Capability、Resolver、Lockを先回りして追加しない。

### 5. 独立した期待値で差分検証する

共有実装と期待値を同じ生成器から作らない。

各Consumerが独立して保持する正負例を使い、抽出前後の振る舞いを比較する。

共通機能の契約試験は共有側へ寄せるが、案件固有の受入・接続試験は残す。

### 6. 案件ごとに採用する

共有Pluginが成立しても一括適用しない。

各案件が独立して次を行えるようにする。

- 採用する
- 採用しない
- 版を固定する
- 新版へ上げる
- 旧版へ戻す

別案件の採用判断を自案件へ伝播させない。

### 7. 効果を実測する

少なくとも次を抽出前後で比較する。

- 案件側の定義量
- 共有側の定義量
- 契約試験量
- 案件固有試験量
- 変更ファイル数
- レビュー対象
- 変更時の影響Consumer数
- 切戻し差分

共有側の固定費を隠さない。

## 採用しない案

### 類似コードを見つけた時点で共通化する

採らない。意味が同じとは限らず、二案件程度では固定費が上回る場合がある。

### 純粋変換をCapabilityとして統一する

採らない。外部Authorityを必要としない処理にProvider ResolutionやRuntime Authorityを持ち込まない。

### Consumerが無いTransform primitiveを先に追加する

採らない。必要な表現力は、具体的Consumerの受入条件から増やす。

### 派生した検証計画を第二の正本にする

採らない。既存Driver/Evaluatorが正本を直接使える場合は、その経路を優先する。

## 2026-10-10 追加調査：実Producer二系統の候補（昇格未承認）

### 比較対象を固定した

次の二つは、合成の受注・請求とは違い、別リポジトリで実装されている**独立したProducer**である。

| | MMI | Excel方眼紙Normalizer |
| --- | --- | --- |
| repository | `hat47x/markdown-matrix-injector` | `hat47x/excel-houganshi-normalizer` |
| 比較したmain revision | `1dc3ae9ad546e61a10e6724fcd3f17bdc4b3abd0` | `1be6cdfd4992925d9362fac0756952b60a00c007` |
| 実装 | [`structuredArtifactSai2CandidateSet.ts`](https://github.com/hat47x/markdown-matrix-injector/blob/1dc3ae9ad546e61a10e6724fcd3f17bdc4b3abd0/03_Implement/src/experimental/structuredArtifactSai2CandidateSet.ts) | [`structured_artifact_sai2_candidate_set.py`](https://github.com/hat47x/excel-houganshi-normalizer/blob/1be6cdfd4992925d9362fac0756952b60a00c007/03_Implement/excel_normalizer/research/structured_artifact_sai2_candidate_set.py) |
| 入力 | MMIの`SheetSource`、used rangeと宣言済みmerge | `openpyxl.Worksheet`、セルとmerge |
| 出力 | `hat47x.structured-artifact-sai2-candidate-set/v0` | 同左 |
| generator ID | `sai2-occupancy-blocks-v0` | 同左 |
| 実装言語 | TypeScript | Python |

双方とも、1始まり・両端を含む座標、昇順で番号を付けた候補矩形、`ok`／`generator_ood`、上限32件、`target_fields_supplied=false`・`ranking_performed=false`等の研究境界を持つ。候補から局所特徴量を作る別実装もあり、`hat47x.structured-artifact-sai2-candidate-local-features/v0`と、五つの同じ特徴キーを用いている。

同じJSON schemaやgenerator IDであっても、同一のアルゴリズムを共有ライブラリとして使っている証拠ではない。

### 記録済みの比較Evidence

Excel側に保持される[fit候補矩形の比較記録](https://github.com/hat47x/excel-houganshi-normalizer/blob/1be6cdfd4992925d9362fac0756952b60a00c007/03_Implement/excel_normalizer/tests/fixtures/structured-artifact-sai2-fit-cross-producer-candidate-parity-v0.json)は**10 source families / 16 candidates**の矩形・順序一致を記録する。[model-selection](https://github.com/hat47x/excel-houganshi-normalizer/blob/1be6cdfd4992925d9362fac0756952b60a00c007/03_Implement/excel_normalizer/tests/fixtures/structured-artifact-sai2-model-selection-cross-producer-candidate-parity-v0.json)は4 family、[final holdout](https://github.com/hat47x/excel-houganshi-normalizer/blob/1be6cdfd4992925d9362fac0756952b60a00c007/03_Implement/excel_normalizer/tests/fixtures/structured-artifact-sai2-final-holdout-cross-producer-candidate-parity-v0.json)も4 familyで、保存された判定は双方の候補矩形と順序の一致を示す。

五つの候補局所特徴量についても、[fit 16行](https://github.com/hat47x/excel-houganshi-normalizer/blob/1be6cdfd4992925d9362fac0756952b60a00c007/03_Implement/excel_normalizer/tests/fixtures/structured-artifact-sai2-fit-candidate-local-feature-parity-v0.json)、[model-selection 8行](https://github.com/hat47x/excel-houganshi-normalizer/blob/1be6cdfd4992925d9362fac0756952b60a00c007/03_Implement/excel_normalizer/tests/fixtures/structured-artifact-sai2-model-selection-candidate-local-feature-parity-v0.json)、[final holdout 7行](https://github.com/hat47x/excel-houganshi-normalizer/blob/1be6cdfd4992925d9362fac0756952b60a00c007/03_Implement/excel_normalizer/tests/fixtures/structured-artifact-sai2-final-holdout-candidate-local-feature-parity-v0.json)が、全行で可搬意味の一致を記録している。

**証拠の限界**：以上は既に保存された比較記録を読み取った結果であり、今回二リポジトリのexact-headチェックアウトで独立に再実行した結果ではない。同一の比較JSONが両リポジトリに存在する場合も、それを独立した二回の試験実行とは数えない。

### 共通化前に確認すべき意味の差

1. **走査限界**：MMIの[`detectRegions`](https://github.com/hat47x/markdown-matrix-injector/blob/1dc3ae9ad546e61a10e6724fcd3f17bdc4b3abd0/03_Implement/src/search/regionDetector.ts)には既定で100万cellのscan budgetがあり、行が打ち切られた場合の扱いを持つ。Excel側の候補生成関数は同等の制限を明示していない。
2. **値の観測**：MMIの`SheetSource.cellAt`はExcelの表示結果を表す`text`を含む。Excel側は`Worksheet.cell.value`を文字列化して占有を判定する。数式・書式・空白について、固定fixture外で同じ結果になるとは限らない。
3. **入力境界**：MMIはSheetSourceの`usedRange`を起点にし、Excel側はWorksheetを走査する。非表示の行・列、巨大シート、merge、欠損値を含む境界の互換性がまだ不明。
4. **実行形態**：一方はTypeScript、他方はPythonで動く。共通Go／WASM等への置換には両案件のbuild・runtime依存が増える可能性がある。単にschemaと出力の形が揃うという理由で共通Pluginにしない。

### #1067の判定

| 再開判定 | 現在地 |
| --- | --- |
| 独立した実Producerが二系統存在する | **確認** |
| 両者が同じ派生情報を必要とする | **部分確認**：研究用の候補矩形・局所特徴量を両方が生成する |
| 意味契約全体が一致する | **未確認**：走査予算、入力値、非表示要素、失敗などの境界に差が残る |
| 共有Pluginで実Consumerの負担が減る | **未計測**：言語横断の移植・保守費も含める必要がある |
| 独立採用・版固定・切戻しが成立する | **未実証**：この二リポジトリでTEI Pluginは未採用 |

**次の実証を一つに絞る**：同じfreeze済みワークブックを両方の本来の入力Adapterから処理し、少なくとも巨大なused range、隠し行・列、数式と書式、merge境界、空入力／候補上限超過の負例を加え、Producer別の元期待値と生成結果を照合する。その後、共通化による定義量・テスト・レビュー負担とbuild/runtime負担を比較する。

この段階では「候補を発見した」のであって、「同じ実装に統合すべき」という結論には達していない。**Trigger Waitingを維持し、対象リポジトリの実装やTEI製品コードを変更しない。** 本比較は、共通化の条件を新しい実例へ適用した読み取り専用の一次スクリーニングである。

## 受入条件の現在地

| 条件 | 状態 | 根拠 |
| --- | --- | --- |
| 意味契約と案件固有差分を分ける | 部分確認 | 合成受注・請求で確認。実Consumer二件では未確認 |
| 類似しても意味が異なる例を共有化しない | 確認済み | SUI/MMIの固定revision比較 |
| 共有Plugin採用後に双方の独立回帰試験へ合格 | 未確認 | 実Consumer二件への採用がまだ無い |
| 共通契約試験と案件固有受入試験を分ける | 部分確認 | 合成Referenceのみ |
| 採用判断なしに変更を伝播させない | 機構確認 | 既存Environment / Plugin Lockで表現可能。実案件未確認 |
| A/Bで別々に版固定・切戻しできる | 機構確認 | 合成試験と既存Lock設計。実案件未確認 |
| 越境禁止情報を取り込まない | 設計済み | 実案件横断の正の実証は未実施 |
| 定義量削減と共有側固定費を実測する | 確認済み | 合成二案件では約72%増 |

## 再開条件

本Issueの実装作業は、次を満たす具体例が見つかった時点で再開する。

```text
independent Consumer A
+ independent Consumer B
+ same semantic contract candidate
+ real need for the same derived mechanism
```

再開時は、候補を共有Pluginへ昇格することを前提にしない。

比較の結果、局所実装を維持する方が合理的なら、その判断を正しい結果として扱う。

## 非主張

現時点では、次を達成したとは扱わない。

- 複数実リポジトリからの共有Plugin自動抽出
- 共有Pluginの実案件二件への導入
- 自動採用
- production Capability追加
- production Transform primitive追加
- 実案件での定義量削減
- 実ProviderやRuntime Authorityの成立

本Issueで確定したのは、**共通化を開始する条件、既存機構を再利用する境界、共通化しない停止条件**である。
