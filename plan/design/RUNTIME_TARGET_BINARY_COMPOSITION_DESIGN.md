# 多様な実行環境に向けた最小ランタイムと配備単位の設計

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

## 2. 最小限の共通構成

「ランタイム」という語は、次の責務を混同しないために限定して使う。

| 区分 | 最小限の責務 | 常駐が必要か |
| --- | --- | --- |
| 意味を扱う共通部分 | 型・識別・制約等の正本から実行に必要な表現を生成する。 | **生成時のみ**でよい。実行先でDSL解析器を動かすことを要求しない。 |
| 構成・ビルド機構 | 固定された契約・実装・依存・Target・Builderから成果物を作る。 | **生成時のみ**。配備先では不要。 |
| 最小実行入口 | 呼出しを受け、固定された実装へ渡し、結果・失敗・不明を区別する。 | その実行単位が呼出しを扱う場合だけ必要。 |
| 実行に必要な能力 | 時計・記憶域・通信・並行処理・秘密情報等のうち宣言されたものをHostへ接続する。 | 必要な能力だけを含める。 |
| 実行時の保護 | 副作用を行う際に、そのHostが所有する最新の認可・Resource権限を検査する。 | 副作用を担う境界では必須。 |
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

## 11. 判断を保留するもの

- `tei.deploy.yaml`という新たな正本を本当に必要とするか（既存Project/Profile/Environment定義への小さな追加で済む可能性がある）。
- Closure Plannerを独立したPackageとするか、既存のTarget packagingへ統合するか。
- 保持集合の宣言方式、Host Capabilityの型、配備目録のschema。
- Wasm Component/WASI版、TinyGo/その他Builderの標準採用。
- 動的更新と差分ファームウェアの共通化、全Target共通のサンドボックス契約。

この文書から直ちにCanonical種別、固定Target列挙、第二のLock/Resolver、大規模なRuntime Frameworkを新設しない。#259の第二Target比較、#943のBuild根拠、#951のCapability解決、#1016の案件構成の実証で不足が判明した部分だけを、Owner側で採用判断する。
