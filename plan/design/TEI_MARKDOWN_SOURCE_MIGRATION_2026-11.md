# TEI Sourceの `*.tei.md` 統一と既存 `*.tei` 移行の検収計画（2026-11-06）

> **方針の状態**：2026-11-06のOwner指示により、**TEI向け文書の標準物理拡張子は `*.tei.md`、新規 `*.tei` は原則停止・既存 `*.tei` は段階廃止**を移行方向として固定する。**実装の状態**は別：現mainのLoader/Parser、正式Markdown文法、Workspace/Target互換、旧Source自動移行、Owner Meaning Acceptanceはこの資料では変更・実証しない。旧2026-08-29 `tei-m0.21-naming.md`やTEI #2621の「`.tei.md`を標準化しない」旧検討案には**今回の決定を優先**する。既存の記録は歴史として保全する。

**責務**：Authoring/物理識別＝[#2621](https://github.com/hat47x/tei-platform/issues/2621)、LoaderとRead Coverage＝[#2823](https://github.com/hat47x/tei-platform/issues/2823)／#2571、Language/Operator Grammar＝#988／#2620、Format Migration＝#1920／#2505、Source/Build・再資格化＝#2595／#2600、独立Expected＝#2514、SHO Writer＝[SHO #19](https://github.com/hat47x/sho-composer/issues/19)、OKFの選択式拡張＝[SHO #18](https://github.com/hat47x/sho-composer/issues/18)／[TEI Draft #3029](https://github.com/hat47x/tei-platform/pull/3029)。新しいParser正本、Source Registry、Host、Acceptance、Scenario Runnerは作らない。

## 1. 決定事項と未決事項を分ける

- **決定**：TEI固有のSource意図をファイル外部からも識別でき、汎用Markdown Editor/検索・文書/図表・自由文章を基礎とする `*.tei.md` を標準とする。通常の `*.md` は通常のMarkdownとして保存でき、**`.md`を一律TEI Parserに渡さない**。既存TEI物理 `*.tei` は新たなAuthoringの標準形にしない。移行中の読取・履歴・旧成果物との互換は別に管理する。
- **未決**：Markdown中のどのRole/Frontmatter/Fence/表・宣言領域を正式DSLとして解釈するか、旧 `*.tei` 文法の原文をどの形式へ無損失で格納するか、`*.tei.md`で必要なRole指定、2-way/AST往復、未知Contentの扱い、Release段階でlegacy readerを廃止する時期。形式選定の第一基準は**ドメイン専門家の可読性と認知順**であり、Go ASTに都合のよい記法は内部実装の候補にとどめる。
- **OKFとの独立性**：`*.tei.md` はMarkdown系ファイルであるが、**拡張子だけでOKF v0.2 Concept適合ではない**。OKFが選択される場合に限り、Bundle/Frontmatter（特に非空の `type`）を当該仕様で判定する。`tags: [tei]`／独自 `tei` metadataは発見・Editor補完の意図であり、TEIの構文適合、Owner受理、実行・開示Grantではない。OKF conceptのpathから得るID（`.md`を外すと `foo.tei` 等）をTEI Semantic ID／旧 `foo.tei`のSource Identityと自動同一視しない。OKF非導入でもTEI文書は編集・保存可能にする。
- **Source正本**：元 `*.tei`と派生 `*.tei.md`を二重のAccepted Sourceとして黙って並行更新しない。移行票にAuthoritative revision/old locator/new locator/原bytes/意味ID・Role/Parser version・差分/Ownerを明記し、Raw Draft Save、Semantic Promote、Build/Consumer Cutoverを独立操作とする。
- **Manifestの例外処理**：現標準の`tei.workspace.yaml`は従来どおりmanifestの選択経路として扱い、拡張子一括改名の対象にしない。legacy `tei.workspace.tei`が残る場合、単純に `tei.workspace.tei.md`と改名するのでなく`tei.workspace.yaml`への明示変換・読込確認を検討。実際のWriter/Reader変更はOwnerが決める。

## 2. 2026-11-06時点のmain実コードとの差分（Source-static、実移行NOT RUN）

[現main `internal/workspace/loader.go`](https://github.com/hat47x/tei-platform/blob/main/internal/workspace/loader.go)を確認。Semantic Sourceは `strings.HasSuffix(name, ".tei")`、Pluginは `plugin.tei`、Bundleは `bundle.tei`、Application manifestは `application.tei`、Pattern/Package/Query/Realization/DB Contract/Persistenceは `*.tei` など、**Source-rootごとに異なるReader/filenameフィルター**を持つ。現mainは `*.tei.md`の全対象を自動的にParsed/Loadedとするコードではない。

`tei workspace-files`のmanaged列挙と`tei validate`の実Parsed/Requiredは別（[#2823](https://github.com/hat47x/tei-platform/issues/2823)）。`*.tei.md`を作成・改名しても`validate ok`や0件だけを見て移行完了とはしない。新規Sourceを除外するlegacy regexを残したまま「移行済み」と表示するのは**誤資格**。Source listing・Loaded/Parsed・Role interpreted・Semantic accepted・実Build consumed・Consumer actual readはそれぞれ別Witnessとする。

## 3. 段階的な移行とRelease互換

| 段階 | 作業とOwner | 次へ進むための実測・停止線 |
| --- | --- | --- |
| **M0 棚卸し** | #2823/#2621：main・既存Reference/Workspace・test/fixture・Plugin等の旧`*.tei`一覧、manifest roots、Source Kind、実Reader、Consumer、Source Span/Hash、OS/Linux/Windowsのcaseを収集 | `managed`≠`parsed`、root綴り誤り、未取得・0件・非対応を見落とさずRequired Countを先に決める |
| **M1 Grammar適合PoC** | #988/#2620/#2621：公開1 Subjectで (A)Markdown構造＋Role、(B)宣言領域/Fence、(C)旧文法原文保持を比較。正式SyntaxはOwner判定まで候補 | 原文/空白/コメント/BOM/CRLF/引用/未知属性のloss、source span・Error位置・同Role曖昧性を独立Expectedで確認。単純Renameが成功という前提は禁止 |
| **M2 Dual-readの限定導入** | #2823/#1920：既存legacy Readerを互換経路として保ち、新 `*.tei.md` のReaderを明示Profile/Source Kindに束縛。必須root／旧新衝突を診断 | **旧新同意味IDの二重定義を辞書順で選ばない**。Reader unavailable/unsupported/partialを`validate ok`で隠さない。自動Fallbackがあれば選択理由・既定とPinを明記 |
| **M3 移行・再資格化** | #2505/#2600/#2595/#2514：1 Workspace・1 Semantic Subjectの旧 Source→新 SourceでRaw File、Semantic World/Meaning、Model Revision、Artifact Revision、Source location/Concept/Review/Grantを対照 | 同内容でもSource path/index/引用は変わる。World/Meaningの同値は実検証対象で、A→B→AやOwner撤回で旧Acceptance/Artifact/Grantを自動復活させない |
| **M4 Authoring Cutover** | #2621/#1124/SHO #19：新規保存は`*.tei.md`、旧`*.tei`は可視のlegacy/read-onlyまたは明示移行操作。異なるOwner／Consumerの履歴参照を保全 | Raw Draft保存（未知Role・構文不正でも可能）と正式Meaning Promote/実行可を分離。旧Client／新Client／旧Source／新Sourceの方向付き互換を確認 |
| **M5 Legacy retire** | #1920/#2823/#1119：必要な実Reader/Builder/Target/Doc/CLI/Skill参照を移行し、互換readerを段階的に削除する適切な版境界を決定 | **残存必須Sourceが0**・Required/Observed/Undischarged／独立Oracle／実Consumer確認が揃った範囲のみ廃止。旧文書とhistoryは検索・監査可能に残す |

## 4. 固定してから実装する独立Expected（計画／NOT RUN）

| ID | 入力／変異 | 許容する結論・検出する誤り |
| --- | --- | --- |
| **TMD-01** | 同一公開Semantic Subjectの旧 `model.tei` と新 `model.tei.md` を別Checkoutに置く | 明示Reader選択後の意味同値だけを認定。ファイルRenameだけを解析成功・Owner再受理としない |
| **TMD-02** | `model.tei.md`のみが存在し、旧Readerが`.tei`のみを探索 | 必須Sourceが`not-loaded/unsupported`として分かる。空Workspace Validate successを「OK」にしない |
| **TMD-03** | 旧 `model.tei` と新 `model.tei.md` の同じSemantic IDが同時に存在 | 明示Migration Window/優先規約が無ければ競合・unknown。登録順のlast-wins禁止 |
| **TMD-04** | `application.tei`、`plugin.tei`、`bundle.tei`、`*.tei`各root別の新名 | 各Source KindごとにLoadedとParser実接続を判定。Semanticのみ対応で「全対応」としない |
| **TMD-05** | 通常 `notes.md`、`notes.tei.md`、`notes.md`＋OKF `tags: [tei]` | 通常MDは任意Authoring、`.tei.md`はTEI意図、tagsのみで実行/Owner受理を付与しない |
| **TMD-06** | OKF Concept `foo.tei.md`（Concept ID `foo.tei`）とlegacy `foo.tei` | Concept ID/Source Path/TEI Meaning IDは別。重複・誤参照を黙って解消しない |
| **TMD-07** | `tei.workspace.yaml`／legacy `tei.workspace.tei`と不正root名 | 正式manifestを失わず、unsupported/unknown source rootを実診断。manifest自体を通常TEI Sourceへ誤解釈しない |
| **TMD-08** | Source CRLF+BOM+日本語コメント＋表・コードFence・不正Role、Raw Draft Save→再Read | 元byte・Source span・診断を保持。Raw SaveをSemantic Promote/Meaning受理へ昇格させない |
| **TMD-09** | 元Sourceは同byteでpath移動、またはA→B→A＋Review撤回 | Meaningの必要再計算とSource citation/現在資格の失効を別判定。旧Review自動復活なし |
| **TMD-10** | 新LoaderだけCLI `validate` PASSだがBuilder/Consumerが旧Sourceを使用 | 実Build Input/Target Binary/必要Artifact memberが未束縛なら`unqualified`、同一改訂E2Eとはしない |
| **TMD-11** | Legacy Reader除去後、古いCLI／SHO文書／Skill参照が残存 | 旧新Client方向別の失敗／移行支援を示す。文書リンクの自動置換で原引用を破壊しない |
| **TMD-12** | 正しい一部`*.tei.md`と壊れた必須`*.tei.md`を同じWorkspaceに置く | 検査した部分のPASSは保持し、必要全集合の未検査／parse errorを`Required`充足にしない |

**初回の実行順**：TMD-01/02/03/04/07を同一固定Revisionのread-only fixture（既存`reference/workspace`をコピーし原本を変更しない）で先行。成功基準は`Source managed/discovered/loaded/parsed/actual consumed`とRequired集合の差分を明示すること。次にTMD-05/06/08でSHO/OKF Writer・原文、TMD-09/10/11/12でBuilder/Consumer資格を確認。独立Expectedは[#2514](https://github.com/hat47x/tei-platform/issues/2514)に事前固定し、Converterが生成したExpectedで自己採点しない。

## 5. 判断の境界・禁止事項

これは**新しい拡張子方針と移行受入の設計**であり、`*.tei.md`の現行製品読込成功、正式なMarkdown DSL文法決定、実Migration Tool、全Source一括Rename、Role/Pluginの自動採用、OKF必須化、Studio/SHO/TAI実利用、Provider/DB/Host実Effect、Owner業務意味受理を報告しない。旧`*.tei`資料・fixtureを手当たり次第書換えず、各Ownerで実Reader接続・独立Expected・Consumerが成立してから段階移行する。現行#1119 P0/G0〜G4、他Targetの資格を無断で書換えない。
