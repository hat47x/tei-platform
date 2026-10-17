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

## 実差分による限定的な計測例（2026年10月17日）

[Issue #1251](https://github.com/hat47x/tei-platform/issues/1251)の受入条件「新しいプラグインを追加したときの既存Coreの変更面積」について、マージ済みPRの**固定された比較コミット**を確認しました。GitHubのPR変更ファイル一覧とコミット間の差分が一致することを確認し、上記の計測ツールと同じパス分類を適用した記録です。ここで`go run`を実行したという記録ではありません。

| 対象 | 変更パス数 | Pluginの製品Goソース | Core候補の製品Goソース | Core候補のGoテスト | その他の変更パス |
| --- | ---: | ---: | ---: | ---: | ---: |
| [PR #1316](https://github.com/hat47x/tei-platform/pull/1316) 独立プロセスPluginを追加 | 9 | 1 | 0 | 0 | 6 |
| [PR #1886](https://github.com/hat47x/tei-platform/pull/1886) 案件Pluginの独立構築を検査 | 3 | 0 | 0 | 0 | 2 |
| [PR #1904](https://github.com/hat47x/tei-platform/pull/1904) 案件Pluginを別Goモジュールから利用 | 2 | 0 | 0 | 0 | 1 |

各行の残りのパスは、計測ツール上の`plugins/<名前>/`にあるREADMEやテストなどです。`otherPaths`へ重ねて計上していません。

比較したコミットは次のとおりです。

- PR #1316: `2a26452f8b7b7a79e99d9712a9e6eda7f4c6b142` → `8687422047d141ed7bafe09a8ed2e70e89235c5b`
- PR #1886: `65339f31dfd805aa2c26f50e48abe2d68f1a7be2` → `35c093b053633c220e88eda2406d50b9dd3521dc`
- PR #1904: `8bb27a1d85c74a45f72affdcc6a596e2777a19b7` → `c9a31efca3142fa8a8bf9ca71f3b5ba29c84fa67`

これらの差分には、測定対象に含めたCore候補の製品Goソース変更はありませんでした。**「変更が0件」は、機能追加時に将来もCoreを変えなくてよいという保証ではありません。** また、#1886と#1904は既存案件プラグインの構築・検証強化であり、新規プラグインの製品ソース追加ではありません。変更の必要性や理由は差分の件数だけでは判断できません。

これはGitのパス単位の測定で、実行時依存・生成された成果物・Contract適合・正式なHostの受理を実測したものではありません。今回の資料追加を根拠にIssue #1251全体の受入完了を宣言しません。
