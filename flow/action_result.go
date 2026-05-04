package flow

type ActionResult struct {
	DataScript string `json:"data_script" yaml:"data_script"`
}

func NewActionResult() ActionResult {
	return ActionResult{}
}

func (r ActionResult) WithDataScript(script string) ActionResult {
	r.DataScript = script

	return r
}
