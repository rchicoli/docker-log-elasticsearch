FROM  golang:1.9.2 as builder

ARG GOOS=linux
ARG GOARCH=amd64
ARG GOARM=

WORKDIR  /go/src/github.com/rchicoli/docker-log-driver-test
COPY . .

RUN go get -d -v ./...
# RUN go build --ldflags '-extldflags "-static"' -o /usr/bin/docker-log-driver
RUN CGO_ENABLED=0 go build -v -a -installsuffix cgo -o /usr/bin/docker-log-driver

FROM alpine:3.7

RUN apk --no-cache add ca-certificates
COPY --from=builder /usr/bin/docker-log-driver /usr/bin/
WORKDIR /usr/bin
ENTRYPOINT [ "/usr/bin/docker-log-driver" ]