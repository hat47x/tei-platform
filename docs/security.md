# Security / Trust Boundary

この文書は、TEIのSecurity、Resource Authority、Runtime Trust Boundaryを説明する**current technical boundary**です。

Product Requirementや個別ProviderのSecurity仕様を第二正本として定義する文書ではありません。ここに書かれた境界が実装でどこまで成立しているかは、対応するConformance、Reference、Evidenceで確認します。

この文書を読むうえで最も重要な前提は、次の一点です。

> **「業務上その機能が存在すること」と、「ある実行主体がその機能やResourceを使ってよいこと」は別です。**

TEIのSemantic ModelにCapabilityが定義されていても、それだけでPluginやAI、Hostへ実行権限が与えられるわけではありません。

## 1. 何を防ぎたいのか

TEIでは、業務上の意味からApplicationやRuntimeへ処理をつなぎます。そのため、意味の定義が正しくても、実行境界で次のような問題が起きると安全性を失います。

- Pluginが、宣言されていないnetworkやfilesystemへアクセスする
- SecretがCanonical Definitionや履歴へ平文で残る
- 一度失効したResource authorityが、その後も使えてしまう
- Host交代後も古いHostがResource操作を続ける
- 外部processの状態が不明なのに、同じ副作用を自動再実行する
- 「実行を要求した」ことと「外部で実際に起きた」ことを混同する

これらは単なるSecurity機能の不足ではなく、TEIが守ろうとしている「意味・権限・実行・観測の分離」を壊します。

そのため、TEIでは次のように段階を分けます。

```text
Business / Semantic meaning
        ↓
Capability exists
        ↓
Environment grants permission
        ↓
Host resolves concrete resource
        ↓
invocation-scoped authority
        ↓
external execution
        ↓
Observation / Evidence
```

上段の意味が存在するからといって、下段のauthorityを自動的に得ることはありません。

## 2. Capability is not permission — 機能の存在と権限を分ける

### 防ぎたい失敗

たとえば「メール送信」というCapabilityが業務モデルに存在していても、すべてのPluginが自由にメールを送れてよいわけではありません。

Capabilityの存在をそのまま実行権限とみなすと、Pluginの追加や再利用だけで権限が拡大する可能性があります。

### TEIの境界

process Pluginの実行時には、EnvironmentがCapability単位でPermission Grantを明示します。

Plugin descriptorが「このCapabilityを使いたい」と要求していても、EnvironmentからGrantされなければ利用できません。

```text
Capability exists
      !=
Plugin may invoke it
```

### この境界が意味すること

AI、Editor、generated code、automationも例外ではありません。高度な自律性を持つことは、強い権限を持つ理由にはなりません。

## 3. Resource Broker — 業務上の識別子と実Resourceを分ける

### 防ぎたい失敗

Database password、API token、network endpoint、filesystem path等をCanonical Definitionへ直接書くと、次の問題が起きます。

- 業務上の意味と環境固有情報が混ざる
- SecretがGit historyや生成artifactへ残る
- 開発・検証・本番で同じ定義を再利用しにくくなる
- Host交代やcredential rotationが業務定義の変更になる

### TEIの境界

Canonical側では、具体値ではなくlogical Resource IDを扱います。

```text
logical Resource ID
    ↓ Environment binding
Host Resource Broker
    ↓ invocation-scoped lease
opaque handle / endpoint / mount
```

**Resource Broker**は、論理的なResource参照を、そのHostで実際に使えるResourceへ解決する境界です。

Secretのraw valueをDurable RunやPlugin provenanceへ保存しないことを基本とします。

### 具体例

業務定義に`production-db-password=...`を書くのではなく、`sales-db`のような論理参照を持ちます。実際のcredentialやendpointはEnvironmentとHost側のResource Brokerが解決します。

### 限界

Resource Brokerという抽象を持つだけでSecretが安全になるわけではありません。credentialの保管・注入・rotationを実際に安全に行う仕組みはHost / Provider側にも必要です。

## 4. Sandbox Profile — 「許可したい権限」と実際のOS制約を分ける

### 防ぎたい失敗

TEI側で「networkは禁止」と記述していても、実行processがOS上では自由にnetworkへ接続できるなら、設計上のpolicyだけが存在し、実際の防御はありません。

### TEIの境界

TEIが作るSandbox Profileは、Hostへ渡す**実行制約の要求**です。

たとえば、

- 許可するnetwork範囲
- filesystem access
- Secret injection

等を表します。

しかし、Profileを作っただけでOS level enforcementが成立したとはみなしません。

Hostは実際のenforcerを提供し、deny-by-default policyを適用する必要があります。

```text
TEI Sandbox Profile
   = requested restriction

Host enforcement
   = actual restriction
```

### 限界

現在のReferenceがgeneric OS sandbox一式をproduction-gradeで提供するわけではありません。TEIが先に固定するのは、「どの制約をHostへ要求し、どこから先をHost責務とするか」という境界です。

## 5. Resource Lease — Resource authorityに有効期間を持たせる

### 防ぎたい失敗

一度取得したResource handleが無期限に使えると、処理が終わった後やownershipが変わった後にも古いauthorityが残ります。

### TEIの境界

外部Resourceへのauthorityは、原則としてinvocation-scoped Leaseとして扱います。

**Lease**は「この実行が、このResourceを、この期間・条件で使ってよい」という期限付きauthorityです。

主な要素は次です。

- expiry — いつ失効するか
- renew-before window — いつまでに更新する必要があるか
- explicit renewal — 明示的な更新
- explicit live refresh — 現在有効であることの再確認
- revoke / cleanup — 取消・後始末
- fail-closed supervision — 監督不能時は権限を広げず停止側へ倒す

### 時刻をなぜ外から入れるのか

M0.22 publication hardeningでは、Lease Supervisorのoperational clockをHostから注入可能にしています。

Leaseの期限判定とResource Brokerの時刻基準がずれると、片方では有効、片方では失効という状態が起こり得るためです。テストでも同じ時刻条件を再現しやすくなります。

### 限界

Leaseを導入しても、Leaseを確認しない外部Resourceまで自動的に保護されるわけではありません。authorityを使う境界で実際にLeaseを検査する必要があります。

## 6. Ownership / fencing — Host交代後の古い操作を拒否する

### 防ぎたい失敗

Host AがSupervisorとして外部Resourceを管理していたとします。

Host Aが停止したように見えたためHost Bが引き継いだ後、実はHost Aが遅れて復旧すると、AとBの双方が同じResourceを操作する可能性があります。

単にDB上の`owner = B`を書き換えるだけでは、Aがすでに持っている接続やhandleを止められない場合があります。

### 起こり得る影響

- 同じ処理の二重実行
- 古いHostによるLease更新
- 新しい状態を古い状態で上書きする
- 外部processへの競合したattach / release

### TEIの境界

TEIはmonotonic fencing tokenを使う契約を検証しています。

**fencing token**は、ownershipの世代を単調増加する値で識別し、古い世代からの操作を受け側で拒否するための情報です。

```text
Host A owns generation 41
        ↓ takeover
Host B owns generation 42

operation with 41 → reject as stale
operation with 42 → current ownerとして検査
```

このtokenをResource Brokerおよびruntime reattach境界へ伝播し、stale hostのrenew / refresh / release / reattach操作を拒否する契約を検証しています。

### 限界

fencing tokenを生成するだけでは安全になりません。Resource操作を受ける側がtokenを検査し、古い世代を拒否できる必要があります。

また、TEI RepositoryのReferenceはdistributed consensus / production ownership store一式を提供するものではありません。

## 7. Durable Invocation Handle — 外部processを「わからないから再実行」にしない

### 防ぎたい失敗

TEIが外部processへ副作用のある処理を依頼した直後にHostが停止した場合、復旧後には次の三つが考えられます。

1. processはまだ動いている
2. processは終了している
3. process自体が見つからない

3だからといって、「実行されなかった」とは限りません。外部副作用だけが完了してprocess情報が失われている可能性もあります。

### TEIの境界

外部process / containerの実行identityを**Durable Invocation Handle**として残します。

これは、Hostが変わっても「どの外部実行を追跡していたか」を再び参照するためのdurableな識別情報です。

```text
Durable Handle exists
        ↓ query runtime
running → new fencing tokenでreattach候補
finished → 結果 / Observationを確認
missing → unknownとしてRecovery判断
```

takeover後にruntimeが`running`なら、新しいfencing tokenでreattachできます。

一方、handleが`missing`の場合は自動再実行しません。

### なぜ自動再実行しないのか

外部副作用がすでに発生したか不明だからです。

たとえば送金、メール送信、外部system更新等では、二重実行の方が「処理が止まっている」ことより重大な場合があります。

そのため、状態を`unknown`としてRecovery判断へ送ります。

## 8. CommandとObservationを分ける

Security / Trust Boundaryでは、権限だけでなく「何が実際に起きたか」の確認も重要です。

```text
Command accepted
      !=
External state changed as intended
```

たとえばmigration commandが成功しても、対象instanceが想定と違っていたり、後続の外部処理が失敗している可能性があります。

TEIでは、必要な処理についてObservationを取り、次へ進む根拠となるEvidenceを確認します。

これは「AIやRuntimeが成功と言ったから成功」としないための境界でもあります。

## 9. 一つのtakeover例で全体を見る

ここまでの境界を一つの例へつなぐと、次のようになります。

```text
1. Host AがCapabilityを実行
   ↓
2. EnvironmentのPermission Grantを確認
   ↓
3. Resource Brokerがlogical Resourceを解決
   ↓
4. invocation-scoped Leaseを取得
   ↓
5. external processを起動しDurable Handleを保存
   ↓
6. Host Aが停止
   ↓
7. Host Bが新しいfencing generationでtakeover
   ↓
8. 古いHost AのResource操作はstaleとして拒否
   ↓
9. Host BがDurable Handleからruntime状態を確認
   ↓
10. runningならreattach、missingなら自動replayせずRecoveryへ
```

この流れで重要なのは、「一つの万能Security機構」が守っているのではなく、**Permission、Resource解決、Lease、fencing、Durable identity、Observationが異なる失敗をそれぞれ防いでいる**ことです。

## 10. 現在どこまで確認しているか

M0.22.1でコードとテストから直接追跡できる保証範囲は、[Plugin / Runtime境界のConformance](conformance/plugin-runtime-m0.22.1.md) に整理しています。

この文書の説明だけで「production-grade Securityが完成した」とは扱いません。

Architecture上の境界、Reference実装、isolated test、actual external Evidence、本番運用保証はそれぞれ別です。

## 11. 現時点で保証しないもの

このRepositoryのReference実装は、次をproduction-gradeとして提供するものではありません。

- generic OS sandbox
- Vault / cloud secret manager adapter
- distributed consensus / production ownership store
- signed artifact supply-chain一式
- production HA guarantee

現在の目的は、これらの製品機能をすべてTEI内へ作ることではありません。

**後から適切なHost / Provider / external systemを接続しても、Capability、Permission、Resource authority、ownership、実行identity、Observationの境界が崩れない契約を先に実行可能にすること**を目的としています。

## 12. 次に読むもの

- 全体Architectureとの関係: [Architecture](architecture.md)
- 現在の実装・Evidence状態: [現在の実装状況](status.md)
- Plugin / Runtime境界の検証内容: [Plugin / Runtime境界のConformance](conformance/plugin-runtime-m0.22.1.md)
- 文書の読み方・品質方針: [TEI 文書品質ガイド](documentation-quality.md)
