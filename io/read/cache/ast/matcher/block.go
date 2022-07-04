package matcher

import "github.com/viant/parsly"

type Block struct {
	escape byte
	begin  byte
	end    byte
}

//TokenMatch matches quoted characters
func (m *Block) Match(cursor *parsly.Cursor) int {
	var matched = 0
	input := cursor.Input
	inputSize := len(input)
	pos := cursor.Pos
	value := input[pos]
	if value != m.begin {
		return 0
	}
	depth := 1
	matched++

	var inQuote byte
	var isEscaped bool
	for i := pos + matched; i < inputSize; i++ {
		value = input[i]
		isInQuote := inQuote != 0
		matched++

		if isEscaped {
			isEscaped = false
			continue
		}

		isEscaped = value == m.escape
		switch value {
		case m.begin:
			if m.begin == m.end {
				return matched
			}

			if isInQuote {
				continue
			}
			depth++

		case m.end:
			if isInQuote {
				continue
			}
			depth--
			if depth == 0 {
				return matched
			}
		}
	}
	return 0
}

func NewBlock(begin, end, escape byte) *Block {
	return &Block{
		escape: escape,
		begin:  begin,
		end:    end,
	}
}
