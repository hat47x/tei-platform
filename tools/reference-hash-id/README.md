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
- Targetで同一名前空間の全割当が得られなければ、Snapshot上で衝突なしと判定できても、実一意性を保証しない。DBへの格納時に一意制約・競合再試行・Transaction等の**実保存境界**が必要。
- 正本との紐付けは `TargetBinding.Subject`（Qualified Source identity）と `TargetBinding.Scheme`（正確な生成規則）に保持する。同じSubjectでもSchemeが異なれば暗黙の冪等再登録とはみなさず、移行を要求する。文字列だけをCanonical ID、権限、Source Acceptanceへ昇格させない。
- 現状はASCIIのみ。Windows予約名/パス末尾規則・実DBのUnicode照合・OS別同値判定はOwnerが宣言する個別Profile/独立試験を要する。

## SQLite実Targetの限定回帰

`sqlite_target_test.go` は、リポジトリ既存の `modernc.org/sqlite` を用いて**実SQLite DB**に `COLLATE NOCASE` と `UNIQUE(namespace, physical_id)` を宣言し、同一大文字小文字Foldの異Subject登録が拒否されることを検査する。もう一つの試験では8つのGo並行呼出しを**単一SQLite接続に直列化**し、1行しか保存されないことを確認する。DB側の一意制約という原子的なGuardの必要性を示すための限定実証であり、マルチ接続・マルチノード・PostgreSQL・Unicode照合・現実の移行/権限まで実証したものではない。

## 型付きTypeScriptと標準cSHAKE検査

`hashid.mts` はHost/Browser/Nodeに依存しない型付き参考モデルで、Scheme検査、binary frame、Base16/Base32/Base64url/radix表現、およびXOFの棄却サンプリングを提供する。SHA-256/SHAKE256/cSHAKE256の実計算は `CryptoAdapter.digest` に委譲する。**Adapterは本物の暗号Providerである必要があり、cSHAKE256ではN=空、S=`tei.hashid.cshake256.v1` を満たす必要がある。** 特定Hostへの結線やKey管理は行わない。

`hashid.typed.test.mjs` は `tsc` で生成した独立`hashid.mjs`をNode `node:crypto` と共通 `vectors.json` に対して試験する。Goの `typed_exec_test.go` は `tsc` とNodeが利用可能な環境でのみ実行し、コンパイラ未導入時は**明示Skip**する。Go CIでのSkipをTypeScript合格と解釈しない。

`cshake_standard_test.go` は[NIST SP 800-185のcSHAKE公開サンプル](https://csrc.nist.gov/projects/cryptographic-standards-and-guidelines/example-values)（X=00010203、N=空、S="Email Signature"、512bit出力）と、N/S両空の場合にcSHAKE256がSHAKE256と一致する性質を独立に確認する。**標準Primitiveの適合試験とTEI独自のFrame・暗号Adapterの相互運用試験は別**。

## 最初のHeadless Consumer：read-only `hashid-probe`（W4候補）

`cmd/hashid-probe/` に、実行基盤・Studio・Hostへ依存しない最小CLIを配置する。保存済みの**正準Scheme Wire**と**raw byteの入力ファイル**、呼出元が明示する`subject`から`Derive`を実行する。必要であればローカルの`target-probe`（宣言Profile＋既存割当Snapshot）を読み、`CheckTargetBinding`による**事前診断だけ**を行う。

```bash
go run ./tools/reference-hash-id/cmd/hashid-probe \
  -scheme tools/reference-hash-id/examples/decimal24.scheme.json \
  -input tools/reference-hash-id/examples/input.raw \
  -subject example:record-one \
  -target-probe tools/reference-hash-id/examples/decimal24.target-probe.json
```

このfixtureは公開の合成バイト`hello TEI`を使い、返答に`status="snapshot-preflight-only"`、`authority="none"`、`storageWrite=false`と、Scheme JSON、入力バイト数/SHA-256、生成文字列、Target名前空間/正規化後キーを含む。入力SHA-256は**この実行が読んだバイト列の説明**であり、受理済Source revision/署名/適切なProviderであることを証明しない。`-target-probe`を省略すると`status="derived-only"`とする。

CLIへのTarget Snapshotは厳密な単一行正準JSON `tei.hashid.target-probe/v1` を要求する。未知・重複フィールド・欠落フィールド・異版・null・過大ファイルを拒否する。**これはReference CLI専用の試験入力で、製品の正本モデル・永続化Schema・完全な現在割当の証明ではない**。入力ファイルは最大4MiB、Scheme最大4KiB、Target Snapshot最大512KiB。すべての局所診断成功後にのみstdoutへJSONを1件出力し、拒否した場合はstderrと非0終了にする。読み込んだSource/Targetのファイルへ書き戻さない。

`subject`は呼出元宣言の識別子であり、このCLIはTEI正本のqualified-refへの存在・Scope・Revision・権限を**検証できない**。CLI出力を受理・発行・割当と呼ばず、登録・署名・Source Acceptance・Atomic Insert/Commitのコードパスを持たせない。実保存は将来のTarget Owner Gate W3、同一ConsumerへScope/Source freshnessを接続する判断はW4として別に扱う。

## 試験

```bash
go test ./tools/reference-hash-id -count=1
gofmt -l tools/reference-hash-id/*.go
go vet ./tools/reference-hash-id
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
