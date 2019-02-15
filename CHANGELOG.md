# Change Log

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

## [0.2.1](https://github.com/getlantern/http-proxy-lantern/tree/0.2.1) (2018-12-04)
[Full Changelog](https://github.com/getlantern/http-proxy-lantern/compare/0.2.0...0.2.1)

**Merged pull requests:**

- add x-forwarded-for to pro server [\#272](https://github.com/getlantern/http-proxy-lantern/pull/272) ([myleshorton](https://github.com/myleshorton))
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
[Full Changelog](https://github.com/getlantern/http-proxy-lantern/compare/0.1.12...0.1.13)

## [0.1.12](https://github.com/getlantern/http-proxy-lantern/tree/0.1.12) (2018-09-30)
[Full Changelog](https://github.com/getlantern/http-proxy-lantern/compare/0.1.11...0.1.12)

**Closed issues:**

- `preferIPV4Dialer` to respond a net.Error for time out [\#121](https://github.com/getlantern/http-proxy-lantern/issues/121)

**Merged pull requests:**

- Upgraded to go 1.11 [\#265](https://github.com/getlantern/http-proxy-lantern/pull/265) ([oxtoacart](https://github.com/oxtoacart))
- Updated dependencies [\#264](https://github.com/getlantern/http-proxy-lantern/pull/264) ([oxtoacart](https://github.com/oxtoacart))
- Fix a few bugs to show the upgrade notice [\#262](https://github.com/getlantern/http-proxy-lantern/pull/262) ([joesis](https://github.com/joesis))
- read full request before sending 302 response [\#261](https://github.com/getlantern/http-proxy-lantern/pull/261) ([joesis](https://github.com/joesis))
- Revert "Changed header name to always set True-Client-IP for config server" [\#260](https://github.com/getlantern/http-proxy-lantern/pull/260) ([joesis](https://github.com/joesis))
- Changed header name to always set True-Client-IP for config server [\#259](https://github.com/getlantern/http-proxy-lantern/pull/259) ([myleshorton](https://github.com/myleshorton))
- wip quic support [\#258](https://github.com/getlantern/http-proxy-lantern/pull/258) ([forkner](https://github.com/forkner))
- Suggested changes to \#256 [\#257](https://github.com/getlantern/http-proxy-lantern/pull/257) ([joesis](https://github.com/joesis))
- Allow turning the data cap off by setting the threshold [\#256](https://github.com/getlantern/http-proxy-lantern/pull/256) ([myleshorton](https://github.com/myleshorton))
- don't fail if unable to load throttle config in the first time [\#246](https://github.com/getlantern/http-proxy-lantern/pull/246) ([joesis](https://github.com/joesis))

## [0.1.11](https://github.com/getlantern/http-proxy-lantern/tree/0.1.11) (2018-07-13)
[Full Changelog](https://github.com/getlantern/http-proxy-lantern/compare/0.1.10...0.1.11)

**Merged pull requests:**

- Log when we close HTTP connections due to errors [\#254](https://github.com/getlantern/http-proxy-lantern/pull/254) ([myleshorton](https://github.com/myleshorton))

## [0.1.10](https://github.com/getlantern/http-proxy-lantern/tree/0.1.10) (2018-07-13)
[Full Changelog](https://github.com/getlantern/http-proxy-lantern/compare/0.1.9...0.1.10)

**Merged pull requests:**

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
- add flag to show binary version [\#180](https://github.com/getlantern/http-proxy-lantern/pull/180) ([joesis](https://github.com/joesis))
- Updated to latest lampshade API [\#179](https://github.com/getlantern/http-proxy-lantern/pull/179) ([oxtoacart](https://github.com/oxtoacart))
- Configurable throttling sensitive to device type [\#178](https://github.com/getlantern/http-proxy-lantern/pull/178) ([oxtoacart](https://github.com/oxtoacart))
- just added comments and another small test [\#177](https://github.com/getlantern/http-proxy-lantern/pull/177) ([myleshorton](https://github.com/myleshorton))
- Capturing google search and capture traffic [\#176](https://github.com/getlantern/http-proxy-lantern/pull/176) ([oxtoacart](https://github.com/oxtoacart))
- Reporting tcp and bbr info to borda [\#175](https://github.com/getlantern/http-proxy-lantern/pull/175) ([oxtoacart](https://github.com/oxtoacart))
- Using special DeviceID for getlantern/lantern\#851 [\#174](https://github.com/getlantern/http-proxy-lantern/pull/174) ([oxtoacart](https://github.com/oxtoacart))
- Using special DeviceID for getlantern/lantern\#851 [\#173](https://github.com/getlantern/http-proxy-lantern/pull/173) ([oxtoacart](https://github.com/oxtoacart))
- slightly randomize reporting period [\#172](https://github.com/getlantern/http-proxy-lantern/pull/172) ([joesis](https://github.com/joesis))
- sleep randomly up to 1 min before start reporting [\#170](https://github.com/getlantern/http-proxy-lantern/pull/170) ([joesis](https://github.com/joesis))
- Pro status is now determined via flag [\#169](https://github.com/getlantern/http-proxy-lantern/pull/169) ([oxtoacart](https://github.com/oxtoacart))
- Use separate options for bandwidth reporting [\#166](https://github.com/getlantern/http-proxy-lantern/pull/166) ([joesis](https://github.com/joesis))
- Improved BBR bandwidth estimation [\#165](https://github.com/getlantern/http-proxy-lantern/pull/165) ([oxtoacart](https://github.com/oxtoacart))
- Redirect to an URL if Lantern client version is below configured version [\#164](https://github.com/getlantern/http-proxy-lantern/pull/164) ([joesis](https://github.com/joesis))
- Added ability to do arbitrarily sized pings using just a URL \(for pro… [\#163](https://github.com/getlantern/http-proxy-lantern/pull/163) ([oxtoacart](https://github.com/oxtoacart))
- dial TLS for config-server requests [\#162](https://github.com/getlantern/http-proxy-lantern/pull/162) ([joesis](https://github.com/joesis))
- Integrated lampshade and bbr statistics [\#161](https://github.com/getlantern/http-proxy-lantern/pull/161) ([oxtoacart](https://github.com/oxtoacart))
- use up-to-date connmux package so it compiles [\#160](https://github.com/getlantern/http-proxy-lantern/pull/160) ([joesis](https://github.com/joesis))
- still allow proxy to start if redis is down [\#158](https://github.com/getlantern/http-proxy-lantern/pull/158) ([myleshorton](https://github.com/myleshorton))
- Added ability to set DiffServ TOS [\#157](https://github.com/getlantern/http-proxy-lantern/pull/157) ([oxtoacart](https://github.com/oxtoacart))
- Enabled connmux for OBFS4 [\#156](https://github.com/getlantern/http-proxy-lantern/pull/156) ([oxtoacart](https://github.com/oxtoacart))
- Submitting to borda using gRPC when possible [\#153](https://github.com/getlantern/http-proxy-lantern/pull/153) ([oxtoacart](https://github.com/oxtoacart))
- Using new measured API and reporting traffic stats to borda [\#151](https://github.com/getlantern/http-proxy-lantern/pull/151) ([oxtoacart](https://github.com/oxtoacart))
- Setting proxy\_host based on configured external IP [\#150](https://github.com/getlantern/http-proxy-lantern/pull/150) ([oxtoacart](https://github.com/oxtoacart))
- close connections with no device ID [\#148](https://github.com/getlantern/http-proxy-lantern/pull/148) ([myleshorton](https://github.com/myleshorton))
- When redis is not configured, disable services that rely on redis [\#146](https://github.com/getlantern/http-proxy-lantern/pull/146) ([oxtoacart](https://github.com/oxtoacart))
- do not throttle certain domains or count them towards the cap. [\#143](https://github.com/getlantern/http-proxy-lantern/pull/143) ([myleshorton](https://github.com/myleshorton))
- just logging successful handshake times [\#142](https://github.com/getlantern/http-proxy-lantern/pull/142) ([myleshorton](https://github.com/myleshorton))
- Defaulting port to 80 for non-CONNECT requests that are missing port [\#140](https://github.com/getlantern/http-proxy-lantern/pull/140) ([oxtoacart](https://github.com/oxtoacart))
- Tracking protocol for site accesses in GA [\#139](https://github.com/getlantern/http-proxy-lantern/pull/139) ([oxtoacart](https://github.com/oxtoacart))
- \[WIP\] Added some tests to debug leaks, improved obfs4listener closing sequence [\#138](https://github.com/getlantern/http-proxy-lantern/pull/138) ([oxtoacart](https://github.com/oxtoacart))
- Keying OBFS4 handshakes to remote host instead of whole address, temp… [\#137](https://github.com/getlantern/http-proxy-lantern/pull/137) ([oxtoacart](https://github.com/oxtoacart))
- Limiting OBFS4 handshakes by client [\#136](https://github.com/getlantern/http-proxy-lantern/pull/136) ([oxtoacart](https://github.com/oxtoacart))
- Cleaned up logging of OBFS4 handshaking, reduced OBFS4 handshake timeout [\#135](https://github.com/getlantern/http-proxy-lantern/pull/135) ([oxtoacart](https://github.com/oxtoacart))
- Fixing comparison of whitelisted config header domains when req.Host … [\#134](https://github.com/getlantern/http-proxy-lantern/pull/134) ([oxtoacart](https://github.com/oxtoacart))
- Added unit test for kcp connection leaks [\#133](https://github.com/getlantern/http-proxy-lantern/pull/133) ([oxtoacart](https://github.com/oxtoacart))
- Updated KCP to use OBFS4 [\#132](https://github.com/getlantern/http-proxy-lantern/pull/132) ([oxtoacart](https://github.com/oxtoacart))
- Added support for stateful HTTP forwarding [\#129](https://github.com/getlantern/http-proxy-lantern/pull/129) ([oxtoacart](https://github.com/oxtoacart))
- Added ability to ping arbitrary urls for timings [\#128](https://github.com/getlantern/http-proxy-lantern/pull/128) ([oxtoacart](https://github.com/oxtoacart))
- Fix incorrect logging [\#127](https://github.com/getlantern/http-proxy-lantern/pull/127) ([uaalto](https://github.com/uaalto))
- Upgraded to go 1.7 [\#125](https://github.com/getlantern/http-proxy-lantern/pull/125) ([oxtoacart](https://github.com/oxtoacart))
- Added support for KCP protocol [\#124](https://github.com/getlantern/http-proxy-lantern/pull/124) ([oxtoacart](https://github.com/oxtoacart))
- Added support for benchmarking mode \(for testing datacenters\) [\#123](https://github.com/getlantern/http-proxy-lantern/pull/123) ([oxtoacart](https://github.com/oxtoacart))
- Removed apache mimic preprocessor [\#120](https://github.com/getlantern/http-proxy-lantern/pull/120) ([oxtoacart](https://github.com/oxtoacart))
- Add Makefile and targets [\#119](https://github.com/getlantern/http-proxy-lantern/pull/119) ([xiam](https://github.com/xiam))
- Set the default PoolSize to 3 [\#117](https://github.com/getlantern/http-proxy-lantern/pull/117) ([xiam](https://github.com/xiam))
- use updated obfs4 fix getlantern/lantern\_aws\#224 [\#116](https://github.com/getlantern/http-proxy-lantern/pull/116) ([fffw](https://github.com/fffw))
- Pro-only proxies and temporary fix for clients with no Pro token [\#114](https://github.com/getlantern/http-proxy-lantern/pull/114) ([uaalto](https://github.com/uaalto))
- prefer ipv4 when dialing origin site fix \#97 [\#112](https://github.com/getlantern/http-proxy-lantern/pull/112) ([fffw](https://github.com/fffw))
- not remove pro token for config server requests fix \#44 [\#111](https://github.com/getlantern/http-proxy-lantern/pull/111) ([fffw](https://github.com/fffw))
- Default server id to hostname, and use glide for vendoring [\#109](https://github.com/getlantern/http-proxy-lantern/pull/109) ([oxtoacart](https://github.com/oxtoacart))
- Eager devices [\#108](https://github.com/getlantern/http-proxy-lantern/pull/108) ([uaalto](https://github.com/uaalto))
- Enable Pro even if there are no users assigned, so we don't need to turn [\#105](https://github.com/getlantern/http-proxy-lantern/pull/105) ([uaalto](https://github.com/uaalto))
- Small improvements done during end-to-end tests [\#104](https://github.com/getlantern/http-proxy-lantern/pull/104) ([uaalto](https://github.com/uaalto))
- Do not throttle Pro users [\#98](https://github.com/getlantern/http-proxy-lantern/pull/98) ([uaalto](https://github.com/uaalto))
- Simply logging handshake failures rather than exposing them through A… [\#95](https://github.com/getlantern/http-proxy-lantern/pull/95) ([oxtoacart](https://github.com/oxtoacart))
- Moved obfs4 handshake to goroutine [\#94](https://github.com/getlantern/http-proxy-lantern/pull/94) ([oxtoacart](https://github.com/oxtoacart))
- only blacklist rapid connect attempts [\#89](https://github.com/getlantern/http-proxy-lantern/pull/89) ([fffw](https://github.com/fffw))
- Remove pipelining [\#87](https://github.com/getlantern/http-proxy-lantern/pull/87) ([uaalto](https://github.com/uaalto))
- Bandwidth usage headers integrating throttling [\#86](https://github.com/getlantern/http-proxy-lantern/pull/86) ([uaalto](https://github.com/uaalto))
- Added flowrate benchmark programs [\#85](https://github.com/getlantern/http-proxy-lantern/pull/85) ([oxtoacart](https://github.com/oxtoacart))
- Factored proxy code into library, switched to stretchr/testify [\#84](https://github.com/getlantern/http-proxy-lantern/pull/84) ([oxtoacart](https://github.com/oxtoacart))
- Using new filter chaining API [\#82](https://github.com/getlantern/http-proxy-lantern/pull/82) ([oxtoacart](https://github.com/oxtoacart))
- Adapted to new measured API [\#81](https://github.com/getlantern/http-proxy-lantern/pull/81) ([oxtoacart](https://github.com/oxtoacart))
- Throttling after 500mb [\#80](https://github.com/getlantern/http-proxy-lantern/pull/80) ([uaalto](https://github.com/uaalto))
- Added support for borda reporting [\#77](https://github.com/getlantern/http-proxy-lantern/pull/77) ([oxtoacart](https://github.com/oxtoacart))
- report blacklisted ip as errors [\#75](https://github.com/getlantern/http-proxy-lantern/pull/75) ([fffw](https://github.com/fffw))
- Expiring bandwidth counters at end of month [\#74](https://github.com/getlantern/http-proxy-lantern/pull/74) ([oxtoacart](https://github.com/oxtoacart))

## [v0.0.20](https://github.com/getlantern/http-proxy-lantern/tree/v0.0.20) (2016-05-20)
[Full Changelog](https://github.com/getlantern/http-proxy-lantern/compare/v0.0.19...v0.0.20)

**Closed issues:**

- Immediately deactivate reporting if too may errors while contacting Redis [\#62](https://github.com/getlantern/http-proxy-lantern/issues/62)

**Merged pull requests:**

- Do not ping Redis and always return the client if successfully created [\#73](https://github.com/getlantern/http-proxy-lantern/pull/73) ([uaalto](https://github.com/uaalto))
- Using rediss URL to identify TLS connections to Redis [\#72](https://github.com/getlantern/http-proxy-lantern/pull/72) ([oxtoacart](https://github.com/oxtoacart))
- Now connecting to redis with TLS [\#70](https://github.com/getlantern/http-proxy-lantern/pull/70) ([oxtoacart](https://github.com/oxtoacart))
- Updated test to work with patched Go 1.6.2 [\#66](https://github.com/getlantern/http-proxy-lantern/pull/66) ([oxtoacart](https://github.com/oxtoacart))
- Smooth reporting [\#63](https://github.com/getlantern/http-proxy-lantern/pull/63) ([uaalto](https://github.com/uaalto))

## [v0.0.19](https://github.com/getlantern/http-proxy-lantern/tree/v0.0.19) (2016-05-03)
[Full Changelog](https://github.com/getlantern/http-proxy-lantern/compare/v0.0.18...v0.0.19)

## [v0.0.18](https://github.com/getlantern/http-proxy-lantern/tree/v0.0.18) (2016-05-03)
[Full Changelog](https://github.com/getlantern/http-proxy-lantern/compare/v0.0.17...v0.0.18)

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
[Full Changelog](https://github.com/getlantern/http-proxy-lantern/compare/v0.0.16...reporting-test-auth)

**Merged pull requests:**

- Added password authentication to Redis client config and flags [\#59](https://github.com/getlantern/http-proxy-lantern/pull/59) ([uaalto](https://github.com/uaalto))
- Correct data transfer measurement [\#56](https://github.com/getlantern/http-proxy-lantern/pull/56) ([uaalto](https://github.com/uaalto))

## [v0.0.16](https://github.com/getlantern/http-proxy-lantern/tree/v0.0.16) (2016-04-18)
[Full Changelog](https://github.com/getlantern/http-proxy-lantern/compare/reporting-test...v0.0.16)

## [reporting-test](https://github.com/getlantern/http-proxy-lantern/tree/reporting-test) (2016-04-18)
[Full Changelog](https://github.com/getlantern/http-proxy-lantern/compare/obfs4test...reporting-test)

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
[Full Changelog](https://github.com/getlantern/http-proxy-lantern/compare/v0.0.7...v0.0.8)

**Merged pull requests:**

- Moved mimic to right place in tokenfilter.go [\#29](https://github.com/getlantern/http-proxy-lantern/pull/29) ([oxtoacart](https://github.com/oxtoacart))
- Supporting multiple auth token headers [\#28](https://github.com/getlantern/http-proxy-lantern/pull/28) ([oxtoacart](https://github.com/oxtoacart))
- Adding user agent to Google Analytics [\#27](https://github.com/getlantern/http-proxy-lantern/pull/27) ([oxtoacart](https://github.com/oxtoacart))

## [v0.0.7](https://github.com/getlantern/http-proxy-lantern/tree/v0.0.7) (2016-01-29)
[Full Changelog](https://github.com/getlantern/http-proxy-lantern/compare/0.0.7...v0.0.7)

## [0.0.7](https://github.com/getlantern/http-proxy-lantern/tree/0.0.7) (2016-01-29)
[Full Changelog](https://github.com/getlantern/http-proxy-lantern/compare/v0.0.6...0.0.7)

**Closed issues:**

- Use config file [\#12](https://github.com/getlantern/http-proxy-lantern/issues/12)

**Merged pull requests:**

- getlantern/lantern\#3451 Doing reverse lookups of ip addresses and onl… [\#23](https://github.com/getlantern/http-proxy-lantern/pull/23) ([oxtoacart](https://github.com/oxtoacart))
- serve command line options from config file [\#15](https://github.com/getlantern/http-proxy-lantern/pull/15) ([fffw](https://github.com/fffw))

## [v0.0.6](https://github.com/getlantern/http-proxy-lantern/tree/v0.0.6) (2016-01-26)
[Full Changelog](https://github.com/getlantern/http-proxy-lantern/compare/v0.0.5...v0.0.6)

**Merged pull requests:**

- getlantern/lantern\#3409 Added tracking of popular sites statistic via… [\#22](https://github.com/getlantern/http-proxy-lantern/pull/22) ([oxtoacart](https://github.com/oxtoacart))

## [v0.0.5](https://github.com/getlantern/http-proxy-lantern/tree/v0.0.5) (2015-11-24)
[Full Changelog](https://github.com/getlantern/http-proxy-lantern/compare/v0.0.5-go.1.5.3...v0.0.5)

## [v0.0.5-go.1.5.3](https://github.com/getlantern/http-proxy-lantern/tree/v0.0.5-go.1.5.3) (2015-11-24)
[Full Changelog](https://github.com/getlantern/http-proxy-lantern/compare/v0.0.4...v0.0.5-go.1.5.3)

## [v0.0.4](https://github.com/getlantern/http-proxy-lantern/tree/v0.0.4) (2015-11-24)
[Full Changelog](https://github.com/getlantern/http-proxy-lantern/compare/v0.0.3...v0.0.4)

**Closed issues:**

- print out the token in proxy request when mismatch [\#17](https://github.com/getlantern/http-proxy-lantern/issues/17)

**Merged pull requests:**

- print out token in log if mismatch [\#19](https://github.com/getlantern/http-proxy-lantern/pull/19) ([fffw](https://github.com/fffw))

## [v0.0.3](https://github.com/getlantern/http-proxy-lantern/tree/v0.0.3) (2015-11-17)
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



\* *This Change Log was automatically generated by [github_changelog_generator](https://github.com/skywinder/Github-Changelog-Generator)*