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

var regressionSource = `syntax = "proto3";

package regression;

message Book {
  option deprecated = true;

  string id = 1;
}

enum BookVisibility {
  option deprecated = true;
  reserved 2;

  BOOK_VISIBILITY_UNKNOWN = 0;
  BOOK_VISIBILITY_PUBLIC = 1;
}

message Library {
  enum ShelfState {
    option deprecated = true;
    reserved 2;

    SHELF_STATE_UNKNOWN = 0;
    SHELF_STATE_OPEN = 1;
  }

  oneof owner {
    option deprecated = true;
    // owner id
    string user_id = 1;
    string group_id = 2;
  }
}`

var regressionExtensionsSource = `syntax = "proto2";

package regression;

message ExtensibleBook {
  optional string legacy_id = 1;

  extensions 100 to max;
}`

//nolint:maintidx
func TestParser(t *testing.T) {
	t.Parallel()

	pathSeparator := "::"

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
					require.Equal(t, "Color", GetFullName(enum, pathSeparator))
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
					require.Equal(t, "Example", GetFullName(model, pathSeparator))
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

				if model, ok := results.Models["Example"+pathSeparator+"Child"]; !ok {
					require.True(t, ok)
				} else {
					require.Equal(t, 0, model.FileId())
					require.Equal(t, "protocol", model.Package())
					require.Equal(t, "Child", model.Name())
					require.Equal(t, "Example"+pathSeparator+"Child", GetFullName(model, pathSeparator))

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

			parser := New(pathSeparator)
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

func TestParserIgnoresOptionsReservedExtensionsAndNestedEnums(t *testing.T) {
	t.Parallel()

	pathSeparator := "::"
	parser := New(pathSeparator)

	results, err := parser.Parse(t.Context(), []Input{
		NewBytesInput(
			"regression.proto",
			[]byte(regressionSource),
			"regression",
		),
		NewBytesInput(
			"regression_extensions.proto",
			[]byte(regressionExtensionsSource),
			"regression",
		),
	})
	require.NoError(t, err)

	require.Len(t, results.Models, 3)
	require.Len(t, results.Enums, 2)

	require.Contains(t, results.Models, "Book")
	book := results.Models["Book"]
	require.Equal(t, "Book", book.Name())
	require.Empty(t, book.Path())

	bookFields := book.Fields()
	require.Len(t, bookFields, 1)
	require.Equal(t, "id", bookFields[0].Name())
	require.Equal(t, "string", bookFields[0].Type())

	require.Contains(t, results.Enums, "BookVisibility")
	visibility := results.Enums["BookVisibility"]
	require.Equal(t, "BookVisibility", visibility.Name())
	require.Empty(t, visibility.Path())

	visibilityEntries := visibility.Entries()
	require.Len(t, visibilityEntries, 2)
	require.Equal(t, "BOOK_VISIBILITY_UNKNOWN", visibilityEntries[0].Name())
	require.Equal(t, 0, visibilityEntries[0].Value())
	require.Equal(t, "BOOK_VISIBILITY_PUBLIC", visibilityEntries[1].Name())
	require.Equal(t, 1, visibilityEntries[1].Value())

	require.Contains(t, results.Enums, "Library"+pathSeparator+"ShelfState")
	shelfState := results.Enums["Library"+pathSeparator+"ShelfState"]
	require.Equal(t, "ShelfState", shelfState.Name())
	require.Equal(t, []string{"Library"}, shelfState.Path())

	shelfStateEntries := shelfState.Entries()
	require.Len(t, shelfStateEntries, 2)
	require.Equal(t, "SHELF_STATE_UNKNOWN", shelfStateEntries[0].Name())
	require.Equal(t, 0, shelfStateEntries[0].Value())
	require.Equal(t, "SHELF_STATE_OPEN", shelfStateEntries[1].Name())
	require.Equal(t, 1, shelfStateEntries[1].Value())

	require.Contains(t, results.Models, "Library")
	library := results.Models["Library"]
	libraryFields := library.Fields()
	require.Len(t, libraryFields, 1)

	owner := libraryFields[0]
	require.Equal(t, "owner", owner.Name())
	require.Equal(t, "oneOf", owner.Type())
	require.True(t, owner.IsOneOf())

	ownerChildren := owner.Children()
	require.Len(t, ownerChildren, 2)
	require.Equal(t, "user_id", ownerChildren[0].Name())
	require.Equal(t, "string", ownerChildren[0].Type())
	require.Equal(t, "group_id", ownerChildren[1].Name())
	require.Equal(t, "string", ownerChildren[1].Type())

	require.Contains(t, results.Models, "ExtensibleBook")
	extensibleBook := results.Models["ExtensibleBook"]
	extensibleBookFields := extensibleBook.Fields()
	require.Len(t, extensibleBookFields, 1)
	require.Equal(t, "legacy_id", extensibleBookFields[0].Name())
	require.Equal(t, "string", extensibleBookFields[0].Type())
	require.True(t, extensibleBookFields[0].IsOptional())
}
