# mainブランチの保護

`main-pr-only.json` は、GitHubの管理者が適用するためのルールセット定義です。**ファイルをリポジトリに置くだけでは、保護設定は変更されません。** Issue #1189の未完了作業として扱います。

## 適用する規則

- `main` の変更は原則プルリクエストを経由する。
- 履歴を壊す強制更新とブランチ削除を拒否する。
- 例外的な迂回権限は設定しない。
- 承認レビューは0件とする。保守担当が一人でも変更を統合できる構成を維持する。
- `repository-health` は自動実行される診断として残すが、現行の開発規則に従い、すべての変更に対する一律の必須チェックにはしない。

## 管理者による適用

必要な権限は、対象リポジトリに対する管理権限です。GitHub CLIと管理操作が可能な認証を使います。

まず既存のルールセットを確認します。

```bash
gh auth status
gh api repos/hat47x/tei-platform/rulesets --jq '.[] | {id, name, enforcement}'
```

同じ目的のルールがまだ存在せず、定義を確認した場合に限り、リポジトリのルートから**一度だけ**次を実行します。

```bash
gh api --method POST repos/hat47x/tei-platform/rulesets \
  --input .github/rulesets/main-pr-only.json
```

重複ルールが既にある場合は新規作成せず、その内容と適用対象を確認してください。適用後、次を読み戻します。

```bash
gh api repos/hat47x/tei-platform/rulesets \
  --jq '.[] | {id, name, enforcement, target, rules: [.rules[].type]}'
gh api repos/hat47x/tei-platform/rules/branches/main
```

`main` 上で危険な書込み・削除・強制更新を試すことはしません。拒否動作の実証が必要なら、同じ規則を一時的な試験ブランチへ適用し、安全な実験コミットの直接pushが拒否されることを確認してから後片付けをします。

## 適用前後の読み取り専用検査

管理設定の有効化とは独立して、次のコマンドで定義自体を検査できます。Python標準ライブラリだけを使い、GitHubを変更しません。

```bash
python3 .github/rulesets/verify-main-protection.py --self-test
python3 .github/rulesets/verify-main-protection.py --local
```

管理者が保護設定を適用した後は、GitHub CLIで認証してから実設定を読み戻します。

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
