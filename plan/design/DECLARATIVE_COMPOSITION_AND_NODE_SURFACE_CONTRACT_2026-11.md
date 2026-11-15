# 宣言的構文の汎化：意味の合成とノード表現の共通化

- 作成：2026-11-15
- 状態：**設計・検収計画**。新構文、Graph Compiler、Runtime、Plugin ABI、Ownerによる業務意味受理を採用・実証した文書ではない。
- 主な引渡し先：[#4359](https://github.com/hat47x/tei-platform/issues/4359)（構文比較）、[#988](https://github.com/hat47x/tei-platform/issues/988)（Language／Operator）、[#2621](https://github.com/hat47x/tei-platform/issues/2621)（Markdown／Source）、[#2497](https://github.com/hat47x/tei-platform/issues/2497)（表現間の損失）、[#4198](https://github.com/hat47x/tei-platform/issues/4198)（Lifecycle Graph）、[#951](https://github.com/hat47x/tei-platform/issues/951)（Plugin）、[#1124](https://github.com/hat47x/tei-platform/issues/1124)（編集）、[#2514](https://github.com/hat47x/tei-platform/issues/2514)（独立Expected）、[#4544](https://github.com/hat47x/tei-platform/issues/4544)（生成・Builder）。

## 1. 何を共通化し、何を統一しないか

TEIの宣言的構文で共通化するのは、文面の形ではなく、**何を定義し、何を参照し、どの条件で評価・実行・表示できるかという意味の接続**である。専門家の認知順を守り、Markdownの文章・表・Mermaid、簡潔なRule、関数チェーン、SQL、ノード画面を適材適所で使う。一つの万能言語や巨大な中間構文へ押し込まない。

既存Data／Function／Business、標準データ構造、Facet／Role、Relation／Operator／Rule／Query／Operation／Lifecycle、Language Plugin、Source Index／Acceptance／Target Generatorが意味と実装の受け皿となる。「型付きノード」「入出力ポート」「矢印」は、それらの**構成・依存・実行条件を見せる用途別表現**として先に扱う。ノード編集のために第二Semantic、汎用Host、独立Graph Runtime、別のPlugin Resolverを追加しない。

宣言を読むときは、少なくとも次の区別を保存する。

| 確認する軸 | 明確にする内容 | 混同してはいけないもの |
| --- | --- | --- |
| 対象 | Canonical ID、Source occurrence、Definition／Instance、業務キー、Scope | 同名のNode、同じ属性名、同じ物理PK |
| 構造 | 型、単位、粒度、Cardinality、Node／Edge Identity、参照Path | 表示上の箱・線、暗黙の集合化 |
| 評価 | Operator契約・版、NULL／欠損／未取得、完全性、順序・短絡、時点 | 結果の数値だけの一致 |
| 実行 | Trigger、入力成立条件、Operation／Effect、失敗・Receipt、現在のGrant | 「図でつながる」ことと「実際に呼ばれた」こと |
| 出典と変更 | 原byte、Reader選択／Lock、Source span、正本、変更可能なSlot | 派生Viewを元Sourceへ無条件に逆写像すること |

この表は**既存モデルを照合するための観点**であって、追加の共通Schemaや型分類の採用ではない。

## 2. 宣言の合成単位を「対象・関係・約束」に分ける

構文を汎化する最小単位は「任意の式」ではない。既存の意味を持つ対象に、型付きの関係と成立条件を結び付けた**検証可能な宣言**である。

- **対象（Subject）**：Data、Attribute、Relation、Function、Rule、Operation、View、Pluginなど。IdentityとSourceを追えることが必要。
- **関係（Relation／Binding）**：含む、参照する、入力を要求する、値を返す、制約する、呼び出す、表示する、実現する等。すべてを「依存する」に丸めない。既存Ownerが意味を持たない関係は候補として止める。
- **約束（Contract／Obligation）**：型・多重度・粒度・時点・欠損・順序・Effect・Failure・可視性・Implementation Pinなど。値を計算するための条件と、実行してよい条件は別である。
- **表層（Projection）**：Markdown表、Mermaid、Rule文、関数チェーン、ノード配線、Studio画面。表現形式はMeaningの出所を勝手に拡張できない。

例えば既存の次のRuleは、表・ノード・CLI表示へ投影する際にも、一つのValidation意味候補へ対応づける。

```tei
namespace sales

rule sales.order_line.quantity_positive:
  sales.order_line.quantity is positive
```

これは**現行内蔵Readerの限定例**であり、将来の汎用式文法の採用宣言ではない。複数表層に同じRule IDの編集可能な定義を置かず、片方を正本、他方を導出Viewとする。どちらが正本か不明なら、読み取り専用の候補比較で停止する。

## 3. ノード表現を共通化する際の決定的な境界

同じノード画面を、データ加工、簡易API、業務手順、Plugin構成、Lifecycleの観察に利用できる。ただし、**線の種類が違えば成立条件も違う**。画面上の配線が一つの実行モデルを意味するわけではない。

| ノード間のつながり | TEIで照合する既存の意味 | ノード画面が追加してはならない意味 |
| --- | --- | --- |
| データの入力・出力 | Data Context、Operatorの入力／出力、型・粒度・多重度 | 暗黙の型変換、未取得の自動補完、無断distinct |
| 判定・条件分岐 | Validation／Decisionと判定不能、Rule集合・Input集合 | Unknownをfalse、競合を先勝ちとして処理 |
| 処理の先行・因果 | Lifecycle／Trigger／Operation、明示された先行条件 | Mermaidの矢印や配置順を自動Dispatch順へ昇格 |
| Plugin間の結合 | Contract／Provider／Implementation／exact Lock、公開Capability | 依存関係だけで実Provider選択・現在Grantを確定 |
| View・説明・証拠 | Projection、Source Locator、Trace／Evidence／Disclosure | 表示しただけで元文書の全開示、Source編集、Effect成立を認める |

ノード／ポートを製品で扱う場合の**表示用照合項目**は、対象ID・ポート名・入出力型・Cardinality・必要Scope／Snapshot・NULL／partial／failureの扱い・線の関係種別・選択実装・Source位置・現在資格の上限である。これは正式な`Port` Core型の新設を要求しない。既存意味が与えない値は「未指定」「対応外」と明示する。

**合成が安全な条件**：接続先の入力が要求する型・粒度・多重度・時点・取得完全性と、接続元が保証する出力が整合すること。さらに、評価順や失敗の観測が変わる場合、短絡とRead Footprintが同等であること。Effectを持つ処理は、表示用の線から暗黙に起動せず、既存Operation／Trigger／Provider／Authorityの検収に戻す。配線の決定は「実装可能候補」であり、そのまま「実行許可」ではない。

## 4. 宣言を別の表現へ移すときの合成則

**構造保存**：Graph→Tree／表／MermaidではNodeとEdgeのID、向き、多重性、意味属性を確認する。表示しきれない情報はlossとして明示し、lossを持つViewから逆書戻ししない。ただし一意にSourceを指せるNode名など、無損失のSlotだけを個別に編集候補とする。

**評価保存**：Rule→決定表、パイプライン→SQL、SQL→別方言などでは、結果だけでなく入力範囲、Bag、多重一致、NULL、短絡、順序、丸め、エラー、必要Readを比較する。Readが不要な場合のProvider呼出しを新たに発生させるなら、値が同じでも全面的な同値とは呼ばない。評価契約が不明なNative SQL部分はOpaqueを保持する。

**作用保存**：read-onlyな式と、Commit・通知・外部呼出しを同一の変換対象にしない。Operationがある場合はTrigger、現在の認可、実Call、Effect／Receipt、失敗後の再開可否を、既存Execution／Provider Ownerの証拠で別々に確認する。ノードの描画と関数の生成成功ではEffectを証明できない。

**出典・編集保存**：Source byteとSource span、選択ReaderとProfile、元の糖衣形式、正本となるOccurrenceを維持する。導出ViewやAI候補からのSource変更は、選択対象→最小差分Preview→新Source版への再読取→既存Meaning／Acceptanceの再照合を経る。位置だけで対象を特定したり、同名・多重Edgeを片方へ集約したりしない。no-op Saveは書換えゼロを基準とする。

**再利用保存**：同じ候補Structure Fingerprint、同じRaw Source SHA、同じAccepted Meaning、同じ生成Artifact、同じ実行結果／Grantは別の一致条件。説明や配置の変更では不要なコード再生成を避ける一方、Source Indexは改版する。Operator／Generator版の変更とA→B→Aでは、旧Artifactや現在の実行資格を自動復活させない。

これらは普遍的なOptimizerの公理ではなく、**観測面を限定した変換を受理するためのチェック**である。正負の独立Expectedと実Consumerを伴わない一般化はしない。

## 5. 同じ例を三つの用途に通す

最初の題材は、既存`sales.order_line.quantity_positive`一規則と`sales.order.submit`を**同じ業務領域の異なるSubject**として用いる。前者はすでに内蔵Rule Readerから限定的に評価・Go生成でき、後者は既存Capability／実行モデルのSource照合の起点である。二者を一つの実トランザクションに接続できたとはまだ主張しない。

| 用途 | 正本として必要なもの | 同じViewerで見せるもの | 合格させない見かけ上の成功 |
| --- | --- | --- | --- |
| **データ加工・Batch** | Operator／Query契約とRowset／Snapshot／Coverage | 入出力の型と粒度、変換、失敗・再実行条件 | 配線だけで全件取得・重複排除・確定保存を推定 |
| **簡易API・マッシュアップ** | 公開Entry、入力・出力型、Provider選択、Scope／Auth | Request→Rule→Result／Operation候補 | Ruleが真なら通知やDB書込が完了したとみなす |
| **Plugin構成・Studio** | Capability／Plugin Lock／公開Facet、Generator入力とBuild | Plugin依存・公開Port・Target別実現・実使用Trace | Descriptorを表示しただけでPluginがロード・呼出されたとみなす |

いずれも、表示上は類似した配線を使える。一方、実行の時間・副作用・権限・失敗が違うため、共通にするのは**表現と意味契約の照合**までとし、巨大な横断Schedulerは設けない。

## 6. 最小の一因子反証（既存Expected／Scenarioへ引き渡す候補）

| Case | 一因子差 | 期待する判別 |
| --- | --- | --- |
| DC-01 | 同名Nodeを二つ、Edgeも同じ表示名で別ID | Identityと多重性を保持。表示名だけで合一しない |
| DC-02 | Data lineをExecution lineに変更 | Data dependencyから自動Dispatchへ昇格しない |
| DC-03 | 配線の送り元がbag、受け側がunique必須 | 無断distinctを挟まず、契約不足を診断 |
| DC-04 | 取得済み0件をpartialに変更 | 空集合の真・不存在・Uniqueを断定しない |
| DC-05 | RuleがNULL／未取得／競合を返す | false／成功／既定の分岐へ暗黙補完しない |
| DC-06 | Short-circuitで不要な非公開Fieldがある | 表／chain変換で不要なRead・開示要求を増やさない |
| DC-07 | Plugin exact Pinは一致するがProvider未選択 | 構成を表示できても実Callを主張しない |
| DC-08 | Rule結果は同じだが後続Operationがtimeout | Predicate合格とEffect確定／Receiptを分ける |
| DC-09 | Mermaidの配置のみ変更 | Source Index版変更とMeaning／Build再利用を別判定 |
| DC-10 | 一つのEdgeを省くTreeへの投影 | 表示は許容、Edgeを変更する逆編集は拒否 |
| DC-11 | 生成Goの判断条件を改変、申告Fingerprintは同じ | 実Artifact byteと独立Consumer結果で改ざんを検出 |
| DC-12 | A→B→Aで原文が戻った | Source一致とOwnerの現在受理／Grant／Effect証拠を分離 |

DC-01〜12は**テスト計画上の名前**に限る。第二Expected Storeや独自Runnerは作らない。[#1180](https://github.com/hat47x/tei-platform/issues/1180)／[#2514](https://github.com/hat47x/tei-platform/issues/2514)の既存独立Expectedへ、一因子変更・対象・判定可能な証拠・未実行の範囲を渡してから、必要なFixtureだけを採択する。

## 7. 実装へ戻す順序と終了条件

1. **今ある製品Readerを基準にする**：mainの内蔵`tei.rule-authoring/v0`から同一`Workspace.Load`の元byte→選択済みRule→Source Index／Inspector→既存Validator／Evaluatorへ至る範囲を、正式ReaderとReferenceを混同せず固定する。Graph／Window／外部Languageは個別にA0〜A3を検収する。#4359の正式構文採否は保留。
2. **R1-Mの変更追随を実測する**：初期定義に存在する二つの`std.decimal`属性を用い、Ruleの参照先一箇所だけを変える。同一経路でLoad・型解決・生成Go・別Process実行・独立Expected・Read先を照合する。非対応`std.integer`への変更は拒否を期待する。
3. **R2の実Builder消費を実証する**：R1の生成Goが実Builderのどの入力byteとして使われ、どのArtifactに入り、別Consumerからどの生成分岐を呼べたかを**同一Build Attempt**で突き合わせる。Goへの単なるdelegateや自己申告Fingerprintでは達成にしない。#4544／#2600／#1251へ渡す。
4. **read-onlyノードViewの最小一画面を作る**：既存Source／Inspector／Plugin Inventory／実行Graphから、一Ruleと一Capabilityの**別Subject**についてNode・Port候補・線種・Source資格・失われた情報を表示する。ドラッグ・書換え・自動配線・実行Dispatchはまだ実装しない。Graph Compilerを先行させない。
5. **限定Patchと異種Consumerへ拡大する**：DC-01／02／10のNode／Edge損失、DC-04／06のRead条件、DC-07／08のPlugin／Effectをそれぞれ既存Ownerに渡す。一意Slotのみ編集候補とし、通常CLI／実Providerと独立Consumerを別に受け入れ、その後Studio Stage1→Stage2 Windows実再構築へ接続する。

**全体のP0は変更しない**：Studio Stage1→Stage2実再構築と、通常`tei` CLIの実Entry→実Provider→独立Consumerを優先する。ノード画面・万能構文・Graph Compiler・Native SQLの全方言化・横断Schedulerはその前提にしない。

**この文書の完了判定**：対象と合成禁止条件、具体的な一因子反証、実装Owner、R1-M→R2→最初のread-only Viewの順を、既存計画へ渡せたこと。新しいParser／GUI／DB／実Target／CIを動かしたという証拠にはならない。