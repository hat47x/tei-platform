# reference-external-query-evidence

`external-observed` DatabaseのContract Queryについて、外部で観測した物理Evidenceと、TEI側で別途明示authoringしたSQL realizationが一致するかを検査するtoolです。

第一Referenceでは`master_data.customer / find_by_id`の実Oracle Evidence受入に利用します。

## 原則

このtoolはEvidenceからCanonical WorldやSQL realizationを生成しません。

```text
observed external facts
        ↓
Evidence JSON
        ↓ compare only
validated Workspace + explicit SQL realization
        ↓
source-fixed acceptance receipt
```

対応する`Database / Contract / Query / provider`のSQL realizationがWorkspaceに存在しなければ停止します。Evidenceに書かれたtable名、parameter名、raw column名、SQL等から不足authoringを補完しません。

このtoolの成功は**external Evidence artifactと明示authoringの受入照合が成立したこと**を示します。Oracleへactual readしたこと自体は証明しません。actual read / Semantic result Evidenceは#214の後続工程で別途取得します。

## 実行

clean repository revisionで実行します。

```sh
go run ./tools/reference-external-query-evidence \
  -workspace reference/workspace \
  -evidence /path/to/observed-evidence.json \
  > /tmp/tei-external-query-evidence-acceptance.json
```

Repository内の新規fileへ直接redirectすると、そのfileがcommand開始前に作られてworking treeをdirtyにするため、Gate B受入れではworking tree外へ出します。

## Repository sourceを固定する

Gate Bへ採用する受入Evidenceは、後から同じTEI sourceへ戻れる必要があります。

そのためtoolはexternal Evidenceを読む前に次を確認します。

```bash
git rev-parse HEAD
git status --porcelain
```

working treeがdirtyなら受入れを停止します。

Evidence / Workspace / SQL realizationの照合後にも同じ状態を再確認し、次の場合は成功receiptを出しません。

- 実行中にHEADが変化した
- 実行後にtracked / untracked差分が残った

Repository状態の観測規則は`tools/internal/repositoryevidence`を利用し、#310 deterministic Repository Evidence、#255 generated artifact Evidence、#308 PostgreSQL Provider Evidenceと同じclean revision規則へ揃えます。

## acceptance receipt

成功時は人向け文字列ではなく、小さいJSON receiptをstdoutへ出します。

概形:

```json
{
  "evidenceNature": "external-query-evidence-acceptance",
  "repositoryRevision": "<git commit>",
  "workingTreeClean": true,
  "reproducibleRevision": true,
  "postRunRepository": {
    "checked": true,
    "repositoryRevision": "<same git commit>",
    "revisionStable": true,
    "workingTreeClean": true
  },
  "evidenceArtifactFingerprint": "sha256:<hex>",
  "target": "<Database / Contract / Query identity>",
  "provider": "oracle",
  "accepted": true
}
```

receiptへ次は複製しません。

- physical object identity
- physical parameter / raw result column
- physical type / NULL情報
- provenance本文
- read-only mechanism本文
- Evidence file path
- credential / secret

`evidenceArtifactFingerprint`で、どのexternal Evidence artifactを実際に検査したかを追跡します。

### Evidence artifact fingerprint

`evidenceArtifactFingerprint`は、toolが読み込んでstrict parseした**Evidence JSON fileのexact bytes**に対するSHA-256です。

```text
sha256:<hex>
```

空白や改行だけを変更しても別artifactとして別fingerprintになります。これは意図的です。受入receiptは意味的に近い別fileではなく、実際に検査したartifactそのものを識別します。

このfingerprintは後述の`nativeSource.fingerprint`とは役割が異なります。

## fail-closedの検査順

1. repository HEAD / working treeを確認する
2. dirtyなら停止する
3. Evidence JSON exact bytesを読み、artifact fingerprintを計算する
4. Evidence JSONをstrict loaderで読む
5. Workspaceを通常Loaderで読む
6. `validation.Validate`でWorkspace全体を検査する
7. Validation `Error`が一つでもあればEvidenceを受け入れず停止する
8. target + providerに完全一致するSQL realizationを一意に解決する
9. external Evidenceとrealizationを照合する
10. repository HEAD / working treeを再確認する
11. 同じclean revisionへ固定できない場合は停止する
12. JSON acceptance receiptを出力する

Warningだけを新しい拒否条件にはしません。通常Validationが`Error`と判定したWorkspaceをEvidence受入の前提にしない、という境界です。

## Evidence JSONで必要な情報

少なくとも次を明示します。

- `target`
  - Database
  - Contract
  - Query
- `provider`
- 実際に観測したphysical objectのkind / identity
- Contract inputとphysical parameterの対応
- 実raw result columnとSemantic attributeの対応
- native SQL sourceのEvidence参照とfingerprint
- physical factsのsource / provenance
- Reference用に匿名化・縮約等を行った場合のprocessing説明
- read-onlyを実効的に保証するmechanismとEvidence参照

unknown fieldや一つ目のJSON値に続く別JSON値は拒否します。

## native SQL fingerprint

`nativeSource.fingerprint`は、Workspace Loaderが`ContractQuerySQLRealization.Source`として保持する**実行SQL本文**との一致を確認するためのfingerprintです。

外部SQL fileをWorkspaceへ読み込む際、loaderはfile全体を読み、先頭・末尾の空白を`strings.TrimSpace`で除いて`Source`へ格納します。その後、Evidence検査ではその`Source`に対してSource Indexと同じSHA-256 fingerprint形式を使います。

```text
sha256:<hex>
```

したがって、これはGit blob SHAや、末尾改行を含むraw file bytesのfingerprintとは同一とは限りません。

`nativeSource.fingerprint`は「観測したnative sourceが現在authoring済み実行SQLと一致するか」、`evidenceArtifactFingerprint`は「どのEvidence JSON artifactを受理検査したか」をそれぞれ固定します。相互に代替しません。

fingerprintだけでEvidenceの独立性を証明するものでもありません。`nativeSource.evidenceRef`と`provenance.sourceRef`で、実際にどの外部資料・SQL・schema情報を観測したかを別途追跡可能にします。

## read-only Evidence

SQLが`SELECT`に見えることだけではread-only保証にしません。

Evidenceには、たとえば次のような実際の強制境界を記録します。

- read-only principal / credential
- Database permission
- read-only session / transaction
- write APIを持たないadapter

採用方式をTEI側から先に固定せず、実環境で確認したmechanismとその根拠を記録します。

## 非目標

- external Database全体のTEI管理化
- 未観測の物理object / column / parameter / SQLの推測
- EvidenceからCanonicalやrealizationを生成すること
- SQL文字列だけからread-only安全性を推論すること
- synthetic fixtureをactual external asset Evidenceとして扱うこと
- credentialやsecretそのものをEvidenceへ保存すること
- acceptance receiptをactual Oracle read Evidenceとして扱うこと
- Evidence本文をacceptance receiptへ複製すること
- dirty / source変化中のrunをGate B受入Evidenceへ採用すること

このtoolが成立していても、#214の第一Reference実Oracle Evidenceを取得したことにはなりません。受入境界、外部観測事実、actual readは分けて扱います。
