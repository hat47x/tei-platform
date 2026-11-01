# check-public-boundary

公開文書へ非公開資料の名前・所在を書かない規則（[`docs/documentation-quality.md`](../../../docs/documentation-quality.md) §11.1、#2342で決定）を、機械で検査する（#2436）。

## 使い方

```sh
python3 tools/check-public-boundary/check.py
python3 tools/check-public-boundary/check_test.py
```

追跡対象のファイル（`git ls-files`）を走査する。禁止語に一致したら、ファイル名と一致した語の種別をstderrへ出して終了コード1、一致しなければ`ok`を出して終了コード0。

## 禁止語の種類

語そのものはこの検査器が断片から組み立てて持つ。READMEには種別だけを書く。

| 種別 | 意味 |
| --- | --- |
| `repository-name` | 非公開資料庫のリポジトリ名 |
| `archive-path` | 非公開資料庫のアーカイブ内パス |
| `research-note-path` | 非公開資料庫の調査記録のパス |

集計値（件数・割合・合計）は、語の一致では検査できない。§11.1の「集計値を書かない」は機械検査の対象外で、書く側の注意に依る。

## 検査の対象外と、その理由

- **公開済みのIssue・PRの本文**: GitHub上にあり、検査器からは参照できない。公開済みの履歴を書き換えるかは#2342では決めていないため、 Issue・PR本文を対象にしない。
- **git履歴**: 検査は現在の追跡対象ファイルが対象で、過去のコミットは読まない。履歴の書換えはAGENTS.mdの規則（影響調査なしに履歴を書き換えない）に従い、この検査では扱わない。
- **検査器自身のファイル**（`check.py`、`check_test.py`）: 禁止語を断片として組み立てているため自己一致しないが、確実に対象から除く。

## CI接続

`tools/repository-health-check.sh`の検査項目として、`repository-health`CI（`.github/workflows/tei-repository-health-ci.yml`）から実行される。必須ステータスチェックにはしていない（開発規則どおり）。

## 境界

この検査は、公開境界の回帰を防ぐ補助であり、GitHubのsecret scanningや、参照モデルの利用条件の確認（#941）の代わりではない。許可リスト（SKIP）を足す場合は、なぜ公開してよいかをIssueへ記録する。
