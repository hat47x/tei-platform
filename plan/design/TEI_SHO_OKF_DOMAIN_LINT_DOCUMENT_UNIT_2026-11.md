# TEI/SHOのOKF知識バンドルとドメインLint：段階導入の検証計画

**位置づけ**：複数Issueにまたがる設計・実証計画。**提案／評価待ち（未採用）**。文書の追加はOKFの正式採用、製品機能の完成、Ownerの意味受理を表さない。

**担当と参照**：[SHO #18](https://github.com/hat47x/sho-composer/issues/18)（OKF文書束・Writerとの接続）、[SHO #4](https://github.com/hat47x/sho-composer/issues/4)（原文/Occurrence/保存再開）、[SHO #7](https://github.com/hat47x/sho-composer/issues/7)・[#16](https://github.com/hat47x/sho-composer/issues/16)（Research Index / G1b Reader）、[SHO #6](https://github.com/hat47x/sho-composer/issues/6)（TEI Projection）、[TEI #2619](https://github.com/hat47x/tei-platform/issues/2619)（Everything is Data）、[#2495](https://github.com/hat47x/tei-platform/issues/2495)（値域・多重facet）、[#2497](https://github.com/hat47x/tei-platform/issues/2497)（投影と逆写像）、[#988](https://github.com/hat47x/tei-platform/issues/988)（Language/Operator Plugin）、[#1251](https://github.com/hat47x/tei-platform/issues/1251)・[#1016](https://github.com/hat47x/tei-platform/issues/1016)（Plugin配置と選択）、[#2516](https://github.com/hat47x/tei-platform/issues/2516)（Consumer投影・開示）、[#2514](https://github.com/hat47x/tei-platform/issues/2514)（独立Expected）。

## 1. 目的と結論（2026-11-06）

Open Knowledge Format（OKF）v0.2を、TEI/SHOの**Markdownの知識整理・交換形式**として任意に利用する。既存の設計文書やSourceを一括変換せず、小さなread-onlyバンドルで有効性を測り、必要であれば**選択式のOKF Plugin**として拡張する。

追加で評価する仮説は二つある。

1. **OKFの「1 Concept = 1 Markdownファイル」を、知識管理の標準的な候補単位とする**。必要な範囲ではSHO Documentの管理・レビュー・差分・リンク・再利用単位と一致させる。ただし文書内のClaim・Block・Inline Occurrence・固定納品Versionは別の識別単位を持ち得る。
2. **OKFの開いた型と任意項目を損なわず、TEIのValue Domain/Role/Sourceから文書用Lint・入力補助を派生する**。検査結果はOKF形式適合性、ドメイン整合、Source/Review現在性、Owner受理、実行Authorityを分離する。

[正式仕様: GoogleCloudPlatform/open-knowledge-format SPEC.md](https://github.com/GoogleCloudPlatform/open-knowledge-format/blob/main/SPEC.md)（v0.2、2026-11-06参照）。仕様の更新・互換性は採用時にexact commitを記録する。Google Cloudの[2026-07-24の仕様紹介](https://cloud.google.com/blog/products/data-analytics/okf-v0-2-adds-trust-signals/)も参考。OKFはスキーマレジストリ/中央認可/特定ランタイムを要求しない。本計画の追加Lintを「OKF本体の必須仕様」と誤表記しない。

## 2. 正本性の境界

| 対象 | 既存の正本・責任 | OKF / Pluginが扱う内容 | 推論・昇格しない内容 |
| --- | --- | --- | --- |
| TEIのData/Function/Business/Rule | Source、受理済み意味、Decision | 説明Concept、型・値域の読取り専用参照、診断 | YAML `type`/文書の`verified`からAccepted Meaning |
| SHO Document/Section/Occurrence/Revision | 著者のMarkdown原byte、構成・引用用途、レビュー・固定版 | 1概念1文書を候補にした構成、索引、補完、文書プレビュー | file path/Concept IDから引用Occurrenceやレビュー版の同一性 |
| MMIのSource/Provenance | Intended/Resolved/Actual byte/Location/Candidate/Ledger | `sources`と引用脚注・外部証拠のリンク | 同じhashの別Locationや解決済みリンクから出典の正当性 |
| Artifacts/Consumer/Host | 生成Rule、Build、Artifact、Provider、Host Grant、実Effect | 設計成果物を指す概念と説明、必要なEvidence参照 | Lint PASSやAttested Computation文面から実行成功・権限 |
| 文書交換用Bundle | 選択された原文書・Sourceの版とBundle生成条件 | OKF形式の派生物、索引・ナビゲーション | 第二の意味正本、中央Registry、第二Provider/Resolver |

**変更方向**：TEI Accepted Meaning→提案されるOKF投影→SHOの人間可読な説明・編集→人が必要に応じてレビュー。SHOの文書編集やOKFの属性を書き換えただけでTEIの意味を更新しない。逆方向の変更は、既存TEI候補・独立Validation・Owner受理・権限経路へ明示的に戻す。

## 3. OKF本体とTEI/SHO品質プロフィールを分離する

OKF v0.2の**形式適合**は、非予約`.md`概念文書でYAML frontmatterが解析でき、`type`が非空であること、存在する予約`index.md`/`log.md`が規定の形を取ることを基本とする。未知の`type`、追加キー、任意項目欠落、切れたリンク、索引欠落でBundleを不適合としない。Bundle root `index.md`だけが`okf_version` frontmatterを置ける。予約名`index.md`/`log.md`はConceptには使えない。

「OKF適合」と別に、TEI/SHOの**追加品質プロフィール**を選択可能にする。後者は次の三層で判定する。

| 判定層 | 例 | 提案する出力 |
| --- | --- | --- |
| F0 形式 | YAML壊れ・`type`欠落、予約ファイル構造 | `format-invalid` / `format-valid`（OKF v0.2基準） |
| F1 文書品質 | 壊れたリンク、未索引Concept、`status`省略、`sources[].id`と脚注の対応、引用用途未指定 | 修正候補／警告、未知は`unexamined`。OKF形式適合は維持 |
| F2 ドメイン適合 | 値域違反、単位/NULL/集合粒度、Role必須、関係Cardinality、版ずれ | `valid`/`invalid`/`unknown`/`not-applicable`を証拠範囲付きで表示。Acceptanceとは別 |

OKFの`status`は`draft | stable | deprecated`で、**省略時にstable扱いとなる**。そのためTEI/SHOの未受理・試作文書では`status: draft`の明記を勧めるが、文書の実業務受理状態を`status`へ上書き投影しない。`generated`は生成履歴、`verified`は内容照合の記録、`human-reviewed`はOKFの信頼表示にすぎない。撤回済みReview、対象Revision不一致、期限前のSource変更はこれと独立に診断する。`stale_after`は絶対時刻の判定であり、Source移動・内容改版・Owner撤回を検知する機構ではない。

仕様には`type: Attested Computation`もあるが、`executor`/`attester`のURLやパスは**実行命令や能力許可ではない**。プラグインは既定でread-only。外部I/O・実行・送出には現在HostのCapability/Grant/Disclosureを確認し、記録されたReceipt・独立Readで別評価する。

## 4. 文書管理の粒度：Concept Fileを原則候補にし、必要な例外を残す

OKFのConcept IDはBundle内の相対パス（`.md`を除いたもの）。**識別子であってTEI Meaning ID/SHO Document IDと同一ではない**。移動・改名・分割・統合はConcept IDを変えるので、古い参照・レビュー・納品を自動引継ぎしない。

| 文書の性格 | 推奨候補 | 理由と例外 |
| --- | --- | --- |
| 単一の安定した概念・業務規則・部品・設計決定 | 原則1 Concept File = 1 SHO Document管理単位 | 独立Version/差分/リンク/Reviewを持ちやすい。見出しの細分化だけで新ファイルにしない |
| 設計判断ADR・Issue・受理済みTEI Meaningの説明 | 正本へリンクするConcept Document、またはread-only投影 | OKFへ正本を複製しない。必要な時点・Source/Revisionを固定 |
| 複数Conceptを横断する設計報告・論文・納品書 | SHOの合成Documentとして保持し、複数Conceptを引用 | 1ファイルへ無理に押込む/全章を自動分割しない。レビューは引用Occurrence/Claim単位でも保持 |
| 数式・計算・表・図の派生 | 必要に応じ独立Concept、ArtifactRef、表示依存を併記 | OKFの粒度だけでは計算/表示/意味の同一性は足りない |
| リポジトリ/Bundleの索引 | `index.md`（任意、手書きまたは派生） | 1概念の代わりに目録を置かず、元Document/旧固定版を上書きしない |

分割の基準は文字数ではなく、**独立して変更・参照・版管理・レビュー・再利用する必要があるか**。一文書に二つの独立した規則を含めると一方の変更で不要な再レビューが生じる一方、細分化し過ぎると全体の認知順・文脈を失う。両失敗を同一読者Taskで比較してから粒度を決める。Bundleは複数Conceptを配布する単位であり、個別Documentや納品Versionの同義語ではない。

## 5. ドメイン値域によるLint・入力支援（提案）

既存TEIの**受理済み、または受理前の候補と明示された**Value Domain/Type/Role/Relation/Rule/Source Revisionをread-onlyで参照し、OKFのProducer自由項目と、明示的に構造化されたMarkdown表・コード片へ適用する。自然文の全断定を自動的に業務意味へ変換しない。

- **入力補完**：対象Conceptの種別と明示Bindingから、許される項目名・値域の例・単位・列挙候補・参照可能な意味ID・説明用テンプレートを提案。表示は候補であり、挿入/変更はユーザー操作とSource版照合後に限る。
- **型・値域Lint**：列挙、整数範囲、Decimal精度・金額通貨、NULLとmissing、日付/有効期間、Relationの多重度・必要入力、Ruleの条件を**必要な入力が完全に確認できたときだけ**検査。関連Dataが未取得/非開示/部分集合なら`unknown`を残す。単なる構文`type`が同じでも業務上の同一性を推論しない。
- **根拠Lint**：`sources[].resource`、脚注`sources[].id`、引用Occurrence、外部Resolved/Actual bytesの確認範囲と検証対象Revisionの対応を可視化。リンクの解決と意味の受理を分ける。
- **編集時助言**：SHO G1bのread-only解析結果、G2のRaw Draft差分適用、G3のCAS/Revision管理を利用。Lintは入力中に非破壊・増分に実施可能な設計とし、**warningだけでWriterの作業保存を禁止しない**。確定投影やOwner受理へ進む場合に、既存Ownerの検査条件を適用。
- **複数プロフィール**：金融・受注・組織・Graph等の値域プロフィールは直交的に選択・合成可能。衝突時はロード順や語の一致で勝手に採用せず、選択Root、意味Revision、Conflict、受理/候補区分を示す。二法人兼務や同Plugin別TenantではPrincipal/Scope/Disclosureを別に照合。ドメイン型が同じでも別Tenantの情報やGrantを混ぜない。

最小表記例（**説明用の仮の追加キーであり、正式OKF標準フィールド・TEIの採用済みSchemaではない**）：

```markdown
---
type: Business Rule
title: 受注確定の金額条件
status: draft
sources:
  - id: model-revision
    resource: ../references/order-model.md
tei_binding:
  subject: sales.order.submit
  meaning_revision: example-r1
---

# 条件

| 項目 | 値 | 単位 |
| --- | --- | --- |
| 注文金額 | 1000.00 | JPY |

この条件は受理前の記述候補である。[^model-revision]

[^model-revision]: 本例の架空の参照モデル
```

この書き方で`tei_binding`が存在するだけでは、実Meaningを発見・受理・閲覧許可したことにはならない。Binding解決不能なら、読み手は原文を表示でき、ドメイン補助だけを`unknown/unavailable`として止める。標準化するフィールド名は二種類以上のReader/Writer/Consumer実試験の後に比較する。

## 6. 選択式OKF Pluginの責務境界

**一つの高凝集Plugin Bundle候補**として配布し、内部の責務は独立したAdapter/Provider群に分ける。最初から新Core/Schema Registry/Host/Parser/Resolverを追加しない。

| 部分 | 提供する能力 | 再利用先/非責務 |
| --- | --- | --- |
| OKF Format Adapter | v0.2前提のfrontmatter/予約名/標準リンク読取、Conformance、read-only Concept/Bundle投影 | SHO Markdown Reader/Source span。未知キー・未知Typeを落とさない |
| Domain Lint Adapter | TEIの契約・値域・Roleの固定Refによる局所Lintと診断 | TEI Meaning・Validation Ownerを参照。新Value Domain正本を持たない |
| Authoring Assistance Adapter | 型・フィールド・例・参照先の補完、編集差分のPreview | SHO Writer/Studioの編集入口を利用。無断正規化・Source書戻しなし |
| Bundle/Index Projection | 必要Conceptの選択、read-only索引生成・Bundle差分・消費入力範囲記録 | 既存SHO/MMIのSource/OccurrenceとTEI Artifact/Evidence参照。全件走査済みを推測しない |
| Optional Publisher/Consumer | 明示された出力範囲を別Format/Consumerへ引渡す | Host Grant・Disclosure・actual sendは既存Owner。既定無効 |

**Plugin選択・配布・実消費を分ける**：available / selected / build-included / invoked / externally-disclosed の各段階を混同しない。Pluginを無効化・削除しても、SHOの通常Markdown読書きとTEIの意味モデルは維持。未選択Plugin追加で既存TargetのBuild/Runtimeを暗黙変更しない。実Binding先の版/契約Pinは#951/#1016/#988等の既存方式を使い、別ResolverやLockを作らない。

初期はread-onlyの局所実験で適合性・コストを測り、**静的合成/動的ロード/外部プロセス**は一律決定しない。決定的な純粋Lintは静的合成候補、独立更新・隔離が必要な未信頼解析や外部入力はプロセス分離候補。ただしどの方式でもCapability最小化、入力上限、no network by default、実Load版、部分効果/クラッシュ、未選択非干渉が必要。性能や単一Go Binaryへの収録で秘密漏洩・安全隔離を省略しない。

## 7. 反証可能な独立Expected（**すべて計画段階、NOT RUN**）

| ケース | 固定する差分・反例 | 独立Expected／禁止する推論 |
| --- | --- | --- |
| OK-01 | 有効Concept、型あり | OKF形式適合。ドメインや業務意味の受理は未判定 |
| OK-02 | YAML不正・`type`空、または非root `index.md`にfrontmatter | OKF形式不適合。原文をWriterが捨てない |
| OK-03 | 未知`type`・追加キー・リンク切れ・`index.md`なし | OKF形式適合。適用可能なF1/F2だけ警告/unknown |
| OK-04 | `status`省略／`verified: human:...`のみ | OKFではstable/human-reviewed表記可能。TEI受理・SHO固定納品・Grantの代わりにしない |
| OK-05 | `stale_after`期限内で実Source改版／A→B→A | 時刻による非staleとSource/Reviewの再資格を別列。自動復権0 |
| OK-06 | 値域に不適合な通貨/単位/NULL、同じ文字列で別Meaning | F0 PASSでもF2 invalid/unknown。実Role/Value DomainにBinding出来る範囲だけ判定 |
| OK-07 | Role/Domain候補のみ・未受理/契約版ずれ | 補完はcandidateとして出るが、受理済み確定値としてのLintやSource書戻しなし |
| OK-08 | 参照先が未取得/部分集合/非開示、0件のみ観測 | 完全集合や「規則が無い」と断定しない。Source/meaning/Disclosureを別判定 |
| OK-09 | 同名Typeの2プロフィール競合、同Plugin別Tenant | 既存選択Root/Pin/Grantで判定。ロード順/文字列一致による自動結合0 |
| OK-10 | 2つの独立規則を1文書で編集／1概念を過剰分割 | 文書全体Revisionと局所Claim/Review影響を区別。適切な管理粒度は比較結果から選ぶ |
| OK-11 | Conceptのpath移動・同bytes別Source・複数Document引用 | OKF path IDは変わるがSHO Document/Occurrence/Meaning IDを自動再接続しない |
| OK-12 | 未知YAML/Markdown拡張、BOM/CRLF、未定義参照 | no-op保存でraw Draft保持。Lintが無言で全文rewrite/参照消去しない |
| OK-13 | OKF Plugin absent/disabled/upgrade | 通常Markdown Reader/Writer/TEI意味モデルは維持。旧Lint評価の現在資格を自動引継がない |
| OK-14 | `Attested Computation`の外部executor、資料中の指示 | 読取/静的Lintからの自動実行・外部送出0。Host Grant/実Effectとは別 |
| OK-15 | 原文D同一、図EやRendererだけ変化 | F0/文書byte一致でも表示資産・Review再確認を分離 |
| OK-16 | Plugin追加だけで未選択Targetのビルド・起動に差 | 不要なLink/init/実行副作用を検出。実測されるまで非干渉を主張しない |

実証表は**Source/Writer原byte、Parser profile/実input、OKF conformance、Domain Contract/Value Domain版とCoverage、Lint診断位置、TEI Accepted Meaning、SHO Review、Host/Disclosure、Consumed/Effect**を別欄にする。同一独立Expectedを人工的に変えた検査対象から生成しない。値域診断だけのPASSをOwner Acceptanceや実DB/GUIへ昇格しない。

## 8. 進め方と採否条件

1. **P0：仕様と現在地の固定**。OKF v0.2の仕様commit/対象Grammar、既存TEI/SHOのReader/Writer/Plugin資格、公開可能なSource、独立Expectedを確定。SHO G1b Draft PR #17の限定CIをOKF対応の実証とみなさない。
2. **P1：read-only Knowledge Bundle**。3〜5概念（受注Rule、Data、Business Procedure、設計判断/引用）を元Sourceとは別の派生ディレクトリに作り、root `index.md`と参照を付ける。元Document/TEI Meaning/Decisionを変更しない。検索・読む順序・参照追跡の改善を測る。
3. **P2：2つの独立Lint**。OKF F0（形式）とTEI値域F2（任意）を同じFixtureで比較。新型・未受理Role・欠損・値域衝突・形式適合/ドメイン不適合を区別。CLI/Editorへの橋渡しはread-onlyで先行。
4. **P3：文書単位と保存**。SHO G2/G3の実Writer入口で1 Concept/1 Documentと複合Documentを差分・局所Review・no-op保存・Source移動で比較。SHOの既存Editor/Storageを置換しない。
5. **P4：Plugin選択・2 Consumer**。OKF Plugin有無、TEI CUIとSHO Editor（またはSHOと外部read-only Reader）の2消費者を比較。Bundleの構造/意味Loss、入力支援の正答・誤補完、誤った完全性の主張、I/O/時間/コスト、無効時の非干渉を測る。
6. **P5：Owner採否判断**。対象範囲・正式Plugin contract・文書粒度標準・追加metadataの固定・厳格化するGateは、観測可能な改善と反例の成績を示して判断。既存ADRなしに自動採用しない。

**停止条件**：OKF互換Consumerの未知拡張拒否、Parserが原文を破壊、Lintが受理/権限を偽装、同名/別Scopeへの暗黙Binding、Plugin未使用による不可逆なBuild混入、出典/引用の誤帰属、Editorの自動書戻しでTEI Meaning変更、固定納品の自動置換。問題が起きればread-only投影に戻し、保存・納品・実行資格を分けて記録する。

**今回の証拠状態**：一次仕様と既存Issue/設計の確認に基づく**計画文書**。OKF Plugin実装、ドメインLintの実CLI、SHO G2/G3保存、元資料のOKF化、実Provider/外部Consumer送出、Owner正式受理はすべて**NOT RUN**。既存CLI/CIの限定PASSを本案へ転用しない。

---

## 9. 2026-11-08 横断更新：OKFを共通交換層・独立検証に限定する

**証拠と判断の状態**：本節はGoogleCloudPlatform/open-knowledge-formatの現行v0.2仕様（2026-11-08参照）と、TEI/SHO/SEI/SUI/TAIの既存計画を接合する**追加の設計・実験計画**。新しいOKF Reader/Exporter、Attester、Studio GUI、実Provider E2Eを実装・試験したものではない。前節P0〜P5・OK-01〜16を置換せず、判別力と実行順を補う。

### 9.1 仕様の出典と仕様外の契約を明確にする

- **canonical upstream**：https://github.com/GoogleCloudPlatform/open-knowledge-format/blob/main/SPEC.md 。旧 https://github.com/GoogleCloudPlatform/knowledge-catalog/tree/main/okf は**凍結されたスナップショット**として扱い、現行仕様の正本リンクにしない。採用実験ではcanonical SPECのexact commit/digestとv0.2をEvidenceへ記録し、更新を検知したら再照合する（現時点で実Pin未実施）。
- **OKFの最低適合**：通常ConceptはMarkdown+parseable YAML frontmatter+非空type。未知type、未定義キー、broken link、index欠落、任意のtrust/freshness項目欠落を形式拒否条件へ増やさない。F0形式／F1文書品質／F2ドメイン適合は独立に判定する。形式に適合することはAuthoring・Meaning・利用権限の証明ではない。
- **ファイル境界**：TEI定義Authoringの標準は *.tei.md、旧 *.tei は段階廃止（TEI #2621、統合済み移行計画PR #3030）。*.tei.mdはOKF Bundle内で必須frontmatter/typeを満たす場合だけOKF Conceptとなる。例：order.tei.md → OKF Concept ID「order.tei」。Concept ID（パス由来）≠ TEI Source ID / Stable Meaning ID ≠ SHO Document/Occurrence ID。
- **現在のReader障害**：未対応の *.tei.md をTEI管理Rootに入れても、現行Loaderが意味解析できるとは見なさない。#3031の暫定 fail-closed ガードの適用状態を実HEADで再照合し、選択済み未対応Sourceを黙って0件にしない。初回OKF read-only Bundleは、既存の許可済みSourceから**派生させた別Root**に置き、従来の実CLI回帰を妨げない。Markdown編集ができることとTEI Parserでモデルとして読めることを分離する。

### 9.2 標準データ構造・表現型・Functionの非同型を交換時に保全する

TEIの**標準データ構造／トポロジー**、再利用可能な**標準表現型**、Function Data ContextとActionの**実Binding**、Renderer/Hostの**実実現**を別の参照として保持する（機能パターンOwner #2507、構造 #2495、投影 #2497、Context #2506、Renderer #997）。OKFの見出し、ディレクトリ階層、通常Markdownリンクはこの型付き関係を表現するのに十分でない場合がある。

最小反例は (a) 有向多重辺と属性のGraphをMarkdownに落として多重度が失われる、(b) Tree親子とSUIの複数Island所属が同じアウトラインに写る、(c) 同じData ContextでもTab/Page/Chart等の表現型・軸・単位が異なる、(d) 表現上のボタンが存在してもAction/Grantが未接続、の4件。Exportでは元Entity/Relation/Occurrence/Revisionへの許可済み参照、型付き関係・順序/多重度の損失、保留/未取得/非開示の**有無を開示できる範囲でのみ**表現する。欠落した関係の再構築、空集合推定、逆書戻し、Meaning自動受理は禁止。SEI #820、SUI #3392、SHO #6/#18の既存判別ケースを再利用する。

### 9.3 Attested Computationは「定義」「実行」「効果」「受理」を別票にする

OKF §10のtype: Attested Computationは独立Conceptにruntime、typed parameters、computation、executor.resource/receipt、attester.resourceを記す。Agentは**宣言された型付きパラメーターの値のみ**を与え、正規の計算定義、依存コード、Operator/Plugin Pinを勝手に改変しない。computationはinlineの# Computation fenceまたは参照ファイルでよい。Executor/Attesterのresource文字列は**実行CapabilityやHost Grantではない**。OKFはReceipt/Verdict wire、Attester ABI、sandbox、cacheを規定せず、既存TEI実行経路・Evidenceを使う。

実験時に記録する**照合項目（TEI側の試験票でありOKF公式標準項目ではない）**：
1. 定義：OKF Conceptのexact byte / Source location、TEI受理済みMeaning・Role/Operator、Source/World revision、typed parameters、Plugin/BOM Pin。OKF verifiedは「定義に対するレビュー」でありOwner Meaning Acceptanceではない。
2. 受付と実行：Agentからのbind値・型検査、実Hostのprincipal/Grant/Disclosure、実Executor・compiled/expanded Artifactのdigest、Attempt/Job ID、実Result、Receiptの観測可否。Agentが自己申告したSQLや結果だけを実入力とみなさない。
3. 非LLM照合：独立Attesterによる正規計算+bindと実行Artifactの同一性、Job/Providerからの独立再Readと表示Resultの同一性、使用したAttester版。PASS/FAIL/UNKNOWN/UNSUPPORTEDを区別する。
4. 効果：read-only結果とwrite/副作用を別Gateにする。Receipt存在・committed表記とResource現在値は別。EffectUnknown/AppliedUnobservedなら盲目的再送を停止する。Owner Meaningと現在Host認可もAttester PASSから推論しない。

初回は**非副作用のread-only計算1件**に限定。OKF §10.5のconsumerフローはinformativeであり、今のTEIにExecutor/Receipt APIが存在すると仮定しない。実装前はUNKNOWN/UNSUPPORTEDを返す。TAI #514 AC-01〜12、TEI #2514、#1163、#1174のOwnerと独立Expectedを再利用する。

### 9.4 Plugin-firstドッグフーディングを共通の最小入口にする

[統合済み #3277](https://github.com/hat47x/tei-platform/pull/3277)で定めた「Core以外はPlugin」方向とRQ-008に沿い、OKF Reader/Validator/ExporterとTEI Domain Lintを**論理Capability**として分け、関連する高凝集な配布Bundleへまとめるかは実測で決める。専用OKF Core/Host/Resolver、独立のKnowledge Registry、別Evidence Engineは設けない。

最初に必要な実Pluginは「許可されたBundle Rootを列挙・read-only解析し、F0/F1/F2と未取得範囲を返す」1 Capability。CLI Consumerで**実Provider呼出し**を確認した後、SHO ReaderとTEI Studioに同Contract・Pinで接続する。Studio/SHOだけの再実装やCLI表示のmockを「共通Plugin実利用」と判定しない。Plugin absent/disabled時の通常Markdown編集と旧TEI Readerの非干渉、未選択PluginのGo Build closure、出典/外部HTTPの自動fetchなしを対照する（PD-0〜PD-5、SHO G1b/G2/G3）。初回のread-only利用をTEIの自動実行・SUI全Graph/SEI全Knowledge Exportへの依存にしない。

### 9.5 最小縦切りと既存Expectedへの差分追加（すべてNOT RUN）

| 順番 | 同じ固定Synthetic Sourceでの試験 | 独立判別・既存Owner |
| --- | --- | --- |
| K0 / 定義の固定 | 現canonical SPEC commit、選択Source/World/Plugin Pin、公開可能範囲、必要Concept集合を固定。通常Markdown・*.tei.md・OKF Conceptの境界を表にする | OK-01〜05、TEI #2621、SHO #18 |
| K1 / read-only Bundle | 受注Data/Function/Business説明の3概念とindex、別Conceptの計算定義1件を別Rootに派生。SHO/外部Readerで読めるがTEI Meaningを変更しない | OK-01/03/11/12、SHO #4/#18 |
| K2 / 共通実Consumer | CLIとSHOまたはStudioで同一Bundle/Provider/Pinの実呼出し・診断と原文byteを比較。Plugin disabled/別Scope/broken link/秘匿を対照 | OK-08/09/13/16、TEI #1251/#2516、PD-1〜4 |
| K3 / 型・意味Loss | Data topology、Representation、Function Binding、SUI Tree/Islandを異形に投影し、Relation方向・多重性・Unknown・非開示を失わず説明できるか比較 | OK-06/07/10/15、TEI #2497/#2507、SUI #3392、SEI #820 |
| K4 / 計算Attest | read-onlyの正規計算+型付きbind→実Executor Artifact/Receipt→非LLM Attester→Job独立Read。計算改変/型違反/古いReceipt/Provider partialを一要因ずつ導入 | OK-04/05/08/14、TAI #514 AC-01〜12、TEI #2514/#1163 |
| K5 / Publication | 承認された表示部分だけindex/文書/診断を共有。未開示ID・件数・保留/Sourceの存在を別Recipientへ漏らさない。Writerのno-op保存とOwner Promotionを区別 | SHO #4/#18、SUI #3392、SEI #820、TEI #2516 |

**各K段階のEvidence票**は「source/実Read coverage／Parser・Readerの消費／F0/F1/F2／表現Loss／Meaning受理／Host Grant／Artifact・Result・Receipt／独立Attester／現在Resource Readback／Recipient Disclosure／未検証」を別欄にする。実Attempt IDとSource/World/Artifact revisionを固定し、過去CIの成功を現在版の資格としない。AIが生成した実装と同じAIのExpectedによる自己一致を合格に数えない。SourceのA→B→A、別Root同bytes、古いインデックス、対象未取得なのにcoverageCompleteの4反例を必ず含める。

**今回の着手順**：K0→K1→K2をP0。K3は並行設計・Ownerの異形Consumer試験を流用。K4は初回H2/H3四SessionやTEI *.tei.md Parser正常受入の前提条件にせず、隔離/実Reader/実Receipt経路が整ってからP1で独立検証。K5の自動公開・逆取込・自動実行はP2以降の明示採否に保留。新しい製品Gateや独自Schemaを先行固定しない。

**Ownerと終了判定**：SHO #18がOKF文書/Reader/Writer、TEI #2621が *.tei.md Reader移行、TEI #1251/#951がPlugin Contract/選択、TEI #2497/#2507が意味保存型付き投影、TEI #2514/#1163が独立Expected・実入力、TAI #514がAttested Computation/Harness、SEI #820とSUI #3392が各DomainのExport/Importと開示境界を所有する。本節は統合案の参照地点であって、第二の受入正本ではない。上記K0〜K5の実験と正式Owner受理、実Plugin/Provider/SHO Writer/Attester/配布は**NOT RUN**。

---

## 10. 2026-11-08 他チャット横断深化：未整理Item・受領閉包・部分失効の実証

**位置づけ**：§9のK0〜K5と既存OK-01〜16へ、TEI「機能整理」・Plugin-first・Design Contextテスト、SUI/SHOの実相互運用負例から得た追加の判別条件を渡す。新しい正式OKF Schema、Item taxonomy、Parser、Runner、Authority、Evidence Registry、追加Case正本は作らない。以下の新試験は本計画としては **NOT RUN**。

### 10.1 未整理ItemをConceptへ強制変換しない

- **native captureとexchangeを分離**：TEI/SUIでItemを未分類・未読・カード化前のRaw情報として取り込み、後から開いて整理できる。ItemがTask/Action/Event/Observation/DocumentのどのRoleを持つかは未決でもよい。Task/Action/Eventは別々の意味Role/Trait候補であり、OKFのtype文字列やMarkdown見出しから業務作用・発生済みEvent・自動Workflowを推論しない。Item Identity/内容Version/出典とRole候補・複数のCanvas/Tree/Island/View Occurrenceを分離する。
- **OKF形式との境界**：v0.2 Conceptは非空typeを要求する。未分類Raw Itemを無言で確定業務Conceptに変えず、初回はnativeに原形保存し、選択した範囲だけ別の派生Concept Draftにする。後続のA/B候補は (A) Rawをnative保持して非公開・OKF交換対象外、(B)明示的な包装種別 type: Unclassified Item（TEI/SHOの仮ラベルでOKF標準typeでも業務受理でもない）＋status: draftとして出す。Bで元byte/未決/出典/未知属性を保てないならAへ戻す。OKFの自由型をTEIの正式分類へ読み替えない。
- **同一Itemの複数Canvas所属**：Canvasを分割しつつItem Identity 1／Occurrence Nを許容する。複数出現、Conceptの置き場所、Markdownリンク数から独立の元Source N件や親子・所有権を推論しない。座標・Island所属/ネスト・Tree順序・多重有向Edge・保留を別々に保ち、許可された型付き参照またはLossとして表現。非公開Item/Edge/Countの有無をLoss説明自体から漏らさない（TEI #2497/#2507、SUI #3392、SEI #820）。
- **逆Import停止線**：OKF Conceptのtypeや見出しはnative Item Roleの受理、Taskの実施、Action Dispatch、Event観測ではない。受領Conceptは元Source参照付き候補として隔離し、正規Binding・Owner判断・現在Grant・独立Expectedまで元Worldに自動書戻し0。

### 10.2 ProducerからConsumerまでの一往復を分解して検収する

**既存の局所実証と上限**：
- SUI [#3396](https://github.com/hat47x/sui-sensemaking/pull/3396)は実buildContextProjectionで、元Source/島所属/座標が異なる二Worldから同じ公開Projectionを生成できる非単射を3件で検証。SUI [#3397](https://github.com/hat47x/sui-sensemaking/pull/3397)は可視・公開許可済みの内容だけを通常Markdown草稿にする5件の限定実試験を持つ。欠けたSourceや未公開Count/Island、OKF sources/verifiedを捏造しない。DraftでありOKF Concept Export成功ではない。
- SHO [Draft #21](https://github.com/hat47x/sho-composer/pull/21)の合成354 UTF-8 bytesはWindowsのcheckout EOL変換で368 bytesとなりRaw identityが一致しなかった。合成FixtureのLF固定で2 OS Reader回帰を確認したが、実SUI Producer→SHO CLIが**同一の一File**を受け渡すH1は最新のRunner CIでSKIP。偽Checkout起動前拒否PASSを実H1 PASSにしない。SHO ReaderがMarkdown link_open 0件と示しても元のSource 0件/全資料取得とはいえない。
- TEI [#3262](https://github.com/hat47x/tei-platform/issues/3262)の既存実CLI9応答を使う試験用Context Gateは [merged #3284](https://github.com/hat47x/tei-platform/pull/3284) / [#3296](https://github.com/hat47x/tei-platform/pull/3296) に限定のsynthetic/transport **計16負例**を持つ。正式Product Design Context Query、外部Harness Replyの全受領、OKF全Concept閉包を証明しない。

**K1/K2の観測点**：①Producer実Source Read（workspace/manifest/realpath/revision/actual bytes/allowed scope）、②P: Data→可視Projection（どのRole/Relation/SourceとDisclosureが現れたか）、③Q: Projection→Markdown/OKFの実生成byte・index・Concept集合・Loss、④OS/File/Distributionの実物理byte/digest/EOL、⑤Consumerが実Readしたbyte/Parser profileと受領範囲、⑥独立Required集合に対するF0/F1/F2・Meaning・Disclosure判定、⑦Owner Promotionと現在用途での独立の資格を別票にする。Pで失われたSource/IslandはQで自動復元できず、①未読なのに③〜⑤がPASSしても原資料確認済みにはしない。

**受領閉包**：#3262のRequired→Discovered→Loaded→Parsed→Bound→Returned→Receivedを再利用。成功したCLI call、JSON parse、index列挙、モデルRevisionや古いPASSを全件証明にしない。独立Required集合・ordered pages・end marker・各part digest・実Consumer receipt、workspace/manifest identity（別Workspaceに同Canonical IDがある場合）を照合不能ならpartial/unknownのまま停止。表示等価なEOL正規化と原byte同一性を別判定。Workspace識別不足の修正OwnerはTEI #3181/CLI契約側であり、本書は別Readerを設けない。

### 10.3 何が変わったかで部分再資格化を決める

| 変化 | 再確認すべきもの | 自動扱いしないもの |
| --- | --- | --- |
| 原Source移動・同bytes別Root・A→B→A | realpath/Source identity/引用Occurrence/現在用途Review | byte一致による旧受理・旧Referenceの復権 |
| Item本文・Claim・業務Ruleの変更 | 該当Meaning/依存投影/Domain Lint | 無関係な全Concept/全Pluginの失効 |
| Canvas座標のみ変更 | 幾何解釈/当該Viewの投影 | 原本文/Value Domain/業務判断の無条件失効 |
| 島所属・Tree親・Edge変更 | Topology・関係・B型説明・P/Q Loss | 本文同じだからWorld不変とする |
| 非公開Item/Count変更 | 内部のDisclosure再審査 | 他Recipientへの存在・件数の開示 |
| OKF verified/status/期限変更 | 文書のtrust/freshness・原Source時点照合 | TEI Meaning・Host Grant・Attester歴史の改変 |
| 未選択Plugin/Generator変更 | Catalog候補/必要ならBuild closure A/B | 無関係Targetの一律rebuild・再受理 |
| 選択済みProvider/Toolchain/Attester変更 | 実Consumed Lock/Artifact/Receipt/独立Expected | 同Source hash/旧CI PASSの無条件流用 |

これは新Invalidation Engineでなく、TEI #2600/#1251/#2497、SHO #4、SUI #3392、SEI #820の既存判定の接合票。OKF stale_after単独ではSource移動・A→B→A、Owner撤回、Toolchain差替を検知できない。World Fingerprint、Model Revision、Artifact Revision、Actual Input、Recipientの同一性を別扱いとする。

### 10.4 OKFの知識BundleとPlugin BOM・Runtime Receiptを分ける

| 対象 | 契約 | 避ける誤認 |
| --- | --- | --- |
| OKF Knowledge Bundle | Markdown Concept/index/logをまとめる**交換・配布単位** | path由来Concept ID≠Item/Meaning/Document Identity、index≠必要集合完全 |
| TEI Plugin Bundle / Resolved BOM | 選択Capability/Provider/Implementation/Targetの**構成・実Build閉包** | Plugin ID≠Capability ID≠Business ID、未選択Provider混入なしを実Build確認 |
| Actual Artifact / Run Receipt | 実Generator/Input/Target/Host/Attemptと実Resultの**観測** | OKF verified、Recipe、形式PASSは実Build/Attested Runの証拠でない |

共通Pluginは高凝集のBundleで配布してよいが、F0/F1/F2とAuthorityの判定を別にし、discovered→selected→build-consumed→artifact-included→runtime-invokedのどこまで達したかを同一Scenarioで表示。初回K2はCLIとSHO/Studioの実二call-siteで同じread-only Contract・Provider Pinを消費することを証明する（#1251/[#3277](https://github.com/hat47x/tei-platform/pull/3277)）。OKFのAttested Computation Run ReceiptはBundle**外**のruntime artifactであり、TEI既存Host/Evidenceへ委譲する。

### 10.5 P0実行票・既存テストへの割当

- **K0-A：Required固定**。公開可の1 Workspace・1 Function/受注Ruleと、必要Data/Function/Business/Relation/Source/Rule集合を独立に定義。元Sourceの実読取・manifest identity・World・Plugin Lockを確認し、未探索を0件として合格させない。既存#3262のContext Gateと#2514の独立Expectedを再利用。
- **K1-A：小さなBundle**。許可済みSourceから3 Conceptとroot indexのread-only派生を別Rootに作る。未分類Raw Item/既存*.tei/SHO正本を上書きしない。F0合格と「原Sourceを読んでいないのでMeaning/Provenance未資格」の負例を同時に保持。
- **K1-B：既存SUI/SHO引継ぎ**。OKF型付けより前にSUI #3397→SHO #21の実一File H1、Raw 354 bytes、Checkout/EOL、元出典未読、Producer/Consumer境界を検証。現CIのSKIPをPASSにしない。K1-AをこのH1の成功までブロックしない。
- **K2：二Consumer**。CLIとSHOまたはStudioで同じContract/Bundle/Provider版を実消費し、F0形式・F1引用・F2 Domain・原byte・未取得/非開示・全応答受領を独立判定。G1b通常Markdown Reader成功はOKF F0 Validator実成功とは別。Plugin OFF/AI OFFでも通常Markdown編集を維持。
- **Case帰属**：OK-03/12（Raw未整理/未知type）、OK-08（0件/部分/秘匿）、OK-10/11（同Item複数Canvasと同文異World）、OK-05/15（座標のみ失効）、OK-09/13/16（Provider選択と非干渉）、OK-14（外部文書の実行指示非実行）。SUI SU-OKF01/03/04/08/10、SHO #4/#18/#21、TEI #3262/#2514、TAI #514の独立Expectedへ関連付け、新番号・第二PASS台帳は増やさない。

**即停止**：独立Requiredがないcomplete、Loss不明のroundtrip、未分類Itemの自動意味分類、文書typeを見たAction自動Dispatch、raw変換を視覚等価で隠すこと、秘匿Count/ID漏えい、verified→Owner受理/Grant自動昇格、未選択PluginのBuild/init混入、実Receipt・Result未読のAttested PASS。書込み/外部送出/自動Import/恒久公開は既存Host・Ownerの別資格まで行わない。

**証拠資格（現在）**：SUI投影非単射・通常Markdown Draft、SHO合成ReaderのOS別回帰、TEI Context試験用Consumerの負例は**別々の局所実測**。OKF Validator実装、OKF Bundleの実2 Consumer、SUI→SHO実One-File H1、元Source→OKF→Consumer全件、SHO Writer、SEI Import、TEI Meaning受理、Attester/Provider Effectは**NOT RUN/UNQUALIFIED**。今回の変更は文書とOwnerへの受渡しのみ。


### 10.6 Traitの強制力・C0/C1・独立検証基準をP0受入へ加える

**Plugin Trait/RoleはOKF typeと別**：TEIではTraitを原子的な必要条件として合成し、付与したTraitが要求するContract/Shape/Operationを既定実装または明示的な実装必須診断で満たす設計を比較している。OKF側の自由なtype/tags/拡張キーの出現だけではTraitを適用しない。明示Mappingがある場合も、受理済みRole・Required Shape・Plugin Pin・実Provider Conformance・Current Grantを別票とする。候補Trait A+Bの合成が個別成立する一方で複合実装は不足、Aだけ実装済み、Aの版だけ変更、同名Traitだが別Scopeという4反例を#988/#2620/#1251/#951の既存受入へ渡す。OKF Readerが正常でもTEI専門Plugin Contractを満たしたとはしない。

**P0 Evidence Acceptanceの五つの欄**（新Gateではなく#2514/#1163/#3262/#1180の記録項目）：
1. **独立Required**：対象Function/Bundle/Consumer/用途と、必要Source・Role・Relation・Claim・Concept・Negative dependencyの集合を実Producerの都合から独立に先に定める。不可視Scopeは開示せず内部で必要条件を判定。
2. **Actual Consumed/Delivered**：実Root/Manifest/CLI Binary/Tool/Plugin/Source Revision、実Read/Parser/Provider取得、全Page/Reply/Part digest、Consumerの受領確認を同一Attemptで照合。K0/K1の計画が存在するだけではPASSにしない。
3. **独立Expected/判別力**：正常正例と一要因負例の両側で判別。ExpectedをCandidate/Exporter/Attesterと同じ実装・生成器から機械的に作らない。Known Mutantを対象Case自身が検出したかを区別し、他のCI job失敗をMutant killとしない。
4. **Coverage**：宣言されたObligationの分母、実検証された分子、行/分岐C0/C1、Source/Relationship/Disclosure/Effectの必要条件を**別々に記録**。#3262の試験用Context Gateの局所C0相当10.1%や16負例を、TEI製品全体のC0/C1、未カバー要件のゼロ件証明として扱わない。AIが新しい未検証Ruleを発見した場合は候補のまま#2514/#2018へ。
5. **Current Qualification**：OKF形式/SHO Reader/TEI Meaning/実Host Grant/Provider ReadまたはEffect/Attested Run/Recipient Disclosure/Owner正式受理は独立資格。過去CI PASS・同一Bundle SHA・verified・実Tool一回成功で現在適用を自動資格化しない。

**P0での具体的な独立負例**：Required Concept 2件に対して実Readerが1件だけ受領したのにindexがcompleteと示す、Consumerの最後のPage/end markerが欠落、同名WorkspaceでManifestが異なる、Role A/BのうちBだけ未実装、2 Canvasに同じItemがあるので独立根拠2件と誤計数する、AIが作ったExpectedとExporterが同じ欠損を共有する、という6つの差をそれぞれ既存OK-08/09/10/11/13と#2514/#3262のテストOwnerへ接続する。T0実Readerの確認が成立した部分に限り、新義務発見のT1テスト生成を提案。新Reader/Runner/型強制の採用・C1達成・製品OKF E2E・正式Owner受理は**NOT RUN**。

---

## 11. 2026-11-08：形式適合と利用資格、実効隔離、検証証拠の二次公開を分ける

**位置づけ**：§9のK0〜K5、§10のItem/Source受領閉包と既存OK-01〜16を保持し、TEI #2515/#2516のOS/送出・監査Evidence、#3262のContext Gate、#2605の実CLI/Skill、SHO #18、SUI #3392、TAI #514を接続する。第二のReader/Policy Engine/Host/Fixture正本を作らない。**本節の追加実験はNOT RUN**。

### 11.1 OKFは「寛容な形式Reader」＋「用途別Admission」の二段階

一次仕様は [GoogleCloudPlatform/open-knowledge-format SPEC.md](https://github.com/GoogleCloudPlatform/open-knowledge-format/blob/main/SPEC.md) v0.2。今回確認できた原文の**blob SHAは c06e3eede0c910d0ecf12524c34204156f8795ac**であり、Repositoryのcommit Pinとは異なる。K0の実験にはSPECの**exact commitと原byte digest**を別途固定する。

- **F0 OKF形式**：Conceptのparseable YAML frontmatter＋非空type、予約index/log構造を判定。未知type・追加key・任意情報欠落・broken link・index欠落はv0.2で形式拒否してはならない。verifiedが単一mappingで書かれたときは検証イベント1件として受け取る。root indexの将来版が不明でもbest-effortで読める部分を保持し、未解釈部分を隠さない。
- **F1/F2と用途別資格**：形式PASSは業務用途のRoot・必要Source/Role/Relation/Domain・現在Grant・Consumer受領を保証しない。未取得や未知版は、その用途の未適格/unknownとして止めるが、通常MarkdownのRaw閲覧・保存は妨げない。管理対象の未対応 *.tei.md をTEI Loaderが明示拒否する境界とは分ける（#2621）。
- **trustの限界**：verified.byにhuman:という接頭辞があるとOKF trust tier上はhuman-reviewedとなる。しかし文字列だけで実本人認証や業務Owner受理を証明しない。status省略時はOKF上stable扱いでもTEI Meaning Acceptedではない。stale_after期限内でもSource改版やOwner撤回による再資格が必要。
- **Sourceの非一様性**：sources[].resourceには具体的なURL/Pathだけでなく、直接辿れない**集合・Scope descriptor**がある。全resourceを実原本URIとみなしてfetch・リンク解決・0件不存在証明を行わない。単なるMarkdown linkもSource Identity/実アクセス権/真正性ではない。

**二段階の出力**：F0 valid/invalid、F1文書品質、F2 Domain valid/invalid/unknown/not-applicable、Consumerに実Loaded/Parsedされた範囲、独立Required closure、現在のDisclosure/Host資格を別列。形式がvalidでも用途が不適格なら静的閲覧のみ許し、意味Promotion/実行/公開は止める。未知拡張を消す・全Bundleを無条件拒否する・不明を完了へ昇格させる、の三誤りを検出する。

### 11.2 OKFのMarkdown/Frontmatter/Executor参照は未信頼の入力データ

OKF Bundleとそのindex/log、他者生成Markdown、executor/attester.resource、手順文を**Agentへの命令やTool許可として解釈しない**。初回F0/F1は本体のReader/Validatorだけで処理し、外部Sourceの取得、リンク修正、Plugin導入、SQL/Shell/Script実行、文書保存、公開は別Operation/現在Grantへ。

**Source境界**：許可Root、実realpath、symlink・相対Path脱出、通常File/特殊File、サイズ制限、BOM/CRLF、Read中の変更、同名異実体を検査し、Source/Readerが確認できなかった範囲をpartial/unknownとする。SHO G1bの既存Reader契約を利用し、巨大PDF/Excel/WebをEditorの文字入力時に自動fetchしない。ファイル名・Root宣言と実OSアクセス制御は別である。

**read-onlyの実効境界**：CLI read-onlyフラグやPluginの効果申告だけではOS上の無権限Read、外部Network、子Process、Tool result、stderr、Model input、Session Log、telemetryの送出を防げたといえない。TEI [#3113](https://github.com/hat47x/tei-platform/pull/3113)のDSH実試験では作業Root外のReadを許したため、H2/H3の本来の隔離条件を満たさなかった。初回OKF外部Harness実験では [#2515](https://github.com/hat47x/tei-platform/issues/2515) のOS Filesystem/Network denialの正負、実Sender/Receiver、未計測出口を別に記録する。これはK1のローカル純粋ReaderのHard dependencyにはしない。

**Writerの境界**：F0/静的Lint/Readはno-op。保存/Auto-fix/Index書換/Remote Syncは明示的なIntentとCAS/Revision照合が必要。Plugin不在・壊れた未完成MarkdownでもRaw Draftを保存可能にし、Accepted TEI Sourceへの暗黙書戻し0。Dirty bufferに古い非同期解析結果が後着したら、本文と診断のVersionを照合し旧結果を現在値へ上書きしない。

### 11.3 Evidenceの観測者と公開先を分離する

TEI [#2515](https://github.com/hat47x/tei-platform/issues/2515)が扱うH6-E13〜15は、**検証器やMock Receiverが集めた生Evidence自体が別Agent、公開Issue、CI Artifactへ漏れる**問題を独立に扱う。OKF K2/K5へ接続する。

- **保護領域の内部Observer**：無害の固定Synthetic canaryを用い、実Source/Network/Host/Tool経路のwire byte・Process identity・OS拒否点・Plugin/Tool版・見えたSinkの集合を記録。内部の生データを一般ログ・公開Issueへコピーしない。
- **外部Viewerに出すProjection**：Recipientの現在Policyが許すCase ID、検査対象の公開可能なScope、確認できた成功/拒否/unknown/not-run、Reader/Verifier版、独立Expectedとの一致といった限定結果のみ。未公開Source path、ID/count、Ref、診断理由、Root構造の存在自体も漏らさない。元のReader Grantを再配布Grantへ自動引継がない。
- **否定主張の上限**：あるMockで受信0件でも子Process、別Socket、DNS、Telemetry、Cloud側など未観測の送出出口が残る。全出口の非漏えいを推論せず、観測範囲と未計測経路を別票にする。漏えいした過去の送信は後のGrant撤回で取り消せない。

**既存独立Expectedへ接続する負例**：公開World同一・秘密Worldだけ異なる2通り（SUI SU-OKF08）、診断に非公開ID/Countが含まれる、静的Readerが外部Linkへ自動アクセス、Source-Grant失効後に旧cacheがToolへ送出、Mock raw bodyをCI Artifactへ無条件添付、executor.resourceの文字列をAIが自動実行、拒否した理由から他Tenantの存在が漏れる。新Case IDや第二Disclosure Engineは作らず、#2515/#2516/#1178、SHO #18、TAI #514がOwnerとなる。

### 11.4 Source更新から受信までの版・実入力競合

R0（ProducerがSourceを読んだ時）、R1（派生Bundleが書かれConsumerが受領した時）、R2（文書を公開/Meaning Promotion/Attested Runする時）は別時点・別資格。Source A→B→A、同byte別Root、原Fileを読取中に置換、bundle Indexだけ新しい、Writer dirty buffer、Consumer旧Replyなどを独立負例へ。実Source/Manifest/World、Writer Document/Occurrence、Projection Loss、Artifact/Runtime、Audience資格を確認し、同じbyte SHAやOKF verifiedだけで旧Reviewを自動復権させない。別世界で同じMarkdownを生成できるSUI #3396の非単射はここでも有効。

### 11.5 次のK0〜K2を既存Contractだけで実行できる形にする

| 段階 | 先に固定する入力と実検査 | 合格主張の上限 |
| --- | --- | --- |
| K0 現行資産棚卸し | canonical SPECのexact commit/byte、既存CLI binary/version/help、実Workspace/Manifest/Required、SHO G1b Reader、Reader権限/公開Audience | 既存CLI/SHOの観測のみ。新OKF Adapterの実装とはしない |
| K1-F0 Bundle検査 | 3 Concept+indexに未知type・追加key・broken link・単一verified・Scope-only resource・必須YAML/type不備を一因子注入 | OKF形式とF1/F2/用途別未資格を判別。既存Markdown ReaderをF0実Validatorと呼ばない |
| K1-IO Source境界 | 許可Root/File、Root外symlink・古いcache・Source途中変更・未取得、明示Preview前の外部I/Oを観測 | Readerのlocalな非書込みとOS/Egress全経路証明を混同しない |
| K2-Consumer実受領 | 同一RevisionのBundleをCLIとSHO/Studioで同じread-only Contract/Providerから実Read。Raw byte・Part/page終端・Scope/Source loss・Consumer側実受領を照合 | 実2 Consumer呼出とRequired closure、元出典/Current Grantを別資格にする |
| K2-Publisher/Observer | 保護内部Mock/OS evidenceと、公開Viewerへの許可済み要約を対照 | 観測されたSinkでの限定結果。公開禁止情報の不在、未計測出口を別列に残す |

**P0受入に必要な失敗の判別力**：形式上許容すべき未知typeを誤拒否しない、必須type欠落を誤合格しない、Source未取得を存在0としない、対象Bundleの全受領をReply一件で推論しない、Root外Read成功をread-only安全と呼ばない、内部Raw Evidenceが公開PR/Model入力へ漏れない、read-only実行不能なexecutor resourceから実効果を起こさない。既存OK-01/02/03/04/08/12/13/14、TEI #3262/#2514/#2515、SHO #4/#18、TAI #514の既存反例へ割当。

**停止条件**：破損データをOKFから消してからPASS、閲覧できただけでAttested Run・Meaning/Owner受理、未計測出口込み漏えい0の断言、現在Audienceに無許可なSource/diagnostic公開、旧Evidenceの再資格化、実Kernel/OS Sandbox未確認のまま外部Harnessの資格主張。外部公開・外部Tool/RunPlan・不可逆なEffectは別のHost/Owner許可まで止める。

**Owner/現状**：SHO #18（文書/Reader/Writer）、TEI #3262/#2514（独立Required/検証基準）、#2515/#2516/#1178（実隔離/二次Disclosure）、#1251/#951（Plugin選択）、#2621（Authoring）、#2600（失効）、SUI #3392／SEI #820（元関係・Claim）、TAI #514（Attestation/Skill）へ分担する。**今回の検証はGitHub一次資料・上流SPECの静的照合のみ**。新OKF Validator、Consumer実2経路、公開Observer/Viewer対照、OS/Network安全、Attester/実Provider/Owner AcceptanceはすべてNOT RUN。


### 11.6 実Invoke variantとSession再開時の情報流を再判定する

**ProviderやCLIの名称・宣言だけではRead-onlyを保証できない**。既存TEI [#3182](https://github.com/hat47x/tei-platform/issues/3182)と改善Draft [#3302](https://github.com/hat47x/tei-platform/pull/3302)では、read-onlyと広告されるview系CLIでもexecute/operation-logなどの実Flagによってファイル出力が生じ得る。これをOKF Reader/Exporter/Attested Computationの誤実行を防ぐ追加圧力とする。**Invocation variant＝実Command名＋argv/flags＋stdin＋現在Source/Target＋Plugin/Host Grant＋出力先**を単位に、実Effectを検査する。Tool Catalogのeffect:read-onlyや利用できるTool名だけから、全Flag variantが無副作用とはしない。#3302はDraftでありTEI CLI全体修正済みとは扱わない。Provider/Hook/Operation Offerの広告と実到達性は重複せず [Plugin-first Draft #3338](https://github.com/hat47x/tei-platform/pull/3338) のPD/PO受入へ渡す。

**受信者SinkとSession再開**：既存 [TEI #3309](https://github.com/hat47x/tei-platform/pull/3309) の合成canary試験（Open）は、read-only DSHでTool結果がstdoutと永続Session Logに残る局所結果を示した。一方、[main統合 #3237](https://github.com/hat47x/tei-platform/pull/3237)のH6-E7合成試験ではSession再開時に**既受理Log区間は再送されないが、前TurnのRaw canaryが再構成されたModel入力には再登場**した。したがって「Logに再送なし」「新Tool readなし」「Pluginがread-only」のいずれも、継続SessionへのRaw再提示がないことを保証しない。OKF/SHO/SEI文書の旧引用・内部診断・未受理SourceをHarnessへ載せた場合も、再開/撤回後のModel入力・Tool結果・永続Sink/外部公開の資格を既存#2515/#1178/TAI #232で別判定する。

**追加の一因子負例**（新Case IDなし）：A同一OKF ConceptでReader純粋preview vs execute指定、B Provider effect read-onlyだがTool/Session LogへCanaryが複写、C Recipient権限がR0→R1で撤回された後のSession再開（旧Model Contextが残る）、Dログ受理がなかった場合と、モデル入力に再登場した場合を混同して偽の非再送PASS。Expectedは「許可された純粋ReaderではReader由来の新外部副作用なし」「未許可Variantは実行前拒否または別Host資格へ」「既観測の過去送出は撤回されない」「全Sinkの不在は実観測なしにはunknown」。OS/Network/Model入力の本番遮断は**NOT RUN**、新Harness/Plugin Engine/強制Policyは設計しない。

---

## 12. 2026-11-08 K0実Pin・K1合成Fixtureの実配置と未実証境界

**これまでの§9〜11との関係**：K0/K1の**入力と期待結果の置き場所を実際に作った**。K2実2 Consumerは未実施。ここに記す正例/負例はOK-01〜16と#2514/#3262のOwnerの受入基準に対応する**暫定Fixture**であり、第二のOKF標準・Test Runner・Meaning正本ではない。新OKF Validator/Plugin実装を行ったという意味ではない。

### 12.1 公式仕様の実Pin（K0の技術入力のみ成立）

- official repository: GoogleCloudPlatform/open-knowledge-format、対象 [SPEC.md at commit ad30107c31c06aec8a7d5636e0d1058118604e6f](https://github.com/GoogleCloudPlatform/open-knowledge-format/blob/ad30107c31c06aec8a7d5636e0d1058118604e6f/SPEC.md)、OKF v0.2。
- 実Git readbackによるGit blob SHA-1: c06e3eede0c910d0ecf12524c34204156f8795ac。原文UTF-8 SHA-256: **26aa5da029278939f914e578107242d9607d4f2dc5fe153272b82f9ed1030101**。計算器はSHA-256既知の空文字・abc二ベクトルと一致することを先に確認した。Git commit・blob SHA・原byte digestを混同しない。
- 旧knowledge-catalog/okfは凍結snapshot。上流mainの将来移動を自動追随せず、本Fixtureの判定は上記Commitで維持する。新Specでの再試験は別Source revisionとして扱う。K0の**全Product Source/Consumer/Provider/Host Pinは未完成**。

### 12.2 実配置した小さなBundle（K1 P0入力、全て合成・公開可能）

- [index.md](../../testdata/okf-k1-v02/index.md)：root限定のokf_version 0.2、Data/Function/Business相当の3概念を列挙。
- [data/order.md](../../testdata/okf-k1-v02/data/order.md)：Data Entity。sources.resourceは直接fetch可能URIでなく**集合を説明するScope descriptor**。実Data ProviderのReader成功を表さない。
- [functions/submit.md](../../testdata/okf-k1-v02/functions/submit.md)：Function。通常の相対LinkでOrder Conceptへ接続。OKF v0.2のbundle-root基準絶対Linkは一時Copyへ注入して形式適合を別試験する。verifiedはsingleton mappingの合成processイベント1件。人間認証・意味受理・実計算Receiptではない。
- [business/submit.md](../../testdata/okf-k1-v02/business/submit.md)：未分類Raw Itemの**交換包装候補**。type Unclassified Draft Item／追加Frontmatter x-tei-experimentを含む。通常は解決可能な相対リンクを持つ。broken linkは一時Copyへ注入し、OKF v0.2 F0適合とF1警告を別検査する。役割Task/Action/Event/Traitの採用を示さない。
- [testdata/okf-k1-v02.expected.json](../../testdata/okf-k1-v02.expected.json)：3 Concept/1 index、正例/負例、上流Spec exact commit+raw digest、各実File raw SHA-256、できたこと/できないことを別票で固定した**暫定期待**。同じ作成Attemptで設計したため、実装と独立の仕様由来であっても**Ownerが独立に審査した検証基準ではない**。#2514の適格性・既知Mutant killが成立するまで合格基準へ昇格しない。
- [repository .gitattributes](../../.gitattributes)へこのBundleとexpected.jsonの-path -text を追加し、Windowsのcheckout EOL変換でRaw期待が壊れることを防ぐ。SHO #21の354→368byte負例を受けた限定対象の対策であり、通常Markdown編集や別のRaw Sourceの改行変換を強制しない。

実GitHub File Readbackで5ファイルとshaを取得した。上流Specと各FixtureのRaw UTF-8 SHA-256はexpected.jsonに書いたものに一致するよう固定。**これは形式適合の実Reader検証ではなくFile/byte入力同一性の静的基線**。

**2026-11-08 CI負例と修正方針**：K1初回実Bundleでは、OKFとして有効な `/data/order.md`（Bundle-root絶対Link）と、形式上許されるbroken link `/business/missing.md` を同梱した。しかし[TEI #3329の旧HEAD CI #42392171058](https://github.com/hat47x/tei-platform/actions/runs/42392171058) の `go test ./tools/repository-plan-check` / `TestMarkdownRelativeLinksResolve` が両件を**実Fail**と判別した。これはOKF v0.2 F0仕様違反ではなく、TEIリポジトリ全体のMarkdownリンク存在検査との**用途・Root解釈の衝突**。現行CheckerをOKFのために緩めない。コミットする正常BundleのLinkは相対かつ解決可能に限定し、broken linkとBundle-root絶対Linkは一時Copyへの独立変異で試験する。成功CIが得られるまで本修正をPASSと呼ばない。原Fixture Raw SHA-256を新原文で必ず更新する。

### 12.3 素朴なFixtureがあっても必ず区別する対照判定

| 個別Mutant/対照 | 公式OKF v0.2の仕様上の期待（製品未実行） | 進行を止める領域 |
| --- | --- | --- |
| 基線:3 Concept＋root index | F0適合候補、F1 unresolved link警告、F2 TEI Meaningは未資格 | 業務Role/Source実Readの偽受理 |
| 1 Conceptのtypeを削る | F0非適合 | OKF形式の受入 |
| YAML frontmatterを壊す | F0非適合 | OKF形式の受入 |
| 未知type、追加key | F0は拒否しない | 該当TEI Roleが実装/受理済みとの誤認 |
| index.mdを欠く | F0は拒否しない | Required全件を読んだとする推論 |
| Concept内Markdown Linkが壊れる | F0は拒否しない | 出典/参照品質・必要Relationの保証 |
| verified単一mapping | verification event 1件として処理 | human Owner承認/実Attesterへの昇格 |
| sources.resourceがScope descriptor | 元Sourceの実fetch未成立と表示 | 0件/存在しない/原資料読み取り済みの断定 |
| 同じConceptを複数のCanvas/SHOで表示 | 文書回数とnative Item Identity/Source根拠件数は別 | 意味上の二重計数・無断逆Patch |
| 原byteは同じでSource Rootだけ変える | Git SHA・内容が一致してもSource Identity/Grantを再評価 | 誤ったCurrent Qualification再利用 |

1要因ずつ変異させる際は、元fixtureファイルを上書きせず別の実行用Copyに適用する。変異後の現在Source/World・Output/Actual Readと元のExpectedのRevisionを同一Attemptで結び付ける。AIが作ったExpectedの自己一致だけでPASSにしない。

### 12.4 実装とテストの最短順（既存CLI/SHOを無理にOKF扱いしない）

1. **K0 Reader Fit/Gap**：TEI #3262の実CLIにOKF Bundle/Concept Readerが既に存在するか、SHO #18 G1bがどこまでFrontmatterを読めるか、現在のactual argv/Source Scopeで確認。未提供ならunsupported/NOT RUNと記録し、仮のteeコマンドや未知のMCP Toolを作らない。通常Markdown読取成功をOKF F0合格にしない。
2. **K1 最小実Validator**：既存の担当Plugin/SHO Reader経路でK1の正例と最低4負例（type欠落、壊れたYAML、未知type、broken link）を**同じFixture Copy**から独立Expectedに対して比較。必要なParserが未実装ならGitHub上の静的検査だけを記録し、実Validator成功を主張しない。短い仕様のため新万能文書エンジンを作らない。
3. **K1 Source/Disclosure**：Scope descriptorにHTTP fetchを試みない、外部Linkや未知typeからToolを自動起動しない、実Readerが未取得/partialを示す、秘密Rootの有無を公開診断で漏らさない。実OS Network否定には#2515の独立観測を要する。
4. **K2二Consumer**：共通選択済みOKF Reader/ValidatorのContract/Plugin Provider Pinを、CLIとSHOまたはStudioの**実二箇所**で同じBundle/Raw digestから呼び出す。Spec-valid、F1文書品質、F2業務、実Read/受領Coverage、Owner/Grantを別列にする。元SUI→SHO一File H1はSHO #21、SEI Claimは#820、Attested ComputationはTAI #514の別資格。
5. **Owner/CI**：#2514/#1180の独立Expectedとmutant判別力、#3262のRequired→Received、#1251/#951のPlugin実選択、#2515の安全な実Sink検証を使い、合格した実段階だけ進める。完了した証拠はtestdataや計画から自動昇格させず、既存Evidence票とその実CI/Receiptへリンクする。

**今回の到達点**：K0の**上流仕様Source Pin成立**、K1の**固定合成Input＋暫定Expected＋原byte EOL制御のGit実配置とReadback（CI負例のリンクを修正した最新版の再照合は別記録）**。まだK0の実Product依存閉包、独立Expectedの正式受理、K1 F0 YAML Parser/Validatorの実行、K2実2 Consumer、SHO Writer、SEI Import、OS/Egress拒否、外部Harness/Attester/Owner Meaningは**NOT RUN／NOT QUALIFIED**。§9〜11の計画全体が成功したという主張ではない。

---

## 13. 2026-11-08 K1実CI負例とK2の現Main Reader/Context再利用

**位置づけ**：この節は第12節で追加した固定合成Fixtureを実Go検査へ接続する狭い差分と、並行mainで成熟したTEI Design Context/Plugin-firstの実測を接合する。新しい第二OKF Parser/Context Query/Host/Scenario正本は作らない。上流OKF v0.2 F0適合を既存TEI Go testで代用しない。

### 13.1 最初のRepository Health FAILを運用要件へ吸収する

旧PR HEAD `665ce2042277f5f62371684900b2f42363333dbe` の [CI #42392171058](https://github.com/hat47x/tei-platform/actions/runs/42392171058) は Repository Healthが**FAIL**。実ログの原因は既存 `tools/repository-plan-check/plancheck_test.go` `TestMarkdownRelativeLinksResolve` が、（a）公式OKF形式では有効なBundle-root absolute link `/data/order.md` と、（b）OKF F0が許容するbroken link `/business/missing.md` の双方を「TEIリポジトリ内の相対Markdownリンク解決失敗」として報告したこと。リンク許容範囲が異なるためであり、**OKF仕様非適合という意味ではない**。

**修正済み設計入力（最新CI確認前）**：コミットする正常BundleのLinkは `../data/order.md` に統一。broken linkとbundle-root absolute linkは`testdata/okf-k1-v02.expected.json`の**一要因Mutant**として、実行時に作る一時Copyへ注入する。これにより、TEIのRepository Healthを弱めず、OKF F0の許容性とF1/Source Resolverの注意表示を両方検査できる。負例を消して検査を回避したのではなく、**常設の正常入力と実行時負例の置き場所を分離**した。旧HEADのCI FAILは失敗の証拠として保持し、修正HEADのCI Successを未確認の段階ではPASSと書かない。

### 13.2 Test-only Witness追加の実装資格

[tools/repository-plan-check/okf_k1_witness_test.go](../../tools/repository-plan-check/okf_k1_witness_test.go)を**既存テストPackage**に追加した。これはReader/Parser/Pluginではなく、K1に固定した元Markdown4ファイルと暫定Expectedの原Byte/SHA-256、UTF-8/LF、Requiredファイル集合、検証資格が`PROVISIONAL`/F0 `NOT RUN`/Owner `NOT RUN`のままかを検査する。試験内のCopyにだけリンクMutant文字列を作り、管理Sourceを書き換えない。これが正常実行されても、**raw-source integrityのみの局所Witness**であり、OKF YAML解析・Conformance・SHO Consumer・Source/Meaning/Grantまで認定しない。

**既存Repo Healthへの適合条件**：新テストを、既存`go test ./tools/repository-plan-check`と`go test ./...`で実際に走らせ、原ファイル改変によるDigest不一致が**そのテスト自身でFAIL**することを確認する。Gofmt/Vet、他のPlan Link検査と別に受入。CI全体の赤だけでは本テストが実行されたか/Mutantを殺したか判別しない。独立Expectedの正式採用は#2514のOwner判断が要る。今回のGoテスト実行・Mutant kill・最新HEADのRepository Health CI結果は確認前のため**NOT RUN / UNQUALIFIED**と記載する。

### 13.3 既存mainのDesign Context・CLI実績を先行利用する

[main統合 #3354](https://github.com/hat47x/tei-platform/pull/3354) が、既存単一`tools/reference-issue-test-ability/context_gate.py`へ**11種類の実read-only CLI Reply＋32合成Consumer Controls**をまとめ、[PR exact CI #42391707962](https://github.com/hat47x/tei-platform/actions/runs/42391707962)で限定PASS。3 Required Validation RuleのSource Index/Loader/Role/PathとRule Canonical/Relationを狭いScopeで対照する。初期9 Reply/16負例は歴史的ベースラインとし、二つのRunnerを並存させない。

**ただしK1/K2は以下を混同しない**：
- **実Read対象**：TEI [#3346](https://github.com/hat47x/tei-platform/issues/3346)のドッグフードではdocs/plan MarkdownがmanagedFilesに列挙されても`readByLoader:false`。K1のOKF概念もファイル存在・Markdown Reader受領でTEI Meaningへ自動登録されない。選択された`*.tei.md`の正常Parserは#2621の別Owner。
- **版と範囲**：[TEI #3345](https://github.com/hat47x/tei-platform/issues/3345)のview namespace局所FingerprintとWorld/Source全体Fingerprintを横並びで「同じHash」とみなさない。別Scopeは比較適格がない。Source Root/manifest、実Consuming Reader、Parser/Provider、Current Consumerの要求Scopeを別に記録する。
- **正常応答の異形**：[TEI #3347](https://github.com/hat47x/tei-platform/issues/3347)と[Draft #3370](https://github.com/hat47x/tei-platform/pull/3370)は、requirementsの正常応答`result: []`がObjectでなくArrayであることを明示する。JSON parse/正常空配列/必要Rule未取得/形式不正/Tool Reply終端欠落は各々別事象である。全Commandに同じ`schemaVersion/resultShape/mode`を推定しない。#3370の改善は正式に統合されるまでDraft資格。
- **局所肯定と不在証明**：[Draft #3365](https://github.com/hat47x/tei-platform/pull/3365)のTT-105/106は部分Scenario目録でも受領済Stepの局所観測を保持するが、未表示Stepを『未テスト確定』としない。OKF indexが部分的でも、**実受領したConceptのraw hashは証拠**にできる一方、未表示Conceptを不存在、Required全件取得済み、全公開可能と主張しない。全体Coverageと未計測のSinkはunknown。
- **Plugin初期化資格**：[main統合 #3338](https://github.com/hat47x/tei-platform/pull/3338)のPlugin-first設計はProvider Hook/Operation宣言、Invoke Variant/Effect、Sink、Itemの複数Occurrenceを独立に要求。K1は現在のRaw/Consumerテスト、K2は#1251/#951のProvider/Lock/実二call-site、K4はTAI #514のTyped Bind/Receiptへ。Plugin宣言がある、合成Packageがある、Go binaryに同梱された、実呼出されたを別に判定する。

### 13.4 K2へ渡す限定的な結果モデル（新公開Schemaなし）

| 入力の実例 | その時点で肯定できる局所的な事実 | 禁止する不当な否定・昇格 |
| --- | --- | --- |
| Required=3、CLIで1 Conceptを実受信 | 当該1 Conceptは実Read・raw hashを持つ | 残る2を不存在/未許可/未テスト確定、全受信complete |
| 正常`result: []` | 応答の配列は正常に受領 | 必要Source 0件証明、JSON破損扱い |
| managed .md、readByLoader:false | 管理Sourceとして発見済み、Semantic Loaderは未読 | TEI Meaning accepted、業務Roleの不存在 |
| 同じ本文、別Canvas/IslandやSource Root | 公開Markdown本文が同じ | 元World/Source identity同一、独立根拠件数倍増 |
| Index欠落／broken linkの一時Mutant | OKF F0では許容される**仕様上の候補** | 実Source取得済み、Domain/品質F1/F2がPASS |
| 正常Raw WitnessのみPASS | 原Byte/SHAとRequiredファイル集合に限定して証拠取得 | YAML parse/F0 validator、実SHO Writer、2 Consumer、Attestation、Host Grant |

ここでの「局所肯定」は対象Scopeが定義された受信結果の事実であり、配布Permissionや業務意味受理と無関係。必要Conceptを独立に定めたうえで、実Read/Parse/Reply到達・欠落、RecipientのDisclosure範囲を評価する。**TEI #3262の既存Context Gateが受け取れないOKF専用情報は、受信できたことにせずunsupported/not-runとして残す**。

**Owner/採否**：#2514は独立Expected・Raw/Mutant判別力、#3262はScope相対Contextと受領、#3149/#3345/#3346/#3347は実CLI Gap、#1251/#951/#3338はPlugin-first、SHO #18/#21はReader/原Byte、SUI #3392/SEI #820は出典・Lossと公開、TAI #514/#271はActivity/Attestationの受け手。#1119の製品Gate/Harness H2/H3を本節の実験で変更しない。**今回の実施範囲はFixtureのリンク修正・既存HealthへTest-only Raw Witness追加・静的Repo/Issue照合**。修正HEADのGo Test/CI、K1製品F0、K2実Plugin二Consumer/Current Scope、独立Oracle正式資格は**未検証**。

---

## 14. 2026-11-08 K1実CI判定とSHO G2-P統合：Raw Reader→Patch候補→保存の段階資格

**この節の変更目的**：前回のK1原文Witnessを実CIで照合し、main統合されたSHO G1b/P0 Handoff・G2-P局所Patchを、OKF交換・TEI設計Contextの将来のConsumerへ**過剰な資格なしで**接続する。§9〜13のK0〜K5/OK-01〜16と既存Ownerを置換せず、Plugin/Reader/Writer/Attesterの第二実装を提案しない。

### 14.1 PR #3329の旧CIはK1 Rawテスト失敗ではない

固定HEAD `62397d07de84e22fc7e694b679ea13896659d048` の [Actions #42397002393](https://github.com/hat47x/tei-platform/actions/runs/42397002393) が `completed/failure` だった理由を実Jobログで特定した。

- `go test -count=1 -timeout 60m ./...` **PASS**、`tei validate -workspace reference/workspace` **PASS**、`go vet ./...` **PASS**。
- 失敗は `gofmt -l . is empty` の対象外ファイル `tools/reference-plugin-capability-coverage/coverage_test.go` の整形差分（1 byte規模）。今回のOKF Fixture/Raw Witnessは旧HEADの全Go testとして実行された限定成果であり、**形式ValidatorやKnown YAML Mutant killではない**。
- 現mainの同ファイルblobは旧PR HEADと異なる修正版になっている。ただし**最新main合成PR HEADのCI再実行を見ずに修正済みPASSとはしない**。失敗原本と修正後の合格証拠を同じCI Resultに合成しない。
- 直前の旧CI [#42392171058](https://github.com/hat47x/tei-platform/actions/runs/42392171058) でのMarkdownリンク2件のFAILは§13.1で修正済み。**異なる2コミットの失敗理由（Link失敗→Gofmt失敗）を混同しない**。

### 14.2 SHO側実装の到達点を更新する

2026-11-08時点の実リポジトリ状態：

| 担当 | main統合・局所実証 | K1/K2/K5としては未資格の領域 |
| --- | --- | --- |
| [SHO #21](https://github.com/hat47x/sho-composer/pull/21) | **merged**。合成354-byte SUI Raw Markdown受領のG1b試験・起動前の偽Checkout拒否。Ubuntu/Windows Jobはsuccess | 実SUI Producerの**同一物理File**をSHO CLIが同一Attemptで読んだH1は通常CIでSKIP。OKF F0ではない |
| [SHO #24](https://github.com/hat47x/sho-composer/pull/24) | **merged**。実 `experiments/g2_byte_patch/plan.cjs` のpure G2-P。Ubuntu/Windowsの[Actions #42397128510](https://github.com/hat47x/sho-composer/actions/runs/42397128510)はいずれもsuccess。実G1b回帰＋**14個のG2-P局所テスト**を実施 | memory候補生成だけ。実Disk Writer、G1bからの正確なUTF-8 span解決、同一Source RevisionでのReader→Planner配線、保存後再Open、CAS/ABA/Owner、OKF F0・全Bundle受領ではない |

G2-Pの入力は`actual sourceBytes/baseSha256/startByte/endByte/expectedBytes/replacementBytes`、出力は`candidateBytes/candidateSha256/changed`と明示された`effect=none-planner-only, sourceRevision=unknown`。明示spanはCallerが別途解決する。G1bのLink line/ordinalを自動的にbyte spanや永続Occurrence Identityと推論しない。新test-only Plannerを正式なOKF Writer/APIへ昇格させない。

### 14.3 次の実験を最短で分割する：K1R→K2C→K5W

**K1R（実Raw Readerの同一Source観測）**：
- 公式Pin済みK1 3 Concept+Indexの4 Fileそれぞれについて、SHO mainの実G1b `sho-inspect`が実バイトを読むこと、Raw SHA-256・Parser版/Grammar scope・受信範囲を観測する。G1bはCommonMarkの`link_open`限定Readerであり、**YAML frontmatter/OKF type/status/sourcesの解析をF0として保証しない**。TEI Raw WitnessのPASSとSHO G1b実Readは**別repo・別実Consumer**。
- Readerごとに `requestedPath/actualRoot/manifest/revision/exact bytes` を確認可能なところだけ記録。未提供なら`unsupported`とし、Scope descriptorから実Source read・owner意味受理を推測しない。
- 固定Bundleと独立ExpectedのSpec reviewを通した後、K1 F0 Parser/Validatorを選択Providerへ接続。正常Bundle、必須type/YAML不備、未知type/key、broken link、bundle-root absolute link、index省略、verified singletonは**F0とF1別列**。旧Repo Link TestはF0判定器にしない。

**K2C（共通Contract二Consumerの実入力と閉包）**：
- TEI `tei commands`等の既存CLI能力とSHO G1b実CLIが別々に原文を読めた、という2件の観測を「同じ選択済みOKF Pluginを使った二Consumer実証」と呼ばない。**Resolved Provider ID/Plugin Lock/実Call-Site/実argv/実Read Content digest/Consumer Delivery Scope/Current Audience**が同一Attemptで追えることを別に検収する（TEI #1251/#951、#3262）。
- [TEI #3346](https://github.com/hat47x/tei-platform/issues/3346)は**closed**。「docs/planをSource inventoryへ置くと `readByLoader:false` とRoot警告が実応答に出る」ことが実確認され、MarkdownをTEI Meaningとして読む開発は**採用せず#2621へ委譲**。Issueを「製品Markdown Meaning Reader完成」と読み替えない。
- [TEI #3354](https://github.com/hat47x/tei-platform/pull/3354)はmain統合済み11実CLI Reply／32合成Controls、既存単一Context Gateの局所限定実証。K2では`managed/loaded/parsed/accepted`、`received/partial`、`source_index mode`、`requirements.result:[]`、`view-specific FP vs World FP`を別軸に保持し、正常JSONや現Source IndexだけでRequired概念の全読取を証明しない。
- K1 F0単体の実行を、Plugin-first Inspector（CLI/Studio）や正式H2/H3の完了までブロックしない。K2の実Plugin/二Consumer claimは同一Contract/Providerの実選択・実呼出しが必要。

**K5W（OKF Writerと保存後検証は後続）**：
- 初回は完全なBundle Writerを作らず、**SHO単独の通常MarkdownのRead→人の局所編集意図→G2-P pure Candidate→保存前Base/Fragment再照合→実Disk CAS Writer→別プロセスReopen/Raw確認**を最小縦切りとする。OKF Concept化/外部Sync/TEI意味逆書戻しを付加しない。
- Write前後のSource identity/Root/Actual bytes、同じContentのA→B→Aと別Root同Bytes、no-op時物理Write0、二重Write/競合拒否、partial effect/Crash、Owner/Grant、BOM/CRLF保持、別Readerの実再開を既存SHO #4/#16/#19/#20で検収。G2-Pのno-op候補`changed=false`をDisk Writer no-opの証拠としない。
- K5でOKF Bundleを保存・再読み取りする段階になっても、同一Item複数Canvas・Topology Loss、非公開Source/Count、文書trust`verified`とTEI Meaning/Host資格、実Attestation Receiptを混ぜない。SUI→SHO H1は引き続き独立試験とする。

### 14.4 資格票を縮小し、重複したGateを設けない

| 証拠 | 到達済みの限定結果 | 次に不足する一個の資格 | 既存Owner |
| --- | --- | --- | --- |
| 上流v0.2 Spec + K1固定Raw | exact SPEC commit・各Raw digest・実Git Readback。旧HEAD Go全体test PASS | 公式F0 YAML Readerの実解析・7変異の独立判別 | TEI #2514、SHO #18 |
| SHO G1b + SUI合成Fixture | read-only Markdown Reader/異OS raw合成候補はmain実装 | SUI実Producer→SHO実CLIへの同一物理File H1 | SHO #21、SUI #3397 |
| SHO pure G2-P | 14局所編集計画テスト、両OS成功 | 実Disk Writer・no-op write0・CAS・再Open | SHO #4/#19/#20 |
| TEI Context Consumer | main統合11実CLI Reply・32合成Controls | 本用途のRequired Function/Business/Relation＋Source actually consumed・外部受領終端 | TEI #3262/#3346/#2621 |
| Plugin Reader/CLI・Studio | Plugin-first計画main統合、D1実二Consumer未達 | 同一選択Provider/Bundleの二Call-Site実Invoke | TEI #1251/#951 |
| K4 Attested Run | 上流OKF仕様とTAI #514の段階設計のみ | typed input→実Provider/Receipt→独立Attester/Result再Read | TAI #514 |

**正式の合否**：実Git/G1b/G2-P/旧CIの局所証拠をそのまま記録し、F0合格・K2同一Provider/二Consumer・K5実Write・Current Grant/Owner受理を**NOT RUN**とする。本節による製品Gate達成・Plugin実行能力・OKF全束到達の新しい主張はない。


---

## 15. 2026-11-08 上流Reference ReaderによるK1実検査と型境界の反例

### 15.1 Parserを複製せず、固定済み上流ReferenceでK1の限定互換を試す

上流 [GoogleCloudPlatform/open-knowledge-format](https://github.com/GoogleCloudPlatform/open-knowledge-format) の **exact commit ad30107c31c06aec8a7d5636e0d1058118604e6f** には、実装例 [src/reference_agent/bundle/document.py](https://github.com/GoogleCloudPlatform/open-knowledge-format/blob/ad30107c31c06aec8a7d5636e0d1058118604e6f/src/reference_agent/bundle/document.py) がある。Git blob **b770a92f33adf9942e2932529308d066430b02a6**。PyYAMLによりYAML frontmatterを解析し、Conceptの必須 type、verified singletonの正規化とmachine/human trust tierを判定する**参考実装**である。仕様文書とReference実装は別のSource/Revisionとして扱う。

本PRに次をtest-onlyで追加した。

- [tools/reference-okf-k1/official_reference_probe.py](../../tools/reference-okf-k1/official_reference_probe.py)：上流Git HEAD・Reference Document Blob・Spec SHA256・PyYAML 6.0.3を確認した後、K1の現実ファイル4件を実raw SHA-256/UTF-8で照合する。上流moduleだけを選択して読み、ADK/BigQuery等のProducer/Host/モデルには接続しない。実Codeからメモリ内で一要因Mutantを生成し、正規Conceptとroot indexを別扱い。実Fileには一切書き込まない。
- [.github/workflows/tei-okf-k1-reference-ci.yml](../../.github/workflows/tei-okf-k1-reference-ci.yml)：Ubuntu上で**公開合成Fixtureのみ**を対象に、2リポジトリの正確なCommitをcheckout。contents:read権限、Secret/外部Dataset/Effect不要。PyYAMLをtest用仮想環境に6.0.3固定し、上流moduleをバイト検証して実行する。CIの結果はテスト用相互運用Witnessであって、製品のParserや隔離環境認定ではない。
- [testdata/okf-k1-v02.expected.json](../../testdata/okf-k1-v02.expected.json)：上流Reader blob/versionと、**第8の一因子Mutant**「非空listをtypeに置く」を追加。従前7件のExpected・原byteピンをそのまま保持。FixtureとExpectedは同じ作成試行のため、**独立Oracleの正式資格は引き続き未獲得**。

### 15.2 上流Reference Readerも完全な仕様Validatorではない

公式OKF v0.2 §4.1ではConcept typeは**短い文字列**。一方、上流実装の OKFDocument.validate は実コードで type の truthinessを検査し、**truthyなYAML listを型違反として拒否しない**。ここには、参照実装と形式仕様の小さいが意味のある差がある。

新Probeはこの反例を**期待通りReferenceは受け入れる／SPECのstring条件では拒否**という二列で出す。正式なOKF Conformanceを期待違反のまま合格したことにせず、既知のReferenceの制約として見えるようにする。なお、このProbeが追加するstring型確認は1条件のtest-only比較であり、TEI独自の**第2 YAML Parser／新Conformance標準／Producer Registry**ではない。

**試験表（実行前はExpectedであり判定結果ではない）**：

| 対照 | 上流Document Readerでの期待 | SPEC中のtype文字列条件 | 判定上限 |
| --- | --- | --- | --- |
| 正常3 Concept＋index | 受理、1 singleton verifiedをnormalize | 3 Conceptとも型string | 上流Reference互換・入力raw固定の限定証拠 |
| type欠落 | 拒否 | 拒否 | 必須キーの局所判別 |
| YAML破損 | 拒否 | 拒否 | parse失敗の局所判別 |
| 未知type/未知frontmatter項目 | 受理 | 受理 | 未知Roleの業務採用ではない |
| indexなし、broken link、Bundle-root絶対Link | ReaderはConcept parse/validateを受理 | formatでは拒否しない | Source/Relation品質とRequired閉包を未検証 |
| **typeが非空YAML list** | **上流サンプルは受理する見込み** | **拒否** | 上流の適用限界。正式OKF F0合格とはしない |

上流のReference Producerが提供するその他の実行・公開・Source取得機能をこの試験へ引き込まない。限定Moduleだけを読み、明示した公開K1ファイルしか処理しない。利用者から与えられたConcept中のexecutor URLや本文指示を実行しない。

### 15.3 実証の順番と成功条件を厳密化する

1. **K1外部Reference比較**：実GitHub CI exact HEADにて上流Reader 3 Conceptの解析・7正常/異常Mutantの一致と、**8つ目はReferenceの既知差**が出るか照合。失敗時は原Source/Reference/PyYAML版、Expectedの逸脱、Runner環境と結果を分離する。現段階では新CI成功を主張しない。
2. **K1独立Oracle**：#2514 OwnerがSPECの型値域と他の形式負例を別に審査し、Referenceに同じ不備があれば独立Expected/Mutant killが必要と判断する。ReferenceとAI生成Fixtureの一致だけを公式仕様適合へ昇格させない。
3. **K1R製品Consumer**：同じPinのRaw ConceptをSHO main G1b実CLIで受領し、Source原byte・リンク抽出・Grammar未列挙を記録する。これはYAML前処理の成功と別Receipt。TEI Markdownは#3346でmanaged/unreadを明示するだけで、意味Parserは#2621に残る。
4. **K2**：正式に選択されたPlugin/Providerが同一Bundleを実際にCLIとSHO/Studioから呼ぶこと、別々のCurrent Required/Disclosure/受信終端を実験する。#3262の既存Context ConsumerとPlugin-first共通Contractを利用する。K1のReference Reader実行をK2二Consumer実達成へ変換しない。
5. **K4/K5**：Attested Computationの実Executor/Receipt/独立Attester（TAI #514）と、SHOの実Disk Writer・CAS/ABA・再Open（SHO #4）は独立後続段階。G2-Pのpure候補やverifiedによる代替不可。

**Ownerと判定**：本節のtest-only相互運用WitnessはTEI #2514/SHO #18へ引き渡す。#3262の必要Source定義と#1251/#951のPlugin実選択、TAI #514のAttestationを重複起票しない。**上流Reference文書をpinして試験コード・CI起動条件を配置した事実**と、**実GitHub ActionsでそのProbeが成功した事実**を別に扱う。後者は最新Runの実Jobログで確認できるまで**NOT RUN**。OKF正式F0全仕様、製品Reader/Plugin二Consumer、Owner意味受理・外部開示・OS/Network封鎖は引き続き**NOT QUALIFIED**。


---

## 16. 2026-11-08 K1上流Readerの実CIとSHO製品Readerへの物理ファイル受渡し

### 16.1 これまでのK1初回実証に対する限定的な結論

[TEI PR #3329固定HEAD 2d4e8508 のCI #42461299002](https://github.com/hat47x/tei-platform/actions/runs/42461299002) は**実Job completed/success**。実ログには次を記録した。

- 上流OKF commit ad30107c31c06aec8a7d5636e0d1058118604e6f / Reference Document git blob b770a92f33adf9942e2932529308d066430b02a6 / SPEC raw SHA256 26aa5da029278939f914e578107242d9607d4f2dc5fe153272b82f9ed1030101 / PyYAML 6.0.3の固定版に対し、3 Conceptとroot indexの実rawファイルを読込。
- 正常3 Conceptの上流Reference parse/validate **PASS**、verified singletonのイベント件数1・synthetic processのmachine-confirmed扱い、原Byte SHAの固定を観測。
- 既存7一因子Mutantは**7 MATCH**。第8Mutantのtype非空YAML listは、上流Reference `OKFDocument.validate()` が**受理**し、SPEC中の文字列条件では**拒否**となる差分を、実ログで**KNOWN_REFERENCE_GAP**として観測。
- これで「K1の**上流実装による限定互換試験を実行した**」と言える。Reference自体が全仕様Validatorではないため、公式OKF F0全適合、独立Owner Oracle適格、全Relation解決、Product Reader、Source/Disclosure/Runtime Grantを推論しない。§15.3の「Probe NOT RUN」はこの**後続CIにより歴史的状態**となったが、記載時点の状態を遡及改ざんしない。

### 16.2 実SHO G1b CLIが同一K1物理ファイルを読めるかの最小P0

この次のSource受渡しには、test-only [tools/reference-okf-k1/sho_g1b_receipt_probe.py](../../tools/reference-okf-k1/sho_g1b_receipt_probe.py)を追加し、既存のK1 GitHub Actionsに**別Job k1-sho-main-raw-reader**を増設した。新たな製品Parser/Runtime/共通Registryは追加していない。

- **Producer側Sourceの定義**：同じTEI Git checkoutの`testdata/okf-k1-v02/`実4 Markdown file。各実FileのSHA-256・bytesを既存Expectedから再照合。Sourceを手作業でSHO fixtureへコピーして「同じ実ファイル」とは呼ばない。
- **Consumer側実装**：SHO mainの固定Git HEAD `fc0127400733bac161b3fc0d556f451991d0a510` に存在する現行`bin/sho-inspect.cjs`、`src/g1b-reader.cjs`、`markdown-it 14.3.3`をtest専用で実呼出し。Node22／npm lockから依存固定。CLI JSONの`source.sha256/byteLength/revision`、Parser grammar scope、`link_open`件数、`coverage`を比較する。
- **否定対照**：不存在のK1 Fileに対してReader CLIが`not-found`/異常終了し、正常なゼロLinkファイルと混同されないことを確認。Grammar外の未定義参照・全Footnote・外部Sourceは`allDocumentReferencesEnumerated=false`等の局所実応答から評価する。G1b ReaderにYAML Concept type/verifiedの理解は要求しない。
- **Host/権限**：既存SHO実CLIで固定TEI公開FixtureをローカルReadするテストであり、既存SOH/TEI Plugin Host、SHO G2 Writer、外部Network/Egressの否定証明やSUI実ProducerのH1とは分ける。実跨製品入力は**TEIのK1物理file→SHO G1b Reader**に限定され、**「共通OKF Providerを用いたTEI CLI＋SHO二Consumer」K2ではない**。

| 個別Claim | 必須Witness | 状態 |
| --- | --- | --- |
| K1公式SPEC/上流サンプルのScope限定相互運用 | pinされた上流moduleの実parse/validate＋八Mutantの内訳 | **PASS**（#42461299002）。非文字列typeのupstream gapを含む |
| K1R SHO Markdown実受信 | TEI Git実ファイルをSHO固定実CLIが実Readしたbyte/Reader結果・missing負例 | 新Job配置。exact HEAD CI成功の実ログを確認するまで**NOT RUN** |
| 形式F0の全仕様適合 | 独立OwnerによるSPEC全条件/未知・悪意入力の判別 | **NOT QUALIFIED** |
| K2二Consumer同一選択Provider | 現World/Manifest/Plugin Lock/2実Call Site＋Recipient Required→Received | **NOT RUN** |
| K5実WriterとK4実Attested Run | 別のCurrent Grant/実Effect/Receipt/独立検査 | **NOT RUN** |

**最小の次工程**：まずSHOの実CLI Jobが固定Markdown4ファイルを読むか赤緑判定し、成功ならK1Rの限定Witnessだけを更新する。そこからK2実Reader契約を設計する際、上流PythonやSHO G1bのParserをTEI Coreへコピーしない。Plugin-first選択・Provider実Load/Invoke、TEIのRequired Sourceが`readByLoader:false`のままか、SHO/StudioのReader実呼出しを#3262/#1251/#951/#2621に分担し、同一Productの別Runnerを増やさない。

**証拠管理**：上流module・参照ファイルの固定Pin、PythonとNodeの試験結果は**test-onlyのGitHub Action Receipt**。本節は正本のOKF仕様や正式なMeaning/Owner/Hostの代替ではない。実H2/H3、正式なProduct間引継ぎ、Disclosure、源資料真正性の未測定は維持する。


### 16.3 公開TEI→非公開SHOの実Read権限断絶とOwned-CIへの変更

2026-11-08の追加試験で、**公開リポジトリのGitHub Actionsのcontents:read権限では、非公開SHO Repoを別Repositoryとしてcheckoutできない**ことを実観測した。[TEI CI #42461670105](https://github.com/hat47x/tei-platform/actions/runs/42461670105)では上流Reference Reader Jobは成功したが、新設SHO G1b Jobはcheckout段階の`repository not found`で**Reader実行前にFAIL**。この赤を「SHO Parser不適合」「K1 Concept不正」「SHO G1bの実受信失敗」とはしない。

**変更した配置**：

1. 公開TEI [tei-okf-k1-reference-ci.yml](../../.github/workflows/tei-okf-k1-reference-ci.yml)から非公開SHO Repoのcheckout Jobを削除。公式公開SPEC・上流Reference・公開合成Fixtureの限定検査だけを維持し、PAT/Secret/クロスRepo権限を追加しない。
2. 非公開SHO側に [Draft PR #27](https://github.com/hat47x/sho-composer/pull/27) を起票し、Private SHOの実CLI `fc0127400733bac161b3fc0d556f451991d0a510` と公開TEI実Fixture/Reader受領Probe `52908f5f4fe95d7049c0beb4d2dd60ee04e80858` を**別々にPin**して同一JobでRead。合成公開Dataのみのread-onlyであり、Raw秘密やSHO Codeを公開TEI repoへ複製しない。
3. 実ファイル毎のByte SHA/長さ、G1b Reader Grammar/coverage、missing File対照を**SHO自身の実CI結果**で検査する。TEI upstream ReferenceのPASSや、PRのGit checkout作成成功だけでSHOのK1R資格を確定しない。

この移動による**権限の実装差は一つ**である。TEI・SHO間の実Plugin/BOM/Host Grantを設計し直すのではなく、GitHub CI RunnerのSourceアクセス権が属する**Repo Ownership**へJobを配置した。SHO Private→TEI Publicの読取は、その限られた公開Fixtureを同一Attemptで受け取るためだけに使う。公開済みK1のSourceを別Audienceへ再配布できる根拠ではない。

**検証資格**：TEI CI #42461670105のcheckout前拒否は実ログで**FAILを観測**したが、非公開Repo Readerの本体は未起動。SHO Draft #27の最新exact HEAD Jobの成功・実CLI Receiptが確認できるまではK1Rを**NOT RUN**とする。K2二Consumer・G2 Writer・SUI H1・Owner Meaning/Disclosure・外部H2/H3は引き続き未達。


### 16.4 Private SHO実Readerの同一物理Source受領を実CIで確認

[SHO Draft PR #27](https://github.com/hat47x/sho-composer/pull/27)の修正HEAD `ec169bf63f5010c3dc5d3389ee4efe2522419775` に対する [実CI #42462172893](https://github.com/hat47x/sho-composer/actions/runs/42462172893) は**completed/success**、実Jobログに`actual_local_sho_reader=PASS`が4ファイル分あり、固定`sha256`と同一Byte長に一致。

| TEI K1現物 → SHO G1b実CLI | Raw Bytes | Scoped `link_open` | 文書内全Reference |
| --- | ---: | ---: | --- |
| index.md | 309 | 3 | UNKNOWN |
| data/order.md | 480 | 0 | UNKNOWN |
| functions/submit.md | 600 | 1 | UNKNOWN |
| business/submit.md | 492 | 1 | UNKNOWN |

不存在の合成PathをCLIが`not-found`/失敗とする**負例もPASS**。いずれも公開TEIの固定Commit `52908f5f4fe95d7049c0beb4d2dd60ee04e80858` にある実物理Fileであり、SHO私有Repo内のコピーFixtureではない。SHO側の実Codeはpin済み`fc0127400733bac161b3fc0d556f451991d0a510`、Parserは`markdown-it 14.3.3`、Source revisionは`unknown`のまま。

**赤→緑の追加反例**：初回[SHO CI #42461984903](https://github.com/hat47x/sho-composer/actions/runs/42461984903)では、実CLIの4 File Receiptは全件出ていたが、CI終了時の`git status --untracked-files=all`が`npm ci`で作る`node_modules`を不正なSource編集と誤認してFAIL。修正後は追跡対象Fileに実差分がないことを`git diff --quiet`と`git diff --cached --quiet`で確認し、同じ実ReadのStepも**全Job success**。`node_modules`の存在を製品Build/BOM/実Runtime意味資格へ転用しない。

**Claim ceilingの更新**：K1Rの**「SHO実G1bが公開TEIのPinされたMarkdown原Byteを直接受領」**は実証済み。従来の「実SHO CLI試行NOT RUN」は履歴状態に変更。これはTEIのK1 Scopeであり、SUI実Producer H1（通常CI SKIP）、SHO G2実Disk Writer、OKFの正式type/frontmatter F0全適合、TEI/SHO同一Plugin二Consumer K2、Owner採用、非公開Source Disclosure、TAI Attested Computationを満たさない。

**次の最小P0**：この実Readの経路を、将来の選択済みOKF Reader/Validator Contractに接続する場合、`Content SHA一致`、`Spec subset解析`、`Required→Received`、`Current Principal/Disclosure`、`Plugin実Invoke`を別票に保つ。TEI/SHOそれぞれで既存Router/Inspector/Readerを使い、第二のパーサ/Host/Registry/Policy Engineを追加しない。PR #27もDraftのまま、Ownerの正式受理を推論しない。


---

## 17. 2026-11-08 K2を既存Plugin-first／Source受領契約へ接続する実装計画

### 17.1 現mainとDraftの再照合（履歴ではなく現在の実装基線）

- TEI PR #3338（実行Offer・Function Binding・変更影響）は**mainへマージ済み**。ただし、横断計画の統合はOKF Providerの実選択・Load・Invoke・受領を自動的には実装しない。Plugin-firstをK2の入口とし、OKF専用の第二Registry／Host／Policy Engineを作らない。
- TEI PR #3354（TT-56の3 Rule/Canon/Relation/Sourceを実CLIゲートで照合）とPR #3370（requirementsのresult配列をDiscoveryに明示）も**mainへマージ済み**。これらはK2で「正常な空result」と「Sourceの未取得」を区別する既存Consumer基線に使えるが、K1のMarkdownがTEI Loaderで意味読取済みとは言えない。
- SHO Draft PR #25（G2-W1 Working Copyの保存・別Open再読取）は**未マージ**。K1RでSHOのread-only実CLIが受け取ったRawと、Working Copy／正式Writer／競合処理の成果を分離する。SHO Draft #27の実CIはK1R限定PASSであり、SHOのOKF意味ParserやK2共通Providerを証明しない。
- SUIのItem/Card/Canvas/Occurrenceは独立Identity。投影後に欠けたSource・Relation・Ownerの意味をMarkdownから逆推定しない。OKFの未知typeを未分類Itemとして保管できても、意味Roleの確定にはならない。

### 17.2 K2の最小Contractと一つの成功経路

**K2の実験単位は「一つのProviderが実際に選択・起動され、二つのConsumerが同じ固定Sourceを別々に受け取ること」**。Bundleファイル一致だけではK2成功としない。既存のPlugin-first Capability/Provider BindingとTEI Context Query/SHO G1b Readerを使い、次の観測を一つのAttemptに束縛する。

| 境界 | 必須の実観測 | 取り違え禁止 |
| --- | --- | --- |
| Source | Rootの実realpath、ファイル原Byte SHA、対象Scope、取得時刻、Spec/Parser Pin | Bundle内のindexやリンクだけで全Sourceを列挙したとしない |
| Binding | Capability要求、選択されたProvider Identity/Version、Plugin Lock、曖昧・未選択・利用不可の理由 | Offer宣言、Plugin BOM列挙、選択済み、実Loadを同一視しない |
| Execution | Providerの実LoadとInvoke、実argv/operation/effect、現在のHost Grant、当該AttemptのReceipt | `effect:read-only`の宣言や前回のReceiptで実作用を認定しない |
| Consumer A（TEI CLI） | 既存read-only Queryの実Tool Call/Reply、Required Source/Role/RelationとReceived、unsupported/unreadの内訳 | `result:[]`をSource不在・全件取得・意味受理のいずれにも自動変換しない |
| Consumer B（SHO） | 実G1b CLIの原Byte受領、限定Markdown token、未解析のReferenceと実Reader版 | Markdown tokenの受領をOKF YAML・Meaning・Writerの受理へ昇格しない |
| Cross-consumer | 同一World/Source Revision/Provider選択の証明、各Consumerの独立Receipt、必要Source差集合 | 二つのCIがそれぞれPASSしただけで同一Providerの二Consumerとしない |
| Admission | F0形式の適用範囲、F1 Required閉包、F2 Owner Accepted Meaning/Disclosure/Grantの個別状態 | verified/human:、source fingerprint、形式F0合格をF2へ転用しない |

**導入の分割**：K2-Aは合成Bundleを一つの選択済みReader ProviderからTEI CLIに実受領させる。K2-Bは同じProviderをSHOの既存Reader境界から実呼出し、同じSourceの原ByteとConsumer別の限定結果を保存する。K2-Cで独立Requiredを先に固定し、両ConsumerのCurrent Receivedを比較する。K2-Dで一方だけ失敗、Provider曖昧、Source ABA、権限撤回、途中部分応答を試す。**K2-A/B/C/Dのいずれも現時点ではNOT RUN**。

### 17.3 追加する一因子の負例と受入条件

| 負例 | 期待する拒否・未資格 | 既存担当 |
| --- | --- | --- |
| Capabilityを宣言したがProviderが未選択／複数候補 | Binding ambiguous/unselected。自動で最初のProviderを選ばない | Plugin-first既存Issue |
| Plugin LockのProviderと実Loadした実体が異なる | K2-A/Bを拒否。BOMの検証済み組合せを実行済みとは呼ばない | Plugin/BOM既存Issue |
| TEI Context Queryが正常に空配列を返すが独立Requiredに一件不足 | Source欠落／未Loadを明示し、C0/Meaning/二Consumerの完備とはしない | TEI #3262 |
| SHOのMarkdown linkは読めるがTEI LoaderはMarkdownをreadByLoader:false | K1Rは維持、K2のTEI意味受領は拒否 | TEI #2621／SHO #18 |
| 同じSHAでもOwner Meaning受理・Host Grant・Audienceが異なる | Source一致のみ認め、Meaning/Disclosure/Executionを拒否 | TEI #2514／#2515 |
| ReaderがSourceを読んだ後に原文がA→B→Aと変化 | 観測間の履歴を未証明とし、Receiptの時点・入力を区別 | SHO #4／TEI Harness既存Issue |
| SUIのCanvas投影から原Relation/Occurrenceが消える | 受領済み表示と原Meaningの完全性を分離。補完生成しない | SUI既存Projection Issue |
| 旧Session Logやstdoutに残ったRawが別Audienceへ見える | 内部Observerの保有と外部Viewer開示を別Grantで検査 | TEI #2515／TAI #514 |

**最小合格**：同一PinされたProvider/Source/Worldに対して、二つの**実Call Site**とそれぞれの独立受領Receiptが存在し、Required→Receivedの差分が機械的に列挙できること。どれかが未観測なら、K1/K1Rの局所成功は維持したままK2を不合格／未実施とする。実Readerの共通化は既存Plugin ContractへのAdapterとして扱い、OKF固有Hostや第二のデータ正本は導入しない。

### 17.4 既存PR健全性の赤を成果へ混入させない

TEI Draft #3329のHEAD `dece223d233ac38e814dd3e789c58187f78fad2b` に対する [PR健全性CI #42462570959](https://github.com/hat47x/tei-platform/actions/runs/42462570959) は**completed/failure**。実Jobログの失敗は `tools/repository-plan-check/TestMarkdownHasNoEscapedBackticks` が本設計文書553行目で不正なバックスラッシュ付きバッククォートを検出したため。上流Reference Readerの独立CI #42462571037と依存の既知脆弱性CIは成功。**前回の「健全性実行中」を成功へ読み替えない**。今回の文書更新で当該エスケープを除去し、最新HEADの実Go/PR Healthが成功するまで全体PASSを保留する。これをOKF形式やSHO Readerの失敗と誤分類しない。


---

## 18. 2026-11-08 横断計画との収束：K2と実業務Providerを混同しない

### 18.1 二つの縦切りと責務の接続

他レーンで進むTEIのApplication API、Plugin-first Dogfooding、実Provider Read、SHO Writer、SEI Method、KON Dataspaceの設計をK2の受入順序へ接続する。ただし、**OKF Knowledge BundleのConsumer実証と、業務DBのRowset取得は別の試験**であり、一方の成功を他方の実装資格へ昇格させない。

| レーン | 一件目の縦切り | K2への利用可能性と境界 |
| --- | --- | --- |
| OKF K2 | 固定合成Bundle→選択済Reader Provider→TEI CLIとSHOの二Consumer | 同一Provider実Invoke・両Consumer独立Receiptが揃って初めてK2。K1/K1Rの既存PASSを再利用しても自動合格しない |
| TEI Function Read | `order_entry.recent_orders`：Function→Query→DB Binding→実Provider Read→Rowset→標準表現 | Providerの宣言と実Invokeの差を測る共通ドッグフード。ただしOKF ReaderとDB Providerは別Capability・別Source・別Effect |
| Application API | `sales.order.submit`：Source→API→Representation→Provider→Effect→Delivery | 実Query/Providerが未接続なら設計参照の成功だけ。書込みにはOperation Identity/Effect/Receipt/Host Grantが追加で必要 |
| SHO | G1b実Raw Reader→G2-W1 Working Copy→G2-W2 Save As／無変更Save／既存文書条件付更新 | Readerの原Byte受領は確認済み。Writerの正式保存・CAS・別Open・競合はDraft試験を経て別資格 |
| SUI | 実Producerの同一File H1とCanvas/Occurrence投影 | 表示OccurrenceはSource Identityではない。投影Lossを暗黙の意味復元に使わない |
| SEI | 実TEI観測→Recommendation保存・再開→Method追加調査の比較 | Observation/Questionの永続化を既存CLIで実装済みとみなさず、提案・判断と実行を分ける |
| KON Dataspace | 将来の汎用・マルチテナントなSource受皿／Document Database | KONをK2の必須HostやOKF Registryにしない。IdP・権限・論理/物理DB分離はKON固有計画に留める |

### 18.2 K2-Aの最小実装単位を実装責務に落とす

最初の実装は**「既存Plugin Bindingが選んだ一Reader Providerを、TEI CLIから一回実Load・Invokeし、Raw/Parser範囲/Requiredを別々に返す」**に限定する。CLI/StudioとSHOで新しい専用Hostを作らない。

1. **契約固定**：Capability名・Provider ID/Version・選択理由・Source Scope/realpath・Raw SHA・Spec pin・Requiredを一つの合成Attemptとして記録。Sourceの内容一致、Provider版一致、Meaning受理、Host Grantは独立。
2. **既存実装の発見**：TEIの現在のPlugin Selector/CLI Tool/Inspectorが実Load/Invokeをどこまでサポートするか、現mainのコード・テストを確認。未実装なら一つの薄いAdapterだけを加え、Registry/Resolverの重複を作らない。
3. **実Call Site**：CLIの実argv/Provider ID/Invocation ID、read-onlyのHost/OS作用境界、実Tool ResponseとReader Receiptを同じAttemptで保存。静的Offer/BOMの一致だけならFAIL。
4. **独立Expected**：合成K1原文の固定SHA、Reference Readerの既知Gap、Consumerの未解析Referenceを分けて比較。独立ExpectedのOwner受理前はK2のSource/Transport限定Witnessとする。
5. **二Consumer化**：SHOの既存G1b Reader入口を再利用し、同一Providerを実InvokeするAdapterを設ける。二つの別CIが別Providerを呼んだ場合はK2未達。TEI CLIとSHOが同じ原Byteを受け取っても、Current Requiredに不足があれば完備としない。

**実装順序のゲート**：K2-A（実選択・Invoke）→K2-B（SHO実Invoke）→K2-C（Required閉包）→K2-D（失敗・失効）。2 Provider×2 Consumerの4 exact tupleはこの後のP2比較であり、初回K2-Aへ先行必須化しない。独自AI実行ループは、既存Harness＋Tool/Skill＋専門Pluginで足りないことが実測されるまで保留する。

### 18.3 判定表の追加：Source／Binding／Transport／Generation／Authority

| 軸 | この段階で要求するWitness | 明示的に未受理とするもの |
| --- | --- | --- |
| S / Source | Raw SHA、Scope、realpath、Source Index revision、実Loader読取状態 | SHAだけでWorld同一、全Source充足、A→B→Aなしとすること |
| B / Binding | Offer→選択→Plugin Lock→実Load→実Invokeの別々のIdentity | 選択可能22/24などの局所値を実Invoke成功へ転用 |
| T / Transport | CLI/SHOそれぞれの実Request/Reply、raw/stdout/stderr/part/end、欠落理由 | 正常空result、部分応答、CI別Attemptを完全配送とみなすこと |
| G / Generation | 既知・未知Ruleの独立Expected、反例、未探索診断 | テストが存在する／コマンドが成功しただけで全必須テストPASS |
| A / Authority | Owner Accepted Meaning、Current Principal/Host Grant、Disclosure、実Effect/Receipt | Verified human欄、認証成功、read-only宣言、前回Receiptを許可へ昇格 |

TEI設計情報の既存実CLI 13応答／Consumer判別42件、Context Gate、Source Index revision失効、CLI/MCP参照Adapter一致は**それぞれの固定試験に限った先行証拠**。K2の実Plugin二Consumer、外部Harness H2/H3完全配送、業務DB Provider、Owner受理を証明しない。

### 18.4 停止条件・横断計画への手渡し

- **K2-A開始前**：現mainのProvider選択・実Load・Invokeを実コードで確認。宣言しか存在しなければ実装Gapとして明示し、独自Runtimeを追加しない。
- **K2-B開始前**：SHO G1bの現物Readerを維持し、SourceをTEIの合成Bundleに限定。非公開SHOの権限を公開TEI CIへ移さない。
- **K2-C開始前**：Requiredを独立指定。TEIの`result:[]`やSHOのMarkdownリンク0件を完全取得と解釈しない。
- **K2-D開始前**：選択不一致・Plugin Lock不一致・Source ABA・権限変更・部分配送・Scope違いを一因子ずつ負例化。
- **正式製品導入前**：KON、SHO、SUI、SEI、TAIに共通の新Host/Registry/Policyを設けず、TEI既存契約を明示Adapterで再利用。KONの権限管理・物理DB配置は独立した製品判断を尊重。

**状態**：本節は既存実証を用いた計画更新であり、K2-A/B/C/D、業務DB実Provider Read、2 Provider×2 Consumer、外部H2/H3、正式Owner受理を実行した記録ではない。


---

## 19. 2026-11-08 K2の選択Provider実呼出し・改訂鮮度・共通APIへの接続

### 19.1 新たに確認した実装基線

TEIの横断レーンを再照合した結果、[PR #3482](https://github.com/hat47x/tei-platform/pull/3482)（独立RequiredをCapability/Application/Sourceに結ぶ設計Context検証）と[PR #3439](https://github.com/hat47x/tei-platform/pull/3439)（CLI/MCP/API/Web/Desktop共通アプリケーションAPIのADR-0048）は**mainへマージ済み**。これらはK2のConsumerと入口の再利用基線であるが、選択Providerの実Load・実Invoke・OKF意味解釈を証明しない。

[PR #3479](https://github.com/hat47x/tei-platform/pull/3479)（外部Provider SourceとManifestを独立改訂要因として扱うGo負例）、[PR #3378](https://github.com/hat47x/tei-platform/pull/3378)（CLI/Studio Inspectorの限定Parity）、[PR #3371](https://github.com/hat47x/tei-platform/pull/3371)（Studio Host Inspector選択Service）は**Draft・未マージ**。Draft内の局所テストと、現在mainの正式な実装資格を混同しない。

SHOのG2-W1実Working CopyとG2-W2のSave As／無変更Save／既存更新は別工程。W2-E1で条件付き更新を保証できない場合、原本を上書きせずDraftを保持する。K1Rのread-only受領をWriterの権限に昇格させない。

### 19.2 K2-Aで最初に証明するべき「選択＝実呼出し」

K2-Aの初回受入は、Providerを一つだけ用意して成功させるテストでは不十分。**A/Bの2候補を用意し、Bを選択したときに実際にBを呼び、Aを呼んだ場合は結果が同じでも失敗**とする。

| 観測 | 最低限記録する値 | 失敗条件 |
| --- | --- | --- |
| 選択 | Capability、候補A/B、選択B、選択根拠、Provider Lock | 未選択・曖昧・B以外 |
| ビルド | Bの実Source digest、Manifest digest、組込Artifact、Toolchain/Generator版 | Lock一致のみ、Source-only/Manifest-only改訂を無視 |
| 実行 | Actual Invoked Provider ID=B、実Module/Entry Point、Invocation ID、Host境界 | Bを選択したのにAを呼ぶ、Call記録なし |
| 受領 | TEI CLIの実Replyと独立Expected、Source Scope/Required差分 | 空resultを完全取得とする、未読MarkdownをMeaningに昇格 |
| 再利用 | World/Model/Artifactの版と実消費Source/Manifest、実行Receiptを別々に記録 | 二つのdigest取得だけで同一Load消費を推論 |

Sourceだけ変更／Manifestだけ変更／両方変更／どちらも変更なしの**2×2改訂行列**を、PR #3479の負例へ接続する。これは実Provider呼出しの実証前段であり、実LoadしたBがどのSource/Manifestを消費したかの同一Attempt証明が次のゲート。単にSourceとManifestのFingerprintを記録しただけではK2-A PASSにしない。

### 19.3 共通APIとCLI/Studio/SHOを同じ意味経路に置く

ADR-0048の共通アプリケーションAPIを入口とし、CLI/Studio/MCPを独立した意味ParserやProvider Selectorにしない。CLI→既存Harness→薄いAdapterを先行し、必要性を実測するまで独自Harnessを追加しない。CLI/StudioのShadow ParityはData/Validation/Relation/Capability/Application/Patternの**限定6対象**について、実選択B・実Call B・同一独立Expectedを照合する後続ゲートであり、現Draftの表示一致だけでは達成しない。

SHOのConsumer Bはprivate repo側のReader/Adapterから同一Providerを呼び、別々の実Receiptを残す。TEIとSHOのCIのGitHub認可境界を尊重し、公開TEIのCIへprivate SHO用のPATやSourceを追加しない。Source受領後のOwner Meaning、外部開示、実Effect、KON Dataspaceの永続化・認可は別資格とする。

### 19.4 計画差分と停止条件

1. **P0（実装探索）**：mainのProvider Selector→Build組込→Actual Invoke経路をコード・テストで確認。未実装の辺を一つだけ特定する。
2. **P1（K2-A）**：選択B→B実組込→B実呼出し→独立Expected。A誤呼出しは同一結果でもFAIL。Source/Manifestの改訂4ケースと実消費証拠を同一Attemptへ束縛。
3. **P1（K2-B/C）**：同一ProviderをSHO実Reader境界から呼び、CLI/SHO別々の受領Receiptと独立Required差分を照合。
4. **P2（K2-D）**：未選択・曖昧・Source-only/Manifest-only・ABA・Grant撤回・途中配送・不一致Providerを負例化。CLI/Studio Parityと2 Provider×2 Consumerの4 tupleはこの段階で評価。
5. **正式採用前**：F0の独立Expected/Owner審査、F1の必要Source閉包、F2のMeaning/Host/Disclosure/Attesterを別々に受理。KONやSHO Writerの実装完了をK2の前提にしない。

**実証状態**：#3482/#3439のmain統合は確認済み。#3479/#3378/#3371はDraft。K2の選択B＝実Invoke B、同一Attemptの実消費Source/Manifest、SHOとの同一Provider二Consumerは**NOT RUN**。この節は横断設計の成熟成果をK2の受入条件へ反映したもので、製品実装済みという宣言ではない。


---

## 20. 2026-11-08 横断収束：K2をライフサイクル・観測可能性・独立証拠に組み込む

### 20.1 K2は独立した「Reader呼出し」ではなく、共通ライフサイクルの一部分

最新の静的／Hybrid／動的生成、横断Plugin、テスト、Source鮮度の検討を踏まえ、OKF Readerを次の既存ライフサイクルへ位置付ける。**各段階はモデル上で表示・追跡できるが、Pluginが安全上の必須検証を省略できない**。

```text
Source (raw bytes, scope, origin, revision)
  → Resolve / Read (realpath, current permission, Loaded Source Set)
  → Structural Observation (parser pin, grammar, evidence)
  → Meaning Candidate (Owner未受理のまま)
  → Provider Selection (capability, chosen ID, lock)
  → Build / Bind (source + manifest + toolchain)
  → Actual Invoke (entrypoint, args, host grant, effect)
  → Consumer Receipt (TEI / SHO independently)
  → Required Closure (independent expected, missing/partial)
  → Review / Acceptance (Owner, scope, audience)
  → Artifact / Publish Approval / External Verification (別資格)
```

Source→Resolve/ReadとProvider Selection→Actual Invokeは互いに依存するが、**前者のraw同一性と後者の実呼出しは別の証拠**。Read-only K1R成功を後続Build/PublishやWriterの権限に昇格しない。Static/Hybrid/Dynamicのいずれもこの段階構造を共有できるが、具体的な実装経路・効果・失敗処理はCapabilityごとに異なる。

### 20.2 K2-A実装に先立つ証拠の最小スキーマ

製品の新規Schemaを作るのではなく、既存のHarness/Plugin/CLI Receiptへ以下の**概念上の必須フィールド**をマッピングする。現在の実API名や実装済みフィールドだと断定しない。

| 分類 | 必須識別・状態 | 単独では証明できないこと |
| --- | --- | --- |
| Source | `source_id`、raw SHA、origin、revision、read timestamp、realpath、loaded set | 全Referenceの取得、Meaning、Grant |
| Selection | `capability_id`、`selected_provider_id`、provider version、lock fingerprint | Provider実Load/実Invoke |
| Build/Bind | 消費したSource digest、Manifest digest、generator/toolchain、artifact digest | 同一SourceをRuntimeが実消費したこと |
| Invocation | `actual_provider_id`、entrypoint、invocation/attempt ID、input binding、effect class | Consumerへの完全配送、Owner許可 |
| Consumer | Consumer ID、actual raw received digest、parser scope、reply/partial/error、required/received差集合 | 未解析部分や別Audienceへの開示資格 |
| Authority | Owner decision、reviewed scope、current host grant、disclosure audience、receipt verifier | 他段階からの暗黙承認 |

**不変条件**：`selected_provider_id == actual_provider_id` を実Call Receiptで判定する。双方の値を静的Configから転記しただけなら不合格。Source-only／Manifest-only改訂を反映した実Build/Loadの版を同一Attemptに結び付ける。異なるAttemptの成功Receiptを合成してK2成功としない。

### 20.3 失効と再利用の行列

| 変化 | Source Observation | Build/Binding | Consumer Receipt | Meaning/Authority |
| --- | --- | --- | --- | --- |
| 原文A→B→A | 再観測。中間履歴はSHA同値から証明不可 | 消費版を再照合 | 旧Receiptを現在受領へ転用不可 | Owner受理範囲・時点を再確認 |
| Manifestのみ変更 | 原文は維持可能 | Provider選択・Load・生成資格を再検証 | 新実Callの要否を判断 | 自動受理しない |
| Provider/Toolchain更新 | 原文は維持可能 | 生成・実Invokeの資格を再検証 | Parser/結果差を確認 | 意味の同一性を推定しない |
| Required Source追加 | 観測済み部分は維持可能 | 依存閉包を再計算 | 未取得を明示 | 完備・意味受理を停止 |
| 権限撤回・Audience変更 | Cacheの存在は保持し得る | Current Grantを再評価 | 新AudienceへRawを送らない | 開示資格を停止 |
| 外部Source unavailable/partial | `unavailable`/`partial`を保持 | 旧Artifact再利用を独立判定 | 欠落を正常空結果にしない | Ownerが未確認のまま許可しない |

Artifact再利用、Build省略、Evidence再利用は別の判断である。Source/Model/Artifact Revisionを一つのFingerprintに畳み込まない。外部参照はfresh/stale/unavailable/partial/not-found/permission-deniedを区別する。

### 20.4 テスト計画への受渡し

- **P0 / 現mainコード探索**：実装されている`selected→loaded→invoked→received`の各辺を実Call Siteで列挙。文書・Offer・CI緑だけの辺は「未実装」と表示。
- **P1 / K2-A**：選択B、A/B両方の実体、実呼出しB、独立Expected、Source-only/Manifest-only変更を一つのAttemptで照合。誤ってAを呼べば同じ出力でもFAIL。
- **P1 / K2-B/C**：SHOのprivate所有CIから同じProviderを実Invokeし、TEI側と別のConsumer Receiptを取得。Requiredを事前固定し、未解析Markdown・部分集合をUNKNOWNとする。
- **P2 / K2-D**：Source ABA、Provider版更新、Required追加、権限撤回、途中失敗、同じラベルの異Identity、未選択/曖昧を一因子負例にする。
- **後続の公開・保存**：静的サイト生成やDocument保存のLifecycleと連携可能だが、K2のread-only合格はPublish Approval、実デプロイ、外部再取得、正式Writer、Owner受理を意味しない。

**現状の確定証拠**：K1上流Reference限定CI、SHO実G1b K1R CI、TEI PR #3329の旧exact HEAD `0d035db63b1410c6cba36a0e4ee98e9e18da7449` に対する全体健全性・脆弱性・OKF専用CIの3件は**completed/success**。ただしmainは当時のブランチより36コミット進み、PRはmergeable=false。旧HEADの緑をmain統合後や本節追加後のCIへ転用しない。K2-A/B/C/Dの実Plugin Invoke、正式F0/Owner、公開・保存は**NOT RUN**。

**情報配置上の境界**：本計画はOKF/TEIの実Provider受入契約のみを記載し、他製品の独立した固有設計・決定を本ファイルへ転記しない。
