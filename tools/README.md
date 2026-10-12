# 検証・参照用ツール

`tools/`には、構成検査、受入根拠の取得、参照実装、再現可能な実験に使う道具を置きます。製品の再利用可能な実装は`internal/`、入力と期待結果は`reference/`、利用例は`examples/`にあります。

表には、この階層の全ディレクトリと直下の実行用ファイルを載せています。各ツールの固有の使い方や判定条件は、そのディレクトリ内のREADME・実装・Issueを正とします。

独立した参照プラグインの検査入口は [`check-reference-plugin-isolation.sh`](check-reference-plugin-isolation.sh) です。`plugins/`配下の参照実装を一時Goモジュールへ複製してオフラインで検証します。

## リポジトリ検査と共通部品

| 場所 | 役割 |
| --- | --- |
| [`internal/`](internal/) | 複数のツールから使う共通処理 |
| [`repository-architecture-check/`](repository-architecture-check/) | パッケージ構成と依存境界の検査 |
| [`repository-dependency-report/`](repository-dependency-report/) | Goの直接依存関係の一覧作成 |
| [`repository-plan-check/`](repository-plan-check/) | ADR・Issueの番号や要求と検証表の対応確認 |
| [`repositoryqualification/`](repositoryqualification/) | リポジトリ検証条件の共通処理 |

## 設計、業務知識、受入れ

| 場所 | 役割 |
| --- | --- |
| [`first-value-ux-audit/`](first-value-ux-audit/) | 初回利用時の操作上のつまずきの確認 |
| [`reference-authoring-witness-openssh/`](reference-authoring-witness-openssh/) | OpenSSHを使った記述作業の参照確認 |
| [`reference-authority-evidence-trust/`](reference-authority-evidence-trust/) | 情報源の権威と検証根拠の信頼境界の確認 |
| [`reference-business-dfd/`](reference-business-dfd/) | 業務データフロー図の読取り専用参照 |
| [`reference-business-procedure-check/`](reference-business-procedure-check/) | 業務手順と担当・機能・データの整合確認 |
| [`reference-capability-invocation-context/`](reference-capability-invocation-context/) | Capability実行時の文脈境界の確認 |
| [`reference-capability-mutation-protocol/`](reference-capability-mutation-protocol/) | 変更を伴うCapabilityの手順確認 |
| [`reference-capability-transport/`](reference-capability-transport/) | Capabilityの伝送形式と境界の確認 |
| [`reference-chat-cli-host/`](reference-chat-cli-host/) | 信頼された開発用Hostの事前診断 |
| [`reference-critical-lowering/`](reference-critical-lowering/) | 意味定義から実行形式への変換条件の確認 |
| [`reference-epistemic-knowledge/`](reference-epistemic-knowledge/) | 文脈ごとの知識状態と根拠の確認 |
| [`reference-external-coordination/`](reference-external-coordination/) | 外部担当との引継ぎ・協調境界の確認 |
| [`reference-shared-acceptance/`](reference-shared-acceptance/) | 人・画面・AIに共通する変更受入れ処理の参照実装 |
| [`reference-standard-function-candidates/`](reference-standard-function-candidates/) | データ定義から実行前の標準機能候補と不足する契約を提示 |
| [`reference-spec-review/`](reference-spec-review/) | 仕様書の不足・曖昧さ・矛盾の検討支援 |
| [`reference-purpose-preflight/`](reference-purpose-preflight/) | 開発目的に対する業務規則の未確定・実行時値待ちを合成データで区別 |
| [`reference-tool-neutral-development/`](reference-tool-neutral-development/) | 特定の開発道具に依存しない作業経路の確認 |

## ビルド、配布、実行

| 場所 | 役割 |
| --- | --- |
| [`reference-build-artifact-store/`](reference-build-artifact-store/) | ビルド成果物の保管と識別 |
| [`reference-build-attempt-evidence/`](reference-build-attempt-evidence/) | ビルド失敗を含む実行記録の保管 |
| [`reference-build-execution-binding/`](reference-build-execution-binding/) | ビルド実行と主体・対象の結び付け |
| [`reference-builder-contract-resolution/`](reference-builder-contract-resolution/) | Builder契約の解決と適用 |
| [`reference-deployment-attempt/`](reference-deployment-attempt/) | 配備試行の参照用起動点 |
| [`reference-durable-run-golden/`](reference-durable-run-golden/) | 永続実行記録の期待出力の再現 |
| [`reference-lease-supervision/`](reference-lease-supervision/) | 実行権の貸出しと監督の確認 |
| [`reference-native-plugin-builder/`](reference-native-plugin-builder/) | ネイティブPluginの構築例 |
| [`reference-plugin-capability-contract/`](reference-plugin-capability-contract/) | Pluginが提供する機能の契約確認 |
| [`reference-plugin-package-builder/`](reference-plugin-package-builder/) | Plugin配布物の組立て例 |
| [`reference-plugin-self-build/`](reference-plugin-self-build/) | Plugin自身が構築手順を提供する場合の確認 |
| [`reference-process-plugin/`](reference-process-plugin/) | 別プロセスで動くPluginの参照用起動点 |
| [`reference-composition-process-bridge/`](reference-composition-process-bridge/) | 合成基盤と従来・独立プロセスプラグインの接続試験 |
| [`reference-resource/`](reference-resource/) | 実行資源の参照用起動点 |
| [`reference-runtime-capability-identity/`](reference-runtime-capability-identity/) | 実行時Capabilityの識別方法の確認 |
| [`reference-runtime-snapshot/`](reference-runtime-snapshot/) | 生成時と実行時の投影境界の確認 |
| [`reference-supervisor-takeover/`](reference-supervisor-takeover/) | Supervisor交代時の引継ぎ確認 |

## ウェブ実行先と判定根拠

| 場所 | 役割 |
| --- | --- |
| [`reference-first-web-target/`](reference-first-web-target/) | 最初のウェブ実行先の参照実装 |
| [`reference-first-web-target-architecture-evidence/`](reference-first-web-target-architecture-evidence/) | ウェブ実行先の構成根拠の確認 |
| [`reference-first-web-target-evidence-audit/`](reference-first-web-target-evidence-audit/) | ウェブ実行先の根拠一式の監査 |
| [`reference-first-web-target-gate-c-readiness-audit/`](reference-first-web-target-gate-c-readiness-audit/) | Gate C判定資料の検査 |
| [`reference-first-web-target-manifest-capture/`](reference-first-web-target-manifest-capture/) | 固定した投影結果の取得 |
| [`reference-first-web-target-r7a-handoff/`](reference-first-web-target-r7a-handoff/) | R7A引継ぎ根拠の検査 |
| [`reference-first-web-target-r7b-intake/`](reference-first-web-target-r7b-intake/) | R7B入力根拠の確認 |
| [`reference-first-web-target-r7b-qualification-receipt-audit/`](reference-first-web-target-r7b-qualification-receipt-audit/) | R7B検証済み記録の検査 |
| [`reference-first-web-target-r7b-replay-plan/`](reference-first-web-target-r7b-replay-plan/) | R7B再生手順の検証 |
| [`reference-first-web-target-visual-evidence-audit/`](reference-first-web-target-visual-evidence-audit/) | 表示根拠の監査 |
| [`reference-first-web-target-visual-replay/`](reference-first-web-target-visual-replay/) | 表示検証の再生処理 |
| [`reference-frozen-manifest-target/`](reference-frozen-manifest-target/) | 固定した投影結果を使う最小起動例 |
| [`reference-gate-b-final-evidence/`](reference-gate-b-final-evidence/) | Gate Bの最終根拠の収集・照合 |
| [`reference-gate-b-readiness-artifact-audit/`](reference-gate-b-readiness-artifact-audit/) | Gate B判定用成果物の監査 |
| [`reference-gate-b-repository-evidence/`](reference-gate-b-repository-evidence/) | Gate Bのリポジトリ根拠の確認 |
| [`reference-generated-artifact-evidence/`](reference-generated-artifact-evidence/) | 生成成果物の再現性根拠の確認 |
| [`reference-golden/`](reference-golden/) | 参照用の期待出力の生成 |
| [`reference-target-package-evidence/`](reference-target-package-evidence/) | 実行先パッケージの構成根拠の確認 |
| [`reference-critical-path-evidence-audit/`](reference-critical-path-evidence-audit/) | クリティカルパスの根拠一式の事前検査 |

## データベースと外部接続

| 場所 | 役割 |
| --- | --- |
| [`reference-driver-postgres-query/`](reference-driver-postgres-query/) | DriverからPostgreSQL問合せを実行する参照用起動点 |
| [`reference-external-query-evidence/`](reference-external-query-evidence/) | 外部問合せの実行根拠を確認 |
| [`reference-postgres-header-detail-runtime/`](reference-postgres-header-detail-runtime/) | PostgreSQL上の親子明細処理の参照実装 |
| [`reference-postgres-integration/`](reference-postgres-integration/) | PostgreSQL接続の統合確認 |
| [`reference-postgres-migration-rendering/`](reference-postgres-migration-rendering/) | PostgreSQL向け移行SQLの生成確認 |
| [`reference-postgres-query-runtime/`](reference-postgres-query-runtime/) | PostgreSQL問合せの実行時処理 |
| [`sql-analyzer/`](sql-analyzer/) | SQL構造をGo側へ渡す解析補助処理 |

## 形式・移行・分野別の参照

| 場所 | 役割 |
| --- | --- |
| [`reference-example-slide/`](reference-example-slide/) | 例から関係図を組み立てる参照資料 |
| [`reference-semantic-extension-catalog/`](reference-semantic-extension-catalog/) | 意味拡張の登録・解決・検証 |
| [`reference-subsystem-cud/`](reference-subsystem-cud/) | 部分システムの登録・参照・更新・削除の境界確認 |
| [`reference-sui-action-boundary/`](reference-sui-action-boundary/) | SUIの操作とTEI Actionの接続例 |
| [`reference-sui-adoption/`](reference-sui-adoption/) | SUI移行の参照データと検査 |
| [`reference-sui-interaction/`](reference-sui-interaction/) | 画面操作とApplication Actionの接続例 |
| [`reference-transform-runtime-engine/`](reference-transform-runtime-engine/) | 決定的な変換処理の実行例 |
| [`reference-transform-runtime-engine-build/`](reference-transform-runtime-engine-build/) | 変換実行基盤の構築確認 |
| [`reference-transform-seed-builder/`](reference-transform-seed-builder/) | 変換用の初期データを組み立てる処理 |
| [`reference-xead-cross-view/`](reference-xead-cross-view/) | Xeadの複数表示間の意味照合 |
| [`reference-xead-source-adapter/`](reference-xead-source-adapter/) | Xead資産を読むSource接続例 |
| [`reference-xead-source-inspect/`](reference-xead-source-inspect/) | Xead資産の読取り・調査補助 |
| [`sei-resume-reference/`](sei-resume-reference/) | SEI参照の再開条件と根拠の確認 |

## 直下の実行用ファイル

| ファイル | 役割 |
| --- | --- |
| [`ci-workflows-test.py`](ci-workflows-test.py) | 手動CIの版ずれ・作業領域検査を、隔離したGit環境で再現する回帰試験（`python3 tools/ci-workflows-test.py`）。本物のCIやTEI全体検証は実行しない |
| [`verify-critical-path-e1-archive.py`](verify-critical-path-e1-archive.py) | E1合成証拠の8件原本と固定SHA目録、R0/R1/R2・未資格状態を照合する（製品Gateの検査ではない） |
| [`test-critical-path-e1-archive.py`](test-critical-path-e1-archive.py) | E1保全検査の改ざん・欠落・偽資格の回帰試験 |
| [`check-core-isolation.sh`](check-core-isolation.sh) | 合成基盤の隔離試験と別モジュールからの公開API試験 |
| [`repository-health-check.sh`](repository-health-check.sh) | 形式、静的検査、全Goテスト、参照Workspaceの検証 |
| [`run-business-procedure-gate-a.sh`](run-business-procedure-gate-a.sh) | 業務手順Gate Aの検証 |
| [`run-first-web-target-gate-c-evidence.sh`](run-first-web-target-gate-c-evidence.sh) | Gate C用のウェブ実行根拠の取得 |
| [`run-first-web-target-r7a-handoff.sh`](run-first-web-target-r7a-handoff.sh) | R7A引継ぎ根拠の取得 |
| [`run-first-web-target-r7b-intake.sh`](run-first-web-target-r7b-intake.sh) | R7B入力根拠の取得 |
| [`run-first-web-target-r7b-local-qualification.sh`](run-first-web-target-r7b-local-qualification.sh) | R7Bのローカル検証 |
| [`run-first-web-target-r7b-replay-plan.sh`](run-first-web-target-r7b-replay-plan.sh) | R7Bの再生計画検証 |
| [`run-knowledge-portability-reference-evidence.sh`](run-knowledge-portability-reference-evidence.sh) | 知識移植の参照根拠の取得 |
| [`run-repository-local-qualification.sh`](run-repository-local-qualification.sh) | リポジトリのローカル検証 |
| [`verify-cli-c1-local.sh`](verify-cli-c1-local.sh) | CLIのC1検証 |
| [`verify-first-web-target-r7b-integration-receipt.sh`](verify-first-web-target-r7b-integration-receipt.sh) | R7B統合記録の照合 |
| [`verify-first-web-target-r7b-post-merge.sh`](verify-first-web-target-r7b-post-merge.sh) | R7B統合後の状態検査 |
| [`business-procedure-gate-a-critical-tests.txt`](business-procedure-gate-a-critical-tests.txt) | Gate Aの確認対象となる重要テストの一覧 |

R7Bの一回限りの準備・公開・後処理を行うスクリプトは、統合と対象ブランチの終了後に削除しました。現在の根拠確認や再現に使う検査・取得ツールは上の一覧に残しています。
