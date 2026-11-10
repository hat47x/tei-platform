package syntax

import (
	"fmt"
	"strconv"

	"tei/internal/application"
	"tei/internal/core"
)

// ParseApplicationPatternは、Application配下の独立したPattern定義を読み取る。
// Pattern Kindごとの必須条件は、実Referenceで確認した意味に限ってここで構文契約として固定する。
func (p *Parser) ParseApplicationPattern() (application.Pattern, error) {
	if _, err := p.expectText("pattern"); err != nil {
		return application.Pattern{}, err
	}
	idTok, err := p.expect(TokenIdent)
	if err != nil {
		return application.Pattern{}, err
	}
	id, err := core.ParseCanonicalID(idTok.Text)
	if err != nil {
		return application.Pattern{}, p.errAt(idTok, err.Error())
	}
	kindTok, err := p.expect(TokenIdent)
	if err != nil {
		return application.Pattern{}, err
	}
	out := application.Pattern{ID: id, Kind: application.PatternKind(kindTok.Text)}
	if !out.Kind.Valid() {
		return out, p.errAt(kindTok, fmt.Sprintf("unsupported application pattern kind %q", kindTok.Text))
	}
	if _, err := p.expect(TokenLBrace); err != nil {
		return out, err
	}

	var hasData, hasRelation, hasQuery, hasCapability, hasDatasource, hasContractQuery, hasInteraction bool
	for {
		t, err := p.peek()
		if err != nil {
			return out, err
		}
		if t.Kind == TokenRBrace {
			_, _ = p.next()
			break
		}
		switch t.Text {
		case "data":
			if hasData {
				return out, p.errAt(t, "data is declared more than once")
			}
			hasData = true
			_, _ = p.next()
			v, err := p.expect(TokenIdent)
			if err != nil {
				return out, err
			}
			out.Data, err = core.ParseCanonicalID(v.Text)
			if err != nil {
				return out, p.errAt(v, err.Error())
			}
		case "relation":
			if hasRelation {
				return out, p.errAt(t, "relation is declared more than once")
			}
			hasRelation = true
			_, _ = p.next()
			v, err := p.expect(TokenIdent)
			if err != nil {
				return out, err
			}
			out.Relation, err = core.ParseCanonicalID(v.Text)
			if err != nil {
				return out, p.errAt(v, err.Error())
			}
		case "query":
			if hasQuery {
				return out, p.errAt(t, "query is declared more than once")
			}
			hasQuery = true
			_, _ = p.next()
			v, err := p.expect(TokenIdent)
			if err != nil {
				return out, err
			}
			out.Query, err = core.ParseCanonicalID(v.Text)
			if err != nil {
				return out, p.errAt(v, err.Error())
			}
		case "aggregate":
			_, _ = p.next()
			nameTok, err := p.expect(TokenIdent)
			if err != nil {
				return out, err
			}
			attrTok, err := p.expect(TokenIdent)
			if err != nil {
				return out, err
			}
			attribute, err := core.ParseCanonicalID(attrTok.Text)
			if err != nil {
				return out, p.errAt(attrTok, err.Error())
			}
			fnTok, err := p.expect(TokenIdent)
			if err != nil {
				return out, err
			}
			out.Aggregates = append(out.Aggregates, application.Aggregate{Name: nameTok.Text, Attribute: attribute, Function: fnTok.Text})
		case "filter":
			_, _ = p.next()
			attrTok, err := p.expect(TokenIdent)
			if err != nil {
				return out, err
			}
			attribute, err := core.ParseCanonicalID(attrTok.Text)
			if err != nil {
				return out, p.errAt(attrTok, err.Error())
			}
			if _, err := p.expect(TokenLBrace); err != nil {
				return out, err
			}
			operators := []string{}
			for {
				entry, err := p.peek()
				if err != nil {
					return out, err
				}
				if entry.Kind == TokenRBrace {
					_, _ = p.next()
					break
				}
				opTok, err := p.expect(TokenIdent)
				if err != nil {
					return out, err
				}
				operators = append(operators, opTok.Text)
			}
			filter := application.Filter{Attribute: attribute, Operators: operators}
			if nt, _ := p.peek(); nt.Text == "parameter" {
				_, _ = p.next()
				paramTok, err := p.expect(TokenIdent)
				if err != nil {
					return out, err
				}
				filter.Parameter = paramTok.Text
			}
			if nt, _ := p.peek(); nt.Text == "operator-parameter" {
				_, _ = p.next()
				opParamTok, err := p.expect(TokenIdent)
				if err != nil {
					return out, err
				}
				filter.OperatorParameter = opParamTok.Text
			}
			out.Filters = append(out.Filters, filter)
		case "sort":
			_, _ = p.next()
			attrTok, err := p.expect(TokenIdent)
			if err != nil {
				return out, err
			}
			attribute, err := core.ParseCanonicalID(attrTok.Text)
			if err != nil {
				return out, p.errAt(attrTok, err.Error())
			}
			dirTok, err := p.expect(TokenIdent)
			if err != nil {
				return out, err
			}
			out.Sorts = append(out.Sorts, application.Sort{Attribute: attribute, Direction: dirTok.Text})
		case "page-size":
			_, _ = p.next()
			sizeTok, err := p.expect(TokenIdent)
			if err != nil {
				return out, err
			}
			size, err := strconv.Atoi(sizeTok.Text)
			if err != nil {
				return out, p.errAt(sizeTok, "page-size requires an integer")
			}
			out.PageSize = size
			if nt, _ := p.peek(); nt.Text == "parameter" {
				_, _ = p.next()
				sizeParam, err := p.expect(TokenIdent)
				if err != nil {
					return out, err
				}
				out.PageSizeParameter = sizeParam.Text
				if nt, _ := p.peek(); nt.Text == "offset" {
					_, _ = p.next()
					offsetParam, err := p.expect(TokenIdent)
					if err != nil {
						return out, err
					}
					out.PageOffsetParameter = offsetParam.Text
				}
			}
		case "print-template":
			_, _ = p.next()
			templateTok, err := p.expect(TokenIdent)
			if err != nil {
				return out, err
			}
			out.PrintTemplate = templateTok.Text
		case "layout":
			_, _ = p.next()
			if _, err := p.expect(TokenLBrace); err != nil {
				return out, err
			}
			layout := &application.Layout{}
			for {
				entry, err := p.peek()
				if err != nil {
					return out, err
				}
				if entry.Kind == TokenRBrace {
					_, _ = p.next()
					break
				}
				if entry.Text != "order" {
					return out, p.errAt(entry, "expected order in layout")
				}
				_, _ = p.next()
				attrTok, err := p.expect(TokenIdent)
				if err != nil {
					return out, err
				}
				attribute, err := core.ParseCanonicalID(attrTok.Text)
				if err != nil {
					return out, p.errAt(attrTok, err.Error())
				}
				layout.Order = append(layout.Order, attribute)
			}
			out.Layout = layout
		case "page-break":
			_, _ = p.next()
			rowsTok, err := p.expect(TokenIdent)
			if err != nil {
				return out, err
			}
			rows, err := strconv.Atoi(rowsTok.Text)
			if err != nil {
				return out, p.errAt(rowsTok, "page-break requires an integer")
			}
			pageBreak := &application.PageBreak{Rows: rows}
			if nt, _ := p.peek(); nt.Text == "repeat-header" {
				_, _ = p.next()
				pageBreak.RepeatHeader = true
			}
			out.PageBreak = pageBreak
		case "capability":
			if hasCapability {
				return out, p.errAt(t, "capability is declared more than once")
			}
			hasCapability = true
			_, _ = p.next()
			v, err := p.expect(TokenIdent)
			if err != nil {
				return out, err
			}
			out.Capability, err = core.ParseCanonicalID(v.Text)
			if err != nil {
				return out, p.errAt(v, err.Error())
			}
		case "datasource":
			if hasDatasource {
				return out, p.errAt(t, "datasource is declared more than once")
			}
			hasDatasource = true
			_, _ = p.next()
			v, err := p.expect(TokenIdent)
			if err != nil {
				return out, err
			}
			out.Datasource, err = core.ParseLocalName(v.Text)
			if err != nil {
				return out, p.errAt(v, err.Error())
			}
		case "contract-query":
			if hasContractQuery {
				return out, p.errAt(t, "contract-queryは一つのPatternで一度だけ定義できます")
			}
			hasContractQuery = true
			invocation, err := p.parseReferenceContractQuery()
			if err != nil {
				return out, err
			}
			out.ContractQuery = invocation
		case "interaction":
			if hasInteraction {
				return out, p.errAt(t, "interactionは一つのPatternで一度だけ定義できます")
			}
			hasInteraction = true
			_, _ = p.next()
			interaction, err := p.parsePatternInteraction()
			if err != nil {
				return out, err
			}
			out.Interaction = interaction
		default:
			return out, p.errAt(t, "expected data, relation, query, capability, datasource, contract-query, or interaction")
		}
	}

	switch out.Kind {
	case application.PatternBrowse:
		if out.Data == "" {
			return out, fmt.Errorf("browse pattern %s requires data", out.ID)
		}
		if out.Query == "" {
			return out, fmt.Errorf("browse pattern %s requires query", out.ID)
		}
		if out.Relation != "" {
			return out, fmt.Errorf("browse pattern %s does not accept relation", out.ID)
		}
		if out.Capability != "" {
			return out, fmt.Errorf("browse pattern %s does not accept capability", out.ID)
		}
		if out.ContractQuery != nil {
			return out, fmt.Errorf("browse pattern %s does not accept contract-query", out.ID)
		}
	case application.PatternRecord:
		if out.Data == "" {
			return out, fmt.Errorf("record pattern %s requires data", out.ID)
		}
		if out.Relation != "" {
			return out, fmt.Errorf("record pattern %s does not accept relation", out.ID)
		}
		if out.Query != "" {
			return out, fmt.Errorf("record pattern %s does not accept query", out.ID)
		}
		if out.Capability != "" {
			return out, fmt.Errorf("record pattern %s does not accept capability", out.ID)
		}
		if out.Datasource != "" {
			return out, fmt.Errorf("record pattern %s does not accept datasource", out.ID)
		}
		if out.ContractQuery != nil {
			return out, fmt.Errorf("record pattern %s does not accept contract-query", out.ID)
		}
	case application.PatternReference:
		if out.Relation == "" {
			return out, fmt.Errorf("reference pattern %s requires relation", out.ID)
		}
		if out.Datasource == "" {
			return out, fmt.Errorf("reference pattern %s requires datasource", out.ID)
		}
		if out.Data != "" {
			return out, fmt.Errorf("reference pattern %s does not accept data", out.ID)
		}
		if out.Query != "" {
			return out, fmt.Errorf("reference pattern %s does not accept query", out.ID)
		}
		if out.Capability != "" {
			return out, fmt.Errorf("reference pattern %s does not accept capability", out.ID)
		}
	case application.PatternCommand:
		if out.Capability == "" {
			return out, fmt.Errorf("command pattern %s requires capability", out.ID)
		}
		if out.Data != "" {
			return out, fmt.Errorf("command pattern %s does not accept data", out.ID)
		}
		if out.Relation != "" {
			return out, fmt.Errorf("command pattern %s does not accept relation", out.ID)
		}
		if out.Query != "" {
			return out, fmt.Errorf("command pattern %s does not accept query", out.ID)
		}
		if out.Datasource != "" {
			return out, fmt.Errorf("command pattern %s does not accept datasource", out.ID)
		}
		if out.ContractQuery != nil {
			return out, fmt.Errorf("command pattern %s does not accept contract-query", out.ID)
		}
	case application.PatternHeaderDetail:
		if out.Relation == "" {
			return out, fmt.Errorf("header-detail pattern %s requires relation", out.ID)
		}
		if out.Data != "" {
			return out, fmt.Errorf("header-detail pattern %s does not accept data", out.ID)
		}
		if out.Query != "" {
			return out, fmt.Errorf("header-detail pattern %s does not accept query", out.ID)
		}
		if out.Capability != "" {
			return out, fmt.Errorf("header-detail pattern %s does not accept capability", out.ID)
		}
		if out.Datasource != "" {
			return out, fmt.Errorf("header-detail pattern %s does not accept datasource", out.ID)
		}
		if out.ContractQuery != nil {
			return out, fmt.Errorf("header-detail pattern %s does not accept contract-query", out.ID)
		}
	}
	if err := p.expectEOF(); err != nil {
		return out, err
	}
	return out, nil
}

// parsePatternInteractionは、Pattern内のinteractionブロックを読み取る。
// 表示上の幅・部品・配置はここへ入れず、項目の並び・参照入力・遷移だけを構文として固定する。
func (p *Parser) parsePatternInteraction() (*application.Interaction, error) {
	if _, err := p.expect(TokenLBrace); err != nil {
		return nil, err
	}
	out := &application.Interaction{}
	for {
		t, err := p.peek()
		if err != nil {
			return nil, err
		}
		if t.Kind == TokenRBrace {
			_, _ = p.next()
			break
		}
		switch t.Text {
		case "item":
			_, _ = p.next()
			attrTok, err := p.expect(TokenIdent)
			if err != nil {
				return nil, err
			}
			attr, err := core.ParseCanonicalID(attrTok.Text)
			if err != nil {
				return nil, p.errAt(attrTok, err.Error())
			}
			item := application.InteractionItem{Attribute: attr}
			next, err := p.peek()
			if err != nil {
				return nil, err
			}
			if next.Kind == TokenIdent && next.Text == "lookup" {
				_, _ = p.next()
				patternTok, err := p.expect(TokenIdent)
				if err != nil {
					return nil, err
				}
				pattern, err := core.ParseCanonicalID(patternTok.Text)
				if err != nil {
					return nil, p.errAt(patternTok, err.Error())
				}
				targetTok, err := p.expect(TokenIdent)
				if err != nil {
					return nil, err
				}
				target, err := core.ParseCanonicalID(targetTok.Text)
				if err != nil {
					return nil, p.errAt(targetTok, err.Error())
				}
				item.Lookup = &application.InteractionLookup{Pattern: pattern, TargetAttribute: target}
			}
			out.Items = append(out.Items, item)
		case "navigate":
			_, _ = p.next()
			if _, err := p.expectText("from"); err != nil {
				return nil, err
			}
			fromTok, err := p.expect(TokenIdent)
			if err != nil {
				return nil, err
			}
			from, err := core.ParseCanonicalID(fromTok.Text)
			if err != nil {
				return nil, p.errAt(fromTok, err.Error())
			}
			out.Navigations = append(out.Navigations, application.InteractionNavigation{From: from})
		case "command":
			_, _ = p.next()
			capabilityTok, err := p.expect(TokenIdent)
			if err != nil {
				return nil, err
			}
			capability, err := core.ParseCanonicalID(capabilityTok.Text)
			if err != nil {
				return nil, p.errAt(capabilityTok, err.Error())
			}
			command := application.InteractionCommand{Capability: capability}
			if nt, _ := p.peek(); nt.Kind == TokenLBrace {
				_, _ = p.next()
				for {
					entry, err := p.peek()
					if err != nil {
						return nil, err
					}
					if entry.Kind == TokenRBrace {
						_, _ = p.next()
						break
					}
					if entry.Text != "input" {
						return nil, p.errAt(entry, "expected input in command binding")
					}
					_, _ = p.next()
					nameTok, err := p.expect(TokenIdent)
					if err != nil {
						return nil, err
					}
					attrTok, err := p.expect(TokenIdent)
					if err != nil {
						return nil, err
					}
					attribute, err := core.ParseCanonicalID(attrTok.Text)
					if err != nil {
						return nil, p.errAt(attrTok, err.Error())
					}
					command.Inputs = append(command.Inputs, application.CommandInputBinding{Name: nameTok.Text, Attribute: attribute})
				}
			}
			out.Commands = append(out.Commands, command)
		case "hook":
			_, _ = p.next()
			phaseTok, err := p.expect(TokenIdent)
			if err != nil {
				return nil, err
			}
			hook := application.Hook{Phase: phaseTok.Text}
			if _, err := p.expect(TokenLBrace); err != nil {
				return nil, err
			}
			for {
				entry, err := p.peek()
				if err != nil {
					return nil, err
				}
				if entry.Kind == TokenRBrace {
					_, _ = p.next()
					break
				}
				switch entry.Text {
				case "reads", "produces":
					_, _ = p.next()
					attrTok, err := p.expect(TokenIdent)
					if err != nil {
						return nil, err
					}
					attribute, err := core.ParseCanonicalID(attrTok.Text)
					if err != nil {
						return nil, p.errAt(attrTok, err.Error())
					}
					if entry.Text == "reads" {
						hook.Reads = append(hook.Reads, attribute)
					} else {
						hook.Produces = append(hook.Produces, attribute)
					}
				case "uses":
					_, _ = p.next()
					capabilityTok, err := p.expect(TokenIdent)
					if err != nil {
						return nil, err
					}
					capability, err := core.ParseCanonicalID(capabilityTok.Text)
					if err != nil {
						return nil, p.errAt(capabilityTok, err.Error())
					}
					hook.Uses = append(hook.Uses, capability)
				case "on-failure":
					_, _ = p.next()
					kindTok, err := p.expect(TokenIdent)
					if err != nil {
						return nil, err
					}
					hook.Failure = kindTok.Text
				case "scope":
					_, _ = p.next()
					scopeTok, err := p.expect(TokenIdent)
					if err != nil {
						return nil, err
					}
					switch scopeTok.Text {
					case "row", "record", "item", "header":
					default:
						return nil, p.errAt(scopeTok, "scope must be row, record, item, or header")
					}
					hook.Scope = scopeTok.Text
				default:
					return nil, p.errAt(entry, "expected scope, reads, uses, produces, or on-failure in hook")
				}
			}
			out.Hooks = append(out.Hooks, hook)
		default:
			return nil, p.errAt(t, "expected item, navigate, or command in interaction")
		}
	}
	if len(out.Items) == 0 && len(out.Navigations) == 0 && len(out.Commands) == 0 && len(out.Hooks) == 0 {
		return nil, fmt.Errorf("interaction requires at least one item, navigation, command, or hook")
	}
	return out, nil
}
