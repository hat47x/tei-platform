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
