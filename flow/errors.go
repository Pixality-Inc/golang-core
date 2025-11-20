package flow

import "errors"

var (
	ErrNoScriptDriver           = errors.New("no script driver")
	ErrActionNoOptions          = errors.New("no options provided")
	ErrActionTooManyOptions     = errors.New("too many options provided")
	ErrActionArgsTooManyOptions = errors.New("too many options for action args provided")
	ErrActionEnvTooManyOptions  = errors.New("too many options for action env provided")
	ErrCommandFailed            = errors.New("command failed")
	ErrAsMapStringString        = errors.New("asMapStringString")
	ErrUnmarshalResultObject    = errors.New("unmarshal result object failed")
)
