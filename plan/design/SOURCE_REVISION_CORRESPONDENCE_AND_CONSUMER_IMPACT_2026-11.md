# Source改版追跡とConsumer別の再適用・再生成（設計候補）

> **状態：提案／未採用。** #2505の一版内SourceRefとA/B版間の対応候補グラフを出発点とする。実原モデルの再検収、全Target/SHO/GUI、Owner意味受理、実DB/Host実行は未確認。2026-11-02時点。

## 目的と既存Owner

一体モデルからLibrary・設計書・Function UIを生成するとき、原Sourceが改版されても、失われていない根拠や歴史的成果物を保持し、**影響する利用先だけを再評価する候補を提示**する。Source形式版の変換は#1920、単一版の原本/Probe/Candidate/Ledger/Evidenceの検収は#1121、意味の受理は#267、部分取込みは#1805、生成物の導出は#2482、物理公開は#2604、実Operationは#2598/#1174が所有。#2505の比較器は読み取り専用のProjectionであり、第二の意味正本・受理Engine・Host・第三の移植検収器を構成しない。

関連する#2582（Claim相対の変化伝播）、#2586（Query→Context→Claim→Pattern→Intent）、#2516（用途相対開示）、#2391（GUI/Computer Use）と同じ試験対象を複製せず、**版差分から何を次のOwnerへ渡すか**を扱う。

## 五つの異なる「同一性」

| レイヤー | 比較対象と粒度 | 同値でも未保証なこと |
| --- | --- | --- |
| S: 原本packet | 原ZIP実byte、member、selected view、版内node/atom SourceRef、Probe版 | 版間Business意味、Candidate全件移植 |
| C: 意味対応候補 | A/Bのnode/role/field/component間の理由付き候補辺・多対多の対応グループ | 意味同値、OwnerのAccepted Decision、旧操作の再利用 |
| M: Accepted Meaning | 受理済みsubject/role/Ruleのrevision・範囲・未解決 | 特定Target上の生成物一致、実DB効果 |
| G: 生成・導出 | 対象Target、Generator/Template/Rule版、入力版、実出力byteとManifest | 出力集合の一括公開・同一世代Consumer読取り |
| P: 公開・消費 | 完成した一式を消費者が実際に読んだbyte、Manifestと世代、時点、開示scope | 業務上の実行資格、Owner完了 |

### 版間グラフ（read-only）

各固定版のSourceNode集合をS_A, S_B, S_C、理由付き候補辺をE_AB ⊆ S_A×S_B、E_BC ⊆ S_B×S_Cとする。|E|、|dom(E)|、|ran(E)|、旧側fan-out、新側fan-in、双方次数1の辺を**別々に計数**する。いずれも受理済み意味の個数ではない。

E_AB∘E_BCは**中間版を通る候補経路**であり、A→Cの意味同一性を含意しない。Aの2要素→Bの1要素→Cの2要素なら、経路は複数残るが、確定した2件の一対一対応とは言えない。経由版・対応理由・局所的に欠落した関係を保持する。SourceRefはview全体digestと構造位置による**版内位置参照**であり、永続Business IDを創作しない。複数のXML成分を一つのatom処遇へ縮約せず、Relation一件と実装Refer候補複数も別の母集団とする。

## Claim別の現在適用性：再評価の最小条件

更新後にどのClaimを利用できるかは、元Evidenceの歴史的実在、現在の出典再解決、Claimが依存する意味・構造、取得Scope/Predicate/時間、Owner Rule、Consumer開示資格、実際の読取byteを区別して判断する。

- **値**：ある属性の変更は、その属性の値を使うClaimを再確認候補とする。変更未観測の周辺Claimでも、依存関係が未知なら「無影響」を確定しない。
- **存在・否定**：既知の一件の存在証人と、全件取得に基づくNOT EXISTS/「変更なし」を区別。Providerの部分取得・RLS不可視を空の完全集合に変換しない。
- **関係・母集団**：既存IDの現在値が同じでも、対象Relationの加入→離脱（ABA）や遅着訂正があれば確認履歴の区間Claimを再評価。現行Snapshotの一致を過去の全区間へ外挿しない。
- **集計**：SUM/AVG/Distinct、JOINのbag多重度、Top-N、LEFT JOINの行成立は入力Collection・順位・スコープへの依存が異なる。D1完全/D2部分ならD1単独SUMを局所保持候補にできても、D1×D2を全件確定とは言えない。
- **文書**：SHO本文、出典参照先、実表示byte・マクロ・テンプレート、著者Review、固定納品版を別の版系列とする。当時のレビューは歴史として保持し、現在の説明への適用性だけ再評価する。
- **外部Effect**：Source対応が一意でも旧ViewChange→Action→RequestのContext Instance/epoch・対象・Binding時点・Host権限・Receiptは再照合が必要。#2598/#1174の担当。再Submitの許可を差分CLIは与えない。

## 増分再生成と物理公開は別工程

ある変更が文書Templateのみなら、設計書再生成の候補とし、Accepted Modelと外部Libraryの業務意味改版を推測しない。逆にAccepted Meaningを変えても、影響しない生成Targetまで一括強制再生成することが正しさの証拠にはならない。変更依存が未知なら局所無影響と断定せず、再検証対象を保持する。

#2604が示した通り、**ManifestのDigest一致／Complete／生成器のexit 0**と、公開ディレクトリ内の複数成果物が同一世代であること、Consumerがその実byteを読んだことは別。旧一式R1の後にR2の一部ファイルだけが更新される途中失敗、旧ファイルの残存、出力名衝突、Manifestと実ファイルの読取時点差を、出典改版の差分判定から独立して観測する。対象の実験と修正は#2604へ渡す。

## 反証計画（合成fixtureを基準にする）

| ID | 対照 | 期待する境界／Owner |
| --- | --- | --- |
| RV-19 | Aで2要素→Bで1候補→Cで2候補 | 経路の列挙は可、A→C意味同値/Owner受理は不可。#2505 |
| RV-20 | D1完全/D2部分、JOIN/NOT EXISTS/順位演算を変更 | Claim別の必要集合と再評価。#2582/#2586 |
| RV-21 | 現在値・集合一致、履歴だけABAまたは遅着 | Current snapshotとinterval confirmationの区別。#1125/#2582 |
| RV-22 | SHO表示Asset/数式Macroのみ変更、固定納品版保存 | Occurrence別の実表示依存と現在Review。SHO/MMI/#2516 |
| RV-23 | R1一式→R2一部だけ公開→Consumerが混在読取り | Source差分はわかっても実納品PASSとは言えない。実行Owner #2604 |
| RV-24 | 旧Source locatorを新版へ対応候補提示→旧ViewChange再利用 | 候補同一性と最新版Action/Request/current Grantは別。#2500/#2598 |
| RV-25 | 内部のみ差がある2世界、集計/診断を同Consumerへ出力 | 開示決定と件数/指紋・理由文による漏出を検査。#2516/#941 |
| RV-26 | 意味同一・Generator/Template変更、または逆 | Target別再生成、Manifest公開、実Consumer読取を分離。#2482/#2604/#2077 |

RV-19の純粋Go helperはA→B→Cの候補経路・経由ノード・未到達・曖昧性を検査する**独立合成実装**として試験可能。他のRV-20〜26は各担当の既存fixture/反例を参照し、Source候補比較器にBusiness Interpreter/DB/SHO Rendererを重複実装しない。

## 証拠段階・着手順

1. **S0 P0**：H1/H2を別の固定packetとして、原ZIP・Probe・Candidate・正式Ledger・実Target・EvidenceをJ5一行で接合。既存private 2 checkerを使う。byte一致、完全性、Builder因果、Owner受理を別欄にする。未実施。
2. **P1 Source差分**：#2505の既存合成CLIを隔離ブランチへ取り込み、追加RV-19とともに製品Go 1.26.10でfocused test。特定版・public-private開示条件・未実行項目をREADMEとDraft PRへ固定。
3. **P1 Consumer統合**：#2586の一Function同一版E2E、#2582変更前後再評価、#2604 Artifact公開の同一版読取、#2516とSHOの根拠開示を各Owner試験で接合。本文の計画を実装完了の主張に使わない。
4. **P2 複数Target**：Java/.NET/WasmとSHO設計成果物について、同じAccepted Meaning revisionから異なる出力へ派生する因果・実消費を個別検証する。

検収や意味受理の未実施を隠さず、局所合成結果やAI提案から移植全件PASS／実業務完了へ昇格させない。公開Issueへ非公開原モデルの個別ID/locator/Raw SHA、原本文を転記しない（#941）。
