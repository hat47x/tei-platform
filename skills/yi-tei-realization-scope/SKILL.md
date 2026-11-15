---
name: yi-tei-realization-scope
description: >-
  TEIの図表・Rule・Operationからコード生成や実装を依頼する前に、
  対象の意味・Target・Hook・書込み範囲・独立Expectedを絞り、
  汎用Functionによる解釈、定型生成、混合、既存Providerへの委譲を比較する。
  未確定の業務意味や実行権限を生成AIに推測させない。
---

# TEI：実現方式を選び、生成AIへの課題を小さく固定する

このSkillは**設計・実装候補の切り出し手順**であり、新しいTEI DSL、Generator、Provider、CLI／MCPの実行機能ではない。既存の[意味Sourceを調べるSkill](../yi-tei-modeling/SKILL.md)と[図表実現方式の設計](../../plan/design/DIAGRAM_TO_REALIZATION_STRATEGY_AND_HOOKS_2026-11.md)に従う。利用者の要求範囲、Git／Workspaceの編集資格、Owner意味受理、実Provider／HostのGrantは別に判断する。

## いつ使うか

`*.tei.md`の図表・Rule・Data Table・Node Flowから実装案を作る場合、手書きコードの一部をTEIへ移す場合、または生成コードへの拡張／既存Pluginとの結合方法を選ぶ場合に用いる。

**前提**：対象SourceとTaskの目的が指定されていること。図表や原文の説明だけから実行可能な意味・外部Effectを推測しない。既存の`tei.rule-authoring/v0` Readerで読める範囲と、まだ読めない任意図表を区別する。

## 手順

1. **対象を一つに絞る**。WorkspaceとSource Revision、Canonical ID、明示的Role、既存ModelとFunction／Rule契約を調べる。図上のNode ID・レイアウト・ラベルは業務の同一性ではない。対象を確定できなければread-only観測で止める。
2. **意味と実現を分ける**。必要な入力型、NULL／欠落、集合の完全性、精度、状態／Effect、許される失敗を列挙する。業務意味の穴をGenerator用のHintで埋めない。
3. **実現の候補を比べる**。既存Evaluator／標準Functionが足りるなら解釈実行、Target-nativeに固定した方が良いなら生成、両者の責務が分けられるなら混合。DB／外部Providerへ委譲できるかも既存能力・実権限に基づき比較する。選択は**Contract適合→実装の実在→費用**の順。
4. **自動生成可能な範囲だけ閉じる**。入力・出力、Target、Generator／Pattern／Operator版、許可Import、対象Package／Module、別ファイル／Binding／HookのIDと版を確定する。未解決の裁量が業務判断の大部分を占めるなら、生成の自動適用は見送る。
5. **独立Expectedを候補生成前に固定する**。正常値、境界、NULL／欠落、部分取得、未対応、同名／重複、Source移動、Generator／Provider版更新、Hookの未登録・衝突・Guard拒否を関連する範囲で選ぶ。期待値をAIやGeneratorの出力から作らない。
6. **候補の作成と実使用を分ける**。生成AIには**限定されたGenerator／Hook候補ソース**のみ作らせる。日々の生成は適合した固定Generatorで決定的に行い、生成ファイル自体を編集しない。未適格候補を自動実行・上書きしない。
7. **独立に検証する**。生成元→出力byte→配置→Builderの実消費→実Entry/Provider→Consumerを必要な範囲で別々に確認する。解釈なら実読込した定義、Evaluator版、実Call／Read Scopeを確認する。同値の検証では独立Expectedを使う。
8. **再利用する**。同じ判断が別対象でも成立したならPattern／Template／Skillへ抽出し、適用条件と不適合例を残す。Stage1→Stage2のセルフビルド合格や正式Owner受理へ無条件に昇格しない。

## AIへ渡す最小のタスク票

実装依頼では、以下を指定した**一つの対象／一つの実現**のタスク票を作る。これはAgentへの作業指示であり、TEIの新しいManifest Schemaではない。

| 項目 | 記入するもの |
| --- | --- |
| 対象 | Workspace、Source版、Canonical ID、選択Reader／Role、受理済みMeaningの参照 |
| 要求 | Function／Input／Output／Failure／Effect／Scope、型・NULL・粒度・時点 |
| 実現 | 解釈／生成／混合／委譲の候補、比較理由、Target／Toolchain、既存Contract／Provider |
| 許可範囲 | 変更してよい**非生成ファイル**とHook／Binding、生成物の所有範囲。既存Core改変は個別承認 |
| 結果 | 必要な生成・配置・実呼出し成果物、診断、変更Preview |
| 検証 | 事前固定したExpectedの所在、最低限の正常／負例、実Builder／Consumerの確認方法 |
| 未確定 | 意味／権限／Target適合の未解決と停止条件、Ownerの引渡し先 |
| 証拠 | 実際に用いたSource／Generator／Builder／Artifactの版、未実施の試験 |

**禁止するまとめ方**：「このER図と業務説明から、DB更新・認可・通知・復旧を含む完成アプリを自由に生成し、既存コードの適切な場所に注入する」。未確定なOwner判断・書込み先・Effectが多いため、定型タスクではない。

**許容するまとめ方**：「選択済みのpositive Rule一件を、既存Go Generator／Validation Contractに従って別ファイルへ生成し、指定のBuild入口で実消費を確認する。生成コードは変更せず、独立ExpectedのNULL／0／負値／Source参照変更を試験する」。これは現在の製品で一般化された実行コマンドを提供するという意味ではない。

## ファイルとデザイナーの役割を固定する

対象が複数のソース・リソース・派生物へ展開される場合は、[ファイル役割とBuild手順](../../plan/design/DIAGRAM_TO_REALIZATION_STRATEGY_AND_HOOKS_2026-11.md#13-source・designer・resource・生成物をファイルの役割として分離する2026-11-15追補)を参照する。

タスク票の「許可範囲」には、必要に応じて次を追記する。

| 役割 | 指定すること |
| --- | --- |
| Authoring Source | `*.tei.md`の選択Role／Canonical ID・改訂、元の編集箇所 |
| Designer | 元Sourceへの可逆な編集候補かread-only投影か |
| Resource Input | 画像・文言・CSS／JSON等の使用先とSource byte参照 |
| Generator Definition | 固定済みTemplate／Transform IR／Target Generator版 |
| Generated Output | 人・AIが編集しない対象ファイルと、実Builderへの配置先 |
| Custom Source／Hook | 編集を許す別ファイル、Hook ID／版・Input／Output・Effect |
| Build Intermediate／Bundle | 生成→配置→Compile／Bundle→埋込み→実Consumerを検証する順序 |

`*.generated.go`や`*.generated.ts`などの命名はTargetの慣例であり、**拡張子だけで正本性・所有権・上書き権限を判断しない**。生成ファイルに手で追記する依頼は、Hookまたは非生成ファイルで実現できるか先に確認する。Goの通常の`go build`は生成を自動実行しないので、生成Goの実配置・コンパイル入力は独立に確かめる。Studio Designer上の変更は、対象が一意に特定できない限り元Sourceへ書き戻さない。

## 合格の限界

「ほぼ確実に自動実装できる」は、対象とTarget、要求保証、固定Fixture、反例を決めた後でのみ測定する。AIが同じコードを再回答する確率を前提にしない。重大な反例を通す候補、選択Provider／Builder入力が不一致な候補、未受理Meaning／不明Effectが残る候補は不適格とする。成功したデモを汎用Generatorの完成や現在Grantの証拠へ昇格させない。

## 関連

- [自動生成前の文脈限定Gate／DRI-G0](../../plan/workstreams/DIAGRAM_REALIZATION_REFERENCE_EXECUTION_2026-11.md)
- [Generator Sourceを固定する方式](../../plan/design/AI_GENERATOR_AUTHORING_STANDARD_2026-11.md)
- [生成物のOwnership／Extension Point](../../plan/design/GENERATOR_CONTRACT_BOUNDARIES_DESIGN_2026-10.md)
- [現行のRule→Builder実証](../../plan/design/PLUGIN_RULE_PRODUCT_VERTICAL_SLICE_2026-11.md)
