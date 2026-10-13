# Gate B → Gate C Handover Contract

## 目的

判定段階Bを通過したアプリケーションとドライバーの意味の基準版を、最初のウェブ実行先と判定段階Cへ、曖昧さを残さず引き継ぎます。

判定段階Cでは、「最新のドライバー」を比較対象にはしません。判定段階Bで合格とした特定のリポジトリの版`R`を、引継ぎ基準版`B`として固定します。

```text
Gate B Ready for R
        ↓
Handover Baseline B
  baselineRepositoryRevision = R
        ↓
Target build T
        ↓
Gate C Evidence
```

引継ぎは、新しい正本のスキーマでも第二の正本でもありません。Gitの版、既存の正本上の識別情報、ドライバーのシナリオ、判定段階Bの検証根拠、未解決事項への参照をまとめる、検証上の契約です。

製品の受入シナリオでは、PAS-02を主な対象、PAS-03を副次的な対象とします。

## 現行Gate B scope

判定段階Bで必須となる検証根拠は、次の三つです。

1. #310 リポジトリの決定論的な検証根拠
2. #255 生成成果物の再現性に関する検証根拠
3. #308 実際のPostgreSQL提供実装と一時的な試験用環境の検証根拠

三つとも、変更のない同じ候補版`R`を対象に取得します。

```text
#310 ─┐
#255 ─┼─ repositoryRevision = R
#308 ─┘
       ↓
#387 final convergence
       ↓
same-run child Evidence artifacts
       ↓
#273 Gate B Readiness Audit for R
       ↓ Ready only
Handover Baseline B / R
```

実際のOracleに関する#214・#371・#376は、外部資産の検証根拠として引き続き重要です。ただし、第四の必須検証経路にはしません。未完了であれば`unresolved`として引き継ぎます。

## 基本原則

### Handoverは参照であり第二の正本ではない

```text
Authoritative Source / Canonical Application
                 │
                 ├─ Driver execution / Evidence
                 │
                 └─ Handover Baseline B
                       └─ revision / Canonical identity / Evidence reference
```

引継ぎ記録へアプリケーションの意味を複製したり、その記録から正本へ書き戻したりしません。

### 一つのbaseline revisionへ揃える

異なる版で得た成功の検証根拠を寄せ集め、単一の合格版での結果として読み替えません。

- #310 `repositoryRevision = R`
- #255 `repositoryRevision = R`
- #308 `repositoryRevision = R`
- 全lane `reproducibleRevision = true`
- #273で同じ版Rを合格と判定

ソースの変更が必要になった場合、旧版Rの成功を新しい版へそのまま引き継ぎません。新しい版で必要な検証根拠を取り直します。

### same-Rだけでなくsame-run provenanceを要求する

引継ぎに使用する#310・#255・#308の詳細な検証根拠は、同じ版Rで成功した任意の三つのJSONではありません。

**#387の最終確認が同じ実行で実際に読み取り、保存した子処理の検証成果物そのもの**を、必須の検証根拠として参照します。

```text
#387 final summary
  repositoryRevision = R
  allLanesSameRevision = true
  childArtifactsRetained = true
        │
        ├─ deterministic-repository.json
        ├─ generated-artifact-reproducibility.json
        └─ real-postgres-ephemeral-fixture.json
              ↓
        #273 Ready Audit for R
              ↓
        Handover B / R
```

同じ版Rで後から再実行した別の成果物を、#387が実際に読み取った成果物と同一だと推測しません。

### Target側の不足をbaselineへ逆輸入しない

ウェブの実装中に必要となったアプリケーションの意味がB・Rにない場合、ウェブ固有の項目や独自の規則で補って判定段階Cを通過させることはしません。

アプリケーションやドライバーへ戻り、正本となる入力元を変更します。そのうえで、新しい版について必要な判定段階Bの検証根拠を再確認し、後継の引継ぎ基準版を発行します。

## Handover Baseline Bに必要な情報

具体的なJSONスキーマやファイル形式は、判定段階Bの通過前には固定しません。ただし、少なくとも次の情報を復元できる必要があります。

### Baseline identity

- 引継ぎ基準版の識別情報`B`
- `baselineRepositoryRevision = R`
- 参照用作業領域の識別情報
- 判定段階Bの合格監査の対象版

### Canonical Application baseline

最初の参照事例では、少なくとも次の識別情報へたどれるようにします。

- Application `order_entry`
- Application Pattern
  - browse
  - record
  - reference
  - header-detail
  - command
- Data / Relation / Query
- Datasource
- Validation
- Authentication / Authorization
- Capability / Binding
- 業務上の結果と名前付きの結果
- Execution Error
- 観測に使用する明示的な読取りの境界
- 入力元と対応付けを追跡できること

引継ぎで列挙するのは参照先の一覧であり、意味の本文を複製するものではありません。

### Driver Scenario baseline

少なくとも次のシナリオへたどれるようにします。

```text
Authentication
  -> browse
  -> record
  -> reference Customer
  -> header-detail OrderLine[]
  -> Canonical Validation
  -> Authorization
  -> Command Business Result
  -> separate Observation read
```

業務上の結果、名前付きの結果、実行時エラー、観測、診断結果の違いを、実行先へ引き継ぎます。

### Datasource / authority boundary

少なくとも次を保持します。

- 読取り専用と読書き可能の違い
- 外部から観測したものか、ここで管理するものかなど、検証根拠に必要な所有・管理上の性質
- データベースの実体と実行器の能力との対応付け
- 秘密情報・認証情報・実行時の引数を通常の検証根拠に保存しない制約

## Gate B Evidence provenance

引継ぎ記録へ検証根拠の本文を複製しません。安定した参照先・識別情報・照合値から、実際の成果物へたどれるようにします。

### #387 final convergence

引継ぎ記録から、合格判定に使用した#387の最終確認の成果物へたどれる必要があります。

少なくとも:

- `evidenceNature = gate-b-three-lane-final-convergence`
- `repositoryRevision = R`
- `reproducibleRevision = true`
- `allLanesSameRevision = true`
- `childArtifactsRetained = true`
- final source check成功
- Go environment安全条件成立

を確認済みのartifactを参照します。

### same-run child exact artifacts

#387が保存した三つの子処理の成果物を、必須の検証根拠として参照します。

- `deterministic-repository.json`
- `generated-artifact-reproducibility.json`
- `real-postgres-ephemeral-fixture.json`

各artifactについて少なくとも:

- 成果物への安定した参照
- 固定したファイル名と検証経路の識別情報
- `evidenceArtifactFingerprint`
- `repositoryRevision = R`
- `reproducibleRevision = true`

へ戻れる必要があります。

#387の実行器は最終的な成功判定の直前にも保存ファイルを読み直し、照合値を確認します。さらに#273の合格監査でも保存成果物を独立して照合します。実行器内の確認だけでは合格を宣言しません。

### #310 deterministic Repository

同じ実行で得た`deterministic-repository.json`から、少なくとも次の結果へたどります。

- 作業領域の検証に成功
- `final-ui-less-scenario`成功
- `external-evidence-intake`成功
- `repository-evidence-source-boundary`成功

合成データから得た検証根拠を、実際の外部資産で得たものとして格上げしません。

### #255 generated artifact reproducibility

同じ実行で得た`generated-artifact-reproducibility.json`から、少なくとも次の結果へたどります。

- 意味の正本とプラグインの実装固定情報が期待出力と一致
- 永続的な実行記録が期待出力と一致

`-write`による修復操作や、修復直後に変更の残る作業ツリーを、完了の検証根拠にはしません。

### #308 real PostgreSQL Provider

同じ実行で得た`real-postgres-ephemeral-fixture.json`から、少なくとも次の結果へたどります。

- 試験用構成要素の作成前に行う安全確認
- 2件の取得・正常に0件を取得・表が存在しない場合の実行時エラー
- 意味上の表現への明示的な対応付け
- 補助列を外部に露出しないこと
- データベース側の所有権を確かめてから後片付けすること
- 後片付け後、利用者が定義した関係0件・イベントトリガー0件・試験用構成要素なし

本番の既存データベースに関する検証根拠とは読み替えません。

### #273 Gate B Readiness Audit

#273と`GATE_B_READINESS_AUDIT.md`によって、同じ版Rについて#387の最終結果と同一実行内の子処理の成果物を監査し、合格と判定したことを参照します。

最終確認の実行器の終了コードだけでは合格と判定しません。

## actual Oracleの扱い

#214・#371・#376は、必須の検証根拠には加えません。

取得済みで、B・Rについて検証済みの外部資産の根拠として十分であれば、追加で参照します。その際、少なくとも次を区別します。

- 実際に観測した成果物
- 検証根拠の照合値
- ソースの版を固定した受理記録
- 実際の読取りの検証根拠
- 実行環境・実現方式・リポジトリの版との関係

を区別します。

未取得、実際の読取りが未完了、またはRとの結び付きが不十分な場合は、`unresolved`として残します。

最初のウェブ実行先側で、Oracleの物理的な構成要素・列・SQL・データ型・ドライバーの動作・読取り専用の方式を推測して補いません。

## Gate C Evidenceからのtrace

判定段階Cの検証根拠では、意味の基準版と実行先の構築版の識別情報を分けます。

```text
Gate C Evidence
  ├─ Handover Baseline B
  │    └─ baselineRepositoryRevision = R
  │         └─ #273 Ready Audit
  │              └─ #387 final convergence
  │                   └─ same-run #310 / #255 / #308 exact artifacts
  │
  └─ Target build identity T
```

通常は`T != R`です。

判定段階Cの検証根拠から、少なくともB・R・Tを追跡できる必要があります。

## baseline変更時

### Target実装だけの変更

CSS、DOM、ウェブ用の変換器、ブラウザーの自動操作などだけが変わり、正本やドライバーの意味が変わらない場合は、B・Rを維持できます。

### Evidence / Diagnosticだけの変更

評価方法や診断結果の変更が、判定段階Bの完了の検証根拠の意味に影響する場合は、該当する必須の検証をやり直します。旧版の検証根拠を、新しい基準版での成功として暗黙のうちに読み替えません。

### Application / Driver meaning変更

検証規則、関係・問い合わせ・データ接続先、能力とその対応付け、業務上の結果、観測の読取り境界、パターンの意図などを変える場合は、次の手順を踏みます。

1. 正本となる入力元を変更
2. ドライバーと検証規則を再確認
3. 必要な判定段階Bの検証根拠を新しい版`R2`で取得
4. #387を`R2`で再実行し、同一実行内の子処理の成果物を取得
5. #273で`R2`を再監査
6. 合格した場合に後継の引継ぎ基準版`B2 / R2`を発行
7. 判定段階Cで影響を受けるシナリオを再実行

旧版のB・Rが表す意味を、暗黙のうちに上書きしません。

## Handover record

判定段階Bの合格後、最初の引継ぎ記録を作成します。

初めから大きな汎用スキーマにはしません。Markdownと安定した参照先を組み合わせても、必要性を確認できれば小さな機械可読の成果物にしても構いません。

最低条件:

- `B / R`を固定
- 正本上の識別情報を追跡できる
- ドライバーのシナリオを追跡できる
- #387の最終確認を追跡できる
- #387が同じ実行で取得した#310・#255・#308の成果物そのものを追跡できる
- 子処理の成果物の照合値を再確認できる
- #273の合格監査を追跡できる
- 任意の外部資産の検証根拠と`unresolved`を必須の検証根拠から分離
- 秘密情報を含まない
- 実行先固有の情報を正本の基準版へ混ぜない

## Gate C開始条件

最初のウェブ実行先の本格的な実装は、少なくとも次の条件を満たしてから開始します。

- Gate B Ready
- Handover Baseline `B / R`を固定
- #310・#255・#308で同じ実行により得た必須の検証根拠がRについて揃っている
- #387の最終確認と、同一実行内の子処理の成果物を追跡できる
- #273で同じ版Rを合格と判定
- 判定段階Cの対象となる正本上の識別情報を列挙できる
- ドライバーのシナリオの基準版を再現できる
- 実際のOracleなどに関する外部の検証根拠を、検証済みか未解決か明示できる
- 未解決事項を明示できる

引継ぎの確認は、新しい番号付きの判定段階ではありません。判定段階Bの出力を判定段階Cの入力として固定する契約です。

## 非目標

- 新しいCanonical Application schema
- Handover machine-readable schemaの先行固定
- Presentation IRの先行設計
- Web frameworkの選定
- Handover artifactからCanonicalへの逆生成
- Git commitだけをSemantic Conformanceの証明とすること
- same-R別runを#387 same-run Evidenceへ読み替えること
- actual Oracleを第四Required laneへ戻すこと
- external-observed未観測事実をsynthetic / PostgreSQL Evidenceから推測すること
- 第一TargetだけでTarget-neutral taxonomyを完成させること

## 関連

- `GATE_B_READINESS_AUDIT.md`
- `GATE_B_EVIDENCE_ACQUISITION.md`
- `GATE_B_EVIDENCE_RUNBOOK.md`
- `GATE_C_EVIDENCE_CONTRACT.md`
- `UPSTREAM_ACCEPTANCE_SCENARIO_TRACEABILITY.md`
- `V1_REFERENCE_APPLICATION_PLAN.md`
- ISSUE-000007-first-web-target-projection.md
- ISSUE-000028-first-web-target-conformance-evidence.md
- ISSUE-000029-gate-b-readiness-audit.md
- ISSUE-000030-gate-b-to-c-handover-baseline.md
- ISSUE-000035-gate-b-external-provider-scope-audit.md
- ISSUE-000037-gate-b-three-lane-final-evidence.md
