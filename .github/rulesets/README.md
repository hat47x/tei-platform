# mainブランチの保護

`main-pr-only.json` は、GitHubのルールセットとして適用するための定義です。ファイルを置くだけでは適用されないため、適用と読み戻しをコマンドで行います。適用は管理権限を持つ認証で行い、人の判断を待ちません。Issue #1189の作業として扱います。

## 適用する規則

- `main` の変更はプルリクエストを経由する。
- 履歴を壊す強制更新とブランチ削除を拒否する。
- 例外的な迂回権限は設定しない。
- 承認レビューは0件とする。主担当者が一人でも変更を統合できる構成を維持する。
- 許可するマージ方式は `merge`・`squash`・`rebase` の3種類。GitHub側の現在の許可設定とも一致させる。
- `repository-health` は自動実行される診断として残すが、現行の開発規則に従い、すべての変更に対する一律の必須チェックにはしない。

## 適用

既存のルールセットを確認します。

```bash
gh auth status
gh api repos/hat47x/tei-platform/rulesets --jq '.[] | {id, name, enforcement}'
```

同じ目的のルールがまだ存在しない場合は、リポジトリのルートから作成します。

```bash
gh api --method POST repos/hat47x/tei-platform/rulesets \
  --input .github/rulesets/main-pr-only.json
```

既存のルールセットがある場合は、作成せずに内容を更新します。定義と実設定が一致しないときは、次の手順で既存IDへ反映します。

```bash
# 一覧からIDを得る
gh api repos/hat47x/tei-platform/rulesets --jq '.[] | {id, name, enforcement}'
# 得たIDで定義を反映する
gh api --method PUT repos/hat47x/tei-platform/rulesets/<RULESET_ID> \
  --input .github/rulesets/main-pr-only.json
```

重複ルールがある場合は、重複分を削除してから再適用します。削除は `gh api --method DELETE repos/hat47x/tei-platform/rulesets/<RULESET_ID>` で行います。

適用後、次を読み戻します。**一覧APIは規則の詳細まで返すとは限らないため、一覧から得たIDで詳細を取得**するか、次節の `--live` 検査を使ってください。

```bash
gh api repos/hat47x/tei-platform/rulesets \
  --jq '.[] | {id, name, enforcement, target}'
gh api repos/hat47x/tei-platform/rules/branches/main
```

ルールセットの詳細は、一覧で取得した実際のIDを指定して確認します。

```bash
gh api repos/hat47x/tei-platform/rulesets/<RULESET_ID>
```

`main` 上で危険な書込み・削除・強制更新を試すことはしません。拒否動作の実証が必要なら、同じ規則を一時的な試験ブランチへ適用し、安全な実験コミットの直接pushが拒否されることを確認してから後片付けをします。

## 適用前後の読み取り専用検査

管理設定の有効化とは独立して、次のコマンドで定義自体を検査できます。Python標準ライブラリだけを使い、GitHubを変更しません。

```bash
python3 .github/rulesets/verify-main-protection.py --self-test
python3 .github/rulesets/verify-main-protection.py --local
python3 -m unittest discover -s .github/rulesets -p 'test_*.py' -v
```

GitHub APIの一覧取得は100件ずつページ送りし、同名ルールの重複や詳細IDの不一致、権限不足による必須情報の欠落を**不合格または判定不能**として扱います。追加の模擬API試験はネットワークや管理者権限を必要としません。この検査はCI（`tei-ruleset-contract-ci.yml`）でも実行します。

適用後は、GitHub CLIで認証してから実設定を読み戻します。

```bash
gh auth status
python3 .github/rulesets/verify-main-protection.py --live
```

実設定の検査は、名称、対象、強制状態、PR必須、履歴保護、迂回権限、レビュー数を比較します。適用前・設定不一致・取得権限不足の場合には成功としません。GitHub APIは閲覧権限が足りないと迂回権限を省略するため、その場合も**未判定**として扱います。検査が成功しても、サーバーが不正なpushを拒否したことの実証にはなりません。

## 運用上の注意

- この定義はGitHub上の実設定と自動同期されません。変更時はIssue #1189へ適用日時・ルールセットID・設定内容・検証結果を記録してください。
- 適用すると、直接mainを更新する既存の自動化も拒否される可能性があります。GitHub Actionsなどの書込み経路を確認したうえで有効化してください。
- 全リポジトリの健全性確認を一律の統合条件に変更する場合は、先に `AGENTS.md` と `CONTRIBUTING.md` の方針を更新し、範囲別の検証規則との整合を取ってください。
- GitHubでの保護状態は `Settings → Rules → Rulesets` と適用済み規則のAPIで検証します。ルールセットファイルの存在やCIの成功だけで「保護済み」とは判定しません。

参考：[GitHubのルールセット作成手順](https://docs.github.com/en/repositories/configuring-branches-and-merges-in-your-repository/managing-rulesets/creating-rulesets-for-a-repository)。
