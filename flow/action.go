package flow

type Action struct {
	Name              string            `json:"name"                            yaml:"name"`
	Command           string            `json:"command,omitempty"               yaml:"command,omitempty"`
	WorkDir           string            `json:"work_dir,omitempty"              yaml:"work_dir,omitempty"`
	Env               map[string]string `json:"env,omitempty"                   yaml:"env,omitempty"`
	EnvTemplate       string            `json:"env_template,omitempty"          yaml:"env_template,omitempty"`
	EnvScript         string            `json:"env_script,omitempty"            yaml:"env_script,omitempty"`
	Args              []string          `json:"args,omitempty"                  yaml:"args,omitempty"`
	ArgsTemplate      string            `json:"args_template,omitempty"         yaml:"args_template,omitempty"`
	ArgsScript        string            `json:"args_script,omitempty"           yaml:"args_script,omitempty"`
	FailIfNonZeroCode *bool             `json:"fail_if_non_zero_code,omitempty" yaml:"fail_if_non_zero_code,omitempty"`
	Stdout            string            `json:"stdout,omitempty"                yaml:"stdout,omitempty"`
	Stderr            string            `json:"stderr,omitempty"                yaml:"stderr,omitempty"`
	ScriptFile        string            `json:"script_file,omitempty"           yaml:"script_file,omitempty"`
	Script            string            `json:"script,omitempty"                yaml:"script,omitempty"`
	When              string            `json:"when,omitempty"                  yaml:"when,omitempty"`
}

func NewAction(name string) Action {
	return Action{
		Name: name,
		Env:  make(map[string]string),
	}
}

func (a Action) WithCommand(cmd string, args ...string) Action {
	a.Command = cmd
	a.Args = args

	return a
}

func (a Action) WithWorkDir(dir string) Action {
	a.WorkDir = dir

	return a
}

func (a Action) WithEnvs(env map[string]string) Action {
	a.Env = env

	return a
}

func (a Action) WithEnv(key, value string) Action {
	a.Env[key] = value

	return a
}

func (a Action) WithEnvTemplate(template string) Action {
	a.EnvTemplate = template

	return a
}

func (a Action) WithEnvScript(script string) Action {
	a.EnvScript = script

	return a
}

func (a Action) WithArgs(args ...string) Action {
	a.Args = args

	return a
}

func (a Action) WithArgsTemplate(template string) Action {
	a.ArgsTemplate = template

	return a
}

func (a Action) WithArgsScript(script string) Action {
	a.ArgsScript = script

	return a
}

func (a Action) WithFailIfNonZeroCode(fail bool) Action {
	a.FailIfNonZeroCode = &fail

	return a
}

func (a Action) WithStdout(stdout string) Action {
	a.Stdout = stdout

	return a
}

func (a Action) WithStderr(stderr string) Action {
	a.Stderr = stderr

	return a
}

func (a Action) WithScript(script string) Action {
	a.Script = script

	return a
}

func (a Action) WithScriptFile(file string) Action {
	a.ScriptFile = file

	return a
}

func (a Action) WithWhen(when string) Action {
	a.When = when

	return a
}
