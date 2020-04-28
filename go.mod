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
	github.com/dchest/siphash v1.2.0 // indirect
	github.com/dustin/go-humanize v1.0.0
	github.com/getlantern/bbrconn v0.0.0-20180619163322-86cf8c16f3d0
	github.com/getlantern/borda v0.0.0-20200427033127-b36d009c6252
	github.com/getlantern/cmux v0.0.0-20200120072431-136083c8edb8
	github.com/getlantern/context v0.0.0-20190109183933-c447772a6520
	github.com/getlantern/ema v0.0.0-20190620044903-5943d28f40e4
	github.com/getlantern/enhttp v0.0.0-20190401024120-a974fa851e3c
	github.com/getlantern/errors v0.0.0-20190325191628-abdb3e3e36f7
	github.com/getlantern/geo v0.0.0-20200423004723-045cc9bbd328
	github.com/getlantern/golog v0.0.0-20190830074920-4ef2e798c2d7
	github.com/getlantern/gonat v0.0.0-20200420153910-d0d331e11ce4
	github.com/getlantern/hidden v0.0.0-20190325191715-f02dbb02be55
	github.com/getlantern/http-proxy v0.0.3-0.20200407205042-2382946d79e7
	github.com/getlantern/kcp-go v0.0.0-20171025115649-19559e0e938c // indirect
	github.com/getlantern/kcpwrapper v0.0.0-20171114192627-a35c895f6de7
	github.com/getlantern/keyman v0.0.0-20180207174507-f55e7280e93a
	github.com/getlantern/lampshade v0.0.0-20200303040944-fe53f13203e9
	github.com/getlantern/measured v0.0.0-20180919192309-c70b16bb4198
	github.com/getlantern/mockconn v0.0.0-20191023022503-481dbcceeb58
	github.com/getlantern/mtime v0.0.0-20200417132445-23682092d1f7
	github.com/getlantern/netx v0.0.0-20190110220209-9912de6f94fd
	github.com/getlantern/ops v0.0.0-20200403153110-8476b16edcd6
	github.com/getlantern/packetforward v0.0.0-20200421081927-11933f311913
	github.com/getlantern/pcapper v0.0.0-20181212174440-a8b1a3ff0cde
	github.com/getlantern/proxy v0.0.0-20200302183216-36afa00d0126
	github.com/getlantern/quic0 v0.0.0-20200121154153-8b18c2ba09f9
	github.com/getlantern/quicwrapper v0.0.0-20200129232925-8ef70253fcae
	github.com/getlantern/ring v0.0.0-20181206150603-dd46ce8faa01 // indirect
	github.com/getlantern/testredis v0.0.0-20190411184556-1cd088e934c0
	github.com/getlantern/tinywss v0.0.0-20200121221108-851921f95ad7
	github.com/getlantern/tlsdefaults v0.0.0-20171004213447-cf35cfd0b1b4
	github.com/getlantern/tlsmasq v0.3.0
	github.com/getlantern/tlsredis v0.0.0-20180308045249-5d4ed6dd3836
	github.com/getlantern/utls v0.0.0-20191119185840-3db8c755b682
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
	github.com/huandu/xstrings v1.2.0 // indirect
	github.com/juju/ratelimit v1.0.1
	github.com/mikioh/tcp v0.0.0-20180707144002-02a37043a4f7 // indirect
	github.com/mikioh/tcpinfo v0.0.0-20180831101334-131b59fef27f // indirect
	github.com/mikioh/tcpopt v0.0.0-20180707144150-7178f18b4ea8 // indirect
	github.com/mitchellh/panicwrap v1.0.0
	github.com/prometheus/client_golang v1.1.0
	github.com/refraction-networking/utls v0.0.0-20190415193640-32987941ebd3 // indirect
	github.com/stretchr/testify v1.5.1
	github.com/templexxx/cpufeat v0.0.0-20180724012125-cef66df7f161 // indirect
	github.com/templexxx/reedsolomon v0.0.0-20170927015403-7092926d7d05 // indirect
	github.com/templexxx/xor v0.0.0-20181023030647-4e92f724b73b // indirect
	github.com/tjfoc/gmsm v1.0.1 // indirect
	github.com/vharitonsky/iniflags v0.0.0-20180513140207-a33cd0b5f3de
	go.opencensus.io v0.17.0 // indirect
	golang.org/x/crypto v0.0.0-20200115085410-6d4e4cb37c7d // indirect
	golang.org/x/net v0.0.0-20200324143707-d3edc9973b7e
	google.golang.org/api v0.0.0-20180921000521-920bb1beccf7
	google.golang.org/genproto v0.0.0-20180918203901-c3f76f3b92d1 // indirect
	google.golang.org/grpc v1.23.1 // indirect
	gopkg.in/redis.v5 v5.2.9
)

replace github.com/lucas-clemente/quic-go => github.com/getlantern/quic-go v0.0.0-20200129225515-f0142adfc7dd

replace github.com/anacrolix/go-libutp => github.com/getlantern/go-libutp v1.0.3

replace github.com/getlantern/testredis => github.com/getlantern/testredis v0.0.0-20180921025736-7a5ea00c9914

// git.apache.org isn't working at the moment, use mirror (should probably switch back once we can)
replace git.apache.org/thrift.git => github.com/apache/thrift v0.0.0-20180902110319-2566ecd5d999
