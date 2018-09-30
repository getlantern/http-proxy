DEP_BIN      ?= $(shell which dep)
UPX_BIN      ?= $(shell which upx)
BUILD_DIR    ?= bin
GIT_REVISION := $(shell git rev-parse --short HEAD)
CHANGE_BIN   := $(shell which github_changelog_generator)

GO_VERSION := 1.11

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

dist: require-dep require-upx require-version require-change
	GOOS=linux GOARCH=amd64 BUILD_DIR=dist $(MAKE) build -o http-proxy && \
	upx dist/http-proxy && \
	$(call tag-changelog,http-proxy-lantern)

deploy: dist/http-proxy
	s3cmd put dist/http-proxy s3://http-proxy/http-proxy && \
	s3cmd setacl --acl-grant read:f87080f71ec0be3b9a933cbb244a6c24d4aca584ac32b3220f56d59071043747 s3://http-proxy/http-proxy

deploy-staging: dist/http-proxy
	s3cmd put dist/http-proxy s3://http-proxy/http-proxy-staging && \
	s3cmd setacl --acl-grant read:f87080f71ec0be3b9a933cbb244a6c24d4aca584ac32b3220f56d59071043747 s3://http-proxy/http-proxy-staging

clean:
	rm -rf dist bin
