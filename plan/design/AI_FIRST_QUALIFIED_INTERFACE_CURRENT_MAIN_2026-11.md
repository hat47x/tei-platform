# AI-firstインターフェイスの資格分離・実測対応（2026-11-07）

- **文書の性格**：現行mainから実装へ渡す、横断受入の圧縮設計。採用済みADR、各Issueの受入基準、実装・CIの代替ではない。
- **来歴**：古いmainを基点とする[PR #2492](https://github.com/hat47x/tei-platform/pull/2492)のAI-first設計§5.4〜5.11で成熟した結果を、**新しい実観測と現在のOwner**に照合して凝縮移送する。#2492全体（要求・研究・基礎データ設計）の自動代替・無条件Closeではない。
- **今回の範囲**：read-only CLI/Skill/Harness、診断・提案、Source/Target完全性、独立期待値、実行・送出・Resource確認の資格を接続する。Go単一実行バイナリ・既存Shared Acceptance・外部Hostの権限主体を維持し、新Host/Runner/Proof Registry/受理正本を設けない。
- **実証状態**：記載する既存PR/固定HEADの限定実績を採用する。本文の統合手順は**新規NOT RUN／NOT ACCEPTED**。

## 1. AIの三経路と「何を受け入れたか」

| AIの出力 | 許可される操作 | 最終的な資格 |
| --- | --- | --- |
| **意味・設計のProposal** | Source/Role/Rule/Queryの引用付きCandidate・Diffを返す | [ADR-0022](../adr/0022-ai-proposal-canonical-acceptance-boundary.md)・[#267](https://github.com/hat47x/tei-platform/issues/267)のShared Acceptance→実Source保存→再Load。[`validation.Accept`](https://github.com/hat47x/tei-platform/pull/3041)の局所`Accepted=true`や[Studioの未受理案表示 #3120](https://github.com/hat47x/tei-platform/pull/3120)は正式受理ではない |
| **正規定義された計算のParameter Bind** | 選択・Pin済みのComputationへ型付き引数を渡す。計算定義の変更はProposalに戻す | 実Executor Input/Result、Receipt、決定的なAttester、独立Expected、Job/Result再Readの別証拠。[OKF v0.2 §10](https://github.com/GoogleCloudPlatform/open-knowledge-format/blob/main/SPEC.md)は実行ABI・Host・Attester Storeを規定しない。TAI #514の後続read-only PoCで検討 |
| **副作用を持つIntent** | Provider/外部Actionの型付き操作候補 | 現在のPrincipal/Tenant/Target/Operation/Guard、外部Hostの実Grant、Commit/Unknown、Receipt/Resource独立Read。Proposal・Tool選択・`authorizationMode=external`は許可証ではない（#2598/#1174） |

**二つの別資格**：A3＝PoC開発の機械確認による自律進行、A4＝対外に取り消せない効果の人への確認（採用[ADR-0035](../adr/0035-autonomy-levels-and-verification-packets.md)）。これは製品上のOwner Meaning AcceptanceとHost Current Grantとは別の軸。PoC内の局所Go/Docs検証を全部人待ちにしないが、製品側の受理・権限・実効果をCI成功で推測しない。

## 2. AI-first受入の一枚の照合列

ここでいう照合列は新公開Schemaではなく、既存[#1163](https://github.com/hat47x/tei-platform/issues/1163)・[#1180](https://github.com/hat47x/tei-platform/issues/1180)に残すClaim相対のWitnessである。

| 段階 | 実際に裏付けるWitness | 不足時・禁止する推論 |
| --- | --- | --- |
| **S Source** | Workspace/Manifest ID、論理Source IDとraw byte/Revision、選択済みLanguage/Operator/Plugin/Build pin、Required→Discovered→Loaded→Parsed→**Actual Consumed**と共同Snapshot | ファイル名・Catalog・`readByLoader`・`modelRevision`・`validate ok`だけで意味入力が完全とはしない。GOOS/tag選択や参照先が複数時点なら「共同版」未確認 |
| **T Tool/Transport** | 実Harness/Model/Tool Profile、**実公開Allowlist**、Tool schema・mode/successShape、Request/argv/cwd/実Binary digest、stdout/stderr/exit、受領Result bytes・全体性 | `tei commands`にないコマンドもCLIが受理する。発見可能性≠公開資格≠実使用。`effect:read-only`だけで`-execute`のローカル書込み無しとしない |
| **C Candidate** | Attempt ID、Diff、引用Span/Source版、依存Context、Unknown/矛盾、実AudienceとDisclosure、再生成・Rejected履歴 | Source本文・Skill/Tool replyの命令をHost Grantへ昇格しない。欠損・NULL・非開示・未取得・partialを同じfalseへ潰さない |
| **V Independent Expected** | Human/独立根拠のExpected、必要Subject/Field/Predicate/Grain/Scope、正常/既知Mutantの検出、Coverageの根拠 | 同じAIが作ったExpectedによる自己追認、Target Setの件数一致・CallerのCoverageComplete自己申告は不適格。非出力Predicateは信頼できる実Operator評価Evidenceがあればよく、Consumerへの機密列露出を要求しない |
| **M Meaning** | #267の適格Decision/Actor/Scopeと、実Source Patch→Reload→Conformance。人のReview撤回・Source ABAを再資格化 | `validation.Accept`やStudio`accepted`表示、モデルの解説、OKF`verified`、PoCのA3統合を製品意味受理としない |
| **G/E Authority and Effect** | 対象・時点・現在Grant/Required-Supplied-Evaluated Guards、実Provider Result/Commit/Unknown、**receipt-observed**と**resource-observed**、Consumerの実再読取と現在Epoch | 参照用allow-all/list・`external` mode・過去Receiptを現在認可にしない。Receipt存在≠Resource現在値、保存値一致≠業務完了、再開後の旧Attempt Resultを新試行へ転記しない |
| **D Disclosure/Egress** | 入力送信・Tool結果・Session Log拡張・Child Process・CI Artifact・監査結果の、Audienceごとの実Sink/byte/許可範囲・OS拒否点 | `read-only`・CWD分離・設定`enabled:false`・Mockでの非観測から、全FS/Network/Providerの非漏えいを推定しない。秘密由来のDigestや拒否理由も無条件に公開しない |

## 3. 新たに確かめた不足と、既存Ownerへの実装引渡し

### 3.1 外部Harness：技術的接続は実証、比較適格は未成立

[main Evidence #3113](https://github.com/hat47x/tei-platform/pull/3113)はDSH 0.2.2-rc.1／Windows／公開`sales.order`、実Trial `h2-08`でTool call 56、Result 55、TEI実Binary起動32を観測した。**H2-T技術的transportは限定成立**。ただし[TEI #2509](https://github.com/hat47x/tei-platform/issues/2509)の現物点検では、`cli-model-catalog.txt`が8,192 byteで途中終了し、stdout JSONにPowerShell stderrが付属する。Resultを受け取ったというだけでは完全なJSON/必要Sourceの全件受領ではない。Skill/Reviewer期待値・別Worktreeがread-onlyの外に読めてしまい、固定4Session比較**0/4適格**、H3 Skill実Load、独立回答採点・費用比較は未実施。

[main Mock Evidence #3153/#3161](https://github.com/hat47x/tei-platform/pull/3161)は`dsh_session_log`が**既定ON**で約45KB送出、Toolの合成MarkerがモデルInput以外のSession Logの2箇所へ複写、Off→Onや503/Stream切断後に同Watermarkが再送されることをローカルMockで実測。[main #3200](https://github.com/hat47x/tei-platform/pull/3200)では子Tool ProcessでConstrainedLanguageによる一部.NET/CIM操作制限が働き、親の合成Credential様環境変数は継承されず、loopback Mockへの接続はできた。**これはDSHの全OS/FS/Network隔離、実Credential、実Provider Egress無漏出の証拠ではない**。

**次のH2出口**：[TAI #168の固定Runbook](https://github.com/hat47x/tai-pleroma/issues/168)のTask/独立ExpectedとH2→H3→H3→H2割付を保持し、(a)禁制Source/Skill/期待値への実アクセス拒否、(b)実公開Tool allowlistと既定Session LogのAudience・Sinkを観測、(c)stdout/stderr分離・JSON全体性・Result数・警告/Unknown、(d)AnswerとConsumer受信を**新たな一適格Session**で確認する。探索9試行の成功を事後選抜しない。#2509はDSH接続、#2515/TAI #232は実隔離・送出、#2605はSkillの実Byte/Load、TAI #168はTrial比較のOwner。第二Harness Runnerは作らない。

### 3.2 CLIが動くこと・公開してよいこと・業務Actionがあることを分離

[main実測 #3210](https://github.com/hat47x/tei-platform/pull/3210)はCLIが**53コマンドを受理し、`tei commands`がそのうち19のread-only公開選択だけを広告**することを確認。残り34は意図的に広告されない。操作面`tei operations`は実行できても、現在の公開discovery・統一エラーJSON契約へは未参加。[main #3145](https://github.com/hat47x/tei-platform/pull/3145)ではFunction Patternを変えても操作面のderived 8・declared 0は不変。AIがPattern表示から実Action/Provider updateを自動発見・許可してはならない。Operation Surface/Pattern公開の採否は#2507/#951/#2824等の既存Owner。

[Dogfooding Draft #3162](https://github.com/hat47x/tei-platform/pull/3162)で`view effect: read-only`でも`-execute -operation-log`等がローカル状態を書いた。発見契約の`modes/successShape`はmain #3148/#3160で改善されたが、**実オプションごとの副作用・stdout/stderr/消費・Source identity**を別に検査する。[`modelRevision`の厳密化 #3163](https://github.com/hat47x/tei-platform/pull/3163)は管理Source Index版でありWorld/意味の資格ではない。

### 3.3 実Read／Build／実保存のそれぞれの閉包

[Draft #3175](https://github.com/hat47x/tei-platform/pull/3175)の同HEAD局所CIは、Q05同件数異集合、Q06 TOP-1、Q07非出力Predicateについて**観測不足から`holds`を導かない**合成反例を実行した。だが実Provider/Function Readの資格は別（#2586/#2506/#2514）。Consumerに非出力Fieldを露出しない場合でも、SQL/OperatorがそのFieldとPredicateを実評価した証拠があれば有効な証人となりうる。Source選択→実Read Plan→必要行列/時点/結果/受信まで同一Claimで結ぶ。

[実Build観測 Draft #3156](https://github.com/hat47x/tei-platform/pull/3156)では、隔離Go Module供給から`cmd/tei` artifact buildはexit 0だった一方、既存`ResolveSourceClosure`は`rejected`。**Artifact Produced / Source Closure Qualified / Toolchain Trust /実Consumer Read**を別欄にする。Go Module `go.sum`との一致確認[main #3142](https://github.com/hat47x/tei-platform/pull/3142)は供給元の署名・Authorityまで証明しない。外部Module根の方針をAIだけで変更しない（#2831/#1090/#2595）。

[main #3124/#3127/#3131/#3155](https://github.com/hat47x/tei-platform/pull/3155)はReceipt Lookupと別接続のResource現在値観測を別々に扱う。`persistence:not-applicable/not-observed/receipt-observed/resource-observed`、`persisted:true`はResource一致の場合だけ。現Principal/Target/Scope/Owner Meaning/業務Outcomeはさらに別。RunPlan旧Attempt書込み防止[main #3139](https://github.com/hat47x/tei-platform/pull/3139)は局所成功だが原子的な排他保証ではない。

## 4. 実装順と受入の最小Done

| 既存Owner・並列レーン | 次の最小Done | そこで止めるClaim |
| --- | --- | --- |
| **Harness H2-Q**：TAI #168／TEI #2509/#2515/#2605 | 既存固定Taskで**隔離・送出・Tool Result完全性が検証されたH2一Session**。H3実Loadと残り3Sessionは同Groupの後続 | Tool transport ≠ 回答正答 ≠ Skill効果 ≠ Egress非漏えい |
| **Source/Runtime P0-L**：#2586/#2506/#2514 | 旧`*.tei`一Function/Queryから**実Provider read-only結果**とScope/Target/独立Expectedを照合。対象集合がpartialなら局所Claimのみ | `validate ok`・行件数・合成Completion PASS ≠ 実Read完全性 |
| **Authoring P0-M→P0-J**：#3030/#3031/#988/#267 | 新`*.tei.md`の実Reader/Parser/Source-span・Build実消費後、旧形式とは別Revision/Sourceで同一独立Expectedへ合流。固定Candidate→Shared Acceptance→実Reload | suffix／Markdownの未受理文書 ≠ Parser適格、Source更新 ≠ Owner Meaning受理 |
| **Effect P1**：#1174/#2598/#1163 | 実Host current GrantとOperationID/Targetに束縛したCommit/Unknown、ReceiptおよびResource readを別確認し、適格Consumerへ戻す | Receipt再読取だけで`persisted`・業務完了にしない |
| **Build P1並列**：#2831/#1090/#2595 | 選択Module・供給byte・Source閉包・Build artifact・実Consumer inputを別測定 | Artifactができたことから閉包資格、配備資格、再現性を導かない |

すでに実証済みのDSH transportや局所Go/PG結果を同一Attemptの横断PASSへ合算しない。P0-LとP0-MおよびH2実験は相互にHard dependencyを置かず、**一つのSource/Attempt/独立期待値を軸に別資格を接続する**。実験のFixtureと期待値は既存#1180/#2514/TAI Runbookが正本で、新Case群・新Runner・第二Meaning/Host/Policy Storeを増やさない。

## 5. 横断の停止条件と今後の扱い

1. **新しい公開・外送**：公開合成データ以外のSource/Raw/Tool結果/検証用Digestを外部へ出す前に、現在Audienceと実Sink・OS/Network拒否を#2515/#2516で評価。公開可能なEvidenceと内部保護Rawを分ける。未知の送出は`unknown`で維持し、許可されたローカルread-only業務全体を停止しない。
2. **再開・差分**：A→B→A、Path移動、Source read時点不一致、GOOS/tag・Plugin/Operator版の違い、Handoff、旧Responseの到着は当該Claimを再資格化。Checkpointは無条件の再実行命令でない（#958/#2533/TAI #271）。
3. **業務・対外効果**：PoC A3の開発自律性と、外部不可逆効果A4の人への確認を維持。AI Candidateの形式的Validationや`Attested Computation`、Main CI・Mutation kill、参照用Grantから製品業務意味・本番権限・実Outcomeを自動推論しない。
4. **統合前の整合**：PR #2492の現行mainからの大幅乖離を踏まえ、旧PRの全文移植ではなく、実装に必要な成熟Contractだけを本書と[基本計画](../MASTER_PLAN.md)へ移送する。旧PRの研究・基礎データ・要求の残余は各Ownerが別途差分採否を判断。最新mainと[Draft #3179](https://github.com/hat47x/tei-platform/pull/3179)のMASTER_PLAN追補は両立を確認する。ここで旧PRや既存Fixtureの自動Closeはしない。

**未実証**：比較適格H2/H3（0/4）、実Skill Load、非公開データを含む実Egress拒否、全Source/Target閉包を持つ新`*.tei.md` Reader、実Providerの一Function read-only縦断、AI Proposalの正式Source更新一往復、実HostのGrant→業務Outcome、OKF Executor/Attester一往復。実測済みClaimは上記の有限実装・HEAD・環境に限る。
