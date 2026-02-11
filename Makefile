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

.PHONY: ensure format lint build test install

ensure:
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
