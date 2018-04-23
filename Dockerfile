FROM  golang:1.10-alpine as builder

ARG GOOS=linux
ARG GOARCH=amd64
ARG GOARM=

WORKDIR  /go/src/github.com/rchicoli/docker-log-elasticsearch
COPY . .

RUN apk add --no-cache git

RUN go get -d -v ./...
RUN go test -cover -v ./...
RUN CGO_ENABLED=0 go build -v -a -installsuffix cgo -o /usr/bin/docker-log-elasticsearch

FROM alpine:3.7

# TZ required to set the localtime
# TZ can be set with docker plugin command
RUN apk --no-cache add tzdata

COPY --from=builder /usr/bin/docker-log-elasticsearch /usr/bin/
WORKDIR /usr/bin
ENTRYPOINT [ "/usr/bin/docker-log-elasticsearch" ]