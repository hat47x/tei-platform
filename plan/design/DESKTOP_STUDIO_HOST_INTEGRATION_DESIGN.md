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
