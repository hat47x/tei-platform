# E5：解析済みApplicationの参照をSource Indexへ結ぶ

- **状態**：読み取り専用Reference。正式なImpact Resolver、Canonical Acceptance、Plugin Lock、Runtime Authorityではない。
- **管理**：[Issue #1016](https://github.com/hat47x/tei-platform/issues/1016)／[Draft PR #1051](https://github.com/hat47x/tei-platform/pull/1051)。
- **前段**：[E4](E4_SOURCE_INDEX_JOIN.md)では呼び出し側が対象Subjectを手で列挙するため、依存辺の漏れを検出できなかった。
- **関連**：[PR #1040](https://github.com/hat47x/tei-platform/pull/1040)（ValidationとBusiness Procedureの差）、[PR #1111](https://github.com/hat47x/tei-platform/pull/1111)（Loader解析時bytesとSource Indexの整合）。いずれの担当コードも変更しない。

## 今回の差分

[Go検証器](e5-world-index.go)は、既存`workspace.Load`が構築した`ir.World`と`workspace.BuildSourceIndex`の結果を読み取り、Applicationに既に記述された次の**明示参照**だけをつなぐ。

| 解析済みの参照元 | Source Indexの対象Subject | 判定するもの |
| --- | --- | --- |
| `Application.ID` | `application:<ID>` | Application自体の唯一のSource定義 |
| `Application.SemanticUses[]` | `semantic:<ID>` | 利用する名前空間が一意で、宣言上の定義参照があること（名前空間内の全定義の出典を表すわけではない） |
| `Application.Patterns[].ID` | `application-pattern:<ID>` | Applicationが所有するPatternの重複を拒否し、定義Sourceを一意に特定すること |
| `Application.Validations[]` | `semantic-validation:<ID>` | 意味定義が一意で、Source上のdefinitionが一意であること |
| `Application.Databases[].TargetDatabase` | `database:<ID>` | 論理Database宣言が一意で、Source上のdefinitionが一意であること |
| `Application.Databases[].RequiredContracts[]` | `database-contract:<ID>` | 選んだDatabaseに当該Contract定義が一意に存在し、Source上のdefinitionを特定できること |

Source Index内の`RoleDefinition`だけを定義出典として扱い、`authoring`／`implementation`／`derived`を代替しない。対応ArtifactのPathと`sha256:<hex>`を採る。Source Indexの出典情報を新しい永続ストアに移さず、その場で結果を返す。

**この検証器自身は新しいDSLを解析しない。** SemanticやContractの参照を文字列検索で補わず、既存Loaderが返した型付きデータだけを使う。Plugin・Runtimeに新しい解決機構を作らない。

### 対象と結果の意味

現行`order_entry`は3件のSemantic Namespace、5件のApplication Pattern、3件のValidation、2件のDatabase、3件のDatabase Contractを参照する。Application本体を含め**17個の定義Source照合**を行う構造である。これは定義上の見込み件数であって、実WorkspaceでのE5実行結果ではない。名前空間の出典はLoaderが選んだ代表の定義参照に限られ、名前空間に属するすべてのデータ・属性・規則の変更をその1参照で捕捉できるわけではない。

- `observed`：対象Applicationと上記の明示参照について、解析済み宣言およびIndexの定義参照が一意に揃ったことだけを表す。
- `unresolved`：Application欠落／重複、Validation定義欠落／重複、DB定義の曖昧さ、Contract取り違え／重複、Source定義の欠落／重複、宣言メジャー不一致など。入力の解析漏れや未解決を成功としない。
- `qualification: declaration-and-source-index-only`：`observed`でも受理済み業務意味、実行権限、実際のProvider互換、Migration、UIの接続を保証しない。

Versionのメジャー照合は明白な不一致の検出のみであり、実際のContract Conformanceや実装のバージョン選択は既存検証とLockへ委譲する。`SemanticUses`とPattern自身の定義出典は追加したが、**Patternが参照するData・Relation・Query・Capability等の推移的な依存**、外部認証、Environment接続、SUI Native依存を自動的に完全列挙したわけではないため、**全依存Closureの成立を主張しない**。

### 同一Databaseへの複数aliasとSourceの一意性

E5の初期実装では、`target database + contract ID` が二度現れると、**異なる正当なdatasource aliasからの利用まで重複エラーとして扱っていた**。これは参照元のaliasとSource Index Subjectを混同した判定だった。

修正後は、**一つのalias内の同じContract要求の重複だけを拒否**し、別aliasから同じDatabase/Contractを選ぶことは許す。必要Contractの存在とCompatibleMajorは**aliasごとに再評価**する。一方、E6へ渡すDatabase/Contractの`definition`出典は同じSubjectなら**一回だけ列挙**する。異なるaliasの一方にメジャー不一致があれば`unresolved`となり、適合するもう一方に隠されない。

追加の回帰試験は3件で、合計26件を記述した。これはGoの既存`DatabaseByAliasExact`が許す「異なるaliasで同一Targetを参照する」設計を構成診断側で勝手に禁止しないための修正である。Goテストは依然として未実行。

## 実行手順と検証Gate

```sh
go test ./reference/application-composition
go run ./reference/application-composition/e5-world-index.go -workspace reference/workspace -application order_entry
```

- [26件のGo回帰テスト](e5-world-index_test.go)を追加した。正常例のほか、欠落・重複・誤ったDatabase Contract、異なるメジャー、同じContract IDの別Databaseからの重複、Semantic Namespaceの欠落・重複参照、Pattern定義Sourceの欠落・重複、Authoring出典の不適切な代用、対象外Source変更などを含む。
- 2026-10-10の時点で**GitHubへGoコードとテストを登録したことは確認したが、このheadのGoテスト、Go fmt、go vet、実WorkspaceでのCLI実行は未実施**。成功件数として数えない。
- `tools/repository-health-check.sh`も未実施。trusted full checkoutのexact-head、実行前後clean、最新mainとの調整が済むまで[PR #1051](https://github.com/hat47x/tei-platform/pull/1051)はDraftのまま、mainへマージしない。

## E7用のオプトイン出力

従来のデフォルトJSON出力は維持する。E6へ渡す診断とSource Indexを同一呼び出しで採取したい場合だけ`-paired-evidence`を指定する。

```sh
go run ./reference/application-composition/e5-world-index.go \
  -workspace reference/workspace -application order_entry -paired-evidence > current.json
```

これにより`format`・`report`・`index`・限定的`qualification`を持つJSONを返す。詳しくは[E7の接続手順](E7_PAIRED_OBSERVATION.md)。Source IndexにはWorkspace相対パスが含まれる。信頼できる解析済みbytesの同一性や改ざん防止、実権限をこのEnvelopeで保証するわけではない。Goテスト2件を追加したが、このheadでのGoテスト実行は未確認。

## E4から確定できる改善と今後の境界

E4はConsumer Subject集合を呼び出し側が作らなければならなかった。E5では、少なくともApplicationが要求するNamespace／Pattern／Validation／Database Contractについて、その直接参照集合を**既存の解析済み`ir.World`から構成**できる。人手の依存一覧をApplication Composition Profileの第二正本にしなくてよい。

一方、同じValidationファイルに複数Validationが定義されている場合、Source IndexのFingerprintは**ファイル単位**である。無関係なValidation変更も同じファイルにあれば保守的に再検証が必要となる。フィールド／宣言単位の意味差を根拠に絞るには、Source Index Rangeや既存Semantic検証結果を利用する別の限定検査が必要。勝手に「互換」と分類しない。

次の判断は、(1)このGo検証器の実Workspace試験と全体Health、(2)E4のSubject集合への受け渡しを追加した際の影響削減、(3)SUI型第二Consumerに固有のSource所有者とNative Action境界、の順とする。必要性が確定するまで正式なComposition Profile Schemaや独立Resolverは導入しない。
