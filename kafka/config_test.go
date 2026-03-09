package kafka

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestValidateConfig(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		cfg     Config
		wantErr string
	}{
		{
			name:    "empty brokers",
			cfg:     &ConfigYaml{TopicValue: "test-topic"},
			wantErr: "brokers must not be empty",
		},
		{
			name:    "empty topic",
			cfg:     &ConfigYaml{BrokersValue: []string{"localhost:9092"}},
			wantErr: "topic must not be empty",
		},
		{
			name: "valid config",
			cfg: &ConfigYaml{
				BrokersValue: []string{"localhost:9092"},
				TopicValue:   "test-topic",
			},
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			err := validateConfig(testCase.cfg)
			if testCase.wantErr != "" {
				require.ErrorContains(t, err, testCase.wantErr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestValidateConsumerConfig(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		cfg     ConsumerConfig
		wantErr string
	}{
		{
			name: "empty group_id",
			cfg: &ConsumerConfigYaml{
				ConfigYaml: ConfigYaml{
					BrokersValue: []string{"localhost:9092"},
					TopicValue:   "test-topic",
				},
			},
			wantErr: "group_id must not be empty",
		},
		{
			name: "valid consumer config",
			cfg: &ConsumerConfigYaml{
				ConfigYaml: ConfigYaml{
					BrokersValue: []string{"localhost:9092"},
					TopicValue:   "test-topic",
				},
				GroupIDValue: "test-group",
			},
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			err := validateConsumerConfig(testCase.cfg)
			if testCase.wantErr != "" {
				require.ErrorContains(t, err, testCase.wantErr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestBuildSASLMechanism(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		mechanism string
		wantErr   bool
	}{
		{name: "PLAIN", mechanism: "PLAIN"},
		{name: "SCRAM-SHA-256", mechanism: "SCRAM-SHA-256"},
		{name: "SCRAM-SHA-512", mechanism: "SCRAM-SHA-512"},
		{name: "unsupported", mechanism: "OAUTHBEARER", wantErr: true},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			cfg := &SASLConfigYaml{
				MechanismValue: testCase.mechanism,
				UsernameValue:  "user",
				PasswordValue:  "pass",
			}

			mechanism, err := buildSASLMechanism(cfg)
			if testCase.wantErr {
				require.Error(t, err)
				require.ErrorIs(t, err, ErrSASLConfig)
				require.Nil(t, mechanism)
			} else {
				require.NoError(t, err)
				require.NotNil(t, mechanism)
			}
		})
	}
}

func TestConfigYaml_ConnectTimeout(t *testing.T) {
	t.Parallel()

	t.Run("zero returns default", func(t *testing.T) {
		t.Parallel()

		cfg := &ConfigYaml{}
		require.Equal(t, defaultConnectTimeout, cfg.ConnectTimeout())
	})

	t.Run("negative returns default", func(t *testing.T) {
		t.Parallel()

		cfg := &ConfigYaml{ConnectTimeoutValue: -1 * time.Second}
		require.Equal(t, defaultConnectTimeout, cfg.ConnectTimeout())
	})

	t.Run("positive returns value", func(t *testing.T) {
		t.Parallel()

		cfg := &ConfigYaml{ConnectTimeoutValue: 10 * time.Second}
		require.Equal(t, 10*time.Second, cfg.ConnectTimeout())
	})
}

func TestConfigYaml_NilSubconfigs(t *testing.T) {
	t.Parallel()

	cfg := &ConfigYaml{}

	require.Nil(t, cfg.RetryPolicy())
	require.Nil(t, cfg.SASL())
	require.Nil(t, cfg.TLS())
	require.Nil(t, cfg.CircuitBreaker())
}

func TestConsumerConfigYaml_MaxProcessingAttempts(t *testing.T) {
	t.Parallel()

	t.Run("zero by default", func(t *testing.T) {
		t.Parallel()

		cfg := &ConsumerConfigYaml{}
		require.Equal(t, 0, cfg.MaxProcessingAttempts())
	})

	t.Run("returns configured value", func(t *testing.T) {
		t.Parallel()

		cfg := &ConsumerConfigYaml{MaxProcessingAttemptsValue: 5}
		require.Equal(t, 5, cfg.MaxProcessingAttempts())
	})
}
