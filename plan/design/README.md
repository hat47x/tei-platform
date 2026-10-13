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
| [`MULTILINGUAL_SEMANTIC_RESOURCE_DESIGN_2026-10.md`](MULTILINGUAL_SEMANTIC_RESOURCE_DESIGN_2026-10.md) | 意味モデルと表示資源を分離する多言語対応、言語資源の所有、代替、配布、翻訳の確認（[GitHub #1396](https://github.com/hat47x/tei-platform/issues/1396)、評価待ち） |
| [`MULTILINGUAL_BUSINESS_DATA_AND_AI_TRANSLATION_2026-10.md`](MULTILINGUAL_BUSINESS_DATA_AND_AI_TRANSLATION_2026-10.md) | 多言語の業務属性、記述の版・保存・検索・公開と、生成AIによる翻訳候補・確認・受理（[GitHub #1453](https://github.com/hat47x/tei-platform/issues/1453)、評価待ち） |
| [`MULTILINGUAL_TRANSPARENT_PLUGIN_OVERLAY_2026-10.md`](MULTILINGUAL_TRANSPARENT_PLUGIN_OVERLAY_2026-10.md) | 通常プラグインへ多言語の表示情報を後付けする意味参照・適用許可・競合解決と合成（[GitHub #1551](https://github.com/hat47x/tei-platform/issues/1551)、参照実装検証中） |
| [`CAPABILITY_INVOCATION_CONTEXT_DESIGN.md`](CAPABILITY_INVOCATION_CONTEXT_DESIGN.md) | 機能に渡す意味上の要求と、ホストが注入する実行時の文脈を分ける、評価待ちの設計（[GitHub #1118](https://github.com/hat47x/tei-platform/issues/1118)） |
| [`SUI_REFERENCE_ADOPTION_DESIGN_2026-09.md`](SUI_REFERENCE_ADOPTION_DESIGN_2026-09.md) | SUIの既存実装を生かしながらTEIへ移行するための設計基準 |
| [`CROSS_VIEW_CHANGE_QUALIFICATION_REFERENCE_DRAFT.md`](CROSS_VIEW_CHANGE_QUALIFICATION_REFERENCE_DRAFT.md) | Issue #1143で検討中の、複数の視点にまたがる変更の適格性確認案。参照用の契約としては未採用 |
| [`BUSINESS_EFFECT_LIFECYCLE_DESIGN_2026-10.md`](BUSINESS_EFFECT_LIFECYCLE_DESIGN_2026-10.md) | 副作用を伴う実行の段階、確定後の外部効果、冪等の識別子、確定状態、補償、一括処理の再開をまとめる、評価待ちの設計（[GitHub #1129](https://github.com/hat47x/tei-platform/issues/1129)、[#1130](https://github.com/hat47x/tei-platform/issues/1130)） |
| [`EXECUTION_CORRELATION_DESIGN_2026-10.md`](EXECUTION_CORRELATION_DESIGN_2026-10.md) | 一つの業務操作を実行先から提供元まで追跡し、意味の版・ビルド・ロックへ戻るための識別子と記録の規則。担当Issueは無く、評価待ち |
| [`SEMANTIC_CHANGE_DATA_EVOLUTION_DESIGN_2026-10.md`](SEMANTIC_CHANGE_DATA_EVOLUTION_DESIGN_2026-10.md) | 受理した意味の変更と既存の業務データの両立性の判定、影響の観測、段階的な移行。担当Issueは無く、評価待ち |
| [`KNOWLEDGE_VARIATION_DESIGN_2026-10.md`](KNOWLEDGE_VARIATION_DESIGN_2026-10.md) | 再利用した知識の変動点、適用先での選択と分岐、再利用元の更新の扱い。RQ-027・RQ-029に対応する、評価待ちの設計 |

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

新しい読み取り専用の検証を追加するのは、R1～R6、S1・S2の候補群の検証、混在する候補の文脈の精緻化、提供実装の実現可能性と結果分類の検証だけでは説明できない問題が、現在の参照事例で見つかった場合に限ります。対象となるのは、具体的な情報の欠落、権限上の矛盾、出所・情報の鮮度・対象範囲・担当への振分けに関する不足です。将来必要になりそうだという理由だけで、抽象化や試験専用の構造を増やしません。

### 判定段階Cの後に引き継ぐ条件

判定段階Cの後に`ISSUE-000047`と`ISSUE-000051`を再開するときは、R5・R6で確認済みの境界を、別のスキーマや認知処理の層として設計し直しません。

`ISSUE-000047`には、次の確認済みの条件を引き継ぎます。

- 意味上の候補と、候補を発見する方式・発見経路の記録を分ける
- 同じ候補を正確な照合、構成パターン、連想処理などの複数の経路で発見しても、候補を複製せず、複数の発見経路を記録できる
- 参照パターンの出所、連想検索の表現を構築した入力の出所と対象範囲内での鮮度、現在のM1の適格性確認をそれぞれ区別する
- 検索の評価点、類似度、過去の利用回数を、適用可能性・検証根拠・受理の判断へ直接変換しない
- 正確に一致する既存資産も、存在するだけで適用可能とは判断しない。現在の識別子、依存関係、対象範囲、必要な受理済みの権限・検証根拠に戻って適格性を確認する

`ISSUE-000051`には、次の確認済みの条件を引き継ぎます。

- 派生した認知用の表現はM2の再生成可能な資源であり、M1で意味を決定する権限ではない
- 候補を検索する方式を変えても、意味上の候補の識別子や受理済みの意味を変えない
- 派生索引を構築した入力の出所・鮮度・失効の判定と、M1で受理済みの権限・証明の鮮度を混同しない
- 派生した表現が最新でも、現在の適用可能性や複数視点の証明は、必要な受理済みの権限と検証根拠に戻って独立に確認する
- R6における証明の鮮度は、証明が依存する受理済みの権限の照合値に関する問題であり、認知用索引のためにリポジトリ全体で共有する改訂番号を導入する根拠にはしない
- 索引を削除したり検索方式を変更したりしても、意味の正本全体が変わらないことを維持する

したがって、再開時には次の仕組みを改めて導入するところから始めません。

```text
candidate.origin = exact | pattern | associative
という排他的origin enum

retrieval score -> confidence / evidence strength
というauthority変換

repository全体のglobal revision counter

cognitive index / pattern memoryを
accepted semantic sourceとする構造
```

実際の判定段階Cの参照事例で、これらの境界による情報の欠落が観測された場合に限って再検討します。判定段階Cの後は、境界を設計し直すのではなく、まず実際のS1の候補群に`ISSUE-000047`の現行の適格性確認を適用します。必要性が確認された場合に限り、`ISSUE-000051`で検索方式を比較します。

### 再開条件

通常は製品の判定段階Cの後に再開します。少なくとも次の条件が実際に成立してから、S1～S5の参照事例を用いた検証に進みます。

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

この一連の改善循環を実際に検証した根拠が得られるまでは、R1～R6の試験専用の構造や、S1・S2の候補群、混在する候補の精緻化、提供実装の実現可能性に関する概念上の構造を、本番用のスキーマには採用しません。

## まだ設計文書を置いていない領域

ここには、実装、ADR、準拠性の検証資料は存在するものの、機能単位の設計本文はまだ置いていない領域を示します。設計本文は必要になった時点で追加し、それまでは対応するADRと準拠性の検証資料を正本とします。

| 領域 | 現在の正本 |
| --- | --- |
| 情報源から正本の意味への追跡と入力元索引 | [ADR-0004](../adr/0004-source-to-canonical-traceability.md)、[`docs/conformance/source-traceability-m0.23.md`](../../docs/conformance/source-traceability-m0.23.md) |
| データベースの物理スキーマに関する期待状態・実際の状態・差分・未解決事項、高度な索引、移行用の生成処理 | ADR-0008〜ADR-0015、`docs/conformance/`のm0.26〜m0.35 |
| プラグインとプロセスの実行基盤、権限、資源の仲介、古い実行担当の排除・担当の交代 | [ADR-0005](../adr/0005-supervisor-and-runtime-fencing-authority.md)、[ADR-0006](../adr/0006-process-command-and-argv.md)、`docs/conformance/`のm0.22.1 / m0.24 / m0.25 |
| 実行先と表示・操作の境界 | 判定段階Cより前のため未確定。[Gate C Evidence契約](../gate/GATE_C_EVIDENCE_CONTRACT.md)とISSUE-000026 |

判定段階の前に確定していない領域について、先回りして設計文書は作りません。
