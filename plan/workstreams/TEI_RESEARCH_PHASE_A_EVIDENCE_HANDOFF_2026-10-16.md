# TEI技術調査 Phase A：一改訂の実行・観測証拠の受渡し — 2026年10月16日

状態：**現行mainとIssueを読んだ実施準備・未認定**  
確認したmain：`ffc866790362e948f56ddd7c223a0470f6b0c110`（2026-10-16時点）。  
親計画：[技術調査の実証計画](../research/TECH_RESEARCH_EXECUTION_PLAN_2026-10-16.md) Phase A、既存の[製品クリティカルパス](TEI_PRODUCT_CRITICAL_PATH_2026-10.md) §30–31。  
**正本の所有者**：#1119（Product Gate）、#267（Shared Acceptance）、#1121（Source→Web）、#1127（業務Action/DB）、#1163（Evidence）、#1445（工程上の影響・再検証提案）。本書はこれらの実装・証拠を置き換えない読取り専用の受渡し表であり、新しいGate、Schema、Authority正本ではない。

## 1. 比較した三つの事実群

| 区分 | 確認できる実績 | 今回の未保証 |
| --- | --- | --- |
| 現行WebのSource拘束 | #1121のPR #1782、#1863、#1873、#1881はmainに統合。合成Host Current Accepted Source Readerを用いた起動時の確認、HTTP要求時の再照合と失効時503／dispatch拒否、Host親context終了時のlistener停止を試験済み | #267が所有する**真正な人的Acceptanceに基づくCurrent Readerの発行・更新・取消**、実Effect Grant/S05、複数ファイルの原子的読込みと実行途中取消は未実証。Synthetic ReaderのPASSを正式Source受理へ読み替えない |
| 歴史的な合成Web E1 | [Actions #38267656696](https://github.com/hat47x/tei-platform/actions/runs/38267656696) の8件の保全済みreceipt＋[manifest](../evidence/TEI_E1_SYNTHETIC_MANIFEST_20261012_38267656696.json)。実HTTPの正例・負例・N11とハーネス側Source差分検知がある | Providerはsynthetic、manifestの`humanAcceptance=false`、`runtimeVersionBindingVerified=false`、`staleSourceRejectedByTarget=false`、`independentDatabaseRead=false`。当時のR0/R1全Workspaceは証拠ファイルだけからは復元不可 |
| 歴史的な実PostgreSQL E2a-1 | [Actions #38311557882](https://github.com/hat47x/tei-platform/actions/runs/38311557882) の[保全済みreceipt](../evidence/TEI_E2A1_REAL_POSTGRES_RECEIPT_38311557882.json)。使い捨てPostgreSQL、`request_log/intake.request.submit`の手書きoverlay/SQLから実write→同一プロセスの別pool read、Schema不一致拒否、N11 read失敗を観測 | `order_entry`ではない、`authority=none`、`generatedRoundTrip=false`、同一プロセス別poolは**別invocationではない**。正式Accepted R1→通常Web→実業務rowを証明しない |

既存の[歴史的E1/E2突合器](../../tools/verify-critical-path-e1-e2-join.py)を**新しいProduct判定器として使わない**。この突合器の記録は`UNJOINABLE_HISTORICAL_EVIDENCE`・`productGate=UNQUALIFIED`であり、下記の8つの未達条件は現行の担当・実証対象を理解するための分類である。元の2試験では、業務主体、実行版、実Provider、Commandの識別が一致していない。研究で成功率を集計するときも、2つの個別のPASSを、一つのE2成功として合算しない。

## 2. 最初に必要な受渡しと、並行できる作業

```text
#267: 現在受理されたR1のHost所有Source witness・取消／差替
    │ exact membership / fingerprints / issuer authority
    ▼
#1121: 実WebがR1を受理、R0と意味同値R2をdispatch前拒否
    │ current Host Reader / DecisionRef / AuthorityRevision
    │
    ├──────────────────────────┐
    ▼                          ▼
#1127: 同じorder_entry       #1163: Source・受理・Target・権限・
       Operationで実PG write        Command・Observationの収集手順
    │                          │
    └─────────┬────────────────┘
              ▼
  別 invocation/context の実DB read、S02/S05/N11反例
              ▼
#1119: Ownerが既存Product Gateで判定
              ▼
#1445: 変更影響・未取得Evidence・検証範囲をread-onlyで提案
```

**最初の信頼境界**：#267からの**現行の真正なHost Current Accepted Source**。CLIが与えた`accepted=true`、Decision JSON、Source digest、実Webのsynthetic Readerだけでは、この部分を置き換えられない。#267の変更を#1121・#1163・#1445へ複製しない。

**並行可能**：#1127は正式Host Readerが利用可能になる前でも、既存の`order_entry`のOperationとPersistence Contractの対応、実Providerのfixture安全条件、別invocation読取用のread identityと業務行キーを決められる。#1163は収集項目とfail-closedな負例の組立てを先行できるが、いずれもProduct PASSとはしない。

## 3. 既存の八つの未達条件と最小の担当実験

八つの未達条件の正本は、既存`tools/verify-critical-path-e1-e2-join.py`内の`OBLIGATIONS`である。下表は各条件の**現行所有先と次の一つの反例**を示す作業用の投影である。検査器の条件を複製して実装するものではない。

| 既存の未達コード | 実証で埋めること／最低限の負例 | 所有Issue |
| --- | --- | --- |
| `SAME_ACCEPTED_SOURCE_REVISION` | 実Webが読み込んだSource全ファイルのmembership・前後SHA・RevisionとHostが現在受理するR1を照合。R0、R2、途中のbyte変更を拒否し、未確定は保留 | #267、#1017、#1121 |
| `SAME_APPLICATION_AND_BUSINESS_SUBJECT` | `order_entry`の同一Data／Action／OperationをSource・Web・DB write/readまで維持。別Applicationの試験を混ぜる負例 | #1127、#1120、#1163 |
| `REAL_WEB_COMMAND_LINKED_TO_POSTGRES_WRITE` | 通常WebからのCommand ID・業務keyと、同じPostgres行へのwriteを相互参照。自己応答・synthetic Provider・手書きの別Applicationを含めない | #1127、#1163 |
| `RUNTIME_REJECTS_R0_AND_R2_BEFORE_DISPATCH` | 実Web自身がHost current witnessでR0と意味同値R2を拒否。ハーネスだけのラベル比較は不可。失効後のHTTP要求はdispatch=0 | #1121、#267 |
| `CURRENT_HOST_WRITE_AUTHORITY` | Current Principal／Subject／Policy版／Effect Grantを実Hostで照合。S05失効・別Scopeはdispatch=0 | #1043、#951、#1127 |
| `INDEPENDENT_READ_INVOCATION` | write終了後、**別の実行呼出し**（可能なら別プロセス・新しいDB connection）から同じ業務keyを読取り比較。N11はCommand完了でも`observed`禁止 | #1127、#1163 |
| `HUMAN_ACCEPTANCE_AUTHORITY` | 人の判断に紐づく実Accepted R1の発行・再取得・取消を検証。合成`accepted`フィールドや署名なしのDecision JSONでは不可 | #267、#1163 |
| `DATA_FIRST_GENERATED_EXECUTION` | 対象の生成／宣言由来と手書きoverlayを区別。通常の実業務Actionが実Targetへ実現されることを証明し、手書きSQLの先行試験だけで済ませない | #1127、#1121 |

**重要**：歴史的突合器はその2つの固定receiptに限る。新しい一改訂のProduct E2証拠が揃った後、その証拠が条件を満たすかどうかは、#1119／#1163の**既存の新証拠の検査経路**で判定する。上記historical E1/E2 verifierの閾値を書き換えて成功させない。

## 4. 一回の正式なE2候補実験で残す最小記録

段階や正本が異なる成果物を混在させず、少なくとも次の情報をたどれることを確認する。形式やフィールド名の正本は**#1163の既存Evidence契約**であり、この文書で新しいPacket schemaを制定しない。

- **Source**：Applicationとtyped Subject、Managed Source membership/正規化パス/前後SHA・Source Revision、業務意味の受理版、元の受理DecisionとHost受理主体、取消・差替履歴への参照。
- **Target／Provider**：実Web Buildと対応Artifact digest、実WorkspaceのLoader/Reader、固定したProvider／DB alias・物理schema／実接続先を識別できる非秘密情報、Hostの実Subject／Current Policy／Effect Grant。署名、Issuer trust、Builder Authority等を別々に判定する。
- **Command／Observation**：正確なBusiness OperationとCommand ID、保存対象の実業務key、書込み結果、異なるinvocationのread ID・主体・時点・照合した業務row、read failureとno-effectの区別。
- **検証**：Scenarioと独立Expectationの出所、負例R0/R2/S02/S05/N11、実行したCLI/Web/DBの別、実行した環境・exact commit・テスト名、合成／実Target／人の確認の別、Sourceと出力を検証器自身が変更していないこと。
- **未実施**：依存Component、Host permission、Source atomicity、read invocationなど未取得なら`unresolved`・`not-evaluated`で残す。旧E1とE2a-1のSUCCESSを埋め草に使わない。

認証情報、接続文字列、生の業務個人情報、元外部資料の非公開本文をEvidenceへ記録しない。許可された使い捨てDB・テスト用データを使用する。歴史的Scratchの欠落を、今のmain上のWorkspaceで代用しない。

## 5. 実行できる既存の再確認入口

新しい検査器や独立履歴DBは増やさず、既存の監査ツールを案内する。

```sh
# 歴史的E1/E2a-1の固定原本を照合し、旧証拠が結合不能と示す
python3 tools/verify-critical-path-e1-e2-join.py

# 「旧原本の二つのPASSではG2にならない」ことをexit code 3で確認
python3 tools/verify-critical-path-e1-e2-join.py --require-g2

# 実在する歴史的Workspace 2個が提供されたときだけSource byte差分を取る
go run ./tools/reference-change-assurance \
  -before-workspace /path/to/actual/r0 \
  -after-workspace /path/to/actual/r1 \
  -scope order-entry-reference \
  -consumer-kind application -consumer-id order_entry
```

**これらのコマンドは本書作成時点では再実行していない。** 過去のPASSや保存済みJSON内容を今回の実行結果と取り違えない。Sourceの完全スナップショットや正式Acceptanceの証拠が未提供なら、対応する列は`not-evaluated`のまま。既存#1445のPOC-8〜13はread-onlyでSource/Scenario/Evidenceの候補を照合するが、Runtime execution Authority・Product Gateを発行しない。

## 6. 10月23日までの小さい担当間受渡し

1. **#267 → #1121**：真正Host Current Accepted Source Readerの「発行・保存・更新・取消」を誰が保持するか、現在の状態で利用できるportと不足分を、一つのApplicationで提示。期待が固まるまでは合成Reader試験のまま。
2. **#1121 → #1127**：通常Webの外部Effect実行入口で必要なSource R1とHost Authorityの入力・失敗と、R0/R2/S05時にdispatchへ進まない点を照合する。
3. **#1127 → #1163**：同じ`order_entry` Operationで実PostgreSQLの業務rowを保存できる最小候補と、**別invocation read**の識別を提出。手書き別ApplicationのE2a-1からの流用不可能点を明記する。
4. **#1163 → #1119**：実際に一つの新しい試験からSource/Host/Provider/Command/readが採れた場合のみProduct用Packet候補へ渡す。欠落部分・拒否・失敗は消さない。
5. **#1445**：人とAIの工程比較に「何の証拠が無いときに判断を保留できたか」「根拠へ到達する操作数／時間」「影響外の不要な再試験数」を加える。目標数値は実測後に評価する。

## 7. 一段先へ進める条件

最初のマイルストーンは8未達条件のチェックリストを埋めることではなく、**同一の実Source／Host／Operation／Command／Observationが一つの新しいrunで辿れるか**を証拠ごとに判定できること。着手可能な作業は並列化し、契約の欠落を全体の新機構で埋めない。Host Readerの真正な情報源が未定のままなら、「実験は続けるが製品用G1/G2は未認定」で停止する。

この受渡しは研究結果に基づく作業案であり、Issueの担当合意、実コード、実DB試験、CI、利用者による受理を済ませたことを意味しない。
