package pushwoosh

type DeviceType int

const (
	DeviceTypeIOS      DeviceType = 1  // iOS
	DeviceTypeAndroid  DeviceType = 3  // Android
	DeviceTypeMacOSX   DeviceType = 7  // Mac OS X
	DeviceTypeWindows  DeviceType = 8  // Windows
	DeviceTypeAmazon   DeviceType = 9  // Amazon
	DeviceTypeSafari   DeviceType = 10 // Safari
	DeviceTypeChrome   DeviceType = 11 // Chrome
	DeviceTypeFirefox  DeviceType = 12 // Firefox
	DeviceTypeEmail    DeviceType = 14 // Email
	DeviceTypeHuawei   DeviceType = 17 // Huawei
	DeviceTypeSMS      DeviceType = 18 // SMS
	DeviceTypeWhatsApp DeviceType = 21 // WhatsApp
)
