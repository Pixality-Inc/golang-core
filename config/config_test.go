package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

type testConfigExample struct {
	Hello string  `yaml:"hello"`
	Foo   int     `yaml:"foo"`
	Bar   float64 `yaml:"bar"`
	Baz   bool    `yaml:"baz"`
}

type testConfig struct {
	Example testConfigExample `yaml:"example"`
}

type testEnvConfig struct {
	Host string `env:"TEST_CONFIG_HOST" env-default:"localhost"`
	Port int    `env:"TEST_CONFIG_PORT" env-default:"5432"`
}

type testEnvOverrideConfig struct {
	Hello string `env:"TEST_CONFIG_HELLO" yaml:"hello"`
}

func TestConfig(t *testing.T) {
	t.Parallel()

	cfg, err := NewConfig[testConfig]("./config.test.yaml")
	if err != nil {
		t.Fatal(err)
	}

	require.NoError(t, err)
	require.Equal(t, "world", cfg.Example.Hello)
	require.Equal(t, 420, cfg.Example.Foo)
	require.InDelta(t, 3.14, cfg.Example.Bar, 0.001)
	require.True(t, cfg.Example.Baz)
}

func TestConfigLoad(t *testing.T) {
	t.Parallel()

	cfg := LoadConfig[testConfig]("./config.test.yaml")

	require.Equal(t, "world", cfg.Example.Hello)
	require.Equal(t, 420, cfg.Example.Foo)
	require.InDelta(t, 3.14, cfg.Example.Bar, 0.001)
	require.True(t, cfg.Example.Baz)
}

func TestConfigNoFile(t *testing.T) {
	t.Parallel()

	_, err := NewConfig[testConfig]("asd")
	require.ErrorIs(t, err, ErrConfigRead)
}

func TestConfigLoadNoFile(t *testing.T) {
	t.Parallel()

	require.Panics(t, func() {
		_ = LoadConfig[testConfig]("asd")
	})
}

func TestConfigMalformedYaml(t *testing.T) {
	t.Parallel()

	filename := writeTempConfig(t, "example: [unclosed")

	_, err := NewConfig[testConfig](filename)
	require.ErrorIs(t, err, ErrConfigRead)
}

func TestConfigTypeMismatch(t *testing.T) {
	t.Parallel()

	filename := writeTempConfig(t, "example:\n  foo: not-an-int\n")

	_, err := NewConfig[testConfig](filename)
	require.ErrorIs(t, err, ErrConfigRead)
}

func TestConfigEmptyFile(t *testing.T) {
	t.Parallel()

	filename := writeTempConfig(t, "")

	_, err := NewConfig[testConfig](filename)
	require.ErrorIs(t, err, ErrConfigRead)
}

func TestConfigEmptyFilenameUsesCwd(t *testing.T) { //nolint:paralleltest // t.Chdir is incompatible with t.Parallel
	dir := t.TempDir()
	content := "example:\n  hello: world\n"
	require.NoError(t, os.WriteFile(filepath.Join(dir, Filename), []byte(content), 0o600))
	t.Chdir(dir)

	cfg, err := NewConfig[testConfig]("")

	require.NoError(t, err)
	require.Equal(t, "world", cfg.Example.Hello)
}

func TestConfigFromEnv(t *testing.T) {
	t.Setenv("TEST_CONFIG_HOST", "example.com")
	t.Setenv("TEST_CONFIG_PORT", "9000")

	cfg, err := NewConfigFromEnv[testEnvConfig]()

	require.NoError(t, err)
	require.Equal(t, "example.com", cfg.Host)
	require.Equal(t, 9000, cfg.Port)
}

func TestConfigFromEnvDefaults(t *testing.T) {
	t.Setenv("TEST_CONFIG_HOST", "")
	t.Setenv("TEST_CONFIG_PORT", "")

	err := os.Unsetenv("TEST_CONFIG_HOST")
	require.NoError(t, err)

	err = os.Unsetenv("TEST_CONFIG_PORT")
	require.NoError(t, err)

	cfg, err := NewConfigFromEnv[testEnvConfig]()

	require.NoError(t, err)
	require.Equal(t, "localhost", cfg.Host)
	require.Equal(t, 5432, cfg.Port)
}

func TestConfigFromEnvInvalidValue(t *testing.T) {
	t.Setenv("TEST_CONFIG_PORT", "not-a-number")

	_, err := NewConfigFromEnv[testEnvConfig]()
	require.ErrorIs(t, err, ErrConfigRead)
}

func TestConfigEnvOverridesYaml(t *testing.T) {
	t.Setenv("TEST_CONFIG_HELLO", "from-env")

	filename := writeTempConfig(t, "hello: from-yaml\n")

	cfg, err := NewConfig[testEnvOverrideConfig](filename)

	require.NoError(t, err)
	require.Equal(t, "from-env", cfg.Hello)
}

func writeTempConfig(t *testing.T, content string) string {
	t.Helper()

	filename := filepath.Join(t.TempDir(), "config.yaml")
	require.NoError(t, os.WriteFile(filename, []byte(content), 0o600))

	return filename
}
