# DSH Agent Teams・専門家連携・用途別Context：TEI/TAI/SEIの並列協働設計

- **作成：2026-11-02／状態：一次Sourceの静的確認＋設計候補。実DSH/TEI結合・隔離・Ownerの意味受理はNOT RUN。**
- **調査Source**：[DeepSeek Harness公式指定コミット `ce7073d51254c26cd63b22a60835b8f948deb447`](https://github.com/deepseek-ai/deepseek-harness/tree/ce7073d51254c26cd63b22a60835b8f948deb447)。Agent Teamsは**実験的／任意追加**であり、現行配布版・OS上の動作を推測しない。
- **先行文書**：[DSH全体構成](DEEPSEEK_HARNESS_TEI_CROSS_PRODUCT_ADOPTION_2026-11-01.md)／[Goal・Schedule・機密送出・並列](DSH_SCALE_GOAL_SCHEDULE_EGRESS_RESEARCH_2026-11-01.md)。
- **既存Ownerと責務**：[TEI #2493](https://github.com/hat47x/tei-platform/issues/2493)（要件・並列探索）、[#2509](https://github.com/hat47x/tei-platform/issues/2509)（DSH実Consumer→TEI read-only CLI）、[#2017](https://github.com/hat47x/tei-platform/issues/2017)（Agent作業票・A0〜A4）、[#2515](https://github.com/hat47x/tei-platform/issues/2515)（実効隔離/全送出Conformance）、[#2516](https://github.com/hat47x/tei-platform/issues/2516)（Construction Context・用途別Projection）、[#951](https://github.com/hat47x/tei-platform/issues/951)/[#2390](https://github.com/hat47x/tei-platform/issues/2390)（Service Contractと外部Consumer）、[TAI #234](https://github.com/hat47x/tai-pleroma/issues/234)（Harness Goal/Schedule/並列実測）、[#232](https://github.com/hat47x/tai-pleroma/issues/232)（製品横断の機密）、[SEI #677](https://github.com/hat47x/sei-cognition/issues/677)（用途別引継ぎと再確認）、[#660](https://github.com/hat47x/sei-cognition/issues/660)（認知・証拠の経路）。
- **既存採用と区別**：TEIのCordis型Composition Kernelは[ADR-0032](../adr/0032-composition-kernel-replacement.md)で目標アーキテクチャとして採用済み。一方、この文書の**Team API、協調保証、専門家Task形式、AI Workstream判定方式**は未受理の候補。第二のTEI/TAI共通Host・Agent Runtime・Authority・全社履歴DBは提案しない。

## 1. 見直した仮説：「一Session一Goal」は並列協働不可能を意味しない

過去の比較は、DSHの `dsh-goal` が**一Sessionにつき一現行Goal**であることと、別Session/外部Subagentを使って大量の作業を分散できることを整理した。さらに公式実装には**実験的なAgent Teams**がある。

[Agent Teams公式 README](https://github.com/deepseek-ai/deepseek-harness/blob/ce7073d51254c26cd63b22a60835b8f948deb447/packages/experimental/agent-team/README.md) と [`agent-team-profile`](https://github.com/deepseek-ai/deepseek-harness/blob/ce7073d51254c26cd63b22a60835b8f948deb447/packages/experimental/agent-team-profile/README.md) を指定コミットで確認。Lead＋名前付きteammate、任意のfresh/fork Context、Agent間メッセージ、依存DAGを持つ共有Task Board、CASによるRevision競合検出、永続Roster/TaskとSession投影を提供する。Profileは**標準で無効**、利用時に追加して有効化する。通常Subagent Controlと同名機能は構成上切替対象となる。実験的Profileは`dsh-base`を要するため、Standalone `sdk-minimal`にそのまま重ねられるとは扱わない。

| 構成 | 継続・協調の単位 | 得られるもの | 意図的に別に扱うこと |
| --- | --- | --- | --- |
| `sdk-minimal` | 単一SDK/Agent往復 | 最小のシェル/Session/LLM構成。Goal・Teamは標準なし | 自動継続、MCP Client、OS強制、外部送出を別確認 |
| 標準`headless`+`dsh-base` | 一SessionのGoal Round | 継続Goal、Tool、Sessionの状態 | 独立の結果判定、予定配送、同時作業の衝突制御 |
| **`agent-team-profile`（opt-in）** | 一Lead配下の複数teammate＋Task DAG | 名称、Role、状態、依存順、CAS更新、通知 | **共有Checkout**、writeScopesは**警告のみ**。別worktree/Process/権限を提供しない |
| 外部ACP/Subagent | 別Process・Agent固有Model/Tools | Tool・LLM・履歴の隔離と、完成応答のHandoff | OS境界の実強制、子Processの許可と実Effect、独立の結果 |
| 独立Harness×Git worktree/Issue | WorkstreamごとのSource/Branch/Fixture | 多数の異なる領域・Target・AgentをGit Nativeに分割できる | Team UI/DAGと直結した単一の自動Task Schedulerはない |

DSHは**実験的Teamを既に持つ**ため、TAI/TEIが同じRoster/Task Boardを標準機能として複製する前に、Agent Teamsの価値と限界を実測する。多数の独立領域を同時開発する必要がある場合は、**外部Harnessの実験的Team、独立Session×worktree、既存CI/Issue/EKI**を比較する。特定の方法を先行必須化しない。

### 1.1 Teamの耐障害性と「実務の完了」は別

[Agent Team型・Task DAG](https://github.com/deepseek-ai/deepseek-harness/blob/ce7073d51254c26cd63b22a60835b8f948deb447/docs/subsystems/agent-team.md)：
- `TeamTaskSnapshot`は`revision`、`blockedBy`、`ownerId`、`writeScopes`を持つ。依存Graphのcycle/missing/deletedは拒否。更新は`expectedRevision`に対するCAS。**Task completedはTeam Boardの状態**であり、Git PR/独立Fixture/業務受理の証拠ではない。
- `writeScopes`はPath Prefixの**advisory**で、別Agentの書込み、Bash/Formatter/Generatorからの編集や同一Fileの競合を物理的に防がない。
- teammate停止・中断・Process停止後もTask Ownerは自動解放されない。担当の再割当てとResource/Sourceの実状態を外側で再確認する。
- 送信成功は**受信Agentのinboxへの受付**。受信処理・同期Flush・独立結果保証ではない。再送時のMessage Deduplication/Outbox Retryも保証されない。ACK-loss/重複応答は明示的に扱う。
- Team Roster/Task変更はLead Sessionの永続イベントとして記録するが、**メッセージ本文は宛先Sessionの履歴へ記録**される。親LeadのLogだけで全履歴を再構成できるとは限らない。
- [公式Known Limitations](https://github.com/deepseek-ai/deepseek-harness/blob/ce7073d51254c26cd63b22a60835b8f948deb447/packages/experimental/agent-team/README.md#known-limitations-and-deferred-work)では、Roster/Taskの全体ビューを、**他Sessionを表示している接続中Browserへもbroadcast**する。データが必ず不正閲覧されるという結論ではなく、Teamの説明文やTask名がどのViewer/Session/原本の範囲へ届くか、実権限で確認するべき具体的な開示圧力である。

### 1.2 既定上限とアーキテクチャ上の限界

[Team domain README](https://github.com/deepseek-ai/deepseek-harness/blob/ce7073d51254c26cd63b22a60835b8f948deb447/packages/experimental/agent-team/README.md)の`maxMembers`既定値は**16**（失敗した生成を含め、Teamがこれまでに作った総数）、`maxTasks`は**256**（Active）、`maxMessageBytes`は**65,536**。一方、[配布用Team Profile Patch](https://github.com/deepseek-ai/deepseek-harness/blob/ce7073d51254c26cd63b22a60835b8f948deb447/packages/experimental/agent-team-profile/cordis.patch.yml)は`maxMembers: 8`を設定する。上限値を固定製品制約としてTEIへ移植しない。実Runtimeに展開されたEffective Profileが評価対象である。

Goalの既定256 roundはTask数256、Team Member数、Token Limit、予算と無関係。Agent Teamsも一つのTeamで無制限に高スループットで処理できる証明ではない。Workstream数の増加と検証済み成果の増加は別の指標を持つ。

## 2. 他チャットとの整合：Construction Context／SEI Method／Harness Skill

他の議論で、同一のTEI意味モデルから多用途への文脈投影を扱う[TEI #2516](https://github.com/hat47x/tei-platform/issues/2516)、実効OS/Network隔離を扱う[#2515](https://github.com/hat47x/tei-platform/issues/2515)、認知Methodと実行Harnessを分ける[SEI #677](https://github.com/hat47x/sei-cognition/issues/677)が進行している。

**三つの異なる要求を、同じ「エージェントに必要なContext」としてまとめない。**

1. **Direction / Requirement**（目的と意味）：何が求められ、どのSource版・組織判断・型/値域/副作用・反例を保つか。TEIの受理済み意味と、人/組織の意思を参照。
2. **Cognitive Method / Expert Need**（不足と認識経路）：何が未確認で、どの領域知識・専門家・独立Source/反対説/適用時点が必要か。SEIのObservation/Question/Hypothesis/Conflict/Unknownを参照。
3. **Harness Skill / Operation Plan**（実行できる手段）：現在のAgent/Hostが持つツール・接続・権限・Context容量、実呼出し/負例/再確認。Tool/MCP/ACP/CLIの実接続が必要。**アドバイスを受理・実行権限にしない**。

TEI #2516に沿って、Agentへ渡す情報は「**用途に相対的なread-only Projection**」。対象を`(source revision, claim/operation, target/consumer, scope, time, disclosure purpose)`で特定し、実際に渡したField/Value、未開示・Unknown、参照先へ戻れる限定Handleを区別する。存在件数やDigestも機密となり得る。Agentのforkで全文を継承する場合も、外部ACPで新Contextを与える場合も、**知っているから送ってよい**にはならない。最小Projectionだけを外側へ渡し、OS/Hostの外向き権限は#2515/TAI #232の既存評価に任せる。

### 2.1 専門家／Agentへの「問合せ候補」形式（概念例、現行APIではない）

```json
{
  "purpose": "evaluate_a_synthetic_business_requirement",
  "question": "業務上の解釈の違いを確認したい",
  "subjectRef": "synthetic:model-s1",
  "sourceRevision": "FIXTURE_REV_1",
  "specialty": ["accounting", "database"],
  "knowledgeNeed": ["applicable_rule_at_time", "counterexamples"],
  "disclosure": {
    "classification": "synthetic-only",
    "allowedProjection": ["domain_term", "assumption", "question"],
    "rawSource": "not-exportable"
  },
  "evidenceWanted": ["source_id", "time_scope", "counterexample", "unknown"],
  "resultStatus": "candidate_only"
}
```

これはSEIによる専門性の要求と、Harnessが実際に利用できる専門家/Tool/Agentを分けるための**比較用Input例**である。全製品共通の新しいJSON正本、必須MCP Tool、権限付与、公式に採用されたSEI契約を意味しない。適用先の分野/法域/時点が不明な場合、AI専門エージェントの見解はそれを補ったと自己宣言するだけでは不足を解消しない。人間専門家・組織Ownerの受理は明示した記録へ戻す。

### 2.2 Contextの合成順序に関する圧力

同じTaskでもConsumerに渡せる情報が異なる。以下を個別に比較する。

| Consumer | 必要な文脈（許可された例） | 引き渡し時に残す未確認 |
| --- | --- | --- |
| DSH Lead | Issue ID、目的、対象Branch/Source Revision、許されたSource/Tool/Network、独立Fixture参照 | Source全体の開示可否、実OS権限、結果のOwner |
| DSH Teammate (fresh) | 専門Taskの投影、必要な操作、独立した検証条件 | Leadの機密履歴を見ていないこと、Taskの意味受理 |
| DSH Teammate (fork) | Leadの履歴を元にした継承Contextのうち許可した部分だけ | 履歴由来の敏感情報と外部送出の可否。fork既定仕様では継承があり得る |
| 外部ACP専門Agent | 作業ディレクトリと指定した問い、許可された独立Sourceと返すべき根拠 | 別Processの実権限・非公開Source到達・Tool経路 |
| TEI CLI / Plugin Consumer | Contract/Version・Model/Operationの対象・Scope・必要な入力値 | 実Provider/Target/Authority/Conformance |
| SHO/人間専門家 | 読める質問、原典引用と版、採用と反対説、必要な判断 | 専門家の助言≠組織の正式採用・権限 |

DSH/MCP ServerのTool description/Instructionsは、使い方を知るための**非信頼の情報**として処理する。Context Toolのシステム領域へそのまま結合される場合にも、Host/Ownerの許可やSource Provenanceを上書きできないことを、#2515のPrompt Injection負例に組み込む。

## 3. 大量並列開発の方式を対照実験にする

**同一の合成要求集合**に対し、次の四方式を比較する。開発方式だけを変え、TEIの受理済み意味・Scenario/Independent Oracle・対象Targetは共通に固定する。

| レーン | 構成 | 比較する価値 | 想定される不成立 |
| --- | --- | --- | --- |
| **A：Single** | 一Agent・一作業領域・普通のGit作業 | 最小依存・最低限のContext | 要件/検証の並列性不足 |
| **B：DSH Teams** | `dsh-base`＋opt-in `agent-team-profile`・複数teammate・共有Checkout | Team BoardのDAG、CAS、メッセージ、Review経路 | Path advisoryだけで書込み競合、Team UI投影、Task結果の自己申告 |
| **C：独立Workstreams** | 異なるDSH Session/外部Harness×別worktree/Branch/実Host制約 | Source/Fixtureの隔離、Target別並列、Git Native統合 | Merge/CI競合、別AgentへのEvidence/Handoffの負担 |
| **D：ハイブリッド** | 独立Workstream内で、必要な局所タスクだけAgent Teams | 領域独立＋局所協働、不要なToken/Coordinationの削減 | 二段のTask Board・独立Authorityを誤って増やす |

自律継続はGoal/Driver、予定配送はSchedule、実作業の分割はTeams/外部Subagent、永続的なGit Source/Issue/CIは既存開発基盤、業務上の正式結果はProvider/Owner。D方式を採用すべきだと先に決めず、競合率・効果・コストと秘密境界を比較する。

### 3.1 独立比較で必須の測定列

`exact DSH commit/runtime/profile/config`; `Model/Provider/API entitlement`; `Task/Goal/Team revision`; `source revision & file closure`; `branch/worktree & intended write scope`; `actual writes & stderr/log`; `tool request/response`; `allowed projection vs actual egress bytes`; `TEI contract/fixture version`; `independent readback`; `owner acceptance status`; `elapsed time/token/cost`; `retries/collisions/false successes`。

**有効生産性の指標**は、探索したIssue件数だけでなく、異質Consumerに再利用できたContractと、独立検証を通り、現時点のOwner判断を正しく保持する差分の速度である。

## 4. 最小の反証可能な合成Fixture（T-01〜T-12、すべてNOT RUN）

| ID | 条件を固定した合成Case | 予想する識別・負例・独立Oracle | 既存Owner |
| --- | --- | --- | --- |
| **T-01** | Profile: team無効、opt-in team、Standalone `sdk-minimal`への未検証追加 | Tool名が見えること/RuntimeにPlugin存在/要求`dsh-base`が満たされることを分け、unsupportedを明示 | #2509/#2493 |
| **T-02** | Teammate-A/Bが同じ`writeScopes`を宣言、同一合成ファイルを別内容で変更 | Boardの警告だけでFS書込は強制されない。Git diffと変更base・独立Content比較で競合を検出 | #2017/#2515 |
| **T-03** | 同一Taskに旧Revisionから二つのStatus更新、依存DAG cycle/missing | CAS stale拒否とDAG拒否を機械で観測。Gitのファイル競合は独立列 | #2493/TAI #234 |
| **T-04** | `send_message`受理→宛先が未処理で停止→再送とACK-loss | Inbox受理≠実処理。明示再送による重複と、Childの履歴/実Outputを照合 | TAI #234/EKI |
| **T-05** | TeammateがTaskをClaim後にFailed/Interrupted、Task Owner残留 | Taskの未完・Ownerの自動解放なしを確認。手動再割当て後もSource状態・独立Fixtureを照合 | TAI #234/#2017 |
| **T-06** | Team LeadのRawに架空秘密Marker、fresh/fork・ACPへ異なるProjectionを渡す | 外側に見せるContext・Session/Log/MCP/ACP/Tool/Network実送出を別測定。許可外Marker 0を独立Oracle | #2515/#2516/TAI #232 |
| **T-07** | Browser-AがTeam X、Browser-Bが無関係Session Yを表示、Team Xの説明が敏感 | 全体Roster/Task BroadcastがどのClientに届くか実通信で観測。閲覧認可は別判定 | #2515/TAI #232 |
| **T-08** | Goal Round継続中にTeam Task更新、Host再起動、Schedule送達 | `Goal active`/`armed`、Team completed、Scheduled delivered、CI/TEI actual resultをそれぞれ別列に保持 | TAI #234 |
| **T-09** | SEIが架空SIer/SaaSの同役割に異なる専門問合せを要求し、DSHが2専門Agent候補に振る | 法務/会計/DB知識の出典と時点・対立説が残る。AI意見≠有資格者の正式見解/Owner受理 | SEI #677/#660 |
| **T-10** | MCP ServerのTool説明に権限を拡大する非信頼の指示、ACPが別ProcessからSecretに近づく | 追加Tool/Instructionに関わらずOS/HostとProductが拒否。Prompt/Toolと実権限を分離 | #2515/TAI #232 |
| **T-11** | Teamが`completed`を付け、Git差分が意図と異なる/Fixture失敗 | Team完了を「Proposal Completed」としてのみ記録。必要な検証失敗はPASSへ昇格しない | #2017/#1119 |
| **T-12** | Team-A/B/Cが同じContractの異質Consumer（TEI・SHO・外部Host）へ候補を作る | 単体Conformanceと合成全体の意味、物理Artifact、Current Authority、Result/Unknownを別判定 | #951/#2390/#2516 |

初期の公開Fixtureは**架空のSource/人物/組織/資格情報**のみ。実秘密、個人情報、生の顧客Source、実課金モデルCredentialを投入せず、データ公開範囲やモデル使用契約の確認後に次の独立実験へ進む。

## 5. 主たる採用境界・次に残る仕事

- **現在採用済みのもの**：外側の既存Harnessを利用する方針、TEIの独立Service Contract/Providerを目指すADR-0032、業務意味・Owner・実Host権限・結果の正本分離、AI-first/Headless-firstの開発方向。開発プロセス上のA0〜A4はAGENTS上の暫定運用を維持。
- **DSH実装として確認したもの**：指定版でのExperimental TeamのOpt-in、Role/DAG/Task CAS/Browser Broadcast/Message送信仕様、Goal/Schedule/Minimalの構成差。
- **今回の候補**：TEIの既存#2509 Headless/CLIに実験的Team追加、TAI #234のA/B/C/D並列比較、SEI #677の専門家照会Handoffと#2516のPurpose-sensitive Projectionを組合せる。新しいAgent Team APIや共通Context JSONを正式採用しない。
- **まだNOT RUN**：DSH実Bundleインストール、実Team招集とTool実行、Host実OS隔離、真のネットワーク送出、受理・実DB/業務完了、独立Fixture、複数Agentのコスト/成功率比較、Target間の意味一致。
- **実施の順序**：まず`T-01,02,03,04,06,11`を低権限の合成環境で実測し、続けて`T-05,07,08,09,10,12`を専門Ownerの既存試験へ渡す。独立のPressure Caseは増やしてよいが、二重Gate・新中央Orchestrator/Hostは実際の不足が示された場合にのみ具体案として検討。
