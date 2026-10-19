# reference-delivery-profile（#2020）

個人・SIer・大規模SaaSの提供形態を、形態別の実装ではなくプラグインの選択として表し、必要な能力が揃っているかを検証するReferenceです。ADR-0036の方針に対応します。

```text
go run ./tools/reference-delivery-profile/cmd/tei-delivery-profile \
  validate -profile tools/reference-delivery-profile/testdata/profiles/saas.json
```

`passed=true`は、選択したpluginの能力宣言がprofileの必須能力を満たしたことだけを示します。実機での動作や、pluginが保証を実際に守ることを証明しません。

`verify`は、conformance testがある能力について、選択したpluginの実動作も確認します。

```text
go run ./tools/reference-delivery-profile/cmd/tei-delivery-profile \
  verify -profile tools/reference-delivery-profile/testdata/profiles/saas.json
```

`verify`の`tenant-isolation`は、[reference-tenant-isolation](../reference-tenant-isolation/README.md)の`Scope`を実際に使い、越境要求、path traversal、自テナントのread/write、symlink越境を確認します。conformance testがない能力は`unverified`として残し、合格とは表示しません。

## Delivery Profile

```json
{
  "schema": "tei.reference.delivery-profile/v0",
  "id": "saas-default",
  "version": "1",
  "requiredCapabilities": ["tenant-isolation", "audit", "authentication"],
  "plugins": [
    {"id": "tenant-isolation", "version": "1", "path": "../../../reference-tenant-isolation/plugin.json"},
    {"id": "audit", "version": "1", "path": "../plugins/audit.json"},
    {"id": "authentication", "version": "1", "path": "../plugins/authentication.json"}
  ]
}
```

`path`はprofile fileからの相対path、または絶対pathです。plugin descriptorのid・versionがprofileの指定と一致しなければ拒否します。

`path`を省略する場合は、`-catalog`でplugin descriptorのdirectoryを指定します。catalogはidとversionの組で解決し、同じ組が複数ある場合は拒否します。

```text
go run ./tools/reference-delivery-profile/cmd/tei-delivery-profile \
  validate -profile PROFILE -catalog PLUGIN_DIR
```

catalogを使う場合、`version`にはexactのほか、`^1.2`、`~1.2`、`>=1.2 <2`、`*`を指定できます。条件を満たす最も新しいversionを選びます。`path`を指定した場合は、そのdescriptorのversionを厳密に比較します。

解決した構成はlock fileへ保存でき、後から同じversion・digest・順序かを検証できます。

```text
go run ./tools/reference-delivery-profile/cmd/tei-delivery-profile \
  resolve -profile PROFILE -catalog PLUGIN_DIR -out PLUGIN_LOCK.json

go run ./tools/reference-delivery-profile/cmd/tei-delivery-profile \
  validate -profile PROFILE -catalog PLUGIN_DIR -lock PLUGIN_LOCK.json
```

plugin descriptorは`requires`と`conflicts`を持てます。`requires`のpluginはprofileで明示的に選択されている必要があります。暗黙に追加しません。

## 検証

- profile schema、id、version、必須能力、plugin一覧を検査する。
- plugin descriptorのschema、id、version、能力を検査する。
- 同じplugin idの重複を拒否する。
- `path`がないplugin参照をcatalogから解決する。catalogにないid・versionは拒否する。
- version制約を満たす最も新しいversionを選び、同値の候補が複数あれば拒否する。
- descriptorのdigestをreceiptへ記録する。
- `-lock`指定時は、lockのprofile・pluginのversionとdigest・順序が解決結果と一致することを確認する。lock自体のdigest不一致も拒否する。
- `requires`のpluginが選択されていない場合、`conflicts`のpluginが選択されている場合は拒否する。
- 依存関係を解決し、`order`へ初期化順を記録する。循環依存は拒否する。
- 必須能力が、選択したpluginの能力で満たされない場合は`missing`へ残し、`passed=false`にする。
- 能力の欠落を、別のpluginや形態別実装で暗黙に補わない。

## 実動作の確認

`verify`は、能力名の一致に加えて、そのReferenceにconformance testがある能力だけを実行します。

- `tenant-isolation`: 二つのテナントrootを作り、越境要求の拒否、`..`による参照の拒否、自テナントのread/write、別テナントrootへのsymlink越境の拒否を確認する。
- それ以外の能力: `unverified`として記録し、動作確認済みとは扱わない。
- conformance testが失敗した場合は`passed=false`とする。

`unverified`は失敗ではありませんが、保証の確認でもありません。実Provider・実Targetでの確認とは分けて扱います。

## 提供形態の例

| profile | 必須能力 | 選択するplugin |
| --- | --- | --- |
| `saas` | tenant-isolation、audit、authentication | tenant-isolation、audit、authentication |
| `individual` | offline-storage、single-user | offline-storage |
| `sier` | export-package、authentication | export-package、authentication、audit（authenticationの依存） |

SaaSの`tenant-isolation`は[reference-tenant-isolation](../reference-tenant-isolation/README.md)のplugin descriptorを使います。形態別のアプリケーションは追加していません。

## 守っている境界

- Profileは選択と検証だけを行い、提供形態の実装を持たない。
- 能力宣言はpluginの契約であり、実装の保証を自動的に証明しない。`guarantees`と`notGuaranteed`を分けて記録する。
- 必須能力が不足する場合はfail closedとし、既定のpluginへフォールバックしない。
- Profileを意味の正本にしない。

## 未実装

- pluginの配布・取得・署名・失効、lockの署名、prerelease・build metadata付きversion。
- `tenant-isolation`以外の能力のconformance test。
- 実Provider・実Targetでの確認。
- 課金・契約・データ所在・監査保存期間。
