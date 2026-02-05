package proto_parser

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/emicklei/proto"
	"github.com/pixality-inc/golang-core/logger"
)

type Parser interface {
	Parse(ctx context.Context, inputs []Input) (*Results, error)
}

type ParserImpl struct {
	log logger.Loggable
}

func New() Parser {
	return &ParserImpl{
		log: logger.NewLoggableImplWithService("proto_parser"),
	}
}

func (p *ParserImpl) Parse(ctx context.Context, inputs []Input) (*Results, error) {
	state := NewState()

	for inputIndex, input := range inputs {
		if err := p.parseInput(ctx, state, inputIndex, input); err != nil {
			return nil, fmt.Errorf("%w: %s: %w", ErrParseInput, input.Name(), err)
		}
	}

	results := NewResult(
		state.models,
		state.enums,
	)

	return results, nil
}

func (p *ParserImpl) parseInput(ctx context.Context, state *State, inputIndex int, input Input) error {
	log := p.log.GetLogger(ctx)

	inputName := input.Name()

	log.Debugf("Parsing input: %s", inputName)

	source, err := input.Source()
	if err != nil {
		return errors.Join(ErrOpenSource, err)
	}

	defer func() {
		if fErr := source.Close(); fErr != nil {
			log.WithError(fErr).Errorf("failed to close input source: %s", inputName)
		}
	}()

	protoParser := proto.NewParser(source)

	protobuf, err := protoParser.Parse()
	if err != nil {
		return errors.Join(ErrParseFile, err)
	}

	inputContext := NewInputContext(inputIndex, input.Package())

	if err = p.processProtobuf(ctx, state, inputContext, protobuf); err != nil {
		return errors.Join(ErrProcessFile, err)
	}

	return nil
}

func (p *ParserImpl) processProtobuf(
	ctx context.Context,
	state *State,
	inputContext *InputContext,
	protobuf *proto.Proto,
) error {
	for _, element := range protobuf.Elements {
		switch elem := element.(type) {
		case *proto.Syntax:
			// skip

		case *proto.Package:
			// skip

		case *proto.Option:
			// skip

		case *proto.Import:
			// skip

		case *proto.Comment:
			// skip

		case *proto.Message:
			if err := p.processMessage(ctx, state, inputContext, nil, elem); err != nil {
				return fmt.Errorf("%w: %s: %w", ErrProcessMessage, elem.Name, err)
			}

		case *proto.Enum:
			if err := p.processEnum(ctx, state, inputContext, nil, elem); err != nil {
				return fmt.Errorf("%w: %s: %w", ErrProcessEnum, elem.Name, err)
			}

		default:
			return fmt.Errorf("%w: %#v", ErrUnknownProtobufElement, elem)
		}
	}

	return nil
}

func (p *ParserImpl) processMessage(
	ctx context.Context,
	state *State,
	inputContext *InputContext,
	path []string,
	message *proto.Message,
) error {
	if message.IsExtend {
		return nil
	}

	fields := make([]Field, 0)

	for _, elementField := range message.Elements {
		switch field := elementField.(type) {
		case *proto.Message:
			if err := p.processMessage(ctx, state, inputContext, append(path, message.Name), field); err != nil {
				return fmt.Errorf("%w: %s: %w", ErrProcessMessage, message.Name, err)
			}

		case *proto.Reserved:
			// skip

		case *proto.Comment:
			// skip

		case *proto.MapField:
			options := make([]FieldOption, 0)

			options = append(
				options,
				WithIsMap(),
				WithAdditionalType(field.KeyType),
			)

			for _, opt := range field.Options {
				options = append(options, WithAttribute(opt.Name, opt.Constant.Source))
			}

			if field.InlineComment != nil {
				options = append(options, WithComment(strings.TrimSpace(field.InlineComment.Message())))
			}

			fields = append(fields, NewField(
				field.Name,
				field.Type,
				options...,
			))

		case *proto.NormalField:
			options := make([]FieldOption, 0)

			for _, opt := range field.Options {
				options = append(options, WithAttribute(opt.Name, opt.Constant.Source))
			}

			if field.InlineComment != nil {
				options = append(options, WithComment(strings.TrimSpace(field.InlineComment.Message())))
			}

			if field.Optional {
				options = append(options, WithIsOptional())
			}

			if field.Repeated {
				options = append(options, WithIsRepeated())
			}

			fields = append(fields, NewField(
				field.Name,
				field.Type,
				options...,
			))

		default:
			return fmt.Errorf("%w: %#v", ErrUnknownProtobufField, field)
		}
	}

	model := NewModel(
		inputContext.FileId,
		inputContext.Package,
		path,
		message.Name,
		fields,
	)

	state.models[model.FullName()] = model

	return nil
}

func (p *ParserImpl) processEnum(
	_ context.Context,
	state *State,
	inputContext *InputContext,
	path []string,
	enum *proto.Enum,
) error {
	entries := make([]EnumEntry, 0)

	for _, elementField := range enum.Elements {
		switch field := elementField.(type) {
		case *proto.Comment:
			// skip

		case *proto.EnumField:
			comment := ""

			if field.InlineComment != nil {
				comment = strings.TrimSpace(field.InlineComment.Message())
			}

			entries = append(entries, NewEnumEntry(field.Name, field.Integer, comment))

		default:
			return fmt.Errorf("%w: %#v", ErrUnknownProtobufEnumField, field)
		}
	}

	enumModel := NewEnum(
		inputContext.FileId,
		inputContext.Package,
		path,
		enum.Name,
		entries,
	)

	state.enums[enumModel.FullName()] = enumModel

	return nil
}
