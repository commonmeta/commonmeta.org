# syntax=docker/dockerfile:1.5
ARG BUILDPLATFORM=linux/amd64
FROM --platform=$BUILDPLATFORM debian:bookworm as builder

ENV PATH="$PATH:/usr/local/go/bin"

RUN apt-get update && apt-get install -y wget && \
    wget -cq https://go.dev/dl/go1.22.2.linux-amd64.tar.gz && \
    tar -C /usr/local -xzf go1.22.2.linux-amd64.tar.gz

RUN mkdir -p /pb
COPY ./main.go /pb/main.go
COPY ./go.mod /pb/go.mod
COPY ./go.sum /pb/go.sum
WORKDIR /pb

RUN CGO_ENABLED=0 go build

FROM --platform=$BUILDPLATFORM debian:bookworm-slim as runtime

COPY --from=builder /pb/ /pb/

EXPOSE 8090

# start commonmeta
CMD ["/pb/commonmeta", "serve", "--http=0.0.0.0:8090"]
