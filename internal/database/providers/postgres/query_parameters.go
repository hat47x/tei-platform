package postgres

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// BoundQueryは、PostgreSQLで実行できるpositional placeholderへ変換したSQLと、
// そのplaceholder順に並べた実行時引数を保持する。
//
// ParameterNamesは値を含まず、どのnamed parameterがどの順序で束縛されたかだけを示す。
// 実行時の値を通常のEvidenceへ保存する用途には使わない。
type BoundQuery struct {
	SQL            string
	Args           []any
	ParameterNames []string
}

// BindNamedParametersは、TEIのnative Queryで使う:name形式のparameterを、
// PostgreSQLの$1, $2, ...形式へ変換する。
//
// SQL文字列、quoted identifier、line/block comment、dollar-quoted bodyの内部は置換しない。
// PostgreSQLの:: castもnamed parameterとは扱わない。同じparameterが複数回現れた場合は、
// 最初に割り当てたplaceholderを再利用する。
//
// この関数はPostgreSQL provider固有のruntime処理であり、Driver CoreへSQL書換え規則を持ち込まない。
func BindNamedParameters(source string, parameters map[string]any) (BoundQuery, error) {
	if parameters == nil {
		parameters = map[string]any{}
	}

	var out strings.Builder
	out.Grow(len(source))

	positions := map[string]int{}
	used := map[string]struct{}{}
	args := make([]any, 0, len(parameters))
	names := make([]string, 0, len(parameters))

	for i := 0; i < len(source); {
		switch source[i] {
		case '\'':
			next, err := copySingleQuoted(&out, source, i)
			if err != nil {
				return BoundQuery{}, err
			}
			i = next
		case '"':
			next, err := copyDoubleQuoted(&out, source, i)
			if err != nil {
				return BoundQuery{}, err
			}
			i = next
		case '-':
			if i+1 < len(source) && source[i+1] == '-' {
				i = copyLineComment(&out, source, i)
				continue
			}
			out.WriteByte(source[i])
			i++
		case '/':
			if i+1 < len(source) && source[i+1] == '*' {
				next, err := copyBlockComment(&out, source, i)
				if err != nil {
					return BoundQuery{}, err
				}
				i = next
				continue
			}
			out.WriteByte(source[i])
			i++
		case '$':
			if i+1 < len(source) && isDigit(source[i+1]) {
				return BoundQuery{}, fmt.Errorf("PostgreSQL native Queryではnamed parameterと既存のpositional placeholderを混在できません: %s", positionalPlaceholderAt(source, i))
			}
			if delimiter, ok := dollarQuoteDelimiter(source, i); ok {
				next, err := copyDollarQuoted(&out, source, i, delimiter)
				if err != nil {
					return BoundQuery{}, err
				}
				i = next
				continue
			}
			out.WriteByte(source[i])
			i++
		case ':':
			// PostgreSQLの:: castをnamed parameterと誤認しない。
			if (i > 0 && source[i-1] == ':') || (i+1 < len(source) && source[i+1] == ':') {
				out.WriteByte(source[i])
				i++
				continue
			}
			if i+1 >= len(source) || !isParameterStart(source[i+1]) {
				out.WriteByte(source[i])
				i++
				continue
			}

			end := i + 2
			for end < len(source) && isParameterContinue(source[end]) {
				end++
			}
			name := source[i+1 : end]
			value, ok := parameters[name]
			if !ok {
				return BoundQuery{}, fmt.Errorf("PostgreSQL native Queryのparameter %qに値がありません", name)
			}

			position, exists := positions[name]
			if !exists {
				position = len(args) + 1
				positions[name] = position
				args = append(args, value)
				names = append(names, name)
			}
			used[name] = struct{}{}
			out.WriteByte('$')
			out.WriteString(strconv.Itoa(position))
			i = end
		default:
			out.WriteByte(source[i])
			i++
		}
	}

	if len(used) != len(parameters) {
		unused := make([]string, 0, len(parameters)-len(used))
		for name := range parameters {
			if _, ok := used[name]; !ok {
				unused = append(unused, name)
			}
		}
		sort.Strings(unused)
		return BoundQuery{}, fmt.Errorf("PostgreSQL native Queryで使用されていないparameterがあります: %s", strings.Join(unused, ", "))
	}

	return BoundQuery{
		SQL:            out.String(),
		Args:           args,
		ParameterNames: names,
	}, nil
}

func copySingleQuoted(out *strings.Builder, source string, start int) (int, error) {
	out.WriteByte(source[start])
	for i := start + 1; i < len(source); i++ {
		out.WriteByte(source[i])
		if source[i] == '\\' && i+1 < len(source) {
			// E''形式を含め、backslashでescapeされた次のbyteを文字列内部として保守的に扱う。
			i++
			out.WriteByte(source[i])
			continue
		}
		if source[i] != '\'' {
			continue
		}
		if i+1 < len(source) && source[i+1] == '\'' {
			i++
			out.WriteByte(source[i])
			continue
		}
		return i + 1, nil
	}
	return 0, fmt.Errorf("PostgreSQL native Queryの文字列リテラルが閉じていません")
}

func copyDoubleQuoted(out *strings.Builder, source string, start int) (int, error) {
	out.WriteByte(source[start])
	for i := start + 1; i < len(source); i++ {
		out.WriteByte(source[i])
		if source[i] != '"' {
			continue
		}
		if i+1 < len(source) && source[i+1] == '"' {
			i++
			out.WriteByte(source[i])
			continue
		}
		return i + 1, nil
	}
	return 0, fmt.Errorf("PostgreSQL native Queryのquoted identifierが閉じていません")
}

func copyLineComment(out *strings.Builder, source string, start int) int {
	i := start
	for i < len(source) {
		out.WriteByte(source[i])
		i++
		if source[i-1] == '\n' {
			break
		}
	}
	return i
}

func copyBlockComment(out *strings.Builder, source string, start int) (int, error) {
	depth := 0
	for i := start; i < len(source); {
		if i+1 < len(source) && source[i] == '/' && source[i+1] == '*' {
			depth++
			out.WriteString("/*")
			i += 2
			continue
		}
		if i+1 < len(source) && source[i] == '*' && source[i+1] == '/' {
			depth--
			out.WriteString("*/")
			i += 2
			if depth == 0 {
				return i, nil
			}
			continue
		}
		out.WriteByte(source[i])
		i++
	}
	return 0, fmt.Errorf("PostgreSQL native Queryのblock commentが閉じていません")
}

func dollarQuoteDelimiter(source string, start int) (string, bool) {
	if start >= len(source) || source[start] != '$' || start+1 >= len(source) {
		return "", false
	}
	if source[start+1] == '$' {
		return "$$", true
	}
	if !isParameterStart(source[start+1]) {
		return "", false
	}
	end := start + 2
	for end < len(source) && isParameterContinue(source[end]) {
		end++
	}
	if end < len(source) && source[end] == '$' {
		return source[start : end+1], true
	}
	return "", false
}

func copyDollarQuoted(out *strings.Builder, source string, start int, delimiter string) (int, error) {
	bodyStart := start + len(delimiter)
	endOffset := strings.Index(source[bodyStart:], delimiter)
	if endOffset < 0 {
		return 0, fmt.Errorf("PostgreSQL native Queryのdollar-quoted body %sが閉じていません", delimiter)
	}
	end := bodyStart + endOffset + len(delimiter)
	out.WriteString(source[start:end])
	return end, nil
}

func positionalPlaceholderAt(source string, start int) string {
	end := start + 1
	for end < len(source) && isDigit(source[end]) {
		end++
	}
	return source[start:end]
}

func isParameterStart(ch byte) bool {
	return ch == '_' || ch >= 'A' && ch <= 'Z' || ch >= 'a' && ch <= 'z'
}

func isParameterContinue(ch byte) bool {
	return isParameterStart(ch) || isDigit(ch)
}

func isDigit(ch byte) bool {
	return ch >= '0' && ch <= '9'
}
