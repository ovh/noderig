BUILD_DIR=build

CC=go build
GITHASH=$(shell git rev-parse HEAD)
DFLAGS=-race
CFLAGS=-ldflags "-X github.com/ovh/noderig/cmd.githash=$(GITHASH)"
CROSS=GOOS=linux GOARCH=amd64

rwildcard=$(foreach d,$(wildcard $1*),$(call rwildcard,$d/,$2) $(filter $(subst *,%,$2),$d))
VPATH= $(BUILD_DIR)

.SECONDEXPANSION:

build: noderig.go $$(call rwildcard, ./cmd, *.go) $$(call rwildcard, ./collectors, *.go)
	$(CC) $(DFLAGS) $(CFLAGS) -o $(BUILD_DIR)/noderig noderig.go

.PHONY: release
release: noderig.go $$(call rwildcard, ./cmd, *.go) $$(call rwildcard, ./collectors, *.go)
	$(CC) $(CFLAGS) -o $(BUILD_DIR)/noderig noderig.go

.PHONY: dist
dist: noderig.go $$(call rwildcard, ./cmd, *.go) $$(call rwildcard, ./collectors, *.go)
	$(CROSS) $(CC) $(CFLAGS) -ldflags "-s -w" -o $(BUILD_DIR)/noderig noderig.go

.PHONY: lint
lint:
	@command -v gometalinter >/dev/null 2>&1 || { echo >&2 "gometalinter is required but not available please follow instructions from https://github.com/alecthomas/gometalinter"; exit 1; }
	gometalinter --deadline=180s --disable-all --enable=gofmt ./cmd/... ./core/... ./
	gometalinter --deadline=180s --disable-all --enable=vet ./cmd/... ./core/... ./
	gometalinter --deadline=180s --disable-all --enable=golint ./cmd/... ./core/... ./
	gometalinter --deadline=180s --disable-all --enable=ineffassign ./cmd/... ./core/... ./
	gometalinter --deadline=180s --disable-all --enable=misspell ./cmd/... ./core/... ./
	gometalinter --deadline=180s --disable-all --enable=staticcheck ./cmd/... ./core/... ./

.PHONY: format
format:
	gofmt -w -s ./cmd ./core noderig.go

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
	$(CC) $(CFLAGS) noderig.go

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
