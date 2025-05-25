SHELL = /bin/bash -o pipefail

.PHONY: test test-ci lint lint-ci fmt fmt-ci upgrade clean release lint-docs audit encrypt decrypt sops

test:
	gotestsum --format pkgname --packages ./... -- -race -timeout 10m -cover -covermode atomic

test-ci:
	gotestsum --format pkgname --packages ./... --junitfile tests.xml -- -race -timeout 10m -coverprofile=coverage.txt -covermode atomic
	gocover-cobertura < coverage.txt > coverage.xml
	go tool cover -html=coverage.txt -o coverage.html

lint:
	golangci-lint run --timeout 4m --color always --allow-parallel-runners --fix --max-issues-per-linter 0 --max-same-issues 0

lint-ci:
	golangci-lint run --timeout 4m --max-issues-per-linter 0 --max-same-issues 0 --out-format colored-line-number,code-climate:codeclimate.json

fmt:
	go mod tidy
	git ls-files --cached --modified --other --exclude-standard -z | grep -z -Z '.go$$' | xargs -0 gofumpt -w
	git ls-files --cached --modified --other --exclude-standard -z | grep -z -Z '.go$$' | xargs -0 goimports -w -local gitlab.com/tozd/go/errors

fmt-ci: fmt
	git diff --exit-code --color=always

upgrade:
	go run github.com/icholy/gomajor@v0.13.2 get all
	go mod tidy

clean:
	rm -f coverage.* codeclimate.json tests.xml

release:
	npx --yes --package 'release-it@15.4.2' --package '@release-it/keep-a-changelog@3.1.0' -- release-it

lint-docs:
	npx --yes --package 'markdownlint-cli@~0.34.0' -- markdownlint --ignore-path .gitignore --ignore testdata/ '**/*.md'

audit:
	go list -json -deps ./... | nancy sleuth --skip-update-check

encrypt:
	gitlab-config sops --encrypt --mac-only-encrypted --in-place --encrypted-comment-regex sops:enc .gitlab-conf.yml

decrypt:
	SOPS_AGE_KEY_FILE=keys.txt gitlab-config sops --decrypt --in-place .gitlab-conf.yml

sops:
	SOPS_AGE_KEY_FILE=keys.txt gitlab-config sops .gitlab-conf.yml
