DEP_BIN      ?= $(shell which dep)
UPX_BIN      ?= $(shell which upx)
BUILD_DIR    ?= bin
GIT_REVISION := $(shell git rev-parse --short HEAD)
CHANGE_BIN   := $(shell which github_changelog_generator)

GO_VERSION := 1.10.7
DOCKER_IMAGE_TAG := http-proxy-builder
DOCKER_VOLS = "-v $$PWD/../../..:/src"

get-command = $(shell which="$$(which $(1) 2> /dev/null)" && if [[ ! -z "$$which" ]]; then printf %q "$$which"; fi)

DEP_BIN   := $(call get-command,dep)
DOCKER    := $(call get-command,docker)
GO        := $(call get-command,go)

.PHONY: dist build require-dep

# This tags the current version and creates a CHANGELOG for the current directory.
define tag-changelog
	echo "Tagging..." && \
	git tag -a "$$VERSION" -f --annotate -m"Tagged $$VERSION" && \
	git push --tags -f && \
	$(CHANGE_BIN) --token "5bfda07d0382fff2285de3579caa92b1764d0db9" getlantern/$(1) && \
	git add CHANGELOG.md && \
	git commit -m "Updated changelog for $$VERSION" && \
	git push origin HEAD
endef

guard-%:
	 @ if [ -z '${${*}}' ]; then echo 'Environment variable $* not set' && exit 1; fi

require-version: guard-VERSION

require-go-version:
	@ if go version | grep -q -v $(GO_VERSION); then \
		echo "go $(GO_VERSION) is required." && exit 1; \
	fi

require-dep:
	@if [ "$(DEP_BIN)" = "" ]; then \
		echo 'Missing "dep" command. See https://github.com/golang/dep or just brew install dep' && exit 1; \
	fi

require-upx:
	@if [ "$(UPX_BIN)" = "" ]; then \
		echo 'Missing "upx" command. See http://upx.sourceforge.net/' && exit 1; \
	fi

require-change:
	@ if [ "$(CHANGE_BIN)" = "" ]; then \
		echo 'Missing "github_changelog_generator" command. See https://github.com/github-changelog-generator/github-changelog-generator or just [sudo] gem install github_changelog_generator' && exit 1; \
	fi

build: require-dep require-go-version
	$(DEP_BIN) ensure && \
	mkdir -p $(BUILD_DIR) && \
	go build -o $(BUILD_DIR)/http-proxy \
	-ldflags="-X main.revision=$(GIT_REVISION)" \
	github.com/getlantern/http-proxy-lantern/http-proxy && \
	file $(BUILD_DIR)/http-proxy

distnochange: require-dep require-upx require-version require-change
	GOOS=linux GOARCH=amd64 BUILD_DIR=dist $(MAKE) build -o http-proxy && \
	upx dist/http-proxy

dist: require-dep require-upx require-version require-change distnochange
	$(call tag-changelog,http-proxy-lantern)

deploy: dist/http-proxy
	s3cmd put dist/http-proxy s3://http-proxy/http-proxy && \
	s3cmd setacl --acl-grant read:f87080f71ec0be3b9a933cbb244a6c24d4aca584ac32b3220f56d59071043747 s3://http-proxy/http-proxy

deploy-staging: dist/http-proxy
	s3cmd put dist/http-proxy s3://http-proxy/http-proxy-staging && \
	s3cmd setacl --acl-grant read:f87080f71ec0be3b9a933cbb244a6c24d4aca584ac32b3220f56d59071043747 s3://http-proxy/http-proxy-staging

clean:
	rm -rf dist bin


system-checks:
	@if [[ -z "$(DOCKER)" ]]; then echo 'Missing "docker" command.'; exit 1; fi && \
	if [[ -z "$(GO)" ]]; then echo 'Missing "go" command.'; exit 1; fi

docker-builder: system-checks
	DOCKER_CONTEXT=.$(DOCKER_IMAGE_TAG)-context && \
	mkdir -p $$DOCKER_CONTEXT && \
	cp Dockerfile $$DOCKER_CONTEXT && \
	docker build -t $(DOCKER_IMAGE_TAG) --build-arg go_version=go$(GO_VERSION) $$DOCKER_CONTEXT

# workaround to build Ubuntu binary on non-Ubuntu platforms.
docker-distnochange: docker-builder require-dep
	docker run -e GIT_REVISION='$(GIT_REVISION)' \
	-e SRCDIR='github.com/getlantern/http-proxy-lantern' \
	-v $$PWD/../../..:/src -t $(DOCKER_IMAGE_TAG) \
	'go build -o $$SRCDIR/dist/http-proxy -ldflags="-X main.revision=$$GIT_REVISION" $$SRCDIR/http-proxy' && \
	file dist/http-proxy

docker-dist: require-dep require-upx require-version require-change docker-distnochange
	$(call tag-changelog,http-proxy-lantern)
