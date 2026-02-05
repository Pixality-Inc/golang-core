package proto_parser

type FieldOption = func(field *FieldImpl)

func WithAdditionalType(additionalType string) FieldOption {
	return func(field *FieldImpl) {
		field.additionalType = additionalType
	}
}

func WithIsMap() FieldOption {
	return func(field *FieldImpl) {
		field.isMap = true
	}
}

func WithIsOptional() FieldOption {
	return func(field *FieldImpl) {
		field.isOptional = true
	}
}

func WithIsRepeated() FieldOption {
	return func(field *FieldImpl) {
		field.isRepeated = true
	}
}

func WithComment(comment string) FieldOption {
	return func(field *FieldImpl) {
		field.comment = comment
	}
}

func WithAttribute(key string, value string) FieldOption {
	return func(field *FieldImpl) {
		field.attributes[key] = value
	}
}
