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

type Config struct {
	Actions []Action `yaml:"actions"`
}
