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

表示面では、Subsystem ER図の内部Dataとexternal参照、CRUD Matrixのeffective/potential/direct/observedを表示区別し、Relation所属Scopeと端点Scopeを保持する。表示圧縮によるedge欠落、非開示/未読取/不明、Renderer能力欠落を「不存在」に読み替えない。表現の損失・逆編集は[#3250の表示引継ぎ](https://github.com/hat47x/tei-platform/issues/3250#issuecomment-6703937718)・[#3236](https://github.com/hat47x/tei-platform/pull/3236)の専門Ownerに委ねる。

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
