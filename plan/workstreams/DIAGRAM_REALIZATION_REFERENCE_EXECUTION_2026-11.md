# 図表から解釈実行・生成・委譲・Hook組込みへ進む実証計画

- **状態**：2026-11-15の引渡し計画。実装・採用・Owner意味受理・実Desktop受領を示さない。
- **設計正本**：[図表の実現方式とHook境界](../design/DIAGRAM_TO_REALIZATION_STRATEGY_AND_HOOKS_2026-11.md)。本書は作業順と証拠の引渡しに限定し、第二のGenerator/Runtime/Build仕様を作らない。
- **優先順位**：TEI Studio DesktopのStage1→Stage2セルフビルドP0を遅らせず、既存`*.tei.md` Ruleの局所試験から並行して進める。

## 検証する中心仮説

一つのAccepted Meaningから、(I)既存標準Evaluatorによる解釈、(G)Target-native生成、(D)DB／外部Providerへの委譲、(H)混合の候補を比較できる。図表の詳細化で見つかった業務上の制約はMeaningへ、実現方法だけの選択はTarget側へ、反復する判断はPattern／Skillへ収められる。生成ソースを直接編集せず、明示HookとBuildへのBindingで実Consumerへ接続できる。

## DRI-G0：生成AIへ渡す前の文脈限定

[実現範囲を定めるProduct Skill](../../skills/yi-tei-realization-scope/SKILL.md)を、生成器候補やHook実装の依頼に使う場合の**作業前確認**として位置づける。Skill自体はMeaning受理・実行資格・製品Generatorではなく、候補を作る際の判断を再利用する手順である。

- 対象を**一つのCanonical Subject／一つの選択Target／一つの実装義務**に絞り、実Readerが受理するRoleとSource版を固定する。
- Required Contract、入力・型・NULL・粒度・時点・Read Scope、Failure・Effectを確かめ、未確定な値は未確定としてOwnerへ返す。
- I／G／D／Hを「適合性→実装の実在→費用」で比較し、生成に必要なTarget、Source、独立Expected、許可された非生成ファイル、Hook ID／版を明示する。
- 生成AIが作るのは限定された**候補**であり、生成物への直書き・不可逆な外部作用・未受理意味の補完を認めない。合格を判定するのは元から独立したExpectedと実Builder／Consumerである。

**負例**：同名Nodeが二つある、SourceのDefinition Roleが不明、Target Packageの書込み権限がない、HookのEffect／順序が未確定、既存Providerが複数で選択不能、独立Expectedが生成出力から逆算されている。これらは任意のTemplateで解消したことにせず、read-onlyまたは`unsupported`で止める。

これは既存DRI-0/1の入口を具体化したものであり、新しいHard GateやRun Engineを追加しない。Stage1→Stage2のP0は継続する。

### 定型自動生成の適格性を判定する

実装を急いで大きなGeneratorへ渡すのではなく、**対象を定型的な変換へ閉じ込められるか**を先に判定する。この判定は設計上の選抜条件であり、新しい製品Hard Gateや第2のRegistryを意味しない。判定票の正本は[設計§10](../design/DIAGRAM_TO_REALIZATION_STRATEGY_AND_HOOKS_2026-11.md#10-生成する文脈を絞り生成対象をボイラープレートに近づける)と既存の独立Expected、Generator適合記録に置く。

| 対象 | 文脈限定に必要な作業 | 判定 |
| --- | --- | --- |
| **Source/Meaning** | 選択済みReaderが読むRoleとCanonical ID、入力型、Read範囲、NULL／欠落、業務効果、必要なScopeを確定 | 不足があれば意味・型Ownerへ返す。AIが補完しない |
| **実現方法** | 標準Function／生成／混合／DB・外部Provider委譲の適合候補を照合 | すでにFunctionで十分なら重複する生成器を作らない |
| **対象コード** | 生成物のOwner、Target・Toolchain、許容Import、出力シンボル、配置、接続Hookを固定 | 任意の既存ファイルを書き換えるタスクをGeneratorに渡さない |
| **検収** | 独立Expectedと負例を先に固定。Source配置・型・Rule参照先の一因子変化、直接編集／二重Hook等を加える | 失敗する条件を検出できない限り、自動生成の適格性を主張しない |
| **再現と実使用** | 固定したGenerator Sourceで再実行し、生成物byte、実Builder消費、実Provider／Consumerを別々に観測 | AIが同じ回答を繰り返すという主観的期待に依存しない |

### 最初に比較する限定タスク

**適格候補**：選択済みの `quantity_positive` の型付きRule、既存Go Generator、既存Evaluator／Adapter、指定Hookのない既定構成。Rule自体の業務判断をAIに創作させず、既存Patternに従ったTarget Adapter／Binding／生成Goの接続を確認する。

**対照負例**：同じRuleの説明から、関連表の更新、通知、再送時の回復、UI権限まで一度に実装するタスク。必要なEffectやGrantが確定していないため、AIに作らせて自動採用しない。

**異質な2番目の対象**：同名ノード・多重有向辺のGraph。選択Reader／Graph Modelの意味IDと描画IDを分け、辺の向き・多重度・並びを変えて必要なロジックが変わるかを検証する。現状の限定Readerを正式な万能Parserと扱わない。

### 完了条件と計測

- 文脈の縮小によって、修正対象ファイル・入力規則・許可Hookが定まること。
- 独立Expectedで重大な反例を拒否し、定型Generatorの再生成が同じ契約で再現可能であること。
- **「自動生成できた」よりも「自動生成してはいけない範囲を止められた」を先に証明すること**。
- AIが一度Generator／案件Hookを実装する工程と、固定Generatorが毎回決定的に生成する工程を別の工数・適合試験として計上すること。
- 成功率の数値を先に掲げず、対象Pattern／Target／負例母集団と試行記録を揃えた後で評価すること。

**反復する課題の切出しは[限定生成のProduct Skill候補](../../skills/yi-tei-realization-scope/SKILL.md)にまとめ、最初の実証ではSkill本文だけで成功を宣言しない。**

**既存の確認済み基線**：PR #5009 の限定的なRule→生成Go→通常CLI Build、PR #5056 の生成GoとBuilder配置byteの取り違え拒否を再利用する。一連のSource出典・正式Provider／Host・Studio Stage1→Stage2が合格したとは主張しない。

### 次段階へ進む条件

DRI-0の直後に**DRI-G0**を行う。ここで「Iで充分／Gに適格／Hが適格／委譲が必要／unsupported」を記録し、DRI-1の選択票へ渡す。DRI-G0で閉じない題材は生成器の追加開発ではなく、設計の属性発見であるDRI-2または既存Ownerへ差し戻す。閉じた題材だけDRI-3のHook契約、DRI-4の実Builderへ進める。DRI-5のStudioセルフビルドは引き続き独立の優先経路とする。

## Source／Designer／Resource／生成物の実体を照合する

設計文書の[§13：ファイル役割・Build工程・Hookの分離](../design/DIAGRAM_TO_REALIZATION_STRATEGY_AND_HOOKS_2026-11.md#13-source・designer・resource・生成物をファイルの役割として分離する2026-11-15追補)をDRI-1〜5へ重ねる。これはIDEのファイル分割を模倣する新しいWorkspace規約ではなく、**TEIの意味からTarget成果物へ到達する際に、何を誰が編集・生成・消費するかを明示するための検査票**である。

| 役割・工程 | 既存実体を使う最初の対照 | 確かめる一つの差 |
| --- | --- | --- |
| Authoring／Designer | 選択Markdown RuleとStudioのSource／Graph／Inspector | 図のレイアウト・説明・正規Meaningを分け、逆対応のない編集を拒否 |
| Resource／契約Source | `inspection-reader.contract.json`、既存WorkbenchのAsset | リソース原byteと使用側の参照、埋込みの最新版を別に照合 |
| Generator Definition | `builder-sources/inspection-reader-v1.transform.json`、`validationgen.GeneratePositive` | Generator版の変更を業務Meaning改版と同一視しない |
| Generated Source | `inspection-reader.generated.ts`、Ruleの生成Go | 手編集を許さず、生成byteと実Build配置byteを照合 |
| Custom／Hook | `presentation.ts`の明示Re-export、既存Extension Point参照 | 生成部品を編集せずにTargetへ接続。Hook契約不一致・二重登録を拒否 |
| Build Intermediate／Bundle | Workbenchの生成部品→Bundle→埋込み、Rule→Go package→Builder | 生成しただけのファイルと実Compiler／Bundlerの消費を区別 |
| Consumer／Evidence | 通常CLI BinaryとWails実画面 | 生成／収録／実使用／業務Effectを別々に判定 |

最初の追補試験は**FL-02（Hookを保持した再生成）、FL-03（生成コード直接編集・Symbol衝突）、FL-04（生成GoがCompiler入力へ届かない）**を優先し、次に**FL-05（Workbenchの生成TSと実埋込みの不一致）**へ接続する。既存RG／LC／R2-PのIndependent Expectedを再利用し、新しいExpected正本を作らない。

**Goの重要な相違点**：コンパイラがソースを部分クラスとして自動合成するTargetとは違い、Goでの継ぎ目はInterface／明示的Function Binding／Compositionへ置く。通常の`go build`がTEIのGeneratorを自動実行するわけではないため、生成をBuildより前の明示Stepとし、同一package／moduleの実消費byteを検査する。TypeScript／Web UIも同様に、生成TSがあるだけではBundler・Wailsの実使用にならない。

**文脈限定との関係**：DRI-G0のAI依頼には`authoringSource`、`generatorDefinition`、`generatedOutputs`、`editableCustomSources`、`resourceInputs`、`targetBuildInputs`を、既存情報への**参照**として指定する。これらはProduct Skillの作業票上の説明項目であり、新しい固定SchemaやDSL属性名ではない。生成ファイルの場所は実Targetに従い、利用者の業務DSLへ物理パスを必須にしない。

## 作業単位と引渡し

| 段階 | 入力・既存の担当 | 最小の証拠と完了条件 | 結果を渡す先 |
| --- | --- | --- | --- |
| DRI-0：実装基線 | `sales.order_line.quantity_positive`、内蔵Markdown Reader、`validationeval`、`validationgen.GeneratePositive` | Rule ID、型、元byte／Source位置、実Evaluator／生成Goの対象版を固定。現状の未対応を一覧化 | #988/#2621、#1121 |
| DRI-1：実現方式の選択票 | 同じ型付きRuleと選択Target、標準Function候補 | I/G/D/Hの候補・棄却理由・要求保証・Target適格性とCost Profile。LC-01/02・GP-01〜06を再利用 | #1121/#4544、#2514 |
| DRI-2：属性の発見と配置 | 図表の詳細化で出た要件候補 | 意味／保証／Target設定／View／Patternの分類を一因子で確認。構文を勝手に増やさず既存Ownerへ戻す | #2619/#2620、#4359 |
| DRI-3：ノータッチ生成とHook | 既存Extension Point契約、独立ファイルの案件実装 | 生成ファイル無変更、版・順序・入出力・Effectの衝突拒否、Hook有無の差分と失敗を提示 | #2090/#951/#4198 |
| DRI-4：実Build・Consumer | 正式Builder、選択Provider、通常CLI、実Evaluator | IとGが各々実Callされた証拠、Builder実入力、結果・Failure・Read範囲。Dは実SQL／Provider／Scopeを別に確認。生成Goが存在するだけでは不可 | #943/#4495/#2514/#951 |
| DRI-5：Studio・再入場 | 生成Inspector等の既存Reference、正規Bundle／Wails | 埋込byteと実WebView利用を照合。Stage1→Stage2は既存P0の独立判定を再利用 | #1824/#2831/#4583 |
| DRI-6：Pattern／Skill化 | DRI-1〜5で反復した判断と負例 | 少なくとも二つの異なる題材で使えた手順をレシピ化。Coreへ新たな万能型を追加しない | #2605、#988、#4544 |


## 既存の実装試験とセルフビルドに接続する順序

本計画のDRIは新しいGateやExpected Corpusの名前ではない。次の既存成果へ、必要な証拠だけを追加する。

| 順序 | 基線・引渡し元 | 加える最小の反例と証拠 | この段階で主張しないこと |
| --- | --- | --- | --- |
| 1 | [成果物要求のLC-01/02](../design/DECLARATIVE_RESULT_PROCESS_COMPOSITION_2026-11.md)、現行Ruleの解釈／生成比較 | 同一Sourceと独立Expectedで、Evaluator／生成Goの実Input・Source版・実Callを区別。Ref Actorの相違だけでなく、NULL／未取得の扱いを検査 | 全OperatorやDB方言の同値性 |
| 2 | PR #5009・#5056の既存限定基線、[R2-P](../design/IMPLEMENTATION_TO_MODEL_DOGFOOD_2026-11.md) | 選択Rule／Generator出典、生成Go、実Build Intent、Builder消費byteを**同じAttempt**で照合。Stage A→B→Aの再利用否定 | 生成Go単体の実行を正式Provider呼出しへ昇格 |
| 3 | [Hook境界](../design/GENERATOR_CONTRACT_BOUNDARIES_DESIGN_2026-10.md)、既存Extension Point参照 | 新たな生成ファイル内編集を許さず、別ファイル／BindingでHookあり・なし・競合・版不一致・Guard拒否を区別 | Hookの説明をもって、実製品の実行安全性と呼ぶ |
| 4 | 通常CLIの選択ProviderとDB／Queryの既存Contract | Declared→Selected→Built→Invoked→Observedの各段に実IDを記録。DB委譲はSnapshot／Scope・型／丸め・実SQLと実Readbackを独立検査 | 手書き外部Providerへの委譲を意味ロジックのTEI化と数える |
| 5 | [自己置換M0〜M3](../design/IMPLEMENTATION_TO_MODEL_DOGFOOD_2026-11.md)、Studio Stage1→Stage2 P0 | `GeneratePositive`のRequired Shape判断をTEI由来の処理へ移し、旧Go条件の実削減と次世代のTEIによる実使用を別々に検査 | 参照Seedの二度ビルドをセルフビルドと呼ぶ |

**発見→反映の小ループ**：実装試験で生じた属性は、まず意味・要求保証・Target設定・View・Pattern/Skillへ分類し、既存定義に対応するものだけをReader／Source WriterのOwnerへ戻す。受理していない図表属性はExecutor・Generatorへ流さない。採用された変更を改めて実Readし、再生成または再解釈、実Consumerへの引渡しを確認する。

**切り上げ条件**：実Readerと選択Contractが見つからない、型や粒度・Null／Error・Scopeが不明、HookのEffectが未定義、または実Providerが解決できない場合は当該方式を`unsupported`とし、暗黙に別方式を選択しない。設計表示は継続できるが実行合格にはしない。Studio Stage1→Stage2を優先する既存の作業は妨げない。

## 先に固定する反例

- Iで読み取ったParameterの版が変わったのに、旧Invocationが同一実行中に新設定を使う。
- Gで正しく生成したが、選択Providerと違う実装を呼ぶ。I/Gの結果が同じでも合格にしない。
- Generatorが作ったファイルを人／AIが編集し、次の生成で黙って上書きする。
- 宣言したHookが未登録／契約不一致／二重到達し、Guard前に外部作用を実行する。
- Markdownの説明・図のレイアウト変更を、業務意味と同じ失効として扱う。
- 類似するPatternの説明を追加しただけで、標準Functionが実装・実使用されたと主張する。

**計測**：成果物の行数だけでなく、TEI側へ移せた意味・判断・検証義務、残った手書き責務、再生成時の変更箇所、実Build時間、実行性能、実際に選ばれたProviderを追う。IとGは同じ入力範囲・Targetで比較し、Dは差異が生じるTargetを明記する。生成型・解釈型・外部委譲型が担った実装義務を別集計にする。

## 統合単位

本計画・設計の整合を確認した文書変更は、実装待ちで保留せず独立PRとして扱える。製品実装はDRI-0/1、DRI-2/3、DRI-4、DRI-5の順に分け、既存Ownerを維持する。受入未確認を既存Gateの成功へ昇格せず、`main`の健全性を確認してから統合する。

## 2026-11-16：状態付きRuleと実Readをつなぐ最小の実装パケット

[図表の実現設計§14](../design/DIAGRAM_TO_REALIZATION_STRATEGY_AND_HOOKS_2026-11.md)を、DRI-0／1／4の次の作業へ引き渡す。規則の種類が増えたため、すべてのRuleを同じGeneratorへ通す方針にはしない。まず既存のReader・Evaluatorを製品の正式入口として使える範囲を広げる。

| 順序 | 実行する最小の変更 | 独立Expectedと停止条件 | 引渡し |
| --- | --- | --- | --- |
| **DRI-R1** 現行基線 | mainの#5142（値の状態）、#5143（記録例）、#5146（日付の順序）を、対応するRule ID・Source位置・Reader版とともに確認 | present／null／not-fetched、complete／partial、成立／不成立／未決定を混同しない。mainに入った#5148（期間非重複）は評価器・記録例まで。実DBの全件性／正式Consumerの資格には算入しない | #988／#2621／#4359 |
| **DRI-R2** 読取Adapter | 一つの既存Query結果について、必要Attribute／RelationのRead ID、Scope、PresenceとCoverageの根拠を示す。最初はread-only Packetとし、何を観測できないか明示する | NULL格納値・外部結合で生じたNULL・未取得、取得した行の粒度と全件保証を区別。Sourceと実DBの結線が不明ならunsupported | #2586／#2622／#2506 |
| **DRI-R3** Ruleへ渡す | 取得資格を確認した値だけを既存の状態付きEvaluatorへ渡し、同じ値でもRead範囲が異なる負例を試す | 見えている確定重複による不成立と、未読行があるため成立と確定できない場合を分ける。FixtureだけのPASSは実Provider接続とはしない | #4359／#1180／#2514 |
| **DRI-R4** I／G／Dの選択 | 同じRule kindと同じ状態付き入出力がTargetで実現可能な場合だけP（既存評価）とG（生成）を比較し、DのProviderの実Callを別に確かめる | positive以外のG未対応を、一般生成済みと見せない。結果が一致してもRead先・実装選択・Builder入力が違えば不合格 | #1121／#4544／#951／#4495 |
| **DRI-R5** 実Consumerと自己再入 | DRI-R3／4で確定したClaimだけを通常CLI／独立Consumerへ渡し、Builder／Toolchain／Artifactの版を照合する | BuildPlan・生成Go・元Markdown、選択実装A→B→Aの旧資格、実Consumer未受領を分ける。Windows Studio Stage1→Stage2は独立P0 | #2586／#2600／#2831／#4583 |

**手元にある証拠の使い方**：#5118の実Binary比較はpositive Ruleの結果とRead IDまで、#5142／#5143／#5146は対応するEvaluator／記録例まで。Function Contextでnative SQLのCoverageがcompleteと確定した証拠はまだない。#5144の列由来調査はクローズ・未マージのため製品の取得資格として利用しない。#5148は期間重複の設計・コードの検討が進む間、実Provider・業務Ownerの受理とは切り離す。

**優先度**：まずDRI-R2の実Readerに不足する情報を一つ特定する。値とScopeの受渡しに不足が残れば、その欠落だけを既存Driver／Function ContextのOwnerへ返す。追加の新規Core型・常設Trace・万能Adapterを先に作らない。MVPでの一条件移管と通常CLI実Provider／DesktopセルフビルドのP0を妨げず、既存のIndependent Expectedを再利用する。

本追補は実装への引渡しであり、新Go・実PostgreSQL・Windows・正式Provider・Owner受理を追加検証した結果ではない。

## 2026-11-16：DRI-R2の実証対象を「QueryからRuleへの一回の引渡し」に固定する

[図表実現設計§15](../design/DIAGRAM_TO_REALIZATION_STRATEGY_AND_HOOKS_2026-11.md)では、実Query・状態付きReader・Ruleを**同じAttempt**で検収するための最小条件を整理した。これを既存DRI-R2〜R4の次の一試行として扱い、別の大きな実装計画を増やさない。

**先に利用する基線**は、#5155／#5165で確認された組織C1の型付きSource／Queryと実PostgreSQL Readの限定PASS、main済み#5142／#5143／#5146の状態付き評価・記録例・日付Rule、main済み#5166の「日付RuleはP評価可能、現行Gでは未対応」という正負対照である。#5149のContract Query行Readerはmainに統合された。SQLiteのSQL Executor→Result Mapping→状態付きRuleの局所試験は存在する。ただし正式なQuery Binding／Environment／実PG／製品CLIの経路ではなく、関係の全件性は呼出し側の申告に留まる。#5148の期間非重複もmainに入ったが、正式Consumerまでの実証ではない。

| 作業の区切り | まず得る一つの証拠 | 未充足なら止める位置 |
| --- | --- | --- |
| **R2-a：対象とRead要求** | 選択したRule ID、Required Attribute／Relation、Query ID・入力・Scope、Expectedを独立に固定 | 同名要素の別Subject、Source改版、必要な読取集合が不明 |
| **R2-b：実Queryと状態の接続** | 同じAttemptの実Provider Query・Read ID・Observed列と行・Presence。completeと記すなら必要集合・Scope・Snapshotを満たす別証拠 | 列名一致だけ、呼出し側の全件申告だけ、RowDescription未観測、開示資格不明 |
| **R3-a：既存Evaluatorへ引渡し** | 既存の状態付き判定で、既知重複partialは不成立・重複なしpartialは未決定。NULL／未取得を別扱い | Fixtureが答えを返しただけ、未読が値なしへ変換された |
| **R4-a：選択方式とCaller** | Pで評価した実RuleとRead ID、Gが対象kindを生成できるか、選択した実装の実Callを別記録 | 未対応Gから自動Fallback、異なるAttributeを読んで結果だけ一致 |
| **R4-b：Consumerと閉包** | 通常CLIの実Entry→Provider→独立Consumerの受領。生成する場合はSource／Plan／Builder実入力を別照合 | Contract Queryの単体PASSを製品R3やWindows Stage2の受領に読み替える |

**最初の試行は一つのRuleと一つのContract Queryに限定する。** 変更する要因も、値の状態、必要集合のCoverage、Read ID、実現方式のいずれか一つずつとする。Read Provenanceが確定できない場合は未証明（not-proven）を残し、実際に取得していない項目を補完しない。Ruleで必要なScopeとDBの実読取Scopeが一致するかも別に確かめる。既存#1180／#2514のExpectedへ引き渡し、追加の常設Trace・Scenario Registryは作らない。

この作業は#2586の通常CLI実Provider・#2831の非Plugin Windowsセルフビルドを遅らせる前提Gateではない。今回は引渡し計画の追加のみで、新Go試験・実DB・Windows・Owner受理を行っていない。

## 2026-11-17：次の一試行をCoverage資格の検証に絞る

前節のR2-bは、[#5149](https://github.com/hat47x/tei-platform/pull/5149)の統合により「SQLの実行結果を状態付きRuleへ渡す局所経路」まで進んだ。[実装上の限界と次の反例](../design/DIAGRAM_TO_REALIZATION_STRATEGY_AND_HOOKS_2026-11.md#16-sqliteの局所到達から正式queryruleへ移すための実証境界2026-11-17)に沿い、残る不足を**正式なBinding／Scope／Snapshot／Coverageの根拠**に限定する。

| 次に行うこと | 固定する独立Expected | 完了と呼べない場合 |
| --- | --- | --- |
| **R2-b1：全件性の根拠**。既存`TestAPageIsNotTheWholeRelation`を土台に、ページをcompleteと誤申告した経路と、申告に根拠がある経路を区別する | 同じ二行でもpartialなら一意性成立は保留。既読の確定重複はpartialでも不成立。Queryのlimit・filter・Scopeと必要集合を併記 | 呼出し側のcompleteをそのままTrusted Evidenceへ昇格した場合 |
| **R2-b2：意味とQueryの同一試行束縛**。選択言語とSource／Rule IDを、Contract QueryのInput／実Binding／Row Mappingへ結ぶ | 同じ出力でも別Relation／別Attribute／別Source版を拒否。#5184の同じMembership投影から異なるPerson数を導ける反例を再利用 | 型付きのRowが返っただけ、元Sourceを再Loadできない場合 |
| **R3-a1：実Callと受領**。状態付き評価を通常CLI側の実Readと独立Consumerへ接続 | 取得値・判定理由・実Read対象とConsumer応答が一つのAttemptに属する。未取得や権限不足は推測で埋めない | SQLiteのユニット試験と別試行のCLI結果を合成した場合 |

`validationread`を新しいReaderで置換しない。現行の`FromHeaderDetail`／`Decide`を使い、必要な資格だけを既存Driver・Function Contextへ返す。#5166の暦日Rule（P可／G未対応）を維持し、未対応なGを暗黙のPへ切り替えない。#5205で改善された`*.tei.md`の言語選択診断はSourceの入口の基線であり、実Providerを呼んだ証拠ではない。読取りコマンドの前提指紋統一[#5198](https://github.com/hat47x/tei-platform/pull/5198)はmainの基線として再利用する。ただしそのSnapshot拒否をQueryの全件性・実Providerの受領に読み替えない。

正式R2-PのBuilder消費、R3の通常CLI実Provider／独立Consumer、Windows Studio Stage1→Stage2は独立P0とする。この追補自体はGo・PostgreSQL・Windowsの新規検証ではなく、既存実装から次に不足する一点への作業引渡しである。
