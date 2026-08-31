from pathlib import Path

path = Path("internal/syntax/parser.go")
text = path.read_text()

marker = '\t\tcase "foreign-key":\n'
if text.count(marker) != 1:
    raise SystemExit(f"foreign-key marker count={text.count(marker)}")

case = '''\t\tcase "unique":
\t\t\t_, _ = p.next()
\t\t\tname, err := p.expect(TokenIdent)
\t\t\tif err != nil {
\t\t\t\treturn m, err
\t\t\t}
\t\t\tconstraint := persistence.UniqueConstraintMapping{Name: name.Text}
\t\t\tif _, err := p.expect(TokenLBrace); err != nil {
\t\t\t\treturn m, err
\t\t\t}
\t\t\tfor {
\t\t\t\tz, err := p.peek()
\t\t\t\tif err != nil {
\t\t\t\t\treturn m, err
\t\t\t\t}
\t\t\t\tif z.Kind == TokenRBrace {
\t\t\t\t\t_, _ = p.next()
\t\t\t\t\tbreak
\t\t\t\t}
\t\t\t\tif _, err := p.expectText("attribute"); err != nil {
\t\t\t\t\treturn m, err
\t\t\t\t}
\t\t\t\ta, err := p.expect(TokenIdent)
\t\t\t\tif err != nil {
\t\t\t\t\treturn m, err
\t\t\t\t}
\t\t\t\tattr, err := core.ParseCanonicalID(a.Text)
\t\t\t\tif err != nil {
\t\t\t\t\treturn m, p.errAt(a, err.Error())
\t\t\t\t}
\t\t\t\tconstraint.Attributes = append(constraint.Attributes, attr)
\t\t\t}
\t\t\tif len(constraint.Attributes) == 0 {
\t\t\t\treturn m, p.errAt(name, "unique requires at least one attribute")
\t\t\t}
\t\t\tm.UniqueConstraints = append(m.UniqueConstraints, constraint)
'''
text = text.replace(marker, case + marker, 1)

old = 'expected database, semantic, table, column, primary-key, index, unique-index, or foreign-key'
new = 'expected database, semantic, table, column, primary-key, unique, index, unique-index, or foreign-key'
if text.count(old) != 1:
    raise SystemExit(f"expected-message count={text.count(old)}")
text = text.replace(old, new, 1)

path.write_text(text)
