# TEI read-onlyモデル調査：CLI手順と主張範囲

この資料は`skills/yi-tei-modeling/SKILL.md`の詳細参照。コマンド例は**2026-11-02のmainソース確認に基づく探索手順**。その後、[実CLI CI #41532988114](https://github.com/hat47x/tei-platform/actions/runs/41532988114)で公開Workspaceを同一ビルド済み実行物から読み取る限定S1試験が成立したが、**外部HarnessがこのSkillを実ロード／実実行した証拠ではない**。通常のCLI仕様・現在のコマンド・版識別は[TEIを動かす](../../../docs/getting-started.md)と`cmd/tei/cli_contract.go`を先に確認する。

## read-only縦切りの例

以下のコマンドはTEIリポジトリのルートで実行する想定。固定されていない環境なら利用者が指定したWorkspaceを使用し、`reference/workspace`を本番正本と推定しない。

```sh
go version
git rev-parse HEAD
git status --short
go run ./cmd/tei commands
go run ./cmd/tei model catalog -workspace reference/workspace
go run ./cmd/tei inspect -workspace reference/workspace -subject-kind semantic-data -subject-id sales.order
go run ./cmd/tei trace -workspace reference/workspace -scope reference-observation -subject-kind semantic-relation -subject-id sales.order.customer -direction both -mode summary
go run ./cmd/tei validate -workspace reference/workspace -json
```

- `commands`は`schemaVersion=tei.cli.discovery/v0`で、`scope=selected-read-only-commands`の候補。すべてのCLI/Plugin能力一覧ではない。
- `model catalog`の各`subject.kind/id`は、続く`inspect`で用いる識別子。上記`sales.order`は**例**であり、実際には選択Workspaceから読み取ったIDを使う。ambiguousな対象のどれかを自動選択しない。
- `trace -scope`は観測用の文字列。実行時の人・組織・テナント認可を示すものではない。結果が直接参照に限定されていることを保持し、全依存関係を確認したと宣言しない。
- `validate`は終了コード0にWarningを含む。JSONの`ok`・`diagnostics`・`error`を合わせて判定し、warningの有無と成否を別に記録する。実行系の失敗や一部コマンドのJSONエラー形式はバージョン依存なので、すべての出力が一律に包まれると仮定しない。
- `worldFingerprint`／`sourceIndexFingerprint`は読取り時点の版同一性の手掛かり。実ContentやSource位置の違い、`modelRevision`の意味を区別し、同一ハッシュを現在の許可・人の意味受理へ読み替えない。
- `go run`ではバイナリにVCS情報が埋め込まれないことがある。ビルド済み実行ファイルの`version -json`、Git HEAD、Working Tree変更、Workspaceの入力バイトを用途に応じて照合する。


**実Harness比較への注意**：上記の`go run`は探索用のコマンド例であり、各行で同じ実行物を使ったことの証明ではない。H2 Tool-only／H3 Skill付きの比較では、[TAI SK-02初回Runbook](https://github.com/hat47x/tai-pleroma/blob/docs/yi-harness-skills-20261102/docs/yi-constellation/first-real-skill-qualification-runbook-2026-11-02.md)に従い、**比較前に実バイナリを一度だけビルド・固定**し、実呼出しのbinary SHA／cwd／argv／Workspace／SourceRevision／exit／diagnosticsを照合する。異なる実行物でも論理的に同じSource FPが出た、というだけで同一実験腕と判定しない。

## 呼出し時の実物・複数Readの接合（TS-11〜TS-15）

**発見したCLIと実際に呼んだCLIを分ける**。最初の`commands`が取得した情報を、後で呼び出した別ビルドや別Working Treeへ無条件に適用しない。再現性を高める場合は、一度ビルドして実行ファイルを固定し、実際に使った絶対パス、artifact bytesのdigest、ビルド元Git revision・Working Tree、呼び出したargv・cwd、Workspace rootと各Readのexit/statusを紐付ける。`go run ./cmd/tei`を繰り返す場合は、各呼出しでGoがソースを読み直し得るため、**すべて同じ実行物を使ったと主張できるわけではない**。観測していないdigestやパスを埋めない。固定しても実Source/Workspaceを同時ロックしたことにはならず、各Readの内容照合は別に必要。

**Traceの二つのfingerprintを取り違えない**。 `trace -mode full -limit <N>`の複数ページを読む場合、次ページの`-offset`には前ページの`result.trace.nextOffset`、`-expected-fingerprint`には**外側**`result.snapshotFingerprint`を使う。内側の`result.trace.graphFingerprint`（旧名`result.trace.snapshotFingerprint`。同じ値で残る）はTrace Graph自体を識別する別の値であり、継続トークンの代用にしない。`result.trace.continuation.fingerprint`は継続に使える値（外側`result.snapshotFingerprint`と同値）で、`nextOffset`の隣にある。誤トークンやSource変更により`STALE_SNAPSHOT`となった場合はページを結合しない。次ページの成功と、最初の対象/Scope/版が整合したことをそれぞれ検査する。

**同一ClaimへのRead接合**：`catalog → inspect → trace → validate`の各結果について、実root/Subject/Scope、観測時点、`worldFingerprint`/`sourceIndexFingerprint`の名前空間と値を比較し、差異を保持する。別checkoutでCRLF/LFが異なる場合、同じ論理モデルに見えてもbyte識別値が違うことがある。正規化や一致推定はしない（既存#2399のOwner判断を参照）。異版でも用途によって局所比較できる場合は、何を比較できて何を断定できないかを明示する。

**開示は実行とは別**： `commands`に`knowledge-context`等のread-only入口が載っていても、内部で読めるSource/Metadataと外部Harnessへ送ってよいProjectionは別である。機密性が要求される場合はProduct/Host/OSの実効境界を確認し、Skillに書いたread-onlyや文字列の`-scope`を外部開示資格と同一視しない。

この節のTS-11〜15は[TEI #2605](https://github.com/hat47x/tei-platform/issues/2605)の反証票。**実CLIのTOCTOU/ページング/異改行環境/実OS開示試験はNOT RUN**。

## 指摘が必要な反例

| 観測 | 解釈・次の確認 |
| --- | --- |
| `commands`がToolを列挙しない | この限定Catalogには載っていない。別の実契約・版を調査し、直ちに存在/不存在を断定しない |
| `model catalog`で同名・ambiguous | 正本Subjectの特定が未完了。Source/Namespaceを調べる |
| `inspect`と`trace`のFingerprintが異なる | 異版/異入力の可能性を報告し、同一版として結合しない |
| Warning付き`validate ok=true` | ValidationはWarning付きで成立。Warningを結果から消さない |
| `trace`に直接辺がない | 宣言済みの対応参照がない/未解決など結果の範囲を調べ、全依存不存在の証明にしない |
| `trace`が`UNSUPPORTED_SUBJECT_KIND`（終了コード2）を返す | その種類は関係図に投影されない（`semantic-candidate-key`・`semantic-functional-dependency`）。`SUBJECT_NOT_FOUND`（同1）の「宣言が無い」と読み替えず、`inspect`で要素自体を読めるか別に確かめる |
| 適用したいFunction Patternが説明されているが実Targetにない | 文書上の提案・参照例と、実Provider・Renderer・Hostの適用資格を別に確認 |
| 公開不可の原資料がある | Source保持側で許可されたProjectionを構成し、外部HarnessへRaw/非公開Metadataをそのまま渡さない |
| 外部Java/.NETアプリケーションでTEI Providerを使いたい | TEI Skillでは開発手順の案内まで。#2390/#2513等で実HostへのContract/Artifact/Loadを別に検証 |

## このSkillで直接扱わない操作

業務意味の正式受理、提案のSource反映、Plugin Providerの自動選択、DB書込み、Targetの本番配備、Java/.NET/Wasm Libraryの生成済み保証、SHO文書の実保存・原資料再解決、外部HarnessでのPlugin配布は別Ownerの領域。要求された場合は、対象Source・Claim・現行版・Owner/対応Issueと未実施事項を明確に引き渡す。Skillの存在は実操作の許可や完了の証拠にならない。

初回受入はTEI #2605のTS-01〜TS-10。TAIの複数HarnessでのSkill効果比較はTAI #168／Draft #270のSK-01へ結果だけ接続する。別の判定器・別のCapability正本は追加しない。
