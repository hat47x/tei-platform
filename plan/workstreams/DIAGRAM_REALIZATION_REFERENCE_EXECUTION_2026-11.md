# 図表から解釈実行・生成・委譲・Hook組込みへ進む実証計画

- **状態**：2026-11-15の引渡し計画。実装・採用・Owner意味受理・実Desktop受領を示さない。
- **設計正本**：[図表の実現方式とHook境界](../design/DIAGRAM_TO_REALIZATION_STRATEGY_AND_HOOKS_2026-11.md)。本書は作業順と証拠の引渡しに限定し、第二のGenerator/Runtime/Build仕様を作らない。
- **優先順位**：TEI Studio DesktopのStage1→Stage2セルフビルドP0を遅らせず、既存`*.tei.md` Ruleの局所試験から並行して進める。

## 検証する中心仮説

一つのAccepted Meaningから、(I)既存標準Evaluatorによる解釈、(G)Target-native生成、(D)DB／外部Providerへの委譲、(H)混合の候補を比較できる。図表の詳細化で見つかった業務上の制約はMeaningへ、実現方法だけの選択はTarget側へ、反復する判断はPattern／Skillへ収められる。生成ソースを直接編集せず、明示HookとBuildへのBindingで実Consumerへ接続できる。

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


## 自動生成を許す前の文脈限定Gate（DRI-G0）

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

## 進める順序の補正

DRI-0の直後に**DRI-G0**を行う。ここで「Iで充分／Gに適格／Hが適格／委譲が必要／unsupported」を記録し、DRI-1の選択票へ渡す。DRI-G0で閉じない題材は生成器の追加開発ではなく、設計の属性発見であるDRI-2または既存Ownerへ差し戻す。閉じた題材だけDRI-3のHook契約、DRI-4の実Builderへ進める。DRI-5のStudioセルフビルドは引き続き独立の優先経路とする。
