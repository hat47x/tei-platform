# 図表から解釈実行・生成・Hook組込みへ進む実証計画

- **状態**：2026-11-15の引渡し計画。実装・採用・Owner意味受理・実Desktop受領を示さない。
- **設計正本**：[図表の実現方式とHook境界](../design/DIAGRAM_TO_REALIZATION_STRATEGY_AND_HOOKS_2026-11.md)。本書は作業順と証拠の引渡しに限定し、第二のGenerator/Runtime/Build仕様を作らない。
- **優先順位**：TEI Studio DesktopのStage1→Stage2セルフビルドP0を遅らせず、既存`*.tei.md` Ruleの局所試験から並行して進める。

## 検証する中心仮説

一つのAccepted Meaningから、(I)既存標準Evaluatorによる解釈、(G)Target-native生成、(H)混合の候補を比較できる。図表の詳細化で見つかった業務上の制約はMeaningへ、実現方法だけの選択はTarget側へ、反復する判断はPattern／Skillへ収められる。生成ソースを直接編集せず、明示HookとBuildへのBindingで実Consumerへ接続できる。

## 作業単位と引渡し

| 段階 | 入力・既存の担当 | 最小の証拠と完了条件 | 結果を渡す先 |
| --- | --- | --- | --- |
| DRI-0：実装基線 | `sales.order_line.quantity_positive`、内蔵Markdown Reader、`validationeval`、`validationgen.GeneratePositive` | Rule ID、型、元byte／Source位置、実Evaluator／生成Goの対象版を固定。現状の未対応を一覧化 | #988/#2621、#1121 |
| DRI-1：実現方式の選択票 | 同じ型付きRuleと選択Target、標準Function候補 | I/G/Hの候補・棄却理由・要求保証・Cost Profile。欠落/null／失敗／Read Scopeの独立Expected | #1121/#4544、#2514 |
| DRI-2：属性の発見と配置 | 図表の詳細化で出た要件候補 | 意味／保証／Target設定／View／Patternの分類を一因子で確認。構文を勝手に増やさず既存Ownerへ戻す | #2619/#2620、#4359 |
| DRI-3：ノータッチ生成とHook | 既存Extension Point契約、独立ファイルの案件実装 | 生成ファイル無変更、版・順序・入出力・Effectの衝突拒否、Hook有無の差分と失敗を提示 | #2090/#951/#4198 |
| DRI-4：実Build・Consumer | 正式Builder、選択Provider、通常CLI | IとGが各々実Callされた証拠、Builder実入力、結果・Failure・Read回数。生成Goが存在するだけでは不可 | #943/#4495/#2514 |
| DRI-5：Studio・再入場 | 生成Inspector等の既存Reference、正規Bundle／Wails | 埋込byteと実WebView利用を照合。Stage1→Stage2は既存P0の独立判定を再利用 | #1824/#2831/#4583 |
| DRI-6：Pattern／Skill化 | DRI-1〜5で反復した判断と負例 | 少なくとも二つの異なる題材で使えた手順をレシピ化。Coreへ新たな万能型を追加しない | #2605、#988、#4544 |

## 先に固定する反例

- Iで読み取ったParameterの版が変わったのに、旧Invocationが同一実行中に新設定を使う。
- Gで正しく生成したが、選択Providerと違う実装を呼ぶ。I/Gの結果が同じでも合格にしない。
- Generatorが作ったファイルを人／AIが編集し、次の生成で黙って上書きする。
- 宣言したHookが未登録／契約不一致／二重到達し、Guard前に外部作用を実行する。
- Markdownの説明・図のレイアウト変更を、業務意味と同じ失効として扱う。
- 類似するPatternの説明を追加しただけで、標準Functionが実装・実使用されたと主張する。

**計測**：成果物の行数だけでなく、TEI側へ移せた意味・判断・検証義務、残った手書き責務、再生成時の変更箇所、実Build時間、実行性能、実際に選ばれたProviderを追う。IとGは同じ入力範囲・Targetで比較する。

## 統合単位

本計画・設計の整合を確認した文書変更は、実装待ちで保留せず独立PRとして扱える。製品実装はDRI-0/1、DRI-2/3、DRI-4、DRI-5の順に分け、既存Ownerを維持する。受入未確認を既存Gateの成功へ昇格せず、`main`の健全性を確認してから統合する。
