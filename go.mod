module github.com/getlantern/http-proxy-lantern/v2

go 1.18

require (
	git.torproject.org/pluggable-transports/goptlib.git v1.2.0
	github.com/blang/semver v0.0.0-20180723201105-3c1074078d32
	github.com/getlantern/bbrconn v0.0.0-20180619163322-86cf8c16f3d0
	github.com/getlantern/cmux/v2 v2.0.0-20200905031936-c55b16ee8462
	github.com/getlantern/cmuxprivate v0.0.0-20200905032931-afb63438e40b
	github.com/getlantern/ema v0.0.0-20190620044903-5943d28f40e4
	github.com/getlantern/enhttp v0.0.0-20190401024120-a974fa851e3c
	github.com/getlantern/errors v1.0.1
	github.com/getlantern/golog v0.0.0-20210606115803-bce9f9fe5a5f
	github.com/getlantern/gonat v0.0.0-20201001145726-634575ba87fb
	github.com/getlantern/http-proxy v0.0.3-0.20211022035117-86faba795750
	github.com/getlantern/kcpwrapper v0.0.0-20220503142841-b0e764933966
	github.com/getlantern/keyman v0.0.0-20210218183930-5e48f8ced961
	github.com/getlantern/lampshade v0.0.0-20200303040944-fe53f13203e9
	github.com/getlantern/lantern-shadowsocks v1.3.6-0.20210601195915-e04471aa4920
	github.com/getlantern/mtime v0.0.0-20200417132445-23682092d1f7
	github.com/getlantern/multipath v0.0.0-20211105161347-48cd80ec7050
	github.com/getlantern/netx v0.0.0-20211206143627-7ccfeb739cbd
	github.com/getlantern/ops v0.0.0-20200403153110-8476b16edcd6
	github.com/getlantern/packetforward v0.0.0-20201001150407-c68a447b0360
	github.com/getlantern/pcapper v0.0.0-20181212174440-a8b1a3ff0cde
	github.com/getlantern/proxy/v2 v2.0.1-0.20220303164029-b34b76e0e581
	github.com/getlantern/psmux v1.5.15-0.20200903210100-947ca5d91683
	github.com/getlantern/quicwrapper v0.0.0-20211104133553-140f96139f9f
	github.com/getlantern/tinywss v0.0.0-20200121221108-851921f95ad7
	github.com/getlantern/tlsdefaults v0.0.0-20171004213447-cf35cfd0b1b4
	github.com/getlantern/tlsmasq v0.4.6
	github.com/getlantern/tlsutil v0.5.1
	github.com/getlantern/withtimeout v0.0.0-20160829163843-511f017cd913
	github.com/gonum/stat v0.0.0-20180125090729-ec9c8a1062f4
	github.com/juju/ratelimit v1.0.1
	github.com/mitchellh/panicwrap v1.0.0
	github.com/refraction-networking/utls v0.0.0-20200729012536-186025ac7b77
	github.com/stretchr/testify v1.7.0
	github.com/vharitonsky/iniflags v0.0.0-20180513140207-a33cd0b5f3de
	github.com/xtaci/smux v1.5.15-0.20200704123958-f7188026ba01
	gitlab.com/yawning/obfs4.git v0.0.0-20220204003609-77af0cba934d
	golang.org/x/net v0.0.0-20210503060351-7fd8e65b6420
)

require (
	filippo.io/edwards25519 v1.0.0-rc.1.0.20210721174708-390f27c3be20 // indirect
	github.com/Yawning/chacha20 v0.0.0-20170904085104-e3b1f968fc63 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/cespare/xxhash/v2 v2.1.1 // indirect
	github.com/cheekybits/genny v1.0.0 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/dchest/siphash v1.2.1 // indirect
	github.com/dustin/go-humanize v1.0.0 // indirect
	github.com/dvyukov/go-fuzz v0.0.0-20210429054444-fca39067bc72 // indirect
	github.com/fsnotify/fsnotify v1.4.9 // indirect
	github.com/getlantern/bufconn v0.0.0-20190625204133-a08544339f8d // indirect
	github.com/getlantern/byteexec v0.0.0-20170405023437-4cfb26ec74f4 // indirect
	github.com/getlantern/cmux v0.0.0-20200905031936-c55b16ee8462 // indirect
	github.com/getlantern/context v0.0.0-20190109183933-c447772a6520 // indirect
	github.com/getlantern/elevate v0.0.0-20200430163644-2881a121236d // indirect
	github.com/getlantern/eventual v0.0.0-20180125201821-84b02499361b // indirect
	github.com/getlantern/fdcount v0.0.0-20210503151800-5decd65b3731 // indirect
	github.com/getlantern/filepersist v0.0.0-20160317154340-c5f0cd24e799 // indirect
	github.com/getlantern/framed v0.0.0-20190601192238-ceb6431eeede // indirect
	github.com/getlantern/go-cache v0.0.0-20141028142048-88b53914f467 // indirect
	github.com/getlantern/grtrack v0.0.0-20160824195228-cbf67d3fa0fd // indirect
	github.com/getlantern/hex v0.0.0-20190417191902-c6586a6fe0b7 // indirect
	github.com/getlantern/hidden v0.0.0-20201229170000-e66e7f878730 // indirect
	github.com/getlantern/idletiming v0.0.0-20200228204104-10036786eac5 // indirect
	github.com/getlantern/iptool v0.0.0-20210721034953-519bf8ce0147 // indirect
	github.com/getlantern/kcp-go/v5 v5.0.0-20220503142114-f0c1cd6e1b54 // indirect
	github.com/getlantern/measured v0.0.0-20210507000559-ec5307b2b8be // indirect
	github.com/getlantern/mitm v0.0.0-20180205214248-4ce456bae650 // indirect
	github.com/getlantern/preconn v1.0.0 // indirect
	github.com/getlantern/reconn v0.0.0-20161128113912-7053d017511c // indirect
	github.com/getlantern/ring v0.0.0-20181206150603-dd46ce8faa01 // indirect
	github.com/getlantern/uuid v1.2.0 // indirect
	github.com/go-stack/stack v1.8.0 // indirect
	github.com/go-task/slim-sprig v0.0.0-20210107165309-348f09dbbbc0 // indirect
	github.com/golang/gddo v0.0.0-20190419222130-af0f2af80721 // indirect
	github.com/golang/protobuf v1.5.2 // indirect
	github.com/golang/snappy v0.0.3 // indirect
	github.com/gonum/blas v0.0.0-20180125090452-e7c5890b24cf // indirect
	github.com/gonum/floats v0.0.0-20180125090339-7de1f4ea7ab5 // indirect
	github.com/gonum/integrate v0.0.0-20181209220457-a422b5c0fdf2 // indirect
	github.com/gonum/internal v0.0.0-20180125090855-fda53f8d2571 // indirect
	github.com/gonum/lapack v0.0.0-20180125091020-f0b8b25edece // indirect
	github.com/gonum/matrix v0.0.0-20180124231301-a41cc49d4c29 // indirect
	github.com/google/go-cmp v0.5.7 // indirect
	github.com/google/gopacket v1.1.17 // indirect
	github.com/google/uuid v1.1.2 // indirect
	github.com/hashicorp/golang-lru v0.5.3 // indirect
	github.com/klauspost/cpuid v1.3.1 // indirect
	github.com/klauspost/reedsolomon v1.9.9 // indirect
	github.com/libp2p/go-buffer-pool v0.0.2 // indirect
	github.com/lucas-clemente/quic-go v0.7.1-0.20190207125157-7dc4be2ce994 // indirect
	github.com/marten-seemann/qtls-go1-16 v0.1.4 // indirect
	github.com/marten-seemann/qtls-go1-17 v0.1.0 // indirect
	github.com/marten-seemann/qtls-go1-18 v0.1.0-beta.1 // indirect
	github.com/matttproud/golang_protobuf_extensions v1.0.1 // indirect
	github.com/mdlayher/netlink v1.1.0 // indirect
	github.com/mikioh/tcp v0.0.0-20180707144002-02a37043a4f7 // indirect
	github.com/mikioh/tcpinfo v0.0.0-20180831101334-131b59fef27f // indirect
	github.com/mikioh/tcpopt v0.0.0-20180707144150-7178f18b4ea8 // indirect
	github.com/mmcloughlin/avo v0.0.0-20200803215136-443f81d77104 // indirect
	github.com/nxadm/tail v1.4.8 // indirect
	github.com/onsi/ginkgo v1.16.4 // indirect
	github.com/op/go-logging v0.0.0-20160315200505-970db520ece7 // indirect
	github.com/oschwald/geoip2-golang v1.4.0 // indirect
	github.com/oschwald/maxminddb-golang v1.6.0 // indirect
	github.com/oxtoacart/bpool v0.0.0-20190530202638-03653db5a59c // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/prometheus/client_golang v1.7.1 // indirect
	github.com/prometheus/client_model v0.2.0 // indirect
	github.com/prometheus/common v0.10.0 // indirect
	github.com/prometheus/procfs v0.1.3 // indirect
	github.com/shadowsocks/go-shadowsocks2 v0.1.4-0.20201002022019-75d43273f5a5 // indirect
	github.com/songgao/water v0.0.0-20200317203138-2b4b6d7c09d8 // indirect
	github.com/templexxx/cpu v0.0.8 // indirect
	github.com/templexxx/xorsimd v0.4.1 // indirect
	github.com/ti-mo/conntrack v0.3.0 // indirect
	github.com/ti-mo/netfilter v0.3.1 // indirect
	github.com/tjfoc/gmsm v1.3.2 // indirect
	gitlab.com/yawning/edwards25519-extra.git v0.0.0-20211229043746-2f91fcc9fbdb // indirect
	golang.org/x/crypto v0.0.0-20210711020723-a769d52b0f97 // indirect
	golang.org/x/mod v0.4.2 // indirect
	golang.org/x/sync v0.0.0-20210220032951-036812b2e83c // indirect
	golang.org/x/sys v0.0.0-20220502124256-b6088ccd6cba // indirect
	golang.org/x/tools v0.1.5 // indirect
	golang.org/x/xerrors v0.0.0-20200804184101-5ec99f83aff1 // indirect
	google.golang.org/protobuf v1.27.1 // indirect
	gopkg.in/tomb.v1 v1.0.0-20141024135613-dd632973f1e7 // indirect
	gopkg.in/yaml.v3 v3.0.0-20200313102051-9f266ea9e77c // indirect
)

replace github.com/lucas-clemente/quic-go => github.com/getlantern/quic-go v0.7.1-0.20220215050330-93bd217f5741

// git.apache.org isn't working at the moment, use mirror (should probably switch back once we can)
replace git.apache.org/thrift.git => github.com/apache/thrift v0.0.0-20180902110319-2566ecd5d999

// Waiting on https://github.com/mitchellh/panicwrap/pull/27 to be merged upstream
replace github.com/mitchellh/panicwrap v1.0.0 => github.com/getlantern/panicwrap v0.0.0-20200707191944-9ba45baf8e51

replace github.com/refraction-networking/utls => github.com/getlantern/utls v0.0.0-20200903013459-0c02248f7ce1

// Version 0.5.6 has a security issue. As this is an indirect dependency, we need to use 'replace'
// over 'require' to fully remove references to 0.5.6 in go.sum
replace github.com/ulikunitz/xz => github.com/ulikunitz/xz v0.5.8
