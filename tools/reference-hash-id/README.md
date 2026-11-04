# 内容導出IDの最小参照実装（#2814）

**状態**: 実験的なGo参照実装。採用済み標準APIではなく、Core ID/Artifact Digest/署名/実行権限には結線しない。

設計: [可変長・可変文字集合の内容導出ID](../../plan/design/HASH_DERIVED_ID_SCHEME_DESIGN_2026-11.md)。

## Scope

- `Scheme` は `schemeId`、`purpose`、`scope`、`inputKind`、`canonicalization`、`algorithm`、`mode`、出力長・表記規則を明示する。
- v1は `inputKind=bytes` + `canonicalization=raw-bytes/v1` **のみ**。文字列/JSON/Unicodeの暗黙正規化はしない。
- `encodedDigest`: SHA-256（256bit固定）、SHAKE256/cSHAKE256（8の倍数の8〜4096bit）。Base16（小文字）、RFC4648 Base32（小文字・paddingなし）、Base64url（paddingなし）、および任意の印字可能ASCII集合の符号なし整数による固定幅radix表現をサポートする。表現形式だけ変更してもバイナリDigestは不変。
- `exactCharacters`: SHAKE256/cSHAKE256のXOF、2〜94種の重複しない印字可能ASCII文字、1〜1024文字。アルファベットの順序と長さはXOF入力に束縛する。単位はASCII文字（1byte）。Digestの可逆表現ではない。
- frameは各項目のUTF-8 byte長をuint64 big-endianで付与する。ドメイン名 `tei.hashid.frame.v1` とcSHAKE customization `tei.hashid.cshake256.v1` を使用。
- `CollisionProbability` / `CheckCollisionBudget` は独立・一様出力を仮定した誕生日近似を返す。実衝突の検出・DB永続化の一意性保証ではない。

## Target制約と割当境界（追加Reference）

`TargetProfile` は物理ID用の名前空間、許容ASCII文字集合、最小/最大/厳密長、ASCII大小文字区別、予約名を**宣言**する。 `CheckTargetBinding` はID生成結果と、その対象のqualified subjectおよび渡された既存割当Snapshot `[]TargetBinding` を比較し、同一Target名前空間における衝突、大小文字折畳みによる別IDの同一視、同一Subjectの暗黙再割当、不正Snapshotを拒否する。正規化後の物理キーを返すだけで登録しない。

- Case-insensitive Targetへは、大文字・小文字の両方を生成し得るAlphabet（例：Base64url、Base62）を拒否する。DBのcollationを自動検出しないため、実Targetのcollationを別途検証したProfileだけに利用する。
- Targetで同一名前空間の全割当が得られなければ、Snapshot上で衝突なしと判定できても、実一意性を保証しない。DBへの格納時に一意制約・競合再試行・Transaction等の**実保存境界**が必要。
- 正本との紐付けは `TargetBinding.Subject`（Qualified Source identity）に保持する。文字列だけをCanonical ID、権限、Source Acceptanceへ昇格させない。
- 現状はASCIIのみ。Windows予約名/パス末尾規則・実DBのUnicode照合・OS別同値判定はOwnerが宣言する個別Profile/独立試験を要する。

## 試験

```bash
go test ./tools/reference-hash-id -count=1
gofmt -l tools/reference-hash-id/*.go
go vet ./tools/reference-hash-id
```

`hashid_test.go` は、外部Python hashlibで計算したSHA-256/SHAKE256固定フレームの期待値、同一Digestの表現差、exact-mode Golden、スコープ分離、重複・Unicode等の入力拒否、桁数/人口に対する衝突予算拒否を確認する。cSHAKEの独立既知ベクトル照合、Go/TypeScriptのクロス言語Golden、非2冪アルファベットの統計的偏り検定、実DB/OSのcollation/target制約、Unicode profile、既存ID移行は**未実装・未実証**。

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
