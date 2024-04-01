# syntax=docker/dockerfile:1.5
ARG BUILDPLATFORM=linux/amd64
FROM --platform=$BUILDPLATFORM python:3.12-bookworm as builder

ARG PB_VERSION=0.22.7
ARG QUARTO_VERSION=1.4.552

ADD https://github.com/pocketbase/pocketbase/releases/download/v${PB_VERSION}/pocketbase_${PB_VERSION}_linux_amd64.zip /tmp/pocketbase_${PB_VERSION}_linux_amd64.zip
ADD https://github.com/quarto-dev/quarto-cli/releases/download/v${QUARTO_VERSION}/quarto-${QUARTO_VERSION}-linux-amd64.deb /tmp/quarto-${QUARTO_VERSION}-linux-amd64.deb
RUN apt-get update && apt-get install -y \
    curl unzip gdebi-core ca-certificates && \
    unzip /tmp/pocketbase_${PB_VERSION}_linux_amd64.zip -d /pb/
    # dpkg -i /tmp/quarto-${QUARTO_VERSION}-linux-amd64.deb

FROM --platform=$BUILDPLATFORM python:3.12-slim-bookworm as runtime

# RUN --mount=type=cache,target=/var/cache/apt apt-get update -y && \
#     apt-get install curl unzip -y --no-install-recommends && \
#     apt-get clean && rm -rf /var/lib/apt/lists/*

# COPY --from=builder /usr/local/bin/quarto /usr/local/bin/quarto
COPY --from=builder /pb/ /pb/
COPY ./ /pb/
COPY ./pb_public /pb/pb_public
COPY ./pb_migrations /pb/pb_migrations
WORKDIR /pb

# uncomment to copy the local pb_hooks dir into the container
# COPY ./pb_hooks /pb/pb_hooks

EXPOSE 8080

# start PocketBase
CMD ["/pb/pocketbase", "serve", "--http=0.0.0.0:8080"]
