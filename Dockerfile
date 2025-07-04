# syntax = docker/dockerfile:1.3
FROM --platform=$BUILDPLATFORM golang:1.24-alpine as builder
ARG TARGETOS TARGETARCH

WORKDIR $GOPATH/src/getlantern/http-proxy-lantern/
COPY . .

RUN apk add git
RUN --mount=type=secret,id=github_oauth_token \
    git config --global url."https://$(cat /run/secrets/github_oauth_token):x-oauth-basic@github.com/".insteadOf "https://github.com/"

RUN GOARCH=$TARGETARCH GOOS=$TARGETOS CGO_ENABLED=0 go build -v \
    -o /usr/local/bin/http-proxy ./http-proxy

FROM --platform=$BUILDPLATFORM alpine as user
RUN adduser -S -u 10000 lantern

FROM alpine
RUN apk add --no-cache iptables

COPY --from=user /etc/passwd /etc/passwd
COPY --from=builder /usr/local/bin/http-proxy /usr/local/bin/http-proxy

USER lantern
CMD ["/usr/local/bin/http-proxy"]
