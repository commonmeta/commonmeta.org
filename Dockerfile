# syntax=docker/dockerfile:1.5
ARG BUILDPLATFORM=linux/amd64
FROM --platform=$BUILDPLATFORM debian:bookworm as builder

ARG PB_VERSION=0.22.7
ADD https://github.com/pocketbase/pocketbase/releases/download/v${PB_VERSION}/pocketbase_${PB_VERSION}_linux_amd64.zip /tmp/pocketbase_${PB_VERSION}_linux_amd64.zip
RUN apt-get update && apt-get install -y unzip && \
    unzip /tmp/pocketbase_${PB_VERSION}_linux_amd64.zip -d /pb/
    
FROM --platform=$BUILDPLATFORM debian:bookworm-slim as runtime

COPY --from=builder /pb/ /pb/
COPY ./pb_public/. /pb/pb_public/

EXPOSE 8090

# start pocketbase
CMD ["/pb/pocketbase", "serve", "--http=0.0.0.0:8090"]
