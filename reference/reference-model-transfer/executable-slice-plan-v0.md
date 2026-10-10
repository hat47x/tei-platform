# Reference Model Transfer — 実行可能な代表シナリオのEvidence契約（候補 v0）

- 状態: 設計候補。実装済み・実行済み・同等性確認済みではない
- 正本: [Reference Model Transfer Evidence Gate](../../plan/REFERENCE_MODEL_TRANSFER_GATE_2026-10.md) の **RMT-4 — Executable representative equivalence**
- 担当: [GitHub #1142](https://github.com/hat47x/tei-platform/issues/1142)
- 入力: RMT Source Adapter / typed Evidence / cross-view Projection
- 参照: [cross-view accounting](cross-view-accounting-v0.json)、[loss ledger](loss-ledger.json)
- 比較観測の担当: GitHub #940。公開境界の担当: GitHub #941

## 1. 次に証明すること

成熟したモデルをXMLとして読めることと、業務システムとして動かせることは異なる。次の最小の縦断を**一件の合成read-onlyシナリオ**で成立させる。

```text
source facts / declared I/O
    ↓ source adapter + scoped evidence
business / function / data cross-view（read-only）
    ↓ candidate proposal + explicit human decision
authoritative TEI definitions（reloadで再現）
    ↓ selected target / plugin realization
TEI Driver execution
    ↓ actual observation: result + before/after DB + external effect + authority
evidence claim（TEI-only observed）
```

**旧Runtimeとの比較は、この縦断とは別のGate**である。旧Runtimeのactual traceがない間は、TEIの実行に成功しても「移植同等」にはしない。Source上で宣言されたFunction-level C/R/U/Dを、Business Actionのactual effectやDB差分のoracleに昇格させない。

注意: GitHub #1031では「RMT-4 cross-view」という作業名が使われているが、Evidence Gateの番号では三視点cross-checkがRMT-3、実行比較がRMT-4である。本書は**Gate側のRMT-4**だけを扱い、#1031のgraphを再実装しない。

## 2. Evidenceの単位

一つのScenarioは、業務目標、選定根拠、開始状態、操作・入力、実行主体と権限、観測可能な結果、実行Target、未解決点により定義する。source由来の意味、設計者が新しく受理した意味、Runtimeが実行して観測した結果は、同じ属性に上書きしない。

| 独立したEvidence | 固定するもの | それだけでは言えないこと |
| --- | --- | --- |
| Source selection | exact artifact digest、source profile、local/private selection Evidence | Canonicalで受理された業務意味 |
| Source observation | type付きfact、token set、derivation rule fingerprint、authority limit | Function-level操作がAction固有effectであること |
| Acceptance | proposal ID、base revision、明示Decision、受理後source revision、reload結果 | Driverが実際に動いたこと |
| TEI execution | target、plugin/lock、code/fixture revision、input/precondition、結果、DB前後・外部effect、transaction、権限 | 旧Runtimeとの一致 |
| Original execution | exact original version、input/precondition、結果、DB前後・外部effect、transaction、権限、観測手法 | TEIとの一致、観測していない意味の確定 |
| Comparison | 二つの実行Evidence、前提の整合、比較軸、差異、判定者と根拠 | 画面やclassの完全互換性 |
| Human review | 誰が何を受理したか、scope、revision | 実行の代わりになる機械的な正しさ |

**fingerprintは内容の固定であってproducer / observer Conformanceではない。** 受理Authority、観測Authority、Runtime Authorityを区別し、互いの存在だけで昇格しない。

個々のScenarioへ振る`scenarioId`はTEI側の新規IDとする。旧ID、SortKey、XML ordinal、opaque source tokenをCanonical identityへ転用しない。

## 3. 比較の判定規則

比較は、観測状態から独立した明示的な状態機械とする。

1. **not-evaluated**: まだ比較していない。Source構造だけの段階もここに留める。
2. **incomparable**: precondition / dataset / version / authority / observation granularityが比較可能でない。欠けた軸を一致扱いにしない。
3. **consistent-on-observed-axes**: 両側にactual execution Evidenceがあり、同じ前提・比較軸に限って整合を確認した。全体的な互換性ではない。
4. **different-on-observed-axes**: 同じ前提と比較軸に明示された差がある。Source / TEIどちらが誤りかを自動判定しない。
5. **unresolved**: 実行がunknown、観測の信頼性や権限に疑義が残るなど、比較の結論を出せない。

`consistent-on-observed-axes`を選ぶには、originalとTEIのactual trace、比較対象のprecondition整合、少なくとも一つの比較可能な軸、差分計算の規則とEvidenceをすべて必要とする。片側が実行されていない、画面メッセージだけ、sourceにCRUD宣言があるだけ、という場合は**必ず拒否する**。

観測軸は以下を混合せず保持する: (a) business result、(b) persistent data delta、(c) external effect、(d) transaction/rollback/cancel、(e) authorization/resource boundary、(f) repeat/retry/unknown、(g) interaction observation。比較軸によっては`not-applicable`を許容するが、`not-observed`を`not-applicable`へ丸めない。

## 4. 最初のread-only slice

最初のTEI-only witnessには権利を確認済みの**独自の合成データ**を使う。旧モデルのTable/Function名やfield本文を直接取り込まない。

- 業務操作: 単一条件での照会。業務上の意味と画面操作手順は別に定義する。
- 構造: Business Action→Function/Interaction→declared Data accessの最小トレースを作る。Actionなしの「SQLを実行しただけ」ではslice完了としない。
- 定義: 新規のTEI Semantic / Plugin declarationからcandidateを作り、Shared Acceptanceを通し、reload後のCanonicalで実行する。AdapterからCanonicalへ直接書かない。
- 実行: snapshotを固定し、戻り値、件数、DB前後、外部effect、principal / policy、runtime artifact / lockを記録する。
- negative: SELECT宣言なのに実際の副作用がある、権限なしで読める、前提が変化した、source/revisionがstale、という状態では成功判定しない。
- 比較: #940のread-only original actual traceが得られ、preconditionが対応づけられた場合にだけ比較する。それまではTEI-only statusを維持する。

旧モデルには148 TaskActionがあり、うち6件はFunction I/O参照がない。これは**manual/external candidate**であり、すべてが必ず手作業だとは確定していない。このclassも後続の代表選定で残す。

## 5. 後続のScenario classと反証したい失敗

class IDはPlugin / Referenceが宣言する拡張可能な識別子とし、Coreの固定enumにしない。

| 次のclass | 優先して反証すること |
| --- | --- |
| 単票保守 | update表示の成功とDB commitが異なる / 権限外の変更が可能 |
| header-detail | headerだけcommitされる / detail失敗時のrollback不整合 |
| 明細一括 | 部分成功、cancel、retry、unknownを単純successに丸める |
| 帳票 | 画面・印刷の表示を業務データへの書込みと取り違える |
| batch / native Script | declared CRUDとScript actual effectが異なる |
| Function call | 宣言されたcall graphを実行順序・再帰に読み替える |
| Validation / derived | derived valueやproseを人のAcceptanceなしに確定する |
| manual / external | Function非参照のActionを削除し、業務フローが途切れる |

最初から全classの旧Runtime実行を待たない。未観測classは未観測と記録し、実行済みclassのcoverageから全Functionの互換性を推定しない。

## 6. 公開境界と保存先

`scenario-manifest-v0.schema.json`は構造候補であり、個別モデルのcontentsではない。公開側のScenario recordには新規`scenarioId`、source digestとschema version、意味を持たないopaque Evidence ref、class、claim status、unresolvedを保存する。元のBusiness名・旧Function ID・record値・画面文言・Script・秘密情報・exact structural locator・local token一覧を保存しない。

選定に必要なlocal/private EvidenceはRepository外に保持する。単なるpath名のhashでも辞書照合できるため、公開tokenをsource local name/pathから単純hashしない。GitHub #941の境界を明示的な許諾なく広げない。

TEIが生成したsynthetic fixtureにもoriginal modelの翻案が混入していないことを独立に点検する。

## 7. 最初の実装受入条件（#1142）

- independent synthetic fixtureのsource→proposal→decision→reload→target executionが**実動**し、exact revisionへ追跡できる。
- 実行前後のDB stateとexternal effectを明示し、画面成功表示だけでcommitを主張しない。
- source/derived CRUDからAction固有actual effectを生成する経路をnegative testで拒否する。
- original Evidence欠落、precondition drift、stale revision、観測unknownで比較をfail closedにする。
- manual/external candidateを「対応Functionなし=削除」としない。
- 原モデルの利用条件が未確定のまま本体や派生fixtureを配布しない。
- Code変更は別Draft PRに限定し、trusted full checkoutのexact-head focused / `tools/repository-health-check.sh` PASS、pre/post clean Evidenceがない限りmergeしない。
- 本文だけを作った状態では実装済みとは判定しない。

### 停止線

初期段階の達成は**TEI-only end-to-end witness 1件**であり、originalとの一致は別に実証する。形式が似たこと、画面が出たこと、生成AIが説明できたことを「移植できた」の代わりにしない。

## 8. Candidateのadversarial qualification（2026-10-10）

本契約のJSON Schemaは**untrusted manifestに書ける内容の上限**を定めるものであり、第三者が報告したdigestや観測結果を信頼済みにする仕組みではない。特に`teiExecution.status=observed`や`comparison.status=consistent-on-observed-axes`は、Schema上の整合条件を満たしても単なる自己申告である。実際にclaimとして発行するには別のEvidence resolver / witness検証が必要である。

- `comparison`の一致・差異判定には、TEIと旧Runtimeのactual trace、受理済みsource、比較軸、整合したpreconditionを要求する。
- `preconditionAlignment.status=aligned`は両Runtimeの観測が存在する場合に限り、片側実行前に一致を先取りしない。
- `publicationReview.status=passed`をmanifest自身から宣言することを禁止する。公開承認は別Authorityが実施し、candidate外で記録する。
- 公開`sourceProfile`は構造的なtokenに限定し、local pathや本文を許さない。ただし識別子への意図的な埋込みは構文制限で検出できないため、独立した公開検査が必要である。
- negative fixtureは出典digest、受理Decision、実行receipt、precondition、比較軸、原model配布情報の漏洩を中心に作る。

`tools/reference-rmt-scenario-manifest/test_manifest_schema.py`にSchema適合・不適合のadversarial testを追加した。Python実行時には`jsonschema>=4.18`が必要である。外部依存をTEI productionへ追加しない。

~~~bash
python3 -m unittest discover -s tools/reference-rmt-scenario-manifest -p 'test_*.py'
go test -count=1 ./tools/reference-shared-acceptance -run '^TestRMTSyntheticReadOnlySlice'
~~~

**既存TEI機構を再利用する合成縦断候補**として、Shared Acceptanceの既存scratch workspaceとDriverのread-only Query実行を接続するGo testを追加した。RMT Evidenceのtyped attachment、明示ReviewにbindしたDecision、source reload、明示Query mapping、合成Storeの前後観測、およびmapping欠落の拒否を一つのtestで確認する構成である。

このtestがPASSしても次は未達として残す。

1. RMT Source Adapterで実際に構成した三要素のsource-backed relationから、受理する候補意味までの一貫した対応づけ。
2. 実Databaseのread-only state / transaction / security boundaryのactual観測。
3. 旧Runtimeを実行した結果との同一条件での比較。
4. 複数Evidence receiptに対する独立したobserver / producer authority検証。
5. trusted full checkoutのexact-head focused test / full `tools/repository-health-check.sh` PASS。

既存の合成fixtureで実行できる範囲を最初に使うが、これを新規のReference Model全体の転写成功へ拡大解釈しない。actual Gate未取得のPRはDraftのまま保持する。
