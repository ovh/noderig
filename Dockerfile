FROM golang:alpine3.7 AS build-env

LABEL MAINTAINER Rachid Zarouali <xinity77@gmail.com>

# install dependencies
RUN apk add --no-cache glide make git

# clone noderig repository and build binary

RUN git clone https://github.com/ovh/noderig.git $GOPATH/src/github.com/ovh/noderig \
    && cd $GOPATH/src/github.com/ovh/noderig \
    && glide install \
    && go build noderig.go

# final stage
FROM alpine:3.7
COPY --from=build-env /go/src/github.com/ovh/noderig/noderig /
EXPOSE 9100
ENTRYPOINT ["/fossil"]