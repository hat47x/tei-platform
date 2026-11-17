# 入れ子Go moduleの検査対象

`go vet ./...`と`go test ./...`をリポジトリ直下で実行しても、入れ子の`go.mod`を持つ
moduleへは入りません。そのため`plugins/`や`cmd/`へ独立したmoduleを置くと、root側の
検査は緑のまま、そのmoduleの試験が一度も走らない状態になります（#4612）。

このツールは、入れ子moduleを列挙し、`scope.json`の宣言と突き合わせ、宣言された
moduleを実際に検証します。

```sh
go run ./tools/repository-module-scope              # 宣言の照合と、宣言moduleの検証
go run ./tools/repository-module-scope -check-only   # 宣言の照合だけ
go test -count=1 ./tools/repository-module-scope
```

`tools/repository-health-check.sh`からは、通常は検証まで実行し、`--fast`では
`-check-only`（宣言の照合だけ）を実行します。`--fast`が省くのはmoduleの`go test`
だけで、新しいmoduleが検査対象から外れることはありません。

## 何を保証し、何を保証しないか

- 保証する：リポジトリ内の入れ子moduleがすべて`scope.json`に載っていること。
  載っていないmoduleは失敗し、黙って検査対象から外れません。
- 保証する：`scope.json`に載っているのに実在しないmoduleも失敗すること。宣言だけが
  残り、現在の構成を説明しなくなる状態を許しません。
- 保証する：宣言された**すべてのbuild variant**を検証すること。`//go:build`で
  分かれたfileは別のbuild単位なので、片方だけの検証ではもう片方が丸ごと対象外に
  なります。結果には`[server]`のようなvariant名が付きます。
- 保証する：宣言されたmoduleの`go vet`と`go test`を、そのmoduleのディレクトリで
  実行した結果。
- 保証する：別platform向けのvariantも`go vet`で型検査されること。`go test`は
  交差compileしたbinaryを実行できないので、そのvariantは`not-run`として記録し、
  合格とは書きません。
- 保証する：少なくとも1つのmoduleが`verify: true`であること。全部を除外する宣言は
  失敗するので、何も実行しないまま成功にはなりません。
- 保証する：各エントリが理由を持つこと。`verify: false`の場合は除外の理由として
  表示されます。
- 保証する：`knownFailures`の試験が実際に失敗していること。通る・skipされる・
  消えた試験は失敗として扱われ、除外が原因より長く残りません。
- 保証しない：そのmoduleの試験が意味を持つこと、結果が製品の受入であること、
  CIのpathフィルタがそのmoduleを含むこと、宣言に無いbuild variantの検証、
  別platform variantの`go test`の実行結果。

## `scope.json`

`modules`は全件を載せます。既定は検証する側で、除外は明示した場合だけです。

| 項目 | 意味 |
| --- | --- |
| `path` | リポジトリ直下からのmoduleディレクトリ |
| `verify` | `true`なら`go vet`と`go test`を実行する |
| `reason` | その検証方式を選ぶ理由。**全エントリで必須**（20文字以上）。`verify: false`なら除外の理由として表示される |
| `variants` | 検証するbuild構成の配列。省略するとtag無しの1構成だけを検証する |
| `env` | そのmoduleだけに足す環境変数（`CGO_ENABLED`など） |
| `knownFailures` | 既存の失敗。`test`（試験名）、`reason`（20文字以上）、`variants`があるmoduleでは`variant`（構成名）が必須 |

`variants`の各要素は`name`（必須・重複不可）、`tags`（省略可）、`os`（省略可）を
持ちます。`//go:build`の両側を検証するときは、両方を宣言します。

```json
"variants": [
  { "name": "server", "tags": "server" },
  { "name": "desktop", "os": "windows" }
]
```

`os`は検証するtarget platform（`GOOS`）です。省略すると実行中のplatformを
対象にします。**platformを指定したvariantは`go vet`で型検査されますが、
`go test`は実行できません。** 交差compileしたtest binaryは実行できないためで、
そのvariantは`not-run`として表示し、合格とは書きません。platformを指定する
理由は、対象platformでしかbuildできないfileを検査対象から落とさないことです。

**同じ`tags`と`os`の組を2つのvariantに書けません。** 同じbuild構成を2回検証
しても増える情報は無く、2つ目の名前が実際には検査していない構成の被覆を主張
してしまうためです。tagの並びや区切りが違うだけの同じ集合も重複として扱います。
`tags`が同じでも`os`が違えば別の構成です（compileされるfileが違うため）。

**宣言は厳格に読みます。** 未知のキーは失敗します。`variants`の綴り違いや、
`variants`導入前の`tags`が残っていると、素の`json.Unmarshal`はそのキーを
黙って捨て、エントリをtag無しの1構成へ降格させます。それはこのファイルが
防ごうとしている脱落そのものなので、書かれた意味を理解できない宣言は
受け付けません。

`knownFailures`は「この変更の外にある既知の失敗」を記録する場所で、放置の許可では
ありません。`variants`を持つmoduleでは`variant`で構成を指定します。ある構成の
ための失敗は他の構成には存在しないので、指定が無いと他の構成で「失敗しなかった」
と報告されてしまいます。試験が通る・skipされる・消えると
`repository-module-scope`は失敗し、項目の削除を求めます。

## 除外の扱い

`vendor/`、`node_modules/`、`testdata/`、ドット始まりのディレクトリは列挙しません。
Goがそれらをこのリポジトリのbuild対象にしないためです。それ以外の入れ子moduleは
すべて宣言が必要です。
