# E13：同じ解析済みWorldからE5と既存Derived Indexを取り出す

- **状態**：Goの読み取り専用Reference実装・テスト候補。実WorkspaceのGoテストと全Repository Healthの合格は**未取得**。正式Composition Profile、modelrefの統合、解析bytesの一貫性証明、Runtime Authorityではない。
- **管理**：[Issue #1016](https://github.com/hat47x/tei-platform/issues/1016)、[Draft PR #1051](https://github.com/hat47x/tei-platform/pull/1051)。
- **横断参照**：[#1120 / PR #1139](https://github.com/hat47x/tei-platform/pull/1139)が型付きGraph・Impactを所有し、[#1017 / PR #1111](https://github.com/hat47x/tei-platform/pull/1111)がLoader解析時bytesとSource Indexを扱う。本件では両者のコードをコピーせず、まだDraftの#1120 Packageをimportしない。

## 埋めた隙間

E5では`workspace.Load(root)`の`loaded.World`からApplication直接依存を列挙し、`workspace.BuildSourceIndex(root,loaded)`からその定義位置を取得していた。一方、#1120の`derivedbridge.FromIndex`が消費する**既存の`internal/index.Build(World)`出力**は、以前は別のGo呼出しに任せる想定だった。これでは、異なる実行時点のWorldを手作業で組み合わせてもJSONの形だけは一致してしまう。

今回、E5に**明示的なオプトイン**`-include-direct-derived`を追加した。同じ一回の`workspace.Load`で得た**一つのインメモリ`loaded.World`**を、E5のApplication直接依存検査と、既存`index.Build(loaded.World)`の双方へ渡す。出力するのは`index.Derived`全体ではなく、#1120の既存Derived Bridgeが必要とする次の5種類だけである。

| キー | 既存の持ち主と内容 |
| --- | --- |
| `semanticElements` | `index.SemanticElementEntry` |
| `semanticDependencies` | `index.SemanticDependency` |
| `declaredDependencies` | `index.DeclaredDependency` |
| `applicationPatterns` | `index.ApplicationPatternIndexEntry` |
| `queries` | `index.QueryIndexEntry` |

これらの型と意味関係は**既存`index.Build`の正本そのもの**であり、Composition側で文字列再解析・第二Graph生成・Source参照の再計算はしない。Environment Binding、Permission Grant、Resource、Plugin／Bundle、Releaseの情報をこの追加Projectionへ複製しない。

## 実行方法（信頼できるローカルWorkspaceで行うこと）

```sh
# 元の呼出しとJSON形状を維持
go run ./reference/application-composition \
  -workspace reference/workspace \
  -application order_entry -paired-evidence > /tmp/e5-baseline.json

# 新しい明示オプトイン。作成したJSONにはWorkspace全体のSource所在等を含む
go run ./reference/application-composition \
  -workspace reference/workspace \
  -application order_entry -paired-evidence -include-direct-derived > /tmp/e5-derived.json

# Goのfocused Testと、既存index.Buildとの一致／実order_entry対応を検査
go test ./reference/application-composition
```

`-include-direct-derived`だけを指定して`-paired-evidence`を省略した場合は入力誤りとして停止する。通常の`-paired-evidence`のJSONには`derived`を追加しない。両フラグを指定し、**E5のApplication直接参照Reportが`observed`の場合のみ**、従来の`{format,qualification,report,index}`に`derived`フィールドを追加する。`unresolved`の場合は従来のIssue/Source Index診断を維持する一方、Derived情報を追加公開しない。既存E7はこの新しいフィールドから実行権限・Sourceの完全性・変更なしの判定を推論しない。

**このコマンド列は実行手順であり、今回実際に成功したという記録ではない。**

## テストとして追加したもの

[e13-direct-derived_test.go](e13-direct-derived_test.go)に、Goの5件の試験を追加した。

1. E13で抽出した5つの配列が`index.Build(World)`の同じフィールドと一致すること。
2. 既存E5 EnvelopeのデフォルトJSONに`derived`が無く、`report`／`index`が保存されること。
3. オプトイン後のJSONに5つの許可フィールドだけが入り、Environment Grant／Provider／Bundle等を追加しないこと。
4. **実`reference/workspace`の`order_entry`**を`workspace.Load`＋`BuildSourceIndex`で一回読み、E5のApplication Pattern所属集合と、同じ`loaded.World`から作ったDerived IndexのApplication Pattern集合が一致すること。
5. **SUI Stage 0の実Workspace**がSemanticの部分参照のままであり、`sui_reference_adoption_stage0`という架空のApplication Patternへ昇格していないこと。

**5件はGoテストとして記述したもので、PASSとは主張しない**。この環境では非公開Repositoryのtrusted full checkoutを取得できず、Goの対象Packageや`tools/repository-health-check.sh`をexact-headで実行していない。新規テストの個別Go構文はローカル`gofmt`で確認したが、既存E5全体のGo Buildが成立することを証明しない。

## 残る未解決

- **同じインメモリWorld ≠ Source Indexの解析bytes一致**：`workspace.Load`と`BuildSourceIndex`の実際のファイル読取りが完全に同じbytesへ固定されたかは、#1017/PR #1111およびtrusted checkout上の検査が担当する。Go側で一度のLoaderを使っただけで、TOCTOUや署名付き出典を証明しない。
- **同じE5 Envelope ≠ #1120の型付きImpactも同じWorld**：まだ`derivedbridge.FromIndex`と`modelref.Validate/Impact`を同じGo呼出しへ配線していない。E11/E12の`unverified_shared_snapshot`を解除しない。ProjectionのJSONが一致しても、別プロセスで組んだImpactが同じSource revisionを利用したとはいえない。
- **限定したSource出力 ≠ 機密性**：今回の`derived`は全`index.Derived`より狭いが、従来の`index`にはWorkspace内の広いSource pathが含まれ得る。機微のIDやSourceを外部AI・共有サービスへそのまま送ってよいわけではない。出典の限定公開はE7aを用い、実ACL・共有可否は別の所有機構へ委譲する。
- **SUIを正式な第二Applicationへ昇格しない**：現行SUI Stage 0にはSemantic Sourceの一部しか存在しない。E13によって正式なApplicationが生成・採用されたことにはならない。

### 次の受入条件

trusted full checkoutのexact candidate headにて`go test ./reference/application-composition`・Go vet・gofmt・実`order_entry` CLI出力を確認し、必要なら#1120の`derivedbridge.FromIndex`を**同じGoプロセスと同じ`loaded.World`**へ結ぶ。最終的には実E7→E11レビューと実SUI第二Consumerの契約・実行Evidenceが揃った場合だけ、E10のComposition Profile採否判断を行う。

`main`への統合は`AGENTS.md`のexact-head・pre/post clean・全Repository Health PASSとDraft解除を満たすまで禁止する。
