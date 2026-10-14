# E22：SUIの操作をTEIに結び付ける最小構成候補

- **管理**：[Issue #1016](https://github.com/hat47x/tei-platform/issues/1016)。実際のSUI操作・本人確認・保存の実装は[Issue #993](https://github.com/hat47x/tei-platform/issues/993)が所有する。
- **範囲**：実在するSUI Stage 0の意味定義に、**試験中だけ**TEI側の最小Application・Command・Capabilityを追加し、既存のWorkspace LoaderとDriverで解決できるかを確認する。既存のSUIのソース、正式なTEI Workspace、Native Canvas、保存先には変更を加えない。
- **実装**：[Go回帰試験](e22_native_application_candidate_test.go)。この検査の成功は、SUIの完成したアプリケーション・操作定義・データ保存・認証資格・実行権限を示さない。

## 何を埋めるのか

[E20](E20_DUAL_CONSUMER_ACTION_WITNESS.md)は、TEIの実受注Commandと**試験内の独自Resolverが返すSUI候補Command**を一つの共通Hostへ接続した。異なるDomainの操作をHostが識別できることは確認できたが、SUIの候補CommandをTEI自身のSourceから読み取って解決したわけではなかった。

E22では、この試験専用Resolverを使わず、**既存のDSL → Workspace Loader → Canonical World → Driver**だけでSUI側の最小Actionが解決できるかを確かめる。SUIのCardやDocumentをTEI Coreへ移植する試験ではない。

## 加えるのは必要最小限の候補だけ

実[Stage 0](../sui-adoption/workspace/tei.workspace.yaml)には`semantic/sui/island.tei`があり、アプリケーション定義はない。試験は`t.TempDir`に元の意味ファイルを複製し、作業領域の参照元を次の2種類に設定する。

```yaml
workspace: sui_composition_e22_candidate
sources:
  - semantic
  - applications
```

試験が追加するTEI宣言は次の3ファイルである。

```tei
// semantic/sui/native-action.tei
namespace sui
capability sui.move {
}

// applications/sui/application.tei
application sui {
    uses semantic sui
}

// applications/sui/patterns/move.tei
pattern sui.move command {
    capability sui.move
}
```

この`sui.move`は**入力・出力と効果が未定義の接続候補**である。既存SUIの`cardId/x/y`を意味的に記述し、型と整合を保証する契約ではない。実際のPayloadの検証とDocument変更はSUIが所有する。TEIへの正式な受理・実装Providerの選択・現行Authorityも存在しない。

書き込み対象は隔離した一時ディレクトリのみ。元のSUI Stage 0、SUI側リポジトリ、TEI製品Sourceは変更しない。

## 検証する観点

1. 元のStage 0にはApplicationがないため、`sui.move`を既存Driverから解決できない。
2. 試験用の3宣言を追加すると、`workspace.Load`と`driver.NewResolver`が`sui.move`をCommandとして解決し、既存のSource出典を取得できる。
3. 実`order_entry.submit_order`も同じ既存Driverで引き続き解決できる。SUI側の別のWorldに`order_entry`を推測で追加しない。
4. Capability宣言が欠ける、同じCommandが二重に定義される場合には、名前や読取り順序から勝手に一件を選ばない。
5. 解決に成功しても現在の認可器を持たないHostは書込みを拒否し、保存先に副作用を残さない。

この試験は、実行可能なBinding・プロファイル、外部証拠、実セッションの署名・Audience・権限、SUIの本物の保存先に接続するものではない。

## 計測できる部分と、まだできない部分

| 観点 | E22で数えられること | まだ測れないこと |
| --- | --- | --- |
| TEIへの試験候補追加 | 新しい意味・Application・Pattern宣言が各1ファイル、作業領域の参照元設定1件 | 実SUIでの構成作業の総量、既存SUI Native Adapterの変更量 |
| 既存機構の再利用 | Loader、Canonical World、Driver、共通Hostを新設せずに使用 | 実TargetとUI Kit／Frameworkの選択、運用上の互換性 |
| 入力・保存の意味 | Capabilityの空の契約を**未定義として観測** | SUI側のPayload型、Document・Revision・島所属の意味をTEI契約として充足したか |
| 実行権限 | 認可器なしのHost拒否を試験する | 署名検証済みの本人委譲、現在のResource権限、権限失効と原子的な保存 |
| 第二利用先 | 実受注明細とSUI候補を既存Driverでそれぞれ解決する | 一つの正式な構成プロファイルで両実アプリが動くこと |

**導かれる判断**：非業務アプリを既存のApplication／Commandへ接続するために、新しいCore種別や第二Resolverを導入する必要性は、今回の範囲では確認されていない。一方、`capability sui.move {}`を最終契約とみなすのも不適切であり、Native Payloadの境界・本人資格・Target採用という残余判断は引き続き明示的に未解決とする。

## 実行と採用条件

```sh
go test -count=1 ./reference/application-composition -run '^TestE22'
```

検査で成功しても、E22は**試験専用の接続候補を既存構造で表現できた**という局所的な証拠に限る。完全な構成の採否判断には[E21](E21_DUAL_CONSUMER_COMPOSITION_GAP.md)に記した二つの実利用先の接続費用と実Target検証が必要である。

`#993`で扱う実SUIのAction Payload・本人確認・Document保存と、`#1016`で扱う構成の選択・参照・更新管理を混同しない。新たな構成DSL、Lock、独立したProvider解決は導入しない。
