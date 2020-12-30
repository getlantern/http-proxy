module github.com/getlantern/http-proxy-lantern/v2

go 1.12

require (
	cloud.google.com/go v0.65.0
	git.torproject.org/pluggable-transports/goptlib.git v0.0.0-20180321061416-7d56ec4f381e
	git.torproject.org/pluggable-transports/obfs4.git v0.0.0-20180421031126-89c21805c212
	github.com/agl/ed25519 v0.0.0-20170116200512-5312a6153412 // indirect
	github.com/anacrolix/envpprof v1.0.0 // indirect
	github.com/anacrolix/go-libutp v1.0.1
	github.com/anacrolix/missinggo v1.1.1 // indirect
	github.com/blang/semver v0.0.0-20180723201105-3c1074078d32
	github.com/bradfitz/iter v0.0.0-20190303215204-33e6a9893b0c // indirect
	github.com/dchest/siphash v1.2.0 // indirect
	github.com/dustin/go-humanize v1.0.0
	github.com/getlantern/bbrconn v0.0.0-20180619163322-86cf8c16f3d0
	github.com/getlantern/borda v0.0.0-20200427033127-b36d009c6252
	github.com/getlantern/cmux v0.0.0-20200905031936-c55b16ee8462 // indirect
	github.com/getlantern/cmux/v2 v2.0.0-20200905031936-c55b16ee8462
	github.com/getlantern/cmuxprivate v0.0.0-20200905032931-afb63438e40b
	github.com/getlantern/context v0.0.0-20190109183933-c447772a6520
	github.com/getlantern/ema v0.0.0-20190620044903-5943d28f40e4
	github.com/getlantern/enhttp v0.0.0-20190401024120-a974fa851e3c
	github.com/getlantern/errors v1.0.1
	github.com/getlantern/geo v0.0.0-20200716105557-5b87700e3d54
	github.com/getlantern/golog v0.0.0-20201105130739-9586b8bde3a9
	github.com/getlantern/gonat v0.0.0-20201001145726-634575ba87fb
	github.com/getlantern/hidden v0.0.0-20190325191715-f02dbb02be55
	github.com/getlantern/http-proxy v0.0.3-0.20200407205042-2382946d79e7
	github.com/getlantern/kcpwrapper v0.0.0-20201001150218-1427e1d39c25
	github.com/getlantern/keyman v0.0.0-20200820153608-cfd0ee278507
	github.com/getlantern/lampshade v0.0.0-20200303040944-fe53f13203e9
	github.com/getlantern/measured v0.0.0-20180919192309-c70b16bb4198
	github.com/getlantern/mockconn v0.0.0-20191023022503-481dbcceeb58
	github.com/getlantern/mtime v0.0.0-20200417132445-23682092d1f7
	github.com/getlantern/multipath v0.0.0-20201027015000-69ed0bd15259
	github.com/getlantern/netx v0.0.0-20190110220209-9912de6f94fd
	github.com/getlantern/ops v0.0.0-20200403153110-8476b16edcd6
	github.com/getlantern/packetforward v0.0.0-20201001150407-c68a447b0360
	github.com/getlantern/pcapper v0.0.0-20181212174440-a8b1a3ff0cde
	github.com/getlantern/proxy v0.0.0-20201001032732-eefd72879266
	github.com/getlantern/psmux v1.5.15-0.20200903210100-947ca5d91683
	github.com/getlantern/quicwrapper v0.0.0-20201013170341-d27d67101f2d
	github.com/getlantern/ring v0.0.0-20181206150603-dd46ce8faa01 // indirect
	github.com/getlantern/testredis v0.0.0-20190411184556-1cd088e934c0
	github.com/getlantern/tinywss v0.0.0-20200121221108-851921f95ad7
	github.com/getlantern/tlsdefaults v0.0.0-20171004213447-cf35cfd0b1b4
	github.com/getlantern/tlsmasq v0.3.0
	github.com/getlantern/tlsredis v0.0.0-20180308045249-5d4ed6dd3836
	github.com/getlantern/utls v0.0.0-20191119185840-3db8c755b682
	github.com/getlantern/waitforserver v1.0.1
	github.com/getlantern/withtimeout v0.0.0-20160829163843-511f017cd913
	github.com/golang/groupcache v0.0.0-20200121045136-8c9f03a8e57e
	github.com/gonum/blas v0.0.0-20180125090452-e7c5890b24cf // indirect
	github.com/gonum/floats v0.0.0-20180125090339-7de1f4ea7ab5 // indirect
	github.com/gonum/integrate v0.0.0-20181209220457-a422b5c0fdf2 // indirect
	github.com/gonum/internal v0.0.0-20180125090855-fda53f8d2571 // indirect
	github.com/gonum/lapack v0.0.0-20180125091020-f0b8b25edece // indirect
	github.com/gonum/matrix v0.0.0-20180124231301-a41cc49d4c29 // indirect
	github.com/gonum/stat v0.0.0-20180125090729-ec9c8a1062f4
	github.com/google/gopacket v1.1.17
	github.com/huandu/xstrings v1.2.0 // indirect
	github.com/juju/ratelimit v1.0.1
	github.com/lucas-clemente/quic-go v0.7.1-0.20190207125157-7dc4be2ce994
	github.com/mikioh/tcp v0.0.0-20180707144002-02a37043a4f7 // indirect
	github.com/mikioh/tcpinfo v0.0.0-20180831101334-131b59fef27f // indirect
	github.com/mikioh/tcpopt v0.0.0-20180707144150-7178f18b4ea8 // indirect
	github.com/mitchellh/panicwrap v1.0.0
	github.com/prometheus/client_golang v1.1.0
	github.com/refraction-networking/utls v0.0.0-20190415193640-32987941ebd3 // indirect
	github.com/spaolacci/murmur3 v1.1.0
	github.com/stretchr/testify v1.6.1
	github.com/vharitonsky/iniflags v0.0.0-20180513140207-a33cd0b5f3de
	github.com/xtaci/smux v1.5.15-0.20200704123958-f7188026ba01
	golang.org/x/net v0.0.0-20200822124328-c89045814202
	google.golang.org/api v0.32.0
	gopkg.in/redis.v5 v5.2.9
)

replace github.com/lucas-clemente/quic-go => github.com/getlantern/quic-go v0.0.0-20201013165432-d264463d99fd

replace github.com/anacrolix/go-libutp => github.com/getlantern/go-libutp v1.0.3

replace github.com/getlantern/testredis => github.com/getlantern/testredis v0.0.0-20180921025736-7a5ea00c9914

// git.apache.org isn't working at the moment, use mirror (should probably switch back once we can)
replace git.apache.org/thrift.git => github.com/apache/thrift v0.0.0-20180902110319-2566ecd5d999

// Waiting on https://github.com/mitchellh/panicwrap/pull/27 to be merged upstream
replace github.com/mitchellh/panicwrap v1.0.0 => github.com/getlantern/panicwrap v0.0.0-20200707191944-9ba45baf8e51
