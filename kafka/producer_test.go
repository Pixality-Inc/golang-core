package kafka

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

var (
	errFail1 = errors.New("fail1")
	errFail2 = errors.New("fail2")
)

func TestBatchProduceError_Error(t *testing.T) {
	t.Parallel()

	bpe := &BatchProduceError{
		Errors: []error{
			errFail1,
			nil,
			errFail2,
		},
	}

	require.Equal(t, "batch produce: 2 of 3 records failed", bpe.Error())
}

func TestBatchProduceError_Unwrap(t *testing.T) {
	t.Parallel()

	bpe := &BatchProduceError{
		Errors: []error{errFail1, nil, errFail2, nil},
	}

	unwrapped := bpe.Unwrap()
	require.Len(t, unwrapped, 2)
	require.Equal(t, errFail1, unwrapped[0])
	require.Equal(t, errFail2, unwrapped[1])
}

func TestBatchProduceError_Unwrap_AllNil(t *testing.T) {
	t.Parallel()

	bpe := &BatchProduceError{
		Errors: []error{nil, nil},
	}

	require.Empty(t, bpe.Unwrap())
}
