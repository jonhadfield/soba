SOURCE_FILES?=$$(go list ./...)
TEST_PATTERN?=.
TEST_OPTIONS?=-race -v

setup:
	go get -u github.com/go-critic/go-critic/...
	go get -u github.com/alecthomas/gometalinter
	go get -u golang.org/x/tools/cmd/cover
	gometalinter --install

clean:
	rm -rf ./dist

# This requires credentials are set for all providers!!!
test:
	echo 'mode: atomic' > coverage.txt && go list ./... | xargs -n1 -I{} sh -c 'go test -v -timeout=600s -covermode=atomic -coverprofile=coverage.tmp {} && tail -n +2 coverage.tmp >> coverage.txt' && rm coverage.tmp

cover: test
	go tool cover -html=coverage.txt

fmt:
	goimports -w . && gofumpt -l -w .

lint:
	golangci-lint run --disable lll --disable interfacer --disable gochecknoglobals --disable gochecknoinits --enable wsl --enable revive --enable gosec --enable exhaustruct --enable unused --enable gocritic --enable gofmt --enable goimports --enable misspell --enable unparam --enable goconst --enable wrapcheck
ci: lint test

BUILD_TAG := $(shell git describe --tags 2>/dev/null)
BUILD_SHA := $(shell git rev-parse --short HEAD)
BUILD_DATE := $(shell date -u '+%Y/%m/%d:%H:%M:%S')

build:
	CGO_ENABLED=0 go build -ldflags '-s -w -X "main.version=[$(BUILD_TAG)-$(BUILD_SHA)] $(BUILD_DATE) UTC"' -o ".local_dist/soba"

build-all: fmt
	GOOS=darwin  CGO_ENABLED=0 GOARCH=amd64 go build -ldflags '-s -w -X "main.version=[$(BUILD_TAG)-$(BUILD_SHA)] $(BUILD_DATE) UTC"' -o ".local_dist/soba_darwin_amd64"
	GOOS=darwin  CGO_ENABLED=0 GOARCH=arm64 go build -ldflags '-s -w -X "main.version=[$(BUILD_TAG)-$(BUILD_SHA)] $(BUILD_DATE) UTC"' -o ".local_dist/soba_darwin_arm64"
	GOOS=linux   CGO_ENABLED=0 GOARCH=amd64 go build -ldflags '-s -w -X "main.version=[$(BUILD_TAG)-$(BUILD_SHA)] $(BUILD_DATE) UTC"' -o ".local_dist/soba_linux_amd64"
	GOOS=linux   CGO_ENABLED=0 GOARCH=386 go build -ldflags '-s -w -X "main.version=[$(BUILD_TAG)-$(BUILD_SHA)] $(BUILD_DATE) UTC"' -o ".local_dist/soba_linux_386"
	GOOS=linux   CGO_ENABLED=0 GOARCH=arm go build -ldflags '-s -w -X "main.version=[$(BUILD_TAG)-$(BUILD_SHA)] $(BUILD_DATE) UTC"' -o ".local_dist/soba_linux_arm"
	GOOS=linux   CGO_ENABLED=0 GOARCH=arm64 go build -ldflags '-s -w -X "main.version=[$(BUILD_TAG)-$(BUILD_SHA)] $(BUILD_DATE) UTC"' -o ".local_dist/soba_linux_arm64"
	GOOS=netbsd  CGO_ENABLED=0 GOARCH=amd64 go build -ldflags '-s -w -X "main.version=[$(BUILD_TAG)-$(BUILD_SHA)] $(BUILD_DATE) UTC"' -o ".local_dist/soba_netbsd_amd64"
	GOOS=openbsd CGO_ENABLED=0 GOARCH=amd64 go build -ldflags '-s -w -X "main.version=[$(BUILD_TAG)-$(BUILD_SHA)] $(BUILD_DATE) UTC"' -o ".local_dist/soba_openbsd_amd64"
	GOOS=freebsd CGO_ENABLED=0 GOARCH=amd64 go build -ldflags '-s -w -X "main.version=[$(BUILD_TAG)-$(BUILD_SHA)] $(BUILD_DATE) UTC"' -o ".local_dist/soba_freebsd_amd64"
	GOOS=windows CGO_ENABLED=0 GOARCH=amd64 go build -ldflags '-s -w -X "main.version=[$(BUILD_TAG)-$(BUILD_SHA)] $(BUILD_DATE) UTC"' -o ".local_dist/soba_windows_amd64.exe"

critic:
	gocritic check  ./...

mac-install: build
	install .local_dist/soba /usr/local/bin/soba

install: build
	go install ./...

find-updates:
	go list -u -m -json all | go-mod-outdated -update -direct

pull-image:
	docker pull jonhadfield/soba:latest

scan-image: pull-image
	trivy image jonhadfield/soba:latest

release:
	goreleaser && git push --follow-tags

help:
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

.DEFAULT_GOAL := build
