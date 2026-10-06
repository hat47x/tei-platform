# Context-scoped Epistemic Knowledge Reference

ADR-0030 / ISSUE-000082 の最初のTEI Referenceです。

目的は、保存された文章を「事実 / 未確認」の一つの値へ潰さず、現在の作業文脈でどのように利用できるかを、由来・review・scope・freshness・Conflictを分けたまま機械判定できるか確認することです。

## 実行

```bash
go run ./tools/reference-epistemic-knowledge \
  -corpus tools/reference-epistemic-knowledge/testdata/epistemic-fixture-v0.json \
  -context tools/reference-epistemic-knowledge/testdata/context-current.json
```

出力schemaは `tei.reference.epistemic-assessment/v0` です。

## このReferenceが分けるもの

一つのconfidenceやtruth scoreにはまとめません。

```text
origin
  user / ai / system / imported / external

assertion mode
  fact / hypothesis / question / instruction / other

review validity
  native / unreviewed / valid / stale / rejected / superseded

context compatibility
  compatible / incompatible / unknown

freshness
  current / stale / not-yet-valid / unknown

conflict
  none / inactive / present
```

そのうえで、現在のcontext assemblyでの利用状態だけを `premise / candidate-only / review-required / blocked` として派生します。この派生状態は客観的Truthを表しません。

## user factとAI生成情報

user origin + fact assertion + `native` は、明示された現在scopeと一致する場合、通常の作業前提として利用できます。外部検証済みという意味ではありません。

AI originは `native` にできません。AIが生成したfact assertionは、保存されただけではreview済みに昇格せず、`unreviewed` のまま `candidate-only` です。human reviewを持つAI assertionはreview済みになり得ますが、review recordがbindingしたmeaning fingerprintと現在値が一致する必要があります。

## context generalizationをしない

assertionのscopeに必要なdimensionが現在contextに無い場合は `unknown` とし、前提へ自動昇格させません。値が異なる場合は `incompatible` です。

したがって、

```text
reviewed in project A / production
  !=
usable in project B
  !=
usable when environment is unknown
```

です。

`explicit` と `inferred` scopeも保持します。inferred scopeを利用したassessmentはその事実を出力へ残します。

## semantic editとreview

review recordは `reviewedMeaningFingerprint` を持ちます。現在assertionの `meaningFingerprint` が変わった場合、以前のreviewを現在値へ継承せず `review-required` に戻します。

このReference自身は自然言語のsemantic diffを判定しません。meaning fingerprintは対象を所有するsurface / transformation contractが供給する前提です。単なるformat変更と意味変更を区別できない場合に、ここで推測してreviewを維持することはしません。

## Conflict

Conflict relationは消しません。現在contextで両方のreviewed assertionが有効候補になるfixtureでは、どちらか一方をAIが選んで解消せず、双方を `review-required` として残します。

## Project Knowledge Health

Healthは「review済み件数 / 全件数」のpercentを主判定にしません。

`dependencies` が、どのassertionをどのconsumerがどのcriticalityで利用しているかを表します。criticalなconsumerがunreviewed / stale / conflict / scope不適合なassertionへ依存していれば `critical` を返します。

多数の低重要度memoが未reviewでも、それだけでcritical dependencyを希釈しません。

## review request handoff

reviewが必要なassertion、またはhigh / critical consumerがpremiseとして利用できないassertionはreview requestへ投影できます。

requestは少なくとも次を保持します。

- assertion identity
- target
- current review state
- context
- Evidence
- reason
- downstream impact
- reviewer candidate

reviewer candidateはAuthorityではありません。Projectionには `reviewer-candidate-is-not-review-authority` を明示します。

## fixture

`testdata/epistemic-fixture-v0.json` は、SUI / SEIへ同じ意味のfixtureを渡すためのtransport candidateです。

含むcase:

- current-context user fact
- AI unreviewed assertion
- human-reviewed assertion
- narrow-scope conditional review
- stale reviewed assertion
- superseded assertion
- reviewed conflict pair
- rejected AI assertion
- critical / high / low downstream dependency

TEI固有UIやruntime objectをfixtureの正本にしていません。

## Authority limits

このReferenceのassessment / health / review requestは、次を与えません。

- objective truth
- Canonical Acceptance
- write authority
- Runtime authority
- reviewer authority

Epistemic sidecarを削除しても既存TEI Canonical Worldが変わらないことが設計前提です。このtoolは現在Canonical loaderへ接続せず、入力sidecarからread-only Projectionを生成します。

## 今回まだ示さないこと

- SUIでのselection + 1操作やbulk review
- SEIでのrelevanceとcontext signalを使った実retrieval
- arbitrary proseからのsemantic meaning fingerprint生成
- reviewer / stakeholderの正式な組織Authority解決
- Semantic Extension / Pluginとしてのproduction registration
- persistent review event store
- Cross-repository fixtureの正式versioning

これらは最初のcontractがRepository Healthを通り、SUI / SEIの第二consumer pressureへ渡せる段階で進めます。

## 検証

focused test:

```bash
go test ./tools/reference-epistemic-knowledge -count=1
```

Repository integration authorityにはfocused testだけでなく、exact headで `bash tools/repository-health-check.sh` のfull PASSとclean pre/post stateが必要です。GitHub Actionsへは依存しません。
