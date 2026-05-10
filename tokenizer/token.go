package tokenizer

type TokenType string

const (
	TokenTypeSpace               TokenType = "space"
	TokenTypeTab                 TokenType = "tab"
	TokenTypeCarriageReturn      TokenType = "carriage_return"
	TokenTypeNewLine             TokenType = "new_line"
	TokenTypeNumber              TokenType = "number"
	TokenTypeFloat               TokenType = "float"
	TokenTypeHexDecimal          TokenType = "hex_decimal"
	TokenTypeBinary              TokenType = "binary"
	TokenTypeSingleQuoteString   TokenType = "single_quote_string" //nolint:gosec
	TokenTypeDoubleQuoteString   TokenType = "double_quote_string" //nolint:gosec
	TokenTypeDiagonalQuoteString TokenType = "diagonal_quote_string"
	TokenTypeWord                TokenType = "word"
	TokenTypeSymbol              TokenType = "symbol"
)

type Token interface {
	Type() TokenType
	Position() Position
	Content() []byte
	Length() uint64
}

type TokenImpl struct {
	tokenType TokenType
	position  Position
	content   []byte
}

func NewToken(
	tokenType TokenType,
	position Position,
	content []byte,
) Token {
	return &TokenImpl{
		tokenType: tokenType,
		position:  position,
		content:   content,
	}
}

func (t *TokenImpl) Type() TokenType {
	return t.tokenType
}

func (t *TokenImpl) Position() Position {
	return t.position
}

func (t *TokenImpl) Content() []byte {
	return t.content
}

func (t *TokenImpl) Length() uint64 {
	return uint64(len(t.content))
}
