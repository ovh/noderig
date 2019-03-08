BUILD_DIR=build

CC=go build
GITHASH=$(shell git rev-parse HEAD)
DFLAGS=-race
CFLAGS=-X github.com/ovh/noderig/cmd.githash=$(GITHASH)
CROSS=GOOS=linux GOARCH=amd64

rwildcard=$(foreach d,$(wildcard $1*),$(call rwildcard,$d/,$2) $(filter $(subst *,%,$2),$d))
VPATH= $(BUILD_DIR)

LINT_PATHS= ./cmd/... ./collectors/... ./core/... ./

.SECONDEXPANSION:

build: noderig.go $$(call rwildcard, ./cmd, *.go) $$(call rwildcard, ./collectors, *.go)
	$(CC) $(DFLAGS) -ldflags "$(CFLAGS)" -o $(BUILD_DIR)/noderig noderig.go

.PHONY: release
release: noderig.go $$(call rwildcard, ./cmd, *.go) $$(call rwildcard, ./collectors, *.go)
	$(CC) -ldflags "$(CFLAGS)" -o $(BUILD_DIR)/noderig noderig.go

.PHONY: dist
dist: noderig.go $$(call rwildcard, ./cmd, *.go) $$(call rwildcard, ./collectors, *.go)
	$(CROSS) $(CC) -ldflags "$(CFLAGS) -s -w" -o $(BUILD_DIR)/noderig noderig.go

.PHONY: lint
lint:
	$(GOPATH)/bin/golangci-lint run --enable-all \
		--disable gochecknoinits \
		--disable gochecknoglobals \
		--disable scopelint \
		--disable goimports \
		$(LINT_PATHS)

.PHONY: format
format:
	gofmt -w -s ./cmd ./core ./collectors noderig.go

.PHONY: dev
dev: format lint build

.PHONY: clean
clean:
	rm -rf $BUILD_DIR

# Docker build

build-docker: build-go-in-docker build-docker-image

glide-install:
	go get github.com/Masterminds/glide
	glide install

go-build-in-docker:
	$(CC) -ldflags "$(CFLAGS)" noderig.go

build-go-in-docker:
	docker run --rm \
		-e GOBIN=/go/bin/ -e CGO_ENABLED=0 -e GOPATH=/go \
		-v ${PWD}:/go/src/github.com/ovh/noderig \
		-w /go/src/github.com/ovh/noderig \
		golang:1.8.0 \
			make glide-install go-build-in-docker

build-docker-image:
	docker build -t ovh/noderig .

run:
	docker run --rm --net host ovh/noderig
