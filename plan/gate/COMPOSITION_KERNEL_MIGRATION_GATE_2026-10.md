# 合成Kernelの移行判定

- 管理課題：[#1223](https://github.com/hat47x/tei-platform/issues/1223)。
- 判断の正本：[ADR-0032](../adr/0032-composition-kernel-replacement.md)。本書は移行段階M1〜M6の確認手順を定める。製品要求、公開API、互換形式を追加しない。
- この文書は受入条件である。判定結果は対象コミットと根拠を固定してIssueまたはPRに記録し、現在の製品状況は[実装状況](../../docs/status.md)へ反映する。

## G1：Kernelと構成案のReference

対象はHost局所のサービス選択、依存宣言の鮮度、世代保持、一時的な作用の解放である。構成案を実行許可として使わない。

| 確認する条件 | 既存の確認先 | 合格に必要な結果 |
| --- | --- | --- |
| 契約と利用範囲の一致、一意なProvider、循環の拒否 | `internal/composition/registry_test.go`、`selection_test.go` | 登録順で競合を解消せず、別ScopeのProviderを利用しない |
| 推移的な依存宣言の変更と候補の鮮度 | `selection_closure_test.go`、`proposal_freshness_test.go` | 下位Providerの変更を検出し、無関係なScopeの変更だけでは失効させない |
| 複数の要求を一つの登録簿の状態から調べる | `proposal_test.go` | 混在Scope、競合する接続、未知の候補を保持し、構造上の候補を適合済みへ昇格しない |
| 準備・Ready・権限・世代CASの失敗 | `coordinator_test.go`、`selection_test.go` | 新世代を公開せず、候補Scopeを回収する。緊急撤去を長い準備や解放で妨げない |
| 処理中の旧世代と後始末 | `composition_test.go`、`coordinator_vertical_test.go` | 呼出しの世代を保持し、終了待ちの取消・解放失敗を成功へ丸めない。逆作用は逆順に一度だけ実行する |
| 外部の適合証拠との責務分離 | `reference/composition-evidence-handoff/`、`tools/reference-plugin-capability-contract/composition_handoff_test.go` | 既存の契約解決器へ委譲し、欠落・不一致・古い宣言・重複を診断する。参照の一致だけで真正性や実行権限を付与しない |
| 同一成果物とApplication別の利用可否 | `cross_application_reuse_test.go` | 片方の拒否と成果物全体の失効を区別し、取得済みLeaseの束縛を改変しない |

信頼できる完全なcheckoutで、対象headと対象mainを固定して次を実行する。

```bash
git rev-parse HEAD
git rev-parse HEAD:tools/repository-health-check.sh
git status --porcelain=v1 --untracked-files=all --ignored
go version
go test -race -count=1 ./internal/composition ./reference/composition-evidence-handoff
go test -count=1 ./tools/reference-plugin-capability-contract
bash tools/check-core-isolation.sh
bash tools/repository-health-check.sh
git status --porcelain=v1 --untracked-files=all --ignored
git rev-parse HEAD
```

作業領域とindex、未追跡・無視対象のファイルは実行前後ともcleanであり、headが一致することを要求する。失敗のある記録をPASSとして使わない。対象headまたはmainが動いた場合は、現行の統合規則に従って資格を取り直す。Draft PRは統合しない。

Kernelの独立性は、`check-core-isolation.sh`が同じソースを別モジュールへコピーし、外部Consumerから実行することでも確認する。試験用に型を作り直したモジュールでは代用しない。

## G2：実際のProcess成果物との接続

`go test -count=1 ./tools/reference-composition-process-bridge`を同じcheckoutで実行する。

- 独立構築した子プロセスの応答と実ファイルのSHA-256を確認する。
- `real_artifact_switch_test.go`で異なる成果物への経路切替、旧Leaseの保持、新規呼出しの旧経路拒否、終了後の解放を確認する。
- `artifact_rotation_test.go`でReadyに失敗した候補だけを回収し、旧成果物が引き続き呼び出せることを確認する。
- Linuxの固定実行試験で、検証した内容と起動した内容、署名の照合、鍵の差し替え・失効を確認する。実行しなかったOS固有試験は未実施として記録する。
- 契約の形、現在のHost権限、成果物検証が拒否した場合は、子プロセスの起動が増えないことを確認する。

このReferenceは一回の呼出しごとに子プロセスを起動する。旧Leaseを保持した経路切替の成功から、常駐サービスの無停止更新や業務処理の永続的な完了を主張しない。実成果物の構築・起動と、合成したHost認可や適合証拠を記録上も区別する。

## G3：製品Consumerの移行と無停止更新

M2〜M4を製品の成立として判定する前に、次の根拠を要求する。

1. 一つの実Consumerについて、契約、Provider、成果物、Target、現在権限、Invocationを同じ受理された経路へ結ぶ。旧Resolverや固定Kindを経由する暫定接続と、置換が完了した経路を区別する。
2. Contract発行者、Provider適合証拠、成果物の真正性、Hostの現在権限・Lease・fencingを、それぞれの所有者で照合する。Scopeや宣言Epochで代用しない。
3. v1を使う二Consumerへ継続的な要求を送り、別ProcessまたはWasm instanceのv2を準備する。実際のReadyと併走する資源の余裕を確認してから新規要求を切り替える。
4. 処理中の要求は開始時の世代で完了させ、旧世代をdrainしてから解放する。Prepare・Ready・権限・容量・世代競合・Provider故障・drain timeout・cleanup失敗を別々に注入して結果を残す。
5. 利用者から見た成功率、欠落・重複要求、P99遅延、旧世代の完了、資源の解放、失敗時の復帰時間を計測する。試験前に運用者が定めたSLOと照合し、測定後に基準を緩めない。
6. 永続状態が旧版へ戻せない場合はrollbackを実行せず、明示された移行・補償・復旧手順を要求する。Effect Scopeの解放をDB・決済・メール等の取消として扱わない。

この判定には対象の配備形態、権限の発行元、資源制限、業務状態の移行計画、SLOが必要である。これらが未指定ならG1/G2の成功を使って合格にしない。

## G4：Target別の縮退と旧形式の撤去

| Target | 判定する更新単位 | 必要な確認 |
| --- | --- | --- |
| 動的Server・Desktop Host | 独立Process／サービスinstance | G3の併走・世代切替・復旧と、実配備先の権限を確認する |
| 長時間稼働するWasm Host | Wasm instance | Host ABI、共有状態、資源制限を含めてG3を実行する |
| 静的Go・単一Wasm | 実行ファイル／モジュール | 同じ論理契約から静的に具体化し、依存一覧と実成果物で動的Loader・Watcherが不要になったことを確認する。サイズ、起動時間、RAMも測る |
| 単一IoT機器 | Firmware slot／再起動 | A/B配備と復旧を確認する。再起動の停止時間を測り、無停止として判定しない |
| 冗長IoT・分散配置 | Replica／Device group | Host別の世代、段階配備、Lease・fencing、サービス全体のSLOを確認する。Host局所CASで全Node同時切替を主張しない |

M6の撤去判定では、移行済みConsumerの一覧、旧形式の読取り・書込み・解決経路の残存一覧、移行前後の回帰結果をそろえる。互換入口を残す場合は一方向の変換と撤去条件を定め、二つの正本やAuthorityを恒久化しない。

## 判定記録に残す情報

対象head、対象main、Health runnerのGit blob、GoとOS・Target、実行コマンドと終了状態、前後のclean状態、ログの保存先とSHA-256を記録する。G2以降では実成果物の指紋、世代とProviderの対応、合成した入力・権限の範囲、実施していない確認も記す。性能値には測定環境、反復数、負荷を添え、比較対象が無い測定から改善率を算出しない。

G1、G2、G3、G4の結果を別々に判定する。#1223はADR-0032の破壊的移行と無停止更新も担当するため、G1の合格や一つのPRの統合だけでIssue全体を完了にしない。
