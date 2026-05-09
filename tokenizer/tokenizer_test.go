package tokenizer

import (
	"bytes"
	"context"
	"testing"

	"github.com/pixality-inc/golang-core/iterator"
	"github.com/stretchr/testify/require"
)

func Test_Tokenizer(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	tok := New()

	source := bytes.NewReader([]byte("hello, world"))

	iter, err := tok.Tokenize(ctx, source)
	require.NoError(t, err)

	values, err := iterator.Materialize(iter)
	require.NoError(t, err)

	require.Len(t, values, 4)
	require.Equal(t, TokenTypeWord, values[0].Type())
	require.Equal(t, []byte("hello"), values[0].Content())
	require.Equal(t, TokenTypeSymbol, values[1].Type())
	require.Equal(t, []byte(","), values[1].Content())
	require.Equal(t, TokenTypeSpace, values[2].Type())
	require.Equal(t, []byte(" "), values[2].Content())
	require.Equal(t, TokenTypeWord, values[3].Type())
	require.Equal(t, []byte("world"), values[3].Content())
}

func Test_TokenizerTokenTypes(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	tok := New()

	source := bytes.NewReader([]byte("alpha_1 69 -42 3.14 -3.14 0 0x2A 0x2a 0b101 'a\\'b' \"c\\\"d\" `raw` == != -"))

	iter, err := tok.Tokenize(ctx, source)
	require.NoError(t, err)

	values, err := iterator.Materialize(iter)
	require.NoError(t, err)

	require.Equal(t, []TokenType{
		TokenTypeWord,
		TokenTypeSpace,
		TokenTypeNumber,
		TokenTypeSpace,
		TokenTypeNumber,
		TokenTypeSpace,
		TokenTypeFloat,
		TokenTypeSpace,
		TokenTypeFloat,
		TokenTypeSpace,
		TokenTypeNumber,
		TokenTypeSpace,
		TokenTypeHexDecimal,
		TokenTypeSpace,
		TokenTypeHexDecimal,
		TokenTypeSpace,
		TokenTypeBinary,
		TokenTypeSpace,
		TokenTypeSingleQuoteString,
		TokenTypeSpace,
		TokenTypeDoubleQuoteString,
		TokenTypeSpace,
		TokenTypeDiagonalQuoteString,
		TokenTypeSpace,
		TokenTypeSymbol,
		TokenTypeSymbol,
		TokenTypeSpace,
		TokenTypeSymbol,
		TokenTypeSymbol,
		TokenTypeSpace,
		TokenTypeSymbol,
	}, tokenTypes(values))

	require.Equal(t, [][]byte{
		[]byte("alpha_1"),
		[]byte(" "),
		[]byte("69"),
		[]byte(" "),
		[]byte("-42"),
		[]byte(" "),
		[]byte("3.14"),
		[]byte(" "),
		[]byte("-3.14"),
		[]byte(" "),
		[]byte("0"),
		[]byte(" "),
		[]byte("0x2A"),
		[]byte(" "),
		[]byte("0x2a"),
		[]byte(" "),
		[]byte("0b101"),
		[]byte(" "),
		[]byte("'a\\'b'"),
		[]byte(" "),
		[]byte("\"c\\\"d\""),
		[]byte(" "),
		[]byte("`raw`"),
		[]byte(" "),
		[]byte("="),
		[]byte("="),
		[]byte(" "),
		[]byte("!"),
		[]byte("="),
		[]byte(" "),
		[]byte("-"),
	}, tokenContents(values))
}

func Test_TokenizerTokenPositions(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	tok := New()

	sourceBytes := []byte("alpha_1 69 -42 3.14 -3.14 0 0x2A 0x2a 0b101 'a\\'b' \"c\\\"d\" `raw` == != -")

	source := bytes.NewReader(sourceBytes)

	iter, err := tok.Tokenize(ctx, source)
	require.NoError(t, err)

	tokens, err := iterator.Materialize(iter)
	require.NoError(t, err)

	for _, token := range tokens {
		offset := token.Position().Offset()
		length := token.Length()

		buf := sourceBytes[offset : offset+length]

		require.Equal(t, string(token.Content()), string(buf))
	}
}

func Test_TokenizerTokenLength(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	tok := New()

	source := bytes.NewReader([]byte("a 'bc' \"d\" `ef` 0x2A"))

	iter, err := tok.Tokenize(ctx, source)
	require.NoError(t, err)

	values, err := iterator.Materialize(iter)
	require.NoError(t, err)

	require.Equal(t, []uint64{
		1,
		1,
		4,
		1,
		3,
		1,
		4,
		1,
		4,
	}, tokenLengths(values))

	for _, token := range values {
		require.Equal(t, uint64(len(token.Content())), token.Length())
	}
}

func Test_TokenizerWhitespaceTokens(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	tok := New()

	source := bytes.NewReader([]byte(" \t\r\n"))

	iter, err := tok.Tokenize(ctx, source)
	require.NoError(t, err)

	values, err := iterator.Materialize(iter)
	require.NoError(t, err)

	require.Equal(t, []TokenType{
		TokenTypeSpace,
		TokenTypeTab,
		TokenTypeCarriageReturn,
		TokenTypeNewLine,
	}, tokenTypes(values))

	require.Equal(t, [][]byte{
		[]byte(" "),
		[]byte("\t"),
		[]byte("\r"),
		[]byte("\n"),
	}, tokenContents(values))
}

func Test_TokenizerPositions(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	tok := New()

	source := bytes.NewReader([]byte("one\n  two"))

	iter, err := tok.Tokenize(ctx, source)
	require.NoError(t, err)

	values, err := iterator.Materialize(iter)
	require.NoError(t, err)

	require.Len(t, values, 5)
	require.Equal(t, uint64(0), values[0].Position().Offset())
	require.Equal(t, uint64(0), values[0].Position().Column())
	require.Equal(t, uint64(1), values[0].Position().Line())
	require.Equal(t, TokenTypeNewLine, values[1].Type())
	require.Equal(t, uint64(3), values[1].Position().Offset())
	require.Equal(t, uint64(3), values[1].Position().Column())
	require.Equal(t, uint64(1), values[1].Position().Line())
	require.Equal(t, TokenTypeSpace, values[2].Type())
	require.Equal(t, uint64(4), values[2].Position().Offset())
	require.Equal(t, uint64(0), values[2].Position().Column())
	require.Equal(t, uint64(2), values[2].Position().Line())
	require.Equal(t, TokenTypeSpace, values[3].Type())
	require.Equal(t, uint64(5), values[3].Position().Offset())
	require.Equal(t, uint64(1), values[3].Position().Column())
	require.Equal(t, uint64(2), values[3].Position().Line())
	require.Equal(t, uint64(6), values[4].Position().Offset())
	require.Equal(t, uint64(2), values[4].Position().Column())
	require.Equal(t, uint64(2), values[4].Position().Line())
}

func Test_TokenizerUnclosedString(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	tok := New()

	source := bytes.NewReader([]byte("'unterminated"))

	iter, err := tok.Tokenize(ctx, source)
	require.NoError(t, err)

	_, err = iterator.Materialize(iter)
	require.ErrorIs(t, err, ErrUnknownToken)
}

func tokenTypes(tokens []Token) []TokenType {
	result := make([]TokenType, 0, len(tokens))

	for _, token := range tokens {
		result = append(result, token.Type())
	}

	return result
}

func tokenContents(tokens []Token) [][]byte {
	result := make([][]byte, 0, len(tokens))

	for _, token := range tokens {
		result = append(result, token.Content())
	}

	return result
}

func tokenLengths(tokens []Token) []uint64 {
	result := make([]uint64, 0, len(tokens))

	for _, token := range tokens {
		result = append(result, token.Length())
	}

	return result
}
