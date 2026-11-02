# TEI read-onlyモデル調査：CLI手順と主張範囲

この資料は`skills/yi-tei-modeling/SKILL.md`の詳細参照。**2026-11-02のmainソース確認に基づく設計例**であり、当該SkillによるCLI実行・受入の証拠ではない。通常のCLI仕様・現在のコマンド・版識別は[TEIを動かす](../../../docs/getting-started.md)と`cmd/tei/cli_contract.go`を先に確認する。

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

## 指摘が必要な反例

| 観測 | 解釈・次の確認 |
| --- | --- |
| `commands`がToolを列挙しない | この限定Catalogには載っていない。別の実契約・版を調査し、直ちに存在/不存在を断定しない |
| `model catalog`で同名・ambiguous | 正本Subjectの特定が未完了。Source/Namespaceを調べる |
| `inspect`と`trace`のFingerprintが異なる | 異版/異入力の可能性を報告し、同一版として結合しない |
| Warning付き`validate ok=true` | ValidationはWarning付きで成立。Warningを結果から消さない |
| `trace`に直接辺がない | 宣言済みの対応参照がない/未解決など結果の範囲を調べ、全依存不存在の証明にしない |
| 適用したいFunction Patternが説明されているが実Targetにない | 文書上の提案・参照例と、実Provider・Renderer・Hostの適用資格を別に確認 |
| 公開不可の原資料がある | Source保持側で許可されたProjectionを構成し、外部HarnessへRaw/非公開Metadataをそのまま渡さない |
| 外部Java/.NETアプリケーションでTEI Providerを使いたい | TEI Skillでは開発手順の案内まで。#2390/#2513等で実HostへのContract/Artifact/Loadを別に検証 |

## このSkillで直接扱わない操作

業務意味の正式受理、提案のSource反映、Plugin Providerの自動選択、DB書込み、Targetの本番配備、Java/.NET/Wasm Libraryの生成済み保証、SHO文書の実保存・原資料再解決、外部HarnessでのPlugin配布は別Ownerの領域。要求された場合は、対象Source・Claim・現行版・Owner/対応Issueと未実施事項を明確に引き渡す。Skillの存在は実操作の許可や完了の証拠にならない。

初回受入はTEI #2605のTS-01〜TS-10。TAIの複数HarnessでのSkill効果比較はTAI #168／Draft #270のSK-01へ結果だけ接続する。別の判定器・別のCapability正本は追加しない。
