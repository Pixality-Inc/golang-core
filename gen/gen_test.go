package gen

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGenFields(t *testing.T) {
	t.Parallel()

	generator := NewGen(nil)

	modelRequest := &ModelRequest{
		daoName:   makeNamed("tests"),
		modelName: makeNamed("test"),
		enumsMap:  nil,
	}

	testCases := []struct {
		name             string
		field            sourceFileField
		sqlResult        string
		modelResult      string
		attributesResult []fieldAttribute
		wantErr          error
	}{
		{
			name: "int64",
			field: sourceFileField{
				Name: "number",
				Type: "int64",
			},
			sqlResult:   `"number" BIGINT NOT NULL`,
			modelResult: "Number int64 `json:\"number\" yaml:\"number\" sql:\"number\"`",
		},
		{
			name: "string",
			field: sourceFileField{
				Name: "name",
				Type: "varchar(69)",
			},
			sqlResult:   `"name" VARCHAR(69) NOT NULL`,
			modelResult: "Name string `json:\"name\" yaml:\"name\" sql:\"name\"`",
		},
		{
			name: "string_nullable",
			field: sourceFileField{
				Name:     "name",
				Type:     "varchar(69)",
				Nullable: true,
			},
			sqlResult:   `"name" VARCHAR(69) NULL`,
			modelResult: "Name *string `json:\"name\" yaml:\"name\" sql:\"name\"`",
			attributesResult: []fieldAttribute{
				fieldAttributeNullable,
			},
		},
		{
			name: "array",
			field: sourceFileField{
				Name:  "path",
				Type:  "uuid",
				Array: true,
			},
			sqlResult:   `"path" UUID[] NOT NULL`,
			modelResult: "Path []uuid.UUID `json:\"path\" yaml:\"path\" sql:\"path\"`",
			attributesResult: []fieldAttribute{
				fieldAttributeUuid,
			},
		},
		{
			name: "array_size",
			field: sourceFileField{
				Name:      "path",
				Type:      "uuid",
				Array:     true,
				ArraySize: 1,
			},
			sqlResult:   `"path" UUID[1] NOT NULL`,
			modelResult: "Path [1]uuid.UUID `json:\"path\" yaml:\"path\" sql:\"path\"`",
			attributesResult: []fieldAttribute{
				fieldAttributeUuid,
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			field, err := generator.generateModelField(modelRequest, testCase.field)

			if testCase.wantErr != nil {
				require.ErrorIs(t, err, testCase.wantErr)
			} else {
				require.NoError(t, err)
			}

			require.Equal(t, testCase.sqlResult, string(field.sqlField.render(0, 0)))
			require.Equal(t, testCase.modelResult, string(field.modelField.render(0, 0)))
			require.Equal(t, testCase.attributesResult, field.attributes)
		})
	}
}
