FROM  golang:1.10-alpine as builder

ARG GOOS=linux
ARG GOARCH=amd64
ARG GOARM=

WORKDIR  /go/src/github.com/rchicoli/docker-log-elasticsearch
COPY . .

RUN apk add --no-cache git curl

# install dep
RUN curl https://raw.githubusercontent.com/golang/dep/master/install.sh | sh

# RUN go get -d -v ./...
RUN dep ensure -v

# vendor/github.com/docker/docker/pkg/term/tc_linux_cgo.go:10:22:
#  exec: "gcc": executable file not found in $PATH
#  fatal error: termios.h: No such file or directory
RUN apk add --no-cache dev86 gcc musl-dev

# https://github.com/docker-library/golang/issues/86
RUN go list ./...
RUN go test -cover ./...

RUN CGO_ENABLED=0 go build -v -a -installsuffix cgo -o /usr/bin/docker-log-elasticsearch

FROM alpine:3.7

# TZ required to set the localtime
# TZ can be set with docker plugin command
RUN apk --no-cache add tzdata

COPY --from=builder /usr/bin/docker-log-elasticsearch /usr/bin/
WORKDIR /usr/bin
ENTRYPOINT [ "/usr/bin/docker-log-elasticsearch" ]