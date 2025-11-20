package flow_gotemplate

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"text/template"

	"github.com/pixality-inc/golang-core/flow"
)

type GoTemplate struct{}

func NewGoTemplate() *GoTemplate {
	return &GoTemplate{}
}

func (d *GoTemplate) Execute(_ context.Context, env *flow.Env, name string, source string) (string, error) {
	tpl, err := template.New(name).Parse(source)
	if err != nil {
		return "", fmt.Errorf("parse template: %w", err)
	}

	var buf bytes.Buffer

	writer := bufio.NewWriter(&buf)

	if err = tpl.Execute(writer, env.Context); err != nil {
		return "", fmt.Errorf("execute template: %w", err)
	}

	if err = writer.Flush(); err != nil {
		return "", fmt.Errorf("flush: %w", err)
	}

	return buf.String(), nil
}
