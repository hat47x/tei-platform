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
