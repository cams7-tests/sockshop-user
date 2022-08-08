FROM golang:1.15.15-alpine3.14

ENV MONGO_HOST 192.168.100.14:27017
ENV HATEAOS user
ENV USER_DATABASE mongodb

RUN apk update && apk add --no-cache git build-base

ENV GO111MODULE on

RUN go get github.com/cweill/gotests/gotests@v1.6.0
RUN go get github.com/fatih/gomodifytags@v1.16.0
RUN go get github.com/josharian/impl@v1.1.0
RUN go get github.com/haya14busa/goplay/cmd/goplay@v1.0.0
RUN go get github.com/go-delve/delve/cmd/dlv@v1.8.3
RUN go get honnef.co/go/tools/cmd/staticcheck@v0.2.2
RUN go get golang.org/x/tools/gopls@v0.8.4

EXPOSE 8084