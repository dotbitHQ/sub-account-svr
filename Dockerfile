# syntax=docker/dockerfile:1

##
## Build
##
FROM golang:1.16-buster AS build

WORKDIR /app

COPY . ./

RUN go build -ldflags -s -v -o sub-account cmd/main.go

##
## Deploy
##
FROM ubuntu

ARG TZ=Asia/Shanghai

RUN export DEBIAN_FRONTEND=noninteractive \
    && apt-get update \
    && apt-get install -y ca-certificates tzdata \
    && ln -fs /usr/share/zoneinfo/${TZ} /etc/localtime \
    && echo ${TZ} > /etc/timezone \
    && dpkg-reconfigure tzdata \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app

COPY --from=build /app/sub-account /app/sub-account
COPY --from=build /app/config/config.example.yaml /app/config/config.yaml

EXPOSE 8125 8126

ENTRYPOINT ["/app/sub-account", "--config", "/app/config/config.yaml"]
