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
- Scope + Source identity + revision + accepted meaningを固定する。同名異scope、異なるSource closeure/Policy/Plugin版を混合しない。
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
