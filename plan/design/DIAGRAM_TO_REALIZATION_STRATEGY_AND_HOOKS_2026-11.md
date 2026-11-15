# 図表から実装へ：解釈実行・コード生成・委譲・フック拡張の選択設計

- **状態**：設計・検収計画。2026-11-15。新しいDSL構文・汎用Runtime・Placement APIの採用、実Targetでの動作、業務意味のOwner受理は主張しない。
- **目的**：`*.tei.md`の図表と定義から、必要に応じて既存の汎用Functionによる解釈実行、Targetコード生成、DBや外部Providerへの委譲、その組合せを選ぶ。生成物は原則ノータッチとし、既存アプリケーションには契約で定めた接続点を通して取り込む。
- **関連**：[図表→Build縦断](TEI_MD_DOMAIN_DIAGRAM_TO_BUILD_LOWERING_2026-11.md)、[Critical Lowering](CRITICAL_LOWERING_CONTRACT_DESIGN.md)、[生成器契約の境界](GENERATOR_CONTRACT_BOUNDARIES_DESIGN_2026-10.md)、[Generator選抜](AI_GENERATOR_AUTHORING_STANDARD_2026-11.md)、[意味と宣言的合成](DECLARATIVE_COMPOSITION_AND_NODE_SURFACE_CONTRACT_2026-11.md)、[成果物要求の複数実現](DECLARATIVE_RESULT_PROCESS_COMPOSITION_2026-11.md)、[Plugin実現と最適化](PLUGIN_REALIZATION_AND_OPTIMIZATION_REQUIREMENTS_2026-11.md)、[実装→モデルdogfooding](IMPLEMENTATION_TO_MODEL_DOGFOOD_2026-11.md)、[Pluginの第一縦断](PLUGIN_RULE_PRODUCT_VERTICAL_SLICE_2026-11.md)、[Studio部品生成](../../tools/reference-studio-part-generator/README.md)、[ADR-0045（提案中）](../adr/0045-generator-artifact-lifecycle-and-evidence.md)。
- **Ownerとの境界**：Reader/Languageは#2621/#988、Semantic/Roleは#2619/#2620、Loweringは#1121/#4544、Plugin/Contract/Hookは#951/#1016/#4198、Generator選抜とSource束縛は#2595、BuildとSelf-buildは#943/#2831、独立Expectedは#1180/#2514、Studio実Consumerは#1824/#4583。第二Parser、第二Semantic、別のBuild/Plugin Registry、TEI独自Hostを設けない。

## 1. 結論：実装方法の選択も、設計の一部である

図表に実装上の属性が増えることは、必ずしも「設計が技術に汚染された」ことを意味しない。型、粒度、更新競合、読取りの完全性、再試行、Hookの位置など、詳細化して初めて見える要件を発見した可能性がある。まずその属性が**業務意味の制約**、**実現に必要な保証**、**Target固有の実現方法**、**表示用設定**、**再利用できるPatternのパラメータ**のどれかを見分ける。この発見を元の設計へ戻して検討できることがTEIの強みである。

実現方式は、アプリケーション全体に一律ではなく、Function／Rule／Operationとその選択Target・Contract単位で比較する。

| 方式 | 主な役割 | 採る条件 | 生成しないもの |
| --- | --- | --- | --- |
| **I：解釈実行** | 既存の標準Function／Evaluatorが、受理済み定義のパラメータを読み動作する | 演算・型・失敗・性能・Target・作用境界を既存Function契約で満たせる | 個別の業務判断用ソースコード |
| **G：静的生成** | 受理済み意味をTarget-nativeなコード・設定・ResourceへLoweringする | 専用最適化、コンパイル時結合、Target固有API、配布時の定義非同梱等が必要 | 実行時に共通Evaluatorが毎回構造を解釈する処理 |
| **D：委譲** | DB、外部Process、API、Plugin等を契約で呼び出す | Target側で型・精度・時点・完全性・権限・Effectを満たせる | 委譲先の判断ロジックのTEIでの実装 |
| **H：混合** | 解釈・生成・委譲をFunction／Operationの境界で合成する | 境界を越えて型・粒度・Scope・Failure・Effectを保存できる | 全領域を一律の方式に揃える必要性 |

**既存Evaluatorは早めに調べるが、自動採用の既定値にはしない。** 選択順は(1)Meaning／Scope／Effect／型など必須条件への適合、(2)Targetと実Providerの解決、(3)性能・更新・配布・保守費の比較とする。Iで十分なら個別Generatorは不要だが、DやGが適するケースもある。複数候補が残る場合、費用順位で暗黙選択せず既存Resolverに戻す。未対応を別方式へ黙って切り替えず、候補の存在も意味の正しさの証明にはしない。

「生成工程を飛ばす」のは**業務判断のコード生成**である。必要に応じて、定義の正規化、実行用パラメータ束のPackaging、Pluginの静的収録、既存TargetのBuild・配置は残る。Iを選んだからTEI常駐Hostが必要になるわけではない。Go単一バイナリには既存Evaluatorを静的に組み込み、外部Java/.NET HostにはTarget-nativeな標準Library／Adapterを配布できる。

## 2. 共通入口と分岐

```text
*.tei.md（原文、図表、表、明示的な定義領域）
  → 選択済みReader / Source Index / typed Meaning / Acceptance
  → 対象Operation・FunctionとRequired Contractを解決
  → 実現候補の適合・実体・費用を照合（I/G/D/Hは設計用の分類）
     ├─ I：標準Function＋版付きパラメータ束
     ├─ G：Generator → 対象言語の生成物 → Placement
     ├─ D：Query／外部Provider → 実Call
     └─ H：各方式の明示的な合成と接続境界の検証
  → 既存Composition / Build Intent / Artifact / 実Target
  → 実ConsumerでのRead・Decision・Failure・Effectを観測
```

分岐前のAccepted Meaningを共用し、実行方法の差で別の業務正本を作らない。Iは「解釈器が自動的に意味を正本化する」経路ではない。定義やParameterが新しくなれば、実際に読み込んだ版と実行主体を照合する。GではGenerator Source、生成物、Builderが読んだbyteを別々に固定する。Dでは変換したQuery／公開Contractと実Provider、入力Scope・時点・Effectを別々に照合する。Hでは各経路の接続点を固定する。

## 3. 図表の属性は、発見された要件の置き場所を確認してから増やす

| 発見した内容 | 置き場所と扱い | 例 |
| --- | --- | --- |
| 業務意味そのもの | 既存Semantic／Rule／Relationへ受理候補として追加 | 重複不可、金額丸め、行の有効期間 |
| 実現に必須の保証 | Contract／Trait／Operation要求へ。解決不能なら未対応 | 競合検出、原子的Commit、欠損値の扱い |
| Target固有の選択 | Realization／Composition設定。意味正本は変えない | Goパッケージ、DB方言、Evaluator版、生成モード |
| 設計・表示の補助属性 | 図のView Profile／説明。意味を変えない | レイアウト、色、折り畳み |
| 繰り返す実装知識 | Pattern定義とSkillの手順。採用・実行の正本ではない | ヘッダ・明細の検証、UI一覧の操作、端数処理 |

通常のMarkdown表やMermaidの装飾へ項目を加えただけで実行意味へ昇格させない。正式に読む範囲はWorkspaceで選択されたLanguage／Roleが決める。生成用の属性も、**明示的な対象IDと版付き定義**を持つ場合だけReaderが消費する。図が複数のOccurrenceを表示しても、対象のIdentityは統一し、表示上のノード名や位置で同一視しない。

特に次を分ける。ビジネスに見える「この値を変更できる」は業務上の許容範囲であり、「GeneratorがこのHookを提供する」は実装上の接続契約である。両者が重なる場合にも、前者が後者の実行権限を与えない。

### 3.1 設計の進展として取り込む手順

1. 実装上の発見を、元の図表に対する**変更提案**として記録する。既存の意図、求める保証、Target固有の理由を分ける。
2. すでに存在する型・Role・Operator・Pattern／Traitへ写像できるか確かめる。表現できない部分だけ候補属性に残す。
3. 表示だけの補助属性、業務意味の変更、実現設定の変更で、失効させるRevisionを変える。
4. 一つのTargetにだけ必要な配置情報を他Targetの業務DSLに強制しない。必要な場合は明示的なTarget側設定へ置く。
5. Sourceの最小差分をPreviewし、保存後は実Readerで再読込する。意味のOwner受理とBuildを別段階で扱う。

**未採用の構文例（概念図）**：

```text
Rule: sales.order_line.quantity_positive
Meaning: quantity is positive
Implementation choice: interpret | generate | delegate | hybrid
Requirement: decimal with missing / null / error preserved
Binding: selected validation contract and provider
Extension: declared validate hook (optional)
```

これは既存の`tei.rule-authoring/v0`へ追加済みの文法ではない。今ある`*.tei.md`では選択された`tei`フェンスだけが意味として読み込まれる範囲がある。構文を先に増やすより、参照用Fixtureと型付きモデルとの対応を検証する。

## 4. 生成物はノータッチ。拡張は契約済みのHookへ

生成ファイルを手動・AIで直接編集しないことを原則とする。例外的なMigration／Read-only観測は別途明示し、通常の再生成契約には含めない。

- **標準コード**：Generatorが完全に所有する。生成時にbyte digest、Generator/Rule版、実Targetの依存を保持。
- **案件固有の実装**：生成物の外に置き、契約に宣言されたExtension Pointへ接続。人・AIの編集許可は書込み窓の契約で限定。
- **Hook契約**：ID、version、Phase、入力／出力型、呼出し条件・回数の上限、順序、Failure、Effectの可否、Scope、必要な保証と検証を宣言する。
- **既定動作**：案件実装がないときも意味が明確である。Hookが存在しない場合の暗黙の許可・黙ったスキップを避ける。
- **合成後の実効フロー**：標準フロー、差分、合成後の順序を相互参照可能にする。Hookを追加しても標準Guard、認可、Audit、Receiptを迂回できない。

生成器が「好きな位置へコードを挿入する」という能力を持つ必要はない。まずは**別ファイルと明示されたBinding**を標準とし、必要なときだけ特定TargetのAST編集を使う。行番号・名前の偶然一致・無限定なテキスト置換では注入しない。

### 4.1 Hookを安全に取り扱う条件

`before-validate`／`after-validate`／`transform`などの純粋段階と、`before-commit`／`after-commit`／`notify`などの作用を伴う段階を同列に扱わない。Hookが外部作用を行う場合、Guard拒否以前に呼ばれないこと、事前のGrant、呼出回数、失敗・結果不明、再試行時のReceipt照合を実Providerで確かめる。標準OperationのEffect契約をHookで上書きしない。

「一つの試行で想定回数だけ呼ばれた」ことと「障害・再送を含めて厳密に一度だけ効いた」ことも別。後者は外部対象の冪等性やReceiptで評価する。

既存`tools/reference-generator-extension-points`は読取り専用の契約・差分検出参照であり、実際のHookを適用する製品機能ではない。この境界を保って実Targetへ拡張する。

## 5. 生成物の配置と接続

| 組込み方法 | 原則 | 受入条件 |
| --- | --- | --- |
| 新規ファイル／Package | **第一候補**。同一Go Packageの生成ファイル、別Packageの明示Import、既存契約への静的Bindingなど | 対象Directory／Module／Build入力に実際に収録される |
| 標準FunctionへのパラメータBinding | Iの第一候補。モデル由来のParameterを版付きArtifactとして収録／実読込 | 選択EvaluatorのID/版、パラメータ、実Read、結果を照合 |
| 生成されたAdapter／Registration | Hまたは静的Plugin合成 | 選択Contractと実Providerの一致、Runtimeに不要なGeneratorを同梱しない |
| 型付き構文編集（AST等） | 必要時のみ。Sourceを一意に解決し、所有範囲を限定 | 既存コメント・書式・Import等の保存性、失敗時無変更、差分Preview |
| 生成ファイルの保護領域・文字列置換 | 例外／移行用途。新規標準にはしない | 所有境界と競合検出、移行先を定義 |

組込み先を解決するにはCanonical ID、選択されたTarget Contract、Target Symbol、現在のSource Locationを分ける。ファイル名やコード行番号を業務Identityにしない。同名関数・複数候補が残る場合は停止する。Source A→B→A、別Module、別ビルド版、Providerの差し替えも、byte一致から資格を継承しない。

複数Generatorが同じTarget Symbol／成果物を変更する場合は暗黙の実行順で勝者を決めず、所有権・依存・合成可能性を先に照合する。適用は作業領域で準備→Revision再確認→差分検証→一括公開を目指す。これは現在の製品で完成した原子的更新APIの主張ではない。

## 6. 汎用Function、Pattern、Skillの適切な役割

同じ目的を満たすものでも、責務は違う。

| 層 | 役割 | 例 | 避けること |
| --- | --- | --- | --- |
| 標準Function／Operator | 定義値を解釈して確定した契約で動く実装 | typed Validation、Dataの写像、定型的なUI表示 | 実行時に意味・権限を推測する |
| Pattern | 対象・必要な保証・設定可能点を示す再利用可能な設計知識 | ヘッダ明細保存、承認手順、API一覧 | 全案件に同じ業務意味を強制する |
| Skill | 人・AIが対象確認、方式選択、差分、負例、引渡しを行う手順 | 「生成か解釈かの選択」「Hookで拡張する」 | Skill本文を実行された証拠とする |
| Generator | Target-native成果物を実際に生成する版付き実装 | Go Rule、UI Component、DB Migration | ケース固有知識を直接Coreへ固定する |

Patternは**適用前の設計候補**、Skillは**進め方のナレッジ**、Functionは**振る舞いを担う実装**である。似た案件が二回現れたらまずPattern／Skill化を検討し、実行契約が定まって共通化できる箇所だけ標準Function化する。反復数だけでCore昇格を決めず、未対応例・性能・失敗条件を比べる。

最初の候補Pattern：
- **P-RULE**：純粋な検証／決定。Iを先に試し、分岐と型をTargetで固定する必要が出ればGを比較。
- **P-UI**：宣言的な一覧・フォーム。汎用UI Componentによる解釈を先に評価し、標準外の表示や型付きBindingは限定的に生成。
- **P-DATA**：Data Relation／Query。入力ScopeとNULL／多重度を先に固定。Native SQLや方言差はOpaque／Target別へ逃がす。
- **P-FLOW**：Process／Lifecycle。Graphは順序・条件を記述するが、実行は既存Operation／Guard／Providerへ接続。無条件に万能Graph Runtimeを設けない。
- **P-PLUGIN**：Plugin配線。型・Contract・Phase・Effectを比較し、静的収録／既存Process Providerの選択はCompositionに委ねる。

## 7. 複数の実現方式の意味保存を値の一致より広く試す

共通の独立Expectedを先に固定し、IとGをそれぞれ**実Consumer**で検査する。DB／外部Providerへの委譲は対象のSnapshot、Scope、型・精度、実SQL／実Callの証拠を加える。I/G両者の結果が一致しただけでは正しさの証明にならない。

| ID | 条件・変化 | 期待する観測 |
| --- | --- | --- |
| RG-01 | 正値／0／負値、欠落／null／型不適合 | 判定・評価不能・実行Errorを区別。既存Evaluatorでは欠落・null・型不適合が同じ評価不能に畳まれる場合があるため、分類の一致はその粒度までとし、Adapter固有診断は別に検査 |
| RG-02 | Ruleが参照するAttributeだけ変更 | Iは実Read対象、Gは生成判断と実Read対象が変わる |
| RG-03 | 説明・図のレイアウトだけ変更 | Source Indexの版変化と意味・Artifactの不要失効を分離 |
| RG-04 | Source A→B→A、Evaluator/Generator版更新 | 現在の受理・実行Grantを古い一致から復活させない |
| RG-05 | 標準Functionが未対応Kind／Target保証を要求 | 未対応を示し、勝手に生成へ切り替えて副作用を追加しない |
| RG-06 | IとGのRead順／回数／partialの扱いが違う | 値が同じでも全体同値と判定しない |
| RG-07 | Hook未登録／契約版不一致／二重登録 | 宣言した既定動作か明示Failure。黙った飛ばしや二重実行を防ぐ |
| RG-08 | Guardが拒否したOperationにeffectful Hookがある | 実Providerへの外部作用0回。実行許可を独立検査 |
| RG-09 | 生成済みファイルへの直接編集、既存Target Symbolが曖昧 | 上書き・注入を拒否し、衝突診断と差分を示す |
| RG-10 | 生成TSは新しいがWails埋込Assetは旧版 | 実DesktopのConsumer資格は不成立 |
| RG-11 | GのBuildが成功した、またはDのSQLが作れたが、実Targetは別Providerを呼んだ | 結果が一致しても選択実装の接続失敗 |
| RG-12 | 解釈用設定を実行中に変更、処理中のInvocationがある | 実行ごとの解決版をPinし、処理途中の意味差替えを避ける |

さらに同じ入力分布・Targetで、起動時間、初回／反復応答、メモリ、ビルド時間、配布物、更新頻度、障害回復を測る。I/G/D/Hの採否は「コード生成の有無」ではなく、**必要保証・性能・保守費・説明可能性**の合計で判断する。測定の対象と環境が異なる数値を並べて一般的な勝敗を主張しない。

## 8. 実装と計画への引渡し

最初の題材は実装済みの`sales.order_line.quantity_positive`に固定する。現行の`tei.rule-authoring/v0` Reader、`semantic.Validation`、`validationeval`、`validationgen.GeneratePositive`を再利用する。

1. **R0 現状を確定**：現行のReaderから意味・Source anchorが得られる範囲と、既存`GeneratePositive`の限定事項を固定。現行EvaluatorをIの参照、生成GoをGの参照として区別する。
2. **R1 方式選択のread-only比較**：I/Gが同じTyped Ruleを使うこと、型・NULL・read footprint・実TargetのRequired Capabilityを示す比較票を作る。新Schemaを作る前に既存CLI/Build/Evidenceが持つ項目へ写像する。
3. **R2 生成物ノータッチ＋Hook**：既存Extension Pointの契約を、独立ファイルへ配置した案件実装と合成する最小例で検証。実Guard／Effectは別試験。生成ファイルの直接編集と複数Slot衝突を拒否する。
4. **R3 実Builderと通常CLI**：I経路（選択Evaluatorが実際に使われる）、G経路（生成Goが実Builderに取り込まれる）を別々にConsumerで観測。固定Fixture、Source変更、一因子負例を通す。Reference試験だけで製品受理としない。
5. **R4 Studioへの展開**：Iの標準UI投影、Gの生成Inspector等を同じTargetに接続。正規BundleとWails埋込、実WebView2利用を分けて検査。Stage1→Stage2は既存セルフビルドP0の判定を使う。
6. **R5 Pattern/Skill化**：R1〜R4で実際に繰り返した判断を、既存Product Skillや設計手順から参照可能なレシピへ圧縮。未対応例も含め、対象を増やすごとに重複する定義・手書き責務・変更箇所・時間を計測する。

**停止条件**：Read Scope、型・粒度・NULL／空集合／partial、Effect、Hookの位相と失敗、Provider／Targetの解決、Ownerの意味受理が必要な箇所のいずれかが不明なら、その段階以降を「成立」と報告しない。説明用グラフを実行可能なProcessとみなさない。テスト用の生成Goが起動しただけでStudio全体のセルフビルドに数えない。

## 9. 今回の確定範囲

本書で**設計上の推奨方針**として整理したのは、(a) 方式I/G/D/Hを同じAccepted Meaningから比較する、(b) 設計の実装属性を発見として評価し意味／保証／Target／表示／ナレッジへ分ける、(c) 生成コードはノータッチで契約済みHookを通して拡張する、(d) 既存のOwner・Evidence・Builderへ接続する、という四点である。

これらは現行Reader、Generator、Extension Pointの実装事実を拡大した**設計案**であり、正式な新DSL、汎用解釈器、Hook注入Runtime、任意図表からのコード生成、実Host上でのI/G/D/H同値はまだ実証していない。

## 10. 他チャットの成果との統合：実現の選択と実使用の照合

I/G/D/Hは理解のための**比較分類**であり、新しいCore列挙型、汎用Execution Engine、Generator Registryではない。実装選択は既存Contract／Capability Resolution／Build／Providerが担う。同じ意味を別のノードに表示しても、図のEdgeからProviderの選択や実行許可を確定しない。

### 10.1 一つの照合票で追跡する

| 段階 | 見るべき根拠 | 取り違えないこと |
| --- | --- | --- |
| **Declared** | 実Readerが選択したSource／Role、Canonical Subject、受理されたMeaning | Markdown説明や図形だけで実行意味を作らない |
| **Candidate** | 標準Evaluator／Generator／DB Adapter／外部Providerの契約と未対応条件 | Generatorがあれば全Targetへ適合すると決めない |
| **Selected** | 既存Resolverが選んだ実装・版・明示Binding | Nodeの配線だけでProviderを決めない |
| **Built／Packaged** | Gの生成byteとBuilder実入力、IのEvaluatorとParameter束の実収録、Dの必要Artifact | Gのファイル存在、Iの設定読取りだけを実Artifact受領としない |
| **Invoked** | 実際に呼んだEvaluator／Go Entry／DB SQL／外部Provider、現在の入力・Grant | 選択Bなのに実Call Aで、偶然結果が同じでも合格にしない |
| **Observed** | 独立Expected、Result・Failure・Read Scope・実Consumer、必要なReceipt | 判定の値一致を副作用・開示・権限の一致に拡張しない |

**選べる候補がないことと、まだ検査していないことは別**。対象のMeaningを設計・表示することはできても、選択した方式と実Callを証明できなければ実行可能とは宣言しない。

### 10.2 解釈方式でも消えない依存と失効

Iには、ParameterをBuild時に静的同梱する方式と、実行環境で版付きParameterを読む方式がある。前者では業務判断のGoを再生成しなくてもBundle更新が必要になり得る。後者では、実Invocationが参照したModel／Parameter／Evaluator版を固定し、処理途中で異なる版を混ぜない。

「解釈なので再Build不要」という一律判断はしない。Parameter版、選択Operator、実TargetのLibrary／Plugin、Build Artifact、Current Grantをそれぞれ照合する。Hの一部にDB委譲を含む場合も、DBのSnapshot、NULL、Decimal、Collation、丸めの位置、Join多重度と入力Scopeを保持する。

### 10.3 ノータッチ生成物の所有とHook接続

生成器が所有するPath・Symbol・Moduleを一意に定め、そこへの手動変更を再生成時に検出する。案件固有の実装は独立ファイルまたはProvider Bindingに置く。HookはID／版／Phase／型／順序／許される実装数／Failure／Effect／欠落時の既定を明示し、暗黙の列挙順で複数候補を選ばない。任意Hookの欠落は宣言済み既定へ、必須Hookの欠落は接続不足として扱う。

Ownerが明示していない生成ファイルへのAST／文字列パッチは行わない。例外として必要な構文編集は、Target Symbolの一意性、編集前版、コメント等の保存性、変更Previewと競合拒否を条件とする。Effectful HookはGuard／Authority／Commit／Receiptの既存境界を越えて呼ばない。これらをGeneratorの独自Runtimeで解決せず、既存Extension Point／Plugin契約に接続する。

### 10.4 コードから図表へ戻す際に、受理を逆算しない

Go ASTから条件・Read・Failureを抽出して図表やSourceへ戻す場合、得られるのは**実装の観測と意味の候補**である。元のSource上のOccurrenceや業務意味の受理を、Goの分岐から推測して確定しない。

最初の自己置換は`GeneratePositive`のRequired Shape検査一件で、[実装→モデルdogfooding](IMPLEMENTATION_TO_MODEL_DOGFOOD_2026-11.md)のM0〜M3を再利用する。M1はTEI由来の検査が実際に判定し、M2は製品経路の旧Go条件を減らし、M3は次のTEIで実使用されたことを確かめる。既存Goに委譲しただけの処理は、意味ロジックをTEIへ移した実績に数えない。

### 10.5 重複する試験体系を作らない

本書のRG-01〜12は**設計上の点検項目**であり、独立Expectedの新Registryではない。実検収では[結果要求のLC-01〜12](DECLARATIVE_RESULT_PROCESS_COMPOSITION_2026-11.md)、[宣言的合成のGP-01〜06](DECLARATIVE_COMPOSITION_AND_NODE_SURFACE_CONTRACT_2026-11.md)、既存#1180/#2514を参照する。

最初に同一Ruleの直接Evaluatorと生成Goを別実装として呼び、固定Independent Expectedと照合する。PR #5009の限定的な通常CLIビルド、PR #5056の実Builder入力とのbyte不一致拒否を既存基線とし、Source／Generator／実Build入力の出典閉包を次に接続する。続いて通常CLIの**選択Providerと実Call**、DB委譲での差異、Studio Stage1→Stage2を独立に確かめる。対象ごとの証拠がそろうまでは、これらを製品全体の実証に合算しない。


## 10. 生成する文脈を絞り、生成対象をボイラープレートに近づける

**目標は、生成AIに自由度の高い業務実装を毎回任せることではない。** 図表／Patternの意味とTargetの責務を先に確定し、コード生成が既知の変換規則・型・接続点から定型的に実現できる部分に限定されるよう、設計側で文脈を狭める。ここでいう「ボイラープレート」はコードが冗長という意味ではなく、**出力の選択肢が契約で十分に絞られ、独立試験で合否を決められる作業**を指す。

受入済みの業務Ruleや計算の本質を、もっともらしい定型コードへすり替えてはいけない。意味の決定はSemantic／Owner、実現方法の決定は既存Resolver／Composition、生成したコードの照合は既存Generator／Builder／Evidenceの責務として保つ。AIが案を出す工程と、確定したGeneratorを再実行する工程も分ける。

### 10.1 生成対象の「閉じた文脈」

Generator／生成AIへ渡すContextは、巨大なWorkspaceや過去の会話の全文ではなく、次の**必要な参照と値だけを含む最小の実現課題**とする。

| 固定する情報 | 最小の内容 | 未確定ならどうするか |
| --- | --- | --- |
| **意味と入力** | 選択したCanonical ID、Accepted Meaning／必要属性、元Source参照と版、型、NULL・欠落・空集合、入力粒度・時点・Scope | 補完せずMeaning／Reader Ownerへ返す |
| **出力** | Target言語と版、実現するContract、公開関数・型・Resourceの必要集合、失敗／Effect境界 | 未対応または別実現方式として比較する |
| **接続点** | 既存Module／Package／Registration／Hook ID・版、所有者、呼出位相、Buildに含める配置方法 | 曖昧なSymbol／書込み先へ挿入しない |
| **変換規則** | 選択済みPattern／Template／Operatorと版、許可された変換・依存・Importの一覧 | 新しい変換案としてレビューへ返す |
| **試験・許容差** | 生成器から独立したExpected、正常・境界・負例、許可される非決定性、性能・配布要件 | 「ほぼ確実」とせず試験可能な範囲を先に確定する |
| **実行資格** | Generator候補の出所・digest、Builder／Target能力、試験時の隔離条件 | 受理・Provider Grant・本番配備を推定しない |

**閉じてよいのは実装上の自由度だけ**であり、業務的な未確定事項を隠して「生成可能」と判定しない。グラフの辺・表の列・Hookの存在を推測した場合は、提案としてSourceに戻す。元のMarkdownや図表の全文を毎回詰め込む必要はないが、出典へ戻れるAnchorとSource Revisionは必須である。GeneratorはReferenceの独立Expectedを入力にして正解を写してはならない。

Contextの範囲は過不足なく固定する。過剰な候補列挙や無関係なSourceはAI実装の揺れを増やす一方、必要な型やEffectを隠した短いContextは誤りを招く。**小ささ自体ではなく、必要十分であること**を評価する。

### 10.2 方式の選択を「何がまだ自由か」で判定する

1. 既存の**汎用Function／Operator**だけで意味・Effect・性能を満たす場合はIを選び、業務判断コードを生成しない。
2. 同じ意味を既存の**定型Pattern／Generator**で決定的に展開できる場合はGを選ぶ。AIを実行のたびに呼ばず、固定Generatorで生成する。
3. 定型部分はGenerator、対象固有の変動は解釈パラメータで受けられる場合はHを選ぶ。
4. DBの同じScope・精度・NULL規則で実現できると確認できれば、既存のQuery／Providerへ委譲する。外部EffectはProvider／Hostの契約に従う。これらはI/G/Hの別の権威ではなく、既存の[成果物・計算方式比較](DECLARATIVE_RESULT_PROCESS_COMPOSITION_2026-11.md#11-一つの意味モデルから複数の実現方式を選ぶ2026-11-15追補)にある実現候補である。
5. **特殊な判断の大部分をAIが推測しなければならない**、または独立Expectedが準備できない場合は、生成を自動適用しない。設計の詳細化、Patternの抽出、限定した手書き実装、未対応という選択肢を残す。

選択順は「契約への適合→実体の存在→保守・実行コスト」である。AIの生成しやすさを理由に、Targetへの適合や受理済み業務意味を下げない。

### 10.3 生成AIに任せる部分と、機械的に確定させる部分

| 対象 | AIの役割 | 最終的な確定手段 |
| --- | --- | --- |
| 図表から不足条件を発見する | Pattern候補、質問、実現属性の分類を提案 | Source Writer／Semantic Owner／受理 |
| 新しいGenerator・Templateを作る | 閉じたContextの中で候補ソースと負例案を作る | 独立Fixture、静的解析、再現試験、レビューによる候補選抜 |
| 既知Generatorを日々適用する | 原則不要。必要なら失敗診断の補助 | 版固定した決定的Generator、実Builder入力照合 |
| 案件固有の拡張を実装する | 許可されたHook契約と別ファイルの範囲内で候補を作る | Hookの独立適合試験、Guard／Effect／実Providerの別検証 |
| 未解決の業務判断 | 正答を推測して埋め込まない | Ownerへ戻す。未受理のまま公開・実行しない |

生成AIが書いたTargetコードを直接正本にするのではなく、**AIが定型Generatorや案件拡張の候補を作り、TEIがその版・適合・実消費を管理する**方式を主軸とする。これは[既存のGenerator Source設計](AI_GENERATOR_AUTHORING_STANDARD_2026-11.md)のA（決定的な変換）とB（外部Source実行）を使い分けるもので、メタGeneratorの無条件実装を要求しない。

### 10.4 「ほぼ確実」を観測可能な受入条件へ置き換える

生成AIの成功確率を主観で宣言しない。あるPatternとTargetを**定型自動生成に適格**と呼ぶには、少なくとも以下を揃える。

- **閉じた入力集合**：型、契約版、Target、Hook、書込み可能範囲が明示され、未知の組合せを拒否できる。
- **事前に固定したOracle**：正常・境界・NULL／欠落・拒否・失効・互換・回復のExpectedを、Generator／AI候補と独立に管理する。
- **変種耐性**：名称・Source配置・順序・入力値などを一因子ずつ変え、対象が変わる場合と変わらない場合を分離する。
- **生成再現性**：選抜したGenerator Sourceを固定して再実行し、規定した同一条件下で成果物byteや正規化された実結果を照合する。AIが再回答して同一コードを出すことは要求しない。
- **配置・拡張の安全性**：生成ファイルの外でHookを実装でき、差分Preview・所有範囲照合・直接編集の検出・再生成時のno-opと競合停止ができる。
- **実使用証拠**：対象Builderが消費したbyte、実Entry／Provider／Consumerの観測、失敗時の区別を得る。生成成功やCompile PASSだけで満たさない。

深刻な負例を一つでも通してしまう場合、対象Patternを自動適用の対象から外す。何件成功すれば確率的に「ほぼ確実」と言えるかは、適用母集団・負例設計・実Targetを定める前に数値化しない。成功事例の増加だけで未試験の分岐を保証しない。

### 10.5 最初の「閉じた文脈」実証

**同一のRule** `sales.order_line.quantity_positive` に対して、次の2種類の生成課題を区別する。

- **課題A（定型化に適する）**：既存`semantic.Validation`のID、Field、型、positive判定、既存Go Target／Hook契約を固定し、判定・Adapter・Bindingを既存Generatorから生成する。既存の[通常CLI実ビルド](IMPLEMENTATION_TO_MODEL_DOGFOOD_2026-11.md#11-i2m-1の実回帰と製品cliビルドの到達点2026-11-15)と生成Go／Builder byte取違え拒否を基線にする。
- **課題B（まだ定型化しない）**：Rule表の説明だけを根拠に、任意のデータ取得・複数表更新・外部通知・トランザクション回復までAIに一括実装させる。Read Scope、Authority、Effect／Receipt、Host Providerが未確定なので、仕様補完や自動注入を拒否する。

これに加えて、**同名ノード・多重有向辺を持つGraph**を第二の異質な題材とし、辺の向き・ID・複数辺・描画順を変える。Graphviz/Mermaidの見た目を意味受理に使わず、選択されたGraph Readerが返した型付き意味からだけ生成する。現行の限定Graph Readerを正式な全図表Parserと取り違えない。

初回の完了条件は、Aで固定Contractからの実生成と実消費までを確認し、Bで未解決を拒否できること。**自動化を増やすより、いつ生成しないかを確実に判断できることを先に示す。**

## 11. 既存の進捗と今回の未実証範囲

2026-11-15の現行mainには、限定的なMarkdown Rule→生成Go→通常CLI `self-build` 経路（PR #5009）と、選択された生成GoとBuilderに置くbyteの取り違え負例（PR #5056）がある。これらは今後のA課題の基線として使える。一方、同一の`*.tei.md`の任意図表をReaderが解釈し、I/G/Hを自動選択し、Hookを実Targetへ配置して、StudioのStage1→Stage2まで再入場する**一つの完成経路**はまだ成立していない。

本書は既存のGenerator／Evaluator／Plugin実装のOwnerに候補比較と負例を渡すものであり、採用済みDSL文法・保証された生成AI成功率・自動運用権限を新設するものではない。
