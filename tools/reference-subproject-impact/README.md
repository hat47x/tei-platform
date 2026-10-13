# サブプロジェクト変更面積の参考計測

[Issue #1251](https://github.com/hat47x/tei-platform/issues/1251)の「新しいプラグインを追加したとき、既存コアを何か所変更したか」を、Gitの**二つの確定コミットの実差分**から測定する参照コマンドです。測定範囲は限定的で、製品上のプラグイン境界や契約の正本を追加しません。

## 実行方法

リポジトリのルートから、比較対象を**40桁の確定コミットSHA**で指定します。可動ブランチ名は入力として受け付けません。

```sh
go test -count=1 ./tools/reference-subproject-impact
go run ./tools/reference-subproject-impact -base <base-commit-sha> -head <candidate-commit-sha>
```

差分は`git diff --no-renames --name-only -z`で取得します。NUL区切りによって改行や特殊文字を持つファイル名でも誤分割せず、移動では移動元と移動先の両方を計上します。最初と最後のリビジョンを固定し、同時に別のチャットが`main`を進めても計測範囲が変わりません。

JSON出力は以下を区別します。

- `pluginSubprojects`：変更された`plugins/<名前>/`の名前（READMEだけの変更も含む）
- `pluginSourcePaths`：変更されたプラグインの製品用Goソース（テストファイルを除く）
- `coreCandidateSources`：`core/`・`internal/core/`・`internal/composition/`・`internal/subject/`・`internal/permission/`・`internal/semantic/`・`internal/stdlib/`にある製品用Goソースの変更
- `coreCandidateTests`：同じ領域の`*_test.go`の変更
- `otherPaths`：上記の領域に含まれない変更、またはコア候補領域の非Goファイルの変更

プラグインREADMEなどはサブプロジェクト名の集合に反映しますが、`otherPaths`へ重複加算しません。このため各項目の数を合算しても`changedPaths`にはなりません。

## 判定と限界

計測コマンドは**Core候補の製品ソース変更がゼロであることを合否条件にしません**。契約を正しく拡張する変更が必要な場合もあり、反対に差分がゼロでも新プラグインが契約に適合する証拠にはなりません。変更箇所とその理由を人間・AIが確認するための事実記録です。

`coreCandidateSources`は測定用に列挙した暫定パスであり、将来の最小コアの完成形を宣言しません。`internal/plugin/`などを一律に「コア」と認定しないため、ここでは`otherPaths`へ分類します。コアの境界自体は[ADR-0032](../../plan/adr/0032-composition-kernel-replacement.md)と[ADR-0033](../../plan/adr/0033-core-and-plugin-subprojects.md)で管理します。

このツールは実際のGo依存閉包、動的ロード、生成器、ビルド成果物の指紋、契約準拠、ホスト権限を検証しません。ビルド入力の追跡は[#1089](https://github.com/hat47x/tei-platform/issues/1089)等の別機構の責務です。必要な動的機構を含む製品実装に移すまで、変更面積は独立性の**補助指標**として扱います。
