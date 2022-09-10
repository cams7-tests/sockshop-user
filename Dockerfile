FROM golang:1.19.0-alpine3.16

RUN apk update && apk add --no-cache gcc libc-dev

RUN go install github.com/cweill/gotests/gotests@v1.6.0
RUN go install github.com/fatih/gomodifytags@v1.16.0
RUN go install github.com/josharian/impl@v1.1.0
RUN go install github.com/haya14busa/goplay/cmd/goplay@v1.0.0
RUN go install github.com/go-delve/delve/cmd/dlv@v1.8.3
RUN go install honnef.co/go/tools/cmd/staticcheck@v0.2.2
RUN go install golang.org/x/tools/gopls@v0.8.4

EXPOSE 8084

ENV HATEAOS user
ENV USER_DATABASE mongodb
ENV MONGODB_CONNECTION_STRING mongodb://192.168.100.14:27017