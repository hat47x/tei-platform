# Cordis由来の合成機構とTEIプラグイン継続更新の評価

- 状態：**比較検討時点の記録**（2026-10-12）。現行ABIを維持するという前提は、後続の[ADR-0032](../adr/0032-composition-kernel-replacement.md)により置き換えられた。本書の採否表は技術評価の履歴として読む。無停止の実環境保証やProduction統合を示さない。
- 管理Issue：[#1223](https://github.com/hat47x/tei-platform/issues/1223)
- 既存の正本：[Plugin / Bundle](../../docs/plugins.md)、[案件構成](PROJECT_PLUGIN_COMPOSITION_DESIGN.md)、[多様なTarget向けランタイム](RUNTIME_TARGET_BINARY_COMPOSITION_DESIGN.md)、[generic Capability移行](GENERIC_CAPABILITY_MIGRATION_DESIGN.md)
- 関連Issue：[#951](https://github.com/hat47x/tei-platform/issues/951)、[#1016](https://github.com/hat47x/tei-platform/issues/1016)、[#259](https://github.com/hat47x/tei-platform/issues/259)、[#1118](https://github.com/hat47x/tei-platform/issues/1118)、[#1174](https://github.com/hat47x/tei-platform/issues/1174)、[#943](https://github.com/hat47x/tei-platform/issues/943)
- 目的：**ほとんどの機能更新をプラグイン単位で実行できる極小・堅牢なコア**を目指しつつ、「更新機構の正常動作」「利用者から見たサービス継続」「業務状態の整合性」を混同しない。

## 1. 参照範囲と結論

Cordisの本質は、作用を所有するスコープ（Revertible Effects）と、サービス依存関係を動的に満たす条件（Reactive Coeffects）を、Contextに集約した合成モデルにある。DeepSeek Harnessでは、依存が成立しないプラグインを待機させ、依存喪失時に破棄、復帰時に再起動する。登録解除や子Fiberの終了もライフサイクルが管理する。

この性質はTEIに有益。ただし、**Cordisをライブラリとして全面移植・模倣しない**。TEIの現行実装は、exact Plugin Lock / Build Plan、Runtime Engine identity、Permission、Resource Lease、Supervisor fencingを別の権限として扱い、static/Wasm/IoT等も対象とする。元技術の利点をTargetごとに必要な最小契約へ還元することを優先する。

一次資料：
- 論文（2026-08-26）：https://arxiv.org/abs/2608.25512
- Cordis本体（API未安定）：https://github.com/cordiverse/cordis
- DeepSeek Harnessサービス依存：https://deepseek-harness.github.io/deepseek-harness/en/develop/framework/service
- DeepSeek Harnessライフサイクル：https://deepseek-harness.github.io/deepseek-harness/en/develop/framework/
- ユーザー提示の紹介記事：https://qiita.com/anchor_/items/74f83c39cd67c59d9f94

## 2. 利点・欠点・採用の優先度

| 候補 | TEIで期待する長所 | 欠点・新たな失敗要因 | 判定 |
| --- | --- | --- | --- |
| Effect所有と逆順解放 | PluginのListener、Timer、Connection、登録済みProviderを漏れなく回収しやすい | 解放処理自体の失敗、遅延、順序依存、外部不可逆作用は残る | **優先して小規模Reference** |
| 依存の宣言とリアクティブ追従 | 実装の追加・喪失を機械的に検出し、影響範囲を局所化 | 依存の連鎖停止、再起動嵐、過剰な動的管理、短時間のサービス停止 | **優先。ただし更新と障害のポリシーは区別** |
| Context / Service隔離 | 具体的なProviderへの直結を避け、案件・テナントごとの差し替えがしやすい | Contextの可視性はアクセス許可を保証しない。スコープ漏れの危険 | **Capability/Host Authorityへ接続する場合に採用** |
| サービスのProvider/Consumer分離 | アプリ、認証方式、通信実装を変更しても契約を保ちやすい | 契約を曖昧にすると実装の交換可能性は幻想になる | **既存 #951 を優先。第二Resolverは作らない** |
| Parent/child scope | 機能単位の所有・一括解放を表現しやすい | 物理プロセス/DB/Appの所有関係と混同するとN:M配置に不適切 | **Runtime instanceの所有だけに限定** |
| 型付きイベント / Waterfall | 計測、UI拡張、処理前後のInterceptorを整理できる | 実行順、例外、短絡、重複登録に依存し、認可の抜け道になり得る | **必要な契約だけ選択** |
| Profile / Bundle / Patch | 案件ごとの差分、配布構成、組合せの再現性を高める | 適用順に依存する暗黙の上書き、運用時Lockとの乖離 | **既存 #1016 / Lockを拡張せず再利用** |
| HMR | 開発サイクルを短縮し、運用更新の足場になる | Loader成功は通信継続・無停止・状態移行の証拠にならない | **開発用途に限定して先行。運用更新は別Gate** |

## 3. 死蔵されやすい技術と採らないもの

Cordisの機能を全面搭載しても、TEIの一部Targetでは費用と複雑性だけが増える。実装を取り込まず、必要になった時点で選べる設計とする。

| 要素 | 活用しにくいTarget/状況 | 判断 |
| --- | --- | --- |
| 常時稼働するDependency watcher / 動的Plugin loader | Go静的リンク、単一Wasmモジュール、制約の強いIoT | ビルド時に依存を解決し、監視器とLoaderを除外する |
| TypeScript declaration merging / decorator由来の注入 | Go、TinyGo、言語をまたぐ契約 | コンパイラ固有の型拡張は移植しない。型付き契約生成で代替 |
| ソースファイル変更検出とJS HMR | 本番、ブラウザの制限環境、署名済みFirmware | 運用時は承認済みartifactのステージングと切替を使用 |
| JavaScript式を評価する動的構成 | deterministic build、closed network、高セキュリティTarget | 副作用のない宣言・明示的解決を使い、任意評価を要求しない |
| すべてのイベントモードをCoreへ内蔵 | 純粋計算、単純な組み込み制御、低メモリTarget | Event型/Modeは必要なPluginの契約に留める |
| Provider消失時にConsumerを即時disposeして復帰時に自動再起動 | 強い可用性SLO、状態移行中の書込み処理 | 自動停止は安全用fallbackであり、計画更新は事前並走・世代切替を使う |
| Pluginの親子Fiberを配備単位・DB所有関係に直接対応づけること | 複数アプリと複数DB、複数Artifactへの分割 | Ownershipと配置/Contract依存は別グラフとする |
| DeepSeek固有のagent loop、会話Event、Tool用のService集合 | 一般の業務アプリ、SUI/VR、IoT | 汎用Plugin基盤へ持ち込まない。AIアプリPluginの候補に留める |
| 全副作用の自動rollbackという期待 | DB transaction、メール送信、決済、機器操作 | **不採用**。永続作用は業務上の補償・冪等性・監査を別に扱う |

**死蔵しないための原則**：機構を「共通Runtimeの機能」として存在させるのではなく、Conformanceを守ったままTarget単位で消去できることを製品価値にする。採用しない機能のための設定フィールドや起動コストを要求しない。

## 4. 極小コアが保証すべき不変条件

コアは機能そのものではなく、**交換が安全に成立する境界**を担う。現行Runtime Authorityを迂回する新しいKernelを作る案ではない。

| コア側／信頼できるHostが失ってはいけない責務 | Pluginに委譲できる部分 |
| --- | --- |
| 契約、Runtime Engine、Plugin、Artifact、Targetのidentity照合 | 業務機能、認証方式、DB Provider、Renderer、連携実装 |
| 現在のAuthorityの適用、Resource Lease/fencingの確認 | 個別ポリシーの実装、外部接続、Plugin固有のリソース |
| 呼出しが参照する世代の確定と、無効化済み世代への新規割当の禁止 | 業務処理、Handler、Interceptor、Listener |
| 解放責務の所有、ライフサイクル終端までの追跡 | 各Pluginが登録するdisposer・正常終了処理 |
| 回復不能時に停止/隔離するfail-closed境界、監査Evidence | 更新計画UI、運用画面、ログ転送、観測Adapter |
| Root of trust / bootstrap recoveryの到達可能性 | 交換可能なUpdater、Downloader、Package Manager |

コアを小さくするとは、全検査を薄めることではない。**安全性の判断を失わず、その実装先を適切に分離する**ことである。コア自体のCVE、Runtime ABI、OS/Host更新はPlugin交換だけで解消できると断定しない。

既存のAuthority：
- Runtime Plugin Lock / Build Plan / Runtime Engine Lock はそれぞれ独立。
- Capabilityの選択と、実際のInvocation Authorityは独立。
- Contextの可視性と、Host Principal/tenant/sessionに基づく権限は独立。
- Bundle / Profileと、Accepted business meaningは独立。
- Artifactのdigest照合と、publisher trust/現在のrevoke判定は独立。

## 5. 無停止交換のReferenceプロトコル（未実装）

最初の対象は、独立したPlugin runtime instanceが提供する**read-only Capability**とする。同期書込み、DB migration、外部不可逆操作を同時に検証対象へ入れない。

1. **Select**：既存のContract/Provider/Target解決とexact Lockに基づいて候補Artifactを選ぶ。複数候補や不適合はfail-closed。
2. **Preflight**：signature/provenance、現在のAuthority、契約互換、Resource/Host制限、併走中のピーク使用量を確認する。
3. **Stage**：新世代を旧世代と独立したスコープに準備する。readiness前に新世代へ本番トラフィックを渡さず、外部副作用を禁ずる。
4. **Ready**：契約単位の健全性と呼出し能力を検証する。単なるProcess起動・Plugin ACTIVEをReadyとしない。
5. **Publish**：当該Hostの新規呼出しが参照するImmutableなGeneration Bindingを切り替える。新規呼出しは旧世代を参照しない。**クラスター全体の瞬間的な原子性までは主張しない**。
6. **Drain**：旧世代の既存呼出しとLeaseを追跡して終了させる。timeout時に旧版を強制解放することは、呼出し結果不明と同義になり得る。既存fencing・Cancellation規則を確認する。
7. **Retire**：旧世代のEffectを逆順解放し、子スコープも停止する。resource leakを監査し、旧世代が再公開されないことを確認する。
8. **Observe**：切替前後の実際の呼出し・エラー・遅延・業務Observationを記録。Availabilityの達成は独立SLOに照合する。
9. **Recover**：切替前の失敗は旧版を保持。切替後の復旧は旧Artifactと状態互換が確認できる場合だけ戻す。非可逆migration後の自動rollbackは禁止する。

### 世代モデルと制約

- Generation = 選ばれたContract/Provider、exact Plugin/Runtime Engine/Artifact identity、runtime scope、Authority参照、運用上の一時状態。identityの正本を重複定義しない。
- 各Invocationは開始時に一世代へ束縛する。処理中のService pointerを無通知で新実装へ置換しない。
- old generationへ新規の権限・Resourceを無条件に付与しない。権限失効時はAvailabilityより安全性を優先する。
- 状態遷移候補：CANDIDATE → STAGED → READY → SERVING → DRAINING → RETIRED。REJECTED/FAILEDは別の観測結果。これはReference用状態であり、production enumではない。
- 同一HostのBinding交換に原子的な実装ができても、全プロセス・ネットワーク間で原子的なグローバル交換が実現したことにはならない。
- memory/CPU/connection上限に旧新二世代の併走余裕がないTargetでは、「無停止」を保証できない。別ノードでの冗長化、段階再起動、もしくは明示的な停止更新を選ぶ。

### 計画更新と故障時の依存消失を分離

計画更新では旧Providerを先に外さず、Readyな新Providerへ新規利用を切り替えた後でdrainする。一方、予期しないProvider喪失では、依存を満たさないConsumerを動かし続けず、権限に従って停止・隔離する。Cordisの単純な「Provider喪失→Consumer破棄→Provider復帰→Consumer再起動」は**後者の基本安全策**としてのみ参考にする。

無停止性を保証する条件は、(a)必要な冗長化・二世代共存、(b)互換Contractと永続状態、(c)現在のAuthority、(d)新旧リソースの容量、(e)実際のobservability、のすべてが確認できる場合に限る。

## 6. 永続状態と更新の分離

Effectの自動解放対象は、Listener・Timer・Connection・サービス登録など、スコープ内で明示的に所有する一時的作用である。DBへの確定済み更新、通知、決済、外部設備への命令を機械的に巻き戻せない。DB Schemaはアプリ/Pluginと独立したライフサイクルを維持し、expand → old/new共存 → contract の段階移行、必要なread/write互換、transaction境界、補償・idempotencyを個別契約として扱う。

Pluginが交換可能でも実行状態が移行できなければ更新は停止する。個別Providerを差し替えるだけで永続ジョブのexactly-onceが成立するとは主張しない。

## 7. Target別の適用

| Target | 実際の更新単位 | Cordis由来の適用範囲 |
| --- | --- | --- |
| Server、Desktopの独立プロセス | Process/Worker/サービスインスタンス | lifecycle・依存宣言・世代切替・drainが有効 |
| 長時間稼働するWasm Host | Hostが許容するWasm instance | 新instance準備とDispatcher切替。Host ABI/共有状態に制限 |
| Go静的リンク・単一プロセス | 実行ファイル/Process | 任意のGo関数をin-placeで安全に差し替えられるとは主張しない。冗長Process/外部Hostが必要 |
| Web browser / Service Worker | 配信資産とClient session | Browserの更新制約と既存sessionの兼ね合いを検証。全Clientの瞬時切替は不可 |
| Single-node IoT / Firmware | Firmware slot / Reboot | 配備・復旧のA/B方式を検討できるが、単一機器の再起動停止を「無停止」と呼ばない |
| 冗長IoT/分散環境 | Replica / Device group | 段階配備、Lease/fencing、観測を前提にサービス単位の継続性を評価 |

## 8. 最初の机上検証：正例と負例

| ID | Given / When | Then | Evidence |
| --- | --- | --- | --- |
| U01 | v1で2 Consumerが呼出し中、v2がReady | v2へ新規呼出しを切替。既存呼出しはv1で完了 | generation/in-flightの紐付け、成功/失敗 |
| U02 | v2の互換Contract・exact Artifact不足 | Publish禁止、v1保持 | reject reason、Lock照合 |
| U03 | v2 readiness失敗・起動は成功 | v1保持。ACTIVEだけで切替しない | readiness observation |
| U04 | v1 drain timeout | 強制解放を成功扱いしない。未完了呼出しの状態を区別 | timeout/unknown evidence |
| U05 | v2のResource usageが併走制限超過 | 無停止と宣言せず、更新の選択を保留 | capacity decision |
| U06 | 切替後にPermission/Lease失効 | stale Authorityで呼出ししない | deny/fencing trace |
| U07 | v2発行後にv1へ遅延した旧参照が到来 | retiredへの新規利用を拒否 | late-binding negative |
| U08 | 永続DB schemaが旧版と非互換 | 自動rollback禁止、明示移行計画要求 | schema compatibility judgment |
| U09 | Provider消失・復帰が頻発 | Consumerの起動嵐を有効な完了扱いしない | bounded retry/diagnostics |
| U10 | 二つのProviderが同条件で候補となる | 勝者をheuristicに決めずunresolved | ambiguity result |
| U11 | Plugin解放後にListener/Timerが残る | 漏れを検出、Cleanup成功を偽らない | owned resources before/after |
| U12 | リモート/複数NodeでBindingを切替 | Hostごとの世代と観測を区別。全Node同時更新を断定しない | host-local generation evidence |

優先順はU01〜U04およびU06・U11とする。最初は**純粋なread-only ProviderのReference**であり、実環境での連続性を証明しない。次段では別Process / Wasm実体とactual routing/traffic testへ進める。

## 9. 実装への移行Gate

- G0：既存Plugin/Capability/Runtime Authority・Build/Lock・Deployment/Targetとの契約棚卸し。第二権限・第二Resolverを作らない。
- G1：U01〜U04/U06/U11の状態遷移と負例を小さなReferenceで再現。実際に確認できないものはunresolved。
- G2：ProcessまたはWasmの一環境で、旧新二世代・readiness・drain・実負荷・resource cleanupを測定。
- G3：書込みやDB移行を伴う場合は#1174を含め、Atomic commit/Authority、expand-contract、Observation、rollback限界を別途検証。
- G4：Targetごとのlinker削減と動的機構の不使用、運用SLO、Trust、signed artifact、回復実績を検証。セキュリティとサービス継続を別の評価軸とする。
- Main統合は現行のAGENTS.md / CONTRIBUTING.mdのexact-head・検証ルールに従う。Doc-onlyの設計整理をproduction合格のEvidenceにはしない。

## 10. 未確定事項と推奨判断

1. **最初に採る**：Effect ownership、generation-bound Service reference、dependency preflight、計画更新と障害停止の分離。
2. **必要性が観測された後に採る**：依存追従の常時監視、任意Event mode、高度なProfile overlay、分散整合調停。
3. **採らない**：CordisのTypeScript固有API/Loader/HMRの全面移植、全てのTargetへの動的ランタイム同梱、永続業務作用の自動取消。
4. **先に決めない**：production schema、共通Lifecycle enum、Go/Wasm共通ABI、2つ目のAuthority、generic Scheduler。小さな実行Referenceが不足を示したときだけ追加する。

この文書は既存設計の正本を置き換えない。特に #951、#1016、#259、#1174、#943 の進行状態が変われば、既存契約と照合して必要な節だけを更新する。
