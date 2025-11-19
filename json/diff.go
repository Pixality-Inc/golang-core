package json

import (
	"errors"
	"fmt"

	"github.com/josephburnett/jd/v2"
)

var ErrUnmarshal = errors.New("can't unmarshal json")

type Diff struct {
	Diffs []jd.DiffElement
}

func NewDiff(jsonA []byte, jsonB []byte) (*Diff, error) {
	aNode, err := jd.ReadJsonString(string(jsonA))
	if err != nil {
		return nil, fmt.Errorf("%w a: %w", ErrUnmarshal, err)
	}

	bNode, err := jd.ReadJsonString(string(jsonB))
	if err != nil {
		return nil, fmt.Errorf("%w b: %w", ErrUnmarshal, err)
	}

	jsonDiff := aNode.Diff(bNode)

	delta := &Diff{
		Diffs: jsonDiff,
	}

	return delta, nil
}
