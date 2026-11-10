# 内容導出IDの最小参照実装（#2814）

**状態**: 実験的なGo/TypeScript参照実装（TypeScriptの暗号計算は外部Adapterへ委譲）。採用済み標準APIではなく、Core ID/Artifact Digest/署名/実行権限には結線しない。

設計: [可変長・可変文字集合の内容導出ID](../../plan/design/HASH_DERIVED_ID_SCHEME_DESIGN_2026-11.md)。

## Scope

- `Scheme` は `schemeId`、`purpose`、`scope`、`inputKind`、`canonicalization`、`algorithm`、`mode`、出力長・表記規則を明示する。
- v1は `inputKind=bytes` + `canonicalization=raw-bytes/v1` **のみ**。文字列/JSON/Unicodeの暗黙正規化はしない。
- `encodedDigest`: SHA-256（256bit固定）、SHAKE256/cSHAKE256（8の倍数の8〜4096bit）。Base16（小文字）、RFC4648 Base32（小文字・paddingなし）、Base64url（paddingなし）、および任意の印字可能ASCII集合の符号なし整数による固定幅radix表現をサポートする。表現形式だけ変更してもバイナリDigestは不変。
- `exactCharacters`: SHAKE256/cSHAKE256のXOF、2〜94種の重複しない印字可能ASCII文字、1〜1024文字。アルファベットの順序と長さはXOF入力に束縛する。単位はASCII文字（1byte）。Digestの可逆表現ではない。
- frameは各項目のUTF-8 byte長をuint64 big-endianで付与する。ドメイン名 `tei.hashid.frame.v1` とcSHAKE customization `tei.hashid.cshake256.v1` を使用。
- `CollisionProbability` / `CheckCollisionBudget` は独立・一様出力を仮定した誕生日近似を返す。実衝突の検出・DB永続化の一意性保証ではない。

## 他レーンとの設計統合（採用前）

[設計第9節](../../plan/design/HASH_DERIVED_ID_SCHEME_DESIGN_2026-11.md#9-他レーンとの統合設計2026-11-06採用済み境界と候補を区別) では、Everything is Data／Script-first Source／Plugin／生成器／ETL／Plugin self-buildとの対応を整理した。重要なのは、`HashIdScheme` を型付き宣言データ、`Derive` を明示的な変換、`HashIdText` を非権威的な派生値、`TargetProfile`と`BindingsSnapshot` を別入力とすることである。

Source Reader・Accepted revision・実Generatorで消費したbyte列・Targetの原子的保存は**このReferenceの外側**。グラフへ未接続の操作を隠して実装済みとみなさない。標準プラグイン（`tei.std.hashId.*`）への採用は候補段階で、既存Capabilityの正本ID、公開RegistryのFingerprint、署名/Artifact Digestを置換しない。

## Schemeの機械可読な固定形式（PoC v1）

`scheme_wire.go`/`scheme_wire_test.go` と型付きTS `hashid.mts` は、`tei.hashid.scheme/v1` の正準JSONを相互運用する。Wireは **`format`＋全11項目が必須の`scheme`** からなる。UTF-8、固定キー順、余分な空白なし、全フィールド省略なし、HTML用追加エスケープなしの1行を唯一受理し、4KiB超・重複/未知キー・null・版違い・非正規エスケープは拒否する。Go `MarshalSchemeJSON/ParseSchemeJSON`、TS `serializeSchemeJSON/parseSchemeJSON`、共有Golden `scheme_wire_vectors.json` で照合する。

Wireの文字列表現自体は **証明・意味IDではない**。 `encodedDigest` でEncodingだけを切替える場合、Wireは異なるがバイナリDigestは同一。導出アルゴリズムのFrameとWireの版は別契約であり、旧版の自動変換や既存Artifact/署名への適用はしない。

## 可逆表現の復号（追加Reference）

`DecodeDigest` は `encodedDigest` だけを対象に、Base16、RFC4648 Base32、Base64url、Base10/36/62など指定Alphabetの固定幅radix表現を、指定ビット長のDigest byte列へ戻す。大文字、padding、非正規の先頭ゼロ/桁数、範囲超過、文字集合外の記号を拒否し、再エンコードした結果が元文字列に**完全一致**することを要求する。`exactCharacters` は不可逆な生成演算であり復号を拒否する。

`decode_test.go` では128bitの10進39桁・36進25桁・62進22桁とゼロ値/最大値、過大数値と短い表記、SHA-256の各表記往復を検査する。なお **Digestに復号できることは入力Sourceの真正性や署名/権限を証明しない**。

## Target制約と割当境界（追加Reference）

`TargetProfile` は物理ID用の名前空間、**検証規則の版**、許容ASCII文字集合、最小/最大/厳密長、ASCII大小文字区別、予約名を**宣言**する。 `CheckTargetBinding` はID生成結果と、その対象のqualified subjectおよび渡された既存割当Snapshot `[]TargetBinding` を比較し、同一Target名前空間における衝突、大小文字折畳みによる別IDの同一視、同一Subjectの暗黙再割当、不正Snapshotを拒否する。正規化後の物理キーを返すだけで登録しない。

- Schemeが生成し得る**すべての文字**がTargetで許されることを要求する。生成済みIDが偶然有効な文字だけでも、生成規則としては不適合なら拒否する。
- Snapshotの同じTarget名前空間に旧ProfileVersionが含まれる場合は、暗黙の再解釈をせず明示移行を要求する。
- Case-insensitive Targetへは、大文字・小文字の両方を生成し得るAlphabet（例：Base64url、Base62）を拒否する。DBのcollationを自動検出しないため、実Targetのcollationを別途検証したProfileだけに利用する。
- Snapshot内の整合性も候補IDとは独立に検査する。同一Target名前空間の旧割当で、**別Subject間の物理名衝突、同一Subjectの重複、旧Scheme宣言の不正**があれば、候補と無関係でもfail-closed。外部namespaceに属する行は混同しない。これは既存割当の検証であり、Snapshot自体の完全性・鮮度・実DB権限を保証しない。
- Targetで同一名前空間の全割当が得られなければ、Snapshot上で衝突なしと判定できても、実一意性を保証しない。DBへの格納時に一意制約・競合再試行・Transaction等の**実保存境界**が必要。
- 正本との紐付けは `TargetBinding.Subject`（Qualified Source identity）と `TargetBinding.Scheme`（正確な生成規則）に保持する。同じSubjectでもSchemeが異なれば暗黙の冪等再登録とはみなさず、移行を要求する。文字列だけをCanonical ID、権限、Source Acceptanceへ昇格させない。
- 現状はASCIIのみ。Windows予約名/パス末尾規則・実DBのUnicode照合・OS別同値判定はOwnerが宣言する個別Profile/独立試験を要する。

## SQLite実Targetの限定回帰

`sqlite_target_test.go` は、リポジトリ既存の `modernc.org/sqlite` を用いて**実SQLite DB**に `COLLATE NOCASE` と `UNIQUE(namespace, physical_id)` を宣言し、同一大文字小文字Foldの異Subject登録が拒否されることを検査する。もう一つの試験では8つのGo並行呼出しを**単一SQLite接続に直列化**し、1行しか保存されないことを確認する。DB側の一意制約という原子的なGuardの必要性を示すための限定実証であり、マルチ接続・マルチノード・PostgreSQL・Unicode照合・現実の移行/権限まで実証したものではない。

## 型付きTypeScriptと標準cSHAKE検査

`hashid.mts` はHost/Browser/Nodeに依存しない型付き参考モデルで、Scheme検査、binary frame、Base16/Base32/Base64url/radix表現、およびXOFの棄却サンプリングを提供する。SHA-256/SHAKE256/cSHAKE256の実計算は `CryptoAdapter.digest` に委譲する。**Adapterは本物の暗号Providerである必要があり、cSHAKE256ではN=空、S=`tei.hashid.cshake256.v1` を満たす必要がある。** 特定Hostへの結線やKey管理は行わない。

`hashid.typed.test.mjs` は `tsc` で生成した独立`hashid.mjs`をNode `node:crypto` と共通 `vectors.json` に対して試験する。Goの `typed_exec_test.go` は `tsc` とNodeが利用可能な環境でのみ実行し、コンパイラ未導入時は**明示Skip**する。Go CIでのSkipをTypeScript合格と解釈しない。

`cshake_standard_test.go` は[NIST SP 800-185のcSHAKE公開サンプル](https://csrc.nist.gov/projects/cryptographic-standards-and-guidelines/example-values)（X=00010203、N=空、S="Email Signature"、512bit出力）と、N/S両空の場合にcSHAKE256がSHAKE256と一致する性質を独立に確認する。**標準Primitiveの適合試験とTEI独自のFrame・暗号Adapterの相互運用試験は別**。

## Source Read Claimの非権威的な比較（P0接続候補）

`read_claim.go` は、既存Source Reader／実Consumerから提供されることを想定した**申告データの比較器**である。正式なSource Reader、Source Index、Acceptance、Reader Adapter、正本Revisionを新設しない。

`ReadClaim`は`QualifiedSubject`、`SourceScope`、`SourceRevision`、`ReadAttemptID`、`SourceSHA256`を保持する。`CompareUntrustedReadClaims(declared, consumed, consumedBytes)`は5項目の一致と実際に渡されたByteのDigestを検査する。異Subject／異Scope／異Revision／異Attempt／Digest不一致を拒否し、成功しても`status="caller-claims-consistent-only"`、`authority="none"`、`sourceReaderVerified=false`、`sourceAccepted=false`を返す。

**SourceScope ≠ Scheme.Scope**：Source ScopeはQualified Sourceの所属、Scheme ScopeはHash導出ドメインであり、それぞれ独立に宣言・比較する。Source内容が同一でもOwnerの意味的同一性は不明。A→B→Aで古いRevision／Attemptを再利用した申告はこの比較器でも見抜けず、実Source Readerが発行した同一Attempt証拠が必要である。

`read_claim_test.go`には同バイト／別Subject・別Scope・別Revision・別Attempt・不一致バイトの拒否、Source A→B→Aで旧Claimの再利用が成立し得る反例を含めた。**現行CLIにはまだ接続していない**。接続する場合、実Readerの資格のないCaller Claimを`Verified`と表示せず、#1017/#1124/#2595のOwnerが実消費Byteと同一Attemptの根拠を提供する必要がある。単一Fileの局所比較をBuild/ETLのSource Closure全量保証とみなさない。

## 既存TEI Source Indexとの局所照合（P0 Reference、2026-11-07）

`read_claim_sourceindex.go`の`CompareDeclaredSourceIndexRead`は、新しいSource Readerを実装せず、既存`tei/internal/sourceindex`の`NormalizeWorkspacePath`、`Fingerprint`、`ModelRevision`を使う。Callerの`ReadClaim.SourceRevision`は**Source集合全体のModelRevision**（`sha256:<hex>`）、`SourceSHA256`は**対象ファイルのraw SHA-256**（prefixなし）として別々に検査する。対象Path／宣言Index／Consumerから渡された単一ファイルのbytesを局所比較し、全Artifact Pathの曖昧な重複、不正Fingerprint、別Path／不一致ModelRevision／不一致bytesをfail-closed。

`read_claim_sourceindex_test.go`では、選択したファイルのbytesが変わらなくても、**管理集合内の別ファイルの内容やPathが変わるとModelRevisionが変わる**ことを反証する。また、同じ集合のA→B→AでModelRevisionも復元し得るため、内容版だけから単調なAttempt/Revisionを証明できないことを固定する。

成功しても`status=caller-index-bytes-consistent-only`、`authority=none`、`sourceReaderVerified=false`、`qualifiedSubjectVerified=false`、`sourceScopeVerified=false`、`readAttemptVerified=false`、`sourceAccepted=false`、`allSourceBytesVerified=false`。**この関数はCallerのIndexと渡されたbytesを比較しているだけ**で、WorkspaceをOpenして読み取った証拠、実ToolのConsumer byte、Source集合の全ファイル読取、正本のQualified Subject/Scope、Owner AcceptanceやTarget保存には接続しない。既存`operations.readWorkspaceSource`のPinned Readerと`WorkspaceTrace`の各Fingerprintを同一Attemptで結ぶ要件は[設計§9.9](../../plan/design/HASH_DERIVED_ID_SCHEME_DESIGN_2026-11.md)に記載する。

## 最初のHeadless Consumer：read-only `hashid-probe`（W4候補）

`cmd/hashid-probe/` に、実行基盤・Studio・Hostへ依存しない最小CLIを配置する。保存済みの**正準Scheme Wire**と**raw byteの入力ファイル**、呼出元が明示する`subject`から`Derive`を実行する。必要であればローカルの`target-probe`（宣言Profile＋既存割当Snapshot）を読み、`CheckTargetBinding`による**事前診断だけ**を行う。

```bash
go run ./tools/reference-hash-id/cmd/hashid-probe \
  -scheme tools/reference-hash-id/examples/decimal24.scheme.json \
  -input tools/reference-hash-id/examples/input.raw \
  -subject example:record-one \
  -target-probe tools/reference-hash-id/examples/decimal24.target-probe.json \
  -expected-source-sha256 a53b0563c14046eb542a3aa0eeeb7b3d502a935477b474fbe78d81bf33fb93c4 \
  -expected-target-sha256 7c920c2b4aa575b1e700d62e991ef3c7d3f4b6b32960d561d95048a1c33b3b72 \
  -expected-population 1000000 \
  -max-collision-probability 1e-6
```

このfixtureは公開の合成バイト`hello TEI`を使い、独立Python SHAKE256計算による期待`hashIdText=590268890628724110017033`（数字24桁）と、入力の独立SHA-256 `a53b0563c14046eb542a3aa0eeeb7b3d502a935477b474fbe78d81bf33fb93c4`をGoldenとして検査する。返答に`status="caller-pinned-budget-snapshot-preflight-only"`、`callerPinsMatched=true`、`authority="none"`、`storageWrite=false`と、Scheme JSON、入力バイト数/SHA-256、生成文字列、Target名前空間/正規化後キーを含む。入力SHA-256は**この実行が読んだバイト列の説明**であり、受理済Source revision/署名/適切なProviderであることを証明しない。`-target-probe`を省略すると`status="derived-only"`とする。

Caller Digestの照合を次のように扱う。Target事前検査では `-expected-source-sha256` と `-expected-target-sha256` の**両方が必須**。一方、Targetを使わない`derived-only`はSource Pinなしでも実行でき、指定時には一致を検査する。CLIはScheme、Source、Targetを最後にもう一度読み、最初に読み取ったbyte列と一致することも確認する。ただし、**Caller Pinを発行した主体の資格、Source revision/Workspaceとの結びつき、Snapshotの網羅性・現在性、ファイル名/symlinkの安全性、ABA、最後の比較後の変更**は保証しない。単なるFile Hashの一致をFreshnessやAuthorityと呼ばない。

**衝突確率の予算（W2の局所Reference、製品未採用）**：Target事前検査には`-expected-population`（2以上の想定ID件数）と`-max-collision-probability`（許容衝突確率上限。2件以上では **0を拒否**、上限1は条件付きで許可）を追加して**必須**とする。以前から存在する`hashid.CheckCollisionBudget`をScheme検査後・ID生成前に呼び、上限超過・NaN/Infinity/負数・**ゼロ予算**・予算未指定・**想定件数が有限ID空間を上回る場合（鳩の巣原理）**を拒否する。成功JSONの`collisionBudget`は`expectedPopulation`、`maximumCollisionProbability`、`estimatedCollisionProbability`、`collisionRiskUpperBound`、`policySource="caller-declared-unverified"`を保持。上記例は「数字24桁、100万件、許容上限1e-6」の**合成値**であり、TEI標準Policy／実Target容量の採用ではない。同じAlphabetでも20桁・10億件・上限1e-6なら拒否する。結果に表示する`estimatedCollisionProbability`はbirthday近似、実際の予算比較には`collisionRiskUpperBound = min(1,n(n−1)/(2S))`（ペア衝突の合併上界）を用いる。小さい空間では近似値が実確率を下回るためであり、**確率の見かけの値で安全性を主張しない**。例えば2文字種×1桁に2件を割り当てた場合、近似約0.393に対して合併上界0.5で判定し、上限0.4は拒否する。極端に大きな空間ではfloat64表示が0にアンダーフローしても、2件以上で`maximum=0`を受理しない。この計算は**独立・一様分布という仮定付きの数学的上界**で、敵対的入力への強度、正規化後Targetの実有効空間、実DBの既存占有数、将来の衝突なし、原子的保存を保証しない。異なるScheme/Scope/Targetを混同してPopulationを合算・再利用しない。

CLIへのTarget Snapshotは厳密な単一行正準JSON `tei.hashid.target-probe/v1` を要求する。未知・重複フィールド・欠落フィールド・異版・null・過大ファイルを拒否する。**これはReference CLI専用の試験入力で、製品の正本モデル・永続化Schema・完全な現在割当の証明ではない**。入力ファイルは最大4MiB、Scheme最大4KiB、Target Snapshot最大512KiB。すべての局所診断成功後にのみstdoutへJSONを1件出力し、拒否した場合はstderrと非0終了にする。読み込んだSource/Targetのファイルへ書き戻さない。

`subject`は呼出元宣言の識別子であり、このCLIはTEI正本のqualified-refへの存在・Scope・Revision・権限を**検証できない**。CLI出力を受理・発行・割当と呼ばず、登録・署名・Source Acceptance・Atomic Insert/Commitのコードパスを持たせない。実保存は将来のTarget Owner Gate W3、同一ConsumerへScope/Source freshnessを接続する判断はW4として別に扱う。

## P0から実Source Reader／実Consumerへ進める条件（2026-11-09）

[設計§9.11のHR-P0-01〜08](../../plan/design/HASH_DERIVED_ID_SCHEME_DESIGN_2026-11.md) は「同じHash/ModelRevision/Workspace Manifest ≠ 同一物理Root・同一Read Attempt・実Consumer全Source Closure」を明示する。既存の[Direct Trace Draft #3321](https://github.com/hat47x/tei-platform/pull/3321)はWorkspace Manifest/World/Source IndexのLoad-time一致、[UI ABA Draft #3327](https://github.com/hat47x/tei-platform/pull/3327)は表示の旧応答拒否、[実Source消費 #2595](https://github.com/hat47x/tei-platform/issues/2595)は実Generator/Interpreterの入力の同一Attemptをそれぞれ別に扱う。**本Referenceはこれらを実行・複製していない**。

`CompareDeclaredSourceIndexRead`は現在もCallerが作成・提示した`sourceindex.Index`と単一ファイルbyteの一致を照合するのみ。実Pinned Reader/実Tool/Ownerが発行した閉じたread-set/consumption witnessの受領APIは**未接続**であり、`AllSourceBytesVerified=false`などの証拠不足フラグを維持する。Source集合の`ModelRevision`は**内容由来で可逆（A→B→A）**であり、単調epochではない。別Workspaceの完全コピーでもManifest/Sourceの指紋が一致し得る。過去の実行資格が失効したかはReader/Consumerの現在Attemptを別に観測する。

先行段階で合格に必要なのは、(1) 既存`internal/workspace`のピン留めSource Snapshot、(2) Source Index/Revisionを同じ取得から構成したEvidence、(3) 実Consumerが読んだ**全Source Closure**と具体的Attempt、(4) 独立した失効・開示/Owner判断を、別正本を生成せずに結線すること。ユーザー入力やGo Hashの一致から`SourceAccepted`、`TargetAllocated`、`EffectCommitted`を自動推論しない。

## 試験

```bash
go test ./tools/reference-hash-id/... -count=1
gofmt -l tools/reference-hash-id/*.go tools/reference-hash-id/cmd/hashid-probe/*.go
go vet ./tools/reference-hash-id/...
node --test tools/reference-hash-id/hashid.crosslang.test.mjs
```

`hashid_test.go` は、外部Python hashlibで計算したSHA-256/SHAKE256固定フレームの期待値、同一Digestの表現差、exact-mode Golden、スコープ分離、重複・Unicode等の入力拒否、桁数/人口に対する衝突予算拒否を確認する。

`vectors.json` は **GoとNode.jsの両方が実際に読む**共有Golden fixture。`crosslang_fixture_test.go` がGoの`Derive`で計算し、`hashid.crosslang.test.mjs` は独立のNode `node:crypto`（SHA-256/SHAKE256）とバイナリ長フレーム/棄却サンプリングで同じ期待値を検査する。`crosslang_exec_test.go` によりGoの全体試験からNode試験も実行する。GitHub ActionsではNodeが見つからなければ失敗、通常のローカルGo試験では明示的にSkipする。

2026-11-04の初回Node.js v22.16.0での`node --test`は**6件PASS**（実行環境を限定した独立試験）。Go/Node双方の**同一exact headでのCI**は対応するActions結果を確認するまで未実証として扱う。これは独立Node.js試験の実績であり、追加したTypeScript・cSHAKE公開ベクトルの現在のCI合格を示さない。非2冪アルファベットの統計的偏り検定、cSHAKE256 TypeScript Provider独立実装、実DB/OSのcollation/target制約、Unicode profile、既存ID移行は**未実装・未実証**。

この実装の結果を正本IDと認定してはならない。既存値と同名になっても、`Scheme`・用途・scope・正本からの対応を検査する。必要な強度・Target名制約・実Collision対策はConsumer側の明示Policyで行う。

### 例（Go）

```go
scheme := hashid.Scheme{
    SchemeID: "tei.std.hashId.v1",
    Purpose: "example.local",
    Scope: "example.application",
    InputKind: "bytes",
    Canonicalization: "raw-bytes/v1",
    Algorithm: hashid.AlgorithmCSHAKE256,
    Mode: hashid.ModeExactCharacters,
    Alphabet: "0123456789abcdefghjkmnpqrstvwxyz",
    Length: 32,
}
result, err := hashid.Derive(scheme, []byte("source bytes"))
// result.Text is not an authority or a stable semantic identity.
```

## 停止条件

新しいスキームを既存のCanonical ID、Artifact/Manifest/Plugin/Scenario Digest、署名対象、認可・Source Acceptanceに暗黙流用しない。JSONや任意Unicode入力については正規化仕様と相互運用試験を追加するまでは拒否する。
