package flow

import (
	"fmt"
	"strings"

	"github.com/pixality-inc/golang-core/json"
	"gopkg.in/yaml.v3"
)

func asMapStringString(jsonObject json.Object) (map[string]string, error) {
	result := make(map[string]string)

	for key, value := range jsonObject {
		stringValue, ok := value.(string)
		if !ok {
			return nil, fmt.Errorf("%w: json object key %s is not a string", ErrAsMapStringString, key)
		}

		result[key] = stringValue
	}

	return result, nil
}

func boolInc(prev int, current bool) int {
	if current {
		return prev + 1
	} else {
		return prev
	}
}

func UnmarshalTemplateResultSlice(result string) ([]string, error) {
	result = strings.TrimSpace(result)

	switch {
	case strings.HasPrefix(result, `[`):
		var arr []string

		if err := yaml.Unmarshal([]byte(result), &arr); err != nil {
			return nil, fmt.Errorf("unmarshalling result as slice of strings: %w", err)
		}

		return arr, nil

	case strings.HasPrefix(result, `"`):
		var str string

		if err := yaml.Unmarshal([]byte(result), &str); err != nil {
			return nil, fmt.Errorf("unmarshalling result as string: %w", err)
		}

		return []string{str}, nil

	default:
		return []string{result}, nil
	}
}

func unmarshalTemplateResultObject(result string) (json.Object, error) {
	result = strings.TrimSpace(result)

	switch {
	case strings.HasPrefix(result, `{`):
		var obj json.Object

		if err := yaml.Unmarshal([]byte(result), &obj); err != nil {
			return nil, fmt.Errorf("unmarshalling result as object: %w", err)
		}

		return obj, nil
	default:
		return nil, ErrUnmarshalResultObject
	}
}
