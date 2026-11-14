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


---

<!-- harness-crosslane-qualification-20261108 -->
## 7. 2026-11-08 横断統合：Skill／実Harness／設計Contextの資格を分ける

### 7.1 計画の旧記録と最新実測の照合

- **実TEI Product CLI**：#2605／merged #2607の公開Source実CLI S1は成立済み。TAIの別CIでも一SubjectのSource Indexと実Source byteを限定照合した。
- **外部DSHでのH3 Skill実読込**：#2509、merged #3113と保存4 SessionではH3-A/BがSkill本文および参照文書を実Tool Readした。Source MarkdownとTool結果の比較ではfrontmatterの除去とLoaderのWrapper付与を別物として扱う。前節の「H3実Load NOT RUN」は歴史的記述であり最新状態ではない。
- **正式比較は未成立**：H2-Aは過去Session再利用。Catalog結果は各8,192byteで途中切断、Tool Request 56／Reply 55、PowerShell stderr混入。比較適格は0/4で、Skillの独立効果採点は未成立。
- **Design Context／Test（2026-11-08更新）**：#3262のmain統合済み試験用Consumerは、初期9応答／16負例から、[merged #3354](https://github.com/hat47x/tei-platform/pull/3354)の**11実CLI応答／32限定Consumer Control**へ進んだ。PRのexact-head [CI #42391707962](https://github.com/hat47x/tei-platform/actions/runs/42391707962) は成功し、実mainのConsumer Blob同一性も照合された。旧#3324/#3326はclosed/unmergedで重複加算しない。**main pushの別CIと外部Harness完全受領は未検収**。merged #3304の行番号0は既知境界T1-Aの実CLI対照であり、未知RuleをAIが発見するT1-Bではない。
- **実効権限**：#2509のDSH read-only／workspace-writeでは許可外Readが観測され、Session LogへTool結果が複写されるMock実測もある。merged #3333のCLI discoveryはview.writeFlagsを公開し、read-only分類であっても実argv次第でローカル書込みがあることを区別した。全OS隔離・全Egressの無漏洩は未実証。

### 7.2 Claimに相対的な資格ベクトル（計画用語彙。正式Schema・新Gateではない）

| 軸 | 実証する問い | 判別不能・失効時 |
| --- | --- | --- |
| **F0 実Source取得** | Task対象のSource／Subject／Roleを、宣言ではなく実Reader/Loader/Parserが取得したか | 未Loadedを不存在とせず、realpath/Source版を記録 |
| **F1 必要情報充足** | 独立に固定したRequired Data/Function/Business/Rule/Relation/Scenario/SourceがTaskの粒度で揃うか | 正常JSON・同FPでも必須欠落ならpartial、追加取得先を示す |
| **F2 実配送／受領** | 実CLIの必要結果がTool Wrapperを経てHarness Consumerに全量届いたか | Request/Reply ID、stdout/stderr、byte/part/endを照合。切断や欠番は不適格 |
| **F3 現行利用・開示** | 現在のPrincipal/Grant/OS Scope/Audienceで実Read・外部送出・操作を許されるか | read-only広告≠OS拒否≠外部開示許可。旧Grantの再利用不可 |
| **F4 独立判別** | 提案・Scenario・生成物をCandidateと独立のExpected/Mutantで検査したか | 自己生成Expected、既知T1-Aを未知T1-B達成にしない |
| **F5 実作用・独立観測** | 実Provider/HostのResult・Receipt・Resource変化が同Attemptで確認されたか | EffectUnknown時は新ID盲再試行禁止、同ID照会／独立Readへ |

これらは必ずしも線形に全てを満たす必要はない。F1未成立でも不足を探すread-only探索は可能だが、完全なTest生成・自動採用の資格はない。F2成立でもF3の外部LLM開示許可にはならない。F4からF5を推論せず、read-only開発作業に実PG副作用Gateを不必要に課さない。

特に非開示対象の件数・ID・Source path・digestは、エラーや「権限なし」という説明から漏らさない。内部Observer向けの詳細診断と外部Consumer向けの許可済Projectionを分ける（#1178/#2515/#2516）。

### 7.3 接合の単位は一Attemptの観測地点であり、新Proof Storeではない

既存Task Packet/Trial Runbook/Receiptに参照させる最小票：

Task／Claim／独立Required → Source Revisionと実Open root/bytes → 実Binary・Skill/Plugin版・実argv/flag variant・OS Principal → request IDとCLI exit/stdout/stderr → Harness reply ID/Wrapper/受領byte・終端 → Consumerの回答Claim → Current Disclosure → Independent Expected/Mutant → Unknown/Claim Ceiling。

同じhashがあることより、Producer、Transport、受領先の因果的な受渡しを優先する。CLI stdout全量≠Tool Reply全量、Skill Markdown raw byte≠Loaderが包んだTool Result、World/Index Fingerprint一致≠Required全件受領、同一名/同一Test/SourceのA→B→A≠旧PASSの現在再資格化。モデルの最終回答が正しそうでも、Tool未受領ならDelivery合格にはしない。モデルが内部で何を「読んだ」かは自己申告で証明できないため、実送信と独立回答評価を別欄にする。

### 7.4 最初の比較適格H2一件から、H2/H3四Slotへの受入順

1. 公開合成Workspaceの一つの利用者定義Subjectを選び、用途相対のRequired、独立Expected、TEI実Binary、Source realpath/Revision、DSH実版・Effective Config、Tool family＋実argv variant、許可Audience、観測対象Sinkを固定する。Holdoutの正答や秘密Sourceは生成Agentへ渡さない。
2. 同じ実OS Principalで許可CLI／公開SourceのRead正例、別Lane／親root／非許可Skill／合成HoldoutのRead拒否負例を観測する。Profile名やAGENTS上の禁止だけで物理拒否とせず、#2515で強制、#2509で実Tool実行を検査する。実外部ProviderにCanaryを送信せずMockで始める。
3. Tool RequestとCLI実argv・exit・stdout/stderr raw、Wrapper・Reply ID・全byte・終端、モデルへ送られた範囲、Session Log・Tool result・診断・CI Artifactの各Sinkを、観測可能な地点だけ結ぶ。8,192byte切断、56/55の欠番、stderr混入、未許可writeFlagsはそれぞれ独立不適格。
4. Design Context #3262の必要Rule/Relation/Source Anchorと、TT-56の「同FPのvalid JSONなのに必須Rule欠落」を照合する。合法な非Required省略を過剰拒否しない対照も必要。全件Source IndexはDisclosure/Payload Budgetを調べてから使う。
5. 新規H2の**一Session**を先に適格化し、失敗時はSource/意味/配送/OS/開示/Oracleのどこに不足があるかを返す。既存探索4 Sessionを事後的に正式比較へ加算しない。
6. 上記条件が固定された後、新規H2 Tool-only 2 SessionとH3 Tool+Skill 2 Sessionを事前割付。H2ではSkill非露出、H3では本文・必要参照の実Loader Readを確認。独立採点で正答、Unknownの保持、誤操作、不要Tool、実Token・時間・人手を比較し、Skillが無効／有害なら縮小する。第二Harness／MCPは後段。

**必須の負例**：部分切断、欠番、stderr混入、Required Rule/Relation欠落、FP/Source revision混合、非許可write flag、OS禁止Read成功、Session LogのみのCanary送出、A→B→A後の旧資格再利用。未計測Sinkを「漏洩なし」に昇格させない。

### 7.5 他チャットからの横断Owner接続

| 既存Owner | Product Skill／Harnessが渡すもの | 禁止する昇格 |
| --- | --- | --- |
| **#3262 Design Context** | 独立Requiredと実Loaded/Parsed/Returned/Receivedの不足理由、scope-qualifiedな再取得先 | 一Subjectの部分取得から全業務/全Source完備 |
| **#3223/#3273/#1180/#2514 Test** | 必須Ruleが充足した領域だけPattern Proposal→Candidate→既存Scenario/独立Expectedへ | 同一Agent作Expectedの自己合格、既知T1-Aから盲検T1-B |
| **#1251/#951/#3278/#3267 Plugin/Trait/BOM** | exact Contract/Provider/Lock、実Build収録・CLI/Studio Invokeの実Consumer別証拠 | 宣言/選択からProvider実利用、単体PASSから2×2全tuple Verified |
| **#1124/#2586/#2497 三視点/図面/CRUD** | Data/Function/Business・Topology/Representationの必要差とSource Binding | 表示から元データ構造を逆復元、ViewからWrite Targetの自動推論 |
| **#2017/#2515/#2516 安全境界** | Task PacketのSource Scope、OS/Tool principal、外部Audience・Sinkの実到達 | read-onlyからOS隔離、Read許可から公開/送出許可 |
| **#1174/#3328/#1163 副作用** | 書込みTask時のOperation ID、EffectUnknown、Recorder/Stage失敗とReceipt/独立Read不足 | Unknown時の盲再送、古いReceiptによる現在Effect資格 |
| **#2831/#1119 Self-build/横断計画** | 実Builder/Artifact/Plugin/Consumerと必要なInput同一性の問い | Go Buildの成功だけでTEI stage1→stage2/L3達成 |

**計画上の依存関係**：P-1b1の一Function Requiredを固定する作業と、#2509/#2515の最初の適格H2一件は**並列**。T1-BでTest生成を採点するのは該当Required/Expectedが成立した範囲のみ。Plugin D1のCLI/Studio一操作を同一Contractで実Invokeする試験も並行させる。D1全移行、全機密Sink制御、二Harness、MCP、G4を最初のread-only探索の新Hard dependencyにしない。照合済みのClaimだけを#1119へ返す。

**現状・採否**：本節は実証済み局所事実と計画上の成熟差分を接続した**文書変更**である。F0〜F5という正式API/Schema、新Trialの実行、OS物理隔離、全Tool result受領、T1-B盲検生成、実Plugin CLI/Studio二Consumer、実Provider Effect、Owner意味受理を実装済み・PASSとは扱わない。第二Reader/Resolver/Runner/Host/Permission Engine/Proof Storeも設けない。


---

<!-- harness-negative-claim-and-coverage-20261108 -->
## 8. 2026-11-08：負のClaim、Coverage分母、Scope非互換をProduct Skillの判断基準にする

### 8.1 既存成果と残る新問題

[merged #3354](https://github.com/hat47x/tei-platform/pull/3354)の実CLI11応答／32限定Consumer Controlsは、Rule・Relation・Canonical・Source Anchorが正常JSON・同Fingerprintでも局所欠落し得ることを判別する。旧#3324/#3326はclosed/unmerged。PRのexact-head CI #42391707962は成功し、実mainの既存ConsumerとBlob同一性も別途確認された。ただし**main pushの別CI結果、Function全閉包、実H3 Tool配送、現在権限、T1-Bは未検収**。

F0〜F5を区別してもAIが「Ruleは存在しない」「このRequirementは未テスト」「同Fingerprintなので受領済み」と断定する余地が残る。#2497/#2516/#3262が一般のProjection／完全性のOwnerであり、**本Product Skillは既存CLIからどこまで発言できるか、何を追加取得・差戻しするか**だけを所有する。

### 8.2 「対象Scopeに存在しない」を言える条件

独立に定めたTask／Claim Q、対象Scope U、Revision V、開示Audience A、必要集合 R(Q,U,V)に対し、「U内にXが存在しない」と言えるのは当該Claimについて次の条件を確認できた場合だけ。

1. UでのSource/Role/Providerの列挙の完全性が独立に定義され、対象集合を実際に列挙・検査できた。全Workspace一般の不存在と拡張しない。
2. 実Source root／realpathと必要RoleのLoader／Parser／Meaning消費を確認。Manifest、Inventory、Source Indexの宣言だけで代用しない。
3. CLIの検索モード、意味粒度、namespace、RevisionをQと比較可能なScopeで固定し、必要なRule/Relation/Source Anchorの隠れた依存を取り落としていない。
4. CLI Producerの必要な結果と、Harness Tool Consumer側のReply全byte・連番・終端を確認。途中切断・古いPageとの混在を「結果0件」にしない。
5. 現在Grant／Disclosureがその否定命題の開示を許し、非開示、partial、unsupported、stale、未測定を不存在の根拠に含めていない。秘密の件数やpathも許可なしに明かさない。
6. 出力のabsence claimに独立Expectedまたは受理済みSource意味の根拠があり、AIの自己申告・検索順位・件数0のみから推論していない。

一つでも欠ければ、実際に確認できた肯定的な事実を維持しながら「限定Scopeでは未判定」と報告する。必要なら許可されたSourceだけ追加取得する。Sourceからの値の欠落、Reader未対応、非開示、Tool切断、不存在を同じエラーへ畳まない。

### 8.3 現行CLIの具体的な非互換を失敗原因として扱う

- **#3345 Fingerprint Scope**：viewのnamespace-scoped semanticFingerprintをmodel catalog等のWorld/SourceIndex fingerprintとそのまま比較せず、比較対象が異なる場合は「scope-not-comparable」。文字列差のみでSemantic Changeを宣言しない。
- **#3346 未解釈Markdown**：docs/planをManaged Filesとして数えられてもLoaderの対象でない場合は「not-loaded/unsupported」。設計要求が不存在、意味が受理済み、変更不要とは言わない。正式Readerは#2621/#2619。
- **#3347 JSONの形**：validateの診断二重構造、requirementsのresult配列、Catalog内のstdと利用者定義の混在を、既存コマンド別のSchemaで読む。共通Objectへの誤平坦化で欠落を捏造しない。新規Parser／新CLI Schemaの実装は#1155/#2354/#3347へ委譲。
- **#3354 Source Index mode**：subject modeを全件index modeと誤認しない。全件Index利用には実AudienceとPayload Budget/Source消費の照合が必要。CLIの11 reply連続取得だけで一つの原子的Snapshotとはしない。
- **履歴の再資格化**：Source/Test/Lock/GrantのA→B→Aで旧digestに戻っても旧PASSは自動復権しない。7日保持のCI Artifactが期限切れの場合はSource意味変更ではなく、独立監査時の再取得可能性不足として記録する。

### 8.4 「見えない」ことを「未テスト」とする誤りを禁止

Coverageの分母はScenario一覧から逆算せず、別途固定した対象Requirement/Rule/Decision Outcome/NFRで定義する。実取得済みScenario、実行Case、独立Expected、Mutant、Source/Testの現行版を各別列で保持する。

既存6 Pinned Step、既知T1-A第7 Step、32合成Consumer Controlsは母集団が異なり合算できない。#3354のcmd/tei局所Go statement C0相当10.0%をGo branch C1や業務Decision GraphのCoverageへ換算しない。Scenario一覧が部分取得なら観測済みRuleの「covered」は残せるが、全体Coverageは「unknown」。空リストを「未テスト0件」「全体0%」に変換しない。

T1-Bの本当の判別は、AIに既存T1-Aの境界入力や非公開Expectedを予告せず、独立に固定した一FunctionのRequiredから**既存6 Step＋既知T1-A以外の新しい負例**を提案させること。候補の新規性、実Pinned CLI、独立Expected、真のProduct Mutation Kill、Owner採否は別Evidenceとする。候補自作のExpectedで成功扱いしない。

### 8.5 既存Consumer/Scenarioだけで試す10件の独立Pressure

| Case | 変える条件 | Harnessに求める判別 |
| --- | --- | --- |
| HC-01 | 必須Ruleは存在するがSource Indexのsubject modeのみ取得 | wrong-mode／未判定。Rule不存在にしない |
| HC-02 | view局所Fingerprintとcatalog World Fingerprintが不一致 | 異Scopeを識別。意味版変化と断定しない |
| HC-03 | Markdown設計SourceをManagedのみでLoader未消費 | not-loaded/unsupported。不在としない |
| HC-04 | validateの入れ子診断、requirements配列、std混在 | 既存CLIのSchemaに応じて解釈し、空判定やUser Source過算を防止 |
| HC-05 | 同FPの正常JSONからRequired Ruleを削除／非Requiredを削除 | 必須欠落はpartial、合法な非Required省略は過剰拒否しない |
| HC-06 | CLI Exit0だがTool Replyが切断／欠番 | F2不成立。先頭FP一致から全受信にしない |
| HC-07 | 同じ公開Slice、異なる秘密Rule/件数の二世界 | 公開Consumerへ非開示の存在・数・pathを漏らさない（実OSとSink測定範囲を併記） |
| HC-08 | Scenario目録が一部未取得 | 局所Coveredと全体Coverage unknownを併存させる |
| HC-09 | 分割Read中にGrant/Source改版、あるいはA→B→A | 旧Page・旧PASSを現在資格へ合成しない |
| HC-10 | CandidateとExpectedを同じAI誤仕様へ改変 | 独立Expected/Mutantが誤りを判別する |

### 8.6 実行順と非重複の担当

**#3262 P-1b1**：mainの単一Context Consumerを使用し、一Functionの独立Requiredを固定。HC-01〜05/08/09の必要な部分を既存CLIの実結果と一時コピーの負例で検証。既存の良い局所Coveredは保持し、全Source Closureが欠ける間は「探索のみ、正式Test採用不可」。

**#2509/#2515とTAI #168**：新しい比較適格H2一Attemptを用い、HC-06/07とOS許可外Read拒否、実argv/Tool Reply/Session Log/Audienceを対照。OSやTool配送が不適格ならH2/H3効果比較へ混ぜない。

**#1180/#2514/#3223**：独立Requiredが満たせる一RuleからHC-08/10のT1-B候補を既存Scenario verify/generate/check→Pinned実CLI→独立Expectedへ渡す。既知T1-Aの再実行を新発見としない。

**その他のOwner**：Plugin/Trait/BOMは#951/#1251/#3278/#3267、三視点の表示・編集写像は#1124/#2497/#2586、副作用は#1174/#3328、Self-buildは#2831。一般の意味十分性は#2497/#2516、実機密非干渉は#2515のOwnerへ戻す。新しいReader/Parser/Runner/Resolver/Permission Engine/Proof Storeは追加しない。

**状態**：main統合#3354の11 Reply/32合成Controls・既知T1-Aは局所実績。HC-01〜10の統合的な新Attempt、Function全閉包、実Harnessの完全Tool配送、T1-B、C1、現在Grant／独立Effect／Owner意味受理は**NOT RUN**。本節は検証義務を定める設計・計画の更新であり、新しい成功を宣言しない。


---

<!-- harness-query-and-resume-witness-20261108 -->
## 9. 2026-11-08 横断深化：Queryの負の依存・結果集合証拠と、Harness再開の別Epochを接合する

### 9.1 二種類の不足を一つのpartialに畳まない

前節HC-01〜10は主に設計情報とTool配送の不足を扱った。ここでは他レーンの**実Readでの結果集合の真偽**（#2506/#2586/#2514/#3238）と、**開示済みSessionを再開するときの送出/資格の継続**（#2515／TAI #168）を別々に扱う。

生成AIが Function の要件を読めることと、そのFunctionが実Data Sourceで要求Rowsetを読み取れたことは別資格である。さらに実Readが成功しても、Owner集合のExtent、Predicateの評価、結果の粒度、時点、許可Audienceを確定しなければ「全件」「0件」「存在しない」「業務条件を満たす」は証明されない。

**具体的な実証入口**は#2586 HR-01/02の既存対象 order_entry.recent_orders と、SQL Source／bind :from → datasource alias sales →実Instance/Provider→実SQL+args→Rowset／Result Slice→同Function Consumer。現main cmd/tei/cli_view.go のBrowseは applicationview.BuildBrowse を返す分岐であり、**その分岐から実SQL Rowset Readerを直接呼ぶコードは観測されない**（別Caller全体の不存在は断定しない）。#3350 main統合のdatabase Capability選択補正（未選択能力6→4）は宣言上の適合であり、Provider actual invoked／SQL actual bytes／DB readの資格とは異なる。

### 9.2 Query Resultから許されるClaimと必要な証人

| 根拠／観測 | 成立し得る限定Claim | 足りないと不成立となる強いClaim |
| --- | --- | --- |
| 同Identity・Scope・Revisionの実Readで一致行1件 | そのRead時点の存在の証人 | 現在も存在する／全集合の完全性 |
| Rows countがOwner期待件数と同数 | そのProjectionにその数のRowsがあった | **対象Identity集合**が同一とは限らない。重複 A/3,A/3 は A/3,A/9 でない |
| TOP1の一ページ／query limit実行 | 必要条件の一部分を取得した | 全Owner集合のCoverage Complete／他行不存在 |
| 表示列にstatus等がない | Consumerにはその値を渡さなかった | SQL WHERE/Operatorでその列を評価しなかったとは限らない |
| 実SQL/actual bindsと独立Expectedで非出力Predicateを照合 | **Provider内のそのPredicate評価**は検証可能 | 非開示列をConsumerに送る許可／全Rowset・Owner意味受理 |
| 未選択属性のRaw slotがnil／NULL AS slot | そのProjectionでそのslotに値がない | DB格納NULL、ユーザーの業務値未設定、Predicate違反 |
| 選ばれたSQL/Provider Capability | Candidate／静的Selection | 実Link、実Invoke、実SQL byte、実取得Rows、Current Grant |

**負の依存（negative dependency）**も保存する。例えば共通SQLのみで選ばれたRealizationは、DB固有SQLが「選択候補に存在しなかった」ことに依存する。後で固有SQLが追加されれば、既存SQL byteが不変でも候補選択資格は失効する。同様にNOT EXISTS／LEFT JOIN／TOP-N／隠れたPredicateは、追加行や非出力列の変更で旧い否定結論・順位・仮NULLが撤回され得る。出力に含まれないから依存しない、というキャッシュ/Context失効規則は禁止する。

**実証済みの局所境界**：Draft #3175のQ05〜07は実Go ReadPlan＋合成Rowsetでの限定試験をPR専用CI #42334653630でPASS。Draft #3238の隔離PostgreSQL CI #42351796330では、実TOP1／非出力 ordered_at >= :from の実SQL WHERE評価とSelected外列の Raw nil 等を限定観測。ただし両者は**Product Function order_entry.recent_orders→実Provider→Consumer E2ではない**。同一Snapshotの二Readとも証明されていない。元のResult Slice/Lineage/MeaningとSQL実行のOwnerは#2506/#2586/#2622、独立Expectedは#2514。

### 9.3 Harnessの作業再開は「成果の再利用」「実行資格」「旧本文の再送」を分ける

再開に必要な区別は、(a) Task/Claimについて確かめた成果と未決事項、(b) Source/Provider/Plugin/Test/Grantなどの**現在資格**、(c) Harness Session／Tool History／送出先の**現在開示資格**、(d) 実行済み／EffectUnknownのOperation identityと実Receipt、である。過去に成功したTool結果は歴史的証拠として保持できるが、今すぐ再利用・再送・再実行できることを意味しない。

実DSH [#2515 H6-E7](https://github.com/hat47x/tei-platform/issues/2515)／PR #3237の合成canary対照では、同じSessionを再開すると、既送信区間がdsh_session_logへ再送されない一方、**以前のRaw canaryが再構成されたモデル入力に再び現れた**。別Mock計測ではSession Logをoff→onに戻すと未送信区間が送出される。よって「Session Log再送なし」「Toolを今回呼んでいない」「権限が後から失効した」を、過去内容のモデル入力非送出へ変換してはならない。

**再開時の最小Check**（正式なSession/Resume API追加ではなくTAI Runbook／既存Task Packetへの観測項目）：

1. **保全**：Task ID、独立Required、完成済みClaim／歴史的Evidence、未実施/Unknown、停止理由、元Source/Model/Test/Tool/Plugin/Build revision、実ReadとTool受領点を保存。履歴やOwner reviewの撤回は削除しない。
2. **現在性**：新epochで実Source/Revision/必要依存・Operator/Provider/Grant/Disclosureを確認し、Claim別にreusable／needs-recheck／must-retract／unknownを区別。一律全再実行も旧PASSの無検証再利用も避ける。可逆/単調な肯定結果でもScope・as-ofの変化を明記。
3. **会話再生前の開示**：resume前に旧Session中のRaw/Tool Result/Skill/AGENTS/暗黙Contextが新Audienceへ再構成され得るかを確認し、許可できない場合は**旧Session継続を許さず、新しい隔離Sessionへ許可済みの要約・参照だけを渡す**。過去の送出を事後取消しできたと主張しない。
4. **出力Sink**：モデル入力の履歴再構成、dsh_session_logの水位・backlog、Tool result、診断/CI Artifact、外部Networkを別観測。新SessionでもOSの禁止Source Readを実際に拒否し、隠し件数/名前の漏出を検証。未測Sinkはunknown。
5. **作用再送**：Write/Commit後の記録障害またはACK-lossではEffectUnknown。新Operation IDで盲再送せず、同ID Receipt照会・現在Grant・独立Resource Readから再開判断。read-only Queryの再取得とは異なる。
6. **比較試験の継続**：初回H2/H3の固定Source/Skill/Binary/Tool allowlist/Independent Expectedを変えた再開を同じ比較Slotへ編入しない。差が必要なら新Group/epochとして測定する。過去探索Sessionの事後再採点も行わない。

**選択的再資格化の基本**：実Readの結果を利用するClaimに対して、Data依存＋**負の依存**＋実Source/Selection/Query/Scope／Audience／Consumerの閉包が立証された場合だけ、無影響部分の既存Evidenceを再利用できる。閉包自体が未取得なら安全側にUnknown／再取得を返し、コスト削減のために証拠を作り直したように装わない。Performance/Token費用は#2018へ別計測する。

### 9.4 試験の新規差分だけを既存Ownerへ追加（HW-11〜18）

| ID | 独立Fixture／差分 | 受入結果／Owner |
| --- | --- | --- |
| **HW-11** | 全Owner L={A/3,A/9}、Result=[A/3,A/3]（countのみ一致） | 重複とIdentity extentを検出、Coverage未成立。#2506/#2514。Go合成境界は既存Q05 |
| **HW-12** | TOP1の1行成功、Owner集合は2行 | 局所取得成功／全件Coverage unknownを併記。#2506/#2586。既存Q06 |
| **HW-13** | statusがSelected外、Predicateが未実証／実SQL WHEREで評価済みを対比 | 「非出力」と「未評価」を別証拠とし、非開示値を漏らさない。#2586/#2514。既存Q07/Q09 |
| **HW-14** | 未選択列がRaw nilだがDB格納値はNOT NULL | NULL ASのProjection由来とDB値NULLを区別。#2506/#3238。既存Q08 |
| **HW-15** | 共通SQL Source不変のまま、後からDB固有SQLを追加／Provider選択条件変更 | 選ばれなかった候補への負の依存を失効し、再Selection。#2586/#2622。既存QX-01 |
| **HW-16** | Read後にGrant縮小、同Session resumeで旧Rawを会話から再構成 | 今回Tool Readなしでもモデル入力へ送る場合は不適格。必要なら隔離Sessionと許可済み要約で再開。#2515/TAI #168。既存H6-E7を拡張 |
| **HW-17** | Session Log一度off→on、あるいは503後resume | 未送信Backlog／同Range再送をSink別に計測。offだけで削除・非送出を宣言しない。#2515/#2509/TAI |
| **HW-18** | Source/Test/Grant/非出力Predicateを変更して一部Claimだけ無効化、他は固定 | 独立Expectedとの対照で選択的再資格化・撤回/再発見を判別。#2533/#2514/#958、#1163へEvidence束縛 |

これらは既存Q05〜09、QX-01、H6-E7、CC-18を**別製品テストとして数え直さないためのHarness受入差分**。最初の実Product試験は#2586 HR-01/02の公開Reference一Functionを優先し、実Reader・実SQL+bind・Rowset/Read extent／Predicate・許可Consumer受領を固定する。HW-16〜17は合成canary/Mock受信で別レーンとして進める。Source/Provider/Grantが揃わなければ「Product Read E2=not-connected/unknown」と報告し、実PG Reference PASSを流用しない。

### 9.5 合流条件と現状

**Claim単位の証拠接合**：Q/Required → 実Function/Query選択＋選択されなかった候補集合の根拠 → SQL Source/actual bind → 実物理Provider/Rowset/Grain/Extent/Predicate evaluation → Tool Reply/Consumer → Scope/as-of/Current Grant → Independent Expected → 出力Claim／Unknown／再資格化対象。セッション再開ではさらに歴史的モデル入力・送出Sinkと現在Audienceを別接合する。これは既存#1163 Evidence Packet/TAI Runbookの**観測欄の接続**であり、Universal Evidence Engineや新Host、SQL Runner、Session SSOT、権限Engineは追加しない。

**今回の作業資格**：#3354 mainの11 Reply/32局所Consumer Controls、Q05〜09の限定Go/PG実証、#2515のH6-E7 Mock再開／過去モデル入力へのRaw再登場は**既存の固定条件下での証拠**。本HW-11〜18のProduct Function E2、現在Grant縮小後の実再開拒否、第二Harness/Egress全経路、T1-B、Owner業務意味は**NOT RUN**。前節HC-01〜10とHW-11〜18は試験計画の数であり、新たな実行PASS数ではない。


---

<!-- harness-imported-provider-occurrence-20261108 -->
## 10. 2026-11-08：外部Provider Sourceの見落としを防ぐ依存閉包と再資格化

### 10.1 現行mainと新反例の差

- **実装済みの狭い事実**：[merged #3477](https://github.com/hat47x/tei-platform/pull/3477)では、同一Workspace内のValidation Sourceを末尾LFだけ変更した場合、Canonical World FPが不変でもSource Index FPの変化を実CLIで検出し、旧Design ContextからScenarioへの採用を拒否する限定試験がmainへ統合された。A→B→Aで元Fingerprintに戻っても**過去の実行・改訂履歴の消失は証明しない**。これを真のAI未知Test生成や実外部Provider Importの成功にはしない。
- **まだ閉じていない反例**：[Draft #3479](https://github.com/hat47x/tei-platform/pull/3479)の現行Go Fixtureは、**別WorkspaceからimportしたProviderの原文末尾LF**、および**Provider Manifestだけ**を変更する。ConsumerのWorld／Semantic Digest／Managed Source Index（Artifact/Reference/ModelRevision）が不変でも、Provider raw SHA／ManifestIdentityは変わり得る。同じConsumer FPから「外部参照先Sourceはfresh」と判定できない。#3479のexact HEAD試験は確認時CI進行中であり、**このDraftのCharacterizationをmainへ統合済みのProduct修正としない**。
- **現在の安全側挙動**：外部Importを含むLoadについて、Consumerだけから完全なSource Indexを組み立てる既存経路は `ErrImportsNotBound` で拒否する。これはSource/Manifest occurrenceの必要性を示すが、正しい複数Provider Snapshot・Current Grant・全Consumer受領の自動保証ではない。
- **上位計画との重複回避**：[merged #3481](https://github.com/hat47x/tei-platform/pull/3481)が `plan/MASTER_PLAN.md` にP0-D（Source）、P0-M/R/C（三視点と実Read）、P1-P（Plugin）、P1-T/C（AI/Context）、P2-B（2×2 BOM）の実装順を反映済み。本節はその繰返しでなく、**Harness ConsumerがどのSource occurrenceを読んだのか**と、再利用/引用できるClaimの資格だけを扱う。

### 10.2 「同じ意味」「同じSource」「同じ受領・実行」は別の同一性

| 比較対象 | 一致からいえること | 一致からはいえないこと |
| --- | --- | --- |
| ConsumerのSemantic/World FP | そのConsumerで観測されたCanonical内容の一致 | 外部Provider raw/Manifestの不変、全Import閉包 |
| ConsumerのManaged Source Index FP | Consumer自身の管理対象Sourceの限定同一性 | Consumer外のProvider Source occurrence・実Import先までの不変 |
| Provider raw SHA／ManifestIdentity | その一対象の原文Byte／Manifest Byte一致 | 実Provider選択・Nested Loadの原子的性質・意味Owner受理 |
| Provider/Plugin Lock・Artifact | 宣言／選択／Build入力等の当該範囲での照合 | 実Load/Invocation、Host認可、現在Disclosure、実結果 |
| Tool Reply body／Session Resume | 受領したTool Byteや履歴の一部 | 必要なすべての外部Source依存、将来の開示許可、現Revisionの正しさ |
| Accepted Meaning／Scenario PASS | 受理範囲での業務意味またはそのTest版の既往結果 | Source引用の鮮度、別Toolchain/Consumerでの現在実行資格 |

外部Sourceの改訂が**Canonical意味を変更しない**場合にも、Sourceへの位置付き引用・設計書出典・Test候補・実Build入力の鮮度は別々に再確認が必要である。逆に、原文Byteが変更されたというだけで、Ownerの意味受理、無関係なSourceや全Scenarioを一律撤回しない。対象Claimから必要な依存Occurrenceへさかのぼれる範囲だけを失効／再資格化する。

### 10.3 Task相対のImport Closureを、既存Resolverの証拠として取得する

一つの公開合成Task／Functionで外部ProviderをRequiredとしている場合、最小受領票に以下の**観測できたものだけ**を束縛する。

`consumerWorkspace/root+revision | Task/Required/Role+Source Anchor | imported provider identity+actual target path | provider raw source occurrence+manifest identity | import/loader/parser resolution status | selected contract/provider+exact version | actual consumer read+Tool reply | disclosure/as-of/current grant | independent expected | unknown/expiry cause`。

- **Import対象の特定**は既存 `LocateImportedSource`／Source Index／Resolver経路に委譲し、新しいProvider Registry、第二解析器、横断Snapshot保証APIを作らない。
- **閉包が未証明**なら、ConsumerのWorld/Index FPが一致していてもSource freshnessのClaimは `unknown/stale-needs-check`。Source「不存在」や正式Test生成可へ自動変換しない。
- **同時性の限界**：ConsumerとProviderの別Open・別Tool Callで各digestが一致しても一つの原子的Snapshotを得たとは限らない。実読込時のSource/Manifest/Import Bindingと受領時点の版ずれを発見できる限度で記録する。A→B→Aの履歴、権限失効、外部Repository移動は別反例。
- **MeaningとEvidence**：REAによる旧資産取込、SHOの文章/図面、KON DataspaceのProvider、Plugin BOM等は、このTask相対のProvenanceを消費する先であって、本章が新たな意味受理・Tool/Host認可を発行する先ではない。
- **MeaningとEvidence**：外部逆解析の旧資産取込、SHOの文章/図面、Plugin BOM等は、このTask相対のProvenanceを消費する先であって、本章が新たな意味受理・Tool/Host認可を発行する先ではない。

### 10.4 外部参照を使う場合だけ追加する反証（HI-19〜24）

| ID | 変更する一条件 | 正しい判定とOwner |
| --- | --- | --- |
| **HI-19** | Consumer内のValidation Source末尾LFを変更（Canonical不変） | Source Indexの時点差で旧Contextを拒否。**main #3477の既存限定試験**を再計上しない |
| **HI-20** | 外部Provider rawだけ末尾LF変更、Consumer FP不変 | 外部Source occurrence未照合ならfreshを否定し再確認。**Draft #3479**のGo反例、#3262のDesign Contextに渡す |
| **HI-21** | Provider Manifestだけ変更、Consumer World/Source Index不変 | Manifest Identityと実選択先を別途照合。Draft #3479のGo反例、#951/#1251のProvider Contractへ |
| **HI-22** | Provider raw A→B→A、またはSource移動・別bytes同名 | 現在のraw同値から途中履歴／旧Receipt再資格を導かない。#2514/#3176の既存反例に照合 |
| **HI-23** | Consumer Source/Meaning同じ、実Plugin版／Nested Import／Build入力だけ変更 | 選択・Build収録・実Invoke・BOM tuple／実Consumer資格をそれぞれ再確認。#3267/#2600/#2831に引渡し |
| **HI-24** | H2/H3 Session再開時に元の外部参照が変わり、旧Rawだけ会話へ残存 | 過去Readの歴史と現在Source freshness/Disclosureを区別。旧Rawを新Audienceへ再送不可なら別隔離Session。#2509/#2515/TAI #168 |

**最短実行順**：①#3479が現行Go FixtureのSource/Manifest変化の反例をexact HEADで検収、②#3262が同じRequired一FunctionでImport時の未報告Sourceを `unknown` と扱えることを実Consumerで確認、③そのSourceを必要とするH2/H3 TrialではTool Replyと外部Source occurrenceの両方を固定、④Product Function→actual Provider→Rowsetは#2586、正式BOM/Build/EffectはそれぞれのOwnerで検証する。**外部Importを用いない最初の公開合成H2を#3479の完了待ちで停止させない**。

**状態**：#3477・#3481はmain統合済み。#3479はDraft、実Provider Import Closureの製品読取・実H2/H3受領・Source原子的Snapshot・SHO/REA/KON実連携、T1-B/Owner意味受理は未成立。HI-19〜24は既存テストの重複カウントを禁止した**受入設計**であり、この節の追加による新しい実行PASSではない。
**状態**：#3477・#3481・#3479はmain統合済み。ただし#3479のGo反例は別々のLoad観測であり、実Provider Import Closureの製品読取・実H2/H3受領・Source原子的Snapshot・SHO/外部逆解析の実連携、T1-B/Owner意味受理は未成立。HI-19〜24は既存テストの重複カウントを禁止した**受入設計**であり、この節の追加による新しい実行PASSではない。

---

<!-- harness-observer-scoped-claim-20261108 -->
## 11. 2026-11-08：Observer相対の資格・不開示・外部実Readを同じClaim境界へ

### 11.1 何を証明したかではなく、誰に対してどのClaimを証明したか

[Design Context #3262](https://github.com/hat47x/tei-platform/issues/3262)のD/T/G分離と、[外部Harness隔離 #2515](https://github.com/hat47x/tei-platform/issues/2515)のH6-E13〜15を組み合わせ、既存Harness Receipt/Contextの読み方を次のように限定する。

- **D / Design Input**：当該Function/Taskの独立Requiredが、Scope・Source・Role・Rule・Relation・外部Import occurrenceを含めて実取得されたか。現mainの11実CLI Reply/32合成Controlsは、外部Importを含む任意のRequired閉包の証明ではない。
- **T / Delivered**：同一Attemptの実CLI stdout/stderr/Tool Reply parts/end markerが、指定Consumerに欠落なく届いたか。Harnessの56 Request/55 Replyや8,192 byte切断の負例を、ローカルJSON Copy検査のPASSで埋めない。
- **G / Generation**：独立Expected/Mutantによって、既知T1-Aと未知T1-Bを分離して採点したか。DとTが成立してもGを自動Qualifiedにしない。
- **A / Audience & Authorization**：同じ内部Evidenceでも、`internal observer`、`public viewer`、`AI harness`、`SHO/Studio`のどの受領者が、**現時点**でどのProjectionを見てよいか。内部で`permission-denied`と`not-found`を区別できても、公開Viewerへその差を返してよいとは限らない。
- **E / Effect**：実Provider/DB/外部操作の効果と独立再Readを確認したか。Query結果が空、Tool exit 0、Plugin選択済みという事実から業務効果を導かない。

**Claim資格はD/T/G/A/Eの全項目が常に必要という意味ではない**。例えば公開合成Sourceのローカル説明にはEを要しない。一方、`safe to disclose`にはA、`AI generated a valid new Test`にはD/T/G、`real database-backed Function works`には実Read/Bindingと必要なA/Eが要る。各Claimが必要とする条件だけを独立Requiredに固定し、未観測軸を`unknown`に保つ。新しい共通Gate/Schema/Proof Storeは作らない。

### 11.2 非表示・欠測・空集合を混同しない

[Scoped Coverage #3365](https://github.com/hat47x/tei-platform/pull/3365)の考え方、#2515のObserver差異、[Query/Function #2586](https://github.com/hat47x/tei-platform/issues/2586)を横断する負例を既存のOwnerへ追加する。

| 条件 | 内部の正当な観測 | 公開／外部Consumerの上限 |
| --- | --- | --- |
| Requiredの一部が非取得・途中切断 | `partial/not-loaded`と欠測地点を記録 | `complete`、全体Coverage、`qualifiedTestGeneration`にしない |
| 非可視Row/Sourceがある／存在しない | 権限を持つ内部Observerが拒否理由を区別 | 未許可なら両者を識別可能な診断・件数・Traceを出さない |
| Filter後0行、実Query未実行、Provider未選択 | `empty-result`／`not-executed`／`unbound`を分離 | 0行を「データなし」や「対象全件検査済み」と断定しない |
| Local Testの一部だけCovered | 局所Coverageと全体Unknownを併記 | 未観測のRule/RelationをCoverage分母から暗黙除外しない |
| 以前のSessionにはRawがあったが権限失効 | Historical Receiptと現在Grantを別管理 | Session Resume/Log Replayで旧Rawを新Audienceへ再送しない |
| 内部Mock/ProxyでEgressが観測されない | 実観測SinkとOS Profileを記録 | 子Process/DNS/Telemetry等未測定の出口まで非漏えいを主張しない |

これは**可視性を理由にRequired不足を隠す設計ではない**。当該Audienceに必要な情報が開示できないなら、内部で不足理由を保持したまま、そのAudience向けClaimは`unqualified/unknown`とする。内部理由の詳細を公開するかは既存Host/Policyの責務。

### 11.3 Product DB実Read・Plugin Trait・SHO/KONを接続する最小Witness
### 11.3 Product DB実Read・Plugin Trait・SHOの実受領を接続する最小Witness

[Draft #3500](https://github.com/hat47x/tei-platform/pull/3500)は、既存Loader Resultからnative Case Queryを組み立て、隔離PostgreSQLの実ExecutorとQuery Mapを使う**オプトイン**の実Read試験を提案する。隔離DB名確認と既存Table拒否を前提とし、環境変数がなければSKIP。したがって通常Go TestのPASS、Fake ProviderのPASS、または本DraftのCIだけから実PG Readが成立したとは言えない。

実DB Readが後日成立しても、`Function Data Context`／Tenant Scope／現在Grant／そのAudienceへの安全なDelivery／Owner意味受理／Product E2は別。必要なら実際に読んだDatabase Identity、Native SQL、Bind Parameter、Result Mapping、Consumer受領を**同一Attempt**で接続する。新しいQuery Executorは作らず既存 `postgresquery.Executor`／`queryresult.Map`を使う。

[Plugin Trait #3278](https://github.com/hat47x/tei-platform/issues/3278)の`Selected / Built-in-included / Loaded-or-Invoked / Exposed / Authorized / Effect-observed`も同様に別資格。選択A・実Invoke B、Trait Contract同版・実装のみ変更、`Retryable`なのに副作用不明、`Encrypted`なのにBrowser plaintext保存を合格にしない。2 Provider×2 Consumerの各tupleは実Build/Invoke/独立Expectedで個別判定し、別TargetのPASSを転記しない。

SHOではCLIからのSource出力とSHO Readerの**実再読取**、Working Copyと保存後再開、Source引用の鮮度を区別する。KON DataspaceはSource/Documentの将来の受け皿候補として検討しても、TEIのDB/Query/意味正本を移さない。SUI側でRaw保存→Reader再取得→複数Canvas投影→Revision/権限/履歴を確認するまで、KON独立運用・SHOクラウド受領の成立を主張しない。
SHOではCLIからのSource出力とSHO Readerの**実再読取**、Working Copyと保存後再開、Source引用の鮮度を区別する。TEIのDB/Query/意味正本を他の記述・表示Consumerへ暗黙移管しない。外部ConsumerでRaw保存→実Reader再取得→複数表示→Revision/権限/履歴を独立確認するまで、SHO等の実受領が成立したとは主張しない。

### 11.4 次の最小受入順と既存Issueへの引渡し（HO-25〜30）

| ID | 最小の反証 | 担当・検収境界 |
| --- | --- | --- |
| **HO-25** | 必要Ruleが欠落するが局所CoverageはPASS | #3262/#3365/#2514：Task RequiredとCoverage分母、Unknownを分離 |
| **HO-26** | 内部Observerにはpermission-denied、公開Viewerには対象存在が非可視 | #2515/#1178：公開投影の観測同値。実OS/Policy受理は別 |
| **HO-27** | 途中切断・空Result・実Query未起動の三ケース | #3262/#2586/#2509：Tool受領とQuery実行の別証拠 |
| **HO-28** | 実PG未設定でGo PASS、設定後に実Read成功または拒否 | #3500/#2586：SKIPを実PG成功にしない。実DBの隔離実行は未測定 |
| **HO-29** | Plugin選択済みAだが実Invoke B／Trait Contract同版実装変更 | #3278/#3267/#951：Lock、Build、実Invoke、Effectを別資格 |
| **HO-30** | SHO/Studio/KON候補へ旧Sourceを引用、Resume時にAudience Grant変更 | #2515/#3262/SHO・KON各Owner：実Reader・Current Disclosure・履歴を別確認 |

**実行順**：①外部Importなし公開一FunctionのD不足を#3262で判別、②独立Expected付きの初回比較適格H2一Sessionを#2509/#2515で取得、③そのTaskが実Providerを必要とするときだけ#3479のImport occurrenceと#3500の実DB Witnessを接続、④SHO/Studio/KONの実Consumer受領とPlugin 2×2を別受入とする。**外部Import、実DB、KON独立化を初回H2の新しい前提条件にはしない**。

**証拠上限**：本節はOwner間の受入設計であり、HO-25〜30、実OS拒否、実PG接続、実H2/H3、未知T1-B、SHO/KON実Reader、Plugin 2×2の新しいPASSは**NOT RUN**。#3500/#3479/#3365等のDraftは現main製品能力として数えない。
| **HO-30** | SHO/Studio等の外部Consumerへ旧Sourceを引用、Resume時にAudience Grant変更 | #2515/#3262/SHO等の各Owner：実Reader・Current Disclosure・履歴を別確認 |

**実行順**：①外部Importなし公開一FunctionのD不足を#3262で判別、②独立Expected付きの初回比較適格H2一Sessionを#2509/#2515で取得、③そのTaskが実Providerを必要とするときだけ#3479のImport occurrenceと#3500の実DB Witnessを接続、④SHO/Studio等の実Consumer受領とPlugin 2×2を別受入とする。**外部Importや実DBの実証を初回H2の新しい前提条件にはしない**。

**証拠上限**：本節はOwner間の受入設計であり、HO-25〜30、実OS拒否、実PG接続、実H2/H3、未知T1-B、外部Consumerの実Reader、Plugin 2×2の新しいPASSは**NOT RUN**。#3500/#3479/#3365等のDraftは現main製品能力として数えない。

---

<!-- harness-api-portability-hv-evaluation-20261108 -->
## 12. 2026-11-08：共通APIの限定同値とHarness方式選定を、実際の判断品質で評価する

### 12.1 mainへ到達した根拠を更新する

- [merged #3439](https://github.com/hat47x/tei-platform/pull/3439)で**ADR-0048**が共通型付きApplication APIを採用した。CLI/MCP/Studio/Web/Desktopの同一意味・Target・権限を目標とするが、採用決定は全Transport実行同値の実証ではない。
- [merged #3403](https://github.com/hat47x/tei-platform/pull/3403)で4 read-only MCP参照Adapterを追加し、[merged #3458](https://github.com/hat47x/tei-platform/pull/3458)では`validate`のCLI/MCP応答を正常・読込失敗・重複宣言の限定3系列で照合した。**MCP Adapterが存在すること**と、外部H2/H3 Harnessの実Tool Reply/Current Grant/完全配送を確認したことは別。
- [merged #3482](https://github.com/hat47x/tei-platform/pull/3482)は`sales.order.submit`について、独立RequiredのCapability/Application/Validation/Source位置を**実CLI 13 Reply／Consumer判別42件**で限定検証した。以前のmain #3354の**11 Reply／32合成Controls**と同一試験の上書き・加算をしない。いずれもProduct Provider実Invoke／Owner業務受理／未知T1-Bを証明しない。
- [Draft #3470](https://github.com/hat47x/tei-platform/pull/3470)のSession Log backlog分割送出／停滞の記録修正は、`request`／`reply`／`wire segment`／`receiver accepted`／`model input replay`を別単位で採点する必要性を補強する。Draftをmain実装済みとしない。

### 12.2 実Harnessの選定は方式名ではなく独立Expectedとの対照で決める

同一Task／Source／独立Required／Audience／Tool Allowlist／Cost条件を固定し、必要な段階だけ比較する。

| 段階 | 比較する構成 | 成功と判定できる上限 |
| --- | --- | --- |
| **HV-0** | 決定的なCLI/既存Testのみ（AIなし） | D/実Source/独立Expectedのベースライン。AIの価値は主張しない |
| **HV-1** | LLM＋明示Tool（最小ループ） | 同一Taskでの不足検出・保留・根拠追跡とTool完全受領 |
| **HV-2** | 既存の外部Harness＋CLI | 実Tool起動・送出/隔離/再開・Cost/操作負担の観測 |
| **HV-3** | 既存Harness＋TEI/SEIのProduct Skill/Plugin | HV-2比で独立Expectedの判断品質／負例検出／作業効率が改善するか |
| **HV-4** | TEI独自の実行ループ | HV-3では満たせない具体的Requiredが実測され、代替不能性・運用費を含めて承認された場合だけ検討 |

**HV-4を製品構成の前提にしない**。既存Harness＋Plugin/Skillで必要な判断が成立するなら、独自ループは作らない。MCPは必要な実操作の薄いAdapterであり、初回H2/H3に新MCP Protocolを必須化しない。A/BのTool-only/Skill比較はモデル・Source・Tool・権限・Run単位を固定し、途中で新Skill説明やMCPを加えたものを同一Groupへ混ぜない。

**評価票**：独立Expectedに対する重大誤判断、未知Required発見、根拠Sourceの鮮度と位置、必要な保留の正しさ、非開示の漏えい、実Tool配送完全性、再現性、実行時間・Token/金額・人間介入回数。回答文の自然さや単純なTool成功率を主要な合否にしない。HV-0〜4の比較実験そのものは**NOT RUN**。

### 12.3 判断結果の保存・再開と実行を混同しない

TEIのSource/Function/Query Read、SEIのRecommendation、SHOの記述、KONの保存候補は**同じ証拠を参照し得るが、同じAuthorityを持たない**。
TEIのSource/Function/Query Read、SEIのRecommendation、SHOの記述は**同じ証拠を参照し得るが、同じAuthorityを持たない**。

- TEIから受けた実Read Evidenceと、SEIが記録する`Recommendation`の**実際の推薦者**を分離する。モデル生成の提案をTEIのOwner受理済み意味へ帰属させない。
- 決定的なD0/D1で十分なTaskでは、M1（モデル利用）を自動採用しない。M1の追加価値を独立Expected／保留／コストで検証する。
- 保存後に`Decision`だけを再開できても、未解決の`Question/Unknown/Conflict`が保存・再読取できたことにはならない。H3のResume受入では、未解決項目の**存在・Source・状態・開示範囲**を独立に再照合する。
- Legacy/REA観測、AIの構造候補、Owner Accepted Desired、Target Observed Actualは別票。SHOの文章化、SEIのRecommendation保存、KONのRaw保持、TEIのBuild/Effectのいずれも、他の意味受理・実実行を自動的に成立させない。
- Legacy/REA観測、AIの構造候補、Owner Accepted Desired、Target Observed Actualは別票。SHOの文章化、SEIのRecommendation保存、TEIのBuild/Effectのいずれも、他の意味受理・実実行を自動的に成立させない。

### 12.4 既存Ownerで検証する追加の差分（HP-31〜35）

| ID | 最小対照 | 受入とOwner |
| --- | --- | --- |
| **HP-31** | 同一`validate`正常／読込失敗／重複宣言をCLI/MCPで取得 | 同じ対象・版・結果を限定比較。既存#3458を再計上しない。#3403/#1158 |
| **HP-32** | `sales.order.submit`のRequired 1要素を意図的に欠落／Scope変更 | #3482の13 Reply/42 Controlsを起点にDだけ再資格化。T/G/実Providerはunknown。#3262/#2514 |
| **HP-33** | HV-0〜HV-3の同一Taskで未知Required・誤判断・保留・費用を比較 | HV-3で充足ならHV-4を棄却。#2509/TAI #168／SEI担当 |
| **HP-34** | H3終了→再開でDecisionは残るがQuestion/Unknown/Conflictが欠落 | Resume不合格。Recommendationの実推薦者とTEI Read Sourceを別保持。SEI担当／#2515 |
| **HP-35** | Tool Reply一部欠落・Session Log分割／backlog・モデル入力への旧Raw再出現 | request/reply/wire/receiver/model-inputを別観測。#2509/#2515／Draft #3470 |

**最短順**：#3482の限定Dを採用しつつ、初回H2の実Tool Receipt/完全配送（T）を先行する。必要なSkill差分を測ってからHV-3を評価し、独自Harness HV-4の必要性を判断する。SEI H3保存再開・外部Provider Import・実DB/Plugin Invocationはそれぞれ独立並走。**HP-31〜35の新たな実行、HV比較、外部H2/H3、SEI実保存再開、Owner意味受理はNOT RUN**。


---
<!-- harness-adaptive-activity-interaction-flow-20261108 -->
## 13. 2026-11-08：Adaptive ActivityとInteraction Flowを分離し、独自Harnessの採否を確定可能にする

### 13.1 「AIが次に何をするか」と「業務操作がどう進むか」は異なる

[TAI Harness/Skill Draft #270](https://github.com/hat47x/tai-pleroma/pull/270)と他製品の横断設計を、既存ADR-0048・TEI Plugin/Operation契約に整合させる。

| 層 | 主な責務 | 担当と境界 |
| --- | --- | --- |
| **Adaptive Activity** | 不足を発見し、次の調査・Tool候補・保留・人への引継ぎを選ぶ | TAI/SEIのHarness・Method・Skill。TEIのOwner意味受理・Host実行権限は発行しない |
| **Interaction Flow** | UI・会話・Tool・外部イベントの入力から、型付きOperationへ進む明示的な状態と遷移をモデル化 | TEIのFunction/Representation/共通Application API。AI専用状態機械に固定しない |
| **Business Operation** | 実業務のValidate/Authorize/Execute/Observe/Resultと効果 | TEI契約＋実行Host/Provider。Harnessの成功/終了コードは業務Effectではない |
| **Evidence & Persistence** | Source/Raw/Meaning/Decision/Unknown/Receiptの版・参照・再読取 | 各製品の既存正本とConsumer。KONは保存先候補であり、意味・権限の中央正本ではない |
| **Evidence & Persistence** | Source/Raw/Meaning/Decision/Unknown/Receiptの版・参照・再読取 | 各製品の既存正本と実Consumer。保存先の存在から意味受理・Host認可を推定しない |

これらを一つの「AI Workflow」定義へ畳み込まない。Interaction Flowは**AIを使わないUI/業務フローでも成立**し、Adaptive Activityは固定Flowの外側で次の観測や推薦を選べる。ただしAdaptive Activityが型付きOperationを呼ぶときは同じ契約・権限・監査点を通す。

### 13.2 共通Lifecycleの不変部とPluginによる可変部

Operationの候補Lifecycleは `Intent → Prepare → Validate → Authorize → Execute → Observe → Result`。これは**横断設計上の候補**であり、現在の製品APIが全Stage/Hookを実装・検証済みという意味ではない。既存ADR-0048、#1158、#3278/#951、実Hostに接続し、Hook追加による実行順序変更や権限迂回を許さない。
Operationの `Intent → Prepare → Validate → Authorize → Execute → Observe → Result` は**一つの説明用Path**であり、製品固定の全順序やCore列挙ではない。Lifecycleは既存モデルの型付き有向Graphとして、分岐・再試行・補償・異質媒体の入力を許容する。保護すべき不変境界と、追加・差替え可能なGraph要素を分離する。現在の製品APIが全Node/Hookを実装・検証済みという意味ではない。既存ADR-0048、#1158、#3278/#951、実Hostに接続し、Hook追加による実行順序変更や権限迂回を許さない。

- 不変境界：実Source/Operation identity、入力/出力型、Current Authority、実Effect/Receipt、公開Disclosure。副作用あるExecute前のHost認可、再試行と冪等性の判断をAI/Pluginの自己申告だけで確定しない。
- 可変部：表示・質問・Tool提案・データ変換・Operation前後の拡張・検証・再開。Pluginは能力発見、情報投影、操作前検査、実行/結果変換、結果検証、再開/引継ぎに接続できるが、`Selected → Loaded → Invoked → Consumer Received`を各段階で観測する。
- Flow図表はモデルの入力とし、**図表の変更が実際のGenerator/Target出力へ到達する**ことを独立Expectedで検証する。説明図だけ更新して実行Flow不変なら製品Flow変更の資格はない。
- 「全Lifecycle自体をPluginで差し替える」場合も、上記不変境界とHost/Ownerの責務を越えられない。既存のHook/Operationモデルで表現不能な最小反例が出るまでは第二Workflow Engineを作らない。

### 13.3 H3で必要なSource付きRecommendationと未解決事項の再開

H2は既存CLI/Tool-onlyの実配送・隔離・Current Grantの資格を先に取る。H3のProduct Skill/SEI連携は次の順序で検証し、H2の前提にしない。

- **H3-A0（最小縦切り）**：TEI実CLI `source-index`等から観測したSource/Role/Revisionを、SEIの既存`Recommendation`に参照として記録し、保存後の`resume --json`で**同一のSource参照が実再取得できる**ことを独立に照合する。TEI SourceがSEIの推薦者やOwner意味受理へ化けないことも確認。
- **H3-A1（追加価値）**：決定的なD0/D1とMethod/LLM利用M1を同一Task・独立Expectedで比較し、未知Required・誤判断・保留・Costの差を測る。M1の説明品質だけで採用しない。
- **H3-A2（未解決の継続）**：Decision/Recommendationの保存と、`Question / Unknown / Conflict`の保存・再読取を別資格にする。現SEI CLIで未実装/未確認の保存型を「既に再開可能」と扱わない。

H3-A0が失敗した場合は既存SEI Reader/Writerの原因と、TEI Sourceの取得/Revision原因を分離する。SEIへ第二TEI Parser/Registryを作らず、SHO/KON/TAIの実保存・Consumer受領も独立に確認する。
H3-A0が失敗した場合は既存SEI Reader/Writerの原因と、TEI Sourceの取得/Revision原因を分離する。SEIへ第二TEI Parser/Registryを作らず、SHO/TAI等の実保存・Consumer受領も独立に確認する。

### 13.4 最小の実行可能な検収（HF-36〜41）

| ID | 一条件だけ変える反証 | 受入のOwner |
| --- | --- | --- |
| **HF-36** | 同一FunctionのInteraction FlowでAIなし／AIありを切替 | TEI FlowのOperation/Grant/Effectは同じ契約、Adaptive Activityは推薦のみ。#1158/#3278 |
| **HF-37** | PluginがAuthorize前後のHook順序を変更し、未認可Executeを提案 | Hostが拒否し、Hook提案/実Invoke/実Effectを別票。#951/#2515 |
| **HF-38** | Flow図表だけ変更、Generator出力は不変 | 実Flow変更は不合格。図表→モデル→生成物の差を確認。三視点/Modeler Owner |
| **HF-39** | SEI Recommendation保存後、TEI Source revisionだけ変更 | H3-A0のSource再読取/失効を別表示。SEI H3/TAI #168 |
| **HF-40** | Decisionのみ再開、Question/Unknown/Conflictは未保存 | H3-A2の未解決継続は不合格。SEI Writer/Reader Owner |
| **HF-41** | HV-3既存Harness+Skillが独立Expectedを満たす | HV-4独自Loopを棄却。実H2/H3資格とCostを比較。#2509/TAI #168 |

**Critical Path**：P0は実Source/FunctionのDと比較適格H2（T/A）を**並列**に進める。H3-A0はSource参照の保存再読取という一つの縦切りで独立実施。Lifecycle/Flow/Pluginの追加実装は既存契約で表現不能な反例から起票し、独自Harness・新Host・新中央Registryを前提にしない。**HF-36〜41、実H3-A0/A1/A2、Plugin Hook実行、Flow図→Generator、HV-4比較は今回NOT RUN**。

---

<!-- harness-question-relative-csw-trial-20261108 -->
## 14. 2026-11-08：問いQに対する十分性、CSW探索の増分価値、DSH適格試行の実行票

### 14.1 D/T/G/A/Eは問いQ相対で評価し、空の実Readを不存在の証明にしない

[SEI Cognitive Method #767](https://github.com/hat47x/sei-cognition/issues/767)と[main統合 #770](https://github.com/hat47x/sei-cognition/pull/770)の合成Referenceを接続する。#770で5 Module・計34件の**ローカルPython限定PASS**が記録された。GitHub Actionsは当面停止のため、CI成功／実SEI Product・TEI SQL Provider・実Hostの証拠にはしない。局所条件を検査する`test_negative_evidence_closure.py`は実行器でも実証明エンジンでもない。

既存のD（必要Source）、T（実受領）、G（独立評価）、A（Audience）、E（Effect）を別の資格としたうえで、**Q = Question/Claim**ごとに必要なPredicate/Relation/対象Universe/Scope/Provider route/時点を独立Requiredとして確認する。これを新しい共通型・Gate・Storageにはしない。とくに否定Claimは片側非対称である。

- `R=[]`というResultだけでは、未取得・対象外・部分結果・全件完全取得・非可視を区別できない。完全な対象集合、実Read、QuestionのPredicate/Relation、Provider routeとSnapshot、現在の開示資格が確認できない`NOT EXISTS`は**unknown**を維持する。
- 返却値が同じ空配列でも、非出力WHERE Predicate／Relation occurrence／新規Dependency追加でQの意味や対象集合が変わる。Consumer World/Source Index Fingerprint、Tool exit 0、SQL IR中の依存候補だけで完全外延を宣言しない。
- 一方、**許可された同じQの対象集合**に属する実Witnessが1件あれば、全件列挙がなくても当該`NOT EXISTS`への反証になり得る。ただし旧Predicate/Scope/GrantからのWitnessを現在Qの証人に流用しない。
- SEI/CSWの責務は「根拠が足りない、次に何を確かめるか」の提案と理解の改善。必要な実Query/Source/Read/Host Grantの証拠はTEI/各Productが発行し、TAIは受領/現在のActivity再資格を扱う。SEIの`conditionally-absent-in-fixture`を業務的な「存在しない」へ昇格させない。
- Audienceが非開示のPredicate/非可視IDによって判定不能なら、公開Consumerには許可済みProjectionで`unknown`を示し、内部の詳細拒否理由／原Path／件数／Secret-derived digestを無許可で出さない。判定を隠すために不存在を偽装しない。

### 14.2 CSWを通常開発の上位承認者にせず、増分価値を独立に実証する

CSW（探索・構造的批評のMethod）は、YIC/TEIの既存工程やOwner意味受理を置換しない。認知・研究段階の産物を`Hypothesis → Design Candidate → Validated → Accepted → Implemented → Observed`として区別するが、**この進行表は説明/評価用であり新しいTEI Core Workflow状態・実行許可の正本にはしない**。未検証のCSW仮説をコード/Policy/業務Meaningへ自動昇格させない。

同一Taskで次の対照を設ける。

| 条件 | 導入するもの | 観測する増分 |
| --- | --- | --- |
| **B0** | 通常のAI開発（既存Tool/Skill） | 基準となる発見・実装/テスト可能性・時間・費用 |
| **B1** | 汎用の発想/独立批評Method | B0比の未知Required、反例、誤判断、冗長性 |
| **B2** | CSW固有の構造探索/批評 | B1比の新規で検証可能な構造的発見、正当な保留、実採用効果 |

B0/B1/B2は**HV-0〜4のHarness構成差と別軸**。モデル/Tool/Source/Task・独立Expected・担当者へ見せる情報を揃え、B2だけ追加で非公開Expectedを読める状態にしない。発見数の多さではなく、`独立レビューで採択された実質的に新しい欠落/反例`、`既存重複`、`危険な誤昇格`、`実装・テスト可能性`、`経過時間/Cost/人手`を対照する。CSWの優位が証明されなければ**任意の研究Method**として残し、TEI build/CI/ReviewerのHard Gateにしない。

DSH採用順は**既存Skillの直接利用→標準Skill機構→明確な不足時だけPlugin/薄いAdapter→それでも不足すれば独自Harness検討**。新しいPluginごとにGoal/Tool/送出/OS権限が拡大するかを実測し、宣言上のread-only、別Worktree、モデルの自己評価を安全証拠にしない。

### 14.3 初回H2/H3の実行票を、既存TAI #168の固定条件へ束縛する

[TAI #168](https://github.com/hat47x/tai-pleroma/issues/168)と[Skill適合 #488](https://github.com/hat47x/tai-pleroma/issues/488)の基準を変更せず、**新しく比較適格な四Sessionを事前割付**する。既存のDSH h2-08では実CLI32起動・56 Tool Call/55 Resultを限定観測、H3-A/BではProduct Skill本文5,482 bytesと参照文書7,710 Result bytesの**実Read**があったが、物理的なWorktree越境Read、同条件不備、Reply不完全等のため正式な**比較適格H2/H3=0/4**。歴史的観測を「H2が未実行」「H3 Skillが一度も読まれていない」と書き換えず、正式資格へも昇格させない。

固定4試行の開始前に記録するもの：
`trialGroup / pre-registered H2→H3→H3→H2 order / public Task+hash / independent Expected held outside Model/Tool reach / TEI binary digest / DSH version+effective config / actual OS principal+FS/net policy / source Workspace+root+revision / tool allowlist+result budget / model+sampling / Skill root+raw digest / audience+current grant / output sink coverage / ITT denominator / stop&retry policy`。

最低限の実観測順（実際の新規試行は別Ownerで実施）：

1. **Preflight OS**：公開合成canaryを対象に許可内Source Readが成功し、禁止されたSkill/Expected/Worktree越境Readが**実OSで拒否**されることを正負で観測。OS policy適用失敗や拒否未観測ならTrial本体へ進めず、そのSetupをFAILED/PREFLIGHTとして記録する。Worktree分離だけを物理隔離扱いしない。
2. **Egress/保管**：Model input、`dsh_session_log`追加request field、in-flight/backlog、Session JSONL・連結圧縮ログ、子Process/別Socket/DNS/Telemetry/CI Artifactを**実観測できた範囲だけ**固定。Mock Endpointに届かなかったことを全出口の非送出とせず、未観測出口はunknown。可視公開結果と保護された原Evidenceを分離。
3. **Transport**：同一AttemptのTool request/CLI argv/stdout/stderr/Result parts/endを照合し、`requests ≠ replies`や8,192byte切断のような欠測を`complete`と数えない。外部HarnessのTool failure、モデル不正、OS deny、Product診断、Consumer未受領を別原因にする。
4. **Blind evaluation**：H2とH3の各Runについて回答が実際にReviewerへ到達したこと、Modelが独立Expectedに接触していないことを検証。割付後の失敗・不正モデル応答もITT分母へ残す。事後に成功試行だけ選別しない。H3ではSkill root/referenceの実Readと**判断価値の増分**を別に採点する。
5. **Outcome/再開**：現在Grantの変化や旧RawのResume入力再出現、Source revisionだけの変更を別反例にする。CLI Tool起動/exit0がBusiness Operation成功、SEI継続、実Provider Effectを意味しない。

**P0とP1の順序**：初回比較適格H2の一件成立がP0。4件が成立して初めてH2/H3比較品質を述べる。CSW B0/B1/B2、SEI Question十分性の実Product Probe、EKI実Worker配送、SHO等の実Handoffは**P1以降の並列試験**。これらを最初のH2の追加阻害条件としない。

### 14.4 検収用の最小差分（HQ-42〜49）

| ID | 一条件変更の反証 | Owner／合格上限 |
| --- | --- | --- |
| **HQ-42** | 同じ`R=[]`で対象Universe/取得完了を変更 | SEI #767/#770は合成基準、TEI #2586/#2516は実Read/全件条件。実不存在は未実証 |
| **HQ-43** | 非出力Predicate/Relation occurrenceだけ変更、Resultは不変 | TEI #2634/#2586とSEI #767。旧不存在Claimの再利用を拒否 |
| **HQ-44** | 一件の現行Witnessと旧版/対象外Witnessを対照 | 問いQとRead/Scope/Grantの一致が必要。SEI #767、TEI #2516 |
| **HQ-45** | B0/B1/B2で新規反例の発見/重複/誤昇格を比較 | CSW研究・TAI #422/#488。CSW採用を事前確定しない |
| **HQ-46** | OS隔離下で許可Read成功／禁止Expected Read拒否 | TAI #168/#232、TEI #2515。実OS拒否を前提にH2開始 |
| **HQ-47** | Tool Requestを受けるがReply欠落、または受信途中切断 | TAI #168、TEI #3262/#2509。Tool実起動と完全Consumer Deliveryは別 |
| **HQ-48** | H2→H3→H3→H2の途中失敗・再実行・Goal不能 | ITT分母と事前割付を保持。TAI #168、TEI #2509 |
| **HQ-49** | H3でSkill本文は実読込、独立回答品質は不改善 | Product Skill Loadと採用価値・正式比較資格を別判定。TAI #488/#422 |

**成熟度の区別**：SEI #770の局所34件は合成Methodの限定PASS。TAI既存Tool/Skill Readは歴史的な技術Witness。HQ-42〜49の**新しい実験、CSW B0/B1/B2、比較適格H2/H3四Session、実OS/全Egress拒否、実Provider負の証明、Owner意味受理はNOT RUN**。AI発見→業務採用の新中央認可、第二Query/Method Engine、別Harness Runner/Hostは作らない。


---

<!-- harness-activity-control-delivery-effect-20261112 -->
## 15. 2026-11-12：Activity Control／Delivery／Business Effect／Decisionの完了条件を独立化

### 15.1 他製品の進捗を横断契約へ還元する

11月9日以降のTAI Activity、SEI再開、SHO保存・Readback、YI共有Session、TEI Plugin/Providerの議論から、`Activity completed`という単一Booleanは不十分と判断する。次の**4つの観測面**を分離し、TEI/TAI/SEIの既存API・Receipt・Source参照に投影する。新しい中央Activity Store／Lease Engine／Permission Engineは作らない。

| 面 | 実際に確認すべき事実 | 成功しても保証しないこと |
| --- | --- | --- |
| **Control** | Activity/Attempt identity、Lease/Fence、割付、終了理由、Retry適格性 | Tool Replyの完全受信、業務副作用、判断受理 |
| **Delivery** | Tool Request→実argv/プロセス→stdout/stderr→Tool Reply終端→Harness Consumer実受領、Egressと開示 | 業務Operationの成功、Sourceの鮮度、Effect |
| **Business Operation** | 型付きFunction/Operation、実Provider Read/Write、Rowset/Effect、Host認可、冪等性・実Receipt | Ownerによる意味受理、AI推薦の正当性 |
| **Decision** | SEI Recommendation、根拠Source/Revision、Question/Unknown/Conflict、Owner受理・再開 | Tool Deliveryや業務Effectの成立 |

`Attempt exit=0`、`Decision=allow`、`CLI stdout received`、`Plugin selected`は互いの代替証拠ではない。再実行時はControlの新Attemptと業務Operationの冪等性キーを混同しない。実Effect後にReplyが欠落した場合、単純再実行せず、既存Host/Providerの実Receiptを照合してから再試行可否を判断する。

### 15.2 初回の縦切りを二本に分ける

**P0-D（業務データ経路）**：`order_entry.recent_orders`の実Function→Query/SQL→Provider→実Rowset→Mapper→Data Context→Consumerを**同一Attempt**で追跡し、独立Expected（行集合・順序・NULL・Predicate・粒度・取得完了）と対照する。既存`sales.order.submit`の13 CLI Reply／42 Consumer Controlsは別FunctionのD局所Witnessであり、このP0-Dの成功に流用しない。実DB/Provider/Consumerのどこで未到達かを段階別に表示する。

**P0-T/A（Harness配送・開示）**：一つのread-only Tool Requestを外部Harnessから実CLIへ送信し、実プロセス・完全Reply・Consumer実受領・現在Grant/Disclosure・OS禁止Read拒否を独立に記録する。H2最初の比較適格Sessionが成立するまで、四Sessionの統計的比較やSkill増分価値を成功と主張しない。P0-DとP0-T/Aは並列に進め、互いの完成をHard dependencyにしない。

**P1-H3（保存・再開）**：SEIのSource Index参照→Recommendation保存→resume再読取は局所の実装・観測範囲を確認し、`Question/Unknown/Conflict`の保存と独自Methodの判断改善を別資格にする。SHOのテスト用実ファイル書込み／別Open再読取が成立しても、正式Editor保存・競合・耐久性の証拠ではない。KONは後続保存先候補であり、最初のH2/H3の必須基盤にしない。

### 15.3 Plugin／Session／UI選択の同一性

- `Declared → Selected → Loaded → Invoked → Source Consumed → Delivered → Accepted`を**必要なClaimに応じた観測段階**として扱う。全部を常に直列必須ゲートにしない。実Inspector/HTTP Host/Source Importの局所証拠は、正式CLI/Studio Provider Invoke、実Host認可、Source消費の独立証拠へ自動昇格させない。
- TEI Modeler/SHO/SUI/CSWの選択対象は、行番号や表示名ではなく**qualified identity + Source Revision + selection epoch**としてHarness Sessionへ渡す。UI上で選択解除が起きても、明示的に固定した対象は維持し、ユーザーの再選択／失効／権限変更でのみ更新する。AIへ渡す前にCurrent Disclosureを再検査する。
- Session共有は、製品別のMeaning・Grant・編集正本を一つのHarness内部Stateへ移管することではない。各製品の既存型付きOperation/APIとEvidence参照を通して再解決し、古いSession Raw/Tool Replyが新しいModel Inputへ混入する反例を試験する。
- `RQ-036`のModel MapとVerified BOMの採番衝突は**別契約**として維持し、Owner判断まで統合しない。新たな中央BOM/Registryを作らない。

### 15.4 実行可能な負例（HA-50〜57）

| ID | 一条件だけ変更する反証 | 判定 |
| --- | --- | --- |
| **HA-50** | Control Attempt終了後にTool Reply終端だけ欠落 | Control完了≠Delivery完了。H2不合格 |
| **HA-51** | Provider Effect成立後にReplyが失われ、Harnessが再試行 | 実Receipt/冪等性確認なしの二重Effectを拒否 |
| **HA-52** | `recent_orders`の非表示Predicateのみ変更し、表示Rowsetは同じ | 独立Expected/Query Source RevisionでD再資格化 |
| **HA-53** | `submit`の13/42だけ成功し`recent_orders`実Provider未到達 | P0-Dは未成立のまま |
| **HA-54** | Plugin Selected=BだがInvoked=A、またはSource Consumedが旧Revision | 実Invoke/消費Sourceで失格 |
| **HA-55** | UIの固定選択を視覚的に解除し、別対象をクリック | 明示解除なしでHarness対象が変わらない |
| **HA-56** | Session Resume後にSource/Grantが失効し、旧RawがModel Inputへ再出現 | 再解決/Disclosure失敗として拒否 |
| **HA-57** | SEI Decisionのみ保存しQuestion/Unknown/Conflictが欠落 | H3未解決継続は不合格、Recommendation保存成功とは別 |

**受入順**：HA-50/51はTAI #168／TEI #2509/#2515、HA-52/53はTEI #2586/#3262、HA-54はPlugin #1251/#951、HA-55/56はModeler/SHO/TAI Session、HA-57はSEI既存Reader/WriterのOwnerへ渡す。独立Expected、実Tool/Provider/Host/Consumerの観測がない限りPASSとしない。

**今回の資格**：§15は既存議論の責務分離と受入票の追加。HA-50〜57、`recent_orders`同一Attempt E2E、比較適格H2四Session、正式Editor/Session固定選択、実Effect後Reply欠落の再実行、Owner意味受理は**NOT RUN**。新中央Host/Registry/Runner/Activity Engineを要求しない。


---

<!-- harness-claim-transition-evidence-20261113 -->
## 16. 2026-11-13：完了状態の「昇格」を明示的に禁止する実装順

### 16.1 4つの完了条件を同一フラグへ畳み込まない

§15のControl/Delivery/Business Operation/Decisionは、相互に代入可能な状態ではない。各Consumerが必要とするClaimを宣言し、そのClaimに対応する**観測者・実対象・版・範囲・否定条件**を既存のResult/Receipt/Contextへ紐付ける。実装上の独立Storeや汎用Evidence Engineは要求しない。

| Claim | 最小の独立Witness | Claimを無効化する変更 |
| --- | --- | --- |
| Control finished | Attempt ID・実終了理由・Fence/Leaseの現在値 | 再試行でAttempt変更、Lease失効 |
| Tool delivered | Request/Reply対応、最終part/終端、Consumer受信側の観測 | 途中切断、part欠落、旧Session Raw混入 |
| Operation effected | Function/Provider実Invoke、Host認可、実Effect/Receipt | Provider切替、対象行/Transaction変更、Receipt未取得 |
| Decision accepted | 対象Meaning・Owner・受理範囲・根拠Source Revision | Source/Required/Grant変更、Owner受理撤回 |

特に`Tool delivered=true`を`Operation effected=true`へ昇格させない。Read-only FunctionではEffectの代わりに、**取得対象・Query Predicate・Rowset Extent/Completeness・Consumer実受領**を観測する。書込みのEffect後Reply喪失では、業務冪等性キーとControl Attempt IDを別に記録し、Receipt未確認なら自動再試行を止める。

### 16.2 最初に実装する縦切りを具体的な試験契約にする

**Slice A（H2 read-only）**：同一Taskで`Tool Request → argv → 実CLI stdout/stderr → Tool Reply end → Harness Consumer input`を記録する。許可Sourceの取得と、禁止Sourceへの実OS Read拒否を対照し、現在のDisclosureをConsumer直前に再評価する。単なるCLI exit=0やTool送信側ログだけでは合格しない。

**Slice B（TEI Data Context）**：`order_entry.recent_orders`について`Function → Query/SQL → Provider → Rowset → Mapper → Data Context → Consumer`を一つのAttemptで追跡する。Expectedは実行経路から独立して定義し、Predicate変更・重複行・TOP1・NULL・0件/未取得・部分集合を区別する。最初のSlice AのHard dependencyにはしない。

**Slice C（保存・再開）**：SEI RecommendationのSource/RevisionとQuestion/Unknown/Conflictの保存・再読取を別々に検証する。SHOの保存/再開もSource Identity・実ファイル位置・現在Grantを再解決し、旧Raw再利用を失格にする。KONや新中央Session Storeは必須としない。

### 16.3 統合前の検収票（HB-58〜63）

| ID | 最小の負例 | 必須観測・Owner |
| --- | --- | --- |
| **HB-58** | Tool最終partが欠落するがCLI exit=0 | Consumer受信側終端不在→Delivery未完了。TAI #168/TEI #2509 |
| **HB-59** | Effect成功後Reply喪失、別Attemptが同じOperationを再試行 | 既存Receipt/冪等性で二重Effect防止。TEI Host/Provider Owner |
| **HB-60** | Read-only Rowsetは同じで非表示Predicateだけ変更 | Query Source/Expected再検証。TEI #2586/#3262 |
| **HB-61** | Selected Provider B、Invoked Provider A | Selectedの一致だけではOperation資格なし。TEI #951/#1251 |
| **HB-62** | Source revision一致、Current Grantだけ失効 | Consumer直前のDisclosure拒否。TEI #2515/TAI #168 |
| **HB-63** | Decision保存済み、Unknownだけ保存されない | Decision成功と未解決継続を分離。SEI Owner |

**実装ゲート**：Slice Aの実Consumer受領を最初のH2の合格条件に据える。Slice Bは並列に実証し、Slice CはP1。HB-58〜63は設計上の反証票であり、**実行・PASS未確認**。既存mainのCLI/Plugin/Context/Cache CI成功を、実外部H2・実Provider E2E・SEI再開・Owner受理へ転用しない。追加のRunner、Registry、Proof Store、Permission Engineは作らない。
