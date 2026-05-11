PROJECT         := github.com/pulumi/pulumi-converter-terraform
BINARY          := pulumi-converter-terraform
VERSION         ?= $(shell pulumictl get version)
VERSION_PATH    := pkg/version.Version
WORKING_DIR     := $(shell pwd)
TESTPARALLELISM := 4
GOPATH          := $(shell go env GOPATH)
#
# Additional arguments to pass to golangci-lint.
GOLANGCI_LINT_ARGS ?=

.PHONY: ensure format lint build test install sync-pcl

ensure:
	go mod tidy

# Pin github.com/pulumi/pulumi/sdk/pcl/v3 to the commit tagged as
# pkg/$(pkg/v3-version) in pulumi/pulumi. sdk/pcl has no release tags of its
# own, so Renovate cannot bump it directly (see renovatebot/renovate#34705).
sync-pcl:
	@PKG_VERSION=$$(go list -m -f '{{.Version}}' github.com/pulumi/pulumi/pkg/v3); \
	echo "Resolving sdk/pcl commit for pkg/$$PKG_VERSION"; \
	PKG_SHA=$$(git ls-remote https://github.com/pulumi/pulumi "refs/tags/pkg/$$PKG_VERSION" | cut -f1); \
	if [ -z "$$PKG_SHA" ]; then echo "Could not resolve refs/tags/pkg/$$PKG_VERSION" >&2; exit 1; fi; \
	go get "github.com/pulumi/pulumi/sdk/pcl/v3@$$PKG_SHA"; \
	go mod tidy

format:
	gofumpt -w cmd pkg

lint:
	golangci-lint run $(GOLANGCI_LINT_ARGS) -c ./.golangci.yml --timeout 10m

build:
	go build -o $(WORKING_DIR)/bin/${BINARY} -ldflags "-X ${PROJECT}/${VERSION_PATH}=${VERSION}" $(PROJECT)/cmd/$(BINARY)

test:
	go test -v -count=1 -cover -timeout 2h -parallel ${TESTPARALLELISM} ./...

install: build
	cp $(WORKING_DIR)/bin/${BINARY} ${GOPATH}/bin

generate_builtins_test:
	if [ ! -d ./scripts/venv ]; then python -m venv ./scripts/venv; fi
	. ./scripts/venv/*/activate && python -m pip install -r ./scripts/requirements.txt
	. ./scripts/venv/*/activate &&  python ./scripts/generate_builtins.py
