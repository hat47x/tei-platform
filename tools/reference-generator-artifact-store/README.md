# reference-generator-artifact-store（#2016）

選抜receiptのGo候補sourceとbinaryを、source digestでcontent-addressedに保存・取得する参照です。保存時と取得時にdigestを再計算し、receipt・object metadata・実byteの不一致を拒否します。

署名・attestation・Authority・実Targetでの実行は扱いません。`grantsAuthority`と`provesRealTarget`は常にfalseです。

```text
go run ./tools/reference-generator-artifact-store/cmd/tei-generator-artifact-store \
  put -receipt /tmp/selection.json -candidate template \
  -candidates tools/reference-generator-selection/testdata/candidates \
  -store /tmp/tei-artifacts

go run ./tools/reference-generator-artifact-store/cmd/tei-generator-artifact-store \
  get -store /tmp/tei-artifacts -source-digest sha256:...
```

## 保存layout

```text
<store>/objects/<source-digest-hex>/
  source
  artifact
  object.json
```

`object.json`は次を記録します。

- `candidateId`、`language`
- `contractId`、`contractVersion`
- `sourceDigest`、`binaryDigest`
- `receiptDigest`

## 保存時の検査

1. 選抜receiptから候補を探す。
2. 候補sourceを読み、`sourceDigest`を再計算する。
3. `go build -trimpath -buildvcs=false`でbinaryを再buildする。
4. 再buildのbinary digestがreceiptの`binaryDigest`と一致するか確認する。
5. objectが既に存在しdigestが一致すれば再利用する。一致しなければ上書きせず、新しい保存として扱う。

source、receipt、実binaryのどれかが一致しなければ`passed=false`にし、現在の証拠として使わない。

## 取得時の検査

- `source`と`artifact`を読み、`object.json`のdigestと再計算値を照合する。
- 保存後にsourceまたはartifactが改変されていれば`passed=false`にする。
- 要求したsource digestとobject metadataが一致しなければ拒否する。

## 検証すること・しないこと

| 検証する | 検証しない |
| --- | --- |
| receiptのsource/binary digestと実byteの一致 | 候補の正しさ・意味の受理 |
| 再buildしたbinary digest | 別machine・別toolchainでの再現性 |
| objectのsource/artifactの改変 | 署名・attestation・署名者Authority |
| content-addressed pathとmetadata | 実Provider・実Target・本番配備 |

署名は[`reference-generator-attestation/`](../reference-generator-attestation/README.md)の範囲です。

## 検証

`go test ./tools/reference-generator-artifact-store/... -count=1`で、次を確認します。

- 選抜receiptからsource/binaryを保存し、source digestで取得できる。
- 同じobjectの再保存が`reused=true`になる。
- receiptのbinary digestを改変すると`passed=false`。
- 保存後にartifactを改変すると`passed=false`。
- 未知の候補を拒否する。

## 未実装・限界

- 暗号学的署名・attestation・署名者Authority。
- 別machine・別toolchain・clean cacheでのbuild再現性。
- garbage collection、保持期間、監査アクセス権、remote object store。
- artifactの実行承認、実Targetへの配備、ロールバック。
- Python/JavaScript候補（binaryを持たない）の保存。
