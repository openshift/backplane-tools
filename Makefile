BUILDFLAGS ?=
unexport GOFLAGS

BASE_DIR=$(shell pwd)
BIN_DIR=${BASE_DIR}/bin

.DEFAULT_GOAL := all
.PHONY: all
all: vet fmt mod build test lint

.PHONY: fmt
fmt:
	@echo "gofmt"
	@gofmt -w -s .

OS := $(shell go env GOOS | sed 's/[a-z]/\U&/')
ARCH := $(shell go env GOARCH)

GORELEASER_VERSION="v1.15.0"
GOLANGCI_LINT_VERSION="v1.53.3"

.PHONY: download-goreleaser
download-goreleaser:
	GOBIN=${BIN_DIR} go install github.com/goreleaser/goreleaser@${GORELEASER_VERSION}

.PHONY: download-golangci-lint
download-golangci-lint:
	GOBIN=${BIN_DIR} go install github.com/golangci/golangci-lint/cmd/golangci-lint@${GOLANGCI_LINT_VERSION}

SINGLE_TARGET ?= false

# Need to use --snapshot here because the goReleaser
# requires more git info that is provided in Prow's clone.
# Snapshot allows the build without validation of the
# repository itself
.PHONY: build
build: download-goreleaser
	${BIN_DIR}/goreleaser build --clean --snapshot --id linux_arm64,linux_amd64,darwin_arm64,darwin_amd64 --single-target=${SINGLE_TARGET}

.PHONY: release
release:
	${BIN_DIR}/goreleaser release --clean

.PHONY: vet
vet:
	go vet ${BUILDFLAGS} ./...

.PHONY: mod
mod:
	go mod tidy

.PHONY: test
test:
	go test ${BUILDFLAGS} ./... -covermode=atomic -coverpkg=./...

.PHONY: lint
lint: download-golangci-lint
	${BIN_DIR}/golangci-lint run
