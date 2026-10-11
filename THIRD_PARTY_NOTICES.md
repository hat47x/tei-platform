# Third-party notices

TEIのProject codeには、この文書および各コンポーネントの個別LICENSEが適用される。第三者コンポーネントを含む配布物では、以下の著作権表示・ライセンス条件を保持する。

## Go module

### github.com/jackc/pgx/v5 v5.7.6

- 用途: `database/sql`互換PostgreSQL driver（`stdlib`）
- Source: <https://github.com/jackc/pgx/tree/v5.7.6>
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
| `github.com/jackc/pgpassfile` | v1.0.0 | MIT | Jack Christensen; <https://github.com/jackc/pgpassfile/tree/v1.0.0> |
| `github.com/jackc/pgservicefile` | v0.0.0-20240606120523-5a60cdf6a761 | MIT | Jack Christensen; <https://github.com/jackc/pgservicefile/tree/v0.0.0-20240606120523-5a60cdf6a761> |
| `github.com/jackc/puddle/v2` | v2.2.2 | MIT | Jack Christensen; <https://github.com/jackc/puddle/tree/v2.2.2> |
| `golang.org/x/crypto` | v0.37.0 | BSD 3-Clause | The Go Authors; <https://go.googlesource.com/crypto/+/v0.37.0> |
| `golang.org/x/sync` | v0.13.0 | BSD 3-Clause | The Go Authors; <https://go.googlesource.com/sync/+/v0.13.0> |
| `golang.org/x/text` | v0.24.0 | BSD 3-Clause | The Go Authors; <https://go.googlesource.com/text/+/v0.24.0> |

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
