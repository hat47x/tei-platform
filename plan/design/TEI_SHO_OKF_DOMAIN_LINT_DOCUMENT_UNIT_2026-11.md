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
