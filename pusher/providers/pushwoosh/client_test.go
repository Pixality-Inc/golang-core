package pushwoosh

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestClient(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	client, err := NewClient(NewClientConfig(
		"https://api.pushwoosh.com",
		"0520F-D98C8",
		"JyyOMvsBDZgyGSViebCfYuKK8VMrgfbjJBXVSUr0hGVupjHy3DL9VTQt3K516nLwjY5y49g0anGuQ9RgvRgx",
	))
	require.NoError(t, err)

	err = client.RegisterDevice(
		ctx,
		DeviceTypeIOS,
		"420",
		"6969",
		"my-test-token",
	)
	require.NoError(t, err)

	err = client.UnregisterDevice(ctx, "6969")
	require.NoError(t, err)
}
