package flow_goja

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/dop251/goja"
	"github.com/pixality-inc/golang-core/flow"
	"github.com/pixality-inc/golang-core/util"
)

var (
	ErrWrongValue           = errors.New("wrong value")
	ErrJsValueToStringArray = errors.New("js value to string array")
)

type Goja struct {
	js *goja.Runtime
}

func NewGoja() *Goja {
	return &Goja{
		js: goja.New(),
	}
}

func (d *Goja) Execute(ctx context.Context, env *flow.Env, name string, script string) (any, error) {
	for key, value := range env.Context {
		if err := d.js.Set(key, value); err != nil {
			return nil, fmt.Errorf("setting env context key %s: %w", key, err)
		}
	}

	result, err := d.js.RunString(script)
	if err != nil {
		return nil, fmt.Errorf("script %s failed: %w", name, err)
	}

	return result, nil
}

func (d *Goja) ValueToString(value any) (string, error) {
	val, ok := value.(goja.Value)
	if !ok {
		return "", fmt.Errorf("%w: value %T is not a goja.Value", ErrWrongValue, value)
	}

	return val.ToString().String(), nil
}

func (d *Goja) ValueToBool(value any) (bool, error) {
	val, ok := value.(goja.Value)
	if !ok {
		return false, fmt.Errorf("%w: value %T is not a goja.Value", ErrWrongValue, value)
	}

	return val.ToBoolean(), nil
}

func (d *Goja) ValueToStringSlice(value any) ([]string, error) {
	val, ok := value.(goja.Value)
	if !ok {
		return nil, fmt.Errorf("%w: value %T is not a goja.Value", ErrWrongValue, value)
	}

	switch jsValue := val.(type) {
	case goja.DynamicArray:
		results := make([]string, jsValue.Len())

		for index := range jsValue.Len() {
			stringValue, err := d.ValueToString(jsValue.Get(index))
			if err != nil {
				return nil, fmt.Errorf("array index %d: %w", index, err)
			}

			results[index] = stringValue
		}

		return results, nil

	case *goja.Object:
		exportedObject := jsValue.Export()

		array, ok := (exportedObject).([]any)
		if !ok {
			return nil, fmt.Errorf("%w: %T is not an array", ErrJsValueToStringArray, exportedObject)
		}

		results := make([]string, len(array))

		for index, element := range array {
			switch elem := element.(type) {
			case string:
				results[index] = elem
			case int64:
				results[index] = strconv.FormatInt(elem, 10)
			case int32:
				results[index] = strconv.FormatInt(int64(elem), 10)
			case bool:
				results[index] = strconv.FormatBool(elem)
			case float64:
				results[index] = strconv.FormatFloat(elem, 'f', -1, 64)
			case float32:
				results[index] = strconv.FormatFloat(float64(elem), 'f', -1, 32)
			default:
				return nil, fmt.Errorf("%w: %T at index %d is not a string", ErrJsValueToStringArray, element, index)
			}
		}

		return results, nil

	default:
		return nil, fmt.Errorf("%w: array type (%T)", ErrJsValueToStringArray, value)
	}
}

func (d *Goja) ValueToMapStringString(value any) (map[string]string, error) {
	// @todo fixme
	return nil, util.ErrNotImplemented
}
