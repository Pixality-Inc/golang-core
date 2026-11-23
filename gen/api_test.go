package gen

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/emptypb"
	"sigs.k8s.io/yaml"
)

func TestGenerateAPIRequestBodyRequired(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	protoPath := filepath.Join(dir, "protocol.proto")
	require.NoError(t, os.WriteFile(protoPath, []byte(`syntax = "proto3"; message Empty {}`), 0o600))

	var schema ApiSchema
	require.NoError(t, yaml.Unmarshal([]byte(`
api:
  routes:
    /required:
      post:
        id: Required
        requestModel: protocol.Empty
    /optional:
      post:
        id: Optional
        requestModel: protocol.Empty
        requestBodyRequired: false
    /raw:
      post:
        id: Raw
        rawBody: true
        requestBodyRequired: true
`), &schema))

	generator := NewGen(NewConfig("", dir, dir, dir, dir, dir, protoPath, "protocol", "api", nil, nil))
	model := &emptypb.Empty{}
	require.NoError(t, generator.generateApi(t.Context(), &schema, nil, ApiModels{
		"protocol.Empty": NewApiModel(model, model),
	}))

	spec, err := openapi3.NewLoader().LoadFromFile(filepath.Join(dir, "swagger.yaml"))
	require.NoError(t, err)
	require.True(t, spec.Paths.Value("/required").Post.RequestBody.Value.Required)
	require.False(t, spec.Paths.Value("/optional").Post.RequestBody.Value.Required)
	require.True(t, spec.Paths.Value("/raw").Post.RequestBody.Value.Required)

	require.NoError(t, os.Mkdir(filepath.Join(dir, "controllers"), 0o700))
	require.NoError(t, generator.generateWeb(t.Context(), &schema, nil, nil))

	router, err := os.ReadFile(filepath.Join(dir, "request_handler_gen.go"))
	require.NoError(t, err)

	_, handlers, found := strings.Cut(string(router), "func handleOptional(")
	require.True(t, found)
	optional, handlers, found := strings.Cut(handlers, "func handleRaw(")
	require.True(t, found)
	raw, required, found := strings.Cut(handlers, "func handleRequired(")
	require.True(t, found)

	require.Contains(t, optional, "if len(ctx.Request.Body()) > 0 {")
	require.Contains(t, optional, "http.ReadBody")
	require.Contains(t, raw, "request body is required")
	require.Contains(t, required, "request body is required")
	require.Contains(t, required, "http.ReadBody")
}

func TestGenerateAPIRejectsRequiredBodyWithoutDefinition(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	protoPath := filepath.Join(dir, "protocol.proto")
	require.NoError(t, os.WriteFile(protoPath, []byte(`syntax = "proto3";`), 0o600))

	var schema ApiSchema
	require.NoError(t, yaml.Unmarshal([]byte(`
api:
  routes:
    /missing:
      post:
        id: Missing
        requestBodyRequired: true
`), &schema))

	generator := NewGen(NewConfig("", dir, dir, dir, dir, dir, protoPath, "protocol", "api", nil, nil))
	err := generator.generateApi(t.Context(), &schema, nil, nil)
	require.ErrorContains(t, err, "requestBodyRequired")
	require.ErrorContains(t, err, "/missing")
	require.ErrorIs(t, generator.generateWeb(t.Context(), &schema, nil, nil), errRequiredBodyWithoutDefinition)
}
