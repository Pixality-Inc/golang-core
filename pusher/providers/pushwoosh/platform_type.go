package pushwoosh

type PlatformType string

const (
	PlatformTypeIOS     PlatformType = "IOS"
	PlatformTypeAndroid PlatformType = "ANDROID"
	PlatformTypeHuawei  PlatformType = "HUAWEI_ANDROID"
	PlatformTypeChrome  PlatformType = "CHROME"
	PlatformTypeSafari  PlatformType = "SAFARI"
	PlatformTypeFirefox PlatformType = "FIREFOX"
	PlatformTypeIE      PlatformType = "IE"
	PlatformTypeWeb     PlatformType = "WEB"
)

var AllPlatformTypes = []PlatformType{
	PlatformTypeIOS,
	PlatformTypeAndroid,
	PlatformTypeHuawei,
	PlatformTypeChrome,
	PlatformTypeSafari,
	PlatformTypeFirefox,
	PlatformTypeIE,
	PlatformTypeWeb,
}

type ContentPlatformType string

const (
	ContentPlatformTypeIOS     ContentPlatformType = "ios"
	ContentPlatformTypeAndroid ContentPlatformType = "android"
	ContentPlatformTypeHuawei  ContentPlatformType = "huawei_android"
	ContentPlatformTypeChrome  ContentPlatformType = "chrome"
	ContentPlatformTypeSafari  ContentPlatformType = "safari"
	ContentPlatformTypeFirefox ContentPlatformType = "firefox"
	ContentPlatformTypeIE      ContentPlatformType = "ie"
)
