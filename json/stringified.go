package json

import (
	"errors"
	"fmt"
	"strconv"
)

var ErrDecodeStringified = errors.New("unable to decode to stringified")

type Stringified string

func (s *Stringified) UnmarshalJSON(bytes []byte) error {
	var stringValue string

	err := Unmarshal(bytes, &stringValue)
	if err == nil {
		*s = Stringified(stringValue)

		return nil
	}

	var intValue int64

	err = Unmarshal(bytes, &intValue)
	if err == nil {
		*s = Stringified(strconv.FormatInt(intValue, 10))

		return nil
	}

	return fmt.Errorf("%w: %s", ErrDecodeStringified, string(bytes))
}
