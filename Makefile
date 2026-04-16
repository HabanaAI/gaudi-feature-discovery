MAIN_PACKAGE_PATH := "./cmd/"

APPLICATION_NAME := habanalabs-feature-discovery
BINARY_NAME := hfd
BINARY_VERSION := 1.0.0
GO_LINT_VERSION := 2.7.2

REGISTRY ?= vault.habana.ai
IMAGE_TAG ?= 1.0.0
IMAGE_PATH = $(REGISTRY)/$(APPLICATION_NAME):$(IMAGE_TAG)
UPDATE_TYPE ?= "patch"

# ==================================================================================== #
# HELPERS
# ==================================================================================== #

## help: print this help message
# .PHONY: help
help:
	@echo 'Usage:'
	@sed -n 's/^##//p' ${MAKEFILE_LIST} | column -t -s ':' |  sed -e 's/^/ /'


# ==================================================================================== #
# QUALITY CONTROL
# ==================================================================================== #

## tidy: tidy modfile
.PHONY: tidy
tidy:
	go mod tidy

## fmt: format code
.PHONY: fmt
fmt:
	go fmt ./...

## audit: run quality control checks
.PHONY: audit
audit:
	go mod verify
	go vet ./...

## lint: run golangci-lint checks
lint:
	docker run --rm -t -v $(shell pwd):/app -w /app \
		golangci/golangci-lint:v$(GO_LINT_VERSION) golangci-lint run

static-analysis: lint audit fmt tidy

# ==================================================================================== #
# DEVELOPMENT
# ==================================================================================== #

## test: run all tests
.PHONY: test
test:
	go test -v -count=1 -race -buildvcs -vet=off ./...

## coverage: run all tests and display coverage
.PHONY: coverage
coverage:
	go test -v -count=1 -race -buildvcs -coverprofile=/tmp/coverage.out ./...
	go tool cover -html=/tmp/coverage.out

## build: build the application
.PHONY: build/bin
build/bin:
	CGO_ENABLED=0 go build -o=./bin/${BINARY_NAME} -ldflags="-w -s -X main.Version=${BINARY_VERSION}" $(MAIN_PACKAGE_PATH)

## run: run the  application
.PHONY: run
run: build
	/tmp/bin/${BINARY_NAME}

# ==================================================================================== #
# OPERATIONS
# ==================================================================================== #

## docker-build: Build the container using docker
.PHONY: docker-build
docker-build:
	docker build \
	--platform linux/amd64 \
	--build-arg HFD_VERSION="$(BINARY_VERSION)" \
	--build-arg BUILD_DATE=`date -u +"%Y-%m-%dT%H:%M:%SZ"` \
	-t $(IMAGE_PATH) \
	.

## docker-push: push the container to registry
.PHONY: docker-push
docker-push:
	docker push $(IMAGE_PATH)

.PHONY: kustomize
kustomize:
	@cd manifests && kustomize edit set image hfd-image=$(IMAGE_PATH) && kustomize build

# upgrade
.PHONY: update
update:
	@if [ "$(UPDATE_TYPE)" = "patch" ]; then \
		GO_MINOR=$$(awk '/^go / {split($$2, v, "."); print v[1] "." v[2]; exit}' go.mod) && \
		go get go@$$GO_MINOR && \
		go get toolchain@go$$GO_MINOR; \
	else \
		go get go@latest && \
		go get toolchain@latest; \
	fi
	go get -u ./... && \
	GO_VERSION=$$(awk '/^go / {print $$2; exit}' go.mod) && \
		sed -i "s/FROM golang:.* AS golang/FROM golang:$$GO_VERSION AS golang/g" Dockerfile

.PHONY: vet
vet: ## Run go vet against code.
	go vet ./...

## Remember to 'export GOTOOLCHAIN=auto' before running this target to use the latest Go toolchain.
.PHONY: upgrade
upgrade: update tidy fmt vet

