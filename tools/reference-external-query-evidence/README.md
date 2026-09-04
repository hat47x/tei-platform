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
```

対応する`Database / Contract / Query / provider`のSQL realizationがWorkspaceに存在しなければ停止します。Evidenceに書かれたtable名、parameter名、raw column名、SQL等から不足authoringを補完しません。

## 実行

```sh
go run ./tools/reference-external-query-evidence \
  -workspace reference/workspace \
  -evidence /path/to/observed-evidence.json
```

成功時だけ、受け入れたtarget / providerをstdoutへ表示します。

## fail-closedの検査順

1. Evidence JSONをstrict loaderで読む
2. Workspaceを通常Loaderで読む
3. `validation.Validate`でWorkspace全体を検査する
4. Validation `Error`が一つでもあればEvidenceを受け入れず停止する
5. target + providerに完全一致するSQL realizationを一意に解決する
6. external Evidenceとrealizationを照合する

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

fingerprintだけでEvidenceの独立性を証明するものではありません。`nativeSource.evidenceRef`と`provenance.sourceRef`で、実際にどの外部資料・SQL・schema情報を観測したかを別途追跡可能にします。

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

このtoolが成立していても、#214の第一Reference実Oracle Evidenceを取得したことにはなりません。受入境界と実外部事実は分けて扱います。
