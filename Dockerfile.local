FROM alpine:3.7

ARG GOOS=linux
ARG GOARCH=amd64
ARG GOARM=

COPY docker-log-elasticsearch /usr/bin/

# TZ required to set the localtime
# TZ can be set with docker plugin command
RUN apk --no-cache add tzdata

WORKDIR /usr/bin
ENTRYPOINT [ "/usr/bin/docker-log-elasticsearch" ]