package tokenizer

import (
	"io"

	"github.com/pixality-inc/golang-core/errors"
	"github.com/pixality-inc/golang-core/iterator"
)

var ErrUnknownToken = errors.New("tokenizer.unknown_token", "unknown token")

type State struct {
	source   io.Reader
	iterator iterator.PeekableConsumable[byte]
	err      error
	offset   uint64
	column   uint64
	line     uint64
}

func NewState(source io.Reader) *State {
	return &State{
		source:   source,
		iterator: iterator.NewPeekableConsumable(iterator.NewReaderIterator(source)),
		err:      nil,
		offset:   0,
		column:   0,
		line:     1,
	}
}

func (s *State) HasNext() bool {
	if s.err != nil {
		return false
	}

	_, err := s.peek()
	if errors.Is(err, iterator.ErrNotEnoughItems) {
		return false
	}

	if err != nil {
		s.err = err

		return false
	}

	return true
}

func (s *State) Next() Token {
	token, err := s.parseToken()
	if err != nil {
		s.err = err

		return nil
	}

	return token
}

func (s *State) Err() error {
	if s.err != nil {
		return s.err
	}

	return s.iterator.Err()
}

func (s *State) parseToken() (Token, error) {
	position, char, err := s.read()
	if err != nil {
		return nil, err
	}

	switch {
	case isWhitespace(char):
		return s.parseWhitespace(position, char)

	case char == '-', isDigit(char):
		return s.parseNumber(position, char)

	case isAlpha(char):
		return s.parseWord(position, char)

	case char == '\'':
		return s.parseSingleQuoteString(position, char)

	case char == '"':
		return s.parseDoubleQuoteString(position, char)

	case char == '`':
		return s.parseDiagonalQuoteString(position, char)

	default:
		return s.parseSymbol(position, char)
	}
}

func (s *State) parseWhitespace(position Position, char byte) (Token, error) {
	var tokenType TokenType

	switch char {
	case ' ':
		tokenType = TokenTypeSpace
	case '\t':
		tokenType = TokenTypeTab
	case '\r':
		tokenType = TokenTypeCarriageReturn
	case '\n':
		tokenType = TokenTypeNewLine
	default:
		return nil, ErrUnknownToken
	}

	return NewToken(tokenType, position, []byte{char}), nil
}

// nolint:gocognit,cyclop
func (s *State) parseNumber(position Position, char byte) (Token, error) {
	content := []byte{char}

	if char == '-' {
		next, err := s.peek()
		if errors.Is(err, iterator.ErrNotEnoughItems) || (err == nil && !isDigit(next)) {
			return s.parseSymbol(position, content...)
		}

		if err != nil {
			return nil, err
		}

		_, next, err = s.read()
		if err != nil {
			return nil, err
		}

		content = append(content, next)
	}

	if (len(content) == 1 && content[0] == '0') || (len(content) == 2 && content[0] == '-' && content[1] == '0') {
		tokenType, matched, err := s.parseBasePrefix(&content)
		if err != nil {
			return nil, err
		}

		if matched {
			return NewToken(tokenType, position, content), nil
		}
	}

	for {
		next, err := s.peek()
		if errors.Is(err, iterator.ErrNotEnoughItems) {
			break
		}

		if err != nil {
			return nil, err
		}

		if !isDigit(next) {
			break
		}

		_, next, err = s.read()
		if err != nil {
			return nil, err
		}

		content = append(content, next)
	}

	if s.hasFloatFraction() {
		_, dot, err := s.read()
		if err != nil {
			return nil, err
		}

		content = append(content, dot)

		for {
			next, err := s.peek()
			if errors.Is(err, iterator.ErrNotEnoughItems) {
				break
			}

			if err != nil {
				return nil, err
			}

			if !isDigit(next) {
				break
			}

			_, next, err = s.read()
			if err != nil {
				return nil, err
			}

			content = append(content, next)
		}

		return NewToken(TokenTypeFloat, position, content), nil
	}

	return NewToken(TokenTypeNumber, position, content), nil
}

func (s *State) parseWord(position Position, char byte) (Token, error) {
	content := []byte{char}

	for {
		next, err := s.peek()
		if errors.Is(err, iterator.ErrNotEnoughItems) {
			break
		}

		if err != nil {
			return nil, err
		}

		if !isWordChar(next) {
			break
		}

		_, next, err = s.read()
		if err != nil {
			return nil, err
		}

		content = append(content, next)
	}

	return NewToken(TokenTypeWord, position, content), nil
}

func (s *State) parseSingleQuoteString(position Position, char byte) (Token, error) {
	return s.parseQuotedString(TokenTypeSingleQuoteString, position, char, true)
}

func (s *State) parseDoubleQuoteString(position Position, char byte) (Token, error) {
	return s.parseQuotedString(TokenTypeDoubleQuoteString, position, char, true)
}

func (s *State) parseDiagonalQuoteString(position Position, char byte) (Token, error) {
	return s.parseQuotedString(TokenTypeDiagonalQuoteString, position, char, false)
}

func (s *State) parseSymbol(position Position, chars ...byte) (Token, error) {
	content := append([]byte(nil), chars...)

	return NewToken(TokenTypeSymbol, position, content), nil
}

func (s *State) parseBasePrefix(content *[]byte) (TokenType, bool, error) {
	prefix, digit, err := s.peek2()
	if errors.Is(err, iterator.ErrNotEnoughItems) {
		return "", false, nil
	}

	if err != nil {
		return "", false, err
	}

	var (
		tokenType  TokenType
		digitCheck func(byte) bool
	)

	switch {
	case (prefix == 'x' || prefix == 'X') && isHexDigit(digit):
		tokenType = TokenTypeHexDecimal
		digitCheck = isHexDigit

	case (prefix == 'b' || prefix == 'B') && isBinaryDigit(digit):
		tokenType = TokenTypeBinary
		digitCheck = isBinaryDigit

	default:
		return "", false, nil
	}

	_, prefix, err = s.read()
	if err != nil {
		return "", false, err
	}

	*content = append(*content, prefix)

	for {
		next, err := s.peek()
		if errors.Is(err, iterator.ErrNotEnoughItems) {
			break
		}

		if err != nil {
			return "", false, err
		}

		if !digitCheck(next) {
			break
		}

		_, next, err = s.read()
		if err != nil {
			return "", false, err
		}

		*content = append(*content, next)
	}

	return tokenType, true, nil
}

func (s *State) parseQuotedString(tokenType TokenType, position Position, quote byte, allowEscapes bool) (Token, error) {
	content := []byte{quote}
	escaped := false

	for {
		_, next, err := s.read()
		if errors.Is(err, iterator.ErrNotEnoughItems) {
			return nil, ErrUnknownToken
		}

		if err != nil {
			return nil, err
		}

		content = append(content, next)

		if allowEscapes && escaped {
			escaped = false

			continue
		}

		if allowEscapes && next == '\\' {
			escaped = true

			continue
		}

		if next == quote {
			return NewToken(tokenType, position, content), nil
		}
	}
}

func (s *State) hasFloatFraction() bool {
	first, second, err := s.peek2()
	if errors.Is(err, iterator.ErrNotEnoughItems) {
		return false
	}

	return err == nil && first == '.' && isDigit(second)
}

func (s *State) peek() (byte, error) {
	return s.iterator.Peek()
}

func (s *State) peek2() (byte, byte, error) {
	return s.iterator.Peek2()
}

func (s *State) read() (Position, byte, error) {
	position := NewPosition(s.offset, s.column, s.line)

	char, err := s.peek()
	if err != nil {
		return position, 0, err
	}

	if err := s.iterator.Consume(); err != nil {
		return position, 0, err
	}

	s.advance(char)

	return position, char, nil
}

func (s *State) advance(char byte) {
	s.offset++

	if char == '\n' {
		s.line++
		s.column = 0

		return
	}

	s.column++
}

func isWhitespace(char byte) bool {
	return char == ' ' || char == '\t' || char == '\r' || char == '\n'
}

func isDigit(char byte) bool {
	return char >= '0' && char <= '9'
}

func isAlpha(char byte) bool {
	return (char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z')
}

func isWordChar(char byte) bool {
	return isAlpha(char) || isDigit(char) || char == '_'
}

func isHexDigit(char byte) bool {
	return isDigit(char) || (char >= 'a' && char <= 'f') || (char >= 'A' && char <= 'F')
}

func isBinaryDigit(char byte) bool {
	return char == '0' || char == '1'
}
