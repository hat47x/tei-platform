# AGENTS.md

TEI Repositoryの文書構成と管理ルールです。ここを文書配置の正本とします。

## 作業前に読むルールと適用範囲

**`AGENTS.md`はリポジトリルートのこの1件だけを管理する。** 下位ディレクトリに別の`AGENTS.md`を作らず、階層ごとの管理規則も本ファイルに集約する。

- TEIを調査・変更する人間と生成AIエージェントは、チャット・セッション開始時と作業再開時に、**対象branchの最新のルート`AGENTS.md`を読む**。別branchや過去の会話で読んだ内容を最新版と仮定しない。
- 変更対象の`README.md`、Requirement、担当Issue、採用済みADR、対象コードとテスト、および必要なGate資料を確認する。複数領域にまたがる場合は、それぞれの正本を読む。
- `README.md`は利用者・開発者向けの案内、`AGENTS.md`はリポジトリ全域の変更作業規則とする。要求・ADR・実装済み状態の第二正本をここに作らない。
- 正本・ADR・Gate・本ファイルの記述が矛盾するときは、都合のよい規則を選ばず、関連する採用判断と実行条件を確認して整合を取る。解決するまで危険な書込み・統合を止める。
- 新しい資料を置く前に既存の所有ディレクトリと同種文書を探す。領域横断の内容は主要責務の場所に本文を一つだけ置き、他方から参照する。

## ディレクトリ階層と管理責務

これは**新規追加・変更時の振り分け規則**であり、既存ファイルの一括移動指示ではない。特に`plan/`直下の過去の調査・Gate資料や`docs/adr/`の互換リンクは、参照・履歴を守りながら扱う。

| 対象 | 所有する内容と変更規則 |
| --- | --- |
| `/` | `AGENTS.md`（作業規則）、`CONTRIBUTING.md`（参加・検証手順）、`README.md`（入口）、`CLAUDE.md`（AGENTSへの参照）、`go.mod` / `go.sum`（Go module）。ルートに個別機能の設計や作業ログを増やさない。 |
| `product/value/` | 守るProduct Valueの正本。実装都合で暗黙に変更しない。 |
| `product/vision/` | Valueに基づく到達像。Milestoneの進捗と混同しない。 |
| `product/principles/` | 設計判断の原則。個別機能の方式・受入Evidenceを書き込まない。 |
| `requirements/` | Product Requirementの正本は`TEI_PRODUCT_REQUIREMENTS.md`。業務設計要件など他文書との関係を明示し、Issue・実装状態を要求そのものにしない。 |
| `plan/` | Master Plan、Milestone、Acceptance Scenario、Gate条件・Runbook・監査・横断的調査の計画管理。既存の直下にある領域別履歴は一括移動せず、新規の機能別詳細設計は`plan/design/`へ。 |
| `plan/issue/` | 6桁IDのRepository内作業定義。減らす不確実性、責務、完了条件、Evidence、状態を所有する。GitHub Issueとは番号体系が異なる。 |
| `plan/adr/` | 4桁IDの採用判断と理由の正本。採用済み判断は後から書き換えず、判断変更は後続ADRで追跡する。 |
| `plan/design/` | 複数Issue等にまたがる機能単位の設計。全体Architecture・ADR・進捗の第二正本にしない。小規模な設計はIssue内で扱う。 |
| `plan/design/mockups/` | 単独で開ける画面モック等の検討成果。動作するTargetや正式なUI仕様のEvidenceにしない。 |
| `docs/` | 利用・理解のための説明とcurrent Architectureの案内。`docs/status.md`は実装状況の正本。要求・ADR・Gateの判断を重複定義しない。 |
| `docs/conformance/` | 対象と検証範囲を明示したConformance記述。設計準拠の説明と実環境の成功を区別する。 |
| `docs/adr/` | 過去URLの互換案内のみ。ADR本文の正本は`plan/adr/`。 |
| `docs/audits/` | 特定時点の監査記録。過去Evidenceを現在の保証へ読み替えない。 |
| `docs/specification/` | 正本文書から読者向けにまとめた仕様ガイド。個別RequirementsやADRの独立した決定元にしない。 |
| `internal/` | Goのproduction実装。packageの責務・依存方向、Canonical / Provider / runtime authorityの境界を守る。検証器やfixtureを製品依存にしない。 |
| `cmd/tei/` | `tei` CLIの入口と引数・出力の接続。汎用業務意味やProviderの内部実装をCLIへ集約しない。 |
| `tools/` | 検証器、Reference runner、開発補助。`tools/repository-health-check.sh`を統合Gateの入口とし、tool自身のPASSをproduction保証にしない。 |
| `reference/` | 再現可能なReference Workspace・入力・観測fixture・golden・試験用構成。生成物の意味上の元データと照合し、合成fixtureをactual Evidenceへ昇格させない。 |
| `bootstrap/` | 初期構成・materialize・publish等のbootstrap経路。生成先・権限・再実行時の副作用を区別し、通常のReference置場にしない。 |
| `examples/` | 独立した利用例・最小再現例。機能の正本や成功実績の代わりにしない。 |
| `skills/` | エージェント向け補助Skillと付属資産。Repository横断ルール、Product Requirement、採用判断の正本にしない。 |
| `.github/` | GitHub Actionsなどホスティング上の自動化設定。CIが停止中でもtrusted localのRepository Health Gateを代替しない。 |

## 領域ごとの変更・検証規則

以下は上の配置表を適用する際の注意である。既存の文書・データを機械的に移動せず、作業単位で責務・正本・参照先・並行branchへの影響を確認する。

### `plan/` — 計画・Issue・ADR・機能設計

- 着手前に[`plan/README.md`](plan/README.md)、現在のMaster Plan、関連するProduct Value / Requirement、採用済みADRと担当Issueを読む。変更pathが既存PRの所有範囲と重なる場合は先に調整する。
- `plan/`直下はMaster Plan、Milestone、Acceptance Scenario、Gate・監査・Runbook、横断調査を扱う。従来の資料は参照を壊す一括移動をしない。
- `plan/issue/`は6桁のリポジトリ内作業定義であり、GitHub Issue番号とは別である。目的、減らす不確実性、完了条件、Evidence、状態を記録する。`plan/adr/`は4桁の継続的な採用判断と理由の正本であり、採用済み判断の変更は後続ADRとする。
- `plan/design/`は複数Issueにまたがる機能内設計を担当し、進捗・全体Architectureの第二正本としない。小規模な設計はIssueに記す。`plan/design/mockups/`の動作するHTMLなどは検討用であり、正式UI契約や動作するTargetのEvidenceではない。
- 提案・評価待ち・採用・実装・Gate PASSを区別する。Gate/Evidenceはsource revision、Provider、場所、実行条件、未検証範囲を記録し、過去の成功を現在の保証としない。新しい設計への導線は[`plan/design/README.md`](plan/design/README.md)などの既存索引で整える。

### `docs/` — 説明・実装状況・適合範囲

- `docs/architecture.md`と`docs/business-system-design.md`は対象領域の現在の説明基線であり、RequirementやADRを新たに決定しない。`docs/status.md`は現在の実装状況の正本で、提案、設計検証、合成試験、外部実証、製品採用を分ける。
- `docs/conformance/`では対象境界・revision・検証条件・非保証範囲を明示する。`docs/adr/`は旧URLの互換案内に限定し、ADR本文は`plan/adr/`に置く。`docs/audits/`は時点付き監査記録、`docs/specification/`は既存正本からの読者向け説明とする。
- 説明の更新前に対応するIssue、Reference、実際のcommitとEvidenceを照合する。リンク、名称、証拠の鮮度、日本語の読みやすさを確認し、[`docs/documentation-quality.md`](docs/documentation-quality.md)を参照する。

### `internal/`・`cmd/tei/` — Go製品実装とCLI

- `internal/core/`・`subject/`等は基礎表現、`semantic/`は業務上の意味、`syntax/`・`workspace/`は入力構文と読込み、`ir/`は統合表現を主に扱う。`plugin/`はPluginの宣言・解決・構成、`plugin/runtime/`は実行時接続を主に担う。これらの責務は既存実装・ADRと照合し、名前だけから固定しない。
- `database/`・`deployment/`・`driver/`・`realization/`等は実現・Provider・外部資産に関わる。Provider固有の観測や破壊的操作をCanonicalな意味に混入させず、Capability、権限、Lock、runtime authorityの所有境界を確認する。
- 変更するpackageの呼出元・依存先・境界テストを確認し、`cmd/`・`tools/`・`reference/`・`examples/`を製品実装からimportしない。`cmd/tei/`はCLI入出力の接続に徹し、業務意味やProvider内部を集中させない。階層を整えるだけの抽象化を増やさない。
- `tools/repository-architecture-check/`の静的依存検査は一部のpackage境界を対象とするもので、全面的な層分離の保証ではない。依存方向の見直しは[`plan/design/REPOSITORY_ARCHITECTURE_DEPENDENCY_GUARD_2026-10.md`](plan/design/REPOSITORY_ARCHITECTURE_DEPENDENCY_GUARD_2026-10.md)を確認する。
- 変更箇所のunit testと必要なReference / CLI / Provider回帰を用意する。製品コードの`main`統合には後述のtrusted exact-head full Repository Health PASSが必須である。

### `reference/`・`examples/` — 再現可能な入力と検証素材

- `reference/workspace/`は`tei`のReference Workspace authoring source。`workspace/.tei/plugin-lock.json`や`reference/world.json`等の導出goldenを独立の編集正本とせず、入力・生成手順・期待値を合わせて管理する。
- `observed/`の外部観測素材、`legacy-auth/`、`business-system-design/`、`reference-model-transfer/`、`sei-adoption/`、`sui-adoption/`、`ui-customization/`、`plugin-self-build/`等のfixtureは目的ごとの検証範囲に限る。`examples/`は利用例であり、機能の正本や製品実証ではない。
- 失敗例・警告・未解決状態・反例も検証契約の一部であり、見栄えのために消さない。秘密情報をfixtureへ含めない。synthetic / mocked / historical / actual providerを区別する。
- goldenを更新するときは正規の生成経路を使い、`go run ./cmd/tei validate -workspace reference/workspace`や`go run ./tools/reference-golden`等、対象に応じた再現手順と差分を確認する。コード変更を伴えば製品実装と同じ統合Gateを適用する。

### `tools/`・`.github/`・`skills/` — 検証器と開発補助

- `tools/repository-health-check.sh`はコード変更の統合Gateの入口。`--fast`や一部検査の成功はfull PASSでない。`repository-architecture-check/`、`repository-dependency-report/`、`repository-plan-check/`等の結果は対象を限定した診断Evidenceであり、Canonicalモデル自体ではない。
- `tools/reference-*/`と`sei-resume-reference/`等は指定シナリオの実行・検証に限る。`tools/internal/`はtool専用の共通補助であり、製品用の`internal/`と混同しない。
- toolの入力・出力・副作用・停止条件とrevisionを把握し、read-only checkerが素材を書き換えないことを確認する。実Providerを扱う実行は資格情報、隔離環境、外部への影響を調べる。検査弱体化は既存Gate・ADRの合意なしに行わない。
- `.github/`のCI設定や`skills/`のAgent Skillは補助手段であり、GitHub Actionsの停止やSkillの存在はtrusted local Gate、Requirement、ADRの代替にならない。

### `bootstrap/` — 初期構成・再構成と公開

- `final-source/`、`source-safe/`、`payload/`、`materialize/`、`publish/`等では、入力、生成物、公開先が異なる。似た名称だけを理由に処理をまとめない。
- 変更前に入力の正本、生成先、上書き条件、実行権限、再実行時の副作用、失敗時の停止・復旧を確認する。公開・書込みを単なる検査として扱わない。
- bootstrapの一時的なartifactや履歴をRequirementやReference Workspaceの代替正本にしない。実行環境・revision・検証範囲・未実行部分を区別し、コード変更には共通の統合Gateを適用する。

## 文書ルール

- 正本を一つにする。価値は`product/`、要求は`requirements/`、進行は`plan/issue/`、判断は`plan/adr/`、機能設計は`plan/design/`。同じ内容を複数の場所へ複製しない。
- `plan/`直下は計画管理文書だけに限定する。機能ごとの設計文書は`plan/design/`へ置く。小さい機能では新しい設計文書を作らず、Issue本文へ書く。
- ADRは4桁連番、Issueは6桁連番。ファイル名は`0001-short-description.md`形式。
- ADRの状態は `提案中` / `採用` / `置き換え済み` / `却下`。採用済みADRを書き換えず、判断が変わったら新しいADRを追加する。
- Issueの状態は `Proposed` / `Ready` / `In Progress` / `Blocked` / `Done` / `Superseded`。
- 未採用の文書は「評価待ち」と明記し、実装やReference変更の根拠にしない。
- 実装済みの機能と設計検証段階を同じ表現で書かない。実装状況は`docs/status.md`が正本。
- 各作業では、どのProduct Requirementの不確実性を減らすのかを書く。
- Canonicalな意味とEvidenceを混ぜない。syntheticな成功を実Provider・外部資産・本番運用の保証へ昇格させない。
- 文章は日本語を基本とし、Repository上の正式名称・identifier・一般的な技術用語は必要に応じて英語を併記する。英語の語を並べるだけで意味を説明したことにしない。
- 詳細な方式やfieldから書き始めず、原則として「背景 → 防ぎたい失敗 → 守る原則 → 必要能力 → 具体例 → 設計詳細 → 未解決 / 非主張範囲」の順で読めるようにする。
- 箇条書きだけで背景や判断理由が失われる場合は文章で補う。Requirement、ADR、Security境界等では「なぜ必要か」「満たさないと何が起きるか」「何をまだ保証しないか」まで説明する。
- TEIの要求技術・業務知識が高いことを、説明不足の理由にしない。主担当より一段浅い知識の読者でも目的・境界・判断理由をレビューできることを目標にする。
- 専門語・TEI固有語は、読者が意味を推測しなくて済むよう、初出または近い入口で短い日本語説明を添える。
- 抽象境界を説明するときは、可能な範囲でReferenceに沿った具体例、失敗例、因果を示す図を使う。ただし例をRequirementやCanonical meaningへ昇格させない。
- 文書の入口では、役割、正本との関係、対象範囲、現在の状態、非主張範囲を必要に応じて明示する。
- 詳細な執筆・レビュー指針は[`docs/documentation-quality.md`](docs/documentation-quality.md)を参照する。

## 外部資料の帰属と固有名詞

- 外部の研究成果や実装から得た知見は、TEIの要件・設計判断・検証可能な根拠へ翻訳して記述する。原則として個人名を要件、ADR、Issue、実装名、テスト名、見出しやファイル名の主語にしない。
- 著者名の明記は、文献情報、権利表示、正確な出典帰属など必要な箇所へ限定する。出典は失わず、同じ人物への言及を各文書に繰り返し複製しない。
- branch名、commit message、PRのtitle / bodyには、出典の特定に不可欠な場合を除き個人名を記載しない。設計上の変更内容と対象境界を記述する。
- 既に共有したcommit履歴を表記整理だけのためにrebase / force updateしない。Git履歴を書き換える場合は、参照しているbranch・PR・共同作業・外部cloneへの影響を個別に評価し、安全な移行手順を別途定める。

## 開発フロー

- human ownerは単独でも、複数のAI session / processが同時にRepositoryを更新する。**単独process前提で`main`へ直接commitしない。**
- 既存のtask / lane branchがある作業はそのbranchを再利用する。並行作業のたびに新しいremote branchを増やさず、milestone / Gate単位で`main`へPRまたは通常mergeし、その後にbranchを収束させる。
- write前に`git fetch --prune`相当のremote再取得、open PR、対象branchのahead / behind、変更pathの競合を確認する。active PRのheadや独自aheadを持つbranchをcleanup対象にしない。
- 複数sessionが共有するlive branchではrebase / reset / force updateをしない。最新`main`への追随は、必要なら現在のlane tipを第一親にした通常mergeで行う。`main`もforce updateしない。
- **コードを変えたら、`main`へ統合する前に`tools/repository-health-check.sh`を通し、失敗または未実行のコードを統合しない。** gofmt差は`--fix`で自動整形できる（`--fast`で`go test`を省く）。GoがPATHにない環境では、Goのあるtrusted local cloneで実行する。GitHub Actionsが利用できない期間もこのGateを弱めない。
- **code変更を含むDraft PRはmerge authorityではない。** exact-headのfull `tools/repository-health-check.sh` PASSとpre/post clean stateが記録されるまで、Draftのまま保持し、generic merge / local merge / branch-to-main mergeを行わない。PRがmergeableであることやdiffが小さいことは代替Evidenceにならない。
- exact-head / exact-mainにbindingされたqualificationを実行中は、**そのreceiptを無効化する`main`移動を起こさない。** 無関係なdocs-only PRでも`main`を動かすなら、既存branchに保持したままPRをclose/parkし、protected sequenceが安定してから同じPR/branchを再開する。Draftであることだけをmerge防止策にしない。
- **`merged`というGitHub状態それ自体はqualification Evidenceではない。** 必須PASSなしにcode-changing PRがmergeされた場合は、そのmergeを既成事実として次のprotected integrationへ流用せず、incidentを正本へ記録し、protected candidate refを凍結したままcurrent `main` exact headのtrusted full Repository healthを新しいforward stopとする。
- open PRを機械的に一括mergeしない。各PRについてGate authority・exact head・main movement・並行laneを個別に再取得してから統合する。
- containment / qualification / candidateとして明示的に保持されているbranchは、PRがclosed/mergedになっていても正本のGate handoffが完了するまでcleanup対象にしない。特に`ci/reference-pressure-evidence`、`research/selective-knowledge-routing-r6`、`research/tei-ld-0a-corpus-v0`はlive stateとIssue/PRの最新記録を再取得してから動かす。
- 文書だけの変更でも、正本・Gate / Evidence claim・並行branchとの競合を確認してから統合する。
- R7Bのlive qualification branch `research/selective-knowledge-routing-r6` は特別なGate対象である。**generic merge / squash / cherry-pick / source copyでcanonical 11を`main`へ入れない。** `plan/issue/000069-first-web-target-r7b-exact-t-replay.md`に定めるtrusted-local PASS receipt → integration preflight → exact-parent merge preparation → non-force publication → post-merge verificationの順だけをintegration authorityとする。receiptが無い状態でcanonical sourceが`main`に見えてもqualifiedとは扱わず、fail closedでreconcileする。
- 詳細は[`CONTRIBUTING.md`](CONTRIBUTING.md)を参照する。

コードの検証手順も[`CONTRIBUTING.md`](CONTRIBUTING.md)を参照します。
