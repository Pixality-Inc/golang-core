package flow

import "context"

type ScriptDriver interface {
	Execute(ctx context.Context, env *Env, name string, script string) (any, error)
	ValueToString(value any) (string, error)
	ValueToBool(value any) (bool, error)
	ValueToStringSlice(value any) ([]string, error)
	ValueToMapStringString(value any) (map[string]string, error)
}
