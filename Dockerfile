# This docker machine is able to compile http-proxy-lantern for Ubuntu Linux

FROM alpine:3.9
MAINTAINER "The Lantern Team" <team@getlantern.org>

# Requisites for building Go.
RUN apk add --update git tar gzip curl bash

# Compilers and tools for CGO.
RUN apk add --update alpine-sdk

# Getting Go.
ENV GOROOT /usr/local/go
ENV GOPATH /

ENV PATH $PATH:$GOROOT/bin

ARG go_version
ENV GO_VERSION $go_version
ENV GO_PACKAGE_URL https://storage.googleapis.com/golang/$GO_VERSION.linux-amd64.tar.gz
RUN curl -sSL $GO_PACKAGE_URL | tar -xzf - -C /usr/local
RUN ls -l /usr/local/go/bin/

ENV WORKDIR /src

# Expect the $WORKDIR volume to be mounted.
VOLUME [ "$WORKDIR" ]

WORKDIR $WORKDIR
