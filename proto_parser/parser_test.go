package proto_parser

import (
	"bytes"
	"errors"
	"io"
	"testing"

	"github.com/stretchr/testify/require"
)

var errTest = errors.New("test error")

type testInput struct {
	source io.ReadCloser
	error  error
}

func newTestInput(source io.ReadCloser, err error) *testInput {
	return &testInput{
		source: source,
		error:  err,
	}
}

func (i *testInput) Name() string {
	return "test"
}

func (i *testInput) Source() (io.ReadCloser, error) {
	if i.error != nil {
		return nil, i.error
	} else {
		return i.source, nil
	}
}

func (i *testInput) Package() string {
	return "example"
}

var testSource = `syntax = "proto3";

package my.cool.protocol;

option go_package = "./internal/protocol";

import "google/protobuf/descriptor.proto";

extend google.protobuf.FieldOptions {
  string name = 1;
  int32 id = 2;
}

// comment

/*
multiline comment
*/

enum Color {
// comment

/*
multiline comment
*/

  COLOR_RED = 0; // comment for red
  COLOR_GREEN = 1;
}

message Child {
// comment

/*
multiline comment
*/

	string name = 1; // name comment
	int32 age = 2;
}

message Example {
  message Child {
    string name = 1 [(name) = "name", (id) = 420]; // ololo
  }

  string required_field = 1; // hello
  optional bool optional_field = 2;
  repeated int64 repeated_field = 3 [(name) = "repeated", (id) = 69];
	map<string, int32> my_map = 4 [(id) = 314]; // foo
	Child child = 5;
	reserved 6 to 10;
}`

func TestParser(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name         string
		inputs       []Input
		matchResults func(t *testing.T, results *Results)
		wantErr      error
	}{
		{
			name: "failed_to_open",
			inputs: []Input{
				newTestInput(nil, errTest),
			},
			wantErr: ErrOpenSource,
		},
		{
			name: "failed_to_parse",
			inputs: []Input{
				newTestInput(io.NopCloser(bytes.NewReader([]byte(`asd`))), nil),
			},
			wantErr: ErrParseFile,
		},
		{
			name: "success",
			inputs: []Input{
				NewBytesInput(
					"test.proto",
					[]byte(testSource),
					"protocol",
				),
			},
			matchResults: func(t *testing.T, results *Results) {
				t.Helper()

				require.Len(t, results.Enums, 1)

				if enum, ok := results.Enums["Color"]; !ok {
					require.True(t, ok)
				} else {
					require.Equal(t, 0, enum.FileId())
					require.Equal(t, "protocol", enum.Package())
					require.Equal(t, "Color", enum.Name())
					require.Equal(t, "Color", enum.FullName())
					require.Empty(t, enum.Path())

					entries := enum.Entries()
					require.Len(t, entries, 2)

					require.Equal(t, "COLOR_RED", entries[0].Name())
					require.Equal(t, 0, entries[0].Value())
					require.Equal(t, "comment for red", entries[0].Comment())

					require.Equal(t, "COLOR_GREEN", entries[1].Name())
					require.Equal(t, 1, entries[1].Value())
					require.Empty(t, entries[1].Comment())
				}

				require.Len(t, results.Models, 3)

				if model, ok := results.Models["Example"]; !ok {
					require.True(t, ok)
				} else {
					require.Equal(t, 0, model.FileId())
					require.Equal(t, "protocol", model.Package())
					require.Equal(t, "Example", model.Name())
					require.Equal(t, "Example", model.FullName())
					require.Empty(t, model.Path())

					fields := model.Fields()
					require.Len(t, fields, 5)

					{
						field := fields[0]

						require.Equal(t, "required_field", field.Name())
						require.Equal(t, "string", field.Type())
						require.Empty(t, field.AdditionalType())
						require.False(t, field.IsMap())
						require.False(t, field.IsOptional())
						require.False(t, field.IsRepeated())
						require.Equal(t, "hello", field.Comment())
						require.Empty(t, field.Attributes())
					}

					{
						field := fields[1]

						require.Equal(t, "optional_field", field.Name())
						require.Equal(t, "bool", field.Type())
						require.Empty(t, field.AdditionalType())
						require.False(t, field.IsMap())
						require.True(t, field.IsOptional())
						require.False(t, field.IsRepeated())
						require.Empty(t, field.Comment())
						require.Empty(t, field.Attributes())
					}

					{
						field := fields[2]

						require.Equal(t, "repeated_field", field.Name())
						require.Equal(t, "int64", field.Type())
						require.Empty(t, field.AdditionalType())
						require.False(t, field.IsMap())
						require.False(t, field.IsOptional())
						require.True(t, field.IsRepeated())
						require.Empty(t, field.Comment())

						attributes := field.Attributes()

						require.Len(t, attributes, 2)

						if attr, ok := attributes["(name)"]; !ok {
							require.True(t, ok)
						} else {
							require.Equal(t, "repeated", attr)
						}

						if attr, ok := attributes["(id)"]; !ok {
							require.True(t, ok)
						} else {
							require.Equal(t, "69", attr)
						}
					}

					{
						field := fields[3]

						require.Equal(t, "my_map", field.Name())
						require.Equal(t, "int32", field.Type())
						require.Equal(t, "string", field.AdditionalType())
						require.True(t, field.IsMap())
						require.False(t, field.IsOptional())
						require.False(t, field.IsRepeated())
						require.Equal(t, "foo", field.Comment())

						attributes := field.Attributes()

						require.Len(t, attributes, 1)

						if attr, ok := attributes["(id)"]; !ok {
							require.True(t, ok)
						} else {
							require.Equal(t, "314", attr)
						}
					}

					{
						field := fields[4]

						require.Equal(t, "child", field.Name())
						require.Equal(t, "Child", field.Type())
						require.Empty(t, field.AdditionalType())
						require.False(t, field.IsMap())
						require.False(t, field.IsOptional())
						require.False(t, field.IsRepeated())
						require.Empty(t, field.Comment())
						require.Empty(t, field.Attributes())
					}
				}

				if model, ok := results.Models["Example__Child"]; !ok {
					require.True(t, ok)
				} else {
					require.Equal(t, 0, model.FileId())
					require.Equal(t, "protocol", model.Package())
					require.Equal(t, "Child", model.Name())
					require.Equal(t, "Example__Child", model.FullName())

					path := model.Path()

					require.Len(t, path, 1)

					require.Equal(t, "Example", path[0])

					fields := model.Fields()

					require.Len(t, fields, 1)

					{
						field := fields[0]

						require.Equal(t, "name", field.Name())
						require.Equal(t, "string", field.Type())
						require.Empty(t, field.AdditionalType())
						require.False(t, field.IsMap())
						require.False(t, field.IsOptional())
						require.False(t, field.IsRepeated())
						require.Equal(t, "ololo", field.Comment())

						attributes := field.Attributes()

						require.Len(t, attributes, 2)

						if attr, ok := attributes["(name)"]; !ok {
							require.True(t, ok)
						} else {
							require.Equal(t, "name", attr)
						}

						if attr, ok := attributes["(id)"]; !ok {
							require.True(t, ok)
						} else {
							require.Equal(t, "420", attr)
						}
					}
				}
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			parser := New()
			result, err := parser.Parse(t.Context(), testCase.inputs)

			if testCase.wantErr != nil {
				require.ErrorIs(t, err, testCase.wantErr)
			} else {
				require.NoError(t, err)
			}

			if testCase.matchResults != nil {
				testCase.matchResults(t, result)
			}
		})
	}
}
