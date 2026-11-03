# TEI DSL：グローバル共通構文・Unicode表層名・Studio表示・AI入力安全性の統合計画（2026-11-03）

- **状態**：設計候補／必要な受入条件を既存Issueへ配分。**DSL文法、Unicode識別子Profile、Formatter規則は正式採用前**。本書は製品実装や受入済みADRではない。
- **親**：[#2619](https://github.com/hat47x/tei-platform/issues/2619) Everything is Data、[#988](https://github.com/hat47x/tei-platform/issues/988) Language Plugin/Operator、[#2621](https://github.com/hat47x/tei-platform/issues/2621) Authoring、[#1124](https://github.com/hat47x/tei-platform/issues/1124) Studio編集。
- **再利用する既存契約**：[#1120](https://github.com/hat47x/tei-platform/issues/1120) Stable ID/Trace（closed）、[#1396](https://github.com/hat47x/tei-platform/issues/1396) 多言語表示資源（closed、read-only E1実装の[#1404](https://github.com/hat47x/tei-platform/pull/1404) merged）、[#2497](https://github.com/hat47x/tei-platform/issues/2497) Projection/逆写像、[#2515](https://github.com/hat47x/tei-platform/issues/2515)/[#2516](https://github.com/hat47x/tei-platform/issues/2516) AI/Consumerへの開示。
- **既存原文保存仕様**：[main統合 #2674](https://github.com/hat47x/tei-platform/pull/2674)／[Markdown Source Fidelity](EVERYTHING_DATA_MARKDOWN_SOURCE_FIDELITY_2026-11.md)。保存は元のシンタックスシュガーを維持し、通常Saveで展開後の標準データ形式へ再直列化しない。
- **最新局所実証との関係**：[Draft #2639](https://github.com/hat47x/tei-platform/pull/2639) は実Go Language Plugin ReferenceとSemantic candidateの一部を試すが、**Unicode表層識別子／Studio名切替／AI入力安全の製品適合は未実証**。別Draft[#2667](https://github.com/hat47x/tei-platform/pull/2667)の横断計画本文を競合編集しない。

## 1. 優先順位：自然なテキスト記述と意味の厳密性を両立

**製品要件として維持するもの**：
1. **グローバル共通のDSL構文**。直接日本語をキーワード化した「日本語DSL」を正式候補にしない。ローカル言語は主にドメイン名・コメント・文書・Studio論理名として扱える。
2. **ドメイン専門家にも違和感の少ない認知順序と少ない構文ノイズ**を最優先。Go ASTで実装しやすいことは二次的な制約。日常の定義に \`package/import/struct/func\` 等のGoのプログラム用宣言を強制しない。
3. Everything is Dataの構文変換・標準構造・意味Roleは、テキスト表層と独立。Markdown表・軽量式・chain・Graphvizのどれかだけに意味的特権を与えない。**同じData/Attribute ID、型、値が同じでも、Rule/Owner/Provider/Contextが違えば業務意味同値とは限らない**。
4. **TEI Studioでは同一Subjectを表／Tree／Graph／Functionやインスタンスへ共通投影**。テキストの構文候補の比較では、GUIの豊富さを評価軸に混ぜず、静的なノイズ・認知順序・局所編集性・明示性を比較する。
5. **表層の名称表記とSource保存は別**。Studioの表示言語切替、AI向け説明、生成Target名の変換を元のMarkdown/.tei Sourceへ無断書戻ししない。

**未決**：具体的な予約語・演算子・識別子Profile・Unicode版・物理名の表記規則・カスタムFormatter。比較実験とOwner判断を経ずに文法や新Core Schemaを確定しない。

## 2. 五種類の「名前」を混ぜない

| 名前の層 | 例（合成の受注Domain） | 実際に何を識別するか |
| --- | --- | --- |
| **Canonical ID / ModelRef** | \`sales.order.quantity\` | 既存TEIで安定した意味参照。原則として言語・翻訳の変更では改名しない |
| **Source表層識別子**（Unicode許容は候補） | \`quantity\` / \`数量\` | .tei やMarkdownの特定Syntax Pluginで記述するローカル変数・Symbol。ScopeとSource occurrenceに依存 |
| **物理名** | \`T_JUCHU.JUCHU_SU\` | 外部DB/Host/Targetにある名称。Dialect・引用・実Schema・版に拘束される |
| **論理表示名／多言語資源** | \`Order quantity\` / \`受注数量\` | ユーザーが選ぶ表示言語のラベル。名前から逆に意味IDや物理対象を推測しない |
| **生成Target用Symbol** | \`orderQuantity\`等 | Go/Java/.NET/WASM等の生成物に必要な各Target表記。衝突と元ID対応を固定 |

**重要な現行実装上の事実**：mainの \`core/identity/id.go\` は \`ParseCanonicalID\` に \`^[a-z][a-z0-9_]*(\\.[a-z][a-z0-9_]*)*$\`、\`ParseLocalName\` に \`^[a-z][a-z0-9_]*$\` 相当の**ASCII制約**を持つ。\`SymbolToSegment\` も非ASCII Symbolを受け付けない。この検証は[#1992](https://github.com/hat47x/tei-platform/pull/1992)（merged）で同じ受理集合を保ちながら高速化された。よって、現状では**「UTF-8 Sourceが書ける」ことから「Canonical IDや既存SymbolにUnicodeを直接使用できる」とは主張できない**。

Unicode表層Symbolを評価する場合の最小案は、**既存ASCII Canonical IDを維持したまま、固定されたLanguage PluginのSymbol→明示的な型付きModelRef Binding**を検討すること。Unicodeからのローマ字化、翻訳、類似名による自動Canonical ID生成は行わない。参照先が0/2件以上、異Scope、別版、不正な名称の場合は候補/診断に留める。将来Canonical ID自体を拡張するかは互換性の大きい**別Owner判断**とし、本計画の前提にしない。

**例（DSL文法の採用ではなく表示比較）**：

\`\`\`text
// English ASCII source symbol
amount = quantity * unitPrice

// 同じ構文のローカル名例（現行Parserで動くと主張しない）
金額 = 数量 * 単価
\`\`\`

テキスト表層から同一Meaningへ到達できるかは**事前に確定したBindingとSource/Role/Operator Pinがある場合だけ**確認できる。単に名前を翻訳して同値とはしない。

## 3. Studioの表示切替とSource不変条件

表示モード候補は \`canonical ID\`、\`source symbol\`、\`physical name\`、\`logical name(locale)\`。現[#1396](https://github.com/hat47x/tei-platform/issues/1396) E1の既存Language Resource Catalogは**固定済み意味参照→確認済み表示資源のread-only解決**に使う。既存の翻訳資源から許される表示と、新たな物理名・Code symbolのRewriteは別。

- モード変更では**同じSubject ID、Meaning、Source byte、Source occurrence、Selection**を維持する。UIラベル変更でArtifact Revision/Model Revisionを勝手に進めない。
- ロケール資源が欠落・失効・未承認の場合、silent translationを避け、表示用の明示fallback／stable ID／unresolvedを区別。旧翻訳表示を現在の確定Meaningと取り違えない。
- 物理名は \`Binding/Source/Target/Version\`が一意に解決した場合だけ表示し、物理名からの語彙類似による参照選択をしない。等しい論理名が複数あるときはScopeやStable IDを併記する。
- Sourceの編集は元の糖衣構文へ**Source-aware局所Patch**。表示モードの切替それ自体では書込ゼロ。表幅やMarkdown formatterは通常Saveから独立。Source移動・A→B→A・旧Span・二つの同名Occurrenceには別の再資格化を要求。
- Read-only Studio Projection / Source編集権限 / Tool Action / 現Host Grant を混同しない。外部AIへ送るラベルやSource本文にはDisclosure Scopeを適用。

## 4. Unicode表層識別子Profileの評価候補（正式字句規則ではない）

参考資料：
- Unicode [UAX #31 Unicode Identifiers and Syntax](https://www.unicode.org/reports/tr31/)（2026-09-01 Unicode 18.0.0 / revision 45）。
- Unicode [UTS #39 Unicode Security Mechanisms](https://www.unicode.org/reports/tr39/)（2026-08-27 Unicode 18.0.0 / revision 34）。
- Unicode [UTS #55 Unicode Source Code Handling](https://www.unicode.org/reports/tr55/)（version 2 / 2024-01-29）。

**版を固定した仕様候補**： \`XID_Start\`/ \`XID_Continue\`等を基礎に、ASCII構文記号・数字先頭・予約語・Unicode識別子の正規化比較・危険なformat/control codeの扱いをLanguage Pluginに明示する。Unicode版更新で受理集合やConfusable結果が変わる可能性を版間差分として扱う。Go標準の\`unicode.IsLetter\`等がそのままUAX #31適合を保証するとは推測しない。

| 検査 | 候補動作 | 避ける危険 |
| --- | --- | --- |
| UTF-8妥当性とSource span | 不正byte列拒否、元byte offset／UTF-16 editor offset区別 | 日本語文字の途中でPatch／別Cell変更 |
| 識別子XID属性／script使用 | 固定Unicode版＋Language Profileで判断。合理的なローカル文字体系を一律拒否しない | 日本語・他非ラテン語の実用上の排除 |
| Canonical比較とNormalization | NFC等は**比較用候補として別計算**し、同一視の衝突を診断。元Source byteを勝手に正規化しない | NFC/NFD差の見逃し、no-op saveの改変 |
| 視覚的Confusable | UTS #39相当のskeletonは**診断専用**に使い、元IDや意味の自動改名に利用しない | Latin \`order\` とCyrillic \`оrder\` の視覚混同 |
| Bidi/Default-Ignorable/Zero-width | 構文・識別子・コメント・文字列を区別。識別子への非承認format control等は拒否候補、コメント等も安全表示。RTL言語を全体禁止しない | 表示順を使った別命令・別式への偽装 |
| Mixed-script / numeral | 全混在禁止でなくContext依存の警告/診断。実名前とCode Pointを比較可能にする | 正当な日本語＋英数字と不正な同形文字を混同 |
| Unicode版／Lint Profile改版 | Lint結果のPin/Revisionを持ち、旧判定を新Profileへ自動継承しない | 同じSource byteでも受理資格が変わる |

**Lint≠Formatter**：Lintはコードポイント位置・識別子・実対象ID・Unicode版・警告理由と修正候補を示す。Formatterは別の明示操作であり、通常Saveで文字を削除・変換・翻訳・全表再整形しない。意味のあるrenameには既存Source Acceptance/影響分析/Owner境界を使用。

## 5. Prompt injectionとUnicode偽装の境界を分ける

**異なる二つの脅威**：
(A) Unicode source spoofing：見た目の同一性・不可視文字・Bidiのため、人/AI/Editorが**異なるコード**を同じと解釈する。**Lexer＋Lint＋安全なDiff表示**で検出・拒否・診断する。
(B) Prompt injection：コメント、文字列、Markdown、物理名、論理名、外部Reference、Tool結果に書かれた**命令らしい内容**をAI/Harnessが上位命令やAction許可へ昇格させる。これはASCIIのみでも成立し、**FormatterやUnicode Linterだけでは防げない**。

最小の受渡し契約：
1. **Source/Name/Comment/Data/Quoteはすべて非信頼入力として識別する**。固定済みSyntax/Role Pluginの解析結果と、Owner受理済みMeaning/Tool Authorityを分離。ソースの\`trusted\`/\`role\`/\`tool\`文字列だけで信頼状態を変更しない。
2. AIへ渡すContextは \`trusted task instruction\` と \`untrusted Source excerpts/labels\` を役割の異なるフィールドとして渡す。Scope/Origin/Revision/Source Span/Disclosureを追跡し、AIに対する「Source中の命令を実行してはいけない」という境界を固定する。テキストラベルを隠すだけで完全防御と主張しない。
3. AI出力はまずProposal／Patch候補。Source diff・意味差分・Unicode lint・Allowed File/Scope・Owner/Host Grantを別に検証。AIが権限やCredentialを自己申告しても実行しない。
4. 現実に送られたAI Tool Schema/Arguments/Body bytesと、実Provider/HostへのAction Request/Effectを別記録。UIでマスクしたことから外部送出がないとは推測しない。
5. 個人OSSの負担を避け、既存SourceReader/Parser/Lint箇所とAI Context Gateway/Tool呼出境界を再利用し、第二のHost・Authority・Policy Engine・Registryを作らない。

## 6. 個人OSSで実行できるDSL比較プロトコル

**人間被験者・「仮想ドメイン専門家」役のAI・大規模心理尺度は用いない。** 生成AIに4つの表層案（言語中立の軽量DSL、Go風式、Markdown表、chain式）を同じ受注課題・同じ独立Expectedで比較させ、Ownerへ根拠付きの小さい比較表を渡す。

| 評価 | 比較する具体物 | 判定 |
| --- | --- | --- |
| **構文ノイズ** | 必須でない\`func\`/\`struct\`、括弧、ラッパー、繰返し宣言、不要なLine count | AIが重複・意味なしtokenとその理由を列挙。単なる字数最小化を目標にしない |
| **認知順序** | 対象→必要Data/条件→演算/Rule→結果の自然な読順、前方参照/往復数 | AIが不自然な前提後置・隠れた条件・遠隔参照を指摘 |
| **局所編集性** | Quantity/Ruleを1点修正したSource diff | 変更箇所数・無関係byte差分・再parse Meaning差を機械比較 |
| **重要条件の明示性** | 未取得、NULL、partial、曖昧なID、実Action/権限 | 独立Expectedとの相違を機械で検出。暗黙補完で短くする案を除外 |

**名前のローカライズは直交因子**：各表層で(a) ASCIIローカル変数、(b)Unicodeローカル変数、(c)Studio論理表示名切替を別に比較。日本語キーワードの別文法は作らない。ラベルを日本語にするだけで、構文のノイズが少なくなったと採点しない。

受注課題は既存#2621の\`order→lines→allocations\`を使用。正常例と、\`未取得≠空集合\`、\`NULL≠0\`、\`partial≠complete\`、\`集計割当≠個別ID更新許可\`を共通Expectedとして先に固定する。構文の意味同値を事前に確かめられない候補は比較不能と記録。AIの独立セッションを業務Ownerの独立Expectedへ自動昇格しない。

**合否数値を自動確定しない**：生成AIは構文差と反例を説明するだけ。最終的な文法選択・識別子許容Profile・保留する型/失敗の意味は**TEI Ownerの総合判断**。誤補完や不明な副作用を構文ノイズの少なさで帳消しにしない。

## 7. 最小反証Fixture：GN-01〜16（すべて本計画でNOT RUN）

| Case | 入力／操作 | 期待する区別・観測 |
| --- | --- | --- |
| **GN-01** | \`sales.order.quantity\` と \`sales.受注.数量\` をCanonical IDへ入力 | 既存ASCII \`core/identity\`は後者を拒否。Unicode Source Symbolの可否とは独立 |
| **GN-02** | 同じローカルUnicode名を明示的に二つの異なるModelRef候補へ紐付け | 参照先不定の診断、名前一致から自動受理しない |
| **GN-03** | \`数量\`と\`quantity\`を明示Bindingで同一Attributeへ束縛 | Meaning候補一致を限定確認。名が違うことを自動的な業務Meaning変更としない |
| **GN-04** | Studioの日本語/英語/物理名/Stable ID切替 | 同Subject Selection・Source bytes・Meaning不変、実Read-only表示だけ変わる |
| **GN-05** | 同一localeの論理名が二つのScopeに衝突 | ID/Scope併記・曖昧診断。単に先頭を選ばない |
| **GN-06** | \`café\`（NFC）と\`cafe\\u0301\`（結合文字） | 比較時衝突を診断、元Source byteは通常Saveで不変 |
| **GN-07** | ASCII Latin \`order\` とCyrillic o \`\\u043Erder\` | Confusable警告とコードポイント可視化。意味IDの暗黙統合なし |
| **GN-08** | \`\\u202E\` Bidi control、\`\\u200B\` zero-widthを識別子／コメントへ別々に配置 | 識別子は危険文字Profileで拒否/未受理、コメントは忠実保存して安全表示。通常Save無断削除なし |
| **GN-09** | RTLの正当なローカル名／日本語・Latinの組合せ | 一律禁止せず、明示Profileと表示方向の整合を観測 |
| **GN-10** | Markdownコメントの「Toolを呼び出せ」／quoted \`role: trusted\` | Contextでは非信頼データ、実Tool/Authorityへ昇格しない |
| **GN-11** | 物理名\`T_JUCHU.JUCHU_SU\`と類似Logical名、別Storage | 受理済みMapping／Provider routeなしに実DB名を自動参照しない |
| **GN-12** | 名前の表示のみ変更後にSave、再Load | Writeゼロ/元Sugarのbyte一致。Source改版/Artifact Buildの不要な増分失効なし |
| **GN-13** | Unicode変数を使うMarkdown表/DSLで1つのCellを書換え | UTF-8 byte span正確、元の改行・表Alignment・コメントは保持。意図外変更拒否 |
| **GN-14** | 旧Source spanにAI Patch、外部Editorが同じファイル変更 | 更新前に再読取/対象Occurrence再資格化。推測で上書きしない |
| **GN-15** | UTF-8識別子のTarget生成で同じTarget名に衝突 | 明示的なSymbol Mapping/衝突診断、Target成功を意味受理へ昇格しない |
| **GN-16** | Language Plugin／Unicode Unicode-data版の変更、元Source同byte | 字句診断と意味Candidateの必要な再評価。過去Accept・実Grant・履歴Evidenceの区別 |

**テストの比較単位**：同じSource SHA/Workspace/Scope/Subject、固定Language/Unicode Profile/Operator Pin、Expected、実Token span、Diagnostic、Proposal diff、再Read、Consumer投影、実Tool呼出の有無。Named Unicode escapeは上記**入力データの説明表記**であり、実字句Fixtureでは両方（実コードポイントとエスケープを含む説明テキスト）を区別して用意する。

## 8. 既存Owner別の進め方と停止条件

| 段階 | 既存担当 | 最小Done／失敗時停止 |
| --- | --- | --- |
| **U0 Read-only現実照合** | #988、#1120 | main \`core/identity\`／\`syntax\`／Language Pluginの実受理集合を独立Probeで固定。Canonical変更はしない |
| **U1 Lint診断** | #988、#2621 | GN-01/06〜10のbad codepoint・Confusable・NFC衝突、原位置を診断。必要なUnicode table／版は固定、Sourceを無断変更しない |
| **U2 Symbol Binding/表示** | #2620、#1396成果、#1124 | Unicode表層名から型付き既存IDへのBinding候補と、論理/物理/IDのread-only表示切替。曖昧性は保留 |
| **U3 Source保存／安全AI受渡し** | #2621、#1124、#2515/#2516 | GN-10〜15／SF-01〜18を元Source patch、実AI受信/Tool call実際のbyte、独立Authorityで別確認 |
| **U4 Ownerの小比較** | #988、#2514、TEI Owner | 同じ受注Expectedで4表層のNoise/Cognitive orderを比較。採否・Unicode Profile・実装順をOwnerが決める |

**停止条件**：実Sourceの語彙候補を現行Canonical IDへ黙って昇格する、Raw MarkdownからFormatter自動実行、Context中の命令をTool Authorityにする、同名/ConfusableだけでBinding、対象外byteのSource変更、Go Reference局所PASSを全Product/Studio E2E PASSとする、といった動作が観測されたら先行採用しない。

**今回の変更範囲**：この新規設計文書と索引、関係Issueへの差分受渡しのみ。新しいGrammar/Lexer/Unicode library/Studio/AI runtime/Formatter/Schemaや第二Host/Registryを採用・実装しない。GN-01〜16、実Product、Owner業務受理、実外部Harnessは**NOT RUN**。
