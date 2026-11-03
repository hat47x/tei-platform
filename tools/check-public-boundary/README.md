# 公開境界のマーカー検査

公開リポジトリの追跡ファイルに、非公開資料を指すマーカーが入っていないかを検査します。

## 目的

[`docs/documentation-quality.md`](../../docs/documentation-quality.md) の§11.1は、公開文書に非公開資料の名前・所在・集計を書かないと定めています。この検査は、そのうち**名前と所在**について、追跡ファイルへの逆戻りを検出します（[#2342](https://github.com/hat47x/tei-platform/issues/2342)、[#2436](https://github.com/hat47x/tei-platform/issues/2436)）。

## 実行

```bash
python3 tools/check-public-boundary/check.py --root .
```

- 一致が無ければ `public boundary: ok` を出して終了コード0
- 一致があれば、ファイル名とマーカーの種別を標準エラーへ出して終了コード1

## 検査するマーカー

次の3種です。値は、検査器自身が一致しないよう、ソース上で分割して組み立てています。

| 種別 | 内容 |
| --- | --- |
| 非公開リポジトリ名 | 非公開資料庫のリポジトリ名 |
| アーカイブ内パス | 保存庫のアーカイブ用ディレクトリ名 |
| 調査記録のパス | 調査記録のディレクトリ名 |

集計値（件数・割合）は検査しません。数値は一般的な文字列と衝突しやすく、規則の趣旨（存在と構造を明かさない）に対して過剰な一致を生むためです。集計値の扱いは文書のレビューで確認します。

## 対象外

- Issue・PRの本文とコメント、コミットメッセージ、Git履歴。これらは書き換えると公開記録が変わるため、機械検査の対象にしません（[#2342](https://github.com/hat47x/tei-platform/issues/2342)の整理を参照）
- 検査器自身（`check.py`）とその試験（`check_test.py`）
- Git管理外のファイル

## 試験

```bash
python3 tools/check-public-boundary/check_test.py
```

一時ディレクトリにGitリポジトリを作り、マーカーを入れたファイルで終了コード1になること、入れなければ0になることを確認します。公開ツリーは汚しません。

## CI

`.github/workflows/tei-repository-health-ci.yml`の`repository-health`ジョブで、`repository-health-check.sh`の前に試験と検査を実行します。必須ステータスチェックにはしません（開発規則どおり）。

## 境界

- この検査は回帰の防止用であり、GitHubのsecret scanningや、参照モデルの利用条件の確認（[#941](https://github.com/hat47x/tei-platform/issues/941)）の代わりではありません
- 許可リストは持ちません。追加が必要になった場合は、なぜ公開してよいかをIssueへ記録します
- 検査できるのは、追跡済みファイルに現れる既知のマーカーだけです。未知の表記や、画像・バイナリの中身は対象外です
