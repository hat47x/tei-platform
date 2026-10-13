# E23：選択済み提供者と委譲先Actionの接続を検証する

**管理：** [Issue #1016](https://github.com/hat47x/tei-platform/issues/1016)。受注明細とSUIの比較に用いる、既存の構成・権限・通信境界を結ぶ参照試験。既存の[#993](https://github.com/hat47x/tei-platform/issues/993)の認証済みSUI実接続や、本番導入の実証ではない。

## E20から進めた点

[E20](E20_DUAL_CONSUMER_ACTION_WITNESS.md)は、共通Hostに二つのActionとメモリ内保存先を登録し、操作の所有範囲を比較した。今回は、そのうちSUI側の候補Actionについて、従来独立して検証されていた次の既存部品を**一つの要求の経路**で組み合わせる。

先行[E22：SUIの最小Application候補](E22_NATIVE_APPLICATION_CANDIDATE.md)が作る隔離Workspaceを読み、**DSL→Loader→Driver**で解決した候補Commandを使用する。E20の試験専用ResolverをSUI意味定義の正本として借用しない。

1. Web Hostが現在のセッション、発行者、偽造要求対策、操作権限を検査する。
2. 既存の構成台帳・明示選択・Coordinatorが、正確な契約ID、版、提供者、登録された成果物指紋と現在の選択状態を照合する。指紋の値を登録しただけで、実ファイルの真正性まで検証したとは扱わない。
3. 既存の共通Action受信口が、アプリケーション固有のPayloadを解釈し、正式なCommandの条件と操作時の認可を確認する。SUIはまだTEI正式Applicationではないため、**試験内限定のCommand候補**とする。
4. 既存のHTTPS委譲保存アダプターが、Hostで検証された主体・テナントと、サーバー側で取得した委譲資格情報を照合した後、固定したTLS受信先へ送る。
5. 模擬のSUI受信先が、対象文書・Revisionを照合して結果を返す。

[Go回帰試験](e23_selected_upstream_test.go)は、**共通受信口・構成選択・TLS送信の実コードを通す**。TLS受信先と主体確認・資格情報は、すべて試験内の合成部品である。SUIの実Python受信処理、DB、ブラウザーは起動しない。

## 拒否・保留を確かめる条件

- 実Hostを通さず選択提供者だけを直接呼んでも拒否する。
- セッションの失効、偽造要求対策の失敗、現在の操作権限の拒否、選択提供者の失効では、委譲先へ送らない。
- 認証済み主体と委譲元の主体が食い違う場合、固定TLS受信先へ送らない。
- ほかのApplication IDをAction本文に書いても、SUIの通信先や権限を借用させない。
- 受信先のRevision競合は409として扱い、成功した更新として返さない。
- 構成台帳から提供者が撤去された後は、新規呼出しを拒否する。
- 受信先で更新された可能性がある後に応答が失敗したときは、結果を不明のまま保持し、**自動再送しない**。再実行ではなく、SUI自身の権限付き再取得による照合が必要。

## 未実施・実装に追加しないもの

| 区分 | 未確認 |
| --- | --- |
| SUIの実利用 | React Canvasのポインター操作、確定Action、Undo・履歴・dirtyの更新、実UI試験 |
| SUI受信口 | 実サーバーの認証、Tenant・Audience・Document ACL、SQLite／PostgreSQLの条件付き保存、ETag再取得 |
| 資格情報 | 実セッション、資格情報提供元、Tokenの発行元・署名・有効期限、権限失効との同時整合 |
| 構成採用 | 実成果物のハッシュ・署名、Framework／Target／UI Kit、第二Consumerの実構成費用の計測 |
| 意味の完成 | 型付き参照の未解決項目の解消、構成Profileの正式採用、業務意味の受理と本番実行権限 |

新しいCore構成表現、Resolver、Graph、保存方式、資格情報発行器は追加しない。E23は**既存実装間の接続失敗を検出する試験**であり、これを理由に本番向け機能を有効化しない。

実行は`go test -count=1 ./reference/application-composition -run '^TestE23' -v`。CIの成否は、登録時の想定ではなく、対象PR headの実行結果で記録する。
