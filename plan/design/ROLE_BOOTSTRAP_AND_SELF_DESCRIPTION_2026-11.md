# TEI Everything is Data — 最小Bootstrap・Role解釈循環・利用資格の接続計画（2026-11-02）

**状態：横断設計候補／実TEI Core・DSL・Role Resolverへの採用前。** 本計画の設計上の中心は、標準データ構造でRole/Operator/構文Plugin自身を記述可能にしながら、**その自己記述を初めて読むための有限な解釈の根**を明示し、循環と未決、実行資格の区別を機械検査できるようにすること。

- **横断Owner**：[Everything is Data #2619](https://github.com/hat47x/tei-platform/issues/2619)、[標準構造・意味Role #2620](https://github.com/hat47x/tei-platform/issues/2620)。
- **実装Owner**：[DSL Language/Operator Plugin #988](https://github.com/hat47x/tei-platform/issues/988)、[Plugin/Service Contract #951](https://github.com/hat47x/tei-platform/issues/951)、既存Semantic Extension Lock／Source/Accepted Meaning。独立のParser Registry・Extension Resolver・実行Hostを新設しない。
- **Consumer**：[Markdown/Source #2621](https://github.com/hat47x/tei-platform/issues/2621)、[Facet/Law #2544](https://github.com/hat47x/tei-platform/issues/2544)、[Product Skill #2605](https://github.com/hat47x/tei-platform/issues/2605)、[将来Skill #2614](https://github.com/hat47x/tei-platform/issues/2614)、[構造保存 #2497](https://github.com/hat47x/tei-platform/issues/2497)、[Studio #1124](https://github.com/hat47x/tei-platform/issues/1124)、[Context #2516](https://github.com/hat47x/tei-platform/issues/2516)。
- **系列の先行計画**：[Draft PR #2610](https://github.com/hat47x/tei-platform/pull/2610)／[#2619 EI-01〜08](https://github.com/hat47x/tei-platform/issues/2619)。本書は**EI-01の上流にある役割記述自体のBootstrapの実証差分**に限る。別の統合受入Gateではない。

## 1. 自己記述可能性と自己実行可能性を分離する

「Everything is Data」は、Role Definition・Rule・Function・Syntax Pluginの説明を同じデータ構造で記述できることを目指す。ただし任意の記述を、それ自身の「これを信頼せよ」指示によって読み込み／実行することはしない。

```text
Layer 0   既存Workspace/Extension Lockに基づく入力・Pinの確認
          + 有限な基本値・Record/Sequence/Refの読取り（既存基盤）
                    ↓
Layer 1   Pinで選ばれたSyntax/Role/OperatorのProvider実装を
          既存Resolverで特定し、要求Shape/依存/適用条件をデータとしてInspect
                    ↓
Layer 2   Source occurrence → desugarされたplain standard data
          → 明示Role binding → typed meaning/validation/qualification
                    ↓
Layer 3   CUI／Studio／Skill／SHO／Targetへの用途別Projection
          + 外部HostのAuthority／実Effectは独立した実行時検証
```

**Layer 0は自己記述データを成立させるための有限な実装上の出発点**。現行の既存Workspace reader、標準基本型、Extension Lock、既存Plugin解決への最小契約として検証し、新しいUniversal Meta-Engineを構築する意味ではない。実際の初期Language/Semantic Setを含む正確な既存経路と、PoC用記号を対応付けてから採用を決める。文字列`role: trusted`や`effect: pure`は自己申告であり、Trusted Provider/Host Grantを生成しない。

## 2. 循環を二種類に分ける

### 2.1 解釈に先立って確定が必要な依存（eager）

例：Role-Aの`required shape`を読むためにRole-Bの**評価完了**が必要であり、同時にBの評価にはAの完了が必要な場合。rootから到達したeager閉路は解釈の根が得られない。

この場合は、既存Plugin ID/versionと依存経路を使って循環を診断し、ロード順・ファイル順・AI推測で一方を選ばない。固定点計算を採用する場合は、その意味・収束条件・失敗時の振る舞いを**該当Operator Owner**の明示契約として別途評価する。初回Referenceではeager閉路を不成立とする。

### 2.2 型・関係の遅延参照（deferred）

例：`Node.children: Sequence<Ref<Node>>`、Tree/Graphの循環辺や、互いを参照する二つのData Model。これは宣言同士の参照であり、直ちにそれらの**値を再帰評価して確定する要求**ではない。

既存Typed Ref/Stable IDへ解決できる遅延参照は、**解釈循環とは別に**成立し得る。データ値を辿る際のtermination、Visited Set/深さ制限、結果の部分取得/未知は利用するOperation・Consumerが定める。「すべての循環参照を拒否する」設計を採用しない。

### 2.3 相互依存の表示

一つのRoleに複数Roleを適用できること、同じ名称で異なるProviderが候補になること、演算`combine`を持つ二つのMonoidを使い分けることを別に扱う。同じ名前・同じCarrierでの自動選択は避け、Use-siteが要求する**Role + Operator ID/version + binding + scope + evidence**で比較する（数学的公理の正本は#2544）。

## 3. 構文糖衣の同一性をBootstrapの証拠へ還元する

- Markdown/DOT/Goa風/MukuroL風/Chain/JSONは各Syntax Pluginが**plain data候補**に変換。Source文書そのものと、変換結果、受理済みMeaningを区別（#2623/#2625）。
- 引用文中の`role`や`op`は**Source occurrence中の値**。Project Profileと明示Role bindingがない限り、実行定義のCandidateとならない（#2621）。
- 同じData byteでもRole/Operator/Syntax PluginのPinが変われば**意味資格の再評価候補**になる。Source byteが変わらないこと、Role名が同じこと、独立Fixtureで同じ値が出たことは、旧Meaning Acceptanceや現在のHost Grantを引き継ぐ根拠にならない。
- 構文／RoleからSkillやTool説明を生成できる場合も、そのSkillが指す**実CLI/API/MCP入口・版**を別に検査（#2605）。未導入Providerの`effect: read-only`宣言で、秘密Sourceの外部開示や自動実行資格を付与しない。

## 4. 独立Reference：RB-01〜RB-19（有界合成の実行済み、実TEI試験はNOT RUN）

初期は`plain data候補 + pinned descriptor候補 + eager/deferred依存`を有限な入力として**read-onlyで**比較する。独立Fixtureの出力Schemaを正式なCore Meta Schemaへ昇格させない。

| Case | 入力 | 期待する判別 |
| --- | --- | --- |
| **RB-01** 最小起動 | Role providerなしでRecord/Sequence/Refを読む | データとしてinspect可能、`Function`のMeaning qualifiedとはしない |
| **RB-02** 決定性 | 同じPin/Source/依存を別列挙順で読む | 同じ候補／診断、file load orderによる採用差なし |
| **RB-03** 未解決Provider | 指定Role Provider ID/versionが存在しない | missing/unsupportedを返し自動導入・自動採用しない |
| **RB-04** 同名異ID | Role名同一でProvider ID・版が異なる | explicit binding不足ならambiguous、first-match採用しない |
| **RB-05** 解釈eager-cycle | A→B、B→Aが双方評価完了を必要とする | 読み込み完了を捏造せず閉路と依存経路を報告 |
| **RB-06** 合法なrecursive Ref | Node.children→Ref<Node>を遅延解決 | eager評価閉路扱いせず、利用時の走査契約を分離 |
| **RB-07** 複数Role | 同一データに独立に適合するModel Roleと説明Role | 各Roleと根拠を保持、片方の採用から他方を実行しない |
| **RB-08** 引用と未信頼文書 | quote中の`trusted-script`／`run-as-admin` | Role実装の信頼／Tool命令／Host権限を生成しない |
| **RB-09** Plugin改版 | Source一致、Role/Operator v1→v2 | 解釈上の資格を失効・再照合対象にし、旧受理を復活させない |
| **RB-10** PureとEffect | `pure`宣言が外部writeを含むProvider | 宣言からpure保証せず、Provider契約と実Targetを検証対象へ |
| **RB-11** Source occurrence | 同じ構造の定義表・Expected・引用、場所は別 | 意味同値候補と編集元Occurrence/Source locatorを分離 |
| **RB-12** Skill/Harness | Role Descriptorから未知コマンドを生成 | 名称解決だけでToolを実在・実行許可・配布済みと認定しない |

### 4.1 依存先もPinする追加負例と現在の実測

最初の12ケースに、初期bootstrapの**根・未受理Role・依存先ピンの扱い**を追加した。合成JS Referenceの識別子や`source.active`は外部で既に分類済みとした**Fixtureの入力**であり、TEIがそのフィールドを見てSourceの信頼性や役割採用を判断する仕様ではない。`requested=[]`でもDataはread-only inspectでき、Roleが解決されていないことを示す。

| Case | 判別すること |
| --- | --- |
| **RB-13** 指定Roleへのpinがない | そのRoleを採用せず`missing-or-stale-pin` |
| **RB-14** eager依存とdeferred参照が混在 | eager依存先は評価し、遅延Refの循環のみで不成立としない |
| **RB-15** Role required shapeを満たさない | 本文値の書換えで補わず未解決 |
| **RB-16** 参照Roleのみpinされ、eager依存先が未pin | **依存先まで版照合が必要**。暗黙に評価可能とみなさない |
| **RB-17** eager依存先も正しくpin | 当該限定shapeではRole候補として検査可能 |
| **RB-18** 同じProvider IDの重複 | 同名・同ID重複は不成立、先勝ち/後勝ちをしない |
| **RB-19** 存在しないeager依存先 | rootが有効でも依存先がなければ未解決 |

**実行済みの限定証拠**：`plan/fixtures/role-bootstrap/check_bootstrap.js`の独立合成Referenceを、コード実行環境のV8 JavaScript isolateで**19/19 PASS**。GitHubへ保存し、保存版を再取得して再実行・SHA照合する。**実Nodeテスト、GitHub CI、TEI Language Plugin/Operator/Extension Lockの実結合はこの試行では未実行**。そもそもFixtureの`registry`は、実TEIのPlugin resolverを模擬している小さな比較器であり、新Resolverのプロダクション実装を提案しない。

この有界Referenceが実証できるのは、指定したデータ・ピン・eager/deferred有界例を区別する**合成アルゴリズムの判別**だけ。Roleの意味付与、可逆Source操作、循環した実行型の一般的許容性、正式Trusted Provider契約、Runtime/Hostの権限、Owner意味受理は証明していない。

## 5. 最短の既存TEIへの接続

**B0**：本Referenceで、必要情報が基本型/Ref/Pin/Provider descriptor/Role候補のどれに相当するかを資料と現行Sourceから明示する。Eager/Deferred依存のモデル化を決定前のpressureとして残す。

**B1**：#988のLanguage/Semantic Extension Lockと既存read-only inspect/validationへ、実在するPin／Provider ID／Role/Operator依存を渡し、RB-02/03/04/05の判別ができるかを確認する。新Provider ResolverやCore演算の再実装はしない。

**B2**：#2625のSyntax正規化fixtureと#2624のFunction別Mappingを**同じ受理Source・同じ意味ID／同じPlugin版**へ繋ぐには別の契約が必要。#2625は`sales.confirmed_total`、#2624は`Order→Line→Note`と現時点で異なる合成Subjectを扱う。これらの局所PASSを単一縦断PASSと呼ばない。[#2586 HJ](https://github.com/hat47x/tei-platform/issues/2586)／[#2619 EI](https://github.com/hat47x/tei-platform/issues/2619)の同一Fixtureに各Ownerが実対応のある受渡しを提供して初めて意味・Provider/Consumer接合を判定する。

**B3**：#2617のScript-first標準UI、#2605のProduct Skill、#2621のMarkdown/SHO、#2622の多Provider、#2516のConsumer Contextへ同じ確定できたReferenceを供給し、`parsed/role candidate/meaning qualified/tool invoked/actual effect/disclosure`の主張を分ける。

### B4 — 変更後の選択的再資格化（歴史・現在・将来を分ける）

複数AI・複数Consumerが同じ役割記述から多種類の結果を導出する場合、以前に観測した「当時の成功」という**歴史的事実**と、いまその結果を再利用してよいかという**現在の資格**を区別する。上記RB-09（Source同byte／Role Plugin版のみ変更）はその最小反例である。

単一のRepo全体Revision Counterや新しい共通Evidence DBを作る代わりに、**Consumer Claimの用途に必要な、既存Source/Plugin/Role/Operator/Function Mapping/Provider/Artifact/Hostの依存参照**から再確認対象を絞る案を比較する。必要な依存が欠ける場合は「影響しない」と断定せず、当該Claimを未確定にする。**見えない依存（where条件／Sort／Rule／Scope／開示条件等）を出力Fieldだけから除外しない**。

| 変更の例 | 既存Ownerで選択的に見直す事項 | 再利用上の注意 |
| --- | --- | --- |
| Markdown改行・Graphviz layout | Source occurrence／Authoring fidelity。意味解釈が同一と独立確認できる場合に限りMeaningを保持 | Source byteの変化を全Artifact意味変更と自動同一視しない |
| Syntax Pluginのv1→v2 | 当該Source occurrenceの正規化結果、Role Binding、下流Meaning | Source同byteでも旧accepted meaningが新版へ自動移行するとは推測しない |
| Role/Operator依存Bだけ改版 | Eager依存を持つ利用箇所のRole資格、利用する演算の型・Law/Effect、関連Fixture・Generated artifact | 影響していないClaimは明示根拠があれば保持可能、依存不明なら保留 |
| Function Mapping／Rule／Provider path変更 | 宣言したRead footprintと実Provider Read Trace、Operation入力/Storage、Consumerの必要Artifact集合 | Operation ID/Basis不変でもInputs/Storageが変わる反例 #2632、生成一式の相対充足 #2630 |
| Host Grant・Disclosure資格変更 | 現在のPrincipal/目的/開示先/時点で実利用前に再照合 | 歴史的なRole/Resultは残せても現在の実行や送出は別資格 |

参照：#2505版間意味対応、#2600増分ビルド、#2630 Consumer別ArtifactSet、#2516用途相対Claim、#1174実Grant。**再資格化は再生成と同義ではない**。同じArtifactを再利用できても、新しいConsumerが要求する範囲・現在の開示資格は再確認が必要になり得る。逆にMaterialized output bytesが同じでも、Rule/Provider/Targetの依存が変われば、元の十分性が維持された保証にはならない。

**選択的再確認ケース（提案／未実行）**：RI-01 layout-only変更、RI-02 Eager Role-B改版で一方のFunctionだけ影響、RI-03 意味に影響するWHERE/Sort依存が出力列に現れない、RI-04 過去Source/Artifact同一のまま現Host Disclosureが撤回。まず既存#2600/#2516/#2630の試験と重複のない入力差を比較し、必要なら#2586／#2619の同一Runへ追加する。適用根拠が取れないものは未確定として報告する。

**効果測定**：新Core特別型数／Roleの重複宣言数／未解決Roleの診断品質／Plugin差替えでの予期しないMeaning変化／標準UI専用コード増分／外部Skillの架空Tool呼出／同一入力での不同結果。単なる`PASS`件数、Role定義数、ソース文字数を製品品質とみなさない。

**責務と優先順位**：#2620がRole自己記述・Bootstrap判断、#988がLanguage/Operator/Pinの実装、#2544がFacetの法則、#2621がSource/引用/テスト、#2586が同一実Fixtureの縦断、#1119が既存P0/G0〜G4の優先を管理。P0-a実PG Receipt/Lookup、P0-b二Function Owner-write、S0 H2/J5とS3意味受理は維持。二Harness配布・OS/Provider/GUI実証はOwner別。

**証拠の状態**：この計画および対応する独立合成Referenceは、TEI実Parser/Role Provider/Studio/Host/DB/Owner受理の成立を意味しない。正式Grammar/初期bootstrap固定仕様と`std.*` Role名は、本実証で既存契約の不足を観測した後にOwner判断する。
