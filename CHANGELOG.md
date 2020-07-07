# Changelog

## [list](https://github.com/getlantern/http-proxy-lantern/tree/list) (2020-07-07)

[Full Changelog](https://github.com/getlantern/http-proxy-lantern/compare/v2.6.12...list)

## [v2.6.12](https://github.com/getlantern/http-proxy-lantern/tree/v2.6.12) (2020-07-07)

[Full Changelog](https://github.com/getlantern/http-proxy-lantern/compare/v2.6.11...v2.6.12)

**Merged pull requests:**

- Use forked panicwrap dependency until upstream PR is merged [\#411](https://github.com/getlantern/http-proxy-lantern/pull/411) ([max-b](https://github.com/max-b))

## [v2.6.11](https://github.com/getlantern/http-proxy-lantern/tree/v2.6.11) (2020-06-29)

[Full Changelog](https://github.com/getlantern/http-proxy-lantern/compare/v2.6.10...v2.6.11)

**Merged pull requests:**

- Not logging closed pipe errors, reduced log level on stackdriver trac… [\#410](https://github.com/getlantern/http-proxy-lantern/pull/410) ([oxtoacart](https://github.com/oxtoacart))

## [v2.6.10](https://github.com/getlantern/http-proxy-lantern/tree/v2.6.10) (2020-06-25)

[Full Changelog](https://github.com/getlantern/http-proxy-lantern/compare/2.6.9...v2.6.10)

**Merged pull requests:**

- Support disabling missing session ticket reaction [\#409](https://github.com/getlantern/http-proxy-lantern/pull/409) ([oxtoacart](https://github.com/oxtoacart))
- Easier building on non-Linux platforms [\#408](https://github.com/getlantern/http-proxy-lantern/pull/408) ([hwh33](https://github.com/hwh33))
- Add replica search to list of configured domains [\#407](https://github.com/getlantern/http-proxy-lantern/pull/407) ([max-b](https://github.com/max-b))
- Bump module to v2 [\#406](https://github.com/getlantern/http-proxy-lantern/pull/406) ([hwh33](https://github.com/hwh33))
- Require properly-formatted semantic version [\#405](https://github.com/getlantern/http-proxy-lantern/pull/405) ([hwh33](https://github.com/hwh33))

## [2.6.9](https://github.com/getlantern/http-proxy-lantern/tree/2.6.9) (2020-06-11)

[Full Changelog](https://github.com/getlantern/http-proxy-lantern/compare/v2.6.9...2.6.9)

## [v2.6.9](https://github.com/getlantern/http-proxy-lantern/tree/v2.6.9) (2020-06-11)

[Full Changelog](https://github.com/getlantern/http-proxy-lantern/compare/2.6.8...v2.6.9)

**Merged pull requests:**

- Signal hotfix [\#404](https://github.com/getlantern/http-proxy-lantern/pull/404) ([myleshorton](https://github.com/myleshorton))
- Bump go version to 1.14 [\#403](https://github.com/getlantern/http-proxy-lantern/pull/403) ([myleshorton](https://github.com/myleshorton))
- Depends on system default to handle OS signals. [\#402](https://github.com/getlantern/http-proxy-lantern/pull/402) ([joesis](https://github.com/joesis))

## [2.6.8](https://github.com/getlantern/http-proxy-lantern/tree/2.6.8) (2020-06-04)

[Full Changelog](https://github.com/getlantern/http-proxy-lantern/compare/2.6.7...2.6.8)

**Merged pull requests:**

- Expose a few TCP per-connection level metrics to Prometheus for block detection [\#401](https://github.com/getlantern/http-proxy-lantern/pull/401) ([joesis](https://github.com/joesis))

## [2.6.7](https://github.com/getlantern/http-proxy-lantern/tree/2.6.7) (2020-06-03)

[Full Changelog](https://github.com/getlantern/http-proxy-lantern/compare/2.6.6...2.6.7)

**Merged pull requests:**

- Multiplex tlsmasq connections [\#400](https://github.com/getlantern/http-proxy-lantern/pull/400) ([hwh33](https://github.com/hwh33))
- Do not report suspected probes as errors [\#399](https://github.com/getlantern/http-proxy-lantern/pull/399) ([hwh33](https://github.com/hwh33))
- Use the default TLS version, now 1.3, when not using sessions [\#398](https://github.com/getlantern/http-proxy-lantern/pull/398) ([myleshorton](https://github.com/myleshorton))

## [2.6.6](https://github.com/getlantern/http-proxy-lantern/tree/2.6.6) (2020-04-30)

[Full Changelog](https://github.com/getlantern/http-proxy-lantern/compare/2.6.5...2.6.6)

**Merged pull requests:**

- Do not complain about absent reaction if ticket is not required [\#397](https://github.com/getlantern/http-proxy-lantern/pull/397) ([joesis](https://github.com/joesis))
- also report proxy name to stack driver [\#396](https://github.com/getlantern/http-proxy-lantern/pull/396) ([joesis](https://github.com/joesis))

## [2.6.5](https://github.com/getlantern/http-proxy-lantern/tree/2.6.5) (2020-04-28)

[Full Changelog](https://github.com/getlantern/http-proxy-lantern/compare/2.6.4...2.6.5)

**Merged pull requests:**

- Fix panicwrap pprof bind conflict [\#395](https://github.com/getlantern/http-proxy-lantern/pull/395) ([max-b](https://github.com/max-b))

## [2.6.4](https://github.com/getlantern/http-proxy-lantern/tree/2.6.4) (2020-04-28)

[Full Changelog](https://github.com/getlantern/http-proxy-lantern/compare/2.6.3...2.6.4)

**Merged pull requests:**

- Use panicwrap to report panics to stackdriver and report all fatal errors [\#394](https://github.com/getlantern/http-proxy-lantern/pull/394) ([max-b](https://github.com/max-b))

## [2.6.3](https://github.com/getlantern/http-proxy-lantern/tree/2.6.3) (2020-04-28)

[Full Changelog](https://github.com/getlantern/http-proxy-lantern/compare/2.6.2...2.6.3)

**Merged pull requests:**

- Override empty country code in reporting Redis [\#393](https://github.com/getlantern/http-proxy-lantern/pull/393) ([joesis](https://github.com/joesis))

## [2.6.2](https://github.com/getlantern/http-proxy-lantern/tree/2.6.2) (2020-04-27)

[Full Changelog](https://github.com/getlantern/http-proxy-lantern/compare/2.6.1...2.6.2)

**Merged pull requests:**

- update to latest borda package [\#392](https://github.com/getlantern/http-proxy-lantern/pull/392) ([joesis](https://github.com/joesis))

## [2.6.1](https://github.com/getlantern/http-proxy-lantern/tree/2.6.1) (2020-04-23)

[Full Changelog](https://github.com/getlantern/http-proxy-lantern/compare/2.6.0...2.6.1)

**Merged pull requests:**

- extracted geo to a reusable package [\#385](https://github.com/getlantern/http-proxy-lantern/pull/385) ([joesis](https://github.com/joesis))

## [2.6.0](https://github.com/getlantern/http-proxy-lantern/tree/2.6.0) (2020-04-21)

[Full Changelog](https://github.com/getlantern/http-proxy-lantern/compare/2.5.8...2.6.0)

**Merged pull requests:**

- Updated to latest packetforward with performance/cpu usage improvements [\#391](https://github.com/getlantern/http-proxy-lantern/pull/391) ([oxtoacart](https://github.com/oxtoacart))

## [2.5.8](https://github.com/getlantern/http-proxy-lantern/tree/2.5.8) (2020-04-20)

[Full Changelog](https://github.com/getlantern/http-proxy-lantern/compare/2.5.7...2.5.8)

**Merged pull requests:**

- Updated to latest packetforward and gonat [\#390](https://github.com/getlantern/http-proxy-lantern/pull/390) ([oxtoacart](https://github.com/oxtoacart))

## [2.5.7](https://github.com/getlantern/http-proxy-lantern/tree/2.5.7) (2020-04-07)

[Full Changelog](https://github.com/getlantern/http-proxy-lantern/compare/2.5.6...2.5.7)

**Merged pull requests:**

- Updated to latest http-proxy with tiny log fix [\#389](https://github.com/getlantern/http-proxy-lantern/pull/389) ([myleshorton](https://github.com/myleshorton))

## [2.5.6](https://github.com/getlantern/http-proxy-lantern/tree/2.5.6) (2020-04-06)

[Full Changelog](https://github.com/getlantern/http-proxy-lantern/compare/2.5.5...2.5.6)

**Merged pull requests:**

- Added debug logging of missing ticket reactions [\#388](https://github.com/getlantern/http-proxy-lantern/pull/388) ([oxtoacart](https://github.com/oxtoacart))

## [2.5.5](https://github.com/getlantern/http-proxy-lantern/tree/2.5.5) (2020-03-30)

[Full Changelog](https://github.com/getlantern/http-proxy-lantern/compare/2.5.4...2.5.5)

**Merged pull requests:**

- Include general server info like name, dc, etc with xfer stats [\#387](https://github.com/getlantern/http-proxy-lantern/pull/387) ([oxtoacart](https://github.com/oxtoacart))
- Update tlsmasq to include bug fix [\#386](https://github.com/getlantern/http-proxy-lantern/pull/386) ([hwh33](https://github.com/hwh33))

## [2.5.4](https://github.com/getlantern/http-proxy-lantern/tree/2.5.4) (2020-03-10)

[Full Changelog](https://github.com/getlantern/http-proxy-lantern/compare/2.5.3...2.5.4)

**Merged pull requests:**

- Just made this a little more DRY [\#384](https://github.com/getlantern/http-proxy-lantern/pull/384) ([myleshorton](https://github.com/myleshorton))
- Don't overwrite measured context from initial request in persistent H… [\#383](https://github.com/getlantern/http-proxy-lantern/pull/383) ([oxtoacart](https://github.com/oxtoacart))

## [2.5.3](https://github.com/getlantern/http-proxy-lantern/tree/2.5.3) (2020-03-10)

[Full Changelog](https://github.com/getlantern/http-proxy-lantern/compare/2.5.2...2.5.3)

**Merged pull requests:**

- HOTFIX: add missing labels for bytesSent and bytesRecv [\#382](https://github.com/getlantern/http-proxy-lantern/pull/382) ([oxtoacart](https://github.com/oxtoacart))

## [2.5.2](https://github.com/getlantern/http-proxy-lantern/tree/2.5.2) (2020-03-10)

[Full Changelog](https://github.com/getlantern/http-proxy-lantern/compare/2.5.1...2.5.2)

**Merged pull requests:**

- Track app version and platform along with proxied bytes metric in Pro… [\#381](https://github.com/getlantern/http-proxy-lantern/pull/381) ([oxtoacart](https://github.com/oxtoacart))

## [2.5.1](https://github.com/getlantern/http-proxy-lantern/tree/2.5.1) (2020-03-03)

[Full Changelog](https://github.com/getlantern/http-proxy-lantern/compare/2.5.0...2.5.1)

**Merged pull requests:**

- Apply idletiming to lampshade streams [\#380](https://github.com/getlantern/http-proxy-lantern/pull/380) ([oxtoacart](https://github.com/oxtoacart))

## [2.5.0](https://github.com/getlantern/http-proxy-lantern/tree/2.5.0) (2020-03-03)

[Full Changelog](https://github.com/getlantern/http-proxy-lantern/compare/2.4.8...2.5.0)

**Merged pull requests:**

- Updated to latest lampshade and proxy packages [\#379](https://github.com/getlantern/http-proxy-lantern/pull/379) ([oxtoacart](https://github.com/oxtoacart))
- use the latest lampshade package [\#378](https://github.com/getlantern/http-proxy-lantern/pull/378) ([joesis](https://github.com/joesis))

## [2.4.8](https://github.com/getlantern/http-proxy-lantern/tree/2.4.8) (2020-02-27)

[Full Changelog](https://github.com/getlantern/http-proxy-lantern/compare/2.4.7...2.4.8)

**Merged pull requests:**

- Updated to latest lampshade with stream closing improvements [\#377](https://github.com/getlantern/http-proxy-lantern/pull/377) ([oxtoacart](https://github.com/oxtoacart))
- export proxied bytes to Prometheus [\#376](https://github.com/getlantern/http-proxy-lantern/pull/376) ([joesis](https://github.com/joesis))
- tlsmasq: allow for configurable minimum version and cipher suites [\#375](https://github.com/getlantern/http-proxy-lantern/pull/375) ([hwh33](https://github.com/hwh33))

## [2.4.7](https://github.com/getlantern/http-proxy-lantern/tree/2.4.7) (2020-02-11)

[Full Changelog](https://github.com/getlantern/http-proxy-lantern/compare/2.4.6...2.4.7)

**Merged pull requests:**

- use our own CDN distribution to overcome rate limit [\#374](https://github.com/getlantern/http-proxy-lantern/pull/374) ([joesis](https://github.com/joesis))

## [2.4.6](https://github.com/getlantern/http-proxy-lantern/tree/2.4.6) (2020-02-11)

[Full Changelog](https://github.com/getlantern/http-proxy-lantern/compare/2.4.5...2.4.6)

**Merged pull requests:**

- Switch to lookup GeoLite2 Country database locally [\#373](https://github.com/getlantern/http-proxy-lantern/pull/373) ([joesis](https://github.com/joesis))

## [2.4.5](https://github.com/getlantern/http-proxy-lantern/tree/2.4.5) (2020-02-06)

[Full Changelog](https://github.com/getlantern/http-proxy-lantern/compare/2.4.4...2.4.5)

## [2.4.4](https://github.com/getlantern/http-proxy-lantern/tree/2.4.4) (2020-02-06)

[Full Changelog](https://github.com/getlantern/http-proxy-lantern/compare/2.4.3...2.4.4)

**Merged pull requests:**

- Updated to latest lampshade to fix goroutine leak [\#372](https://github.com/getlantern/http-proxy-lantern/pull/372) ([oxtoacart](https://github.com/oxtoacart))

## [2.4.3](https://github.com/getlantern/http-proxy-lantern/tree/2.4.3) (2020-02-04)

[Full Changelog](https://github.com/getlantern/http-proxy-lantern/compare/2.4.2...2.4.3)

**Merged pull requests:**

- Updated to latest lampshade with latest probing mitigation [\#371](https://github.com/getlantern/http-proxy-lantern/pull/371) ([oxtoacart](https://github.com/oxtoacart))
- do not crash if missing-session-ticket-reaction parameter is invalid [\#370](https://github.com/getlantern/http-proxy-lantern/pull/370) ([joesis](https://github.com/joesis))
- Quic ietf draft 24 [\#369](https://github.com/getlantern/http-proxy-lantern/pull/369) ([forkner](https://github.com/forkner))

## [2.4.2](https://github.com/getlantern/http-proxy-lantern/tree/2.4.2) (2020-01-24)

[Full Changelog](https://github.com/getlantern/http-proxy-lantern/compare/2.4.1...2.4.2)

**Merged pull requests:**

- Integrated latest flashlight with unlimited client init read timeout [\#368](https://github.com/getlantern/http-proxy-lantern/pull/368) ([oxtoacart](https://github.com/oxtoacart))
- Integrate tlsmasq protocol [\#363](https://github.com/getlantern/http-proxy-lantern/pull/363) ([max-b](https://github.com/max-b))

## [2.4.1](https://github.com/getlantern/http-proxy-lantern/tree/2.4.1) (2020-01-21)

[Full Changelog](https://github.com/getlantern/http-proxy-lantern/compare/2.4.0...2.4.1)

**Merged pull requests:**

- update to latest cmux and tinywss [\#367](https://github.com/getlantern/http-proxy-lantern/pull/367) ([joesis](https://github.com/joesis))

## [2.4.0](https://github.com/getlantern/http-proxy-lantern/tree/2.4.0) (2020-01-21)

[Full Changelog](https://github.com/getlantern/http-proxy-lantern/compare/2.3.10...2.4.0)

**Merged pull requests:**

- Address build issues in Makefile [\#366](https://github.com/getlantern/http-proxy-lantern/pull/366) ([max-b](https://github.com/max-b))
- Logging privacy sensitive messages at TRACE level so they get exclude… [\#365](https://github.com/getlantern/http-proxy-lantern/pull/365) ([oxtoacart](https://github.com/oxtoacart))
- Default to no-op instrument [\#364](https://github.com/getlantern/http-proxy-lantern/pull/364) ([hwh33](https://github.com/hwh33))
- a few small fixes on throttling [\#362](https://github.com/getlantern/http-proxy-lantern/pull/362) ([joesis](https://github.com/joesis))
- Use flags for new changelog generator [\#361](https://github.com/getlantern/http-proxy-lantern/pull/361) ([myleshorton](https://github.com/myleshorton))

## [2.3.10](https://github.com/getlantern/http-proxy-lantern/tree/2.3.10) (2019-12-20)

[Full Changelog](https://github.com/getlantern/http-proxy-lantern/compare/2.3.12...2.3.10)

**Merged pull requests:**

- Update quic go [\#360](https://github.com/getlantern/http-proxy-lantern/pull/360) ([myleshorton](https://github.com/myleshorton))

## [2.3.12](https://github.com/getlantern/http-proxy-lantern/tree/2.3.12) (2019-12-19)

[Full Changelog](https://github.com/getlantern/http-proxy-lantern/compare/2.3.11...2.3.12)

**Merged pull requests:**

- Not redirect if the request has no version header [\#359](https://github.com/getlantern/http-proxy-lantern/pull/359) ([joesis](https://github.com/joesis))

## [2.3.11](https://github.com/getlantern/http-proxy-lantern/tree/2.3.11) (2019-12-16)

[Full Changelog](https://github.com/getlantern/http-proxy-lantern/compare/2.3.9...2.3.11)

**Merged pull requests:**

- update lampshade to be compatible with old clients [\#358](https://github.com/getlantern/http-proxy-lantern/pull/358) ([joesis](https://github.com/joesis))
- defer correctly to measure version checks [\#357](https://github.com/getlantern/http-proxy-lantern/pull/357) ([joesis](https://github.com/joesis))
- measure version checks and redirects [\#356](https://github.com/getlantern/http-proxy-lantern/pull/356) ([joesis](https://github.com/joesis))
- export tls resumption configs as Prometheus labels [\#354](https://github.com/getlantern/http-proxy-lantern/pull/354) ([joesis](https://github.com/joesis))
- Allow configuring reaction to unexpected ClientHellos [\#353](https://github.com/getlantern/http-proxy-lantern/pull/353) ([joesis](https://github.com/joesis))

## [2.3.9](https://github.com/getlantern/http-proxy-lantern/tree/2.3.9) (2019-12-05)

[Full Changelog](https://github.com/getlantern/http-proxy-lantern/compare/2.3.8...2.3.9)

## [2.3.8](https://github.com/getlantern/http-proxy-lantern/tree/2.3.8) (2019-12-05)

[Full Changelog](https://github.com/getlantern/http-proxy-lantern/compare/2.3.7...2.3.8)

**Merged pull requests:**

- Integrated latest lampshade with support for mitigating replay attacks [\#355](https://github.com/getlantern/http-proxy-lantern/pull/355) ([oxtoacart](https://github.com/oxtoacart))
- stop collecting active probe source ip [\#352](https://github.com/getlantern/http-proxy-lantern/pull/352) ([joesis](https://github.com/joesis))

## [2.3.7](https://github.com/getlantern/http-proxy-lantern/tree/2.3.7) (2019-11-20)

[Full Changelog](https://github.com/getlantern/http-proxy-lantern/compare/2.3.6...2.3.7)

**Merged pull requests:**

- handle the case of zero byte session ticket separately [\#351](https://github.com/getlantern/http-proxy-lantern/pull/351) ([joesis](https://github.com/joesis))

## [2.3.6](https://github.com/getlantern/http-proxy-lantern/tree/2.3.6) (2019-11-19)

[Full Changelog](https://github.com/getlantern/http-proxy-lantern/compare/2.3.5...2.3.6)

**Merged pull requests:**

- more standard naming [\#350](https://github.com/getlantern/http-proxy-lantern/pull/350) ([myleshorton](https://github.com/myleshorton))
- Decrypt session tickets to see if they're ours [\#349](https://github.com/getlantern/http-proxy-lantern/pull/349) ([myleshorton](https://github.com/myleshorton))
- allow connections from localhost to have no session ticket [\#346](https://github.com/getlantern/http-proxy-lantern/pull/346) ([joesis](https://github.com/joesis))
- report unexpected client hello as suspected probing [\#345](https://github.com/getlantern/http-proxy-lantern/pull/345) ([joesis](https://github.com/joesis))
- Specify go minor version for Docker [\#344](https://github.com/getlantern/http-proxy-lantern/pull/344) ([myleshorton](https://github.com/myleshorton))
- Added test for aborting on ClientHello with no session tickets [\#343](https://github.com/getlantern/http-proxy-lantern/pull/343) ([myleshorton](https://github.com/myleshorton))
- Support Apache mimicry when multiplexing [\#342](https://github.com/getlantern/http-proxy-lantern/pull/342) ([hwh33](https://github.com/hwh33))
- Read hellos if we require tickets and kill clients w/o 'em [\#341](https://github.com/getlantern/http-proxy-lantern/pull/341) ([myleshorton](https://github.com/myleshorton))
- Require Go 1.13.x [\#340](https://github.com/getlantern/http-proxy-lantern/pull/340) ([hwh33](https://github.com/hwh33))

## [2.3.5](https://github.com/getlantern/http-proxy-lantern/tree/2.3.5) (2019-10-31)

[Full Changelog](https://github.com/getlantern/http-proxy-lantern/compare/2.3.4...2.3.5)

**Merged pull requests:**

- updating lampshade [\#339](https://github.com/getlantern/http-proxy-lantern/pull/339) ([myleshorton](https://github.com/myleshorton))

## [2.3.4](https://github.com/getlantern/http-proxy-lantern/tree/2.3.4) (2019-10-25)

[Full Changelog](https://github.com/getlantern/http-proxy-lantern/compare/2.3.3...2.3.4)

**Merged pull requests:**

- Use latest proxy package and go mod tidy [\#338](https://github.com/getlantern/http-proxy-lantern/pull/338) ([myleshorton](https://github.com/myleshorton))
- Fix Proxy-Connection header handling [\#337](https://github.com/getlantern/http-proxy-lantern/pull/337) ([myleshorton](https://github.com/myleshorton))

## [2.3.3](https://github.com/getlantern/http-proxy-lantern/tree/2.3.3) (2019-10-22)

[Full Changelog](https://github.com/getlantern/http-proxy-lantern/compare/2.3.1...2.3.3)

**Merged pull requests:**

- Fix Proxy-Connection header handling [\#336](https://github.com/getlantern/http-proxy-lantern/pull/336) ([myleshorton](https://github.com/myleshorton))

## [2.3.1](https://github.com/getlantern/http-proxy-lantern/tree/2.3.1) (2019-10-17)

[Full Changelog](https://github.com/getlantern/http-proxy-lantern/compare/2.3.0...2.3.1)

**Merged pull requests:**

- Capping max TLS version to 1.2 to allow tls session resumption trick … [\#334](https://github.com/getlantern/http-proxy-lantern/pull/334) ([oxtoacart](https://github.com/oxtoacart))
- Explicitly close incoming connections requesting internal services [\#332](https://github.com/getlantern/http-proxy-lantern/pull/332) ([myleshorton](https://github.com/myleshorton))

## [2.3.0](https://github.com/getlantern/http-proxy-lantern/tree/2.3.0) (2019-10-15)

[Full Changelog](https://github.com/getlantern/http-proxy-lantern/compare/2.3.2...2.3.0)

**Merged pull requests:**

- Use http2 for outgoing TLS client connections [\#330](https://github.com/getlantern/http-proxy-lantern/pull/330) ([myleshorton](https://github.com/myleshorton))

## [2.3.2](https://github.com/getlantern/http-proxy-lantern/tree/2.3.2) (2019-10-10)

[Full Changelog](https://github.com/getlantern/http-proxy-lantern/compare/2.2.24...2.3.2)

**Merged pull requests:**

- Added support for persistent session ticket keys [\#331](https://github.com/getlantern/http-proxy-lantern/pull/331) ([oxtoacart](https://github.com/oxtoacart))

## [2.2.24](https://github.com/getlantern/http-proxy-lantern/tree/2.2.24) (2019-09-17)

[Full Changelog](https://github.com/getlantern/http-proxy-lantern/compare/2.2.23...2.2.24)

**Merged pull requests:**

- exit if fail to setupPacketForward [\#329](https://github.com/getlantern/http-proxy-lantern/pull/329) ([joesis](https://github.com/joesis))
- update quicwrapper package [\#327](https://github.com/getlantern/http-proxy-lantern/pull/327) ([forkner](https://github.com/forkner))

## [2.2.23](https://github.com/getlantern/http-proxy-lantern/tree/2.2.23) (2019-08-21)

[Full Changelog](https://github.com/getlantern/http-proxy-lantern/compare/0.2.22...2.2.23)

**Merged pull requests:**

- oquic v0 [\#326](https://github.com/getlantern/http-proxy-lantern/pull/326) ([forkner](https://github.com/forkner))
- update tlsdialer to the correct latest master [\#325](https://github.com/getlantern/http-proxy-lantern/pull/325) ([joesis](https://github.com/joesis))
- update code and dependencies to use latest version of golog [\#324](https://github.com/getlantern/http-proxy-lantern/pull/324) ([joesis](https://github.com/joesis))
- Update anacrolix/missinggo to v1.1.1 [\#323](https://github.com/getlantern/http-proxy-lantern/pull/323) ([anacrolix](https://github.com/anacrolix))
- Fix typo [\#322](https://github.com/getlantern/http-proxy-lantern/pull/322) ([bcmertz](https://github.com/bcmertz))

## [0.2.22](https://github.com/getlantern/http-proxy-lantern/tree/0.2.22) (2019-07-11)

[Full Changelog](https://github.com/getlantern/http-proxy-lantern/compare/0.2.21...0.2.22)

**Closed issues:**

- Refactor pro-filter/pro-config [\#107](https://github.com/getlantern/http-proxy-lantern/issues/107)
- Limit global count of devices [\#91](https://github.com/getlantern/http-proxy-lantern/issues/91)
- Error forwarding [\#71](https://github.com/getlantern/http-proxy-lantern/issues/71)
- Test failed on Go 1.6 [\#37](https://github.com/getlantern/http-proxy-lantern/issues/37)

**Merged pull requests:**

- Update dependencies to use xtaci/smux v1.3.4 [\#321](https://github.com/getlantern/http-proxy-lantern/pull/321) ([joesis](https://github.com/joesis))
- update smux to fork, bring in memory fix for smux\#52 [\#320](https://github.com/getlantern/http-proxy-lantern/pull/320) ([forkner](https://github.com/forkner))
- update tinywss,smux,cmux to fix wss bbr estimate [\#319](https://github.com/getlantern/http-proxy-lantern/pull/319) ([forkner](https://github.com/forkner))
- add consecutive error count metrics for connection errors [\#318](https://github.com/getlantern/http-proxy-lantern/pull/318) ([joesis](https://github.com/joesis))
- Updated to latest borda client with correct gRPC hostname [\#317](https://github.com/getlantern/http-proxy-lantern/pull/317) ([oxtoacart](https://github.com/oxtoacart))

## [0.2.21](https://github.com/getlantern/http-proxy-lantern/tree/0.2.21) (2019-06-19)

[Full Changelog](https://github.com/getlantern/http-proxy-lantern/compare/0.2.20...0.2.21)

**Merged pull requests:**

- Updated to latest packetforward without verbose packet logging [\#316](https://github.com/getlantern/http-proxy-lantern/pull/316) ([oxtoacart](https://github.com/oxtoacart))
- Made libutp dependency match client [\#315](https://github.com/getlantern/http-proxy-lantern/pull/315) ([oxtoacart](https://github.com/oxtoacart))
- use our own fork of go-libutp [\#314](https://github.com/getlantern/http-proxy-lantern/pull/314) ([joesis](https://github.com/joesis))

## [0.2.20](https://github.com/getlantern/http-proxy-lantern/tree/0.2.20) (2019-06-03)

[Full Changelog](https://github.com/getlantern/http-proxy-lantern/compare/0.2.19...0.2.20)

**Merged pull requests:**

- Carry through cloudfront country header on wss requests [\#313](https://github.com/getlantern/http-proxy-lantern/pull/313) ([forkner](https://github.com/forkner))

## [0.2.19](https://github.com/getlantern/http-proxy-lantern/tree/0.2.19) (2019-06-02)

[Full Changelog](https://github.com/getlantern/http-proxy-lantern/compare/0.2.18...0.2.19)

**Merged pull requests:**

- Updated to packetforward with corrected buffer pool sizing [\#312](https://github.com/getlantern/http-proxy-lantern/pull/312) ([oxtoacart](https://github.com/oxtoacart))
- Make sure we have the right QUIC version and its dependencies [\#311](https://github.com/getlantern/http-proxy-lantern/pull/311) ([oxtoacart](https://github.com/oxtoacart))

## [0.2.18](https://github.com/getlantern/http-proxy-lantern/tree/0.2.18) (2019-05-31)

[Full Changelog](https://github.com/getlantern/http-proxy-lantern/compare/0.2.17...0.2.18)

**Merged pull requests:**

- Updated to latest packetforward [\#310](https://github.com/getlantern/http-proxy-lantern/pull/310) ([oxtoacart](https://github.com/oxtoacart))
- make wss respect the https flag [\#309](https://github.com/getlantern/http-proxy-lantern/pull/309) ([forkner](https://github.com/forkner))

## [0.2.17](https://github.com/getlantern/http-proxy-lantern/tree/0.2.17) (2019-05-10)

[Full Changelog](https://github.com/getlantern/http-proxy-lantern/compare/0.2.16...0.2.17)

**Merged pull requests:**

- set up tls on listener given to wss [\#308](https://github.com/getlantern/http-proxy-lantern/pull/308) ([forkner](https://github.com/forkner))
- Updated ipproxy and packetforward dependencies [\#307](https://github.com/getlantern/http-proxy-lantern/pull/307) ([oxtoacart](https://github.com/oxtoacart))
- Updated to latest borda client library [\#306](https://github.com/getlantern/http-proxy-lantern/pull/306) ([oxtoacart](https://github.com/oxtoacart))
- Multiplexing UTP [\#305](https://github.com/getlantern/http-proxy-lantern/pull/305) ([oxtoacart](https://github.com/oxtoacart))
- add options for websocket [\#304](https://github.com/getlantern/http-proxy-lantern/pull/304) ([forkner](https://github.com/forkner))
- Refactoring and bringing in latest utp dependencies [\#303](https://github.com/getlantern/http-proxy-lantern/pull/303) ([myleshorton](https://github.com/myleshorton))
- Added support for using utp in place of tcp [\#302](https://github.com/getlantern/http-proxy-lantern/pull/302) ([oxtoacart](https://github.com/oxtoacart))

## [0.2.16](https://github.com/getlantern/http-proxy-lantern/tree/0.2.16) (2019-04-01)

[Full Changelog](https://github.com/getlantern/http-proxy-lantern/compare/0.2.15...0.2.16)

**Merged pull requests:**

- use ticker to track blacklist [\#301](https://github.com/getlantern/http-proxy-lantern/pull/301) ([joesis](https://github.com/joesis))

## [0.2.15](https://github.com/getlantern/http-proxy-lantern/tree/0.2.15) (2019-03-28)

[Full Changelog](https://github.com/getlantern/http-proxy-lantern/compare/0.2.14...0.2.15)

**Merged pull requests:**

- Enable packetforwarding by default [\#300](https://github.com/getlantern/http-proxy-lantern/pull/300) ([oxtoacart](https://github.com/oxtoacart))
- RequestNewDeviceUsage: adding to ongoing only after successfully queued [\#298](https://github.com/getlantern/http-proxy-lantern/pull/298) ([joesis](https://github.com/joesis))

## [0.2.14](https://github.com/getlantern/http-proxy-lantern/tree/0.2.14) (2019-03-28)

[Full Changelog](https://github.com/getlantern/http-proxy-lantern/compare/0.2.13...0.2.14)

**Merged pull requests:**

- Export basic metrics to Prometheus [\#296](https://github.com/getlantern/http-proxy-lantern/pull/296) ([joesis](https://github.com/joesis))
- Ending dial op [\#295](https://github.com/getlantern/http-proxy-lantern/pull/295) ([oxtoacart](https://github.com/oxtoacart))
- Add support packet forwarding [\#291](https://github.com/getlantern/http-proxy-lantern/pull/291) ([oxtoacart](https://github.com/oxtoacart))

## [0.2.13](https://github.com/getlantern/http-proxy-lantern/tree/0.2.13) (2019-03-27)

[Full Changelog](https://github.com/getlantern/http-proxy-lantern/compare/0.2.12...0.2.13)

**Merged pull requests:**

- Applying reporting on dial metrics at a lower level [\#299](https://github.com/getlantern/http-proxy-lantern/pull/299) ([oxtoacart](https://github.com/oxtoacart))

## [0.2.12](https://github.com/getlantern/http-proxy-lantern/tree/0.2.12) (2019-03-21)

[Full Changelog](https://github.com/getlantern/http-proxy-lantern/compare/0.2.11...0.2.12)

## [0.2.11](https://github.com/getlantern/http-proxy-lantern/tree/0.2.11) (2019-03-18)

[Full Changelog](https://github.com/getlantern/http-proxy-lantern/compare/1.1.1...0.2.11)

## [1.1.1](https://github.com/getlantern/http-proxy-lantern/tree/1.1.1) (2019-03-18)

[Full Changelog](https://github.com/getlantern/http-proxy-lantern/compare/0.2.10...1.1.1)

**Merged pull requests:**

- explicitly use bash over default [\#294](https://github.com/getlantern/http-proxy-lantern/pull/294) ([bcmertz](https://github.com/bcmertz))
- Reporting time to dial origins [\#292](https://github.com/getlantern/http-proxy-lantern/pull/292) ([oxtoacart](https://github.com/oxtoacart))

## [0.2.10](https://github.com/getlantern/http-proxy-lantern/tree/0.2.10) (2019-03-09)

[Full Changelog](https://github.com/getlantern/http-proxy-lantern/compare/0.2.9...0.2.10)

**Merged pull requests:**

- Don't log individual errors when recording entries to borda [\#293](https://github.com/getlantern/http-proxy-lantern/pull/293) ([oxtoacart](https://github.com/oxtoacart))

## [0.2.9](https://github.com/getlantern/http-proxy-lantern/tree/0.2.9) (2019-02-15)

[Full Changelog](https://github.com/getlantern/http-proxy-lantern/compare/0.2.8...0.2.9)

**Merged pull requests:**

- Lampshade now acks quickly on first frame of connection to try and pr… [\#290](https://github.com/getlantern/http-proxy-lantern/pull/290) ([oxtoacart](https://github.com/oxtoacart))

## [0.2.8](https://github.com/getlantern/http-proxy-lantern/tree/0.2.8) (2019-02-13)

[Full Changelog](https://github.com/getlantern/http-proxy-lantern/compare/0.2.7...0.2.8)

**Merged pull requests:**

- Fixed rewriting of https [\#289](https://github.com/getlantern/http-proxy-lantern/pull/289) ([oxtoacart](https://github.com/oxtoacart))
- Update packages to fix missing commit in qtls dependency [\#288](https://github.com/getlantern/http-proxy-lantern/pull/288) ([forkner](https://github.com/forkner))

## [0.2.7](https://github.com/getlantern/http-proxy-lantern/tree/0.2.7) (2019-02-06)

[Full Changelog](https://github.com/getlantern/http-proxy-lantern/compare/0.2.6...0.2.7)

**Merged pull requests:**

- update quic package [\#287](https://github.com/getlantern/http-proxy-lantern/pull/287) ([forkner](https://github.com/forkner))
- Add ability to release linux binary on non-linux platforms [\#286](https://github.com/getlantern/http-proxy-lantern/pull/286) ([joesis](https://github.com/joesis))
- update quic package to pick up race fix [\#285](https://github.com/getlantern/http-proxy-lantern/pull/285) ([forkner](https://github.com/forkner))
- Fixed data race in obfs4listener [\#284](https://github.com/getlantern/http-proxy-lantern/pull/284) ([oxtoacart](https://github.com/oxtoacart))
- Strip Lantern internal headers and Proxy-Connection header if not goi… [\#283](https://github.com/getlantern/http-proxy-lantern/pull/283) ([oxtoacart](https://github.com/oxtoacart))
- update to latest quic packages [\#282](https://github.com/getlantern/http-proxy-lantern/pull/282) ([forkner](https://github.com/forkner))

## [0.2.6](https://github.com/getlantern/http-proxy-lantern/tree/0.2.6) (2019-01-08)

[Full Changelog](https://github.com/getlantern/http-proxy-lantern/compare/0.2.5...0.2.6)

**Merged pull requests:**

- Rewrite all methods but CONNECT to HTTPS for designated domains [\#281](https://github.com/getlantern/http-proxy-lantern/pull/281) ([joesis](https://github.com/joesis))
- update quic packages [\#280](https://github.com/getlantern/http-proxy-lantern/pull/280) ([forkner](https://github.com/forkner))
- bumped required version [\#279](https://github.com/getlantern/http-proxy-lantern/pull/279) ([myleshorton](https://github.com/myleshorton))
- not default to 8080 when addr is not supplied [\#278](https://github.com/getlantern/http-proxy-lantern/pull/278) ([joesis](https://github.com/joesis))

## [0.2.5](https://github.com/getlantern/http-proxy-lantern/tree/0.2.5) (2018-12-17)

[Full Changelog](https://github.com/getlantern/http-proxy-lantern/compare/0.2.4...0.2.5)

**Merged pull requests:**

- Tracking apache mimicry and connects without request in borda [\#277](https://github.com/getlantern/http-proxy-lantern/pull/277) ([oxtoacart](https://github.com/oxtoacart))
- http-proxy captures packets and dumps them under unusual circumstances [\#273](https://github.com/getlantern/http-proxy-lantern/pull/273) ([oxtoacart](https://github.com/oxtoacart))

## [0.2.4](https://github.com/getlantern/http-proxy-lantern/tree/0.2.4) (2018-12-15)

[Full Changelog](https://github.com/getlantern/http-proxy-lantern/compare/0.2.3...0.2.4)

**Merged pull requests:**

- Updated to latest lampshade for tracking stats on streams [\#276](https://github.com/getlantern/http-proxy-lantern/pull/276) ([oxtoacart](https://github.com/oxtoacart))

## [0.2.3](https://github.com/getlantern/http-proxy-lantern/tree/0.2.3) (2018-12-13)

[Full Changelog](https://github.com/getlantern/http-proxy-lantern/compare/0.2.2...0.2.3)

**Merged pull requests:**

- Reporting all errors \(subject to sampling\) to borda [\#275](https://github.com/getlantern/http-proxy-lantern/pull/275) ([oxtoacart](https://github.com/oxtoacart))

## [0.2.2](https://github.com/getlantern/http-proxy-lantern/tree/0.2.2) (2018-12-13)

[Full Changelog](https://github.com/getlantern/http-proxy-lantern/compare/0.2.1...0.2.2)

**Merged pull requests:**

- Updated to latest lampshade to fix hanging in lampshade when closing … [\#274](https://github.com/getlantern/http-proxy-lantern/pull/274) ([oxtoacart](https://github.com/oxtoacart))
- add x-forwarded-for to pro server [\#272](https://github.com/getlantern/http-proxy-lantern/pull/272) ([myleshorton](https://github.com/myleshorton))

## [0.2.1](https://github.com/getlantern/http-proxy-lantern/tree/0.2.1) (2018-10-23)

[Full Changelog](https://github.com/getlantern/http-proxy-lantern/compare/0.2.0...0.2.1)

**Merged pull requests:**

- Made dist option without creating changelog [\#271](https://github.com/getlantern/http-proxy-lantern/pull/271) ([myleshorton](https://github.com/myleshorton))
- Added support for multiplexing http\(s\) and obfs4 [\#269](https://github.com/getlantern/http-proxy-lantern/pull/269) ([oxtoacart](https://github.com/oxtoacart))

## [0.2.0](https://github.com/getlantern/http-proxy-lantern/tree/0.2.0) (2018-10-04)

[Full Changelog](https://github.com/getlantern/http-proxy-lantern/compare/0.1.15...0.2.0)

**Merged pull requests:**

- Allow proxy server to be configured to respond immediately to CONNECT… [\#268](https://github.com/getlantern/http-proxy-lantern/pull/268) ([oxtoacart](https://github.com/oxtoacart))

## [0.1.15](https://github.com/getlantern/http-proxy-lantern/tree/0.1.15) (2018-10-04)

[Full Changelog](https://github.com/getlantern/http-proxy-lantern/compare/0.1.14...0.1.15)

**Merged pull requests:**

- Added support for DialTimeoutHeader on CONNECT requests [\#267](https://github.com/getlantern/http-proxy-lantern/pull/267) ([oxtoacart](https://github.com/oxtoacart))

## [0.1.14](https://github.com/getlantern/http-proxy-lantern/tree/0.1.14) (2018-09-30)

[Full Changelog](https://github.com/getlantern/http-proxy-lantern/compare/0.1.13...0.1.14)

**Merged pull requests:**

- Upgraded to go 1.10.4 [\#266](https://github.com/getlantern/http-proxy-lantern/pull/266) ([oxtoacart](https://github.com/oxtoacart))

## [0.1.13](https://github.com/getlantern/http-proxy-lantern/tree/0.1.13) (2018-09-30)

[Full Changelog](https://github.com/getlantern/http-proxy-lantern/compare/0.1.10...0.1.13)

**Merged pull requests:**

- Upgraded to go 1.11 [\#265](https://github.com/getlantern/http-proxy-lantern/pull/265) ([oxtoacart](https://github.com/oxtoacart))
- Updated dependencies [\#264](https://github.com/getlantern/http-proxy-lantern/pull/264) ([oxtoacart](https://github.com/oxtoacart))
- Fix a few bugs to show the upgrade notice [\#262](https://github.com/getlantern/http-proxy-lantern/pull/262) ([joesis](https://github.com/joesis))
- read full request before sending 302 response [\#261](https://github.com/getlantern/http-proxy-lantern/pull/261) ([joesis](https://github.com/joesis))
- Revert "Changed header name to always set True-Client-IP for config server" [\#260](https://github.com/getlantern/http-proxy-lantern/pull/260) ([joesis](https://github.com/joesis))
- wip quic support [\#258](https://github.com/getlantern/http-proxy-lantern/pull/258) ([forkner](https://github.com/forkner))

## [0.1.10](https://github.com/getlantern/http-proxy-lantern/tree/0.1.10) (2018-08-10)

[Full Changelog](https://github.com/getlantern/http-proxy-lantern/compare/0.1.12...0.1.10)

**Closed issues:**

- `preferIPV4Dialer` to respond a net.Error for time out [\#121](https://github.com/getlantern/http-proxy-lantern/issues/121)

**Merged pull requests:**

- Changed header name to always set True-Client-IP for config server [\#259](https://github.com/getlantern/http-proxy-lantern/pull/259) ([myleshorton](https://github.com/myleshorton))
- Suggested changes to \#256 [\#257](https://github.com/getlantern/http-proxy-lantern/pull/257) ([joesis](https://github.com/joesis))
- Allow turning the data cap off by setting the threshold [\#256](https://github.com/getlantern/http-proxy-lantern/pull/256) ([myleshorton](https://github.com/myleshorton))
- don't fail if unable to load throttle config in the first time [\#246](https://github.com/getlantern/http-proxy-lantern/pull/246) ([joesis](https://github.com/joesis))

## [0.1.12](https://github.com/getlantern/http-proxy-lantern/tree/0.1.12) (2018-07-14)

[Full Changelog](https://github.com/getlantern/http-proxy-lantern/compare/0.1.11...0.1.12)

## [0.1.11](https://github.com/getlantern/http-proxy-lantern/tree/0.1.11) (2018-07-13)

[Full Changelog](https://github.com/getlantern/http-proxy-lantern/compare/0.1.9...0.1.11)

**Merged pull requests:**

- Log when we close HTTP connections due to errors [\#254](https://github.com/getlantern/http-proxy-lantern/pull/254) ([myleshorton](https://github.com/myleshorton))
- Added VERSION check [\#253](https://github.com/getlantern/http-proxy-lantern/pull/253) ([myleshorton](https://github.com/myleshorton))
- Add IP check to deploy script [\#252](https://github.com/getlantern/http-proxy-lantern/pull/252) ([myleshorton](https://github.com/myleshorton))
- Better error logging for upstream timeouts and lower stackdriver sample [\#251](https://github.com/getlantern/http-proxy-lantern/pull/251) ([myleshorton](https://github.com/myleshorton))
- Update to latest proxy repo with server timing and better TLS SD logs [\#250](https://github.com/getlantern/http-proxy-lantern/pull/250) ([myleshorton](https://github.com/myleshorton))
- Better logging of upstream timeouts and stackdriver reports [\#249](https://github.com/getlantern/http-proxy-lantern/pull/249) ([myleshorton](https://github.com/myleshorton))

## [0.1.9](https://github.com/getlantern/http-proxy-lantern/tree/0.1.9) (2018-07-12)

[Full Changelog](https://github.com/getlantern/http-proxy-lantern/compare/0.1.8...0.1.9)

## [0.1.8](https://github.com/getlantern/http-proxy-lantern/tree/0.1.8) (2018-07-09)

[Full Changelog](https://github.com/getlantern/http-proxy-lantern/compare/0.1.7...0.1.8)

**Merged pull requests:**

- Add client IP to Lampshade init message error logging [\#248](https://github.com/getlantern/http-proxy-lantern/pull/248) ([myleshorton](https://github.com/myleshorton))
- Updated TLS error logging in proxy repository [\#247](https://github.com/getlantern/http-proxy-lantern/pull/247) ([myleshorton](https://github.com/myleshorton))

## [0.1.7](https://github.com/getlantern/http-proxy-lantern/tree/0.1.7) (2018-06-26)

[Full Changelog](https://github.com/getlantern/http-proxy-lantern/compare/0.1.6...0.1.7)

**Merged pull requests:**

- Adding adyenpayments.com to fasttrack domains [\#245](https://github.com/getlantern/http-proxy-lantern/pull/245) ([myleshorton](https://github.com/myleshorton))
- Add Alipay,PaymentWall,Stripe,Adyen to fasttrack domains/better defaults [\#244](https://github.com/getlantern/http-proxy-lantern/pull/244) ([myleshorton](https://github.com/myleshorton))
- Made OBFS4 handshake concurrency configurable [\#243](https://github.com/getlantern/http-proxy-lantern/pull/243) ([oxtoacart](https://github.com/oxtoacart))

## [0.1.6](https://github.com/getlantern/http-proxy-lantern/tree/0.1.6) (2018-06-26)

[Full Changelog](https://github.com/getlantern/http-proxy-lantern/compare/0.1.5...0.1.6)

**Merged pull requests:**

- Updated to latest proxy with better tls error logging [\#242](https://github.com/getlantern/http-proxy-lantern/pull/242) ([myleshorton](https://github.com/myleshorton))
- Locally set stackdriver sample percentage [\#241](https://github.com/getlantern/http-proxy-lantern/pull/241) ([myleshorton](https://github.com/myleshorton))

## [0.1.5](https://github.com/getlantern/http-proxy-lantern/tree/0.1.5) (2018-06-26)

[Full Changelog](https://github.com/getlantern/http-proxy-lantern/compare/0.1.4...0.1.5)

**Merged pull requests:**

-  Always include external IP in stackdriver errors [\#240](https://github.com/getlantern/http-proxy-lantern/pull/240) ([myleshorton](https://github.com/myleshorton))

## [0.1.4](https://github.com/getlantern/http-proxy-lantern/tree/0.1.4) (2018-06-26)

[Full Changelog](https://github.com/getlantern/http-proxy-lantern/compare/v0.1.3...0.1.4)

**Merged pull requests:**

- back to per connection limiting only [\#235](https://github.com/getlantern/http-proxy-lantern/pull/235) ([forkner](https://github.com/forkner))

## [v0.1.3](https://github.com/getlantern/http-proxy-lantern/tree/v0.1.3) (2018-06-25)

[Full Changelog](https://github.com/getlantern/http-proxy-lantern/compare/v0.1.2...v0.1.3)

**Merged pull requests:**

- Return an error if client hello cipher suites are empty [\#239](https://github.com/getlantern/http-proxy-lantern/pull/239) ([myleshorton](https://github.com/myleshorton))

## [v0.1.2](https://github.com/getlantern/http-proxy-lantern/tree/v0.1.2) (2018-06-22)

[Full Changelog](https://github.com/getlantern/http-proxy-lantern/compare/v0.1.1...v0.1.2)

**Merged pull requests:**

- Added another common client cipher suite to ignore and cleaned up [\#238](https://github.com/getlantern/http-proxy-lantern/pull/238) ([myleshorton](https://github.com/myleshorton))

## [v0.1.1](https://github.com/getlantern/http-proxy-lantern/tree/v0.1.1) (2018-06-22)

[Full Changelog](https://github.com/getlantern/http-proxy-lantern/compare/v0.1.0...v0.1.1)

**Merged pull requests:**

- Fix Makefile search for changelog binary [\#237](https://github.com/getlantern/http-proxy-lantern/pull/237) ([myleshorton](https://github.com/myleshorton))
- Updates dependency to proxy repo [\#236](https://github.com/getlantern/http-proxy-lantern/pull/236) ([myleshorton](https://github.com/myleshorton))
- Don't log when the user reports standard mobile cipher suites [\#234](https://github.com/getlantern/http-proxy-lantern/pull/234) ([myleshorton](https://github.com/myleshorton))
- Automatic changelog generation and version tagging [\#232](https://github.com/getlantern/http-proxy-lantern/pull/232) ([myleshorton](https://github.com/myleshorton))
- more debugging for tls connections [\#231](https://github.com/getlantern/http-proxy-lantern/pull/231) ([myleshorton](https://github.com/myleshorton))
- Add a sample rate for sending errors to Stackdriver [\#224](https://github.com/getlantern/http-proxy-lantern/pull/224) ([myleshorton](https://github.com/myleshorton))

## [v0.1.0](https://github.com/getlantern/http-proxy-lantern/tree/v0.1.0) (2018-06-19)

[Full Changelog](https://github.com/getlantern/http-proxy-lantern/compare/v0.0.20...v0.1.0)

**Fixed bugs:**

- CI panics in `tcpinfo.parseCCAlgorithmInfo` [\#181](https://github.com/getlantern/http-proxy-lantern/issues/181)

**Closed issues:**

- Add explicit config.ini parameter to tell free vs pro proxies [\#168](https://github.com/getlantern/http-proxy-lantern/issues/168)
- No redis for pro-only data centers [\#144](https://github.com/getlantern/http-proxy-lantern/issues/144)
- Redis Restart Wreaks Havoc on http-proxy [\#130](https://github.com/getlantern/http-proxy-lantern/issues/130)
- Seeing throttling messages on a pro server [\#126](https://github.com/getlantern/http-proxy-lantern/issues/126)
- Some headers getting truncated during reading [\#115](https://github.com/getlantern/http-proxy-lantern/issues/115)
- Enabling bandwidth tracking and throttling significantly increases load [\#110](https://github.com/getlantern/http-proxy-lantern/issues/110)
- Pro is not enabled properly [\#103](https://github.com/getlantern/http-proxy-lantern/issues/103)
- Preprocessor error parse request [\#101](https://github.com/getlantern/http-proxy-lantern/issues/101)
- XBQ header not appearing [\#100](https://github.com/getlantern/http-proxy-lantern/issues/100)
- Staging proxies overloaded even as they have no traffic [\#99](https://github.com/getlantern/http-proxy-lantern/issues/99)
- Access origin site through the same IP version as client [\#97](https://github.com/getlantern/http-proxy-lantern/issues/97)
- Throttling is applied to Pro users too [\#96](https://github.com/getlantern/http-proxy-lantern/issues/96)
- OBFS4 proxies getting stuck in accept loop [\#93](https://github.com/getlantern/http-proxy-lantern/issues/93)
- panics when `curl` directly [\#88](https://github.com/getlantern/http-proxy-lantern/issues/88)
- Report correct version to Loggly [\#76](https://github.com/getlantern/http-proxy-lantern/issues/76)
- Proxy might be throwing too many Bad Gateway errors [\#68](https://github.com/getlantern/http-proxy-lantern/issues/68)
- Secure Redis communication [\#60](https://github.com/getlantern/http-proxy-lantern/issues/60)
- Limit concurrent devices [\#50](https://github.com/getlantern/http-proxy-lantern/issues/50)
- Have lantern clients report blockages to Google Analytics via domain fronting [\#47](https://github.com/getlantern/http-proxy-lantern/issues/47)
- Not remove Pro related headers to config-server requests [\#44](https://github.com/getlantern/http-proxy-lantern/issues/44)
- Print out current version [\#16](https://github.com/getlantern/http-proxy-lantern/issues/16)

**Merged pull requests:**

- Switched to mikioh/tcpinfo now that it supports BBR and fixed race condition [\#230](https://github.com/getlantern/http-proxy-lantern/pull/230) ([oxtoacart](https://github.com/oxtoacart))
- Fixed data race in reporting xfer stats [\#229](https://github.com/getlantern/http-proxy-lantern/pull/229) ([oxtoacart](https://github.com/oxtoacart))
- Only including x-forwarded-for for our fasttrack domains [\#228](https://github.com/getlantern/http-proxy-lantern/pull/228) ([oxtoacart](https://github.com/oxtoacart))
- Updated deps [\#227](https://github.com/getlantern/http-proxy-lantern/pull/227) ([myleshorton](https://github.com/myleshorton))
- No longer logging to http-proxy.log [\#226](https://github.com/getlantern/http-proxy-lantern/pull/226) ([oxtoacart](https://github.com/oxtoacart))
- added remote addr logging [\#225](https://github.com/getlantern/http-proxy-lantern/pull/225) ([myleshorton](https://github.com/myleshorton))
- script for deploying to a given proxy [\#223](https://github.com/getlantern/http-proxy-lantern/pull/223) ([myleshorton](https://github.com/myleshorton))
- Added measured reporting for physical lampshade connections [\#222](https://github.com/getlantern/http-proxy-lantern/pull/222) ([oxtoacart](https://github.com/oxtoacart))
- update lampshade [\#221](https://github.com/getlantern/http-proxy-lantern/pull/221) ([forkner](https://github.com/forkner))
- Pull in recent versions of dependencies, in particular lampshade [\#220](https://github.com/getlantern/http-proxy-lantern/pull/220) ([oxtoacart](https://github.com/oxtoacart))
- add error reporting to stackdriver [\#219](https://github.com/getlantern/http-proxy-lantern/pull/219) ([myleshorton](https://github.com/myleshorton))
- Switched to latest netx that handles panics in BidiCopy [\#218](https://github.com/getlantern/http-proxy-lantern/pull/218) ([oxtoacart](https://github.com/oxtoacart))
- Juju ratelimit [\#217](https://github.com/getlantern/http-proxy-lantern/pull/217) ([forkner](https://github.com/forkner))
- \#2016 device based throttling [\#216](https://github.com/getlantern/http-proxy-lantern/pull/216) ([joesis](https://github.com/joesis))
- Enable throttling for Pro proxies [\#215](https://github.com/getlantern/http-proxy-lantern/pull/215) ([joesis](https://github.com/joesis))
- Added ability to probe upstream for BBR bandwidth [\#214](https://github.com/getlantern/http-proxy-lantern/pull/214) ([oxtoacart](https://github.com/oxtoacart))
- Actually fail the proxy op if there's a problem [\#213](https://github.com/getlantern/http-proxy-lantern/pull/213) ([oxtoacart](https://github.com/oxtoacart))
- Added more borda telemetry to figure out what's going on with proxies [\#212](https://github.com/getlantern/http-proxy-lantern/pull/212) ([oxtoacart](https://github.com/oxtoacart))
- Disabled blacklisting [\#211](https://github.com/getlantern/http-proxy-lantern/pull/211) ([oxtoacart](https://github.com/oxtoacart))
- Re-enabling connections to config server [\#210](https://github.com/getlantern/http-proxy-lantern/pull/210) ([oxtoacart](https://github.com/oxtoacart))
- move to dep [\#209](https://github.com/getlantern/http-proxy-lantern/pull/209) ([myleshorton](https://github.com/myleshorton))
- Totally disabling connections to config server for performance [\#208](https://github.com/getlantern/http-proxy-lantern/pull/208) ([oxtoacart](https://github.com/oxtoacart))
- removes config caching [\#207](https://github.com/getlantern/http-proxy-lantern/pull/207) ([myleshorton](https://github.com/myleshorton))
- ENHTTP servers now check auth tokens and handle pings [\#206](https://github.com/getlantern/http-proxy-lantern/pull/206) ([oxtoacart](https://github.com/oxtoacart))
- PR \#203 [\#205](https://github.com/getlantern/http-proxy-lantern/pull/205) ([joesis](https://github.com/joesis))
- Cache the IPs of clients that have successfully received configs [\#203](https://github.com/getlantern/http-proxy-lantern/pull/203) ([myleshorton](https://github.com/myleshorton))
- Don't throttle checkfallbacks [\#200](https://github.com/getlantern/http-proxy-lantern/pull/200) ([oxtoacart](https://github.com/oxtoacart))
- Using latest proxy that makes chunked transfer encoding explicit when… [\#199](https://github.com/getlantern/http-proxy-lantern/pull/199) ([oxtoacart](https://github.com/oxtoacart))
- Wrapping physical rather than virtual connections with idletiming [\#197](https://github.com/getlantern/http-proxy-lantern/pull/197) ([oxtoacart](https://github.com/oxtoacart))
- Domain-fronting using encapsulated HTTP protocol [\#196](https://github.com/getlantern/http-proxy-lantern/pull/196) ([oxtoacart](https://github.com/oxtoacart))
- throttle connections with no device ID [\#195](https://github.com/getlantern/http-proxy-lantern/pull/195) ([myleshorton](https://github.com/myleshorton))
- Tighter kcp integration using kcpwrapper [\#192](https://github.com/getlantern/http-proxy-lantern/pull/192) ([oxtoacart](https://github.com/oxtoacart))
- tests clobber Host and Port here in race condition [\#191](https://github.com/getlantern/http-proxy-lantern/pull/191) ([myleshorton](https://github.com/myleshorton))
- Added ability to tunnel traffic over kcp [\#190](https://github.com/getlantern/http-proxy-lantern/pull/190) ([oxtoacart](https://github.com/oxtoacart))
- Updated to latest lampshade and http-proxy [\#189](https://github.com/getlantern/http-proxy-lantern/pull/189) ([oxtoacart](https://github.com/oxtoacart))
- Periodically forcing GC [\#188](https://github.com/getlantern/http-proxy-lantern/pull/188) ([oxtoacart](https://github.com/oxtoacart))
- Updated to latest http-proxy [\#187](https://github.com/getlantern/http-proxy-lantern/pull/187) ([oxtoacart](https://github.com/oxtoacart))
- Making sure that resp.Header is not nil [\#186](https://github.com/getlantern/http-proxy-lantern/pull/186) ([oxtoacart](https://github.com/oxtoacart))
- Using new filters API [\#185](https://github.com/getlantern/http-proxy-lantern/pull/185) ([oxtoacart](https://github.com/oxtoacart))
- Switched to using shared tlsredis and testredis [\#184](https://github.com/getlantern/http-proxy-lantern/pull/184) ([oxtoacart](https://github.com/oxtoacart))
- fasttrack domains matches subdomains, closes getlantern/lantern-inter… [\#183](https://github.com/getlantern/http-proxy-lantern/pull/183) ([oxtoacart](https://github.com/oxtoacart))
- fix \#181 [\#182](https://github.com/getlantern/http-proxy-lantern/pull/182) ([joesis](https://github.com/joesis))
- Updated to latest lampshade API [\#179](https://github.com/getlantern/http-proxy-lantern/pull/179) ([oxtoacart](https://github.com/oxtoacart))
- Configurable throttling sensitive to device type [\#178](https://github.com/getlantern/http-proxy-lantern/pull/178) ([oxtoacart](https://github.com/oxtoacart))
- Using special DeviceID for getlantern/lantern\#851 [\#174](https://github.com/getlantern/http-proxy-lantern/pull/174) ([oxtoacart](https://github.com/oxtoacart))

## [v0.0.20](https://github.com/getlantern/http-proxy-lantern/tree/v0.0.20) (2016-05-20)

[Full Changelog](https://github.com/getlantern/http-proxy-lantern/compare/v0.0.18...v0.0.20)

**Closed issues:**

- Immediately deactivate reporting if too may errors while contacting Redis [\#62](https://github.com/getlantern/http-proxy-lantern/issues/62)

**Merged pull requests:**

- Do not ping Redis and always return the client if successfully created [\#73](https://github.com/getlantern/http-proxy-lantern/pull/73) ([uaalto](https://github.com/uaalto))
- Using rediss URL to identify TLS connections to Redis [\#72](https://github.com/getlantern/http-proxy-lantern/pull/72) ([oxtoacart](https://github.com/oxtoacart))
- Now connecting to redis with TLS [\#70](https://github.com/getlantern/http-proxy-lantern/pull/70) ([oxtoacart](https://github.com/oxtoacart))
- Updated test to work with patched Go 1.6.2 [\#66](https://github.com/getlantern/http-proxy-lantern/pull/66) ([oxtoacart](https://github.com/oxtoacart))
- Smooth reporting [\#63](https://github.com/getlantern/http-proxy-lantern/pull/63) ([uaalto](https://github.com/uaalto))

## [v0.0.18](https://github.com/getlantern/http-proxy-lantern/tree/v0.0.18) (2016-05-03)

[Full Changelog](https://github.com/getlantern/http-proxy-lantern/compare/v0.0.19...v0.0.18)

## [v0.0.19](https://github.com/getlantern/http-proxy-lantern/tree/v0.0.19) (2016-05-03)

[Full Changelog](https://github.com/getlantern/http-proxy-lantern/compare/v0.0.17...v0.0.19)

**Closed issues:**

- go test -race discovers some race conditions [\#64](https://github.com/getlantern/http-proxy-lantern/issues/64)
- Server being DOS'ed [\#55](https://github.com/getlantern/http-proxy-lantern/issues/55)
- Try to deploy HAProxy in front of http-proxy-lantern [\#39](https://github.com/getlantern/http-proxy-lantern/issues/39)
- Analyze our SSL profile with SSL Server Test [\#26](https://github.com/getlantern/http-proxy-lantern/issues/26)
- mimic a minimum configured apache over https [\#7](https://github.com/getlantern/http-proxy-lantern/issues/7)

**Merged pull requests:**

- Fix data races [\#65](https://github.com/getlantern/http-proxy-lantern/pull/65) ([uaalto](https://github.com/uaalto))

## [v0.0.17](https://github.com/getlantern/http-proxy-lantern/tree/v0.0.17) (2016-04-26)

[Full Changelog](https://github.com/getlantern/http-proxy-lantern/compare/reporting-test-auth...v0.0.17)

**Closed issues:**

- Pro purchases on mobile: reliability of browser redirects [\#61](https://github.com/getlantern/http-proxy-lantern/issues/61)
- Use Redis passwords [\#52](https://github.com/getlantern/http-proxy-lantern/issues/52)

**Merged pull requests:**

- Added IP-based blacklisting of clients that consistently fail to prov… [\#58](https://github.com/getlantern/http-proxy-lantern/pull/58) ([oxtoacart](https://github.com/oxtoacart))

## [reporting-test-auth](https://github.com/getlantern/http-proxy-lantern/tree/reporting-test-auth) (2016-04-25)

[Full Changelog](https://github.com/getlantern/http-proxy-lantern/compare/reporting-test...reporting-test-auth)

**Merged pull requests:**

- Added password authentication to Redis client config and flags [\#59](https://github.com/getlantern/http-proxy-lantern/pull/59) ([uaalto](https://github.com/uaalto))
- Correct data transfer measurement [\#56](https://github.com/getlantern/http-proxy-lantern/pull/56) ([uaalto](https://github.com/uaalto))

## [reporting-test](https://github.com/getlantern/http-proxy-lantern/tree/reporting-test) (2016-04-18)

[Full Changelog](https://github.com/getlantern/http-proxy-lantern/compare/v0.0.16...reporting-test)

## [v0.0.16](https://github.com/getlantern/http-proxy-lantern/tree/v0.0.16) (2016-04-18)

[Full Changelog](https://github.com/getlantern/http-proxy-lantern/compare/obfs4test...v0.0.16)

**Fixed bugs:**

- Measurements sometimes use IP:port as key, instead of Device-Id [\#53](https://github.com/getlantern/http-proxy-lantern/issues/53)

**Closed issues:**

- Sent lots of "Unable to set read deadline" error to Loggly [\#45](https://github.com/getlantern/http-proxy-lantern/issues/45)
- connect: network is unreachable [\#25](https://github.com/getlantern/http-proxy-lantern/issues/25)
- GA related errors logged in loggly [\#24](https://github.com/getlantern/http-proxy-lantern/issues/24)

**Merged pull requests:**

- Do not report connections without proper Device ID \(closes \#53\) [\#54](https://github.com/getlantern/http-proxy-lantern/pull/54) ([uaalto](https://github.com/uaalto))
- Reduce the number of keys generated in Redis [\#51](https://github.com/getlantern/http-proxy-lantern/pull/51) ([uaalto](https://github.com/uaalto))
- Add obfs4 support to proxy [\#46](https://github.com/getlantern/http-proxy-lantern/pull/46) ([oxtoacart](https://github.com/oxtoacart))

## [obfs4test](https://github.com/getlantern/http-proxy-lantern/tree/obfs4test) (2016-03-28)

[Full Changelog](https://github.com/getlantern/http-proxy-lantern/compare/v0.0.13...obfs4test)

## [v0.0.13](https://github.com/getlantern/http-proxy-lantern/tree/v0.0.13) (2016-03-10)

[Full Changelog](https://github.com/getlantern/http-proxy-lantern/compare/v0.0.14...v0.0.13)

## [v0.0.14](https://github.com/getlantern/http-proxy-lantern/tree/v0.0.14) (2016-03-10)

[Full Changelog](https://github.com/getlantern/http-proxy-lantern/compare/v0.0.15...v0.0.14)

## [v0.0.15](https://github.com/getlantern/http-proxy-lantern/tree/v0.0.15) (2016-03-10)

[Full Changelog](https://github.com/getlantern/http-proxy-lantern/compare/v0.0.12...v0.0.15)

**Closed issues:**

- Remove port from X-Lantern-Config-Client-IP header [\#42](https://github.com/getlantern/http-proxy-lantern/issues/42)

**Merged pull requests:**

- split host from RemoteAddr fixes \#42 [\#43](https://github.com/getlantern/http-proxy-lantern/pull/43) ([fffw](https://github.com/fffw))

## [v0.0.12](https://github.com/getlantern/http-proxy-lantern/tree/v0.0.12) (2016-03-09)

[Full Changelog](https://github.com/getlantern/http-proxy-lantern/compare/v0.0.11...v0.0.12)

**Merged pull requests:**

- Checking for empty domain closes getlantern/lantern\#3753 [\#41](https://github.com/getlantern/http-proxy-lantern/pull/41) ([oxtoacart](https://github.com/oxtoacart))

## [v0.0.11](https://github.com/getlantern/http-proxy-lantern/tree/v0.0.11) (2016-03-07)

[Full Changelog](https://github.com/getlantern/http-proxy-lantern/compare/v0.0.10...v0.0.11)

**Closed issues:**

- Limit CONNECT requests to well known http\(s\) ports of target site [\#35](https://github.com/getlantern/http-proxy-lantern/issues/35)
- Device registry for pro users [\#21](https://github.com/getlantern/http-proxy-lantern/issues/21)

**Merged pull requests:**

- Tracking analytics per server closes getlantern/lantern\#3581 [\#40](https://github.com/getlantern/http-proxy-lantern/pull/40) ([oxtoacart](https://github.com/oxtoacart))
- Loader [\#38](https://github.com/getlantern/http-proxy-lantern/pull/38) ([oxtoacart](https://github.com/oxtoacart))

## [v0.0.10](https://github.com/getlantern/http-proxy-lantern/tree/v0.0.10) (2016-03-03)

[Full Changelog](https://github.com/getlantern/http-proxy-lantern/compare/v0.0.9...v0.0.10)

**Closed issues:**

- Attach auth token to requests fetch cloud config [\#33](https://github.com/getlantern/http-proxy-lantern/issues/33)

**Merged pull requests:**

- Add option to allow certain tunnel ports resolves \#35 [\#36](https://github.com/getlantern/http-proxy-lantern/pull/36) ([fffw](https://github.com/fffw))
- attach X-Lantern-Config-Auth-Token to requests to config-server [\#34](https://github.com/getlantern/http-proxy-lantern/pull/34) ([fffw](https://github.com/fffw))
- Pro support [\#32](https://github.com/getlantern/http-proxy-lantern/pull/32) ([uaalto](https://github.com/uaalto))

## [v0.0.9](https://github.com/getlantern/http-proxy-lantern/tree/v0.0.9) (2016-02-12)

[Full Changelog](https://github.com/getlantern/http-proxy-lantern/compare/v0.0.8...v0.0.9)

**Merged pull requests:**

- Add Wercker CI [\#31](https://github.com/getlantern/http-proxy-lantern/pull/31) ([uaalto](https://github.com/uaalto))
- Added ping capability to proxy [\#30](https://github.com/getlantern/http-proxy-lantern/pull/30) ([oxtoacart](https://github.com/oxtoacart))

## [v0.0.8](https://github.com/getlantern/http-proxy-lantern/tree/v0.0.8) (2016-02-04)

[Full Changelog](https://github.com/getlantern/http-proxy-lantern/compare/0.0.7...v0.0.8)

**Merged pull requests:**

- Moved mimic to right place in tokenfilter.go [\#29](https://github.com/getlantern/http-proxy-lantern/pull/29) ([oxtoacart](https://github.com/oxtoacart))
- Adding user agent to Google Analytics [\#27](https://github.com/getlantern/http-proxy-lantern/pull/27) ([oxtoacart](https://github.com/oxtoacart))

## [0.0.7](https://github.com/getlantern/http-proxy-lantern/tree/0.0.7) (2016-01-29)

[Full Changelog](https://github.com/getlantern/http-proxy-lantern/compare/v0.0.7...0.0.7)

## [v0.0.7](https://github.com/getlantern/http-proxy-lantern/tree/v0.0.7) (2016-01-29)

[Full Changelog](https://github.com/getlantern/http-proxy-lantern/compare/v0.0.6...v0.0.7)

**Closed issues:**

- Use config file [\#12](https://github.com/getlantern/http-proxy-lantern/issues/12)

**Merged pull requests:**

- Supporting multiple auth token headers [\#28](https://github.com/getlantern/http-proxy-lantern/pull/28) ([oxtoacart](https://github.com/oxtoacart))
- getlantern/lantern\#3451 Doing reverse lookups of ip addresses and onl… [\#23](https://github.com/getlantern/http-proxy-lantern/pull/23) ([oxtoacart](https://github.com/oxtoacart))
- serve command line options from config file [\#15](https://github.com/getlantern/http-proxy-lantern/pull/15) ([fffw](https://github.com/fffw))

## [v0.0.6](https://github.com/getlantern/http-proxy-lantern/tree/v0.0.6) (2016-01-26)

[Full Changelog](https://github.com/getlantern/http-proxy-lantern/compare/v0.0.4...v0.0.6)

**Merged pull requests:**

- getlantern/lantern\#3409 Added tracking of popular sites statistic via… [\#22](https://github.com/getlantern/http-proxy-lantern/pull/22) ([oxtoacart](https://github.com/oxtoacart))

## [v0.0.4](https://github.com/getlantern/http-proxy-lantern/tree/v0.0.4) (2015-11-24)

[Full Changelog](https://github.com/getlantern/http-proxy-lantern/compare/v0.0.5-go.1.5.3...v0.0.4)

## [v0.0.5-go.1.5.3](https://github.com/getlantern/http-proxy-lantern/tree/v0.0.5-go.1.5.3) (2015-11-24)

[Full Changelog](https://github.com/getlantern/http-proxy-lantern/compare/v0.0.5...v0.0.5-go.1.5.3)

## [v0.0.5](https://github.com/getlantern/http-proxy-lantern/tree/v0.0.5) (2015-11-24)

[Full Changelog](https://github.com/getlantern/http-proxy-lantern/compare/v0.0.3...v0.0.5)

**Closed issues:**

- print out the token in proxy request when mismatch [\#17](https://github.com/getlantern/http-proxy-lantern/issues/17)

**Merged pull requests:**

- print out token in log if mismatch [\#19](https://github.com/getlantern/http-proxy-lantern/pull/19) ([fffw](https://github.com/fffw))

## [v0.0.3](https://github.com/getlantern/http-proxy-lantern/tree/v0.0.3) (2015-11-17)

[Full Changelog](https://github.com/getlantern/http-proxy-lantern/compare/dd0f429f20ea84d689f9bf50068ce6674279d6be...v0.0.3)

**Closed issues:**

- Memory leak in preprocessor [\#13](https://github.com/getlantern/http-proxy-lantern/issues/13)
- Log errors handling incoming requests [\#11](https://github.com/getlantern/http-proxy-lantern/issues/11)
- Add http pprof option [\#9](https://github.com/getlantern/http-proxy-lantern/issues/9)
- Vulnerability: Several requests with Host: localhost:\<proxy-port\> will segfault the server [\#6](https://github.com/getlantern/http-proxy-lantern/issues/6)
- Not mimic Apache when "unexpected EOF" got [\#4](https://github.com/getlantern/http-proxy-lantern/issues/4)

**Merged pull requests:**

- close request body closes \#13 [\#14](https://github.com/getlantern/http-proxy-lantern/pull/14) ([fffw](https://github.com/fffw))
- add pprofAddr option [\#10](https://github.com/getlantern/http-proxy-lantern/pull/10) ([fffw](https://github.com/fffw))
- add token filter to test mimicking apache [\#8](https://github.com/getlantern/http-proxy-lantern/pull/8) ([fffw](https://github.com/fffw))
- ignore unexpected EOF error [\#5](https://github.com/getlantern/http-proxy-lantern/pull/5) ([fffw](https://github.com/fffw))
- mimic apache when request has multiple Content-Length header [\#3](https://github.com/getlantern/http-proxy-lantern/pull/3) ([fffw](https://github.com/fffw))
- Updates from proxy [\#2](https://github.com/getlantern/http-proxy-lantern/pull/2) ([uaalto](https://github.com/uaalto))
- Updates from proxy [\#1](https://github.com/getlantern/http-proxy-lantern/pull/1) ([uaalto](https://github.com/uaalto))



\* *This Changelog was automatically generated by [github_changelog_generator](https://github.com/github-changelog-generator/github-changelog-generator)*
