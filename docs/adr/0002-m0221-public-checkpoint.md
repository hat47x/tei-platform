# ADR-0002: M0.22.1を最初の公開設計チェックポイントとする

- 状態: 採用
- 日付: 2026-08-30
- 関連Issue: #2

## 背景

TEIは、完成したローコード製品を先に作るのではなく、後から壊れやすい設計境界を実行可能なコードで先に確かめてきました。

M0.22では、Plugin / Runtime境界にDurable Invocation Handleを導入し、Host takeover後に外部runtimeへ再接続する場合の扱いまで到達しました。その後の公開準備で、Lease Supervisorの時間基準がResource Brokerのテスト用clockと一致していないことが分かり、Hostからclockを注入できるように補正しました。このhardeningを含む状態をM0.22.1とします。

また、GitHubへ最初に配置した段階では、ソースを一時的な分割データとして搬送する必要がありました。公開チェックポイントとして残すには、その搬送方式ではなく、通常のGoソースツリーとしてclone・検証できる状態へ戻す必要がありました。

## 判断

M0.22.1を、TEIの最初の公開設計チェックポイントとして扱います。

`main` には、一時的なsource transportではなく、通常のソースツリーを保持します。M0.22.1では、少なくとも次の流れが一つのRepository内で接続されていることを確認対象とします。

- Semantic / Workspace
- Applicationと複数Databaseの関係
- Database Contract / Persistence Mapping / Schema Diff / Migration
- Database Release / Application Release / Solution Release
- Deployment Release Graph / Durable Run / Observation / Evidence
- Plugin / Bundle / Plugin Lock
- Permission / Resource Broker / Lease
- Supervisor ownership / fencing
- Durable Invocation Handle / takeover後のruntime reattach

M0.22.1は「production ready」の宣言ではありません。完成したEditor、本番用Runtime、分散永続Store、各種cloud adapterなどを提供する段階ではなく、設計境界を検証するための参照実装です。

## 検証条件

公開チェックポイントとして、cloneしたRepositoryから少なくとも次が成功する状態を保ちます。

```bash
go vet ./...
go test ./...
go run ./cmd/tei validate -workspace reference/workspace
go run ./examples/runtime-reattach
```

Reference Workspaceに含まれるhistorical contractについては、意図したWarningが出ても最終的に`ok`となることを正常とします。

## この切り目を採る理由

M0.22.1以前でも個々のモデルやテストは成立していましたが、runtime takeoverと再接続まで含めることで、TEIが単なる静的なメタモデルではなく、意味・release・実行・観測・回復を一続きに扱えるかを確かめられるところまで来ました。

一方で、ここから先はproduction adapterやAuthoring体験など、複数の方向へ進めます。次の実装を急いで混ぜる前に、一度この地点を再現可能な基準点として固定する価値があります。

## 影響

今後の変更は、M0.22.1から何を変えたのかを比較しやすくなります。また、別の生成AIへ監査や次段階の検討を依頼するときも、共通の出発点を示せます。

その代わり、M0.22.1に含まれる仕組みを完成仕様と誤解しないよう、READMEとstatusでは「設計チェックポイント」「参照実装」であることを明示し続けます。
