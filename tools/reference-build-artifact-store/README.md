# Local Build Artifact Store Reference

GitHub #1105 の最小Referenceです。

Build / Runtime Lockが固定したSHA-256 digestからexact artifact bytesを再取得するための、local content-addressed directoryだけを扱います。

production Artifact Registry、cache、Acceptance Authorityではありません。

## Identity

```text
Artifact identity = exact SHA-256 digest
Locator           = store-relative content-addressed locator
```

同じbytesを異なるstoreへ置いてもdigestは同じです。store rootやabsolute Host pathをartifact identityへ含めません。

Receiptの`Locator`は`sha256/<64-lowercase-hex>`のstore-relative形式です。別Host / 別store rootへEvidenceを持ち運んでもabsolute pathを持ち込みません。

## Layout

Reference layout:

```text
<store-root>/
  sha256/
    <64 lowercase hex>
```

入力digestは現行TEIのSHA-256 validationと合わせてhex大文字小文字を受理し、store pathはlowercaseへ正規化します。

mutable tag、friendly name、latest pointerは持ちません。

## Put

`PutBytes`は:

1. input bytesからSHA-256を計算
2. 同digest pathが既にあればstored bytesを再検証
3. 無ければsame-directory tempへ書く
4. temp bytesをdigest再検証
5. hard linkでno-clobber publish
6. concurrent writerが先に同digest pathを作った場合は、そのstored bytesを再検証
7. digest一致ならidempotent success

とします。

既存content-addressed pathが壊れている場合、正しいbytesで暗黙修復しません。corruptionをEvidenceとして表面化させ、fail-closedです。

## Get

`GetBytes(expectedDigest)`はstore filenameを信用しません。

- symlink / non-regular objectをreject
- configured Reference maxBytesを超えるobjectをreject
- open前後でsame fileを確認
- bytesを読みながらSHA-256を再計算
- expected digest不一致をreject
- read後にpathが別fileへ差し替わっていないことを確認

unknown digestはnot-foundとして区別します。

## Bounded Reference profile

このReferenceは`maxBytes`をopen時に明示し、`GetBytes`の無制限allocationを避けます。

この値はartifact content identityではなく、local consumerのresource policyです。別のmaxBytesで同じstoreを開いてもdigest identityは変わりません。

productionで大きなartifactを扱う場合はstreaming materialization等の別APIを検討できます。最初のReferenceから無制限readへはしません。

## Authority

store receiptは次を明示します。

```text
store availability != artifact trust
store availability != build provenance
store availability != Acceptance Authority
store availability != Runtime Authority
locator != content identity
```

bytesをdigestどおり再取得できても、そのartifactのBuild Attestationや現在のTrustは#1104等で別に検証します。

## Artifact Set

#1102のArtifact Setを保存する場合も、このstore primitiveはmember bytesをdigestで扱うだけです。

Set manifestやRole解決は上位契約です。Artifact Set manifestだけ存在してmemberが欠ける状態を「set available」と判定する責務は、この最小Storeへ入れません。

## 非目標

- remote object storage
- OCI / package registry
- replication
- GC
- cache replacement
- rebuild-on-miss
- crash recovery / orphan temp GC
- signature / Attestation verification
- Runtime deployment selection


## Filesystem boundary

Store open時に`sha256/`directoryのfilesystem identityを保持します。

put / get時に:

- algorithm directoryがsymlinkへ変わっていない
- directory自体が別inodeへ差し替わっていない

ことを再確認します。

content-addressed filenameだけ正しくても、Store directoryを後から別場所へ差し替えてartifactを書かせる状態を受理しません。

normal concurrent putではsame-directory tempをno-clobber hard linkでpublishし、各operation終了時にtemp linkをcleanupします。crash途中のorphan temp回収は非目標のままです。

このReferenceは悪意あるfilesystem管理者に対する完全なsandboxを主張しません。Store rootはTEI build processが管理できるfilesystem領域であることを前提とし、その範囲でpath差替え・symlink・content driftをfail-closedにします。