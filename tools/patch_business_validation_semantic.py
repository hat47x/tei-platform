from pathlib import Path


def replace_once(path: Path, old: str, new: str) -> None:
    text = path.read_text()
    count = text.count(old)
    if count != 1:
        raise SystemExit(f"{path}: expected exactly one match, got {count}")
    path.write_text(text.replace(old, new, 1))


parser = Path("internal/syntax/parser.go")
if "func (p *Parser) parseSemanticValidation(" in parser.read_text():
    print("semantic validation parser is already patched")
    raise SystemExit(0)

replace_once(
    parser,
    "// ParseSemanticFile parses a semantic source file. M0 deliberately keeps the\n// syntax small: a namespace followed by type, data and capability declarations.",
    "// ParseSemanticFile parses a semantic source file. The syntax remains intentionally\n// small: a namespace followed by type, data, capability and typed validation declarations.",
)
replace_once(
    parser,
    '''\t\tcase "capability":
\t\t\tc, err := p.parseSemanticCapability(nsID)
\t\t\tif err != nil {
\t\t\t\treturn ns, err
\t\t\t}
\t\t\tns.Capabilities = append(ns.Capabilities, c)
\t\tdefault:
\t\t\treturn ns, p.errAt(t, "expected type, data, or capability")''',
    '''\t\tcase "capability":
\t\t\tc, err := p.parseSemanticCapability(nsID)
\t\t\tif err != nil {
\t\t\t\treturn ns, err
\t\t\t}
\t\t\tns.Capabilities = append(ns.Capabilities, c)
\t\tcase "validation":
\t\t\tv, err := p.parseSemanticValidation(nsID)
\t\t\tif err != nil {
\t\t\t\treturn ns, err
\t\t\t}
\t\t\tns.Validations = append(ns.Validations, v)
\t\tdefault:
\t\t\treturn ns, p.errAt(t, "expected type, data, capability, or validation")''',
)

marker = "func parseVersion(s string) (database.ContractVersion, error) {"
text = parser.read_text()
if text.count(marker) != 1:
    raise SystemExit("internal/syntax/parser.go: parseVersion marker mismatch")
validation_parser = '''func (p *Parser) parseSemanticValidation(namespace core.CanonicalID) (semantic.Validation, error) {
\t_, _ = p.next() // validation
\tidTok, err := p.expect(TokenIdent)
\tif err != nil {
\t\treturn semantic.Validation{}, err
\t}
\tvar id core.CanonicalID
\tsymbol := idTok.Text
\tif strings.Contains(idTok.Text, ".") {
\t\tid, err = core.ParseCanonicalID(idTok.Text)
\t\tif err != nil {
\t\t\treturn semantic.Validation{}, p.errAt(idTok, err.Error())
\t\t}
\t\tif !strings.HasPrefix(string(id), string(namespace)+".") {
\t\t\treturn semantic.Validation{}, p.errAt(idTok, "validation canonical id must belong to the declared semantic namespace")
\t\t}
\t\tparts := strings.Split(string(id), ".")
\t\tsymbol = parts[len(parts)-1]
\t} else {
\t\tseg, err := core.SymbolToSegment(idTok.Text)
\t\tif err != nil {
\t\t\treturn semantic.Validation{}, p.errAt(idTok, err.Error())
\t\t}
\t\tid, _ = core.ParseCanonicalID(string(namespace) + "." + seg)
\t}

\tkindTok, err := p.expect(TokenIdent)
\tif err != nil {
\t\treturn semantic.Validation{}, err
\t}
\tv := semantic.Validation{ID: id, Symbol: symbol, Kind: semantic.ValidationKind(kindTok.Text)}
\tif !v.Kind.Valid() {
\t\treturn v, p.errAt(kindTok, fmt.Sprintf("unsupported semantic validation kind %q", kindTok.Text))
\t}
\tif _, err := p.expect(TokenLBrace); err != nil {
\t\treturn v, err
\t}

\tvar hasAttribute, hasRelation bool
\tfor {
\t\tt, err := p.peek()
\t\tif err != nil {
\t\t\treturn v, err
\t\t}
\t\tif t.Kind == TokenRBrace {
\t\t\t_, _ = p.next()
\t\t\tbreak
\t\t}
\t\tswitch t.Text {
\t\tcase "attribute":
\t\t\tif hasAttribute {
\t\t\t\treturn v, p.errAt(t, "attribute is declared more than once")
\t\t\t}
\t\t\thasAttribute = true
\t\t\t_, _ = p.next()
\t\t\ta, err := p.expect(TokenIdent)
\t\t\tif err != nil {
\t\t\t\treturn v, err
\t\t\t}
\t\t\tv.Attribute, err = core.ParseCanonicalID(a.Text)
\t\t\tif err != nil {
\t\t\t\treturn v, p.errAt(a, err.Error())
\t\t\t}
\t\tcase "relation":
\t\t\tif hasRelation {
\t\t\t\treturn v, p.errAt(t, "relation is declared more than once")
\t\t\t}
\t\t\thasRelation = true
\t\t\t_, _ = p.next()
\t\t\tr, err := p.expect(TokenIdent)
\t\t\tif err != nil {
\t\t\t\treturn v, err
\t\t\t}
\t\t\tv.Relation, err = core.ParseCanonicalID(r.Text)
\t\t\tif err != nil {
\t\t\t\treturn v, p.errAt(r, err.Error())
\t\t\t}
\t\tdefault:
\t\t\treturn v, p.errAt(t, "expected attribute or relation")
\t\t}
\t}

\tif v.Attribute == "" {
\t\treturn v, fmt.Errorf("semantic validation %s requires attribute", v.ID)
\t}
\tswitch v.Kind {
\tcase semantic.ValidationPositive:
\t\tif v.Relation != "" {
\t\t\treturn v, fmt.Errorf("positive semantic validation %s does not accept relation", v.ID)
\t\t}
\tcase semantic.ValidationRelationUnique:
\t\tif v.Relation == "" {
\t\t\treturn v, fmt.Errorf("relation-unique semantic validation %s requires relation", v.ID)
\t\t}
\t}
\treturn v, nil
}

'''
parser.write_text(text.replace(marker, validation_parser + marker, 1))

loader = Path("internal/workspace/loader.go")
replace_once(
    loader,
    '''\t\tr.World.Semantics[idx].Capabilities = append(r.World.Semantics[idx].Capabilities, ns.Capabilities...)
\t\tfor _, typ := range ns.Types {''',
    '''\t\tr.World.Semantics[idx].Capabilities = append(r.World.Semantics[idx].Capabilities, ns.Capabilities...)
\t\tr.World.Semantics[idx].Validations = append(r.World.Semantics[idx].Validations, ns.Validations...)
\t\tfor _, typ := range ns.Types {''',
)
replace_once(
    loader,
    '''\t\tfor _, c := range ns.Capabilities {
\t\t\tr.Sources["semantic-capability:"+string(c.ID)] = SourceLocation{File: relPath(root, path), Line: 1, Column: 1}
\t\t}
\t}
\treturn nil''',
    '''\t\tfor _, c := range ns.Capabilities {
\t\t\tr.Sources["semantic-capability:"+string(c.ID)] = SourceLocation{File: relPath(root, path), Line: 1, Column: 1}
\t\t}
\t\tfor _, validation := range ns.Validations {
\t\t\tr.Sources["semantic-validation:"+string(validation.ID)] = SourceLocation{File: relPath(root, path), Line: 1, Column: 1}
\t\t}
\t}
\treturn nil''',
)
replace_once(
    loader,
    '''\t\tsort.Slice(w.Semantics[i].Capabilities, func(a, b int) bool { return w.Semantics[i].Capabilities[a].ID < w.Semantics[i].Capabilities[b].ID })
\t}
\tsort.Slice(w.Applications,''',
    '''\t\tsort.Slice(w.Semantics[i].Capabilities, func(a, b int) bool { return w.Semantics[i].Capabilities[a].ID < w.Semantics[i].Capabilities[b].ID })
\t\tsort.Slice(w.Semantics[i].Validations, func(a, b int) bool { return w.Semantics[i].Validations[a].ID < w.Semantics[i].Validations[b].ID })
\t}
\tsort.Slice(w.Applications,''',
)

print("semantic validation parser and loader patched")
