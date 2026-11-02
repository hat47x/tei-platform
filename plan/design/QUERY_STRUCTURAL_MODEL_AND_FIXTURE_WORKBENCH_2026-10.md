# SQL構造モデル・テストデータ付きQuery Workbench：最小実証計画（提案）

- 作成：2026-10-30
- 状態：**設計・検証計画候補**。公開契約の採用、実装完了、SQL/DB間の意味同等性、Ownerの業務受理を表さない。
- 基準：2026-10-30確認時のmain `2503a25fee9ffa47592d47818c2bb35c0a2c50ab`。実装着手時は再照合。
- Query構造Explainの一次Owner：**#2387**（M0静的解析確認→M1 read-only投影→M2 fixture照合→M3 Modeler/SHOへの出力）。本書/PR #2388はその追加実証計画であり、重複する製品機能Ownerではない。
- 主な担当：#1640（集合演算/評価と計画）、#1126（データモデル）、#1131（型付きDataset/Snapshot/Expectation）、#1124（Modelerの投影）、#1803（設計書導出）。
- 関連：#988（SQL/DSLのOperator）、#1706（集合束縛・重複）、#1121（Lowering）、#1063（設計時/実行時のDSL）、#1180（保存済み業務Scenario）、#1807（外部データ取込）。新しい第二Semantic正本や第二Driverを設けない。

## 1. 検証する製品価値

**SQLをDB製品から独立したDSLとして構造的に解釈し、業務エンティティからCTE・サブクエリ等の仮想関係を経て結果集合へ至る過程を、人と生成AIが直接読める設計成果として示す。** その各段階に局所/共有のテストデータを束縛し、入力・中間・最終結果と差分を検査可能にする。優先順位は **設計書の可読性 → 意味の検証可能性 → 実DBへの実現** とする。

これを全SQLの移植性保証、SQLの最適化器、正式なテストデータ生成言語、業務結果のOwner受理として扱わない。三視点（Data / Function / Business）に関係する同じ対象の投影であり、SQLだけの独立Modelerを設けない。元のデータモデル表記に基づく横長属性・関連線・高密度表現の可読性を重視するが、許諾未確認の図面や素材を転載せず、既存形式の完全複製を必須としない。

## 2. 既存の足場と現在の不足（静的読解）

- `internal/query/derived.go` の `DerivedIR` はQuery scope（root/cte/subquery）、Source occurrence、logical/physical field、出力とfield lineage、解析coverageを保持する。CTEのSource bindingを物理表と区別するテストが `internal/query/derived_test.go` に存在。
- `internal/query/relational.go` は source/join/filter/aggregate/project/distinct/sort/limit/opaque の関係ノードを提供。これをWorkBenchの**派生投影入力**として使う。
- `internal/query/derived_select.go` の現在のtarget-neutral `DerivedSelect` は意図的に狭い。Sourceが一表の単純射影以外（CTE、JOIN、計算式等）の場合は拒否する。**現状でCTE中間結果を評価可能とは言えない**。#1640の取得前後の集合範囲・安全性を維持する。
- `internal/driver/queryresult/mapping.go` はraw列からSemantic attributeへの明示mappingを保持する。CTEの一時出力や行level lineageを、この対応が自動確定した意味属性にしない。
- `tools/reference-design-output/` は受理済みWorld由来の一部文書とmanifestを導出するが、今回のQuery WorkbenchやSHO編集・保存を実装済みとはしない。
- `docs/status.md`（2026-10-29更新）は統合Modelerと汎用実行基盤を未完成としている。本計画のfixture・表示・evaluatorの試験は未実行。

## 3. 一つの問い合わせの三層構造

1. **業務データ層**：受注/受注明細/顧客等のAccepted Entity、Business identity、候補キー、FD、Relation、意味型・値域制約（NULL・精度・列挙等）、Constraint。#1126が所有。Domainという語を単一値の値域の意味で使わない。
2. **問い合わせ構造層**：著者が保持するnative SQL → analyzerのScope/RelationalIR/Field lineage → `QueryVirtualRelation`相当の**表示用概念**。各CTE/derived tableをscope-localに投影するが、新Core kindとして先行採用しない。CTEは物理的な一時表を意味せず、CTE名も業務エンティティIDではない。
3. **観測・説明層**：fixtureの入力行、中間結果、期待結果、実行結果、差分、未解析/未評価/失敗診断、SHO設計書用の図表。結果観測は意味正本ではない。

### 仮想関係のカード（表示例の意味契約）

- identity：Query ID + SQL digest + lexical scope path + projection/view revision。Schema/Business CanonicalIDへ自動昇格させない。
- interface：列順、alias、derived/physical field refs、型・NULL性（解決済み/未解決）、集合の種類、観測行数、candidate keyとその根拠。
- join・aggregate・filter等の演算参照と、入力/出力の列対応（column lineage）を保持。
- **三つの異なる線**：業務Relation（親子・参照等）／問い合わせの行集合変換／field lineage。描画を変えて識別する。
- 行lineageは実行時の寄与証拠（行ID、入力fixture版、演算ID、評価条件）を持つときだけ表示する。集計出力のfield lineageが存在することを、その集計へ実際に寄与した行の証明としない。
- key/cardinalityのラベルを **declared / statically derived / fixture-observed / unknown** に分ける。テストデータ内で一意でも全入力に対するFDの証明としない。
- SQLは基本的に重複行を保持する（bag）ので、関係図上の行集合を暗黙に集合化しない。DISTINCT、UNION ALL、重複を生むJOIN、ROW_NUMBER、GROUP BY、NULLの三値論理、並べ替えをモデル上に残す。

### 3.1 識別性・一意性の伝播（静的推論の候補）

問い合わせの途中段階では、行数・キー・一意性は**データモデルからそのまま継承されるとは限らない**。次の判定を最初の静的診断の候補とする。

| 演算 | 推論できること | 推論してはいけないこと |
| --- | --- | --- |
| WHERE | 入力集合に成立していた一意性は行の絞込みで失われない | 残った行を観測しただけで新しい業務キーを採用する |
| SELECT | 既に保証されたキー列を保持し、行増殖がなければその一意性を引き継ぎ得る | 名前がIDに見えるだけでキーを決める |
| JOIN/LEFT JOIN | 結合の相手側が条件上高々1件と証明できれば左行の重複増加がない | 外部キーがあるだけで結合先の一意性を推測する |
| GROUP BY | GROUPの項目組はSQLのグループ同値性では一意 | NULL許容のGROUP項目組をそのまま業務識別子にする |
| DISTINCT | 出力**列の組全体**は重複除去される | そこから特定の部分列が一意、または持続的な業務IDとする |
| UNION ALL | 各枝の由来と重複を保つ | 枝ごとの一意性だけで統合結果を一意とする |
| ORDER BY/LIMIT | 結果に提示順/件数上限を課す | 無指定のtie-breakerや表示行番号を業務キーとする |

重複して**値が完全に同じ二行**を、画面の値だけで一意に識別できると仮定しない。UIの行ハイライトに試験実行専用の観測IDを付けることはできるが、再実行をまたいで安定するBusiness identityではない。結合や集計のrow lineageを示す場合は入力fixtureの行IDとscope/operator IDに束縛し、追跡不能なら「未取得」と表示する。

## 4. 構造とモデル表示の実証対象

最小SQL（業務名/入力は架空の合成例）：

```sql
WITH line_totals AS (
  SELECT l.order_id, SUM(l.quantity * l.unit_price) AS amount
  FROM order_lines l
  GROUP BY l.order_id
),
open_orders AS (
  SELECT o.order_id, o.customer_id,
         CASE WHEN t.order_id IS NULL THEN 0 ELSE t.amount END AS amount
  FROM orders o
  LEFT JOIN line_totals t ON o.order_id = t.order_id
  WHERE o.status = 'OPEN'
)
SELECT q.order_id, c.customer_name, q.amount
FROM open_orders q JOIN customers c ON q.customer_id = c.customer_id
WHERE q.amount >= :threshold
ORDER BY q.order_id
```

各段階の表示：

```text
[orders] ──[親子Relation]──> [order_lines]     [customers]
                                     │ SUM/GROUP
                                     ▼
                              <line_totals>
                                     │
[orders] ───────── LEFT JOIN ──────────┘
   │ WHERE status=OPEN
   ▼
<open_orders> ── JOIN customers ── WHERE amount>=:threshold
   └───────────────> <final result>
```

上記は**説明用の配置**であり、SQLの実行順序/物理マテリアライズを示すものではない。Modelerは「エンティティ＋属性一覧」「CTEの演算DAG」「field lineage」「fixture行と中間結果」「SQL本文のsource span」を切り替え/連動させる。SELECT列を選択すればどのscope/演算/元属性に由来するか移動できる。

### 固定fixture F0（#2387のQXケースに対する補助ケース。値・期待値は独立に指定）

- customers: (C01, 青空商事), (C02, 山田商店)
- orders: (O01,C01,OPEN), (O02,C01,OPEN), (O03,C02,OPEN), (O04,C02,CLOSED)
- order_lines: (O01,1,2,100), (O01,2,1,50), (O02,1,1,50), (O04,1,3,100)
- データ型/制約前提：orders.order_id一意、order_lines(受注ID,行番号)一意、参照先存在、**F0版ではquantity/price非NULL**。整数例は桁あふれしない。通貨/丸めなど業務採用は対象外。
- `line_totals`: (O01,250),(O02,50),(O04,300)。
- `open_orders`: (O01,C01,250),(O02,C01,50),(O03,C02,0)。
- `threshold=200`、final: (O01,青空商事,250)。
- assert：O03は明細なし→LEFT JOINで残る→CASEで0→最終WHEREで落ちる。O04は集計に存在してもOPEN絞込で落ちる。

期待値はSQL/evaluatorから逆生成せず、手書きの独立oracleとして固定。fixtureは**事前にデータモデルへの適合性を検査**する。

### 反証ケース（予定。現在未実行）

| ID | 最小変更 | 期待する区別/診断 |
| --- | --- | --- |
| N1a / N1b | F0（quantity非NULL）へNULLを入れる / **nullableを明示した別モデル版**でO02.quantity=NULLを評価する | N1aはモデル制約違反として入力拒否。N1bはO02のSUM=NULL、O03のabsent group→0を区別。意味の違う二つのDataset/Model版を同じfixtureと見なさない |
| N2 | JOIN先customersの重複キーを混入 | モデル制約違反としてfixture拒否。JOINの二重計上と誤った一意性を隠さない |
| N3 | 件数制限を集計・Filterより前へ移す候補 | 不正/非同等な計画を拒否。既存#1640の先行Limit禁止を再利用 |
| N4 | GROUP BY前に同じ受注の明細が複数存在 | 2行が1集約行へ寄与することを明示。row lineageが無ければ寄与行は未確認 |
| N5 | ORDER BYの一意なtie-breakerなしにページ分割 | 安定結果を保証しない表示。実行の偶然一致で保証済みにしない |
| N6 | 未解決のCTE alias/列、サポート外の再帰/ウィンドウ関数 | opaque/unresolvedとsource span。成功したかのように空結果を返さない |
| N7 | 局所fixtureが共有fixtureと同じキーの異なる値を供給 | 合成衝突を表示して明示拒否/人の選択へ戻す。last-wins不可 |
| N8 | SQL digestは同じでmodel/dataset/rule revisionが変化 | staleを検出。旧結果を現在の意味評価として表示しない |
| N9 | Providerが対象集合の一部しか返さない | observed subsetとcomplete inputを区別。全件集合述語やCOUNTの証明に使わない |
| N10 | 参照元の読取り権限がなく、集約結果だけ許可される | trusted providerの権限分離に従い、入力行の詳細lineageを漏らさず開示能力を分離 |
| N11 | 相関EXISTS/NOT EXISTS | scalar/boolean条件・相関bindingとして表示し、普通の内部JOINとして行数を増やさない |
| N12 | 生成データの制約が矛盾/未対応 | unsatisfiable と unsupported と evaluation error を別分類 |

N1は原fixtureのO02の行を**差替え**、O03は明細ゼロのままにする（同一テスト内の衝突を作らない）。N2はキー制約検査前に問い合わせ実行しない。

### 4.1 最小SQL観測とその限界（2026-10-30追加）

[再現用SQLite試料](../../plan/fixtures/query-workbench-f0/verify_sqlite.py)は**独立のSQL実行観測**に限定する。使い捨てのin-memory SQLite、テスト用架空データ、手書き期待値のみ使用。外部DB、TEI評価器、Host/Provider、Productionの認可には接続しない。単に「期待値が一度一致した」ことをSQLの意味証明へ昇格させない。

```sh
python3 plan/fixtures/query-workbench-f0/verify_sqlite.py
```

作業コンテナで同等の初期試料を手動実行した結果（**GitHub上の保存ファイルのexact-byte検証、TEIの実行試験、CIではない**）：F0のline_totals=[(O01,250),(O02,50),(O04,300)]、open_orders=[(O01,C01,250),(O02,C01,50),(O03,C02,0)]、final=[(O01,青空商事,250)]。N1a非NULL制約違反で拒否、N1b nullable別モデルではO02の集計がNULL、N2の顧客キー重複を拒否、N3の早すぎるLimitは空結果となりF0と非同等。CI/Goテスト/実PostgreSQL/Modeler操作は未実行。SQLite内部の検証が成功しても、中間CTEを元SQL実行時の物理的な作業表として取得した事実にはならない。

**重要な追加境界**：通常のCTEに対し`WITH ... SELECT * FROM cte`を別SQLとして発行して得る値は、元のQueryと**別の実行**の観測である。純粋演算・同じ入力snapshot等の条件を固定した説明用の結果として扱う。時刻・乱数・volatile関数、再帰CTE、DMLを含むWITH、複数参照CTE、評価停止条件、環境/権限差があれば、別実行の中間結果を元SQLのactual中間結果と断定できない。PostgreSQLのCTEはinline/実体化等の選択が可能（https://www.postgresql.org/docs/current/queries-with.html）。Explain/Analyzeの物理的な推定・計測行数は、論理的な全中間行の値とは別の観測である。

## 5. テストデータはモデルに束縛された資産

- #1131の型付き`Dataset / Snapshot / Expectation`を所有元とし、問い合わせ局所のfixtureは独立した業務Scenario正本ではなく**Queryテスト用の局所入力と結果期待**。
- 共有Datasetを選択し、必要なEntityの行をscopeへ投影。上書きは暗黙で行わず、差分と衝突を先に検証する。組合せにはDataset revision、局所patch digest、選択基準、パラメータ、意味型・値域制約/Constraint revision、評価時計とlocaleを固定する。
- Modeler編集UIは、キー・親子・参照・サブタイプ・意味型・値域制約・NULL/欠損の区別を表示。入力値の修正はDataset側に戻し、CTE中間結果の表を直接編集しても元のデータモデルへ反映しない。
- CTE局所テストで仮想出力に直接行を与える場合、「下流scopeのみを検証」「上流未検証」を明示。全問い合わせ試験と同じ資格にしない。
- 実DBからデータ取込する際は#1807のschema/key/権限/個人情報・来歴検査を通す。外部の実データをそのまま公開fixtureへ混ぜない。
- Model/fixture/SQLの三つのrevisionは独立。テスト結果は`input snapshot + operator/SQL digest + evaluator/target + execution context + expected + observed + unresolved`へ追跡できる形で既存Evidence境界に紐付け、新しい一律Registryを作らない。

### 制約と生成（段階候補）

M1では既存の意味型・値域制約（型・範囲・列挙値）とRelation（キー・参照）の決定的な合成入力、境界値、負例を優先。M2以降にConstraint-based generation（組合せ制約、Coverage-guided generation、失敗時の縮小）を追加。Lean/SMTは一つの形式的記法候補であり、最初に新証明基盤は採用しない。任意のLean述語からの完全な生成/判定は要求しない。

生成の四つの状態を分ける：`satisfying candidate`（検査できた範囲で制約充足）、`unsatisfiable`（確認できた矛盾）、`unsupported/timeout`、`unverified`。生成できたこと、Queryの独立oracleが一致したこと、Ownerが業務意味を受理したことは非同等。意味のあるNULL/欠損/空集合や多重度はOwner未決なら仮定を明記する。

## 6. 解析・評価・実行の責務と停止条件

- **解析**：native SQLは正本、Scope/IRは導出。クエリ名やSQL aliasから業務意味を推測して昇格しない。構文/解決不完全なら各要素にcoverageを表示。
- **意味評価器**：主要業務のRead-only関係演算を外部Hostなしで検証する候補。現在の`DerivedSelect`はCTEを拒否するため、そのまま実装済みとして接続しない。対応演算と意味同値を独立fixtureで限定実証する。
- **SQL Provider**：必要な権限と同版snapshotがある場合にsandbox/使い捨てDBで評価。テストのために元SQLを切り出して中間結果を得る場合、書換え後の問合せと元の意味同値を確認できない限り**説明用の派生評価**と表示し、正式結果と同一視しない。
- **実行計画**：#1640のFilter/Derive/Aggregate/Sort/Limitの依存順を守る。全行に対する述語は、Providerが返却した集合Rへの`checkAll(R)`だけで、業務対象集合Lの性質`P(L)`を証明しない。`R=L`を検証する入力同定・Provider完全性・同版鮮度は別の証拠。
- **副作用**：Queryの局所試験はread-only。業務Actionの変更前後、Commit、Receipt、Replayは#1129/#1174/#1180の別経路で検証し、表示上の成功を実際の業務効果へ読み替えない。
- **開示**：権限のないフィールドや中間行を表示しない。許可された集約結果から保護対象の元行を逆表示することを禁止する。
- **停止**：NULL/文字照合/丸め/時点/重複/相関/Recursive等の意味が実現先で一致する根拠がない場合、unsupported/unresolvedを示しsilent fallbackしない。

## 6.1 直交するScopeと説明の可否

他レーン（#2386組織/時点、#1769行スコープ、#1706集合完全性、#2056入力NULL）との接続では**一つのScope enumに統合しない**。少なくとも四つの独立軸を使う。

| 軸 | 主な問い | 誤って同一視しない対象 |
| --- | --- | --- |
| Lexical SQL Scope | このCTE/alias/相関列はどこで見えるか | Tenant/業務対象の許可範囲 |
| Business/Temporal Scope | どの受注/組織関係・業務有効時点が対象か | SQLの単なるWHERE文字列 |
| Read-set Scope | どの集合Rが取得され、R=Lと証明できるか、どのrevisionのsnapshotか | Providerの返したページや一部subset |
| Authority/Disclosure Scope | 誰がどの元行・属性・集約結果・Lineageを見られるか | モデルに宣言された組織上のRole |

「取得できない」「0件だった」「業務集合が空」「認可により非公開」「評価不能」「NULL」は別の状態。NOT EXISTSや全件合計での不在の確定は、対象集合と読み取りの完全性・鮮度・権限の根拠が必要。要約だけ開示可能な場合に、詳細行lineageを画面へ漏らさない。

組織例（#2386、Draft PR #2389）への将来適用では「組織の複数経路JOIN→重複計上」や「as-of組織版・Rule版だけ変更・委任失効」を別の反例とする。Query説明結果が同じでも、Rule版・組織版/Grant版が異なれば現在の判断/表示が有効とは限らない。組織用の新規SQL実行器・認可正本は導入しない。

### 6.2 説明用の評価表の最小レコード（表示向け概念、公開schema非採用）

- Queryのauthoring sourceとsource digest（native SQL、2waySQLならtemplate source digest、選択条件/適用分岐、実際に評価するSQL digestを分ける）、解析器/言語/Operatorの固定版。
- Query scope/演算IDと入力/出力のField IDs、parse/resolution/lowering/説明coverage。Opaqueや解決不能の残りを正常値へ埋めない。
- model revision、Dataset/局所patch revision、parameter presence（missing/null/present）、snapshot/read-set completeness、評価時点/時区間、Rule/意味型・値域制約 revision。
- **expected（独立oracle）**と**observed（評価器/Providerの実測）**、行/多重度の比較規則、unorderedならbagの多重度比較、ORDER BYの保証がある場合だけ順序比較。デフォルトの行番号をIdentityや安定ソート保証にしない。
- Evaluation path：説明専用/TEI意味評価/別SQLへの差替え実行/元SQLの実行/対象DB物理計画の観測を区別。元SQLとは異なるSQLへの書換えを同一実行結果として表示しない。
- Completeness/Authority/Provider/Execution failure、stale reason、証拠未取得の状態を別欄に記載。署名・byte一致・可読性評価は別資格。

### 6.3 実装順に伴う停止条件

- M0ではCTEの解析coverageを固定SQLで確認。Query側のscope-lowering coverageがFULLでも、Evaluatorの対応範囲は個別判定。
- M1ではread-only図/表だけを出す。『CTE結果』として示す手書き期待値はexpectedと明記し、Provider actualと混同しない。
- M2では既存#1131のDatasetで独立期待値・型付き検査・反例を実施。`DerivedSelect`の拒否を緩めず、必要なExecutor追加は#1640/#1121の別レビュー。
- M3でModeler/SHOの版付き表示を展開。文書内の画像・参照先・説明文の改訂とSource/fixture/Ruleの改訂を分ける。説明を読めるかは「なぜO03=0か」「O02=NULLの理由」「GROUP後の一意性が業務IDでない理由」「全件集計の完全性」を人が正しく再説明できるかで圧力試験。

## 7. 表示とSHO設計書

1. **構造ビュー**：実Entityと仮想Relationの図。Entityの親子/参照は関係線、CTE導出は演算線。ひとつのQueryを高密度表・図・ツリーに投影。
2. **値ビュー**：fixture行、演算前後の行数、行の残留/除外、金額などの値変化。行を選択すると確認できた根拠範囲だけ強調。
3. **論証ビュー**：key/FD/cardinalityの根拠、型・NULL/重複、解析coverage、期待/実際の差、未検証の前提とその依存範囲。
4. **Sourceビュー**：SQL本文のsource spanと選択したscope/field、編集できるauthoritative sourceへの戻り先。派生中間表を編集可能な正本として誤認させない。
5. **設計書ビュー**：SHO向けに入力Entity、各CTEの役割、仮想表、試料、期待・観測結果、未確認事項を配置。説明文、モデル由来の図、読み取った画像などの表示依存を分け、固定納品版と更新可能な作業版を区別。

#1803と既存`reference-design-output`のsourceRevision/structureRevision/artifact digestを再利用する。fixture revision、evaluated context、evaluator version等は**説明に必要な追加の来歴参照**であり、artifact byte一致だけで評価・意味・人のレビューの現在性を保証しない。SHO Composerの別文書正本を作らない。

### 7.1 Plugin-firstと外部Consumerへの投影（#2390と整合）

- **解析/DSL**：SQL構文・テンプレート分岐・Operator意味は#988のLanguage/Operator Pluginによる解釈候補。Pluginを組み込むことだけで受理済みMeaningを拡張しない。
- **純粋な説明出力**：既存のDerivedIR/RelationalIRと許されたモデル参照から、仮想関係、列lineage、演算依存、Coverage、source locator、許可済みfixture例を返すread-only投影。出力構造を新しい公開標準ABIへ即時固定しない。内部のGo実装やTEI Studio UIを**必須の実行依存にしない**構成を最初に比較する。
- **Consumer**：Modeler、SHO文書、CLI/静的HTML、外部Host（例：外部Editorのプラグイン）から利用できるように、構造の提示・ナビゲーション指示・画面描画を分離。外部Hostでの呼出し可能性は#2390のC0〜C3資格を参照し、Query ExplainのReferenceだけで外部利用可能と宣言しない。
- **操作と権限**：read-only Explainの契約はDB read grant、Sourceを書き換える権限、テストデータの注入権限、AI/自動テスト向けUI外部制御権限を内包しない。必要操作は既存のHost/Invocation Context、Provider binding、明示Capabilityを介す。生成アプリで外部制御プラグインを含めないBuild構成も独立に維持する。
- **AIハーネス**：最小のtyped inspect/explain/fixture-compare結果が読めれば、AIはユーザー向け可読性と根拠を確認できる。AIが承認/受理済み意味へ変換することや、GUI制御を必須の正確性証拠にすることは避ける。

最初のP0受入では、少なくとも**CLI/JSON等のUI非依存出力**が同じscope/field identityを参照できることを確認し、Modelerへの投影は次段階とする。このConsumer境界は、TEI専用Hostを再導入することを意味しない。

### 7.2 M0の先行Gate：SQLワーカーの出所、スコープ付き束縛と説明の真偽（2026-10-31追補、未実行）

[#2387のQX-12 相関サブクエリ・scalar・LATERAL・Windowの検討](https://github.com/hat47x/tei-platform/issues/2387#issuecomment-6534035206)を踏まえ、複雑なSQLの演算対応に進む前に、**現在存在する単純な列参照の解決でも誤った「完全解決」が生じないこと**を先に保証する。旧X-TEAの設計例、組織の関係経路、Smart DataのFunction Port、Value Contract/型付き項目にも共通する根拠境界である。

**コードに基づく所見（main `4325ca79a52bf0d576db6fb4027bdc516b1023fd`、実Goテスト未実行）**

- `internal/query/analyzer/client.go` は `RunPythonWorker` を備え、`internal/query/analyzer/model.go` の `ColumnOccurrence` は `ScopeID` と `Correlated/OuterScopeID` を備える。一方、`reference/workspace/plugins/tei-editor-intelligence-sql/1.0.0/plugin.tei` は外部 `python3 -m tei_sql_worker` と `sqlglot` を宣言するが、確認したmainの全追跡ファイル内に `tei_sql_worker` のPythonソースはない。配布済み外部パッケージや開発環境への導入の有無は**未確認**。同宣言を「クリーンcheckoutでSQL→wire ResultのE2Eが可能」とみなさない。
- `DeriveFromAnalysis` 内の `resolveColumn` は、物理列解決に**ScopeID/OuterScopeIDを渡さず**、平坦な `TableAliases` と全Scopeの `SourceRef` を見て先行一致する別名を採る。内側と外側で同じ別名が異なる物理表を指すと、異なる列が同じ表へ誤束縛され得る（**静的コード読取りに基づく反例候補**。実行結果の観測ではない）。
- 同じ物理表の二つのSource occurrenceを使った自己JOINでは、列の曖昧性を表名で `uniqueStrings` に潰すと、未修飾列を一意に解決したと誤認し得る。名称が同じでもSource occurrenceが二つなら同一参照ではない。
- `outputByScopeName[scope][lowercase(name)] = outputID` は、同Scopeに重複する出力別名があると最後の列で上書きし得る。CTE列参照を「一意」と断定するには実際に一意か検証が必要。またCTEの `SourceKind` が`cte`でも、`Binding` が存在するScopeIDを参照しているかを検証せず `resolved++` する箇所がある。
- 既存の `Coverage.Parse=full` は `result.Status=ok` から、`Coverage.Lowering=full` はworker側の申告文字列から導出される。これらと、**スコープ付き物理列解決・CTE参照・実SQLの利用形態まで一貫して確証されたこと**を同値扱いしない。

**M0の最小反証テスト（いずれも今回NOT RUN）**

| ID | 入力/対象 | 判別すべき事実 | 不合格／停止条件 |
| --- | --- | --- | --- |
| QX-M0-1 | クリーンなmainとPlugin宣言 | SQLワーカーの実物・配布単位・依存固定版・実行環境・protocol version・出所を特定 | descriptorだけでworker実装・SQLGlot実行をPASSにする |
| QX-M0-2 | rootで`public.orders AS x`、subqueryで`public.order_lines AS x`、両方に`order_id`列 | inner `x.order_id`→`order_lines`、outer `x.order_id`→`orders`、同じalias文字列でも別Source occurrence | 先に列挙されたorders、または平坦Alias Mapで双方を同じ表へ束縛し、`Resolution=full`と表示 |
| QX-M0-3 | `orders a JOIN orders b` 内の未修飾 `order_id` | 二つのSource occurrenceがあるため曖昧性を維持し、実SQLが拒否されるなら解析/実行エラーを保持 | 物理表名が同じことだけで曖昧性を消去 |
| QX-M0-4 | 同ScopeのCTE出力別名二重宣言、CTE Bindingの未存在ScopeID | 出力別名の一意性・Binding参照先・未知列を検査し、少なくとも未解決/拒否 | mapの後勝ちや存在しない`Binding`から完全Lineageを描く |
| QX-M0-5 | outer相関列、inner同名列、`EXISTS`/scalar/LATERAL各利用形態 | `OuterScopeID`と式の利用形態は独立。具体的利用形態が現wireで未識別なら部分/unsupported | 相関という理由だけでscalar/EXISTSを普通のJOINへ推測変換 |
| QX-M0-6 | source SQLのみ変更、Schema Snapshotのみ変更、Output ordinalのみ変更 | 同じ名前・位置でも別identity/意味の場合は再解決・stale表示。構造・Lineage・fixture・評価観測の無効化境界を区別 | 古い解決済みLineage/受理済み意味を自動継承 |

最初の作業単位は `internal/query/derived_test.go` の**synthetic `analyzer.Result` を使うQX-M0-2/3/4の負例**と、既存 `resolveColumn` の対象範囲の狭い修正を別の小Draftで検討すること。今回の計画文書はコードを変更しない。Testを実行しないうちは不具合の実測PASS/FAILとは呼ばず、原因候補として保持する。誤束縛は見た目の問題だけでなくSource/Field lineageが誤った物理表を指すため、単なる`unresolved`より危険な**false-positive resolution**として先行対処する。

次に、実SQL→ワーカー→`analyzer.Result`→`DeriveFromAnalysis`へ同じSQLを流して、静的Go fixtureと差分比較を行う。workerが利用できないなら、実物と出所を特定する段階を明示的に停止し、Goで手組みしたResultをE2E証拠へ読み替えない。Plugin配布・Host/実行許可は#988/#2390/#951を参照し、通常のQuery説明のために独自Hostを追加しない。

### 7.2.1 M0の実装修正候補とその資格（2026-10-31）

QX-M0-2/3/4への最初のGo側対策は、先行[PR #2398](https://github.com/hat47x/tei-platform/pull/2398)に分離された。その差分と8件の回帰テストは後続[Draft PR #2454](https://github.com/hat47x/tei-platform/pull/2454)が包含することを両headで照合し、**二重マージを避けるため#2398を未統合のままクローズ**（2026-11-01）。実装の現役候補は#2454に一本化する。#2388は引き続き設計/fixtureのみのDraftであり、両者を同じ完成判定にしない。

- コード候補：`ColumnOccurrence.ScopeID`と明示`OuterScopeID`に基づく物理・論理Source occurrence解決、自己JOINにおける出現箇所保持、CTE出力別名重複の後勝ち回避、存在しないCTE Bindingの未解決化、物理＋CTEに同名列がある場合の曖昧性維持。
- 合成Go回帰テスト（Draftへの追加定義であり**まだ実測ではない**）：同名Alias異Scope、内外の無修飾列、自己JOIN、相関した外側物理列、重複CTE出力、存在しないBinding、物理＋CTE出力衝突、相関した外側CTE列。
- 先行#2398はGo試験を実行していないが、後続#2454は別環境（Ubuntu 22.04/Go 1.27.3）の限定headで関連3パッケージのrace試験・vet/gofmtが成功した旨を記録。**現head `06c2007b71b3675f4e0bffc35b004a7dda99aed7`のexact検証はこの計画更新時には再取得していない**。実worker/実SQL E2E/実Providerの検証も未実施。過去headの成功を現headのPASSへ昇格させない。
- 特に残る問題：CTE Bindingの**可視性/名前解決**はtarget scopeの存在・kindのみでは確定しない。SELECT出力のデータ型・NULL可能性、引用識別子の方言別大文字小文字規則、外部workerの出所・runtime・SQLからの実写像も後続Gate。Go側の誤束縛防止が単独でAnalyzerの完全性やSQL実行許可を与えない。
- 合格条件：exact headのfocused Go testsが再現可能にPASSし、残る未解決/曖昧性を`Resolution/Lineage/Coverage`として説明でき、既存のCTE合成testとD2拒否が後退しないこと。次にQX-M0-1の実worker実体の特定と、実SQL→wire→IRを別の資格として実施する。

### 7.3 設計者が受入できる最小の閲覧操作

最初のread-only Consumerは、**SQL本文の一つの式を選択→対応するScope/Source occurrence/演算/出力列へ移動→逆方向に根拠へ戻る**という一往復を最小操作とする。静的解析のCoverageと評価証拠は別欄、未解決・反例は非表示にしない。CTE中間表・相関した外側行ごとの例は固定Dataset/Scenario参照から示し、テスト値を実DBの物理中間値と呼ばない。

合格条件は、同じ別名を持つ二つのScopeと同じ物理表を二度参照する自己JOINにおいて、選択先が正しいSource occurrenceか未解決であり、他Scopeへ誤Navigateしないこと。失敗した場合は投影を広げる前に束縛・source locatorの不確実性を表示する。編集・Source Acceptance・Host実行・Smart Data Hookはこの操作に含めない。

### 7.4 QX-14：解析の入力版と結果の帰属（2026-11-01 横断更新、提案）

**現在の分担と実際の状態**（確認時main `43517af5450aacb2fdd6c48585dc143f52d57da4`）：

- QX-14aの**別Query ID/空ID拒否**は[PR #2465](https://github.com/hat47x/tei-platform/pull/2465)でmain統合済み。これによりID不一致の手組みResultは`Coverage=none`となるが、**同じIDでSQL本文を変更した場合の解析結果が現在版に属すること**までは示さない。
- QX-14dのWorker request/response **Query ID・protocol版照合**は[Draft #2471](https://github.com/hat47x/tei-platform/pull/2471)（候補、未統合）。ダミーワーカーの関連試験は報告されているが、実SQLGlotワーカーの配布実物/互換性、実際のSQL→IR一巡は未確認。
- QX-13のSource occurrenceスコープ/曖昧性の防御は[Draft #2454](https://github.com/hat47x/tei-platform/pull/2454)が旧[Draft #2398](https://github.com/hat47x/tei-platform/pull/2398)を取り込み、未知Source・誤ったOuterScope・循環/欠落/重複Scopeの追加負例まで含む。特定headで`race`/vet/gofmtの報告あり。ただし**現在head `06c2007b71b3675f4e0bffc35b004a7dda99aed7` に対する今回の再実行は未確認**。同じ修正を独立に二重マージしない。
- 型付き文字列投影の4状態（present/empty/missing/unresolved）に関する[PR #2489](https://github.com/hat47x/tei-platform/pull/2489)、読取り整合・参照不在の[PR #2487](https://github.com/hat47x/tei-platform/pull/2487)はmain統合済み。しかし前者の成功は**Queryの業務型／結果行の意味受理**ではなく、後者の合成Referenceの診断は**実Provider横断のsnapshot証明**ではない。

**QX-14b/cの第一級の欠落**：`analyzer.Request`は`Query/Source/Dialect/Schema/ProtocolVersion`を持ち、`analyzer.Result`の現行wireには入力SQL原byteのSHA・Schema identity・実Worker artifactへの直接の帰属がない。単にResult.Queryが等しく`Status=ok`であっても、旧Source S1解析を同一IDの現行Source S2の`Lineage`として見せられる余地がある。元SQLのsource revision/byteを採用済み意味や実行権限と混同しない。

M0→M1の**説明用入力固定（新Core/Wire schemaの先行採用なし）**：

| 入力区分 | 記録・照合対象 | その違いがあるときの扱い |
| --- | --- | --- |
| Authoring | Query CanonicalID、**native SQL原byte digest**、2waySQLならtemplate原byteと実際に展開したSQL/選択branch、Query編集revision | 新SQLへの再解析。旧解析は同一IDでもhistorical |
| Analysis | Dialect、protocol版、SchemaContextの内容/digest・観測時点/identity、worker実行物・依存版、解析Operation | 互換性不明なら`unverified`/unsupported。workerの自己申告digestだけを因果証拠としない |
| Meaning binding | Accepted Semantic revision、明示raw→Semantic Field mapping、意味型・値域制約/Rule版 | SQL構造の参照だけは残せても、意味・制約や妥当性を再照合 |
| Example/evaluation | Dataset revision、局所patch、parameterのmissing/null/present、Operator/evaluator、入力scopeと取得完全性・整合要求 | Expected・Evaluatedを別に失効/再検証。閉じたfixtureの通常0件を毎回未判定にしない |
| DB observation | Provider/DB条件、SQL実行入力、読取り保証・snapshot、観測Run、Authority・Disclosure context | 現在の実DB観測と歴史的証拠を分離。Query構造用の中間別SQLは元SQLの物理中間結果と呼ばない |
| Presentation | Modeler/SHOの対象表示契約、Source/Template/文書・表示版 | 見た目だけの変更は元Source/意味の変更と区別。byte一致で人の受理を推定しない |

必要なのは一個の「全体Fresh/Stale boolean」より、**どの主張・Projectionを再使用できるかの依存マトリクス**。例：

- SQLの改訂（同一IDの`SELECT qty`→`SELECT price`）：構造・Lineage・SQL由来の期待値は原則再確認。以前のRunは historical に残す。出力FieldID（scope+ordinal）が同じでも同一意味とみなさない。
- Schemaに同名列を追加：SQL textと表示テキスト自体は同じでも、無修飾列の解決/Lineageは再確認。ScopeのLexical構造は保存できる範囲がある。
- DatasetやParameterだけ改訂：純粋なSQL構文図の再生成は不要な場合があるが、段階別Example/Expectation/評価/観測は対象Input Contextとの一致を確認。
- 業務Rule/意味型だけ改訂：SQL構文の妥当性は直ちに変わらなくても、モデル上の許容値や業務上の説明・Expectedは再審査。Owner受理と文書の固定納品版は別。
- 表示テンプレートだけ改訂：SQLの検証結果を単純に破棄しない。表示上のread-only説明と文書成果物のdigest/受理状態を再検証。
- 認可だけ改訂：解析済み構造が過去に正しかったことは残せるが、**現在の値・実行結果の開示**はHost/Providerの現在Authorityで判定。非公開の行ID/件数を診断から漏らさない。

最小反例を**QX-14b（same-ID source edit）／QX-14c（same SQL・different schema）**とし、生成物/表示の`current`と`historical`が区別できるかを先に問う。実Workerへの送信元で同一Invocationと固定入力を束縛する実装責務は#988/#2387に寄せ、D2のSQL digest検査や#1708のread consistencyを**Query Explainの証明済み境界**に読み替えない。

### 7.5 結果の粒度（Grain）・参照Identity・Function Data Contextとの接続（2026-11-01提案）

[Issue #2506](https://github.com/hat47x/tei-platform/issues/2506)はFunction Data Contextの**型付き参照・結果集合・ページ・選択epoch**、[Issue #2507](https://github.com/hat47x/tei-platform/issues/2507)は機能Patternが要求するShape/Grain/Operation/Rendererを担当する。#2387が所有するのは**SQLから読み取ったQuery-localな列・演算・出力の静的/fixture上の意味**まで。Query結果をFunction内の更新対象Entityに自動変換しない。

| Query境界／観測単位 | SQLに基づく読み方 | TEI意味モデル/Functionへの接続判断 |
| --- | --- | --- |
| `order_lines` | 明細行（同一orderに複数） | (order_id,line_number)等の**明示された**業務識別条件がある場合に限る |
| `line_totals` | `GROUP BY order_id`の**グループ単位**。O01=250、O02=50。O03のグループ行は存在しない | `order_id`は集約グループのキーだが、その集約行が受注Entityそのものになるわけではない |
| `open_orders` | LEFT JOIN+CASEにより、明細が無いO03も**出力行**を持ちamount=0（モデル/fixture前提） | `orders.order_id`の一意性と`line_totals`のグループ単位が確認できる範囲で1受注1行。集約のNULLと欠落は異なる |
| `final` | JOIN・threshold判定を通った**Query結果bag**。F0ではO01だけ | `customers.customer_id`の一意性と演算条件が固定されている場合に限り、O01は受注の説明対象に束縛可能。表示行index≠Action対象Identity |
| D1(2件)×D2(3件)のJOIN | 同じHの二つのRelation Pathが結合すると**6行**に増幅し得る | 6行を業務上の6明細や6承認対象とみなさない。D1/D2は#2506の独立Contextとして保持 |
| `DISTINCT`/Window/再帰 | 列組の重複除去・行ごとの値・反復Pathの帰結 | 業務Entity/Path occurrence/集約のIdentityを推測で作らない。未対応演算はCoverageを分ける |

**粒度は複数Facetの組合せ**：Entity/Relation occurrence/Path occurrence/Aggregate group/Query result rowという観測単位、Business identity（受理済み）とQuery occurrence identity（局所）、多重度、NULL・missing・empty、Read-set completeness/版、編集可能性/Actionの前提を**直交する注釈**として持つ。SQLの`GROUP BY`や`DISTINCT`は表示・評価の粒度を変えるが、Entityを新規受理する行為ではない。既存#1126/#2495の意味型・構造Facetを再利用し、Query用の第二Meta Modelを導入しない。

**QX-15（設計受入候補、NOT RUN）**：
1. `line_totals`のO01（明細2行→集約1行）と`open_orders`のO03（LEFT JOIN後の受注1行、金額0）を横並び表示。行数/出力grainが同じでも**根拠と意味**は異なる。
2. D1=2/D2=3の分離ContextとJOIN=6を比較し、`6`をEntity数/Relationship実体数へ自動採用しない。並列パスは#1706・#2506の既存fixtureを参照する。
3. Query結果を表/チャート/SHO設計書へ投影しても同じSource/Scope/FieldとGrainへ戻れる。集約行の逆更新/クリックによる業務Action対象が一意でなければ、必要な明示Bindingが得られるまで編集Intentは保留する（実行許可は既存#951/#1174）。
4. 同じQuery IDのSQL変更、Dataset変更、Context選択A→B時の遅延応答を別試験とし、前回Runを現在のFunction Contextへ表示・混入しない。選択epochは#2506が所有。
5. `best-effort`の混合入力と`snapshot`強要求は[Issue #1708](https://github.com/hat47x/tei-platform/issues/1708)の既存契約に合わせる。参照先不在、Query未取得、NULLを区別し、異Provider間の同時点保証を推測しない。

最初のConsumerはCLI/JSONのread-only一覧（scope→operator→出力grain→独立fixture oracleのexpected→実観測/未実行→元SQL source位置）。Modeler/SHOはこの同じ説明構造を観測者に応じて可視化・文章化する。**SHOの言語的な補正がAccepted SQL/Business Meaningを無断変更することはない**。新しい正式`Grain`公開型やRow provenanceエンジンは、このQX-15小さな試験で採否を判断する。



### 7.6 QX-16：演算DAG・出力Claimに相対的な入力充足と部分結果の片側判断（2026-11-01追補）

**目的と分担**：[QX-15の同額実体・JOIN増幅](https://github.com/hat47x/tei-platform/issues/2387#issuecomment-6557423877)と[Function Contextの入力依存別完全性](https://github.com/hat47x/tei-platform/issues/2506#issuecomment-6557431216)を受け、**SQLのどの出力・比較Claimが、実際の演算経路上のどの入力集合に依存するか**を読み取り専用Explainで観測する。部分集合から片側の結論が成立し得る数学的前提は[#1063のPM-1〜6](https://github.com/hat47x/tei-platform/issues/1063#issuecomment-6557440885)を参照し、Operatorの数学法則・意味型/単位の受理は同Issue等のOwnerへ委ねる。Query側では**演算形状によって入力集合の包含関係や行の多重度に関する仮定が保たれるか**を検証する。新たな証明器・Evaluatorは導入しない。

#### 三つの依存を分ける

1. **式／値への寄与**：SUM(l.amount)の元金額列など。既存OutputField.InputsとField Lineageは静的な列依存の足場になる。
2. **行の存在・評価条件への依存**：WHERE/HAVING、JOIN相手、EXISTS/NOT EXISTS、ORDER BY/LIMIT等。**出力列に現れなくても、結果の真偽・件数・粒度に影響する**。Field Lineageだけではこの依存を完全に説明できない。
3. **当該Claimの確定条件**：Source population、Identity、対象Scope、Predicate、入力の包摂／完全性、bag重複、値域制約、単位・通貨、時点・読取り整合、実際の開示条件。Sourceに関する観測とSQL演算の論理的な結果を別に保持する。

**同じ表示名でも演算経路が違えば値が変わる**。明細二件L1/L2（金額600・500）、配送履歴二件D1/D2なら、明細単独のSUMは1100。明細を受注単位に集約し、配送履歴の件数も独立集約した後で結合する例ではlineTotal=1100を維持する。一方、明細と配送履歴をJOINした**後**で金額SUMを取ると2200。2200というSQL結果が誤計算なのではなく、これを**「重複なしの業務明細総額」**と説明・利用することが誤り。元列Lineageに明細金額のみが示されていても、JOINによる行多重度の依存は消えない。事前集約が常に同値な書換えであるとも主張しない。

#### QX-16：Operator別の条件付き判別票（表示候補／未採用）

| Claim／演算 | 部分取得で成立し得ること | 停止・追加根拠の条件 |
| --- | --- | --- |
| EXISTS(P)／NOT EXISTS(P) | 同じ母集団・述語・版の一致行を1件確認できればEXISTS=TRUE、NOT EXISTS=FALSE | 一致行の未観測からEXISTS=FALSE／NOT EXISTS=TRUEを全集合の事実として宣言しない。閉じたfixtureの正常0件は通常どおり確定可能 |
| 非負値のSUM | RがLに同一Identity・単位・時点で包含され、未観測値にも非負性が保証され、JOIN等で重複増幅しないならSUM(R)はSUM(L)の下界。SUM(R)≥Tから片側の閾値結論 | exact sum、SUM(L)<T、負値許容、NULLの算術規則未決、単位/通貨混合、版ずれ、JOIN fan-outは独立評価 |
| COUNT(*) | 観測したQuery入力bagの出現数は包含・重複の条件付きで件数の下界となり得る | 業務Entity件数、COUNT(DISTINCT)、JOINの出現行数を同一視しない |
| MIN／MAX | 比較可能な観測値があれば、全集合MINは観測MIN以下、全集合MAXは観測MAX以上 | 未対応順序・collation・NULL・未取得・時間変化があれば法則の適用を保留。最終極値を確定しない |
| NOT EXISTS／差集合／反結合 | 実際に条件へ一致するWitnessは不在Claimの反例になり得る | 不在の確定には当該述語・母集団に対する閉じた入力範囲が必要。秘匿範囲を推測表示しない |
| LEFT JOINの未一致行／右列IS NULL | 閉じた入力・確定したJOIN条件では、右一致なしの行を説明可能 | 右側が部分取得のとき、仮の(H,NULL)行は右行D1が追加されると(H,D1)に**置換され得る**。入力包含R⊆Lでも**LEFT JOIN出力bagの包含は一般に成り立たない**。CASEで0を埋めた値も全集合の確定0とはみなさない |
| ORDER BY/LIMIT／Top-N | 観測済み範囲内の順序・順位候補を説明可能 | 未取得候補・タイブレーカー・方言・collationを無視して全体順位を確定しない |
| 不透明な式・Window・再帰CTE | 既存IRが取得したScope/Source occurrenceを表示 | Window frameや再帰停止条件・演算利用形態が現wireに無ければunsupported/partialに留め、数学的性質を推測しない |

#### 最小判別ケース QX-16a〜g（互いに独立した確認列）

- **a：同一観測と異なる全体**。観測R={A:600}に対しL1={A:600,B:500}はSUM=1100、L2={A:600,B:1000}はSUM=1600。部分観測だけでexact sumや合計<1500を確定しない。非負などの適用条件が受理・確認された場合の「合計≥500」だけは片側の結論となり得る。
- **b：値域制約変更の反例**。負のB=-200を認める別の値域制約revisionならSUM=400。「SUM」の名前や観測済み値が正であったことを非負Lawの根拠にしない。
- **c：JOIN fan-out**。A=600/B=500を配送履歴2件とJOINした後でSUMすると2200。これを元の明細総額1100の下界に転用して合計≥1500という業務Claimを成立させるのは誤り。Operator DAG上でJOIN前/後を区別する。
- **d：存在と不在**。観測R={A:600}でamount≥500のWitnessがあればEXISTS真／NOT EXISTS偽を限定的に示せる。一方、Rが空でも真の全集合は空または未観測Aを含む可能性があり、不在は決まらない。SQL三値論理UNKNOWNと観測不明を区別する。
- **e：独立依存**。明細の同一必要版・必要Scopeでの全集合2件がそろい、配送履歴は1/2件のみ取得したとき、**明細のみを集約するノード**の正確なSUMは条件付きで使える。配送履歴JOIN後の集計や全配送Countには波及させない。Context全体の単一FULLビットで一律に止める必要もない。
- **f：現在性と用途**。t0の成立根拠は履歴として保持する。tCommitや表示時に対象・値・規則・認可が変わったら現在のAction完了や実行許可へ自動転用しない。#1708/#1641の整合性、#1125/#1174のBusiness Action、#2516の用途相対Projectionを参照する。
- **g：LEFT JOINの非単調な未一致行**。固定親H、右の配送履歴未取得時に一旦右行0件で計算するとLEFT JOINは(H,NULL)、右D1を加えると(H,D1)であり、前者の出力行は後者に含まれない。さらに右ID IS NULLによる反結合出力は(H)→空へ変わる。これは入力行が増えれば出力行もそのまま残るという単調包含の**反例**であり、F0の明細なしO03→金額0という説明も、対象右集合の完全性を前提にする。入力の局所完全性を持たない段階では「受領済み範囲で未一致」と表示する。

**最小read-onlyの表示案**：Queryの正本rev／OutputField・RelNode／依存するSource occurrence・集合粒度／各入力の読取Scope・Predicate・Snapshot／Claim（exact・条件付き下界・存在Witness等）／仮定と不足／独立Expected・Observed・NOT RUN／Consumer用途。これは必要条件を観測する**表示用の候補**であり、新CoreのClaimProof種別、機械的に自動受理する証明書、第二Provider完全性機構、正式公開APIではない。#2387が演算構造、#1063がOperator Law、#2506がData Context、#2514が独立Oracle、#2497/#2500が表示写像・編集逆対応を所有する。

**現行SSOTの補足**：[docs/architecture.md](https://github.com/hat47x/tei-platform/blob/main/docs/architecture.md)が定める、SQLを直接書く方式ではSQLが正本、図／モデルから記述する方式ではモデルが正本という**著者モード別の正本**を維持する。QX-16の最初の対象はSQL著者モードのread-only Explainであり、全Query表現を一律「SQLが正本」に変更する提案ではない。

**Evidence**：a/b/c/d/gの数値・行は、ローカル隔離Python sqlite3上で独立に再構成した合成SQLの参考観測（R:600、L1:1100、L2:1600、負値版:400、二配送JOIN:2200、EXISTS true／NOT EXISTS false、LEFT JOINは右空時(H,NULL)、右D1時(H,D1)）。e/fと数学的条件の製品適用は紙上の設計候補。GitHubの[既存F0合成script](https://github.com/hat47x/tei-platform/blob/work/query-structural-fixture-design-20261030/plan/fixtures/query-workbench-f0/verify_sqlite.py)のexact blob `b5939379f77f2b343be31e3464678328a9780a82`は後続の2026-11-01に実行され、上記QX-16比較を含め期待値照合PASS。追加のread-only Referenceおよび回帰テストは次節7.7で別々に記録。TEI Query Evaluator・実Worker・実Provider/DB・Function Context/CUI・SHO/GUI・Lean・Owner意味受理はNOT RUN／未採用。

### 7.7 QX-16の最小read-only Reference：exact実測と次の境界（2026-11-01）

**完成した局所的成果**：同じDraft #2388の[合成fixtureディレクトリ](https://github.com/hat47x/tei-platform/tree/work/query-structural-fixture-design-20261030/plan/fixtures/query-workbench-f0)へ、元の`verify_sqlite.py`に加えて`explain_qx16.py`と`test_explain_qx16.py`を保存。新しいRuntime/SQL worker/DSL evaluator/Core schemaを作らず、合成fixture専用の手動`NODES`と`CASES`から**演算依存とClaim条件の読み取り専用JSON**を投影する。

- **実測A**：GitHubとGit blobが一致する`verify_sqlite.py` (`b5939379f77f2b343be31e3464678328a9780a82`)をPython 3.13.5/SQLite 3.46.1の隔離ローカル実行でPASS。旧F0/N1a/N1b/N2/N3に加え、QX-16の8つの合成SQLite比較ケースで、期待値と一致することを確認。
- **実測B**：`explain_qx16.py`のexact blob (`2b6c0b570b1da0085972fddee81d1c759ebd3d0b`)を実行してread-only JSONが生成されることを確認。**7つの明示Operator/Claim依存ノード、10シナリオ**を出力する。
- **実測C**：`test_explain_qx16.py`のexact blob (`0b47846ecb730bd3feaa53040798d2e06fbb538a`)で`python3 -m unittest -v test_explain_qx16.py` **10件PASS**。同じ観測R=600と異なる全集合、非負条件付き下界、負値モデル、Lineだけcomplete/Delivery partial、JOIN fan-out、EXISTS/NOT EXISTS、空SUM=NULL、右未一致の仮行消失、未受理状態、全Claimと依存ノードの対応を検査。
- 実行コマンド、blob SHA、説明ラベル、禁止する昇格を[README](https://github.com/hat47x/tei-platform/blob/work/query-structural-fixture-design-20261030/plan/fixtures/query-workbench-f0/README.md)へ記録。**このブランチの該当3ファイルだけの局所Python/SQLite実測**であり、TEI mainのGo/worker/Provider/Context/SHOテストのPASSではない。

**追加の負例：SQL NULL と未確定な結論の区別**。閉じた空明細に対するSQL `SUM`は実際にNULLを返す。一方、部分取得で全体SUMを断定できないときは**結論が成立していない**。両者の`conclusion`をJSONの`null`にするだけではConsumerで混同し得るため、fixture Referenceに`conclusionPresence = sql-null / not-established / value`を追加し、10件の回帰テスト内で区別を確認した。正式Status enumを新設したのではなく説明用JSONの明示フィールドであり、#2506のResult Slice・#2516の用途相対説明へ渡す時に必要な差分を測定する。

**補足：出力の粒度が変わる演算境界**。固定NODESで`join_pairs`は`join-pair:H`、その後の`join_sum`は`group:H`と分離し、後者が前者を入力に取ることを回帰テストで固定した。元の明細Entityとも別である。列の出所が同じでもJOIN出現数と集約後のGroupを同じIdentityへ写さない。

**観測からの設計判断**：出力フィールドの列由来だけでなく、演算依存（JOIN後の多重度、反結合の有無、比較条件）をたどらなければ、同じ金額の出力を「正しい業務総額」へ写せない。完全取得は単純なContext-wide 1 bitではなく、**個々のClaimが必要とする入力集合/版/Scope**を保持したときに、部分取得でも独立した明細合計を条件付きで説明できる。既存の#1063による数学的片側法則も、必要条件が成立するときに限る。

**次の具体的な縦切り（M0→M1、未実行）**：このReferenceで手書きした`NODES`を、#2454/#2471のScope/解析入力の正当性を照合した既存`RelationalIR`と付き合わせる。対象は`SOURCE → AGGREGATE → JOIN → FILTER`と`LEFT JOIN → IS NULL`の2経路のみ。各ノードについて「入力への寄与」と「結果行成立への依存」を別々に候補抽出し、現行IRに表現できないEXISTS/CTE相関/Window等は`unsupported`へ留める。QueryのJSON・CLI可視化とSHOはその後。FunctionのResult SliceやSelection Epochは#2506の責務、Operator Lawは#1063、Action/Effectは#1174、独立Oracleは#1131/#2514の既存資産を再利用する。

**未確認・Owner未決**：手動NODESが実SQLから正しく導出されること、Provider read-set包含/完全性、別Dialectでの一貫性、Business ID/単位・非負性/Action完了の意味受理、実Hostの開示・認可。局所ReferenceのPASSはSource Acceptance、実行許可、正式公開型・ADR採用、PR統合を意味しない。


### 7.8 次段M0：手書き依存グラフと既存RelationalIRの対応範囲を先に固定（2026-11-01静的照合・未実行）

main `fd59b6b1bac340b67244f9441a492c460921671a` の`internal/query/relational.go`、`derived.go`、`analyzer/model.go`を静的照合。現行`RelNode`は`ID/Kind/ScopeID/Inputs/Source/SourceKind/Binding/Alias/Resolution/JoinKind/Expressions`を保持する。`OutputField.Inputs`は元ColumnOccurrenceの参照を保持できる。一方、現時点のwireには**Claim種別・Operator数学法則・集合の取得完全性・演算ごとの結果Grain・SQL式の型付き構造を全て照合できる項目は揃っていない**。とくに`derived.go`のField Lineageは出力Columnの入力列であり、JOINの行数・WHEREの行排除・存在確認の必要母集団を全て表すものではない。

| 固定QX-16 Node／演算 | 現IRで候補として読めるもの | まだ確定不能なもの／次の反例 |
| --- | --- | --- |
| Source occurrence（明細・配送） | `RelSource`、`ScopeID`、`Source/Binding/Alias` | #2454のScope/曖昧性/unknown Sourceが未統合なら`Resolution`の意味拡大を停止。実Provider Result Sliceの全件性は#2506 |
| 明細集約 `line_sum` | `RelAggregate`、`RelNode.Inputs`、文字列`Expressions`、`OutputField.Inputs` | Group key、SUM対象、NULL算術、値域非負性、単位・通貨が型付きで保証されるかは未検証。手書き`group:H`を正本へ転記しない |
| 結合出現 `join_pairs` | `RelJoin/JoinKind/Inputs`をQuery内の演算順へ追える可能性 | join pair粒度と元Entityを別とする受入、cardinality、キーの一意性、NULL条件、二集合の取得範囲 |
| 結合後集約 `join_sum` | `RelAggregate`の前段InputがJOIN Nodeであるという演算DAG | `group:H`の集約結果と`join-pair:H`の出現行のIdentity非同値、増幅による業務総額の誤用 |
| `line_sum_gte_500`／`exists_500`／`not_exists_500` | `RelFilter`と文字列`Expressions`、scope source参照を候補提示 | 比較条件/EXISTS use-kind/相関scopeの厳密な式構造は現wireで専用表現不在。単調・非単調の法則は#1063、`unsupported`の維持 |
| `left_unmatched` | `RelJoin`のJoinKind候補＋`RelFilter`の式文字列 | RIGHT側の必要取得完全性、NULL由来、LEFT JOINの仮未一致行消失。任意SQLの式文字列から自動不在確定を行わない |

**最初の反証可能なM0比較**：手書きReferenceの7 Nodeのうち、`RelSource/RelJoin/RelAggregate/RelFilter`に対応できる構造と、`unsupported`にする部分を`DerivedIR`の合成`analyzer.Result`で観測する。ノードは「正しく導出」「構造候補のみ」「未表現」「誤ったFull」の四区分で記録し、Query入力版(QX-14)とScope解決(#2454)の証拠欄を別にする。出力は**read-only JSONの比較票**であり、ProviderのSQL実行や業務Meaning/権限を先に要求しない。一般SQL構文の解釈は既存SQL worker/DSL Owner(#988)へ委ねる。

**停止条件**：`RelNode.Kind`や`Expressions`を文字列として読むだけでPredicate/EXISTS/Window/Recursive・Operation Lawの意味が正しいと宣言しない。初回のM0で不足する情報を実測した後、既存`analyzer.Result`の対象限定拡張かConsumerの別説明候補かを比較する。新Core kind、SQL書換え、第二のEvaluator・Provider完全性証明器、可視化が動いただけのMeaning受理を導入しない。


### 7.9 QX-17：現行RelationalIRに対する構造依存トレースを実測（2026-11-02）

**追加実装（Draft #2388の同一ブランチ、既存Query Core・Worker・Go実装への変更なし）**：前節QX-16で固定した手書き演算依存を、実際に存在する`internal/query/relational.go`のJSON shapeへ投影した3種類の合成`RelationalIR`と、`RelNode.Inputs`を保守的に追うread-only Pythonプローブを[既存fixtureディレクトリ](https://github.com/hat47x/tei-platform/tree/work/query-structural-fixture-design-20261030/plan/fixtures/query-workbench-f0)へ追加。

main参照：`2197baaedbde178a042a280ea1bd1d534617bfc7`。現行`relational.go`のblobは`4a91112b557585a4799b681c6348b46d85698f59`。手書き3例の型・JSONタグはこの構造を基準とする。**SQL worker生成済みResultやGo DeriveFromAnalysis実行物ではない**。

| QX-17ケース／対象 | 保守的な構造トレースの結果 | 意図的に未確定へ残すもの |
| --- | --- | --- |
| `aggregate_before_join` | 明細集約`agg_l`はline Source occurrenceのみ、後続`join_groups`はline+delivery | SUM対象の正確な式意味、GROUP key/業務識別、演算で求められるInput completeness |
| `join_before_aggregate` | `join_rows`がline+delivery、後段`agg_after`も両集合への依存を持つ | JOIN pairとGroupの実際の行値・数量、正しい業務明細総額というClaim |
| `left_join_filter` | `outer_join`と`anti_filter`はheader+deliveryに依存 | 右集合の全件性と`IS NULL`の意味。`(H,NULL)`を実業務の不在へ自動受理しない |
| 同一物理Sourceの二出現 | Source名`line`が同じでもOperation ID別に2 occurrenceを保持 | 相関・Scopeごとの正しい束縛や受理されたRelation |
| 負例（欠落、循環、重複、未知演算、異Scope、CTE、Arity、Root欠落） | `unresolved`と局所diagnosticに分類。cross-scopeは即時不正とせず`unverified` | Worker/SQL方言で正当な可視性を満たすか、補完できるか |

**実測の範囲**：
- Python 3.13.5：`test_probe_relational_ir.py` **14件PASS**。`probe_relational_ir.py`は3例をJSONに投影し、Root構造候補と各ノードのSource operation ID・grain候補・unverified式を出力。
- GitHubに保存した3ファイルとローカルで実行したファイルの**Git blob SHA一致**：`probe_relational_ir.py = 733d36dcfe4cd30becf0f5825e9a4ae2b863923d`、`relational_ir_synthetic.json = 9a47ab095e504798726976239dbf2c56dd123ce5`、`test_probe_relational_ir.py = ecd704ea714ecba996da60b6cb35a767c712d616`。
- Go 1.23.2の隔離テストでは、現mainの`relational.go`と**バイト同一の型定義**を使用し、3種類の合成IRの`encoding/json.Unmarshal/Marshal`往復1件PASS。これは*Go型とJSON shapeの限定照合*。型ファイルをRepoへ複製・導入していない。repo指定Go toolchainでの正式`go test`や`DeriveFromAnalysis`、worker/実Provider/GUI/SHOは未実行。

**QX-16との4点接合（fixture-only実測）**：前段`explain_qx16.py`が出力した`operationDependencies`を読み、今回の合成IRの`agg_l`→`line_sum`（line）、`join_rows`→`join_pairs`（line+delivery）、`agg_after`→`join_sum`（line+delivery）、`anti_filter`→`left_unmatched`（header+delivery）という**手動の対応表**でSource collectionの集合を比較した。ローカル実行では**4/4一致**。比較対象の手書き`join_sum`からdeliveryを外す負例は`unresolved-or-mismatch`として検出し、`test_probe_relational_ir.py`の**14件目**に追加。**任意SQLのSource/Claimを自動対応付けた結果ではなく、独立に表現した同じ固定fixtureの依存構造が矛盾しないかの照合**である。

**構造を読めることとClaimが成立することの間の欠落**：
1. `RelNode.Inputs`は演算Node ID、`OutputField.Inputs`はColumnOccurrence IDに由来するField Lineage。**出力Field→どのRelNodeの演算・GROUP/WHERE条件で成立したか**を現IRだけで自動認定する追加根拠は未確認。2種類のIDを短絡して結ばない。
2. `RelNode.Expressions`は文字列配列。SQLの比較や`EXISTS`のPredicate、`SUM`の対象/NULL・非負値域型、Window/Recursive、SQL dialectを型付きで十分説明したと推定しない。プローブの`expressionSemantics`は`unverified`、`claimConclusionPresence`は常に`not-established`。
3. `RelNode.Resolution`や`Coverage.Full`が仮に存在しても、Provider read-set、必要なsnapshotの一致、Ownerの業務Meaning/受理、Host実行・開示許可の証拠にはならない。source scopeの判定はDraft #2454、worker応答ID/版はDraft #2471、同一Query IDのSQL原byte変化はQX-14の未解決課題。

**次のM1判定方法（未実行）**：既存Queryパッケージの合成`analyzer.Result`から`DeriveFromAnalysis`→`RelationalIR`を**実際のGo呼出しで生成した結果**を、今回のQX-17プローブへ入力。3例の「Scope→Source occurrence→Operation Inputs→OutputFields」を別々に照合し、(a)同じ依存を辿れる、(b)構造は辿れるが出力Field接合は無根拠、(c)未表現／`opaque`、(d)誤った`Full`の4群へ分類する。現在のIRで説明の成立条件が表せない場合は、まず具体的な不足する項目を記録し、第二のParser/SQL Evaluator/新Core Provenance正本を作る前に既存Owner #2387/#988/#2506で接続を評価する。

**同時に満たすべき横断要件**：#2506のData Context Instance/選択epochと#2500のAction対象IDは、本QX-17 Source operation IDと別のID。部分取得・JOINの行増幅・集約の操作不可をModeler/SHO/外部Consumerへ説明する際は、構造観測→対象束縛→現在性→Authority/Effectをそれぞれ独立に扱う。QX-17はread-only構造候補のStageに留め、業務完成・Commit・GenerationのPASSへ昇格しない。


### 7.10 QX-17 M1：実Go `DeriveFromAnalysis`への最小縦切り（2026-11-02、別Draftで実装）

**他チャットとの接続**：AI-first/Headless-first（#2492のDraft）、Function Data ContextのDefinition/Instance/Result Slice・各集合の取得Scope（#2506）、表示と更新逆写像を分離する#2497/#2500、異なるConsumerで同じSource/OutputField/Grainを保持するSHO/Modeler/外部Consumer（#1803/#2516）、宣言Sourceと実消費・Build/Worker証拠の帰属（#1103/#2514）を横断。この節の当面のOwnerは**Queryの構造導出 #2387**であり、Function ContextやMorphism、実Host/Provider側の契約を二重化しない。

**Go実行候補の配置**：現main `98e08841ccd03574690fc6195e2cce62454a5fa3`から新枝を作り、[Draft PR #2577](https://github.com/hat47x/tei-platform/pull/2577)に`internal/query/derived_qx17_characterization_test.go`を追加。旧Draft #2388の資料・固定fixtureをmainへrebaseして混ぜず、**このPRはGoテスト1ファイルだけ**。Go実装の`DeriveFromAnalysis`を、3種類の手書き合成`analyzer.Result`から直接呼ぶ。

| 合成Resultの演算形状 | Goから確認したい構造 | 意味・実行の未確認 |
| --- | --- | --- |
| `Source(line)→Aggregate(line)`, `Source(delivery)→Aggregate(delivery)`→JOIN | line Aggregateはline Source occurrenceだけ、最終JOINは双方 | 集約SQLの正確性・JOINのCardinality・データ集合の全件性 |
| `Source(line)`, `Source(delivery)`→JOIN→Aggregate | 最終Aggregateはline・deliveryの両Source occurrenceへ参照を持つ | 値だけのLineageで行多重度を説明したとみなさない。SUMの業務総額は別 |
| `Source(header)`, `Source(delivery)`→LEFT JOIN→Filter | 最終Filterの入力が双方へ連なる | `d.id IS NULL`での不存在確定には右側の必要全集合を確認。Expression文字列だけでNULL/不在法則を受理しない |

このテストでは、`Lowering=full`が得られても`Snapshot=nil`により物理Schemaの`Resolution=none`となることを**別軸で観測**する。`OutputField.Inputs`は列Occurrence/Field ID由来、`RelNode.Inputs`は演算IDであり、**Field→Operatorの型付き関連付けが実装済みと推定しない**。JSON marshal/unmarshalの往復はデータ構造保持の試験であって、ModelerやSHOの設計書が業務意味を理解した証拠ではない。

**現在の限定確認**：Go 1.23.2のローカル`gofmt -l`でフォーマット差分なし、GitHub保存した同テストのblob `7170f6e2eeb1d6b90b121474ba8e1b8b27a0d44d`がローカル実行予定のテスト内容と一致した。リポジトリ指定Go 1.26.10による実テストはGitHub Actions起動を確認し、**結果待ち**（完了結果を確認してから本節へ追記）。従来のQX-17のPython14件・構造の4点照合は別工程の証拠である。

**主な反証点・次段の観測**：(i)実Go結果と手書きIRでNode ID・入力依存が一致するか、(ii)同一`OutputField.Inputs`のまま演算順を変えると最終依存が変わるか、(iii)Analyzerの`Lowering=full`と実構造の参照整合、Source byte/Schema/Worker正統性、Output Field bindingの欠落がどこで検査されるか。(iii)は**QX-14b/c未解決**として、同一Query IDの再利用で旧SQL解析結果を意味受理へ持ち込まない。業務Entity Key/Path/Owner受理、Functionのselection epoch、Provider snapshot、Go/Java/.NET別TargetおよびHost実行権限も本テストでは証明できない。

**Go本体/他Consumerへの接続順**：#2577の固定head Go試験→#2387のread-only `RelationalIR`構造照合→#2506へのCollection/Result Sliceの**不足情報の受け渡し**→CLI/JSONとSHO/Modelerの同一説明参照。QX-16の手書き片側Claimの数学的法則は#1063、Independent Oracleは#1131/#2514が所有し、Query結果への自動的なAction適用は#2500/#1125/#1174が所有する。分岐を先回りして結合せず、必要な形だけを段階実証する。


### 7.11 QX-17 M2：値の由来と「行が現れる条件」を別に扱う（2026-11-02、構造反証計画）

**横断する設計課題**：#2387のQX-17実Go `RelationalIR`構造、#2506のResult Slice/Grain、#2582の反応的Invalidation、#2586のQuery→Context→Claim縦断を結ぶと、同じ「列の依存」の語で少なくとも次の4種を混同しやすい。**Value dependence**（値の計算に使った元列）、**qualification/existence dependence**（WHERE/EXISTS/JOIN等、その出力行・groupが成立する条件）、**multiplicity dependence**（bagとして何回出現するか）、**ordering/selection dependence**（ORDER/LIMIT/Windowによりどの行が選ばれるか）。さらに、Contextでいう業務Entity identityとQueryのrow occurrenceは別である。既存`OutputField.Inputs`は主にColumnOccurrence由来の値のlineage、`RelNode.Inputs`はOperation ID由来の構造経路で、**この4依存の証明書そのものではない**。

**固定反例の対照**（既存F0/QX-15の合成Entity/集合を再利用する。SQL原文・独立期待・結果Grain・完全性は#1131/#2506と照合）：

| ケース | 出力として同じ列/値が見える状況 | 結果の成立・多重度・意味の差 | M2で必要な診断 |
| --- | --- | --- | --- |
| QX-17M2-A | `SELECT l.amount FROM line l` と `SELECT l.amount FROM line l JOIN delivery d ON ...` の`l.amount` | 値の元列は同じでも、INNER JOINはdeliveryの存在で行を選別し、複数deliveryがあると同じline値を増殖する | Field lineageがlineのみでも、root結果のexistence/multiplicityはdeliveryに依存。JOIN入力がunknownならoutputの全件性は未確定 |
| QX-17M2-B | `SELECT h.id FROM header h LEFT JOIN delivery d ON ...` | `h.id`自体の値はheader由来。LEFT JOINは未一致でも行を残すが、右集合の追加で`(H,NULL)`が置換され、出力出現数が変わる | 「値のSource」と「出力行の出現・由来・grain」を切り分ける。未一致NULLを実体NULL・不存在証明へ昇格しない |
| QX-17M2-C | `LEFT JOIN ... WHERE d.id IS NULL` | 同じheader値でも、deliveryに一致行が追加されると以前のheader出力が消える | Filterの真理条件が文字列表現だけならanti-join/不存在の意味は`unverified`。右集合partialから「なし」を確定しない |
| QX-17M2-D | `SELECT l.amount`の集合から`SUM(l.amount)`を計算、JOIN前後で順序を変更 | Group単位で値の元列が同じでもJOIN後のSUMはfan-outして変わる。lineを事前集約した値も、後段JOINで出現が増える | Claimの対象が「line全集合に対するSUM」か「JOIN後bagのSUM」かを区別。Column `Inputs`だけで演算上の値依存を完全証明しない |
| QX-17M2-E | `ORDER BY d.created_at LIMIT 1`で`l.amount`を投影 | 元列はlineでも、結果に選ばれる行はdeliveryと順位条件に依存。入力追加で先頭行が替わる | order/tie-break、必要な母集団、同一Snapshotが不明なら「最初/唯一」の業務主張を保留 |

**ここで新たに置く判定原則**：
1. `Relational.Root`から到達するSource occurrence集合は、**出力Query全体の構造的な依存候補**として扱う。特定`OutputField`の意味・寄与行・業務Action対象・部分取得でも成立するClaimを、その到達集合から自動決定しない。到達しないSourceが「影響なし」であるとの強い主張には、Operation ID一意性、欠落/循環/opaque/相関/CTE境界、Source版の照合が必要。
2. **保守的な上界**としてroot到達Source集合を示せても、各出力Fieldの厳密な最小依存集合とは限らない。`OutputField.Inputs`とOperation DAGのIDが異なる点を保持し、対応が未実証なら`field-to-operation-binding: unverified`とする。全root依存に基づく再取得は過剰失効となり得るが、無根拠に局所維持するfalse-positiveより優先する。#2582は独立Claimの依存が証明できた場合に限り局所維持を試す。
3. 解析の結果が`Parse/Lowering=full`であっても、Value/Existence/Multiplicity/Selectionの4種類のClaim資格は**別欄**。SQL式の型付き意味、Source revision/dialect/schema、必要Collection完全性、Snapshot、認可/Disclosureを満たさないClaimは`not-established`。SQL `NULL`は`sql-null`と分離する。正式な公開Enum/API追加はまだ行わない。
4. 新たな万能SQL EvaluatorやQuery Plannerを作る代わりに、まず#2577の実Go `DerivedIR`、#2388のPython QX-17プローブ、#2506のContext/Result Slice合成参照を**同一固定入力**で比較し、各行の「確定できる構造／型付き根拠の欠落／誤って確定しそうなClaim」を記録する。受入テストは独立Oracleを#1131/#2514に接続し、手書きOperator対応を実IRの自動Binding成功と呼ばない。

**今回新たに確認したCI証拠（QX-17 M1の限定前進）**：Draft [PR #2577](https://github.com/hat47x/tei-platform/pull/2577) のhead `31a2fc2d6ddd0fe4c343c00df39ccfdb65723a23` を含むGitHub Actionsの**PR merge checkout `f01f4322b3287e277b31c59b02a0f7eda33bc6aa`**で、Ubuntu runner/Go **1.26.10**、`go test -count=1 -timeout 60m ./...`と`go vet ./...`が成功し、`tei/internal/query` は `ok` と記録されている（workflow run `41388183698`、job `125093564871`、2026-11-02 UTC）。このログは`TestQX17`を**名前指定で個別実行した記録ではない**。PR merge refによる関連パッケージの回帰PASSと、PR head単独/実SQL worker/SQLの意味/Provider/SHO/Owner受理を切り分ける。元§7.10の「CI結果待ち」は、本節の観測後には**完了済み**だが、PR本文・Core・製品契約の採用を意味しない。

**M2最小完了条件（提案）**：A〜Eを少なくとも「構造追跡と式意味の別資格」で比較し、Aの6出現と2業務Entity、Cの入力追加によるretraction、Dの200/600/100、Eの選択変更を独立期待で検査。QX-17実Go→read-only probe→#2506の取得状態という一往復で、説明に必要な情報が欠落した箇所を具体的に報告できること。#2586のHJ-01〜04はこの受渡しの統合Owner、#2582が変更後の局所維持／失効、#2500が更新Intent、#1803/#2516が説明文書、#1174が実効果を所有する。**本節は計画のみ。A〜EのGo/Worker/Context統合試験はNOT RUN**。


### 7.11.1 QX-17 M2のexact SQLite対照を固定し、Measure/Contextへ接続（2026-11-02、局所6件実測）

QX-17 M2の**計画A〜Eに実測の限定証拠を加え**、他チャットで整備された[Measure #2593](https://github.com/hat47x/tei-platform/issues/2593)の非加法・平均再集計の論点を新ケースFとして接合した。既存SQL/TEIに全体的な変更を入れる前に、同じ値由来でも演算DAGにより結果行・bag・Measureがどう変わるかを、独立SQLiteで再現する。

- 新規単一fixture：[test_qx17m2_sqlite.py](https://github.com/hat47x/tei-platform/blob/work/query-structural-fixture-design-20261030/plan/fixtures/query-workbench-f0/test_qx17m2_sqlite.py)、Git blob `972720f96e90504b947cdd2e81ec04791d929ea4`。LinuxのPython **3.13.5** / SQLite **3.46.1**、`python3 -m unittest -v test_qx17m2_sqlite.py`で**6/6 PASS**。GitHub上のexact blobとローカル実行バイトのSHAが一致。合成入力・手書き期待による実測であり、TEI本体・SQLGlot・Provider・Contextは実行していない。
- このファイルは**既存QX-16/17 fixtureフォルダの補助検証**。既存Scenario/Datasetの第二正本や別SQL Evaluatorを導入しない。

| 局所テスト | SQLite合成実測と独立Expectedの一致 | 設計へ返す判別条件 |
| --- | --- | --- |
| M2-A 値とJOIN row | line単独2行（100×2）、delivery 3行とのJOINでは6行、右0行ならJOIN結果0行 | 同じ`l.amount`でもdeliveryの存在・多重度が結果bagを決定 |
| M2-B LEFT未一致 | `(H,NULL)` →右D1追加後`(H,D1)`。旧仮行は結果に残らない | 部分集合への後続追加はappendだけとは限らず、`retract/replace`を必要とする |
| M2-C 反結合 | `WHERE d.id IS NULL`でHが1行→右一致D1追加後0行 | 局所未観測の右集合から業務不存在を確定しない |
| M2-D SUMと再集計 | line合計**200**、結合後SUM**600**、SUM(DISTINCT value)**100**。**先にGROUP BYして200へ縮約した行をその後delivery3件へJOINし、もう一度SUMすると600** | 先集約した事実だけでは、その後の結合と再集計が意味保存するとは限らない。Measureの集計軸・対象Grain・中間状態を#2593で確認 |
| M2-E 順位選択 | delivery順位の追加で`ORDER BY rank,l.id LIMIT 1`の結果は`(L2,200)`→`(L1,100)` | 出力値の元列がlineでも、選択された行はdeliveryの順位と対象母集団に依存 |
| M2-F AVG重み | line単独AVG=**150**（100,200）、delivery側の3:1重みのJOIN後AVG=**125** | 同じ値のField Lineageでも、JOIN後のMeasureは異なる。平均を別軸で再集計・比較する際は分母/入力出現数/対象Grainが必要 |

**この差分から得るアーキテクチャ上の結論候補**：

1. *値の出典を示す線*と、*結果の成立・多重度・順位を支える演算上の依存*をModeler/SHO/AI向け説明で区別する。Aggregate後の値にSourceの正しいLineageがあっても、**再集計可能性（#2593）は別の適合条件**。JOIN前集約は一般のSQLで後続集約の安全性を保証する処方ではない。
2. #2582のReactive更新は、M2-B/C/Eで`insert-only`更新の危険を実測反証。M2-D/Fでは、**変更時に再評価が必要なMeasure Claim**をResult SliceのContext Epoch/対象Grain/必要集合へ結ぶ。業務上の保存値とQuery由来の表示計算値は別。
3. #2586 HJ-02〜04では**Query全体のSource graph reachability**と、**個別ClaimのMeasure/対象集合への適用資格**を別の証拠欄とする。SQL上の有効な「集約値」と、業務上受理した「測定値の意味」は別。QX-18の検索Path/RLSによる入力母集団の違いも、そのClaim資格に独立に作用する。
4. 公開API追加や万能依存解析を先決せず、現在の`OutputField.Inputs`（ColumnOccurrence/Field）と`RelNode.Inputs`（Operation）の間で**欠落している根拠**を同一fixtureで表示し、未知のClaimを`not-established`に留める。Provider全件・Targetの実効物理名解決・RLS、業務Identity・Owner受理はSQLite単体実測の範囲外。

**次段の実証Gate**：本ファイルのSQLite6件PASSとは別に、(i) 実Go `DeriveFromAnalysis`の合成IRからA〜FのOperation source occurrenceとOutputFieldを抽出、(ii) 欠落しているField→Operation対応を`unverified`としてHeadless JSONへ運ぶ、(iii) #2506のD1完全集合/D2部分集合を注入し、#2593のMeasure適合、#2582の変更影響、#2586 HJ-01〜04のClaim別結果を比較する。SQL worker実物→Go、実Provider/RLS、SHO/Modeler、Host Authority/Owner Meaningは**NOT RUN**。Go CIの成功（§7.11）やSQLiteの値一致から意味受理を推測しない。


### 7.12 QX-18：SQL解析結果の実効名前解決環境と可視入力集合（2026-11-02、Target差分・未実行）

[#2387の一次反例・静的根拠](https://github.com/hat47x/tei-platform/issues/2387#issuecomment-6572048044)を本DraftのM0→M2へ反映する。QX-13〜14の「lexical scopeと正しいSource/解析Inputの帰属」、QX-15〜17の「結果Grain・Value/Existence/Multiplicity/Selectionの別資格」が成立しても、**Target DBセッションの名前解決環境と可視行集合**が変われば、同じSQL Source byte・Query ID・Schema定義で異なる結果になる。この二条件は同一の資格ではない。

**一次外部根拠**：PostgreSQL公式の[Schema search path](https://www.postgresql.org/docs/current/ddl-schemas.html)・[search_path/一時Schema](https://www.postgresql.org/docs/current/runtime-config-client.html)・[Row Security Policy](https://www.postgresql.org/docs/current/ddl-rowsecurity.html)・[PREPARE](https://www.postgresql.org/docs/current/sql-prepare.html)。これらは外部DBの仕様であり、TEIでの実PG通し動作を示した証拠ではない。

**現mainの静的境界**：`internal/query/analyzer/model.go`のRequestはQuery ID/Source/Dialect/SchemaContext（Table名と列名）を持つが、`search_path`やsession-local object、principal/RLS policyの**実効条件は現行wireにない**。SourceRef/FieldRefやCoverageに足りない条件は、推測したschema-resolvedではなく未確認として説明する。#2465による別Query IDの拒否は統合済みだが、同一Query IDの別SQL Source・実効Schema探索を結び付けたこととは別。#2454/#2471のDraftを置換しない。

| Case | Source/DB条件 | 変える入力 | 手書き期待（実PG NOT RUN） | 不合格表示 |
| --- | --- | --- | --- | --- |
| QX-18a / Path shadow | `tenant_a.orders`=100、`tenant_b.orders`=900が同じSchema Snapshotに存在。SQLは`SELECT SUM(amount) FROM orders`固定 | `SET search_path TO tenant_a,public` vs `tenant_b,public` | 参照実表とSUMが100/900へ変わる | SQL SHA＋Catalog一致だけで同一Sourceと確定 |
| QX-18b / Temp shadow | `public.orders`とsession-local `pg_temp.orders` | 当該接続のtemp schemaの存在・探索順 | 無修飾`orders`はセッションに応じて別Relationを指し得る | 静的catalogだけで実テーブルを固定 |
| QX-18c / RLS visible subset | `public.orders`にAlice:100/Bob:900、非ownerのselect可能なロールと`USING(owner=current_user)` policy | Role Alice vs Bob（SQLは`public.orders`明示・同一Snapshot） | 実テーブルは同じ、visible SUMは100/900、DB全体の1000とは異なる | 権限下の0件/合計を全業務母集団の事実と断定 |
| QX-18d / Explicit target | SQLは`SELECT SUM(amount) FROM tenant_a.orders` | search_pathだけ変更 | Relation束縛は安定。ただしRLS・Snapshot・時刻等は別条件 | schema-qualifiedだけで全件/現在性/認可を保証 |

**M0（Query/Target対応）**：同じQuery ID・SQL byte・Dialect/Schemaの反例a/bを、Targetの実効名前解決条件を固定した場合と未知の場合で比較する。Source occurrence名／意味ModelのEntity ID／実際に束縛した物理Relationの三者を別に扱う。匿名または未知のsearch_pathでは、候補解決と確定解決を区別する。Target固有条件をTEI Coreへ全部取り込む決定は先行せず、既存Plugin/ProviderとExternal Hostの観測可能な条件をinspectする。実効search_pathは`SHOW search_path`と必要に応じて展開結果・実参照先まで、同じConnection/試行に対応付ける。Prepared statementの場合はDB側の再解析条件も対象とする。

**M1（read-only Explain）**：a/bは同じSQLを描いても元の物理TableへのLineage矢印が変わる、cは物理Tableへの矢印が同じでも**visible input population**が変わるという対照。前者は`resolution context unknown`、後者は`visible scope conditional`等の表示候補とし、公開Enum/APIは未決。未知なら`Coverage.Parse=full`のままでも実効物理対応・Claimは未検証とする。Scope/Result/Grainを表示する際、権限・Tenant等の機微情報の表示/開示判断は#1693に委譲する。

**M2（Context/Claim）**：#2586 HJ-01へsource＋実効Relation bindingの一致、HJ-03へContext Result Slice/Scope/epochと実効可視母集団、HJ-04へ`EXISTS`/0件/NOT EXISTS/集計の必要な取得範囲を引き渡す。#2593で同じSUM/AVGの演算が一致しても、Visible SliceとTimeが異なる値を同一Measure観測としない。#2582の再取得/失効と#1803/#2516の資料投影は必要な範囲だけ参照する。外部Hostの認可・Effect(#1174)は別資格として維持。

**次の証拠欄（本書追加時点すべてNOT RUN）**：①外部PostgreSQLの公式仕様、②TEI現在の静的コード、③隔離PG a〜dの期待/実測、④SQL worker実体→Result→Go IR、⑤Headless Query→Context→Claim、⑥GUI/SHO/Host／Owner受理、を独立に記録。実PG試験の前にSQLを文字列置換してSourceを確定したことにしない。正式Dataset/Scenarioは#1131/#1180、Oracleは#2514を使用し、#1119既存G0〜G4へBlocking条件を追加しない。


### 7.13 QX-19：View/FunctionのDefiner・Invokerと、Queryから見えない実効依存（2026-11-02、提案・未実行）

[#2387のQX-19原票](https://github.com/hat47x/tei-platform/issues/2387#issuecomment-6572368011)を既存QX-18の後続として接続する。**実テーブルのsearch_pathや直接RLSだけでは説明が閉じない**。同じSQL byteと同じ `reporting.orders_v` でも、Viewの `security_invoker`・所有者・RLS Policy、内部SQL Functionの `SECURITY DEFINER/INVOKER`、権限/Policy版が変われば、可視入力集合と結果Claimは変わる。PostgreSQL [CREATE VIEW](https://www.postgresql.org/docs/current/sql-createview.html)、[Row Security](https://www.postgresql.org/docs/current/ddl-rowsecurity.html)、[CREATE FUNCTION](https://www.postgresql.org/docs/current/sql-createfunction.html)が外部仕様の根拠。これらをTEIで実証済みとの主張ではない。

**既存の実証との非重複**：[#1769](https://github.com/hat47x/tei-platform/issues/1769)・[PR #1791](https://github.com/hat47x/tei-platform/pull/1791)が扱うのは、TEIの明示Tenant SQL Scopeを実Providerへ押し下げた場合のFilter/Sort/Top・列の投影。**DBによるRLS、View owner/invoker、Function definerの実証とは別**。同じ`Tenant`という語で証拠を流用しない。Query構造#2387、データ取得#2506、開示#1693、実行権限/Effect #1174/#1129が既存Owner。

| Case | 隔離DBの入力・変更 | 独立Expected（実PG未実行） | 説明時に保持する不確定性 |
| --- | --- | --- | --- |
| QX-19a default View | app_owner所有のbase ordersにalice:100、bob:900。RLS ENABLE、policyは`owner_name=current_user`。Viewもowner所有、既定security_invoker=false、非owner role alice/bobで同じ`SELECT SUM(amount) FROM reporting.orders_v` | owner通常RLSをbypassする前提なら各呼出で1000 | 見えた全額を呼出者のbase table read grantに帰属しない。Viewの開示許可は独立 |
| QX-19b invoker View | 同名Viewを`security_invoker=true`へ改版、同SQL。利用者へのbase SELECT grantとRLS条件を確保 | alice=100、bob=900 | SQL SHA同じでもeffective inputの版が異なる |
| QX-19c nested View | 外側は既定View、内側はinvoker View | 内側基礎関係への権限/RLSはinvokerで評価され得る | どの層も無条件に外側View owner主体へまとめない |
| QX-19d SQL Function | `SELECT reporting.total_amount()`。Function内部がbase ordersをSUMし、SECURITY INVOKER→DEFINERに改版 | invokerなら100/900、base table owner definerなら1000になり得る | Function内部Lineage/volatile/side effectの推測展開をしない |
| QX-19e policy・owner改版 | SQL、外側View/Function名、base列定義は固定。Policy/owner/FORCE RLSを変える | 以前の値が現在の表示・Claimに適格とは限らない | Source SQL revisionだけをfreshnessの根拠にしない |

**M0実装/試験**：まず既存Analyzerの `SourceKind(table/cte/subquery)` と `Operation.Expressions` の表現範囲を、合成Result・既存Go変換・実SQL Workerの証拠列で分ける。外側View/Function名の参照を内部表のSource occurrenceと自動的に同一視しない。Catalog定義に正当にアクセスできる場合のみ内部定義へのread-only説明を候補とし、定義/権限が取得不可なら`opaque`または未検証へ留める。静的Explainのために任意SQL Functionを**実行しない**。関数の`VOLATILE`や`SECURITY DEFINER`が想定された場合も、既存Hostの実行認可・推移的Effectを省略しない。

**M1/M2のContract判別**：外側SQL Source ref／内部View定義の版／実効権限主体／RLS・Policy版／Result Sliceの可視集合／Claimの必要母集団を**別資格で受渡し**。表示可能なLineageは許可されたSource境界までに制限する。#2586 HJ-01〜04、#2582の改版による条件付き失効、#2593 Measureの合計の意味、#1693 derived aggregateの開示資格、#1803/#2516の文書投影で必要に応じて使用する。正式公開の契約追加・Plugin-ABIの変更は実証後の判断。

**試験環境/合否**：専用の隔離PostgreSQLでrole・grant・base-owner・RLS PolicyとView Option/Function属性を正確に固定する。a〜dは同じfixture値と外側SQLを条件ごとに比較、a→bは明示的なView definition revisionを計測。Postgres catalogのView Option/Policy/owner、実際のRole/Session、出力1000/100/900を観測し、手書きExpectedとの差と不一致原因を記録する。通常のデータアクセス権確認と**Viewによる集約結果の開示許可**は別評価。PGがない環境・Worker不在・権限不足では該当工程を`NOT RUN`とし、合成静的fixtureやSQLの単純模擬でPASSへ代用しない。

**採用状態**：この§7.13は検証計画。実PostgreSQL/TEI Worker/Go変更/正式Function Context/GUI/SHO/Host AuthorityのQX-19試験は未実施。既存#2388 F0/QX-17、Go Draft #2577、Source Scope Draft #2454、worker応答Draft #2471、D2拒否、#1119 G0〜G4は変更しない。


### 7.14 QX-20：Explain取得の実行効果と公開される観測の二つの安全境界（2026-11-02／局所fixture実測）

**一次Owner**：[#2387のQX-EX1〜06](https://github.com/hat47x/tei-platform/issues/2387#issuecomment-6572575006)を参照。この節は前節QX-18/19の実効Target・View/Function依存に、**説明取得の操作自体の性質**と、**説明を消費者へ渡す際に観測可能な情報**を付け加える。実SQL ExecuteのHost・Provider権限、Result Slice、Consumer別Disclosureは引き続き別Ownerとし、第二のSQL Executor・新Host・新認可基盤を増やさない。

#### QX-20a：取得モードの違いを契約に明示

| 要求した説明 | 実行対象と実測の有無 | 効果と開示の扱い |
| --- | --- | --- |
| TEI静的Query構造（既存DerivedIR/RelationalIR） | 与えたSQL、Schema Model、解析結果の読取り／図表化。**対象DBを実行していない** | Source文・構造・定数・診断が機密情報であり得る。解析器に実SQL実行許可を付与しない。DB物理Plan/可視行・実測時間は未観測 |
| DBの通常`EXPLAIN` | DBの計画器に問い合わせる。通常は対象Queryを実行せず計画を表示 | 外部DB操作・接続・権限・内部関数やlock/リソース・Plan情報開示の条件をTarget別に審査。名前に`EXPLAIN`が含まれるだけで全般の副作用なしを保証しない |
| DBの`EXPLAIN ANALYZE` | PostgreSQLでは**対象SQLを実行**して計測する | INSERT/UPDATE等の副作用、SELECT中の副作用関数、外部呼出し、lock、資源消費を実行境界で扱う。`BEGIN; ...; ROLLBACK;`だけでは`nextval/setval`によるSequence更新等の残存Effectや外部作用の不存在を保証しない |

外部仕様：PostgreSQL [EXPLAIN](https://www.postgresql.org/docs/current/sql-explain.html)、[Using EXPLAIN](https://www.postgresql.org/docs/current/using-explain.html)、[Sequence関数](https://www.postgresql.org/docs/current/functions-sequence.html)。`EXPLAIN ANALYZE`はSELECT出力行をClientへ返さなくてもQueryの実処理が行われる。`EXPLAIN`実行を含むDBアクセスは既存Host/Providerの権限に従い、静的ExplainのCLI/SHO利用から自動許可されない。実測時間やRowsを、計画値や手書き期待から生成して`observed`と記録しない。

#### QX-20b：秘密の行値を載せなくてもExplainを通じて情報を漏らし得る

**既存型の静的確認**：`internal/query/derived.go`の`SourceRef.Name`、`FieldRef.PhysicalSource`、`OutputField.Expression`、`Lineage`、`internal/query/relational.go`の`RelNode.Expressions`、Analyzerの`Diagnostics`がある。これらは開示可能性を自動判断するフィールドではない。DB物理PlanもRelation名・推定件数・式等を含み得るため、`row valuesを除いたJSON`を安全な公開Projectionと見なさない。

**保存した独立合成テスト**：[test_qx20_explain_observation.py](https://github.com/hat47x/tei-platform/blob/work/query-structural-fixture-design-20261030/plan/fixtures/query-workbench-f0/test_qx20_explain_observation.py) は、同じ公開入力 `public.orders` を持ちながら、非公開のRelation名・Filter定数・推定件数だけが異なる二つの**人工世界**を作る。Python標準ライブラリだけの`unittest` 10件を実行し、**10/10 PASS**。実行バイトとGitHub保存blobは SHA `18302f39d3366665681553424815f3ea4aa61f5a` で一致。これは**人工JSONの観測同値・差分検出試験**であり、TEIの実IR→Consumer公開処理や開示認可が実装済み／PASSという証拠ではない。

| 比較項目 | 局所テストで検出・確認したこと | 製品側で別途必要な根拠 |
| --- | --- | --- |
| 明示的に許されたSource IDからの合成Projection | 非公開世界が変わっても、投影後のJSONが一致 | Source grantの真正性／実Consumerの開示権限（既存#1178/#2516） |
| 生IR／秘密のExpression／診断文 | 公開される文字列が変わることを検出 | Source本文・内部参照・Error・TraceをConsumerごとに公開できるか |
| 非公開Source数のplaceholder／Plan Rows／Status／配列順 | **値をマスクしても**要素数・推定件数・診断Status・配列順の差が観測可能 | 開示可否が未定のときは「非公開の数」や内訳からも推論されない出力設計 |
| 明示許可した一つの集計値だけが異なる二世界 | 集計値の差だけを`authorized release`として比較から除くと、残る差の有無を検査可能。別のdebug差があれば拒否 | **releaseの許可そのもの**はテスト関数の引数で仮定しており、Policy/Host/Owner/Consumer別の認可判定は未実装 |

**重要な限界**：この観測同値チェックは固定した一組の二世界とJSON文字列に関する**必要条件の検査手法**である。すべての秘密状態・タイミング・例外・通信/ファイル送出・反復照会に対する非干渉の数学的証明ではない。また、テスト内`granted_source_ids`と`released_keys`は**あらかじめ許可済みと仮定した入力**であり、名称が`public.`だから自動で公開許可されるという意味ではない。許可の根拠は既存のHost/Authority/Disclosureに委ねる。

#### QX-20の段階別受入・停止線

- **M0／source inspect**：現在の静的IRに含まれるSource/Expression/診断がConsumerから見える実出力Pathを棚卸しする。実際のTEI ConsumerでどのJSON・ログが外部へ送られるか確認できなければ`unverified`。新しいCore Provenance/Disclosure engineやグローバルな`safe`フラグを作らない。
- **M1／説明取得の資格**：同じSQLに対し`structural` / `planner` / `execute-and-measure`を出所・実行済み/未実行・権限・取得範囲の異なる要求として検証。前2者も相応のProvider/Schema/Info権限が必要。危険な実行モードを単なる解析の上位オプションとして自動有効化しない。必要な実PG試験は隔離DB・接続・Resource/Effect観測・Owner許可の条件を固定する。
- **M2／Consumer投影**：#2516の観測同値・Consumer用途別開示に本局所oracleを提供。#1178でのMetadata/Trace/Diagnostic開示判定、#1693での許可された集計のみの公開、#2586のQuery→Context→Claimで必要なScope、SHO #1803の固定版資料に引渡す。ただしAI用に必要なLineageを全削除して意味情報が失われた場合は`unsupported/withheld`とし、「説明できた」と主張しない。
- **M3／実出力E2E**：保存済みPython fixtureではなく、実際のTEI static IRから（許可された）CLI/SHO/外部AI Consumerへ投影したJSONを対象とし、秘密部分だけを変えた二世界差分を比較する。Logging/例外/ツール戻り値/ファイル成果物の観測も別ゲート。外部AIのOSファイル権限・egress隔離は#2017/#2515がOwner。
- **採用判定**：本節追記時点でPostgreSQL Plan/ANALYZE、実TEI IR/Worker、Provider、OS egress、SHO・外部AIへの実送信、実Host Authority、Owner意味受理は**NOT RUN**。既存#1640 D2のCTE/JOIN実行拒否と#1119既存G0〜G4の優先順位・Blocking条件を変更しない。


### 7.15 QX-21：同じ説明・表示値を再利用するための、用途別「根拠の再確認」（2026-11-02、局所15件実測）

#### 横断的な設計判断

QX-14〜20はSQLの構文・解析入力/版・演算構造と、DBの実効名前解決・RLS/View/Function・説明取得時のEffect/情報開示が**別々の根拠**を必要とすることを示している。さらに[#2506](https://github.com/hat47x/tei-platform/issues/2506)のFunction Result Slice、[#2593](https://github.com/hat47x/tei-platform/issues/2593)のMeasure/再集計、[#2582](https://github.com/hat47x/tei-platform/issues/2582)の結果失効、[#2597](https://github.com/hat47x/tei-platform/issues/2597)の未保存Patchと編集セッション、[#2516](https://github.com/hat47x/tei-platform/issues/2516)のSHO/AI/外部Consumer別Projectionを接合する。

**提案する共通原則**：Queryの説明や値の再利用は、「SQL byteが同じ」「Query IDが同じ」「以前の数値表示が同じ」という単一の同値性では決められない。一方、任意の変更ですべてのClaimと編集状態を必ず破棄するのも誤り。**既存の個別Ownerが持つ根拠を、Consumerが必要とする資格ごとにread-onlyで照合する**。この比較票は新しい永続SSOT、実行許可、全Scope共通の版Registryや再計算Engineではない。

| 説明・利用資格の系統 | 根拠を確認する軸（例） | 特に区別する反例 | 所有/参照 |
| --- | --- | --- | --- |
| 解析の実入力 | Query ID、SQL exact byte/digest、Dialect、Schema、Analyzer request/responseと実消費物 | 同一Query IDの別SQL／Source検査後に別byteを実消費／別Attemptの解析結果 | #2387/#2471/#2595 |
| DBの実効対象 | 実Connection/Sessionのsearch_path、temp relation、View/Function定義と有効な実行主体 | 同じSQLで別Tenant物理表を読む、同じView名でdefiner/invoker変更 | #2387 QX-18/19、既存Provider |
| 可視集合と演算 | RLS/Policy/Principal、Provider Snapshot/可視Scope、RelNodeとOutputField、bag/Group/Sort | RLS下の0件を業務母集団の不存在へ昇格、列LineageだけでTOP-1依存を確定 | #1693/#2506/#2387 |
| Measure/Claim | 元FactのGrain/Identity、SUM/AVGの集計状態、適用軸、NULL/欠損/未確定、丸め時点、意味Owner | 先集約200を後続JOIN/再SUMすると600、同じDecimal値でround-before/after差 | #2593/#1063/#2497 |
| Contextの配置と現在表示 | Function Instance、親Anchor、選択epoch、Result Sliceと取得時点 | A→B→Aの古い応答混入、同一Definition二Instanceの誤共有 | #2506/#2582 |
| 編集の対象帰属 | Dirty Patchの元Subject/Relation occurrence、baseVersion、Instance/Parent Anchor | Bへ再選択した後もAに帰属する未保存Patchを保持。**Bへ付替えたり無音破棄しない** | #2597/#2500/#2598 |
| Disclosure/Projection | 現在のPurpose/Consumer、Row/Field/Metadata/Trace/Query Planを公開するGrant・出典境界 | RLSに従って計算可能な値を無条件に外部AI/SHOへ開示、hidden sort由来の推論漏洩 | #1693/#1178/#2516/#1656 |
| 表示・文書の版 | Renderer/SHO Source Reference・保存文書/局所編集と有効なモデル版 | 見た目の変更を業務事実の変更、資料版の一致を現在の実行許可へ同一視 | #1803/#2516 |
| Operationと実効果 | Intent→実Requestの対象/時点再束縛、Host Grant、Provider commit、receipt/再読込の因果一致 | 同じ数値の再表示を当該Submitのcommit成功と断定 | #2598/#1174 |

**時点の明確化**：業務規則の有効時点、データのProvider Snapshot、Query/Explain実行Attempt、Consumerの表示・開示時点、Contextの選択epochは別の時計/世代。Query構造の意味の再利用と、DB Plan/実行時間/行数の再利用も別（[#2387 EX-07〜11](https://github.com/hat47x/tei-platform/issues/2387#issuecomment-6572850875)）。「説明時のsearch_pathで後日の実行も同じ表を読んだ」とは推測しない。最終的な実行とEffectは既存外部Host/Providerが受け持つ。

#### 局所Reference（実測）／未確認を変更なしにしない

[既存fixture内の test_qx21_evidence_axes.py](https://github.com/hat47x/tei-platform/blob/work/query-structural-fixture-design-20261030/plan/fixtures/query-workbench-f0/test_qx21_evidence_axes.py)を追加し、**Python 3.13.5／標準ライブラリ unittest 15/15 PASS**。GitHub exact blobとローカル実行byteのGit hash一致：`826ddb03242340f5415b33bab44e356a42063a0b`。実行：`cd plan/fixtures/query-workbench-f0 && python3 -m unittest -v test_qx21_evidence_axes.py`。

本Referenceは**19の人工ラベルの比較だけ**を行う。各根拠を実Source/Provider/Hostから取得・証明する機能も、Business Claimを計算する機能も持たない。比較結果の`claim/disclosure/presentation/editing`は*再確認先の候補*であって、正式なTEI enum／認可・失効・使用許可の結果ではない。

| 反証軸 | 局所実測で検証したこと |
| --- | --- |
| QX21-A〜E | 同Query IDのSQL版変更、物理Relation/Role Policy/Snapshot/Measure丸め段階変更を別々に検出。表示値一致の偽陽性を防ぐ*比較候補* |
| QX21-F〜H | A→Bの選択変更は既存AのDirty Patch所有先を変えず、Grant版変更はValue自体の変更と区別、Renderer変更は提示資格のみ再確認 |
| QX21-I〜J | **証拠軸の欠落/None**は同値ではない。すべてのラベルが一致しても`provesCurrent=false`、`provesDisclosureAllowed=false`、`provesEffectCommitted=false` |
| QX21-K〜M | Owner review撤回と複合版差を区別。JSONに比較元/先のPrincipal等の**具体値を含めない**（ただし軸名・差の存在自体の公開資格は別） |
| QX21-N | **当初Referenceの不備を発見・修正**。欠落した`target_binding`とRenderer変更が同時にある場合、「changedだけ」を見ていた判定では必要なClaim/Disclosure/Editing再確認が漏れる。修正版は`unknownAxes`と`unknownQualifications`を独立表示し、**changedとunknown両方の必要再確認を和集合**で報告 |
| QX21-O | SQL bytes不変でもsearch_pathとAnalyzer invocationの版変更を区別。静的構造説明と実DB物理対象・実行時点の同一資格を混同しない |

とくに **unknown ≠ unchanged** が今回確定した局所設計上の反例。先行実装で比較不能な根拠があれば、差分検出されなかった軸も安全とは扱えない。一方、未知を理由に全履歴Claimを即座に廃棄することもせず、現在の該当用途に対する再確認・提示制約として扱う。これは「キャッシュが使える」「編集Intentを実行できる」と結論する実装ではない。

**表示/開示への注意**：本fixtureは差分を`changedAxes`等の軸名で示し、具体的なPrincipalや秘密Source文字列を出さないが、**どの軸が変更されたかというメタ情報も公開側には機密であり得る**。#2387 QX-20の人工二世界／#1693 D5の開示判断を通さない限り、内部比較票をSHO/AI/外部APIへそのまま配布しない。比較票による再確認候補と、既存Authorityで許可されたConsumer向けProjectionは別。

#### HJ/Consumerへの段階的受入・実行位置

1. **P0／同一Attemptの証拠対応**：#2387はQX-14/QX-18〜20と#2577の`DeriveFromAnalysis`合成Go試験を参照し、SQL source byte・Analyzer呼出し・Target名解決・実効可視集合・Result Source occurrenceを**observed/unknown**で整理。QX21ではversionラベルの真正性を推定しない。実DB EXPLAINと実行付き計測はHost/Provider権限の別要求。
2. **P1／#2586 HJ-01〜04**：Query演算/Input/Output Fieldを#2506のContext Instance/Slice/Grain/Completenessへ受渡し、QX21で指摘した未観測条件はClaimの適用資格を**未確定**に留める。#2593のMeasure（SUM/AVG/Stock・丸め/再集計）を対象ごとに評価する。QX17-M2のSQLite6件とは**別の製品Go/Worker/Provider統合試験**。
3. **P2／HJ-05〜07・編集後段**：#2582は既知依存の局所再利用とLEFT JOIN/NOT EXISTS/TOP-Nの撤回、#2597はAのDirty Patch保持、#2598はCandidate→実Requestの対象再束縛を独立受入。画面上同じ値とBusiness Save成功は分離。
4. **P3／用途別Projectionと性能**：#2516でSHO/Headless/外部APIの情報欠落・開示資格・保存版を比較し、#2600で不要な再計算と誤った省略の両方を測る。独立Oracleは#2514、要件の正式採用・意味承認は各Owner。実SQL Worker/実PG、実Provider/実Authorization、SHO保存、実Effect/Receipt、完全E2Eは**本QX21についてNOT RUN／未受理**。

**アーキテクチャ境界**：TEI独自Host、第二Query Runner、第二Provider、Evidence Registry、万能Policy Gate、新しい公開Core enumは先行導入しない。比較票は既存責務の不足を**実証可能な単位で接合**するための設計参照。QX-20のDB側実行モードを、静的Explainの標準呼出しに自動結合しない。


### 7.16 QX-20 M1：実Go DerivedIR→JSONを用いた「説明できる／公開できる」の別資格（2026-11-02、独立Draft）

**後続CI確認（2026-11-02）**：PR #2618のexact head `a54437c3da33b39ff12b919d6c92873f0eb35bdd`で、[Actions run 41419226291](https://github.com/hat47x/tei-platform/actions/runs/41419226291) の`repository-health` job 125190478489がsuccess。全体健全性・E1回帰・候補HEAD検証stepもsuccess。**QX-20の合成Go IRテストを含むPR回帰資格**であり、実SQL worker、正式Disclosure、実Consumer送信・Host/OSの合格とは別。下記の「Go/CI結果待ち」は起票当時の状態としてこの注記で更新する。


**前節との差**：§7.14では手組みのJSONで秘密の異なる二世界を比較した。今回はその実装隙間を小さくし、**mainの実`query.DeriveFromAnalysis`から生成された`DerivedIR`を、Goの`encoding/json`でシリアライズした出力**を対象にする。正式のDisclosure判断・SQL Worker・実Consumer公開を通した試験ではない。

**独立[Draft PR #2618](https://github.com/hat47x/tei-platform/pull/2618)**：main `9c9ef4d160640908e06ad13afb6c2b45c664c748`起点、`internal/query/derived_qx20_visibility_characterization_test.go` **Goテスト1ファイル**。head更新版 `a54437c3da33b39ff12b919d6c92873f0eb35bdd`（要exact-head再確認）。`go fmt`形状修正コミット済み。PRはDraftのまま。2026-11-02追検証で、head `a54437c3da33b39ff12b919d6c92873f0eb35bdd`を含むActions PR merge checkout `ce44843aa`のrepository-health [run 41419226291](https://github.com/hat47x/tei-platform/actions/runs/41419226291)、job `125190478489`について実ログでGo **1.26.10**、`gofmt -l . is empty`、`go vet ./...`、`go test -count=1 -timeout 60m ./...`、`tei/internal/query ok`、job successを確認。これは**PR merge refの全体Go suite PASS**であり、`TestQX20`名指定の単独実行や実Consumer/Disclosure受入を意味しない。

| Go合成Case | 直接確認する境界 | この試験の保証外 |
| --- | --- | --- |
| `TestQX20GoIRRawJSONCarriesPrivateIdentifiersAndPredicate` | 手組みAnalyzer Resultのprivate Source/Filter定数が、**実Go変換後のIR/JSONに残り得る**。`Coverage.Parse=full, Lowering=full`は公開許可の意味でない | 実SQL解析の正確さ・そのSourceの秘密指定・Host Disclosure |
| `TestQX20GoIRSameQueryIDAndFieldIDDoNotProvePublicObservationalEquality` | Query IDとOrdinal由来OutputField IDが同じでも、private Sourceだけ異なる二世界のraw IR JSONは異なる。**テスト専用で予めwithheldと仮定した要約**は同じ | Go本番Consumerの正規化/許可、異なるSQLの解析Invocation一致、情報非干渉の証明 |
| `TestQX20GoIRAnalyzerProvidedCoverageDoesNotEraseExpressions` | Analyzerが`Lowering=partial`でも、生IRの式文字列は保持され得る | `partial`を強制redaction policyとみなすこと |

**横断Ownerへの受渡し**：
1. #2387は`Query SQL source/result revision → Analyzer Result → Go IR/JSON`の実構造と不確定性を記録。OutputField IDの安定性をSource/Meaningの同一性と同一視しない（QX-14）。#2454/#2471 Draftや#2577 QX-17の範囲を上書きしない。
2. [#2516](https://github.com/hat47x/tei-platform/issues/2516)はConsumer/Claim/Scope相対の公開Projectionの**意味保存と説明十分性**、[#1178](https://github.com/hat47x/tei-platform/issues/1178)はDisclosureの現在の許可と開示される粒度を所有。`withheld`で非干渉だけ満たしても、利用者に必要な説明を満たすとは限らない。**公開不能と意味不成立は別**。
3. [#2515](https://github.com/hat47x/tei-platform/issues/2515)は実送出先・OS/Tool・DSH`messages`/`dsh_session_log`・Queue/Retry・実Receiverの証拠を所有。許可時点から送信時点までのPolicy/Role変更を加え、Test-only JSON値が等しいことから通信遮断を証明しない。
4. [#2604](https://github.com/hat47x/tei-platform/issues/2604)はSHO等の出力Artifact・Manifestの公開世代/実読取byteを所有。説明の固定版と現在のConsumerが実読込したbyteを別資格とし、第二の文書正本・Hostを設けない。
5. [#2514](https://github.com/hat47x/tei-platform/issues/2514)が独立Oracle・観測十分性を検証。手組みAnalyzer Resultとテスト専用要約の比較は、正式なDisclosure Policyに対する正例・負例を満たしたこととは別。

**次段の判別試験（M2案）**：同じ固定二世界を、`実SQL→Worker→Go DerivedIR`（Worker実体が入手できる場合）と`実Go DerivedIR→権限付きCLI/CUI→外部Consumerの実受信JSON`の2系統で比較。後者は`Consumer × Claim × Scope × Source/Policy Revision × Destination`ごとに、許可済み公開値・説明不足・公開禁止・実送信/不達を独立列にする。暗号化やハッシュ化だけではSourceの存在件数/秘密の同一性が漏れないことを証明できない。Mock Receiver/Canaryによる観測も検証証拠の再公開に注意する。

**実証・採否**：本節記載時点、実Goテストファイルは独立Draftへ保存済み。**PR merge refのGo 1.26.10全体CIはrun 41419226291でPASS確認済み。ただしSQLGlot Worker/実DB、正式CUI/SHO/AI Receiver、実Host Authority/OS egress、Owner受理は未実施・未確認**。QX-20の10件Python局所試験からGo3件/本番開示へ自動的にPASSを繰り上げない。#1119 G0〜G4のBlocking条件は追加しない。


### 7.17 QX-22：Native SQLの構造説明と汎用Data Operator/DSLへの移行可能性を分離する（2026-11-02、SQLite限定10件実測）

#### 1. 他レーンから統合する新しい方針

[#2620 Everything is Data](https://github.com/hat47x/tei-platform/issues/2620)、[#2621 Markdown Authoring](https://github.com/hat47x/tei-platform/issues/2621)、[#2622 SmartDB一般化](https://github.com/hat47x/tei-platform/issues/2622)、[#988 Operator/Language Plugin](https://github.com/hat47x/tei-platform/issues/988)では、Native SQL、簡潔なテキストDSL（Goa的宣言性／MukuroL的階層構文を候補）、表、関数記法、チェーン表現等を、**標準データ構造への異なる表層・Import/Projection**として扱う方向が明確になった。利用者の狙いは、SQLを既存の強力な入力形式として利用しつつ、必要に応じて結果や処理を共通DSL/標準データ構造で扱い、RDB・階層・Graph/Document等を同じStudio共通UIから説明・編集できることにある。

本Query領域の追加責務は、**SQL由来の構造候補が、どのOperator意味まで既に確認済みで、どこから先がTarget固有／表層差／未検証かを計測する**こと。DSL全体の文法/標準形の定義を#2387へ取り込まない。RDB・Graph・Documentを同じOperator名で記述可能にする目標と、意味的に同等であるという保証は別である。

#### 2. Authoring Sourceが正本になる三つの経路（候補）

| 経路 | 編集正本と参照の位置 | 何を確認してから編集・適用できるか |
| --- | --- | --- |
| **Q22-A SQL Native** | Sourceは既存native SQLのexact bytes。AnalyzerのScopes/RelationalIR/Lineage、標準データ形・Studio/SHO説明は**Sourceに帰属した導出Projection** | 説明は部分Coverageを保持。SQL本文の変更はSQL原本へ一意のSource span/版で反映可能な場合だけ。表やNodeを別正本として更新しない |
| **Q22-B SQL Import→DSLへ明示移管** | 既存SQLを解析し、DSL/標準構造Sourceとして保存する候補を生成。新Sourceを正式採用するかは別のOwner判断 | SQL入力の版・方言・パラメータ・Null/Bag/Order/Scope/Effectの保存、独立fixtureで観測比較、未知演算はopaque/unsupported。**単純なdesugarだけで正本を無言で置換しない** |
| **Q22-C Native SQL結果→Data Operator変換** | SQL Sourceは元の読み取り正本のまま。取得済みResult Sliceに対するフィルター／集計／Graph Traversal等の**別の導出処理**を標準構造・DSLで構成 | Result SliceのIdentity/Grain/可視Scope/Snapshot、Operatorの実Source/Target/純粋性・Context依存を引き継ぎ、結果の意味が不明ならCandidate。TEIが勝手にSQLを追加実行する契約ではない |

これらはいずれも、**データ操作の意味と実行を分ける**：SQL Targetへpushdownできるかは#2622/#1121/既存Data Providerが実現と実測を担当し、Python/Goで同等計算をする場合もそのProvider/Targetの適合を独立検証する。SQL ImportはSQL意味を全面再実装したと主張せず、未知のWindow/Function/CTE/方言・時点・実効security contextを保持できないときは読取り専用の不完全構造に留める。#1640 D2の実行前拒否は保持し、構造表示成功からSQL実行許可や書戻し権限を付与しない。

#### 3. `where`相当の標準Operatorは「同じ形」と「同じ意味」を別判定

SQLの`WHERE`、LINQ風filter、階層TreeのNode選別、GraphのVertex/Edge Traversal、JSON配列の絞込みは、形が似ていても、入力集合の種別・パス/関係Role、値のPresence、順序・重複、評価時点、副作用・遅延評価の条件が異なる。小さく合成可能なOperatorを目指しながら、以下を**Operator Contract/Role適合の質問**として保持する。

- **入力／出力のgrain**：row bag、identity集合、path occurrence、sequence、Group、Document nodeを区別。元Entity IDとQuery行・Graph traversal occurrenceを同一視しない。
- **述語の真偽**：SQL `TRUE/FALSE/UNKNOWN`、JSON `missing/null`、部分取得による`not-established`、権限上の`withheld`は同義ではない。`where`の否定・null比較や、存在を確認できたかの条件をOperator/Sourceで示す。
- **配置と順序**：LEFT JOINの`ON`内述語とJOIN後`WHERE`、`LIMIT`と`WHERE`、集約前`WHERE`と集約後`HAVING`は、一般に同じ`filter`を位置無視で移動してよいという意味ではない。実行順/短絡・関数のpure/volatile/effect要件は#988と既存Target Contractに従う。
- **存在/多重度**：`JOIN`と`EXISTS`、`NOT EXISTS`とnullableなRight fieldの`IS NULL`、`SUM(DISTINCT value)`と業務実体重複排除を区別。自動同値書換えには反例に勝つ根拠・型/Role/演算Lawが必要。
- **実現／観測**：SQLiteなどのローカル一例で結果一致しても、全SQL方言やProvider、現在のRLS/View/Function、実行計画や対象Datasetに対する一般同値は未立証。Workerの解析結果と実Targetで消費されたSource byte・Scope・時点を接合する。

**Positive vs Negativeの区別**：一つのINNER JOIN fixtureでは`ON`述語と`WHERE`述語で一致する一例がある。だがLEFT JOINやNullable列を含む別例では不一致になる。**同一fixtureでの正例を、抽象DSLの無条件な同値化規則に格上げしない**。また`where`が三値論理で`UNKNOWN`を落とすことと、Providerから実際に対象集合が欠けたためClaimが`not-established`であることを別列にする（QX-16/QX-21）。

#### 4. 独立SQLite対照：10件PASS（実SQL観測／Parser・DSL移行はNOT RUN）

新規[fixture/test_qx22_sql_operator_boundaries.py](https://github.com/hat47x/tei-platform/blob/work/query-structural-fixture-design-20261030/plan/fixtures/query-workbench-f0/test_qx22_sql_operator_boundaries.py)。実行環境Python **3.13.5**、SQLite **3.46.1**（JSON1使用）、`python3 -m unittest -v test_qx22_sql_operator_boundaries.py`で**10/10 PASS**。ローカル実行版とGitHub保存ファイルのGit blob SHA **`d9f887716db2b785a2424737e00fbff3fe4786dd`**が一致、bytes=**5400**。各SQL結果はテスト内の明示Expectedと比較した自作合成反例。

| Case | Native SQLite SQLの固定観測（自作fixture） | DSL/標準Operatorへ引き継ぐ条件 |
| --- | --- | --- |
| QX22-01 | LEFT JOIN `ON d.rank=99`はH1/H2をNULL右行として残す、JOIN後`WHERE d.rank=99`は0件 | Predicate配置とoptional sideの存在条件 |
| QX22-02 | 配送2件へJOINしたH1の明細2件は計4行、相関EXISTSなら2行 | EXISTSの存在判定とJOIN bag multiplicityを区別 |
| QX22-03 | 真のNOT EXISTSはH2だけ、`WHERE d.note IS NULL`はnullableな右実値D1を持つH1も含む | anti-joinへ直すためのright witness column/nullability証拠 |
| QX22-04 | `note = NULL`は0行、`note IS NULL`はD1、`NOT(note='ok')`はNULL行を含めない | SQL三値論理、Source NULLと取得未確定の分離 |
| QX22-05 | `LIMIT 1→amount>100`は0行、`amount>100→LIMIT 1`はL3=200 | 順位選択・母集団と演算順序 |
| QX22-06 | `WHERE amount>100→GROUP`はH2だけ、`GROUP→HAVING SUM>100`はH1/H2 | Row predicateとGroup predicateの型/Grain |
| QX22-07 | H1明細100+100はSUM=200、SUM(DISTINCT)=100 | 値同一性とFact/Entity同一性の別 |
| QX22-08 | 同一lineを自己JOINしてH1で(L1,L2),(L2,L1)の2ペア | Source occurrence/alias/scopeを保持 |
| QX22-09 | JSON `null`とキー欠落は`json_type`が異なる。空配列・空Objectのunnest=0行、JSON nullのunnest=1行 | 階層/DocumentのMissing/Null/Empty、unnest位置と結果grain |
| QX22-10 | このINNER JOINと純粋な`l.id='L1'`の局所例ではON述語とWHERE述語が同じ2行 | 局所同値Oracleを持つ**限定的な正例**。他Target/外部関数/方言へ一般化する前に法則検証 |

**証拠の射程**：実行したのはローカルSQLiteだけ。各ケースのTEI SQL Worker→Go DerivedIR、SQL→共通データ構造desugar、Goa/MukuroL風DSL parser、LINQ風`where`/Data Provider実行、RDB/Graph/Document共通評価器、Studio/SHO編集、実PostgreSQL、権限・Owner意味受理は**全てNOT RUN**。Native SQLのExpectedと自作人工DSL値を自動的に一致とした試験でもない。

#### 5. 既存Ownerへの実装可能な引継ぎ

1. **QX22-M0 / #2387**：実Worker/Go`DerivedIR`からQX22-01〜10に対応するOperation/Scope/Field/Expression/unknownをinspectし、SQL Authored Source・版・解析入力・Target実効Bindingの証拠と結びつける。`RelNode.Inputs` Operationと`OutputField.Inputs` Fieldは別名前空間、式文字列だけなら`operator semantics unverified`と出す。
2. **QX22-M1 / #2620/#988**：同じ例をFunction表／Goa風／MukuroL風／チェーン／標準データ形で与え、明示Role/Operator ID・semantics・Source span・Plugin版・NULL/Bag/Orderを保持してどの表層で同じ**プレーンなデータ構造**へ落ちるか測る。与えた構文は採用前の比較入力。Native SQL parser/source-of-truthを別実装で置換しない。
3. **QX22-M2 / #2622/#2506/#1121**：SQL Native→取得Result Slice→型付きData Operator（Data Provider依存）を一件通し、RDB/Document/Graphそれぞれ1例で適合/不適合・Target境界を調べる。元SQLへの自動書換え、Provider pushdownは対象Targetでの実証まで候補。
4. **QX22-M3 / #2621/#2497/#1124/#2516**：Markdown/Studio/SHOへ説明・Diff・Source navigationを共通投影。SQL Native由来の図/表の変更をSourceへ書き戻す場合、**一意なSource spanとOwner受理できる意味差分**が確認された箇所だけtyped patch候補。他のProjectionはread-only、曖昧なJOIN/Group/WindowやCTEはSourceへnavigate。
5. **採用判断**：#2514独立Oracleと#1131/#1180 Fixture、#1693/#1178開示、#1174実Effectを分担。新共通Parser・SQL executor・独自Hostを先行採用せず、構造的等価性・実Target同値・Owner意味受理・実Consumer受信を異なる証拠として保持する。


### 7.18 QX-23：SQL→共通Data Operatorの「局所的な一致」と「一般的な同義性」を分離する反証検査（2026-11-02、独立Oracle実測）

**他チャットからの接合**：[#2620 Everything is Data](https://github.com/hat47x/tei-platform/issues/2620)、[#2621 Markdown/Studio Authoring](https://github.com/hat47x/tei-platform/issues/2621)、[#2622 Data Operations](https://github.com/hat47x/tei-platform/issues/2622)、[#2625 Syntax Sugar](https://github.com/hat47x/tei-platform/pull/2625) の「複数表層を標準データ構造＋Role/Operator版へ解決する」方針を、前節QX-22のSQL構造理解へ接続する。QX-22の10件はSQL対SQLの固定期待比較だったため、**Native SQL実測と、SQLを呼ばない手書きPythonのPlain Data Operator参照計算の結果を対照し、誤った同義化を退ける変異（mutant）に反応するか**を追加検査した。Parser/Importer/DSLの新正本・共通Executorは導入しない。

**固定した局所実測**：[test_qx23_sql_operator_metamorphic.py](https://github.com/hat47x/tei-platform/blob/work/query-structural-fixture-design-20261030/plan/fixtures/query-workbench-f0/test_qx23_sql_operator_metamorphic.py) をPython stdlib `sqlite3/unittest`で実行して**9/9 PASS**、`py_compile` PASS。自作Line・Deliveryだけを用い、Null/0/2の3値×3行による27通り、Right 0〜3件、LEFT JOINの優先度パターン、NULL/重複の集約などを比較。**手元実行ファイルとGitHub保存のGit blob SHA `324e1a8594663ecdd6084040ca144e14b10ff69e`、bytes `7263`が一致**。Go/SQLGlot Worker/TEI DSL/Providerの実行証拠ではない。

| Pressure | SQL実測と独立Referenceの結果 | **潰すべき誤同義化** |
| --- | --- | --- |
| QX23-M1：JOIN bag対存在判定 | Right件数0〜3に対し、物理JOINの(row L, row D)多重集合を手書き直積＋条件で再現 | 1一致時の出力が等しくても、2一致の追加で`JOIN→EXISTS`は行多重度を失う |
| QX23-M2：限定的な正例 | このfixtureの非NULL一意Line IDについて、`WHERE EXISTS`と`JOIN→DISTINCT line.id`はRight0/1/2で一致 | 正例を全SQL方言・非純粋式・任意結果列・未確認Providerへ一般化する |
| QX23-M3：三値論理 | NULL/0/2の全27配列で`WHERE value != 0`と「NULL除外後の比較」を一致させる | Pythonの`None != 0`（True）をSQL UNKNOWNに対するTRUEとして扱う |
| QX23-M4：LEFT JOIN | 右の0件・NULL・priority 0/正値に対するLEFT JOIN/ONを手書きoptional-matchで再現 | ON内条件を外側WHEREへ動かし、右未一致行を消す |
| QX23-M5：Aggregate/Identity | GROUP SUM、空とNULL-onlyのSUM、COUNT(*)、COUNT(nonNULL)、SUM DISTINCTを独立計算 | 空SUM→0、SUM DISTINCTを業務Factの重複解消と扱う |
| QX23-M6：否定存在 | NOT EXISTSを手書き「右一致0件」と一致させる | nullableな右列の`IS NULL`を「右Relation不存在」と扱う |
| QX23-M7：順序 | Filter後ORDER/LIMITを別評価し、順序入替の誤結果を検出 | `filter∘limit`と`limit∘filter`の無条件同値 |
| QX23-M8：構文/Role候補 | 1一致時に出力値が等しくても、`join@v1/bag`と`exists@v1/set`の契約が別であることを明示 | 「実測の値一致」だけでRole/Operator identityを統合・Owner意味受理する |

**採用候補の重要原則：Operator意味の局所Conformance Packet**。最初から新Core Schemaを固定せず、各SQL→Plain Data変換候補に対し既存契約の参照として次の四層を並べる：(a) `Native Source bytes/Dialect/Schema/実効Target binding/Policy版`、(b) `明示Role・Operator ID/Version、入力と出力のGrain、Bag/Set/Sequence、Truth/NULL/Presence、演算順序・Effect`、(c) `Fixture と独立Expected、同義を破るMutant、観測値、不明な母集団/時点`、(d) `Target Provider/Consumer/Ownerの別資格`。**Mutantをkillできないfixtureは同義受入の証明力が不足**。Mutantに成功してもテストした有限値の判別力だけを示し、全入力・Targetの数学的同値は別証拠とする。

**実装・Owner分担**：
- #2387/#1640：Native SQLが正本のまま構造候補をExplainし、実Worker/Go解析が同じOperator情報を保持できるかの照合。D2が拒否するJOIN等の実行範囲を緩めない。
- #2620/#988/#2625：同じ候補をMarkdown表・Goa/MukuroL風・chain・Typed DOT・Plain Dataで記述し、Role/Operator版/参照・Source spanが保持されたか比較。**Source byteの違い／解釈候補の同値／Owner Accepted Meaning**を区別する。
- #2622/#2506/#2593：Provider pushdownまたはData Context内でのData Operator適用に対して、実Result Slice・取得Scope/Grain・再集計条件を取得して照合。Data Operationsが追加で読んだFactと、明示Mappingから要求された領域を分ける。#2611の複数DB alias同一Semantic対象は選択された実Targetまで確認。
- #2514/#1131/#1180：独立Oracle/Scenarioの所有。今のPython関数は実ProviderのTarget互換を自動判定する正式ルールではない。#2497/#2516/#1178はMeaning/Projection/Disclosure、#1174は現在Hostの実行/effect権限、#2604は固定版成果物の実Consumer読取を保持する。

**次段の2つのGate**：M1に`実SQLGlot Worker→Go RelationalIR→明示Operator Candidate`のうち**未対応・不明を残した受渡し**、M2に同じ受理済みRole/Operator契約を`RDB Provider + 非RDB Provider`で照合（物理Read、NULL/Missing、Bag、Order、Snapshot、権限、結果/Explain出所）する。両Providerで局所値が一致しても現在のCredential/Disclosure/Effectは別判定。新Parser・万能SQL Evaluator・独自Host・新正本を作らず、#1119 G0〜G4のBlocking条件を変更しない。

**状態**：Python/SQLite合成 **9/9局所PASS**まで。正式SQL Worker、Go Query、実DB Provider／RDB＋非RDB比較、DSL desugar、SHO/Studio/外部AI、Ownerによる意味受理は**NOT RUN**。同じSQLの静的Explain／ImportによるSource候補／SQL結果への後続Operator適用の三経路（§7.17）を区別する。


### 7.19 QX-24：Function Data Contextの明示Scopeと、Provider Pushdown／局所Data Operator実行の分割が意味を保存する条件（2026-11-02、独立SQLite＋Python 12件）

**横断方針との接続**：[Everything is Data #2620](https://github.com/hat47x/tei-platform/issues/2620)、[Data Operations #2622](https://github.com/hat47x/tei-platform/issues/2622)、[Function Context #2506](https://github.com/hat47x/tei-platform/issues/2506)、[Functionの明示Access Mapping Draft #2624](https://github.com/hat47x/tei-platform/pull/2624)、[Query→Context→Claim #2586](https://github.com/hat47x/tei-platform/issues/2586)から、通常の取得は**Data Model上のRelationを参照しつつ、Function Data Contextが使用するRelation path・Field・Scope・Grainを明示Mappingし、Providerを選ぶ**という方向を採る。SQL Nativeは特殊な必要性がある場合の明示Sourceであり、SQL構造Explainはその説明、標準Data Operatorは取得結果への意味付き演算という別責務。ModelにRelationが存在するだけで関連行を自動的に全取得することは意味しない。

QX-23はSQL内のOperatorの同義性／Mutant感度を試した。**QX-24の新差分は、同じ明示Operator列を「Providerで実行する部分」と「取得済みResult Slice上で実行する部分」に分割したとき、どの条件で同じClaimを成立させられるか**。SQLとプレーンデータの演算値が同じでも、実際に物理的に読んだデータ範囲／Consumerへの開示資格は別。すべてのOperatorをProviderへ押し下げることを要求する設計でも、すべてを局所実行にする設計でもない。

**限定実測の根拠**：[test_qx24_operator_split.py](https://github.com/hat47x/tei-platform/blob/work/query-structural-fixture-design-20261030/plan/fixtures/query-workbench-f0/test_qx24_operator_split.py)、Python **3.13.5**、SQLite **3.46.1**、標準ライブラリ`sqlite3/unittest/json`。ローカル`python3 -m unittest -v /mnt/data/test_qx24_operator_split.py`で **12/12 PASS**、`python3 -m py_compile` PASS、実行したUTF-8ファイルとGitHub保存blobのGit SHA-1 **`172da5ed0f4e91aca718b276ade30c5aab6d337e`**が一致（8636 bytes、SHA-256 `f516bbae26d6c8ace8bd9cb896c4af56951c549a9cf4b875a638b5c66c1b1ce1`）。SQL側は隔離SQLiteを本当に実行し、対照は手書きPython List/JSON計算。ただしPythonの参照計算は**TEIのData Operator実装でも、非RDB Provider実行でもない**。

| Case | 固定した観測結果（合成） | Provider／Context／Claimの必要条件 |
| --- | --- | --- |
| **QX24-01** 正常Pushdown/Local | Fullかつ同じauthorized Scopeの入力から、NULL除外＋Filter＋安定Sort/Top2は同じ`A2,A4` | Row Scope、Null Truth、Order/Stable Tie、完全な対象候補、同じSnapshotの仮定を保持すれば局所処理候補として適合 |
| **QX24-02** Scope→TopとTop→Scope | Scope後Top2 `A1,A2`、Top後Scopeは`A1`だけ | 任意`LIMIT`の前にScope条件が意味上適用されること。後段Filterが同じ対象を復元すると推測しない |
| **QX24-03** 公開値一致と実Read範囲 | SQLで`tenant=A`を絞った結果と、全A/BをSQLで取得後に局所的にAへ絞った結果は同じ | **値の等価性≠物理Read範囲／Least Privilege／送信許可の等価性**。これはDB Auth/RLSの実証ではなく、両SQLの取得行範囲差の有限観測 |
| **QX24-04** Partial page→Filter | `tenant=A`の先頭2件を読んでFilterした結果は`A2`、全候補からのTop2は`A2,A4` | `Partial`の後段Filter結果をGlobal Top-NのcompleteとしてClaimしない。追加ページを読んで必要な順位範囲の被覆を証明する方式は別途適合候補 |
| **QX24-05/06** Child Collection部分取得 | `A1`子先頭2件合計200だが実全3件合計250。`A2`はページに不在でも実子`C4`が存在 | 正例の存在Witnessと、Sum/不存在の完全性をClaimごとに判定。Providerの未取得集合とSQL NULLを区別 |
| **QX24-07** Page別平均 | 有効値10,20,50。page平均15と50の単純平均は32.5、全集合AVGは26.666…、sum+countの再合成なら一致 | Measure #2593の再集計則。Page境界の`AVG`値だけでは全集合`AVG`へ合成不可 |
| **QX24-08** 可視Scope集計 | AのSUM=80、世界全体のSUM=1380 | Result Sliceの`Scope`はMeasure Claimの意味の一部。全体SUMをAの可視SUMへラベルだけ付け替えない |
| **QX24-09** Document Presence | JSONの空children配列／`null`／キー欠落／子1件をSQLite JSON1 `json_type`で区別 | Record/Document providerのMissing/Null/Empty、Property Access、Flattenの演算契約をSQL NULLや未取得へ一括同値化しない |
| **QX24-10** Path Occurrence | 親A1が1件でも子配列Occurrenceは2件 | Graph/Tree等のPath/Edge occurrenceとEntity Identity、表示行Grainを区別 |
| **QX24-11** 観測値と現在再利用資格 | SUM=80が同じでもSource revisionが別、Complete属性が別 | QX-21のRevision/Snapshot/Completenessを現在のClaim再利用と混同しない |
| **QX24-12** 条件付き正常JOIN | 一意キー・同じLine/Name集合では局所join結果とSQLite SQL JOINが一致 | 正例のProvider分割は可能。ただしJoin Key/Occurrence/重複/NULL・時点/実権限が整うことを別検証する |

**Operator配置の意味保存を判別する提案（正式Schemaではない）**：
1. **要求**：Function Data Contextから明示された`Relation path / selected fields / requested row-scope / grain / claim population / source revision`。Data Modelの全Relationを自動取得対象へ拡大せず、特殊Native SQLは版付きSourceとして明示。
2. **計画候補**：`pushdown candidate`と`local-after-materialization candidate`を、(a) **Input Read/Disclosure権限**、(b) **物理Read footprint**、(c) **Provider能力とSQL/DSL Null/Bag/Sort/Collation/Effectの一致**、(d) **取得SliceのClaim相対CompletenessとSnapshot/epoch**、(e) **Measure law/Join occurrence**という独立資格で比較する。どこまで押し下げるかは実Providerの保証範囲に従う。
3. **正負双方の条件**：同じ権限で取得した**必要な候補全集合**があるなら純粋なFilter/Group/Sort等の局所評価は同義候補となる。Top-Nは全件Materialize以外にも、同じSnapshotの安定Sort/Keyset Cursorで**必要な順位範囲を取り切った証拠**があれば成立し得る。逆に限定ページや未知Collectionを`complete`に昇格しない。`EXISTS`の正Witnessは部分取得でも成立し得るが、`NOT EXISTS`・COUNT・SUMの全集合主張は閉じた母集団を要する。
4. **物理境界**：`SQL WHERE tenant=A`を実行した観測と、`SQL SELECT all → local filter A`は最終値が同じでも、先行Read範囲が違う。セキュリティ条件は実DB/Host/Providerの既存Authorityへ委譲し、内部で許可される局所評価・外部AI/SHOへの開示を別資格とする。Claimを公開できてもそのProducerによる物理Readが許されたことにはならない。
5. **結果資格**：候補Planの静的表現、実Providerで発行されたQuery/Read Footprint、実Observed Result Slice、Claimの十分性、Consumer向け開示、SHO/Artifactの実読取版を別列へ。#2516/#1178/#2515/#2604の既存担当を再利用する。新中央Provider Resolver、TEI独自Host、SQL Evaluator、第二データ正本は増やさない。

**次段M1/M2**：M1は実`analyzer.Result→DerivedIR`のSource/Operationと、#2624の`Function Access Map`から明示されたRead Footprint要求の対応を、同一Functionの1本のHeadless Packetで静的対照する。M2は#2622 DO-07のRDB系／非RDB系**実Provider**で同じRole/Operator契約を通し、実Read発行・Page/Continuation・NULL/Presence・時点・結果・権限／開示を観測。Providerの論理Resultだけ比較して物理取得範囲を証明したことにしない。Owner意味受理とD2 #1640の拒否境界、#1119既存G0〜G4は維持。

**状態**：Python＋SQLite局所**12/12 PASS**。TEI SQL Worker/Go Query↔Context↔Providerの同一Attempt、実RDB/非RDB Provider、正式Host/RLS、UI/SHO/外部AI、Owner Meaning/Releaseは**NOT RUN**。


### 7.20 QX-25：表示列と「計算に必要な列」と現在の開示資格を分離する（2026-11-02／SQLite Authorizer実測12件）

#### 横断Issueの接合と新しい欠落

[#2624 明示Function Data Context Mapping](https://github.com/hat47x/tei-platform/pull/2624)は宣言したRelation/Fieldの限定Projectionと、未宣言の秘密列の変更に依存しない合成結果を実証し、[#2620 Everything is Data](https://github.com/hat47x/tei-platform/issues/2620)／[#2625 Syntax Normal Form](https://github.com/hat47x/tei-platform/pull/2625)は表層構文を共通データ候補へdesugarする。これらは重要だが、実Queryは**出力列に現れないSourceをFilter、ORDER、EXISTS、JOIN、HAVING、集計、Window等で参照し得る**。公開可能な列名だけからデータ依存・取得必要列・開示安全性を決定できない。

既存の[#2387 QX-17](https://github.com/hat47x/tei-platform/issues/2387)ではValue LineageとExistence/Multiplicity/Selection dependencyの分離、QX-20ではJSON内の機密構造を扱った。今回は**SQLite本体の列参照認可コールバック（`set_authorizer`の`SQLITE_READ`）と、二つのデータ世界の実Query出力を同一試験で比較**する。静的Query IRでも手書きSQLの存在でもない、DB API上の狭い追加観測。あくまでSQLの準備時にSQLiteが確認する列参照であり、**実行時に物理ページや行を読んだ証明ではない**。

#### 四つのFootprint／資格の区別

| 層 | 表現すべきもの／実際の確認先 | 混同による誤判定 |
| --- | --- | --- |
| **Declared/semantic need** | Functionの明示MappingとOperator/Queryの入力依存、Scope・出現ID・Grain。出力列だけでなくPredicate/Sort/Join/Exists等の条件列を含む | 画面に`id`しか出ないのでQueryが`hidden_score`を使わない、または暗黙に読取許可済みとする |
| **Provider/SQL authorization reference** | どのSQL式に対し、どの列について権限判定が要求されたか。SQLite合成例は`SQLITE_READ`で観測 | 列参照がないことを全Provider/実物理row readなしと同一視、SQL authorization hookをHost Authorityへ昇格 |
| **Observed result/effect** | 実Queryの結果行、順序、Count、未確定/不完全集合、Target実行Attempt、Snapshot | 同じ結果が出たので秘匿列に依存しない、または異なるSQL実行が同じProvider可視Scope/実行許可を持つ |
| **Consumer disclosure and action** | Resultだけでなくmembership/aggregate/order、Explain/Diagnostic/Reason、Purpose・Caller・現在Grant、Intent/実Effect | 「秘密列を出力していない」だけでSHO/外部AI/標準UIへ開示可能と判断する |

**認可と秘匿の重要な区別**：秘匿Fieldを信頼済みProviderが内部で使用して、明示的に許可された集計だけ返す場合があり得る。この可能性を`全部拒否`へ潰さない。ただし許可済み公開集合や同一ProviderのRLS/Policy／対象Roleから、別のConsumerの公開権限を推論しない（[#1693 D5](https://github.com/hat47x/tei-platform/issues/1693)）。本ReferenceはSQLite内で選択的に拒否した例を示すだけで、認可済み集約のPolicyや差分攻撃防止を実証していない。

#### 検査の設計と実測（同じデータセットで列参照と行結果を比較）

追加：[test_qx25_hidden_read_disclosure.py](https://github.com/hat47x/tei-platform/blob/work/query-structural-fixture-design-20261030/plan/fixtures/query-workbench-f0/test_qx25_hidden_read_disclosure.py)。Python **3.13.5**、SQLite **3.46.1**、`unittest` **12/12 PASS**、`py_compile` PASS。GitHub保存版と実行版のGit blob SHA **`4695728b9d8345ea0eab2a982c9c730f30332852`**一致（6514 bytes）。すべて自作のSQLiteメモリ内表`item(id,visible,hidden_score)`と`link(item_id,hidden_note)`を使用し、秘密ラベルは合成Fixture上の分類である。

| ID | SQLiteで実際に比較したこと | Common Data/Function Contextへの引継ぎ |
| --- | --- | --- |
| QX25-01 | `SELECT id ... ORDER BY id`は`item.hidden_score`の列参照を要求しない | 出力・選択・順序の必要Sourceが本当に局所な場合だけ、その部分に限定して理由を説明 |
| QX25-02〜04 | `SELECT id WHERE hidden_score>0`、秘密列による`ORDER BY`、`COUNT(*) WHERE`が`hidden_score`のSQLITE_READを要求する | Plain Data/Syntax Sugarから`where/order/count`へ変換する際、OutputFieldの元列以外の依存を保持 |
| QX25-05／11 | `EXISTS`やJOIN Filterで別表`link.hidden_note`を参照する一方、出力は`item.id`のみ | Cross-relationのPredicate dependencyと実Providerで許されるRead範囲を独立に照合 |
| QX25-06〜07／11 | `SQLITE_READ`のうち秘密列をDENYした場合に、表示列が公開項目だけでもQuery準備・実行がエラーになる | Authorizationが必要なInput dependencyを表示列だけから決めず、denyを黙って空結果へ変換しない |
| QX25-08〜10 | 秘密Fieldだけを変え、Public ID列を不変にしても、結果membership、COUNT、TOP-1が変化する | 取得Snapshot/Read scope、結果/開示の現在資格。結果の存在／数／順序も開示し得る |
| QX25-12 | 実際の出力値が同じ`('I1',)`でも、Predicateの有無で`hidden_score`の列参照集合が異なる | 同じSource Query ID／結果値／Data Shapeは依存同一性・公開許可の証拠ではない |

**制約の明示**：SQLite `set_authorizer`はSQLの準備時の列参照検査であり、実表へアクセスした行数、物理IO、実接続時RLS、時刻、Provider実行主体、外部Host認可、許可された集約の公開Policy、OS/Network側の実送信、反復照会の推論漏洩を観測できない。このコールバックをTEIのData Provider共通API／監査APIに固定する提案ではない。`SQLITE_READ`列名の検出を、TEIの正式`read footprint`や完全なQuery依存として自動採用しない。

#### 次の統合受入（新Core/APIではなく既存Ownerの一往復比較票）

1. **M0 Query→Condition**（#2387）：実SQL worker／`DerivedIR`が出力列に加え、Filter/Sort/Exists/Join/GroupのOperandをどこまで保持しているかを固定SQLから検証。Expression文字列だけで全依存を確定できない場合は`unknown`。同じSQL・Schema/Source版・実Attempt・SQLite列参照記録を**異なる観測証拠として**一件に結合する。
2. **M1 Context Mapping→Provider**（#2624/#2506/#2622）：宣言されたFunction必要Fieldと、Queryが実際に必要とする条件Fieldを比較。Mapping不足の場合はData Context不足／Provider固有の許可された内部計算／実取得観測不可を区別。秘密列を自動追加取得したり、非公開列をRendererへ載せたりしない。物理的な過剰Readの実証はProvider Ownerへ委譲。
3. **M2 Disclosure→Consumer**（#1693/#1178/#2516）：`SELECT id WHERE hidden_score>0`の行集合、`COUNT(*)`、Sort/Top-N、Query Explain/Errorを公開する先ごとに、処理の許可・結果開示許可・現在のGrantを分離して比較。公開不可の場合でも内部の分析資格を不当に消さず、Consumerには許容された情報だけを投影。
4. **M3 横断Conformance**（#2625/#988/#2514/#2619）：共通データの`where/order/count`がSQLのOperand/Scope/Truth/Bag・Source列を保つことを試験。有限Fixtureの結果一致と、実Provider権限・業務意味受理は別のGate。合成Syntax normal formが同じでも実読取/開示の条件が違うTargetでは同値ではない。

**現状**：実SQLite AuthorizerとQueryの12件の局所観測をGitHubへ保存済み。**実TEI SQLGlot Worker/Go→Context/Provider/Studio/SHO、実RLS/Host認可、複数Consumer実送信、Owner意味受理は未実施**。QX-17〜24の既存設計、#1640 D2拒否、別Draft #2624/#2625、正式Semantic/Target Ownerを維持する。


### 7.21 QX-26：ページ取得の継続カーソルと共同Read Cohortの資格を、実トランザクションの前後で検証する（2026-11-02、SQLite WAL 11件実測）

**前節との差と他レーンの接合**：QX-24（§7.19）は`同じSnapshot + 安定Sort + 必要な順位範囲まで取得`でLocal/PushdownのTop-N同値を候補とした。QX-25（§7.20）は表示しないWHERE/ORDER/EXISTS列の依存を分離した。[#2627 Provider Route/Read Cohort](https://github.com/hat47x/tei-platform/pull/2627)の合成Go Referenceは異経路ReadのSlice比較資格、[#2582 Reactive Context](https://github.com/hat47x/tei-platform/issues/2582)はepoch/遅着、[#2506 Function Data Context](https://github.com/hat47x/tei-platform/issues/2506)はRequested Scope vs Observed Completenessを扱う。ここでは**同じSQL・同じTarget・同じPrincipalでも、複数ページの間に書込みがあれば、結合した結果がどの時点にも存在しない偽Snapshotになり得る**ことを、実SQLite二接続とWAL Snapshotの対照で確かめる。ProviderやDB一般のトランザクション仕様は推測しない。

**実測**：[test_qx26_pagination_read_cohort.py](https://github.com/hat47x/tei-platform/blob/work/query-structural-fixture-design-20261030/plan/fixtures/query-workbench-f0/test_qx26_pagination_read_cohort.py) をPython stdlib`sqlite3/tempfile/unittest`で隔離実行。ファイルDBに`reader`と`writer`の別接続を設け、`journal_mode=WAL`で**書込みをページ間に実コミット**する。Python環境の`python3 -m unittest -v /mnt/data/test_qx26_pagination_read_cohort.py`は **11/11 PASS**、`python3 -m py_compile` PASS。手元の実行ソースとGitHub保存blob Git SHA-1 **`1ad11f63c909fb4af37d859e8e6b1121deea1539`**一致（7844 bytes）、SHA-256 `fc4a4f8602690b7cea0560952320b80d83e25d03a59b8a0d59bdee3de321b814`。SQLite WAL/同一DBの事実であり、PostgreSQLやREST/GraphQLのIsolation/Cursorを代弁しない。

| Case | ページ間で固定・変更した入力と実測 | 必要な資格 |
| --- | --- | --- |
| **QX26-01** 正常Offset | 静止集合A1〜A4に安定`ORDER BY priority,id`、Offset 0/2を適用し全4件取得 | 有限固定集合では正常なページ連結が成立する正例 |
| **QX26-02/03** Offsetと先頭へのInsert/Delete | 先頭A0を追加すると`A1,A2`＋`A2,A3`、A1を削除すると`A1,A2`＋`A4` | **重複/未取得**とSource Snapshotの変更を分け、OffsetはConcurrent Write下で全件の証明にならない |
| **QX26-04** Keyset + 先頭Insert | Keysetの`(priority,id) > (2,A2)`で再開すると旧A1〜A4を取得、現時点のA0を欠落 | Keysetにより重複を回避しても、**現在集合に対するComplete**は独立 |
| **QX26-05** Keyset + 旧値変更＋新行挿入 | 旧A1.amount=10、次ページで新A25（同priority2だがID順でA2の後）を取得。同時に現在A1.amount=100 | 結合した行値・membershipは**旧Snapshotにも新Snapshotにも同時には存在しない**。同じQuery byte/Keysetだけでは共同Read保証にならない |
| **QX26-06** WAL read transactionの限定正例 | Reader`BEGIN`→Page1、別WriterがA0追加/A1変更をcommit、Reader Page2とCOUNTは旧A1〜A4・COUNT4を維持。Reader rollback後は新A0を観測 | SQLite WALの**実読取トランザクション**における固定Snapshotと、新しい取引での別Snapshotを区別。RESTページや他RDBへ一般化しない |
| **QX26-07** キー重複・Tie breaker | priority1のA1/A2で`priority>1`の継続はA2欠落、`(priority,id)>(1,A1)`ならA2取得 | Cursorの安定総順序、同一Collation/NULL/Key Contract/Identity |
| **QX26-08** Cursor Scope不一致 | Aの継続`after=(2,A2)`をBへ使用するとB1が返らない。B単独全件はB1あり | 継続対象Source/tenant/Filter/Policy/Query版にCursorを束縛し、別Contextへの持越しを再資格化。Fixtureのtoken字句は暗号学的証拠ではない |
| **QX26-09** Filter改訂 | `amount>=20`のPage1から`amount>=0`で再開しても旧条件で除外したA1を復元できない | 同じTargetでもOperator predicate/Parameter/Rule Versionの変更でCursorの適用資格が失効 |
| **QX26-10** Count一致の罠 | 旧COUNT4と、旧ページ連結4件が一致しても、ページ間Insert後の現COUNTは5 | `count(rows seen)==old count`は同一Read Cohort・現行Completeの証拠ではない |
| **QX26-11** Snapshot名だけの一致 | `snapshot='s1'`・同じrevision文字列を持った人工Cursorの前後でDB行集合が変更 | **文字列ラベル一致≠実Providerが継続して同一Snapshotを読んだ証拠**。Token issuerと実Read期間を検証 |

#### 再開可能なQuery/Context/Operatorの契約候補（正式Schemaではない）

- **要求の帰属**：`Source SQL/Operator ID+version, Dialect, Source digest, Target/DB Alias, effective principal/RLS/policy, Mapping/Function Instance, row scope, parameter/predicate, grain, output order+tie-breaker/collation, requested snapshot/freshness`を、既存Source/Context/Hostの正本へ参照する。全フィールドを新中央Packetへ複製しない。
- **ページ実観測**：各Pageの`Actual Provider/Endpoint, request+continuation, consumed SQL/parameters, result slice+field presence, row identity/order, actual snapshot/transaction evidence, physical read footprint, page termination/continuation`を実Provider側の証拠として対照。**全ページが同じSnapshotとみなせる保証**がない限り、複数ページの結合を`observed_complete`と無条件に昇格しない。`same token`の主張だけではなく実Providerが保障する方法が必要。
- **Claim-relative application**：部分Pageでも具体的存在Witnessは成立し得る一方、`NOT EXISTS`、全集合`COUNT/SUM`、Global`TopN`、再計算されたValidation、Action全件更新は必要なRead Cohort/Scopeの閉域性を別判定。ABAや非表示Rule入力の変化は#2582/#2624の依存伝播を再利用する。入替えがあれば表示だけでなくRule/Action Candidateの再評価資格にも影響する。
- **正規の実装分担**：本#2387はSQL Query/Explainの読取意味・Source/Order/Scope前提のみ、[#2506](https://github.com/hat47x/tei-platform/issues/2506)/[#2586](https://github.com/hat47x/tei-platform/issues/2586)はContext→Claimの適用可否、[#2622](https://github.com/hat47x/tei-platform/issues/2622)/[#2627](https://github.com/hat47x/tei-platform/pull/2627)は実ProviderのRead Cohort/Capability、[#2582](https://github.com/hat47x/tei-platform/issues/2582)は異ReadのInvalidation、[#1174](https://github.com/hat47x/tei-platform/issues/1174)は実Host/Effect。#2516/#1178/SHOは同じCorpusを公開可能な範囲だけ投影する。**新Provider/Transaction Manager/Query Executor/共通Host/第二Source正本は追加しない**。

**次段M1/M2**：M1は現行`analyzer.Result→DerivedIR`のSort/Limit/Filter/Source、Function Mapping、Provider Page request/resume条件を**同じFunctionのHeadless Attempt**へ結ぶ。M2は少なくとも実RDB Providerと実API Providerで、それぞれが提供するCursor/Snapshot/ETag/Continuationの能力と、**ページ間更新・Role/Filter改訂・Token再利用**時の正負を独立に計測。RDBのSQLite WAL一例や合成Snapshot文字列を多Providerの同値証拠へ昇格しない。#1640 D2安全拒否・#1119 G0〜G4・Owner意味受理のOwnerと優先順位は維持。

**状態**：Python＋SQLite WAL限定の**11/11 PASS**まで。実TEI SQL Worker/Go Query/Function Context/正式Provider、PostgreSQL/API Cross Provider、Host/RLS、GUI/SHO/外部AI送信、正式Owner Meaning Acceptanceは**NOT RUN**。


### 7.22 QX-27：非出力Predicate・Sort・相関条件の依存と、Go DerivedIRのCoverage信頼境界（2026-11-02、独立Go Draft）

#### QX-25（実SQLite）から一段進めた実Goの検証点

QX-25（§7.20、SQLite Authorizer 12件）は、`SELECT id`でもWHERE/ORDER/EXISTS/JOIN/HAVINGなど**出力しない列の参照**により、集合の存在・件数・順序が変化することを確認した。[#2624 Function Context](https://github.com/hat47x/tei-platform/pull/2624)では表示依存とRule/Validation入力依存を区別し、[#2622 Data Operations](https://github.com/hat47x/tei-platform/issues/2622)では論理要求／実物理Read／Consumer開示を分割した。

**新しい狭い確認**：[Draft #2634](https://github.com/hat47x/tei-platform/pull/2634)の[test `internal/query/derived_qx27_dependency_characterization_test.go`](https://github.com/hat47x/tei-platform/blob/work/qx27-go-hidden-dependency-20261102/internal/query/derived_qx27_dependency_characterization_test.go)は、現行main `9c9ef4d`の実Go `DeriveFromAnalysis`を、**手製の`analyzer.Result`**から呼ぶ5テストだけを加える。本体の`DerivedIR`/契約やRuntimeには変更を加えず、SQL Authoring Sourceを正本、IRを導出Projectionとして扱う。

| Go特性確認 | 与えた合成Analyzer入力と判別点 | 一般化する前に要る証拠 |
| --- | --- | --- |
| **QX27-M1** 非出力WHERE | 出力`o.id`だけが`OutputFields[0].Inputs`に載り、`o.secret_score`は`Fields`の別occurrenceと`RelFilter.Expressions`に残る | **Output値Lineageと全Input Read Dependencyを同一視しない**。条件列へのSource/Scope束縛・存在/選択への影響は別 |
| **QX27-M2** 同じOutputでも違うPredicate | 同じQuery ID／OutputField ID・Inputsのまま、`secret_score>50`↔`>70`で`RelationalIR`全JSONが異なる | 画面に出たField ID・Projection JSON一致だけでRule/Claim依存不変や公開許可を保証しない |
| **QX27-M3** 非出力ORDER | `o.id`だけ出力、`secret_rank`は列OccurrenceとSort Expressionに残る | Sortの値・比較規則/NULL・総順序/選択TopNを別依存として保持。QX-26 Page/Continuationに接合 |
| **QX27-M4** 相関EXISTS相当 | root出力と、subqueryの非出力`hidden_flag`、outer相関`o.id`を別Scope/Occurrenceで保持。root Relational Rootはsubquery列挙で置き換わらない | 相関Field・Parent Scopeの構造記述と、実SQL Workerが存在/否定の意味を正しくLowerすることは別 |
| **QX27-M5** Coverageの入力信頼 | 故意に`secret_score`の`ColumnRefs`だけを落とした手製の`analyzer.Result`でも、`Lowering=full`はGo側で`CoverageFull`として残る | **実Workerが欠落する証拠ではない**。Go LayerのCoverage表示は独立の入力列依存完全性／実Read/開示資格の検査ではない |

**証拠状態**：テストは`gofmt -l`空出力、実行前のローカルGoソースとGitHub保存Blob `04ef05606fae03e96d60c3e52d8be5d4bdced021`一致（7400 bytes）。[Draft #2634](https://github.com/hat47x/tei-platform/pull/2634) exact head `0485441f3c6a8dec718e63548c68abce069bb02a`のGitHub ActionsはPR起票後の観測で**queued**。この時点ではGoテストを実行したPASSは宣言しない。#2618のQX-20 Raw JSON特性試験は別PRとして維持する。

#### 後続への引渡し：Column dependencyの用途別表現

- **Display/Output Dependency**：出力値への直接Lineage（`OutputField.Inputs`）を表示する。
- **Selection/Existence/Order Dependency**：WHERE/HAVING/JOIN/EXISTS/ORDER/LIMIT/Window/GroupのInput Field/Relation/Scopeを、SQL解析が明示できた範囲で説明する。`ColumnRefs`存在と、条件式への厳密なDataflow Role解決は別。式文字列から全依存を正確に復元できる保証はない。
- **Rule/Decision Dependency**：[#2624](https://github.com/hat47x/tei-platform/pull/2624)のFunction Rule Definition/Attachmentと[#2506](https://github.com/hat47x/tei-platform/issues/2506)の明示Contextから要求。Displayに見えないRule Fieldを実依存から除外せず、かつ未宣言Relationを黙って取得しない。入力欠落／版/Scope不整合は`unresolved`、業務Rule falseとは区別。
- **Actual Read Footprint/Disclosure**：SQL Layerが参照可能と示した列と、実Providerが読んだ列・行・Snapshot、外部Consumerに開示できる集合は異なる。#2622/#1174/#1178/#2516/#2515の既存Ownerで別証拠を確認。Raw Go IRの直接公開を安全とみなさない。

**次段M1/M2**：M1は実Python SQL Workerから生成した`analyzer.Result`の正負`ColumnRefs`と`Operations`／Coverageを、SQLite QX-25の列参照・行結果と同一Source/版/SQL上で比較。M2は実Go IR → Function Access Map → Data Operations/Validation → Reader/Consumerの**同一Attempt**で入力依存、部分Read、Snapshot/Disclosureの資格を確認。Go test CharacterizationからSQL全面同義・実Provider適合・Owner受理には昇格しない。#1640 D2の拒否と#1119 G0〜G4は維持し、新Query Executor/中央Host/第二Policy Engineを追加しない。

**未実証**：本DraftのGoテスト実行結果はCI確認まで未確定。実SQL Worker、実PostgreSQL/API、Function Data Context結合、Windows/SHO/AI実Receiver、Owner MeaningはNOT RUN。


### 7.23 D2AF：Provider読取り後のHost項目認可再確認を、Query Result公開境界へ接続する（2026-11-02、実Go Driver回帰候補）

#### 他レーンの成熟結果から発見したD2／D3の違い

QX-20／QX-25／QX-27は、SQLやGo `DerivedIR`の説明能力・Input Lineage・Raw JSONと、実Consumerへの公開資格を分けた。QX-24／QX-26は`Provider Read`と`Result Slice / Claim`が別の段階・時点の観測であると確認した。これらを実際のGo Driver実装で照合すると、**D2通常の`ExecuteDerivedBrowseQuery`がProvider実読取り前に`ValidateDerivedSelect`を実行しても、RawとDerivedの値を返す直前には現在のHost項目読取権限を再検証していない**。一方、既存のD3 `DerivedMemo.Resolve`は非cache/計算後やSource witnessの再確認時に`ValidateDerivedSelect`を再実行する。この相違は`Query結果の取得を始めてよい`と`取得した結果を呼出側へ返してよい`の間の失効条件として具体化できる。

**局所変更候補**：[独立Draft #2637](https://github.com/hat47x/tei-platform/pull/2637)は、現行main `9c9ef4d160640908e06ad13afb6c2b45c664c748`からDriver本体の最小変更＋Goテスト1ファイルだけを追加した。SQL/Query/ResultColumnMappingやD2 `ValidateDerivedSelect`の制約、`CompleteBrowseQueryExecutor`の契約を変更せず、現在Hostの`FieldReadAuthorization`を**dispatch前とDerived Resultを返す直前**の二地点で確認する。既存Derived Result側のRaw/Derived項目、Operator、Snapshot/Target/Consumer公開の権限正本を増やさない。

| Case | 合成Host／Providerで行う具体的変化 | 期待する受入 |
| --- | --- | --- |
| **D2AF-01 Raw input** | Preflightでは`inventory.product.stock`のHost Readを許可。Providerのbounded complete read callbackでそのGrantを失効させる | 実Provider callbackは1度だけ。**Raw Result/Derived Resultを返さず**`ErrDerivedUnauthorized`となる |
| **D2AF-02 Derived target** | Preflightでは導出項目`inventory.product.available`を許可、Provider read中に失効 | Query/Raw値の取得が完了しても、導出結果の公開直前に拒否。生データと導出値を異なる公開資格に昇格しない |
| **D2AF-03 Unchanged authority** | 全Mapped Raw項目とDerived TargetのHost Readがread前後に維持される | 同じ既存D2源データの計算結果を返せる。既存Preflight拒否テストを維持 |
| **D2AF-04／今後** | 実HostのAuthority Stateが途中でrevoked/ABA、別Principal、別Provider/DB Sessionに切替 | 二地点のCallback試験だけでは決定不能。Hostの認可取得時点／原子性・実Provider Readの帰属を別証拠にする |
| **D2AF-05／今後** | Providerが返却するRawに非表示Field/Rule Inputが含まれ、SHO/外部AIへ後段公開 | Driverでread許可が成立してもConsumer目的/PolicyとOS/Network実Receiverの公開資格を別検収 |

**今回の変更の正確な範囲**：`ValidateDerivedSelect`の後段再検査は、Provider callback中に失効しその状態が続いた場合を識別する**現在Grantのフェンス**であり、トランザクション単位の権限Snapshot、ABA（失効→復活）、DB/RLSを含む一貫したRead Cohort、ConsumerのDisclosure、Hostが既に外部へ送出した情報の巻戻しまで保証しない。元の`canRead` CallbackのAuthorityは外部Host／Provider側に帰属し、TEI内に独自Host・Policy Engineは設けない。

**受入状態**：実Go Driverの変更と合成Callback試験は[Draft #2637](https://github.com/hat47x/tei-platform/pull/2637)にGitHub保存済み。**本節追記時点でActions exact-headはqueued、Go PASS未確認**。実DB Provider、Host Grant変更同時実行、Wails/SHO/AI実送出、RLS、Owner意味受理はNOT RUN。#1640のD2実行範囲（単一raw simple projectionのみ）の拒否と#1119 G0〜G4を維持する。

**Owner分担と実装順**：#1640/D2 DriverはPreflight/Result返却資格、#1641/D3はCache Witness・Version/鮮度/現在認可、#1174は実Host Current Authority、#2622/#2627は実Provider Read/Completeness、#2586はFunction Context/Claim帰属、#2516/#1178はConsumer別Disclosure、#2514は独立Oracle。最小Go回帰PASS→実Host/ProviderでGrant更新を模擬した同一Attempt→Raw/Derived出力およびConsumer/Sinkの拒否観測へと段階的に進める。


### 7.24 QX-28：Data Operatorの多Target同値を「演算名」ではなくValue Semantics／Evaluation Profileで判定する（2026-11-02、SQLite＋Python/Decimal 14件実測）

#### 新差分と横断的な根拠

QX-22〜24はSQL→標準Operator／Provider Pushdown→Local計算の局所同値を扱い、QX-25/27は値に表示されない依存・許可条件、QX-26はページ間のSort/Continuation条件を扱った。しかし、**同じ演算名・同じ入力値・同じ表示型に見えても、Collation・Unicode正規化・NULL配置・数値の内部表現・丸め・整数除算・オーバーフローによって結果・行集合・次ページ・例外が変わる**。TEIで一つのData Operator定義をRDB、非RDB、Goから生成するJava／.NET／WASMの各Targetへ展開するなら、これらの意味を明示して適合判定する必要がある。

既存Ownerは[#988 Operatorの意味契約](https://github.com/hat47x/tei-platform/issues/988)、[#2622 DO-07のRDB/非RDB Provider比較](https://github.com/hat47x/tei-platform/issues/2622)、[#2593 Measure・再集計](https://github.com/hat47x/tei-platform/issues/2593)、[#2077 別言語Target](https://github.com/hat47x/tei-platform/issues/2077)、[#2056 Query inputのMissing/Null](https://github.com/hat47x/tei-platform/issues/2056)、[#2506 Function Data Context](https://github.com/hat47x/tei-platform/issues/2506)、[#2586 同一Function縦断](https://github.com/hat47x/tei-platform/issues/2586)。新しいDecimal型・Collation API・独立Query Evaluator・共通Hostの採用決定ではなく、既存意味契約へ渡す差分の計測。

#### 限定実測：SQLite実Query対Python標準ライブラリ

新規：[test_qx28_value_semantics_cross_target.py](https://github.com/hat47x/tei-platform/blob/work/query-structural-fixture-design-20261030/plan/fixtures/query-workbench-f0/test_qx28_value_semantics_cross_target.py)。Python 3.13.5、SQLite 3.46.1、stdlib `sqlite3/decimal/unicodedata/unittest`。ローカルの**14/14 unittest PASS／py_compile PASS**、実行したUTF-8版とGitHub保存版のblob SHA **`2ef8f46ca02301491ce1ef7426ece88b164e496f`**一致（**7491 bytes**、SHA-256 `4b1f97b2d858dba3506c29820490e26d530074ef1dd1abbe462c63183ffafc54`）。自作・隔離Fixtureのみであり、実TEI Worker／Provider／Java／.NET／WASM実装の試験ではない。

| Case | SQLとPythonの有限観測／負例 | 契約の必要観点 |
| --- | --- | --- |
| **QX28-01** | `val`のBINARY Top2はID **4,2**、NOCASE Top2は**1,4** | Sort/Top-NのCollationとTie-breaker、Source/Contextを変えた時の再開資格 |
| **QX28-02/03** | SQLite標準NOCASEは`Ä`≠`ä`、Python `casefold`では一致。合成文字`é`≠分解文字`e + combining acute`（SQLite BINARY）、NFC後は一致 | UnicodeのCase-fold/Normalizationは暗黙同一視せず、明示Spec/versionとSource由来を別扱い |
| **QX28-04** | SQLiteの通常ASC NULL先頭に対し`NULLS LAST`では末尾 | NULL/Missing/Absent/Unfetchedの区別とSortのNULL配置。QX-26 Cursor資格にも影響 |
| **QX28-05/06** | JOIN Key `a`対`A`がBINARYでは0件／NOCASEでは1件。`TEXT '10','2'`の順序は数値順序と逆 | Equality/Join/FilterとSortのType/Collation/Coercionを表示の装飾属性ではなく演算意味として扱う |
| **QX28-07/09** | SQLite REAL `0.1+0.2`、`SUM(CAST(text AS REAL))`はPython `Decimal('0.3')`と厳密同値にならない | Money等のFixed Point/Decimal・中間演算・集計・Scale・丸め時点を型付き契約で確認する |
| **QX28-08** | 64bit整数`9007199254740993`はSQL Integer保持、一度REALへCastすると`9007199254740992` | 多言語TargetのBigInt/Decimal/Float移送、JSON Number経由の精度・Identity損失 |
| **QX28-10** | SQLite `ROUND(2.5,0)=3`、`ROUND(-2.5,0)=-3`、Decimal half-evenは±2 | 業務丸めをSQLite標準やJava BigDecimalのデフォルトへ無根拠に推測しない。Ownerが採用した規則と演算時点が必要 |
| **QX28-11** | SQLite `5/2=2`、`5/2.0=2.5`、Python Decimal `5/2=2.5` | Operand Type/CoercionとOperator versionが同一でない限り式字面の一致は同値証明ではない |
| **QX28-12** | SQLite INTEGER `SUM(INT64_MAX,1)`は`integer overflow`、Python `int`のsumは成功 | 成功値だけでなくOverflow/Error/RejectとRaw/Domain/Technical Failureの表現が一致するか |
| **QX28-13** | BINARYで1ページ目 `B,D`を取得してNOCASE Cursor `>D`で続けると残り0件。NOCASEで最初から読むと`a,B,c,D` | 同じCursor値・不変Datasetでも、Sort/Collation差だけでRead Cohort Completenessを失う |
| **QX28-14** | SQL空集合 `SUM`はNULL、`TOTAL`は0.0、Python `sum([])`は0 | Empty Set Aggregateの結果は演算名とOutput Type/NULL則で明示し、0と不成立を混同しない |

#### 多Provider／多言語への引継ぎ：用途相対Evaluation Profile（**正式Schema候補ではない**）

1. **Input Value Contract**：値型、整数幅／Decimal precision・scale、NULL／missing／unset、Field presence、Units、Path/Relation Grain、必要な強制変換と失敗を、受理済みSemantic・Query入力へ戻して参照する。型名一致だけで暗黙型変換が一致するとはみなさない。
2. **Operator Semantics**：Equality/Filter、Sort／Total Order、Join/Group、Null Truth／Null Sort、Unicode Normalization/Collation実装・version、数値演算／中間丸め・Overflow、結果がNULLか0かの規則を、既存#988のOperator/Target実現の**個別適合条件**にする。全Operatorへ常に全属性を強制するのではなく、意味に関係する軸を明示する。
3. **Execution Placement**：#2622のProvider Pushdown／局所計算／生成先Targetそれぞれが要求される部分契約を満たすときだけ同義候補。Provider Capabilityがない場合にLocalへ戻せるかは、#2506の取得許可・完全性、#1174の権限、#2516の開示で再判定する。コスト最適化は意味とPermissionの成立後の比較。
4. **Snapshot/Continuation**：#2627／QX-26のPage Cursorは、Source Revision・Query/Mapping・Effective Principal/Scope・Snapshotに加え、**Collation/Normalization/Sort null placement/tie-breakerの意味版**が変われば失効対象。ラベルのbyte一致や`ORDER BY`句の字面一致だけでは保証にならない。
5. **Independent Oracle**：#1131/#2514の同じ型付きDatasetに、期待値と失敗の独立判定を固定。Java `BigDecimal`の入力変換／NULLやJSON欠落、.NET `decimal`、WASMのint/float、SQL RDBとREST/GraphQLは、それぞれ実Adapter/Providerの行動を別に測定する。丸め・税区分・通貨・時点はOwner判断事項として記録し、未受理の試験値からBusiness Contractへ昇格しない。
6. **Evidence/Disclosure**：手書きPython/SQLite比較で判明した差→実TEI演算の値／Errorを測定→実Provider/Target毎のRelease資格→SHO用の説明と出典→Owner受理を別欄に保持。新しい中央評価Engine/Renderer正本を増やさない。

**優先実証M1/M2**：M1はQX-27の実Worker出力・DerivedIRと、Sort/Filter/Result型、#2624 Mappingの参照対象から必要Evaluation Profileを列挙するRead-only比較。M2はDO-07で実RDBとAPI系Provider、#2077でJava/.NET/WASMの**同じAccepted Operand/Operator期待値・Error**を独立Oracleで測り、値・Sort/TopN・Error・Read Footprint・Current Authority・Disclosureを別記する。QX-27実Go PR #2634の旧exact headで、GoテストのJSON `>` が `\\u003e`へHTML escapeされることを見落とした**検証コード側の偽失敗**を観測した。比較文字列をIRの式へ変更する修正は別PRの新HEADに提出済みだが、修正版のCI結果は別確認する。#2624も文書一覧への未登録によるCI失敗を旧HEADで観測し、該当Draftへ一覧修正を提出した。**PRの合成テスト結果とCI健全性を必ずexact headで確認し、失敗の種類を区別する**。

**状態**：SQLite/Python local **14/14 PASS**／GitHub blob一致。TEI SQL Worker→Go→Function Context→実Provider、RDB/APIの多ProviderConformance、Java/.NET/WASM、Host/Current Grant・RLS、SHO/UI/外部AIへの実開示、Owner業務意味受理は**NOT RUN**。



### 7.25 QX-29：標準Data OperatorをJavaの実言語意味へ写像する場合のValue／Error／丸め時点を独立測定（2026-11-02、JDK 21実行20件）

**新差分**：QX-28（§7.24）はSQLite SQLとPython/Decimalの標準ライブラリによる局所比較であり、TEIの多言語Targetという製品価値に対し、Java実行そのものの実測が欠けていた。そこで既存[#2077 別言語生成](https://github.com/hat47x/tei-platform/issues/2077)、[#988 Operator Contract](https://github.com/hat47x/tei-platform/issues/988)、[#2622 DO-07](https://github.com/hat47x/tei-platform/issues/2622)、[#2593 Measure](https://github.com/hat47x/tei-platform/issues/2593)、[#2056 Missing/NULL](https://github.com/hat47x/tei-platform/issues/2056)の受入観点へ、Javaの実行環境で観測した**計算結果と技術失敗**を追加する。**TEIが生成したJavaコードではなく、手書き独立Reference**であり、SQL⇄Javaの全面同義性や正式DSL semanticsを採用したものではない。

[QX29JavaValueReference.java](https://github.com/hat47x/tei-platform/blob/work/query-structural-fixture-design-20261030/plan/fixtures/query-workbench-f0/QX29JavaValueReference.java) をローカルOpenJDK 21.0.12.1で`javac -encoding UTF-8`→`java QX29JavaValueReference`として**実際にコンパイル・実行し20/20 PASS**を確認。実行ソースのGit blob SHAとGitHub保存ファイルblob SHAはともに **`4ab62ebf4ec56caf17d02267ce25d69f59c37da2`**（実ファイル5563 byte）で一致。JUnit等の別依存はなし、Java classファイルはGitHubへ納品せずReferenceのSourceのみ保存した。

| Test | 実Java 21で固定した出力／失敗 | SQL / Python / TEIの意味比較に必要な区別 |
| --- | --- | --- |
| **QX29-01〜03** | `new BigDecimal("0.1") + new BigDecimal("0.2") = 0.3`。Java `double 0.1+0.2 != 0.3`。`new BigDecimal(0.1d)`は`new BigDecimal("0.1")`と数値不一致、`BigDecimal.valueOf(0.1)`はこの値例で一致 | 入力の10進文字列／binary float／float→Decimal Adapterの違い、Precision Source |
| **QX29-04〜05** | ±2.5にHALF_EVENは±2、HALF_UPは±3 | SQLite `ROUND(±2.5,0)`とも比較、未受理の業務丸めをTargetデフォルトへ固定しない |
| **QX29-06** | `1.005×2`をhalf-evenで明細先行2桁に丸めると**2.00**、総和後2桁に丸めると**2.01** | **同じ丸めモード・Scaleでも丸め時点が違えば結果が違う**。Data Rule/Measure/集計Grain/Stageへの明示Bindingが必要 |
| **QX29-07〜08** | `Math.addExact(Long.MAX_VALUE,1)`は`ArithmeticException`、`BigInteger+1`は成功し9223372036854775808 | 入力型の幅・Overflow／Error vs 成功値・生成先ABI・上限 |
| **QX29-09〜10** | 不正数値文字列→`NumberFormatException`。Java Mapでキー欠落とキーあり明示`null`を区別 | 入力変換失敗をBusiness Rejectへ付け替えず、missing/nullのRole/Meaningを保持 |
| **QX29-11** | `BigDecimal("1.0").compareTo(BigDecimal("1.00")) == 0` だが`equals`はfalse | **数値的な同値≠Scale/型表現としての同一性**。Query DISTINCT/Group/Keyの比較則とSource byte identityを別扱い |
| **QX29-12〜13** | `BigDecimal.ONE.divide(new BigDecimal("3"))`は非終端小数で`ArithmeticException`、Scale2/HALF_EVEN指定なら0.33 | 割り算の精度/Scale/丸めが未指定なら「実装が自然に正しい結果」を出すとは扱わない |
| **QX29-14〜16** | Java `"Ä".equalsIgnoreCase("ä")`はtrue（QX-28のSQLite NOCASEではfalse）。Unicode NFCは明示すると合成/分解を同一化。`Comparator.nullsLast`はNULLを末尾へ | Unicode/Collation/Normalization/NULL orderを明示Evaluation条件とし、Java String比較をSQL COLLATEと無条件交換しない |
| **QX29-17〜18** | `Long.parseLong("1e2")`は`NumberFormatException`、`new BigDecimal("1e2")`は数値100 | **指数表記の受容可否はAdapter／型の仕様判断**。同じ100を返し得ることから入力言語が同義とはならない |
| **QX29-19** | 空Listを`Stream.reduce(add)`すると`Optional.empty`、単位元`BigDecimal.ZERO`指定なら0 | SQL空SUM=NULL、TOTAL=0.0との接合。Empty/Missing/Nullと元集合の完全性は別 |
| **QX29-20** | BigDecimalの`-0.00`は`signum=0`、Java doubleの`-0.0`と`0.0`はraw bitsが異なる | 0と負ゼロの数値同値・表現同一性・serializeを混同しない |

**成熟した設計含意**：
- SQL/Go/Java/.NET/WASMの**値だけが一致しても**、入力パース／Null Presence／Representation Scale／関数評価順／rounding stage／Overflow・Technical Reject／Result Serializationが異なれば、全体契約は同値とはいえない。対象Function/Operatorの意味に関わる軸だけを#988/#2077/#2622の既存適合試験へ加える。
- Javaでは`BigDecimal`を使用すれば自動的に意味保存が保証されるわけではない（数値文字列生成の起点、`new BigDecimal(double)`、`equals`対`compareTo`、`divide`の失敗、丸め時点）。「推測する丸め規則」については現行の未対応規則拒否を維持し、Ownerが受理した規則と入力契約なしに生成器で既定採用しない。
- **小さく再現可能な受入段階**：V0 Source/Operand/Role/Operator版の照合、V1 独立Java/SQL/Go計算とParserの対照、V2 TEI同一モデルから生成したJava JAR／.NET assembly／WASMの**公開API + JSON Adapter**を別々に実行、V3 外部HostでのAuth/Tx/Provider/Receipt（Read/Write別）、V4 Source/Data/Function/Business/SHOのOwnerによる個別受理。V1のPASSをV2〜V4へ自動昇格しない。
- 新しいTEI独自Host、中央Provider Resolver、独立SQL Emulator、第二Canonical Store、JDKへのランタイム依存を生成物全体へ一律追加する提案ではない。Goは開発ツール実装、Javaは生成先の一候補。Context/Operator/Adapter/Hostを具体的案件に応じた薄いBindingで接続する。

**未実証**：TEI実モデル→同一生成器→JAR作成、生成した全.classの固定・比較、実Java JSON入力Adapter、同一Sourceの.NET/WASM Target、実Provider/Host/OS Egress/SHO、Owner Meaning Acceptanceは**NOT RUN**。Java 20件は実JDKの限定比較であり、実TEI生成器や対象データへの意味適用実証ではない。


### 7.26 QX-30：値の意味だけでなく、外部JVMが実際に組み立てるJava全class閉包の整合性を分離する（2026-11-02、JDK 21実行12件）

#### QX-29からの追加発見：不一致が実行エラーにならず「正常に誤計算」する

QX-29（§7.25）は同じJava 21処理系の`BigDecimal`・数値変換・例外・丸め時点を20例で独立測定した。しかし**主クラスのバイトが完全一致していても、依存する副クラスの違いだけで外部Consumerが値としては正常に受理できる誤結果を返す**場合を検証していなかった。既存[#2512 Java ArtifactSet](https://github.com/hat47x/tei-platform/issues/2512)の`Main.class`だけでは`Main$Input.class`が無く`NoClassDefFoundError`という試験と異なり、今回は**必要classは存在し、外部JVM実行も正常終了するが、値が意味的に変わる負例**を固定する。#2513型付きJava Library、#1102 ArtifactSet、#1105 Store、#1370 Verified→Loaded byte、#2390 外部Consumer、#2600 増分ビルド、#2077 多言語Target、#2514独立Oracleと接続する。

**独立実測Reference**：[test_qx30_java_artifact_semantics.py](https://github.com/hat47x/tei-platform/blob/work/query-structural-fixture-design-20261030/plan/fixtures/query-workbench-f0/test_qx30_java_artifact_semantics.py)。Python 3.13.5の`unittest/tempfile/subprocess/hashlib/json`で**実OpenJDK 21.0.12.1のjavac＋java**を呼び、各試験ごとに固定したJava `qx30.LineAmount`公開クラス、内部`qx30.Rule`、**別source/別compile outputの`ExternalConsumer`**を組み立てる。実JDKで**12/12 unittest PASS**、`py_compile` PASS。Python実行ソースとGitHub保存ファイルのblob SHA-1 **`6352be74621aa3e5f954a8349095306d49b2f02d`**一致、8416 bytes、SHA256 `b17c50d1cfc83d8cfe24d028d61e0fda13c4e8b1723ca9a5fbfc89c277d5c12a`。個別生成classのSHA-256とパス集合を比較し、集合fingerprintは**隔離fixture独自のJSON正規化**であって#1102の正式ArtifactSet Schema/Digestではない。

| Case | JDK実行・全class観測 | 設計上の意味 |
| --- | --- | --- |
| **QX30-01/02** 主`LineAmount.class`のbyteはHALF_UP版とHALF_EVEN版でSHA-256完全一致。内部`Rule.class`は異なり、二member集合digestも異なる | 主class単独digest／主API identityと必要な実行class閉包を分離。生成物が同じに見えても丸め実装は一致しない |
| **QX30-03/04** clean Java Consumerの`1.005`は両版で正常終了するが、UP=**1.01**、EVEN=**1.00**。主classをUP版のまま内部RuleだけEVEN版に入れ替えても正常終了で1.00 | **必要memberの混在はNoClassDefFoundErrorを伴わず、成功として誤った値を返し得る**。主classの検証成功・実行成功・Meaning一致は別 |
| **QX30-05** 主classだけを独立classpathへ置くと実`NoClassDefFoundError: qx30/Rule`で失敗 | 既存#2512の主class単体では実行閉包を満たさない負例を別実装で再現 |
| **QX30-06/07** 二classを新規隔離先へコピーし、全member digestと集合digestが一致すれば正常なJava Consumerが1.01を返す。一方主class同じ／Rule異なる混成Setはmember検査で拒否候補 | byte一致した全memberコピーで実行可能性を限定実証。ただし本fixtureは**TEI Store Put/Getや後続ClassLoaderが実際に読んだbyteを逐次検証したものではない** |
| **QX30-08** 両版を同時にclass pathへ置き順序を入れ替えると、両方正常終了でもUP優先1.01／EVEN優先1.00、CodeSource URLも先行ディレクトリを示す | **Classpath shadow**は主class・外部ConsumerのSource不変でも解決する実装が変わる。CodeSource URLだけを検証済み実ロードbyteの証明にしない |
| **QX30-09/10** `1.006`なら両版とも1.01であり、観測値一致でも依存Rule集合は異なる。`-1.005`のtieではUP=-1.01、EVEN=-1.00で差を検出 | Oracleは選んだ入力に依存してmutantが生き残る。Case判別力・必要な境界値を#2514へ |
| **QX30-11/12** fixture用Member Mapの列挙順を変えてもSet digest一致。一member欠落と一member内容差を別分類 | 依存閉包の`missing member`、`member byte change`、`order-only presentation`を別証拠として保持 |

#### 実装／受入の新しい判断境界

1. **生成・配布物の必要閉包**：同一Accepted Model/Function/Operator version/Generator/Toolchain→Java生成Source→全class role/path/digest（必要ならResource/JAR/Moduleと依存Libraryの閉包）を#1102へ渡す。ソース版/公開APIのbyte一致があっても`Rule.class`だけの変更を再生成・配布・Conformance検査の対象から除外しない。設計書SHOの説明に書かれた半偶数丸め等は**同じSource Meaningを参照する導出情報**であり、コメント文面と主classバイトだけではRuntime Rule版を固定しない。
2. **Artifact Source identity／Store availability／Loaded codeを分ける**：全memberのRole/Path/Digestを#2512の実Store Put/Getで再構成した後、#1370で**検証したbytesと実ClassLoaderが解決したbytes**の境界を確認する。今回のCodeSource URLはロード元ディレクトリの観測にとどまり、ロード途中の改変・既ロードclass・外部依存のshadow・JVMTI等の一般防御を証明しない。
3. **意味上の丸めと実際の選択**：HALF_UPとHALF_EVENの両方を人工fixtureで正常に実行できたが、どちらがTEIの受理済み業務丸めかは未決。#2291の未対応丸めを生成前拒否する境界と#2077の正式Target実現を維持。Consumerが成功したことをOwner Meaning受理とみなさない。
4. **独立Oracleと現在再利用資格**：`1.006`の値一致はRule差の判別力がない。正負tie・Scale・Missing/Null・adapter parse・overflowなど必要集合に対する独立Oracleを#2514と#1131に置く。#2600では説明だけの変更とRuntime Rule.class変更を別Impactとし、**ArtifactSet全member閉包・Consumer必要Role・Target実ロード元・Test Evidence**まで実消費依存として扱う。変更されたRule.classがクラスパスに残り得るとき、主class cache hitだけでskipしない。
5. **順序**：P1は#2512の既存実Store/Set codeと本fixtureの必要member比較を一試行に接続→P2は#2513の正式TEIモデルから生成Java Library/公開型付きAPI/JSON Adapterを**別プロジェクトのConsumer**で実起動し、同一Contract/Fixtureに対する結果を測定→P3はSource+生成器+全artifact digest+実Host/Consumer時点を同一AttemptのReceiptに結合、同じ意味からSHO資料への出典/版を投影。新Host/Registry/中央Cache/新Operator Meaning正本の導入はしない。

**状態の区別**：実Java外部Consumer、二版classの実Build/実Load、仮の全class集合fingerprint、反例を含む**12/12 RUN PASS**。既存TEI製品の`ArtifactSet`/Store Put/Get・生成Java JAR/正式JSON Adapter、同一受理モデル・実Provider・Host/Trust/SHO、Owner意味受理は**NOT RUN**。本書/PRはDraft/Openであり、#1640 D2の実行拒否境界、#1119既存G0〜G4、TEI独自Host非採用を維持。


### 7.27 QX-31：実SQLのRelation path再束縛と否定的集合依存を、現在Claimへ接続する（2026-11-02、SQLite12件実測）

詳細な実証票／独立期待値／引継ぎは、追加文書 [**QUERY_QX31_DYNAMIC_PATH_AND_NEGATIVE_DEPENDENCY_2026-11.md**](./QUERY_QX31_DYNAMIC_PATH_AND_NEGATIVE_DEPENDENCY_2026-11.md) に切り出す（既存計画の長大化と同時編集を抑えるため）。[実行済みFixture](../fixtures/query-workbench-f0/test_qx31_relation_path_negative_read.py)はPython **3.13.5**／SQLite **3.46.1**、`unittest 12/12 PASS`／`py_compile PASS`。実行した6298-byte sourceとGitHub保存Git blob **`187277b93fb61e7e715c1f39fe9d2c362925567c`**一致。

[#2582の動的依存・0件否定依存DC-01〜12](https://github.com/hat47x/tei-platform/issues/2582#issuecomment-6579762087)と[#951顧客照会の0件・ABA追補](https://github.com/hat47x/tei-platform/pull/2640)を、**実SQLiteの`orders→customer→account` JOIN／相関`NOT EXISTS holds`**で独立比較した。O1の表示金額と旧A1与信上限を固定してO1.customerRefをC1→C2へ再束縛すると、実入力はA2へ移りResultが撤回される。現在Hold0件に行IDが無くてもActive Hold追加でResultは消える。逆にO2 Hold追加はこの合成O1 SQLを変えず、**過大失効と過少失効を同一Fixtureで分けた**。0→1→0のABA、同じResultでも別Relation path、部分ページで見えないHoldからの偽`NOT EXISTS`、NULLによるUNKNOWNも検査。

**設計の停止線**：Value Lineage／出力ID／表示値の一致、Query静的ColumnRefs、Read snapshot文字列の一致を、動的Binding・対象Scope・否定的集合のComplete witness／現在Grantの証明にしない。SQL 0件はBusiness false、Provider未取得、開示不可の同義ではない。静的条件／実Provider read／Consumer claim／Host effectを既存#2387/#2506/#2624/#2582/#2622/#2627/#2586/#2516/#1174へ受け渡す。**実TEI SQLGlot→Go IR→Function Context→Provider→SHO/Owner**はNOT RUN。D2拒否・SQL正本・Draftを保持し、新Query Engine／共有Host／Dependency Registryを追加しない。


### 7.28 QX-32：二つのProviderが個別には正常でも、一つのFunctionの判断を成立させる「共通の読取り時点」は保証されない（2026-11-03、SQLite 2DB/WAL 13件実測）

#### 横断議論から確定した独立の課題

QX-26（同一SQLite DBの2ページとSnapshot）、QX-31（Relation path再束縛・否定依存）、[#2627 Read Cohort](https://github.com/hat47x/tei-platform/pull/2627)（合成複数Provider Route）、[#2624 Function Access Map](https://github.com/hat47x/tei-platform/pull/2624)（Functionごとの明示Data Mapping）、[#2582 Context Reactive](https://github.com/hat47x/tei-platform/issues/2582)（遅着/再評価）、[#2640 顧客照会のQuery Result](https://github.com/hat47x/tei-platform/pull/2640)を接合すると、**各Providerが自身の現在データを正しく返しても、別々の時点で読んだ値を一つのClaimに合成すると、どの実時点にも存在しなかった結論を作り得る**。

これは、単に「同じSQL/Function版」「Query結果の同じ形」「Providerがそれぞれcomplete」「二つのProviderが同じopaque snapshot labelを返した」という確認では消えない。TEI独自の分散Txや共通Snapshot Managerを導入する話ではなく、**必要な共同Read/Claimを証明できなければ `unresolved / stale / cannot-establish` として保留する**受入条件である。単一ProviderのSQLite WALトランザクション成功を、API/GraphQL/OData等の複数Providerが共有する原子的Snapshotに昇格しない。

#### 再現可能な限定実測（実SQLite DBを二つ使う）

追加：[test_qx32_cross_provider_read_cohort.py](https://github.com/hat47x/tei-platform/blob/work/query-structural-fixture-design-20261030/plan/fixtures/query-workbench-f0/test_qx32_cross_provider_read_cohort.py)。Python 3.13.5、SQLite 3.46.1の標準`sqlite3/unittest/tempfile`で、**異なるSQLite DBファイル（Routing / Policy）、それぞれWriter/Readerを分離し、WALで実コミットを挟む**。手書き期待値と実SQL値を比較し、`python3 -m unittest -v` 13/13 PASS、`py_compile` PASS。GitHub保存blobと実行ソースのGit blob SHAが**`d22bf05866e0cf5af96a3b59e5de450353197ec9`**で一致（**9614 bytes**、SHA-256 `b5aafd34e69e374635f509747f0538ecbc0d96b32dfa0c120f80c527eda83f33`）。合成モデルは架空の`tenant/order→account`と`account→credit ceiling`、Active Holdの関係。正式な与信業務Ruleではない。

| Case | 実SQLite二DBで観測したこと | 条件／不成立の理由 |
| --- | --- | --- |
| **QX32-01** | 初期`O1→A1`、A1 ceiling500、amount600でDenied | 同一世界で正しく不成立を示す基準 |
| **QX32-02** | t0:`O1→A1, A1=500` → t1:Routeだけ`A2`へ変更 → t2:旧A1のceiling700へ変更。各時点ともDenial。しかしt0に読んだA1とt2に読んだA1=700を合成すると**偽のAllow** | **False Positive / Fractured Read**：各Reader個別正常でも合成した値はどの実時点にも一致しない。旧Path＋新Valueの結合を拒否 |
| **QX32-03** | t0:`A1=700`、t1 Route`A2`（A2も700）、t2旧A1だけ500。各時点ともAllowだが旧Route＋新A1=500を合成すると**偽のDeny** | **False Negative**もある。性能問題でなく意味/判断の誤り |
| **QX32-04** | RouteのRevisionを読取り前後で比較すると、間のRoute変更を検知 | 明示された単一Providerの単調な実Revisionなら、ある種類の変更検知に利用可能。ただし共同Snapshot証明やABA防止は別 |
| **QX32-05** | Routing側の実WAL読取トランザクションを`BEGIN`で固定しても、Policy側は新Commitを読めるため偽Allowになり得る | **一方のDBだけSnapshot固定≠二Providerの共同Snapshot**。別Providerへ保証を勝手に拡張しない |
| **QX32-06** | 人工Token文字列`cohort-1`は不変のまま、双方の実DB Revisionが変わる | Label一致からProvider issuer/取得期間の証拠を推測しない |
| **QX32-07〜08** | `holds`が0件だった後にActive Hold追加→古い不存在証明を使うと偽Allow。逆に古いActive Hold観測後、削除されると偽Deny | **NOT EXISTS／Exists Witness**にはRelation Path／Predicate／Authority Scope／Read Cohort／実時点と失効が必要 |
| **QX32-09** | 同じ`account_id=A1`でもTenant Aはceiling500、Tenant Bは1000 | Provider横断IDはAccount名だけでは足りず、**Tenant/Context/Resolved Binding**を一致させる。悪いBindingは偽Allowを導く |
| **QX32-10** | O2のPathだけ変更すると粗いRouting Revisionは進むがO1の現在Claim値は不変 | Global revisionによる過剰失効と、依存閉包の局所正しさを区別。未照合依存を省略する理由にはしない |
| **QX32-11** | Active Holdが実在しても、部分取得`LIMIT 0`は空Listを返す。空Listのみで評価すれば偽Allow | `requested all`／`returned empty`／`observed complete`は別。Negative Claimには閉域証拠が必要 |
| **QX32-12** | 未存在テーブルのProvider SQLは`OperationalError`。同じ世界に実Holdがある | Provider Error→空Collection/0件への変換を禁止。技術失敗・権限拒否・業務不成立は別 |
| **QX32-13** | Read前の各Revisionが1/1、Read後が2/2であり、その間に偽Allowを構成できる | 単なるVersionの採番/照合では原子的Readを作れない。**同一認証主体・Scope・全入力取得が一つのClaimに適用可能**である根拠が必要 |

#### 既存Ownerへの追加受入条件（正式新Schemaではなく、同一Attemptの比較票）

1. **Logical dependency**（#2506/#2387/#2586）：Function/Queryが読む`tenant,order→account`と`tenant,account→ceiling`、HoldのNegative Predicateを、Source occurrence/Identity/Path/Scope/Output grainで明示。利用しないRelationへ暗黙に拡張しない。QX31の0件依存とQX27の非出力条件も列挙する。
2. **Per-Provider actual read**（#2622/#2627/#2611）：必要な各Entity/Field/Routeと、実際のQuery/Request/Actor/RLS/Revision/Read interval/CompletenessをProviderごとに記録。`same caller`・`same declared Scope`・`same token`でも、そのProviderでの読取り結果しか主張しない。
3. **Joint Claim qualification**（#2497/#2582/#2586/#2593）：二つのReadが同時に成立したとみなせる保証がなければ、値が単純なBooleanでも`unresolved`。API ProviderのVersion/ETag/Continuationは意味が違い、同一Snapshot APIが無い場合には再取得・相関版照合・用途別拒否候補を検討し、**Read→Validate→Actの間にさらに変更され得る**ことも保持する。無制限の再試行や誤った強制直列化を提案しない。
4. **Current Authority/Disclosure**（#1174/#2516/#1178）：結論の正しさと公開可能性/Hostの現在Grant/Effectは独立。旧Claimや古いNegative WitnessからAction許可やSHO/外部LLMへの公開を推定しない。成功したProvider readを完了したBusiness Actionへ昇格させない。
5. **M1/M2**：まず既存TEI `Query Analyzer→DerivedIR→Function Access Map→Provider Route/Read Cohort→Claim/Pattern`を**一つのFunction/同一Run**で接続し、Routing/Policyの二Providerで上記の正負と`unknown`を記録。次に少なくとも実RDB Provider＋実API Providerで、Connection／Version/ETag／RLS／当該Consumerの可視入力を分けて測る。新たな中央分散Tx Engine・Global Revision・新Host・第二Canonical/Oracle正本は作らない。

**資格と優先順位**：SQLite実DB二つを使った局所13件のみ実測。TEI SQLGlot Worker→Go IR→実Function Context、TEIの二つの実Provider、API/GraphQL/OData横断Read、実Host/Consumer/SHO、Ownerによる業務意味受理は**NOT RUN**。#1640 D2のComplex Query実行拒否、#1119 G0〜G4、外部Host方針を維持。PR #2624（明示Function Context）、#2627（合成Provider Route）、#2634（Go依存）、#2637（D2の読取後Grant再検査）、#2629（横断計画）は**本節更新のGitHub照合ではmerged**。それぞれ局所実装・CI資格を示すにとどまり、単一Function E2Eは別途検証する。


### 7.29 QX-33：Analyzer ResponseのQuery ID照合とSource／Schema実帰属の差を実Goプロトコルで特性記述（2026-11-03）

別途進んだ§7.28 QX-32は**二つのProviderにまたがる共同Read Cohort**の有限実証であり、本節の**Analyzer Invocation境界**とは別。重複する試験番号・受入資格にしない。詳細は[**QUERY_QX33_ANALYZER_INVOCATION_BINDING_AND_VERTICAL_HANDOFF_2026-11.md**](./QUERY_QX33_ANALYZER_INVOCATION_BINDING_AND_VERTICAL_HANDOFF_2026-11.md)。

実Worker実験の着手可否・Producer所在の監査・V01-A〜F正負対と停止条件は[**QX-33 V0/V1 Producer監査・受入票**](./QUERY_QX33_V0_V1_WORKER_READINESS_AND_ACCEPTANCE_2026-11.md)に分離した。main `2e16b2af9531225b73430a32a6dbb2031e07f4c1`のrecursive treeではSQLGlotを実行するPython WorkerをRepo内で特定できず、現実行環境にも`sqlglot` packageがないため、現時点で**V0/V1は`NOT RUN: producer-unavailable`**（外部Repo／別配布での存在は未調査）。これはQX-14b/c/dおよび隔離Go HelperのPASSとは独立した欠落であり、SQLGlotの代替Parser／証明Registry／新Hostを作る理由にしない。

**実コード・状態の更新**：既存`internal/query/analyzer/client.go`の`RunPythonWorker`は実`Request`をstdinでWorkerへ渡し、`Result.Query`と`ProtocolVersion`を要求へ照合する（#2471、**merged**）。一方ResultにはSource/Schemaの実消費byteがechoされず、Query ID/Protocolだけでは保存済みResultを別SQL/Schema/Targetへ適用できる**独立の版同一性証明**とはならない。#2634（QX-27実Go DerivedIRの合成解析入力特性）、#2637（D2 Provider後のHost FieldRead再検査）も**merged**。本書§7.22/7.23の「queued」は当時の履歴であり、現在の未mergeを意味しない。両者のmergeは**実SQLGlot Worker→実Go→Provider→SHO**の合格ではない。

**保存した実Goテスト**：[client_qx33_request_binding_characterization_test.go](../../internal/query/analyzer/client_qx33_request_binding_characterization_test.go)（Git blob `f0c6113ca2ff5a3fd3b3febe8faf18c6855d1cb2`、3443 bytes）。既存`RunPythonWorker`へ自己プロセスのテストHelperを渡す。HelperがSQL/Schemaを**意図的に無視する**とき、同じQuery/Protocolを保つSQL Filter変更・Table変更・Schema変更の応答を静的契約上受理し得ることをcharacterizeする。**ローカル`gofmt -l`空／SourceのGitHub byte一致を確認**。さらにGo **1.23.2**の隔離moduleへ、GitHub mainとbyte一致する実`analyzer/client.go`（Git blob `9620cd9e2ad6d7b041ca868318a3f45da09fc59d`）と本テストのexact sourceを配置し、周辺`model.go`型・`core`／`environment`のみstubにして、4 subtest＋Helperの**`go test -v` PASS**、`go test -race`／`go vet` PASSを実測。**実TEI repo全体のexact-head Actionsは未確認**。これは実SQLGlot Workerの結果取り違えバグを観測したものではない。実Process invocationが渡したstdin/stdoutをCaller側で結び付ける可能性も残す。

**次の一本の試験**：QX33-V0：実Authored SQL byte／選択Template分岐／Dialect／Schema Context／Worker実行版と実stdinを同一Attemptに帰属 → V1：Workerの実Output／Go `DerivedIR`の`OutputField`値Lineage、非表示Predicate/Sort/EXISTS/Path、Partial/Unknownを**欠落時はunknownのまま**比較 → V2：Function Data Context/受理済みRole・Operatorと必要Field/Relation pathを接合 → V3：Provider/Read Cohort/否定的集合のComplete Witness → V4：Consumer開示／実Host Grant/Effectを別資格とする。**SQLGlot Worker実呼出しのV0/V1が未実証のうちは、合成Python/Goの成功数を増やしても製品E2E受理に昇格させない**。

既存#2387/#2506/#2586/#2622/#2582/#2516/#1174のOwnerへ段階検証を引継ぎ、Worker responseの新Field・新Proof Registry・新Query Executor／Hostを即採用しない。#1640 D2の複雑SQL実行拒否、#1119の既存G0〜G4、Source一正本を維持。


## 8. 実装順・担当引継ぎ（提案）

| 段階 | 所有 | 実施 | 完了の根拠 |
| --- | --- | --- | --- |
| P0：静的説明スライス | #1640 + #1126、投影は#1124 | 既存CTE scope/RelationalIR→仮想関係の表・三種の線。source span、未解決ラベル | F0の構造と入力→出力属性の対応。未評価を明示し、SQLを変更しない |
| P1：fixture/期待値の一巡 | #1131 + #1640 | F0、N1〜N4、独立oracle、CTE各段階の期待/実際と差分 | 正負と0件/NULL/重複を区別。既存D2の範囲を再実装しない |
| P2：Modeler/資料連結 | #1124 + #1803 | SQL ↔ モデル ↔ 行 ↔ 設計書の一往復。編集可能元と導出部分を区別 | 同じrevisionを辿れ、変更後はstale表示。既存manifest/文書出力を活用 |
| P3：生成制約と別Target比較 | #1131 + #988/#1121、#1180はScenario再利用 | 意味型・値域制約/Constraintに基づく生成、最小の第二eval/実DB比較 | 生成の適用範囲と差異を説明できる。Lean/SMT、全SQL、各DBは先行決定しない |

#2387が静的read-only説明の一次Owner。#2387のQX初期fixture（数量0・顧客C1/C2/C3等）を最初の構造試験に使い、本書F0（数量が非NULL、二CTE・thresholdでFilter）はP1で演算・NULL比較を明確にする**補助fixture**とする。二つの独立Dataset正本を作らず#1131の同じ型付き契約へ接続する。#2386のORG-13（対象集合完全性）・ORG-21（複数経路計上）はその次の拡張例とする。P0/P1の固定fixture・紙上oracleは並行可能。P2はP0のID/出力を受け取る。P3はP1で見つかった実不足をもとに範囲決定。既存Issue/PRに所有機能があればそこへ差分を寄せ、新しいWorkBenchモノリスを作らない。

## 9. 将来の形式検証・行の由来を扱う研究参照（採用候補）

SQLの各結果行について「どの入力行が、どの演算によってこの出力を支えるか」を一般化する際、**row provenance**（由来情報）の理論が有効。Provenance semiringでは、単純な入力行のリスト以上に、複数の導出経路・重複・JOIN・集計について何を示せるかを選べる。設計ビューでは次の三種を区別する。

- **which**：寄与した入力行の和集合（軽量。ただし導出方法は失う）。
- **why**：結果を成立させる入力行の組合せ（証拠の候補を示せる）。
- **how**：導出経路と多重度を表す代数的表現（コストが高い）。「寄与の証拠」と「業務上の因果・正当性」は非同等。

公開の **provenance-lean v1.2.0**（2026-09-22、Pierre Senellart、DOI: https://doi.org/10.5281/zenodo.22900284、https://github.com/PierreSenellart/provenance-lean）は、Lean 4でsemiring由来情報、差集合、集計、HAVINGを含むモデルを形式化する。**TEI向け実装・テストデータ生成器・全SQL意味の正本ではない**ため、P3以降の比較対象とする。特に集計後の由来と、負例としてのNOT EXISTS/差集合を単純な行追跡で同一視しない。

Apache Calciteの関係代数・変換規則（https://calcite.apache.org/docs/algebra.html）は意味保存の検査観点、SQLancerのTLP/NoREC（https://github.com/sqlancer/sqlancer）は独立oracleに加えて同値変換を利用するメタモルフィック試験の参考となる。ただしTEIでのSQL書換えや各DB製品での同値を未検証のまま採用しない。

最初のReferenceでは、固定fixtureの明細2行が集計結果1行へ寄与したことを**観測/手書きoracle**で示す範囲にとどめる。一般的なwhy/how provenanceや証明書の生成、Leanライブラリの依存導入、実行時追跡の性能予算は、実際の表示価値と必要性を確認してから決める。

## 10. Owner判断と未実証

- Owner判断：このQuery Workbenchの最初の製品受入対象を「**設計書上の可読性**」に限定し、意味評価/制約生成を段階追加としてよいか。query-local fixtureを#1131 Datasetへ接続する最小契約と、手書き期待値のreview主体は誰か。NULL/空集合/0件の業務上の扱いが業務固有なら受理前に固定しない。
- 未実証：CTE evaluator、SQL変換の同値性、行lineage、編集UI、SHOへのQuery資料生成、Lean/SMT生成、別Providerの結果同値、Owner業務受理。
- **本書の採否**：現時点では候補。既存#1640/#1131/#1124/#1803の受入条件を無断変更せず、担当レビュー後に最初の実証範囲へ反映する。実装や試験を実行したとの主張はしない。
