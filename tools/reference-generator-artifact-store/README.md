# reference-generator-artifact-store（#2016）

選抜receiptのGo/Java/WASM/WASI候補sourceとbinaryを、source digestでcontent-addressedに保存・取得する参照です。保存時と取得時にdigestを再計算し、receipt・object metadata・実byteの不一致を拒否します。

署名・attestation・Authority・実Targetでの実行は扱いません。`grantsAuthority`と`provesRealTarget`は常にfalseです。

```text
go run ./tools/reference-generator-artifact-store/cmd/tei-generator-artifact-store \
  put -receipt /tmp/selection.json -candidate template \
  -candidates tools/reference-generator-selection/testdata/candidates \
  -store /tmp/tei-artifacts

go run ./tools/reference-generator-artifact-store/cmd/tei-generator-artifact-store \
  get -store /tmp/tei-artifacts -source-digest sha256:...

go run ./tools/reference-generator-artifact-store/cmd/tei-generator-artifact-store \
  list -store /tmp/tei-artifacts

go run ./tools/reference-generator-artifact-store/cmd/tei-generator-artifact-store \
  prune -store /tmp/tei-artifacts -keep sha256:... -dry-run

go run ./tools/reference-generator-artifact-store/cmd/tei-generator-artifact-store \
  audit -store /tmp/tei-artifacts

go run ./tools/reference-generator-artifact-store/cmd/tei-generator-artifact-store \
  checkpoint -store /tmp/tei-artifacts -out /tmp/checkpoint.json

go run ./tools/reference-generator-artifact-store/cmd/tei-generator-artifact-store \
  verify-checkpoint -store /tmp/tei-artifacts -checkpoint /tmp/checkpoint.json \
  -signature /tmp/checkpoint.json.sig \
  -allowed-signers /tmp/allowed_signers -identity signer
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

## 一覧と削除

`list`は全objectのmetadataと再計算digestを返します。壊れたobjectが1件でもあれば`passed=false`にします。

`prune`は、明示した`-keep`のsource digestだけを残し、それ以外のobjectを削除します。

- `-dry-run`は削除候補を`wouldDelete`へ返し、実byteを消しません。
- keep setが空、未知のdigestを含む、またはstoreに検証失敗objectがある場合は、何も削除せず`passed=false`にします。
- 削除後にkept objectを取得できること、削除したobjectを取得できないことを試験で確認します。
- 保持期間・監査アクセス権・削除承認は決めません。呼出し側がkeep setを明示する参照です。

## 削除audit

成功した`prune`は、`<store>/prune-audit.jsonl`へ1行のJSONを追記します。

- `operation: prune`、`recordedAt`、`dryRun`
- `keep`、`kept`、`deleted`、`wouldDelete`
- `passed`
- `previousDigest`、`recordDigest`

`recordDigest`はrecord自身のdigest、`previousDigest`は直前recordのdigestです。`audit`はJSONLを読み、schema・必須field・各recordのdigest・previous linkを検証します。壊れた行や改変・並べ替え・欠落があれば`passed=false`にします。

`prune`は既存auditが壊れている場合、削除前に拒否します。失敗した`prune`（空keep、未知digest、検証失敗object、壊れたaudit）は削除を行わないため、成功auditを書きません。

このauditはローカルのhash chainであり、署名・実行者identity・削除承認を証明しません。ファイル全体の置換やchainごと作り直す攻撃は防げません。

`checkpoint`は、検証済みの`recordCount`、`headDigest`、audit file全体の`auditDigest`を返します。`-out`を指定すると、同じJSONをfileへ書き出します。checkpoint自体は署名しません。

```text
tei-generator-artifact-store checkpoint -store /tmp/tei-artifacts -out /tmp/checkpoint.json
ssh-keygen -Y sign -f /tmp/audit-key -n tei-audit /tmp/checkpoint.json
ssh-keygen -Y verify -f /tmp/allowed_signers -I signer -n tei-audit \
  -s /tmp/checkpoint.json.sig < /tmp/checkpoint.json
```

署名はcheckpointのbyte列に束縛されます。署名者のAuthority・trust root・削除承認は検証しません。checkpointと署名を別の場所へ保管すれば、後からaudit全体を照合する参照にできます。

`verify-checkpoint`は、保管したcheckpointの`recordCount`・`headDigest`・`auditDigest`を、現在のstoreから再計算したcheckpointと比較します。mismatchは`passed=false`にします。

`-signature`・`-allowed-signers`・`-identity`を指定すると、先にcheckpointのdetached signatureを`ssh-keygen -Y verify`で検証し、その後にstoreとの一致を検証します。署名が不正なら`signatureVerified=false`、storeと不一致なら`signatureVerified=true`のまま`passed=false`になります。署名者のAuthority・trust rootは検証しません。

## 検証すること・しないこと

| 検証する | 検証しない |
| --- | --- |
| receiptのsource/binary digestと実byteの一致 | 候補の正しさ・意味の受理 |
| 再buildしたbinary digest（Go/Java/WASM/WASI） | 別machine・別toolchainでの再現性 |
| objectのsource/artifactの改変 | 署名・attestation・署名者Authority |
| content-addressed pathとmetadata | 実Provider・実Target・本番配備 |

署名は[`reference-generator-attestation/`](../reference-generator-attestation/README.md)の範囲です。

## 検証

`go test ./tools/reference-generator-artifact-store/... -count=1`で、次を確認します。

- 選抜receiptからsource/binaryを保存し、source digestで取得できる。
- WASI候補のsourceとwasm binaryを保存・取得できる。
- Java候補のsourceと`Main.class`を保存・取得できる。
- 同じobjectの再保存が`reused=true`になる。
- receiptのbinary digestを改変すると`passed=false`。
- 保存後にartifactを改変すると`passed=false`。
- 未知の候補を拒否する。
- `list`が全objectを検証し、改変objectがあれば不合格にする。
- `prune`が明示keepだけを残し、dry-runでは削除しない。
- keep set欠落・未知digest・検証失敗objectでは削除しない。
- 成功したpruneがauditへ記録され、`audit`で読み戻せる。
- 2回のpruneがprevious linkでつながる。
- record改変をdigest不一致で拒否する。
- 壊れたaudit行を拒否し、その状態ではpruneしない。
- `checkpoint`がrecord数・head・audit file digestを返し、改変auditを拒否する。
- checkpointのJSONを`ssh-keygen`で署名・検証でき、改変されたcheckpointを拒否する。
- `verify-checkpoint`が保存checkpointと現在のaudit chainを比較し、mismatchを拒否する。
- `-signature`指定時は署名とstore一致の両方を検証し、改変checkpointを拒否する。

## 未実装・限界

- 暗号学的署名・attestation・署名者Authority。
- 別machine・別toolchain・clean cacheでのbuild再現性。
- 保持期間、監査アクセス権、削除承認、audit/checkpointの署名・実行者identity、chain全体の置換耐性、remote object store。
- artifactの実行承認、実Targetへの配備、ロールバック。
- Python/JavaScript候補の保存。Go/Java/WASM/WASIだけを対象とする。
- WASM/WASIの実runtimeでの実行、実Targetへの配備。
