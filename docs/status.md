# TEIの実装状況

- 整理日: 2026-10-14
- 開発段階: **参照実装を使った実験と検証**
- 品質に関する注意: **生成AIを用いて開発中であり、コード・設計・安全性の体系的な人的確認は未実施**です。本番利用を想定していません。

TEIは、分野固有の知識と実装技術を分けて管理するローコード開発基盤を目指しています。現在は業務システムを主な参照事例とし、コマンドラインと参照用作業領域を使って、設計・実装・実行・観測の関係を確かめています。現時点では、完成した製品、画面編集機能、汎用の実行基盤は提供していません。全体像は[README](../README.md)、価値と対象範囲は[製品が大切にする価値](../product/value/TEI_PRODUCT_VALUE.md)と[製品要求](../requirements/TEI_PRODUCT_REQUIREMENTS.md)を参照してください。

## 実装と検証で確認できること

| 領域 | 状況 | 確認先 |
| --- | --- | --- |
| 統合した受注業務の参照モデル | 開発候補として、ルール改訂後の受注締めに、入力訂正、部分確定、確定許可の失効、停止後照合、明示的な残件確定、最終CSV／JSONを統合。各件で選択済み確定世代を保持し、同時編集後の古い選択と撤去済み実装を拒否する。世代復旧とDB許可更新は別に検査。訂正・許可管理は検証用SQLで、製品の編集・運用操作、人の承認、本番認可、永続要求IDによる再開は未成立 | [受注ルールの改訂から業務完了まで](../reference/order-workflow/README.md) |
| 常駐する参照プロセスの世代交換 | 開発候補として、旧プロセスで処理を保持しながら新しい実行物とPIDへ切り替える試験を追加。Linuxでは検証した実行内容を起動まで固定し、元パスの差替えを反映させない。終了待ちの失敗、準備拒否、取消・異常終了と明示的な復旧を扱う。処理要求の受領までの参照例であり、永続的な更新、本番権限、性能・可用性目標は未成立 | [実行方法と制約](../tools/reference-composition-process-bridge/README.md) |
| 参照シナリオと新しいプラグイン基盤 | 開発候補として、型付き実装を選択済み世代へ接続。`-composition-scenarios`で受注の読取と登録・後続観測、修理受付の顧客参照を実行する。アプリケーション専用ビルドでも接続を選べる。修理受付は模擬Oracleを使い、実接続・本番の提供元移行・無停止運用は未成立 | [実行方法](../tools/reference-first-web-target/README.md)、[接続計画](../plan/workstreams/REFERENCE_MODEL_PLUGIN_EXECUTION_2026-10.md) |
| PostgreSQL上の受注シナリオ | 開発候補として、受注・明細の読取、確定、別読取りによる観測を実DBへ接続。受注専用ビルドと固定WorldのAPIでも同じアダプターを使う。現在の実行許可、排他トークン、古い読取版、重複確定を検査する。顧客参照と認証は模擬実装で、本番接続や汎用の権限基盤への移行は未成立 | [参照データと再現手順](../reference/order-workflow/README.md) |
| 画面を使わない帳票・一括処理 | 開発候補として、同一契約のCSV／JSON整形と、対象を固定した少数件の受注確定を接続。受注専用・固定Worldの実行物でも利用できる。各件の検証・認可・更新・観測を共有し、停止後はID指定で現在のDB状態を読取り専用で診断する。取消と部分成功の記録、現在の状態による重複確定拒否を扱う。永続的な要求IDによる重複排除、自動再開、PDF表示、本番運用は未成立 | [実行方法と制約](../tools/reference-first-web-target/NON_SCREEN.md)、[整形プラグイン](../plugins/report-renderer/README.md) |
| 共通受理と実行版の接続 | 開発候補として、共通受理で変更した検証用ソースを再読込みし、正確なソース版を投影・プラグイン呼出し・DBの実行許可へ結ぶ。古い版や読取後のソース変更は確定前に拒否する。人の承認・Studio編集・無停止のモデル交換は未成立 | [変更案から再実行までの参照例](../reference/order-workflow/README.md) |
| 初回の業務モデル変更 | 独立した小さな作業領域で定義・入力元索引・検証・修復を扱う参照事例と回帰試験を用意。**画面生成・保存・外部連携の実証ではありません** | [初回モデル変更の手順](../examples/first-workspace/README.md)、[課題 #1161](https://github.com/hat47x/tei-platform/issues/1161) |
| GitHub Actionsによるコード検証 | 2026年10月12日に自動検査を再開。固定した`main`コミット`4fbcd92003f04254ef25882deec5549872a3c58e`で、コードの整形、静的解析、Goの全試験、参照用作業領域の検証がすべて成功。**製品の実行先や外部データベースの検証とは別** | [実行 #38221407786](https://github.com/hat47x/tei-platform/actions/runs/38221407786)、[CI整備の課題 #1189](https://github.com/hat47x/tei-platform/issues/1189) |
| Goのコマンドラインと作業領域 | `cmd/tei`に作業領域の読み込み、検証、入力元索引の作成などを実装。コマンドの例は導入手順に記載 | [使い始める](getting-started.md)、[参照用資料](../reference/README.md) |
| アプリケーションとドライバー | `order_entry`で一覧表示、記録、参照、親子関係にある明細、操作などの意味上の境界を実証 | [全体構成](architecture.md)、[版1の計画](../plan/V1_REFERENCE_APPLICATION_PLAN.md) |
| プラグイン合成基盤 | 契約と利用範囲による提供元の選択、推移的な依存宣言の鮮度、呼出しの世代保持、逆順解放を扱うReferenceを実装。構成案の一括診断と、未検証の適合証拠参照との照合も実装。対象コミットでの全体健全性とRace Detectorによる試験を確認 | [KernelのReference](../reference/composition-kernel/README.md)、[資格記録 #1233](https://github.com/hat47x/tei-platform/pull/1233)、[二Applicationの資格記録 #1622](https://github.com/hat47x/tei-platform/pull/1622) |
| プラグイン間の項目対応 | 既存の登録簿を使い、局所名、型・単位・多重度、出力から入力への順序を検査する参照実装を追加。恒等のスカラー試料を投影し、空文字・null・欠損・取得不能・未確定を区別する。製品の入力形式や実行時の対応付けは未確定 | [項目対応の参照実装](../reference/composition-io-mapping/README.md)、[資格記録 #1667](https://github.com/hat47x/tei-platform/pull/1667) |
| 判定段階A | **合格**。アプリケーションの型からドライバーへ渡す意味上の境界を確認した過去の判定 | [段階Aの判定記録](../plan/gate/GATE_A_READINESS_AUDIT.md) |
| 判定段階B | **合格**。2026年9月19日に固定した版`e5231839a5dcc62cce1181a2731fcb7c70d16736`について、リポジトリ上の再現性、生成物、PostgreSQL実環境を同じ実行で確認 | [段階Bの判定記録](../plan/gate/GATE_B_READINESS_AUDIT.md)、[実行手順](../plan/gate/GATE_B_EVIDENCE_RUNBOOK.md) |
| 判定段階Cと最初のウェブ実行先 | **合格**。2026年9月19日に固定したビルド`0ba87a4e0a96f70a198b03ff7652fb00e3826abe`の意味、機能、表示を検証。**表示の忠実度は一人のAI評価者が判定しており、人による承認ではありません** | [段階Cの判定記録](../plan/gate/GATE_C_READINESS_AUDIT.md)、[検証根拠の条件](../plan/gate/GATE_C_EVIDENCE_CONTRACT.md) |
| 共通受理と二つ目のアプリケーション | `repair_intake`の意味定義の一部を、明示的な受理を経て参照用作業領域へ取り込む経路などを検討・実装。受理済みであることと、実行先で動作を確認済みであることは別 | [参照用作業領域の案内](../reference/README.md)、[実装の構成](architecture.md) |
| Web起動前の入力元照合 | 複合Web TargetにHost Readerの注入口を追加し、同じ読込結果から管理対象Sourceを照合してからRunnerとHTTPを生成する。通常CLIは正式Reader未接続のため起動拒否。合成fixtureには明示的な診断起動を用いる。**真正なHost受理・人的受理・製品G1/G2は未達** | [Web起動前の照合](../tools/reference-first-web-target/README.md)、[横断受入 #1119](https://github.com/hat47x/tei-platform/issues/1119) |

判定段階A・B・Cの「合格」は、**各判定時点に固定したソース、ビルド、検証根拠に対するもの**です。現在の`main`、すべての提供元、未知の業務、本番環境、別の実行先を自動的に合格とするものではありません。過去の検証結果を、そのまま新しい変更の検証根拠にはしません。

合成基盤と[実Processとの接続Reference](../tools/reference-composition-process-bridge/README.md)では、実際に構築した成果物と子プロセスの応答を照合します。呼出しごとに起動するProcessの経路切替と、試験用のHostガードを確認する範囲です。常駐サービスの無停止更新、実配備先の現在権限、製品Consumerからの旧Resolver撤去は未完了で、[合成Kernelの移行判定](../plan/gate/COMPOSITION_KERNEL_MIGRATION_GATE_2026-10.md)と[課題 #1223](https://github.com/hat47x/tei-platform/issues/1223)で追跡します。入出力の一般的な対応付けは[評価中の設計候補](../plan/design/PLUGIN_DEPENDENCY_IO_MAPPING_DESIGN_2026-10.md)で、正本の項目定義、集合の縮約、演算子との接続が残ります。

## 未完成または未検証のこと

- **統合モデラーと編集環境**: 人やAIの入力からデータ、機能、業務の三つの視点を一貫して編集・受理・実行する製品としては未完成です。
- **二つ目の実行先と多様な配備先**: 同じ業務定義から汎用ウェブ、デスクトップ、モバイル、組み込み機器、IoTを動かした製品上の検証根拠はありません。
- **外部接続と実際の提供元**: 参照実装で使う合成試験データによる確認と、Oracleなどの実環境で認証・接続・操作が成立したことを区別します。未取得の検証根拠を成功として扱いません。
- **実行先での再現可能な操作**: 判定段階Cの初回結果と、後続のR7再実行・反復確認は別の検証です。未実施の確認も残っています。
- **安全性と品質**: 安全性、個人情報、第三者の素材の利用条件、権限設定に関する包括的な監査、体系的な人的確認、本番運用での耐久性・高可用性の確認は未完了です。
- **開発用CIと製品上の受入**: GitHub Actionsによるコードの全体検証は再開し、上記の特定コミットで成功しました。一方、まだ確認していない改訂や別のプルリクエストまで、自動的に合格とはなりません。この自動検査には、初めて利用する人による一連の操作、三つの視点の編集と再反映、実際のウェブ操作、データベースへの保存、独立した読取りによる結果確認は含まれません。必要な製品実証は[要件の優先順位 #982](https://github.com/hat47x/tei-platform/issues/982)と[横断受入 #1119](https://github.com/hat47x/tei-platform/issues/1119)で追跡します。主ブランチの保護と必須チェックの設定も[課題 #1189](https://github.com/hat47x/tei-platform/issues/1189)で未完了です。

検証済みの実装と、まだ確定していない設計を区別して開発を進めます。優先順位は[TEIの全体計画](../plan/MASTER_PLAN.md)、技術上の境界は[全体構成](architecture.md)、業務設計の考え方は[業務システム設計](business-system-design.md)を参照してください。

## 最初に動かす

Goの開発環境で行う主なローカル確認は、次のとおりです。

```bash
go test ./...
go run ./cmd/tei validate -workspace reference/workspace
```

詳しい確認手順と参照用作業領域の範囲は[使い始める](getting-started.md)を参照してください。手順の掲載は、公開候補や最新のコミットが試験に合格したことを意味しません。
