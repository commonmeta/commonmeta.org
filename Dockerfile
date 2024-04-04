# syntax=docker/dockerfile:1.5
ARG BUILDPLATFORM=linux/amd64
FROM --platform=$BUILDPLATFORM python:3.12-bookworm as builder

# ARG PB_VERSION=0.22.7

# ADD https://github.com/pocketbase/pocketbase/releases/download/v${PB_VERSION}/pocketbase_${PB_VERSION}_linux_amd64.zip /tmp/pocketbase_${PB_VERSION}_linux_amd64.zip
ENV PATH="$PATH:/usr/local/go/bin" \
    GOCACHE="/tmp/gocache"

RUN apt-get update && \
    wget -c https://go.dev/dl/go1.22.2.linux-amd64.tar.gz && \
    tar -C /usr/local -xzf go1.22.2.linux-amd64.tar.gz && \
    go version && \
    mkdir /pb

COPY ./main.go /pb/main.go
COPY ./go.mod /pb/go.mod
COPY ./go.sum /pb/go.sum
COPY ./pb_public /pb/pb_public
COPY ./pb_migrations /pb/pb_migrations
WORKDIR /pb

RUN --mount=type=cache,target=$GOCACHE CGO_ENABLED=0 go build
    
FROM --platform=$BUILDPLATFORM python:3.12-slim-bookworm as runtime

COPY --from=builder /pb/ /pb/

EXPOSE 8090

# start commonmeta
CMD ["/pb/commonmeta", "serve", "--http=0.0.0.0:8090"]
