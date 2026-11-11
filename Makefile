GO_LINT_CONFIG     ?= .build/golangci.yaml
CANONICAL_LINT_URL := https://raw.githubusercontent.com/yama6a/gha/v2/.golangci.yaml
GOVULNCHECK_URL    := https://raw.githubusercontent.com/yama6a/gha/v2/scripts/govulncheck.sh
FUZZTIME           ?= 30s
BENCHTIME          ?= 1s

.PHONY: lint-config generate fmt fmt-check lint vet test cover vuln tidy tidy-check \
	generate-check mod ci cross fuzz bench golden

lint-config:
	mkdir -p .build
	curl -fsSL $(CANONICAL_LINT_URL) -o .build/canonical-golangci.yaml
	if [ -f .golangci.local.yaml ]; then \
		yq eval-all '. as $$item ireduce ({}; . *+ $$item)' \
			.build/canonical-golangci.yaml .golangci.local.yaml > $(GO_LINT_CONFIG); \
	else \
		cp .build/canonical-golangci.yaml $(GO_LINT_CONFIG); \
	fi

generate:
	go generate ./...

fmt: lint-config
	golangci-lint fmt -c $(GO_LINT_CONFIG)

fmt-check: lint-config
	golangci-lint fmt --diff -c $(GO_LINT_CONFIG)

lint: lint-config
	golangci-lint run ./... -c $(GO_LINT_CONFIG)

vet:
	go vet ./...

test:
	go test ./... -race -count=1

cover:
	go test ./... -coverprofile=cover.out -covermode=atomic
	go tool cover -func=cover.out | tail -1

# The same wrapper CI runs, so a .govulncheck-ignore entry counts in both places.
vuln:
	mkdir -p .build
	curl -fsSL $(GOVULNCHECK_URL) -o .build/govulncheck.sh
	bash .build/govulncheck.sh

tidy:
	go mod tidy

tidy-check:
	go mod tidy
	git diff --exit-code -- go.mod go.sum

generate-check: generate
	git diff --exit-code

mod:
	go get -u -t ./...
	go mod tidy

ci: tidy-check generate-check fmt-check lint vet test vuln

# go-ci builds linux/amd64 and linux/arm64 only. Importers also build this module on darwin and windows, so it must compile there.
cross:
	for target in darwin/amd64 darwin/arm64 windows/amd64 windows/arm64; do \
		CGO_ENABLED=0 GOOS=$${target%/*} GOARCH=$${target#*/} go build ./... || exit 1; \
		CGO_ENABLED=0 GOOS=$${target%/*} GOARCH=$${target#*/} go vet ./... || exit 1; \
	done

# go test -fuzz takes one package and one target per run.
fuzz:
	pkgs=$$(go list ./...) || exit 1; \
	for pkg in $$pkgs; do \
		targets=$$(go test -list '^Fuzz' $$pkg) || exit 1; \
		for target in $$(echo "$$targets" | grep '^Fuzz'); do \
			go test -run '^$$' -fuzz "^$$target\$$" -fuzztime $(FUZZTIME) $$pkg || exit 1; \
		done; \
	done

bench:
	go test -run '^$$' -bench . -benchmem -benchtime $(BENCHTIME) ./...

# Only the packages that declare the -update flag, because another test binary rejects the unknown flag.
golden:
	pkgs=$$(grep -rl --include='*_test.go' 'flag.Bool("update"' . | xargs -r -n1 dirname | sort -u); \
	if [ -z "$$pkgs" ]; then echo "no test declares the -update flag" >&2; exit 1; fi; \
	go test -count=1 -run Golden $$pkgs -update
