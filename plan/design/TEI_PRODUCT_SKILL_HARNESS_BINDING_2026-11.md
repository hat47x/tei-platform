# TEI Product Skill：既存CLIへの結合と外部Harnessでの利用を検証する設計

- **状態（2026-11-05更新）**：Product Skill文書・CIは[merged #2607](https://github.com/hat47x/tei-platform/pull/2607)でmainに存在。公開Workspaceでの実TEI CLI限定S1は[Actions #41532988114](https://github.com/hat47x/tei-platform/actions/runs/41532988114)で成功。外部HarnessのSkill実load/invoke、H2/H3比較、正式Owner意味受理は未実施。
- **Owner**：[TEI #2605](https://github.com/hat47x/tei-platform/issues/2605)。
- **横断受入**：[TAI #168](https://github.com/hat47x/tai-pleroma/issues/168)、[TAI Draft #270](https://github.com/hat47x/tai-pleroma/pull/270)のSK-01、[TAI #74](https://github.com/hat47x/tai-pleroma/issues/74)の配布。
- **関連**：TEI #1155/#2354（既存CLI）、#2017（AI作業票）、#2516（Context/開示）、#1178（LLM Disclosure）、#2390（TEI非依存の外部Consumer）、#2493（大量並列要求）、#951（Service Contract）。

## 1. 所有権を分ける

`TEI Product Skill`はTEIの読み取り・操作契約の**使い方**を案内する小さなportable手順。機能の正本は各モデル/Source/CLI/API/Service Contractにあり、Skill本文はProduct機能、認可、業務受理、資格結果を生成しない。

- **TEI Product Skill**：モデル・Relation/Sourceの発見、read-only検査、Validatorの診断・未検証の説明。TEI repoに正本を置き、TEI単独で利用できる。
- **TAI横断Skill**：ユーザー目的からTEI/SEI/SUI/SHO/EKIを選び、製品ごとの資格・権限・引継ぎを組む。製品別Skillへの参照で連携し、同一説明を複製しない。
- **Harness Plugin/Adapter**：配布とロード形式・Tool接続の選択。各Harnessの実機能に応じる。Skillの存在とToolの存在・実接続は別。
- **TEIで作った外部配布Provider/Library**：外部ConsumerはTEI CLI/Skill/Studioを必要としない。契約＋実Artifact＋Host Adapter・必要なRuntime条件の評価は#2390。

TAIが無くてもTEIを単独使用でき、TEI Skillが無くても既存CLIを直接使用できる。MCPは必要時の薄いTransport Adapterとし、TEI Pluginの必須ABIや新Registryへ昇格しない。

## 2. 実在のread-only入口に限定する初期Skill

初期候補は2026-11-02のmainを参照。その後[限定CLI CI #41532988114](https://github.com/hat47x/tei-platform/actions/runs/41532988114)で既存read-only入口を実行したが、全Command・全Schema・全Workspaceの包括保証ではない。実環境のコマンドと利用権限を確認する。

| 用途 | 現在の入口 | 注意すべき限界 |
| --- | --- | --- |
| 初回発見 | `go run ./cmd/tei commands` | `tei.cli.discovery/v0`で`selected-read-only-commands`。全Plugin/全Tool列挙ではない |
| 実行版の手掛かり | ビルド済み`tei version -json`、Git/Working Tree | `go run`などではVCS revisionが返らない場合。識別≠公開資格 |
| 要素一覧 | `go run ./cmd/tei model catalog -workspace <path>` | ambiguousなSubjectを勝手に採用しない |
| 対象詳細 | `go run ./cmd/tei inspect -workspace <path> -subject-kind <kind> -subject-id <id>` | 実際のCanonical IDで指定。source indexedの有無と存在を区別 |
| 参照 | `go run ./cmd/tei trace -workspace <path> -scope <scope> -subject-kind <kind> -subject-id <id> -direction both -mode summary` | 直接参照の投影であり、全依存閉包・権限・実Source同時Snapshot保証ではない |
| 検証 | `go run ./cmd/tei validate -workspace <path> -json` | exit 0はWarningを含み、ok=trueでもdiagnosticsを読む。Source/Model fingerprintは意味受理ではない |

CLIは`cmd/tei/cli_contract.go`の静的Command Catalogと`docs/getting-started.md`を参照する。全コマンドの失敗時JSONが統一されているという仮定は置かない（#2354）。`trace`の`-scope`は参照の読取りに用いる文脈識別子であり、組織の業務権限を生まない。

最初のSkill資料は`skills/yi-tei-modeling/SKILL.md`と`references/read-only-model-investigation.md`に限定し、詳細なCLI実装を重複して文書化する代わりに`docs/getting-started.md`を参照する。コマンドが変わった場合は現物を優先してSkill候補を更新・検証する。

## 3. Claimの適格範囲をSkillの手順で維持

`Discovered`（使用可能候補として発見）→`Inspected`（指定ID/Sourceの読取り観測）→`Validated`（対象版に対する静的Validation）→`Qualified`（用途相対の要求・必要証拠を充足）→`Invoked`（実入口から呼出し）→`Observed effect`（Host/外部Providerの独立結果）→`Owner accepted`（正式な意味受理等）は**比較上の異なる主張**。これを製品の固定状態機械や型enumとして新設しない。read-only Skillは主にDiscovered/Inspected/Validatedまでの**実際に観測した範囲**を報告する。

- **Source identity**：Workspaceの所在、Git revision/作業ツリー、Subjectのkind/idとSourceIndexを区別。世界のMeaning/Source fingerprintは元ファイルの完全同時Snapshotや人の承認ではない。
- **結果の範囲**：Catalogのscope、見つかったSubject件数、Traceの直接参照、各結果のsnapshot/revisionが異なれば対応関係を保留する。見えている資料を「世界全体」とみなさない。
- **情報開示**：TEI内の通常の業務・組織Dataは利用できる。特に個人番号などの秘密・管理されるRawがある場合、内部でのread-only、外部Harnessへ渡すProjection、実OS/Host許可を分ける。秘匿Scopeは#1178、#2516、TAI #232に委譲。
- **作用**：Scope、ValidatorのPASS、Skill指示文、MCP annotationは書込み許可でない。write/release/actual Effectは既存Host/Ownerに確認する。
- **外部実現**：Java Library/Wasm/Process Providerの生成・配布・呼出し適合は#2513/#1861/#2390に渡す。TEI Skill利用が外部Consumerの依存に混入しない。

## 4. 反例を事前に固定する（TS-01〜TS-15）

以下のケースは[TEI #2605](https://github.com/hat47x/tei-platform/issues/2605)本文を正本とし、本計画では試験の順序と実行経路だけを定める。新しいテスト正本/検証基準/別CLIは作らない。

### S0 — 静的契約・資料
- `SKILL.md`のfrontmatter `name: yi-tei-modeling`とディレクトリ名の対応、参照pathの実在、実CLIとの記述整合を静的確認。
- TAIの`yi-tai-activity` SkillやMMIの`skills/mmi`をTEI repoへコピーしない。Product専門機能と横断手順を分ける。
- TAI SK-01のH2（Toolのみ）・H3（Product Skill＋Tool）で**同じ情報条件**を採用する。Skill内に検証用Expectedを埋めない。

### S0b — Product Skillの静的契約点検

Python標準ライブラリのみの`skills/yi-tei-modeling/verify_skill.py`と`tests/test_verify_skill.py`を試作する。実行は`python3 skills/yi-tei-modeling/verify_skill.py`、合成試験は`python3 -m unittest discover -s skills/yi-tei-modeling/tests -v`。

対象はSkill名、Markdown相対リンク、既存read-only Command Catalogと文書内CLI例の整合、Traceページ継続用fingerprintの説明。これは**静的検査であり実TEI CLIの起動・Harness接続・業務意味・実Host権限を検査しない**。孤立した合成fixtureでの7件は**試験コードを保存済み**。以前の「ローカルPASS」は独立の実行ログで裏付けられていなかったため、現時点の試験実績としては撤回する。GitHub Actionsで実行し、対象コミットとログを確認できた時点でPASSへ更新する。

**2026-11-03の独立した再実行（7/7 PASS）**：上の古い「ローカルPASS」主張を一度撤回した上で、GitHubの保存Sourceと一致する`verify_skill.py`（git blob `c5e18e7a5a1b881c24b0214611fa111ab4d12a94`）と`tests/test_verify_skill.py`（`fde1599106476955da97d03cb3864279f9fa9fc4`）の**正確な全文**を隔離一時Dirに配置し、双方の`git hash-object`一致を確認した。Python **3.13.5**の`python3 -m unittest discover -s ... -p 'test_*.py' -v`を**実行し、7 tests / OK**。同じ二ファイルの`py_compile`もPASS。この実行は**合成fixtureだけ**であり、TEI実WorkspaceのCLI/実Host/Harness Testではない。新PR HEADのCI・現在checkout全体に対する静的checkは別確認。

### S0c — read-only CLI例の合成Shell演算によるすり抜けを拒否（TS-16〜19）

[TEI #2649](https://github.com/hat47x/tei-platform/pull/2649)で、**CLIのコマンド名や発見Catalogのread-only表示だけでは、一つのShell行全体の副作用は証明できない**ことが明瞭になった。`artifacts -out`の実ファイル書込みだけでなく、`validate`などの安全なread-onlyコマンドの後に`;`や`&&`で書込コマンドを連結する場合、旧`verify_skill.py`は先頭のコマンド名だけを見て静的PASSにしてしまう。これは**初期Skill静的検査器自身の反例**であり、TEI CLI/Hostの脆弱性を断定しない。

この反例を受け、`verify_skill.py`のShell code fence検査を、`shlex`のpunctuation-awareな**限定トークン化**へ修正。読取り用TEIを含む行で`;`、`&&`、`|`、redirect等の複合Shell演算を検出した場合、コード例を**未検証として拒否**する。先頭が`env`等でTEIを包む行も、未検証の呼出し文脈として拒否する。別の実行許可／Trustエンジンを追加するものではない。

| 反例 | 旧検査の危険な推定 | 新検査の期待 |
| --- | --- | --- |
| **TS-16** `go run ./cmd/tei validate ...;go run ./cmd/tei artifacts -out ...` | 先頭validateだけ見てread-only | 複合Shell例として拒否 |
| **TS-17** `tei validate ... && tei artifacts -out ...` | validate名で一致 | 複合Shell例として拒否 |
| **TS-18** `tei validate ... | tee /tmp/...` | CLIはread-onlyでもShell出力をファイル化 | Pipeを含む例として拒否 |
| **TS-19** `env TEI_ENV=1 tei artifacts -out ...` | 行頭が`env`なので検査対象から漏れる | 非先頭TEI呼出しとして拒否 |

`tests/test_verify_skill.py`へ上記4負例を追加した後、別レーンで以下3件も追加され、旧7件＋7件で**14件の合成ケース候補**となった。Pythonの分離した`shlex`判別ロジックを**6入力**で実行し、想定した分類6/6を確認したが、これは保存済みの**14件`unittest`をそのまま走らせた証拠ではない**。その11件と実Checkout全体の`verify_skill.py`は**新exact-head CIによる確認前**とする。

**TS-20〜22：文書中のCLI引数も正確な契約と照合する（追加のOwner受入候補）**：

- **TS-20**：`tei validate`はread-onlyでも、未確認の`-out`フラグが付いていれば文書内のコマンドを検証済みとみなさない。
- **TS-21**：`tei validate ... && echo accepted`のような追加プロセスを伴う例を一つのread-only CLI呼出しに正規化しない。
- **TS-22**：Traceの`-scope`、`-direction`、`-mode`、`-offset`、`-limit`、外側Snapshot用`-expected-fingerprint`など既知の引数を持つ正例は誤拒否しない。

これらに対応して、静的Checkerは**文書内CLI例に限った引数の小さな許可表**を持つ。これは第二のCLI/Capability Registryではなく、正本`cmd/tei`が変われば更新・点検が必要な**検証用の固定例**である。既存CLI実装から引数を自動抽出できたわけではなく、`commands`一覧が完全であるという推定もしない。新11→14件のGitHub保存版Python `unittest` と、実Checkout上での静的Checkerの正否は**新exact-head CI結果を確認するまで未確定**。先に報告した独立Shellの6分類PASSは、追加3件のCLI引数試験を証明しない。

**非保証**：これはMarkdown内の限られた`sh/bash` fenced exampleに対する事故防止の静的点検であって、任意のShell構文の完全解析、実行時Sandbox、OS/RLS/Host Grant、Consumer向け開示判定ではない。例えばShell拡張や間接Commandを全種類形式的に排除する保証はない。外部AgentにこのCheckerのPASSだけをもって任意のCode実行権限を与えない。実作用は既存Host/OS、製品Ownerが定める。

### S0d — 既知Command名とShell全体・flagの資格を分離（TS-20〜22、独立再実行済み）

S0cまでに発見した複合Shellの誤認に加え、**既知read-only Command名に未知のflagを付けても、旧Checkerがコマンド名だけで受理していた**追加反例を確認した。たとえば`go run ./cmd/tei validate -workspace reference/workspace -out /tmp/result`は、通常のValidate入口でない文書上の引数を含むにもかかわらず、旧Checkerで`errors=[]`だった。さらに`validate ... && echo accepted`も、先頭名だけの確認から漏れていた。これらは文書静的検査の不足であり、実Host/CLIが書込みを実行した証拠ではない。

`verify_skill.py`にはread-only手順の**ドキュメント例限定**の`EXAMPLE_FLAGS`照合と、複合Shell演算・不明な呼出し文脈の拒否を追加した。Commandの真のflag定義・Effect・現在権限は既存`cmd/tei`／Host側にあり、この小さなallowlistを第二CLI Catalog／Permission Engineとして公開しない。CLIの引数仕様変更時にはSkill文書の見直しが必要となる。

| 新規回帰Case | 判別 |
| --- | --- |
| **TS-20** 既知`validate`名＋未確認の`-out` | `unverified CLI flag/argument`として拒否 |
| **TS-21** `validate`＋`&& echo` | 複合Shellとして拒否。後続が別TEI Commandでなくても先頭Commandのread-onlyを行全体へ拡大しない |
| **TS-22** 既存`trace`の`-scope/-direction/-mode/-limit/-offset/-expected-fingerprint` | 当該限定手順の正常コマンド例は受理し、過剰な全拒否を防ぐ |

**新たに実行した厳密な局所証拠（2026-11-03）**：GitHub保存版の`verify_skill.py`（Git blob **`d0b1b6ee0fa3f479d01f9e4dff6163f68c872185`**）と`tests/test_verify_skill.py`（Git blob **`0f89ad2dccf836a1e13d2a1475eade1b07f4da3f`**）を隔離作業環境へ再現し、`git hash-object`を双方照合。**Python 3.13.5 / `unittest discover -p 'test_*.py' -v`で14 tests / OK、`py_compile` PASS**。これは旧7件、S0c追加4件、TS-20〜22追加3件を合算した**同じ保存済みコードに対する新規独立実行結果**である。前段S0cで「11件は未実行」と記したのは当該先行時点の状態であり、この14件の実行後の現在状態には適用しない。

**未成立**：14件は合成CLI declaration／文書fixtureの検証であり、実TEI checkout全体の文書整合、CLI binaryのread-only smoke、実Skill load、TAI SK-02の4独立Harness Session、実Provider/OS/外部送出やOwner意味受理は**この14件でNOT RUN**。異なる実行物／Source版・指紋を混在させないS1a、および比較適格性を固定するS2aへ進む。Workflowの結果は更新後exact-head Actionsを観測して記録する。

**TS-11〜15追補**（[Issue #2605](https://github.com/hat47x/tei-platform/issues/2605#issuecomment-6572810341)）：発見時CLIと実呼出しCLIの相違、Trace継続tokenとGraph fingerprintの混同、改行差によるSource byte相違、複数readの異版接合、外部開示範囲を、それぞれ別の負例としてS1/S2で検証する。Consumer向けの証拠接合は#2516 W10へ委ねる。

### S1 — TEI CLIを実動で検証
- 同一checkout/Workspaceに対し、`commands`→`model catalog`→`inspect`→`trace`→`validate -json`の順でread-onlyに実行。
- `exit code`と`ok/diagnostics/error`を別に観測。対象SubjectのID、Source所在とcontent fingerprint、Working Tree、実CLIのバージョンを残す。
- 関連する反例としてambiguous、Source更新直後のstale、Warning付き成功、未対応Subject、存在しないCommand、`trace`のscope偽Authorityを固定。
- 生成AIが成功と報告しても、コマンド未実行ならNot-run。CLIのread-only PASSは外部Host・Product Releaseを保証しない。

### S1a — 同一checkout・単一実行物でCLIをread-only実測する

初回S1を実行可能にするため、`.github/workflows/tei-product-skill-qualification.yml`を候補ブランチに追加した。**当該Workflowの実際のCI結果は対象HEADごとに別途観測する**。手順は次の通り。

1. `python3 -m unittest discover -s skills/yi-tei-modeling/tests`で14件の静的チェッカーの正常/負例を実行し、同じcheckoutの`verify_skill.py`で文書・read-only Command Catalog一致を診断する。文書にある宣言が実CLI実行資格と一致することまでは保証しない。
2. `go build -o "$RUNNER_TEMP/tei" ./cmd/tei`で**一つの実行物**を作る。`commands`、`model catalog`、`inspect semantic-data:sales.order`、`trace semantic-relation:sales.order.customer`、`validate -json`を同じ`reference/workspace`に対して順に実行する。JSONを読み取り、要求される`worldFingerprint`と`sourceIndexFingerprint`の一致を比較する。Outputはrunner一時領域だけに保存し、repo/元Sourceを変更しない。
3. 実mainには統合済み[#2587](https://github.com/hat47x/tei-platform/pull/2587)の`operations/artifacts`入口もあるが、`commands`一覧の`selected-read-only-commands`には含まれない。この**非網羅カタログ**と実dispatch・実権限の違いは[Draft #2649](https://github.com/hat47x/tei-platform/pull/2649)がOwnerで、本初期Skillが新操作を宣伝したことにはしない。`artifacts -out`はローカル書込みなのでこのread-only Smokeへ混ぜない。
4. 静的コマンド参照は**探索候補**であり、実ToolのPath/Argv/Source bytes/OS・Runtimeおよび実際にSkillがinstall/loadされ利用された事実と別。`commands`の一覧成功、CLI exit0、Validation Warning有りはOwnerの意味受理・実行許可・情報外部送出・外部Host効果の証拠ではない。

**実施範囲を過大に主張しない**：S1aのCIが成功しても「単一checkout／単一CLIの選んだread-onlyコマンドと14件検査の成功」が確認できるだけ。並行変更のABA、Source move、検証範囲外のSubjectや全Plugin、Role/Operatorの実依存閉包、実Harnessからのload、現在Grantと開示先の適合、外部Consumer受領は未証明。Source relocationの実CLI証拠は別Ownerの[統合済み#2626](https://github.com/hat47x/tei-platform/pull/2626)を引用し同一Attemptへ合算しない。

**S2への新しい引渡し**：[#2638](https://github.com/hat47x/tei-platform/pull/2638)の合成Role/依存Providerのpin識別（RB-20〜24）と[#2639](https://github.com/hat47x/tei-platform/pull/2639)の実Go Language Pluginは、Skillが実在Toolとして何を説明できるかの**上流証拠候補**。Role解釈・Language/Operator版・Source Reader・Tool Discovery・実CLI/Capability・Skill Load・Consumer Disclosure・Host Grantは独立資格。TAI [#271](https://github.com/hat47x/tai-pleroma/issues/271)のH2/H3/異Harness比較は、同じ実行物・入力/Expected・現在資格で行い、結果値が等しいことから外部受信安全を推定しない。

### S2 — Product Skillの増分価値
- TAI #270 SK-01のH0（Harnessのみ）／H2（Toolのみ）／H3（Tool+TEI Skill）に同一の入力を渡し、作業成功・Source対応・未決保持・正しく拒否した誤操作・Tool呼出回数・渡したtoken/内容・人間確認数を比較。
- Skillの文章にだけ業務正答や未公開Sourceを持たせない。差分の有用性をSkillそのものへ因果帰属できるよう、追加資料・Tool能力・権限をそろえる。
- H4/H5（複数Product＋TAI）とH6（別Harness）はTAI側が評価する。TEI単体の合成fixtureを異Harness本番利用へ転用しない。

### S2a — 実Product Skill価値の比較資格と結果資格を分離（TAI SK-02へ受渡し）

[TAI Draft #270のSK-02初回ペア実験](https://github.com/hat47x/tai-pleroma/blob/docs/yi-harness-skills-20261102/evaluations/SK-02-first-real-product-skill-paired-pilot-2026-11-03.md)を、TEI Product側から実行する前の契約に接続する。**新しい評価Engineや二つめのTask 検証基準をTEIへ複製しない**。TEI #2605はSkill/CLI/Sourceの実挙動と提出可能なEvidenceを担当し、TAIはSession間比較適格性・Harness実ロード・コスト比較を担当する。

**最小の区別**（新Schemaではなく既存票で管理）：

| 何を確認するか | TEI Product側で取得する根拠 | 確認できない場合 |
| --- | --- | --- |
| S0文書整合 | 実Skill root/参照資料のbytes/commit、`verify_skill.py`試験と対象CLI code | 静的checker successを実Tool存在の根拠に使わない |
| S1実CLI | 1 binaryのdigest、実argv/cwd、`commands`の限定scope、Catalogで選んだ非曖昧Subject、Inspect/Trace、Validateのexit・Warning・error | 実Callを観測できない試行はTool利用済とせず、Skill試験に進む前にProduct不足を分類 |
| S1 Source/版整合 | `result.worldFingerprint`/`result.sourceIndexFingerprint`等の**実JSON schemaでの位置**を守り、Checkout/Workspace/path/Trace外側Tokenまで照合。Source移動は別の条件 | 意味一致から引用先・開示先の現在性を自動推定しない |
| S2 H2/H3比較条件 | 同じProduct CLI/Workspace/Task/Permission/開示対象、H2へSkill本文・検証基準が流出していないこと | 不適格なペアを`Skill効果0`と計上しない |
| S2 H3 Skill実Load | Harness側が読み込んだProduct Skillのroot/refs/scripts、実Tool呼出し、Session IDの独立Witness | `Skillを使用した`というAgent自己報告だけならloaded=unknown |
| S2比較の正答 | Product観測から**別に構成した独立検証基準**によるSubject/Source/Warning/Unknown/架空CLI・無断write評価 | Output値の一致だけで外部受理やSkill効果を採用しない |
| S2費用・安全 | Tool呼出回数、時間、Provider token/費用（未観測はunknown）、人の再確認、実External Disclosureの適格範囲 | 実送出の未観測を安全の証明としない |

**最初の比較は H2→H3→H3→H2 の4独立Session**（順序交絡を減らすため）。H2はToolのみ、H3は同じTool＋実際にLoadされた`yi-tei-modeling`。両腕のPrompt/Model build/Tool/Principal/Source/実行権限/検証基準非開示条件を揃える。**forced Skill activationとauto-discoveryは別Group**。先行4Sessionで一般の効果量は決定しない。Tool/Skill/SourceのBlobが異なるなら新Groupに分ける。

[TAI SK-02 test-only適格性チェッカー](https://github.com/hat47x/tai-pleroma/blob/docs/yi-harness-skills-20261102/evaluations/tools/check-sk02-pair-eligibility.cjs)は20件の**架空Trial票の合成自己検査**を持つが、合成`eligible-to-score`はTask正答や本番許可ではない。TEIのS1 Workflow成功は**実Harness SessionのH2/H3比較PASSへ繰り上げない**。

**独立した中止・再計画条件**：Product-only S1が失敗／同版と見なせるSource Witnessが不足→TEI CLI/Source Ownerへ戻す。H3 Loadが確認できない／H2にSkill混入→TAIで比較不適格、Task自体の品質と分けて記録。現在Grant/Disclosureが不一致→外部送信やEffectを正当化しない。P0-a/P0-b等の既存製品優先と、TEI単独利用・TAI任意の原則は維持する。

### S3 — 必要性が実証された拡張
- CLI discoveryに不足が確認された場合、既存Service ContractとCapability Providerからの**派生Projection**を#951/#1155へ提案。第二のRegistry/Tool Contract正本を作らない。
- Function Data Contextや標準Plugin、Java/.NET/Wasm生成、外部Consumer・SHO設計書等は、その実在/適用条件と独立Evidenceが取れた段階で**目的別の参照資料**へ拡張。未採用機能をSkillが実行可能と広告しない。
- Product単独Skillをもとに、TAI配布・Harness Adapterの差分を検証。共同版固定と権利確認はTAI #74。

## 5. 当初の効果仮説と撤回条件

| 仮説 | 観測方法 | 効果不成立の場合 |
| --- | --- | --- |
| 生成AIは対象ID・Source・版を追跡しやすくなる | H2/H3のSubject誤選択、stale混合、Source参照欠落の比較 | Skillを簡素化しCLI docへ統合する |
| 不用意な実装済み判定・権限昇格が減る | TS-04/06/07で不正なPASS、架空Command、情報送出を対照 | 実CLI/Contract上の不足を優先して修正する |
| 独立Product SkillはTAIを追加しなくても有用 | TEI単体でH3を成立させ、H5との差分を見る | 横断Skillで説明し、不要な重複配布を避ける |
| 複数Harness間で手順を移植できる | H6でTool/Capability/Host差を説明できるか | 必要箇所だけHarness Adapterを加える |

**2026-11-02当時の実証状態（後続の正式な観測は§6を参照）**：本計画と試作Skillは採用前の設計・検証候補。元7件の合成Python試験について別実行を報告している一方、S0c追加後の14件・実Checkout全体checker・S1実CLIの現HEAD CIは確認前。TS-01〜22の実製品・実Harness受入、S2/S3・H0〜H6の比較は行っていない。OwnerによるProduct Skill正式採用、実CLI/実Host/別Harness・実OS情報境界の受入は今後。


---

<!-- tei-skill-reconciled-s1-source-completeness-20261105 -->
## 6. 2026-11-05：Product Skillの実証状態を一段だけ更新

[merged TEI #2607](https://github.com/hat47x/tei-platform/pull/2607)でSkill rootと参照文書がmainへ統合済み。[GitHub Actions #41532988114](https://github.com/hat47x/tei-platform/actions/runs/41532988114)のproduct-skill-read-only Jobは成功。対象版 d0de429ae5aac38b7949c2cfe6eacafff059675f では、公開reference/workspaceのsales.orderとsales.order.customerを固定したビルド済TEI CLIによるcommands、model catalog、inspect、trace、validateの限定read-only実行が成立した。**外部AI HarnessがこのSkillをロード・呼び出ししたことにはならない。**

実Source適用の境界：[TEI #2822](https://github.com/hat47x/tei-platform/issues/2822)では空Workspaceでもvalidate.ok=true、標準stdのCatalogが13件ある。[#2823](https://github.com/hat47x/tei-platform/issues/2823)ではmanifestの未知rootやworkspace-files列挙対象が実Loaderで未解析となる。[#2824](https://github.com/hat47x/tei-platform/issues/2824)ではcommands 19件と旧文書の13件、worldの契約対象外が未整合。よってTaskで要求する**利用者定義Subjectのkind/idとそのSource**を実Workspace・版・Catalog・Inspectで照合する。宣言／列挙／Loaded／検証／Scenario Coverage／実Consumer利用を単一の「成功」に昇格させない。Sourceが未読込ならunknownを保持し、不存在や全件検証済みと断定しない。

最初の後続は[TAI #168](https://github.com/hat47x/tai-pleroma/issues/168)／[SK-02 Runbook](https://github.com/hat47x/tai-pleroma/blob/docs/yi-harness-skills-20261102/docs/yi-constellation/first-real-skill-qualification-runbook-2026-11-02.md)の実一Harness四Session。固定した必要User Subject、実Binary、SourceIndex、Tool Profile、独立Expectedを用い、H2 Tool-only二SessionとH3 Product Skill実ロード二Sessionを正答・Warning/Unknown・Token/時間/人手で比較。Skillが有益でなければCLI/Tool-onlyを維持する。別Harnessはこの後のSK-03。

[merged TEI #2850](https://github.com/hat47x/tei-platform/issues/2850)のEffectObserver欠落時の無人実行停止は**書込み・確定後Effectの後続条件付き経路**へ適用し、最初のread-only比較の必須機構にしない。EKI低遅延/マルチモーダル、SHO、MCP Tasks、Role自己記述/DB方言、複数Buildも用途で必要な時に各Ownerで実証する。新Host/Registry/第二試験Runnerは作らない。

**状態**：本節は既存製品事実の反映と受入条件の整理。実CLIの限定S1は成功済。実外部HarnessのSkill load/invoke、H2/H3四Session、二Harness、全Source Closure、外部OS/Host/Provider、独立業務Oracle、Owner受理は**NOT RUN**。
