package flow

type Env struct {
	WorkDir string
	Context map[string]any
}

type Result struct {
	ActionsResponses map[string]*ActionResponse
}
