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

[main Evidence #3113](https://github.com/hat47x/tei-platform/pull/3113)はDSH 0.2.2-rc.1／Windows／公開`sales.order`、実Trial `h2-08`でTool call 56、Result 55、TEI実Binary起動32を観測した。**H2-T技術的transportは限定成立**。ただし[TEI #2509](https://github.com/hat47x/tei-platform/issues/2509)の現物点検では、`cli-model-catalog.txt`が8,192 byteで途中終了し、stdout JSONにPowerShell stderrが付属する。Resultを受け取ったというだけでは完全なJSON/必要Sourceの全件受領ではない。Skill/Reviewer期待値・別Worktreeがread-onlyの外に読めてしまい、**比較適格は0/4**、独立回答採点・費用比較は未実施。**一方、後続の[4 Session原Evidence](https://github.com/hat47x/tei-platform/blob/main/plan/evidence/DSH_H2_H3_FOUR_SESSION_2026-11-07/README.md)ではH2-A→H3-A→H3-B→H2-Bを探索実行し、H3-A/Bで同じProduct Skill本文（各5,482 byte）と参照1件を実Toolから読んだ**限定H3 Load**は成立した**。H2-Aは先行Session再利用で独立試験群ではない。Skill本文のTool-result wrapperを元Markdown Source byteと同一扱いしない。

[main Mock Evidence #3153/#3161](https://github.com/hat47x/tei-platform/pull/3161)は`dsh_session_log`が**既定ON**で約45KB送出、Toolの合成MarkerがモデルInput以外のSession Logの2箇所へ複写、Off→Onや503/Stream切断後に同Watermarkが再送されることをローカルMockで実測。[main #3200](https://github.com/hat47x/tei-platform/pull/3200)では子Tool ProcessでConstrainedLanguageによる一部.NET/CIM操作制限が働き、親の合成Credential様環境変数は継承されず、loopback Mockへの接続はできた。**これはDSHの全OS/FS/Network隔離、実Credential、実Provider Egress無漏出の証拠ではない**。

**次のH2出口**：[TAI #168の固定Runbook](https://github.com/hat47x/tai-pleroma/issues/168)のTask/独立ExpectedとH2→H3→H3→H2割付を保持し、(a)禁制Source/Skill/期待値への実アクセス拒否、(b)実公開Tool allowlistと既定Session LogのAudience・Sinkを観測、(c)stdout/stderr分離・JSON全体性・Result数・警告/Unknown、(d)AnswerとConsumer受信を**新たな一適格Session**で確認する。探索9試行の成功を事後選抜しない。#2509はDSH接続、#2515/TAI #232は実隔離・送出、#2605はSkillの実Byte/Load、TAI #168はTrial比較のOwner。第二Harness Runnerは作らない。



**再開時の二つの流れ**：[実測PR #3237](https://github.com/hat47x/tei-platform/pull/3237)（Open・未統合）の合成Canaryでは、同じSessionを再開すると既ACK区間の`dsh_session_log`に過去Canaryは載らなかった一方、モデル向け`messages`に過去会話が再構成され、Toolを再呼出しする前からCanaryが入った。**Logの非再送≠モデル入力の非再開示**。失効後に再開するときは現在Audience/Policyを`messages`とLog/Tool別に再評価し、過去のRawを自動的に再送可とはしない。これは実Provider/全ネットワーク／Grant失効後再開の合格証拠ではない。

### 3.2 CLIが動くこと・公開してよいこと・業務Actionがあることを分離

[main実測 #3210](https://github.com/hat47x/tei-platform/pull/3210)はCLIが**53コマンドを受理し、`tei commands`がそのうち19のread-only公開選択だけを広告**することを確認。残り34は意図的に広告されない。操作面`tei operations`は実行できても、現在の公開discovery・統一エラーJSON契約へは未参加。[main #3145](https://github.com/hat47x/tei-platform/pull/3145)ではFunction Patternを変えても操作面のderived 8・declared 0は不変。AIがPattern表示から実Action/Provider updateを自動発見・許可してはならない。Operation Surface/Pattern公開の採否は#2507/#951/#2824等の既存Owner。

[Dogfooding Draft #3162](https://github.com/hat47x/tei-platform/pull/3162)で`view effect: read-only`でも`-execute -operation-log`等がローカル状態を書いた。発見契約の`modes/successShape`はmain #3148/#3160で改善されたが、**実オプションごとの副作用・stdout/stderr/消費・Source identity**を別に検査する。[`modelRevision`の厳密化 #3163](https://github.com/hat47x/tei-platform/pull/3163)は管理Source Index版でありWorld/意味の資格ではない。



**Workspace IDと引数契約の実消費**：[実装PR #3247](https://github.com/hat47x/tei-platform/pull/3247)（Open・未統合）は、5つの読み取り応答に宣言されたWorkspace名・manifest format/version/path/SHAを付け、固定Windows Go/実CLIで限定検証した。ただし`name`は作者の任意文字列で一意・不変のIdentityではない。`source-index`の対象なし応答とWorld直接入口に同情報は無く、対象Source/実Read/Consumer epochへ同一性を引継ぐには追加照合が必要。[未決 #3242](https://github.com/hat47x/tei-platform/issues/3242)では`tei commands`の引数型・enum・defaultが機械可読でないため、Skillが`subject-kind`などを推測してしまう問題が残る。Discoveryを充実するか文書＋両方向試験で固定するかは既存CLI Ownerが決める。新Tool全公開や第二契約Schemaをこの文書で要求しない。

### 3.3 実Read／Build／実保存のそれぞれの閉包

[現行main向けDraft #3238](https://github.com/hat47x/tei-platform/pull/3238)は旧#3175のQ00/Q05〜07合成試験を再配置し、**専用PostgreSQLの実Read**を追加した。固定HEAD `469cdbadb1fcf23d65526ec3c5899f4d94e8390b` の[専用CI #42346176936](https://github.com/hat47x/tei-platform/actions/runs/42346176936)では、隔離DB `tei_web_scoped_e2e` 上でTenant Aの独立Expected `{3,9}`に対し`$top=2`→`[[3],[9]]`、`$top=1`→`[[3]]`を実取得・PASS（SKIPなし）と報告されている。**実Rowset対照が一件成立**した点を従来の「実Providerは全てNOT RUN」から更新する。ただし専用Web Referenceの限定Readであり、TEI製品のFunction→Query選択→Provider→Context/Consumerを同じAttemptで検証したわけではない。CallerがTOP1に誤って`CoverageComplete`を与えると局所`holds`になり得るため、Coverage根拠は独立TargetSet/実Readと結び付ける。Q07非出力Predicateは実SQL/Operator評価が観測された範囲だけ根拠とし、Consumerへの非公開Fieldの開示を要求しない。実Product縦断のOwnerは#2586/#2506/#2514。

[実Build観測 Draft #3156](https://github.com/hat47x/tei-platform/pull/3156)では、隔離Go Module供給から`cmd/tei` artifact buildはexit 0だった一方、既存`ResolveSourceClosure`は`rejected`。**Artifact Produced / Source Closure Qualified / Toolchain Trust /実Consumer Read**を別欄にする。Go Module `go.sum`との一致確認[main #3142](https://github.com/hat47x/tei-platform/pull/3142)は供給元の署名・Authorityまで証明しない。外部Module根の方針をAIだけで変更しない（#2831/#1090/#2595）。

[main #3124/#3127/#3131/#3155](https://github.com/hat47x/tei-platform/pull/3155)はReceipt Lookupと別接続のResource現在値観測を別々に扱う。`persistence:not-applicable/not-observed/receipt-observed/resource-observed`、`persisted:true`はResource一致の場合だけ。現Principal/Target/Scope/Owner Meaning/業務Outcomeはさらに別。RunPlan旧Attempt書込み防止[main #3139](https://github.com/hat47x/tei-platform/pull/3139)は局所成功だが原子的な排他保証ではない。

### 3.4 操作候補が古くなったことを実行時に見分ける

[mainに統合された実測 #3254](https://github.com/hat47x/tei-platform/pull/3254)では、固定合成Workspaceに属性1個を追加すると`worldFingerprint`と`modelRevision`は変化したが、`tei operations`の5件と各`basis`は不変だった。`basis`は派生根拠であり、World全体のRevisionではない。`tei operations`には比較相手のRevisionを渡す引数がなく、観測した`period_guard.go`には版の照合がない。従って**古いOperation Surfaceのまま現在WorldでActionを許可できる、というClaimは未成立**。

既存#2598/#1125/#1174のRequest/Dispatch境界で、`proposed action / selected workspace manifest / observed World+Model revision / operation basis / current Target+Period / actual current Host Grant`の**各版の出所と照合先**を別々に記録する。1個の一致しないRevisionで直ちに副作用が起きたとは推定せず、既定でrejectするか・再導出するか・保存済みOperationに版を渡すかは担当Ownerが決める。Recorderや後段Stageが失敗した時のEffect/Receiptは#1174へ残し、Unknownをno-effectに変換しない。新Admission Engine/第二Operation正本を作らない。

## 4. 実装順と受入の最小Done

| 既存Owner・並列レーン | 次の最小Done | そこで止めるClaim |
| --- | --- | --- |
| **Harness H2-Q**：TAI #168／TEI #2509/#2515/#2605 | 既存固定Taskで**隔離・送出・Tool Result完全性が検証されたH2一Session**。H3実Load自体は探索で限定観測済み。**隔離済み新Groupでの再測定**と残り3適格Sessionは後続 | Tool transport ≠ 回答正答 ≠ Skill効果 ≠ Egress非漏えい |
| **Source/Runtime P0-L**：#2586/#2506/#2514 | #3238の専用PG Reference実Readを有限対照として保持し、旧`*.tei`一Function/Queryから**実Provider read-only結果**とScope/Target/独立Expectedを照合。対象集合がpartialなら局所Claimのみ | `validate ok`・行件数・合成Completion PASS ≠ 実Read完全性 |
| **Authoring P0-M→P0-J**：#3030/#3031/#988/#267 | 新`*.tei.md`の実Reader/Parser/Source-span・Build実消費後、旧形式とは別Revision/Sourceで同一独立Expectedへ合流。固定Candidate→Shared Acceptance→実Reload | suffix／Markdownの未受理文書 ≠ Parser適格、Source更新 ≠ Owner Meaning受理 |
| **Effect P1**：#1174/#2598/#1163 | #3254の古いOperation/World版の実行直前照合先を定め、実Host current GrantとOperationID/Targetに束縛したCommit/Unknown、ReceiptおよびResource readを別確認し、適格Consumerへ戻す | Receipt再読取だけで`persisted`・業務完了にしない |
| **Build P1並列**：#2831/#1090/#2595 | 選択Module・供給byte・Source閉包・Build artifact・実Consumer inputを別測定 | Artifactができたことから閉包資格、配備資格、再現性を導かない |

すでに実証済みのDSH transportや局所Go/PG結果を同一Attemptの横断PASSへ合算しない。P0-LとP0-MおよびH2実験は相互にHard dependencyを置かず、**一つのSource/Attempt/独立期待値を軸に別資格を接続する**。実験のFixtureと期待値は既存#1180/#2514/TAI Runbookが正本で、新Case群・新Runner・第二Meaning/Host/Policy Storeを増やさない。

## 5. 横断の停止条件と今後の扱い

1. **新しい公開・外送**：公開合成データ以外のSource/Raw/Tool結果/検証用Digestを外部へ出す前に、現在Audienceと実Sink・OS/Network拒否を#2515/#2516で評価。公開可能なEvidenceと内部保護Rawを分ける。未知の送出は`unknown`で維持し、許可されたローカルread-only業務全体を停止しない。
2. **再開・差分**：A→B→A、Path移動、Source read時点不一致、GOOS/tag・Plugin/Operator版の違い、Handoff、旧Responseの到着は当該Claimを再資格化。Checkpointは無条件の再実行命令でない（#958/#2533/TAI #271）。
3. **業務・対外効果**：PoC A3の開発自律性と、外部不可逆効果A4の人への確認を維持。AI Candidateの形式的Validationや`Attested Computation`、Main CI・Mutation kill、参照用Grantから製品業務意味・本番権限・実Outcomeを自動推論しない。
4. **統合前の整合**：PR #2492の現行mainからの大幅乖離を踏まえ、旧PRの全文移植ではなく、実装に必要な成熟Contractだけを本書と[基本計画](../MASTER_PLAN.md)へ移送する。旧PRの研究・基礎データ・要求の残余は各Ownerが別途差分採否を判断。最新mainと[Draft #3179](https://github.com/hat47x/tei-platform/pull/3179)のMASTER_PLAN追補は両立を確認する。ここで旧PRや既存Fixtureの自動Closeはしない。

**未実証**：比較適格H2/H3（0/4）、適格TrialでのSkill Load/効果・費用採点（探索H3実Loadは限定観測済み）、非公開データを含む実Egress拒否、全Source/Target閉包を持つ新`*.tei.md` Reader、実Providerの一Function read-only縦断、AI Proposalの正式Source更新一往復、実HostのGrant→業務Outcome、OKF Executor/Attester一往復。実測済みClaimは上記の有限実装・HEAD・環境に限る。


## 6. Design Context取得から「未カバーテスト一件」を生む最短の実証（2026-11-07追加）

### 6.1 P-1aとT0は実証範囲を限定して採用し、P-1b/T1を次の差分とする

[Design Context QueryのOwner #3262](https://github.com/hat47x/tei-platform/issues/3262)が定めた**「テスト生成に先立ち、生成AIも人間も迷わず必要設計情報・欠落を取得できること」**を、初回テスト生成の前提とする。対象を無条件の全Workspaceに広げず、**一つの利用者Subjectと用途**について独立に定めたRequired設計入力が、実CLI→実Loaded/Parsed→実Consumer受領まで到達したかを診断する。

| 段階 | 確認済みの成果 | 次の資格と現時点のGap |
| --- | --- | --- |
| **P-1a 現行CLIによる読取り** | [main統合 #3264](https://github.com/hat47x/tei-platform/pull/3264)は同一実Go CLIから9読取り応答、`semantic-data:sales.order`のWorld/Source Index束縛、指定`semantic/sales/order.tei`の`readByLoader:true`、Source欠落/異版/未読込の負例DF-02/03/09を固定CIで検証。[最終HEAD専用CI #42357451983](https://github.com/hat47x/tei-platform/actions/runs/42357451983)は限定成功 | **Claim=partial**。Data以外の全Function/Business/Rule/Procedure/Requirement、Parser意味解釈・全必要Dependency、実Consumerへの分割完遂を認定しない |
| **T0 固定済みScenario判別** | [main統合 #3266](https://github.com/hat47x/tei-platform/pull/3266)の実Binaryからpinned Scenario **6 Step**、実CLIの不正データ/期待値/Fixture/Plan **4負例**＋独立義務**2負例**を検出。[最終HEAD CI #42359069285](https://github.com/hat47x/tei-platform/actions/runs/42359069285)限定成功。Go statement C0相当**10.1%は一テスト選択の局所値**。[main #3268](https://github.com/hat47x/tei-platform/pull/3268)はSource/Scenario/Skill/依存変更によるテスト再起動の`paths`対象を拡充 | 既知の検証対象を**実行し判別**した証拠であり、Issueから新規Caseを設計した証拠ではない。**C1未測定、全Requirement/NFR Coverage未測定**。PR仮統合Runとmain push Runは別 |
| **P-1b Design Context整形** | #3262にScope・Source/Role・Required/Observed・欠落理由・連番/継続/End Marker・Audience別開示の設計候補あり | **未実装/未実証**。実CLIの9応答を単にAIが要約したことを、欠落のない一つのDesign Context Product応答としない。新CLI名を実在と偽らない |
| **T1 未カバー領域から新しい負例を提案** | 既存[#1180](https://github.com/hat47x/tei-platform/issues/1180) Scenario／[#2514](https://github.com/hat47x/tei-platform/issues/2514) 独立Expectedと、[#3223](https://github.com/hat47x/tei-platform/issues/3223) Pattern候補を再利用。[#3273](https://github.com/hat47x/tei-platform/issues/3273)はT0からの次の負例選択を計画 | **新しいAI/人間の候補生成と独立判別はNOT RUN**。既存T0の既知6 Stepを再実行しただけでT1達成とはしない |

**次の最小受入（1 Rule、1 Goal、1新負例）**：公開`sales.order`の「行番号は正」のRuleを選び、Design Contextへ必要Source/Rule/Relation/独立要求を**実Loaderから**返す。既存Scenarioに無い「行番号=0」の候補を、AIまたは人が**最小の入力操作で**提案できるか測る。候補自身が`Expected`を自己定義しただけでは採点せず、AIへ非公開の**事前固定独立期待結果**と既知Mutant/負例で、実CLI結果・ルール選択・拒否理由を照合する。欠落Rule・未取得Source・非出力Predicate・異Tenant/Scopeなら`incomplete/unverified`、実選択/出力範囲が不足なら`partial`として、検証可能領域だけにCandidateを出す。実装側で新しいDriver Scenario正本／Test Runner／Expected Storeを作らない。

**同じCoverageを混ぜない**：①対象Domain/Requirementの列挙Coverage、②Source→Required/Observedの取得Coverage、③Scenario義務Coverage、④既知Mutantに対する判別力、⑤Go C0/C1、⑥Workflow起動Coverage、⑦実CLI/Provider/Harnessの受領Coverageは別分母。初回T1でC0/C1やNFR全体の達成率を見せず、**未測定・未取得・未実装を0%やPASSに変換しない**。測定項目＝AIの必要Tool呼出し数、迷った選択数、実Bytes/分割/切断、作成時間、Test Candidate重複、誤合格/誤棄却、最小反例、既知Mutation結果、Token/費用・人手。

### 6.2 三要素Diagram・Function表現・テストの対応は同一Sourceから導出

[三要素図面・CRUD計画Draft #3263](https://github.com/hat47x/tei-platform/pull/3263)および標準Data構造／標準表現型の議論は、Design ContextのRequired Data/Function/Business、Role・多重度・Rule/Procedure・Projectionを**同じSourceの別表現**として扱うために接続する。表示を持つ表現型InstanceはFunction文脈に配置し、そのBindingの意味・Axis/単位・粒度・NULL/欠落を評価する。Diagram/CRUDで見つかる検証義務（例えばRelationshipの1:N、FunctionのCRUD、Businessの実承認）をテストPattern候補へ投影しても、**図の編集・AI表現の変更は第二Meaning Sourceにしない**。SHOの自然なMarkdown図はDocument Consumerであり、TEI図の正本/実CRUDを代用しない。初回T1のためのStudio Renderer・SHO Writer実装をHard dependencyにしない。

### 6.3 Pluginの検証済み組合せとSnapshotは後続の共通資格として再利用

[製品要求 #3267](https://github.com/hat47x/tei-platform/issues/3267)はVerified Composition BOMとVersion Matrixを標準能力に位置付け、[Draft #3269](https://github.com/hat47x/tei-platform/pull/3269)は**渡されたexact Lock**の2×2等をread-onlyで列挙し、pairwise Coverageを決定論的に説明するReferenceを実装した。ただし`qualified:false`固定で、**実Resolver・実Builder/Provider・実独立Test・公式BOM発行は未達**。パターン候補、resolved lock、実Artifact、テスト実証、Qualified tuple、SBOM部品一覧を混同しない。pairwise合格から未試験の三者相互作用をPASSにしない。

[#3271](https://github.com/hat47x/tei-platform/issues/3271)の派生Snapshot/Cache Provider構想は、同一Source/Plugin Pin/実消費閉包と現在Disclosureの資格再利用を検討する後続横断。**Cache hit・Artifact byte一致・過去BOM・過去Owner承認を、Current Grant・検証済み意味・現在のProvider全件Snapshotへ昇格しない**。P0 Design Context/T1/H2 Trialを、BOM Matrix/Cache Backend導入待ちにしない。#951/#1016/#2600/#1180/#2514既存SSOTへ検証結果を接続するだけとし、Pluginの公開Open/Stable/Internal保証と実Tenant開示/Host Permissionも別にする。

**この章の適用範囲**：main統合された#3264/#3266/#3268の限定実行結果を計画へ反映しただけで、P-1b、新T1テスト生成、Function/Businessの全Required閉包、実Modeler/SHO/Plugin Matrixの動作、正式Meaning/Provider Business E2Eは**今回NOT RUN**。現在mainの証拠と本PRの統合時Headを別に照合する。
