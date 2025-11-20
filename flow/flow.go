package flow

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/pixality-inc/golang-core/cli"
	"github.com/pixality-inc/golang-core/errors"
	"github.com/pixality-inc/golang-core/logger"
	"github.com/pixality-inc/golang-core/storage"
	"github.com/pixality-inc/golang-core/timetrack"
	"github.com/pixality-inc/golang-core/util"
)

var (
	ErrActionNoName        = errors.New("flow.action_no_name", "action does not contain an action name")
	ErrActionDuplicateName = errors.New("flow.action_name_duplicate", "action name is duplicated")
)

type Flow interface {
	Validate(ctx context.Context) error
	Run(ctx context.Context, env *Env) (*Result, error)
	EvalTemplate(ctx context.Context, env *Env, name string, source string) (string, error)
	EvalScript(ctx context.Context, env *Env, name string, script string) (any, error)
	ValueToString(value any) (string, error)
	ValueToBool(value any) (bool, error)
	ValueToStringSlice(value any) ([]string, error)
	ValueToMapStringString(value any) (map[string]string, error)
}

type Impl struct {
	log            logger.Loggable
	config         *Config
	storage        storage.Storage
	templateDriver TemplateDriver
	scriptDriver   ScriptDriver
}

func New(
	config *Config,
	storage storage.Storage,
	templateDriver TemplateDriver,
	scriptDriver ScriptDriver,
) *Impl {
	return &Impl{
		log:            logger.NewLoggableImplWithService("flow"),
		config:         config,
		storage:        storage,
		templateDriver: templateDriver,
		scriptDriver:   scriptDriver,
	}
}

func (f *Impl) Validate(_ context.Context) error {
	actionsNames := make(map[string]struct{})

	for actionIndex, action := range f.config.Actions {
		if action.Name == "" {
			return fmt.Errorf("%w: index %d", ErrActionNoName, actionIndex)
		}

		if _, ok := actionsNames[action.Name]; ok {
			return fmt.Errorf("%w: %s at index %d", ErrActionDuplicateName, action.Name, actionIndex)
		}

		actionsNames[action.Name] = struct{}{}
	}

	return nil
}

func (f *Impl) Run(ctx context.Context, env *Env) (*Result, error) {
	log := f.log.GetLogger(ctx)

	result := &Result{
		ActionsResponses: make(map[string]*ActionResponse, len(f.config.Actions)),
	}

	for _, action := range f.config.Actions {
		track := timetrack.New()

		actionResponse, err := f.runAction(ctx, env, action)
		if err != nil {
			return nil, fmt.Errorf("action '%s' failed in %s: %w", action.Name, util.FormatDuration(track.Finish()), err)
		}

		duration := track.Finish()

		log.Debugf("Action '%s' executed in %s", action.Name, util.FormatDuration(duration))

		result.ActionsResponses[action.Name] = actionResponse.WithDuration(duration.Milliseconds())
	}

	return result, nil
}

func (f *Impl) EvalTemplate(ctx context.Context, env *Env, name string, source string) (string, error) {
	return f.templateDriver.Execute(ctx, env, name, source)
}

func (f *Impl) EvalScript(ctx context.Context, env *Env, name string, script string) (any, error) {
	return f.scriptDriver.Execute(ctx, env, name, script)
}

func (f *Impl) ValueToString(value any) (string, error) {
	return f.scriptDriver.ValueToString(value)
}

func (f *Impl) ValueToBool(value any) (bool, error) {
	return f.scriptDriver.ValueToBool(value)
}

func (f *Impl) ValueToStringSlice(value any) ([]string, error) {
	return f.scriptDriver.ValueToStringSlice(value)
}

func (f *Impl) ValueToMapStringString(value any) (map[string]string, error) {
	return f.scriptDriver.ValueToMapStringString(value)
}

func (f *Impl) runAction(ctx context.Context, env *Env, action Action) (*ActionResponse, error) {
	hasCommand := action.Command != ""
	hasScript := action.Script != ""
	hasScriptFile := action.ScriptFile != ""

	optionsSum := util.SliceSum([]bool{hasCommand, hasScript, hasScriptFile}, 0, boolInc)

	if optionsSum <= 0 {
		return nil, ErrActionNoOptions
	}

	if optionsSum > 1 {
		return nil, ErrActionTooManyOptions
	}

	if ok, err := f.actionWhen(ctx, env, action); err != nil {
		return nil, fmt.Errorf("when failed: %w", err)
	} else if !ok {
		return NewActionResponse().WithSkipped(true), nil
	}

	switch {
	case hasCommand:
		return f.runActionCommand(ctx, env, action)

	case hasScript:
		return f.runActionScript(ctx, env, action)

	case hasScriptFile:
		return f.runActionScriptFile(ctx, env, action)
	}

	return nil, util.ErrNotImplemented
}

//nolint:cyclop
func (f *Impl) runActionCommand(ctx context.Context, env *Env, action Action) (*ActionResponse, error) {
	log := f.log.GetLogger(ctx)

	log.Debugf("Running action '%s' as command", action.Name)

	failIfNonZeroCode := util.OrDefault(action.FailIfNonZeroCode, true)

	command := action.Command

	if command != "" {
		evalResult, err := f.evalTemplate(ctx, env, "action."+action.Name+".command", command)
		if err != nil {
			return nil, fmt.Errorf("eval template for action '%s' command: %w", action.Name, err)
		}

		command = evalResult
	}

	workDir := action.WorkDir

	if workDir != "" {
		evalResult, err := f.evalTemplate(ctx, env, "action."+action.Name+".work_dir", workDir)
		if err != nil {
			return nil, fmt.Errorf("eval template for action '%s' work dir: %w", action.Name, err)
		}

		workDir = evalResult
	}

	if workDir == "" {
		workDir = env.WorkDir
	}

	if workDir != "" {
		if err := os.MkdirAll(workDir, os.ModePerm); err != nil {
			return nil, fmt.Errorf("mkdir %s: %w", workDir, err)
		}
	}

	commandStdout := action.Stdout

	if commandStdout != "" {
		evalResult, err := f.evalTemplate(ctx, env, "action."+action.Name+".stdout", commandStdout)
		if err != nil {
			return nil, fmt.Errorf("eval template for action '%s' stdout: %w", action.Name, err)
		}

		commandStdout = evalResult
	}

	if commandStdout != "" && !strings.HasPrefix(commandStdout, "/") {
		commandStdout = path.Join(workDir, commandStdout)
	}

	commandStderr := action.Stderr

	if commandStderr != "" {
		evalResult, err := f.evalTemplate(ctx, env, "action."+action.Name+".stderr", commandStderr)
		if err != nil {
			return nil, fmt.Errorf("eval template for action '%s' stderr: %w", action.Name, err)
		}

		commandStderr = evalResult
	}

	if commandStderr != "" && !strings.HasPrefix(commandStderr, "/") {
		commandStderr = path.Join(workDir, commandStderr)
	}

	cmdArgs, err := f.getActionArgs(ctx, env, action)
	if err != nil {
		return nil, fmt.Errorf("get action '%s' arguments: %w", action.Name, err)
	}

	cmdEnvs, err := f.getActionEnv(ctx, env, action)
	if err != nil {
		return nil, fmt.Errorf("get action '%s' environment: %w", action.Name, err)
	}

	cmd := exec.CommandContext(
		ctx,
		command,
		cmdArgs...,
	)

	cmd.Dir = workDir

	if cmdEnvs != nil {
		var envs []string

		for k, v := range cmdEnvs {
			envs = append(envs, fmt.Sprintf("%s=%s", k, v))
		}

		cmd.Env = append(os.Environ(), envs...)
	}

	log.Debugf("Running command: %s", cmd.String())

	exitCode, stdout, stderr, err := cli.ExecCommand(cmd, false)
	if err != nil {
		return nil, fmt.Errorf("%w: command '%s' for action '%s' failed: %w: %s", ErrCommandFailed, command, action.Name, err, stderr)
	}

	response := NewActionResponse().
		WithExitCode(exitCode).
		WithStdout(string(stdout)).
		WithStderr(string(stderr))

	if commandStdout != "" {
		log.Debugf("Writing command '%s' stdout to %s", action.Name, commandStdout)

		response.Stdout = ""

		//nolint:gosec
		if err = os.WriteFile(commandStdout, stdout, os.ModePerm); err != nil {
			return nil, fmt.Errorf("write command stdout '%s' to %s: %w", action.Name, commandStdout, err)
		}
	}

	if commandStderr != "" {
		log.Debugf("Writing command '%s' stderr to %s", action.Name, commandStderr)

		response.Stderr = ""

		//nolint:gosec
		if err = os.WriteFile(commandStderr, stderr, os.ModePerm); err != nil {
			return nil, fmt.Errorf("write command stderr '%s' to %s: %w", action.Name, commandStderr, err)
		}
	}

	if failIfNonZeroCode && exitCode != 0 {
		return response, fmt.Errorf("%w: command '%s', action '%s', exit code %d: %s", ErrCommandFailed, command, action.Name, exitCode, stderr)
	}

	return response, nil
}

func (f *Impl) actionWhen(ctx context.Context, env *Env, action Action) (bool, error) {
	if action.When == "" {
		return true, nil
	}

	whenResult, err := f.evalScript(ctx, env, "action."+action.Name+".when", action.When)
	if err != nil {
		return false, fmt.Errorf("eval js when: %w", err)
	}

	boolResult, err := f.scriptDriver.ValueToBool(whenResult)
	if err != nil {
		return false, fmt.Errorf("jsValue to bool: %w", err)
	}

	return boolResult, nil
}

func (f *Impl) getActionEnv(ctx context.Context, env *Env, action Action) (map[string]string, error) {
	envs := make(map[string]string)

	hasEnv := len(action.Env) > 0
	hasEnvTemplate := action.EnvTemplate != ""
	hasEnvScript := action.EnvScript != ""

	envOptionsSum := util.SliceSum(
		[]bool{hasEnv, hasEnvTemplate, hasEnvScript},
		0,
		boolInc,
	)

	if envOptionsSum > 1 {
		return nil, ErrActionEnvTooManyOptions
	}

	switch {
	case hasEnv:
		for envName, envValue := range action.Env {
			templateName := "actions." + action.Name + "." + envName

			evalResult, err := f.evalTemplate(ctx, env, templateName, envValue)
			if err != nil {
				return nil, fmt.Errorf("eval env %s key %s: %w", templateName, envName, err)
			}

			envs[envName] = evalResult
		}

	case hasEnvTemplate:
		templateName := "actions." + action.Name

		evalResult, err := f.evalTemplate(ctx, env, templateName, action.EnvTemplate)
		if err != nil {
			return nil, fmt.Errorf("eval env template %s: %w", templateName, err)
		}

		envResult, err := unmarshalTemplateResultObject(evalResult)
		if err != nil {
			return nil, fmt.Errorf("unmarshal env template %s: %w", templateName, err)
		}

		envs, err = asMapStringString(envResult)
		if err != nil {
			return nil, fmt.Errorf("asMapStringString env for action %s: %w", action.Name, err)
		}

	case hasEnvScript:
		templateName := "actions." + action.Name

		evalResult, err := f.evalScript(ctx, env, templateName, action.EnvScript)
		if err != nil {
			return nil, fmt.Errorf("eval env script %s: %w", templateName, err)
		}

		envs, err = f.scriptDriver.ValueToMapStringString(evalResult)
		if err != nil {
			return nil, fmt.Errorf("eval env script %s result to string: %w", templateName, err)
		}
	}

	return envs, nil
}

func (f *Impl) getActionArgs(ctx context.Context, env *Env, action Action) ([]string, error) {
	args := make([]string, 0)

	hasArgs := len(action.Args) > 0
	hasArgsTemplate := action.ArgsTemplate != ""
	hasArgsScript := action.ArgsScript != ""

	argsOptionsSum := util.SliceSum([]bool{hasArgs, hasArgsTemplate, hasArgsScript}, 0, boolInc)

	if argsOptionsSum > 1 {
		return nil, ErrActionArgsTooManyOptions
	}

	switch {
	case hasArgs:
		for argIndex, arg := range action.Args {
			templateName := "actions." + action.Name + "." + strconv.Itoa(argIndex)

			evalResult, err := f.evalTemplate(ctx, env, templateName, arg)
			if err != nil {
				return nil, fmt.Errorf("eval arg %s index %d: %w", templateName, argIndex, err)
			}

			args = append(args, strings.TrimSpace(evalResult))
		}

	case hasArgsTemplate:
		templateName := "actions." + action.Name

		evalResult, err := f.evalTemplate(ctx, env, templateName, action.ArgsTemplate)
		if err != nil {
			return nil, fmt.Errorf("eval arg template %s: %w", templateName, err)
		}

		argsResult, err := UnmarshalTemplateResultSlice(evalResult)
		if err != nil {
			return nil, fmt.Errorf("unmarshal arg template %s: %w", templateName, err)
		}

		args = append(args, argsResult...)

	case hasArgsScript:
		templateName := "actions." + action.Name

		evalResult, err := f.evalScript(ctx, env, templateName, action.ArgsScript)
		if err != nil {
			return nil, fmt.Errorf("eval arg script %s: %w", templateName, err)
		}

		argsResult, err := f.scriptDriver.ValueToStringSlice(evalResult)
		if err != nil {
			return nil, fmt.Errorf("eval arg script %s result to string: %w", templateName, err)
		}

		args = append(args, argsResult...)
	}

	filteredArgs := make([]string, 0)

	for _, arg := range args {
		filteredArg := strings.TrimSpace(arg)

		if filteredArg != "" {
			filteredArgs = append(filteredArgs, filteredArg)
		}
	}

	return filteredArgs, nil
}

func (f *Impl) runActionScript(ctx context.Context, env *Env, action Action) (*ActionResponse, error) {
	return f.runActionScriptCode(ctx, env, action, "action."+action.Name+".script", action.Script)
}

func (f *Impl) runActionScriptFile(ctx context.Context, env *Env, action Action) (*ActionResponse, error) {
	log := f.log.GetLogger(ctx)

	filename, err := f.evalTemplate(ctx, env, "action."+action.Name+".script_filename", action.ScriptFile)
	if err != nil {
		return nil, fmt.Errorf("eval action script filename %s: %w", action.Name, err)
	}

	filenameStr := filename

	var file io.ReadCloser

	if strings.HasPrefix(filenameStr, "/") {
		file, err = os.Open(filenameStr)
		if err != nil {
			return nil, fmt.Errorf("open %s: %w", filenameStr, err)
		}
	} else {
		storageFilename := path.Join(env.WorkDir, filenameStr)

		file, err = f.storage.ReadFile(ctx, storageFilename)
		if err != nil {
			return nil, fmt.Errorf("open storage file %s: %w", storageFilename, err)
		}
	}

	defer func() {
		if fErr := file.Close(); fErr != nil {
			log.WithError(fErr).Errorf("failed to close file %s", filenameStr)
		}
	}()

	buf, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", filenameStr, err)
	}

	basename := filepath.Base(filenameStr)

	code := string(buf)

	return f.runActionScriptCode(ctx, env, action, basename, code)
}

func (f *Impl) runActionScriptCode(ctx context.Context, env *Env, _ Action, name string, code string) (*ActionResponse, error) {
	evalResult, err := f.evalScript(ctx, env, name, code)
	if err != nil {
		return nil, fmt.Errorf("eval script %s: %w", name, err)
	}

	resultStr, err := f.scriptDriver.ValueToString(evalResult)
	if err != nil {
		return nil, fmt.Errorf("eval script %s result to string: %w", name, err)
	}

	return NewActionResponse().WithResult(resultStr), nil
}

func (f *Impl) evalTemplate(ctx context.Context, env *Env, name string, source string) (string, error) {
	return f.templateDriver.Execute(ctx, env, name, source)
}

//nolint:unused
func (f *Impl) evalScript(ctx context.Context, env *Env, name string, source string) (any, error) {
	return f.scriptDriver.Execute(ctx, env, name, source)
}
