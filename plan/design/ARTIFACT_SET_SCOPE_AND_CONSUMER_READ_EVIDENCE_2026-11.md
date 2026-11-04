# ArtifactSetの対象閉包とConsumer実読取の検証計画（2026年11月2日）

- **状態**: 設計・実証計画の提案。公開契約・製品Gate・Owner意味受理は未採用。
- **確認した基準**: TEI main `9c9ef4d160640908e06ad13afb6c2b45c664c748`、[Data-first PR #2587](https://github.com/hat47x/tei-platform/pull/2587) exact HEAD `dc2842d241082d6dee89343cfbd9294153d0d403`。照合時点ではPRが未統合だったが、**2026-11-02T14:08:40Zにmainへ統合済み**。以降はそのコミット自体と現在mainの資格を独立確認する（PR #2587統合＝本書のASケース受入ではない）。
- **局所Owner**: [#2482](https://github.com/hat47x/tei-platform/issues/2482)（Artifact/Manifest/CheckSetの入力・内容・集合）、[#2604](https://github.com/hat47x/tei-platform/issues/2604)（複数ファイルの公開・実Consumer読取）。
- **連携Owner**: [#2595](https://github.com/hat47x/tei-platform/issues/2595)（実Producerへ渡したSource）、[#2600](https://github.com/hat47x/tei-platform/issues/2600)（増分再利用）、[#2505](https://github.com/hat47x/tei-platform/issues/2505)／[Draft #2615](https://github.com/hat47x/tei-platform/pull/2615)（Source版間対応）、[#2497](https://github.com/hat47x/tei-platform/issues/2497)（Projection意味保存）、[#2516](https://github.com/hat47x/tei-platform/issues/2516)（Consumer別Claim・開示）、[#267](https://github.com/hat47x/tei-platform/issues/267)（意味受理）。
- **適用範囲**: 人が読む表・機能設計書、Operation Surface、SHO Composerへ渡す設計素材、AI/CUIが参照するread-only説明、外部Targetが使用するArtifact一式。実行の認可やHostの業務効果は各専門契約が担当する。

## 1. 中心命題：内部に整合した「何か」と、必要な「一式」は違う

新しい `internal/artifact` は `Artifact.New/Verify`、`NewManifest/Verify`、`CheckSet` を備える。これらはContent、版、Rule、Unresolved、Manifest登録、Operation Surface等の**既知の内部不整合**を検査する。

しかし、ある利用者・Application・Operation・用途にとって**本来必要だった成果物集合**がそろっているか、選択した外部Semanticが現在資格を満たすか、そして公開された実ファイルをConsumerが読んだかは、別の命題である。

### 現行PR HEADで静的に確認できる差

1. [`workspace.go::RenderWorkspace`](https://github.com/hat47x/tei-platform/blob/dc2842d241082d6dee89343cfbd9294153d0d403/internal/artifact/workspace.go) は指定したApplicationを選び、到達可能なPersistenceごとの表定義、機能定義、Operation Surfaceを生成する。ただし `InputRevision` は `BuildSourceIndex` の `ModelRevision` に由来する。外部選択Semanticの現在資格そのものではない（[#2482 DA-01〜04](https://github.com/hat47x/tei-platform/issues/2482#issuecomment-6572425506)）。
2. [`manifest.go::NewManifest`](https://github.com/hat47x/tei-platform/blob/dc2842d241082d6dee89343cfbd9294153d0d403/internal/artifact/manifest.go) は全Artifactの `InputRevision` 一致と `Verify` を検査するが、**Application、選択Scope、期待されるKind/Subjectの網羅**を引数に取らない。 `ManifestEntry` は `Kind/Rule/Version/Fingerprint/Digest/Unresolved` で、Applicationや公開パスを持たない。
3. [`consistency.go::CheckSet`](https://github.com/hat47x/tei-platform/blob/dc2842d241082d6dee89343cfbd9294153d0d403/internal/artifact/consistency.go) はManifestに**現に渡されたArtifact**が掲載されているか等を照合する。Generic `CheckSet` は対象Applicationの期待集合から欠落を再計算しない。`RenderWorkspace` はその外側で `checkSurfaceTargets` も使うが、出力Setを後から読むGeneric `CheckSet` に同じWorld/Applicationは渡されない。
4. [`cli_operations_artifacts.go`](https://github.com/hat47x/tei-platform/blob/dc2842d241082d6dee89343cfbd9294153d0d403/cmd/tei/cli_operations_artifacts.go) はContent先頭行からファイル名を作り、順次 `WriteFile` する。Manifest Entryにはその実ファイル名がない。出力先に旧ファイルが残る場合や、読取途中で世代が切り替わる場合の一式性は別検査（[#2604 AP-01〜09](https://github.com/hat47x/tei-platform/issues/2604)）。
5. `CheckSet`のSurface JSON parse失敗、post-manifest改変、SchemaVersionの文字列検索、Gap理由の縮約の具体的不足は[#2482 AC-01〜04](https://github.com/hat47x/tei-platform/issues/2482#issuecomment-6576528616)に別記。本稿はその再実装・試験ID複製を避け、**期待集合の範囲とConsumerの取得資格**に焦点を当てる。

ここで `Manifest.Complete = true` が表すのは、**列挙されたArtifactの `Unresolved` がゼロ**という内部的性質である。対象Applicationの成果物がすべて存在する、JSONが読める、現在のMeaningが受理済み、物理一式公開が終わった、という主張ではない。

## 2. 二つの閉包を使い分ける

### 2.1 Read/Generation Footprint（実際に使った入力）

導出器・Function・Pluginが実際に参照したSource、外部Semantic選択、Mapping、Rule、Generator/Toolchain、Target Profileを、その処理の**宣言された入力**と**実際に消費した入力**として分ける。

これは[#2624 明示Function Data Mapping](https://github.com/hat47x/tei-platform/pull/2624)／[#2619](https://github.com/hat47x/tei-platform/issues/2619)のRead Footprintと相似である。Document用入力の閉包を共通意味正本や第二Resolverへ昇格させる意図ではない。外部Provider意味変更・Source-only改行・Builderだけ変更・Rule更新をそれぞれ別の差分として扱う（[#2482 DA-01〜11](https://github.com/hat47x/tei-platform/issues/2482)）。

### 2.2 Consumer Claim Closure（何を示すために必要か）

Consumerが提出・利用するClaim、対象Application/Subjectと粒度、適用時点、必要なArtifact種別・Logical Subject・外部依存、許容できるpartial/unknown、読取版、開示範囲を定める。**現在存在するArtifactだけを列挙したManifest**から、その期待対象を逆算しない。

例：設計レビューの「Application Aに対応する表定義書・機能定義書・操作面一式がそろった」と、限定された読み取り専用の「表Xの物理列だけ確認した」とでは、要求されるKind・対象範囲・意味Facetが異なる。どちらのClaimにも、同一Source Revisionだけでは必要集合の充足を保証できない。

`ExpectedMembers(C, A, Scope, asOf)`は**設計比較の記法**であり、新Core schema/API/公開Gateの決定ではない。`ActualMembers`、`VerifiedMembers`、`ConsumedMembers`との差を、Claimに必要な範囲だけ説明する。

## 3. 反証可能な最小試験（新規AS-01〜05、すべてNOT RUN）

| ID | 固定した合成入力 | 期待する判別と禁じる飛躍 | Owner |
| --- | --- | --- | --- |
| **AS-01 同版・異Application混在** | 同じ管理Workspace revisionから、Application A用FunctionDefinitionとApplication B用OperationSurfaceを取得し、個々に有効なdigest・Ruleを保ち `NewManifest` する。 | `NewManifest/CheckSet`の内部整合が成功しても**Application Aの一式**とはいえない。Generic Manifestで許容するなら、Application指定のConsumer側で不足・混合を診断する。 | #2482 |
| **AS-02 一部メンバー削除後の再目録** | 必要なTable A/Bを含むSetからTable Bを削除し、残りの正しいArtifactだけで `NewManifest` を再作成する。 | 自己整合するManifestと、期待集合の網羅を区別。 `Manifest.Complete` がtrueでも「Bは不要」と推測しない。 | #2482 |
| **AS-03 外部Meaningだけ変更** | Consumer Sourceは固定、選択ProviderのType.Underlyingを変更（E4eの既存fixture）。 | Source Revision不変と、選択外部Meaning変更を別に観測。必要Facetの説明が欠ければ `unknown/needs-review`。既存DA-01を引用し二重実装しない。 | #2482／#1570 |
| **AS-04 Consumer必要集合の差** | 同一Setについて、物理表だけ読むViewerと、Applicationの全設計書・Operation Surfaceを検収するReviewerを比較。 | 一方のConsumerで十分なArtifactが、他方にも十分とは限らない。用途/開示資格も区別する。 | #2516／#2497 |
| **AS-05 再利用と現資格** | Sourceのみ変更、Providerのみ変更、Generator版のみ変更、同内容へ戻るr1→r2→r3。 | `Output bytes`再利用、`provenance`、`Meaning acceptance`、`current Consumer qualification`を別列にし、過少再生成と過剰失効を測る。既存IB-01〜10／DA-08〜11を再利用する。 | #2600／#2482 |

**試験の設計上の注意**：AS-01/02は、`NewManifest`を汎用Set constructorとして利用すること自体が誤りという命題ではない。必要性がApplication相対のConsumerにある場合、追加の期待集合チェックをどこへ置くかを検証する。現在の `CheckSet` へ固定Kind一覧を強制することは前提としない。実行前の「PASS/FAIL予測」を製品実績として報告しない。

## 4. 同一世代の公開、取得、利用を結ぶ比較票

受渡しの最小比較票を、既存Evidenceから**read-onlyに組み立てる**。単一の新Ledger/Artifact Store/Hostを作る規定ではない。

| 観測 | 既存情報の例 | その観測だけで言えること／残るUnknown |
| --- | --- | --- |
| **Intended** | Consumer Claim、Application、必要なKind/Subject、Target/Scope、Disclosure | 何を必要としているか。利用されたことは未確認 |
| **Resolved** | Consumer管理Source Revision、選択 `ResolvedImport.SemanticDigest`、Current Project Meaning | 選択された意味候補と資格。実消費bytesと人的受理は別 |
| **Produced** | Artifact Kind/Rule/Generator/Content/Digest/Unresolved、exact Producer commit/Build Attempt | 成果物の生成。宣言Generatorラベルと実Producer byteの同一性は別（#2595） |
| **Indexed** | Manifest Digest・Entries・Findingsと**期待Member集合**の照合結果 | 列挙物の内部整合と用途相対の網羅。自動的な業務意味受理ではない |
| **Published** | 公開世代、Artifact Digest↔相対path、実byte、切替/失敗の観測 | 物理一式として公開されたか（#2604）。Digestだけでは実ファイル所在を証明しない |
| **Consumed** | ConsumerのRead Attempt、実読取byte/digest、参照Manifest世代、対象Occurrence | Consumerが実際に読んだ範囲。内部レビューの理解や実利用は未確認 |
| **Delivered** | SHO/MMIで保存した文書Occurrence・納品版、AIへの明示送信/許可、外部Consumerへの配送 | 相手に届けたこと。開示許可・対象・現在性・受領観測を別にする |
| **Interpreted/Used** | 当該Claimに必要なFacet・Scope・時点の説明/型付き結果、レビュー・Independent Oracle | 目的に対する利用の妥当性。Owner受理・実業務Effect・再実行許可は別 |
| **Applied/Observed** | 当該Host/Providerの実Request→Effect→独立Read/Receiptと現在Grant | 実環境の効果・業務完了は専門Ownerが判断 |

**受渡しの基準**：同じdigest、同じファイル名、同じApplication名、同じ「実行成功」だけでは、これらの行を結合したことにしない。SubjectとOccurrence、出典Scope、版/時点、必要な関連パス、実Read Attemptを照合する。欠けている要素は`not-observed`または`unknown`とし、明示拒否`denied`や意味不一致`mismatch`とは区別する。

## 5. 他製品との境界

- **SHO Composer／MMI**：TEI Artifactは設計の確認用のread-only材料。SHOは文書構成、表示依存、引用Occurrence、保存・レビュー・固定納品の品質を扱う。MMI [Draft #753](https://github.com/hat47x/markdown-matrix-injector/pull/753)／[#754](https://github.com/hat47x/markdown-matrix-injector/pull/754) の引継ぎ・納品集合は実製品の保証として未統合。生成ArtifactのManifestだけでSHO実読取・完成を主張しない。
- **AI/CUI／TAI Harness**：Product Skillが説明する「参照可能」「利用可能」は、実Provider Read/Consumer Delivery/開示可否のWitnessを意味しない。TAI [Draft #270](https://github.com/hat47x/tai-pleroma/pull/270)のProduct別Skill＋TAI横断Skillと、[#2516](https://github.com/hat47x/tei-platform/issues/2516)のConsumer別Claimへ接続する。SourceやDesign文中の`role: trusted-script`を実操作権限に変換しない（[#2623](https://github.com/hat47x/tei-platform/pull/2623)）。
- **外部Target／UI**：生成図やJSONを正しく表示できることと、実UIロード版、選択対象、操作Intent、権限、Provider効果は別。#2391/#2598が実UI・Actionを所有。TEI独自Hostの追加は本計画の前提としない。
- **Source／Meaning／Rule**：Source改版のA→B→C候補連鎖は意味同一性ではない（#2505／Draft #2615）。Data構造・構文糖衣・Operator意味・実行資格も別（#2619〜#2623）。同一表示から失われたMeaningを逆生成しない（#2497）。

## 6. 最小の実施順序と証拠

1. **P0 Source-static確認**：mainとPR HEADを固定。上記コード・既存#2482 AC-01〜04／#2604 AP-01〜09／#2600 IB-01〜10との重複を表へ記録。新Issue/新Host/新Registryの追加は不要。
2. **P1 focused Go**：#2482がAS-01/02とAC-01〜04の局所fixtureを検証。まず現行実装の**実際の**`NewManifest/CheckSet`結果を取得し、その後必要な最小Consumer qualificationの実装位置を比較する。Result、期待Member、欠落Memberを独立Oracleで検証。
3. **P2 Publish/Read**：#2604がAP-01〜09で実公開世代・Path/byte・切替・旧ファイル残存を観測。Publishの完全性とReader取得の完全性は別列。Windows/WSL/POSIXの違いを明記。
4. **P3 Consumer比較**：同じ固定ArtifactSetをTEI CLI/AI-CUI向けread-only表示/SHO引用のうち少なくとも二つの異質Consumerへ渡し、**それぞれの期待集合・出典・実Read・情報損失・開示資格**を比較。SHOのG0-0・保存再解決C-01/T2は未成立として保持。
5. **P4 増分評価**：#2600に結果を渡し、false reuse（不足Memberや旧版の再利用）・false invalidation（不要な再生成）の両方を測る。正しさと待ち時間改善を別のOutcomeとする。

### 最低限の試験ログ列（公開APIではない）

`caseID | exact main/head | claim/purpose | consumer | application/subject/grain | source revision | selected external semantic | expected members | produced members | validated members | published generation | consumed member bytes | rule/producer attempt | unresolved/findings | disclosure status | owner decision | actual observer | result | residual`

- `result` は `PASS / FAIL / NOT RUN / UNSUPPORTED`等を**検証した主張に相対的**に扱う。
- `residual` に不明依存、未読取、Scope不一致、Rule未帰属、資格失効を記録する。
- 独立OracleはProducer自身のManifestだけを期待値の正本にしない。必要集合は元のApplication・Consumer要求・対象Mapping等から別に与える。
- 新テストの合成PASSとOwnerによる業務意味受理・実Provider/外部Host/SHO納品のE2Eを混同しない。

## 7. 受入時のOwner判断と非主張

- **Owner判断候補**：期待Member集合をgeneric `Manifest`の必須属性にするか、Application/Consumer相対の外部検収に分けるか。既存`NewManifest`が汎用Setである利点を残す。まずAS-01/02の失敗特性を実行で観察してから決める。
- **本書だけで変更しない基準**：Gate A/B/Cの過去のReady、#1119 G0〜G4、M3/M4/M9、#2484 D-1〜D-3採用・D-4保留、新しいRQ追加、SHOの独立製品性、Target外部Host方針。
- **実行状況**：本書で新たに定義したAS-01〜05、Consumer同版Read/Deliver、外部Meaning資格、Windows/SHO/EKI/TAI E2E、Owner意味受理は**NOT RUN／未採用**。他PRで報告された合成テストPASSを本書の実行結果として再解釈しない。

**次の成果物**は、上記P1/P2の同一固定Revisionでの独立Oracle＋実Go/Reader試験記録である。設計票はその受入責務を明確化するもので、試験成功そのものではない。

## 8. Everything is Data・Function Data Contextからの追加圧力（2026-11-02追補）

**新しい成熟判断**：Artifact集合の「揃っている」は、**要求したLogical Memberの充足**だけでは十分とは限らない。適用先がAI/SHO/外部Consumerなら、`Actual Read Footprint`（何を取り出したか）、`Rule/Query Dependency Footprint`（出力に表示しないが判定に使った項目）、`Result Completeness`、`Disclosure/Egress`（何を誰へ渡したか）も別資格として比較する。一方で通常のローカル設計書閲覧に、実DB Read WitnessやAI開示を一律の必須条件として課すものではない。

### 8.1 他チャットで実証された限定Reference

- [Draft #2624](https://github.com/hat47x/tei-platform/pull/2624)：Function別の明示Data Mappingと標準Role。固定Python合成20例について、未Mappingの秘密Fieldが変わっても限定Projectionへ暗黙混入しないことを確認。ただし実Providerアクセスの最小性は未証明。別の[#2387 QX-25](https://github.com/hat47x/tei-platform/issues/2387)では、出力にない秘密FieldをWHERE/Sort/COUNT/JOIN等が参照し、**見える列が同じでも結果が変わり得る**反例を独立SQLite Fixtureで観測。Rule/Query依存と表示Mappingを同一視しない。
- [Draft #2628](https://github.com/hat47x/tei-platform/pull/2628)：Declared footprint→Realized request→Synthetic Access Witness→Result Slice→Consumer Disclosureの違いを12件合成で比較。Outputが同一でもProviderが不要な秘密列や別Relationを内部で読んだ可能性が残る。12/12は局所Python Referenceであり、実DB/実Providerの物理読取り保証ではない。
- [Draft #2627](https://github.com/hat47x/tei-platform/pull/2627)：同じSemantic Dataに複数Provider/Alias/Scopeが対応する場合、意味上のSubject一致から単一物理経路や同Snapshotを決めない。Provider route/read cohort計32例の局所合成検証。実`DeriveOperationSurface`、Registry/Provider、実DBは別Owner・未検証。
- [Draft #2626](https://github.com/hat47x/tei-platform/pull/2626)：#2516のClaim/Scope/Completeness、混合版、二世界開示を15件合成。対象HEAD `60b9a2e145529d6dd832f5a41c40bc999d4992b7`に対する[focused CI #41442415419](https://github.com/hat47x/tei-platform/actions/runs/41442415419)はSUCCESS。ただしTEI本体/実CLI・SHO/外部HarnessのE2Eではない。
- [MMI Draft #753](https://github.com/hat47x/markdown-matrix-injector/pull/753)／[#754](https://github.com/hat47x/markdown-matrix-injector/pull/754)、[SHO #10](https://github.com/hat47x/sho-composer/issues/10)：取得・文書への出現・表示・レビュー・固定納品と、ローカル読取／外部AI送信を独立資格として扱う。MMIのC-01/T2 G0-0未通過やSHO実製品未成立を、この比較からPASSへ昇格させない。

上記は**別PRの限定合成結果**であり、本PR #2630のAS試験や実TEI Product/Consumer受入のPASSではない。必要なFixtureと観測項目だけ借り、合成Oracle/実Provider/Producerを複製しない。

### 8.2 AS-06〜10：期待集合の充足とRead/Disclosureを分ける（全件NOT RUN）

| ID | 最小の二世界／二Consumer反例 | 守る資格とOwner |
| --- | --- | --- |
| **AS-06 秘密Dataの非干渉** | 同じ公開Subject/Fieldを投影する二世界で、未Mapping秘密Fieldだけを変更。Artifactの公開Content/Manifestの**存在・件数・Kind・出力ファイル名**にも差が漏れていないか。 | 公開観測対象は指定ConsumerのPolicy/Claim相対。Outputが同じでも実Providerが秘密を読まなかった証明にはならない。#2516/#2624/#2628 |
| **AS-07 非表示述語依存** | 出力される列は同じだが、WHERE/COUNT/JOIN/Sortに秘密Fieldが影響するQueryを生成由来の説明へ参照させる。 | Display Mappingの一致を、Rule Dependencyの一致や秘密情報非利用と誤報しない。処理許可と出力開示は別。#2387/#2624/#1178 |
| **AS-08 二Provider・同Subject別実装** | 同一Semantic SubjectをReplica/Primary/API/Archiveの異なる経路が提供。Snapshot token文字列が等しいケースと、同じRow数でもNULL/absent・時点の異なるケース。 | Semantic一致から実Provider・Scope・Snapshot/Row完全性を捏造しない。Artifact生成Inputと実Provider Read Evidenceも別。#2627/#2622/#2506 |
| **AS-09 ローカルReadから外部AI送信** | AI/CUIがローカルでは参照できるArtifactを取得。その一部に非公開なSource locator/関係名、またはActionの存在情報が含まれる。 | `read allowed`、`egress allowed`、`received`、`review accepted`を別判定。拒否時は秘匿対象の存在・件数まで外部から推測できる返答に注意。#2516/#2515/#1178／SHO #10 |
| **AS-10 Manifest同版でもContext混在** | 正常なManifest AとBから、個別には有効なArtifactを取得し、Scope/時点/Provider候補が異なるRead Sliceと併置。 | Artifact byte一致・個別Read成功・同名Snapshotだけで、Consumerの複合Claimを肯定しない。#2516/#2626/#2604 |

**重要な境界**：AS-06の「公開観測が同じ」は、匿名化や秘密漏洩防止の**十分条件全般**を主張するものではない。対象Consumerが観測し得る名前・存在・件数・schema・file path・error categoryを適用Scopeとして決めるための有限Pressureである。非表示の秘密Fieldへ合法的に依存して生成結果が変わる場合もあるため、AS-07の**必要で許可された依存**と、AS-06の**許可外依存の露出**を混同しない。

## 9. PR #2630の受入・実行Evidence管理

初回PR候補のRepository Health、依存脆弱性確認、focused Fixture CIは、それぞれ**実際に実行した対象SHA**とログの最終結論を記録する。一般的な`go test ./...`の成功で、他DraftのPython/Go/Shell Fixtureがすべて動いたと推論しない。Focused CI #2626のように、必要なケースについて**exact HEAD checkout・実スクリプトblob・実行後のclean state**を明示する経路を優先する。

- このPRの`AS-01〜10`は現時点で**検証計画のみ**。#2624・#2626・#2627・#2628の合成PASSやmainの旧Gate Readyを、本PRのE2E達成としない。
- **最初に実証する組み合わせ**：AS-01/02（対象Member）＋#2482 AC-01〜04（内容検証）＋#2604 AP-08/09（公開path/実読取）。続いてAS-06/07/09の公開/外部送信差、AS-08/10のProvider/時点・Scope混合を、別Consumer条件で比較する。
- **失敗時の分類**：`missing-member`、`wrong-application`、`invalid-content`、`unverified-producer`、`mixed-published-generation`、`read-not-observed`、`scope-incomparable`、`disclosure-denied`、`unknown`を、**比較票内の診断候補**として使う。新公開JSON/Schema/Result enumの採用ではない。
- **Ownerの採否判断**：Manifest自身が用途相対のExpectedMemberを含むべきか、必要集合をApplication/Consumerのread-only検収が定めるべきかは、AS-01/02実行後に判断する。局所的なRead/Disclosureの追加は既存#2506/#2622/#2516へ分担し、TEI独自Host・中央Permission/ArtifactStore・全用途共通Gateを前提にしない。

## 10. Go候補実装への試験接合（2026-11-02追補）

### 10.1 合成比較から実`internal/artifact`へ

[#2633](https://github.com/hat47x/tei-platform/pull/2633)に、**PR #2587の統合前HEADから分岐した独立Draft**として`internal/artifact/consumer_scope_characterization_test.go`を追加した。#2587の所有ブランチとmainには変更を加えていない。コード変更はGo testファイル1件（6ケース）のみ。

| Case | 現実装に対するGo Characterization | 既存Ownerの次の判断 |
| --- | --- | --- |
| AS-01 | 異ApplicationのFunctionDefinitionを同じSource Revisionで混ぜ、Generic Manifest/CheckSetがApplication Scopeを検査しないことを観測 | #2482：要求Applicationを外部Claimから照合する必要性と配置 |
| AS-02 | 2個の期待Memberから1個だけ残したSetをManifest化し、`Complete=true`が**Expected集合充足ではない**ことを観測 | #2482：ExpectedMember独立Oracle |
| AC-01 | Surface JSONの構文が壊れていても、内容内にschema文字列がある場合の検査経路 | #2482：Decode失敗Finding |
| AC-02 | Manifest作成後にArtifact Contentを改変した場合の`Artifact.Verify`と`CheckSet`の診断差 | #2482：Reader時点での各Artifact Verify |
| AC-03 | Gap Reasonだけを変えてdigestを再生成した場合のUnresolved照合差 | #2482：Gap→Unresolvedの理由と多重度 |
| AC-04 | JSONの`schemaVersion`と別フィールド中のschema文字列が異なる場合 | #2482：実SchemaVersion fieldの検証 |

**現在の資格**：#2633はGo test SourceをGitHubに保存して存在と6関数を再取得確認済み、**Go実行・CI結果の取得前に作成した記録**。PR #2587はその後main統合済み。#2633は2026-11-02、baseを`main`へ変更し、独立Consumer ExpectedMemberを明示するGo test追加コミット `244fccde0bc9ad5a40d4dd203157becd856017b9` を保存した。これにより通常の`pull_request.branches: [main]`対象になり、実Goの結果を新Runから取得する段階へ進んだ（テスト結果は後続の実測で確定する）。最初の試験成功は実候補HEADへのFocused CIまたは信頼可能なGo checkoutの実測が必要。これらのテストは**現行の穴を特徴づけるもので修正後の仕様として承認したものではない**。将来の最小修正時には修正後の検出を期待する正の回帰試験へ更新する。

### 10.2 同Semantic Subjectの複数Mappingを追加圧力にする

[独立Draft #2632](https://github.com/hat47x/tei-platform/pull/2632)は、[#2611](https://github.com/hat47x/tei-platform/issues/2611)／[#2627](https://github.com/hat47x/tei-platform/pull/2627)の多経路合成比較を、PR #2587の**実`internal/operations.DeriveOperationSurface`**に接続した別担当の7ケースGo Characterization候補（初回4件＋Mapping/Storageの変更3件）である。報告された候補Source上の反例は、同じSemantic DataへArchive/Primary 2 Mappingが到達すると第一AliasのMappingだけを選び、後発候補をConflictとして表現しないこと、Alias名の変更だけでOperation Basis/Inputが入れ替わること、選択されなかったMappingだけの変更ではSurface JSONが不変になることなど。#2632のテストは取得時点でGo実行**NOT RUN**であり、独立Oracle／正式Validator／実DB未適合のまま扱う。

**ArtifactSetとの接点**：`RenderWorkspace`が各Persistence Mappingを表定義書へ生成できても、同じWorldから得たOperation Surfaceが**複数候補の意味をすべて説明した**とは限らない。これはAS-01/02の「Member数が存在する」とは別の`Meaning/Route Coverage`である。Generic Artifact CheckSetの成功を、複数Providerの選択理由・Constraint/Gapの正しさ・実CapabilityのBinding・現在Host Grantへ昇格させない。#2611/#2627が経路候補を、#2485がOperation/Actionの選択意味を、#2598が実Request対象を、#1174が実Effectをそれぞれ検証する。

**追加AS-11（予定／NOT RUN）**：同一Semantic Dataに二つの受理可能なMapping/Provider候補を宣言した合成Worldで、(a) `RenderWorkspace`の表定義書一覧、(b) `DeriveOperationSurface`のOperation/Basis/Gap、(c) `Manifest`、(d) ConsumerがClaimする「利用可能な候補経路」を別列比較する。一方のみ表示された場合、全経路網羅のClaimは`unknown/incomplete`とし、単一候補しかないことや実行権限を断定しない。対象Worldが現Validatorで拒否されるなら、拒否条件を証拠として別記し、無理に本番適格Worldと見なさない。

### 10.3 CIの資格

- **PR #2630初回HEAD** `4d48bfada4d2107e9183b9b6b9c96a2cb440ec40`の[Repository Health #41442614894](https://github.com/hat47x/tei-platform/actions/runs/41442614894)はFAIL。原因は新規設計Markdownを`plan/design/README.md`へ登録していなかったことで、`TestPlanReadmesMentionEveryEntry`が報告した。Readme索引は`9febca20ed3529b5786c502c1be1623f9c03fe88`で修正。
- **PR #2630当該HEAD** `3262037b177e0536407148dc5a4b8ff06f7eae86`の[Repository Health #41444279121](https://github.com/hat47x/tei-platform/actions/runs/41444279121)は次のpushでキャンセルされ、後続HEAD `fac313da42736c7b5dda967a8a3fb0fae7a23319`の[Repository Health #41445539469](https://github.com/hat47x/tei-platform/actions/runs/41445539469)と[依存脆弱性確認 #41445539564](https://github.com/hat47x/tei-platform/actions/runs/41445539564)は**いずれもsuccess**。ただしこのRunが検査したのは設計文書・索引であり、#2633の新Goテストを実行した証拠ではない。、[依存脆弱性確認 #41444279151](https://github.com/hat47x/tei-platform/actions/runs/41444279151)はsuccess。CI結果が更新されたらexact HEADと結び付けて判断する。
- #2630は設計文書・索引のみ、#2633は未統合の実コードへのGo Testのみ、#2632は別Ownerの多経路Go Testで、**それぞれ別PR/別HEAD/別実行証拠**である。PR #2630のREADME/Repository Health PASSが後で確認できても、#2633/#2632のGo試験を実行したことにはならない。

## 11. 2026-11-02 14:40Z 更新：Data-first統合後の実装資格と横断受入

### 11.1 基準Revisionの更新

- PR [#2587](https://github.com/hat47x/tei-platform/pull/2587)は2026-11-02T14:08:40Zに**mainへ統合済み**。以後のArtifact/Manifest/Operationsは、main上の実装が比較基準になる。ただしPR #2587の統合が#2482 AC-01〜04/AS-01〜11の修正完了や、#2604の物理公開保証を意味しない。
- 2026-11-02T14:26:07Zの照合main `b1c019fec4f05ca65e3d7491608cb849ce523d29`。mainの`internal/artifact/manifest.go`／`consistency.go`は、#2587の読取時検証上の論点が依然として残る版である。対象のBlob版／Goテストを次段階で比較する。
- [#2633](https://github.com/hat47x/tei-platform/pull/2633)は親PRを変更せず**baseをmainへ変更**。PR差分は`internal/artifact/consumer_scope_characterization_test.go`の1ファイル。変更後の実HEAD `244fccde0bc9ad5a40d4dd203157becd856017b9`では、AS-01/02の期待Member集合を`Manifest`とは別のテストOracleとして保持することを明示した。正式な製品Oracleを新設したわけではない。
- [#2632](https://github.com/hat47x/tei-platform/pull/2632)の多Mapping試験も7件まで増えた。これは別担当のDraftであり、本書のAS-11に必要な`Operation.ID / Basis / Inputs / Mapping Storage / WorldRevision / Provider`差を共有するが、同一Go Runとしては未結合。

### 11.2 実Source読取・Snapshot・Current資格を混同しない

[#267](https://github.com/hat47x/tei-platform/issues/267)に関係するmainの2026-11-02更新では、issuerの署名Source IndexがLoad時点に固定したbytesを用いるよう変更された（main `b1c019fec…`）。この更新は、**別時点に改めて読み取ったSource bytesを、既に検証した入力とみなさない**ための実装上の進展である。

一方、[#1017](https://github.com/hat47x/tei-platform/issues/1017)の[Draft #2646](https://github.com/hat47x/tei-platform/pull/2646)（Workspace直接Read検出）・[Draft #2647](https://github.com/hat47x/tei-platform/pull/2647)（Trace Sourceを固定ルートReaderで読む）は、なお別Ownerの候補。個別ファイルの固定ルート読取でも複数Sourceの同時Snapshot、実ProviderのReader資格、将来のConsumer実取得が成立したとは限らない。

**試験に残す条件（AS-12候補／NOT RUN）**：Load時点の管理Source bytesを固定してArtifactを生成し、その後同一pathのSourceを変更。Source Index/Artifact Set/Published Generation/Consumer actual read/Owner Acceptanceの**どの版が観測されたか**を別列比較する。個別Sourceが同名・同Digestでも、外部Semantic依存やRule/Provider/Host現在Grantの再資格化を自動省略しない。#267/#1017のI/O境界を重複実装せず、#2600 F0/F1/S1/S2へ渡す。

### 11.3 Product実CLIからSHO/AIまでの接続条件

[Draft #2626](https://github.com/hat47x/tei-platform/pull/2626)では、[Focused CI #41462654810](https://github.com/hat47x/tei-platform/actions/runs/41462654810)が**候補HEAD `42b7ec43fa7901359120008105e812eda7e9d6e7`**で、合成55件と実TEI Go CLIのread-only証拠、Source-only移動の前後観測、Human Markdown/AI JSONの同一局所Witnessからの生成・再読取を行い、二つのjobがSUCCESSと報告されている。ただしTEI設計Artifact `tei artifacts`のManifest/実Body一式をSHOや外部Harnessが取得・保存/再開・引用/レビューした実証ではない。

そこで実Consumerの受入を、次の3レベルとして組み合わせる。

- **L0 実TEI単体**：main exact HEADで`tei artifacts`を実行し、必要なMember期待集合（別Oracle）、個別Artifact.Verify/Manifest.Verify/CheckSet、Contentの読取、実ファイルPathとbyte、公開世代を個別に観測。権限判断や外部Consumerは主張しない。
- **L1 異質な二Consumer**：Human設計ビューとAI向けJSON/CUIのそれぞれが、同一Manifest世代から対象Memberの実byteを読み、Claim/Scope/Disclosure/Failureを比較。見た目の同値やJSON生成成功は実SHO受理ではない。
- **L2 SHO／Harness／実Host**：SHOのOccurrence/Review/固定納品、TAI Product Skill/外部送信、Host/Providerの現在資格・実Effectを各Ownerで独立確認する。不存在のAPIや未成熟SHO保存機能を前提にしない。

[#2636](https://github.com/hat47x/tei-platform/pull/2636)／[#2600](https://github.com/hat47x/tei-platform/issues/2600)の増分F0/F1/S1/S2では、**L0で必要Memberを確定し、S1のselective再利用とF1のfull再生成を単純に比較するだけでなく、独立Expected/OracleとReader資格の双方を検査**する。結果一致だけで共通の誤りを見逃さない。時間計測は正しさの確認後。

### 11.4 リリース資格への非昇格

- `main`への製品コード統合、PRの合成CI成功、`Artifact/Manifest Digest`一致、Source移動前後のMeaning不変、同一FunctionのContext合成PASS、Product Skillの説明可能性、SHO文書の存在は**独立した資格**。
- #2633の新試験は現状の不備の特性記述であり、**検出できない挙動がPASSしたことは修正済みと同義ではない**。
- 必要Memberが揃わない・Sourceの実Consumed bytesが未特定・外部Provider/Claimのscopeが不明・公開世代が混在・送信権限がない場合、Consumerの`usable`や`approved`を自動的にtrueにしない。

## 12. 2026-11-02 18:40Z 更新：Characterization統合から検査修正と公開byteの実証へ

### 12.1 新しいmain資格：合成反例の固定は修正完了ではない

- main照合 `f84ca05936370e431d4ad27922eca29cbcc7d41b`（2026-11-02T18:38:41Z）。[PR #2633](https://github.com/hat47x/tei-platform/pull/2633)は2026-11-02T18:37:06Zに**mainへ統合済み**、exact head `244fccde0bc9ad5a40d4dd203157becd856017b9`の[Repository Health #41464423243](https://github.com/hat47x/tei-platform/actions/runs/41464423243)、[Self-build #41464423053](https://github.com/hat47x/tei-platform/actions/runs/41464423053)、[Dependency #41464423090](https://github.com/hat47x/tei-platform/actions/runs/41464423090)がsuccess。**この6テストのうちAC-01〜04は検査が通過してしまう欠点を特徴づけていた**ので、成功は修正完了を意味しない。
- [Draft PR #2659](https://github.com/hat47x/tei-platform/pull/2659)をmain `f84ca05936370e431d4ad27922eca29cbcc7d41b`から分岐して作成。変更は`internal/artifact/consistency.go`／そのGoテスト2ファイルだけ。`CheckSet`で個別`Artifact.Verify()`、Operation Surface実JSON parseと`schemaVersion`、Gap→Unresolvedのexact multiset（Subject＋Reason＋Kind/Detail＋多重度）を検査し、AC-01〜04を異常検出の**正の回帰**に変更する。既存構造検査の責務を強化する修正候補であり、Consumer Scope/Source/公開一式の意味契約を変更するものではない。
- #2659の保存HEAD `10ad24748308eb47152050411b52e6edce614311`に対するGitHub Actionsは初回照合時点でqueued（[Repository Health #41503078091](https://github.com/hat47x/tei-platform/actions/runs/41503078091)）。**Go/CI成功・Ownerレビュー・Mergeは未確認**。追加されるFinding Codeを利用Consumerが安全に扱えるか確認を要する。
- [PR #2632](https://github.com/hat47x/tei-platform/pull/2632)は**closed・未merge**であり、7件の多Mapping Characterizationをmainの合格試験に数えない。AS-11の経路選択/Map/Storage型差を示す候補資料として残す。別途実World Validator／Source/Provider/Targetと接続するまで製品保証を主張しない。
- Workspace Source読取の[#1017](https://github.com/hat47x/tei-platform/issues/1017)はclosed、[PR #2646](https://github.com/hat47x/tei-platform/pull/2646)はmainに統合済み。一方、[Draft #2647](https://github.com/hat47x/tei-platform/pull/2647)のTraceの固定ルートReaderや、複数ファイル同時Snapshot・外部Consumerが読んだbyteの証明は別資格。closedだけでArtifact一式の同時Snapshotは保証しない。

### 12.2 AP-10/11：「Artifact Digest」は「保存した本文のDigest」ではない

[Issue #2604追補 AP-10/11](https://github.com/hat47x/tei-platform/issues/2604#issuecomment-6585308926)で、**公開Manifestの内容と保存ファイルの間にある検証不能部分**を特定した。

- `Artifact.Digest`は`Kind, InputRevision, Rule, Generator, Content, Unresolved`のArtifact envelope全体をfingerprintする。
- `ManifestEntry`が保持するのは`Kind, Rule(ID/Version/Fingerprint), Digest, Unresolved(count)`までで、Generator・Unresolved各Reason/Detail・相対Path・Content単独Digestが欠ける。
- 実`tei artifacts -out`は各`.md/.json`へ**Contentだけ**を書き、`manifest.json`には`{manifest,findings}`を書き込む。したがって出力Dirとmanifestだけの外部Readerは、実byteを取得してもその情報だけでは`Artifact.Verify()`の原入力を再構成できない。**Contentの生SHA-256と`Artifact.Digest`を直接比較してはいけない**。
- **AP-10（同Content／別Rule・Generator）**、**AP-11（同Content・同Unresolved件数／別Reason・Detail）**を#2604へ追加。どちらもNOT RUN。#2659のインメモリ検査がPASSしても、`-out`の公開一式がReaderによって完全検証可能になったとは扱わない。

この欠落に対する最低限の対照を、既存のAP-08/09に重ねる。

| 入力・検査 | Readerが見た内容 | 限定的に言えること／不足 |
| --- | --- | --- |
| `tei artifacts`（`-out`なし）の`WorkspaceSet` | Artifact envelope（`Content/Generator/Unresolved`）、Manifest、Findings | Envelopeの`Verify`と`CheckSet`を試みられる。ただし外部公開ファイルや実Host効果は不明 |
| `tei artifacts -out DIR`の出力Dir | 本文ファイル群＋`manifest.json`（Manifest/Findings） | 出力Dirの保存物を個々に読める。しかし各本文ファイルの`Artifact.Digest`再計算に必要な情報が不足 |
| 固定世代の公開・Consumer実読取（#2604候補） | Entry↔相対Path、実Content digest、実読取byte、必要なEnvelope情報、公開世代 | 比較票上の要件。公開API、ディレクトリ形式、保存方式はOwnerの試験後に判断 |
| SHO・AI／Harness Reader | 上記に加え用途相対ExpectedMembers、Claim／Scope、DisclosureとOccurrence | 読めたことと内容を使用・レビュー・外部送出する資格は別 |

### 12.3 次に行う検証の優先順序

1. **P0 CheckSetの欠落修正**：#2659のactual Go testsをGitHub Actions実HEADと同一のcheckoutで検証。AS-01/02が引き続きGeneric Manifestでは検出不能であること、AC-01〜04が今度は検出されることを別列で記録。意図的なGeneric Setの用途を損なわない。
2. **P1 出力物の証拠形式検証**：#2604でAP-08〜11を実`tei artifacts`／`-out`の二経路比較で検証。途中失敗、旧ファイル残留、ContentとEnvelopeの異版、ConsumerのPath/read/verify結果を独立Oracleで判定。ManifestのEntryを取り出しただけで実Fileを取得済みと主張しない。
3. **P2 版と動的依存**：#2582/#2586/#2600（Draft #2636）に、Source/Rule/Predicate/Provider/Read Cohort/Operation Basis/Storage/Content/Consumer Purposeの差を渡す。F1 fullとS1 selectiveの一致だけでなく**独立Expected**を確認し、false reuseとfalse invalidationを分けて測る。Store破損、Receipt欠落、Grant失効を単なるcache missへ丸めない。
4. **P3 二種類のReader**：TEI側で実Human表示・AI JSONの同一出典を検証（#2626の実CLI read-only成功は独立の先行根拠）。SHO #4/#6の実保存→再開→出典再解決、TAI Product Skillの実受信・開示・撤回は**各Ownerの別実験**として扱う。
5. **P4 実Provider／外部Host**：Function ContextのDeclared/Actual Read、複数ProviderのSnapshot、Operation選択と現在Grant/実Effectの関連を検査。設計資料の正常性を実行許可へ昇格しない。

**今回の新しい判断候補**：Manifest/Artifact自体に物理Path・Content digestを足すか、用途限定の公開側read-only目録と固定世代を使うか。Generic Artifactの意味モデルとは混ぜず、#2604のAP-08〜11実行結果を受けてOwnerが選ぶ。未採用の新Core/共通Artifact Store/Host/配布Gateは導入しない。

**資格**：#2633の6件はmainへ統合した実Go Characterization。#2659は修正候補Draft／新CI未確認。AP-08〜11、AS-11/12、実SHO・TAI・外部Providerの統合試験はNOT RUN。この文書自体は正式な製品要求・Gate/Owner意味受理を変更しない。
