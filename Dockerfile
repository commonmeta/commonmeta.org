# syntax=docker/dockerfile:1.5
ARG BUILDPLATFORM=linux/amd64
FROM --platform=$BUILDPLATFORM python:3.12-slim-bookworm as builder

ARG PB_VERSION=0.22.7

ADD https://github.com/pocketbase/pocketbase/releases/download/v${PB_VERSION}/pocketbase_${PB_VERSION}_linux_amd64.zip /tmp/pocketbase_${PB_VERSION}_linux_amd64.zip
RUN apt-get update && apt-get install -y \
    curl unzip ca-certificates && \
    unzip /tmp/pocketbase_${PB_VERSION}_linux_amd64.zip -d /pb/

FROM --platform=$BUILDPLATFORM python:3.12-slim-bookworm as runtime

# COPY --from=builder /pb/ /pb/
COPY ./commonmeta /pb/pocketbase
COPY ./pb_public /pb/pb_public
COPY ./pb_migrations /pb/pb_migrations

# uncomment to copy the local pb_hooks dir into the container
# COPY ./pb_hooks /pb/pb_hooks

EXPOSE 8080

# start PocketBase
CMD ["/pb/pocketbase", "serve", "--http=0.0.0.0:8080"]
