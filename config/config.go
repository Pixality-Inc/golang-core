package config

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/ilyakaznacheev/cleanenv"
	"github.com/pixality-inc/golang-core/errors"
)

var (
	EnvName               = "CONFIG_FILE"
	DefaultConfigFilename = "config.yaml"
)

var (
	ErrConfigRead = errors.New("config.read", "reading config")
	ErrConfigLoad = errors.New("config.load", "loading config")
)

func NewConfig[T any](envPrefix string) (*T, error) {
	cfg := new(T)

	configFileName := os.Getenv(envPrefix + EnvName)

	if len(configFileName) == 0 {
		var (
			_, b, _, _ = runtime.Caller(0)
			basepath   = filepath.Join(filepath.Dir(b), "../..")
		)

		configFileName = filepath.Join(basepath, DefaultConfigFilename)
	}

	if err := cleanenv.ReadConfig(configFileName, cfg); err != nil {
		return nil, fmt.Errorf("%w: %s: %w", ErrConfigRead, configFileName, err)
	}

	return cfg, nil
}

func LoadConfig[T any](envPrefix string) *T {
	cfg, err := NewConfig[T](envPrefix)
	if err != nil {
		panic(errors.Join(ErrConfigLoad, err))
	}

	return cfg
}
