# SUIの操作をTEI Actionへ接続する参照実装

この実装は、非業務アプリケーションの独自UIをTEI ApplicationのActionへ接続する際の境界を検証するものです。**本番用のCanvas接続、SUIの移行、永続化、実セッション認可は実装していません。**

## 所有する責務

| 関心事 | 所有者・現状 |
| --- | --- |
| ポインター操作、選択、ドラッグ中の表示、取消し | SUIの既存React Canvas。TEIには未接続 |
| ApplicationとActionの対応 | 参照用`Host` |
| 宣言済みCommandの解決 | `CanonicalCommandGate`。`driver.Resolver.ResolvePattern`へ接続可能 |
| 実行時の認可 | `CapabilityAuthorizer`。既存`security.authorize`を使用するが、実セッションは未接続 |
| Documentの識別、Revision照合、更新 | アプリケーション所有の`Store`。今回の実装はメモリ内の模擬保存 |
| SUIのDocumentV1、島への所属、履歴、SafeMode | SUIの既存実装。TEIには移植しない |

`Host.Commit`はApplication ID、Document ID、Action ID、期待Revision、アプリケーション固有のPayloadを受け取ります。更新前にActionの対応と認可を確認し、保存側へ委譲します。**Hostだけでは同時更新を防げません。** Revisionの原子的な照合と更新はStoreの責務です。

## SUIの実コードとの対応

SUI `03_Implement/frontend/src/domain/types.ts` の`DocumentV1`では、Cardの座標は`card.x`、`card.y`です。`card.position`という構造ではありません。参照用`JSONDocumentStore`はこの座標形式に合わせ、`version: 1`と文書本体の`id`を確認します。Sourceは`card.meta.source`、保留は`card.holdState`、レビューは`reviewAttribution`などを保持します。未知のEdge種別も移動によって変更しません。

ただし、参照用StoreはSUIのTypeScript検証器を呼び出さず、島への自動所属やUndo/Redoも再現しません。構造が似ていることと、SUIのドメイン処理に接続したことを混同しないでください。

**Canvas接続前に解消すべき差分**（SUI mainを2026-10-10に確認）：

- `src/canvas/CardView.tsx`の`handlePointerMove`は移動中の各イベントで`onMove`を呼びます。Drop時だけ呼び出すAPIではありません。
- `src/App.tsx`の`handleCardMove`は移動中に`setHistory`と`setIsDirty`を実行します。
- 同じ`App.tsx`では`pointerup`と`pointercancel`の両方で`commitCardDragSnapshot`を呼びます。現状の取消しを「永続状態に影響しない」と扱うことはできません。
- Drop時の処理はCardの移動だけでなく、島への所属変更を同じUndo操作にまとめます。単純な`x/y`更新だけでは動作互換になりません。

したがって、TEI接続時は既存のCanvasを置き換えるのではなく、**局所プレビュー／取消し／確定**を分けるSUI側の接続口を用意し、確定時にだけ`Host.Commit`へ最終座標とRevisionを渡す必要があります。認可拒否やRevision競合では、プレビューを破棄し、Documentを再取得します。SUIの既存島所属・Undo/Redo規則はSUI側のAction実装に残します。

## 実行

```sh
go test -race -count=1 ./tools/reference-sui-action-boundary/actionboundary
```

このREADMEはテストの成功を主張しません。統合時は対象コミットを固定し、`tools/repository-health-check.sh`全項目のPASSと実行前後のclean状態を別途確認してください。

## 検証範囲と未実装

参照テストは正常更新、認可拒否、Revision競合、同時更新、Document IDの不一致、未知Card、非有限座標、不正Document構造、無関係な情報の保持を対象にします。第二Applicationのテストは模擬Storeを用いており、汎用プラグインの本番動作を証明しません。

未実装：実際のReact Canvasとの接続、SUIのDocumentV1検証器と保存機構の呼出し、実セッションからのSubject解決、実ランタイム権限、ブラウザー配備、SafeModeとExport保護、性能測定、変更箇所数の比較。

作業窓口はGitHub Issue #993、設計の正本は`plan/issue/000052-sui-reference-adoption-and-plugin-surface-evaluation.md`です。GitHub Issue #52は別件です。
