package storage

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_GetFileMetadataByName(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name                string
		filename            string
		wantContentType     string
		wantContentEncoding string
		wantErr             error
	}{
		{
			name:                "text",
			filename:            "test.txt",
			wantContentType:     "text/plain",
			wantContentEncoding: "",
		},
		{
			name:                "text gzipped",
			filename:            "test.txt.gz",
			wantContentType:     "text/plain",
			wantContentEncoding: ContentEncodingGzip,
		},
		{
			name:                "json",
			filename:            "test.json",
			wantContentType:     "application/json",
			wantContentEncoding: "",
		},
		{
			name:                "json gzipped",
			filename:            "test.json.gz",
			wantContentType:     "application/json",
			wantContentEncoding: ContentEncodingGzip,
		},
		{
			name:                "csv",
			filename:            "test.csv",
			wantContentType:     "text/csv",
			wantContentEncoding: "",
		},
		{
			name:                "csv gzipped",
			filename:            "test.csv.gz",
			wantContentType:     "text/csv",
			wantContentEncoding: ContentEncodingGzip,
		},
		{
			name:                "jpg",
			filename:            "test.jpg",
			wantContentType:     "image/jpeg",
			wantContentEncoding: "",
		},
		{
			name:                "jpeg",
			filename:            "test.jpeg",
			wantContentType:     "image/jpeg",
			wantContentEncoding: "",
		},
		{
			name:                "png",
			filename:            "test.png",
			wantContentType:     "image/png",
			wantContentEncoding: "",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			metadata, err := GetFileMetadataByName(testCase.filename)

			if testCase.wantErr != nil {
				require.ErrorIs(t, err, testCase.wantErr)
			} else {
				require.NoError(t, err)
				require.NotNil(t, metadata)

				require.Equal(t, testCase.wantContentType, metadata.ContentType())
				require.Equal(t, testCase.wantContentEncoding, metadata.ContentEncoding())
			}
		})
	}
}
