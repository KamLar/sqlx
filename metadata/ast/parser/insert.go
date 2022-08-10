package parser

import (
	"github.com/viant/parsly"
	"github.com/viant/sqlx/metadata/ast/expr"
	"github.com/viant/sqlx/metadata/ast/insert"
	"strings"
)

func ParseInsert(SQL string) (*insert.Statement, error) {
	result := &insert.Statement{}
	cursor := parsly.NewCursor("", []byte(SQL), 0)
	return result, parseInsert(cursor, result)
}

func parseInsert(cursor *parsly.Cursor, stmt *insert.Statement) error {
	match := cursor.MatchAfterOptional(whitespaceMatcher, insertIntoKeywordMatcher)
	switch match.Code {
	case insertIntoKeyword:
		match = cursor.MatchAfterOptional(whitespaceMatcher, selectorMatcher)
		switch match.Code {
		case selectorTokenCode:
			sel := match.Text(cursor)
			stmt.Target.X = expr.NewSelector(sel)
			match = cursor.MatchAfterOptional(whitespaceMatcher, commentBlockMatcher)
			if match.Code == commentBlock {
				stmt.Target.Comments = match.Text(cursor)
			}
			match = cursor.MatchAfterOptional(whitespaceMatcher, parenthesesMatcher)
			if match.Code == parenthesesCode {
				matched := match.Text(cursor)
				stmt.Columns = extractColumnNames(matched[1 : len(matched)-1])
			}

			match = cursor.MatchAfterOptional(whitespaceMatcher, insertValesKeywordMatcher)
			if match.Code != insertValuesKeyword {
				return cursor.NewError(insertValesKeywordMatcher)
			}
			offset := cursor.Pos
			match = cursor.MatchAfterOptional(whitespaceMatcher, parenthesesMatcher)
			if match.Code != parenthesesCode {
				return cursor.NewError(parenthesesMatcher)
			}
			matched := match.Text(cursor)
			var err error
			if stmt.Values, err = parseInsertValues(matched[1:len(matched)-1], offset); err != nil {
				return err
			}

		}
	default:
		return cursor.NewError(insertIntoKeywordMatcher)
	}
	return nil
}

func parseInsertValues(encodedValues string, offset int) ([]*insert.Value, error) {
	cursor := parsly.NewCursor("", []byte(encodedValues), offset)
	var values []*insert.Value
	if err := expectInsertValue(cursor, &values); err != nil {
		return nil, err
	}
	return values, nil
}

func expectInsertValue(cursor *parsly.Cursor, values *[]*insert.Value) error {
	operand, err := expectOperand(cursor)
	if err != nil || operand == nil {
		return err
	}
	*values = append(*values, &insert.Value{Expr: operand})

	match := cursor.MatchAfterOptional(whitespaceMatcher, nextMatcher)
	if match.Code != nextCode {
		return nil
	}
	return expectInsertValue(cursor, values)
}

func extractColumnNames(separatedColumns string) []string {
	var result = make([]string, strings.Count(separatedColumns, ",")+1)
	var index = 0
	for _, column := range strings.Split(separatedColumns, ",") {
		column = strings.TrimSpace(column)
		if column == "" {
			continue
		}
		result[index] = column
		index++
	}
	return result[:index]
}
