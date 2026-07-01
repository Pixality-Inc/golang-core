package pushwoosh

type ClientConfig interface {
	BaseApiUrl() string
	ApplicationId() string
	ApiKey() string
}

type ClientConfigImpl struct {
	BaseApiUrlValue    string `json:"base_api_url"   yaml:"base_api_url"`
	ApplicationIdValue string `json:"application_id" yaml:"application_id"`
	ApiKeyValue        string `json:"api_key"        yaml:"api_key"`
}

func NewClientConfig(baseApiUrl string, applicationId string, apiKey string) ClientConfig {
	return &ClientConfigImpl{
		BaseApiUrlValue:    baseApiUrl,
		ApplicationIdValue: applicationId,
		ApiKeyValue:        apiKey,
	}
}

func (c *ClientConfigImpl) BaseApiUrl() string {
	return c.BaseApiUrlValue
}

func (c *ClientConfigImpl) ApplicationId() string {
	return c.ApplicationIdValue
}

func (c *ClientConfigImpl) ApiKey() string {
	return c.ApiKeyValue
}
