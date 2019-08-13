module github.com/getlantern/http-proxy-lantern

go 1.12

require (
	cloud.google.com/go v0.28.0
	contrib.go.opencensus.io/exporter/stackdriver v0.6.0 // indirect
	git.torproject.org/pluggable-transports/goptlib.git v0.0.0-20180321061416-7d56ec4f381e
	git.torproject.org/pluggable-transports/obfs4.git v0.0.0-20180421031126-89c21805c212
	github.com/agl/ed25519 v0.0.0-20170116200512-5312a6153412 // indirect
	github.com/anacrolix/envpprof v1.0.0 // indirect
	github.com/anacrolix/go-libutp v1.0.1
	github.com/anacrolix/missinggo v1.1.1 // indirect
	github.com/blang/semver v0.0.0-20180723201105-3c1074078d32
	github.com/bradfitz/iter v0.0.0-20190303215204-33e6a9893b0c // indirect
	github.com/cloudflare/sidh v0.0.0-20190228162259-d2f0f90e08aa // indirect
	github.com/dchest/siphash v1.2.0 // indirect
	github.com/dustin/go-humanize v1.0.0
	github.com/getlantern/bbrconn v0.0.0-20180619163322-86cf8c16f3d0
	github.com/getlantern/borda v0.0.0-20190809122504-668025f4c2b9
	github.com/getlantern/bytecounting v0.0.0-20190530140808-3b3f10d3b9ab // indirect
	github.com/getlantern/cmux v0.0.0-20190809092548-4b7adf243efe
	github.com/getlantern/ema v0.0.0-20190620044903-5943d28f40e4
	github.com/getlantern/enhttp v0.0.0-20190401024120-a974fa851e3c
	github.com/getlantern/errors v0.0.0-20190325191628-abdb3e3e36f7
	github.com/getlantern/fronted v0.0.0-20190606212108-e7744195eded // indirect
	github.com/getlantern/geolookup v0.0.0-20180719190536-68d621f75f46
	github.com/getlantern/golog v0.0.0-20190809085441-26e09e6dd330
	github.com/getlantern/gonat v0.0.0-20190809093358-98412e37c429
	github.com/getlantern/hidden v0.0.0-20190325191715-f02dbb02be55
	github.com/getlantern/http-proxy v0.0.3-0.20190809090746-b2f0d5c04754
	github.com/getlantern/kcp-go v0.0.0-20171025115649-19559e0e938c // indirect
	github.com/getlantern/kcpwrapper v0.0.0-20171114192627-a35c895f6de7
	github.com/getlantern/keyman v0.0.0-20180207174507-f55e7280e93a
	github.com/getlantern/lampshade v0.0.0-20190507122828-84b870a67bd6
	github.com/getlantern/measured v0.0.0-20180919192309-c70b16bb4198
	github.com/getlantern/mockconn v0.0.0-20190708122800-637bd46d8034
	github.com/getlantern/mtime v0.0.0-20170117193331-ba114e4a82b0
	github.com/getlantern/netx v0.0.0-20190110220209-9912de6f94fd
	github.com/getlantern/ops v0.0.0-20190325191751-d70cb0d6f85f
	github.com/getlantern/packetforward v0.0.0-20190809094443-386cbcc0d498
	github.com/getlantern/pcapper v0.0.0-20181212174440-a8b1a3ff0cde
	github.com/getlantern/preconn v0.0.0-20180328114929-0b5766010efe // indirect
	github.com/getlantern/proxy v0.0.0-20181004033118-a1730c79960f
	github.com/getlantern/quicwrapper v0.0.0-20190820201154-8079fdf487de
	github.com/getlantern/ring v0.0.0-20181206150603-dd46ce8faa01 // indirect
	github.com/getlantern/testredis v0.0.0-20190411184556-1cd088e934c0
	github.com/getlantern/tinywss v0.0.0-20190809093313-4439caa924e5
	github.com/getlantern/tlsdefaults v0.0.0-20171004213447-cf35cfd0b1b4
	github.com/getlantern/tlsdialer v0.0.0-20190606180931-1ac26445d532 // indirect
	github.com/getlantern/tlsredis v0.0.0-20180308045249-5d4ed6dd3836
	github.com/getlantern/waitforserver v1.0.1
	github.com/getlantern/withtimeout v0.0.0-20160829163843-511f017cd913
	github.com/golang/groupcache v0.0.0-20180513044358-24b0969c4cb7
	github.com/gonum/blas v0.0.0-20180125090452-e7c5890b24cf // indirect
	github.com/gonum/floats v0.0.0-20180125090339-7de1f4ea7ab5 // indirect
	github.com/gonum/integrate v0.0.0-20181209220457-a422b5c0fdf2 // indirect
	github.com/gonum/internal v0.0.0-20180125090855-fda53f8d2571 // indirect
	github.com/gonum/lapack v0.0.0-20180125091020-f0b8b25edece // indirect
	github.com/gonum/matrix v0.0.0-20180124231301-a41cc49d4c29 // indirect
	github.com/gonum/stat v0.0.0-20180125090729-ec9c8a1062f4
	github.com/googleapis/gax-go v2.0.0+incompatible // indirect
	github.com/hashicorp/golang-lru v0.5.1
	github.com/huandu/xstrings v1.2.0 // indirect
	github.com/juju/ratelimit v1.0.1
	github.com/lucas-clemente/quic-go v0.7.1-0.20190207125157-7dc4be2ce994 // indirect
	github.com/marten-seemann/qtls v0.0.0-20190207043627-591c71538704 // indirect
	github.com/mikioh/tcp v0.0.0-20180707144002-02a37043a4f7 // indirect
	github.com/mikioh/tcpinfo v0.0.0-20180831101334-131b59fef27f // indirect
	github.com/mikioh/tcpopt v0.0.0-20180707144150-7178f18b4ea8 // indirect
	github.com/prometheus/client_golang v0.9.2
	github.com/prometheus/client_model v0.0.0-20190129233127-fd36f4220a90 // indirect
	github.com/prometheus/common v0.2.0 // indirect
	github.com/prometheus/procfs v0.0.0-20190129233650-316cf8ccfec5 // indirect
	github.com/refraction-networking/utls v0.0.0-20190415193640-32987941ebd3 // indirect
	github.com/stretchr/testify v1.3.0
	github.com/templexxx/cpufeat v0.0.0-20180724012125-cef66df7f161 // indirect
	github.com/templexxx/reedsolomon v0.0.0-20170927015403-7092926d7d05 // indirect
	github.com/templexxx/xor v0.0.0-20170926022130-0af8e873c554 // indirect
	github.com/tjfoc/gmsm v0.0.0-20171124023159-98aa888b79d8 // indirect
	github.com/vharitonsky/iniflags v0.0.0-20180513140207-a33cd0b5f3de
	go.opencensus.io v0.17.0 // indirect
	golang.org/x/net v0.0.0-20190603091049-60506f45cf65
	google.golang.org/api v0.0.0-20180921000521-920bb1beccf7
	google.golang.org/genproto v0.0.0-20180918203901-c3f76f3b92d1 // indirect
	gopkg.in/redis.v5 v5.2.9
)

replace github.com/lucas-clemente/quic-go => github.com/getlantern/quic-go v0.7.1-0.20190818104938-28e3ca4262e1

replace github.com/marten-seemann/qtls => github.com/marten-seemann/qtls-deprecated v0.0.0-20190207043627-591c71538704

replace github.com/anacrolix/go-libutp => github.com/getlantern/go-libutp v1.0.3

replace github.com/getlantern/testredis => github.com/getlantern/testredis v0.0.0-20180921025736-7a5ea00c9914
