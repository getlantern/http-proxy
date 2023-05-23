module github.com/getlantern/http-proxy-lantern/v2

go 1.19

require (
	cloud.google.com/go/errorreporting v0.3.0
	git.torproject.org/pluggable-transports/goptlib.git v1.2.0
	github.com/Jigsaw-Code/outline-ss-server v1.4.0
	github.com/OperatorFoundation/Replicant-go/Replicant/v3 v3.0.18
	github.com/OperatorFoundation/Starbridge-go/Starbridge/v3 v3.0.15
	github.com/blang/semver v3.5.1+incompatible
	github.com/dustin/go-humanize v1.0.0
	github.com/getlantern/bbrconn v0.0.0-20180619163322-86cf8c16f3d0
	github.com/getlantern/broflake v0.0.0-20230523155939-6087765e3b2d
	github.com/getlantern/cmux/v2 v2.0.0-20230228131144-addc208d233b
	github.com/getlantern/cmuxprivate v0.0.0-20200905032931-afb63438e40b
	github.com/getlantern/ema v0.0.0-20190620044903-5943d28f40e4
	github.com/getlantern/enhttp v0.0.0-20190401024120-a974fa851e3c
	github.com/getlantern/errors v1.0.3
	github.com/getlantern/fdcount v0.0.0-20210503151800-5decd65b3731
	github.com/getlantern/geo v0.0.0-20221101125300-c661769d5822
	github.com/getlantern/golog v0.0.0-20230503153817-8e72de7e0a65
	github.com/getlantern/gonat v0.0.0-20201001145726-634575ba87fb
	github.com/getlantern/grtrack v0.0.0-20160824195228-cbf67d3fa0fd
	github.com/getlantern/http-proxy v0.0.3-0.20230405160101-eb4bf4e4a733
	github.com/getlantern/kcpwrapper v0.0.0-20220503142841-b0e764933966
	github.com/getlantern/keyman v0.0.0-20210218183930-5e48f8ced961
	github.com/getlantern/lampshade v0.0.0-20200303040944-fe53f13203e9
	github.com/getlantern/measured v0.0.0-20210507000559-ec5307b2b8be
	github.com/getlantern/mockconn v0.0.0-20200818071412-cb30d065a848
	github.com/getlantern/mtime v0.0.0-20200417132445-23682092d1f7
	github.com/getlantern/multipath v0.0.0-20220920195041-55195f38df73
	github.com/getlantern/netx v0.0.0-20211206143627-7ccfeb739cbd
	github.com/getlantern/packetforward v0.0.0-20201001150407-c68a447b0360
	github.com/getlantern/proxy/v2 v2.0.1-0.20220303164029-b34b76e0e581
	github.com/getlantern/psmux v1.5.15-0.20200903210100-947ca5d91683
	github.com/getlantern/quicwrapper v0.0.0-20230523101504-1ec066b7f869
	github.com/getlantern/ratelimit v0.0.0-20220926192648-933ab81a6fc7
	github.com/getlantern/tinywss v0.0.0-20200121221108-851921f95ad7
	github.com/getlantern/tlsdefaults v0.0.0-20171004213447-cf35cfd0b1b4
	github.com/getlantern/tlsmasq v0.4.7-0.20230302000139-6e479a593298
	github.com/getlantern/tlsutil v0.5.1
	github.com/getlantern/waitforserver v1.0.1
	github.com/getlantern/withtimeout v0.0.0-20160829163843-511f017cd913
	github.com/go-redis/redis/v8 v8.11.5
	github.com/golang/groupcache v0.0.0-20210331224755-41bb18bfe9da
	github.com/gonum/stat v0.0.0-20180125090729-ec9c8a1062f4
	github.com/mitchellh/panicwrap v1.0.0
	github.com/op/go-logging v0.0.0-20160315200505-970db520ece7
	github.com/prometheus/client_golang v1.15.0
	github.com/quic-go/quic-go v0.34.0
	github.com/refraction-networking/utls v0.0.0-20200729012536-186025ac7b77
	github.com/siddontang/go v0.0.0-20180604090527-bdc77568d726
	github.com/spaolacci/murmur3 v1.1.0
	github.com/stretchr/testify v1.8.3
	github.com/vharitonsky/iniflags v0.0.0-20180513140207-a33cd0b5f3de
	github.com/xtaci/smux v1.5.15-0.20200704123958-f7188026ba01
	gitlab.com/yawning/obfs4.git v0.0.0-20220204003609-77af0cba934d
	go.opentelemetry.io/otel v1.16.0
	go.opentelemetry.io/otel/exporters/otlp/otlptrace v1.16.0
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp v1.16.0
	go.opentelemetry.io/otel/sdk v1.16.0
	go.opentelemetry.io/otel/trace v1.16.0
	golang.org/x/net v0.10.0
	google.golang.org/api v0.110.0
)

require (
	cloud.google.com/go/compute v1.18.0 // indirect
	cloud.google.com/go/compute/metadata v0.2.3 // indirect
	filippo.io/edwards25519 v1.0.0-rc.1.0.20210721174708-390f27c3be20 // indirect
	github.com/OperatorFoundation/ghostwriter-go v1.0.6 // indirect
	github.com/OperatorFoundation/go-bloom v1.0.1 // indirect
	github.com/OperatorFoundation/go-shadowsocks2 v1.1.15 // indirect
	github.com/Yawning/chacha20 v0.0.0-20170904085104-e3b1f968fc63 // indirect
	github.com/aead/ecdh v0.2.0 // indirect
	github.com/andybalholm/brotli v1.0.4 // indirect
	github.com/benbjohnson/clock v1.3.0 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/cenkalti/backoff/v4 v4.2.1 // indirect
	github.com/cespare/xxhash/v2 v2.2.0 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/dchest/siphash v1.2.1 // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
	github.com/dsnet/compress v0.0.2-0.20210315054119-f66993602bf5 // indirect
	github.com/dvyukov/go-fuzz v0.0.0-20210429054444-fca39067bc72 // indirect
	github.com/felixge/httpsnoop v1.0.3 // indirect
	github.com/fsnotify/fsnotify v1.5.4 // indirect
	github.com/getlantern/bufconn v0.0.0-20190625204133-a08544339f8d // indirect
	github.com/getlantern/byteexec v0.0.0-20170405023437-4cfb26ec74f4 // indirect
	github.com/getlantern/cmux v0.0.0-20230301223233-dac79088a4c0 // indirect
	github.com/getlantern/context v0.0.0-20220418194847-3d5e7a086201 // indirect
	github.com/getlantern/elevate v0.0.0-20200430163644-2881a121236d // indirect
	github.com/getlantern/eventual v0.0.0-20180125201821-84b02499361b // indirect
	github.com/getlantern/filepersist v0.0.0-20160317154340-c5f0cd24e799 // indirect
	github.com/getlantern/framed v0.0.0-20190601192238-ceb6431eeede // indirect
	github.com/getlantern/go-cache v0.0.0-20141028142048-88b53914f467 // indirect
	github.com/getlantern/hex v0.0.0-20190417191902-c6586a6fe0b7 // indirect
	github.com/getlantern/hidden v0.0.0-20201229170000-e66e7f878730 // indirect
	github.com/getlantern/idletiming v0.0.0-20200228204104-10036786eac5 // indirect
	github.com/getlantern/iptool v0.0.0-20230112135223-c00e863b2696 // indirect
	github.com/getlantern/kcp-go/v5 v5.0.0-20220503142114-f0c1cd6e1b54 // indirect
	github.com/getlantern/keepcurrent v0.0.0-20221014183517-fcee77376b89 // indirect
	github.com/getlantern/mitm v0.0.0-20180205214248-4ce456bae650 // indirect
	github.com/getlantern/ops v0.0.0-20230424193308-26325dfed3cf // indirect
	github.com/getlantern/preconn v1.0.0 // indirect
	github.com/getlantern/reconn v0.0.0-20161128113912-7053d017511c // indirect
	github.com/getlantern/telemetry v0.0.0-20230523155019-be7c1d8cd8cb // indirect
	github.com/getlantern/uuid v1.2.0 // indirect
	github.com/go-logr/logr v1.2.4 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/go-stack/stack v1.8.1 // indirect
	github.com/go-task/slim-sprig v0.0.0-20210107165309-348f09dbbbc0 // indirect
	github.com/golang/mock v1.6.0 // indirect
	github.com/golang/protobuf v1.5.3 // indirect
	github.com/golang/snappy v0.0.4 // indirect
	github.com/gonum/blas v0.0.0-20180125090452-e7c5890b24cf // indirect
	github.com/gonum/floats v0.0.0-20180125090339-7de1f4ea7ab5 // indirect
	github.com/gonum/integrate v0.0.0-20181209220457-a422b5c0fdf2 // indirect
	github.com/gonum/internal v0.0.0-20180125090855-fda53f8d2571 // indirect
	github.com/gonum/lapack v0.0.0-20180125091020-f0b8b25edece // indirect
	github.com/gonum/matrix v0.0.0-20180124231301-a41cc49d4c29 // indirect
	github.com/google/go-cmp v0.5.9 // indirect
	github.com/google/pprof v0.0.0-20210720184732-4bb14d4b1be1 // indirect
	github.com/google/uuid v1.3.0 // indirect
	github.com/googleapis/enterprise-certificate-proxy v0.2.3 // indirect
	github.com/googleapis/gax-go/v2 v2.7.0 // indirect
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.10.2 // indirect
	github.com/hashicorp/golang-lru v0.5.4 // indirect
	github.com/klauspost/compress v1.15.12 // indirect
	github.com/klauspost/cpuid v1.3.1 // indirect
	github.com/klauspost/pgzip v1.2.5 // indirect
	github.com/klauspost/reedsolomon v1.9.9 // indirect
	github.com/libp2p/go-buffer-pool v0.0.2 // indirect
	github.com/matttproud/golang_protobuf_extensions v1.0.4 // indirect
	github.com/mdlayher/netlink v1.1.0 // indirect
	github.com/mholt/archiver/v3 v3.5.1 // indirect
	github.com/mikioh/tcp v0.0.0-20180707144002-02a37043a4f7 // indirect
	github.com/mikioh/tcpinfo v0.0.0-20180831101334-131b59fef27f // indirect
	github.com/mikioh/tcpopt v0.0.0-20180707144150-7178f18b4ea8 // indirect
	github.com/mmcloughlin/avo v0.0.0-20200803215136-443f81d77104 // indirect
	github.com/montanaflynn/stats v0.6.6 // indirect
	github.com/nwaples/rardecode v1.1.2 // indirect
	github.com/onsi/ginkgo/v2 v2.2.0 // indirect
	github.com/oschwald/geoip2-golang v1.8.0 // indirect
	github.com/oschwald/maxminddb-golang v1.10.0 // indirect
	github.com/oxtoacart/bpool v0.0.0-20190530202638-03653db5a59c // indirect
	github.com/pierrec/lz4/v4 v4.1.12 // indirect
	github.com/pion/datachannel v1.5.5 // indirect
	github.com/pion/dtls/v2 v2.2.7 // indirect
	github.com/pion/ice/v2 v2.3.5 // indirect
	github.com/pion/interceptor v0.1.17 // indirect
	github.com/pion/logging v0.2.2 // indirect
	github.com/pion/mdns v0.0.7 // indirect
	github.com/pion/randutil v0.1.0 // indirect
	github.com/pion/rtcp v1.2.10 // indirect
	github.com/pion/rtp v1.7.13 // indirect
	github.com/pion/sctp v1.8.7 // indirect
	github.com/pion/sdp/v3 v3.0.6 // indirect
	github.com/pion/srtp/v2 v2.0.15 // indirect
	github.com/pion/stun v0.6.0 // indirect
	github.com/pion/transport/v2 v2.2.1 // indirect
	github.com/pion/turn/v2 v2.1.0 // indirect
	github.com/pion/webrtc/v3 v3.2.6 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/prometheus/client_model v0.3.0 // indirect
	github.com/prometheus/common v0.42.0 // indirect
	github.com/prometheus/procfs v0.9.0 // indirect
	github.com/quic-go/qtls-go1-19 v0.3.2 // indirect
	github.com/quic-go/qtls-go1-20 v0.2.2 // indirect
	github.com/shadowsocks/go-shadowsocks2 v0.1.4-0.20201002022019-75d43273f5a5 // indirect
	github.com/songgao/water v0.0.0-20200317203138-2b4b6d7c09d8 // indirect
	github.com/templexxx/cpu v0.0.8 // indirect
	github.com/templexxx/xorsimd v0.4.1 // indirect
	github.com/ti-mo/conntrack v0.3.0 // indirect
	github.com/ti-mo/netfilter v0.3.1 // indirect
	github.com/tjfoc/gmsm v1.3.2 // indirect
	github.com/ulikunitz/xz v0.5.10 // indirect
	github.com/vishvananda/netns v0.0.0-20210104183010-2eb08e3e575f // indirect
	github.com/xi2/xz v0.0.0-20171230120015-48954b6210f8 // indirect
	gitlab.com/yawning/edwards25519-extra.git v0.0.0-20211229043746-2f91fcc9fbdb // indirect
	go.opencensus.io v0.24.0 // indirect
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.42.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/internal/retry v1.16.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlpmetric v0.39.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp v0.39.0 // indirect
	go.opentelemetry.io/otel/metric v1.16.0 // indirect
	go.opentelemetry.io/otel/sdk/metric v0.39.0 // indirect
	go.opentelemetry.io/proto/otlp v0.19.0 // indirect
	go.uber.org/atomic v1.9.0 // indirect
	go.uber.org/goleak v1.1.12 // indirect
	go.uber.org/multierr v1.8.0 // indirect
	go.uber.org/zap v1.21.0 // indirect
	golang.org/x/crypto v0.9.0 // indirect
	golang.org/x/exp v0.0.0-20221205204356-47842c84f3db // indirect
	golang.org/x/mod v0.8.0 // indirect
	golang.org/x/oauth2 v0.6.0 // indirect
	golang.org/x/sync v0.1.0 // indirect
	golang.org/x/sys v0.8.0 // indirect
	golang.org/x/text v0.9.0 // indirect
	golang.org/x/tools v0.6.0 // indirect
	google.golang.org/appengine v1.6.7 // indirect
	google.golang.org/genproto v0.0.0-20230306155012-7f2fa6fef1f4 // indirect
	google.golang.org/grpc v1.55.0 // indirect
	google.golang.org/protobuf v1.30.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
	nhooyr.io/websocket v1.8.7 // indirect
)

//replace github.com/lucas-clemente/quic-go => github.com/getlantern/quic-go v0.31.1-0.20230104154904-d810c964a217

// git.apache.org isn't working at the moment, use mirror (should probably switch back once we can)
replace git.apache.org/thrift.git => github.com/apache/thrift v0.0.0-20180902110319-2566ecd5d999

// Waiting on https://github.com/mitchellh/panicwrap/pull/27 to be merged upstream
replace github.com/mitchellh/panicwrap v1.0.0 => github.com/getlantern/panicwrap v0.0.0-20200707191944-9ba45baf8e51

replace github.com/refraction-networking/utls => github.com/getlantern/utls v0.0.0-20200903013459-0c02248f7ce1

// Version 0.5.6 has a security issue. As this is an indirect dependency, we need to use 'replace'
// over 'require' to fully remove references to 0.5.6 in go.sum
replace github.com/ulikunitz/xz => github.com/ulikunitz/xz v0.5.8

replace github.com/Jigsaw-Code/outline-ss-server => github.com/getlantern/lantern-shadowsocks v1.3.6-0.20230301223223-150b18ac427d
