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

作業領域とindex、未追跡・無視対象のファイルは実行前後ともcleanであり、headが一致することを要求する。失敗のある記録をPASSとして使わない。対象headまたはmainが動いた場合は、現行の統合規則に従って資格を取り直す。Draftは作業中の表示であり、解除して統合する。

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

## 横断追補（2026-11-02）：外部Hostと製品Consumerの適用条件

**状態：G3/G4の受入計画を具体化する検討差分。実装・業務意味の受理・新しいGateの採用ではない。** 対象は#1223、#951、#2390、#259、#1119。TEI独自の常駐Host、中央Registry、第二Contract Resolver、第二Authority、統一Plugin ABIを新設しない。外部Harness（DSH等）は開発・観測を担うConsumer候補であり、TEIの業務意味や実行権限の発行元ではない。

### 1. G3/G4が適用される実行形態を最初に分ける

| Consumerと配備形態 | 必須の検証 | 適用しない一般化 |
| --- | --- | --- |
| **動的な常駐Service**（既存Host内のProcess/Wasm） | G3：実Providerのv1/v2併走、二Consumer、Ready、切替、旧in-flight、容量、Drain、故障とSLOを実測 | 一つの子Processの起動成功を無停止性としない |
| **静的Library/.NET assembly/Java JAR/単一Wasm** | G4：受理済み意味からのTarget別生成、必要な依存閉包、実リンク成果物と独立Consumerでの適合を確認 | 常駐Coordinator・世代Router・動的Loaderの導入を合格条件にしない |
| **単独のネイティブ実行ファイル** | G4：入口、実際の組込み機能、実OS上の効果、再構築/交換・停止手順を計測 | 動的Plugin機構を要求しない |
| **外部Harness／headless CUI／MCP薄いAdapter** | #2390/#1118：操作に必要な意味、Scope、公開可能な情報、Tool/Providerの現在の許可を既存Host/OSで確認 | Tool一覧、MCP呼出し成功、AIの判断をContract適合・権限・業務完了としない |

同じ論理Service Contractを複数方式で実現してよいが、Java/.NET等で外部Hostが利用する生成物にTEIのGo製`internal/servicehost`やCLIを必須依存として持ち込まない。各HostのDI、認証・認可、Transaction、OS隔離、ライフサイクルに必要な薄いBindingだけを接続する。**G3は動的配備の継続運用資格、G4はTarget/静的縮退・移植資格であり、一方の不合格から別方式の不可能性を推定しない。** ADR-0032/0033の当時の設計判断は履歴として保持し、独自Host非必須の文書整理は[#2226](https://github.com/hat47x/tei-platform/pull/2226)でmainへ反映済み。ただし実Java/.NET/Wasm Hostとの配布・接続が検証済みになったわけではない。

### 2. 「接続済み」の主張を段階別に固定する

| 辺／時点 | 現在の確認に利用できる既存実装・証拠 | 製品受入までに必要な別証拠 |
| --- | --- | --- |
| Web Action → Canonical認可 | `reference/web-composition/go-host/canonical_generation_service_handoff_test.go`（PR #1760/#2275） | 本物のPrincipal/SubjectLink/Policy、正規のOperationと業務意味のOwner受理 |
| 選択 → Generation Lease → Host current Route | `composition.NewSelectedCoordinator`、`servicehost.NewSelectedServiceCarrierInvoker`、`BindHostApplicationEnvironmentRoute`（#1736/#1785/#2275） | 正式なApplication/Environmentの利用資格と現在Deployment revisionを外部の正本から取得 |
| Route → Service current Acceptance → Runtime | `pruntime.Registry.InvokeSelectedServiceCarrier`と`ServiceAcceptanceReader` | Publisher/Providerの現在受理・失効を持つ信頼済みSource、実Artifact/実起動対象byteとの束縛 |
| Runtime → 結果 | 合成ProcessとRuntime provenance | 同じContractのrequest/result/failure意味を独立検証基準で検証し、0件・NULL・欠損・部分取得・Snapshotと不確実性を区別 |
| 書込み要求 → Effect確定と再照会 | `internal/security/effectcommit.Runner`、`driver_binding_reference_test.go`（PR #2273） | 正式CallerとGrant、同一DBのAuthority/Resource/Receipt、Commit ACK喪失・再起動・同じOperationID再試行・独立観測 |

第2・第3行の`Host`は**既存の実行環境が所有する呼出し責務**であり、TEI独自の製品Hostを追加する指定ではない。PR #2275は`currentRoute`の取込み口とEnvironment不一致の拒否を合成試験で示すが、`currentRoute`の真正なProviderや正式な業務DBを実証しない。PR #2273のDriver→Runner試験もmemory Providerと合成Grantであり、結果の形が適合しなくても既にCommitが成立し得る負例を含む。**CI/merge、byte digest、Provider宣言、Ownerの意味受理、実行許可、DB最終効果を一つの「成功」に畳まない。**

### 3. 相互に独立な識別子・時点・証拠を残す

1. **意味**：Data/Function/BusinessのAccepted modelとService Contractを正本とする。仕様候補、生成AIによる提案、SHO設計書、生成コードは意味の受理ではない。
2. **実現**：論理Provider/物理Plugin/Implementation/Artifact/TargetとApplication/Environment Routeを分離する。BuildPlan fingerprint、Artifact digest、選抜receiptは現在Authorityや実行許可ではない（ADR-0041）。
3. **Invocation**：Generation Leaseは実行中の実装束縛、RequestIDは呼出し追跡、OperationIDは永続的な副作用再試行の同一性である。Lease解放、Requestの応答、Effect receipt、独立した最終Observationは別証拠とする。
4. **時間・対象**：現在Provider受理、Host Permission、入力集合/read-set、対象Resource version、Commit fenceはそれぞれ必要な時点で検査する。v1へのpinから、失効後の永続書込み許可を導かない（#1174/#1708）。
5. **Consumer/開示**：同じ型・値・JSONでもCollectionの被覆範囲、粒度、時刻、失敗・NULL/空集合の意味が異なれば合成適合を保留する。通常の閉じたQueryで0件は正当な結果であり、既知の部分取得・全件主張がある場合にのみ追加の被覆根拠を要求する（#982/#2390）。外部AI/Toolへ非公開の生データや内部Authority診断を渡さない。

### 3補. Contract単体適合からConsumer全体適合への接続条件（設計・反例計画）

**合成したProvider A/Bが個々にPASSしても、Consumerの業務操作・入力集合・間接権限・結果確定が全体としてPASSするとは限らない。** #2390の外部Consumer合成適合、#2485のOperation導出、#2598のIntent→Request、#2611のPersistence Alias、#2514のObservation十分性を参照する。本節はG3/G4の**既存の判定欄に必要な反証観点**を追加するもので、共通Contract/Result型や新しい認証Engineを制定しない。

一つの要求が通過する境界を、少なくとも次の**異なる閉包**として監査する。

| 閉包 | 受理する事実と証拠 | 既存Owner |
| --- | --- | --- |
| **意味／Operation** | Accepted Data/Function/Businessとその適用Context、導出されたOperationのRule/Model版、業務Actionの明示Binding、各ProviderのContract/依存・集合被覆条件 | #1127/#2485/#2506/#2507/#951 |
| **配置／Load** | どのTarget/外部Hostが何のartifact/member/class/importを実際にロードし、どの間接呼出しが到達可能か。byte指紋と実Load setを別欄で保持 | #943/#1370/#2390/#2512/#2513 |
| **Authority／Invocation** | 外部Hostが固定したApp/Env、呼出し毎の現在Route・Provider受理、委任主体とResource Scope、Generation/Lease、Commit直前の権限 | #951/#1118/#1223/#1174 |
| **Effect／Consumer Claim** | exact送信Request、OperationID、Receipt、返されたBusinessResult、再照会成否、独立最終観測、利用者/AI/SHOへ開示できる範囲 | #1174/#2598/#2514/#2515 |

以下は**既存E0〜E4へ配賦する最小負例**であり、テストを本書へ複製しない。全ての受入は対象Consumer・Source版・Contract・Scope・Target/Host・時点・独立検証基準を指定して記録する。

| 負例／担当工程 | 比較する条件と期待する停止・限定結果 | 実証Owner／現在の位置づけ |
| --- | --- | --- |
| **CB-1（E0→E1）Operationの静かな意味変更** | 同じOperation ID/Basisでも、R1の`submit`からR2で`cancel`へ候補Bindingが変わり得る。旧CUI/UI/AI呼出しはOperation IDだけで実行せず、Rule/Model版・意味Binding・利用資格を再照合。複数DB Aliasが同じSemantic Dataを指す場合も、別Persistence候補を無音で除外しない | #2485 W7-B6〜8／#2611 PM-01〜07。PR #2587は2026-11-02にmain統合済み。`internal/operations/surface.go`の実際の候補選択・重複省略に基づく静的反例であり、複数候補の再現試験・Owner意味受理は別途未取得 |
| **CB-2（E2）推移的Providerの局所PASSと集合不足** | Consumer→Provider A→Provider BでA/Bの局所ContractがPASSしても、Bが対象行を部分取得・古いSnapshot・別Scopeで取得したらAの「全対象」Claimを成立させない。通常Queryの正当な0件を一律エラーにせず、全件主張または既知の欠落がある箇所で被覆証拠を要求。異なる原状態が同じProjectionへ潰れClaimだけ変わる識別不能対も比較 | #2390 T1〜T4／#2514 D-01〜04。異質Consumerの実E2Eは未実施 |
| **CB-3（E4）成果物digest一致と実Load集合の不足** | Javaの`Main.class`だけを保存し、必要な`Main$Input.class`を配らなければ外部JVMでLoad失敗し得る。全class/依存の配布・実Loadと公開型付きAPI、TEI CLI/Go RuntimeなしのConsumer実行を別観測 | #2512のArtifactSet、#2513の型付きJava C1。#2077の独立再構成Java実験を本PRの生成Library製品受入へ流用しない |
| **CB-4（E3）OperationIDとRequest byte/意味の相違** | 同じOperationIDでも空白・キー順だけ異なるJSON byteは現`RequestHash`上で別Requestとなり得る。逆に異なる厳密値が途中のfloat64変換で同一byteに潰れた場合、Hashは元Intentの差を検出できない。初回のexact送信Requestと型付きIntentの保存を分け、Replay/Lookupと実DB Effectを独立観測 | #2598 IB-14/15・#1174。限定Go標準ライブラリ小実験と現行実装静的照合を、TEI実DB実測へ昇格しない |
| **CB-5（E1→E3）委任経由の権限増幅** | 一般利用者Macro→Trusted Script→Providerの間接呼出しで、呼出元がScriptの高権限、raw Connection、別TenantのResourceを暗黙継承しない。公開契約へのアクセスと実施可能なEffectを既存Hostの現在Grant/Scopeに戻して照合 | #1129/#951/#1174/#2390。正式Run-as-Service・組織承認Workflow・OS隔離はOwner決定と実証待ち |

採用時に生成する情報は**既存Source locator、適用Contract、Consumer・Target・実行時Scope、試験fixture/検証基準、未確認点への参照**で足りるか先に検証する。新しい横断`Approval`フラグ、独立Conformance Registry、全形式共通の`receipt` SSOTは先決しない。標準データ表現・Semantic Context・Data Operationsの拡張は[#2619](https://github.com/hat47x/tei-platform/issues/2619)の未採用の横断計画を参照し、このGateがそれらの正式DSL/Schemaを制定したとは扱わない。

### 3追. Generation固定の外側にあるFunction StateとRead Witness（2026-11-02横断接合）

**適用対象：G3の動的切替とG4の外部Consumer資格。追加の採用済み仕様・新Gateではない。** [Composite Functionの版・状態進化 Draft #2612](https://github.com/hat47x/tei-platform/pull/2612)、[Read Cohort/Field観測 Draft #2627](https://github.com/hat47x/tei-platform/pull/2627)、[Function Mapping/Rule依存 Draft #2624](https://github.com/hat47x/tei-platform/pull/2624)、[Everything is Dataの同一版接合 #2629](https://github.com/hat47x/tei-platform/pull/2629)（main統合済みの横断計画。Owner意味受理・実E2Eは未達）の**独立した設計・合成成果**を、#1223が実際に必要とするHost側の継続性／資格判断に接続する。各Draftの試験成功を合算して製品G3/G4合格とは扱わない。

一つのInvocationについて、固定された**実装**と、時間とともに更新される**意味・状態・取得データ・現在権限**を次のように別の証拠として扱う。

| 検査する事実 | 必要な固定・照合の対象 | Generationをpinしても残る不確実性 |
| --- | --- | --- |
| **意味・構造** | Accepted meaning/Rule、Source locator・Role・Operator版、Function Definition/Instance、明示Data Context | Syntaxの正規化一致や同じOperation IDだけではAction意味・改版後の適用資格を説明できない（#2619/#2620/#2598） |
| **実行・状態** | 選択済みContract/Provider/実ロードArtifactとHost-local Generation、Instance state/version、移行候補の事前条件 | v1のin-flight保持は、状態付きFunctionのv1→v2変換可能性・rollback・Domain有効時点を保証しない（#2612/#1223） |
| **入力・観測** | Consumerが要求するRead footprint、実Provider/Scope/Source版/取得時刻、Read cohort、Rowsetの被覆と各Fieldのobserved/value/null/absent | 同じSnapshot名、行集合`complete`、同じ表示値でも、別Provider・隠れた絞込項目・未観測Fieldを同一観測と証明できない（#2624/#2627/#2628） |
| **利用・確定** | 呼出し時の現在Service Acceptance/Host Grant、exact送信Request/OperationID、Commit時Fence、Receiptと独立Effect観測、Consumer向け開示条件 | 旧Lease保持・Tool/Skill成功・Receipt照会失敗を現在の実行許可、永続作用の成功、適切な開示と同一視できない（#1174/#2390/#2605） |

G3のv2公開判定に加える最小の**反証ケース**を既存Ownerへ割り当てる（本PRではいずれも**NOT RUN**）。

- **CB-6：State移行とGeneration切替の非同値（E4/G3、#2612）。** Stateを持つFunction v1の処理を保持したままv2をReadyにしても、新版Stateに必須な旧履歴・発効日・Contextが失われるなら、当該Instanceの移行／旧版への復元を成功としない。必要な状態移行を`M: S1 ⇀ S2`という条件付き部分写像として比較し、切替可能なServiceと移行不能なInstanceを別列で記録。無状態のProvider交換では状態移行を必須としない。
- **CB-7：Read CohortとField観測の相違（E2→E3、#2627/#2506）。** 同名Snapshot・同じ行ID集合で`Complete=true`でも、一方のProviderの対象Fieldが`absent`なら明示`NULL`や同一値と断定しない。非公開のFilter Fieldを物理Readした場合はConsumerの開示範囲とProviderの取得権限を独立照合。得られるClaimの十分性だけを判断し、欠落値を外部Harnessへ補完送出しない。
- **CB-8：Source表現から実利用までの資格の分断（E0→E2/G4、#2625/#2629/#2390）。** Markdown/DOT/DSLから同じplain Dataを得ても、引用・文字列・実行Role、意味版、参照Scopeが異なる場合は同じ実行Contractとみなさない。外部Consumerが実ロードしたLibrary/ArtifactSet、現在Host権限、実際のResult/Disclosureを別々に観測する。構文変換の局所PASSは外部製品での実利用資格ではない。

各Caseの**実証は対応Issue／既存Scenario／独立検証基準**が所有する。#1223ではGeneration/Hostを横断する結果だけを照合する。必要な確認にはSource/Meaning revision、Instance IDと状態、選択済みProvider/Artifact、Read Scope/時点/観測Field、Request/Effect ID、実Target、現在Authority、Consumerが受け取ったDisclosureを、**既存正本への参照として**一行ずつ結び付ける。共通Clock API、状態移行Engine、全情報流Checker、第二Source正本は導入しない。観測の欠落は当該Claimの`unproven`として残し、別レーンの実行可能な仕事は継続する。

### 3再. 変更後の再資格化をBuild再利用・Consumer Claim・現在権限で分ける

**2026-11-02の実装状態を修正**：標準Operationの導出・束縛実行・設計成果物・CLIのPR [#2587](https://github.com/hat47x/tei-platform/pull/2587)はmainへ統合（merge `8464db0bbbac216ede15d3344e72b7e8bd90cb65`）。固定して照合したmain `b1c019fec4f05ca65e3d7491608cb849ce523d29` の `internal/operations/surface.go`（blob `e1024a56e060c2dda4e64343d4eaae647782f84b`）は、同じEffect/DataのCapabilityから辞書順最小IDをBindingに選び、Alias順に導出した同一Operation IDの後続候補を内容比較せず省略する。**現行選択規則が持つ静的反例**であり、異なる実DBへの誤更新を再現した証拠ではない。#2485/#2611の複数Capability・Mapping検証を引き続き利用する。

既存G3/G4の受入時、変更の種類ごとに**過去Evidenceの適用資格**と**現在の利用資格**を区別する。具体的な全再生成／選択的再利用のコスト比較は[#2600のDraft #2636](https://github.com/hat47x/tei-platform/pull/2636)へ委譲し、本節はComposition・外部Consumer固有の資格に限る。

| 変更対象 | 再照合する事実 | 安全に言えないこと |
| --- | --- | --- |
| 構文・表示位置だけの変更 | Source locator・Role・Operator版・実依存閉包とSHOレビュー対象 | 意味依存が不変との確認なしに旧Claimを再利用／無関係な実行Artifactまで必ず再生成 |
| Capability・Alias・Action追加 | Operation ID、Basis、**実Bindingと候補集合**、Rule/Meaning版、選択したPersistence、Consumer保持Intent | ID/Basis不変を意味同一や現在の実行資格に昇格。別Scopeまで一律失効 |
| 表示に現れないRule/Query条件の変更 | WHERE・ORDER BY・Hidden Field、Function Contextの実Read footprint、Provider Scope、独立検証基準 | 表示値/出力列の一致を同一Claimや未取得データへの許可とみなす |
| Artifact/Target/Hostの改版 | 対象Consumerのrequired member・実ロードbyte、Generation、Hostの現在Route | 単一Manifest/digestや過去のLoad成功で現在配備・外部Host適合を証明 |
| Provider/Host Grant・開示資格の失効 | Invocation時、送信/Result公開直前、不可逆EffectのCommit直前の該当Ownerの現在条件 | 以前の署名・CI・Receiptを現在Authorityとみなす。旧世代のLease解放でDB効果を取り消す |

**CB-9：取得と公開の間のAuthority変化（E1→E3、試験未実施）**。T0でProviderのField Readが許可され値を得た後、T1でHostのField Grant/Consumer Disclosureが失効し、T2でResponse公開しようとする。T0時点の許可だけでT2の外部送出を成功扱いしない。[PR #2637](https://github.com/hat47x/tei-platform/pull/2637)はmain統合済みで、Derived QueryのProvider取得前と結果返却直前のField Read Grant再照合を実装した。ただし合成Callbackでの二地点確認であり、実Hostの並行失効・ABA・送出後の撤回保証ではない。**CB-9の実Host/Consumer公開の一体試験はNOT RUN**。Result公開とDB Commit、内部診断の外部開示をそれぞれ別判断にする。拒否診断から非公開項目の存在や内部権限状態を外部へ漏らさない。

再資格化票は`Consumer/Claim、元Source・意味版、依存辺の変更、必要な再取得・検証基準、保持できる過去Evidence、利用時点のHost/Disclosure、未確認理由`を既存正本への参照で示す。**共通キャッシュ・新しいAuthority・全局Revision・新Gateは追加しない**。局所的に無関係なConsumerの再実証を自動要求せず、影響範囲不明の場合だけ当該Claimを`unknown`とする。

### 4. 次に採る最小の縦断Evidence（重複実装しない）

| 順序と担当 | 具体的な反例と観測 | 合格主張の上限／停止条件 |
| --- | --- | --- |
| **E0：既存呼出し棚卸し（#951/#1223）** | Web read-only quote、Driver submit、Service Carrierの各Callerから、Source→選択→Host/Target→Provider→結果の実際の辺をSource locator/固定SHAで列挙 | 静的探索は実結線ではない。存在しない辺を新Hostで埋めず、既存Resolver/Caller所有者を記録 |
| **E1：現在Route/受理の実Source接続（#951/#1118）** | Hostが固定したApp/Env、現在Deployment rev、Publisher/Provider受理の失効を各段階で注入。自分で整合化した別EnvironmentのPlanや旧受理も拒否 | 合成`currentRoute`/Readerだけでは不足。実Sourceの提供元、版、取消時の権限、非公開診断の開示境界を確認できなければ未合格 |
| **E2：読み取りConsumer（#2390/#1708）** | 同一ContractをTEI内部と外部Hostの各1 Consumerで呼ぶ。全件／既知部分取得、0件／NULL／欠損、Snapshot時点とread-setがConsumer要求に合うか比較 | 型・レスポンス一致だけで意味適合にしない。実際の外部Hostで実行できなければ「外部利用可能」を主張しない |
| **E3：書込みの確定境界（#1174/#1129）** | 同一OperationIDでcommit→ACK喪失→再起動→Lookup、権限失効・CAS衝突、post-commit不正Result。DB状態とReceiptを別Connectionで観測 | `Committed`と`Unknown`を区別。Generation/HostのDrainをDB rollbackとしない。Owner未受理のDenied/Outcomeを創作しない |
| **E4：動的／静的Target比較（#1223/#259/#2390）** | 動的形態はG3の実2 Consumer/常駐v1/v2/容量/失敗復旧/SLO、静的形態はG4の実リンクされた依存閉包・サイズ/起動/RAM・外部HostでのContract回帰 | 片方の結果を他方へ流用しない。Studio Wails Hostを第二Application Targetとして数えない |

**2026-11-09 E0/E2追加Evidence（Gateの判定変更なし）**：[PR #3757](https://github.com/hat47x/tei-platform/pull/3757)の[6宣言Consumer調査](../evidence/BINDING_FORMS_CONSUMER_SURVEY_2026-11-09/README.md)（Source `main 0837a6f4`、[全体CI #42606374129](https://github.com/hat47x/tei-platform/actions/runs/42606374129) success）では、通常`cmd/tei -plan`で`observation.Binding`／`evidence.Binding`の宣言IDを実解決する一方、非試験`observation.Router{}`／`evidence.Router{}`の組立て・`Observe/Collect`実Invokeは**同調査で0件**。C-3のようなID解決器を増設するのではなく、既存Result/Ownerへ接続する実Consumerが必要かをC-4外部委譲を含めて判断する。調査は時点付き静的Caller記録で、現在Source・受信byte・Open Scope・Grant・外部観測を確定しない。

**2026-11-09 E3追加Evidence（Gateの判定変更なし）**：[PR #3717](https://github.com/hat47x/tei-platform/pull/3717)の[実PostgreSQL CI #42570360880](https://github.com/hat47x/tei-platform/actions/runs/42570360880)はcompleted/successで、`lifecyclepg`、標準CRUD、Expected State CAS/ACK喪失、tenant範囲Web APIの4系統をDSN付きで実行し、DSN未設定SKIPを禁止した。これは**実DBが動いた資格**としてE3の部分Evidenceへ追加できる。一方、各試験は別DB・別Operation/Contractであり、E3が要求する**同一OperationID**のHost実Command→Commit→ACK喪失→再起動→Receipt別Connection読取→Client Reactionの連続性を証明しない。E3全体は未受入のまま維持する。Hookの`NotSatisfied`とCommand業務拒否名の[部分対応 #3721](https://github.com/hat47x/tei-platform/pull/3721)も実Plugin経由のHTTP Status/Effect判定を代替しない。HTTP側の独立Expectedは[設計§23](../design/HTTP_RESPONSE_DECISION_GRAPH_DESIGN.md)へ集約し、E3と第二のEffect Fenceを作らない。

### 4.1 統合済み部品から始める最初の実証パケット（2026-11-02のmain照合／局所試験は2026-11-07に実行）

**E0〜E2の実行入口であり、新Gate・実行APIではない。** #2587（標準Operation/Artifact/CLI）、#2275（Host Application RouteとService Carrierの合成）、#2637（D2 Field Grant再検査）はmain統合済み。これらは**異なるContract・Callerを持つ独立経路**であり、同一業務RequestのE2E実証ではない。#2629の横断計画は正式Meaning/Provider接続を代行しない。

| 局所経路 | 実装済みの確認入口と必要Evidence | まだ接続できていない辺 |
| --- | --- | --- |
| **A：Source→Operation→Artifact** | `reference/standard-crud/workspace`で`tei validate`、`tei operations -application ledger`、`tei artifacts -application ledger`。Operation ID/Basis/Binding/Rule、Gap、Manifest/Required memberを固定し、`TestSurfaceWorkspaceStandardCRDFixture`と比較 | `ledger.update.sales.order`は候補であって、正式Owner受理・Host current Route・DB効果の証拠ではない。#2611の複数Aliasは別Owner |
| **B：保護Web Action→Service Carrier** | `TestCanonicalQuoteActionUsesHostOwnedGenerationServiceRoute`で合成Scope、Generation、環境不一致・失効拒否と実行回数を固定 | `order_entry.preview_order_quote`をAの`ledger.update.sales.order`と同一Contract/業務Actionとして接合しない。正式current Source・実Provider未確認 |
| **C：Provider Read→Consumer Result** | `TestD2DriverRechecksAuthorityAfterCompleteSourceRead`でProvider途中失効後の空戻り・拒否を確認 | BのService実装やHTTP送出の原子的開示、実DB Snapshot/Commitをこの合成D2試験で証明しない |

最初に**同じ固定checkoutで局所試験を別々に実行**し、対象Git HEAD、前後のclean、Toolchain、Source実byte/Source Index、WorldFingerprint/ModelRevision・Ruleを保存する。特に`tei operations`と`tei artifacts`は**別プロセス・別Load**であり、同一パスだけからSource Snapshotの同一性は導けない。さらに現mainの`cmd/tei/main.go`は両コマンドの分岐前に`loadWorld`を呼び、分岐後の`operations.SurfaceWorkspace`／`artifact.RenderWorkspace`でもWorkspaceを読み込むため、**各CLI呼出し内にも二度のLoadがある**。この事実は静的な実装所見であり、実際に異なるSourceを使ったという観測ではない。今後、不要な二重Loadの解消と、実際に出力された内容・Source Indexの束縛を別途検証する。出力の比較と独立Expectedを分離する。

```sh
git rev-parse HEAD
git status --porcelain=v1 --untracked-files=all
go run ./cmd/tei validate -workspace reference/standard-crud/workspace
go run ./cmd/tei operations -workspace reference/standard-crud/workspace -application ledger
go run ./cmd/tei artifacts -workspace reference/standard-crud/workspace -application ledger
go test ./internal/operations -run '^TestSurfaceWorkspaceStandardCRDFixture$' -count=1
go test ./reference/web-composition/go-host -run '^TestCanonicalQuoteActionUsesHostOwnedGenerationServiceRoute$' -count=1
go test ./internal/driver -run '^TestD2DriverRechecksAuthorityAfterCompleteSourceRead$' -count=1
git status --porcelain=v1 --untracked-files=all
git rev-parse HEAD
```

**2026-11-07の実行記録**：上のコマンド群を、`origin/main`の`18cf8ae0`で一つの固定checkoutから**別プロセスで**実行しました。記録と生データは[構成Kernel移行Gateの最初の実証パケット実行記録](../evidence/COMPOSITION_FIRST_CONSUMER_RUN_2026-11-07/README.md)にあります。

- A経路で`validate`・`source-index`・`operations`・`artifacts`が成功し、`operations`の`modelRevision`と`artifacts`の`inputRevision`が同じ値（`sha256:c60ed68c…b04d`）でした。操作面の5操作・4欠落は`TestSurfaceWorkspaceStandardCRDFixture`の期待と矛盾しません。
- B（`TestCanonicalQuoteActionUsesHostOwnedGenerationServiceRoute`）とC（`TestD2DriverRechecksAuthorityAfterCompleteSourceRead`）も、Aとは別の試験としてPASSしました。
- 依存の形でも、Aの`cmd/tei`は`internal/composition`を含まず、Bの実行単位は`internal/operations`を含みません。`internal/servicehost`は非試験のファイルからどこもimportされていないため、**Cの試験成功を実HostからのService Carrier呼出しと読み替えません**。
- **この実行が示さないもの**：A・B・Cが同一の意味／Operation／Request identityを通ったこと。Owner受理、Host current Route、実DB Effect、実Provider、実SUI Browser、`tools/repository-health-check.sh`の全項目。`operations`と`artifacts`が読んだSource bytesの同一性も、`modelRevision`の一致だけでは示していません。**G1〜G4の合否は変わらず**、判定記録も別に必要です。

**製品としての次の一辺**：Aの選択したBusiness Action／Contractと、BがHost所有の現在Route・Service Acceptanceから実Providerへ渡すContractを、実際に**同一の意味／Operation／Request identity**として結ぶ既存Callerを探索する。なければ「未接続」と記録し、A/B/Cの個別PASSを混ぜない。まず実在するread-only Contractを固定し、書込み・Receipt・実DB EffectはE3/#1174、外部Java/.NET/Wasm実LoadはE4/#2390に渡す。

### 4.2 Product CLI・Skill・外部Harness・Hostの適合境界（横断成果の受渡し）

[TEI Product Skill Draft #2607](https://github.com/hat47x/tei-platform/pull/2607)と[TAIの横断Skill計画](https://github.com/hat47x/tai-pleroma/issues/168)は、**実在するCLIの使い方をAgentへ知らせる層**であり、Pluginの配布ABIやHost実行権限を発行する層ではない。以下を混同せず、#2605/#2390/#2516へ試験責務を戻す。#2607の候補CIは、実CLI・他Harness・外部Hostの正式受入ではない。

| 境界 | 必要な証拠 | できても証明しないこと |
| --- | --- | --- |
| **Source/Meaning** | Source byte/locator、Role・Operator依存、Accepted Meaningと候補の区別 | Syntax/Roleが読めることによるOwner意味受理 |
| **Product CLI** | 同一実行バイナリ、実サブコマンド・入力、Source版、警告/未解決、JSON/Traceの観測を保存 | read-onlyコマンド成功によるProvider契約や永続Effect |
| **TEI Product Skill** | 実CLIの説明との差分・コマンドと引数・版固定・拒否/未対応案内、必要最小Context | Skill文書や合成検査のPASSによる外部Harness動作・実開示許可 |
| **TAI/DSH等のHarness** | 実際のSkillロード、渡したPrompt/Tool引数、外部送出範囲、再取得時の資格、別Harnessでの再現 | AIの回答・Tool選択・MCP疎通による実行Authority/Owner受理 |
| **既存Host/外部Consumer** | 現在Grant/Service Acceptance、実ロードArtifact/Provider、呼出し・結果の独立検証基準 | TEI CLIが使えたことによるJava/.NET/Wasm製品互換 |

**最小の横断負例 HS-1〜3（本書ではNOT RUN）**：HS-1はDiscovery時のCLI版AとInvoke時の版Bを取り違えて実行不可なコマンドを勧める。HS-2はread-onlyな`artifacts`を`-out`付きで実行し、ファイル書込みが発生したのに無副作用と表示する。HS-3は同じRead結果でも内部raw/開示禁止FieldをSkillやHarnessへ渡す、または先に得たGrant失効後の現在結果とする。いずれも新Agent Runtimeや共通Hostを作らず、既存CLI/Skill/Hostの担当ごとの独立観測で否定する。**Skill案内成功を外部Library Consumerの可搬性合格へ昇格させない。**

### 4.3 空結果・否定依存・時間差を同じRead-only Consumerへ接続する

他チャットからの横断成果（[#2582](https://github.com/hat47x/tei-platform/issues/2582)、[#2586](https://github.com/hat47x/tei-platform/issues/2586)、[#2516](https://github.com/hat47x/tei-platform/issues/2516)、[#2619](https://github.com/hat47x/tei-platform/issues/2619)）を、**E2のConsumer Claimの判定条件**として接合する。新Negative Cache/Proof Engineや第二のQuery/Policy正本は作らない。ここでの各ケースは**次段の試験入力案であり、PR #2660ではNOT RUN**。

| 変化・対照 | 最低限保持する依存と独立Expected | 合格主張の上限 |
| --- | --- | --- |
| **RC-N1：完全な0件と部分取得0件** | 対象Subject/Relation occurrence、Predicate、取得Scope、Provider/Route、Snapshot、Rowset/Field完全性、否定の独立検証基準 | **完全な母集団についてその条件・時点で0件**とは言えても、別Scopeや未来を含む「存在しない」とは言わない。部分取得の0件では不存在Claimを保留 |
| **RC-N2：0→1→0、表示値が同じABA** | 中間変更、Provider cohort、Result Slice epoch、実Source/Rule/Authorityの時点、どこまで変更を観測したか | 最終JSONと初回JSONが同じでも「何も変わらなかった」の証拠ではない。ただし中間変更を観測できないProviderでは観測事実を捏造しない |
| **RC-N3：非出力のRule・Predicate/WHERE/ORDERが変わる** | `Query→Function Context→Claim`の必要入力・否定依存、実取得とConsumer向け表示の差 | 出力列・集計値・Operation IDのbyte一致だけで旧Claimの現在適用資格を再利用しない。無関係なClaimを一律失効させない |
| **RC-N4：Sourceの移動、Grant失効、Harness送出** | World/Meaning FingerprintとSource Index/Trace locator、選択したReadの現在Host Grant、Tool候補/診断/Renderer/実Egressを別時点で観測 | 実TEI CLIのSource再配置Reference [#2626](https://github.com/hat47x/tei-platform/pull/2626)が局所PASSでも、SHO引用自動復権・外部AIの現在開示許可は未証明 |

初回は同じ固定Fixture/Consumerの**閉じたread-only Query**で、正常0件・既知の部分取得・追加/削除によるABA・別Scopeを対照する。必要なSource/Predicate/Route/時点が取れなければ該当Claimだけ`unknown`とし、実際に許された0件を一律エラーに変えない。過去Evidenceの再利用（#2600）と、今取得しなおすべき業務Data/Disclosure（#2582/#2516）は別の判定。#2514の独立検証基準で正負両方を照合し、#2018には必要な再取得量・検証負担を計測して渡す。

**E3との接続境界**：[#2664](https://github.com/hat47x/tei-platform/pull/2664)は、`effectcommit.Runner`をWeb以外のCallerへ渡す`submit`参照と、CLI形の**プロセス内試験Driver**のDraft候補。報告されている使い捨てPostgreSQLでの結果・再照会・権限版の試験は、実DBを使った限定的な局所証拠であり、本書のE2 Consumerと**同じOperation/Source/Grant/Provider/Request**を通った証拠ではない。CLI形Driverを製品の`tei`コマンドと呼ばず、合成Grantを実発行済み権限に昇格させない。E2の完了前でも#1174担当で独立実験は進められるが、E3を一体合格と判定しない。

AI-first/Headless-firstの実施順序（外部要求の机上評価→既存機構の最小結線→CUI/標準Plugin並行→headless縦断→必要なGUI）は全体計画Draft [#2492](https://github.com/hat47x/tei-platform/pull/2492)を参照し、本Gateで独立の新開発プロセスを発明しない。DSH/TAI/SEI/SHO等への投影では、準備・送付・生成・保存・独立確認の各事実がどこまで観測されたかを表示する。Tool呼出し、構文適合、バイト一致、AIの説明文を独立検証基準やOwner受理の代用にしない。

**採否の停止条件**：現在のTrust/Authority正本を特定できない、Consumer間で意味・開示Scopeが一致しない、外部Hostへの権限伝播を未検証、Commit結果が不明、又は旧ABI撤去の全Callerを列挙できない場合、該当する受入だけを保留する。別レーンの局所的な実装・試験は継続できる。新しい共通Host/Resolver/Authority/Manifestを増やす必要性は、既存の不足辺と独立Consumerの反例が示すまで未立証とする。

## 判定記録に残す情報

対象head、対象main、Health runnerのGit blob、GoとOS・Target、実行コマンドと終了状態、前後のclean状態、ログの保存先とSHA-256を記録する。G2以降では実成果物の指紋、世代とProviderの対応、合成した入力・権限の範囲、実施していない確認も記す。性能値には測定環境、反復数、負荷を添え、比較対象が無い測定から改善率を算出しない。

G1、G2、G3、G4の結果を別々に判定する。#1223はADR-0032の破壊的移行と無停止更新も担当するため、G1の合格や一つのPRの統合だけでIssue全体を完了にしない。
