# Everything is Data：Markdown原表記・書式保持とSource-aware保存契約（2026-11-03）

> **文書の役割**：TEI/SHO間で共有可能な設計判断と受入提案。**状態：実装前設計／保存エンジン・API未採用**。最終的な採否・Source Acceptanceは各Owner。親[#2619](https://github.com/hat47x/tei-platform/issues/2619)、Authoring [#2621](https://github.com/hat47x/tei-platform/issues/2621)、構造写像[#2497](https://github.com/hat47x/tei-platform/issues/2497)、Studio[#1124](https://github.com/hat47x/tei-platform/issues/1124)、Language/Operator[#988](https://github.com/hat47x/tei-platform/issues/988)、SHO[#4](https://github.com/hat47x/sho-composer/issues/4)。

## 1. 利用者が要求する不変条件

**Markdownへの保存では、展開後の標準データ構造をMarkdown形式のJSON/YAML/展開テーブルとして書き出さず、利用者が編集した元のシンタックスシュガーを原則維持する。** 原文中の表、式、Graphviz/DOTフェンス、関数／チェーン／Goa・MukuroL風の宣言DSLは、**元の表記法・引用・コメント・配置・改行・桁揃えを意図せず変更しない**。内部の標準データへのdesugarは解釈・検証・生成用の投影であって、通常の「保存」のための再直列化命令ではない。

この不変条件は、次の三つを分離して初めて成立する。

| レーン | 主に保存するもの | 更新の責務 |
| --- | --- | --- |
| **Authoring Source** | 元Markdownの実byte、選択したsyntax sugar、文書構成・trivia・明示Style | 利用者の原文編集、または対象を一意に特定した最小Source Patch。元文書表記の正本 |
| **Plain Data / Accepted Meaning** | 型付き標準データ候補、Role/Operator/Ref、Domainで正式受理された意味、SourceへのTrace | 既存Language Plugin／Semantic／Acceptanceで解決・再資格化。**Markdown Writerの二重意味正本ではない** |
| **Consumer Projection** | TEI Studioの表/Graph/式/Instance、SHO文書、外部AI表示、生成Target | 表示や実行用の派生物。勝手に原Markdownを書き換えない |

**例**（表記・RoleのSchemaは採用前の例）：

元のMarkdown：

```markdown
<!-- 計算方法に関する著者コメント -->
| 数量 | 単価 | 金額 |
| :--- | ---: | ---: |
|  2   | 100  | 200  |

```

TEI内部で`quantity`、`unitPrice`、`amount`と型付きRecord/Functionへ展開されても、**無変更の保存結果は元Markdownのbyte列と完全一致**する。Quantity等をStudioで編集する場合、変更対象のCell/ExpressionのSource部分だけへPatchし、原表の区切り、コメント、空行、列揃え、前後の文章、末尾改行は、利用者が別途書式変更を要求しない限り維持する。

> 「意味同値」と「書式同値」と「実byte一致」を混同しない。本文を変更した場合は全文byte一致は当然成立しないが、**無関係部分のbyte不変**を要求する。表幅の増大等で、局所セル以外の書式を変える必要があると判断した場合は、**必要な変更範囲の差分をPreviewし、明示的に許容された範囲だけをPatch**する。黙って表全体を再formatしない。

## 2. Source-aware変換の基本契約

```text
 Markdown Source (bytes + style + provenance)
     │ [既存Workspace Source Reader／Syntax Plugin]
     ├──────────────> Source occurrence / spans / token+trivia / style hints
     └──desugar─────> Plain Standard Data ──Role/Operator──> Meaning
                               │                            │
                Studio / AI / CUI / SHO の構造Projection     │
                               │                            │
                           編集Intent ──typed delta ───────┘
                               │
          Source-aware Patch Planner (対象Occurrenceとbase byte束縛)
                               │
           Candidate Markdown patch + semantic/byte diff preview
                               │
         再読取／reparse／Meaning差分・Scope・Authority照合
                               │
                   既存Acceptance／Source Writerの確定経路
                               │
              元構文を保持した新Markdown Source (bytes)
```

- `parse/desugar` と `format/pretty-print` は別Operation。**一般の保存ではformatterを実行しない**。Format Documentは利用者が明示実行する独立機能で、通常SaveやAI修正とは分離。
- Parserが捨てがちな情報（space/tab、CRLF/LF混在、BOM、コードフェンス種別・言語名、表pipeとalignment、inline code、エスケープ、コメント、空行、引用、attribute順、DOT layout）は、**元Source byteとOccurrence spanで保持**する。CST/lossless tokenは実装候補であり、すべてをCanonical/Role ASTへ追加しない。
- 構造化編集は原文からの**局所Patch**。保存時に元MarkdownをSemantic Dataから再生する行為を禁止。CSTが必要な箇所は限定導入とし、行全体やAST全体の無条件再直列化をしない。
- 不明なDSL・Graphviz要素やPlugin未対応の構文は**opaque Sourceとして保持**し、読取表示・Raw編集は許容しても、意味を補完した書戻しは止める。説明中・HTMLコメント中・inline code・code fence内の文字列を、実行可能なRole／外部参照として勝手に認定しない（MMI [#658](https://github.com/hat47x/markdown-matrix-injector/issues/658)／[Draft #755](https://github.com/hat47x/markdown-matrix-injector/pull/755)の負例と整合）。
- 文書上の`role`、`tool`、`trusted`宣言は、Authoritative Role/Capability/Host Grantの根拠ではない。

## 3. Source Patchに必要な最小の情報（**新Core固定Schemaではない**）

| 情報 | 必要な理由 |
| --- | --- |
| Stable Subject/Role と**Source occurrence** | 同じ関数や同じ表が二度登場しても、編集対象の出現箇所を一意に特定。例示・本定義・引用を区別 |
| Source path／Load identity、byte digest、Source revision | 古い表示／外部Editor／Git変更／別Windowとの競合。**同内容別path**やファイル移動で古いLocatorを無条件再利用しない |
| UTF-8 byte-span（範囲）＋受理したSyntax Plugin ID/version | Markdownの絵文字・日本語でUTF-16 positionとbyte offsetを混同せず、表層が何かを説明 |
| Trivia/Style（元Sourceへの参照） | 表pipes/space/alignment、引用、BOM、改行、インデント、DOT内レイアウト等を失わない |
| Semantics before／after、Edited fields、依存Role/Operator pins | 指示した意味変更だけに対応し、別のFunction/RuleのMeaningを壊さない |
| Writable Scope／編集の可逆性（構造写像#2497） | 単一Cellや一意Expressionへの書戻しか、集約/図の非可逆な編集かを識別 |
| 現時点のSource Writer/Hostの権限、影響範囲 | Previewが成立してもファイル更新権限・正式Source Acceptanceには昇格しない |

**Patch採用条件**：保存前に同じ固定RootでSourceを再読取し、対象のbase byte/span/occurrence/revisionを再照合する。別編集によりSource byteが変わった場合、対象領域の再アンカーと意図した差分を再評価できない限り、**conflictとして停止**。位置だけを更新して無条件に上書きしない。既存[#1017](https://github.com/hat47x/tei-platform/issues/1017)のWorkspace境界、[#1529](https://github.com/hat47x/tei-platform/issues/1529)の外部編集競合Reference、[#1158](https://github.com/hat47x/tei-platform/issues/1158)の共通Operationを再利用する。新しいSource Reader、Writer、Authorityや分散Txを構築しない。

## 4. 保存時の振る舞いの確定候補

| 操作 | Source保存の既定 |
| --- | --- |
| **No-op Save** | **書込み自体を行わない**。byte digest、ファイル内容、改行、書式、可能な範囲でmtimeも不変。構造を全展開して書き直さない |
| **Raw Markdown編集** | 元のSource形式で保存。Parserは結果を再解釈して診断するが、意図的な本文の書式変更を勝手に元へ戻さない |
| **Studio表の既存Cellの値変更** | 出現箇所が一意ならCell内の最小byte範囲を差替え、残りの部分は実byte不変。Markdown escapeと`|`は安全な構文表記を選び、行・列全体のformatを既定にしない |
| **構造化DSLの式／引数変更** | 元の「式」「関数記法」「チェーン」「Goa/MukuroL風階層表記」を維持し、その構文の既存Styleに従う局所PatchをPreview |
| **Graphviz/DOTフェンスの内容変更** | 元DOTのNode/Edge識別と文脈付き変換#2497が一意に成立した要素のみPatch。Graphvizのlayout/rank/colorやコメントは保全。図示用で意味不足ならRawまたはread-only |
| **新しい定義の追加** | Sourceに既存の表層プロファイル/近傍Styleがあれば同じSyntaxで**新しいSugar**を生成。既存定義全体を書き換えない。新定義の書式選択は明示Profileに従う |
| **大量の構造再配置／表幅再調整** | 保存とは別の**Review可能なFormat/Refactor操作**。変換が必ずlosslessと証明できなければPreview/拒否・元Source保持 |
| **構文/Role不明、Source競合、対応先複数** | Fail-closed。未受理候補・差分を保持し、無断Source Writeや別の対象へ誤反映しない |
| **SHO本文の編集** | SHO文書のAuthoring Sourceへ保存。TEI Accepted Meaningへは自動逆書戻ししない。外部リンク・Source Referenceの再解決と文書Review資格はSHO側で独立確認 |

追加の自動整形を行うEditor ExtensionやMarkdown formatterが介在する場合は、TEIのSave経路で暗黙に起動させない。外部のWriterが変更した場合は再読取後に**実byteの差分・意図範囲**を検査し、Source改版として扱う。生成物（顧客向けMarkdown仕様書等）の書式テンプレートとは、**利用者が編集する原Markdownの保存**を区別する。

## 5. 二つの独立した保存法則

- **SF-L1 No-op byte law**：`save(S, no_change) = S`（実byte一致、書込みゼロ）。`parse→desugar→save`で原Markdownを再直列化してはいけない。
- **SF-L2 Framed local edit law**：対象出現を`S = prefix · old_fragment · suffix`と一意に識別したとき、成功した局所Patchは`S' = prefix · new_fragment · suffix`。**prefix/suffixはbyte一致**。追加の整形が必要ならScopeを明示的に拡張してPreview・再受理する。
- **SF-L3 Semantic intent law**：Patch後の実Sourceを再解析し、依存Role/Operator／Source出現／Meaning候補の差分が**意図された変更集合の範囲内**にあることを検査。期待以外が変化したらrejectまたはReviewに戻す。既存のOwner意味受理と異なる自動acceptはしない。
- **SF-L4 Stale base law**：別Process/WindowのSource/Path/版が変わったら旧spanを使用した無条件保存は拒否。外部変更の自動3-way mergeは**対象を一意に特定し、差分が競合しないことを説明できる場合だけ**候補。
- **SF-L5 Opaque preservation**：解釈対象外や未知のSyntaxは、隣接セルやRoleを変更してもソース上保持。自動マイグレーションやFormatによる削除はしない。
- **SF-L6 Authoring/meaning separation**：Authoring文書の書式変更により意味が不変でもSource occurrence/revisionとReview資格は別に更新/再検証。意味結果の一致を元書式の復元根拠にしない。

SF-L1とSF-L2は**実byteの法則**、SF-L3とSF-L6は意味・編集資格、SF-L4は競合／Writer、SF-L5は未知Source保持の法則。構造写像のGetPut/PutGetは、対応可能な領域に限定してこれらと照合する。

## 6. 初回の限定受入マトリクス（SF-01〜18、**未実施**）

| Case | 入力/操作 | 合格条件 |
| --- | --- | --- |
| **SF-01** | Markdown表・コードフェンス・Graphviz・式の無変更保存 | 原文の全byte一致、**No write** |
| **SF-02** | 表Cellの1値を変更（元列幅内） | 対象Cellのbyte差分だけ。pipe/space/alignment/他行は不変 |
| **SF-03** | UTF-8日本語・emojiの入った表で値編集 | UTF-8 byte spanが正しく、異なる列や前後の文章に破壊なし |
| **SF-04** | CRLF/LF混在、末尾改行なし、BOMありを保存 | 非対象改行/BOM/EOFはbyte不変 |
| **SF-05** | Markdown表の`|`エスケープ、inline code、引用内の表 | 誤った列分割や実Role昇格なし。表記保持可能領域のみ局所Patch |
| **SF-06** | HTMLコメント、フェンス内の例示、実定義が共存 | コメント/例示を勝手に定義扱い・書換えしない |
| **SF-07** | 同一内容の表が2箇所にある | Source occurrenceが曖昧なら保存拒否、対象を誤選択しない |
| **SF-08** | インデントDSL/メソッドチェーンの引数1項変更 | 元のSugarを保持。正規化JSONへ置換・式全体のformatter起動なし |
| **SF-09** | DOT node属性1件変更 | コメント・layout/edge orderingと他nodeを保持。文脈Role未解決なら編集拒否 |
| **SF-10** | 表Cell値が列幅を超える | 周辺全文の自動再整形なし。局所的な幅変化は差分Preview、必要な他列調整は明示承認 |
| **SF-11** | Role Plugin変更、Schema拡張、未知コードフェンス | opaque Sourceは失われず、Raw閲覧可、意味編集は候補に留める |
| **SF-12** | Studio編集後、外部Editorが同じファイルを変更 | Source digest/span/revision再確認でconflict、無断上書きなし |
| **SF-13** | 内容byte不変のファイル移動/別Path | Semantic ID同じでもSource locatorの再解決・Reviewの現在性は別 |
| **SF-14** | Raw Markdownを整形してから再解析 | Authoringで意図した書式変更は保持。Meaning変化なければコード不要再生成候補、Source/SHOは再資格化 |
| **SF-15** | Studio表編集のあと、再開・再保存を繰返す | 意図しない整形の蓄積がない。No-opは毎回書込みゼロ |
| **SF-16** | SHO本文で引用・脚注・MMI埋込参照を含む保存→再開 | TEI/SHO自体の元文書byte・Occurrence保持とSource再解決を別検証。TEI意味への自動逆書戻しなし |
| **SF-17** | AIがMarkdown全体を整形し直すPatchを提案 | Semantic同値でも無関係差分を拒否/Reviewへ戻す。AIの自動Saveを無条件採用しない |
| **SF-18** | Semantic Dataの非可逆集約ViewをStudioで編集 | 元Sourceを推測で更新せず、明示Actionかread-onlyとして提示 |

**試験証拠**：固定原文Fixtureの実byte/SHA、Operator/Role/Parser版、対象Occurrence/changed Span、原Source→候補Patchの最小diff、再解析前後のMeaning差分、実Writeの有無、外部競合時の元Source不変、消費側の再読取byteを保存する。**見た目のスクリーンショットのみでは原書式不変の証拠にならない。**

## 7. 最小実装の進め方とOwnership

| 段階 | 実装・評価 | 終了条件 |
| --- | --- | --- |
| **F0 Contract/Reference** | 既存Workspace SourceReader・Source Index・三視点Modeler編集・Shared Acceptanceと、有限Source Edit Fixtureの対応を調べる | 「現行SourcePatchで既にできること」「新たな必要Patch」「未解決」を分ける |
| **F1 No-op/Cell Patch** | 実Source Reader/Writerを通す`no-change`と「同一表内Cell1個」の契約を試す | SF-01〜04/07/12の実byte不変、原文保持、競合拒否。新Parser/Writerを先に増やさない |
| **F2 異構文と複合Markdown** | インデントDSL/チェーン/DOT/HTMLコメント/コードフェンスのSource occurrenceへ拡張 | SF-05/06/08/09/11/17の正負ケース、Role/Operator版固定 |
| **F3 Studio/SHOの二Consumer** | Studio表/Graph編集と、SHO文書・MMI出典参照の保存再開を別々に検証 | SF-10/13〜16/18と実Target/Consumer開示資格を分離 |
| **F4 AI/高速Build** | AIの局所Patch成功率、意図外byte差分、診断・修正回数、token、変更影響・再生成計測 | Source form preservationを失わずにMeaning変更・Source Fidelity・Build再利用を別合格 |

**Owner境界**：[#2621](https://github.com/hat47x/tei-platform/issues/2621)がAuthoring Sugar/Source preservation、[#1124](https://github.com/hat47x/tei-platform/issues/1124)がStudio edit/Source PatchとUI版、[#2497](https://github.com/hat47x/tei-platform/issues/2497)が構造変換と逆写像、[#988](https://github.com/hat47x/tei-platform/issues/988)がSyntax Plugin/Operator、[#2600](https://github.com/hat47x/tei-platform/issues/2600)が差分再生成、[#2514](https://github.com/hat47x/tei-platform/issues/2514)が独立検証基準、SHO [#4](https://github.com/hat47x/sho-composer/issues/4)は保存再開・文書正本、MMI [#658](https://github.com/hat47x/markdown-matrix-injector/issues/658)はMarkdown例示と参照の識別。

本設計は既存の[Everything is Data横断計画](../workstreams/EVERYTHING_DATA_CROSSLANE_HANDOFF_2026-11.md)のAuthoring保存契約の詳細である。追加するのは**Sourceを保った保存の受入要件**であり、Source→Standard Data→Meaningへの変換やGraphviz文脈写像の価値を否定しない。新しい共通Host、第二Canonical、万能lossless AST、Sourceに自動挿入する編集用メタタグは必須条件としない。

## 8. この設計が主張しないこと

- 現時点でTEI StudioのMarkdown表構造編集・実Writer/永続保存・SHO Composer/Multi-process競合が実装・実測されたという主張。
- すべてのMarkdown方言と任意のSyntax Pluginについて、一般的にCST完全可逆が成立するという主張。
- 同じSemantic Dataが、情報の欠けた元Markdownの装飾やコメントを自動復元できるという主張。
- GitHub CI/合成fixturesのPASSから、Owner意味受理、実外部Host権限、納品固定版の資格を与えること。
