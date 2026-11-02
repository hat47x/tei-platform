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
