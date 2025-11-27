package flow

import "time"

type ActionResponse struct {
	ErrorCode  int           `json:"error_code"`
	Stdout     string        `json:"stdout"`
	Stderr     string        `json:"stderr"`
	Skipped    bool          `json:"skipped"`
	StartedAt  time.Time     `json:"started_at"`
	FinishedAt time.Time     `json:"finished_at"`
	Duration   time.Duration `json:"duration"`
	Result     string        `json:"result"`
}

func NewActionResponse() *ActionResponse {
	return &ActionResponse{
		ErrorCode:  0,
		Stdout:     "",
		Stderr:     "",
		Skipped:    false,
		StartedAt:  time.Time{},
		FinishedAt: time.Time{},
		Duration:   0,
		Result:     "",
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

func (r *ActionResponse) WithStartedAt(startedAt time.Time) *ActionResponse {
	r.StartedAt = startedAt

	return r
}

func (r *ActionResponse) WithFinishedAt(finishedAt time.Time) *ActionResponse {
	r.FinishedAt = finishedAt

	return r
}

func (r *ActionResponse) WithDuration(duration time.Duration) *ActionResponse {
	r.Duration = duration

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
