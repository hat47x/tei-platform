# 機能設計文書

ここには、複数のIssueにまたがる機能について、意味モデル、記述方法、検証、実行時の境界をまとめた設計文書を置きます。採用前の案も、対象となるIssueと文書の状態を冒頭に明記して管理します。

- 作業内容と完了条件：GitHubのIssue
- 方針を採用した理由：[`../adr/`](../adr/README.md)
- 作業の順序と必要な検証根拠：[`../README.md`](../README.md)
- 現行実装の構成と技術的な境界：[`../../docs/architecture.md`](../../docs/architecture.md)
- 実装と検証根拠の現状：[`../../docs/status.md`](../../docs/status.md)
- 画面の試作例：[`mockups/`](mockups/README.md)

設計文書はIssueやADRの代わりにはなりません。作業の進行状況はIssue、採用した設計判断はADRを正本とします。ここには、それらの記録だけでは一つにまとまらない、機能単位の設計本文を置きます。小さな機能については新たな文書を作らず、Issue本文に記します。

`docs/`とは役割を分けます。`docs/`では実装済みの構成と利用方法を説明し、ここでは採用前の案や変更に必要な機能設計を扱います。実装状況が変わった場合は、この設計文書を現行構成の説明に流用せず、`docs/architecture.md`を更新します。

## 体裁

新しい文書は[`docs/README.md`](../../docs/README.md)の「設計文書の基本体裁」に沿って作ります。冒頭で文書の役割、正本との関係、対象範囲、現在の結論、根拠、何を主張しないかを明らかにします。形式上の項目を埋めること自体を目的にはしません。

文書の状態は各設計文書の冒頭に記し、その文書自身を状態の正本とします。この索引には、変わり得る状態を重複して記載しません。状態を二択に固定せず、文書の性格に応じて`実装済みの現行基線`、`実装前設計`、`評価待ち（未採用）`などと区別します。

ここで示すのは**文書の性格**です。Issueの進行状況、判定段階の結果、実際に得られた検証根拠など、変化する状況は対応するIssue、`docs/status.md`、判定段階の監査記録などを正本とします。設計文書の状態欄には重複して記載しません。

## 一覧

| 文書 | 対象 |
| --- | --- |
| [`APPLICATION_PATTERN_V1_DESIGN.md`](APPLICATION_PATTERN_V1_DESIGN.md) | アプリケーションの構成パターンの設計・実装基準（ISSUE-000005 / ADR-0017 / ADR-0018） |
| [`LEGACY_AUTH_V1_DESIGN.md`](LEGACY_AUTH_V1_DESIGN.md) | 既存の認証仕様を保ち、認証・認可の機能へ接続する設計（ISSUE-000003 / ADR-0019） |
| [`CAPABILITY_OUTCOME_V1_DESIGN.md`](CAPABILITY_OUTCOME_V1_DESIGN.md) | 機能の業務結果・名前付きの結果と実行エラーを分離する設計（ISSUE-000009 / ADR-0020） |
| [`APPLICATION_DRIVER_V1_DESIGN.md`](APPLICATION_DRIVER_V1_DESIGN.md) | 構成パターンの解決から、機能と契約に基づく問い合わせの実行、観測までの境界（ISSUE-000006 / ADR-0019 / ADR-0020 / ADR-0021 / ADR-0023） |
| [`DATABASE_CONTRACT_QUERY_V1_DESIGN.md`](DATABASE_CONTRACT_QUERY_V1_DESIGN.md) | 契約に基づく問い合わせの形式、SQLによる実現、実行器との対応付け、提供実装の実行基盤という四層（ISSUE-000018〜ISSUE-000021 / ADR-0007 / ADR-0021） |
| [`BUSINESS_LIFECYCLE_V1_DESIGN.md`](BUSINESS_LIFECYCLE_V1_DESIGN.md) | 業務の状態遷移を意味の正本として定義し、実行・観測する設計（ISSUE-000038 / ADR-0024 / ADR-0025） |
| [`DESIGN_SYNTHESIS_CROSS_VIEW_PROOF_DESIGN.md`](DESIGN_SYNTHESIS_CROSS_VIEW_PROOF_DESIGN.md) | RQ-026の複数視点の照合を、対象を限定した証明上の条件として設計案の候補を絞り込む処理へ接続する、評価待ちの設計 |
| [`PROJECT_PLUGIN_COMPOSITION_DESIGN.md`](PROJECT_PLUGIN_COMPOSITION_DESIGN.md) | 案件ごとのプラグイン、画面部品一式、仕様テンプレート、構成プロファイルの境界（[GitHub #1016](https://github.com/hat47x/tei-platform/issues/1016)、評価待ち） |
| [`CAPABILITY_INVOCATION_CONTEXT_DESIGN.md`](CAPABILITY_INVOCATION_CONTEXT_DESIGN.md) | 機能に渡す意味上の要求と、ホストが注入する実行時の文脈を分ける、評価待ちの設計（[GitHub #1118](https://github.com/hat47x/tei-platform/issues/1118)） |
| [`SUI_REFERENCE_ADOPTION_DESIGN_2026-09.md`](SUI_REFERENCE_ADOPTION_DESIGN_2026-09.md) | SUIの既存実装を生かしながらTEIへ移行するための設計基準 |
| [`CROSS_VIEW_CHANGE_QUALIFICATION_REFERENCE_DRAFT.md`](CROSS_VIEW_CHANGE_QUALIFICATION_REFERENCE_DRAFT.md) | Issue #1143で検討中の、複数の視点にまたがる変更の適格性確認案。参照用の契約としては未採用 |

## 関連する参照事例と評価


`DESIGN_SYNTHESIS_CROSS_VIEW_PROOF_DESIGN.md`の評価順序は、後続の読み取り専用の検証結果に合わせ、R1～R6へ反映済みです。当初、変更や古い状態の再評価として扱う予定だった論点は、R3～R5を先に個別に評価したうえで、R6のスナップショットの失効として扱います。

R3の初期構想にある`proposal_origin`は、現在の本番用フィールドの候補ではありません。R5と、異なる種類の候補が混在する場合の精緻化を優先し、候補が表す意味と、発見経路・出所の記録を別の観点で扱います。

## 設計案の統合で判定段階C前に立ち止まる地点

設計案の統合とローカル認知を扱う作業領域Dでは、判定段階Cより前に安全に進められる読み取り専用の設計評価として、R1～R6、S1・S2の候補群の検証、異なる種類の候補が混在する場合の文脈の精緻化、提供実装の実現可能性と結果の分類に関する検証、S1～S5の受理に関する素案までを実施しました。

ここまでの評価で確認した現行の境界は、次のとおりです。

```text
candidate discovery
  -> exact / Pattern / associative provenanceを分離

candidate qualification
  -> current accepted authorityへ戻る

cross-view proof
  -> supported / contradiction / unresolvedを分離

candidate contraction
  -> exact scopeのcontradictionだけを決定論的に除外
  -> unresolvedは残余dimensionとして保持
  -> frontierを必ず1 candidateへ縮約しない
  -> survived != supported
  -> supported existing candidate != preferred / winner / accepted candidate

provider feasibility
  -> semantic candidate != realization / binding variant
  -> missing provider Evidenceはsemantic contradictionにしない
  -> provider-specific incompatibilityはmatching realization variantへ閉じる
  -> hard Target / Provider requirement時だけcomposite candidate eliminationへ拡張する
  -> provider limitationからBusiness / Canonical meaningを逆authorしない
  -> same error surfaceをsame resultへ畳まず、reason / owner / scopeへ戻す
  -> missing Evidence / operational recoveryはunresolvedとして保持する
  -> observed incompatibility / structural unsupportedだけをexact realization claimへ縮約する

context packet
  -> candidate subject != discovery trace != candidate-local proof / qualification
  -> R3初期の単一proposal_originへ戻さない
  -> supported existing + unresolved proposalをflattenせずS3 / human reviewへ渡す

proof freshness
  -> required accepted authorityのsemantic fingerprintへ束縛
  -> stale contradictionをcandidate eliminationへ使わない

LLM boundary
  -> bounded residualをProposalとして扱う
  -> minimum semantic deltaはS2 proofではなくS3 / human comparisonで扱う
  -> Acceptance / authorityを所有しない
```

判定段階Cより前には、これらの結果だけを根拠として、次の項目を本番向けの契約として固定しません。

- 正本に新設する要素の種類としての`DesignCandidate`
- 本番用の`DesignContextPacket`スキーマ
- 候補の出所を表す排他的な列挙値
- 証明の有効期間や結果を保存する列挙値
- 永続化する`AcceptanceSnapshot`や証明の評価時点の記録
- 本番用の構成パターンの記憶スキーマ
- ローカル・外部の言語モデルを提供する仕組みの契約
- ベクトルデータベース、グラフデータベース、汎用の制約解決機構
- 共通の受理境界での自動的な受理
- 意味の差分を永続化するモデル
- フィードバックの出来事を分類する体系
- 本番用の`ProviderFeasibility`列挙値や候補の階層
- 提供実装のエラーと結果の分類体系を本番向けに固定すること

新しいread-only作業を追加するのは、R1〜R6、S1 / S2 frontier pressure、mixed-frontier context refinement、provider feasibility scope / outcome-classification pressureでも説明できない具体的なinformation loss、authority contradiction、provenance / freshness / scope / owner-routing gapがcurrent Referenceで見つかった場合に限る。単に将来必要そうという理由で抽象化やtest-only shapeを増やさない。

### Gate C後のdownstream handoff guard

Gate C後に`ISSUE-000047` / `ISSUE-000051`を再開するとき、R5 / R6で確認済みの境界を別のschemaや認知層として再設計しない。

`ISSUE-000047`へ渡す既知条件は次である。

- semantic candidateとdiscovery mechanism / discovery traceを分離する
- 同一candidateがexact / Pattern / associative等の複数経路で見つかっても、candidateを複製せず複数traceを保持できる
- Reference Patternのprovenance、associative retrieval representationのbuild-input provenance / scoped freshness trace、current M1 qualificationを別物として扱う
- retrieval score / similarity / 過去利用回数をapplicability / Evidence / Acceptanceへ変換しない
- exact existing assetも「存在する」だけでapplicableとせず、current identity / dependency / scopeとrequired accepted authority / Evidenceへ戻ってqualifyする

`ISSUE-000051`へ渡す既知条件は次である。

- derived cognitive representationはM2のregeneratable resourceであり、M1 authorityではない
- candidate retrieval方式を変えてもsemantic candidate identityやaccepted meaningを変えない
- derived indexのbuild-input provenance / freshness / stale判定と、M1 accepted authority / proof freshnessを混同しない
- derived representationがfreshでも、current applicabilityやcross-view proofはrequired accepted authority / Evidenceへ独立に戻って確認する
- R6のproof freshnessは「proofが依存するaccepted authority fingerprint」の問題であり、cognitive index用のglobal revision counter導入根拠にはしない
- index削除・方式差し替え後もCanonical Worldが変わらないことを維持する

したがって再開時に、次を改めて導入するところから始めない。

```text
candidate.origin = exact | pattern | associative
という排他的origin enum

retrieval score -> confidence / evidence strength
というauthority変換

repository全体のglobal revision counter

cognitive index / pattern memoryを
accepted semantic sourceとする構造
```

actual Gate C Referenceでこれらの境界では情報が失われることが観測された場合にだけreopenする。Gate C後の最初の仕事は境界の再設計ではなく、実在するS1 candidate frontierへ000047のcurrent qualificationを当て、必要性が出たときだけ000051のretrieval方式を比較することである。

### 再開条件

通常の再開点はProduct Gate C後とする。少なくとも次が実在してから、S1〜S5のactual Referenceへ進む。

```text
Gate B Ready
  -> Handover Baseline B / baseline revision R
  -> First Web Target build T1
  -> Gate C Evidence
```


```text
S1 realistic intent / candidate frontier
  -> S2 incremental cross-view contraction
  -> actual information-lossを見てbounded packetを調整
  -> S3 local LLM / minimum semantic delta
  -> Shared Acceptanceで小さいsemantic changeを1件Accept
  -> S4 implementation / test derivation
  -> S5 one observable feedbackをmeaning ownerへroute
  -> closed loopを1周
```

このclosed loopのactual Evidenceを得る前に、R1〜R6のtest-only shapeやS1 / S2 frontier pressure / mixed-frontier refinement / provider feasibility pressureのconceptual shapeをproduction schemaへ昇格しない。

## まだ設計文書を置いていない領域

実装とADR、Conformanceは存在しますが、機能単位の設計本文をまだ置いていない領域です。必要になった時点で追加し、それまでは各ADRとConformanceを正本とします。

| 領域 | 現在の正本 |
| --- | --- |
| Source-to-Canonical Traceability / Source Index | [ADR-0004](../adr/0004-source-to-canonical-traceability.md)、[`docs/conformance/source-traceability-m0.23.md`](../../docs/conformance/source-traceability-m0.23.md) |
| Database物理schemaのDesired / Actual / Diff / Unresolved、Advanced Index、Migration rendering | ADR-0008〜ADR-0015、`docs/conformance/`のm0.26〜m0.35 |
| Plugin / Process Runtime、Permission、Resource Broker、fencing / takeover | [ADR-0005](../adr/0005-supervisor-and-runtime-fencing-authority.md)、[ADR-0006](../adr/0006-process-command-and-argv.md)、`docs/conformance/`のm0.22.1 / m0.24 / m0.25 |
| Target / Presentation境界 | Gate C前のため未確定。[Gate C Evidence契約](../gate/GATE_C_EVIDENCE_CONTRACT.md)とISSUE-000026 |

Gate前に確定していない領域へ、先回りして設計文書を作りません。
