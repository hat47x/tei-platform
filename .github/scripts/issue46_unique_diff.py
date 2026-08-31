from pathlib import Path

path = Path("internal/database/schemadiff/diff.go")
text = path.read_text()

replacements = [
    (
        '\tChangePrimaryKey ChangeKind = "change-primary-key"\n\tAddForeignKey    ChangeKind = "add-foreign-key"',
        '\tChangePrimaryKey     ChangeKind = "change-primary-key"\n\tAddUniqueConstraint  ChangeKind = "add-unique-constraint"\n\tDropUniqueConstraint ChangeKind = "drop-unique-constraint"\n\tAddForeignKey        ChangeKind = "add-foreign-key"',
    ),
    (
        '\tPrimaryKey       []string             `json:"primaryKey,omitempty"`\n\tBeforePrimaryKey []string             `json:"beforePrimaryKey,omitempty"`\n\tForeignKey       *physical.ForeignKey `json:"foreignKey,omitempty"`',
        '\tPrimaryKey       []string                   `json:"primaryKey,omitempty"`\n\tBeforePrimaryKey []string                   `json:"beforePrimaryKey,omitempty"`\n\tUniqueConstraint *physical.UniqueConstraint `json:"uniqueConstraint,omitempty"`\n\tForeignKey       *physical.ForeignKey       `json:"foreignKey,omitempty"`',
    ),
    (
        '\t\t\tfor _, fk := range dt.ForeignKeys {',
        '\t\t\tfor _, uc := range dt.UniqueConstraints {\n\t\t\t\tuu := uc\n\t\t\t\td.Changes = append(d.Changes, Change{Kind: AddUniqueConstraint, Table: n, UniqueConstraint: &uu})\n\t\t\t}\n\t\t\tfor _, fk := range dt.ForeignKeys {',
    ),
    (
        '\tif !sameStrings(desired.PrimaryKey, actual.PrimaryKey) {\n\t\tout = append(out, Change{Kind: ChangePrimaryKey, Table: name, PrimaryKey: append([]string(nil), desired.PrimaryKey...), BeforePrimaryKey: append([]string(nil), actual.PrimaryKey...)})\n\t}\n\tout = append(out, diffForeignKeys(name, desired.ForeignKeys, actual.ForeignKeys)...)',
        '\tif !sameStrings(desired.PrimaryKey, actual.PrimaryKey) {\n\t\tout = append(out, Change{Kind: ChangePrimaryKey, Table: name, PrimaryKey: append([]string(nil), desired.PrimaryKey...), BeforePrimaryKey: append([]string(nil), actual.PrimaryKey...)})\n\t}\n\tout = append(out, diffUniqueConstraints(name, desired.UniqueConstraints, actual.UniqueConstraints)...)\n\tout = append(out, diffForeignKeys(name, desired.ForeignKeys, actual.ForeignKeys)...)',
    ),
]

for old, new in replacements:
    count = text.count(old)
    if count != 1:
        raise SystemExit(f"expected one replacement, found {count}: {old[:80]!r}")
    text = text.replace(old, new, 1)

marker = "func diffForeignKeys(table string, desired, actual []physical.ForeignKey) []Change {"
if text.count(marker) != 1:
    raise SystemExit("diffForeignKeys marker mismatch")

helper = '''func diffUniqueConstraints(table string, desired, actual []physical.UniqueConstraint) []Change {
\tdg, ag := uniqueConstraintGroups(desired), uniqueConstraintGroups(actual)
\tkeys := make([]string, 0, len(dg)+len(ag))
\tseen := map[string]struct{}{}
\tfor k := range dg {
\t\tseen[k] = struct{}{}
\t\tkeys = append(keys, k)
\t}
\tfor k := range ag {
\t\tif _, ok := seen[k]; !ok {
\t\t\tkeys = append(keys, k)
\t\t}
\t}
\tsort.Strings(keys)
\tvar drops, adds []Change
\tfor _, k := range keys {
\t\td, a := dg[k], ag[k]
\t\tmatched := len(d)
\t\tif len(a) < matched {
\t\t\tmatched = len(a)
\t\t}
\t\tfor i := matched; i < len(a); i++ {
\t\t\tuc := a[i]
\t\t\tdrops = append(drops, Change{Kind: DropUniqueConstraint, Table: table, UniqueConstraint: &uc})
\t\t}
\t\tfor i := matched; i < len(d); i++ {
\t\t\tuc := d[i]
\t\t\tadds = append(adds, Change{Kind: AddUniqueConstraint, Table: table, UniqueConstraint: &uc})
\t\t}
\t}
\treturn append(drops, adds...)
}

func uniqueConstraintGroups(constraints []physical.UniqueConstraint) map[string][]physical.UniqueConstraint {
\tout := map[string][]physical.UniqueConstraint{}
\tfor _, constraint := range constraints {
\t\tkey := uniqueConstraintStructuralKey(constraint)
\t\tout[key] = append(out[key], constraint)
\t}
\tfor key := range out {
\t\tsort.Slice(out[key], func(i, j int) bool { return norm(out[key][i].Name) < norm(out[key][j].Name) })
\t}
\treturn out
}

func uniqueConstraintStructuralKey(constraint physical.UniqueConstraint) string {
\treturn strings.Join(normalizedStrings(constraint.Columns), "\\x1f")
}

'''

text = text.replace(marker, helper + marker, 1)
path.write_text(text)
