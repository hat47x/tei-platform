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


## 8. R1-Mの最初の製品Reader→参照Generator接合候補（2026-11-15）

[Draft PR #4970](https://github.com/hat47x/tei-platform/pull/4970) は、§7の「参照先一因子変更」を既存製品コードに対して回帰試験にする作業である。`tools/reference-generator-selection/rule_markdown_one_factor_generator_test.go` が扱う範囲は次のとおり。

- **比較対象を固定**：同一`sales.order_line.quantity_positive`、同一Language選択`tei.rule-authoring/v0`、同一`std.decimal`二属性、同一生成器Sourceを用い、MarkdownのRule参照先だけを`quantity`→`approved_quantity`に変更する。二属性の定義は両Workspaceであらかじめ同一にしておく。
- **製品側の読取を使う**：実`workspace.Load`→選択済みReader→既存`semantic.Validation`→型Validator→同一LoadのSource Indexを通し、Rule IDと元byte位置の整合を検査する。候補がParser単体で読めただけなら合格にしない。
- **意味と論理Readを同時に照合**：既存`validationeval.Evaluate`のSource interfaceで、元Ruleは`Number(quantity)`・`satisfied`、変更後は`Number(approved_quantity)`・`not-satisfied`を独立に確認する。実Providerの物理Read・完全取得・現在Grantを証明したことにはならない。
- **生成へ明示的に接合**：既存Referenceの`GenerateFromTransformSource`／`VerifyTransformCandidate`を使い、変更後のTyped Ruleと明示更新したGenerator入力契約の一致、生成Go byte変化、別Process`go build`と同一JSON入力での出力変化を調べる。古い入力契約の流用と`std.text`への非数値変更は拒否する。
- **証拠の分割**：`semantic.Validation`の参照先変更、Raw Markdownの変更、Generator入力契約変更、生成Go変更、別Processの結果変更をそれぞれ記録する。Goを一度コンパイルできたことや同じRule IDを保ったことは、Product Builder実消費・業務意味の継続受理・実Provider効果の証明ではない。

**受入上限**：PRのexact HEADでfocused Go testとRepository Healthの実行結果を確認するまで、R1-Mは**候補／未検収**。実Builderの消費MemberとArtifact byte、同一Build Attempt、独立Consumerは次の**R2**に残す。read-only Node Viewを作るときは、このRuleの参照線を意味／論理Readの線として表示し、実行・Provider呼出し・編集可能配線に見せない。新しい共通Core Port型、Graph Runtime、第二Expected Storeは導入しない。

## 9. 横断設計の追加結論：宣言の「合成」と「投影」を別の操作にする

宣言的構文を増やすこと自体は目的ではない。既存のData／Function／Business、標準データ構造、Rule／Query／Operation、PluginとTarget Generatorを、**変更できる境界が分かる一つのモデル**として組み立てることが目的である。構文を増やさなくても、同じ定義からCLI、Studio、業務図、ノード画面、生成Go、設計文書を作れるなら、その方が基盤としては強い。

このため、宣言的構文の機能を次の三つに分ける。

| 機能 | 入力と出力 | 成立条件・禁止事項 |
| --- | --- | --- |
| **宣言の読取り** | 明示選択されたSource＋Language／Role → 元位置を持つ既存Meaning Candidate | Reader選択、対象範囲、原byteを確定する。見出し、表の列名、識別子名から自動受理しない |
| **意味の合成** | 既存Subjectと型付きRelation／Binding／Contract → 接続が成立するかの診断・必要条件 | 型、粒度、時点、集合の完全性、NULL、Effect、現在資格を**必要な問いに応じて**照合。不足条件を暗黙補完しない |
| **用途別の投影** | 同じMeaning／Source／必要なEvidence → DSL、表、Graph、UI、図解、Target／説明文書 | Sourceの正本を増やさず、表現による損失と編集可能なSlotを宣言。表示可能と実行可能を混同しない |

「対象・関係・約束」という§2の分解は、この三つを結ぶ**検証の見取り図**であり、第二の万能IRや新たなCore Schemaを要求するものではない。既存モデルの属性が足りない場合だけ、正規のOwnerで追加すべき最小の契約を検討する。

### 9.1 合成可能性はポートの形だけでは決まらない

接続 `A → B` の候補を評価するとき、少なくとも `A` の保証と `B` の要求について、**型と単位、定義／Instance、BagとCardinality、対象範囲と粒度、評価時点と既知時点、NULL／欠損／partial、順序と短絡、EffectとFailure**の不一致を検出する。

例えば「Rowsの配列」というポート型が同じでも、対象が注文単位か注文行単位か、全件が観測済みか、RLSで一部の行が見えないかによって、`NOT EXISTS`、件数、Uniqueの意味が変わる。つなげられると見えることは合成可能性の十分条件ではない。安全な暗黙変換は最初から設定せず、必要なら既存Operator契約の明示Adapterとして示す。

条件を説明するための形式は `Guarantees(A) satisfies Requirements(B)` と表せるが、これは**証明済みの型システムや汎用Solverが実装されたという主張ではない**。粒度、Snapshot、権限のように一般的な包含判定が困難な属性は、不明なら「接続候補・未受理」で止める。

### 9.2 ノードの見た目を共通化し、線の意味は分ける

データパイプライン、簡易API、業務手順、Plugin構成、モデルの依存関係、実行Traceにノード表示を再利用する。ただし、各線を一種類の「実行する矢印」へ統合しない。

| 線の意味 | 採用すべき主たる確認先 | 初期UIの挙動 |
| --- | --- | --- |
| Data flow／Relation path | Data Context、Operator、Query、粒度とRead Scope | read-onlyで型・多重性・未取得を表示 |
| Conditional／Decision | Validation、Rule／Decisionの評価状態とHit Policy | unknown、error、競合、未評価を見分ける |
| Causal／Execution | Trigger、Operation、Lifecycle、Receipt | 必須先行・並行可能・順序不明を区別 |
| Plugin binding／Realization | Capability、Contract、Provider選択、Implementation Lock | 宣言・選択・実ロード・実使用を別列で表示 |
| Evidence／Projection | Source Index、出典位置、用途別開示、Loss | 現Source版と編集資格を提示。実行線を生成しない |

同じSubjectを複数Viewに表示しても、Identityを複製しない。ノードUIのレイアウト情報とMeaningを別に保持し、同じ表示名の二Node、同端点・同ラベルの多重Edgeを保持する。図の読みやすさを優先してEdgeを隠した場合は、正本へ逆編集できない理由を示す。Plugin間の配線でも、宣言の接続・選択された実装・実際に呼び出されたProviderを混同しない。

## 10. ドキュメントも宣言の検証面として使う

Markdown-firstの利点は記法の短さだけでなく、**人の認知順に沿って目的、対象、規則、例外、理由、根拠を近くに配置できること**にある。

通常の `*.tei.md` は文章、見出し、入れ子リスト、小表、Mermaidと必要最小限の `tei` Fenceで構成する。規則、SQL、大表、業務図面は各Source形式を保持し、該当範囲を選択済みReader／Roleで結び付ける。説明段落・引用・図の矢印・AIの解釈を、そのまま型付きMeaningや実行権限に昇格させない。Markdownの見出し文字列を日本語Grammarとして固定もしない。

ドキュメントから機能への接続は、次のように**照合可能な経路**で成立させる。

`原文とSource位置 → 選択Reader／Role → Canonical候補 → 独立ExpectedとOwner受理 → Target Generator → Artifact／Builder実消費 → 独立Consumerによる観測`

Mermaid・Graphviz・決定表・関数チェーン・ER／業務フロー・設計書は、この経路のどこを可視化したものかを明示する。Graphの構造等価、Ruleの評価等価、Effectの確定、画面操作からSourceへの逆書戻しは別の検収であり、**同じ図を描ける**だけでは代用できない。

文書には、何が分かっていて何が不足しているかを、長い免責文ではなく対象のすぐ近くに表示する。例えば「Readerで実読取済み／Meaningは候補／Providerは未観測／Sourceへの逆編集は不可」という短い資格表示は、概要の注意書きを増やすより実務で役立つ。必要なEvidenceと開示可否は用途相対で判定する。

## 11. 自己記述・ドッグフーディングへの接続

宣言的構文の成熟は、Goの構造をDSLに見せることではない。**実際に存在するTEIの実装を意味のまとまりとして抽出し、その抽象化がコード生成と実動作を改善すること**で判断する。

最初の題材はすでに現行製品Readerに存在する `sales.order_line.quantity_positive` とする。次の順序を保つ。

1. 実装から既存の抽象境界を発見する。Source Reader、`semantic.Validation`、既存Validator／Evaluator、Generator Source、生成Go、実Builder入力を実ファイル・実Callで照合する。推測した構造は候補として分ける。
2. 同じRule IDを保ち、`std.decimal` の参照属性だけを一箇所変更したときに、Load、型、Read要求先、生成Go、実行結果が追随するかを比較する。正常対照と`std.text`への非互換変更を分ける。
3. 同じ意味・違う表層（旧Validation／選択Markdown Rule）と、違う意味・同じ最終Result（参照属性だけが違う）を、別の問いとして検収する。実Provider Readがない試験では「SourceインターフェースへのRead要求」までしか主張しない。
4. 機能の重要部分が実際に生成されたGoへ移ったこと、生成Goを製品の実Builderが**そのAttemptで消費したこと**、同じArtifactを独立Consumerが実行したことを次段で検収する。Goへの委譲だけでは構造のセルフビルドと認定しない。
5. 再利用可能と確認できたOperator／Data Context／Projectionの表現を、必要な箇所だけ`*.tei.md`へ戻す。その変更により定義量、冗長性、検証対象、生成率、失敗診断が改善したかを計測する。Parserを増やして「DSL化率」を見かけ上高めない。

B0固定Bootstrap→B1選択Language→B2自己記述候補→B3独立受理→B4生成とStage2という既存の区分は保持する。受理済みSubjectの再利用と、Interpreter自身をTEIで再構築したことは同一ではない。Plugin／Targetの拡張も、Source権限やRuntime Authorityを自動で獲得する抜け道にはしない。

## 12. 今回のR1-M最小試験とR2以降の引渡し

現行`main`には、選択済み内蔵`tei.rule-authoring/v0`を使う実`Workspace.Load`、元byteを保持するSource Index、`semantic.Validation`への接続、`validationeval.Source`でのRead要求、参照用の`deterministic-transform-v1`→Go生成器が**別々に存在**する。これらは同じ製品レベルのBuilder／Provider資格として連結済みではない。

本ブランチのFocused Go試験は、現行の固定Workspaceを複製し、両方の版に`quantity`と`approvedQuantity`という`std.decimal`属性を用意する。正規のRule IDは変えず、Markdown内の`quantity is positive`だけを`approved_quantity is positive`へ変更する（データ属性宣言のCamelCaseとCanonical IDのsnake_caseを区別）。変更に対応する外部Generator契約の入力AttributeとJSON入力Keyも明示的に更新し、**既存`GenerateFromTransformSource`に渡す**。これらの契約修正はSource差分を受けた必要な調整であり、暗黙の推論ではない。

今回の局所試験では、同一JSONにある正数（`quantity=2`）と負数（`approvedQuantity=-2`）を独立Expectedとして固定し、実Evaluatorが要求したAttributeと生成Goの別プロセス実行が参照先変更に追随することを確かめる。変更前のGenerator入力契約を使い回した場合は、生成前に拒否する。**0・NULL・欠落、旧入力だけを渡す実行時Fallback検査は次の追加試験に残す**。`item_name`（`std.text`）への変更は、正数判定に適した型ではないため既存ValidatorとGenerator契約で拒否する。生成Go byteが異なることは検査しても、Compilerの出力が業務意味を受理したことにはしない。

**この試験の資格の上限**は、製品の選択済みMarkdown Reader→型付きValidationと、既存**参照用**Generator→外部Go Build／別Process実行の局所接合である。契約が自動再生成されるわけではなく、現実のProvider Read、実ProviderへのFunction Entry、製品Plugin実装選択、製品Builderの消費、Windows、Studio Stage1→Stage2は別途未実証とする。CIとexact HEADが確認されるまでは新Go試験自体をPASSと記録しない。

### 12.1 次に残す具体的な差分

| 段階 | 一件の成果物と必要な証拠 | 不足すれば不合格とすること |
| --- | --- | --- |
| **R1-M** | 選択Source→Rule意味→Read要求→参照生成Go→外部Binary結果 | 旧InputへのFallback、意味は同じとするRead先の取り違え |
| **R2** | 同一Attempt内の生成Go元byte、既存実Builderが読むファイル／Module、最終Binary実byte、別Consumer | 生成ソースだけを置き、実Builderが読んだ証拠がない |
| **R3（製品）** | 通常`tei` CLI→選択済みPlugin／実Provider→独立したRead／Effect／Consumer | reference実行を製品の呼出しと誤認する |
| **Stage1→Stage2** | TEI定義・Generatorの自己構築比率、Stage2 Studioの実BuilderとWindows／Native Consumer | Goコード委譲のみ、画面表示のみ、二つの別AttemptのEvidence寄せ集め |

R1-Mを局所的に満たしても、R2以降のGateを先取りしない。既存[#4544](https://github.com/hat47x/tei-platform/issues/4544)（Generator）、[#2600](https://github.com/hat47x/tei-platform/issues/2600)（再利用）、[#2831](https://github.com/hat47x/tei-platform/issues/2831)（self-build）、[#4583](https://github.com/hat47x/tei-platform/issues/4583)（Studio）へ、同じSubject／Source Revision／Build Attemptを紐付けて引き渡す。

## 13. 適用域と採用保留の境界

大規模バッチ／データマート、ノード型フローによるサービス連携、低コードCRUD、Web／モバイル／Desktop、Rule／Decision／BRMS、組織・手続・REA、数理・最適化、TEI／SUI／SHOのModeler／文書化は、**同じMeaningから複数の用途別表層を作る**実例となる。ただし単一の実行Schedulerや統一された副作用モデルを先に作らない。各分野に固有のTransaction／Snapshot／Event／Resourceの制約を、既存OperatorやProviderに明示して吸収できるかで採否を判断する。

本設計の追加結論は「共通化するのは表記ではなく合成条件と検収の接続」「Nodeは用途別の投影」「文書は意味を検査する入口」「実装からDSLへ戻すドッグフーディングを先行」の四点とする。新Grammar、共通Graph Runtime、第二の意味正本、横断Plugin Host、万能Evidence Store、新しいP0 Hard Gateの採用は保留する。
