FROM  golang:1.9.2-alpine as builder

ARG GOOS=linux
ARG GOARCH=amd64
ARG GOARM=

WORKDIR  /go/src/github.com/rchicoli/docker-log-elasticsearch
COPY . .

RUN apk add --no-cache git

RUN go get -d -v ./...
RUN CGO_ENABLED=0 go build -v -a -installsuffix cgo -o /usr/bin/docker-log-elasticsearch

FROM alpine:3.7

# RUN apk --no-cache add ca-certificates
COPY --from=builder /usr/bin/docker-log-elasticsearch /usr/bin/
WORKDIR /usr/bin
ENTRYPOINT [ "/usr/bin/docker-log-elasticsearch" ]