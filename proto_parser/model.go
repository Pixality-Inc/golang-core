package proto_parser

type Model interface {
	FileId() int
	Package() string
	Path() []string
	Name() string
	Fields() []Field
}

type ModelImpl struct {
	fileId int
	pkg    string
	path   []string
	name   string
	fields []Field
}

func NewModel(
	fileId int,
	pkg string,
	path []string,
	name string,
	fields []Field,
) Model {
	return &ModelImpl{
		fileId: fileId,
		pkg:    pkg,
		path:   path,
		name:   name,
		fields: fields,
	}
}

func (m *ModelImpl) FileId() int {
	return m.fileId
}

func (m *ModelImpl) Package() string {
	return m.pkg
}

func (m *ModelImpl) Path() []string {
	return m.path
}

func (m *ModelImpl) Name() string {
	return m.name
}

func (m *ModelImpl) Fields() []Field {
	return m.fields
}
