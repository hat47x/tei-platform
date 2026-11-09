# reference-access-matrix（#1803）

Application × Data の**宣言された利用**の対応表を、受理済みの意味から導出する読取り専用の参照です。Workspaceは変更しません。

~~~text
go run ./tools/reference-access-matrix/cmd/tei-access-matrix -workspace reference/workspace            # JSON
go run ./tools/reference-access-matrix/cmd/tei-access-matrix -workspace reference/workspace -out matrix.md  # Markdown（既存ファイルは上書きしません）
~~~

## 読み取る宣言

- 参照、単票、一覧、ヘッダ明細のPatternが持つData: 参照（R）
- コマンドのPatternが指すCapabilityの効果（作成 / 更新）: 作成（C）、更新（U）

各欄に、宣言したPatternの識別子（`via`）を残します。

## 守っている境界

- 宣言であり、実際の副作用の観測ではありません。表にもその旨を書きます。
- 削除は現在のモデルで表現できません。空欄を「削除しない」と読ませないよう、表に明記します。
- Dataを持たないPattern、見つからないCapability・Data、効果を宣言しないCapabilityは、推測せず「読み取れなかった宣言」に列挙します。
- 導出元の構造の版を記録します。出力は決定的です。

## 未実装

- Relationだけを持つPattern（参照、ヘッダ明細）からのDataの特定。現在は読み取れなかった宣言として残ります
- 主題領域・作業・機能など、他の軸の対応表（Data以外の視点は#1125の担当）
- 実際の副作用の観測との比較（#1132）
