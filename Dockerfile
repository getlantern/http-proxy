# This docker machine is able to compile http-proxy-lantern for Linux

FROM fedora:21
MAINTAINER "The Lantern Team" <team@getlantern.org>

# Updating system.
RUN yum install -y deltarpm && yum update -y && yum clean packages

# Requisites for building Go.
# Touching /var/lib/rpm/* to work around an issue building container. Same below.
# See https://github.com/moby/moby/issues/10180#issuecomment-378005800.
RUN touch /var/lib/rpm/* && yum install -y git tar gzip curl hostname && yum clean packages

# Compilers and tools for CGO.
RUN touch /var/lib/rpm/* && yum install -y gcc gcc-c++ libgcc.i686 gcc-c++.i686 pkg-config make libpcap-devel && yum clean packages

# Getting Go.
ENV GOROOT /usr/local/go
ENV GOPATH /

ENV PATH $PATH:$GOROOT/bin

ARG go_version
ENV GO_VERSION $go_version
ENV GO_PACKAGE_URL https://storage.googleapis.com/golang/$GO_VERSION.linux-amd64.tar.gz
RUN curl -sSL $GO_PACKAGE_URL | tar -xvzf - -C /usr/local

ENV WORKDIR /src

# Expect the $WORKDIR volume to be mounted.
VOLUME [ "$WORKDIR" ]

WORKDIR $WORKDIR
