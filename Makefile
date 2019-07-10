NAME				 := noderig
BUILD_DIR  	 := build
GITHASH			 := $(shell git rev-parse HEAD)

CC					 := GO111MODULE=on go build
CROSS				 := GOOS=linux GOARCH=amd64
DFLAGS			 := -race
CFLAGS			 := -i -v -mod vendor
LDFLAGS			 := -X github.com/ovh/noderig/cmd.githash=$(GITHASH)

rwildcard		 := $(foreach d,$(wildcard $1*),$(call rwildcard,$d/,$2) $(filter $(subst *,%,$2),$d))

FORMAT_PATHS := ./cmd ./core ./collectors $(NAME).go
MODULE_PATHS := ./cmd/... ./core/... ./collectors/...
FILE_PATHS	 := $(call rwildcard, cmd, *.go) $(call rwildcard, core, *.go) $(NAME).go

.PHONY: all
all: init dep format lint release

.PHONY: init
init:
	GO111MODULE=on go get -u -v github.com/golangci/golangci-lint/cmd/golangci-lint

.PHONY: dep
dep:
	GO111MODULE=on go mod vendor -v

.PHONY:	fmt
format: $(FILE_PATHS)
	gofmt -s -w $(FORMAT_PATHS)

.PHONY: lint
lint: $(FILE_PATHS)
	golangci-lint run

.PHONY: build
build: $(FILE_PATHS)
	$(CC) $(CFLAGS) $(DFLAGS) -ldflags '$(LDFLAGS)' -o $(BUILD_DIR)/$(NAME)

.PHONY: dev
dev: fmt lint build

.PHONY: release
release: $(FILE_PATHS)
	$(CC) $(CFLAGS) -ldflags '$(LDFLAGS) -s -w' -o $(BUILD_DIR)/$(NAME)

.PHONY: dist
dist: $(FILE_PATHS)
	$(CROSS) $(CC) $(CFLAGS) -ldflags '$(LDFLAGS) -s -w' -o $(BUILD_DIR)/$(NAME)

.PHONY: clean
clean:
	rm -rfv $(BUILD_DIR)
	rm -rfv vendor

# Docker build

.PHONY: build-docker
build-docker: build-go-in-docker build-docker-image

.PHONY: build-go-in-docker
build-go-in-docker:
	docker run --rm \
		-e GOBIN=/go/bin/ -e CGO_ENABLED=0 -e GOPATH=/go \
		-v ${PWD}:/go/src/github.com/ovh/noderig \
		-w /go/src/github.com/ovh/noderig \
		golang:1.12.6 \
			make

.PHONY: build-docker-image
build-docker-image:
	docker build -t ovh/noderig .

.PHONY: run
run:
	docker run --rm --net host ovh/noderig
