# Gate C Evidence Contract

## 目的

この文書は、判定段階Cの判定に必要な契約を定めます。最初のウェブ実行先で画面が表示できただけでは合格としません。判定段階Bを通過したアプリケーションの意味を実行先へ反映し、その意味・操作・表示品質を後から確かめられる検証根拠として残す必要があります。

判定段階Cでは、ドライバーと最初のウェブ実行先が、アプリケーションの正本にある同じ意味を共有する必要があります。そのうえで、何が実行先に依存せず、何がウェブ固有だったかを説明できなければなりません。

この文書は、ウェブ開発基盤、ブラウザーの自動操作、画面操作エージェント、視覚言語モデル、表示検査製品を標準化するものではありません。実装技術によらず保持すべき検証根拠の意味を定めます。

- Product Acceptance Scenario: **PAS-04 primary**
- inherited baseline context: PAS-02 / PAS-03

関連:

- `plan/gate/GATE_B_TO_C_HANDOVER_CONTRACT.md`
- ISSUE-000007-first-web-target-projection.md
- ISSUE-000028-first-web-target-conformance-evidence.md
- `plan/gate/GATE_C_FIRST_WEB_TARGET_EVIDENCE_RUNBOOK.md`
- ISSUE-000026-target-capability-execution-profile-boundary.md
- GitHub Issue #70 / #259 / #270 / #291

## Gate CはHandover Baselineから開始する

最初のウェブ実行先の構築は、曖昧な「最新のドライバー」を基準に開始しません。

判定段階Bの合格時に固定した引継ぎ基準版を`B`、そのリポジトリの版を`R`とします。

```text
Gate B Ready Evidence
        ↓
Handover Baseline B
  └─ baselineRepositoryRevision = R
        ↓
First Web Target implementation
  └─ targetBuildRevision = T
        ↓
Gate C Evidence
```

通常、`T`にはウェブ実行先の実装コミットが含まれるため、`R`より後の版になります。そのため、判定段階Cの検証根拠では、実行先の構築版に`repositoryRevision = R`を強制しません。

代わりに、次を分けて追跡します。

### Handover Baseline identity

- Handover Baseline ID
- `baselineRepositoryRevision = R`
- アプリケーション・パターン・データ・関係・問い合わせ・能力など、正本への参照
- Driver Scenario identity
- 判定段階Bの検証根拠への参照
- unresolved

### Target build identity

- `targetBuildRevision = T`
- 実行先の変換器とホストの識別情報
- build / run identity
- 実行先で実際に利用した正本上の対象

### RとTの関係

`T`でウェブ用の変換器、ホスト、派生した表現などを追加しても、引継ぎ基準版が表すアプリケーションの意味を暗黙のうちに変更しません。

判定段階Cの途中で`order_entry`など基準版の対象の意味を変更する必要が生じた場合は、次のように扱います。

1. ウェブ固有の項目や規則で不足を補わない
2. 正本となる入力元へ戻って修正する
3. 必要なドライバーと判定段階Bの検証根拠を再確認する
4. 新たな引継ぎ基準版`B'`と、そのリポジトリの版`R'`を発行する
5. 以後の判定段階Cの検証根拠を`B'`へ結び付ける

旧版`B`の検証根拠を、新しい基準版でも成功した根拠として暗黙のうちに読み替えません。

## Gate Cで分ける三種類のEvidence

最初のウェブ実行先の評価を、一つの点数や「成功」という一つの結果にまとめません。

```text
Handover Baseline B / meaning at R
                │
                ├─ Semantic Conformance
                │
Target build T  ├─ Interaction / Functional Evidence
                │
                └─ Visual Fidelity Evidence
```

### 1. Semantic Conformance

ウェブ実行先が、引継ぎ基準版で定めた正本上の意味を保っているか確認します。

少なくとも次を対象にします。

- アプリケーションパターンの識別情報
- データ・関係・問い合わせ・データ接続先の識別情報
- 読取り専用と読書き可能の境界
- 検証規則
- 認証と認可
- 能力とその対応付け
- 業務上の結果と名前付きの結果
- 実行時のエラー
- 命令と観測
- 入力元から正本への追跡関係

意味の準拠確認は、判定段階Cの必須条件です。

DOM、経路、HTTPの状態コード、画面部品名などの一致だけで判定しません。引継ぎ基準版に含まれるアプリケーションの意味との対応を確かめます。

### 2. Interaction / Functional Evidence

利用者が実行先の構築版`T`で、参照シナリオを実行できるか確認します。

ブラウザーの自動操作、画面操作エージェント、人による再現手順などは実行手段です。どの手段でも、実行した引継ぎ基準版とアプリケーションの意図へたどれる必要があります。

#### Replayable Behavior

機能の検証根拠は、一度取得した画面の状態や最終的なDOMだけではありません。必要に応じて、再実行できる操作として残せます。

```text
Handover Scenario / Semantic Operation
        ↓
behavior precondition
        ↓
Target-specific interaction realization
        ↓
Business Result / Execution Error / Observation
        ↓
Functional conformance result
```

操作を実行先固有の選択子、経路、座標、画面部品の識別子などで実現しても、それらを対話の正本上の識別子にはしません。

一つの操作が成功しても、複数の機能にまたがる一連の業務手順全体の成立を代わりに証明したことにはなりません。参照シナリオが複数段階の操作を求める場合は、その手順を保った再実行も検証根拠として確認します。

異なるブラウザー自動操作、画面操作エージェント、実行先で得たイベントを比較するため、正規化したイベントや操作の記録を派生情報として作ることはできます。ただし、実行先で得た元の検証根拠へたどれる来歴を保持し、正規化によって失われる実行先固有の違いを成功扱いにしません。汎用イベントモデルを判定段階Cの前提にはしません。

### 3. Visual Fidelity Evidence

配置、まとまり、読みやすさ、視覚的な優先順位、操作結果の伝わりやすさなど、実行先としての表示品質を記録します。

表示の忠実度は重要ですが、意味や機能の検証根拠の代わりにはしません。

視覚言語モデル、画像の類似度、画面画像の差分などを使っても、その評価点だけで意味の準拠確認やシナリオの成立を合格にしません。

## Scenario Matrix

少なくとも次の項目を、引継ぎ基準版にあるドライバーのシナリオと対応付けます。

| Scenario | Semantic確認 | Functional確認 |
| --- | --- | --- |
| browse | 同じbrowse Pattern / Query / Datasource | 一覧を取得・閲覧できる |
| record open | 同じrecord Pattern / Data | 一件を開ける |
| reference lookup | 同じRelation / read-only Datasource | 参照候補を検索・選択できる |
| header-detail read | 同じ1:N Relation | HeaderとDetailを閲覧できる |
| header-detail edit | baselineにwrite意味がある場合、同じRelation / Validation | 許可された範囲だけ編集できる |
| validation satisfied | 同じCanonical Validation | 正常入力が次へ進める |
| validation not satisfied | 同じCanonical Validation | 不成立理由を返しCommandへ進まない |
| authorization allowed | 同じAuthorization Capability | 許可された操作を実行できる |
| authorization denied | 同じAuthorization意味 / Outcome | Execution Errorへ畳まず扱える |
| command | 同じcommand Pattern / Capability Binding | 明示Capabilityをinvokeできる |
| business result | 同じoutput / named Outcome | Target表現へProjectionできる |
| execution error | 同じInvocation境界 | 業務Outcomeと区別して扱える |
| observation | 同じObservation意図 / read境界 | acknowledgementとObserved stateを区別できる |

引継ぎ基準版にないシナリオの意味をウェブ側だけで追加し、確認範囲の不足を隠しません。

## Negative / Architecture Conformance

正常な操作だけで判定段階Cの合否を決めません。

### Canonical不変

ウェブ用の変換器を外しても、意味の正本が変わらないことを確認します。

ウェブ固有の画面部品、経路、HTTPの状態コード、DOMの選択子、CSSのクラスなどを意味の正本の層へ追加しません。

### 二重実装を作らない

次をWeb Target内へ独立正本として複製しません。

- native SQL
- Capability implementation
- 検証規則
- Authorization rule
- Business Outcome定義
- Observation対象の意味

### Result境界を保つ

少なくとも次を分けます。

```text
Business Result
Execution Error
Observation
Developer Diagnostic
```

HTTP status、toast、dialog、inline message等は表現方式です。transport / UI表現からCanonical意味を逆定義しません。

### Command acknowledgementとObservationを分ける

Commandが正常なBusiness Resultを返しただけで、外部状態が成立済みとみなしません。

Observationが必要なScenarioではCommand EvidenceとObservation Evidenceを別々に残します。

### Datasource境界を破らない

read-only DatasourceをTarget都合でwrite可能にしません。browser storageやform stateもCanonical Datasourceのwrite authorityにはしません。

### Standard ProjectionとCustom UI

少なくとも一箇所でCustom UIの接続を確認します。

Custom UIでもApplication Pattern、Validation、Authorization、Capability、Result、Observation等の上位意味へ接続し、独自の業務正本を作りません。

## Evidence Recordの最小項目

最初から最終JSON schemaを固定しません。ただしEvidence artifactから少なくとも次を復元できる必要があります。

### Baseline provenance

- Handover Baseline ID
- `baselineRepositoryRevision = R`
- Driver Scenario identity
- Gate B Evidence reference
- baseline unresolved

### Target provenance

- `targetBuildRevision = T`
- 実行先の変換器とホストの識別情報
- build / run identity
- Reference Workspace / Canonical subject identity

### Canonical references

- Application ID
- Pattern ID
- Data / Relation / Query / Capability等、Scenarioで参照したCanonical ID
- 必要なSource Reference

### Runtime / Target observations

- observed Target Capability
- observed Runtime Binding
- execution artifact / ABI / Host identity / observations（適用可能な範囲）
- observed Execution / Trust条件
- interaction steps
- replayable behavior identity / precondition / expected semantic operation（利用する場合）
- raw Target Evidence reference
- normalized event / behavior Evidence referenceとraw provenance（利用する場合）
- Business Result / Execution Error / Observationの分類

### Conformance result

- Semantic Conformance result
- Functional result
- Visual Evidence reference
- performance measurement reference
- Diagnostic / unresolved

### Evaluator information

自動評価を使った場合は少なくとも次を追跡します。

- evaluator / automation tool identity
- version
- 入力artifact
- evaluatorが判定したEvidence種別

prompt、trajectory、score等を保存してもCanonical meaningへ含めません。secret、credential、不要なprivate dataを通常Evidenceへ保存しません。

## Evidence artifactはCanonicalではない

Gate C Evidenceはderivedな検証成果物です。

```text
Authoritative Source / Handover Baseline
          ↓
Driver / Target execution
          ↓
Evidence artifact
```

Evidence artifactからCanonicalへ暗黙に逆書きしません。

Screenshot、DOM snapshot、browser trace、VLM judge結果、GUI agent trajectory、performance trace等は再確認や比較に利用できますが、source of truthにはしません。

## Target / Runtime Binding差の引き継ぎ

第一Web Targetだけで完全なtaxonomyを作りません。

Gate Cでは観測した差を次へ整理し、`ISSUE-000026` / GitHub Issue #259へ渡します。

1. Application上の意味
2. Target-neutral Presentation候補
3. Target Capability候補
4. Runtime Binding差
5. execution artifact / ABI / Host差（観測可能な範囲）
6. Execution / Trust差
7. Web固有実装
8. Host control plane固有差
9. Canonicalへ持ち上げてはいけない差
10. unresolved

分類できないものは無理に既存bucketへ押し込まず`unresolved`として残します。

Gate Cでは、複数execution artifactを実装・比較すること自体は要求しません。第一Web Targetで実際に使ったartifact / ABI / Host identityと、観測できた実現差をEvidenceへ残せればよく、native Go対Wasm等の比較はGate C後のportable artifact ReferenceでPressureします。

## Visual Fidelity

初期Referenceでは一律の数値閾値を置きません。

少なくとも次を比較可能にします。

- field groupingの可読性
- Relation / referenceの理解しやすさ
- header-detailの局所性
- validation feedbackの識別性
- authorization denialとsystem errorの識別性
- Business Result / Observationの識別性
- standard ProjectionとCustom UIの差

Screenshotまたは同等の再確認可能なartifactを優先して残します。VLM等のjudge結果だけを保存して入力artifactを捨てません。

## Performance Evidence

RQ-024の最終performance contractは第二Target以降の比較から決めますが、第一Web Targetから測定点を用意します。

- initial projection / load
- browse result render
- record open
- reference lookup
- validation
- authorization
- command response
- Business Result projection
- 実行時のエラー projection
- observed completion

根拠のない一律閾値は第一Targetだけから固定しません。

## Gate C判定

Gate CはIssue closeやWeb画面の存在だけでは通過しません。

少なくとも次をEvidenceから説明できることを要求します。

- すべてのGate C runが明示Handover Baseline `B` / revision `R`へ追跡できる
- Target build revision `T`をbaseline `R`と混同していない
- `T`でbaseline subjectの意味を変更した場合、新Handover Baselineへ切り替えている
- DriverとWebが同じApplication / Pattern meaningを共有している
- Semantic Conformanceが成立している
- Reference ScenarioのFunctional Evidenceが揃っている
- replay可能なbehaviorを利用する場合、behaviorがHandover Scenario / semantic operation / raw Target Evidenceへ追跡できる
- Visual Fidelityが別Evidenceとして残っている
- Result / Execution Error / Observationの境界が保たれている
- Web固有実装をCanonicalへ逆流させていない
- Web Adapterを除去してもCanonical Worldが変わらない
- standard ProjectionとCustom UIが同じ上位意味へ接続できる
- Target Capability / Runtime Binding / execution artifact・ABI・Host / Execution・Trust差を第二Targetへ渡せる
- unresolvedを成功へ読み替えず明示できる

Gate C通過後、`ISSUE-000008`のEditor / AI支援保留解除と、`ISSUE-000026`の第二Target圧力試験を判断します。

判定記録: [`GATE_C_READINESS_AUDIT.md`](GATE_C_READINESS_AUDIT.md)（第一Web Target、2026-09-19、Ready）

## 非目標

- 第一TargetだけでTarget-neutral Presentation modelを完成させること
- Web frameworkをTEI標準として固定すること
- 完全なTarget Capability taxonomy
- Gate Cのためだけに複数execution artifactやWasmを実装・比較すること
- execution artifact / ABI / Hostのproduction taxonomy / schema
- pixel-perfect design system
- VLM / GUI agentの標準化
- visual scoreによる単一合否
- 第二Targetの実装
- Evidence artifactのCanonical化
- Target build revisionをHandover Baseline revisionと同一に固定すること
