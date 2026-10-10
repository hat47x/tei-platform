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
