# TEI 業務システム設計 読み方ガイド

この文書は、[`business-system-design.md`](business-system-design.md) の内容を、**業務知識やモデリング経験が十分でなくても、なぜその境界が必要なのかを追えるようにする入口**です。

業務システム設計Architectureの正本は`business-system-design.md`です。この文書は新しいRequirementやCanonical schemaを追加しません。

TEIでは、Data Model、画面、業務手順のどれか一つを詳しく作るだけでは、業務システム全体を設計したとはみなしません。一方で、三つを別々に人が管理すると、今度は同じ意味を重複して持つことになります。

この文書では、まずその問題から説明します。

## 1. 何を防ぐための設計か

たとえば「受注を登録する」業務を考えます。

Dataだけを見ると、次のようにきれいに設計できるかもしれません。

```text
Order
  ├─ customerId
  ├─ orderedAt
  └─ lines[]
```

しかし、それだけでは次のことがわかりません。

- 誰が、何をきっかけに受注登録を始めるのか
- Customerはどこから選ぶのか
- 顧客masterは参照だけなのか、更新もできるのか
- 数量が不正な場合、どこで止めるのか
- 登録後に別部署へ何を引き継ぐのか
- Application上の「登録成功」と、現実の業務上の完了は同じなのか

逆に、画面だけを詳しく設計しても、画面上の操作がどのDataや業務意味へ対応するかわからなければ、実装都合が業務設計を支配します。

TEIが三つの視点を使うのは、そのためです。

```text
                 Business
                /        \
               /          \
            Data -------- Function / Interaction
```

三つの別システムを作るのではありません。

**同じaccepted meaningを別の角度から見て、片方だけでは見つからない欠落・矛盾・思い込みを発見する**ための視点です。

## 2. 三つの視点を日常語にすると

### Data — 何について、どんな事実を持つか

Data視点では、業務で扱う事実や関係を見ます。

受注なら、たとえば次です。

- Orderは何で識別するか
- Customerとどう関係するか
- OrderLineをいくつ持てるか
- quantityにはどんな制約があるか
- 状態はどう変化するか

ここでは「どの画面componentを使うか」「PostgreSQLでどのtableにするか」はまだ中心ではありません。

### Function / Interaction — 人や外部systemはどう利用するか

Function / Interaction視点では、その意味をApplicationからどう利用するかを見ます。

受注なら、たとえば次です。

- 受注一覧を見る
- 受注を一件開く
- 顧客masterからCustomerを選ぶ
- 複数の明細を編集する
- 受注を登録する

TEIでは、`browse`、`record`、`reference`、`header-detail`、`command`等のApplication Patternがこの視点を支えます。

ただし、Web画面そのものを業務意味にはしません。同じ意味をDesktop、API、MCP等へProjectionできる余地を残します。

### Business — なぜ、誰が、どんな仕事として行うか

Business視点では、Applicationの外側も含めて仕事そのものを見ます。

- 何をきっかけに仕事が始まるか
- 誰／どの主体が責任を持つか
- 人が判断する箇所はどこか
- 電話・現物確認・外部承認等があるか
- Applicationをどこで使うか
- 次の仕事へ何を引き継ぐか
- 何をもって業務上の完了とするか

ここで重要なのは、**Applicationを使わない仕事も正当な業務意味として残すこと**です。

## 3. Business ActionとApplication Functionを分ける理由

たとえば、受注前に営業担当者が電話で顧客と納期を確認するとします。

これはBusiness Actionです。

しかし、TEI Applicationに「電話する」というFunctionが存在する必要はありません。

```text
Business Action
  顧客へ納期を確認する

実現
  人が電話する
```

これを「すべての業務行為にはApplication Functionが必要」とすると、実行可能に見せるためだけの架空FunctionやCapabilityが増えます。

逆に、一つのApplication Functionが複数のBusiness Actionから使われることもあります。

そのため、TEIは次を同一視しません。

```text
Business Action
    != Application Function
    != Capability
    != Realization
```

この分離により、「仕事として何が必要か」と「現在のsystemがどこまで支援するか」を別々にレビューできます。

## 4. 「指示した」「systemが成功した」「現実に起きた」を分ける

業務システムでは、この区別が特に重要です。

たとえば倉庫へ出荷を依頼する場合、次の四つは別の状態です。

```text
1. Business instruction
   出荷してください

2. system result
   出荷依頼APIが成功した

3. external actuality
   現物が実際に出荷された

4. Business acceptance / continuation
   出荷済みとして請求業務へ進めてよい
```

APIが成功しても、現物が出荷されたとは限りません。

現物が出荷されても、それだけで責任が自動的に次部署へ移ったとは限りません。

このためTEIでは、次を短絡しません。

```text
Business instruction
  != system result
  != external fact
  != Business acceptance
```

domain contractで同値性が明示された範囲だけを同じ意味として扱います。

## 5. Handoffは「次のStepへ進んだ」ことではない

Handoffは、単なる順序ではなく**責任ある仕事の継続関係**です。

たとえば、営業が受注内容を確定し、物流が出荷準備を開始する場合を考えます。

```text
営業側Step
  受注内容を確定する
        ↓
Business condition
  出荷可能な受注として受理されている
        ↓
物流側Step
  出荷準備を開始する
```

ここで営業と物流でactorが変わることは、Handoffを考えるヒントにはなります。しかし、actorが変わっただけでHandoffが成立するわけではありません。

同様に、画面遷移やFunction呼び出しが次へ進んでも、それだけでBusiness responsibilityが移ったとは限りません。

```text
actor changed      != Handoff proven
Step adjacency     != Handoff proven
command succeeded  != Handoff proven
```

何を条件に、どの仕事が、どこへ継続するのかをBusiness meaningとして確認します。

## 6. `unresolved`を残すことは設計放棄ではない

上流設計では、資料やヒアリングだけでは確定できない情報があります。

たとえば、

- 誰が最終承認者なのかわからない
- 外部systemの応答を業務完了と見なせるか確認できない
- 明細順序が業務identityなのか表示順なのかわからない

といった状態です。

この場合、もっともらしい値で埋めるより、`unresolved`として残す方が安全です。

```text
explicit unresolved
  = 何がわかっていないかをレビューできる

unsupported completion
  = 根拠のない仮定が設計へ混入する
```

TEIでは「空欄がないこと」よりも、known / accepted / derived / proposal / unresolvedの境界を説明できることを重視します。

## 7. Cross-view reviewは何を見つけるのか

三視点を合わせると、各視点だけでは見つけにくい問題を発見できます。

### 例1 — read-only masterなのに編集できる

```text
Data / Database
  Customer master = read-only

Function / UI
  Customerを編集できる

→ contradiction candidate
```

### 例2 — 業務で必要だがDataにない

```text
Business
  出荷可否を判断するには配送条件が必要

Data
  配送条件を保持・参照するtraceが見つからない

→ unresolved / missing meaning candidate
```

traceがないから「配送条件は不要」とは判断しません。

### 例3 — system成功を業務完了へ昇格している

```text
Function
  外部API success

Business
  実際の現物作業完了を要求

→ system resultとactualityの混同
```

### 例4 — actor名から権限を推測している

```text
Business actor
  営業担当者

Security
  update permission

→ actor labelだけではAuthorization根拠にならない
```

このようなdiagnosticは設計者を責めるためではなく、**まだ確認すべき意味を表面化するため**に使います。

## 8. MatrixやGraphは何のためにあるか

CRUD表、Matrix、Graph、Flowは、関係を高密度に見つけるために有効です。

たとえば、

```text
Business Action × Application Pattern
Business Action × Data
Procedure Step × View / Action
Data × impacted Procedure
```

をMatrixで見ると、欠落や不自然な偏りを発見しやすくなります。

しかし、Matrix自体を第二の正本にすると、人がCanonicalとMatrixの両方を更新し続けることになります。

そのため、可能な限りaccepted meaningからderived Projectionとして生成します。

Matrixの空欄も即座に「関係なし」とは解釈しません。

- 本当にnot applicableなのか
- traceがまだ無いのか
- unresolvedなのか

を必要に応じて区別します。

## 9. EditorとAIはこの設計をどう支援するか

将来のEditorでは、Data、Function、Businessを別々の画面へ完全分断するのではなく、関連する意味を局所的に追える高密度Projectionを目指します。

```text
Business Procedure
  └─ Step
      └─ Application Pattern
          ├─ Data / Relation
          ├─ Query / Capability
          ├─ Rule / Validation
          └─ Source / Evidence
```

AIも、この関係から次のような支援ができます。

- 既存の関連Data / Patternを探す
- cross-viewの矛盾候補を出す
- まだ確認されていない前提を示す
- 複数のモデリング案を比較する
- 影響範囲を示す

ただし、AIが「もっともらしい関係」を見つけたことと、その関係がaccepted Business meaningであることは別です。

```text
AI proposal != accepted meaning
similarity  != applicability
```

最終的にはShared Acceptance Boundaryを通してauthoritative sourceへ戻します。

### AIが設計変更へ入るときの読み順

AIが設計変更を始めるときは、通常のKnowledge Contextとは別に、**設計確認計画（Builder Review Plan）**を取得できます。

```text
tei knowledge-review-plan
  -workspace <workspace>
  -subject-kind <kind>
  -subject-id <id>
```

このProjectionは、個々の設計原則を重複して持ちません。`routingScope = current-knowledge-context`の範囲で、`recommendedDisclosureSections`に現在利用できる`source`、`definition`、`data-design`、`dependencies`のうち何を確認すべきかを示し、`reviewSequence`で確認順を示します。推奨された区画は、そのまま適用可能・Qualification済み・開示済みのKnowledgeにはなりません。必要なQualificationを経てから設計判断へ使います。Data設計の詳細な確認事項は、従来どおり`dataDesign.designReviewProtocol`が担当します。

Review Planで読む候補を得た後、設計変更のtask scopeが明示できる場合は、既存のKnowledge Qualificationを通した**Builder Review Handoff**を作れます。

```text
tei knowledge-review-handoff
  -workspace <workspace>
  -subject-kind <kind>
  -subject-id <id>
  -qualification-requirements <requirements.json>
```

HandoffはKnowledge本文を含みません。Review Planのrevision / fingerprint、routing scope、選択候補となる区画、明示したQualification requirements、Qualification結果とそのSource Evidenceを結びます。Qualificationが`applicable`で、authoritative Evidenceと必要なdependency Evidenceが揃い、ContextとSource Indexの候補subject Evidenceが同一revisionである場合だけ`qualified-for-disclosure`になります。`unknown`は`blocked-unresolved`、`incompatible`は`blocked-incompatible`として保持します。

```text
available
  -> Builder Review Planでselected候補を決める
  -> explicit requirementsでqualifiedを確認する
  -> Builder Review Handoff
  -> current Knowledgeからdiscloseする
```

Handoff自体はDisclosureではありません。Handoffのfingerprintも開示内容のfreshnessを証明しないため、実際に開示したKnowledgeの再利用にはDisclosure側のfingerprintを使います。

Qualification済みのHandoffから現在のKnowledgeを開示するときは、Handoff JSONを入力にして`knowledge-review-disclose`を使えます。

```text
tei knowledge-review-disclose
  -workspace <workspace>
  -subject-kind <kind>
  -subject-id <id>
  -review-handoff <handoff.json>
```

このコマンドは、Handoff自身のfingerprintを検査し、同じrequirementsからcurrent Context / Source Index上のHandoffを再構成します。fingerprintが一致し、statusが`qualified-for-disclosure`の場合だけ、既存の`DisclosureProjection`を返します。stale、改ざん済み、subject不一致、blockedなHandoffからは開示しません。

返すJSONは通常のDisclosureと同じです。HandoffやQualification結果を本文へ混ぜないため、同じKnowledgeを別taskへ再利用するときは、そのtaskについて改めてQualificationします。また、blockedでも診断や根拠確認のためにKnowledgeを読むこと自体を禁止するものではありません。禁止するのは、未QualificationのKnowledgeをそのtaskへ適用可能な意味として扱うことです。

Review PlanはAI Builder向けの用途別Projectionであり、通常のKnowledge Context、人間向け表示、業務実行用Contextへ混入させません。また、現在のKnowledge Contextに区画が見つからないことを、Repositoryや外部Knowledge全体に情報が存在しない証拠にはしません。Review Planのfingerprintが一致しても、選択したKnowledgeの内容が最新であることまでは証明しないため、Knowledgeの再利用には対応するData Design / Disclosure等のfingerprintを別に確認します。

```text
Builder Review Plan
  -> 何をどの順で読むか
  -> 必要なQualification境界を通す

dataDesign.designReviewProtocol
  -> Data設計で何を確認するか

どちらも
  != accepted business meaning
  != semantic applicability
  != write / runtime authority
```

### AIがData設計を変更するときの入口

Data、Relation、Validation、Persistence、record系Patternに関わる変更をAIが検討するときは、設計資料を自由検索するだけでなく、対象DataのKnowledge Contextを先に取得します。

```text
tei knowledge-context
  -workspace <workspace>
  -subject-kind semantic-data
  -subject-id <data-id>
  -disclose data-design
```

`data-design`には、現在のData / Relation / Constraint / Persistenceの対応関係だけでなく、識別子、候補キー、関数従属性、複合的な値制約、履歴・有効期間について確認すべき問いと、誤って意味を確定しないための制約が含まれます。

さらに、`designReviewProtocol`が設計レビューの順序を機械可読な形で返します。AIは少なくとも、次の順序を崩しません。

1. 対象Dataと、その根拠となる現在のSource / Evidenceを特定する。
2. 業務上の同一性を、Databaseの主キー、画面やRuntimeのIDから分離する。
3. 識別子と候補キーについて、重複不可の範囲、時点・期間、同値性の扱いをEvidenceから確認する。
4. surrogate IDを採用する場合も、業務上必要な一意性を消さない。
5. 関数従属性は業務上の意味・規約として確認し、現在のデータでたまたま一意であることや統計的な相関から確定しない。
6. 関数従属性と、単なる複合的な値制約を分ける。
7. Relation key、局所的な一意制約、Persistenceの主キー・UNIQUEを、業務上の識別子や関数従属性へ自動昇格しない。
8. 根拠が足りなければ`unresolved`を残す。
9. 導出した識別子・従属性候補はProposalとして扱い、Shared Acceptanceを経ずにaccepted meaningへ変えない。

この手順自体もCanonicalな業務意味ではありません。現在のKnowledgeから導いた設計支援の手順であり、`authorityLimits`でもその境界を明示します。Knowledge Contextのfingerprintが変わった場合は、古い文脈をそのまま再利用せず、現在のData Design Contextを取り直します。

## 10. 一つの業務例を縦に追う

受注登録から出荷準備までを簡略化すると、三視点は次のように重なります。

```text
Business
  顧客から注文を受ける
  ↓
  内容を確認する
  ↓
  出荷可能なら物流へ引き継ぐ

Data
  Customer
  Order
  OrderLine
  shipping condition

Function / Interaction
  customer reference
  order record
  order lines header-detail
  submit command
```

ここから、たとえば次を確認します。

- Customerはread-only参照でよいか
- OrderLineの数量制約は何か
- 出荷可能性の判断に必要なDataは揃っているか
- `submit`成功と物流へのHandoffを同一視していないか
- 人による確認作業を架空Functionへ変換していないか

このように、一つの視点を完成させてから次へ進むのではなく、**同じ業務意味を往復しながら不足を発見する**のがTEIのBusiness System Designです。

## 11. 読む順序

初めてこの領域を読む場合は、次の順を推奨します。

1. 本書 — 問題と三視点を具体例から理解する
2. [`requirements/README.md`](../requirements/README.md) — Product Requirement全体の中での位置を確認する
3. [`requirements/BUSINESS_SYSTEM_DESIGN_REQUIREMENTS.md`](../requirements/BUSINESS_SYSTEM_DESIGN_REQUIREMENTS.md) — RQ-025 / RQ-026の解釈を確認する
4. [`business-system-design.md`](business-system-design.md) — Architecture境界を詳細に読む
5. [`../plan/BUSINESS_PROCEDURE_DESIGN_QUALITY_MODEL_2026-09.md`](../plan/BUSINESS_PROCEDURE_DESIGN_QUALITY_MODEL_2026-09.md) — Procedure reviewの具体的な品質観点へ進む
6. 対応するReference / Plan Issue / Evidence — 実際にどこまで確認されたかを見る

## 12. このガイドから一般化しないこと

この文書の例は説明用です。

次を意味しません。

- `Procedure`を必ずtop-level Canonical objectにする
- Handoff schemaが確定済みである
- 三視点を同型のobjectとして保存する
- Matrixを必ず永続化する
- 受注domainがTEIの標準domainである
- AI diagnosticをhard errorへ自動昇格する

表現方式はReference EvidenceとADRで判断します。
