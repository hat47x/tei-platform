# GitHub Issues second-consumer profile

ISSUE-000084 / GitHub #977のE2-A Referenceです。Git text profileで見つけた交換項目がGit固有の都合に寄り過ぎていないかを、実際のGitHub IssueとPull Requestで確認します。

この文書はGitHub Issues Adapterのproduction仕様ではありません。GitHub固有のfieldをTEI Coreへ持ち込まず、第二consumerから共通部分を削るための記録です。

## Actual consumer

GitHub Issue #977をWork、Draft PR #978をResult / Artifact referenceとして利用しました。Issueの目的は利用者の依頼から生じていますが、起票操作はAgentがConnector経由で行いました。GitHubのnative authorは接続アカウントとして記録されるため、authorだけではWorkの由来と作成経路を区別できません。

```text
workRef      = github-issue://hat47x/tei-platform/977
origin.kind  = human
createdVia   = agent-connector
backendActor = connected GitHub account

resultRef    = github-pr://hat47x/tei-platform/978
```

`workRef` / `resultRef`はBackend instanceまで含めてAdapterが一意に解決できるopaque locatorの例です。TEI CoreがこのURI構造をparseすることは要求しません。

`backendActor`はBackend側の監査情報として有用ですが、Human-origin、Agent-origin、Acceptance authorityのどれにも自動変換しません。

## Operation mapping

| External Coordination operation | GitHub Issues / PRでの候補 | Referenceで確認した範囲 |
| --- | --- | --- |
| Work discovery | Issue search / list | Issueを検索・取得できる |
| Work read | Issue body / metadata | #977を取得できる |
| Result append | Issue comment | E1 / E2結果を#977へ追記した |
| Artifact reference | PR / commit / external URI | Draft PR #978へ参照できる |
| Backend state | Issue state / state reason | `open`を観測した |
| Human / Agent origin | explicit metadata | native authorだけでは不足 |
| Parent / relation | Issue / PR reference | URL / numberによる参照を利用した |
| Exclusive lease | 未確定 | GitHub Issue自体には要求しない |
| Canonical Acceptance | 対応させない | ADR-0022 / ISSUE-000027へ戻す |
| Runtime Authority | 対応させない | Runtime / Resource境界へ戻す |

## Stateを黙って正規化しない

Git text E1ではReference内の`coordination.status`として`ready`と`completed`を使えます。一方、GitHubの`open / closed`を同じ語へ機械的に変換する根拠はありません。`closed`であっても、完了、重複、対象外等の意味があり得ます。

このため、第二consumerの段階ではBackend native stateをそのまま保持します。共通lifecycleが本当に必要になった場合に限り、Adapterが明示的なmappingと元stateを保持する方式を候補にします。Backend stateはCanonical AcceptanceやProduct completionの証拠として扱いません。

## E0 contractから残す候補

二つのconsumerを比べると、backend-instance-scopedなopaque `workRef` / 任意の`resultRef`、`origin`、`createdVia`、人が読めるobjective、任意のSource context、Result / Artifact referenceは引き続き有用です。ただしSource contextはTask Trackerで常に存在するとは限らないためoptionalのままにします。

Backend author、label、Issue type、project、state reason等はAdapter-localな情報として保持できますが、現時点で共通contractの必須fieldにはしません。Agent Session、model、token / quota、credential、Git branch ownershipも追加しません。

## Authority

Issueがopen / closedであること、PRがDraft / Ready / mergedであること、commentが投稿されたことはCoordination上の事実です。それだけでCanonical meaning、Acceptance authority、Runtime authority、Resource authorityは変化しません。

GitHub上のactorが管理者権限を持つ場合でも、そのRepository権限をTEI Application Runtime authorityへ読み替えません。Canonical AcceptanceやRuntime Authorityを否定するbooleanをexchange documentへ追加することもしません。Authorityはこのexchange contractの外側にあります。

## Non-claims

このpressureだけではGitHub Issuesを標準Backendに決めません。GitHub AppによるService Identity、Issue type、custom field、sub-issue、Webhook、rate limit、atomic claim等もproduction要件として固定しません。

E2の次段では、Task Tracker以外のProfileが必要かを判断します。Box / SharePoint等のArtifact-heavy Backendを追加する場合も、GitHub向けに増やしたfieldを前提にせず、Artifact referenceとBackend capabilityの境界から比較します。
