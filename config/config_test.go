package config

import (
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
