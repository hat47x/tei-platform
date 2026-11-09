# E4e：外部Semanticの選択と利用側Sourceを同じHost受理判断で照合する（参照実装）

- 所有課題：[外部Semantic参照 #1570](https://github.com/hat47x/tei-platform/issues/1570)
- 先行：[E4dの署名付き選択検査](README.md)、[Source受理境界 #267](https://github.com/hat47x/tei-platform/issues/267)
- 設計正本候補：[外部Semantic受理証拠の受渡し](../../plan/design/EXTERNAL_SEMANTIC_ACCEPTANCE_HANDOFF_2026-10.md)
- 実装：`current_project_meaning.go` と `current_project_meaning_test.go`
- **状態：合成Host Readerを用いる負例を含む参照試験。本番のHost Reader／業務意味受理／Runtime権限ではない。**

## なぜ同じ判断が必要か

`sourceacceptance.VerifyCurrent`は、利用側WorkspaceのManaged Source
（Source IndexとModelRevision）を、Hostが受理した現在のSource記録と比較する。
外部Providerの`semantic.Namespace`は利用側のManaged Sourceに含まれない。
従ってProviderが変更された場合、利用側Sourceが同じでも
そのままSource照合が成功する。この反例は`external_acceptance_boundary_test.go`で確認している。

E4eでは、`CurrentProjectMeaningReader`が**一回の現在のHost判定**
`CurrentProjectMeaning`として次の両方を返す契約を試す。

- `Source`：既存の`sourceacceptance.Record`（Application ID、DecisionRef、AuthorityRevision、ModelRevision、Managed Source一覧）。
- `Selected`：受理判断が対象とする外部Namespaceの**局所Alias、型付き`modelref.Ref`、正確な`SemanticDigest`**。

両者を別々のHost呼出しで取得したものから合成してはならない。
Hostは判断の発行元・権限・取消し、同一時点の一貫性を独立して保証する必要がある。
検査側はその保証を実装せず、信頼済みHostから返された一つのSnapshot内の照合だけをする。

## 読取り専用の検査

`InspectCurrentProjectMeaning(ctx, reader, applicationID, workspaceRoot, requested)`は以下を行う。

1. 利用側があらかじめ**明示した**`requested []ProjectMeaningRequest`（Aliasと型付き論理ResourceRef）を入力として受ける。Hostの受理記録からこの要求を逆算しない。既存`workspace.Load`でPath Importを含むWorkspaceを**一度だけ**読み込む。
2. 同じLoadから`workspace.BuildSourceIndex`を構築し、既存`validation.Validate`を確認。
3. Host Readerを一度呼び、同じ受理判断のSourceとExternal選択を取得。
4. 既存`sourceacceptance.VerifyCurrent`にSource照合を委譲。検証した`Record`のみ使用。
5. すべての`ResolvedImport`について、利用側の明示要求とHostの受理選択がAlias・**論理Scopeを含む**`modelref.Ref`で正確に一致し、Namespace IDとSemanticDigestが実際のLoadに一致するか一対一照合。余分・不足・曖昧な選択は拒否。
6. 不一致の場合、WorkspaceのWorldや受理記録を呼出側へ返さない。

`modelref.Ref.Scope`は論理参照Scopeであり、Composition ServiceのScopeではない。
現在のPath ImportにはProviderの契約版やPackage版がないため、
`modelref.Ref.Version`を持つ受理記録は検査不能として拒否する。
物理Source Pathは正本の意味IDやHost選択の照合値に含めない。
現行のAliasは利用側が選んだ局所の参照キーとして照合する。

これは`semanticDigest`の任意性を無くす変更ではない。
開発中のローカルWorkspaceはこれまで通り固定なしで読み込めるが、
**受理済み外部意味資産として使用する場合**はHostが正確な意味Digestを承認していることが別途必要、という扱いである。

## 負例と資格の限界

この実装はProviderのSemantic Digestだけが変わった場合、
旧Host判断からの照合を拒否する。一方、Providerの物理ファイル移動だけで
意味Digestが変わらなければ、業務意味が変わったと捏造しない。

拒否ケース：

- Host Reader不在・拒否・取消し
- Accepted Sourceの対象Application・Source revision・証拠が不一致
- Hostが未受理のNamespace、余分なNamespace、Alias誤り、空Alias、利用側の明示ScopeとHost判断の相違、型・ID・Versionの不整合
- 一致しないSemantic Digest、参照先の型・意味Validationエラー

検証コマンド（対象パッケージと既存受理部分のみ）：

```sh
go test -count=1 ./reference/semantic-composition-preflight ./internal/sourceacceptance ./internal/workspace
go vet ./reference/semantic-composition-preflight
```

### 実証できないこと

- **本当のHost受理判断の作成、人的承認、信頼済みReaderや取消し機構の実装**。
  合成Readerの`DecisionRef`は単なるFixtureであり、権限証拠ではない。
- CAS/BuildArtifactLockを選択した配布形態と`InspectPublished`の同一判断への結合。
  E4eが対象とするのは現行**Path Workspace Import**のみ。
- 複数Workspaceの完全な原子的Snapshot、検査後の改訂・差替えの防止。
  取得中のProvider内容変更や実行時現在権限は別途Hostが再評価する。
- 有効な`sourceacceptance.Record`からの独立した外部意味受理の**推論**。
  外部意味の受理はその判断に明記されていなければ成立しない。なお、`requested`は利用側が明示的に選んだ論理参照であり、Hostから返ってきた受理候補をそのままコピーして作ってはならない。
- ResourceRef Kindの正式採用、Production Profile Schema、別のRegistry・Resolver・Lock。

次段階は#267のShared Acceptanceで、同じ案件受理判断に外部Namespaceの
選択を添付するための正式な更新・保存・Current Readerを設計し、
`#1016`のApplication Composition Profileおよび`#951`のProvider選択と
接続する。E4eの合成Readerを正式なHostの代わりに置かない。
