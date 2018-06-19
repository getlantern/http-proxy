DEP_BIN      ?= $(shell which dep)
UPX_BIN      ?= $(shell which upx)
BUILD_DIR    ?= bin
GIT_REVISION := $(shell git rev-parse --short HEAD)
CHANGE_BIN   := $(call get-command,github_changelog_generator)

.PHONY: dist build require-dep

# This tags the current version and creates a CHANGELOG for the current directory.
define tag-changelog
	echo "Tagging..." && \
	git tag -a "$$VERSION" -f --annotate -m"Tagged $$VERSION" && \
	git push --tags -f && \
	$(CHANGE) getlantern/$(1) && \
	git add CHANGELOG.md && \
	git commit -m "Updated changelog for $$VERSION" && \
	git push origin HEAD
endef

require-version:
	@if [[ -z "$$VERSION" ]]; then echo "VERSION environment value is required."; exit 1; fi

require-gh-token:
	@if [[ -z "$$CHANGELOG_GITHUB_TOKEN" ]]; then echo "CHANGELOG_GITHUB_TOKEN environment value is required."; exit 1; fi

require-dep:
	@if [ "$(DEP_BIN)" = "" ]; then \
		echo 'Missing "dep" command. See https://github.com/golang/dep or just brew install dep' && exit 1; \
	fi

require-upx:
	@if [ "$(UPX_BIN)" = "" ]; then \
		echo 'Missing "upx" command. See http://upx.sourceforge.net/' && exit 1; \
	fi

require-change:
	@if [ "$(CHANGE_BIN)" = "" ]; then \
		echo 'Missing "github_changelog_generator" command. See https://github.com/github-changelog-generator/github-changelog-generator or just [sudo] gem install github_changelog_generator' && exit 1; \
	fi

build: require-dep
	$(DEP_BIN) ensure && \
	mkdir -p $(BUILD_DIR) && \
	go build -o $(BUILD_DIR)/http-proxy \
	-ldflags="-X main.revision=$(GIT_REVISION)" \
	github.com/getlantern/http-proxy-lantern/http-proxy && \
	file $(BUILD_DIR)/http-proxy

dist: require-dep require-upx require-version require-gh-token require-change
	GOOS=linux GOARCH=amd64 BUILD_DIR=dist $(MAKE) build -o http-proxy && \
	upx dist/http-proxy && \
	$(call tag-changelog,http-proxy-lantern)

deploy: dist/http-proxy
	s3cmd put dist/http-proxy s3://http-proxy/http-proxy && \
	s3cmd setacl --acl-grant read:f87080f71ec0be3b9a933cbb244a6c24d4aca584ac32b3220f56d59071043747 s3://http-proxy/http-proxy

clean:
	rm -rf dist bin
