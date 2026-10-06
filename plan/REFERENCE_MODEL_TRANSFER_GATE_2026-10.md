# Reference Model Transfer Evidence Gate — 2026-10

- 状態: **In Progress / Evidence contract**
- 性格: Product Value横断のReference / Conformance pressure
- 担当Issue: [ISSUE-000078](issue/000078-reference-model-transfer-gate.md)
- 対象Requirement: RQ-001 / 002 / 003 / 004 / 010 / 011 / 012 / 013 / 021 / 022 / 025 / 026 / 029 / 031 / 032 / 033
- Product Gate A / B / Cとの関係: **独立した横断Evidence Gate。Gate Dではない**

## 1. 目的

TEIが小規模な自作Referenceだけに適合した基盤ではなく、長年の実務設計で使われてきた成熟した業務システムモデルを受け止められることを実証する。

最初のbenchmarkには、渡辺幸三氏のX-TEA / XEAD系Reference Modelである「CONCEPTWARE／受注生産」を用いる。目的はX-TEA互換製品を作ることでも、X-TEAのschemaをTEI Coreへ複製することでもない。

確認したいのは次である。

> X-TEA上で表現されているData、Function、Business、UI、Rule、Script、Runtime binding等を、意味を黙って捨てたり、TEI側の都合で推測補完したりせず、TEIのCanonical meaning / Realization / Native asset / Projection / Unresolvedへ分類して転写できるか。

この実証では、旧環境で問題にならなかった暗黙の前提もそのまま引き継がない。単一Desktop Driver、同一Java Runtime、file path、共有Session、暗黙transaction等によって吸収されていた意味を、multi-Target、AI agent、複数Provider、長期保守へ持ち出しても安全かという観点から再評価する。

## 2. なぜProduct Valueに直結するか

現在のTEIは、`order_entry`と`repair_intake`を中心に、意味のSSOT、Driver、第一Web Target、Shared Acceptance、第二Applicationで主要境界を確認している。一方、Referenceの規模と業務機能の広がりはまだ小さい。

成熟した既存モデルの転写は、Data / Function-Interaction / Business Procedureを同じ業務システムの異なる視点として保持できるか、画面・帳票・batch・script・Database I/O・手作業を一つの実装型へ潰さず扱えるか、既存資産をnativeなまま残しても意味・依存・実行権限へ戻れるかを一度に圧迫する。

さらに、同じ意味をDriverやWeb等の複数Projectionへ出してもTarget固有実装が第二正本にならないか、AIや別担当者が元のModeler / Editor / Driverの暗黙知なしに根拠と未解決点へ再入場できるか、新しい意味体系が必要になった場合にCoreへ場当たり的に追加せずPlugin / DSLとして合成できるかも確認できる。

したがって本Gateは、単なるImporter互換試験ではなく、TEIの中心価値に対する規模・複雑性・異質性のpressure testである。

## 3. Source baselineとprovenance

### 3.1 渡辺幸三氏のGitHub source

設計・実装上の一次sourceとして、次のRepositoryをexact revisionで読む。

| Source | Baseline |
| --- | --- |
| `xead/XeadModeler` | `68f1535b1c5dd80a327dfced37be401ecd3093c9` |
| `xead/XeadEditor` | `ee19a27609b2749110ec39b62d4540693347fe09` |
| `xead/XeadDriver` | `6c077773b74c6d7dbcc253835cf8d347e4ffb1ef` |

Repository:

- <https://github.com/xead/XeadModeler>
- <https://github.com/xead/XeadEditor>
- <https://github.com/xead/XeadDriver>

これらのsource codeはBSD-3-Clauseで公開されている。TEIでは、Modeler / Editor / Driverの実装をsemantic authorityとしてコピーするのではなく、XML要素の実際の解釈、matrix導出、Function I/O、Script / Session / transaction等の暗黙挙動を確認する一次Evidenceとして使う。

### 3.2 Reference Model / manual snapshot

利用者が保持するReference Modelとinstaller同梱manualを、Repository外のsource snapshotとして用いる。内容そのものをTEI Repositoryへ転載しない。

| Source snapshot | SHA-256 |
| --- | --- |
| `MfgOrder160627(1).zip` | `2e2324ba6e7a2ec3e52f783011bfb655520589483d1f3f5890a932bd6a0e0aec` |
| `MfgOrder.xead` | `31af1fe317c661f57c94c69285b7b3e91c7c2fcad940cd7639956aff781088a9` |
| `MfgOrder.xeaf` | `f4b73b36abd8398aa25ad86ecd7570ae0334e7fd93c3ee204c5a05bddebdb754` |
| `XTeaModelerInstaller1_4_50__Files.zip` | `257a333a8db9319fb5da49689751ccb90bbaa384021bf1e6b8e2623ac3cc74bc` |
| `XTeaDriverInstaller230127__Files.zip` | `f79f6825a6d6dce3c4c41f3c7028b92c3e18fd33f06e9090bee064f79cfbfe32` |

installer同梱の`XeadFileFormat.pdf`、`XeafFileFormat.pdf`およびHTML manualも補助sourceとする。

Reference Model本体にはGitHub source codeとは別の利用条件・著作権表示がある。再配布や大量転載を前提にせず、初期段階ではfingerprint、集計値、TEI側の独自mapping / fixture、loss / unresolved EvidenceだけをRepositoryへ残す。公開Repositoryへモデル内容を取り込む必要が生じた場合は、先に利用条件を確認する。

## 4. 最初のsource inventory

`MfgOrder.xead`の観測値を、scaleの把握と転写coverage確認に使う。数値自体をProduct KPIにはしない。

| 要素 | 観測数 |
| --- | ---: |
| Subsystem | 12 |
| Table | 56 |
| Relationship | 80 |
| Function | 227 |
| Task | 62 |
| TaskAction | 148 |
| TaskFunctionIO | 407 |
| Instance | 35 |
| SubjectArea | 7 |
| DataflowNode | 92 |
| DataflowLine | 81 |

`MfgOrder.xeaf`には203 Functionが存在する。基本設計側227 Functionと実行仕様側203 Functionは1:1ではない。Editorで詳細化された補助Functionや、基本設計だけに存在するFunctionがあるため、TEIでは「設計上のFunction」と「実行Realization」を同一identityへ機械的に畳まない。

このinventoryは転写完了数を競うためではなく、source constructが黙って消えていないかを確認する母集団として使う。

## 5. 「転写できた」の定義

本Gateでは、XMLを別形式へ変換できただけでは転写成功としない。

source上の各意味要素を、少なくとも次のいずれかへ分類できることを要求する。

| Classification | 意味 |
| --- | --- |
| Canonical | Target / Runtimeから独立したTEIの意味として保持する |
| Realization | Canonical meaningを特定技術で実現するbinding / implementationとして保持する |
| Native preserved | SQL / Script / report layout等をnative assetのまま保持し、Source / Contract / Capabilityへ接続する |
| Derived projection | Matrix、CRUD、graph、screen等、authoritative sourceから再導出する |
| Presentation-only | layout座標や説明順等、意味の正本にしないが必要ならProjection metadataとして保持する |
| Unresolved | sourceだけでは意味を一意に判断できず、推測せず残す |
| Unsupported with evidence | TEIが現時点で保持できないことを明示し、lossとして記録する |

**silent dropを許さない。**

unsupportedをゼロにするためにCoreへ概念を足すのではなく、実際のReferenceから必要性を確認し、Plugin / DSL、native preservation、Projectionで解けないものだけをCore変更候補にする。

## 6. Evidence段階

### RMT-0 — Source / License / Format baseline

- exact source revisionとartifact fingerprintを固定する
- Xead / Xeaf formatと実装上の解釈を照合する
- 公開可能なEvidenceとRepository外に留めるsourceを分ける
- model contentの再配布可否を未確認のままpublic fixtureへコピーしない

### RMT-1 — Complete structural accounting

`MfgOrder.xead` / `MfgOrder.xeaf`のsource constructをinventory化し、各要素を「転写分類」のいずれかへ割り当てる。要素数の一致だけで成功にしない。ID、reference、ordering、Function I/O、Script dependency等のrelationも追跡する。

最低限、次の視点を別々にaccountする。

~~~text
Data
  Table / Field / Key / Relationship / DataType / Instance

Function / Interaction
  Function / FunctionType / Panel / Spool / Table I/O / Web I/O
  Function call / parameter / return / Script

Business
  Role / Task / TaskAction / TaskFunctionIO
  SubjectArea / Dataflow / manual or external work

Runtime / Environment
  Database / Session / Login Script / shared Script
  credential / path / transaction / external call

Projection
  CRUD / matrix / layout / report / menu
~~~

### RMT-2 — Semantic transcription and loss ledger

構造をTEI表現へ転写し、source nodeごとに次へ戻れるようにする。

~~~text
source artifact + source locator
        ↓
transcription decision
        ↓
TEI subject / native asset / realization / unresolved
        ↓
derived projection / validation
~~~

Source Indexから元sourceへ戻れない転写はcompleteとしない。

特に、source proseからsemantic meaningを抽出した場合はAI推論だけでAccepted Canonicalへ昇格させず、Proposal / Unresolvedとして扱う。

### RMT-3 — Three-view cross-check

RQ-025 / RQ-026に沿い、Data / Function-Interaction / Businessの三視点を相互検証する。

最低限、Business Actionが参照するFunction / Interactionの存在、Functionが使うData / Relation / Validationへのtrace、Function / Table I/OからのCRUD再構成、Subsystem境界を越えるC/U/D、Data変更からFunction / Businessへのimpact、Functionの無いmanual / external Actionの保持を確認する。

traceが無いことを「依存なし」と読み替えない。

### RMT-4 — Executable representative equivalence

全Functionの画面を再実装する前に、各主要実行型を代表するScenarioを選び、TEI Driver / Targetで意味境界を再現する。

少なくとも、read-only照会、単票保守、明細一括、header-detail、帳票、batch / Script launcher、Function call、Validation / derived field、manual / external continuationを含める。

ここで比較するのはpixel一致やJava class一致ではない。business input / precondition、referenced meaning、command / result / error、persistent or external effect、observation、transaction boundary、authorization / resource boundary、unresolvedが同じ意味として説明できることを確認する。

### RMT-5 — Full-reference pressure

RMT-1〜4で得たmappingを、特定の数FunctionだけでなくReference Model全体へ適用する。

完了時に、source construct全体についてCanonical / Realization / Native / Derived / Presentation / Unresolved / Unsupportedの分類ができ、silent dropがないことを確認する。

実行Evidenceは全203 FunctionをE2E操作することを初期完了条件にはしない。ただし、あるFunction classだけが未検証のままなら、そのclass全体を「実行可能性確認済み」と主張しない。

## 7. 旧X-TEAで暗黙でも成立した点を再検討する

転写は旧構造の模倣ではない。次は、source / manual / Reference Modelからすでに確認できたmodernization pressureである。

| 観測した構造・挙動 | TEIで防ぎたい失敗 | 初期方針 |
| --- | --- | --- |
| `SortKey`が表示順だけでなくimport時の対応付けにも使われる | order変更やimportでsemantic identityが変わる | legacy key / provenanceとCanonical identityを分離する |
| Xead FunctionとXeaf Functionが1:1でない | helper / runtime Functionを業務Functionへ誤昇格する | design meaningとRealizationを分離する |
| Relationshipのfield対応をfield数・DataType等から補う経路がある | 同型fieldが複数あると誤った関係を確定する | explicit field mappingを要求し、不明はUnresolvedにする |
| CRUDはFunction / Table I/Oから明示される | Business Procedure文言やPattern名からCRUDを推測する | CRUDはexplicit operation authorityからderiveする |
| Scriptから任意SQL / TableOperatorを実行できる | 宣言CRUDとactual effectが乖離しても見えない | declared effectとnative Script effectをcross-checkし、未知effectを残す |
| commit / rollbackがcaller、return code、logout等のcontextへ依存する | Targetやagentを変えるとtransaction意味が変わる | Commandのtransaction / effect boundaryを明示する |
| Database URL、user、password、path等が実行仕様へ同居する | Semantic sourceへsecret / Environment authorityが混入する | Environment binding / Secret / Runtime Authorityを分離する |
| Login Script、共有Script、Session attributeが暗黙依存になる | 個別Functionだけ読んでも挙動を再現できない | dependency / provenanceを明示し、native codeならrequired Capabilityを付ける |
| Business RoleとDriver user / menu権限が別層にある | actorをAuthorization Roleへ変換する | Business actor / principal / policyを分離する |
| Business flowのStoreがDB以外の帳票・物・人・待ちを表せる | すべてをData/Tableへ押し込む | external / physical / human stateを別境界として扱う |
| Instanceの順序が時系列にも説明順にも使われる | presentation orderをstate transitionへ誤変換する | Dataset / Snapshot / Scenario / Expectation / Presentation Orderを分離する |
| 外部Table定義をpath + IDで参照・同期する | copy / syncが第二正本になり、Relationだけdriftする | semantic identity / version / provenanceによる再利用へ置換する |
| Field description等のproseにRule / derived logicが含まれる | AIが文章から暗黙に確定仕様を発明する | extractionはProposal、Accepted meaningには明示受理を要求する |
| XF100等のFunctionTypeがDriver/UI方式と強く結び付く | Java Driverの画面型をTEI Coreの業務型へ固定する | Pattern / Interaction meaningとTarget realizationを分離する |
| `parmMap` / `returnMap` / Session attributeが動的 | magic keyの意味が呼出側知識へ閉じる | 安定した業務契約はtyped input / resultへ上げる |
| layout座標・report文字配置がFunction定義と近い | visual realization変更がsemantic changeになる | Semantic / Functional / Visual Evidenceを分離する |
| 単一Driver前提ではretry / unknown / idempotencyを明示しなくても動く | network / agent / distributed実行で二重副作用や誤成功が起きる | RQ-018〜020のCommand / Observation / retry / unknown境界へ接続する |

この表を「X-TEAの欠陥一覧」として扱わない。旧Runtimeの利用条件では合理的だった設計も含む。TEIでは対象Runtimeと運用範囲が広いため、暗黙のままでは持ち運べない点を再検討する。

## 8. X-TEA固有schemaをCoreへ昇格させない

Reference Modelが成熟していることと、そのschemaがTEI Coreに適していることは別である。

次の順で解決する。

~~~text
existing TEI meaning
    ↓ insufficient
existing Plugin / DSL extension mechanism
    ↓ insufficient
native asset + typed binding
    ↓ insufficient
Reference-local extension candidate
    ↓ repeated pressure
Core common mechanism candidate
~~~

`Subsystem`、`FunctionType`、`Task`、`DataflowNode`等の名前を、対応するsource要素があるという理由だけでCore型として追加しない。

一方、既存TEIの表現では意味が失われることがReferenceで確認された場合は、X-TEA側で従来問題にならなかった点であってもTEI側の欠落として扱う。

## 9. Evidence artifact

Repositoryへ保持するEvidenceは、原モデルの大量転載ではなく次を基本とする。

~~~text
source-baseline.json
source-inventory.json
transcription-map.json
loss-ledger.json
cross-view-diagnostics.json
scenario-manifest.json
execution-evidence/
~~~

各artifactには、少なくともsource artifact digest、transcriber / mapping contract version、TEI repository revision、accepted / derived / unresolvedのnatureを保持する。

`transcription-map.json`はsource contentsのコピーではなく、source locatorとTEI側の対応関係を表す。著作物の内容を公開する必要がある場合は、利用条件確認後に別判断する。

## 10. 判定

### Ready

次をすべて満たしたとき、本GateをReadyと判定できる。

1. RMT-0〜RMT-3がactual artifactで成立している。
2. source inventory全体にsilent dropがなく、Unsupported / Unresolvedも明示されている。
3. RMT-4で主要実行型を代表するScenarioが、TEI自身のDriver / Target / Evidence境界で再現されている。
4. RMT-5でmappingをReference Model全体へ適用し、特定sliceだけへの過適合でないことを確認している。
5. X-TEA固有schemaを理由なくCoreへ複製していない。
6. native Script / SQL等を保持する場合、そのSource、依存、Capability、権限、effectの未解決点へ辿れる。
7. original X-TEA runtimeでのactual behavior比較が必要な項目は、actual実行したか、未取得Evidenceとして明示されている。
8. Reference Modelの利用条件と公開Evidenceの境界が確認されている。

### Readyであっても主張しないこと

- TEIがX-TEA/XEADのdrop-in互換実装であること
- 全X-TEA modelを自動変換できること
- すべてのReference Model contentをTEIが再配布できること
- 全203 Functionの人手E2E操作を完了したこと
- production-ready ERP / 生産管理製品が完成したこと
- X-TEAの設計判断よりTEIが優れていること

## 11. Webチャット外へ渡す作業

本チャットで進められるsource解析、mapping設計、Repository文書化、fixture設計はISSUE-000078で継続する。

次はactual environmentが必要なため、GitHub Issueへ切り出す。

- [GitHub #940](https://github.com/hat47x/tei-platform/issues/940) — original X-TEA Modeler / DriverでReference Modelを起動し、代表Scenarioのactual behavior、transaction、Script effectを取得する
- [GitHub #941](https://github.com/hat47x/tei-platform/issues/941) — CONCEPTWARE Reference Modelの公開Repository内再利用・転載条件を確認する
- TEI importer / execution codeを実装した後、対象exact-headを固定してtrusted local cloneで`tools/repository-health-check.sh`を実行する
- 人による業務設計レビューが必要になった場合、AIの独立レビューと区別して記録する

これらが未実施でもRMT-0〜3のresearchは進められるが、actual比較が必要なclaimを推測で埋めない。
