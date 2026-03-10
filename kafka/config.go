package kafka

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/pixality-inc/golang-core/circuit_breaker"
	"github.com/pixality-inc/golang-core/retry"

	"github.com/twmb/franz-go/pkg/kgo"
	"github.com/twmb/franz-go/pkg/sasl"
	"github.com/twmb/franz-go/pkg/sasl/plain"
	"github.com/twmb/franz-go/pkg/sasl/scram"
)

const defaultConnectTimeout = 5 * time.Second

var (
	ErrTLSConfig    = errors.New("kafka tls config error")
	ErrSASLConfig   = errors.New("kafka sasl config error")
	ErrNotConnected = errors.New("kafka client is not connected")
	ErrNoBrokers    = errors.New("kafka: brokers must not be empty")
	ErrNoTopic      = errors.New("kafka: topic must not be empty")
	ErrNoGroupID    = errors.New("kafka: group_id must not be empty for consumer")
)

type SASLConfig interface {
	Mechanism() string
	Username() string
	Password() string
}

type SASLConfigYaml struct {
	MechanismValue string `env:"MECHANISM" yaml:"mechanism"`
	UsernameValue  string `env:"USERNAME"  yaml:"username"`
	PasswordValue  string `env:"PASSWORD"  yaml:"password"`
}

func (c *SASLConfigYaml) Mechanism() string {
	return c.MechanismValue
}

func (c *SASLConfigYaml) Username() string {
	return c.UsernameValue
}

func (c *SASLConfigYaml) Password() string {
	return c.PasswordValue
}

type TLSConfig interface {
	Enabled() bool
	CAFile() string
	CertFile() string
	KeyFile() string
}

type TLSConfigYaml struct {
	EnabledValue  bool   `env:"ENABLED"   yaml:"enabled"`
	CAFileValue   string `env:"CA_FILE"   yaml:"ca_file"`
	CertFileValue string `env:"CERT_FILE" yaml:"cert_file"`
	KeyFileValue  string `env:"KEY_FILE"  yaml:"key_file"`
}

func (c *TLSConfigYaml) Enabled() bool {
	return c.EnabledValue
}

func (c *TLSConfigYaml) CAFile() string {
	return c.CAFileValue
}

func (c *TLSConfigYaml) CertFile() string {
	return c.CertFileValue
}

func (c *TLSConfigYaml) KeyFile() string {
	return c.KeyFileValue
}

type Config interface {
	Brokers() []string
	Topic() string
	RetryPolicy() retry.Policy
	SASL() SASLConfig
	TLS() TLSConfig
	CircuitBreaker() circuit_breaker.Config
	ConnectTimeout() time.Duration
}

type ConsumerConfig interface {
	Config
	GroupID() string
	AutoCommit() bool
	MaxProcessingAttempts() int // 0 = infinite (current behavior)
}

type ConfigYaml struct {
	BrokersValue        []string                    `env:"BROKERS"                 yaml:"brokers"`
	TopicValue          string                      `env:"TOPIC"                   yaml:"topic"`
	RetryPolicyValue    *retry.ConfigYaml           `env-prefix:"RETRY_"           yaml:"retry"`
	SASLValue           *SASLConfigYaml             `env-prefix:"SASL_"            yaml:"sasl"`
	TLSValue            *TLSConfigYaml              `env-prefix:"TLS_"             yaml:"tls"`
	CircuitBreakerValue *circuit_breaker.ConfigYaml `env-prefix:"CIRCUIT_BREAKER_" yaml:"circuit_breaker"`
	ConnectTimeoutValue time.Duration               `env:"CONNECT_TIMEOUT"         yaml:"connect_timeout"`
}

func (c *ConfigYaml) Brokers() []string {
	return c.BrokersValue
}

func (c *ConfigYaml) Topic() string {
	return c.TopicValue
}

func (c *ConfigYaml) RetryPolicy() retry.Policy {
	if c.RetryPolicyValue == nil {
		return nil
	}

	return c.RetryPolicyValue
}

func (c *ConfigYaml) SASL() SASLConfig {
	if c.SASLValue == nil {
		return nil
	}

	return c.SASLValue
}

func (c *ConfigYaml) TLS() TLSConfig {
	if c.TLSValue == nil {
		return nil
	}

	return c.TLSValue
}

func (c *ConfigYaml) CircuitBreaker() circuit_breaker.Config {
	if c.CircuitBreakerValue == nil {
		return nil
	}

	return c.CircuitBreakerValue
}

func (c *ConfigYaml) ConnectTimeout() time.Duration {
	if c.ConnectTimeoutValue <= 0 {
		return defaultConnectTimeout
	}

	return c.ConnectTimeoutValue
}

type ConsumerConfigYaml struct {
	ConfigYaml `yaml:",inline"`

	GroupIDValue               string `env:"GROUP_ID"                yaml:"group_id"`
	AutoCommitValue            bool   `env:"AUTO_COMMIT"             yaml:"auto_commit"`
	MaxProcessingAttemptsValue int    `env:"MAX_PROCESSING_ATTEMPTS" yaml:"max_processing_attempts"`
}

func (c *ConsumerConfigYaml) GroupID() string {
	return c.GroupIDValue
}

func (c *ConsumerConfigYaml) AutoCommit() bool {
	return c.AutoCommitValue
}

func (c *ConsumerConfigYaml) MaxProcessingAttempts() int {
	return c.MaxProcessingAttemptsValue
}

func buildKgoOpts(cfg Config) ([]kgo.Opt, error) {
	opts := []kgo.Opt{
		kgo.SeedBrokers(cfg.Brokers()...),
	}

	if cfg.SASL() != nil {
		mechanism, err := buildSASLMechanism(cfg.SASL())
		if err != nil {
			return nil, err
		}

		opts = append(opts, kgo.SASL(mechanism))
	}

	if cfg.TLS() != nil && cfg.TLS().Enabled() {
		tlsConfig, err := buildTLSConfig(cfg.TLS())
		if err != nil {
			return nil, err
		}

		opts = append(opts, kgo.DialTLSConfig(tlsConfig))
	}

	return opts, nil
}

func validateConfig(cfg Config) error {
	if len(cfg.Brokers()) == 0 {
		return ErrNoBrokers
	}

	if cfg.Topic() == "" {
		return ErrNoTopic
	}

	return nil
}

func validateConsumerConfig(cfg ConsumerConfig) error {
	if err := validateConfig(cfg); err != nil {
		return err
	}

	if cfg.GroupID() == "" {
		return ErrNoGroupID
	}

	return nil
}

func buildSASLMechanism(cfg SASLConfig) (sasl.Mechanism, error) {
	switch cfg.Mechanism() {
	case "PLAIN":
		return plain.Auth{
			User: cfg.Username(),
			Pass: cfg.Password(),
		}.AsMechanism(), nil
	case "SCRAM-SHA-256":
		return scram.Auth{
			User: cfg.Username(),
			Pass: cfg.Password(),
		}.AsSha256Mechanism(), nil
	case "SCRAM-SHA-512":
		return scram.Auth{
			User: cfg.Username(),
			Pass: cfg.Password(),
		}.AsSha512Mechanism(), nil
	default:
		return nil, fmt.Errorf("%w: unsupported SASL mechanism: %s", ErrSASLConfig, cfg.Mechanism())
	}
}

func buildTLSConfig(cfg TLSConfig) (*tls.Config, error) {
	tlsConfig := &tls.Config{
		MinVersion: tls.VersionTLS12,
	}

	if cfg.CAFile() != "" {
		caCert, err := os.ReadFile(cfg.CAFile())
		if err != nil {
			return nil, fmt.Errorf("%w: failed to read CA file: %w", ErrTLSConfig, err)
		}

		caCertPool := x509.NewCertPool()
		if !caCertPool.AppendCertsFromPEM(caCert) {
			return nil, fmt.Errorf("%w: failed to parse CA certificate", ErrTLSConfig)
		}

		tlsConfig.RootCAs = caCertPool
	}

	hasCert := cfg.CertFile() != ""
	hasKey := cfg.KeyFile() != ""

	if hasCert != hasKey {
		return nil, fmt.Errorf("%w: both cert_file and key_file must be provided together", ErrTLSConfig)
	}

	if hasCert && hasKey {
		cert, err := tls.LoadX509KeyPair(cfg.CertFile(), cfg.KeyFile())
		if err != nil {
			return nil, fmt.Errorf("%w: failed to load client certificate: %w", ErrTLSConfig, err)
		}

		tlsConfig.Certificates = []tls.Certificate{cert}
	}

	return tlsConfig, nil
}
