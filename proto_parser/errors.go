package proto_parser

import "errors"

var (
	ErrParseInput               = errors.New("parse input")
	ErrOpenSource               = errors.New("open source")
	ErrParseFile                = errors.New("parse file")
	ErrProcessFile              = errors.New("process file")
	ErrUnknownProtobufElement   = errors.New("unknown protobuf element")
	ErrUnknownProtobufField     = errors.New("unknown protobuf field")
	ErrUnknownProtobufEnumField = errors.New("unknown protobuf enum field")
	ErrProcessMessage           = errors.New("process message")
	ErrProcessEnum              = errors.New("process enum")
)
