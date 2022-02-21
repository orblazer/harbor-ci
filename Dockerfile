# syntax=docker/dockerfile:1

##
## Build
##
FROM golang:1.17-alpine as build

ARG VERSION
ARG BUILDTIME
ARG REVISION

WORKDIR /src

# Download mod
COPY src/go.mod .
# COPY src/go.sum .
# RUN go mod download

# Copy source
COPY src .

# Build image
RUN CGO_ENABLED=0 go build -o /harbor-cli \
  -ldflags="-X 'github.com/orblazer/harbor-cli/build.Version=$VERSION' \
  -X 'github.com/orblazer/harbor-cli/build.Time=$BUILDTIME' \
  -X 'github.com/orblazer/harbor-cli/build.Revision=$REVISION'"

##
## Generate latest ca-certificates
##

FROM debian:buster-slim AS certs

RUN \
  apt update && \
  apt install -y ca-certificates && \
  cat /etc/ssl/certs/* > /ca-certificates.crt

##
## Deploy
##
FROM scratch

COPY --from=build /harbor-cli /usr/local/bin/harbor-cli
COPY --from=certs /ca-certificates.crt /etc/ssl/certs/

COPY --from=busybox:1.34.1 /bin /busybox
# Since busybox needs some lib files which lie in /lib directory to run the executables on s390x,
# the below COPY command is added to address "ld64.so.1 not found" issue. This extra copy action will not
# happen on amd64 or arm64 platforms since /lib does not exist in amd64 or arm64 version of busybox container.
# Similar issues could be found in https://github.com/multiarch/qemu-user-static/issues/110#issuecomment-652951564.
COPY --from=busybox:1.34.1 /*lib /lib
# Declare /busybox as a volume to get it automatically in the path to ignore
VOLUME /busybox

ENV HOME /root
ENV USER root
ENV PATH /usr/local/bin:/busybox

WORKDIR /workspace

RUN ["/busybox/mkdir", "-p", "/bin"]
RUN ["/busybox/ln", "-s", "/busybox/sh", "/bin/sh"]

ENTRYPOINT [ "harbor-cli" ]
