package flow

import "context"

//go:generate mockgen -destination mocks/script_driver_gen.go -source script_driver.go
type ScriptDriver interface {
	Execute(ctx context.Context, env *Env, name string, script string) (any, error)
	ValueToString(value any) (string, error)
	ValueToBool(value any) (bool, error)
	ValueToStringSlice(value any) ([]string, error)
	ValueToMapStringString(value any) (map[string]string, error)
	AnyToValue(value any) (any, error)
}
