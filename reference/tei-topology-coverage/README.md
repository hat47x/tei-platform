# 実CLI direct traceの同一性境界（read-only Reference）

**目的**：旧[Draft #2881](https://github.com/hat47x/tei-platform/pull/2881)で反証済みの、実`tei trace`→別言語Consumerの限定的な同一性検査だけを、2026-11-08時点のmainへ分離移植。旧Draftの「合成Graph/Tableの完全性」31ケースは**このPRへ移植しない**。新しいTEI Core/関係正本/Source Reader/Policy Engine/Hostを設けない。

## 何を照合するか

既存 `cmd/tei trace` の `tei.cli.trace/v0` と、`operations.TraceWorkspace`のRead-only結果をNodeから検査する。

- **Trace selector**：scope付きSubject／方向／full 1ページ／row cardinality／解決・未解決を明示。0件・不完全ページからRelation不存在を結論しない。
- **Source navigation**：`source.key`が`trace.subject`に一致。Located時に非空のSource Index referencesが必要で、各参照の`subject.kind/id`も選択対象に一致することを検査する。Source Indexにはscopeがないため、Sourceから独立のscopeを推測しない。
- **Workspace identity**（現mainの#3181）：`workspace.manifest.path`と`sha256`、`formatVersion`、任意の`name`をLoad時の由来情報として要求。二つのCLI観測を結合する際は、Manifest/Name/FormatVersionとWorld・Source Index Fingerprintを**別々に**照合する。同じWorld/Source FPでもManifestが違えば`workspace_manifest_mismatch`に停止する。**Manifestの観測はSource完全性、現在のSourceの独立再読取、受理済みモデル、権限、Lockを意味しない**。
- **Scope由来**：既存`derivedbridge.FromIndex(d,scope)`のscopeはCaller提供。実Workspaceを異なる2つの`-scope`で読み、選択子bound fingerprint・qualified Keyだけ変わることを確認する。Source自身の宣言scopeとは言わない。
- **直接Edgeの資格**：`Edge.Relation`は種別であり、独立した業務Relation ID／所属Scope／多項Roleではない。`relationEntityIdentity:not-provided`、`multiRoleClosure:unproven`、`absentClaim:unknown`、`authority:read-only-candidate`を維持。

## 再現

```sh
node --test reference/tei-topology-coverage/direct-trace.test.mjs
go run ./cmd/tei trace -workspace reference/workspace -scope reference-observation \
  -subject-kind semantic-relation -subject-id sales.order.customer \
  -direction both -mode full > /tmp/tei-trace-a.json
go run ./cmd/tei trace -workspace reference/workspace -scope alternate-observation \
  -subject-kind semantic-relation -subject-id sales.order.customer \
  -direction both -mode full > /tmp/tei-trace-b.json
# Manifest名だけを変えた、業務Sourceが同じ別Workspaceを一時作成
mkdir -p /tmp/tei-topology-other-workspace
cp -a reference/workspace/. /tmp/tei-topology-other-workspace/
sed -i 's/^workspace: tei_reference$/workspace: tei_reference_copy/' \
  /tmp/tei-topology-other-workspace/tei.workspace.yaml
go run ./cmd/tei trace -workspace /tmp/tei-topology-other-workspace \
  -scope reference-observation -subject-kind semantic-relation \
  -subject-id sales.order.customer -direction both -mode full > /tmp/tei-trace-c.json
node reference/tei-topology-coverage/verify-cli-trace.mjs \
  /tmp/tei-trace-a.json /tmp/tei-trace-b.json /tmp/tei-trace-c.json
```

**注意：** 実CLIの生JSONにはSource pathやEvidenceを含む可能性がある。公開資格を持たない外部Consumerへ渡さず、CI内の一時ファイルだけで検査する。Node Referenceは実Disclosure Policyや現在Grantを評価しない。CLIのSource Navigationを**同じ応答内で検査**し、二つのCLI応答のLoad-time Manifest／World／Source Indexの対応を比較するだけで、独立Reader／同一Attemptの別観測による真正性を証明しない。`workspaceLoad`は内部Reference用の比較メタデータであり、そのままPublic Consumerへ公開しない。

**検証していないこと：** #1124の実Workbench 4入口・旧DOM、#2516の実Policy/二世界、#267のOwner意味受理、#2620のRole全解析/必要Root、#2831のセルフビルド実工程、#2542の配送Receipt、[統合済み研究#2966](https://github.com/hat47x/tei-platform/pull/2966)の人間/AI認知負荷比較。旧#2881の合成Graph/Table結果と本PRの実CLI結果は別資格として保持。

**関連設計**：[トポロジー・複合図表](../../plan/design/TOPOLOGY_COMPOSITE_VIEWS_AND_PIPELINE_2026-10.md)。Intent／Realization／Assessmentを一つの「存在している」の判断へ混同しない。試験とソース更新を実際に確認してからmain採用を検討する。
