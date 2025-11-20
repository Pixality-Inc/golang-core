package flow

import "context"

//go:generate mockgen -destination mocks/template_driver_gen.go -source template_driver.go
type TemplateDriver interface {
	Execute(ctx context.Context, env *Env, name string, source string) (string, error)
}
