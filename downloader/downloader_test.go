package downloader

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/pixality-inc/golang-core/http_client"
	mockHttpClient "github.com/pixality-inc/golang-core/http_client/mocks"
	"github.com/pixality-inc/golang-core/logger"
)

var errNetwork = errors.New("network error")

func TestDownloader_Download(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		setupMock   func(client *mockHttpClient.MockClient, resp *mockHttpClient.MockResponse)
		ctx         context.Context // nolint:containedctx
		url         string
		expected    []byte
		expectError bool
	}{
		{
			name: "success",
			ctx:  t.Context(),
			url:  "https://example.com/file",
			setupMock: func(client *mockHttpClient.MockClient, resp *mockHttpClient.MockResponse) {
				resp.EXPECT().
					GetBody().
					Return([]byte("ok"))
				client.
					EXPECT().
					Get(gomock.Any(), "https://example.com/file").
					Return(resp, nil)
			},
			expected: []byte("ok"),
		},
		{
			name: "http_client_error",
			ctx:  t.Context(),
			url:  "https://example.com/file",
			setupMock: func(client *mockHttpClient.MockClient, _ *mockHttpClient.MockResponse) {
				client.
					EXPECT().
					Get(gomock.Any(), "https://example.com/file").
					Return(nil, errNetwork)
			},
			expectError: true,
		},
		{
			name: "context_canceled",
			ctx: func() context.Context {
				ctx, cancel := context.WithCancel(t.Context())
				cancel()

				return ctx
			}(),
			url: "https://example.com/file",
			setupMock: func(client *mockHttpClient.MockClient, _ *mockHttpClient.MockResponse) {
				client.
					EXPECT().
					Get(gomock.Any(), "https://example.com/file").
					DoAndReturn(func(ctx context.Context, _ string, _ ...http_client.RequestOption) (http.Response, error) {
						return http.Response{}, ctx.Err()
					})
			},
			expectError: true,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			httpClient := mockHttpClient.NewMockClient(ctrl)
			responseMock := mockHttpClient.NewMockResponse(ctrl)
			testCase.setupMock(httpClient, responseMock)

			log := logger.NewLoggableImplWithService("downloader")
			d := &Impl{
				httpClient: httpClient,
				log:        log,
			}

			body, err := d.Download(testCase.ctx, testCase.url)

			if testCase.expectError {
				require.Error(t, err)
				require.Nil(t, body)
			} else {
				require.NoError(t, err)
				require.Equal(t, testCase.expected, body)
			}
		})
	}
}
