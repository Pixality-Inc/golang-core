package proto_parser

type InputContext struct {
	FileId  int
	Package string
}

func NewInputContext(fileId int, pkg string) *InputContext {
	return &InputContext{
		FileId:  fileId,
		Package: pkg,
	}
}
