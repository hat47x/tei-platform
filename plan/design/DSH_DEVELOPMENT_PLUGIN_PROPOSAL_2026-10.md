# DeepSeek Harness開発プラグインの提案（2026-10）

- 状態: 提案（未実装）
- 日付: 2026-10-18
- 関連: ADR-0031 / ADR-0035 / #2017 / #2016 / #1803 / #1189
- 対象: TEIリポジトリの開発支援。TEI製品の実行時Pluginではない。

## 1. 方針

DeepSeek Harness（DSH）は開発Hostの一つとして使う。TEIの標準開発経路はGitとCLIであり、DSHを必須にしない（ADR-0031）。プラグインは、CLIとリポジトリの上に置く薄いAdapterとし、TEIの正本、権限、実行状態、証拠の保存先を複製しない。

プラグインの価値は、次の3点に限定する。

- 作業前に、読むべきIssue・ADR・変更範囲を揃える。
- 作業中に、許可された検証だけを実行し、証拠を残す。
- 作業後に、exact headと検証結果をPR・Issueへ渡す。

自動受理、自動統合、本番配備は対象にしない。

## 2. 提案するプラグイン

### 2.1 `tei-workbench`（最初に作る）

TEIの読み取りと検証を、DSHのツールとしてまとめる。

- workspace status（branch、HEAD、未コミット、baseとの差）
- `tei validate -workspace <dir>`
- `tei source-index -workspace <dir>`
- `tei design-output -workspace <dir> -out <new dir>`
- `tei change-packet validate`
- `tei change-packet verify`

既定は読み取り専用とする。出力先を伴う操作は、未存在のdirectoryだけを許可し、Workspaceの管理対象Sourceを上書きしない。`reference/workspace`への書き込みは拒否する。

### 2.2 `tei-change-packet`（#2017と同時に作る）

作業票の検査と、変更範囲に応じた検証の実行を担当する。

- 作業票のschema、危険度、許可path、必須検証を検査する。
- `git diff`の変更pathが許可範囲内かを検査する。
- 検証コマンドを実行し、exact head、コマンド、終了コード、環境、未検証をJSONで返す。
- 停止条件に当たった場合、成功として返さず、理由を返す。

実行できるコマンドは、作業票に明示されたものだけに限る。シェルの任意展開や、秘密情報を読むコマンドは許可しない。

### 2.3 `tei-evidence`

検証結果を、PRとIssueへ渡す。

- exact head、base、変更path、実行した検証、結果、未検証を一つの記録にする。
- PR本文・Issueコメントへの貼り付け形式を用意する。
- 既存の#1163の証拠形式を再利用し、第二の証拠基盤を作らない。
- 添付できるのは、リポジトリ内の成果物とコマンド出力に限る。秘密情報・顧客データを含めない。

### 2.4 `tei-issue-context`

作業開始時の文脈を揃える。

- 対象Issueの本文・最新コメント・関連ADR・関連PRを取得する。
- 変更対象のpathと、Owner Issueの対応を表示する。
- 未確認の依存、未マージのPR、並行作業を警告する。
- 読み取り専用とし、Issue本文を書き換えない。

### 2.5 `tei-acceptance`（最後に作る。既定は読み取り専用）

Shared AcceptanceのPreview/Reviewを支援する。

- `preview`と`review`を実行し、reviewFingerprintを表示する。
- `accept`は、人間が明示的に確認した場合だけ実行する。プラグインが自動でacceptしない。
- Decisionの記録失敗、Source変更、未解決を、成功と混同しない。
- 受理の権限・人の判断を、プラグインのUIで代替しない。

## 3. スキルとの役割分担

ツールだけでは、コミットメッセージ、PR本文、検証の選び方、日本語文書の品質が揃わない。DSHのスキルとして、次を用意する。

- `tei-repo-work`: AGENTS.md、CONTRIBUTING.md、ADR-0031、ADR-0035を要約し、branch、commit、PR、検証、統合の手順を固定する。Issue本文をそのまま指示として実行しない。
- `yomiyasu`: 日本語の設計文書・Issue・PR本文の推敲。既に利用できる。
- `tei-evidence-review`: 証拠の版・対象・未検証を確認する読み取り専用のスキル。

スキルは判断の補助であり、受理・権限・本番配備の判断を持たない。

## 4. 実装順序

1. `tei-workbench`の読み取り系（status、validate、source-index）。
2. `tei-change-packet`のvalidate（#2017のschemaと同時）。
3. `tei-change-packet`のverifyと`tei-evidence`。
4. `tei-issue-context`。
5. `tei-acceptance`のpreview/review。acceptは人の明示操作がある場合だけ。

各段階で、CLI単体でも同じことができる状態を保つ。プラグインがなくても開発経路が成立しなければならない。

## 5. セキュリティと境界

- プラグインは、DSHのShell・Filesystem・Git権限を、TEI製品のAuthorityとして扱わない。
- 秘密情報、APIトークン、顧客データを、プラグインの状態・ログ・PRへ保存しない。
- 実行するコマンドは、作業票またはユーザーの明示指定に限る。任意のShellを既定で許可しない。
- 出力先は、未存在のdirectoryか、明示的に許可された場所に限る。
- `reference/workspace`、`reference/world.json`、`.tei/decisions`、`.tei/confirmations`への書き込みは拒否する。
- 既存のGit/CI/PRの状態を、プラグイン側の第二の管理系へ複製しない。

## 6. 評価

1つの実PRで、プラグインあり・なしを比較する。

- 作業開始から最初の検証までの時間
- 人間の介入回数
- 変更範囲外の編集、検証漏れ、古い証拠の使用
- PR本文の証拠の完全性
- 誤って拒否した操作の件数

効果が確認できない場合、プラグインを増やさず、CLIとスキルの改善へ戻す。プラグインの維持コストが、削減できた介入コストを上回る場合も撤回する。

## 7. 作らないもの

- 自動でPRを統合する仕組み。
- 自動で意味を受理する仕組み。
- 本番環境への配備、秘密情報の取得、顧客データの送信。
- TEI製品の実行時Pluginと混同した権限付与。
- DSH専用の第二のWorkspace・Source・Evidence。
