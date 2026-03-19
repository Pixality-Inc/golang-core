package flow

import "context"

type ActionTriggerFunc func(ctx context.Context, data map[string]any) (map[string]any, error)

type ActionTrigger struct {
	Name       string         `json:"name"        yaml:"name"`
	Data       map[string]any `json:"data"        yaml:"data"`
	DataScript string         `json:"data_script" yaml:"data_script"`
	Async      bool           `json:"async"       yaml:"async"`
}

func NewActionTrigger(name string) ActionTrigger {
	return ActionTrigger{
		Name: name,
	}
}

func (t ActionTrigger) WithData(data map[string]any) ActionTrigger {
	t.Data = data

	return t
}

func (t ActionTrigger) WithDataScript(script string) ActionTrigger {
	t.DataScript = script

	return t
}

func (t ActionTrigger) WithAsync() ActionTrigger {
	t.Async = true

	return t
}
