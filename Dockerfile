# syntax = docker/dockerfile:1.3
FROM --platform=linux/amd64 golang:alpine as builder

RUN apk add build-base gcc libpcap-dev git

WORKDIR $GOPATH/src/getlantern/http-proxy-lantern/
COPY . .

RUN --mount=type=secret,id=github_oauth_token \
    git config --global url."https://$(cat /run/secrets/github_oauth_token):x-oauth-basic@github.com/".insteadOf "https://github.com/"

RUN --mount=type=cache,mode=0755,target=/root/.cache/go-build \
    GOARCH=amd64 GOOS=linux CGO_ENABLED=1 go build -v -o /usr/local/bin/http-proxy ./http-proxy

FROM alpine as user
RUN adduser -S -u 10000 lantern

FROM --platform=linux/amd64 alpine
RUN apk add libpcap libgcc libstdc++
COPY --from=user /etc/passwd /etc/passwd
COPY --from=builder /usr/local/bin/http-proxy /usr/local/bin/http-proxy

USER lantern
CMD ["/usr/local/bin/http-proxy"]
