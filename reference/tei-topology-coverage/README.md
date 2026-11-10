# 複合図表の限定不在判定：read-only合成Reference

- **状態**：評価用Reference。新Core型・正式なProjectionWitness/Source Reader・製品API・権限契約ではありません。
- **設計根拠**：[トポロジー・複合図表・開発パイプライン設計 §9〜13](../../plan/design/TOPOLOGY_COMPOSITE_VIEWS_AND_PIPELINE_2026-10.md)。
- **既存責務**：#1120 ModelRef、#1124 Studio Consumer、#2620 標準データ＋Role、#943 Build、#951 Provider、#2516 Disclosure、#267 Source Acceptance。第二Resolver、Host、Source正本は設けません。

## 実験の問い

「図に表示されないので存在しない」と言えるのはいつか。図に含まれる要素集合、Relationの**すべてのroleと端点**、表と図の被覆、閉じたscope、参加した同一版、Source/Plugin/Provider解決、表示省略、opaque処理を別々に扱います。

`project`は同一合成SnapshotからGraphとTableを導出します。Graph上のrelationノード＋role付きEdgeと、表のrole配列は同じ多項関係を表します。`selectView`はview IDをscope付きsubjectへ戻す**候補**だけ返し、InspectやEditは呼びません。`boundedAbsence`は完全性witnessが供給された**閉じた合成集合の範囲だけ**`absent-in-declared-snapshot`を返し、不足・非開示・版混在なら`unknown`です。

この関数はwitnessの**真正性も網羅性も証明できません**。`subjectsComplete: true`などはfixtureの申告値であり、実際のTEIで主張するには独立したSource Reader、Composition/Plugin closure、Disclosure Policy、Target-bound Evidenceによる検証が必要です。外部Builder、OS、DB、ネットワークの未知副作用がないという証明にも使えません。

## 実行

```sh
node --test reference/tei-topology-coverage/coverage.test.mjs
```

Node.js 22以上の組込みtest runnerだけを用います。

**Relationの識別上の注意（2026-11-06追補）**：FixtureではRelationの`scope + id`を端点Subjectとは独立したqualified identityとし、GraphのRelation node・Role edge・TableのRelation recordに同じ`relationKey`を保ちます。端点のscopeはRelationの宣言scopeの根拠にはなりません。同一relation IDが別scopeに現れること、別scopeを結ぶrole、Graph/Table間のrelation typeの齟齬、Relation ID/Scopeの再束縛を負例に含めます。これは既存TEI ProductのRelation Scope ABIを定義したことにはなりません。

実験ケースは、同kind/ID異scope、複数View、三項関係role保持、閉じた集合内の条件付き不在、欠落role、Graph/Table欠落、opaque/省略/未固定Reader/開示不明、Source版・Graph/Table公開版の混在、レイアウトだけの差分、欠落端点、非開示下の現在情報、Table行のscopeとqualified keyの不一致、Graph旧view IDの別scope誤結合、Relation ID指定の取り違えです。Graph/Tableの各Occurrenceが整合していない場合は、欠落の証拠を作るだけでなく、存在するという合成Claimや選択候補も拒否します。

**実行証拠（限定）**：GitHub Actions [synthetic topology P1 #42155243921](https://github.com/hat47x/tei-platform/actions/runs/42155243921) は、exact head `577bbe60d961c58e142502ab5835c9a7610214df`、Node 24.21.0で **18/18 PASS**。試験した `coverage.mjs` のGit blob `5cc3ca120e2995650bcaba1121a2416b3be8010a`、`coverage.test.mjs` のblob `6ece714d49a7b6f8dfa4ff7d376dfe4eb61e7f4d` と試験後clean treeをActions記録で確認しました。通常のRepository Health / 脆弱性確認と、**このReferenceのfocused試験が実際に走ったこと**は別証拠として記録します。

**追加の実証（Relation scopeとtype）**： [dedicated CI #42156720965](https://github.com/hat47x/tei-platform/actions/runs/42156720965) は、exact head `760597b2aaddde1bf3990ee8f57a35deb9bdb9ea`、Node 24.21.0で **26/26 PASS**。対象コードGit blob `c20ceb222a690d486ce5b8f95782dba6c84d9a13`、試験blob `9ec51ea2518b05499405bc86e21398b91856084a`。旧18/18は過去版。今回も合成関係の帰属・表示整合を確認しただけで、実Readerによる完全性を証明しません。

**Query条件の範囲（2026-11-06追補）**：本合成Evaluatorが解析するのは`kind/scope/type/id`だけです。Relationの`scope`はRelation自身の**宣言scope**であり、端点scope・Role・依存の推移・ETLのWindow・BuildのTarget/Selected Inputを意味しません。入力に`participantScope`／`role`／`where`／`window`／`subjectKey`など未対応の条件がある場合は、**存在・不存在どちらのClaimも出さず`unknown:unsupported_query_predicate`**で拒否します。未対応条件を黙って落とすと、指定範囲で存在しないという偽の結論につながるためです。正式Query DSLや実Readerの受入規則ではありません。

**回帰試験（新規）**：対応範囲外のRole/Participant/WHERE/Window条件による偽の不存在と、対応外条件を無視した偽の存在を別々に検査します。[専用GitHub Actions #42159700190](https://github.com/hat47x/tei-platform/actions/runs/42159700190)（code head `4352ef7ff1a253acb9334bfec7f974dfdb66e0d3`）のNode 24.21.0 focused testは**29/29 PASS**。Implementation blob `37682bdaeb7df5dd2ba62344fea6140dd5546aec`、test blob `f2ff56da9b15a15d4e0eeb90863e63af342bd62b`。これ以降の文書更新headに対するCIは別に記録します。

**Claim上限**：`present-in-snapshot`も`absent-in-declared-snapshot`も現在のTEI Source、Provider、実ビルド、実転送、実Authorityを意味しません。ReferenceにはSource/Plugin実ReaderやPolicy評価器は含まれません。現物入力と現在の公開資格を独立確認せずにConsumerへ流用しないでください。非公開・保留・不明の違いが低権限UIに現れてよいかは既存#2516へ委ねます。

**未実証**：実Go ModelRef/Source Index・TEI CLI出力との結線、Providerによる実閉包証明、実Graph Renderer/Workbench、複数Window/旧DOM、実SHO/AI、外部Effect、認知負荷削減。#1124 M0/M1を代替するテストではなく、#2352 P1の後続評価用合成試験です。実験を製品化する前に、既存型からの変換のlosslessnessと現在のReader/Policy資格を独立検証します。

**未開示時の結果均一化（合成限定）**：開示資格のないFixtureでは、View/Readerの版やGraphの形を検査する前に`unknown:disclosure_unqualified`を返し、低権限Fixture間で「版の古さ」「要素欠落」などの診断差が結果に現れないようにしています。これは内部関数のsynthetic W0/W1等値試験であり、DOM/ARIA・件数・Network・Tool JSONの非干渉や実Policy判定を実証しません。

**専用CIの追加結果（31ケース）**：[Actions #42160035359](https://github.com/hat47x/tei-platform/actions/runs/42160035359) はNode 24.21.0で **31/31 PASS**、失敗0。実装blob `82ad9c0ef8e5943ee55c1611e212a06e525819ac`、テストblob `be80612ee541b0d1f2abb67e076c4000bb015f1f`。Go側の現物ReaderとそのQuery意味へは未接続です。

## 実CLI直接参照の限定接続（2026-11-06）

`direct-trace.mjs`は、既存Goの`tei trace`が実Workspaceから出力した **`tei.cli.trace/v0`** の成功報告を、**direct-edge候補**としてだけ観測するread-only Referenceです。新しいSource ReaderやModelRef Resolver、Relation Registry、権限判定器ではありません。

```sh
node --test reference/tei-topology-coverage/direct-trace.test.mjs
go run ./cmd/tei trace \
  -workspace reference/workspace \
  -scope reference-observation \
  -subject-kind semantic-relation \
  -subject-id sales.order.customer \
  -direction both -mode full > /tmp/tei-topology-trace.json
go run ./cmd/tei trace \
  -workspace reference/workspace \
  -scope alternate-observation \
  -subject-kind semantic-relation \
  -subject-id sales.order.customer \
  -direction both -mode full > /tmp/tei-topology-trace-alternate.json
node reference/tei-topology-coverage/verify-cli-trace.mjs \
  /tmp/tei-topology-trace.json /tmp/tei-topology-trace-alternate.json
```

この固定例では、実CLIの**incoming** `order_entry.customer_reference → sales.order.customer`、**outgoing** `sales.order.customer → customer.customer`を独立のNode Consumerで照合します。Source navigationの`located`は読取り位置の情報に過ぎず、編集権限・Owner意味受理にはならない。Traceの`Edge.Relation`は**直接Edgeの種別文字列**であり、`declaringScope + relationID`や任意の多項Roleの正本ではないため、`observeDirectTrace`は`relationEntityIdentity:not-provided`、`multiRoleClosure:unproven`、`sourceAndPluginClosure:unproven`、`absentClaim:unknown`を保持します。**このアダプターの結果は`boundedAbsence`へ渡しません。**

対象Graph/Traceがfull一ページ・scope一致・候補read-onlyでない場合、`unknown`で止めます。0行でも「Relationshipが存在しない」にはしません。`ProjectionUnresolved`、Source位置不明、SourceIndex未閉包、非公開Policy、過去のGrantはそれぞれ独立の資格です。実CLIの生JSONにはSourceパスやEvidenceを含む可能性があるため、**許可済みpublic sliceを持たない外部Consumerへそのまま出力しない**。本アダプターはCurrent Policyを検証しません。

**実施済み証拠**：[専用Actions #42205912384](https://github.com/hat47x/tei-platform/actions/runs/42205912384)。exact head `6b99fed018dc9cc9c44bd0cee2031aecb1939a7b`、Go 1.26.10、Node 24.21.0で、合成Graph/Table **31/31 PASS**、CLIアダプターの合成負例 **7/7 PASS**、実`reference/workspace`を通したGo `tei trace`→Node Consumer **PASS**。Source改変なしのclean treeをworkflowが検査。これが証明するのは、固定Workspaceの直接参照2本を実CLI JSONから同一Consumerが読んだことまでです。多項Relation完全性・全Source/Plugin閉包・実Disclosure/Policy・実WebView2/Studio・ETL/Build実行・認知改善・Owner受理は **NOT PROVEN**。

**追加の実CLI負例：Caller ScopeはSource宣言ではない**：`derivedbridge.FromIndex(d, scope)`は呼出し側指定のscopeをDerived Index Projectionへ付与します。`reference-observation`と`alternate-observation`の2回の`tei trace`で、**Source／World fingerprintおよびscope以外のDirect Edgeが同一、qualified keyとselector-bound snapshot fingerprintが異なる**ことを別言語Consumerで照合しました。したがって`observeDirectTrace`は`scopeOrigin:caller-supplied-projection-scope`と明記し、CLIで見えるscopeを元Sourceのdeclaring scope、Principal権限、正本Relation scopeへ昇格しません。

**exact-head CI**：[Actions #42206842519](https://github.com/hat47x/tei-platform/actions/runs/42206842519)、コードhead `8bf4347fa587a66bbc500a8a2fd5927d150437d2`、Go 1.26.10実CLI **2実行**→Node独立比較PASS、Node 24.21.0のGraph/Table 31件＋CLI境界7件PASS、失敗0。実Sourceの全scopeや未読込Root、Policy、実Studio UIは **NOT PROVEN**。

**後続のFit/Gap**：実`tei trace`は直接宣言Edgeを取得できる一方、合成Referenceの任意の多項Relationに必要な独立の宣言scope・relation ID・role列は一般には復元できない。今あるGoの`TraceWorkspace`と`SourceBridge`を再利用し、実Sourceに必要な意味・roleが存在するケースを別Ownerで特定した後にだけ、追加のMapping／標準構造を検討する。#1124 M0/M1と#2831 G4の受入を代替しない。

## 2026-11-07 SourceNavigationのKey照合（実CLI観測アダプター）

既存GoのSourceBridge Navigationは `{key,status,references,authorityLimits}` を返します。Referenceの `observeDirectTrace` は従来、`status:"located"` のみで `sourceNavigation:"located-navigation-only"` と表示し、**別Subjectや別scopeに結び付いたSource位置であっても見かけ上Locatedにできる**問題がありました。

修正は同じCLI応答中の `trace.subject` と `source.key` の**scope/kind/id一致**を検証し、located時に空でない`references`を要求します。Sourceがunresolvedの場合はDirect Edge候補の存在を保持しつつ `sourceNavigation:"unresolved"` のまま。欠けたSource／別Key／located-without-references／未解決情報の不正Shapeは `unknown` とし、誤帰属に成功しません。

検証には合成負例（他Subject/他scope、Key欠落、located空参照、未解決の分離、ProjectionUnresolved不正型）と、**実CLIで取得したJSONのSource Keyだけを別Subjectへ改変するnegative control**を併用します。既存 `tei trace`、SourceIndex、ModelRef、SourceBridgeは一切変更しません。

本チェックは**same-packet整合**であり、独立Source Reader／同一時点の全Source閉包／本番Policy／Current Grant／編集権限／Accepted Meaningを証明しません。研究計画[PR #2966](https://github.com/hat47x/tei-platform/pull/2966)（main統合済み）のIntent／Realization／Assessmentを、今回の局所Source帰属検査へ適用しただけです。より詳しいFit/Gapは[設計§15](../../plan/design/TOPOLOGY_COMPOSITE_VIEWS_AND_PIPELINE_2026-10.md)を参照してください。
# 実CLI direct traceの同一性境界（read-only Reference）

**目的**：旧[Draft #2881](https://github.com/hat47x/tei-platform/pull/2881)で反証済みの、実`tei trace`→別言語Consumerの限定的な同一性検査だけを、2026-11-08時点のmainへ分離移植。旧Draftの「合成Graph/Tableの完全性」31ケースは**このPRへ移植しない**。新しいTEI Core/関係正本/Source Reader/Policy Engine/Hostを設けない。

## 何を照合するか

既存 `cmd/tei trace` の `tei.cli.trace/v0` と、`operations.TraceWorkspace`のRead-only結果をNodeから検査する。

- **Trace selector**：scope付きSubject／方向／full 1ページ／row cardinality／解決・未解決を明示。0件・不完全ページからRelation不存在を結論しない。
- **GraphとCLI継続Tokenは別**：現行ModelRefの`trace.graphFingerprint`をGraph候補の正本として読み、旧`trace.snapshotFingerprint`は**一致する場合だけ**許すdeprecated aliasとする。ページ継続を表す外側`result.snapshotFingerprint`とは混同しない。full 1ページで`nextOffset/continuation`が現れる場合も拒否。Rule fingerprintの存在も確認する。
- **Source navigation**：`source.key`が`trace.subject`に一致。Located時に非空のSource Index referencesが必要で、各参照の`subject.kind/id`も選択対象に一致することを検査する。Source Indexにはscopeがないため、Sourceから独立のscopeを推測しない。
- **Workspace identity**（現mainの#3181）：`workspace.manifest.path`と`sha256`、`formatVersion`、任意の`name`をLoad時の由来情報として要求。二つのCLI観測を結合する際は、Manifest/Name/FormatVersionとWorld・Source Index Fingerprintを**別々に**照合する。同じWorld/Source FPでもManifestが違えば`workspace_manifest_mismatch`に停止する。**Manifestの観測はSource完全性、現在のSourceの独立再読取、受理済みモデル、権限、Lockを意味しない**。
- **Scope由来**：既存`derivedbridge.FromIndex(d,scope)`のscopeはCaller提供。実Workspaceを異なる2つの`-scope`で読み、選択子bound fingerprint・qualified Keyだけ変わることを確認する。Source自身の宣言scopeとは言わない。
- **直接Edgeの資格**：`Edge.Relation`は種別であり、独立した業務Relation ID／所属Scope／多項Roleではない。`relationEntityIdentity:not-provided`、`multiRoleClosure:unproven`、`absentClaim:unknown`、`authority:read-only-candidate`を維持。
- **解決先Refの整合**：`row.edge.to`の宣言Subject（Kind/ID／指定されていればScope）を、`row.resolvedTo`のGraph解決Keyと照合。矛盾すれば`unknown:resolved_edge_target_mismatch`。これは実Sourceを独立再読込したことではない。

## 再現

```sh
node --test reference/tei-topology-coverage/direct-trace.test.mjs
go run ./cmd/tei trace -workspace reference/workspace -scope reference-observation \
  -subject-kind semantic-relation -subject-id sales.order.customer \
  -direction both -mode full > /tmp/tei-trace-a.json
go run ./cmd/tei trace -workspace reference/workspace -scope alternate-observation \
  -subject-kind semantic-relation -subject-id sales.order.customer \
  -direction both -mode full > /tmp/tei-trace-b.json
# Manifest名だけを変えた、業務Sourceが同じ別Workspaceを一時作成
mkdir -p /tmp/tei-topology-other-workspace
cp -a reference/workspace/. /tmp/tei-topology-other-workspace/
sed -i 's/^workspace: tei_reference$/workspace: tei_reference_copy/' \
  /tmp/tei-topology-other-workspace/tei.workspace.yaml
go run ./cmd/tei trace -workspace /tmp/tei-topology-other-workspace \
  -scope reference-observation -subject-kind semantic-relation \
  -subject-id sales.order.customer -direction both -mode full > /tmp/tei-trace-c.json
# 別の物理ディレクトリへManifestを含めて完全コピー（同一バイト）
mkdir -p /tmp/tei-topology-identical-workspace
cp -a reference/workspace/. /tmp/tei-topology-identical-workspace/
cmp reference/workspace/tei.workspace.yaml \
  /tmp/tei-topology-identical-workspace/tei.workspace.yaml
go run ./cmd/tei trace -workspace /tmp/tei-topology-identical-workspace \
  -scope reference-observation -subject-kind semantic-relation \
  -subject-id sales.order.customer -direction both -mode full > /tmp/tei-trace-d.json
node reference/tei-topology-coverage/verify-cli-trace.mjs \
  /tmp/tei-trace-a.json /tmp/tei-trace-b.json \
  /tmp/tei-trace-c.json /tmp/tei-trace-d.json
```

**注意：** 実CLIの生JSONにはSource pathやEvidenceを含む可能性がある。公開資格を持たない外部Consumerへ渡さず、CI内の一時ファイルだけで検査する。Node Referenceは実Disclosure Policyや現在Grantを評価しない。CLIのSource Navigationを**同じ応答内で検査**し、二つのCLI応答のLoad-time Manifest／World／Source Indexの対応を比較するだけで、独立Reader／同一Attemptの別観測による真正性を証明しない。`workspaceLoad`は内部Reference用の比較メタデータであり、そのままPublic Consumerへ公開しない。

**今回の実測（2026-11-08）**：[専用CI #42434859063](https://github.com/hat47x/tei-platform/actions/runs/42434859063) はGo1.26.10実CLI 3回（同Workspace・別Caller scope 2件と、同Source・異Manifest Workspace 1件）＋Node24.21.0 **17/17 PASS**。**別WorkspaceでもWorld／Source Index fingerprintとSnapshot tokenが一致した**（`snapshot-token-equal=true`）が、ManifestのName/SHAは異なり、`compareObservedLoadContext`は`workspace_manifest_mismatch`で拒否。実装Git blob `bdff1a072eff1242e89329b1390b362b935eff50`、test Git blob `1b7b988c8a4535ffa7473aef61fe8438c5dc7e2c`、実CLI検証script Git blob `36de67555d9403133cd746891a61a67d3fb53a7e`。この結果はコード版`cfa1fe1ccad79d60e7c854020f5624e6300d174d`時点の実行であり、文書更新後headとmainの全CI資格は別に確認する。

**完全コピーの残余Gap（2026-11-09）**：別ディレクトリにManifestと全Sourceをバイト単位でコピーしても、CLI応答の`WorkspaceIdentity`・World・Source Index・Snapshotの値が一致し得ます。この実験は、入力引数でRootが異なると知っているハーネスに対し、Consumerが**同じ観測値のみから物理Root同一性を判断できない**ことを示します。`compareObservedLoadContext`の結果には`physicalRootIdentity:not-provided`と`readAttemptIdentity:not-provided`を明示し、比較は`same-observed-load-context-only`のread-only候補までとします。実System Root/Reader identity、Current Grantや業務受理を認定しません。追跡と証拠は[設計§11.7](../../plan/design/TOPOLOGY_COMPOSITE_VIEWS_AND_PIPELINE_2026-10.md)を参照してください。

**2026-11-09最新main追随**：Main `e348bd987626de2c14bddb45f852fae7457b51a7`のGraph/Workspace Traceの別Token契約と`Ref`解決整合をNode負例に追加。独立CIは正確なheadの結果を別に記録。詳細は[設計§11.8](../../plan/design/TOPOLOGY_COMPOSITE_VIEWS_AND_PIPELINE_2026-10.md)。Graphの形が正しいことは、Workspaceの物理Root・Read Attempt、Current Grant、全Role/Source閉包の資格にはならない。

**検証していないこと：** #1124の実Workbench 4入口・旧DOM、#2516の実Policy/二世界、#267のOwner意味受理、#2620のRole全解析/必要Root、#2831のセルフビルド実工程、#2542の配送Receipt、[統合済み研究#2966](https://github.com/hat47x/tei-platform/pull/2966)の人間/AI認知負荷比較。旧#2881の合成Graph/Table結果と本PRの実CLI結果は別資格として保持。

**関連設計**：[トポロジー・複合図表](../../plan/design/TOPOLOGY_COMPOSITE_VIEWS_AND_PIPELINE_2026-10.md)。Intent／Realization／Assessmentを一つの「存在している」の判断へ混同しない。試験とソース更新を実際に確認してからmain採用を検討する。
