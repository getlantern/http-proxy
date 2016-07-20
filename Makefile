GLIDE_BIN    ?= $(shell which glide)
UPX_BIN      ?= $(shell which upx)
BUILD_DIR    ?= bin

.PHONY: dist build require-glide

build: require-glide
	$(GLIDE_BIN) install && \
	mkdir -p $(BUILD_DIR) && \
	go build -o $(BUILD_DIR)/http-proxy-lantern github.com/getlantern/http-proxy-lantern/http-proxy && \
	file $(BUILD_DIR)/http-proxy-lantern

require-glide:
	@if [ "$(GLIDE_BIN)" = "" ]; then \
		echo 'Missing "glide" command. See https://github.com/Masterminds/glide' && exit 1; \
	fi

require-upx:
	@if [ "$(UPX_BIN)" = "" ]; then \
		echo 'Missing "upx" command. See http://upx.sourceforge.net/' && exit 1; \
	fi

dist: require-glide require-upx
	GOOS=linux GOARCH=amd64 BUILD_DIR=dist $(MAKE) build && \
	upx dist/http-proxy-lantern

clean:
	rm -rf dist bin
