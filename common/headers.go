package common

const (
	AppHeader               = "X-Lantern-App"
	PlatformHeader          = "X-Lantern-Platform"
	LegacyVersionHeader     = "X-Lantern-Version"
	AppVersionHeader        = "X-Lantern-App-Version"
	LibraryVersionHeader    = "X-Lantern-Lib-Version"
	DeviceIdHeader          = "X-Lantern-Device-Id"
	SupportedDataCapsHeader = "X-Lantern-Supported-Data-Caps"
	TimeZoneHeader          = "X-Lantern-Time-Zone"
	TokenHeader             = "X-Lantern-Auth-Token"
	PingHeader              = "X-Lantern-Ping"
	PingURLHeader           = "X-Lantern-Ping-Url"
	PingTSHeader            = "X-Lantern-Ping-Ts"
	ProTokenHeader          = "X-Lantern-Pro-Token"
	CfgSvrAuthTokenHeader   = "X-Lantern-Config-Auth-Token"
	CfgSvrClientIPHeader    = "X-Lantern-Config-Client-IP"
	LocaleHeader            = "X-Lantern-Locale"
	XBQHeader               = "XBQ"
	XBQHeaderv2             = "XBQv2"
)

// This standardizes the keys we use for storing data in the request context
// and for reporting to teleport.
const (
	Platform         = "client_platform"
	Version          = "client_version"
	Locale           = "client_locale"
	AppVersion       = "client_app_version"
	LibraryVersion   = "client_lib_version"
	App              = "client_app"
	DeviceID         = "device_id"
	OriginHost       = "origin_host"
	OriginPort       = "origin_port"
	ProbingError     = "probing_error"
	ClientIP         = "client_ip"
	ThrottleSettings = "throttle_settings"
	TimeZone         = "time_zone"
	SupportDataCaps  = "supported_data_caps"
)
