# TEI Product Skill：既存CLIへの結合と外部Harnessでの利用を検証する設計

- **状態**：2026-11-02 提案／初期portable Skill文書候補。正式Skill採用、実インストール、実Harness・本番外部Host接続は未実施。
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

2026-11-02の`main`を参照。以下は**ソース静的確認に基づく候補**。各環境で使用する前に現実のコマンドと利用権限を確認する。

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

以下のケースは[TEI #2605](https://github.com/hat47x/tei-platform/issues/2605)本文を正本とし、本計画では試験の順序と実行経路だけを定める。新しいテスト正本/Oracle/別CLIは作らない。

### S0 — 静的契約・資料
- `SKILL.md`のfrontmatter `name: yi-tei-modeling`とディレクトリ名の対応、参照pathの実在、実CLIとの記述整合を静的確認。
- TAIの`yi-tai-activity` SkillやMMIの`skills/mmi`をTEI repoへコピーしない。Product専門機能と横断手順を分ける。
- TAI SK-01のH2（Toolのみ）・H3（Product Skill＋Tool）で**同じ情報条件**を採用する。Skill内に検証用Expectedを埋めない。

### S0b — Product Skillの静的契約点検

Python標準ライブラリのみの`skills/yi-tei-modeling/verify_skill.py`と`tests/test_verify_skill.py`を試作する。実行は`python3 skills/yi-tei-modeling/verify_skill.py`、合成試験は`python3 -m unittest discover -s skills/yi-tei-modeling/tests -v`。

対象はSkill名、Markdown相対リンク、既存read-only Command Catalogと文書内CLI例の整合、Traceページ継続用fingerprintの説明。これは**静的検査であり実TEI CLIの起動・Harness接続・業務意味・実Host権限を検査しない**。孤立した合成fixtureでの7件は**試験コードを保存済み**。以前の「ローカルPASS」は独立の実行ログで裏付けられていなかったため、現時点の試験実績としては撤回する。GitHub Actionsで実行し、対象コミットとログを確認できた時点でPASSへ更新する。

**TS-11〜15追補**（[Issue #2605](https://github.com/hat47x/tei-platform/issues/2605#issuecomment-6572810341)）：発見時CLIと実呼出しCLIの相違、Trace継続tokenとGraph fingerprintの混同、改行差によるSource byte相違、複数readの異版接合、外部開示範囲を、それぞれ別の負例としてS1/S2で検証する。Consumer向けの証拠接合は#2516 W10へ委ねる。

### S1 — TEI CLIを実動で検証
- 同一checkout/Workspaceに対し、`commands`→`model catalog`→`inspect`→`trace`→`validate -json`の順でread-onlyに実行。
- `exit code`と`ok/diagnostics/error`を別に観測。対象SubjectのID、Source所在とcontent fingerprint、Working Tree、実CLIのバージョンを残す。
- 関連する反例としてambiguous、Source更新直後のstale、Warning付き成功、未対応Subject、存在しないCommand、`trace`のscope偽Authorityを固定。
- 生成AIが成功と報告しても、コマンド未実行ならNot-run。CLIのread-only PASSは外部Host・Product Releaseを保証しない。

### S1a — 同一checkout・単一実行物でCLIをread-only実測する

初回S1を実行可能にするため、`.github/workflows/tei-product-skill-qualification.yml`を候補ブランチに追加した。**当該Workflowの実際のCI結果は対象HEADごとに別途観測する**。手順は次の通り。

1. `python3 -m unittest discover -s skills/yi-tei-modeling/tests`で7件の静的チェッカーの正常/負例を実行し、同じcheckoutの`verify_skill.py`で文書・read-only Command Catalog一致を診断する。文書にある宣言が実CLI実行資格と一致することまでは保証しない。
2. `go build -o "$RUNNER_TEMP/tei" ./cmd/tei`で**一つの実行物**を作る。`commands`、`model catalog`、`inspect semantic-data:sales.order`、`trace semantic-relation:sales.order.customer`、`validate -json`を同じ`reference/workspace`に対して順に実行する。JSONを読み取り、要求される`worldFingerprint`と`sourceIndexFingerprint`の一致を比較する。Outputはrunner一時領域だけに保存し、repo/元Sourceを変更しない。
3. 実mainには統合済み[#2587](https://github.com/hat47x/tei-platform/pull/2587)の`operations/artifacts`入口もあるが、`commands`一覧の`selected-read-only-commands`には含まれない。この**非網羅カタログ**と実dispatch・実権限の違いは[Draft #2649](https://github.com/hat47x/tei-platform/pull/2649)がOwnerで、本初期Skillが新操作を宣伝したことにはしない。`artifacts -out`はローカル書込みなのでこのread-only Smokeへ混ぜない。
4. 静的コマンド参照は**探索候補**であり、実ToolのPath/Argv/Source bytes/OS・Runtimeおよび実際にSkillがinstall/loadされ利用された事実と別。`commands`の一覧成功、CLI exit0、Validation Warning有りはOwnerの意味受理・実行許可・情報外部送出・外部Host効果の証拠ではない。

**実施範囲を過大に主張しない**：S1aのCIが成功しても「単一checkout／単一CLIの選んだread-onlyコマンドと7件検査の成功」が確認できるだけ。並行変更のABA、Source move、検証範囲外のSubjectや全Plugin、Role/Operatorの実依存閉包、実Harnessからのload、現在Grantと開示先の適合、外部Consumer受領は未証明。Source relocationの実CLI証拠は別Ownerの[統合済み#2626](https://github.com/hat47x/tei-platform/pull/2626)を引用し同一Attemptへ合算しない。

**S2への新しい引渡し**：[#2638](https://github.com/hat47x/tei-platform/pull/2638)の合成Role/依存Providerのpin識別（RB-20〜24）と[#2639](https://github.com/hat47x/tei-platform/pull/2639)の実Go Language Pluginは、Skillが実在Toolとして何を説明できるかの**上流証拠候補**。Role解釈・Language/Operator版・Source Reader・Tool Discovery・実CLI/Capability・Skill Load・Consumer Disclosure・Host Grantは独立資格。TAI [#271](https://github.com/hat47x/tai-pleroma/issues/271)のH2/H3/異Harness比較は、同じ実行物・入力/Expected・現在資格で行い、結果値が等しいことから外部受信安全を推定しない。

### S2 — Product Skillの増分価値
- TAI #270 SK-01のH0（Harnessのみ）／H2（Toolのみ）／H3（Tool+TEI Skill）に同一の入力を渡し、作業成功・Source対応・未決保持・正しく拒否した誤操作・Tool呼出回数・渡したtoken/内容・人間確認数を比較。
- Skillの文章にだけ業務正答や未公開Sourceを持たせない。差分の有用性をSkillそのものへ因果帰属できるよう、追加資料・Tool能力・権限をそろえる。
- H4/H5（複数Product＋TAI）とH6（別Harness）はTAI側が評価する。TEI単体の合成fixtureを異Harness本番利用へ転用しない。

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

**実証状態**：本計画と試作Skillはドキュメント段階。設計として記載したTS-01〜15、S1〜S3、H0〜H6の実行は行っていない。OwnerによるProduct Skill正式採用、実CLI/実Host/別Harness・実OS情報境界の受入は今後。
