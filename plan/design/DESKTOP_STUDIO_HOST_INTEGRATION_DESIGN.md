# Desktop Studio Hostの操作・状態・資源境界（設計候補）

- **状態**: 設計候補。Wailsの実機起動、Desktopの操作E2E、OS別の安全性・配布検証は未実施。
- **整理日**: 2026-10-13
- **対象**: 開発者向けDesktop Studioの外側のHost。CLI/Webと共有するApplication Operationの利用者。
- **既存の担当**: [#1158](https://github.com/hat47x/tei-platform/issues/1158)（型付き操作契約）、[#1124](https://github.com/hat47x/tei-platform/issues/1124)（三視点の編集・投影）、[#1178](https://github.com/hat47x/tei-platform/issues/1178)（情報開示）、[#259](https://github.com/hat47x/tei-platform/issues/259)（第二Application Target）、[#1180](https://github.com/hat47x/tei-platform/issues/1180)（Driver Scenario）。
- **前提文書**: [CLI主入口](CLI_FIRST_INTERFACE_DESIGN.md)、[経路間の業務機能完全性](CROSS_ROUTE_FUNCTIONAL_PARITY_AND_TEST_DERIVATION.md)、[全体構成](../../docs/architecture.md)、[実装状況](../../docs/status.md)。

## 1. 対象と非対象

Desktop Studioは、Sourceの参照・編集提案・差分確認・受理・投影に使う**開発者向けのAuthoring Client**である。Canonical SourceやApplication Driverの第二実装ではない。Web/CLIと同じ意味操作、Validation、Shared Acceptance、Authorization、Plugin/Capability解決を再利用する。

**生成業務アプリケーションのDesktop Targetとは別物**である。

```text
                         Canonical Source / Shared Acceptance
                                           ^
                                  Application Services
                                           ^
                   +-----------------------+----------------------+
                   |                       |                      |
                CLI adapter             Web adapter       Desktop Studio adapter
                   |                       |                      |
               argv/stdin             HTTP/Auth           Wails/WebView/Host
```

- Studio HostのWebView、IPC、ファイル選択、ウィンドウ状態をCanonicalに持ち込まない。
- StudioがWailsで動いたことを、生成業務Desktop Targetの実証やPAS-04の第二Target合格とは数えない。
- Generated Desktop Targetが必要になったら、[#259](https://github.com/hat47x/tei-platform/issues/259)でTarget-neutral Presentation IRから独立したTarget Packとして検証する。
- Desktop導入を口実に第二Plugin Resolver、第二Policy Engine、CLI子プロセスによる業務APIの常用経路を作らない。
- 初期実装の置き場やGo package名、永続schemaは[#1158](https://github.com/hat47x/tei-platform/issues/1158)の実consumerを確認してから決める。

### 現行の共通Operationから見える制約

- `internal/operations/validate.go` の `ValidateWorkspace` は現に存在するread-only入口であり、Source変更や受理権限は提供しない。
- `internal/operations/inspect.go` は正規化されたWorldとSource Indexの投影を返すが、`WorldFingerprint` / `SourceIndexFingerprint` は観測値であり、書込みrevisionや受理許可ではない。`internal/operations/snapshot.go` にもその区別が明記されている。
- したがって、既存のfingerprint値をそのままDesktopの楽観ロックに転用しない。書込み直前の実Source byte照合と、既存Acceptanceのrevision/authority条件を橋渡しする最小Operationが必要であり、この接続は**未実装・未検証**である。

## 2. 呼出し経路と型付きOperation

```text
Desktop UI
  -> Wails service binding / Host IPC
  -> Desktop Studio Adapter（Hostの入力検証・変換）
  -> framework-neutral Application Operation
  -> existing Studio/Source/Acceptance/Driver/Authorization
  -> typed Result/Diagnostic/Failure + provenance
  -> UI projection
```

### 2.1 最初の操作集合（候補）

| 操作意図 | 作用 | 所有者 | 初期評価 |
| --- | --- | --- | --- |
| Workspaceの選択・読込み | local read | Host + 既存Workspace操作 | 実Path・ファイル内容を照合 |
| Validate / Inspect / Trace | read | 共通Operation | 出所情報は閲覧と外部送信を分けて認可 |
| Proposal Preview | proposalのみ | Studio Service | Accepted Sourceを変更しない |
| Accept / Reject | protected mutation / decision | Shared Acceptance | Revisionと実Sourceを再検証 |
| Driver Scenario確認 | read / isolated execute | 既存Driver | 本物の業務副作用には実行権限と分離されたEvidence |
| Progress / Cancel | Host control | Host + operation lifecycle | eventを新しいmutation経路にしない |

**下記は試験用の概念的なメッセージ形状であり、採用済みAPI・schemaではない。**

```json
{
  "operation": "studio.proposal.preview",
  "workspaceRef": "workspace-handle-01",
  "baseModelRevision": 214,
  "sourceSnapshot": "sha256:...",
  "candidateRef": "draft-12"
}
```

- `workspaceRef`はHostで解決した限定的な参照。入力文字列やファイルパスそのものを権限とみなさない。
- `baseModelRevision`はAccepted Modelの競合検知。`sourceSnapshot`は外部ファイル編集の検知に必要な**別の情報**。前者だけで後者を代用しない。
- `candidateRef`は未受理の候補の識別子。受理前の候補をAccepted Sourceとして参照しない。
- Principal、認証済みCaller、tenant/scope、Policy revision、Resource Leaseは、検証済みExecution Contextから渡す。WebViewやAIが送った自己申告は権限にならない。
- 操作結果はBusiness Outcome、Execution Error、Diagnostic、独立Observation、Host transport failureを区別する。UIの「成功表示」やIPC正常応答は業務成功の証拠ではない。
- 進捗通知・ファイル監視イベントはread-onlyのHost通知。Acceptや外部副作用をイベント購読だけで発火させない。
- 副作用の完了が不明な場合は`UNKNOWN`として観測を求め、UI再接続やリトライで無条件に再実行しない。

### 2.2 最初のAuthoring一巡

1. Workspaceを開き、Accepted Sourceの`modelRevision`と対象Sourceのsnapshot/fingerprintを取得する。
2. Data / Function / BusinessのProjectionから変更候補を生成し、PreviewでSemantic Diff、Validation、Persistence・UI影響を返す。
3. PreviewはAccepted Sourceを変更しない。Rejectも変更しない。
4. Accept直前にAuthority、Revision、参照先、Sourceの実バイト、必要なResourceを再確認する。stale/unknown/deniedならfail closed。
5. Shared Acceptanceへ引き渡し、再読込みしたAccepted SourceからProjectionを作り直す。UI内の変更済みオブジェクトを正本として再利用しない。
6. 手編集・AI提案・CLI・Webからの同内容の変更は、Host表現を除いた意味と失敗分類を比較する。

## 3. Desktop固有の状態分類

| 状態 | 保存先・権限 | Sourceへの反映 |
| --- | --- | --- |
| Accepted Source / modelRevision | Workspaceの既存Authority | Shared Acceptance経由のみ |
| 編集中のDraft | Host管理の限定的な作業領域 | 明示受理まで反映しない |
| 未受理CandidateとPreview | Candidate Store / 派生Cache | Preview/Rejectでは変更しない |
| Data/Function/Business Projection | Source revisionに紐付く再生成可能なCache | 正本にはしない |
| 選択・タブ・パネル・ウィンドウ位置 | DesktopのUI設定 | 業務意味にしない |
| 進行中Operation / 外部副作用 | 既存Run/Execution Authorityと必要なHost receipt | UI状態だけで完了を確定しない |
| OS秘密情報 / Resource Lease | OSのSecret Adapter / Resource Broker | Draft/Log/Canonicalへ埋め込まない |

### 3.1 ファイル変更と競合

- Hostのファイル監視は**変更の可能性の通知**に限る。通知回数・mtime・Watcherの順序をAccepted revisionと同一視しない。
- 保存またはAccept前に対象Sourceの実バイト指紋を取り直す。外部エディター・Git操作・別TEIプロセスによる変更を検知したら、古いPreviewから自動確定しない。
- Snapshotが一致しなければ差分確認・再Preview・再受理を要求する。Watcherの通知漏れでも保存時の検査で捕捉する。
- 出力先がSymlink等でWorkspace外へ逸脱しないことを、開く時点だけでなく、実際の読取り／書込み直前のHost境界で確認する。Path文字列の正規化だけで競合・差替えに安全とは宣言しない。
- 原子的なSource更新や複数ファイル更新の成否・ロールバックは既存Acceptance/Source所有者の契約に従い、Hostで擬似的な第二トランザクションを作らない。
- 同じWorkspaceを複数ウィンドウ・プロセスから扱う場合、UIタブごとの排他ではなく、Sourceのrevision/snapshot条件で衝突を判定する。

### 3.2 クラッシュ・復旧

1. WebViewまたはDesktop Host再起動後、最初にAccepted Sourceを再読込みする。
2. 保存済みDraftは、出所・保存時の基準revision・内容の機密区分を確認したうえで復元候補として表示する。**自動Acceptしない**。
3. Draftの安全な保存が未実装なら、その状態を利用者へ示し、機密Draftを安易に平文ファイルへ保存しない。
4. Hostが不明な外部Commandを再実行してはならない。Durable Run/Observationで状態を確認し、未知なら利用者判断へ戻す。
5. バージョン変更後の旧Draftはschema互換性と出所を検査し、変換不能なら読み取り専用で保全する。

## 4. Host Resourceと安全性

Desktopでローカル実行できることは、外部DB、ネットワーク、Secret、プロセス実行、Pluginの権限を意味しない。

- **File**: 明示的に選んだWorkspace・ファイル範囲だけを扱う。ディレクトリの再解決・Symlink・競合を負例試験に含める。
- **Process**: 任意shell実行をFrontendへ公開せず、既存のCapability/Providerと許可済み実行の範囲に限定。Hostの環境変数・Secretを暗黙継承させない。
- **Network / Secret**: Endpoint・作用・Lease期間を制限し、Secretの生値はWebView、Canonical、持続Logへ渡さない。
- **Authorization**: read、mutation、外部egress、外部effectを分け、各操作直前に必要なAuthorityを確認する。
- **UI origin / IPC**: 開発用画面の差替え、外部URLのナビゲーション、Deep Link、任意のJS入力を権限昇格経路にしない。公開Serviceと渡すDTOを最小化する。
- **Observability**: Path・Source Index・対象名も情報開示の対象。Logとクラッシュレポートの送出には[#1178](https://github.com/hat47x/tei-platform/issues/1178)の開示境界を適用する。
- **Distribution**: OS署名・公証・Artifact署名/検証・更新後の再起動/失敗復旧はHost固有の試験。署名済みという事実を業務Authorizationの代替にしない。

既存のResource Broker / Lease、Plugin Lock、Artifact検証を使い、Desktop固有の汎用Policy EngineやPlugin実行器を複製しない。Hostで制約を実効的に強制できないCapabilityは`unsupported`/`unresolved`にする。

## 5. ローカル・接続・ハイブリッド

| モード | 許される操作 | 明示すべき条件 |
| --- | --- | --- |
| Local | ローカルSource参照・検証・ローカル権限内の提案 | 認可されていない外部副作用を伴わない |
| Connected | 認証済みRemote Providerの利用 | 期限・Policy revision・ネットワーク断・再認証 |
| Hybrid | ローカルDraft/Sourceと共有先の明示的な統合 | Git/Source競合と業務DB競合を混同しない |

オフラインだからといって、期限切れのRemote Authorizationをローカルの権限で置き換えない。同期不能なら未解決として扱い、過去の成功を再生して共有先の状態を作らない。

Localモードでは認証を要求しないのを原則とする（[ADR-0044](../adr/0044-authentication-not-a-core-concern.md)）。ローカルWorkspaceへの参照・検証・提案・受理にログインやcredentialを要求せず、認証が必要なのはConnected / Hybridで接続先の提供主体（Plugin、既存IdP、Remote Provider）が契約として解決する範囲である。認証機構をStudio HostやCLIの必須依存にしない。

## 6. Wails v3の採否判定（現時点では候補）

第一候補をGo + Wails v3とするが、HostでなくGoの共通Application Serviceが意味操作を所有する。Wails v3の依存は外側のHostへ閉じ込め、Browser共通UIの変更と区別する。

2026-10-13時点の公式資料では、Wails v3はbeta、Go 1.25以上が必要。OS別のWebView Runtime/Toolchainを要求する。リリース番号の固定、Framework APIの互換性、Windows WebView2、macOS WKWebView、Linux WebKitGTKを実機で確認するまでProduction採用は確定しない。

検証時に固定するもの: Host OS/Architecture、Go/Wails version、WebView Runtime、frontend build identity、Application/Source revision、Plugin/Resource構成、再現手順。

Wails依存を外したGo Service/合成Adapterの試験と、実WebView起動・入力・Host permission・OS別配布のE2Eは**別のEvidence**として記録する。

参照: [Wails v3 Installation](https://v3.wails.io/quick-start/installation/)、[Wails v3 FAQ](https://v3.wails.io/faq/)、[Wails Releases](https://github.com/wailsapp/wails/releases)。

## 7. 最小Referenceの受入条件と負例

| ID | 実証条件 | 不合格とする状態 |
| --- | --- | --- |
| DS-01 | 一つのAccepted WorkspaceをCLIとIn-process Studio Adapterが同じ意味で読める | 表示だけを同等と判断 |
| DS-02 | Relation変更のPreview→RejectではSource byte/revision不変 | UI非更新のみで成功 |
| DS-03 | 同じProposalのPreview→Acceptが共有Acceptanceを通り、再読込み後にRevisionが更新 | WebViewが直接Sourceを書換え |
| DS-04 | 別プロセス編集・stale revision・fingerprint不一致を拒否 | Watcher通知がなかったので受理 |
| DS-05 | 権限不足・失効・別Workspaceのmutating actionを拒否 | Desktopはローカルなので許可 |
| DS-06 | WebView/Host再起動でDraftとAcceptedを取り違えない | Draftの自動受理・副作用再実行 |
| DS-07 | Hostのfile/process/secret/network denialと不確定結果を識別できる | IPC成功・空の結果を業務成功 |
| DS-08 | Core・Application ServiceにWails/DOM/Hostのimportがなく、Adapterから受理内部へ直接到達しない | Host固有の第二Mutation経路 |
| DS-09 | 実Wails起動・Native file dialog・Frontend binding・crash recoveryを少なくともWindows環境で観測 | 合成AdapterのPASSを実HostのPASSへ読み替え |
| DS-10 | 生成業務Desktop Targetの同等性は[#259](https://github.com/hat47x/tei-platform/issues/259)と[#1180](https://github.com/hat47x/tei-platform/issues/1180)で別Evidenceを取得 | Studio Host成功を第二Target合格とする |

### 推奨する実証順序

1. **[#1158](https://github.com/hat47x/tei-platform/issues/1158)**: 既存`internal/operations`を再利用し、`read/validate`・`preview`・`accept`の必要最小限を型付きOperationへ切り出す。既存のデータ型と一致しないDTOを先に正本化しない。
2. **[#1124](https://github.com/hat47x/tei-platform/issues/1124)**: 同じServiceで一つのRelation変更をPreview→Acceptし、stale/invalid/unauthorizedを確認する。
3. **Desktop固有の小さなReference**: 既存Operationの合成Host consumerでDS-01〜DS-08のうちHostに関係する負例を先に実証する。
4. **Wails v3実機ゲート**: DS-09を実WebViewとOS Resourceで確認し、実行結果からWails採否を決める。
5. **別Target**: DS-10はStudioと別ブランチ・別成果物・別Evidenceにする。

この文書は要求の新しい正本や実装完了証明ではない。採否は実測・既存担当Issue・設計判断を経て更新する。

<!-- tei-wails-ui-extensions-20261103 -->
## 8. 2026-11-03 追補：Wailsの上に構築するUI/Go拡張とStudio/生成Targetの切分け（Draft／NOT RUN）

**外部前提の更新**：本書 §6 は2026-10-13時点のbeta情報を残す歴史的記述。Wails公式が2026-10-26に **v3.0.0正式公開**と告知し、正式版は**Go 1.26以上**を要求している。従って以後のPoCは beta/Go1.25 前提でなく、実Wails・Go Toolchain・Frontend runtime/生成Binding・OS別WebViewをexact pinして評価する。公開はTEIのProduction採用・Windows実機検証を意味しない。

**区分**：
- **Studio（Authoring Client）**：既存 #1158 Typed Operation と #1124 Shared Acceptance を呼ぶ。Wails serviceは外側のtransport/lifecycle adapterであり、Sourceの直接書換え/Business Authorityの正本にならない（既存DS-01〜09）。
- **生成業務Desktop Application**：Accepted ApplicationからTarget-neutral Presentation/Actionを導き、#997 のUI KitとTarget固有Wails Adapter/Go Serviceを**Build時に静的合成**する対象。#259/#1182 の独立Target Build/配布Evidenceを必要とする。Studioの成果物・DS-09成功を第二Target合格へ流用しない。
- **Web Browser Target**：共通React等のUI部品を再利用できても、Host側Go service・OS API・local processをBrowserへ露出させない。Browser assetとAPI unit、Origin/Auth/CORS/CSRF/secretの実境界は #1182へ委譲する。
- **外部プロセス**：独立更新/Native機能/障害隔離の実要件がある場合だけ選択。Wailsを増築する代わりに毎回プロセスを立てることを標準化しない。既存Service Contract/Provider/配置に従い、起動失敗・プロセス終了・ACK喪失/Commit unknownを別扱いにする。

**Go ServiceとUI Extensionの責務**：案件Go serviceは生成物に直接追記せず案件Sourceとして持ち、選択されたService/Operation契約・Binding生成・必要なOS CapabilityをBuild時に登録する。React/TypeScript UI Extensionは #997 のRoot Kitによる公開Slotに provide/replace/decorateし、Wailsの生成Bindingはtransport artifactとして扱う。標準Kitの内部DOMや未公開Go methodへ依存する拡張はUpstream追従保証外。業務Action、Validation、保存Intent、実Provider Commit、Host現在認可をUIやWails serviceへ重複実装しない。

**公開境界とセキュリティ**：同一WebViewに取り込まれた任意JavaScriptは、Kit Slotを使ったことだけでは安全なsandboxにならない。WebView bridgeへ登録するGo methodを絞り、外部URL/非信頼HTML/child frameからの到達、secret、任意file/process、Window lifecycleを実Target条件で検査する。許可・失効・read-onlyはRequest時に再照合。開発・試験用MCP/InstrumentationのProduction物理除去は #2391 の別Evidenceとする。

**既存DSケースへの追加判定（新Gateは設けない）**：
- **DS-08に加える**：案件Extensionを追加しても共通Operation/AcceptanceへのWails依存逆流、UI→Source/DB直接mutation、第二Resolverを導入しない。
- **DS-09に加える**：正式版Wails v3.0.0+・Go 1.26+・OS/WebView/Toolchain/Frontend assetの実物版を固定した上で、Service Binding生成・案件Go static composition・起動/終了・untrusted content・更新後の署名/再起動を個別に検証する。
- **DS-10に加える**：同一Accepted Meaning、Action/Result/Failure/Unknownに対してWeb実Browserと生成Desktop実WebViewの差をTarget別に観測し、Studioと混同しない。
- **#997 UI-5へ返す**：親Kit旧版→新版、案件Overlay旧版→候補の三者比較、Slot/lifecycle/CSS・exact fingerprint・Target互換・実動作の未実証を診断し、衝突時は稼働中の固定成果物を保持する。

受入Evidenceは **(a) 設計/Reference Resolver、(b) Go/TS buildと依存閉包、(c) 実OS WebView動作、(d) Host認可・Effect/Receipt、(e) 案件UI更新追従** を別列で記録する。今回の追補は文書のみ。Go/Wails実機、生成業務Desktop、独立E2E、正式Extension API/Kit ABI、Owner業務意味受理はすべて**NOT RUN／未採用**。

参照：[UI構成キット階層Reference](../../reference/ui-customization/composition/README.md)、[#997](https://github.com/hat47x/tei-platform/issues/997)、[#1824](https://github.com/hat47x/tei-platform/issues/1824)、[#259](https://github.com/hat47x/tei-platform/issues/259)、[#1158](https://github.com/hat47x/tei-platform/issues/1158)、[#951](https://github.com/hat47x/tei-platform/issues/951)、[#2391](https://github.com/hat47x/tei-platform/issues/2391)。Wails外部根拠：[2026-10-26 v3.0.0告知](https://wails.io/blog/wails-v3-released/)、[Wails Releases](https://github.com/wailsapp/wails/releases)。

<!-- tei-wails-ui-upgrade-live-context-20261103 -->
## 9. 2026-11-03 横断再深化：Wails UI拡張の実行時継続性とFunction配置（設計／NOT RUN）

#997 UI-5の更新と#1130 Function起動・単一/マルチBuild、#2598 Intent→実Request、#1174 Provider Commit/Receipt、#2391 Wails外部操作の隔離・除去を既存DS試験へ結ぶ。**Wails serviceを共通業務Hostへ昇格させず**、Studioと生成業務Targetを混ぜない。

### 一つの画面と複数の寿命
同じWails Windowの内側にも、Renderer component（mount/dispose・購読/Focus）、Editing Session（未送信Draft・Context Instance/epoch）、Application Operation（Request/attempt・unknown Commit）、OS Host resource（file handle/process/window）、Browser cache / built asset（版）という異なる寿命がある。Wails window reload後にMarkerが復元されても未送信Draftの実内容やAcceptを復元したことにはならない。逆に、再描画を理由に送信済Actionを再Submitしてはならない。

**受入追加**：
- DS-04/06：別窓や再起動、Kit v1→v2、Result更新の後着と旧epochが重なるとき、新画面/新Workspaceへ古いイベント・Draft・書込みIntentを流し込まない。未送信Draftを消す必要があるなら利用者に説明し、既存Sessionの保存・復元手段を確認する。見た目の復元≠Sourceの正式Accept。
- DS-07：Go/Wails bindingが返した成功・Timeout・Window crashは業務結果/Commitと区別。Request送信済みでACK喪失の場合、Operation/Request identityを維持した観測を要求し、自動再送せず旧UI再起動時にも状態不明を表示する。フロント側の同時送信抑止だけを冪等性と扱わない。
- DS-08：UI Extension/Native componentが直接Source・DBを編集したり、任意Go Methodを実行できる一般Bridgeを持ったりしない。実際の公開MethodとBridge到達性を確認し、拡張のService契約/Target bindingを明示。単なるSlot登録はsandboxではない。
- DS-09：Wails Window close/reopen、untrusted HTML/navigation、Main/child frame、Go service登録、assetの実digest、frontend bundleのCSS/React競合とIME/focus、起動/停止時のResource解放を実OSごとに観測。開発MCPを組込む場合は#2391に従いProductionビルドで物理不在を証明する。
- DS-10：同一Business FunctionをWeb/ReactとDesktop/Wailsから呼ぶとき、UI Kit共有部分だけで合格とせず、各Artifactの入口・実Provider経路・Host認可/Disclosure・Business Result/Error/Unknownを独立に比較する。

### FunctionとBuildとの対応（Build管理Ownerへの参照であり第二管理表ではない）
- 業務System/Subsystem/Functionの定義上の帰属 ≠ UI画面/Slotの配置 ≠ Build Unit/配備Artifact ≠ CLI/HTTP/cron等の外部起動入口。
- **単一Go実行ファイルを既定**とする。UIとJobを同じArtifactへ静的組み込み可能だが、含有FunctionがそのままCLIやWails Bridgeへ一般公開されるわけではない。呼出対象、必要なGrant/Capability、実Action適用を呼出境界で再確認する。
- Jobだけ別Buildへ移行する要求が現れた時点で、#1130が担当する既存Function ID→Entry Intent→Target/Build Group→Artifactの導出を利用する。Desktop UI Kitに再入力する配置表は追加しない。UIからJobを起動してもDispatcher/長時間Result/部分失敗/Receiptは通常のAction契約へ戻す。
- Wails Desktop binary、Web asset、API server、外部Workerは必ずしも同時に更新されない。Build profileとactual artifact identity、旧クライアント/新APIのContract range、公開Endpoint/Callback有無、offline/reconnectを照合し、silent fallbackや旧結果の現在化を拒否する。
- 依存更新で影響を受けるArtifactだけを再生成できるかは#943/#2600のBuild閉包で評価する。Sourceは同じでもKit/Toolchain/Framework/Providerが変われば、使える旧Evidenceは用途相対に限定される。

**P0 read-only受渡し票（正式Schemaではない）**：
`[Function/Action ID & accepted revision, UI Extension/Slot & kit pin, context instance/epoch/scope, logical entry, Target/profile, Build unit intent/resolved artifact digest, exposed route, actual Host/Provider/Grant, request/attempt, receipt/observation]`。
各列の正本は #997/#1158/#1130/#943/#2598/#1174 の**既存Owner**。DS試験は同一試行で照合できた列と未実証列を表示するだけとする。

**受入段階**：P0 契約・Build mappingのread-only説明 → P1 Web/React Slot/Actionの実試験 → P2 Studio Wails service/Window/UI拡張の実OS検証 → P3 **生成**Desktopの第二Target独立比較 → P4 分割Build・混在配備と復旧。UX-08〜14は #997 の追補に置いた。すべて実OS/実Provider/生成Artifactの新規E2Eは **NOT RUN**。新OS permission engine、Kit独自ABI、Scheduler、常駐TEI Hostはこの節では採用しない。

## 10. 2026-11-08 実装に着手するDesktop D1の限定受入：Inspector・Source・配布資産・作用を混同しない
<!-- tei-desktop-host-d1-proof-limits-20261108 -->

**状態**：#1824 Studio Hostの実装・テストに関する*受入計画*。§1〜9のHost/Target分離を維持し、#1251/#951/#1016のPlugin Contract/Resolver一般論、#3262のDesign Context、#2514/#1180のExpected/Scenario、#1174のEffect・Receiptを第二正本として複製しない。最初の契約は既存read-only `inspect` 一操作だけ。正式な公開API/Enum/ABI、UI Extension/Go Backendをここでは増設しない。

### 10.1 現mainとDraftの事実上限（固定して扱う）

| 観測済み事実 | ここで承認しない拡張 |
| --- | --- |
| `internal/operations/inspect.go::InspectSubject` とCLI、Wails `StudioBridge.InspectSubject` → `studiohost.ReadOnly` の既存Callはある。戻るWorld/SourceIndex fingerprintはread-only投影識別 | **選択したPlugin Providerを実Invokeした**、Sourceが完全、CAS可能、Accepted Meaningが現在も有効、とは言えない |
| `reference/tei-studio-workbench/main.ts` のCatalog/Inspect/Graphはread-only。Graphは単親Catalog最大8 Entry、`createResponseScope`は旧UI結果を失効する | Function Data Context/Provider Rowset、多重Relation/Graph Path、現在Disclosure/Grantの証明ではない |
| [main統合 #3318](https://github.com/hat47x/tei-platform/pull/3318)の標準業務Data例を受け、[main統合 #3334](https://github.com/hat47x/tei-platform/pull/3334)で`ApplicationView.standardBusiness`の `tax/exchange/numbering/holiday` を`tei view`とStudio **bootstrap `/`**の`assets/view-format.mjs`へ宣言投影した | 別Surface `/workbench/` Inspectorの二Consumerを満たした、実データの税額計算/DB読取/書戻しや本番Grantが成立した、とは言えない |
| [main統合 #3340](https://github.com/hat47x/tei-platform/pull/3340)はCLI `inspect/trace/view/knowledge-context`の`-h`とdiscoveryに許容値・既定値の共通出典を追加 | Helpに表示された引数・Plugin/Hook/Operation Offerが実Provider経由で到達可能、とは言えない |
| [#3327 exact-head CI #42385644277](https://github.com/hat47x/tei-platform/actions/runs/42385644277)でNode24 Workbench `npm test` **20/20**（A→B→A旧成功/失敗2件を含む）、typecheck/buildはPASS | 同じCIの**embedded bundle byte比較はFAIL**（dist/main.js対Wails組込main.js、649340 byte位置）。PR全体GREEN/配布一致/実WebView OKとしない |
| [main統合 #3284/#3296](https://github.com/hat47x/tei-platform/pull/3296)の9既存CLI返却・16合成負例による限定Design Context Gateは`partial`、`qualifiedTestGeneration=false` | Function/Business/Relation Required Closure・実Provider Read・外部Harnessへの完全受領資格を満たさない |
| [Draft #3328](https://github.com/hat47x/tei-platform/pull/3328)はExecutor後Recorder/Lifecycle書込み失敗を`EffectUnknown`へ保つ修正候補。 [Draft #3338](https://github.com/hat47x/tei-platform/pull/3338)は実行可能Offer・受信側Sink・不可逆Projectionを分離する横断設計 | 現mainの全Command ClientでUnknown UIや再送Fenceが完成、Provider read-onlyならAI/CLI/DevTools/Logの受信側への保存・外部転送も無い、とは言えない |

**Studioの二つのSurface**：`/`のPoC A bootstrap（`cmd/tei-studio/assets/main.js`と`view-format.mjs`）と`/workbench/`（`reference/tei-studio-workbench/main.ts`をBuildして埋め込む）は別のFrontend Artifact・RPC利用者。D1は`CLI inspect`と実`/workbench/`の二Consumerに絞る。`CLI view`／bootstrap `view`との同値検収は次の別Caseで行い、見た目・JSONが同一でも同じContract Provider/Artifact/Source Scopeとは推認しない。

### 10.2 D1で先に不足する証拠：Provider選択→実呼出しの一本

1. **Source/Caller**：同じ宣言Workspace名ではなく、実Root/Manifest byte identity/Source Index、受理資格・必要Domain Closure、CLIとStudioのCaller/公開Scopeを明示する。Source未読込・未Parse・対象外・非開示を`not-found`や完全な空集合に変えない。
2. **Selection**：既存#951/#1016のContract ID/Scope・Provider版・選択Lockと依存閉包を固定し、Plugin不在・Trait実装義務欠落・曖昧な複数選択・変更後の古いLockを未適格にする。Authoring ConsumerにRuntime用Providerを暗黙昇格しない。
3. **Actual Invoke**：既存`InspectSubject`のService adapterを再利用して、選択したProviderと実際に呼ばれたInvoker ID/Version/Artifactを**同一Attempt**で照合。Bを選択したのにAへディスパッチした場合、A/Bが同一JSONを返してもFAIL。既存静的CLI/Wailsコードへ無断fallbackしない。
4. **Two Consumer**：CLI `inspect` JSONとWails `/workbench/`実RPC/DOMを別call-siteからInvokeし、Result/Failure/Source/Disclosure/Unknownの扱いと実Provider IDを一致照合する。Node単体テストだけではWails Windowの資格を与えない。
5. **Actual Host/Effect境界**：D1はRead専用。ただしProviderへのRead requestと、Studio・Tool/Session・ログ・Screenによる**結果の受領・保存・公開**は別の観測。外部Sinkの保存/送出は当該Principal/環境/用途のDisclosure条件で扱う。D1を使った認可の横展開や書込みGrantを許さない。

### 10.3 検収・失敗先行の最小表（新Scenario SSOTではなく既存DS/#2514へ渡す）

| Desktop検収 | 一因子差・Expected | Product受入上限 |
| --- | --- | --- |
| **DS-D1-A** 実選択と呼出し | Contract Provider A/B、Pin B→実Invoke AをFAIL、Pin B→Bだけ局所合格。同じReply bytesに影響されない | 新規実テスト **NOT RUN** |
| **DS-D1-B** 二Surfaceと二Consumer | `/` Bootstrap Application Viewと`/workbench/` Inspectを区別し、D1のCLI Inspector ↔ Workbench Inspectorの別call-site実RPC/Invokerを突合 | 新規E2E **NOT RUN** |
| **DS-D1-C** Workspace/Subject ABA | A1→B→A2に戻っても旧A1成功・失敗を反映しない。別Windowを不必要に巻込まず、Source/Context revisionとUI response epochは別記録 | #3327 Node24 **20/20局所PASS**、実UI/Provider **NOT RUN** |
| **DS-D1-D** Artifact配布閉包 | Source/lock/build出力digestと実埋込みJSを比較し、差異を不合格にする。**現#3327の埋込み不一致を修復未確認のまま消さない**。既存Workflowが検出した差は別のOwner/根因で再確認 | 既存埋込み比較 **FAIL**。本書は修復しない |
| **DS-D1-E** Providerと受信側 | ProviderはReadでもCLI/Studio/AI Session/Trace/Logの保存先/送出Scopeが異なる負例。必要な部分受領/遮断/切断を`complete`へ昇格させない | 現CLI Context Gateは`partial`、実Sink適合 **NOT RUN** |
| **DS-D1-F** 将来のWrite/Unknown | 実行前拒否、送出後記録失敗`EffectUnknown`、ACK喪失、Receipt Lookup、独立Resource Readを別結果とする。`failure`から自動再送しない | #3328はDraft、Desktop Write **NOT RUN** |

### 10.4 実装順・運用と統合衝突の回避

- **P0 Source／Bundle**：既存`/workbench/`とCLIの実入口/Source/依存を固定。#3327で観測した`dist`対Wails埋込み不一致は、**main既存資産と同一Toolchainで再現するかを先に確認**する。失敗は`Source-Distribution-Mismatch`の構築資格として扱い、NodeのABA PASSと分ける。大きな生成JSの手書き修正を前提にしない。
- **P1 Provider**：#1251/#951/#1016のD1a shadow→D1b実選択へ。契約・Trait・Resolverの一つの仕組みをAuthoring `inspect`の既存Operationから消費し、二重Registry/Resolverを作らない。Probe/Adapterを既存Service境界へ接続する。
- **P2 実CLI/Wails二Consumer**：DS-D1-A/B/C/Dの実証。新UIの総当たりではなく現在の`inspect`1操作で完結させる。実Targetで広告と実引数/Outcomeの一致・Source Scope・Failure・Disclosure・Data mutabilityを別Claimで記録。
- **P3 Function実Read**：#2586のQuery→実Provider Row/Field/Scope/cohort→#2506のData Contextから、#2507/#3222の多表現を検証。現Catalog parent Graphに業務Relation多重性・集計結果・非可逆編集を推測合成しない。再読取/完全性が不明ならAction Offerを停止。
- **P4 Host/異質Target**：#3270 Data Source WorkbenchのSchema CRUD・JSONL、#3274 HTTP Decision Graph、SHO/SUI、生成Go Desktop Target、#3267 BOM 2×2は**別の実配置・実Consumer**として比較。Studio PoC成功を生成アプリに転用しない。

**正本とPR配置**：本§10はDesktop/Wails固有の検収**Owner #1824**。広域Plugin-first契約は[main統合 #3285](https://github.com/hat47x/tei-platform/pull/3285)の`plan/design/PLUGIN_FIRST_TOOLING_RUNTIME_DOGFOOD_2026-11.md`§8〜13、[Draft #3338](https://github.com/hat47x/tei-platform/pull/3338)§14がOwner。従来[Draft #3301](https://github.com/hat47x/tei-platform/pull/3301)のDC-01〜06からDesktop固有部分のみ移送した**置換候補**であり、#3301を同じPlugin共通文書へ並行mergeするのではなく、差分・受入票を照合して二重計画を解消する。既存#2514/#1180 Test/Expected SSOT、#3262 Design Context、#1163 Evidence、#1174 Host Effectを別名で再発行しない。

**今回の証拠区分**：確認したGitHubのmain Source/merged PR/固定CIの事実と、この文書の計画は別。**本書追加をもって、Go/Node/Wails/Windows実機/実Provider/実DB/実External Sink・Owner意味受理が完了したとはしない**。

### 10.5 実CLI観測を増やしてもD1の実Provider受入へ短絡しない（2026-11-08追補）
<!-- tei-desktop-host-d1-real-selector-closure-next-slice-20261108 -->

**mainで増えた限定実装**：
- [main #3350](https://github.com/hat47x/tei-platform/pull/3350)：既存Plugin Capability Coverageのdatabase-instanceは能力 `SupportsDatabaseProvider` によって実装候補を*選択*するよう更新。文字列一致や命名に依存しない選択のテストはあるが、selectedはContract実装・Build/Link・実Invoke・Current Grant・DB操作成功ではない。Desktop D1では**(a) advertised/declared, (b) eligible, (c) selected, (d) loaded/linked, (e) actual invoked, (f) receiver accepted**を一列のBooleanにせず段階観測する。最初に`inspect`対象のContract・Provider ID／版／Artifactを現在Selectionから解決し、既存Service実Callを同じAttemptに結び付ける。
- [main #3306](https://github.com/hat47x/tei-platform/pull/3306)：Hookが使う能力とApplication内Command Patternの接続がないとValidationで拒否する。DesktopでのOperation Offerは**実行経路を持つ宣言か**を調べる前提を得た。ただし複数Command候補・Phase・実HandlerとHost authorization、Unknownの正しい表示は未完了。UIへのOfferをGrantと同一視しない。
- [main #3354](https://github.com/hat47x/tei-platform/pull/3354)：既存CLIのRule/Canonical/Relation/Source間対応に対して、独立3-Rule slice/Canon/Sourceの合成負例を含む限定TT-56ゲートがある。これをSource同一性の*前段資料*とし、全Function/Business required nodes、外部Toolの実受領、Disclosure、Owner Meaningが閉じたとは判断しない。#3262のDesign Context `partial`と`qualifiedTestGeneration=false`は、別Packetで明示的に資格が証明されない限り維持する。
- [Draft #3356](https://github.com/hat47x/tei-platform/pull/3356)の第2回CLIドッグフード記録は、19 CLI commandsの検査範囲で**TEI独自MCP serverの実入口を確認できない**とする。実行できたのはCLI経路であり、MCP経路は`NOT RUN`。従って「CLI用のSkillはMCPにもそのまま使える」と宣言しない。D1の二ConsumerはあくまでCLIとWails Workbenchの**別の実入口**であり、将来のMCP Adapter追加は既存OperationへのAdapterとして独立に検収する。
- [Draft #3338](https://github.com/hat47x/tei-platform/pull/3338)の§15は、Data topology／Functionに配置されたRepresentation Instance／Binding topology、変更時の局所再資格化を担当する。Desktopで一つのNodeが複数Canvasに現れるとき、**Entity ID・View Occurrence ID・Function Context epochを混同しない**。Presentersの再描画やBundle新旧一致だけで必要Data SliceのContent/Read cohort/許可を証明しない。

**D1最小実試験への増分要求（#1180/#2514のScenarioへ渡す、ここには第二Fixture正本を作らない）**

| 反例／観測 | 独立Expectedと担当 | UI/CLIの停止線 |
| --- | --- | --- |
| DS-D1-G：技術は一致するが必要Capability不在／Providerが選択されても実Artifact不在／選択B・実呼出A | #3350 capability選択を入口に#951/#1016 actual invoke・#1251 Contract適合を別列で検査。例示BがAと同じResult bytesでもFAIL | `selected`や`available`だけで「実行可能」表示しない |
| DS-D1-H：3 Rule対応は観測できても必要Function/Business/Relation一部が未読／Provider未取得／非開示／空集合 | #3354 TT-56と#3262 Context Readinessを段階照合。必要な集合の閉包と独立根拠があるまでpartial。受信側Sinkへの完全配送は別票 | 空のGraph/Tableを「関係なし・0件・検証済み」に変えない。対象外と証拠欠落を区別 |
| DS-D1-I：同一EntityがCanvas A/Bに出る／旧View epochのReplyが再選択後に届く／Projectionが集約・非単射 | #2506/#2507/#2497/#3222 Function Context・Occurrenceの既存負例を参照。UI応答世代とData Read cohortを分離し、編集可能との誤認を抑止 | read-only Inspect以外の自動Writeback・Action Offerはなし |

**実装に渡す最小Receipt項目（構想上の観測列であり公開Schemaは未採用）**：
`[actual Workspace/Source fingerprint, required-domain closure status, caller/consumer, selected Contract/Capability/Provider/version, linked Artifact digest, observed invoker identity, operation+attempt, result fingerprint, receiver/sink/disclosure, UI epoch, unresolved reason]`。
初回はD1のread-only Inspectで既存Result/診断から取得できる項目だけを出し、未実装の実Invoker identityやSource Closureを**推測で埋めない**。実Callerが異なるCLIとWailsは各自のAttemptを持ち、同じ資格条件下で比較する（別Callを「同一Attempt」とは呼ばない）。これをデバッグ画面の常設公開ABIにせず、まずCIで比較可能な限定Evidenceとして使用。

**現時点の評価**：#3350/#3306/#3354のmain統合済み限定実証と、#3356/#3338の未統合設計・観測を区別。本追補の実CLI→Wails二Consumer、実Pluginロード/選択実呼出、H3・外部Sink、全Source Closure、Windows実機、Owner受理は**NOT RUN**。最初の欠けた辺を埋める優先順位（P0 Bundle/Source → P1 Actual Provider → P2 二Consumer）を変更しない。

### 10.6 CLI観測とStudio UI表示の残る非同値・Hookの経路未接続
<!-- tei-desktop-d1-client-contract-dogfood-d3-d10-20261108 -->

[Draft #3356](https://github.com/hat47x/tei-platform/pull/3356)の固定実行物による第2回CLIドッグフードでは、D-3（`view`は`semanticFingerprint`のみで他の`inspect`/`catalog`系との版表現が揃わない）、D-9（設計文書を意味へ取り込めない）、D-10（`validate`診断が二重構造のまま）が残存として観測された。**これらは本PRのコード修正ではなく、Desktopで不完全な証拠を完了表示しないための既存未達条件**。

| 下流の検証を追加する場所 | 異質Consumerの期待結果と責務 |
| --- | --- |
| DS-D1-B／#1824（CLIとWorkbench） | `InspectSubject`のWorld/SourceIndex fingerprintをUIの表示版識別に用いる範囲と、`view`の`semanticFingerprint`しか存在しない範囲を**別形式として観測**する。見た目が同じでも同一World・Source epochを補完しない。D-3の契約統一は#3149/#2824のOwnerと調整し、Workbenchだけに独立Revisionを作らない |
| DS-D1-E／#2514（診断・文書） | `validate`の`Direct`/Diagnostics配列とCLI `diagnostics[]`の構造差を、その場限りの正規化と元応答の保持に分ける。`missing`・`null`・`[]`・`not-run`を「診断0件」へ丸めず、未解析の設計文書をMeaning候補として自動受理しない |
| DS-07／#1158（後続Write、現D1外） | [main #3348](https://github.com/hat47x/tei-platform/pull/3348)でHookの位相順、warn継続、不成立・失敗時の停止位置を`RunSequence`のGo局所試験で固定。ただしCLI/Studio実行経路への呼出しは未接続。Hook順序の表示・広告を、実行した事実／Transaction境界／Plugin選択／外部Effectと同一視しない |

**First Missing Edgeの更新**：実Read-only Inspectorへの次の一手は、`requiredDomainClosure`がunknown/partialの場合に**Provider選択へ先行させないこと**ではなく、十分に収集できた範囲を限定表示し、未実証の追加Source/Role/Relationを欠落と誤判定しないこと。実Provider D1で要求されるSource/Contract/Disclosure閉包が揃わない場合、`qualifiedInvoke=false`のまま実呼出適格性を否定する。これは設計候補であり、現CLI/Hostの既存Result Schema/Permission/Owner受理を上書きする強制的な新boolean fieldではない。

### 10.7 実コード上の最初の欠落辺：InspectはRootServicesを使っていない
<!-- tei-desktop-d1-first-missing-host-seam-20261108 -->

**2026-11-08のmain実ソース照合**：
- `internal/studiohost/readonly.go`：`RootServices`の提供口は`Catalog CatalogService`と`Validation ValidationService`だけ。`ReadOnly.CatalogWorkspace`／`ValidateWorkspace`は非nil Factoryを使い、必要なServiceが欠ければ拒否する。一方、`ReadOnly.InspectSubject`は現在`h.workspace(id)`のallowlist/root same-file照合後、`operations.InspectSubject(root, ...)`を**無条件に直接呼び出す**。`services` Factoryによる選択/差替えは通っていない。従って既存Wails `StudioBridge.InspectSubject`が動いた事実をPlugin-backed Inspectと読まない。
- `cmd/tei/main.go`：`inspect`分岐は`inspectWorkspaceJSON`を直接呼ぶ。Studioだけの差替えに成功しても、同じContractがCLIと二Consumerで使われたとは認めない。
- `internal/plugin/runtime/service_carrier.go::validateServiceSelection`：`selection.Scope != "application-runtime"`を明示拒否。Authoring `inspect`へ旧CarrierのScope文字列を追加/変更するだけでは、Target/Host所有のSelection・Artifact・Current Grantを保護できない。[main #3300](https://github.com/hat47x/tei-platform/pull/3300)のAS-01〜12と実装着手票は**この不足辺の詳細Owner**であり、本書で二重の選択器や新契約を実装しない。

**直近の差分最小案（DS-D1-A/Bの実装担当へ渡す）**：

1. **Studio側のService境界**：現`RootServices`に対応するInspectの*型付きread-only Port*を加えるか、既存operationsのService化候補を利用する。nil Factory時の既存PoC経路と、non-nilでProviderが不在・誤選択のときを分ける。non-nilのD1選択経路でInspectorが無い場合に**暗黙の`operations.InspectSubject`へfallbackして成功させない**。実変更前に既存RootServices consumer/Go testsに対する影響を列挙。
2. **CLI側の同一Contract**：CLIは既存`inspectWorkspaceJSON`を見せかけだけ包装せず、実際のContract/Provider選択→Host-owned Binding→実Invoker/実Sourceを照合する。StudioとCLIは別Attemptでよいが、それぞれ同一SelectionとResultに対して**actual Provider identityの証拠**が必要。
3. **負例を先に固定**：Workspace許可外/Root identity変更 → ServiceFactory呼出`0`回、Selected B→Actual A→Result bytes同一 → FAIL、Non-nil FactoryでInspect Service欠落 → FAIL、Scope application-runtimeの旧CarrierをAuthoringへ誤流用 → FAIL、古いSource/Provider pin → FAIL。正常CaseはHostが選んだContract/Providerで一度だけInvokeし、Responseに残す出所とConsumerが一致する。Factory nilの既存PoC結果は互換性として保持しつつ、**Plugin実証の合格数に含めない**。
4. **本番権限ではない**：RootServicesへのPort追加は受理済みModel/Source Closure、Build Artifact trust、OS Grant、Actual Provider Read、Owner Meaningを証明しない。D1はread-onlyのみ。後続Function Data Context/Graph Canvas/Write Receiptは同じ受入票の別列。

これでD1の**First Missing Edge**は「抽象的なPlugin/Studio統合」ではなく、**Studio `ReadOnly.InspectSubject`の実Provider Port不在とCLI直接分岐**と特定できる。上流のSource/Providerの選択・現在資格も未解決のまま明記し、旧Runtime Carrierを破壊的変更するのではなく、Host-ownedなAuthoring入口から既存#1251/#951/#1016の同じ選択機構を消費する。

**資格**：本§はmainの3 Goファイルと[main統合 #3300](https://github.com/hat47x/tei-platform/pull/3300)の静的照合による計画。新Port・差替え・Go単体試験・CLI/Wails二Consumer・Plugin実Invoke/Host Grant、正式受理は**NOT IMPLEMENTED / NOT RUN**。

### 10.8 D1の実装候補を限定受入へ接続し、選択表示と実利用を区別する
<!-- tei-desktop-inspection-port-slice-20261108 -->

**実行に着手した範囲**：[#3371](https://github.com/hat47x/tei-platform/pull/3371)は、§10.7で指摘した`ReadOnly.InspectSubject`のServiceFactory迂回に対し、既存`RootServices`へ`InspectionService`を加える**小さなGo変更のDraft**。Factory無しのPoC read-only経路は維持し、Factory非nilのときInspector未提供なら暗黙の`operations.InspectSubject`へfallbackせず拒否する。Subject/schema/effect/Canonicalの妥当なJSON/World・Source Index fingerprintが欠落・不整合となる応答は成功扱いしない。Root allowlist/Identity検査はFactory呼出前に維持し、Host固有エラーは公開Responseへそのまま返さない。

**新Go試験の役割**：Factoryからの一回の実Service呼出し、旧built-inへの無断fallback拒否、誤Subject・Effect・版/内容欠落、許可外Workspace/Root差替えのFactory呼出0回、Host-private error秘匿を対照する。これらは**Host-selected typed Inspector入口の局所試験候補**であり、PluginのContract/Provider exact選択、実Composition Lock→actual invoked Providerの対応、CLIとWailsが二つの独立Consumerとして同じProviderを利用した証拠ではない。初回PRの[Actions](https://github.com/hat47x/tei-platform/actions)は作成時queued/pendingで、Go/CIのPASSは未確認。ここでDS-D1-A/Bを`PASS`にしない。

| 次の検証順 | 必要な反証・確認 | 間違った合格判定 |
| --- | --- | --- |
| **S1: Host Port** | #3371のFactory非nilでService missing/Error/Subject mismatch/Allowlist異常がFail closed、Factory nil legacyが保持。PR exact-head Go+Repository Healthを実測 | テストコード追加だけで実成功、Contract選択済みと主張 |
| **S2: Composition actual** | #3300 AS-01〜12のHost-owned Authoring Selectionと同一Scope/Provider版/Artifactを`InspectionService`実装に束縛。Pin B→実Aは結果同一でもFAIL。旧`application-runtime`専用Carrierを流用しない | typed Portができた＝実Plugin Invoke／Runtime認可あり |
| **S3: CLI+Wails** | CLI直接`inspectWorkspaceJSON`経路も同一Contractへ切替。CLI・`/workbench/`を別Attempt/別Call siteで実呼出し比較。Workspace/Source/Disclosure/Unknownと同一Artifact、旧A→B→A応答とBundle配布バイトを検証 | Studio単独Port PASSやNode UI20/20を二Consumer／実Desktopの合格へ流用 |
| **S4: Data Source/Offer** | [Draft #3363](https://github.com/hat47x/tei-platform/pull/3363)ではreference/workspaceの24 Binding中22 selected・2 missing（`editor.database_workbench`、dev/prod）。ただし現Validatorは`editor.sql_analysis`のみ能力照合、`database_workbench`にはKnown検査のみ。#3270/#951の適用条件・実Provider Invoke/Target検証までを別資格として説明 | 参照Matrixのmissingを製品エラーとして強制／selectedを実対応・Grantに昇格 |

**判断上の留意**：Studioが扱うread-only結果でも、CLI／Tool／Session Log等の受信側は保存・送出し得る。Source/ResultのDisclosureとSinkの観測は#2515/#3338に委ねる。#3354の11実CLI Reply・32局所対照はRule/Source Closureの**一部を拒否できる**証拠で、全設計情報・外部Tool配送やWails側の実選択資格ではない。#3359のHook Reads限定はDraftで、RunnerからCLI/Studioへの配線とProduces sideは未実証。FunctionのOfferやHook宣言が画面に見えても実行可能・意味受理済みとは表示しない。

**計画統合の状態**：本§は同じDesktop設計Draftへ実装候補の所在・依存関係・不合格条件を追記したもの。#3371は別Draft、#3352はまだmain未統合、#3363/#3359も未統合。新しいWails/Windows/Provider/CLI+Studio二Consumer・外部Sinkの試験を本節では実施していない。**既存FactoryのCatalog/Validation専用ConsumerがInspectを呼ぶ場合の後方互換は別途実Call Site調査を要し、Factory非nilでInspector未提供を黙って成功に戻さない。**§10.1〜10.7や#1180/#2514の独立Expected/Gateは維持する。

### 10.9 共通Application API・Plugin実利用・Source鮮度の三つの受入を交差させる（2026-11-08）
<!-- tei-desktop-adr0048-source-drift-20261108 -->

**追加判断の正本**：[merged #3439 / ADR-0048](https://github.com/hat47x/tei-platform/pull/3439)により、CLI/MCP/HTTP/Web/Desktopは**同じ型付きApplication Operation**を異なるAdapterから利用する方針が採用済み。これは#1251/#951/#1016の**実際のPlugin Provider選択・実Invoke**とは別の条件であり、どちらか一方のPASSからもう一方を推論しない。共通APIはHTTP必須でなく、StudioはHostが許可したRootからプロセス内／IPCで呼んでよい。CAPI-1〜4の全体受入は[#3419〜3422](https://github.com/hat47x/tei-platform/issues/1158)がOwnerであり、本節はDesktop`/workbench/` Consumerの失効・表示・配布を限定して扱う。

**観測済みの段差**：
- [merged #3458](https://github.com/hat47x/tei-platform/pull/3458)は`validate`についてCLIと**参照MCP Adapter**の共通投影、ロード失敗、意味エラーの`isError`／`ok`／文字列failure.codeを実Go対照で修正・検証した。CLI JSONが同byteでもStudioの`ValidationResponse`・UI診断・実IPC・現在Disclosureが同値になった証拠ではない。`catalog/inspect/trace`の全MCP envelope・未知フィールド・MCP規格適合も別。
- **Go Portの状態**：[Draft #3371](https://github.com/hat47x/tei-platform/pull/3371)はHost-selected`InspectionService`への委譲、[Draft #3381](https://github.com/hat47x/tei-platform/pull/3381)はCLI側`inspectWorkspaceJSONWithService`の型付き注入口、[Draft #3378](https://github.com/hat47x/tei-platform/pull/3378)は旧Direct CLI↔Studioのshadow parity。**3件ともmain未統合**であり、通常起動の`inspect`がCompositionで選ばれたProviderを実Invokeしたという証拠は無い。
- **検証の実際**：#3371旧HEAD `a42df66...` の[Studio PoC A CI #42395396780](https://github.com/hat47x/tei-platform/actions/runs/42395396780)は**FAIL**（既存`TestInjectedOperationsOnlyReceiveAllowlistedResolvedRoot`が旧「InspectorはFactory非経由」期待で失敗）。全体[Health #42395396786](https://github.com/hat47x/tei-platform/actions/runs/42395396786)も`readonly.go`と当時の`coverage_test.go`のgofmt、上記Go testでFAIL。#3371には旧テストのfail-closed契約への更新が追加されたが、**最新HEADのCI成功は独立再確認が必要**。#3381／#3378の旧HEAD Health FAILも別Scopeの整形や環境差を含むため原因を確認し、PASSとは扱わない。
- **Source-only driftの異なる二つの観測**：[merged #3477](https://github.com/hat47x/tei-platform/pull/3477)はConsumer所有Source Indexだけが変わる実CLI事例を検出。[Draft #3479](https://github.com/hat47x/tei-platform/pull/3479)はImported Provider Raw／Manifestだけが変わり、ConsumerのWorld／SourceIndex／ModelRevision・意味digestが**変わらない**反例を追加。後者はProvider原文・Manifestの再取得による限定試験で、同一の原子的Provider Snapshotや実D1 Provider Artifactの失効範囲は未証明。詳細なSource/Build/BOM適格性は[#3488](https://github.com/hat47x/tei-platform/pull/3488)と#2743/#2600/#3267へ委ねる。

**Desktop固有の受入（#1180/#2514の既存Scenario/独立Expectedへ引き渡し、新しい正本を作らない）**：

| ID | 反例／比較条件 | Desktopで期待する処置と判定上限 |
| --- | --- | --- |
| **DS-D1-J／CAPI** | 同じHost許可Workspace・Source版で`validate`のCLI/MCPを共通APIから取得。別途Studio`/workbench/`も表示。意味不合格と操作失敗・Transport失敗を変える | `ok`と検証合否・警告・Source Locationを混ぜない。MCP `isError`はプロトコル用、安定failure.codeは意味用。Studioは独立実IPC/DOM観測が無ければ未合格 |
| **DS-D1-K／Plugin** | Typed Resultは一致しても、HostでProvider Bをselected／実Invoke A、またはPortだけ注入／Actual Sourceなし、Service未提供でBuiltinへfallback | `B selected→A invoked`はFAIL。Port有り・結果一致・Scope/Capability selectedだけではPlugin-backed Operationを広告しない。#3371のFactory nil PoCはcompatibility lane |
| **DS-D1-L／Imported Source** | Consumer World/SourceIndex/Context epoch不変のままImported ProviderのRaw末尾LF／Manifestだけが変化。Inspector画面・Source引用・Result cacheを再表示 | Meaning投影の再計算要否と、**Source Anchor/引用の鮮度・Disclosure**を分ける。Consumer FP一致だけで`source current`/引用保証を表示せず、対象Provider revisionの再観測が無いとstale/unknownに留める。変化した全Providerで一律Build再実行も要求しない |
| **DS-D1-M／Delivery** | `/workbench/` Nodeテスト成功だが、生成`dist/main.js`とWailsへ実埋込んだbyteが違う。さらにA→B→A旧応答、Receiver Session/Log保存あり | 実配布Asset SHAとCLI/Studio Source/Resultの両方を別資格として確認。旧`#3327`の[Node24 UI20/20とbundle byte FAIL](https://github.com/hat47x/tei-platform/pull/3327)を消さない。UI世代失効は実Provider Cancellationや秘密のSink非残留を証明しない |

**一改訂の最小実証順**：①CAPIは既存`validate`共通投影の結果型/失敗を**CLI・参照MCP**の限定基線として固定（新Go Core共通Serviceや第二HTTP Hostを作らない）→②`inspect`について#3371/#3381の**既存型付きPortの契約差**を検査し、重複Result guardの有無を確認してから共通Operationへ束縛→③Authoring scopeでHost-selected exact Provider→actual invoked identityを一Consumerから試験→④CLIと実`/workbench/`の別Attempt／別Call-siteで同一Contract・Source／Workspace／DisclosureとUI後着を比較→⑤Imported Provider Raw/Manifest Source-only driftをSource引用UIとCacheから圧力試験→⑥Bundle byte・Windows WebView2・Receiver Sinkを実Targetの独立資格で検収する。最初から`validate`のMCP合格を`inspect`のProvider合格や全DesktopのContract実装完了に昇格させない。

**Product表示への制約**：`Inspection.Subject/Canonical`をRendererへ渡せても、Required Relation/Function ClosureやHost Grantは推測補完しない。取得範囲外は空集合・検証合格ではなく明示的に未観測／部分とする。外部Providerの出典が変化したときは画面上のSource引用の現在性を再判定し、未受理の別Meaningへ変更したと扱わない。現`readInspection`が照合するWorld/SourceIndex二Fingerprintと`createResponseScope`のUI応答世代はこれらの外部Source/Provider資格の代替ではない。

**状態とOwner**：ここで新しいGo/Node/CLI/MCP/Studio/Provider/Windows実試験は**NOT RUN**、本§は設計だけ。採用済ADR-0048とmerged #3458/#3477、未統合Draft #3371/#3381/#3378/#3479/#3488を明示的に区別。共通APIは#1158/#3419〜3422、Resolver/Providerは#951/#1016/#1251、Source鮮度は#2743、Build/BOMは#2600/#3267、実UI/配布は#1824、シナリオと独立Expectedは#1180/#2514。既存§10.1〜10.8・DS-D1-A〜I・PD/Gateを上書きしない。


### 10.10 Desktop固有のHost接続・Graph・Window状態の追加ストレス（2026-11-08）
<!-- tei-desktop-d1-host-graph-window-negatives-20261108 -->

§10.9の**共通Application Operation／実Plugin選択／Imported Source鮮度**に対し、本節は実`/workbench/`・Wails Hostの固有反例だけを補う。Contract/Provider選択の方式、Source/Build/Design Context、独立ExpectedのOwnerは§10.9と#951/#1016/#1180/#2514を参照する。第二Registry/Dispatcher/Graph Engine/Scenario正本は作らない。

- **Host Serviceの接続**：Factoryを有効にした既存`RootServices`はCatalog/ValidationへのService注入が先行しているが、mainのInspectorはbuilt-in直呼出し。[Draft #3371](https://github.com/hat47x/tei-platform/pull/3371)のInspector Portは未統合で、Factory非nilかつInspector Binding欠落をfail closedとする変更と既存Go試験の期待変更を**同じHEAD**で確認する必要がある。Factory nilのPoCは互換性経路であってPlugin資格ではない。
- **Workbench Graphの意味限界**：現`relatedGraph`は単親Catalogから最大8 Entryの参照ナビゲーションを作るだけ。#3192/#3222の多重Relation/Result grain/Representation occurrence、部分取得・非開示・集約の負例を実Data Graphの資格へ拡張しない。印刷Hookの宣言が描けても実Hook Invoke／印刷効果と同一ではない。
- **Window／配布の独立性**：Node側のUI後着拒否・A→B→Aテストは、実WebView2 Window再起動・複数Window、実Wailsに埋め込まれたFrontend Asset digest、Host/Provider Read cohort・Grant・Receiver Sinkの完全性を証明しない。Bootstrap`/`と`/workbench/`も別のFrontend Artifact/Consumerである。

| 既存DS/#2514へ渡す追加照合 | 一因子反例 | 必須の停止条件 |
| --- | --- | --- |
| **DS-D1-11** | FactoryはCatalog/Validationのみ提供 | 既存2操作は注入Serviceを利用し、Inspectorは`OPERATION_FAILED`。built-in暗黙fallback不可。Factory nilの正常系を対照 |
| **DS-D1-12** | Inspectorが別Subject／write Effect／不正Canonical JSON／欠落指紋を返す | 誤Projectionを成功表示しない。Host-private errorをRendererへ流さない |
| **DS-D1-13** | Consumer FP不変・Imported Provider Raw／Manifestだけが変化 | 現在のProvider Source引用とConsumer World Contextを別に判定（Source版の主張は§10.9／#3479） |
| **DS-D1-14** | CLI/MCP/Desktopの片入口だけ受領が欠落し、同じApplication Operationは処理済み | Transport配送とOperation/Effectを分離。無断local fallback・write自動再送なし |
| **DS-D1-15** | Catalog Graphに多重Relation・集約Row・秘匿Edgeを入力 | Catalog navigationの一致は実Relation多重度や現在Disclosureの証拠ではない |
| **DS-D1-16** | 旧Inspect応答が再選択A→B→AのA2やWindow再起動後に届く | UI epochで旧応答を拒否。Source Revision/Provider Raw/Host Grant・実Effectまで自動再受理しない |

**実測と計画の線引き**：#3371旧HEAD `a42df66` の[Studio PoC A CI #42395396780](https://github.com/hat47x/tei-platform/actions/runs/42395396780)は旧Direct Inspector期待でFAILし、Healthも整形差を検出。旧Testは新しいfail-closed期待へ改訂されたが、最新exact HEADの合否は別。#3479のProvider Raw/Manifest-only Source実負例もそのPRの最新HEADと独立に検証する。本節は**計画・検収票のみ**で、Wails実UI／CLI-Plugin実Invoke／実Provider Read／Owner Meaning AcceptanceはNOT RUN。


### 10.11 選択Providerの局所実呼出しReferenceをDesktop実Consumerへ接続する停止線（2026-11-08）
<!-- tei-desktop-selected-inspector-actual-reference-20261108 -->

**今回増えた実装証拠**：[Draft #3381](https://github.com/hat47x/tei-platform/pull/3381)は型付きCLI Portに加え、既存Composition Registryのauthoring-tool選択とHost登録済みInspector関数を結ぶ内部Adapterを実装した。局所Goテストは、Bを選択→B実呼出、B選択→A登録の拒否、Contract/Scope/Artifact不一致、Host Guard撤回、Registry epoch失効・A→B→A、無関係Scope変更、Result不適合を対象とする。**Reference内の実関数呼出しが存在する**ことは前進だが、既定CLI起動のBuiltin経路やStudioのWails Bridgeは未接続。選択ID/合成digestだけで実Build Artifact・署名・現在認可を証明しない。

**二Consumer化の最小変更境界**：①#3381の既存Selected CoordinatorをCLI起動から利用するHost-owned設定・失敗経路を検証する。②[Draft #3371](https://github.com/hat47x/tei-platform/pull/3371)のStudio InspectionServiceへ同じContractのHost登録済み実装を注入する。③[Draft #3378](https://github.com/hat47x/tei-platform/pull/3378)の既存Direct CLI/Studio shadow parityは**移行前の比較基線**として維持し、両Consumerの実Provider呼出し・Attempt/Source/Disclosureの独立Expectedを別途要求する。CLIとStudioが同一Go関数を偶然呼ぶだけではPluginの選択同一性の証拠にしない。新Resolver、第二Application API、別Host、第二Graph Engineを作らない。

| 次の受入 | 正例・負例の最小条件 | 資格の上限 |
| --- | --- | --- |
| D1-R1 | 同一LockでB選択→CLI通常起動がBを実呼出。B選択→A実呼出/未Bindingは拒否 | テスト専用注入経路のPASSを通常CLI起動へ流用しない |
| D1-R2 | 同じHost選択BをStudio Factoryへ接続、実Wails /workbench/の別Call siteでBを実呼出。Factory nil Builtinを対照 | Studio Go Host単体・Node DOMだけでWails実Consumer PASSにしない |
| D1-R3 | CLI/Studio別Attemptの同一Contract/Scope/Source版、Provider Raw drift、Disclosure、unknown/partial、選択撤回、A→B→A | 値一致・World FP一致だけでProvider Revision/現在Grantを受理しない |
| D1-R4 | 実Link/Bundle/Frontend Assetの選択版一致、異版混入、WebView2での実操作 | Reference Registryの合成IDを実Binary/Conformance/Verified BOM資格にしない |

**関連する別系統の進展**：[Draft #3492](https://github.com/hat47x/tei-platform/pull/3492)の三視点Graph/CRUD、[Draft #3496](https://github.com/hat47x/tei-platform/pull/3496)のREA外部Evidence、[Draft #3491](https://github.com/hat47x/tei-platform/pull/3491)のUI整数精度、[Draft #3497](https://github.com/hat47x/tei-platform/pull/3497)のMigration履歴Unknownは、Desktopに接続し得るが、現D1 Inspectorの受入を拡張・合格させる根拠ではない。Data/Function/Businessの意味受理、外部Observation、表示値と型付きAction引数、DB Apply履歴を別のSource/Effect資格として維持する。

**実測状態**：#3381最新HEAD 5cdd3b9、#3371最新HEAD e5510f4の各Actionsは確認時queued/pendingが多く、**exact-head Go/Health全PASSは未確認**。#3371のWebView smoke単独SUCCESSは、D1-R2の実Provider呼出し成功ではない。#3378の既存Direct parity試験もSelected Providerを通さない。ここでは計画Markdownだけを更新し、実Go/Windows/DB/REA/Graph/Owner受理はNOT RUN。


---

## 11. Desktop製品統合の現行判断と実測引継ぎ（2026-11-14）

<!-- tei-desktop-product-integrated-20261114-v2 -->

本節は、§1〜10.11のDesktop Host境界を現在の製品開発へつなぐ**受入計画**である。§1の「実機起動未実施」は2026-10-13の基準時点を示す。後続の実装・CIの到達点は、各PRのexact HEAD・Attempt・Targetに限定して以下に記す。新しいDesktop Host、意味Resolver、認可機構、Build Engine、Test Runner、共通状態Schemaは導入しない。

### 11.1 現在の到達点と、成立していない資格

| 対象 | 既存の限定実績 | Desktop製品として未成立の点 |
| --- | --- | --- |
| 実Windows生成版（#4583／[Draft #4587](https://github.com/hat47x/tei-platform/pull/4587)） | exact HEAD `357345d7`のWindows Workflowは**生成版Job成功**。N0→N2選択、N2→N3埋込み、生成版のWindows WebView2受領が報告された | 同じWorkflowの**通常版JobはPowerShellのOSVersion取得15秒タイムアウトで失敗**。全Workflowの成功でも、正式BuildIntent選択でも、Stage1→Stage2再入でもない |
| 選択Builder（[Draft #4634](https://github.com/hat47x/tei-platform/pull/4634)） | exact HEAD `7d6a1238`において8/8 Workflow成功。宣言→Resolver整合拒否と起動前Child digest照合を同じCheckoutで検証 | 対象は主にPlugin BuildPlan。**non-Plugin Studio Subject→全N0/N1/N2/N3→ArtifactSet**の正式実行、OS Loader読取時点の全bytes、Stage1→Stage2は未受理 |
| 公開面監査（[Draft #4627](https://github.com/hat47x/tei-platform/pull/4627)） | exact HEAD `b9cf5447`、5/5 Workflow成功。登録Serviceと15公開Methodの静的Inventory | 実HTTP/WSの認証済みPrincipal、Workspace Grant、Event Audience隔離・SHO/Commandの公開拒否は未証明。Wails Serverは現状loopback PoC |
| 外観（[Draft #4628](https://github.com/hat47x/tei-platform/pull/4628)） | Node/Chromiumで明暗・OS追従・密度・表示状態保持等の限定試験が報告された | 変更HEADごとの全CI、実Windows WebView2のIME・読み上げ・全アクセシビリティ・生成業務フォームの製品受入は別 |
| REA bounded read（[Draft #4637](https://github.com/hat47x/tei-platform/pull/4637)） | 従前HEADでは12/12 Workflow成功。Host固定Selector→bounded bytes→既存R2 read-only投影の限定Go実装 | **新たなHEADの結果を旧HEADから継承しない**。Studio Bridge/Web/AI Consumer、現在Grant・Audience・Source A→B→Aの正式Product受入は未実施 |
| Script Launch（[main統合 #4636](https://github.com/hat47x/tei-platform/pull/4636)） | 不正なInput/Effect宣言をGo Adapter呼出し前に拒否する限定実装が統合済み | 実Providerの選択・Current Grant・業務Effect/Receipt・通常CLI/Studio製品入口が受入済みという意味ではない |

**優先する実証**は、①現行A方式での実Windows Studio IPC→Source-backed Catalog/Inspect、②非Plugin Studio BuildIntentの実Invoker/ArtifactSet、③生成Stage1自身のStage2再構築、④当該Stage2 WebView2実受領である。異なるHEAD／異なるAttemptの成功数を足して①〜④合格にしない。

### 11.2 ローカル実行・通信の方式を四つに分離

| 方式 | JS→Go入口 | Go処理の配置／配布 | 評価段階 |
| --- | --- | --- | --- |
| **A：同一Process** | 現行Wails Bridge | Wails GUIとApplication Operationが同一Goプロセス。標準一exe | **P0基準** |
| **B0：同一exe従属Worker** | Wails Bridge→親Go | 同じexeをWorker modeで自己起動。親→子のNamed Pipe/UDS等は別設計 | **P1比較** |
| **B1：別Wails Server** | Browser/Frontend→HTTP/WS | `-tags server`等で構築した**別Artifact**。同一exe配布は未証明 | **P1比較** |
| **B2：別Server/Worker＋Local IPC** | Wails Bridge→親Goまたは限定Local UI | 別Worker/ServerとNamed Pipe/UDSを用いる。OS process/loader/ACLに別責務 | **P1比較** |

JS↔Wailsの既存Service Bindingと、Go親↔子のOS IPCを混同しない。B0でWebViewがNamed Pipeを直接開くことは前提としない。Wails Custom Transportは必要時の比較対象であって、B0の実装に必須ではない。

B0/B2の検討ではOSのACL・peer identity・接続元Session・実行File Digest・復旧時のWorkspace/Configuration Epoch・secret deliveryを一組として検査する。Pipe名や起動tokenを単独のHost Authorizationとみなさない。B1ではOrigin/Host/loopback制限と、認証・現在Grant・WS Audience別配送を独立検証する。Web Shared Hostにlocal no-authをfallbackしない。**P0のStage1→Stage2へB系比較をHard dependencyにしない**。

### 11.3 Desktopの編集・表示・実行資格

UIで保持する**Focus、Pin、Editor Dirty Buffer、Undo/IME、Panel/Window Session、Theme/密度**はPresentation Stateである。Source Revision／Provider Raw Revision／Model Acceptance／Plugin Settings Version／Build Artifact／Execution Request・Receiptとは別のものとして扱う。新たな一括状態表は作らず、既存OperationとHostから取得できる識別子を、当該UI応答の照合に使う。

- **Markdown/SHO**：通常の`*.md`の原文保存はDocument Writerの責務。管理対象`*.tei.md`のTEI意味SourceとしてのLoader/Parser bindingは#2621/#3030の受入が必要。未束縛Parserを黙って通さず、独立した非管理Draft保存は可能にする。SHOから編集できてもAccepted Meaningの変更とはしない。
- **三視点と図面**：Text/Structure/Graphは同じSourceへの投影。Graphレイアウト、Fold、Pin、外観CSSをSourceの意味変更にしない。多重辺・向き・関係ID・粒度の不足をCatalogの8ノード程度のnavigation表示から補完しない。#1124/#4611/#4612へ引渡す。
- **外観と機能**：#4628のTheme/密度はHost/Plugin設定やBuild Profileから独立したローカルPreference。Theme変更でEditor remount、Dirty Buffer消失、追加のHost/Provider呼出しを起こさない。Light/Dark/System、幅390/1024/1440、倍率・IME・キーボード・読み上げで別検収する。
- **生成/ビルド**：Preview ReadyやSource digest一致を、正式Builder選択・Build適格性・実Artifact Consumer受領に昇格させない。#4641のGenerator Contract、#1080/#1097のBuild Execution/ArtifactSetへ接続。仮のPlugin Subjectでnon-Plugin Studio Buildを偽装しない。
- **Plugin管理**：宣言、導入可能性、有効設定、構成選択、実Build包含、実Invoke、現在Grant、Source保持を独立表示する。Inventory→Plan→Apply。実設定源が未確定なら`notModelled`を維持し、設定変更とRuntimeの開始/停止、データ削除を自動同一視しない（#4376〜#4380/#2550）。
- **外部Evidence/DB/AI**：raw OS PathをWails/Web/MCPへ一般公開せず、Host固定Workspace/Resource Selector→bounded bytes→既存Operation。Read時の資格と画面・AI・WSへのDelivery/Disclosureを分ける。DB結果0件と未読取／partial／権限遮蔽を区別し、DB移行・非同期Effectの完了は実Commit/Receipt/独立Readで検証する。
- **Command**：参照実行の`mode=reference, committed=true`を外部DBへのCommitと表示しない。接続断、timeout、cancel、Worker crashは実業務Effectの失敗証明ではない。既存Request/Receiptをlookup-firstで照合し、unknownのまま再Submitしない。

### 11.4 UI後着・Process境界の必須負例（既存#1180/#2514へ接続）

| Case | 独立した対照入力 | 必須Expected |
| --- | --- | --- |
| **DH-01 現行Source** | W-AのInspectを送信→W-Bへ変更、同名IDで異なるSourceを選択 | AのReplyは現DOM/ARIA/AIへ不採用。Aの実Operationの有無は別票 |
| **DH-02 ABA/Window** | 同じWorkspaceでA→B→A、Panel close/open、別Window、Plugin A→B→A | byteや文字列一致だけで旧Replyを再受理しない。現Scope/Epochに束縛されないOutputはstale |
| **DH-03 Edit/SHO** | 未保存`*.tei.md`＋外部Writerの競合、通常`*.md`、生成物read-only | Dirty内容喪失0、Sourceの無断書戻し0、意味受理・Artifact書換え0。Parser未束縛を意味成功にしない |
| **DH-04 Worker/Effect** | 正常、子crash、親crash、切断、timeoutの同Request／実Provider | Process/transport failureとEffect unknownを区別。自動再Submit 0、Lookup/独立Readが得られた範囲だけ確定 |
| **DH-05 Public Surface** | 2 Workspace×2 Principal×2 Tabでread/plan/SHO/command methodとWS Eventを調査 | Method Inventory PASSだけでPublic Web安全扱いにしない。未認可実HTTP拒否、Audienceへの機密Event誤配送0、local no-auth fallback 0 |
| **DH-06 Build再入** | non-Plugin Studio正式Subject、N0→N1→N2→N3、Stage1 Process、Stage2 exe、WebView2受領 | selected Builder/actual Child、入力bytes、artifact、Stage1実再入・Stage2実GUIを**同一Build Attempt**で結線。CI外の生成を隠さない |
| **DH-07 外観/IME** | 外観切替、入力途中IME、狭い幅、拡大、再起動 | 編集内容・Focus/Pin維持、Themeだけの変更でHost呼出し0、操作可能なControlとラベル/読み上げを確認 |
| **DH-08 Disclosure/Resource** | 別Root同名Evidence、symlink/Path再束縛、読取り後Grant撤回、WS/AI配送 | Host-bound Readerと公開結果の別資格。生Path/secret/隠れた件数を漏らさず、現在のGrantが不明なら未受理 |

すべて、独立Expected・Known Mutant・actual Producer/Consumer・OS/Arch・exact HEAD・実Binary SHA・Scenario・未実施理由を#1180/#2514既存Packetで記録する。新Runner／別Scenario ID体系を正式正本にしない。DH番号は本節の検収索引に限る。

### 11.5 性能・UXと実測選定

一つの固定性能値を設計段階で採用せず、まず**同じOperation・実Source・実GUI**でA/B0/B1/B2を比較する。cold/warm起動、キーストローク→画面反映、IME確定、モデル変更→診断、Preview、生成、Host Response→描画、p50/p95/p99、CPU/RSS、Worker再起動、Artifact数と配布byte、障害時の未保存内容保持を収集する。

IPC microbenchmarkの低遅延だけでB方式を採用しない。Process分離で保証できる障害隔離・権限・運用上の利点と、増加する複雑性を同じ検収票に示す。利点がなければAを維持する。Web Shared Server/Go-WASM pure read-only Projectionの比較はP2以降で独立に行う。

### 11.6 既存Ownerへの工程引渡し

| 順 | 実装・判断Owner | 目標と停止条件 |
| --- | --- | --- |
| **P0-1** | #1824／#3421 | A方式のWindows実WebView2→Wails実RPC→Source-backed Catalog/Inspect。後着/Source資格を独立観測 |
| **P0-2** | #2831／#4583／#1080／#1097 | non-Plugin Studioを正式Build Subjectにし、Builder/ArtifactSet/N0〜N3を同一Attemptで実証。#4634と#4587の別HEAD合格を足さない |
| **P0-3** | #2831／#4583 | 生成Stage1自身→Stage2 Desktop生成→当該Stage2 Windows WebView2使用の独立Exit |
| **P1** | #1824／#1124／#4611／#4612／#2621 | 編集/三視点/SHO/外観の実ConsumerとA/B方式比較。安全・障害・性能を実測 |
| **P1** | #4376〜#4380／#2550／#3394／#2516 | Plugin管理とREA/DB/外部EvidenceのHost/current disclosure/Source保持を各Ownerで検収 |
| **P2+** | #3421／#2391／#1251 | 共有WebのActor/WS分離、任意外部UI制御の本番物理除去、pure Go/WASM Projection比較 |

**文書のみの追補**であり、ここに列挙した実Windows試験、四方式の比較、最新HEADのCI、non-Plugin全Build、Stage1→Stage2、Owner受理を新たに実施したものではない。#4583のP0順位、ADR-0048のTransport分離、Studioと生成業務Desktop Targetの別資格を維持する。


---

## 12. 同一Sourceから「編集・実データ・生成・実画面」へつなぐ最小製品一巡（2026-11-14）

<!-- tei-desktop-vertical-witness-20261114 -->

§11の方式選択・検収票に、[MVP Dogfood Draft #4651](https://github.com/hat47x/tei-platform/pull/4651)のDG-0〜4、[Subject Pin #4515](https://github.com/hat47x/tei-platform/issues/4515)、[Function実Query #2586](https://github.com/hat47x/tei-platform/issues/2586)、[Toolchain実観測 #4648](https://github.com/hat47x/tei-platform/pull/4648)、[段階別Plugin Contribution #4646](https://github.com/hat47x/tei-platform/pull/4646)、[DSL診断四層 Draft #4652](https://github.com/hat47x/tei-platform/pull/4652)を接続する。**実装する機能の数ではなく、使用したSource／Builder／Provider／結果が実際にどこまで届いたか**を優先して検証する。

### 12.1 二つの一巡を分け、同じSourceで接続する

| 一巡 | 操作の流れ | 最初の受入・非受入 |
| --- | --- | --- |
| **製品を作る一巡** | 受理済みSource／選択済みGenerator → 正式non-Plugin Studio BuildIntent → N0/N1/N2/N3実Invoker → Stage1実Process → Stage2 Artifact → **当該Stage2実Windows WebView2** | P0最優先。各工程のselected inputとactual read / artifact / actual Consumerを記録。単なるBuilderの事前pinと、OS loaderが読んだ全byteは別 |
| **製品を使う一巡** | Workspace Open → Source-backed Catalog/Inspect → Subject Pin（明示操作）→ 診断・参照探索 → 必要な場合だけData Context実Read → 候補Preview／独立Expected → 人間による受理／適格なInvoke | Pinと参照は読み取りの独立lane。DB・Authoring・AI協業をStage1→Stage2の追加Hard Gateにしない。Pinの実装未接続なら既存Inspectのまま正確に未対応と示す |

両者は、同じSubject・SourceRevision／Provider Revision・Workspaceに紐づく**既存Operationの観測**で結ぶ。新しい「製品の正本」や中央Evidence DBは作らない。Document Writerの保存、TEI Meaning Acceptance、Build Artifact、Host Effect、Consumer DeliveryはそれぞれのOwnerが保持する。

### 12.2 Subject Pinは「参照意図」であり、現在のSource資格でもAI送出権限でもない

[Issue #4515](https://github.com/hat47x/tei-platform/issues/4515)の現在のWorkbenchでは、`selected: Subject | null`の一時選択とUI応答世代制御が存在するが、**明示Pin/Unpin・Task Shelfは未実装**。したがって、画面に選択されたものを「固定済み」「実Sourceに再束縛済み」と呼ばない。

第一実装は**SubjectのWorkspace/kind/idを明示的に固定**するread-only UIのみとし、既存`readInspection`／`InspectSubject`を再利用する。FocusやGraph選択の変化でPin Intentを変更しない。Source/Workspace/Providerが変化したらIntentを残して再適格化を要求する。権限／Source現在性の情報が無ければunknownとする。Undoによって戻るのはローカルPin Intentだけであり、Source Acceptanceや以前のGrantは復帰させない。

**描画用Relation edge `parent:${subjectKey(child)}`を正式Relation occurrence IDへ昇格しない。** 多重辺・向き・意味属性・Source identityに対応したRelation Pinは#1124/#4472の後続とし、現時点ではunsupportedと明示する。GraphでNodeが選べることはRelation編集資格の証明ではない。

### 12.3 実データの表示は「取得できた結果」と「業務上の完全な対象集合」を分ける

[main #4539](https://github.com/hat47x/tei-platform/pull/4539)の実PostgreSQL→限定Reference Function Data Contextと、[統合Draft #4644](https://github.com/hat47x/tei-platform/pull/4644)のRA-0/1/2実PG参照Readは、**通常製品CLI／Studio実Consumerの合格ではない**。[Draft #4647](https://github.com/hat47x/tei-platform/pull/4647)は、RLS/Limit/関係不一致等の観測できた制限を`partial`で運ぶ候補であり、native SQLの閉包不確定なら`complete`／`EmptyJustified=true`を主張しない。NULL由来と粒度がunestablishedなら、StudioでEntity完全性や業務0件を推定しない。

最初のDesktop Data Panelでは、**実取得行、取得対象・Snapshot/Scope、unknown/partialの根拠、現在の開示資格**を別表示する。0件は観測したRowsetの状態であって、必要集合の不存在やAction成功ではない。実PGを見せられない状態なら、合成Fixtureの結果と明示し、Product実Readのように表示しない。

### 12.4 Source診断とCapability実行前拒否は四層で検収する

[Draft #4652](https://github.com/hat47x/tei-platform/pull/4652)の責務を、Desktop表示へ次のように接続する。

1. **Source**：原文の構文、複合拡張子`*.tei.md`のLoader/Parser選択、実読取byte／位置。未読込Sourceへ架空のSource Spanを付けない。
2. **Meaning**：Workspace全体での対象解決、Role/Type/Relation/Effect契約。Source Parse成功を意味受理にしない。
3. **Pre-dispatch**：main統合済み#4629/#4636のResult/Input/Effect構造拒否。Adapter呼出し0と、Providerの意味・権限の未受理を区別する。
4. **Entry/Host**：公開された製品Operation/Entry、Hostが選択したProviderの実呼出し、現在GrantとEffect/Receipt。前段の静的成功からEntry公開済みを推定しない。

StudioのProblems Panelには上記の**発生層と、実Source位置が存在する範囲**を表示する。エラー行へのJump/Quick Fixは実Sourceが一意に再解決できた場合のみ提示し、AIがSourceを推測して自動修正・自動Acceptしない。

### 12.5 Pluginを「一つのPackageが全段階で同じ実物」という前提から外す

[Draft #4646](https://github.com/hat47x/tei-platform/pull/4646)で、意味定義／Studio／Generator／Target Runtime／Testの段階ごとにContributionを対応付けるReferenceが作られた。これは将来案であり、Studio Productの正式Plugin Manager/APIに直ちに採用しない。

Desktopではまず**段階別の選択事実を一覧表示できること**を設計要件とする。StudioのInspectionだけ提供するPluginを選んだからといって、生成RuntimeにもPackageが組み込まれたとは扱わない。Editor-only、Runtime未提供、同一意味IDで版が異なる、Target/Scope不一致、宣言のみで実Builderが未選択、といった状態は候補・不足・未受理として区別する。実管理操作Inventory→Plan→Applyは既存#4376〜#4380、正式Provider Invocationは#951/#1251がOwnerである。

### 12.6 Go Toolchainの実同一性はBuilder/Artifactの資格と別票にする

[Draft #4648](https://github.com/hat47x/tei-platform/pull/4648)では、Goの駆動役の種と実コンパイルtoolchainを変えても`l2b.claimed=true`になり得て、生成byteはコンパイルに使ったtoolchainへ依存することをLinux/amd64で観測している。このReferenceを**Windows Desktopの同一入力・同一Artifactの保証へ昇格しない**。

Desktop Buildの報告では、観測できたGo実Toolchain版・digest、Node/esbuild/Wails版、module root/replace、Build Tag、生成JSと埋込みbyte、親子Image digest、実Stage1/Stage2 Processを既存Receipt/Evidenceへ分けて結び付ける。Go toolchain digestを正式Build Planの必要入力に採用するか、Seed Trustをどう受理するかは#2831/#1119のOwner判断であり、本節で新Ruleを採用しない。

### 12.7 AI協業は実Sourceの再解決後に限定し、AI OFFで同じ基本操作を成立させる

AIハーネスは「現在のPinはどのWorkspace・Subjectか」「何が実Sourceから読まれ、何が未取得か」「どのOperationを要求できるか」を、**Hostが現在開示できる範囲の投影**として受け取る。既存CLI/CUIの型付きOperationへ接続し、UIでの見かけのPinを強い権限にしない。Source/Grantが不確実な状態では、候補の文面・独立Expectedの下書きに留め、無人Accept/Invokeをしない。

一方、AI OFFでもWorkspace Open→Source編集/保存→Validate/Inspect→Preview→独立Test→生成が動くことを成立条件とする。SHO/Plugin/DBが未接続・offの場合でも原文編集の最低限の機能を維持する。AIの提案履歴はAccepted Sourceや実行結果の正本ではない。

### 12.8 最短実装順と実機で初めて判定できること

| 順 | 既存Owner／作業 | 最小Exit／停止線 |
| --- | --- | --- |
| **DT-0（P0）** | #4583/#2831、#1080/#1097 | 正式non-Plugin Build Subjectの選択→実Builder→N0/N2/N3→ArtifactSetの同一Attempt。名称だけのRecipe追加やCI script直呼びなら未合格 |
| **DT-1（P0）** | #4583/#1824 | 生成Stage1実Process→Stage2 Desktop生成→そのStage2のWindows WebView2実受領。旧配布bundle・別Attempt結果は不合格 |
| **DT-2（非阻害）** | #4515/#1824 | Node Subject Pinの明示Pin/Unpin、Focus独立、Workspace/Source/ABA失効、旧応答拒否。Relation Pinはunsupported |
| **DT-3（非阻害）** | #2621/#1124/SHO担当 | 管理外Markdown Draftの保存・競合維持と、TEI意味Source受理の別資格。同じEditor Coreを共有し第二Parserなし |
| **DT-4（非阻害）** | #2586/#1251/#2516 | 実PG Readの製品CLI/Studio別Consumer受領、partial/unknown/0件の非誤表示、Grant/DisclosureとEntry/Provider actual use |
| **DT-5（P1）** | #1824/#3421/#2391 | A/B0/B1/B2比較、実HTTP Actor/WS Audience、GUI操作の外部制御、Productionビルドで専用計装の物理除去 |

これらは**新たな一括Hard Gateではない**。最初にDT-0/1を実装する一方、DT-2/3/4は互いの準備を待たず独立実証できる。DG-0〜4（#4651）はDogfood全体の既存索引であり、DT番号はDesktop担当間の読み替えに限る。既存Scenario/Independent Expected/Receiptの第二正本を作らない。

### 12.9 最初の横断負例（単一公開合成Workspaceで実施）

| 対照操作 | 独立Expected |
| --- | --- |
| Workspace AでOrderをPin→Workspace Bへ移動→同じOrder IDをInspect→AのReply後着 | Pin IntentをBへ自動再束縛しない。旧AのDOM/ARIAへの採用0、Host Source/Grantが未確認ならunknown |
| Orderの二つの異なるRelationを同じ端点で描画→`parent:`のedgeクリック | 正式Relation Pin 0。描画Nodeへの一時Focusと意味Relationの選択を混同しない |
| PostgreSQLのnative Queryが0件、RLS/Limitでpartial、結果列のNULL由来が未確定 | 行数0の観測は可、`complete`／`EmptyJustified=true`／entity grain／stored NULLの表示0 |
| `*.tei.md`をSHOで保存→未束縛TEI Parserへ渡す→AIが変更提案 | Markdown原文は保持。Meaning受理0、実Provider/Build実行0、Source位置がない診断へ偽spanを生成0 |
| PluginのStudio Contributionのみ選択→RuntimeなしでGenerateを要求 | Inspector可能性と生成Target必要Contribution欠落を区別。選択していないRuntimeを実包含したと表示0 |
| Toolchain X/Y、選択Builder B／実Child A、Bundle旧版 | 段階ごとのactual bytes/Toolchain/Builderを明示し、不一致なら当該Artifact／Consumer資格FAIL。先行段階の成功は消さない |
| Worker/Transport timeout後にAIが再実行要求 | 既存Request/Receipt lookup-first。確認不能はEffect unknown、無条件の再Submit 0 |

**実施状態**：本節は既存PR・Issue・実測記録の設計上の統合であり、DT-0〜5の新しい実Build・Windows GUI・実PG Product Consumer・AI送出/受理試験を行っていない。採用済み正本、各Ownerと独立Expectedの境界を維持する。
---

## 13. Desktopの操作境界を編集・認可・実消費まで通す（2026-11-14、設計・受入追補）

<!-- tei-desktop-operation-authoring-slice-20261114 -->

§11の配置比較と§12の二つの一巡を、実装時に混同しやすい**画面の操作、保存されるSource、実行されるOperation**の境界で補う。以下は既存の型付きApplication Operation／Source Reader／BuildIntent／Receiptを使うための受入仕様であり、Desktop専用の第二API・Parser・権限Engine・Event Runtimeを採用するものではない。

### 13.1 Wails Bridgeの「登録済み」と「公開してよい」を分ける

現行\`cmd/tei-studio/main.go\`の\`StudioBridge\`には15個の公開メソッドがある。\`reference/tei-studio-workbench/responses.ts\`はUIの後着応答を除外するが、Host権限検査や実処理の取消しを提供しない。現状の登録メソッドをWeb／AIへそのまま公開してよいと解釈しない。

| 実際のBridge入口 | 当面の扱い | Desktop／Webで検収すること |
| --- | --- | --- |
| \`ListWorkspaces\`、\`CatalogWorkspace\`、\`ValidateWorkspace\`、\`InspectSubject\`、\`ObserveXead\`、\`ViewPattern\`、\`ViewApplication\`、\`ViewPackage\` | Source／Workspaceに基づく参照・診断・投影。read-onlyであっても開示可能範囲を伴う | 実Hostが許可したRoot／Subjectだけを返し、異Workspaceの同名IDや古いSourceの結果を現在値と表示しない。結果本文・DOM・ARIA・ログへの開示も別途検査 |
| \`PlanCommand\` | 実行に先立つ候補計画。**計画結果は実行許可ではない** | 選択したCapability、入力、必要Effect、未満足条件を示す。Pre-dispatch拒否時のAdapter呼出し0を確認 |
| \`ExecuteCommand\`と\`WithPeriods\`／\`WithPolicy\`／\`WithReferenceStore\` | Invoke系。現Reference Storeの作用は一時的・非永続であり、本番のDB Commit成功と区別 | プラン時点の資格を流用せず、Invoke時のHost Grant／実Provider／Operation Identity／Effect／Receiptを確認。タイムアウト後はlookup-first、unknownなら自動再実行しない |
| \`OpenShoDocument\`、\`ResumeShoDocument\` | **外部SHO CLIとの連携**。意味受理・保存・実Recipient結果をStudio自身が代行したことにしない | プロセス起動と入出力、Document Writerの保存結果、TEI Sourceへの再束縛を独立記録。DesktopのDocument操作とTEI Meaning Acceptanceを分離 |

本表は**メソッドを操作意図別に整理したもの**で、現行コードの権限実装完了を意味しない。現在のServer PoCのloopback拘束は共有WebのPrincipal認可、HTTP/WSのAudience隔離、実行系入口の安全な公開を証明しない。共有Webの初回公開は参照系からの明示allowlistとし、Invoke／外部プロセス／Source書換えをデフォルト非公開とする（Owner #3421／#2516／#1158）。Local Desktopでも、UIからの文字列や選択表示をHostの権限根拠として使わない。

### 13.2 一つの画面には複数の「版」がある

Studioが表示・編集する値を、次の別々の実物として扱う。

| 対象 | 正本／現在性の見方 | 禁止する推定 |
| --- | --- | --- |
| 編集中のBuffer | ユーザーの未保存入力。IME確定中、取り消し履歴、選択範囲を含む | 編集画面にある＝ディスクに保存済み |
| 保存済みSource | 実ファイルのbyte／既存Source Index、Readerの取得位置 | 保存成功＝Meaning受理済み |
| Meaning／候補差分 | 既存Validation／Shared Acceptanceが対象Revisionについて判定 | Parser成功＝現行Hostで実行可能 |
| Build入力とArtifact | 正式BuildIntent／選択Builder、実読込byte、各生成段階のArtifact、Attempt | Preview成功＝実Stage1→Stage2の生成物 |
| 実行結果／Delivery | 実Provider／Receipt、実WebView2／外部Consumerの受領 | 応答受信＝業務Commit／現在Disclosure成功 |

表示には、観測できた対象・Revision・Scope・欠けている資格を対応付ける。未観測のRevisionやAuthorityを推定で埋めない。Bufferの一時版とSourceの実byte版、意味版とArtifact版に一律の連番・同一ハッシュを要求しない。必要な関係だけ既存Index／Build Evidence／Receiptに結び付ける。

### 13.3 編集・修正・検証の最小遷移

1. **Locate／Inspect**：Source-backedの\`WorkspaceRef\`／Subjectを確定できる範囲だけ表示する。Graphの描画Node・辺IDやMarkdownの見出し名だけで編集対象を確定しない。
2. **Draft／Review**：テキスト編集、Quick Fix、AI候補、SHO編集はBufferへの提案・差分として扱う。各候補は元のSource／Buffer版と対象範囲、変更後の期待を示す。保存／受理／Build／Invokeは連動しない。
3. **Apply／Save**：現在の保存済みbyteと候補の基準版を照合してから最小Patchを適用する。同名別ファイル・A→B→A・外部Editor更新・保存直前のWorkspace切替・symlink等のRoot逸脱は再解決または拒否する。行末、BOM、未変更のMarkdown糖衣表現を不必要に正規化しない。安全な保存方法と衝突判定のOS別受入はDocument Writer／Source Ownerへ委ねる。
4. **Validate／Accept**：保存済みSourceを再読込し、Source・Meaning・Pre-dispatch・Entry/Hostの診断を別々に表示する。Unqualifiedやpartialを\`ok\`へ丸めない。意味受理には既存Ownerの明示的な判断が必要。
5. **Build／Invoke／Observe**：対象と入力Revisionを再確定して既存Operationを呼ぶ。Builderが実読込した入力とWebView2が実消費したAsset、Invoke後のReceipt／独立再読込みをそれぞれ観測する。

現行Wails Bridgeに一般的な\`SaveSource\`等の入口があるとは扱わない。未実装の段階は「未対応」と表示し、WebView側だけで擬似的な保存成功を成立させない。SHOの\`OpenShoDocument\`もStudio共通のSource書換えAPIへ読み替えない。UIが取得した\`sourceSpan\`が現在Revisionへ再解決できない場合、Jumpは保留し、Quick Fixの自動適用も行わない。

### 13.4 Pin／応答順序／操作の実行状態を独立させる

\`selected\`（今見ている対象）、明示\`Pin\`（参照し続けたい対象）、Dirty Buffer、Host実処理、Task／AIへのHandoffは別々の状態である。Pin AをBの閲覧中に裏で再Inspectしても、BのEditor／Graph／Dirty／Selectionを変更しない。Undoが戻すのはBufferやPin Intentのローカル状態だけであり、過去のGrant・Source資格や実行済みEffectではない。

\`responseScope\`のlane／epochによる後着除外は、**表示の現在性**だけを担う。Workspace、Source byte、対象Identity、Plugin／Provider選択、Panel破棄、実Host Grantが変わった場合は、その応答を現在の説明・選択資格として再利用できるか各Ownerに確認する。UI上でAbort／閉じるを行ってもHostの副作用が取り消されたとは表示しない。Effectの状態が不明ならReceiptを先に照会し、重複Submitを避ける。

### 13.5 プロセス起動から終了までを同じOperation観測でつなぐ

Desktopの基準方式Aは、**OS起動 → Wails／Goサービス初期化 → Window／WebView2接続 → UI Event → 型付きApplication Operation → 必要なResponse／Effect → Window破棄／終了**の一巡として捉える。Editor・タブ・Windowが閉じても、Go側実行の成否を画面の消滅から推定しない。長時間操作、異常終了、再起動後のReceipt／Artifact／Draftの再読込は既存Ownerの実状態で判定する。

B0同一exe子Worker、B1別Server、B2別Server/Workerはこの同じOperation／Result／実消費を維持してP1に比較する。親子ProcessのIPC（Named Pipe／UDS候補）とWebView2↔GoのWails Bridgeは異なる通信境界であり、一つの「IPC検証済み」にまとめない。Worker再起動でMeaning／Grant／Effectを再利用できるとは推定しない。

### 13.6 実装に引き渡す独立負例と優先順位

新しいScenario正本は作らず、次の対照を#1180／#2514へ既存DH／DT／PIN／UX受入の補助Expectedとして引き渡す。

| 優先／Owner | 対照操作 | 観測すべき失敗の防止 |
| --- | --- | --- |
| **P0 #4583／#2831** | Buffer R1・保存済みSource R0のままPreview→正式Build→Stage2実起動 | Buildが実読込した版を記録し、R1を実Artifactに含めたと誤表示しない。R0/R1混在を別AttemptのPASSで隠さない |
| **P0 #1824／#4515** | BのDirty Buffer閲覧中、Pin Aの裏Inspectが後着。さらにWorkspace A→B→A | BのEditor／Graph／Dirty不変。旧Aを現在資格へ自動昇格しない。旧ReplyをDOM／ARIAに採用しない |
| **並行 #1124／#2621／SHO** | CRLF＋BOMを含む\`*.tei.md\`の一部分だけ修正し、外部Editorが同名ファイルを差替え | 未変更byte／糖衣形式を保持。競合・Root逸脱を拒否し、保存とMeaning受理は別票 |
| **並行 #1158／#2516／#3421** | Server／AIから\`ExecuteCommand\`系や外部SHO連携を直接要求 | 明示許可前は公開しない。Hostで拒否し実Provider／外部Process呼出し0、開示対象外の本文／DOM／ログ送出0 |
| **並行 #1174／#1251** | Invoke中にWebView2／Workerが終了、Responseなし、再操作 | Receipt先行照会。unknown中に無条件の再作用0。Reference Storeの一時作用を永続Commitと呼ばない |
| **P1 #1824／#2391** | A/B0/B1/B2の同一Fixtureでcold/warm・p50/p95・memory・障害復旧を比較 | 成功したTransportだけで製品同値を主張せず、Productionに外部GUI計装が残っていないことを実バイナリで確認 |

最初の実装着手は**P0の正式non-Plugin Studio BuildIntent→Stage1→Stage2→当該Windows WebView2**と、これを妨げない**現行WorkbenchのPin／後着独立性**である。一般Source保存、Web公開、Worker導入、SHO/AI完全統合をP0の追加Hard Gateにしない。実際のOwnerが選ぶ実装方法・既存Scenario ID・受理の所在をこの文書が上書きするものではない。

**現状**：Bridge／Workbench実コードと既存Issueを照合して設計を補ったものであり、本節の新たなWindows／IME／実Web認可／OS保存／Stage2／Host Effect試験は **NOT RUN**。文書の更新やPR CIの成功を製品受理として扱わない。

---

## 14. 標準データ構造からSourceへ戻れるDesktop投影と編集（2026-11-14、設計・受入追補）

<!-- tei-desktop-source-projection-roundtrip-20261114 -->

§13で分離したBuffer／Source／Meaning／Build／Effectを、実際のStudio画面で使えるようにするための追補。自己記述RoleとDSL構文拡張（#2620／#988）、Markdown Authoring（#2621）、意味を保つ投影と逆写像（#2497）、統合Modeler（#1124）、Design Contextの取得範囲（#3262）、Graphの意味Identity（#1120／#1126）を横断する。**新しい画面用意味モデルやParserを設けるのではなく、既存の意味構造を読める表現と安全な編集操作へつなぐ。**

### 14.1 画面を「意味の正本」にせず、対象の同一性を保つ

TEI Studioでは、同じ定義をSource／Property／Table／Tree／Graph／式／設計図として見せられることを目指す。ただし、各画面の行番号、図形の描画キー、ラベル、配置座標、表の列順は**意味上のIdentityではない**。少なくとも次の関係を維持する。

- **対象**：Workspaceと\`kind/id\`、必要なScope・Source/Meaning Revision、Sourceに戻れる場合の実Locator。表示用の別名や人間向け見出しから自動生成した仮IDは、その対象の正本にならない。
- **関係**：source／targetのNode Identityと、関係そのもののOccurrence Identity・向き・多重性・属性を区別する。同じ端点に同じ名前の関係が二本ある場合、Graph描画のために一つへ縮約しない。
- **出所と取得範囲**：どのSourceを実際に読み、どのRole／Plugin／Operatorで解釈し、対象のうち何が未読込・partial・unknownなのかを保持する。Sourceの記述位置が移動してもIdentityを文字位置へ置き換えない。
- **投影の性質**：並び順、NULL、時点、有効期間、単位、粒度、集約、多対多関係など、その表示で失う情報を明示できるようにする。損失のある投影から自動的に元データを編集できるとは扱わない。

現行Workbenchの\`relatedGraph()\`が生成する\`parent:\` edge IDは表示専用であり、正式Relation occurrence IDが得られるまでRelation Pin・Relation編集は**unsupported**とする。NodeのInspectができても、異なる意味関係の編集を許可する根拠にはならない。

### 14.2 構文・意味Role・表示を三つの独立した層として扱う

| 層 | Desktopに必要な機能 | 未成立時に許さないこと |
| --- | --- | --- |
| **原文と構文** | Markdown表・式・Graphviz等の糖衣表現や独立DSLを原文のまま保持し、実際に選ばれたLanguage Plugin／Parser、位置、診断へ結び付ける | 見た目が表／コードブロックという理由だけで意味を決めたり実行したりしない |
| **標準構造と意味Role** | 共通のRecord／Sequence／Relation／Graph／Expression等をRole・Operator・型・適用条件に解決。Role重複・未定義・Source競合を既存Validatorへ渡す | Role付与＝Owner受理、Parser選択＝実Provider起動許可、構造一致＝意味一致とはみなさない |
| **表示と操作** | 一つの意味対象を複数の表示に投影し、選択・Sourceへ移動・診断・変更候補を結ぶ。表示方式は用途に応じて選択できる | 投影結果／視覚的な操作を第二Canonicalや第二Authorityにしない |

この分離によって、JSON Schema等の機械向け定義も、利用者には図・表・説明・入力支援として提示できる。画面には必要な項目を先に出し、詳細な制約・型／出典・未解決の依存をジャンプやツールチップで参照する。ただし、**人間向けの簡潔な表示を作るために必要な契約を省略するのではない**。DSLの本文にGoの\`struct\`／\`func\`等の実装形を露出させない。

Studio標準Rendererが知らないRoleやPluginでも、原文表示・一般的な表/Tree/Graph投影・Source位置・既存診断は可能な範囲で提供し、不足している専用意味表示を隠して成功扱いにしない。Reader未束縛の管理対象\`*.tei.md\`を、ただのMarkdownとして開けたことをTEI Source受理と表示しない。

### 14.3 「表示できる」と「その表示から編集できる」は別の能力

最初の段階では、どの表示もread-onlyで使えることを優先する。編集操作は、**対象Identity、現在のSource Revision、一意な元位置、表示からSourceへ戻る写像、必要な入力制約**が確認できた範囲に限る。

- **編集可能候補**：元の定義に一意に対応するProperty値やTableセルなど。編集前に差分と対象Sourceを提示し、適用直前に基準Revisionを再確認する。
- **表示専用**：集計表、部分Query、Graph→Tree、複数のSourceを併合した一覧、複数の実Relationを一つに見せた図、現時点で意味IDがない描画Edge。ユーザーが編集したい場合は、元Sourceへ移動するか、明示的に編集可能な対象を選び直す。
- **説明と入力支援**：補完、候補、AI修正案、Diagramのレイアウト変更は、実Sourceの変更と区別する。表示位置だけの変更は意味変更ではない。逆に、構造が変わったときにSourceとMeaningの変化を可視化する。

Sourceへの反映は、**Preview → Sourceとの差分 → 明示Apply → 実保存 → 別Readerで再取得 → Validate／再投影**を基本とする。編集前の表示と再投影後のMeaningが一致するかは、#2497のGetPut／PutGet等の保存法則を必要な箇所に限って利用する。法則の一般的成立を宣言するのではなく、適用できない表示を編集不可として扱う。No-opの編集では、Markdownの本文・コメント・コードフェンス・行末・BOM・糖衣表記の元byteを不必要に変えない。Sourceが同名で差替わった場合やA→B→Aのとき、ハッシュや位置の偶然の再一致だけで旧編集資格を復活させない。

### 14.4 発見からSource・診断へ戻る操作を一つの経路にする

最初のユーザー操作を、**対象の検索／選択 → 定義と関連構造の表示 → 正本Sourceへ移動 → 診断または変更候補の確認**とする。これをCLIの\`catalog/inspect/trace/source-index/validate\`等の既存Operationと対応付け、Studioだけが独自の意味Resolverを持つ構成にしない。

UIの状態は以下を別表示する。

1. **見つけた対象**：現在のWorkspaceで一意に解決できたか、候補・曖昧・not-foundか。
2. **読めたSource**：どの位置・Revisionを実際に取得したか。取得していないものを「存在しない」と表示しない。
3. **意味上の関係**：その投影で保持したIdentity・向き・多重性と、未解決・省略した関係の範囲。
4. **編集資格**：read-only／可逆な候補／適用前再照合が必要／unsupported。
5. **検証結果**：Source Parse、Meaning、実Consumer、Owner Acceptanceのどれを確認した結果か。

この5区分は新しい公開Status Enumの提案ではない。現在存在するResponse／Source Status／既存診断の組合せを、人間が誤読しにくい形で表示するための**UI設計上の区分**である。対象内の参照数0、SQL取得行0、Graph表示辺0を、Workspace／Requirement全体の完全な不存在へ昇格しない。未取得の必要集合や権限で見えない情報については、#2516のDisclosureも確認する。

### 14.5 初回Fixtureと実装順（他のP0を阻害しない）

公開可能な一つの合成Workspaceを用い、同じSubject・Source RevisionをCUIとStudioで照合する。現行\`relatedGraph\`のNode-only選択と、正式なRelation編集が未実装である事実を前提にする。

| 独立Expected候補 | 入力／操作 | 必要な結果 |
| --- | --- | --- |
| **SPV-01 対象同一性** | 同じ表示名のNode A/BをGraphと一覧に表示して別々にInspect | 表示名だけで合体せず、Source-backedの\`kind/id\`／Workspaceを正しく保つ。誤ジャンプ0 |
| **SPV-02 多重辺** | A→Bに異なる意味Relation E1/E2、逆向きB→Aを用意 | 正式Occurrence Identity／向きが観測できないうちは描画Edgeを編集・Pin不可とし、関係省略を「無関係」と推定しない |
| **SPV-03 Roleの曖昧さ** | 同形のMarkdown表を説明・Function・Testに配置し、Role/Parser未選択も作る | 既存Sourceで確証できるRoleのみ提示。不明なら候補／診断に留め、誤った自動実行0 |
| **SPV-04 原文保持** | BOM/CRLF/コメント/Graphvizを含むSourceでNo-opと単一セル修正 | No-opは元byte不変。編集許可がある箇所のみ局所Patchし、別Readerで再Parse／再投影を照合。未対応なら安全に拒否 |
| **SPV-05 非可逆View** | 多対多GraphをTree、複数行を集計表へ投影し、その表示から保存要求 | 表示に現れない親／行／重複を暗黙削除せず、直接書戻し0。元対象へのNavigateは可能な範囲で維持 |
| **SPV-06 現在性** | 同じID・spanでSource A→B→A、Workspace切替、旧Inspect後着 | 旧資格自動復活0。Pin Intentと現在の表示資格を別表示、旧Replyの採用0 |
| **SPV-07 不完全な取得** | Scope外依存、未読込Role、取得0件、partial Graph | 「見えている範囲の0件」を未使用／不存在／全部読取済みへ変換しない。非公開部分の存在・件数・pathを漏らさない |
| **SPV-08 Consumer分離** | 同じSubjectをCLI／StudioのTable・Graph・Source表示で観測 | Identity／Meaning／根拠は比較可能。CLI成功をUI操作・OS WebView2・保存成功・実EffectのPASSにしない |

**順序**：P0セルフビルド（#4583/#2831）と並行して、(A)既存Source-backed Catalog/Inspectを使う**read-only Sourceジャンプ**、(B)同名Node＋多重辺の表示と未対応操作の明示、(C)Roleを付けたProperty/Table/Treeの汎用表示、(D)可逆な一つのセル編集と保存前後の独立再読込、(E)SHO／AI／Graph意味Relation／Plugin拡張へ進める。A/Bの検証結果を待たずにP0を止めない。編集逆写像や全Graph表示のための別Parser・別Meaning／Writer／UI Editor Coreは作らない。

**Owner**：#1124（表示とユーザー操作）、#2497（投影・編集逆写像の保存法則）、#2621（Markdown／Source Writer）、#2620/#988（Role・Language Plugin）、#3262（Required／取得範囲）、#4515（Pinと現在性）、#1180/#2514（独立Expected）、#2516（開示）、#1824（実Workbench）、#2831/#4583（P0 Build）。新しいSPV番号は**本設計書内の引渡し用**であり、Scenario／テスト定義の第二正本ではない。

**実施状態**：本節は既存Issueおよび実Source構造に整合させた設計追補。新Graph UI、Markdown Parser、Source保存、可逆編集、実Windows WebView2、CLI↔Studio E2E、Owner意味受理は**NOT RUN**。実績として扱うには別途exact HEAD／実Consumer／独立Expectedが必要。

---

## 15. Desktopの起動・終了・再開を、保存済みの事実と画面の記憶に分ける（2026-11-14、設計・受入追補）

<!-- tei-desktop-lifecycle-recovery-witness-20261114 -->

§3.2の復旧原則、§8〜9のWindow/Component寿命、§13のBuffer・Source・Effect分離を、実Wails/Desktopで確かめられる**操作の時間軸**に接続する。最初の製品はGo＋Wails/WebViewの同一プロセス方式Aを基準とし、B0子Worker／B1 Server／B2別Workerはその比較対象に留める。プロセス管理・Draft復旧・副作用確定の新しい共通Engineを構築する提案ではない。

### 15.1 現行コードで確認したことと、まだ保証できないこと

現行mainの\`reference/tei-studio-workbench/main.ts\`では、\`pagehide\`時に\`disposed=true\`、\`responses.dispose()\`、Workspace／Inspect世代の更新、Command Panel／Graph／Editorの破棄を行う。この処理は**旧画面で後着Replyを採用しないためのUI側後始末**である。保存済みDraftの復元、実Go Operationの停止、既に送信されたEffectの取消し、Window再起動後のHost権限再取得までは保証しない。

現行のWorkbench Editorは\`createReadOnlyTextEditor\`を利用し、Wails Bridgeに汎用\`SaveSource\`入口はない。したがって、現行read-only StudioにDirty Draft保存や終了確認が実装済みとは表示しない。外部SHO CLIへの\`OpenShoDocument/ResumeShoDocument\`も、StudioのEditor Session保存やTEI意味受理の代替ではない。Wails \`Run()\`の終了コード、\`pagehide\`発火、ユーザーがWindowを閉じた事実を相互に同一視しない。

### 15.2 四つの寿命と、終了・再開時に持ち越す資格

| 対象 | 寿命／Owner | 終了・再開時の判断 |
| --- | --- | --- |
| **画面の記憶**（タブ、Focus、未保存Buffer、Pin Intent、Panel状態） | Window／UI Session。現行の\`responseScope\`は当該画面のReply順序だけを管理 | 再起動で消える可能性がある。復元できても対象の現在資格・意味受理・Host Grantは復元されない |
| **保存済み原文**（TEI Source／SHO Document／許可されたDraft） | 各Source/Document Writerが管理。保存前にRoot・基準byteを確認 | 再起動後は実Sourceを再Read。Draftが存在するなら出所／機密区分／基準版を確認して候補表示し、自動適用・自動Acceptしない |
| **実行中のOperation**（CLI／Builder／Provider／外部SHO） | 既存Application Operationと実Host／Process／Effect Owner | Window消滅・Reload・Cancel UIと実処理停止を区別。実行が続いたか、終了したか、結果不明かは既存Attempt／Receipt／Provider観測で判断 |
| **配布された実物**（Go exe、埋込みJS、Wails Runtime、Plugin選択） | BuildIntent／ArtifactSet、配布先OS、実Loader | Window再生成で別版Assetを読んだ可能性がある。実Consumerが用いたbyte／版とSource/Builderの選択を別票で確認。旧版の表示記憶を新版の証明にしない |

「画面を復元できる」は「元の編集対象を現在も操作できる」と異なる。「実行できた」は「画面で結果を受け取れた」と異なる。説明文・通知・ログは、この区別に従って出す。

### 15.3 起動・終了・再開の操作順

**起動**：実exe/OS/Wails Runtimeと埋込みAssetを観測 → Hostが許可したWorkspace Rootを確定 → 現在のSourceを再読込 → WindowとWebViewを接続 → Catalog/Inspectの実Replyを画面に表示する。Workspaceの記憶や前回のPinは、Hostが現在のSource／開示資格を確認する前に「再接続済み」と見せない。複数Window・二重起動を許可するかは実要件が生じてから選定し、ローカルUIだけの排他をSource書換えの排他保証に使わない。

**通常終了**：読み取り専用の現段階では既存\`pagehide\` cleanupで登録解除・破棄を確認する。書込みEditorを実装した後は、未保存Bufferの有無、保存可能なDraftの状態、実行中Operationをそれぞれ表示し、可能な範囲で保存／中止／終了継続の選択を明示する。OS強制終了やクラッシュでは確認ダイアログもcleanupも必ず実行できるとは限らない。したがって\`pagehide\`に永続保存や重要な効果の取消しを依存させない。

**再開**：実Sourceと現在のWorkspace Rootを再解決 → 安全に保管されたDraftがあれば**復元候補**として別表示 → 旧BufferとSourceの差分・版競合を確認 → 必要なら明示的にReview/Apply。Pin Intentは復元してもよいが、新たにInspectしてSource／Grantを確認できるまで現在適格化しない。旧Preview・AI提案・Build成功・Provider成功を画面キャッシュから復活させない。

**通信断・応答喪失**：UI側の取消しや\`responses.dispose()\`は**描画への採用停止**であって、Hostの実行取消しではない。すでに送出されたEffectの状況はOperation Identityを用いて既存Receiptをlookup-firstで確認し、\`unknown\`なら自動再Submitしない。参照系の安全な再取得と、書込みを伴う操作の再試行を同じ挙動にしない（#1174／#1129）。

### 15.4 保存しないもの／独立して確認するもの

初期実装では、永続化対象をむやみに増やさない。画面配置・Theme・Pinの**選択意図**などはUI設定の候補になり得るが、Credential／Grant／Source本文を無条件に永続化しない。Draft保存が未実装なら、それを明示する。セキュアな保存先・所有者・削除／再開仕様が確認できるまでは、便宜的な平文自動保存を導入しない。

保存済みDraftを復元できても、外部SHOの保存・TEI Source Loaderの受理・意味のOwner受理・Source Index・Builder入力の実消費・実DB Effectを自動的に成立させない。AI OFFでも、実装済みの参照・原文操作を利用できることを優先する。Plugin／Runtime／UI Kitを更新した場合も、旧画面に残っていたFunction名／Provider選択を実装済みのまま復元せず、現在の選択事実と適合を再照合する。

### 15.5 性能と障害耐性は一つの「Desktopは高速」という数字にしない

最初の測定は、同じ公開合成Workspace・Source Revision・OS環境・起動方法を使い、**起動 → Catalog/Inspect → UI表示**までの区間を分ける。

| 測る区間 | 記録するもの | 誤って主張しないこと |
| --- | --- | --- |
| OSによる実exe起動 → Go Host初期化 | cold/warm、OS/Arch/Process、Go・Wails版、RSS/CPU | 「画面を使える状態になった」とは別 |
| WebView2起動 → 実Bridge接続 | 実Asset digest、WebView2版、Wails RPCの開始/終了 | HTML配信200を操作成功へ代用しない |
| Source Index／Host Inspect → WebView表示 | Required/Loaded/Partial、実Response・到達時刻、DOM/ARIA表示、p50/p95 | UI表示の速さをSource完全性／Disclosureへ代用しない |
| Input → Feedback／診断／再投影 | キー入力・IME確定、巨大モデルの再描画、メモリ伸長 | チャート表示成功を意味のEditor適用・保存成功へ代用しない |
| Window Close／再開・異常終了後 | Resource／Handler解放、旧Reply採用数、実Source再Read、未確定Operationの照会 | Window消滅をEffect取消し、復帰を前回状態の権限維持としない |

p50/p95/p99の採用や仮想化／遅延読込は、まず実測で遅い区間を特定してから選ぶ。画面上のモデル件数／図形数、Source Indexの構築、Provider Read、描画を切り分け、早い段階で一律の性能閾値や第二Indexを固定しない。A方式とB0/B1/B2の比較は、同じOperation・実Consumer／障害条件で行い、通信方式の違いに意味実装を重複させない。

### 15.6 製品受入への引渡し（既存DS/DT/UX・Scenarioを使用）

| 受入補助ID | 操作と一因子反例 | 観測するExpected／責任分界 |
| --- | --- | --- |
| **DL-01 実起動** | 許可Workspaceで実exe起動 → 実Window/WebView2 → Host Catalog/Inspect | 画面に実Replyを受領。静的Asset閲覧・Go直呼びだけでは不合格。#1824 |
| **DL-02 画面破棄** | Inspectを開始 → \`pagehide\`／Windowを閉じる → 旧Reply到着 | 旧画面への採用0。Host Command取消し・Source変更は推定しない。#1824／#4515 |
| **DL-03 再開と同名Workspace** | AでPin／Draft候補 → 閉じる → 同名IDの別Root Bで再起動 | 元AをBへ自動束縛0。Draftを自動Apply/Acceptせず、現在Grant不明はunknown。#4515／#2621／#2516 |
| **DL-04 強制終了** | 未保存Bufferを含む将来の書込みEditorを強制終了 | \`pagehide\`や確認ダイアログを必須前提としない。復旧対象の有無を正確に表示。保存未実装なら保証しない。#1124／#2621 |
| **DL-05 作用の不明状態** | 実行送信直後に応答喪失、Window／Worker終了 → 再開 | 同一OperationのReceipt照会が先。unknownから自動重複Submit 0。#1174／#1129 |
| **DL-06 Asset版変更** | 起動中のUI Kit/Plugin/生成Assetを更新し再起動 | 実消費Asset／選択Pluginを別記録。旧Preview/Pin/Grantを自動昇格しない。#1824／#2831／#4583 |
| **DL-07 OS・性能区間** | cold/warm、通常終了・強制終了、IME入力、要素多数／少数 | Host／Bridge／Index／描画の各時間・RSS/CPUを分けて観測。しきい値は測定後に決める。#1824／#2391 |

**最短順**：まず既存のread-only WorkbenchでDL-01/02を実Windows Window＋Source-backed Hostとつなぎ、現行cleanupの資格を実測する。DL-03〜06は書込みEditor／Draft／実Effectが実装される段階で、それぞれのOwnerに引き継ぐ。DL-07は方式Aの基準値を取り、Worker/Serverを採用する根拠が生じたときに比較を拡張する。P0の正式non-Plugin BuildIntent→Stage1→Stage2／当該Windows WebView2実使用を新しい復旧基盤の完成待ちにしない。

**実施状態**：現行コードの静的読取と既存Issueの責務を照合した**設計追補**である。新たな実Windows起動／強制終了／復旧／機密Draft保管／Performance E2E・Owner受理は**NOT RUN**。DL番号は第二のScenario正本ではなく#1180／#2514・既存DS/DTへの引渡し用である。

---

## 16. CLIから育てる実質的セルフビルドとDesktopへの展開（2026-11-14、設計・受入追補）

<!-- tei-desktop-substantive-selfbuild-cli-first-20261114 -->

他チャットで整理した「Studio GUIは初期のセルフビルドの入口として必須ではない」「\`*.tei.md\`がGoコードを単に呼び分けるだけではTEIによる実装とは言い難い」「Plugin自身も可能な範囲でTEI定義から再構築する」「導入が進むほど定義を簡潔にできるか検証する」という目標を、既存#2831／#4583／#2620／#2621／#988／#951の作業へ接続する。Studioの製品目標を取り下げるのではなく、**CLIでの反復が最短で進む工程と、実Windows Desktopでしか立証できない工程を区別する**。

### 16.1 「自分をビルドできる」と「自分の実装を記述できる」は別の軸

| 軸 | 検証したいこと | 合格を推測してはいけない証拠 |
| --- | --- | --- |
| **Build再入** | TEIが選択したBuildIntent／実BuilderからStage1を構築し、実Stage1 ProcessがStage2の実生成を指示する | CIが\`go build\`やshellを二回呼んだだけ、Stage1/2のdigestが一致しただけ |
| **定義による実装** | 業務・UI・Operation・検証・生成の少なくとも一つの**実際の振る舞い**がTEIで解釈されたSourceから生成される | \`*.tei.md\`がGo関数名だけを指す、任意Goコードを文字列のまま埋め込む、生成器が外部の固定Sourceを丸写しする |
| **Plugin自身の再構築** | Language／Operator／Generator／UI／Test等のPlugin Contributionについて、どの部分がTEI定義から生成され、どの部分がBootstrap/外部Adapterか実物で説明できる | Pluginがカタログに登録された、PluginのBuildだけTEIから起動した、Editor向けContributionがあるだけ |
| **定義の簡素化** | 同等以上の振る舞いと独立Expectedを維持しながら、手書き実装・重複した設定・認知上の記述負担を段階的に減らせる | 見かけの行数だけ短い、契約や条件を暗黙のGo側に押し込めただけ |

これらを一つの「セルフビルド達成率」に足し合わせない。最初の実Stage2 Desktop成功は**製品への到達**を示すが、それだけでStudio全部の意味がTEIから生成されているとはいえない。他方、CLI上で\`*.tei.md\`から一つの意味ある部品を生成できても、製品のWindows Wails/WebView2で実使用された証拠にはならない。

### 16.2 初期の開発はCLI主導とし、MCP／Studioは同じ操作を利用する

基本経路は **定義Source → 既存Loader／Role／Meaning解決 → BuildIntent／Plan → Generator → Go/TS等のTarget Source → 実Builder → 独立テスト／実Consumer** とする。操作の最初のConsumerはTEI CLIでよい。生成AIや外部HarnessはCLIを直接利用して構わず、MCP接続が利用できるときは既存の型付きOperationへ対応付ける。MCP専用の第二Build Engineや認可経路は作らない。

Source形式については、現行の\`*.tei.md\`が正式TEI Readerとして無条件に読み込めるわけではない（#2621のLoader/Parser負例）。そのため、**形式が先行した段階**と**実Loaderが正しく受領する段階**を別の検収票にする。既存\`*.tei\`／Contract／Transformから局所生成を先行することはできるが、それを「\`*.tei.md\`から実生成した」と表示しない。管理対象\`*.tei.md\`が正式なBuild入力になるのは、Reader選択、Source Span、Role/Meaning、実Builder消費、変換の根拠を確認してからとする。

GUIはその後、CLIのResult／Source Position／Build Evidenceを同じ型付きOperationで投影する。UIの図解や入力支援は実作業を短縮するための機能であり、CLIでの実構築や独立Expectedの代わりではない。

### 16.3 意味あるTEI実装かを反証する最小の検査

「TEIで実装した」と数える最初の対象は、大規模なStudio全体ではなく、**既存Studioに実際に使われる一部品**とする。候補はInspectの表示に用いるProjection、型付きOperationへの入力束縛、Source-backedなGraph／Property表示、生成器の一つの決定論的Transform、またはPluginが提供する構造的検証である。既存の\`inspection-reader\`生成Referenceは再利用可能だが、現時点ではContract JSON／Transformと実TS生成の限定証拠であり、\`*.tei.md\`の意味実装そのものとは別に扱う。

1. **独立した振る舞いの仕様**：定義で表したルール・型・関係・入力／出力・エラー・未対応を、実装コードから写さないExpectedとして固定する。
2. **生成差分の実証**：TEI Sourceの意味ある一要素を変更すると、既存Generatorが生成した該当Go/TS等の構造・実動作が意図どおり変化する。コメントやファイル名だけの差分では不十分。
3. **誤実装の拒否**：TEI側から必須条件を削る／参照先を入れ替える／関係の多重度を変える等の反例を与え、独立Expectedが失敗を検出する。固定Go実装へ丸投げしていると検査で露見する構成とする。
4. **実消費の確認**：生成ファイルの存在だけでなく、実Builderがそれを取り込み、CLIまたは実Studio Consumerが生成した振る舞いを使ったことを確認する。旧生成物を混入させる負例を含める。
5. **簡素化の確認**：機能・Expectedを維持して次の定義へ書き直し、手書きSource、同一意味の重複、変更箇所、非局所的な説明の必要性がどう変わったかを比較する。短縮だけを目標にしない。

Goの既存RuntimeやWailsのOS結合を使うこと自体は失敗ではない。**何をTEIが記述・生成し、何を外部Adapterが責任を持って提供するか**が明瞭で、その境界が検査できればよい。GoをDSLへ露出させず、言語依存の機能はTarget Generator／Provider側へ配置する。

### 16.4 Pluginも段階別のContributionでセルフビルド範囲を増やす

Pluginは一つのPackageで全段階を同時に満たすとは限らない。#4646のReferenceに従い、Language/Operator意味定義、Studio/Editor、Generator、Target Runtime、Test／独立Expectedを**それぞれの実Sourceと生成物**に対応付ける。

| Plugin領域 | 最初のTEI化候補 | 外部実装として残してよいもの |
| --- | --- | --- |
| Language／Operator | Role/Shape、契約、演算の型・制約・診断定義 | 最小Bootstrap Reader、必須の低位Parser／実性能Kernel |
| Generator | 入出力の対応、Template選択、Source→Targetの構造的変換・テスト | Go/TS compiler、Target固有のEmitter／ツールチェーン |
| Studio UI | Panel/Operation Binding、Property/Graph表示の宣言と生成 | WebView2／Wails／ブラウザ描画、IME／OSイベント |
| Data／Effect Provider | 必須Context、Capability、Validation、Read/Writeの契約 | RDB Driver、OS Secret、実Tx/外部APIの副作用 |
| Test／検証 | Scenario、独立Expected、生成器負例、Source Trace | 信頼済みTest Runner、実OS／DB Fixture |

**Pluginのセルフビルド比率**は、単なるファイル数でなく、代表的な**振る舞い単位**ごとに「TEI定義で仕様を記述／生成まで到達／独立Expectedで実検証／実Consumerで使用」を別列にし、対象となる全機能の分母と未分類を示す。必要に応じてコード量や生成物byteも補助指標として記録するが、それらだけで品質を判定しない。正確な基準率や目標割合は実測前に固定しない。bootstrap核を残す場合は、そこへ何を委譲しているか明示する。

Plugin自身をセルフビルドする際も既存Builder/BuildIntent/Stage1→Stage2を使用する。TEI Compiler全体をいきなり自己記述する無限再帰を要求せず、実用的な最小seedとその信頼境界を維持しながらTEI定義が担う領域を広げる。

### 16.5 受入順は「CLIで定義実装を検証」から「生成Desktopの実利用」へ

| 作業束／Owner | 実行する一巡 | 合格条件・残すべき上限 |
| --- | --- | --- |
| **SB-CLI（P0並行） #2831／#2620／#2621** | 既存TEI CLIが、実Loaderで読んだSourceから非Pluginの一Build Subjectを選択し、Generator/Builderを実起動 | CLIの実行、selected/actual source、Artifact/Result。形式が未受理なら\`*.tei.md\`実装済みと呼ばない |
| **SB-MEAN（並行） #1124／#988／#951** | 意味を持つ一つの定義を変更→実Go/TS生成変更→独立Expected／Consumer結果が変化 | Goへの単純delegateと区別できる負例、旧生成物拒否、Source位置／意味Diff |
| **SB-PLUG（段階） #951／#4646／#2831** | 最小Plugin Contributionの一部をTEIから生成し、そのPluginを利用して次の定義・部品を生成 | bootstrapとTEI生成範囲を区別し、生成器・Plugin実物の版・依存・呼出し証拠を保持 |
| **SB-DESK（既存P0到達条件） #4583／#1824** | non-Plugin正式BuildIntent→N0/N2/N3→実Stage1 Process→Stage2 Wails/Windows→当該WebView2 | 既存B1-B/C・B2・B3の同一Attempt条件。CLI/Pluginの局所PASSをDesktop完成へ転記しない |
| **SB-SIMPLIFY（継続評価） #2621／#2497／#1180** | 同じ動作・独立Expectedの下でScript／図表／表・生成コード境界を次版へ改善 | 定義の読みやすさ・修正箇所・重複・手書き実装量と動作の非退行を対で確認 |

\`SB-CLI\`と\`SB-MEAN\`は**Studio GUIが未完成でも進める**。既存#2831／#4583が担う\`SB-DESK\`の製品到達条件を取り消すのではなく、そこに「TEI定義による実装」がどれだけ含まれるか別票で説明できるようにする。各作業のOwnerが別であれば並行可能であり、新しい一括Hard Gateを追加しない。

### 16.6 最初の独立負例（既存#1180／#2514へ引渡し）

| 受入候補 | 失敗を露見させる入力 | 期待結果 |
| --- | --- | --- |
| **SSB-01** | \`*.tei.md\`未束縛でCLIが既存Go Generatorを起動 | Go Generator起動の事実は記録できるが、\`*.tei.md\`読取・意味受理・TEI実装のPASSにしない |
| **SSB-02** | TEIの演算条件だけを変更。固定Go delegateのまま生成Byte・振る舞い不変 | 定義→生成→動作の因果連鎖を未成立と判定。単なる起動ラッパーを実質セルフビルドへ数えない |
| **SSB-03** | TEI定義の型・Relation多重度・Validationを破壊 | 独立Expectedが失敗または明示拒否を検出。旧生成物や未関係の固定Go実装で偶然PASSさせない |
| **SSB-04** | PluginはStudio Contributionのみ生成済みでGenerator／Runtime側の実物が不足 | Stageごとの不足・未資格を維持。Plugin全体をTEIセルフビルド済みと表示しない |
| **SSB-05** | 旧Compiler/Plugin版で生成したArtifactが新Targetで使用される | Selected/Actual/Consumed版差を検出し、現在のPlugin機能完成・受理へ自動昇格しない |
| **SSB-06** | DSLを短縮する代わりにRuleをGo側へ移し、生成成果物・動作は同じ | LOC減少だけで改善としない。暗黙移管した意味／独立Expected／変更時の編集箇所を明示 |
| **SSB-07** | CI ShellがStage1を用いずStage2を外側から生成しWindowsだけ成功 | Stage2 Desktopの実使用は限定観測として残すが、Stage1→Stage2の真の再入はFAIL/NOT RUN |
| **SSB-08** | 違うWorkspace／Source RevisionでGo/TS生成・Test／Desktopを合成 | 同一Subject/Source/Attempt/実Consumerの資格を失効させ、別HEAD成功を足さない |

**現状・停止線**：mainの\`internal/buildintent/intent.go\`では\`go-command\`／\`go-package\`のRecipe/Source概念はある一方、\`BuildIntent.Validate\`の\`Subject.Kind\`は現状pluginに限定される。#4583の限定N0子Process起動や生成Windows Jobは、正式non-Plugin BuildIntent全閉包、\`*.tei.md\`からの生成、Plugin自体の再帰、Stage2 Desktopの製品合格を立証しない。本節は設計・受入方針のみで、新CLI/MCP実装・Windows Build／Owner意味受理は**NOT RUN**。新Self-build Engine、第二Plugin Registry、独自Host、TEI専用AI実行機構、Go AST依存DSLは導入しない。

