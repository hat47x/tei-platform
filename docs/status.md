# TEIの実装状況

- 整理日: 2026-10-25
- 開発段階: **参照実装を使った実験と検証**
- 品質に関する注意: **生成AIを用いて開発中であり、コード・設計・安全性の体系的な人的確認は未実施**です。本番利用を想定していません。

TEIは、分野固有の知識と実装技術を分けて管理するローコード開発基盤を目指しています。現在は業務システムを主な参照事例とし、コマンドラインと参照用作業領域を使って、設計・実装・実行・観測の関係を確かめています。現時点では、完成した製品、画面編集機能、汎用の実行基盤は提供していません。全体像は[README](../README.md)、価値と対象範囲は[製品が大切にする価値](../product/value/TEI_PRODUCT_VALUE.md)と[製品要求](../requirements/TEI_PRODUCT_REQUIREMENTS.md)を参照してください。

## 実装と検証で確認できること

| 領域 | 状況 | 確認先 |
| --- | --- | --- |
| 初回の業務モデル変更 | 独立した小さな作業領域で定義・入力元索引・検証・修復を扱う参照事例と回帰試験を用意。**画面生成・保存・外部連携の実証ではありません** | [初回モデル変更の手順](../examples/first-workspace/README.md)、[課題 #1161](https://github.com/hat47x/tei-platform/issues/1161) |
| GitHub Actionsによるコード検証 | 2026年10月18日、PR #2012の統合コミット`0cc76fd992b0d25960ad0a6b27b96c569a9472b4`でRepository Healthが成功。確認時の`main`は`2c8c5bfe01e77a40ce1b1072f9d76dad82ef15d0`でこのコミットを含み、その後の差分は`tools/`配下とDesktop WebView smoke試験。**最新main SHAそのもののRepository Health再実行は未確認** | [Repository Health #39112031894](https://github.com/hat47x/tei-platform/actions/runs/39112031894)、[CI整備の課題 #1189](https://github.com/hat47x/tei-platform/issues/1189) |
| Goのコマンドラインと作業領域 | `cmd/tei`に作業領域の読み込み、検証、入力元索引の作成などを実装。コマンドの例は導入手順に記載 | [使い始める](getting-started.md)、[参照用資料](../reference/README.md) |
| TEI StudioのPoC A（Wails Web/Desktop Host） | 2026年10月18日、PR #2012の検証済み候補`c2d8866ba72bb877c4b74f444fe95b3b89368097`で二つのBrowserContext間のWorkspace選択・Catalog・Inspect・Validate・ObserveXeadと遅延応答の分離を確認し、統合コミット`0cc76fd992b0d25960ad0a6b27b96c569a9472b4`でもHost、Windows Desktop WebView2、Repository Healthが成功。後続PR #2014では空結果と読取失敗のDesktop表示も確認。**証拠は同一OS利用者・loopback Host内に限り、Workspace-ID認可、共有WebやPrincipal別の利用者分離、実利用者の価値を示しません。PoC A/B/C全体の完了も示しません** | [協調Issue #1824](https://github.com/hat47x/tei-platform/issues/1824)、[PR #2012](https://github.com/hat47x/tei-platform/pull/2012)、[PR #2014](https://github.com/hat47x/tei-platform/pull/2014)、[Wails Host #39112031888](https://github.com/hat47x/tei-platform/actions/runs/39112031888)、[Desktop WebView2 #39112031903](https://github.com/hat47x/tei-platform/actions/runs/39112031903)、[PR #2014 Desktop WebView2 #39113218123](https://github.com/hat47x/tei-platform/actions/runs/39113218123) |
| アプリケーションとドライバー | `order_entry`で一覧表示、記録、参照、親子関係にある明細、操作などの意味上の境界を実証 | [全体構成](architecture.md)、[版1の計画](../plan/V1_REFERENCE_APPLICATION_PLAN.md) |
| プラグイン合成基盤 | 契約と利用範囲による提供元の選択、推移的な依存宣言の鮮度、呼出しの世代保持、逆順解放を扱うReferenceを実装。構成案の一括診断と、未検証の適合証拠参照との照合も実装。対象コミットでの全体健全性とRace Detectorによる試験を確認 | [KernelのReference](../reference/composition-kernel/README.md)、[資格記録 #1233](https://github.com/hat47x/tei-platform/pull/1233)、[二Applicationの資格記録 #1622](https://github.com/hat47x/tei-platform/pull/1622) |
| プラグイン間の項目対応 | 既存の登録簿を使い、局所名、型・単位・多重度、出力から入力への順序を検査する参照実装を追加。恒等のスカラー試料を投影し、空文字・null・欠損・取得不能・未確定を区別する。製品の入力形式や実行時の対応付けは未確定 | [項目対応の参照実装](../reference/composition-io-mapping/README.md)、[資格記録 #1667](https://github.com/hat47x/tei-platform/pull/1667) |
| 判定段階A | **合格**。アプリケーションの型からドライバーへ渡す意味上の境界を確認した過去の判定 | [段階Aの判定記録](../plan/gate/GATE_A_READINESS_AUDIT.md) |
| 判定段階B | **合格**。2026年9月19日に固定した版`e5231839a5dcc62cce1181a2731fcb7c70d16736`について、リポジトリ上の再現性、生成物、PostgreSQL実環境を同じ実行で確認 | [段階Bの判定記録](../plan/gate/GATE_B_READINESS_AUDIT.md)、[実行手順](../plan/gate/GATE_B_EVIDENCE_RUNBOOK.md) |
| 判定段階Cと最初のウェブ実行先 | **合格**。2026年9月19日に固定したビルド`0ba87a4e0a96f70a198b03ff7652fb00e3826abe`の意味、機能、表示を検証。**表示の忠実度は一人のAI評価者が判定しており、人による承認ではありません** | [段階Cの判定記録](../plan/gate/GATE_C_READINESS_AUDIT.md)、[検証根拠の条件](../plan/gate/GATE_C_EVIDENCE_CONTRACT.md) |
| 共通受理と二つ目のアプリケーション | `repair_intake`の意味定義の一部を、明示的な受理を経て参照用作業領域へ取り込む経路などを検討・実装。受理済みであることと、実行先で動作を確認済みであることは別 | [参照用作業領域の案内](../reference/README.md)、[実装の構成](architecture.md) |
| Web起動前の入力元照合 | 複合Web TargetにHost Readerの注入口を追加し、同じ読込結果から管理対象Sourceを照合してからRunnerとHTTPを生成する。通常CLIは正式Reader未接続のため起動拒否。合成fixtureには明示的な診断起動を用いる。**真正なHost受理・人的受理・製品G1/G2は未達** | [Web起動前の照合](../tools/reference-first-web-target/README.md)、[横断受入 #1119](https://github.com/hat47x/tei-platform/issues/1119) |
| 生成器候補の選抜・来歴・artifact | 固定fixtureの選抜、source/binary digest、Go/Java/WASM/WASI/Python/JavaScriptのartifact保存、prune auditとcheckpointの署名・再検証を参照実装で確認。**実Provider・実Target・署名者Authority・別machine再現性は未確認** | [tools索引](../tools/README.md)、[ADR-0041](../plan/adr/0041-generated-artifact-provenance-boundaries.md)、[#2016](https://github.com/hat47x/tei-platform/issues/2016) |

判定段階A・B・Cの「合格」は、**各判定時点に固定したソース、ビルド、検証根拠に対するもの**です。現在の`main`、すべての提供元、未知の業務、本番環境、別の実行先を自動的に合格とするものではありません。過去の検証結果を、そのまま新しい変更の検証根拠にはしません。

合成基盤と[実Processとの接続Reference](../tools/reference-composition-process-bridge/README.md)では、実際に構築した成果物と子プロセスの応答を照合します。呼出しごとに起動するProcessの経路切替と、試験用のHostガードを確認する範囲です。常駐サービスの無停止更新、実配備先の現在権限、製品Consumerからの旧Resolver撤去は未完了で、[合成Kernelの移行判定](../plan/gate/COMPOSITION_KERNEL_MIGRATION_GATE_2026-10.md)と[課題 #1223](https://github.com/hat47x/tei-platform/issues/1223)で追跡します。入出力の一般的な対応付けは[評価中の設計候補](../plan/design/PLUGIN_DEPENDENCY_IO_MAPPING_DESIGN_2026-10.md)で、正本の項目定義、集合の縮約、演算子との接続が残ります。

## 未完成または未検証のこと

- **統合モデラーと編集環境**: 人やAIの入力からデータ、機能、業務の三つの視点を一貫して編集・受理・実行する製品としては未完成です。
- **二つ目の実行先と多様な配備先**: 同じ業務定義から汎用ウェブ、デスクトップ、モバイル、組み込み機器、IoTを動かした製品上の検証根拠はありません。
- **外部接続と実際の提供元**: 参照実装で使う合成試験データによる確認と、Oracleなどの実環境で認証・接続・操作が成立したことを区別します。未取得の検証根拠を成功として扱いません。
- **実行先での再現可能な操作**: 判定段階Cの初回結果と、後続のR7再実行・反復確認は別の検証です。未実施の確認も残っています。
- **安全性と品質**: 安全性、個人情報、第三者の素材の利用条件、権限設定に関する包括的な監査、体系的な人的確認、本番運用での耐久性・高可用性の確認は未完了です。
- **開発用CIと製品上の受入**: GitHub Actionsでは固定した版ごとにコード検証とStudioのHost・実ブラウザー・Windows WebView2スモーク確認を実施しています。一方、まだ確認していない改訂や別のプルリクエストまで自動的に合格とはなりません。Studioの確認は、初めて利用する人による一連の操作、三つの視点の編集と再反映、データベースへの保存、独立した読取りによる結果確認、認可境界、実利用者の価値を示しません。PoC A/B/Cの完了判定と必要な製品実証は[協調Issue #1824](https://github.com/hat47x/tei-platform/issues/1824)、[要件の優先順位 #982](https://github.com/hat47x/tei-platform/issues/982)、[横断受入 #1119](https://github.com/hat47x/tei-platform/issues/1119)で追跡します。主ブランチの保護と必須チェックの設定も[課題 #1189](https://github.com/hat47x/tei-platform/issues/1189)で未完了です。

検証済みの実装と、まだ確定していない設計を区別して開発を進めます。優先順位は[TEIの全体計画](../plan/MASTER_PLAN.md)、技術上の境界は[全体構成](architecture.md)、業務設計の考え方は[業務システム設計](business-system-design.md)を参照してください。

## 最初に動かす

Goの開発環境で行う主なローカル確認は、次のとおりです。

```bash
go test ./...
go run ./cmd/tei validate -workspace reference/workspace
```

詳しい確認手順と参照用作業領域の範囲は[使い始める](getting-started.md)を参照してください。手順の掲載は、公開候補や最新のコミットが試験に合格したことを意味しません。
