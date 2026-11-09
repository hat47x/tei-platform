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

<!-- tei-d1-two-offline-staging-seams-20261107 -->
**D1で共通計算部を抽出する前の、実Sourceに基づく互換性関門（未実装）**：`plugins/reference-order-quote/main.go` の現`package main`には、quote入力・結果の型（88〜103行）、厳密JSONのメンバー検証（`strictQuoteFields`、127〜166行）、`decodeQuoteRequest`（168〜200行）、`calculate`（202〜224行）が同居する。一方、carrierの`fingerprint`/`serve`/`main`等は別責務。**金額計算関数だけを抽出してJSON厳密拒否を別実装にする案は、現23件の重複Key/大小文字差の判定が変わる可能性があり採用しない**。ここはコード位置に基づく分離候補であって、Go package変更済みではない。

同じGo Providerの自己完結Buildには**異なる独立Stagingが2経路**ある。① `tools/reference-capability-transport/order_quote_process_test.go` は直下`.go`を一時 `module example.invalid/reference-order-quote`へコピーし、offline `go test`→build→実Processを検査。② `tools/reference-capability-transport/quote_go_js_vectors_test.go` は同じ`.go`を `module example.invalid/quote-go-js-comparison`へコピーし、`GOWORK=off`/`GOPROXY=off`でGo build後、Node実Processと23件を比較する。後者はWindows/Node不在で明示SKIPし得る。両テストともProvider Source直下の**サブディレクトリ・Symlink・Go以外（README除く）を現状は拒否**する。`plugins/reference-order-quote/quotelib/`等を安易に置いてもStagingが自動追跡するわけではなく、`internal`に移せば新importが各一時module内で未解決になり得る。**前者だけを直して後者のImport閉包を壊す、または後者のSKIP緑をGo/JS同等実証へ昇格することを禁止**。

| 実装候補と扱い | 使える根拠・変更範囲 | 停止条件／受入 |
| --- | --- | --- |
| **E0 現`package main`の直接呼出し維持** | #3021のD0直接Go計算とD2 Process基線を保全。別Static Consumerがないならここで完了可 | `package main`の関数を別packageからそのままimport可能とはみなさない。D1実Target資格ではない |
| **E1 単一のインポート可能な計算packageへ限定抽出（最初のD1実Consumerが必要な場合の候補）** | Quote型・厳密JSON検証・Decode・Calculateを一組のSourceとして再利用し、Process側は薄いAdapterのまま。別の業務Rule正本は作らず、D0/D1で同じ23期待値を再使用 | Go Process用の**二つの一時moduleとも**ローカルに必要package/sourceを閉包し、外部Repo/Network/Workspaceへ隠れ依存しない。D0テストのroot用期待値欠落はFAILのまま、Process stagingの必要SKIPをD0 PASSに数えない。既存Go/JS/D2両CIが同一変更HEADで実PASSするまでExtraction成立と記録しない |
| **E2 Sourceコピー/生成の合成Static Consumer** | `main.go`からソースを再利用する一時Targetだけなら現状を動かせる可能性 | それがProject/`#951`の実Selected Provider→実Build→実Consumer Callに接続されない限り、Test Harnessの限定Proofであって**製品D1実装ではない** |

**最短の実装順**：(1) #1016の実ConsumerがD1を必要とする理由と厳格単一Go Binary制約を固定 → (2) E0/E1/E2のSource/Build移動コストを小さく比較し、E1が必要ならProvider根幹とProcess carrierの境界をSourceごと明示 → (3) 両Staging・root D0・23件の同一Expected・`go list -deps`/実Go Target Build/Invokeを同じHEADで検証 → (4) B選択・B実呼出しの正例とB選択・A実呼出しの負例、未選択Plugin混入・依存欠落を別判定。新共通ABI/第二Rule/新Hostを導入せず、**Consumer要求が無ければE0で停止できる**。採否・試験PASSは未確認のまま#1251/#951/#2390へ引き渡す。
<!-- tei-d1-go-module-preflight-and-native-builtin-claim-ceiling-20261107 -->
**現行mainの既存実装を用いるD1の検証順（新Plugin基盤を追加しない）**：main統合済み[Go閉包事前診断 #3091](https://github.com/hat47x/tei-platform/pull/3091)の`tools/internal/gobuilder.ResolveModuleClosurePreflight`は`GOWORK=off`/`GOPROXY=off`/隔離`GOMODCACHE`と`go list -deps -json -e -mod=readonly`で、`supported`／`external-unresolved`／`missing-module`／`unknown`を分類する。ただし**分類成功は実Build/Link/Target Source closure成功ではない**。固定main `4fdad69d...`のWindows/Go1.27観測では`./cmd/tei`に外部3依存の供給不足があり`missing-module`、自己完結した参照Toolは`supported`だった。これはD1 Quote Target固有の観測ではなく、現在のGo Toolchain/Target/外部モジュール供給に一般化しない。Module内容のオフライン供給と`go.sum`/byte整合（#2831 SB-0b）、その後の`ResolveSourceClosure`/実Buildは別段階である。

**空観測の追加負例**：`classifyModuleRoots`は、選択Module rootが0件かつ未解決packageも0件だと` supported`を返し得る。Go listが空出力、あるいは標準ライブラリしか返さず対象Moduleの選択が観測されなかった場合、**「0件の依存が問題ない」ことと「実際に対象を解析した」ことを同一視しない**。この論理上の空虚成功に対し、現行main上のコードを最小修正した[Draft PR #3130](https://github.com/hat47x/tei-platform/pull/3130)で`unknown`へ戻し、空Stream／stdlib-onlyの正負ではない2つの拒否負例を追加した。#3130は本記録時点Draft、**新HEADのGo試験・CIは確認前**。既存#3091の限定観測も、新PRの成功へ流用しない。

**既存Native Plugin参照との差分**：現行`internal/workspace/plugin_native_implementation_reference_test.go`の`TestTEINativePluginImplementationReferenceRunsThroughExistingBuiltinABI`は、旧`plugin.tei`／`implementation.tei`をLoaderで読み、`BuildPlan`＋`implementation.Materialize`＋Build/Engine Lock＋Builtin登録＋`DeploymentExecutor.Execute`まで検証する実Goテストが存在する。ただし`Recipe.Kind=tei-native-static-response`、`Target=interpreted-builtin/host`であり、`nativeReferenceBuilderDigest`／`nativeReferenceEngineDigest`は固定したテスト用値。**実Goコンパイラが選択Provider Bを静的リンクした、実Binary bytesがEngine fingerprintに束縛された、同じQuoteの23件を呼んだ、とは証明していない**。現Native ReferenceはD1に必要な**論理的なSource→Lock→Invoker接続の既存Fit**として再利用し、物理Go Build/Link/Callは#1251/#951の別Witnessとする。わざわざ第二のRuntime ABIやProvider Registryを新設しない。

**次の実装ゲート**：D1実Consumerが必要だと#1016で確定した場合のみ、(A)対象Module/Go版/隔離状態の事前分類（`missing`/`unknown`は調査・停止、`supported`は必要条件）→(B)Quote純粋計算Sourceと二つのoffline stagingの実消費/Module供給（E1なら新packageの実path・`go.mod`/`go.sum`）→(C)選択Bと実Static Link/実Callの同一Target証拠＋B呼出し正常対照／誤接続A負例→(D)23独立期待値・未選択混入・費用の比較、の順。事前分類CLI終了0、Native Builtin Reference PASS、D0/D2 CI PASSの**合算でD1のPASSを作らない**。D1実Targetと#3130の最新結果は未確認。
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

P0 **read-only棚卸し**：既存Workspace Manifest、domain loader別の全*.tei選別点・生成書き出し・Test Fixture・外部Consumer参照・文書リンクを列挙し、現行負例を固定。原ファイル名・byte・Semantic ID・Source index/Build closureを結ぶ。移行全体のOwner計画は[Draft #2667](https://github.com/hat47x/tei-platform/pull/2667)、Markdown Authoringの表層比較は[Draft #3030](https://github.com/hat47x/tei-platform/pull/3030)/#2621、実Loader未読込診断は#2823/[Draft #3031](https://github.com/hat47x/tei-platform/pull/3031)、Language/Roleは#988/#2620。別個の移行Gateをこの文書で設けない。

P1 **最小Parser経路の検証**：1つの公開Semantic定義とPlugin Descriptorの二つを、旧.teiと新.tei.mdで比較。プレーンMarkdownのRaw保全、明示構造ブロックの解析・byte span、旧/新独立Expected、必要Source完全性を検証。旧Grammarを無条件で捨てず、明示した互換期間中だけread-onlyまたは限定実行経路を残す。単純renameだけで適合したとしない。

P2 **Workspace/Build/外部Target再資格**：現行#1251 PB・#2600 IBで実Read→Meaning→Build Source/Artifact/Readerを一改訂へ束縛。旧/新重複・リンク切れ・未使用Plugin非混入、Target固有依存充足を検査。適格なConsumerに限り移行したSourceを適用し、元Source/旧Artifactを保全。

P3 **段階廃止**：既存*.teiへの新規追加を停止する運用、旧ファイルの参照・Legacy Reader撤去のGateを、P1/P2で成立したConsumerごとに判断。移行前の一斉rename/削除や、旧Fallbackで新Required Sourceの欠落を黙殺することは禁止。Ownerの形式方針は決定済みだが、文法/移行・互換期間/Plugin ABI/本番資格は実証・判断待ち。

<!-- tei-plugin-source-root-diagnostic-qualification-20261107 -->
**Source Root検査のmain前進と資格の上限**：[merged #3073](https://github.com/hat47x/tei-platform/pull/3073)は未知・存在しないManifest source rootと、現Loaderが読まず取り残す近似名のfileを診断する。空Sourceを見落とした成功と、明示された空Workspaceを区別しやすくなる。[merged #3069](https://github.com/hat47x/tei-platform/pull/3069)は標準定義を除いたdeclarationCountsを返すが、0件自体では実Required Source閉包・Import/Plugin/Provider完全性を証明しない。未知RootをWarningのまま許容する方針の是非や、workspace-filesのreadByLoader表示は依然Owner判断。これらのmain統合済み診断と、[Draft #3031](https://github.com/hat47x/tei-platform/pull/3031)の*.tei.md未Binding専用Guardを混同しない。未選択Pluginへの誤導入・必要Pluginの黙示欠落を防ぐ受入には、Source/Loader実読取とCatalog/Resolved選択・Target Build/Consumerの実依存を一版で照合する必要がある（#2823/#951/#1251）。

**更新された実装Evidence（2026-11-07）**：[Draft #3031](https://github.com/hat47x/tei-platform/pull/3031)の固定HEAD `63953ccb`では、未対応の*.tei.mdを黙って未読込扱いせずParser未Binding診断で停止する限定Workspace Go試験がCI成功。これは**新形式を安全に拒否できる候補**の実証であり、正式Markdown Reader・Parsed/Qualified/Build-consumed・旧新意味互換の実証ではない。PRはDraft・未merge。移行の受入条件は#2667/#3030へ委譲する。

**状態**：2026-11-06の新Source命名方針を設計へ反映する追補。現mainの旧*.tei読取の静的照合と、別Draft #3031の限定拒否試験のみ確認。*.tei.mdのProduct Parser/Writer/Go Target・外部Consumer E2E・OS/DB・Owner業務意味受理は NOT RUN / NOT PROVEN。新Source Reader/Registry/Host/Policy Engine/第二Acceptanceを作らない。

<!-- tei-tool-plugin-principal-separation-20261107 -->
## 23. Tool／Plugin／実行主体を同一視しない：外部ToolのCapability Grant停止を起点とした受入設計（2026-11-07、方式未採用）

### 23.1 新しい実観測と、Plugin設計に返す問い

[merged #3121](https://github.com/hat47x/tei-platform/pull/3121) は、`reference/workspace`の一時複製へ参照ツール能力 `tei.frozen_world.lineage` の書込み付与を加えて `tei validate` を**実行**し、`PERMISSION_GRANT_CAPABILITY_NOT_RESOLVED` を観測した。現 `internal/validation/validation.go` のEnvironment Grant検証は `resolvedPlugins.Capability(capKind, g.CapabilityID)` でProviderを要求するが、この参照ToolはPlugin Catalogに登録されていない。元Workspaceは変更せず、この経路は**未接続／未利用可能**。故障の観測は事実だが、ToolをPluginとして登録すべきかという設計判断は**未決定**。

必要な区別：**(1)論理Capability Contract＝何ができるか、(2)実装/配布＝Plugin Bundle・独立CLI Tool・Builder・Host内蔵実装等のどれが実現するか、(3)呼出主体・実行Authority＝誰が現在のTenant/Resource/Effectで実行できるか**。Bundle内包、Executable Toolとして存在、契約定義、Plugin Catalogで発見できることは互いの自動証明ではない。Developer/AI向けToolを一律にRuntime Pluginへ変換することも、ToolをPlugin外として権限検証を外すことも要求しない。

### 23.2 最初に比較するのは二つの有限な適用案（採否はOwner留保）

| 観点 | 案A：実Toolを既存Plugin契約として提供 | 案B：Toolを別の実行主体として既存権限契約へ明示接続 |
| --- | --- | --- |
| 導入条件 | Contract/Provider identity、Source・実Artifact・Build/Lock、Catalog解決を満たす。Pluginとして要求・配布する実需要を示す | Toolを名指す主体・実装/実Targetを明示し、既存のHost Grant/Consumer境界に委譲する経路が必要。Toolであるだけで許可しない |
| 現状Fit | 現 `resolvedPlugins.Capability` を流用しやすいが、Runtime Plugin依存・配布コストが増える | 権限契約の対象拡張が必要かもしれない。現行ValidatorはToolを未解決として拒否するため**実装済みと見なさない** |
| 必須負例 | Tool未選択/未Load、同名Contract権限横取り、署名・Bundle版だけでGrant発行、未使用PluginのGo init混入 | Tool名を文字列で宣言しただけのPermission bypass、他TenantへのGrant漏れ、発行元が名乗っただけの真正性偽装 |
| 方式の採否 | 外部配布・第三者Conformance・Target選択の利益を実証した場合に検討。全ToolのPlugin化を強制しない | 実Hostが既にTool Principal/Delegationを提供する場合を比較。第二のAuthority Registryや万能Tool Grant正本は先行導入しない |

選定の前提は `Source/Contract/selected implementation/actual binary or process/actual invocation/current Host or Provider Grant` の同一対象・版・Scope束縛。案A/Bとも契約の発行者/namespace Authority（#951）と、実Grantのissuer/権限（#1174および既存Access Owner）を混同しない。文字列`issuedBy`/`ResolvedBy`やArtifact digestは、真正な発行者・OS隔離・現在許可の代替ではない（merged #3035/#3075/#3072）。

### 23.3 HookとCommandで実装済みになった境界を再利用し、未配線を保留する

[merged #3077](https://github.com/hat47x/tei-platform/pull/3077)の `hook.Declared` は、宣言外Capability呼出しを`HOOK_UNDECLARED_CALL`で拒否し、宣言内だが実装不在の`HOOK_IMPLEMENTATION_UNAVAILABLE`と区別する。[merged #3108](https://github.com/hat47x/tei-platform/pull/3108)の `hook.DeclaredCommand` は、Patternが宣言した能力とCommand境界へ委譲する限定Adapterを追加した。ただし**hook.Runnerからの実配線・実Host Grant・実Providerは未実証**。したがって「Hook Usesに宣言済み」「Patternに存在」「Command Boundで受け取った」「現在Grantがある」「実実装が呼ばれた」を一つのProvider Enabledへ畳まない。

HookもToolも「声明されたUses/Target Capabilityが存在する」ことから、それを発行した主体の許可を推論しない。最初は同一Hook・同一Argumentの正負一因子比較で、未宣言→拒否、宣言済みだがImplementation不在→unavailable、宣言済み・Implementationあり・Current Grant欠落→Command/Host拒否、完全Binding・必要Grantあり→該当実Targetへの限定到達を別々に観測する。A/B方式で実Effectが発生する場合だけOperation Identity/Receiptを追加する（#1016/#951/#1174）。

### 23.4 Persistenceの観測対象を四状態と実Read範囲で表す（2026-11-07更新）

[merged #3124](https://github.com/hat47x/tei-platform/pull/3124)／[merged #3127](https://github.com/hat47x/tei-platform/pull/3127)の当初の`observed`だけではReceipt Lookupと資源現在値が混同された。後続[merged #3155](https://github.com/hat47x/tei-platform/pull/3155)で、`not-applicable / not-observed / receipt-observed / resource-observed`を区別し、`persisted:true`は資源の独立ReadでCommitしたpayloadとの一致を確認した場合に限定する。本文以前の「実PostgreSQLのLookup／Resource Readは未実施」という記述は**旧HEAD時点の履歴**であり、現状の未着手事項としない。

実装された`effectcommit.ResourceReader`／別poolの`postgres.Provider.ReadResource`と、[実PGのpostgres-conformance #42328707393](https://github.com/hat47x/tei-platform/actions/runs/42328707393)の成功を確認した。ただしReceiptはOperationの記録、ResourceReadは指定されたResourceの観測時点での値である。ResourceのVersionと当該Operationとの因果、Provider取得母集団・全対象集合、別Tenant/Replica、現在Read Grant、後続更新とConsumer受領は別Evidenceとする（#1174/#2598/#1163/#2514）。

[merged #3143](https://github.com/hat47x/tei-platform/pull/3143)の`authorizationMode=reference-allow-all / reference-allow-list / external`はAuthorizerの**種別**であって実Issuer/Policy/Grantの真正性ではない。Reference全許可で成功したCommandをProduction Hostの許可済みOperationへ昇格しない。Compositionが`ResourceRead`を有効にしたことも、自動的には全Providerに適用されない。
### 23.5 重複しない最小負例・実施順

| 対照 | 独立検証基準 | 担当と証拠の上限 |
| --- | --- | --- |
| ToolをEnvironment Grantで指すがCatalogで未解決 | `PERMISSION_GRANT_CAPABILITY_NOT_RESOLVED`で停止。無音で別Providerへ切替・単にTool名で許可しない | #3121現物の既観測。今後の設計案A/B共通負例（#951/#1016） |
| 同じCapabilityを別Tenant/主体が持つ | 契約一致だけでGrantを横流ししない。current issuer/subject/Resource/Scopeを実比較 | #2903/#1174。実同Host多Tenant E2EはNOT RUN |
| Hookが宣言外Call／実装欠落／Grant欠落／許可済みCall | 「拒否理由」「Provider未導入」「Authority拒否」「実Call」の四ケースを別結果とする | #3077/#3108の局所Go testは限定Fit、Hook Runner/本番HostはNOT RUN |
| Tool/PluginのSourceは管理済みでもLoader未読込 | `workspace-files.readByLoader`（merged #3078）とSyntax/Meaning/Build memberを分ける。未知Source root/モデル0件warningを全件適合としない | #2823/#3031/#3081。新*.tei.md実ReaderはNOT RUN |
| 純粋D1 Provider B選択、実Call A | 同じ23独立期待値の一致ではPASSとせず、Bの実Link/InvokeをTarget相対に照合 | #2885 §18.13/#1251。D1実静的GoはNOT RUN |
| Receipt Lookupが成功、ResourceReadは未実行 | `receipt-observed`と表示。Resourceの現在値を確認済みにしない | #3131/#3155の別経路で実PG試験あり。当前Attemptで未読取なら未確認 |
| ResourceReadがpayload一致を返した | 当該Resource・観測時点のみ`resource-observed`。全件完全・現在Grant・Owner受理とは別 | #3155実PG限定成功／#1174／#2514 |

**着手順**：(P0) #3121の実拒否を再現可能な既存Evidenceとして固定し、案A/Bの必要Actor/Contract/Grant/ArtifactのFit/Gapを比較。(P1) #3077/#3108のHook→Command局所負例を既存実装へ引継ぎ、必要な実行主体/Host Grantの観測点を特定。(P2) 実Consumerの要件が明らかな方だけ実Targetに接続し、ReceiptとResource Readは別の観測票にする。これらと独立して**D1静的Provider実Link/Invokeの最短試験を優先**できる（新*.tei.md Reader、DB/Host Effect、Tool権限方式の決定をHard dependencyにしない）。

**資格**：現物のTool Grant拒否、mainのHook/Command/Persistence/Receipt・Source列挙の限定Go実装を確認して設計へ接合した。本節はDocs-onlyで、案A/Bの正式採否、Tool権限新実装、Hook Runner結合、D1静的Build、全業務・全Tenant・ConsumerでのSource→Grant→Resource独立Read、Owner意味受理は**NOT RUN / NOT ACCEPTED**。新Registry/独立Grant正本/万能Plugin Host/Tool API Schemaを導入しない。
<!-- tei-plugin-postmerge-source-closure-harness-function-20261107 -->
### 23.6 外部Moduleの実供給→Build→再現性→Source閉包観測→資格を非推論でつなぐ

[merged #3142](https://github.com/hat47x/tei-platform/pull/3142)の`cmd/tei`実選択12 Moduleの`go.sum` h1照合、[merged #3156](https://github.com/hat47x/tei-platform/pull/3156)の隔離した供給だけによる`cmd/tei`実Build（Source closureは`rejected`）、[merged #3193](https://github.com/hat47x/tei-platform/pull/3193)の同一Linux/Go条件で独立2試行・同一Artifact digest、を**それぞれ成立した限定Claim**として扱う。これらがmainに統合済みであることと、TEI標準Build GraphからのSelf-build G4資格は別。旧Draft #2885の『#3156未merge・再現性未測定』という履歴記述を現状として再使用しない。

[merged #3217](https://github.com/hat47x/tei-platform/pull/3217)は選択Sourceを`repositoryMembers`／`externalMembers`／`externalModules`へ列挙し、資格ある閉包のdigestとは異なる領域の`observedDigest`を出す**実装済みの観測経路**。限定実測は`cmd/tei`のRepository 318件、外部12 Module/325 Members。`qualified`は意図的に常にfalseで、実Buildが正確に同じ全メンバーを使用した証明ではない。**mainに観測機構があることは、外部Module Source閉包の資格が成立したことではない。**

| 接続点 | 期待する独立対照 | 受理責務 |
| --- | --- | --- |
| Module supply→選択member | 実選択・`go.sum`・`.mod/.zip`が一致し、未選択5 Module不足は不合格にしない。改ざん・不一致は拒否 | #2831/#1090、供給者Trustは別 |
| Build→Artifact | 同一対象・試行別隔離cacheでBuildを実施し、Artifact digestとToolchain/OSを記録。二回一致してもTarget横断を保証しない | #2831 SB-B、#2600 |
| Source Observation→Qualified Closure | #3217の外部Member観測が成功しても`ResolveSourceClosure`の`rejected`を保持。採用するなら#1090のOwner判断と独立検証が必要 | #2831/#1090、G4=UNQUALIFIED |
| D1 Provider選択→実Link→実Call | Provider A/Bが同じ23 Expectedを返す状態でBを選択し、誤ってAをLink/Invokeした場合は不合格。選択B・実B呼出し・未選択実装非混入・D2非退行の正負対照を要求 | #1251/#951/#2514、未実施 |

**後続の方式比較（採否未決）**：[merged #3246](https://github.com/hat47x/tei-platform/pull/3246)の判断資料は、A＝現行の外部Module拒否を維持、B＝検証済み実選択供給のみSource閉包へ採用、C＝既定拒否のまま明示opt-inで段階的に認める、の三案を示し**Cを推奨するが正式採用していない**。採用前には、(1) Goの実Build入力と観測Member集合の一致、(2) opt-in/既定のClosure意味の識別、(3) `go.sum`自体の受理と供給元Trust、(4) Windows/Linuxの相対Member identity、(5) #1090/#2831の既存BuildSource契約への接続を個別に判定する。Aの継続は失敗ではなく安全な設計判断であり、B/Cが未実装のまま `qualified:true` へ変更しない。

**実施優先**：#1251のD1静的Link/Invokeは既存Go Sourceと独立Expectedで先行でき、Go全体のSelf-build G4や`.tei.md` ReaderをHard依存にしない。外部Moduleを含むTargetだけは、BuildができてもSource closure拒否を隠さない。新Module Provider、第二Resolver、無条件のTrust昇格は行わない。

### 23.7 外部Harness／Toolの読取・実効隔離・副次送出を分ける

[merged #3173](https://github.com/hat47x/tei-platform/pull/3173)のDSH 0.2.2-rc.1 `read-only`・ローカルMock検証では、許可投影だけでなく合成`raw/secret.txt`、`.env`、cwd外canaryも読み取られ、モデル入力とSession Logへ到達した。[merged #3191](https://github.com/hat47x/tei-platform/pull/3191)では`AGENTS.md`/`CLAUDE.md`とSkill descriptionが明示Tool Read前に投入された。[merged #3216](https://github.com/hat47x/tei-platform/pull/3216)では別cwd/Lane間の相手canaryを読めた。いずれも合成データのローカルMock観測であって、実Providerの送出・全Tool経路や実OS分離を網羅しない（#2515/#2509／TAI担当）。

Pluginに内包するAgent Skill/Toolを選択する場合、`Accessible`、実`Read`、モデル入力`Delivered`、Session Log/Telemetry等の`Egress`、実OSの拒否/隔離を**別の事実**として保持する。`read-only`は業務DataのMutation不許可を意味し得ても、OS上のRead拒否・ローカルLog/Cache非生成・送出なし・worktree間秘密分離の証明にはならない。[merged #3152](https://github.com/hat47x/tei-platform/pull/3152)の`os.Root`によるPath containmentもProcess/Networkの隔離とは別。

[Open #3162](https://github.com/hat47x/tei-platform/pull/3162)のTEI CLIドッグフーディングは、`view`がread-onlyと広告しても、`-execute`追加オプションでoperation log/lifecycle stateがローカル生成されることを実測した。AI Harnessへ与えるread-only Capabilityは、CLI Command名だけでなく、全argv・実File effect・出力先とWorkspace/Artifact同一性で限定する（#3149/#2605）。本節ではEgress Policy/権限エンジンの新正本は作らない。

### 23.8 標準表現型とPlugin実装・Function Contextの非同一性

[Draft #3222](https://github.com/hat47x/tei-platform/pull/3222)および[Draft #3225](https://github.com/hat47x/tei-platform/pull/3225)の横断検討に従い、`Standard Data Shape`、`Presentation Kind`、**Function内のRepresentation Instance/Slot**、実Interaction/Binding/Host Effectを分離する。標準表現型やRendererをPlugin Bundleで提供しても、そのKindをCatalogで発見しただけでは、特定Functionでの配置・Data mapping・Operation Offer・保存Intent・Host Grantは確定しない。Headless Functionが0 Slotを持てることと、表示が不要な業務意味が無いことも別。

複数の表現型を同期する場合、表示のSelection/epochとDomain Subject/Relation occurrence、Data取得集合、Update Intentを別に検証する。未取得/部分取得/秘匿のDataを空集合で補完したり、UI上の選択変更をDomain Source Mutationとして自動保存したりしない。Mappingの正式な受入は#2507/#2506/#2514、媒体/時間軸・複数画面の詳細は#3192/#3222/#3225へ委譲し、このPlugin設計内に第二のUI Binding SchemaやOwner Meaningの正本を作らない。

**三要素CRUD・Cross-referenceへの接続**：[Project Plugin Compositionの型付きCross-reference追補](PROJECT_PLUGIN_COMPOSITION_DESIGN.md)では、Plugin選択・実呼出しのRelationと、三要素分析法Data/Function/Businessのdirect CRUD／derived effective effect／observed operationを、同じ標準Relation occurrenceからGraph/Table/Matrixへ投影する最小検収を定義する。図上の一本のEdgeから原Sourceの複合変更・実Grant・CRUDを推論せず、表示Occurrence→qualified Relationの逆参照、public Slice/Source/時点・変換Lossを保持する。Method固有の意味は三要素設計と#1124/#1126、対応写像は#2497/#2507へ委譲。新Graph Core/Matrix SchemaやHostは追加しない。

**Plugin構成の標準表現への具体化**：[Project Plugin Compositionの2026-11-07追補](PROJECT_PLUGIN_COMPOSITION_DESIGN.md)では、Plugin/Contract/Provider/Bundle/Selection/実Build/Invocationを既存Record/Relation/Collection/Graphから投影する。Graph/Table/Tree/Matrix/Timelineは標準表現型として再利用し、Graph layoutやDragを契約/権限の更新へ自動昇格させない。接続の多項関係は既存Record＋Relation集合で表現し、新Plugin Graph Core型・専用Rendererは作らない（#1016/#2497/#2507）。

**状態と未決**：§23.6〜23.8は既存の限定実証をmainの計画へ接合した設計追補。D1 Provider Bの実静的Link/Invoke、Source閉包の外部Module受入、OS/Network全経路の隔離、標準表現型の実Target・UI/DB縦断、Owner意味受理は未実施／未採用。#3217はmainへ統合済み（ただし`qualified:false`）。#3222/#3225/#3162のOpen Draftは未統合として扱う。

<!-- tei-plugin-release-consumer-witness-handoff-20261107 -->
### 23.9 第三者PluginのRelease証拠から実Consumerへ：公開・独立検証・利用資格の橋渡し（2026-11-07）

第三者PluginのRegistry正本は[#2589](https://github.com/hat47x/tei-platform/issues/2589)／[公開設計](PLUGIN_REGISTRY_PUBLICATION_VERIFICATION_DESIGN_2026-11.md)に置く。ここでは§21のContract-first／§23.6のSupply・Build・閉包と接続する**最短Consumer検収**だけを定義する。新しいRegistry、Release schema、Attestation Profile、二つ目のResolver/Lock/Hostを新設しない。

[Draft #3244](https://github.com/hat47x/tei-platform/pull/3244) exact head `b5d389cfa7e6b98a589cba54fff4d837848e1a80` の[専用CI #42347303135](https://github.com/hat47x/tei-platform/actions/runs/42347303135)で、Go/Nodeの独立Readerが**テスト専用Fixture**のPublisher Ed25519署名・Release SHA-256、二つの必須Artifact Role/Byte、Verifier声明のRelease Subject/Profile/Targetを照合し、正常例と14種類の否定例を実行した。CIの`release-evidence-binding` Jobはsuccess。Fixture側の公開鍵を信頼起点に使わずReaderに固定した点は適切だが、これは実Publisherの権限・現在のIssuer Trust/失効、実Conformance実行、外部配布・Hostでの利用資格を証明しない。`fresh`/Owner epochは模擬Policy入力であり、その現在性を問合せていない。

| 接続の相 | 成立を示す最小の独立観測 | 絶対に推論しないもの／既存担当 |
| --- | --- | --- |
| **R0 Contract / Provider Candidate** | Contract/Profile/Targetと実装候補の明示参照、独立期待値と拒否例。未Providerは未提供として扱う | Catalog発見＝Provider存在、AI生成候補＝Owner意味受理（#951/#1975） |
| **R1 Release Frame検証** | Release digest・必要Artifact集合・Role/Bytes、Publisher署名とVerifier声明の対象を独立に照合（#3244 fixture-only成功） | テスト鍵のSignature PASS＝現在の実Publisher権限、実Conformance結果（#2589/#1104） |
| **R2 実Publication確定** | 同じPlugin identity/versionに対するVersion Binding確定、署名時点のAuthority/Release全MemberをPublication Attemptに固定 | Registry掲載＝実GitHub Release byteの将来受領、同Version別Byteの上書き許容（#2589/#1105） |
| **R3 Consumer Readback** | 別Attempt/別Readerが公開Locatorから同一Releaseと各必須Artifactの実Byteを取得・再hash。R1 manifest＋R2 memberの混合を拒否 | 署名済みManifestの存在＝実ConsumerにByteが届いたこと（#2604/#2390） |
| **R4 Source / Build / Selected Provider** | Consumer Targetに必要な選択Provider・Source/Build closure・実Artifact・Version/Target Profileを一致させる。Go外部Moduleは§23.6の限定資格を使用 | Release digestと`go.sum`一致＝Build Provenance、Module Source資格、D1 B実Link/Invoke（#2831/#1251） |
| **R5 Loaded / Granted / Invoked** | 実HostでLoadしたbyte identity、対象Tenant/Resource/Current Grant、Consumer実Callと結果・Effectの同一Attempt証拠 | Signed release＝実行許可、本番Host強制、Owner Acceptance、Receipt/Resourceの現在性（#1370/#1174/#2390） |

**最小の反証**：①署名済みだが必須Memberが欠落／別版のMemberを混ぜた、②同Version別Byteが別Locatorへ載った、③Publisher/Verifierの権限が既に失効したが署名は検証できる、④Profile/Targetが違う声明を流用、⑤R3 Readback後にR5 Load Byteが変わった（TOCTOU）、⑥R4でProvider Bを選んだのにAを実Call、⑦Releaseだけ有効でConsumerは未取得／権限なし、を**異なる不合格・未取得**として区別する。既存#2589 RG、#1251 PB、#2514独立検証基準に引継ぎ、第二のFixture正本は作らない。

**適用の重さ**：R1〜R3は第三者公開PluginのTrust/Distributionが必要なConsumerで優先する。ローカル純粋計算D1の最初のA/B実験に、実Registry/Publisher署名/本番HostをHard dependencyとして課さない。逆に外部Releaseから実装を採用する場合、Go二ReaderのテストFixture成功だけでR3〜R5を省略しない。

**状態**：#3244はDraft・未マージ、専用CIの成功は固定Headに限る。実GitHub Release Readback、現在Trust/失効照会、実Conformance実行、二つの独立Host、D1選択Bの実Static Link/Invoke、Owner Meaning受理はNOT RUN/NOT PROVEN。これは既存Evidenceを接合する**設計追補**で、業務Grantや第三者Plugin方式の新規採用ではない。

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

<!-- tei-function-qualification-20261108-consolidated -->
## Functionの設計充足・実消費・外部利用資格を一つの照合で説明する（2026-11-08、横断実装計画）

**状態：採用済み方針への実装・検証順序の提案。新しい製品API・Go実装・Owner意味受理の実証ではない。** 本節は旧[Draft #3283](https://github.com/hat47x/tei-platform/pull/3283)の累積・重複する設計追記を、現行mainの採用[ADR-0048](../adr/0048-shared-application-api-and-transport-adapters.md)と実CIを基準に整理し直したもの。§14〜15のFunction/Build Unit/Job方針、単一Go Binary既定、必要時のみ2〜3 Unit比較、外部Host、Owner別責務は変更しない。

### 1. Functionは「対象への見方」であり、新しい業務意味の正本型を要求しない

X-TEAのFunction概念を、TEIの利用者が設計・構成・配備・診断する**一つの機能の単位**として扱う。ただし[現行アーキテクチャ](../../docs/architecture.md)どおり、旧製品にFunction Objectがあったことを理由にTEI Canonicalへ万能なFunction kind/Registryを追加しない。受理済みApplication/Pattern/Query/Command/Business Procedure/Rule/Data・依存関係と選択したEntryから、**根拠付きのFunction view**を導出する。正式な意味受理はSource Ownerにある。Function一覧[#2482](https://github.com/hat47x/tei-platform/issues/2482)・三視点図/CRUD・Studio・SHOはこのviewの別Projectionであり、「一覧にない」ことからFunction不存在を推論しない。

実行入口はCLI、Web/HTTP、Desktop/IPC、OS Job/EKI等のAdapterごとに実現するが、**意味・型付き入出力・必要な権限・結果・エラーは既存共通アプリケーションAPI**に寄せる（[ADR-0048](../adr/0048-shared-application-api-and-transport-adapters.md)、#1158）。CLIをHTTP専用Clientにせず、TEI共通常駐Hostを新設しない。MCPも必要な時だけ薄いAdapterとして利用する。

### 2. 要求した範囲と実際に消費・受領した範囲を混同しない

Function viewの照合票は既存Source/Model/Artifact/Run/Scenario Evidenceへの**read-only導出**とする。単一の`available: true`、第二Canonical/Parser/Resolver/Runtime Policy/Proof Storeは作らない。

| 照合Claims | 何がそろえばその範囲で主張できるか | 禁止する昇格・戻すOwner |
| --- | --- | --- |
| **Design**（設計の必要閉包） | 対象/用途別の**独立Required**と、Declared→Discovered→Loaded→Parsed→Role/Meaning Bound、未発見の依存、Readerの対象Mode、出典・Identity・Revisionを比較 | `managed`や`readByLoader`、11応答、空`result:[]`、図の空欄から全Source/Function/Ruleが存在・受理済みとはしない（#3262/#2620） |
| **Source occurrence / 鮮度** | Consumer自身のSource/Workspace Manifestだけでなく、**Imported ProviderのRaw Source・Manifestの別Revision/実読込時点**、否定依存とSource-only変更を必要範囲で束縛 | Consumer World/SourceIndex/Model FPが同じでも外部Provider改訂やA→B→Aの履歴が不明なら現在引用・実Build入力再利用は保留（#3477、Draft #3479、#2600） |
| **Delivery**（実Consumer受領） | 同一Snapshot/ScopeのResponse ID・全part/順序/end marker・実受信byte・許容Disclosureを受領側で確認 | 内部JSON値一致/CLI exit 0／`content[0].text`一致だけで外部H2/H3配送完全・Authorizationを保証しない（#2509/#2515） |
| **Test**（新候補の判別力） | 既存Scenario[#1180](https://github.com/hat47x/tei-platform/issues/1180)と独立Expected[#2514](https://github.com/hat47x/tei-platform/issues/2514)に対し、既知T1-Aと**未知Obligation発見T1-B**、実Mutant・要件/C0/C1/NFR分母を別に評価 | 既存Pinned 6 Step・既知7th・32合成Consumer負例を全Coverage/独立AI発見としない |
| **Read / Projection** | Function/Queryの選択と**実際のSQL・Bind・Provider instanceのInvoke**→Rowset/Scope/Grain/Completeness→Data Contextを同一Attemptで照合。Graph/CRUD/Form/Tableへの投影は別Context/Selection epoch・情報Lossを説明 | SQL Candidate/Plugin selected・CRUD宣言`R`・参照PGの別Fixture成功をProduct実ReadやView変更の可逆Action資格へ転用しない（#2586/#2506/#2497/#3263） |
| **Build / Entry** | 選択rootと依存閉包、**actual builder-consumed** Source/Target/Toolchain/Lock、実Binary/ArtifactSet member、公開Entry/dispatcher、対象Consumerが読んだ版の順に照合 | Manifest-only、Plugin Capability Binding、Traitラベル、生成済Binary digestだけで実Link/Runtime Invoke・現在Grantを保証しない（#1167/#1130/#2595/#2604/#3267） |
| **Authority / Effect** | 対象Actor/Tenant/Scopeの現在Grant、実Guard/Operation ID/Attempt、Effect Certainty、Recorder/Provider Receipt、独立Resource再Read・外部handoffの**別観測** | HTTP 2xx/202、Job Ack、古いReceipt、空対象集合の正当化をBusiness Completedとしない（#1174/#3274/#3353、Draft #3328） |

「完全性」は**Claim相対**。設計閲覧・三視点図read-only生成は実DB Write GrantやEを待たない。他方、Eの未知をDesignの部分成功で相殺しない。秘匿対象の存在/件数/ID/digestは未許可Consumerへ開示せず、表示用の曖昧な「見つからない」で真の不在を主張しない。Function view自体もOwnerが受理した意味・Source version・actual observed内容の範囲を明示する。

### 3. 2026-11-08に成立した基線と、まだ接続されない最初の辺

- **Design/T0**：main [#3354](https://github.com/hat47x/tei-platform/pull/3354)の既存`context_gate.py`は**11実read-only CLI Reply・独立3 Rule/Relation/Source・32合成Consumer検査**を限定実証。main [#3304](https://github.com/hat47x/tei-platform/pull/3304)の既知行番号0はT1-A。現資格は`partial / exploration-only / qualifiedTestGeneration=false`であり、全Function閉包・T1-B・C1ではない。main [#3477](https://github.com/hat47x/tei-platform/pull/3477)の同Workspace Source-only差を検出する実CLI/Goは、**Imported Provider Source/Manifest**の現在性を自動証明しない（追加反証はDraft [#3479](https://github.com/hat47x/tei-platform/pull/3479)、新HEAD CIは別確認）。
- **Transport**：main [#3403](https://github.com/hat47x/tei-platform/pull/3403)の**read-only MCP参照Adapter（catalog/inspect/trace/validateの4 tool）**は既存`internal/operations`とCLIの限定値一致を実測。採用[ADR-0048](../adr/0048-shared-application-api-and-transport-adapters.md)の**全入口実装完了**、MCP Provider/Grant、外部Harnessの全Tool Reply受領は未実証。read-onlyという業務意味とOS上のFile Writeがゼロであることは別（#3333）。
- **Plugin/Inspector**：main [#3350](https://github.com/hat47x/tei-platform/pull/3350)/[#3363](https://github.com/hat47x/tei-platform/pull/3363)のCapability選択と、Draft [#3378](https://github.com/hat47x/tei-platform/pull/3378)のCLI/Studio direct Inspectorの基準同値、Draft [#3371](https://github.com/hat47x/tei-platform/pull/3371)/[#3381](https://github.com/hat47x/tei-platform/pull/3381)の型付き注入口を区別する。**実際に#951で選択したProviderのCLI/Studio双方でのInvokeはまだ未証明**。Provider B selected→A invokedを同じ返却値でも拒否する必要がある。Atomic TraitのRequired FulfillmentとVerified BOMのExact Tupleも独立（#3278/#3267、Draft #3379）。
- **実Function ReadとEffect**：[#2586](https://github.com/hat47x/tei-platform/issues/2586)の`order_entry.recent_orders`では、実Productの選択Query/SQL/bind→実Provider Rowset→Function Context→Consumerの**同一Attemptの縦断PASSが未確認**。main [#3353](https://github.com/hat47x/tei-platform/pull/3353)は空集合でもhandoff未確認をCompletionとしない回帰を追加したが、Callerの受領確認自己申告は外部独立証拠ではない。Draft [#3328](https://github.com/hat47x/tei-platform/pull/3328)のExecutor後Recorder/Stage失敗`EffectUnknown`修正は現main実装完了とみなさず、統合・再試験までunknown効果の安全な再送を主張しない。

### 4. 新たな巨大な計画や万能Gateを作らず、次の不足辺だけを埋める

| 優先 | 実行する最短比較と独立期待 | 担当・成功の上限 |
| --- | --- | --- |
| **P-1 / Context** | 一つの意味参照（`sales.order`のData・Rule・Relation）と、一つのProduct Function view（`recent_orders`）について独立Requiredを固定し、**既存CLI/Source Index/Parserにあるものとないもの**を差集合で記録。Source/Imported Providerの原文変更、Manifest変更、空・省略・Role欠落をそれぞれ区別 | #3262/#3477。欠落が見えることまでで、全Required完了やOwner Meaning受理ではない |
| **P0-A / Inspector実Provider** | 同じread-only Inspectの既存direct経路と、**選択したexact Contract/Providerが実Callされた経路**をCLIで比較し、selected B→invoked A、Builtin暗黙fallback、間違ったSubject/Sourceの3負例を拒否。次にStudioを**別Consumer**として確認 | #1251/#951/#1158/#1824。Draft #3378のdirect同値は先行baselineであってProvider D1bではない |
| **P0-B / Product実Read** | `recent_orders`を実Query→SQL raw/bind→Provider/Rowset/Contextまで通し、Scope/Grain/NULL/部分取得を独立Expectedと照合。別PG Referenceの成功を合算しない | #2586/#2506/#2514。実ReadはEditor/Write/Host Effectの資格ではない |
| **P0-C / Consumer接合** | 同じ実Rowset/Identity/Scopeと三視点図・CRUD・Form/Tableをread-onlyで照合。未知CRUD・多重辺・集計の逆写像不能、旧Selection Epochを検査 | #1124/#3263/#2497。SHO/Studio実Rendererと逆編集は別段階 |
| **P1 / Build・Plugin BOM** | 初めは標準**単一Go Binary**でroot/actual Source/Artifact/実Callを対照。必要性が示された場合のみ2/3 Unitを実験し、Provider×Consumer 2×2 exact Lockを同一Attempt・独立Testで検証 | #1130/#1167/#2600/#2604/#3267。Pairwise候補や過去RuleFingerprint一致はVerified配布・現在Grantではない |
| **P2 / Effect・Recovery・外部配送** | CLI/Web/Jobの同一意味を観測し、HTTP ACK-loss、Recorder/Stage失敗、期間Guard、旧Receipt、空Target＋handoff未確認、Import Source ABAを一因子差分で拒否。実Grant/Receipt/独立Read/Consumer Endを各Ownerに確認 | #1174/#3274/#1130/#2509。命令成功とBusiness完了を分け、未観測では無条件に再送しない |

個々のCaseのExpected・Execution・Coverage正本は[#1180](https://github.com/hat47x/tei-platform/issues/1180)/[#2514](https://github.com/hat47x/tei-platform/issues/2514)に置く。共通APIとSourceの実装は既存Ownerが持つ。**数え方**は「対象Scopeの独立Required/実取得/実消費/Consumer受領/未解決、最初に未成立となった辺、false-complete・false-stale、手設定数、Build latency/size、手修正量」。異なるRunのPASSを加算せず、正しい部分成果の再利用は妨げない。単一Binaryが優位な場合も結論として認める。

**撤回・縮退条件**：Derived Function viewが実際のSourceから導出できず第二の意味正本を必要とする、各CLI/MCP/Studio入口が同じ意味/結果を保てない、Imported Source/actual Providerを観測できず資格が常にunknown、複数Unit/Plugin分割が性能・理解・検証可能性を悪化させる場合には、追加抽象を撤回して既存の限定read-only操作・単一Binaryの構成に戻す。ここで新Parser/Registry/Host/Resolver/Policy/Test SSOTを採用しない。今回の文書変更で**新Go/CLI/実Provider/MCP-Harness/実Web/Job/Build/Effect、Owner受理はNOT RUN**。
<!-- tei-function-entry-qualification-crosslane-20261107 -->
## Functionの定義・実現・現在の提供可否を同じ根拠から照合する（2026-11-07、横断設計候補）

**目的**：§14/15のFunction→Build割当を、現在のCLI/MCPによる設計照会、標準Data/Function/Business図とCRUD対応、標準表現型、Web/CLI/Jobの共通Function、実Provider、Plugin組合せの検証と接続する。これらを「Function一覧に載った＝いま呼べる」という誤った一段階判定にしない。本文は既存Ownerへの受入条件の受渡しであり、新しい中央Registry、Catalog正本、Host、Runtime Policy Engine、Parser、テストSSOTを採用しない。

### 定義・物理実現・現在の利用資格は別の問い

「Functionがある」「そのビルドに収録された」「利用者がいま実行できる」「業務効果を確認した」は、それぞれ異なる主張である。判定キーは、少なくとも **Functionの安定IDと意味Revision、Application/Project、依存するContract/Plugin版、Target/Build Unitと実Artifact identity、Entry Point/Consumer、Actor/Tenant/Scope/現在Grant、Resource/環境/評価時点** のうち当該Claimに必要な項目を保持する。固定Handoverの意味版と実Target build版、生成物の公開版と実Consumerが読んだ版を一つのRevisionに畳まない。

| 層（read-onlyな照合列） | 正にするため必要な根拠 | 見落としてはいけない不成立 |
| --- | --- | --- |
| 定義と意味 | 実際に選択・Loaded/ParsedされたSource、stable Function/Action/Ruleと意味受理範囲、Source location/revision | Manifest掲載、readByLoader=true、機能定義書の生成だけではAccepted Meaningや実行資格にならない。未取得・部分・未解析を空集合にしない |
| 必須設計閉包 | Data/Function/Business、Query/SQL/Relation、Rule、要求/Scenario、依存するProvider・ContractのRequired集合と充足根拠 | 観測した行数/表示上のRelationだけでNOT EXISTS・全必須入力やC0/C1/NFR coverageを推定しない |
| Build実現 | 選択rootと依存閉包、宣言Sourceと実消費byte、Target/Toolchain/Plugin lock、実Go生成・link後Artifact/memberの証拠 | Selector/manifestが選択を宣言しただけ、別UnitのFunctionが存在するだけ、同digestだけでは物理包含・除外や再利用を保証しない |
| Entry公開と受領 | 明示されたCLI selector / Web route / OS Job trigger、実Dispatcher/Hostへのbinding、版互換、外部Consumerの実受領 | 同じBinaryへの収録は公開ではない。HTTP routeの存在・API schema・cron登録も実呼出し成功ではない |
| 現在の実行資格 | 実Actor/ScopeとHost grant、対象Resource/Provider資格、実Requestのtyped input/Context、必要Guardの充足 | PluginのOpen/Stable区分、UIでのボタン表示、Skill記述、CLIのexit 0は認可・安全な副作用の証拠にならない |
| 結果・業務効果 | Attemptごとの受付/起動/Result、Operation ID/Effect fence、Provider Receipt、独立した対象資源の再Readと業務完了条件 | Task completed、HTTP 2xx、Receipt取得、RunPlan step completedのいずれも、現在のBusiness Outcomeとは同一でない |

各列は **proven / contradicted / unknown / stale / unsupported / partial / not-observed** 等の根拠と適用範囲で示す。「全機能が利用可能」の単一booleanを新設しない。権限によって開示不能なSource・ID・件数・digest自体は非開示にできるようにし、権限不足と本当に存在しないことを外部Consumerへ漏らさない。

### 設計照会とBuild管理の共同入口：先にread-onlyで不備を見つける

最小の入口は [#3262 Design Context Query](https://github.com/hat47x/tei-platform/issues/3262) の **用途と対象に相対的なRequired閉包** とし、[#2482 機能定義書・対応一覧](https://github.com/hat47x/tei-platform/issues/2482) はそこからの派生Projectionとする。対象Functionの所属、Data/Function/Business三視点、CRUD宣言/業務経路/実Effectの区別、Context/Selectionと標準Data Shape・機能内Presentationの対応は既存[#1125](https://github.com/hat47x/tei-platform/issues/1125)、[#1127](https://github.com/hat47x/tei-platform/issues/1127)、[#2507](https://github.com/hat47x/tei-platform/issues/2507) に帰属させる。図面やCRUD表に描けることと、元Sourceへ可逆編集できることは別の資格とする。

CLIの既存inspect/catalog/operations/artifacts/validate等を実在する契約の範囲で照合し、まず **Required → Declared → Discovered → Loaded → Parsed → Meaning-bound → Build-consumed → Consumer-observed** の差を可視化する。各段階を同名の「発見済み」へ丸めない。特に現行mainの **readByLoaderはbyte取得の証拠のみ**であり、*.tei.mdはAuthoring上の統一方針と実Reader移行の完了を分ける。現在のoperations一覧が、宣言された全Function/Patternを網羅すると仮定しない。テスト設計や生成AIへの推薦は、独立Required集合と出力完全性（同Snapshotのpagination、終端、切断の有無）が確定した部分に限定する。CLIとMCPと人間向け表は同じQueryの別Projectionとし、MCP専用Reader/正本は作らない。

Build管理Pluginが提案する変更も、**照会結果 → 配置候補 → 差分Preview → 許可されたProject設定保存 → Build実行 → 成果物検証** を分離する。初期は設定不要の単一Go Binaryを維持し、複数Unitが必要な場合だけ既存[#1016](https://github.com/hat47x/tei-platform/issues/1016) のProject compositionと[#951](https://github.com/hat47x/tei-platform/issues/951) のResolverによる解決候補を利用する。Plugin版のrange・正確なlock・検証済みBOMの効力範囲は組合せ試験のOwnerに返す。OpenなPlugin APIの存在は実Runner/OS Secretの公開許諾に拡張しない。

### 複数入口の契約を「同じFunctionが同じ意味を維持する」条件で検証

CLI、Web/REST、Desktop、非対話CLI/cron、EKI委任は**Invocationの実現形**であり、業務Functionの別正本ではない。ただし入口の実装差（必要な対話、OS Resource、Session、Authn、Timeout、再試行、戻り値）は無視しない。共通のtyped Intent/Resultと [#1180 Scenario](https://github.com/hat47x/tei-platform/issues/1180) の独立保存Expectedを基準に、[#1158](https://github.com/hat47x/tei-platform/issues/1158) のAdapterで現実の到達範囲を比較する。CLIが存在するだけでWeb/Jobへ自動公開せず、対話必須Functionをheadless Jobへ無断変換しない。

HTTP経路の例外は[#3274](https://github.com/hat47x/tei-platform/issues/3274) の **Business Outcome → Execution/Effect certainty → HTTP projection → Client Reaction** で検証し、HTTP status・OpenAPI・Problem DetailsをFunction意味や回復規則の正本としない。Jobがtimeout/ACK喪失した場合は受付不明・Commit不明・Receipt Lookup不能・独立再Read未完了を別状態にする。安易に新Operation IDで再送せず、[#1130](https://github.com/hat47x/tei-platform/issues/1130) のRunPlan前提条件・部分Effectと[#1174](https://github.com/hat47x/tei-platform/issues/1174) の実Provider Effect FenceのOwnerへ渡す。

### 最短の縦断試行と停止条件（既存Fixtureを再利用）

| 順番 | 同一Function/Scenarioで追う差分 | 独立期待と受け渡し先 |
| --- | --- | --- |
| P-1 設計対象確定 | Acceptedな一つのFunctionに必要なSource/Business/Data/Rule/Query/Scenarioを列挙。実Loader/CLI結果とRequired集合の差を表示 | #3262 DCQ-01〜10、#2823。宣言だけ・切断・欠落・権限非開示でcomplete禁止。新CLI名の捏造禁止 |
| P0 単一Build | rootとnon-rootを含めてゼロ割当設定のGo単一Binaryへ対応付け、実Consumed input・Artifact/Entryを実測 | #1130 BU-01/08、#2595、#1167。実link、除外、Dispatcherが未観測ならunknown |
| P1 必要時のみ分割 | 同じMeaningで1→2→3 Build、依存共有/Source移動/版混在/Consumerが読んだbyteを比較 | #1130 BU-02/05/06/07、#2600 IB、#2604 AP。ビルド成功と公開可能を別判定 |
| P2 同一Scenarioの二入口 | まず副作用のないFunctionをCLIとWeb、次に非対話Job入口へ。typed inputとHost条件、実Result、旧新Consumer互換を比較 | #1180、#1158、#3274、#1130 BU-04。入口不足はunsupported、UIだけの成功は非代替 |
| P3 効果あり・回復 | 一つのEffectありFunctionで権限拒否、期間Guard欠落、部分Effect、ACK-loss、同一/異なる冪等key、旧Job再開を区別 | #1130 BU-09/ACU/RPL、#1174、#2514。Commit/独立Read/Owner完了のないResultは未確定 |

試行ごとに同じ **Function/Scenario/Source revision/Target/Artifact/Consumer/Actor/Attempt** に根拠を束縛し、どの一辺が初めて未成立かを示す。既存Scenarioと独立Oracleがテスト正本であり、この表は追加のFixture正本でも新Gateでもない。比べるのは、手設定数・手修正箇所・誤生成/誤除外・Build latency/size・起動コストと、利用者/生成AIが未充足理由を特定できるか。省略ビルドやPlugin分割が常に優位とは先に決めない。

**状態・採用線**：本追補は横断会話の成熟した制約を、既存のBuild/Job/Design Context/Scenario各Ownerへつなぐ設計候補。新規CLI/MCP、実マルチBuild、Web/cron/EKI/Studio/SHO、実Provider Read/Effect、Plugin BOMの実互換Matrix、正式Owner意味受理の新しい合格を主張しない。コード試験・E2Eは本変更では **NOT RUN**。

<!-- tei-function-executable-intake-20261107 -->
### 既存の実証が到達した範囲と、Function縦断の「最初の不足辺」（2026-11-07のmain再照合）

新たにmainへ統合された[#3284](https://github.com/hat47x/tei-platform/pull/3284)の`tools/reference-issue-test-ability/context_gate.py`は、同一TEI Binaryで受領した既存CLI **9応答**を使い、必須Sourceの`readByLoader`・Subjectの非曖昧性・Fingerprint一致・欠落/陳腐化などの7種類の負例を検査する。実CI [#42370519406](https://github.com/hat47x/tei-platform/actions/runs/42370519406)で限定実証され、探索用の結果は`boundedReference=partial`、`testExecutionAdmission=exploration-only`、`qualifiedTestGeneration=false`。その試験対象は**`semantic-data:sales.order`と`semantic/sales/order.tei`**であって、**任意の業務Functionの閉包や、Function自体の実行資格を証明していない**。既存の生成済Pinned Scenario 6 StepのT0成功も、新しい未カバールールを発見・生成するT1、実DB Read、別Targetへの移植、実Effectとは別。

[#3254](https://github.com/hat47x/tei-platform/issues/3254)では、Sourceへ属性を追加するとWorld FingerprintとModel Revisionは変化したが、5つのOperationの`basis`は変わらないという局所観測が記録された。Function一覧または派生Operationを選んだ時点の`basis`一致**だけ**で、呼出し時の新旧Meaning/Source同一性を判定してはならない。現CLI `tei operations` に実行時照合用`-expected-fingerprint`がないことは、現実の未実装として扱う。追加フラグ名・Runtime照合方式は[#3254](https://github.com/hat47x/tei-platform/issues/3254)/既存Operation Ownerの判断とする。

**最初の縦断対象は、実ProductにあるRead-only Functionを一つに限定する**。候補は[#2586](https://github.com/hat47x/tei-platform/issues/2586)が扱う`order_entry.recent_orders`であり、既存Pattern/Query/Data Source Alias宣言と、実際に選択されるSQL・引数・DB Instance・Provider Rowset・Function Data Contextがまだ**同一Attemptで証明されていない**点を起点にする。別PR[#3238](https://github.com/hat47x/tei-platform/pull/3238)の限定PostgreSQL Read成功、#3284の`sales.order` Source Read、#3254のOperation Drift観測は**別対象・別試行の部分証拠**であり、これらを連結済みProduct Function PASSと記載しない。

以下の最小受渡し表は**手編集して維持する第二SSOTではなく、現行証拠を照合するための読み取り専用Packetの形**とする。空欄に既定値や成功を補わず、当該Claimについて必要な一致キーと最初の不足辺を表示する。

| Claim/用途 | 実証済みの最低限の材料 | 現時点で次に照合すべき一辺 | 「成功」にしてはいけない状態 |
| --- | --- | --- | --- |
| D: 設計を読む | 既存CLI実9応答の限定`sales.order`局所Source/Subject/FP、T0.5 | **Function自身**のRequired Data/Business/Rule/Query/Roleを同Source/Revisionで解析・関連付け（#3262） | `readByLoader`から全Function・Accepted Meaningへ昇格 |
| T: テストを作る | Pinned Scenarioの既存6 Step・限定negative、`qualifiedTestGeneration=false` | 必要Ruleに対する新規Negativeの独立Expected/MutantとScope（#3273/#2514） | 既存T0、C0やCI triggerからT1/C1を推定 |
| R: 実データを読む | `recent_orders`の宣言候補、隔離PG Referenceの限定Rowset | Function→選択SQL原byte/Bind→実Provider Instance→Rowset/Scopeを**同一Attempt**で接続（#2586） | SQL digest一致、別FixtureのPG成功からProduct Read成立 |
| P: 見せる・操作候補 | Data/Representation Topology、三視点図・CRUD・Function Contextの設計候補 | **同じActual Result Slice**をForm/Tableと三視点図の対象Identity/Selectionに束縛し、損失・非可逆Actionを診断（#3236/#3263） | 図・CRUDの宣言だけで実Read/Write/Host Grantを推定 |
| B: Buildと提供入口 | 単一Go Binary既定・Build/Entry分離設計、既存Operation候補 | Function root/closure→実Consumed Source→Artifact→実Dispatcher/Consumer→現在Grant（#1167/#1130） | Built-in/Derived Operationまたは配置設定から公開・利用可能を推定 |
| E: 副作用と業務確定 | Effect/Receiptの段階分離・実PG限定試験、HTTP Graphの局所負例監査 | 実Request/Operation/Host/Provider/Receipt/独立ReadとHTTP/Jobの不明結果（#1174/#3274） | HTTP 2xx、Task completed、receipt-observedからBusiness Completedへ昇格 |

D/T/R/P/B/Eは**相互に独立した資格Claim**であり、Dが限定的に観測できた時点で、実Provider/実HostがないP0-Mの三視点図read-only投影やT0探索・Go単体Build検証を停止させない。反対に、B（物理包含）・P（描画）がPASSでもR（実読込）・E（効果）を引き上げない。D/T/R/P/Eの用途別定義は[#3288](https://github.com/hat47x/tei-platform/pull/3288)の**Draft候補**と整合させ、本節だけで正式Schema/Gateや順序の直列強制を決めない。

### Functionごとの「利用可能」の導出条件をさらに限定する

同じFunction IDでも **Query Providerの実版/DB instance、Plugin Target/Lock、Current Actor/Tenant/Scope/Grant、実Consumerが読んだArtifact/Contract** が違えば呼出し資格は違う。組合せBOMの[#3267](https://github.com/hat47x/tei-platform/issues/3267)はExact Tupleの互換性とCoverageの対象限定に利用し、Referenceのpairwise計画・合成`observed-pass`やSBOM一覧は、FunctionのRuntime Admissionに自動転用しない。異Version/同Version異digest・未試験高次組合せ・Open/unpromised APIの下流Adapterは個別にunknown/degraded/unsupportedを許す。

Atomic Traitを今後Plugin/Functionの契約義務に使う場合、**宣言による実装提供、実装を要求する検証、Provider/Hostへ必要能力を要求すること**を区別する。たとえば非対話Job可否・再試行時の冪等性・明示的な認可Guardの必要性を「Trait名が付与されているだけ」で実証済みとはしない。未実装Traitは開発時診断、実実装の局所Conformanceと現在Host Grantは別。Traitの正式語彙・組合せ・採否は既存Plugin/標準Data/Contract Ownerに返し、Job用の包括TraitやCore Enumは先行新設しない。

**最短の次の実験**：①#3284をそのまま再利用し、限定`sales.order` Readinessを保持する。②対象を`order_entry.recent_orders`へ切り替えたとき不足するFunction/Query/Rule/Loader/Actual Buildの列を、既存CLI/Sourceで示す。③不足が最初に確認された一辺だけを#3262または#2586の既存Ownerで実装・計測し、同じSource Revisionで比較する。④同じFunctionへのP0-M図/CRUD投影とT1 Negative Caseを**並行**で進める。⑤これらが限定成立してから実CLI/Web/Job、2/3 Unit、実Effectへ拡張する。比較の計測は「必要だった手動指定数、初めて不成立となる一辺の検出率、false complete/false stale、Consumer受領、Target/Bundle適合、1→N Build追加コスト」で行う。

**本追補の実行資格**：#3284の現行mainコードと既存PRの公開CI記録・Issueを読取照合した計画差分。新しいFunction Context Product E2E、実`recent_orders` Provider/Action、Graph/CRUD Renderer、2/3 Binary、Job/HTTP Consumer、Trait Conformance、BOM実Compat Matrix、Owner意味受理は**本PRではNOT RUN**。

<!-- tei-function-qualification-exec-maturity-20261108 -->
### 2026-11-08 実CLI・Plugin選択・実行後不明状態の前進を受けたFunction資格の更新（設計候補）

**今回の基準の変更**：「Source/Functionを読めるか」から「どの必要集合をどの実Consumerが使ったか」へ進める。全Workspace共通の`complete=true`やFunction一般の`executable=true`を導入せず、**用途（設計D／テストT／実Read R／表現P／実Build・入口B／副作用E）と評価したSubject・Source snapshot・Actor/Scope・実Target・Attemptごとの最初の不足辺**を表示する。

#### 実装済みの限定証拠と未成立の正確な境界

| 同一時点の根拠 | 確認できた範囲 | Function資格へ昇格しない範囲 |
| --- | --- | --- |
| main統合 [#3354](https://github.com/hat47x/tei-platform/pull/3354)、[exact-head専用CI #42391707962](https://github.com/hat47x/tei-platform/actions/runs/42391707962) | 既存`context_gate.py`を単一経路に維持し、**11実CLI read-only Reply**（全`source-index`と`inspect-validation`を含む）・公開合成**3 Rule/Relation/Source**の独立Required・**32件の合成Consumer判別**、既存6 Pinned Step＋既知Rule候補を実行。旧#3324/#3326は未統合でクローズ | **全Function/Business/Procedure閉包、原子的Workspace snapshot、独立C1/全NFR、未知RuleをAIが発見するT1-B、外部H3完全受領は未成立**。32件はProduct Code Mutant 32件でない。従来の「9 Replyのみ」は履歴 |
| main統合 [#3315](https://github.com/hat47x/tei-platform/pull/3315) | read-only CLIの複数応答にWorkspace名、Manifest format版、Manifest byte digestを付加 | 同名別Workspace、A→B→A、同Manifest下のSource/Grant変更、全Required sourceの原子的同版を証明しない。`source-index`等すべての応答にWorkspace Identityがあるとは仮定しない |
| main統合 [#3350](https://github.com/hat47x/tei-platform/pull/3350) | `database-instance`技術名を`SupportsDatabaseProvider`で選択し、Reference上のunselected数を6→4へ訂正 | **選択候補≠Link/Load/Invoke/Grant/SQL実発行/Rowset**。Plugin-first D1のCLI/Studio二Consumerは別試行（#1251/#951/#1824） |
| main統合 [#3304](https://github.com/hat47x/tei-platform/pull/3304) | 公開Ruleの行番号0を既存Scenarioの一時第7 Stepとして実CLI判別（既知T1-A） | 生成AIが未知Ruleを盲検探索して新規負例を採用したT1-B、独立Product Mutant/C1や正式Scenario採用ではない |
| 未統合 [Draft #3328](https://github.com/hat47x/tei-platform/pull/3328) | Executor後のRecorder/Stage失敗を`EffectUnknown`、`MayReplay`とDispatchの不整合を修正するGo提案と5テスト | **mainの現`internal/command/invoke.go`ではRecorder失敗時`EffectUnknown`が未設定、`lifecycle.go`のStage記録失敗も同様**。Draftの修正はmainの安全性へ未反映、実PG/プロセス再起動・Fence/Receipt未検証 |
| 未統合 [Draft #3353](https://github.com/hat47x/tei-platform/pull/3353) | 空TargetでもHandoff未確認ならCompletion不確定とする修正候補 | `EmptyJustified`が正しくても外部受領済み・業務完了の証拠はない。Caller申告の`HandoffConfirmed`自体も独立確認ではない |

※ここでのCIは、それぞれ記載した**対象HEAD/Fixtureでの限定実証**である。異なるPR/Build/Actor/時点で得たPASSを、一つの`recent_orders`実Product End-to-Endに足し合わせない。特にmainの`internal/command`は今回GitHubで読み取った旧挙動であり、[#3328](https://github.com/hat47x/tei-platform/pull/3328)が統合されれば、その**新しいmain HEADで再評価**する。

#### 最小のread-only Function資格照合（具体例）

**入力**（新API/JSON Schemaの採用ではなく既存Readerへの照会条件）：`order_entry.recent_orders`（Candidate Function）／対象WorkspaceとそのSource・Manifest同一性／利用目的`R`（Product Read）／要求するContextのData/Rule/Query/Result grain／許されたActor/Tenant/Scope／期待Source/Operation Revision／Plugin exact resolution／Targetと実Consumer。既存CLIで得られない欄は「ない」と推測せず`unknown / unobserved / unsupported`で返す。必要がない実Write GrantまでRの読取り資格に要求しない。

**最小出力の導出列**：`Required IDs+Roles → Found/Loaded/Parsed/Bound IDs → Declared Function/Query & SQL candidates → Selected SQL/args & actual Provider Invoke → actual Result Slice/Scope/Completeness → Context/Consumer accepted bytes → Claim evidence ceiling`。Root/Negative Dependencyや全件空集合も「列挙されなかった」だけで不要・なしとしない。同じSubject/FPの各CLI replyを逐次受領した**11コマンドの一致**は原子的Snapshotを意味しない。実行途中でSource/Manifest/Grant/Selection Epochが変化した場合は、実入力と受領byteを別に記録して旧結果の現在資格を保留する（#3181/#3327/#3254/#2582）。

**結果の期待**：現時点は`recent_orders`の宣言されたQuery/aliasが存在しても、**BuildBrowseの候補選択→実Product Caller→選択SQLとbind値の実消費→実Provider接続→Rowset→Function Context→Consumer**の一連が未確認。製品Rについて`unverified`を返し、最初の失われた辺だけを[#2586](https://github.com/hat47x/tei-platform/issues/2586)（Function/Query/Result縦断）へ、Context Requiredの取得問題を[#3262](https://github.com/hat47x/tei-platform/issues/3262)へ渡す。別Referenceの実PostgreSQL read、Read UIだけ、Plugin selected、World Fingerprint一致は代替にならない。

#### Build/Entry/Effectへ進む際の反例を一つのScenarioから分岐する

- **Read-onlyで実Providerを呼べた後**：Function/Plugin Contractと実Consumed SQL/SourceをArtifactへ結び、単一Go Binaryの実包含・実Dispatcher・外部Consumer受領を検査してB資格とする。2/3 Unit分割は初回実ReadやCLI Context完全性の先行必須条件にはしない（#1130/#1167/#2600/#2604）。
- **直前改訂と現在Grant**：#3254で確認した`basis`不変・World/Model改訂の負例を、実行直前の適合チェックに再利用。Command/Jobの入場はSource/Operation/Contract/Provider/Actorの必要条件を照合する。Snapshotが古い／必要Grant不明なら要求を拒否または保留し、単に`basis`一致・Open Plugin/Trait宣言や旧BOMで許可しない。採用するRuntime API/Policyは各既存Owner判断。
- **Executor以後の記録失敗**：Recorder/Stageが失敗しても既に業務Effectが起こった可能性がある。**Clean Rejectedと断定しない**。#3328の限定修正を統合できたか最新mainで再確認した上で、#1174/#2564の独立PG Receipt・別Connection再Read、ACK-loss/同Operation再送/別key/ABA、永続Stageの実試験へ進む。Resultの`EffectUnknown`は外部Effectの観測済み証明や自動回復ではない。
- **空集合と業務完了**：必要な対象集合が正当に0件でも、Workflowのhandoff/External Consumer ACK・対象ResourceのIndependent Readは別。#3353の空集合正当化とHandoff未確認を、Job完了判定・表示上の「完了」に混ぜない。対象集合の真の空／未探索／権限非開示を分ける（#1706/#1125/#1130）。

**引渡しと段階**：先行してよいのは `D`の既存11-Reply範囲を使った欠落表示、`T`の既知T1-A後続（盲検T1-B）、`P`の三視点図・CRUD read-only Projection（後継Draft #3303）、`R`の一Function実Product Readの**並列**。B（物理配置）とE（実副作用）をすべての作業の直列Gateにしない。初回Rが成立した時点で、同一Row Identity/Scope/Source Revisionを`P`の表/図と結ぶCrossviewを確認し、以後同一ScenarioのCLI/Web/Jobへ拡張する。各段階のError/Incomplete/Unknownと人・生成AIの必要手順数を[#2018](https://github.com/hat47x/tei-platform/issues/2018)へ返す。

**今回の追加は計画文書のみ**。新Product Query接続、MCP、Plugin D1実呼出、11コマンド原子的Snapshot、実2/3 Build、Web/Job/Studio/SHO、実DB Command故障注入/回復、外部H3完備、Owner意味受理は**本PRではNOT RUN**。第二Reader/Host/Scenario SSOT/Resolver/Policy/Runtime Registryを追加しない。
