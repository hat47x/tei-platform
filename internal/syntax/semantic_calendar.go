package syntax

import (
	"strconv"
	"strings"

	"tei/internal/core"
	"tei/internal/semantic"
)

// parseSemanticCalendar reads a business calendar declaration. It records when
// the fiscal year starts, which weekday starts the week, and which data holds
// the exceptions; it does not compute a date or decide a period boundary.
func (p *Parser) parseSemanticCalendar(namespace core.CanonicalID) (semantic.Calendar, error) {
	_, _ = p.next() // calendar
	symTok, err := p.expect(TokenIdent)
	if err != nil {
		return semantic.Calendar{}, err
	}
	var id core.CanonicalID
	if strings.Contains(symTok.Text, ".") {
		id, err = core.ParseCanonicalID(symTok.Text)
		if err != nil {
			return semantic.Calendar{}, p.errAt(symTok, err.Error())
		}
		if !strings.HasPrefix(string(id), string(namespace)+".") {
			return semantic.Calendar{}, p.errAt(symTok, "calendar canonical id must belong to the declared semantic namespace")
		}
	} else {
		seg, err := core.SymbolToSegment(symTok.Text)
		if err != nil {
			return semantic.Calendar{}, p.errAt(symTok, err.Error())
		}
		id, _ = core.ParseCanonicalID(string(namespace) + "." + seg)
	}
	calendar := semantic.Calendar{ID: id}
	if _, err := p.expect(TokenLBrace); err != nil {
		return calendar, err
	}
	for {
		t, err := p.peek()
		if err != nil {
			return calendar, err
		}
		if t.Kind == TokenRBrace {
			_, _ = p.next()
			break
		}
		switch t.Text {
		case "fiscal-year":
			_, _ = p.next()
			if _, err := p.expectText("starts"); err != nil {
				return calendar, err
			}
			if _, err := p.expectText("month"); err != nil {
				return calendar, err
			}
			monthTok, err := p.expect(TokenIdent)
			if err != nil {
				return calendar, err
			}
			month, convErr := strconv.Atoi(monthTok.Text)
			if convErr != nil {
				return calendar, p.errAt(monthTok, "fiscal-year starts month requires an integer")
			}
			calendar.FiscalYearStartMonth = month
		case "week":
			_, _ = p.next()
			if _, err := p.expectText("starts"); err != nil {
				return calendar, err
			}
			dayTok, err := p.expect(TokenIdent)
			if err != nil {
				return calendar, err
			}
			calendar.WeekStarts = dayTok.Text
		case "business-day":
			_, _ = p.next()
			if _, err := p.expectText("excludes"); err != nil {
				return calendar, err
			}
			value, err := p.expectCanonicalReference()
			if err != nil {
				return calendar, err
			}
			calendar.BusinessDayExcludes = value
		default:
			return calendar, p.errAt(t, "expected fiscal-year, week, or business-day")
		}
	}
	return calendar, nil
}
