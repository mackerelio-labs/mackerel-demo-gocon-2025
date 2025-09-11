FROM golang:1.25

RUN apt-get update && apt-get install -y git protobuf-compiler

WORKDIR /pb
COPY go.mod go.sum ./
RUN go install google.golang.org/protobuf/cmd/protoc-gen-go
RUN go install google.golang.org/grpc/cmd/protoc-gen-go-grpc
