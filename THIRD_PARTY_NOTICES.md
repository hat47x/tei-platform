# Third-party notices

TEIのProject codeには、この文書および各コンポーネントの個別LICENSEが適用される。第三者コンポーネントを含む配布物では、以下の著作権表示・ライセンス条件を保持する。

## go.mod 全依存の網羅表

`tools/repository-plan-check`（`go test ./...` から実行される、#2330の機械検査）が、ルート `go.mod` と `cmd/tei-studio/go.mod` のrequire全件が、この文書の**module名と該当版が同じ行に載っている**かを検査する。新しいmoduleを追加したら、該当版とともにこの表へ載せる。載せないmoduleは、この文書に載せない理由を記録する。

分類は、依存がどの配布物に入るかの区別（#2330 完了条件3点目）を示す。

- **製品CLIに入る**: `go list -deps ./cmd/tei` の依存グラフに現れる。
- **参照ツール専用**: ルートmoduleの依存のうち、製品CLIには入らず、参照ツール・参照実装・生成物のbuildと実行に使う。
- **Studio Desktop専用**: `cmd/tei-studio` のdesktop appに入る。

2026-10-31時点の事実として、`go list -deps ./cmd/tei` は外部moduleを1つも含まない（標準libraryのみ）。したがってルートmoduleの外部依存はすべて「参照ツール専用」であり、製品CLIの配布物に第三者Go moduleは現れない。

| Module | Version | 分類 | License | 掲載理由・用途 |
| --- | --- | --- | --- | --- |
| `github.com/jackc/pgx/v5` | v5.9.2 | 参照ツール専用 | MIT | PostgreSQL driver。`internal/security/effectcommit/postgres` と `tools/internal/postgresdb`、生成したサーバーの実行に使う。本文は下記の専用節。 |
| `github.com/jackc/pgpassfile` | v1.0.0 | 参照ツール専用 | MIT | pgxの実行時依存。 |
| `github.com/jackc/pgservicefile` | v0.0.0-20240606120523-5a60cdf6a761 | 参照ツール専用 | MIT | pgxの実行時依存。 |
| `github.com/jackc/puddle/v2` | v2.2.2 | 参照ツール専用 | MIT | pgxの実行時依存。 |
| `golang.org/x/sync` | v0.22.0 | 参照ツール専用 | BSD 3-Clause | pgxの実行時依存。 |
| `golang.org/x/text` | v0.41.0 | 参照ツール専用 | BSD 3-Clause | 参照実装（`reference/multilingual-*`、`reference/standard-messages`）の多言語処理。 |
| `github.com/sijms/go-ora/v2` | v2.9.0 | 参照ツール専用 | MIT | Oracleの隔離検証用データを扱う参照ツールの暫定driver。本文は下記の専用節。 |
| `modernc.org/sqlite` | v1.34.5 | 参照ツール専用 | BSD 3-Clause（SQLiteのC部分はpublic domain） | 参照ツール `tools/reference-target-profile` のartifact保存・rollback。本文は下記の専用節。 |
| `modernc.org/libc` | v1.55.3 | 参照ツール専用 | BSD 3-Clause（musl由来部分はMIT） | sqliteの実行時依存。 |
| `modernc.org/mathutil` | v1.6.0 | 参照ツール専用 | BSD 3-Clause | sqliteの実行時依存。 |
| `modernc.org/memory` | v1.8.0 | 参照ツール専用 | BSD 3-Clause | sqliteの実行時依存。 |
| `github.com/dustin/go-humanize` | v1.0.1 | 参照ツール専用 | MIT | sqliteの実行時依存。 |
| `github.com/google/uuid` | v1.6.0 | 参照ツール専用 | BSD 3-Clause | sqliteの実行時依存。 |
| `github.com/mattn/go-isatty` | v0.0.20 | 参照ツール専用 | MIT | sqliteとWailsの実行時依存（ルート・`cmd/tei-studio` の両go.mod）。 |
| `github.com/ncruces/go-strftime` | v0.1.9 | 参照ツール専用 | MIT | sqliteの実行時依存。 |
| `github.com/remyoudompheng/bigfft` | v0.0.0-20230129092748-24d4a6f8daec | 参照ツール専用 | BSD 3-Clause | sqliteの実行時依存。 |
| `golang.org/x/sys` | v0.22.0（ルート）/ v0.46.0（cmd/tei-studio） | 参照ツール専用 | BSD 3-Clause | sqliteとWailsの実行時依存。両moduleで版が異なるため併記する。 |
| `github.com/wailsapp/wails/v3` | v3.0.0-beta.29 | Studio Desktop専用 | MIT | TEI Studio desktop appのUI runtime。下記の専用節。 |
| `github.com/adrg/xdg` | v0.5.3 | Studio Desktop専用 | MIT | Wailsの実行時依存。 |
| `github.com/coder/websocket` | v1.8.14 | Studio Desktop専用 | ISC | Wailsの実行時依存。 |
| `github.com/go-ole/go-ole` | v1.3.0 | Studio Desktop専用 | MIT | Wailsの実行時依存。 |
| `github.com/godbus/dbus/v5` | v5.2.2 | Studio Desktop専用 | BSD 2-Clause | Wailsの実行時依存。 |
| `github.com/mattn/go-colorable` | v0.1.14 | Studio Desktop専用 | MIT | Wailsの実行時依存。 |

各License欄と引用したlicense本文は、該当版を `go mod download` で取得した実文件（LICENSE、SQLITE-LICENSE等）から確認した事実の記録である。配布物への最終的な法的な扱いはAGENTS.mdのA4に従い人間の判断を待つ。

## Go module

### github.com/jackc/pgx/v5 v5.9.2

- 用途: `database/sql`互換PostgreSQL driver（`stdlib`）
- Source: <https://github.com/jackc/pgx/tree/v5.9.2>
- License: MIT

```text
Copyright (c) 2013-2021 Jack Christensen

MIT License

Permission is hereby granted, free of charge, to any person obtaining
a copy of this software and associated documentation files (the
"Software"), to deal in the Software without restriction, including
without limitation the rights to use, copy, modify, merge, publish,
distribute, sublicense, and/or sell copies of the Software, and to
permit persons to whom the Software is furnished to do so, subject to
the following conditions:

The above copyright notice and this permission notice shall be
included in all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND,
EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF
MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND
NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE
LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION
OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION
WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.
```

### pgx runtime dependencies

The following modules are required by the `pgx/stdlib` runtime graph. Their license files were inspected in the exact versions resolved by `go.mod`:

| Module | Version | License | Copyright / source |
| --- | --- | --- | --- |
| `github.com/jackc/pgpassfile` | v1.0.0 | MIT | Copyright (c) 2019 Jack Christensen; <https://github.com/jackc/pgpassfile/tree/v1.0.0> |
| `github.com/jackc/pgservicefile` | v0.0.0-20240606120523-5a60cdf6a761 | MIT | Copyright (c) 2020 Jack Christensen; <https://github.com/jackc/pgservicefile/tree/v0.0.0-20240606120523-5a60cdf6a761> |
| `github.com/jackc/puddle/v2` | v2.2.2 | MIT | Copyright (c) 2018 Jack Christensen; <https://github.com/jackc/puddle/tree/v2.2.2> |
| `golang.org/x/sync` | v0.22.0 | BSD 3-Clause | The Go Authors; <https://go.googlesource.com/sync/+/v0.22.0> |
| `golang.org/x/text` | v0.41.0 | BSD 3-Clause | The Go Authors; <https://go.googlesource.com/text/+/v0.41.0> |

The MIT license text above also applies to the three Jack Christensen modules listed here; their individual copyright notices are listed in the table.

```text
Copyright 2009 The Go Authors.

Redistribution and use in source and binary forms, with or without
modification, are permitted provided that the following conditions are
met:

   * Redistributions of source code must retain the above copyright
notice, this list of conditions and the following disclaimer.
   * Redistributions in binary form must reproduce the above
copyright notice, this list of conditions and the following disclaimer
in the documentation and/or other materials provided with the
distribution.
   * Neither the name of Google LLC nor the names of its
contributors may be used to endorse or promote products derived from
this software without specific prior written permission.

THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS
"AS IS" AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT
LIMITED TO, THE IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR
A PARTICULAR PURPOSE ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT
OWNER OR CONTRIBUTORS BE LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL,
SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT
LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES; LOSS OF USE,
DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND ON ANY
THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
(INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE
OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
```

The root [`LICENSE`](LICENSE) applies to TEI project code. Developer-side authoring tools and skills are maintained outside this product repository and are not product dependencies.


### github.com/sijms/go-ora/v2 v2.9.0

- 用途: Oracleの隔離された検証用データを扱う参照ツールの暫定ドライバー。製品ドライバーの採用決定ではない。
- Source: <https://github.com/sijms/go-ora/tree/v2.9.0>
- License: MIT

```text
MIT License

Copyright (c) 2020 Samy Sultan

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
```

### modernc.org/sqlite v1.34.5（参照ツール専用）

- 用途: 参照ツール `tools/reference-target-profile` がartifactの保存・rollbackに使う埋込みSQLite（`deploy_sqlite.go`）。製品CLI（`cmd/tei`）には入らない。
- Source: <https://pkg.go.dev/modernc.org/sqlite@v1.34.5>
- License: SQLiteのGo translationはBSD 3-Clause。翻訳元のSQLite C sourceはpublic domain（同moduleの`SQLITE-LICENSE`）。

```text
Copyright (c) 2017 The Sqlite Authors. All rights reserved.

Redistribution and use in source and binary forms, with or without
modification, are permitted provided that the following conditions are met:

1. Redistributions of source code must retain the above copyright notice, this
list of conditions and the following disclaimer.

2. Redistributions in binary form must reproduce the above copyright notice,
this list of conditions and the following disclaimer in the documentation
and/or other materials provided with the distribution.

3. Neither the name of the copyright holder nor the names of its contributors
may be used to endorse or promote products derived from this software without
specific prior written permission.

THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS" AND
ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE IMPLIED
WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE
DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT HOLDER OR CONTRIBUTORS BE LIABLE
FOR ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL
DAMAGES (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR
SERVICES; LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER
CAUSED AND ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY,
OR TORT (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE
OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
```

`SQLITE-LICENSE`は、出荷対象のSQLiteのcodeとdocumentがpublic domainに捧げられていることを述べる。

```text
SQLite Is Public Domain

All of the code and documentation in SQLite has been dedicated to the public
domain by the authors. All code authors, and representatives of the companies
they work for, have signed affidavits dedicating their contributions to the
public domain and originals of those signed affidavits are stored in a firesafe
at the main offices of Hwaci. Anyone is free to copy, modify, publish, use,
compile, sell, or distribute the original SQLite code, either in source code
form or as a compiled binary, for any purpose, commercial or non-commercial,
and by any means.
```

#### sqliteの実行時依存

`go.mod`でindirectとして現れる、sqliteの実行時依存。License本文は該当版の実文件から確認した。

| Module | Version | License | Copyright / source |
| --- | --- | --- | --- |
| `modernc.org/libc` | v1.55.3 | BSD 3-Clause（`LICENSE`）。Go標準library由来のfileはBSD 3-Clause（`LICENSE-GO`）。musl由来の部分はMIT（`COPYRIGHT-MUSL`） | Copyright (c) 2017 The Libc Authors; LICENSE-GO: Copyright (c) 2009 The Go Authors; COPYRIGHT-MUSL: Copyright © 2005-2020 Rich Felker, et al.; <https://pkg.go.dev/modernc.org/libc@v1.55.3> |
| `modernc.org/mathutil` | v1.6.0 | BSD 3-Clause | Copyright (c) 2014 The mathutil Authors; <https://pkg.go.dev/modernc.org/mathutil@v1.6.0> |
| `modernc.org/memory` | v1.8.0 | BSD 3-Clause（`LICENSE`・`LICENSE-GO`・`LICENSE-MMAP-GO`） | Copyright (c) 2017 The Memory Authors; LICENSE-GO: Copyright (c) 2009 The Go Authors; LICENSE-MMAP-GO: Copyright (c) 2011, Evan Shaw; <https://pkg.go.dev/modernc.org/memory@v1.8.0> |
| `github.com/dustin/go-humanize` | v1.0.1 | MIT | Copyright (c) 2005-2008 Dustin Sallings; <https://pkg.go.dev/github.com/dustin/go-humanize@v1.0.1> |
| `github.com/google/uuid` | v1.6.0 | BSD 3-Clause | Copyright (c) 2009,2014 Google Inc.; <https://pkg.go.dev/github.com/google/uuid@v1.6.0> |
| `github.com/mattn/go-isatty` | v0.0.20 | MIT | Copyright (c) Yasuhiro MATSUMOTO; <https://pkg.go.dev/github.com/mattn/go-isatty@v0.0.20> |
| `github.com/ncruces/go-strftime` | v0.1.9 | MIT | Copyright (c) 2022 Nuno Cruces; <https://pkg.go.dev/github.com/ncruces/go-strftime@v0.1.9> |
| `github.com/remyoudompheng/bigfft` | v0.0.0-20230129092748-24d4a6f8daec | BSD 3-Clause | Copyright (c) 2012 The Go Authors; <https://pkg.go.dev/github.com/remyoudompheng/bigfft@v0.0.0-20230129092748-24d4a6f8daec> |
| `golang.org/x/sys` | v0.22.0（ルート）/ v0.46.0（cmd/tei-studio） | BSD 3-Clause | Copyright (c) 2009 The Go Authors; <https://pkg.go.dev/golang.org/x/sys@v0.22.0> / <https://pkg.go.dev/golang.org/x/sys@v0.46.0> |

MITの本文は「pgx runtime dependencies」の節に載せたものが、表の著作権表示とともに該当moduleへそのまま適用される。BSD 3-Clauseの本文（Go Authors、行頭`*`形）も同じく「pgx runtime dependencies」の節に載せたものが適用される。`modernc.org/sqlite`自身のBSD本文は行頭が番号の形のため上に全文を引用した。

### cmd/tei-studio（TEI Studio Desktop）の依存

`cmd/tei-studio` はWails v3でbuildするdesktop appで、独立した `cmd/tei-studio/go.mod` を持つ。実行時依存はWailsのUI runtime graphである。`github.com/mattn/go-isatty`（v0.0.20）と `golang.org/x/sys`（v0.46.0）は上のsqliteの表に併記した。

| Module | Version | License | Copyright / source |
| --- | --- | --- | --- |
| `github.com/wailsapp/wails/v3` | v3.0.0-beta.29 | MIT | Copyright (c) 2018-Present Lea Anthony; <https://pkg.go.dev/github.com/wailsapp/wails/v3@v3.0.0-beta.29> |
| `github.com/adrg/xdg` | v0.5.3 | MIT | Copyright (c) 2014 Adrian-George Bostan; <https://pkg.go.dev/github.com/adrg/xdg@v0.5.3> |
| `github.com/coder/websocket` | v1.8.14 | ISC | Copyright (c) 2025 Coder; <https://pkg.go.dev/github.com/coder/websocket@v1.8.14> |
| `github.com/go-ole/go-ole` | v1.3.0 | MIT | Copyright © 2013-2017 Yasuhiro Matsumoto; <https://pkg.go.dev/github.com/go-ole/go-ole@v1.3.0> |
| `github.com/godbus/dbus/v5` | v5.2.2 | BSD 2-Clause | Copyright (c) 2013, Georg Reinke, Google; <https://pkg.go.dev/github.com/godbus/dbus/v5@v5.2.2> |
| `github.com/mattn/go-colorable` | v0.1.14 | MIT | Copyright (c) 2016 Yasuhiro Matsumoto; <https://pkg.go.dev/github.com/mattn/go-colorable@v0.1.14> |

MITの本文は「pgx runtime dependencies」の節に載せたものが適用される。ISCとBSD 2-Clauseは短いため、次に全文を載せる。

```text
Copyright (c) 2025 Coder

Permission to use, copy, modify, and distribute this software for any
purpose with or without fee is hereby granted, provided that the above
copyright notice and this permission notice appear in all copies.

THE SOFTWARE IS PROVIDED "AS IS" AND THE AUTHOR DISCLAIMS ALL WARRANTIES
WITH REGARD TO THIS SOFTWARE INCLUDING ALL IMPLIED WARRANTIES OF
MERCHANTABILITY AND FITNESS. IN NO EVENT SHALL THE AUTHOR BE LIABLE FOR
ANY SPECIAL, DIRECT, INDIRECT, OR CONSEQUENTIAL DAMAGES OR ANY DAMAGES
WHATSOEVER RESULTING FROM LOSS OF USE, DATA OR PROFITS, WHETHER IN AN
ACTION OF CONTRACT, NEGLIGENCE OR OTHER TORTIOUS ACTION, ARISING OUT OF
OR IN CONNECTION WITH THE USE OR PERFORMANCE OF THIS SOFTWARE.
```

```text
Copyright (c) 2013, Georg Reinke (<guelfey at gmail dot com>), Google
All rights reserved.

Redistribution and use in source and binary forms, with or without
modification, are permitted provided that the following conditions
are met:

1. Redistributions of source code must retain the above copyright notice,
this list of conditions and the following disclaimer.

2. Redistributions in binary form must reproduce the above copyright
notice, this list of conditions and the following disclaimer in the
documentation and/or other materials provided with the distribution.

THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS
"AS IS" AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT
LIMITED TO, THE IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR
A PARTICULAR PURPOSE ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT
HOLDER OR CONTRIBUTORS BE LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL,
SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT LIMITED
TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES; LOSS OF USE, DATA, OR
PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND ON ANY THEORY OF
LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT (INCLUDING
NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE OF THIS
SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
```
