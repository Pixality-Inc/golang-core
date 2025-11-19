package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/ilyakaznacheev/cleanenv"
	"github.com/pixality-inc/golang-core/errors"
)

var (
	Dir      = ""
	Filename = "config.yaml"
)

var (
	ErrConfigCwd  = errors.New("config.cwd", "getting current working directory")
	ErrConfigRead = errors.New("config.read", "reading config")
	ErrConfigLoad = errors.New("config.load", "loading config")
)

func NewConfig[T any](filename string) (*T, error) {
	cfg := new(T)

	if filename == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return nil, errors.Join(ErrConfigCwd, err)
		}

		filename = filepath.Join(cwd, Dir, Filename)
	}

	if err := cleanenv.ReadConfig(filename, cfg); err != nil {
		return nil, fmt.Errorf("%w: %s: %w", ErrConfigRead, filename, err)
	}

	return cfg, nil
}

func LoadConfig[T any](filename string) *T {
	cfg, err := NewConfig[T](filename)
	if err != nil {
		panic(errors.Join(ErrConfigLoad, err))
	}

	return cfg
}
