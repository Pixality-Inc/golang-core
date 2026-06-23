package proto_parser

type Field interface {
	Name() string
	Type() string
	AdditionalType() string
	IsOneOf() bool
	IsMap() bool
	IsOptional() bool
	IsRepeated() bool
	Comment() string
	Attributes() map[string]string
	Children() []Field
}

type FieldImpl struct {
	name           string
	typ            string
	additionalType string
	isOneOf        bool
	isMap          bool
	isOptional     bool
	isRepeated     bool
	comment        string
	attributes     map[string]string
	children       []Field
}

func NewField(
	name string,
	typ string,
	options ...FieldOption,
) Field {
	field := &FieldImpl{
		name:           name,
		typ:            typ,
		additionalType: "",
		isOneOf:        false,
		isMap:          false,
		isOptional:     false,
		isRepeated:     false,
		comment:        "",
		attributes:     make(map[string]string),
		children:       make([]Field, 0),
	}

	for _, option := range options {
		option(field)
	}

	return field
}

func (f *FieldImpl) Name() string {
	return f.name
}

func (f *FieldImpl) Type() string {
	return f.typ
}

func (f *FieldImpl) AdditionalType() string {
	return f.additionalType
}

func (f *FieldImpl) IsOneOf() bool {
	return f.isOneOf
}

func (f *FieldImpl) IsMap() bool {
	return f.isMap
}

func (f *FieldImpl) IsOptional() bool {
	return f.isOptional
}

func (f *FieldImpl) IsRepeated() bool {
	return f.isRepeated
}

func (f *FieldImpl) Comment() string {
	return f.comment
}

func (f *FieldImpl) Attributes() map[string]string {
	return f.attributes
}

func (f *FieldImpl) Children() []Field {
	return f.children
}
