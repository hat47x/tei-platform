# 多様な実行環境に向けた最小ランタイムと配備単位の設計

> **後続判断（2026-10-24）**：[ADR-0043](../adr/0043-target-native-artifacts-without-tei-host.md)により、最小ランタイムはTEI共通Hostの必須構成を意味しません。対象環境の既存機構の利用、アプリケーションへの組込み、必要機能の静的結合を基本の選択肢として評価します。本書は評価待ちの候補であり、採用済みRuntime仕様ではありません。

- **状態**: 評価待ち（2026-10-09）。設計候補であり、実装済みの機能、採用済みの配備定義形式、全環境への対応を主張しない。
- **主担当**: [Target Capability / 実行・信頼条件の検証 #259](https://github.com/hat47x/tei-platform/issues/259)。ビルドの入力・成果物・固定情報は[#943](https://github.com/hat47x/tei-platform/issues/943)、Capability契約と実装選択は[#951](https://github.com/hat47x/tei-platform/issues/951)、案件構成は[#1016](https://github.com/hat47x/tei-platform/issues/1016)へ戻す。
- **役割**: Web・スマートフォン・デスクトップ・サーバー・分散処理・組み込み機器を、一つの巨大な汎用ランタイムに押し込まず、必要な機能だけを含む配備成果物として構築するための設計。既存の[全体構成](../../docs/architecture.md)、[Plugin / Bundle](../../docs/plugins.md)、[案件構成設計](PROJECT_PLUGIN_COMPOSITION_DESIGN.md)を第二の正本にしない。

## 1. 避けたい失敗と基本判断

TEIの標準機能をプラグイン化しても、すべての実行先で「プラグイン管理機構を内蔵したフル機能ランタイム」を起動すれば、未使用の解析器、管理画面、データベース接続、監視機構などが成果物へ残る。メモリや起動時間の制約が厳しい機器では、この設計そのものが配備を妨げる。

そこで**プラグインは論理的な機能・契約の単位、配備物は選択された実装の集合を最適化して生成する物理的な単位**とする。

```text
採用済みの業務・アプリケーション上の意味
           │
案件構成・要求Capability・環境への参照
           │
既存の契約/Provider解決・各Lockの照合
           ↓
実行単位ごとの配置計画（追加する判断だけ）
           ↓
実行単位ごとに必要な実装と依存関係を確定
           ↓
Target別Builder・静的結合／Wasm／別プロセス化
           ↓
最適化済み成果物 + 内容と生成根拠を示す目録
           ↓
配備時の環境Binding・最新の実行権限確認
```

同じ論理プラグインは、Linux上ではGoの静的結合、ブラウザではJavaScript/Wasm、マイコンでは専用のファームウェア実装など、異なる成果物で実現してよい。**同じGoバイナリがすべての環境で動くこと**は要求しない。反対に、実行形式が異なるだけで業務意味やCapability契約を変更してはならない。

## 2. 成果物に必要な機能と解決時点

「ランタイム」という語は、次の責務を混同しないために限定して使う。

| 区分 | 最小限の責務 | 常駐が必要か |
| --- | --- | --- |
| 意味を扱う共通部分 | 型・識別・制約等の正本から実行に必要な表現を生成する。 | **生成時のみ**でよい。実行先でDSL解析器を動かすことを要求しない。 |
| 構成・ビルド機構 | 固定された契約・実装・依存・Target・Builderから成果物を作る。 | **生成時のみ**。配備先では不要。 |
| 最小実行入口 | 呼出しを受け、固定された実装へ渡し、結果・失敗・不明を区別する。 | その実行単位が呼出しを扱う場合だけ必要。 |
| 実行に必要な能力 | 時計・記憶域・通信・並行処理・秘密情報等のうち宣言されたものを、既存環境または成果物内の実装へ対応付ける。 | 必要な能力だけを含める。 |
| 実行時の保護 | 副作用を担う主体が、その時点の認可・Resource権限を適切な境界で確認する。 | 副作用を担う境界では必要。 |
| 運用・監督 | 永続ジョブ、再試行、観測、Lease、fencing、複数ノードの調停等。 | 必要な実行単位だけ。組み込み機器には一律搭載しない。 |

最小実行入口は「汎用プラグインをその場で発見する管理画面」でも「必ず起動する常駐Supervisor」でもない。静的結合された一つの純粋な計算関数やイベント処理器なら、呼出し・結果の契約を保つ小さな関数群が実行時の全体になり得る。

ただし、組み込み処理の一部を省いたことを理由に、存在しない認可、耐障害性、サンドボックスを「基盤が保証する」と表示してはならない。

## 3. 配備に関係する四つの独立した軸

- **Application / Projection**: 何を実現するか。受注画面、API、通知処理、センサー集約など。
- **配置と信頼境界**: どの処理をどの実行単位へ置き、どの呼出しがプロセス・ネットワーク・信頼境界を越えるか。二層・三層・分散等は、この配置から生まれる性質である。
- **実行Host / Target**: ブラウザ、OS上のプロセス、モバイルアプリ、WebView、WASI Host、RTOS・ベアメタル等。OS/CPU/ABI、利用できる機能と資源制約を含む。
- **成果物の実現方式**: 静的結合した実行ファイル、Web資産、Wasmモジュール、Wasm Component、別プロセス、共有ライブラリ、ファームウェア等。

この四軸は階層でも排他的な「プラグイン種別」でもない。WasmをWeb Targetと同義にせず、Wailsをデスクトップ用の共通ランタイムともみなさない。

特に、クライアントの画面処理と、権限を伴うサーバー側の業務処理を同じ実行単位へ置く必要はない。オフライン端末にローカルの業務処理を置く場合は、競合解決・同期・秘密情報・再接続後の承認を追加の契約として扱う。

## 4. 配備指定の最小候補

案件構成やPlugin Lockから導出できる情報は重複記述しない。まず、**何をどこで動かし、実行単位間のどの契約を結ぶか**だけを記述する案を比較する。

次は説明用の候補であり、`tei.deploy.yaml`という正式なファイル名や構文を採用したものではない。識別子は実際の`order_entry`契約と同一とは限らない。

```yaml
version: tei.deploy/v0             # 未採用の設計例
application: order_entry           # 既存の正本を参照
units:
  browser:
    host: browser
    target: web
    roots: [order_entry.web]
    calls:
      order_entry.submit: api      # 呼出し先の実行単位
  api:
    host: process
    target: linux/amd64
    roots: [order_entry.submit, order_entry.list]
    uses:
      order_entry.persistence: env.sales_db
```

この例ではBrowser側からDatabase Driverやサーバー専用認証実装を除外し、API側からEditor、Web画面資産、不要なDSL構文解析器を除外できる。ただし、`calls`は通信と実行権限を成立させる宣言ではない。採用した契約、通信方式、認証・認可、環境Binding、境界を越えた結果の扱いは、既存のCapability・Environment・Hostの契約と照合して初めて成立する。

### 最小候補フィールドの意味

| 項目 | 必須とする条件 | 扱い |
| --- | --- | --- |
| `application` | 常に | Accepted sourceへ戻る参照。業務意味を複製しない。 |
| `units` | 常に | 実行、停止、更新、障害分離の単位。単一実行なら一つ。 |
| `host` / `target` | 各unit | 実行環境と出力条件。より詳細なCPU、OS、ABI、利用能力はTarget/Builder側の仕様に委譲する。 |
| `roots` | 各unit | 実際に外部から到達可能な入口。Plugin package名ではなく、Projection/Action/Capability等の論理参照とする。 |
| `calls` / `uses` | 境界をまたぐ場合だけ | 実行単位間の明示的な接続と環境参照。Provider選択は既存の解決・Lockへ戻す。 |
| `limits` | 資源制約がある場合だけ | 例: Flash/RAM上限、起動時間、接続可能なAPI。満たせない構成はビルド時に拒否する。 |

不要な項目: プラグイン全一覧の複製、Secret値、全依存の展開済み一覧、固定のレイヤー番号、すべてのTargetに共通の認証・監視設定、Build Lockの再掲。

## 5. 静的最適化と実行形式

最小経路は**ビルド時合成（静的結合）**とする。実行時にプラグインを差し替える機能は、要求されたHostでのみ追加する。

1. **解決**: 実行単位の入口から必要なCapabilityを辿り、既存の契約定義権限、契約の意味・版、Provider適合性、作用域、Plugin Lockを検証する。依存関係はbuild-timeとruntimeに分ける。
2. **閉包の確定**: 入口から呼び得る実装とHost機能を求める。未使用実装は取り込まない。Reflection、動的Dispatch、生成コード、FFI等、静的解析だけでは到達を証明できない辺は、明示した保持集合か実行時解決契約がなければ失敗とする。
3. **配置との照合**: クライアントにServer専用Secret、不可視のDB接続、特権的なHost操作を含めない。信頼境界を越える呼出しは、版付きの契約と具体的なTransport Adapterを要求する。
4. **成果物生成**: Goでは選択済みProviderだけを取り込む登録コードを生成し、ビルド制約、リンク時の未使用コード除去、必要に応じた最適化を使う。WasmやファームウェアはそれぞれのCompiler/Builder Adapterが作る。全Hostで同じリンク方式を必須にしない。
5. **成果物の確認**: 実際に含まれる依存、出口となる呼出し、Target互換性、サイズ、起動・メモリ使用量を確認する。静的解析上の「除去予定」と実バイナリから「除去された」は区別する。
6. **生成根拠の保存**: 成果物のdigestと、その成果物を作ったexact Source/Build Plan/Toolchain/Target、含まれる論理契約とPlugin版を追跡する。既存BuildArtifactLockとPlugin Lockを正とし、重複する第三のLockを作らない。

成果物は、同一Pluginの複数実装を含む必要がなく、逆に複数Pluginが一つの共有ライブラリやWasmへ統合されてもよい。ただし元の契約、Provider、問題箇所への対応は失わない。

### 実現方式の使い分け

| 方式 | 優先する状況 | 注意点 |
| --- | --- | --- |
| ネイティブ静的結合 | OSサーバー、デスクトップ、固定構成のエッジ端末 | デプロイ後の実装交換には再ビルドが必要。 |
| Web資産 / JS / ブラウザWasm | ブラウザ画面、純粋計算、ローカル入力検証 | Wasmが必須ではない。ブラウザHost権限とServer Authorityを混同しない。 |
| Wasm / WASI Component | Host間の可搬性、Import/Export境界、隔離・配布 | Hostが提供するWASI版・import・resource・実行制限の適合を確認する。 |
| 独立プロセス / RPC | 障害・信頼境界を分けたい場合、言語の異なるProvider | 通信・再接続・未知の実行結果・Resource/Leaseが必要。 |
| MCU用ファームウェア | RAM/Flash制約が厳しく、機能構成が固定された機器 | Go全機能や動的ロード、HTTP Host等は前提にしない。 |

Go標準の`plugin`パッケージを標準方式にしない。対応OSとビルド互換性の制約があるため、移植性・サプライチェーン・隔離の基盤としては不適切である。Goコードの静的結合を第一候補とし、隔離が必要ならOSプロセスやWasm Host等を選ぶ。

TinyGoはブラウザWasm、WASI、対応マイコン等を実証する候補である。ただしGoのすべての機能・標準ライブラリ・対象機器に対応するとはみなさない。HikeはGo構文に近い別言語であり、既存の任意のGo実装をそのままWasm化するCompilerではない。両者とも**共通の契約からTarget向けの小さな実装を生成するBuilder候補**として評価する。

## 6. 具体的な配置例

| 構成 | 実行単位 | 生成・配備するもの | 決定的な境界 |
| --- | --- | --- | --- |
| Web二層相当 | Browser + API/DBを所有するサーバー | Web資産、API実行ファイル | Browserは特権を持たない。DB認証はServer側。 |
| スマートフォン + サーバー | iOS/Androidアプリ + API + DB | OSごとのアプリ成果物、API実行ファイル | Mobile OSのAPI、保存・配布・バックグラウンド制限。 |
| デスクトップ二層 | Desktop + DB | Wails等のGUI Hostを含むDesktop配布物、DB接続Provider | DB直接接続を許すならDB側の利用者別権限・秘密情報管理が必要。 |
| デスクトップ三層 | Desktop + API + DB | Desktop配布物、API実行ファイル | ClientはUI、Serverは業務実行と権限判定を所有する。 |
| 分散・エッジ | API + Worker + Node/Gateway | unit別実行ファイル・Container・Wasm等 | 再試行、重複、副作用、観測不能、fencingを必要な単位だけに持たせる。 |
| IoT / MCU | Sensor firmware + Gateway/API | ファームウェア、Gateway実行ファイル | センサー側は収集・簡単な判断に絞り、重い認証・集計・永続化をGatewayへ分離できる。 |

二層・三層を固定したプラグイン種別にはしない。実際にどこへAction、データ保存、通信、保護境界を置くかによって構成が決まる。

デスクトップの「二層」はDB直接接続を推奨するという意味ではない。利用者ごとのDB認可やSecretを安全に成立させられない場合は、三層構成へ寄せる。

分散構成でも各Nodeへ完全なSupervisorを配布する必要はない。一方、副作用を伴う処理を別のNodeへ移す場合、既存のSupervisor所有権とRuntime側のfencing確認が一致しない状態を成功とみなさない。

## 7. 必須規約

1. **意味不変**: あるTargetで機能を実現できない場合は、明示的に非対応・未解決とする。業務意味を自動的に弱めない。
2. **依存の閉包**: Build対象となる全実装の契約、版、digest、Provider適合性、Target適合性を固定する。複数候補をロード順で選ばない。
3. **明示した境界**: 実行単位の内部呼出しと、ネットワークやプロセスを越える呼出しを区別する。前者の権限が後者へ自動的に移ることはない。
4. **実際の安全性**: in-process Pluginを同一OS権限で動かしている場合、TEI上のPermission宣言だけでOSへの直接アクセスを止められるとは主張しない。信頼できない実装には隔離可能なHostが必要。
5. **不明の保存**: Timeoutや通信切断を、相手側の処理失敗・未実行と断定しない。Business Result、Execution Error、Observation、Unknownを維持する。
6. **最小配備**: Editor、Compiler、依存解決機構、管理UI、監督機構は、配備先で使うことが立証された場合を除いて含めない。
7. **更新可能性**: Web資産、実行ファイル、Wasm、ファームウェアごとに更新・撤回・復元方法を区別する。実行中の動的交換を全Targetへ要求しない。
8. **測定可能性**: ビルド結果のサイズ、初期化時間、常時/最大メモリ、初回処理遅延、必要な外部依存をTarget/Host/Builder条件付きで比較できること。

## 8. 実装構造の候補

実装を先取りして独立Kernelや第二Resolverを増やさない。既存の`internal/`にあるPlugin/Capability/Build/Runtimeの境界へ、以下の最小機能を追加できるか確認する。

```text
既存Application Driver / Target Projection
                 │
       各unitの論理的入口
                 ↓
    配置検査・Closure Planner   ← 既存Plugin/Capability解決とLock
                 ↓
       Builder Adapter
         ├─ go
         ├─ tinygo
         ├─ wasm component / JS
         └─ firmware / other toolchain
                 ↓
      unit別成果物と派生目録
```

Plannerの新規責務は、(a) unitへの配置、(b) Rootからの到達可能性、(c) Host/Target別の保持・除外判断、(d) 境界を越える契約の確認に限る。Provider解決、Contract Publisher Authority、Build Lock、Secret解決、実行時Permissionの正本はPlannerへ移さない。

Builder Adapterが複数あっても、`Builder != Plugin Contract != Runtime Host`とする。標準Builderをプラグイン化できても、ビルド済みの小さな成果物へそのBuilderを同梱する必要はない。

## 9. 最初の検証と採否の条件

**E0 — 既存構造への対応付け**: `order_entry`の第一Web Targetで、実際に使われる入口、Provider、生成資産、Runtime依存をread-onlyで列挙する。未使用と思われるものと、実際にバイナリへ含まれているものは区別する。

**E1 — 二つの配備単位**: 同じAccepted meaningから、Browser用とLinux/API用の二つの成果物を生成する。Provider選択、契約、Transport、環境Bindingの所在を明示し、Browser側へServer専用Providerが入らないことを検証する。WebのBundleがWasmである必要はない。

**E2 — 第二Targetと二層/三層の差**: 生成されたDesktopアプリを第二のConsumerとする。Studio Wails Hostは証拠に数えない。同じ意味を保ちながら、二層/三層でClient・API・Database実装の配備境界を切り替え、サイズ・起動・メモリ・権限・障害時の違いを記録する。

**E3 — 制約の強い実行先**: Go/TinyGo等で生成可能な小さな計算・センサー処理を選び、Flash/RAMの実測と未対応機能を記録する。実機動作、エミュレータ、合成テストを区別する。最初から全TEI Application DriverをMCUへ載せようとしない。

**負例**: 未固定の動的依存、BrowserからのServer Secret参照、非対応ABI、必須Capability不在、曖昧なProvider、契約不一致、静的結合した実装による権限越境、未確認のRemote副作用を、Buildまたは適切な実行境界で拒否できること。

最低限の成果は、**同じApplication意味から二つ以上の実行単位を構築し、プラグイン選択が異なれば実際の成果物の構成も変わることを観測する**ことである。コンパイル成功だけでは、バイナリ削減、性能改善、実環境での安全性を立証できない。

対象のCPU・OS・配布規約に依存する普遍的な数値目標はまだ置かない。まず同じScenarioとBuilder条件で、不要機能を含む基準構成と必要機能だけを選んだ構成の実測差を記録し、Target別の受入目標を決める。

## 10. 実行時に残す最小契約をさらに絞る

プラグインの登録と解決をビルド時に完了できるなら、配備後の実行単位へ汎用のPlugin ManagerやDSL解析器を要求する必要はない。ただし、境界を越える呼出しの意味まで省略してよいわけではない。

**単一プロセス内**では、生成された型付き関数や小さな関数表から実装を直接呼べることを最小形とする。内部呼出しを必ずJSON化したり、汎用RPCを経由させたりしない。論理Capabilityの契約同一性とProviderへの逆追跡は生成情報として保持する。

**別プロセス・別ノード間**では、既存の契約から導かれた通信Adapterを用い、少なくとも次を失わない。

| 境界 | 必要な情報 | 要求しないこと |
| --- | --- | --- |
| 呼出し先 | 契約・操作の同一性と対応する実装 | 全環境共通の単一RPC方式 |
| 入出力 | 型、NULL、順序、多重度、精度、版の扱い | 内部Go構造体の全環境への直輸出 |
| 結果 | 業務結果、実行エラー、未確認状態の区別 | Timeoutを業務失敗へ変換すること |
| 権限 | 呼出し境界ごとの認証、現在の認可、Resource権限 | Clientから受け取ったRole文字列の無条件信用 |
| 副作用 | 競合・再試行・重複実行と必要な観測 | 通信の再送を安全な再実行とみなすこと |
| 運用 | その実行単位に必要な初期化・停止・障害検知 | 全Pluginへの共通Supervisor常駐 |

組み込み機器の単一計算処理では、通信、Principal、Durable Invocationのすべてが常に必要とは限らない。反対に、外部リソースを書き換える機能には、構成を小さくしたという理由で必要な保護を省略しない。

### 最小の生成結果

Builder Adapterへ引き渡す候補は、配備単位ごとの`UnitBuildInput`という**考え方**に限る。新しい独立Lockや権限の正本を導入するものではない。

```text
UnitBuildInput (概念上)
  unit identity
  target/host/build profile reference
  entry points
  exact selected capability / provider references
  reachable implementation closure
  explicit cross-unit calls + adapter references
  required host imports
  existing BuildPlan / Artifact / Plugin Lock references

  -> builder adapter
  -> artifact digest + actual linkage / size / imports observation
```

これにより、**論理的に不要と判定した依存**と、**生成したバイナリから実際に除去されたコード**を別々に検証できる。一つの物理成果物へ複数のCapabilityを含めた場合、論理的な閉包だけでは一部のコードが除去されたとは断定しない。

### 現在の局所Referenceと限界

[配備単位別の依存閉包 Reference](../../reference/runtime-packaging/README.md)では、既存の契約解決とLockの結果が与えられているという前提で、Browser・API間の明示呼出し、DB用Host import、未使用Editor実装の除外を**読取り専用のGoコード**で試した。合成入力による17テストは局所環境でPASSしている。

これは完成した配備機構ではない。実`order_entry`のPlugin LockやTargetビルドへ接続しておらず、実バイナリの生成・最適化も観測していない。依存閉包の導出が成立しても、実コード削減や実行環境への適合は別のEvidenceを要する。

特にビルド前のProvider選択では成果物digestがまだ存在しない。その場合は既存の`BuildPlanFingerprint`を用い、ビルド後の`ArtifactDigest`と混ぜずに保持する。ビルド前に成果物digestを必須とすると、生成しようとしている成果物の識別子が未生成であるという循環が発生するためである。

## 11. 既存order_entryとの接続で見えた境界

[実Workspaceの読取り参照](../../reference/runtime-packaging/README.md)を追加し、`workspace.Load`から読み取ったApplication/Environmentと、既存`plugin.Catalog.ResolveEnvironmentWithLock`による確定済みPlugin集合の対応を観測できるようにした。実行時のProvider呼出しやDB接続は行わない。

`order_entry`では、開発環境について少なくとも次の参照を確認する必要がある。

| Application上の要件 | Environmentの接続 | Runtime Plugin Lockから解決する候補 |
| --- | --- | --- |
| `sales` / `sales_core` / read-write | `sales_core.dev`（PostgreSQL） | `tei.database.postgres@1.0.0` |
| `master` / `master_data` / read | `master_data.dev`（Oracle） | `tei.database.oracle@1.0.0` |

ここで得られるのは**DB接続の構成情報だけ**である。これをそのまま二層・三層アプリケーション用のBuild Planへ変換すると、次の情報が欠落する。

- `order_entry.submit_order`から`order_entry/sales.order.submit`のCustom Go宣言へは既存Resolverで対応できるが、その期待するAdapter名と**実際にコンパイル・リンクされるGoパッケージ**を結ぶ情報。
- Web Targetの画面・HTML・React等の資産と、それらを参照するHost側の実装。
- 認証、認可、環境別の秘密情報・通信方式・信頼境界の配置。
- 生成後のバイナリ内に、実際にどのGo実装と依存ライブラリが結合されるか。

さらに[Command Patternの読取り参照](../../reference/runtime-packaging/README.md)を追加し、`realization.ResolveCustomGoBinding`によって`order_entry.submit_order → sales.order.submit → order_entry/sales.order.submit`を取得する実装とテストを記述した。これはCustom Goの宣言を既存の解決境界で確認するものであり、生成後のバイナリに含まれる実装をまだ証明しない。

また、既存のformat 2 Plugin LockではBuiltin Pluginが成果物digestを持たず、**Plugin descriptor fingerprintだけで固定される**場合がある。そこで参照実装のBinding候補は、`PluginFingerprint`、`BuildPlanFingerprint`、`ArtifactDigest`を別々に扱うよう修正した。既存のPlugin定義指紋を、まだ正式化されていないCapability Contract fingerprintとして再利用しない。

この結果、現状の最小ランタイム設計で最も大きな不足は「別のプラグイン解決器」ではなく、**既存のApplication Pattern / Realization / Target実装から、配備時の実装参照へつながる正しい依存辺の取得**であると整理できる。

次の実証では、その参照辺が既存のDriver/Realization/Targetコードから導出可能かを確認し、導出できない部分にだけ追加の構成指定を検討する。現時点で実Workspace全体のpackage readinessは成立していない。試験結果は局所的なReferenceの範囲に限り、全Repositoryに対するexact-headテスト完了前に正式仕様へ昇格しない。

## 12. 判断を保留するもの

- `tei.deploy.yaml`という新たな正本を本当に必要とするか（既存Project/Profile/Environment定義への小さな追加で済む可能性がある）。
- Closure Plannerを独立したPackageとするか、既存のTarget packagingへ統合するか。
- 保持集合の宣言方式、Host Capabilityの型、配備目録のschema。
- Wasm Component/WASI版、TinyGo/その他Builderの標準採用。
- 動的更新と差分ファームウェアの共通化、全Target共通のサンドボックス契約。

この文書から直ちにCanonical種別、固定Target列挙、第二のLock/Resolver、大規模なRuntime Frameworkを新設しない。#259の第二Target比較、#943のBuild根拠、#951のCapability解決、#1016の案件構成の実証で不足が判明した部分だけを、Owner側で採用判断する。

## 13. 多言語・多方式の抽象化を支える構成原理（2026-10-17、評価候補）

本節は、ネイティブ実行、.NETアセンブリ、Javaクラス/JAR、Wasm、および将来の実行方式に対して、**同じ構成原理を保てるか**を評価するための論点である。既存の[ADR-0032](../adr/0032-composition-kernel-replacement.md)の目標アーキテクチャに従うが、ここで公開API・共通ABI・Plugin記述構文・Runtimeの固定列挙を決定しない。製品Runtimeへの移行完了も意味しない。

重要なのは、あらゆる実装を単一の`Invoke(JSON)`へ押し込むことではない。**提供・要求・明示的な接続・実行・結果・解放という共通する関係**を抽象化し、具体的な呼出し形式と配備形態を交換可能にすることである。

### 13.1 独立させる軸

| 対象 | 共通の構成で保持するもの | 特定方式に委ねるもの |
| --- | --- | --- |
| Service Contract | 識別、版、形、要求・結果・失敗・未確定、作用、Scope | 言語のクラス名・メソッド名・Wasm export名 |
| Consumer requirement | 必要な契約、局所名、必須性、多重度、明示Binding | Plugin packageの読み込み順 |
| Provider | 契約を実装する論理的な提供元と適合の根拠 | 個別の言語・ライブラリ実装 |
| Artifact | exactな内容、構築条件、依存閉包、来歴 | DLL/JAR/実行バイナリ/Wasmの形式 |
| Runtime Engine | 実行装置の識別、版、必要機能、Target適合 | CLR/JVM/Wasm Engine/OS loaderの実装 |
| Hostと配置 | 現在権限、資源、隔離、fencing、通信境界 | 同一Process、独立Process、RPC等の選択 |
| Generation・Effect Scope | 呼出し中の世代固定、準備、切替、後始末と失敗の識別 | 各Engineのインスタンス・ロード・解放 |

Plugin、Provider、Artifact、Engine、Hostの同一性は、一対一とは限らない。同じDLL/JARやEngineを二つのProviderが利用しても、契約適合と現在権限はそれぞれの利用側・Scopeで評価する。一方、同一PluginからTarget別の複数Artifactを構築してよい。Runtime Engineの照合は、そのEngineが読み込むDLL/JARおよび推移的な依存の検証に代わらない。

### 13.2 「接続」の三つの別の責務

1. **意味上の対応付け**：業務・分野固有の属性とService Contract項目が何を意味するか。項目名・基本型が一致しても同等性や受理を推定しない。[入出力対応付けの設計候補](PLUGIN_DEPENDENCY_IO_MAPPING_DESIGN_2026-10.md)の候補・受理の区別を維持する。
2. **Providerの選択**：どの契約・版・Scope・利用側をどの実装へ結ぶか。既存Composition、契約適合とLockを利用し、第二のResolver・Authorityを設けない。
3. **物理的な呼出し**：選択された実装へ値を渡すためのC#引数、Javaメソッド、Wasm import/export、JSON/RPC等の符号化。Adapterは業務意味の受理を代行しない。

Providerの準備時に決める構成パラメータ、要求ごとの入力、Hostが与える現在の実行文脈は別の寿命と所有者を持つ。主体・権限・秘密情報へのアクセスを、利用者やPluginが渡す通常の値から昇格させない。

言語境界では、数値の精度・単位、日時、null・欠損・取得不能・未確定、多重度・機密区分、および**Business Result / Execution Error / Observation / Unknown**を区別する。例外や通信断を業務拒否や未実行へ暗黙変換しない。任意JSONを通せることは、型・意味の契約適合の証拠ではない。

### 13.3 相互作用の差を失わない

純粋な計算、長期間維持するデータ接続、イベント購読、UI部品の寄与は同じ呼出し方式ではない。単発`Invoke`へ強制すると、購読解除・参照の寿命・資源の所有・集合の意味を失う可能性がある。一方で、相互作用の種類をKernelの固定`CapabilityKind`へ際限なく追加することも避ける。

共通化するのは**契約に対する提供・要求・選択・作用域・来歴の規律**であり、相互作用ごとの型付きの詳細はService ContractとProvider/Adapter側に残す。この考え方で異なる相互作用を十分に記述できるかは、具体例と負例で評価する。ここから新しい万能Interfaceや固定種別を採用しない。

### 13.4 方式別に残す検証事項

| 実行形態 | 最初の実証候補 | 未確認の差 |
| --- | --- | --- |
| Native | 静的結合または独立実行ファイル | 実Artifact、OS権限、更新単位 |
| .NET | 独立した.NET HostでDLLをロード | Runtime版、DLL・依存閉包、ロード/解放、例外、隔離 |
| JVM | 独立したJVM HostでJAR/クラスをロード | JVM版、依存JAR、ClassLoader、解放、例外 |
| Wasm | 既存WASI参照実装と適合するEngine | Module/Component/WIT、Host import、資源制約 |
| UI/イベント | Target固有のHost/Adapter | 単発呼出しではない相互作用、破棄/購読解除 |

.NET/JVMの初期試験は、既存の[Process接続Reference](../../tools/reference-composition-process-bridge/README.md)で使用する選択・権限・Artifact照合の責務を再利用できるか調べる。ただし、そこで使われる**配備処理用の旧Process JSON carrier**を全言語・全Serviceに共通する公開ABIにしない。`dotnet`/`java`実行ファイルの指紋だけで、読み込むDLL/JARの指紋が確認済みとみなさない。

### 13.5 汎用化を反証可能な形で評価する

既存の[Composition Kernel](../../reference/composition-kernel/README.md)、[項目対応付け](../../reference/composition-io-mapping/README.md)、[実Process接続](../../tools/reference-composition-process-bridge/README.md)、[配備依存閉包](../../reference/runtime-packaging/README.md)のReferenceを使い、当初は業務意味の受理を要しない**副作用なしの独立した読み取り専用Contract**を用いる。以下は設計評価の順序であって、新しいGateではない。

| 段階 | 比較対象 | 汎用性が成立しない兆候 |
| --- | --- | --- |
| A | 同じContractとConsumerで二つのProviderを明示的に切替 | 新ProviderごとにCore固定列挙や固有分岐が増える |
| B | Nativeと別Process上の.NET/JVM候補による同じ要求・結果 | 実装言語のメソッド名が意味の正本に入る |
| C | 欠損/null/Unknown、精度、Shape変更、Scope違い | 黙った値変換、旧受理の再利用、曖昧選択 |
| D | Artifact・Runtimeだけ交換しHost権限失効と起動回数を照合 | 拒否後に子Process/Providerを起動する |
| E | 購読または長寿命サービスへ相互作用を変える | 解放・多重度・寿命が一回呼出しに隠される |
| F | 同じ論理的要求を静的Targetと動的Hostへ具体化 | 静的成果物にも不要なLoader/管理機構が残る |

追加の負例候補は、非対応Runtime、異なるDLL/JAR、依存閉包の漏れ、応答の偽造、通信断の結果不明、世代切替、drain/cleanup失敗である。既存Referenceの合格と将来の.NET/JVM製品対応、正式Host権限、常駐サービスの無停止更新は別の証拠とする。

採否では対応言語の数より、**新しい契約・実装・Runtime/Targetを追加しても意味の正本やKernelの抽象規律を変えずに済むか**を重視する。Core変更面積の分析には既存の[#1251](https://github.com/hat47x/tei-platform/issues/1251)の観点を再利用する。

### 13.6 責務と未決を維持する

Contract/Conformanceは[#951](https://github.com/hat47x/tei-platform/issues/951)、Composition/Generationは[#1223](https://github.com/hat47x/tei-platform/issues/1223)、Artifact/Buildは[#943](https://github.com/hat47x/tei-platform/issues/943)、Runtime/Target/Trustの差は[#259](https://github.com/hat47x/tei-platform/issues/259)に戻す。Providerの申告、契約の意味上の受理、Artifact検証、現在のHost権限を相互に代用しない。

標準Adapter、CLR/JVM直接ホスティング、Wasm Component/WIT、製品用の項目対応付けの記述構文、相互作用ごとの型付き契約は**未決**である。これらを新しい固定種別、第二のLock、独立Resolver、公開仕様として先取りしない。

## 14. 業務FunctionとBuild Unitの割当・Job入口（2026-11-03、評価候補）

**状態：設計候補／NOT RUN／Owner未受理。** X-TEA XF000の対話画面・定時起動を起点に、同一の型付きFunctionをUIボタン・CLI・cron・Web・既存Host/EKI等から呼び出す要求を、本書の配備単位に接続する。Goの単一実行ファイルを標準的な生成物としつつ、必要になった時点で設定を最小限追加し複数バイナリへ分割できることを狙う。TEI独自の共通Hostや常駐Schedulerは追加しない（[ADR-0043](../adr/0043-target-native-artifacts-without-tei-host.md)）。

### 14.1 論理構造、ビルド、配備、入口の独立性

| 区分 | 所有する内容 | 誤って同一視しないもの |
| --- | --- | --- |
| System／Subsystem／Function／Action／Data Rule | 業務・分野の論理構造とAccepted Meaning | フォルダ名・Go package・実行ファイル名 |
| Invocation Binding | UIイベント／CLI／時刻／外部要求とFunctionの引数対応 | 実行権限・物理配置 |
| Build Unit（候補名） | 特定Targetの入口集合、必要実装の依存閉包、Builder入力 | Subsystem、Deployment Unit |
| Artifact／ArtifactSet | exact bytes、member、ビルド・Source・Toolchain来歴 | 起動可否、現在の認可 |
| Deployment Unit／Entry Point | 配置・更新・停止の単位、実際のCLI/HTTP等の呼出し入口 | 内部に含まれるFunction全件 |

一つのFunctionが複数Build Unitに含まれてよい。一つのArtifactを複数Deploymentで使用しても、そこでの主体・Environment・権限・寿命は別。Functionがバイナリに静的結合されても、公開Entry Pointに指定されなければ外部から呼べない。PluginとArtifactの多対多は既存[#1097](https://github.com/hat47x/tei-platform/issues/1097)/[#1102](https://github.com/hat47x/tei-platform/issues/1102)へ戻し、新しいBuild Engine／Artifact Lockは作らない。

### 14.2 設定なしから必要時だけマルチビルドへ

1. **単一ビルドの既定**：Build割当ファイルは不要。既存のApplication/Targetに指定された起動入口から、適合する実装の依存閉包を生成する。全Functionの自動公開や全Pluginの無条件同梱を意味しない。入口未指定・未配置・非対応は診断して黙って省略しない。
2. **有効化時のPreview**：ビルド管理PluginのCUI/Studioから、Function ID・所属Subsystem・Source・起動契機・現在のTarget依存・構成結果を読み取り専用で一覧化する。既定と同等の最小設定を提案する。明示確認後にだけProject側の構成差分を保存する。
3. **一覧と設定の分離**：機能一覧表はモデルと割当設定からの再生成可能な投影。列にはFunction ID、業務所属、公開入口、root/closure区分、適用Build Unit、割当根拠、Source/Model版、conflict/unresolvedを含める。Markdown/CSV/Studioで閲覧してよいが第二正本にしない。表編集は設定変更Intent→Preview→保存とし、業務Functionの意味を直接変更しない。
4. **設定ファイルは小さく**：最初から全Functionの個別行を列挙しない。default、Namespace/業務Scope/Source Path規則、例外Override、外部公開する入口だけを記す。新しいファイル名・schema（例えばtei.build.yaml相当）は未採用であり、既存[案件構成](PROJECT_PLUGIN_COMPOSITION_DESIGN.md)やWorkspace設定への最小追加で済むか先に評価する。

説明専用の構文案（**正式DSLではない**）：

~~~yaml
build:
  default: app
  units:
    app: { target: go/linux-amd64 }
    batch: { target: go/linux-amd64 }
  rules:
    - match: { namespace: "sales.jobs.*" }
      unit: batch
  overrides:
    sales.recalculate: [batch]
  entrypoints:
    - { function: "sales.jobs.daily-close", unit: batch, surface: cli-job }
~~~

未指定のFunctionをどのようにroot化するかは、既存のApplication/Action契約と照合して決める。Build Unitに含まれるData Ruleや依存Functionを外部公開しない。Secret、cron時刻、認証トークンは業務意味やBuild設定に埋め込まず、実行・配備時の契約に委譲する。

### 14.3 割当と依存閉包の決定規則

- 評価候補の順序は「Function ID個別指定 ＞ Namespace/業務Scope ＞ Source Path/Folder ＞ default」。同じ優先順位で異なるunitが選ばれた場合は失敗とし、宣言順・名称順で勝者を決めない。同一結果の重複は正規化し、**複数unitへの明示包含**と曖昧な割当を区別する。規則の最終優先順位はOwner判断待ち。
- Folder規則はソース整理に過ぎない。Source Pathの移動で割当が変わった場合は差分を警告し、Function IDとAccepted Meaningを勝手に変えない。未解決Source参照や旧Model Revisionでは配置を確定させない。
- rootから到達するData Rule、標準Operator、共通実装、Providerを依存閉包として導出する。動的呼出し・Reflection・外部API越境・Go importの到達性が未知なら「依存なし」と推定しない。隔離・認可・トランザクションを跨ぐ呼出しを自動的に同一プロセスへ静的結合しない。
- Build Unit、ArtifactSet、公開Entry Point、Deployment bindingを、Source/Meaning/Rule/Config revision、Builder/Generator/Toolchain/Target/Plugin版と逆追跡可能にする。新旧Artifact・一部member欠落・A→B→A・同一Source別Toolchainについて、成果物再利用／検証根拠再利用／現在資格を分離する（[#2600](https://github.com/hat47x/tei-platform/issues/2600)）。

### 14.4 同じJob Functionの外部提供

単一Goバイナリは、例として「sales serve」と「sales job run sales.jobs.daily-close --business-date 2026-11-03」の複数入口を提供できる。分割した場合は「sales-batch job run ...」として同じFunctionへ到達させる。これらは正式CLI構文ではない。cron/OS Task Schedulerは生成したCLI入口を起動するだけでよく、TEIの常駐を要求しない。実際の定時契約（タイムゾーン、同時起動、回復、サービス主体、ログ保存）はDeployment側で扱う。

Webでは、認可済みRoute→Action/Function→同一プロセス実行または外部Job Providerのどちらも許す。ただし任意Function名をURLから受けて任意shellを実行する公開入口は作らない。引数の業務上の型・既定値・検証はFunction側、CLI/HTTPの文字列表現はAdapter側。必要な利用者入力がない非対話Jobは「事前引数あり」「明示的な中断／再開契約」「unsupported」を区別し、Swing/Wailsの画面を前提にしない。

Jobを要求したというACK、実際の起動、進捗、業務結果、DB Commit Receipt、独立Observationは別々に観測する。通信断・二重依頼・認可失効・旧Artifactでの起動では、OperationID/Attempt/実ロード版/現在Grantを照合し、不明な副作用を「失敗＝未実行」へ変換しない。権限とScript信頼境界は[#2504](https://github.com/hat47x/tei-platform/issues/2504)/[#1174](https://github.com/hat47x/tei-platform/issues/1174)、外部起動・回復は[#1130](https://github.com/hat47x/tei-platform/issues/1130)、EKIの実Task/Lease/資源AuthorityはEKI側の担当とする。

### 14.5 最小の反証Fixture（すべて未実行）

| ID | 条件 | 独立Expected／停止条件 |
| --- | --- | --- |
| MB-01 | Build設定なし、Web/Jobの二つの入口 | 単一Goバイナリから両方に到達し、内部Functionは非公開。未配置は未解決として診断 |
| MB-02 | 同一モデルを1→2→3バイナリに変更 | Function/Rule/Scenarioの意味は不変。Build/Artifact/Deploymentの識別は別 |
| MB-03 | 100 Function、Namespace規則1件＋個別Override1件 | 一覧の根拠は決定的。100件の設定コピー不要。表示表は再生成可能 |
| MB-04 | 規則の同順位衝突／Source Path移動／古いModel版 | conflict、割当差分、staleを独立検出。勝手に採用しない |
| MB-05 | 同じFunctionを2単位に包含、Data Ruleは依存で共通 | root/closure/publicの区別、実Goの不要package/linked symbol除外を確認 |
| MB-06 | CLI・cron相当・Webから同一Jobに型付き引数 | 業務結果は同じ意味、Host主体/認可/Transportは別。拒否時Job起動0件 |
| MB-07 | バッチに対話必須、またはSecretがBuild設定に混入 | 不適合を明示、CredentialをArtifactや一覧に露出させない |
| MB-08 | ACK消失、二重要求、旧Artifact/Providerで実行 | Unknown保持→照会／限定回復。無条件リトライ・旧権限での実行を拒否 |
| MB-09 | SHO説明のみ変更／Rule改版／ArtifactSet一member欠落 | 必要再生成、Artifact再利用、Evidence再利用、現在資格を別に判定 |

期待値のテスト正本をここで作らず、[#1180](https://github.com/hat47x/tei-platform/issues/1180)の保存済みScenarioと[#2514](https://github.com/hat47x/tei-platform/issues/2514)の独立Oracleに結び付ける。

### 14.6 段階計画と所有境界

- **P0 読取り専用**：既存Function/Operation Surface、業務IDとPath、#1167のGo入口、#1097/#1102のArtifact束縛を現在版で棚卸しし、MB-01〜09のExpectedを固定する。
- **P1 Headless実証**：同じMeaningから1/2/3単位の入口集合と依存閉包を生成し、実Go build、link後内容、ArtifactSet、CLI入口まで確認する。単にmanifestを分けただけで実際に分割されたとはしない。
- **P2 管理コスト**：設定なし→Preview→最小設定の初期生成→派生機能一覧→例外変更→差分診断をCUI先行で測り、同じ契約をStudioへ投影する。手作業の設定行数、重複、誤割当、影響Build数、編集から利用可能までの時間を比較する。
- **P3 外部起動**：実Web HostとOS Scheduler相当を用いて同じJobを呼び出し、認証・進捗・取消・未知Commit・Recoveryを分けて観測する。実EKI接続は別受入。

担当は配備／Target[#259](https://github.com/hat47x/tei-platform/issues/259)/[#1182](https://github.com/hat47x/tei-platform/issues/1182)、案件構成と初期設定[#1016](https://github.com/hat47x/tei-platform/issues/1016)、Job入口[#1130](https://github.com/hat47x/tei-platform/issues/1130)、Script/Macro契機[#2502](https://github.com/hat47x/tei-platform/issues/2502)、Build閉包・増分Evidence[#943](https://github.com/hat47x/tei-platform/issues/943)/[#2600](https://github.com/hat47x/tei-platform/issues/2600)、設計成果物としてのFunction一覧[#2482](https://github.com/hat47x/tei-platform/issues/2482)。第二のHost、Scheduler、契約Resolver、Function意味の正本を作らない。

**Owner判断待ち**：Build Unitの正式な型と名称、設定ファイルの配置、暗黙のroot集合、割当規則の優先順位、複数unitにおけるFunction公開方法、OS Schedulerの設定生成範囲。実マルチビルド／ジョブ起動／Studio初期設定UI／実認可／本番回復のE2EはNOT RUN。本節を現行実装の合格として扱わない。

## 15. 複数Buildの配置決定・実入力・公開資格を分離する（2026-11-04、横断照合）

**状態：設計補強案・新しい横断試験はNOT RUN。** §14の単一ビルド既定／必要時のみ明示分割という方針を維持し、2026-11-04に進んだ[#1130](https://github.com/hat47x/tei-platform/issues/1130) BU-01〜09、[#2600](https://github.com/hat47x/tei-platform/issues/2600) IB-15〜18、[#2595](https://github.com/hat47x/tei-platform/issues/2595) EB-01、[#2604](https://github.com/hat47x/tei-platform/issues/2604) AP-01〜07、[#2787](https://github.com/hat47x/tei-platform/pull/2787)のEKI OpenAPIとの比較結果と接続する。これらの試験正本・Build Engine・Artifact Store・Job Coordinatorを本書へ複製しない。

### 15.1 配置の意図から利用可能な入口までの五つの照合点

| 段階 | 観測すること | ここから推定してはいけないこと |
| --- | --- | --- |
| **I0 業務Functionと公開意図** | Accepted Function/Action/Rule ID、業務所属、型付き入出力、公開が必要な起動契機、必要なInteractionの有無 | Folder名からの業務意味、Functionの自動公開、現在の実行Grant |
| **I1 Build選択** | 単一既定／明示unit、選択規則の一致理由、root集合と推移的閉包、Target適合、除外対象、未配置・競合 | 選択されたというだけでビルド成功、実ソース消費済み |
| **I2 実ビルド入力とArtifact** | required/declared/resolved/parsed/actual-consumed Source集合、負の依存、Builder/Toolchain、実入力bytes、生成実ファイル、ArtifactSet/memberのdigest | 宣言Source digestから実ビルドSourceを自動証明、ファイル作成から一式公開を自動証明 |
| **I3 公開と受領** | どのArtifactSetのどのmemberが、どのDeployment/実入口に配備され、実Consumerがどのbyte/契約版を読んだか | exit=0、manifest complete、単独digest一致から実配備の版一致・公開成功を断定 |
| **I4 現在の呼出し・業務効果** | Principal/Service Identity、Context/対象、現在Grant、Transport、OperationID/Attempt、Provider Receipt・独立Observation | CLI/HTTP ACK、EKI Task success、旧Artifactの過去PASSを現在の業務Commit資格へ昇格 |

これらは**既存Evidenceへの用途別照合票**であり、新しいProduction schema、実行状態機械、五段階の許可エンジンではない。単一BuildでもI1〜I4の検証は省略されない。現行のファイル一覧が完全でないときは、存在するファイルだけで「必要なソースがすべて含まれる」と判断しない。Provider内部Source path/Raw fingerprintは、許可のないStudio/SHO/AI Consumerに出さず、公開可能な範囲の不足・診断だけを投影する（[#2812](https://github.com/hat47x/tei-platform/pull/2812)）。

### 15.2 「Function一覧」の役割をさらに限定する

一覧は**配置検討・診断のための導出表**とし、Function正本／Build Intent／実Artifact証拠を一つの手編集スプレッドシートへ潰さない。初期の表示列は、Function ID、業務所属、Logical Entryの種類、Root/Dependency/Non-root/Unresolved、推奨または明示Build Unit、割当根拠とrule revision、必要Target能力、公開状態、実証済みArtifact参照の有無。実ファイルパス・秘密情報・Provider内部参照を出すかは表示先の開示権限で制御する。

- **暗黙のdefaultは「全ソースを無条件に公開する」ではない**：受理済みApplication/Action/Jobの公開要求をroot候補として取り出す。UIだけで使うFunctionはその契機、未接続のFunctionは非rootとして一覧に残す。必須rootが解決不能ならビルドを拒否し、意図的に非公開の関数は警告または情報表示に留める。root発見契約が定まらない場合は空の成功ビルドを作らず、必要な選択の不足を診断する。
- **割当規則の優先順位は§14.3では未採用候補**：安定Function IDの明示指定はその対象に対して優先する案とし、Scope/Namespace/Pathの複数ルールが異なる単位を選ぶ場合は、採用済みの明示優先条件がない限り衝突。最長prefixやファイル走査順で勝者を決めない。Source移動はFunction Identityを変えず、配置差分を表示する。
- **同じFunctionの二重配置**は、明示したrootまたは両unitの依存閉包から得られる包含として説明し、暗黙の公開や独立した業務Ownerの生成とはしない。Data Ruleは意味・phase/Effectの正本を参照し、各unitでの実現方法はTarget/Provider適合で検証する。
- **設定自動初期化の安全条件**：Build Pluginはread-only提案→内容diff→保存Intent→Projectへの明示適用を分ける。既存設定ファイルがあれば上書きせず差分/競合を返す。再初期化時も同じ入力に対して正規化した設定結果が同じで、選択・改版・移動の理由を一覧に残す。初期配置で業務Meaning、Owner受理、Entry Point公開、OS Schedule登録を自動変更しない。

### 15.3 実Go Sourceと再利用根拠の強化

[#2819](https://github.com/hat47x/tei-platform/pull/2819)は**現行Go Generator Selectionの宣言Sourceと実ビルドSourceが不一致でも通る**ことを、test-only実Goで記録してmainへ統合済み。これは実装の安全な拒否を立証したのではなく、残る問題の実証である。Function→Build Unitの割当を正しく解決できても、最後のコンパイル入力が別ファイルなら正しい成果物とはいえない。

Build Unitごとに、選択したroot・依存閉包・実Generator/Builderが**実際に消費した全関連入力**の束縛を確認する。否定依存・未読込必須Source・Queryで出力されなかった行／Predicate依存も、必要なClaimに応じて扱う。完全な入力閉包が示せない場合は当該BuildのskipやArtifact再利用をunknownへ保留する。バイナリの同じdigestと、生成器の資格／Owner受理／現在Grantは別物。Go packageの静的import closure、link後byte/シンボル、呼出し可能な入口の確認を分け、単なるmanifest-only・ソース上の推論を配布物からの不要機能除外のPASSにしない。

増分ビルドで**一つのRuleが両unitへ到達するケース**、Source移動のみで意味は同じだがpath由来割当が変わるケース、Sourceは同じでToolchainのみ変えるケースを比較する。正しいBuild省略、Artifact再利用、Evidence再利用、現在資格はそれぞれ独立の結果として記録する（#2600）。

### 15.4 複数unitの公開と非同期ジョブ完了

実ファイルの生成はリリースではない。複数unitを更新する場合、期待するFunction/Contract revisionに対する**互換な配備組合せ**を診断できる必要がある。例えばWeb/APIがv2のJob要求を出し、Batch側がv1のままなら、受理可能な互換範囲を既存Service ContractとConsumer側で確認し、未確認なら切替えない。これは新しいRelease Registry/統一Txを要求するのではなく、既存ArtifactSet/Deployment evidence/Consumer observationの比較である。

一つのArtifactSetでも、旧manifestと新memberの混在、書込途中失敗、ファイル名衝突があると、外部Consumerが異なる世代のファイルを読む可能性がある。#2604のAP-01〜07を公開責務の正本として参照し、Build選択が正しかったことを公開・受領の成功として誤転記しない。Releaseする対象unitがN個なら、各unitの生成、検証、公開、現実のConsumer読取と契約適合を**別々の状態**として記録する。

ジョブはさらに「起動要求・受付」「実起動」「終了・Result」「実副作用の確定」「業務完了」を分ける。OS cronであっても、重複起動・途中中断・再送・旧版Jobの回復ではOperationID/Attempt/同じ要求の再送か新しい業務実行かを区別する。Webサービスからの呼出しは許可されたroute/actionだけ。長時間実行のためだけに共通TEI常駐Hostを新設せず、既存OS Scheduler/Host/EKI等の実行主体を選ぶ。

**EKI P0への直接変換は現時点で一般Jobの保証にならない**：Draft #2787の版固定OpenAPI比較では、TaskRequest.parametersのWorker配送、effect_modeの副作用種類、Executor版のAdmission検査、output artifact body取得・業務Commitの証明などが未充足。合成19件PASSは実Task/Worker/DBの成功ではない。対応不可の必須Job属性がある場合は、部分的なwire shape一致でも、表面上の成功とせずnot-representable/unsupportedとして止める。限定した副作用なしJobの静的試験から開始し、実WorkerやResultの照合は別途必要。

### 15.5 他チャットの設計を圧力条件へ接続（新しい独立テスト正本は作らない）

| 差分観点 | 先行するFixture／Owner | 今回Build側で見る停止条件 |
| --- | --- | --- |
| **実Source差／必須Source未取得** | #2595 EB-01〜04、#2600 IB-15〜18、#2822/#2823 | 必須Source欠落・宣言/実消費差・不明閉包のBuild再利用を資格ありとしない |
| **業務意味と起動配置の差** | #1130 BU-01〜09、#1180 Scenario、#2514独立Expected | 同じMeaningでも旧Buildの未公開rootや未対応Functionを見落とさない |
| **公開の一式性と旧ファイル** | #2604 AP-01〜07 | 成果物生成済みでもConsumerが旧member・混合版を読んだら成功としない |
| **EKI委任の意味欠落** | Draft #2787 WP-01〜19、#1130 SJ/LR | 必須引数・副作用・版・結果bodyの欠落をTask成功へ読み替えない |
| **Desktop/WebのActionとData Rule** | #997/#1824、#2507、#1129、#2502 | 画面の有効表示、Wailsの接続、Script存在を実実行Grant/Business Commitにしない |
| **隔離・秘匿と実ロード** | #2504、#1370、#1174、ADR-0044 | 同一バイナリ同居による権限共有を許可済みMacroの隔離保証にしない |

**横断受入の終了条件（提案）**：固定した一つのAccepted Function/Rule/Scenarioを使い、①単一Goバイナリ、②Web/API＋Batchの2unit、③Job別の3unitで、全必須rootの配置、実消費入力、ビルド済みArtifact、公開入口、Target適合、外部Consumerの実受領、権限・結果を別列の証拠として比較できること。必要なら単一Buildが最善であるとの結論も許す。自動設定の記述行数・人が修正した箇所・Build latency/size/起動遅延・false reuse・false invalidationを測るが、設計だけで高速化を主張しない。

**採用前の判断**：§14.3の優先順位・Build構成ファイル名・Root発見方法・Release切替契約は未採用。まず既存#1167/#1182と実現在のCLIを調べ、新しい構成Schema・コード生成入口・公開APIの実装が不可避かを判別する。#1130のBU、#2600のIB、#2604のAP、#2787のWPを重複コピーして新Gateにしない。追加した本文は設計・責務接合のみであり、実マルチBuild・Job起動・Web/EKI/Studio/SHO/実DB/Owner受理はいずれも**NOT RUN**。
