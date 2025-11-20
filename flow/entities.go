package flow

type Env struct {
	WorkDir string
	Context map[string]any
}

func NewEnv(workDir string, context map[string]any) *Env {
	return &Env{
		WorkDir: workDir,
		Context: context,
	}
}

type Result struct {
	ActionsResponses map[string]*ActionResponse
}
