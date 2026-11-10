# HTTP Response Decision Graph：Server/Client例外契約とGraph/Table/Traceの相互検証（設計候補）

- **状態**：2026-11-07の横断設計候補。仕様採用・TEI標準Graphでの保存・実HTTP/GUI/実ProviderでのE2Eは **NOT RUN / NOT ACCEPTED**。
- **受入票**：[Issue #3274](https://github.com/hat47x/tei-platform/issues/3274)。この文書は複数Issueの接続と検証計画であり、新規Core ABI、第二Driver、第二Web Host/Authority、HTTP専用Workflow Engineの正本ではない。
- **前提**：[Web構成 #1043](WEB_APPLICATION_COMPOSITION_DESIGN.md)、[標準Web API #1656](WEB_READ_WRITE_API_ADAPTER_DESIGN.md)、[#1158 UI Adapter](https://github.com/hat47x/tei-platform/issues/1158)、[#1174 Effect Fence](https://github.com/hat47x/tei-platform/issues/1174)、[#2620 Everything is Data](https://github.com/hat47x/tei-platform/issues/2620)、[#2497 Projection](https://github.com/hat47x/tei-platform/issues/2497)、[#3262 Design Context Query](https://github.com/hat47x/tei-platform/issues/3262)、[#1180 Scenario](https://github.com/hat47x/tei-platform/issues/1180)。

## 1. 問題・価値・基準

OpenAPIはHTTPステータス・Media Type・Header・Schemaを表現できるが、業務上の拒否、効果の確定、通信断、再認証、再取得、再送可否までの**クライアント動作契約**を自動決定しない。Web/Desktopで共通Application Actionを呼ぶTEIでは、HTTP statusを業務結果の正本にすると、CLI・Wails IPC・非HTTP Consumerが不整合になる。

目指すのは、(a) ドメインの結果がTransportで変質しない、(b) Client Reactionを宣言し、書込み後の不明状態を自動再送しない、(c) 実用的なHTTP Decision FlowをTEIの標準データ構造でGraph/Table/Traceに投影し、欠落・保護迂回・誤投影をテストできること。

参考：Liberator（Clojure、Webmachineに着想）、WebmachineのDecision Flow / debugging。**参考実装の旧HTTP処理順を標準としてコピーせず**、RFC 9110、RFC 9457、適用するAPI profileとTEIの実契約で再判定する。

## 2. 境界と意味の最小分離

| 対象 | 持つ情報 | 関連Owner／禁止する短絡 |
| --- | --- | --- |
| Application Outcome | named Business Outcome、Validation、Business Denial | 既存Action/Capability。HTTP codeから推定しない |
| Execution / Effect Disposition | invoked / denied / error、known-not-applied / known-applied / unknown、partial | #1174等。受理=Commitではない |
| Operation / Evidence | operation identity、effect receipt、Host Grant、read-back、source/build/contract revision | #940/#1163/#1174。古いreceiptを現在の完了証明にしない |
| HTTP Response Projection | method/pathに応じたstatus、headers、media type、Problem Typeと公開可能なextension | #1656/#1043。第二の業務正本にしない |
| Client Reaction | display/field-focus、reauth、refresh/diff、reconcile、retry候補、operator escalation | #1158/#997/#1000。UIが認可や実効果を確定しない |
| Observation / Recovery | 独立した状態・Receiptの再取得と照合、保留、明示的補償 | #1174/#2597。無根拠なreplay/rollbackを行わない |
| Topology / Projection / Trace | graph identity、node/edge、条件、参照、差分、実行経路 | #2495/#2497/#1124/#2620。図=実行済み・受理済みとは言わない |

**特に区別すること**
- HTTP 202 Acceptedは要求受理を表すが、業務完了の証拠ではない。HTTP 2xx全体を「effect known applied」にしない。
- 送信前失敗は「送信前と観測できた場合」に限って効果不在を主張できる。Request送信後のTimeout・Connection reset・Proxy error・Malformed bodyは、writeの副作用について原則unknown（独立証拠が別にあれば再評価）。
- 既存のCommand Lifecycle（`dispatched / observed / unknown`）と`persistence / durability`は既存実装のまま尊重。ここで新しい共通enumに固定せず、候補の列で同一性・差を比較する。
- 業務拒否を、Actionで未宣言のOutcomeへ無断追加しない。Output-only Capabilityの拒否とExecution Errorの表現問題は#1174のOwnerへ戻す。
- OData Response/Errorの規格、GraphQL等のTransport独自エラーをRFC 9457で無条件に置換しない。採用profileと互換性は#1656で判定。

## 3. グラフは処理・提示・復旧を混同しない

同じAccepted Application Meaningに関連する **三つの別用途の部分グラフ**を扱う。
1. **Server Request/Response Decision Graph**：信頼済みRoute、必須Security Guard、Validation、条件付き要求、Application Actionの呼出し、response transform、short circuit、audit。
2. **Client Reaction Graph**：受信可能・不可、status/header/bodyの信頼できる解釈、操作契約・UI contextによる表示／再認証／再取得／状態照合。
3. **Recovery / Observation Graph**：OperationIDとReceiptの資格を調べ、独立した現在状態を確認。再送・補償は明示された許可・冪等性・現在の権限と必要な前提を別途満たす場合のみ。

これは同一巨大な「実行状態機械」へ統合する指示ではない。Server/Client/Recoveryで**時点と知識主体が異なる**。互いの境界を明示した参照エッジを持たせ、ContextやHTTP codeを共通中核型へ埋め込まない。

### 3.1 標準Graphの候補属性（新形式の採用ではない）

| 項目 | 意味上必要な性質 |
| --- | --- |
| Graph | ID、kind/role、Source refとfingerprint、Accepted/Provisional、version、適用scope |
| Node | stable ID（表示名と独立）、role（decision/action/handler/terminal等）、参照Function/Contract、入力/出力、effect、disclosure |
| Edge | stable ID、from/to、branch predicate、guard dependency、source span、priority/ambiguity rule（必要なときだけ） |
| Binding | 既存Action ID、typed Context、Operation identity、Guard proof / Provider contract、Target |
| Projection | Graph JSON / Table / Mermaid / StudioのTarget、保持した情報・欠落と逆書戻し可否 |
| Execution Trace | graph snapshot、実通過node/edge、request/responseの観測状態、受領完全性、runtime/provider revision、Evidence refs |

Graphの通常の有向辺、多重辺、同名Node、循環、分岐の排他性、別scopeの同名IDを明示する。ループは原則禁止ではなく、**当該Policyで許される有界retry/recovery**か、異常循環かを区別する。Graph/Traceはビジネス意味の受理根拠ではない。Node kindの一覧は表現例であり固定Core enum化しない。

### 3.2 例：order.submit のServer Graph（説明用。実装/受理された経路ではない）

```mermaid
flowchart TD
    S["S: request"] --> G1{"G1: verified principal?"}
    G1 -->|no| R401["R401: unauthenticated"]
    G1 -->|yes| G2{"G2: request origin + action authority?"}
    G2 -->|no| R403["R403: forbidden"]
    G2 -->|yes| V{"V: typed validation?"}
    V -->|invalid| R422["R422: validation rejected"]
    V -->|valid| P{"P: HTTP precondition?"}
    P -->|failed| R412["R412: precondition failed"]
    P -->|valid| A["A: existing Action / Effect Fence"]
    A -->|business denial| RB["RB: mapped denial"]
    A -->|effect known| RK["RK: result projection"]
    A -->|unknown| RU["RU: unresolved result"]
```

上図は実コードの必須Guard順序を証明するものではなく、実Hostに対応するGuard Scope・依存証拠を#1043から照合するための候補。Authを通過しただけで業務実行のResource Grantを満たすとは言えない。未認証とResource非存在の判別方法はAPIのDisclosure policy次第で変わる。

| Node ID | 認知順 | 参照すべき既存契約 | 副作用 | 独立して検証する点 |
| --- | --- | --- | --- | --- |
| G1 | Principalの真正性 | #1043 verified principal | no | headerからの主体偽装拒否 |
| G2 | Origin/CSRF + Action認可 | #1043、実Security Capability | no | 全Action到達経路の必須Guard |
| V | typed payload検証 | #1158/#1656 | no | 不正JSON/NULL、Field mapping |
| P | resource/precondition version | #1174、HTTP ETag/If-Match profile | no | 412と409の誤一律化を防ぐ |
| A | Action + Effect Fence | #1174、existing Driver | **yes** | unknown/partialを成功扱いしない |
| RU | 不明状態の安全な公開 | #1656/#1178 | no | クライアント再送を誘発しない |

| Edge ID | From | To | 条件/照合事項 |
| --- | --- | --- | --- |
| E-01 | S | G1 | 信頼済みルーティング後 |
| E-02 | G1 | G2 | verified=true |
| E-03 | G1 | R401 | verified=false |
| E-04 | G2 | V | origin+authority valid |
| E-05 | G2 | R403 | 不成立（内部理由開示は別判定） |
| E-06 | V | P | typed input valid |
| E-07 | V | R422 | declarative validation failure |
| E-08 | P | A | precondition valid |
| E-09 | P | R412 | precondition false |
| E-10 | A | RU | effect disposition unknown |

**同じIDの参照解決と実配置の一意性は別問題**。Graph→Table→Mermaidへの変換でE-01等のID・多重辺が消えるならlossy projectionであり、Mermaidを変更して自動逆写像してはならない。

## 4. Response Mapping と Client Reaction

RFC 9457の`type`をProblem Typeの一次識別とする。Title/Detailは人間用で、Clientが本文の文字列を解析して分岐しない。`status`と実HTTP statusの不整合、未知extension、レスポンスなしを正常Resultと扱わない。URI形式・Problem定義・情報公開はprofile単位で管理する。実APIへの採用は#1656で検証する。

| ケース | HTTP候補 | Client Reaction候補 | 実行効果・安全条件 |
| --- | --- | --- | --- |
| 構文不正 | 400 | 入力修正／拒否説明 | 原則Action前 |
| 型は妥当だが入力が業務制約に違反 | 422候補 | Field mappingで補正 | Action内業務拒否との区分を定義 |
| 未認証 | 401（WWW-Authenticate要件に注意） | 認証状態の確認・明示再認証 | 弱い方式への自動降格禁止 |
| 権限不足 | 403／情報非公開なら404候補 | 操作停止・可能な範囲で説明 | Resource存在を漏らさない |
| 一般競合 | 409 | 最新版取得・差分提示 | 競合解決なしに再実行しない |
| If-Match等の前提条件失敗 | 412 | ETag/Revisionの再取得 | 409へ機械的に丸めない |
| レート制限 | 429 | 読込みbackoff／書込みは効果照合 | Retry-Afterは単なるhint |
| 一時障害 | 503 | 状態提示／read retry候補／write照合 | 冪等性・Authority確認なしの再送禁止 |
| HTTP 202 | 202 | 進捗・結果照会（対応時） | Accepted ≠ Completed |
| write後のtimeout / 破損応答 / transport disconnect | **応答なし、あるいは信用できない応答** | operation照合、保留、manual escalation | Commit **unknown**、自動重複送信禁止 |
| Desktop local IPC | HTTPなし | 同じmeaning resultをIPC表現で受領 | HTTP 401等をCanonical Outcomeへ逆流させない |

Client Reactionの採択条件は、Problem Typeだけでなく**Action effect、Subject/Instance Anchor、UI dirty、base Revision、Principal/Grant、OperationID、Receipt/Observationの現在資格、Target transport、Disclosure**を含める。Reactive refreshで編集途中のDirty Patchを上書きせず、古い選択epochのResponseを現在の画面へ誤適用しない（#2597/#2582）。Webセッション切れとDesktop Host IPC切断を同一の再認証経路へ潰さない。

Problem Detailsの最小サンプル（設計例、**TEI正式Schemaではない**）：

```json
{
  "type": "https://example.invalid/problems/revision-conflict",
  "title": "Precondition Failed",
  "status": 412,
  "instance": "/problem-occurrences/request-123",
  "operationId": "op-123",
  "expectedRevision": "rev-10"
}
```

`example.invalid`は外部のProblem定義ではないため、実配備で使わない。Serverの内部stack、実Resource ID、禁止Field、検証用Holdout、Grantのtoken、テナント存在、履歴を一般クライアントへ露出しない。

### 4.1 Client Graph：実行結果の認知状態で分岐する（説明用）

```mermaid
flowchart TD
    C0["C0: request attempted"] --> C1{"C1: trusted response?"}
    C1 -->|no / incomplete| CU["CU: write effect unknown"]
    C1 -->|yes| C2{"C2: typed response + operation outcome?"}
    C2 -->|invalid / unrecognized| CU
    C2 -->|known success| CS["CS: show qualified result"]
    C2 -->|validation rejected| CV["CV: bind field errors"]
    C2 -->|revision conflict| CR["CR: refresh and reconcile"]
    C2 -->|accepted only| CP["CP: pending / status lookup"]
    CU --> CO["CO: independent observation"]
    CP --> CO
    CO -->|not confirmed| CH["CH: hold, do not replay"]
    CO -->|qualified evidence| CS
```

Client GraphはResponse codeごとの一律switch文ではない。例えば`C1 → CU`は、**書込みを送信した可能性がある**ときの経路であり、安全と証明されたread-only Requestとは別のRetry Policyを参照する。証拠が不明なら`CO`が成功とも不成功とも推定しない。Client Graphの実装は、Webの画面・CLIの終了コード・Desktop IPCのUI処理へそれぞれ投影する。

| Edge identity | 分岐 | 評価に必要なデータ／不正な推論 |
| --- | --- | --- |
| C-E01 | C1 → CU | write-dispatched可能性、OperationID。HTTPなし＝副作用なし、としない |
| C-E02 | C2 → CR | typed revision/ETag、未保存差分。旧結果を新選択へ適用しない |
| C-E03 | C2 → CP | accepted-but-not-completed。202をCommit Receiptにしない |
| C-E04 | CO → CS | 実Operationに束縛された現在資格の独立Observation。単なる過去Receiptでは不可 |
| C-E05 | CO → CH | 未観測、Denied、stale、不一致。無条件replayを禁止 |

ServerとClientのGraph間を結ぶのは、**宣言されたResponse Projection/typed Resultの参照関係**であり、一方のEdge IDをもう一方へ流用しない。Traceには「どこまで実観測できたか」を付け、ブラウザ外／Proxy上の障害やクライアント未受領をserver-sideの応答生成ログだけで否定しない。

## 5. CLI/MCP/Studioを同じ設計Contextから駆動する

既存CLIには`commands`・`model catalog`・`inspect`・`trace`・`validate`等があるが、**Decision Graphの読み込み・完全性照会・実行経路可視化が完成したとは主張しない**。

1. #3262 Design Context Queryで必要なSubject、Node/Edge、既存Function/Action、Guard、Scenario、Source/Revision、権限Scope、covered/required集合を漏れなく**scope相対で**取得する。取得未了／権限による秘匿を明示。
2. 標準Data Structure Graphから、read-onlyなGraph表示・Node/Edge表・Mermaidテキストを同じtyped refで導出。出力形態と欠落／不完全を記録。
3. その後にTest Authoring。#1180の保存済みScenarioを唯一の受入ルートにし、Test候補とAccepted Scenarioは区別。生成AIが追加テスト案を出す場合、独立Expectedを参照する。
4. Synthetic dry-run trace（設計上の想定経路）とactual HTTP trace（実通過の証拠）を別レコードで結ぶ。受信したGraph snapshot/fingerprint、Target build、Generator version、Provider Lock、Grant/Subject scope、request identity、Trace completenessを表示。実HTTP EvidenceをGraph上に重ねても業務Completionを自動確定しない。

Web/Desktop UIに先行して、AI/CUIがNode IDで参照し、Branchの説明／依存／変更Impact／反例を整形して取得できることをP-1とする。Schema未採用のCLIサブコマンド名を本書で作って実在機能と称さない。

## 6. 確認すべきConformance Matrix（現状すべて NOT RUN）

| ID | 最小独立Expected / mutation | 期待する検出 |
| --- | --- | --- |
| HDG-01 | typed Application Outcome → CLI/Web/Desktop | 意味同一、HTTPだけが状態の正本にならない |
| HDG-02 | Guard bypass edge / 逆接続 | Actionへ到達する経路のGuard欠落を構造的拒否 |
| HDG-03 | 409/412とIF-Match/ETag | 更新前提条件と一般競合の区別 |
| HDG-04 | 400/422不正Field、未知problem type、不一致status | 合法な拡張と契約違反、開示制限の峻別 |
| HDG-05 | Effect前/後で通信断 | Client unknown、二重再実行なし、Operation/Receipt照合 |
| HDG-06 | 202/partial/旧receipt | Accepted/部分効果/現在のObservationを区別 |
| HDG-07 | 429/503 Retry-After、Grant失効 | Retry hintが権限・安全な再送へ昇格しない |
| HDG-08 | 同名Node/多重辺、表・図→表 | ID/方向/多重性の保存、lossy逆編集拒否 |
| HDG-09 | Graph/Source/Build/Plugin revision変更 | semantic acceptance、build/trace/evidenceの失効を独立判定 |
| HDG-10 | 部分Graph取得、秘密ノード、MCP/Tool途中切断 | 完全性を偽装せず、許可外情報を返さない |
| HDG-11 | 遅延Response後にUI選択とdirtyが変化 | responseの誤配置、dirty上書き、旧epochの採用を拒否 |
| HDG-12 | Source等価・Provider異版でTraceだけ流用 | 実走行経路の証拠を旧Providerから借用しない |

Coverageの分母を **required node / edge / guard path / named outcome / client reaction / effect disposition / requirement / NFR**に分離し、C0/C1（コードの行/分岐）とグラフのbranch coverageを混同しない。未取得・unsupported・disabled・not-runをcoverage 100%へ暗黙で足さない。Known mutant（Guard bypass、unconditional replay、Receipt stale、対象Anchor混同等）に対する検出力を記録し、独立ExpectedとActualの循環参照を避ける。

## 7. 着手順・停止条件・決定を戻す場所

| 段階 | 最小証拠 | 成果を戻すOwner |
| --- | --- | --- |
| P-1 現行調査 | 実CLI/既存Graph/Host/Command Lifecycleから対象設計Contextの一覧・欠落・受領完全性 | #3262/#2620/#1158 |
| P0 Graph/Tableの同一性 | 保存Sourceを実Loaderが読んだ証拠、Node/Edge ID/属性/Relationの同一性、GraphとTableのDiff | #1124/#2497/#2620 |
| P1 合成Contract | 代表Read/Write各1件、Result→HTTP→Client Reactionの正/負例と独立Expected | #1656/#1158 |
| P2 実Go HTTP | route、Auth/CSRF/Origin、受信失敗、Operation/Receiptの独立照合 | #1043/#1174 |
| P3 Target横断 | 同一ScenarioからCLIとWeb（後続Desktop）を生成・実行。実Target分のTrace・Guard・UI/transport Evidence | #1180/#1163/#1824 |

**Go単一バイナリ／外部Host非依存**という既存方針を維持。Decision GraphがGo/Wailsそのものに依存したり、HTTPのためにTEI独自Hostを必須にしたりしない。局所的に効果があるgraph roleやOperatorはPluginとして提供し、組合せの適合性とテストは既存Plugin/Verified Composition BOMの枠へ戻す。

**実行を停止・未検証とする例**：必須Guard証拠・生のAuthority欠損、Unknown commit、Receiptの実操作不一致、別revision/GrantのTrace混合、Graphを実Loaderが読まずに見た目だけ描いた、Secret漏出の疑い、部分取得の欠落、独立Expectedなし、Target固有の証拠なし。明示的に scope を絞った局所検証は継続可能とし、局所PASSを全体の成立へ昇格させない。

## 8. 2026-11-07 P-1a：現行コードに対するread-only Gap監査（**実コード読取のみ**）

この監査は[現状資料](../../docs/status.md)・該当Sourceを同じ`main`から照合したもので、テストや新Parserの実行ではない。特に「構成Graph」は「HTTPリクエストの分岐実行Graph」と同じではない。

| 実在箇所（観測したGit Blob） | 読み取れた事実 | 今回の要求に不足する部分／次のOwner |
| --- | --- | --- |
| `reference/web-composition/plan-check.mjs` (`cadfe94900fae4a7adc4fe99ddd27d27e087b5ea`) | `composeRequestPlan`は契約`requires`で必須GuardのActionへの因果的な祖先関係を検査。ルート分類、ステップID、Provider競合、依存DAG、順序の循環を検査する | **これは構成時のDAG**。実HTTPのYes/No/short-circuitをたどるDecision Graphではない。Terminalイベントは`onTerminal`の全集合の存在確認で、各Terminal経路が監査を通ることの証拠ではない。#1043 |
| `reference/web-composition/go-host/host.go` (`407aadbb87fdee4f59f96cf2d2b2f8115ed31281`) | Session verifierとContext markerが合成Host側境界にある。Contextの現在資格を説明する仕組みがある | 実IdP/Browser/Proxyの安全性や、分岐Graph上でのRuntime Guard通過とClient受領は、この静的読取だけでは確定しない。#1043 |
| `reference/web-api-contract/README.md` (`af7a96edbdab04f2f93a56d39fbff2fa6d14f2ad`) | Read/Command Binding、実Driver Resolverと合成認可・Scoped Providerへの接続が分かれている | RFC 9457 Response Mapの正式採用・Actual HTTP Handler／Client Reaction／Effect recoveryは未成立とのREADME記述。#1656 |
| `docs/status.md` (`ab349b8ea778adb550be4600c17c99de21821fc4`) | Command `dispatched/observed/unknown`、`persistence=not-applicable/not-observed/receipt-observed/resource-observed`、実PostgreSQLでの一部receipt/read試験を記載 | HTTP 2xx／HostによるCommit呼出し／Receipt確認／独立Resource確認の4者を一つの「成功」に統合できない。#1174/#2999 |
| 本文のGraph/Table例 | 保存可能なID・Edge・Branchに必要な属性を定義した | **TEI実Loaderが保存した受理Sourceを消費した証拠がない**。構想上のMermaidを実TEI Graphとしてカウントしない。#2620/#2497/#3262 |

**静的に発見した限定Gap（新規テストは未実施）**：`composeRequestPlan`は`terminalEvents`をStepから集めて集合包含で検査するが、「`rejected`に到達する**すべての実行経路**で実際にAudit/Recoveryが発火するか」は判定していない。構成の必須証拠とRuntime Path Coverageを分離する必要がある。現在の実コードに脆弱性が実在すると断定するものではない。

P-1bの第一歩として、[HDG-21 characterization test](../../reference/web-composition/plan-check.terminal-gap.test.mjs)を追加した。①Guard依存DAGが成立する一方、②terminal宣言だけを持ちActionの因果祖先にならないStepを置いた場合、③現行E1構成器は宣言済Terminal集合を受理する。独立した`requires`祖先探索では`invoke`へのAudit前提がないことを確認する。実HostがTerminal eventを別方式で必ず監査する可能性は残るため、この局所GapをRuntime脆弱性と呼ばない。

**限定実行Evidence（2026-11-07）**：GitHub取得Blob `plan-check.mjs=cadfe94900fae4a7adc4fe99ddd27d27e087b5ea`、`plan-check.terminal-gap.test.mjs=e2fd7f571783f64332649dfb275c28e6e803aff2`をV8上で直接評価し、2件PASS/0 FAIL。Nodeの`node --test`、実Host分岐、Goテスト、実Provider・実Clientは**NOT RUN**。このPASSは**現在の静的Plannerの成立範囲を表すCharacterization**に限り、Audit Coverage適合のPASSではない。

## 9. Failureを三つの直交軸で表す：知っていること・運ぶこと・できること

Serverの応答生成に成功してもClient受信に成功したとは限らず、Clientが2xxを受信しても業務上の結果は確定しない。このため以下を**既存Result/Receiptの読み取り専用な投影**として比較する。

| 軸 | 区別する状態の例 | 独立に必要な証拠 |
| --- | --- | --- |
| **Effectの確定性** | 実行前の拒否が確定／効果が確認された／部分適用／未確定・不明 | Effect Fence・Operation identity・Receipt・資源再読取、同じResource/Versionの条件 |
| **Transportの受領性** | 応答未生成／Host生成／送出試行／Client全体受領／Parser解釈失敗 | HostとClient双方のtrace、payload framing/headers、response bodyの完全受領 |
| **Client現在資格** | Context最新／選択epoch・baseRevision不一致／Grant失効／Actionの再試行資格不明 | Instance anchor、dirty set、current policy/grant、Retry contract、現在のContext |

この三軸は単純な大小関係ではない。たとえば`Receipt observed`は当該操作の結果に関する証拠であり、`Resource observed`はある時点の資源状態の証拠である。両者は異なる観測対象を持つため、無条件の全順序や、`status=success`への収束を禁じる。

**再送の決定**：呼出し側・HTTP Client・Proxy・Service Worker・Runtimeの再試行動作をすべて射程に入れ、単なる`Retry-After`や`Idempotency-Key`の存在を安全な自動再送の証明としない。RFC 9110 §9.2.2の非冪等メソッドの制約が基線。実業務のEffect Contract、同一Operationの重複抑制範囲、現在Authorization、payload指紋、Provider側確定・結果照会の可用性を確認する。POSTでも明示的に冪等な業務契約がある場合は再試行可能性を検証してよいが、**未知Commitの自動再送は既定禁止**。安全な再照会は業務Actionの再実行とは区別する。

### 9.1 例外の優先順位と秘密情報の非干渉

同一Requestに不正JSON・禁止Resource・Revision不一致・無効Sessionが同時に存在する場合、どの情報が外部へ見えるかは単なるStatus優先順位表からは導けない。Context検証・認証／対象開示・Action認可・Validation等の**順序と短絡**を、Hostごとの信頼境界と開示Policyで確定させる。未認証のCallerにSecret Field名・存在・Count・ETag・Transaction情報を返さない。

二つのWorldでPublic情報が一致しPrivate対象が異なる場合、Clientへ出るProblem Type、Detail、Headers、Trace、Graph TableのID/Count、Timingによる観測差を限定比較する。Timingの完全な非干渉は難しいため、初期は意図しない明白な差異を検出する負例とし、本番の情報漏洩不存在を主張しない。

## 10. Trait義務・標準Plugin・検証済みBOMへの引渡し（候補）

他の設計で検討中の**Traitを原子的な実装・検証契約**として扱う発想を、HTTP Graphにも適用できるか比較する。Traitの正式なDefinition/Intent/Observation/AssessmentのOwnerは既存の[Draft #3027](https://github.com/hat47x/tei-platform/pull/3027)等であり、本書で第二Trait体系を作らない。

| Trait相当の義務（**識別子は例示**） | 具体的なFulfillment | 必須のConformance |
| --- | --- | --- |
| GuardEvidence | 既存`security.authorize`等の既定/独自実装、実Guard witness | 全Action到達経路の因果Guard；宣言だけのGuardは拒否 |
| TerminalCoverage | Serverが各実行経路でAudit・Responseを行う | short-circuit、内部Failure、Client切断、Response headers送出後の失敗を区別 |
| TypedResponseProjection | Application Result → HTTP status/body/Problem Type | JSON schema、HTTP semantics、redaction、未宣言Outcome拒否 |
| ClientReactionPolicy | typed Result＋Action効果＋受領資格 → 対応動作 | 書込みunknown時の再送防止、dirty/epoch保護 |
| EffectReconciliation | Operation identity / Receipt / Resource観測 | Provider固有のObservation保証、Grant現在資格、部分成功 |
| TraceEmission | Graph Node/Edge、Operation、Build/Plugin Lockに束縛したtrace | 観測された経路とExpected/モデル上の経路を混同せず開示制限 |

この表は**義務と実装が一対一である必要を意味しない**。Traitは`Definition → Binding → Fulfillment（Generated / Provided / Manual / External）→ Evidence`を分け、手動実装でも実装義務とテストを省略しない。Providerが対応宣言しただけでは実行保証としない。

**Dogfoodingの順序**：CLI/Studio開発環境のHTTP/IPC Error Adapterを標準Plugin構成候補として利用し、同じTraitとScenarioを生成アプリTargetへ適用。Host-specificの認証・Origin・Cookie責務やLocal IPC障害はAdapterに保持する。新しいHTTP Response Registry、全Client共通Framework、Universal Hostを生やさない。

**BOMへの引渡し**（#3267/#951/#1251）：`Client Reaction Provider × HTTP Response Provider × Host Guard Provider × Effect Provider × Target/version`という**exact組合せ**に対してBuild/Artifact/Runtime Conformanceの実際の範囲を記録する。2×2の最初の行列には、`Problem Typeを追加したが旧Clientが未知typeを安全に扱う`、`Host Guard更新で必須経路が途切れない`を含める。型だけの互換性やSchemaのハッシュ一致では、Effect/安全性の互換性は宣言できない。未選択Plugin追加時にRuntime Buildが変化しないかも比較する。

## 11. RFC／Transportの細部とGraphに追加する反証

**RFC 9457の厳密化**：`type`は一次識別子、欠落時は`about:blank`。未知Problem TypeはProtocolエラーと決めつけず、既知の安全な一般処理へ落とす。**未知extensionは無視すること**が規格上の原則であり、未知extensionがあるだけでレスポンスを拒否しない。既知拡張の型が間違っていれば、それに依存するReactionを実行せず安全な処理へ落とす。`status`はadvisoryで、Origin generatorはHTTP Responseと一致させる義務があるが、中継で書換えられたケースでは両者が不一致になる。クライアントは不一致を診断し、業務効果や再送資格をどちらからも推定しない。出典：[RFC 9457 §3](https://www.rfc-editor.org/rfc/rfc9457.html)。

| 圧力ケース | Expectedに含める区別 | 既存Owner |
| --- | --- | --- |
| HDG-13 | `type`省略＝`about:blank`、未知extensionはignore、未知typeの一般安全処理 | #1656 |
| HDG-14 | actual HTTP statusとProblemの`status`不一致、Proxyのstatus書換え | #1656/#1043 |
| HDG-15 | `401`のWWW-Authenticate、`403/404`の情報開示差、`428`前提条件要求 | #1043/#1656 |
| HDG-16 | `204/304/HEAD`のbody非依存、cache/ETagの鮮度、Response statusと業務状態の混同 | #1656/#2582 |
| HDG-17 | Body/Streaming途中切断、headers送信後の失敗、Client断のServer実行継続 | #1043/#1174 |
| HDG-18 | HTTP client, proxy, service workerの再試行とIdempotency契約の不一致 | #1174/#1145 |
| HDG-19 | 2 WorldのPrivate情報差がError body/header/trace/Graphへ漏れる | #2516/#1178/#1043 |
| HDG-20 | Target別OpenTelemetry traceのstatus意味とBusiness Outcomeを混同 | #1163/#2845 |
| HDG-21 | あるTerminalの`onTerminal`を未到達Stepに付与しても総集合だけ成立する | #1043/#3274 |
| HDG-22 | Provider A/B更新・BOM 2×2でClient Reactionが変質、未知typeで無根拠retry | #3267/#1158 |

`428`は全APIへの義務ではなく、HTTP前提条件を要求するprofileの一候補。`304`は再利用可能な既存Representationの鮮度判定に依存するため、単独の成功業務Payloadとして解釈しない。`204`やHEADのbodyなしは通常のProtocol経路であり、Problem Detailsがないだけでエラーにしない。

**TelemetryとEvidence**：OpenTelemetry HTTP Semantic ConventionsはHTTP Serverの4xxを常にSpan Errorへ写すものではなく、HTTP Client/Serverの文脈差もある。したがってSpan Status→Business Denial/Effect dispositionを逆推定しない。保存するtraceの`http.route`は高Cardinalityの生URIを避け、Secret/個人情報を必要範囲外へ出さない（[OTel HTTP Spans](https://opentelemetry.io/docs/specs/semconv/http/http-spans/)）。Graph/TraceのSource identityとRun identityは別参照とする。

## 12. 独立Expectedの固定と次の一手（Issue #3274）

既存HDG-01～12と追加HDG-13～22は**要求候補**で、実テストの合格ではない。最初の実行可能な対照は、`reference/web-composition/plan-check.mjs`の**構成DAG**と実HTTP **Decision path**の違いを明示する`HDG-21`を優先する。

1. **P-1b（限定V8 Characterizationを追加）**：現行`plan-check.mjs`でGuard因果関係とTerminalイベント集合を分離する2ケースを実評価。次はリポジトリ実ファイルを`node --test reference/web-composition/plan-check.terminal-gap.test.mjs`で走らせ、同一HEADのRunner exitと限定Evidenceを記録する。疑義を実Runtimeの監査迂回と断定しない。
2. **P0**：必須Guard・Terminal Auditについて、実分岐の経路が観測できる場合だけPath Coverageを判定。未知・未ロードのBranchを見なかったことにしない。対象が構成DAGのままなら「経路の静的検査はできない」という明示Resultを返す。
3. **P0/P1並行**：HTTP Client/Server/Recoveryの三つのGraphに、責務・Version・開示・Effect stateを含むread-only Diagram/Tableを同じModelから投影。現行Loaderに不存在ならまずそのGapを確定し、別のGraph専用正本を作らない。
4. **P1**：成功・Validation・競合・Unknown Effectの4系統から`typed Result → Transport Mapping → Client Reaction`の最小Contractケースを選ぶ。Problem Detailsの拡張互換を独立Expectedで評価。
5. **P2/P3**：同じOperation・同じPlugin/BOM・同じGraph Revisionの実Go HTTP/Studio IPCとRecovery Observationを比較。Sourceが同じでもBuilder/Providerの違うtraceや、受領できなかったTool結果をPASSの根拠に混ぜない。

**受入の反証条件**：独立ExpectedをActualから作った／Loader未消費の図をCanonicalと称した／短絡TerminalからAuditまでの証拠が途切れた／有効な旧Clientの未知extensionを不正として拒否した／unknown commitをRetry-Afterで再実行した／別TargetのGrantや旧Receiptを現在へ使い回した、のいずれかが検出されたら該当資格は不成立。最小作業単位で修正し、合成Fixture PASS、実Host Path PASS、実Provider Effect PASS、Owner正式受理を別に記録する。

## 13. 2026-11-07 後続横断：設計Contextの実Consumer、Graph受領資格、回復・Plugin版の合成

前段で「Design Context Queryは未実装」と表現していた箇所は、**製品の完全な共通Queryが未実装**という意味に限定する。現在、[#3262](https://github.com/hat47x/tei-platform/issues/3262)には[merged #3284](https://github.com/hat47x/tei-platform/pull/3284)および[merged #3296](https://github.com/hat47x/tei-platform/pull/3296)の既存CLI応答を入力にした**試験用read-only Context Consumer**がある。#3296の専用CIに16件のsynthetic／transport負例が記録されている。ただし、Graphの全Node/Edge・Business Rule・依存Sourceの必要閉包を資格化したものではない。局所Go C0やT0/T0.5の合格を「Graph実Loaderが意味を受け取った」とは扱わない。

**設計書の出力が存在することと、Consumerに必要な関係が渡り切ったことは別である。** 今回のHTTP Graphに関しては、最低でも次の**異なるID・証拠**を分離する。

| 別個の識別と証拠 | 必須の確認／失ってはならないこと |
| --- | --- |
| Semantic Node/Relation occurrence | 元Sourceのstable ref、役割、Source revision、必要な重複と向き。同じ表示ラベルの別Nodeは別 |
| Graph edge / branch instance | 多重辺のEdge ID、from/to、Branch条件、Evaluation context。表・Mermaidの描画用IDを元Relation IDへ逆流させない |
| Response / Client Reaction binding | Action/Outcomeの型付きref、Problem typeとHandler版、クライアントのInstance anchor・Selection epoch |
| Actual execution path | 実際のHost／Target／Plugin artifactとRun/Operation ID、同一Attemptの通過Edgeと受領状態 |
| Qualification/Disclosure | Required vs discovered/loaded/parsed/bound/returned、Tool complete/end marker、Policy/Grant revision、現在の開示可能範囲 |

この区別は、#2497の「構造の非単射なProjectionから原Sourceへ書き戻せない」、#1124の「古い応答の適用拒否と**既表示**の開示失効は別」、#3262の「CLI応答の個数や`ok:true`は必要Source閉包を証明しない」という反例に基づく。Graph v4等の関連ConsumerでEdge表現が改善されていても、**実TEI Loader→現行Graph→実Consumer受領**がこのHTTP案件で成立したとはまだ主張しない。各Consumerの実Source/Graph版と結果を個別に資格化する。

### 13.1 Client Reactionの二段階判定：理解できることと、今できること

1. **Response interpretation**：Transportで全Responseが受領されたか、Media Type・Envelopeが適格か、既知のProblem Typeまたは安全な未知Type処理へ分類できるか。ここでは業務効果の現在値や再実行許可を推論しない。
2. **Reaction qualification**：同じOperationに属するEffect disposition、Receiptと独立Read、現在のSubject/Target/Grant、User Action/Dirty Patch/Instance epoch、Provider実装のConformanceを照合し、表示・再認証・再取得・結果照合・明示再試行の可否を決める。

例えば`401`が到着しても、直前の**write**要求がすでに適用されたか分からなければ、再認証後に**同じ注文を自動POSTしない**。先に同Operationの結果を照合する。更新競合後の再取得も、別Instanceへの旧Response上書きやdirty消失を引き起こさない。

[#1174](https://github.com/hat47x/tei-platform/issues/1174)では [merged #2853](https://github.com/hat47x/tei-platform/pull/2853) により、**特定の**`CommitUncertainError`が`COMMAND_COMMIT_UNCERTAIN`へ伝播する現実装の限定修正がある。したがって「CommandがすべてUnknownをExecution Errorへ丸める」は現在の事実ではない。一方、**すべてのProvider／HTTP切断でUnknown伝播・再照合が成立した**という主張もできない。現行Command `dispatched / observed / unknown`、`persistence`の区分をそのまま受け渡すのが先で、Graph内に第二Lifecycleを実装しない。

### 13.2 Trait/BOM：実装契約と現在の資格を別々に検査

[#3279](https://github.com/hat47x/tei-platform/issues/3279)と[#3278](https://github.com/hat47x/tei-platform/issues/3278)のAtomic Trait案を引用し、`TerminalCoverage`、`TypedResponseProjection`、`ClientReactionPolicy`、`EffectReconciliation`を**原子的な義務候補**として比較する。義務を満たす実装がGenerated/Manual/Providedのどれであっても、同一Conformanceに通る必要がある。単なる`retryable`ラベル、Provider宣言、BOMの型整合を現在のEffect/Grant保証へ昇格させない。

[#3267](https://github.com/hat47x/tei-platform/issues/3267)の2×2版行列を借用し、例えば`Client V1/V2 × HTTP Response Provider A/B`で未知のProblem Type、Extensionの追加、既存ClientのUnknown-write停止を測る。同じProvider/Consumer version名でも実Artifact/Target/Source/Scope/Grantが違えば再資格化が必要。**本件専用BOM Registryは作らない**。

### 13.3 追加する独立Expected（全件NOT RUN）

| ID | ケース | 独立Expected・停止条件 |
| --- | --- | --- |
| HDG-23 | Design Context CLI replyが全件parseableでも一つの必須Rule/Branch未Loaded | `complete=false`／テスト生成資格なし。未取得を0件と扱わない |
| HDG-24 | 同ラベルNode二つ＋並列Edge二本をGraph→表→Mermaidへ投影 | Semantic ref・Edge occurrence ID・方向・Branch条件保存。Lossy表示から自動逆Patchなし |
| HDG-25 | POST送出後、サーバが401／Proxy disconnect、後続Clientが再認証成功 | 旧Writeの効果は未確定のまま照会。認証成功だけで再POSTしない |
| HDG-26 | 412応答の後でSelection epoch変更・Dirty Patch保持 | 旧対象のConflictとして保存し、新対象へ差分適用・Dirty上書きをしない |
| HDG-27 | 公開Graph同一、Private Guard/Routeだけ異なる二World | Public Graph/JSON/Problem/診断・件数・TraceにPrivate情報を漏らさない。未測定のTimingはunknown |
| HDG-28 | Provider A/B × Client V1/V2、未知Problem Type／旧Receipt | 正確な4 tupleそれぞれの独立Expectedで合否・SKIPを分け、旧証拠を流用しない |

これらのGraph/Client/Permission/Plugin Matrixは**Ownerへの適用ケース**であり、汎用Core State・専用SchemaをこのIssueで決定しない。

### 13.4 実行可能性の前進：HDG-21正式Node Runner

以前のHDG-21はGitHub Sourceを隔離V8に取り出した2件のCharacterizationだった。今回、同じ実Source・同じテストを**Node内蔵test runnerで実行する限定CI Workflow**を追加する。[Workflow](../../.github/workflows/tei-http-decision-characterization.yml)は、PRとmainの関連Source変更時だけ実行し、checkout HEAD・Planner Blob・Test Blob・Node version・`node --check`・`node --test`の結果を記録する。**CIの実成否が取得できるまではNode本RunnerはNOT VERIFIED**。PASSしても合格するのは構成DAGのCharacterizationだけで、Runtime Audit Path・実HTTP・実DB・Graph Loaderの資格にはならない。

**次に必要な最小一往復**：必要なHTTP Graph Contextの取得→Source/Graphの本物のLoader消費→Node/Edge/Function/Guardの同一性確認→静的構成とshort-circuit経路の差→既存Go HostでのRuntime audit/witness→ClientでのResponse受領＋Effect unknownの保持、という順に段階分けする。1段階だけ成立しても次段階へ無条件に昇格させない。

## 14. 2026-11-08：実Node完了・実Go Hostの分岐観測へ進む（P-1c）

### 14.1 前回CIの完了資格を更新

PR [#3382](https://github.com/hat47x/tei-platform/pull/3382) exact head `c31ade7851a4de5576845a88ee2ac0e9f2772f56` に対する[専用CI #42396614454](https://github.com/hat47x/tei-platform/actions/runs/42396614454) は**completed/success**。実Nodeのcheckout・版記録・`node --check`・`node --test`各Stepはsuccess。これによりHDG-21の**現在の構成器の限定Characterization**は、V8だけでなく実Node test runnerの成功として報告できる。ただしSourceで明示される`configurationAccepted=true, pathAuditProof=false, runtimeAudit=unknown`を上書きしない。

同PRの[全体Health #42396614480](https://github.com/hat47x/tei-platform/actions/runs/42396614480)は**completed/failure**。Jobログでは`go vet ./...`、`go test -count=1 -timeout 60m ./...`、参照Workspace validateがそれぞれPASSし、失敗したのは`gofmt -l . is empty`（`tools/reference-plugin-capability-coverage/coverage_test.go` の整形ドリフト）。**当該PRのGo/HTTP機能が全体資格を獲得したという意味ではない**。整形ドリフトの所有範囲と、HTTP実証の対象変更は分離する。

### 14.2 現行Go Hostの実装を読んだFit/Gap

現行 `reference/web-composition/go-host/host.go` blob `407aadbb87fdee4f59f96cf2d2b2f8115ed31281` は `NewHost`で必須Callbacks（`VerifySession`・`VerifyCSRF`・`Authorize`・`Invoke`・`Audit`）の非nilを要求し、**各request handlerに`defer`で`AuditRecord`を一回出力する**実装を持つ。認証失敗・Issuer差・CSRF拒否・Authorize障害・Handler成功・panicの経路を通る。既存`host_test.go` blob `d1abb5db50ec523c61a784c8a5dad30c1779c93c`には、拒否Pathの監査、Loopbackの202/403、レスポンス開始後のpanicを検査するGo試験がすでにある。

**以前の『実Hostの監査は全く未検証』という表現は不正確**。正確には、合成Go Hostに**監査の実装と既存試験はある**が、**E1静的構成DAGの`onTerminal`宣言がこのGo Hostの実際の分岐/監査にBindingされ、すべての必要なRuntime/Effect経路で強制されるという縦断証拠がない**。ここを#1043が担当する。独立Graph Runnerや第二Audit Engineを作る理由にはならない。

さらに、`AuditRecord.Stage`はこのHostでは**HTTP Handler処理の終了状態**であり、正式Business OutcomeやEffect確定状態ではない。HTTP 202と`Stage=completed`が同時に現れても、Business Commit/独立read証拠は存在しない。Handlerが202を送出してからpanicすると、**Client visible statusは202のまま、Host監査は`Stage=error`**になり得る。この差は既存の`TestPanicAfterResponseStartedRemainsAuditError`にも現れる。Problem Details/WWW-Authenticate等のProtocol conformanceは、実際の採用Auth Scheme・HTTP API profileが未決の段階でこのsynthetic Hostへ一律強制しない（#1656）。

### 14.3 Go Hostの分岐Witnessを同じ受入軸へ追加

[新テスト `decision_path_test.go`](../../reference/web-composition/go-host/decision_path_test.go)は、現行Go HostのCallback実通過順とAuditを**独立した期待順序**で比較する（Go HTTP Handlerを実際に動かす）。実行対象と独立Expectedは次の10経路に限定する。

| Path ID | 期待HTTP | 通過Callbackの要点 | 最低限の資格 |
| --- | ---: | --- | --- |
| HDG-H01 | 404 | Auditのみ | route不一致で他Guard不実行 |
| HDG-H02 | 405 | Auditのみ | method不一致でAction不実行 |
| HDG-H03 | 401 | Session→Audit | Session拒否 |
| HDG-H04 | 401 | Session→Audit | Issuer不一致 |
| HDG-H05 | 403 | Session→CSRF→Audit | Origin/CSRF拒否 |
| HDG-H06 | 403 | Session→CSRF→Authorize→Audit | Action認可拒否 |
| HDG-H07 | 503 | Session→CSRF→Authorize→Audit | 認可Service障害 |
| HDG-H08 | 202 | Session→CSRF→Authorize→Invoke→Audit | `Audit.Stage=completed`だがEffect未確定 |
| HDG-H09 | 422 | Session→CSRF→Authorize→Invoke→Audit | Handler呼出後の拒否、no-effectは未証明 |
| HDG-H10 | 202 | Session→CSRF→Authorize→Invoke→Audit | Header送出後panic、`Audit.Stage=error` |

このWitnessは**Go Reference HostのCallbackのみ**の実観測。Route/Methodの内部処理、Authorization Providerの本物の妥当性、任意のMiddleware/Proxy、Browserでの完全受領、実業務Action Effect、TEI標準Graph Loader、全可到達Pathの網羅性は証明しない。Sourceに安定Node/Edge IDを新設していないため、表のPath IDも**テストケース識別子**に限り、TEI Canonical Edge identityへ昇格させない。

既存専用Workflowは**NodeのHDG-21限定テストとGo Hostのこの限定経路試験の双方**を同一checkoutで走らせる候補へ拡張する。結果が得られるまではGo新テストを実Go PASSと記載しない。全Repo HealthとGo Host局所PASSは別資格とする。

### 14.4 他チャットで成熟した計画との統合と、次の一辺

- **#3262 / merged #3354**：実CLI11種と必要Validation/Relation/Sourceの限定Context Consumerを活用するが、そのTT-56の一部のSource集合が**HTTP GraphのRequired Node/Edge/Guard分母**に一致するわけではない。`required/discovered/loaded/parsed/bound/returned`をHTTP対象で独立に固定し、未消費GraphはUNKNOWNのまま保留する。
- **#1124 / #2497 / SUI・SHO Consumer**：同名Nodeと多重Edge、Graph→表→Markdownの非単射、Canvas/OccurrenceのUI選択とSemantic Subjectの区別を維持。CLIから実Graph Edge identityがLoaderを経由して来るまで、Mermaidの見た目やテストPath名で欠落したRelationを補わない。投影表示の現在資格と、過去に受信したエラーの保持を別に検証する。
- **#3278 / #3279 / #951 / #3267**：`TerminalCoverage`と`ClientReactionPolicy`をatomic Traitの**義務候補**として保持する。Plugin宣言・Provider選択・実呼出・Conformance・Runtime Grant・BOM exact tupleを区別し、標準Pluginへ移すのは現Hostの実Callback Evidenceから必要な不足が特定できた後。
- **#1174 / #1158 / #997 / KON/SEI**：ClientはBusiness Decisionの正本でも分散Effect coordinatorでもない。将来KON Dataspace等がData Provider、SEIが意思決定先となっても、Response Interpretation → Reaction Qualification → 認可済みAction/Effect → 独立Observationの分離を維持し、それら別製品のHostや認可正本をHTTP Graphへ移植しない。

次のP0最小縦切りは**既存Go Hostの10経路Witnessを基線に、E1構成DAG側のRoute/Guard/Terminal宣言とRuntime実装をread-onlyの照合票で結ぶ**こと。対応が証明できない項目は`unbound/unknown`として停止し、同名のGuardやHTTP statusからBindingを捏造しない。続いて1 Functionの実TEI Source Reader→Required Relation/Edge/Guard→Graph/Table投影を実Consumerで通してはじめてP0 Graph資格を評価する。既存の三視点・Scenario・Effect EvidenceのOwnerを維持する。

## 15. 参考資料と採否
## 8. 参考資料と採否
## 13. 参考資料と採否
## 14. 2026-11-08：実Node完了・実Go Hostの分岐観測へ進む（P-1c）

### 14.1 前回CIの完了資格を更新

PR [#3382](https://github.com/hat47x/tei-platform/pull/3382) exact head `c31ade7851a4de5576845a88ee2ac0e9f2772f56` に対する[専用CI #42396614454](https://github.com/hat47x/tei-platform/actions/runs/42396614454) は**completed/success**。実Nodeのcheckout・版記録・`node --check`・`node --test`各Stepはsuccess。これによりHDG-21の**現在の構成器の限定Characterization**は、V8だけでなく実Node test runnerの成功として報告できる。ただしSourceで明示される`configurationAccepted=true, pathAuditProof=false, runtimeAudit=unknown`を上書きしない。

同PRの[全体Health #42396614480](https://github.com/hat47x/tei-platform/actions/runs/42396614480)は**completed/failure**。Jobログでは`go vet ./...`、`go test -count=1 -timeout 60m ./...`、参照Workspace validateがそれぞれPASSし、失敗したのは`gofmt -l . is empty`（`tools/reference-plugin-capability-coverage/coverage_test.go` の整形ドリフト）。**当該PRのGo/HTTP機能が全体資格を獲得したという意味ではない**。整形ドリフトの所有範囲と、HTTP実証の対象変更は分離する。

### 14.2 現行Go Hostの実装を読んだFit/Gap

現行 `reference/web-composition/go-host/host.go` blob `407aadbb87fdee4f59f96cf2d2b2f8115ed31281` は `NewHost`で必須Callbacks（`VerifySession`・`VerifyCSRF`・`Authorize`・`Invoke`・`Audit`）の非nilを要求し、**各request handlerに`defer`で`AuditRecord`を一回出力する**実装を持つ。認証失敗・Issuer差・CSRF拒否・Authorize障害・Handler成功・panicの経路を通る。既存`host_test.go` blob `d1abb5db50ec523c61a784c8a5dad30c1779c93c`には、拒否Pathの監査、Loopbackの202/403、レスポンス開始後のpanicを検査するGo試験がすでにある。

**以前の『実Hostの監査は全く未検証』という表現は不正確**。正確には、合成Go Hostに**監査の実装と既存試験はある**が、**E1静的構成DAGの`onTerminal`宣言がこのGo Hostの実際の分岐/監査にBindingされ、すべての必要なRuntime/Effect経路で強制されるという縦断証拠がない**。ここを#1043が担当する。独立Graph Runnerや第二Audit Engineを作る理由にはならない。

さらに、`AuditRecord.Stage`はこのHostでは**HTTP Handler処理の終了状態**であり、正式Business OutcomeやEffect確定状態ではない。HTTP 202と`Stage=completed`が同時に現れても、Business Commit/独立read証拠は存在しない。Handlerが202を送出してからpanicすると、**Client visible statusは202のまま、Host監査は`Stage=error`**になり得る。この差は既存の`TestPanicAfterResponseStartedRemainsAuditError`にも現れる。Problem Details/WWW-Authenticate等のProtocol conformanceは、実際の採用Auth Scheme・HTTP API profileが未決の段階でこのsynthetic Hostへ一律強制しない（#1656）。

### 14.3 Go Hostの分岐Witnessを同じ受入軸へ追加

[新テスト `decision_path_test.go`](../../reference/web-composition/go-host/decision_path_test.go)は、現行Go HostのCallback実通過順とAuditを**独立した期待順序**で比較する（Go HTTP Handlerを実際に動かす）。実行対象と独立Expectedは次の10経路に限定する。

| Path ID | 期待HTTP | 通過Callbackの要点 | 最低限の資格 |
| --- | ---: | --- | --- |
| HDG-H01 | 404 | Auditのみ | route不一致で他Guard不実行 |
| HDG-H02 | 405 | Auditのみ | method不一致でAction不実行 |
| HDG-H03 | 401 | Session→Audit | Session拒否 |
| HDG-H04 | 401 | Session→Audit | Issuer不一致 |
| HDG-H05 | 403 | Session→CSRF→Audit | Origin/CSRF拒否 |
| HDG-H06 | 403 | Session→CSRF→Authorize→Audit | Action認可拒否 |
| HDG-H07 | 503 | Session→CSRF→Authorize→Audit | 認可Service障害 |
| HDG-H08 | 202 | Session→CSRF→Authorize→Invoke→Audit | `Audit.Stage=completed`だがEffect未確定 |
| HDG-H09 | 422 | Session→CSRF→Authorize→Invoke→Audit | Handler呼出後の拒否、no-effectは未証明 |
| HDG-H10 | 202 | Session→CSRF→Authorize→Invoke→Audit | Header送出後panic、`Audit.Stage=error` |

このWitnessは**Go Reference HostのCallbackのみ**の実観測。Route/Methodの内部処理、Authorization Providerの本物の妥当性、任意のMiddleware/Proxy、Browserでの完全受領、実業務Action Effect、TEI標準Graph Loader、全可到達Pathの網羅性は証明しない。Sourceに安定Node/Edge IDを新設していないため、表のPath IDも**テストケース識別子**に限り、TEI Canonical Edge identityへ昇格させない。

既存専用Workflowは**NodeのHDG-21限定テストとGo Hostのこの限定経路試験の双方**を同一checkoutで走らせる候補へ拡張する。結果が得られるまではGo新テストを実Go PASSと記載しない。全Repo HealthとGo Host局所PASSは別資格とする。

### 14.4 他チャットで成熟した計画との統合と、次の一辺

- **#3262 / merged #3354**：実CLI11種と必要Validation/Relation/Sourceの限定Context Consumerを活用するが、そのTT-56の一部のSource集合が**HTTP GraphのRequired Node/Edge/Guard分母**に一致するわけではない。`required/discovered/loaded/parsed/bound/returned`をHTTP対象で独立に固定し、未消費GraphはUNKNOWNのまま保留する。
- **#1124 / #2497 / SUI・SHO Consumer**：同名Nodeと多重Edge、Graph→表→Markdownの非単射、Canvas/OccurrenceのUI選択とSemantic Subjectの区別を維持。CLIから実Graph Edge identityがLoaderを経由して来るまで、Mermaidの見た目やテストPath名で欠落したRelationを補わない。投影表示の現在資格と、過去に受信したエラーの保持を別に検証する。
- **#3278 / #3279 / #951 / #3267**：`TerminalCoverage`と`ClientReactionPolicy`をatomic Traitの**義務候補**として保持する。Plugin宣言・Provider選択・実呼出・Conformance・Runtime Grant・BOM exact tupleを区別し、標準Pluginへ移すのは現Hostの実Callback Evidenceから必要な不足が特定できた後。
- **#1174 / #1158 / #997 / KON/SEI**：ClientはBusiness Decisionの正本でも分散Effect coordinatorでもない。将来KON Dataspace等がData Provider、SEIが意思決定先となっても、Response Interpretation → Reaction Qualification → 認可済みAction/Effect → 独立Observationの分離を維持し、それら別製品のHostや認可正本をHTTP Graphへ移植しない。

次のP0最小縦切りは**既存Go Hostの10経路Witnessを基線に、E1構成DAG側のRoute/Guard/Terminal宣言とRuntime実装をread-onlyの照合票で結ぶ**こと。対応が証明できない項目は`unbound/unknown`として停止し、同名のGuardやHTTP statusからBindingを捏造しない。続いて1 Functionの実TEI Source Reader→Required Relation/Edge/Guard→Graph/Table投影を実Consumerで通してはじめてP0 Graph資格を評価する。既存の三視点・Scenario・Effect EvidenceのOwnerを維持する。
## 15. 2026-11-08 P0-R：E1宣言とGo Hostの明示Route照合（Source二視点・限定PoC）

### 15.1 今回の最小実験と資格の限界

前回統合PR [#3478](https://github.com/hat47x/tei-platform/pull/3478)の専用CI [#42461546535](https://github.com/hat47x/tei-platform/actions/runs/42461546535) は**completed/success**。その実行でNodeの2件Characterizationと、Go Hostの実Callback順序10経路がPASSした。これらは**別々の試験**であり、E1 `composeRequestPlan`のGuard/TerminalがGo Hostの同名実処理へ*選択・ロード・束縛*された証拠ではない。前段§14の「Go新テストは未実行」は**当時の記録**として読み、現在の限定証拠をこの節で更新する。PR全体Healthは本節作成時点では継続確認中であり、専用CIのPASSを全体Healthyへ昇格しない。

そこで[二視点の合成Fixture](../../reference/web-composition/e1-go-route-pair.fixture.json)を共有して、[実Node試験](../../reference/web-composition/e1-go-route-pair.test.mjs)と[Go Host試験](../../reference/web-composition/go-host/e1_go_route_pair_test.go)で、**現行コードが受け入れる宣言と、比較可能なRouteの差**を確認する。Fixtureに同じRoute/Actionが書かれていても、それは**Reference内で宣言上の値が一致する**ことを示すだけ。TEIの実Source Reader、正式Binding、Provider選択・実呼出、Endpoint生成の経路を通っていない。

| 独立に判定する段階 | 現在の実証／これからの実証 | 合格しても成立しないこと |
| --- | --- | --- |
| E1 declaration qualified | 実`composeRequestPlan`とrequired Guard DAG / Terminal label集合 | Terminal全経路強制、Runtime Grant |
| Go Host construction qualified | 実`NewHost`で構成拒否／受理を観測 | そのHostがE1選択結果から作られたこと |
| Route pair identity qualified | E1とGoそれぞれのRoute method/path/actionIdを独立比較 | Auth/CSRF/Issuer/Effect Providerの同一性、Source意味の正式受理 |
| Runtime path observed | 10経路のGo Callback順・Audit Stage | Graph実Edge、Client応答完全受領、業務Commit/部分適用 |
| Semantic / Effect / Consumer qualified | **未達**：実TEI Source→Role/Edge→Host Binding→Client/Receipt/独立Read | 前段の局所PASSから推測しない |

### 15.2 Counterfactual：構成双方が局所PASSでも対応付けはFAIL

| ID | 独立Expected（Fixtureは固定、変更は一方だけ） | 局所で成立すること／拒否する推論 |
| --- | --- | --- |
| HDG-B01 | E1のRoute `POST /scenario/submit` とGoのPath/Actionが一致 | 宣言Routeだけpaired。Runtimeは未資格 |
| HDG-B02 | E1またはGoのPathを別の**有効な**Pathへ変更 | 両者は各々構成可能、明示ペアの同一性はFAIL |
| HDG-B03 | 一方のAction IDを別の**有効な**IDへ変更 | 各側の構成可能性は残る、Action BindingはFAIL |
| HDG-B04 | Go Host MethodをGETへ変更 | E1 pairは不一致、Go Host自身もProtected Writeとして拒否 |
| HDG-B05 | Route等値のままGo Hostのissuer/providerを改版 | route pairingは同一でも、Subject/Provider/Grant/実Invocation資格はUNKNOWN |

これらは**テスト専用の比較関数**であり、公開Resolver、共通Graph Registry、第二Web Host、正式Schemaではない。今の`Graph Edge`識別子をRoute IDと同一にしない。#1124/#2497の同名Node・多重Edgeの原Source Occurrenceが実Loaderから届くまでは、分岐関係の`stable ID`を捏造しない。#3263の三視点Graph/CRUD計画はDraft段階なので、同PRにあるEdge identity検討をmain正式機能として数えない。

### 15.3 TEI/SHO/SUI/SEI/KONとの責務分離

今回の「構成＝実Bindingではない」は、他チャットの構想にもそのまま適用する。TEIのモデル・DSLがデジタルツインと意思決定候補を記述しても、SUIの提示済みCardやSHOの草稿、SEIへ送るメッセージ、KON DataspaceのObject/Revision保存は、**業務上のDecision採用・現在のGrant・実Effectの証拠ではない**。SUI Canvas上の表示EdgeやDocument内のリンクから元Relation/Authorityが逆復元できるとしない。

SEIを意思決定の主体とする構想は、本件では**Candidate decision→権限ある主体の判断→認可済みAction→現在Effectの独立観測**の境界として参照する。KONはデータ保存・履歴や取得Providerの候補であって、本件のAuth/Commit authority正本を代替しない。EKI等のMessagingがRetry/Redeliveryしても、HTTPのACKだけを理由に非冪等Actionを再実行しない。これらの製品の実接続・Ontology DSL実装・SEI意思決定の実運用は**NOT RUN**。HTTP案件専用の分散Transaction Coordinatorや第二意思決定主体を増やさない。

### 15.4 次のP0ゲート（Owner別、第二正本なし）

1. **本PRの局所実Node/Go CI**：HDG-B01〜05と旧HDG-21/Go 10経路についてexact HEAD / Source Blob / runner exitを確認。Failedまたは未受領なら本段階は不成立。
2. **#1043/#951**：実際のHost Compositionが`E1 plan→selected Plugin/provider→Go Host Policy/Callback`を消費・実結合する証拠を得る。現在のRoute二視点fixtureを本番Runtime入力へ昇格させない。
3. **#3262/#1124/#2497/#2620**：HTTP対象1 Functionで独立Required / discovered / loaded / parsed / bound / returned / disclosableを固定し、実TEI SourceからのNode・多重Edge・Branch/Guard・Relation occurrenceを同じConsumerが受け取る。部分・不明なら合格停止。
4. **#1158/#1174/#1656/#1180/#2514**：同じOperationのtyped Result→HTTP/IPC→Client Reaction→Receipt/独立Readを結合。202/401/422/Response loss/再送の各場合にExpectedとActual、Run Receiptを別々に照合。
5. **#3278/#3279/#3267**：上述の必要実装が存在すると確認してからAtomic Trait fulfillmentとProvider/Client exact BOMを検証。選択済み／宣言済みを実利用可能、またBOM同名を同一実Artifactと呼ばない。

**停止・撤回条件**：Routeペアの比較だけでSecurity/Audit/Effectが保証されたと記録する、同一FixtureからExpectedを自動生成して独立証拠と呼ぶ、E1/Host二側のSourceを本実装の単一Canonicalモデルへ無根拠に統合する、Scope/Grant変更に対し旧Bindingを再利用する、これらのいずれかが起きた場合は資格を撤回。局所PoCの価値が低ければFixtureはtest-onlyに留め、プロダクト側へ移植しない。

## 16. 2026-11-08 P0-S：実WorkspaceのPattern→Capability参照をread-onlyで検査

### Source経路を一つだけ実消費する

前段§15のE1/Go二視点比較は**両側とも合成宣言**だった。今回は既存の`internal/operations.InspectSubject`（ADR-0048のApplication API方針に接続するread-only API）を、[実Goテスト](../../reference/web-composition/go-host/e1_workspace_source_admission_test.go)から直接呼び出す。Referenceの実Sourceは以下の二つである。

| 実Source（読み込み、変更しない） | Canonical Subjectと固定Expected | 段階 |
| --- | --- | --- |
| `reference/workspace/applications/order-entry/patterns/submit-order.tei` | `application-pattern:order_entry.submit_order`、`kind=command`、`capability=sales.order.submit` | 実Workspace Loader→正規化IR→Source Index→`InspectSubject` |
| `reference/workspace/semantic/sales/order.tei` | `semantic-capability:sales.order.submit` | 同上。独立Subjectとして取得 |

**独立して証明する最小条件**：両Subjectの`Canonical`が期待IDと参照先を保持し、`SourceStatus=indexed`で同SubjectのSource Reference・Artifactが取得されること。二回の`InspectSubject`が同じ`WorldFingerprint`と`SourceIndexFingerprint`を返すこと。さらに前段のtest-only E1/Go Fixtureが指定したActionIDが、**実Workspaceから得たPattern IDと一致**すること。正しいGraph Edgeを描いたと主張するわけではなく、`Pattern→Capability`という一つの実Canonically-authored参照を確認する。

**反例**：双方のRoute/Actionを同じ架空ID `order_entry.unknown_action`に変更すれば宣言ペアとしては一致しても、`InspectSubject(application-pattern,...)`の実結果は`SUBJECT_NOT_FOUND`であるべき。**一致した文字列からCanonical存在を逆推論しない**。そしてPattern/Capabilityを読み込めても、Runtime Providerや選択済みHostでの実呼出し、Action Authorization/Grant、DB効果は不明のまま（HDG-S03）。

| ケース | 実テストの対象 | この時点の独立Expected／限界 |
| --- | --- | --- |
| HDG-S01 | 実Workspaceの2 Subject＋E1/Go宣言 | Source位置とCanonical参照の限定一貫性。二回のreadは原子的Snapshotを保証しない |
| HDG-S02 | Route pairは完全一致、Canonical Patternは未存在 | `SUBJECT_NOT_FOUND`。Route一致をSource採用に昇格しない |
| HDG-S03 | Patternが実Capabilityへ参照している | Provider選択、実Invoke、実効果は**NOT QUALIFIED** |

既存`InspectSubject`は一件ずつWorkspaceを読む。fingerprintが同じことは内容一致の**限定観測**であり、Workspace Rootの移動・Source A→B→A・権限/Grant・TOCTOU・完全Required Closureの証拠ではない。実`*.tei`を先行し、未実証の`*.tei.md`完全Readerや独自Graph Loaderを仮定しない。#3492の進行中の三視点実Source入場作業へは共通OperationとSource Referenceの利用例として引き渡し、同PRの別の受入判定を奪わない。

**専用CIに追加**：実Go Host packageテストを継続し、対象`submit-order.tei`、`order.tei`、追加GoテストのGit Blobを同一checkoutで記録する。関連Sourceの変更でもCIを再実行する。CIでGo PASSを確認するまではHDG-S01〜03の実行資格は未成立。

### 製品縦断へ進むための停止線

- **済みの局所証拠**：実E1静的検査、Go Hostの有限分岐Audit、test-only E1/Go宣言ペア。今回の追加試験は、実Source Readerの一辺を確認する段階。
- **未実証**：E1宣言が実Loaded Application Patternから**生成**されたこと、実Resolver/Providerが`sales.order.submit`を選択・Invokeしたこと、実TEI GraphのNode/Edge occurrence一意性、Client Reaction、実効果、現在Authority、同一SnapshotのRequired Source closure。
- **次の最短一辺**：#951/#1043/#1158の現在のResolver/Host/Command受理契約を**実Pattern+Capability IDに基づいてread-onlyで対応表へ投影**する。Route名の一致から選択済みProviderを推測せず、`selected→loaded→invoked`を異なる証拠へ分ける。GraphのOwnerは#1124/#2497/#2620、Function完備性は#3262、exact Artifact/Plugin/BOMは#3267/#3279。第二Registryや第二Hostは設けない。

## 17. 2026-11-08 P0-B：実Workspace→既存DriverでCommand Bindingを解決する

### 実装の現在値と前段CIの資格

[前段PR #3501](https://github.com/hat47x/tei-platform/pull/3501)の[限定Node/Go CI #42464774706](https://github.com/hat47x/tei-platform/actions/runs/42464774706)、[全体Health #42464774715](https://github.com/hat47x/tei-platform/actions/runs/42464774715)、既知脆弱性確認 #42464774725は**同じ前段HEADでcompleted/success**。P0-Sの実Workspace Pattern/Capability `InspectSubject`はこの範囲で実Go PASS。前段§16に記載したQueuedは作成時点の履歴であり、現在の実行資格とは区別する。

現行 `internal/driver/resolver.go` と `internal/driver/binding.go` を実コード照合したところ、`ResolveCommandBinding(applicationID, patternID)` は、`ResolvePattern`でcommand→Capability参照を読み、`ResolveCapabilityBinding`でApplication-scopedな `CustomGoBinding` を実解決する。既存の`realization.ResolveCustomGoBinding`が対象Bindingの**0件・重複2件以上を拒否**する。実Source `reference/workspace/applications/order-entry/realizations/submit-order.tei` には `realize sales.order.submit with custom go` があり、正規化されたBindingの実Identityは `order_entry/sales.order.submit` である（ファイル名由来の実装推測ではない）。

今回[Go Referenceテスト](../../reference/web-composition/go-host/e1_real_workspace_command_binding_test.go)では実`workspace.Load`→`driver.NewResolver`→`ResolveCommandBinding`を**本物のDriver APIで実呼出し**する。二視点FixtureのAction ID・前段実SourceのPattern ID・Capability ID・Application-scoped Binding Identityを**独立固定Expected**で比較し、結果が同じ文字列になるだけでTarget Providerの実起動を断言しない。新しい独自Resolverや`Binding`製品型は導入しない。

| ケース | 入力・負例の加工 | 独立Expectedと保証の上限 |
| --- | --- | --- |
| HDG-R01 | 実Reference Workspace、`order_entry.submit_order` | `Pattern(command)→sales.order.submit→order_entry/sales.order.submit`の実Driver解決成功。Provider Adapter実ロード／Invokeは未資格 |
| HDG-R02 | test-only Worldから該当Bindingを削除 | 明示Bindingがなければ`ResolveCommandBinding`はエラー。名前やファイルパス推測による代替禁止 |
| HDG-R03 | 同じApplication/CapabilityのBindingを複製 | 曖昧な選択を拒否。任意一件への暗黙選択は禁止 |
| HDG-R04 | PatternのCapability参照だけ、別の**有効な**`security.authorize`に差し替え | Driverが別の有効なBindingを解決できても、固定Expected `sales.order.submit`とは不一致。構成可能性≠業務上の正しいAction |

Source Indexを前段P0-Sで読めても、このDriver解決は`Source Index→Application Bindings`の**実読込・解決の限定一辺**であり、既存`reference/workspace`の全部署・全Relation、原子的Source Snapshot、Versioned BOM、実Host Gatewayを証明しない。今回の負例はメモリ上で`World`をコピーして加工し、固定Gate B/Cの`*.tei`やBusiness Accepted Meaningを変更しない。

### 他チャットの統合境界

- **共通Application API / ADR-0048・#1158**：Application Pattern→Capability→Bindingの意味は媒体非依存。CLI・MCP・HTTP・Desktopはその選択と結果を利用するが、**宣言の解決＝実Adapter呼出し**ではない。共有OperationのConsumerとして実行し、媒体ごとに別のCommand/Policy Engineを作らない。
- **Plugin/Trait / #951・#3267・#3297**：`selected→loaded→invoked→received`を区別する。今回の`ResolveCommandBinding`は**resolved／declared**までであり、Plugin Provider Bの実選択／Adapter実呼出しや現在資格、BOM Artifactsは未実証。`CustomGoBinding`という既存方式から一般的な動的Plugin方式へ結論を飛躍させない。
- **Design Context / #3262・#3482**：`sales.order.submit`の限定Required Sourceが現行CLIで一部確認されていても、HTTP Graph Required Node/Edge/Guard・公開資格・単一Snapshot・全Function closureは未資格。新テストによって独立Expectedを「全Required完備」と誤認しない。
- **三視点Graph/SHO・KON**：Binding解決で得たPattern/CapabilityをGraphへ投影する場合、Canonical Relationshipと元Source occurrenceを保存する。Document/Graphの保存版、Runtime Bindingと現行Grantの鮮度を個別に検証する。#3492の三視点実Source ConsumerがこのHTTPテストによってPASSしたとはしない。
- **HTTP実行/Client Reaction / #1043・#1656・#1174**：次の一辺は、解決された`CustomGoBinding`が**実Reference Hostの選択済みAction Invocationで使われた**という同一Attemptの証拠。未到達ならunbound。HTTP 202やGo Audit completedでEffect Commit・Client受信を推測せず、unknown/Receipt/独立readを保持する。

**本PRの実CIが成功するまではHDG-R01〜04はNOT VERIFIED**。Go Host packageの本Runner成功後も成立するのは限定Resolverの正負例だけ。次に独立Target Provider/Adapter実呼出しを確認し、HTTP Handlerの有効経路とDecision Graphを突合する。

## 18. 2026-11-08 P0-I：実Go Host→既存Driver→合成Adapterの同一HTTP Attempt観測

### 18.1 前段の合格と、本段階の識別対象

前段[PR #3506](https://github.com/hat47x/tei-platform/pull/3506)の[専用CI #42470312187](https://github.com/hat47x/tei-platform/actions/runs/42470312187)（Node 7/7、Go Host package全PASS）、全体Health #42470312240、脆弱性確認 #42470312239 は**いずれもcompleted/success**。この結果により、実Workspaceからの`ResolveCommandBinding`によるPattern→Capability→Application-scoped Binding解決は局所資格を得た。ただし前段の成功を、新しいAdapter呼出し成功へ流用しない。

現行実装を追った結果、すでに`driver.Resolver.InvokeCapability`が存在し、Canonical `Capability`の`CustomGoBinding`を再解決して、呼出し側の`BindingAdapterResolver.ResolveAdapter`へ渡し、`capabilityinvoke.Invoke`でAdapter本体を実呼出し・Result契約を検証する。`NewCanonicalActionAuthorizer`は実Reference Workspaceと`security.authorize`の既存Adapterを利用する。したがって追加のAdapter選択Core／Web Command Engineは不要。

[新しいGo Test](../../reference/web-composition/go-host/e1_real_workspace_driver_invocation_test.go)は、この既存経路に合成`sales.order.submit` Adapterを挿入し、**実HTTP Handler内で**`NewHost`→実`NewCanonicalActionAuthorizer`→既存Driver `InvokeCapability`→合成Adapter `Invoke`→HTTP Status→Host Audit を一つのリクエスト内で観測する。ビジネス側Adapterは**テスト用関数**であり、DB書込・実外部Provider・永続Receiptは呼び出さない。

| Witness | 固定Expected（Callback順は独立宣言） | 認定すること |
| --- | --- | --- |
| HDG-I01 | Session→CSRF→実Canonical Authorize→Binding Adapter resolve→Adapter invoke→Audit、HTTP 202 | 実Hostから実Driverへ到達して正しいBindingで**合成Adapterの実呼出し**。HTTP 202は業務Commitの証拠ではない |
| HDG-I02 | Viewer認可拒否、403。Command Adapter resolve/invoke=0 | `security.authorize`の既存業務拒否はコマンド実行へ進まない |
| HDG-I03 | CSRF拒否、403。Authorize/Command Adapter=0 | Host owned guardによる短絡を保持 |
| HDG-I04 | Adapter resolverが障害、503。Invoke=0 | 解決失敗をBusiness deniedへ変えずExecution ErrorとしてHTTP側で処理 |
| HDG-I05 | Adapter Invokeが障害、503。Invoke=1 | Adapterまで到達したが結果は不確定。作用なしとは推定できない |
| HDG-I06 | Session拒否、401。Authorize/Command Adapter=0 | 未認証から業務権限を借用しない |
| HDG-I07 | AdapterがCanonical result mode違反、503 | Result契約違反を正常Business Outcomeへ昇格しない |

**重要な負例**：HDG-I05のHTTP 503やAudit Stage=errorは、Adapterへ渡った後の**業務副作用なし**を証明しない。HDG-I01の202/Audit completedも、Clientが全応答を受信したこと、業務上の永続Commitまたは独立Observationを証明しない。今回の`BusinessResult`はResultの**契約形状**だけを検証する。本番Credential、実OIDC、Browser Cookie、真のCSRF署名、外部DB/Service、分散再送の安全性は対象外。

### 18.2 他チャットとの接続を限定資格で表す

- **Plugin-first／Trait／Verified BOM**：宣言だけの`Binding resolved`から一歩進み、合成Adapterが実`ResolveAdapter`で返され、実`Invoke`が呼ばれたことを**同一HTTPリクエストのCallback列**で記録する。ただしAdapterはtest doubleでありPlugin Registry／Provider Artifact検証・署名・現在Grant・BOM exact tupleの資格ではない。既存Service CarrierとAtom TraitのOwner（#951/#3267/#3279）を維持する。
- **Client Reaction／Effect Unknown**：HTTP 401/403は認証・承認条件の現在の拒否に関わる。503はService障害と業務効果の不確定性を分ける必要がある。Adapter実Invoke後の503は**実際の適用有無がunknown**とし、Clientによる非冪等Commandの自動再実行を禁止する次の検証の入力へ渡す（#1158/#1174/#1656）。
- **Context Query／Graph・SHO**：実Driver経路を通ることは、HTTP Graphの**全Node/Edge・Guard BranchをSource Readerがenumerate**する証拠ではない。現在のCB実証をDesign Context Required集合、Source Occurrence、Graph/Table/Traceへ公開するには#3262/#3482/#1124/#2497側の実Consumerと同時に検査する。#3492三視点Graph/CRUD実Source入場は確認時点openであり、未マージ成果を現行main正式仕様と呼ばない。
- **SEI／KON・組織/Workflow**：候補判断、Authority判定、現在のUser・Operation Grant、効果発生、独立観測、文書保存の時点と出典を分離する。同じLifecycleを外部Pluginが拡張する場合も、「Auditの宣言名」だけで実Terminateを保証せず既存Hostの所有する必須処理を残す。第二のHost、分散Transaction Coordinator、PluginレジストリをHTTP分野へ持ち込まない。

### 18.3 次のP0の最短一辺

今回の局所Go Witnessが**実CIで成功した場合のみ**、`Host request→Canonical Authorization→Driver binding→Adapter invoked→Host audit`を合成Referenceについてqualifiedとする。その次は`Route/Action/Contract ID`と`Binding`に加え、実Pluginのselected / artifact verified / provider invoked / effect receiptを同一Attemptで確認する。実World Fingerprintの一致は現在Authorityを意味せず、Grant RevocationやPlugin/Toolchain改版時にはEvidenceを再評価する。Client Reactionまでの縦断検証で一つでも未取得ならその欄をunknownのまま提示する。実装全体のProvider、Graph、BOM、Execution Receiptへの採用Gateは引き続き未到達。

## 19. 2026-11-08 P0-E：HTTP結果とEffectの確定性を実観測で切り分ける

### 19.1 実行経路の成熟差分と最小実験

[PR #3516](https://github.com/hat47x/tei-platform/pull/3516)の限定[CI #42474818698](https://github.com/hat47x/tei-platform/actions/runs/42474818698)は、実`NewHost`→実`NewCanonicalActionAuthorizer`→実`driver.InvokeCapability`→合成Adapterの有限7経路でcompleted/success。これにより「Adapter到達」の局所資格を得た。一方、[同HEAD全体Health #42474818899](https://github.com/hat47x/tei-platform/actions/runs/42474818899)は**failure**。失敗箇所は`tools/repository-plan-check/TestPlanReadmesMentionEveryEntry`であり、並行追加された`AI_PERFORMANCE_TUNING_OBSERVABILITY_2026-11.md`を当時の`plan/design/README.md`が列挙していなかった。Go vetとWorkspace validateはPASS、専用HTTPテストはPASS。**現在mainのREADMEでは当該文書を列挙済み**であるが、異なるHEADを根拠に過去の全体CI failureをsuccessへ書き換えない。

ここまでの証拠だけでは、HTTP 202・503をEffectの有無やClient側再試行可能性に写像できない。新しい[Go Test](../../reference/web-composition/go-host/effect_unknown_observation_test.go)では、既存の実Reference Workspace、`NewCanonicalActionAuthorizer`、`driver.InvokeCapability`、`NewHost`を再利用し、合成`sales.order.submit` Adapterに**テスト専用状態**を持たせる。HTTP実行中には**独立readを呼ばず**、後続の`driver.ExecuteObservationRead`に明示的なread identityと`ObservationEvidenceSynthetic`を渡した場合にだけ状態が観測できるかを検査する。実DB Commit、Operation Receipt、Provider適格性を実装したものではない。

| ケース | 合成Adapter内の作用とHTTP表面 | 独立readの固定Expected | 合格範囲 |
| --- | --- | --- | --- |
| **HDG-E01** | 状態を`submitted`へ更新した後にerror。HTTP **503**／Audit error | 後続readは`submitted` | 503・Adapter errorから**未適用**を推定できない |
| **HDG-E02** | 更新せず正常outputを返す。HTTP **202**／Audit completed | 後続readは`pending` | 202・正常outputから**適用済み**を推定できない |
| **HDG-E03** | 更新して正常outputを返す。HTTP **202** | 後続readは`submitted` | 同じHTTP 202でも業務状態は異なる |
| **HDG-E04** | 更新前にerror。HTTP **503** | 後続readは`pending` | 同じ503でもE01と実際の作用が異なる |
| **HDG-E05** | 更新してerror。HTTP **503**、独立read自体が障害 | Observation結果なし、read identityと`synthetic` natureだけ保持 | read失敗を「未適用」に書換えない |

全ケースでHost認可を既存`security.authorize`経由で通し、実Driverの`BindingAdapterResolver`と`capabilityinvoke`で合成Adapterを実呼出しする。HTTP中の暗黙read=0、Adapter invocation=1、明示read=1、Command自動再実行=0を比較する。固定のtest-only`operationRef`は読取対象の対応付けを示すだけで、**製品Operation IDや冪等キー、Receiptを新たに定義しない**。同一メモリの合成readは外部系からの独立した永続証拠ではない。

**境界**：P0-Eが実CIで成功した場合も、合格するのはHTTP応答／監査と、別の読み取り経路が示す**合成状態**の区別のみ。実DBのCommitted、実Clientによる応答完了、非冪等Commandの安全な再試行、Hostによる長期効果照合、現在のGrant、Source/BOM鮮度、実Plugin Providerの検証済み実行は引き続きUNKNOWN。

### 19.2 Lifecycle Graph・共通Application API・Plugin-firstへの接続

他領域で成熟しているLifecycle Graphの考え方を、この実験の**型付き責務境界**として扱う。実行可能Graphを固定の逐次手順に閉じず、Guard、拒否、取消、Retry、補償、非同期応答を扱えるようにする一方、全実行可能経路で`Authorize→Effect`、必要なHost guard、実Effect後の結果不確定性を迂回しないことを不変条件とする。**構成時GraphとPlugin適用後Graph、生成物の制御フロー、実走行Traceは別証拠**であり、`onTerminal`という宣言名やAudit callback 1回だけから完全Graphを推論しない。

製品横断の**現行S2-reference**は、[main #3540](https://github.com/hat47x/tei-platform/pull/3540)で通常`tei inspect`と通常・生成Studio Hostから**同じ静的Reference Inspector Provider**を選択・実呼出しするところまで統合済み（exact-head Health/Wails/CLI CIはsuccess）。次のP0である**S2-product**は、正式Profileが選んだProvider Bと両Consumerの実呼出し先Bの同一性、Artifact/Entryの独立検証であり、**まだ未資格**。HTTP側に専用Plugin Registryを新設してそのS2を横取りしない。HTTPの次段は、既存Service Carrier／Plugin Runtimeの**選択・Artifact/Policy・実Invoke・現在Authority・同一Attemptの結果**が揃ってから同じClient Reaction/Evidence分類を利用する。CLI/StudioのProvider Bとこの合成Submit Adapterを同一資格として扱わない。

- **Graph／SHO**：E01とE04は`503/error`表面が同じだが内部Effectが違う。Graph/Table/Traceでは`HTTP branch/status`、`Adapter invoked`、`Effect disposition (unknown)`、`observation nature/result`を別列で表し、出典Occurrence・Node/Edge identityを保持する。全Required Node/Guard/Relationの取得は#3262/#1124/#2497で別途検証。
- **TEI性能観測／最適化**：`HTTP 503`の件数から「失敗した更新件数」や「再試行可能件数」を推定しない。技術的タイミング・Adapter呼出し・実業務確定・read後の状態を別指標として扱い、実統計・独立Before/After比較がない最適化は候補にとどめる。
- **KON Dataspace／SEI／Effect管理**：将来の保存・決定・再開・再送はOperation identity、現在Authority、Receipt、独立Observationと鮮度を別々に保持する。ここで追加する合成状態はKONへの保存証拠ではない。既存#1158/#1174/#1656がClient ReactionとResult／EffectのOwnerであり、HTTP専用の第二Operation LedgerやTransaction Managerは作らない。

### 19.3 受入順序

まず今回の`HDG-E01〜E05`について、**新規テスト自体のCI PASS**を独立に取得する。その後のP0一辺は、既存実Provider/Service Carrierの同一Attemptにおける選択・Artifact資格・実呼出し・Result受領と、外部Observationを**異なる資格票**として結合すること。完了した部分だけを昇格し、Client完全受領／実DB/外部Effectの意味受理は独立確認が得られるまでUNKNOWNとする。

## 20. 2026-11-09 横断統合：HTTP効果観測の実証とComposition接続判断を分離する

### 20.1 実行済み証拠と資格票を確定

前段[PR #3537](https://github.com/hat47x/tei-platform/pull/3537)はmainへ統合済み。exact HEAD `a3448847bac5fcefac304f3f32b08150a1e3f282`の[限定CI #42486408608](https://github.com/hat47x/tei-platform/actions/runs/42486408608)は**completed/success**で、Node 7/7、Go Host package全PASS、HDG-E01〜E05の各サブテストとGo formatting PASS。脆弱性確認 #42486408701もsuccess。同HEADの[全体Health #42486408590](https://github.com/hat47x/tei-platform/actions/runs/42486408590)は**completed/failure**。全体`gofmt/go vet/go test ./.../tei validate`は**すべてPASS**したが、別ステップの`tools/ci-workflows-test.py`が、並行追加の`tei-ai-tuning-reference-ci.yml`の`actions/checkout@v4`未ピン留めと`synthetic-controls`の`timeout-minutes`欠落を検出した。現行mainの当該WorkflowはSHAピン留め・10分timeoutへ修正済み（**異なるHEAD**）。以前のfailureをsuccessへ改変しない。

| 証拠面 | 実証した最小事実 | 残る資格 |
| --- | --- | --- |
| Canonical / Driver | 実`order_entry.submit_order→sales.order.submit→CustomGoBinding`を既存Driverで解決 | 現在の業務意味Owner受理、外部Provider適格性 |
| Runtime HTTP | 実Reference Host・Canonical認可・Driverの有限Guard/分岐と**合成**Adapter実Invoke／Audit | E1静的`onTerminal`→Runtime全Branch保証、実IdP／Cookie／CSRF |
| Effect | 合成Adapterの更新前後と別`ExecuteObservationRead`の値の違い（503/202で一意でない） | 実DB Commit/Receipt、外部Observation、現在Snapshot、Client受領 |
| Plugin/Composition | 既存`servicehost`／Service Carrierのtest-only selected／Invoke witnessは別テストに存在 | **製品**CLI/Studio/Runtimeの宣言→実行への接続、実Provider B／Artifact・Authority |
| Design Context / Graph | Required Source、Source Index、Pattern/Capability等の別検証実績がある | HTTPの全Required Graph Node/Edge/Branchを同Source/Versionで実Consumerへ投影 |

**Client Reactionの非推論規則**：HTTP `202`でも「業務成功・保存済み」と表示せず、`503`でも「未適用だから非冪等POSTを再送してよい」と判断しない。Result/Receipt/Observationの未取得・不明・staleを明示し、実行後不確定なら現在Grant／Operation identityに束縛した**既存**のCommand Lifecycle/Effect fence（#1174/#2999）へ戻す。この表は**設計上の停止線**であり、製品Clientに自動Retry抑止機構が実装済みとの主張ではない。

### 20.2 他チャットのComposition実行接続判断が先行Gate

[構成接続点の判断資料（PR #3549）](../workstreams/COMPOSITION_CONNECT_POINT_BRIEF_2026-11.md)の6つの実測記録によれば、現在の**製品**`cmd/tei`は`operations`入口を持つが`composition`を直接使わず、Reference`go-host`は別経路で`composition`を使う。`servicehost`の非試験呼出し元は無く、宣言されたEnvironment Bindingは検証・計画へ届いても、非試験`routing.Router`の組立てにはまだ届いていない。この**発見は参考実装中のHost→selected Service Carrier呼出し成功と矛盾しない**。test-onlyの局所結合と、製品入口の宣言→実実行配線は別である。

| 上流の未決選択 | HTTPでの意味 | 今回扱う/扱わない |
| --- | --- | --- |
| C-4: 製品が宣言→実行を接続せず外部Hostへ委ねる | 外部Host契約が、現在Authority/Plugin artifact/Effect receiptの実行資格を所有し得る | **有効な選択肢として保持**。HTTP分野が暗黙にC-4を棄却しない |
| C-1: `routing.Router`を製品CLI入口で組立てる | Deployment Commandの宣言を実実行へ接続する候補 | 上流Gate決定以前にHTTP独自CLI Routerを新設しない |
| C-2: `servicehost`をCLI/Hostへ接続する | ProviderのHost世代ピン留めとGateway配線の候補 | 既存Referenceの成功を製品実行の受理へ昇格しない |
| C-3: `DeploymentExecutor`のCapability IDをBinding由来にする | Reference固定ID呼出しの宣言との一致を強化し得る | C-1/C-2決定と固定Referenceの目的を検討するまで保留 |

優先判断は**対象Operation/Targetの実行Owner（C-4外部委譲を含む）→役割別C-1 Deployment/C-2 Service→必要ならC-3**。Ownerは[Composition Kernel移行Gate](../gate/COMPOSITION_KERNEL_MIGRATION_GATE_2026-10.md)と製品成立状況（`docs/status.md`）であり、HTTP設計では選択・採用を決定しない。C-1/C-2は別種Operationなら共存できるが、**同一Attemptの二重Owner**は許さない。Pluginの[静的S2-reference #3540](https://github.com/hat47x/tei-platform/pull/3540)成功を正式Provider Bの**S2-product**へ昇格せず、HTTP本番結合より先に選択B→実Invoke Bを独立確認する。

### 20.3 次の最小Exitを「追加参考テスト」ではなく実Consumerに置く

**次の成果物候補は新しいGraph engine/Resolverではない**。上流C-4の判断を得るか、外部Hostに委ねる場合の現在資格・実行証拠IFを定めた後、選んだ**一つの製品実入口**で、同一`Operation`について（1）宣言のSource/Version、（2）実選択・Artifact/Host Grant、（3）Provider invoked/received、（4）Effect Receipt/独立観測、（5）HTTP Client Reactionの各Evidenceを別々に記録する。各証拠は一部のみ得られても残りを`unknown`として保持する。SUI/SHOへGraph/Table/Traceを描く場合、`Source occurrence`と実Attemptを区別し、KON/SEIに保存された履歴・提案を現在の権限や実Effectへ昇格しない。

性能計測の接続では、`503件数`や`Audit completed件数`を「反映失敗／成功件数」に読み替えない。[AI性能チューニング設計](AI_PERFORMANCE_TUNING_OBSERVABILITY_2026-11.md)の**統計・ヒント・実測・独立比較**を区別する方針に従い、正しい分母は`eligible attempts / actually invoked / externally observed`等の観測定義を固定してから計測する。AIの推奨・予測だけでContext Cache、Plugin Provider、Retry Policyを適用しない。

## 21. 参考資料と採否
## 16. 参考資料と採否
## 17. 参考資料と採否
## 18. 参考資料と採否
## 19. 参考資料と採否
## 21. 2026-11-09 C-1 Bridge：配備Executorの呼出し後不確定とHTTP Responseを同じ停止線へ置く

他チャットから統合された[PR #3573](https://github.com/hat47x/tei-platform/pull/3573)はEnvironmentの`deployment-executor` Bindingを、既存`routing.RouteCommand→routing.Router.Dispatch→routing.ValidateResult`で**Host登録の記録用Executor**へ渡すC-1のReferenceである。これは前段§20の`C-1/C-2/C-4`判断を解決しないが、HTTP Command処理と共通の重要な失敗境界を持つことが分かった。

`tools/reference-binding-to-execution/bindingexecution.go`のv0では、`Router.Dispatch`失敗後、または`ValidateResult`の応答不一致後に`refusedCount++`とした。ところが`Router.Dispatch`の既存コードは**実`Executor.Execute`を呼んだ後に失敗する場合がある**。後から届くエラーを「Executorへ未到達」という意味のRefusedへ再分類すると、HTTP 503を未適用と誤認するHDG-E01と同じ情報喪失が起きる。

**成熟差分（Reference v1）**：新たな製品型ではなく、既存のHost登録Executorにread-onlyの呼出し観測ラッパーを被せて`Execute`入口だけを識別する。`ranCount`＝実呼出し入口、`refusedCount`＝入口以前の拒否、`unverifiedCount`＝入口以後のエラーまたは不正応答（`ranCount`の内数）として別々に報告する。`executorInvoked`は外部Effect/Grant/Receiptの証拠ではない。`Report Schema/RuleVersion/RuleFingerprint`はv1へ更新し、[旧v0実測](../evidence/BINDING_TO_EXECUTION_REFERENCE_2026-11-09/README.md)を上書きしない。

| C-1 Reference反例 | HTTP E系との意味対応 | 不変条件 |
| --- | --- | --- |
| `Supports=false`、未呼出し | Session/CSRF/Authorization拒否でCommand Adapter未到達 | Refused。未実行という**限定**観測 |
| Executeで合成更新後にエラー／不正ID ACK | HTTP 503後に合成状態が`submitted`にもなり得る | **実呼出し済み・Result未検証**。Effect適用有無は別Observation |
| Executeで更新前にエラー | HTTP 503後に状態が`pending`にもなり得る | 同じエラー形から自動再送を許可しない |
| Accepted ACK、合成状態は更新なし | HTTP 202後にも`pending`があり得る | `OutcomeAccepted`≠Commit/永続保存 |

[テストBXE-01〜06](../../tools/reference-binding-to-execution/bindingexecution_effect_unknown_test.go)は本物の`Router.Dispatch`経由、合成Executorの呼出し回数とメモリ状態を独立Expectedで照合する。**[限定Go CI #42505232348](https://github.com/hat47x/tei-platform/actions/runs/42505232348)がcompleted/success**で、実`Router.Dispatch`によるBXE-01〜06・元の別Executor ACK回帰・Go vet/gofmtはPASS。ここでqualifiedなのはこのReference v1の有限Caseだけであり、製品のC-1採用、外部Provider/実Effect、完全なGraph/Client Reactionではない。先行する[HDG-E01〜05の専用CI成功 #42486408608](https://github.com/hat47x/tei-platform/actions/runs/42486408608)は別HEAD/別実装の証拠であり、今回のC-1変更のテスト合格に流用しない。

**次のExit**はHTTP専用の第二Result EngineやDeployment Executorへ自動Retryを組み込むことではない。#1174/#2999の既存Operation identity・Effect fence、#951/#3279/#3267のProvider/Host資格と、#3262のDesign Context出典を**製品で選択された一つのOwner/Consumer**へ接続する。C-4外部Host委譲も有効な選択肢。SUI/SHO/StudioとCLIが同じ種別のResultを表示するときも、`pre-execute refused`と`post-execute unverified`を同じ業務拒否や無条件再試行可能エラーに平坦化しない。実外部Effect/Client受領/Owner受理は引き続き未実証。

## 22. 2026-11-09：C-1 Deployment／C-2 Serviceの別経路をHTTP Client Reactionへ接続

最新の[Composition判断資料§10〜11](../workstreams/COMPOSITION_CONNECT_POINT_BRIEF_2026-11.md)と[独立したCarrierの実測](../evidence/CONNECT_POINT_C1_C2_2026-11-09/README.md)から、以前の「C-1 RouterとC-2 Service Carrierは必ず二者択一」という前提を訂正する。C-1は**配備Command**、C-2は**Service Request**であり、同じ業務Operationの二重実行Ownerにしない限り共存できる。C-4外部Host委譲はOperation／Targetごとに判断し、C-3はDeployment側のCapability解決であってHTTP Serviceへ無条件に適用しない。

| 実入口 | 呼出しを実際に観測した時点 | Refused／Unverifiedの境界 | HTTP/Graph表示で追加すべき列 |
| --- | --- | --- | --- |
| C-1`routing.Router.Dispatch` | `Executor.Execute`へ入場 | `Execute`前停止／入場後障害または不正ACK | Deployment Action／Executor ID／Attempt／ACKの有効性／外部Effect Unknown |
| C-2`SelectedServiceCarrierInvoker.Invoke` | Host-pinned Service Carrierへ入場 | Carrier入場前停止／入場後障害または不正世代結果 | Service Contract／Provider Selection／Host Generation／Carrier入場／Process実Invokeの有無／外部Effect Unknown |

**C-2で特に注意する点**：Carrierへの入場はProvider Processへの到達ではない。Host-owned RouteやArtifact Verifierが**Carrierの内部**で拒否する経路があるため、`carrierInvoked=true`だけで`providerInvoked=true`と推論しない。Carrier参照v0ではReportの版名と一部説明が食い違っていたため、`tei.reference.service-carrier-connect/v1`へ切り分け、実Resultの事前拒否と呼出し後不確定を別のError Codeで扱う。保存済みv0 RAW/SHAと新v1を混ぜず、新規CIの結果を受けて別途Qualifiedとする。

**次のプロダクト受入**：選択したOperation Ownerがどちらかを先に確認し、`selected→actual carrier/process invocation→returned ACK→effect independent read→Client Reaction`を同一Attemptの**別証拠**として要求する。既存Result/Command Effect fence（#1174）、Plugin/Current GrantとBOM（#951/#3279/#3267）、Design Context（#3262）を利用し、第二Resolver/Authority/Receipt StoreをHTTP専用に作らない。CLI/Studioの静的Reference Providerについて**通常入口への接続がある**ことと、正式Provider BでのProduct S2／Artifact検証が未成立であることを区別する。

## 23. 2026-11-09 横断取込み：C-3解決・Hook結末・実PostgreSQLをHTTP受入へ束縛

### 23.1 直近の実行済み証拠を、同一HTTP Requestの実証と取り違えない

前段§19〜22では、実Go Host→Canonical Authorization→Driver→合成Adapterを一つのHTTP Requestで実行し、HTTP 202/503と合成状態の独立Readが一致しない反例を確認した。またC-1 Deployment Executor、C-2 Service Carrierの有限Referenceが個別に成立し、どちらの入場・応答も外部Effect Receiptとは異なることを固定した。その後、**異なるレーンの実装とCI**が進んだため、現在のOwnerと入場条件を整理する。

| 新たに確認した成果（main統合） | 実証している最小事実 | HTTP／Productについて未実証 |
| --- | --- | --- |
| [C-3 PR #3692](https://github.com/hat47x/tei-platform/pull/3692)、[判断資料更新 #3720](https://github.com/hat47x/tei-platform/pull/3720) | `routing.RouteCommand`→`pruntime.BuildPlan`→`Capability.SupportsAction`で宣言Binding由来のExecutor IDを**読取解決**。異なるActionで直書き`migration.atlas`と実選択`runtime.reference`が食い違う反例。Lockなしは`LOCK_REJECTED` | `Executor.Execute`もCarrierも**呼ばない**。C-3 `v0`は「未実装／C-1・C-2 v1より遅れ」ではなく、**実行前限定の別目的**。採用するDeployment製品経路も未決 |
| [Hook結末名 PR #3721](https://github.com/hat47x/tei-platform/pull/3721) | `internal/hook.NotSatisfiedOutcomes`、`CommandOutcomeFor`、`OutcomeOf`を導入し、**参照Hookの** `not-found`／`value-rejected`等を既存Command拒否名へ対応付ける8件のGo試験。 `not-satisfied`はCommand拒否名へ写さず`false`、技術的Hook Errorから業務結末を生成しない | **後続main #3761で`hook.Runner`境界の語彙検査は実装済み**。ただし外部Plugin Transportを通る入力検査とCLI/HTTPへの実Consumer接続は未実証。`OutcomeOf`は`NotSatisfied.Outcome`を返すが、この関数だけでは外部文字列の語彙適格性を証明しない。CLI JSON→Command→HTTPの一貫したResult変換・Client受領も未確認 |
| [実PostgreSQL CI PR #3717](https://github.com/hat47x/tei-platform/pull/3717)、[専用CI #42570360880](https://github.com/hat47x/tei-platform/actions/runs/42570360880) | **completed/success**。`postgres:16`でDSNを明示し、`internal/command/lifecyclepg`、`internal/driver`、`tools/reference-expected-state-update`、`reference/web-api-contract`の必要試験を実行。行ロック、Commit後ACK喪失、冪等再適用拒否、CAS、tenant範囲等の個別テストがPASS。SKIPで緑にならないチェックあり | 4テスト系列は**別DB・別Action・別Request／Operation**。`order_entry.submit_order`の実Go Host→実Command→DB Commit→別Connection Receipt→実Client Reactionの**同一Attempt**を実証しない。実Provider選択／Grant／Owner意味受理も別 |
| [CLI/MCP継続トークン実測 PR #3723](https://github.com/hat47x/tei-platform/pull/3723) | 参照Design Contextの継続トークンはSource byte・scope・direction変更で`STALE_SNAPSHOT`、同じ選択子ではCLI/MCPとも継続できる。非意味的なコメント変更でも旧Tokenは失効 | CLI/MCPのSnapshot一致を、HTTP Responseを生成した実Runtime Source revisionや現在Grantと同じものとして自動受理しない |

これらの成果は**別々の資格票として有効**であり、同一リクエストの縦断実証として足し合わせない。C-3の`RESOLVED`は実行権限でも外部Effectでもなく、Hookの`not-satisfied`はCommand拒否でもHTTP 4xxでもない。Hook業務拒否、Command検証、HTTP決定、外部DBへの副作用、Clientへの完全送達は別Ownerの段階である。

### 23.2 Resultを勝手に昇格しないための受入行列（提案・NOT RUN）

| ID | 対照する実経路／反例 | 必須の独立Expectedと停止線 | Owner |
| --- | --- | --- | --- |
| HDG-X01 | C-3実Bindingで`runtime.reference`選択、比較用固定IDは`migration.atlas` | 選択IDをResolver/Plan/Supportsで照合し、違うExecutorへ`Execute`しない。Read-onlyの`RESOLVED`から`invoked`を創作しない | #1016／#951 |
| HDG-X02 | Hook `NotSatisfied{Outcome:"not-satisfied"}` | Hook判定は記録するが、`CommandOutcomeFor`は`false`。業務失敗種別・HTTP Status・Effectの有無を推測しない | X-TEA Hook Owner／#1158 |
| HDG-X03 | Hook `NotSatisfied{Outcome:"value-rejected"}`と技術Errorの対照 | 前者はCommand拒否名への候補対応があるが、**実HTTP応答と連動するか**は実Consumerで別検査。後者を`value-rejected`に変換しない。`hook.Runner`での未知語彙拒否（#3761）は有効だが、実Plugin Transportからの未知語彙やHTTPへの投影は**別の実Consumer試験**で確認する | Hook／#1043／#1158 |
| HDG-X04 | 実DBでCommitしたがHTTP Ack/Client受領を喪失 | 同一Operation ID・Idempotency Key・Source revision・Grant epoch・実DB状態・Receipt/別Connection Observation・実Client Reactionを別記録。HTTP 503単独でno-effectや安全Retryにしない | #1174／#2999／#1158 |
| HDG-X05 | 旧Design Context継続TokenでSource/Scopeが変化 | Read-only Contextで`STALE_SNAPSHOT`を返し、旧SnapshotのGraph・Provider/Operation判断を現在のAuthorityに流用しない。新Snapshotの選択・実呼出しは独立資格 | #3262／#3420／#951 |
| HDG-X06 | C-1／C-2／C-4で同一業務Operationを二重実行 | 異なるDeployment/Service仕事の併存は許容、**同一Operation／Attemptの実Ownerは一つ**に限定。委譲先は外部Receiptの取得までは`delegated/unknown`。無条件再実行しない | #1016／#1174 |

受入開始時は、**一つの現実のOperation／Target**を先に固定し、必要Sourceの`required/discovered/loaded/bound/returned/disclosable`とHost世代・本番Provider/現在Grant・外部Readを揃える。その上で、既存Command/Hook/HTTPと別DB Observationの各実Callbackが**同じAttempt**を参照するかを検証する。実DBスイートの成功や、C-3／Hook単体Go試験の成功を材料にしても、欠けた段階は`unknown`のまま残す。

### 23.3 製品横断の優先順位と撤回条件

1. **既存S2-referenceを再実装しない**。[#3540](https://github.com/hat47x/tei-platform/pull/3540)の通常CLI/Studioの静的Inspectorは成立済み。先行するS2-productは正式Profileで`selected Provider B`と`actual Provider B`の同一性、B選択→A実行の負例、実Build Artifact/現在Grantを別々に確認する（#1251/#951）。
2. **HTTPの新Engineより実Consumerを優先**。C-1 Deployment／C-2 Serviceは別Operationなら両立でき、C-3は実行前解決、C-4外部Host委譲は有効な未決選択肢。Product Ownerの選択を下流HTTP設計が先取りせず、同じHook/Command/Effectの資格を第二Host／Registryへ複製しない。
3. **PGの資格を正しい場所で利用**。[#3717のCI](https://github.com/hat47x/tei-platform/actions/runs/42570360880)は実DBテストがskipでないことを証明したため、次の縦断テストではそのテスト基盤とOperation Effect Fenceを再利用する。ただし実HTTP Client/Provider受領まで含まない場合、E3の完了扱いにはしない。

本節の行列は**実装前の独立Expected／受入計画**であり、HDG-X01〜06をGo/DB/HTTP E2Eで実行したことを示さない。現在のHook結果の語彙対応をそのままHTTPコード表やPluginの権限検査へ昇格させることはしない。

## 24. 2026-11-09 第2次横断反映：Hook実境界の語彙制約・6種Binding Consumer・Build Plan出典

### 24.1 別PRの限定合格を更新する

[PR #3736](https://github.com/hat47x/tei-platform/pull/3736)の[全体Health #42576080511](https://github.com/hat47x/tei-platform/actions/runs/42576080511)は**completed/success**。ただし同PRが定義した`HDG-X01〜X06`は独立Expectedの**計画でありNOT RUN**という状態を変えない。

[Hook境界 PR #3761](https://github.com/hat47x/tei-platform/pull/3761)（[全体CI #42606902787](https://github.com/hat47x/tei-platform/actions/runs/42606902787)、completed/success）は、`hook.ValidateOutcome`を実装し、`Runner`の`wrapFailure`で**実装が報告した結末名が`NotSatisfiedOutcomes`内にあるか**を確認する。`valueRejected`など未知名は`HOOK_OUTCOME_UNDECLARED`で拒否し、`value-rejected`のような既知名と、名前のない理由だけの不成立は許容。既存`CommandOutcomeFor`との対応関係を変更せず、`not-satisfied`をCommand拒否へ創作しない。**実外部PluginのProcess/IPC/API経由とHTTP/Web/Studio Clientの伝送・表示は未実証**。

[6宣言のConsumer調査 PR #3757](https://github.com/hat47x/tei-platform/pull/3757)（[全体CI #42606374129](https://github.com/hat47x/tei-platform/actions/runs/42606374129)、completed/success）は`Environment`の6種について、解析／検証と**宣言由来の実解決**と**実ProviderのInvoke**を分離した。[観測の元データ](../evidence/BINDING_FORMS_CONSUMER_SURVEY_2026-11-09/README.md)は`main 0837a6f4`の一時点であり、現在mainの呼出し件数の証明ではない。通常`cmd/tei`が`observation.Binding`／`evidence.Binding`を`-plan`のため呼ぶ経路は既存だが、同調査では`observation.Router{}`／`evidence.Router{}`の非試験組立ては0件。`resolved`≠`invoked`≠`received`を**観測・証拠の両Provider**にも適用する（[Composition判断資料§12](../workstreams/COMPOSITION_CONNECT_POINT_BRIEF_2026-11.md)）。

[Self-build PR #3760](https://github.com/hat47x/tei-platform/pull/3760)（[全体CI #42605259664](https://github.com/hat47x/tei-platform/actions/runs/42605259664)、completed/success）は、呼び手指定の`-build-catalog`・digest・Source Rootから`self-build`のentryがin-processでPlanを導出し、別child Referenceと`planFingerprint`一致を確認した。**`ok=true`でも`qualified=false`／`l2bClaimed=false`**であり、catalog／digestの同一性や`planArtifactBinding=bound`からRuntime ArtifactのTrust、現在Grant、業務Effectを推論しない。HTTPのPlugin/Source取得へは「読んだ宣言／生成した計画／実ConsumeしたArtifact」の分離を再利用するだけで、この場ではSelf-build Gateを変更しない。

### 24.2 更新した独立受入Expected（本PRでは新E2EはNOT RUN）

| ID | Sourceから実Consumerまで必要な観測 | 誤った昇格を拒否する条件 |
| --- | --- | --- |
| **HDG-X02a** | 実`Runner.Run/RunReachable`へ既知`value-rejected`、未知`valueRejected`、空Outcome、`not-satisfied`を供給し、Hook Resultと`CommandOutcomeFor`の照合を別記録 | 既知名でも`HTTP 4xx`は自動決定しない。未知名は`HOOK_OUTCOME_UNDECLARED`、空／`not-satisfied`をCommand拒否へ創作しない |
| **HDG-X03a** | 実外部Pluginが報告したResultを`hook.Runner`境界へ渡し、受信したOutcomeのbyte／拒否理由／Sourceと実Client表示を同一Attemptで別票化 | 内部Goテストの語彙検査成功を、Plugin輸送・HTTP Status/Problem・Client送達に流用しない |
| **HDG-X05a** | `ObservationProviders`／`EvidenceProviders`の宣言・Binding・選択Providerと`Observe`／`Collect`実呼出しの有無・Partial／Snapshot／Open Scopeを分けて記録 | 通常CLIの`-plan`でBindingが出るだけなら`invoked=not-observed`。Router不在を自動Provider実行で補完しない |
| **HDG-X05b** | 実行元のSource/World revision、Build Plan fingerprint、実Artifact/PluginのVersion/Grant epoch、受信Observer/Clientの時点を別Evidenceとして対応付け | digest一致／Plan`bound`だけでRuntime Trust、Accepted Meaning、現在のEffect/Client Receiptへ昇格しない |

この行列は前段`HDG-X01〜X06`の**追加負例案**であり、本PRによってGo、Provider、実HTTP、実DB、実GUI/Clientを通した検証が完了したものではない。Function/Graph/SHO・KONへの表示は`Source declared`、`resolved`、`selected`、`actual invoked`、`received`、`observed effect`、`disclosable`を**同一列に畳まない**。Source location・Relation/Edge occurrence・Current Authorityが欠ければ欠けた列を`unknown`として保持する。#3419/#3421の共通APIと#1174のEffect FenceをOwnerにし、第二Hook Engine、Provider Registry、Tx HostをHTTP側に作らない。

### 24.3 最短の次アクション

**優先1**は#1251のS2-product（正式Profileの`selected B`と通常CLI＋Studio実`invoked B`の照合）で、静的Referenceを再実装しない。**優先2**はC-4を含む実行Owner判断に応じた既存Observer/Evidence Consumerの一つの実Invoke Gateで、Bindingがすでに製品`-plan`に出るところを重複実装しない。**優先3**は単一OperationのHook→Command→HTTP→Postgres Commit/Receipt→Client Reactionを共通APIで連結するE3/E2E。各優先の独立Expected・Source／Artifact／Client受領を記録し、未確認なら未受理として保留する。

## 25. 参考資料と採否

- [Liberator repository](https://github.com/clojure-liberator/liberator) ― ClojureのREST decision modelの先行例。ライブラリ移植ではない。
- [Webmachine mechanics](https://webmachine.github.io/mechanics.html)、[Decision flow](https://webmachine.github.io/diagram.html)、[Resource functions](https://webmachine.github.io/resources.html) ― decision flow・契約・処理に分離する参考資料（古いHTTP版があるため仕様の正本にはしない）。
- [RFC 9110: HTTP Semantics](https://www.rfc-editor.org/rfc/rfc9110.html) ― HTTPの安全性/冪等性/条件付き要求/自動retry制約。
- [RFC 9457: Problem Details for HTTP APIs](https://www.rfc-editor.org/rfc/rfc9457.html) ― HTTPエラー表現。response失敗時のアプリ状態とClient Reactionまでは規定しない。
- [OpenAPI specification](https://spec.openapis.org/oas/latest.html) ― HTTP interfaceの記述。TEIのClient Reaction・effect dispositionの正本として単独利用しない。

**採否の分岐**：既存標準Graphで必要なidentity/branch/guardを保持できれば追加Core Typeは不要。保持できなければ喪失箇所と小さな改善案を#2620/#2497へ返す。Graphと表・実Traceの結合が実Consumerで有用と確認できなければ専用のHTTP Graph UIを採用しない。P-1〜P3が揃うまで「実装済み」「受理済み」とは記載しない。
