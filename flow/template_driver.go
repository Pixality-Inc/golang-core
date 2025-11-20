package flow

import "context"

type TemplateDriver interface {
	Execute(ctx context.Context, env *Env, name string, source string) (string, error)
}
