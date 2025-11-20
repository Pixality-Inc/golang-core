package flow

type ActionResponse struct {
	ErrorCode int    `json:"error_code"`
	Stdout    string `json:"stdout"`
	Stderr    string `json:"stderr"`
	Duration  int32  `json:"duration"`
	Skipped   bool   `json:"skipped"`
	Result    string `json:"result"`
}

func NewActionResponse() *ActionResponse {
	return &ActionResponse{
		ErrorCode: 0,
		Stdout:    "",
		Stderr:    "",
		Duration:  0,
		Skipped:   false,
		Result:    "",
	}
}

func (r *ActionResponse) WithExitCode(exitCode int) *ActionResponse {
	r.ErrorCode = exitCode

	return r
}

func (r *ActionResponse) WithStdout(stdout string) *ActionResponse {
	r.Stdout = stdout

	return r
}

func (r *ActionResponse) WithStderr(stderr string) *ActionResponse {
	r.Stderr = stderr

	return r
}

func (r *ActionResponse) WithDuration(millis int64) *ActionResponse {
	r.Duration = int32(millis) //nolint:gosec

	return r
}

func (r *ActionResponse) WithSkipped(skipped bool) *ActionResponse {
	r.Skipped = skipped

	return r
}

func (r *ActionResponse) WithResult(result string) *ActionResponse {
	r.Result = result

	return r
}
