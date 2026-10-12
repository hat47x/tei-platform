# Gate C → Second Application Reference Start Packet

- 状態：進行中。S0〜S3完了／S4は全体検証の事前確認後に改訂／S5はレビューの再生成待ち
- 日付：2026年9月19日
- 開始後の主担当：ISSUE-000040
- 入力：ISSUE-000047／ISSUE-000053の第5段階／ISSUE-000027／ISSUE-000039
- 判定段階より前の契約：
  - `plan/workstreams/KNOWLEDGE_PORTABILITY_R0_BOUNDARY_2026-09.md`
- 判定段階CのReady前に正本のアプリケーションへ与える影響：なし

## 1. 目的

判定段階CのReady後に第二の参照アプリケーションへ着手する際、探索・知識の移植可能性・構成案・受理を別々の作業正本に分けません。ISSUE-000040の一つの作業として継続できるようにします。

この文書は、第二アプリケーションのソースではありません。

```text
Start Packet
  != Gate C Ready
  != Application Proposal
  != Shared Acceptance
  != Canonical source
```

## 2. Entry Gate

開始前の機械的な確認には、次を正本として用います。

- `plan/gate/GATE_C_SECOND_APPLICATION_ENTRY_PREFLIGHT_2026-09.md`
- `tools/reference-business-procedure-check/gate_c_second_application_entry_preflight_pressure_test.go`

第二アプリケーションの作業を開始するには、判定段階CのReadyを**明示的な検証根拠**で確認します。

計画文書の見出しや将来の予定からReadyと推測しません。

開始時には、次の情報を一つの開始記録に固定します。

```text
gateC.status = ready

baseline:
  handover baseline B
  baseline repository revision R

first target:
  Web build revision T1

evidence:
  semantic conformance = <ref>
  interaction / functional = <ref>
  visual fidelity = <ref>

target differences:
  <explicit summary>

unresolved:
  <remaining items>
```

### Stop

次のいずれかに該当する場合、第二アプリケーションのソースの記述へ進みません。

- 判定段階Cの状態がReadyではない
- `R`と`T1`を区別できない
- 意味の検証根拠がない
- 機能の検証根拠がない
- 表示の検証根拠がない
- 実行先の違いが整理されておらず、第二アプリケーションの設計へ混入する可能性がある

表示の検証根拠だけで開始条件を満たしたとは扱いません。

## 3. Development-flow / workstream continuity

この文書が定める単独開発の進め方では、ブランチやPRを製品の状態やレビューの境界として使わず、mainへ直接コミットします。

第二の参照アプリケーションの作業の継続性は、ブランチではなく、ISSUE-000040、構成案の照合値、ソースと検証根拠、受理の状態によって管理します。

原則は次のとおりです。

```text
current main
  ↓
ISSUE-000040 qualification / portability
  ↓
composition proposal
  ↓
Shared Acceptance boundary
  ↓ Accept後のみ
authoritative source
  ↓
reload / validation
  ↓
Environment / Driver / Target evidence
  ↓
change propagation Evidence
```

### 禁止

- 探索・移植可能性・パターン・環境・ドライバーごとにリモートブランチを作ること
- 同じ第二アプリケーションの正本候補を複数の作業系列で並行して記述すること
- ブランチ名やPRの状態を受理や正本の権限とみなすこと
- 並行するセッションが保持する古いmainの状態から、そのまま変更を適用すること
- 健全性確認をせずにコードの変更をmainへ入れること

各作業ではpushの直前に現在のmainへのrebaseまたは最新状態の確認を行い、コードを変更する場合はリポジトリで定めた健全性確認を通します。文書のみの変更でも、リンク、状態、判定段階と検証根拠に関する主張、正本との関係を確認します。

## 4. First Reference scope

最初の実際の参照事例は、読取り専用の顧客検索に限定します。

### Shared authority candidates

- `customer.customer`
- `customer.customer_id`
- `master_data.customer ^2`
- `master_data.customer/find_by_id`

### Destination-owned decisions

判定段階Cの後で初めて決定します。

- 正本上のアプリケーションID
- 業務上の文脈
- 移行先の関係
- 移行先のパターンID
- データ接続先の別名
- 認可の要件
- 最初の実行先における表示・操作面

### Environment-owned decisions

- 開発環境での対応付け
- 本番環境での対応付け
- 実際に必要となる場合の資源・権限

### Out of scope

- 顧客データの書込み
- 顧客データの保守管理権限
- 新たな書込み契約
- 書込みの能力
- 二つ目の実行先
- 汎用パッケージ管理機構
- 移植可能性に関する継承・上書き
- リポジトリをまたぐ同期

## 5. Work sequence

### S0 — Gate C entry capture

判定段階Cの検証根拠を開始記録に固定し、開始前の確認で`entry-evidence-complete`であることを確かめます。

この事前確認は判定段階Cの判断権限そのものではありません。受理済みの検証根拠が、第二アプリケーションの開始に必要な形で揃っているかを確かめます。

Output:

- Gate C Evidence refs
- `R`
- `T1`
- target difference summary
- unresolved

ソースは変更しません。

### S1 — ISSUE-000047 candidate qualification

目的：

> destination Business contextからCustomerをIDで参照したい。

現在のリポジトリから、候補を読取り専用で探索します。

最低限の対象：

- 顧客の意味上の定義
- 顧客IDの型
- 顧客に関する契約
- 契約に基づく問い合わせ
- 元となる参照パターンの来歴

判定：

- applicable
- incompatible
- unknown

結果には、権威ある入力元・発見の来歴・適格性の判断理由・不足条件を記録します。

### S2 — Phase 5 portability relation

S1で得た候補を次のように分類します。

- linked reuse
- conditional linked reuse
- adopted local variation
- rebind-required

保持する情報：

- 管理権限の所有者
- 依存関係の対象
- 来歴
- 情報の鮮度を判断する入力
- 移行先の要件

`applicable`を`accepted`として扱いません。

### S3 — Destination decision packet

次の未決定事項について、明示的に判断します。

1. アプリケーションID
2. 業務上の文脈と関係
3. パターンID
4. データ接続先の別名
5. 認可の要件
6. 開発環境と本番環境の対応付け方針
7. 最初の実行先における表示範囲

未決定の項目は`unknown`として残し、元の参照事例から推測して補いません。

### S4 — ISSUE-000040 composition candidate

既存の文法だけを使って構成案を作成します。

概念上の形：

```text
application <destination> {
    uses semantic customer

    database <destination-alias> {
        target master_data
        access read
        requires {
            contract master_data.customer ^2
        }
    }
}
```

参照するパターンは移行先が所有します。

元の`order_entry.customer_reference`を複製して第二の正本にはしません。

### S5 — Shared Acceptance

構成案の候補を、ISSUE-000027／ADR-0022で定めた受理の境界へ渡します。

受理前の禁止事項：

- 権威ある入力元を変更しない
- 環境の対応付けを作成しない
- AIの構成案を受理済みの判断とみなさない

受理した後に限り、ソースの記述へ進みます。

### S6 — Source / reload / validation

受理した構成案を通常のTEIソースとして記述します。

確認する事項：

- AIや探索用の索引がなくても再読込みできること
- 同じ正本上の意味を再構成できること
- 検証規則とソース索引が成立すること
- 移植可能性専用の文法は不要という仮説を再評価すること

### S7 — Environment Binding Proposal

アプリケーションのソースを受理した後、環境側を別の権限領域として扱います。

事前確認：


当時のソース確認では、開発環境と本番環境にOracleの既存インスタンス`master_data.dev`と`master_data.prod`がありました。そのため、最小の候補は移行先が所有する別名を各インスタンスに結び付ける、二つの対応付けです。

```text
development:
  bind repair_intake.customer_master -> master_data.dev

production:
  bind repair_intake.customer_master -> master_data.prod
```

新しいデータベースの実体、Oracleプラグイン、提供実装の一式を作成する必要性を示す検証根拠はありません。

確認する事項：

- 移行先のアプリケーションと別名
- 対象となるデータベースの実体
- データベースの定義
- 提供実装の解決
- 権限と資源の要件
- 対応付けによって権限・資源・実行時の権限を暗黙に付与しないこと

元の`order_entry.master`の対応付けを、複製の正当化に使う権限とはみなしません。

### S8 — Driver / First Web Target

事前確認：


読取り専用の顧客検索を、

1. Driver
2. 第一Web Target

これらで再現します。

開発環境には`master_data.customer/find_by_id`の契約に基づく問い合わせの実行器がすでに対応付けられています。そのため、合成データを使う決定論的なドライバーのシナリオは既存の境界で検証できます。一方、本番環境には同等の実行器の対応付けがないため、開発環境の設定を暗黙のうちに流用しません。

当時の最初のウェブ実行先では、派生表示だけを`-application`で切り替えられますが、読取りシナリオの実行器は`order_entry`に固定されていました。そのため、`repair_intake`の一覧情報と`order_entry`の実行が混在する可能性があります。S8では、この状態を安全側に停止させます。顧客IDの入力から`repair_intake.intake.customer_id`に対応付けた行を経て、ドライバーの参照用検索へ進む、実行先固有の読取り画面も追加して検証します。

同じ業務上・意味上の内容を実行先側で再定義しません。

S8で実行先を検証するときは、過去の`B / R`をそのまま再利用しません。S8aで`repair_intake`を含むドライバーの検証根拠を取得した後、引継ぎ契約の後継版に関する規則に従い、`B2 / R2`を固定します。最初のウェブ実行先の構築版は、別の識別情報`T2`として追跡します。

### S9 — Change propagation Evidence

少なくとも次の変化に対する影響を検証します。

- 共有する意味の定義を変更 → 依存関係の適格性を再確認
- 契約の互換性を保つ小さな変更 → 互換性を再評価
- 契約の大きな変更 → 依存関係を再評価
- 元のパターンの変更 → 来歴とのずれを確認
- 移行先のパターン変更 → 元の定義へ書き戻さない
- 元の別名や環境の対応付けを変更 → 移行先へ自動伝播しない
- 無関係な元の定義の変更 → すべてを古い状態とは判定しない

ISSUE-000039の機能を利用できる場合は、共通の影響分析を使います。

ISSUE-000040専用の影響分析機構は作りません。

## 6. Stop conditions

作業中に次の状況が発生した場合、次の段階へ自動的には進みません。

### Candidate unknown

必要な検証根拠を追加で取得します。

元の定義にある選択内容を推測して補いません。

### Candidate incompatible

別の候補の検討、または移行先の設計変更に戻ります。

意味上の識別情報そのものまで、安易に互換性なしと判定しません。

### New schema required

既存の文法では表現できないことを示す具体的な検証根拠を記録します。

先回りして知識パッケージ・継承・上書きの機構を作りません。

### Authorization unresolved

読取りの経路が分かっても、実行権限があるとは推測しません。

### Environment unresolved

アプリケーションが受理されたことだけを根拠に、対応付けを自動生成しません。

### Target semantic divergence

実行先の実装上の都合で、正本上の意味を暗黙のうちに変更しません。

必要であれば基準版や受理済みのソースの変更手順へ戻ります。

## 7. AI Builder handoff

AIによる構築支援へ渡す文脈には、少なくとも次の情報を含めます。

```text
Gate C entry
  R / T1 / Evidence

intent
  read-only customer lookup

shared candidates
  identity / authority / qualification

local decisions
  accepted / unresolved

provenance
  origin Pattern / source

portability
  reuse relation / freshness

acceptance
  current stage

implementation
  allowed source kinds
  forbidden shortcuts
```

AIにリポジトリ全体を無差別に読ませる方法を、標準の入口にはしません。

## 8. Human review handoff

人によるレビューでは、生成コードより先に次の情報を提示します。

- intent
- shared authority
- destination-owned meaning
- Environment-owned choices
- unresolved
- impact candidates
- exact source refs
- Acceptance status

これらを提示します。

```text
less code reading
  != less Evidence
```

この関係を維持します。

## 9. Commit granularity

ISSUE-000040の作業では、意味上の境界ごとにmainへコミットします。

推奨する単位：

1. 判定段階Cの開始記録と適格性の検証根拠
2. 移植可能性と移行先での判断
3. 構成案
4. 受理済みソース
5. 環境の対応付け
6. ドライバーと実行先の検証根拠
7. 変更の伝播に関する検証根拠

ファイル単位ではなく、検証可能な意味の単位でコミットします。

一つの意味の変更を複数の作業系列に分けません。セッションが並行する場合は、現在のmainと関連するソース・検証根拠を変更直前に再確認します。

## 10. main commit completion

意味の単位ごとのコミットでは、少なくとも次のうち該当する情報を正本文書または検証根拠に残します。

- 判定段階Cの開始時の検証根拠
- 共有・移行先・環境のそれぞれが持つ権限の境界
- 再利用した正本上の識別情報
- 移行先が所有する識別情報
- 未確認事項や互換性がない項目
- 受理に関する検証根拠
- 再読込みと検証に関する根拠
- ドライバーと実行先の検証根拠
- 変更の伝播に関する検証根拠
- スキーマの追加を実施したか見送ったか、その理由

ブランチやPRの状態を完了条件にはしません。コードを変更した場合はpush前にリポジトリの健全性確認を必須とし、文書だけの場合もリンク・状態・判定段階と検証根拠に関する主張を確認します。

## 11. 現在状態

2026年9月19日、最初のウェブ実行先は判定段階CのReadyに到達しました。

実際の開始記録：

- `B / R / T = B / e5231839a5dcc62cce1181a2731fcb7c70d16736 / 0ba87a4e0a96f70a198b03ff7652fb00e3826abe`

S0は完了しています。

S1・S2については、

- `tools/reference-business-procedure-check/second_application_s1_s2_actual_pressure_test.go`

この試験により、ソースで定義された状態まで進みました。



2026年9月21日時点で、改訂後の実行可能な検証、多層の共有受理機構の実装、実際の`reference/workspace`を使った読取り専用のプレビュー、別の生成AIによる独立レビュー（`review-clear-with-conditions`）まで完了しました。受理前に必須となる修正はありません。S5に残る停止条件は、正確な`c18ae...`に対する**人による明示的な受理の判断**だけです。

受理手続きの実行契約：

受理に成功した後のS6で使うソース・再読込み・検証の契約：

一般的な作業継続の指示を、受理を証明する記録として扱いません。

## 12. Start Packet完了条件

- [x] Gate C entry recordの必須項目を固定した
- [x] 一つのISSUE-000040 workstreamへ収束し、remote branchを増やさない規則へ更新した
- [x] first Reference scopeをread-only customer lookupへ限定した
- [x] ISSUE-000047 → Phase 5 → ISSUE-000040 → Acceptanceの順序を固定した
- [x] Application / Environment authorityを分離した
- [x] unknown / incompatibleのstop条件を固定した
- [x] existing grammar firstを維持した
- [x] AI Builder / Human Review handoffを定義した
- [x] direct-to-main / health-check / concurrent freshness規則へ更新した
- [x] Gate C前にsecond Application sourceを追加していない
