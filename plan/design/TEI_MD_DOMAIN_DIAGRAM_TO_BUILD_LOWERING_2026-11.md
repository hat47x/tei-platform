# *.tei.md ドメイン知識の図表表現からコード生成・ビルドへ到る縦断設計（設計候補）

- **状態**: 設計候補。本書は `*.tei.md` の図表・表形式のドメイン知識をコードへ展開しビルドする縦断の設計・受入計画である。実装・実証・Owner意味受理は未実施。既存Gate B/C、`reference/workspace` の固定版、`.tei/decisions/` を遡及変更しない。
- **作成日**: 2026-11-11
- **関連PR**: [#4293](https://github.com/hat47x/tei-platform/pull/4293)（`tools/reference-application-binary` に database migration parsing / generation / startup integration を追加）
- **既存計画との所有境界**: `*.tei.md` の形式統一・移行・Reader/Parser の主担当は [TEI_MD_UNIFIED_AUTHORING_MIGRATION_2026-11.md](TEI_MD_UNIFIED_AUTHORING_MIGRATION_2026-11.md)（[Draft #3030](https://github.com/hat47x/tei-platform/pull/3030)）および #2621/#988/#2823/#2822。図表の投影・ナビゲーションの製品標準要件は [MASTER_PLAN.md](../MASTER_PLAN.md) §RQ-036／[三要素Method §19.8](THREE_ELEMENT_METHOD_DIAGRAM_CONTRACT_2026-10.md)。本書は**この2つの間に立つ「Parse → Model → Generate → Build」の縦断接続**を所有し、第二のParser・Source Registry・Acceptance・Semantic SSOT を作らない。

## 1. 目的と位置づけ

[RQ-036](../MASTER_PLAN.md) は「TEIのドメイン全容を図表とナビゲーションで理解・拡張できること」を製品標準要件としている。本書は、その図表・表形式のドメイン知識が**コードとして展開され、ビルドされる**ところまでの縦断を、`tools/reference-application-binary` で得られた知見を使って設計する。

**本書が示す最小命題**: `*.tei.md` に記述されたドメイン知識（図表・表・SQL・制約）は、固定された独立Expectedに対して、Parse → Model 抽出 → コード生成 → ビルド → 実行 の各段階を追跡可能に経由して、実行可能な単一バイナリへ展開される。各段階の成否は前の段階の成功から自動継承しない。

## 2. #4293 で確認できた知見（設計への入力）

以下は #4293 で実装・検証された範囲である。製品全体の `*.tei.md` Reader/Build 資格は**まだ成立しない**。

| # | 知見 | 本書での扱い |
| --- | --- | --- |
| K-01 | `teiparse.World` の `Databases`/`Migrations` と `parseDatabaseDir`/`parseDatabaseFile`/`parseMigrationFile` により、`databases/` 配下の `.tei` からマイグレーションSQLを抽出できる | `*.tei.md` 内の図表・表から同等の意味を抽出するReader設計の参照実装とする。ただし現行Parserは行指向の暫定実装であり、正式な `*.tei.md` Reader の代替ではない |
| K-02 | `appgen.GenerationContext` に `Databases`/`Migrations` を持たせ、`ExtractModel` がパース結果をテンプレートへ渡す | Model抽出層を「図表のSubject → 生成コンテキスト」の対応規則として設計する。Subject/Role/Occurrence の追跡は [三要素Method §2](THREE_ELEMENT_METHOD_DIAGRAM_CONTRACT_2026-10.md) の ProjectionIdentity/Evidence に接続する |
| K-03 | `migrations.go.tmpl` が `toGoIdentifier`/`quoteSQL` を用いて raw string literal のSQLと `applyMigrations` を生成する | 図表→コードのテンプレート層は「表記で新しい意味を創作しない」制約を満えるか検証する。生成物のprovenance（`schemaVersion`/`modelRevision`/`toolchain`/`buildArtifactDigest`）を正本とする |
| K-04 | `main.go.tmpl` の `db.Ping()` 後 `applyMigrations` を呼ぶ | 起動時にマイグレーションを適用する契約は、図表に記述された「状態遷移・イベント群」と対応するExecution順序の最小例として扱う。順序の保証は独立Expectedで検証する |
| K-05 | 生成物は `modernc.org/sqlite` を埋め込み、外部TEI runtime・動的Pluginなしで単一バイナリとしてビルド・実行できる | 「外部連携・Provider」を図表の辺として表現した場合、Target/Provider境界は [ADR-0043](../adr/0043-target-native-artifacts-without-tei-host.md) の「TEI独自Host原則非必須」を維持する |
| K-06 | Windows の `GOCACHE`/`GOTMPDIR` をソースツリー内ローカルに設定し、バイナリ出力先もソース配下にすることで、権限でbuildが失敗する問題を回避できた | Build環境の再現性は「環境差を独立Expectedに混ぜない」ための実運用知見として記録する。CIとローカルのToolchain版を別資格とする |
| K-07 | `reference/workspace` の migration SQL は PostgreSQL 方言（`public.orders` 等）であり、SQLiteでは実行時に構文エラーとなる | 方言差は「同じ図表が複数Targetで意味保存されるか」の反例候補として管理する。SQLite PASSはPG/Host/Owner受理を証明しない |
| K-08 | `TestGeneratedBinaryMigrationsApplied`・`TestParseWorldReferenceWorkspace` でParse/Generate/Buildの各段階を検証した | 受入は「Parse PASS ≠ Generate PASS ≠ Build PASS ≠ 実行PASS」を段階別に分離して行う。K-07のため実行テストは方言非互換時にskipする |

## 3. 理想形の定義（到達目標）

**目標状態**: 利用者が `*.tei.md` に業務の対象構造・イベント群・状態遷移・Function・Data・Rule・外部連携を**図表と表**で記述すると、TEIはそれを一貫した図表体系の中に位置づけ、**同じ意味正本からコードを生成し、ビルドされた単一バイナリがその通りに動作する**。利用者は「全体として何をするか・何が起きるか・どこを変えられるか」を、実装以前の設計書として、かつ実装・試験の所在へ辿れるガイドとして理解できる。

**最低不変条件（本書で固定する保証）**:

1. **一貫した図表体系内の位置づけ**: 各Patternは個別説明ではなく、全体のFlow・状態遷移・構成・Data Relationの中の現在地として表示・ナビゲーションできる。詳細を開いても概観へ戻れる。一枚で読めない図は意味上の区画で分割する。
2. **標準固定とCustomの区別**: Flow・Graph・Matrix・CLI説明で、標準固定の意味・Guard・Authority/Effect境界、設定可能な既定値、明示Extension Slot/Hook、案件で置換可能な部分、観測のみの部分を同じように区別する。差し込み個所は**標準フロー／差分／合成後の実効フロー**を相互参照可能にする。
3. **完全性の明示**: 指定したWorld／Source Scope／Consumer公開資格に対し、要求される図表・関係・Pattern/Event/Stateの対象集合と、実際に解決・表示した集合、未解決・非公開・未対応の集合を区別して表示する。必要集合自体が未確認なら「全体像はpartial」と説明し、表示したものだけで完全性を主張しない。
4. **共通Accepted Meaning/Sourceからの導出**: 図表・表・CLIは共通のAccepted Meaning/Sourceから導出する。手書きのガイド図やAI説明を第二正本にしない。
5. **段階資格の分離**: 「Source発見 → Loader実読込 → Parse/Role occurrence → Required依存照合 → Owner受理 → Build → Consumer実読取 → Host Grant/Effect」を独立の段階として扱う。前段の成功を次段の合格に自動昇格させない。

## 4. 縦断設計：Parse → Model → Generate → Build → Execute

### 4.1 全体フローと追跡

```
*.tei.md（説明・図表・表・SQL・制約）
  │  (A) Source発見・Selection固定
  ├─> Managed Source / Source Index / Revision
  │  (B) Markdown Reader / Role解釈（#3030 M2/M3、#988）
  ├─> Parsed Occurrence / Source Span
  │  (C) Model抽出（本書: 図表Subject → GenerationContext）
  ├─> GenerationContext（DataModels / Queries / Commands / Validations / Databases / Migrations / ...）
  │  (D) Template投影（本書: 表記で新しい意味を創作しない）
  ├─> Generated Go source（provenance埋め込み）
  │  (E) Build（Toolchain / GOCACHE / Target）
  ├─> 単一バイナリ（外部runtime・動的Pluginなし）
  │  (F) Execute / Consumer read
  └─> 独立Expected照合 → Claim判定（holds / fails / unverified）
```

各辺(A)〜(F)は**独立の受入対象**である。#4293で確認したK-01〜K-08は、(C)(D)(E)(F)の**旧 `.tei` 限定**の参照実装であり、(B)の `*.tei.md` Reader は未接続。

### 4.2 Model抽出層（C）の設計候補

図表のSubjectから生成コンテキストへの対応規則を定義する。#4293の `extractDataModels`/`extractQueries`/`extractCommands`/`extractValidations` は旧 `.tei` の行指向Parseだが、対応規則の骨格は流用できる。

| 図表の表現 | 生成コンテキストへの対応候補 | 追跡事項 |
| --- | --- | --- |
| ER図のEntity | `DataModel`（`GoTypeName`/`Fields`） | Semantic Subject ID、Source occurrence、Attribute→Fieldの型写像 |
| ER図のRelation | `DataField`（外部キー相当）＋Relation Storage | 論理キーと物理FKの別扱い、多重度 |
| DFDのProcess/Action | `Command`（`GoFunctionName`/`InputType`/`OutputType`） | Function-level C/R/U/DとAction-specific effectの区別 |
| 状態遷移図のState/Event | 実行順序・Guard（`applyMigrations` 相当の起動契約） | 遷移順序、不変条件、欠落遷移 |
| 業務ルール表 | `Validation`（`Rules`） | Rule closure、NULL/unknown、独立Expected |
| クエリ表/SQL | `Query`（`SQL`） | 方言差、Parameter、Provider Scope |
| DB定義・マイグレーション表 | `Database`/`Migration`（`SchemaSQL`/`DataSQL`/`VerifySQL`） | Target方言、recovery、検証SQL |

**写像の制約**: 同じSubjectの複数Occurrenceを許す。表示名や観測tokenだけでacceptedとしない。Evidence（accepted / proposal / source-observation / derived）を保持する。

### 4.3 Template投影層（D）の設計候補

生成テンプレートは「表記で新しい意味を創作しない」。#4293の `migrations.go.tmpl` は `toGoIdentifier`/`quoteSQL` で識別子とSQL literalを安全に変換し、provenance（`GeneratorID`/`GeneratorVersion`/`SourceRevision`/`ModelRevision`/`Toolchain`/`BuildArtifactDigest`/`BuiltAt`）を定数として埋め込む。

| 生成対象 | 図表からの入力 | 不変条件 |
| --- | --- | --- |
| `models.go` | DataModel/Field | 型写Mappingは明示した規則のみ。既定推測で型を決めない |
| `queries.go` | Query SQL/Parameter | SQLはraw literalで保持。方言はTarget別に解決 |
| `commands.go` | Command/Pattern | 入力JSONの型契約、欠落Fieldの扱い |
| `validations.go` | Validation/Rule | Rule closure、誤引用検出 |
| `migrations.go` | Migration SQL | 起動順序、recovery方言、検証SQL |
| `provenance.go` | 生成メタデータ | artifact digest、toolchain、revision |

### 4.4 Build層（E）とExecute層（F）

#4293で確認したとおり、`GOCACHE`/`GOTMPDIR` のローカル化、バイナリ出力先のソース配下化、`-trimpath -buildvcs=false -ldflags=-s -w` により、外部runtimeなしの単一バイナリが得られる。

**検証の分離**: 生成物のcompile PASS、バイナリの起動PASS、独立Expectedに対する実行PASSは別Claimとする。K-07のとおり、SQLite実行PASSはPostgreSQL実Provider・Host Grant・Owner受理を証明しない。

## 5. 受入計画（段階別・独立Expected）

本書の受入は「前段のPASSを次段へ継承しない」ことを原則とする。各段階は同じ固定Source／独立Expectedで照合する。

### P0-A（本書の最初の検証）: 旧 `.tei` 縦断の再現

- **目的**: #4293の参照実装を、独立Expectedを持つ公開合成Scenarioで再現する。
- **対象**: `reference/workspace` の `databases/sales-core/migrations/20260826_001/migration.tei` と `applications/order-entry/application.tei`。
- **手順**: (A) Source固定 → (C) Model抽出（`Databases`/`Migrations` の件数・内容）→ (D) `migrations.go` 生成（`applyMigrations`/migration ID/SQL）→ (E) Build → (F) provenance/query実行。
- **独立Expected**: migration ID `20260826_001`、`SchemaSQL`/`DataSQL`/`VerifySQL` の内容、生成ファイル名集合、provenance fields。
- **合格条件**: 各段階の実測が独立Expectedと一致。方言非互換の実行は `unverified` として明示（K-07）。
- **未達成の扱い**: 新 `*.tei.md` Reader への自動継承はしない。

### P1-B（図表の豊富化）: `*.tei.md` サンプルの設計

- **目的**: ドメイン知識の図表表現を `*.tei.md` に持ち、Markdownとして表示・保全できることを確認する。
- **対象**: `order_entry` のER図（Entity/Relation）、状態遷移、業務ルール表、DFD、マイグレーション表を1ファイルに共存させた合成サンプル。
- **手順**: (A) サンプルSource固定 → 表示・原文保全（SHO/Obsidian想定）→ (B) のReaderには**まだ依存しない**read-only確認。
- **独立Expected**: 原文byte不変、図表のSubject集合、説明文と定義の境界。
- **禁止**: サンプルの存在を「実Source消費」「Meaning受理」「Build資格」と読み替えない。

### P2-C（Reader接続）: Markdown → Parsed Occurrence

- **目的**: `*.tei.md` の明示Role/Spanを実Readerが消費する。
- **依存**: #3030 M2/M3、#988、#2823。本書は**受入の照合のみ**を所有し、Parser実装は各Ownerへ委譲。
- **手順**: (B) Reader実読込 → byte/span/Source location観測 → (C) Model抽出 → (D) 生成 → (E) Build → (F) 実行。
- **独立Expected**: 旧 `.tei` と新 `*.tei.md` の**同一Subject**に対する旧新比較。消失項目・重複Subject・誤引用を負例とする。
- **禁止**: 単純rename、同byte移動、`validate ok`、CI Greenを新形式の合格としない。

### P3-D（複数Consumer一致）: 図表・CLI・SHOの同一Source投影

- **目的**: 同じAccepted Meaning/Sourceから、図表・CLI・SHOが同じ型付き参照と適用資格を渡す。
- **依存**: #1124（Diagram/Navigation）、#2497（投影意味保持）、#2516（Consumer資格）。
- **独立Expected**: 各ConsumerのScope/Coverage、未解決・非公開の集合、現在地・近傍の構造化テキスト。

## 6. 反例・失敗条件（本書で管理するもの）

| ID | 反例 | 期待する挙動 | 既存Owner |
| --- | --- | --- | --- |
| DG-01 | 図表中にないPatternを「存在しない」と断定 | 固定scope/kinds/rule versionの中でのみ成立することを説明。必要集合自体が未確認ならpartial | #2497/#2514 |
| DG-02 | 生成AIの説明図や手書きガイド図を第二正本に扱う | 共通Accepted Meaning/Sourceからの導出に限定。AI説明はProjectionであって正本ではない | #2497/#1124 |
| DG-03 | SQLiteのcompile/起動PASSをPostgreSQL実Provider合格に昇格 | 方言差を明示。K-07を負例として維持 | #2622/#1174 |
| DG-04 | Parse PASSをGenerate/Build/実行PASSに自動継承 | 段階別独立Expected。前段成功で次段を断定しない | #2514/#1163 |
| DG-05 | `*.tei.md` の説明表・コードフェンス・引用を無条件に実行 | 明示Selection/Roleが無ければ実定義0。空結果を「Ruleが0件なのでPASS」としない | #988/#2621/#2823 |
| DG-06 | 図表から型・Rule・Effectを既定推測で補完 | 写Mappingは明示規則のみ。未知はunknownとして独立Expectedへ | #1126/#2624 |
| DG-07 | Build成功・artifact digest一致を意味受理・Consumer実読取の証拠にする | Envelope同一性・物理公開一式・実Consumer読取は三つの別資格 | #2482/#2604/#2516 |
| DG-08 | 標準フローと差分・合成後の実効フローを混同 | 差し込み個所は相互参照可能に。Plugin/Generator/Target版と受理資格を隠さない | #951/#1124 |

## 7. 実装順序と並行レーン

1. **P0-A（本書）**: 旧 `.tei` 縦断の独立Expected再現。新Readerに依存しない。
2. **P1-B（本書）**: `*.tei.md` サンプルのread-only確認。Reader未接続のまま表示・保全のみ。
3. **P2-C（#3030主担当）**: Reader実装。本書は旧新同一Subjectの比較受入を提供。
4. **P3-D（#1124/#2497/#2516）**: 複数Consumer一致。

**並行**: P0-AとP2-Cは独立に進める。P0-AのPASSをP2-Cへ継承しない。P2-CのParse PASSを実Provider成立へ拡大しない。既存P0-L（旧 `*.tei` 実Provider）・P0-M（新 `*.tei.md` Reader）・P0-J（旧新比較）と重複起票しない。

## 8. 証拠の上限と状態

- #4293は**旧 `.tei` 限定**のParse/Generate/Build/起動の参照実装と、migration SQLの抽出・生成・起動適用を確認した。**新 `*.tei.md` Reader、図表からのModel抽出、実Provider/Host/Owner受理は NOT RUN／NOT ACCEPTED**。
- 本書は設計・受入計画の追補である。新Parser/Runner/Registry/第二Acceptance/Semantic SSOT/Hostは作らない。既存 #3030/#3031/#2621/#988/#2823/#1124/#2497/#2514/#2516/#951/#1174/#2622/#2482/#2604/#2600/#1163 のOwnerを維持する。
- `reference/workspace` は Gate B/C 固定版と現在の参照用作業領域を区別する。固定版を遡及変更しない。`.tei` ファイルを手編集しない。変更は Issue番号と理由をPRへ明記し、`reference-shared-acceptance` の受理経路を使う。
