# DeepSeek Harnessを大量並列開発に利用する際のGoal・Schedule・機密送出・独立検証

- **2026-11-01／研究・設計案／実DSH起動 NOT RUN／Owner未受理**。
- **主たる問い**：安価な生成AIトークンを大量に利用できるとき、TEIの要件・Issue・実装・Target・関連製品を広く並列に開発できるようにする。そのために、DSHに**本当に存在する自律継続の構成要素**と、利用者・Product・実Hostが負う保証を分ける。
- **公式Source固定**：[`deepseek-ai/deepseek-harness` `ce7073d51254c26cd63b22a60835b8f948deb447`](https://github.com/deepseek-ai/deepseek-harness/tree/ce7073d51254c26cd63b22a60835b8f948deb447)。`master`は頻繁に変わり得るため本書の「現状」は指定版での確認であり、常に最新版であることを保証しない。
- **前提との接続**：[TEIのDSH構成研究](DEEPSEEK_HARNESS_TEI_CROSS_PRODUCT_ADOPTION_2026-11-01.md)、[TEI AI-first設計](../design/AI_FIRST_REQUIREMENT_PRESSURE_AND_HEADLESS_VERIFICATION_2026-11.md)、[ADR-0032](../adr/0032-composition-kernel-replacement.md)、[TEI #2493](https://github.com/hat47x/tei-platform/issues/2493)、[TAI #168](https://github.com/hat47x/tai-pleroma/issues/168)、[TAI #232](https://github.com/hat47x/tai-pleroma/issues/232)、[TEI #2017](https://github.com/hat47x/tei-platform/issues/2017)。

## 1. 先行する結論：自律継続の四つの「動く」を別々に評価する

大量トークンで自律開発が加速しても、以下は別の実装と保証を必要とする。

| 層 | 何を継続するか | DSHの主要候補 | TEI/TAI/EKI等への帰結 |
| --- | --- | --- | --- |
| **目的・状態** | 長期にどこまで達成したか、保留・阻害・修正 | `dsh-goal`（**一つのSessionに現行Goal一つ**）、Session Log | GitHub Issue/TEIの採用済み意味が正本。DSH GoalはそのSessionの実施目標・進捗 |
| **次の推論** | 人の次のチャット送信を待たずに次の思考・Tool呼出しへ | `dsh-goal-round-driver`＋`tool-goal`。256 roundsは初期既定値 | Session継続は**目標達成の証明でも、独立結果検証でもない** |
| **再起動・予定時刻** | 時刻/定期実行、復旧時の再開、複数Sessionへの指示 | `dsh-schedule`（Host保持）、`goal`の再開 | Hostが稼働中に配信。Goalの再起動後は**人が明示的にre-arm**する条件を確認 |
| **外部作業と結果** | Subagent/別Process/CI/Host副作用の実施、状態照合、受理 | `subagent`、`ACP`、MCP/Tool、`jobs` | EKIの限定委任・独立観測、TEIのSource/意味受理/Build/DB effectを分ける |

**意味的な到達点は「Agentが完了と言った」ことから決めない。** 仕事のGoal、Tool result、実Sourceの改訂、独立検査、Providerの実効果、Ownerの判断をそれぞれの正本へ束縛する。

## 2. 公式実装で確認した構成別の事実

### 2.1 `sdk-minimal`は軽量だが、Goalの自律ループを含まない

指定コミットの[`sdk-minimal/cordis.patch.yml`](https://github.com/deepseek-ai/deepseek-harness/blob/ce7073d51254c26cd63b22a60835b8f948deb447/packages/bundle/sdk-minimal/cordis.patch.yml)と[Python SDK公式案内](https://github.com/deepseek-ai/deepseek-harness/blob/ce7073d51254c26cd63b22a60835b8f948deb447/docs/user/guide/python-sdk.md)を確認した。

- Standalone bundleであり、`dsh-base`を継承しない。Session、Agent Loop、永続Bash/PowerShell、画像読取、作業ディレクトリ、Context圧縮、Projection Registry、基本Toolが主。
- `goal`、`goal-round-driver`、`tool-goal`、Schedule、Subagentは**このbundle内にない**。`mcp-resources`が存在することは、外部MCP Serverを呼ぶ`mcp-client`が有効という意味ではない。
- **重要**：bundleの`sandbox-policy`は`mode: danger-full-access`を設定している。これを実OS上の強い隔離環境と誤認しない。利用前にProfile展開後の実構成とProcessのOS権限を確認する。
- **重要**：`session-log-deepseek`は組込まれている。モデル入力とは別のRequest拡張に、Sessionのcanonical Eventを渡す機能を持つ（§3）。

`sdk-minimal`へGoal関係Pluginを追加できるかは、必要Contextの充足・権限・Session/Web/SDKの進行駆動方式を伴う**組合せ検証の対象**である。「Goal Pluginを一つ足せば完全無人化できる」と断言しない。最初に試すのは「MinimalのままGoal無しでヘッドレス一回」と「標準`headless`を使い、実構成にGoalが存在する場合の一回継続」の二者比較。

### 2.2 `base/headless`にはGoalの部品があり、Scheduleは別

指定版の[`base/cordis.patch.yml`](https://github.com/deepseek-ai/deepseek-harness/blob/ce7073d51254c26cd63b22a60835b8f948deb447/packages/bundle/base/cordis.patch.yml)に`goal`、`goal-round-driver`、`command-goal`、`tool-goal`がある。[`headless/cordis.patch.yml`](https://github.com/deepseek-ai/deepseek-harness/blob/ce7073d51254c26cd63b22a60835b8f948deb447/packages/bundle/headless/cordis.patch.yml)はBaseの機能を使い、HMRを停止し、sandbox policyの既定を`workspace-write`にする（実OS強制を別確認）。

**長期Goalは一Session一件**。[Goal Service](https://github.com/deepseek-ai/deepseek-harness/blob/ce7073d51254c26cd63b22a60835b8f948deb447/packages/goal/goal/README.md)では、`active`/`paused`/`blocked`/`complete`と、プロセス内でのみ有効な`armed`/`disarmed`を分離する。Goalの改訂には`id + revision`のCASを用いる。再開/Fork/プロセス開始後には自動継続が**disarmed**となり、明示的な再開を必要とする。既定256 roundはトークン数、金額、経過時間、API枠の上限ではない。モデルが`complete`と報告したことについて独立の評価器はない。

[Goal Round Driver](https://github.com/deepseek-ai/deepseek-harness/blob/ce7073d51254c26cd63b22a60835b8f948deb447/packages/goal/goal-round-driver/README.md)はAgentがidleとなったとき順次Goal Roundを発行する。故障・取消・最大Token・永続化失敗・上限到達等で止まり、**再起動だけで意図せず再武装しない**。これは安全な停止点として利用できる一方、夜間に無人で継続復旧する保証ではない。

[Web bundle](https://github.com/deepseek-ai/deepseek-harness/blob/ce7073d51254c26cd63b22a60835b8f948deb447/packages/bundle/web-app/cordis.patch.yml)には別途`schedule`/UI Scheduleの構成がある。[`dsh-schedule`](https://github.com/deepseek-ai/deepseek-harness/blob/ce7073d51254c26cd63b22a60835b8f948deb447/packages/schedule/schedule/README.md)はHostに予定を保存し、Host稼働時にSessionへ通常のFollowupを届ける。Host停止期間の全回を再実行するわけではなく、繰返しタスクは復帰後の**直近の欠落回だけ**を配送する仕様。配送Receiptは作業完了の証拠ではない。標準`headless`にWeb Scheduleが常駐すると推測しない。

### 2.3 SubagentとMCPによる多様な専門能力

- [DSH Subagent](https://github.com/deepseek-ai/deepseek-harness/blob/ce7073d51254c26cd63b22a60835b8f948deb447/packages/subagent/README.md)はin-process Spawn/Forkと外部Backendを分ける。外部[ACP Backend](https://github.com/deepseek-ai/deepseek-harness/blob/ce7073d51254c26cd63b22a60835b8f948deb447/packages/subagent/subagent-acp/README.md)は子Process自身のSession/Model/Toolを持ち、Agent間で共有するものを選択する。外部Agentの応答・終了コードは受理済みの成果ではない。
- [DSH MCP Client](https://github.com/deepseek-ai/deepseek-harness/blob/ce7073d51254c26cd63b22a60835b8f948deb447/packages/mcp/mcp-client/README.md)はServerごとに明示的構成を行う。発見したToolのServer instructionsはモデルのSystem Promptへ連結されるため、**外部の指示文・Tool名を信頼済み規則へ昇格させない**。名前が一致しても実Contract・権限・出典を検査する。
- TEIはService Contract/Providerと型付きCLI/Operationを優先して公開し、必要に応じてMCP/ACP/Library/Wasmとして提供する。SEIは専門性不足・必要な証拠・法域・時点を抽出し、外側のHarnessへ「専門家照会候補」を渡す。人間専門家の正式な助言や組織の決定をAI Toolに委譲したことにはしない。

## 3. 企業機密運用で最も先に検討すべき外部送信

指定版の[`dsh-session-log-deepseek`](https://github.com/deepseek-ai/deepseek-harness/blob/ce7073d51254c26cd63b22a60835b8f948deb447/packages/session/session-log-deepseek/README.md)が重要な発見。

1. **既定`enabled: true`**で、公式DeepSeek APIへのモデルリクエストの拡張フィールド`dsh_session_log`へ、送信対象Sessionのcanonical Event群を追加し得る。既定`maxBytes`は**1リクエスト当たり最大8 MiB**。この数値は送信可否の判断基準ではなく、1回の送信byte上限。
2. これはモデルへの`messages`ではなく、APIに送る**別のリクエストフィールド**である。モデルのToken使用量が増えなくても、外部へ送信される情報は増え得る。
3. `enabled: false`で後続リクエストへの寄与を止められる。ただし**送信準備済みin-flight Request**のpayloadはそのまま。後に再有効化すると停止期間中の未送信Eventまでbacklogとして送信対象になり得る。2xx応答とwatermark永続化の間のCrash後は重複再送も起こり得る。
4. `sdk-minimal`の公式Patchにも同Pluginが含まれている。**最小Profile = 最小情報送信**という推測は危険。標準の`base`にも組み込まれている。
5. `session-log-deepseek`を無効にしても、**通常のモデル入力・Tool結果・添付資料・別のtelemetryやログ・他Provider固有の送信**まで止まるわけではない。実際の機密境界はHost/OSの通信許可、別Identity/Filesystem、最小Projection、Provider条件で確立する。

**安全な初回試験案**：合成の架空Markerしか持たない独立作業ディレクトリ・専用OS Identityで実構成をdumpし、ローカルFake API/捕捉Proxy等、許可された検査用送信先へ`enabled:true/false/false→true`とin-flight/Restartを比較する。DeepSeek本番APIへ実機密・認証情報・顧客資料を送る試験はしない。比較観測列は`profile sha / merged config / request field names / outbound bytes / event seq / watermark / LLM-visible messages / stderr+log / network recipients`。Proxyの観測と実OS層のegress拒否も独立に検査する。完成時の机上判断を「現在のProductの機密性を証明済み」としない。

**運用Profile別の結論候補**：
- LAB：非機密合成のみ。DSHの標準構成と送出を観測する。
- CONTROLLED：用途限定Projection、送出監査、Credential/Session原本の保護。申告でなく実効Config確認。
- CONFIDENTIAL/RESTRICTED：Host側でRaw Store・Credential・NetworkをOS権限/隔離で制限。モデルAPIへ出せる内容を事前に規定し、`dsh_session_log`が要求範囲を越えるなら**起動前に無効化し、Backlog再送・in-flightにも条件を適用**。本当に送出を拒否できる実構成をNegative Testで観測する。

これは[TAI #232](https://github.com/hat47x/tai-pleroma/issues/232)の新しい**具体的なProtocol egress負例**であり、新たな中央DLP/Authorization Serviceを一律に設ける提案ではない。

### 2.4 追加観測（2026-11-02）：実験的Agent Teamsの有無で並列構成を分ける

当初の本書はGoalと独立Sessionを主に扱った。DSH指定版には[実験的Agent Teams](https://github.com/deepseek-ai/deepseek-harness/blob/ce7073d51254c26cd63b22a60835b8f948deb447/packages/experimental/agent-team/README.md)が存在する。**標準Profileでは無効**で、Lead＋teammate・メッセージ・Revision付きTask DAGを提供する一方、**同一Checkoutを共有し、writeScopesは競合警告だけ**である。自動Owner解放、Message再送/重複防止、別Processやworktreeの分離を含まない。Roster/Taskの全体情報が別Session表示中のBrowserへbroadcastされる仕様も、用途別情報開示に対する圧力となる。

したがって並列方式は「Goal一件 vs 独立Session」だけでなく、**A=単一Agent、B=opt-in Team、C=独立Session×worktree、D=ハイブリッド**で比較する。詳細と独立の反証可能なT-01〜T-12は[Agent Teams×Construction Context×専門家連携](DSH_AGENT_TEAMS_CONTEXT_EXPERT_HANDOFF_2026-11-02.md)へ集約し、実測は既存[TEI #2509](https://github.com/hat47x/tei-platform/issues/2509)、[TAI #234](https://github.com/hat47x/tai-pleroma/issues/234)、[#2515](https://github.com/hat47x/tei-platform/issues/2515)/[#2516](https://github.com/hat47x/tei-platform/issues/2516)のOwnerへ渡す。**Source静的確認のみ／実Team起動NOT RUN**。

## 4. 安価なトークンによる大規模並列開発の設計

**IssueとRequirementの探索空間は積極的に広げる。受理と実行の権限は従来通り厳密に扱う。** 競合の少ない単位へ分解し、比較・実証の並列度を拡大する。

### 4.1 並列処理の単位はGoal一件ではなく「独立Workstream × exact Source」

```text
TEI Requirement / Issue Portfolio (#2493)
  ├─ 探索：異なる業種・データ形状・DSL・Target・Pluginを幅広く調査
  ├─ 合成：共通Semantic/Service Contractを参照し、機能ごとにCandidateを作る
  └─ Work packets (#2017) + independent fixture + target/paths/allowed effects
        ├─ DSH Session/Goal A : Query/SQL/Graph (branch A)
        ├─ DSH Session/Goal B : .NET/Java/Wasm (branch B)
        ├─ DSH Session/Goal C : SUI/SHO/SEI context (branch C)
        └─ DSH Session/Goal D : independent fixture review (branch D)
                 ↓
          Git Issue/PR/commit, Source provenance, focused tests
                 ↓
          Independent Evidence / Owner Meaning Acceptance / Host actual effects
```

DSHの「一Session一Goal」と、多数の独立Workstreamの並列実行は矛盾しない。**複数Session、独立Git worktree/branch、外部Harness/CI**を分けて運用すればよい。一方、DSHが複数Sessionを**自動的に依存順でスケジュールし、人の意味受理まで担う**という保証は公式Goalの範囲に存在しない。現行のGit Issue/PR/CIと必要に応じEKIの限定委任を活用し、横断Orchestratorの新設は具体的な不足が見つかった場合の候補とする。

### 4.2 扱う要件・Issue・機能のスコープを意図的に拡大する

増やすのは、機能名だけでなく**異質な利用者・業務意味・Provider・Targetで同じ概念が成立するかを検証するPressure Case**である。

- **TEI**：Finance/ERP・受託SI・SaaS運営・製造/PLM・設備保全・在庫/物流・BPM・CMS・研究/文書・IoT等で、多数の1:N/N:M/時間/数量/単位/関数従属性/副作用・承認等の複合要件。
- **SEI**：組織別の暗黙要求・MVV/慣行・認知Method・法務/会計/DB専門AI・人間専門家への委譲・根拠の時間的有効性。
- **SUI**：KJ/マインドマップ/Canvas/タスク文脈、違和感/異論の多主体収集と安全なAI投影。
- **SHO/MMI**：複数文書・引用・人レビュー・数式KaTeX/設計書・納品版、出典再解決と増分生成。
- **EKI**：複数Harness/Agentのbounded delegationとBudget、Cancellation、Partial Success、Result independent readback。
- **TAI**：組織・個人・社会の継続活動と成果比較、未知課題適応、複数ステークホルダーの影響。

ただし、異種要件の追加を理由にTEI Coreへ万能型・全機能固定enum・第二のHubを追加しない。既存の[TEI #2493](https://github.com/hat47x/tei-platform/issues/2493)をポートフォリオの正本とし、詳細な実装は各Owner Issueへ分配する。

### 4.3 安価な推論と有効な検証を組み合わせる

「使ったToken数」ではなく**受理可能な変更・発見が増えたか**を比べる。低価格・大容量のModelは発散・検索・候補実装・負例生成で幅広く利用し、独立のFixtureや別Model/Process/人の判断を必要な確認へ配分する。

| 軸 | 指標の候補 | 意味の限界 |
| --- | --- | --- |
| 探索量 | 異種Requirements、反例、未解決仮説、既存Issueへの正しい割当て | Issue数や生成コード量そのものは品質でない |
| 実装効率 | 有効なCommit、focused test、再利用されたContract、手戻り、PR差分 | CI PASSは意味受理ではない |
| 並列効率 | WIPとブロック時間、Merge競合率、Agent/Host失敗、再起動後復旧 | ワーカー数と有効スループットは一致しない |
| モデル利用 | Token/Output・費用・再試行・Quota停止/Rate Limit・Context圧縮 | Goal round上限は金額/Tokenの停止ではない |
| 独立証拠 | exact Source→Build/Artifact→test/Provider→Observed Resultの接続率 | モデルの「完了」は独立検査でない |
| 機密/権限 | 送出の実Byte/送信先、拒否試験、OS隔離とTool/副作用の実権限 | `safeMode`や`readOnlyHint`だけで強制しない |

モデルProvider/Subscriptionごとに**利用可能な正規API/認証・請求経路・契約上の利用枠・Rate Limit**を確認する。外部モデルの互換APIが動くことから、別サービスで購入した定額契約枠をDSHから再利用できるとは推測しない。実利用枠・費用の比較は用途別に記録し、高価な追加契約や正規でない代理経路を前提にしない。

## 5. 少ない試験で大量開発の能力差を測る：六つの合成ケース

| Case | 固定する入力と手順 | 最低限区別する観測 | 反例・未達条件 |
| --- | --- | --- | --- |
| **DGA-01 Profile** | 同一合成Taskを`sdk-minimal`/標準`headless`/他の承認済みHarnessで1回ずつ実施 | 展開されたBundle、Tool/Goal/MCP公開面、OS Sandbox、実Model/版 | 同じ`dsh`という名前でTool/権限/自律継続まで同一と誤認 |
| **DGA-02 Goal** | 目標1件→3 rounds→block/resume→restart/fork、round capを変える | Goal Event/Revision、armed、進行Tool、停止理由、再開の人承認 | restartで無断再武装、round capを費用上限と誤認、完了自己申告を合格 |
| **DGA-03 Schedule** | 毎時task→Host停止→複数回欠落→復旧 | Scheduled state、latest missed delivery、Session受信/未実施、実Run | 失われたすべての定期回を実行したと誤報 |
| **DGA-04 Multiple Workstreams** | 異なる3 Issue/branch/worktree→同じ契約Fixtureの独立比較 | 同時タスク、Source revision、競合/再試験、ConformanceとOwner未決 | 別AgentのCommitを古いbaseへ統合、相互の安全設定を上書き |
| **DGA-05 Outbound metadata** | 架空Markerで`enabled true/false/false→true`・in-flight・restart | `dsh_session_log`の外部Request field、8MiB境界、Event Seq/watermark、送信先 | 無効化後も古いin-flightや再有効化Backlogを見落とす |
| **DGA-06 Product/Evidence** | DSHからTEI CLI read-only→判断不足をSEIに記録候補→別Harnessへ引継ぎ | Tool/Process/Model refs、原典・Contract、Unknown、実Readback、Owner権限 | Exit0/Agent Goal completeをTEI意味受理・本番効果と誤認 |

**PoC結果の分類**：`Source verified` / `Design proposed` / `Synthetic executed` / `External Harness actual` / `OS/network actual` / `Owner accepted` / `Business outcome observed`。現在は**Source verified + Design proposed**に限る。

## 6. Issue・計画の責任分担

| 責任対象 | 受け皿 | 今回追加する具体的内容 |
| --- | --- | --- |
| AI-first・TS/Go/Wasmに依存しないTEI Capability | [Draft #2492](https://github.com/hat47x/tei-platform/pull/2492)、[#951](https://github.com/hat47x/tei-platform/issues/951)、[#1223](https://github.com/hat47x/tei-platform/issues/1223) | DSH base/headless vs sdk-minimal、Goal/MCP/Plugin効果、反例 |
| 要件ポートフォリオ・大量並列開発 | [TEI #2493](https://github.com/hat47x/tei-platform/issues/2493) | Issueを広く増やすことと、Source改訂/検証/受理を分離した並列作業票 |
| AI作業票・独立Result | [TEI #2017](https://github.com/hat47x/tei-platform/issues/2017)、[#1163](https://github.com/hat47x/tei-platform/issues/1163) | Goal終了・Tool result・実CommitのEvidence Join |
| Harness連携と活動 | [TAI #168](https://github.com/hat47x/tai-pleroma/issues/168)、[#165](https://github.com/hat47x/tai-pleroma/issues/165) | DGA-02/03/04を異Harness・異Workstreamで比較 |
| 機密保持とOS/Host責務 | [TAI #232](https://github.com/hat47x/tai-pleroma/issues/232) | `session-log-deepseek`の既定送出・Backlog・8MiB・`danger-full-access`負例 |
| 専門家連携 | [SEI #381](https://github.com/hat47x/sei-cognition/issues/381)、[#660](https://github.com/hat47x/sei-cognition/issues/660)、[#677](https://github.com/hat47x/sei-cognition/issues/677) | 専門性不足の認知とHarnessのMCP/ACP/人間専門家照会を区別 |

**実装・試験のOwnerは各既存Issueと、必要になった際に切り出す独立Pressure Case Issue**。研究文書を新しいCore Contract・中央Orchestrator・共通Gateの第二正本にしない。並列探索は拡大し、意味・権限・本番利用の採用は既存のOwnerが担う。

## 7. 最初の実行順と未解決点

- **すぐ実行可能な机上調査**：DSHの固定SHAで`sdk-minimal/base/headless/web`の構成・送信・Goal/Tool/ACP/MCP依存を棚卸し（本書の範囲で済）。特に`danger-full-access`と`session-log-deepseek`の選択理由を対象とする。
- **最初の実験**：合成の読取り専用TEI Fixture一件を、専用の空の作業ディレクトリ・認証情報なし・Fake Model Responseもしくは許可されたMock通信先で`sdk-minimal`と`headless`から比較する。生データ送出・モデル契約・OS隔離が未確認なら実外部API呼出しをしない。
- **次の実験**：Goal/Restart/Scheduleを隔離Hostで別々に実測。Goalの再開とScheduleによる新規Session通知の効果を混同しない。
- **並列拡大**：三つ以上の異種Workstreamを各Git branch/worktreeで実行し、時間/検証/統合の詰まりとコストを測る。実Provider/実DBへの副作用は既存の認可に従う。
- **製品化の判断**：CLIだけで十分ならMCP/Agent Controllerを増やさず、Tool discovery/構成差分/安全な投影を既存TEI機能へ取り込む。自律継続・Sessionの運用で実際に不足する場合、EKI/TAI/外部Harnessのどの既存責務に追加すべきかを判断する。

**未解決**：DSH配布版とGit SHAの一致、SDKと通常HeadlessのGoal動作、実Profileの有効Sandbox、正規モデルProvider/サブスクリプションの互換性、実送出のByte、Goalと外部Workerを結ぶResult/Evidenceの実整合、Owner正式受理。いずれも今回の静的調査だけではPASSにしていない。
