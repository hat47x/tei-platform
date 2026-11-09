# サブシステム分割の提案・評価：更新凝集と越境CUD最小化（設計候補）

- 日付: 2026-11-07
- 状態: **設計候補／未採用／未実装／新しい試験 NOT RUN**
- 担当Issue: [#3250](https://github.com/hat47x/tei-platform/issues/3250)
- 既存Reference: [#1132](https://github.com/hat47x/tei-platform/issues/1132)（read-only CUD診断、合成Referenceのみ）
- 目的: 複雑な業務系モデルを、更新責任が明瞭な論理サブシステムへ分割し、全体CRUD表とサブシステム別ER図で設計・変更の見通しを改善する。設計候補の生成と受理済み構造は分ける。

## 1. 原則と非目標

**第一原則**：システム全体のCRUD関係から、サブシステム内部の更新系凝集度を高め、他サブシステムが所有するデータへの**直接C/U/Dを原則ゼロ**に近づける。ただし機械的なゼロが妥当な業務境界を証明しない。外部Readは通常許されるが、認可・開示・完全性は別契約である。

サブシステムはまず**論理的な設計単位**であり、DB Schema、マイクロサービス、Build Unit、プロセス、テナント、組織部門のいずれとも1:1としない。単一Go BinaryもJava/.NETも、必要時の分散Targetも同一論理分割を利用できる。物理配置は独立したLowering/Build/Provider判断とする。

以下を別の軸に保持する。

| 軸 | 意味と責務 | 混同してはいけないもの |
| --- | --- | --- |
| Logical Subsystem | 設計上の機能・Data・Businessのまとまり | プロセス/Host/Build/組織部門 |
| Data ownership | 受理済みDataに対する変更責任 | 名前の一致・物理Tableの更新権限 |
| Direct CRUD | 宣言・静的抽出した直接Read/Write | 実際の適用・他Functionへの間接効果 |
| Effective effect | 明示的な呼出し・Rule・Trigger・Eventが起こし得る変更 | 実Commit・独立Readで確認した状態 |
| Transaction / Recovery | 期待状態、一貫性、拒否、部分成功、回復 | 直接CUDゼロという構造指標 |
| Deployment / Build | Targetでの配置/接続/Artifact | 論理モデルの一意の実現 |

**非目標**：旧形式の機械的移植、サービス分割の強制、DB migration実装、第二のSemantic Canonical、第二のGraph/ID/Acceptance、認可Engine、万能Generator/Scheduler/Host、実Provider effectの推測。公開外の参照モデル本文をコピーしない。

## 2. 標準構造とSource/Acceptance

TEIの既存標準Data / Relation / Graph / Collection / Constraint / typed Portを最初に利用する。必要な業務意味はPluginのrole / facet / relation contractとして載せ、Coreへ先に専用enumを追加しない。Subsystem・Membership・Ownership・Access・BoundaryPolicy・PartitionProposalは**候補となるrole名**であり、新Core型の確定ではない。

出発点はScript/DSL・Markdown・標準構造などのauthoritative Source。Data / Function / Business、ER図/CRUD表/Subsystem図は同一SourceのRole別Projection。編集可能な表・図もSourceへ戻せない部分は表示状態のままとし、変換が曖昧・非可逆なら変更を拒否する。変更はSource Proposal → Validation/Impact → Shared Acceptance →再Projectionを経る。

- Dataの論理Identity/Entity/Relation/Field/Key/FDと、物理Table/Schema/PK/UI Row Key/生成IDを区別する。
- 一つのDataを複数Subsystemが参照することと、そのDataの正本更新責任が複数あることを区別する。
- 同じ実世界の概念が別のbounded contextで異なるモデルを持ち得る。機械的な同名一致でDataを同一視せず、相互変換・同期の契約を明示する。
- Scope + Source identity + revision + accepted meaningを固定する。同名異scope、異なるSource closure/Policy/Plugin版を混合しない。
- データオーナーが未確定、SourceのCRUDが未被覆、関係が複数候補、Function IOが不明の場合は**unresolved**。未観測をCUDゼロと表示しない。
- Source tokenやsource-indexの存在・fingerprintの一致はAuthenticity/業務Acceptance/Effect実証ではない。
- 図の座標・レイアウトや表示折り畳みは意味の正本に昇格させない。

## 3. 読取りパイプライン：提案と既存Owner

1. **Source selection**：対象Project/Scope/Revision/Source closure、Current Disclosure Policyと読取主体を固定する（#1120/#267/#2516）。
2. **Cross-view evidence**：FunctionからDataへの宣言C/R/U/D、Procedureとの関連、manual/external action、候補Owner、出所と未結合の母数を集める（#1031/#1125/#1127）。Source由来と推論由来を分離する。
3. **Ownership diagnostic**：既存#1132のread-onlyルールで、Sourceの整合が確認できる範囲に限定してcross-subsystem direct write・未知Pattern等を判定する。#1132のCandidate checkerをAccepted producerとして扱わない。
4. **Partition analyzer（本提案の新規責務）**：制約付き二部グラフ上で複数の候補を生成し、評価内訳・根拠・Unknownを返す。意味の受理やSource書換えを行わない。
5. **Proposal/Review**：Studio/CUIは同じ候補を表・図・差分として投影する。Selectionのscope/epoch/Sourceが古ければ編集・Inspectを拒否する（#1124）。人間の操作はSource変更候補だけを生む。
6. **Acceptance / Realization**：既存Shared AcceptanceとTarget Lowering/Build/Effect/Observation（#267/#1121/#1129/#1174）に委ねる。Proposalの評価良好を現在の認可・実行可能性・Commitへ読み替えない。

## 4. グラフと評価

機能集合 F、論理Data集合 D、Subsystem集合 S、候補所属関数 p を持つ。Access関係は**Operation kind, Target Data, Scope, Source/evidence, confidence, revision**を備え、R辺とC/U/D辺を混合しない。Function呼出し/Effect/TxとData参照は別Relation。

既知のdirect writeだけに対し、越境重み付き更新数は

    Wcross(p) = Σ_(f,d) w(f,d) · Write(f,d) · [p(f) != p(d)]

と定義する。ただしWriteがunknownの組には0を代入しない。既知・不明・未被覆の母数とSource closureを併記する。潜在的な自動実行経路（Trigger、Native SQL、外部Job、File/ETL、Rule）は対応Evidenceのない限り未解決。

**アルゴリズム方針（未決）**
- まずhard constraintsを検査する。必須の同居/分離、禁止移動、明示Owner、トランザクション不変条件、理解可能な規模の上限などは、未決のまま自動選択しない。
- 次に越境direct CUD最小化を優先する。内部CUD凝集・Subsystem間Read依存・Function/Procedure結合・変更影響・境界の安定性・移行コスト・サブシステム規模を評価し、複数のパレート候補を返す。
- 「全部一つへ集約」がゼロ越境CUDを達成しても、規模・分離制約・理解性に反すれば採用候補としない。コスト重みが不明なら恣意的に単一ランキングへ潰さず複数案を残す。
- 大きなモデルでは、可視化可能な変更単位を保つ階層/コミュニティ探索、決定的seed/安定tie-break、差分更新を比較する。特定アルゴリズムの採用や性能保証はReference実測後に決定する。
- 枝の重みは宣言上の「CUDの種類」と、実行頻度・変更頻度・業務重要度を区別し、推定頻度を既知値として扱わない。

**候補の最低限の出力**：sourceBinding（Scope/Revision/Closure）、evidenceCoverage、candidateMembership、proposedOwnership、hardConstraints、directCrossWrites、effectiveDependencies、unresolved、tradeoffScores、changeImpact、optimizerPolicyVersion、candidateDigest。全てProposal/Assessmentであり、Canonical identityではない。

## 5. 合成例：販売と在庫

変更前の宣言例：

| Function | Order（販売がOwner） | Stock（在庫がOwner） | 現所属 |
| --- | --- | --- | --- |
| 受注登録 | C | U | 販売 |
| 在庫引当 | R | U | 在庫 |
| 出荷確認 | R | R | 販売 |

直接越境CUDは受注登録→Stock Uの1関係。これは**明示された静的declared edge**の例であり、実Effectの証拠ではない。

候補A「販売・在庫を統合」はdirect越境CUDを減らすが巨大Subsystem化の圧力がある。候補B「在庫Owner側の明示Commandを呼ぶ」では販売のStock直接Uをなくせるが、販売→在庫の委譲、受注の状態遷移、在庫引当の整合性、部分成功・回復は**新しい依存として表示し続ける**。候補C「注文と引当の状態/更新時点を分離」は別の業務意味を要求するので、単なる機械的な機能移動として提案しない。

実行で在庫Owner側を呼ぶ場合も、権限があるとは判定せず、Effect/Owner/Receipt/独立Readの既存Gateを維持する。実際のトランザクション保証が欠ける候補は、安全な置換案として受理できない。

## 6. Studioの三層表示と境界編集

| View | 主表示 | 必ず残す情報 |
| --- | --- | --- |
| 全体Subsystem構成 | 所有・参照・Command委譲・Effect依存 | 外部参照、未解決の関係数、表示/開示範囲 |
| Subsystem ER | 内部Data/Key/Relation、外部Dataの参照ノード | externalを消して独立と偽らない。Relationの所属と端点scopeは別 |
| CRUD / Effect matrix | Function×DataのDirect CRUDとEffective依存（別層） | declared/derived/observed、Unknown/coverage、source位置 |

各図表の完全性は**選択したScope/Projection/Disclosureの範囲内**で判定する。「図中になければ存在しない」と結論できるのは、Read権限・Source closure・解析被覆が十分で、対象要素の存在判定がclosed-worldとして保証された範囲に限る。非開示Dataの件数、位置、診断経由の漏洩も独立に検証する。

Studioの逆操作は、ドラッグによるMembership変更も含め必ず候補・差分。図上のSubsystem境界と実Scope/Role/Ownership契約が一致しないときは無断同期しない。旧応答、古いGraph選択や部分成功後の画面が新しいAccepted meaningのように残らないよう#1124のfreshness規則を再利用する。

## 7. ETL・ファイル連携・Self-build・プラグインへの影響

- **ETL/ELT/Batch**：Read→Transform→Materialize/Publishと、業務DataへのCreate/Update/Deleteを分ける。ファイルを転送した事実を業務Dataの更新済みと呼ばず、書込先のOwner/Operation、Commit/独立Observationを別に追う。#1130/#2542/#2619/ETL設計を再利用する。
- **Legacy移行**：既存SQL/Script/ORM/Trigger/JobからのCRUDはObserved/Extracted candidateとして入力する。Static call graphは到達可能性であり、actual EffectでもAccepted Ownerでもない。未知の動的SQLなどをcoverageに残す。
- **Plugin**：tei.std.*（標準の汎用構造）、tei.experimental.*（分割最適化の検証候補）、tei.contrib.*（分野固有な評価）という分担候補。ただしnamespace配置/標準化はOwner判断待ち。Pluginは複数インスタンスをNode/Portで結合可能で、唯一のSubsystem Organizer singletonを想定しない。
- **Self-build**：同じtyped Relation/Dependency/Constraintを開発パイプラインにも転用できるかpressure-testする。ただし業務SubsystemとBuild Unit/Artifact closureの意味は同一ではない。必要なら型付きrole/mappingを別に定義し、Toolchain/実Consumerを新しいGateなく偽装しない。
- **Target**：Single Binaryから多プロセスへの分割は選択式であり、Host必須化なし。Go/Wails Desktop、Java/.NET、REST/OData、ファイル連携の各実現で同じ論理境界を保持できるか比較する。

## 8. 受入計画（いずれも新規の実施報告ではない）

| ID | 最小試験 | 合格の観点 |
| --- | --- | --- |
| SP-01 | 同一Source/Constraints/Policyを2回評価 | 候補集合と順序、Score内訳、fingerprintが決定的 |
| SP-02 | R-only外部Dataと外部Uを対比 | Readを誤違反にせず、明示Uのみdirect cross-write候補 |
| SP-03 | 同名異scope、Owner複数、Function未解決 | 不明を自動確定せず候補のconfidence/coverageに反映 |
| SP-04 | Sourceに現れないNative SQL/Trigger/Batchを含む | missing coverageを「越境0で安全」と言わない |
| SP-05 | 所属全集約と規模/分離制約 | 全部一つが常に最良とならない |
| SP-06 | Sales→Stock更新をOwner委譲へ変更 | direct越境CUDは消えてもEffective/Tx依存が残る |
| SP-07 | 同一意味でレイアウトだけ変更 | Candidate score/Accepted meaningは不変 |
| SP-08 | Source/Policy/Plugin revision変更 | 旧候補はstale、適用不可。変更範囲を検証 |
| SP-09 | Subsystem ERに外部Dataと跨りRelationを表示 | Scope/Relation identity/Disclosureの非干渉と曖昧拒否 |
| SP-10 | 同じ論理分割から単一Go/複数Targetを検討 | 論理意味とBuild/Provider/Effect資格を混同しない |

**段階**：P0はAccepted Source→既存#1132 diagnosticの接続範囲とcoverageの検証。P1は候補/制約/決定的採点の純粋Reference。P2はSource変更ProposalとStudio三層Projection。P3はLegacy/ETL/実Provider/複数Target・認知負荷測定のpressure test。各段階を#1119の製品Gateに勝手に昇格しない。

## 9. 既存Ownerと依存

- #1031: 旧Sourceの三視点Evidence（Accepted identityへの自動昇格なし）
- #1132: 候補Owner/direct CUD/Procedureのread-only診断Reference。診断を再実装しない
- #1120: Scope/typed ID/Source、#1125: 業務手順、#1126: Entity/Relation/FD/ER、#1127: Function/CRUD、#1129/#1174: Rule/Effect/Commit
- #1124: StudioのSelection/Freshness/Source往復、#267: Shared Acceptance、#1121: Lowering、#1803: 設計成果物
- #2619/#2495: Everything is Data/標準Facet、#2516: Disclosure、#1130/#2542: ETL/Batch/File、#2831: Self-build、#1119: 製品Critical Gate

## 10. 原典・参考

- [三要素分析法とサブシステム/CRUD](https://www.itmedia.co.jp/im/articles/0605/09/news085_2.html)
- [業務システムとマイクロサービス（1）](https://dbconcept.hatenablog.com/entry/2020/11/05/074400)
- [業務システムとマイクロサービス（2）](https://dbconcept.hatenablog.com/entry/2020/11/06/100003)
- [Microsoft Domain Analysis / Bounded Context](https://learn.microsoft.com/en-us/azure/architecture/microservices/model/domain-analysis)

この文書は**設計候補の保存**であり、正式なSemantic変更・Optimizer実装・実験PASS・Studio表示・実DB/Target動作・人間による設計受理を宣言しない。


## 11. 2026-11-07 深化：分割と境界修復は別のProposal

**分類A: Partition-only** は、元のData/Function/Procedure/CRUD/Effect宣言を**不変**としたまま、所属・境界の候補だけを変える。候補の越境CUD改善は、同一Source closure・edge universeを母集団とした場合に限って比較可能である。Functionを別Subsystemへ移動しても、Functionが既存Dataへ行うDirect CUDは消失せず、所属に応じて内部/越境へ再分類されるだけである。

**分類B: Boundary repair** は、既存の直接更新をOwner側Commandへ委譲、Business RuleをOwner側へ移動、Data責任を分解するなど、元の操作契約やSource自体を変える。修復候補は、先にBefore/AfterのAction・Read/Write/Effect・Transaction・失敗回復・開示・Authorityの対応を明示し、同義性の未証明を保持する。自動生成した修復を採用済み意味とみなさず、#267/#1129/#1174/#1180の受理・独立Expectedへ渡す。

| 例 | Partition-only | Boundary repair |
| --- | --- | --- |
| 外部Table Uを行うFunctionの所属移動 | 旧U edgeは残る。新境界で再集計 | なし |
| 外部Table UをOwner Commandへ置換 | 対象外 | 直接U→Call/Effectへ変更。Tx、拒否、再試行、Unknownの契約を別途検証 |
| 共通TableをDomain別Dataへ分割 | 対象外 | Identity/意味/Sync/移行・Source-Target mappingを変更。人間の業務判断が必要 |
| 同一論理Subsystemを二つのArtifactへ配置 | 論理変更なし | Target/Buildの実現候補であり、業務操作の修復ですらない |

**反例**：候補が越境CUDを0にしたと報告する一方で、元の外部U edgeを根拠なく削除している場合、分割改善ではなく未受理の業務変更である。比較不能として停止する。

## 12. 最適化の評価契約・被覆率

### 12.1 計算母集団

固定されたScope/Revision/Accepted Subject集合に対し、Function–DataのAccess Relationを、Source-backed declared / parsed-extracted / derived-potential / actual-observed / unresolvedで分離する。論理Relation自身の所属Scopeは両端のScopeから推定しない。Reference/Query/Trigger/ETL/外部Fileが参照するDataを名前のみで紐付けない。

各Accessについて、operation C/R/U/D（拡張Operationは対応写像が明示されるまでunknown）、Data粒度、Read/Write集合、出典、解釈規則版、対象Scope、coverage、stalenessを記録する。CallやEventの多段到達は直接CRUDの代用としない。複数CUD操作の重複は集合/多重集合のどちらを数えるか明示する。

評価値は、**closed and qualified scope の既知direct write集合**の越境数を算定し、その他は unresolved/write-coverage として必ず別表示する。スコアが0でも、母集団の閉包が証明されない限り「越境更新が存在しない」と表示しない。

### 12.2 制約と複数目的

| 評価順 | 判断 | 停止線 |
| --- | --- | --- |
| Hard | 明示Owner/業務不変条件、must-link/cannot-link、許可された変更、必要なScope/開示、サイズ制約 | 未確定・矛盾するなら計算上の候補に留める |
| Primary | 同一known direct CUD集合での越境数/重み | 未知edgeを0に補完しない。非同一closure間の改善率を比較しない |
| Secondary | 内部凝集、Read cut、Function/Procedure連携、影響半径、トランザクション依存、設計理解性 | Hard制約の違反を重み付き平均で相殺しない |
| Cost | 移動数、既存Owner変更、連携契約と移行費用 | 動的値・推定値を測定済みと呼ばない |

重みが不明なら任意の単一総合順位へ縮約しない。制約充足した**Pareto候補**と、何を優先すると候補が逆転するかを説明する。関係の重みが一部のみ既知なら、既知値に基づく評価と不確定範囲を分ける。評価パラメータやtie-break policyは版付きで再現可能にする。

**候補比較可能性の条件**：Candidate A/Bが同じSource closure・解析policy・accepted revision・Disclosureの範囲を共有すること。そうでなければ改善値は単純比較不可とし、差分入力や不足Evidenceを先に表示する。

### 12.3 検証票（例）

あるScopeについてData 8件、Function 5件、SourceのWrite edge 12件が宣言され、8件だけがqualifiedなら、越境既知edgeが0でも結果は「既知8件の越境0、未判定4件（+他に未探索Sourceが存在する可能性）」である。**システム全体の越境0**ではない。

## 13. Intent / Realization / Assessment とConsumer適合

他チャットのTEI共通方針に合わせ、以下を別Projectionで表す。

| Projection | 表示と証拠 | 境界 |
| --- | --- | --- |
| Intent | Accepted論理所属・Owned Data・Function Port・明示CRUD・制約 | 設計者の意図であり実SQL/Providerの更新結果ではない |
| Realization | Source parser・SQL/Trigger・Target Artifact・Plugin/Host/Batchから読める実装上の関係 | 未解析・動的経路、実権限/実Commit不明を保持 |
| Assessment | Scope/Revision/Policy/Source closureに結び付いた一致・齟齬・不明・陳腐化 | 不一致を報告する前に両者の同一対象性と充足根拠が必要 |

標準構造Roleの語彙が受理できること（V0）と、Subsystem/Owner/Accessという役割がSemantic型やBinder要求へ適合すること（V1）、実Studio/SHO/CUI/Target Consumerが正しく利用できること（V2）は異なる。#2495/#2619/#2620の既存Role契約を利用し、名前や表示型だけからV1/V2を自動合格にしない。

表示面では、Subsystem ER図の内部Dataとexternal参照、CRUD Matrixのeffective/potential/direct/observedを表示区別し、Relation所属Scopeと端点Scopeを保持する。表示圧縮によるedge欠落、非開示/未読取/不明、Renderer能力欠落を「不存在」に読み替えない。表現の損失・逆編集は[#3250の表示引継ぎ](https://github.com/hat47x/tei-platform/issues/3250#issuecomment-6703937718)と標準表現型の既存Ownerへ照会する。[#3236](https://github.com/hat47x/tei-platform/pull/3236)はclosedの検討記録であり、内容のmain統合・採用を前提としない。

Grant失効やSource改版をConsumerが**観測**したら、旧成功応答の再表示と既表示対象の現在資格を別に判定し、既存#1124のstale/epoch規則へ従う。観測されていない失効について即時無効化を証明したとは言わない。Diagnostic件数・拒否理由・境界位置から非公開データを漏らさないよう現在Disclosureに従う。

## 14. ETL / Plugin / Self-buildへの限定した接続

- ETL/MFTではDataset境界とBusiness Data Owner境界が一致するとは限らない。抽出Read、配送ACK、Import、Business Write、永続化後の独立Readを別edge/receiptとして扱い、転送済みを業務更新済みとしない（#2542）。
- Plugin合成は**要求固定→既存Provider探索→適合するCapability合成→不足だけ実装→独立Conformance→最適化**を既定とする。最適化が古いSource/Policyや対象Targetの能力制約を飛び越えない（#2933）。
- Self-buildに転用する場合も、業務Subsystemの所属とBuild GraphのRequired/Selected/Consumed/Observedを同型と決めつけない。選択されないPluginやGraphに無いStepが成果物に混入すれば拒否。Stage1/Stage2を同じ論理図として示すだけで実セルフビルドと主張しない（#2831）。
- Go/Wails Desktop、Java/.NET、REST/OData/File、単一Binary/多ProcessというTarget配置候補は、同じAccepted論理分割から別々に検証する。共通Hostを必須化しない。

## 15. 追加の反証可能な受入候補（SP-11〜18、すべて NOT RUN）

| ID | 負例・対照 | 期待する観測 |
| --- | --- | --- |
| SP-11 | 同じFunction/Data/Access集合で所属のみ移動 | write edgeそのものは不変、内外判定だけ変化 |
| SP-12 | 越境Uを削除・Owner Commandへ置換する修復候補 | Partition-onlyで改善と評価せず、変更契約と別受理が必要 |
| SP-13 | 一部Source/Trigger/動的SQLが不明、既知越境0 | 「既知0・未判定n」、全体ゼロ保証を拒否 |
| SP-14 | 同一名称のDataが異Scope、Relation所属は第三Scope | owner/Relationを名前や端点Scopeから推定しない |
| SP-15 | 同じ図でもSource/Grant/Disclosureが改版・失効 | stale候補適用拒否、既表示と後着応答を別検証 |
| SP-16 | レイアウト折畳み/表示圧縮/非開示で辺が見えない | 図中不存在から意味上の不存在を推定しない |
| SP-17 | ETL転送ACKありだがImport/Business effect/独立Readなし | 完了・Owner Writeの保証を拒否 |
| SP-18 | 同じ論理分割の単一Buildと分離Buildを比較 | 受理済み業務意味は不変、実Artifact/Dependency/Authorityは別の検証 |

**検証順序**：P0のread-only Source closure/coverage→既存#1132との差分、P1のPartition-onlyとBoundary repairを分けた小さな合成Reference、P2のStudio/CLI説明・Source Change Proposal、P3の実Reader/ETL/複数Targetへの接続。既存#1119 Gateを増設・変更せず、#1124のStudio直接P0、#2831の実セルフビルドP0に新しいhard dependencyを作らない。

### 今回照合した追加研究

- [Design Structure Matrixによる分割・統合のレビュー](https://doi.org/10.1109/17.946528)（DSMの構造依存行列）
- [CRUD MatrixとAxiomatic Design](https://doi.org/10.1016/j.compind.2021.103569)（CRUDによる情報システム設計の先行研究）
- [Microsoft: Microservice Boundaries](https://learn.microsoft.com/en-us/azure/architecture/microservices/model/microservice-boundaries)（Aggregate、業務責任、通信負担、非機能制約）
- [AWS: Business-domain service boundaries](https://docs.aws.amazon.com/wellarchitected/2024-06-27/framework/rel_service_architecture_business_domains.html)（Domain境界と組織・設計上の責務）

これらは比較研究の根拠であり、特定のアルゴリズム・係数・物理マイクロサービスへの自動変換を正式採用した根拠ではない。


## 16. MECE・閉値域・強属性とSubsystem責任（2026-11-07）

設計特性の検討（#2495/#1126、Roleとfacetの既存Owner）と混同しないため、次を分ける。

- **Subsystem分割のMECEは対象集合と軸を指定した場合だけの設計制約**。全業務Function/Procedure/Read Projectionの所属が必ず互いに排他的とは限らない。Shared Function/共通参照、外部System、未割当、段階移行を明示的に表す。
- **Dataのprimary write ownerがexactly-one**という条件は、受理済みの対象データ・有効期間・scope・ownership policyがあって初めて検証するガバナンス規則。Owner候補が複数/0件の場合、勝手に選ばずambiguous/unresolved。バックアップやレプリカが別Systemに存在しても、同じ論理正本を勝手に追加する根拠にならない。
- **Closed enum/Closed Domain・網羅性・排他性・識別性・有効時点**は別の性質。Subsystem集合を有限に列挙できることは、所属の網羅・重複無し・業務境界の正しさを保証しない。
- **強属性**は原典上の厳密定義・TEI共通受理が未完の候補であり、候補キーや識別子やSubsystem ownerと自動的に同一視しない。識別/更新の責任検証に必要な別Traitは受理済みRole/Ruleから参照する。
- 例外のある共有Dataを単に「複数Subsystemで更新可能」と正規化せず、どちらの業務Command/Scope/期/Authorityがどう引き受けるかの明示契約へ分解可能か検討する。契約が不足する場合は未解決のまま保持。

**追加候補 SP-19 / SP-20（NOT RUN）**：SP-19はShared Read Function・手作業・外部Systemがあるモデルを無理に全件exactly-one membershipへ押し込まないこと。SP-20は時点付きprimary write owner候補が複数/欠落しても、自動更新許可や実Owner決定へ昇格しないこと。


## 17. 固定Owner下の越境CUD下界と修復必要性の証明（2026-11-08、設計候補）

本Issueの新しい価値は「最良の候補を出す」に加えて、**所属変更だけで達成できない目標を反証可能な根拠で説明する**点である。既存#1132の候補CUD診断を使い、別のOwner判定器や実Effect推論を実装しない。

### 17.1 成立条件と定義

同一revision・Scope・解析Policyで固定された、十分に照合されたdirect-write辺集合 E を考える。Data d は受理済みかつ一意のprimary write owner g(d) を持つ。各Function f はこの比較の間は**一つの所属先** p(f) を持つ。Data ownerのgroupingは固定、Functionを分割しない、Functionのwrite辺を変更しない。許される所属候補を A(f) とする。Eの各辺 e=(f,d,operation) に非負の重み w(e) を与える（重みの根拠が不明なら全辺1とする暫定分析を明示する）。

固定Owner下での越境既知重みは次である。

    Cross(p) = Σ_(f,d,operation)∈E w(f,d,operation) × [p(f) ≠ g(d)]

Functionごとに write 先Owner group g の重みを W(f,g) とする。許された所属先のうち、最も大きい内部重みを残せた場合でも、次の**下界**未満にはできない。

    LB_fixed = Σ_f (Σ_g W(f,g) - max_(s∈A(f)) W(f,s))

A(f)が空なら分割候補自体が不成立。A(f)がすべてのOwner groupを含み、Function間の追加制約がない単純な場合、この下界は達成できる。must-link、can't-link、Subsystem規模、Function同居など追加制約がある場合は**下界に過ぎない**。全機能を一つのSubsystemへ入れる、Data Owner境界を変更する、Functionを別Operationへ分解するなどはこの証明の前提から外れるため、同じ下界と単純比較しない。

無重みの場合、一つのFunctionが直接更新するDataの固定Owner groupが k 個あり、Functionが一つのgroupにしか属せないなら、異なるgroupにつき少なくとも k-1 本のdirect-write関係は越境として残る（各groupへ少なくとも一つの関係がある場合）。ただし総合スコアやTransactionの安全性を証明するものではない。

**重要**：LB_fixed が正なら「所属変更だけでは既知越境CUDゼロ不可」の構造的証明になる。LB_fixed がゼロでも、制約追加・未解析辺・実Effect・業務意味により「全システム越境更新ゼロ」は証明されない。

### 17.2 合成例と期待値（本設計上の手計算・製品試験NOT RUN）

固定Owner Sales=Order、Inventory=Stock。

| Function | 更新先と重み | 許容所属 |
| --- | --- | --- |
| PlaceOrder | Order:C(1)、Stock:U(2) | Sales / Inventory |
| Restock | Stock:U(1) | Inventoryのみ |

候補Aは PlaceOrder=Sales、Restock=Inventory。known direct-write relations 3本、越境1本（PlaceOrder→Stock U）、越境重み2。候補Bは PlaceOrder=Inventory、Restock=Inventory。**同じ3本**の関係を保持し、越境1本（PlaceOrder→Order C）、越境重み1。従って **LB_fixed=1**（重み）、最良候補でも越境本数は1。いずれも越境ゼロではない。業務上PlaceOrder=Sales固定ならLB_fixed=2となる。重み2/1は説明用の仮定で、実測頻度ではない。

この例は変更操作を許可した場合の構造的候補比較であり、PlaceOrderの担当者・業務責任がInventoryへ移せると判断したものではない。Owner側Command委譲によるU消失は新たなFunction/call/effect契約を必要とするBoundary repairとして別比較に回す。

### 17.3 Analyzerの結果契約

最低限、(1)入力Source closure/known E/Unknown母数、(2)Owner binding/allowed membership/hard constraints、(3)LB_fixedとその適用前提、(4)各FunctionのOwner group別write weightと残存越境の根拠、(5)分割候補と実越境値、(6)ゼロ不可/条件付きゼロ可能/不明の**判定理由**を返す。

「ゼロ不可」は固定Owner・非分割Function・既知Eという問題設定に相対的で、普遍的なシステム設計の不可能性ではない。入力Sourceの不完全性がある場合、Eに対する下界の説明は可能でも、未知の辺についてゼロ保証を与えない。Assessmentが最小の変更理由を示し、人間が再設計を検討する材料とする。

## 18. 分割最適化と既存移行検収の接点

他チャットの[設計移行Draft #3252](https://github.com/hat47x/tei-platform/pull/3252)は、既存ソース由来の設計候補、旧実装の動作、TEI受理済み意味、生成先実装を区別して比較する。#3250ではSubsystem境界とCRUD/Effectの**差分説明**だけを提供し、旧Java/C#/SQL/Triggerの抽出・実装互換の受理は#3252/#2555と既存Scenario/独立ExpectedのOwnerに委ねる。

| 比較 | 本Issueが報告できること | 別Ownerによる検証が必要なこと |
| --- | --- | --- |
| 現行所属→Partition-only案 | 同じ既知edge集合での越境本数・重み・影響Scope | 業務責任変更の正式受理 |
| 現行所属→Boundary repair案 | Source提案のwrite/call/effect差分、前提と未決 | 旧動作・業務Meaning・新実装の三者比較、Tx/Authority/回復 |
| Owner/DB配置の移動・Cutover | logical Owner変更、残存越境依存、Data identity対応候補 | 実データ移行、物理Schema、二重Writer排除、切替時点、独立Read |
| 単一Artifact→複数Build Target | logical partitionが不変か、期待される通信境界 | 実Build closure、Process/Provider/Grant/Artifact一致 |

「旧実装と新実装が同じ越境CUD数」でも意味互換の証拠にはならない。旧不具合の保存と意図的な修正も、同じ静的Matrixからは区別できない。逆に意図的修正のためMatrixが変わっても、新しいOwner Meaning受理なしに不適合と決めない。

## 19. 最小Referenceの具体的着手順と反証票（提案/未実装）

**Step P0-a（Read-only）**：現在の#1132 candidate checkerと、#1120/#1127等の受理済みProducerを比較し、Accepted Scope/Source closure/Owner/Function IOのうち**実在する契約**だけを再利用する。実Producerが無い場合は接続を「未実装」として止め、用意した合成JSONをAcceptedと呼ばない。

**Step P0-b（Partition-only）**：固定Owner、fixed known CUD edges、Function allowed membershipの極小fixtureだけを用い、エッジの保存不変条件、LB_fixed計算、越境edgeの再分類、単純全列挙による小規模oracleを比較する。OracleはOptimizerとは独立した実装で計算する。大規模ソルバーや新Constraint DSLは採用しない。

**Step P1-a（候補と説明）**：複数候補の違反・Score内訳・Source位置・未解決母数をCUI read-onlyで比較する。解けないhard constraintは「最適化失敗」ではなく矛盾集合候補と未決の出所を表示する。Closed-worldの根拠なしに全体ゼロ判定をしない。

**Step P1-b（修復の分岐）**：LB_fixed>0のFunctionだけを修復検討対象に優先表示するが、Command委譲等の業務変更を自動生成・受理しない。所属変更を伴わない修復、物理Target移動を別カテゴリに保つ。

| ID | 反証入力 | 期待する結果 |
| --- | --- | --- |
| SP-21 | 2固定Ownerを跨いで書くFunction、所属先2候補 | LB_fixed>0、越境0を不可能と報告（前提付き） |
| SP-22 | 同じedge集合のまま所属先/重みだけを変える | direct edge universe不変、越境分類と重み値は再計算 |
| SP-23 | 既知edge0だが未知Trigger/Sql解析不足 | LB_fixed=0でも全体越境0を保証しない |
| SP-24 | must-link/allowed-member制約、またはData Owner境界を変更 | LBの適用条件を更新し、旧Problemの値を流用しない |
| SP-25 | 外部直接U→Owner Commandの提案 | Partition-onlyではなくBoundary repairへ送り、Effect/Tx/独立Expectedが未検証なら実現資格なし |
| SP-26 | 同じCRUD表だが旧実装と新生成物の業務結果が異なる | CRUD一致だけで互換/意味適合のPASSを出さない |

**優先順**：P0-aでAccepted Producer/Scope/coverageの実際の欠落を先に解く。その後P0-bの小規模独立Referenceで構造的な下界・不変条件を確認し、Studio図表/P2、旧資産移行・複数Target/P3へ進む。既存#1119/#1124/#2831の直接P0に新たなHard Dependencyを置かない。上記のSP試験は**すべて提案・NOT RUN**。


## 20. P0 Source/Owner bridge：現mainで実在する入力境界（2026-11-08、設計／NOT RUN）

優先は新Solverではなく、**受理済みSourceに結び付いたCRUD/Owner事実の取得資格**である。Source Index、管理Source集合、実Reader消費、Parser/Role、意味受理、Current Grantは別Claimであり、単一のfingerprint一致から全てを推論しない。

| 静的確認対象 | 現在確認できる契約 | 不足を埋めずに残す範囲 |
| --- | --- | --- |
| [reference-subsystem-cud main.go](https://github.com/hat47x/tei-platform/blob/main/tools/reference-subsystem-cud/main.go) | WorldからsemanticsのData IDとapplicationsのPattern IDを読む。Owner/FunctionIO/ProcedureStepは別途callerが与えるcandidate facts | checker自体に受理済みOwner/FunctionIOのProducer/Authority認定はない |
| [Reference README](https://github.com/hat47x/tei-platform/blob/main/tools/reference-subsystem-cud/README.md) | World/facts bytesのdigest一致、Procedure Source Indexの位置候補検証 | candidate digest一致はAccepted Source・完全なFunction CRUD・実Effectを証明しない |
| [#1127](https://github.com/hat47x/tei-platform/issues/1127) | UI向けderived CRUD Operationの検討・限定試験 | Derived Operationを受理済みFunction direct writeへ自動転用しない |
| [#2620](https://github.com/hat47x/tei-platform/issues/2620) | Required／Declared／Resolved-Consumed／Observed-DisclosedのClaimを分離 | Loaderが読んだRoleを解析済み／受理済み／現在開示可能とみなさない |

以上は**このcheckerの静的事実**である。TEI repository全体にProducerが存在しないと断定せず、#1120の型付き関係、#267のAcceptance、現在のReader・Role・Source Indexを実際に照合して不足を確定する。

### Bridgeが渡すべき照合項目（新しい正本Schemaではない）

- typed Subject / Scope / Workspace / current accepted revisionと、関係のData・Function・Procedure粒度を固定。
- Claim相対Required Source closure、選択済みSource、実Readerが読んだbytes、Parsed/Anchored関係を突合する。未読込・未Parse・未知SQL/Trigger/ETLの理由を区別。
- Owner/Function direct CRUDはAccepted宣言か・caller候補か・source抽出かを区別し、適用時点・Scope・未解決を保持。
- Evidence/projectionのバージョンと実Consumer Attemptを結び、Source digest一致だけではApprovalやCurrent Grantを出さない。
- 意味上の宣言CRUDの完全性と、実装上のNative SQL/Trigger/Job/外部Writerの完全性は**別の閉包**で評価する。宣言側のclosed-worldはactual effectの不存在を証明しない。

**P0の合否**：既存Sourceからaccepted ownerとaccepted function-data operationの両方を同一Scope/Revisionで構成できるか実測する。不可の場合は「未結線」と報告し、合成factsをAcceptedとして改名しない。

## 21. 候補の証明資格：Feasible・Lower bound・Optimalityを別表示

固定Ownerの既知Write集合から得たLB_fixedは**下界**であり、ゼロ達成の実現可能性やOptimizer最適性の証明ではない。

| Claim | 証拠 | 誤認を防ぐ条件 |
| --- | --- | --- |
| Feasible | 具体的な所属割当と全hard constraintの確認 | 実現可能だが最適とは限らない |
| Lower bound | 対象edge・Owner group重み・allowed membership・固定前提 | 計算上の最小下界であり達成可能とは限らない |
| Optimality | feasible候補値が証明済み下界に一致、または独立の全列挙/厳密証明 | heuristicの探索終了だけでは最適性を認定しない |
| Infeasible/Unknown | 矛盾する制約の証跡、または探索/Sourceの不足 | 未探索と数学的不可能を混同しない |

**SP-27の合成反例**：Owner(A)=S1、Owner(B)=S2、fAはAのみU、fBはBのみU。fA/fBとも所属候補はS1/S2。独立したFunctionごとのLBは双方0となる。しかしmust-link(fA,fB)により二機能が同じサブシステムに所属しなければならないなら、S1でもS2でも必ず直接越境Uが1本残る。ゆえにLB=0はゼロ達成可能の証明ではない。二候補の独立全列挙Oracleで、合成例の最適越境本数が1であることを確認する（試験自体は未実施）。

Candidateの説明には、固定Problem identity、Source closure/解析Policy、受理済みOwner/Function scope、allowed membership/hard constraints、原Direct Write集合、所属配置、更新辺のInternal/Cross分類、LBと適用仮定、solver実装/版、feasible/optimalityの独立Witness、Unknown/Disclosure状態を含める。これは既存Proposal/Assessmentへの要求であり、第二Canonicalや新Core列挙型の採用ではない。

## 22. 秘密Edgeが候補順位・Scoreから漏れる問題（Disclosure）

設計図から秘密のData nodeを消すだけでは不十分である。**private write edgeが変わることで公開候補の順位・Score・診断数・表示サイズが変われば、秘密の存在を推測し得る**。全体Optimizerと低権限Consumerの資格を分ける。

- Current Policyが事前にQualifiedした**Public Sliceだけ**を公開向けAnalyzerへ渡す。Private Worldを丸ごと最適化し、最後にRendererで秘密nodeだけ非表示にする方法は公開非干渉を保証しない。
- 公開Sliceで求めた候補は公開Slice**相対**の候補であり、全世界の最適化結果と呼ばない。秘密制約に依存する全体候補は許可されたConsumerだけに扱わせる。
- 低権限Consumerへ「非公開Writerがある」「非公開制約で候補が除外された」などの件数・拒否理由を漏らさない。必要であれば既存#2516 Policy結果で説明の範囲を制限・拒否する。
- 二世界W0/W1で公開Sliceは同一、private write edgeのみ異なるとき、低権限の候補・順位・Score・件数・診断・図形・DOM/ARIA・Tool JSONが同一であるか実Producer→実Consumerを比較する。Timing/caching副作用まで保証したと主張しない。
- Source改版・Grant失効をConsumerが観測したら旧表示と旧結果の新規開示資格を**別々に**確認する。未観測の即時失効を保証せず、#1124/#2516の既存Ownerに委譲する。

公開可能な依存閉包と「未解析のSource」は別の観点であり、private edge数をunresolved件数に転写しない。「図中になければ無い」は対象と公開可能閉包が確定した場合に限定する。

## 23. Design Decision Reviewと最小受入順

Studio/CLI/AIで共通の設計判断Reviewは、**既存Source Proposal/Impact/Acceptanceのread-only Projection**として実現する。候補配置を比較する時点から、Ownerの業務判断、実行資格、物理Target適用を分離する。

最小のReview表示に必要なのは、比較可能なProblem binding、提案所属差分、Hard制約、元のDirect Write edge集合、越境再分類、LB/Feasible/Optimalの根拠、影響を受けるFunction/Data/Procedure/ER/Source、Unresolved/Disclosure状態である。SourceをR10→R11→R10と変更した場合、digestが一致しても過去のAttempt/Accept/Grantが現在有効であるとは限らない。旧Proposalの自動再受理はしない。

| ID | 反証入力（すべて設計候補、NOT RUN） | 期待する結果 |
| --- | --- | --- |
| SP-27 | 各Functionの独立下界0、must-linkによりゼロ不可 | LBと実現可能性を分離、全列挙Oracleで最小1を検証 |
| SP-28 | heuristicが停止し改善候補を返した | 可行性は示せても最適性証明は出さない |
| SP-29 | caller作成factsとWorld digest一致 | accepted Owner/FunctionIOの根拠にはせずcandidate-only |
| SP-30 | LoaderはSourceを読んだがRole/Required closure未確定 | 未解析を「更新関係無し」にしない |
| SP-31 | 同一Public Slice、秘密write関係だけ異なる二世界 | 低権限の順位・Score・診断・図形等が変化しない |
| SP-32 | 全世界の最適候補から秘密nodeだけ消去 | Public-only再計算またはPolicy拒否。全世界の最適性表示禁止 |
| SP-33 | Source R10→R11→R10のABA、同digest・異Attempt | 旧Review/Selection/現在権限を自動復活させない |
| SP-34 | 所属変更後の外部Relation/手順が未再評価 | 図の一致だけで変更・意味受理完了としない |
| SP-35 | 一部Subsystemの変更による差分再評価 | unaffected側を最小再評価できても、依存閉包不足を隠さない |
| SP-36 | 公開向けScoreから非公開Writer数を推定できる | 既存開示境界に従って出力不可・不変を検証 |

**実装提案の順**：(a) accepted Producerと現在の#1132 checkerのGapを実Sourceで固定、(b)SP-27/28の二機能二Dataの独立全列挙Reference、(c)候補/影響表示はSource修正なしのread-only Consumer、(d)Disclosure二世界のProduct受入は既存#2516/#1124へ引継ぐ。新Optimizer/Core/Host/Policy engineを先行実装しない。既存#1119 Gate、Studio直接P0、セルフビルドP0へHard Dependencyを増やさない。


## 24. Must-linkの連結成分による厳密最小値（2026-11-08／設計候補）

### 24.1 追加の数理的結論

§17のFunction独立下界を一段強める。**固定Data Owner、同一既知Write辺集合、Function非分割、各Function単一所属、非負辺重み、hard constraintが「所属許可集合」と「Function間must-link」だけ**という条件に限定する。

must-linkによる無向グラフの連結成分 C に含まれるFunctionは、必ず同一Subsystemへ所属する。連結成分Cについて、許容所属は

    A(C) = intersection_{f in C} A(f)

となる。A(C)が空集合なら、この条件系自体が不成立である。Dataの固定Owner g に対する既知Direct Write重みは、成分内の全Functionについて集計した W(C,g) とする。成分の合計更新重みを W_total(C) とすると、越境更新の最小値は

    OPT_mustlink = Σ_C [ W_total(C) - max_{s in A(C)} W(C,s) ]

で**厳密に達成できる**。理由は、成分内の全機能は同じ所属を選ばなければならない一方、異なる成分間には追加の結合制約がなく、各成分はそれぞれ内部更新重みを最大にする所属を独立に選べるためである。

この条件下では必要な計算は、must-linkの連結成分、所属許可集合の積集合、Owner別Write重み集計、確定的tie-breakで足りる。専用の大規模Solver/DSLは必要ない。

**適用外**：cannot-link、サブシステム定員/規模、部分集合同居や各成分の相互依存、Owner改版・業務操作の書換え・Time-varying ownership、負の重み、未解決Direct Write。これらを含む場合、成分別の値は（前提が有効なら）下界であってGlobal最適性証明にはならない。明示的なhard constraint不成立と、Source closureの不足による不明を混同しない。

### 24.2 Zero-write-crossingの判定

上記に限定すれば、全成分について (a) A(C) が非空、(b) Cが更新するDataの固定Owner groupがゼロ個か一個、(c) 一個ならそのOwnerがA(C)に含まれる、という三条件が**既知Direct Write辺に対するゼロ越境可否の必要十分条件**になる。Writerのいない成分は任意の許可所属先へ配置できる。

ただしこのゼロ可否は受理済みOwnerと対象known Eを固定した数学的事実であり、全Sourceの閉包、実Effect、複数Ownerの解消、実認可、実Target配置、公開Sliceの完全性を証明しない。

### 24.3 独立全列挙との比較（小例の計算確認、製品Test NOT RUN）

| 入力 | 成分別の結果 | 独立全列挙の最適値 |
| --- | --- | --- |
| fA→S1 U(1)、fB→S2 U(1)、must-linkなし | 0 | 0（4配置中） |
| 同じ2 Functionにmust-linkを追加 | 1 | 1（2可行配置） |
| fA→S1(2)、fB→S2(3)、fC→S3(4)、三者must-link。許可所属の共通部分S1/S2 | 6 | 6（2可行配置） |
| fA許可S1のみ、fB許可S2のみ、must-link | 所属集合空・不成立 | 可行配置なし |

数値は説明用の非負重みであり、業務頻度の実測値ではない。小例の列挙はこの設計検討で実施した独立計算であり、TEI Repoに追加したGo/CI/Accepted Sourceの試験結果ではない。

## 25. 最新mainのSource/三視点契約への接続更新

[新しい三視点・CRUD Draft #3492](https://github.com/hat47x/tei-platform/pull/3492)は、古い#3263を現mainから再基準化し、既存Workspace Inspectorで特定Subjectを読むM1aの限定的なGo試験を含む。Source refs、Workspace Identity、WorldとSource Index fingerprintを照合するが、**正式なTEI三視点Graph/CRUDの全Product Projection、完全Function/Business closure、Accepted Subsystem Owner、逆編集、実DB Providerは未証明**である。

本IssueのP0 Bridgeはこの実Workspace Inspector Readを再実装しない。段階を次に改める。

1. M1aの実Workspace Subject Readで取得された実Source/Scope・Reader証拠を**入力側の限定基線**として確認する（Draft試験が本Issueで実行済みとはしない）。
2. そのSame Subject/Workspace/Revisionで、Accepted Data Owner、Function direct CUD、ProcedureとEffectへの別Relationが**本当に受理済みProducerから解決できるか**照合する。欠落するならcandidate-onlyを維持。
3. 既存#1132の診断と接続し、fixed owner / unknown / conflicting ownerを区別する。UI向けDerived Operation・既存Reference factsをaccepted writerに繰り上げない。
4. 条件が成立する範囲だけ、§24のmust-link exact evaluatorを独立の小Referenceとして試す。条件外は下界/Unknown/修復検討の別分類へ送る。
5. #1124 Studio、#2516 Disclosure、#267 Acceptance、#1180/#2514 Expected、#2831 Self-build、#3252既存移行検収は別Ownerとし、実効果/権限/物理配置を候補スコアで確定しない。

最新mainには採用ADR-0048の共通Application API（CLI/MCP/Web/Desktop各Adapter）とread-only MCP参照Adapterがある。これらの入口へ同じ分析結果を提示する場合も、共通API契約とCurrent Consumer認可・受領の受入を分け、新Analyzer固有の第二Transportを作らない（[Draft #3303](https://github.com/hat47x/tei-platform/pull/3303)、[Draft #3495](https://github.com/hat47x/tei-platform/pull/3495)参照）。

## 26. 証拠・再現性・新しい反証票（SP-37〜44）

| ID | 条件付き反例（新実行 NOT RUN） | 期待 |
| --- | --- | --- |
| SP-37 | 独立2機能は下界0、二者must-linkで相異なるOwnerへU | 成分の厳密最小値1、既知越境ゼロ不可 |
| SP-38 | must-linkが推移的に3機能へ波及 | 連結成分単位にOwner別更新を集約し全列挙と一致 |
| SP-39 | must-link成分の許可所属集合の積集合が空 | InfeasibleとSource条件を示し、Unknown Sourceと混同しない |
| SP-40 | must-link成分では最良0だが別成分とのcannot-link/容量制約がある | 成分値をGlobal最適性証明と呼ばない |
| SP-41 | Writeが0件の成分、Allowedが非空 | Ownerなしを偽のWriter/違反と扱わない |
| SP-42 | 実Workspace M1aのSourceは読めるがAccepted Owner/FunctionIOが欠落 | candidate-onlyに停止し実Source Read≠Accepted CRUDとする |
| SP-43 | 同じReadがCLI/MCP/Studioで見えるがCurrent Grantや実受領が異なる | 同一Answer/Hashから現在資格・同一Attemptを推測しない |
| SP-44 | 修復でFunction→Owner Commandへ変更しEffect/Txが未実証 | Partition-onlyの厳密最小と別にし、既存受理・独立Expectedへ渡す |

**実装順**：P0は#3492の実Workspace Readの限定範囲と、受理済みOwner/Function I/OのGapを確定する。P1はmust-linkのみの厳密評価と独立全列挙Oracleを最小のread-only Referenceで比較し、cannot-link/容量/業務意味が現れた段階で初めてSolverや追加評価の費用を検討する。専用Core型・第二Graph/Acceptance・新Host/Policy/Transportを導入しない。

現#3255はmainから大きく乖離したDraftである。本文の数理的結論とSource接続の最小契約を保存した上で、**mainの文書を消さない非強制追随・差分再確認**を行う。旧HEAD時点のtest結果や他PRの合成greenを現在HEADの製品受理に転用しない。


## 27. 2026-11-09 実Referenceの分離とP0判断の更新

[Draft PR #3662](https://github.com/hat47x/tei-platform/pull/3662) に、この文書§24の **must-linkのみの条件付き厳密解** を純粋Go実装として独立させた。これは別正本・新Gateではなく、以下を切り分けるためのread-only候補試験である。

- **コードにした範囲**：固定候補Data owner、明示direct C/U/D relation occurrence、Functionのallowed所属、must-link、正の整数重み。連結成分のAllowed積集合、Owner別Write集計、厳密最小越境重み、deterministicな選択、edge witness。cannot-link/規模制限/変更系Effectには対応しない。
- **独立Oracleの候補**：SP-37〜41に対応する全配置列挙、3機能×3所属先のallowed/must-link 1,372組、無更新・入力拒否・順序安定性。Testソースは実装Draftに保存したが、この記録時点では正規Go Test・Repository Healthの成功を**未確認**とする。数学例のJSでの列挙とGo Testの実行資格は別。
- **Product接続で未実証**：Accepted Subsystem Owner/FunctionIOを一つのScope/Source/Revisionで取得する実Producer、#1132 checkerとの束縛、#3492のSource/Workspace M1aとの同一性、Disclosureに基づくPublic SliceとConsumer実受領、Studio/CLI/MCP/DB/Target、業務Owner意味受理。
- **優先順**：P0は必ずAccepted Source→Owner/Function direct CRUDの実Read/資格確認。P1は#3662の小Reference exact-head Go Testと独立全列挙の確認。P2は#3255の候補説明と既存Proposal/Impact/Acceptanceへの接続。P3はLegacy/ETL/実Targetの別試験。P1の合成成功をP0の完成に読み替えない。
- **他担当との衝突回避**：#1132の既存checker、#3492三視点Workspace Inspector、#1124 Studio、#2516 Policy、#267 Acceptance、#2831 Self-buildとファイル/責務を分離。新Planner/Graph/Parser/Host/Policy engine/Canonicalを追加しない。#1119のProduct Gateや他レーンの直接P0には新Hard Dependencyを置かない。

この文書・#3662ともにOwner正式採用前のDraftであり、**Go Test/CI/Owner受理/実Source結線/実Targetの合格は、対応するexact-headの証拠が取れた時だけ明示する**。旧§26末尾のmain乖離は当時の履歴であり、2026-11-08にはmainを非強制mergeした。次のmain前進には再照合が必要である。


## 28. 2026-11-09 現行ConsumerとOwner/Direct CRUDの入場条件

他レーンの限定実証を取り込んでP0をより小さくする。ただし各レーンの成功を相互に合算せず、**同一Claim/Source/Scope/Attemptで再照合**して初めて一つの証拠とする。

| 既存成果 | 今回再利用できる境界 | この成果だけで資格を与えない対象 |
| --- | --- | --- |
| [Draft #3492](https://github.com/hat47x/tei-platform/pull/3492) M1a/M1b-local | 実Workspace InspectorでSubject、Relation targetの一意性、Source Index/World/Workspaceの限定照合。後続でData/Function/Relation IDの逆参照へ使用 | Accepted Primary Owner、Function direct CRUD、全Relation/Source closure、同一原子的snapshot、実Provider |
| [Draft #3518](https://github.com/hat47x/tei-platform/pull/3518) M0 Model Map | 種類付きData→Relation→Data、Pattern→Data/Queryのread-only構造的追跡、欠落Link/重複Identityの負例 | Functionの更新責任とOwner意味、完全Graph、実Effect・利用権限 |
| [Draft #3543](https://github.com/hat47x/tei-platform/pull/3543) PoC A | 実CLI曖昧A/一意Jから同一Workbench Initial/List/Inspect 0/0/1という限定経路。#1124でSelection不正対象を拒否できる根拠 | 別Scope、Source改版、Current Grant、Graph/Command四入口、Owner正式受理、Subsystem最適化結果の開示 |
| [Draft #3321](https://github.com/hat47x/tei-platform/pull/3321) Trace境界 | byte同一の別物理Workspace RootがCLI観測上区別不能になる負例。Root/Read Attemptの同一性が必要とされるClaimを識別する | 同一snapshot token/digestのみでPhysical RootやCurrent Read Attemptの同一性を保証すること |
| [#1132 Reference](https://github.com/hat47x/tei-platform/tree/main/tools/reference-subsystem-cud) | callerが明示したCandidate Owner/Function I/O/Procedureの整合診断 | Input bytes hash一致をSource/Owner Meaning Acceptanceへ昇格させること |
| [Draft #3662](https://github.com/hat47x/tei-platform/pull/3662) | **候補グラフ限定**の厳密must-link分割最適値とCrossWrite witness | 未知Sourceなし、公開安全性、実業務・Tx効果、Accepted SourceへのBinding |

**最小P0プローブ（提案、NOT RUN）**：

1. Current Read資格のある実Workspaceから**一つのDataと一つのFunction/Pattern**を選び、既存Inspector/ModelMapで型付きSubject ID・Scope・Source位置を照合する。見つからないRoleを名前から補わない。
2. 同一RevisionでDataのPrimary OwnerとFunctionの**宣言Direct C/U/D**を既存Source/Acceptanceから取得できるか確認する。UI用Derived CRUD、Potential Effect、Observed Writeは宣言Direct CRUDではない。
3. Required→Selected→Read→Parsed→Bound→AcceptedのClaim相対閉包を確認し、Workspace/Root/Read Attemptが識別可能な範囲を分離。Physical Rootが必要なのにRoot IDが無い場合はnot-proven。Manifest/外部Provider Raw改訂時にはWorld/Index fingerprint不変でも旧Source引用の現在資格を否定する。
4. 現#1132のcandidate factsと#3662のProblemに、それぞれ**caller-supplied candidate**としてだけread-only入力する。Accepted Bindが別途証明されても、Evaluator自体にOwner資格の判定権は与えない。
5. 現行#2516で許可されたPublic Sliceを入力に用いる。秘密のWriter/MustLinkがGlobal optimumへ影響する場合、そのScore/順位/理由を公開Consumerへ流さない。

**停止線**：実Workspaceの限定Readが成功しても(2)の受理済みOwner/FunctionIOが無い／意味が曖昧なら、候補計算はできてもAccepted Subsystem Partitionではない。Source改版やRole自己申告でこの不足を埋めない。

## 29. Must-link成分の局所再計算と変更影響証明

§24の成立条件が維持される間は、問題をmust-link連結成分Cに分割でき、全体の既知越境更新最小値は各成分の最小値の総和となる。この**加法性**により、以下の変更は局所再評価できる。

- **一Functionのallowed set、またはそのFunctionだけのdirect writeの変更**：属する成分だけの最適配置・CrossWrite witnessを再計算すれば、他の不変成分の数理値は不変。
- **Data Ownerの改訂**：そのDataへ書く**すべてのFunction**が属する成分へ影響が及ぶ。変更Functionのみの再評価では足りない。
- **must-linkの追加**：結線により統合される成分を再計算。**削除**は旧成分を分裂させ得るため、旧成分全体の影響を確認。
- **Cannot-link、容量、複数時点Owner等の追加**：成分間独立の前提から外れれば、局所最適性証明の資格を撤回する。
- **Source/Grant/Disclosure/Plugin/解析Policy変化**：数理的な局所値の不変と、現在のSource・開示・実受理資格は別。後者のClaim閉包は再評価する。

この局所性は、元のtyped Graphの依存を完全に図示する場合の**再計算計画**である。隠れCache、暗黙hook、新Canonicalの導入根拠としない。Candidate計算の局所成功をStudio旧DOMの失効・Actual Effectの合格に昇格させない。

## 30. SP-45〜53：実Referenceと統合Producerの追加反証票（提案／NOT RUN）

| ID | 条件 | 期待 |
| --- | --- | --- |
| SP-45 | Function/Write/MustLink配列順のみ変更 | 最適値・割当・witnessは不変 |
| SP-46 | Data Owner改版が別成分の複数Writerへ波及 | 全Writer依存成分を再計算 |
| SP-47 | must-link追加で成分合流／削除で分裂 | 影響成分を特定し全量計算と一致 |
| SP-48 | 一成分内のDirect Uだけ変更 | 他の不変成分の数理witnessを保持し、Problem版は更新 |
| SP-49 | allowed/must-linkの結果にCannot-link/容量を後付け | 旧厳密最適性Claimを撤回 |
| SP-50 | 同Source digestの別Workspace Root/Read Attempt | 識別できなければ同一Root/現在資格を認定しない |
| SP-51 | #3492でRelation targetが一意でもData Owner未受理 | candidate-onlyを維持 |
| SP-52 | #3543の曖昧A/一意J選択をAnalyzer入口で利用 | 曖昧対象拒否、対象Scope/Grantを維持 |
| SP-53 | Public Slice同値でPrivate Write/MustLinkのみ異なる | 低権限候補・Score・図表・診断が秘密に依存しない |

**優先順**：Reference #3662の**exact HEAD gofmt/Go Test/Health**を確定し、次に(28)のProducer資格を1 Data×1 Functionで確認。数理Referenceの合成成功をP0のProduct資格へ読み替えない。#3492/#3543の別Consumer結果と#3321のRoot反例、#1163同一Attempt Packetは、既存#2514 Independent Expectedと#267 Acceptanceへ引き渡す。


## 31. 2026-11-09 Sourceから最適化までの権限付き接続：既存mainと検証の停止線

2026-11-09の現mainでは、[PR #3492](https://github.com/hat47x/tei-platform/pull/3492) と [PR #3728](https://github.com/hat47x/tei-platform/pull/3728) は **merged**。それぞれの実装済み範囲は、限定的な実Workspace Subject／Relation Target Inspectorと、Catalogへ既存Validationの要約を追加するものである。#3492の「偽装されたIndex SubjectでもByte/Range妥当性検査がPASSし得る」負例は、Source Indexの構造的妥当性を、実Canonical Subjectの存在や意味受理へ昇格できないことを具体化する。

この観測を元に、**既存read-only APIは使うが独自Reader/Owner Authorityは追加しない**最小縦断契約を確定する。

| 検査段階 | 現mainで確認できる具体的入口 | 合格で証明できる範囲 | 次の必要条件 |
| --- | --- | --- | --- |
| L0 Workspace load | internal/workspace.Load（operations.InspectSubjectから使用） | 当該呼出しが一つのWorkspaceを読み取って正規Worldを得た | 全Required Source/Role closure、同一Attemptの証明 |
| L1 Canonical Subject | operations.InspectSubjectのinspectCanonical（Source Index取得**前**） | kind+IDが当該Worldで存在・一意として解決される | Data→Primary OwnerとFunction Direct CRUDの意味モデル |
| L2 Source locator | BuildSourceIndex/TraceSubjectSourcesFromIndexとIndex.Validate | indexed ArtifactのByte fingerprintとRangeの整合 | Source refのCanonical意味・現在性・Role/Accepted Source |
| L3 Validation | CatalogWorkspaceのvalidation.ok/errorCount/warningCount（#3728 merged） | 既存Validate相当の構造検査の結果を要約する | Validation成功と業務Owner受理・全操作Effectを別判定 |
| L4 Business Meaning | 既存Shared Acceptance #267、型付きSource/Identity #1120 のOwnerによる受理 | **当該DataのPrimary Ownerと当該Function Direct C/U/Dの採用された意味がある場合に限る** | 具体的な根拠版・Scope・Source closure・受理主体 |
| L5 Candidate diagnostic | #1132のsubsystem-cud-candidate-v1とWorld/Facts bytes | Caller供給候補の整合と既知越境Writerの候補診断 | 候補→Acceptedへの自動昇格禁止 |
| L6 Fixed-owner exact | #3662 partition.Evaluate（candidate-only） | 固定Owner、正の重み、allowed/must-linkだけに相対した厳密最適値 | Full Source closure・非開示条件・実業務Writer・Target配備を推論しない |
| L7 Current disclosure / UI | #2516/Studio #1124（各Owner） | 現在有効なAudience向け資格が別に確認できた結果だけ表示 | Private全世界のScore/理由を公開へ転写しない |

**決定**：P0は**最小1 Data×1 FunctionのL0→L4**が同一Scope・Revision・Required Source閉包で完了できるかを判定する。L0〜L3が実装済みでも、Owner/Direct CRUDの明示宣言が受理済みSourceに存在しなければ、L4は「Productでは未結線」で止める。旧#1132 candidate factsや旧X-TEAから推測して架空のAccepted Sourceを作らない。

[Source A/B read-only Draft #3742](https://github.com/hat47x/tei-platform/pull/3742)は、Loader/Index上でSourceを改版したときの有限差分の独立レーンであり、Source IndexとWorld fingerprintがあることをSource/Manifest全閉包の真正性へ昇格しない。最新Source rootの選択・Provider raw・Owner Meaningの受理は#3262/#3477/#3479/#267の各Ownerへ返す。#3321で既に反証した物理Root/Read Attempt未識別も存置する。

## 32. 越境CUDの目標関数：本数と重み付き重要度の非混同

今回の純粋Go Referenceは**正の整数重みをCallerが明示した候補更新関係**に対してだけ、厳密な最小越境重みを計算する。これを「最小の越境CUD本数」「実業務負荷・遅延・障害確率の改善」と呼ばない。

- 本数を優先する比較では、各Direct Write occurrenceの重みを**すべて1**とする。これで計算値は既知更新関係の越境**本数**に一致する。各Function→Dataの複数distinct C/U/D occurrenceの計上単位を固定し、同一関係の重複Evidenceを二重加算しない。
- 重み付き比較では、重みの**意味（更新回数、変更リスク、障害影響、合成の意思決定重みなど）、単位、Scope、版、出典、Owner/評価者による採用**を別に開示する。例として付けた1/2/3は実測データではない。恣意的な重みの調整で候補順位が変わるため、別重みPolicyは別Problemとして扱う。
- 本数最小と重み最小は**一般には別解**。例えば候補S1では2本(各重み1)、候補S2では1本(重み5)なら、本数の目的はS2、重み付き目的はS1を選ぶ。目標関数が決まっていない状態で「最適分割」と単一表示しない。
- 巨大Subsystemへ全Functionを一括配置すると一見越境更新が減る場合があるが、境界サイズ/責務/組織能力/セキュリティ/変更容易性などのhard制約に違反し得る。未モデル化の制約は満たされたものと見なさない。将来のmulti-objectiveやParetoは追加意思決定であり、#3662の厳密性の範囲には無い。
- Public Slice相対の重み/ScoreはFull Worldの最適値と独立。秘密側の重み・存在を表示や診断で漏らさず、元のScope限定の説明責任を維持する。

## 33. 追加受入候補SP-54〜60（提案、NOT RUN）

| ID | 入力／操作 | 期待される結果 |
| --- | --- | --- |
| SP-54 | Source IndexでByte/Range妥当だがCanonical Subject IDを偽装 | Index.Validateが通り得ても、実InspectSubjectはSUBJECT_NOT_FOUNDで拒否。Owner/Direct CRUD未採用を維持 |
| SP-55 | Catalog.validation.ok=trueだがOwner宣言なし | 構造検査成功からPrimary Owner・完全Direct CRUDを推測しない |
| SP-56 | 同一Data・FunctionのSource A→B改版、Owner/CRUDの出典片方のみ旧版 | 同一Required閉包/Revisionの受理根拠が揃うまでL4へ進めない |
| SP-57 | 同一ReadのWorldFP一致だが異Workspace/Root/Read Attempt | 要求された対象・Attemptの同一性をfingerprintから補わない |
| SP-58 | Write3本を全重み1で評価し、別の重みPolicyで再評価 | 本数目的と重要度目的を区別。評価値・Candidate順位に使った重み版を示す |
| SP-59 | 同一Source関係が複数証拠で重複提示される | 同一relation occurrenceを一度だけ計上し、別distinct C/U/Dがあればその粒度を保持 |
| SP-60 | 承認済みSourceをFull Worldとして解析した後でPublic Nodeだけ隠して表示 | 公開Sliceの資格を再確認し、Private依存のScoreや順位を無権限Consumerへ流さない |

**Implementation/Plan**：先に#3662の tools/README.md 索引不足によるRepository Health失敗を修正したexact-head CIを確認する。続く実Source P0は #3492/#3728のmerged既存APIで一Data×一Functionを読み、L4受理契約の存在/欠落を記録する最小試験。公開Cursor/Studio/Source Write/Provider/DBの接続は各Ownerに引渡す。新Reader/Policy Engine/Graph/Source Resolver/Host/Acceptance SSOTを作らない。論理Subsystem境界はPhysical TargetやHash ID Namespaceと混同しない。


## 34. 実Worldの意味モデルとDirect CUD候補の停止線（2026-11-09夜）

現main（調査時点 fecf876059ba9fbab45058df9ee7c9847ba62cfa）の具体的な型・関数を静的照合した。

| 実装箇所 | 現在ある明示的な情報 | **この情報だけでは証明できないこと** |
| --- | --- | --- |
| internal/semantic/model.go : Data | Canonical ID、Attribute、Relation、CandidateKey、FunctionalDependency | Primary Subsystem Owner（少なくともこのData型にはフィールドなし）、全Business Writer |
| internal/application/model.go : Pattern | Kind、Data、Query、Capability、Relation、Interaction、ContractQueryの参照 | Pattern KindやData参照が直接C/U/Dであること |
| internal/semantic/model.go : Capability / DataEffect | CapabilityのIDと optional Effect {Kind:create/update/delete, Data} | FunctionからDataへの**直接**更新、DB Providerでのdurable Effect、Operationの完全集合 |
| internal/operations/surface.go : OperationSurface | derived/declared operation、target、binding、gaps、rule fingerprint。実行権限でないread-only投影 | Derived操作からの直接Write復元、Function全体の直CRUD/現在Grant |
| internal/modelref/derivedbridge/projection.go : FromIndex | Derived Indexからtyped graphへのread-only候補投影。Scope必須 | Business Procedure、accepted Source、Function-level CRUD |
| internal/operations/inspect.go : InspectSubject | 同一Load内でCanonical Subjectを先に存在・一意解決してからSource Indexを辿る | 複数の独立InspectSubject呼出しが同一原子的Snapshot/Read Attemptを共有したこと |

**追加の結論**：P0の最小モデルは単に「Data×Function」ではなく、もし現行Worldの明示参照を用いるなら

    application-pattern --declares-reference--> semantic-capability
    semantic-capability --declares-business-effect(create/update/delete)--> semantic-data
    semantic-data --accepted-primary-owner?--> subsystem
    function --accepted-direct-write?--> semantic-data

という4種類の**区別された関係**として読む必要がある。最初の2関係はWorldに候補となる宣言があっても、3・4番目は自動導出不可。最後の2関係が未受理のまま、Capability Effectを#1132 FunctionIOや#3662 Writeへ直接流し込まない。

特に1 Capabilityを複数Patternが参照しても、**Pattern数をWrite実発生回数・独立のFunctionIO本数として計上してはならない**。1 Capabilityにはoptional Effect 1件を格納できるだけで、関連するSQL/Trigger/Batch/副次的更新の全量を保証しない。Effect未宣言も「Readである」「Writeゼロである」とは解釈しない。Capability Effectは業務の宣言意図であって、Provider実行で生じたObserved Writeとも異なる。

「Direct」は既存の業務／Function契約として明示された関係に限る。Owner側Commandへの委譲はInvocation EdgeとOwner側Writerとして別に数え、元のFunctionが別SubsystemのDataを直接CUDしたという偽陽性を作らない。一方、Handler/HookやETL/外部バッチの実Writeが未知なら、境界を跨がないと断言しない。#1127 Operations Surfaceと#1132候補診断の既存責務を保つ。

## 35. 直列GateではなくClaim依存DAGとしてのSource/Validation/Acceptance

§31のL0〜L7は**説明上の段階**であって、全てが単一の直列Gateに成功しなければ読取自体を禁止するという意味ではない。現mainはCatalogのValidation診断とread-only Inspectorの対象解決を別APIで提供する。[Draft #3808](https://github.com/hat47x/tei-platform/pull/3808)は検証NG Workspaceでも各read-only投影がどう振る舞うかを研究中であり、Source根のみに関するValidation判定には未証明の負例候補が残る（当Draft自身の実証範囲を超えて断定しない）。

**採るべきClaim依存**：

1. Read/Diagnostic Claimは、対象Workspace・Canonical Identity・当該操作のSource読取の事実を示す。Validation findingが存在しても、Inspectorの限定読取ができた事実は保存できる。
2. Model Validity Claimは、同じ対象・必要なValidationルール・その実行結果を示す。Catalog.validation.ok、WorldFP、ModelRevisionだけでBusiness Meaningを受理しない。異なるread呼出しのFingerprint同値だけから同一Attempt/物理Rootを推定しない（#3321）。
3. Business Ownership/Function Direct CRUD Claimは、(1)(2)に加えて必要な**受理済み意味宣言・対象Scope/Source/Role closure**の根拠を持つ。Requirementごとの検証は別にする。現DataにPrimary Owner欄が無いことから、「空欄だからownerなし」とは断定しない。
4. Candidate Optimization Claimは、入力したOwner/Write/Constraint/Weightの**正確なProblem**に対してのみexact。Accepted Meaning/Current Disclosure/Production Effectは別Claimであり、自動伝播しない。
5. Public Consumer Claimは先に当該AudienceのCurrent Public Sliceを定め、秘密側のWrite/Owner/Constraint/Weight/スコアを計算してから表示だけ削ることはしない。

**既存実装からのP0 Probe**（提案、NOT RUN）：同一実Workspaceで、明示したapplication-patternのCapability IDとsemantic-capabilityのDataEffect {Kind,Data}、そのsemantic-dataを各々型付きIDで読取照合する。成功した場合も得られるのは「宣言されたCapability Business Effect Intent」まで。次に、Accepted Primary OwnerとFunctionからDataへのDirectnessを証する既存Source/Acceptanceが同Scope/Revisionで見つからなければ、unresolvedを維持する。複数Inspectorの呼出しを「一回の原子的Read」と記録しない。必要な場合は既存Loader/Projectionの同一Load内で相互参照するが、別のReader/Host/Ownership SSOTは作らない。

## 36. 新しい独立反証票（SP-61〜69）と採用優先度

| ID | 条件（追加試験はNOT RUN） | 期待 |
| --- | --- | --- |
| SP-61 | Pattern.Dataは存在、Capability.Effectなし | Direct CRUDを自動生成しない。Writeゼロとも断定しない |
| SP-62 | Pattern.Kind=command、Capability参照はあるがDataEffect未宣言 | KindだけでC/U/Dを補わず、Meaning Claimを未解決に保持 |
| SP-63 | 一つのCapabilityを複数Patternが参照 | Declared Business Effect一つをPattern数分の独立Direct Writeとして二重計上しない |
| SP-64 | Capability.Effect.update(Data A)、Data AにAccepted Primary Ownerなし | 宣言Intentは残すが、Subsystem配置の実資格を認定しない |
| SP-65 | PatternがCapability Aを参照、DataEffectはData B、Pattern.DataはData C | 別の参照と意味対象を保持し、不一致/追加更新を都合よく解決しない |
| SP-66 | Source index上の正しい参照でもBusiness Ownership未受理 | Source locator成功とCurrent Meaning資格を混同しない |
| SP-67 | 構造Validation NGでもInspectorの該当Subject限定Readは成功 | Read ObservedとModel Validity/Business Acceptanceを別ステータスで保存 |
| SP-68 | 同一WorldFPでPatternとCapabilityを異Read Attemptにて取得 | 同一atomic Snapshotの証拠なしにBinding/Owner/Writeの完全性を主張しない |
| SP-69 | Owner側Capabilityへの委譲は明示、外部ETL/Trigger更新が不明 | 元FunctionのDirect Writeと委譲を区別し、未知Effectにより全域「越境ゼロ」を主張しない |

**優先度更新**：#3662の同一HEAD Repository Health/脆弱性検査は2026-11-09時点で両方 **completed/success**（Health #42620332995、脆弱性 #42620333108）。これによりcandidate-only最適化器の局所リファレンスは回帰検証済みとなった。次のP0は**Owner/Function Directnessの正本宣言と受理根拠の所在確認**。見つからなければ、この1 Probeは未解決を示す成果として閉じ、Source/Type/Acceptanceの既存Ownerに設計Gapを返す。P1は新型の採用を決める前に、現存するCapability Effect Intentとの対応と手順/ETL/Providerとの非同値をレビュー。#3255/#3662はともにDraft・main未統合で、実業務OwnerとRuntime Effectの正式受入・公開資格は依然NOT PROVEN。
