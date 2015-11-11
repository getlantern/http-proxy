# HTTP/S Proxy with extensions for Lantern

These are Lantern-specific middleware components for the HTTP Proxy in Go:

* A filter for access tokens

* A filter for devices, based on their IDs

* A filter for Pro users


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

*Keep in mind that cURL doesn't support tunneling through an HTTPS proxy, so if you use the -https option you have to use other tools for testing.

Run the server with:

```
go run http_proxy.go -https -token=<your-token>
```

You have two options to test it: the Lantern client or [checkfallbacks](https://github.com/getlantern/lantern/tree/valencia/src/github.com/getlantern/checkfallbacks).

Keep in mind that they will need to send some headers in order to avoid receiving 404 messages (the chained server response if you aren't providing them).

Currently, the only header you need to add is `X-Lantern-Device-Id`.

If you are using checkfallbacks, make sure that both the certificate and the token are correct.  A 404 will be the reply otherwise.  Running the server with `-debug` may help you troubleshooting those scenarios.

