package parser

import (
	"github.com/viant/parsly"
	"github.com/viant/sqlx/metadata/ast/expr"
	"github.com/viant/sqlx/metadata/ast/query"
	"strings"
)

func ParseQuery(SQL string) (*query.Select, error) {
	result := &query.Select{}
	cursor := parsly.NewCursor("", []byte(SQL), 0)
	return result, parseQuery(cursor, result)
}

func parseQuery(cursor *parsly.Cursor, dest *query.Select) error {
	match := cursor.MatchAfterOptional(whitespaceMatcher, selectKeywordMatcher)
	switch match.Code {
	case selectKeyword:
		match = cursor.MatchAfterOptional(whitespaceMatcher, selectionKindMatcher)
		if match.Code == selectionKind {
			dest.Kind = match.Text(cursor)
		}
		dest.List = make(query.List, 0)
		if err := parseSelectListItem(cursor, &dest.List); err != nil {
			return err
		}
		match = cursor.MatchAfterOptional(whitespaceMatcher, fromKeywordMatcher)
		switch match.Code {
		case fromKeyword:
			dest.From = query.From{}
			match = cursor.MatchAfterOptional(whitespaceMatcher, selectorMatcher, parenthesesMatcher)
			switch match.Code {
			case selectorTokenCode:
				dest.From.X = expr.NewSelector(match.Text(cursor))
			case parenthesesCode:
				dest.From.X = expr.NewRaw(match.Text(cursor))
			}

			dest.From.Alias = discoverAlias(cursor)

			match = cursor.MatchAfterOptional(whitespaceMatcher, commentBlockMatcher)
			if match.Code == commentBlock {
				dest.From.Comments = match.Text(cursor)
			}

			dest.Joins = make([]*query.Join, 0)

			match = cursor.MatchAfterOptional(whitespaceMatcher, joinToken, whereKeywordMatcher, groupByMatcher, havingKeywordMatcher, orderByKeywordMatcher, windowMatcher)
			if match.Code == parsly.EOF {
				return nil
			}
			hasMatch, err := matchPostFrom(cursor, dest, match)
			if !hasMatch && err == nil {
				err = cursor.NewError(joinToken, whereKeywordMatcher, groupByMatcher, havingKeywordMatcher, orderByKeywordMatcher, windowMatcher)
			}
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func matchPostFrom(cursor *parsly.Cursor, dest *query.Select, match *parsly.TokenMatch) (bool, error) {
	switch match.Code {
	case joinTokenCode:
		if err := appendJoin(cursor, match, dest); err != nil {
			return false, err
		}
	case whereKeyword:
		dest.Qualify = expr.NewQualify()
		if err := ParseQualify(cursor, dest.Qualify); err != nil {
			return false, err
		}
		match = cursor.MatchAfterOptional(whitespaceMatcher, groupByMatcher, havingKeywordMatcher, orderByKeywordMatcher, windowMatcher)
		return matchPostFrom(cursor, dest, match)

	case groupByKeyword:
		if err := expectIdentifiers(cursor, &dest.GroupBy); err != nil {
			return false, err
		}
		match = cursor.MatchAfterOptional(whitespaceMatcher, havingKeywordMatcher, orderByKeywordMatcher, windowMatcher)
		return matchPostFrom(cursor, dest, match)

	case havingKeyword:
		dest.Having = expr.NewQualify()
		if err := ParseQualify(cursor, dest.Having); err != nil {
			return false, err
		}
		match = cursor.MatchAfterOptional(whitespaceMatcher, orderByKeywordMatcher, windowMatcher)
		return matchPostFrom(cursor, dest, match)

	case orderByKeyword:
		if err := parseSelectListItem(cursor, &dest.OrderBy); err != nil {
			return false, err
		}

		match = cursor.MatchAfterOptional(whitespaceMatcher, windowMatcher)
		return matchPostFrom(cursor, dest, match)
	case windowTokenCode:
		matchedText := match.Text(cursor)
		dest.Window = expr.NewRaw(matchedText)
		match = cursor.MatchAfterOptional(whitespaceMatcher, intLiteralMatcher)
		if match.Code == intLiteral {
			literal := expr.NewNumericLiteral(match.Text(cursor))
			switch strings.ToLower(matchedText) {
			case "limit":
				dest.Limit = literal
			case "offset":
				dest.Offset = literal
			}
		}
	case parsly.EOF:
		return true, nil
	default:
		return false, nil
	}
	return true, nil
}

func expectExpectIdentifiers(cursor *parsly.Cursor, expect *[]string) (bool, error) {
	match := cursor.MatchAfterOptional(whitespaceMatcher, identifierMatcher)
	switch match.Code {
	case identifierCode:
		item := match.Text(cursor)
		*expect = append(*expect, item)
	default:
		return false, nil
	}

	snapshotPos := cursor.Pos
	match = cursor.MatchAfterOptional(whitespaceMatcher, nextMatcher)
	switch match.Code {
	case nextCode:
		has, err := expectExpectIdentifiers(cursor, expect)
		if err != nil {
			return false, err
		}
		if !has {
			cursor.Pos = snapshotPos
			return true, nil
		}
	}
	return true, nil
}

func expectIdentifiers(cursor *parsly.Cursor, expect *[]string) error {
	match := cursor.MatchAfterOptional(whitespaceMatcher, identifierMatcher)
	switch match.Code {
	case identifierCode:
		item := match.Text(cursor)
		*expect = append(*expect, item)
	default:
		return nil
	}

	match = cursor.MatchAfterOptional(whitespaceMatcher, nextMatcher)
	switch match.Code {
	case nextCode:
		return expectIdentifiers(cursor, expect)
	}
	return nil
}
