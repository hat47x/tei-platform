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
| `master` / `master_data` / read | `master_data.dev`（検証基準 Database） | `tei.database.oracle@1.0.0` |

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

<!-- tei-extension-selection-three-gates-20261106 -->
## 14. 拡張方式の三段階選定：Plugin化の必要性・契約・物理実現（2026-11-06、設計／NOT RUN）

2026-11-05の[拡張方式選定表](https://github.com/hat47x/tei-platform/issues/951)と、大プラグイン主義への反証・高凝集Bundle/未使用機能の非干渉（#1251/#1016/#2600）、Data Operationsと多Provider（#2622）、DSL/Operator（#988）、外部Consumer（#2390）、実Effect（#1174）を**既存の§13の方式比較に接続する追補**。これは採用前の意思決定補助であり、正式Plugin ABI/Schema、第二Resolver、独自Host、新Gateを追加しない。現行の採用済み境界はADR-0043（対象Hostを再利用）、ADR-0044（認証の実現をCoreに強制しない）、ADR-0032（契約中心の目標）へ戻す。

### 14.1 正しい順序：まずPlugin化を反証する

1. **Plugin境界の要否**：実Consumerごとの独立契約・選択/版・配布更新・権限・障害隔離の必要性を調べる。純粋な関数、固定の業務Rule、描画設定は、標準ライブラリ・明示Data Operations/Rule・設定のままで十分な場合がある。Pluginであること自体を価値や拡張可能性の証拠にしない。
2. **何の契約を追加するか**：Language/Parser-Lowering、Operator Semantic、Service/Capability Provider、Target Generator、外部Service/Transport-Lifecycleのどれか。外部Processは**責務の種類ではなく物理接続方式**でもある。同一Providerが静的Goと独立Processの両方で実現されることを許容する。
3. **物理実現を選択**：静的合成、動的ロード、外部Process（必要なら既存Host libraryやWasmをそのTargetの実現として比較）。単一Go binaryを第一候補とし、Target/Hostに動的更新・独立障害境界などの実要件がある場合だけ追加機構を検討する。生成時に動的Pluginを使っても、生成アプリの実行時にそのLoaderを含める必要はない。

### 14.2 拡張対象×最小契約×不適合条件（設計判断の表）

| 対象・例 | Pluginでなくても成立する候補 | 独立契約が必要になる条件 | 主要失敗影響・棄却基準 | 既存担当 |
| --- | --- | --- | --- | --- |
| 構文/DSL（pipe、Markdown/表記糖衣、別言語） | 表層糖衣→同じCanonical/Operatorへの静的Lowering | 文法Producer/Parser/Lowererを独立選択・版固定し、元Source/Source span/診断・編集往復が必要 | Parserの登録順で意味が変化、Source原構文/位置喪失、同名Roleから意味推測、Semantic Acceptanceすり替え | #988/#2619 |
| Operator（filter/aggregate・数量計算等） | 純粋Go関数/標準Library、Data Operationsの合成 | 名前・意味・型/粒度・NULL/精度/順序・Effectを版付きで多数Consumerが再利用し、Provider差し替えが必要 | 型・戻り値だけ一致してNULL、bag、多重度、単位、丸め、欠損、read scopeが異なる。独立Expected不在 | #988/#2622 |
| Data/Service Provider（DB、OData等） | 単一Target向けの明示Adapter、設定済み既存Driver | 実装/DB方言/Capability（Read・PATCH・Replace・batch・CAS・Receipt）と現在のHost Authorityの選択が分離 | 曖昧な対象、無認可、partial 0件→不存在、batch=原子性の推測、部分Commit/ACK-lossの復旧不能 | #2622/#951/#1174 |
| 生成コード／UIの案件拡張点（Hook・公開Slot） | 生成物と分離した案件Source、#2090宣言済Hook、#997 Kit Layer | 複数案件/Consumerへ独立実装を配布し、Hook/Slot契約や更新責任を別にする必要が実証された場合 | 再生成で案件差分消失、基底Hookの引数・呼出順変更を無検出、未公開DOM/生成ファイル直接編集、UI側からHost認可を迂回 | #2090/#997/#2785 |
| Target Generator（Go/Wails、Java/.NET/Wasm） | 既存Generator内の明示設定、変換関数 | Target固有Artifact/Builder/Toolchain・生成器更新・実現Capability・必要依存を独立に選択 | CanonicalにTarget APIが逆流、未選択コード/秘密混入、実Targetが必要能力を欠く、生成物・実ロードByte不一致 | #1121/#259/#943 |
| 外部Process/Service連携 | 同一Process内の関数またはProvider、既存Host library | 異言語/異更新周期/隔離/既存外部製品が独立配置を要求し、Transportとlifecycleを明示 | Single-binary厳格要件違反、起動/停止時の権限漏れ、結果不明を失敗扱い、途中確定の無条件再送 | #1145/#2390/#1223/#1174 |
| Owner固有業務Rule | Data/Function/Businessの受理済Rule、明示Data Operations | 同じRuleを異Owner/Targetで独立配布すべき**具体的Consumer**がある時だけ再検討 | RuleをPlugin設定へ移し、業務意味の第二正本/Providerの自己申告へOwner受理を委譲する | #1129/#2622/#267 |

Pluginの**論理境界と高凝集Bundleの配布境界は1:1ではない**。共通Interfaceと複数Implementationを一Bundleへ同梱できるが、同梱≠Selected≠Buildで実消費≠Linked≠Exposed≠Current Grant≠Invoked≠Effect。Root packageの強制import/init登録や全実装のリンクによって未使用コードが成果物へ混入した場合、「未使用Plugin非干渉」は未達と扱う（#1251 PBN/PB、#1167、#2600）。

### 14.3 物理方式の選択：同じ契約を条件付きでLoweringする

| 比較軸 | 静的合成（既定候補） | 動的ロード（必要時のみ） | 外部Process（必要時のみ） |
| --- | --- | --- | --- |
| Single-binary | Go単一成果物/不要依存除外に有利。生成時Pluginは同梱不要 | 別ロード実装・Engineの存在を要求する場合、厳密な単一配布実行物に非適合 | 別実行物が必要なら厳密な単一バイナリと非適合。単一インストーラか否かは別要件 |
| Provenance | exact Source closure + selected Contract/Provider/Generator/Toolchain/flags → actual Binary/member | 上記に加え、ロード物/推移依存/ロード時点とunload結果 | 上記に加え、実起動Executable/子Process依存/Protocol+起動Context |
| Compatibility | Build時に一部検出可能だが意味/Owner/Host資格は別 | Loader/ABI/Engine/旧新世代共存の検証を追加 | Wire semantics、Client/Server混在版、timeout/再接続・二重送信を検証 |
| Capability/Trust | static linkしても実行Grantにはならない | Plugin自己申告/署名・ロード成功はHost許可ではない | 別Processも自動sandboxではない。OS/Host実隔離・現在Grantを確認 |
| Partial effect | 同一Process内でも永続化後の失敗があり得る | 旧世代in-flightと切替前後の作用/解放が問題 | ACK-loss・外部不可逆作用・部分Commitとread-after-write資格が問題 |
| Update frequency | 再Build/配布が必要。実時間で評価 | 無停止が必要なら世代併走/Request pin/Drain/cleanup/rollbackの実測が必要 | プロセス単位の独立更新可能。ただし旧版Request・外部状態・Effectを区別 |
| Test isolation | 独立テストプロセス・Buildで隔離。in-process panicやglobal stateも注入 | ローダー版競合・漏れ・解放失敗/再ロードを注入 | kill/timeout/partition/応答偽造/再起動と二重効果を注入 |

**棄却/保留**：Sourceの実消費・契約fingerprint・意味の受理範囲・選択Provider・必要Target Capabilityが未特定なら候補確定を止める。副作用ある拡張で先行Step確定、後続Step blocked、Commit応答喪失を区別できないならProduction採用を止める。動的ロードが「更新しやすそう」という理由だけなら採らない。実単一Binary要件下で外部実行物が必要なら別方式を検討し、要件を暗黙に緩和しない。

### 14.4 Ownerへの質問と方式選定の停止線

| 確認事項 | 判定に必要な証拠・質問 | 資格不足なら |
| --- | --- | --- |
| 機能価値と境界 | 「既存標準関数/Rule/設定で実現不能な点」「別Consumer・別Ownerが存在するか」 | Plugin化を先取りせず標準機能案へ戻す |
| 意味とSource | 「何が業務意味の正本か」「元構文・Source span・Model/World・Rule版・実消費byteは何か」 | read-only説明まで。意味自動受理禁止 |
| 呼出と権限 | 「どの主体・Resource/Scopeで実行」「Capabilityを誰が実効強制」「Included/Exposed/Grantedは別か」 | protected Invocationを拒否/保留 |
| 副作用と回復 | 「何が原子的か」「一部確定、ACK喪失、重複要求時の独立Receipt/Readは誰の責務か」 | 業務Writeの本番採用を停止 |
| Target/配布 | 「厳密な単一バイナリか、複数Artifactの単一製品でよいか」「対象OS/Hostは何か」 | 要件と矛盾する方式を棄却 |
| 更新・障害隔離 | 「更新頻度、許容停止、ロールバック、信頼できない実装の隔離が本当に必要か」 | 過剰なLoader/Processを採用しない |
| 試験・費用 | 「独立Expected、fault injection、対象exact Source/Build、保守/配布費用の比較が可能か」 | 設計候補に留め、優劣/Production資格を宣言しない |

### 14.5 最小の反証Fixture（既存の検証基準/Ownerへ割付）

新しい正本Scenarioを作らず、11/5選定表の EX-01〜08 を下記の既存ケースへ**対応付ける**。番号は計画上の参照であり新規Go試験のPASSではない。

| 参照 | 第一の対照と追加の負例 | Owner／独立証拠 |
| --- | --- | --- |
| EX-01/02 Language+Operator | 同じSourceでParser pinを変える、同型OperatorのNULL/精度/Role差。原Source/診断とSemantic acceptanceを分離 | #988/#2639。Source byte・実Parser/Operator版・独立Expected |
| EX-03/04 Provider+Partial effect | Read-only ProviderでWrite要求を拒否。A Commit後B blocked、receipt欠落でRun全体effect=0としない | #2622/#1130/#1174。実Provider target・Transaction/Receipt・独立Read |
| EX-05/07 Generator+Single binary | 未選択Generator/Provider追加→実Go依存閉包不変。Process必須を厳密Single-binaryへ要求→不適合 | #1251 PB/PBN、#1167/#2600/#943。Go import/init/link/member＋Build provenance |
| EX-06/08 Process+Isolation | 選択済み実子Processをkill、ACK-loss、Runtime版混線、unload/cleanup失敗 | #1223/#2390/#1174。実起動物・失敗分類・再送規則とReceipt |

**最初の小さな実証**は (A) 純粋Operator: 標準ライブラリ vs 契約Provider、(B) DB Provider: 静的組込み vs 既存外部接続、(C) Target Generator: 未使用実装追加 vs 実生成対象、の三例に限定する。同一独立Expectedを先に固定し、契約・Source実消費・Build実物・権限・Effectをそれぞれ別列に記録する。外部Processについては既存実Process Referenceを再利用し、実DB/Persistent Effectの成功を合成Receiverから推論しない。

**製品資格に関する非主張**：一部ReferenceにGo/Node PASS、main統合、Issue closeが存在しても、ここで挙げたEX-01〜08の最新main上の横断実行、実Wails/Windows、実DB、独立Provider/Host/Receipt、無停止更新、業務Owner受理を示すものではない。評価結果に必要なら「候補／不適合／未実証」を表示し、新しい固定status enumへ先行昇格しない。

### 14.6 前回比較との差分・既存計画への引渡し

[従来の動的更新方式比較](CORDIS_CONTINUOUS_PLUGIN_UPDATE_EVALUATION_2026-10.md)はEffect scope、Reactive Coeffect、世代切替・無停止運用を中心に比較した。**本節は方式の優劣ではなく、拡張を要求する主体の制約から候補を棄却する手順**を追加したものである。業務Ruleや純粋演算をPlugin万能化しない入口、Bundle非干渉、実消費・契約・Runtime Grantの分離、partial effect/unknownでの停止、Ownerへの質問を追加する。

Implementation/配布の責務：#1251（凝集Bundle/テストと未使用非干渉）、#1016（Projectの構成選択）、#951（Contract/Provider/Conformance）、#988（Language/Operator）、#2622（Data Provider/Operations）、#1121/#259（Target）、#943/#2600（Artifactと増分判定）、#1223/#2390/#1145（動的・外部ProcessのTarget条件）、#1174/#1130（Effect/Receipt）、#2514/#1180（独立Expected）、#2018（コスト）。Issueの受入・ABI/DSL Schema・正式方式は各Ownerの実Consumerと実測後に判定し、第二Resolver/Policy Engine/Hostは設けない。

<!-- tei-extension-evidence-handoff-20261106-v2 -->

<!-- tei-extension-pr2888-specialized-delta-20261106 -->
### 14.7 PR #2888に固有だった「再生成拡張点」とPlugin非必須化の受入

[Draft PR #2888](https://github.com/hat47x/tei-platform/pull/2888)の三軸比較で具体化された **案件が所有するSourceと生成基底の変更検知** を、#2090（限定Closed）、#997（UI Kit）、#2785（Kit更新三者比較）へ戻す。拡張できること、独立論理Pluginであること、実行時にロードすることは非同値。例えば案件独自の保存前検査はProject Accepted Ruleのままでよく、生成されたTargetコードの入口を差し替える必要があれば**公開Hookへ案件Sourceを静的接続**すればよい。独立更新・配布を求める具体的Consumerが存在するまでは、Plugin packagingへ強制しない。

| 変更 | 保持する正本と対応 | 失敗・停止条件（既存Fixtureへ追加する対照） |
| --- | --- | --- |
| 生成基底v1→v2でHookの引数、型、呼出順が変化 | #2090にある宣言済Extension Point/Versionと案件Sourceを、元の生成物から分離 | 再生成が成功しても案件拡張が無効化・誤発火すればconflict/unsupported。Hook名/型が一致するだけで業務意味同一としない |
| 生成コードに直接追記した案件コードを再生成 | 案件が編集したSourceと生成器のOwned領域を分離。#2090の直接編集検出/反復再生成の限定既存検証を再利用 | 無断上書き、未検出の消失、生成器の黙示的な案件コード取込みは棄却。案件SourceのOwner/Acceptanceは別に確認 |
| Root UI Kit + Project Overlay + 新Root Kit | #997/#2785のprovide/replace/decorateとSlot公開点を使用 | UIのDOM/Focus/IME・Action表示・実Grantの変化を区別。複数Overlayの無音勝者、契約外DOMへの依存を適合としない |
| 同じ意味ActionをWeb/Go Desktop/外部Hostへ生成 | #1121/#259 TargetのAdapterと#943実Artifactを分離 | 生成側HookのPASSを実Web/Wails/Host、現在Grant、実Provider Effect/ReceiptのPASSへ転用しない |
| in-process拡張から外部Processへ変更 | 既存Service/Transport contract、Target固有Host、Runtime Scope/実起動物 | Process crashの影響局所化を比較するが、外部DB確定EffectやOS Resourceまで隔離できたと推測しない（#1223/#2390/#1174） |

**受入の出口**：同一Source/独立Expectedの下で、非Plugin手段（Project Rule/静的Source/Extension Point）、独立Plugin化、静的実装と外部Processの候補を比較し、(a) 新たな実Consumer、(b) 独立更新責任、(c) 現場で成立するFailure isolation、(d) Source/Build/Artifact影響の減少、のいずれに**測定可能な差**があるかを記録する。差がない場合は追加Kernel/ABI/Loaderを採用しない。

**EXDとの重複整理**：#2888のEXD-01（非Plugin反証）・02（構文意味）・03/04（非干渉・Selection drift）・05/06（Provider/partial Effect）・08/09（Target/Process）は既存§14 EX-01〜08、§15 EX-H1〜6、#1251 PB/PBNに対応付ける。固有のEXD-07（再生成拡張点）は#2090の既存Caseと本節の表に接続し、独立した第二の検証基準の正本・正式な新試験IDを作らない。#2090 Closedは限定Referenceの範囲だけであり、実生成Target・別言語・Wailsでの新規E2E、Owner業務受理が済んだという証拠にはならない。

**採用前の停止線**：Projectに必要なHook契約やOwner Accepted Ruleが未確定なまま、Plugin選定・Target生成・実実行を「互換」と判定しない。本節の追補はdocs-only、差分統合の実コード試験はNOT RUN。

## 15. 選定後の証拠受渡しと差戻し（2026-11-06、設計／NOT RUN）

§14の選定は「方式候補を選ぶ」まで。本節は**選定した実現方式が実Consumer・Targetでも成立するか**を、既存Ownerの証拠へ接続する。新たなDecision Receipt Schema、第二Resolver、第二Build/テスト正本、採用Gateは作らない。#1251のPB/PBN、#2600のIB、#951のConformance、#1180/#2514の検証基準、#1174のEffect/Receiptを再利用。

### 15.1 最小の証拠受渡し票

| 段階 | 固定するもの | それだけでは証明しないもの | Owner |
| --- | --- | --- | --- |
| 要求/Plugin不要反証 | 既存関数・Rule・Data Operations・設定では満たせない具体的要求、代替案、実Consumer | Plugin化の費用優位・別配備の必要性 | #1016/#2018 |
| Contract | Language/Operator/Service/Generatorの意味・型・Effect/Scope・版、Contract発行者、選択Provider | 現在の実行Grant・実Artifactへの物理収録 | #988/#951/#2622 |
| Source/Meaning | 実Source byte/span/外部Import、Parser/Operator版、候補とOwner Accepted Meaning | Parser成功やSource一致による意味の自動受理 | #2619/#267 |
| Build Closure | 実消費byte、選択Generator/Provider、Go import/init/embed、nested module、Toolchain/flags、Target | manifestだけによる依存完全性やlink後の物理除外 | #1251/#1089/#1167/#2600 |
| Artifact/Reader | 生成Source、artifact member/実起動byte digest、Runtime/Target環境、実ロード物 | 一致するVersionからのCurrent Grant | #943/#1370/#259 |
| Invocation/Effect | Caller、Current Grant、Request/Attempt、実Provider Target、Outcome/Unknown、Receipt・独立Read | Exit0/ACK/RunPlan completedによる業務完了 | #951/#1174/#1130 |
| Consumer | CLI/Headless、Web/Studio、生成Desktop、外部Hostごとの実要求・独立Expected | 単一Consumer PASSによる全Target資格 | #1180/#2514/#259/#2390 |

採用済みの論理選択、実Build可能性、実ロード、現在認可、実Effect観測は**別資格**。実消費入力が不明なら「依存なし」「未使用」「Cache再利用可能」と断定しない。参照だけの列はNOT RUN、適合不能はunsupported、根拠不足はunknownとして既存の診断機構で表示する（固定enum採用ではない）。

### 15.2 非対称な二つの依存失敗（#1251 PB-10を再利用）

**過剰混入**：未選択Bundle/Provider/Generator追加で、明示的Selectionが不変なのに Go init、auto-register、embed、import、generator入力、実Binary・起動動作が変わるなら、未使用非干渉は不成立。候補Catalog版変化とEffective Selection/Build Input変化を区別する（#1251 PB-01/02/07、#2600）。

**必要依存の欠落**：#1251に記録されたmain統合#2868のCore依存追加後、別Go moduleのStudio実Binary buildではgo.sum不足が露呈し、#2872/#2873/#2874の修正がmainへ統合された。root Go test/healthの成功をnested moduleの実Build成功と見なしてはならない。選択済みTargetのgo.mod/go.sum、GOWORK=off、-mod=readonly、必要なら実Link/起動を個別に確認する（PB-10）。この既知障害と修正を新しいEX試験のPASSには転用しない。

さらに、Source/SQL AssetのA→B→A、同名同versionの異Byte、Provider Default選択の変化、Query ID同一で方言SQLのBindingが変わる場合は、Semantic/Parser Cache、説明の現在資格、実Read/実Targetを別々に再審査する。候補Catalog変更だけによる過剰失効も、選択実装の変化を無視した誤再利用も拒否する（#2622/#2600）。

### 15.3 Failure containmentと実方式の差戻し

| 観測された失敗 | 差戻し・拒否の対象 | 保持するEvidence |
| --- | --- | --- |
| 既存関数/Rule/設定で要求を満たせる | Plugin化案を撤回し標準機能へ戻す。意味・Expectedは維持 | 原要求、比較した代替案 |
| Contract/Publisher/Conformance不一致 | Providerの暗黙fallback禁止、選択を保留 | Source/Contract pin、要求/提供Scope、不一致理由 |
| Build closureの過剰混入または不足 | Target/Module固有のBuild Planを修正。稼働中の検証済みArtifactを破棄しない | 実消費byte、import/init/link、Toolchain、artifact digest |
| Load/Host Capability・Grant不一致 | 呼出しを拒否。自動的に別Host/Processへ迂回しない | 実ロードbyte、Principal/Scope、対象、拒否理由 |
| 部分Commit／ACK喪失 | 既存実業務EffectはBundle/Processのrollbackでは元に戻らない。自動再Submit禁止 | Request/Attempt ID、先行Effect、Result unknown、Receipt、独立Read |
| CLIでPASS、Studio/WailsでFAIL | Consumer/Target固有Adapterだけを調査し、共通Meaningを一律不適合と断定しない | OS、Host、Build/Loaded Artifact、実入力・独立Expected |

### 15.4 EX試験の補助観点（新しいFixture正本ではない）

- **EX-H1/2**：未選択Plugin混入（PB-01/02/07）と、選択済みnested module依存欠落（PB-10）を同一TargetのA/B比較票で判別。
- **EX-H3**：Version不変のSource差、A→B→A、SQL/Provider Binding差で、変更対象だけを再資格化。未取得依存は空集合扱いしない。
- **EX-H4**：同じJSON/同じ表示結果でもNULL・欠損・partial・数量単位・多重度・業務Purposeの差は、Operator/Provider適合を未証明とする。
- **EX-H5**：A Step確定後B blocked、ACK lost/二重送信、UI再起動の負例では、Run全体effect=0/失敗完了を主張せずProvider Receipt/独立Readを待つ。
- **EX-H6**：同一read-only ContractでGo単一Binaryと外部Process/外部Hostを比較。外部ConsumerがStudio/TEI Runtimeを暗黙に必要とする場合、外部利用適合を拒否。

**実証順**：P0 read-only選択/実依存照合 → P1 純粋Operator/選択済みGo buildと未選択Bundle影響 → P2 DB Provider能力・部分Read/Write（実DBは別Evidence） → P3 外部Process/異Consumer/Loadと失敗注入。独立Expectedは#1180/#2514に固定し、対象版と実消費byte、実Artifact、Host/Grant、Effect観測は別列とする。新たな万能Fixture/Execution Engineは作らない。

**評価の出口**は、具体的棄却理由、失敗時の回復、採用候補のConsumer・Target適合範囲、必要なOwner判断を既存Issueへ返せること。製品適用・意味受理・Production資格とは別。今回のEX-H1〜6、実Go/DB/外部Process/Windows/Studio統合、性能値、Owner受理はNOT RUN／未採用。


<!-- tei-actual-pg-command-extension-reuse-20261106 -->
### 15.5 実PostgreSQL Commandの既存Evidenceを再利用する「最初の実Provider」縦断計画

前節までのEX-03/04/06・EX-H5の**実証上の出発点**として、現mainの `internal/security/effectcommit/commandadapter/executor.go`（確認Blob `47c2996c961cfb9b0f07a3e6e4223da787a7d204`）を確認した。`CommitUncertainError` を `COMMAND_COMMIT_UNCERTAIN` のUnknownへ変換し、同Attemptの再送を拒否する実装がある。 `internal/security/effectcommit/postgres/command_lifecycle_integration_test.go`（確認Blob `5368cb3501b7d8da62a8ef27c6727ede3943dc44`）では、固定Grantを用いた実PGの成功CAS、古いExpectedVersionの拒否、Commit応答喪失によるUnknown、同じCommand/Operation IDの再実行拒否、higher Generation takeoverでReceiptを再観測して二重更新しない事例がコード化されている。#1174は[merged #2853](https://github.com/hat47x/tei-platform/pull/2853)でのUnknown伝播修正と過去の実PG CI成功を記録し、[merged #2854](https://github.com/hat47x/tei-platform/pull/2854)ではABA後に**別Idempotency Keyなら再適用され得る**ことを確認した。これらは**既存の限定試験・記録**であり、今回実行した試験ではない。

| 段階 | 既存Evidenceを使う部分 | 未証明・採用停止線／主担当 |
| --- | --- | --- |
| V0 純粋/Read-only比較 | 同じFunction/意味と固定Source/Scopeの標準Operator/明示Data Operations vs 独立Plugin候補（#988/#2622） | 独立Expected・NULL/short-circuit/partial・実Read Scopeが不明なら結果値だけで互換としない（#1180/#2514） |
| V1 Provider選択→実呼出 | #951/#1016のexact Contract/Provider/Publisher/Conformance/Selectionを参照 | **実Plugin Selectionが現main Command AdapterのProvider/Target/ExpectedVersionへ同一版で接続された証拠は未取得**。名目上のSelected/Linkedで現在Grantを付与しない |
| V2 Target収録・ロード | #1251 PB/PBN/PB-10、#943/#1089/#1167/#1370で実消費閉包・Artifact・ロードbyteを固定 | 未選択Bundleの非干渉と選択済みnested moduleの依存欠落を別検証。root Go test成功から全Target資格を推論しない |
| V3 実Effect・Receipt | 上記PG Command CAS/Unknown/Receipt、#1174のoperation identityとcurrent Host Grant | **選択Provider・現Principal/Scope/Grant・実Target・Effect Receiptを同一Attemptで束縛する実縦断は未実証**。拒否時Effect=0、別Key ABA、失効・期間・After-commitと独立Readを要する |
| V4 外部Process選択時のみ | 同一のread-only Contract/ExpectedをGo静的Targetと別Processで比較（#2390/#1223） | 起動実物/Transport/kill/timeout/現在Grant/隔離/費用を実測。strict single-binaryと矛盾なら外部Process案を棄却。Process再起動で永続DB効果は戻らない |

**RunPlanとの重要な差**： `internal/execution/runplan.go` の `EffectPreconditions` はObserverの**存在確認**であり、ここで `ObserveEffect` を呼んで実Effectを確認するわけではない。先行StepがEffectを持った後の後続Step blockedも、全Run effect=0ではない（#1130/#1174）。Providerの原子性自己申告、Command success、RunLog completed、Consumerの成功表示は業務Owner受理へ自動昇格させない。

**実証順と非主張**：P0はV0/V1のread-only Source/Contract/Selection照合、P1はV2の実Go Build入力、P2でV3の既存PG試験に欠けるPlugin選択・現Grant・同Attemptの負例を接続、P3は必要なTargetだけV4へ進む。DB方言SQL切替は#2622/#2862へ戻し、固定SQL Asset欠落で共通SQLへ黙示fallbackしない。新Plugin ABI/Transaction Engine/Receipt Store/Hostは作らない。**新しいEX/Go/PG/Process/OSテスト、統合E2E、正式Owner受理はNOT RUN**。

<!-- tei-authoring-plugin-trust-boundary-20261106 -->
## 16. 開発用Pluginの作用と生成アプリケーションへの間接影響（2026-11-06、設計・NOT RUN）

### 16.1 「Runtime非同梱」と「無作用」は別

§14の方式選定・§15の実Build/Consumer資格と直交し、**開発時に実際に作用する対象**を区別する。開発用Pluginは生成Binaryに含まれなくても、Source/Meaning候補、生成器入力、Generated Source、Build Artifact、外部通信、秘密情報に作用し得る。非同梱を理由に「無権限で実行可」「成果物・実行性能に影響しない」と推論しない。一方、標準の純粋演算にはDB Commit/Receipt機構を強制しない。

| 能力の例 | 実際に確認すべき境界 | それだけでは主張できないこと |
| --- | --- | --- |
| Parser/Linter/Source Inspect | 許可された実Source/Import byteとRead scope、診断・位置。stale/unavailable/partialの区別 | Read-only宣言からOS上の自由なファイル/Network利用を認めること、正常Parseから意味受理 |
| AI/DSL変換・Source Patch候補 | base revision、対象ID/span、根拠、差分・不確実性。外部Tool文面は未信頼入力 | 引用位置や型整合から業務意味の支持、AI回答からCommand実行許可 |
| Formatter/Generator/Build Tool | 実入力閉包、GeneratorとToolchainのexact版、Generated Source・Artifact差分。外部Processの実起動Resource | Generatorが実行Binaryに非同梱だから生成結果不変、Exit 0から意味適合 |
| Studio Script・Data/外部Provider呼出 | 実Principal・Resource・Effect・Target・現在Grant、実呼出とReceipt/独立Read | Plugin署名・自己申告Capability・開発者権限から本番DB書込み許可 |
| Runtime Provider | Selected/Build-consumed/Linked/Loaded/Invokedを別記録 | Bundle同梱から実利用・現在実行資格 |

同じ高凝集Bundleへこれらを同梱しても、**Bundle一括導入から個々のResource/Effect権限を自動発行しない**。基本Interfaceと実装の同梱は許容するが、実装版・契約版・実消費byte・作用面の資格は個別に追跡。Studioローカルno-authもWorkspace/OS制約を解除しない（ADR-0044、#951/#2017）。

### 16.2 Source候補から実Effectまでの停止線

1. **外部SourceとTool結果**：Reference/SQL Asset/AI応答/Plugin Manifestは出典・固定版と取得状態を残す。Source内の命令注入を実Commandへ昇格させない。Source移動、A→B→A、offline/stale、read-time TOCTOUを再評価（#2743/#988/#2600）。
2. **Candidate/Preview**：Parser/AI/Language Pluginの結果は候補。型・引用span・実Read footprint・独立Expectedを検査し、未確定は保留する。CandidateからProviderへ直接作用する経路は認めない（#988/#2514/#2745）。
3. **Patch適用と意味受理**：base revision、対象ID、適用する実差分と実保存byteを照合。候補作成後のSource改版ならstaleとして再比較。Preview、Validate、Patch applied、Owner Accepted Meaningを分ける（#2497/#2621）。
4. **生成とBuild**：選択Generator/Toolchain・実消費Source/Import・Go nested module/全Artifact memberと結果を結ぶ。未使用実装の混入と選択済み依存の欠落の両方を検査する（#1251 PB/PBN/PB-10、#1089/#1167）。
5. **実呼出/永続作用**：Target側Host/Providerの現在Grantで実効制限。Process ACK喪失、部分Commitはunknown/Operation ID/Receipt/独立Readを保持し、Plugin revertや再Installで業務Effectが元に戻るとは主張しない（#1129/#1174/#1223/#2390）。

### 16.3 既存Fixtureへ返す差分負例（新Runner/新Test IDを作らない）

| 対照 | 独立に観測するもの | 既存Owner |
| --- | --- | --- |
| Read-onlyを宣言した開発Pluginが許可外ファイル書換・外部送信を試みる | 実OS/Hostの拒否と実ファイル/通信結果。Manifest/署名のPASSで代用しない | #2017/#951 |
| AIが正しいSource spanと不当な実行指示を同時に返す | Candidateとして保留、意味的支持と実Command非実行を別観測 | #988/#2514/#2745 |
| Patch候補後にSourceが改版・移動、A→B→A | base revisionと実byteの不一致検出。旧Patch/受理の自動復活を禁止 | #2743/#2497/#2600 |
| 未利用Generator更新と、実利用Generatorの同名異byte更新 | 前者のArtifact非干渉と後者の実Build再資格を区別 | #1251/#1089/#2600 |
| 同じ表示RowsでもRead scope/NULL/partial/意味版が違う | 結果値の一致からMeaning/Grant/Read完全性を推論しない | #988/#2622/#2515 |
| 外部ProcessがACKを返さず先行StepはCommitした可能性 | Operation ID、unknown、独立Receipt/Read。無条件再送と自動Effect rollback禁止 | #1129/#1174/#2390 |

### 16.4 実証の順序と責務

- P0：既存Plugin/AI Tool/Generator/Studio Scriptの宣言Capabilitiesと、実際のfile/network/process/DB作用・強制主体・未証明領域をread-onlyで棚卸し。万能Plugin権限レベルや新enumは導入しない。
- P1：#988/#2514/#2745の既存Source・独立Expectedで候補とPatch適用/Owner受理が自動連鎖しないことを検証。MockだけでOS非作用PASSとは主張しない。
- P2：#2017のOS資源・情報流と#1251 PB-01/02/07/10のGo実Build/実依存検査を接続。Windows Studio、外部Host、実DBはそれぞれ別資格。
- P3：静的Library/Rule/設定で実需要を満たせばPluginを新設せず、独立更新・隔離・外部Host可搬性が必要なときだけ§14の物理方式選定に戻す。

**状態**：本追補は既存PRにおけるdocs-onlyの受入・停止線。実OS隔離・AI→Provider遮断・Go差分Build・Windows/DB/外部Process実走・Owner正式受理は新規NOT RUN/NOT ACCEPTED。第二Host/Registry/Resolver/Policy Engine/Plugin ABI/検証基準を追加しない。


<!-- tei-plugin-update-impact-qualification-20261106 -->
## 17. 高凝集Bundleの更新・失効を局所判定する（2026-11-06、設計・未採用）

§14のPlugin要否と物理方式、§15の実Build/Consumer資格、§16の開発時作用とは別に、**配布Bundleの更新を一律にBuild/Owner受理へ伝播させない更新ゲート**を定める。これは新しい更新マネージャー・Registry・Lock Schema・汎用Cacheの設計ではない。#2600のIB/Source closure、#1251のPB/PBN、#951/#1016の選択、#943/#1370のArtifact、#1174の現行Effect資格を同じ選択されたTargetに対して照合する手順である。

### 17.1 変更されたものと、実際に利用されたものを分ける

**別々に追跡する版・同一性**：
(1) 配布Bundle/候補Catalog、(2) 公開Service/Operator/Language Contractとその意味・Effect、(3) Provider/Generator等の実装artifact・digest、(4) Source/参照位置・Import閉包、(5) Application Profileの選択とResolved Lock、(6) 実Builder/Toolchain・Target・全Artifact member、(7) 独立Expected・Conformance/Owner Accepted Meaning、(8) 実Loader/Consumerと現在のProvider/Host Authority。これらをBundleの一つのversionへ縮約しない。

**選択の否定依存**：Plugin候補が増えた際、固定SelectionのままならBuild非干渉があり得る。一方、default順位・同Contract候補競合・対象Capability不在時の解決結果が変わるなら「未使用だから無影響」とは判定できない。Providerの登録順で暗黙選択を変えない。選択が不明なら未知/保留とし、既存#951/#1016に解決を委ねる。

### 17.2 更新イベント別の差分受入行列

| 更新・事件 | Artifactの初期判断 | Evidence・現在資格の判断／停止条件 | 既存Owner |
| --- | --- | --- | --- |
| U-01 未選択Bundleの文書・補助実装更新 | 選択済み実消費入力が同じなら対象Targetは再Build不要候補 | Catalog/Repository改訂は記録。否定依存（default・競合）を確認し未使用の物理混入なし | #1251/#2600 |
| U-02 同じBundleに新Provider/Generatorを追加 | 全同梱コードを自動Buildしない。選択・import/init/embedが不変ならArtifact非干渉候補 | 既定候補が変われば再解決。起動時登録副作用があれば未使用判定を撤回 | #1016/#951/#1167 |
| U-03 選択された純粋Operatorの実装を変更（Contract同一） | 実消費が変われば計算／対象Buildを再評価 | NULL・未取得・short-circuit・Decimal等の独立Conformance。Owner受理を自動変更しない | #988/#2514/#2600 |
| U-04 利用Generator更新、生成Source/Artifact bytesは偶然同一 | 入力実装・Build Attemptと出力を別記録。内容再利用の可否を局所評価 | 旧Generator適合Evidenceは新Generatorの資格ではない。独立Expected/Toolchain/Target適合を再評価 | #1121/#2016/#2514 |
| U-05 Contractの型・Effect・Scope・Failure semantics変更 | 同じProvider名やbinary byteでも互換とは決めない | 旧Consumerの受理Baselineと新Contractの関係を確認。旧Grantや意味受理を無音移行しない | #951/#1121/#1174 |
| U-06 SQL Asset/Import/Source A→B→A、または同名同版で異byte | 実消費Source・選択Realization・Derived Queryを再資格化。旧Source hashだけで履歴失効を取り消さない | Source位置と解釈、実DB Read Scope/Disclosure、現在Grant/Owner判断を独立確認 | #2622/#2600/#2516 |
| U-07 選択Runtime Provider更新・欠落・撤回 | Go静的Targetなら対象Build/再配布、外部Hostなら実ロードbytesを確認。未選択実装へ黙示fallbackしない | 旧Receipt/Artifact履歴は保全、現在のInvocation許可は別判定。再ビルドだけではGrant復活しない | #951/#1223/#1370/#1174 |
| U-08 Owner業務Rule/発効時点が変更（Plugin同一） | Pluginを更新しなくてもAffected Function/出力が変わり得る | Rule/meaning revision・valid/known時点・入力snapshotと実行時Grant/Receiptを区別し、Owner再受理を求める | #1129/#1174/#2600 |
| U-09 選択済みTargetのGo nested module/依存が増減 | root test成功でTarget成功としない。選択moduleのsum・readonly build・artifact memberの充足を個別検証 | 依存欠落と未使用Plugin混入は別の否定対照。既存修正PRのPASSを新変更へ転用しない | #1251 PB-10/#2831/#1167 |

※表中のU-01〜09は**今回の照合行ラベル**であり、第二のFixture正本や新しいStatus enumではない。内容同一の再利用、解析・生成の省略、Conformance Evidenceの再利用、実行許可は個別のClaim。実消費Source closureが不足・不明なら「無依存」や「再利用可」を確定しない。

### 17.3 更新失敗・配備時の回復を方式別に扱う

- **標準Go静的Binary**：Runtime Providerを変更するなら、当該Target用の生成・Build/検証済み配布物で更新する。未選択Pluginの交換を動的に反映するための常駐TEI Host/Loaderを追加しない。既存配布物の歴史的な同一性は保つが、古い配布物の現在使用許可は別判断。
- **独立Process/動的Targetを選択した場合のみ**：実ロードByte、世代pin、Client/Server Contract、旧要求のin-flight、Resource/Lease/Fence・readiness・cutover/drainを対象Host/Runtimeで確認する。reload 成功だけで無停止更新・実業務Effect移行を宣言しない（#1223/#2390）。
- **失敗後の復元**：Providerの更新取消や旧Artifactへの切戻しは、すでにCommitしたDB/外部Effectを取り消さない。ACK-loss/partial/unknown時はOperation ID、RequestHash、Receiptと独立Readを先に確認し、無条件再Submitを禁止する（#1174/#1129）。
- **資格の撤回**：過去にConformanceを通過したこと、ArtifactのDigestが同じこと、SourceがAへ戻ったことは、現在のTrust/Permission/Owner受理を自動復活させない。

### 17.4 最小実証順

P0：既存#1251 PB-01/02/07と#2600 IBの同じ選択済みGo Targetを固定し、Catalog/Bundled/Selected/Actually-consumed/Includedを分ける。独立Expectedと実Input closure、Builder/Toolchain/flags、全Artifact member、Readerを固定する。

P1：**U-01/02（過剰失効）対U-03/04/09（誤再利用・依存欠落）**を同じ比較票で検証。変更が未選択なら不要な再計算を防ぎ、実消費が変わったときの失効とTarget build失敗を検出する。対象ファイル・全moduleの実Go検証は#1251/#2831、測定は#2600/#2018へ委譲。

P2：#988/#2514で標準pure Operatorと代替実装、#2622/#2862でProvider別SQL/Read Scopeを独立検証基準と照合。値だけの一致を意味互換・現在権限へ昇格しない。

P3：実外部Host/DB・長寿命Targetの更新、現在Grant/Lease、partial Effect/Receiptの回復は#1223/#2390/#1174の資格でのみ評価。最初のGo単一Binary要件へこれらの動的機構を逆流させない。

**採用判断**：Bundle版と契約版の分離・局所再資格化が、変更波及、false reuse/false invalidation、Test漏れ、ビルド時間を実際に改善した範囲だけを製品設計候補とする。新API/Registry/Cache/Runtimeの実装とOwnerの業務意味受理は本追補では未決。U-01〜09を含む今回の実Go/DB/Host試験・性能測定は NOT RUN。

<!-- tei-extension-first-comparable-probe-20261106 -->
## 18. 初回の同一要求A/B比較：Plugin不要案を含む実Consumer試験（2026-11-06、D0限定CI確認／D1比較未実施）

§14〜16を再度一般論として増やさず、**方式の採否を変え得る最初の実証**を固定する。元の業務意味・検証基準・Scenarioの正本は増設しない。[#1861](https://github.com/hat47x/tei-platform/issues/1861)の `reference.order.quote-lines/v1` 読取り専用見積もりプレビューを第一候補とする。正常8件・業務拒否15件の**既存23ベクトル**（#1819/#1840のGo Native・Node JS基線、#1848のWIT型投影Reference）を参照する。固定23件は**未受理の業務候補**であり、全業務意味がAccepted済みだとは主張しない。

### 18.1 比較対象を混ぜない

| 用途 | 比較経路（同じ23件と独立Expectedを再利用） | 現時点で可能な主張と不足 |
| --- | --- | --- |
| **D0：Pluginなし** | 既存の純粋なGo Native/Node JS計算・ローカル関数として使用。Provider解決/動的Loader/Process transportを要求しない | 既存Issueに限定基線がある。今回のexact revisionで再実行/受理した訳ではない |
| **D1：静的Provider** | 同じ意味契約を明示選択した実装としてGo単一Binaryへ含める案。必要なら薄いAdapterに留める | Plugin化による独立更新・再利用の追加価値、実Build closure/物理除外/起動時間は未実証 |
| **D2：独立Process Provider** | 同じ見積もりContract/Expectedを既存stdio transport/Runtimeへ接続。Go／JavaScriptの独立実装、Host Guard、実起動子Processと実Transportを区別して観測 | **既存の同Contract 23件Process比較は存在**（`tools/reference-capability-transport/quote_go_js_vectors_test.go`）。一方、同じ23件を現在のRuntime Registry/正式Provider Conformance/Host Grantへ直列に通した証拠、D0/D1との同一Target比較は未確定。#1145の旧Carrierは共通ABIではない |
| **D3：Wasm Component / 外部Host** | #1861のWIT/Component比較。D1/D2とは別のTarget・Artifact/Host実現候補 | **実Componentの限定CI証拠あり**：2026-10-15の[CI #38746502755](https://github.com/hat47x/tei-platform/actions/runs/38746502755)でNode、実Chromium（Jco変換成果物）、Wasmtime（元Component直接呼出し）が固定23件に対する正常8・Ingress拒否7・Guest拒否8を検証。ただし3 Hostが全23件をGuest呼出したわけではなく、実Host Authority/意味受理/製品採用や現PR exact-headを証明しない |

D0→D1/D2を自動的な「成熟段階」としない。**D0が要求を満たし、独立更新・障害隔離・外部Consumer等の追加必要性がないなら、D0選定で正常終了**してよい。D2の既存同Contract Processの限定実績を保持しつつ、別ContractのProcess Bridge PASSや別Attemptの実行を、D0/D1/D2の一貫した最新比較に合算しない。#2090のProject Hookや#997のUI Kitは本試験とは異なるContractであり、同じ23 Expectedを機械的に転用しない。

### 18.2 実行前に固定する比較条件

| 項目 | 事前に保持すべき値（新Schemaを採用しない） | 合格を装わないための対照 |
| --- | --- | --- |
| Source/Revision | 既存23件のSource/期待値ファイルの原byte・reference revision、Contract/Provider版、生成器/Go toolchain/Build flags | 旧PRのPASSを新HEADへコピーしない。Source A→B→Aと同名異byteを別Attempt |
| Independent Expected | 同じ23件の数量・通貨・小計または業務拒否、NULL/欠損/数値Lexeme/順序等、既存独立期待の適用範囲 | ProviderからExpectedを生成しない。輸送型一致だけではBusiness Meaning一致としない |
| Environment | OS/CPU・Go/Node版・Target・依存Module・実プロセス/外部Host、分離されたBuild/Run Root | root Go Test成功をnested module/外部Consumer成功に転用しない |
| Capability/Effect | 読取り専用、DB/Network/Secret/ファイル変更不要という初回要件。実Hostで許可したResource | Plugin自己申告のread-onlyをOS保護とみなさない。権限不足時の実起動拒否を対照 |
| Artifact/Consumer | D0/D1の実リンク済Binaryと必要member・実ロードByte、D2の実Process executableと依存閉包 | manifest/Bundle同梱と実ロード・実行を混同しない。未選択Plugin追加で非干渉検証 |
| Failure/Isolation | Process kill/timeout・旧新版競合・Host guard拒否・結果不明・unload/cleanup、影響する子/親Process | Transport成功を業務適合に、Process隔離を無条件のOS sandboxに読み替えない |
| Evidence有効範囲 | Attempt/Head/Source closure/Toolchain/Consumer/Environment/Observed resultと、失効要因 | Artifact不変でも現Grant/Publisher適格・業務Owner受理を自動復活させない |

**試験レベル**：Unit/Fixture（23件の結果と独立検証基準）→ Conformance（Contract/Provider/Transportの相互作用・不適合負例）→ actual Build/Process/Host（本当にBuild・ロード・起動されたか）→ 実Provider/DB/Effect（この読み取り専用ケースでは該当しない項目をnot-applicableにする）→ Owner Meaning acceptance（別審査）。「テストはPASSしたが実物をロードしていない」を明示する。#1180のScenario SSOT、#2514の検証基準、#1163の同一改訂Evidenceの資格を再利用する。

### 18.3 最小コスト指標と判断Owner

実行前に**必要な比較項目と測定環境を固定**し、D0/D1/D2で追加した手書きAdapter・接続設定、Source変更箇所、Coreへの逆依存、実Build入力とBinary/Artifactサイズ、clean build・incremental build時間、起動/定常呼出Latency、プロセス数、更新/復旧手順、negative controlの判別数を同一条件で測定する。数値閾値や性能優劣はOwnerのTarget/SLOが未指定なら確定しない。契約形だけを同じにした結果、NULL・業務拒否・単位・失敗・Unknownの区別が落ちる場合は不適合とする。

**停止/選択条件**：既存の純粋計算で十分なら非Plugin案を残す。複数Consumer・独立実装版・Target固有対応が具体的に必要になったときだけProviderを比較する。実隔離・個別更新が必要ならD2の候補評価に進むが、厳密な単一Go Binary要件と矛盾するときはD2を棄却する。実DB書込みやCommit不明はこの23件に紛れ込ませず、別途#1174/#1130の既存Write Scenario・実独立Readへ段階的に進む。

**責務と実証状態**：#1016は案件要求と方式選定、#1861/#2390は外部Consumer・WIT必要性、#951/#1145はContract/Transport、#1251/#1167/#2600は未選択混入・必須依存・性能、#2514/#1180/#1163は独立Expected/改訂Evidence、#1174は実Effect。今回のD0再試験、D1の同一Go Target静的Provider構築、D0/D1/D2を揃えた23件横断比較、今回のexact-head再実行と定量測定、Owner正式採用は**NOT RUN/NOT ACCEPTED**。D2単独の既存実Process実装・限定23件実績は上記のとおり保持する。新ABI、全Plugin共通Process Runtime、第二Resolver/検証基準/Hostは増設しない。

<!-- tei-input-closure-selection-negative-controls-20261106 -->
### 18.4 方式比較より前に「必要Sourceが読まれたこと」を確定する（#2822/#2823）

**他レーンの具体的反例**：[#2822](https://github.com/hat47x/tei-platform/issues/2822)では利用者宣言0件のWorkspaceが `validate -json` で `ok:true` を返し、標準型のCatalog件数が非0でも利用者の入力が存在した証拠にはならない。[#2823](https://github.com/hat47x/tei-platform/issues/2823)では、未知/存在しないmanifest `sources`、別拡張子の管理ファイル、`.TEI`／`Application.tei`等の読み飛ばしが発生しても、ある構成では成功診断になる。これは**既存の別版CLI再現記録**であり、このDraftのHEADで再現済みとはしない。この状況でD0/D1/D2を比較すると、Required Inputが欠けているのに「同じSourceでPASS」「未使用Plugin非干渉」「Build skip可」と誤判定する。

D0〜D2の第一回比較では、#1861の固定23ベクトルとContract、実際に読むProvider/Generator/AdapterのSourceを**試験開始前に固定**する。次の集合の包含/差を同じSource Revisionに対して報告し、不明を空集合として補完しない。新しいManifestやValidatorは作らず、既存Workspace Source Reader/Index・Plugin Lock・実Builderの観測値を用いる。

| 観測すべき集合 | 独立に確認すること | 不足時の判定・既存Owner |
| --- | --- | --- |
| Required（期待される必須入力） | 23期待値とSource/Contract/Generator・必要Plugin/Importを、利用側要求と独立Expectedから定義 | RequiredそのものをResolverが観測したSourceから逆生成しない。#1180/#2514/#1016 |
| Declared / Discovered | manifestに記載されたSource rootとWorkspace上の候補、同名異byte・命名規則/拡張子を区別 | Unknown root、存在しないroot、管理対象だが未Parser対象のファイルは診断/保留。#2823/#2743 |
| Resolved / Parsed | 実Provider/Language Parserが解決したSource位置・Revision・byte、Parse診断とSkipped理由 | `validate.ok`/Catalog件数だけでRequired全件がParsedとはしない。#2822/#2823/#988 |
| Actually consumed | 実Go Build/Generator/Operator/SQL Assetが参照したbyte・import/init/embed、否定依存・選択根拠 | Parser済みだけでも実Build Inputにはならない。未観測ならCache skip/非干渉/Artifact equivalenceはunknown。#2600/#1251/#1089 |
| Consumer result | 実Target/Host/Read scopeで使ったSource/Artifact、Independent Expectedと必要な拒否負例 | D0の計算結果PASSからD1/D2の実Build/Transport/Grantへ昇格しない。#951/#2390/#1163 |

**最小の停止試験**（独立Expectedは既存#2822/#2823と#2514へ委譲）：(a) 必須Rootを一文字変える、(b) Root名は正しいが必要ファイルを`.tei.bak`等へ変更、(c) empty Workspaceに標準型だけある、(d) 選択済みProvider Sourceが未取得、(e) 未選択Providerのみ増減する。期待は、(a)〜(d)を「必須Source完備／現在の実行資格」とせず診断またはunknownへ、(e)は実Selection・消費と否定依存が不変なときだけArtifact非干渉候補とすること。実際のCLI診断仕様（警告/エラー）は#2823、空Workspaceの契約は#2822が決め、ここで新StatusやCLI schemaを先取りしない。

**実証順の補正**：D0は固定23ケースを自己完結の関数として実行可能な範囲だけ「入力充足の主張」を行い、Workspace連携のSource完全性は別のClaimに留保。D1/D2へ進む前には選択済みSource/Contract/Build Inputの閉包を実際に確認し、未知なら方式優劣やCache再利用を結論しない。Sourceの宣言・解決・実消費と現在Grant/Owner受理を分離する既存§15/§17の原則を適用する。

**状態**：#2822/#2823の過去の実CLI反例を受入条件として参照した設計追補。今回のCLI/Workspace/Go/Process/23ケース再実行、正式診断採用、Owner受理はNOT RUN。第二Source Index・Validator・Resolver・検証基準は増設しない。

<!-- tei-compare-oracle-decision-ceiling-20261106 -->
### 18.5 検証基準の判別力と比較の公正性：23件PASSを方式採用に読み替えない

[#1861](https://github.com/hat47x/tei-platform/issues/1861)の23ベクトルは、見積もりPreviewの**未受理の有限な業務候補**。実際のSource/期待値ファイルの原byte、Contract/対象Consumer、Revisionを固定し、D0/D1/D2のInput/Meaning/結果分類が同一であることを確認する。Sourceが正しくロードされていても、ExpectedがCandidateと同じEvaluator・Rule・AI解釈から生成されていれば**共有起源の誤り**を検出できない。独立検証基準の出所・受理資格・判別能力は#2514、Scenarioの正本と実Target派生は#1180へ戻す。

| 観測項目 | 必須の反証と判定上限 |
| --- | --- |
| Meaning/Shape | 数量・通貨・小計、NULL/欠損、順序・多重度、金額精度、業務拒否と実行失敗/unknownを、実際のAdapter前後で独立照合。表面のJSON型・結果文字列が同じでも契約同等とはしない |
| 検証基準の独立性 | 独立Expectedの由来とProducer/Observerの独立性を示す。意図的な通貨無視・小計変更・業務拒否→技術失敗の誤変換等のMutantを、**既存の同じ検証基準が検出するか**計測。検出漏れと未実施を隠さない |
| Actual implementation | D1は実選択Provider/Go binary/Link/Loaderまで、D2は既存Go/JSの23件Process比較を出発点に、同一Attemptの実Runtime Registry/Conformance/Host Guard/実ArtifactとD0/D1比較の不足を追加観測する。他ContractのProcess BridgeのPASSは同Contractの新しい上位資格に流用しない |
| Same-environment cost | D0対D1の差を評価するときは同じGo Target・Host・OS・Toolchain・Source revision・負荷でclean/incremental build、起動・定常Latencyとサイズ/依存を反復測定。Go対Nodeの実行時間差をPlugin化の効果と見なさない |
| Negative controls | #1251の未使用実装混入/必要依存欠落、#2600の誤skip/過剰失効、#2822/#2823の必須Source未ロードと、D2のHost拒否・異byte起動物・timeoutを別Failureとして分類 |
| Claim ceiling | 23件の結果一致はread-only Previewの限定一致。Owner Meaning受理、OS隔離、実DB更新/Receipt、全Target/本番互換性は別。該当しないWrite義務はnot-applicable、未実施はNOT RUN |

### 18.6 採否の出口と再審査条件（新しい判定票正本は作らない）

- **D0で十分**：対象の要求・実Consumer・独立Expectedが満たされ、独立版管理/交換、別製品への公開、別Process隔離・無停止更新を必要とする実例がない。これは「D0がD1/D2より高速・優秀」との比較証明ではなく、**追加Pluginを作る必要がないという用途限定の選択**。D1/D2未実証をD0優位のPASSとして数えない。
- **D1を候補として維持**：複数の独立Consumerや契約付きProvider差替えが具体化し、通常Library/静的Ruleでは満たせない制約がある。単一Go Binaryの選択済み実依存・必要能力・Bundle非干渉と、追加の保守負担を実測して採否を判断。
- **D2を候補として維持／棄却**：異言語Host・隔離・独立更新が必要な場合のみ評価。厳密Single-binaryに合わなければ棄却。Process境界だけではOS sandboxやAuthorityを証明せず、実Host拒否・障害波及・再接続/unknownを観測。
- **保留・差戻し**：Required Source/実消費不明、検証基準共有起源・Mutant検出漏れ、意味・Contract差、実Target未実行、current Host Grant不明などは、**該当するClaim**をunknown/not-runに留めて担当Issueへ返す。採用を急ぐために別Providerを黙示選択しない。

再審査は、Requirement/Owner受理範囲、新Consumer、実選択/Source closure、Artifact/Target、Providerの現在資格、独立検証基準の判別能力、測定した費用が**実際に変化したとき**に限定する。無関係なCatalog/Bundle更新だけで全Targetを過剰失効させず、A→B→Aや同version異byteでも自動的な古い受理の復活は認めない（§17、#2600）。

**実行責務**：#1016（用途別選定）、#2514（検証基準/Mutant）、#1180（Scenario）、#1861（23件）、#951/#1145/#2390（実Provider/Transport/Host）、#1251/#2600（実Buildと費用）。本節に別のExpected・Runner・新ABI・中央Decision Engineを置かない。D0/D1/D2を揃えた同一改訂・同一条件の23件横断比較、Mutant実測、性能測定、Owner正式採用は現時点でNOT RUN/NOT ACCEPTED（D2単独の既存Process比較は§18.7）。


<!-- tei-plugin-dynamic-read-dependency-closure-20261106 -->
### 18.7 Source閉包と実行時Read閉包を分離する：0件・非出力・否定依存

§18.4のRequired/Parsed/Actually-consumedは**ファイル・契約・Build入力の閉包**であり、実行時Query/Providerの対象集合が完全であったことを証明しない。他レーン#2600 IB-16（非出力・否定依存）、#2506（Data Context/Result Slice）、#2582（再取得・retract）、#2387（Queryの演算）、#2516（開示）、#1708（Read cohort/snapshot）へ接続する。**新しいRead-Dependency EngineやClosure Schemaは設けない**。各Claimに必要な既存Source/Query/Reader/Providerの観測内容を既存Evidence欄へ参照するだけとする。

まず初回の[#1861](https://github.com/hat47x/tei-platform/issues/1861) `reference.order.quote-lines/v1` 23件は、固定引数を用いる**外部DB/Network/Secret/Fileを必要としない純粋計算**として比較する。D0は「その計算に外部Readは不要」の契約が適用できるが、D1は実選択Provider/Builderの不要依存・実I/O有無を別検査する必要がある。D2は**計算対象の業務データReadは不要でも、Processの起動・Transportは実I/O**である。D0/D1/D2の23件PASSだけで、独立Processの情報流・OS隔離、別DB Queryの完全性を主張しない。読取り専用（writeなし）と、I/Oなし（pure）は非同値。

| 比較対象・Claim | 必要となるRead／否定依存の観測 | 証拠不足時の停止・別Owner |
| --- | --- | --- |
| 固定23件の純粋計算 | 入力Vectorの実byte、Semantic Operator/Unit/NULL・Decimal、選択実装と参照先なしの契約、必要なRuntime/Resource | 自称pure・Read-only manifestだけで実OS非アクセスを証明しない。読取のないData Rule比較はこの範囲に限る（#1861/#951/#2515） |
| 複数Relationを用いるData Provider／Function | 対象Relation occurrence／join・predicate／非出力条件、Provider/DB Alias、Read scope・対象identity・Version、結果Slice（未取得／0／部分／完全）、取得時点・cohort/snapshot | 行値一致、同名Semantic Data、同じQuery IDでReader同一を推測しない。必要な粒度/対象が未観測ならunknown（#2506/#2611/#2387） |
| `NOT EXISTS`／anti-joinで正常0件を導くClaim | 対象母集団、anti-join右辺とpredicate、filter/RLS・権限・ページング・Limit、対象epoch、結果Rowset/Field完全性と否定結果が有効な時点 | 空Rowsを「依存なし」「対象不存在」「Build Cache再利用可」としない。Providerがpartial/hidden/offlineなら否定Claimは保留（#2600 IB-16/#2582） |
| 見かけのResult不変のまま更新 | 依存したRelation/Predicateへの行追加・削除、Filter/Role/Schema更新、対象Scope/Source A→B→A、Read cohortの変化 | LEFT JOIN/NOT EXISTS/Top-Nなどは追加で既存結果をretract/replaceし得る。再評価が必要なClaimのみ失効し、無関係なTarget/Bundleを一律失効しない（#2582/#2600） |
| 実OS/Hostで「外部データ取得は不要」 | Plugin宣言のCapabilityと実行物のFile/Network/Process/DB到達・拒否ログ、D2ならTransportと業務Readを分離 | read-onlyの自己申告はOS強制ではない。秘密・隠れたRow・内部Source/件数を低権限Consumerへ表示しない（#2017/#2515/#2516） |

**独立Expectedと反例の接続**：新しい23件の業務Fixtureを追加せず、純粋計算のD0/D1/D2は§18.5の既存の検証基準を使う。実Data Providerへ進む場合だけ、既存#2600 IB-16／#2506／#2582／#2387の「同じ表示値・異Read scope」「0件だがpartial/hidden」「後から一致行が追加」「SQL/Provider route変更」を独立Expectedとして参照。どのClaimがどのSource/Read/Targetに依存するかを**出典・版・実行時点ごとに**記録し、実Read footprint未観測なら「完全」「非干渉」「現在不存在」と結論しない。

**導入順の制限**：P0は23件のpure/transport/I/O区分をread-onlyで確認、P1は§18.4のWorkspace/Build入力閉包を実Targetで確認。P2で実Query・Relationに依存するConsumerを選んだ場合に限り非出力依存・0件・retractを既存Provider/独立検証基準で検証。P3のWrite/Commit・Receiptは§15.5と#1174が別に検証する。本節によって初回pure `quote-lines` に実DBを導入したり、全Providerに同一Snapshotを仮定したりしない。

**状態**：他レーンの設計・Issueの成熟した負例を方式選定の受入上限へ参照接続したdocs-only追補。実23件、実Read trace、二DB、OS隔離、dynamic dependencyの実装・性能、Owner受理はこの追補では**NOT RUN／未採用**。

<!-- tei-bundle-granularity-and-required-guard-20261106 -->
<!-- tei-quote-comparison-actual-baseline-evidence-20261106 -->
### 18.8 既存コード・CIから判明した比較の出発点と残り一つの決定的差分（証拠更新／D1未実証）

§18.1のD0〜D3について、**2026-11-06時点で閲覧できるmainのReferenceと過去CI**を照合し、計画上の「すべて未実行」という旧表現を補正する。GitHubのSourceと過去CIを確認したことであり、本PR HEAD上でテストを実行したことではない。

| 既存の実物・根拠 | 実際に確認できる範囲 | まだ断定できないもの |
| --- | --- | --- |
| [quote Go Process](../../plugins/reference-order-quote/README.md)、[Process比較テスト](../../tools/reference-capability-transport/quote_go_js_vectors_test.go)、[cross-language](../../reference/quote-cross-language/README.md) | 同じ未受理23ケースでGo/JavaScriptの独立したProcess Providerを既存stdio Carrierで比較するテストが存在。Goは隔離されたGo moduleからBuild。入力重複キー・大整数・欠損・業務拒否とProcess失敗を分離する負例もある | 最新PR exact-headの再実行、Go静的D1との一対一費用比較、全23件の正式Provider/Host Grant適合 |
| [Runtime実Process試験](../../tools/reference-capability-transport/runtime_authorized_quote_process_test.go)・[候補形→Runtime直列試験](../../tools/reference-plugin-capability-contract/order_quote_runtime_chain_test.go) | 特定の見積もり要求について、合成Shape/Provider証拠、実Runtime Registry、テスト用Digest検証、子Process、結果確認を結ぶ限定試験が存在 | 合成Publisher/Conformanceを正式Authorityとみなせない。実Sandbox/Lease、同一23件を通した本番GrantやOS実隔離 |
| [Component候補](../../reference/quote-component-candidate/README.md)・[専用CI #38746502755](https://github.com/hat47x/tei-platform/actions/runs/38746502755) | 2026-10-15、固定HEAD `7a3b65d228937d2df56b11afe0ed1fbc4b5936ea` で成功。後続の[PR #1878](https://github.com/hat47x/tei-platform/pull/1878)も[専用CI #38746880110](https://github.com/hat47x/tei-platform/actions/runs/38746880110)で同様の限定実証に成功し、merge commit `787a05e5ad5573db28fd4a617404dd4124bea279` でmain統合済み。Node/実ChromiumのJco変換成果物、Wasmtimeの元Component直接呼出し。正常8件／Ingress拒否7件／Guest拒否8件（Guest呼出しは各Host16件） | 同じ生ComponentをBrowserが直接実行したわけではない。未受理業務意味・Production Authority・全言語/全Target・このPRの現在HEADのPASS |
| [未受理23候補](../../reference/candidates/order-quote/semantic-vectors-v0.json) | 同一Contractの固定Expectedとして複数Referenceに再利用。業務拒否・入力解析拒否・Runtime起動失敗は別義務 | 独立検証基準のMutant検出率、業務Owner正式受理、別意味版の互換性 |

**P0で実施すべきこと**は、新しいPlugin Engineではなく、(1)実23ケースのSource/期待値ファイルのRevision・ケースごとの拒否段階・既存Test/CIのHEADを一枚の既存#2514/#1180 Evidence参照で照合、(2)現行main exact-headの既存Go Process/異言語/候補Componentのうち実行可能なものを再資格化、(3)D0の純計算とD1の静的Providerを**同じGo Target/同じ計算意味・独立検証基準**で検証する最小差分を#1251/#951の現行契約上で特定する、の順。現行Go見積もりSourceは `plugins/reference-order-quote/main.go` の独立Process `package main` であり、単にそのpackageをimportして静的Provider検証済みとは言わない。再利用を可能にする抽出・Adapterが本当に必要かを測定し、業務Rule/Expectedの第二正本を作らない。

<!-- tei-quote-preflight-skip-nonexecution-gate-20261106 -->

**次回実証で先に固定する最小の実行判定**：既存 [見積もり異言語比較](../../reference/quote-cross-language/README.md) の次の一つを、対象Source/Toolchain/OS/Node版・実Provider/期待値ファイルの原byteを固定した独立Attemptで再実行する（これは既存の実Process比較であってD0/D1比較ではない）。

```sh
go test -count=1 -run '^TestIndependentGoAndJavaScriptQuoteSemanticParity$' ./tools/reference-capability-transport
```

この既存TestはWindows環境またはNode未導入では `t.Skip` し得る。したがって **Goコマンドのexit 0だけを「23件を実行しPASS」としない**。Test自体のPASS/SKIP、独立したGo/Node Process双方の実起動、業務受理/拒否の内訳、実SourceとArtifact digest、Targetを分けて記録する。未起動・実行Skippedならその部分はNOT RUN。

**Static D0/D1の証拠上限**：この既存Testは双方とも `process-stdio-v1` Carrierによる**独立子Process**比較である。Go ProviderがGo製であることだけでは、同じGo Binaryでのin-process D0（通常関数）とD1（静的Provider）を実測したことにならない。元の計算実装とExpectedの分離を保持した最小比較を行い、追加価値が生じない場合はPlugin不要案を残す。

**採否の補正**：D2の既存Process成功だけでD2標準化を決めない。D3の過去Component CI成功は限定的なHost/Contract/版に結び付け、現PR exact-headの資格にはしない。次に効果が大きい未解決は同じGo RuntimeのD0対D1と、独立Providerを必要とする実Consumerの証拠である。

**状態**：ここでは実Source/既存コマンド/非実行の停止線を文書に整理した。現HEADでの実Process再実行、D0/D1比較、性能計測、正式Owner受理はNOT RUN。

<!-- tei-static-d0-d1-first-executable-slice-20261106 -->
### 18.9 D0通常関数とD1静的Provider：一改訂・二入口の最小実証手順（D0限定CI成功／D1実装前）

これ以上抽象的なPlugin比較表を増やさず、§18.1の**同じ見積もり計算をin-processで二つの経路から呼ぶ**ことを最初の未実証差分とする。業務意味・未受理検証基準の正本は既存 [23ケース](../../reference/candidates/order-quote/semantic-vectors-v0.json)。新たな意味Ruleや別Expected、公開Plugin ABI、実行用常駐TEI Hostを作らない。既存Go Process実装 `plugins/reference-order-quote/main.go` は `package main` なので、静的Targetへそのままimportできると仮定しない。

| 最小手順 | D0／D1の一改訂の実証方法 | 未実証なら停止するClaim |
| --- | --- | --- |
| **S0 入力と計算の固定** | 23ケースのraw JSON byte、独立Expected出典、拒否理由の層、計算上限/単位/NullとSource Revisionを固定。元 `calculate` の実装をread-onlyに解析し、共有可能な純粋計算部分とwire/parser/Process固定部分を分ける。 | 23件一致からOwner Accepted MeaningやProvider適合を推定しない |
| **S1 D0 direct entry** | 同じGo Target・Toolchainの最小な直接関数入口で計算する。Domain検査や入力Lexeme検査は既存意味から欠落させない。原実装と別の業務Ruleを再実装する場合は公平性が崩れるため保留。 | 同じ関数を呼んでいない状態でLatencyを比較しない |
| **S2 D1 static entry** | 同じ純粋計算本体を用い、薄い明示的なService/Provider選択・Adapterを通じて**同一Process／単一Go Binary**内で呼ぶ。現行Contract/Compositionで不可能なら必要な最小差分をFit/Gapとして記録し、Core共通ABIの先行追加はしない。 | 子Processへフォールバックしたものを静的D1と報告しない |
| **S3 独立Expected** | D0とD1双方に未受理23件を同じbyteで適用。受理8、拒否15を検査し、入力拒否／業務拒否／技術失敗／未実行を別にする。#2514の誤計算・重複Key受理・業務拒否の誤写像Mutantを流用し、検出/未検出を記録。 | 両実装の一致だけで検証基準独立性や業務受理を保証しない |
| **S4 実Targetの差** | D1選択の実Contract/Provider、Go import/init/embed、実Build閉包、Binary/member digestと実ロード入口を照合。未選択実装追加・必須依存欠落を#1251 PB/PBNで対照にする。 | Bundle解決・root test PASSを実Link/除外/全module buildのPASSとしない |
| **S5 費用と代替** | 同一CPU/OS/Toolchain/revision/環境でclean/incremental build・Artifact bytes・同一Processのwarmup/反復呼出時間、追加Adapter・設定・更新箇所を測定。外部Process版D2の起動・IPC費用は**別列**で扱う。 | 無測定の性能優位・単一Runによるp95・Go/Node差によるPlugin overhead主張をしない |
| **S6 採否とConsumer** | 実際の第二Consumer/交換・更新要求があるか、選択/維持費用が削減されたかを#1016/#1251に返す。必要性が無ければD0を選ぶ。 | 仮想Consumerのみで静的Plugin方式の標準化を決めない |

<!-- tei-quote-d0-build-copy-compatibility-gate-20261106 -->

**現main実装を壊さない最初の実装切り分け（コード照合、NOT RUN）**：`plugins/reference-order-quote/main.go` は `package main` で、`calculate(raw []byte) (quoteResult, error)`（約202行）に純粋なJSON decode・業務条件・合計計算が集約され、`serve()`（約226行）以降はCarrier/Dispatch/Process用の境界になっている。一方、既存 `tools/reference-capability-transport/quote_go_js_vectors_test.go` は `plugins/reference-order-quote/` の各 `.go`（READMEを除く）を一時Go moduleへ**ファイル単位でコピー**し、offline・`GOWORK=off`・`GOPROXY=off` で独立Buildする。したがって `calculate` を新しいRepo内共通packageへ移しただけでは、この既存D2の完全なoffline-buildとSource閉包が維持されるとは限らない。

- **先に行うD0の小さな検証**：現 `package main` 内の`calculate`をプロセス起動・stdio/Transportなしで直接呼び、既存`reference/candidates/order-quote/semantic-vectors-v0.json`のraw Requestをそのまま使用。正常8件は既存Expectedの通貨/件数/小計、拒否15件は `error` を確認し、入力解析拒否／業務拒否の厳密な区分は既存の検証基準・Process応答層の証拠と別記する。JSONを事前に一般`map`へ変換して重複キーや数値lexemeを失わない。**これはD0のみであり、D1 PASSと記録しない**。
- **D1の導入条件**：既存 `internal/plugin.Catalog.ResolveEnvironment` が返す `ResolvedSet.Capability` は解決済み候補であり、特定のquote実装が静的Go成果物に**実リンク・呼出**されたことを証明しない。既存 `RuntimeBuiltin` 記述だけをD1の実実装とみなさない。D0と同一 `calculate` を必要な静的Provider/Service入口経由で呼ぶ実Consumerを用意できるかFit/Gapを記録。新共通packageへ抽出が必要なら、**D2のoffline module stagingと既存23件Process回帰を同時に保つ変更**を同一PR・同一HEADで確認する。事前抽出や第二計算実装は行わない。
- **新HEADの実行判定**：D0 direct単体Go test／既存D2 process test／将来D1 actual static buildとInvocationを別のGateにする。`go test` exit 0でもWindowsまたはNode不存在時のD2 `t.Skip`は実行PASSではない。新しいテストでbuild-timeの静的関数をfunction pointer経由で呼ぶだけなら「Adapter dispatchの合成確認」であり、Plugin選択・Link・Builderを含む製品D1として報告しない。比較対象の同一Target/Toolchain、23件のSource byte、Binary digest、未選択Plugin混入・依存欠落を#1251/#2600で確認する。
<!-- tei-d1-registry-build-metadata-vs-linked-entry-gap-20261106 -->
**S2の実装Fit/Gap（固定された現main Sourceの限定監査）**：`internal/composition/registry.go` の `Provider{ID, Scope, Offers, Requires}` は提供者の宣言と依存Eligibleを扱うもので、同ファイルの `Assessment` にも「actual process readiness/artifact trust/permission/activated instanceではない」と明記されている。`internal/plugin/resolve.go` の `ResolvedSet.Capability` も候補解決までであり、`RuntimeBuiltin` の指定だけでは選択ProviderのGo関数入口は存在しない。`internal/plugin/build.go` の `BuildPlan` はexact Source/Builder/Targetを記述し、`LockBuildOutput`/`VerifyRuntimePluginBuild` は指紋/Lock整合を検査できるが、**それら自体はGo compilerの起動、実リンクされた関数の列挙、Consumerによる実呼出しを実行・証明しない**。この三つの独立面の間が、D1の現時点のFit/Gapである（#951/#1251/#1167）。

**最小のD1負例**：AとBの同Contract Provider候補を用意し、(1)Bの選択記録だけがあるが実TargetにAしかリンク/呼出されていない、(2)Bの必要Source/Moduleが欠落、(3)未選択Bの追加でA成果物や通常D0出力が変わる、を独立に拒否/影響限定できるか測る。`composition`のEligible、`plugin`のResolved/Build Lock、実Go Binary digest/LinkとConsumer結果、current Host Grantをそれぞれ別のEvidence列として持つ。**同名Providerの署名・Registry選択やArtifact Digest一致をD1実InvokerのPASSへ昇格させない**。

**実装時の停止線**：現在のGo見積もりは`package main`であり、安易な共通package化は既存D2のoffline stagingを破る。まずD0直接結果を固定し、明示された第二Consumerが要求する場合だけ静的入口の追加と必要な共有化を同一PRで行い、既存D2の無回帰とGo import/Linkを同じHEADで実測する。初回は新しいCore Plugin ABI/Registry/Host、全Provider共通起動サービス、UI/Tenant固有Policyを導入しない。

**最小の実行出口**：同一Revision/Expected/OS/TargetでD0とD1のactual Go Build/Invocation、23件のExpected別Outcome、少なくとも一つの意味負例と一つの選択/依存負例、選択された/実消費されたSourceとBinary、費用を結んだ**既存Evidenceへの参照**。D1の静的呼出しが成立しなければその不適合と必要差分を成果とし、動的Loader/Processへ自動迂回しない。試験結果のOwnerは#2514（検証基準 Database）、#951（Contract/Conformance）、#1251/#2600（Build/Bundle/費用）、#1016（実要求/採否）、#1180（Scenario SSOT）。本節は新実装の合否ではなく作業の最小仕様。**S1のD0は別Draft #3021の修正HEAD `dfc133e2...`でroot Go Testと既存D2 Process CIの限定PASSを得た。S2の実静的D1、S3の独立Mutant検出、S4〜S6の実比較・費用測定、正式Owner受理はNOT RUN。**

<!-- tei-d0-ci-actual-oracle-scope-negative-20261106 -->
### 18.10 実CIで判明したD0/D2の期待値ファイルの配布境界（2026-11-06、現物故障の反証）

新[D0直接計算Draft #3021](https://github.com/hat47x/tei-platform/pull/3021)は `plugins/reference-order-quote/candidate_vectors_test.go` から既存23ベクトルを参照する1ファイルのGo試験。**初回exact HEAD `faee399c6c23d02300d3cd8988005d0f20ab0148` の実CIは失敗**した。独立[Actions #42196353181](https://github.com/hat47x/tei-platform/actions/runs/42196353181)と[Health #42196353500](https://github.com/hat47x/tei-platform/actions/runs/42196353500)のジョブログで、root `go test ./...` 中の `tei/plugins/reference-order-quote` 単体は `ok` であった一方、既存 `TestIndependentBusinessQuoteOverExistingGenericProcessTransport` が一時Go moduleへ全 `.go` をコピーして再実行した際、**候補の期待値JSONはコピーされない**ため新D0試験の相対path読取りが `no such file or directory` で失敗したことを確認。業務計算の不一致ではなく、**Test Sourceの依存とProcess配布閉包の適用Scopeが違う**問題。Go vet/gofmt合格・脆弱性CI成功を、この失敗の解消や23件の新HEAD合格へ読み替えない。

**局所修正**：#3021 の新試験だけで、既存D2が生成する `go.mod` の `module example.invalid/reference-order-quote` を厳密に識別できる場合に限り、D0共有検証基準試験を `t.Skip` する。TEIルートからの直接D0試験で検証基準が見つからなければ従来どおり **hard fail**。実Providerコード、D2 Staging (`.go`全件コピー/Offline Build/元の内部単体試験)、既存の検証基準を改変しない。修正コミット `dfc133e2ce1877e3593f1e507cbc3c2b7a82bfc5`。**修正後exact-headのCI 3件はすべて completed/success**：[Isolated #42203769225](https://github.com/hat47x/tei-platform/actions/runs/42203769225)、[Repository Health #42203769269](https://github.com/hat47x/tei-platform/actions/runs/42203769269)、[Dependency #42203769210](https://github.com/hat47x/tei-platform/actions/runs/42203769210)。取得したJobログでPR_HEAD_SHA一致、root `tei/plugins/reference-order-quote ok`、全Root `go test -count=1 -timeout 60m ./...`・`go vet ./...`・`gofmt`、既存 `TestIndependentBusinessQuoteOverExistingGenericProcessTransport` の限定独立Processジョブが成功したことを確認した。CIの通常ログは各23 subtest名を個別表示しないため、**23件の各結果についてはtest Sourceのloop・root go.mod条件とGo package PASSに基づく範囲**とし、別の独立検証基準受理やMutation検出実測はまだ主張しない。

| Claim | 必要な合格証拠 | 誤判定の禁止 |
| --- | --- | --- |
| D0 root direct | ルートSource上で共有23件（成功8・拒否15）の各subtestを実際に実行し、SKIPなしでExpectedに一致 | Standalone stagingでのD0 `t.Skip` をD0 PASSへ算入しない |
| D2 standalone process | 固定Sourceから独立offline Go moduleの `go test`/`go vet`/`go build` が成功し、元Carrier/Host/実子Processの限定ケースが実呼出しされる | D0共有検証基準がコピーされないことをD2の不完全な業務Sourceと断定しない。逆にD2試験PASSを全23 D0の証拠へ転用しない |
| D0/D1比較 | 同一23 検証基準・同じGo Target/CPU/Toolchainで直接入口と明示静的Provider入口の実Build/実Invokeを比較 | D2のProcess又は `ResolvedSet` 合成選択をD1静的Link資格としない |

本負例は§18.4（必要Source閉包）の**Consumer/Attempt相対性**と§18.5（同じ検証基準の再利用）を補強する。Second 検証基準/Host/Resolver、D2の全Test Source排除、Processの新Runtime実装は行わない。正式業務Owner受理、独立Mutant、D1、実OS/DB/Host資格は別。#1251（Go module/Build閉包）、#2514（検証基準実実行）、#1145（Process境界）、#2600（実消費と再資格）へ引き渡す。

<!-- tei-d1-d2-hard-constraints-before-benchmark-20261106 -->
### 18.11 D1／D2の比較前にハード要件で候補をふるい分ける（他レーン§23との責務分離・未採用）

他レーン[DSH横断Draft #2903 §23.3](https://github.com/hat47x/tei-platform/pull/2903)が整理した二法人／別Tenant／Grant撤回／部分効果の**方式別Stop条件**を、方式選定側の§18.6へ適用するための最小関門。DSH §23のSCOPE-1〜4を新たな業務Fixture正本へコピーせず、実行方式の不適合判定だけをこの節に保持する。D0、D1、D2、動的Load、Wasmを性能の高低や開発の成熟順で並べない。

| 用途で先に確認するHard requirement | D0／D1静的Goへの帰結 | D2 Process／動的Loaderへの帰結 | 必要な独立証拠・未満時の扱い |
| --- | --- | --- | --- |
| **同一成果物だけを厳格に配布**し、外部実行ファイルを配置できない | 通常関数D0と静的選択D1は候補。D1は現Go Targetの選択・実Link/Invocation・不要initの排除を証明 | 独立Process D2は配布条件に不適合（別の同梱方式を黙って『単一バイナリ』とは呼ばない）。Wasmを内包しても実Host要件は別 | 配布契約・Targetのactual member/Artifact bytes。条件が矛盾する場合は要件変更のOwner判断まで保留 (#1251/#1167) |
| **停止・再ビルド不可の独立Hot Swap**を要求 | 同一Go Binaryに固定された通常D0／静的D1では満たせず、その用途では棄却候補 | D2や動的方式は候補だが実drain／再選択／世代切替／継続リクエストの挙動が未実証なら保留 | 実Consumer負荷・失敗注入・旧新世代のEffect/Receipt。単にPlugin Versionを変えてもHot Swap実証にならない (#1223/#2604) |
| **不信Providerを同Processから強制隔離**する必要 | 信頼されないコードに実権限が届くならD0／D1は原理的に不適合。論理Contract分離だけではメモリ・File・Secret権限を隔離できない | D2のProcess境界は候補だが、OS Resource/Network制限、Secret継承・IPC／Tenant Bindingの強制が無ければ採用不可。in-process動的Loadも隔離証明の代用不可 | 実OS強制拒否・漏えい負例・攻撃対象Resource。manifest/read-only自己申告をSecurity PASSにしない (#2515/#2509/#2903) |
| **Tenant別Grantの撤回をCommit時まで強制／部分Effectを復旧** | D0/D1も実Host/DB fenceに依存。Library/CompositionからGrantを推論しない | D2も同じ。別Process起動やACKはcommit資格・効果確定・duplicate suppressionでない | SCOPE-2〜4、実Issuer/Subject/Tenant+Target/Grant epoch、Operation、Effect Receipt、独立Read。Host/Provider間の失効原子性保証が無い場合、厳密要件なら方式を保留/棄却 (#1174/#1129) |
| **契約だけを先に配布、実装は案件後付け** | D0で用途が満たせる場合はPlugin不要。Contract資産と静的実現の段階は分離 | D2/Wasmを先行義務化しない。外部Consumerが実際に要求したTargetだけ追加 | Provider 0のinspectと呼出拒否→Candidate→独立Conformance→実Targetの順。公開ABI・Marketplace先行禁止 (#1975/#2390/#951) |

**判定順序**：①Consumer要求からHard/Soft制約と出典を確定 → ②Hard違反方式を比較対象から除外／不明なら保留 → ③残る方式だけで同一ExpectedのS0〜S6・実費用・更新境界を比較 → ④Ownerが業務意味、Hostが実行資格、Target管理者が配布要件をそれぞれ判断。**厳格単一Goバイナリと無停止個別Hot Swapが同時に絶対要件**で、実現可能な代替配布仕様も無いなら、比較用の架空D1/D2を増やさず**現要求は同時充足不明／不適合**と記録して差し戻す。

**現在の試験適用範囲**：[D0直接Go Draft #3021](https://github.com/hat47x/tei-platform/pull/3021) のCIは純粋計算のReference成功と既存Process回帰までであり、上表のHot Swap・OS隔離・Tenant/Commit/Receiptの試験ではない。D1 staticのS2/S4、実費用のS5、SCOPE-1〜4の実Provider/OS/DB、Owner正式受理は **NOT RUN / NOT ADOPTED**。採否の正本は本§18.6/§18.11、Scope負例の正本は#2903 §23、Host/Effectは#1174、Application選択は#1016。第二Policy/Host/Conformance Engine/新Fixture正本は作らない。

<!-- tei-plugin-md-migration-source-qualification-20261106 -->
### 18.12 `*.tei.md` 方針をPlugin/Bundleの入力閉包へ反映する（Authoring移行への依存条件）

2026-11-06のユーザー方向決定は、**TEI Authoring Sourceは `*.tei.md` を標準とし、旧 `*.tei` は段階廃止**。正式な移行計画と実装Ownerは[Draft #3030](https://github.com/hat47x/tei-platform/pull/3030)（M0〜M6、TM-01〜14）／#2621/#988/#2823。ここでは一般Markdown ReaderやDSL構文を再設計しない。対象は**Plugin/Bundle/ProviderのRequired Source、Actual Source Loaded、Build Input、実Target再資格**へ限定する。現mainの `internal/workspace/loader.go` は `plugin.tei`/`application.tei`等の固定名と `*.tei` glob、`filepath.Ext` に依存し、`*.tei.md` は単に`.md`として分類され得る。現在のLoader/Parserで新名ファイルを置くだけでは、Pluginの実Read・意味・Build/Conformanceに到達したとは言えない。

| 依存するPlugin Claim | 最小の観測・禁止推論 | 引渡しOwner・既存負例 |
| --- | --- | --- |
| **Contract/Descriptor Sourceが発見された** | `*.tei.md`/Legacy `*.tei`/通常`.md`を区別し、Declared/Discovered/Loaded/Parsed/Qualifiedを別列とする。Catalogに同名Providerがあるだけで現在のSourceが解析されたとしない | #3030 TM-01/02/12/14、#2822/#2823、#988 |
| **旧`plugin.tei`→`plugin.tei.md`で同じPluginを選択可能** | 新旧双方の実Loader対象・元byte/span・内部TEI領域、Source occurrenceとMeaning/Contract versionを比較。二つ存在する場合は曖昧性を検出して明示選択を要求、先勝ち/後勝ちで解決しない | #3030 TM-03/04/06、#951/#1016 |
| **Authoringの整形・Prose変更とBuild再利用** | 文書全体のraw revision、選択TEI部分の実byte/span、Role/Meaning受理、実Build/Target digestの一致/不一致を別投影。表示内容不変・同じMeaning IDだけで旧Source受理/旧Build/旧Conformanceを再資格化しない | #3030 TM-05/08/09、#2600/#1251/#2514 |
| **新`.tei.md`をProviderが実際に消費した** | `filepath.Ext==.md`や単なるread-only Workspace Indexの成功を実Parser/Provider Consumeと混同しない。Build-time用Schema/Loaderが存在しないなら `not connected/unsupported` で停止 | #3030 M2/M3/M5、#2823、#1167 |

**段階導入**：M0 inventory／M1新規Source標準化の判断は#3030へ集約。M2 Markdown span readerとM3 Loader対応が実装・独立試験される前に、既存`*.tei` Assetの一括Rename・Legacy拒否、固定Gate原典の変更、Plugin名からの暗黙推定を実施しない。M4以降、Source Location差・改訂・決定受理範囲を照合した後だけSelected Contract/Build再資格へ進む。`.tei/decisions/`・`tei.workspace.yaml`、既存23件JSON形式の期待値／Go直接計算・D2 Processは移行対象外。`*.tei.md`と同一ではない `.md`文章やOKFタグをPlugin Source Acceptedと扱わない。

<!-- tei-markdown-unsupported-explicit-ci-20261107 -->
**Parser未接続の実失敗境界（限定試験）**：[Draft #3031](https://github.com/hat47x/tei-platform/pull/3031)のHEAD `63953ccb5c12ead26ec1268e7c0afdba4eff66ae` では、未対応`*.tei.md`のMarkdown本文をTEI構文として成功扱いしないfail-closed境界が8つの[対象CI](https://github.com/hat47x/tei-platform/actions/runs/42217379093)等で成功している。これは**不正な実行資格への昇格を拒否できた限定証拠**であり、実Markdown AST/フェンスのReader、Plugin DescriptorのParsed、Source span/Build Actual Consumptionが成立した証拠ではない。従って新形式のD1実証を完成扱いせず、既存Go Source/23候補のD0/D1比較は別に進められる。

**証拠上限**：新Authoring形式の方向決定は把握済みだが、Plugin Descriptor/Reader/Loader・Markdown/TEI Lexer・Source Index/Build Select・Studio/SHOを接続する新試験は **NOT RUN**。方式選定の実証D0/D1/D2にも移行完了という資格を上乗せしない。対応実装のOwnerは#3030/#2621/#988/#2823、Bundle/Build観測は#1251/#2600、本§は方式比較の停止条件のみ（新Parser/Registry/Authority/Fixture正本なし）。

<!-- tei-d1-claim-relative-witness-join-20261107 -->
### 18.13 D1静的Providerの最小接続票：個別PASSを同一実行の資格へ昇格させない（JOIN-05適用）

他レーン[Draft #2903 §24 / JOIN-05](https://github.com/hat47x/tei-platform/pull/2903)で新たに明示された**Evidence接合の対象一致**を、初回D1静的Go実証のS2/S4へ絞って適用する。JOIN-01〜06や六Scopeを複製せず、ここでは同じquote計算を呼んだつもりでも**選択Bと実Call Aが食い違う**場合を検出する。仮のEvidence Schemaや新Registryは作らない。

| 接続点（同じ実Consumer／Target） | 必ず照合する関係 | 不一致・根拠不足の場合 |
| --- | --- | --- |
| **S0→選択** | ReferenceのContract ID/意味・Input raw bytes/Expected、ApplicationのScope/Target、`#951`の実Selected Provider ID・固定版 | 名前が同じでも意味/Contract版/Scopeが違うなら結果一致をConformanceへ昇格しない |
| **選択→実Build** | Selected Provider Bに対応する必要Source・Go package import/init・Target Build入力/Toolchain・実Binary digest（`#1251/#1167`）、Build Lockの参照 | B選択なのにAだけリンク、又はA/B両方収録の理由未説明ならD1未資格。宣言済みArtifact fingerprintだけでは実Linkを証明しない |
| **実Build→Invocation** | 同一Binary/実呼出し入口と、今回のInput Digest/Contract/Provider B・結果、実Consumerが使用したTarget・実行版 | Aの計算結果がたまたまBと同一でも、BのProvider実行PASSにはしない。単なる`composition.Assessment.Eligible`をInvokeに接合しない |
| **結果→Claim** | 23候補（8正常/15拒否）のExpected出典・拒否層、D0直接関数/D1実静的入口/D2別Processの**それぞれのAttempt** | [#3021](https://github.com/hat47x/tei-platform/pull/3021)のD0/D2 CI成功をD1と結合しない。純粋計算に不要なGrant/Effectを必須条件へ捏造しない |

**一因子の負例と合格出口**：Provider A/Bは同じContract・同じ見積結果を返すが、Bを選択したTargetでAの関数を呼んでいる対照を置く。結果値だけを比較する検査は偽陽性になるため、**実選択→Build member→実呼出し入口→対象Attempt**の一致が証拠として得られる場合だけ「このTargetでBが実呼出しされた」と判断する。別条件で必要ProviderのModule欠落、未選択Bの`init`混入を#1251 PB-01/07/10へ返す。`go tool nm`等は補助であり最適化によりsymbolが見えない場合は実Call witnessで確認する；nm不存在だけでBuildに含まれないと断言しない。

**Claimごとの非適用項目**：quoteは純粋なread-only計算なので、実DB Read/Commit、Host Current Grant、Receipt、Tenant Record集合は本D1測定では`not applicable`（存在しない権限を得たとの意味ではない）。別の業務PluginがDB/外部送出を行うなら、このD1のPASSを再利用せず、#2903 §24のMeaning/Scope/Grant/Read Coverage/Effect/Disclosure接合へ戻す。`semantic.EvaluateCompletion`（main統合[#2926](https://github.com/hat47x/tei-platform/pull/2926)）も呼出側が供給するCoverageを証明しないため、Quote Resultの行数や一致からBusiness TargetSetの完全性へ飛躍しない。

<!-- tei-provider-source-proof-crosslane-20261107 -->
**隣接Referenceの正負対照から得た検証要件**：[Language Bootstrap Draft #3028](https://github.com/hat47x/tei-platform/pull/3028)のBS-03は、**同一Source・同一Parser callback**に対しProviderのImplementation Fingerprintのみを不一致→一致へ変え、不一致ではCallbackを呼ばず、一致なら実呼出しまで到達する限定Go試験を追加した。exact HEAD `aa51946a8d979aa1c3e0b3648b96df1de44cf812`の[健全性CI #42240218716](https://github.com/hat47x/tei-platform/actions/runs/42240218716)と[依存検査 #42240218680](https://github.com/hat47x/tei-platform/actions/runs/42240218680)はともにsuccess。ただし対象は**Language/Parser Candidateの結合**で、D1のGo静的Link/実Target/23件の適合判定ではない。D1でも同じく「不一致を拒否する」だけでなく「一致なら選択Provider Bへ実Callできる」正例を必ず対にし、常時拒否の偽陽性を排除する（#951/#1251）。

**実証状態**：#3021修正HEAD `dfc133e2`のCI 3件成功はD0／既存D2の固定証拠。上記A/Bの実static link、実Invocation別Witness、ビルド差分、費用・Mutant、外部Target/Grant/Receiptは**NOT RUN**。Source位置、完全性、Owner Accepted Meaning、正式Contract採用は別Owner。D1実装判断は#951/#1251/#1016へ返し、Plugin化不要案を残す。

## 19. 高凝集Bundleの適正規模と実行前提の完全性（2026-11-06、設計・未採用）

### 19.1 Bundleのサイズはファイル数ではなく、実変更の結合度で判断する

§18の同一要求D0（Plugin不要）/D1（静的Provider）/D2（Process）比較は維持する。Bundle分割を採否する際は、#1251 PB-09に次の**実測可能な反証**を付加する。細分化と統合のどちらも既定の正解にはしない。

| 観測軸 | 高凝集Bundle案が成立する条件 | 過剰な統合・分割の兆候 |
| --- | --- | --- |
| 利用者の操作 | 関連機能の導入と更新を一括で説明できる | 互いに無関係な機能の選択・設定まで強制する／分割しすぎで依存設定が増える |
| 変更の波及 | 共通契約・UI/DSL/Generator等の関連変更を局所検証できる | 未使用実装の改版で無関係TargetのBuildが変わる |
| Versioning | Bundle版と内部Contract/Implementation版を追跡できる | 一つの実装パッチで全契約を意味的に再受理させる |
| 依存・ビルド | 実Selected/Build-consumed/LinkedがTargetごとに明示できる | 共通packageのinit・blank import・embedや全件登録で未使用実装まで混入する |
| 権限・作用 | 同梱したread-only解析とSource書換／DB Writeの作用を別々に制御できる | Bundle全体の承認・署名だけで個々のResource/Effectを自動許可する |
| 保守費用 | 同じ固定Expectedで変更箇所・試験/配布・復旧の負担を比較できる | 分割/統合に測定可能な利益がないのに新ABIやResolverを増設する |

比較するのは同じ機能・同じ業務Source/独立Expectedを実現する、(A)通常Library/標準Data Operations、(B)細分化Plugin群、(C)高凝集Bundleの少数の対照。B/Cとも論理Contractと実選択・実消費の区別は共通。独立配布や別Targetへの展開要求が存在しない純粋演算やOwner RuleはAで完了してよい。サイズを一律のmodule数・LOC閾値で固定しない（#1251/#2018）。

### 19.2 既存Goコードから確定した上限：ResolvedSetはFunction-usedの証拠ではない

mainの `internal/plugin/resolve.go`（照合byteは別Evidence）で `Catalog.ResolveEnvironment` は選択Bundleの `Plugins` 要求をキューに展開し、依存を解決して `ResolvedSet` を作る。`internal/plugin/model.go` の `Bundle{ID,Version,Plugins}` と旧 `CapabilityKind` は現状の実装形状であり、将来の一般Plugin ABIを決定する契約ではない（#951/#1223）。**Bundle内の全Requirementが解決できたことは、特定Functionに必要、Go Binaryへリンク済み、現在Grant取得済みのいずれの証明でもない**。実利用とBuild closureの追跡は#1251/#1089/#1167/#2600へ委譲する。

### 19.3 Capabilityを選択できることとRequired Guardを満たすことを分ける

他レーン[Draft #2889](https://github.com/hat47x/tei-platform/pull/2889)は、Go `internal/execution.RunPlan` への**toy Step/Journalを使った限定的な実Go特性テスト**。専用CIでは3件PASSが報告されているが、実DB/Host認可/Owner受理ではない。特徴付けの内容は、Callerが期間チェックだけを渡してGrantチェックを提供しないとtoy Stepが `completed` になり得る、Observerの存在は実Effect観測ではない、先行Step後のblockは全Effect=0の証明ではない、という境界である。

ここからPlugin連携の受渡し票には、当該操作の**Required Guards（選択済みContract/Meaning/Targetから必要と判断した検査）／Supplied Guards（Callerが渡したもの）／Evaluated Guards（実際の現在版・対象に対し評価したもの）／Effect・Consumerの独立Evidence**を別欄に設けることを提案する。必要集合の導出自体が不明なら `unknown` として実行資格を主張しない。`preconditions != nil`・Pluginの `read-only` 宣言・Provider署名・RunPlanの `completed` だけでGuard completenessを証明しない。全Operationへ同じGuardを強制せず、純粋計算には該当しない効果検査をnot-applicableとする。

**最小負例の割付**：#1130/#1129/#1174の既存RunPlan・Effect Fenceで「必要Grant未供給」「Observer存在のみ」「途中block後の先行結果」を区別。#951は選択Contract/Provider・Capability要求の束縛、#1251はBundleが増えても権限が増幅しないこと、#2514は独立Expectedの品質を担当する。第二RunPlan/Guard Registry/TEI共通Hostを作らない。実Provider Grant・Commit/Receipt・外部HostとWindows Studioは別の実受入を必要とする。

### 19.4 採否と証拠の状態

まず#1251 PB-09の同機能A/B/Cで導入設定数・Source変更量・必要Target/Build依存・実Binary/Artifact・検証範囲を比較し、その次に#2889の既存Guard不足の負例を対応するPlugin経路に当てる。比較は独立Expectedを先に固定し、費用/Latencyの数値を捏造しない。Pluginの粒度・公開Contract・Runtime方式は正式採用していない。**今回の§19はDocs-only、Go/DB/Hostの新実行、Bundle比較実測、Owner意味受理はNOT RUN**。


<!-- tei-dsh-third-party-plugin-supplychain-20261106 -->
## 20. DSH型Pluginエコシステムに学ぶ：外部供給・選択・利用責任の分離（2026-11-06、設計・未採用）

### 20.1 DSHから取り込むこと／取り込まないこと

DeepSeek Harness公式[Architecture](https://github.com/deepseek-ai/deepseek-harness/blob/master/docs/architecture.md)と[Cordis Primer](https://github.com/deepseek-ai/deepseek-harness/blob/master/docs/cordis-primer.md)（2026-11-06参照）により、Service key、typed event、inject依存、ProfileとBundle・構成overlay、解除可能な登録を確認。**採るべき本質**は「第三者が能力を提供し、Consumerが必要な契約を選択し、実ライフサイクルを観測できる拡張面」であり、TEIであらゆるデータ/業務RuleをRuntime Pluginへ置換することではない。DSHのordered overlayによる行置換をTEIのBusiness Meaning/Provider/Grantの暗黙勝者選択へ流用しない。DSHの動的Plugin Tree、HMR、常駐LoaderをGo単一Binary/外部Java/.NET/Wasm Targetへ義務づけない。

### 20.2 外部Contributorを受け入れるための最小公開情報（新Manifest Schemaではない）

| 主体 | 提供・確認するもの | それだけで成立しないもの |
| --- | --- | --- |
| Contract提案者・保守者 | Contract ID/意味版、型・NULL・Failure・Effect/Scope、Source根拠、独立正負Conformance Case、互換/破壊的変更条件 | 特定案件のOwner意味受理 |
| Provider/Generator供給者 | Implementation exact digest、出所・供給主体、対象Target、Build input/Toolchainと外部依存、Resource要求、失効・撤回方法 | Contract適合、実行Grant、Runtime安全性 |
| Bundle配布者 | 高凝集な実装の選択可能集合、内包Module/契約・実装版とPin、ライセンス・既知の制限 | 全Moduleの自動選択・全Targetへの組込み |
| Project/外部Consumer | Required Contract、明示選択済みImplementation/Lock、実Build/Loaded Artifact、独立Expectedと採用範囲 | 最新Catalog候補の暗黙採用・永続的な現在資格 |
| 実Host/Provider | 実Principal/Scope/Resource/Grant、Invocation、Effect/Receipt/独立Read | Bundle署名・CI PASSだけで業務完了したという主張 |

**識別と資格を分ける**：Publisher identity/署名は出所、digestはbyte同一性、独立Conformanceは固定版での挙動、Host Authorityは現在許された作用、Owner Acceptanceは案件の業務意味。これらを「検証済みPlugin」という単一真偽値にしない。Community Repository/検索Catalog/配布Bundleの存在は選択済みTargetの利用・現在Grantを意味しない（#951/#1016/#1251/#2600）。新Marketplaceや中央Publisher Registryを先行導入しない。

### 20.3 自己記述Bootstrap・Dependency・更新時の停止線

[#2620](https://github.com/hat47x/tei-platform/issues/2620)の有限Bootstrapと自己記述Role/Operatorを使用し、外部Descriptorは**意味解釈前でも通常DataとしてInspect可能**にする。Required Shape、eager/deferred依存、Role/Operator ID+版、Source occurrence、実装digest、既存Pin/Lockを照合し、eager循環・未Pin推移依存・同名異byte・複数候補の曖昧性を診断する。合法なdeferred Graph参照を誤ってeager Plugin循環と扱わない。Descriptorが正しくParseできても、意味認定・公開契約互換・Host Resource Grantが成立するわけではない。

DSH型の解除可能な登録（Tool/Listener/Service）はLifecycle設計の参考になるが、Unloadによって永続DB Write、既送信通知、配布済みArtifactまで戻るわけではない。Candidate/選択変更・Listener/Resource cleanup・Build/配布・Business Effect/Receiptを別列で確認する（#1223/#1174）。

### 20.4 最小反証とOwner配置

| 否定対照 | 既存受入へ渡す判定 | Owner |
| --- | --- | --- |
| 同Plugin ID/versionだがDescriptorまたは実装byteが異なる | 実Pin/Lockに不一致があれば再資格化。単なるVersion文字列を信用しない | #951/#2600 |
| 未選択Bundleが同名Contractを追加 | 固定Selection/Buildへの非干渉を検査。Default競合を無音解決しない | #1016/#1251 |
| 配布者署名が正しくても意味・Effect不適合 | 署名だけでProvider Conformance/Owner受理/現在Grantを発行しない | #951/#2514 |
| DescriptorはInspectできるが推移eager依存が欠落 | Data閲覧を維持しMeaning qualifiedは保留。deferred参照は別扱い | #2620/#988 |
| Plugin unloadは成功、先行StepのDB効果が確定 | 登録解除と業務Effect rollbackを混同せずReceipt/独立Readで照合 | #1223/#1174 |
| 外部ConsumerでのProvider実Loadに失敗 | TEI Studio/CLIでの発見・Skill説明を外部可搬性のPASSへ転用しない | #2390/#2513 |

P0は公開可能なContract/Descriptor/Artifact/独立Expectedを既存資料からread-onlyでFit/Gap棚卸し。P1は同名異byte・候補競合・未Pin依存の既存負例と照合。P2は#2390のC0契約だけの公開とC1外部Consumer、#1251の実Go Build、必要時のみ#1223/#1174のLifecycle/実Effectへ進む。**本節は配布規約・第三者登録制度・Plugin ABI/Marketplace/中央Registryを採用するものではない。新Go/Host/DB試験とOwner受理はNOT RUN。**


<!-- tei-plugin-contract-first-release-vertical-slice-20261106 -->
## 21. 第三者Pluginの最小公開・互換性検収：Contractだけから後付けProviderへ（2026-11-06、計画／未採用）

### 21.1 一つのContractが通過する受入手順

§20は第三者の供給・受入責務を定義した。本節は**初回の実証に必要な順序と、公開してはいけない誤資格**を定める。既存[#1975](https://github.com/hat47x/tei-platform/issues/1975)の「ProviderなしのSQLite向け狭い契約→案件内候補実装→独立Conformance→隔離した実SQLite読取」を最小縦断とする。最初から全DB／全Target／共通Marketplaceの実装を要求しない。

| 検証段階 | 入力・結果の限定Claim | 不適合・保留条件（原Sourceを保持する） |
| --- | --- | --- |
| K0 Contract only | Contract ID/意味版・Read/NULL/Failure/unknown/Scope、必要Target・独立Expectedを公開。実Provider 0件でもinspect/参照できる | Provider不足を契約不存在・業務0件・解析成功と扱わない。実行要求はunresolved/unsupportedで停止 |
| K1 Candidate provision | 人またはAIが生成するProviderを通常の案件Source候補として作成。base Source Revision、変更対象、利用Scopeを明示 | AI自己評価・Publisher署名・Source位置の一致から採用やSource Patch実行を許可しない |
| K2 Resolution/Binding | 既存#951/#1016のProvider選択でContract/Implementation・依存・Targetと版を固定 | 同名/同版異byte、二つの同順位Candidate、欠落した明示SQL/Role依存を後勝ち・ロード順・最新候補で黙って決めない |
| K3 Independent conformance | Provider作者と独立したExpected、NULL/文字列・型・失敗・不正SQL等の正負Fixtureを適用 | Candidate実装とExpectedを同じAIが生成した自己PASS、未知やSkipをPASS件数へ算入しない |
| K4 Actual Provider read | 実SQLite版・exact Artifact・実QueryとRead scope・失敗/0行/部分結果・独立Readerを記録 | 合成ProviderだけのPASS、返却Rows一致のみから実読取範囲/権限を推論しない |
| K5 Release/Consumption eligibility | C0契約資産とC1実装・必要memberを別に配布・保全。対象Consumerが実Load/Invokeした証拠にのみ限定した資格を付与 | Builder成功・Artifactの存在・署名から実外部Host適合やOwner受理/Grantへ昇格させない |

**独立する別Consumer試験**：[#2390](https://github.com/hat47x/tei-platform/issues/2390)は既存の読取り専用見積もり計算 reference.order.quote-lines/v1 と23独立Vectorによる外部Consumerの実利用を優先している。SQLite 方言Contractとは**異なる業務意味/検証基準**なので、同じScenarioとして混合せず、K0〜K5の共通の証拠列だけを流用する。見積もり計算のProcess/Componentの過去CIは実Source/Head/Host単位で参照し、新K試行のPASSへ転記しない。

### 21.2 互換性・撤回・説明の取り扱い

- **意味の変更**：入力型、NULL、桁・単位、失敗/unknown、Effect/Read Scopeに変更があれば、Semantic Contract互換性を検討する。同じ文字列Versionや同じメソッド名でも無条件互換とはしない。互換不明なConsumerは選択を保留し、勝手な代替Providerを利用しない（#951）。
- **実装の更新**：Contract意味が同じでもImplementation実byte・Build provenance・対象SQLite版・Toolchain/Targetが変われば、Conformance/Evidenceを対象版に結び直す。ただし未選択Pluginの変更によって無関係Buildを失効させない（#1251/#2600）。
- **配布終了／失効**：Catalogから消えたことと、過去に配布したArtifact/Receiptの履歴破棄を混同しない。新規選択可否・既存稼働物の現在Grant・旧Sourceの説明資格はそれぞれ独立に判定する。確定したDB EffectのRollbackをPlugin削除で自動達成したとは報告しない。
- **新しいStatus enumの先行採用禁止**：K0〜K5は検証順序の参照名であって新しい公開API/Schemaや一枚の「Pluginが安全」という証明ではない。各段階の不足・unknown・not-run・not-applicableを既存Evidenceへ記録する。

### 21.3 既存横断反例の追加割当

[Draft #2898](https://github.com/hat47x/tei-platform/pull/2898)の重複Role拒否と[Draft #2891](https://github.com/hat47x/tei-platform/pull/2891)のFixture JSON重複member拒否は、**Source fidelity**上の別々の局所修正である。これを全Pluginの新Parserへ移植するのではなく、公開Descriptor/Contract/選択表についても「曖昧性が失われる前に拒否する」という既存#988/#2620/#951/検証基準の負例へ対応付ける。具体的には同じContract IDを複数提供、同じDependency IDの別Revision、Role/Operator Pin欠落、二重指定による後勝ち、不正なValidator入力の黙示補完を拒否・保留する。元Sourceは閲覧・修正可能に保持し、意図した正式な複数Role構文まで禁止するとは決めない。

**実施順序**：P0は#1975の契約だけを既存Data/Sourceでinspectし、Providerゼロの拒否を観測。P1で人/AIの候補と独立検証基準・意図的なmutantを固定し、K2/K3で既存Resolver/Conformanceを照合。P2は隔離した実SQLiteで実Readの対象/完全性/結果を別Evidence化。P3で#2390の別契約・独立Consumerへ展開し、配布方式・Adapter量・Core変更量・修復回数・検証/更新費用を測る。#1975/#2390/#951/#2514/#2620に責務を戻し、第二Registry/Resolver/Plugin Host/AI特権実行/共通Fixture正本は作らない。**K0〜K5の新実行・実SQLite/外部Consumer・公開Contract採用・Owner受理はいずれもNOT RUN**。


<!-- tei-md-plugin-source-migration-20261106 -->
## 22. TEI Source統一（*.tei.md）とPlugin/Buildの互換移行（2026-11-06、Owner方針／実装未着手）

### 22.1 決定と適用境界

2026-11-06のOwner方針：TEI向けSourceは原則 ***.tei.md** に統一し、従来の ***.tei** は新規作成を原則停止して段階的に廃止する。先行[#2621](https://github.com/hat47x/tei-platform/issues/2621)の「.tei.mdは標準化しない」「.mdと.teiの二本立て」を最終決定として維持しない。これは**ファイル名・Authoringの基準変更**であり、現行Markdown Reader/TEI DSL Reader・Role/Operator/Owner受理・Build/実行資格が完成したという宣言ではない。

対象はTEIが所有するModel、Business Function、Rule、Query/Binding、Plugin/Bundle/Contract Descriptor、Test/Scenario等の**TEI Source**。通常の説明用.md、SHO/OKF文書、外部Provider実装の.go/.java等、Native SQL asset、成果物・Lockはそれぞれの役割を維持し、拡張子を無差別に書き換えない。既存のtei.workspace.yamlはWorkspace Manifestとして扱い、legacy tei.workspace.teiは明示的な移行対象とする（Manifest format変更の採否は別判定）。

**.tei.mdという名前はTEI Authoring intentの識別子**。その中の文章・表・式・図・コードフェンスがすべて実行可能ではない。Markdownとして表示・保存可能な原文と、明示的に選択したLanguage/Role/Operatorで意味を解釈する範囲を分離する。OKF v0.2のtype/tags/未知キーもその仕様に沿って保持できるが、tagだけではMeaning/Grantを生成しない。採用するコードフェンス・frontmatter・表記Profileの詳細文法は#2621/#988/#2620で独立Expectedを使って決める。特定の未実装Parserを既定と推測しない。

### 22.2 現行mainのコード事実と移行影響

2026-11-06にmainの internal/workspace/loader.go をread-onlyで照合した。
- loadSemantics：ファイル名末尾 .tei の選別と、選別されたraw sourceを ParseSemanticFile に供給。
- loadPlugins / loadBundles：plugin.tei / bundle.tei の完全一致。loadApplications は application.tei、query/pattern/package/binding 等を *.tei で選択。
- loadDatabases/Environment/Release：database.tei、migration-provider.tei、environment.tei、その他 *.tei 等に固定。
- Workspace Manifest は tei.workspace.yaml 優先、legacy tei.workspace.tei 読取り分岐。
- listPinnedDomainSources は選別されたFile名だけを列挙し、pinned readで実内容を取得する。管理ファイルとして列挙されたことと、Parserが実読取したことは別。

したがって単純renameした *.tei.md は現在のParser対象へ自動的には加わらない。また、単にsuffix判定を *.tei.md に変えて生Markdown全体を従来のTEI文法Parserへ渡しても、混在する文章/フェンス/表の解釈は成立しない可能性がある。**Root/実列挙/実読取/実Parse/Meaning Qualified/Build consumed/Consumer loaded** を別列で検収する（#2823/#2621/#988/#2620/#1251/#2600）。

### 22.3 Plugin設計との接続：論理契約の同一性とSourceの変更を分ける

| 変更 | 保持すべきもの | 別資格で再確認すべきもの |
| --- | --- | --- |
| 同じPlugin/Bundle定義の .tei → .tei.md 移行 | Logical Contract ID・意味版・Owner根拠、旧Source revisionとMigration記録 | 新Sourceの実Loader選別、Language/Role Pin、実Span、Plugin選択・Conformance |
| Markdown本文＋コードフェンスへ変換 | 元raw bytes、説明文/表/コメント、埋め込みDSL位置のsource map、Conversion Diff | 抽出範囲が意図したDefinitionを過不足なく含むか。記述例・引用・Testを勝手にProvider実装扱いしない |
| Third-party Contract-only Bundle配布 | C0契約資産・独立Expected、Publisher/Artifact provenance、対象Target | .tei.md内の契約文面が実装の存在・配布者信頼・Host current Grantを自動付与しない（#1975/#2390） |
| Build input/Dependency file path変更 | Meaning IDとRule/Contractの継承候補、旧Build/Receipt履歴 | Source occurrence/realpath・実content byte・Builder/Toolchain/Plugin Pin・ArtifactSetの再資格。path変更だけで全意味改変とせず、Source ABAで旧Evidenceを復活させない |
| 旧 .tei と新 .tei.md が共存 | 移行中の旧データとRevision履歴を保持 | 二つが同一Canonical IDを定義するときの重複診断/明示Cutover。last-winsや二重Owner SSOTは禁止 |

### 22.4 既存Fixtureを使う移行の否定対照

| 対照 | 独立Expected／失敗時の停止線 | 責務 |
| --- | --- | --- |
| M-01 既存TEI DSL Sourceを改名するだけ | 新名が現Loaderで読まれないことを成功と数えない。必要Source未読込ならvalidate okでも unqualified | #2823/#2621 |
| M-02 旧/新の同一意味ID共存 | 無音上書き・二重Provider選択拒否。元Source/パスは保全 | #2621/#988/#951 |
| M-03 普通の.mdとTEI向け.tei.md | 前者をTEI Sourceとして自動処理しない。後者の説明文・例示用フェンスは自動実行しない | #2621/#2620 |
| M-04 Markdown＋複数TEI構造ブロック／BOM/CRLF/日本語 | 元raw byteと各blockのUTF-8 byte span・順序・意味IDを照合。見出し/改行の整形で無音位置ズレ禁止 | #2621/#988/#2497 |
| M-05 Plugin/Operator Descriptorと依存が未Pin・eager循環 | Sourceはinspect可能でもInterpretation/Activationは保留。Plugin自己記述のSeed/Pinを別資格とする | #988/#2620 |
| M-06 Browser/Studio/SHO上の編集・no-op保存 | Markdown原文の無断整形・変換、Role未解決の意味自動受理を禁止。Apply時Revision不一致は差戻し | #2621/#1124/#2497 |
| M-07 未読込Required Rootと旧パス参照 | 実Read/Dependency/Build Input Closureが不足なら0件ではなくunknown/unqualified | #2823/#2600 |
| M-08 Migration後のBuild/ConsumerとRollback | exact Source/Artifact/loaded bytesを対象Targetで確認。旧成果物保全と業務Effect取消は別 | #1251/#2390/#1174 |

上記M-01〜08は比較用ラベルで、新しい第二の検証基準の正本・実験Runner・固定Product API enumではない。既存#2621 FM/AP、#2823 Source root、#988 Language pin、#2620 SDX、#2514 Independent Expected、#2600 IBを共有する。

### 22.5 段階的移行とOwnerへの受渡し

P0 **read-only棚卸し**：既存Workspace Manifest、domain loader別の全*.tei選別点・生成書き出し・Test Fixture・外部Consumer参照・文書リンクを列挙し、現行負例を固定。原ファイル名・byte・Semantic ID・Source index/Build closureを結ぶ。TEI新Source形式規約の正本は#2621、実Loader対象と未読込診断は#2823、Language/Roleは#988/#2620。

P1 **最小Parser経路の検証**：1つの公開Semantic定義とPlugin Descriptorの二つを、旧.teiと新.tei.mdで比較。プレーンMarkdownのRaw保全、明示構造ブロックの解析・byte span、旧/新独立Expected、必要Source完全性を検証。旧Grammarを無条件で捨てず、明示した互換期間中だけread-onlyまたは限定実行経路を残す。単純renameだけで適合したとしない。

P2 **Workspace/Build/外部Target再資格**：現行#1251 PB・#2600 IBで実Read→Meaning→Build Source/Artifact/Readerを一改訂へ束縛。旧/新重複・リンク切れ・未使用Plugin非混入、Target固有依存充足を検査。適格なConsumerに限り移行したSourceを適用し、元Source/旧Artifactを保全。

P3 **段階廃止**：既存*.teiへの新規追加を停止する運用、旧ファイルの参照・Legacy Reader撤去のGateを、P1/P2で成立したConsumerごとに判断。移行前の一斉rename/削除や、旧Fallbackで新Required Sourceの欠落を黙殺することは禁止。Ownerの形式方針は決定済みだが、文法/移行・互換期間/Plugin ABI/本番資格は実証・判断待ち。

**状態**：2026-11-06の新Source命名方針を設計へ反映する追補。既存mainの*.tei読取挙動のread-only静的照合のみ。*.tei.mdのProduct Parser/Writer/CLI/Go Target・外部Consumer E2E・OS/DB・Owner業務意味受理は NOT RUN / NOT PROVEN。新Source Reader/Registry/Host/Policy Engine/第二Acceptanceを作らない。
