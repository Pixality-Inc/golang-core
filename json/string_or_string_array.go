package json

import (
	"errors"
	"fmt"
)

var ErrDecodeStringOrStringArray = errors.New("unable to decode to string or string array")

type StringOrStringArray []string

func NewStringOrStringArray(strings ...string) StringOrStringArray {
	return strings
}

func (s *StringOrStringArray) UnmarshalJSON(bytes []byte) error {
	var arrValue []string

	err := Unmarshal(bytes, &arrValue)
	if err == nil {
		*s = arrValue

		return nil
	}

	var strValue string

	err = Unmarshal(bytes, &strValue)
	if err == nil {
		*s = []string{strValue}

		return nil
	}

	return fmt.Errorf("%w: %s", ErrDecodeStringOrStringArray, string(bytes))
}
