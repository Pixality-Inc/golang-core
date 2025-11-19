package scripts

import "context"

type Script interface {
	Run(ctx context.Context, args []string) error
}
