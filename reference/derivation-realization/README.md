# D6 案件固有計算の実現選択：最小Go Reference（Issue #1694）

このPoCは**D1の既存規則とGoによる実計算**を、#988のOperator pinと現在の`internal/plugin`環境Lockへ接続する。新しいPlugin Resolver、Lock、Capability enum、Expression Engine、Acceptance Authorityを定義しない。

`HostSelection.Evaluate`は次を拒否優先で評価する。

1. D1 `derivationref.Definition` の規則ID・宣言済み入力・pure・Operator IDを確認する。既存BindingのTarget IDや規則identityを書き換えない。
2. #988 `languagereference.ResolveOperator` でexact Operator ID/version/fingerprint/type/effectを確認する。今回のGo fixtureは `decimal,decimal -> decimal-or-null` だけ。
3. `plugin.Catalog.ResolveEnvironmentWithLock` で環境に選択されたexact Plugin ID/version/定義fingerprint/artifact digestを検証する。環境外・版違い・重複宣言・破損したLockを拒否する。
4. 呼出し側が明示した`go/reference`のBuiltin実装と同じPlugin identity、Operator契約、Targetであることを確認する。**SQL/Go間の暗黙の代替は行わない**。
5. 以上の確認後だけ、D1 `derivationref.Multiply`を実際に呼び出し、正確な十進数・NULL・未取得・参照先不在・複数関係・不正入力をそのまま返す。

`go test -count=1 ./reference/derivation-realization` で、2×300・0×300・0.2×0.5・NULL propagationとMissing区別・不正小数を確認する。さらにRule IDを変更せず新Plugin版がCatalogへ現れても環境Lockの旧版だけを選択すること、22種類のPin/Plugin/Operator/Target/effect欠落負例で**実装関数が呼ばれない**ことを確認する。

この実装はM1の**一実現方式・明示選択**のみ。GoからSQL/Wasmへの計算意味の同等性は未検証。SQL pushdown/Native expressionを書き換えず、D2のQuery計画、D3の鮮度、D4のBinding、D5の認可、および#1129のwrite/transactionへ一切変更を加えない。

HostがRule/Plugin/Artifactを承認した事実やCapability Grantの検証、AI生成プラグインの安全性、実ProviderのRuntime dispatch、暗号学的artifact真正性、実DB・Windows・WasmはこのReferenceだけでは成立しない。Go実装の契約試験に合格したことを「任意のProviderが業務的に同等」とは扱わない。
