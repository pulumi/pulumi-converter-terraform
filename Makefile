PROJECT         := github.com/pulumi/pulumi-converter-terraform
BINARY          := pulumi-converter-terraform
VERSION         ?= $(shell pulumictl get version)
VERSION_PATH    := pkg/version.Version
WORKING_DIR     := $(shell pwd)
TESTPARALLELISM := 4
GOPATH			:= $(shell go env GOPATH)

ensure::
	go mod tidy

lint::
	cd "pkg" && golangci-lint run -c ../.golangci.yml --timeout 10m
	cd "cmd" && golangci-lint run -c ../.golangci.yml --timeout 10m

build::
	(cd cmd && go build -o $(WORKING_DIR)/bin/${BINARY} -ldflags "-X ${PROJECT}/${VERSION_PATH}=${VERSION}" $(PROJECT)/cmd/$(BINARY))

test::
	cd pkg && go test -short -v -count=1 -cover -timeout 2h -parallel ${TESTPARALLELISM} ./...

install:: build
	cp $(WORKING_DIR)/bin/${BINARY} ${GOPATH}/bin