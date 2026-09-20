# ISSUE-000039: Semantic差分・依存影響解析を業務変更の共通機能として実装する

- 状態: In Progress — 最初の解析境界（Validation差分・明示依存の到達・file移動のno-op）を`tools/internal/changeanalysis`に実装済み（2026-09-21）。Binding kind・Scenario到達・Evidence鮮度・JSON / 人向け出力・変更リハーサルは未着手
- 提案日: 2026-09-04
- 提案順位: 2（根幹機能の新規提案内）
- 本格着手条件: Gate C通過後（成立済み）

## 根幹機能としての価値

TEIが業務の意味を長く保持するには、「意味のどこが変わり、どの業務を再検証すべきか」を共通機能として判断できる必要があります。変更箇所の一覧だけでは、Validation変更やCapability Binding交換の影響を開発者が別々に調べる負担が残ります。

固定した二つの設計からSemantic差分を計算し、明示依存を伝播して関連Application／Pattern／Driver Scenarioを特定します。変更前後のfixture実行へ接続し、影響と検証結果を一続きに確認します。AIを使わないCLIから始め、Editor／Acceptanceも同じ解析を利用できる境界を作ります。

関連要求: RQ-001 / RQ-002 / RQ-003 / RQ-008 / RQ-011 / RQ-012 / RQ-017 / RQ-021 / RQ-022。

## 既存機能と不足

`internal/sourceindex/model.go`にはSource追跡、`internal/consumerimpact/model.go`にはDatabase Release交換時のContract／consumer影響解析があります。Workspaceのload／normalize／validateとDriverも実装済みです。

追加するのはValidation／Capability Bindingの意味差分と業務Scenarioまでの依存伝播です。Database Contractの互換性判定は`consumerimpact`を利用し、その`Safe`を業務変更全体の安全性へ拡張解釈しません。

[ISSUE-000027](000027-ai-proposal-acceptance-boundary.md)はProposal受理・source反映を扱います。本Issueはそこで使うread-onlyな差分・影響解析を所有します。000027側に解析器を重複実装せず、採用判断や書込みは本Issueへ移しません。

Gate Bの必須Evidenceは[ISSUE-000035](000035-gate-b-external-provider-scope-audit.md)で3系統へ整理済みです。主要意味境界の実装とactual runは区別し、本提案を追加Gate条件にはしません。actual Oracleは[ISSUE-000022](000022-external-master-data-physical-query-evidence.md)配下で継続します。Gate C後の実資産から対象を確定します。

## 最初の実装範囲

`order_entry`に限定し、変更対象のsupported kindsは次の二つから始めます。

1. Canonical Validationの宣言・rule変更。既存の評価可能なruleとApplicationからの明示参照を対象とします。
2. Application＋Capability identityで識別されるBindingの交換・追加・削除。業務契約の差分と実現方法の差分を分けます。

両revisionを既存loaderで読み、正規化してidentityごとに追加／削除／変更と変更fieldを計算します。file pathはidentityに使いません。Canonical ID変更をrenameと推測せず、明示情報がなければ削除＋追加とします。未対応kindやparse／validation失敗を空の差分へ変換しません。

依存はCanonical参照から導出し、各edgeに始点・終点・関係種別・導出元field／source・revisionを保持します。ApplicationのValidation参照、PatternのCapability参照、Binding identityを経由して到達先と経路を列挙し、Source Indexへ戻れるようにします。循環や複数経路では重複を制御します。

Scenarioとsubjectは既存Scenarioの明示参照から対応付けます。導出できない箇所は限定fixtureの明示対応と由来を残します。test名の類似やGoコードの文字列検索で依存を断定せず、業務ruleも複製しません。到達先は再検証候補であり、実行による影響確認と区別します。

## 共通解析結果と変更リハーサル

CLIは共通解析結果をJSONと人向けProjectionへ出力します。両者はDerivedで、Canonicalへ逆書きしません。

- before／afterの固定repository revision、source fingerprint、解析方式のversion
- supported kinds、比較した範囲、確定差分、解析失敗・未対応を含むunresolved
- 到達したsubject、依存経路とedgeの由来、探索できた範囲と境界
- 再検証候補Scenario、実行有無、before／after Evidenceへの参照、結果差分

native SQL／Goの内部依存や外部状態など解析できない範囲はcoverageの欠落として表示し、「影響なし」へ丸めません。「検出なし」は確認範囲内の結論に限定します。

二つの設計を同じ入力・時計・fixture条件のDriverで実行します。外部DB、network、process、実Capabilityへ接続しないfixture adapterに限定します。Validation／Authorization／Business Result／Execution Error／Observationを分け、acknowledgementを観測完了とみなしません。

Evidenceはrevision・Scenario・fixture条件へ結び付け、欠落・不一致・古いEvidenceをunknown／未検証とします。synthetic／localとactual provider／external assetを区別し、fixture成功を本番動作の証明にしません。

## 完了条件

- [ ] Gate C baseline、対象二kind、明示依存の取得元を固定する。
- [x] Canonical IDを保ったfile rename／moveだけならSemantic差分がなく、Source locatorだけ変わる。（`changeanalysis`の`TestFileMoveIsASemanticNoOpButALogicalRenameIsASemanticChange`）
- [ ] Validation変更からApplication／Scenarioへ到達し、同じ入力で意図した評価差・command実行可否の差が出る。
- [ ] Binding交換では業務identity／契約を保ち、実現差と再検証候補を検出する。未登録adapterは未検証とする。
- [ ] 到達先の経路・由来を示し、nativeの未知依存と未対応kindをcoverage／unresolvedへ残す。
- [ ] stale／欠落Evidenceはunknownとなり、revision混在を成功に使わない。
- [ ] JSONと人向け出力は同じ結果から生成し、同じ入力で再現できる。
- [x] fixture以外の副作用を呼ばず、authoritative sourceを変更しない。（`changeanalysis`はsourceを読み書きしない純粋な解析。Scenario実行は未実装）
- [ ] [ISSUE-000027](000027-ai-proposal-acceptance-boundary.md)が使う解析入力・結果・未確定状態を文書化し、AI専用解析器を作らない。

## 実装状況

### 最初の解析境界（2026-09-21）

[`tools/internal/changeanalysis`](../../tools/internal/changeanalysis)が、二つのCanonical Worldのread-onlyな差分（subject単位のadded / removed / changed）、World直下collectionの変更有無、追加・変更されたValidationから辿れるexact identityの依存（Attribute → 所有Data → そのDataを指すRelation → Application Pattern、namespaceを利用するApplication）と、解析できなかった範囲（`unresolved`）を返す。file pathをidentityに使わず、Canonical IDの変更をrenameと推測せず削除＋追加として扱う。sourceを読み書きせず、採用判断もしない。

[ISSUE-000027](000027-ai-proposal-acceptance-boundary.md)の受入境界（`tools/reference-shared-acceptance`）がこれを呼ぶ。解析器を受入境界側へ重複実装していない。

未着手: Binding kind、Scenario / Driverへの到達と変更前後のfixture実行、Evidenceのrevision結び付け・stale判定、JSONと人向けProjectionの同一生成、native SQL / Goの未知依存のcoverage表示。

## 規模を抑える理由と非目標

全言語解析や巨大な依存graph基盤は、根拠が薄い推測と実装費用を増やします。二kind、既存loader／Source Index／consumerimpact、限定Scenario、CLIに絞り、差分・依存伝播・検証の接続を先に成立させます。

汎用SQL／Go解析、全kind対応、Migration生成、外部環境への適用、AI interface、Editor UI、書込みAPI、Approval／Acceptance Policyの新設、自動修正、production安全判定は対象外です。

## 関連

- [ISSUE-000038](000038-business-lifecycle-semantics.md): 業務Lifecycle。初回の解析対象はValidation / Bindingに限定し、Lifecycle対応は別の拡張判断とする。
- [ISSUE-000040](000040-semantic-component-composition.md): 組み立て時の依存・追跡情報を共有する。初回実装の必須依存にはしない。
- `plan/adr/0022-ai-proposal-canonical-acceptance-boundary.md`
- `internal/driver/submit_order_gate_scenario_test.go`
