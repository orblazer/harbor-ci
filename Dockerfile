# syntax=docker/dockerfile:1

##
## Build
##
FROM golang:1.17 as build

ARG VERSION
ARG BUILDTIME
ARG REVISION

WORKDIR /src

# Download mod
COPY src/go.mod .
COPY src/go.sum .
RUN go mod download

# Copy source
COPY src/**/*.go .

# Build image
RUN CGO_ENABLED=0 go build -o /harbor-ci \
  -ldflags="-X 'src/build.Version=$VERSION' -X 'app/build.Time=$BUILDTIME' -X 'app/build.Revision=$REVISION'"

##
## Deploy
##
FROM scratch

COPY --from=build /harbor-ci /usr/local/bin/harbor-ci

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

ENTRYPOINT [ "harbor-ci" ]
