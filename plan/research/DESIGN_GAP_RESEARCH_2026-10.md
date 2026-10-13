# 現行設計の空白と外部方式の調査 — 2026年10月

- 状態: 調査記録（採用前の設計評価）
- 調査日: 2026-10-12
- 対象: `main`のコミット`daa591d6`（調査開始時は`0f90c2ca`。調査中に進んだ`main`を取り込み、空白の領域が埋まっていないことを再確認した）
- 役割: 現行の設計で担当も設計文書も無い領域を特定し、外部の確立した方式を比較して、設計候補の根拠を残す
- 正本との関係: 要求の正本は[製品要求](../../requirements/TEI_PRODUCT_REQUIREMENTS.md)、要求別の検証状況は[要求の不足監査](../evidence/PRODUCT_REQUIREMENT_GAP_AUDIT.md)、実装状況は[実装状況](../../docs/status.md)である。この文書はどれも置き換えず、要求別の評価区分も変更しない

## 1. 位置づけと方法

### 1.1 この文書で判断できること

この文書を読むと、次を判断できます。

- 現行の設計のどこに、担当課題も設計文書も無い空白があるか
- その空白が、どの要求、受入シナリオ、既存のコードに関係するか
- 外部ではどの方式が確立しており、TEIはどれを取り込み、どれを取り込まないか

設計そのものは、各章の末尾に挙げる`plan/design/`の設計候補に置きます。どの設計候補も評価待ちであり、採用はしていません。

### 1.2 空白の判定基準

次の三つをすべて満たす領域を空白としました。

1. 要求、受入シナリオ、設計判断記録のいずれかが、その領域の振る舞いを必要としている。
2. その振る舞いを定めた設計文書・設計判断記録が無いか、あっても複数の文書が互いに矛盾している。
3. 進行中の課題やプルリクエストが、その領域を担当として扱っていない。または担当が分かれていて、統合する担当が無い。

### 1.3 確認したもの

- 要求の不足監査、クリティカルパス計画、機能設計の索引、設計判断記録の索引
- 2026-10-12時点の未完了の課題（約100件）と、未統合のプルリクエスト
- 文書中の「未設計」「未定義」「未決定」などの記述
- 空白と判断した領域に関係するコード（`internal/`、`tools/`、`reference/`）

文書の記述だけで空白と判断せず、関係するコードを読んで現状を確かめました。各章の「空白の根拠」に、確かめた場所を記します。

### 1.4 結果の概要

| 領域 | 空白の内容 | 主に関係する要求・試験 | 設計候補 |
| --- | --- | --- | --- |
| 業務副作用 | 確定後の外部効果、処理の段階、一括処理の再開に設計が無い。冪等・不明・再実行の用語が文書ごとに異なる | RQ-005、RQ-020、否定試験N10 | [業務副作用の段階・確定後効果・再試行の設計](../design/BUSINESS_EFFECT_LIFECYCLE_DESIGN_2026-10.md) |
| 実行相関 | 一つの業務操作を層をまたいで追跡する識別子の規則が無い。観測が先行コマンドを参照するというADR-0023の要件が実装されていない | RQ-018、RQ-020、RQ-021 | [実行相関と運用診断の設計](../design/EXECUTION_CORRELATION_DESIGN_2026-10.md) |
| データ移行 | 意味の変更に伴う既存データの変換、影響の確認、段階的な移行が設計されていない。データを失う移行用SQLが確認なしに生成される | RQ-005、RQ-006、受入シナリオV03・V04 | [意味の変更に伴う既存データの移行設計](../design/SEMANTIC_CHANGE_DATA_EVOLUTION_DESIGN_2026-10.md) |
| 知識の変種 | Variationと適用先固有の差分に表現が無く、三つの領域が別々の衝突規則を作り始めている | RQ-027、RQ-029 | [再利用した知識の変動点と適用先差分の設計](../design/KNOWLEDGE_VARIATION_DESIGN_2026-10.md) |

今回は扱わなかった空白を6章に記します。

## 2. 業務副作用の段階と確定後効果

### 2.1 空白の根拠

- 要求の不足監査は、副作用を伴う汎用実行について「transaction / prepare-commit / compensation / retry / result validation failureの扱いが未設計」と記している。
- 準備と確定の参照実装（#1157、[`tools/reference-capability-mutation-protocol/`](../../tools/reference-capability-mutation-protocol/README.md)）と、副作用フェンスの参照実装（#1174、[`reference/effect-fence/`](../../reference/effect-fence/README.md)）は、どちらも`main`に統合済みだが、互いを参照していない。
- 確定後の外部効果は、副作用フェンスでは`RemoteEffect`による保証対象外の判定（`effect-outside-boundary`）として現れるだけで、扱う設計が無い。SUIの参照実装も、自身が耐久的なアウトボックスではないと明記している。
- 処理の段階（書込み前の変更、同一トランザクション内の連鎖更新、確定後効果）は#1129の課題本文にだけあり、否定試験N10は試験の番号だけが定まっている。
- 一括処理の途中停止と再開は#1130の課題本文にだけある。
- 用語が文書ごとに異なる。冪等の識別子は`Invocation Key`、`OperationID`、配備の`IdempotencyKey`に分かれている。「不明」は、確定状態、Goのエラー、再試行できる配備結果、未宣言の業務結果名のそれぞれを指している。「再実行」は、副作用の再実行と、既存結果の返却の両方に使われている。

### 2.2 外部の方式

| 方式 | 要点 | 出典 |
| --- | --- | --- |
| トランザクショナル・アウトボックス | 送るべき通知を業務更新と同じトランザクションで表に書き、別の中継処理が配送する。確定した場合にだけ送られる。配送は少なくとも一回で、受け手が識別子で重複を排除する | [microservices.io](https://microservices.io/patterns/data/transactional-outbox.html)、[冪等な受け手](https://microservices.io/patterns/communication-style/idempotent-consumer.html)、[Debezium Outbox Event Router](https://debezium.io/documentation/reference/stable/transformations/outbox-event-router.html) |
| PostgreSQLの論理デコードメッセージ | `pg_logical_emit_message`の第1引数を真にすると、メッセージはトランザクションの一部になり、確定した場合だけ読み出される | [PostgreSQLの管理関数](https://www.postgresql.org/docs/current/functions-admin.html) |
| 冪等のための識別子 | 識別子を別の内容の要求に再利用しない。内容の不一致と処理中の重複を区別して返す。保持期間を公開する。IETFの仕様案は2026年4月に期限切れとなり、RFCにはなっていない | [IETFの仕様案](https://datatracker.ietf.org/doc/draft-ietf-httpapi-idempotency-key-header/)、[Stripe](https://docs.stripe.com/api/idempotent_requests)、[AWS Builders' Library](https://aws.amazon.com/builders-library/making-retries-safe-with-idempotent-APIs/) |
| 原子的な段階と再開地点 | 外部への変更の間を局所的なトランザクションで区切り、再開地点を同じトランザクションに記録する | [Brandur Leach](https://brandur.org/idempotency-keys) |
| 耐久的な実行の基盤 | 実行の手順と結果を記録し、完了した手順を再実行しない。外部呼出しは少なくとも一回で、冪等性は呼出し先が保証する | [Temporal](https://docs.temporal.io/activity-definition)、[Restate](https://docs.restate.dev/foundations/key-concepts)、[DBOS](https://docs.dbos.dev/architecture) |
| サーガと補償 | 補償は意味上の取消しであり、元の状態を正確には戻さない。後戻りできない手順（ピボット）の後には再試行で完了できる手順だけを置く | [Azure Architecture Center](https://learn.microsoft.com/en-us/azure/architecture/patterns/saga)、[microservices.io](https://microservices.io/patterns/data/saga.html) |
| 仮予約・確定・取消し | 取消しが先に届いた場合の空の取消しと、遅れて届いた仮予約の拒否を記録で扱う | [Apache Seata](https://seata.apache.org/blog/seata-tcc-fence/) |
| 再開できる一括処理 | 区切りごとに進捗を同じトランザクションで記録し、再開時は最後に確定した位置から続ける。区切りを取り消すと項目が再処理されるため、項目の処理は冪等にする | [Spring Batch](https://docs.spring.io/spring-batch/reference/step/chunk-oriented-processing/restart.html)、[Jakarta Batch 2.0](https://jakarta.ee/specifications/batch/2.0/jakarta-batch-spec-2.0.html) |
| ローコード基盤の処理段階 | 確定後の処理を、確定前の処理と分けて実行する。一方で、名前が「確定後」でもトランザクション内で動く仕組みや、実行順を定めない仕組みがある | [Mendix](https://docs.mendix.com/refguide/event-handlers/)、[OutSystems](https://success.outsystems.com/Documentation/11/Reference/OutSystems_Language/Data/Database_Reference/Handling_Transactions/) |
| フェンシングトークン | 単調に増える番号を書込み先が照合し、古い所有者の書込みを拒否する | [Martin Kleppmann](https://martin.kleppmann.com/2016/02/08/how-to-do-distributed-locking.html) |

### 2.3 TEIへの取り込み

取り込むものは次のとおりです。

- 確定後の外部効果を、業務更新と同じトランザクションで書く効果記録に限る。これにより否定試験N10が構造上成立する。既存の副作用フェンスの判定（外部効果を含む確定は保証対象外）とも矛盾しない。
- 冪等の記録を「範囲・識別子・要求の照合値」とし、保持期間を宣言する。
- 不明な結果は照会で解消し、新しい識別子での再実行と自動の補償を行わない。Stripeの指針と#1157の既存方針が一致している。
- 補償を別の業務操作として宣言する。空の取消しと遅れて届いた元操作の扱いを含める。
- 一括処理の進捗を、項目の更新と同じトランザクションで記録する。

取り込まないものは次のとおりです。

- 汎用の手順調整基盤の採用。TEIの中核を特定の実行基盤に依存させないためです。比較の参考に留めます。
- 一回だけの実行の保証。どの方式も、外部への呼出しについては少なくとも一回の実行と冪等性の組合せで成り立っています。
- 実行順を定めない連鎖処理や、名前と実際の段階が一致しない処理段階。

## 3. 実行相関と運用診断

### 3.1 空白の根拠

- [製品品質の構成](../../docs/product-quality-architecture.md)の10章は相関の原則だけを示し、要求の不足監査は「Target / production横断diagnostic correlation」を残る不足に挙げている。担当課題は無い。
- `internal/plugin/runtime/model.go`の`Request.RequestID`は発行規則の無い文字列である。`InvocationProvenance`は、アプリケーション、実行先、受理済みの意味の版、業務操作の識別子を持たない。
- `internal/driver/observation.go`の観測の記録は、先行するコマンドへの参照を持たない。ADR-0023は観測の検証根拠に「先行Commandのidentity / run reference」を求めている。
- Webの参照実装の監査記録は、要求の識別子、時刻、主体の参照を持たない。
- 要求ごとのログ、計測、追跡の実装は無い。W3C Trace Contextへの言及もリポジトリに無い。
- `internal/query/analyzer/client.go`は生の標準エラー出力をエラー文へ含めており、プラグインの実行環境の縮約規則と揃っていない。

### 3.2 外部の方式

| 方式 | 要点 | 出典 |
| --- | --- | --- |
| W3C Trace Context | `traceparent`と`tracestate`で追跡情報を受け渡す。第1版は2021年11月の勧告、第2版は2024年3月の勧告候補草案。個人情報を入れてはならない。呼出元の標本抽出の指示は推奨に過ぎない | [第1版](https://www.w3.org/TR/trace-context/)、[第2版](https://www.w3.org/TR/trace-context-2/) |
| W3C Baggage | 利用者を特定できる情報を含み得るため、同じ信頼境界の中で使う。2024年5月の勧告候補 | [W3C Baggage](https://www.w3.org/TR/baggage/) |
| 環境変数による受渡し | OpenTelemetryの仕様ではリリース候補。起動時に一度だけ読まれ、他のプロセスから読める場合がある | [OpenTelemetry](https://opentelemetry.io/docs/specs/otel/context/env-carriers/) |
| MCPの付帯情報 | `params._meta`に`traceparent`などを入れる仕様（SEP-414）が確定している | [SEP-414](https://modelcontextprotocol.io/seps/414-request-meta) |
| OpenTelemetryの意味規約 | サービスと配備環境の属性は安定版。データベースの呼出しでは、値を埋め込んだSQL文を既定で記録しない。版管理や成果物の属性は成熟度が低い | [サービス](https://opentelemetry.io/docs/specs/semconv/registry/attributes/service/)、[データベース](https://opentelemetry.io/docs/specs/semconv/database/database-spans/)、[成果物](https://opentelemetry.io/docs/specs/semconv/registry/attributes/artifact/) |
| Goの実装 | 追跡と計測は安定版。ログのAPIと実装は2026年10月の1.47.0で安定版になった | [opentelemetry-go 1.47.0](https://github.com/open-telemetry/opentelemetry-go/releases/tag/v1.47.0) |
| 監査記録 | 監査記録は運用ログと分け、追記のみ、改ざん検知、論理的に分けた保存先とする。セッションの識別子、資格情報、個人情報を直接記録しない | [OWASP Logging Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Logging_Cheat_Sheet.html)、[OWASP ASVS 5.0 V16](https://github.com/OWASP/ASVS/blob/master/5.0/en/0x25-V16-Security-Logging-and-Error-Handling.md)、[NIST SP 800-92](https://csrc.nist.gov/pubs/sp/800/92/final) |
| 問題詳細の形式 | `type`で問題の種類、`instance`で発生を示す。内部の構造を露出しない | [RFC 9457](https://www.rfc-editor.org/rfc/rfc9457.html) |
| 標本抽出 | 先頭での標本抽出は、誤りを含む追跡をすべて残すことを保証できない | [OpenTelemetry](https://opentelemetry.io/docs/concepts/sampling/) |

### 3.3 TEIへの取り込み

取り込むものは次のとおりです。

- 業務操作の識別子と、追跡用の識別子を分ける。追跡用の識別子は欠け得るため、冪等や監査に使わない。
- 実行先の境界で外部の追跡情報を親として採用せず、関連として結び付ける。
- プラグインのプロセスへは、要求の封筒の項目として追跡情報を渡す。MCPの付帯情報と形を揃える。
- 記録してよい項目を許可制にする。値を埋め込んだSQL文や業務データの値を記録しない。
- 監査記録、運用計測、検証根拠を分ける。運用計測を検証根拠にしない。
- 運用者向けの失敗を診断符号で表し、HTTPではRFC 9457へ写す。

取り込まないものは次のとおりです。

- OpenTelemetryを実行基盤の必須依存先にすること。製品品質の構成13章がこれを先に固定しないとしているためです。
- 環境変数による追跡情報の受渡しを、長く動くプロセスで使うこと。

## 4. 意味の変更に伴う既存データの移行

### 4.1 空白の根拠

- 物理スキーマの差分と移行用SQLの生成はあるが、既存データの値を新しい意味へ変換する設計は無い。PostgreSQLの準拠確認文書（m0.26）は、本番運用できる品質で実現したとは主張しない機能の一つに、移行処理を実行する機構を挙げている。
- `internal/database/providers/postgres/migration_sql.go`は、既定値の無い必須列の追加を`ADD COLUMN … NOT NULL`として出力する。行のある表ではPostgreSQLがこの文を拒否する。型の変更は変換式なしの`ALTER COLUMN … TYPE`、列の削除は確認なしの`DROP COLUMN`として出力する。
- 破壊的変更の事前確認（`destructive_readiness.go`）は表の削除だけを対象とする。
- 手書きの移行の`data sql`手順はあるが、差分から生成した移行だけを受け付ける適用処理では実行できない。
- リリース計画の段階（`expand`、`populate`、`transition`、`rollout`、`contract`）のうち、検査を伴うのは`contract`だけである。データの準備状態は作者が宣言する値である。
- 意味の差分の影響は定義だけを対象にし、既存データの件数を示さない。
- クリティカルパス計画は受理と適用を分け、移行が必要な変更の分類と停止の規則（S02）を定めたが、移行が必要と判定した後の既存データの扱いは定めていない。
- 担当課題は無い。

### 4.2 外部の方式

| 方式 | 要点 | 出典 |
| --- | --- | --- |
| 並行変更（拡張・移送・縮約） | 新しい形を古い形と並べて追加し、利用者を移してから古い形を除く。縮約を省くと元より悪い状態になる | [Martin Fowler](https://martinfowler.com/bliki/ParallelChange.html)、[データベースのリファクタリングの目録](https://databaserefactoring.com/) |
| 複数リリースに分けた手順 | 列の削除は、無視する設定、削除、設定の撤去を別のリリースで行う | [GitLab](https://docs.gitlab.com/development/database/avoiding_downtime_in_migrations/) |
| 版ごとのビューによる共存 | 版ごとにビューのスキーマを作り、古い版と新しい版を同時に動かす。値の変換を前向きと後ろ向きの式で宣言する | [pgroll](https://github.com/xataio/pgroll/blob/main/docs/concepts.md)、[Reshape](https://github.com/fabianlindfors/reshape) |
| 移行の検査 | 削除、データ次第で失敗する変更、古い版と両立しない変更、長い排他を取る変更を検出する | [Atlas](https://atlasgo.io/lint/analyzers)、[strong_migrations](https://github.com/ankane/strong_migrations) |
| データ移行の分離 | 構造の変更とデータの変換を同じ移行に混ぜない。大きな表では区切って処理する | [Django](https://docs.djangoproject.com/en/5.2/ref/migration-operations/) |
| PostgreSQLの機能 | 変動しない既定値付きの列の追加は書き直しを伴わない（11以降）。制約を未検査で加えて後で検査できる。有効な検査制約があれば必須化の全件走査を省ける（12以降）。必須の制約を未検査で加えられる（18以降）。小数の桁を減らすと値は丸められ、明示的な型変換は文字列を切り詰める | [ALTER TABLE](https://www.postgresql.org/docs/current/sql-altertable.html)、[数値型](https://www.postgresql.org/docs/current/datatype-numeric.html)、[文字型](https://www.postgresql.org/docs/current/datatype-character.html) |
| ローコード基盤の扱い | 型の変換ができない場合に既定値で作り直す例、属性を削除しても列を物理的に消さない例、型の変更をほとんど許さない例がある。多対多から1対多へ変えると重複が残り、新しい版が起動できなくなる例が報告されている | [Mendix](https://docs.mendix.com/refguide/attributes-type-migration/)、[OutSystems](https://success.outsystems.com/documentation/outsystems_developer_cloud/building_apps/data_management/data_modeling/entities/)、[Dataverse](https://learn.microsoft.com/en-us/power-apps/maker/data-platform/create-edit-field-portal) |
| イベントを正本とする方式 | 保存した事実を変えず、読み込み時に古い形を新しい形へ変換する | [Greg Young](https://leanpub.com/esversioning/read) |

### 4.3 TEIへの取り込み

取り込むものは次のとおりです。

- 意味の変更を既存データとの両立性で分類し、分類ごとに必要な宣言・観測・段階を決める。
- 既存データの影響を、件数による観測として受理前に示し、適用の直前に測り直す。
- 導けない変換は作者がデータ変換として宣言する。提供元は変換式や既定値を作らない。
- 既存のリリース計画の段階を使い、段階の終わりを測定で確かめる。
- PostgreSQLの移行用SQLを、行のある表で安全に動く手順にする。
- データを失う操作は縮約の段階でだけ行う。

取り込まないものは次のとおりです。

- 版ごとのビューの自動生成。有力な方式ですが、提供元とアプリケーションの構成に依存するため、変更ごとに選べる方法の一つとして扱います。
- イベントを正本とする保存方式。TEIの現在の保存方式ではないためです。

## 5. 再利用した知識の変動点と適用先差分

### 5.1 空白の根拠

- RQ-029はVariationと適用先固有の差分の追跡を求め、RQ-027は再入場したAIがVariationへ到達できることを求めている。要求の不足監査は、どちらにも表現が無く（`internal/`に0件）、担当課題も設計判断記録も無いと記している。
- [知識移植のR0境界](../workstreams/KNOWLEDGE_PORTABILITY_R0_BOUNDARY_2026-09.md)は汎用の上書き機構を作らないとし、複数の実際の参照例で差分の関係が繰り返し現れてから判断するとした。
- その後、UI構成キット（#997）、業務仕様テンプレート（#998）、構成プロファイル（#1016）が、それぞれ別の衝突規則を作り始めている。
- 二つ目のアプリケーション`repair_intake`では、再利用元との関係が`.tei/decisions/`の自由記述の注記にしか残っていない。再利用元の版も照合値も無いため、再利用元の変更を検出できない。
- RQ-015とRQ-029で、Variationが「再利用できる変種」と「適用先での違い」の二つの意味に使われている。

### 5.2 外部の方式

| 方式 | 要点 | 出典 |
| --- | --- | --- |
| 共通可変性言語 | 可変性を持たない基底モデル、可変性のモデル、選択の結果を分ける。変動点には値、有無、差替えなどの種類がある。標準化は採択されていない | [CVL](https://caosd.lcc.uma.es/vexgine/cvl/) |
| 汎用可変性言語 | 機能の構成と制約を記述する。2025年に論文として公開され、事実上の共通形式として使われている。機能と成果物の対応は扱わない | [UVL](https://universal-variability-language.github.io/)、[論文](https://doi.org/10.1016/j.jss.2024.112326) |
| 差分指向の構成 | 一つの有効な中核と、適用条件付きの差分で製品を作る。差分の順序を明示する | [Schaefer ほか 2010](https://research.chalmers.se/en/publication/124332) |
| 上書きによる重ね合わせ | 後の層が値を上書きする。順序に依存し、衝突は表に出ない | [Kustomize](https://kubectl.docs.kubernetes.io/references/kustomize/kustomization/patches/)、[Helm](https://helm.sh/docs/chart_template_guide/values_files)、[Nix](https://nixos.org/manual/nixpkgs/stable/#chap-overlays) |
| 単一化 | 値と制約を一つの体系で扱い、順序に依存しない。矛盾は誤りになり、見た値が最終結果になる | [CUE](https://cuelang.org/docs/concept/the-logic-of-cue/) |
| 所有者の記録 | 項目ごとに管理者を記録し、他の管理者が持つ項目の変更を衝突とする | [Kubernetes Server-Side Apply](https://kubernetes.io/docs/reference/using-api/server-side-apply/) |
| 雛形の更新 | 雛形の版と回答を記録し、旧版での再生成・新版・利用者の変更の三者で更新する | [Copier](https://copier.readthedocs.io/en/stable/updating/) |
| 依存の固定 | 内容の照合値で依存を固定する。置換指定は主モジュールでだけ効き、依存先へ伝播しない | [Go Modules](https://go.dev/ref/mod) |
| 業務基盤のパッケージ | 提供元が、利用者の編集できる属性と固定する属性を決める。利用者の層が提供元の更新を隠す場合がある。利用者の変更の一括削除は取り消せない | [Salesforce](https://developer.salesforce.com/docs/atlas.en-us.packagingGuide.meta/packagingGuide/packaging_component_attributes.htm)、[Power Platform](https://learn.microsoft.com/en-us/power-platform/alm/solution-layers-alm)、[管理プロパティ](https://learn.microsoft.com/en-us/power-platform/alm/managed-properties-alm) |
| モジュールの三者比較による更新 | 取り込んだ版、現在の版、新しい版の三者で比較する。要素は埋め込んだ識別子で対応付ける | [Mendix](https://docs.mendix.com/refguide/updating-marketplace-modules/) |
| 継承による拡張 | 対象を位置の式で指し、一致しなければ誤りにする。置換した箇所はその後の更新を受け取らない | [Odoo](https://www.odoo.com/documentation/18.0/developer/reference/user_interface/view_records.html) |

### 5.3 TEIへの取り込み

取り込むものは次のとおりです。

- 変動点を再利用元が宣言し、適用先はそこだけを選択できる。それ以外は分岐として記録する。
- 選択の組合せを単一化で扱い、同じ変動点への二つの選択を衝突にする。
- 対象を正本識別子で指し、見つからなければ誤りにする。
- 採用記録に、固定した版と照合値を持たせる。更新は三者比較で提案を作り、共通受理を経て固定した版を進める。
- 適用先の選択を、さらに先の再利用へ伝播させない。

取り込まないものは次のとおりです。

- 上書きによる重ね合わせを共通の規則にすること。後の層が、気づかないうちに値を変え得るため、人とAIが変更の意味を確かめにくくなります。
- 可変性の制約を解く汎用の解法器。必要性が参照例で示されたときに検討します。
- 利用者の変更の一括削除のような、取り消せない操作。

## 6. 今回は扱わなかった空白

| 領域 | 現状 | 扱わなかった理由 |
| --- | --- | --- |
| 実行先と表示の境界 | [機能設計の索引](../design/README.md)の「まだ設計文書を置いていない領域」は、この領域を「判定段階Cより前のため未確定」としている。判定段階Cは2026-09-19に合格したため、この理由はすでに当てはまらない | #259、#1158、#997が担当している。二つ目の実行先の検証根拠が必要であり、文書だけで埋めるべきではない。索引の記述の更新は、担当課題での判断を待つ |
| 承認に十分な表示の条件（RQ-028） | 要求の不足監査は条件案と既存の確認表示の机上評価があるとし、#1143が視点をまたぐ変更の適格性確認を検討している | 担当がある |
| AIエージェントによる業務遂行の閉包条件（RQ-030） | 要求の不足監査は、圧力試験をどこで終えるかの条件が無いと指摘している | 設計方式ではなく、担当者の判断を要する計画上の問題である |
| 共有する形の定義と成果物の発行者・署名・失効 | 形の定義の照合値を検証する参照実装はあるが、発行者の信頼と失効は未実証 | #1104、#1105が担当している |
| 利用者組織の区分（テナント） | SUIの移行設計が、TEIにテナントに当たる境界が無いと記している | 現行の要求にテナントの分離が無い。要求として必要かを先に判断する必要がある |
| 業務状態の遷移履歴の保存場所 | ADR-0024が未決定としている | ADR-0024が提案中であり、その判断の中で扱う |

## 7. 次の手順

この文書と四つの設計候補は、採用を意味しません。次の順で評価することを提案します。

1. 業務副作用の設計を、#1129、#1130、#1157、#1174の担当者が確認する。最初に否定試験N10とEF-02（応答喪失後の再送）を参照実装で再現する。
2. データ移行の設計の担当課題を新設するか、#1126の範囲に含めるかを決める。最初に、移行用SQLの検査（既定値の無い必須列の追加、変換規則の無い型の縮小、保持の判断の無い列の削除を止める）を小さな変更として実装できる。
3. 実行相関の設計の担当課題を新設するか判断する。最初に、観測の記録へ先行する業務操作の識別子を加える（ADR-0023の要件との差の解消）。
4. Variationの設計を、`repair_intake`の顧客参照に採用記録を置く参照例で評価する。RQ-027とRQ-029の担当課題を新設するかを決める。

課題の新設と、要求の不足監査の評価区分の変更は、この文書では行っていません。

外部の仕様や製品の状態は、2026年10月12日の確認によります。採用時に再確認してください。原文を取得できなかった資料（Salesforceの一部の文書、Mendixの一部の文書、差分指向の構成の論文の本文など）は、要約や関連資料から得た内容であり、設計の主要な根拠にはしていません。
