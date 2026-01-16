package mailer

import (
	"bytes"
	"context"
	"errors"
	"text/template"

	"github.com/pixality-inc/golang-core/util"
)

var (
	errTemplateParse   = errors.New("template parse")
	errTemplateExecute = errors.New("template execute")
)

type BodyType string

const (
	BodyTypeHtml       BodyType = "html"
	BodyTypeText       BodyType = "text"
	BodyTypeTemplate   BodyType = "template"
	BodyTypeGoTemplate BodyType = "go_template"
)

type Body interface {
	Type() BodyType
	Content(ctx context.Context) (string, error)
}

type BodyImpl struct {
	bodyType BodyType
}

func NewBodyImpl(bodyType BodyType) *BodyImpl {
	return &BodyImpl{
		bodyType: bodyType,
	}
}

func (b *BodyImpl) Type() BodyType {
	return b.bodyType
}

type HtmlBody struct {
	*BodyImpl

	content string
}

func NewHtmlBody(content string) *HtmlBody {
	return &HtmlBody{
		BodyImpl: NewBodyImpl(BodyTypeHtml),
		content:  content,
	}
}

func (b *HtmlBody) Content(_ context.Context) (string, error) {
	return b.content, nil
}

type TextBody struct {
	*BodyImpl

	content string
}

func NewTextBody(content string) *TextBody {
	return &TextBody{
		BodyImpl: NewBodyImpl(BodyTypeText),
		content:  content,
	}
}

func (b *TextBody) Content(_ context.Context) (string, error) {
	return b.content, nil
}

type TemplateBody struct {
	*BodyImpl

	name      string
	variables map[string]any
}

func NewTemplateBody(name string, variables map[string]any) *TemplateBody {
	return &TemplateBody{
		BodyImpl:  NewBodyImpl(BodyTypeTemplate),
		name:      name,
		variables: variables,
	}
}

func (b *TemplateBody) Name() string {
	return b.name
}

func (b *TemplateBody) Variables() map[string]any {
	return b.variables
}

func (b *TemplateBody) Content(_ context.Context) (string, error) {
	return "", util.ErrNotImplemented
}

type GoTemplateBody struct {
	*BodyImpl

	name      string
	template  string
	variables map[string]any
}

func NewGoTemplateBody(name string, template string, variables map[string]any) *GoTemplateBody {
	if variables == nil {
		variables = make(map[string]any)
	}

	return &GoTemplateBody{
		BodyImpl:  NewBodyImpl(BodyTypeGoTemplate),
		name:      name,
		template:  template,
		variables: variables,
	}
}

func (b *GoTemplateBody) Content(_ context.Context) (string, error) {
	tpl, err := template.New(b.name).Parse(b.template)
	if err != nil {
		return "", errors.Join(errTemplateParse, err)
	}

	outputWriter := bytes.NewBuffer(nil)

	if err = tpl.Execute(outputWriter, b.variables); err != nil {
		return "", errors.Join(errTemplateExecute, err)
	}

	return outputWriter.String(), nil
}
