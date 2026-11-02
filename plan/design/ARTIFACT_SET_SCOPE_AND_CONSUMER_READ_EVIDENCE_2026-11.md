# ArtifactSetの対象閉包とConsumer実読取の検証計画（2026年11月2日）

- **状態**: 設計・実証計画の提案。公開契約・製品Gate・Owner意味受理は未採用。
- **確認した基準**: TEI main `9c9ef4d160640908e06ad13afb6c2b45c664c748`、[Data-first PR #2587](https://github.com/hat47x/tei-platform/pull/2587) exact HEAD `dc2842d241082d6dee89343cfbd9294153d0d403`。PRは未統合であり、mainの製品機能として記述しない。
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
