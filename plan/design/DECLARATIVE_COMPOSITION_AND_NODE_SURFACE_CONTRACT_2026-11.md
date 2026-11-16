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

[PR #4970](https://github.com/hat47x/tei-platform/pull/4970) はクローズ済み（未マージ）だが、関連Focused Go試験はmainに存在する。同PRは、§7の「参照先一因子変更」を既存製品コードに対して回帰試験にする作業である。`tools/reference-generator-selection/rule_markdown_one_factor_generator_test.go` が扱う範囲は次のとおり。

- **比較対象を固定**：同一`sales.order_line.quantity_positive`、同一Language選択`tei.rule-authoring/v0`、同一`std.decimal`二属性、同一生成器Sourceを用い、MarkdownのRule参照先だけを`quantity`→`approved_quantity`に変更する。二属性の定義は両Workspaceであらかじめ同一にしておく。
- **製品側の読取を使う**：実`workspace.Load`→選択済みReader→既存`semantic.Validation`→型Validator→同一LoadのSource Indexを通し、Rule IDと元byte位置の整合を検査する。候補がParser単体で読めただけなら合格にしない。
- **意味と論理Readを同時に照合**：既存`validationeval.Evaluate`のSource interfaceで、元Ruleは`Number(quantity)`・`satisfied`、変更後は`Number(approved_quantity)`・`not-satisfied`を独立に確認する。実Providerの物理Read・完全取得・現在Grantを証明したことにはならない。
- **生成へ明示的に接合**：既存Referenceの`GenerateFromTransformSource`／`VerifyTransformCandidate`を使い、変更後のTyped Ruleと明示更新したGenerator入力契約の一致、生成Go byte変化、別Process`go build`と同一JSON入力での出力変化を調べる。古い入力契約の流用と`std.text`への非数値変更は拒否する。
- **証拠の分割**：`semantic.Validation`の参照先変更、Raw Markdownの変更、Generator入力契約変更、生成Go変更、別Processの結果変更をそれぞれ記録する。Goを一度コンパイルできたことや同じRule IDを保ったことは、Product Builder実消費・業務意味の継続受理・実Provider効果の証明ではない。

**受入上限**：Focused Go試験がmainに存在しても、同一exact HEADでの実行ログとRepository Healthの確認なしにはR1-M全体のPASSを主張しない。実Builderの消費MemberとArtifact byte、同一Build Attempt、独立Consumerは次の**R2**に残す。read-only Node Viewを作るときは、このRuleの参照線を意味／論理Readの線として表示し、実行・Provider呼出し・編集可能配線に見せない。新しい共通Core Port型、Graph Runtime、第二Expected Storeは導入しない。

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

mainに収録済みのFocused Go試験は、現行の固定Workspaceを複製し、両方の版に`quantity`と`approvedQuantity`という`std.decimal`属性を用意する。正規のRule IDは変えず、Markdown内の`quantity is positive`だけを`approved_quantity is positive`へ変更する（データ属性宣言のCamelCaseとCanonical IDのsnake_caseを区別）。変更に対応する外部Generator契約の入力AttributeとJSON入力Keyも明示的に更新し、**既存`GenerateFromTransformSource`に渡す**。これらの契約修正はSource差分を受けた必要な調整であり、暗黙の推論ではない。

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

## 14. 接続適格性の判定を単一の可否にしない（2026-11-15追補）

宣言的構文の次の実装論点は、**異なる宣言を接続するときに不足・矛盾・成立を区別する診断**である。「Portの型が同じ」「同じRule結果になった」だけで合成可とする設計は避ける。ここで述べる判定は、既存の型検査・Validator・Source Index・Plugin／Execution契約の結果を束ねて**読み取り専用で説明する方法**であり、新しいCore enum、万能Solver、第二のMeaning／Expected正本ではない。

接続A→Bに対して、Bが要求する事実を、Aの**宣言上の保証**と**現在観測できた保証**から検査する。関係する軸ごとに次の三状態を区別する。

| 判定 | 意味 | 最小の扱い |
| --- | --- | --- |
| **成立を裏付ける** | 該当する要求が既存Ownerの契約と具体的証拠で充足される | 該当軸の候補を前進できる。ただし実行許可とは別 |
| **矛盾がある** | 型・単位・多重度・参照先・明示契約等が要求に反する | 接続を拒否し、原因のSource位置と要求側を示す |
| **判定材料が不足** | 対応外、未宣言、取得不完全、Snapshot不一致、未選択Provider等で充足を確認できない | 候補・read-onlyに留める。false／空集合／承認に置換しない |

全体を単一の序列にせず、**構造／評価意味／Source資格／実現・Builder／実行時Authority／Effect・Consumer**を別の問いとして扱う。構造が合致しても業務意味は未受理であり得る。実Buildが成功しても現在Grantや実Provider呼出しを証明しない。意味が受理されていても現在Source Revisionが失効していれば、その受理を自動復活させない。

「不足」は、宣言がない場合と、宣言はあるが**現在の入力・権限・取得完全性では証明できない場合**を診断上区別する。後者は実Providerや実行時の再確認に委ねる。一方、局所的な純粋Ruleの比較にHTTP Delivery Receiptを要求するなど、問いに無関係な軸を共通必須条件に増やさない。

### 14.1 既存Ownerへの照合順

| 調べる内容 | 既存情報 | 無断で行わないこと |
| --- | --- | --- |
| Subject／DefinitionとOccurrence | Canonical ID、Source Index、Role、同名・多重Edgeの個別ID | ラベル一致による対象の統合 |
| 型と構造 | Data型／Facet、単位、Cardinality、Bag／Set、業務キー、粒度 | 暗黙cast、distinct、集約・展開 |
| Rule／Query評価 | Operator版、NULL／欠損、Read Footprint、短絡、時点、取得完全性 | NULL→false、unfetched→0件、余分なRead |
| Process／Operation | Trigger、先行条件、Failure／Retry／Effect境界 | Data EdgeからDispatch順を推定 |
| Pluginと生成 | Contract／Provider／Implementation exact Pin、生成Source、Builder入力とArtifact | Descriptor表示からLoaded／実Callを推定 |
| 出典・編集・現在資格 | 原byte、Source Revision、選択Reader、Acceptance、Authority／Disclosure | 導出Viewから原文を無条件更新・実行 |

比較するGuaranteeとRequirementの出典位置およびOwnerを表示できなければ、型名だけから「接続可能」を製品判断として出さない。Scope、Snapshot、Row Coverageは一般的な型包含だけでは決まらない。非公開情報の診断結果もDisclosureを通して表示する。

## 15. 意味の合成・用途別投影・実行資格を分ける

- **意味保存**：同じSubject、型、Input範囲、Bag／NULL、評価順、Read Footprint、エラーを保つ。結果値の一致だけでは足りない。
- **情報保存**：Source位置、Node／Edge identity、関係属性、完全性、時間条件を失う表現はLossとして示す。損失のある表現から対応不能なSlotを逆編集しない。
- **作用保存**：RuleからOperationへ進むときはTrigger／現在Grant／選択Provider／実Call／Receiptを別途証明する。描画線から自動起動しない。
- **再利用保存**：原byte、Accepted Meaning、生成Artifact、実行時資格は異なる失効単位。A→B→AでSourceが一致しても旧Authority・Effectの証拠は復元しない。

とくに**表現を移すこと**と**新しい接続を意味へ追加すること**を区別する。Ruleをread-onlyノードへ投影しても意味は増えない。一方、Rule結果を「注文確定」Operationの起動条件として接続するなら、新たな業務・実行意味の候補としてSourceとOwnerの受理が必要になる。

CUEの制約統合（[公式仕様](https://cuelang.org/docs/reference/spec/)）は型・制約の静的適合の参考になる。ただしTEIのRead Footprint、現在Grant、外部Effectは単純な値の統合では保証できない。DMNがDecision RequirementsとDecision Logicを分け、Processとの接続を別に扱う点（[OMG DMN仕様](https://www.omg.org/spec/DMN/1.5/)）も参考になる。両言語をTEIの新たな表層文法に採用する趣旨ではない。

## 16. 変更で失効する範囲を分ける

| 一因子変更 | 必ず再確認するもの | それだけで失効／成立を断言しないもの |
| --- | --- | --- |
| Markdown説明・図の配置のみ | 原byte／Source Index、表示・編集Slot | 評価意味や生成Artifactの再生成必要性 |
| Rule参照属性 | Reader→型付きRule、Read要求、Generator入力、生成Go、独立Expected | 実Provider Read、業務Ownerの受理 |
| 型は同じで粒度・Snapshotが異なる | Query／Data ContextのScope・集約・完全性 | 型名一致による接続適格 |
| BagをUnique要求に接続 | 業務Key／多重度・明示Operator契約 | 暗黙distinctやKey推測 |
| Plugin Pin変更 | 選択契約、実ロード・Builder入力・Call追跡 | 過去の実行や現在Grant |
| 認可・Provider観測が変化 | 実行時Authority／Read／Effect／Consumer | Sourceや生成Go byteの変更 |
| A→B→Aで元Sourceに戻る | 現在版と意味・生成・実行の資格を別々に確認 | 旧Receipt、旧Grantの自動復活 |

この表を新しい横断的失効Engineの仕様とはしない。まず一Ruleと二つの表層で、既存World／Model／Artifact Revision、Source Index、Build Attemptの証拠を照合する。

## 17. 製品へ渡す最小実装・反証

最初の差分は新たなComposition EngineやGraph Runtimeでなく、**選択済みRuleの接続診断をread-onlyで提示する局所投影**とする。

1. 実製品Loadの既存テスト（internal/workspace/tei_md_rule_surface_equivalence_test.go）を基点とし、Markdown Ruleと旧ValidationのTyped Meaning一致、Source occurrence／Reader資格の独立を保つ。
2. mainに存在する参照先変更のFocused試験（tools/reference-generator-selection/selected_markdown_rule_rebind_test.go、rule_markdown_one_factor_generator_test.go）から、型・Source位置・論理Read要求・生成Go・別Process結果の現存する観測だけを利用する。ファイル収録と**今回のexact HEAD CIのPASS**は別の証拠。
3. 一つのRule ID、参照Attributeと型、Source位置、選択Reader、依存関係の種類、資格の上限を一覧化する。未定義のPort IDや実Providerの架空の観測値を作らない。
4. std.decimal参照とstd.text不適合を対照し、NULL・欠落・旧入力Fallbackを混ぜない。読取専用投影から実Build、実Provider、現在Grantの成立を主張しない。
5. R2では生成Go byteと既存実Builder入力・最終Artifact・独立Consumerを同一Attemptで結ぶ。R3の通常tei CLI→実Providerは別経路とし、両者の成功後に接合する。Studio Stage1→Stage2を引き続き並行P0とする。

| 反証ID | 一因子変更 | 独立Expectedで確認すること |
| --- | --- | --- |
| DC-13 | 型は同じで入力粒度のみ変更 | 型一致と評価範囲を区別 |
| DC-14 | 完全な0件→partial 0件 | 不在・空集合を確定しない |
| DC-15 | 同結果の式に余分なField Readを追加 | 値一致でもRead Footprint非同値 |
| DC-16 | 図の線をDataからTriggerに変更 | 配置は同じでも実行意味の変更を認識 |
| DC-17 | 同じRuleをMarkdown／旧Validationから読込 | Typed Meaning一致、Source／Reader資格は独立 |
| DC-18 | 選択Provider Bに対して実Call A | Result一致でも実使用不一致 |
| DC-19 | LossのあるGraph→Treeから逆編集 | 不確定なEdge変更を拒否 |
| DC-20 | 説明変更→Rule変更→A→B→A | Source／Meaning／Artifact／Authorityの失効差 |

DC-13〜20は既存#1180／#2514に渡す**反証候補**であり、今回新たにテストを実装・実行・採択したものではない。#2621（Source）、#988（Language／Operator）、#2497（Loss）、#4544（Generator／Builder）、#951（Plugin）、#1124（編集）、#2831／#4583（Studio）へ必要な問いだけを渡す。

成熟度は構文数やノード数でなく、重複しない意味定義の再利用率、接続の誤受理と余分なReadの減少、失効範囲の精度、実Builder消費・独立Consumer観測、同じ機能の定義量と診断手数の削減で測る。

**本追補の受入上限**：診断の設計と反証候補。製品互換性判定器、R2／R3、Source逆編集、Owner意味受理、Studio Stage2の実装を完了したと主張しない。

## 18. 宣言の役割を表記形式から切り離す（2026-11-15横断追補）

### 18.1 同じMarkdown表でも、定義・事実・要求・実現・観測は異なる

横断設計の共通結論は、**表記を統一するのでなく、記述内容の役割と、受理後にどの契約へ接続するかを統一的に説明する**ことである。Markdown／SQL／Mermaid／関数チェーン／Node Viewを、同じ万能Grammarに変換することは目標ではない。

| 文書・構文の役割 | 具体的な対象例 | 既存の受け皿・受入条件 | してはならないこと |
| --- | --- | --- | --- |
| **定義（Definition）** | 異動申請の項目定義表、受注明細のRule、Relation定義 | 選択Reader／明示Roleから既存Data／Validation／Relationへ。Definitionと出典位置を一意にする | 表に見える任意の記録をData宣言に変える |
| **事実（Instance／Observation）** | ある人物の所属履歴、ある時点の注文、計測結果 | Data Context／Providerから、業務キー・値・取得範囲・時点・出典を確認 | 事実の一行をSchemaやRuleとして採用する |
| **成果要求（Desired Result）** | 店舗×日次売上、二つの異なる粒度の帳票、APIの応答 | 要求粒度、評価範囲、集約範囲、取得完全性、品質、利用者を既存Query／Function／Processへ接続 | 要求しただけで物理計算や公開が完了したとみなす |
| **実現選択（Realization）** | Operator実装選択、Plugin Provider、Target Generator | Contract、Scope、選択実装・Lock、Builder実入力、Artifactを別々に追う | 選択済みを実呼出し済みや現在の実行許可に置換する |
| **実行・受領の記録（Evidence）** | 実Call、DB Readback、Receipt、独立Consumerの受信 | 実Attempt、現在の対象／版、観測範囲と完全性、受領先を確認 | ログ不在から無実行を断定したり、別AttemptのPASSを足す |

これは説明のための**役割区分**であり、五つのCore型や第二の文書Engineを採用する提案ではない。既存のSource Roleと各OwnerのSemantic、Execution／Evidence契約に割り当てる。

異動申請の表を例にすると、**項目の名前・型・必須性**を列にした表は定義候補である。一方、「人Aが部署Bへ異動した」という行は記録であり、通常はData InstanceとしてProvider／Queryから読む。さらに「各部署の人数を有効日時点で求める」という表は結果要求となる。同じ表構文を使うという理由で三者を合流させない。組織の有効時点と記録・観測時点、所属と申請・現在Grantも区別する（#2386／#4501）。

### 18.2 文書からMeaningへ入場させる条件

選択済みSourceから既存Meaningへ至る入口は、次の順序で検査する。全体に共通する新しいReaderやAdmission Storeを設けない。

1. **対象と形式の選択**：Workspace identity／epoch、Source path、原byte、選択Language ID／版、Source occurrence、Definition／Reference／Example等のRoleを固定する。見出し名や表の列名だけで、採用するReaderや意味を推定しない。
2. **Roleに対応する既存受け皿の確定**：Data定義表はData、Rule文はValidation、実記録はProviderのInput、成果要求はQuery／Function／Result契約等へ引き渡す。受け皿のない記法はunsupported／候補として扱う。
3. **意味・同一性の照合**：既存ID、型、Key、Relation、単位、粒度、時点、NULL／欠損／未取得、Source位置を、必要な対象について検査する。同じCanonical IDの二重Definitionを表層の優先順位で黙って解決しない。
4. **変更と投影の境界**：元表面の文面とSpanを保ち、無損失の編集Slotだけを別途資格化する。見せられるが逆編集できない図や要約はread-onlyとし、no-op Saveはbyte不変を基準にする。
5. **製品経路の受入**：SourceをLoadできること、型付きMeaningが一致すること、実Builderが生成Goを消費したこと、選択実装が呼ばれたこと、Consumerが結果を受領したことを別の試験とする。

既存の内蔵Rule Reader（tei.rule-authoring/v0）は実製品Loaderの基線である。PR [#5010](https://github.com/hat47x/tei-platform/pull/5010) は、明示されたData項目定義表を読み、既存Dataへ接続する提案として参照する。**この追補の確認時点でPR #5010は未マージ**。同PRが提案する言語名や表構文を本設計で正式採用しない。Source種類が増えた場合でも、Readerの選択・既存Meaningとの同値・元位置・重複拒否・正本選択をそれぞれ検収する（#2621／#4359／#988）。

## 19. 適格性診断と実行の受入結果は異なる

### 19.1 接続候補の三判定と、実証の六状態を混同しない

§14の「成立を裏付ける／矛盾／判定材料不足」は、**接続A→Bの要求と保証についての設計時の判断**である。一方、既存の検収票のPASS／FAIL／NOT RUN／UNSUPPORTED／UNKNOWN／STALEは、**指定された実経路を検証した際の状態**である。互いに一対一対応しない。

たとえば、型とSourceが一致した接続は構造上「成立を裏付ける」と表示できる。それでも実Providerの呼出しはNOT RUNかもしれない。Providerが実行されていてもReceiptが欠落すればEffectの結果はUNKNOWNとなる。以前PASSしたProducerがSource改版でSTALEになった場合、古い結果を現在の接続資格に流用しない。未選択Providerは実装選択の不足であり、型矛盾と混同しない。

**判定票の最小項目**（現行Schemaの新設ではなくread-only照合用）：問い／Workspace ID・epoch／Subject ID・Source occurrence／Role・Reader／要求する保証／実際に確認した保証／既存Owner／当該Source・Revision／観測範囲・時点／判断理由／次に不足する証拠／現在Disclosure。生成・実行・受領の欄は実物がある場合だけ関連Attempt、Artifact、Provider、Consumerを記す。

一つの画面に一覧・ノード図・文章説明を同時表示しても、同じ判定票の異なるProjectionとし、UIの線や位置からMeaningやGrantを生成しない。見る人に開示してよい情報と、編集・実行を許してよい情報も別に検査する。

### 19.2 三つの異種接続で受理条件を確かめる

| 接続候補 | 保持したい意味 | 静的に検査するもの | その後の独立受入 |
| --- | --- | --- | --- |
| **Data項目定義表 → Rule** | 対象Attribute、型、元Sourceを保つ | Data定義とRuleが別Sourceでも、同じWorkspace・Canonical ID・型へ解決されるか | Ruleの評価Read要求・生成Go・型不適合・旧契約拒否を独立Expectedと比較。実Providerは別 |
| **Rule → Desired Result／Node** | 条件を使った結果と、条件に使う入力範囲を分ける | Ruleの真偽と結果の粒度・NULL・Read Scope、EdgeのData／Control役割 | 無断集約・未取得→0件・図の線→Triggerの誤昇格を拒否。実Build／実Consumerは別 |
| **Operator宣言 → 実装選択 → CLI** | 契約に合う選択と、実際に呼ばれた実装を分ける | Operator ID／版、Workspace manifestの選択、対応関数、実装候補の一意性 | 選択B／実Call Aの負例、古い選択／生成Commandの失効、実Host・Grant・Consumerを別に検証 |

PR [#5007](https://github.com/hat47x/tei-platform/pull/5007)では、WorkspaceにおけるOperator実装選択と局所的な実Call観測がmainに統合された。一方、同PRの説明でも、生成済みCommandの選択ID固定、一般Plugin Lockからの選択、通常CLI全経路の受入には残差がある。**選択実装を観測できた局所試験と、Pluginの製品E2Eが成立したことは別**として扱う。

## 20. 最小の縦断と作業の出口

既存のR1-M／R2／R3、Node N0〜N4、Studio Stage1→Stage2、I2M-1、#1180／#2514に接続し、新しい並行ロードマップやHard Gateは作らない。

| 順序と引渡し | 対象・成果物 | 独立の反証と終了条件 |
| --- | --- | --- |
| **A：Reader境界を確定**（#2621／#988） | 選択済みRuleと、必要に応じて受理されたData項目定義表から同一既存SemanticとSource Indexを得る | 定義表と記録表を混同しない。二表層で同一ID・同一Meaningなら二重Definition拒否、別Workspaceの同名を合一しない |
| **B：局所診断を一つだけ生成**（#1124／#2497／#4198） | quantity_positive一RuleのSource・Role・型・Read・接続理由・不足証拠を、CLI／Inspectorが参照できるread-only投影として表示 | 図・表・説明が暗黙DispatchやProvider成功を主張しない。曖昧なSpan、LossのあるEdgeを逆編集させない |
| **C：製品生成と呼出しを分けて証明**（#4544／#2595／#4495／#951） | R2は生成Go実byte→実Builder入力→Artifact→別Consumer。R3は選択Provider→通常製品CLIでの実Call | 同じRule／Attemptを照合できない間は一連の成功とみなさない。選択と実Callの不一致、古いCommandを検出 |
| **D：Studioで同じ問いを観測**（#1824／#2831／#4583） | PinしたWorkspace・Subject・Source版を、実Windows Stage2のInspectorと生成物へ結ぶ | Pin／遅着／ABAの反例、実Windows WebView2の受領、Stage1によるStage2の再構築を別々に検収 |

**最初の着手はAの既存基線確認とBの一Rule read-only診断だけ**とする。実装可能性を調べるために、Data表・Rule・Operatorの全汎化構文を同時採用しない。構造の同値検査を製品Builder／ProviderのP0完了条件へ追加しない。

### 20.1 既存Expectedへ渡す一因子反証候補

| ID | 変更 | 拒否／保存すべき境界 |
| --- | --- | --- |
| DC-21 | 同じMarkdown表でRoleだけをDefinition→Instanceへ変更 | 表の外形一致からData宣言を自動生成しない |
| DC-22 | 同じData IDをMarkdown表と従来DSLから二重に定義 | Readerの優先度で片方を黙殺せず、二重Definitionを拒否 |
| DC-23 | 同じ型とKeyの成果要求についてGrainだけを注文→注文明細へ変更 | 型一致と集計結果の同値を混同しない |
| DC-24 | 取得0件がcomplete→partialになる | 不在やRuleの適用対象不存在を確定しない |
| DC-25 | 実装選択Bのまま局所Call先をAへ差替え | 結果値が同じでも実Callの不一致を検出 |
| DC-26 | 同じSourceへA→B→Aで戻し、古いCommand／Source Indexを渡す | byte一致だけで実装選択・実行・開示の現行資格を復活させない |

いずれも**未実行の候補**であり、追加のScenario正本は設けない。#1180／#2514の独立Expectedに割り当てる際は、実装から期待値を作り直さず、正例・負例・観測Owner・受入の上限を先に固定する。

### 20.2 汎化の成果を測る

表現形式の数ではなく、(1)定義と記録の誤読取を防げるか、(2)同じ意味定義をRule／表／Node／CLI／Studioへ再利用しても正本が増えないか、(3)失効・不足・禁止を原因つきで説明できるか、(4)実際の生成・Builder消費・Provider呼出しが増えたか、(5)新機能に要するDSL量・手書きGo量・変更手数が減ったかを測る。二つ目の異質な領域でも有効になるまで、汎用Writer／Optimizer／新Core型の採用判断は保留する。

**この追補は実装前の設計判断と作業引渡しであり、PR #5010のLanguage仕様採用、GUI Writer、Rule／Data表の汎用逆編集、R2／R3、Stage2セルフビルドの実証ではない。**

## 21. 宣言の入口を「形式・役割・意味」の三つで照合する（2026-11-15追補）

### 21.1 現行製品で確かめられる範囲を更新する

現在のmainには、明示選択した内蔵ReaderでMarkdownのRuleを読み取る経路に加え、**印を付けたData項目定義表を既存semantic.Dataへ変換する実コードとWorkspace Loader試験**が存在する。実在する実装は、internal/syntax/data_table_authoring.go、internal/workspace/authoring_language.go、internal/workspace/tei_md_data_table_test.goで確認できる。元の提案PR #5010はclosed／unmergedであり、PRの状態だけを製品実装の有無と同一視しない。採用来歴と現在のmainでの実体は別に確認する。

| 一つのSourceを読む時の問い | 現行基線 | 追加の判断を要すること |
| --- | --- | --- |
| **どのReaderが読むか** | Workspace manifestでLanguage契約と版を明示する。実Loaderはちょうど一つの内蔵実装を選ぶ | 外部Language Pluginの選択／Lock、複数Reader競合の製品資格 |
| **どこが定義か** | tei Rule領域と印付きData項目定義表を読み、元Sourceの位置を保持する | 文章・例示表・業務記録を推測で定義に昇格しない |
| **既存Meaningへどう対応するか** | Ruleはsemantic.Validation、Data項目定義表はsemantic.Dataを使う。後者には従来Dataとの型付き同値を比較する試験がある | 同値な候補と、同一Workspace内で競合する二つのDefinitionを混同しない |
| **読めた結果をどこまで使えるか** | Source Index、型Validator、限定Rule評価の範囲まで局所試験がある | Query／Persistence／生成／公開Entry／権限を読取成功だけで確定しない |

この三つの問いを一つの「Markdown対応済み」に潰さない。利用者の認知順に沿った表や文章を維持し、Goの構文、Readerの技術的な構造や内部的な識別子をDSLへ持ち込まない。既存のtei.rule-authoring/v0とtei.authoring/v1がどう共存するかは、Source選択の曖昧さを増やさないという条件で#988／#1018の判断へ戻す。

### 21.2 定義と記録の違いは構文ではなくRoleで守る

項目定義表は「所属履歴を記録するDataの形」を宣言するもので、実際の所属行や異動履歴の正本ではない。所属・責任・現在Grant、validAt・knownAt・取得完全性は、実際のRecordとQuery／Providerで確認する。将来、文書中の例表や試験入力もReaderで扱う場合には、採用するRoleと対象Snapshotを先に定義し、Definitionへ暗黙昇格させない。

## 22. Operatorの選択と実行が成立しても、意味の適合は別に確かめる

[PR #5007](https://github.com/hat47x/tei-platform/pull/5007) では、WorkspaceのOperator宣言とmanifestの実装選択を分離し、Scenario評価関数で選択した組込み実装への局所Callを観測できるようになった。これは既存意味モデルを維持する上で重要な前進である。一方、実際に呼ばれた実装が、業務上要求した演算規則と**すべての入力で一致するか**は別問題である。

たとえば数値0に対して、非負判定はtrue、非ゼロ判定はfalseとなる。両方とも数値を受け取るため型だけでは選択を排除できない。選択IDを変えても同じWorldを読めることを成功としつつ、**演算の意味・境界値・NULL／欠損・失敗時の振舞い**はOperator契約の独立Expectedと照合する。適合しなければ、正しく選択・呼び出せた実装でも業務意味の受入は不可である。

| 層 | 確認すること | 現時点で主張しないこと |
| --- | --- | --- |
| Operator定義 | 意味ID、対象型、要求するLaw・入力／出力条件 | 名前や引数型だけで実装が適合すると決めること |
| 実装選択 | manifestのOperator→実装名、重複・未宣言の拒否 | その選択が永続すること、Plugin Lockが存在すること |
| 実Call | 選択した組込み実装が評価時に呼ばれた事実 | 外部Pluginの実Provider呼出し、通常CLI全体のE2E |
| Conformance | 0・正負・NULL／欠損・競合／エラーなど、Ownerが定めた独立Expected | 型適合や同じ一入力の結果だけによる意味受理 |
| 版の現在性 | 実装選択を含めたCommandとBuildの再資格化 | Source byteが変わらなければ旧Commandを再使用できるとの推定 |

特に**Operator実装選択のみの変更**は、Ruleの意味IDを変えずに実行結果を変え得る。manifestを変更しても旧Scenario Commandを有効のまま利用できる残差は、選択識別子・生成コマンド・実Attemptを照合する対象とする。新しいResolverや汎用Executorを追加するのではなく、既存のSelection／Source・Artifact・Executionの失効規則へ戻す。

## 23. 宣言から実Artifactまでの出典を一つの問いで追う

### 23.1 生成されたGoが動くことと、元宣言からの連続性

[PR #5009](https://github.com/hat47x/tei-platform/pull/5009) により、製品Workspaceから選択したRuleが生成したGoを、通常のtei self-build、既存Build Intent、実Builderの経路でビルドし、実Binaryの判断を独立入力に照合する限定回帰がmainに入った。[PR #5020](https://github.com/hat47x/tei-platform/pull/5020) では、元RuleのSource位置と生成Go AST上のRead先・条件を一試行で照合する回帰もmainに入った。

二つの実証を連結したように見える図は描ける。しかし、**同一Build Attemptで元Markdown byte、Reader選択、型付きRule、Generator実装版、生成Go byte、Build Intent、Builder実入力、Artifact、Consumerを結んだ証拠**がまだなければ、正式R2-Pの全経路を合格としない。

最小限必要な照合順は以下である。新しい永続Evidence Storeを設けず、既存Ownerの情報を一問一対象で参照する。

1. **原Source**：Workspace ID／epoch、元byteとSource Index、Reader ID／版、Rule Definition occurrence。
2. **意味と生成**：選択した型付きRule、Data型、Generator実コード／版、生成Goの実byteとContent Digest。
3. **実Build**：同じ試行のBuild Intent／Definition Source、Builderが実際に読んだ入力、Artifact memberと最終Binaryの実byte。
4. **実利用**：別Processの独立Consumer、必要なら公開Function Entry／選択Providerの実Callと現在Grantをさらに別票で照合する。

いずれかの対応関係が不明なら、その**辺だけを未接続**として示す。ASTのGoファイル位置を原MarkdownのSource位置として扱わない。Rule IDが等しくても別Workspaceや別Source RevisionのArtifactを流用しない。

### 23.2 最初の一画面・一問い合わせ

CLI／Studio共通のread-only表示は、既存のInspect結果を入口とし、一つのRuleについて**原定義／Reader／型付き意味／論理Read／生成Go／実Builder／実Call／独立Consumer**を必要な範囲だけ並べる。現在の製品APIが持たない欄は未取得、接続関係が不明な欄は未接続と示す。Inspectorでリンクを開けたことをSource編集資格や実行許可に昇格しない。

同じ情報をMarkdown表、Source近傍の説明、Node Viewへ投影できるが、表示を変えても意味の正本を複製しない。最初は一つのRule IDに限定し、Graph／Process／Pluginへの拡張はOwnerが実際の異種ケースで保証を検証してから進める。

### 23.3 既存独立Expectedへ渡す反証候補

| ID | 一因子の変更・負例 | 判定してほしいこと |
| --- | --- | --- |
| DC-27 | 印のない事実の表をData定義表と同じ列順で配置 | SourceのRoleを推測で昇格しない |
| DC-28 | 二つのLanguage実装が同じSourceを受け取る | 選択順で勝者を決めず、曖昧さを拒否 |
| DC-29 | Operator実装を非負から非ゼロへ変え、入力0を与える | 型／Call成功と意味Lawの不一致を区別 |
| DC-30 | Ruleと原byteは同じでmanifestの実装選択のみ変更 | 旧Scenario Command／関連Artifactの現在性を再確認 |
| DC-31 | 生成GoとASTは一致するが、Build Intentが別のSourceを指す | 生成結果の一致だけで元Sourceとの連続性を認めない |
| DC-32 | 同じRule IDを別Workspace／別Source版から読んで一つの実Binary結果へ接続 | 同名・同IDを結合キーにせず、一試行のSource資格を求める |

DC-27〜32は新規テスト実施を表さず、#1180／#2514の独立Expectedへの**引渡し候補**とする。既存#988／#1018（Language／Operator）、#2621（Source）、#1121／#4544（Generator／Builder）、#951／#4495（Plugin／Entry）、#1124／#1824（Inspector）、#2831／#4583（Stage2）へ担当の問いとして戻す。

**優先順位は従来どおり**、通常CLIでの実Providerと、Stage1が再構築したStudio Stage2の実Windows受領をP0とする。本追補によって新Grammar、横断Runtime、第二Meaning／Evidence Store、全面的なSource Writerを前提条件にしない。

## 24. Data定義表を既存の製品Inspectorで確認する最初の差分（2026-11-15）

§20〜23の「定義／事実を分け、同じSourceと意味IDを追う」を、構文数を増やさず実コードで確かめる。現行のinternal/workspace/testdata/tei-md-data-tableには、明示選択したtei.authoring/v1、印付きData項目定義表、印のない記録例、隣接するRuleが**一つのMarkdown Source**にある。Workspace.Loadでの型付き意味の同値は既存のinternal/workspace/tei_md_data_table_test.goが別に検証する。

最初の追加試験internal/operations/inspect_markdown_data_table_source_evidence_test.goは、**既存InspectionFromLoad／ReadWorkspaceSubjectを使い、Product Inspectorまで到達すること**を検証対象とする。新しいReader・Inspector schema・CLIサブコマンドは作らない。

| 検証対象 | 独立して固定する観測 | 資格の上限 |
| --- | --- | --- |
| **選択ReaderとDefinition** | Reader tei.authoring/v1が一つのSourceを読み、Data／Attribute／Ruleの三Subjectを既存の意味IDで解決する | Language選択とSource traceは業務意味のOwner受理ではない |
| **Source Span／Role** | 原byteとIndexのFingerprint、Data名・項目セル・Rule名のbyte開始位置、RoleDefinitionを一Loadから照合する | span一致だけで元Sourceへの編集資格は付与しない |
| **非定義の境界** | 印のない例表から架空Dataを生成せず、Ruleを公開Capabilityへ変換しない | 読取・Rule定義が外部Function／Provider呼出しを意味しない |
| **説明文のみの改版** | Canonical Data／World fingerprintは不変、Source Index／Model Revisionは変化する。A→B→Aでは記述的Fingerprintだけが復帰する | 古いPin、Grant、Artifact／Effectの証拠は復活しない |

この小実証が成立すれば、**Source Role→既存Meaning→Inspector→位置の検証**というP1の最小経路を、Ruleだけでなく異種のData表にも広げたと判断できる。ただし実装済みとはGoコードがmainへ収録された範囲であり、exact HEADのGo試験成功はCIの個別結果で確かめる。InspectorはAuthoringSourcesにあるReader選択を独立した公開列としては返さないため、Loader内の実読取記録とInspectorのSource traceを**同じ一Loadの試験で照合**する。Studioで選択Languageが表示できるという主張はしない。

次のP1は一問一Subjectのread-only説明表示である。先にReadWorkspaceSubjectとInspectionFromLoadの既存列を再利用し、将来の表示列はProduct Inspectorの公開契約に存在するかを検査してから追加する。Ruleの論理Read Footprintは別の既存評価観測が必要であり、ここでData項目定義から推測して埋めない。原宣言→Generator→Build Intentの正式R2-P、実ProviderのR3、WindowsのStage1→Stage2は別P0で維持する。

この節は**実装回帰の根拠と次の受入条件**であり、Data項目定義表の全構文採用、Query／Persistence／型付き逆編集の完成、Sourceへの書戻し、実Provider Callの証明ではない。

## 25. Markdownの定義を実CLIから同じMeaningで読む（2026-11-15製品追補）

### 25.1 二つの異なる表層で、同じInspect契約を検証する

これまでの製品Inspector回帰は、選択済みRuleの実CLI経路と、Data項目定義表の同一Load内の出典照合に分かれていた。main統合済み[PR #5037](https://github.com/hat47x/tei-platform/pull/5037)では、前者について、実OS Processで起動したtei inspectと既存operations.InspectSubjectを比較する試験がある。

これをData項目定義表にも広げる最小差分が、cmd/tei/cli_inspect_markdown_data_table_process_test.goである。既存のtei.authoring/v1を明示選択したWorkspaceを読み、追加のCLIやReaderを作らず、実Go製品バイナリをビルドしてinspectを別Processで呼ぶ。

| 確認点 | 一件の試験が比較する内容 | 意味の範囲 |
| --- | --- | --- |
| **実CLIとApplication Operation** | Data／Attribute／Ruleをkind＋厳密IDで取得し、CLIのResult全体を直接InspectSubject結果と比較 | Readerが選んだ既存Semanticのread-only表示であり、外部Providerの実Callではない |
| **原Source** | 三SubjectのRoleDefinition、同一Markdown Source Path、原byte Fingerprint、Data名・セル・Rule名の範囲 | 同一表示名や字面を別KindのSubjectへ自動転用しない |
| **非定義の拒否** | 印なしの記録例から架空Dataを作らず、Rule IDを使うCapability読取りもSUBJECT_NOT_FOUNDで拒否 | Sourceに文字があることと、公開可能なOperationの存在は別 |
| **変更の再取得** | 説明文のみ追記後、CLIのWorld／Canonicalが同一でもSource Index／Artifact Fingerprintが変わることを確認 | 新Source版が見えたことは、Grant・Artifact・Owner受理を更新しない |

この回帰の対象は**内蔵CLIのdirect inspect経路**である。選択した外部Inspector Provider、Studio Bridge、MCP、独立した実Host、DBとAPI Consumerを一度に試験したと扱わない。特に同じJSONが出ても、実際に選ばれた実装が違えば、実行資格の同値はまだ成立しない。

### 25.2 Sourceの状態とMeaningの状態を別の証拠として表示する

同じData定義が、従来DSLとMarkdown表の両方から構築されても、**型付きMeaningの一致は一つの比較**、Reader契約・Source位置・Source改版の一致は別の比較である。Sourceだけの変更に対してWorld Fingerprintまで必ず失効させる実装は避ける。ただし、Source位置に基づくPin・編集・開示判断は独立に再資格化する。

当面のread-only説明欄は、Product Inspectorが返せるSubject Kind・厳密ID、Canonical、Workspace、World Fingerprint、Source Index Fingerprint、Source Status、Source Reference／Artifact、AuthorityLimitsを基線とする。Reader ID／Versionは現行のLoader記録から試験で照合できるが、Inspectorの公開JSONにないため、そこへ架空の列を付けない。

これにより、Studioのノード図、Markdown表、CLI一覧を**同じSubjectの異なる投影**と説明できる。ただしData参照線、評価条件線、Process Trigger、Plugin依存、実Provider Callの線は混同しない。Node Viewへ持ち込むのはこの区別を表現できるようになってからであり、新しいノード実行Engineを先行させない。

## 26. DSL→生成Go→Studioへの接続で何を証拠にするか

並行チャットでは、main統合済み[PR #5051](https://github.com/hat47x/tei-platform/pull/5051)により、選択元Rule MarkdownのbyteコピーをGo packageへ配置し、go:embedでBinaryに収録してDigestを照合する限定試験が追加された。これは§23の生成元閉包と消費閉包を近づける成果である。一方、**元ファイルのLocator・Reader・Generator資格まで正式なBuild Intentと一Attemptに束縛したR2-P全体**ではない。

また、[PR #5052](https://github.com/hat47x/tei-platform/pull/5052)は生成Workbench PoC BについてWindows WebView2の通常Job／生成Jobの成功を整理した。これは生成UIの実Consumerとして重要だが、**TEI自身が非PluginのStudio Stage1を使ってStage2 Desktopを再構築する製品セルフビルド**の完了ではない。

このため、宣言的構文の製品成熟度は次の**三つの独立した実物**で測る。

| 実物 | 次に確認する独立の成立条件 | 担当する既存経路 |
| --- | --- | --- |
| **宣言の読取結果** | 選択WorkspaceでRule／DataがInspector・CLIから同じMeaningとSourceへ解決される | #2621／#988／#1124／#1824 |
| **生成・Buildの結果** | 元DSLの正本／実Generator／実生成byteと、実Builderが消費したGo＋DSLの対応を同一Attemptで証明する | #1121／#4544／#1180／#2514 |
| **利用先の実結果** | Studio／公開Entry／通常CLIのどのConsumerが、どのArtifact／選択Providerを本当に使ったかを独立に観測する | #4495／#951／#2831／#4583 |

最初の一問はData表の実CLI Inspector、次はRuleの実Builder出典、並行P0はStudio Stage1→Stage2の実再構築と実Provider／独立Consumerである。CLIから読めたこと、生成Sourceが同じこと、PoCのWindows UIが起動したことを、ひとつの万能なPASSへ合成しない。

### 26.1 既存独立Expectedに渡す負例

| ID | 一因子変更 | 失格とする短絡 |
| --- | --- | --- |
| DC-33 | Markdown Data定義の元Sourceを変えず、CLI inspectを直接Operationと比較 | JSONのSubject、Source、Authorityの一部だけ一致しても全面同値と呼ばない |
| DC-34 | 同じMarkdownにRuleがあっても、kindだけsemantic-capabilityへ変更 | 文字列IDが一致するだけで公開Functionとして取得しない |
| DC-35 | 宣言の後へ説明文だけを追記して再度CLI inspect | Worldが一致するからと古いSource Index／Pinを流用しない |
| DC-36 | 同じ正値Rule Goに別の原文を同梱してBuildし、結果値は同じ | 実生成・原文byte・Builder入力の同一性を結果値だけで確定しない |
| DC-37 | Studio生成PoCが正常起動するが、Stage1からStage2を再構築していない | 生成PoCのConsumer PASSをStudioセルフビルドPASSと呼ばない |
| DC-38 | Data配線の見た目だけをProcess Triggerへ変更 | 表現の編集が実行意味やGrantを自動生成しない |

DC-33〜35は追加Go回帰の観測対象と重なる。DC-36〜38は**未実装の独立Expected候補**であり、新しいExpected正本を作らず既存#1180／#2514へ渡す。追加Go試験のファイル収録とCI exact HEADのPASS、実Windows／Providerの実観測はそれぞれ別に報告する。

## 27. 同じ意味を実現する方法として、コード生成とパラメータ実行を選び分ける（2026-11-15）

宣言・Markdown表・Graphのパラメータから処理を実現する際、**すべてのModelを個別のGoソースに変換する必要はない**。既存の型付きOperator／Functionが、受理済みパラメータを受け取って同じ意味を実行できる場合は、その汎用処理を再利用する方が、生成・コンパイル・再配布を省ける。逆にTargetへの静的な組込み、固定された処理の高速化、外部Goへの委譲を減らすこと、実装の自己生成が目的なら、Sourceから判断処理を生成する価値がある。**意味の正本と検収条件を共通にし、実現方式だけを選択する**。

### 27.1 三つの実現方式は、できたことを別に数える

| 方式 | 処理を実現するもの | 適する状況／確認するもの | 数えない成果 |
| --- | --- | --- | --- |
| **G：Targetコード生成** | 受理済みRule／Graph等から生成した分岐・型・参照・接続処理を、既存Builderで実Artifactへ組み込む | 静的Target、配布先の実行環境制約、頻出処理の最適化、TEI自身／Pluginの実装置換。生成元・実Generator・Builder入力・Artifactを追跡 | 固定Goへの薄い委譲ラッパー、生成SourceをBuildせず放置 |
| **P：パラメータによる汎用処理** | **既に存在する**型付きOperator／Functionの実装が、Modelの構造・条件・Port binding等を入力として解釈し実行する | 同じ演算を多数のModelで再利用、設定の変更頻度が高い、再Build不要なread-only変換。実Param／契約版・Provider・呼出しと独立Expectedを追跡 | 設定を読めただけ、表示だけ変わったこと、実処理未呼出し、生成コードと同じ実装比率への加算 |
| **D：外部実装への委譲** | 手書きGo・外部ライブラリ・外部Serviceが意味処理の主体となる | 初期Seed、Target固有I/O、専門計算、未対応Plugin。Scope・入力／出力／Effect／Failure／実Provider選択を明示 | TEI自身がその業務判断を実装・生成したとの主張 |

Pは**第二Interpreter／Rule Engine／Graph Runnerの新設を意味しない**。標準Operatorの実装に必要な純粋計算が既にあり、パラメータを型付きで受け取れる場合だけ利用する。OperatorがTargetに存在しない場合は、Target-nativeな実装・コード生成・明示的unsupportedを比較し、Go Runtimeの常駐やWails Hostを全Targetの必須依存にしない。Effectfulな処理はどの方式でも現在Grantと実Caller／Effect Receiptを既存Host側で検証する。

### 27.2 実現方式を切り替えても、Source・Meaning・型・実結果の合格基準は動かさない

第一比較は、製品が選んだ`sales.order_line.quantity_positive`一規則と同じ`std.decimal`属性・入力を固定し、(G) 現行`validationgen.GeneratePositive`で生成した判断Go、(P) 既存`validationeval`等が同じ型付きValidationを受ける評価経路、(D) 固定手書きGoの三つを、**実際に呼べる経路だけ**比較する。Gの限定生成と別Process実行はmainにあるが、**Pの同一条件での製品実Call・G/P/D三方式の独立E2E比較はこの計画では未実行**。未対応Kind／Relation／型を、別方式なら自動許容されるとしない。

別のPressure Caseとして、二Node・二重Edgeから「指定Nodeの出辺IDを返す」Graphを使う。Pで実装するなら、Node／Edge／Port／Roleと方向・多重性を保持した**標準のGraph値**を、実Operatorの入力へ渡す。Graphのレイアウト、表示名、抽出時の行順を業務IDにしない。これは現段階では**Reference候補**であり、正式Graph Reader・汎用Operatorの製品実装があるとは主張しない。Graph ViewからPへの配線も実行許可に昇格させない。

| 反例 | P（パラメータ実行）で検証すること | G（生成）で検証すること |
| --- | --- | --- |
| GP-01 閾値／参照属性など意味のある入力を変更 | **同じ実Binary**でも、現在Param／Inputを渡した実Callの結果が変わる。旧Paramを現在結果に使わない | 生成Go・実Builder入力・実Artifactが必要に応じて改版される。旧Artifactは現在資格でなくなる |
| GP-02 図の座標・配色・説明だけを変更 | 意味Param不変なら判断結果は不変。元Source／Projectionの改訂は観測する | 生成処理は意味上同じになり得るが、旧Source証拠を無条件流用しない |
| GP-03 同名二Node・多重Edge・逆方向・不適格ID | OccurrenceとEdge IDを失わず、不適格を明示拒否 | 同じ保持・拒否義務を、生成された実処理でも満たす |
| GP-04 取得不足、NULL、missing、空集合、Error | 型／取得範囲／未確立を汎用関数が適当に埋めない | Source契約と生成分岐のError／Read footprintが欠落しない |
| GP-05 演算実装の版・選択A→B | Model Revisionが不変でも現在実装の選択／呼出しと実行資格を再判定 | 生成器／依存Implementation／Artifactの必要部分を再資格化 |
| GP-06 選択B、実Call A、値だけ一致 | 選択と実Callの不一致で停止 | 選択Generator／Builder／実Consumerの不一致で停止 |

GP-01の「生成し直す必要がない」は、**Paramを固定したBundleや事前配布物を更新しなくてよい**ことまで自動保証しない。実Binaryを再利用できても、現在のModel／Param Revision、実Operator版・実Provider、選択Scope・HostのGrantは別に照合する。PとGで結果が一致しても、Source／Generator／実現Methodの資格は合算しない。

### 27.3 採否は同一の一問と、更新時の負担を比較して決める

まずOwnerが既存Standard Type／Role／Operatorで意味と境界を表せるか確認し、既存実装Pがあるなら**生成しない方式**を比較に含める。候補Pがない、Targetに載らない、未受理演算を任意スクリプトとして実行しなければならない場合は、Pを実装済み扱いにせずG／D／unsupportedへ明示的に振り分ける。

P/G両経路を実行できる対象では、同じSourceの受理済みMeaning、同じFixtureと独立Expected（[#1180](https://github.com/hat47x/tei-platform/issues/1180)／[#2514](https://github.com/hat47x/tei-platform/issues/2514)）で、**意味適合率、モデル変更時の再Buildの要否・所要時間、実呼出し回数、実行時間・メモリ、Target対応、手書きコード／設定の重複、デバッグ・Source位置追跡**を測る。値だけでなくNULL・Error・多重度・取得範囲の反例を揃え、性能値の単一閾値を先に定めない。実測がなければ採否は保留する。

セルフビルドの計測では、(a) **TEI由来の生成処理が実装する義務**、(b) **TEIが型付きパラメータで駆動した既存汎用処理の義務**、(c) **外部実装への委譲**、(d) **未対応**を分け、分母は対象義務を事前固定する。Pが優れている機能を不必要にコード生成しない一方、Pの利用数をGのコード生成・Plugin自己生成の実証率へ加算しない。再生成に伴う定義の簡素化・変更負担は別の指標として評価する。

**導入順**：既存RuleのGをR2-P正式Builderへ接続する最優先と、TEI Studio Stage1→Stage2セルフビルド、通常CLI実ProviderのR3は変更しない。この比較は[#1121](https://github.com/hat47x/tei-platform/issues/1121)（Lowering）、[#4544](https://github.com/hat47x/tei-platform/issues/4544)（Generator）、[#988](https://github.com/hat47x/tei-platform/issues/988)／[#951](https://github.com/hat47x/tei-platform/issues/951)（Operator／Plugin）、[#4198](https://github.com/hat47x/tei-platform/issues/4198)（Graph）、[#1180](https://github.com/hat47x/tei-platform/issues/1180)／[#2514](https://github.com/hat47x/tei-platform/issues/2514)（独立Expected）、[#2831](https://github.com/hat47x/tei-platform/issues/2831)／[#4583](https://github.com/hat47x/tei-platform/issues/4583)（セルフビルド）へ既存差分として渡す。**新しいDSL Grammar／Runner／自動Execution Grant／Runtime Registryは追加しない。**


### 27.4 図表の実装属性は、意味上の項目と混ぜない

設計図の詳細化によって「どの列をIndexにするか」「どのSQLをTargetに選ぶか」「どのPortをコンパイル時に束縛するか」が見えてくることは自然な設計の進展である。しかし、そのために業務Dataのすべての宣言へGoの型・関数・インポートやTarget固有のパラメータを露出させない。

図表の項目は、**業務上の意味・標準データ構造の制約・Provider／Targetの実現選択・表示設定**のどこに属するかをまず区別する。業務上のRuleやRelationのキーは既存Canonical、実現方法の情報は既存Target Generator／Plugin／Provider契約へ接続する。未対応の設定はCandidateとして残し、新しいCore属性や無制限の`key=value`袋へ自動採用しない。

例えば、ER図の「FK」は、明示されたRelationとキー対応の代用にならない。物理Indexの有無は論理Relationの同一性と別であり、Mermaidの矢印や線種、ノード画面の配線順から生成SQL・実Call順を推測してはいけない。実装属性を設計の知識として再利用できるようにすることと、図形から意味を推論することは異なる。

### 27.5 解釈と生成の同値性は、比較する観測面を指定する

現行の`validationeval.Evaluate`は`semantic.Validation`を純粋に評価し、`Source.Number`／`RelationKeys`へ**論理Read要求**を出す。他方、既存の参照`deterministic-transform-v1`は選択されたValidationと明示契約からGoを生成し、JSONアダプタを通して別OSプロセスで動く。両者は同一のCanonical IDと条件を共有できるが、入出力の外形は同じではない。

「同じ結果」の検査を次の範囲に分ける。

| 比較対象 | 一致と呼べる条件 | 今回の試験で主張しないこと |
| --- | --- | --- |
| **値・判定** | 同じ受理済みRule、型／評価時点／Scope、明示入力について`satisfied`／`not-satisfied`が独立期待値に一致 | JSONの全形式、SQL方言の全同値性、すべてのOperator |
| **評価不能** | NULL・未取得・型不適合を正常な`not-satisfied`へ押し込まない | InterpreterのGo errorと生成Goの`unevaluable`／診断コードの完全一致 |
| **Read／順序** | Interpreterが要求したField／Relationと、その必要集合・短絡を記録する | 生成Goの物理DB Read回数、RLS、未公開Dataの無取得 |
| **実現とビルド** | 生成Goの実byte、外部Buildと実Binaryの入出力を確認する | 製品Builderが同一Attemptでそのbyteを消費したこと、実Provider Call |
| **Effect／現在資格** | 実Provider、Operation、Receipt、Grantを各Ownerの実観測で確認 | 純粋判定の試験からの実行許可、成功したCommitの推定 |

今回`tools/reference-generator-selection/selected_markdown_rule_rebind_test.go`へ追加する`TestSelectedRuleInterpretedAndGeneratedRealizationsAgree`は、**同じ選択済みMarkdown Rule、同じ`semantic.Validation`と固定Expected**で、直接Evaluatorと既存Transform Generatorの生成Goを照合する。正値、小数、0、負数、負の0、明示NULL、未取得、型不適合をそれぞれ別Caseにする。生成Goは一度ビルドして同じBinaryに別入力を渡す。参照名が違うFieldを与えても黙って採用しない。

この試験のInterpreter側では、NULL・未取得・型不適合が同じ`nil`の数値として現れるため、**評価不能という粗い分類まで**を比較する。一方、生成GoのJSON Adapterは`null`／`missing`／`type-mismatch`の区別を独立に検査する。両者のエラー表現を文字列単位で等価と宣言したり、両者で物理Read Footprintが等しいと主張したりしない。対象となるデータ取得契約は実Provider／Context Ownerが別途検収する。

この「実現方式を変えても、何を保存したかを言える」性質は、宣言的構文汎化の中核的な検収条件になる。最終結果だけ一致して余分な非公開Fieldを先読みするDecision表への書換えは、評価順・Read／Disclosure契約の比較では不一致となり得る。結果一致を万能な変換許可へ昇格させない。

### 27.6 標準データ様式とノード表現への展開

構文を増やすよりも先に、**既存の一つのMeaningに対して、適用可能な実現方式と必要な不変条件を説明できること**を目指す。これにより、Node-RED型のFlow、ODIP型の大規模加工、業務Function、Rule／BRMS、Studio内部のPlugin配線を、用途ごとに異なる実現方式でつなぎ、無理に同じ実行モデルへ揃えずに済む。

表現の共通化には同じNode／Edgeを描画できることだけでなく、次を要求する。

- **構造**：GraphのNode・Edge ID、方向、多重性、端点、属性と、Tree／Tableへの投影損失が分かること。表示用EdgeとCanonical Relationを別に扱う。
- **評価**：入力の粒度、集約Scope、時点、NULL／欠損、データ取得の完全性、順序が分かること。ポートの型が同じだけでは結線を確定しない。
- **実行**：Data flow、判定、前提条件、Trigger、Effect、Plugin Implementationの選択が別の関係として追えること。選択されたProviderと実際のCall先を一致検証する。
- **出典と編集**：同じSourceの原byteとSelection、Meaning、表示由来のLossを保持すること。一意に逆対応できるSlot以外はread-onlyとする。
- **成果物**：解釈実行と生成コードのいずれでも、必要なSource・Contract版・Builder／Provider実消費・Consumer結果を問いに応じて取得できること。

Node Viewはこれらの情報への**共通の入口**であり、単一の実行Schedulerではない。画面からの編集やドラッグ配線は、既存Source Writerで一意の変更候補を作り、Preview→Resolve→Owner受理→Consumer再読取を通過できる範囲だけで認める。実際のSource書込み、権限・Effect、Bundle更新は既存Ownerが担う。

同じことはドキュメントにも当てはまる。Markdownの本文・小表・Mermaid・コードFenceを、専門家の理解順に配置できることを優先し、説明のための図・引用・表を自動的に実行可能な意味へ昇格させない。Graphviz、KaTeX、決定表、関数チェーンはそれぞれの適した表層を維持し、どのMeaningを表しているかと、必要な損失を明示する。

### 27.7 既存実装への引渡しと最小の次工程

現行mainでは、明示選択した`tei.authoring/v1`の項目定義表と同じMarkdown中のRuleを、製品`tei inspect`と共通`operations.InspectSubject`で読む試験が追加された（#5058）。旧Validation／選択Markdown Ruleの型付き一致、Generatorへの一因子変更、Rule由来の生成Goを通常`tei self-build`へ渡す試験（#5009）や原文byteの埋め込み確認（#5051）、生成WorkbenchのWindows PoC B（#5052）も存在する。**個別に成立した経路を、一つの製品Stage2セルフビルドや実Provider Invocationとして合算しない**。

次の最小工程では、既存のNode／Rule／Query／Plugin Viewに新しい共通Run Engineを追加せず、一Ruleのread-only照合票に次の列を一度だけ置く。

| 列 | 値の出どころ | 未確認の扱い |
| --- | --- | --- |
| Source／Role／Meaning | 製品Workspace、既存Source Index、Inspector | Sourceの位置が古ければ再読取。二重Definitionなら拒否 |
| 実現方式 | 既存EvaluatorまたはGenerator／Provider契約、実選択 | Interpreter・Generator・Pluginが選択不能なら未対応 |
| 評価契約 | 既存Validation／Operator、独立Expected、Read要求 | NULL、完全性、粒度、短絡が不明なら値同値だけを表示 |
| Artifact／Effect | 実Build Intent、Builder入力、Artifact、Provider、Receipt | 結果値だけでビルド消費・Commit・Grantを成功扱いしない |
| Consumer／編集 | 実CLI・Windows／Studioの受領、Source Writer資格 | GUIに表示できてもSource書込み・Stage2実現を主張しない |

試験の優先順は、**今回の「解釈 vs 生成」の一Rule同値検査 → #5056の生成GoとBuilder入力の取り違え拒否の統合判定 → Rule原文・Generator版・実Artifactの同一Attempt束縛 → 通常CLIから選択済み実ProviderへのCall／Readback → Studio Stage1→Stage2**である。PR #5056はこの追補時点では未マージであり、検収結果を先取りしない。標準データ構造・表現型・ノードUIの拡張を製品P0の新Hard Gateにはしない。

**終了条件**：新試験がexact HEADのGo／CIでPASSした範囲だけを認定し、実現方式の選択・契約の手作業調整・Reader／Generator実装の版固定・Provider Read・Owner受理・実Builder消費の各残差を明記する。新DSL Grammar、第二Semantic／Expected Store、独自Graph Runtime、TEI専用Hostは導入しない。

## 28. Data表と隣接Ruleの二実現方式を製品Loadから検証する（2026-11-15）

### 28.1 §27のG／P／D判断を一つの既存Fixtureへ適用する

§27の**G：生成／P：汎用パラメータ実行／D：外部委譲**という分類を、別のDSLや判断エンジンを設けず、既存の製品実装で確かめる。対象は、選択済みtei.authoring/v1で読む異動申請のMarkdownである。印付きData項目表のallocation（std.decimal）と、隣接するallocation_positive Ruleを同じWorkspace.Loadで取得し、Source Index／InspectorのAttribute・Ruleの出典位置を確認する。

今回のFocused Go試験はinternal/operations/inspect_markdown_data_table_rule_evaluation_test.goに置く。**二つの実現方式が同じ型付きRuleを受け取る範囲だけ**を検証し、プレビュー可能なP経路をGのセルフビルド実績として数えない。

| 段階 | 実際の確認対象 | 到達点の上限 |
| --- | --- | --- |
| **Meaning** | Data項目表→既存semantic.Attribute（decimal）と隣接Rule→semantic.Validation。元byteとRoleDefinitionを一Loadで照合 | Reader成功・型一致は実データ取得／Ownerの業務意味受理を保証しない |
| **P：汎用評価** | 既存validationeval.Evaluateへ同じValidationを渡し、正値／0／負値／NULL／未取得の五入力とNumber Read一回を独立Expectedで検証 | テスト用の厳密なSourceであり、製品DB／実Providerを呼んだ証拠ではない |
| **G：Go生成** | 同じValidationとAttribute型を既存validationgen.GeneratePositiveへ渡し、生成SourceのNumber参照先と正値分岐を照合 | このFocused試験はGoのコンパイル・実Binary同値を検証しない。#5009／#5051等の別の実証を合算しない |
| **一因子の型変更** | Data項目表のallocationだけをdecimal→textへ変更。Rule本文・Rule IDは維持し、Validatorの非数値拒否とGo生成拒否を確認 | 不適合をP／Gどちらかの暗黙Fallbackで隠さず、旧Source Index／Worldの現在性も流用しない |

### 28.2 反証と次の一手

| 対象 | 一因子変更 | 失格となる短絡 |
| --- | --- | --- |
| **DC-39／GP-04** | 同じRuleの入力を正値→0→負値→NULL→未取得へ | 未取得をfalseとして受理したり、異なるAttributeを読んで結果を合わせたりする |
| **DC-40／GP-01** | Ruleをそのままに、表の対象Attribute型だけをdecimal→textへ | Rule ID／Source位置の一致で不適合な解釈・生成を許す |
| **DC-41（次段）** | 同じ結果を返すPとGで、実Read先が異なる | 値一致だけで実Binary／Read Footprintの同値を宣言する |

DC-39〜40は**試験ファイルへ記述した期待条件**であり、exact HEADの実Goテスト成功はCIで確認する。DC-41は未実装の独立Expected候補で、#1180／#2514のOwnerへ渡す。

次の製品検証は、PとGを同一入力・同一型付きRule・同一Source revisionで**実際に呼び出し**、値だけでなくRead／Errorと実Attemptを比較すること。単純な汎用関数が既存実装で足りるときは生成を省く。一方、外部Targetへの納品やTEI自身の実装生成には正式R2-Pの実Builder・Artifact・独立Consumerが必要である。R3の公開Function→実Provider、Studio Stage1→Stage2実Windows受領は並行P0に保つ。新Executor／Resolver／第二Meaning／第二Expected Storeは作らない。

## 29. 値の一致と実際の読取対象の一致を分ける（2026-11-15）

### 29.1 G／Pの二方式をData表から実Binaryまで照合する

§28の回帰は、Markdown項目定義表と隣接するRuleを同じ型付きMeaningとして選択し、P（既存Evaluator）の実判断とG（既存Go生成器）の生成Sourceを別に確かめた。しかし生成コードの見た目だけでは、**実際に動いた生成Binaryが正しいAttributeを読んだか**は証明できない。

今回の追加試験は、internal/operations/inspect_markdown_data_table_rule_binary_parity_test.go である。製品が明示選択したtei.authoring/v1から異動申請のallocation（std.decimal）とallocation_positive Ruleを、同一Load／Source Index／Inspectorで取り出す。同じ型付きValidationを既存EvaluatorとGeneratePositiveへ渡し、生成Goを実コンパイルして別OS Processから実行する。

| 観測 | P：既存汎用Evaluator | G：生成Go実Binary |
| --- | --- | --- |
| **意味と出典** | 同じ選択済みRule・属性ID・型・元SourceのRoleとFingerprint | 同じ型付きRuleから生成したGo。正式Build Intentはまだ使わない |
| **独立Expected** | 2.5／0／-1／-0／NULL／未取得の六入力を評価 | 六入力で判断とError有無を比較 |
| **Read契約** | Numberの読取先ID・呼出回数を観測 | Binary内部のSourceが実際に受け取ったID・回数をJSONで返す |
| **反証** | 正しいallocationへのReadを固定 | 生成GoのRead先だけperson_idへ変更。同じ値を返せても実ReadのID不一致で拒否 |
| **未確認** | 実DB／実Plugin ProviderのRead、Host Grant | 製品tei self-build、正式R2-PのBuilder／Artifact閉包、外部Consumer |

**同じ判断結果が出ても、読んだ対象が違えば意味上の置換許可は成立しない。** SourceのRole、Type、Input、Read、Effect、実行資格をそれぞれ確認する。この区別はPluginの複合型Port・Field単位・明示Adapterにも適用可能だが、本試験からGraph多重性、時点、取得完全性、開示条件の合格を推定しない。

### 29.2 方式分類と実現配置は直交する

§27のG（生成）／P（既存関数のパラメータ評価）／D（外部委譲）は**実現方式の分類**である。図表からの解釈・生成・混合やHookは、同じ業務意味をどの実装へ配置するかという別の判断軸である。「Pだから常駐Interpreterを新設する」「Gなら必ず納品Targetで実行された」「Hookを選べば選択Providerを実Callした」とは認定しない。

方式の切替えは、SourceとMeaningが同じだけで自動受入されるものではない。型・粒度・時点・入力完全性・Read／Disclosure・Effect・Targetを比較する。確認できない契約は未観測、非対応の方式はunsupportedとし、別方式への無音Fallbackを禁止する。Artifact再利用、Source鮮度、現在Grant、Owner受理も別票とする。

新しい共通RunnerやRealization Registryを追加せず、既存のRule・Source Index・Generator・Evaluator・Builder・Provider・Consumerの資格と、その接続の不足だけを示す。

### 29.3 独立Expectedの引渡し

| ID | 一因子反証 | 判定すること |
| --- | --- | --- |
| **DC-42** | 同じData表／Ruleと六入力をP／生成Go実Binaryへ渡す | 値／Error／Read ID・回数を独立Expectedと一致させる |
| **DC-43** | GのNumber参照先だけ別Attributeにし、テストSourceは同じ値を返す | 結果が同じでも実Read IDが違えば同値を拒否 |
| **DC-44** | Provider実体、時点、取得範囲またはAudienceだけ変更 | このFocused試験では未観測。実Provider／ConsumerのOwnerへ渡す |
| **DC-45** | GからPへ変更し、旧ArtifactやGrantを再利用する | 実現選択とArtifact再利用／実行受理の資格を分ける |

DC-42／43は新Go試験の受入対象とし、ファイル収録とexact HEADのGo／CI成功を混同しない。DC-44／45は#1180／#2514の独立Expected候補で、未実装の段階では未実施と扱う。次段は正式Build Intent→選択Builder入力→Artifact→独立Consumerと、公開Entry→選択Provider Callを別経路で実証する。Studio Stage1→Stage2も独立P0を維持し、速度向上やコード量削減を未計測のまま数値化しない。

## 30. BuildPlan確定後の元宣言改版を、実self-buildで拒否する（2026-11-15）

### 30.1 生成Goと元Markdownは、別々に変化し得るBuilder入力である

§29で検証対象にしたG／Pの結果・Read一致に加え、**元宣言の記述的な改版が、生成Goのbyteを変えない場合**を扱う。業務判断が変わらなくても、Source資格とBuildPlanが同一であるとは限らない。既存の選択Markdown Rule（sales.order_line.quantity_positive）と、通常のtei self-build CLI・Build Intent・Go Builderをそのまま使う。

既存のcmd/tei/cli_self_build_generated_rule_go_test.goでは、(1)元Markdownの説明変更でBuildPlanのSource Closureが変わること、(2)BuildPlan確定後にgenerated.goを変更した際は旧Planでの通常self-buildを拒否すること、(3)戻した入力から実Binaryを生成して判断・同梱原文Digestを照合することを個別に確認している。今回、その隙間に**元MarkdownだけをPlan確定後に変更するケース**を追加する。

| 変更・時点 | 既存／今回の実検証 | 許されない短絡 |
| --- | --- | --- |
| 元Markdownの説明だけを変更し、BuildPlanを再解決 | Source Closure／Plan Fingerprintの違いを確認（既存） | 判断結果が同じだからSource改版を無視 |
| **Planを固定した後、生成Goはそのままで同梱Markdownのみ変更** | **旧plan.jsonを使った通常tei self-buildを実行し、失敗・L2-b非Claim・成果物未公開を要求（今回）** | 新Sourceを旧Plan・旧Artifactの資格でビルド |
| Planを固定した後、生成Goのみ変更 | 旧Planを使った通常self-buildを拒否（#5109） | Goがコンパイル可能なら旧入力を許容 |
| 元Markdownと生成Goを復元した後 | BuildPlanの記述的Fingerprint一致を再確認し、既存の実Builder・Binary判断へ進む | A→B→Aで旧Grant／業務意味受理まで自動復活 |

新しいGo回帰は既存テスト内への一因子追加であり、Generator・Reader・Builder・Hostを増やさない。BuildPlanの一致・拒否は**ビルド入力の一致に関する技術的証拠**である。元MarkdownのLocator／Reader版／Generator版・契約を一つの正式Attemptへ完全束縛したR2-Pの証明、現在Grant、業務Ownerの受理、R3実Provider Callは引き続き別の資格とする。

### 30.2 G／P／Dと型付きPlugin配線への引渡し

Gの既存Artifactを再利用する判断と、Pの再評価を行う判断は別である。Sourceが変わってもCanonical Meaningが同じなら、意味の再評価・生成Go再生成・Artifact再ビルドを必ず一括実行する必要はない。しかし、**必要なSource資格・Builder消費・Provider実選択・Audience・Effectの根拠を取り直す前に**旧証拠をそのまま流用してはならない。

Plugin間の複合型全体／Field／明示Adapter（#5101）では、型の一致と実際に渡した項目・取得範囲・開示の証拠を分ける。Operator実装切替時のScenario Pin（#5114は別PRで審査対象）も、同じ「実現選択の鮮度」の問題だが、今回の元Markdown byte差と混合しない。Sourceの宣言内容と、実際に利用したImplementationの選択が独立に変化するためである。

| ID | 一因子反証 | 今回の受入範囲 |
| --- | --- | --- |
| **DC-46** | 生成Goを固定し、BuildPlan解決後に同梱元Markdownの説明だけ変更 | 旧Planの通常CLIビルドは拒否、L2-bを主張せずArtifactなし |
| **DC-47** | 生成GoだけをBuildPlan解決後に変更 | #5109の独立既存検証。元Source変更の証拠に合算しない |
| **DC-48** | SourceをA→B→Aに戻し、旧Grant／Owner受理を再利用 | 記述的Fingerprintの復帰と資格・歴史の復帰を区別（未実装のOwner反証） |
| **DC-49** | Gの実Build入力とPの意味判断が一致しても、選択Provider／Disclosureが異なる | 実Provider・独立Consumerによる別検証（未実装） |

DC-46は今回のFocused Go試験に記述する期待条件で、**exact HEADのGo／CI合格は別途確認**する。DC-47は既存#5109の結果と混同せず、DC-48〜49は#1180／#2514およびSource／Builder／ProviderのOwnerへ引き渡す。生成・汎用処理の選び分けを理由に第二Evidence Store、Source Writer、Runtime、Registry、新P0 Hard Gateを導入しない。

**P0は従来どおり**、通常CLI公開Entry→実Provider→別Consumerと、Studio Stage1→Stage2の実Windows再構築である。

## 31. Ruleの意味が増えても、実現方式を暗黙に増やさない（2026-11-16）

### 31.1 受理されたRuleと実装できるRuleは同じ集合ではない

他レーンの製品実装では、値がない理由を区別するDecide（#5142）、記録の例と期待結果の照合（#5143）、同じ記録の暦日を比較するnot-after Rule（#5146）がmainへ導入された。これは、§27のG（生成）・P（汎用評価）・D（外部委譲）の選択基準を、positive一種類から拡げて検証できる段階になったことを示す。

しかし、**意味モデルがあるRule種別を受理したことは、すべての実現方式がその種別を実装できるという意味ではない。** 現行のGeneratePositiveはpositive／std.decimalに限定している。暦日比較を誤って数値の正値条件へ落とすよりも、未対応を明示して停止する方が正しい。

| 比較点 | 正値Rule | 同一記録の日付前後Rule |
| --- | --- | --- |
| Markdown Source | 印付きData表とteiコードフェンスでRuleを定義できる | 同じ表層でstd.date二項目とnot-after／if presentを宣言できる |
| 型付きMeaning | 既存ValidationPositive、対象Attributeはdecimal | 既存ValidationNotAfter、開始／終了の二AttributeとIfPresent |
| P：既存評価 | 数値を一件読む既存Evaluatorで評価する | StatefulDateSourceを通じて二日付を読み、NULL／未取得を区別してDecideする |
| G：既存Go生成 | GeneratePositiveが限定的に対応 | GeneratePositiveはunsupported generated kindで拒否し、Goを出力しない |
| D：外部委譲 | Providerの実選択が別途必要 | Provider選択・実Calledの根拠なしに代替しない |

この確認は「Pが全Rule種別へ対応する」という主張でもない。特に、関係全行を扱うRuleではCoverageがpartialなら判断できない場合がある。反対に、読み取った範囲だけで不成立が確定する場合は、未取得の行があっても不成立と判断できる。単純に空をfalse、partialをcompleteに置き換えない。

### 31.2 一つの製品Readerを使う対照試験

Focused Go試験 internal/operations/inspect_markdown_data_table_date_rule_realization_test.go は、既存異動申請のMarkdown項目定義表の日時型を、テスト専用コピーで暦日型std.dateへ変更する。そのうえで、同じteiコードフェンスに日付の前後Ruleを一件追加し、既存tei.authoring/v1・Workspace.Load・Source Index・Inspectorを使う。

試験では、同じ文書の正値RuleはGを維持するが、新しいnot-after Ruleは現在のGでは拒否されることを固定する。Pは、早い日・同日・遅い日、終了日NULL、未取得を含む入力について、判断と読取Attribute二件のID・順序を独立Expectedで検査する。

別の一因子反例では、暦日項目をstd.datetimeに戻す。Readerが構文を読めても、**暦日比較に日時型を混ぜたRuleは既存Validatorが拒否する**ことを確認する。実行可能かどうかをGo生成器の都合で判定したり、逆にValidatorの成功だけで外部ProviderやBuildを許可したりしない。

### 31.3 次段はRuleごとの利用可能な実現方式と不足理由の表示

既存の製品Inspectorが返すSubject・Canonical・Source Reference／Role・Fingerprint・AuthorityLimitsを入口とし、実現の適合性は実装契約に照らして別に表示する。現時点では新しい公開フィールドやRealization Resolverを作らず、次の差分として取り扱う。

| ID | 一因子の反証 | 今回／次段の扱い |
| --- | --- | --- |
| **DC-50** | Data表に正値Ruleと日付前後Ruleを併記し、同じReaderで読む | 共通Meaning・厳密Source・異なるG対応範囲をGo回帰へ追加 |
| **DC-51** | not-afterの終了日をNULL／未取得にする | IfPresentと状態・読取先に従って判定／未判定を区別 |
| **DC-52** | 暦日比較の対象属性一件をstd.dateからstd.datetimeへ変更 | Ruleを構文として読めても、意味の型検査で拒否 |
| **DC-53** | Pでは判定できる日付Ruleを、G／Dへ暗黙に切り替える | 未対応Gは拒否。Dの実選択／実Callは未実装の独立検証候補 |
| **DC-54** | 記録例が同じ結果だが、Ruleが要求する取得範囲を一部省く | 例の一致を入力完全性やRuleのLaw全面保証に昇格させない（別Owner） |

DC-50〜52は新Go試験の期待に対応するが、**コード収録・Goテスト合格・Owner受理は別に判断する**。DC-53〜54は#1180／#2514の独立Expectedへ渡す。

生成Goの正値比較や以前のG／P実Binary同値の結果を、日付RuleのGo実行成功へ流用しない。新しい第二Generator、Rule Engine、万能Scheduler、Runner、Host、Plugin Registry、Source Writerを作らない。通常CLI実Providerへの呼出し、正式Source→Generator→Builderの同一Attempt束縛、Stage1→Stage2実WindowsのP0を維持する。

## 32. 記録例はMeaningを増やさず、期待と出典の検証を増やす（2026-11-16）

### 32.1 同じMarkdown内でも、定義と例は独立した情報である

Ruleの表現力が増え、値の未取得・NULL・部分取得、日付前後、関係の期間、Operator選択を扱う際には、**Ruleが何を意味するか**と、**どの記録例にどんな答えを期待するか**を混同しない。Readerで読めた表、定義のAnchor、記録の観測、期待のAnswer、Evaluatorの実際の判断は異なる役割を持つ。

mainへ統合済み#5142（Decide）・#5143（記録例の期待照合）・#5146（日付Rule）を、既存の「印付きData項目定義表と隣接Rule」の製品Fixtureへ接続する。ここでの実証は、新しい文法・第二Meaning・例専用Runnerを必要としない。

| 同じMarkdownの要素 | 保持するもの | 保持しないもの |
| --- | --- | --- |
| Data項目定義表 | Data／Attributeの型付きMeaning、SourceのDefinition | 記録例の値を実データや追加Dataとして登録しない |
| 隣接するRule | 既存Validationと厳密な対象Attribute、Definitionの元byte位置 | Rule IDから公開Function／Capabilityを作らない |
| 印のない参考表 | 人が読む説明 | 自動的なData／Rule／例として取り込まない |
| 印付き記録例 | 行・期待したRuleの答え・期待セルの元byte位置 | Canonical Worldの変更、Ownerの受理、実DBデータの証明 |
| 例との照合結果 | 検査したRuleの答えと、違う場合のRULE_EXAMPLE_MISMATCH | 未実行Providerや、例に存在しない入力全体の正しさ |

したがって、記録例の期待だけを修正しても、World FingerprintとData／RuleのCanonicalは変わらない。一方、Sourceの原byteとSource Index Fingerprintは変わり、診断結果が変わる場合がある。この状態を「意味は同じだから古いSourceを使える」と扱わない。また、例に書いた期待が一致しても、実Builder入力・Plugin選択・現在Grantが受理されたことにはならない。

### 32.2 製品の一文書から定義・例・診断まで観測する

`internal/operations/inspect_markdown_data_table_rule_example_validation_test.go`では、従来の異動申請文書に、明示的に印を付けた記録例を追記する。既存の`tei.authoring/v1`／Workspace Loadを用い、正値、ゼロ、NULL、負値の四行について、Ruleの期待を独立に記述して検証する。

同一Workspaceの製品InspectorとValidateWorkspaceで確認する点は三つである。第一にData／RuleのCanonicalとDefinitionのSource位置はそのままであり、例だけが別のAuthoring情報として読み取られる。第二にゼロの行の期待を「不成立」から「成立」へ変更すると、Rule定義ではなく**期待を書いたセル**の位置に一件の診断を返す。第三に、値を空欄に書いたNULLと、例の表に項目列そのものを書かなかった未取得を別に扱い、判定不能の理由を保存する。

この試験で実証するのは、**製品Reader／Inspector／例のEvaluator／診断投影の局所的な接続**である。実Record Provider・実DBの取得完全性、Studioでの編集位置確定、Ruleの全入力に対する妥当性、独立Consumerの受領は別に検収する。

### 32.3 選択資格・取得資格への引渡し

| ID | 一因子反証 | 受入の判断 |
| --- | --- | --- |
| **DC-55** | 同じData表とRuleへ、期待が合う記録例だけを追加 | World／Canonicalは維持、Source Indexは改版。例はDefinitionへ昇格しない |
| **DC-56** | ゼロの行の期待セルだけを不成立から成立へ変更 | 診断を期待セルの元byte位置へ返し、Rule Sourceの定義位置を誤って指さない |
| **DC-57** | 空欄の値を示す例を、項目列そのものがない例へ変更 | NULLと未取得を別の理由として扱い、いずれも偽の値へ補完しない |
| **DC-58** | 期待が一致する一例だけを根拠に、P／G／D・Grant／Audienceまで成立と判断 | 局所例の一致と実実装選択・実Call・取得範囲・開示を分離（後続の独立検収） |

DC-55〜57はGo試験に書く独立Expectedであり、**exact HEADのCI合格とOwner意味受理は別**に判定する。DC-58は#1180／#2514、Rule／Provider／Studioの各Ownerに引き渡す。

なお、#5148（期間の非重複Rule）、#5149（Contract Queryから取得状態を申告するReader）、#5150（Operator選択Fingerprintの共通化）、#5168（Studioの後着応答とPin）、#5156（PluginのRecord／Field投影）は、現時点でmainへ統合済みと仮定しない。各PRが実際に採用された版とConsumerから検収する。特に取得した値の状態と取得・開示の許可は別の資格であり、SUI等の文書取得やUIへの返却でも版と現在Grantを改めて確認する。

製品P0の正式R2-P出典・Generator／Builderの同一Attempt、通常CLI公開Entry→実Provider→独立Consumer、Studio Stage1→Stage2の実Windows再構築は維持する。新しい万能Engine・Evidence Store・Graph Runtime・Reader Registry・Hostを追加しない。
