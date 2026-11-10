# TEI モデル由来テストデータ生成・検証・開発支援（採用前設計）

**位置づけ**：2026-11-07の横断要求を既存計画へ渡すためのDraft。提案段階であり、Product API・強属性の原典定義・新Test Runner/DSL/SSOT・Ownerの業務意味受理を宣言しない。主な関連担当は Data #1126/#1806/#1705、Function #2506/#2586、Business #1125、テスト #2514/#1180/#3223、証拠 #1163、広域Snapshot #1708、Consumer #2516。

参考（公開説明資料）：[三要素分析法のデータモデル例とインスタンス併記（2006年、ITmedia/＠IT）](https://www.itmedia.co.jp/im/articles/0605/09/news085_2.html)は、データの例をモデルと並べて業務理解を促す方法を紹介している。TEIではこれを単なるサンプル表示にとどめず、**意味的に受理されたインスタンス例と故意の違反例を、テストに使える独立期待値付きDataset/Scenario候補へ発展**させる。ただし同記事は本計画の「強属性」の厳密定義や新機能の原典根拠ではない。TEIは業務外Domainにも適用するため、三視点の具体的モデルはPlugin/Projectionで拡張可能とし、業務向け固有概念をCoreへ強制しない。

## 1. 製品価値と三要素の相互牽制

TEIのテスト支援は単純な型に合う乱数やDB行を作るのではなく、**モデルから「どの不変条件を、どんな反例で確かめるか」を根拠付きで提案し、安全な隔離データと検査へ変換する**。Data・Function・Businessはそれぞれの正本/Projectionから同じMeaning IDを参照し、相互に不足・矛盾を露出する。

| 入力視点 | 知識の種類 | 提案できるObligation | 自動決定してはいけないもの |
| --- | --- | --- | --- |
| Data | 意味型、値域、単位、物理型、業務キー、候補キー、関数従属性、Relation多重度・必須性・有効期間、履歴 | 境界値、必須/NULL、参照先欠落、同一業務キー多重、同長異ID体系、JOIN増幅、期間重複 | 業務キー=物理PK、同名/同桁=同じ意味、NULL=未取得/非開示 |
| Function | I/O、Query、必要入力・出力、Predicate、Data Context、対象集合、表現型、保存Hook | 入力組合せ、非出力依存、部分/全件、取得失敗、更新Intentと保存後read-back | Query定義=実行、表示行数=対象全件、Output非開示=内部未評価 |
| Business | 業務Task/Role/Actor、業務ルール、先行/例外/手作業/承認/差戻し、Procedure/Handoff | 不正遷移、未承認の変更、兼務・代理失効、競合・取消・再送、手作業未完了 | DFDの矢印=Workflow自動実行、業務Actor=Host Grant、Function成功=業務完了 |
| 横断 | Scope/Tenant、権限、Snapshot・時点・Revision、Provider差、Consumer開示 | 二Tenant同名鍵、古い応答混入、複数Readの非原子性、非表示Predicateからの情報推知、Provider移行 | 一つのRead時刻=同Snapshot、成功Receipt=Resource確定、表示値の秘匿=結果の非干渉 |

**強属性の扱い**：原典における「強属性」の正確な定義は未照合。現段階では独立のTEI Core概念として確定せず、Ownerが受理した属性の識別・決定関係・値の従属性などがある場合に、検証候補へ接続する。候補キー/物理PK/関数従属性へ無条件に同一視しない。原典確認時に Definition と Provenance を追記する。

## 2. 生成支援の入力契約と資格

1. **Meaning reference**：Source path/span、World/Model/Accepted Meaning Revision、Data/Function/Business Subject ID、Relation occurrence/Role、Authority/Owner、適用scope、valid-timeを識別する。
2. **制約と根拠**：Rule/Value Domain、nullable/missing、Precision/Scale/Rounding、Predicate/Input Dependency、状態遷移前提、関数従属性、関係多重度、Provider固有制約とそのSource/Acceptanceを保存。推測は `candidate`、既存Ownerが意味を受理したものだけ `accepted` として別分類する。
3. **目的と範囲**：検証対象Claim、対象集合 L、grain、許される操作、Fixture隔離境界、Consumer/Audience/Disclosure、時点保証、テストTarget/Provider、変異させる因子を宣言する。
4. **十分性の区分**：`required` / `known` / `unknown` / `unavailable` / `not-disclosed` / `conflicting` を説明上区別し、NULL/空集合/0/部分取得へ収斂させない。内部ツールのStatus enum追加は後段で要否判断する。
5. **データ生成の安全条件**：個人情報・実顧客データの無断複写はしない。原則、Synthetic ID・専用test database・既存表非存在・read/write権限の限定・テスト後cleanup。現行Host/Providerの許可なしで本番書込みをしない。

### 生成候補の分類

- **満足例（positive）**：受理した必須制約を満たす最小集合。どの値域・Rule・Roleを満たしたか説明する。
- **単一制約違反（negative）**：一回に一つの条件を崩し、他条件は可能な限り維持する。例えば業務キー同一で同Tenantの明細番号だけ重複。独立Expectedは明示拒否/失敗。
- **相互作用/境界**：二つ以上の制約が独立でないとき、pairwise、状態遷移前後、時間・権限・Tenantの組合せを選び、単純組合せ数増加を抑制する。
- **識別不能/観測不足例**：同じ公開Rowsetを返す異なる内部Worldに、対象Claim真偽の違いを持たせる。証明不能なら`unverified`。逆向きの公開値が変わる正例も持ち、常時拒否・常時unknownの実装も落とす。
- **メタモルフィック対照**：対象として許された変換（順序だけ変更、View変更、別Target実現等）で保持すべき関係と、保持してはならない資格（Scope/Grant/epoch/丸め違い）を分離する。
- **実測との差分からの最小化**：CI/Provider failureを独立Expectedへ突合し、失敗再現に必要な対象・Rule・入力の集合を縮小する。Shrinkは本来の違反因子を消してはならない。

## 3. 独立Expectedを循環させない設計

生成AIの同じ推測を (A)実装、(B)テストデータ、(C)期待結果の三者で共有すると「誤った仕様の一致」を起こす。**生成Candidateが提案したExpectedだけで合格を認めない**。

- 独立基線はOwner受理されたBusiness Rule、既存固定Golden Scenario、手で固定した有限表、必要なら別の実測/構造根拠を使う。生成モデル由来の仮説は `candidate expected` に留める。
- `Generator`が出すのは Dataset/Scenario/Obligation の**採否候補**。正式Scenario保存・再生成は #1180、判別力とKnown Mutantは #2514、実行AttemptとReceiptは #1163 が引き受ける。専用第二Expected Store/Engineは作らない。
- Source/Rule Revision、生成手段/Seed、Fixture、Expectedの出所、Test Target/Target Version、実実行のSQL+bind+Provider/Reader、Diff/診断を混同しない。
- 誤実装 Mutant は source/target の範囲を最小化し、元の固定Expectedを変えずに検出したかで判定。Redの理由がbuild/CI/秘密・Host拒否ならMutant killに数えない。
- モデルに矛盾があり制約充足データを生成できないときは、勝手に条件を緩めるのでなく **矛盾する制約の最小集合・出典・受理者・対象時点**をレビューへ返す。
- Model改訂・Plugin/Operator改訂・Owner受理撤回・同bytes別意味・A→B→Aでは、旧Case/Expected/Receiptを無条件再資格化しない。

## 4. 初期受入コーパス：受注を縦断した有限ケース

`order_entry`の検証用公開データを第一のスライスとする。下記は**採用前の独立Expected候補**であり、業務Ownerが規則を確認するまでは製品意味を断言しない。

| Case | モデル由来の検証義務 | 最小Fixture/操作 | Expected候補・証拠水準 |
| --- | --- | --- | --- |
| MDT-01 | 値域の上下界/NULL/未取得 | 数量 0/境界/超過、required入力欠落、明示NULL | 許容域内は候補成立、域外/禁止NULL拒否。Value Domainの受理必須 |
| MDT-02 | 数量・通貨・精度 | 個×JPY/個、kg×JPY/個、JPY+USD、tie丸め | 一致計算と単位拒否を分ける。#1705の意味契約を再利用 |
| MDT-03 | 業務キーと物理PKの区別 | Tenant A/Bで同じorder番号、同Tenantで重複明細番号 | Claimに応じて許容/拒否を変える。強属性を推定しない |
| MDT-04 | 多重度・関数従属性 | 注文H→D1=2件、D2=3件、JOIN=6行 | JOIN6≠実体5件、表示行数/集約から業務対象を同定しない |
| MDT-05 | Context全件性 | L={A/3,A/9}、Read R=[A/3,A/3] | 行数一致でもunqualified。#2514 Q05 |
| MDT-06 | ページと対象集合 | 同L、Read top1=[A/3] | 局所Partial→unverified、全件と自己申告しても業務Claim不適格。#2514 Q06 |
| MDT-07 | OutputとPredicate | Selected=[orderId]、status=READY実評価証拠なし | Holds自己申告を資格化しない。#2514 Q07 |
| MDT-08 | 物理NULLのlineage | Raw `NULL AS status`、格納値`READY` | Projection placeholder nil≠DB値NULL。#2514 Q08 |
| MDT-09 | 非出力列の実演算 | SQL WHERE ordered_at>=from、出力IDのみ | SQL+bind/Scope/固定Fixtureで判別。#2514 Q09、実PGは限定実証済み |
| MDT-10 | 結果による情報漏えい | private statusだけ異なる2世界、ConsumerはorderIdのみ | 禁止Audienceへ件数・存在・順序の違いを送らない。実Egress NOT RUN |
| MDT-11 | 業務フロー・Actor | 受付→人手確認→承認/差戻し→出荷、代理失効 | Function success≠手作業Handoff確認≠Business completion |
| MDT-12 | 状態と有効期間 | 承認前後の修正、遡及訂正、取消後の出荷Attempt | 許可された状態遷移と歴史・現行状態を別根拠で判定 |
| MDT-13 | 2 Providerの読取整合 | Aをrev2、Bをrev1、同時刻風timestamp | same-time表示から同Snapshot不可。要求保証未達ならunverified/unsupported |
| MDT-14 | 再試行と部分Effect | commit ACK lost→同OperationID lookup→独立Read | 新OperationIDの盲再送不可。ReceiptとResource/Businessを分離 |
| MDT-15 | CUI/表現型間の意味保存 | 同ContextをForm/Table/Treeに投影、epoch A→B→A | 値・Subject/Grain/Scopeの同一性だけ保持。過去応答を今の選択へ混入させない |
| MDT-16 | ターゲット・方言差 | Go / Java / PostgreSQLでNULL、数値、丸め、拒否を照合 | 必要同等性が未証明ならTarget固有PASSを全Targetへ広げない |

**現行実績**：#3238の限定Web ReferenceではMDT-05〜09の一部が合成Go/実PG16.15の固定Fixtureで観測済み。ただし完了ClaimやProduct Function→Query→Provider→Context→Consumerまで実装・受理した主張ではない。#3238最新HEAD Healthは結果を別確認する。その他のCaseは設計候補であり今回実行していない。

## 5. 開発者向け支援体験（設計候補）

**既存CLI/CUIとModelerを起点とする**。以下はUIの操作意図であり、新CLIサブコマンド・公開APIの決定ではない。

1. **Inspect**：「この機能/データ/業務手順が依存するAccepted Meaning・値域・Rule・Source span・Consumer・Revisionを表示」。未読込・候補・不明点を一覧化する。
2. **Propose obligations**：「受注確定のため何を試すべきか」を三視点で列挙。型・Relation・Workflow・Scope・Snapshot・Disclosureから必要条件を抽出し、元Source・Owner・Claim Ceiling付きで提案。未採用規則は独立に保留。
3. **Preview datasets**：正常・単一違反・境界・二世界・状態履歴・故障注入の最小Fixtureを表示。関係グラフと外部キー順の作成/削除順・触るProvider/DBを示す。実DB実行前に危険な表/本番接続を拒否。
4. **Review/accept**：生成AIが作ったFixture/Rule/Expectedを別々にレビュー。Ownerが業務意味を受理するまではテスト自体が通っても `unqualified`。Accepted Scenarioの版と変更理由を保持。
5. **Execute through existing Driver Scenario**：#1180からCLI/Web/Desktopの実行計画を派生し、安全な隔離Providerに適用。今のHost Grant・Read/Write Effect・実行結果/独立Observationを分ける。
6. **Diagnose and repair proposal**：単なるassertion差分でなく「どのRuleの、どの対象Occurrence/Field/Version/Scope/Stepが矛盾したか」を説明。修正案をRule、データ、Function Binding、Predicate、Workflow、テスト基線のどこに適用するか示し、受理前に自動更新しない。
7. **Change impact**：Source/Value Domain/Rule/Relation/Provider版の変更時、影響を受けるScenarioと検証義務だけを再候補化。未探索/未取得の範囲は「影響なし」にしない。
8. **Coverage explain**：ケース数・コード行カバレッジだけでなく、要求Obligationのうち `required/supplied/executed/observed/qualified` まで満たした範囲と、未検出/未許可/省略の残余を表示する。

## 6. 実装順序と既存Issueへの責務分配

| 段階 | 最小導入内容 | 採否に使う観測 | 既存責務 |
| --- | --- | --- | --- |
| P0-A | 三視点のAccepted値域・Key/Relation・Function I/O・Business Stepを**read-only**で結ぶ Obligation Candidateの一覧 | 各候補にSource/Meaning/Owner/Status/Claim、欠測が付く。推論だけでAcceptedへ昇格しない | #1124/#1126/#1806/#1125/#2506 |
| P0-B | `order_entry`の1 Function/1 L/P、MDT-01/03/05/06/07/09の**固定有限Fixtureと負例** | 候補データが必要制約を満たす/一因子だけ違反する、誤モデル/常時PASS Mutantを識別 | #2514/#1180、Providerは#2586 |
| P0-C | 実RunnerなしのTest Plan Preview→既存Scenario適用→証拠受領 | Seed/Expected/Provider/Revision/DB safety/Scope/CI Runが一致。実PG SKIPは未実行 | #1180/#1163/#3223 |
| P1 | 状態遷移・人手Handoff・競合・複数子集合・広域Snapshot | 違法Step、A→B→A、JOIN増幅、非原子Read、再送を検出し、必要な状態変化を保持 | #1125/#1708/#2506/#1174 |
| P2 | 複数Target・Plugin/Operator版・Consumer漏えい・再資格化、AI提案修正 | 別実現の丸め/NULL差、non-output leakage、同bytes別Meaning、新旧Expected資格を区別 | #1705/#1694/#2516/#267 |

**受入基準**：①各候補Obligationの出典・Scope・受理状態が追跡可能、②正常データ・1因子違反・不足証拠の三類が独立Expectedと区別できる、③不正Rule/強属性誤解/値域誤統合を自動確定しない、④既存Scenario/Provider/Hostを再利用、⑤最小反例の再現にSource版とSeedと隔離DB条件がある、⑥業務Owner受理と実Target資格を別記録、⑦定義矛盾時に制約緩和で隠さず矛盾候補を提示。

**現在の状態**：この文書の保存とIssue引渡しは設計作業。新たなGenerator/Obligation Extractor、Modeler UI、実Provider E2E、強属性の原典確認、Owner Meaning採用、採用済みDriver Scenarioへの新Case統合は**NOT RUN／未実装**。既存#3238の限定実PG証拠をProduct E2へ昇格させない。