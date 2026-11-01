# reference-type-reorg-impact（#1806）

意味の型を統合する案について、影響を受ける要素を一覧する読取り専用の参照です。型の参照、対応付け、Workspaceは変更しません。

~~~text
go run ./tools/reference-type-reorg-impact/cmd/tei-type-reorg-impact \
  -workspace reference/workspace -from <統合元の型ID,...> -into <統合先の型ID>
~~~

## 型分割の読取り専用モード

```text
go run ./tools/reference-type-reorg-impact/cmd/tei-type-reorg-impact \
  -workspace reference/workspace -split -from <元型ID> -into <移行先型ID1,移行先型ID2>
```

- 型分割は影響一覧と未割当診断を出す参照機能です。現行CLIは項目別の再割当を受け取りません。
- **`-split` と `-candidate-out` の併用はエラーで拒否**します。属性別移行先を保持せず統合用の候補へ写すと誤った型付替えになるためです。
- `SplitImpact` の結果には `split-candidate-unsupported` 診断を含め、内部APIから統合専用 `Candidate` に渡した場合も候補化を拒否します。分割の分析結果はJSONで参照できますが、この診断を意味上の異常や型自体の不正とは解釈しません。
- 型統合の `-candidate-out` は引き続き利用できます。いずれも受理やSource書換えは別の明示操作です。

## 出力

- 統合元の型を使う属性
- 統合元を基にした別名の型
- 影響を受ける候補キー、関数従属性、関連のキー
- 診断と、構造の版（`structureRevision`）

## 守っている境界

- `reviewRequired`は常に`true`、`applied`は常に`false`です。表現が同じでも、意味が同じとは判断しません。
- 名称が異なる型には`symbol-differs-meaning-unconfirmed`を返します。
- 基の表現が異なる場合は`underlying-representation-differs`を返します。
- 関連のキーの型がすでに異なる場合は`relation-key-type-already-differs`を返します。統合でその差が見えなくなる場合は`relation-key-difference-would-be-hidden`を併せて返します。
- 統合元の属性の物理的な格納型が異なる場合は`physical-storage-differs`を返します。物理型の変更は既存の移行の安全境界で扱います。
- 統合元に基本型を指定した場合、統合先が自分自身の場合、未定義の型を指定した場合は診断にします。

## 未実装

- 変更案の受理（#267）の実行と、受理後の再投影
- 導出規則、Validationからの型の利用の追跡
- 型の分割の変更候補生成・受理と型の置換（分割の影響一覧は実装済み）

## 受理境界への接続（#1935）

統合案は、Shared Acceptanceの`set-attribute-type`演算の変更候補（統合元の型を使う属性ごとに一件）へ変換できます。

~~~text
REV=$(go run ./tools/reference-shared-acceptance revision -workspace <対象> | grep -o 'sha256:[0-9a-f]\{64\}')
go run ./tools/reference-type-reorg-impact/cmd/tei-type-reorg-impact -workspace <対象> \
  -from sales.order_line_id -into sales.order_id \
  -candidate-out candidate.json -base-revision "$REV"
go run ./tools/reference-shared-acceptance preview -workspace <対象> -candidate candidate.json
~~~

- 候補は、出力先が既にあれば作りません。受理（`accept`）はこのツールでは行いません。
- 次の診断が一つでもある統合案は候補にしません: `underlying-representation-differs`、`physical-storage-differs`、`unknown-target-type`、`unknown-source-type`、`merge-into-itself`、`source-is-primitive`。統合元を基にした別名の型が残る統合、統合元を使う属性がない統合も拒否します。
- `symbol-differs-meaning-unconfirmed`と、関連のキーの型に関する診断は、候補を妨げません。人が`Preview → Review → Accept`で判断します。
- 影響の構造の版と、受理境界の`baseModelRevision`は別の値です。後者を`-base-revision`で渡します。
- `sales.order_line_id`を`sales.order_id`へ統合する候補が、`preview`の全11段階を通ることを手元で確認しました（`accept`は実行していません）。基の表現が異なる統合は、候補を作る前に拒否されます。
