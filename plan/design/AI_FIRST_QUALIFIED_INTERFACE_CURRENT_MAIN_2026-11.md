# AI-firstインターフェイスの資格分離・実測対応（2026-11-07）

- **文書の性格**：現行mainから実装へ渡す、横断受入の圧縮設計。採用済みADR、各Issueの受入基準、実装・CIの代替ではない。
> **2026-11-08 現行mainへの移送**：本文§1〜9は[旧PR #3241](https://github.com/hat47x/tei-platform/pull/3241)の時点別受入設計を保持する。**実装状態・次の作業の現行判断は§10を優先**。例えば§9の「TT-56二候補の統合待ち」は[main統合 #3354](https://github.com/hat47x/tei-platform/pull/3354)で解決済みであり、旧候補[#3324](https://github.com/hat47x/tei-platform/pull/3324)・[#3326](https://github.com/hat47x/tei-platform/pull/3326)はclosed/unmerged。ここに記す過去のRunとDraftは当時のEvidenceであって現mainの未実装一覧ではない。本文の設計はIssue・ADR・Runbook/現行実装を上書きしない。


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
| **T1-A 既知の新しい境界を実CLIで判別** | [main統合 #3304](https://github.com/hat47x/tei-platform/pull/3304)の専用CI #42378033965（**旧HEAD**）で、既存6 Stepを維持した一時第7 Stepに行番号0を渡し、期待値・Fixture・古いPinの3負例を実CLIで判別した。採用済みScenario正本は未変更。新HEAD／main pushのCIは別資格 | **既提案Ruleの具体例の実行＝限定T1-A**。AIが未知のRule/Caseを独自に発見したこと、独立Holdoutにアクセスせず盲検生成したことは示さない。Design Contextはpartial、C1未測定 |

**次の最小受入（1 Rule、1 Goal、1新負例）**：公開`sales.order`の「行番号は正」のRuleを選び、Design Contextへ必要Source/Rule/Relation/独立要求を**実Loaderから**返す。既存Scenarioに無い「行番号=0」の候補を、AIまたは人が**最小の入力操作で**提案できるか測る。候補自身が`Expected`を自己定義しただけでは採点せず、AIへ非公開の**事前固定独立期待結果**と既知Mutant/負例で、実CLI結果・ルール選択・拒否理由を照合する。欠落Rule・未取得Source・非出力Predicate・異Tenant/Scopeなら`incomplete/unverified`、実選択/出力範囲が不足なら`partial`として、検証可能領域だけにCandidateを出す。実装側で新しいDriver Scenario正本／Test Runner／Expected Storeを作らない。

**同じCoverageを混ぜない**：①対象Domain/Requirementの列挙Coverage、②Source→Required/Observedの取得Coverage、③Scenario義務Coverage、④既知Mutantに対する判別力、⑤Go C0/C1、⑥Workflow起動Coverage、⑦実CLI/Provider/Harnessの受領Coverageは別分母。初回T1でC0/C1やNFR全体の達成率を見せず、**未測定・未取得・未実装を0%やPASSに変換しない**。測定項目＝AIの必要Tool呼出し数、迷った選択数、実Bytes/分割/切断、作成時間、Test Candidate重複、誤合格/誤棄却、最小反例、既知Mutation結果、Token/費用・人手。

### 6.2 三要素Diagram・Function表現・テストの対応は同一Sourceから導出

[三要素図面・CRUD計画Draft #3263](https://github.com/hat47x/tei-platform/pull/3263)および標準Data構造／標準表現型の議論は、Design ContextのRequired Data/Function/Business、Role・多重度・Rule/Procedure・Projectionを**同じSourceの別表現**として扱うために接続する。表示を持つ表現型InstanceはFunction文脈に配置し、そのBindingの意味・Axis/単位・粒度・NULL/欠落を評価する。Diagram/CRUDで見つかる検証義務（例えばRelationshipの1:N、FunctionのCRUD、Businessの実承認）をテストPattern候補へ投影しても、**図の編集・AI表現の変更は第二Meaning Sourceにしない**。SHOの自然なMarkdown図はDocument Consumerであり、TEI図の正本/実CRUDを代用しない。初回T1のためのStudio Renderer・SHO Writer実装をHard dependencyにしない。

### 6.3 Pluginの検証済み組合せとSnapshotは後続の共通資格として再利用

[製品要求 #3267](https://github.com/hat47x/tei-platform/issues/3267)はVerified Composition BOMとVersion Matrixを標準能力に位置付け、[Draft #3269](https://github.com/hat47x/tei-platform/pull/3269)は**渡されたexact Lock**の2×2等をread-onlyで列挙し、pairwise Coverageを決定論的に説明するReferenceを実装した。ただし`qualified:false`固定で、**実Resolver・実Builder/Provider・実独立Test・公式BOM発行は未達**。パターン候補、resolved lock、実Artifact、テスト実証、Qualified tuple、SBOM部品一覧を混同しない。pairwise合格から未試験の三者相互作用をPASSにしない。

[#3271](https://github.com/hat47x/tei-platform/issues/3271)の派生Snapshot/Cache Provider構想は、同一Source/Plugin Pin/実消費閉包と現在Disclosureの資格再利用を検討する後続横断。**Cache hit・Artifact byte一致・過去BOM・過去Owner承認を、Current Grant・検証済み意味・現在のProvider全件Snapshotへ昇格しない**。P0 Design Context/T1/H2 Trialを、BOM Matrix/Cache Backend導入待ちにしない。#951/#1016/#2600/#1180/#2514既存SSOTへ検証結果を接続するだけとし、Pluginの公開Open/Stable/Internal保証と実Tenant開示/Host Permissionも別にする。

**この章の適用範囲（更新）**：#3264/#3266/#3268に加え、mainの#3296（Context負例16）、#3299（Source/Lock失効）、#3304（既知の第7 Step）という**限定成果**がある。P-1bの完成、盲検T1-Bの未知条件発見、Function/Business全Required閉包、実Modeler/SHO/Plugin Matrix、正式Meaning/Provider Business E2Eは**NOT RUN／未適格**。各CIの実HEADと本PR／mainの試行を同一視しない。


## 7. Design Context→T1-B盲検発見とPlugin実消費へ進める最短接続（2026-11-08）

### 7.1 既実装を再認定し、残る一辺を限定する

前章の§6.1は現在の実装に合わせてT1-Aへ修正した。**新しいCaseを一件実行できること**と、**AIが未カバー条件を自ら発見すること**は別のClaimである。

| 接続 | 現在確認できる有限実績 | 次に残る一辺とOwner |
| --- | --- | --- |
| **Context取得** | [main統合 #3296](https://github.com/hat47x/tei-platform/pull/3296)は9件の実CLI応答を使い、既存負例7＋Manifest欠落2＋Tool Reply byte改変7の**16件の合成拒否**を専用CI #42371310577で実検出した。ただし実Harnessから途中切断された応答そのものは使用せず、一時コピーの改変である | [#3262](https://github.com/hat47x/tei-platform/issues/3262) P-1bのRequired Domain/Rule/Function/Source・Dependency Closure・Response全体性・Audienceの共通整形を**実Consumerへ配線**する。現Contextは`partial`、`qualifiedTestGeneration=false` |
| **Source/Lockの再資格化** | [main統合 #3299](https://github.com/hat47x/tei-platform/pull/3299)はSource/Plugin Lockの実byte変更後に旧Pinned Scenarioを拒否する。A→B→AでContentが戻る対照も確認した | [#3262](https://github.com/hat47x/tei-platform/issues/3262)/[#2600](https://github.com/hat47x/tei-platform/issues/2600)で、最新の選択Source/Required ClosureとPlugin**実Load・Operator/Build入力**を区別する。Content pinの復帰≠過去Decision/現在Grant/歴史の不存在 |
| **T1-A既知候補実走** | [main統合 #3304](https://github.com/hat47x/tei-platform/pull/3304)の旧HEAD専用CI #42378033965は、行番号1正常／0違反、独立Expected、偽Expected／偽Fixture／旧Pinの3負例を実CLIで検出した。**Scenario正本へ第7 Stepを自動採用していない** | [#3273](https://github.com/hat47x/tei-platform/issues/3273)／[#1180](https://github.com/hat47x/tei-platform/issues/1180)の既存受入票へ、Candidateの出所と採用を分離して記録。既提案を実行しただけなので**未知RuleのAI発見は未実証** |
| **T1-B盲検新規発見** | **未実施**。既存AI追加ScenarioのMutant Kill #3017/#3128とも別 | AIへ未カバーRule一覧・正答・期待違反条件・Mutantの内容を露出せず、独立事前固定のCoverage Gap/Expectedに照らして一つの**未予告Candidate**を生成・検証する。#2514がExpectedと判別、#1180がScenario受入、#1163が同改訂Evidence。新Generator/Runner/Oracle Storageは作らない |

**重要な非昇格**：合成Tool Reply byteを改変して拒否できた事実は、DSHの実返却8,192-byte切詰め/56 Call対55 Resultの実検出・全Span再構成を保証しない。**T1-B実行条件はP-1bが`partial`の間は、実観測できたScopeへ限定して候補を生成し、Required不足に関する総合合格を返さない**。C0 10.1%は局所Go statement測定であり、C1・要件/非機能Coverageの分母とは別。

### 7.2 初回の盲検試験は既存Fixture・Test能力の上で一度だけ

[Draft #3273](https://github.com/hat47x/tei-platform/pull/3273)と[テスト専門製品比較 Draft #3223](https://github.com/hat47x/tei-platform/pull/3223)の既存テストPatternを増殖させず、以下を**単一の実験手順**とする。

1. **Reviewerが先に固定する**：既存公開合成Modelの対象一Function/一Rule、選択SourceとRule ID、Required Input・Coverage Gap、独立Expectedと既知Mutant、実Binary/CLIの受入可能Option、公開許可範囲。期待値はAI参加者・Tool/Skill FileSystemから実アクセス拒否し、後から差し替えない。
2. **AIに与える**：#3262の読取り専用Design Contextで実取得済みの説明・出典Span・`required/observed/missing`・Unknown・対象版だけ。既提案の「行番号0」という正解の文字列をCandidate選択のヒントとして露出しない（そのケースはT1-Aとして別票で保持）。
3. **AIに要求する**：未カバーの検証義務を一つ選択し、Source citationと必要な条件・Case Candidateを記述させる。実行可能なScenario変換には#1180の既存契約を使用。候補の期待結果をAIが記述しても**独立Expectedへ流用しない**。
4. **Reviewer/Runnerで判定する**：入力Diffを実Source/Scenario Pinで固定し、実CLIで正常/不正の一対と既知Mutantを実行。AIが観測していない対象・誤って推測した関係・空集合・非出力Predicate・古いSourceを発見したら`unverified`や`rejected`を独立に記録する。
5. **同一Attemptで測る**：Model/Skill/Tool版、Tool呼出回数、実argv・read Scope、Candidateの新規性、Required/Consumed、独立判別、C0/C1（未測定は未測定）、警告/Unknown、消費時間/費用、再試行・Consumerの実受領。**候補を発見したが試験不可能**と**実行したが新規ではない**を別状態に残す。

**停止線**：この手順はまだ**NOT RUN**。T1-Aの成功をT1-Bの予行演習の証拠にはできるが、T1-B達成と表示しない。実外部Harness/H3 Skill Loadの本番適格試験をT1-Bの代替にせず、機密のHoldoutを外送しない。

### 7.3 Plugin-firstの実機能を次のConsumerとして扱う

[main統合 #3277](https://github.com/hat47x/tei-platform/pull/3277)は、開発用CLI/Studioと生成Runtimeを**同じPlugin Contract/Provider/Resolver/Build/Testで利用する**方針を製品要求へ追加した。これは設計/要件であり、現CLIが全機能でPlugin-firstへ移行した証拠ではない。最小の次段は[Inspector選択案 #3292](https://github.com/hat47x/tei-platform/pull/3292)の**一つのread-only Inspector**で、`Workspace選択→Source/Pin/Plugin候補→Resolver選択→実Go Build/Link→CLIとStudioから同じProviderの**実Invoke**→同一Resultの独立判別`。選択Bでも実呼出が旧Aなら不合格。**二つのSurface**と**二Provider×二Consumerの4版組合せ**は別軸にし、組合せの提供だけでVerified BOMとはしない。

[main統合 #3287](https://github.com/hat47x/tei-platform/pull/3287)のWorkbenchは標準Data/Function/Presentationの異種Sourceを試すFixtureとGo局所判別の足場まで。実JSONL/DB Provider読取、CLI/Studio共通Invoke、実生成アプリの同一Contract資格は未実施。**初回T1-BやH2比較のHard dependencyにしない**。標準業務Roleの[main #3308](https://github.com/hat47x/tei-platform/pull/3308)型照合と[main #3318](https://github.com/hat47x/tei-platform/pull/3318)の休日/税率/為替/採番Data例は後続のUnit/Date/Amount/NFRケースに再利用するが、直接与えた行と実DB Readerの結果を区別する。

[Verified BOM Draft #3297](https://github.com/hat47x/tei-platform/pull/3297)はRQ-035（Workbench）とRQ-036（BOM）を分け、2×2組合せとWitness照合を持つが、**Draft／この設計の参照時点でexact-head CI未確定**。`qualified:false`の表を正式BOM／Plugin実Load合格と表示しない。#3297の設計索引・PAS・Gap Auditが未統合の間、RQ-036を別の番号で再起票したり本PRに似たTrace表を再実装しない。

**Tool公開と不作用の現実的な注意**：[main #3210](https://github.com/hat47x/tei-platform/pull/3210)のCLIは53コマンド受理／discovery公開19。[`view -execute`除外案 Draft #3302](https://github.com/hat47x/tei-platform/pull/3302)の`excludedFlagNames`は提案段階であり、公開Tool側の実argv Filter/OS拒否を置換しない。Workspace名・manifest版/照合値を応答へ付与する[#3315](https://github.com/hat47x/tei-platform/pull/3315)は**closed/unmerged**で、現mainの全CLI出力に存在するものと仮定しない。現実のSource Identity/Closure/Consumer Scopeが不足する限り`partial`を保持する。

### 7.4 依存と担当を固定し、同じ実証を何度も起こさない

- **P0**：#3262 Design Contextの実Required/Observed、#2514／#1180の独立Expected/ScenarioのT1-B候補1件、TAI #168の適格H2一Session、#2586の実Provider read-only一Functionは**並列**。互いの不完成を全体停止にしない。
- **P1**：#951/#1251のread-only Inspector Plugin実消費とCLI/Studio二Consumer、#3267/#3297の2×2 BOM tuple、#3270 Workbench異種Source実Providerを、同じInput/Plugin Pin/Consumerの差分で段階検証。
- **P2**：#3030 `*.tei.md` Reader移行、#267 Shared Acceptance、#1174のHost Effect/Receipt/Resource、#1119の横断検収を実Source/実Targetが接続した範囲で拡張。表示・図面/SHO・事前Cache/BOMの完成をP0のGateにしない。

PoCのコード/合成Fixture/内部CIは採用済みADR-0035の自律開発区分で進め、外部への不可逆効果だけA4で人に確認する。**業務意味のOwner Acceptance**と**HostのCurrent Grant**はなお別の製品資格。元Source byteと読取時点、独立Expected、Plugin/Moduleの版・実Build入力、実Consumer、開示Audienceを常に区別し、同一Attempt/同一時点で確認していない組合せを`qualified`にしない。


## 8. 必要Rule閉包と実Tool副作用を別資格とし、T1-Bの入口を狭く実証する（2026-11-08）

### 8.1 正常JSONの「取得できた」は必要Ruleの「揃った」と異なる

[Draft #3324](https://github.com/hat47x/tei-platform/pull/3324) は、既存[main #3296](https://github.com/hat47x/tei-platform/pull/3296)の9実CLI応答・16合成負例に**`inspect-validation`を加えた実10応答**で、`validation.tei`の実Loader読取、Rule/RelationのIdentityと非曖昧性、`positive`検証のCanonical Attribute/Source Anchor、Source Indexの`role=definition`・実Path/FPを比較する差分を実装候補として保持する。正常JSON・World/Index FP一致でも、Rule/Relationだけ省略・複製・別Attribute・Anchor不一致・Loader未読取・異版Indexなら不足。**Tool Replyの一時コピーに作った8負例**は実Harness配送の欠落や実Providerからの全件取得を検証したことにはならない。

| 個別の受入資格 | 必須の実観測 | 欠落・陳腐化したとき |
| --- | --- | --- |
| **Protocol/Delivery** | Tool request/response ID・実Byte全体・Schema、ページ順・終了、Model/Consumerの実受領 | parse success、先頭digest一致、`Tool result`件数だけでは`unverified`。#2509/TAI #168が実Harness境界 |
| **Source/Meaning Closure** | 対象一Ruleの独立Required、宣言Source/Role/Relation→実Loader-read→Parser/Canonical Rule/Attribute/Anchor→実Consumed byteと同じ版 | `validate ok`/空Catalog/同一World FPで代用しない。#3324はこのうちRule単位の局所照合を提案するのみ |
| **Test Candidate Admission** | 当該Ruleの必要Source・前提・独立期待結果・Known Mutant・既存Scenarioとの重複比較 | `qualifiedTestGeneration=false`は維持。**狭いScopeの手動/探索Candidate評価**と、欠落ゼロの自動生成資格を混同しない |
| **Action/Disclosure** | Tool Profileで許した正確なInvocation Variantと実argv、実File/Network/Log効果、Current Audience/Grant | Command名や`effect:read-only`から無書込み・無送出を推論しない。非公開資料に対する拒否がない限りT1-B Blind Holdoutは未適格 |

[#3262](https://github.com/hat47x/tei-platform/issues/3262)をDesign ContextのRequired/Observed/不足理由・返却/受領のOwner、[#1180](https://github.com/hat47x/tei-platform/issues/1180)/[#2514](https://github.com/hat47x/tei-platform/issues/2514)をCandidate/独立Expected/判別のOwnerとし、**T1-Bは二段階**とする。(a) Source/Rule Closureが実観測できた狭い一領域について、既知行番号0のような答えをAIに伏せて候補の新規性を測る、(b) 参加者が独立Expected・Required Gap・既知MutantへTool経由で到達できない実OS/Harness拒否と、実Consumerの全Byte受領を測って初めて**盲検適格**にする。aができてもbの未達は合算しない。既に提案済みの行番号0を「AIによる未知条件発見」と計上せず、常時Rejectや誤Expectedの自動合格を禁止する。

[テスト基本設計 Draft #3223](https://github.com/hat47x/tei-platform/pull/3223) のTT-73〜80/81〜88を**追加のテスト正本として複製せず**、既知Case焼直し・偽Mutant kill・C1のC0代用・未知BOMを合格とする誤判定を、この初回実証の独立負例に再利用する。C0相当の局所数値とC1・要件/NFR Coverage・Tool受領・実Rule Closureはすべて異なる分母。未知母集団や欠落Dependencyから`100%`を出さない。

**現在の証拠更新（2026-11-08）**：#3324の固定HEAD `4f4b114a` は[専用CI #42385606408](https://github.com/hat47x/tei-platform/actions/runs/42385606408)、Repository Health #42385606448、依存チェック #42385606458 がいずれも completed/success。別Draft #3326の固定HEAD `c2d2da10` も[専用CI #42385726717](https://github.com/hat47x/tei-platform/actions/runs/42385726717)、Health #42385726733、依存 #42385726722 がいずれも completed/success。**両方式の統合はNOT RUN／両Draftは未マージ**、`qualifiedTestGeneration=false`を維持。#3304の既知T1-Aはmain統合済みだが、**T1-B盲検発見/独立採点、P-1b完全投影、実Harness配送の全件・禁制資料拒否はNOT RUN**。

### 8.2 実CLIのwriteFlagsとOS隔離を連結して検証する

[main統合 #3333](https://github.com/hat47x/tei-platform/pull/3333) は`tei.cli.discovery/v0`の互換な追加欄`writeFlags`に、`view`の`-operation-log`・`-lifecycle-state`を明記した。Goの限定試験で`view -execute`と指定Flagによる実ローカル書込み、指定Flagなしの対照、Discoveryの他Entryに誤記載されないことを確認した。`effect:read-only`は**Canonical Meaningを書き換えない**という限定表現であり、「ローカルファイルも通信も変更しない」という意味ではない。旧Draft[#3302](https://github.com/hat47x/tei-platform/pull/3302)の`excludedFlagNames`をmainの現行契約と混同しない。

**H2/T1-B向け具体的採否**：`tei commands`から安全なread-only Profileを導出する際には`writeFlags`の有無だけでなく、許可するCommand＋Flag集合・完全argv・実File/Process/Network/Session Log・実Tool Resultの受領を照合する。初回は`view -execute`等不要な効果を持つInvocation Variantを参加者Toolから除外し、OS側で禁止Pathと独立Holdoutの拒否を試験する。CLIが広告するVariantだけを禁止しても、Shell/別Tool/子Processが同じFileへ書けない証明にはならない。実OS/Egressは#2515/TAI #232、Tool公開は#2509/#2605/#951、現在Host Grantは#1174で別資格。

[main #3310](https://github.com/hat47x/tei-platform/pull/3310)はAccess Chain v2にRecord全欄のJSONとchainVersionを含めることで、欄追加漏れ・規則版誤認を減らしたが、ハッシュ鎖を**第三者不変性やAuthoritative Permission、現在の外部Egress拒否**へ昇格させない。外部にPinしたHead、観測者Identity、許可された公開投影は別。

### 8.3 Consumerで「表示された標準Data」を実Providerでの利用と混同しない

[main #3334](https://github.com/hat47x/tei-platform/pull/3334) は標準業務Data（休日・税・為替・採番）の`ApplicationView.standardBusiness`とStudio表示を限定実装した。これは[main #3308/#3318](https://github.com/hat47x/tei-platform/pull/3318)のRole/Primitive検証と例Workspaceへの宣言を**Application Consumerへ投影**した実績であり、行データの取得・税率計算・実DB Bind・Wails RPC E2E・本番Hostでの作用はまだ示さない。

Plugin-first Inspectorの[D1 Draft #3292](https://github.com/hat47x/tei-platform/pull/3292)では、`selected Plugin/Provider→actual loaded/linked/invoked Provider→CLI/Studio Consumer`と、UI上のModel Projectionだけを別票とする。値が同じでもBを選択してAを実呼出しした場合は不合格。複数Surfaceの表示が成立しても、実Provider行の完全性、Function Data Context、現在Tenant/Scopeは[#2586](https://github.com/hat47x/tei-platform/issues/2586)の実Read試験に戻す。対応する[Studio ABA Draft #3327](https://github.com/hat47x/tei-platform/pull/3327)の2件の局所Node試験は、A→B→A後の**古いUI応答の不採用**に限定成功であり、実Provider取消・新Consumer資格・Host Effectを保証しない。

### 8.4 次の一回の実行順

- **P0-A（先行できる）**：#3262/#3324の狭い一Ruleで、独立RequiredとRule/Relation/Source Anchorの実Read一致・不一致を実現し、欠落理由を返す。Draft #3324のexact-head Go/CI・現mainとのDiff/実Toolの受信を別判定。単に既知の合成8負例がPASSしただけで`qualifiedTestGeneration=true`にしない。
- **P0-B（並行準備）**：#2509/#2515/TAI #168で実OS/HarnessのHoldout拒否・`writeFlags`/完全argvと全Tool Reply byte/Session Log Sinkを照合。これが不成立なら盲検資格は保留するが、公開合成DataによるローカルT1-AやP-1b研究は止めない。
- **P0-C（P0-AとBに適格証拠がある範囲だけ）**：#3273/#1180/#2514のT1-Bで、未予告のRule GapからAI Candidateを生成し、非公開Expected/既知Mutantを独立評価。一回のSessionで新規性と誤採用を同時に判別する。Resultがpartialなら未観測を不合格/成功のいずれにも変えない。
- **P1別レーン**：#3292の一Inspector実Plugin Invoke、#3334の表示と#2586実Provider readを比較。Plugin/Workbench/BOM全部、`*.tei.md` Reader・Host・OKF本番E2EはP0-CのHard dependencyにしない。

**本節は受入条件の更新のみ**。実P0-A/B/C、新AI候補生成、実CLI/Studio Plugin同一Provider利用、実DB・Host・C1は今回**NOT RUN**。Scenario/Expected/Tool/Host/Policy/Readerの第二実装や新しい総合Gateを設けない。


## 9. TT-56を一つのSource資格に収束させ、選択・起動・受理を混同しない（2026-11-08）

### 9.1 #3324と#3326の独立成功を、一本の検査へ接合する

両PRは**同じ既存**`tools/reference-issue-test-ability/context_gate.py`・Workflow・Design Context計画を変更しており、併置・両方の新Gate採用はSource SSOTを分裂させる。各PRのexact HEAD CIは§8.1のとおり成功したが、**相互統合HeadのCI／main pushは未検証**。

| 既存候補 | 良い点・実測済みの限定境界 | 一本化時の位置づけ |
| --- | --- | --- |
| [Draft #3326](https://github.com/hat47x/tei-platform/pull/3326) | 独立に固定した**必須Validation 3件**と、実CLIの`source-index` **indexモード**全件（従来のsubjectモードとの違いが旧CI #42385502023 failureで判明）を照合。7件の不整合＋非必須省略1件の正対照 | **集合の全件性・欠落・正当な非必須省略の基線**に採用する候補。実取得のSource Index mode/RawとRequiredの由来を明示 |
| [Draft #3324](https://github.com/hat47x/tei-platform/pull/3324) | `inspect-validation`から**特定RuleのCanonical Positive属性・Relation・Source Anchor/Role**まで照合。Rule/Relationが見えていても定義や関係がずれる8一時コピー負例 | #3326基線の**必要な一Ruleに対する詳細資格**として取込可否を比較。全件Indexに対する二つ目のRequired集合を新設しない |

**統合前の差分試験**：候補となる単一Branchへ、(1) 独立Required 3 Rule（全件Index対照）→(2) 必要な1 RuleのInspect Canonical/Relation/Attribute/Source Anchor→(3) 既存16欠落・変形負例＋#3326の8対照＋#3324固有の意味参照ずれ負例を**重複排除**して移植、(4) 完全な実CLI応答とSource/World Fingerprintを要求する。実際に相違点がない負例は二回数えず、必要でないRuleの非公開・未取得をfalse/欠陥としない。結果は`required`、`observed`、`unresolved`、`Consumer Received`の各集合を分ける。

この段階でSource Root/FP一致は**連続10回のread-only CLI起動を同時刻の原子的Snapshotへ束縛しない**。A→B→Aの旧Source/Plugin再資格、実Tool Reply終端・分割全Byte、Role/Parser実消費、外部Principal・Tenant/Disclosureは引き続き別Claim。さらにHuman/OwnerのMeaning Acceptance・External Host Grant・Provider Effect/ResultはこのContext Validatorに実装しない。

**採否とOwner**：#3262がP-1b Required/Observed Contextの一つの検査を所有し、#3324/#3326の重複Branchを**Ownerの差分判断まで同時mergeしない**。#1180/#2514の独立Expectedと既知Mutantを再利用する。どちらのCI成功も`qualifiedTestGeneration=true`へ変更する根拠ではない。

### 9.2 「宣言→選択→実装の存在→実呼出し→適格」も一つの真偽値にしない

[main統合 #3337](https://github.com/hat47x/tei-platform/pull/3337)のread-only `reference-plugin-capability-coverage` は公開合成Workspaceで12 Capabilityのうち**現参照ルールでは6つがEnvironment Bindingから非選択**と報告し、Pluginの選択とその提供Capabilityの選択、同Pluginの一部Capabilityだけ選択されることを分離した。これは**現参照ルールに依存した観測**であり、[後続PR #3350](https://github.com/hat47x/tei-platform/pull/3350)は既存`Capability.SupportsDatabaseProvider(instance.Provider)`に揃えてdatabase-providerの技術名/Capability IDの誤比較を直す候補を提案（12中の未選択は**候補ルールでは4**）。#3350はOpen・未マージであり、先行6件を「実Pluginの欠陥」と再解釈したり、新候補4件を現在mainの結果としない。

[main統合 #3336](https://github.com/hat47x/tei-platform/pull/3336)の子Process Build Referenceでは、**invoker executable digest**と**invoked executable digest/argv/exit**、child申告artifact digestと実Disk artifact digestの一致を別観測し、`l3Claimed=false`を維持した。Build開始主体が分かったことは、Go Source Closure qualified、Stage1生成物がStage2を再ビルドするSelf Build L3、実Runtime Provider Load、BOM tuple、Deployment Grantの資格を与えない（#2831/#1090/#3267）。

AI/Plugin Inspectorの一縦切りは、**Declared Offer/Capability→Environment/Contract Selection→Source/Build pinned→actual executable/Provider Loaded→CLI/Studioの実Invoke→同Resultと独立Expected→Current Grant/Effect**を分けて記録する。選択Bで実呼出しAのままならResult値一致でも不合格。CLI/Studio二Surfaceの成功は、Provider二版×Consumer二版の組合せ試験・Trusted Evidenceを代替しない。#1251/#951/#3292を実装Ownerとし、新Resolver/Host/BOM発行をAI-first側へ追加しない。

### 9.3 AIに公開する使い方を「説明が書いてある」から「実契約と同じ」へ

[main #3340](https://github.com/hat47x/tei-platform/pull/3340)は`-h`の引数Value集合／DefaultをDiscoveryと**同じ内部変数**から投影するGo限定試験に成功した。[main #3333](https://github.com/hat47x/tei-platform/pull/3333)の`view.writeFlags`と併せて、Tool Profile/Skillが知る`Command + Flag + Value/Default + Effect`と、実Binary受理・実argv/副作用を比較する。Help/Discoveryの整合は**Permission/安全なShell/OS Egress**の実証ではない。Script/ToolのRead-only選択はcanonical meaning非変更と、named-output・Session Log・実Network送出を別資格にする。

**同じSourceを複数利用する実例**：[main #3343](https://github.com/hat47x/tei-platform/pull/3343)は一Holiday Dataを複数Calendarで使う場合に`holiday-calendar`で行を選択し、Calendarを省略した曖昧Readを拒否する限定Code/Go試験を実装した。これは**同Data identityでもFunction/Calendar ContextとSelection条件でActual Sliceが異なる**ことの例であり、Provider取得した0行・単純行数一致から業務Scopeが網羅されたとは推定しない。行のCalendar値が宣言済みCalendarかの検査、実DB Reader/本番の休日計算はなお別。

### 9.4 次のDoneは、新Runnerよりも「一本のRequiredと一回の実Consumer」

1. **P0 Context**：#3262の既存参照Consumerで#3326の独立Required集合と#3324の意味参照・Relation差分を一本化し、合成の重複負例を整理。**統合Head／main pushのCI**で一RuleだけでもSource/Role/Attribute/Index/Inspectorの実観測を確保する。Scope/Source選択が不成立なら`partial`と不足を返し、全域`qualified`にしない。
2. **P0 Tool/Holdout**：#2509/TAI #168の実H2を、実OS/Harness Forbidden Source/独立Expected拒否、Tool全Byte終端、CLI実argv/Session Log/ネットワークSinkを観測した一Sessionとして資格化。旧探索やMockの局所成功から比較四Sessionを成立扱いにしない。
3. **P0 T1-B**：実Contextが照合できたScopeで、[既存#3273](https://github.com/hat47x/tei-platform/pull/3273)の盲検Candidateを1件選択し、#1180/#2514の独立Expected/Mutantで判別。**B0/B1/B2/BX**などの比較ラベルは既存テスト設計の範囲で使用し、新しいCase台帳・AI専用Expectedは作らない。
4. **P1 Plugin/Build**：一つのread-only Inspectorの実Provider Invokeと、Build invoker/invoked・Source Closureの別証拠を接続。#3337の未選択Capability一覧や#3350の選択解決の改善候補は**診断**であり、実Host GrantやBOMの資格ではない。

**実証の上限**：本節は既存のmainコード・固定HEAD CIとDraft仕様の照合による**設計/計画追補のみ**。#3324/#3326合同CI、新H2/H3比較、盲検T1-B、CLI/Studio実同一Plugin Invoke、実Provider DB、Owner正式Meaning受理、Self Build L3は今回**NOT RUN**。#3324/#3326、#3350の製品main導入を先取りしない。


## 10. TT-56統合後のAI-first受入順：不足の診断から盲検新規発見へ（2026-11-08、現行状態）

### 10.1 今回、新しく資格を持った部分

| 既存Owner／成果 | 最新の実証・採用限界 | 新しく主張してよいこと／いけないこと |
| --- | --- | --- |
| **[#3262](https://github.com/hat47x/tei-platform/issues/3262) Design Context** | [main統合 #3354](https://github.com/hat47x/tei-platform/pull/3354) が、旧[#3324](https://github.com/hat47x/tei-platform/pull/3324)/[#3326](https://github.com/hat47x/tei-platform/pull/3326)を**一つのContext Gate**に統合。専用[exact-head CI #42391707962](https://github.com/hat47x/tei-platform/actions/runs/42391707962) success、[PR Health #42392009764](https://github.com/hat47x/tei-platform/actions/runs/42392009764) success。固定公開Caseで**11実CLI read-only Reply＋32 Consumer Controls**：7基礎＋2 Manifest＋7 Reply Byte＋8 Required三Rule集合（拒否7・過剰拒否防止1）＋8 Canonical/Relation/Source参照ずれ | 「実Tool出力の一時コピーを変形した負例を正しく拒否する」限定成立。実外部Harnessでの11 Reply完全配送・原子的Workspace snapshot・全Function/Business/Rule依存・実Provider行集合・Unknown Goalを発見するAI能力の証明**ではない**。`boundedReference=partial`、`testExecutionAdmission=exploration-only`、`qualifiedTestGeneration=false`を維持。旧二Draftはclosed/unmerged（合同で単純マージしない） |
| **[#1180](https://github.com/hat47x/tei-platform/issues/1180)/[#2514](https://github.com/hat47x/tei-platform/issues/2514) Scenario/Expected** | main #3304の「既知の行番号=0」T1-A実走は別件。続く[Draft #3365](https://github.com/hat47x/tei-platform/pull/3365)の固定HEAD `1e2531fe` は[専用Dogfood CI #42394673357](https://github.com/hat47x/tei-platform/actions/runs/42394673357)＋Health #42394673331＋依存 #42394673273 が各completed/success。部分Scenario Receipt目録のtest-only投影と7対照を試す | 「指定Stepが返却されない」＝`notObserved`。未テストが証明された`untestedConfirmed`へ変換しない。別の受領済Stepの局所肯定を保持し、**全Product Coverageはunknown**。このDraftは未マージ。AIが未知Gapを盲検で見つける**T1-BはNOT RUN** |
| **[#1125](https://github.com/hat47x/tei-platform/issues/1125) Completion/Handoff** | [main #3353](https://github.com/hat47x/tei-platform/pull/3353)は`EmptyPolicyHandoff`＋空Targets＋`EmptyJustified=true`でも`HandoffConfirmed=false`なら`CompletionUnverified`とする限定修正を統合 | 空集合の証拠と受領確認は独立。Caller申告`HandoffConfirmed`/ `CoverageComplete` は外部Host/相手方の実Receiptでも必要行・Scopeの全読取り証拠でもない。両方の真偽をProduct Completionへ自動昇格しない |
| **[#951](https://github.com/hat47x/tei-platform/issues/951)/[#1251](https://github.com/hat47x/tei-platform/issues/1251) Capability/Provider** | [main #3350](https://github.com/hat47x/tei-platform/pull/3350)は`SupportsDatabaseProvider`を再利用して未選択数を旧方式6→**現参照方式4**へ修正。[main #3363](https://github.com/hat47x/tei-platform/pull/3363)は実Workspaceの24環境Bindingに`selected=22/missing=2/ambiguous=0`を返す（missingは`editor.database_workbench`2件） | 一覧の`missing`は**製品Validatorの違反とは限らない**。Referenceは`editor.database_workbench`まで調べるが製品Validatorは`editor.sql_analysis`だけに対応検査を限定。旧6件を現結果へ再利用しない。`selected`≠実Provider実装/Load/Invoke/Grant |
| **[#1824](https://github.com/hat47x/tei-platform/issues/1824) Studio/Inspector** | [Draft #3371](https://github.com/hat47x/tei-platform/pull/3371)はHost ServiceFactoryが非nilなら`RootServices.Inspection.Inspect`へ委譲し、未提供・誤Subject・誤Schema・誤Fingerprint等をfail closedするGo候補。PR作成後のCIは**一部成功・一部進行中/queuedを観測**。CLIは現在手書きInspect経路 | 「Hostが選んだServiceの型付き入口を利用可能にする」候補まで。exact selected Plugin/Providerが実Build/Load/Invokeされたこと、CLIとStudioの同一Provider Consumer、Wails実WebViewでのSource/権限適格は**未達** |
| **[#3149](https://github.com/hat47x/tei-platform/issues/3149) CLI実ドッグフーディング** | [Open #3356](https://github.com/hat47x/tei-platform/pull/3356)の第2回実Binary計測ではD-1/2/4/5/7/8に改善、D-3（`view`版のScope）、D-9（Managed文書≠Loaded Meaning）、D-10（内側JSON形状）が残る。CLIの`commands`19件にMCP入口は無く、独自MCP Server実装も未確認 | **実測できたのはCLIのみ**。利用可能な現行MCP経路を作り出したり、新MCPをP0のHard dependencyにしない。Tool resultの`result`配列/オブジェクト差やSource managed/Parsed/Consumedを同一視しない |

### 10.2 実証の「次の不足一辺」を小さく測る

**P0-A｜Source/必要Ruleの受領資格（main #3354を再利用）**。公開合成Subject一つで、`Required→Selected→Managed→LoaderRead→Canonical Rule/Relation/Attribute/Source Anchor→Tool Returned→Consumer Received`を同一Inputの版に束縛する。11のCLI応答は逐次取得なので共同原子的Snapshotではない。`partial`の残余Domain/Function/Businessと、欠損を示した時の否定を区別し、同じWorld FP・JSON正常だけで全域の存在/不存在を断定しない。これ以上TT-56 Gateを複製せず、必要なら#3262の現実装へ最小差分。

**P0-B｜実外部Harnessの盲検資格**。TAI #168／TEI #2509/#2515で、Participantに**独立Expected・未公開Gap・旧回答・Skill（H2）**への物理アクセスを拒否し、Toolの許可`Command+Flag variant`・実argv・Raw stdout/stderr→Tool Replyの全Byte/終端・Session Log再送/Audience/実Network Sinkを別欄で固定。ローカルMock/Tool制限/CLI`writeFlags`の合成PASSは実OSでの全出口拒否を代用しない。比較適格なH2/H3の事前割当4Sessionは依然0/4。

**P0-C｜T1-Bの盲検新規Rule/Case発見**。#3273/#1180/#2514が、AI参加者へ見せない未予告Gap・独立Expected/Mutantを事前固定し、#3354で実取得が成立した**狭いScope**に限ってAIが新Candidate一つを生成する。既知の行番号0を未知発見として数えず、Candidate新規性、実CLI実行、Known Mutantを判別したか、局所Coverage、未測定C1/NFRを別分母で採点。部分Scenario Resultについて`notObserved`を「未実施」や「失敗」へ推定しない（#3365）。

**P1｜Inspector一Pluginの実選択・実Invoke**。#3371のHost factory委譲候補とmain #3363のEnvironment Capability診断を接続するが、`Declared→Env Selected→Contract/Provider Resolved→Build Included→Actually Loaded/Invoked→CLI & Studio Result→Independent Expected`の各Witnessが必要。B SelectedなのにA InvokedならResult値一致でも不合格。実Provider Row/現在Grant/Effectや、同Provider×Consumer 2×2 Version BOMはさらに別（#3292/#3297/#1174）。新Kernel/Registry/Hostは作らない。

**P1別｜実Read／業務Handoff**。旧`*.tei`一Function→実SQL/Provider/Row/Predicateへ#2586で接続する。#3353の空Target Handoff改善は実Commit/外部確認の代用でない。新`*.tei.md` Reader、Diagram/CRUD/SHO、OKF Attester/Matrix、Self Build L3は各既存Ownerへ残し、P0-B/Cの前提にしない。

### 10.3 統合計画の保守と証拠上限

本節への移送では、旧PR #3241の設計§1〜9を履歴資料として保存し、実装状態の変更は§10を優先する。基本計画Hには**次の受入順と担当だけを一節にまとめ**、旧PR #3241の4つの重複小節をそのまま持ち込まない。新たなScenario集合・Helper Registry・Domain Gate・Test Runner・独立Expected Storeを増やさない。

**実施済みは既存mainの有限Go/CLI/Consumer・特定PRのCIと、今回の計画移送のみ**。新しい外部H2/H3正式比較、実OSの機密到達拒否、盲検T1-B、全Product Scenario Catalog/C1、実CLI/Studio同Plugin、実Provider DB・Host/Ownerの正式Meaning・外部Handoff受領は**今回NOT RUN／NOT QUALIFIED**。必ずPR exact-headのGo/CI、main push、同一Attemptの証拠を分けて示す。


## 11. Inspector D1の直接経路→Host選択Provider実消費を最短で証明する（2026-11-08）

### 11.1 既存の二Surfaceは、同じProviderを実行した証拠ではない

他チャットのInspector D1/Plugin-first設計では、**宣言→Capability選択→Contract/Provider解決→実Build収録→実Provider呼出し→CLI/Studio受領→現在Grant/Effect**を別々に扱う。これは[main統合Plugin-first §14/15 #3338](https://github.com/hat47x/tei-platform/pull/3338)や[#3367](https://github.com/hat47x/tei-platform/pull/3367)の既存契約に従うもので、本AI-first側に新Kernel・Registry・Host・BOM発行Engineは作らない。

| 既存PRの責務 | 実コード/試験の範囲 | 採用上限 |
| --- | --- | --- |
| **D1a 既存直結の基準**：[Draft #3378](https://github.com/hat47x/tei-platform/pull/3378) | Data `sales.order`、Validation `sales.order_line.line_number_positive`、Relation `sales.order.lines`を、CLIの`inspectWorkspaceJSON`とStudioHost`ReadOnly.InspectSubject`で別々に読み、Canonical/Source Anchor/Workspace・World・Index FP・Subject・Effectと不存在Errorを対照。**両方とも既存の`operations.InspectSubject`直接経路** | 「従来経路の応答基準」を得る候補のみ。Plugin経由・2実Binary・実Wails・Providerの選択・本番権限ではない。PR exact-head Repository Healthは**failure**、他Workflowの成功を一括PASSにしない |
| **D1b-C CLI型付き入口**：[Draft #3381](https://github.com/hat47x/tei-platform/pull/3381) | `inspectWorkspaceJSONWithService`の明示的な`cliInspectionService`で、nil/Subject不一致/schema/effect/Canonical/FP不正・未分類エラーを拒否。既定`inspectWorkspaceJSON`は従来Builtinのまま | **注入可能なPort**でありEnvironmentで選択したProviderが実ロードされた証拠ではない。exact-head Repository Healthは**failure**（ログは`gofmt`）、局所Workflow成功と別扱い |
| **D1b-S Studio型付き入口**：[Draft #3371](https://github.com/hat47x/tei-platform/pull/3371) | Host `ServiceFactory`非nil時は`RootServices.Inspection.Inspect`へ必ず委譲、Service未提供/不正応答はfail-closed。factory nilのPoC経路は従来Operationへ委譲 | **Host選択Serviceへの委譲**であり、Hostが実ProviderをResolve・Loadした証拠ではない。exact-head Repository Healthに`TestInjectedOperationsOnlyReceiveAllowlistedResolvedRoot`失敗も観測され、Wails PoC Aも失敗。他のStudio/WebView Smoke成功をこの失敗へ合算しない |
| **D1c 選択・実呼出しの連結（未実証）**：[#951](https://github.com/hat47x/tei-platform/issues/951)/[#1251](https://github.com/hat47x/tei-platform/issues/1251)/[Draft #3292](https://github.com/hat47x/tei-platform/pull/3292) | 一つのread-only Inspectorについて、実Environment/Plugin Lock→exact Contract/Provider B→Build/Loadの実Byte→CLI/Studioの型付きService注入→**actual Provider B invocation**を同一Attemptに固定 | **B選択→Aを呼ぶ負例**は値が一致してもFAIL。Provider未提供→Builtin黙示Fallback、旧Source/FP、別Workspace、誤Canonical、権限外Subject、旧Consumer応答の採用を拒否する。二Consumerの結果一致＝2×2 BOM適合ではない |

**次の最小受入票（追加のContract正本ではない）**：正例ではA/Bの二Providerのうち明示したBを選択し、CLIとStudioがBを各1回実呼出し、`Provider identity + artifact/contract pin + Source Revision + Subject + Result digest + Consumer/epoch`を照合する。負例はB選択に対するA実呼出し、Provider missing/ambiguous、型付きService内部エラー、旧Responseの遅着とContext/Provider版ずれを**明示的な不合格・unknown**に分ける。基本の独立Expected/Mutationは[#1180](https://github.com/hat47x/tei-platform/issues/1180)/[#2514](https://github.com/hat47x/tei-platform/issues/2514)、契約と実Provider選択は#951/#1016、二Consumer/Wailsは#1824が所有する。

**実装順**：#3378のdirect parityを基線として保持→#3371 Studioと#3381 CLIのそれぞれの型付きPortを個別に正す→両者が共有する`internal/operations/inspect_projection.go`の一つの検証規則へ正規化（**結果検証の二重実装を放置しない**）→#951の現行Resolver/Contractから一つのProviderを実Composeして二Consumerへ注入→同一Attemptの実呼出し/独立Expectedを確認。旧Builtin direct経路をPoC互換として残す場合も、**明示Host選択時だけは暗黙Fallback禁止**。既存`application-runtime`のService CarrierをInspector目的で無条件に拡張しない。

### 11.2 AI Toolの結果受領と試験の資格もD1と独立

CLIは実BinaryのCommand/Flag/Value/Default`をDiscoveryに広告するが、[現在Draft #3370](https://github.com/hat47x/tei-platform/pull/3370)は`requirements`の内側`result`が**array**であることを限定契約`resultShape:array`へ追加する候補。`successShape`・外側Envelope・内側Resultの三者を分ける。未広告Shapeを`object`と補完して実Toolの配列を欠落扱いにせず、`resultShape`の存在からSource/対象集合/独立Expectedの完全集合性を推論しない。現在MCP Serverの実入口は未確認なので、新MCPをD1実証の依存条件としない。

[main #3354](https://github.com/hat47x/tei-platform/pull/3354)の11 CLI Reply/32 Controlsは**取得した設計情報の局所判別**、[Draft #3365](https://github.com/hat47x/tei-platform/pull/3365)のTT-105/106は**部分Scenarioの局所観測**、T1-Bは**独立Holdoutを秘匿してAIが未知Gapから候補を新規発見**できるかの試験。各ClaimのRequired Universe/Actually Consumed/Tool Reply全Byte/Source Pin/適格Audienceを別資格とし、`notObserved`を`untestedConfirmed`へ、局所C0をC1/NFR/全業務Coverageへ転換しない。Plugin D1とT1-Bは並列で進め、双方が他方のHard dependencyにならない。

### 11.3 実CIで観測した停止理由を現在の計画に反映する

[PR #3385](https://github.com/hat47x/tei-platform/pull/3385)のexact-head [Repository Health #42396703358](https://github.com/hat47x/tei-platform/actions/runs/42396703358)は**failure**。Job logでは`go vet ./...`、`go test -count=1 -timeout 60m ./...`、`tei validate -workspace reference/workspace`がPASSし、`gofmt -l .`が`tools/reference-plugin-capability-coverage/coverage_test.go`の差分によりFAIL。これは当PRのDocs/Plan内容に関係するGoテスト失敗と同義ではない。現在mainの整形済み同ファイルへ当Branchを揃える**別Commit**を行ったが、**当該新HEADのCI成功は別途要検証**。

並行Draftの限定CIも同一視しない。#3381の旧exact-head Healthは`gofmt`で失敗、#3371の旧exact-head HealthはStudioHost `TestInjectedOperationsOnlyReceiveAllowlistedResolvedRoot`失敗と`gofmt`が重なる。#3378のHealthもfailure。個別Go/Node/WebView/ドッグフードの成功だけをPR完成やBOM/Provider実証の資格へ昇格しない。修正に際しては**そのPRのHeadと仮統合Baseで失敗を再現**し、現在mainにある共通Formatting driftと、Branch自身のGoテスト契約不整合を分けてOwnerが対処する。

**実証上限**：この§11は他チャット設計の現行差分・CI原因の照合およびDocs追補のみ。新D1a/b/cのGo/CLI/Wails検収、T1-B盲検、H2/H3隔離4Session、実Provider DB、業務Meaningの正式受理・Host current Grantを新たに実行していない。
