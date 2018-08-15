package common

const (
	VersionHeader         = "X-Lantern-Version"
	DeviceIdHeader        = "X-Lantern-Device-Id"
	TokenHeader           = "X-Lantern-Auth-Token"
	PingHeader            = "X-Lantern-Ping"
	PingURLHeader         = "X-Lantern-Ping-Url"
	PingTSHeader          = "X-Lantern-Ping-Ts"
	ProTokenHeader        = "X-Lantern-Pro-Token"
	CfgSvrAuthTokenHeader = "X-Lantern-Config-Auth-Token"

	// TrueClientIP is set to "True-Client-IP" to overwrite any spoofed
	// "True-Client-IP" arriving from clients. Most CDNs add a True-Client-IP
	// header before forwarding traffic.
	TrueClientIP                        = "True-Client-IP"
	BBRRequested                        = "X-BBR"
	BBRAvailableBandwidthEstimateHeader = "X-BBR-ABE"
	XBQHeader                           = "XBQ"
)
