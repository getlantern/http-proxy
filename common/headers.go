package common

const (
	AppHeader               = "X-Lantern-App"
	KernelArchHeader        = "X-Lantern-KernelArch"
	PlatformHeader          = "X-Lantern-Platform"
	PlatformVersionHeader   = "X-Lantern-PlatVer"
	LibraryVersionHeader    = "X-Lantern-Version"
	AppVersionHeader        = "X-Lantern-App-Version"
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
	Platform        = "client_platform"
	PlatformVersion = "client_platform_version"
	KernelArch      = "client_kernel_arch"

	// Note this is the flashlight version.
	LibraryVersion = "client_version"
	Locale         = "client_locale"

	// This is the version of the app that's using the library.
	AppVersion        = "client_app_version"
	App               = "client_app"
	DeviceID          = "device_id"
	OriginHost        = "origin_host"
	OriginPort        = "origin_port"
	ProbingError      = "probing_error"
	ClientIP          = "client_ip"
	ThrottleSettings  = "throttle_settings"
	TimeZone          = "time_zone"
	SupportedDataCaps = "supported_data_caps"
	UnboundedTeamId   = "unbounded_team_id"
)
