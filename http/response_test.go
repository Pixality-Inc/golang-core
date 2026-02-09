package http

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/valyala/fasthttp"
)

func TestParseAcceptHeader(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		header string
		want   []acceptRange
	}{
		{
			name:   "empty header",
			header: "",
			want:   []acceptRange{{mediaType: "", q: 1.0}},
		},
		{
			name:   "single type without params",
			header: "application/json",
			want:   []acceptRange{{mediaType: "application/json", q: 1.0}},
		},
		{
			name:   "single type with q param",
			header: "*/*; q=0.5",
			want:   []acceptRange{{mediaType: "*/*", q: 0.5}},
		},
		{
			name:   "multiple types no q",
			header: "application/json, application/xml",
			want: []acceptRange{
				{mediaType: "application/json", q: 1.0},
				{mediaType: "application/xml", q: 1.0},
			},
		},
		{
			name:   "stripe accept header",
			header: "*/*; q=0.5, application/xml",
			want: []acceptRange{
				{mediaType: "*/*", q: 0.5},
				{mediaType: "application/xml", q: 1.0},
			},
		},
		{
			name:   "complex header with multiple q values",
			header: "application/json; q=0.9, application/protobuf; q=1.0, */*; q=0.1",
			want: []acceptRange{
				{mediaType: "application/json", q: 0.9},
				{mediaType: "application/protobuf", q: 1.0},
				{mediaType: "*/*", q: 0.1},
			},
		},
		{
			name:   "trailing comma ignored",
			header: "application/json,",
			want: []acceptRange{
				{mediaType: "application/json", q: 1.0},
			},
		},
		{
			name:   "whitespace around parts",
			header: "  application/json , application/xml  ",
			want: []acceptRange{
				{mediaType: "application/json", q: 1.0},
				{mediaType: "application/xml", q: 1.0},
			},
		},
		{
			name:   "invalid q value falls back to 1.0",
			header: "application/json; q=abc",
			want: []acceptRange{
				{mediaType: "application/json", q: 1.0},
			},
		},
		{
			name:   "q out of range falls back to 1.0",
			header: "application/json; q=2.0",
			want: []acceptRange{
				{mediaType: "application/json", q: 1.0},
			},
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			got := parseAcceptHeader(testCase.header)
			assert.Equal(t, testCase.want, got)
		})
	}
}

func TestMediaTypeToFormat(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		mediaType string
		want      dataFormatType
	}{
		{name: "json", mediaType: "application/json", want: DataFormatJson},
		{name: "protobuf", mediaType: "application/protobuf", want: DataFormatProtobuf},
		{name: "x-protobuf", mediaType: "application/x-protobuf", want: DataFormatXProtobuf},
		{name: "wildcard", mediaType: "*/*", want: DataFormatJson},
		{name: "empty", mediaType: "", want: DataFormatJson},
		{name: "unknown xml", mediaType: "application/xml", want: DataFormatUnknown},
		{name: "unknown text", mediaType: "text/html", want: DataFormatUnknown},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			got := mediaTypeToFormat(testCase.mediaType)
			assert.Equal(t, testCase.want, got)
		})
	}
}

func TestGetAcceptFormat(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		headerName string
		headerVal  string
		wantFmt    dataFormatType
		wantErr    bool
	}{
		{
			name:       "empty accept defaults to json",
			headerName: "Accept",
			headerVal:  "",
			wantFmt:    DataFormatJson,
		},
		{
			name:       "exact json",
			headerName: "Accept",
			headerVal:  "application/json",
			wantFmt:    DataFormatJson,
		},
		{
			name:       "exact protobuf",
			headerName: "Accept",
			headerVal:  "application/protobuf",
			wantFmt:    DataFormatProtobuf,
		},
		{
			name:       "exact x-protobuf",
			headerName: "Accept",
			headerVal:  "application/x-protobuf",
			wantFmt:    DataFormatXProtobuf,
		},
		{
			name:       "wildcard",
			headerName: "Accept",
			headerVal:  "*/*",
			wantFmt:    DataFormatJson,
		},
		{
			name:       "stripe header - wildcard low q then xml",
			headerName: "Accept",
			headerVal:  "*/*; q=0.5, application/xml",
			wantFmt:    DataFormatJson,
		},
		{
			name:       "xml first then wildcard - should fallback to wildcard",
			headerName: "Accept",
			headerVal:  "application/xml, */*; q=0.5",
			wantFmt:    DataFormatJson,
		},
		{
			name:       "json preferred over wildcard by q",
			headerName: "Accept",
			headerVal:  "*/*; q=0.1, application/json; q=0.9",
			wantFmt:    DataFormatJson,
		},
		{
			name:       "protobuf highest q among mixed types",
			headerName: "Accept",
			headerVal:  "application/json; q=0.5, application/protobuf; q=1.0, */*; q=0.1",
			wantFmt:    DataFormatProtobuf,
		},
		{
			name:       "q=0 means not acceptable - skip json, pick protobuf",
			headerName: "Accept",
			headerVal:  "application/json; q=0, application/protobuf",
			wantFmt:    DataFormatProtobuf,
		},
		{
			name:       "all supported types with q=0 - error",
			headerName: "Accept",
			headerVal:  "application/json; q=0, application/protobuf; q=0",
			wantFmt:    DataFormatUnknown,
			wantErr:    true,
		},
		{
			name:       "all unsupported types",
			headerName: "Accept",
			headerVal:  "application/xml, text/html",
			wantFmt:    DataFormatUnknown,
			wantErr:    true,
		},
		{
			name:       "content-type json",
			headerName: "Content-Type",
			headerVal:  "application/json",
			wantFmt:    DataFormatJson,
		},
		{
			name:       "content-type with charset param",
			headerName: "Content-Type",
			headerVal:  "application/json; charset=utf-8",
			wantFmt:    DataFormatJson,
		},
		{
			name:       "case insensitive",
			headerName: "Accept",
			headerVal:  "Application/JSON",
			wantFmt:    DataFormatJson,
		},
		{
			name:       "equal q preserves order - json first",
			headerName: "Accept",
			headerVal:  "application/json, application/protobuf",
			wantFmt:    DataFormatJson,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			ctx := &fasthttp.RequestCtx{}
			if testCase.headerVal != "" {
				ctx.Request.Header.Set(testCase.headerName, testCase.headerVal)
			}

			got, err := getAcceptFormat(ctx, testCase.headerName)
			if testCase.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			assert.Equal(t, testCase.wantFmt, got)
		})
	}
}
