# TEI DSL：グローバル共通構文・Unicode表層名・Studio表示・AI入力安全性の統合計画（2026-11-03）

- **状態**：設計候補／必要な受入条件を既存Issueへ配分。**DSL文法、Unicode識別子Profile、Formatter規則は正式採用前**。本書は製品実装や受入済みADRではない。
- **親**：[#2619](https://github.com/hat47x/tei-platform/issues/2619) Everything is Data、[#988](https://github.com/hat47x/tei-platform/issues/988) Language Plugin/Operator、[#2621](https://github.com/hat47x/tei-platform/issues/2621) Authoring、[#1124](https://github.com/hat47x/tei-platform/issues/1124) Studio編集。
- **再利用する既存契約**：[#1120](https://github.com/hat47x/tei-platform/issues/1120) Stable ID/Trace（closed）、[#1396](https://github.com/hat47x/tei-platform/issues/1396) 多言語表示資源（closed、read-only E1実装の[#1404](https://github.com/hat47x/tei-platform/pull/1404) merged）、[#2497](https://github.com/hat47x/tei-platform/issues/2497) Projection/逆写像、[#2515](https://github.com/hat47x/tei-platform/issues/2515)/[#2516](https://github.com/hat47x/tei-platform/issues/2516) AI/Consumerへの開示。
- **既存原文保存仕様**：[main統合 #2674](https://github.com/hat47x/tei-platform/pull/2674)／[Markdown Source Fidelity](EVERYTHING_DATA_MARKDOWN_SOURCE_FIDELITY_2026-11.md)。保存は元のシンタックスシュガーを維持し、通常Saveで展開後の標準データ形式へ再直列化しない。
- **最新局所実証との関係**：[Draft #2639](https://github.com/hat47x/tei-platform/pull/2639) は実Go Language Plugin ReferenceとSemantic candidateの一部を試すが、**Unicode表層識別子／Studio名切替／AI入力安全の製品適合は未実証**。別Draft[#2667](https://github.com/hat47x/tei-platform/pull/2667)の横断計画本文を競合編集しない。

## 0. 直近実装キュー：一つの意味・Scenario・生成入口を分けて確認する（2026-11-04）

**この節を新規実装の着手票とし、§1〜14は根拠・反例・Owner境界の参照用とする。** 追加のDSL/テスト/ビルド管理の正本を作らず、既存資産に依存することを明示する。最初に全部のUnicode／SQL／Provider／EKIを完成させるのではなく、**外部Importを使わない受注Source**で構文と意味の最小同値を確かめ、その後必要なレーンだけを接続する（§14.3）。

### 0.1 最小一件の入力を固定する

試験対象は公開Referenceの`order_entry`を起点とした**合成受注O1**のread-onlyモデルとする。以下の値はテスト用であり、本番業務Rule・正式DSL Syntax・採用済みCanonical Attributeの宣言ではない。

| 観測対象 | 比較元・Expected | 非同値を示す負例 |
| --- | --- | --- |
| 意味参照 | `sales.order.quantity`等の**既存ASCII Canonical ID**、対象O1、Rule IDと型、Source revisionを明示Binding | `数量`と`quantity`を名前だけで同じStable IDに決定する／同一日本語論理名の別Subjectを混同 |
| 純粋な計算 | quantity=2、unitPrice=300、amount=600（整数精度で厳密）、同じ独立Expected | NULL→0、missing→empty、partial→complete、`std.decimal`→`std.string`でも同じ意味だとする |
| Validation／決定表 | quantity=-1で業務拒否、数量未取得は評価不能、後続`creditHold`が未取得でもshort circuit契約を保持 | eager評価で別Errorを発生／`positive`を`>=0`と同義へ補完／Hit Policyを省略 |
| Unicode／Source | Unicode表層名は**字句ProfileとBindingの採用前候補**、Studio多言語論理名はread-only表示に限定。元Source byte/span・Sugarを保持 | NFC/NFD・Confusable・Bidi・同名Scope衝突、UTF-8 byteとEditor offset混同、Formatを通常Saveで暗黙実行 |
| Scenario／経路 | [#1180](https://github.com/hat47x/tei-platform/issues/1180)の**保存済みDriver Scenarioがテスト正本**。CLI/Web/DesktopへのStep/Outcome/独立Expectedはそれから派生 | CLIの成功ログをWebのテスト正本として逆生成／未対応Web Stepを`PASS`扱い／GUI screenshotを業務意味のExpectedにする |
| Build／公開入口 | [#2788](https://github.com/hat47x/tei-platform/pull/2788)の単一Goバイナリ（割当設定不要）から開始。必要時の1→2 Build Unitは**同一Function ID/業務Meaning/Scenario**を保つ比較 | 表示ラベル／ローカルSymbol／Source Folderを実行入口名や公開Capability/Grantへ自動昇格、内部Function全件を公開 |

**ここで保つ「同一」の対象を限定する**：構文Source byte一致、型付きMeaning同値、ScenarioのExpected同値、生成Artifact byte一致、実行結果/Effect同値は独立した判定。Sourceが異なる4表層から同じ型付きCandidateを得られても、実行権限やBuild完全性が成立したことにはならない。

### 0.2 まず着手できる3段階（実コードとOwnerにひも付ける）

| 順 | 最小実作業 | Doneと未達を分ける根拠 |
| --- | --- | --- |
| **A：既存Goの基準線** | `core/identity/id.go`のASCII受理集合、`reference/dsl-language-plugin`のParser/Language pin/Source diagnostics、`internal/driver/final_gate_b_scenario_test.go`のtest-local Stepを**変更せず観察**し、同一FixtureのExpectedとReference接点を列記する | [#2639](https://github.com/hat47x/tei-platform/pull/2639)現HEADのGitHub Actions [#41578551459](https://github.com/hat47x/tei-platform/actions/runs/41578551459) はcompleted/success。ただしtest-only JSON Adapter・Semantic Candidateの限定Go証拠。正式Unicode Lexer／保存済みScenario／業務Meaning受理／製品CLI ParserのPASSではない |
| **B：最小記法の比較票** | Aの同じIndependent Expectedで、言語中立低ノイズDSL／Go風式／Markdown表／chain式の4表層について、Source token/順序/NULL・未取得・短絡・型・diagnostic locationと**非対象Source byteへの影響**を並べる。現在Parserが受けない表層は`unsupported`と記録 | AIは**構文ノイズと認知順序の指摘のみ**。事前のExpectedとSource/Role/Operator pinの検査は別。人間被験者・仮想専門家の採点・固定数値閾値は不要。Ownerが文法・Unicode Profileを採否判断 |
| **C：Scenarioと生成入口へ接続** | [#1180](https://github.com/hat47x/tei-platform/issues/1180)に既存Driverの1 read-only Step＋1 Validation負例を**保存・再生成可能なScenario候補**として引き渡す。CLI argv/stdin、Web/Desktopの業務操作Binding、独立結果・Host/VisualのEvidenceを別列で出す。Build単位は[#2788](https://github.com/hat47x/tei-platform/pull/2788)へ参照だけ渡す | Scenario GeneratorやWeb E2Eがまだ動かないなら`NOT IMPLEMENTED`。CLI実実行・Web Host・実Provider・実ジョブはそれぞれOwnerが検証。Source名/翻訳によって実Entry Point/権限を変更しない |

**必須の前提／停止条件**：外部Importのある案件だけ[#2743](https://github.com/hat47x/tei-platform/pull/2743)のP0b実Read Cohortの証拠を求める。実Command/業務Effectへ進む時だけ[#2780](https://github.com/hat47x/tei-platform/pull/2780)のO1/O2 Target不一致・二重Tx・Replay/Unknown負例に接続。最小のDSL可読性PoCをこれらの実装完了まで止めない。仮に既存`operations`がgaps=0でも[実CLI #2666](https://github.com/hat47x/tei-platform/pull/2666)のようにArtifact ManifestがincompleteならConsumer/Build完全性を主張しない。

### 0.3 実行資格と配布単位はDSL構文では決まらない

**Functionの業務意味・Scenarioの期待結果・Invocation Binding・Build Unit・Deployment/公開Entry Point・Host Grantを分ける**。特にUnicodeローカルSymbol・翻訳ラベル・ファイル名からCLIコマンド/HTTP Route/公開Job名を黙って生成しない。Stable IDと実行Contextを固定した**明示的なInvocation Mapping**が必要であり、Schema・対象ID・権限を実行時に再検証する。単一Goバイナリへの静的同梱も、Functionの自動公開や認可を意味しない。

[Build設計 Draft #2788 §14](https://github.com/hat47x/tei-platform/blob/work/issue-259-function-build-assignment-20261103/plan/design/RUNTIME_TARGET_BINARY_COMPOSITION_DESIGN.md)が**設定なし単一ビルドと必要時のみの複数Build**を担当し、本書は表層名→意味Refが変更・再配置で揺れないことを担当する。Function/Source Folder移動・表示言語変更・Build割当変更を同じ「モデルの変更」にまとめない。変更によって必要になるSource/Meaning/Artifact/Deployment/Execution Receiptの再資格化をそれぞれ求める。

**一旦採用しないもの**：Unicode Canonical IDそのものへの変更、日本語用Keyword Grammar、新しい共通Parser/Host/Registry/Scenario Runner/独立Oracle/Build割当Schema、Go AST都合の普段のDSL記法、中央のカバレッジ数値。**設計採用待ち**：字句Unicode Profile、4表層から選ぶ記法、Decision TableのHit Policy、Build Unit正式型・Invocation Mapping形式。

**検証状態**：本節のA〜Cは作業の提案であり、実行した新たなGo Test/CLI/Web/Studio/Unicode Parser/Build構成・Host/Provider E2Eはない。#2639 exact-head CIはGo Referenceの限定合格、#2788もdocs-onlyのDraftである。Ownerの意味受理と正式Syntax採用は行っていない。

## 1. 優先順位：自然なテキスト記述と意味の厳密性を両立

**製品要件として維持するもの**：
1. **グローバル共通のDSL構文**。直接日本語をキーワード化した「日本語DSL」を正式候補にしない。ローカル言語は主にドメイン名・コメント・文書・Studio論理名として扱える。
2. **ドメイン専門家にも違和感の少ない認知順序と少ない構文ノイズ**を最優先。Go ASTで実装しやすいことは二次的な制約。日常の定義に `package/import/struct/func` 等のGoのプログラム用宣言を強制しない。
3. Everything is Dataの構文変換・標準構造・意味Roleは、テキスト表層と独立。Markdown表・軽量式・chain・Graphvizのどれかだけに意味的特権を与えない。**同じData/Attribute ID、型、値が同じでも、Rule/Owner/Provider/Contextが違えば業務意味同値とは限らない**。
4. **TEI Studioでは同一Subjectを表／Tree／Graph／Functionやインスタンスへ共通投影**。テキストの構文候補の比較では、GUIの豊富さを評価軸に混ぜず、静的なノイズ・認知順序・局所編集性・明示性を比較する。
5. **表層の名称表記とSource保存は別**。Studioの表示言語切替、AI向け説明、生成Target名の変換を元のMarkdown/.tei Sourceへ無断書戻ししない。

**未決**：具体的な予約語・演算子・識別子Profile・Unicode版・物理名の表記規則・カスタムFormatter。比較実験とOwner判断を経ずに文法や新Core Schemaを確定しない。

## 2. 五種類の「名前」を混ぜない

| 名前の層 | 例（合成の受注Domain） | 実際に何を識別するか |
| --- | --- | --- |
| **Canonical ID / ModelRef** | `sales.order.quantity` | 既存TEIで安定した意味参照。原則として言語・翻訳の変更では改名しない |
| **Source表層識別子**（Unicode許容は候補） | `quantity` / `数量` | .tei やMarkdownの特定Syntax Pluginで記述するローカル変数・Symbol。ScopeとSource occurrenceに依存 |
| **物理名** | `T_JUCHU.JUCHU_SU` | 外部DB/Host/Targetにある名称。Dialect・引用・実Schema・版に拘束される |
| **論理表示名／多言語資源** | `Order quantity` / `受注数量` | ユーザーが選ぶ表示言語のラベル。名前から逆に意味IDや物理対象を推測しない |
| **生成Target用Symbol** | `orderQuantity`等 | Go/Java/.NET/WASM等の生成物に必要な各Target表記。衝突と元ID対応を固定 |

**重要な現行実装上の事実**：mainの `core/identity/id.go` は `ParseCanonicalID` に `^[a-z][a-z0-9_]*(\.[a-z][a-z0-9_]*)*$`、`ParseLocalName` に `^[a-z][a-z0-9_]*$` 相当の**ASCII制約**を持つ。`SymbolToSegment` も非ASCII Symbolを受け付けない。この検証は[#1992](https://github.com/hat47x/tei-platform/pull/1992)（merged）で同じ受理集合を保ちながら高速化された。よって、現状では**「UTF-8 Sourceが書ける」ことから「Canonical IDや既存SymbolにUnicodeを直接使用できる」とは主張できない**。

Unicode表層Symbolを評価する場合の最小案は、**既存ASCII Canonical IDを維持したまま、固定されたLanguage PluginのSymbol→明示的な型付きModelRef Binding**を検討すること。Unicodeからのローマ字化、翻訳、類似名による自動Canonical ID生成は行わない。参照先が0/2件以上、異Scope、別版、不正な名称の場合は候補/診断に留める。将来Canonical ID自体を拡張するかは互換性の大きい**別Owner判断**とし、本計画の前提にしない。

**例（DSL文法の採用ではなく表示比較）**：

```text
// English ASCII source symbol
amount = quantity * unitPrice

// 同じ構文のローカル名例（現行Parserで動くと主張しない）
金額 = 数量 * 単価
```

テキスト表層から同一Meaningへ到達できるかは**事前に確定したBindingとSource/Role/Operator Pinがある場合だけ**確認できる。単に名前を翻訳して同値とはしない。

## 3. Studioの表示切替とSource不変条件

表示モード候補は `canonical ID`、`source symbol`、`physical name`、`logical name(locale)`。現[#1396](https://github.com/hat47x/tei-platform/issues/1396) E1の既存Language Resource Catalogは**固定済み意味参照→確認済み表示資源のread-only解決**に使う。既存の翻訳資源から許される表示と、新たな物理名・Code symbolのRewriteは別。

- モード変更では**同じSubject ID、Meaning、Source byte、Source occurrence、Selection**を維持する。UIラベル変更でArtifact Revision/Model Revisionを勝手に進めない。
- ロケール資源が欠落・失効・未承認の場合、silent translationを避け、表示用の明示fallback／stable ID／unresolvedを区別。旧翻訳表示を現在の確定Meaningと取り違えない。
- 物理名は `Binding/Source/Target/Version`が一意に解決した場合だけ表示し、物理名からの語彙類似による参照選択をしない。等しい論理名が複数あるときはScopeやStable IDを併記する。
- Sourceの編集は元の糖衣構文へ**Source-aware局所Patch**。表示モードの切替それ自体では書込ゼロ。表幅やMarkdown formatterは通常Saveから独立。Source移動・A→B→A・旧Span・二つの同名Occurrenceには別の再資格化を要求。
- Read-only Studio Projection / Source編集権限 / Tool Action / 現Host Grant を混同しない。外部AIへ送るラベルやSource本文にはDisclosure Scopeを適用。

## 4. Unicode表層識別子Profileの評価候補（正式字句規則ではない）

参考資料：
- Unicode [UAX #31 Unicode Identifiers and Syntax](https://www.unicode.org/reports/tr31/)（2026-09-01 Unicode 18.0.0 / revision 45）。
- Unicode [UTS #39 Unicode Security Mechanisms](https://www.unicode.org/reports/tr39/)（2026-08-27 Unicode 18.0.0 / revision 34）。
- Unicode [UTS #55 Unicode Source Code Handling](https://www.unicode.org/reports/tr55/)（version 2 / 2024-01-29）。

**版を固定した仕様候補**： `XID_Start`/ `XID_Continue`等を基礎に、ASCII構文記号・数字先頭・予約語・Unicode識別子の正規化比較・危険なformat/control codeの扱いをLanguage Pluginに明示する。Unicode版更新で受理集合やConfusable結果が変わる可能性を版間差分として扱う。Go標準の`unicode.IsLetter`等がそのままUAX #31適合を保証するとは推測しない。

| 検査 | 候補動作 | 避ける危険 |
| --- | --- | --- |
| UTF-8妥当性とSource span | 不正byte列拒否、元byte offset／UTF-16 editor offset区別 | 日本語文字の途中でPatch／別Cell変更 |
| 識別子XID属性／script使用 | 固定Unicode版＋Language Profileで判断。合理的なローカル文字体系を一律拒否しない | 日本語・他非ラテン語の実用上の排除 |
| Canonical比較とNormalization | NFC等は**比較用候補として別計算**し、同一視の衝突を診断。元Source byteを勝手に正規化しない | NFC/NFD差の見逃し、no-op saveの改変 |
| 視覚的Confusable | UTS #39相当のskeletonは**診断専用**に使い、元IDや意味の自動改名に利用しない | Latin `order` とCyrillic `оrder` の視覚混同 |
| Bidi/Default-Ignorable/Zero-width | 構文・識別子・コメント・文字列を区別。識別子への非承認format control等は拒否候補、コメント等も安全表示。RTL言語を全体禁止しない | 表示順を使った別命令・別式への偽装 |
| Mixed-script / numeral | 全混在禁止でなくContext依存の警告/診断。実名前とCode Pointを比較可能にする | 正当な日本語＋英数字と不正な同形文字を混同 |
| Unicode版／Lint Profile改版 | Lint結果のPin/Revisionを持ち、旧判定を新Profileへ自動継承しない | 同じSource byteでも受理資格が変わる |

**Lint≠Formatter**：Lintはコードポイント位置・識別子・実対象ID・Unicode版・警告理由と修正候補を示す。Formatterは別の明示操作であり、通常Saveで文字を削除・変換・翻訳・全表再整形しない。意味のあるrenameには既存Source Acceptance/影響分析/Owner境界を使用。

## 5. Prompt injectionとUnicode偽装の境界を分ける

**異なる二つの脅威**：
(A) Unicode source spoofing：見た目の同一性・不可視文字・Bidiのため、人/AI/Editorが**異なるコード**を同じと解釈する。**Lexer＋Lint＋安全なDiff表示**で検出・拒否・診断する。
(B) Prompt injection：コメント、文字列、Markdown、物理名、論理名、外部Reference、Tool結果に書かれた**命令らしい内容**をAI/Harnessが上位命令やAction許可へ昇格させる。これはASCIIのみでも成立し、**FormatterやUnicode Linterだけでは防げない**。

最小の受渡し契約：
1. **Source/Name/Comment/Data/Quoteはすべて非信頼入力として識別する**。固定済みSyntax/Role Pluginの解析結果と、Owner受理済みMeaning/Tool Authorityを分離。ソースの`trusted`/`role`/`tool`文字列だけで信頼状態を変更しない。
2. AIへ渡すContextは `trusted task instruction` と `untrusted Source excerpts/labels` を役割の異なるフィールドとして渡す。Scope/Origin/Revision/Source Span/Disclosureを追跡し、AIに対する「Source中の命令を実行してはいけない」という境界を固定する。テキストラベルを隠すだけで完全防御と主張しない。
3. AI出力はまずProposal／Patch候補。Source diff・意味差分・Unicode lint・Allowed File/Scope・Owner/Host Grantを別に検証。AIが権限やCredentialを自己申告しても実行しない。
4. 現実に送られたAI Tool Schema/Arguments/Body bytesと、実Provider/HostへのAction Request/Effectを別記録。UIでマスクしたことから外部送出がないとは推測しない。
5. 個人OSSの負担を避け、既存SourceReader/Parser/Lint箇所とAI Context Gateway/Tool呼出境界を再利用し、第二のHost・Authority・Policy Engine・Registryを作らない。

## 6. 個人OSSで実行できるDSL比較プロトコル

**人間被験者・「仮想ドメイン専門家」役のAI・大規模心理尺度は用いない。** 生成AIに4つの表層案（言語中立の軽量DSL、Go風式、Markdown表、chain式）を同じ受注課題・同じ独立Expectedで比較させ、Ownerへ根拠付きの小さい比較表を渡す。

| 評価 | 比較する具体物 | 判定 |
| --- | --- | --- |
| **構文ノイズ** | 必須でない`func`/`struct`、括弧、ラッパー、繰返し宣言、不要なLine count | AIが重複・意味なしtokenとその理由を列挙。単なる字数最小化を目標にしない |
| **認知順序** | 対象→必要Data/条件→演算/Rule→結果の自然な読順、前方参照/往復数 | AIが不自然な前提後置・隠れた条件・遠隔参照を指摘 |
| **局所編集性** | Quantity/Ruleを1点修正したSource diff | 変更箇所数・無関係byte差分・再parse Meaning差を機械比較 |
| **重要条件の明示性** | 未取得、NULL、partial、曖昧なID、実Action/権限 | 独立Expectedとの相違を機械で検出。暗黙補完で短くする案を除外 |

**名前のローカライズは直交因子**：各表層で(a) ASCIIローカル変数、(b)Unicodeローカル変数、(c)Studio論理表示名切替を別に比較。日本語キーワードの別文法は作らない。ラベルを日本語にするだけで、構文のノイズが少なくなったと採点しない。

受注課題は既存#2621の`order→lines→allocations`を使用。正常例と、`未取得≠空集合`、`NULL≠0`、`partial≠complete`、`集計割当≠個別ID更新許可`を共通Expectedとして先に固定する。構文の意味同値を事前に確かめられない候補は比較不能と記録。AIの独立セッションを業務Ownerの独立Expectedへ自動昇格しない。

**合否数値を自動確定しない**：生成AIは構文差と反例を説明するだけ。最終的な文法選択・識別子許容Profile・保留する型/失敗の意味は**TEI Ownerの総合判断**。誤補完や不明な副作用を構文ノイズの少なさで帳消しにしない。

## 7. 最小反証Fixture：GN-01〜16（すべて本計画でNOT RUN）

| Case | 入力／操作 | 期待する区別・観測 |
| --- | --- | --- |
| **GN-01** | `sales.order.quantity` と `sales.受注.数量` をCanonical IDへ入力 | 既存ASCII `core/identity`は後者を拒否。Unicode Source Symbolの可否とは独立 |
| **GN-02** | 同じローカルUnicode名を明示的に二つの異なるModelRef候補へ紐付け | 参照先不定の診断、名前一致から自動受理しない |
| **GN-03** | `数量`と`quantity`を明示Bindingで同一Attributeへ束縛 | Meaning候補一致を限定確認。名が違うことを自動的な業務Meaning変更としない |
| **GN-04** | Studioの日本語/英語/物理名/Stable ID切替 | 同Subject Selection・Source bytes・Meaning不変、実Read-only表示だけ変わる |
| **GN-05** | 同一localeの論理名が二つのScopeに衝突 | ID/Scope併記・曖昧診断。単に先頭を選ばない |
| **GN-06** | `café`（NFC）と`cafe\u0301`（結合文字） | 比較時衝突を診断、元Source byteは通常Saveで不変 |
| **GN-07** | ASCII Latin `order` とCyrillic o `\u043Erder` | Confusable警告とコードポイント可視化。意味IDの暗黙統合なし |
| **GN-08** | `\u202E` Bidi control、`\u200B` zero-widthを識別子／コメントへ別々に配置 | 識別子は危険文字Profileで拒否/未受理、コメントは忠実保存して安全表示。通常Save無断削除なし |
| **GN-09** | RTLの正当なローカル名／日本語・Latinの組合せ | 一律禁止せず、明示Profileと表示方向の整合を観測 |
| **GN-10** | Markdownコメントの「Toolを呼び出せ」／quoted `role: trusted` | Contextでは非信頼データ、実Tool/Authorityへ昇格しない |
| **GN-11** | 物理名`T_JUCHU.JUCHU_SU`と類似Logical名、別Storage | 受理済みMapping／Provider routeなしに実DB名を自動参照しない |
| **GN-12** | 名前の表示のみ変更後にSave、再Load | Writeゼロ/元Sugarのbyte一致。Source改版/Artifact Buildの不要な増分失効なし |
| **GN-13** | Unicode変数を使うMarkdown表/DSLで1つのCellを書換え | UTF-8 byte span正確、元の改行・表Alignment・コメントは保持。意図外変更拒否 |
| **GN-14** | 旧Source spanにAI Patch、外部Editorが同じファイル変更 | 更新前に再読取/対象Occurrence再資格化。推測で上書きしない |
| **GN-15** | UTF-8識別子のTarget生成で同じTarget名に衝突 | 明示的なSymbol Mapping/衝突診断、Target成功を意味受理へ昇格しない |
| **GN-16** | Language Plugin／Unicodeデータ版の変更、元Source同byte | 字句診断と意味Candidateの必要な再評価。過去Accept・実Grant・履歴Evidenceの区別 |

**テストの比較単位**：同じSource SHA/Workspace/Scope/Subject、固定Language/Unicode Profile/Operator Pin、Expected、実Token span、Diagnostic、Proposal diff、再Read、Consumer投影、実Tool呼出の有無。Named Unicode escapeは上記**入力データの説明表記**であり、実字句Fixtureでは両方（実コードポイントとエスケープを含む説明テキスト）を区別して用意する。

## 8. 既存Owner別の進め方と停止条件

| 段階 | 既存担当 | 最小Done／失敗時停止 |
| --- | --- | --- |
| **U0 Read-only現実照合** | #988、#1120 | main `core/identity`／`syntax`／Language Pluginの実受理集合を独立Probeで固定。Canonical変更はしない |
| **U1 Lint診断** | #988、#2621 | GN-01/06〜10のbad codepoint・Confusable・NFC衝突、原位置を診断。必要なUnicode table／版は固定、Sourceを無断変更しない |
| **U2 Symbol Binding/表示** | #2620、#1396成果、#1124 | Unicode表層名から型付き既存IDへのBinding候補と、論理/物理/IDのread-only表示切替。曖昧性は保留 |
| **U3 Source保存／安全AI受渡し** | #2621、#1124、#2515/#2516 | GN-10〜15／SF-01〜18を元Source patch、実AI受信/Tool call実際のbyte、独立Authorityで別確認 |
| **U4 Ownerの小比較** | #988、#2514、TEI Owner | 同じ受注Expectedで4表層のNoise/Cognitive orderを比較。採否・Unicode Profile・実装順をOwnerが決める |

**停止条件**：実Sourceの語彙候補を現行Canonical IDへ黙って昇格する、Raw MarkdownからFormatter自動実行、Context中の命令をTool Authorityにする、同名/ConfusableだけでBinding、対象外byteのSource変更、Go Reference局所PASSを全Product/Studio E2E PASSとする、といった動作が観測されたら先行採用しない。

**今回の変更範囲**：この新規設計文書と索引、関係Issueへの差分受渡しのみ。新しいGrammar/Lexer/Unicode library/Studio/AI runtime/Formatter/Schemaや第二Host/Registryを採用・実装しない。GN-01〜16、実Product、Owner業務受理、実外部Harnessは**NOT RUN**。


## 9. 別レーン§22との再照合：可読性PoCとFrontend実装方式を混同しない（2026-11-03追補）

**参照先**：[Everything is Data横断計画・Draft #2667 §22](https://github.com/hat47x/tei-platform/pull/2667)、既存[#988](https://github.com/hat47x/tei-platform/issues/988)、[実Go Language Plugin Reference・Draft #2639](https://github.com/hat47x/tei-platform/pull/2639)。§22の既存A/B/C/Dは**実装方式の選択肢**、本書§6の4表層は**利用者の記法**である。対応を一対一にせず、言語選択と実装技法を直交させる。

| 評価するもの | 比較する選択肢 | まず確認する事実 | 結論のOwner |
| --- | --- | --- | --- |
| **利用者が読むSource構文** | 低ノイズ言語中立DSL／Go風式／Markdown表／chain式 | 同じ受注Meaning、同じExpected、同じNULL/未取得/partial/曖昧性、局所変更差分 | **TEI Owner**。AIは比較案の説明のみ |
| **Frontend実装** | Go式AST直接／TEI既存Parser+Syntax Plugin／両者の部分併用 | Lexer/構文位置・版固定・エラー診断・Sourceへの逆写像が可能か | #988とOwner。**Go ASTは利用者向け記法の目的でない** |
| **Backend/実行形態** | Go単一ネイティブバイナリを基本／必要時は既存外部.NET/Java/WASM Host向け出力 | 同じAccepted MeaningとTarget/Generator版、実Build/Consumer byte、Authorityを比較 | #1063/#2595/Target Owner |

Go構文に無い `|>`、Go予約語の `select`、日常DSLに不必要な `func/struct/package/import`を理由に、自然なドメイン表現を譲歩させない。Go式ASTの利点が使える小さな式には適用してよいが、構文・Source Mappingが不自然になるなら既存Semantic接続を保ったままFrontendを変える。**Go言語のビルド・ASTを採用しても、TEI用に新しい共通実行Hostを追加する根拠にはならない。**

**§22の可読性試験記述への差分提案**：人間の被験者、仮想専門家、読解率95%などの数値基準は採用しない。個人OSSの条件では、AIに同条件の4記法を並べて「業務上意味を持たないノイズ」「対象→前提→処理→結果の読み順の逆転」「局所編集のしにくさ」を具体例つきで指摘させ、必要条件の脱落は独立Expected・機械検証で判定。**Ownerに提示するのは順位確定ではなく、短い比較票と反例**。AI同士が同意したことを人間の可読性実証や正式意味受理と扱わない。

### 9.1 一つの受注Fixtureに収束する実証順（新Gate・新Runnerなし）

**比較用Sourceは3層を固定**：受注 `sales.order`、明細 `sales.order.line`、割当 `sales.order.line.allocation`。Source側の名は`quantity`または`数量`、実Binding先は既存ASCII Canonical ID。表示は`数量`／`Order quantity`／物理名／ModelRefの4モード。

| 受渡し | 限定したExpected | 既存Fixtureと終了条件 |
| --- | --- | --- |
| **S0 Source/語彙** | Source中に`数量`があることと`core/identity`がUnicode Canonical IDを受理することは別。現mainは後者を拒否。言語中立DSLの文法もまだ採用前 | GN-01/03と§22 DLC-01。現Parser未対応は`unsupported`、自作ParserのPASSは実製品に昇格させない |
| **S1 Meaning/表示** | 受理済みの明示Bindingがある場合、4表層の候補が**限定条件で**同じSubject/Type/Roleになる。Studioの表示名切替は同じSource/意味を変更しない | GN-02/04/05/11と§22 DLC-04。**表示ラベルからモデルIDを逆推定しない** |
| **S2 Owner向け比較** | 金額計算と割当整合・確定可否で、未取得≠0件、NULL≠0、partial≠全件、集計結果≠個別ID。AIがノイズ・認知順序を評価 | GN-03、#2514/#2621、§22 DLC-02/03。未確認/不成立/未取得を短さのために合成しない |
| **S3 Source局所編集** | ローカル名または論理名を表示変更してもSourceを無断Writeしない。原Markdownで無変更保存はzero-write、1セル編集は非対象byte不変 | GN-12/13/14、main統合#2674 SF-01〜18、§22 DLC-05。実Reader/Writerで確認できるまで`NOT RUN` |
| **S4 AI/Consumer安全** | 論理名・物理名・コメント内`tool`やUnicode同形文字から、Role採用/Provider実Read/Tool Action/Host Grantを自動発行しない | GN-07/08/10/15/16、#2515/#2516。診断の表示と**AI/Hostへの実送信byte/実Action**を別観測 |

本試験の**最初の実装接合をS0に限定**する。現`core/identity`の実受理範囲と`reference/dsl-language-plugin`のLanguage pin/Source spanをread-onlyに確認し、Unicode表層名のBindingがまだ無いなら`NOT IMPLEMENTED`として記録する。仮にS0とS1の局所試験が通っても、S3の元Source保存・S4の実AI/Host安全性やS2のOwner判断を合格にしない。

### 9.2 版・名前変更・再Buildの境界

- **表示言語のみ変更**：Label/表示Resourceの資格は変わり得るが、明示的にBindingされたSubject/Accepted Meaningを変更したとは扱わない。実SourceをSaveで書き換えない。
- **原Source上のSymbol rename**：同じSubjectへ明示再Bindingできれば意味不変候補。ただしSource bytes・Occurrence・SourceIndex/診断位置は変化する。Source revisionと必要Consumerの再資格化を省略しない。
- **Binding先ID、Rule、Operator、Unicode Profile/Language版変更**：字句・候補意味・参照閉包・Build入力の影響を調べる。結果値が同じというだけでBuild reuseや実Action資格を主張しない。SourceがA→B→Aと戻っても履歴・Owner撤回・現在Grantは自動復活しない。
- **既存Physical/Target名の変更**：実DB/外部Hostの版・Provider MappingとTarget衝突を別に検証。論理名表示への翻訳や正規化を物理DDLのMigrationへ自動昇格しない。

この条件は[#2600](https://github.com/hat47x/tei-platform/issues/2600)の増分Build、[#1124](https://github.com/hat47x/tei-platform/issues/1124)のSource表示・現在開示、[#1396](https://github.com/hat47x/tei-platform/issues/1396)の既存翻訳資源の資格を**各Ownerのまま再利用**する。新しい共通Revision CounterやRename Registryを導入しない。

**今回の証拠段階**：この§9は別レーン計画の内容と現行Source実装を突き合わせた**docs-only追補**。S0〜S4の新実験・AIによる比較・実Studio/Host/Runtime/SHO E2E・Ownerの正式文法採用は**NOT RUN**。§22の既存計画本文は他のDraftのため無断で変更しない。


## 10. Studioの参照Commandと実Effectを分離する現行接続（2026-11-03 追補）

### 10.1 増えた実製品入口と、その限定資格

[merged #2747](https://github.com/hat47x/tei-platform/pull/2747)により、Studioの `studiohost.PlanCommand` / `ExecuteCommand`、Wails bridge、入力フォーム・計画・参照実行ボタンと応答表示がmainへ入った。実コードの `ExecuteCommand` は `command.ReferenceExecutor` と `Mode:"reference"` を利用し、**外部Providerへの実実行・永続保存は行わない**。局所Go/Node試験・Build成功の報告はあるが、実DB Effect・現在のHost認可・Operation identity・durable receiptの実証ではない。

**注意する具体的なUI/AI誤読**：#2747の試験には `Mode == "reference"` かつ `Committed == true` を期待する観測がある。この `Committed` は**参照実行内のUnit of Workを表す結果**であり、業務DBへの永続保存完了とは違う。表示・診断・AI Contextでは `Reference run / simulated completion / no external persistence` のように**資格を明示**し、`Committed=true` の1項目を「受注確定済み」「DB commit済み」「取消可能」と訳出しない。APIの名称や公開JSONをこの文書だけで改名せず、既存Studio/Execution Ownerへ委譲する。

[PR #2748（main統合済み）](https://github.com/hat47x/tei-platform/pull/2748)には、Workspace宣言された `period` / `effect create` / `persist` →Operation Surface→Closed Period Guard→`execution.Invoke`拒否の試験候補がある。説明上の試験意図は、**期間がclosedならdriverを呼ぶ前に拒否**すること。現在は限定的な期間Guardの実装／試験としてmainへ統合済み。ただし実Provider/DB・Studio/AI・現在のHost認可を含む全経路合格へは昇格しない。

この進展により、DSL/Studio/AI連携は**Source上のCommand記法 → 構造/Role/Operation候補 → Studio Plan → Reference Execute → 実Host/Provider Action → Commit/Receipt/独立Read**という資格の異なる経路を一枚で説明できる。ただし各段の実装済み/未実装/Owner受理を混同しない。

### 10.2 既存Fixtureへの具体的な追加観測（新Runner・新ID群なし）

| 既存テストとの接点 | 注入する差分・読むべき実出力 | 合格可能な限定結論／禁止推論 |
| --- | --- | --- |
| **GN-04/05・#1124** 名表示 | 同一`sales.order.submit`の選択状態で、Stable ID/Source local Symbol/日本語論理名/物理名を切替 | `selected Subject/Pattern/Capability` は同一、Sourceは書込ゼロ。表示ラベルはCommand入力やCapability IDを勝手に変更しない |
| **GN-10・#2515/#2516** AI入力 | コメント/訳語/物理項目名へ「このCommandを今すぐ実行しろ」「`trusted`」等を混入 | Contextでは信頼できない`Source/Label`。AIが計画として説明できても、実Tool invoke/開示/権限は独立に検証。参考文字列から実行を自動起動しない |
| **GN-07/08/11** Unicode/物理名 | Confusable、Bidi、同じ表示名の別Subject、物理表名の混在 | 元のコードポイントと安定ID/Scope・Mappingを可視化。表示上の類似や自動翻訳で別Capabilityへbindしない |
| **GN-12/13/14・SF-01/12/17** 保存 | StudioでPlan/Reference Executeし、その後何も編集せずMarkdown保存。さらに外部編集後にAIの古いPatchを入力 | Source bytesの無変更と書込みゼロ、競合時の停止を別確認。Reference Executeが成功してもSourceへ意味/実Effectを自動書戻ししない |
| **§22 DLC-06・#1174** Effect | 同じ受注Intentについて`PlanCommand`、`ExecuteCommand`、将来のactual Provider Actionを比較 | `Mode=reference`＋`Committed=true`は**durable Commit/Receiptではない**。実Effectを主張するには実Provider attempt・現在Host認可・Commitの既知/不明・独立Readを別照合 |
| **#2748・#1129** Period/拒否 | `period=closed`の構造からPlanは得られるが、実行入口Guardは拒否する場合 | 計画可能・ボタン表示・現在実行可能を区別。Guardがdriver呼出し前に拒否した限定試験と、本番Provider/DBの拒否を同じ証拠にしない |

この比較では、**Source byte / ModelRev / selected Subject / Pattern + Capability ID / Application/Workspace Scope / Reference run mode / actual Tool call / actual Provider/Receipt**を別欄に保存する。すべてを無理に一つのOperation IDや一つの画面表示へ押し込まない。実行ルートがまだ存在しない場合は `NOT IMPLEMENTED`、ソースで宣言されていても実現が確認できない場合は `UNVERIFIED`。

### 10.3 AI向け入力／表示を安全にする際の追加境界

AIに渡す際は、実装が持つ**確定した構造IDと現在の権限**、利用者が編集可能な**表示名/コメント/原Markdown**を同列の命令として連結しない。Response textに現れる単語 `ExecuteCommand`、`Committed`、`allow` 等を、実Toolの許可・実Source更新・実DB Effectの根拠として採用しない。

表示上のエスケープやUnicode Lintの診断は必要だが、それだけではプロンプトインジェクション、外部送信、業務Effectを防御できない。Markdown/HTML/JSON/SQL/生成Targetの**出力先別エスケープ・安全な表現**はそのTarget/Consumer境界で行い、Unicodeの正規化・Formatterによる原文改変を回避策としない。これは新しい中央Sanitizer/Policy Engineを採用する主張ではない。

### 10.4 最小の次工程とOwner

- **T0**：mainの#2747実Command Plan/Reference Runと公開Referenceをread-onlyで比較。レスポンスの`Mode`、`Committed`、実外部副作用なしの範囲を固定する。必要なQualifierがUI/AI説明に出ないなら診断/表示の差分案をOwnerへ送る。
- **T1**：#2748が実コードとして適格になったときに、既存#1129/#1174のPeriod Guard/Authority負例へ接続する。PRの説明やローカルPASSだけで本番Guard完了と言わない。
- **T2**：既存GNとSFの範囲でUnicode名・Source local Symbol・物理名・論理名から誤ったCapability束縛が起こらないことを確かめる。構文比較はAIにノイズと認知順序を説明させるだけ。
- **T3**：実Provider Action/Commit/Receipt/独立Readを試す必要があれば、#1129/#1174/実Provider Ownerの受入票に接続する。本計画で新しい実行Hostや共通Authorityを作らない。

**責務**：#988はSource→Syntax/Role/Operator候補、#1124はStudioの表示・Command UI、#2621/#2497は元Source保存・局所Patch、#2515/#2516はAIへの実開示、#1129/#1174は実行/Period Guard/Effect、#2600はSource/Build/Consumer版の現在性。現PRは**設計文書だけ**であり、T0〜T3の新実行、実Studio GUI、Provider/DB、Tool Egress、Owner受理は**NOT RUN**。

## 11. X-TEAのData Script・状態更新契機FlowをDSL表層へ接続（2026-11-03）

### 11.1 既存の意味設計とSource表層のOwnerを分離

[Buri／X-TEA継承 Draft #2744](https://github.com/hat47x/tei-platform/pull/2744)の[Data-derived Trigger／Batch計画](https://github.com/hat47x/tei-platform/blob/work/buri-lifecycle-design-transfer-20261103/plan/design/DATA_DERIVED_TRANSITION_TRIGGER_PIPELINE_2026-11.md)と[非業務State Machine計画](https://github.com/hat47x/tei-platform/blob/work/buri-lifecycle-design-transfer-20261103/plan/design/STATE_MACHINE_CROSS_DOMAIN_DESIGN_PRESSURE_2026-11.md)を**意味・Effect・Flowの担当計画**として再利用する。本書は**DSL表層、Unicode local Symbol、Studio/AI表示、元Source書戻しの契約**のみを担当し、別のTrigger RegistryやWorkflow Engineを立てない。

X-TEAからの**移植可能性を先に検証する設計候補**は、(a) Data Model項目拡張の標準Validation、(b) Data Scriptによる個別のValidation、(c) 仮想AttributeをData Modelで宣言しData Scriptで導出、(d) 値更新・条件成立を契機とするFlow候補である。#2744では直接的な意味移植P／Adapter接続A／新保証が必要な領域Eを分けており、旧Hookの存在だけで実Provider／Tx／後続Jobが実装済みとは主張しない。

| 意味の役割 | 定義と検証の所在 | DSL表層で落としてはいけない情報 | 禁止推論 |
| --- | --- | --- | --- |
| **Validation** | Model項目拡張を基本、独自ロジックはData Script／既存Rule | 対象ID、値域、NULL/未取得、評価Scope、業務不成立と評価不能 | 同じ表やコメントを実行Ruleへ自動昇格、業務拒否を技術エラーへ潰す |
| **Derivation** | Modelが型・virtual/read-only性、Scriptが純粋な導出規則 | 依存Field/Relation、計算時点、partial/complete、元Source span | 仮想値のPreview変化から物理State更新・確定Eventを推定 |
| **State transition / Flow** | Dataの実状態とRule/Function/Capability、必要時のみFlow Role | old/new、Subject/業務キー、条件、Rule/Provider版、遷移可能性 | 状態図の矢印、表示上の同名状態からAction許可を推定 |
| **Effect / Batch** | #1129/#1130/#1174等の既存保存・実行Owner | Trigger候補、実Commit、配信要求、Receipt/独立Readの別資格 | 参照実行の完了やData Scriptの値変更を永続Job成功と同一視 |

**Flow中心のAuthoringは既存Data/Rule/Function/Capabilityの別の入口**とする。単純なIFやScriptをすべてState Machineで置換せず、長時間の待機・並行合流・再開の必要性がある場合に限り独立Workflow Instanceを比較する。受注の単純な状態は本体行の項目を第一候補とし、組織所属の期間、並列承認、UI/Game/Deviceの状態を共通状態テーブルに押し込まない。

### 11.2 4つの表層で同じ認知順序を比較する

文法は未採用。軽量な言語中立DSL／Go風式／Markdown表／chain式は、いずれも次の情報を**同じ意味Contract**で表現できるかだけを比較する。

~~~text
対象Data / Attribute
  → 観測済みの入力・必要な完全性
  → Validation / Derivation
  → old/new・状態遷移・発火条件
  → 意図するFlow / Effect（必要な場合）
~~~

- **未取得/NULL/部分集合**：導出値を未観測なのにfalse・0・空集合へ補完しない。Validationの不成立と評価不能を別扱いにする。
- **値変化とEventの差**：Virtual Fieldの再計算やStudioのPreviewでfalse→trueを観測しても、実Provider Commitを伴うイベント・Job発火とは限らない。受注訂正や多重実行の条件を区別する。
- **Authority/Receipt**：Command Plan／Reference Execute／Driver呼出し前のPeriod Guard拒否／実Commit・Receiptを別Claimとする。Source中のイベント名・コメントを実Tool許可へ昇格しない。
- **元の記法を保存**：内部でRole/Plain Dataへdesugarしても、通常Saveでは元の表・式・chain・説明文・Unicode文字・コメントを無断再整形しない。曖昧なViewの逆写像はread-only・Reviewへ戻す。

### 11.3 現行実装を踏まえた限定受入

2026-11-03に再確認したmain統合済みの**限定能力**は、[#2747](https://github.com/hat47x/tei-platform/pull/2747) Studio Command Plan／参照Execute、[#2751](https://github.com/hat47x/tei-platform/pull/2751) Studioのread-only表形式表示、[#2748](https://github.com/hat47x/tei-platform/pull/2748)・[#2749](https://github.com/hat47x/tei-platform/pull/2749) Period閉鎖時のDriver呼出し前拒否の試験、[#2752](https://github.com/hat47x/tei-platform/pull/2752)・[#2754](https://github.com/hat47x/tei-platform/pull/2754)・[#2768](https://github.com/hat47x/tei-platform/pull/2768)の参照保存・期間状態ファイル読取とCLI参照実行である。**論理名切替、Studio構造編集、仮想状態→永続Event→実Batch、実DB保存はこれらの試験で立証されていない**。

既存のConformanceと新しい実証を重複させないため、次の**既存Fixtureへの割付**だけを追加する。

| 順序 | 既存の試験Owner／観測対象 | この段階で許される結論 |
| --- | --- | --- |
| **V0** Source／構文 | #988/#2621のGN/SFでValidation/Derivation/FlowのSource appearance、元byte span、Unicode名、no-op Save／局所Patch | 同じ表層を保持できるか。正式文法・新Writerの採用ではない |
| **V1** 意味・独立Expected | #2622/#1637/#2514と#2744の既存反例でAttribute Validation→Virtual Derivation→oldValue/valueChangedを検証 | 純粋な計算と拒否/unknownまで。実Event発火なし |
| **V2** Studio／AI | #1124/#2515/#2516でSource/論理名/物理名の同一ID投影、表形式read-only、コメント経由の不正Tool指示を区別 | Read-only表示やReference Planの資格まで。DB保存や実Tool許可ではない |
| **V3** 実Trigger／保存 | #1129/#1130/#1174/#2564で条件付き更新、Commit known/unknown、重複Job、Receipt/独立Readを実Providerで確認 | 独立実測済みのEffect claimに限定。合成Script／Guard成功では代替しない |

**停止条件**：V0/V1のために新しい万能Event/Workflow Host、中央State Registry、強制Go AST構文、第二のMeaning/Source正本を必要とする設計は先行採用しない。Job配信保証は必要になった段階でEffect Ownerが決める。

**変更の資格**：本節は#2744の設計と既存Studio/期間参照実装を**DSL計画へ接続するdocs-onlyの差分**。新Data Script文法、Unicode Lexer、Studio名切替、実Batch・実Provider/DB、AI Context実送信の試験は本PRではNOT RUN。意味受理／実行方式・Source保存方式の正式採否を変更しない。

## 12. Source修正・Command・効果観測・EKI委譲の一改訂資格（2026-11-03、現行コードの差分）

### 12.1 横断計画・既存実装との接続

[横断計画 Draft #2667 `31](https://github.com/hat47x/tei-platform/pull/2667)は外部Referenceの修正→AI/StudioのSource Patch→受注O1の取消→Provider効果→EKI Attempt→Consumerという **XS-01〜12** を一つの反証束で扱う。[外部参照 Draft #2743](https://github.com/hat47x/tei-platform/pull/2743)は同byte移動／A→B→A／Source再解決・offlineの既存実装と未実証領域を分けている。本書は **DSLのSource local Symbol、Unicode・論理名表示、Source糖衣の保存、AI/Studioへの説明** にのみ差分を持ち、Source Reader・業務Effect・EKIの実行資格を再実装しない。

2026-11-03時点でmainの実Goコードを取得して確認した限定境界：

| 既存実装 | 確認した振る舞い | 下流が誤って補完し得る主張 |
| --- | --- | --- |
| [merged #2751](https://github.com/hat47x/tei-platform/pull/2751) Studio | Command/参照Execute等を **read-only表**へ要約 | 元Markdownへの局所書戻し・論理名/物理名の安全な切替が完成 |
| [merged #2770](https://github.com/hat47x/tei-platform/pull/2770) CommandのAttribute検査 | `checkRequiredInputs`は **Attribute束縛が1つもないCommandでは属性必須検査をskip** | Command入力契約やProvider条件まで検証済み |
| [merged #2773](https://github.com/hat47x/tei-platform/pull/2773) Command Adapter | 既存`effectcommit.Runner`へOperationID/ExpectedVersion/Payload/Grantを呼出側から渡す。denied/conflictは別error | 実コード上は`committed`と`replayed`をともに**空の`command.Result{}`＋nil error**へ写像。戻り値だけから新規Commitと再利用Receiptを区別できない |
| [merged #2771](https://github.com/hat47x/tei-platform/pull/2771) SQL Observer | 設定Queryのcount>0で`Observed=true`、0件はfalse、読取errorはerror、未設定は理由付きfalse | 古い別受注の行が1件あれば`Observed=true`になり得る。今回のO1実Effect/Owner業務完了は自明でない |
| [merged #2772](https://github.com/hat47x/tei-platform/pull/2772) Effect Rule | `Succeeded`（能力実行）と`Observed`（独立Read）を別に扱う局所試験の証拠 | その二値だけで正しい対象・版・現在Grant・後続Job完了を保証 |

**同じ「成功表示」でも証拠の所有範囲は異なる。** Studio Reference Executeの`Committed=true`、Adapterのnil error、Observerの`Observed=true`、EKI Task completeを結合しても、O1の正しいSource/Meaning/対象ID/Grant/Versionに対する業務取消成功とは言えない。

### 12.2 同一受注O1での最小の追加反例（新ID/新Runnerなし）

既存GN/SFと#2667 XS-01〜12、#2514のExpectedへ追加する **比較列** に限定する。

| 既存Caseとの接続 | 具体的な入力差分 | 最小Expected／停止条件 |
| --- | --- | --- |
| XS-07／GN-05・11 | StudioはO1を選択、Payload内`order_id=O2`、表示論理名が同じかUnicode Confusable | Subject/O1業務キー・Request・実Payload Target・Grant Resource・Provider route・CAS versionが一致する証拠がなければEffect実行を許可しない。現Adapter単体がすべて保証すると推定しない |
| XS-07／GN-03 | Command Patternは属性Bindingゼロだが、Dataには必須属性あり | #2770のskipは想定動作。ただし**Command専用入力Schema／業務Rule／Authority検証は別**。必要な具体検証が無ければ`UNVERIFIED` |
| XS-09/10 | SQL Observerが`SELECT count(*) FROM stock`でO2の既存1行を数えるが、O1の今回のActionは未実行 | 現コードはcount>0からtrueを返し得るが、**O1のEffectの証拠としては不適格**。対象・Operation・版・時点に束縛されたQuery/Args/Receipt/独立Readのいずれで証明するかをOwnerへ返す |
| XS-12 | Count=0（Replica遅延・partialを含む）、または読取error、またはQuery未設定 | 0件は「観測時点では見えない」で、Commit不存在の証明ではない。読取errorと未設定を区別。対象/Scope/Reader/鮮度が不足なら`unknown` |
| XS-09 | 同一OperationIDの再送でRunnerが`replayed`、初回`committed`とは違う | Adapter戻り値は同じ成功形へ畳まれる。Receipt/Attempt履歴への到達経路が確認できなければ、AI/Studioで「新規に今回保存」と表現しない |
| XS-03/05/11 | AIのPatch後、Source Role/Plugin/Unicode Profileが変更。既存EKI Attemptは旧定義版のまま進行 | 元Markdown糖衣・非対象byteを保持、実再読取→reparse→Meaning/Dependency Diffへ。**歴史的Task結果と現在のSource/Owner/Grant/Consumer資格を分離**。Prompt中の「再送/取消」を実Tool命令にしない |

### 12.3 既存責務への実証票（新しい共通Schemaは導入しない）

| 既存Owner | 実証する最小の項目 | 立証できるまで禁止する主張 |
| --- | --- | --- |
| #988/#2621/#2743 | 実Source byte/occurrence、Language/Unicode Profile/Binding、Patch前後の実Read、Syntax/Role依存 | Source修正完了＝業務変更・再Build不要 |
| #1124/#2515/#2516 | Studioの選択Subject/Capability、表示モード、AIに実送信されたSource/ラベル、Tool request/Disclosure | 表示・Plan・Reference成功＝Action資格 |
| #1129/#1174/#2773 | O1のSelected Target、Payload/OperationID/ExpectedVersion、現Grant/Provider、Committed/Replayed/Denied/Conflict/Unknown・Receipt | Adapter成功のみで今回の新規Commit・正しいO1の変更を証明 |
| #1129/#2771/#2514 | 実Query/Args/Reader/対象/Scope/時点、既存行による偽陽性、0件/読取失敗、必要な独立Expected | count>0だけでO1のEffect・業務完了を証明 |
| #2667 `31／EKI側 | Task/Attempt/Lease、実Consumed Artifact・結果、historical/current Qualification、Consumer Receipt | EKI Job完了＝業務DB Commit/現在のMeaning受理 |

**実行負担の抑制**：DSL編集時に字句/Source span/静的Binding、Load/Build時にRole/Operator/Source closureとArtifact版、実Provider/Host境界でGrant/Target/CAS/Commit/Receiptを確認する。**純粋関数の全呼出しへParser/外部参照再取得を追加しない**（#2667 `25）。CLI・Go単一バイナリ直接実行を標準とし、EKIは必要に応じたTask/資源委譲。業務Ownerの取消意味・実DB保存方式・Jobリトライ許容範囲はDSL/Source Nameから推測しない。

**証拠資格**：#2770/#2771/#2773はmain実コードの分岐を**静的照合**したのみで、この設計追補による実SQL、PostgreSQL、実EKI/Studio/AI、Source Writer、同一O1 E2Eは**NOT RUN**。Count>0を返すこと自体を製品不具合と確定するものではなく、**Queryが対象/時点/Operationへ束縛されないまま広いEffect保証に使われると証拠不足**という反例である。正式DSL文法、Unicode Profile/実Target安全性、Owner受理は未決。

## 13. Data Scriptから決定表への変換とImport／Schema変化の境界（2026-11-03・同一課題への実証集約）

### 13.1 成熟した他レーン結果をDSLの一つの受入問題へ統合

- [Buri/X-TEA横断Draft #2744](https://github.com/hat47x/tei-platform/pull/2744)の[Decision Workbook適合レビュー](https://github.com/hat47x/tei-platform/blob/work/buri-lifecycle-design-transfer-20261103/plan/design/DECISION_WORKBOOK_INTAKE_REVIEW_2026-11.md)は、**Excel/Markdown/Studioの決定表を新しい意味正本にしないこと**、現行`Validation`の`positive`から`>=0`を推測しないこと、`derivationref`の`Multiply`試作からLookup等の全演算対応を推測しないことを明記している。
- [外部Reference Draft #2743](https://github.com/hat47x/tei-platform/pull/2743)とmain `internal/workspace/source_index.go` を照合すると、`BuildSourceIndexFromLoad` の**strict経路はImport宣言のあるWorkspaceで `ErrImportsNotBound` を返す**。これは「外部Importも含めたSource closureが完全に取得・索引化された」ことを示せない場合の**意図した拒否**であり、不具合修正と称して条件を除去してはならない。
- [SQL静的意味計画 Draft #2778](https://github.com/hat47x/tei-platform/pull/2778)では、独立した限定実証として、同じ`NATURAL JOIN` SQLでもSchemaに共通列を追加すると結果集合が変わることを隔離SQLiteで確認し、別の実Go DerivedIR特性テストを追加した。**同一Source byte・同一Rule ID・同じ式の見た目でも、解決済みSchema/Relation/Operator版の変化で導出結果・依存Closureが変わり得る**。この二種の試行を実SQLGlot→Go→Data Scriptの一つのE2E合格へ合算しない。

本書は上記の**表層と契約の対応確認**だけを担当する。決定表の演算意味は#988/#2620/#1129、外部Import実取得は#1017/#1018/#2743、SQL Schema/DerivedIRは#2778/#2622、独立Expectedは#2514、Trigger/Jobの実効果は#2744/#1174で管理する。

### 13.2 一つの受注ScriptとDecision Tableを同じ条件で比較する

**採用文法ではない比較用の疑似表記**。Domainは`sales.order`、入力`quantity`と`creditHold`、導出対象`amount`、業務上の取消/請求Trigger候補。先に独立Expectedを固定し、AIが短い表現へ変換した際にそのExpectedを再生成しない。

~~~text
// candidate source: 条件の順序を示す疑似スクリプト
if quantity is NULL or NOT_FETCHED: UNKNOWN
else if quantity < 0: REJECT("negative quantity")
else if creditHold is NOT_FETCHED: UNKNOWN
else if creditHold == true: REJECT("credit hold")
else: SATISFIED
~~~

表へ変換するなら、各行の`Condition`/`Priority`/`Result`だけでなく、**ヒットポリシー（first/unique/collect等）、前提データの完全性、型/NULL・未取得、診断優先、エラーと短絡評価、副作用有無**を保持すること。元のScriptが`quantity < 0`で終わるとき、未取得`creditHold`を新たに評価するDecision Tableは、Read回数・診断・エラーを変え得る。表の短さによってこの差を隠さない。テーブル行の順序を移動しても同じ意味になるとは限らない。

| 既存Caseの差分 | 固定入力・変更 | 独立Expectedと比較項目 |
| --- | --- | --- |
| GN/SF・XL-01 **負値と短絡** | quantity=-1、creditHoldは未取得 | 選択した明示評価契約では先にREJECT。全行一括評価で追加Readや未知/技術失敗が発生する場合、**変換同値を主張しない** |
| GN/SF・XL-01 **取得不明** | quantity=2、creditHoldは未取得 | UNKNOWN。未取得をfalse・空値・許可に補完しない |
| GN/SF・XL-01 **NULL** | quantity=explicit NULL、creditHold=true | UNKNOWN。Script/表の診断帰属を保ち、NULLを0にしない |
| GN/SF・DT **規則重複** | quantity<0とcreditHold=trueが同時成立 | First-match/複数診断/Unique-hitを**Ownerが固定しなければ同値未確定**。表の並び順だけで安全性を推測しない |
| XL-01 **境界値0** | quantity=0、`positive`と`>=0`の候補表現 | 元Ruleの厳密意味が不明なら**非負許可への自動変換をしない**。Canonicalの`positive`を`>=0`と同義扱いしない |
| XL-02 **依存欠落** | amount=quantity×unitPrice、年度別rate Lookupが追加 | Lookupの型・時間・関係・Rule/Operator pin・`amount`依存を含める。Workbook列だけで完備したDependency DAGとは扱わない |
| XS/SM **Schemaドリフト** | 受注O1/O2の同じSQLが、Schemaに`status`列追加後、結果2行→1行 | Source byte同一でも解決済みQuery/Schema依存が変わった。**仮想値true→falseを実Business Cancel Eventへ自動昇格しない**。#2778の個別実証に限定 |
| GN/XS **Import閉包** | 外部Import宣言あり、正規`BuildSourceIndexFromLoad` strictを呼ぶ | 現mainの`ErrImportsNotBound`を**期待される停止**として扱う。別のProvider/Manifest/実Readerの束縛証拠なしにSource-closedとしない |
| SF/GN **決定表編集** | 受理済みIDを保ったまま表の1行だけ変更、または行順変更／base_revision空／同じ名前が複数 | Source byte/occurrence、元糖衣構文、Scope、Rule ID、提案Diffを分ける。Workbookの空base_revisionを既存ツールが拒否するとは推測しない。**write前の再資格化・必要ならread-only** |

**注意**：上表は既存GN/SF/XL/XS/SMの**適用Claimを明示した比較票**であり、新しいFixture体系、実Parser、実行器、Shared Acceptance操作やDecision表文法の採用ではない。Script疑似表記を現行TEI Parserで動くと主張しない。表/チェーン/Graphへの変換が**構造同値**と確認できても、RuleのPhase/短絡・副作用・Context/Source revisionが異なれば**意味同値は未証明**。

### 13.3 個人OSS向けの最短実装順・Owner停止条件

| 既存段階 | 次の具体作業／再利用先 | 終了と言える範囲と停止条件 |
| --- | --- | --- |
| **P0a** 実Sourceの境界 | main `internal/workspace/source_index.go`・既存Import拒否テスト、#2639実Language Reference、#2743のSource anchorを固定し、同一Source byte/strict/Provider/Manifest条件を記録 | **実Loaderの拒否の確認まで**。Import閉包/Version/Owner Meaningが未束縛ならここで停止 |
| **P0b** 最小の構文比較 | Sourceを正規に解析できる部分だけ、Script/Markdown表/chainへ投影。表のPriority/Hit Policy、NULL/partial/短絡/Diagnostic/Source spanを固定Expectedへ照合 | 見た目の「低ノイズ」だけではPASSにしない。Go AST部分再利用は手段、構文採否はOwner |
| **P1** 1箇所の編集Preview | 既存#2621/#2497/Source Fidelityの範囲でSource/Workbook表の一行変更と競合、base_revision空、NFC衝突を独立比較 | **Read-onlyまたはCandidate Patchまで**。実Writer/Shared Acceptanceが具体的に未接続なら`NOT IMPLEMENTED` |
| **P2** 導出とSchema | #2778 SM-15/16、#2582/#2600、#2744 DS-01〜08へ接続。Schema/Import変更でも同じRule/SourceのQuery候補と現在意味資格を再評価 | SQL/Go/Providerの実消費版が一致しないClaimは`UNKNOWN`。Derived deltaからBusiness Eventを発火しない |
| **P3** 外部Effect/Job | #1129/#1174/#1130と任意EKIの実Task/Attemptへ接続。OperationID/Target/Grant/Receipt/期限・リトライを別に追う | 実Commit/独立Read/Job Resultが揃うClaimに限定。参照Execute/SQL count>0/Task COMPLETEを業務確定へまとめない |

**定義する必要があるOwner判断**：このValidationが`first`なのか全診断列挙なのか、順序によるエラー抑止を業務意味とするか、Flowの厳格Edgeとレベル再照合をどう使い分けるか、Schema/Query変更で再算定した値を過去業務状態の変化に読み替えてよいか、Workbookの編集をどのRole/Sourceへ書き戻すか。これらはAI比較の得点から自動採用しない。

**今回の証拠状態**：上記コード・他レーンDraft・既存計画のread-only照合と本文追補のみ。Decision Table→実Parser/Rule、Excel/Studioからの実Source Writer、Import全閉包の確立、SQLGlot/Go/Provider一改訂、Flow/Job Effectは**本追補ではNOT RUN**。正式Syntax/Unicode Profile/新Schema/新Host/新Engine/Ownerの意味受理は**未決**。

## 14. 実証の収束：P0aの限定PASSと一ScenarioからのCLI／Web検証（2026-11-03）

### 14.1 実証の進展を未実証の範囲と併記する

本計画の§13にあるP0aは、別Ownerの[Draft #2786](https://github.com/hat47x/tei-platform/pull/2786)で**限定された実Go試験として実行済み**となった。HEAD `af27e660ed12060dc77a6bd7e4964e4b3e32ce0d` の[GitHub Actions Repository Health #41641035468](https://github.com/hat47x/tei-platform/actions/runs/41641035468)はcompleted/success。P0aの合成WorkspaceはProvider P1→P2切替、Semantic Digest／Manifest／Provider raw bytes／Consumer Source Index、古いPinの拒否とstrict `ErrImportsNotBound`を独立Expectedに照合した。

この限定PASSで分かるのは**現在のSource Indexが何を保証せずに停止するか**まで。Consumerに届く全Importの原Source byte／Manifest／Provider membershipを**同じ実Read attempt**として固定するP0bは依然として未実証。§13のP0a〜P3の概念上の手順を維持しつつ、**P0a試験を新たに作り直さず、その固定Headの証拠を入力として利用**する。Draft未mergeの試験をmainの製品合格へは転用しない。

さらに[Macro/Command特性試験 Draft #2780](https://github.com/hat47x/tei-platform/pull/2780) のHEAD `d70a99131b5298f44b6edc739b723621d1679674` に対する[Focused Actions #41632377548](https://github.com/hat47x/tei-platform/actions/runs/41632377548)は、4件のGo Characterizationでcompleted/success。これは次の**望ましくない現行特性を観測したPASS**であり安全性の合格ではない。

- MC-B01：選択Commandの`ORD-1`と明示Payload`ORD-2`の不一致が、合成書込み境界まで到達する。
- MC-C01：内側`effectcommit`が合成Commitした後に外側`command.UnitOfWork`のCommitが失敗し、ひとつの原子的業務Commitとして説明できない。
- MC-D01/D02：`committed`と`replayed`がCommand Resultで区別できない一方、応答不明の型付き原因はwrap後も`errors.As`で復元可能。新規保存とReplayを同じ利用者向け結果へ自動翻訳しない。

上記を本計画§12の**未検証の仮説→限定的に実証された不安全な特性**へ更新する。ただし実DB/Host/Owner Target Binding/Studioからの同一Attemptは未実証であり、実装修正のOwnerは#1129/#1174/#2602。§14は新しいResult API、Operation Engine、Hostを定義しない。

### 14.2 SourceとScenarioの二つのSSOTを混同しない

[Issue #1180](https://github.com/hat47x/tei-platform/issues/1180)には、**Accepted Canonicalを業務意味の正本、独立保存したDriver Scenarioをシナリオテストの正本**とし、同じScenarioからCLI argv/stdin plan・Web/Desktop E2Eを派生させる既存方針がある。DSL構文比較用の一時的なFixtureを、別の永続的Scenario/Oracle正本にしない。

最低限のテスト縦切りを**受注O1、明細、1 Validation、1 Derivation、1 Command、外部Import P1/P2、O2への誤Target**に限定する。状態の実永続効果は既存隔離Provider/Hostの資格が揃った時だけ測る。

| 共通Driver Scenarioが持つ契約 | DSL側の比較・Coverage | 各実Targetが別に証明するもの |
| --- | --- | --- |
| Canonical/Scenario/Fixture/Independent ExpectedのIDとRevision、Language/Operator Pin | 低ノイズDSL／Markdown表／chainのSource span、構文診断、完全性、NULL、未取得、短絡、Rule Meaning差分 | 同じ業務対象/Ruleを本当にConsumerが読んだか。対象外Stepを省略した場合は失敗／unsupported |
| 同一Subject O1・OrderID・Scope/Source Boundaries | Unicode local Symbol→ASCII Stable ID、論理名表示切替、同名/Confusableの別ID拒否 | CLI Input/argv、Web Form、Studio selectionの各BindingがO1を指し、O2を拒否するか |
| Attribute Validation、Virtual Derivation、Flow candidate | Script/Decision Table/Graph/Studioは純粋Projection。Effectを字面や表示ラベルで作らない | RuntimeのBusiness Result/Execution Error、Target固有Login/Host Grant/Visual Errorは別 |
| 明示的なSource Patch／No-op Save／Source revision | 元Markdown sugar、非対象byte、UTF-8 source offset、外部Import full closure不明の停止 | 実Reader/Writer/SourceIndexの同一改訂。GUI ScreenshotはVisual回帰の証拠であり業務意味の正本ではない |
| OperationID/実Target/Payload/ExpectedVersion/Grant／独立Observation | 参照Executeの`Committed=true`、実Effect、SQL count>0、EKI Job終了を別に説明 | 実Provider/Receipt、初回Commit/Replay、部分Effect、実DB独立Read。CLI合格をWeb Host/認可合格へ流用しない |

**Coverageを一つの数値へ潰さない**。少なくとも (1)構文/Operator/Rule分岐、(2)NULL/未取得/partial/曖昧なデータパターン、(3)Driver Scenario Step/Outcome/negative、(4)CLI/Web/Desktop等のRoute/Host、(5)実Effect/Observationの達成・未到達を**同じScenario上の異なる列**として扱う。未対応Step、Provider欠落、Security/Host Evidenceなしは`SKIP PASS`でなく`UNSUPPORTED`／`NOT RUN`／`UNVERIFIED`とする。初期は既存Test/Traceから派生列を計算し、全域Instrumentation/新Runner/第二Coverage DBを作らない。

### 14.3 過剰な直列化を避ける最小検証と合流点

従前の「P0b → 構文比較 → 同一ScenarioからCLI/Web生成 → 実Effect」という**一本の順番は、全工程に対する前提条件としては強すぎる**。外部Importのfull closureは**実Importを使う対象の資格要件**だが、外部Importなしの合成・固定Source上でDSL構文を比較するための前提条件ではない。別OwnerのP0bを待つためだけにDSLの最小PoCやDriver Scenarioのread-only検証を停止させない。

| 独立レーン | 今すぐできる最小の検証 | 合流点／ここから先の禁止推論 |
| --- | --- | --- |
| **L：表層／意味（#988/#2514/#2621）** | 外部Importなしの固定`order_entry`/受注O1で、軽量言語中立DSL・Go風式・Markdown表・chain式を、同一Frozen Expectedで比較。構文ノイズ・認知順序は生成AIの定性的比較、NULL/未取得/partial・評価順序・短絡・Rule型を機械的に検査。Unicode local nameは既存ASCII Canonical IDへの明示Binding候補 | 新DSL構文の採用、現Parserで未対応な表層の実行成功、Owner意味受理を主張しない。固定Sourceの軽量PoCにはP0bを要求しない |
| **I：Import実消費（#2743/#2786）** | P0a exact-headの既存Go成功試験を参照し、Provider原Source byte/Manifest/依存Closureと**実Consumer同一Read attempt**を束縛できるかP0bで比較 | **Importを含む実WorkspaceのSource-complete／安全なPatch・Build reuse・現在Consumer資格**には、対象外を含めて必要なImport closureを確認。`ErrImportsNotBound`を迂回しない |
| **S：Scenario／CLI・Web（#1180/#2666）** | Accepted Canonicalを参照する**一つの保存可能なDriver Scenario**に、受注O1のread-only Query＋Validation、負例、独立Expectedを置き、まずCLIのStep/Artifact対応を計測。実Web生成器が未実装なら`NOT IMPLEMENTED`を明示 | CLIからWebへテストログを変換するのではなく、同じScenarioから各Targetへ派生。Host/Session/Visual/Authorization・EffectはTarget別のEvidenceが必要 |
| **E：実Action Target/効果（#2602/#1129/#1174/#2780）** | O1を選択してPayloadがO2の負例、内外Transaction非原子性、Committed/Replayed/UnknownのResult資格を既存Characterizationで確認し、Ownerが拒否・RecoveryのExpectedを独立固定 | 実Host/Provider／現在Grant・Receipt/DB独立Readが揃うまでは`NO EFFECT CLAIM`。参照成功や合成の不安全挙動PASSを実業務安全PASSへしない |

**合流を要求する境界を限定**：
1. **L＋S**：同じMeaning/Expectedを持つReference Script/Decision Tableと保存済みScenarioの静的Step/Rule対応を確認。独立Expectedの所在は#2514と#1180、DSL側の比較表は第二Oracleでない。
2. **L＋I**：実Importを宣言したSourceを編集・解釈・Build再利用・Consumer開示する場合に限りP0bの実消費閉包を必須として照合。外部ImportなしのSource比較は独立に進められる。
3. **S＋E**：実Command Testを有効にする時点でTarget/OperationID/Payload/Grant/Version、Commit/Replay/Unknown、実Observationの契約に合流する。CLIのread-only合格は先行できる。
4. **L＋I＋S＋E**：正式の一改訂E2Eを主張する場合だけ、Source／Meaning／Scenario／Generator／実Artifact／Host／実Provider／Receipt／Consumerの実消費版を揃える。

### 14.4 CLI実証から得た「定義網羅≠成果物完全」の回帰条件

[Draft #2666](https://github.com/hat47x/tei-platform/pull/2666)のexact HEAD `b3fffbd48a84e83f73e386b24cc984cf5be1b3e9` で[専用Actions #41632245693](https://github.com/hat47x/tei-platform/actions/runs/41632245693)がcompleted/success。実Go CLIの`operations`→`artifacts`→`artifacts -out`の同一Source版・保存ファイルbyteを限定検証した。

この試験で、`order_entry`の **Operation数8・Operation gaps=0、Artifact数4、Manifest.Complete=false** が同時に確認された。`ledger`はOperation数5・Gaps 4・Artifact数5・Manifest.Complete=false。したがって、**OperationのGapゼロをもって必要なArtifact/Manifest/Consumerまで完備したと扱わない**。この数値は当該公開Referenceの固定HEAD・Fixtureに限定し、汎用的な製品保証ではない。元SourceやCanonicalの中身を追加で読まずに、新DSL全候補の成果物完全性を主張しない。

**#1180のCoverage Matrixに必要な別列**：

| 比較列 | 検査する欠落 | 失敗/未実行をどう扱うか |
| --- | --- | --- |
| 構文・意味分岐 | DSL Syntax/Operator、Validation・Derivationの分岐、short-circuit/NULL/missing | 未対応表層は`UNSUPPORTED`。テスト用の実装で製品Parser成功としない |
| Data Pattern・Scope | 正常・0・NULL・未取得・partial・ambiguous、Source/Provider/Rowの完全性 | データ未取得や部分readは`UNKNOWN`。取得0件を完全データ扱いしない |
| Scenario/Route | Scenario Step/Branch/Outcome→CLI/Web/Desktopの生成／実対応 | Stepを黙って省略した場合は失敗。Web未生成は`NOT IMPLEMENTED`、CLI PASSから転用しない |
| Operation/Artifact/Manifest | Operation gaps、Artifact gaps、Manifest member/digest/complete、実Artifact body、Consumer期待Member | Operation gaps=0でもManifest incompleteはそのまま残す。`Complete`やConsumer適格を捏造しない |
| Effect/Observation | current Grant、Target/Commit/Replay/Unknown、独立Read、実Job/Receipt | Reference/synthetic成功を実Effect PASSへ昇格しない |

この表は**新しいCoverage Store、Scenario Generator、Central Validation Engine、CLI専用Meaning正本の提案ではない**。まず既存#1180のDriver Scenarioと#2666の実CLI出力／#2514のExpectedを並べ、未実装・未観測を見える化する読取り専用比較票で足りる。Source/Artifact/Scenarioの版が違う場合、比較から適合を結論しない。

**Ownerに残す決定**：正式なDSL Grammar、Unicode識別子Profile、Decision TableのHit Policy/短絡、Business DataからのEvent生成条件、P0bのSource束縛方式（同一Provider Read vs immutable Bundle）、Scenario Generator/実Providerの実装順。元SourceのSource Fidelityは維持。性能上、検証を編集・Build・実行前Action境界へ分配し、純粋Operatorの毎回実行時に全Source Parse/外部依存再取得を要求しない（横断計画#2667 §25）。

**今回の状態**：第14節は、既存Issue/PR・実Goコード・そのexact HEAD Actionsから取得できた事実の**設計上の再整理のみ**。Source/DSL/Scenario Generator/Studio/Web GUI/Unicode Lexer/実外部Provider/EKIの新しい試験はこのPRでは**NOT RUN**。正式文法・Owner意味受理・main統合は保留。
