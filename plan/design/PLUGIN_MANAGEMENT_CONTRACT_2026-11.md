# プラグイン管理の共通契約

- **役割**：TEIの「Everything is Plugin」方針に対するプラグイン管理の管理契約（語彙・状態の分離・診断）を、複数のIssueにまたがってまとめる機能設計。
- **正本との関係**：作業・受入・進捗は [GitHub #2044](https://github.com/hat47x/tei-platform/issues/2044) とその子Issue。要求は `requirements/TEI_PRODUCT_REQUIREMENTS.md`。この文書は設計本文だけを持ち、状態の正本ではない。
- **状態**：実装前設計＋S1の実装済み範囲（[#4376](https://github.com/hat47x/tei-platform/issues/4376)）。S2〜S5は未実装であり、この文書へ節を追記する。
- **根拠**：`origin/main` `d58363e1328f25e81923542423bb65c7844989d9`（2026-11-11）時点のコード、`tools/reference-studio-plugin-inventory`・`-disable`・`-data-lifecycle` の3参照実装、#2044の子Issue。
- **主張しないこと**：この文書の語彙・分類は受理・権限・実行許可ではない。実画面・利用者価値・実Invocation・現在Grant・実Effect・Owner受理は別の資格である。

## 1. なぜ管理契約を1箇所に置くか

プラグインの更新・無効化・削除のあとでも、Git管理された設計Sourceは残さなければならない。人と生成AIが「このプラグインは何ができて、何が足りないか」を判断できる必要がある。ところが現行TEIには、その判断に必要な事実が複数の場所に散っている。

- Source宣言は `workspace.Load` の `World.Plugins` と `World.Bundles` にある。
- どのプラグインが選ばれるかは `plugin.Catalog.ResolveEnvironment` が決める。
- 版の固定は `plugin.Lock` が持つ。
- 実行境界と権限は `pluginruntime.BuildPlan` が組み立てる。
- 導入（install）と有効設定（enable）は、**どの構造にも無い**。

同じ事実を画面・CLI・Adapterが別々に導出すると、同じ入力に違う診断が出る。そこで管理側の語彙と導出を `internal/plugin/management` の1箇所に置き、CLIとStudioはその結果を写すだけにする。

## 2. 状態を1つの値に畳まない

管理一覧の行は、次の**別々の事実**を別の欄として持つ。取得できない事実は `null` と `notModelled` で示し、`false` や `enabled` へ読み替えない。

| 事実 | 欄 | 情報源 | 取得できないとき |
| --- | --- | --- | --- |
| 宣言されているか | `declared` | `World.Plugins` | — |
| その環境で選ばれているか | `envSelected` | `Environment.Plugins`／`Bundles`／解決結果 | — |
| 解決されたか | `resolved` | `plugin.ResolveEnvironment` | — |
| lockで固定されたか | `lockPinned` | `plugin.Lock` | `null`（`runtime-unverified`） |
| Artifact | `artifactDigest` | `LockedPlugin.ArtifactDigest` | 空 |
| Build入力 | `buildPlanFingerprint` | `LockedPlugin.BuildPlanFingerprint` | 空 |
| 実行境界 | `runtimeBoundary`／`runtimeTrust` | `pluginruntime.BuildPlan` | `runtime-unverified` |
| 権限 | `reasonCodes` の `permission-missing`／`grant-unverified` | `BuildPlan` の `CapabilityRuntime.Permissions` | `grant-unverified` |
| 導入済み | — | **無い** | `notModelled: install` |
| 有効設定 | — | **無い** | `notModelled: enable` |
| 初期化失敗 | — | **無い**（プロセス起動境界のエラーはある） | `notModelled: init-failure` |
| 実呼出し・実作用 | — | #1174/#1251 の別資格 | `notModelled: invocation`／`effect` |
| 設定版 | `settingsVersion` | **無い** | `null` |

`enabled` という欄は存在しない。有効設定の情報源ができるまでは、`envSelected` がその近似であり、同じものではない。

## 3. 代表的な管理対象と、今日の契約

管理対象を Language Plugin・Operator・Data Provider・Target Generator・UI/Studio拡張・外部連携・テスト診断機構へ広げる要求に対し、現行の `CapabilityKind` は7種で閉じている（deployment-executor / observation-provider / evidence-provider / database-provider / contract-query-executor / migration-provider / editor-intelligence）。管理側は**第2のenumを作らない**。代わりに対応表を出し、未採用の対象を `role-contract-not-adopted` と表示する。

| 管理対象 | 状態 | 今日の表現 |
| --- | --- | --- |
| Language Plugin | 未採用 | #988/#1018 の契約候補 |
| Operator | 未採用 | #988 |
| Data Provider | 表現済み | `database-provider`・`contract-query-executor` |
| Target Generator | 別軸 | Build入力（#943/#1121）。プラグイン行とは別 |
| UI / Studio拡張 | 部分 | `editor-intelligence`。UI Kitは #997 の別軸 |
| 外部連携 | 宣言のみ | `ExternalRequirement{scope,kind,name}` |
| テスト・診断機構 | 未採用 | #4225/#3267/#1180 |

対応表は `plugin.CapabilityKinds` から導出する。契約に新しい種別が増えても、管理表が自動で「採用済み」と主張しないよう、表に無い種別は `capability-kind-not-in-management-table` として別に出す。

## 4. 理由コードを二重に持たない

解決の拒否理由は `internal/plugin/diagnostics.go` の9種（`PLUGIN_RESOLVE_*`）である。参照実装は `plugin-disabled`・`stale-settings:...` などの別語彙を持つ。両方をそのまま製品へ持ち込むと、同じ事実に2つの正本ができる。

管理側は次の規則で1箇所に対応を持つ（`internal/plugin/management/reasons.go`）。

- 解決拒否は `ResolverReason` として **管理コードと解決コードの両方**を返す。解決コードは書き換えない。
- 管理コードは `version-unsupported`・`dependency-cycle`・`dependency-unresolved`・`contract-incompatible`・`resolver-refused` に写す。写す先が無い解決コードは `resolver-refused` に落とし、特定の管理主張へ自動昇格させない。
- lockの不整合は**入力の問題**として扱い（#3184）、`source-fingerprint-mismatch`／`lock-not-provided` を返す。CLIは既存の `INVALID_PLUGIN_LOCK` を返し、新しい失敗符号体系を作らない。

## 5. 診断の形

各行と解決結果は次を持つ。診断は許可を与えない。

- `reasonCodes[]`（表示文言と独立）と `messages`（コード→文言）。文言を変えてもコードは変えない。
- `remediation[]`（次にできること：`supply-plugin-lock`・`grant-permission`・`update-plugin-version` 等）。
- `impact`（`resolved`／`no-data-access`／`unknown`）。
- `recheck[]`（設定変更後・lock提供後・Source変更後・scan完了後）。
- `remaining[]`（再走査の要否など）。
- `evidence[]`（種別・参照・指紋）。Source本文・データ値・秘密は含めない。
- `notModelled[]`（上述の未モデル事実）。
- `scanState`（`complete`／`partial`／`not-run`）。**部分走査で合計を0や `confirmed=true` にしない。**
- `authorityLimits[]`（この診断が何を保証しないか）。

## 6. データの対応（level）

環境が宣言するデータの単位を `levels` として出す。

- `database-instance`：`Environment.Instances` の provider と、その provider を提供する宣言済みプラグインを突き合わせる。1提供元は `provided`、複数は `multiple-providers`、0は `no-provider`。
- `feature`：`Environment.Features` と、その feature を宣言した capability を突き合わせる。0は `unsupported`。

**名前の一致やロード順で提供元を選ばない。** 複数提供元のときは両方を並べ、`covered` を1回だけ数える。

## 7. S1で実装した範囲

- `internal/plugin/management`：導出（`Derive`）、語彙、理由コード対応、role表、level導出、詳細の文。
- CLI `plugin-inventory`：`-workspace`|`-world`・`-environment`・`-plugin-lock`（任意）・`-plugin`（詳細）・`-diagnose`（理由コードのある行だけ）。`mode` は `""`・`detail`・`diagnose` の閉じた集合。
- `tei commands` と `plan/design/CLI_JSON_OUTPUT_CONTRACT_2026-10.md` の表へ登録。
- 負例の試験（状態の分離、部分走査、未宣言、未選択、版不一致、循環、lock不整合、権限、実行環境未対応、複数提供元、決定性）。

### 7.1 検証（2026-11-11、`d58363e13`）

```text
gofmt -l internal/plugin/management cmd/tei            差分なし
go vet ./internal/plugin/management ./cmd/tei          ok
go test ./internal/plugin/management/ -count=1         ok（17件）
go test ./cmd/tei -run 'PluginInventory' -count=1      ok
go test ./cmd/tei -run 'Contract|Commands|Flag|Catalog|Discovery' -count=1  ok
go run ./cmd/tei plugin-inventory -workspace reference/workspace -environment development
go run ./cmd/tei plugin-inventory -workspace reference/workspace -environment development -plugin-lock <lock>
go run ./cmd/tei plugin-inventory -workspace reference/workspace -environment production -diagnose
```

`reference/workspace` の `development` はlock無しで8行（`runtime-unverified`・`grant-unverified`）、lockありで `planBuilt=true` と `runtimeBoundary` が埋まる。`production` は `editor.database_workbench` が `unsupported` となる。

## 8. 未実装（S2以降）

- 管理操作の**計画**（候補展開・差分・影響・再生成／再起動の必要性）は [#4377](https://github.com/hat47x/tei-platform/issues/4377)。
- 設定源・`settingsVersion`・Source保持・監査・原子性の確定は [#4378](https://github.com/hat47x/tei-platform/issues/4378)。これが無い限り「有効設定」は `notModelled` のままである。
- Studio表示・操作Adapterと実画面は [#4379](https://github.com/hat47x/tei-platform/issues/4379)／#2547／#2550。
- 不可逆操作（データ削除・変換・外部リソース解放）は [#4380](https://github.com/hat47x/tei-platform/issues/4380)／#2553。Owner判断が揃うまで実行しない。
- 実 `.tei` の未知属性・未知子要素は現行パーサが読み込みを拒否する。バイト保持を主張できるのは #1920 の版寛容リーダー以降である。

## 9. 第二の基盤を作らない

- 解決は `plugin.Catalog.ResolveEnvironment`／`ResolveEnvironmentWithLock` を1回だけ使う。管理側は解決順・候補・fallbackを持たない。
- 実行境界は `pluginruntime.BuildPlan` の結果を写す。管理側で境界を決めない。
- 権限は `BuildPlan` の `CapabilityRuntime` に対する `permission.Missing` を写す。別の権限エンジンを作らない。
- `tools/` の参照実装は仕様であり、製品importしない（依存試験で固定）。

## S2〜S5への接続：起動・配布・更新と管理状態（2026-11-15、設計引渡し）

[Studio／CLI／Web Backend・配布更新統合設計](STUDIO_CLI_BACKEND_PROCESS_DISTRIBUTION_UPDATE_2026-11.md) §11〜17を参照する。Desktopの子Backendが`CORE_READY`であることは、個々のPluginがInstalled／Enabled／Activeであることを意味しない。Studio・CLI・Webは管理契約の同じInventory／Planを投影し、未知の状態は`notModelled`のまま返す。

- **S2 #4377**：Plugin更新と本体更新の候補について、依存Closure、必要な停止・再起動、対象UI Asset、Source不変、現在Grant／実Providerへの影響を計画できるようにする。計画で導入・有効化を確定しない。
- **S3 #4378**：Installed／Enabled等の設定源・settingsVersion・操作履歴をSourceの正本から分離する。Plugin更新のStagingと有効世代の切替は別操作とし、実行中のInvocationを旧・新構成へ混合しない。
- **S4 #4379**：Studio画面とCLIから同じ状態・診断・構成版を参照する。GUIはControllerではなくAdapterであり、Remote CLIはServer側の選択済み構成を使用する。CLI引数から未選択Providerを自動で採用しない。
- **S5 #4380**：無効化・実装削除・Plugin保持データ削除・形式変換・外部Effectは別操作。更新前の旧Artifactへ戻しても、DB Migration・確定済みEffectは元に戻らない。
- **説明文書とSkill**：Plugin Package／Releaseが公開するContract、Manual、Example、短いAI Skill、上位依存の文書参照をVersion／Digest／公開範囲で辿る。文書からの実行Grant継承や秘密情報の無断開示を認めない。

現時点で`install`・`enable`・`init-failure`を実装したという主張ではない。受入Ownerは引き続き#2044とS1〜S5の各Issueであり、新しい管理Resolverや更新状態の第二正本は設けない。