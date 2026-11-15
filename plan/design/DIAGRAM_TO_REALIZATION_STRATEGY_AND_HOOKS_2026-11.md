# 図表から実装へ：解釈実行・コード生成・フック拡張の選択設計

- **状態**：設計・検収計画。2026-11-15。新しいDSL構文・汎用Runtime・Placement APIの採用、実Targetでの動作、業務意味のOwner受理は主張しない。
- **目的**：`*.tei.md`の図表と定義から、必要に応じて既存の汎用Functionによる解釈実行、Targetコード生成、両者の合成を選ぶ。生成物は原則ノータッチとし、既存アプリケーションには契約で定めた接続点を通して取り込む。
- **関連**：[図表→Build縦断](TEI_MD_DOMAIN_DIAGRAM_TO_BUILD_LOWERING_2026-11.md)、[Critical Lowering](CRITICAL_LOWERING_CONTRACT_DESIGN.md)、[生成器契約の境界](GENERATOR_CONTRACT_BOUNDARIES_DESIGN_2026-10.md)、[Generator選抜](AI_GENERATOR_AUTHORING_STANDARD_2026-11.md)、[意味と宣言的合成](DECLARATIVE_COMPOSITION_AND_NODE_SURFACE_CONTRACT_2026-11.md)、[実装→モデルdogfooding](IMPLEMENTATION_TO_MODEL_DOGFOOD_2026-11.md)、[Pluginの第一縦断](PLUGIN_RULE_PRODUCT_VERTICAL_SLICE_2026-11.md)、[Studio部品生成](../../tools/reference-studio-part-generator/README.md)、[ADR-0045（提案中）](../adr/0045-generator-artifact-lifecycle-and-evidence.md)。
- **Ownerとの境界**：Reader/Languageは#2621/#988、Semantic/Roleは#2619/#2620、Loweringは#1121/#4544、Plugin/Contract/Hookは#951/#1016/#4198、Generator選抜とSource束縛は#2595、BuildとSelf-buildは#943/#2831、独立Expectedは#1180/#2514、Studio実Consumerは#1824/#4583。第二Parser、第二Semantic、別のBuild/Plugin Registry、TEI独自Hostを設けない。

## 1. 結論：実装方法の選択も、設計の一部である

図表に実装上の属性が増えることは、必ずしも「設計が技術に汚染された」ことを意味しない。型、粒度、更新競合、読取りの完全性、再試行、Hookの位置など、詳細化して初めて見える要件を発見した可能性がある。まずその属性が**業務意味の制約**、**実現に必要な保証**、**Target固有の実現方法**、**表示用設定**、**再利用できるPatternのパラメータ**のどれかを見分ける。この発見を元の設計へ戻して検討できることがTEIの強みである。

実現方式は、アプリケーション全体に一律ではなく、Function／Rule／Operationとその選択Target・Contract単位で比較する。

| 方式 | 主な役割 | 採る条件 | 生成しないもの |
| --- | --- | --- | --- |
| **I：解釈実行** | 既存の標準Function／Evaluatorが、受理済み定義のパラメータを読み動作する | 演算・型・失敗・性能・Target・作用境界を既存Function契約で満たせる | 個別の業務判断用ソースコード |
| **G：静的生成** | 受理済み意味をTarget-nativeなコード・設定・ResourceへLoweringする | 専用最適化、コンパイル時結合、Target固有API、配布時の定義非同梱等が必要 | 実行時に共通Evaluatorが毎回構造を解釈する処理 |
| **H：混合** | 頻用・安定領域は生成し、変動しやすい部分は共通Function／設定で解釈する | 境界・契約・コストを明示し、両者を別々に検証できる | 全領域を一律に生成／一律に解釈する必要性 |

**既定候補はIを先に評価するが、Iが常に最適とはしない。** 生成器を開発する前に既存の標準Functionで十分か確かめる。Iで要件を満たすならGを作らず、Gの方が運用・性能・配布条件に適うなら根拠を残して選ぶ。Gも意味の正しさを自動保証しない。IとGが同じ振る舞いを示しても、どちらが業務上正しいかは別に受理する。

「生成工程を飛ばす」のは**業務判断のコード生成**である。必要に応じて、定義の正規化、実行用パラメータ束のPackaging、Pluginの静的収録、既存TargetのBuild・配置は残る。Iを選んだからTEI常駐Hostが必要になるわけではない。Go単一バイナリには既存Evaluatorを静的に組み込み、外部Java/.NET HostにはTarget-nativeな標準Library／Adapterを配布できる。

## 2. 共通入口と分岐

```text
*.tei.md（原文、図表、表、明示的な定義領域）
  → 選択済みReader / Source Index / typed Meaning / Acceptance
  → 対象Operation・FunctionとRequired Contractを解決
  → 実現方式の選択（I / G / H、Target・Provider・保証・コストを照合）
     ├─ I：標準Function＋版付きパラメータ束
     ├─ G：Generator → 対象言語の生成物 → Placement
     └─ H：生成された接続部＋解釈実行の標準Function
  → 既存Composition / Build Intent / Artifact / 実Target
  → 実ConsumerでのRead・Decision・Failure・Effectを観測
```

分岐前のAccepted Meaningを共用し、実行方法の差で別の業務正本を作らない。Iは「解釈器が自動的に意味を正本化する」経路ではない。定義やParameterが新しくなれば、実際に読み込んだ版と実行主体を照合する。GではGenerator Source、生成物、Builderが読んだbyteを別々に固定する。Hでは両経路の接続点を固定する。

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
Implementation choice: interpret | generate | hybrid
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

## 7. IとGの意味保存を値の一致より広く試す

共通の独立Expectedを先に固定し、IとGをそれぞれ**実Consumer**で検査する。I/G両者の結果が一致しただけでは正しさの証明にならない。

| ID | 条件・変化 | 期待する観測 |
| --- | --- | --- |
| RG-01 | 正値／0／負値、欠落／null／型不適合 | 結果・評価不能・Errorを区別。偽値へ潰さない |
| RG-02 | Ruleが参照するAttributeだけ変更 | Iは実Read対象、Gは生成判断と実Read対象が変わる |
| RG-03 | 説明・図のレイアウトだけ変更 | Source Indexの版変化と意味・Artifactの不要失効を分離 |
| RG-04 | Source A→B→A、Evaluator/Generator版更新 | 現在の受理・実行Grantを古い一致から復活させない |
| RG-05 | 標準Functionが未対応Kind／Target保証を要求 | 未対応を示し、勝手に生成へ切り替えて副作用を追加しない |
| RG-06 | IとGのRead順／回数／partialの扱いが違う | 値が同じでも全体同値と判定しない |
| RG-07 | Hook未登録／契約版不一致／二重登録 | 宣言した既定動作か明示Failure。黙った飛ばしや二重実行を防ぐ |
| RG-08 | Guardが拒否したOperationにeffectful Hookがある | 実Providerへの外部作用0回。実行許可を独立検査 |
| RG-09 | 生成済みファイルへの直接編集、既存Target Symbolが曖昧 | 上書き・注入を拒否し、衝突診断と差分を示す |
| RG-10 | 生成TSは新しいがWails埋込Assetは旧版 | 実DesktopのConsumer資格は不成立 |
| RG-11 | Gは正しくBuildされたが実Targetは別Providerを呼んだ | 結果が一致しても選択実装の接続失敗 |
| RG-12 | 解釈用設定を実行中に変更、処理中のInvocationがある | 実行ごとの解決版をPinし、処理途中の意味差替えを避ける |

さらに同じ入力分布・Targetで、起動時間、初回／反復応答、メモリ、ビルド時間、配布物、更新頻度、障害回復を測る。I/Gの採否は「コード生成の有無」ではなく、**必要保証・性能・保守費・説明可能性**の合計で判断する。測定の対象と環境が異なる数値を並べて一般的な勝敗を主張しない。

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

本書で**設計上の推奨方針**として整理したのは、(a) 方式I/G/Hを同じAccepted Meaningから選ぶ、(b) 設計の実装属性を発見として評価し意味／保証／Target／表示／ナレッジへ分ける、(c) 生成コードはノータッチで契約済みHookを通して拡張する、(d) 既存のOwner・Evidence・Builderへ接続する、という四点である。

これらは現行Reader、Generator、Extension Pointの実装事実を拡大した**設計案**であり、正式な新DSL、汎用解釈器、Hook注入Runtime、任意図表からのコード生成、実Host上でのI/G同値はまだ実証していない。
