# ISSUE-000041: PostgreSQL fixture削除をDB側ownership markerで保護する

- 状態: In Progress — 実装 / 隔離test済み、実PostgreSQL確認待ち
- GitHub Issue: #427
- 親: `ISSUE-000031` / #308
- Gate B最終収束: `ISSUE-000037` / #387
- Gate B監査: `ISSUE-000029` / #273

## 目的

#308のEvidence toolが、自分で作成した`public.order_lines`だけを削除し、実行途中に同名objectが別主体のtableへ置き換わった場合は削除せずfail closedで停止することを保証します。

Evidence専用空Databaseという運用前提は維持しますが、削除時の所有判定をGo側の`fixtureOwned` booleanだけに依存させません。

## 背景

PR #341以降、#308はfixture cleanup後にuser relation / Event Trigger / fixture object不在を再観測し、Databaseが空状態へ戻ったことを確認します。

一方、従来の明示cleanupは次の形でした。

```text
fixture作成成功
  ↓
Go側 fixtureOwned = true
  ↓
...
  ↓
DROP TABLE public.order_lines
```

通常の専用DB運用では問題ありませんが、fixture作成後に他主体がtableをDROPして同じ名前で別tableをCREATEした場合、Go側booleanはDB上のobject identity変化を知りません。

この状態で名前だけを根拠にDROPすると、toolが作っていないobjectを削除する可能性があります。

## 実装 — PR #428

### 作成時

cryptographically randomなownership markerを生成し、次を一つのtransactionで行います。

```text
BEGIN
  ↓
CREATE TABLE public.order_lines (...)
  ↓
COMMENT ON TABLE public.order_lines IS '<ownership marker>'
  ↓
COMMIT
```

CREATEとmarker付与を同一transactionへ入れるため、markerなしのtool-created tableを正常作成結果として外から利用可能にしません。

markerは固定prefixと16-byte random値のlowercase hexで構成します。

```text
tei:reference-postgres-header-detail-runtime:<32 hex chars>
```

このmarkerはcredentialではありませんが、通常Evidence JSON、stderr、Repositoryへ値そのものを保存しません。

### 削除時

明示cleanupとpanic時のdeferred cleanupは、どちらも同じ`removeOwnedFixtureTable`を使います。

```text
BEGIN
  ↓
LOCK TABLE public.order_lines IN ACCESS EXCLUSIVE MODE
  ↓
obj_description(...)でDB側markerを取得
  ↓
作成時markerとexact match
  ↓
DROP TABLE public.order_lines
  ↓
COMMIT
```

lockをmarker確認より先に取得します。

- lock取得前に別objectへ置き換わっていた場合はmarker mismatchで停止する
- lock取得後は同名objectをDROP / rename / replacementできない状態でmarker照合からDROPまで進む

という順序にします。

次の場合はDROPしません。

- tableがすでに存在しない
- lockを取得できない
- DB側COMMENTを観測できない
- markerがない
- markerが作成時と一致しない
- transactionを開始・確定できない

安全性を優先し、cleanupできなかったtool fixtureが残る可能性は許容します。残存tableがあれば次回の専用空DB guardがfail closedで停止するため、別objectを推測して削除するより安全です。

## Go側booleanとの関係

`fixtureOwned`はdeferred cleanupを呼ぶ必要があるかを制御するprocess-local stateとして残します。

ただし、`fixtureOwned = true`だけではDROPを許可しません。実際の削除許可はDB側markerとの一致で決めます。

したがって役割は次のように分離されます。

```text
fixtureOwned boolean
  = cleanupを試みる必要があるか

DB-side ownership marker
  = その同名objectを削除してよいか
```

## 既存の安全境界との関係

本変更は、`ISSUE-000031`で成立済みの次の境界を置き換えません。

- clean Repository source確認
- Canonical Workspace Validation
- PostgreSQL既定Database拒否
- user relation 0件要求
- Event Trigger 0件要求
- `public.order_lines`事前不在確認
- DSN / credential非保存
- 初期接続errorのcredential-safe診断
- cleanup後のuser relation 0件再確認
- cleanup後のEvent Trigger 0件再確認
- cleanup後のfixture object不在再確認
- post-run Repository source再確認

ownership markerは、その中の**tool-owned cleanupそのもの**を補強する境界です。

## 隔離test

ownership helperはstandard-libraryだけで切り出せるため、隔離Go moduleでcompile / testしました。

```text
ok  tei-ownership-check  0.002s
```

pure testでは次を確認しています。

- 生成markerが制約された形式を満たす
- malformed markerを拒否する
- marker未観測を拒否する
- marker mismatchを拒否する
- COMMENT SQLへ任意文字列を持ち込めない

さらにstandard-libraryのfake SQL driverでtransaction順序を固定しました。

作成成功:

```text
begin → create → comment → commit
```

cleanup成功:

```text
begin → lock → marker → drop → commit
```

marker mismatch:

```text
begin → lock → marker → rollback
```

marker mismatch時に`drop` eventが発生しないことも確認しています。

## まだ確認していないこと

隔離testはPostgreSQL server自身のlock / COMMENT実装を模倣するものではありません。

したがって、次は#308 actual runで確認します。

- `COMMENT ON TABLE`がCREATEと同じtransactionで正常確定する
- `ACCESS EXCLUSIVE` lock取得後にmarkerを取得できる
- marker一致後のDROPが正常確定する
- cleanup後にuser relation / Event Trigger / fixture objectが0 / 不在へ戻る

通常TEI Repository checkout上のpackage testも未実施です。

## 完了条件

- [x] DB側ownership marker生成を実装する（PR #428）
- [x] CREATE + marker付与を同一transactionへする（PR #428）
- [x] lock + marker照合後だけDROPするcleanupへ変更する（PR #428）
- [x] deferred cleanupも同じ保護経路へ統一する（PR #428）
- [x] 旧name-only CREATE / DROP helperを削除する（PR #428）
- [x] marker pure testを追加する（PR #428）
- [x] fake SQL driverでtransaction順序とmismatch時no-DROPを固定する（PR #428）
- [x] 隔離Go moduleでhelper testを実行する
- [ ] tool README / #308へ境界を同期する
- [ ] 実TEI checkout上の通常Go testで再確認する
- [ ] #308 actual PostgreSQL Evidenceでownership付きcleanup / 空状態復元を確認する

## 非目標

- Evidence専用空Database guardを置き換えること
- production Databaseでfixture実行を許可すること
- ownership markerをCanonical modelへ追加すること
- ownership markerをEvidence JSONへ保存すること
- 他主体へ協調ロック運用を要求すること
- GitHub Actionsを再有効化すること

## 関連

- #427
- PR #428
- `ISSUE-000031` / #308
- `ISSUE-000037` / #387
- `ISSUE-000029` / #273
