package parser

import (
	"github.com/viant/parsly"
	"github.com/viant/sqlx/metadata/ast/expr"
)

func ParseLiteral(cursor *parsly.Cursor) (*expr.Literal, error) {
	return parseLiteral(cursor, true)
}

func TryParseLiteral(cursor *parsly.Cursor) (*expr.Literal, error) {
	return parseLiteral(cursor, false)
}

var literalTokens = []*parsly.Token{
	asKeywordToken,
	nextToken,
	nullKeywordToken,
	boolLiteralToken,
	doubleQuotedStringLiteralToken,
	singleQuotedStringLiteralToken,
	intLiteralToken,
	numericLiteralToken,
}

func parseLiteral(cursor *parsly.Cursor, shallRaiseInvalidToken bool) (*expr.Literal, error) {
	match := cursor.MatchAfterOptional(whitespaceToken, literalTokens...)
	switch match.Code {
	case asKeyword, nextCode:
		cursor.Pos -= match.Size
		return nil, nil
	case nullKeyword:
		return expr.NewNullLiteral(match.Text(cursor)), nil
	case singleQuotedStringLiteral, doubleQuotedStringLiteral:
		return expr.NewStringLiteral(match.Text(cursor)), nil
	case boolLiteral:
		return expr.NewBoolLiteral(match.Text(cursor)), nil
	case intLiteral:
		return expr.NewIntLiteral(match.Text(cursor)), nil
	case numericLiteral:
		return expr.NewNumericLiteral(match.Text(cursor)), nil
	case parsly.EOF:
		return nil, nil
	case parsly.Invalid:
		if shallRaiseInvalidToken {
			return nil, cursor.NewError(literalTokens...)
		}
	}
	return nil, nil
}
