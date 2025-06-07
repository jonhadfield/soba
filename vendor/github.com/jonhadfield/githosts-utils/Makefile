SOURCE_FILES?=$$(go list ./... | grep -v /vendor/ | grep -v /mocks/)
TEST_PATTERN?=.
TEST_OPTIONS?=-race -v

setup:
	GO111MODULE=on go get mvdan.cc/gofumpt
	go install -v golang.org/x/tools/cmd/goimports@latest
	go install -v mvdan.cc/gofumpt@latest
	go get -u golang.org/x/tools/cmd/cover

test:
	echo 'mode: atomic' > coverage.txt && go list ./... | grep -v testing.go | xargs -n1 -I{} sh -c 'go test -v -failfast -p 1 -parallel 1 -timeout=600s -covermode=atomic -coverprofile=coverage.tmp {} && tail -n +2 coverage.tmp >> coverage.txt' && rm coverage.tmp

cover: test
	go tool cover -html=coverage.txt
    # don't open browser...	go tool cover -html=coverage.txt -o coverage.html

fmt:
	goimports -w . && gofumpt -l -w .

lint:
	golangci-lint run --config .golangci.yml

ci: lint test

BUILD_TAG := $(shell git describe --tags 2>/dev/null)
BUILD_SHA := $(shell git rev-parse --short HEAD)
BUILD_DATE := $(shell date -u '+%Y/%m/%d:%H:%M:%S')

critic:
	gocritic check ./...

help:
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

.DEFAULT_GOAL := build

