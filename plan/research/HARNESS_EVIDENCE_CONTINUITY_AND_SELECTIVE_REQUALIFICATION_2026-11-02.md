# AI Harnessの結果継承・選択的再資格化：TEI/SEI/TAIを横断する根拠の現在性

> 2026-11-02。公式Sourceの静的照合と横断設計の提案。**新たなDSH実行、実Provider/OS/network、同一Fixtureの統合実験、Owner意味受理はNOT RUN**。本稿は新たなRuntime/Schema/Store/必須Gateを採用するものではない。

## 0. 問いと背景

安価なトークンを大量に利用する開発では、Agent/Domain/Target/仮説・Issueの探索空間を広げられる。一方、別Agentの成果を使い回すことが増え、**過去のPASS・提案・資料を「現在のこの仕事でどのClaimまで使えるか」**が新たなボトルネックになる。情報を豊富に生成することと、その根拠を特定用途で使えると結論することを分ける。

この課題を別の共通Agent Harnessや第二Evidence SSOTで解こうとせず、**Git/TEI Source/Oracle/SHO・SEI記録/実Hostの既存正本に対する限定的な参照・再評価**として扱う。候補探索・CSW等の生成的な発散は自由に続け、後工程の受理・利用資格・副作用判定を限定する。

### 既存Owner（新しい一元的判定器は設置しない）
- **外部Harness実接続**：[TEI #2509](https://github.com/hat47x/tei-platform/issues/2509)、[TAI #234](https://github.com/hat47x/tai-pleroma/issues/234)。
- **開発作業票・過去一改訂の根拠**：[TEI #2017](https://github.com/hat47x/tei-platform/issues/2017)（現在closed、既存成果を参照）／[#1163](https://github.com/hat47x/tei-platform/issues/1163)、既存CI/PR。
- **Oracle Soundness・Evaluation Binding・Claim Coverage**：[TEI #2514](https://github.com/hat47x/tei-platform/issues/2514)。Go/Scriptの実Build/Read-byte帰属は[#2595](https://github.com/hat47x/tei-platform/issues/2595)。
- **変更依存と局所再取得**：[TEI #2582](https://github.com/hat47x/tei-platform/issues/2582)、Source/構造の用途投影[#2516](https://github.com/hat47x/tei-platform/issues/2516)。
- **認知判断の四条件**：[SEI #767](https://github.com/hat47x/sei-cognition/issues/767)（External completeness / Intensional adequacy / Disclosure-constrained adequacy / Current applicability）。
- **通信/秘密・事後の外部効果**：[TEI #2515](https://github.com/hat47x/tei-platform/issues/2515)、[TAI #232](https://github.com/hat47x/tai-pleroma/issues/232)、[TEI #1174](https://github.com/hat47x/tei-platform/issues/1174)。
- **一つの実縦断参照**：[TAI SCN-001 #28](https://github.com/hat47x/tai-pleroma/issues/28)（MMI観測と成果物の原本束縛、Digestと実byte、再開後の説明を別扱い）。

上記は別々の責任のまま参照する。本稿が所有するのは**外部HarnessのNDJSON/Sessionを入口とした「結果を再取得し、当該Claimの資格についてどのOwnerへ再照会するか」の反証シナリオ**。

## 1. DSH公式実装を調べて見つかった、結果継承の落とし穴

Source固定：[DSH `ce7073d51254c26cd63b22a60835b8f948deb447`](https://github.com/deepseek-ai/deepseek-harness/tree/ce7073d51254c26cd63b22a60835b8f948deb447)（指定版）。以下は**実装/公式READMEの静的事実**であり、TEIで実走した事実ではない。

### 1.1 Headlessの`--json`は完全な監査履歴でも結果の証明でもない

[Headless README](https://github.com/deepseek-ai/deepseek-harness/blob/ce7073d51254c26cd63b22a60835b8f948deb447/packages/bundle/headless/README.md)：
- `--json`はNDJSON。Sessionの`session`イベントから始まり、`status/text/thinking/tool_call/tool_result`を出し、`final`で終わる。**途中経過は投影**であり、現在のTokenをリアルタイムに漏れなく追跡するものではない。特に`text/thinking`はコミットしたAssistant Messageの投影。Discard/Retryされた試行は観測されない場合がある。
- `final`が**存在してもターンが失敗**している場合がある。`error`が無くても成功とは言えない。`process exit=0`と`final.reason.kind=completed`を**Agent Runの完了条件の候補**として同時確認し、さらに**Tool自身のコマンド終了コード・成果物・独立Gate**を照合する。
- `tool_result.status=completed`はToolの復帰であり、内部Shell commandがexit 0という意味ではない。
- `final`以外の文字列/キーには**8KiB**の上限、1イベント行には**32KiB**の上限、深さにも制限があり、`truncated`が付く可能性がある。監査に必要なSource全体・Test結果・JSONの型/数値/Exitを**NDJSON投影だけ**から断定しない。
- Runnerの外側で失敗した場合、`error`Eventだけで`final`が出ない。Pluginロードより早い失敗はstderrのみもあり得る。

[Session Projection README](https://github.com/deepseek-ai/deepseek-harness/blob/ce7073d51254c26cd63b22a60835b8f948deb447/packages/session/session-projection/README.md)では、Projectionはコミット済みSession Eventから導出されたクライアント向けSnapshot/Watermarkで、完全なraw logと同一物ではない。**出力を失っても修復可能なように、Sessionの永続LogとGit/TEI/実Providerの現物を別経路で調べる**。Session Logは作業履歴であり、TEIの業務意味/許可/実効果の正本ではない。

### 1.2 Session再開とworktree/権限は同一性を要求する

同じHeadless公式READMEは`--session-id`で保存Sessionを再利用できるが、Sessionの記録したWorking Directory、Preset、起動者の排他状態等に一致条件があると明記する。未知IDは空履歴へサイレントFallbackせず、SubagentのSessionや別Working Directoryなどは拒否する。**別worktreeへ移したから同一Sessionを引き継げる**と推測しない。外部Handoffでは原Sessionの参照と作業Ticketを明示し、新Agent/Sessionの開始は旧成果の再審査を経る。

公式は一Task一Git worktreeでのHeadless利用例を示す。`--add-dir <git-common-dir>`はshared refs/config/hooksを含むGit metadataに**広範な書込み権限を与える**。Git source Fileの分離とGit共有Metadataの隔離は別。**信用しないAgent/非公開repoにそのままGrantすることを前提にしない**。[#2515](https://github.com/hat47x/tei-platform/issues/2515)が実Hostの評価Owner。認可とCredential/Outward egressはRunnerのPrompt指示でなくHostで評価する。

### 1.3 Session Log追加送出とTeam Projection

[Session-log-deepseek README](https://github.com/deepseek-ai/deepseek-harness/blob/ce7073d51254c26cd63b22a60835b8f948deb447/packages/session/session-log-deepseek/README.md)の既定Upload、未送信Suffix/再有効化後Backlog、in-flight送出と、[Experimental Team README](https://github.com/deepseek-ai/deepseek-harness/blob/ce7073d51254c26cd63b22a60835b8f948deb447/packages/experimental/agent-team/README.md)の全Roster/Task Browser Broadcastは、`AgentにTool Outputを短く渡した`だけでは達成できない**外部通信・閲覧者への出力面**を示す。private holdoutのExpected、SEIのRaw、顧客固有資料を、外部HarnessのSession/Tool/Team UI/NDJSON/HTTPへ不用意に混入させない。

## 2. 利用資格を「履歴の保存」と「現在のClaimへの使用」に分割する

### 2.1 最小の根拠関係（概念、固定契約/APIではない）

対象Claim `Q` を次の観測から評価する。

```text
Q: (意味/主張, 対象とgrain, 量化, Consumer目的, 時点, 許された開示)
   ↑        ↑                   ↑                        ↑
 S: Source/Model/Contract revision & real bytes           │
 O: 独立Oracleの意味/定義域/既知mutant                      │
 B: 実Build/Interpreter/Target/Tool/Consumerの使用byte       │
 C: Coverage (母集団/Provider/Query/Scope/粒度/取得状態)     │
 D: Disclosure: 実許可・実Projection・実Outward bytes       │
 T: Time/Freshness: Read set / Rule/Provider/Review時点    │
 A: 実Host Authority / Business outcome / Owner decision   │
   ↓
当該Qの用途ごとのHistory / Candidate / Qualified / Needs-review / Unknown
```

**これらの名称・状態は比較表の列候補であり、新しい全製品SSOT／Core enumではない。**

- **保存した履歴の真正性**（誰のどのSource/Tool/Artifactだったか）と、**そのClaimの現在の適合**（時点/対象/権限が合うか）を独立に表示する。歴史的に真だったことを後から消去せず、現在使える資格を別評価する。
- #2514の**Oracle Soundness / Evaluation Binding / Claim Coverage**はそれぞれ独立な必要条件である。さらに#767の**External Completeness / Intensional Adequacy / Disclosure / Current Applicability**がClaim利用者に必要となる。元Sourceのdigest一致・AI同士の一致・CI greenをこの全体の代替にしない。
- `Supported(Q)`はClaimが要求する列について**独立した根拠が揃うときに限る**。`Refuted(Q)`は適用範囲と反例の条件が明確なとき、`Unknown(Q)`は対象・依存・読取・実送出・現在Authority・原本束縛の必要な情報が欠けるとき。Unknownを空集合・失敗・falseに丸めない。
- Claimに必要な条件は違う。`存在する実例1件`は証人の正しさで言える一方、`全件一致`や`存在しない`はProvider/Scope/対象全件性を要求する。秘匿Projection `V(S1)=V(S2)` で非公開`Q(S1)≠Q(S2)`なら、外部HarnessはVだけで判別できない。保持Product内部での許可された計算か、Unknownを返す。

### 2.2 「何が変化したときに、何だけを見直すか」の局所性

[#2582](https://github.com/hat47x/tei-platform/issues/2582)はData ContextのResult Slice/Query依存に基づくreactive refreshを扱う。これを外部Harnessが持つ**作業結果/評価資格の再利用**へ対応づけるのは**研究仮説**であり、Data Context実装にAgent専用Schedulerを組込む提案ではない。

| 変更されたもの | 既存根拠の扱い（候補） | 不成立の反例 |
| --- | --- | --- |
| **依存していない別集合** D2のRowだけ | D1だけに依存するClaimは、D1 Source/Scope/時点が有効と分かる範囲で保持。JOIN/SUM等D2依存Claimは再検証 | 変化があれば全Claimを無効化→過剰再実行。逆に全て有効→誤再利用 |
| **未知の依存**・Providerの部分更新 | 無影響を断定せず、必要ClaimをUnknown/再取得へ | 取得範囲が不明な`NOT EXISTS`やLEFT JOINの未一致を不存在の証明へ |
| **Source byte・Toolchain・実Build閉包** | 実評価Snapshotを再束縛、必要なProfile/Caseを再実行 | [#2595](https://github.com/hat47x/tei-platform/issues/2595)のGo宣言≠実Build byte、Script ABAの前後Digest一致 |
| **Oracle Expected/意味規則** | 過去PASSは履歴として保持。独立Oracleを改訂し、新期待値で再判定 | `Got`同一でも旧`Want`の誤りでPASS/FAILが逆転 |
| **Projection形式のみ**（内容・対象同じ） | 当該読み取りClaimが必要とする意味保存が示せれば再投影/表示確認のみ | 新投影がNULL/未取得/大整数を落としているのに同等と断定 |
| **Reviewer/Ownerの確認撤回**（Source byte同じ） | 歴史上のReview Eventは残し、**現在の意思の適用**を再確認 | 同じGit SHA・同じModelで以前の電話確認/業務承認を現在も有効と推測 |
| **実Host/Targetの現在Authority** | 設計/静的Caseの結果と実行可否を分離。現在実Authorityは再確認 | Agent Task/Tool/Goal completeによって本番保存を許可 |
| **開示方針/Recipient/Harness Profile** | 過去の公開可能性を新宛先へ自動継承しない。実外向きField/Bytesを確認 | DSH Session LogのBacklog、Team Browser投影、ACPの別Processの到達範囲 |
| **業務CommitのACK消失/Receipt不明** | 当該Operation IDと独立Readbackを照合。値が同じだけでは当該実効果へ帰属不可 | Userに二重再Submitさせる、同一値だから成功とする |

これらを**一つの"dirty=true"ビット**に畳まず、Claimごとの必要証拠とChange Eventの交差を評価する。変更の直接/間接依存が見える場合は**局所再検証**、依存が見えない場合は**必要範囲のみ未確定**とする。全件の無限自動再実行を保証するものではない。

## 3. 大量並列開発で「安価なTokenの拡大」と「評価工数」を共存させる

### 3.1 二つの異なるレーン

```text
自由な候補・異説・反例・専門Domain探索
       ↓ (任意量のCandidateを保存し、未採用仮説も保持)
Issue / Git worktree / DSH Goal・Agent Teams / SEI Inquiry
       ↓
既存の契約・Source・対象範囲を参照する「評価に使うための投影」
       ↓
独立Oracle・実Build/Consumer・現在のScope/開示の照合
       ├─ Qualified: 当該Claim/用途/時点に限定した使用資格
       ├─ Needs-more-evidence: 欠けたOwnerへ戻る
       └─ Refuted/Unknown: 反例と原候補を保持
       ↓
既存のPR/Issue/Owner判断/実Providerの結果を参照
```

探索の発散と評価資格を分ける。**PoCの途中でAIが自由な候補生成を自動停止する規則にはしない**。安価なTokenによる発散と独立Agentによる異説生成は広げ、評価・本番適用・正式受理に必要な実証を対象Claimに応じて分配する。あるClaimの未確定性は、独立した別Claim/別Workstreamの調査継続を妨げない。

### 3.2 共通化するのは「正本」より参照先の互換性

HarnessごとのSession Log/NDJSONを唯一の成果保存としない。既存GitHub Issue/PR/Artifact、TEIのSource/Model/生成物、SEIのObservation/Method、SHOのSource reference、CIの試験、実Host結果を**当該用途に必要な範囲だけ再読する**。パケット形状は#2017の既存Task Packetや#1163証拠の実装範囲を先に再利用し、情報不足は#2516の投影上の不足として表示。一般化された新エンジンを導入する前に2種類の異質Consumerで観測する。

推奨する評価列（表/Issueに置く候補）：
`claim & intended use`, `domain+scope+grain+quantifier`, `sourceRevision & actual consumed bytes`, `oracle lineage+mutation sensitivity`, `evaluation attempt/exit/Tool result`, `provider/read-set completeness`, `consumer+projection loss/unknown`, `current owner/review/authority status`, `outbound recipient+field exposure`, `current verdict with basis+limits`。**誰でも再利用できる認可Tokenをこの票へ埋め込まない**。

## 4. 反証可能な合成試験（EQ-01〜EQ-12：未実施）

| ID | 入力と変化の対照 | 期待する識別・失敗モード | 既存Owner |
| --- | --- | --- | --- |
| **EQ-01** | DSH Headless NDJSONで`final`あり＋`reason`非完了、Shell内部exit非0 | `final`とTool Statusを業務/CI PASSにせず、Process Exit+reason+別Gateを照合 | #2509/TAI #234 |
| **EQ-02** | 10KiB tool_result・33KiBイベント相当をMockで出す | `truncated`や投影欠落時に原Sessionと実Tool/実Artifactを必要なら再取得、投影を完全証拠にしない | #2509/#1163 |
| **EQ-03** | 同一Session IDを違うworktree/cwd/preset・既存live ownerで再開 | 公式Runnerが示す拒否と、別Sessionを作る際のHandoff/Source readbackを区別 | #2509/TAI #234 |
| **EQ-04** | 3 Agentが別worktreeだがGit common dirを共有してMetadataを変更する合成条件 | Source隔離とGit refs/hooks/configの実権限を別証拠、`--add-dir`で実Grantした範囲の試験 | #2515/TAI #232 |
| **EQ-05** | H→D1(2件)・D2(3件)でD2のみ変更、D1-only SUMとD1×D2 JOIN Claimがある | D1-onlyの妥当な局所保持と、JOIN側の再検証を両立。Provider Snapshot不明ならUnknown | #2582/#2516 |
| **EQ-06** | D2部分取得→LEFT JOIN未一致(H,NULL)→後から一致Rowを取得 | 撤回・置換、未取得≠NULL/不存在、過去Claimと現行Claimを分離 | #2582/#2387 |
| **EQ-07** | 同じSource/Build/Case`Got`でOracle`Want`のみ一件変える（half-even） | 履歴PASSは残し現在資格を再評価、実CaseResultのGot不変とOracle Soundnessを別に検証 | #2514/#2016 |
| **EQ-08** | `manifest.Source=D1`と実Build=D2、あるいはScript ABA | pre/post digest一致・Report一致だけで実用Sourceが同じと断定しない。#2595の実証を参照 | #2595 |
| **EQ-09** | 同一公開Projection `V(S1)=V(S2)`、秘匿Rawにより`Q(S1)≠Q(S2)` | 外部Agentは独立全件判定をしない。内部の許可された限定検査orUnknownを返す | SEI #767/TEI #2516 |
| **EQ-10** | 同じSource BytesでReview/外部専門家の確認だけ撤回、現在Authorityも独立に変化 | 歴史上の受理Event/現在の判断/権限/Tool結果を区別。原因別局所再資格化 | #1125/#1174/SEI #767 |
| **EQ-11** | DSH Sessionに合成holdout Markerを記録、`dsh_session_log`有効/無効/再有効化、Team UI表示 | Model messages/Session backend/NDJSON/Request拡張/Broadcastの各到達を分け、許可範囲外のRaw0を実測する | #2515/TAI #232 |
| **EQ-12** | 同一MMI観測を使いD-04文書作成とSCN判断、一方だけ改版/保存artifact変更 | 判断根拠と成果物の実byte/digestが同一改訂を指すか、当該ClaimのQualificationを別に検証 | TAI #28/TEI #1163 |

最初に着手するのは**EQ-01/02/03/05/07/09**のread-only・Mock・決定的Fixture。次に**EQ-04/06/08/10/11/12**の各既存実装/OS/外部Consumerの担当が検証。実行順の提案は探索活動全般の停止条件にはしない。PoCに新Runner・共通Evidence Store・全Claim共通枚挙enumを追加することは初期要件としない。

## 5. 品質・費用の計測と採用判断

**評価指標は多軸**：`候補探索の幅`、`真に独立したOracleが示せるClaim数`、`適用範囲が明示されたQualified Outcome`、`Source/Agent/Host変更後の再資格化負担`、`過剰失効率`、`誤再利用率`、`CI/Target/Owner待ち時間`、`Token/実費`、`秘匿漏れ・権限逸脱`。一つの複合スコアで業務意味と情報秘匿と生産性を相殺しない。

**今回固定しない事項**：一般の完全な依存グラフ、全Providerの自動Change Feed、AI Agent専用Runtime、TEI所有のHost、横断Broker/Storage、正式Conformance Schema、勝手なレビュア・Owner承認、旧Sessionの無制限再実行、DSH定額課金の契約互換性。具体的な不足が2種類以上の独立Consumerで観測された場合、各Ownerの適切なPlugin/API/運用へ提案する。

**検証状態（2026-11-02）**：DSHのHeadless/Session/Team/Session-log READMEと既存TEI/SEI/TAI GitHub Issue/設計の照合は実施。本稿のEQ-01〜12、DSH実起動、Go/Java/JS Target、異Harness横断の実資格判定、OS通信、Owner受理は実施していない。旧Issue/過去PRの局所証拠を今回実施した試験と混同しない。
