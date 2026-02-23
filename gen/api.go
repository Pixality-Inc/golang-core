// nolint
package gen

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path"
	"reflect"
	"slices"
	"sort"
	"strconv"
	"strings"

	"github.com/pixality-inc/golang-core/util"

	protoParser "github.com/emicklei/proto"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/gobeam/stringy"
	"github.com/gogo/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"gopkg.in/yaml.v3"
)

var (
	errUnknownSecurityType    = errors.New("unknown security type")
	errNotAStruct             = errors.New("not a struct")
	errFieldNotFound          = errors.New("field not found in model")
	errUnknownEnum            = errors.New("unknown enum")
	errKindNotSupported       = errors.New("kind is not supported")
	errNoSchemaFound          = errors.New("no schema found for model, try to run gen again")
	errOnlyOneSecurityAllowed = errors.New("only 1 security is allowed for operation")
	errUnknownRouteOperation  = errors.New("unknown route operation")
	errUnknownFieldType       = errors.New("unknown field type")
)

type ProtoField interface {
	IsRequired() bool
	IsOptional() bool
	IsRepeated() bool
	IsMap() bool
	IsOneOf() bool
	IsOneOfField() bool
}

type NormalProtoField struct {
	normalField *protoParser.NormalField
}

func NewProtoFieldFromNormalField(normalField *protoParser.NormalField) ProtoField {
	return &NormalProtoField{
		normalField: normalField,
	}
}

func (f *NormalProtoField) IsRequired() bool {
	return f.normalField.Required
}

func (f *NormalProtoField) IsOptional() bool {
	return f.normalField.Optional
}

func (f *NormalProtoField) IsRepeated() bool {
	return f.normalField.Repeated
}

func (f *NormalProtoField) IsMap() bool {
	return false
}

func (f *NormalProtoField) IsOneOf() bool {
	return false
}

func (f *NormalProtoField) IsOneOfField() bool {
	return false
}

type MapProtoField struct {
	mapField *protoParser.MapField
}

func (f *MapProtoField) IsRequired() bool {
	return true
}

func (f *MapProtoField) IsOptional() bool {
	return false
}

func (f *MapProtoField) IsRepeated() bool {
	return false
}

func (f *MapProtoField) IsMap() bool {
	return true
}

func (f *MapProtoField) IsOneOf() bool {
	return false
}

func (f *MapProtoField) IsOneOfField() bool {
	return false
}

func NewProtoFieldFromMapField(mapField *protoParser.MapField) ProtoField {
	return &MapProtoField{
		mapField: mapField,
	}
}

type OneOfProtoField struct {
	oneOf *protoParser.Oneof
}

func (f *OneOfProtoField) IsRequired() bool {
	return true
}

func (f *OneOfProtoField) IsOptional() bool {
	return false
}

func (f *OneOfProtoField) IsRepeated() bool {
	return false
}

func (f *OneOfProtoField) IsMap() bool {
	return false
}

func (f *OneOfProtoField) IsOneOf() bool {
	return true
}

func (f *OneOfProtoField) IsOneOfField() bool {
	return false
}

func NewProtoFieldFromOneOf(oneOf *protoParser.Oneof) ProtoField {
	return &OneOfProtoField{
		oneOf: oneOf,
	}
}

type OneOfFieldProtoField struct {
	oneOfField *protoParser.OneOfField
}

func (f *OneOfFieldProtoField) IsRequired() bool {
	return true
}

func (f *OneOfFieldProtoField) IsOptional() bool {
	return false
}

func (f *OneOfFieldProtoField) IsRepeated() bool {
	return false
}

func (f *OneOfFieldProtoField) IsMap() bool {
	return false
}

func (f *OneOfFieldProtoField) IsOneOf() bool {
	return false
}

func (f *OneOfFieldProtoField) IsOneOfField() bool {
	return true
}

func NewProtoFieldFromOneOfField(oneOfField *protoParser.OneOfField) ProtoField {
	return &OneOfFieldProtoField{
		oneOfField: oneOfField,
	}
}

type ProtoreflectDescriptor interface {
	Name() protoreflect.Name
	Parent() protoreflect.Descriptor
}

type ProtoreflectFile interface {
	IsPlaceholder() bool
}

type ApiInfo struct {
	Title       string `json:"title"       yaml:"title"`
	Description string `json:"description" yaml:"description"`
	Version     string `json:"version"     yaml:"version"`
}

type ApiServer struct {
	Url         string `json:"url"         yaml:"url"`
	Description string `json:"description" yaml:"description"`
}

type ApiSecurityType string

const (
	ApiSecurityTypeBearer ApiSecurityType = "bearer"
)

type ApiSecurity struct {
	Type        ApiSecurityType `json:"type"        yaml:"type"`
	Model       string          `json:"model"       yaml:"model"`
	ModelGetter string          `json:"modelGetter" yaml:"modelGetter"`
}

type ApiImport = []string

type ApiRouteOperation string

const (
	ApiRouteOperationGet    ApiRouteOperation = "get"
	ApiRouteOperationPost   ApiRouteOperation = "post"
	ApiRouteOperationPut    ApiRouteOperation = "put"
	ApiRouteOperationPatch  ApiRouteOperation = "patch"
	ApiRouteOperationDelete ApiRouteOperation = "delete"
)

type ApiRouteParameterIn string

const (
	ApiRouteParameterInPath   ApiRouteParameterIn = "path"
	ApiRouteParameterInQuery  ApiRouteParameterIn = "query"
	ApiRouteParameterInHeader ApiRouteParameterIn = "header"
)

type ApiRouteParameterEnumValue struct {
	Name  string `json:"name"  yaml:"name"`
	Value string `json:"value" yaml:"value"`
}

type ApiRouteParameter struct {
	In          ApiRouteParameterIn          `json:"in"          yaml:"in"`
	Type        string                       `json:"type"        yaml:"type"`
	Array       bool                         `json:"array"       yaml:"array"`
	Format      string                       `json:"format"      yaml:"format"`
	Model       string                       `json:"model"       yaml:"model"`
	ModelGetter string                       `json:"modelGetter" yaml:"modelGetter"`
	Required    bool                         `json:"required"    yaml:"required"`
	Description string                       `json:"description" yaml:"description"`
	Enum        []ApiRouteParameterEnumValue `json:"enum"        yaml:"enum"`
}

type ApiRouteRequestFile struct {
	Name string `json:"name" yaml:"name"`
}

type ApiRouteParameterEntry struct {
	Name      string
	Parameter ApiRouteParameter
}

type ApiRoute struct {
	Id             string                       `json:"id"             yaml:"id"`
	Hidden         bool                         `json:"hidden"         yaml:"hidden"`
	Tags           []string                     `json:"tags"           yaml:"tags"`
	Title          string                       `json:"title"          yaml:"title"`
	Description    string                       `json:"description"    yaml:"description"`
	Security       []string                     `json:"security"       yaml:"security"`
	AuthRequired   bool                         `json:"authRequired"   yaml:"authRequired"`
	IsHttp         bool                         `json:"http"           yaml:"http"`
	Parameters     map[string]ApiRouteParameter `json:"parameters"     yaml:"parameters"`
	RequestFiles   []ApiRouteRequestFile        `json:"requestFiles"   yaml:"requestFiles"`
	RequestModel   string                       `json:"requestModel"   yaml:"requestModel"`
	ResponseModels []string                     `json:"responseModels" yaml:"responseModels"`
	RawBody        bool                         `json:"rawBody"        yaml:"rawBody"`
	RawHeaders     bool                         `json:"rawHeaders"     yaml:"rawHeaders"`
}

func (ar *ApiRoute) GetParameters() []ApiRouteParameterEntry {
	result := make([]ApiRouteParameterEntry, 0)

	for name, param := range ar.Parameters {
		result = append(result, ApiRouteParameterEntry{
			Name:      name,
			Parameter: param,
		})
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].Name < result[j].Name
	})

	return result
}

type ApiRouteEntry struct {
	Url        string
	Operations []ApiRouteOperationEntry
}

type ApiRouteOperationEntry struct {
	Operation ApiRouteOperation
	Route     ApiRoute
}

type Api struct {
	Info              ApiInfo                                   `json:"info"              yaml:"info"`
	Servers           []ApiServer                               `json:"servers"           yaml:"servers"`
	Security          map[string]ApiSecurity                    `json:"security"          yaml:"security"`
	Imports           []ApiImport                               `json:"imports"           yaml:"imports"`
	ControllerImports []ApiImport                               `json:"controllerImports" yaml:"controllerImports"`
	RouterImports     []ApiImport                               `json:"routerImports"     yaml:"routerImports"`
	ServerImports     []ApiImport                               `json:"serverImports"     yaml:"serverImports"`
	Routes            map[string]map[ApiRouteOperation]ApiRoute `json:"routes"            yaml:"routes"`
	Errors            map[string]string                         `json:"errors"            yaml:"errors"`
}

func (a *Api) GetRoutes() []ApiRouteEntry {
	routes := make([]ApiRouteEntry, 0)

	for url, routeOperations := range a.Routes {
		var operations []ApiRouteOperationEntry

		for operation, route := range routeOperations {
			operations = append(operations, ApiRouteOperationEntry{
				Operation: operation,
				Route:     route,
			})
		}

		sort.Slice(operations, func(i, j int) bool {
			return operations[i].Operation < operations[j].Operation
		})

		routes = append(routes, ApiRouteEntry{
			Url:        url,
			Operations: operations,
		})
	}

	sort.Slice(routes, func(i, j int) bool {
		return routes[i].Url < routes[j].Url
	})

	return routes
}

type ApiSchema struct {
	Api Api `json:"api" yaml:"api"`
}

type ApiModel struct {
	Message proto.Message
	Reflect protoreflect.ProtoMessage
}

func NewApiModel(protoMessage proto.Message, protoreflectMessage protoreflect.ProtoMessage) ApiModel {
	return ApiModel{
		Message: protoMessage,
		Reflect: protoreflectMessage,
	}
}

type ApiEnumValueEntry struct {
	Name  string
	Value int
}

type ApiEnum struct {
	Name   string
	Values map[string]int
}

func NewApiEnum(name string, values map[string]int) ApiEnum {
	return ApiEnum{
		Name:   name,
		Values: values,
	}
}

func (ae *ApiEnum) GetValues() []ApiEnumValueEntry {
	result := make([]ApiEnumValueEntry, 0)

	for name, value := range ae.Values {
		result = append(result, ApiEnumValueEntry{
			Name:  name,
			Value: value,
		})
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].Name < result[j].Name
	})

	return result
}

type ApiEnums = map[string]ApiEnum

type ApiModels = map[string]ApiModel

type TagEntry struct {
	Name string
	Tag  *openapi3.Tag
}

func (g *Gen) generateApi(ctx context.Context, apiSchema *ApiSchema, apiEnums ApiEnums, apiModels ApiModels) error {
	log := g.log.GetLogger(ctx)

	swaggerFilename := path.Join(g.config.DocsDir, "swagger.yaml")
	genFilename := path.Join(g.config.ApiDir, "gen.go")

	modelFields, parsedEnums, err := g.generateApiGen(ctx, genFilename, apiSchema)
	if err != nil {
		return err
	}

	for k, v := range parsedEnums {
		apiEnums[k] = v
	}

	// Base schema

	spec := &openapi3.T{
		OpenAPI: "3.0.0",
		Info: &openapi3.Info{
			Title:       apiSchema.Api.Info.Title,
			Version:     apiSchema.Api.Info.Version,
			Description: apiSchema.Api.Info.Description,
		},
		Servers: []*openapi3.Server{},
		Tags:    []*openapi3.Tag{},
		Paths:   nil,
		Components: &openapi3.Components{
			Schemas:         map[string]*openapi3.SchemaRef{},
			SecuritySchemes: map[string]*openapi3.SecuritySchemeRef{},
		},
	}

	// Servers

	for _, server := range apiSchema.Api.Servers {
		spec.Servers = append(spec.Servers, &openapi3.Server{
			URL:         server.Url,
			Description: server.Description,
		})
	}

	// Security

	for securityName, security := range apiSchema.Api.Security {
		switch security.Type {
		case ApiSecurityTypeBearer:
			spec.Components.SecuritySchemes[securityName] = &openapi3.SecuritySchemeRef{
				Value: &openapi3.SecurityScheme{
					Type:        "http",
					Scheme:      "bearer",
					Description: "`Authorization: Bearer ...`",
				},
			}
		default:
			return fmt.Errorf("%w: %s", errUnknownSecurityType, security.Type)
		}
	}

	// Paths

	schemas := make(map[string]*openapi3.SchemaRef)

	var addSchema func(modelName string) (string, error)

	type PropertyExtras struct {
		Title       string
		Description string
		Format      string
	}

	applyExtras := func(schema *openapi3.Schema, extras PropertyExtras) {
		if extras.Title != "" {
			schema.Title = extras.Title
		}

		if extras.Description != "" {
			schema.Description = extras.Description
		}

		if extras.Format != "" {
			schema.Format = extras.Format
		}
	}

	makeProperty := func(propertyType string, extras PropertyExtras) *openapi3.SchemaRef {
		schema := &openapi3.Schema{
			Type: &openapi3.Types{propertyType},
		}

		applyExtras(schema, extras)

		return &openapi3.SchemaRef{
			Value: schema,
		}
	}

	stringProperty := func(extras PropertyExtras) *openapi3.SchemaRef {
		return makeProperty(openapi3.TypeString, extras)
	}

	integerProperty := func(extras PropertyExtras) *openapi3.SchemaRef {
		return makeProperty(openapi3.TypeInteger, extras)
	}

	numberProperty := func(extras PropertyExtras) *openapi3.SchemaRef {
		return makeProperty(openapi3.TypeNumber, extras)
	}

	booleanProperty := func(extras PropertyExtras) *openapi3.SchemaRef {
		return makeProperty(openapi3.TypeBoolean, extras)
	}

	enumProperty := func(enumValues []ApiEnumValueEntry, extras PropertyExtras) *openapi3.SchemaRef {
		var values []*openapi3.SchemaRef

		for _, enumValueEntry := range enumValues {
			enumKey := enumValueEntry.Name
			enumValue := enumValueEntry.Value
			title := enumKey + " = " + strconv.Itoa(enumValue)

			values = append(values, &openapi3.SchemaRef{
				Value: &openapi3.Schema{
					Default: enumValue,
					Title:   title,
				},
			})
		}

		schema := &openapi3.Schema{
			Type:  &openapi3.Types{openapi3.TypeNumber},
			OneOf: values,
		}

		applyExtras(schema, extras)

		return &openapi3.SchemaRef{
			Value: schema,
		}
	}

	refProperty := func(refName string) *openapi3.SchemaRef {
		return &openapi3.SchemaRef{
			Ref: "#/components/schemas/" + refName,
		}
	}

	objectProperty := func(
		name string,
		properties map[string]*openapi3.SchemaRef,
		requiredProperties []string,
		extras PropertyExtras,
	) *openapi3.SchemaRef {
		schema := &openapi3.Schema{
			Type:       &openapi3.Types{openapi3.TypeObject},
			Required:   requiredProperties,
			Properties: properties,
		}

		if name != "" {
			schema.Description = fmt.Sprintf("[%s](#/schemas/%s)", name, name)
		}

		applyExtras(schema, extras)

		return &openapi3.SchemaRef{
			Value: schema,
		}
	}

	arrayProperty := func(of *openapi3.SchemaRef, extras PropertyExtras) *openapi3.SchemaRef {
		schema := &openapi3.Schema{
			Type:  &openapi3.Types{openapi3.TypeArray},
			Items: of,
		}

		applyExtras(schema, extras)

		return &openapi3.SchemaRef{
			Value: schema,
		}
	}

	mapProperty := func(refName string, extras PropertyExtras) *openapi3.SchemaRef {
		schema := &openapi3.Schema{
			Type: &openapi3.Types{openapi3.TypeObject},
			AdditionalProperties: openapi3.AdditionalProperties{
				Schema: refProperty(refName),
			},
		}

		applyExtras(schema, extras)

		return &openapi3.SchemaRef{
			Value: schema,
		}
	}

	visited := make(map[string]string)

	describeModel := func(modelName string, protoModelName string, msgRef protoreflect.ProtoMessage) error {
		if _, ok := visited[modelName]; ok {
			return nil
		}

		refField := reflect.TypeOf(msgRef)

		if refField.Kind() == reflect.Ptr {
			refField = refField.Elem()
		}

		if refField.Kind() != reflect.Struct {
			return fmt.Errorf("%w: %v (%T)", errNotAStruct, refField, refField)
		}

		visited[modelName] = modelName

		model, ok := modelFields[protoModelName]
		if !ok {
			model = make(map[string]ProtoField)
		}

		protoRef := msgRef.ProtoReflect()
		descriptor := protoRef.Descriptor()
		fields := descriptor.Fields()

		objectProperties := make(map[string]*openapi3.SchemaRef)
		requiredProperties := make([]string, 0)

		isOneOfField := func(fieldDesc protoreflect.FieldDescriptor) bool {
			containingOneOf := fieldDesc.ContainingOneof()
			if containingOneOf != nil {
				containingOneOfFields := containingOneOf.Fields()
				if containingOneOfFields != nil {
					return containingOneOfFields.Len() > 1
				}
			}

			return false
		}

		oneOfs := make(map[string]map[string]protoreflect.FieldDescriptor)

		fieldToObjectProperty := func(fieldDesc protoreflect.FieldDescriptor) (*openapi3.SchemaRef, bool, error) {
			kind := fieldDesc.Kind()
			name := string(fieldDesc.Name())

			field, ok := model[name]
			if !ok {
				return nil, false, fmt.Errorf("%w: '%s' in model '%s' (%s)", errFieldNotFound, name, modelName, protoModelName)
			}

			isRepeated := field.IsRepeated()
			isMap := field.IsMap()

			extras := PropertyExtras{}

			if refFieldField, hasRefField := refField.FieldByName(name); hasRefField {
				extras.Title = refFieldField.Tag.Get("title")
				extras.Description = refFieldField.Tag.Get("description")
				extras.Format = refFieldField.Tag.Get("format")
			}

			var property *openapi3.SchemaRef

			switch kind {
			case protoreflect.MessageKind:
				msg := fieldDesc.Message()
				if isMap {
					// @todo! validate key is string only
					msg = fieldDesc.MapValue().Message()
				}

				modelNames := []string{string(msg.Name())}

				parentMsg, ok := msg.Parent().(protoreflect.MessageDescriptor)
				for ok {
					modelNames = append(modelNames, string(parentMsg.Name()))
					parentMsg, ok = parentMsg.Parent().(protoreflect.MessageDescriptor)
				}

				slices.Reverse(modelNames)

				msgName := g.config.ApiModelsPrefix + strings.Join(modelNames, "_")

				if resultName, err := addSchema(msgName); err != nil {
					return nil, false, err
				} else {
					if isMap {
						property = mapProperty(resultName, extras)
					} else {
						property = refProperty(resultName)
					}
				}

			case protoreflect.StringKind:
				property = stringProperty(extras)

			case protoreflect.BoolKind:
				property = booleanProperty(extras)

			case protoreflect.Int32Kind, protoreflect.Uint32Kind, protoreflect.Sint32Kind, protoreflect.Int64Kind, protoreflect.Uint64Kind, protoreflect.Sint64Kind:
				property = integerProperty(extras)

			case protoreflect.FloatKind, protoreflect.DoubleKind:
				property = numberProperty(extras)

			case protoreflect.EnumKind:
				enum := fieldDesc.Enum()

				enumData, ok := apiEnums[string(enum.Name())]
				if !ok {
					return nil, false, fmt.Errorf("%w: %s", errUnknownEnum, enum.Name())
				}

				property = enumProperty(enumData.GetValues(), extras)

			default:
				return nil, false, fmt.Errorf("%w: %v", errKindNotSupported, kind)
			}

			isRequired := field.IsRequired()
			isOptional := field.IsOptional()

			resultIsRequired := isRepeated || isRequired || !isOptional

			if isRepeated {
				property = arrayProperty(property, extras)
			}

			return property, resultIsRequired, nil
		}

		for i := range fields.Len() {
			fieldDesc := fields.Get(i)

			if isOneOfField(fieldDesc) {
				oneOfName := string(fieldDesc.ContainingOneof().Name())

				if _, ok := oneOfs[oneOfName]; !ok {
					oneOfs[oneOfName] = make(map[string]protoreflect.FieldDescriptor)
				}

				oneOfs[oneOfName][string(fieldDesc.Name())] = fieldDesc

				continue
			}

			property, isRequired, err := fieldToObjectProperty(fieldDesc)
			if err != nil {
				return err
			}

			objectProperties[string(fieldDesc.Name())] = property

			if isRequired {
				requiredProperties = append(requiredProperties, string(fieldDesc.Name()))
			}
		}

		for oneOfName, oneOfFields := range oneOfs {
			schemaFields := make(map[string]*openapi3.SchemaRef, len(oneOfFields))

			for _, oneOfField := range oneOfFields {
				property, _, err := fieldToObjectProperty(oneOfField)
				if err != nil {
					return err
				}

				schemaFields[string(oneOfField.Name())] = property
			}

			schemaRefs := make([]*openapi3.SchemaRef, 0, len(schemaFields))

			for fieldName, property := range schemaFields {
				schemaRefs = append(schemaRefs, &openapi3.SchemaRef{
					Value: &openapi3.Schema{
						Type:        &openapi3.Types{openapi3.TypeObject},
						Description: fieldName,
						Properties: map[string]*openapi3.SchemaRef{
							fieldName: property,
						},
					},
				})
			}

			objectProperties[oneOfName] = &openapi3.SchemaRef{
				Value: &openapi3.Schema{
					OneOf: schemaRefs,
				},
			}
		}

		extras := PropertyExtras{}

		schemas[modelName] = objectProperty(modelName, objectProperties, requiredProperties, extras)

		return nil
	}

	addSchema = func(modelName string) (string, error) {
		if errorModel, ok := apiSchema.Api.Errors[modelName]; ok {
			return addSchema(errorModel)
		}

		split := strings.Split(modelName, ".")
		modelSlag := split[len(split)-1]

		modelNameStr := stringy.New(modelSlag)
		modelNameSnake := modelNameStr.SnakeCase().ToLower()

		if _, has := schemas[modelNameSnake]; has {
			return modelNameSnake, nil
		}

		if apiModel, ok := apiModels[modelName]; !ok {
			return "", fmt.Errorf("%w: '%s' (%s)", errNoSchemaFound, modelName, modelNameSnake)
		} else {
			log.Infof("Adding schema component '%s' for model %s", modelNameSnake, modelName)

			if err = describeModel(modelNameSnake, modelName, apiModel.Reflect); err != nil {
				return "", err
			} else {
				return modelNameSnake, nil
			}
		}
	}

	tags := make(map[string]*openapi3.Tag)

	paths := make(map[string]*openapi3.PathItem)

	for _, routeEntry := range apiSchema.Api.GetRoutes() {
		routeUri := routeEntry.Url
		operations := routeEntry.Operations

		hasItems := false
		pathItem := &openapi3.PathItem{}

		for _, operationEntry := range operations {
			routeOperation := operationEntry.Operation
			operation := operationEntry.Route

			if operation.Hidden {
				continue
			}

			hasItems = true

			for _, tag := range operation.Tags {
				if _, has := tags[tag]; !has {
					tags[tag] = &openapi3.Tag{
						Name: tag,
					}
				}
			}

			pathOperation := &openapi3.Operation{
				Summary:     operation.Title,
				Description: operation.Description,
				Tags:        operation.Tags,
				OperationID: operation.Id,
				Security:    nil,
			}

			if operation.Security != nil {
				if len(operation.Security) > 1 {
					return fmt.Errorf("%w: '%s'", errOnlyOneSecurityAllowed, operation.Id)
				}

				pathOperation.Security = &openapi3.SecurityRequirements{openapi3.SecurityRequirement{
					operation.Security[0]: []string{},
				}}
			}

			var parameters []*openapi3.ParameterRef

			for _, paramEntry := range operation.GetParameters() {
				paramName := paramEntry.Name
				param := paramEntry.Parameter

				schema := &openapi3.SchemaRef{
					Value: &openapi3.Schema{
						Type:   &openapi3.Types{param.Type},
						Format: param.Format,
					},
				}

				if len(param.Enum) > 0 {
					var enumValues []any

					for _, enumValue := range param.Enum {
						enumValues = append(enumValues, enumValue.Value)
					}

					schema.Value.Enum = enumValues
				}

				parameters = append(parameters, &openapi3.ParameterRef{
					Value: &openapi3.Parameter{
						Name:        paramName,
						In:          string(param.In),
						Description: param.Description,
						Required:    param.Required,
						Schema:      schema,
					},
				})
			}

			pathOperation.Parameters = parameters

			if len(operation.RequestFiles) > 0 {
				filesMap := make(map[string]*openapi3.SchemaRef, len(operation.RequestFiles))
				requiredProperties := make([]string, 0, len(operation.RequestFiles))

				for _, fileEntry := range operation.RequestFiles {
					filesMap[fileEntry.Name] = &openapi3.SchemaRef{
						Value: &openapi3.Schema{
							Type:   &openapi3.Types{openapi3.TypeString},
							Format: "binary",
						},
					}

					requiredProperties = append(requiredProperties, fileEntry.Name)
				}

				pathOperation.RequestBody = &openapi3.RequestBodyRef{
					Value: &openapi3.RequestBody{
						Description: "Request File",
						Content: map[string]*openapi3.MediaType{
							"multipart/form-data": {
								Schema: objectProperty(
									"",
									filesMap,
									requiredProperties,
									PropertyExtras{},
								),
							},
						},
					},
				}
			} else if operation.RequestModel != "" {
				if requestModelName, err := addSchema(operation.RequestModel); err != nil {
					return err
				} else {
					pathOperation.RequestBody = &openapi3.RequestBodyRef{
						Value: &openapi3.RequestBody{
							Description: requestModelName,
							Content: map[string]*openapi3.MediaType{
								"application/json": {
									Schema: refProperty(requestModelName),
								},
							},
						},
					}
				}
			} else if operation.RawBody {
				pathOperation.RequestBody = &openapi3.RequestBodyRef{
					Value: &openapi3.RequestBody{
						Description: "Raw Body",
					},
				}
			}

			var responses []openapi3.NewResponsesOption

			for _, responseModel := range operation.ResponseModels {
				if responseModelName, err := addSchema(responseModel); err != nil {
					return err
				} else {
					responseRef := &openapi3.ResponseRef{
						Value: &openapi3.Response{
							Description: util.MakeRef(responseModelName),
							Headers:     nil,
							Content: map[string]*openapi3.MediaType{
								"application/json": {
									Schema: refProperty(responseModelName),
								},
							},
						},
					}

					if _, ok := apiSchema.Api.Errors[responseModel]; ok {
						statusCode, err := strconv.Atoi(responseModel)
						if err != nil {
							return err
						}

						responses = append(responses, openapi3.WithStatus(statusCode, responseRef))
					} else {
						responses = append(responses, openapi3.WithStatus(200, responseRef))
					}
				}
			}

			pathOperation.Responses = openapi3.NewResponses(responses...)

			switch routeOperation {
			case ApiRouteOperationGet:
				pathItem.Get = pathOperation
			case ApiRouteOperationPost:
				pathItem.Post = pathOperation
			case ApiRouteOperationPut:
				pathItem.Put = pathOperation
			case ApiRouteOperationPatch:
				pathItem.Patch = pathOperation
			case ApiRouteOperationDelete:
				pathItem.Delete = pathOperation
			default:
				return fmt.Errorf("%w: '%s'", errUnknownRouteOperation, routeOperation)
			}
		}

		if hasItems {
			paths[routeUri] = pathItem
		}
	}

	// Schemas

	spec.Components.Schemas = schemas

	// Tags

	tagsSlice := make([]TagEntry, 0)

	for _, tag := range tags {
		tagsSlice = append(tagsSlice, TagEntry{
			Name: tag.Name,
			Tag:  tag,
		})
	}

	sort.Slice(tagsSlice, func(i, j int) bool {
		return tagsSlice[i].Name < tagsSlice[j].Name
	})

	for _, tag := range tagsSlice {
		spec.Tags = append(spec.Tags, tag.Tag)
	}

	// Paths

	pathOptions := make([]openapi3.NewPathsOption, 0)

	for pathName, pathItem := range paths {
		pathOptions = append(pathOptions, openapi3.WithPath(pathName, pathItem))
	}

	spec.Paths = openapi3.NewPaths(pathOptions...)

	// Swagger.yaml

	specBuf, err := yaml.Marshal(spec)
	if err != nil {
		return err
	}

	//nolint:gosec // G306: generated file permissions are intentionally permissive
	if err = os.WriteFile(swaggerFilename, specBuf, os.ModePerm); err != nil {
		return err
	}

	return nil
}

func (g *Gen) generateApiGen(ctx context.Context, genFilename string, apiSchema *ApiSchema) (map[string]map[string]ProtoField, ApiEnums, error) {
	log := g.log.GetLogger(ctx)

	protoFileReader, err := os.Open(g.config.ProtoFilename)
	if err != nil {
		return nil, nil, err
	}

	defer func() {
		if err := protoFileReader.Close(); err != nil {
			log.Errorf("failed to close proto %s file reader: %v", g.config.ProtoFilename, err)
		}
	}()

	parser := protoParser.NewParser(protoFileReader)

	definition, err := parser.Parse()
	if err != nil {
		return nil, nil, err
	}

	genContent := generateFile(g.config.ApiPackageName, apiSchema.Api.Imports)

	parsedEnums := make(ApiEnums)

	{
		genContent = append(genContent, '\n')
		genContent = append(genContent, '\n')

		enums := make(map[string]bool)

		addEnum := func(enumName string, enumValues map[string]int) {
			if _, has := enums[enumName]; has {
				return
			}

			enums[enumName] = true

			var enumValuesEntries []ApiEnumValueEntry

			for key, value := range enumValues {
				enumValuesEntries = append(enumValuesEntries, ApiEnumValueEntry{
					Name:  key,
					Value: value,
				})
			}

			sort.Slice(enumValuesEntries, func(i, j int) bool {
				return enumValuesEntries[i].Value < enumValuesEntries[j].Value
			})

			var enumValuesArr []string

			for _, enumValueEntry := range enumValuesEntries {
				enumValuesArr = append(enumValuesArr, `"`+enumValueEntry.Name+`": `+strconv.Itoa(enumValueEntry.Value))
			}

			enumValuesMap := "map[string]int{" + strings.Join(enumValuesArr, ", ") + "}"

			genContent = append(genContent, '\t')
			genContent = append(genContent, []byte(`"`+enumName+`": gen.NewApiEnum("`+enumName+`", `+enumValuesMap+`),`)...)
			genContent = append(genContent, '\n')
		}

		genContent = append(genContent, []byte("var ApiEnums = gen.ApiEnums{\n")...)

		protoParser.Walk(definition,
			protoParser.WithEnum(func(enum *protoParser.Enum) {
				enumValues := make(map[string]int)

				for _, element := range enum.Elements {
					enumField, ok := element.(*protoParser.EnumField)
					if !ok {
						continue
					}

					enumValues[enumField.Name] = enumField.Integer
				}

				addEnum(enum.Name, enumValues)
				parsedEnums[enum.Name] = NewApiEnum(enum.Name, enumValues)
			}),
		)

		genContent = append(genContent, []byte("}\n")...)

		genContent = append(genContent, '\n')
	}

	modelsFields := make(map[string]map[string]ProtoField)

	{
		genContent = append(genContent, []byte("var ApiModels = gen.ApiModels{\n")...)

		schemas := make(map[string]bool)

		addSchema := func(modelName string) {
			if _, has := schemas[modelName]; has {
				return
			}

			schemas[modelName] = true

			genContent = append(genContent, '\t')
			genContent = append(genContent, []byte(`"`+modelName+`": gen.NewApiModel(new(`+modelName+`), new(`+modelName+`)),`)...)
			genContent = append(genContent, '\n')
		}

		protoParser.Walk(definition,
			protoParser.WithMessage(func(msg *protoParser.Message) {
				var modelName string

				_, ok := msg.Parent.(*protoParser.Proto)
				if ok {
					modelName = g.config.ApiModelsPrefix + msg.Name
				} else {
					names := []string{msg.Name}

					parentMessage, ok := msg.Parent.(*protoParser.Message)

					for ok {
						names = append(names, parentMessage.Name)
						parentMessage, ok = parentMessage.Parent.(*protoParser.Message)
					}

					slices.Reverse(names)

					modelName = g.config.ApiModelsPrefix + strings.Join(names, "_")
				}

				addSchema(modelName)

				modelsFields[modelName] = make(map[string]ProtoField)

				for _, element := range msg.Elements {
					switch el := element.(type) {
					case *protoParser.NormalField:
						modelsFields[modelName][el.Name] = NewProtoFieldFromNormalField(el)
					case *protoParser.MapField:
						modelsFields[modelName][el.Name] = NewProtoFieldFromMapField(el)
					case *protoParser.Oneof:
						modelsFields[modelName][el.Name] = NewProtoFieldFromOneOf(el)

						for _, oneOfField := range el.Elements {
							switch field := oneOfField.(type) {
							case *protoParser.OneOfField:
								modelsFields[modelName][field.Name] = NewProtoFieldFromOneOfField(field)
							default:
								panic(fmt.Errorf("%w: %T in oneof %q of message %q", errUnknownFieldType, field, el.Name, modelName))
							}
						}
					case *protoParser.Comment:
						// skip
					case *protoParser.Option:
						// skip
					case *protoParser.Reserved:
						// skip
					case *protoParser.Message:
						// skip
					case *protoParser.Enum:
						// skip
					default:
						panic(fmt.Errorf("%w: %T in message %q", errUnknownFieldType, element, modelName))
					}
				}
			}),
		)

		genContent = append(genContent, []byte("}\n")...)
		genContent = append(genContent, '\n')
	}

	//nolint:gosec // G306: generated file permissions are intentionally permissive
	if err := os.WriteFile(genFilename, genContent, os.ModePerm); err != nil {
		return nil, nil, err
	}

	return modelsFields, parsedEnums, nil
}
