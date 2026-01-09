// nolint
package gen

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path"
	"strconv"
	"strings"

	"github.com/gobeam/stringy"
)

var (
	errSecurityNotFound           = errors.New("can't find security")
	errUnknownParameterFormat     = errors.New("unknown format for parameter in operation")
	errUnknownParameterType       = errors.New("unknown type for parameter in operation")
	errUnknownOperationType       = errors.New("unknown operation type for operation")
	errUnknownParameterIn         = errors.New("unknown parameter 'in'")
	errParameterArrayNotSupported = errors.New("parameter is an array, but only strings are supported for now")
	errNoModelGetterSet           = errors.New("no modelGetter set for parameter in operation")
)

type controllerMethod struct {
	Name                string
	RouteUrl            string
	ParametersModelName string
	ResponseModel       string
	OperationType       ApiRouteOperation
	Operation           *ApiRoute
	IsHttp              bool
}

func (g *Gen) generateWeb(ctx context.Context, apiSchema *ApiSchema, apiEnums ApiEnums, apiModels ApiModels) error {
	controllerFilename := path.Join(g.config.ApiDir, "controllers", "controller_gen.go")
	routerFilename := path.Join(g.config.ApiDir, "request_handler_gen.go")

	var controllerMethods []controllerMethod

	hasUuid := false
	hasTime := false
	hasFile := false

	for _, routeEntry := range apiSchema.Api.GetRoutes() {
		routeUrl := routeEntry.Url
		operations := routeEntry.Operations

		for _, operationEntry := range operations {
			operationType := operationEntry.Operation
			operation := operationEntry.Route

			var (
				parametersModelName string
				responseModel       string
			)

			if len(operation.Parameters) > 0 ||
				operation.RequestModel != "" ||
				len(operation.RequestFiles) > 0 ||
				len(operation.Security) > 0 {
				parametersModelName = operation.Id + "Request"
			}

			for _, modelName := range operation.ResponseModels {
				if _, ok := apiSchema.Api.Errors[modelName]; ok {
					continue
				}

				responseModel = "*" + modelName

				break
			}

			if len(operation.RequestFiles) > 0 {
				hasFile = true
			}

			for _, param := range operation.Parameters {
				switch param.Format {
				case "uuid":
					if param.Model == "" {
						hasUuid = true
					}
				case "unix_time":
					hasTime = true
				}
			}

			controllerMethods = append(controllerMethods, controllerMethod{
				Name:                operation.Id,
				RouteUrl:            routeUrl,
				ParametersModelName: parametersModelName,
				ResponseModel:       responseModel,
				OperationType:       operationType,
				Operation:           &operation,
				IsHttp:              operationEntry.Route.IsHttp,
			})
		}
	}

	// Controller

	{
		var controllerImports [][]string

		controllerImports = append(controllerImports, []string{"", "context"})

		if hasUuid {
			controllerImports = append(controllerImports, []string{"", "github.com/google/uuid"})
		}

		if hasTime {
			controllerImports = append(controllerImports, []string{"", "time"})
		}

		if hasFile {
			controllerImports = append(controllerImports, []string{"", "github.com/pixality-inc/golang-core/http"})
		}

		for _, method := range controllerMethods {
			if method.IsHttp && !hasFile {
				controllerImports = append(controllerImports, []string{"", "github.com/pixality-inc/golang-core/http"})

				break
			}
		}

		controllerImports = append(controllerImports, apiSchema.Api.ControllerImports...)

		controllerGen := generateFile("controllers", controllerImports)

		controllerGen = append(controllerGen, '\n', '\n')

		for _, method := range controllerMethods {
			if method.ParametersModelName == "" {
				continue
			}

			controllerGen = append(controllerGen, []byte(fmt.Sprintf("type %s struct {\n", method.ParametersModelName))...)

			if len(method.Operation.Security) > 0 {
				for _, security := range method.Operation.Security {
					var securityType string

					if sec, ok := apiSchema.Api.Security[security]; !ok {
						return fmt.Errorf("%w: '%s'", errSecurityNotFound, security)
					} else {
						securityType = sec.Model
					}

					controllerGen = append(controllerGen, []byte(fmt.Sprintf("  Security %s\n", securityType))...)
				}
			}

			if method.Operation.RequestModel != "" {
				controllerGen = append(controllerGen, []byte(fmt.Sprintf("  Request *%s\n", method.Operation.RequestModel))...)
			}

			if len(method.Operation.RequestFiles) > 0 {
				for _, fileEntry := range method.Operation.RequestFiles {
					variableName := stringy.New(fileEntry.Name).CamelCase().UcFirst()

					controllerGen = append(controllerGen, []byte(fmt.Sprintf("  %s *http.File\n", variableName))...)
				}
			}

			for _, paramEntry := range method.Operation.GetParameters() {
				paramName := paramEntry.Name
				param := paramEntry.Parameter

				name := stringy.New(paramName).SnakeCase().CamelCase().UcFirst()

				var typeStr string

				if param.Model != "" {
					typeStr = param.Model
				} else if param.Format != "" {
					switch param.Format {
					case "uint64":
						typeStr = "uint64"
					case "uuid":
						typeStr = "uuid.UUID"
					case "unix_time":
						typeStr = "time.Time"
					default:
						return fmt.Errorf("%w: '%s' for '%s' in '%s'", errUnknownParameterFormat, param.Format, paramName, method.Operation.Id)
					}
				} else {
					switch param.Type {
					case "string":
						typeStr = "string"
					default:
						return fmt.Errorf("%w: '%s' for '%s' in '%s'", errUnknownParameterType, param.Type, paramName, method.Operation.Id)
					}
				}

				if param.Array {
					typeStr = "[]" + typeStr
				} else if !param.Required {
					typeStr = "*" + typeStr
				}

				controllerGen = append(controllerGen, []byte(fmt.Sprintf("  %s %s\n", name, typeStr))...)
			}

			controllerGen = append(controllerGen, []byte("}\n")...)
			controllerGen = append(controllerGen, '\n')
		}

		controllerGen = append(controllerGen, []byte("type Controller interface {\n")...)

		for _, method := range controllerMethods {
			methodParams := []string{
				"ctx context.Context",
			}

			if method.ParametersModelName != "" {
				methodParams = append(methodParams, "params "+method.ParametersModelName)
			}

			responseModel := method.ResponseModel

			if method.IsHttp {
				responseModel = "http.HttpResponse[" + responseModel + "]"
			}

			controllerGen = append(controllerGen, []byte(fmt.Sprintf(
				"  %s(%s) (%s, error)",
				method.Name,
				strings.Join(methodParams, ", "),
				responseModel,
			))...)

			controllerGen = append(controllerGen, '\n')
		}

		controllerGen = append(controllerGen, []byte("}\n")...)

		//nolint:gosec // G306: generated file permissions are intentionally permissive
		if err := os.WriteFile(controllerFilename, controllerGen, os.ModePerm); err != nil {
			return err
		}
	}

	// Router

	{
		var routerImports [][]string

		routerImports = append(routerImports, []string{"", "slices"})
		routerImports = append(routerImports, []string{"", "context"})
		routerImports = append(routerImports, []string{"", "github.com/valyala/fasthttp"})

		routerImports = append(routerImports, apiSchema.Api.RouterImports...)

		routerGen := generateFile(g.config.ApiPackageName, routerImports)

		routerGen = append(routerGen, '\n', '\n')

		routerGen = append(routerGen, []byte("func NewRequestHandler(\n")...)
		routerGen = append(routerGen, []byte("  ctx context.Context,\n")...)
		routerGen = append(routerGen, []byte("  router http.Router,\n")...)
		routerGen = append(routerGen, []byte("  controller controllers.Controller,\n")...)
		routerGen = append(routerGen, []byte("  middlewares ...http.Middleware,\n")...)
		routerGen = append(routerGen, []byte(") (fasthttp.RequestHandler, error) {\n")...)

		for _, method := range controllerMethods {
			var routerMethod string

			switch method.OperationType {
			case ApiRouteOperationGet:
				routerMethod = http.MethodGet
			case ApiRouteOperationPost:
				routerMethod = http.MethodPost
			case ApiRouteOperationPut:
				routerMethod = http.MethodPut
			case ApiRouteOperationPatch:
				routerMethod = http.MethodPatch
			case ApiRouteOperationDelete:
				routerMethod = http.MethodDelete
			default:
				return fmt.Errorf("%w: '%s' for '%s'", errUnknownOperationType, method.OperationType, method.Operation.Id)
			}

			handlerName := "handle" + method.Name
			routeUrl := strconv.Quote(method.RouteUrl)

			routerGen = append(routerGen, []byte(fmt.Sprintf("  router.%s(%s, handleWithController(controller, %s))\n", routerMethod, routeUrl, handlerName))...)
		}

		routerGen = append(routerGen, []byte(`
  resultHandler := router.Handle()

  slices.Reverse(middlewares)

  for _, middleware := range middlewares {
    resultHandler = middleware(resultHandler)
  }

  return resultHandler, nil
`)...)

		routerGen = append(routerGen, []byte("}\n")...)

		routerGen = append(routerGen, []byte(`
func handleWithController(controller controllers.Controller, handler func(ctx *fasthttp.RequestCtx, controller controllers.Controller)) fasthttp.RequestHandler {
  return func(ctx *fasthttp.RequestCtx) {
    handler(ctx, controller)
  }
}
`)...)
		routerGen = append(routerGen, '\n')

		for _, method := range controllerMethods {
			handlerName := "handle" + method.Name
			controllerMethodName := method.Name

			routerGen = append(routerGen, []byte(fmt.Sprintf("func %s(ctx *fasthttp.RequestCtx, controller controllers.Controller) {\n", handlerName))...)

			if method.ParametersModelName != "" {
				routerGen = append(routerGen, []byte(fmt.Sprintf("  params := controllers.%s{}\n", method.ParametersModelName))...)

				if len(method.Operation.Security) > 0 {
					routerGen = append(routerGen, '\n')

					for _, security := range method.Operation.Security {
						if sec, ok := apiSchema.Api.Security[security]; !ok {
							return fmt.Errorf("%w: '%s'", errSecurityNotFound, security)
						} else {
							routerGen = append(routerGen, []byte(fmt.Sprintf("  security, err := %s(ctx)\n", sec.ModelGetter))...)

							routerGen = append(routerGen, []byte(`  if err != nil {
    http.HandleError(ctx, err)

    return
  }
`)...)
							routerGen = append(routerGen, '\n')

							if method.Operation.AuthRequired {
								routerGen = append(routerGen, []byte(`  if security == nil {
    http.Unauthorized(ctx)

    return
  }
`)...)
								routerGen = append(routerGen, '\n')
							}

							routerGen = append(routerGen, []byte(`  params.Security = security`)...)
							routerGen = append(routerGen, '\n')
						}
					}
				}

				if len(method.Operation.Parameters) > 0 {
					routerGen = append(routerGen, '\n')

					for _, paramEntry := range method.Operation.GetParameters() {
						paramName := paramEntry.Name
						param := paramEntry.Parameter

						modelParamName := stringy.New(paramName).SnakeCase().CamelCase().UcFirst()

						routerGen = append(routerGen, '\n')

						paramNameStr := strconv.Quote(paramName)

						routerGen = append(routerGen, []byte(`  {
`)...)

						switch param.In {
						case ApiRouteParameterInPath:
							if param.Required {
								routerGen = append(routerGen, []byte(fmt.Sprintf(`    param, ok := ctx.UserValue(%s).(string)
`, paramNameStr))...)

								routerGen = append(routerGen, []byte(fmt.Sprintf(`    if !ok {
      http.HandleError(ctx, fmt.Errorf("%%w: no required parameter '%s' found", http.ErrBadRequest))

      return
    }
`, paramName))...)
							} else {
								routerGen = append(routerGen, []byte(fmt.Sprintf(`    param, _ := ctx.UserValue(%s).(string)
`, paramNameStr))...)
							}
						case ApiRouteParameterInQuery:
							routerGen = append(routerGen, []byte(fmt.Sprintf(`    param := string(ctx.FormValue(%s))
`, paramNameStr))...)
						default:
							return fmt.Errorf("%w: '%s' in '%s'", errUnknownParameterIn, paramName, param.In)
						}

						routerGen = append(routerGen, '\n')

						routerGen = append(routerGen, []byte(`    param = strings.TrimSpace(param)

`)...)

						if param.Required {
							routerGen = append(routerGen, []byte(fmt.Sprintf(`    if param == "" {
      http.HandleError(ctx, fmt.Errorf("%%w: no required parameter '%s' found", http.ErrBadRequest))

      return
    }

`, paramName))...)
						}

						routerGen = append(routerGen, []byte(`    if param != "" {
`)...)

						checkErrAndSetValue := []byte(fmt.Sprintf(`      if err != nil {
        http.HandleError(ctx, fmt.Errorf("%%w: parameter '%s' is malformed: %%w", http.ErrBadRequest, err))

        return
      }

`, paramName))

						if param.Required {
							checkErrAndSetValue = append(checkErrAndSetValue, []byte(fmt.Sprintf(`      params.%s = paramValue
`, modelParamName))...)
						} else {
							checkErrAndSetValue = append(checkErrAndSetValue, []byte(fmt.Sprintf(`      if param == "" {
        params.%s = nil
      } else {
        params.%s = &paramValue
      }
`, modelParamName, modelParamName))...)
						}

						if param.Array && (param.Model != "" || param.Format != "") {
							// @todo support other arrays
							return fmt.Errorf("%w: '%s' in '%s'", errParameterArrayNotSupported, paramName, method.Operation.Id)
						}

						if param.Model != "" {
							if param.ModelGetter == "" {
								return fmt.Errorf("%w: '%s' in '%s'", errNoModelGetterSet, paramName, method.Operation.Id)
							}

							routerGen = append(routerGen, []byte(fmt.Sprintf(`      paramValue, err := %s(param)
`, param.ModelGetter))...)
							routerGen = append(routerGen, checkErrAndSetValue...)
						} else if param.Format != "" {
							switch param.Format {
							case "uuid":
								routerGen = append(routerGen, []byte(`      paramValue, err := http.ParseUUID(param)
`)...)
								routerGen = append(routerGen, checkErrAndSetValue...)
							case "unix_time":
								routerGen = append(routerGen, []byte(`      paramValue, err := http.ParseUnixTime(param)
`)...)
								routerGen = append(routerGen, checkErrAndSetValue...)
							case "uint64":
								routerGen = append(routerGen, []byte(`      paramValue, err := http.ParseUint64(param)
`)...)
								routerGen = append(routerGen, checkErrAndSetValue...)
							default:
								return fmt.Errorf("%w: '%s' format '%s' in '%s'", errUnknownParameterFormat, paramName, param.Format, method.Operation.Id)
							}
						} else {
							switch param.Type {
							case "string":
								// @todo
								if param.Array {
									routerGen = append(routerGen, []byte(`      var paramValue []string

      parts := strings.Split(param, ",")

      for _, part := range parts {
        part = strings.TrimSpace(part)

        if part == "" {
          continue
        }

        paramValue = append(paramValue, part)
      }
`)...)
									routerGen = append(routerGen, []byte(fmt.Sprintf(`      params.%s = paramValue
`, modelParamName))...)
								} else if param.Required {
									routerGen = append(routerGen, []byte(fmt.Sprintf(`      params.%s = param
`, modelParamName))...)
								} else {
									routerGen = append(routerGen, []byte(fmt.Sprintf(`      if param == "" {
        params.%s = nil
      } else {
        params.%s = &param
      }
`, modelParamName, modelParamName))...)
								}
							default:
								return fmt.Errorf("%w: '%s' type '%s' in '%s'", errUnknownParameterType, paramName, param.Type, method.Operation.Id)
							}
						}

						routerGen = append(routerGen, []byte(`
    }

`)...)

						routerGen = append(routerGen, []byte(`
  }
`)...)
					}
				}

				if method.Operation.RequestModel != "" {
					routerGen = append(routerGen, '\n')

					routerGen = append(routerGen, []byte(fmt.Sprintf(`  {
    var request %s

    if err := http.ReadBody(ctx, &request); err != nil {
      http.HandleError(ctx, fmt.Errorf("%%w: invalid request body: %%w", http.ErrBadRequest, err))

      return
    }

    params.Request = &request
  }
`, method.Operation.RequestModel))...)
				}

				if len(method.Operation.RequestFiles) > 0 {
					for _, fileEntry := range method.Operation.RequestFiles {
						variableName := stringy.New(fileEntry.Name).CamelCase().UcFirst()

						routerGen = append(routerGen, '\n')

						paramFileName := strconv.Quote(fileEntry.Name)

						routerGen = append(routerGen, []byte(fmt.Sprintf(`  {
    fileHeader, err := ctx.FormFile(%s)
    if err != nil {
      http.HandleError(ctx, fmt.Errorf("failed to get file: %%w", err))

      return
    }

    file, err := fileHeader.Open()
    if err != nil {
      http.HandleError(ctx, fmt.Errorf("failed to open file: %%w", err))

      return
    }

    defer func() {
    	if err := file.Close(); err != nil {
    		logger.GetLogger(ctx).WithError(err).Error("failed to close file")
    	}
    }()

    params.%s = http.NewFile(file, fileHeader.Filename, uint64(fileHeader.Size))
  }
`, paramFileName, variableName))...)
					}
				}

				routerGen = append(routerGen, '\n')

				routerGen = append(routerGen, []byte(fmt.Sprintf("  response, err := controller.%s(ctx, params)\n", controllerMethodName))...)
			} else {
				routerGen = append(routerGen, []byte(fmt.Sprintf("  response, err := controller.%s(ctx)\n", controllerMethodName))...)
			}

			routerGen = append(routerGen, []byte(`  if err != nil {
    http.HandleError(ctx, err)

    return
  }
`)...)

			if method.IsHttp {
				routerGen = append(routerGen, []byte(fmt.Sprintf(`	http.HandleHttp[%s](ctx, response)`, method.ResponseModel))...)
			} else {
				routerGen = append(routerGen, []byte(`	http.Ok(ctx, response)`)...)
			}

			routerGen = append(routerGen, '\n')

			routerGen = append(routerGen, []byte("}\n")...)
			routerGen = append(routerGen, '\n')
		}

		//nolint:gosec // G306: generated file permissions are intentionally permissive
		if err := os.WriteFile(routerFilename, routerGen, os.ModePerm); err != nil {
			return err
		}
	}

	return nil
}
