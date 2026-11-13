# main健全性とCI Evidenceの横断運用計画（2026-11-12）

- **状態**：既存のリポジトリ運用・製品品質・テスト基盤へ接続する実行計画案。新しい正式Gate、認可主体、中央Registry、常駐CI監視サービスを設ける決定ではない。
- **対象**：TEIを最初の実証先とし、SUI・SHO・TAI・MMI・SEI・EKI・CSW・VSクマ等へ適用条件を確認する。各リポジトリのAGENTS.md、workflow、費用制約、既存Ownerを優先する。
- **担当先の既存契約**：製品品質は[品質設計](../../docs/product-quality-architecture.md)、自律統合は[AGENTS.md](../../AGENTS.md)／[ADR-0035](../adr/0035-autonomy-levels-and-verification-packets.md)、独立Expectedは[#2514](https://github.com/hat47x/tei-platform/issues/2514)、増分再利用は[#2600](https://github.com/hat47x/tei-platform/issues/2600)、生産性は[#2018](https://github.com/hat47x/tei-platform/issues/2018)、外部Harnessの権限とEvidenceは[#2515](https://github.com/hat47x/tei-platform/issues/2515)を参照する。本書を第二の正本にしない。

## 1. 目的と避ける誤判定

「過去の赤を消す」よりも「**現在のmainの同一HEADについて、必要な検査が実行され、観測結果が説明できる**」ことを優先する。失敗履歴を削除・隠蔽することで緑にしない。逆に、旧SHAの失敗を現mainの失敗として繰り返し修正しない。

CIはアプリケーションの業務意味を受理する場ではない。対象SHAでの静的検証、ユニット／統合試験、Windows・WebView2・実DB等の実環境試験、ビルド、成果物の実使用はそれぞれ別の主張であり、ある一つのsuccessを他の主張へ流用しない。

## 2. 検証状態を混同しない

| 判定 | 判定に必要な観測 | 行動 |
| --- | --- | --- |
| **current-main-pass** | 現在のmain HEADに一致するrun SHAで、必要なworkflow・jobがsuccess | 対象scope内で正常。必要時のみ次段の検証 |
| **current-main-fail** | 同じHEADの必要jobがfailure/timed_out等 | 失敗stepとログを取得して実装・環境・契約の原因を分ける |
| **pending / unknown** | queued/in_progress、結果欠落、SHA不一致、必要runなし | PASSと呼ばず、対象runを再照合 |
| **not-configured** | 該当workflow自体がなく実行記録もない | 「エラーなし」「品質確認済み」と呼ばない。必要性を別に判断 |
| **intentionally-disabled** | workflowやjobの明示的停止とその理由がある | 解除条件を確認。履歴の失敗を現在の失敗と誤認しない |
| **historical-failure** | 失敗runのhead SHAが現在mainでなく、同種検証の最新mainで成功 | 修正不要。履歴を保持し、未解決Issueを事実に基づき整理 |
| **not-applicable** | 対象機能・環境に適用されない根拠を明示 | skipしただけの試験を自動的にnot-applicableへ昇格しない |

「緑」を判定するときは *repo + branch + exact HEAD + workflow/job + run attempt + required scope + environment/toolchain + conclusion* を最低限区別する。古い成功、PRのhead成功、CIセットアップ成功、テストのskip、手動実行だけを現mainの合格へ昇格しない。該当workflowのpaths filterやscheduleが対象変更で起動しない場合は「runなし」と記録する。

## 3. 失敗原因の分類と最初の対処

| 分類 | 例 | 対処方針 |
| --- | --- | --- |
| 製品・テストの回帰 | コンパイル失敗、独立Expected不一致、実WebView/DB試験の失敗 | exact SHAで再現し、最小修正＋負例テストをPRにする |
| Workflow・索引契約の不整合 | 無効なAction SHA、Go版と例外注記の食い違い、README未登録 | セキュリティ基準や実検証を落とさず契約を修正 |
| Runner／外部依存 | network/proxy、ダウンロード障害、host差、locale、供給元の一時不調 | 真因をログで区別し、再試行時はattemptを新しいEvidenceとして保存 |
| セキュリティ | govulncheckで標準ライブラリまたは依存に到達可能な脆弱性 | ツールチェーン・依存を更新し、検出そのものを無効化しない |
| 並行PR由来の不整合 | 旧base、他PRと同じファイルの競合、既にmainで修正済み | main現HEADと差分を比較し、重複PRは理由を残して整理 |
| 意図的停止・無構成 | 費用枠のため停止したGodot CI、配布用テキストのみのrepo | 意図と制約を保持し、無断でCIを有効化・追加しない |

**再実行は修正ではない**。一時障害であることをログから説明できる場合に限定してrerunし、同一SHAの新attemptとして記録する。実装不具合を再実行やallow-failureで隠さない。

## 4. PRからmainへ：既存統合規則を保った最短手順

1. GitHubの現main HEADと失敗runのSHA、workflowの起動条件・scopeを取得し、実行時間の新旧順ではなく**対象SHAの一致**から開始する。
2. 失敗jobの最後だけでなく、最初に失敗したstep、原因行、外部依存、追加の並列jobを確認する。必要なテストのscopeと独立Expectedを確認する。
3. 既存Issue・修正PR・並行ブランチとdiffを比較し、重複修正やmainへの直書きを避ける。
4. 狭いブランチで変更し、まず変更箇所の自動検査、境界の負例、静的検査を選ぶ。全スイートは変更の影響・既存規則・実行費用によって選ぶ（[AGENTS.md](../../AGENTS.md)の原則）。
5. PRで **検査したexact head／未実行／既知のmain由来失敗／機能上の残余** を列挙する。外部ハーネスから得たpatchや検査報告はGit差分と実runで独立確認する。
6. マージ直前にhead SHAとmainのbase SHAを再確認し、可能ならexpected headを指定して統合する。競合時に古いbaseや他者の変更をforceで上書きしない。
7. マージ後、**統合commit** のmainで必要workflowの実行を追跡する。successしたscopeだけを確認済みとし、未起動・未完了は保留。後始末では冗長PRと解消済みIssueを説明付きで閉じる。

CIの一括successだけを「実Provider、実Consumer、Host効果、Ownerの業務受理、製品全体完成」に昇格しない。ビルド／Artifact／実Use／許可／業務Effectの資格は既存担当へ残す。

## 5. テストの階層と横断実証

| 層 | 例 | 位置づけ |
| --- | --- | --- |
| T0: 差分局所 | formatter、Workflow静的規約、変更対象の単体・契約試験 | 普段の開発の最短反復。偽陽性を減らす |
| T1: リポジトリ回帰 | 主入口のbuild/test、CLI、実行可能なself-build | 変更影響に応じてPRで検証 |
| T2: 実環境／独立Expected | Windows、WebView2、DB Providerの実Read/Write、CLI/Web共通Scenario | 同一fixture・同一expected・実Observedを分ける。合成PASSを実接続へ流用しない |
| T3: 複数製品・Harness | SEI/SUI/TEI/SHO/TAI・外部Harness経由の実Tool呼出しとConsumer受領 | 選択／読込み／呼出し／受領／権限を別票にし、ハーネスの自己申告を実効果とみなさない |

層を常に全部走らせるのではなく、依存追跡で必要scopeを判断し、十分条件を説明できない場合は必要な上位検証を選ぶ。運用費用を守るためのskipは可能だが、未観測のscopeは未観測と表示する。拡張は既存テストDriver／Pluginの範囲で進め、新しいCI専用エンジンを先行開発しない。

## 6. Evidenceの保持・再資格化

- Gitには小さな**追跡情報**（実行の対象SHA、workflow/run/attempt/job、fixtureと独立Expectedの識別、toolchain/runner、ログ・成果物のdigest、保存先と取得状態、受入Claimの上限）を保持する候補とし、大容量のログ・動画・スクリーンショット・バイナリは必要に応じてNAS/S3等の外部Storeに置く。全EvidenceをGitHubへアップロードすることを前提としない。
- Git・NAS・S3・GitHub Actions Artifactのどれを使う場合も、byte digest・retention/expiry・参照可否・権限・Revisionを区別する。URLが残っているだけでは実体の存在や現在資格を証明しない。
- *artifact再利用／build省略／Evidence再利用* は別判断。Source変更、Toolchain/Plugin更新、Required依存の追加・削除、A→B→A、権限変更、部分出力、取得不能を負例として選択的再資格化を検証する。
- 既存[#2600](https://github.com/hat47x/tei-platform/issues/2600)のFull/Selective比較を使い、必要な結果と独立Expectedが維持されることを確認してから削減時間・費用を測る。GitHub成功表示と外部Evidence取得可否は別の判定。
- 秘密・顧客情報・非公開原典・認証情報はGit、Issue、PR、CIログ、公開Artifactへ入れない。公開可能な合成fixtureで始める。

## 7. 2026-11-12時点の各リポジトリの観測と判断

以下は**この時点の観測**であり、将来の現在値や製品全体の合格を保証しない。

| 対象 | 観測 | 判断／次の仕事 |
| --- | --- | --- |
| TEI | main `0a4d3e3132cd` の[repository-health](https://github.com/hat47x/tei-platform/actions/runs/43139891553)、[cache](https://github.com/hat47x/tei-platform/actions/runs/43139891603)、[脆弱性](https://github.com/hat47x/tei-platform/actions/runs/43139891484) はsuccess。2026-11-11のPR [#4459](https://github.com/hat47x/tei-platform/pull/4459)はmainへ統合済み | 当該検証は復旧。旧runや重複PRを新規修正の理由にしない |
| SUI | main `5d2348921dbd` の[定期e2e](https://github.com/hat47x/sui-sensemaking/actions/runs/43130646502)と[backend](https://github.com/hat47x/sui-sensemaking/actions/runs/43127557269) がsuccess | 現行scopeは正常 |
| TAI / SHO / MMI | それぞれ最新mainの[tai-reference](https://github.com/hat47x/tai-pleroma/actions/runs/43139882466)、[SHO G2](https://github.com/hat47x/sho-composer/actions/runs/43091596693)、[MMI CI](https://github.com/hat47x/markdown-matrix-injector/actions/runs/42817866434) がsuccess | 対象テストの合格として保持 |
| CSW | 2026-11-01の[Dependabot失敗](https://github.com/hat47x/cultural-substrate-weaving/actions/runs/41265714063)は旧SHA。現mainは旧CI/CDを撤去し、`.github/dependabot.yml` もない | 過去エラーを現在のmain回帰と認定しない。更新監視の再導入は別判断 |
| VSクマ | 2026-09-22の[CI失敗](https://github.com/hat47x/vs-bear-simulator/actions/runs/35679790202)は旧SHA。現mainのCIはActions費用枠を理由にmanual-only＋job `if: false`で停止 | **intentionally-disabled**。緑とは呼ばず、既存ローカル検証のexact-ref記録を維持。費用・必要性・再開条件を確認せずCIを有効化しない |
| SEI / EKI / KON等 | SEI・EKIは最近のmainに対応するCI実行がなく、KON等の一部はrunなし | 不合格でも合格でもない **not-observed/not-configured** として扱う |

## 8. 実装に渡す作業順と完了条件

**P0（今すぐ・追加基盤なし）**：各Repoの既存CIをexact-main SHA基準で分類し、失敗ログ・Scope・PR重複を比較する運用を確立する。TEIで現在の合格を再確認し、旧エラーを現mainの不具合として扱わない。完了条件は「少なくともTEIのmainの必須scopeについて根拠付き判定ができ、pending/disabled/not-configuredを成功に含めない」こと。

**P1（再発検出）**：無効Action SHA、Go版の安全下限と説明付き例外、Workflow索引、WindowsのUTF-8復号、PR headとmerge SHAの食い違いを合成負例で確かめる。最新mainで既に修正されたものは新実装せず、既存テストの成立範囲を確認して足りない反例だけ追加する。完了条件は「誤って緑になる負例と誤って赤になる負例を一つずつ説明できる」こと。

**P2（証拠の取り回し）**：実行根拠の軽量な追跡と、大容量EvidenceのNAS/S3等への退避・復元を、既存Artifact/Providerの枠で1本だけ実証する。外部Store未取得、digest不一致、URL失効はunknown/failを保ち、CIを緑にするための証拠欠落容認をしない。

**P3（横断品質と費用）**：SUI/SHO/TEIの同一ScenarioでCLI/Web/Desktop/Consumer受領の実証範囲を比較し、外部HarnessのOS隔離・送出・実Tool結果を個別観測する。VSクマの再開は停止理由・利用枠・ローカル検証と実Actionsの差を評価して別判断とする。CI実行費用と誤判定率、変更から利用可能な結果までの待ち時間を反復測定し、必要な変更だけ既存IssueのOwnerへ戻す。

本書の記述は**実施済み／計画中／未検証を分けるための整理**であり、新しい統合条件の自動強制、全リポジトリへのCI新設、Ownerの意味受理、Evidence保存方式の採用を宣言しない。
