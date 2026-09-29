package pushwoosh

import "time"

// ClientConfig carries both pushwoosh api tokens: the device api methods (registerDevice,
// unregisterDevice) accept only the device token, every other method accepts only the server token
type ClientConfig interface {
	BaseApiUrl() string
	ApplicationId() string
	DeviceApiKey() string
	ServerApiKey() string
	Timeout() time.Duration
}

type ClientConfigImpl struct {
	BaseApiUrlValue    string        `json:"base_api_url"   yaml:"base_api_url"`
	ApplicationIdValue string        `json:"application_id" yaml:"application_id"`
	DeviceApiKeyValue  string        `json:"device_api_key" yaml:"device_api_key"`
	ServerApiKeyValue  string        `json:"server_api_key" yaml:"server_api_key"`
	TimeoutValue       time.Duration `json:"timeout"        yaml:"timeout"`
}

func NewClientConfig(
	baseApiUrl string,
	applicationId string,
	deviceApiKey string,
	serverApiKey string,
	timeout time.Duration,
) ClientConfig {
	return &ClientConfigImpl{
		BaseApiUrlValue:    baseApiUrl,
		ApplicationIdValue: applicationId,
		DeviceApiKeyValue:  deviceApiKey,
		ServerApiKeyValue:  serverApiKey,
		TimeoutValue:       timeout,
	}
}

func (c *ClientConfigImpl) BaseApiUrl() string {
	return c.BaseApiUrlValue
}

func (c *ClientConfigImpl) ApplicationId() string {
	return c.ApplicationIdValue
}

func (c *ClientConfigImpl) DeviceApiKey() string {
	return c.DeviceApiKeyValue
}

func (c *ClientConfigImpl) ServerApiKey() string {
	return c.ServerApiKeyValue
}

func (c *ClientConfigImpl) Timeout() time.Duration {
	return c.TimeoutValue
}
