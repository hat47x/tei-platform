# Generic Capability移行設計

- 状態: 評価待ち（未採用）
- 対象: ISSUE-000080 / GitHub #951
- 関連: RQ-019 / RQ-023 / RQ-024 / RQ-032 / RQ-033

この文書は、既存の`internal/plugin.CapabilityKind`からgeneric Capability Contractへ移る場合の境界を整理する設計候補です。作業の進行はISSUE-000080、採用判断は将来のADRを正とします。この文書だけを根拠にproduction schemaやRuntime Authorityを変更しません。

## 1. 目的

TEIでは標準機能もPluginとして提供する方針ですが、現行RuntimeはCapabilityの種類を`CapabilityKind`へ固定列挙しています。新しいCapabilityを追加するたびにCoreを変更する構造は、Plugin-firstの拡張性と両立しません。

一方、`CapabilityKind`を自由な文字列へ変えるだけでは不十分です。Contractの意味、Provider適合性、Target、Permission、Resource、Sandbox、実行時Context、transportの責任範囲が曖昧になるためです。

現在のReferenceでは、Coreへ新しいenumを追加せず、Source AdapterとContract Queryをgeneric Capability Contractとして解決・実行できるところまで確認しています。Runtime側にも、既存Authorityからgeneric Runtime Capability identityへ一方向に投影するReferenceがあります。

次の問題は、既存Capabilityをどの順序で移し、どの時点で旧`CapabilityKind`をAuthorityから外せるかです。

## 2. 現在の結論

移行はRepository全体を一度に切り替えず、Capabilityごとに進めます。

各Capabilityについて、ある時点の実行Authorityは一つだけにします。移行期間に旧`CapabilityKind + CapabilityID`とgeneric Contract / Providerを二つの独立した正本として同期しません。

初期段階では既存Plugin / Environment定義をAuthorityに保ち、generic表現を一方向に導出します。generic側のContract、Provider、Conformance、Runtime Authorityがproduction境界で成立したCapabilityだけ、明示的なAcceptanceを経てAuthorityをgeneric側へ移します。

Authorityを移した後にlegacy表現が必要なら、genericなAccepted stateから互換projectionを導出します。旧定義へ自動fallbackして二系統を同時に有効化する構成には戻しません。

## 3. 分けて扱う五つの面

### 3.1 Contract meaning

Contractは「何を呼び出すか」を表します。

最低限、logical identity、version、request / result / failure semantics、effect、shape identity、Conformance profileを持ちます。実行Contextを要求する場合はContext Slot Contractへの参照を持つ候補とします。

Contract identityはPlugin package、artifact、process、HTTP等のtransportへ依存させません。

### 3.2 Provider realization

ProviderはContractを「何で実現するか」を表します。

Plugin ID / version、implementation identity、exact artifact、Target realizationをContract meaningから分離します。同じContractをDesktopではbuiltin、WebではWasm等で実現しても、Contractそのものを別の意味へ変えません。

Providerの自己申告だけでは利用可能とせず、AcceptedなConformance Evidenceへ戻れることを要求します。

### 3.3 Runtime / Tool Authority

ContractとProviderを解決できることは、実行Authorityを意味しません。

Application RuntimeではEnvironment、Permission、Resource、Lease、Sandbox、Artifact Verification等の既存Authorityを通します。Authoring ToolではApplication RuntimeのEnvironment Authorityを流用しません。

generic Runtime identityを導入する場合も、新しいPermission StoreやResource Brokerを作らず、既存Authorityへ接続します。

### 3.4 Invocation Context

authorization、verified Principal、tenant、session等のcurrent execution stateはContract meaningへ埋め込みません。

Contractは必要なContext Slotのidentityを宣言し、Hostがcurrent bindingを注入する設計候補を評価中です。actual credentialやsession valueはContract fingerprint、Plugin Lock、Runtime Capability identityへ固定しません。

### 3.5 Physical transport

in-process、process、HTTP、Wasm等のtransportはContract identityと別です。

transportがrequest内の自己申告identityからProviderやAuthorityを決めてはいけません。上流で確定したContract / Provider / Target / Authorityからdispatch planを導出し、transportはそのplanを運ぶ構造を基本とします。

## 4. 単一Authorityの原則

移行の中心となる不変条件は次です。

```text
legacy authority
  -> generic projection candidate
  -> verification / conformance
  -> explicit acceptance
  -> generic authority
  -> legacy compatibility projection when required
```

双方向同期はしません。

legacy authorityからgeneric candidateを作る段階では、旧Plugin declarationやEnvironment設定のfingerprintを残します。旧Sourceが変わればprojectionをstaleにします。

Authorityをgeneric側へ移した後は、旧設定を暗黙fallbackに使いません。rollbackが必要なら、以前のAccepted authority stateへ戻す別の明示変更として扱います。

## 5. Capabilityごとの移行状態

移行状態は全Repository共通の一つのversionではなく、Capability単位で考えます。

| 状態 | Authority | generic側の扱い | legacy側の扱い |
| --- | --- | --- | --- |
| L0 | legacy | 無し | 正本 |
| L1 | legacy | read-only projection / shadow validation | 正本 |
| L2 | legacy | generic resolution / invocationを既存Runtime Authorityの後ろで利用 | 正本 |
| G1 | generic | Contract / Provider / Authorityが正本 | 必要時だけderived compatibility projection |
| G2 | generic | legacy enumを使わない | 対象Capabilityのlegacy pathを削除可能 |

L1やL2を「genericへ移行済み」とは扱いません。旧Authorityが正本である限り、legacy declarationの変更がgeneric executionを無効化できなければなりません。

G1へ進むときだけ、Authorityの所有者を明示的に切り替えます。

## 6. 最初の既存Runtime Capability候補

最初の移行候補には`contract-query-executor`を使うのが妥当です。

理由は、既存のEnvironment Plan、Permission / Resource境界、Contract Queryの入力・結果型、Runtime provenanceがすでに存在し、generic Invocation側でも第二consumerとして利用しているためです。また、現在のReferenceはgeneric Invocationをread-onlyへ限定しており、Contract Queryはこの停止線と整合します。

Source AdapterはCore enum追加なしで新しいCapabilityを導入できる反例として重要ですが、Authority scopeがauthoring-toolです。既存Application Runtime Capabilityの移行を実証する対象とは分けます。

Database ProviderやMigration Providerは、Resource、DDL / DML、副作用、運用Authorityの範囲が広いため、最初の移行対象にはしません。

## 7. L0からL2まで

### L0: legacyだけを使う

現行の`plugin.Definition.Provides`、`CapabilityKind + CapabilityID`、Environment Authorityをそのまま正とします。

generic Contractを追加しただけで実行経路を変えません。

### L1: 一方向projection

authoritativeなlegacy declarationからgeneric Contract / Provider candidateへ一方向に投影します。

ここでは次を確認します。

- Plugin ID / versionが一致する
- exact implementation artifactへ戻れる
- legacy Permission declarationとgeneric Execution Profileが一致する
- Contract / Provider Conformanceが成立する
- projection元のSource fingerprintを保持する
- projection元が変わった場合はstaleになる

generic側の変更をlegacy側へ書き戻しません。

### L2: generic Invocationをlegacy Authorityの後ろで使う

generic resolver / InvocationでContractとProviderを選んでも、実行可否は既存Environment Planが決めます。

```text
generic Contract / Provider resolution
        |
        | identity must match
        v
existing Environment Plan
        |
        + Permission
        + Resource
        + Sandbox
        + Artifact verification
        v
existing Runtime invocation
```

generic Provider identityだけでPermissionやResourceを得ないことをnegative testで固定します。

L2は重要ですが、まだAuthority移行ではありません。generic側は既存Runtime Authorityを迂回できないconsumerです。

## 8. G1へ進むためのGate

既存Capabilityをgeneric Authorityへ切り替える前に、少なくとも次を確認します。

### Contract

Contract identity / version / fingerprintがexactに再現でき、request / result / failure semanticsが旧Capabilityの実利用と矛盾しないこと。

shapeを使う場合は、shape artifactとAccepted Shape Authority Evidenceへ戻れること。

### Provider

Provider identity、Target realization、artifact、Conformance Evidenceがexactに結び付くこと。

同じlogical Providerに複数realizationがある場合、Targetや明示wiringなしに一つを選ばないこと。

### Runtime Authority

generic Runtime identityからPermission、Resource、Lease、Sandbox、Artifact Verificationへ到達できること。

旧Authorityとgeneric Authorityを同時に有効化した状態を拒否できること。

### Invocation Context

Capabilityが必要とするcurrent execution contextをContract meaningと分離できること。

consumerがauthorization等をsemantic requestへ埋め込んで自己申告しても、Host bindingを上書きできないこと。

### Transport

in-process / process / remote transportの違いでContract / Provider identityを作り直さないこと。

transport request自身をProvider selection Authorityへ昇格させないこと。

### Effect

read-only以外のCapabilityは、transaction、prepare / commit、compensation、retry、result validation failureの扱いが決まるまでG1へ進めません。

PermissionやSandboxだけで副作用の原子性を満たしたとは扱いません。

### Replay / rollback

AcceptedなContract / Provider / Authorityを後から再現できること。

generic pathが利用できないときにlegacy pathへ自動fallbackせず、以前のAccepted stateへ戻す明示手順を持つこと。

## 9. G1以後のAuthority

G1ではgeneric Contract / Provider / Runtime Authorityを正本にします。

legacy `CapabilityKind + CapabilityID`を必要とする既存Hostが残る場合、compatibility envelopeはgeneric Accepted stateから導出します。legacy IDを再び独立したProvider選択やPermission判断の正本にはしません。

互換projectionには、元のgeneric identityとprojection ruleのfingerprintを残します。generic ContractやProviderが変わればcompatibility projectionもstaleになります。

この段階で初めて、対象Capabilityについてlegacy declarationへの新規writeを停止できます。

## 10. G2とenum削除

`CapabilityKind`全体を一度に削除しません。

あるkindをCoreから削除できるのは、少なくとも次を満たした後です。

- そのkindのproduction consumerがgeneric Contractへ移行済み
- Environment Authorityがgeneric identityから再現できる
- process / builtin等のHostがlegacy kindを選択Authorityとして使っていない
- observability / provenanceからgeneric Contract / Providerへ戻れる
- rollbackが旧enumの暗黙fallbackへ依存しない
- Repository内でそのenum値を増やさずに新規Providerを追加できる
- compatibility projectionだけが残る場合、削除時期を明示できる

複数のkindを一つの大きなgeneric contractへまとめる必要もありません。既存kind一つが複数Contractへ分かれる可能性も、複数kindが同じContract familyへ収束する可能性もあります。実際のconsumerから判断します。

## 11. 新しいCapabilityの扱い

移行判断が成立する前でも、Source Adapterのように既存Core enumへ自然に対応しない新しいTool Capabilityは、Referenceでgeneric contractとして評価できます。

ただし、productionへ新しいCapabilityを追加するときに「generic化が未完だから」という理由だけでCore enumを増やすと、移行対象が増え続けます。

将来のADRでは、少なくとも次の規則を採用候補とします。

> 新しいCapabilityKindの追加は原則停止し、generic contractでは表現できない具体的なAuthority / execution gapがReferenceで確認された場合だけ例外として再検討する。

この規則自体は、まだ採用判断ではありません。

## 12. Lockを一つに統合しない

Semantic Lock、Capability resolution、Plugin Lock、BuildArtifactLock、current execution contextは、固定する対象が異なります。

generic Capability移行を理由に一つのUniversal Lockへまとめません。

```text
Contract / Provider selection
  -> exact resolution identity

Runtime Plugin selection
  -> Plugin Lock / Environment Authority

Build provenance
  -> Build Plan / BuildArtifactLock

current execution context
  -> invocation-time binding / receipt

Semantic meaning
  -> Semantic Lock
```

相互にfingerprintを参照することはできますが、更新頻度やAuthorityの違う情報を一つのlockへ押し込みません。

## 13. Physical packagingとの関係

generic Capabilityへの移行は、Pluginごとの物理artifact分割を要求しません。

同じContractをTargetごとに別artifactへ実現したり、複数のlogical Capabilityを一つのartifactへ統合したりできる余地を残します。

一方、現行Runtime Plugin Lockが表現できるartifact粒度と、将来のTarget packagingが必要とする粒度が一致するとは限りません。Web向けlazy loadingやWasm分割を理由にPlugin Lock schemaを先に増やさず、actual Target buildで必要になったidentityだけを追加します。

## 14. Production変更の順序

production codeへ進む場合も、最初から`CapabilityKind`を削除しません。

推奨する順序は次です。

1. generic Contract / Provider identityをproduction境界で表現できる最小primitiveを決める。
2. `contract-query-executor`一件をlegacy-authoritativeなL1 / L2で通す。
3. generic identityから既存Runtime Authorityへ到達するE2Eを取得する。
4. legacyとgenericのAuthority一致を機械確認する。
5. G1への切替案をADRとして審査する。
6. 一件だけgeneric-authoritativeへ切り替える。
7. provenance、failure、rollbackを確認する。
8. 次の既存Capabilityへ広げる。

この順序なら、generic contractの採用判断とenum削除を同じ変更へまとめずに済みます。

## 15. 現在のEvidence

mainには、少なくとも次のReferenceが存在します。

- Core enum追加なしのCapability Contract resolution
- Xead / Xeaf Source Adapterの実consumer
- Source AdapterとContract Queryのgeneric Invocation
- request / result shape artifactとAuthority EvidenceのGate
- read-only effectの停止線
- generic Runtime Capability identityとlegacy Authorityからの一方向projection
- 既存Resource Broker / Host Sandboxを再利用するgeneric Runtime Reference

一方、次は評価中または未実証です。

- generic Invocationをproduction Environment Planの後ろで実行する統合
- Invocation Context Slotの最小契約
- generic Invocation identityをphysical transportから分離したE2E
- actual browser / Wasm realization
- mutating Capabilityの実行契約
- Contract / Shape / Conformance Evidenceを発行するpublisher trust
- 一件の既存Runtime Capabilityをgeneric-authoritativeへ切り替えたproduction Evidence

このため、G1以後はまだ採用できません。

## 16. 敵対的評価

### dual authority

旧Environment Authorityとgeneric Runtime Authorityが同時に有効な場合、片方だけPermissionが更新されても実行経路によって結果が変わります。

**判定: Blocker。** 一件のCapabilityについて、実行Authorityは一つにします。

### automatic legacy fallback

generic Providerが見つからないとき旧`CapabilityKind`へ戻ると、generic側のConformanceやTarget不適合を迂回できます。

**判定: Blocker。** rollbackは明示したAccepted authority stateの切替として扱います。

### free-string replacement

固定enumを任意文字列へ置き換えるだけでは、Contract meaningとConformanceがありません。

**判定: Blocker。** generic化はtyped Contract / Provider / Evidenceの導入とセットです。

### transport-owned identity

processやHTTP requestがContract / Provider identityを自己申告し、その値でHostが実装を選ぶと、transportがAuthorityを持ちます。

**判定: Blocker。** dispatch identityは上流Authorityから導出します。

### context in semantic request

authorization tokenやPrincipalをsemantic request shapeへ埋め込むと、current execution stateでContract identityが変わるか、consumerがAuthority入力を偽装できます。

**判定: Blocker。** execution contextはHost bindingとして分離します。

### enum removal before observability migration

実行はgeneric化しても、diagnosticやprovenanceがlegacy kindだけを出す状態でenumを削除すると、障害調査からContract / Providerへ戻れません。

**判定: High。** observabilityのre-entry経路も移行Gateへ含めます。

### universal lock

意味、実装、build、current sessionを一つのlockへまとめると、一部の更新だけで無関係なidentityまでstaleになります。

**判定: High。** Authorityごとのlockを維持し、必要なfingerprintだけを参照します。

## 17. ADRへ進む条件

次が揃うまでは、この文書を採用済みADRへ昇格しません。

- `contract-query-executor`でL2のactual E2Eが成立する
- generic Invocationが既存Runtime Authorityを迂回できないことをactual testで確認する
- Invocation Contextの最小境界が二つ以上のconsumerで成立する
- transport層が独自にProvider / Authorityを作れないことを確認する
- G1で必要なgeneric Runtime Authorityの正本候補を一つに絞る
- read-only既存Capability一件についてlegacy / genericのAuthority一致を機械確認できる
- rollback / provenance / stale detectionを含む切替手順を説明できる
- exact-headのfocused testとRepository healthを取得する

ここまで揃った時点で、初めて「新しいCapabilityKind追加を原則停止するか」と「最初の既存kindをG1へ移すか」をADRとして判断します。

## 18. 非主張範囲

この設計候補は、次を主張しません。

- generic Capability Contractをproduction schemaとして採用済みである
- 既存`CapabilityKind`を削除してよい
- Source AdapterがApplication Runtime Capabilityである
- process / HTTP / JSONを標準transportにする
- Wasmを必須Plugin ABIにする
- mutating Capabilityがgeneric Invocationで安全に実行できる
- Shape Authority Evidenceだけでpublisher trustが成立する
- Referenceのfingerprint規則をproduction標準として固定する
- generic Runtime identityだけでPermissionやResource Authorityを得られる

現段階の価値は、移行時に二重Authorityを作らず、Referenceごとに確認した境界を一つの順序へまとめた点にあります。

## 19. Contract Query結果の意味保存：二つのReference反例とTarget投影（2026-11-02）

> **位置づけ：未採用の移行評価追補。** #951の既存Reference feature freezeとL0/L1/L2/G1/G2を維持する。本節は#179のQuery契約を再定義せず、正式なService Contractの発行、Owner受理、新Gate・公開Status enum・共通ABI・独自TEI Hostの追加を指示しない。対象はread-onlyの一件であり、mutating ActionのEffect/Receipt/Unknownをこの結論から導かない。

### 19.1 原Contractと二つの実行Referenceを混同しない

照合基準：TEI main `9c9ef4d160640908e06ad13afb6c2b45c664c748`（2026-11-02）。当該ソースはこの固定版で静的に再読した。Git blobはCustomer試験Adapter `5a0d60be265047d391061d66dbe833764e24e538`、Generic result shape `bf7a5391fa20ca27d14558ff185e11ee801f2789`。

| 対象 | 定義・経路 | 成立する範囲／成立しない範囲 |
| --- | --- | --- |
| Database Contract Query | `reference/workspace/databases/master-data/contracts/customer.tei`: `master_data.customer v2.4/find_by_id`、input `customer.customer_id`、result `customer.customer zero-or-one` | #179の正常0件・正常1件・実行失敗の区別を比較基準にする。実Oracle readの証拠ではない（#214）。 |
| Customer Service Carrier試験 | `internal/driver/reference_two_application_service_carrier_test.go:82-119` の `sharedCustomerServiceExecutor` → `InvokeSelectedServiceCarrier` | `test-only.customer-query-service` の合成Route/Acceptance/Verifier/Processを使用。正式な上記Database Contractと同一Contract identityまたは意味Acceptedではない。 |
| Generic Contract Query Invocation Reference | `tools/reference-plugin-capability-contract/contract_query_invocation.go` と `shapes/contract-query-result-v0.json` | `driver.ContractQueryExecutor` の結果をJSONに符号化し構造的Shapeを照合するReference。Customer試験Adapterとは別経路であり、相互のPASSを一件のE2E証拠に合算しない。 |

`customer.customer_id`はSemantic alias `std.identifier`、試験用Adapterの物理入力はGo `int64`。同じ見かけの値でも型・値域・IDの意味適合は未受理。`customer.customer`には`customerId`と`name`が宣言されるが、試験AdapterはIDしか投影しない。全属性の必須性はOwner確認対象であり、Shapeだけから必須と決めない。

### 19.2 互いに異なる二つの損失

| ケース | 現行固定ソースからの予測 | 失われる区別・不足する検証 |
| --- | --- | --- |
| A1 Customer Service応答 `{"customer_id":42}` | ID42の1行を生成 | 単一正常値の合成接続のみ。実DB検索や返却属性の完全性は未検証。 |
| A2 応答 `{}`、`null`、`{"customer_id":null}` | Goの非ポインタ`int64`のゼロ値を経て、ID0の**正常1行**を無条件生成 | missing/null/明示0の区別を喪失。正常な未検出のService表現は未定義。 |
| A3 応答 `{"customer_id":42,"customer_id":null}` | 標準`json.Unmarshal`では重複メンバーの拒否がなく、ID42の1行となり得る | 曖昧なWireを正常レコードとして処理。Generic Referenceの重複JSON拒否の既存資産と対照。 |
| A4 Serviceの実行エラー／Runtime拒否 | Adapterは`InvokeSelectedServiceCarrier`のerrorを返す | **全失敗を1行へ潰しているわけではない**。実行失敗と正常0件の区別は残るが、本番Provider/Host保証ではない。 |
| B1 Generic Executorが`ContractQueryResult{Rows:nil}`を正常返却 | `json.Marshal`で`"Rows":null`、Generic結果Shapeはarray要求で拒否 | 既存Driverでは正常0件だが、Wire表現がShape不適合。 |
| B2 Generic Executorが空の非nil Rowsを正常返却 | `"Rows":[]`は構造的Shapeへ適合し得る | 同じ正常0件に別のWire結果。**当該QueryのWire境界だけ**の正規化候補であり、全TEIのnil一般則ではない。 |
| B3 Generic Executorが2行または空/NULLのAttributesを返す | `Rows`は単なるarray、`Attributes`の値は`any`で構造的に通過し得る | `zero-or-one`の件数・Customer属性の必須性/実行時意味型をShape単独では保証しない。 |

`internal/driver/contract_query_execution.go`には0件・1件・複数件のcardinality検査と、結果Dataに属さない属性の拒否がある。ただし、必須属性・値域・nullの完全な実行時検証ではない。Generic Referenceは`executor.ExecuteContractQuery`を直接呼ぶため、このDriver側のcardinality検査を経由したと推定してはならない。

Aは**異なる入力状態を同一の正常1行へ潰す**損失、Bは**同じ正常0件の物理表現差で拒否する**不整合であり、修正箇所も評価票も分ける。各表の挙動はこの版の静的ソースとGo/JSON仕様からの予測。新規のTEI結合試験・実Provider実行のPASSではない。

### 19.3 結果のClaimをScope・時点・完全性に相対化する

評価用比較票の軸は、Query/意味版、Selected Logical InstanceとEnvironment/権限Scope、読取条件（Snapshot/時点/整合性）、対象集合の完全性、Outcome（正常0件・正常1件・契約違反・実行失敗・不明）、それぞれの観測根拠、とする。**新しい公開Claim Schemaではない**。

- 正常0件とは、**指定されたQuery・選択Instance・可視範囲・読取条件において完了した照会の0件**。世界全体に顧客が存在しないという証明ではない。ページ未取得、途中失敗、部分取得、RLS等の可視範囲差、Snapshot不明を完全な正常0件へ昇格させない。
- `NotFound`（正常0件）、`Invalid result`（欠損/件数/意味型不適合）、`Execution failure`、`Authority denied`、`Unresolved/Unknown`は比較票上で区別する。これらの語をそのままCore固定enumへ追加しない。
- `Rows:nil`と`Rows:[]`は**このDriverの正常0件意味においてのみ**同等扱いできる。`#1121`の`SourceInputs=nil`（closure未報告）と`[]`（確認済み0件）は逆に意味が異なる。Contractを跨ぐ一律nil→空正規化は禁止。
- Read結果のConsumer解釈（表示/選択/Business Action）は別契約。顧客未検出から受注拒否を自動生成しない。Viewの部分取得・逆写像は#1127/#2497/#2500、Mutating Effect/receiptは#1174がOwner。

### 19.4 二経路の独立した最小適合票（新Gateではない）

| ID | Referenceと入力 | 確認する義務 | 証拠状態 |
| --- | --- | --- | --- |
| CQ-R1 | Customer試験Adapter、正常ID42 | 同じIDの1行を維持 | 既存合成正常系の静的所見。 |
| CQ-R2 | Customer試験Adapter、`{}` / `null` / 属性null / 重複キー | 欠損/曖昧入力を正常顧客へ偽装しない | 現行コードでは不成立と予測。差分試験未実行。 |
| CQ-R3 | Customer試験Adapter、Service実行失敗/Acceptance撤回 | 正常0件とerror/拒否を混同しない。Verifier/Process到達条件も分離 | 既存合成負例あり。今回再実行なし。 |
| CQ-R4 | Generic Invocation、`Rows:nil` と`Rows:[]` | 同じ正常0件意味をWireで保持するか、差を明示 | nil→null/array拒否、非nil空→[]を静的予測。新試験なし。 |
| CQ-R5 | Generic Invocation、2行/空Attributes/NULL値 | Shape構造合格とQuery多重度・Data/型適合を別に検査 | Generic Shape単独では不十分。 |
| CQ-R6 | 入力のmissing/null/型違い/値域違い | `#2056`の入力契約と出力欠損を混同しない。ID0へ黙ってcoerceしない | 入力側実装・検証は独立。業務値域はOwner未受理。 |
| CQ-R7 | 二Application/別Instance/Host current route | Contractが同じでもInstance・Scope・Generation・Provider/Artifact/現在Authorityが異なる場合を拒否/再検査 | #951/#1223既存試験を参照。実DB・本番Host未検証。 |
| CQ-R8 | 外部Target（Java/.NET/Wasm等） | 元の意味をTarget固有Collection/型/失敗へ対応させ、未対応は明示 | #2077既存主対象は`line-amount`。Customer Queryの別Target実走は未取得。 |

最初の実装作業はCQ-R2とCQ-R4を**別々にcharacterize**し、実測でSource予測を確定すること。既存#179の件数検査や#951のShape/Authority Registryを再実装しない。修正候補は(1)Service応答を正しく検査してからSemantic Resultへ投影、(2)Queryで意味の定まった正常空結果をWire表現に適合させる、の局所二件に限定する。Serviceで正常0件をどう符号化するかは正式Contract/Owner判断に戻す。

### 19.5 G1/G2判断・外部Target・Owner境界

- **既存の確定基準**：#179の`zero-or-one`、0件とExecution Errorの非同一性。これを新しいService契約の未受理なJSON表現で上書きしない。
- **Ownerへ残す問い**：`customer.customer_id`と`int64`の値域・ID0の許容、顧客結果の必須属性/null、Serviceでの0件表現、Instance/Scope/Consistency/Completenessに必要な保証、Query→Serviceの意味的Conformanceと公開/受理。Publisher/Provider/現在Authorityは各Ownerが別確認する。
- **L1/L2**：旧Authorityが正本。限定的な合成Service疎通やGeneric Shape成功は正式意味適合の証拠ではない。二つのReferenceに別々の成功があっても同一Attempt/Contract/Provider/ResultのE2Eと合算しない。
- **G1/G2**：#951のContract・Provider Conformance・Publisher/Issuer Trust・Shape Authority・Host current Scope/Permission/Resource/Artifact/Generation・Result/Failure意味が揃うまでは移さない。旧Kindの0（宣言/Validation/Lock）・A（Executor選択）・B（実行Carrier/Runtime）を独立に撤去判定。#214 actual Oracleの0/1/error、実効read-only、外部Source/DB事実は別証拠。
- **Target**：TEI独自共通Hostを新設しない。TEIは意味/契約/生成/適合と来歴を担当し、外部Java/.NET/Wasm/Nativeの各Hostが実効権限・寿命・副作用境界を担う。#2390の外部Consumerと#2513のJava型付きLibrary実Loadは、#2077の生成器fixture選抜とは別の合格条件。
- **SHO/説明**：#1803とSHO系の設計成果物では「意味仕様」「対象版/Source/結果Claim」「技術的適合」「Owner受理」「実DB/外部Hostの観測」「未確認」を明示的に区別する。図・JSON・digest一致を業務意味/実行資格へ昇格させない。

責務参照：#951（Service Contract/Provider/Authority）、#179（Database Query意味）、#2056（入力missing/NULL）、#1121（Lowering/Target）、#2077（別言語生成比較）、#2390/#2513（外部Consumer）、#214（Oracle actual read）、#1803（設計成果物）、#1127/#2497/#2500（取得範囲/表示・編集）、#1174（書込み確定）。

**未実施・未採用**：今回の追補はmainの静的照合と既存Issue/PRの議論に基づく設計更新。新たなGo/Java/.NET/DB/Host試験、正式Service公表、実Provider受理、Owner業務受理、G1移行、旧Kind削除、製品schema/API変更は行っていない。

### 19.6 Function Data Context・Queryの集合意味と実取得/開示の接合（2026-11-02追補）

**追加圧力の由来：** #2506（Result Sliceのgrain/partial/complete）、#2387（Query Lineage/Bag/Hidden dependencies）、#2622（Declared Read→Actual Acquisition→Disclosed Output）、#2627（複数ProviderのRead Cohort合成32件）、#1708（Snapshot整合とread-set）、#2516（Consumer開示）、#2598/#2500（型付きActionへの帰属）。これらは#951のService Contract/Provider結果意味に対する**Consumer条件**であり、#951に第二のContext Model、SQL Analyzer、Disclosure Policy Engineを作る要求ではない。異質Consumerの局所試験PASSを、今回のCustomer Service AdapterのPASSと合算しない。

**結果件数を比較する対象の粒度を先に固定する。** Query Contract `result customer.customer zero-or-one`は、選択されたQueryが返す**契約上のSemantic Customer結果**への件数制約であり、無条件に「物理SQLのraw join行数」「Function Context内の別Collection件数」「画面へ公開した表示行数」へ読み替えない。ただし、複数物理行を一つのCustomerへ**暗黙にdeduplicate**することも不可。明示されたmapping、Identity、候補キー、関係・bag多重度、集計/絞込みの有無を根拠に、結果の粒度と件数を判定する。#2506のH/D1(2件)/D2(3件)→Join投影6行はEntity件数が6という証拠でない。Group/Aggregateの1行も元Entityの1件を意味しない。意味上のCustomerを一意に投影できない場合は、1件と捏造せずunresolved/contract違反の判断材料として保持する。Semantic Key・row identityの業務採否は#1126/#179 Ownerに戻す。

**一つのQuery結果について、下記の観測辺を別々に残す（比較票であり新Schemaではない）。**

| 辺 | 比較する実体 | 守るべき非推論 |
| --- | --- | --- |
| Q0 Declared Read | Query/Source revision、必要Relation/Field、Predicate/ORDER/LIMIT、選択DataSource alias・意味Scope | SELECTの表示列だけを必要Read Footprintとせず、WHERE/JOIN/ORDER/演算依存を含む。 |
| Q1 Actual Acquisition | 実Provider/Instance/環境、実Query/request、取得列・行のScope、Snapshot/Query評価条件、物理Read witness | 表示結果のbyte一致は実Read最小性や認可済み取得の証拠ではない。実Readが未観測なら`unverified`。 |
| Q2 Semantic Result | Queryの結果grain/identity、0/1/多行、取得complete/partial/failed、missing/null/unknown、値・operator評価規則 | 不完全な0行やraw Joinの行数を正常なSemantic `NotFound`にしない。Shape適合をBusiness Result適合へ昇格しない。 |
| Q3 Disclosed Projection | Consumerに渡したValue/件数/診断/存在情報、許可Scope、必要な非開示 | フィールドを隠しただけでQ1の過剰Readが無かったと主張しない。開示上の0行は内部Q2の0行と同一とは限らない。 |
| Q4 Business/Action | Function Context/選択対象、Action Purpose、Target Binding/現在Grant/Effect/Receipt、完了判定 | Readの`NotFound`やUI空表示からAction拒否・実行資格・Owner完了を自動決定しない。 |

**既存CQ-R評価票に追加する比較条件（新Gateを設けない）：**

- **CQ-R5の粒度負例**：CustomerとContactのJoinで同じCustomerが2物理行になり、別の二子集合Joinでは出力が6行になる。Contract結果粒度への明示的な還元が未定なら0/1違反の判断自体を未確定として報告する。「重複だから一意化して1行」も「物理2行だから顧客2件」も独断しない。
- **CQ-R7の完全部分/Provider負例**：先頭ページが空でも後続未取得、あるいは別Providerに`Snapshot:"s7"`が存在するだけでは、同一Cohortの完全な正常0件を保証しない。#2627の`fixture_consistent / contradicted / not_established`は合成Fixture内の観測分類であり、実DBのSnapshot保証ではない。逆に単純な完了済み単一Query正常0件へ、全Provider横断の数学的不存在証明を一律要求しない。
- **CQ-R8の演算/Target負例**：同じ`find_by_id`名でも、ID比較のCollation/正規化、NULL、型変換が異なれば0/1が変わり得る。ORDER/LIMIT/集計があるQueryではTie/NULL Order、Overflow/空集合Aggregateの差も別の意味条件。#2387 QX-28の独立SQLite/Python14件PASSは参考であり、顧客Queryの正式2 Provider・Java/.NET/Wasm適合は未実測。Provider PushdownとLocal fallbackを同義とみなすには、そのQueryが実際に使う演算・Scope・完全性条件の証拠が必要。
- **Q3の開示負例**：内部には1件あるが、Consumerの許可範囲では空表示とする場合、内部結果・開示結果・拒否/非開示理由の混同を避ける。ただし顧客の存在自体が秘密なら、外部への診断文面や件数の露出も#2516/#1178の開示Authorityに従わせる。内部診断を無条件で外部へ渡さない。

**実施優先度：** (1)既存CQ-R2/R4の独立characterization、(2)CQ-R5でraw/semantic/displayのgrain差を固定Fixtureの期待値で比較、(3)CQ-R7/Q0〜Q3の実Provider観測、(4)必要なConsumer/外部Targetへ展開。新しい全件性証明、全SQL最適化、全Target動作を最初のread-only反証の阻害条件にしない。Q0/Q1/Q2/Q3/Q4の証拠が独立に揃う前に「Query→Business E2E」を主張しない。

**Owner境界：** Contractの意味・grain/Identity・nullable/演算条件（#179/#1126/#2495）、Result Sliceと部分取得（#2506）、実SQL/Lineage/物理Read（#2387/#2622）、Provider/Cohort（#2627/#1708）、開示（#2516）、Operation/Effect（#2500/#2598/#1174）、外部Target（#2077/#2390/#2513）は既存の責務を保持。#951は意味的Conformanceを比較して必要な証拠の欠落を指摘する。実Provider・実Host/DB・正式Provider/Meaning受理、Consumer開示テスト、各Target E2Eはいずれも本追補では**NOT RUN**。

### 19.7 SQL PushdownとTEI後処理の交換条件・二種類の完全性（2026-11-02追補）

**他レーンからの成熟条件：** Native SQLを直接使う実現と、SQL結果をTEI DSL/Data Operatorで後加工する実現はともに許容する。TEI標準の書き方を新たに必須化するのは、記述の明瞭性・検証性などに**観測可能な優位**がある場合に限る。DB内で処理するか外部Hostで処理するかは性能だけでなく、契約意味・取得範囲・権限・データ開示の境界を伴う。#2387 QX-24〜28、#2622 DO-07、#2506/#2627、#2593の新たな比較条件を、#951の**Provider意味Conformanceに必要な場合のみ**受け渡す。#951自身がSQL OptimizerやData Operatorの第二実装を所有しない。

#### 19.7.1 非可換な演算の最小反例：FilterとLimit

`customers(id,active)` に `('A',0)`, `('B',1)` がある独立した**公開合成Fixture**で、Queryが「activeな顧客をID昇順に並べ先頭1件」と要求する場合：

```sql
-- 期待意味：先に絞込み、その後に順序・件数制限
SELECT id FROM customers WHERE active=1 ORDER BY id LIMIT 1;
-- => B

-- 違反候補：DBから先頭1件だけ取得し、その後TEI側でactiveを選別
SELECT id,active FROM customers ORDER BY id LIMIT 1;
-- => A,0 ；この1件をactive=1で後処理すると0件
```

上記をLinuxの**独立SQLite 3.46.1**で実行し、前者`[('B',)]`、後者の後段Filter結果`[]`を観測。これは非可換な演算順序に起因する**偽の正常0件**であり、Go/TEI製品Query、実Provider、実DB、`master_data.customer/find_by_id`の個別適合を実測したわけではない。現行の`find_by_id`にORDER/LIMITが定義されているとも主張しない。別の汎用Query Consumerにおける意味保持の圧力例である。

**判断：** PushdownとLocal Evaluateの選択は最適化の自由度であって、Source/Operator/契約が規定する`Filter → Sort → Limit`等の順序・bag/identity・NULL/collation・精度・失敗を変更する自由ではない。DB側のLimitを外して全件取得すれば値が合う場合でも、全件物理Readが許可・予算・機微Field最小化・Snapshot完全性を満たすことは別の資格。自動的なFull Read fallbackは採用しない。Pushdownできず局所後処理でも必要な候補集合を取得できないときは、正常な0件ではなく、必要な範囲の`unsupported/unresolved`として根拠を返す候補とする（公開Statusの新設ではない）。

#### 19.7.2 Rowset完全性とField観測完全性は異なる

#2627の既存合成Referenceでは、**Rowset complete**と**比較対象Fieldがobserved**であることを別々に照合する。両Read Sliceの`Complete=true`でも属性`absent`を含めば値の同等性は`not_established`、観測された`NULL`は未観測の`absent`と区別される。PR #2627は合成32件のGo実行/race/vet PASSとexact-head CIを記録するが、実DBのread/NULL保証は未証明。

Customer Queryへ転用する際には次を混同しない。

| 観測 | 言えること | 言えないこと |
| --- | --- | --- |
| 0行＋対象ScopeのRowset完了が**実際に立証された** | そのScope/時点/Query/Predicate下で正常な未検出の候補 | 他Scope/別Provider/別時点の不存在 |
| 1行＋Rowset完了＋`customer_id=42`観測 | 当該行IDの観測 | `name`の値が既知、全属性・型の意味適合、Owner受理 |
| 1行＋`name`フィールド未観測/absent | 行の存在とname未観測を保持 | `name=null`または空文字と同じにすること |
| 1行＋`name`を明示`NULL`観測 | NULLを観測した事実 | 当該DataにNULLが許容されるというOwner判断 |
| 0行＋部分取得/実Provider評価範囲不明 | その断片で観測された0行 | 正常な0件を確定すること |

`Rowset Complete`という自己申告ビットだけで実データの全件取得を証明したことにはしない。必要なWitness（Source/Query/Provider/Instance/selection/snapshot/範囲とその実行観測）を別に確認する。反対に、単一の完了済みLookupに常に全世界の不存在や数学的な完全証明を要求しない。顧客`name`の必須性、`NULL`の意味、情報開示可否はそれぞれ別Ownerが判定する。

#### 19.7.3 CQ評価票への追加接続・作業順

| 比較条件 | 独立期待と反例 | 実装Owner／証拠 |
| --- | --- | --- |
| **CQ-P1 演算順序**（別の汎用Query Consumer） | Filter→LimitとLimit→Local Filterが異なる。ProviderでPushdown不可でも誤った正常0件を返さない | #2387/#2622が演算/取得のOwner。**独立SQLite 3.46.1の局所実測のみ**。TEI/実Provider/Owner未実施 |
| **CQ-P2 行集合 vs 属性観測** | Complete Rowsetでも一属性absentならそのFieldの値同等性は未確定。absent/null/0/空文字を区別 | #2506/#2627がResult Slice/Read Cohort、#179/#1126がQuery/Data意味。#2627の合成PASSを正式Customer Contractへ流用しない |
| **CQ-P3 Measure/集約**（将来の別Consumer） | Join後の重複・Groupの1行・AVGの再平均やsemi-additiveな時間軸をCustomer Entity件数と混ぜない | #2593が集計軸・単位・時間のOwner。Draft #2594の局所FixtureはService Conformanceを実証しない |
| **CQ-P4 要求と実Readのずれ** | 同じ出力でも、実取得した隠れField/Scope、RLS/Permission/取得コストが異なる。説明と開示は別 | #2622 Q0/Q1、#2516 Q3、#1174等の実Host。未観測を`least-read proven`としない |

**優先を変えない：** 最初は既存CQ-R2（試験Adapterの欠損/null）とCQ-R4（Generic `Rows:nil`）を二経路で独立実測する。その後、CQ-R5/R7/R8で必要となるContext/QueryだけにCQ-P1/P2を適用する。CQ-P3は別のMeasure Consumerの検証待ち。成熟した比較条件を#951の契約適合票へ**参照可能にすること**と、#951内に新実装・Gateを追加することは異なる。Source/Provider/Host/Resultの同一Attemptが未接続なら、SQLite正常系・Reference PASS・Shape PASSを一件のE2Eへ合算しない。

**未実施・未採用：** 新Go/Java/.NET/Wasm/実PG/Oracle/Host/TEI Query試験、二Providerの物理Read比較、Rowset完全性の実観測、標準SQL/DSL境界の決定、正式Customer Query Service Result Shape・Owner意味受理、G1/G2変更はすべて未実施／未決。今回の実測は上記2行の**独立SQLite合成例のみ**。

### 19.8 正常0件の否定依存と時間的な利用資格（2026-11-02追補）

**新たに区別すべき義務：** §19.3/19.6の「指定Scope・時点で0件と観測された」が成立しても、**後の利用時点で0件のまま**とは自動的に言えない。#2582のRC-01〜09、[DC-01〜12の独立合成負例](https://github.com/hat47x/tei-platform/issues/2582#issuecomment-6579762087)、#1124のR10→R11遅延応答・用途相対再検証、#1641のキャッシュ鮮度、#1708の同時点読取を、#951に属するService結果の**再利用・失効に関するConsumer必要条件**として接続する。新たなCache/Reactive Engineや現在状態のAuthorityは#951で作らない。

#### 19.8.1 0件にも対象集合への依存がある

顧客`find_by_id(C42)`の正常0件には、**該当レコードIDが存在しないから依存行は空**という推論を使わない。結果に行IDが無くても、Queryの選択Predicate、対象集合/Relation path、Scope、Query/Meaning版、選択されたProvider/Instance、取得条件・Snapshot/Read Cohort、必要な全件性に依存する。変更観測が必要かはConsumerの要求する現在性に依存し、常にCDCや全世界監視を要求するものではない。

- **時点t0**：`find_by_id(C42)`があるScope/Instanceで完了、0件（正常な履歴的結果）。
- **時点t1**：そのScopeにCustomer C42が追加される。新しい同条件Queryは1件になり得る。旧t0の0件は履歴として保持できるが、**t1現在の未存在証明ではない**。
- **時点t2**：C42が削除され再び0件となる（ABA）。t0とt2の値が同じでも「t0以来対象が変化していない」とは言えない。最終Value一致と変化履歴は別Claim。
- **別経路**：実際に一致するC42の1件を権限内で観測できれば、そのScope/時点の**存在側**は部分取得から裏付けられ得る。しかし、部分取得で一致が観測されないことを**不存在側**へ変えるには、必要な集合の取得完了証拠が不足する。

同じことは`NOT EXISTS(ActiveHold for Customer)`、LEFT JOINの未一致、Filter、Top-N、集合差の結果にも適用される。新行の加入が旧未一致結果を撤回し、古い0件を1件へ反転させる。差分更新を`append-only`と仮定しない。必要なPredicate/Relation path/順序依存をOutputFieldの値Lineageだけから復元したと主張しない（#2387 QX-27/#2582）。

#### 19.8.2 取得・表示・再利用・Actionの時点を分ける

| 消費目的と時点 | 既存0件の利用可否 | 追加で確認するもの |
| --- | --- | --- |
| **t0を説明する監査・固定版SHO** | 当時の証拠として保持候補。後のbranch r2が存在するだけで、明示されたr1/t0の説明が誤りとはしない | 利用者が指定した説明対象版、当時のSource/Query/Instance/Scope・Reader開示権限、歴史的Evidenceの実在と一致 |
| **t1の現在値を表示するConsumer** | 旧0件を`current`と自動表示しない。観測済みの関係変更/選択変更なら表示資格を失効させ、未観測なら「変更なしを証明済み」と言わない | Function Context Instance ID、selection epoch、Query/Source/Rule revision、Result Slice/Provider read witness、Change観測の適用範囲 |
| **t1以降に到着するt0の遅延Query/Inspect応答** | 受信できたことと現在表示へ採用することを分離。Consumerの対象Binding/epoch不一致なら現在結果へ混入させない | 起動時に固定されたInput/Instance/epochと、適用先の現在Binding/epoch。別Windowが同じ旧対象を選択中ならそのWindowの資格は個別判断 |
| **t1の結果をCache/差分更新で再利用** | 当該Claimの依存・有効期間・許容Freshness/Consistencyを証拠で示せる範囲だけ再利用候補。不明なら再読込/未確定 | Query Predicate/Relation path/Hidden ORDER/Scope、Provider change observation/通知欠落、Rule版、選択されたReaderの用途 |
| **Action候補→実Request/Commit** | 画面上の旧0件や歴史的成功を現在の実行資格へ自動昇格しない | #2598のRequest対象/時点、#1174のHost現在Grant/Effect/Receipt、Business Ownerの完了判定 |

**重要な限定**：失効した「現在表示の資格」と、履歴的な結果の真正性は別。UIの古い成功応答を拒否することと、表示済み値を失効させることも別の実装上の操作。実ConsumerがSource変更を観測できる範囲を超えて`current`を保証しない。一方、無関係な別Relation/別Instanceの変更だけで、固定された独立Claimをすべて失効させる必要もない。局所再利用には**依存閉包が十分である根拠**が必要。

#### 19.8.3 最小の追加負例（CQ-R2/R4の優先は維持）

| ID | 同じContract意味の入力変化 | 必要な判別／担当 | 検証 |
| --- | --- | --- | --- |
| CQ-T1 **Negative membership** | C42未存在（完了0件）→C42追加 | 旧0件の履歴保存と新しい現在0件Claimを分ける。空のrow ID集合を「依存なし」にしない／#2582・#2506 | 既存合成DC-04〜06を参照。Customer Provider実実行NOT RUN |
| CQ-T2 **ABA** | C42の加入→削除、最終0件に復帰 | 値の再一致と「変化無し」/旧Snapshotの継承を分離／#2582・#1641 | 設計比較、製品NOT RUN |
| CQ-T3 **Late response** | Window1で選択A→B、Aの0件成功が後着。Window2はAを選択したまま | Window1の現在BへA結果を注入しない。Window2の既存Aを一律失効させない／#1124・#2582 | Consumerブラウザー/Windows NOT RUN |
| CQ-T4 **Unobserved change** | Provider内部では対象更新、Consumerは変更通知を未受信/Read Cohort不明 | 旧0件を`current`と実証済みにしない。実観測不能を「実際に未変更」に変換しない／#1708・#2582 | 実Provider/通知欠落試験NOT RUN |
| CQ-T5 **Selective retention** | 無関係なCustomer C99のみ追加。QueryはC42、十分な依存範囲が判明 | 局所的に利用できる候補と過剰失効を比較。未探索依存を無影響確定にはしない／#2582 | 局所証拠の資格、製品NOT RUN |
| CQ-T6 **Safe disclosure** | 低権限ConsumerにC42の実在や変更を示してはならない | 内部の未確定・不許可・理由を区別しても外部の件数/診断/Latency等を無条件に変えない／#2516・#1124 | 実Policy/二世界テストNOT RUN |

**再評価の順序と既存資産**：最初のCQ-R2/R4で**一回のRequest/Result変換**の意味喪失を固定。その後、CQ-T1/T3だけを一つのFunction Context/Consumerに適用し、負例の再現を独立Oracle/既存#1131/#1180 Scenarioへ接続。T2/T4/T5/T6は局所キャッシュ・Provider観測・Disclosureを検証する別Ownerの圧力条件。#2582で報告済みの独立Python3 DC-01〜12 **12/12 PASS**は著者の合成Worldであり、本Customer Query/実Source通知/Provider/Hostの実観測には転用しない。

**表現上の制約：** このT1〜T6や比較表は新Core型・汎用Token/Negative Cache・Subscription/Watch Broker、Release-wide`stale` flag、Provider共通Snapshot API、第二Authority/Resolver/Gateの追加案ではない。履歴・現在・表示・Actionは用途別のClaimとして扱い、既存#951/#1124/#1641/#2582/#2516/#1174の責務を保持する。正式Ownerの意味受理、実Customer Query/実Provider実行、Host現在権限、実UI/SHO Reader、長期Cache・通知欠落検証は**未実施・未受理**。

## 20. SQLを第一級Data Transformationとした場合のService Query意味適合（2026-11-03）

> **位置付け：#951のProducer/Provider Conformanceに対するConsumer条件を整理する追補。** 2026-11-03時点の追加ユーザー方針（#2387/#2619/#2622のコメントに記録済み）と、#2640でmainへ統合された§19の反証群を接続する。製品Schema/ABI、第二SQL Engine/Resolver、共通TEI Host、新しいRelease Gateは決定しない。#951 Reference feature freeze、L0→L2→G1/G2のAuthority境界、#179の既存Query意味を維持する。

### 20.1 Data Model正本、SQLの主用途、二つの解釈

**Data Model**は原則としてTable/Field/Key/Relationを直接宣言し、その受理済み意味を正本とする（#1126）。SQLを解析して基本Data Modelや業務Relationを自動生成・受理することを通常経路にしない。DDL/既存DBの取込は補助的な候補として区別する。

**SQLの主用途**はLINQに類似した**データ変換**である。受理済みData Modelや型付きDatasetから、filter/project/join/group/aggregate/sort/expand/nest等の変換によってDerived Dataset/仮想テーブルやFunction結果を得る。SQLは「DBアクセスだけの特殊記述」ではない。一方、SQL構文を万能な全構造共通Executorへ直接結び付けるわけでもない。#988のnamed/method/pipe/SQL構文は、型付きOperatorの意味・評価順・引数Binderが一致する範囲で別の書き方となり得る。

同じSQL Sourceに対して、**TEI Query解釈**（可能な範囲を型付きOperator/結果grain/Lineageへ解釈し、Pushdown/Local実現候補を評価）と、**Native SQL解釈**（原文を保存し、対象Provider/Dialect/Schema/Sessionによって意味付け・実行）を区別する。Native資産をTEI解釈できないだけで失効させない。ただし双方の同義性・Effect/Failure・取得Scope/時点は**自動承認しない**。形式が同じ、SQL byteが同じ、Result Shapeが同じ、という事実だけではProviderの意味適合を示さない。

| 軸 | 正本／入力 | #951が受け取る確認義務（実装Ownerは別） |
| --- | --- | --- |
| **M：Data Model** | 直接宣言のTable/Field/Key/RelationとAccepted revision（#1126） | QueryのJOIN演算辺やDocument Pathを業務Relationへ自動昇格しない |
| **Q：Data Transformation** | SQL/typed Operator、Source occurrence、Operator/Expression version、Query入力と結果grain（#2387/#988） | Queryの意味版と必要な変換順・Bag/NULL/Identity・失敗の同等性を識別 |
| **B：解釈とBinding** | TEI Query/Nativeの解釈選択、解決時Schema/Dialect/Session、Target/Provider、Mapping/Context/Scope（#2387/#2622/#2506） | 同じSource SQLでもBinding違いで意味が変わる場合に、旧Conformanceを流用しない |
| **E：実行・結果Claim** | 実Provider SQL/request、Read Cohort、Result Slice/取得完全性、WireとConsumer開示（#2622/#2516） | 物理Execution成功、意味的Result、現在許可、Consumer開示を別証拠として記録 |

これらの列は既存のScenario/Conformance比較票への受渡しであり、新しいCanonicalや固定Context Schemaではない。データモデルの正本とSQL Sourceの原文正本はそれぞれ保持する。SQL/Markdownの人間向け原文を、解析・保存によって無断で別表記へ正規化しないことは#2387/Modeler/SHOの担当。

### 20.2 同一SQL byteでもSchema変化により結果が変化する反例

**独立SQLiteの最小対照（2026-11-03に再現）**：

```sql
SELECT o.order_id, f.flag_id
FROM orders AS o NATURAL JOIN flags AS f
ORDER BY o.order_id;
```

- **Schema A**：両表の共有列は`customer_id`のみ。`orders`の(10,C1),(11,C2)、`flags`の(100,C1),(101,C2)から **(10,100),(11,101)**。
- **Schema B**：双方に`status`列を追加。`orders`は(10,C1,open),(11,C2,closed)、`flags`は(100,C1,open),(101,C2,open)。SQL byte不変でも`NATURAL JOIN`の条件は`customer_id AND status`となり **(10,100)** のみ。
- **対照**：`JOIN ... USING(customer_id)`ならSchema Bでも2行。

Python標準`sqlite3`のin-memoryで上記2 SchemaのSQLを独立実行してこの結果を観測した。#2387にはより広い5/5合成テストの記録があるが、**TEIの実SQLGlot Worker/Go DerivedIR/本番DB/Provider Conformance PASSではない**。今回の観測は通常の顧客`find_by_id`が`NATURAL JOIN`を使用することを意味しない。

**設計上の差分**：Query Source digestだけを再利用キーとして旧Query Explain、Operator Lowering、Provider Conformanceを`current`へ自動継承しない。**当該SQLで実際に解決されたSchema列集合・暗黙Join比較列・Dialect/名前解決・Provider Binding**が変化したら、必要な意味対応を再確認する。さらに`SELECT *`のShape変化、Viewの列改版、Native SQLのsession設定も別の依存になり得る。業務ERのRelation、PK、顧客IdentityはSQLの実効Join辺から推定採用しない。

### 20.3 階層・多構造変換と正常0件を混同しない

#2622の階層/Document案HQ-01〜08および多構造案MS-01〜08は、同じ型付き操作を**適用可能な基本構造へ広げる**ことを求める。Table/RelationだけでなくRecord/Tuple、Sequence/Array、Bag/Set、Map、Tree/Document/JSONB、Graph、Stream/Time Series、Tensor/Spatial等を候補とする。ただし、**同じOperator名が同じ構造法則を持つことは証明を要する**。全構造を一度TableへFlattenすることも、全構造にsort/join/window等を要求することもしない。

顧客照会の`Result Rows`と同様、階層Path/Expandの返却でも次の入力差を失わない。

| Document中の`items` | Value/Presenceとしての区別 | 未受理の暗黙変換 |
| --- | --- | --- |
| `{"items":[]}` | 観測済みの空配列 | 値のないField／未取得として扱う |
| `{}` | Field自体が欠落 | 空配列として「展開正常0件」に変換 |
| `{"items":null}` | 明示JSON null | 属性未存在・空配列と同一視 |
| `{"items":[null]}` | 要素が1個で、その値がnull | 要素数0と同一視 |
| `{"items":{}}` | arrayでなくobject | 暗黙に1要素に包む |
| **SQL NULL／未取得／非開示** | データソースNULL、Read未実施、Policy上非開示は別 | 同じ`[]`/正常NotFoundに合成 |

上表は**未採用のOperatorポリシーを勝手に決めるものではない**。許容するPath緩和・欠損処理は各意味契約の明示方針で判断する。Pathの子要素occurrence、Document Identity、配列位置、業務Entity ID、QueryのResult grainを区別する。Nested→Flat→Nestedの可逆性や配列Indexの安定した更新対象Identityを推測しない。JSONB/Document/Local/Java/.NET間の同義性は各Provider/Targetの独立Fixtureで確認する。

### 20.4 保存後の0件は書込み失敗・未実行の証拠ではない

§19.3/19.8の「0件をScope/時点に相対化する」原則に加え、**直前のActionのEffectが当該Readへ反映されたか**という因果的な保証を別に確認する（#2598 IB-N1〜06、#1174）。

- t1でPrimaryへ`allocation A1`を作成し、Provider Receiptが`committed`。
- t2で追随前のReplicaへ同じ対象のQueryを実行し、完了した`[]`を返す。
- これは「**そのReplica上で0件を観測**」と「**PrimaryでCommit済み**」が両立する例。Rowset`Complete=true`だけでは当該Operationを含むReadとは限らない。
- Commit ACKが失われてActionが`unknown`の場合、空Readを根拠に無条件で再作成してはならない。実ProviderのIdempotency/Receipt/Operation Identity/RYW fenceのOwner判断へ戻す。
- RLSによる不可視も空Readの理由となり得る。実行Provider/Principal/Scope/因果位置と、Consumer向けに開示可能な理由は別。

**#951はread-onlyの結果契約の範囲で「空の観測を勝手に書込み未実施へ拡張しない」ことだけを保持**する。Read-your-writes、Operation Identity、実Receipt、無確認再送の防止、現在Host Grantを#951に重複実装しない。#2598/#1174/#2611が保存→再読取の因果的適合を所有する。

### 20.5 既存の実行計画へ接続する最小比較票

| ID（本節の参照名） | 最小反証・受入条件 | 主担当・停止点 |
| --- | --- | --- |
| **CQ-I1 Model→Query** | 直接宣言ModelとSQL変換Sourceを別ID/版で保持。SQL JOINやDocument Pathを業務Relationへ自動受理しない | #1126/#2387。基本ModelやNative SQL Source正本を変更しない |
| **CQ-I2 Schema drift** | 上記Schema A/Bで同一SQLが2行→1行、明示USING対照は2行。旧Result/旧Explainを同一Meaningと誤認しない | #2387 QX-18/33、#2582。**実Worker実体と起動・依存が未確定なら`NOT RUN: producer-unavailable`** |
| **CQ-I3 Native vs TEI** | 同じSourceを二解釈した際、解析CoverageとProvider実行の対象版を独立記録。同義性未証明なら技術的成功を意味一致へ昇格させない | #988/#2387/#2622。全方言実行器なし |
| **CQ-I4 Hierarchical presence** | empty/missing/null/[null]/object/partialを区別し、展開後のOccurrence/Bag/Identityを保持 | #2622 HQ-01〜04/#2506。実PG JSONB/Document未実施 |
| **CQ-I5 Causal empty read** | Primary Commit→Replica空Readを、未保存・業務不存在へ変換しない。ACK unknownで盲目的再実行しない | #2598 IB-N1〜06/#1174。実Host/DB/ReceiptはNOT RUN |
| **CQ-I6 Claim/Disclosure** | 上記反例のどれも内部Parser/Service診断からConsumerの現在表示・業務権限・外部開示を自動採用しない | #2516/#1124/#2390。Reader/Policy/Host実行別 |

**順序は維持**：最初の#951内局所作業は§19の**CQ-R2（Service Adapterの欠損/null）とCQ-R4（Generic`Rows:nil`）の独立characterization**。本節のCQ-I1〜I4は#2387/#2622等の既存評価票へ参照接続し、CQ-I5は#2598/#1174へ委譲する。SQL解釈の統合実証は#2387 QX-33の**実Worker→実Go DerivedIR同一Attempt**が成立するまでSource-static/独立SQLiteと区別し、mockでE2E PASSを作らない。必要な新統合は既存#1119/#2586の縦断計画へ渡し、新Gate/第二正本は作らない。

**Evidence・非主張**：main固定`a5cb7c4d58b5cf42d1c9c73810f86398987ae32c`（2026-11-03 JST）に統合済み§19を正とした上で、この第20節のみ追加した。独立SQLiteのSchema A/B対照以外、新TEI module Go/SQLGlot Worker、PostgreSQL/Document Provider、Java/.NET/Wasm Target、外部Host/CLI/GUI/SHO E2E、Ownerの意味受理、公開Schema/ABI、G1/G2移行は**NOT RUN/未受理**。
