GLIDE_BIN    ?= $(shell which glide)
GOOS         ?= linux
GOARCH       ?= amd64

.PHONY: dist build require-glide

require-glide:
	@if [ "$(GLIDE_BIN)" = "" ]; then \
		echo 'Missing "glide" command. See https://github.com/Masterminds/glide' && exit 1; \
	fi

dist: require-glide
	$(GLIDE_BIN) install && \
	mkdir -p dist && \
	go build -o bin/http-proxy-lantern && \
	file bin/http-proxy-lantern

build:
	go build -o bin/http-proxy-lantern

clean:
	rm -rf dist
