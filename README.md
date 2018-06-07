# HTTP/S Proxy with extensions for Lantern

[![wercker status](https://app.wercker.com/status/67d1598d6205dce4aee80a9631d109e5/m/master "wercker status")](https://app.wercker.com/project/byKey/67d1598d6205dce4aee80a9631d109e5)

Note - this project versions its dependencies using [dep](https://github.com/golang/dep).

Just run `dep ensure` to download the vendored packages.

These are Lantern-specific middleware components for the HTTP Proxy in Go:

* A filter for access tokens

* A filter for devices, based on their IDs

* A filter for Pro users

* A connection preprocessor to intercept bad requests and send custom responses

* Custom responses for mimicking Apache in certain cases


### Usage

Build it with `dep ensure && go build` or with `make build`.

To get list of the command line options, please run `http-proxy-lantern -help`.

`config.ini.default` also has the list of options, make a copy (say, `config.ini`) and tweak it as you wish, then run the proxy with

```
http-proxy-lantern -config config.ini
```

To regenerate `config.ini.default` just run `http-proxy-lantern -dumpflags`.

### Testing with Lantern extensions and configuration

### Run tests

```
go test
```

Use this for verbose output:

```
TRACE=1 go test
```

### Manual testing

*Keep in mind that cURL doesn't support tunneling through an HTTPS proxy, so if you use the -https option you have to use other tools for testing.*

Run the server with:

```
cd http-proxy
go install && http-proxy -https -token <your-token> -enablereports -stackdriver-project-id http-proxy-lantern -stackdriver-creds /Users/afisk/lantern_aws/salt/http_proxy/lantern-stackdriver.json
```

Run a Lantern client accordingly, as in:

```
./lantern -force-proxy-addr localhost:8080 -force-auth-token <your-token>
```

You have two options to test it: the Lantern client or [checkfallbacks](https://github.com/getlantern/lantern/tree/valencia/src/github.com/getlantern/checkfallbacks).

Keep in mind that they will need to send some headers in order to avoid receiving 404 messages (the chained server response if you aren't providing them).

Currently, the only header you need to add is `X-Lantern-Device-Id`.

If you are using checkfallbacks, make sure that both the certificate and the token are correct.  A 404 will be the reply otherwise.  Running the server with `-debug` may help you troubleshooting those scenarios.

### Handle requests config server specially

[To prevent spoofers from fetching Lantern config with fake client IP](https://github.com/getlantern/config-server/issues/4), we need to attach auth tokens to such requests.  Both below options should be supplied. Once `http-proxy-lantern` receives GET request to one of the `cfgsvrdomains`, it sets `X-Lantern-Config-Auth-Token` header with supplied `cfgsvrauthtoken`, and `X-Lantern-Config-Client-IP` header with the IP address it sees.

```
  -cfgsvrauthtoken string
        Token attached to config-server requests, not attaching if empty
  -cfgsvrdomains string
        Config-server domains on which to attach auth token, separated by comma
```

### When something bad happens

With option `-pprofAddr=localhost:6060`, you can always access lots of debug information from http://localhost:6060/debug/pprof. Ref https://golang.org/pkg/net/http/pprof/.

***Be sure to only listen on localhost or private addresses for security reason.***

## Building for distribution and deploying

When building for distribution make sure you're creating a linux/amd64 binary
and that the resulting binary is compressed with
[upx](http://upx.sourceforge.net/).

You can use the following command to do all this automatically:

```
make dist
```
