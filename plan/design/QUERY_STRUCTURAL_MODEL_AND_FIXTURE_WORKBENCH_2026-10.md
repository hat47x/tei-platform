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
- Python 3.13.5：`test_probe_relational_ir.py` **13件PASS**。`probe_relational_ir.py`は3例をJSONに投影し、Root構造候補と各ノードのSource operation ID・grain候補・unverified式を出力。
- GitHubに保存した3ファイルとローカルで実行したファイルの**Git blob SHA一致**：`probe_relational_ir.py = 0166c89dc5b85d6ff721a959a2f242e82b50c770`、`relational_ir_synthetic.json = 9a47ab095e504798726976239dbf2c56dd123ce5`、`test_probe_relational_ir.py = 1edcce76f59982055603134eaaca7b6955c1a196`。
- Go 1.23.2の隔離テストでは、現mainの`relational.go`と**バイト同一の型定義**を使用し、3種類の合成IRの`encoding/json.Unmarshal/Marshal`往復1件PASS。これは*Go型とJSON shapeの限定照合*。型ファイルをRepoへ複製・導入していない。repo指定Go toolchainでの正式`go test`や`DeriveFromAnalysis`、worker/実Provider/GUI/SHOは未実行。

**構造を読めることとClaimが成立することの間の欠落**：
1. `RelNode.Inputs`は演算Node ID、`OutputField.Inputs`はColumnOccurrence IDに由来するField Lineage。**出力Field→どのRelNodeの演算・GROUP/WHERE条件で成立したか**を現IRだけで自動認定する追加根拠は未確認。2種類のIDを短絡して結ばない。
2. `RelNode.Expressions`は文字列配列。SQLの比較や`EXISTS`のPredicate、`SUM`の対象/NULL・非負値域型、Window/Recursive、SQL dialectを型付きで十分説明したと推定しない。プローブの`expressionSemantics`は`unverified`、`claimConclusionPresence`は常に`not-established`。
3. `RelNode.Resolution`や`Coverage.Full`が仮に存在しても、Provider read-set、必要なsnapshotの一致、Ownerの業務Meaning/受理、Host実行・開示許可の証拠にはならない。source scopeの判定はDraft #2454、worker応答ID/版はDraft #2471、同一Query IDのSQL原byte変化はQX-14の未解決課題。

**次のM1判定方法（未実行）**：既存Queryパッケージの合成`analyzer.Result`から`DeriveFromAnalysis`→`RelationalIR`を**実際のGo呼出しで生成した結果**を、今回のQX-17プローブへ入力。3例の「Scope→Source occurrence→Operation Inputs→OutputFields」を別々に照合し、(a)同じ依存を辿れる、(b)構造は辿れるが出力Field接合は無根拠、(c)未表現／`opaque`、(d)誤った`Full`の4群へ分類する。現在のIRで説明の成立条件が表せない場合は、まず具体的な不足する項目を記録し、第二のParser/SQL Evaluator/新Core Provenance正本を作る前に既存Owner #2387/#988/#2506で接続を評価する。

**同時に満たすべき横断要件**：#2506のData Context Instance/選択epochと#2500のAction対象IDは、本QX-17 Source operation IDと別のID。部分取得・JOINの行増幅・集約の操作不可をModeler/SHO/外部Consumerへ説明する際は、構造観測→対象束縛→現在性→Authority/Effectをそれぞれ独立に扱う。QX-17はread-only構造候補のStageに留め、業務完成・Commit・GenerationのPASSへ昇格しない。


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
