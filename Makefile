DEP_BIN    ?= $(shell which dep)
UPX_BIN      ?= $(shell which upx)
BUILD_DIR    ?= bin
GIT_REVISION := $(shell git rev-parse --short HEAD)

.PHONY: dist build require-dep

build: require-dep
	$(DEP_BIN) ensure && \
	mkdir -p $(BUILD_DIR) && \
	go build -o $(BUILD_DIR)/http-proxy \
	-ldflags="-X main.revision=$(GIT_REVISION)" \
	github.com/getlantern/http-proxy-lantern/http-proxy && \
	file $(BUILD_DIR)/http-proxy

require-dep:
	@if [ "$(DEP_BIN)" = "" ]; then \
		echo 'Missing "dep" command. See https://github.com/golang/dep or just brew install dep' && exit 1; \
	fi

require-upx:
	@if [ "$(UPX_BIN)" = "" ]; then \
		echo 'Missing "upx" command. See http://upx.sourceforge.net/' && exit 1; \
	fi

dist: require-dep require-upx
	GOOS=linux GOARCH=amd64 BUILD_DIR=dist $(MAKE) build -o http-proxy && \
	upx dist/http-proxy

deploy: dist/http-proxy
	s3cmd put dist/http-proxy s3://http-proxy/http-proxy && \
	s3cmd setacl --acl-grant read:f87080f71ec0be3b9a933cbb244a6c24d4aca584ac32b3220f56d59071043747 s3://http-proxy/http-proxy

clean:
	rm -rf dist bin
