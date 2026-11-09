# Data Determinant Association Reference — Coverage Assignment

Status: **Reference business pressure / not Canonical syntax**

## 1. 目的

独立した二つのbusiness occurrenceを関連付けるDataについて、単純な「二つのFK」や「surrogate ID」では説明できないidentity / functional dependency / temporal semanticsをpressureする。

このReferenceは新しいTEI syntaxを定義しない。Core contractへbusiness determinantを昇格する前に、必要なmeaningを具体化するための入力である。

## 2. Scenario

保守契約と保守対象Assetは、それぞれ独立したbusiness occurrenceとして存在する。

`CoverageAssignment`は、ある保守契約のもとで、あるAssetをある時点から保守対象として有効化した一件を表す。

~~~text
ServiceContract
      \
       CoverageAssignment
      /
Asset
~~~

### business facts

1. ServiceContractとAssetは、CoverageAssignmentから独立して存在する。
2. CoverageAssignmentは、ServiceContractとAssetを結ぶだけの表示用join rowではない。
3. 同じServiceContractとAssetの組合せを、終了後に再度有効化できる。
4. 再有効化は過去assignmentのhistory versionではなく、別のCoverageAssignment occurrenceとして扱う。
5. `effectiveFrom`は、そのactivation occurrenceを区別するbusiness factである。
6. `effectiveTo`は同一occurrenceの終了条件であり、変更しても別CoverageAssignmentにはならない。
7. ServiceContractまたはAssetを別occurrenceへ差し替える操作は、既存CoverageAssignmentのidentity変更ではなく、新しいCoverageAssignmentを作る。
8. 監査のためにrevision番号を保存する場合、そのrevisionはCoverageAssignmentのhistory / version identityであり、business occurrence identityではない。
9. Persistenceが`assignment_id`等のsurrogate primary keyを持っても、それだけをbusiness occurrence identifierへ昇格しない。

## 3. Business occurrence identity pressure

このReferenceで検証したいbusiness identityは、概念上次の組合せである。

~~~text
ServiceContract occurrence
+ Asset occurrence
+ effectiveFrom
~~~

ただし、これはphysical composite primary keyを要求するものではない。

~~~text
business occurrence identity
  != physical primary key
  != row identity
  != history revision identity
~~~

### negative cases

- ServiceContract + Assetだけをidentityにする  
  同じ組合せを時間を変えて再有効化できるため。
- `effectiveFrom`だけをidentityにする  
  別Contract / Assetでも同時刻があり得るため。
- surrogate `assignment_id`だけをbusiness identityとする  
  business上「何が同じ一件か」を説明できないため。
- revision番号をbusiness identityへ含める  
  同じCoverageAssignmentのhistory versionを別occurrenceへ分解してしまうため。

## 4. Functional dependency pressure

Reference business factとして、同じscope内では次を検討対象とする。

~~~text
(ServiceContract occurrence, Asset occurrence, effectiveFrom)
    -> effectiveTo
    -> coverage status / terms that belong to the assignment
~~~

ただし、どのdependentが本当にbusiness semanticとして従属するかは、個別fieldの意味に依存する。

逆方向のdependencyは自動で成立させない。

## 5. Temporal pressure

時間軸はidentityと分離して確認する。

- `effectiveFrom`はこのscenarioではactivation occurrenceを区別するbusiness fact
- `effectiveTo`は同一occurrenceのvalidity終端
- revision番号はhistory version identity
- current / latest flagはbusiness occurrence identityではない
- validity期間の重複可否は別Constraint
- 同じContract / Assetの再activation可否も別business rule

これにより、datetime属性があるという理由だけでbusiness temporal semanticsを推測しない。

## 6. Relation archetype

このReferenceはcontained repeatではない。

- CoverageAssignmentはServiceContractの内部rowとしてのみ存在するものではない
- Assetの内部rowとしてのみ存在するものでもない
- ServiceContract / Assetの双方を独立したbusiness occurrenceとして参照する
- Relation先を差し替えて同じassignmentとみなすことはしない

したがって、`parent occurrence + local line number`のcontained-repeat candidateをそのまま適用しない。

## 7. Persistence independence

少なくとも次の二つのRealizationが同じbusiness meaningを維持できる必要がある。

### Realization A

~~~text
assignment_id PK
contract_id
asset_id
effective_from
effective_to
UNIQUE(contract_id, asset_id, effective_from)
~~~

### Realization B

~~~text
contract_id
asset_id
effective_from
effective_to
PRIMARY KEY(contract_id, asset_id, effective_from)
~~~

物理primary keyが違っても、business occurrence identityは同じである。

## 8. Current TEI pressure result

現行Data Design Contextは次を安全に扱える。

- physical keyをbusiness identityへ昇格しない
- temporal / history意味を名称だけから補完しない
- identity / FDをunresolvedへ残す
- Source / Evidence requirementを提示する
- derived candidateをAcceptanceへ昇格しない

一方、このReferenceのbusiness occurrence identityを**accepted Canonical meaningとしてauthorするfirst-class contract**はまだ無い。

したがって、このReferenceはCore primitive追加を直ちに承認するものではないが、次のrepresentational gapを示す。

~~~text
source-backed business occurrence identity
  -> current Core contract cannot express directly

source-backed scoped functional dependency
  -> current Core contract cannot express directly
~~~

## 9. Core contract候補へ求める能力

将来のcontract候補は、少なくとも次を表せる必要がある。

- business occurrence identityとhistory / version identityの区別
- determinant componentとしてAttributeだけでなくRelation先occurrenceを参照
- composite determinant
- uniqueness scope
- temporal scope / validity
- Source / Contract trace
- alternative Persistence realizationからの独立
- Shared Acceptanceへ渡せるproposal identity

具体syntaxはこのReferenceでは決めない。

## 10. Acceptance pressure

Core contract候補を採用する前に、最低でも次を比較する。

1. contained repeat — Order / OrderLine
2. changeable reference — Intake / Customer
3. independent association — CoverageAssignment
4. temporal / history variation — CoverageAssignment revision

共通部分だけをCoreへ上げ、類型固有の意味を一つの万能primitiveへ押し込まない。
