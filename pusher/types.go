package pusher

type TokenType string

const (
	TokenTypeIOS     TokenType = "ios"
	TokenTypeAndroid TokenType = "android"
	TokenTypeHuawei  TokenType = "huawei"
	TokenTypeChrome  TokenType = "chrome"
	TokenTypeSafari  TokenType = "safari"
	TokenTypeFirefox TokenType = "firefox"
)

type UserId string

type DeviceId string
